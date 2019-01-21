
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
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/configtx/test"
	errors2 "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/common/flogging/floggingtest"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/mocks/validator"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/privdata"
	"github.com/hyperledger/fabric/gossip/state/mocks"
	gutil "github.com/hyperledger/fabric/gossip/util"
	pcomm "github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	transientstore2 "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	portStartRange = 5610

	orgID = []byte("ORG1")

	noopPeerIdentityAcceptor = func(identity api.PeerIdentityType) error {
		return nil
	}
)

type peerIdentityAcceptor func(identity api.PeerIdentityType) error

type joinChanMsg struct {
}

func init() {
	gutil.SetupTestLogging()
	factory.InitFactories(nil)
}

//SequenceNumber返回消息
//来源于
func (*joinChanMsg) SequenceNumber() uint64 {
	return uint64(time.Now().UnixNano())
}

//成员返回频道的组织
func (jcm *joinChanMsg) Members() []api.OrgIdentityType {
	return []api.OrgIdentityType{orgID}
}

//anchor peers of返回给定组织的锚定对等方
func (jcm *joinChanMsg) AnchorPeersOf(org api.OrgIdentityType) []api.AnchorPeer {
	return []api.AnchorPeer{}
}

type orgCryptoService struct {
}

//orgByPeerIdentity返回orgIdentityType
//给定对等身份的
func (*orgCryptoService) OrgByPeerIdentity(identity api.PeerIdentityType) api.OrgIdentityType {
	return orgID
}

//验证验证JoinChannelMessage，成功时返回nil，
//失败时出错
func (*orgCryptoService) Verify(joinChanMsg api.JoinChannelMessage) error {
	return nil
}

type cryptoServiceMock struct {
	acceptor peerIdentityAcceptor
}

func (cryptoServiceMock) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	return time.Now().Add(time.Hour), nil
}

//getpkiidofcert返回对等身份的pki-id
func (*cryptoServiceMock) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return common.PKIidType(peerIdentity)
}

//verifyblock如果块被正确签名，则返回nil，
//else返回错误
func (*cryptoServiceMock) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

//用该对等方的签名密钥和输出对消息进行签名
//如果没有出现错误，则返回签名。
func (*cryptoServiceMock) Sign(msg []byte) ([]byte, error) {
	clone := make([]byte, len(msg))
	copy(clone, msg)
	return clone, nil
}

//验证检查签名是否是对等验证密钥下消息的有效签名。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peercert为nil，则根据该对等方的验证密钥验证签名。
func (*cryptoServiceMock) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	equal := bytes.Equal(signature, message)
	if !equal {
		return fmt.Errorf("Wrong signature:%v, %v", signature, message)
	}
	return nil
}

//VerifyByChannel检查签名是否为消息的有效签名
//在对等机的验证密钥下，也在特定通道的上下文中。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为nil，则使用该对等方的验证密钥验证签名。
func (cs *cryptoServiceMock) VerifyByChannel(chainID common.ChainID, peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return cs.acceptor(peerIdentity)
}

func (*cryptoServiceMock) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	return nil
}

func bootPeers(portPrefix int, ids ...int) []string {
	peers := []string{}
	for _, id := range ids {
		peers = append(peers, fmt.Sprintf("localhost:%d", id+portPrefix))
	}
	return peers
}

//简单介绍同行，仅包括
//通讯模块、八卦和状态转移
type peerNode struct {
	port   int
	g      gossip.Gossip
	s      *GossipStateProviderImpl
	cs     *cryptoServiceMock
	commit committer.Committer
}

//关闭所有使用的模块
func (node *peerNode) shutdown() {
	node.s.Stop()
	node.g.Stop()
}

type mockTransientStore struct {
}

func (*mockTransientStore) PurgeByHeight(maxBlockNumToRetain uint64) error {
	return nil
}

func (*mockTransientStore) Persist(txid string, blockHeight uint64, privateSimulationResults *rwset.TxPvtReadWriteSet) error {
	panic("implement me")
}

func (*mockTransientStore) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore2.TxPvtReadWriteSetWithConfigInfo) error {
	panic("implement me")
}

func (mockTransientStore) GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error) {
	panic("implement me")
}

func (*mockTransientStore) PurgeByTxids(txids []string) error {
	panic("implement me")
}

type mockCommitter struct {
	*mock.Mock
	sync.Mutex
}

func (mc *mockCommitter) GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error) {
	args := mc.Called()
	return args.Get(0).(ledger.ConfigHistoryRetriever), args.Error(1)
}

func (mc *mockCommitter) GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	args := mc.Called(blockNum, filter)
	return args.Get(0).([]*ledger.TxPvtData), args.Error(1)
}

func (mc *mockCommitter) CommitWithPvtData(blockAndPvtData *ledger.BlockAndPvtData) error {
	mc.Lock()
	m := mc.Mock
	mc.Unlock()
	m.Called(blockAndPvtData.Block)
	return nil
}

func (mc *mockCommitter) GetPvtDataAndBlockByNum(seqNum uint64) (*ledger.BlockAndPvtData, error) {
	args := mc.Called(seqNum)
	return args.Get(0).(*ledger.BlockAndPvtData), args.Error(1)
}

func (mc *mockCommitter) LedgerHeight() (uint64, error) {
	mc.Lock()
	m := mc.Mock
	mc.Unlock()
	args := m.Called()
	if args.Get(1) == nil {
		return args.Get(0).(uint64), nil
	}
	return args.Get(0).(uint64), args.Get(1).(error)
}

func (mc *mockCommitter) GetBlocks(blockSeqs []uint64) []*pcomm.Block {
	if mc.Called(blockSeqs).Get(0) == nil {
		return nil
	}
	return mc.Called(blockSeqs).Get(0).([]*pcomm.Block)
}

func (*mockCommitter) GetMissingPvtDataTracker() (ledger.MissingPvtDataTracker, error) {
	panic("implement me")
}

func (*mockCommitter) CommitPvtDataOfOldBlocks(blockPvtData []*ledger.BlockPvtData) ([]*ledger.PvtdataHashMismatch, error) {
	panic("implement me")
}

func (*mockCommitter) Close() {
}

type ramLedger struct {
	ledger map[uint64]*ledger.BlockAndPvtData
	sync.RWMutex
}

func (mock *ramLedger) GetMissingPvtDataTracker() (ledger.MissingPvtDataTracker, error) {
	panic("implement me")
}

func (mock *ramLedger) CommitPvtDataOfOldBlocks(blockPvtData []*ledger.BlockPvtData) ([]*ledger.PvtdataHashMismatch, error) {
	panic("implement me")
}

func (mock *ramLedger) GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error) {
	panic("implement me")
}

