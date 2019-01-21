
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


package state

import (
	"bytes"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/golang/protobuf/proto"
	vsccErrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	common2 "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//gossipstateprovider是获取分类账块序列的接口
//能够通过运行状态复制和
//正在向其他节点发送获取丢失块的请求
type GossipStateProvider interface {
	AddPayload(payload *proto.Payload) error

//停止终止状态传输对象
	Stop()
}

const (
	defAntiEntropyInterval             = 10 * time.Second
	defAntiEntropyStateResponseTimeout = 3 * time.Second
	defAntiEntropyBatchSize            = 10

	defChannelBufferSize     = 100
	defAntiEntropyMaxRetries = 3

	defMaxBlockDistance = 100

	blocking    = true
	nonBlocking = false

	enqueueRetryInterval = time.Millisecond * 100
)

//gossipadapter定义状态提供程序所需的八卦/通信接口
type GossipAdapter interface {
//发送向远程对等发送消息
	Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer)

//accept返回与某个谓词匹配的其他节点发送的消息的专用只读通道。
//如果passthrough为false，则消息将由八卦层预先处理。
//如果passthrough是真的，那么八卦层不会介入，消息也不会
//可用于将答复发送回发件人
	Accept(acceptor common2.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage)

//更新LedgerHeight更新Ledger Height the Peer
//发布到频道中的其他对等端
	UpdateLedgerHeight(height uint64, chainID common2.ChainID)

//peersofchannel返回被认为是活动的网络成员
//也订阅了给定的频道
	PeersOfChannel(common2.ChainID) []discovery.NetworkMember
}

//消息加密服务接口的mcsadapter适配器到绑定
//状态转移服务所需的特定API
type MCSAdapter interface {
//如果块已正确签名，并且声明的seqnum是
//块头包含的序列号。
//else返回错误
	VerifyBlock(chainID common2.ChainID, seqNum uint64, signedBlock []byte) error

//VerifyByChannel检查签名是否为消息的有效签名
//在对等机的验证密钥下，也在特定通道的上下文中。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为零，则验证失败。
	VerifyByChannel(chainID common2.ChainID, peerIdentity api.PeerIdentityType, signature, message []byte) error
}

//LedgerResources定义Ledger提供的功能
type ledgerResources interface {
//StoreBlock提供带有下划线的私有数据的新块
//返回缺少的事务ID
	StoreBlock(block *common.Block, data util.PvtDataCollections) error

//storepvdtdata用于将私有日期持久化到临时存储中
	StorePvtData(txid string, privData *transientstore.TxPvtReadWriteSetWithConfigInfo, blckHeight uint64) error

//getpvtdata和blockbynum按编号获取block并返回所有相关的私有数据
//pvtDataCollections切片中私有数据的顺序并不意味着
//块中与这些私有数据相关的事务，以获得正确的位置
//需要读取txpvtdata.seqinblock字段
	GetPvtDataAndBlockByNum(seqNum uint64, peerAuthInfo common.SignedData) (*common.Block, util.PvtDataCollections, error)

//获取最近的块序列号
	LedgerHeight() (uint64, error)

//关闭分类帐资源
	Close()
}

//ServicesMediator聚合适配器以组合所有中介
//状态转换为单个结构所需
type ServicesMediator struct {
	GossipAdapter
	MCSAdapter
}

//gossipstateproviderMPL gossipstateprovider接口的实现
//在内存中处理的结构
//要由超级分类帐获取的新分类帐块
type GossipStateProviderImpl struct {
//链ID
	chainID string

	mediator *ServicesMediator

//阅读八卦信息的渠道
	gossipChan <-chan *proto.GossipMessage

	commChan <-chan proto.ReceivedMessage

//尚未获得的有效载荷队列
	payloads PayloadsBuffer

	ledger ledgerResources

	stateResponseCh chan proto.ReceivedMessage

	stateRequestCh chan proto.ReceivedMessage

	stopCh chan struct{}

	done sync.WaitGroup

	once sync.Once

	stateTransferActive int32
}

var logger = util.GetLogger(util.StateLogger, "")

