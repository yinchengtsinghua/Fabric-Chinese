
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


package graph

type orderedSet struct {
	elements []interface{}
}

func (s *orderedSet) add(o interface{}) {
	s.elements = append(s.elements, o)
}

type indiceSet struct {
	indices []int
}

type indiceSets []*indiceSet

func factorial(n int) int {
	m := 1
	for i := 1; i <= n; i++ {
		m *= i
	}
	return m
}

func nChooseK(n, k int) int {
	a := factorial(n)
	b := factorial(n-k) * factorial(k)
	return a / b
}

func chooseKoutOfN(n, k int) indiceSets {
	var res indiceSets
	subGroups := &orderedSet{}
	choose(n, k, 0, nil, subGroups)
	for _, el := range subGroups.elements {
		res = append(res, el.(*indiceSet))
	}
	return res
}

func choose(n int, targetAmount int, i int, currentSubGroup []int, subGroups *orderedSet) {
//检查当前子组中是否有足够的元素
	if len(currentSubGroup) == targetAmount {
		subGroups.add(&indiceSet{indices: currentSubGroup})
		return
	}
//如果没有足够的剩余候选人可供选择，请提前返回
	itemsLeftToPick := n - i
	if targetAmount-len(currentSubGroup) > itemsLeftToPick {
		return
	}
//我们要么选择当前元素
	choose(n, targetAmount, i+1, append(currentSubGroup, i), subGroups)
//或者不选它
	choose(n, targetAmount, i+1, currentSubGroup, subGroups)
}
