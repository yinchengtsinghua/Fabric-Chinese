
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


package gossip

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip/algo"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
)

var timeout = time.Second * time.Duration(180)

var testWG = sync.WaitGroup{}

var tests = []func(t *testing.T){
	TestPull,
	TestConnectToAnchorPeers,
	TestMembership,
	TestDissemination,
	TestMembershipConvergence,
	TestMembershipRequestSpoofing,
	TestDataLeakage,
	TestLeaveChannel,
//测试分发所有2所有：，
	TestIdentityExpiration,
	TestSendByCriteria,
	TestMultipleOrgEndpointLeakage,
	TestConfidentiality,
	TestAnchorPeer,
	TestBootstrapPeerMisConfiguration,
	TestNoMessagesSelfLoop,
}

func init() {
	util.SetupTestLogging()
	rand.Seed(int64(time.Now().Second()))
	aliveTimeInterval := time.Duration(1000) * time.Millisecond
	discovery.SetAliveTimeInterval(aliveTimeInterval)
	discovery.SetAliveExpirationCheckInterval(aliveTimeInterval)
	discovery.SetAliveExpirationTimeout(aliveTimeInterval * 10)
	discovery.SetReconnectInterval(aliveTimeInterval)
	discovery.SetMaxConnAttempts(5)
	for range tests {
		testWG.Add(1)
	}
	factory.InitFactories(nil)
}

var expirationTimes map[string]time.Time = map[string]time.Time{}

var orgInChannelA = api.OrgIdentityType("ORG1")

func acceptData(m interface{}) bool {
	if dataMsg := m.(*proto.GossipMessage).GetDataMsg(); dataMsg != nil {
		return true
	}
	return false
}

func acceptLeadershp(message interface{}) bool {
	validMsg := message.(*proto.GossipMessage).Tag == proto.GossipMessage_CHAN_AND_ORG &&
		message.(*proto.GossipMessage).IsLeadershipMsg()

	return validMsg
}

type joinChanMsg struct {
	members2AnchorPeers map[string][]api.AnchorPeer
}

