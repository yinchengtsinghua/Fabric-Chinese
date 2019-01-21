
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


package v13

import (
	"fmt"
	"regexp"

	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms/ccmetadata"
	. "github.com/hyperledger/fabric/core/common/validation/statebased"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/capabilities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/identities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/policies"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/state"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

var logger = flogging.MustGetLogger("vscc")

var validCollectionNameRegex = regexp.MustCompile(ccmetadata.AllowedCharsCollectionName)

//go:generate mokery-dir../../api/capabilities/-name capabilities-case underline-output mocks/
//go:generate mokery-dir../../api/state/-name statefetcher-case underline-output mocks/
//go:generate mokery-dir../../api/identities/-name identitydeserializer-case underline-output mocks/
//go:generate mokery-dir../../api/policies/-name policyEvaluator-case underline-output mocks/
//去：生成mokery-dir。-name statebasedvalidator-case下划线-输出模拟/

//新建创建默认VSCC的新实例
//通常，每个对等机只调用一次。
func New(c Capabilities, s StateFetcher, d IdentityDeserializer, pe PolicyEvaluator) *Validator {
	vpmgr := &KeyLevelValidationParameterManagerImpl{StateFetcher: s}
	sbv := NewKeyLevelValidator(pe, vpmgr)

	return &Validator{
		capabilities:        c,
		stateFetcher:        s,
		deserializer:        d,
		policyEvaluator:     pe,
		stateBasedValidator: sbv,
	}
}

//验证程序实现默认事务验证策略，
//检查读写设置和背书的正确性
//针对作为参数提供给
//每次调用
type Validator struct {
	deserializer        IdentityDeserializer
	capabilities        Capabilities
	stateFetcher        StateFetcher
	policyEvaluator     PolicyEvaluator
	stateBasedValidator StateBasedValidator
}

type validationArtifacts struct {
	rwset        []byte
	prp          []byte
	endorsements []*peer.Endorsement
	chdr         *common.ChannelHeader
	env          *common.Envelope
	payl         *common.Payload
	cap          *peer.ChaincodeActionPayload
}

func (vscc *Validator) extractValidationArtifacts(
	block *common.Block,
	txPosition int,
	actionPosition int,
) (*validationArtifacts, error) {
//拿到信封…
	env, err := utils.GetEnvelopeFromBlock(block.Data.Data[txPosition])
	if err != nil {
		logger.Errorf("VSCC error: GetEnvelope failed, err %s", err)
		return nil, err
	}

//…有效载荷…
	payl, err := utils.GetPayload(env)
	if err != nil {
		logger.Errorf("VSCC error: GetPayload failed, err %s", err)
		return nil, err
	}

	chdr, err := utils.UnmarshalChannelHeader(payl.Header.ChannelHeader)
	if err != nil {
		return nil, err
	}

//验证有效负载类型
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		logger.Errorf("Only Endorser Transactions are supported, provided type %d", chdr.Type)
		err = fmt.Errorf("Only Endorser Transactions are supported, provided type %d", chdr.Type)
		return nil, err
	}

//…交易…
	tx, err := utils.GetTransaction(payl.Data)
	if err != nil {
		logger.Errorf("VSCC error: GetTransaction failed, err %s", err)
		return nil, err
	}

	cap, err := utils.GetChaincodeActionPayload(tx.Actions[actionPosition].Payload)
	if err != nil {
		logger.Errorf("VSCC error: GetChaincodeActionPayload failed, err %s", err)
		return nil, err
	}

	pRespPayload, err := utils.GetProposalResponsePayload(cap.Action.ProposalResponsePayload)
	if err != nil {
		err = fmt.Errorf("GetProposalResponsePayload error %s", err)
		return nil, err
	}
	if pRespPayload.Extension == nil {
		err = fmt.Errorf("nil pRespPayload.Extension")
		return nil, err
	}
	respPayload, err := utils.GetChaincodeAction(pRespPayload.Extension)
	if err != nil {
		err = fmt.Errorf("GetChaincodeAction error %s", err)
		return nil, err
	}

	return &validationArtifacts{
		rwset:        respPayload.Results,
		prp:          cap.Action.ProposalResponsePayload,
		endorsements: cap.Action.Endorsements,
		chdr:         chdr,
		env:          env,
		payl:         payl,
		cap:          cap,
	}, nil
}

//validate验证与带有背书的交易相对应的给定信封
//以其序列化形式提供的策略。
//请注意，在块中存在依赖项的情况下，例如Tx修改背书策略
//对于键A和Tx_n+1修改键A的值，validate（Tx_n+1）将阻止，直到validate（Tx_n）
//已解决。如果使用有限数量的Goroutine进行并行验证，请确保
//它们按升序分配给事务。
func (vscc *Validator) Validate(
	block *common.Block,
	namespace string,
	txPosition int,
	actionPosition int,
	policyBytes []byte,
) commonerrors.TxValidationError {
	vscc.stateBasedValidator.PreValidate(uint64(txPosition), block)

	va, err := vscc.extractValidationArtifacts(block, txPosition, actionPosition)
	if err != nil {
		vscc.stateBasedValidator.PostValidate(namespace, block.Header.Number, uint64(txPosition), err)
		return policyErr(err)
	}

	txverr := vscc.stateBasedValidator.Validate(
		namespace,
		block.Header.Number,
		uint64(txPosition),
		va.rwset,
		va.prp,
		policyBytes,
		va.endorsements,
	)
	if txverr != nil {
		logger.Errorf("VSCC error: stateBasedValidator.Validate failed, err %s", txverr)
		vscc.stateBasedValidator.PostValidate(namespace, block.Header.Number, uint64(txPosition), txverr)
		return txverr
	}

//执行特定于LSCC的额外验证
	if namespace == "lscc" {
		logger.Debugf("VSCC info: doing special validation for LSCC")
		err := vscc.ValidateLSCCInvocation(va.chdr.ChannelId, va.env, va.cap, va.payl, vscc.capabilities)
		if err != nil {
			logger.Errorf("VSCC error: ValidateLSCCInvocation failed, err %s", err)
			vscc.stateBasedValidator.PostValidate(namespace, block.Header.Number, uint64(txPosition), err)
			return err
		}
	}

	vscc.stateBasedValidator.PostValidate(namespace, block.Header.Number, uint64(txPosition), nil)
	return nil
}

func policyErr(err error) *commonerrors.VSCCEndorsementPolicyError {
	return &commonerrors.VSCCEndorsementPolicyError{
		Err: err,
	}
}
