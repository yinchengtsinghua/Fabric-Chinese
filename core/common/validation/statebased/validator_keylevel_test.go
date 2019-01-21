
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


package statebased

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

type mockPolicyEvaluator struct {
	EvaluateRV          error
	EvaluateResByPolicy map[string]error
}

func (m *mockPolicyEvaluator) Evaluate(policyBytes []byte, signatureSet []*common.SignedData) error {
	if res, ok := m.EvaluateResByPolicy[string(policyBytes)]; ok {
		return res
	}

	return m.EvaluateRV
}

func buildBlockWithTxs(txs ...[]byte) *common.Block {
	return &common.Block{
		Header: &common.BlockHeader{
			Number: 1,
		},
		Data: &common.BlockData{
			Data: txs,
		},
	}
}

func buildTXWithRwset(rws []byte) []byte {
	return utils.MarshalOrPanic(&common.Envelope{
		Payload: utils.MarshalOrPanic(
			&common.Payload{
				Data: utils.MarshalOrPanic(
					&pb.Transaction{
						Actions: []*pb.TransactionAction{
							{
								Payload: utils.MarshalOrPanic(&pb.ChaincodeActionPayload{
									Action: &pb.ChaincodeEndorsedAction{
										ProposalResponsePayload: utils.MarshalOrPanic(
											&pb.ProposalResponsePayload{
												Extension: utils.MarshalOrPanic(&pb.ChaincodeAction{Results: rws}),
											},
										),
									},
								}),
							},
						},
					},
				),
			},
		),
	})
}

func rwsetBytes(t *testing.T, cc string) []byte {
	rwsb := rwsetutil.NewRWSetBuilder()
	rwsb.AddToWriteSet(cc, "key", []byte("value"))
	rws := rwsb.GetTxReadWriteSet()
	rwsetbytes, err := rws.ToProtoBytes()
	assert.NoError(t, err)

	return rwsetbytes
}

func TestKeylevelValidation(t *testing.T) {
	t.Parallel()

//Scenario: we validate a transaction that writes
//到包含键级验证参数的键。
//我们模拟策略检查的成功和失败

	vpMetadataKey := pb.MetaDataKeys_VALIDATION_PARAMETER.String()
	mr := &mockState{GetStateMetadataRv: map[string][]byte{vpMetadataKey: []byte("EP")}, GetPrivateDataMetadataByHashRv: map[string][]byte{vpMetadataKey: []byte("EP")}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsb := rwsetBytes(t, "cc")
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	endorsements := []*pb.Endorsement{
		{
			Signature: []byte("signature"),
			Endorser:  []byte("endorser"),
		},
	}

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err := validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), endorsements)
	assert.NoError(t, err)

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), endorsements)
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}

