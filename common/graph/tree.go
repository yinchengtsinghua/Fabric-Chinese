
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

//迭代器定义可用于遍历顶点的迭代器
//图的
type Iterator interface {
//next返回迭代顺序中的下一个元素，
//如果没有这样的元素，则为零。
	Next() *TreeVertex
}

//tree vertex定义树的顶点
type TreeVertex struct {
Id          string        //ID唯一标识树中的treevertex
Data        interface{}   //数据保存任意数据，供包的用户使用
Descendants []*TreeVertex //子代是树中此treevertex是其父代的顶点。
Threshold   int           //阈值符号创建树排列时要拾取的子树/叶的计数
}

//newtreevertex创建具有给定唯一ID和给定任意数据的新顶点
func NewTreeVertex(id string, data interface{}, descendants ...*TreeVertex) *TreeVertex {
	return &TreeVertex{
		Id:          id,
		Data:        data,
		Descendants: descendants,
	}
}

//is leaf返回给定顶点是否为叶
func (v *TreeVertex) IsLeaf() bool {
	return len(v.Descendants) == 0
}

//addDescendant创建一个新顶点，其父顶点是调用器顶点，
//具有给定的ID和数据。返回新顶点
func (v *TreeVertex) AddDescendant(u *TreeVertex) *TreeVertex {
	v.Descendants = append(v.Descendants, u)
	return u
}

//totree创建的树的根顶点是当前顶点
func (v *TreeVertex) ToTree() *Tree {
	return &Tree{
		Root: v,
	}
}

//查找搜索ID为给定ID的顶点。
//返回找到的具有此ID的第一个顶点，如果未找到，则返回nil
func (v *TreeVertex) Find(id string) *TreeVertex {
	if v.Id == id {
		return v
	}
	for _, u := range v.Descendants {
		if r := u.Find(id); r != nil {
			return r
		}
	}
	return nil
}

//exists搜索一个id为给定id的顶点，
//并返回是否找到这样的顶点。
func (v *TreeVertex) Exists(id string) bool {
	return v.Find(id) != nil
}

//克隆克隆当前顶点为根顶点的树。
func (v *TreeVertex) Clone() *TreeVertex {
	var descendants []*TreeVertex
	for _, u := range v.Descendants {
		descendants = append(descendants, u.Clone())
	}
	copy := &TreeVertex{
		Id:          v.Id,
		Descendants: descendants,
		Data:        v.Data,
	}
	return copy
}

//替换替换标识为给定标识的顶点的子树
//子树的根顶点是r。
func (v *TreeVertex) replace(id string, r *TreeVertex) {
	if v.Id == id {
		v.Descendants = r.Descendants
		return
	}
	for _, u := range v.Descendants {
		u.replace(id, r)
	}
}

//树定义treevertex类型的顶点树
type Tree struct {
	Root *TreeVertex
}

//permute返回顶点和边都存在于原始树中的树。
//排列是根据所有顶点的阈值计算的。
func (t *Tree) Permute() []*Tree {
	return newTreePermutation(t.Root).permute()
}

//bfs返回迭代顶点的迭代器
//按广度优先搜索顺序
func (t *Tree) BFS() Iterator {
	return newBFSIterator(t.Root)
}

type bfsIterator struct {
	*queue
}

func newBFSIterator(v *TreeVertex) *bfsIterator {
	return &bfsIterator{
		queue: &queue{
			arr: []*TreeVertex{v},
		},
	}
}

//next返回迭代顺序中的下一个元素，
//如果没有这样的元素，则为零。
func (bfs *bfsIterator) Next() *TreeVertex {
	if len(bfs.arr) == 0 {
		return nil
	}
	v := bfs.dequeue()
	for _, u := range v.Descendants {
		bfs.enqueue(u)
	}
	return v
}

//由切片支持的队列的基本实现
type queue struct {
	arr []*TreeVertex
}

func (q *queue) enqueue(v *TreeVertex) {
	q.arr = append(q.arr, v)
}

func (q *queue) dequeue() *TreeVertex {
	v := q.arr[0]
	q.arr = q.arr[1:]
	return v
}
