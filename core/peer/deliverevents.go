
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


package peer

import (
	"runtime/debug"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/deliver"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("common.deliverevents")

//PolicyCheckerProvider为
//给定资源名称
type PolicyCheckerProvider func(resourceName string) deliver.PolicyCheckerFunc

//服务器保留创建传递服务器所需的依赖项
type server struct {
	dh                    *deliver.Handler
	policyCheckerProvider PolicyCheckerProvider
}

//用于发送块响应的BlockResponseSender结构
type blockResponseSender struct {
	peer.Deliver_DeliverServer
}

//sendstatusResponse生成状态回复协议消息
func (brs *blockResponseSender) SendStatusResponse(status common.Status) error {
	reply := &peer.DeliverResponse{
		Type: &peer.DeliverResponse_Status{Status: status},
	}
	return brs.Send(reply)
}

//sendblockresponse生成带有块消息的传递响应
func (brs *blockResponseSender) SendBlockResponse(block *common.Block) error {
	response := &peer.DeliverResponse{
		Type: &peer.DeliverResponse_Block{Block: block},
	}
	return brs.Send(response)
}

//用于发送筛选的块响应的filteredBlockResponseSender结构
type filteredBlockResponseSender struct {
	peer.Deliver_DeliverFilteredServer
}

//sendstatusResponse生成状态回复协议消息
func (fbrs *filteredBlockResponseSender) SendStatusResponse(status common.Status) error {
	response := &peer.DeliverResponse{
		Type: &peer.DeliverResponse_Status{Status: status},
	}
	return fbrs.Send(response)
}

//isfiltered是一个标记方法，它指示此响应发送者
//发送筛选的块。
func (fbrs *filteredBlockResponseSender) IsFiltered() bool {
	return true
}

//sendblockresponse生成带有块消息的传递响应
func (fbrs *filteredBlockResponseSender) SendBlockResponse(block *common.Block) error {
//生成过滤块响应
	b := blockEvent(*block)
	filteredBlock, err := b.toFilteredBlock()
	if err != nil {
		logger.Warningf("Failed to generate filtered block due to: %s", err)
		return fbrs.SendStatusResponse(common.Status_BAD_REQUEST)
	}
	response := &peer.DeliverResponse{
		Type: &peer.DeliverResponse_FilteredBlock{FilteredBlock: filteredBlock},
	}
	return fbrs.Send(response)
}

//peer.transaction pointers切片的transactionactions别名
type transactionActions []*peer.TransactionAction

//blockEvent是common.block结构的别名，用于
//使用辅助功能扩展
type blockEvent common.Block

//交付在承诺之后向客户发送一个块流
func (s *server) DeliverFiltered(srv peer.Deliver_DeliverFilteredServer) error {
	logger.Debugf("Starting new DeliverFiltered handler")
	defer dumpStacktraceOnPanic()
//正在根据resources.event_filteredblock resource name获取策略检查器
	deliverServer := &deliver.Server{
		Receiver:      srv,
		PolicyChecker: s.policyCheckerProvider(resources.Event_FilteredBlock),
		ResponseSender: &filteredBlockResponseSender{
			Deliver_DeliverFilteredServer: srv,
		},
	}
	return s.dh.Handle(srv.Context(), deliverServer)
}

//交付在承诺之后向客户发送一个块流
func (s *server) Deliver(srv peer.Deliver_DeliverServer) (err error) {
	logger.Debugf("Starting new Deliver handler")
	defer dumpStacktraceOnPanic()
//正在根据resources.event\u块资源名称获取策略检查器
	deliverServer := &deliver.Server{
		PolicyChecker: s.policyCheckerProvider(resources.Event_Block),
		Receiver:      srv,
		ResponseSender: &blockResponseSender{
			Deliver_DeliverServer: srv,
		},
	}
	return s.dh.Handle(srv.Context(), deliverServer)
}

//NewDeliverEventsServer创建一个对等端。传递服务器以传递块和
//筛选的块事件
func NewDeliverEventsServer(mutualTLS bool, policyCheckerProvider PolicyCheckerProvider, chainManager deliver.ChainManager, metricsProvider metrics.Provider) peer.DeliverServer {
	timeWindow := viper.GetDuration("peer.authentication.timewindow")
	if timeWindow == 0 {
		defaultTimeWindow := 15 * time.Minute
		logger.Warningf("`peer.authentication.timewindow` not set; defaulting to %s", defaultTimeWindow)
		timeWindow = defaultTimeWindow
	}
	metrics := deliver.NewMetrics(metricsProvider)
	return &server{
		dh:                    deliver.NewHandler(chainManager, timeWindow, mutualTLS, metrics),
		policyCheckerProvider: policyCheckerProvider,
	}
}

func (s *server) sendProducer(srv peer.Deliver_DeliverFilteredServer) func(msg proto.Message) error {
	return func(msg proto.Message) error {
		response, ok := msg.(*peer.DeliverResponse)
		if !ok {
			logger.Errorf("received wrong response type, expected response type peer.DeliverResponse")
			return errors.New("expected response type peer.DeliverResponse")
		}
		return srv.Send(response)
	}
}

func (block *blockEvent) toFilteredBlock() (*peer.FilteredBlock, error) {
	filteredBlock := &peer.FilteredBlock{
		Number: block.Header.Number,
	}

	txsFltr := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	for txIndex, ebytes := range block.Data.Data {
		var env *common.Envelope
		var err error

		if ebytes == nil {
			logger.Debugf("got nil data bytes for tx index %d, "+
				"block num %d", txIndex, block.Header.Number)
			continue
		}

		env, err = utils.GetEnvelopeFromBlock(ebytes)
		if err != nil {
			logger.Errorf("error getting tx from block, %s", err)
			continue
		}

//从信封中获取有效载荷
		payload, err := utils.GetPayload(env)
		if err != nil {
			return nil, errors.WithMessage(err, "could not extract payload from envelope")
		}

		if payload.Header == nil {
			logger.Debugf("transaction payload header is nil, %d, block num %d",
				txIndex, block.Header.Number)
			continue
		}
		chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			return nil, err
		}

		filteredBlock.ChannelId = chdr.ChannelId

		filteredTransaction := &peer.FilteredTransaction{
			Txid:             chdr.TxId,
			Type:             common.HeaderType(chdr.Type),
			TxValidationCode: txsFltr.Flag(txIndex),
		}

		if filteredTransaction.Type == common.HeaderType_ENDORSER_TRANSACTION {
			tx, err := utils.GetTransaction(payload.Data)
			if err != nil {
				return nil, errors.WithMessage(err, "error unmarshal transaction payload for block event")
			}

			filteredTransaction.Data, err = transactionActions(tx.Actions).toFilteredActions()
			if err != nil {
				logger.Errorf(err.Error())
				return nil, err
			}
		}

		filteredBlock.FilteredTransactions = append(filteredBlock.FilteredTransactions, filteredTransaction)
	}

	return filteredBlock, nil
}

