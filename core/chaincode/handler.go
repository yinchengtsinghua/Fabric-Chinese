
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


package chaincode

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/flogging"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/container/ccintf"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

var chaincodeLogger = flogging.MustGetLogger("chaincode")

//ACLprovider在调用时执行访问控制检查
//链码。
type ACLProvider interface {
	CheckACL(resName string, channelID string, idinfo interface{}) error
}

//注册表负责跟踪处理程序。
type Registry interface {
	Register(*Handler) error
	Ready(cname string)
	Failed(cname string, err error)
	Deregister(cname string) error
}

//调用程序调用链代码。
type Invoker interface {
	Invoke(txParams *ccprovider.TransactionParams, cccid *ccprovider.CCContext, spec *pb.ChaincodeInput) (*pb.ChaincodeMessage, error)
}

//SystemCCProvider提供系统链代码元数据。
type SystemCCProvider interface {
	IsSysCC(name string) bool
	IsSysCCAndNotInvokableCC2CC(name string) bool
}

//TransactionRegistry跟踪每个通道的活动事务。
type TransactionRegistry interface {
	Add(channelID, txID string) bool
	Remove(channelID, txID string)
}

//
type ContextRegistry interface {
	Create(txParams *ccprovider.TransactionParams) (*TransactionContext, error)
	Get(chainID, txID string) *TransactionContext
	Delete(chainID, txID string)
	Close()
}

//InstantiationPolicyChecker用于评估实例化策略。
type InstantiationPolicyChecker interface {
	CheckInstantiationPolicy(name, version string, cd *ccprovider.ChaincodeData) error
}

//从函数到InstantiationPolicyChecker接口的适配器。
type CheckInstantiationPolicyFunc func(name, version string, cd *ccprovider.ChaincodeData) error

func (c CheckInstantiationPolicyFunc) CheckInstantiationPolicy(name, version string, cd *ccprovider.ChaincodeData) error {
	return c(name, version, cd)
}

//QueryResponseBuilder负责为查询生成QueryResponse消息
//由链码启动的事务。
type QueryResponseBuilder interface {
	BuildQueryResponse(txContext *TransactionContext, iter commonledger.ResultsIterator,
		iterID string, isPaginated bool, totalReturnLimit int32) (*pb.QueryResponse, error)
}

//chaincodedefinitiongetter负责检索chaincode定义
//来自系统。InstantiationPolicyChecker使用该定义。
type ChaincodeDefinitionGetter interface {
	ChaincodeDefinition(chaincodeName string, txSim ledger.QueryExecutor) (ccprovider.ChaincodeDefinition, error)
}

//Ledgergetter用于获取链码的分类帐。
type LedgerGetter interface {
	GetLedger(cid string) ledger.PeerLedger
}

//UuidGenerator负责创建唯一的查询标识符。
type UUIDGenerator interface {
	New() string
}
type UUIDGeneratorFunc func() string

func (u UUIDGeneratorFunc) New() string { return u() }

//ApplicationConfigRetriever检索通道的应用程序配置
type ApplicationConfigRetriever interface {
//getapplicationconfig返回通道的channelconfig.application
//以及应用程序配置是否存在
	GetApplicationConfig(cid string) (channelconfig.Application, bool)
}

//处理程序实现链代码流的对等端。
type Handler struct {
//keep alive指定保持活动消息的发送间隔。
	Keepalive time.Duration
//systemcversion指定当前系统链码版本
	SystemCCVersion string
//definitiongetter用于从
//生命周期系统链码。
	DefinitionGetter ChaincodeDefinitionGetter
//调用程序用于调用链代码。
	Invoker Invoker
//注册表用于跟踪活动的处理程序。
	Registry Registry
//aclprovider用于检查是否允许调用chaincode。
	ACLProvider ACLProvider
//
//通过通道名称和事务ID访问的。
	TXContexts ContextRegistry
//ActiveTransactions保存活动事务标识符。
	ActiveTransactions TransactionRegistry
//SystemCCProvider提供对系统链码元数据的访问
	SystemCCProvider SystemCCProvider
//InstantiationPolicyChecker用于评估链码实例化策略。
	InstantiationPolicyChecker InstantiationPolicyChecker
//
	QueryResponseBuilder QueryResponseBuilder
//Ledgergetter用于获取与渠道关联的分类帐
	LedgerGetter LedgerGetter
//
	UUIDGenerator UUIDGenerator
//AppConfig用于检索通道的应用程序配置
	AppConfig ApplicationConfigRetriever

//状态保持当前处理程序状态。它将被创建、建立或
//准备好了。
	state State
//chaincode id保存向对等方注册的链码的ID。
	chaincodeID *pb.ChaincodeID
//CCInstances保存与
//同辈。
	ccInstance *sysccprovider.ChaincodeInstance

//seriallock用于序列化跨GRPC聊天流的发送。
	serialLock sync.Mutex
//Chatstream是双向GRPC流，用于与
//链代码实例。
	chatStream ccintf.ChaincodeStream
//errchan用于将异步发送到接收循环的错误进行通信。
	errChan chan error
//度量保存链码处理程序度量
	Metrics *HandlerMetrics
}

//processstream调用handleMessage来调度消息。
func (h *Handler) handleMessage(msg *pb.ChaincodeMessage) error {
	chaincodeLogger.Debugf("[%s] Fabric side handling ChaincodeMessage of type: %s in state %s", shorttxid(msg.Txid), msg.Type, h.state)

	if msg.Type == pb.ChaincodeMessage_KEEPALIVE {
		return nil
	}

	switch h.state {
	case Created:
		return h.handleMessageCreatedState(msg)
	case Ready:
		return h.handleMessageReadyState(msg)
	default:
		return errors.Errorf("handle message: invalid state %s for transaction %s", h.state, msg.Txid)
	}
}

