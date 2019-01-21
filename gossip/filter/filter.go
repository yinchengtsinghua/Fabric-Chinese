
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
	"math/rand"

	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/util"
)

//routingfilter定义网络成员上的谓词
//它用于断言给定的networkmember是否应
//为获得消息而选择
type RoutingFilter func(discovery.NetworkMember) bool

//SelectNonePolicy selects an empty set of members
var SelectNonePolicy = func(discovery.NetworkMember) bool {
	return false
}

//selectallpolicy选择给定的所有成员
var SelectAllPolicy = func(discovery.NetworkMember) bool {
	return true
}

//combineroutingfilters返回给定路由筛选器的逻辑与
func CombineRoutingFilters(filters ...RoutingFilter) RoutingFilter {
	return func(member discovery.NetworkMember) bool {
		for _, filter := range filters {
			if !filter(member) {
				return false
			}
		}
		return true
	}
}

//selectpeers返回一个与路由筛选器匹配的对等片
func SelectPeers(k int, peerPool []discovery.NetworkMember, filter RoutingFilter) []*comm.RemotePeer {
	var res []*comm.RemotePeer
	rand.Seed(int64(util.RandomUInt64()))
//以随机顺序迭代可能的候选对象
	for _, index := range rand.Perm(len(peerPool)) {
//如果我们收集了k个对等点，就可以停止迭代。
		if len(res) == k {
			break
		}
		peer := peerPool[index]
//For each one, check if it is a worthy candidate to be selected
		if !filter(peer) {
			continue
		}
		p := &comm.RemotePeer{PKIID: peer.PKIid, Endpoint: peer.PreferredEndpoint()}
		res = append(res, p)
	}
	return res
}

//first返回与给定筛选器匹配的第一个对等机
func First(peerPool []discovery.NetworkMember, filter RoutingFilter) *comm.RemotePeer {
	for _, p := range peerPool {
		if filter(p) {
			return &comm.RemotePeer{PKIID: p.PKIid, Endpoint: p.PreferredEndpoint()}
		}
	}
	return nil
}

//any match筛选出与任何给定筛选器都不匹配的对等方
func AnyMatch(peerPool []discovery.NetworkMember, filters ...RoutingFilter) []discovery.NetworkMember {
	var res []discovery.NetworkMember
	for _, peer := range peerPool {
		for _, matches := range filters {
			if matches(peer) {
				res = append(res, peer)
				break
			}
		}
	}
	return res
}