//newgossipstateprovider使用协调器实例创建状态提供程序
//在将私有RWset和块提交到分类帐之前协调它们的到达。
func NewGossipStateProvider(chainID string, services *ServicesMediator, ledger ledgerResources) GossipStateProvider {

	gossipChan, _ := services.Accept(func(message interface{}) bool {
//仅获取数据消息
		return message.(*proto.GossipMessage).IsDataMsg() &&
			bytes.Equal(message.(*proto.GossipMessage).Channel, []byte(chainID))
	}, false)

	remoteStateMsgFilter := func(message interface{}) bool {
		receivedMsg := message.(proto.ReceivedMessage)
		msg := receivedMsg.GetGossipMessage()
		if !(msg.IsRemoteStateMessage() || msg.GetPrivateData() != nil) {
			return false
		}
//确保我们只处理属于此频道的消息
		if !bytes.Equal(msg.Channel, []byte(chainID)) {
			return false
		}
		connInfo := receivedMsg.GetConnectionInfo()
		authErr := services.VerifyByChannel(msg.Channel, connInfo.Identity, connInfo.Auth.Signature, connInfo.Auth.SignedData)
		if authErr != nil {
			logger.Warning("Got unauthorized request from", string(connInfo.Identity))
			return false
		}
		return true
	}

//仅与节点状态传输相关的筛选消息
	_, commChan := services.Accept(remoteStateMsgFilter, true)

	height, err := ledger.LedgerHeight()
	if height == 0 {
//这里有恐慌，因为这是无效情况的迹象，正常情况下不应该发生。
//代码路径。
		logger.Panic("Committer height cannot be zero, ledger should include at least one block (genesis).")
	}

	if err != nil {
		logger.Error("Could not read ledger info to obtain current ledger height due to: ", errors.WithStack(err))
//如果没有分类账，就不可能退出。
//交付新块
		return nil
	}

	s := &GossipStateProviderImpl{
//消息加密服务
		mediator: services,

//链ID
		chainID: chainID,

//从中读取新消息的通道
		gossipChan: gossipChan,

//从其他对等方读取直接消息的通道
		commChan: commChan,

//为接收的负载创建队列
		payloads: NewPayloadsBuffer(height),

		ledger: ledger,

		stateResponseCh: make(chan proto.ReceivedMessage, defChannelBufferSize),

		stateRequestCh: make(chan proto.ReceivedMessage, defChannelBufferSize),

		stopCh: make(chan struct{}, 1),

		stateTransferActive: 0,

		once: sync.Once{},
	}

	logger.Infof("Updating metadata information, "+
		"current ledger sequence is at = %d, next expected block is = %d", height-1, s.payloads.Next())
	logger.Debug("Updating gossip ledger height to", height)
	services.UpdateLedgerHeight(height, common2.ChainID(s.chainID))

	s.done.Add(4)

//监听传入的通信
	go s.listen()
//按顺序将消息传递到传入通道
	go s.deliverPayloads()
//执行反熵以填补缺失的空白
	go s.antiEntropy()
//处理状态请求消息
	go s.processStateRequests()

	return s
}

func (s *GossipStateProviderImpl) listen() {
	defer s.done.Done()

	for {
		select {
		case msg := <-s.gossipChan:
			logger.Debug("Received new message via gossip channel")
			go s.queueNewMessage(msg)
		case msg := <-s.commChan:
			logger.Debug("Dispatching a message", msg)
			go s.dispatch(msg)
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			logger.Debug("Stop listening for new messages")
			return
		}
	}
}
func (s *GossipStateProviderImpl) dispatch(msg proto.ReceivedMessage) {
//检查消息类型
	if msg.GetGossipMessage().IsRemoteStateMessage() {
		logger.Debug("Handling direct state transfer message")
//得到状态转移请求响应
		s.directMessage(msg)
	} else if msg.GetGossipMessage().GetPrivateData() != nil {
		logger.Debug("Handling private data collection message")
//处理私有数据复制消息
		s.privateDataMessage(msg)
	}

}
func (s *GossipStateProviderImpl) privateDataMessage(msg proto.ReceivedMessage) {
	if !bytes.Equal(msg.GetGossipMessage().Channel, []byte(s.chainID)) {
		logger.Warning("Received state transfer request for channel",
			string(msg.GetGossipMessage().Channel), "while expecting channel", s.chainID, "skipping request...")
		return
	}

	gossipMsg := msg.GetGossipMessage()
	pvtDataMsg := gossipMsg.GetPrivateData()

	if pvtDataMsg.Payload == nil {
		logger.Warning("Malformed private data message, no payload provided")
		return
	}

	collectionName := pvtDataMsg.Payload.CollectionName
	txID := pvtDataMsg.Payload.TxId
	pvtRwSet := pvtDataMsg.Payload.PrivateRwset

	if len(pvtRwSet) == 0 {
		logger.Warning("Malformed private data message, no rwset provided, collection name = ", collectionName)
		return
	}

	txPvtRwSet := &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{{
			Namespace: pvtDataMsg.Payload.Namespace,
			CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{{
				CollectionName: collectionName,
				Rwset:          pvtRwSet,
			}}},
		},
	}

	txPvtRwSetWithConfig := &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: txPvtRwSet,
		CollectionConfigs: map[string]*common.CollectionConfigPackage{
			pvtDataMsg.Payload.Namespace: pvtDataMsg.Payload.CollectionConfigs,
		},
	}

	if err := s.ledger.StorePvtData(txID, txPvtRwSetWithConfig, pvtDataMsg.Payload.PrivateSimHeight); err != nil {
		logger.Errorf("Wasn't able to persist private data for collection %s, due to %s", collectionName, err)
msg.Ack(err) //发送nack以指示存储收集失败
	}

	msg.Ack(nil)
	logger.Debug("Private data for collection", collectionName, "has been stored")
}