func (h *Handler) handleMessageCreatedState(msg *pb.ChaincodeMessage) error {
	switch msg.Type {
	case pb.ChaincodeMessage_REGISTER:
		h.HandleRegister(msg)
	default:
		return fmt.Errorf("[%s] Fabric side handler cannot handle message (%s) while in created state", msg.Txid, msg.Type)
	}
	return nil
}

func (h *Handler) handleMessageReadyState(msg *pb.ChaincodeMessage) error {
	switch msg.Type {
	case pb.ChaincodeMessage_COMPLETED, pb.ChaincodeMessage_ERROR:
		h.Notify(msg)

	case pb.ChaincodeMessage_PUT_STATE:
		go h.HandleTransaction(msg, h.HandlePutState)
	case pb.ChaincodeMessage_DEL_STATE:
		go h.HandleTransaction(msg, h.HandleDelState)
	case pb.ChaincodeMessage_INVOKE_CHAINCODE:
		go h.HandleTransaction(msg, h.HandleInvokeChaincode)

	case pb.ChaincodeMessage_GET_STATE:
		go h.HandleTransaction(msg, h.HandleGetState)
	case pb.ChaincodeMessage_GET_STATE_BY_RANGE:
		go h.HandleTransaction(msg, h.HandleGetStateByRange)
	case pb.ChaincodeMessage_GET_QUERY_RESULT:
		go h.HandleTransaction(msg, h.HandleGetQueryResult)
	case pb.ChaincodeMessage_GET_HISTORY_FOR_KEY:
		go h.HandleTransaction(msg, h.HandleGetHistoryForKey)
	case pb.ChaincodeMessage_QUERY_STATE_NEXT:
		go h.HandleTransaction(msg, h.HandleQueryStateNext)
	case pb.ChaincodeMessage_QUERY_STATE_CLOSE:
		go h.HandleTransaction(msg, h.HandleQueryStateClose)

	case pb.ChaincodeMessage_GET_STATE_METADATA:
		go h.HandleTransaction(msg, h.HandleGetStateMetadata)
	case pb.ChaincodeMessage_PUT_STATE_METADATA:
		go h.HandleTransaction(msg, h.HandlePutStateMetadata)
	default:
		return fmt.Errorf("[%s] Fabric side handler cannot handle message (%s) while in ready state", msg.Txid, msg.Type)
	}

	return nil
}

type MessageHandler interface {
	Handle(*pb.ChaincodeMessage, *TransactionContext) (*pb.ChaincodeMessage, error)
}

type handleFunc func(*pb.ChaincodeMessage, *TransactionContext) (*pb.ChaincodeMessage, error)

//
//将消息转发给提供的委托之前的上下文。响应消息
//代表返回的消息将发送到聊天流。返回的任何错误
//委托打包为链码错误消息。
func (h *Handler) HandleTransaction(msg *pb.ChaincodeMessage, delegate handleFunc) {
	chaincodeLogger.Debugf("[%s] handling %s from chaincode", shorttxid(msg.Txid), msg.Type.String())
	if !h.registerTxid(msg) {
		return
	}

	startTime := time.Now()
	var txContext *TransactionContext
	var err error
	if msg.Type == pb.ChaincodeMessage_INVOKE_CHAINCODE {
		txContext, err = h.getTxContextForInvoke(msg.ChannelId, msg.Txid, msg.Payload, "")
	} else {
		txContext, err = h.isValidTxSim(msg.ChannelId, msg.Txid, "no ledger context")
	}

	chaincodeName := h.chaincodeID.Name + ":" + h.chaincodeID.Version
	meterLabels := []string{
		"type", msg.Type.String(),
		"channel", msg.ChannelId,
		"chaincode", chaincodeName,
	}
	h.Metrics.ShimRequestsReceived.With(meterLabels...).Add(1)

	var resp *pb.ChaincodeMessage
	if err == nil {
		resp, err = delegate(msg, txContext)
	}

	if err != nil {
		err = errors.Wrapf(err, "%s failed: transaction ID: %s", msg.Type, msg.Txid)
		chaincodeLogger.Errorf("[%s] Failed to handle %s. error: %+v", shorttxid(msg.Txid), msg.Type, err)
		resp = &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: []byte(err.Error()), Txid: msg.Txid, ChannelId: msg.ChannelId}
	}

	chaincodeLogger.Debugf("[%s] Completed %s. Sending %s", shorttxid(msg.Txid), msg.Type, resp.Type)
	h.ActiveTransactions.Remove(msg.ChannelId, msg.Txid)
	h.serialSendAsync(resp)

	meterLabels = append(meterLabels, "success", strconv.FormatBool(resp.Type != pb.ChaincodeMessage_ERROR))
	h.Metrics.ShimRequestDuration.With(meterLabels...).Observe(time.Since(startTime).Seconds())
	h.Metrics.ShimRequestsCompleted.With(meterLabels...).Add(1)
}

func shorttxid(txid string) string {
	if len(txid) < 8 {
		return txid
	}
	return txid[0:8]
}

