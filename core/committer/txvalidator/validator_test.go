
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


package txvalidator_test

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	ctxt "github.com/hyperledger/fabric/common/configtx/test"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	ledger2 "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/common/mocks/scc"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/committer/txvalidator/mocks"
	"github.com/hyperledger/fabric/core/committer/txvalidator/testdata"
	ccp "github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	"github.com/hyperledger/fabric/core/handlers/validation/builtin"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	lutils "github.com/hyperledger/fabric/core/ledger/util"
	mocktxvalidator "github.com/hyperledger/fabric/core/mocks/txvalidator"
	mocks2 "github.com/hyperledger/fabric/discovery/support/mocks"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	mb "github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/peer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/semaphore"
)

func signedByAnyMember(ids []string) []byte {
	p := cauthdsl.SignedByAnyMember(ids)
	return utils.MarshalOrPanic(p)
}

func setupLedgerAndValidator(t *testing.T) (ledger.PeerLedger, txvalidator.Validator) {
	plugin := &mocks.Plugin{}
	plugin.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	plugin.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	return setupLedgerAndValidatorExplicit(t, &mockconfig.MockApplicationCapabilities{}, plugin)
}

func preV12Capabilities() *mockconfig.MockApplicationCapabilities {
	return &mockconfig.MockApplicationCapabilities{}
}

func v12Capabilities() *mockconfig.MockApplicationCapabilities {
	return &mockconfig.MockApplicationCapabilities{V1_2ValidationRv: true, PrivateChannelDataRv: true}
}

func v13Capabilities() *mockconfig.MockApplicationCapabilities {
	return &mockconfig.MockApplicationCapabilities{V1_2ValidationRv: true, PrivateChannelDataRv: true, V1_3ValidationRv: true, KeyLevelEndorsementRv: true}
}

func fabTokenCapabilities() *mockconfig.MockApplicationCapabilities {
	return &mockconfig.MockApplicationCapabilities{V1_2ValidationRv: true, FabTokenRv: true}
}

func setupLedgerAndValidatorExplicit(t *testing.T, cpb *mockconfig.MockApplicationCapabilities, plugin validation.Plugin) (ledger.PeerLedger, txvalidator.Validator) {
	return setupLedgerAndValidatorExplicitWithMSP(t, cpb, plugin, nil)
}

func setupLedgerAndValidatorWithPreV12Capabilities(t *testing.T) (ledger.PeerLedger, txvalidator.Validator) {
	return setupLedgerAndValidatorWithCapabilities(t, preV12Capabilities())
}

func setupLedgerAndValidatorWithV12Capabilities(t *testing.T) (ledger.PeerLedger, txvalidator.Validator) {
	return setupLedgerAndValidatorWithCapabilities(t, v12Capabilities())
}

func setupLedgerAndValidatorWithV13Capabilities(t *testing.T) (ledger.PeerLedger, txvalidator.Validator) {
	return setupLedgerAndValidatorWithCapabilities(t, v13Capabilities())
}

func setupLedgerAndValidatorWithFabTokenCapabilities(t *testing.T) (ledger.PeerLedger, txvalidator.Validator) {
	return setupLedgerAndValidatorWithCapabilities(t, fabTokenCapabilities())
}

func setupLedgerAndValidatorWithCapabilities(t *testing.T, c *mockconfig.MockApplicationCapabilities) (ledger.PeerLedger, txvalidator.Validator) {
	mspmgr := &mocks2.MSPManager{}
	idThatSatisfiesPrincipal := &mocks2.Identity{}
	idThatSatisfiesPrincipal.SatisfiesPrincipalReturns(nil)
	idThatSatisfiesPrincipal.GetIdentifierReturns(&msp.IdentityIdentifier{})
	mspmgr.DeserializeIdentityReturns(idThatSatisfiesPrincipal, nil)

	return setupLedgerAndValidatorExplicitWithMSP(t, c, &builtin.DefaultValidation{}, mspmgr)
}

func setupLedgerAndValidatorExplicitWithMSP(t *testing.T, cpb *mockconfig.MockApplicationCapabilities, plugin validation.Plugin, mspMgr msp.MSPManager) (ledger.PeerLedger, txvalidator.Validator) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/validatortest")
	ledgermgmt.InitializeTestEnv()
	gb, err := ctxt.MakeGenesisBlock("TestLedger")
	assert.NoError(t, err)
	theLedger, err := ledgermgmt.CreateLedger(gb)
	assert.NoError(t, err)
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: theLedger, ACVal: cpb, MSPManagerVal: mspMgr}, semaphore.NewWeighted(10)}
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	pm := &mocks.PluginMapper{}
	factory := &mocks.PluginFactory{}
	pm.On("PluginFactoryByName", txvalidator.PluginName("vscc")).Return(factory)
	factory.On("New").Return(plugin)

	theValidator := txvalidator.NewTxValidator("", vcs, mp, pm)

	return theLedger, theValidator
}

func createRWset(t *testing.T, ccnames ...string) []byte {
	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	for _, ccname := range ccnames {
		rwsetBuilder.AddToWriteSet(ccname, "key", []byte("value"))
	}
	rwset, err := rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	rwsetBytes, err := rwset.GetPubSimulationBytes()
	return rwsetBytes
}

func getProposalWithType(ccID string, pType common.HeaderType) (*peer.Proposal, error) {
	cis := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: ccID, Version: ccVersion},
			Input:       &peer.ChaincodeInput{Args: [][]byte{[]byte("func")}},
			Type:        peer.ChaincodeSpec_GOLANG}}

	proposal, _, err := utils.CreateProposalFromCIS(pType, util.GetTestChainID(), cis, signerSerialized)
	return proposal, err
}

const ccVersion = "1.0"

func getEnvWithType(ccID string, event []byte, res []byte, pType common.HeaderType, t *testing.T) *common.Envelope {
//得到一个玩具建议
	prop, err := getProposalWithType(ccID, pType)
	assert.NoError(t, err)

	response := &peer.Response{Status: 200}

//签署以获得提案响应
	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response, res, event, &peer.ChaincodeID{Name: ccID, Version: ccVersion}, nil, signer)
	assert.NoError(t, err)

//根据该提议和背书进行交易
	tx, err := utils.CreateSignedTx(prop, signer, presp)
	assert.NoError(t, err)

	return tx
}

func getEnv(ccID string, event []byte, res []byte, t *testing.T) *common.Envelope {
	return getEnvWithType(ccID, event, res, common.HeaderType_ENDORSER_TRANSACTION, t)
}

func getEnvWithSigner(ccID string, event []byte, res []byte, sig msp.SigningIdentity, t *testing.T) *common.Envelope {
//得到一个玩具建议
	pType := common.HeaderType_ENDORSER_TRANSACTION
	cis := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: ccID, Version: ccVersion},
			Input:       &peer.ChaincodeInput{Args: [][]byte{[]byte("func")}},
			Type:        peer.ChaincodeSpec_GOLANG,
		},
	}

	sID, err := sig.Serialize()
	assert.NoError(t, err)
	prop, _, err := utils.CreateProposalFromCIS(pType, "foochain", cis, sID)
	assert.NoError(t, err)

	response := &peer.Response{Status: 200}