func (mock *ramLedger) GetPvtDataAndBlockByNum(blockNum uint64, filter ledger.PvtNsCollFilter) (*ledger.BlockAndPvtData, error) {
	mock.RLock()
	defer mock.RUnlock()

	if block, ok := mock.ledger[blockNum]; !ok {
		return nil, errors.New(fmt.Sprintf("no block with seq = %d found", blockNum))
	} else {
		return block, nil
	}
}

func (mock *ramLedger) GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	panic("implement me")
}

func (mock *ramLedger) CommitWithPvtData(blockAndPvtdata *ledger.BlockAndPvtData) error {
	mock.Lock()
	defer mock.Unlock()

	if blockAndPvtdata != nil && blockAndPvtdata.Block != nil {
		mock.ledger[blockAndPvtdata.Block.Header.Number] = blockAndPvtdata
		return nil
	}
	return errors.New("invalid input parameters for block and private data param")
}

func (mock *ramLedger) GetBlockchainInfo() (*pcomm.BlockchainInfo, error) {
	mock.RLock()
	defer mock.RUnlock()

	currentBlock := mock.ledger[uint64(len(mock.ledger)-1)].Block
	return &pcomm.BlockchainInfo{
		Height:            currentBlock.Header.Number + 1,
		CurrentBlockHash:  currentBlock.Header.Hash(),
		PreviousBlockHash: currentBlock.Header.PreviousHash,
	}, nil
}

func (mock *ramLedger) GetBlockByNumber(blockNumber uint64) (*pcomm.Block, error) {
	mock.RLock()
	defer mock.RUnlock()

	if blockAndPvtData, ok := mock.ledger[blockNumber]; !ok {
		return nil, errors.New(fmt.Sprintf("no block with seq = %d found", blockNumber))
	} else {
		return blockAndPvtData.Block, nil
	}
}

func (mock *ramLedger) Close() {

}

//用于八卦和通信模块的默认配置
func newGossipConfig(portPrefix, id int, boot ...int) *gossip.Config {
	port := id + portPrefix
	return &gossip.Config{
		BindPort:                   port,
		BootstrapPeers:             bootPeers(portPrefix, boot...),
		ID:                         fmt.Sprintf("p%d", id),
		MaxBlockCountToStore:       0,
		MaxPropagationBurstLatency: time.Duration(10) * time.Millisecond,
		MaxPropagationBurstSize:    10,
		PropagateIterations:        1,
		PropagatePeerNum:           3,
		PullInterval:               time.Duration(4) * time.Second,
		PullPeerNum:                5,
		InternalEndpoint:           fmt.Sprintf("localhost:%d", port),
		PublishCertPeriod:          10 * time.Second,
		RequestStateInfoInterval:   4 * time.Second,
		PublishStateInfoInterval:   4 * time.Second,
		TimeForMembershipTracker:   5 * time.Second,
	}
}

//创建八卦实例
func newGossipInstance(config *gossip.Config, mcs api.MessageCryptoService) gossip.Gossip {
	id := api.PeerIdentityType(config.InternalEndpoint)
	return gossip.NewGossipServiceWithServer(config, &orgCryptoService{}, mcs,
		id, nil)
}

//创建用于测试的kvledger的新实例
func newCommitter() committer.Committer {
	cb, _ := test.MakeGenesisBlock("testChain")
	ldgr := &ramLedger{
		ledger: make(map[uint64]*ledger.BlockAndPvtData),
	}
	ldgr.CommitWithPvtData(&ledger.BlockAndPvtData{
		Block: cb,
	})
	return committer.NewLedgerCommitter(ldgr)
}

func newPeerNodeWithGossip(config *gossip.Config, committer committer.Committer, acceptor peerIdentityAcceptor, g gossip.Gossip) *peerNode {
	return newPeerNodeWithGossipWithValidator(config, committer, acceptor, g, &validator.MockValidator{})
}

//构建伪对等节点，只模拟流言和状态转移部分
func newPeerNodeWithGossipWithValidator(config *gossip.Config, committer committer.Committer, acceptor peerIdentityAcceptor, g gossip.Gossip, v txvalidator.Validator) *peerNode {
	cs := &cryptoServiceMock{acceptor: acceptor}
//基于所提供配置和通信模块的八卦组件
	if g == nil {
		g = newGossipInstance(config, &cryptoServiceMock{acceptor: noopPeerIdentityAcceptor})
	}

	g.JoinChan(&joinChanMsg{}, common.ChainID(util.GetTestChainID()))

//初始化伪对等模拟器，它只有三个
//基本部件

	servicesAdapater := &ServicesMediator{GossipAdapter: g, MCSAdapter: cs}
	coord := privdata.NewCoordinator(privdata.Support{
		Validator:      v,
		TransientStore: &mockTransientStore{},
		Committer:      committer,
	}, pcomm.SignedData{})
	sp := NewGossipStateProvider(util.GetTestChainID(), servicesAdapater, coord)
	if sp == nil {
		return nil
	}

	return &peerNode{
		port:   config.BindPort,
		g:      g,
		s:      sp.(*GossipStateProviderImpl),
		commit: committer,
		cs:     cs,
	}
}

//构建伪对等节点，只模拟流言和状态转移部分
func newPeerNode(config *gossip.Config, committer committer.Committer, acceptor peerIdentityAcceptor) *peerNode {
	return newPeerNodeWithGossip(config, committer, acceptor, nil)
}

func TestNilDirectMsg(t *testing.T) {
	t.Parallel()
	mc := &mockCommitter{Mock: &mock.Mock{}}
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &mocks.GossipMock{}
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	portPrefix := portStartRange + 50
	p := newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g)
	defer p.shutdown()
	p.s.handleStateRequest(nil)
	p.s.directMessage(nil)
	sMsg, _ := p.s.stateRequestMessage(uint64(10), uint64(8)).NoopSign()
	req := &comm.ReceivedMessageImpl{
		SignedGossipMessage: sMsg,
	}
	p.s.directMessage(req)
}

