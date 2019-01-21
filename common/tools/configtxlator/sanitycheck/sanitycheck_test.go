
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package sanitycheck

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/tools/configtxgen/configtxgentest"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

var (
	insecureConfig  *cb.Config
	singleMSPConfig *cb.Config
)

func init() {
	factory.InitFactories(nil)

	insecureChannelGroup, err := encoder.NewChannelGroup(configtxgentest.Load(genesisconfig.SampleInsecureSoloProfile))
	if err != nil {
		panic(err)
	}
	insecureConfig = &cb.Config{ChannelGroup: insecureChannelGroup}

	singleMSPChannelGroup, err := encoder.NewChannelGroup(configtxgentest.Load(genesisconfig.SampleSingleMSPSoloProfile))
	if err != nil {
		panic(err)
	}
	singleMSPConfig = &cb.Config{ChannelGroup: singleMSPChannelGroup}
}

func TestSimpleCheck(t *testing.T) {
	result, err := Check(insecureConfig)
	assert.NoError(t, err, "Simple empty config")
	assert.Equal(t, &Messages{}, result)
}

func TestOneMSPCheck(t *testing.T) {
	result, err := Check(singleMSPConfig)
	assert.NoError(t, err, "Simple single MSP config")
	assert.Equal(t, &Messages{}, result)
}

func TestEmptyConfigCheck(t *testing.T) {
	result, err := Check(&cb.Config{})
	assert.NoError(t, err, "Simple single MSP config")
	assert.Empty(t, result.ElementErrors)
	assert.Empty(t, result.ElementWarnings)
	assert.NotEmpty(t, result.GeneralErrors)
}

func TestWrongMSPID(t *testing.T) {
	localConfig := proto.Clone(insecureConfig).(*cb.Config)
	policyName := "foo"
	localConfig.ChannelGroup.Groups[channelconfig.OrdererGroupKey].Policies[policyName] = &cb.ConfigPolicy{
		Policy: &cb.Policy{
			Type:  int32(cb.Policy_SIGNATURE),
			Value: utils.MarshalOrPanic(cauthdsl.SignedByMspAdmin("MissingOrg")),
		},
	}
	result, err := Check(localConfig)
	assert.NoError(t, err, "Simple empty config")
	assert.Empty(t, result.GeneralErrors)
	assert.Empty(t, result.ElementErrors)
	assert.Len(t, result.ElementWarnings, 1)
	assert.Equal(t, ".groups."+channelconfig.OrdererGroupKey+".policies."+policyName, result.ElementWarnings[0].Path)
}

func TestCorruptRolePrincipal(t *testing.T) {
	localConfig := proto.Clone(insecureConfig).(*cb.Config)
	policyName := "foo"
	sigPolicy := cauthdsl.SignedByMspAdmin("MissingOrg")
	sigPolicy.Identities[0].Principal = []byte("garbage which corrupts the evaluation")
	localConfig.ChannelGroup.Policies[policyName] = &cb.ConfigPolicy{
		Policy: &cb.Policy{
			Type:  int32(cb.Policy_SIGNATURE),
			Value: utils.MarshalOrPanic(sigPolicy),
		},
	}
	result, err := Check(localConfig)
	assert.NoError(t, err, "Simple empty config")
	assert.Empty(t, result.GeneralErrors)
	assert.Empty(t, result.ElementWarnings)
	assert.Len(t, result.ElementErrors, 1)
	assert.Equal(t, ".policies."+policyName, result.ElementErrors[0].Path)
}

func TestCorruptOUPrincipal(t *testing.T) {
	localConfig := proto.Clone(insecureConfig).(*cb.Config)
	policyName := "foo"
	sigPolicy := cauthdsl.SignedByMspAdmin("MissingOrg")
	sigPolicy.Identities[0].PrincipalClassification = mspprotos.MSPPrincipal_ORGANIZATION_UNIT
	sigPolicy.Identities[0].Principal = []byte("garbage which corrupts the evaluation")
	localConfig.ChannelGroup.Policies[policyName] = &cb.ConfigPolicy{
		Policy: &cb.Policy{
			Type:  int32(cb.Policy_SIGNATURE),
			Value: utils.MarshalOrPanic(sigPolicy),
		},
	}
	result, err := Check(localConfig)
	assert.NoError(t, err, "Simple empty config")
	assert.Empty(t, result.GeneralErrors)
	assert.Empty(t, result.ElementWarnings)
	assert.Len(t, result.ElementErrors, 1)
	assert.Equal(t, ".policies."+policyName, result.ElementErrors[0].Path)
}
