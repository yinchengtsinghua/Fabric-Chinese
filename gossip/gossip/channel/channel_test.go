
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


package channel

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gproto "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip/algo"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type msgMutator func(message *proto.Envelope)

var conf = Config{
	ID:                          "test",
	PublishStateInfoInterval:    time.Millisecond * 100,
	MaxBlockCountToStore:        100,
	PullPeerNum:                 3,
	PullInterval:                time.Second,
	RequestStateInfoInterval:    time.Millisecond * 100,
	BlockExpirationInterval:     time.Second * 6,
	StateInfoCacheSweepInterval: time.Second,
	TimeForMembershipTracker:    time.Second * 5,
}

func init() {
	util.SetupTestLogging()
	shortenedWaitTime := time.Millisecond * 300
	algo.SetDigestWaitTime(shortenedWaitTime / 2)
	algo.SetRequestWaitTime(shortenedWaitTime)
	algo.SetResponseWaitTime(shortenedWaitTime)
	factory.InitFactories(nil)
}

var (
//组织：组织1，组织2
//A通道：ORG1
	channelA                  = common.ChainID("A")
	orgInChannelA             = api.OrgIdentityType("ORG1")
	orgNotInChannelA          = api.OrgIdentityType("ORG2")
	pkiIDInOrg1               = common.PKIidType("pkiIDInOrg1")
	pkiIDnilOrg               = common.PKIidType("pkIDnilOrg")
	pkiIDInOrg1ButNotEligible = common.PKIidType("pkiIDInOrg1ButNotEligible")
	pkiIDinOrg2               = common.PKIidType("pkiIDinOrg2")
)

type joinChanMsg struct {
	getTS               func() time.Time
	members2AnchorPeers map[string][]api.AnchorPeer
}

//SequenceNumber返回块的序列号
//此joinchanmsg是从派生的。
//我在这里使用时间戳只是为了测试。
func (jcm *joinChanMsg) SequenceNumber() uint64 {
	if jcm.getTS != nil {
		return uint64(jcm.getTS().UnixNano())
	}
	return uint64(time.Now().UnixNano())
}

//成员返回频道的组织
func (jcm *joinChanMsg) Members() []api.OrgIdentityType {
	if jcm.members2AnchorPeers == nil {
		return []api.OrgIdentityType{orgInChannelA}
	}
	members := make([]api.OrgIdentityType, len(jcm.members2AnchorPeers))
	i := 0
	for org := range jcm.members2AnchorPeers {
		members[i] = api.OrgIdentityType(org)
		i++
	}
	return members
}

//anchor peers of返回给定组织的锚定对等方
func (jcm *joinChanMsg) AnchorPeersOf(org api.OrgIdentityType) []api.AnchorPeer {
	if jcm.members2AnchorPeers == nil {
		return []api.AnchorPeer{}
	}
	return jcm.members2AnchorPeers[string(org)]
}

type cryptoService struct {
	mocked bool
	mock.Mock
}

func (cs *cryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	return time.Now().Add(time.Hour), nil
}

func (cs *cryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	panic("Should not be called in this test")
}

func (cs *cryptoService) VerifyByChannel(channel common.ChainID, identity api.PeerIdentityType, _, _ []byte) error {
	if !cs.mocked {
		return nil
	}
	args := cs.Called(identity)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(error)
}

func (cs *cryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	args := cs.Called(signedBlock)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(error)
}

func (cs *cryptoService) Sign(msg []byte) ([]byte, error) {
	panic("Should not be called in this test")
}

func (cs *cryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	panic("Should not be called in this test")
}

func (cs *cryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	panic("Should not be called in this test")
}

type receivedMsg struct {
	PKIID common.PKIidType
	msg   *proto.SignedGossipMessage
	mock.Mock
}

//GetSourceEnvelope返回接收到的消息所在的信封
//建筑用
func (m *receivedMsg) GetSourceEnvelope() *proto.Envelope {
	return m.msg.Envelope
}

func (m *receivedMsg) GetGossipMessage() *proto.SignedGossipMessage {
	return m.msg
}

func (m *receivedMsg) Respond(msg *proto.GossipMessage) {
	m.Called(msg)
}

//ACK向发送方返回消息确认
func (m *receivedMsg) Ack(err error) {

}

func (m *receivedMsg) GetConnectionInfo() *proto.ConnectionInfo {
	return &proto.ConnectionInfo{
		ID: m.PKIID,
	}
}

type gossipAdapterMock struct {
	mock.Mock
}

func (ga *gossipAdapterMock) Sign(msg *proto.GossipMessage) (*proto.SignedGossipMessage, error) {
	return msg.NoopSign()
}

func (ga *gossipAdapterMock) GetConf() Config {
	args := ga.Called()
	return args.Get(0).(Config)
}

func (ga *gossipAdapterMock) Gossip(msg *proto.SignedGossipMessage) {
	ga.Called(msg)
}

func (ga *gossipAdapterMock) Forward(msg proto.ReceivedMessage) {
	ga.Called(msg)
}

func (ga *gossipAdapterMock) DeMultiplex(msg interface{}) {
	ga.Called(msg)
}

func (ga *gossipAdapterMock) GetMembership() []discovery.NetworkMember {
	args := ga.Called()
	members := args.Get(0).([]discovery.NetworkMember)

	return members
}

//查找返回网络成员，如果未找到，则返回nil
func (ga *gossipAdapterMock) Lookup(PKIID common.PKIidType) *discovery.NetworkMember {
//确保之前已配置查找
	if !ga.wasMocked("Lookup") {
		return &discovery.NetworkMember{}
	}
	args := ga.Called(PKIID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*discovery.NetworkMember)
}

func (ga *gossipAdapterMock) Send(msg *proto.SignedGossipMessage, peers ...*comm.RemotePeer) {
//确保我们已配置发送优先权
	if !ga.wasMocked("Send") {
		return
	}
	ga.Called(msg, peers)
}

func (ga *gossipAdapterMock) ValidateStateInfoMessage(msg *proto.SignedGossipMessage) error {
	args := ga.Called(msg)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(error)
}

func (ga *gossipAdapterMock) GetOrgOfPeer(PKIIID common.PKIidType) api.OrgIdentityType {
	args := ga.Called(PKIIID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(api.OrgIdentityType)
}

func (ga *gossipAdapterMock) GetIdentityByPKIID(pkiID common.PKIidType) api.PeerIdentityType {
	if ga.wasMocked("GetIdentityByPKIID") {
		return ga.Called(pkiID).Get(0).(api.PeerIdentityType)
	}
	return api.PeerIdentityType(pkiID)
}

func (ga *gossipAdapterMock) wasMocked(methodName string) bool {
//以下调用只是为了同步预期的调用
//通过测试goroutine的“on”调用访问
	ga.On("bla", mock.Anything)
	for _, ec := range ga.ExpectedCalls {
		if ec.Method == methodName {
			return true
		}
	}
	return false
}

func configureAdapter(adapter *gossipAdapterMock, members ...discovery.NetworkMember) {
	adapter.On("GetConf").Return(conf)
	adapter.On("GetMembership").Return(members)
	adapter.On("GetOrgOfPeer", pkiIDInOrg1).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDInOrg1ButNotEligible).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDinOrg2).Return(orgNotInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDnilOrg).Return(nil)
	adapter.On("GetOrgOfPeer", mock.Anything).Return(api.OrgIdentityType(nil))
}

func TestBadInput(t *testing.T) {
	cs := &cryptoService{}
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{}).(*gossipChannel)
	assert.False(t, gc.verifyMsg(nil))
	assert.False(t, gc.verifyMsg(&receivedMsg{msg: nil, PKIID: nil}))
}

func TestSelf(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	pkiID1 := common.PKIidType("1")
	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(orgInChannelA): {},
		},
	}
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	adapter.On("Gossip", mock.Anything)
	gc := NewGossipChannel(pkiID1, orgInChannelA, cs, channelA, adapter, jcm)
	gc.UpdateLedgerHeight(1)
	gMsg := gc.Self().GossipMessage
	env := gc.Self().Envelope
	sMsg, _ := env.ToGossipMessage()
	assert.True(t, gproto.Equal(gMsg, sMsg.GossipMessage))
	assert.Equal(t, gMsg.GetStateInfo().Properties.LedgerHeight, uint64(1))
	assert.Equal(t, gMsg.GetStateInfo().PkiId, []byte("1"))
}

func TestMsgStoreNotExpire(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}

	pkiID1 := common.PKIidType("1")
	pkiID2 := common.PKIidType("2")
	pkiID3 := common.PKIidType("3")

	peer1 := discovery.NetworkMember{PKIid: pkiID2, InternalEndpoint: "1", Endpoint: "1"}
	peer2 := discovery.NetworkMember{PKIid: pkiID2, InternalEndpoint: "2", Endpoint: "2"}
	peer3 := discovery.NetworkMember{PKIid: pkiID3, InternalEndpoint: "3", Endpoint: "3"}

	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(orgInChannelA): {},
		},
	}

	adapter := new(gossipAdapterMock)
	adapter.On("GetOrgOfPeer", pkiID1).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiID2).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiID3).Return(orgInChannelA)

	adapter.On("ValidateStateInfoMessage", mock.Anything).Return(nil)
	adapter.On("GetMembership").Return([]discovery.NetworkMember{peer2, peer3})
	adapter.On("DeMultiplex", mock.Anything)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("GetConf").Return(conf)

	gc := NewGossipChannel(pkiID1, orgInChannelA, cs, channelA, adapter, jcm)
	gc.UpdateLedgerHeight(1)
