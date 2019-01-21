
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


package txvalidator

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/mocks/config"
	util2 "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	ledger2 "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/ledger/util"
	ledgerUtil "github.com/hyperledger/fabric/core/ledger/util"
	mocktxvalidator "github.com/hyperledger/fabric/core/mocks/txvalidator"
	"github.com/hyperledger/fabric/core/mocks/validator"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

func testValidationWithNTXes(t *testing.T, ledger ledger2.PeerLedger, gbHash []byte, nBlocks int) {
	txid := util2.GenerateUUID()
	simulator, _ := ledger.NewTxSimulator(txid)
	simulator.SetState("ns1", "key1", []byte("value1"))
	simulator.SetState("ns1", "key2", []byte("value2"))
	simulator.SetState("ns1", "key3", []byte("value3"))
	simulator.Done()

	simRes, _ := simulator.GetTxSimulationResults()
	pubSimulationResBytes, _ := simRes.GetPubSimulationBytes()
	_, err := testutil.ConstructBytesProposalResponsePayload("v1", pubSimulationResBytes)
	if err != nil {
		t.Fatalf("Could not construct ProposalResponsePayload bytes, err: %s", err)
	}

	mockVsccValidator := &validator.MockVsccValidator{}
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: ledger, ACVal: &config.MockApplicationCapabilities{}}, semaphore.NewWeighted(10)}
	tValidator := &TxValidator{"", vcs, mockVsccValidator}

	bcInfo, _ := ledger.GetBlockchainInfo()
	assert.Equal(t, &common.BlockchainInfo{
		Height: 1, CurrentBlockHash: gbHash, PreviousBlockHash: nil,
	}, bcInfo)

	sr := [][]byte{}
	for i := 0; i < nBlocks; i++ {
		sr = append(sr, pubSimulationResBytes)
	}
	block := testutil.ConstructBlock(t, 1, gbHash, sr, true)

	tValidator.Validate(block)

	txsfltr := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])

	for i := 0; i < nBlocks; i++ {
		assert.True(t, txsfltr.IsSetTo(i, peer.TxValidationCode_VALID))
	}
}

