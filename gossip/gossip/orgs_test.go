
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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
)

func init() {
	util.SetupTestLogging()
	aliveTimeInterval := time.Duration(1000) * time.Millisecond
	discovery.SetAliveTimeInterval(aliveTimeInterval)
	discovery.SetAliveExpirationCheckInterval(aliveTimeInterval)
	discovery.SetAliveExpirationTimeout(aliveTimeInterval * 10)
	discovery.SetReconnectInterval(aliveTimeInterval)
	factory.InitFactories(nil)
}

type configurableCryptoService struct {
	m map[string]api.OrgIdentityType
}

func (c *configurableCryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	return time.Now().Add(time.Hour), nil
}

func (c *configurableCryptoService) putInOrg(port int, org string) {
	identity := fmt.Sprintf("localhost:%d", port)
	c.m[identity] = api.OrgIdentityType(org)
}

//orgByPeerIdentity返回orgIdentityType
//给定对等身份的
func (c *configurableCryptoService) OrgByPeerIdentity(identity api.PeerIdentityType) api.OrgIdentityType {
	org := c.m[string(identity)]
	return org
}

//verifybychannel验证上下文中消息上对等方的签名
//特定通道的
func (c *configurableCryptoService) VerifyByChannel(_ common.ChainID, identity api.PeerIdentityType, _, _ []byte) error {
	return nil
}

func (*configurableCryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	return nil
}

//getpkiidofcert返回对等身份的pki-id
func (*configurableCryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return common.PKIidType(peerIdentity)
}