//sequence number返回此joinchanmsg块的序列号
//来源于
func (*joinChanMsg) SequenceNumber() uint64 {
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

type naiveCryptoService struct {
	sync.RWMutex
	allowedPkiIDS map[string]struct{}
	revokedPkiIDS map[string]struct{}
}

func (cs *naiveCryptoService) OrgByPeerIdentity(api.PeerIdentityType) api.OrgIdentityType {
	return nil
}

func (*naiveCryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	if exp, exists := expirationTimes[string(peerIdentity)]; exists {
		return exp, nil
	}
	return time.Now().Add(time.Hour), nil
}

type orgCryptoService struct {
}

//orgByPeerIdentity返回orgIdentityType
//给定对等身份的
func (*orgCryptoService) OrgByPeerIdentity(identity api.PeerIdentityType) api.OrgIdentityType {
	return orgInChannelA
}

//verify验证joinchanmessage，成功时返回nil，
//失败时出错
func (*orgCryptoService) Verify(joinChanMsg api.JoinChannelMessage) error {
	return nil
}

//verifybychannel验证上下文中消息上对等方的签名
//特定通道的
func (cs *naiveCryptoService) VerifyByChannel(_ common.ChainID, identity api.PeerIdentityType, _, _ []byte) error {
	if cs.allowedPkiIDS == nil {
		return nil
	}
	if _, allowed := cs.allowedPkiIDS[string(identity)]; allowed {
		return nil
	}
	return errors.New("Forbidden")
}

func (cs *naiveCryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	cs.RLock()
	defer cs.RUnlock()
	if cs.revokedPkiIDS == nil {
		return nil
	}
	if _, revoked := cs.revokedPkiIDS[string(cs.GetPKIidOfCert(peerIdentity))]; revoked {
		return errors.New("revoked")
	}
	return nil
}

//getpkiidofcert返回对等身份的pki-id
func (*naiveCryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return common.PKIidType(peerIdentity)
}

//verifyblock如果块被正确签名，则返回nil，
//else返回错误
func (*naiveCryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

//用该对等方的签名密钥和输出对消息进行签名
//如果没有出现错误，则返回签名。
func (*naiveCryptoService) Sign(msg []byte) ([]byte, error) {
	sig := make([]byte, len(msg))
	copy(sig, msg)
	return sig, nil
}

//验证检查签名是否是对等验证密钥下消息的有效签名。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peercert为nil，则根据该对等方的验证密钥验证签名。
func (*naiveCryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	equal := bytes.Equal(signature, message)
	if !equal {
		return fmt.Errorf("Wrong signature:%v, %v", signature, message)
	}
	return nil
}

func (cs *naiveCryptoService) revoke(pkiID common.PKIidType) {
	cs.Lock()
	defer cs.Unlock()
	if cs.revokedPkiIDS == nil {
		cs.revokedPkiIDS = map[string]struct{}{}
	}
	cs.revokedPkiIDS[string(pkiID)] = struct{}{}
}

func bootPeers(portPrefix int, ids ...int) []string {
	peers := []string{}
	for _, id := range ids {
		peers = append(peers, fmt.Sprintf("localhost:%d", id+portPrefix))
	}
	return peers
}

func newGossipInstanceWithCustomMCS(portPrefix int, id int, maxMsgCount int, mcs api.MessageCryptoService, boot ...int) Gossip {
	port := id + portPrefix
	conf := &Config{
		BindPort:                   port,
		BootstrapPeers:             bootPeers(portPrefix, boot...),
		ID:                         fmt.Sprintf("p%d", id),
		MaxBlockCountToStore:       maxMsgCount,
		MaxPropagationBurstLatency: time.Duration(500) * time.Millisecond,
		MaxPropagationBurstSize:    20,
		PropagateIterations:        1,
		PropagatePeerNum:           3,
		PullInterval:               time.Duration(4) * time.Second,
		PullPeerNum:                5,
		InternalEndpoint:           fmt.Sprintf("localhost:%d", port),
		ExternalEndpoint:           fmt.Sprintf("1.2.3.4:%d", port),
		PublishCertPeriod:          time.Duration(4) * time.Second,
		PublishStateInfoInterval:   time.Duration(1) * time.Second,
		RequestStateInfoInterval:   time.Duration(1) * time.Second,
		TimeForMembershipTracker:   5 * time.Second,
	}
	selfID := api.PeerIdentityType(conf.InternalEndpoint)
	g := NewGossipServiceWithServer(conf, &orgCryptoService{}, mcs,
		selfID, nil)

	return g
}

func newGossipInstance(portPrefix int, id int, maxMsgCount int, boot ...int) Gossip {
	return newGossipInstanceWithCustomMCS(portPrefix, id, maxMsgCount, &naiveCryptoService{}, boot...)
}

func newGossipInstanceWithOnlyPull(portPrefix int, id int, maxMsgCount int, boot ...int) Gossip {
	port := id + portPrefix
	conf := &Config{
		BindPort:                   port,
		BootstrapPeers:             bootPeers(portPrefix, boot...),
		ID:                         fmt.Sprintf("p%d", id),
		MaxBlockCountToStore:       maxMsgCount,
		MaxPropagationBurstLatency: time.Duration(1000) * time.Millisecond,
		MaxPropagationBurstSize:    10,
		PropagateIterations:        0,
		PropagatePeerNum:           0,
		PullInterval:               time.Duration(1000) * time.Millisecond,
		PullPeerNum:                20,
		InternalEndpoint:           fmt.Sprintf("localhost:%d", port),
		ExternalEndpoint:           fmt.Sprintf("1.2.3.4:%d", port),
		PublishCertPeriod:          time.Duration(0) * time.Second,
		PublishStateInfoInterval:   time.Duration(1) * time.Second,
		RequestStateInfoInterval:   time.Duration(1) * time.Second,
		TimeForMembershipTracker:   5 * time.Second,
	}

	cryptoService := &naiveCryptoService{}
	selfID := api.PeerIdentityType(conf.InternalEndpoint)
	g := NewGossipServiceWithServer(conf, &orgCryptoService{}, cryptoService,
		selfID, nil)
	return g
}

func TestLeaveChannel(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
	portPrefix := 4500
//场景：在一个通道中有3个对等点，让其中一个离开它。
//确保对等端离开通道时不识别另一个对等端

	p0 := newGossipInstance(portPrefix, 0, 100, 2)
	p0.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	p0.UpdateLedgerHeight(1, common.ChainID("A"))
	defer p0.Stop()

	p1 := newGossipInstance(portPrefix, 1, 100, 0)
	p1.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	p1.UpdateLedgerHeight(1, common.ChainID("A"))
	defer p1.Stop()

	p2 := newGossipInstance(portPrefix, 2, 100, 1)
	p2.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	p2.UpdateLedgerHeight(1, common.ChainID("A"))
	defer p2.Stop()

	countMembership := func(g Gossip, expected int) func() bool {
		return func() bool {
			peers := g.PeersOfChannel(common.ChainID("A"))
			return len(peers) == expected
		}
	}

//等到每个人都在频道里看到对方
	waitUntilOrFail(t, countMembership(p0, 2))
	waitUntilOrFail(t, countMembership(p1, 2))
	waitUntilOrFail(t, countMembership(p2, 2))

//现在p2离开频道
	p2.LeaveChan(common.ChainID("A"))

//确保相应地调整渠道成员资格
	waitUntilOrFail(t, countMembership(p0, 1))
	waitUntilOrFail(t, countMembership(p1, 1))
	waitUntilOrFail(t, countMembership(p2, 0))

}

func TestPull(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
	portPrefix := 5610
	t1 := time.Now()
//场景：关闭转发，只使用基于拉的八卦。
//第一阶段：确保所有节点的完整成员身份视图
//第二阶段：传播10条消息，确保所有节点都收到

	shortenedWaitTime := time.Duration(200) * time.Millisecond
	algo.SetDigestWaitTime(shortenedWaitTime)
	algo.SetRequestWaitTime(shortenedWaitTime)
	algo.SetResponseWaitTime(shortenedWaitTime)

	defer func() {
		algo.SetDigestWaitTime(time.Duration(1) * time.Second)
		algo.SetRequestWaitTime(time.Duration(1) * time.Second)
		algo.SetResponseWaitTime(time.Duration(2) * time.Second)
	}()

	stopped := int32(0)
	go waitForTestCompletion(&stopped, t)

	n := 5
	msgsCount2Send := 10

	peers := make([]Gossip, n)
	wg := sync.WaitGroup{}
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func(i int) {
			defer wg.Done()
			pI := newGossipInstanceWithOnlyPull(portPrefix, i, 100, 0)
			pI.JoinChan(&joinChanMsg{}, common.ChainID("A"))
			pI.UpdateLedgerHeight(1, common.ChainID("A"))
			peers[i-1] = pI
		}(i)
	}
	wg.Wait()

	time.Sleep(time.Second)

	boot := newGossipInstanceWithOnlyPull(portPrefix, 0, 100)
	boot.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	boot.UpdateLedgerHeight(1, common.ChainID("A"))

	knowAll := func() bool {
		for i := 1; i <= n; i++ {
			neighborCount := len(peers[i-1].Peers())
			if n != neighborCount {
				return false
			}
		}
		return true
	}

	receivedMessages := make([]int, n)
	wg = sync.WaitGroup{}
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func(i int) {
			acceptChan, _ := peers[i-1].Accept(acceptData, false)
			go func(index int, ch <-chan *proto.GossipMessage) {
				defer wg.Done()
				for j := 0; j < msgsCount2Send; j++ {
					<-ch
					receivedMessages[index]++
				}
			}(i-1, acceptChan)
		}(i)
	}

	for i := 1; i <= msgsCount2Send; i++ {
		boot.Gossip(createDataMsg(uint64(i), []byte{}, common.ChainID("A")))
	}

	waitUntilOrFail(t, knowAll)
	waitUntilOrFailBlocking(t, wg.Wait)

	receivedAll := func() bool {
		for i := 0; i < n; i++ {
			if msgsCount2Send != receivedMessages[i] {
				return false
			}
		}
		return true
	}
	waitUntilOrFail(t, receivedAll)

	stop := func() {
		stopPeers(append(peers, boot))
	}

	waitUntilOrFailBlocking(t, stop)

	t.Log("Took", time.Since(t1))
	atomic.StoreInt32(&stopped, int32(1))
	fmt.Println("<<<TestPull>>>")
}