func (s *GossipStateProviderImpl) directMessage(msg proto.ReceivedMessage) {
	logger.Debug("[ENTER] -> directMessage")
	defer logger.Debug("[EXIT] ->  directMessage")

	if msg == nil {
		logger.Error("Got nil message via end-to-end channel, should not happen!")
		return
	}

	if !bytes.Equal(msg.GetGossipMessage().Channel, []byte(s.chainID)) {
		logger.Warning("Received state transfer request for channel",
			string(msg.GetGossipMessage().Channel), "while expecting channel", s.chainID, "skipping request...")
		return
	}

	incoming := msg.GetGossipMessage()

	if incoming.GetStateRequest() != nil {
		if len(s.stateRequestCh) < defChannelBufferSize {
//将状态请求转发到通道（如果有的话）
//许多状态请求消息都忽略了以避免洪水泛滥。
			s.stateRequestCh <- msg
		}
	} else if incoming.GetStateResponse() != nil {
//如果没有状态转移程序激活，则
//没有理由处理消息
		if atomic.LoadInt32(&s.stateTransferActive) == 1 {
//发送状态响应报文信号
			s.stateResponseCh <- msg
		}
	}
}

func (s *GossipStateProviderImpl) processStateRequests() {
	defer s.done.Done()

	for {
		select {
		case msg := <-s.stateRequestCh:
			s.handleStateRequest(msg)
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			return
		}
	}
}

//handleStateRequest处理状态请求消息，验证批大小，将当前领导状态读取到
//获取所需的块，生成响应消息并将其发回
func (s *GossipStateProviderImpl) handleStateRequest(msg proto.ReceivedMessage) {
	if msg == nil {
		return
	}
	request := msg.GetGossipMessage().GetStateRequest()

	batchSize := request.EndSeqNum - request.StartSeqNum
	if batchSize > defAntiEntropyBatchSize {
		logger.Errorf("Requesting blocks batchSize size (%d) greater than configured allowed"+
			" (%d) batching for anti-entropy. Ignoring request...", batchSize, defAntiEntropyBatchSize)
		return
	}

	if request.StartSeqNum > request.EndSeqNum {
		logger.Errorf("Invalid sequence interval [%d...%d], ignoring request...", request.StartSeqNum, request.EndSeqNum)
		return
	}

	currentHeight, err := s.ledger.LedgerHeight()
	if err != nil {
		logger.Errorf("Cannot access to current ledger height, due to %+v", errors.WithStack(err))
		return
	}
	if currentHeight < request.EndSeqNum {
		logger.Warningf("Received state request to transfer blocks with sequence numbers higher  [%d...%d] "+
			"than available in ledger (%d)", request.StartSeqNum, request.StartSeqNum, currentHeight)
	}

	endSeqNum := min(currentHeight, request.EndSeqNum)

	response := &proto.RemoteStateResponse{Payloads: make([]*proto.Payload, 0)}
	for seqNum := request.StartSeqNum; seqNum <= endSeqNum; seqNum++ {
		logger.Debug("Reading block ", seqNum, " with private data from the coordinator service")
		connInfo := msg.GetConnectionInfo()
		peerAuthInfo := common.SignedData{
			Data:      connInfo.Auth.SignedData,
			Signature: connInfo.Auth.Signature,
			Identity:  connInfo.Identity,
		}
		block, pvtData, err := s.ledger.GetPvtDataAndBlockByNum(seqNum, peerAuthInfo)

		if err != nil {
			logger.Errorf("cannot read block number %d from ledger, because %+v, skipping...", seqNum, err)
			continue
		}

		if block == nil {
			logger.Errorf("Wasn't able to read block with sequence number %d from ledger, skipping....", seqNum)
			continue
		}

		blockBytes, err := pb.Marshal(block)

		if err != nil {
			logger.Errorf("Could not marshal block: %+v", errors.WithStack(err))
			continue
		}

		var pvtBytes [][]byte
		if pvtData != nil {
//封送私有数据
			pvtBytes, err = pvtData.Marshal()
			if err != nil {
				logger.Errorf("Failed to marshal private rwset for block %d due to %+v", seqNum, errors.WithStack(err))
				continue
			}
		}

//将结果附加到响应
		response.Payloads = append(response.Payloads, &proto.Payload{
			SeqNum:      seqNum,
			Data:        blockBytes,
			PrivateData: pvtBytes,
		})
	}
//发送缺少块的响应
	msg.Respond(&proto.GossipMessage{
//从请求中复制nonce字段，以便能够匹配响应
		Nonce:   msg.GetGossipMessage().Nonce,
		Tag:     proto.GossipMessage_CHAN_OR_ORG,
		Channel: []byte(s.chainID),
		Content: &proto.GossipMessage_StateResponse{StateResponse: response},
	})
}