//verifyblock如果块被正确签名，则返回nil，
//else返回错误
func (*configurableCryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

//用该对等方的签名密钥和输出对消息进行签名
//如果没有出现错误，则返回签名。
func (*configurableCryptoService) Sign(msg []byte) ([]byte, error) {
	sig := make([]byte, len(msg))
	copy(sig, msg)
	return sig, nil
}

//验证检查签名是否是对等验证密钥下消息的有效签名。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peercert为nil，则根据该对等方的验证密钥验证签名。
func (*configurableCryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	equal := bytes.Equal(signature, message)
	if !equal {
		return fmt.Errorf("Wrong signature:%v, %v", signature, message)
	}
	return nil
}

func newGossipInstanceWithExternalEndpoint(portPrefix int, id int, mcs *configurableCryptoService, externalEndpoint string, boot ...int) Gossip {
	port := id + portPrefix
	conf := &Config{
		BindPort:                   port,
		BootstrapPeers:             bootPeers(portPrefix, boot...),
		ID:                         fmt.Sprintf("p%d", id),
		MaxBlockCountToStore:       100,
		MaxPropagationBurstLatency: time.Duration(500) * time.Millisecond,
		MaxPropagationBurstSize:    20,
		PropagateIterations:        1,
		PropagatePeerNum:           3,
		PullInterval:               time.Duration(2) * time.Second,
		PullPeerNum:                5,
		InternalEndpoint:           fmt.Sprintf("localhost:%d", port),
		ExternalEndpoint:           externalEndpoint,
		PublishCertPeriod:          time.Duration(4) * time.Second,
		PublishStateInfoInterval:   time.Duration(1) * time.Second,
		RequestStateInfoInterval:   time.Duration(1) * time.Second,
		TimeForMembershipTracker:   5 * time.Second,
	}
	selfID := api.PeerIdentityType(conf.InternalEndpoint)
	g := NewGossipServiceWithServer(conf, mcs, mcs, selfID,
		nil)

	return g
}

func TestMultipleOrgEndpointLeakage(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
//场景：创建2个组织，每个组织有5个对等方。
//第一个组织将有一个锚定对等组织，但第二个组织不会。
//每个组织的前两个对等体将有一个外部端点，其余的则没有。
//让所有同行加入这两个组织的渠道。
//确保会员资格稳定后：
//-唯一了解其他组织同行的同行是
//配置外部终结点。
//-具有外部终结点的对等方不知道内部终结点
//其他组织的同行。
	cs := &configurableCryptoService{m: make(map[string]api.OrgIdentityType)}
	portPrefix := 11610
	peersInOrg := 5
	orgA := "orgA"
	orgB := "orgB"
	orgs := []string{orgA, orgB}
	orgs2Peers := map[string][]Gossip{
		orgs[0]: {},
		orgs[1]: {},
	}
	expectedMembershipSize := map[string]int{}
	peers2Orgs := map[string]api.OrgIdentityType{}
	peersWithExternalEndpoints := make(map[string]struct{})

	shouldAKnowB := func(a common.PKIidType, b common.PKIidType) bool {
		orgOfPeerA := cs.OrgByPeerIdentity(api.PeerIdentityType(a))
		orgOfPeerB := cs.OrgByPeerIdentity(api.PeerIdentityType(b))
		_, aHasExternalEndpoint := peersWithExternalEndpoints[string(a)]
		_, bHasExternalEndpoint := peersWithExternalEndpoints[string(b)]
		bothHaveExternalEndpoints := aHasExternalEndpoint && bHasExternalEndpoint
		return bytes.Equal(orgOfPeerA, orgOfPeerB) || bothHaveExternalEndpoints
	}

	shouldKnowInternalEndpoint := func(a common.PKIidType, b common.PKIidType) bool {
		orgOfPeerA := cs.OrgByPeerIdentity(api.PeerIdentityType(a))
		orgOfPeerB := cs.OrgByPeerIdentity(api.PeerIdentityType(b))
		return bytes.Equal(orgOfPeerA, orgOfPeerB)
	}

	amountOfPeersShouldKnow := func(pkiID common.PKIidType) int {
		return expectedMembershipSize[string(pkiID)]
	}

	for orgIndex := 0; orgIndex < 2; orgIndex++ {
		for i := 0; i < peersInOrg; i++ {
			id := orgIndex*peersInOrg + i
			port := id + portPrefix
			org := orgs[orgIndex]
			endpoint := fmt.Sprintf("localhost:%d", port)
			peers2Orgs[endpoint] = api.OrgIdentityType(org)
			cs.putInOrg(port, org)
membershipSizeExpected := peersInOrg - 1 //组织中的所有其他人
			var peer Gossip
			var bootPeers []int
			if orgIndex == 0 {
				bootPeers = []int{0}
			}
			externalEndpoint := ""
if i < 2 { //每个组织的前2个对等方将有一个外部端点
				externalEndpoint = endpoint
				peersWithExternalEndpoints[externalEndpoint] = struct{}{}
membershipSizeExpected += 2 //应该知道另外两个组织的同事
			}
			expectedMembershipSize[endpoint] = membershipSizeExpected
			peer = newGossipInstanceWithExternalEndpoint(portPrefix, id, cs, externalEndpoint, bootPeers...)
			orgs2Peers[org] = append(orgs2Peers[org], peer)
		}
	}

	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			orgA: {
				{Host: "localhost", Port: 11611},
				{Host: "localhost", Port: 11616},
			},
			orgB: {
				{Host: "localhost", Port: 11615},
			},
		},
	}

	channel := common.ChainID("TEST")

	for _, peers := range orgs2Peers {
		for _, p := range peers {
			p.JoinChan(jcm, channel)
			p.UpdateLedgerHeight(1, channel)
		}
	}

	membershipCheck := func() bool {
		for _, peers := range orgs2Peers {
			for _, p := range peers {
				peerNetMember := p.(*gossipServiceImpl).selfNetworkMember()
				pkiID := peerNetMember.PKIid
				peersKnown := p.Peers()
				peersToKnow := amountOfPeersShouldKnow(pkiID)
				if peersToKnow != len(peersKnown) {
					t.Logf("peer %#v doesn't know the needed amount of peers, extected %#v, actual %#v", peerNetMember.Endpoint, peersToKnow, len(peersKnown))
					return false
				}
				for _, knownPeer := range peersKnown {
					if !shouldAKnowB(pkiID, knownPeer.PKIid) {
						assert.Fail(t, fmt.Sprintf("peer %#v doesn't know %#v", peerNetMember.Endpoint, knownPeer.Endpoint))
						return false
					}
					internalEndpointLen := len(knownPeer.InternalEndpoint)
					if shouldKnowInternalEndpoint(pkiID, knownPeer.PKIid) {
						if internalEndpointLen == 0 {
							t.Logf("peer: %v doesn't know internal endpoint of %v", peerNetMember.InternalEndpoint, string(knownPeer.PKIid))
							return false
						}
					} else {
						if internalEndpointLen != 0 {
							assert.Fail(t, fmt.Sprintf("peer: %v knows internal endpoint of %v (%#v)", peerNetMember.InternalEndpoint, string(knownPeer.PKIid), knownPeer.InternalEndpoint))
							return false
						}
					}
				}
			}
		}
		return true
	}

	waitUntilOrFail(t, membershipCheck)

	for _, peers := range orgs2Peers {
		for _, p := range peers {
			p.Stop()
		}
	}
}

