
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


package chaincode

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/stretchr/testify/assert"
)

type mockMetadataRetriever struct {
	res *chaincode.Metadata
}

func (r *mockMetadataRetriever) Metadata(channel string, cc string, loadCollections bool) *chaincode.Metadata {
	return r.res
}

func TestSupport(t *testing.T) {
	emptySignaturePolicyEnvelope, _ := proto.Marshal(&common.SignaturePolicyEnvelope{})
	ccmd1 := &chaincode.Metadata{Policy: emptySignaturePolicyEnvelope}
	notEmptySignaturePolicyEnvelope, _ := proto.Marshal(&common.SignaturePolicyEnvelope{
		Rule:       &common.SignaturePolicy{},
		Identities: []*msp.MSPPrincipal{{}},
	})
	ccmd2 := &chaincode.Metadata{Policy: notEmptySignaturePolicyEnvelope}

	tests := []struct {
		name        string
		input       *chaincode.Metadata
		shouldBeNil bool
	}{
		{
			name:        "Nil instantiatedChaincode",
			input:       nil,
			shouldBeNil: true,
		},
		{
			name:        "Invalid policy bytes",
			input:       &chaincode.Metadata{Policy: []byte{1, 2, 3}},
			shouldBeNil: true,
		},
		{
			name:        "Empty signature policy envelope",
			input:       ccmd1,
			shouldBeNil: true,
		},
		{
			name:        "Not Empty signature policy envelope",
			input:       ccmd2,
			shouldBeNil: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			sup := NewDiscoverySupport(&mockMetadataRetriever{res: test.input})
			res := sup.PolicyByChaincode("", "")
			if test.shouldBeNil {
				assert.Nil(t, res)
			} else {
				assert.NotNil(t, res)
			}
		})
	}
}
