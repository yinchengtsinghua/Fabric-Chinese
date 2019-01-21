
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


package builtin

import (
	"testing"

	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	. "github.com/hyperledger/fabric/core/handlers/validation/api"
	vmocks "github.com/hyperledger/fabric/core/handlers/validation/builtin/mocks"
	"github.com/hyperledger/fabric/core/handlers/validation/builtin/v12/mocks"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInit(t *testing.T) {
	factory := &DefaultValidationFactory{}
	defValidation := factory.New()

	identityDeserializer := &mocks.IdentityDeserializer{}
	capabilities := &mocks.Capabilities{}
	stateFetcher := &mocks.StateFetcher{}
	polEval := &mocks.PolicyEvaluator{}

	assert.Equal(t, "stateFetcher not passed in init", defValidation.Init(identityDeserializer, capabilities, polEval).Error())
	assert.Equal(t, "identityDeserializer not passed in init", defValidation.Init(capabilities, stateFetcher, polEval).Error())
	assert.Equal(t, "capabilities not passed in init", defValidation.Init(identityDeserializer, stateFetcher, polEval).Error())
	assert.Equal(t, "policy fetcher not passed in init", defValidation.Init(identityDeserializer, capabilities, stateFetcher).Error())

	fullDeps := []Dependency{identityDeserializer, capabilities, stateFetcher, polEval}
	assert.NoError(t, defValidation.Init(fullDeps...))
}

func TestErrorConversion(t *testing.T) {
	validator := &vmocks.TransactionValidator{}
	capabilities := &mocks.Capabilities{}
	validation := &DefaultValidation{
		TxValidatorV1_2: validator,
		Capabilities:    capabilities,
	}
	block := &common.Block{
		Header: &common.BlockHeader{},
		Data: &common.BlockData{
			Data: [][]byte{{}},
		},
	}

	capabilities.On("V1_3Validation").Return(false)
	capabilities.On("V1_2Validation").Return(true)

//场景一：不是*commonErrors.ExecutionFailureError或*commonErrors.vscc背书策略错误的错误
//应该引起恐慌
	validator.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("bla bla")).Once()
	assert.Panics(t, func() {
		validation.Validate(block, "", 0, 0, txvalidator.SerializedPolicy("policy"))
	})

//场景二：非执行错误按原样返回
	validator.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&commonerrors.VSCCEndorsementPolicyError{Err: errors.New("foo")}).Once()
	err := validation.Validate(block, "", 0, 0, txvalidator.SerializedPolicy("policy"))
	assert.Equal(t, (&commonerrors.VSCCEndorsementPolicyError{Err: errors.New("foo")}).Error(), err.Error())

//场景三：执行错误转换为插件错误类型
	validator.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&commonerrors.VSCCExecutionFailureError{Err: errors.New("bar")}).Once()
	err = validation.Validate(block, "", 0, 0, txvalidator.SerializedPolicy("policy"))
	assert.Equal(t, &ExecutionFailureError{Reason: "bar"}, err)

//场景四：没有错误被转发
	validator.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	assert.NoError(t, validation.Validate(block, "", 0, 0, txvalidator.SerializedPolicy("policy")))
}

func TestValidateBadInput(t *testing.T) {
	validator := &vmocks.TransactionValidator{}
	validation := &DefaultValidation{
		TxValidatorV1_2: validator,
	}

//场景一：零块
	validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	err := validation.Validate(nil, "", 0, 0, txvalidator.SerializedPolicy("policy"))
	assert.Equal(t, "empty block", err.Error())

	block := &common.Block{
		Header: &common.BlockHeader{},
		Data: &common.BlockData{
			Data: [][]byte{{}},
		},
	}
//场景二：1个交易块，但位置也在1
	validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	err = validation.Validate(block, "", 1, 0, txvalidator.SerializedPolicy("policy"))
	assert.Equal(t, "block has only 1 transactions, but requested tx at position 1", err.Error())

//场景三：没有标题的块
	validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	err = validation.Validate(&common.Block{
		Data: &common.BlockData{
			Data: [][]byte{{}},
		},
	}, "", 0, 0, txvalidator.SerializedPolicy("policy"))
	assert.Equal(t, "no block header", err.Error())

//方案四：未传递序列化策略
	assert.Panics(t, func() {
		validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
		err = validation.Validate(&common.Block{
			Header: &common.BlockHeader{},
			Data: &common.BlockData{
				Data: [][]byte{{}},
			},
		}, "", 0, 0)
	})

//方案五：传递的策略不是序列化策略
	assert.Panics(t, func() {
		validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
		err = validation.Validate(&common.Block{
			Header: &common.BlockHeader{},
			Data: &common.BlockData{
				Data: [][]byte{{}},
			},
		}, "", 0, 0, []byte("policy"))
	})

}
