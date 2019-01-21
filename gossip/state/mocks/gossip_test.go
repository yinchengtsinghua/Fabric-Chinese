
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
	"testing"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGossipMock(t *testing.T) {
	g := GossipMock{}
	mkChan := func() <-chan *proto.GossipMessage {
		c := make(chan *proto.GossipMessage, 1)
		c <- &proto.GossipMessage{}
		return c
	}
	g.On("Accept", mock.Anything, false).Return(mkChan(), nil)
	a, b := g.Accept(func(o interface{}) bool {
		return true
	}, false)
	assert.Nil(t, b)
	assert.NotNil(t, a)
	assert.Panics(t, func() {
		g.SuspectPeers(func(identity api.PeerIdentityType) bool { return false })
	})
	assert.Panics(t, func() {
		g.Send(nil, nil)
	})
	assert.Panics(t, func() {
		g.Peers()
	})
	g.On("PeersOfChannel", mock.Anything).Return([]discovery.NetworkMember{})
	assert.Empty(t, g.PeersOfChannel(common.ChainID("A")))

	assert.Panics(t, func() {
		g.UpdateMetadata([]byte{})
	})
	assert.Panics(t, func() {
		g.Gossip(nil)
	})
	assert.NotPanics(t, func() {
		g.UpdateLedgerHeight(0, common.ChainID("A"))
		g.Stop()
		g.JoinChan(nil, common.ChainID("A"))
	})
}