func (s *GossipStateProviderImpl) handleStateResponse(msg proto.ReceivedMessage) (uint64, error) {
	max := uint64(0)
//发送已收到给定nonce响应的信号
	response := msg.GetGossipMessage().GetStateResponse()
//提取有效负载，验证并推入缓冲区
	if len(response.GetPayloads()) == 0 {
		return uint64(0), errors.New("Received state transfer response without payload")
	}
	for _, payload := range response.GetPayloads() {
		logger.Debugf("Received payload with sequence number %d.", payload.SeqNum)
		if err := s.mediator.VerifyBlock(common2.ChainID(s.chainID), payload.SeqNum, payload.Data); err != nil {
			err = errors.WithStack(err)
			logger.Warningf("Error verifying block with sequence number %d, due to %+v", payload.SeqNum, err)
			return uint64(0), err
		}
		if max < payload.SeqNum {
			max = payload.SeqNum
		}

		err := s.addPayload(payload, blocking)
		if err != nil {
			logger.Warningf("Block [%d] received from block transfer wasn't added to payload buffer: %v", payload.SeqNum, err)
		}
	}
	return max, nil
}

//stop函数向所有go例程发送停止信号
func (s *GossipStateProviderImpl) Stop() {
//确保停止不会执行两次
//停止频道将不再使用
	s.once.Do(func() {
		s.stopCh <- struct{}{}
//确保所有的执行程序都已完成
		s.done.Wait()
//关闭所有资源
		s.ledger.Close()
		close(s.stateRequestCh)
		close(s.stateResponseCh)
		close(s.stopCh)
	})
}

//QueueNewMessage生成新消息通知/处理程序
func (s *GossipStateProviderImpl) queueNewMessage(msg *proto.GossipMessage) {
	if !bytes.Equal(msg.Channel, []byte(s.chainID)) {
		logger.Warning("Received enqueue for channel",
			string(msg.Channel), "while expecting channel", s.chainID, "ignoring enqueue")
		return
	}

	dataMsg := msg.GetDataMsg()
	if dataMsg != nil {
		if err := s.addPayload(dataMsg.GetPayload(), nonBlocking); err != nil {
			logger.Warningf("Block [%d] received from gossip wasn't added to payload buffer: %v", dataMsg.Payload.SeqNum, err)
			return
		}

	} else {
		logger.Debug("Gossip message received is not of data message type, usually this should not happen.")
	}
}