//接收来自其他对等方的StateInfo消息
	gc.HandleMessage(&receivedMsg{PKIID: pkiID2, msg: createStateInfoMsg(1, pkiID2, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiID3, msg: createStateInfoMsg(1, pkiID3, channelA)})

	simulateStateInfoRequest := func(pkiID []byte, outChan chan *proto.SignedGossipMessage) {
		sentMessages := make(chan *proto.GossipMessage, 1)
//确保我们使用有效的MAC响应StateInfoSnapshot请求
		s, _ := (&proto.GossipMessage{
			Tag: proto.GossipMessage_CHAN_OR_ORG,
			Content: &proto.GossipMessage_StateInfoPullReq{
				StateInfoPullReq: &proto.StateInfoPullRequest{
					Channel_MAC: GenerateMAC(pkiID, channelA),
				},
			},
		}).NoopSign()
		snapshotReq := &receivedMsg{
			PKIID: pkiID,
			msg:   s,
		}
		snapshotReq.On("Respond", mock.Anything).Run(func(args mock.Arguments) {
			sentMessages <- args.Get(0).(*proto.GossipMessage)
		})

		go gc.HandleMessage(snapshotReq)
		select {
		case <-time.After(time.Second):
			t.Fatal("Haven't received a state info snapshot on time")
		case msg := <-sentMessages:
			for _, el := range msg.GetStateSnapshot().Elements {
				sMsg, err := el.ToGossipMessage()
				assert.NoError(t, err)
				outChan <- sMsg
			}
		}
	}

	c := make(chan *proto.SignedGossipMessage, 3)
	simulateStateInfoRequest(pkiID2, c)
	assert.Len(t, c, 3)

	c = make(chan *proto.SignedGossipMessage, 3)
	simulateStateInfoRequest(pkiID3, c)
	assert.Len(t, c, 3)

//现在在成员视图中模拟对等端3的过期
	adapter.On("Lookup", pkiID1).Return(&peer1)
	adapter.On("Lookup", pkiID2).Return(&peer2)
	adapter.On("Lookup", pkiID3).Return(nil)
//确保在继续之前至少扫描了一次
//测试
	time.Sleep(conf.StateInfoCacheSweepInterval * 2)

	c = make(chan *proto.SignedGossipMessage, 3)
	simulateStateInfoRequest(pkiID2, c)
	assert.Len(t, c, 2)

	c = make(chan *proto.SignedGossipMessage, 3)
	simulateStateInfoRequest(pkiID3, c)
	assert.Len(t, c, 2)
}

func TestLeaveChannel(t *testing.T) {
//场景：让我们的同伴接收StateInfo消息
//从离开通道的对等机，确保它跳过通道
//返回成员身份时。
//接下来，让我们自己的同伴离开频道并确保：
//1）查询时不返回任何频道成员
//2）不再发送拉块
//3）当要求拉块时，忽略请求。
	t.Parallel()

	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			"ORG1": {},
			"ORG2": {},
		},
	}

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	members := []discovery.NetworkMember{
		{PKIid: pkiIDInOrg1},
		{PKIid: pkiIDinOrg2},
	}
	var helloPullWG sync.WaitGroup
	helloPullWG.Add(1)
	configureAdapter(adapter, members...)
	gc := NewGossipChannel(common.PKIidType("p0"), orgInChannelA, cs, channelA, adapter, jcm)
	adapter.On("Send", mock.Anything, mock.Anything).Run(func(arguments mock.Arguments) {
		msg := arguments.Get(0).(*proto.SignedGossipMessage)
		if msg.IsPullMsg() {
			helloPullWG.Done()
			assert.False(t, gc.(*gossipChannel).hasLeftChannel())
		}
	})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDinOrg2, msg: createStateInfoMsg(1, pkiIDinOrg2, channelA)})
//让某个对等发送一个块给我们，这样当向我们发送hello时，我们可以向某个对等发送摘要
	gc.HandleMessage(&receivedMsg{msg: createDataMsg(2, channelA), PKIID: pkiIDInOrg1})
	assert.Len(t, gc.GetPeers(), 2)
//现在，在org2中让对等机“离开频道”通过发布是一个更新
	stateInfoMsg := &receivedMsg{PKIID: pkiIDinOrg2, msg: createStateInfoMsg(0, pkiIDinOrg2, channelA)}
	stateInfoMsg.GetGossipMessage().GetStateInfo().Properties.LeftChannel = true
	gc.HandleMessage(stateInfoMsg)
	assert.Len(t, gc.GetPeers(), 1)
//确保org1中的对等方保持不变，跳过org2中的对等方
	assert.Equal(t, pkiIDInOrg1, gc.GetPeers()[0].PKIid)
	var digestSendTime int32
	var DigestSentWg sync.WaitGroup
	DigestSentWg.Add(1)
	hello := createHelloMsg(pkiIDInOrg1)
	hello.On("Respond", mock.Anything).Run(func(arguments mock.Arguments) {
		atomic.AddInt32(&digestSendTime, 1)
//确保我们离开频道前只回复摘要
		assert.Equal(t, int32(1), atomic.LoadInt32(&digestSendTime))
		DigestSentWg.Done()
	})
//等我们发送“你好”拉消息
	helloPullWG.Wait()
	go gc.HandleMessage(hello)
	DigestSentWg.Wait()
//让对方离开频道
	gc.LeaveChannel()
//再打个招呼。不应该回应
	go gc.HandleMessage(hello)
//确保它现在不知道其他同行
	assert.Len(t, gc.GetPeers(), 0)
//睡眠时间是拉动间隔的3倍。
//在这段时间内我们不应该拉车。
	time.Sleep(conf.PullInterval * 3)

}

func TestChannelPeriodicalPublishStateInfo(t *testing.T) {
	t.Parallel()
	ledgerHeight := 5
	receivedMsg := int32(0)
	stateInfoReceptionChan := make(chan *proto.SignedGossipMessage, 1)

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)

	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("Gossip", mock.Anything).Run(func(arg mock.Arguments) {
		if atomic.LoadInt32(&receivedMsg) == int32(1) {
			return
		}

		atomic.StoreInt32(&receivedMsg, int32(1))
		msg := arg.Get(0).(*proto.SignedGossipMessage)
		stateInfoReceptionChan <- msg
	})

	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	gc.UpdateLedgerHeight(uint64(ledgerHeight))
	defer gc.Stop()

	var msg *proto.SignedGossipMessage
	select {
	case <-time.After(time.Second * 5):
		t.Fatal("Haven't sent stateInfo on time")
	case m := <-stateInfoReceptionChan:
		msg = m
	}

	assert.Equal(t, ledgerHeight, int(msg.GetStateInfo().Properties.LedgerHeight))
}

func TestChannelMsgStoreEviction(t *testing.T) {
	t.Parallel()
//场景：创建4个阶段，其中通道的pull中介将接收块
//通过拉力。
//块的总量应该受到消息存储容量的限制。
//拉动阶段结束后，我们确保只有最新的块保留在拉动中。
//调解人和旧街区被驱逐。
//We test this by sending a hello message to the pull mediator and inspecting the digest message
//作为响应返回。

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything).Run(func(arg mock.Arguments) {
	})

	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	defer gc.Stop()
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(100, pkiIDInOrg1, channelA)})

	var wg sync.WaitGroup

	msgsPerPhase := uint64(50)
	lastPullPhase := make(chan uint64, msgsPerPhase)
	totalPhases := uint64(4)
	phaseNum := uint64(0)
	wg.Add(int(totalPhases))

	adapter.On("Send", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		msg := args.Get(0).(*proto.SignedGossipMessage)
//忽略发送的所有其他消息，如stateinfo消息
		if !msg.IsPullMsg() {
			return
		}
//当我们到达最后阶段时停止拉动
		if atomic.LoadUint64(&phaseNum) == totalPhases && msg.IsHelloMsg() {
			return
		}

		start := atomic.LoadUint64(&phaseNum) * msgsPerPhase
		end := start + msgsPerPhase
		if msg.IsHelloMsg() {
//前进阶段
			atomic.AddUint64(&phaseNum, uint64(1))
		}

//创建并执行当前拉阶段
		currSeq := sequence(start, end)
		pullPhase := simulatePullPhase(gc, t, &wg, func(envelope *proto.Envelope) {}, currSeq...)
		pullPhase(args)

//如果我们完成了最后一个阶段，请保存顺序以便稍后检查。
		if msg.IsDataReq() && atomic.LoadUint64(&phaseNum) == totalPhases {
			for _, seq := range currSeq {
				lastPullPhase <- seq
			}
			close(lastPullPhase)
		}
	})
//等待所有拉动阶段结束
	wg.Wait()

	msgSentFromPullMediator := make(chan *proto.GossipMessage, 1)

	helloMsg := createHelloMsg(pkiIDInOrg1)
	helloMsg.On("Respond", mock.Anything).Run(func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.GossipMessage)
		if !msg.IsDigestMsg() {
			return
		}
		msgSentFromPullMediator <- msg
	})
	gc.HandleMessage(helloMsg)
	select {
	case msg := <-msgSentFromPullMediator:
//这只是为了检查我们是否及时回复了摘要。
//将消息放回通道以进行进一步检查
		msgSentFromPullMediator <- msg
	case <-time.After(time.Second * 5):
		t.Fatal("Didn't reply with a digest on time")
	}
//仅发送1个摘要
	assert.Len(t, msgSentFromPullMediator, 1)
	msg := <-msgSentFromPullMediator
//它是摘要，而不是其他任何东西，比如更新
	assert.True(t, msg.IsDigestMsg())
	assert.Len(t, msg.GetDataDig().Digests, adapter.GetConf().MaxBlockCountToStore+1)
//检查最后一个序列是否保留。
//因为我们检查了长度，它证明了旧块被丢弃了，因为我们有更多的
//总块数超过我们的容量
	for seq := range lastPullPhase {
		assert.Contains(t, msg.GetDataDig().Digests, []byte(fmt.Sprintf("%d", seq)))
	}
}

func TestChannelPull(t *testing.T) {
	t.Parallel()
	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	receivedBlocksChan := make(chan *proto.SignedGossipMessage, 2)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything).Run(func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.SignedGossipMessage)
		if !msg.IsDataMsg() {
			return
		}
