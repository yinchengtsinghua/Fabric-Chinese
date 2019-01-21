
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


package capabilities

import (
	"testing"

	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestChannelV10(t *testing.T) {
	op := NewChannelProvider(map[string]*cb.Capability{})
	assert.NoError(t, op.Supported())
	assert.True(t, op.MSPVersion() == msp.MSPv1_0)
}

func TestChannelV11(t *testing.T) {
	op := NewChannelProvider(map[string]*cb.Capability{
		ChannelV1_1: {},
	})
	assert.NoError(t, op.Supported())
	assert.True(t, op.MSPVersion() == msp.MSPv1_1)
}

func TestChannelV13(t *testing.T) {
	op := NewChannelProvider(map[string]*cb.Capability{
		ChannelV1_1: {},
		ChannelV1_3: {},
	})
	assert.NoError(t, op.Supported())
	assert.True(t, op.MSPVersion() == msp.MSPv1_3)
}
