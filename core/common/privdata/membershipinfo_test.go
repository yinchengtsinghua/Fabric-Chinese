
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


package privdata

import (
	"testing"

	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestMembershipInfoProvider(t *testing.T) {
	peerSelfSignedData := common.SignedData{
		Identity:  []byte("peer0"),
		Signature: []byte{1, 2, 3},
		Data:      []byte{4, 5, 6},
	}

	identityDeserializer := func(chainID string) msp.IdentityDeserializer {
		return &mockDeserializer{}
	}

//验证成员身份提供程序是否返回true
	membershipProvider := NewMembershipInfoProvider(peerSelfSignedData, identityDeserializer)
	res, err := membershipProvider.AmMemberOf("test1", getAccessPolicy([]string{"peer0", "peer1"}))
	assert.True(t, res)
	assert.Nil(t, err)

//验证成员身份提供程序是否返回false
	res, err = membershipProvider.AmMemberOf("test1", getAccessPolicy([]string{"peer2", "peer3"}))
	assert.False(t, res)
	assert.Nil(t, err)

//验证成员身份提供程序返回nil，并且当收集策略配置为nil时出错
	res, err = membershipProvider.AmMemberOf("test1", nil)
	assert.False(t, res)
	assert.Error(t, err)
	assert.Equal(t, "Collection policy config is nil", err.Error())
}

func getAccessPolicy(signers []string) *common.CollectionPolicyConfig {
	var data [][]byte
	for _, signer := range signers {
		data = append(data, []byte(signer))
	}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), data)
	return createCollectionPolicyConfig(policyEnvelope)
}