//签署以获得提案响应
	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response, res, event, &peer.ChaincodeID{Name: ccID, Version: ccVersion}, nil, sig)
	assert.NoError(t, err)

//根据该提议和背书进行交易
	tx, err := utils.CreateSignedTx(prop, sig, presp)
	assert.NoError(t, err)

	return tx
}

func getTokenTx(t *testing.T) *common.Envelope {
	transactionData := &token.TokenTransaction{
		Action: &token.TokenTransaction_PlainAction{
			PlainAction: &token.PlainTokenAction{
				Data: &token.PlainTokenAction_PlainImport{
					PlainImport: &token.PlainImport{
						Outputs: []*token.PlainOutput{
							{Owner: []byte("owner-1"), Type: "TOK1", Quantity: 111},
							{Owner: []byte("owner-2"), Type: "TOK2", Quantity: 222},
						},
					},
				},
			},
		},
	}
	tdBytes, err := proto.Marshal(transactionData)
	assert.NoError(t, err)

	signerBytes, err := signer.Serialize()
	assert.NoError(t, err)
	nonce := []byte{0, 1, 2, 3, 4}
	txID, err := utils.ComputeTxID(nonce, signerBytes)
	assert.NoError(t, err)

	hdr := &common.Header{
		SignatureHeader: utils.MarshalOrPanic(
			&common.SignatureHeader{
				Creator: signerBytes,
				Nonce:   nonce,
			},
		),
		ChannelHeader: utils.MarshalOrPanic(
			&common.ChannelHeader{
				Type: int32(common.HeaderType_TOKEN_TRANSACTION),
				TxId: txID,
			},
		),
	}

//根据该提议和背书进行交易
//创建交易记录
	taa := &peer.TransactionAction{Header: hdr.SignatureHeader, Payload: tdBytes}
	taas := make([]*peer.TransactionAction, 1)
	taas[0] = taa
	tx := &peer.Transaction{Actions: taas}

//序列化Tx
	txBytes, err := utils.GetBytesTransaction(tx)
	assert.NoError(t, err)

//创建有效载荷
	payl := &common.Payload{Header: hdr, Data: txBytes}
	paylBytes, err := utils.GetBytesPayload(payl)
	assert.NoError(t, err)

//签署有效载荷
	sig, err := signer.Sign(paylBytes)
	assert.NoError(t, err)

//这是信封
	return &common.Envelope{Payload: paylBytes, Signature: sig}
}

func putCCInfoWithVSCCAndVer(theLedger ledger.PeerLedger, ccname, vscc, ver string, policy []byte, t *testing.T) {
	cd := &ccp.ChaincodeData{
		Name:    ccname,
		Version: ver,
		Vscc:    vscc,
		Policy:  policy,
	}

	cdbytes := utils.MarshalOrPanic(cd)

	txid := util.GenerateUUID()
	simulator, err := theLedger.NewTxSimulator(txid)
	assert.NoError(t, err)
	simulator.SetState("lscc", ccname, cdbytes)
	simulator.Done()

	simRes, err := simulator.GetTxSimulationResults()
	assert.NoError(t, err)
	pubSimulationBytes, err := simRes.GetPubSimulationBytes()
	assert.NoError(t, err)
	bcInfo, err := theLedger.GetBlockchainInfo()
	assert.NoError(t, err)
	block0 := testutil.ConstructBlock(t, 1, bcInfo.CurrentBlockHash, [][]byte{pubSimulationBytes}, true)
	err = theLedger.CommitWithPvtData(&ledger.BlockAndPvtData{
		Block: block0,
	})
	assert.NoError(t, err)
}

func putSBEP(theLedger ledger.PeerLedger, cc, key string, policy []byte, t *testing.T) {
	vpMetadataKey := peer.MetaDataKeys_VALIDATION_PARAMETER.String()
	txid := util.GenerateUUID()
	simulator, err := theLedger.NewTxSimulator(txid)
	assert.NoError(t, err)
	simulator.SetStateMetadata(cc, key, map[string][]byte{vpMetadataKey: policy})
	simulator.SetState(cc, key, []byte("I am a man who walks alone"))
	simulator.Done()

	simRes, err := simulator.GetTxSimulationResults()
	assert.NoError(t, err)
	pubSimulationBytes, err := simRes.GetPubSimulationBytes()
	assert.NoError(t, err)
	bcInfo, err := theLedger.GetBlockchainInfo()
	assert.NoError(t, err)
	block0 := testutil.ConstructBlock(t, 2, bcInfo.CurrentBlockHash, [][]byte{pubSimulationBytes}, true)
	err = theLedger.CommitWithPvtData(&ledger.BlockAndPvtData{
		Block: block0,
	})
	assert.NoError(t, err)
}

func putCCInfo(theLedger ledger.PeerLedger, ccname string, policy []byte, t *testing.T) {
	putCCInfoWithVSCCAndVer(theLedger, ccname, "vscc", ccVersion, policy, t)
}

func assertInvalid(block *common.Block, t *testing.T, code peer.TxValidationCode) {
	txsFilter := lutils.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	assert.True(t, txsFilter.IsInvalid(0))
	assert.True(t, txsFilter.IsSetTo(0, code))
}

func assertValid(block *common.Block, t *testing.T) {
	txsFilter := lutils.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	assert.False(t, txsFilter.IsInvalid(0))
}

func TestInvokeBadRWSet(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeBadRWSet(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeBadRWSet(t, l, v)
	})
}

func testInvokeBadRWSet(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	tx := getEnv(ccID, nil, []byte("barf"), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 1}}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_BAD_RWSET)
}

func TestInvokeNoPolicy(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNoPolicy(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNoPolicy(t, l, v)
	})
}

func testInvokeNoPolicy(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, nil, t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_INVALID_OTHER_REASON)
}

func TestInvokeOK(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOK(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOK(t, l, v)
	})
}

func testInvokeOK(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertValid(b, t)
}

func TestInvokeNoRWSet(t *testing.T) {
	plugin := &mocks.Plugin{}
	plugin.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	t.Run("Pre-1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicit(t, preV12Capabilities(), plugin)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		ccID := "mycc"

		putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

		tx := getEnv(ccID, nil, createRWset(t), t)
		b := &common.Block{
			Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
			Header: &common.BlockHeader{},
		}

		err := v.Validate(b)
		assert.NoError(t, err)
		assertValid(b, t)
	})

	mspmgr := &mocks2.MSPManager{}
	idThatSatisfiesPrincipal := &mocks2.Identity{}
	idThatSatisfiesPrincipal.SatisfiesPrincipalReturns(errors.New("principal not satisfied"))
	idThatSatisfiesPrincipal.GetIdentifierReturns(&msp.IdentityIdentifier{})
	mspmgr.DeserializeIdentityReturns(idThatSatisfiesPrincipal, nil)

