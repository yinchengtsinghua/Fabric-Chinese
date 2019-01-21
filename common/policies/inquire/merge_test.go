
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


package inquire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	member1 = NewComparablePrincipal(member("Org1MSP"))
	member2 = NewComparablePrincipal(member("Org2MSP"))
	member3 = NewComparablePrincipal(member("Org3MSP"))
	member4 = NewComparablePrincipal(member("Org4MSP"))
	member5 = NewComparablePrincipal(member("Org5MSP"))
	peer1   = NewComparablePrincipal(peer("Org1MSP"))
	peer2   = NewComparablePrincipal(peer("Org2MSP"))
	peer3   = NewComparablePrincipal(peer("Org3MSP"))
	peer4   = NewComparablePrincipal(peer("Org4MSP"))
	peer5   = NewComparablePrincipal(peer("Org5MSP"))
)

func TestString(t *testing.T) {
	cps := ComparablePrincipalSet{member1, member2, NewComparablePrincipal(ou("Org3MSP"))}
	assert.Equal(t, "[Org1MSP.MEMBER, Org2MSP.MEMBER, Org3MSP.ou]", cps.String())
}

func TestClone(t *testing.T) {
	cps := ComparablePrincipalSet{member1, member2}
	clone := cps.Clone()
	assert.Equal(t, cps, clone)
//
	cps[0] = nil
	assert.False(t, cps[0] == clone[0])
}

func TestMergeInclusiveWithPlurality(t *testing.T) {
//脚本：
//
//s2=org1.peer，org2.peer，org3.member，org4.member
//预期合并结果：
//org1.peer，org2.peer，org2.member，org3.peer，org4.peer_

	members12 := ComparablePrincipalSet{member1, member2, member2}
	members34 := ComparablePrincipalSet{member3, member4}
	peers12 := ComparablePrincipalSet{peer1, peer2}
	peers34 := ComparablePrincipalSet{peer3, peer4}

	peers12member2 := ComparablePrincipalSet{peer1, peer2, member2}

	s1 := ComparablePrincipalSets{members12, peers34}
	s2 := ComparablePrincipalSets{peers12, members34}

	merged := Merge(s1, s2)
	expected := ComparablePrincipalSets{peers12member2, peers34}
	assert.Equal(t, expected, merged)

//
	s1 = ComparablePrincipalSets{peers34, members12}
	s2 = ComparablePrincipalSets{members34, peers12}
	merged = Merge(s1, s2)
	assert.Equal(t, expected, merged)

//将顺序移动到呼叫
	merged = Merge(s2, s1)
	expected = ComparablePrincipalSets{peers34, peers12member2}
	assert.Equal(t, expected, merged)
}

func TestMergeExclusiveWithPlurality(t *testing.T) {
//脚本：
//s1=org1.member，org2.member，org2.member，org3.peer，org4.peer
//s2=org2.member，org3.member，org4.peer，org5.peer
//预期合并结果：
//org1.member，org2.member，org3.member，org3.peer，org4.peer，org5.peer，
//org1.member，org2.member，org4.peer，org5.peer，org3.peer，org4.peer，org2.member，org3.member

	members122 := ComparablePrincipalSet{member1, member2, member2}
	members23 := ComparablePrincipalSet{member2, member3}
	peers34 := ComparablePrincipalSet{peer3, peer4}
	peers45 := ComparablePrincipalSet{peer4, peer5}
	members1223 := ComparablePrincipalSet{member1, member2, member2, member3}
	peers345 := ComparablePrincipalSet{peer3, peer4, peer5}
	members122peers45 := ComparablePrincipalSet{member1, member2, member2, peer4, peer5}
	peers34members23 := ComparablePrincipalSet{peer3, peer4, member2, member3}

	s1 := ComparablePrincipalSets{members122, peers34}
	s2 := ComparablePrincipalSets{members23, peers45}
	merged := Merge(s1, s2)
	expected := ComparablePrincipalSets{members1223, peers345, members122peers45, peers34members23}
	assert.True(t, expected.IsEqual(merged))
}

func TestMergePartialExclusiveWithPlurality(t *testing.T) {
//脚本：
//
//s2=org2.member，org3.member，org3.peer，org4.member，org4.member，org4.peer，org5.member
//
//org1.member，org2.member，org3.member，org3.peer，org4.peer，org4.member，org5.member，org3.peer，org4.peer，org5.member_
//
//因此，应该优化结果，因此org3.peer、org4.peer、org4.member、org5.member将被省略。
//预期合并结果：
//org1.成员，org2.成员，org3.成员，org3.对等，org4.对等，org5.成员

	members12 := ComparablePrincipalSet{member1, member2}
	members23 := ComparablePrincipalSet{member2, member3}
	peers34member5 := ComparablePrincipalSet{peer3, peer4, member5}
	peer3members44 := ComparablePrincipalSet{peer3, member4, member4}
	peer4member5 := ComparablePrincipalSet{peer4, member5}
	members123 := ComparablePrincipalSet{member1, member2, member3}

	s1 := ComparablePrincipalSets{members12, peers34member5}
	s2 := ComparablePrincipalSets{members23, peer3members44, peer4member5}
	merged := Merge(s1, s2)
	expected := ComparablePrincipalSets{members123, peers34member5}
	assert.True(t, expected.IsEqual(merged))
}

func TestMergeWithPlurality(t *testing.T) {
	pair := comparablePrincipalSetPair{
		contained:  ComparablePrincipalSet{peer3, member4, member4},
		containing: ComparablePrincipalSet{peer3, peer4, member5},
	}
	merged := pair.MergeWithPlurality()
	expected := ComparablePrincipalSet{peer3, peer4, member5, member4}
	assert.Equal(t, expected, merged)
}

func TestIsSubset(t *testing.T) {
	members12 := ComparablePrincipalSet{member1, member2, member2}
	members321 := ComparablePrincipalSet{member3, member2, member1}
	members13 := ComparablePrincipalSet{member1, member3}
	assert.True(t, members12.IsSubset(members12))
	assert.True(t, members12.IsSubset(members321))
	assert.False(t, members12.IsSubset(members13))
}

func TestReduce(t *testing.T) {
	members12 := ComparablePrincipalSet{member1, member2}
	members123 := ComparablePrincipalSet{member1, member2, member3}
	members12peers45 := ComparablePrincipalSet{member1, member2, peer4, peer5}
	peers45 := ComparablePrincipalSet{peer4, peer5}
	peers34 := ComparablePrincipalSet{peer3, peer4}
	s := ComparablePrincipalSets{members12, peers34, members123, members123, members12peers45, peers45}
	expected := ComparablePrincipalSets{members12, peers34, peers45}
	assert.Equal(t, expected, s.Reduce())
}

//IsEqual返回此ComparablePrincipalsets是否包含给定ComparablePrincipalsets的元素
//按某种顺序
func (cps ComparablePrincipalSets) IsEqual(sets ComparablePrincipalSets) bool {
	return cps.IsSubset(sets) && sets.IsSubset(cps)
}

//is subset返回此ComparablePrincipalset是否是给定ComparablePrincipalset的子集
func (cps ComparablePrincipalSets) IsSubset(sets ComparablePrincipalSets) bool {
	for _, sets1 := range cps {
		var found bool
		for _, sets2 := range sets {
			if sets1.IsEqual(sets2) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

//
//按某种顺序
func (cps ComparablePrincipalSet) IsEqual(otherSet ComparablePrincipalSet) bool {
	return cps.IsSubset(otherSet) && otherSet.IsSubset(cps)
}