func TestConfidentiality(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
//场景：创建4个组织：A、B、C、D，每个组织有3个对等组织。
//仅使前两个对等机具有外部终结点。
//另外，将对等端添加到以下通道：
//信道C0：ORGA，ORGB
//通道C1：ORGA，ORGC
//信道C2：orgb，orgc
//通道C3：orgb，orgd
//[a]-c0-[b]-c3-[d]
///
///
//C1 C2
///
///
//[C]
//为每个对等点订阅所有成员身份消息，
//如果消息发送到ORGX中的对等方，则测试失败，
//从组织Y中的一个对等点关于组织Z中的一个对等点而不是X，Y
//或者，如果除orgb之外的任何组织认识orgd中的对等组织（反之亦然）。

	portPrefix := 12610
	peersInOrg := 3
	externalEndpointsInOrg := 2

//组织机构：12610、12611、12612
//组织机构：12613、12614、12615
//组织机构：12616、12617、12618
//组织：12619、12620、12621_
	peersWithExternalEndpoints := map[string]struct{}{}

	orgs := []string{"A", "B", "C", "D"}
	channels := []string{"C0", "C1", "C2", "C3"}
	isOrgInChan := func(org string, channel string) bool {
		switch org {
		case "A":
			return channel == "C0" || channel == "C1"
		case "B":
			return channel == "C0" || channel == "C2" || channel == "C3"
		case "C":
			return channel == "C1" || channel == "C2"
		case "D":
			return channel == "C3"
		}

		return false
	}

//创建消息加密服务
	cs := &configurableCryptoService{m: make(map[string]api.OrgIdentityType)}
	for i, org := range orgs {
		for j := 0; j < peersInOrg; j++ {
			port := portPrefix + i*peersInOrg + j
			cs.putInOrg(port, org)
		}
	}

	var peers []Gossip
	orgs2Peers := map[string][]Gossip{
		"A": {},
		"B": {},
		"C": {},
		"D": {},
	}

	anchorPeersByOrg := map[string]api.AnchorPeer{}

	for i, org := range orgs {
		for j := 0; j < peersInOrg; j++ {
			id := i*peersInOrg + j
			port := id + portPrefix
			endpoint := fmt.Sprintf("localhost:%d", port)
			externalEndpoint := ""
if j < externalEndpointsInOrg { //每个组织的第一个对等体将有一个外部端点
				externalEndpoint = endpoint
				peersWithExternalEndpoints[string(endpoint)] = struct{}{}
			}
			peer := newGossipInstanceWithExternalEndpoint(portPrefix, id, cs, externalEndpoint)
			peers = append(peers, peer)
			orgs2Peers[org] = append(orgs2Peers[org], peer)
			t.Log(endpoint, "id:", id, "externalEndpoint:", externalEndpoint)
//组织的第一个对等点将用作锚定对等点
			if j == 0 {
				anchorPeersByOrg[org] = api.AnchorPeer{
					Host: "localhost",
					Port: port,
				}
			}
		}
	}

	msgs2Inspect := make(chan *msg, 3000)
	defer close(msgs2Inspect)
	go inspectMsgs(t, msgs2Inspect, cs, peersWithExternalEndpoints)
	finished := int32(0)
	var wg sync.WaitGroup

	msgSelector := func(o interface{}) bool {
		msg := o.(proto.ReceivedMessage).GetGossipMessage()
		identitiesPull := msg.IsPullMsg() && msg.GetPullMsgType() == proto.PullMsgType_IDENTITY_MSG
		return msg.IsAliveMsg() || msg.IsStateInfoMsg() || msg.IsStateInfoSnapshot() || msg.GetMemRes() != nil || identitiesPull
	}
//听取所有同行成员的信息，并将其转发到检查通道。
//如果发现违反保密规定，测试将失败。
	for _, p := range peers {
		wg.Add(1)
		_, msgs := p.Accept(msgSelector, true)
		peerNetMember := p.(*gossipServiceImpl).selfNetworkMember()
		targetORg := string(cs.OrgByPeerIdentity(api.PeerIdentityType(peerNetMember.InternalEndpoint)))
		go func(targetOrg string, msgs <-chan proto.ReceivedMessage) {
			defer wg.Done()
			for receivedMsg := range msgs {
				m := &msg{
					src:           string(cs.OrgByPeerIdentity(receivedMsg.GetConnectionInfo().Identity)),
					dst:           targetORg,
					GossipMessage: receivedMsg.GetGossipMessage().GossipMessage,
				}
				if atomic.LoadInt32(&finished) == int32(1) {
					return
				}
				msgs2Inspect <- m
			}
		}(targetORg, msgs)
	}

//现在，构造连接通道消息
	joinChanMsgsByChan := map[string]*joinChanMsg{}
	for _, ch := range channels {
		jcm := &joinChanMsg{members2AnchorPeers: map[string][]api.AnchorPeer{}}
		for _, org := range orgs {
			if isOrgInChan(org, ch) {
				jcm.members2AnchorPeers[org] = append(jcm.members2AnchorPeers[org], anchorPeersByOrg[org])
			}
		}
		joinChanMsgsByChan[ch] = jcm
	}

//接下来，让同龄人加入频道
	for org, peers := range orgs2Peers {
		for _, ch := range channels {
			if isOrgInChan(org, ch) {
				for _, p := range peers {
					p.JoinChan(joinChanMsgsByChan[ch], common.ChainID(ch))
					p.UpdateLedgerHeight(1, common.ChainID(ch))
					go func(p Gossip) {
						for i := 0; i < 5; i++ {
							time.Sleep(time.Second)
							p.UpdateLedgerHeight(1, common.ChainID(ch))
						}
					}(p)
				}
			}
		}
	}

//睡一会儿，让同龄人互相八卦。
	time.Sleep(time.Second * 7)

	assertMembership := func() bool {
		for _, org := range orgs {
			for i, p := range orgs2Peers[org] {
				members := p.Peers()
				expMemberSize := expectedMembershipSize(peersInOrg, externalEndpointsInOrg, org, i < externalEndpointsInOrg)
				peerNetMember := p.(*gossipServiceImpl).selfNetworkMember()
				membersCount := len(members)
				if membersCount < expMemberSize {
					return false
				}
//确保没有人知道太多
				assert.True(t, membersCount <= expMemberSize, "%s knows too much (%d > %d) peers: %v",
					membersCount, expMemberSize, peerNetMember.PKIid, members)
			}
		}
		return true
	}

	waitUntilOrFail(t, assertMembership)
	stopPeers(peers)
	wg.Wait()
	atomic.StoreInt32(&finished, int32(1))
}