func TestNilAddPayload(t *testing.T) {
	t.Parallel()
	mc := &mockCommitter{Mock: &mock.Mock{}}
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &mocks.GossipMock{}
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	portPrefix := portStartRange + 100
	p := newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g)
	defer p.shutdown()
	err := p.s.AddPayload(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestAddPayloadLedgerUnavailable(t *testing.T) {
	t.Parallel()
	mc := &mockCommitter{Mock: &mock.Mock{}}
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &mocks.GossipMock{}
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	portPrefix := portStartRange + 150
	p := newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g)
	defer p.shutdown()
//模拟分类帐中的问题
	failedLedger := mock.Mock{}
	failedLedger.On("LedgerHeight", mock.Anything).Return(uint64(0), errors.New("cannot query ledger"))
	mc.Lock()
	mc.Mock = &failedLedger
	mc.Unlock()

	rawblock := pcomm.NewBlock(uint64(1), []byte{})
	b, _ := pb.Marshal(rawblock)
	err := p.s.AddPayload(&proto.Payload{
		SeqNum: uint64(1),
		Data:   b,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed obtaining ledger height")
	assert.Contains(t, err.Error(), "cannot query ledger")
}

func TestLargeBlockGap(t *testing.T) {
//场景：同行知道一个拥有更高分类账高度的同行。
//比自己高500个街区。
//对等机需要以这样的方式请求块：有效负载缓冲区的大小
//永远不要超过某个门槛。
	t.Parallel()
	mc := &mockCommitter{Mock: &mock.Mock{}}
	blocksPassedToLedger := make(chan uint64, 200)
	mc.On("CommitWithPvtData", mock.Anything).Run(func(arg mock.Arguments) {
		blocksPassedToLedger <- arg.Get(0).(*pcomm.Block).Header.Number
	})
	msgsFromPeer := make(chan proto.ReceivedMessage)
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &mocks.GossipMock{}
	membership := []discovery.NetworkMember{
		{
			PKIid:    common.PKIidType("a"),
			Endpoint: "a",
			Properties: &proto.Properties{
				LedgerHeight: 500,
			},
		}}
	g.On("PeersOfChannel", mock.Anything).Return(membership)
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, msgsFromPeer)
	g.On("Send", mock.Anything, mock.Anything).Run(func(arguments mock.Arguments) {
		msg := arguments.Get(0).(*proto.GossipMessage)
//对等机请求状态请求
		req := msg.GetStateRequest()
//构建响应的框架
		res := &proto.GossipMessage{
			Nonce:   msg.Nonce,
			Channel: []byte(util.GetTestChainID()),
			Content: &proto.GossipMessage_StateResponse{
				StateResponse: &proto.RemoteStateResponse{},
			},
		}
//根据对等端的请求，用有效负载填充响应
		for seq := req.StartSeqNum; seq <= req.EndSeqNum; seq++ {
			rawblock := pcomm.NewBlock(seq, []byte{})
			b, _ := pb.Marshal(rawblock)
			payload := &proto.Payload{
				SeqNum: seq,
				Data:   b,
			}
			res.GetStateResponse().Payloads = append(res.GetStateResponse().Payloads, payload)
		}
//最后，将响应发送到对等端期望接收它的通道中
		sMsg, _ := res.NoopSign()
		msgsFromPeer <- &comm.ReceivedMessageImpl{
			SignedGossipMessage: sMsg,
		}
	})
	portPrefix := portStartRange + 200
	p := newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g)
	defer p.shutdown()

//以20毫秒的速度处理每个块。
//对国家做出反应的富有想象力的同龄人
//如果有效负载缓冲区扩展到defmaxblockDistance*2+defantEntropyBatchSize块以上，则测试失败。
blockProcessingTime := 20 * time.Millisecond //总共500个街区10秒
	expectedSequence := 1
	for expectedSequence < 500 {
		blockSeq := <-blocksPassedToLedger
		assert.Equal(t, expectedSequence, int(blockSeq))
//确保有效负载缓冲区没有过度填充
		assert.True(t, p.s.payloads.Size() <= defMaxBlockDistance*2+defAntiEntropyBatchSize, "payload buffer size is %d", p.s.payloads.Size())
		expectedSequence++
		time.Sleep(blockProcessingTime)
	}
}

func TestOverPopulation(t *testing.T) {
//场景：添加到状态提供程序块
//之间有间隙，并确保有效载荷缓冲
//拒绝从分类帐高度到最新高度之间的距离开始的块
//它包含的块大于defmaxblockDistance。
	t.Parallel()
	mc := &mockCommitter{Mock: &mock.Mock{}}
	blocksPassedToLedger := make(chan uint64, 10)
	mc.On("CommitWithPvtData", mock.Anything).Run(func(arg mock.Arguments) {
		blocksPassedToLedger <- arg.Get(0).(*pcomm.Block).Header.Number
	})
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &mocks.GossipMock{}
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	portPrefix := portStartRange + 250
	p := newPeerNode(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor)
	defer p.shutdown()

//按顺序添加一些块并确保其工作
	for i := 1; i <= 4; i++ {
		rawblock := pcomm.NewBlock(uint64(i), []byte{})
		b, _ := pb.Marshal(rawblock)
		assert.NoError(t, p.s.addPayload(&proto.Payload{
			SeqNum: uint64(i),
			Data:   b,
		}, nonBlocking))
	}

//添加10到defmaxblockDistance的有效负载，而我们缺少块[5,9]
//应该成功
	for i := 10; i <= defMaxBlockDistance; i++ {
		rawblock := pcomm.NewBlock(uint64(i), []byte{})
		b, _ := pb.Marshal(rawblock)
		assert.NoError(t, p.s.addPayload(&proto.Payload{
			SeqNum: uint64(i),
			Data:   b,
		}, nonBlocking))
	}

//将有效载荷从defmaxblockDistance+2添加到defmaxblockDistance*10
//应该失败。
	for i := defMaxBlockDistance + 1; i <= defMaxBlockDistance*10; i++ {
		rawblock := pcomm.NewBlock(uint64(i), []byte{})
		b, _ := pb.Marshal(rawblock)
		assert.Error(t, p.s.addPayload(&proto.Payload{
			SeqNum: uint64(i),
			Data:   b,
		}, nonBlocking))
	}

//确保只有块1-4被传递到分类帐
	close(blocksPassedToLedger)
	i := 1
	for seq := range blocksPassedToLedger {
		assert.Equal(t, uint64(i), seq)
		i++
	}
	assert.Equal(t, 5, i)

//确保我们不会在内存中存储太多块
	sp := p.s
	assert.True(t, sp.payloads.Size() < defMaxBlockDistance)
}