//ParseName将一个链代码名称解析为一个链代码实例。名字应该
//格式为“chaincode name:version/channel name”，带有可选元素。
func ParseName(ccName string) *sysccprovider.ChaincodeInstance {
	ci := &sysccprovider.ChaincodeInstance{}

	z := strings.SplitN(ccName, "/", 2)
	if len(z) == 2 {
		ci.ChainID = z[1]
	}
	z = strings.SplitN(z[0], ":", 2)
	if len(z) == 2 {
		ci.ChaincodeVersion = z[1]
	}
	ci.ChaincodeName = z[0]

	return ci
}

func (h *Handler) ChaincodeName() string {
	if h.ccInstance == nil {
		return ""
	}
	return h.ccInstance.ChaincodeName
}

//serialsend序列化msgs，以便grpc高兴
func (h *Handler) serialSend(msg *pb.ChaincodeMessage) error {
	h.serialLock.Lock()
	defer h.serialLock.Unlock()

	if err := h.chatStream.Send(msg); err != nil {
		err = errors.WithMessage(err, fmt.Sprintf("[%s] error sending %s", shorttxid(msg.Txid), msg.Type))
		chaincodeLogger.Errorf("%+v", err)
		return err
	}

	return nil
}

//serialsendasync与serialsend（序列化msgs以便grpc
//快乐点。另外，它也是异步的，所以send remoterecv--localrecv循环
//可以是非阻塞的。只需处理错误，这些错误由
//提供的错误通道上的通信。典型的用途是非阻塞或
//零通道
func (h *Handler) serialSendAsync(msg *pb.ChaincodeMessage) {
	go func() {
		if err := h.serialSend(msg); err != nil {
//向调用方提供错误响应
			resp := &pb.ChaincodeMessage{
				Type:      pb.ChaincodeMessage_ERROR,
				Payload:   []byte(err.Error()),
				Txid:      msg.Txid,
				ChannelId: msg.ChannelId,
			}
			h.Notify(resp)

//表面发送错误到流处理
			h.errChan <- err
		}
	}()
}

//检查事务处理程序是否允许在此通道上调用此链码
func (h *Handler) checkACL(signedProp *pb.SignedProposal, proposal *pb.Proposal, ccIns *sysccprovider.ChaincodeInstance) error {
//确保我们不调用系统链代码
//不能通过cc2cc调用调用的
	if h.SystemCCProvider.IsSysCCAndNotInvokableCC2CC(ccIns.ChaincodeName) {
		return errors.Errorf("system chaincode %s cannot be invoked with a cc2cc invocation", ccIns.ChaincodeName)
	}

//如果我们在这里，我们只知道被调用的链代码是
//
//（但我们仍然需要确定调用程序是否可以执行此调用）
//-应用程序链代码
//（我们仍然需要确定调用程序是否可以调用它）

	if h.SystemCCProvider.IsSysCC(ccIns.ChaincodeName) {
//允许这个呼叫
		return nil
	}

//对于非系统链码，将拒绝nil signedProp。
	if signedProp == nil {
		return errors.Errorf("signed proposal must not be nil from caller [%s]", ccIns.String())
	}

	return h.ACLProvider.CheckACL(resources.Peer_ChaincodeToChaincode, ccIns.ChainID, signedProp)
}

func (h *Handler) deregister() {
	if h.chaincodeID != nil {
		h.Registry.Deregister(h.chaincodeID.Name)
	}
}

func (h *Handler) ProcessStream(stream ccintf.ChaincodeStream) error {
	defer h.deregister()

	h.chatStream = stream
	h.errChan = make(chan error, 1)

	var keepaliveCh <-chan time.Time
	if h.Keepalive != 0 {
		ticker := time.NewTicker(h.Keepalive)
		defer ticker.Stop()
		keepaliveCh = ticker.C
	}

//保留下面GRPC Recv的返回值
	type recvMsg struct {
		msg *pb.ChaincodeMessage
		err error
	}
	msgAvail := make(chan *recvMsg, 1)

	receiveMessage := func() {
		in, err := h.chatStream.Recv()
		msgAvail <- &recvMsg{in, err}
	}

	go receiveMessage()
	for {
		select {
		case rmsg := <-msgAvail:
			switch {
//推迟此处理程序的注销。
			case rmsg.err == io.EOF:
				chaincodeLogger.Debugf("received EOF, ending chaincode support stream: %s", rmsg.err)
				return rmsg.err
			case rmsg.err != nil:
				err := errors.Wrap(rmsg.err, "receive failed")
				chaincodeLogger.Errorf("handling chaincode support stream: %+v", err)
				return err
			case rmsg.msg == nil:
				err := errors.New("received nil message, ending chaincode support stream")
				chaincodeLogger.Debugf("%+v", err)
				return err
			default:
				err := h.handleMessage(rmsg.msg)
				if err != nil {
					err = errors.WithMessage(err, "error handling message, ending stream")
					chaincodeLogger.Errorf("[%s] %+v", shorttxid(rmsg.msg.Txid), err)
					return err
				}

				go receiveMessage()
			}

		case sendErr := <-h.errChan:
			err := errors.Wrapf(sendErr, "received error while sending message, ending chaincode support stream")
			chaincodeLogger.Errorf("%s", err)
			return err
		case <-keepaliveCh:
//如果没有来自serialsend的错误消息，请保持快乐，不要在意错误。
//（也许以后会有用）
			h.serialSendAsync(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_KEEPALIVE})
			continue
		}
	}
}

//sendReady以串行方式发送ready-to-chaincode（就像寄存器一样）
func (h *Handler) sendReady() error {
	chaincodeLogger.Debugf("sending READY for chaincode %+v", h.chaincodeID)
	ccMsg := &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_READY}