//对等机应该对2个分类账块进行多路复用。
		assert.True(t, msg.IsDataMsg())
		receivedBlocksChan <- msg
	})
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	go gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(100, pkiIDInOrg1, channelA)})

	var wg sync.WaitGroup
	wg.Add(1)
	pullPhase := simulatePullPhase(gc, t, &wg, func(envelope *proto.Envelope) {}, 10, 11)
	adapter.On("Send", mock.Anything, mock.Anything).Run(pullPhase)

	wg.Wait()
	for expectedSeq := 10; expectedSeq <= 11; expectedSeq++ {
		select {
		case <-time.After(time.Second * 5):
			t.Fatal("Haven't received blocks on time")
		case msg := <-receivedBlocksChan:
			assert.Equal(t, uint64(expectedSeq), msg.GetDataMsg().Payload.SeqNum)
		}
	}
}

func TestChannelPullAccessControl(t *testing.T) {
	t.Parallel()
//场景：我们在渠道中有两个组织：org1，org2
//“代理对等”来自ORG1，对等“1”、“2”、“3”来自
//以下组织：
//Org1：“1”
//Org2：“2”，“3”
//我们测试了2个案例：
//1）我们不回复外国机构同行的问候信息。
//2）拉的时候，我们不从国外组织中选择同行。

	cs := &cryptoService{}
	adapter := new(gossipAdapterMock)
	cs.Mock = mock.Mock{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)

	pkiID1 := common.PKIidType("1")
	pkiID2 := common.PKIidType("2")
	pkiID3 := common.PKIidType("3")

	peer1 := discovery.NetworkMember{PKIid: pkiID1, InternalEndpoint: "1", Endpoint: "1"}
	peer2 := discovery.NetworkMember{PKIid: pkiID2, InternalEndpoint: "2", Endpoint: "2"}
	peer3 := discovery.NetworkMember{PKIid: pkiID3, InternalEndpoint: "3", Endpoint: "3"}

	adapter.On("GetOrgOfPeer", pkiIDInOrg1).Return(api.OrgIdentityType("ORG1"))
	adapter.On("GetOrgOfPeer", pkiID1).Return(api.OrgIdentityType("ORG1"))
	adapter.On("GetOrgOfPeer", pkiID2).Return(api.OrgIdentityType("ORG2"))
	adapter.On("GetOrgOfPeer", pkiID3).Return(api.OrgIdentityType("ORG2"))

	adapter.On("GetMembership").Return([]discovery.NetworkMember{peer1, peer2, peer3})
	adapter.On("DeMultiplex", mock.Anything)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("GetConf").Return(conf)

	sentHello := int32(0)
	adapter.On("Send", mock.Anything, mock.Anything).Run(func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.SignedGossipMessage)
		if !msg.IsHelloMsg() {
			return
		}
		atomic.StoreInt32(&sentHello, int32(1))
		peerID := string(arg.Get(1).([]*comm.RemotePeer)[0].PKIID)
		assert.Equal(t, "1", peerID)
		assert.NotEqual(t, "2", peerID, "Sent hello to peer 2 but it's in a different org")
		assert.NotEqual(t, "3", peerID, "Sent hello to peer 3 but it's in a different org")
	})

	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			"ORG1": {},
			"ORG2": {},
		},
	}
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, jcm)
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(100, pkiIDInOrg1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiID1, msg: createStateInfoMsg(100, pkiID1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiID2, msg: createStateInfoMsg(100, pkiID2, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiID3, msg: createStateInfoMsg(100, pkiID3, channelA)})

	respondedChan := make(chan *proto.GossipMessage, 1)
	messageRelayer := func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.GossipMessage)
		respondedChan <- msg
	}

	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(5, channelA), PKIID: pkiIDInOrg1})

	helloMsg := createHelloMsg(pkiID1)
	helloMsg.On("Respond", mock.Anything).Run(messageRelayer)
	go gc.HandleMessage(helloMsg)
	select {
	case <-respondedChan:
	case <-time.After(time.Second):
		assert.Fail(t, "Didn't reply to a hello within a timely manner")
	}

	helloMsg = createHelloMsg(pkiID2)
	helloMsg.On("Respond", mock.Anything).Run(messageRelayer)
	go gc.HandleMessage(helloMsg)
	select {
	case <-respondedChan:
		assert.Fail(t, "Shouldn't have replied to a hello, because the peer is from a foreign org")
	case <-time.After(time.Second):
	}

//睡一会儿，让八卦频道发出问候信息。
	time.Sleep(time.Second * 3)
//请确保我们至少发送了一条Hello消息，否则测试将毫无意义地通过。
	assert.Equal(t, int32(1), atomic.LoadInt32(&sentHello))
}

func TestChannelPeerNotInChannel(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	gossipMessagesSentFromChannel := make(chan *proto.GossipMessage, 1)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})

//首先，我们测试块只能从位于通道中的组织中的对等方接收。
//空的pki-id，应该删除块
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(5, channelA)})
	assert.Equal(t, 0, gc.(*gossipChannel).blockMsgStore.Size())

//已知的pki-id但不在通道中，应删除该块
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(5, channelA), PKIID: pkiIDinOrg2})
	assert.Equal(t, 0, gc.(*gossipChannel).blockMsgStore.Size())
//已知的pki-id，在通道中，应该添加块
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(5, channelA), PKIID: pkiIDInOrg1})
	assert.Equal(t, 1, gc.(*gossipChannel).blockMsgStore.Size())

//接下来，我们确保通道不会响应来自不在通道中的对等方的拉消息（hello或请求）。
	messageRelayer := func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.GossipMessage)
		gossipMessagesSentFromChannel <- msg
	}
//首先，确保它可以从通道中的对等端提取消息。
//让对等方首先发布它在通道中
	gc.HandleMessage(&receivedMsg{msg: createStateInfoMsg(10, pkiIDInOrg1, channelA), PKIID: pkiIDInOrg1})
	helloMsg := createHelloMsg(pkiIDInOrg1)
	helloMsg.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(helloMsg)
	select {
	case <-gossipMessagesSentFromChannel:
	case <-time.After(time.Second * 5):
		t.Fatal("Didn't reply with a digest on time")
	}
//现在，对于不在通道中的对等端（不应发送回消息）
	helloMsg = createHelloMsg(pkiIDinOrg2)
	helloMsg.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(helloMsg)
	select {
	case <-gossipMessagesSentFromChannel:
		t.Fatal("Responded with digest, but shouldn't have since peer is in ORG2 and its not in the channel")
	case <-time.After(time.Second * 1):
	}

//现在，对于一个更高级的场景来说——同行声称在正确的组织中，并且声称在渠道中
//但是MSP声明它不符合该频道的条件
	gc.HandleMessage(&receivedMsg{msg: createStateInfoMsg(10, pkiIDInOrg1ButNotEligible, channelA), PKIID: pkiIDInOrg1ButNotEligible})
//配置MSP
	cs.On("VerifyByChannel", mock.Anything).Return(errors.New("Not eligible"))
	cs.mocked = true
//模拟配置更新
	gc.ConfigureChannel(&joinChanMsg{})
	helloMsg = createHelloMsg(pkiIDInOrg1ButNotEligible)
	helloMsg.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(helloMsg)
	select {
	case <-gossipMessagesSentFromChannel:
		t.Fatal("Responded with digest, but shouldn't have since peer is not eligible for the channel")
	case <-time.After(time.Second * 1):
	}

	cs.Mock = mock.Mock{}

//Ensure we respond to a valid StateInfoRequest
	req, _ := gc.(*gossipChannel).createStateInfoRequest()
	validReceivedMsg := &receivedMsg{
		msg:   req,
		PKIID: pkiIDInOrg1,
	}
	validReceivedMsg.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(validReceivedMsg)
	select {
	case <-gossipMessagesSentFromChannel:
	case <-time.After(time.Second * 5):
		t.Fatal("Didn't reply with a digest on time")
	}

//确保我们不响应来自错误组织中对等方的StateInformationRequest
	invalidReceivedMsg := &receivedMsg{
		msg:   req,
		PKIID: pkiIDinOrg2,
	}
	invalidReceivedMsg.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(invalidReceivedMsg)
	select {
	case <-gossipMessagesSentFromChannel:
		t.Fatal("Responded with digest, but shouldn't have since peer is in ORG2 and its not in the channel")
	case <-time.After(time.Second * 1):
	}

//确保我们不会从正确组织中的对等方对错误通道中的stateinforequest作出响应。
	req2, _ := gc.(*gossipChannel).createStateInfoRequest()
	req2.GetStateInfoPullReq().Channel_MAC = GenerateMAC(pkiIDInOrg1, common.ChainID("B"))
	invalidReceivedMsg2 := &receivedMsg{
		msg:   req2,
		PKIID: pkiIDInOrg1,
	}
	invalidReceivedMsg2.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(invalidReceivedMsg2)
	select {
	case <-gossipMessagesSentFromChannel:
		t.Fatal("Responded with stateInfo request, but shouldn't have since it has the wrong MAC")
	case <-time.After(time.Second * 1):
	}
}

func TestChannelIsInChannel(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)

	assert.False(t, gc.IsOrgInChannel(nil))
	assert.True(t, gc.IsOrgInChannel(orgInChannelA))
	assert.False(t, gc.IsOrgInChannel(orgNotInChannelA))
	assert.True(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.False(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDinOrg2}))
}

func TestChannelIsSubscribed(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	gc.HandleMessage(&receivedMsg{msg: createStateInfoMsg(10, pkiIDInOrg1, channelA), PKIID: pkiIDInOrg1})
	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
}

func TestChannelAddToMessageStore(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	demuxedMsgs := make(chan *proto.SignedGossipMessage, 1)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything).Run(func(arg mock.Arguments) {
		demuxedMsgs <- arg.Get(0).(*proto.SignedGossipMessage)
	})

