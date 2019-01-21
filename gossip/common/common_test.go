
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


package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNotSame(t *testing.T) {
	id := PKIidType("1")
	assert.True(t, id.IsNotSameFilter(PKIidType("2")))
	assert.False(t, id.IsNotSameFilter(PKIidType("1")))
	assert.False(t, id.IsNotSameFilter(id))
}

func TestPKIidTypeStringer(t *testing.T) {
	tests := []struct {
		input    PKIidType
		expected string
	}{
		{nil, "<nil>"},
		{PKIidType{}, ""},
		{PKIidType{0, 1, 2, 3}, "00010203"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.input.String())
	}
}