func TestConnectToAnchorPeers(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
//场景：生成10个对等点，并让它们加入一个通道
//三个尚未存在的锚定对等。
//等待5秒，然后从3中生成一个随机锚点。
//确保所有对等方在通道中成功地看到对方

	portPrefix := 8610
//场景：生成5个对等，并使每个对等连接到
//另2个使用连接通道。
	stopped := int32(0)
	go waitForTestCompletion(&stopped, t)
	n := 10
	anchorPeercount := 3

	jcm := &joinChanMsg{members2AnchorPeers: map[string][]api.AnchorPeer{string(orgInChannelA): {}}}
	for i := 0; i < anchorPeercount; i++ {
		ap := api.AnchorPeer{
			Port: portPrefix + i,
			Host: "localhost",
		}
		jcm.members2AnchorPeers[string(orgInChannelA)] = append(jcm.members2AnchorPeers[string(orgInChannelA)], ap)
	}

	peers := make([]Gossip, n)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			peers[i] = newGossipInstance(portPrefix, i+anchorPeercount, 100)
			peers[i].JoinChan(jcm, common.ChainID("A"))
			peers[i].UpdateLedgerHeight(1, common.ChainID("A"))
			wg.Done()
		}(i)
	}

	waitUntilOrFailBlocking(t, wg.Wait)

	time.Sleep(time.Second * 5)

//现在开始一个随机锚点对等
	anchorPeer := newGossipInstance(portPrefix, rand.Intn(anchorPeercount), 100)
	anchorPeer.JoinChan(jcm, common.ChainID("A"))
	anchorPeer.UpdateLedgerHeight(1, common.ChainID("A"))

	defer anchorPeer.Stop()
	waitUntilOrFail(t, checkPeersMembership(t, peers, n))

	channelMembership := func() bool {
		for _, peer := range peers {
			if len(peer.PeersOfChannel(common.ChainID("A"))) != n {
				return false
			}
		}
		return true
	}
	waitUntilOrFail(t, channelMembership)

	stop := func() {
		stopPeers(peers)
	}
	waitUntilOrFailBlocking(t, stop)

	fmt.Println("<<<TestConnectToAnchorPeers>>>")
	atomic.StoreInt32(&stopped, int32(1))

}

func TestMembership(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
	portPrefix := 4610
	t1 := time.Now()
//场景：生成20个节点和一个引导节点，然后：
//1）检查除引导节点之外的所有节点的完整成员身份视图。
//2）更新最后一个对等点的元数据，确保其传播到所有对等点

	stopped := int32(0)
	go waitForTestCompletion(&stopped, t)

	n := 10
	var lastPeer = fmt.Sprintf("localhost:%d", n+portPrefix)
	boot := newGossipInstance(portPrefix, 0, 100)
	boot.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	boot.UpdateLedgerHeight(1, common.ChainID("A"))

	peers := make([]Gossip, n)
	wg := sync.WaitGroup{}
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func(i int) {
			defer wg.Done()
			pI := newGossipInstance(portPrefix, i, 100, 0)
			peers[i-1] = pI
			pI.JoinChan(&joinChanMsg{}, common.ChainID("A"))
			pI.UpdateLedgerHeight(1, common.ChainID("A"))
		}(i)
	}

	waitUntilOrFailBlocking(t, wg.Wait)
	t.Log("Peers started")

	seeAllNeighbors := func() bool {
		for i := 1; i <= n; i++ {
			neighborCount := len(peers[i-1].Peers())
			if neighborCount != n {
				return false
			}
		}
		return true
	}

	membershipEstablishTime := time.Now()
	waitUntilOrFail(t, seeAllNeighbors)
	t.Log("membership established in", time.Since(membershipEstablishTime))

	t.Log("Updating metadata...")
//更改最后一个节点中的元数据
	peers[len(peers)-1].UpdateMetadata([]byte("bla bla"))

	metaDataUpdated := func() bool {
		if !bytes.Equal([]byte("bla bla"), metadataOfPeer(boot.Peers(), lastPeer)) {
			return false
		}
		for i := 0; i < n-1; i++ {
			if !bytes.Equal([]byte("bla bla"), metadataOfPeer(peers[i].Peers(), lastPeer)) {
				return false
			}
		}
		return true
	}
	metadataDisseminationTime := time.Now()
	waitUntilOrFail(t, metaDataUpdated)
	fmt.Println("Metadata updated")
	t.Log("Metadata dissemination took", time.Since(metadataDisseminationTime))

	stop := func() {
		stopPeers(append(peers, boot))
	}

	stopTime := time.Now()
	waitUntilOrFailBlocking(t, stop)
	t.Log("Stop took", time.Since(stopTime))

	t.Log("Took", time.Since(t1))
	atomic.StoreInt32(&stopped, int32(1))
	fmt.Println("<<<TestMembership>>>")

}

func TestNoMessagesSelfLoop(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
	portPrefix := 17610

	boot := newGossipInstance(portPrefix, 0, 100)
	boot.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	boot.UpdateLedgerHeight(1, common.ChainID("A"))

	peer := newGossipInstance(portPrefix, 1, 100, 0)
	peer.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	peer.UpdateLedgerHeight(1, common.ChainID("A"))

//等待两个对等机连接
	waitUntilOrFail(t, checkPeersMembership(t, []Gossip{peer}, 1))
	_, commCh := boot.Accept(func(msg interface{}) bool {
		return msg.(proto.ReceivedMessage).GetGossipMessage().IsDataMsg()
	}, true)

	wg := sync.WaitGroup{}
	wg.Add(2)

//确保发送的对等端不能获取自己的对等端
//报文
	go func(ch <-chan proto.ReceivedMessage) {
		defer wg.Done()
		for {
			select {
			case msg := <-ch:
				{
					if msg.GetGossipMessage().IsDataMsg() {
						t.Fatal("Should not receive data message back, got", msg)
					}
				}
//等待2秒钟以确保我们不会
//回复W.H.P.
			case <-time.After(2 * time.Second):
				{
					return
				}
			}
		}
	}(commCh)

	peerCh, _ := peer.Accept(acceptData, false)

//确保收件人收到他的消息
	go func(ch <-chan *proto.GossipMessage) {
		defer wg.Done()
		<-ch
	}(peerCh)

	boot.Gossip(createDataMsg(uint64(2), []byte{}, common.ChainID("A")))
	waitUntilOrFailBlocking(t, wg.Wait)

	stop := func() {
		stopPeers([]Gossip{peer, boot})
	}

	waitUntilOrFailBlocking(t, stop)
}

