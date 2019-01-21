
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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVertex(t *testing.T) {
	v := NewVertex("1", "1")
	assert.Equal(t, "1", v.Data)
	assert.Equal(t, "1", v.Id)
	u := NewVertex("2", "2")
	v.AddNeighbor(u)
	assert.Contains(t, u.Neighbors(), v)
	assert.Contains(t, v.Neighbors(), u)
	assert.Equal(t, u, v.NeighborById("2"))
	assert.Nil(t, v.NeighborById("3"))
}
