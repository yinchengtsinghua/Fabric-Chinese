
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


package filter

import (
	"testing"

	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/stretchr/testify/assert"
)

func TestSelectPolicies(t *testing.T) {
	assert.True(t, SelectAllPolicy(discovery.NetworkMember{}))
	assert.False(t, SelectNonePolicy(discovery.NetworkMember{}))
}

func TestCombineRoutingFilters(t *testing.T) {
	nm := discovery.NetworkMember{
		Endpoint:         "a",
		InternalEndpoint: "b",
	}
//确保组合路由筛选器是逻辑和
	a := func(nm discovery.NetworkMember) bool {
		return nm.Endpoint == "a"
	}
	b := func(nm discovery.NetworkMember) bool {
		return nm.InternalEndpoint == "b"
	}
	assert.True(t, CombineRoutingFilters(a, b)(nm))
	assert.False(t, CombineRoutingFilters(CombineRoutingFilters(a, b), SelectNonePolicy)(nm))
	assert.False(t, CombineRoutingFilters(a, b)(discovery.NetworkMember{InternalEndpoint: "b"}))
}

func TestAnyMatch(t *testing.T) {
	peerA := discovery.NetworkMember{Endpoint: "a"}
	peerB := discovery.NetworkMember{Endpoint: "b"}
	peerC := discovery.NetworkMember{Endpoint: "c"}
	peerD := discovery.NetworkMember{Endpoint: "d"}

	peers := []discovery.NetworkMember{peerA, peerB, peerC, peerD}

	matchB := func(nm discovery.NetworkMember) bool {
		return nm.Endpoint == "b"
	}
	matchC := func(nm discovery.NetworkMember) bool {
		return nm.Endpoint == "c"
	}

	matched := AnyMatch(peers, matchB, matchC)
	assert.Len(t, matched, 2)
	assert.Contains(t, matched, peerB)
	assert.Contains(t, matched, peerC)
}

func TestFirst(t *testing.T) {
	peerA := discovery.NetworkMember{Endpoint: "a"}
	peerB := discovery.NetworkMember{Endpoint: "b"}
	peers := []discovery.NetworkMember{peerA, peerB}
	assert.Equal(t, &comm.RemotePeer{Endpoint: "a"}, First(peers, func(discovery.NetworkMember) bool {
		return true
	}))

	assert.Equal(t, &comm.RemotePeer{Endpoint: "b"}, First(peers, func(nm discovery.NetworkMember) bool {
		return nm.PreferredEndpoint() == "b"
	}))

	peerAA := discovery.NetworkMember{Endpoint: "aa"}
	peerAB := discovery.NetworkMember{Endpoint: "ab"}
	peers = append(peers, peerAA)
	peers = append(peers, peerAB)
	assert.Equal(t, &comm.RemotePeer{Endpoint: "aa"}, First(peers, func(nm discovery.NetworkMember) bool {
		return len(nm.PreferredEndpoint()) > 1
	}))
}

func TestSelectPeers(t *testing.T) {
	a := func(nm discovery.NetworkMember) bool {
		return nm.Endpoint == "a"
	}
	b := func(nm discovery.NetworkMember) bool {
		return nm.InternalEndpoint == "b"
	}
	nm1 := discovery.NetworkMember{
		Endpoint:         "a",
		InternalEndpoint: "b",
		PKIid:            common.PKIidType("a"),
	}
	nm2 := discovery.NetworkMember{
		Endpoint:         "a",
		InternalEndpoint: "b",
		PKIid:            common.PKIidType("b"),
	}
	nm3 := discovery.NetworkMember{
		Endpoint:         "d",
		InternalEndpoint: "b",
		PKIid:            common.PKIidType("c"),
	}
	assert.Len(t, SelectPeers(3, []discovery.NetworkMember{nm1, nm2, nm3}, CombineRoutingFilters(a, b)), 2)
	assert.Len(t, SelectPeers(5, []discovery.NetworkMember{nm1, nm2, nm3}, CombineRoutingFilters(a, b)), 2)
	assert.Len(t, SelectPeers(1, []discovery.NetworkMember{nm1, nm2, nm3}, CombineRoutingFilters(a, b)), 1)
}