func (ta transactionActions) toFilteredActions() (*peer.FilteredTransaction_TransactionActions, error) {
	transactionActions := &peer.FilteredTransactionActions{}
	for _, action := range ta {
		chaincodeActionPayload, err := utils.GetChaincodeActionPayload(action.Payload)
		if err != nil {
			return nil, errors.WithMessage(err, "error unmarshal transaction action payload for block event")
		}

		if chaincodeActionPayload.Action == nil {
			logger.Debugf("chaincode action, the payload action is nil, skipping")
			continue
		}
		propRespPayload, err := utils.GetProposalResponsePayload(chaincodeActionPayload.Action.ProposalResponsePayload)
		if err != nil {
			return nil, errors.WithMessage(err, "error unmarshal proposal response payload for block event")
		}

		caPayload, err := utils.GetChaincodeAction(propRespPayload.Extension)
		if err != nil {
			return nil, errors.WithMessage(err, "error unmarshal chaincode action for block event")
		}

		ccEvent, err := utils.GetChaincodeEvents(caPayload.Events)
		if err != nil {
			return nil, errors.WithMessage(err, "error unmarshal chaincode event for block event")
		}

		if ccEvent.GetChaincodeId() != "" {
			filteredAction := &peer.FilteredChaincodeAction{
				ChaincodeEvent: &peer.ChaincodeEvent{
					TxId:        ccEvent.TxId,
					ChaincodeId: ccEvent.ChaincodeId,
					EventName:   ccEvent.EventName,
				},
			}
			transactionActions.ChaincodeActions = append(transactionActions.ChaincodeActions, filteredAction)
		}
	}
	return &peer.FilteredTransaction_TransactionActions{
		TransactionActions: transactionActions,
	}, nil
}

func dumpStacktraceOnPanic() {
	func() {
		if r := recover(); r != nil {
			logger.Criticalf("Deliver client triggered panic: %s\n%s", r, debug.Stack())
		}
		logger.Debugf("Closing Deliver stream")
	}()
}
