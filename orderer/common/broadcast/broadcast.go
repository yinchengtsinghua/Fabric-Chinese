
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package broadcast

import (
	"io"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/orderer/common/msgprocessor"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("orderer.common.broadcast")

//go：生成伪造者-o mock/channel_-support_-registrar.go——伪造名称channel support registrar。频道支持注册器

//ChannelSupportRegistrar为处理程序提供了一种查找通道支持的方法
type ChannelSupportRegistrar interface {
//BroadcastChannelSupport返回消息通道头，无论该消息是否为配置更新
//如果消息不是可以
//直接处理（如配置和订购者事务消息）
	BroadcastChannelSupport(msg *cb.Envelope) (*cb.ChannelHeader, bool, ChannelSupport, error)
}

//go：生成仿冒者-o mock/channel_support.go--forke name channel support。渠道支持

//频道支持提供支持频道广播所需的支持资源
type ChannelSupport interface {
	msgprocessor.Processor
	Consenter
}

//同意者提供通过协商一致发送消息的方法
type Consenter interface {
//ORDER接受一条消息或返回一个指示失败原因的错误
//它最终传递到共识链接口
	Order(env *cb.Envelope, configSeq uint64) error

//configure接受重新配置或返回一个错误，指示失败原因
//它最终传递到共识链接口
	Configure(config *cb.Envelope, configSeq uint64) error

//waitready阻止等待同意者准备接受新消息。
//当同意者需要临时阻止进入消息时，这很有用，因此
//飞行中的信息可以被消耗掉。如果同意者是
//处于错误状态。如果不需要这种阻塞行为，同意者可以
//只需返回零。
	WaitReady() error
}

//处理程序用于处理来自广播AB GRPC服务的连接
type Handler struct {
	SupportRegistrar ChannelSupportRegistrar
	Metrics          *Metrics
}

//句柄从广播流读取请求，处理它们，并返回对该流的响应
func (bh *Handler) Handle(srv ab.AtomicBroadcast_BroadcastServer) error {
	addr := util.ExtractRemoteAddress(srv.Context())
	logger.Debugf("Starting new broadcast loop for %s", addr)
	for {
		msg, err := srv.Recv()
		if err == io.EOF {
			logger.Debugf("Received EOF from %s, hangup", addr)
			return nil
		}
		if err != nil {
			logger.Warningf("Error reading from %s: %s", addr, err)
			return err
		}

		resp := bh.ProcessMessage(msg, addr)
		err = srv.Send(resp)
		if resp.Status != cb.Status_SUCCESS {
			return err
		}

		if err != nil {
			logger.Warningf("Error sending to %s: %s", addr, err)
			return err
		}
	}

}

type MetricsTracker struct {
	ValidateStartTime time.Time
	EnqueueStartTime  time.Time
	ValidateDuration  time.Duration
	ChannelID         string
	TxType            string
	Metrics           *Metrics
}

func (mt *MetricsTracker) Record(resp *ab.BroadcastResponse) {
	labels := []string{
		"status", resp.Status.String(),
		"channel", mt.ChannelID,
		"type", mt.TxType,
	}

	if mt.ValidateDuration == 0 {
		mt.EndValidate()
	}
	mt.Metrics.ValidateDuration.With(labels...).Observe(mt.ValidateDuration.Seconds())

	if mt.EnqueueStartTime != (time.Time{}) {
		enqueueDuration := time.Since(mt.EnqueueStartTime)
		mt.Metrics.EnqueueDuration.With(labels...).Observe(enqueueDuration.Seconds())
	}

	mt.Metrics.ProcessedCount.With(labels...).Add(1)
}

func (mt *MetricsTracker) BeginValidate() {
	mt.ValidateStartTime = time.Now()
}

func (mt *MetricsTracker) EndValidate() {
	mt.ValidateDuration = time.Since(mt.ValidateStartTime)
}

func (mt *MetricsTracker) BeginEnqueue() {
	mt.EnqueueStartTime = time.Now()
}

//processMessage验证单个消息并使其排队
func (bh *Handler) ProcessMessage(msg *cb.Envelope, addr string) (resp *ab.BroadcastResponse) {
	tracker := &MetricsTracker{
		ChannelID: "unknown",
		TxType:    "unknown",
		Metrics:   bh.Metrics,
	}
	defer func() {
//这看起来有点不必要，但如果直接这样做
//延迟，resp获取resp的（始终为零）当前状态
//而不是返回值
		tracker.Record(resp)
	}()
	tracker.BeginValidate()

	chdr, isConfig, processor, err := bh.SupportRegistrar.BroadcastChannelSupport(msg)
	if chdr != nil {
		tracker.ChannelID = chdr.ChannelId
		tracker.TxType = cb.HeaderType(chdr.Type).String()
	}
	if err != nil {
		logger.Warningf("[channel: %s] Could not get message processor for serving %s: %s", tracker.ChannelID, addr, err)
		return &ab.BroadcastResponse{Status: cb.Status_BAD_REQUEST, Info: err.Error()}
	}

	if !isConfig {
		logger.Debugf("[channel: %s] Broadcast is processing normal message from %s with txid '%s' of type %s", chdr.ChannelId, addr, chdr.TxId, cb.HeaderType_name[chdr.Type])

		configSeq, err := processor.ProcessNormalMsg(msg)
		if err != nil {
			logger.Warningf("[channel: %s] Rejecting broadcast of normal message from %s because of error: %s", chdr.ChannelId, addr, err)
			return &ab.BroadcastResponse{Status: ClassifyError(err), Info: err.Error()}
		}
		tracker.EndValidate()

		tracker.BeginEnqueue()
		if err = processor.WaitReady(); err != nil {
			logger.Warningf("[channel: %s] Rejecting broadcast of message from %s with SERVICE_UNAVAILABLE: rejected by Consenter: %s", chdr.ChannelId, addr, err)
			return &ab.BroadcastResponse{Status: cb.Status_SERVICE_UNAVAILABLE, Info: err.Error()}
		}

		err = processor.Order(msg, configSeq)
		if err != nil {
			logger.Warningf("[channel: %s] Rejecting broadcast of normal message from %s with SERVICE_UNAVAILABLE: rejected by Order: %s", chdr.ChannelId, addr, err)
			return &ab.BroadcastResponse{Status: cb.Status_SERVICE_UNAVAILABLE, Info: err.Error()}
		}
} else { //ISCONFIG
		logger.Debugf("[channel: %s] Broadcast is processing config update message from %s", chdr.ChannelId, addr)

		config, configSeq, err := processor.ProcessConfigUpdateMsg(msg)
		if err != nil {
			logger.Warningf("[channel: %s] Rejecting broadcast of config message from %s because of error: %s", chdr.ChannelId, addr, err)
			return &ab.BroadcastResponse{Status: ClassifyError(err), Info: err.Error()}
		}
		tracker.EndValidate()

		tracker.BeginEnqueue()
		if err = processor.WaitReady(); err != nil {
			logger.Warningf("[channel: %s] Rejecting broadcast of message from %s with SERVICE_UNAVAILABLE: rejected by Consenter: %s", chdr.ChannelId, addr, err)
			return &ab.BroadcastResponse{Status: cb.Status_SERVICE_UNAVAILABLE, Info: err.Error()}
		}

		err = processor.Configure(config, configSeq)
		if err != nil {
			logger.Warningf("[channel: %s] Rejecting broadcast of config message from %s with SERVICE_UNAVAILABLE: rejected by Configure: %s", chdr.ChannelId, addr, err)
			return &ab.BroadcastResponse{Status: cb.Status_SERVICE_UNAVAILABLE, Info: err.Error()}
		}
	}

	logger.Debugf("[channel: %s] Broadcast has successfully enqueued message of type %s from %s", chdr.ChannelId, cb.HeaderType_name[chdr.Type], addr)

	return &ab.BroadcastResponse{Status: cb.Status_SUCCESS}
}

//ClassifierRor将错误类型转换为状态代码。
func ClassifyError(err error) cb.Status {
	switch errors.Cause(err) {
	case msgprocessor.ErrChannelDoesNotExist:
		return cb.Status_NOT_FOUND
	case msgprocessor.ErrPermissionDenied:
		return cb.Status_FORBIDDEN
	default:
		return cb.Status_BAD_REQUEST
	}
}
