
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

//Treepermutations表示可能的排列
//一棵树
type treePermutations struct {
originalRoot           *TreeVertex                     //所有子树的根顶点
permutations           []*TreeVertex                   //累积排列
descendantPermutations map[*TreeVertex][][]*TreeVertex //基于当前顶点的阈值定义子树的组合
}

//NewTreePermuation创建具有给定根顶点的新TreePermuations对象
func newTreePermutation(root *TreeVertex) *treePermutations {
	return &treePermutations{
		descendantPermutations: make(map[*TreeVertex][][]*TreeVertex),
		originalRoot:           root,
		permutations:           []*TreeVertex{root},
	}
}

//permute返回顶点和边都存在于原始顶点树中的树。
//是treepermutations的“originalRoot”字段
func (tp *treePermutations) permute() []*Tree {
	tp.computeDescendantPermutations()

	it := newBFSIterator(tp.originalRoot)
	for {
		v := it.Next()
		if v == nil {
			break
		}

		if len(v.Descendants) == 0 {
			continue
		}

//迭代所有存在v的置换
//将它们分为两组：一组存在的指标，一组不存在的指标
		var permutationsWhereVexists []*TreeVertex
		var permutationsWhereVdoesntExist []*TreeVertex
		for _, perm := range tp.permutations {
			if perm.Exists(v.Id) {
				permutationsWhereVexists = append(permutationsWhereVexists, perm)
			} else {
				permutationsWhereVdoesntExist = append(permutationsWhereVdoesntExist, perm)
			}
		}

//从排列中删除V存在的排列
		tp.permutations = permutationsWhereVdoesntExist

//接下来，我们将替换存在v的置换的每一次出现，
//其后代排列出现多次
		for _, perm := range permutationsWhereVexists {
//对于V的后代的每个排列，克隆该图
//并用置换图替换v创建一个新图
//与后代排列有关的
			for _, permutation := range tp.descendantPermutations[v] {
				subGraph := &TreeVertex{
					Id:          v.Id,
					Data:        v.Data,
					Descendants: permutation,
				}
				newTree := perm.Clone()
				newTree.replace(v.Id, subGraph)
//将新选项添加到排列中
				tp.permutations = append(tp.permutations, newTree)
			}
		}
	}

	res := make([]*Tree, len(tp.permutations))
	for i, perm := range tp.permutations {
		res[i] = perm.ToTree()
	}
	return res
}

//可计算的静态置换计算子树的所有可能组合
//对于所有顶点，基于阈值。
func (tp *treePermutations) computeDescendantPermutations() {
	it := newBFSIterator(tp.originalRoot)
	for {
		v := it.Next()
		if v == nil {
			return
		}

//箕叶
		if len(v.Descendants) == 0 {
			continue
		}

//遍历从后代中选择阈值的所有选项
		for _, el := range chooseKoutOfN(len(v.Descendants), v.Threshold) {
//对于每个这样的选项，将其附加到当前的treevertex
			tp.descendantPermutations[v] = append(tp.descendantPermutations[v], v.selectDescendants(el.indices))
		}
	}
}

//selectDescendants根据给定的索引返回后代的子集
func (v *TreeVertex) selectDescendants(indices []int) []*TreeVertex {
	r := make([]*TreeVertex, len(indices))
	i := 0
	for _, index := range indices {
		r[i] = v.Descendants[index]
		i++
	}
	return r
}