func TestBlockingEnqueue(t *testing.T) {
//场景：同时，从八卦和订购者那里获得障碍。
//我们从订购者那里得到的数据块是八卦数据块数量的2倍。
//我们从八卦中得到的障碍是随机指数，以最大限度地减少干扰。
	t.Parallel()
	mc := &mockCommitter{Mock: &mock.Mock{}}
	blocksPassedToLedger := make(chan uint64, 10)
	mc.On("CommitWithPvtData", mock.Anything).Run(func(arg mock.Arguments) {
		blocksPassedToLedger <- arg.Get(0).(*pcomm.Block).Header.Number
	})
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &mocks.GossipMock{}
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	portPrefix := portStartRange + 300
	p := newPeerNode(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor)
	defer p.shutdown()

	numBlocksReceived := 500
	receivedBlockCount := 0
//每1毫秒从订购者那里得到一个数据块
	go func() {
		for i := 1; i <= numBlocksReceived; i++ {
			rawblock := pcomm.NewBlock(uint64(i), []byte{})
			b, _ := pb.Marshal(rawblock)
			block := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			p.s.AddPayload(block)
			time.Sleep(time.Millisecond)
		}
	}()

//每1分钟也要远离流言蜚语
	go func() {
		rand.Seed(time.Now().UnixNano())
		for i := 1; i <= numBlocksReceived/2; i++ {
			blockSeq := rand.Intn(numBlocksReceived)
			rawblock := pcomm.NewBlock(uint64(blockSeq), []byte{})
			b, _ := pb.Marshal(rawblock)
			block := &proto.Payload{
				SeqNum: uint64(blockSeq),
				Data:   b,
			}
			p.s.addPayload(block, nonBlocking)
			time.Sleep(time.Millisecond)
		}
	}()

	for {
		receivedBlock := <-blocksPassedToLedger
		receivedBlockCount++
		m := &mock.Mock{}
		m.On("LedgerHeight", mock.Anything).Return(receivedBlock, nil)
		m.On("CommitWithPvtData", mock.Anything).Run(func(arg mock.Arguments) {
			blocksPassedToLedger <- arg.Get(0).(*pcomm.Block).Header.Number
		})
		mc.Lock()
		mc.Mock = m
		mc.Unlock()
		assert.Equal(t, receivedBlock, uint64(receivedBlockCount))
		if int(receivedBlockCount) == numBlocksReceived {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func TestHaltChainProcessing(t *testing.T) {
	gossipChannel := func(c chan *proto.GossipMessage) <-chan *proto.GossipMessage {
		return c
	}
	makeBlock := func(seq int) []byte {
		b := &pcomm.Block{
			Header: &pcomm.BlockHeader{
				Number: uint64(seq),
			},
			Data: &pcomm.BlockData{
				Data: [][]byte{},
			},
			Metadata: &pcomm.BlockMetadata{
				Metadata: [][]byte{
					{}, {}, {}, {},
				},
			},
		}
		data, _ := pb.Marshal(b)
		return data
	}
	newBlockMsg := func(i int) *proto.GossipMessage {
		return &proto.GossipMessage{
			Channel: []byte("testchainid"),
			Content: &proto.GossipMessage_DataMsg{
				DataMsg: &proto.DataMessage{
					Payload: &proto.Payload{
						SeqNum: uint64(i),
						Data:   makeBlock(i),
					},
				},
			},
		}
	}

	l, recorder := floggingtest.NewTestLogger(t)
	logger = l

	mc := &mockCommitter{Mock: &mock.Mock{}}
	mc.On("CommitWithPvtData", mock.Anything)
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &mocks.GossipMock{}
	gossipMsgs := make(chan *proto.GossipMessage)

	g.On("Accept", mock.Anything, false).Return(gossipChannel(gossipMsgs), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	g.On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{})

	v := &validator.MockValidator{}
	v.On("Validate").Return(&errors2.VSCCExecutionFailureError{
		Err: errors.New("foobar"),
	}).Once()
	portPrefix := portStartRange + 350
	newPeerNodeWithGossipWithValidator(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g, v)
	gossipMsgs <- newBlockMsg(1)
	assertLogged(t, recorder, "Got error while committing")
	assertLogged(t, recorder, "Aborting chain processing")
	assertLogged(t, recorder, "foobar")
}

func TestFailures(t *testing.T) {
	t.Parallel()
	portPrefix := portStartRange + 400
	mc := &mockCommitter{Mock: &mock.Mock{}}
	mc.On("LedgerHeight", mock.Anything).Return(uint64(0), nil)
	g := &mocks.GossipMock{}
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	g.On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{})
	assert.Panics(t, func() {
		newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g)
	})
//重编程模拟
	mc.Mock = &mock.Mock{}
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), errors.New("Failed accessing ledger"))
	assert.Nil(t, newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g))
}

func TestGossipReception(t *testing.T) {
	t.Parallel()
	signalChan := make(chan struct{})
	rawblock := &pcomm.Block{
		Header: &pcomm.BlockHeader{
			Number: uint64(1),
		},
		Data: &pcomm.BlockData{
			Data: [][]byte{},
		},
		Metadata: &pcomm.BlockMetadata{
			Metadata: [][]byte{
				{}, {}, {}, {},
			},
		},
	}
	b, _ := pb.Marshal(rawblock)

	newMsg := func(channel string) *proto.GossipMessage {
		{
			return &proto.GossipMessage{
				Channel: []byte(channel),
				Content: &proto.GossipMessage_DataMsg{
					DataMsg: &proto.DataMessage{
						Payload: &proto.Payload{
							SeqNum: 1,
							Data:   b,
						},
					},
				},
			}
		}
	}

	createChan := func(signalChan chan struct{}) <-chan *proto.GossipMessage {
		c := make(chan *proto.GossipMessage)

		go func(c chan *proto.GossipMessage) {
//等待调用accept（）。
			<-signalChan
//使用无效通道模拟来自八卦组件的消息接收
			c <- newMsg("AAA")
//模拟来自八卦组件的消息接收
			c <- newMsg(util.GetTestChainID())
		}(c)
		return c
	}

	g := &mocks.GossipMock{}
	rmc := createChan(signalChan)
	g.On("Accept", mock.Anything, false).Return(rmc, nil).Run(func(_ mock.Arguments) {
		signalChan <- struct{}{}
	})
	g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
	g.On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{})
	mc := &mockCommitter{Mock: &mock.Mock{}}
	receivedChan := make(chan struct{})
	mc.On("CommitWithPvtData", mock.Anything).Run(func(arguments mock.Arguments) {
		block := arguments.Get(0).(*pcomm.Block)
		assert.Equal(t, uint64(1), block.Header.Number)
		receivedChan <- struct{}{}
	})
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	portPrefix := portStartRange + 450
	p := newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g)
	defer p.shutdown()
	select {
	case <-receivedChan:
	case <-time.After(time.Second * 15):
		assert.Fail(t, "Didn't commit a block within a timely manner")
	}
}