func TestDissemination(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
	portPrefix := 3610
	t1 := time.Now()
//场景：20个节点和一个引导节点。
//引导节点发送10条消息，我们计算
//每个节点在几秒钟后收到10条消息

	stopped := int32(0)
	go waitForTestCompletion(&stopped, t)

	n := 10
	msgsCount2Send := 10
	boot := newGossipInstance(portPrefix, 0, 100)
	boot.JoinChan(&joinChanMsg{}, common.ChainID("A"))
	boot.UpdateLedgerHeight(1, common.ChainID("A"))
	boot.UpdateChaincodes([]*proto.Chaincode{{Name: "exampleCC", Version: "1.2"}}, common.ChainID("A"))

	peers := make([]Gossip, n)
	receivedMessages := make([]int, n)
	wg := sync.WaitGroup{}
	wg.Add(n)
	for i := 1; i <= n; i++ {
		pI := newGossipInstance(portPrefix, i, 100, 0)
		peers[i-1] = pI
		pI.JoinChan(&joinChanMsg{}, common.ChainID("A"))
		pI.UpdateLedgerHeight(1, common.ChainID("A"))
		pI.UpdateChaincodes([]*proto.Chaincode{{Name: "exampleCC", Version: "1.2"}}, common.ChainID("A"))
		acceptChan, _ := pI.Accept(acceptData, false)
		go func(index int, ch <-chan *proto.GossipMessage) {
			defer wg.Done()
			for j := 0; j < msgsCount2Send; j++ {
				<-ch
				receivedMessages[index]++
			}
		}(i-1, acceptChan)
//更改最后一个节点中的元数据
		if i == n {
			pI.UpdateLedgerHeight(2, common.ChainID("A"))
		}
	}
	var lastPeer = fmt.Sprintf("localhost:%d", n+portPrefix)
	metaDataUpdated := func() bool {
		if 2 != heightOfPeer(boot.PeersOfChannel(common.ChainID("A")), lastPeer) {
			return false
		}
		for i := 0; i < n-1; i++ {
			if 2 != heightOfPeer(peers[i].PeersOfChannel(common.ChainID("A")), lastPeer) {
				return false
			}
			for _, p := range peers[i].PeersOfChannel(common.ChainID("A")) {
				if len(p.Properties.Chaincodes) != 1 {
					return false
				}

				if !reflect.DeepEqual(p.Properties.Chaincodes, []*proto.Chaincode{{Name: "exampleCC", Version: "1.2"}}) {
					return false
				}
			}
		}
		return true
	}

	membershipTime := time.Now()
	waitUntilOrFail(t, checkPeersMembership(t, peers, n))
	t.Log("Membership establishment took", time.Since(membershipTime))

	for i := 2; i <= msgsCount2Send+1; i++ {
		boot.Gossip(createDataMsg(uint64(i), []byte{}, common.ChainID("A")))
	}

	t2 := time.Now()
	waitUntilOrFailBlocking(t, wg.Wait)
	t.Log("Block dissemination took", time.Since(t2))
	t2 = time.Now()
	waitUntilOrFail(t, metaDataUpdated)
	t.Log("Metadata dissemination took", time.Since(t2))

	for i := 0; i < n; i++ {
		assert.Equal(t, msgsCount2Send, receivedMessages[i])
	}

//发送领导信息
	receivedLeadershipMessages := make([]int, n)
	wgLeadership := sync.WaitGroup{}
	wgLeadership.Add(n)
	for i := 1; i <= n; i++ {
		leadershipChan, _ := peers[i-1].Accept(acceptLeadershp, false)
		go func(index int, ch <-chan *proto.GossipMessage) {
			defer wgLeadership.Done()
			msg := <-ch
			if bytes.Equal(msg.Channel, common.ChainID("A")) {
				receivedLeadershipMessages[index]++
			}
		}(i-1, leadershipChan)
	}

	seqNum := 0
	incTime := uint64(time.Now().UnixNano())
	t3 := time.Now()

	leadershipMsg := createLeadershipMsg(true, common.ChainID("A"), incTime, uint64(seqNum), boot.(*gossipServiceImpl).conf.InternalEndpoint, boot.(*gossipServiceImpl).comm.GetPKIid())
	boot.Gossip(leadershipMsg)

	waitUntilOrFailBlocking(t, wgLeadership.Wait)
	t.Log("Leadership message dissemination took", time.Since(t3))

	for i := 0; i < n; i++ {
		assert.Equal(t, 1, receivedLeadershipMessages[i])
	}

	t.Log("Stopping peers")

	stop := func() {
		stopPeers(append(peers, boot))
	}

	stopTime := time.Now()
	waitUntilOrFailBlocking(t, stop)
	t.Log("Stop took", time.Since(stopTime))
	t.Log("Took", time.Since(t1))
	atomic.StoreInt32(&stopped, int32(1))
	fmt.Println("<<<TestDissemination>>>")
}

func TestMembershipConvergence(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
	portPrefix := 2610
//场景：生成12个节点和3个引导对等节点
//但是将每个节点分配给它的引导对等组模块3。
//然后：
//1）检查所有组只知道视图中的自己，而不知道其他组。
//2）打开一个将连接到所有引导对等机的节点。
//3）等待几秒钟，检查所有视图是否聚合为一个视图。
//4）杀死最后一个节点，等待一段时间，然后：
//4）a）确保所有节点都认为它是死的
//4）b）确保所有节点仍然互相了解

	t1 := time.Now()

	stopped := int32(0)
	go waitForTestCompletion(&stopped, t)
	boot0 := newGossipInstance(portPrefix, 0, 100)
	boot1 := newGossipInstance(portPrefix, 1, 100)
	boot2 := newGossipInstance(portPrefix, 2, 100)

	peers := []Gossip{boot0, boot1, boot2}
//0：3、6、9、12_
//1：4、7、10、13_
//2：5、8、11、14_
	for i := 3; i < 15; i++ {
		pI := newGossipInstance(portPrefix, i, 100, i%3)
		peers = append(peers, pI)
	}

	waitUntilOrFail(t, checkPeersMembership(t, peers, 4))
	t.Log("Sets of peers connected successfully")

	connectorPeer := newGossipInstance(portPrefix, 15, 100, 0, 1, 2)
	connectorPeer.UpdateMetadata([]byte("Connector"))

	fullKnowledge := func() bool {
		for i := 0; i < 15; i++ {
			if 15 != len(peers[i].Peers()) {
				return false
			}
			if "Connector" != string(metadataOfPeer(peers[i].Peers(), "localhost:2625")) {
				return false
			}
		}
		return true
	}

	waitUntilOrFail(t, fullKnowledge)

	t.Log("Stopping connector...")
	waitUntilOrFailBlocking(t, connectorPeer.Stop)
	t.Log("Stopped")
	time.Sleep(time.Duration(15) * time.Second)

	ensureForget := func() bool {
		for i := 0; i < 15; i++ {
			if 14 != len(peers[i].Peers()) {
				return false
			}
		}
		return true
	}

	waitUntilOrFail(t, ensureForget)

	connectorPeer = newGossipInstance(portPrefix, 15, 100)
	connectorPeer.UpdateMetadata([]byte("Connector2"))
	t.Log("Started connector")

	ensureResync := func() bool {
		for i := 0; i < 15; i++ {
			if 15 != len(peers[i].Peers()) {
				return false
			}
			if "Connector2" != string(metadataOfPeer(peers[i].Peers(), "localhost:2625")) {
				return false
			}
		}
		return true
	}

	waitUntilOrFail(t, ensureResync)

	waitUntilOrFailBlocking(t, connectorPeer.Stop)

	t.Log("Stopping peers")
	stop := func() {
		stopPeers(peers)
	}

	waitUntilOrFailBlocking(t, stop)
	atomic.StoreInt32(&stopped, int32(1))
	t.Log("Took", time.Since(t1))
	fmt.Println("<<<TestMembershipConvergence>>>")
}