//不需要为以前的测试用例定义验证行为，因为1.2之前的版本我们不验证事务
//没有写入设置。
	t.Run("Post-1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, v12Capabilities(), &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNoRWSet(t, l, v)
	})

//在这里，我们测试如果我们有1.3的能力，我们仍然拒绝只包含
//如果不符合背书政策，请阅读
	t.Run("Post-1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, v13Capabilities(), &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNoRWSet(t, l, v)
	})
}

func testInvokeNoRWSet(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE)
}

//并行验证测试的序列化实体模拟
type mockSI struct {
	SerializedID []byte
	MspID        string
	SatPrinError error
}

func (msi *mockSI) ExpiresAt() time.Time {
	return time.Now()
}

func (msi *mockSI) GetIdentifier() *msp.IdentityIdentifier {
	return &msp.IdentityIdentifier{
		Mspid: msi.MspID,
		Id:    "",
	}
}

func (msi *mockSI) GetMSPIdentifier() string {
	return msi.MspID
}

func (msi *mockSI) Validate() error {
	return nil
}

func (msi *mockSI) GetOrganizationalUnits() []*msp.OUIdentifier {
	return nil
}

func (msi *mockSI) Anonymous() bool {
	return false
}

func (msi *mockSI) Verify(msg []byte, sig []byte) error {
	return nil
}

func (msi *mockSI) Serialize() ([]byte, error) {
	sid := &mb.SerializedIdentity{
		Mspid:   msi.MspID,
		IdBytes: msi.SerializedID,
	}
	sidBytes := utils.MarshalOrPanic(sid)
	return sidBytes, nil
}

func (msi *mockSI) SatisfiesPrincipal(principal *mb.MSPPrincipal) error {
	return msi.SatPrinError
}

func (msi *mockSI) Sign(msg []byte) ([]byte, error) {
	return msg, nil
}

func (msi *mockSI) GetPublicVersion() msp.Identity {
	return msi
}

//用于并行验证测试的MSP模拟
type mockMSP struct {
	ID           msp.Identity
	SatPrinError error
	MspID        string
}

func (fake *mockMSP) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	return fake.ID, nil
}

func (fake *mockMSP) IsWellFormed(identity *mb.SerializedIdentity) error {
	return nil
}
func (fake *mockMSP) Setup(config *mb.MSPConfig) error {
	return nil
}

func (fake *mockMSP) GetVersion() msp.MSPVersion {
	return msp.MSPv1_3
}

func (fake *mockMSP) GetType() msp.ProviderType {
	return msp.FABRIC
}

func (fake *mockMSP) GetIdentifier() (string, error) {
	return fake.MspID, nil
}

func (fake *mockMSP) GetSigningIdentity(identifier *msp.IdentityIdentifier) (msp.SigningIdentity, error) {
	return nil, nil
}

func (fake *mockMSP) GetDefaultSigningIdentity() (msp.SigningIdentity, error) {
	return nil, nil
}

func (fake *mockMSP) GetTLSRootCerts() [][]byte {
	return nil
}

func (fake *mockMSP) GetTLSIntermediateCerts() [][]byte {
	return nil
}

func (fake *mockMSP) Validate(id msp.Identity) error {
	return nil
}

func (fake *mockMSP) SatisfiesPrincipal(id msp.Identity, principal *mb.MSPPrincipal) error {
	return fake.SatPrinError
}