//检查添加错误类型的消息是否不会使程序崩溃
	gc.AddToMsgStore(createHelloMsg(pkiIDInOrg1).GetGossipMessage())

//我们要确保如果我们收到一条新的消息，它是去复用的，
//但是如果我们把这样的消息放在消息存储中，当我们
//再次收到该消息
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(11, channelA), PKIID: pkiIDInOrg1})
	select {
	case <-time.After(time.Second):
		t.Fatal("Haven't detected a demultiplexing within a time period")
	case <-demuxedMsgs:
	}
	gc.AddToMsgStore(dataMsgOfChannel(12, channelA))
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(12, channelA), PKIID: pkiIDInOrg1})
	select {
	case <-time.After(time.Second):
	case <-demuxedMsgs:
		t.Fatal("Demultiplexing detected, even though it wasn't supposed to happen")
	}

	gc.AddToMsgStore(createStateInfoMsg(10, pkiIDInOrg1, channelA))
	helloMsg := createHelloMsg(pkiIDInOrg1)
	respondedChan := make(chan struct{}, 1)
	helloMsg.On("Respond", mock.Anything).Run(func(arg mock.Arguments) {
		respondedChan <- struct{}{}
	})
	gc.HandleMessage(helloMsg)
	select {
	case <-time.After(time.Second):
		t.Fatal("Haven't responded to hello message within a time period")
	case <-respondedChan:
	}

	gc.HandleMessage(&receivedMsg{msg: createStateInfoMsg(10, pkiIDInOrg1, channelA), PKIID: pkiIDInOrg1})
	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
}

func TestChannelBlockExpiration(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	demuxedMsgs := make(chan *proto.SignedGossipMessage, 1)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything).Run(func(arg mock.Arguments) {
		demuxedMsgs <- arg.Get(0).(*proto.SignedGossipMessage)
	})
	respondedChan := make(chan *proto.GossipMessage, 1)
	messageRelayer := func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.GossipMessage)
		respondedChan <- msg
	}

//我们要确保如果我们收到一条新的消息，它是去复用的，
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(11, channelA), PKIID: pkiIDInOrg1})
	select {
	case <-time.After(time.Second):
		t.Fatal("Haven't detected a demultiplexing within a time period")
	case <-demuxedMsgs:
	}

//让我们检查摘要和状态信息存储
	stateInfoMsg := createStateInfoMsg(10, pkiIDInOrg1, channelA)
	gc.AddToMsgStore(stateInfoMsg)
	helloMsg := createHelloMsg(pkiIDInOrg1)
	helloMsg.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(helloMsg)
	select {
	case <-time.After(time.Second):
		t.Fatal("Haven't responded to hello message within a time period")
	case msg := <-respondedChan:
		if msg.IsDigestMsg() {
			assert.Equal(t, 1, len(msg.GetDataDig().Digests), "Number of digests returned by channel blockPuller incorrect")
		} else {
			t.Fatal("Not correct pull msg type in response - expect digest")
		}
	}

	time.Sleep(gc.(*gossipChannel).GetConf().BlockExpirationInterval + time.Second)

//消息在存储中过期，但在我们
//再次收到该消息
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(11, channelA), PKIID: pkiIDInOrg1})
	select {
	case <-time.After(time.Second):
	case <-demuxedMsgs:
		t.Fatal("Demultiplexing detected, even though it wasn't supposed to happen")
	}

//让我们检查摘要和状态信息存储-状态信息已过期，其添加将不做任何操作，并且不应发送摘要
	gc.AddToMsgStore(stateInfoMsg)
	gc.HandleMessage(helloMsg)
	select {
	case <-time.After(time.Second):
	case <-respondedChan:
		t.Fatal("No digest should be sent")
	}

	time.Sleep(gc.(*gossipChannel).GetConf().BlockExpirationInterval + time.Second)
//消息已从存储中删除，因此当我们
//再次收到该消息
	gc.HandleMessage(&receivedMsg{msg: dataMsgOfChannel(11, channelA), PKIID: pkiIDInOrg1})
	select {
	case <-time.After(time.Second):
		t.Fatal("Haven't detected a demultiplexing within a time period")
	case <-demuxedMsgs:
	}

//让我们检查摘要和状态信息存储-状态信息也被删除，因此它将被重新添加并创建摘要
	gc.AddToMsgStore(stateInfoMsg)
	gc.HandleMessage(helloMsg)
	select {
	case <-time.After(time.Second):
		t.Fatal("Haven't responded to hello message within a time period")
	case msg := <-respondedChan:
		if msg.IsDigestMsg() {
			assert.Equal(t, 1, len(msg.GetDataDig().Digests), "Number of digests returned by channel blockPuller incorrect")
		} else {
			t.Fatal("Not correct pull msg type in response - expect digest")
		}
	}

	gc.Stop()
}

func TestChannelBadBlocks(t *testing.T) {
	t.Parallel()
	receivedMessages := make(chan *proto.SignedGossipMessage, 1)
	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})

	adapter.On("DeMultiplex", mock.Anything).Run(func(args mock.Arguments) {
		receivedMessages <- args.Get(0).(*proto.SignedGossipMessage)
	})

//发送有效的块
	gc.HandleMessage(&receivedMsg{msg: createDataMsg(1, channelA), PKIID: pkiIDInOrg1})
	assert.Len(t, receivedMessages, 1)
<-receivedMessages //排水

//发送带有错误通道的块
	gc.HandleMessage(&receivedMsg{msg: createDataMsg(2, common.ChainID("B")), PKIID: pkiIDInOrg1})
	assert.Len(t, receivedMessages, 0)

//发送一个有效负载为空的块
	dataMsg := createDataMsg(3, channelA)
	dataMsg.GetDataMsg().Payload = nil
	gc.HandleMessage(&receivedMsg{msg: dataMsg, PKIID: pkiIDInOrg1})
	assert.Len(t, receivedMessages, 0)

//发送带有错误签名的块
	cs.Mock = mock.Mock{}
	cs.On("VerifyBlock", mock.Anything).Return(errors.New("Bad signature"))
	gc.HandleMessage(&receivedMsg{msg: createDataMsg(4, channelA), PKIID: pkiIDInOrg1})
	assert.Len(t, receivedMessages, 0)
}

func TestChannelPulledBadBlocks(t *testing.T) {
	t.Parallel()

//用坏通道块测试拉力
	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	adapter.On("DeMultiplex", mock.Anything)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})

	var wg sync.WaitGroup
	wg.Add(1)

	changeChan := func(env *proto.Envelope) {
		sMsg, _ := env.ToGossipMessage()
		sMsg.Channel = []byte("B")
		sMsg, _ = sMsg.NoopSign()
		env.Payload = sMsg.Payload
	}

	pullPhase1 := simulatePullPhase(gc, t, &wg, changeChan, 10, 11)
	adapter.On("Send", mock.Anything, mock.Anything).Run(pullPhase1)
	adapter.On("DeMultiplex", mock.Anything)
	wg.Wait()
	gc.Stop()
	assert.Equal(t, 0, gc.(*gossipChannel).blockMsgStore.Size())

//用符号错误的块测试拉力
	cs = &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(errors.New("Bad block"))
	adapter = new(gossipAdapterMock)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	gc = NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})

	var wg2 sync.WaitGroup
	wg2.Add(1)
	noop := func(env *proto.Envelope) {

	}
	pullPhase2 := simulatePullPhase(gc, t, &wg2, noop, 10, 11)
	adapter.On("Send", mock.Anything, mock.Anything).Run(pullPhase2)
	wg2.Wait()
	assert.Equal(t, 0, gc.(*gossipChannel).blockMsgStore.Size())

//Test a pull with an empty block
	cs = &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter = new(gossipAdapterMock)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	gc = NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})

	var wg3 sync.WaitGroup
	wg3.Add(1)
	emptyBlock := func(env *proto.Envelope) {
		sMsg, _ := env.ToGossipMessage()
		sMsg.GossipMessage.GetDataMsg().Payload = nil
		sMsg, _ = sMsg.NoopSign()
		env.Payload = sMsg.Payload
	}
	pullPhase3 := simulatePullPhase(gc, t, &wg3, emptyBlock, 10, 11)
	adapter.On("Send", mock.Anything, mock.Anything).Run(pullPhase3)
	wg3.Wait()
	assert.Equal(t, 0, gc.(*gossipChannel).blockMsgStore.Size())

//使用非阻塞消息测试拉取
	cs = &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)

	adapter = new(gossipAdapterMock)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	gc = NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})

	var wg4 sync.WaitGroup
	wg4.Add(1)
	nonBlockMsg := func(env *proto.Envelope) {
		sMsg, _ := env.ToGossipMessage()
		sMsg.Content = createHelloMsg(pkiIDInOrg1).GetGossipMessage().Content
		sMsg, _ = sMsg.NoopSign()
		env.Payload = sMsg.Payload
	}
	pullPhase4 := simulatePullPhase(gc, t, &wg4, nonBlockMsg, 10, 11)
	adapter.On("Send", mock.Anything, mock.Anything).Run(pullPhase4)
	wg4.Wait()
	assert.Equal(t, 0, gc.(*gossipChannel).blockMsgStore.Size())
}

func TestChannelStateInfoSnapshot(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	adapter.On("Lookup", mock.Anything).Return(&discovery.NetworkMember{Endpoint: "localhost:5000"})
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	sentMessages := make(chan *proto.GossipMessage, 10)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("ValidateStateInfoMessage", mock.Anything).Return(nil)

//确保忽略不在通道中的对等端的StateInfo快照
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: stateInfoSnapshotForChannel(common.ChainID("B"), createStateInfoMsg(4, pkiIDInOrg1, channelA))})
	assert.Empty(t, gc.GetPeers())