func TestLedgerHeightFromProperties(t *testing.T) {
//场景：对于每个测试，生成一个对等点并提供它
//从对等端对PeersofChannel进行了特定的模拟，
//要么正确设置两个元数据，要么只设置属性，要么不设置，要么两者都设置。
//确保逻辑根据需要处理所有可能的4种情况

	t.Parallel()
//返回是否选择给定的NetworkMember
	wasNetworkMemberSelected := func(t *testing.T, networkMember discovery.NetworkMember, wg *sync.WaitGroup) bool {
		var wasGivenNetworkMemberSelected int32
		finChan := make(chan struct{})
		g := &mocks.GossipMock{}
		g.On("Send", mock.Anything, mock.Anything).Run(func(arguments mock.Arguments) {
			defer wg.Done()
			msg := arguments.Get(0).(*proto.GossipMessage)
			assert.NotNil(t, msg.GetStateRequest())
			peer := arguments.Get(1).([]*comm.RemotePeer)[0]
			if bytes.Equal(networkMember.PKIid, peer.PKIID) {
				atomic.StoreInt32(&wasGivenNetworkMemberSelected, 1)
			}
			finChan <- struct{}{}
		})
		g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
		g.On("Accept", mock.Anything, true).Return(nil, make(chan proto.ReceivedMessage))
		defaultPeer := discovery.NetworkMember{
			InternalEndpoint: "b",
			PKIid:            common.PKIidType("b"),
			Properties: &proto.Properties{
				LedgerHeight: 5,
			},
		}
		g.On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{
			defaultPeer,
			networkMember,
		})
		mc := &mockCommitter{Mock: &mock.Mock{}}
		mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
		portPrefix := portStartRange + 500
		p := newPeerNodeWithGossip(newGossipConfig(portPrefix, 0), mc, noopPeerIdentityAcceptor, g)
		defer p.shutdown()
		select {
		case <-time.After(time.Second * 20):
			t.Fatal("Didn't send a request within a timely manner")
		case <-finChan:
		}
		return atomic.LoadInt32(&wasGivenNetworkMemberSelected) == 1
	}

	peerWithProperties := discovery.NetworkMember{
		PKIid: common.PKIidType("peerWithoutMetadata"),
		Properties: &proto.Properties{
			LedgerHeight: 10,
		},
		InternalEndpoint: "peerWithoutMetadata",
	}

	peerWithoutProperties := discovery.NetworkMember{
		PKIid:            common.PKIidType("peerWithoutProperties"),
		InternalEndpoint: "peerWithoutProperties",
	}

	tests := []struct {
		shouldGivenBeSelected bool
		member                discovery.NetworkMember
	}{
		{member: peerWithProperties, shouldGivenBeSelected: true},
		{member: peerWithoutProperties, shouldGivenBeSelected: false},
	}

	var wg sync.WaitGroup
	wg.Add(len(tests))
	for _, tst := range tests {
		go func(shouldGivenBeSelected bool, member discovery.NetworkMember) {
			assert.Equal(t, shouldGivenBeSelected, wasNetworkMemberSelected(t, member, &wg))
		}(tst.shouldGivenBeSelected, tst.member)
	}
	wg.Wait()
}

func TestAccessControl(t *testing.T) {
	t.Parallel()
	bootstrapSetSize := 5
	bootstrapSet := make([]*peerNode, 0)

	authorizedPeers := map[string]struct{}{
		"localhost:5610": {},
		"localhost:5615": {},
		"localhost:5618": {},
		"localhost:5621": {},
	}
	portPrefix := portStartRange + 600

	blockPullPolicy := func(identity api.PeerIdentityType) error {
		if _, isAuthorized := authorizedPeers[string(identity)]; isAuthorized {
			return nil
		}
		return errors.New("Not authorized")
	}
	for i := 0; i < bootstrapSetSize; i++ {
		commit := newCommitter()
		bootstrapSet = append(bootstrapSet, newPeerNode(newGossipConfig(portPrefix, i), commit, blockPullPolicy))
	}

	defer func() {
		for _, p := range bootstrapSet {
			p.shutdown()
		}
	}()

	msgCount := 5

	for i := 1; i <= msgCount; i++ {
		rawblock := pcomm.NewBlock(uint64(i), []byte{})
		if b, err := pb.Marshal(rawblock); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootstrapSet[0].s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}

	standardPeerSetSize := 10
	peersSet := make([]*peerNode, 0)

	for i := 0; i < standardPeerSetSize; i++ {
		commit := newCommitter()
		peersSet = append(peersSet, newPeerNode(newGossipConfig(portPrefix, bootstrapSetSize+i, 0, 1, 2, 3, 4), commit, blockPullPolicy))
	}

	defer func() {
		for _, p := range peersSet {
			p.shutdown()
		}
	}()

	waitUntilTrueOrTimeout(t, func() bool {
		for _, p := range peersSet {
			if len(p.g.PeersOfChannel(common.ChainID(util.GetTestChainID()))) != bootstrapSetSize+standardPeerSetSize-1 {
				t.Log("Peer discovery has not finished yet")
				return false
			}
		}
		t.Log("All peer discovered each other!!!")
		return true
	}, 30*time.Second)

	t.Log("Waiting for all blocks to arrive.")
	waitUntilTrueOrTimeout(t, func() bool {
		t.Log("Trying to see all authorized peers get all blocks, and all non-authorized didn't")
		for _, p := range peersSet {
			height, err := p.commit.LedgerHeight()
			id := fmt.Sprintf("localhost:%d", p.port)
			if _, isAuthorized := authorizedPeers[id]; isAuthorized {
				if height != uint64(msgCount+1) || err != nil {
					return false
				}
			} else {
				if err == nil && height > 1 {
					assert.Fail(t, "Peer", id, "got message but isn't authorized! Height:", height)
				}
			}
		}
		t.Log("All peers have same ledger height!!!")
		return true
	}, 60*time.Second)
}

func TestNewGossipStateProvider_SendingManyMessages(t *testing.T) {
	t.Parallel()
	bootstrapSetSize := 5
	bootstrapSet := make([]*peerNode, 0)
	portPrefix := portStartRange + 650

	for i := 0; i < bootstrapSetSize; i++ {
		commit := newCommitter()
		bootstrapSet = append(bootstrapSet, newPeerNode(newGossipConfig(portPrefix, i, 0, 1, 2, 3, 4), commit, noopPeerIdentityAcceptor))
	}

	defer func() {
		for _, p := range bootstrapSet {
			p.shutdown()
		}
	}()

	msgCount := 10

	for i := 1; i <= msgCount; i++ {
		rawblock := pcomm.NewBlock(uint64(i), []byte{})
		if b, err := pb.Marshal(rawblock); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootstrapSet[0].s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}

	standartPeersSize := 10
	peersSet := make([]*peerNode, 0)

	for i := 0; i < standartPeersSize; i++ {
		commit := newCommitter()
		peersSet = append(peersSet, newPeerNode(newGossipConfig(portPrefix, bootstrapSetSize+i, 0, 1, 2, 3, 4), commit, noopPeerIdentityAcceptor))
	}

	defer func() {
		for _, p := range peersSet {
			p.shutdown()
		}
	}()

	waitUntilTrueOrTimeout(t, func() bool {
		for _, p := range peersSet {
			if len(p.g.PeersOfChannel(common.ChainID(util.GetTestChainID()))) != bootstrapSetSize+standartPeersSize-1 {
				t.Log("Peer discovery has not finished yet")
				return false
			}
		}
		t.Log("All peer discovered each other!!!")
		return true
	}, 30*time.Second)

	t.Log("Waiting for all blocks to arrive.")
	waitUntilTrueOrTimeout(t, func() bool {
		t.Log("Trying to see all peers get all blocks")
		for _, p := range peersSet {
			height, err := p.commit.LedgerHeight()
			if height != uint64(msgCount+1) || err != nil {
				return false
			}
		}
		t.Log("All peers have same ledger height!!!")
		return true
	}, 60*time.Second)
}

