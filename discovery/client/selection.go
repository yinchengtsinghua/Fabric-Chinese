
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


package discovery

import (
	"math/rand"
	"sort"
	"time"
)

//过滤和排序给定的背书人
type Filter interface {
	Filter(endorsers Endorsers) Endorsers
}

//如果给定的对等端
//is not to be considered when selecting peers
type ExclusionFilter interface {
//exclude返回是否排除给定的对等机
	Exclude(Peer) bool
}

type selectionFunc func(Peer) bool

func (sf selectionFunc) Exclude(p Peer) bool {
	return sf(p)
}

//PrioritySelector通过
//给予同伴相对优先的选择权
type PrioritySelector interface {
//比较两个对等点和返回之间的比较
//他们的相对分数
	Compare(Peer, Peer) Priority
}

//优先级定义选择对等机的可能性
//超过另一个同伴。
//正优先级表示选择了左对等
//负优先级意味着选择正确的对等体。
//零优先级意味着它们的优先级相同
type Priority int

var (
//PrioritiesByHeight selects peers by descending height
	PrioritiesByHeight = &byHeight{}
//NoExclusion接受所有对等方，不拒绝任何对等方
	NoExclusion = selectionFunc(noExclusion)
//NoPriorities is indifferent to how it selects peers
	NoPriorities = &noPriorities{}
)

type noPriorities struct{}

func (nc noPriorities) Compare(_ Peer, _ Peer) Priority {
	return 0
}

type byHeight struct{}

func (*byHeight) Compare(left Peer, right Peer) Priority {
	leftHeight := left.StateInfoMessage.GetStateInfo().Properties.LedgerHeight
	rightHeight := right.StateInfoMessage.GetStateInfo().Properties.LedgerHeight

	if leftHeight > rightHeight {
		return 1
	}
	if rightHeight > leftHeight {
		return -1
	}
	return 0
}

func noExclusion(_ Peer) bool {
	return false
}

//excludehosts返回排除给定端点的exclusionfilter
func ExcludeHosts(endpoints ...string) ExclusionFilter {
	m := make(map[string]struct{})
	for _, endpoint := range endpoints {
		m[endpoint] = struct{}{}
	}
	return ExcludeByHost(func(host string) bool {
		_, excluded := m[host]
		return excluded
	})
}

//ExcludeByHost creates a ExclusionFilter out of the given exclusion predicate
func ExcludeByHost(reject func(host string) bool) ExclusionFilter {
	return selectionFunc(func(p Peer) bool {
		endpoint := p.AliveMessage.GetAliveMsg().Membership.Endpoint
		var internalEndpoint string
		se := p.AliveMessage.GetSecretEnvelope()
		if se != nil {
			internalEndpoint = se.InternalEndpoint()
		}
		return reject(endpoint) || reject(internalEndpoint)
	})
}

//Filter filters the endorsers according to the given ExclusionFilter
func (endorsers Endorsers) Filter(f ExclusionFilter) Endorsers {
	var res Endorsers
	for _, e := range endorsers {
		if !f.Exclude(*e) {
			res = append(res, e)
		}
	}
	return res
}

//shuffle按随机顺序对背书人排序
func (endorsers Endorsers) Shuffle() Endorsers {
	res := make(Endorsers, len(endorsers))
	rand.Seed(time.Now().UnixNano())
	for i, index := range rand.Perm(len(endorsers)) {
		res[i] = endorsers[index]
	}
	return res
}

type endorserSort struct {
	Endorsers
	PrioritySelector
}

//根据给定的PrioritySelector对背书人排序
func (endorsers Endorsers) Sort(ps PrioritySelector) Endorsers {
	sort.Sort(&endorserSort{
		Endorsers:        endorsers,
		PrioritySelector: ps,
	})
	return endorsers
}

func (es *endorserSort) Len() int {
	return len(es.Endorsers)
}

func (es *endorserSort) Less(i, j int) bool {
	e1 := es.Endorsers[i]
	e2 := es.Endorsers[j]
	less := es.Compare(*e1, *e2)
	return less > Priority(0)
}

func (es *endorserSort) Swap(i, j int) {
	es.Endorsers[i], es.Endorsers[j] = es.Endorsers[j], es.Endorsers[i]
}
