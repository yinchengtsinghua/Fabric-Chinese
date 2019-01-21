
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


package channelconfig_test

import (
	"testing"

	newchannelconfig "github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/tools/configtxgen/configtxgentest"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func TestWithRealConfigtx(t *testing.T) {
	conf := configtxgentest.Load(genesisconfig.SampleSingleMSPSoloProfile)

//没有一个示例配置文件定义应用程序配置节
//在一个创世时期（因为这是个坏主意），但我们把它们结合起来
//这里是为了更好地练习代码。
	conf.Application = &genesisconfig.Application{
		Organizations: []*genesisconfig.Organization{
			conf.Orderer.Organizations[0],
		},
	}
	conf.Application.Organizations[0].AnchorPeers = []*genesisconfig.AnchorPeer{
		{
			Host: "foo",
			Port: 7,
		},
	}
	gb := encoder.New(conf).GenesisBlockForChannel("foo")
	env := utils.ExtractEnvelopeOrPanic(gb, 0)
	_, err := newchannelconfig.NewBundleFromEnvelope(env)
	assert.NoError(t, err)
}