func TestGossipStateProvider_TestStateMessages(t *testing.T) {
	t.Parallel()
	portPrefix := portStartRange + 700
	bootPeer := newPeerNode(newGossipConfig(portPrefix, 0), newCommitter(), noopPeerIdentityAcceptor)
	defer bootPeer.shutdown()

	peer := newPeerNode(newGossipConfig(portPrefix, 1, 0), newCommitter(), noopPeerIdentityAcceptor)
	defer peer.shutdown()

	naiveStateMsgPredicate := func(message interface{}) bool {
		return message.(proto.ReceivedMessage).GetGossipMessage().IsRemoteStateMessage()
	}

	_, bootCh := bootPeer.g.Accept(naiveStateMsgPredicate, true)
	_, peerCh := peer.g.Accept(naiveStateMsgPredicate, true)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		msg := <-bootCh
		t.Log("Bootstrap node got message, ", msg)
		assert.True(t, msg.GetGossipMessage().GetStateRequest() != nil)
		msg.Respond(&proto.GossipMessage{
			Content: &proto.GossipMessage_StateResponse{StateResponse: &proto.RemoteStateResponse{Payloads: nil}},
		})
		wg.Done()
	}()

	go func() {
		msg := <-peerCh
		t.Log("Peer node got an answer, ", msg)
		assert.True(t, msg.GetGossipMessage().GetStateResponse() != nil)
		wg.Done()
	}()

	readyCh := make(chan struct{})
	go func() {
		wg.Wait()
		readyCh <- struct{}{}
	}()

	chainID := common.ChainID(util.GetTestChainID())
	waitUntilTrueOrTimeout(t, func() bool {
		return len(peer.g.PeersOfChannel(chainID)) == 1
	}, 30*time.Second)

	t.Log("Sending gossip message with remote state request")
	peer.g.Send(&proto.GossipMessage{
		Content: &proto.GossipMessage_StateRequest{StateRequest: &proto.RemoteStateRequest{StartSeqNum: 0, EndSeqNum: 1}},
	}, &comm.RemotePeer{Endpoint: peer.g.PeersOfChannel(chainID)[0].Endpoint, PKIID: peer.g.PeersOfChannel(chainID)[0].PKIid})
	t.Log("Waiting until peers exchange messages")

	select {
	case <-readyCh:
		{
			t.Log("Done!!!")

		}
	case <-time.After(time.Duration(10) * time.Second):
		{
			t.Fail()
		}
	}
}

//启动一个bootstrap对等机，并将defantentropyBatchSize+5消息提交到
//本地分类账，下一个生成新的对等等待反熵过程
//完成丢失的块。由于状态传输消息现在已成批处理，因此应该
//查看两条带有状态转移响应的消息。
func TestNewGossipStateProvider_BatchingOfStateRequest(t *testing.T) {
	t.Parallel()
	portPrefix := portStartRange + 750
	bootPeer := newPeerNode(newGossipConfig(portPrefix, 0), newCommitter(), noopPeerIdentityAcceptor)
	defer bootPeer.shutdown()

	msgCount := defAntiEntropyBatchSize + 5
	expectedMessagesCnt := 2

	for i := 1; i <= msgCount; i++ {
		rawblock := pcomm.NewBlock(uint64(i), []byte{})
		if b, err := pb.Marshal(rawblock); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootPeer.s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}

	peer := newPeerNode(newGossipConfig(portPrefix, 1, 0), newCommitter(), noopPeerIdentityAcceptor)
	defer peer.shutdown()

	naiveStateMsgPredicate := func(message interface{}) bool {
		return message.(proto.ReceivedMessage).GetGossipMessage().IsRemoteStateMessage()
	}
	_, peerCh := peer.g.Accept(naiveStateMsgPredicate, true)

	messageCh := make(chan struct{})
	stopWaiting := make(chan struct{})

//因此，提交的消息数为defantentropyBatchSize+5。
//预期的批数为expectedMessagesCnt=2。执行例行程序
//确保它接收到预期数量的消息并发送成功信号
//继续测试
	go func(expected int) {
		cnt := 0
		for cnt < expected {
			select {
			case <-peerCh:
				{
					cnt++
				}

			case <-stopWaiting:
				{
					return
				}
			}
		}

		messageCh <- struct{}{}
	}(expectedMessagesCnt)

//等待消息，该消息指示接收到的预期消息批数
//否则，在2*defantentropyInterval+1秒后超时
	select {
	case <-messageCh:
		{
//一旦我们收到一条消息，表明收到了两批产品，
//确保消息确实已提交。
			waitUntilTrueOrTimeout(t, func() bool {
				if len(peer.g.PeersOfChannel(common.ChainID(util.GetTestChainID()))) != 1 {
					t.Log("Peer discovery has not finished yet")
					return false
				}
				t.Log("All peer discovered each other!!!")
				return true
			}, 30*time.Second)

			t.Log("Waiting for all blocks to arrive.")
			waitUntilTrueOrTimeout(t, func() bool {
				t.Log("Trying to see all peers get all blocks")
				height, err := peer.commit.LedgerHeight()
				if height != uint64(msgCount+1) || err != nil {
					return false
				}
				t.Log("All peers have same ledger height!!!")
				return true
			}, 60*time.Second)
		}
	case <-time.After(defAntiEntropyInterval*2 + time.Second*1):
		{
			close(stopWaiting)
			t.Fatal("Expected to receive two batches with missing payloads")
		}
	}
}

//用于捕获模拟接口的CoordinatorLock模拟结构
//coord在测试期间模拟coord流
type coordinatorMock struct {
	committer.Committer
	mock.Mock
}

func (mock *coordinatorMock) GetPvtDataAndBlockByNum(seqNum uint64, _ pcomm.SignedData) (*pcomm.Block, gutil.PvtDataCollections, error) {
	args := mock.Called(seqNum)
	return args.Get(0).(*pcomm.Block), args.Get(1).(gutil.PvtDataCollections), args.Error(2)
}

func (mock *coordinatorMock) GetBlockByNum(seqNum uint64) (*pcomm.Block, error) {
	args := mock.Called(seqNum)
	return args.Get(0).(*pcomm.Block), args.Error(1)
}

func (mock *coordinatorMock) StoreBlock(block *pcomm.Block, data gutil.PvtDataCollections) error {
	args := mock.Called(block, data)
	return args.Error(1)
}

func (mock *coordinatorMock) LedgerHeight() (uint64, error) {
	args := mock.Called()
	return args.Get(0).(uint64), args.Error(1)
}

