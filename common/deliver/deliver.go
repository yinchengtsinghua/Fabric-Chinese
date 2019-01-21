
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


package deliver

import (
	"context"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("common.deliver")

//go：生成伪造者-o mock/chain\u manager.go-fake name chain manager。连锁经理

//ChainManager为处理程序提供了查找链的方法。
type ChainManager interface {
	GetChain(chainID string) Chain
}

//去：生成伪造者-o mock/chain.go-伪造名字链。链

//链封装了链操作和数据。
type Chain interface {
//sequence返回当前配置序列号，可用于检测配置更改
	Sequence() uint64

//policyManager返回由链配置指定的当前策略管理器
	PolicyManager() policies.Manager

//读卡器返回链的链读卡器
	Reader() blockledger.Reader

//出错返回一个通道，当支持同意者出错时该通道关闭。
	Errored() <-chan struct{}
}

//go:生成仿冒者-o mock/policy\u checker.go-forke name policy checker。策略检查器

//policyChecker根据提供的策略逻辑检查信封
//功能。
type PolicyChecker interface {
	CheckPolicy(envelope *cb.Envelope, channelID string) error
}

//policycheckerfnc是一个适配器，允许使用普通的
//作为策略检查器。
type PolicyCheckerFunc func(envelope *cb.Envelope, channelID string) error

//checkpolicy调用pcf（信封，channelid）
func (pcf PolicyCheckerFunc) CheckPolicy(envelope *cb.Envelope, channelID string) error {
	return pcf(envelope, channelID)
}

//go：生成伪造者-o mock/inspector.go-伪造姓名inspector。检查员

//检查器验证消息和上下文之间的适当绑定。
type Inspector interface {
	Inspect(context.Context, proto.Message) error
}

//inspectorfunc是一个适配器，允许使用普通的
//作为检查员。
type InspectorFunc func(context.Context, proto.Message) error

//检查呼叫检查员（CTX，P）
func (inspector InspectorFunc) Inspect(ctx context.Context, p proto.Message) error {
	return inspector(ctx, p)
}

//
type Handler struct {
	ChainManager     ChainManager
	TimeWindow       time.Duration
	BindingInspector Inspector
	Metrics          *Metrics
}

//go：生成伪造者-o mock/receiver.go-fake name receiver。接收机

//接收器用于接收封装的寻道请求。
type Receiver interface {
	Recv() (*cb.Envelope, error)
}

//go：生成伪造者-o模拟/响应\u sender.go-伪造姓名响应者。应答者

//ResponseSender定义处理程序必须实现以发送的接口
//响应。
type ResponseSender interface {
	SendStatusResponse(status cb.Status) error
	SendBlockResponse(block *cb.Block) error
}

//筛选是指示响应发送者的标记接口
//配置为发送筛选的块
type Filtered interface {
	IsFiltered() bool
}

//服务器是一个多态结构，支持此处理程序的泛化。
//能够提供不同类型的响应。
type Server struct {
	Receiver
	PolicyChecker
	ResponseSender
}

//ExtractChannelHeaderCertHash从通道头提取TLS证书哈希。
func ExtractChannelHeaderCertHash(msg proto.Message) []byte {
	chdr, isChannelHeader := msg.(*cb.ChannelHeader)
	if !isChannelHeader || chdr == nil {
		return nil
	}
	return chdr.TlsCertHash
}

//newhandler创建处理程序接口的实现。
func NewHandler(cm ChainManager, timeWindow time.Duration, mutualTLS bool, metrics *Metrics) *Handler {
	return &Handler{
		ChainManager:     cm,
		TimeWindow:       timeWindow,
		BindingInspector: InspectorFunc(comm.NewBindingInspector(mutualTLS, ExtractChannelHeaderCertHash)),
		Metrics:          metrics,
	}
}

//句柄接收传入的传递请求。
func (h *Handler) Handle(ctx context.Context, srv *Server) error {
	addr := util.ExtractRemoteAddress(ctx)
	logger.Debugf("Starting new deliver loop for %s", addr)
	h.Metrics.StreamsOpened.Add(1)
	defer h.Metrics.StreamsClosed.Add(1)
	for {
		logger.Debugf("Attempting to read seek info message from %s", addr)
		envelope, err := srv.Recv()
		if err == io.EOF {
			logger.Debugf("Received EOF from %s, hangup", addr)
			return nil
		}
		if err != nil {
			logger.Warningf("Error reading from %s: %s", addr, err)
			return err
		}

		status, err := h.deliverBlocks(ctx, srv, envelope)
		if err != nil {
			return err
		}

		err = srv.SendStatusResponse(status)
		if status != cb.Status_SUCCESS {
			return err
		}
		if err != nil {
			logger.Warningf("Error sending to %s: %s", addr, err)
			return err
		}

		logger.Debugf("Waiting for new SeekInfo from %s", addr)
	}
}

func isFiltered(srv *Server) bool {
	if filtered, ok := srv.ResponseSender.(Filtered); ok {
		return filtered.IsFiltered()
	}
	return false
}

func (h *Handler) deliverBlocks(ctx context.Context, srv *Server, envelope *cb.Envelope) (status cb.Status, err error) {
	addr := util.ExtractRemoteAddress(ctx)
	payload, err := utils.UnmarshalPayload(envelope.Payload)
	if err != nil {
		logger.Warningf("Received an envelope from %s with no payload: %s", addr, err)
		return cb.Status_BAD_REQUEST, nil
	}

	if payload.Header == nil {
		logger.Warningf("Malformed envelope received from %s with bad header", addr)
		return cb.Status_BAD_REQUEST, nil
	}

	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		logger.Warningf("Failed to unmarshal channel header from %s: %s", addr, err)
		return cb.Status_BAD_REQUEST, nil
	}

	err = h.validateChannelHeader(ctx, chdr)
	if err != nil {
		logger.Warningf("Rejecting deliver for %s due to envelope validation error: %s", addr, err)
		return cb.Status_BAD_REQUEST, nil
	}

	chain := h.ChainManager.GetChain(chdr.ChannelId)
	if chain == nil {
//注意，我们在调试时记录这个，因为SDK将轮询等待创建通道。
//所以我们希望我们的日志中会有大量的这些
		logger.Debugf("Rejecting deliver for %s because channel %s not found", addr, chdr.ChannelId)
		return cb.Status_NOT_FOUND, nil
	}

	labels := []string{
		"channel", chdr.ChannelId,
		"filtered", strconv.FormatBool(isFiltered(srv)),
	}
	h.Metrics.RequestsReceived.With(labels...).Add(1)
	defer func() {
		labels := append(labels, "success", strconv.FormatBool(status == cb.Status_SUCCESS))
		h.Metrics.RequestsCompleted.With(labels...).Add(1)
	}()

	erroredChan := chain.Errored()
	select {
	case <-erroredChan:
		logger.Warningf("[channel: %s] Rejecting deliver request for %s because of consenter error", chdr.ChannelId, addr)
		return cb.Status_SERVICE_UNAVAILABLE, nil
	default:
	}

	accessControl, err := NewSessionAC(chain, envelope, srv.PolicyChecker, chdr.ChannelId, crypto.ExpiresAt)
	if err != nil {
		logger.Warningf("[channel: %s] failed to create access control object due to %s", chdr.ChannelId, err)
		return cb.Status_BAD_REQUEST, nil
	}

	if err := accessControl.Evaluate(); err != nil {
		logger.Warningf("[channel: %s] Client authorization revoked for deliver request from %s: %s", chdr.ChannelId, addr, err)
		return cb.Status_FORBIDDEN, nil
	}

	seekInfo := &ab.SeekInfo{}
	if err = proto.Unmarshal(payload.Data, seekInfo); err != nil {
		logger.Warningf("[channel: %s] Received a signed deliver request from %s with malformed seekInfo payload: %s", chdr.ChannelId, addr, err)
		return cb.Status_BAD_REQUEST, nil
	}

	if seekInfo.Start == nil || seekInfo.Stop == nil {
		logger.Warningf("[channel: %s] Received seekInfo message from %s with missing start or stop %v, %v", chdr.ChannelId, addr, seekInfo.Start, seekInfo.Stop)
		return cb.Status_BAD_REQUEST, nil
	}

	logger.Debugf("[channel: %s] Received seekInfo (%p) %v from %s", chdr.ChannelId, seekInfo, seekInfo, addr)

	cursor, number := chain.Reader().Iterator(seekInfo.Start)
	defer cursor.Close()
	var stopNum uint64
	switch stop := seekInfo.Stop.Type.(type) {
	case *ab.SeekPosition_Oldest:
		stopNum = number
	case *ab.SeekPosition_Newest:
		stopNum = chain.Reader().Height() - 1
	case *ab.SeekPosition_Specified:
		stopNum = stop.Specified.Number
		if stopNum < number {
			logger.Warningf("[channel: %s] Received invalid seekInfo message from %s: start number %d greater than stop number %d", chdr.ChannelId, addr, number, stopNum)
			return cb.Status_BAD_REQUEST, nil
		}
	}

	for {
		if seekInfo.Behavior == ab.SeekInfo_FAIL_IF_NOT_READY {
			if number > chain.Reader().Height()-1 {
				return cb.Status_NOT_FOUND, nil
			}
		}

		var block *cb.Block
		var status cb.Status

		iterCh := make(chan struct{})
		go func() {
			block, status = cursor.Next()
			close(iterCh)
		}()

		select {
		case <-ctx.Done():
			logger.Debugf("Context canceled, aborting wait for next block")
			return cb.Status_INTERNAL_SERVER_ERROR, errors.Wrapf(ctx.Err(), "context finished before block retrieved")
		case <-erroredChan:
			logger.Warningf("Aborting deliver for request because of background error")
			return cb.Status_SERVICE_UNAVAILABLE, nil
		case <-iterCh:
//迭代器已设置块和状态变量
		}

		if status != cb.Status_SUCCESS {
			logger.Errorf("[channel: %s] Error reading from channel, cause was: %v", chdr.ChannelId, status)
			return status, nil
		}

//如果未准备好交付行为，则增加块号以支持失败
		number++

		if err := accessControl.Evaluate(); err != nil {
			logger.Warningf("[channel: %s] Client authorization revoked for deliver request from %s: %s", chdr.ChannelId, addr, err)
			return cb.Status_FORBIDDEN, nil
		}

		logger.Debugf("[channel: %s] Delivering block for (%p) for %s", chdr.ChannelId, seekInfo, addr)

		if err := srv.SendBlockResponse(block); err != nil {
			logger.Warningf("[channel: %s] Error sending to %s: %s", chdr.ChannelId, addr, err)
			return cb.Status_INTERNAL_SERVER_ERROR, err
		}

		h.Metrics.BlocksSent.With(labels...).Add(1)

		if stopNum == block.Header.Number {
			break
		}
	}

	logger.Debugf("[channel: %s] Done delivering to %s for (%p)", chdr.ChannelId, addr, seekInfo)

	return cb.Status_SUCCESS, nil
}

func (h *Handler) validateChannelHeader(ctx context.Context, chdr *cb.ChannelHeader) error {
	if chdr.GetTimestamp() == nil {
		err := errors.New("channel header in envelope must contain timestamp")
		return err
	}

	envTime := time.Unix(chdr.GetTimestamp().Seconds, int64(chdr.GetTimestamp().Nanos)).UTC()
	serverTime := time.Now()

	if math.Abs(float64(serverTime.UnixNano()-envTime.UnixNano())) > float64(h.TimeWindow.Nanoseconds()) {
		err := errors.Errorf("envelope timestamp %s is more than %s apart from current server time %s", envTime, h.TimeWindow, serverTime)
		return err
	}

	err := h.BindingInspector.Inspect(ctx, chdr)
	if err != nil {
		return err
	}

	return nil
}