func (s *GossipStateProviderImpl) deliverPayloads() {
	defer s.done.Done()

	for {
		select {
//等待下一个序列已到达的通知
		case <-s.payloads.Ready():
			logger.Debugf("[%s] Ready to transfer payloads (blocks) to the ledger, next block number is = [%d]", s.chainID, s.payloads.Next())
//收集所有后续有效载荷
			for payload := s.payloads.Pop(); payload != nil; payload = s.payloads.Pop() {
				rawBlock := &common.Block{}
				if err := pb.Unmarshal(payload.Data, rawBlock); err != nil {
					logger.Errorf("Error getting block with seqNum = %d due to (%+v)...dropping block", payload.SeqNum, errors.WithStack(err))
					continue
				}
				if rawBlock.Data == nil || rawBlock.Header == nil {
					logger.Errorf("Block with claimed sequence %d has no header (%v) or data (%v)",
						payload.SeqNum, rawBlock.Header, rawBlock.Data)
					continue
				}
				logger.Debugf("[%s] Transferring block [%d] with %d transaction(s) to the ledger", s.chainID, payload.SeqNum, len(rawBlock.Data.Data))

//将所有私有数据读取到切片中
				var p util.PvtDataCollections
				if payload.PrivateData != nil {
					err := p.Unmarshal(payload.PrivateData)
					if err != nil {
						logger.Errorf("Wasn't able to unmarshal private data for block seqNum = %d due to (%+v)...dropping block", payload.SeqNum, errors.WithStack(err))
						continue
					}
				}
				if err := s.commitBlock(rawBlock, p); err != nil {
					if executionErr, isExecutionErr := err.(*vsccErrors.VSCCExecutionFailureError); isExecutionErr {
						logger.Errorf("Failed executing VSCC due to %v. Aborting chain processing", executionErr)
						return
					}
					logger.Panicf("Cannot commit block to the ledger due to %+v", errors.WithStack(err))
				}
			}
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			logger.Debug("State provider has been stopped, finishing to push new blocks.")
			return
		}
	}
}

func (s *GossipStateProviderImpl) antiEntropy() {
	defer s.done.Done()
	defer logger.Debug("State Provider stopped, stopping anti entropy procedure.")

	for {
		select {
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			return
		case <-time.After(defAntiEntropyInterval):
			ourHeight, err := s.ledger.LedgerHeight()
			if err != nil {
//无法从分类帐中读取继续下一轮
				logger.Errorf("Cannot obtain ledger height, due to %+v", errors.WithStack(err))
				continue
			}
			if ourHeight == 0 {
				logger.Error("Ledger reported block height of 0 but this should be impossible")
				continue
			}
			maxHeight := s.maxAvailableLedgerHeight()
			if ourHeight >= maxHeight {
				continue
			}

			s.requestBlocksInRange(uint64(ourHeight), uint64(maxHeight)-1)
		}
	}
}

//maxavailableledgerheight在所有可用的对等机上迭代，并检查公布的元状态
//跨对等方查找最大可用分类帐高度
func (s *GossipStateProviderImpl) maxAvailableLedgerHeight() uint64 {
	max := uint64(0)
	for _, p := range s.mediator.PeersOfChannel(common2.ChainID(s.chainID)) {
		if p.Properties == nil {
			logger.Debug("Peer", p.PreferredEndpoint(), "doesn't have properties, skipping it")
			continue
		}
		peerHeight := p.Properties.LedgerHeight
		if max < peerHeight {
			max = peerHeight
		}
	}
	return max
}

//RequestBlocksInRange能够用序列获取块
//范围内的数字[开始…结束]。
func (s *GossipStateProviderImpl) requestBlocksInRange(start uint64, end uint64) {
	atomic.StoreInt32(&s.stateTransferActive, 1)
	defer atomic.StoreInt32(&s.stateTransferActive, 0)

	for prev := start; prev <= end; {
		next := min(end, prev+defAntiEntropyBatchSize)

		gossipMsg := s.stateRequestMessage(prev, next)

		responseReceived := false
		tryCounts := 0

		for !responseReceived {
			if tryCounts > defAntiEntropyMaxRetries {
				logger.Warningf("Wasn't  able to get blocks in range [%d...%d), after %d retries",
					prev, next, tryCounts)
				return
			}
//选择要请求块的对等方
			peer, err := s.selectPeerToRequestFrom(next)
			if err != nil {
				logger.Warningf("Cannot send state request for blocks in range [%d...%d), due to %+v",
					prev, next, errors.WithStack(err))
				return
			}

			logger.Debugf("State transfer, with peer %s, requesting blocks in range [%d...%d), "+
				"for chainID %s", peer.Endpoint, prev, next, s.chainID)

			s.mediator.Send(gossipMsg, peer)
			tryCounts++

//等待超时或响应到达
			select {
			case msg := <-s.stateResponseCh:
				if msg.GetGossipMessage().Nonce != gossipMsg.Nonce {
					continue
				}
//得到状态请求的相应响应，可以继续
				index, err := s.handleStateResponse(msg)
				if err != nil {
					logger.Warningf("Wasn't able to process state response for "+
						"blocks [%d...%d], due to %+v", prev, next, errors.WithStack(err))
					continue
				}
				prev = index + 1
				responseReceived = true
			case <-time.After(defAntiEntropyStateResponseTimeout):
			case <-s.stopCh:
				s.stopCh <- struct{}{}
				return
			}
		}
	}
}