func TestDetectTXIdDuplicates(t *testing.T) {
	txids := []string{"", "1", "2", "3", "", "2", ""}
	txsfltr := ledgerUtil.NewTxValidationFlags(len(txids))
	markTXIdDuplicates(txids, txsfltr)
	assert.True(t, txsfltr.IsSetTo(0, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(1, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(2, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(3, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(4, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(5, peer.TxValidationCode_DUPLICATE_TXID))
	assert.True(t, txsfltr.IsSetTo(6, peer.TxValidationCode_NOT_VALIDATED))

	txids = []string{"", "1", "2", "3", "", "21", ""}
	txsfltr = ledgerUtil.NewTxValidationFlags(len(txids))
	markTXIdDuplicates(txids, txsfltr)
	assert.True(t, txsfltr.IsSetTo(0, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(1, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(2, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(3, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(4, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(5, peer.TxValidationCode_NOT_VALIDATED))
	assert.True(t, txsfltr.IsSetTo(6, peer.TxValidationCode_NOT_VALIDATED))
}

func TestBlockValidationDuplicateTXId(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/txvalidatortest")
	ledgermgmt.InitializeTestEnv()
	defer ledgermgmt.CleanupTestEnv()

	gb, _ := test.MakeGenesisBlock("TestLedger")
	gbHash := gb.Header.Hash()
	ledger, _ := ledgermgmt.CreateLedger(gb)
	defer ledger.Close()

	txid := util2.GenerateUUID()
	simulator, _ := ledger.NewTxSimulator(txid)
	simulator.SetState("ns1", "key1", []byte("value1"))
	simulator.SetState("ns1", "key2", []byte("value2"))
	simulator.SetState("ns1", "key3", []byte("value3"))
	simulator.Done()

	simRes, _ := simulator.GetTxSimulationResults()
	pubSimulationResBytes, _ := simRes.GetPubSimulationBytes()
	_, err := testutil.ConstructBytesProposalResponsePayload("v1", pubSimulationResBytes)
	if err != nil {
		t.Fatalf("Could not construct ProposalResponsePayload bytes, err: %s", err)
	}

	mockVsccValidator := &validator.MockVsccValidator{}
	acv := &config.MockApplicationCapabilities{}
	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: ledger, ACVal: acv}, semaphore.NewWeighted(10)}
	tValidator := &TxValidator{"", vcs, mockVsccValidator}

	bcInfo, _ := ledger.GetBlockchainInfo()
	assert.Equal(t, &common.BlockchainInfo{
		Height: 1, CurrentBlockHash: gbHash, PreviousBlockHash: nil,
	}, bcInfo)

	envs := []*common.Envelope{}
	env, _, err := testutil.ConstructTransaction(t, pubSimulationResBytes, "", true)
	envs = append(envs, env)
	envs = append(envs, env)
	block := testutil.NewBlock(envs, 1, gbHash)

	tValidator.Validate(block)

	txsfltr := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])

	assert.True(t, txsfltr.IsSetTo(0, peer.TxValidationCode_VALID))
	assert.True(t, txsfltr.IsSetTo(1, peer.TxValidationCode_VALID))

	acv.ForbidDuplicateTXIdInBlockRv = true

	tValidator.Validate(block)

	txsfltr = util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])

	assert.True(t, txsfltr.IsSetTo(0, peer.TxValidationCode_VALID))
	assert.True(t, txsfltr.IsSetTo(1, peer.TxValidationCode_DUPLICATE_TXID))
}

func TestBlockValidation(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/txvalidatortest")
	ledgermgmt.InitializeTestEnv()
	defer ledgermgmt.CleanupTestEnv()

	gb, _ := test.MakeGenesisBlock("TestLedger")
	gbHash := gb.Header.Hash()
	ledger, _ := ledgermgmt.CreateLedger(gb)
	defer ledger.Close()

//这里我们用一个Tx测试一个块的有效性。
	testValidationWithNTXes(t, ledger, gbHash, 1)
}

func TestParallelBlockValidation(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/txvalidatortest")
	ledgermgmt.InitializeTestEnv()
	defer ledgermgmt.CleanupTestEnv()

	gb, _ := test.MakeGenesisBlock("TestLedger")
	gbHash := gb.Header.Hash()
	ledger, _ := ledgermgmt.CreateLedger(gb)
	defer ledger.Close()

//这里我们用128个Txes测试一个块的有效性。
	testValidationWithNTXes(t, ledger, gbHash, 128)
}

func TestVeryLargeParallelBlockValidation(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/txvalidatortest")
	ledgermgmt.InitializeTestEnv()
	defer ledgermgmt.CleanupTestEnv()

	gb, _ := test.MakeGenesisBlock("TestLedger")
	gbHash := gb.Header.Hash()
	ledger, _ := ledgermgmt.CreateLedger(gb)
	defer ledger.Close()

//在这里，我们用4096txes测试一个块的有效性，
//这比
//通道中的池和缓冲区
	testValidationWithNTXes(t, ledger, gbHash, 4096)
}

func TestTxValidationFailure_InvalidTxid(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/txvalidatortest")
	ledgermgmt.InitializeTestEnv()
	defer ledgermgmt.CleanupTestEnv()

	gb, _ := test.MakeGenesisBlock("TestLedger")
	ledger, _ := ledgermgmt.CreateLedger(gb)

	defer ledger.Close()

	vcs := struct {
		*mocktxvalidator.Support
		*semaphore.Weighted
	}{&mocktxvalidator.Support{LedgerVal: ledger, ACVal: &config.MockApplicationCapabilities{}}, semaphore.NewWeighted(10)}
	tValidator := &TxValidator{"", vcs, &validator.MockVsccValidator{}}

	mockSigner, err := mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	assert.NoError(t, err)
	mockSignerSerialized, err := mockSigner.Serialize()
	assert.NoError(t, err)

//创建简单的背书交易
	payload := &common.Payload{
		Header: &common.Header{
			ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{
				TxId:      "INVALID TXID!!!",
				Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
				ChannelId: util2.GetTestChainID(),
			}),
			SignatureHeader: utils.MarshalOrPanic(&common.SignatureHeader{
				Nonce:   []byte("nonce"),
				Creator: mockSignerSerialized,
			}),
		},
		Data: []byte("test"),
	}

	payloadBytes, err := proto.Marshal(payload)

//检查编组是否失败
	assert.NoError(t, err)

	sig, err := mockSigner.Sign(payloadBytes)
	assert.NoError(t, err)

//包裹有效载荷
	envelope := &common.Envelope{
		Payload:   payloadBytes,
		Signature: sig,
	}

	envelopeBytes, err := proto.Marshal(envelope)

//检查编组是否失败
	assert.NoError(t, err)

	block := &common.Block{
		Data: &common.BlockData{
//编码交易
			Data: [][]byte{envelopeBytes},
		},
	}

	block.Header = &common.BlockHeader{
		Number:   0,
		DataHash: block.Data.Hash(),
	}

//初始化元数据
	utils.InitBlockMetadata(block)
	txsFilter := util.NewTxValidationFlagsSetValue(len(block.Data.Data), peer.TxValidationCode_VALID)
	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter

//将块提交到分类帐
	ledger.CommitWithPvtData(&ledger2.BlockAndPvtData{
		Block: block,
	})

//验证应使事务失效，
//因为已经承诺了
	tValidator.Validate(block)

	txsfltr := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	assert.True(t, txsfltr.IsInvalid(0))

//我们希望Tx无效，因为TxID不正确
	assert.True(t, txsfltr.Flag(0) == peer.TxValidationCode_BAD_PROPOSAL_TXID)
}

func createCCUpgradeEnvelope(chainID, chaincodeName, chaincodeVersion string, signer msp.SigningIdentity) (*common.Envelope, error) {
	creator, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	spec := &peer.ChaincodeSpec{
		Type: peer.ChaincodeSpec_Type(peer.ChaincodeSpec_Type_value["GOLANG"]),
		ChaincodeId: &peer.ChaincodeID{
			Path:    "github.com/codePath",
			Name:    chaincodeName,
			Version: chaincodeVersion,
		},
	}

	cds := &peer.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: []byte{}}
	prop, _, err := utils.CreateUpgradeProposalFromCDS(chainID, cds, creator, []byte{}, []byte{}, []byte{}, nil)
	if err != nil {
		return nil, err
	}

	proposalResponse := &peer.ProposalResponse{
		Response: &peer.Response{
Status: 200, //已成功背书
		},
		Endorsement: &peer.Endorsement{},
	}

	return utils.CreateSignedTx(prop, signer, proposalResponse)
}

func TestGetTxCCInstance(t *testing.T) {
//设置MSP管理器，以便我们可以签名/验证
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		t.Fatalf("Could not initialize msp, err: %s", err)
	}
	signer, err := mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		t.Fatalf("Could not initialize signer, err: %s", err)
	}

	chainID := util2.GetTestChainID()
	upgradeCCName := "mycc"
	upgradeCCVersion := "v1"

	env, err := createCCUpgradeEnvelope(chainID, upgradeCCName, upgradeCCVersion, signer)
	assert.NoError(t, err)

//从信封中获取有效载荷
	payload, err := utils.GetPayload(env)
	assert.NoError(t, err)

	expectInvokeCCIns := &sysccprovider.ChaincodeInstance{
		ChainID:          chainID,
		ChaincodeName:    "lscc",
		ChaincodeVersion: "",
	}
	expectUpgradeCCIns := &sysccprovider.ChaincodeInstance{
		ChainID:          chainID,
		ChaincodeName:    upgradeCCName,
		ChaincodeVersion: upgradeCCVersion,
	}

	tValidator := &TxValidator{}
	invokeCCIns, upgradeCCIns, err := tValidator.getTxCCInstance(payload)
	if err != nil {
		t.Fatalf("Get chaincode from tx error: %s", err)
	}
	assert.EqualValues(t, expectInvokeCCIns, invokeCCIns)
	assert.EqualValues(t, expectUpgradeCCIns, upgradeCCIns)
}