//Ensure we ignore invalid stateInfo snapshots
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: stateInfoSnapshotForChannel(channelA, createStateInfoMsg(4, pkiIDInOrg1, common.ChainID("B")))})
	assert.Empty(t, gc.GetPeers())

//确保忽略来自不在通道中的对等方的StateInfo消息
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: stateInfoSnapshotForChannel(channelA, createStateInfoMsg(4, pkiIDinOrg2, channelA))})
	assert.Empty(t, gc.GetPeers())

//确保忽略不在组织中的对等方的stateinfo快照
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDinOrg2, msg: stateInfoSnapshotForChannel(channelA, createStateInfoMsg(4, pkiIDInOrg1, channelA))})
	assert.Empty(t, gc.GetPeers())

//确保忽略带有错误macs的stateinfo消息的stateinfo快照
	sim := createStateInfoMsg(4, pkiIDInOrg1, channelA)
	sim.GetStateInfo().Channel_MAC = append(sim.GetStateInfo().Channel_MAC, 1)
	sim, _ = sim.NoopSign()
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: stateInfoSnapshotForChannel(channelA, sim)})
	assert.Empty(t, gc.GetPeers())

//确保使用正确的stateinfo消息忽略stateinfo快照，但使用错误的macs
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: stateInfoSnapshotForChannel(channelA, createStateInfoMsg(4, pkiIDInOrg1, channelA))})

//确保我们处理的状态信息快照正常
	stateInfoMsg := &receivedMsg{PKIID: pkiIDInOrg1, msg: stateInfoSnapshotForChannel(channelA, createStateInfoMsg(4, pkiIDInOrg1, channelA))}
	gc.HandleMessage(stateInfoMsg)
	assert.NotEmpty(t, gc.GetPeers())
	assert.Equal(t, 4, int(gc.GetPeers()[0].Properties.LedgerHeight))

//检查我们没有用错误的MAC响应StateInfoSnapshot请求
	sMsg, _ := (&proto.GossipMessage{
		Tag: proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_StateInfoPullReq{
			StateInfoPullReq: &proto.StateInfoPullRequest{
				Channel_MAC: append(GenerateMAC(pkiIDInOrg1, channelA), 1),
			},
		},
	}).NoopSign()
	snapshotReq := &receivedMsg{
		PKIID: pkiIDInOrg1,
		msg:   sMsg,
	}
	snapshotReq.On("Respond", mock.Anything).Run(func(args mock.Arguments) {
		sentMessages <- args.Get(0).(*proto.GossipMessage)
	})

	go gc.HandleMessage(snapshotReq)
	select {
	case <-time.After(time.Second):
	case <-sentMessages:
		assert.Fail(t, "Shouldn't have responded to this StateInfoSnapshot request because of bad MAC")
	}

//确保我们使用有效的MAC响应StateInfoSnapshot请求
	sMsg, _ = (&proto.GossipMessage{
		Tag: proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_StateInfoPullReq{
			StateInfoPullReq: &proto.StateInfoPullRequest{
				Channel_MAC: GenerateMAC(pkiIDInOrg1, channelA),
			},
		},
	}).NoopSign()
	snapshotReq = &receivedMsg{
		PKIID: pkiIDInOrg1,
		msg:   sMsg,
	}
	snapshotReq.On("Respond", mock.Anything).Run(func(args mock.Arguments) {
		sentMessages <- args.Get(0).(*proto.GossipMessage)
	})

	go gc.HandleMessage(snapshotReq)
	select {
	case <-time.After(time.Second):
		t.Fatal("Haven't received a state info snapshot on time")
	case msg := <-sentMessages:
		elements := msg.GetStateSnapshot().Elements
		assert.Len(t, elements, 1)
		sMsg, err := elements[0].ToGossipMessage()
		assert.NoError(t, err)
		assert.Equal(t, 4, int(sMsg.GetStateInfo().Properties.LedgerHeight))
	}

//确保在收到无效状态信息消息时不会崩溃
	invalidStateInfoSnapshot := stateInfoSnapshotForChannel(channelA, createStateInfoMsg(4, pkiIDInOrg1, channelA))
	invalidStateInfoSnapshot.GetStateSnapshot().Elements = []*proto.Envelope{createHelloMsg(pkiIDInOrg1).GetSourceEnvelope()}
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: invalidStateInfoSnapshot})

//确保我们不会崩溃，如果我们从一个同伴那里得到一条StateInfoMessage，它的组织不为人所知。
	invalidStateInfoSnapshot = stateInfoSnapshotForChannel(channelA, createStateInfoMsg(4, common.PKIidType("unknown"), channelA))
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: invalidStateInfoSnapshot})
}

func TestInterOrgExternalEndpointDisclosure(t *testing.T) {
	t.Parallel()
	cs := &cryptoService{}
	adapter := new(gossipAdapterMock)
	pkiID1 := common.PKIidType("withExternalEndpoint")
	pkiID2 := common.PKIidType("noExternalEndpoint")
	pkiID3 := common.PKIidType("pkiIDinOrg2")
	adapter.On("Lookup", pkiID1).Return(&discovery.NetworkMember{Endpoint: "localhost:5000"})
	adapter.On("Lookup", pkiID2).Return(&discovery.NetworkMember{})
	adapter.On("Lookup", pkiID3).Return(&discovery.NetworkMember{})
	adapter.On("GetOrgOfPeer", pkiID1).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiID2).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiID3).Return(api.OrgIdentityType("ORG2"))
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	configureAdapter(adapter)
	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(orgInChannelA): {},
			"ORG2":                {},
		},
	}
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, jcm)
	gc.HandleMessage(&receivedMsg{PKIID: pkiID1, msg: createStateInfoMsg(0, pkiID1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiID2, msg: createStateInfoMsg(0, pkiID2, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiID2, msg: createStateInfoMsg(0, pkiID3, channelA)})

	sentMessages := make(chan *proto.GossipMessage, 10)

//检查是否只返回具有外部终结点的对等方的StateInfo消息
//其他组织的同行
	sMsg, _ := (&proto.GossipMessage{
		Tag: proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_StateInfoPullReq{
			StateInfoPullReq: &proto.StateInfoPullRequest{
				Channel_MAC: GenerateMAC(pkiID3, channelA),
			},
		},
	}).NoopSign()
	snapshotReq := &receivedMsg{
		PKIID: pkiID3,
		msg:   sMsg,
	}
	snapshotReq.On("Respond", mock.Anything).Run(func(args mock.Arguments) {
		sentMessages <- args.Get(0).(*proto.GossipMessage)
	})

	go gc.HandleMessage(snapshotReq)
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Should have responded to this StateInfoSnapshot, but didn't")
	case msg := <-sentMessages:
		elements := msg.GetStateSnapshot().Elements
		assert.Len(t, elements, 2)
		m1, _ := elements[0].ToGossipMessage()
		m2, _ := elements[1].ToGossipMessage()
		pkiIDs := [][]byte{m1.GetStateInfo().PkiId, m2.GetStateInfo().PkiId}
		assert.Contains(t, pkiIDs, []byte(pkiID1))
		assert.Contains(t, pkiIDs, []byte(pkiID3))
	}

//检查我们是否将所有stateinfo消息返回给组织中的对等方，无论
//外国组织的同行是否有外部端点
	sMsg, _ = (&proto.GossipMessage{
		Tag: proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_StateInfoPullReq{
			StateInfoPullReq: &proto.StateInfoPullRequest{
				Channel_MAC: GenerateMAC(pkiID2, channelA),
			},
		},
	}).NoopSign()
	snapshotReq = &receivedMsg{
		PKIID: pkiID2,
		msg:   sMsg,
	}
	snapshotReq.On("Respond", mock.Anything).Run(func(args mock.Arguments) {
		sentMessages <- args.Get(0).(*proto.GossipMessage)
	})

	go gc.HandleMessage(snapshotReq)
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Should have responded to this StateInfoSnapshot, but didn't")
	case msg := <-sentMessages:
		elements := msg.GetStateSnapshot().Elements
		assert.Len(t, elements, 3)
		m1, _ := elements[0].ToGossipMessage()
		m2, _ := elements[1].ToGossipMessage()
		m3, _ := elements[2].ToGossipMessage()
		pkiIDs := [][]byte{m1.GetStateInfo().PkiId, m2.GetStateInfo().PkiId, m3.GetStateInfo().PkiId}
		assert.Contains(t, pkiIDs, []byte(pkiID1))
		assert.Contains(t, pkiIDs, []byte(pkiID2))
		assert.Contains(t, pkiIDs, []byte(pkiID3))
	}
}

func TestChannelStop(t *testing.T) {
	t.Parallel()

	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	adapter := new(gossipAdapterMock)
	var sendCount int32
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	adapter.On("Send", mock.Anything, mock.Anything).Run(func(mock.Arguments) {
		atomic.AddInt32(&sendCount, int32(1))
	})
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	time.Sleep(time.Second)
	gc.Stop()
	oldCount := atomic.LoadInt32(&sendCount)
	t1 := time.Now()
	for {
		if time.Since(t1).Nanoseconds() > (time.Second * 15).Nanoseconds() {
			t.Fatal("Stop failed")
		}
		time.Sleep(time.Second)
		newCount := atomic.LoadInt32(&sendCount)
		if newCount == oldCount {
			break
		}
		oldCount = newCount
	}
}