func expectedMembershipSize(peersInOrg, externalEndpointsInOrg int, org string, hasExternalEndpoint bool) int {
//X<--皮尔辛诺格
//Y<--外部端点Sinorg
//（x+2y）[a]-c0-[b]--c3--[d]（x+y）
//（/（x+3y）
///
//C1 C2
///
///
//[C] x+2y

	m := map[string]func(x, y int) int{
		"A": func(x, y int) int {
			return x + 2*y
		},
		"B": func(x, y int) int {
			return x + 3*y
		},
		"C": func(x, y int) int {
			return x + 2*y
		},
		"D": func(x, y int) int {
			return x + y
		},
	}

//如果对等端没有外部端点，
//它不知道外国组织的同行有一个
	if !hasExternalEndpoint {
		externalEndpointsInOrg = 0
	}
//因为对等端本身不算数，所以减去1
	return m[org](peersInOrg, externalEndpointsInOrg) - 1
}

func extractOrgsFromMsg(msg *proto.GossipMessage, sec api.SecurityAdvisor) []string {
	if msg.IsAliveMsg() {
		return []string{string(sec.OrgByPeerIdentity(api.PeerIdentityType(msg.GetAliveMsg().Membership.PkiId)))}
	}

	orgs := map[string]struct{}{}

	if msg.IsPullMsg() {
		if msg.IsDigestMsg() || msg.IsDataReq() {
			var digests []string
			if msg.IsDigestMsg() {
				digests = util.BytesToStrings(msg.GetDataDig().Digests)
			} else {
				digests = util.BytesToStrings(msg.GetDataReq().Digests)
			}

			for _, dig := range digests {
				org := sec.OrgByPeerIdentity(api.PeerIdentityType(dig))
				orgs[string(org)] = struct{}{}
			}
		}

		if msg.IsDataUpdate() {
			for _, identityMsg := range msg.GetDataUpdate().Data {
				gMsg, _ := identityMsg.ToGossipMessage()
				id := string(gMsg.GetPeerIdentity().Cert)
				org := sec.OrgByPeerIdentity(api.PeerIdentityType(id))
				orgs[string(org)] = struct{}{}
			}
		}
	}

	if msg.GetMemRes() != nil {
		alive := msg.GetMemRes().Alive
		dead := msg.GetMemRes().Dead
		for _, envp := range append(alive, dead...) {
			msg, _ := envp.ToGossipMessage()
			orgs[string(sec.OrgByPeerIdentity(api.PeerIdentityType(msg.GetAliveMsg().Membership.PkiId)))] = struct{}{}
		}
	}

	res := []string{}
	for org := range orgs {
		res = append(res, org)
	}
	return res
}

