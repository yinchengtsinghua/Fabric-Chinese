
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
	"fmt"
	"reflect"

	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/capabilities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/identities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/policies"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/state"
	"github.com/hyperledger/fabric/core/handlers/validation/builtin/v12"
	"github.com/hyperledger/fabric/core/handlers/validation/builtin/v13"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("vscc")

type DefaultValidationFactory struct {
}

func (*DefaultValidationFactory) New() validation.Plugin {
	return &DefaultValidation{}
}

type DefaultValidation struct {
	Capabilities    Capabilities
	TxValidatorV1_2 TransactionValidator
	TxValidatorV1_3 TransactionValidator
}

//去：生成mokery-dir。-name transactionvalidator-case下划线-输出模拟/
type TransactionValidator interface {
	Validate(block *common.Block, namespace string, txPosition int, actionPosition int, policy []byte) commonerrors.TxValidationError
}

func (v *DefaultValidation) Validate(block *common.Block, namespace string, txPosition int, actionPosition int, contextData ...validation.ContextDatum) error {
	if len(contextData) == 0 {
		logger.Panicf("Expected to receive policy bytes in context data")
	}

	serializedPolicy, isSerializedPolicy := contextData[0].(SerializedPolicy)
	if !isSerializedPolicy {
		logger.Panicf("Expected to receive a serialized policy in the first context data")
	}
	if block == nil || block.Data == nil {
		return errors.New("empty block")
	}
	if txPosition >= len(block.Data.Data) {
		return errors.Errorf("block has only %d transactions, but requested tx at position %d", len(block.Data.Data), txPosition)
	}
	if block.Header == nil {
		return errors.Errorf("no block header")
	}

	var err error
	switch {
	case v.Capabilities.V1_3Validation():
		err = v.TxValidatorV1_3.Validate(block, namespace, txPosition, actionPosition, serializedPolicy.Bytes())

	case v.Capabilities.V1_2Validation():
		fallthrough

	default:
		err = v.TxValidatorV1_2.Validate(block, namespace, txPosition, actionPosition, serializedPolicy.Bytes())
	}

	logger.Debugf("block %d, namespace: %s, tx %d validation results is: %v", block.Header.Number, namespace, txPosition, err)
	return convertErrorTypeOrPanic(err)
}

func convertErrorTypeOrPanic(err error) error {
	if err == nil {
		return nil
	}
	if err, isExecutionError := err.(*commonerrors.VSCCExecutionFailureError); isExecutionError {
		return &validation.ExecutionFailureError{
			Reason: err.Error(),
		}
	}
	if err, isEndorsementError := err.(*commonerrors.VSCCEndorsementPolicyError); isEndorsementError {
		return err
	}
	logger.Panicf("Programming error: The error is %v, of type %v but expected to be either ExecutionFailureError or VSCCEndorsementPolicyError", err, reflect.TypeOf(err))
	return &validation.ExecutionFailureError{Reason: fmt.Sprintf("error of type %v returned from VSCC", reflect.TypeOf(err))}
}

func (v *DefaultValidation) Init(dependencies ...validation.Dependency) error {
	var (
		d  IdentityDeserializer
		c  Capabilities
		sf StateFetcher
		pe PolicyEvaluator
	)
	for _, dep := range dependencies {
		if deserializer, isIdentityDeserializer := dep.(IdentityDeserializer); isIdentityDeserializer {
			d = deserializer
		}
		if capabilities, isCapabilities := dep.(Capabilities); isCapabilities {
			c = capabilities
		}
		if stateFetcher, isStateFetcher := dep.(StateFetcher); isStateFetcher {
			sf = stateFetcher
		}
		if policyEvaluator, isPolicyFetcher := dep.(PolicyEvaluator); isPolicyFetcher {
			pe = policyEvaluator
		}
	}
	if sf == nil {
		return errors.New("stateFetcher not passed in init")
	}
	if d == nil {
		return errors.New("identityDeserializer not passed in init")
	}
	if c == nil {
		return errors.New("capabilities not passed in init")
	}
	if pe == nil {
		return errors.New("policy fetcher not passed in init")
	}

	v.Capabilities = c
	v.TxValidatorV1_2 = v12.New(c, sf, d, pe)
	v.TxValidatorV1_3 = v13.New(c, sf, d, pe)

	return nil
}