func TestChannelReconfigureChannel(t *testing.T) {
	t.Parallel()

//场景：我们测试以下内容：
//用过时的joinChannel消息更新频道不起作用
//从一个渠道中删除一个组织确实反映在这一点上
//八卦频道不认为来自该组织的同龄人是
//通道中的对等方，并且拒绝与通道相关的任何联系
//与该频道的同行

	cs := &cryptoService{}
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})

	adapter.On("GetConf").Return(conf)
	adapter.On("GetMembership").Return([]discovery.NetworkMember{})
	adapter.On("OrgByPeerIdentity", api.PeerIdentityType(orgInChannelA)).Return(orgInChannelA)
	adapter.On("OrgByPeerIdentity", api.PeerIdentityType(orgNotInChannelA)).Return(orgNotInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDInOrg1).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDinOrg2).Return(orgNotInChannelA)

	outdatedJoinChanMsg := &joinChanMsg{
		getTS: func() time.Time {
			return time.Now()
		},
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(orgNotInChannelA): {},
		},
	}

	newJoinChanMsg := &joinChanMsg{
		getTS: func() time.Time {
			return time.Now().Add(time.Millisecond * 100)
		},
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(orgInChannelA): {},
		},
	}

	updatedJoinChanMsg := &joinChanMsg{
		getTS: func() time.Time {
			return time.Now().Add(time.Millisecond * 200)
		},
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(orgNotInChannelA): {},
		},
	}

	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, api.JoinChannelMessage(newJoinChanMsg))

//再打一次，确保东西不会掉下来。
	gc.ConfigureChannel(api.JoinChannelMessage(newJoinChanMsg))

	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)

	assert.True(t, gc.IsOrgInChannel(orgInChannelA))
	assert.False(t, gc.IsOrgInChannel(orgNotInChannelA))
	assert.True(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.False(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDinOrg2}))

	gc.ConfigureChannel(outdatedJoinChanMsg)
	assert.True(t, gc.IsOrgInChannel(orgInChannelA))
	assert.False(t, gc.IsOrgInChannel(orgNotInChannelA))
	assert.True(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.False(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDinOrg2}))

	gc.ConfigureChannel(updatedJoinChanMsg)
	gc.ConfigureChannel(updatedJoinChanMsg)
	assert.False(t, gc.IsOrgInChannel(orgInChannelA))
	assert.True(t, gc.IsOrgInChannel(orgNotInChannelA))
	assert.False(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.True(t, gc.IsMemberInChan(discovery.NetworkMember{PKIid: pkiIDinOrg2}))

//确保我们不响应来自错误组织中对等方的StateInformationRequest
	sMsg, _ := gc.(*gossipChannel).createStateInfoRequest()
	invalidReceivedMsg := &receivedMsg{
		msg:   sMsg,
		PKIID: pkiIDInOrg1,
	}
	gossipMessagesSentFromChannel := make(chan *proto.GossipMessage, 1)
	messageRelayer := func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.GossipMessage)
		gossipMessagesSentFromChannel <- msg
	}
	invalidReceivedMsg.On("Respond", mock.Anything).Run(messageRelayer)
	gc.HandleMessage(invalidReceivedMsg)
	select {
	case <-gossipMessagesSentFromChannel:
		t.Fatal("Responded with digest, but shouldn't have since peer is in ORG2 and its not in the channel")
	case <-time.After(time.Second * 1):
	}
}

func TestChannelNoAnchorPeers(t *testing.T) {
	t.Parallel()

//场景：我们收到一条没有锚点对等的加入通道消息
//在这种情况下，我们应该在频道里

	cs := &cryptoService{}
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})

	adapter.On("GetConf").Return(conf)
	adapter.On("GetMembership").Return([]discovery.NetworkMember{})
	adapter.On("OrgByPeerIdentity", api.PeerIdentityType(orgInChannelA)).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDInOrg1).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDinOrg2).Return(orgNotInChannelA)

	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(orgInChannelA): {},
		},
	}

	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, api.JoinChannelMessage(jcm))
	assert.True(t, gc.IsOrgInChannel(orgInChannelA))
}

func TestGossipChannelEligibility(t *testing.T) {
	t.Parallel()

//场景：我们在一个组织中有一个对等点，它将一个通道与org1和org2连接起来。
//它接收其他同行的状态信息和资格信息。
//这些同龄人中的一个被检查。
//在测试过程中，通道被重新配置，并且到期
//对对等身份进行了模拟。

	cs := &cryptoService{}
	selfPKIID := common.PKIidType("p")
	adapter := new(gossipAdapterMock)
	pkiIDinOrg3 := common.PKIidType("pkiIDinOrg3")
	members := []discovery.NetworkMember{
		{PKIid: pkiIDInOrg1},
		{PKIid: pkiIDInOrg1ButNotEligible},
		{PKIid: pkiIDinOrg2},
		{PKIid: pkiIDinOrg3},
	}
	adapter.On("GetMembership").Return(members)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	adapter.On("GetConf").Return(conf)

//首先，除了pkiidinorg3，所有对等端都在通道中。
	org1 := api.OrgIdentityType("ORG1")
	org2 := api.OrgIdentityType("ORG2")
	org3 := api.OrgIdentityType("ORG3")

	adapter.On("GetOrgOfPeer", selfPKIID).Return(org1)
	adapter.On("GetOrgOfPeer", pkiIDInOrg1).Return(org1)
	adapter.On("GetOrgOfPeer", pkiIDinOrg2).Return(org2)
	adapter.On("GetOrgOfPeer", pkiIDInOrg1ButNotEligible).Return(org1)
	adapter.On("GetOrgOfPeer", pkiIDinOrg3).Return(org3)

	gc := NewGossipChannel(selfPKIID, orgInChannelA, cs, channelA, adapter, &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(org1): {},
			string(org2): {},
		},
	})
//每个对等端都发送一条StateInfo消息
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDinOrg2, msg: createStateInfoMsg(1, pkiIDinOrg2, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1ButNotEligible, msg: createStateInfoMsg(1, pkiIDInOrg1ButNotEligible, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDinOrg3, msg: createStateInfoMsg(1, pkiIDinOrg3, channelA)})

	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg2}))
	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1ButNotEligible}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg3}))

//确保返回通道中的对等点
	assert.True(t, gc.PeerFilter(func(signature api.PeerSignature) bool {
		return true
	})(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.True(t, gc.PeerFilter(func(signature api.PeerSignature) bool {
		return true
	})(discovery.NetworkMember{PKIid: pkiIDinOrg2}))
//但不是那些不在渠道中的同行
	assert.False(t, gc.PeerFilter(func(signature api.PeerSignature) bool {
		return true
	})(discovery.NetworkMember{PKIid: pkiIDinOrg3}))

//确保考虑给定谓词
	assert.True(t, gc.PeerFilter(func(signature api.PeerSignature) bool {
		return bytes.Equal(signature.PeerIdentity, []byte("pkiIDinOrg2"))
	})(discovery.NetworkMember{PKIid: pkiIDinOrg2}))

	assert.False(t, gc.PeerFilter(func(signature api.PeerSignature) bool {
		return bytes.Equal(signature.PeerIdentity, []byte("pkiIDinOrg2"))
	})(discovery.NetworkMember{PKIid: pkiIDInOrg1}))

//从通道中删除org2
	gc.ConfigureChannel(&joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(org1): {},
		},
	})

	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg2}))
	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1ButNotEligible}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg3}))

//现在模拟一个从通道读卡器中删除pkiidinorg1buttnoteligible的配置更新
	cs.mocked = true
	cs.On("VerifyByChannel", api.PeerIdentityType(pkiIDInOrg1ButNotEligible)).Return(errors.New("Not a channel reader"))
	cs.On("VerifyByChannel", mock.Anything).Return(nil)
	gc.ConfigureChannel(&joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			string(org1): {},
		},
	})
	assert.True(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg2}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1ButNotEligible}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg3}))

//现在模拟pkiidingorg1的证书过期。
//这是通过要求适配器通过pki-id查找标识来完成的，但是如果证书
//已过期，映射将被删除，因此查找不会产生任何结果。
	adapter.On("GetIdentityByPKIID", pkiIDInOrg1).Return(api.PeerIdentityType(nil))
	adapter.On("GetIdentityByPKIID", pkiIDinOrg2).Return(api.PeerIdentityType(pkiIDinOrg2))
	adapter.On("GetIdentityByPKIID", pkiIDInOrg1ButNotEligible).Return(api.PeerIdentityType(pkiIDInOrg1ButNotEligible))
	adapter.On("GetIdentityByPKIID", pkiIDinOrg3).Return(api.PeerIdentityType(pkiIDinOrg3))

	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg2}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1ButNotEligible}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg3}))

//现在对stateinfo消息进行另一次更新，这次使用更新的分类帐高度（覆盖以前的消息）
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(2, pkiIDInOrg1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(2, pkiIDinOrg2, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(2, pkiIDInOrg1ButNotEligible, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(2, pkiIDinOrg3, channelA)})

//确保访问控制分辨率没有更改
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg2}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDInOrg1ButNotEligible}))
	assert.False(t, gc.EligibleForChannel(discovery.NetworkMember{PKIid: pkiIDinOrg3}))
}

func TestChannelGetPeers(t *testing.T) {
	t.Parallel()

//场景：我们在一个组织中有一个对等方，并且通知该对等方几个对等方
//存在，其中一些：
//（1）加入其信道，并有资格接收数据块。
//（2）加入其通道，但不符合接收块的条件（MSP不允许这样做）。
//（3）说他们加入了它的频道，但实际上来自一个不在频道中的组织。
//getpeers查询只应返回属于第一个组的对等方。
	cs := &cryptoService{}
	adapter := new(gossipAdapterMock)
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	members := []discovery.NetworkMember{
		{PKIid: pkiIDInOrg1},
		{PKIid: pkiIDInOrg1ButNotEligible},
		{PKIid: pkiIDinOrg2},
	}
	configureAdapter(adapter, members...)
	gc := NewGossipChannel(common.PKIidType("p0"), orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDinOrg2, channelA)})
	assert.Len(t, gc.GetPeers(), 1)
	assert.Equal(t, pkiIDInOrg1, gc.GetPeers()[0].PKIid)