func TestInvalidTXsForUpgradeCC(t *testing.T) {
	txsChaincodeNames := map[int]*sysccprovider.ChaincodeInstance{
0: {ChainID: "chain0", ChaincodeName: "cc0", ChaincodeVersion: "v0"}, //调用cc0/chain0:v0，不应受到其他链中升级tx的影响。
1: {ChainID: "chain1", ChaincodeName: "cc0", ChaincodeVersion: "v0"}, //调用cc0/chain1:v0，应该由cc1/chain1升级tx无效
2: {ChainID: "chain1", ChaincodeName: "lscc", ChaincodeVersion: ""},  //将cc0/chain1升级到v1，后一个cc0/chain1 upgtade-tx将失效。
3: {ChainID: "chain1", ChaincodeName: "cc0", ChaincodeVersion: "v0"}, //调用cc0/chain1:v0，应该由cc1/chain1升级tx无效
4: {ChainID: "chain1", ChaincodeName: "cc0", ChaincodeVersion: "v1"}, //调用cc0/chain1:v1，应该由cc1/chain1升级tx无效
5: {ChainID: "chain1", ChaincodeName: "cc1", ChaincodeVersion: "v0"}, //调用cc1/chain1:v0，不应受其他chaincode升级tx的影响。
6: {ChainID: "chain1", ChaincodeName: "lscc", ChaincodeVersion: ""},  //将cc0/chain1升级到v2，后一个cc0/chain1 upgtade-tx将失效。
7: {ChainID: "chain1", ChaincodeName: "lscc", ChaincodeVersion: ""},  //将cc0/chain1升级到v3
	}
	upgradedChaincodes := map[int]*sysccprovider.ChaincodeInstance{
		2: {ChainID: "chain1", ChaincodeName: "cc0", ChaincodeVersion: "v1"},
		6: {ChainID: "chain1", ChaincodeName: "cc0", ChaincodeVersion: "v2"},
		7: {ChainID: "chain1", ChaincodeName: "cc0", ChaincodeVersion: "v3"},
	}

	txsfltr := ledgerUtil.NewTxValidationFlags(8)
	txsfltr.SetFlag(0, peer.TxValidationCode_VALID)
	txsfltr.SetFlag(1, peer.TxValidationCode_VALID)
	txsfltr.SetFlag(2, peer.TxValidationCode_VALID)
	txsfltr.SetFlag(3, peer.TxValidationCode_VALID)
	txsfltr.SetFlag(4, peer.TxValidationCode_VALID)
	txsfltr.SetFlag(5, peer.TxValidationCode_VALID)
	txsfltr.SetFlag(6, peer.TxValidationCode_VALID)
	txsfltr.SetFlag(7, peer.TxValidationCode_VALID)

	expectTxsFltr := ledgerUtil.NewTxValidationFlags(8)
	expectTxsFltr.SetFlag(0, peer.TxValidationCode_VALID)
	expectTxsFltr.SetFlag(1, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
	expectTxsFltr.SetFlag(2, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
	expectTxsFltr.SetFlag(3, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
	expectTxsFltr.SetFlag(4, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
	expectTxsFltr.SetFlag(5, peer.TxValidationCode_VALID)
	expectTxsFltr.SetFlag(6, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
	expectTxsFltr.SetFlag(7, peer.TxValidationCode_VALID)

	tValidator := &TxValidator{}
	tValidator.invalidTXsForUpgradeCC(txsChaincodeNames, upgradedChaincodes, txsfltr)

	assert.EqualValues(t, expectTxsFltr, txsfltr)
}