func TestMembershipRequestSpoofing(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
//场景：g1、g2、g3是对等的，g2是恶意的，并且想要
//向g1发送成员请求时模拟g3。
//预期输出：g1应*不*响应g2，
//但是，g1在发送消息本身时应该响应g3。

	portPrefix := 2000
	g1 := newGossipInstance(portPrefix, 0, 100)
	g2 := newGossipInstance(portPrefix, 1, 100, 2)
	g3 := newGossipInstance(portPrefix, 2, 100, 1)
	defer g1.Stop()
	defer g2.Stop()
	defer g3.Stop()

//等待G2和G3互相了解
	waitUntilOrFail(t, checkPeersMembership(t, []Gossip{g2, g3}, 1))
//从P3获取活动消息
	_, aliveMsgChan := g2.Accept(func(o interface{}) bool {
		msg := o.(proto.ReceivedMessage).GetGossipMessage()
//确保我们收到一条关于G3的有效信息
		return msg.IsAliveMsg() && bytes.Equal(msg.GetAliveMsg().Membership.PkiId, []byte("localhost:2002"))
	}, true)
	aliveMsg := <-aliveMsgChan

//获取从g1到g2的消息通道
	_, g1ToG2 := g2.Accept(func(o interface{}) bool {
		connInfo := o.(proto.ReceivedMessage).GetConnectionInfo()
		return bytes.Equal([]byte("localhost:2000"), connInfo.ID)
	}, true)

//获取从g1到g3的消息通道
	_, g1ToG3 := g3.Accept(func(o interface{}) bool {
		connInfo := o.(proto.ReceivedMessage).GetConnectionInfo()
		return bytes.Equal([]byte("localhost:2000"), connInfo.ID)
	}, true)

//现在，创建成员请求消息
	memRequestSpoofFactory := func(aliveMsgEnv *proto.Envelope) *proto.SignedGossipMessage {
		sMsg, _ := (&proto.GossipMessage{
			Tag:   proto.GossipMessage_EMPTY,
			Nonce: uint64(0),
			Content: &proto.GossipMessage_MemReq{
				MemReq: &proto.MembershipRequest{
					SelfInformation: aliveMsgEnv,
					Known:           [][]byte{},
				},
			},
		}).NoopSign()
		return sMsg
	}
	spoofedMemReq := memRequestSpoofFactory(aliveMsg.GetSourceEnvelope())
	g2.Send(spoofedMemReq.GossipMessage, &comm.RemotePeer{Endpoint: "localhost:2000", PKIID: common.PKIidType("localhost:2000")})
	select {
	case <-time.After(time.Second):
		break
	case <-g1ToG2:
		assert.Fail(t, "Received response from g1 but shouldn't have")
	}

//现在从g3发送相同的消息到g1
	g3.Send(spoofedMemReq.GossipMessage, &comm.RemotePeer{Endpoint: "localhost:2000", PKIID: common.PKIidType("localhost:2000")})
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Didn't receive a message back from g1 on time")
	case <-g1ToG3:
		break
	}
}

func TestDataLeakage(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
	portPrefix := 1610
//场景：生成一些节点并让它们全部
//建立正式会员资格。
//然后，让一半在通道A中，另一半在通道B中。
//但是，要使每个通道的前3个
//有资格从他们所在的频道获取数据块。
//确保节点仅获取其通道的消息，如果
//有资格使用这些频道。

totalPeers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} //这一定是偶数而不是奇数
//Peer0和Peer5散布块
//只有1、2和6、7可以得到块。

	mcs := &naiveCryptoService{
		allowedPkiIDS: map[string]struct{}{
//通道A
			"localhost:1610": {},
			"localhost:1611": {},
			"localhost:1612": {},
//通道B
			"localhost:1615": {},
			"localhost:1616": {},
			"localhost:1617": {},
		},
	}

	stopped := int32(0)
	go waitForTestCompletion(&stopped, t)

	n := len(totalPeers)
	peers := make([]Gossip, n)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			totPeers := append([]int(nil), totalPeers[:i]...)
			bootPeers := append(totPeers, totalPeers[i+1:]...)
			peers[i] = newGossipInstanceWithCustomMCS(portPrefix, i, 100, mcs, bootPeers...)
			wg.Done()
		}(i)
	}

	waitUntilOrFailBlocking(t, wg.Wait)
	waitUntilOrFail(t, checkPeersMembership(t, peers, n-1))

	channels := []common.ChainID{common.ChainID("A"), common.ChainID("B")}

	height := uint64(1)

	for i, channel := range channels {
		for j := 0; j < (n / 2); j++ {
			instanceIndex := (n/2)*i + j
			peers[instanceIndex].JoinChan(&joinChanMsg{}, channel)
			if i != 0 {
				height = uint64(2)
			}
			peers[instanceIndex].UpdateLedgerHeight(height, channel)
			t.Log(instanceIndex, "joined", string(channel))
		}
	}

