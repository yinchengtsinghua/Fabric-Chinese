
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/deliver"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/orderer/common/broadcast"
	localconfig "github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/common/msgprocessor"
	"github.com/hyperledger/fabric/orderer/common/multichannel"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
)

type broadcastSupport struct {
	*multichannel.Registrar
}

func (bs broadcastSupport) BroadcastChannelSupport(msg *cb.Envelope) (*cb.ChannelHeader, bool, broadcast.ChannelSupport, error) {
	return bs.Registrar.BroadcastChannelSupport(msg)
}

type deliverSupport struct {
	*multichannel.Registrar
}

func (ds deliverSupport) GetChain(chainID string) deliver.Chain {
	chain := ds.Registrar.GetChain(chainID)
	if chain == nil {
		return nil
	}
	return chain
}

type server struct {
	bh    *broadcast.Handler
	dh    *deliver.Handler
	debug *localconfig.Debug
	*multichannel.Registrar
}

type responseSender struct {
	ab.AtomicBroadcast_DeliverServer
}

func (rs *responseSender) SendStatusResponse(status cb.Status) error {
	reply := &ab.DeliverResponse{
		Type: &ab.DeliverResponse_Status{Status: status},
	}
	return rs.Send(reply)
}

func (rs *responseSender) SendBlockResponse(block *cb.Block) error {
	response := &ab.DeliverResponse{
		Type: &ab.DeliverResponse_Block{Block: block},
	}
	return rs.Send(response)
}

//newserver基于广播目标和分类帐阅读器创建ab.atomicbroadcastserver
func NewServer(r *multichannel.Registrar, metricsProvider metrics.Provider, debug *localconfig.Debug, timeWindow time.Duration, mutualTLS bool) ab.AtomicBroadcastServer {
	s := &server{
		dh: deliver.NewHandler(deliverSupport{Registrar: r}, timeWindow, mutualTLS, deliver.NewMetrics(metricsProvider)),
		bh: &broadcast.Handler{
			SupportRegistrar: broadcastSupport{Registrar: r},
			Metrics:          broadcast.NewMetrics(metricsProvider),
		},
		debug:     debug,
		Registrar: r,
	}
	return s
}

type msgTracer struct {
	function string
	debug    *localconfig.Debug
}

func (mt *msgTracer) trace(traceDir string, msg *cb.Envelope, err error) {
	if err != nil {
		return
	}

	now := time.Now().UnixNano()
	path := fmt.Sprintf("%s%c%d_%p.%s", traceDir, os.PathSeparator, now, msg, mt.function)
	logger.Debugf("Writing %s request trace to %s", mt.function, path)
	go func() {
		pb, err := proto.Marshal(msg)
		if err != nil {
			logger.Debugf("Error marshaling trace msg for %s: %s", path, err)
			return
		}
		err = ioutil.WriteFile(path, pb, 0660)
		if err != nil {
			logger.Debugf("Error writing trace msg for %s: %s", path, err)
		}
	}()
}

type broadcastMsgTracer struct {
	ab.AtomicBroadcast_BroadcastServer
	msgTracer
}

func (bmt *broadcastMsgTracer) Recv() (*cb.Envelope, error) {
	msg, err := bmt.AtomicBroadcast_BroadcastServer.Recv()
	if traceDir := bmt.debug.BroadcastTraceDir; traceDir != "" {
		bmt.trace(bmt.debug.BroadcastTraceDir, msg, err)
	}
	return msg, err
}

type deliverMsgTracer struct {
	deliver.Receiver
	msgTracer
}

func (dmt *deliverMsgTracer) Recv() (*cb.Envelope, error) {
	msg, err := dmt.Receiver.Recv()
	if traceDir := dmt.debug.DeliverTraceDir; traceDir != "" {
		dmt.trace(traceDir, msg, err)
	}
	return msg, err
}

//广播接收来自客户端的消息流以进行排序
func (s *server) Broadcast(srv ab.AtomicBroadcast_BroadcastServer) error {
	logger.Debugf("Starting new Broadcast handler")
	defer func() {
		if r := recover(); r != nil {
			logger.Criticalf("Broadcast client triggered panic: %s\n%s", r, debug.Stack())
		}
		logger.Debugf("Closing Broadcast stream")
	}()
	return s.bh.Handle(&broadcastMsgTracer{
		AtomicBroadcast_BroadcastServer: srv,
		msgTracer: msgTracer{
			debug:    s.debug,
			function: "Broadcast",
		},
	})
}

//Deliver在订购后将块流发送到客户机
func (s *server) Deliver(srv ab.AtomicBroadcast_DeliverServer) error {
	logger.Debugf("Starting new Deliver handler")
	defer func() {
		if r := recover(); r != nil {
			logger.Criticalf("Deliver client triggered panic: %s\n%s", r, debug.Stack())
		}
		logger.Debugf("Closing Deliver stream")
	}()

	policyChecker := func(env *cb.Envelope, channelID string) error {
		chain := s.GetChain(channelID)
		if chain == nil {
			return errors.Errorf("channel %s not found", channelID)
		}
		sf := msgprocessor.NewSigFilter(policies.ChannelReaders, chain)
		return sf.Apply(env)
	}
	deliverServer := &deliver.Server{
		PolicyChecker: deliver.PolicyCheckerFunc(policyChecker),
		Receiver: &deliverMsgTracer{
			Receiver: srv,
			msgTracer: msgTracer{
				debug:    s.debug,
				function: "Deliver",
			},
		},
		ResponseSender: &responseSender{
			AtomicBroadcast_DeliverServer: srv,
		},
	}
	return s.dh.Handle(srv.Context(), deliverServer)
}

func (s *server) sendProducer(srv ab.AtomicBroadcast_DeliverServer) func(msg proto.Message) error {
	return func(msg proto.Message) error {
		response, ok := msg.(*ab.DeliverResponse)
		if !ok {
			logger.Errorf("received wrong response type, expected response type ab.DeliverResponse")
			return errors.New("expected response type ab.DeliverResponse")
		}
		return srv.Send(response)
	}
}
