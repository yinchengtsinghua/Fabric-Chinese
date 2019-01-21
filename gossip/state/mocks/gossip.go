
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


package mocks

import (
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	"github.com/hyperledger/fabric/gossip/gossip"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/mock"
)

type GossipMock struct {
	mock.Mock
}

func (g *GossipMock) SelfMembershipInfo() discovery.NetworkMember {
	panic("implement me")
}

func (g *GossipMock) SelfChannelInfo(common.ChainID) *proto.SignedGossipMessage {
	panic("implement me")
}

func (*GossipMock) PeerFilter(channel common.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error) {
	panic("implement me")
}

func (g *GossipMock) SuspectPeers(s api.PeerSuspector) {
	g.Called(s)
}

//更新LedgerHeight更新Ledger Height the Peer
//发布到频道中的其他对等端
func (g *GossipMock) UpdateLedgerHeight(height uint64, chainID common.ChainID) {

}

//更新链码更新对等发布的链码
//到渠道中的其他同行
func (g *GossipMock) UpdateChaincodes(chaincode []*proto.Chaincode, chainID common.ChainID) {

}

func (g *GossipMock) LeaveChan(_ common.ChainID) {
	panic("implement me")
}

func (g *GossipMock) Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer) {
	g.Called(msg, peers)
}

func (g *GossipMock) Peers() []discovery.NetworkMember {
	return g.Called().Get(0).([]discovery.NetworkMember)
}

func (g *GossipMock) PeersOfChannel(chainID common.ChainID) []discovery.NetworkMember {
	args := g.Called(chainID)
	return args.Get(0).([]discovery.NetworkMember)
}

func (g *GossipMock) UpdateMetadata(metadata []byte) {
	g.Called(metadata)
}

func (g *GossipMock) Gossip(msg *proto.GossipMessage) {
	g.Called(msg)
}

func (g *GossipMock) Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage) {
	args := g.Called(acceptor, passThrough)
	if args.Get(0) == nil {
		return nil, args.Get(1).(chan proto.ReceivedMessage)
	}
	return args.Get(0).(<-chan *proto.GossipMessage), nil
}

func (g *GossipMock) JoinChan(joinMsg api.JoinChannelMessage, chainID common.ChainID) {
}

//IdentityInfo返回已知对等标识的信息
func (g *GossipMock) IdentityInfo() api.PeerIdentitySet {
	panic("not implemented")
}

func (g *GossipMock) Stop() {

}

func (g *GossipMock) SendByCriteria(*proto.SignedGossipMessage, gossip.SendCriteria) error {
	return nil
}