//等待所有对等端在每个通道视图中都有其他对等端
	seeChannelMetadata := func() bool {
		for i, channel := range channels {
			for j := 0; j < 3; j++ {
				instanceIndex := (n/2)*i + j
				if len(peers[instanceIndex].PeersOfChannel(channel)) < 2 {
					return false
				}
			}
		}
		return true
	}
	t1 := time.Now()
	waitUntilOrFail(t, seeChannelMetadata)

	t.Log("Metadata sync took", time.Since(t1))
	for i, channel := range channels {
		for j := 0; j < 3; j++ {
			instanceIndex := (n/2)*i + j
			assert.Len(t, peers[instanceIndex].PeersOfChannel(channel), 2)
			if i == 0 {
				assert.Equal(t, uint64(1), peers[instanceIndex].PeersOfChannel(channel)[0].Properties.LedgerHeight)
			} else {
				assert.Equal(t, uint64(2), peers[instanceIndex].PeersOfChannel(channel)[0].Properties.LedgerHeight)
			}
		}
	}

	gotMessages := func() {
		var wg sync.WaitGroup
		wg.Add(4)
		for i, channel := range channels {
			for j := 1; j < 3; j++ {
				instanceIndex := (n/2)*i + j
				go func(instanceIndex int, channel common.ChainID) {
					incMsgChan, _ := peers[instanceIndex].Accept(acceptData, false)
					msg := <-incMsgChan
					assert.Equal(t, []byte(channel), []byte(msg.Channel))
					wg.Done()
				}(instanceIndex, channel)
			}
		}
		wg.Wait()
	}

	t1 = time.Now()
	peers[0].Gossip(createDataMsg(2, []byte{}, channels[0]))
	peers[n/2].Gossip(createDataMsg(3, []byte{}, channels[1]))
	waitUntilOrFailBlocking(t, gotMessages)
	t.Log("Dissemination took", time.Since(t1))
	stop := func() {
		stopPeers(peers)
	}
	stopTime := time.Now()
	waitUntilOrFailBlocking(t, stop)
	t.Log("Stop took", time.Since(stopTime))
	atomic.StoreInt32(&stopped, int32(1))
	fmt.Println("<<<TestDataLeakage>>>")
}

func TestDisseminateAll2All(t *testing.T) {
//场景：生成一些节点，每个节点
//将块传播到所有节点。
//确保接收到所有块

	t.Skip()
	t.Parallel()
	portPrefix := 6610
	stopped := int32(0)
	go waitForTestCompletion(&stopped, t)

	totalPeers := []int{0, 1, 2, 3, 4, 5, 6}
	n := len(totalPeers)
	peers := make([]Gossip, n)
	wg := sync.WaitGroup{}

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			totPeers := append([]int(nil), totalPeers[:i]...)
			bootPeers := append(totPeers, totalPeers[i+1:]...)
			pI := newGossipInstance(portPrefix, i, 100, bootPeers...)
			pI.JoinChan(&joinChanMsg{}, common.ChainID("A"))
			pI.UpdateLedgerHeight(1, common.ChainID("A"))
			peers[i] = pI
			wg.Done()
		}(i)
	}
	wg.Wait()
	waitUntilOrFail(t, checkPeersMembership(t, peers, n-1))

	bMutex := sync.WaitGroup{}
	bMutex.Add(10 * n * (n - 1))

	wg = sync.WaitGroup{}
	wg.Add(n)

	reader := func(msgChan <-chan *proto.GossipMessage, i int) {
		wg.Done()
		for range msgChan {
			bMutex.Done()
		}
	}

	for i := 0; i < n; i++ {
		msgChan, _ := peers[i].Accept(acceptData, false)
		go reader(msgChan, i)
	}

	wg.Wait()

	for i := 0; i < n; i++ {
		go func(i int) {
			blockStartIndex := i * 10
			for j := 0; j < 10; j++ {
				blockSeq := uint64(j + blockStartIndex)
				peers[i].Gossip(createDataMsg(blockSeq, []byte{}, common.ChainID("A")))
			}
		}(i)
	}
	waitUntilOrFailBlocking(t, bMutex.Wait)

	stop := func() {
		stopPeers(peers)
	}
	waitUntilOrFailBlocking(t, stop)
	atomic.StoreInt32(&stopped, int32(1))
	fmt.Println("<<<TestDisseminateAll2All>>>")
	testWG.Done()
}

func TestSendByCriteria(t *testing.T) {
	t.Parallel()
	defer testWG.Done()

	portPrefix := 20000
	g1 := newGossipInstance(portPrefix, 0, 100)
	g2 := newGossipInstance(portPrefix, 1, 100, 0)
	g3 := newGossipInstance(portPrefix, 2, 100, 0)
	g4 := newGossipInstance(portPrefix, 3, 100, 0)
	peers := []Gossip{g1, g2, g3, g4}
	for _, p := range peers {
		p.JoinChan(&joinChanMsg{}, common.ChainID("A"))
		p.UpdateLedgerHeight(1, common.ChainID("A"))
	}
	defer stopPeers(peers)
	msg, _ := createDataMsg(1, []byte{}, common.ChainID("A")).NoopSign()

//我们发送时没有指定最大对等点，
//将其设置为零值，以及
//这是不允许的。
	criteria := SendCriteria{
		IsEligible: func(discovery.NetworkMember) bool {
			t.Fatal("Shouldn't have called, because when max peers is 0, the operation is a no-op")
			return false
		},
		Timeout: time.Second * 1,
		MinAck:  1,
	}
	assert.NoError(t, g1.SendByCriteria(msg, criteria))

//发送时不指定超时
	criteria = SendCriteria{
		MaxPeers: 100,
	}
	err := g1.SendByCriteria(msg, criteria)
	assert.Error(t, err)
	assert.Equal(t, "Timeout should be specified", err.Error())

//我们发送时没有指定最小确认阈值
	criteria.Timeout = time.Second * 3
	err = g1.SendByCriteria(msg, criteria)
//应该有效，因为minack为0（未指定）
	assert.NoError(t, err)

//我们不指定频道发送
	criteria.Channel = common.ChainID("B")
	err = g1.SendByCriteria(msg, criteria)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "but no such channel exists")

//我们从频道发送给同行，但我们期望得到10份确认。
//它应该立即返回，因为我们不知道大约10个同龄人，所以即使尝试也没有意义
	criteria.Channel = common.ChainID("A")
	criteria.MinAck = 10
	err = g1.SendByCriteria(msg, criteria)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Requested to send to at least 10 peers, but know only of")

//我们至少向3个对等方发送确认消息，而没有对等方确认消息。
//等到g1看到通道中的其他对等点
	waitUntilOrFail(t, func() bool {
		return len(g1.PeersOfChannel(common.ChainID("A"))) > 2
	})
	criteria.MinAck = 3
	err = g1.SendByCriteria(msg, criteria)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, err.Error(), "3")

