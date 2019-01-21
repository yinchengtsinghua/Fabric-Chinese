
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

	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestSatisfied(t *testing.T) {
	var capsMap map[string]*cb.Capability
	for _, provider := range []*registry{
		NewChannelProvider(capsMap).registry,
		NewOrdererProvider(capsMap).registry,
		NewApplicationProvider(capsMap).registry,
	} {
		assert.Nil(t, provider.Supported())
	}
}

func TestNotSatisfied(t *testing.T) {
	capsMap := map[string]*cb.Capability{
		"FakeCapability": {},
	}
	for _, provider := range []*registry{
		NewChannelProvider(capsMap).registry,
		NewOrdererProvider(capsMap).registry,
		NewApplicationProvider(capsMap).registry,
	} {
		assert.Error(t, provider.Supported())
	}
}
