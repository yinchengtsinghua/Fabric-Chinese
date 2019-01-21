
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
*/


package inquire

import (
	"reflect"

	"github.com/hyperledger/fabric/common/policies"
)

//
type ComparablePrincipalSets []ComparablePrincipalSet

//TopPrincipalsets将此可比较的Principalsets转换为Principalsets
func (cps ComparablePrincipalSets) ToPrincipalSets() policies.PrincipalSets {
	var res policies.PrincipalSets
	for _, cp := range cps {
		res = append(res, cp.ToPrincipalSet())
	}
	return res
}

//合并返回基础Principalset包含的可比较Principalset
//
//
//p1和p2是p1中每个主集p满足ep1的主集，
//
//分别用s1和s2表示从ep1和ep2得出的可比较原则集。
//
//这样，S中的每一个可比较原则都满足EP1和EP2。
func Merge(s1, s2 ComparablePrincipalSets) ComparablePrincipalSets {
	var res ComparablePrincipalSets
	setsIn1ToTheContainingSetsIn2 := computeContainedInMapping(s1, s2)
	setsIn1ThatAreIn2 := s1.OfMapping(setsIn1ToTheContainingSetsIn2, s2)
//
//由s2中的主体集包含，以便不具有重复项
	s1 = s1.ExcludeIndices(setsIn1ToTheContainingSetsIn2)
	setsIn2ToTheContainingSetsIn1 := computeContainedInMapping(s2, s1)
	setsIn2ThatAreIn1 := s2.OfMapping(setsIn2ToTheContainingSetsIn1, s1)
	s2 = s2.ExcludeIndices(setsIn2ToTheContainingSetsIn1)

//在过渡期间，结果包含来自第一个或第二个的集
//
	res = append(res, setsIn1ThatAreIn2.ToMergedPrincipalSets()...)
	res = append(res, setsIn2ThatAreIn1.ToMergedPrincipalSets()...)

//现在，从原始组s1和s2中清除主体集
//找到包含其他组中的集的。
//动机是留给s1和s2，它们只包含不包含
//包含或包含其他组的任何集合。
	s1 = s1.ExcludeIndices(setsIn2ToTheContainingSetsIn1.invert())
	s2 = s2.ExcludeIndices(setsIn1ToTheContainingSetsIn2.invert())

//两个集合或其中一个集合中都有主集合。
//
	if len(s1) == 0 || len(s2) == 0 {
		return res.Reduce()
	}

//
//在这两个集合。因此，我们应该把它们组合成主集
//包含两个集合的。
	combinedPairs := CartesianProduct(s1, s2)
	res = append(res, combinedPairs.ToMergedPrincipalSets()...)
	return res.Reduce()
}

//CartesianProduct返回由组合组成的ComparablePrincipalsetPairs
//在每对可能的可比较原理集合中，第一个元素在s1中，
//第二个元素在s2中。
func CartesianProduct(s1, s2 ComparablePrincipalSets) comparablePrincipalSetPairs {
	var res comparablePrincipalSetPairs
	for _, x := range s1 {
		var set comparablePrincipalSetPairs
//对于第一组中的每一组，
//把它和第二组中的每一组结合起来
		for _, y := range s2 {
			set = append(set, comparablePrincipalSetPair{
				contained:  x,
				containing: y,
			})
		}
		res = append(res, set...)
	}
	return res
}

//ComparablePrincipalsetPair是两个ComparablePrincipalsets的元组
type comparablePrincipalSetPair struct {
	contained  ComparablePrincipalSet
	containing ComparablePrincipalSet
}

//
//ComparablePrincipalset对中包含的ComparablePrincipalset保留
func (pair comparablePrincipalSetPair) MergeWithPlurality() ComparablePrincipalSet {
	var principalsToAdd []*ComparablePrincipal
	used := make(map[int]struct{})
//对包含的集和每个主体进行迭代
	for _, principal := range pair.contained {
		var covered bool
//
		for i, coveringPrincipal := range pair.containing {
//
			if _, isUsed := used[i]; isUsed {
				continue
			}
//所有满足找到的主体的标识也应该满足覆盖的主体。
			if coveringPrincipal.IsA(principal) {
				used[i] = struct{}{}
				covered = true
				break
			}
		}
//如果我们找不到校长的掩护，那是因为我们已经用尽了所有可能的候选人
//在包含集中，只需将其添加到稍后要添加的主体集。
		if !covered {
			principalsToAdd = append(principalsToAdd, principal)
		}
	}

	res := pair.containing.Clone()
	res = append(res, principalsToAdd...)
	return res
}