func (mock *coordinatorMock) Close() {
	mock.Called()
}

//storepvdtdata用于将私有日期持久化到临时存储中
func (mock *coordinatorMock) StorePvtData(txid string, privData *transientstore2.TxPvtReadWriteSetWithConfigInfo, blkHeight uint64) error {
	return mock.Called().Error(0)
}

type receivedMessageMock struct {
	mock.Mock
}

//ACK向发送方返回消息确认
func (mock *receivedMessageMock) Ack(err error) {

}

func (mock *receivedMessageMock) Respond(msg *proto.GossipMessage) {
	mock.Called(msg)
}

func (mock *receivedMessageMock) GetGossipMessage() *proto.SignedGossipMessage {
	args := mock.Called()
	return args.Get(0).(*proto.SignedGossipMessage)
}

func (mock *receivedMessageMock) GetSourceEnvelope() *proto.Envelope {
	args := mock.Called()
	return args.Get(0).(*proto.Envelope)
}

func (mock *receivedMessageMock) GetConnectionInfo() *proto.ConnectionInfo {
	args := mock.Called()
	return args.Get(0).(*proto.ConnectionInfo)
}

type testData struct {
	block   *pcomm.Block
	pvtData gutil.PvtDataCollections
}

func TestTransferOfPrivateRWSet(t *testing.T) {
	t.Parallel()
	chainID := "testChainID"

//第一个八卦实例
	g := &mocks.GossipMock{}
	coord1 := new(coordinatorMock)

	gossipChannel := make(chan *proto.GossipMessage)
	commChannel := make(chan proto.ReceivedMessage)

	gossipChannelFactory := func(ch chan *proto.GossipMessage) <-chan *proto.GossipMessage {
		return ch
	}

	g.On("Accept", mock.Anything, false).Return(gossipChannelFactory(gossipChannel), nil)
	g.On("Accept", mock.Anything, true).Return(nil, commChannel)

	g.On("UpdateChannelMetadata", mock.Anything, mock.Anything)
	g.On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{})
	g.On("Close")

	coord1.On("LedgerHeight", mock.Anything).Return(uint64(5), nil)

	var data = map[uint64]*testData{
		uint64(2): {
			block: &pcomm.Block{
				Header: &pcomm.BlockHeader{
					Number:       2,
					DataHash:     []byte{0, 1, 1, 1},
					PreviousHash: []byte{0, 0, 0, 1},
				},
				Data: &pcomm.BlockData{
					Data: [][]byte{{1}, {2}, {3}},
				},
			},
			pvtData: gutil.PvtDataCollections{
				{
					SeqInBlock: uint64(0),
					WriteSet: &rwset.TxPvtReadWriteSet{
						DataModel: rwset.TxReadWriteSet_KV,
						NsPvtRwset: []*rwset.NsPvtReadWriteSet{
							{
								Namespace: "myCC:v1",
								CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
									{
										CollectionName: "mysecrectCollection",
										Rwset:          []byte{1, 2, 3, 4, 5},
									},
								},
							},
						},
					},
				},
			},
		},

		uint64(3): {
			block: &pcomm.Block{
				Header: &pcomm.BlockHeader{
					Number:       3,
					DataHash:     []byte{1, 1, 1, 1},
					PreviousHash: []byte{0, 1, 1, 1},
				},
				Data: &pcomm.BlockData{
					Data: [][]byte{{4}, {5}, {6}},
				},
			},
			pvtData: gutil.PvtDataCollections{
				{
					SeqInBlock: uint64(2),
					WriteSet: &rwset.TxPvtReadWriteSet{
						DataModel: rwset.TxReadWriteSet_KV,
						NsPvtRwset: []*rwset.NsPvtReadWriteSet{
							{
								Namespace: "otherCC:v1",
								CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
									{
										CollectionName: "topClassified",
										Rwset:          []byte{0, 0, 0, 4, 2},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for seqNum, each := range data {
  /*rd1.on（“getpvtdata和blockbynum”，seqnum）.return（each.block，each.pvtdata，nil/*无错误*/）
 }

 协调1.开启（“关闭”）

 servicesadapter：=&servicesmeditor gossipadapter:g，mcsadapter：&cryptoservicesock acceptor:nooppeerIdentityAcceptor
 st：=newgossipstateprovider（chainID，servicesadapather，coord1）
 推迟ST停止（）

 //模拟状态请求消息
 requestMsg：=新建（receivedMessageLock）

 //获取状态请求消息，块[2…3]
 requestgossipmsg：=&proto.gossipmessage_
  //从请求中复制nonce字段，以便能够匹配响应
  随机数：1，
  标签：proto.gossipmessage_chan_or_org，
  通道：[]字节（chainID），
  内容：&proto.gossipmessage staterequest staterequest：&proto.remotestaterequest
   StaseQuNUM：2，
   EndSeqNum：3，
  }，
 }

 msg，：=requestgossipmsg.noopsign（）。

 requestmsg.on（“getgossipmessage”）.return（msg）
 requestmsg.on（“getconnectioninfo”）.return（&proto.connectioninfo_
  授权：&proto.authinfo，
 }）

 //返回响应的通道
 responseChannel:=生成（chan proto.receivedMessage）
 延迟关闭（responseChannel）

 requestmsg.on（“response”，mock.anything）.run（func（args mock.arguments）
  //获取八卦响应以响应状态请求
  响应：=args.get（0）。（*proto.gossipmessage）
  //将其包装成接收到的响应
  receivedmsg：=新（receivedmessagelock）
  //创建符号响应
  msg，：=response.noopsign（）。
  //模拟响应
  receivedmsg.on（“getgossipmessage”）.return（msg）
  //发送响应
  响应通道<-receivedmsg
 }）

 //通过通信通道向状态传输发送请求消息
 通信信道<-requestmsg

 //状态转移请求应导致状态响应返回
 响应：=<-responseChannel

 //启动断言部分
 StateResponse:=Response.GetGossipMessage（）.GetStateResponse（））

 断言：=assert.new（t）
 //nonce应等于请求的nonce
 assertion.equal（response.getGossipMessage（）.nonce，uint64（1））。
 //有效负载不需要为零
 断言.notnil（StateResponse）
 断言.notnil（stateresponse.payloads）
 //只需要两条消息
 断言.equal（len（stateresponse.payloads），2）

 //断言我们拥有所有数据，并且与预期的相同
 对于u，每个：=range stateresponse.payloads
  块：=&pcomm.block
  错误：=pb.unmashal（each.data，block）
  断言。无错误（错误）

  断言.notnil（block.header）

  测试块，确定：=data[block.header.number]
  断言。正确（确定）

  对于i，d：=range testblock.block.data.data_
   断言.true（bytes.equal（d，block.data.data[i]））
  }

  对于i，p：=range testblock.pvtdata_
   pvtDataPayload：=&proto.pvtDataPayload
   错误：=pb.unmashal（每个.privatedata[i]，pvtDataPayload）
   断言。无错误（错误）
   pvtrwset：=&rwset.txpvtreadwriteset
   错误=pb.unmashal（pvtDataPayload.Payload，pvtrwset）
   断言。无错误（错误）
   断言.true（pb.equal（p.writeset，pvtrwset））
  }
 }
}

