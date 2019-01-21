
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionSerialization(t *testing.T) {
	h1 := NewHeight(10, 100)
	b := h1.ToBytes()
	h2, n := NewHeightFromBytes(b)
	assert.Equal(t, h1, h2)
	assert.Len(t, b, n)
}

func TestVersionComparison(t *testing.T) {
	assert.Equal(t, 1, NewHeight(10, 100).Compare(NewHeight(9, 1000)))
	assert.Equal(t, 1, NewHeight(10, 100).Compare(NewHeight(10, 90)))
	assert.Equal(t, -1, NewHeight(10, 100).Compare(NewHeight(11, 1)))
	assert.Equal(t, 0, NewHeight(10, 100).Compare(NewHeight(10, 100)))

	assert.True(t, AreSame(NewHeight(10, 100), NewHeight(10, 100)))
	assert.True(t, AreSame(nil, nil))
	assert.False(t, AreSame(NewHeight(10, 100), nil))
}

func TestVersionExtraBytes(t *testing.T) {
	extraBytes := []byte("junk")
	h1 := NewHeight(10, 100)
	b := h1.ToBytes()
	b1 := append(b, extraBytes...)
	h2, n := NewHeightFromBytes(b1)
	assert.Equal(t, h1, h2)
	assert.Len(t, b, n)
	assert.Equal(t, extraBytes, b1[n:])
}
