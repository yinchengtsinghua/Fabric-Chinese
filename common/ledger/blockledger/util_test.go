
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
*/


package blockledger_test

import (
	"testing"

	"github.com/hyperledger/fabric/common/deliver/mock"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestClose(t *testing.T) {
	for _, testCase := range []struct {
		name               string
		status             common.Status
		isIteratorNil      bool
		expectedCloseCount int
	}{
		{
			name:          "nil iterator",
			isIteratorNil: true,
		},
		{
			name:               "Next() fails",
			status:             common.Status_INTERNAL_SERVER_ERROR,
			expectedCloseCount: 1,
		},
		{
			name:               "Next() succeeds",
			status:             common.Status_SUCCESS,
			expectedCloseCount: 1,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var iterator *mock.BlockIterator
			reader := &mock.BlockReader{}
			if !testCase.isIteratorNil {
				iterator = &mock.BlockIterator{}
				iterator.NextReturns(&common.Block{}, testCase.status)
				reader.IteratorReturns(iterator, 1)
			}

			blockledger.GetBlock(reader, 1)
			if !testCase.isIteratorNil {
				assert.Equal(t, testCase.expectedCloseCount, iterator.CloseCallCount())
			}
		})
	}
}