//具有大量事务和SBE依赖关系的块的并行验证
func TestParallelValidation(t *testing.T) {
//块中的事务数
	txCnt := 100

//创建两个MSP来控制策略评估结果，其中一个MSP返回satisfiesprincipal（）上的错误
	msp1 := &mockMSP{
		ID: &mockSI{
			MspID:        "Org1",
			SerializedID: []byte("signer0"),
			SatPrinError: nil,
		},
		SatPrinError: nil,
		MspID:        "Org1",
	}
	msp2 := &mockMSP{
		ID: &mockSI{
			MspID:        "Org2",
			SerializedID: []byte("signer1"),
			SatPrinError: errors.New("nope"),
		},
		SatPrinError: errors.New("nope"),
		MspID:        "Org2",
	}
	mgmt.GetManagerForChain("foochain")
	mgr := mgmt.GetManagerForChain("foochain")
	mgr.Setup([]msp.MSP{msp1, msp2})

	vpKey := pb.MetaDataKeys_VALIDATION_PARAMETER.String()

	l, v := setupLedgerAndValidatorExplicitWithMSP(t, &mockconfig.MockApplicationCapabilities{V1_2ValidationRv: true, V1_3ValidationRv: true}, &builtin.DefaultValidation{}, mgr)
	defer ledgermgmt.CleanupTestEnv()
	defer l.Close()

	ccID := "mycc"

	policy := cauthdsl.SignedByMspPeer("Org1")
	polBytes := utils.MarshalOrPanic(policy)
	putCCInfo(l, ccID, polBytes, t)

//创建多个Txe
	blockData := make([][]byte, 0, txCnt)
	col := "col1"
	sigID0 := &mockSI{
		SerializedID: []byte("signer0"),
		MspID:        "Org1",
	}
	sigID1 := &mockSI{
		SerializedID: []byte("signer1"),
		MspID:        "Org2",
	}
	for txNum := 0; txNum < txCnt; txNum++ {
		var sig msp.SigningIdentity
//根据txnum为tx-kvs键创建rwset
		key := strconv.Itoa(txNum % 10)
		rwsetBuilder := rwsetutil.NewRWSetBuilder()
//选择要执行的操作：读取/修改值或ep
		switch uint(txNum / 10) {
		case 0:
//设置键的值（有效）
			rwsetBuilder.AddToWriteSet(ccID, key, []byte("value1"))
			sig = sigID0
		case 1:
//设置密钥的ep（无效，因为org2的msp返回principal不满足）
			metadata := make(map[string][]byte)
			metadata[vpKey] = signedByAnyMember([]string{"SampleOrg"})
			rwsetBuilder.AddToMetadataWriteSet(ccID, key, metadata)
			sig = sigID1
		case 2:
//设置键的值（有效，因为之前的ep更改无效）
			rwsetBuilder.AddToWriteSet(ccID, key, []byte("value2"))
			sig = sigID0
		case 3:
//设置密钥的ep（有效）
			metadata := make(map[string][]byte)
			metadata[vpKey] = signedByAnyMember([]string{"Org2"})
			rwsetBuilder.AddToMetadataWriteSet(ccID, key, metadata)
			sig = sigID0
		case 4:
//设置键的值（无效，因为之前的ep更改有效）
			rwsetBuilder.AddToWriteSet(ccID, key, []byte("value3"))
			sig = &mockSI{
				SerializedID: []byte("signer0"),
				MspID:        "Org1",
			}
//对私有数据执行相同的Txes
		case 5:
//设置键的值（有效）
			rwsetBuilder.AddToPvtAndHashedWriteSet(ccID, col, key, []byte("value1"))
			sig = sigID0
		case 6:
//设置密钥的ep（无效，因为org2的msp返回principal不满足）
			metadata := make(map[string][]byte)
			metadata[vpKey] = signedByAnyMember([]string{"SampleOrg"})
			rwsetBuilder.AddToHashedMetadataWriteSet(ccID, col, key, metadata)
			sig = sigID1
		case 7:
//设置键的值（有效，因为之前的ep更改无效）
			rwsetBuilder.AddToPvtAndHashedWriteSet(ccID, col, key, []byte("value2"))
			sig = sigID0
		case 8:
//设置密钥的ep（有效）
			metadata := make(map[string][]byte)
			metadata[vpKey] = signedByAnyMember([]string{"Org2"})
			rwsetBuilder.AddToHashedMetadataWriteSet(ccID, col, key, metadata)
			sig = sigID0
		case 9:
//设置键的值（无效，因为之前的ep更改有效）
			rwsetBuilder.AddToPvtAndHashedWriteSet(ccID, col, key, []byte("value3"))
			sig = sigID0
		}
		rwset, err := rwsetBuilder.GetTxSimulationResults()
		assert.NoError(t, err)
		rwsetBytes, err := rwset.GetPubSimulationBytes()
		tx := getEnvWithSigner(ccID, nil, rwsetBytes, sig, t)
		blockData = append(blockData, utils.MarshalOrPanic(tx))
	}

//从所有的TXE组装块
	b := &common.Block{Data: &common.BlockData{Data: blockData}, Header: &common.BlockHeader{Number: uint64(txCnt)}}

//验证块
	err := v.Validate(b)
	assert.NoError(t, err)

//块元数据数组位置，用于存储无效事务的序列化位数组筛选器
	txsFilter := lutils.TxValidationFlags(b.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
//TX有效性
	for txNum := 0; txNum < txCnt; txNum += 1 {
		switch uint(txNum / 10) {
		case 1:
			fallthrough
		case 4:
			fallthrough
		case 6:
			fallthrough
		case 9:
			assert.True(t, txsFilter.IsInvalid(txNum))
		default:
			assert.False(t, txsFilter.IsInvalid(txNum))
		}
	}
}

func TestChaincodeEvent(t *testing.T) {
	t.Run("PreV1.2", func(t *testing.T) {
		t.Run("MisMatchedName", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithPreV12Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			ccID := "mycc"

			putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

			tx := getEnv(ccID, utils.MarshalOrPanic(&peer.ChaincodeEvent{ChaincodeId: "wrong"}), createRWset(t), t)
			b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

			err := v.Validate(b)
			assert.NoError(t, err)
			assertValid(b, t)
		})

		t.Run("BadBytes", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithPreV12Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			ccID := "mycc"

			putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

			tx := getEnv(ccID, []byte("garbage"), createRWset(t), t)
			b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

			err := v.Validate(b)
			assert.NoError(t, err)
			assertValid(b, t)
		})

		t.Run("GoodPath", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithPreV12Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			ccID := "mycc"

			putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

			tx := getEnv(ccID, utils.MarshalOrPanic(&peer.ChaincodeEvent{ChaincodeId: ccID}), createRWset(t), t)
			b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

			err := v.Validate(b)
			assert.NoError(t, err)
			assertValid(b, t)
		})
	})

	t.Run("PostV1.2", func(t *testing.T) {
		t.Run("MisMatchedName", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithV12Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			testCCEventMismatchedName(t, l, v)
		})

		t.Run("BadBytes", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithV12Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			testCCEventBadBytes(t, l, v)
		})

		t.Run("GoodPath", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithV12Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			testCCEventGoodPath(t, l, v)
		})
	})

	t.Run("V1.3", func(t *testing.T) {
		t.Run("MisMatchedName", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithV13Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			testCCEventMismatchedName(t, l, v)
		})

		t.Run("BadBytes", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithV13Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			testCCEventBadBytes(t, l, v)
		})

		t.Run("GoodPath", func(t *testing.T) {
			l, v := setupLedgerAndValidatorWithV13Capabilities(t)
			defer ledgermgmt.CleanupTestEnv()
			defer l.Close()

			testCCEventGoodPath(t, l, v)
		})
	})
}

func testCCEventMismatchedName(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, utils.MarshalOrPanic(&peer.ChaincodeEvent{ChaincodeId: "wrong"}), createRWset(t), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
assert.NoError(t, err) //TODO，转换测试，以便检查错误文本是否无效\其他\原因
	assertInvalid(b, t, peer.TxValidationCode_INVALID_OTHER_REASON)
}

func testCCEventBadBytes(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, []byte("garbage"), createRWset(t), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
assert.NoError(t, err) //TODO，转换测试，以便检查错误文本是否无效\其他\原因
	assertInvalid(b, t, peer.TxValidationCode_INVALID_OTHER_REASON)
}

func testCCEventGoodPath(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, utils.MarshalOrPanic(&peer.ChaincodeEvent{ChaincodeId: ccID}), createRWset(t), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertValid(b, t)
}

func TestInvokeOKPvtDataOnly(t *testing.T) {
	mspmgr := &mocks2.MSPManager{}
	idThatSatisfiesPrincipal := &mocks2.Identity{}
	idThatSatisfiesPrincipal.SatisfiesPrincipalReturns(errors.New("principal not satisfied"))
	idThatSatisfiesPrincipal.GetIdentifierReturns(&msp.IdentityIdentifier{})
	mspmgr.DeserializeIdentityReturns(idThatSatisfiesPrincipal, nil)

	t.Run("V1.2", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, v12Capabilities(), &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKPvtDataOnly(t, l, v)
	})

	t.Run("V1.3", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, v13Capabilities(), &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKPvtDataOnly(t, l, v)
	})
}

func testInvokeOKPvtDataOnly(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToPvtAndHashedWriteSet(ccID, "mycollection", "somekey", nil)
	rwset, err := rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	rwsetBytes, err := rwset.GetPubSimulationBytes()
	assert.NoError(t, err)

	tx := getEnv(ccID, nil, rwsetBytes, t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err = v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE)
}

func TestInvokeOKMetaUpdateOnly(t *testing.T) {
	mspmgr := &mocks2.MSPManager{}
	idThatSatisfiesPrincipal := &mocks2.Identity{}
	idThatSatisfiesPrincipal.SatisfiesPrincipalReturns(errors.New("principal not satisfied"))
	idThatSatisfiesPrincipal.GetIdentifierReturns(&msp.IdentityIdentifier{})
	mspmgr.DeserializeIdentityReturns(idThatSatisfiesPrincipal, nil)

	t.Run("V1.2", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, &mockconfig.MockApplicationCapabilities{V1_2ValidationRv: true, PrivateChannelDataRv: true}, &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKMetaUpdateOnly(t, l, v)
	})

	t.Run("V1.3", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, &mockconfig.MockApplicationCapabilities{V1_3ValidationRv: true, V1_2ValidationRv: true, PrivateChannelDataRv: true}, &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKMetaUpdateOnly(t, l, v)
	})
}