//ComparablePrincipalsetPairs聚合[]ComparablePrincipalsetPairs
type comparablePrincipalSetPairs []comparablePrincipalSetPair

//TopPrincipalsets将ComparablePrincipalsetPair转换为ComparablePrincipalsets
//同时考虑到每对的多个
func (pairs comparablePrincipalSetPairs) ToMergedPrincipalSets() ComparablePrincipalSets {
	var res ComparablePrincipalSets
	for _, pair := range pairs {
		res = append(res, pair.MergeWithPlurality())
	}
	return res
}

//
func (cps ComparablePrincipalSets) OfMapping(mapping map[int][]int, sets2 ComparablePrincipalSets) comparablePrincipalSetPairs {
	var res []comparablePrincipalSetPair
	for i, js := range mapping {
		for _, j := range js {
			res = append(res, comparablePrincipalSetPair{
				contained:  cps[i],
				containing: sets2[j],
			})
		}
	}
	return res
}

//
//
func (cps ComparablePrincipalSets) Reduce() ComparablePrincipalSets {
	var res ComparablePrincipalSets
	for i, s1 := range cps {
		var isContaining bool
		for j, s2 := range cps {
			if i == j {
				continue
			}
			if s2.IsSubset(s1) {
				isContaining = true
			}
		}
		if !isContaining {
			res = append(res, s1)
		}
	}
	return res
}

//excludeindexs返回一个可比较的Principalset，但没有在键中找到给定的索引
func (cps ComparablePrincipalSets) ExcludeIndices(mapping map[int][]int) ComparablePrincipalSets {
	var res ComparablePrincipalSets
	for i, set := range cps {
		if _, exists := mapping[i]; exists {
			continue
		}
		res = append(res, set)
	}
	return res
}

//包含返回此ComparablePrincipalset是否包含给定的ComparablePrincipalset。
//
//在x中有一个可比较的主体x，这样x.is a（y）。
//由此可知，每个满足x的签名集也满足y。
func (cps ComparablePrincipalSet) Contains(s *ComparablePrincipal) bool {
	for _, cp := range cps {
		if cp.IsA(s) {
			return true
		}
	}
	return false
}

//iscontainedIn返回此ComparablePrincipalset是否包含在给定的ComparablePrincipalset中。
//
//如果x中的每个可比较原则集x在y中都有一个可比较原则集y，则y.is a（x）为真。
//
//
//恒等式，使恒等式满足y，因此也满足x。
func (cps ComparablePrincipalSet) IsContainedIn(set ComparablePrincipalSet) bool {
	for _, cp := range cps {
		if !set.Contains(cp) {
			return false
		}
	}
	return true
}

//ComputeContainedInMapping返回第一个可比较原则集中的索引的映射
//对第二个可比较原则中的指数进行比较
//
func computeContainedInMapping(s1, s2 []ComparablePrincipalSet) intMapping {
	mapping := make(map[int][]int)
	for i, ps1 := range s1 {
		for j, ps2 := range s2 {
			if !ps1.IsContainedIn(ps2) {
				continue
			}
			mapping[i] = append(mapping[i], j)
		}
	}
	return mapping
}

//Intmap将整数映射到整数集
type intMapping map[int][]int

func (im intMapping) invert() intMapping {
	res := make(intMapping)
	for i, js := range im {
		for _, j := range js {
			res[j] = append(res[j], i)
		}
	}
	return res
}

//is subset返回此ComparablePrincipalset是否为给定ComparablePrincipalset的子集
func (cps ComparablePrincipalSet) IsSubset(sets ComparablePrincipalSet) bool {
	for _, p1 := range cps {
		var found bool
		for _, p2 := range sets {
			if reflect.DeepEqual(p1, p2) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