//StateRequestMessage为范围[BeginSeq…EndSeq]中的给定块生成状态请求消息
func (s *GossipStateProviderImpl) stateRequestMessage(beginSeq uint64, endSeq uint64) *proto.GossipMessage {
	return &proto.GossipMessage{
		Nonce:   util.RandomUInt64(),
		Tag:     proto.GossipMessage_CHAN_OR_ORG,
		Channel: []byte(s.chainID),
		Content: &proto.GossipMessage_StateRequest{
			StateRequest: &proto.RemoteStateRequest{
				StartSeqNum: beginSeq,
				EndSeqNum:   endSeq,
			},
		},
	}
}

//selectpeertorequestfrom选择具有请求丢失块所需块的对等
func (s *GossipStateProviderImpl) selectPeerToRequestFrom(height uint64) (*comm.RemotePeer, error) {
//筛选具有所需丢失块范围的对等点
	peers := s.filterPeers(s.hasRequiredHeight(height))

	n := len(peers)
	if n == 0 {
		return nil, errors.New("there are no peers to ask for missing blocks from")
	}

//选择对等请求块
	return peers[util.RandomInt(n)], nil
}

//filterpeers返回对齐所提供谓词的对等方列表
func (s *GossipStateProviderImpl) filterPeers(predicate func(peer discovery.NetworkMember) bool) []*comm.RemotePeer {
	var peers []*comm.RemotePeer

	for _, member := range s.mediator.PeersOfChannel(common2.ChainID(s.chainID)) {
		if predicate(member) {
			peers = append(peers, &comm.RemotePeer{Endpoint: member.PreferredEndpoint(), PKIID: member.PKIid})
		}
	}

	return peers
}

//hasRequiredHeight返回谓词，该谓词能够筛选分类帐高度高于指示高度的对等项
//按提供的输入参数
func (s *GossipStateProviderImpl) hasRequiredHeight(height uint64) func(peer discovery.NetworkMember) bool {
	return func(peer discovery.NetworkMember) bool {
		if peer.Properties != nil {
			return peer.Properties.LedgerHeight >= height
		}
		logger.Debug(peer.PreferredEndpoint(), "doesn't have properties")
		return false
	}
}

//AddPayload将新的有效负载添加到状态。
func (s *GossipStateProviderImpl) AddPayload(payload *proto.Payload) error {
	blockingMode := blocking
	if viper.GetBool("peer.gossip.nonBlockingCommitMode") {
		blockingMode = false
	}
	return s.addPayload(payload, blockingMode)
}

//AddPayload将新的有效负载添加到状态。它可以（或不可以）根据
//给定参数。如果它在阻塞模式下得到一个阻塞-它会等到
//块被发送到有效负载缓冲区。
//否则-如果有效负载缓冲区太满，它可能会丢弃块。
func (s *GossipStateProviderImpl) addPayload(payload *proto.Payload, blockingMode bool) error {
	if payload == nil {
		return errors.New("Given payload is nil")
	}
	logger.Debugf("[%s] Adding payload to local buffer, blockNum = [%d]", s.chainID, payload.SeqNum)
	height, err := s.ledger.LedgerHeight()
	if err != nil {
		return errors.Wrap(err, "Failed obtaining ledger height")
	}

	if !blockingMode && payload.SeqNum-height >= defMaxBlockDistance {
		return errors.Errorf("Ledger height is at %d, cannot enqueue block with sequence of %d", height, payload.SeqNum)
	}

	for blockingMode && s.payloads.Size() > defMaxBlockDistance*2 {
		time.Sleep(enqueueRetryInterval)
	}

	s.payloads.Push(payload)
	return nil
}

func (s *GossipStateProviderImpl) commitBlock(block *common.Block, pvtData util.PvtDataCollections) error {

//使用可用的私有事务提交块
	if err := s.ledger.StoreBlock(block, pvtData); err != nil {
		logger.Errorf("Got error while committing(%+v)", errors.WithStack(err))
		return err
	}

//更新分类帐高度
	s.mediator.UpdateLedgerHeight(block.Header.Number+1, common2.ChainID(s.chainID))
	logger.Debugf("[%s] Committed block [%d] with %d transaction(s)",
		s.chainID, block.Header.Number, len(block.Data.Data))

	return nil
}

func min(a uint64, b uint64) uint64 {
	return b ^ ((a ^ b) & (-(uint64(a-b) >> 63)))
}