//我们重试上面的测试，但这次对等方确认
//现在同行
	acceptDataMsgs := func(m interface{}) bool {
		return m.(proto.ReceivedMessage).GetGossipMessage().IsDataMsg()
	}
	_, ackChan2 := g2.Accept(acceptDataMsgs, true)
	_, ackChan3 := g3.Accept(acceptDataMsgs, true)
	_, ackChan4 := g4.Accept(acceptDataMsgs, true)
	ack := func(c <-chan proto.ReceivedMessage) {
		msg := <-c
		msg.Ack(nil)
	}

	go ack(ackChan2)
	go ack(ackChan3)
	go ack(ackChan4)
	err = g1.SendByCriteria(msg, criteria)
	assert.NoError(t, err)

//我们发送给3个对等方，但3个对等方中有2个承认错误
	nack := func(c <-chan proto.ReceivedMessage) {
		msg := <-c
		msg.Ack(fmt.Errorf("uh oh"))
	}
	go ack(ackChan2)
	go nack(ackChan3)
	go nack(ackChan4)
	err = g1.SendByCriteria(msg, criteria)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uh oh")

//我们试图发送到g2或g3，但两者都不会确认我们，所以我们会失败。
//但是-我们在这个测试中实际检查的是，我们根据
//筛选通过条件
	failOnAckRequest := func(c <-chan proto.ReceivedMessage, peerId int) {
		msg := <-c
		if msg == nil {
			return
		}
		t.Fatalf("%d got a message, but shouldn't have!", peerId)
	}
	g2Endpoint := fmt.Sprintf("localhost:%d", portPrefix+1)
	g3Endpoint := fmt.Sprintf("localhost:%d", portPrefix+2)
	criteria.IsEligible = func(nm discovery.NetworkMember) bool {
		return nm.InternalEndpoint == g2Endpoint || nm.InternalEndpoint == g3Endpoint
	}
	criteria.MinAck = 1
	go failOnAckRequest(ackChan4, 3)
	err = g1.SendByCriteria(msg, criteria)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, err.Error(), "2")
//最后，确认丢失的消息，以便为下一个测试清除
	ack(ackChan2)
	ack(ackChan3)

//我们可以发送给2个对等方，但我们会检查如果我们指定max criteria.max peers，
//此属性受到尊重-只有1个对等方接收到消息，而不是同时接收到消息和消息
	criteria.MaxPeers = 1
//如果收到消息，则调用f（）。
	waitForMessage := func(c <-chan proto.ReceivedMessage, f func()) {
		select {
		case msg := <-c:
			if msg == nil {
				return
			}
		case <-time.After(time.Second * 5):
			return
		}
		f()
	}
	var messagesSent uint32
	go waitForMessage(ackChan2, func() {
		atomic.AddUint32(&messagesSent, 1)
	})
	go waitForMessage(ackChan3, func() {
		atomic.AddUint32(&messagesSent, 1)
	})
	err = g1.SendByCriteria(msg, criteria)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
//检查发送了多少条消息。
//只应发送1个
	assert.Equal(t, uint32(1), atomic.LoadUint32(&messagesSent))
}

func TestIdentityExpiration(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
//场景：生成5个对等点，并使MessageCryptoService撤销前4个对等点中的一个。
//最后一个对等方的证书在几秒钟后过期。
//最后，其他的同龄人不应该与
//被吊销的对等机，因为他们认为它的身份已过期

//将最后一个对等点的过期时间设置为从现在起5秒
	expirationTimes["localhost:7004"] = time.Now().Add(time.Second * 5)

	portPrefix := 7000
	g1 := newGossipInstance(portPrefix, 0, 100)
	g2 := newGossipInstance(portPrefix, 1, 100, 0)
	g3 := newGossipInstance(portPrefix, 2, 100, 0)
	g4 := newGossipInstance(portPrefix, 3, 100, 0)
	g5 := newGossipInstance(portPrefix, 4, 100, 0)

	peers := []Gossip{g1, g2, g3, g4}

//使最后一个对等机从现在起5秒钟内被吊销
	time.AfterFunc(time.Second*5, func() {
		for _, p := range peers {
			p.(*gossipServiceImpl).mcs.(*naiveCryptoService).revoke(common.PKIidType("localhost:7004"))
		}
	})

	seeAllNeighbors := func() bool {
		for i := 0; i < 4; i++ {
			neighborCount := len(peers[i].Peers())
			if neighborCount != 3 {
				return false
			}
		}
		return true
	}
	waitUntilOrFail(t, seeAllNeighbors)
//现在撤销一些同行
	revokedPeerIndex := rand.Intn(4)
	revokedPkiID := common.PKIidType(fmt.Sprintf("localhost:%d", portPrefix+int(revokedPeerIndex)))
	for i, p := range peers {
		if i == revokedPeerIndex {
			continue
		}
		p.(*gossipServiceImpl).mcs.(*naiveCryptoService).revoke(revokedPkiID)
	}
//触发对其余对等机的配置更新
	for i := 0; i < 4; i++ {
		if i == revokedPeerIndex {
			continue
		}
		peers[i].SuspectPeers(func(_ api.PeerIdentityType) bool {
			return true
		})
	}
//确保没有人与被撤销的对等方对话
	ensureRevokedPeerIsIgnored := func() bool {
		for i := 0; i < 4; i++ {
			neighborCount := len(peers[i].Peers())
			expectedNeighborCount := 2
//如果是被吊销的对等机，或者是最后一个证书的对等机
//已经过期
			if i == revokedPeerIndex || i == 4 {
				expectedNeighborCount = 0
			}
			if neighborCount != expectedNeighborCount {
				fmt.Println("neighbor count of", i, "is", neighborCount)
				return false
			}
		}
		return true
	}
	waitUntilOrFail(t, ensureRevokedPeerIsIgnored)
	stopPeers(peers)
	g5.Stop()
}

func TestEndedGoroutines(t *testing.T) {
	t.Parallel()
	testWG.Wait()
	ensureGoroutineExit(t)
}

func createDataMsg(seqnum uint64, data []byte, channel common.ChainID) *proto.GossipMessage {
	return &proto.GossipMessage{
		Channel: []byte(channel),
		Nonce:   0,
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{
				Payload: &proto.Payload{
					Data:   data,
					SeqNum: seqnum,
				},
			},
		},
	}
}