func testInvokeOKMetaUpdateOnly(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToMetadataWriteSet(ccID, "somekey", map[string][]byte{})
	rwset, err := rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	rwsetBytes, err := rwset.GetPubSimulationBytes()
	assert.NoError(t, err)

	tx := getEnv(ccID, nil, rwsetBytes, t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err = v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE)
}

func TestInvokeOKPvtMetaUpdateOnly(t *testing.T) {
	mspmgr := &mocks2.MSPManager{}
	idThatSatisfiesPrincipal := &mocks2.Identity{}
	idThatSatisfiesPrincipal.SatisfiesPrincipalReturns(errors.New("principal not satisfied"))
	idThatSatisfiesPrincipal.GetIdentifierReturns(&msp.IdentityIdentifier{})
	mspmgr.DeserializeIdentityReturns(idThatSatisfiesPrincipal, nil)

	t.Run("V1.2", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, &mockconfig.MockApplicationCapabilities{V1_2ValidationRv: true, PrivateChannelDataRv: true}, &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKPvtMetaUpdateOnly(t, l, v)
	})

	t.Run("V1.3", func(t *testing.T) {
		l, v := setupLedgerAndValidatorExplicitWithMSP(t, &mockconfig.MockApplicationCapabilities{V1_3ValidationRv: true, V1_2ValidationRv: true, PrivateChannelDataRv: true}, &builtin.DefaultValidation{}, mspmgr)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKPvtMetaUpdateOnly(t, l, v)
	})
}

func testInvokeOKPvtMetaUpdateOnly(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToHashedMetadataWriteSet(ccID, "mycollection", "somekey", map[string][]byte{})
	rwset, err := rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	rwsetBytes, err := rwset.GetPubSimulationBytes()
	assert.NoError(t, err)

	tx := getEnv(ccID, nil, rwsetBytes, t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err = v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE)
}

func TestInvokeOKSCC(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKSCC(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeOKSCC(t, l, v)
	})
}

func testInvokeOKSCC(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	cds := utils.MarshalOrPanic(&peer.ChaincodeDeploymentSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			Type:        peer.ChaincodeSpec_GOLANG,
			ChaincodeId: &peer.ChaincodeID{Name: "cc", Version: "ver"},
			Input:       &peer.ChaincodeInput{},
		},
	})
	cis := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: "lscc", Version: ccVersion},
			Input:       &peer.ChaincodeInput{Args: [][]byte{[]byte("deploy"), []byte(util.GetTestChainID()), cds}},
			Type:        peer.ChaincodeSpec_GOLANG}}

	prop, _, err := utils.CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, util.GetTestChainID(), cis, signerSerialized)
	assert.NoError(t, err)
	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	rwsetBuilder.AddToWriteSet("lscc", "cc", utils.MarshalOrPanic(&ccp.ChaincodeData{Name: "cc", Version: "ver", InstantiationPolicy: cauthdsl.MarshaledAcceptAllPolicy}))
	rwset, err := rwsetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	rwsetBytes, err := rwset.GetPubSimulationBytes()
	assert.NoError(t, err)
	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, &peer.Response{Status: 200}, rwsetBytes, nil, &peer.ChaincodeID{Name: "lscc", Version: ccVersion}, nil, signer)
	assert.NoError(t, err)
	tx, err := utils.CreateSignedTx(prop, signer, presp)
	assert.NoError(t, err)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 1}}

	err = v.Validate(b)
	assert.NoError(t, err)
	assertValid(b, t)
}

func TestInvokeNOKWritesToLSCC(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKWritesToLSCC(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKWritesToLSCC(t, l, v)
	})
}

func testInvokeNOKWritesToLSCC(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID, "lscc"), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ILLEGAL_WRITESET)
}

func TestInvokeNOKWritesToESCC(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKWritesToESCC(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKWritesToESCC(t, l, v)
	})
}

func testInvokeNOKWritesToESCC(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID, "escc"), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ILLEGAL_WRITESET)
}

func TestInvokeNOKWritesToNotExt(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKWritesToNotExt(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKWritesToNotExt(t, l, v)
	})
}

func testInvokeNOKWritesToNotExt(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID, "notext"), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ILLEGAL_WRITESET)
}

func TestInvokeNOKInvokesNotExt(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKInvokesNotExt(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKInvokesNotExt(t, l, v)
	})
}

func testInvokeNOKInvokesNotExt(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "notext"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ILLEGAL_WRITESET)
}

func TestInvokeNOKInvokesEmptyCCName(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKInvokesEmptyCCName(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKInvokesEmptyCCName(t, l, v)
	})
}

func testInvokeNOKInvokesEmptyCCName(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := ""

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_INVALID_OTHER_REASON)
}

func TestInvokeNOKExpiredCC(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKExpiredCC(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKExpiredCC(t, l, v)
	})
}

func testInvokeNOKExpiredCC(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfoWithVSCCAndVer(l, ccID, "vscc", "badversion", signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_EXPIRED_CHAINCODE)
}

func TestInvokeNOKBogusActions(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKBogusActions(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKBogusActions(t, l, v)
	})
}

func testInvokeNOKBogusActions(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, []byte("barf"), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_BAD_RWSET)
}

func TestInvokeNOKCCDoesntExist(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKCCDoesntExist(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKCCDoesntExist(t, l, v)
	})
}

func testInvokeNOKCCDoesntExist(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_INVALID_OTHER_REASON)
}

func TestInvokeNOKVSCCUnspecified(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKVSCCUnspecified(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNOKVSCCUnspecified(t, l, v)
	})
}

func testInvokeNOKVSCCUnspecified(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	ccID := "mycc"

	putCCInfoWithVSCCAndVer(l, ccID, "", ccVersion, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_INVALID_OTHER_REASON)
}

func TestInvokeNoBlock(t *testing.T) {
	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNoBlock(t, l, v)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		testInvokeNoBlock(t, l, v)
	})
}

func testInvokeNoBlock(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) {
	err := v.Validate(&common.Block{
		Data:   &common.BlockData{Data: [][]byte{}},
		Header: &common.BlockHeader{},
	})
	assert.NoError(t, err)
}