//Ensure envelope from GetPeers is valid
	gMsg, _ := gc.GetPeers()[0].Envelope.ToGossipMessage()
	assert.Equal(t, []byte(pkiIDInOrg1), gMsg.GetStateInfo().PkiId)

	gc.HandleMessage(&receivedMsg{msg: createStateInfoMsg(10, pkiIDInOrg1ButNotEligible, channelA), PKIID: pkiIDInOrg1ButNotEligible})
	cs.On("VerifyByChannel", mock.Anything).Return(errors.New("Not eligible"))
	cs.mocked = true
//模拟配置更新
	gc.ConfigureChannel(&joinChanMsg{})
	assert.Len(t, gc.GetPeers(), 0)

//现在重新创建gc并损坏mac
//确保stateinfo消息不算数
	gc = NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	msg := &receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)}
	msg.GetGossipMessage().GetStateInfo().Channel_MAC = GenerateMAC(pkiIDinOrg2, channelA)
	gc.HandleMessage(msg)
	assert.Len(t, gc.GetPeers(), 0)
}

func TestOnDemandGossip(t *testing.T) {
	t.Parallel()

//场景：更新元数据并确保只有1个分发
//成员不为空时发生

	cs := &cryptoService{}
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter)

	gossipedEvents := make(chan struct{})

	conf := conf
	conf.PublishStateInfoInterval = time.Millisecond * 200
	adapter.On("GetConf").Return(conf)
	adapter.On("GetMembership").Return([]discovery.NetworkMember{})
	adapter.On("Gossip", mock.Anything).Run(func(mock.Arguments) {
		gossipedEvents <- struct{}{}
	})
	adapter.On("Forward", mock.Anything)
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, api.JoinChannelMessage(&joinChanMsg{}))
	defer gc.Stop()
	select {
	case <-gossipedEvents:
		assert.Fail(t, "Should not have gossiped because metadata has not been updated yet")
	case <-time.After(time.Millisecond * 500):
	}
	gc.UpdateLedgerHeight(0)
	select {
	case <-gossipedEvents:
	case <-time.After(time.Second):
		assert.Fail(t, "Didn't gossip within a timely manner")
	}
	select {
	case <-gossipedEvents:
	case <-time.After(time.Second):
		assert.Fail(t, "Should have gossiped a second time, because membership is empty")
	}
	adapter = new(gossipAdapterMock)
	configureAdapter(adapter, []discovery.NetworkMember{{}}...)
	adapter.On("Gossip", mock.Anything).Run(func(mock.Arguments) {
		gossipedEvents <- struct{}{}
	})
	adapter.On("Forward", mock.Anything)
	gc.(*gossipChannel).Adapter = adapter
	select {
	case <-gossipedEvents:
	case <-time.After(time.Second):
		assert.Fail(t, "Should have gossiped a third time")
	}
	select {
	case <-gossipedEvents:
		assert.Fail(t, "Should not have gossiped a fourth time, because dirty flag should have been turned off")
	case <-time.After(time.Millisecond * 500):
	}
	gc.UpdateLedgerHeight(1)
	select {
	case <-gossipedEvents:
	case <-time.After(time.Second):
		assert.Fail(t, "Should have gossiped a block now, because got a new StateInfo message")
	}
}

func TestChannelPullWithDigestsFilter(t *testing.T) {
	t.Parallel()
	cs := &cryptoService{}
	cs.On("VerifyBlock", mock.Anything).Return(nil)
	receivedBlocksChan := make(chan *proto.SignedGossipMessage, 2)
	adapter := new(gossipAdapterMock)
	configureAdapter(adapter, discovery.NetworkMember{PKIid: pkiIDInOrg1})
	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("DeMultiplex", mock.Anything).Run(func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.SignedGossipMessage)
		if !msg.IsDataMsg() {
			return
		}
//对等机应该对1个分类块进行多路复用。
		assert.True(t, msg.IsDataMsg())
		receivedBlocksChan <- msg
	})
	gc := NewGossipChannel(pkiIDInOrg1, orgInChannelA, cs, channelA, adapter, &joinChanMsg{})
	go gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(100, pkiIDInOrg1, channelA)})

	gc.UpdateLedgerHeight(11)

	var wg sync.WaitGroup
	wg.Add(1)

	pullPhase := simulatePullPhaseWithVariableDigest(gc, t, &wg, func(envelope *proto.Envelope) {}, [][]byte{[]byte("10"), []byte("11")}, []string{"11"}, 11)
	adapter.On("Send", mock.Anything, mock.Anything).Run(pullPhase)
	wg.Wait()

	select {
	case <-time.After(time.Second * 5):
		t.Fatal("Haven't received blocks on time")
	case msg := <-receivedBlocksChan:
		assert.Equal(t, uint64(11), msg.GetDataMsg().Payload.SeqNum)
	}

}

func createDataUpdateMsg(nonce uint64, seqs ...uint64) *proto.SignedGossipMessage {
	msg := &proto.GossipMessage{
		Nonce:   0,
		Channel: []byte(channelA),
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Content: &proto.GossipMessage_DataUpdate{
			DataUpdate: &proto.DataUpdate{
				MsgType: proto.PullMsgType_BLOCK_MSG,
				Nonce:   nonce,
				Data:    []*proto.Envelope{},
			},
		},
	}
	for _, seq := range seqs {
		msg.GetDataUpdate().Data = append(msg.GetDataUpdate().Data, createDataMsg(seq, channelA).Envelope)
	}
	sMsg, _ := msg.NoopSign()
	return sMsg
}

func createHelloMsg(PKIID common.PKIidType) *receivedMsg {
	msg := &proto.GossipMessage{
		Channel: []byte(channelA),
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Content: &proto.GossipMessage_Hello{
			Hello: &proto.GossipHello{
				Nonce:    500,
				Metadata: nil,
				MsgType:  proto.PullMsgType_BLOCK_MSG,
			},
		},
	}
	sMsg, _ := msg.NoopSign()
	return &receivedMsg{msg: sMsg, PKIID: PKIID}
}

func dataMsgOfChannel(seqnum uint64, channel common.ChainID) *proto.SignedGossipMessage {
	sMsg, _ := (&proto.GossipMessage{
		Channel: []byte(channel),
		Nonce:   0,
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{
				Payload: &proto.Payload{
					Data:   []byte{},
					SeqNum: seqnum,
				},
			},
		},
	}).NoopSign()
	return sMsg
}

func createStateInfoMsg(ledgerHeight int, pkiID common.PKIidType, channel common.ChainID) *proto.SignedGossipMessage {
	sMsg, _ := (&proto.GossipMessage{
		Tag: proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_StateInfo{
			StateInfo: &proto.StateInfo{
				Channel_MAC: GenerateMAC(pkiID, channel),
				Timestamp:   &proto.PeerTime{IncNum: uint64(time.Now().UnixNano()), SeqNum: 1},
				PkiId:       []byte(pkiID),
				Properties: &proto.Properties{
					LedgerHeight: uint64(ledgerHeight),
				},
			},
		},
	}).NoopSign()
	return sMsg
}

func stateInfoSnapshotForChannel(chainID common.ChainID, stateInfoMsgs ...*proto.SignedGossipMessage) *proto.SignedGossipMessage {
	envelopes := make([]*proto.Envelope, len(stateInfoMsgs))
	for i, sim := range stateInfoMsgs {
		envelopes[i] = sim.Envelope
	}
	sMsg, _ := (&proto.GossipMessage{
		Channel: chainID,
		Tag:     proto.GossipMessage_CHAN_OR_ORG,
		Nonce:   0,
		Content: &proto.GossipMessage_StateSnapshot{
			StateSnapshot: &proto.StateInfoSnapshot{
				Elements: envelopes,
			},
		},
	}).NoopSign()
	return sMsg
}

func createDataMsg(seqnum uint64, channel common.ChainID) *proto.SignedGossipMessage {
	sMsg, _ := (&proto.GossipMessage{
		Nonce:   0,
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Channel: []byte(channel),
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{
				Payload: &proto.Payload{
					Data:   []byte{},
					SeqNum: seqnum,
				},
			},
		},
	}).NoopSign()
	return sMsg
}

func simulatePullPhase(gc GossipChannel, t *testing.T, wg *sync.WaitGroup, mutator msgMutator, seqs ...uint64) func(args mock.Arguments) {
	return simulatePullPhaseWithVariableDigest(gc, t, wg, mutator, [][]byte{[]byte("10"), []byte("11")}, []string{"10", "11"}, seqs...)
}

func simulatePullPhaseWithVariableDigest(gc GossipChannel, t *testing.T, wg *sync.WaitGroup, mutator msgMutator, proposedDigestSeqs [][]byte, resultDigestSeqs []string, seqs ...uint64) func(args mock.Arguments) {
	var l sync.Mutex
	var sentHello bool
	var sentReq bool
	return func(args mock.Arguments) {
		msg := args.Get(0).(*proto.SignedGossipMessage)
		l.Lock()
		defer l.Unlock()
		if msg.IsHelloMsg() && !sentHello {
			sentHello = true
//Simulate a digest message an imaginary peer responds to the hello message sent
			sMsg, _ := (&proto.GossipMessage{
				Tag:     proto.GossipMessage_CHAN_AND_ORG,
				Channel: []byte(channelA),
				Content: &proto.GossipMessage_DataDig{
					DataDig: &proto.DataDigest{
						MsgType: proto.PullMsgType_BLOCK_MSG,
						Digests: proposedDigestSeqs,
						Nonce:   msg.GetHello().Nonce,
					},
				},
			}).NoopSign()
			digestMsg := &receivedMsg{
				PKIID: pkiIDInOrg1,
				msg:   sMsg,
			}
			go gc.HandleMessage(digestMsg)
		}
		if msg.IsDataReq() && !sentReq {
			sentReq = true
			dataReq := msg.GetDataReq()
			for _, expectedDigest := range util.StringsToBytes(resultDigestSeqs) {
				assert.Contains(t, dataReq.Digests, expectedDigest)
			}
			assert.Equal(t, len(resultDigestSeqs), len(dataReq.Digests))
//当我们发送数据请求时，模拟数据更新的响应
//来自收到请求的虚拟对等机
			dataUpdateMsg := new(receivedMsg)
			dataUpdateMsg.PKIID = pkiIDInOrg1
			dataUpdateMsg.msg = createDataUpdateMsg(dataReq.Nonce, seqs...)
			mutator(dataUpdateMsg.msg.GetDataUpdate().Data[0])
			gc.HandleMessage(dataUpdateMsg)
			wg.Done()
		}
	}
}