类型testpeer结构
 *嘲弄.gossippock
 ID字符串
 八卦频道chan*proto.gossipmessage
 通信信道信道信道协议接收消息
 坐标*坐标锁
}

func（t testpeer）八卦（）<-chan*proto.gossipmessage_
 返回t.gossipsipchannel
}

func（t testpeer）comm（）chan proto.receivedMessage_
 返回T.commchannel
}

var peers=map[string]testpeer_
 “PEE1”：{
  id:“peer1”，
  八卦频道：制作（chan*proto.gossipmessage）
  通信信道：make（chan proto.receivedmessage）
  八卦：&mocks.gossipmock，
  COORD：新（COORDINAtorMOCK）
 }
 “Pee2”：{
  id:“peer2”，
  八卦频道：制作（chan*proto.gossipmessage）
  通信信道：make（chan proto.receivedmessage）
  八卦：&mocks.gossipmock，
  COORD：新（COORDINAtorMOCK）
 }
}

pvdt数据在每个人之间的func测试传输（t*testing.t）
 /*
    这个测试涵盖了非常基本的场景，有两个对等方：“peer1”和“peer2”，
    而peer2在分类帐中缺少几个块，因此要求复制这些块
    与第一个对等点的块。

    测试将检查来自一个对等机的块是否将复制到第二个对等机，以及
    内容相同。
 **/

	t.Parallel()
	chainID := "testChainID"

//初始化对等体
	for _, peer := range peers {
		peer.On("Accept", mock.Anything, false).Return(peer.Gossip(), nil)

		peer.On("Accept", mock.Anything, true).
			Return(nil, peer.Comm()).
			Once().
			On("Accept", mock.Anything, true).
			Return(nil, make(chan proto.ReceivedMessage))

		peer.On("UpdateChannelMetadata", mock.Anything, mock.Anything)
		peer.coord.On("Close")
		peer.On("Close")
	}

//第一个同行将拥有更高级的分类帐
	peers["peer1"].coord.On("LedgerHeight", mock.Anything).Return(uint64(3), nil)

//第二个对等机的间隔为一个块，因此必须从上一个块复制它。
	peers["peer2"].coord.On("LedgerHeight", mock.Anything).Return(uint64(2), nil)

	peers["peer1"].coord.On("GetPvtDataAndBlockByNum", uint64(2)).Return(&pcomm.Block{
		Header: &pcomm.BlockHeader{
			Number:       2,
			DataHash:     []byte{0, 0, 0, 1},
			PreviousHash: []byte{0, 1, 1, 1},
		},
		Data: &pcomm.BlockData{
			Data: [][]byte{{4}, {5}, {6}},
		},
	}, gutil.PvtDataCollections{&ledger.TxPvtData{
		SeqInBlock: uint64(1),
		WriteSet: &rwset.TxPvtReadWriteSet{
			DataModel: rwset.TxReadWriteSet_KV,
			NsPvtRwset: []*rwset.NsPvtReadWriteSet{
				{
					Namespace: "myCC:v1",
					CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
						{
							CollectionName: "mysecrectCollection",
							Rwset:          []byte{1, 2, 3, 4, 5},
						},
					},
				},
			},
		},
	}}, nil)

//返回对等成员身份
	member2 := discovery.NetworkMember{
		PKIid:            common.PKIidType([]byte{2}),
		Endpoint:         "peer2:7051",
		InternalEndpoint: "peer2:7051",
		Properties: &proto.Properties{
			LedgerHeight: 2,
		},
	}

	member1 := discovery.NetworkMember{
		PKIid:            common.PKIidType([]byte{1}),
		Endpoint:         "peer1:7051",
		InternalEndpoint: "peer1:7051",
		Properties: &proto.Properties{
			LedgerHeight: 3,
		},
	}

	peers["peer1"].On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{member2})
	peers["peer2"].On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{member1})

	peers["peer2"].On("Send", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		request := args.Get(0).(*proto.GossipMessage)
		requestMsg := new(receivedMessageMock)
		msg, _ := request.NoopSign()
		requestMsg.On("GetGossipMessage").Return(msg)
		requestMsg.On("GetConnectionInfo").Return(&proto.ConnectionInfo{
			Auth: &proto.AuthInfo{},
		})

		requestMsg.On("Respond", mock.Anything).Run(func(args mock.Arguments) {
			response := args.Get(0).(*proto.GossipMessage)
			receivedMsg := new(receivedMessageMock)
			msg, _ := response.NoopSign()
			receivedMsg.On("GetGossipMessage").Return(msg)
//将响应发送回对等端
			peers["peer2"].commChannel <- receivedMsg
		})

		peers["peer1"].commChannel <- requestMsg
	})

	wg := sync.WaitGroup{}
	wg.Add(1)
	peers["peer2"].coord.On("StoreBlock", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
wg.Done() //完成一次第二个对等点点击块的提交
}).Return([]string{}, nil) //没有要完成的pvt数据，没有错误

	cryptoService := &cryptoServiceMock{acceptor: noopPeerIdentityAcceptor}

	mediator := &ServicesMediator{GossipAdapter: peers["peer1"], MCSAdapter: cryptoService}
	peer1State := NewGossipStateProvider(chainID, mediator, peers["peer1"].coord)
	defer peer1State.Stop()

	mediator = &ServicesMediator{GossipAdapter: peers["peer2"], MCSAdapter: cryptoService}
	peer2State := NewGossipStateProvider(chainID, mediator, peers["peer2"].coord)
	defer peer2State.Stop()

//确保状态已复制
	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		break
	case <-time.After(30 * time.Second):
		t.Fail()
	}
}

func waitUntilTrueOrTimeout(t *testing.T, predicate func() bool, timeout time.Duration) {
	ch := make(chan struct{})
	go func() {
		t.Log("Started to spin off, until predicate will be satisfied.")
		for !predicate() {
			time.Sleep(1 * time.Second)
		}
		ch <- struct{}{}
		t.Log("Done.")
	}()

	select {
	case <-ch:
		break
	case <-time.After(timeout):
		t.Fatal("Timeout has expired")
		break
	}
	t.Log("Stop waiting until timeout or true")
}

func assertLogged(t *testing.T, r *floggingtest.Recorder, msg string) {
	observed := func() bool { return len(r.MessagesContaining(msg)) > 0 }
	waitUntilTrueOrTimeout(t, observed, 30*time.Second)
}