func TestValidateTxWithStateBasedEndorsement(t *testing.T) {

//场景：我们验证一个写入密钥“key”的事务。这把钥匙
//具有无法满足的基于州的认可政策，而
//此事务满足链码认可策略。
//当我们使用1.2功能运行时，我们希望事务
//成功验证，而当我们使用1.3功能运行时，
//由于SBEP不合格，预计验证将失败。
//注意，在实践中不应该出现这种情况，因为
//能力也决定了诚实的同行是否会支持
//设置基于状态的认可策略的链代码。不过，测试
//很有价值，因为它显示了受基于状态的事务影响的方式
//背书政策由1.3验证处理，并且
//被忽略了1.2。

	t.Run("1.2Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV12Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		err, b := validateTxWithStateBasedEndorsement(t, l, v)

		assert.NoError(t, err)
		assertValid(b, t)
	})

	t.Run("1.3Capability", func(t *testing.T) {
		l, v := setupLedgerAndValidatorWithV13Capabilities(t)
		defer ledgermgmt.CleanupTestEnv()
		defer l.Close()

		err, b := validateTxWithStateBasedEndorsement(t, l, v)

		assert.NoError(t, err)
		assertInvalid(b, t, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE)
	})
}

func validateTxWithStateBasedEndorsement(t *testing.T, l ledger.PeerLedger, v txvalidator.Validator) (error, *common.Block) {
	ccID := "mycc"

	putCCInfoWithVSCCAndVer(l, ccID, "vscc", ccVersion, signedByAnyMember([]string{"SampleOrg"}), t)
	putSBEP(l, ccID, "key", cauthdsl.MarshaledRejectAllPolicy, t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 3}}

	err := v.Validate(b)

	return err, b
}

func TestTokenValidTransaction(t *testing.T) {
	t.Skip("Skipping TestTokenValidTransaction until token transaction is enabled after v1.4")
	l, v := setupLedgerAndValidatorWithFabTokenCapabilities(t)
	defer ledgermgmt.CleanupTestEnv()
	defer l.Close()

	tx := getTokenTx(t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 1}}

	err := v.Validate(b)
	assert.NoError(t, err)
	assertValid(b, t)
}

func TestTokenCapabilityNotEnabled(t *testing.T) {
	l, v := setupLedgerAndValidatorWithPreV12Capabilities(t)
	defer ledgermgmt.CleanupTestEnv()
	defer l.Close()

	tx := getTokenTx(t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 1}}

	err := v.Validate(b)

	assertion := assert.New(t)
//我们希望没有验证错误，因为我们只是将Tx标记为无效
	assertion.NoError(err)

//我们希望Tx无效，因为TxID重复
	txsfltr := lutils.TxValidationFlags(b.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	assertion.True(txsfltr.IsInvalid(0))
	assertion.True(txsfltr.Flag(0) == peer.TxValidationCode_UNKNOWN_TX_TYPE)
}

func TestTokenDuplicateTxId(t *testing.T) {
	t.Skip("Skipping TestTokenDuplicateTxId until token transaction is enabled after v1.4")
	theLedger := new(mockLedger)
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: theLedger, ACVal: fabTokenCapabilities()}, semaphore.NewWeighted(10)}
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	pm := &mocks.PluginMapper{}
	validator := txvalidator.NewTxValidator("", vcs, mp, pm)

	tx := getTokenTx(t)
	theLedger.On("GetTransactionByID", mock.Anything).Return(&peer.ProcessedTransaction{}, nil)

	b := testutil.NewBlock([]*common.Envelope{tx}, 0, nil)

	err := validator.Validate(b)

	assertion := assert.New(t)
//我们希望没有验证错误，因为我们只是将Tx标记为无效
	assertion.NoError(err)

//我们希望Tx无效，因为TxID重复
	txsfltr := lutils.TxValidationFlags(b.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	assertion.True(txsfltr.IsInvalid(0))
	assertion.True(txsfltr.Flag(0) == peer.TxValidationCode_DUPLICATE_TXID)
}

//用于测试分类帐的Mockledger结构
//失败，因此利用模拟
//图书馆需要模拟分类账
//能够访问状态数据库
type mockLedger struct {
	mock.Mock
}

//GetTransactionByID按UD返回事务
func (m *mockLedger) GetTransactionByID(txID string) (*peer.ProcessedTransaction, error) {
	args := m.Called(txID)
	return args.Get(0).(*peer.ProcessedTransaction), args.Error(1)
}

//GetBlockByHash使用其哈希值返回块
func (m *mockLedger) GetBlockByHash(blockHash []byte) (*common.Block, error) {
	args := m.Called(blockHash)
	return args.Get(0).(*common.Block), nil
}

//GetBlockByXid给定的事务ID返回块事务是用提交的
func (m *mockLedger) GetBlockByTxID(txID string) (*common.Block, error) {
	args := m.Called(txID)
	return args.Get(0).(*common.Block), nil
}

//gettxvalidationcodebytxid返回give tx的验证代码
func (m *mockLedger) GetTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	args := m.Called(txID)
	return args.Get(0).(peer.TxValidationCode), nil
}

//newtxSimulator创建新的事务模拟器
func (m *mockLedger) NewTxSimulator(txid string) (ledger.TxSimulator, error) {
	args := m.Called()
	return args.Get(0).(ledger.TxSimulator), nil
}

//NewQueryExecutor创建查询执行器
func (m *mockLedger) NewQueryExecutor() (ledger.QueryExecutor, error) {
	args := m.Called()
	return args.Get(0).(ledger.QueryExecutor), nil
}

//NewHistoryQueryExecutor历史查询执行器
func (m *mockLedger) NewHistoryQueryExecutor() (ledger.HistoryQueryExecutor, error) {
	args := m.Called()
	return args.Get(0).(ledger.HistoryQueryExecutor), nil
}

//getpvtdata和blockbynum检索pvt数据和块
func (m *mockLedger) GetPvtDataAndBlockByNum(blockNum uint64, filter ledger.PvtNsCollFilter) (*ledger.BlockAndPvtData, error) {
	args := m.Called()
	return args.Get(0).(*ledger.BlockAndPvtData), nil
}

//getpvtdatabynum检索pvt数据
func (m *mockLedger) GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	args := m.Called()
	return args.Get(0).([]*ledger.TxPvtData), nil
}

//commitWithpvtData在原子操作中提交块和相应的pvt数据
func (m *mockLedger) CommitWithPvtData(pvtDataAndBlock *ledger.BlockAndPvtData) error {
	return nil
}

//pugeprivatedata清除私有数据
func (m *mockLedger) PurgePrivateData(maxBlockNumToRetain uint64) error {
	return nil
}

//privatedataminblocknum返回保留的最低认可块高度
func (m *mockLedger) PrivateDataMinBlockNum() (uint64, error) {
	return 0, nil
}

//使用策略修剪
func (m *mockLedger) Prune(policy ledger2.PrunePolicy) error {
	return nil
}

func (m *mockLedger) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	args := m.Called()
	return args.Get(0).(*common.BlockchainInfo), nil
}

func (m *mockLedger) GetBlockByNumber(blockNumber uint64) (*common.Block, error) {
	args := m.Called(blockNumber)
	return args.Get(0).(*common.Block), nil
}