//如果发送错误，请删除H
	if err := h.serialSend(ccMsg); err != nil {
		chaincodeLogger.Errorf("error sending READY (%s) for chaincode %+v", err, h.chaincodeID)
		return err
	}

	h.state = Ready

	chaincodeLogger.Debugf("Changed to state ready for chaincode %+v", h.chaincodeID)

	return nil
}

//一旦注册成功，notifyregistry将发送ready，并且
//更新处理程序注册表中链代码的启动状态。
func (h *Handler) notifyRegistry(err error) {
	if err == nil {
		err = h.sendReady()
	}

	if err != nil {
		h.Registry.Failed(h.chaincodeID.Name, err)
		chaincodeLogger.Errorf("failed to start %s", h.chaincodeID)
		return
	}

	h.Registry.Ready(h.chaincodeID.Name)
}

//当chaincode尝试注册时调用handleregister。
func (h *Handler) HandleRegister(msg *pb.ChaincodeMessage) {
	chaincodeLogger.Debugf("Received %s in state %s", msg.Type, h.state)
	chaincodeID := &pb.ChaincodeID{}
	err := proto.Unmarshal(msg.Payload, chaincodeID)
	if err != nil {
		chaincodeLogger.Errorf("Error in received %s, could NOT unmarshal registration info: %s", pb.ChaincodeMessage_REGISTER, err)
		return
	}

//现在向chaincode支持注册
	h.chaincodeID = chaincodeID
	err = h.Registry.Register(h)
	if err != nil {
		h.notifyRegistry(err)
		return
	}

//获取组件部件以便使用根链代码
//钥匙名称
	h.ccInstance = ParseName(h.chaincodeID.Name)

	chaincodeLogger.Debugf("Got %s for chaincodeID = %s, sending back %s", pb.ChaincodeMessage_REGISTER, chaincodeID, pb.ChaincodeMessage_REGISTERED)
	if err := h.serialSend(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTERED}); err != nil {
		chaincodeLogger.Errorf("error sending %s: %s", pb.ChaincodeMessage_REGISTERED, err)
		h.notifyRegistry(err)
		return
	}

	h.state = Established

	chaincodeLogger.Debugf("Changed state to established for %+v", h.chaincodeID)

//对于dev模式，这也将自动移动到就绪状态
	h.notifyRegistry(nil)
}

func (h *Handler) Notify(msg *pb.ChaincodeMessage) {
	tctx := h.TXContexts.Get(msg.ChannelId, msg.Txid)
	if tctx == nil {
		chaincodeLogger.Debugf("notifier Txid:%s, channelID:%s does not exist for handling message %s", msg.Txid, msg.ChannelId, msg.Type)
		return
	}

	chaincodeLogger.Debugf("[%s] notifying Txid:%s, channelID:%s", shorttxid(msg.Txid), msg.Txid, msg.ChannelId)
	tctx.ResponseNotifier <- msg
	tctx.CloseQueryIterators()
}

//这是有有效txsim的txid吗
func (h *Handler) isValidTxSim(channelID string, txid string, fmtStr string, args ...interface{}) (*TransactionContext, error) {
	txContext := h.TXContexts.Get(channelID, txid)
	if txContext == nil || txContext.TXSimulator == nil {
		err := errors.Errorf(fmtStr, args...)
		chaincodeLogger.Errorf("no ledger context: %s %s\n\n %+v", channelID, txid, err)
		return nil, err
	}
	return txContext, nil
}

//注册txid以防止来自链码的重叠句柄消息
func (h *Handler) registerTxid(msg *pb.ChaincodeMessage) bool {
//检查这是否是来自链码txid的唯一状态请求
	if h.ActiveTransactions.Add(msg.ChannelId, msg.Txid) {
		return true
	}

//记录问题并删除请求
	chaincodeName := "unknown"
	if h.chaincodeID != nil {
		chaincodeName = h.chaincodeID.Name
	}
	chaincodeLogger.Errorf("[%s] Another request pending for this CC: %s, Txid: %s, ChannelID: %s. Cannot process.", shorttxid(msg.Txid), chaincodeName, msg.Txid, msg.ChannelId)
	return false
}

func (h *Handler) checkMetadataCap(msg *pb.ChaincodeMessage) error {
	ac, exists := h.AppConfig.GetApplicationConfig(msg.ChannelId)
	if !exists {
		return errors.Errorf("application config does not exist for %s", msg.ChannelId)
	}

	if !ac.Capabilities().KeyLevelEndorsement() {
		return errors.New("key level endorsement is not enabled")
	}
	return nil
}

func errorIfCreatorHasNoReadAccess(chaincodeName, collection string, txContext *TransactionContext) error {
	accessAllowed, err := hasReadAccess(chaincodeName, collection, txContext)
	if err != nil {
		return err
	}
	if !accessAllowed {
		return errors.Errorf("tx creator does not have read access permission on privatedata in chaincodeName:%s collectionName: %s",
			chaincodeName, collection)
	}
	return nil
}

func hasReadAccess(chaincodeName, collection string, txContext *TransactionContext) (bool, error) {
//检查是否已在此链码模拟范围内检查了读取访问
	if txContext.AllowedCollectionAccess[collection] {
		return true, nil
	}

	cc := common.CollectionCriteria{
		Channel:    txContext.ChainID,
		Namespace:  chaincodeName,
		Collection: collection,
	}

	accessAllowed, err := txContext.CollectionStore.HasReadAccess(cc, txContext.SignedProp, txContext.TXSimulator)
	if err != nil {
		return false, err
	}
	if accessAllowed {
		txContext.AllowedCollectionAccess[collection] = accessAllowed
	}

	return accessAllowed, err
}