func inspectMsgs(t *testing.T, msgChan chan *msg, sec api.SecurityAdvisor, peersWithExternalEndpoints map[string]struct{}) {
	for msg := range msgChan {
//如果目标组织与源组织相同，
//消息可以包含任何组织
		if msg.src == msg.dst {
			continue
		}
		if msg.IsStateInfoMsg() || msg.IsStateInfoSnapshot() {
			inspectStateInfoMsg(t, msg, peersWithExternalEndpoints)
			continue
		}
//否则，这是一个跨组织的信息。
//将SRC组织表示为S，DST组织表示为D。
//消息的总组织必须是S U D的一个子集。
		orgs := extractOrgsFromMsg(msg.GossipMessage, sec)
		s := []string{msg.src, msg.dst}
		assert.True(t, isSubset(orgs, s), "%v isn't a subset of %v", orgs, s)

//确保除了B之外没有人知道D，反之亦然。
		if msg.dst == "D" {
			assert.NotContains(t, "A", orgs)
			assert.NotContains(t, "C", orgs)
		}

		if msg.dst == "A" || msg.dst == "C" {
			assert.NotContains(t, "D", orgs)
		}

//如果这是身份快照，请确保只有对等机的身份
//在组织之间传递外部端点。
		isIdentityPull := msg.IsPullMsg() && msg.GetPullMsgType() == proto.PullMsgType_IDENTITY_MSG
		if !(isIdentityPull && msg.IsDataUpdate()) {
			continue
		}
		for _, envp := range msg.GetDataUpdate().Data {
			identityMsg, _ := envp.ToGossipMessage()
			pkiID := identityMsg.GetPeerIdentity().PkiId
			_, hasExternalEndpoint := peersWithExternalEndpoints[string(pkiID)]
			assert.True(t, hasExternalEndpoint,
				"Peer %s doesn't have an external endpoint but its identity was gossiped", string(pkiID))
		}
	}
}

func inspectStateInfoMsg(t *testing.T, m *msg, peersWithExternalEndpoints map[string]struct{}) {
	if m.IsStateInfoMsg() {
		pkiID := m.GetStateInfo().PkiId
		_, hasExternalEndpoint := peersWithExternalEndpoints[string(pkiID)]
		assert.True(t, hasExternalEndpoint, "peer %s has no external endpoint but crossed an org", string(pkiID))
		return
	}

	for _, envp := range m.GetStateSnapshot().Elements {
		msg, _ := envp.ToGossipMessage()
		pkiID := msg.GetStateInfo().PkiId
		_, hasExternalEndpoint := peersWithExternalEndpoints[string(pkiID)]
		assert.True(t, hasExternalEndpoint, "peer %s has no external endpoint but crossed an org", string(pkiID))
	}
}

type msg struct {
	src string
	dst string
	*proto.GossipMessage
}

func isSubset(a []string, b []string) bool {
	for _, s1 := range a {
		found := false
		for _, s2 := range b {
			if s1 == s2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