func (m *mockLedger) GetBlocksIterator(startBlockNumber uint64) (ledger2.ResultsIterator, error) {
	args := m.Called(startBlockNumber)
	return args.Get(0).(ledger2.ResultsIterator), nil
}

func (m *mockLedger) Close() {

}

func (m *mockLedger) Commit(block *common.Block) error {
	return nil
}

//getconfigHistoryRetriever返回configHistoryRetriever
func (m *mockLedger) GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error) {
	args := m.Called()
	return args.Get(0).(ledger.ConfigHistoryRetriever), nil
}

func (m *mockLedger) CommitPvtDataOfOldBlocks(blockPvtData []*ledger.BlockPvtData) ([]*ledger.PvtdataHashMismatch, error) {
	return nil, nil
}

func (m *mockLedger) GetMissingPvtDataTracker() (ledger.MissingPvtDataTracker, error) {
	args := m.Called()
	return args.Get(0).(ledger.MissingPvtDataTracker), nil
}

//mock query executor查询执行器的mock，
//需要模拟无法访问状态数据库，例如
//由于数据库故障不可能
//查询状态，例如，如果要查询
//VSCC信息和数据库的LCCC不可用，我们希望
//停止验证块并失败提交过程
//一个错误。
type mockQueryExecutor struct {
	mock.Mock
}

func (exec *mockQueryExecutor) GetState(namespace string, key string) ([]byte, error) {
	args := exec.Called(namespace, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (exec *mockQueryExecutor) GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error) {
	args := exec.Called(namespace, keys)
	return args.Get(0).([][]byte), args.Error(1)
}

func (exec *mockQueryExecutor) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ledger2.ResultsIterator, error) {
	args := exec.Called(namespace, startKey, endKey)
	return args.Get(0).(ledger2.ResultsIterator), args.Error(1)
}

func (exec *mockQueryExecutor) GetStateRangeScanIteratorWithMetadata(namespace, startKey, endKey string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	args := exec.Called(namespace, startKey, endKey, metadata)
	return args.Get(0).(ledger.QueryResultsIterator), args.Error(1)
}

func (exec *mockQueryExecutor) ExecuteQuery(namespace, query string) (ledger2.ResultsIterator, error) {
	args := exec.Called(namespace)
	return args.Get(0).(ledger2.ResultsIterator), args.Error(1)
}

func (exec *mockQueryExecutor) ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	args := exec.Called(namespace, query, metadata)
	return args.Get(0).(ledger.QueryResultsIterator), args.Error(1)
}

func (exec *mockQueryExecutor) GetPrivateData(namespace, collection, key string) ([]byte, error) {
	args := exec.Called(namespace, collection, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (exec *mockQueryExecutor) GetPrivateDataMetadataByHash(namespace, collection string, keyhash []byte) (map[string][]byte, error) {
	args := exec.Called(namespace, collection, keyhash)
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (exec *mockQueryExecutor) GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([][]byte, error) {
	args := exec.Called(namespace, collection, keys)
	return args.Get(0).([][]byte), args.Error(1)
}

func (exec *mockQueryExecutor) GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (ledger2.ResultsIterator, error) {
	args := exec.Called(namespace, collection, startKey, endKey)
	return args.Get(0).(ledger2.ResultsIterator), args.Error(1)
}

func (exec *mockQueryExecutor) ExecuteQueryOnPrivateData(namespace, collection, query string) (ledger2.ResultsIterator, error) {
	args := exec.Called(namespace, collection, query)
	return args.Get(0).(ledger2.ResultsIterator), args.Error(1)
}

func (exec *mockQueryExecutor) Done() {
}

func (exec *mockQueryExecutor) GetStateMetadata(namespace, key string) (map[string][]byte, error) {
	return nil, nil
}

func (exec *mockQueryExecutor) GetPrivateDataMetadata(namespace, collection, key string) (map[string][]byte, error) {
	return nil, nil
}

func createCustomSupportAndLedger(t *testing.T) (*mocktxvalidator.Support, ledger.PeerLedger) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/validatortest")
	ledgermgmt.InitializeTestEnv()
	gb, err := ctxt.MakeGenesisBlock("TestLedger")
	assert.NoError(t, err)
	l, err := ledgermgmt.CreateLedger(gb)
	assert.NoError(t, err)

	identity := &mocks2.Identity{}
	identity.GetIdentifierReturns(&msp.IdentityIdentifier{
		Mspid: "SampleOrg",
		Id:    "foo",
	})
	mspManager := &mocks2.MSPManager{}
	mspManager.DeserializeIdentityReturns(identity, nil)
	support := &mocktxvalidator.Support{LedgerVal: l, ACVal: &mockconfig.MockApplicationCapabilities{}, MSPManagerVal: mspManager}
	return support, l
}

func TestDynamicCapabilitiesAndMSP(t *testing.T) {
	factory := &mocks.PluginFactory{}
	factory.On("New").Return(testdata.NewSampleValidationPlugin(t))
	pm := &mocks.PluginMapper{}
	pm.On("PluginFactoryByName", txvalidator.PluginName("vscc")).Return(factory)

	support, l := createCustomSupportAndLedger(t)
	defer l.Close()

	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{support, semaphore.NewWeighted(10)}
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()

	v := txvalidator.NewTxValidator("", vcs, mp, pm)

	ccID := "mycc"

	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

//对块执行验证
	err := v.Validate(b)
	assert.NoError(t, err)
	assertValid(b, t)
//记录调用功能和MSP管理器的次数
	capabilityInvokeCount := support.CapabilitiesInvokeCount()
	mspManagerInvokeCount := support.MSPManagerInvokeCount()

//执行另一个验证过程，并确保其有效
	err = v.Validate(b)
	assert.NoError(t, err)
	assertValid(b, t)

//确保两次从支援中取回能力，
//这证明了每次从支持中动态地检索能力
	assert.Equal(t, 2*capabilityInvokeCount, support.CapabilitiesInvokeCount())
//确保两次从支持中检索到MSP管理器，
//这证明了每次从支持中动态地检索MSP管理器
	assert.Equal(t, 2*mspManagerInvokeCount, support.MSPManagerInvokeCount())
}

//testlegrisnoavailable为以下场景模拟并提供测试，
//基于FAB-535。测试检查验证路径，该路径预期
//当试图从lccc中查找vscc时，db将不可用，因此
//事务验证必须失败。在这种情况下，结果应该是
//验证块方法返回的错误和事务处理
//必须停止。假设有明确的故障和错误指示。
//从函数调用返回。
func TestLedgerIsNoAvailable(t *testing.T) {
	theLedger := new(mockLedger)
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: theLedger, ACVal: &mockconfig.MockApplicationCapabilities{}}, semaphore.NewWeighted(10)}
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	pm := &mocks.PluginMapper{}
	validator := txvalidator.NewTxValidator("", vcs, mp, pm)

	ccID := "mycc"
	tx := getEnv(ccID, nil, createRWset(t, ccID), t)

	theLedger.On("GetTransactionByID", mock.Anything).Return(&peer.ProcessedTransaction{}, ledger.NotFoundInIndexErr(""))

	queryExecutor := new(mockQueryExecutor)
	queryExecutor.On("GetState", mock.Anything, mock.Anything).Return([]byte{}, errors.New("Unable to connect to DB"))
	theLedger.On("NewQueryExecutor", mock.Anything).Return(queryExecutor, nil)

	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := validator.Validate(b)

	assertion := assert.New(t)
//我们假定得到一个错误，它表明我们不能提交该块。
	assertion.Error(err)
//检测到的错误类型为vsccinfo lookupfailureerror
	assertion.NotNil(err.(*commonerrors.VSCCInfoLookupFailureError))
}