//处理对分类帐的查询以获取状态
func (h *Handler) HandleGetState(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	getState := &pb.GetState{}
	err := proto.Unmarshal(msg.Payload, getState)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	var res []byte
	chaincodeName := h.ChaincodeName()
	collection := getState.Collection
	chaincodeLogger.Debugf("[%s] getting state for chaincode %s, key %s, channel %s", shorttxid(msg.Txid), chaincodeName, getState.Key, txContext.ChainID)

	if isCollectionSet(collection) {
		if txContext.IsInitTransaction {
			return nil, errors.New("private data APIs are not allowed in chaincode Init()")
		}
		if err := errorIfCreatorHasNoReadAccess(chaincodeName, collection, txContext); err != nil {
			return nil, err
		}
		res, err = txContext.TXSimulator.GetPrivateData(chaincodeName, collection, getState.Key)
	} else {
		res, err = txContext.TXSimulator.GetState(chaincodeName, getState.Key)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if res == nil {
		chaincodeLogger.Debugf("[%s] No state associated with key: %s. Sending %s with an empty payload", shorttxid(msg.Txid), getState.Key, pb.ChaincodeMessage_RESPONSE)
	}

//
	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: res, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

//
func (h *Handler) HandleGetStateMetadata(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	err := h.checkMetadataCap(msg)
	if err != nil {
		return nil, err
	}

	getStateMetadata := &pb.GetStateMetadata{}
	err = proto.Unmarshal(msg.Payload, getStateMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	chaincodeName := h.ChaincodeName()
	collection := getStateMetadata.Collection
	chaincodeLogger.Debugf("[%s] getting state metadata for chaincode %s, key %s, channel %s", shorttxid(msg.Txid), chaincodeName, getStateMetadata.Key, txContext.ChainID)

	var metadata map[string][]byte
	if isCollectionSet(collection) {
		if txContext.IsInitTransaction {
			return nil, errors.New("private data APIs are not allowed in chaincode Init()")
		}
		if err := errorIfCreatorHasNoReadAccess(chaincodeName, collection, txContext); err != nil {
			return nil, err
		}
		metadata, err = txContext.TXSimulator.GetPrivateDataMetadata(chaincodeName, collection, getStateMetadata.Key)
	} else {
		metadata, err = txContext.TXSimulator.GetStateMetadata(chaincodeName, getStateMetadata.Key)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var metadataResult pb.StateMetadataResult
	for metakey := range metadata {
		md := &pb.StateMetadata{Metakey: metakey, Value: metadata[metakey]}
		metadataResult.Entries = append(metadataResult.Entries, md)
	}
	res, err := proto.Marshal(&metadataResult)
	if err != nil {
		return nil, errors.WithStack(err)
	}

//将响应消息发送回链码。GetState不会触发事件
	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: res, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

//处理对分类帐的查询以调整查询状态
func (h *Handler) HandleGetStateByRange(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	getStateByRange := &pb.GetStateByRange{}
	err := proto.Unmarshal(msg.Payload, getStateByRange)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	metadata, err := getQueryMetadataFromBytes(getStateByRange.Metadata)
	if err != nil {
		return nil, err
	}

	totalReturnLimit := calculateTotalReturnLimit(metadata)

	iterID := h.UUIDGenerator.New()

	var rangeIter commonledger.ResultsIterator
	var paginationInfo map[string]interface{}

	isPaginated := false

	chaincodeName := h.ChaincodeName()
	collection := getStateByRange.Collection
	if isCollectionSet(collection) {
		if txContext.IsInitTransaction {
			return nil, errors.New("private data APIs are not allowed in chaincode Init()")
		}
		if err := errorIfCreatorHasNoReadAccess(chaincodeName, collection, txContext); err != nil {
			return nil, err
		}
		rangeIter, err = txContext.TXSimulator.GetPrivateDataRangeScanIterator(chaincodeName, collection,
			getStateByRange.StartKey, getStateByRange.EndKey)
	} else if isMetadataSetForPagination(metadata) {
		paginationInfo, err = createPaginationInfoFromMetadata(metadata, totalReturnLimit, pb.ChaincodeMessage_GET_STATE_BY_RANGE)
		if err != nil {
			return nil, err
		}
		isPaginated = true

		startKey := getStateByRange.StartKey

		if isMetadataSetForPagination(metadata) {
			if metadata.Bookmark != "" {
				startKey = metadata.Bookmark
			}
		}
		rangeIter, err = txContext.TXSimulator.GetStateRangeScanIteratorWithMetadata(chaincodeName,
			startKey, getStateByRange.EndKey, paginationInfo)
	} else {
		rangeIter, err = txContext.TXSimulator.GetStateRangeScanIterator(chaincodeName, getStateByRange.StartKey, getStateByRange.EndKey)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	txContext.InitializeQueryContext(iterID, rangeIter)

	payload, err := h.QueryResponseBuilder.BuildQueryResponse(txContext, rangeIter, iterID, isPaginated, totalReturnLimit)
	if err != nil {
		txContext.CleanupQueryContext(iterID)
		return nil, errors.WithStack(err)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		txContext.CleanupQueryContext(iterID)
		return nil, errors.Wrap(err, "marshal failed")
	}

	chaincodeLogger.Debugf("Got keys and values. Sending %s", pb.ChaincodeMessage_RESPONSE)
	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: payloadBytes, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

//处理下一个查询状态的分类帐查询
func (h *Handler) HandleQueryStateNext(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	queryStateNext := &pb.QueryStateNext{}
	err := proto.Unmarshal(msg.Payload, queryStateNext)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	queryIter := txContext.GetQueryIterator(queryStateNext.Id)
	if queryIter == nil {
		return nil, errors.New("query iterator not found")
	}

	totalReturnLimit := calculateTotalReturnLimit(nil)

	payload, err := h.QueryResponseBuilder.BuildQueryResponse(txContext, queryIter, queryStateNext.Id, false, totalReturnLimit)
	if err != nil {
		txContext.CleanupQueryContext(queryStateNext.Id)
		return nil, errors.WithStack(err)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		txContext.CleanupQueryContext(queryStateNext.Id)
		return nil, errors.Wrap(err, "marshal failed")
	}

	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: payloadBytes, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

//处理状态迭代器的关闭
func (h *Handler) HandleQueryStateClose(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	queryStateClose := &pb.QueryStateClose{}
	err := proto.Unmarshal(msg.Payload, queryStateClose)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	iter := txContext.GetQueryIterator(queryStateClose.Id)
	if iter != nil {
		txContext.CleanupQueryContext(queryStateClose.Id)
	}

	payload := &pb.QueryResponse{HasMore: false, Id: queryStateClose.Id}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshal failed")
	}

	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: payloadBytes, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

//处理对分类帐的查询以执行查询状态
func (h *Handler) HandleGetQueryResult(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	iterID := h.UUIDGenerator.New()

	getQueryResult := &pb.GetQueryResult{}
	err := proto.Unmarshal(msg.Payload, getQueryResult)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	metadata, err := getQueryMetadataFromBytes(getQueryResult.Metadata)
	if err != nil {
		return nil, err
	}

	totalReturnLimit := calculateTotalReturnLimit(metadata)
	isPaginated := false

	var executeIter commonledger.ResultsIterator
	var paginationInfo map[string]interface{}

	chaincodeName := h.ChaincodeName()
	collection := getQueryResult.Collection
	if isCollectionSet(collection) {
		if txContext.IsInitTransaction {
			return nil, errors.New("private data APIs are not allowed in chaincode Init()")
		}
		if err := errorIfCreatorHasNoReadAccess(chaincodeName, collection, txContext); err != nil {
			return nil, err
		}
		executeIter, err = txContext.TXSimulator.ExecuteQueryOnPrivateData(chaincodeName, collection, getQueryResult.Query)
	} else if isMetadataSetForPagination(metadata) {
		paginationInfo, err = createPaginationInfoFromMetadata(metadata, totalReturnLimit, pb.ChaincodeMessage_GET_QUERY_RESULT)
		if err != nil {
			return nil, err
		}
		isPaginated = true
		executeIter, err = txContext.TXSimulator.ExecuteQueryWithMetadata(chaincodeName,
			getQueryResult.Query, paginationInfo)

	} else {
		executeIter, err = txContext.TXSimulator.ExecuteQuery(chaincodeName, getQueryResult.Query)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

	txContext.InitializeQueryContext(iterID, executeIter)

	payload, err := h.QueryResponseBuilder.BuildQueryResponse(txContext, executeIter, iterID, isPaginated, totalReturnLimit)
	if err != nil {
		txContext.CleanupQueryContext(iterID)
		return nil, errors.WithStack(err)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		txContext.CleanupQueryContext(iterID)
		return nil, errors.Wrap(err, "marshal failed")
	}

	chaincodeLogger.Debugf("Got keys and values. Sending %s", pb.ChaincodeMessage_RESPONSE)
	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: payloadBytes, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

//处理对分类帐历史记录数据库的查询
func (h *Handler) HandleGetHistoryForKey(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	iterID := h.UUIDGenerator.New()
	chaincodeName := h.ChaincodeName()

	getHistoryForKey := &pb.GetHistoryForKey{}
	err := proto.Unmarshal(msg.Payload, getHistoryForKey)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	historyIter, err := txContext.HistoryQueryExecutor.GetHistoryForKey(chaincodeName, getHistoryForKey.Key)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	totalReturnLimit := calculateTotalReturnLimit(nil)

	txContext.InitializeQueryContext(iterID, historyIter)
	payload, err := h.QueryResponseBuilder.BuildQueryResponse(txContext, historyIter, iterID, false, totalReturnLimit)
	if err != nil {
		txContext.CleanupQueryContext(iterID)
		return nil, errors.WithStack(err)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		txContext.CleanupQueryContext(iterID)
		return nil, errors.Wrap(err, "marshal failed")
	}

	chaincodeLogger.Debugf("Got keys and values. Sending %s", pb.ChaincodeMessage_RESPONSE)
	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: payloadBytes, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

func isCollectionSet(collection string) bool {
	return collection != ""
}

func isMetadataSetForPagination(metadata *pb.QueryMetadata) bool {
	if metadata == nil {
		return false
	}

	if metadata.PageSize == 0 && metadata.Bookmark == "" {
		return false
	}

	return true
}

func getQueryMetadataFromBytes(metadataBytes []byte) (*pb.QueryMetadata, error) {
	if metadataBytes != nil {
		metadata := &pb.QueryMetadata{}
		err := proto.Unmarshal(metadataBytes, metadata)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal failed")
		}
		return metadata, nil
	}
	return nil, nil
}

func createPaginationInfoFromMetadata(metadata *pb.QueryMetadata, totalReturnLimit int32, queryType pb.ChaincodeMessage_Type) (map[string]interface{}, error) {
	paginationInfoMap := make(map[string]interface{})

	switch queryType {
	case pb.ChaincodeMessage_GET_QUERY_RESULT:
		paginationInfoMap["bookmark"] = metadata.Bookmark
	case pb.ChaincodeMessage_GET_STATE_BY_RANGE:
//这是范围查询的非操作
	default:
		return nil, errors.New("query type must be either GetQueryResult or GetStateByRange")
	}

	paginationInfoMap["limit"] = totalReturnLimit
	return paginationInfoMap, nil
}

func calculateTotalReturnLimit(metadata *pb.QueryMetadata) int32 {
	totalReturnLimit := int32(ledgerconfig.GetTotalQueryLimit())
	if metadata != nil {
		pageSize := int32(metadata.PageSize)
		if pageSize > 0 && pageSize < totalReturnLimit {
			totalReturnLimit = pageSize
		}
	}
	return totalReturnLimit
}

func (h *Handler) getTxContextForInvoke(channelID string, txid string, payload []byte, format string, args ...interface{}) (*TransactionContext, error) {
//如果我们有通道ID，只需从isvalidtxsim获取txsim
	if channelID != "" {
		return h.isValidTxSim(channelID, txid, "could not get valid transaction")
	}

	chaincodeSpec := &pb.ChaincodeSpec{}
	err := proto.Unmarshal(payload, chaincodeSpec)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

//获取要调用的chaincodeid。要调用的chaincodeid可以
//包含复合信息，如“chaincode name:version/channel name”
//我们现在不使用版本，但默认为最新版本
	targetInstance := ParseName(chaincodeSpec.ChaincodeId.Name)

//如果TargetInstance不是SCC，则应调用ISvalidTxsim，这将返回错误。
//我们不想在原始调用是SCC时将调用传播到用户ccs
//没有通道上下文（即没有分类帐上下文）。
	if !h.SystemCCProvider.IsSysCC(targetInstance.ChaincodeName) {
//正常路径-具有空（“”）通道的UCC调用：ISvalidTxsim将返回错误
		return h.isValidTxSim("", txid, "could not get valid transaction")
	}

//在没有chainID的情况下调用scc，那么假设这是客户端调用的外部scc（特殊情况），不涉及ucc，
//因此不需要事务模拟器验证，因为没有提交到分类帐，如果不为零，则直接获取txContext
	txContext := h.TXContexts.Get(channelID, txid)
	if txContext == nil {
		return nil, errors.New("failed to get transaction context")
	}

	return txContext, nil
}

func (h *Handler) HandlePutState(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	putState := &pb.PutState{}
	err := proto.Unmarshal(msg.Payload, putState)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	chaincodeName := h.ChaincodeName()
	collection := putState.Collection
	if isCollectionSet(collection) {
		if txContext.IsInitTransaction {
			return nil, errors.New("private data APIs are not allowed in chaincode Init()")
		}
		err = txContext.TXSimulator.SetPrivateData(chaincodeName, collection, putState.Key, putState.Value)
	} else {
		err = txContext.TXSimulator.SetState(chaincodeName, putState.Key, putState.Value)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

func (h *Handler) HandlePutStateMetadata(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	err := h.checkMetadataCap(msg)
	if err != nil {
		return nil, err
	}

	putStateMetadata := &pb.PutStateMetadata{}
	err = proto.Unmarshal(msg.Payload, putStateMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	metadata := make(map[string][]byte)
	metadata[putStateMetadata.Metadata.Metakey] = putStateMetadata.Metadata.Value

	chaincodeName := h.ChaincodeName()
	collection := putStateMetadata.Collection
	if isCollectionSet(collection) {
		if txContext.IsInitTransaction {
			return nil, errors.New("private data APIs are not allowed in chaincode Init()")
		}
		err = txContext.TXSimulator.SetPrivateDataMetadata(chaincodeName, collection, putStateMetadata.Key, metadata)
	} else {
		err = txContext.TXSimulator.SetStateMetadata(chaincodeName, putStateMetadata.Key, metadata)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

func (h *Handler) HandleDelState(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	delState := &pb.DelState{}
	err := proto.Unmarshal(msg.Payload, delState)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

	chaincodeName := h.ChaincodeName()
	collection := delState.Collection
	if isCollectionSet(collection) {
		if txContext.IsInitTransaction {
			return nil, errors.New("private data APIs are not allowed in chaincode Init()")
		}
		err = txContext.TXSimulator.DeletePrivateData(chaincodeName, collection, delState.Key)
	} else {
		err = txContext.TXSimulator.DeleteState(chaincodeName, delState.Key)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

//将响应消息发送回链码。
	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

//处理修改分类帐状态的请求
func (h *Handler) HandleInvokeChaincode(msg *pb.ChaincodeMessage, txContext *TransactionContext) (*pb.ChaincodeMessage, error) {
	chaincodeLogger.Debugf("[%s] C-call-C", shorttxid(msg.Txid))

	chaincodeSpec := &pb.ChaincodeSpec{}
	err := proto.Unmarshal(msg.Payload, chaincodeSpec)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal failed")
	}

//获取要调用的chaincodeid。要调用的chaincodeid可以
//包含“chaincode name:version/channel name”等复合信息。
//我们现在不使用版本，但默认为最新版本。
	targetInstance := ParseName(chaincodeSpec.ChaincodeId.Name)
	chaincodeSpec.ChaincodeId.Name = targetInstance.ChaincodeName
	if targetInstance.ChainID == "" {
//使用调用方的通道，因为被调用的链码在同一个通道中
		targetInstance.ChainID = txContext.ChainID
	}
	chaincodeLogger.Debugf("[%s] C-call-C %s on channel %s", shorttxid(msg.Txid), targetInstance.ChaincodeName, targetInstance.ChainID)

	err = h.checkACL(txContext.SignedProp, txContext.Proposal, targetInstance)
	if err != nil {
		chaincodeLogger.Errorf(
			"[%s] C-call-C %s on channel %s failed check ACL [%v]: [%s]",
			shorttxid(msg.Txid),
			targetInstance.ChaincodeName,
			targetInstance.ChainID,
			txContext.SignedProp,
			err,
		)
		return nil, errors.WithStack(err)
	}

//如果在其他通道上，则为被调用的链代码设置新上下文
//我们抓取被调用频道的分类帐模拟器来保持新状态
	txParams := &ccprovider.TransactionParams{
		TxID:                 msg.Txid,
		ChannelID:            targetInstance.ChainID,
		SignedProp:           txContext.SignedProp,
		Proposal:             txContext.Proposal,
		TXSimulator:          txContext.TXSimulator,
		HistoryQueryExecutor: txContext.HistoryQueryExecutor,
	}

	if targetInstance.ChainID != txContext.ChainID {
		lgr := h.LedgerGetter.GetLedger(targetInstance.ChainID)
		if lgr == nil {
			return nil, errors.Errorf("failed to find ledger for channel: %s", targetInstance.ChainID)
		}

		sim, err := lgr.NewTxSimulator(msg.Txid)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		defer sim.Done()

		hqe, err := lgr.NewHistoryQueryExecutor()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		txParams.TXSimulator = sim
		txParams.HistoryQueryExecutor = hqe
	}

	chaincodeLogger.Debugf("[%s] getting chaincode data for %s on channel %s", shorttxid(msg.Txid), targetInstance.ChaincodeName, targetInstance.ChainID)

	version := h.SystemCCVersion
	if !h.SystemCCProvider.IsSysCC(targetInstance.ChaincodeName) {
//如果是用户链代码，请获取详细信息
		cd, err := h.DefinitionGetter.ChaincodeDefinition(targetInstance.ChaincodeName, txParams.TXSimulator)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		version = cd.CCVersion()

		err = h.InstantiationPolicyChecker.CheckInstantiationPolicy(targetInstance.ChaincodeName, version, cd.(*ccprovider.ChaincodeData))
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

//
	chaincodeLogger.Debugf("[%s] launching chaincode %s on channel %s", shorttxid(msg.Txid), targetInstance.ChaincodeName, targetInstance.ChainID)

	cccid := &ccprovider.CCContext{
		Name:    targetInstance.ChaincodeName,
		Version: version,
	}

//执行链码…至少现在不能是init
	responseMessage, err := h.Invoker.Invoke(txParams, cccid, chaincodeSpec.Input)
	if err != nil {
		return nil, errors.Wrap(err, "execute failed")
	}

//有效负载被编组并发送到调用链代码的填充程序，该填充程序取消标记和
//发送到链码
	res, err := proto.Marshal(responseMessage)
	if err != nil {
		return nil, errors.Wrap(err, "marshal failed")
	}

	return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE, Payload: res, Txid: msg.Txid, ChannelId: msg.ChannelId}, nil
}

func (h *Handler) Execute(txParams *ccprovider.TransactionParams, cccid *ccprovider.CCContext, msg *pb.ChaincodeMessage, timeout time.Duration) (*pb.ChaincodeMessage, error) {
	chaincodeLogger.Debugf("Entry")
	defer chaincodeLogger.Debugf("Exit")

	txParams.CollectionStore = h.getCollectionStore(msg.ChannelId)
	txParams.IsInitTransaction = (msg.Type == pb.ChaincodeMessage_INIT)

	txctx, err := h.TXContexts.Create(txParams)
	if err != nil {
		return nil, err
	}
	defer h.TXContexts.Delete(msg.ChannelId, msg.Txid)

	if err := h.setChaincodeProposal(txParams.SignedProp, txParams.Proposal, msg); err != nil {
		return nil, err
	}

	h.serialSendAsync(msg)

	var ccresp *pb.ChaincodeMessage
	select {
	case ccresp = <-txctx.ResponseNotifier:
//响应被发送到用户或调用链码。链码信息\错误
//通常被视为错误
	case <-time.After(timeout):
		err = errors.New("timeout expired while executing transaction")
		ccName := cccid.Name + ":" + cccid.Version
		h.Metrics.ExecuteTimeouts.With(
			"chaincode", ccName,
		).Add(1)
	}

	return ccresp, err
}

func (h *Handler) setChaincodeProposal(signedProp *pb.SignedProposal, prop *pb.Proposal, msg *pb.ChaincodeMessage) error {
	if prop != nil && signedProp == nil {
		return errors.New("failed getting proposal context. Signed proposal is nil")
	}
//托多：这没什么意义。感觉两者都是必需的或者
//两者都不应设置。请咨询一位知识渊博的专家。
	if prop != nil {
		msg.Proposal = signedProp
	}
	return nil
}

func (h *Handler) getCollectionStore(channelID string) privdata.CollectionStore {
	csStoreSupport := &peer.CollectionSupport{
		PeerLedger: h.LedgerGetter.GetLedger(channelID),
	}
	return privdata.NewSimpleCollectionStore(csStoreSupport)

}

func (h *Handler) State() State { return h.state }
func (h *Handler) Close()       { h.TXContexts.Close() }

type State int

const (
	Created State = iota
	Established
	Ready
)

func (s State) String() string {
	switch s {
	case Created:
		return "created"
	case Established:
		return "established"
	case Ready:
		return "ready"
	default:
		return "UNKNOWN"
	}
}