func createLeadershipMsg(isDeclaration bool, channel common.ChainID, incTime uint64, seqNum uint64, endpoint string, pkiid []byte) *proto.GossipMessage {

	leadershipMsg := &proto.LeadershipMessage{
		IsDeclaration: isDeclaration,
		PkiId:         pkiid,
		Timestamp: &proto.PeerTime{
			IncNum: incTime,
			SeqNum: seqNum,
		},
	}

	msg := &proto.GossipMessage{
		Nonce:   0,
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Content: &proto.GossipMessage_LeadershipMsg{LeadershipMsg: leadershipMsg},
		Channel: channel,
	}
	return msg
}

type goroutinePredicate func(g goroutine) bool

var connectionLeak = func(g goroutine) bool {
	return searchInStackTrace("comm.(*connection).writeToStream", g.stack)
}

var connectionLeak2 = func(g goroutine) bool {
	return searchInStackTrace("comm.(*connection).readFromStream", g.stack)
}

var runTests = func(g goroutine) bool {
	return searchInStackTrace("testing.RunTests", g.stack)
}

var tRunner = func(g goroutine) bool {
	return searchInStackTrace("testing.tRunner", g.stack)
}

var waitForTestCompl = func(g goroutine) bool {
	return searchInStackTrace("waitForTestCompletion", g.stack)
}

var gossipTest = func(g goroutine) bool {
	return searchInStackTrace("gossip_test.go", g.stack)
}

var goExit = func(g goroutine) bool {
	return searchInStackTrace("runtime.goexit", g.stack)
}

var clientConn = func(g goroutine) bool {
	return searchInStackTrace("resetTransport", g.stack)
}

var resolver = func(g goroutine) bool {
	return searchInStackTrace("ccResolverWrapper", g.stack)
}

var balancer = func(g goroutine) bool {
	return searchInStackTrace("ccBalancerWrapper", g.stack)
}

var clientStream = func(g goroutine) bool {
	return searchInStackTrace("ClientStream", g.stack)
}

var testingg = func(g goroutine) bool {
	if len(g.stack) == 0 {
		return false
	}
	return strings.Index(g.stack[len(g.stack)-1], "testing.go") != -1
}

func anyOfPredicates(predicates ...goroutinePredicate) goroutinePredicate {
	return func(g goroutine) bool {
		for _, pred := range predicates {
			if pred(g) {
				return true
			}
		}
		return false
	}
}

func shouldNotBeRunningAtEnd(gr goroutine) bool {
	return !anyOfPredicates(
		runTests,
		goExit,
		testingg,
		waitForTestCompl,
		gossipTest,
		clientConn,
		connectionLeak,
		connectionLeak2,
		tRunner,
		resolver,
		balancer,
		clientStream)(gr)
}

func ensureGoroutineExit(t *testing.T) {
	for i := 0; i <= 20; i++ {
		time.Sleep(time.Second)
		allEnded := true
		for _, gr := range getGoRoutines() {
			if shouldNotBeRunningAtEnd(gr) {
				allEnded = false
			}

			if shouldNotBeRunningAtEnd(gr) && i == 20 {
				assert.Fail(t, "Goroutine(s) haven't ended:", fmt.Sprintf("%v", gr.stack))
				util.PrintStackTrace()
				break
			}
		}

		if allEnded {
			return
		}
	}
}

func metadataOfPeer(members []discovery.NetworkMember, endpoint string) []byte {
	for _, member := range members {
		if member.InternalEndpoint == endpoint {
			return member.Metadata
		}
	}
	return nil
}

func heightOfPeer(members []discovery.NetworkMember, endpoint string) int {
	for _, member := range members {
		if member.InternalEndpoint == endpoint {
			return int(member.Properties.LedgerHeight)
		}
	}
	return -1
}

func waitForTestCompletion(stopFlag *int32, t *testing.T) {
	time.Sleep(timeout)
	if atomic.LoadInt32(stopFlag) == int32(1) {
		return
	}
	util.PrintStackTrace()
	assert.Fail(t, "Didn't stop within a timely manner")
}

func stopPeers(peers []Gossip) {
	stoppingWg := sync.WaitGroup{}
	stoppingWg.Add(len(peers))
	for i, pI := range peers {
		go func(i int, p_i Gossip) {
			defer stoppingWg.Done()
			p_i.Stop()
		}(i, pI)
	}
	stoppingWg.Wait()
	time.Sleep(time.Second * time.Duration(2))
}

func getGoroutineRawText() string {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	return string(buf)
}

func getGoRoutines() []goroutine {
	goroutines := []goroutine{}
	s := getGoroutineRawText()
	a := strings.Split(s, "goroutine ")
	for _, s := range a {
		gr := strings.Split(s, "\n")
		idStr := bytes.TrimPrefix([]byte(gr[0]), []byte("goroutine "))
		i := strings.Index(string(idStr), " ")
		if i == -1 {
			continue
		}
		id, _ := strconv.ParseUint(string(string(idStr[:i])), 10, 64)
		stack := []string{}
		for i := 1; i < len(gr); i++ {
			if len([]byte(gr[i])) != 0 {
				stack = append(stack, gr[i])
			}
		}
		goroutines = append(goroutines, goroutine{id: id, stack: stack})
	}
	return goroutines
}

type goroutine struct {
	id    uint64
	stack []string
}

func waitUntilOrFail(t *testing.T, pred func() bool) {
	start := time.Now()
	limit := start.UnixNano() + timeout.Nanoseconds()
	for time.Now().UnixNano() < limit {
		if pred() {
			return
		}
		time.Sleep(timeout / 60)
	}
	util.PrintStackTrace()
	assert.Fail(t, "Timeout expired!")
}

func waitUntilOrFailBlocking(t *testing.T, f func()) {
	successChan := make(chan struct{}, 1)
	go func() {
		f()
		successChan <- struct{}{}
	}()
	select {
	case <-time.NewTimer(timeout).C:
		break
	case <-successChan:
		return
	}
	util.PrintStackTrace()
	assert.Fail(t, "Timeout expired!")
}

func searchInStackTrace(searchTerm string, stack []string) bool {
	for _, ste := range stack {
		if strings.Index(ste, searchTerm) != -1 {
			return true
		}
	}
	return false
}

func checkPeersMembership(t *testing.T, peers []Gossip, n int) func() bool {
	return func() bool {
		for _, peer := range peers {
			if len(peer.Peers()) != n {
				return false
			}
			for _, p := range peer.Peers() {
				assert.NotNil(t, p.InternalEndpoint)
				assert.NotEmpty(t, p.Endpoint)
			}
		}
		return true
	}
}