func sequence(start uint64, end uint64) []uint64 {
	sequence := make([]uint64, end-start+1)
	i := 0
	for n := start; n <= end; n++ {
		sequence[i] = n
		i++
	}
	return sequence

}

func TestChangesInPeers(t *testing.T) {
//测试更改在通道中脱机和联机对等之后进行对等跟踪
//场景1：没有新的对等方-对等方列表保持不变
//场景2：添加了新的对等机-老对等机保持不变
//场景3:添加了新对等机-删除了一个旧对等机
//场景4:添加了新对等-一个旧对等尚未更改
//场景5：添加了新的对等机，以前没有其他对等机
//场景6：删除了一个对等点，未添加新对等点
//场景7：删除一个对等机，所有其他对等机保持不变
	t.Parallel()
	type testCase struct {
		name                     string
		oldMembers               map[string]struct{}
		newMembers               map[string]struct{}
		expected                 []string
		entryInChannel           func(chan string)
		expectedReportInvocation bool
	}
	var cases = []testCase{
		{
			name:       "noChanges",
			oldMembers: map[string]struct{}{"pkiID11": {}, "pkiID22": {}, "pkiID33": {}},
			newMembers: map[string]struct{}{"pkiID11": {}, "pkiID22": {}, "pkiID33": {}},
			expected:   []string{""},
			entryInChannel: func(chStr chan string) {
				chStr <- ""
			},
			expectedReportInvocation: false,
		},
		{
			name:       "newPeerWasAdded",
			oldMembers: map[string]struct{}{"pkiID1": {}},
			newMembers: map[string]struct{}{"pkiID1": {}, "pkiID3": {}},
			expected: []string{"Membership view has changed. peers went online: [[pkiID3]], current view: [[pkiID1] [pkiID3]]",
				"Membership view has changed. peers went online: [[pkiID3]], current view: [[pkiID3] [pkiID1]]"},
			entryInChannel:           func(chStr chan string) {},
			expectedReportInvocation: true,
		},

		{
			name:       "newPeerAddedOldPeerDeleted",
			oldMembers: map[string]struct{}{"pkiID1": {}, "pkiID2": {}},
			newMembers: map[string]struct{}{"pkiID1": {}, "pkiID3": {}},
			expected: []string{"Membership view has changed. peers went offline: [[pkiID2]], peers went online: [[pkiID3]], current view: [[pkiID1] [pkiID3]]",
				"Membership view has changed. peers went offline: [[pkiID2]], peers went online: [[pkiID3]], current view: [[pkiID3] [pkiID1]]"},
			entryInChannel:           func(chStr chan string) {},
			expectedReportInvocation: true,
		},
		{
			name:       "newPeersAddedOldPeerStayed",
			oldMembers: map[string]struct{}{"pkiID1": {}},
			newMembers: map[string]struct{}{"pkiID2": {}},
			expected: []string{"Membership view has changed. peers went offline: [[pkiID1]], peers went online: [[pkiID2]], current view: [[pkiID2]]",
				"Membership view has changed. peers went offline: [[pkiID1]], peers went online: [[pkiID2]], current view: [[pkiID2]]"},
			entryInChannel:           func(chStr chan string) {},
			expectedReportInvocation: true,
		},
		{
			name:                     "newPeersAddedNoOldPeers",
			oldMembers:               map[string]struct{}{},
			newMembers:               map[string]struct{}{"pkiID1": {}},
			expected:                 []string{"Membership view has changed. peers went online: [[pkiID1]], current view: [[pkiID1]]"},
			entryInChannel:           func(chStr chan string) {},
			expectedReportInvocation: true,
		},
		{
			name:                     "PeerWasDeletedNoNewPeers",
			oldMembers:               map[string]struct{}{"pkiID1": {}},
			newMembers:               map[string]struct{}{},
			expected:                 []string{"Membership view has changed. peers went offline: [[pkiID1]], current view: []"},
			entryInChannel:           func(chStr chan string) {},
			expectedReportInvocation: true,
		},
		{
			name:       "onePeerWasDeletedRestStayed",
			oldMembers: map[string]struct{}{"pkiID01": {}, "pkiID02": {}, "pkiID03": {}},
			newMembers: map[string]struct{}{"pkiID01": {}, "pkiID02": {}},
			expected: []string{"Membership view has changed. peers went offline: [[pkiID03]], current view: [[pkiID01] [pkiID02]]",
				"Membership view has changed. peers went offline: [[pkiID03]], current view: [[pkiID02] [pkiID01]]"},
			entryInChannel:           func(chStr chan string) {},
			expectedReportInvocation: true,
		},
	}
	invokedReport := false
//保存报告输出的通道
	chForString := make(chan string, 1)
	defer func() {
		invokedReport = false
	}()
	funcLogger := func(a ...interface{}) {
		invokedReport = true
		chForString <- fmt.Sprint(a...)
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			tickChan := make(chan time.Time)
			i := 0

			buildMembers := func(rangeMembers map[string]struct{}) []discovery.NetworkMember {
				var members []discovery.NetworkMember
				for peerID := range rangeMembers {
					peer := discovery.NetworkMember{
						Endpoint:         peerID,
						InternalEndpoint: peerID,
					}
					peer.PKIid = common.PKIidType(peerID)
					members = append(members, peer)
				}
				return members
			}

			stopChan := make(chan struct{}, 1)

			getListOfPeers := func() []discovery.NetworkMember {
				var members []discovery.NetworkMember
				if i == 1 {
					members = buildMembers(test.newMembers)
					i++
					stopChan <- struct{}{}
				}
				if i == 0 {
					members = buildMembers(test.oldMembers)
					i++
				}
				return members
			}

			mt := &membershipTracker{
				getPeersToTrack: getListOfPeers,
				report:          funcLogger,
				stopChan:        stopChan,
				tickerChannel:   tickChan,
			}

			wgMT := sync.WaitGroup{}
			wgMT.Add(1)
			go func() {
				mt.trackMembershipChanges()
				wgMT.Done()
			}()
			defer wgMT.Wait()

			tickChan <- time.Time{}
			if test.name == "noChanges" {
				test.entryInChannel(chForString)
			}
			select {
			case actual, _ := <-chForString:
				assert.Contains(t, test.expected, actual)
				assert.Equal(t, test.expectedReportInvocation, invokedReport)
			}
		})
	}
}

func TestMembershiptrackerStopWhenGCStops(t *testing.T) {
//当八卦频道启动时调用MembershipTracker
//会员追踪，只要八卦频道没有停止，就打印了正确的东西。
//停止八卦频道后，MembershipTracker不打印
//停止八卦频道后，MembershipTracker停止运行
	t.Parallel()
	checkIfStopedChan := make(chan struct{}, 1)
	cs := &cryptoService{}
	pkiID1 := common.PKIidType("1")
	adapter := new(gossipAdapterMock)

	jcm := &joinChanMsg{}

	peerA := discovery.NetworkMember{
		PKIid:            pkiIDInOrg1,
		Endpoint:         "a",
		InternalEndpoint: "a",
	}
	peerB := discovery.NetworkMember{
		PKIid:            pkiIDinOrg2,
		Endpoint:         "b",
		InternalEndpoint: "b",
	}

	conf := conf
	conf.RequestStateInfoInterval = time.Hour
	conf.PullInterval = time.Hour
	conf.TimeForMembershipTracker = time.Millisecond * 10

	adapter.On("Gossip", mock.Anything)
	adapter.On("Forward", mock.Anything)
	adapter.On("Send", mock.Anything, mock.Anything)
	adapter.On("DeMultiplex", mock.Anything)
	adapter.On("GetConf").Return(conf)
	adapter.On("GetOrgOfPeer", pkiIDInOrg1).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", pkiIDinOrg2).Return(orgInChannelA)
	adapter.On("GetOrgOfPeer", mock.Anything).Return(api.OrgIdentityType(nil))

	waitForHandleMsgChan := make(chan struct{})

	adapter.On("GetMembership").Return([]discovery.NetworkMember{peerA}).Run(func(args mock.Arguments) {
		waitForHandleMsgChan <- struct{}{}
	}).Once()

	gc := NewGossipChannel(pkiID1, orgInChannelA, cs, channelA, adapter, jcm)

	gc.HandleMessage(&receivedMsg{PKIID: pkiIDInOrg1, msg: createStateInfoMsg(1, pkiIDInOrg1, channelA)})
	gc.HandleMessage(&receivedMsg{PKIID: pkiIDinOrg2, msg: createStateInfoMsg(1, pkiIDinOrg2, channelA)})
	<-waitForHandleMsgChan

	adapter.On("GetMembership").Return([]discovery.NetworkMember{peerB}).Run(func(args mock.Arguments) {
		gc.(*gossipChannel).Stop()
	}).Once()

	flogging.Global.ActivateSpec("info")
	gc.(*gossipChannel).logger = gc.(*gossipChannel).logger.(*flogging.FabricLogger).WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if !strings.Contains(entry.Message, "Membership view has changed. peers went offline:  [[a]] , peers went online:  [[b]] , current view:  [[b]]") {
			return nil
		}
		checkIfStopedChan <- struct{}{}
		return nil
	}))

	<-checkIfStopedChan

	time.Sleep(conf.TimeForMembershipTracker * 2)
	adapter.AssertNumberOfCalls(t, "GetMembership", 2)

}