func TestKeylevelValidationPvtData(t *testing.T) {
	t.Parallel()

//场景：我们验证一个写
//到包含键级验证参数的pvt键。
//我们模拟策略检查的成功和失败

	vpMetadataKey := pb.MetaDataKeys_VALIDATION_PARAMETER.String()
	mr := &mockState{GetStateMetadataRv: map[string][]byte{vpMetadataKey: []byte("EP")}, GetPrivateDataMetadataByHashRv: map[string][]byte{vpMetadataKey: []byte("EP")}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsbu := rwsetutil.NewRWSetBuilder()
	rwsbu.AddToPvtAndHashedWriteSet("cc", "coll", "key", []byte("value"))
	rws := rwsbu.GetTxReadWriteSet()
	rwsb, err := rws.ToProtoBytes()
	assert.NoError(t, err)
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}

func TestKeylevelValidationMetaUpdate(t *testing.T) {
	t.Parallel()

//场景：我们验证一个更新的事务
//键的键级验证参数。
//我们模拟策略检查的成功和失败

	vpMetadataKey := pb.MetaDataKeys_VALIDATION_PARAMETER.String()
	mr := &mockState{GetStateMetadataRv: map[string][]byte{vpMetadataKey: []byte("EP")}, GetPrivateDataMetadataByHashRv: map[string][]byte{vpMetadataKey: []byte("EP")}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsbu := rwsetutil.NewRWSetBuilder()
	rwsbu.AddToMetadataWriteSet("cc", "key", map[string][]byte{})
	rws := rwsbu.GetTxReadWriteSet()
	rwsb, err := rws.ToProtoBytes()
	assert.NoError(t, err)
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}

func TestKeylevelValidationPvtMetaUpdate(t *testing.T) {
	t.Parallel()

//场景：我们验证一个更新的事务
//pvt密钥的密钥级别验证参数。
//我们模拟策略检查的成功和失败

	vpMetadataKey := pb.MetaDataKeys_VALIDATION_PARAMETER.String()
	mr := &mockState{GetStateMetadataRv: map[string][]byte{vpMetadataKey: []byte("EP")}, GetPrivateDataMetadataByHashRv: map[string][]byte{vpMetadataKey: []byte("EP")}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsbu := rwsetutil.NewRWSetBuilder()
	rwsbu.AddToHashedMetadataWriteSet("cc", "coll", "key", map[string][]byte{})
	rws := rwsbu.GetTxReadWriteSet()
	rwsb, err := rws.ToProtoBytes()
	assert.NoError(t, err)
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}

func TestKeylevelValidationPolicyRetrievalFailure(t *testing.T) {
	t.Parallel()

//场景：我们验证一个更新的事务
//键的键级验证参数。
//我们模拟无法检索的情况
//分类帐中的验证参数。

	mr := &mockState{GetStateMetadataErr: fmt.Errorf("metadata retrieval failure")}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	validator := NewKeyLevelValidator(&mockPolicyEvaluator{}, pm)

	rwsb := rwsetBytes(t, "cc")
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err := validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCExecutionFailureError{}, err)
}

func TestKeylevelValidationLedgerFailures(t *testing.T) {
//场景：我们验证一个更新的事务
//键的键级验证参数。
//我们模拟无法检索的情况
//来自分类帐的验证参数
//确定性错误和非确定性错误

	rwsb := rwsetBytes(t, "cc")
	prp := []byte("barf")

	t.Run("CollConfigNotDefinedError", func(t *testing.T) {
		mr := &mockState{GetStateMetadataErr: &ledger.CollConfigNotDefinedError{Ns: "mycc"}}
		ms := &mockStateFetcher{FetchStateRv: mr}
		pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
		validator := NewKeyLevelValidator(&mockPolicyEvaluator{}, pm)

		err := validator.Validate("cc", 1, 0, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
		assert.NoError(t, err)
	})

	t.Run("InvalidCollNameError", func(t *testing.T) {
		mr := &mockState{GetStateMetadataErr: &ledger.InvalidCollNameError{Ns: "mycc", Coll: "mycoll"}}
		ms := &mockStateFetcher{FetchStateRv: mr}
		pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
		validator := NewKeyLevelValidator(&mockPolicyEvaluator{}, pm)

		err := validator.Validate("cc", 1, 0, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
		assert.NoError(t, err)
	})

	t.Run("I/O error", func(t *testing.T) {
		mr := &mockState{GetStateMetadataErr: fmt.Errorf("some I/O error")}
		ms := &mockStateFetcher{FetchStateRv: mr}
		pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
		validator := NewKeyLevelValidator(&mockPolicyEvaluator{}, pm)

		err := validator.Validate("cc", 1, 0, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
		assert.Error(t, err)
		assert.IsType(t, &errors.VSCCExecutionFailureError{}, err)
	})
}

func TestCCEPValidation(t *testing.T) {
	t.Parallel()

//场景：我们验证一个没有
//使用国家认可政策触摸任何钥匙；
//我们希望检查正常的CC背书政策。

	mr := &mockState{GetStateMetadataRv: map[string][]byte{}, GetPrivateDataMetadataByHashRv: map[string][]byte{}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsbu := rwsetutil.NewRWSetBuilder()
	rwsbu.AddToWriteSet("cc", "key", []byte("value"))
	rwsbu.AddToWriteSet("cc", "key1", []byte("value"))
	rwsbu.AddToReadSet("cc", "readkey", &version.Height{})
	rwsbu.AddToHashedReadSet("cc", "coll", "readpvtkey", &version.Height{})
	rws := rwsbu.GetTxReadWriteSet()
	rwsb, err := rws.ToProtoBytes()
	assert.NoError(t, err)
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}

func TestCCEPValidationReads(t *testing.T) {
	t.Parallel()

//场景：我们验证一个没有
//使用国家认可政策触摸任何钥匙；
//我们希望检查正常的CC背书政策。

	mr := &mockState{GetStateMetadataRv: map[string][]byte{}, GetPrivateDataMetadataByHashRv: map[string][]byte{}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsbu := rwsetutil.NewRWSetBuilder()
	rwsbu.AddToReadSet("cc", "readkey", &version.Height{})
	rws := rwsbu.GetTxReadWriteSet()
	rwsb, err := rws.ToProtoBytes()
	assert.NoError(t, err)
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}

func TestOnlySBEPChecked(t *testing.T) {
	t.Parallel()

//场景：我们确保只要有一个键
//需要国家支持，我们只检查该政策
//我们不检查CC EP。我们通过设置
//策略评估器模拟为所有策略返回错误
//但以国家为基础，并期待成功的评估

	vpMetadataKey := pb.MetaDataKeys_VALIDATION_PARAMETER.String()
	mr := &mockState{GetStateMetadataRv: map[string][]byte{vpMetadataKey: []byte("SBEP")}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsb := rwsetBytes(t, "cc")
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")
	pe.EvaluateResByPolicy = map[string]error{
		"SBEP": nil,
	}

	err := validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)

//我们还使用具有读和写的读写集进行测试
	rwsbu := rwsetutil.NewRWSetBuilder()
	rwsbu.AddToWriteSet("cc", "key", []byte("value"))
	rwsbu.AddToReadSet("cc", "key", nil)
	rws := rwsbu.GetTxReadWriteSet()
	rwsb, _ = rws.ToProtoBytes()

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)
}

func TestCCEPValidationPvtReads(t *testing.T) {
	t.Parallel()

//场景：我们验证一个没有
//使用国家认可政策触摸任何钥匙；
//我们希望检查正常的CC背书政策。

	mr := &mockState{GetStateMetadataRv: map[string][]byte{}, GetPrivateDataMetadataByHashRv: map[string][]byte{}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	pe := &mockPolicyEvaluator{}
	validator := NewKeyLevelValidator(pe, pm)

	rwsbu := rwsetutil.NewRWSetBuilder()
	rwsbu.AddToHashedReadSet("cc", "coll", "readpvtkey", &version.Height{})
	rws := rwsbu.GetTxReadWriteSet()
	rwsb, err := rws.ToProtoBytes()
	assert.NoError(t, err)
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, fmt.Errorf(""))
	}()

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.NoError(t, err)

	pe.EvaluateRV = fmt.Errorf("policy evaluation error")

	err = validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}

func TestKeylevelValidationFailure(t *testing.T) {
	t.Parallel()

//场景：我们验证一个写
//到包含键级验证参数的键。
//验证失败，因为块包含上一个
//更新密钥级验证参数的事务
//因为那把钥匙是一样的。

	vpMetadataKey := pb.MetaDataKeys_VALIDATION_PARAMETER.String()
	mr := &mockState{GetStateMetadataRv: map[string][]byte{vpMetadataKey: []byte("EP")}, GetPrivateDataMetadataByHashRv: map[string][]byte{vpMetadataKey: []byte("EP")}}
	ms := &mockStateFetcher{FetchStateRv: mr}
	pm := &KeyLevelValidationParameterManagerImpl{StateFetcher: ms}
	validator := NewKeyLevelValidator(&mockPolicyEvaluator{}, pm)

	rwsb := rwsetBytes(t, "cc")
	prp := []byte("barf")
	block := buildBlockWithTxs(buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")), buildTXWithRwset(rwsetUpdatingMetadataFor("cc", "key")))

	validator.PreValidate(1, block)

	go func() {
		validator.PostValidate("cc", 1, 0, nil)
	}()

	err := validator.Validate("cc", 1, 1, rwsb, prp, []byte("CCEP"), []*pb.Endorsement{})
	assert.Error(t, err)
	assert.IsType(t, &errors.VSCCEndorsementPolicyError{}, err)
}
