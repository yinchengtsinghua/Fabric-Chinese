
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


package service

import (
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type secAdvMock struct {
}

func init() {
	util.SetupTestLogging()
}

func (s *secAdvMock) OrgByPeerIdentity(identity api.PeerIdentityType) api.OrgIdentityType {
	return api.OrgIdentityType(identity)
}

type gossipMock struct {
	mock.Mock
}

func (g *gossipMock) SelfChannelInfo(common.ChainID) *proto.SignedGossipMessage {
	panic("implement me")
}

func (g *gossipMock) SelfMembershipInfo() discovery.NetworkMember {
	panic("implement me")
}

func (*gossipMock) PeerFilter(channel common.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error) {
	panic("implement me")
}

func (*gossipMock) SuspectPeers(s api.PeerSuspector) {
	panic("implement me")
}

func (*gossipMock) Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer) {
	panic("implement me")
}

func (*gossipMock) Peers() []discovery.NetworkMember {
	panic("implement me")
}

func (*gossipMock) PeersOfChannel(common.ChainID) []discovery.NetworkMember {
	panic("implement me")
}

func (*gossipMock) UpdateMetadata(metadata []byte) {
	panic("implement me")
}

//更新LedgerHeight更新Ledger Height the Peer
//发布到频道中的其他对等端
func (*gossipMock) UpdateLedgerHeight(height uint64, chainID common.ChainID) {
	panic("implement me")
}

//更新链码更新对等发布的链码
//到渠道中的其他同行
func (*gossipMock) UpdateChaincodes(chaincode []*proto.Chaincode, chainID common.ChainID) {
	panic("implement me")
}

func (*gossipMock) Gossip(msg *proto.GossipMessage) {
	panic("implement me")
}

func (*gossipMock) Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage) {
	panic("implement me")
}

func (g *gossipMock) JoinChan(joinMsg api.JoinChannelMessage, chainID common.ChainID) {
	g.Called(joinMsg, chainID)
}

func (g *gossipMock) LeaveChan(chainID common.ChainID) {
	panic("implement me")
}

func (g *gossipMock) IdentityInfo() api.PeerIdentitySet {
	panic("implement me")
}

func (*gossipMock) Stop() {
	panic("implement me")
}

func (*gossipMock) SendByCriteria(*proto.SignedGossipMessage, gossip.SendCriteria) error {
	panic("implement me")
}

type appOrgMock struct {
	id string
}

func (*appOrgMock) Name() string {
	panic("implement me")
}

func (ao *appOrgMock) MSPID() string {
	return ao.id
}

func (ao *appOrgMock) AnchorPeers() []*peer.AnchorPeer {
	return []*peer.AnchorPeer{}
}

type configMock struct {
	orgs2AppOrgs map[string]channelconfig.ApplicationOrg
}

func (c *configMock) OrdererAddresses() []string {
	return []string{"localhost:7050"}
}

func (*configMock) ChainID() string {
	return "A"
}

func (c *configMock) Organizations() map[string]channelconfig.ApplicationOrg {
	return c.orgs2AppOrgs
}

func (*configMock) Sequence() uint64 {
	return 0
}

func TestJoinChannelConfig(t *testing.T) {
//场景：我们加入的频道只有一个组织-组织0
//但我们的组织ID实际上是负路径中的ORG0MSP
//在积极的道路上

	failChan := make(chan struct{}, 1)
	g1SvcMock := &gossipMock{}
	g1SvcMock.On("JoinChan", mock.Anything, mock.Anything).Run(func(_ mock.Arguments) {
		failChan <- struct{}{}
	})
	g1 := &gossipServiceImpl{secAdv: &secAdvMock{}, peerIdentity: api.PeerIdentityType("OrgMSP0"), gossipSvc: g1SvcMock}
	g1.updateAnchors(&configMock{
		orgs2AppOrgs: map[string]channelconfig.ApplicationOrg{
			"Org0": &appOrgMock{id: "Org0"},
		},
	})
	select {
	case <-time.After(time.Second):
	case <-failChan:
		assert.Fail(t, "Joined a badly configured channel")
	}

	succChan := make(chan struct{}, 1)
	g2SvcMock := &gossipMock{}
	g2SvcMock.On("JoinChan", mock.Anything, mock.Anything).Run(func(_ mock.Arguments) {
		succChan <- struct{}{}
	})
	g2 := &gossipServiceImpl{secAdv: &secAdvMock{}, peerIdentity: api.PeerIdentityType("Org0"), gossipSvc: g2SvcMock}
	g2.updateAnchors(&configMock{
		orgs2AppOrgs: map[string]channelconfig.ApplicationOrg{
			"Org0": &appOrgMock{id: "Org0"},
		},
	})
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "Didn't join a channel (should have done so within the time period)")
	case <-succChan:

	}
}

func TestJoinChannelNoAnchorPeers(t *testing.T) {
//场景：我们加入的频道有两个组织，但没有锚定伙伴
//该测试确保使用具有成员的JoinChannelMessage调用JoinChan
//由给定配置的组织组成。

	var joinChanCalled sync.WaitGroup
	joinChanCalled.Add(1)
	gMock := &gossipMock{}
	gMock.On("JoinChan", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		defer joinChanCalled.Done()
		jcm := args.Get(0).(api.JoinChannelMessage)
		channel := args.Get(1).(common.ChainID)
		assert.Len(t, jcm.Members(), 2)
		assert.Contains(t, jcm.Members(), api.OrgIdentityType("Org0"))
		assert.Contains(t, jcm.Members(), api.OrgIdentityType("Org1"))
		assert.Equal(t, "A", string(channel))
	})

	g := &gossipServiceImpl{secAdv: &secAdvMock{}, peerIdentity: api.PeerIdentityType("Org0"), gossipSvc: gMock}

	appOrg0 := &appOrgMock{id: "Org0"}
	appOrg1 := &appOrgMock{id: "Org1"}

//确保ApplicationOrgs确实没有锚定对等点
	assert.Empty(t, appOrg0.AnchorPeers())
	assert.Empty(t, appOrg1.AnchorPeers())

	g.updateAnchors(&configMock{
		orgs2AppOrgs: map[string]channelconfig.ApplicationOrg{
			"Org0": appOrg0,
			"Org1": appOrg1,
		},
	})
	joinChanCalled.Wait()
}
