
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


package cc_test

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/core/cclifecycle"
	"github.com/hyperledger/fabric/core/cclifecycle/mocks"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestChaincodeInspection(t *testing.T) {
	acceptAll := func(cc chaincode.Metadata) bool {
		return true
	}

	policy := &common.SignaturePolicyEnvelope{}
	policy.Rule = &common.SignaturePolicy{
		Type: &common.SignaturePolicy_NOutOf_{
			NOutOf: &common.SignaturePolicy_NOutOf{
				N: int32(2),
			},
		},
	}
	policyBytes, _ := proto.Marshal(policy)

	cc1 := &ccprovider.ChaincodeData{Name: "cc1", Version: "1.0", Policy: policyBytes, Id: []byte{42}}
	cc1Bytes, _ := proto.Marshal(cc1)
	acceptSpecificCCId := func(cc chaincode.Metadata) bool {
//
		return bytes.Equal(cc.Id, []byte{24})
	}

	var corruptBytes []byte
	corruptBytes = append(corruptBytes, cc1Bytes...)
	corruptBytes = append(corruptBytes, 0)

	cc2 := &ccprovider.ChaincodeData{Name: "cc2", Version: "1.1"}
	cc2Bytes, _ := proto.Marshal(cc2)

	tests := []struct {
		name              string
		expected          chaincode.MetadataSet
		returnedCCBytes   []byte
		queryErr          error
		queriedChaincodes []string
		filter            func(cc chaincode.Metadata) bool
	}{
		{
			name:              "No chaincodes installed",
			expected:          nil,
			filter:            acceptAll,
			queriedChaincodes: []string{},
		},
		{
			name:              "Failure on querying the ledger",
			queryErr:          errors.New("failed querying ledger"),
			filter:            acceptAll,
			queriedChaincodes: []string{"cc1", "cc2"},
			expected:          nil,
		},
		{
			name:              "The data in LSCC is corrupt",
			returnedCCBytes:   corruptBytes,
			filter:            acceptAll,
			queriedChaincodes: []string{"cc1"},
			expected:          nil,
		},
		{
			name:              "The chaincode is listed as a different chaincode in the ledger",
			returnedCCBytes:   cc2Bytes,
			filter:            acceptAll,
			queriedChaincodes: []string{"cc1"},
			expected:          nil,
		},
		{
			name:              "Chaincode doesn't exist",
			returnedCCBytes:   nil,
			filter:            acceptAll,
			queriedChaincodes: []string{"cc1"},
			expected:          nil,
		},
		{
			name:              "Select a specific chaincode",
			returnedCCBytes:   cc1Bytes,
			queriedChaincodes: []string{"cc1", "cc2"},
			filter:            acceptSpecificCCId,
		},
		{
			name:              "Green path",
			returnedCCBytes:   cc1Bytes,
			filter:            acceptAll,
			queriedChaincodes: []string{"cc1", "cc2"},
			expected: chaincode.MetadataSet([]chaincode.Metadata{
				{
					Id:      []byte{42},
					Name:    "cc1",
					Version: "1.0",
					Policy:  policyBytes,
				},
				{
					Name:    "cc2",
					Version: "1.1",
				},
			}),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			query := &mocks.Query{}
			query.On("Done")
			query.On("GetState", mock.Anything, mock.Anything).Return(test.returnedCCBytes, test.queryErr).Once()
			query.On("GetState", mock.Anything, mock.Anything).Return(cc2Bytes, nil).Once()
			ccInfo, err := cc.DeployedChaincodes(query, test.filter, false, test.queriedChaincodes...)
			if test.queryErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.expected, ccInfo)
		})
	}
}