func TestLedgerIsNotAvailableForCheckingTxidDuplicate(t *testing.T) {
	theLedger := new(mockLedger)
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: theLedger, ACVal: &mockconfig.MockApplicationCapabilities{}}, semaphore.NewWeighted(10)}
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	pm := &mocks.PluginMapper{}
	validator := txvalidator.NewTxValidator("", vcs, mp, pm)

	ccID := "mycc"
	tx := getEnv(ccID, nil, createRWset(t, ccID), t)

	theLedger.On("GetTransactionByID", mock.Anything).Return(&peer.ProcessedTransaction{}, errors.New("Unable to connect to DB"))

	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := validator.Validate(b)

	assertion := assert.New(t)
//我们预计会出现验证错误，因为分类帐没有准备好告诉我们是否存在具有该ID的Tx
	assertion.Error(err)
}

func TestDuplicateTxId(t *testing.T) {
	theLedger := new(mockLedger)
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: theLedger, ACVal: &mockconfig.MockApplicationCapabilities{}}, semaphore.NewWeighted(10)}
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	pm := &mocks.PluginMapper{}
	validator := txvalidator.NewTxValidator("", vcs, mp, pm)

	ccID := "mycc"
	tx := getEnv(ccID, nil, createRWset(t, ccID), t)

	theLedger.On("GetTransactionByID", mock.Anything).Return(&peer.ProcessedTransaction{}, nil)

	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	err := validator.Validate(b)

	assertion := assert.New(t)
//我们希望没有验证错误，因为我们只是将Tx标记为无效
	assertion.NoError(err)

//我们希望Tx无效，因为TxID重复
	txsfltr := lutils.TxValidationFlags(b.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	assertion.True(txsfltr.IsInvalid(0))
	assertion.True(txsfltr.Flag(0) == peer.TxValidationCode_DUPLICATE_TXID)
}

func TestValidationInvalidEndorsing(t *testing.T) {
	theLedger := new(mockLedger)
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: theLedger, ACVal: &mockconfig.MockApplicationCapabilities{}}, semaphore.NewWeighted(10)}
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	pm := &mocks.PluginMapper{}
	factory := &mocks.PluginFactory{}
	plugin := &mocks.Plugin{}
	factory.On("New").Return(plugin)
	plugin.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	plugin.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("invalid tx"))
	pm.On("PluginFactoryByName", txvalidator.PluginName("vscc")).Return(factory)
	validator := txvalidator.NewTxValidator("", vcs, mp, pm)

	ccID := "mycc"
	tx := getEnv(ccID, nil, createRWset(t, ccID), t)

	theLedger.On("GetTransactionByID", mock.Anything).Return(&peer.ProcessedTransaction{}, ledger.NotFoundInIndexErr(""))

	cd := &ccp.ChaincodeData{
		Name:    ccID,
		Version: ccVersion,
		Vscc:    "vscc",
		Policy:  signedByAnyMember([]string{"SampleOrg"}),
	}

	cdbytes := utils.MarshalOrPanic(cd)

	queryExecutor := new(mockQueryExecutor)
	queryExecutor.On("GetState", "lscc", ccID).Return(cdbytes, nil)
	theLedger.On("NewQueryExecutor", mock.Anything).Return(queryExecutor, nil)

	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

//保留默认回调
	err := validator.Validate(b)
//恢复默认回调
	assert.NoError(t, err)
	assertInvalid(b, t, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE)
}

func createMockLedger(t *testing.T, ccID string) *mockLedger {
	l := new(mockLedger)
	l.On("GetTransactionByID", mock.Anything).Return(&peer.ProcessedTransaction{}, ledger.NotFoundInIndexErr(""))
	cd := &ccp.ChaincodeData{
		Name:    ccID,
		Version: ccVersion,
		Vscc:    "vscc",
		Policy:  signedByAnyMember([]string{"SampleOrg"}),
	}

	cdbytes := utils.MarshalOrPanic(cd)
	queryExecutor := new(mockQueryExecutor)
	queryExecutor.On("GetState", "lscc", ccID).Return(cdbytes, nil)
	l.On("NewQueryExecutor", mock.Anything).Return(queryExecutor, nil)
	return l
}

func TestValidationPluginExecutionError(t *testing.T) {
	plugin := &mocks.Plugin{}
	plugin.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	l, v := setupLedgerAndValidatorExplicit(t, &mockconfig.MockApplicationCapabilities{}, plugin)
	defer ledgermgmt.CleanupTestEnv()
	defer l.Close()

	ccID := "mycc"
	putCCInfo(l, ccID, signedByAnyMember([]string{"SampleOrg"}), t)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	plugin.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&validation.ExecutionFailureError{
		Reason: "I/O error",
	})

	err := v.Validate(b)
	executionErr := err.(*commonerrors.VSCCExecutionFailureError)
	assert.Contains(t, executionErr.Error(), "I/O error")
}

func TestValidationPluginNotFound(t *testing.T) {
	ccID := "mycc"
	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	l := createMockLedger(t, ccID)
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: l, ACVal: &mockconfig.MockApplicationCapabilities{}}, semaphore.NewWeighted(10)}

	b := &common.Block{
		Data:   &common.BlockData{Data: [][]byte{utils.MarshalOrPanic(tx)}},
		Header: &common.BlockHeader{},
	}

	pm := &mocks.PluginMapper{}
	pm.On("PluginFactoryByName", txvalidator.PluginName("vscc")).Return(nil)
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	validator := txvalidator.NewTxValidator("", vcs, mp, pm)
	err := validator.Validate(b)
	executionErr := err.(*commonerrors.VSCCExecutionFailureError)
	assert.Contains(t, executionErr.Error(), "plugin with name vscc wasn't found")
}

var signer msp.SigningIdentity

var signerSerialized []byte

func TestMain(m *testing.M) {
	msptesttools.LoadMSPSetupForTesting()

	var err error
	signer, err = mgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		fmt.Println("Could not get signer")
		os.Exit(-1)
		return
	}

	signerSerialized, err = signer.Serialize()
	if err != nil {
		fmt.Println("Could not serialize identity")
		os.Exit(-1)
		return
	}

	os.Exit(m.Run())
}
