
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

//顶点定义图形的顶点
type Vertex struct {
	Id        string
	Data      interface{}
	neighbors map[string]*Vertex
}

//newVertex创建具有给定ID和数据的新顶点
func NewVertex(id string, data interface{}) *Vertex {
	return &Vertex{
		Id:        id,
		Data:      data,
		neighbors: make(map[string]*Vertex),
	}
}

//neighborbyid返回具有给定id的邻居顶点，
//如果没有具有此ID的顶点是邻居，则为零。
func (v *Vertex) NeighborById(id string) *Vertex {
	return v.neighbors[id]
}

//邻居返回顶点的邻居
func (v *Vertex) Neighbors() []*Vertex {
	var res []*Vertex
	for _, u := range v.neighbors {
		res = append(res, u)
	}
	return res
}

//addneighbor将给定顶点添加为邻居
//顶点的
func (v *Vertex) AddNeighbor(u *Vertex) {
	v.neighbors[u.Id] = u
	u.neighbors[v.Id] = v
}
