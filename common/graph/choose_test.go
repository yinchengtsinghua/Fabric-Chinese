
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

func TestChoose(t *testing.T) {
	assert.Equal(t, 24, factorial(4))
	assert.Equal(t, 1, factorial(0))
	assert.Equal(t, 1, factorial(1))
	assert.Equal(t, 15504, nChooseK(20, 5))
	for n := 1; n < 20; n++ {
		for k := 1; k < n; k++ {
			g := chooseKoutOfN(n, k)
			assert.Equal(t, nChooseK(n, k), len(g), "n=%d, k=%d", n, k)
		}
	}
}
