
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/

package qscc

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/aclmgmt/mocks"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	ledger2 "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/protos/common"
	peer2 "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLedger(chainid string, path string) (*shim.MockStub, error) {
	mockAclProvider.Reset()

	viper.Set("peer.fileSystemPath", path)
	peer.MockInitialize()
	peer.MockCreateChain(chainid)

	lq := &LedgerQuerier{
		aclProvider: mockAclProvider,
	}
	stub := shim.NewMockStub("LedgerQuerier", lq)
	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		return nil, fmt.Errorf("Init failed for test ledger [%s] with message: %s", chainid, string(res.Message))
	}
	return stub, nil
}

//pass the prop so we can conveniently inline it in the call and get it back
func resetProvider(res, chainid string, prop *peer2.SignedProposal, retErr error) *peer2.SignedProposal {
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", res, chainid, prop).Return(retErr)
	return prop
}

func tempDir(t *testing.T, stem string) string {
	path, err := ioutil.TempDir("", "qscc-"+stem)
	require.NoError(t, err)
	return path
}

func TestQueryGetChainInfo(t *testing.T) {
	chainid := "mytestchainid1"
	path := tempDir(t, "test1")
	defer os.RemoveAll(path)

	stub, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(GetChainInfo), []byte(chainid)}
	prop := resetProvider(resources.Qscc_GetChainInfo, chainid, &peer2.SignedProposal{}, nil)
	res := stub.MockInvokeWithSignedProposal("1", args, prop)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	args = [][]byte{[]byte(GetChainInfo)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	args = [][]byte{[]byte(GetChainInfo), []byte("fakechainid")}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestQueryGetTransactionByID(t *testing.T) {
	chainid := "mytestchainid2"
	path := tempDir(t, "test2")
	defer os.RemoveAll(path)

	stub, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(GetTransactionByID), []byte(chainid), []byte("1")}
	prop := resetProvider(resources.Qscc_GetTransactionByID, chainid, &peer2.SignedProposal{}, nil)
	res := stub.MockInvokeWithSignedProposal("1", args, prop)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTransactionByID should have failed with invalid txid: 1")

	args = [][]byte{[]byte(GetTransactionByID), []byte(chainid), []byte(nil)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTransactionByID should have failed with invalid txid: nil")

//参数数目错误的测试
	args = [][]byte{[]byte(GetTransactionByID), []byte(chainid)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTransactionByID should have failed due to incorrect number of arguments")
}

func TestQueryGetBlockByNumber(t *testing.T) {
	chainid := "mytestchainid3"
	path := tempDir(t, "test3")
	defer os.RemoveAll(path)

	stub, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}

//0号区块（Genesis区块）已经存在于分类账中。
	args := [][]byte{[]byte(GetBlockByNumber), []byte(chainid), []byte("0")}
	prop := resetProvider(resources.Qscc_GetBlockByNumber, chainid, &peer2.SignedProposal{}, nil)
	res := stub.MockInvokeWithSignedProposal("1", args, prop)
	assert.Equal(t, int32(shim.OK), res.Status, "GetBlockByNumber should have succeeded for block number: 0")

//block number 1 should not be present in the ledger
	args = [][]byte{[]byte(GetBlockByNumber), []byte(chainid), []byte("1")}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByNumber should have failed with invalid number: 1")

//块编号不能为零
	args = [][]byte{[]byte(GetBlockByNumber), []byte(chainid), []byte(nil)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByNumber should have failed with nil block number")
}

func TestQueryGetBlockByHash(t *testing.T) {
	chainid := "mytestchainid4"
	path := tempDir(t, "test4")
	defer os.RemoveAll(path)

	stub, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(GetBlockByHash), []byte(chainid), []byte("0")}
	prop := resetProvider(resources.Qscc_GetBlockByHash, chainid, &peer2.SignedProposal{}, nil)
	res := stub.MockInvokeWithSignedProposal("1", args, prop)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByHash should have failed with invalid hash: 0")

	args = [][]byte{[]byte(GetBlockByHash), []byte(chainid), []byte(nil)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByHash should have failed with nil hash")
}

func TestQueryGetBlockByTxID(t *testing.T) {
	chainid := "mytestchainid5"
	path := tempDir(t, "test5")
	defer os.RemoveAll(path)

	stub, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(GetBlockByTxID), []byte(chainid), []byte("")}
	prop := resetProvider(resources.Qscc_GetBlockByTxID, chainid, &peer2.SignedProposal{}, nil)
	res := stub.MockInvokeWithSignedProposal("1", args, prop)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByTxID should have failed with blank txId.")
}

func TestFailingAccessControl(t *testing.T) {
	chainid := "mytestchainid6"
	path := tempDir(t, "test6")
	defer os.RemoveAll(path)

	_, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}
	e := &LedgerQuerier{
		aclProvider: mockAclProvider,
	}
	stub := shim.NewMockStub("LedgerQuerier", e)

//获取信息
	args := [][]byte{[]byte(GetChainInfo), []byte(chainid)}
	sProp, _ := utils.MockSignedEndorserProposalOrPanic(chainid, &peer2.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	sProp.Signature = sProp.ProposalBytes
//将aclprovider设置为失败
	resetProvider(resources.Qscc_GetChainInfo, chainid, sProp, errors.New("Failed access control"))
	res := stub.MockInvokeWithSignedProposal("2", args, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo must fail: %s", res.Message)
	assert.Contains(t, res.Message, "Failed access control")
//断言达到了预期
	mockAclProvider.AssertExpectations(t)

//GetBlockBy编号
	args = [][]byte{[]byte(GetBlockByNumber), []byte(chainid), []byte("1")}
	sProp, _ = utils.MockSignedEndorserProposalOrPanic(chainid, &peer2.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	sProp.Signature = sProp.ProposalBytes
//将aclprovider设置为失败
	resetProvider(resources.Qscc_GetBlockByNumber, chainid, sProp, errors.New("Failed access control"))
	res = stub.MockInvokeWithSignedProposal("2", args, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByNumber must fail: %s", res.Message)
	assert.Contains(t, res.Message, "Failed access control")
//断言达到了预期
	mockAclProvider.AssertExpectations(t)

//获取块哈希
	args = [][]byte{[]byte(GetBlockByHash), []byte(chainid), []byte("1")}
	sProp, _ = utils.MockSignedEndorserProposalOrPanic(chainid, &peer2.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	sProp.Signature = sProp.ProposalBytes
//将aclprovider设置为失败
	resetProvider(resources.Qscc_GetBlockByHash, chainid, sProp, errors.New("Failed access control"))
	res = stub.MockInvokeWithSignedProposal("2", args, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByHash must fail: %s", res.Message)
	assert.Contains(t, res.Message, "Failed access control")
//断言达到了预期
	mockAclProvider.AssertExpectations(t)

//获取数据块
	args = [][]byte{[]byte(GetBlockByTxID), []byte(chainid), []byte("1")}
	sProp, _ = utils.MockSignedEndorserProposalOrPanic(chainid, &peer2.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	sProp.Signature = sProp.ProposalBytes
//将aclprovider设置为失败
	resetProvider(resources.Qscc_GetBlockByTxID, chainid, sProp, errors.New("Failed access control"))
	res = stub.MockInvokeWithSignedProposal("2", args, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlockByTxID must fail: %s", res.Message)
	assert.Contains(t, res.Message, "Failed access control")
//断言达到了预期
	mockAclProvider.AssertExpectations(t)

//GetTransactionByID
	args = [][]byte{[]byte(GetTransactionByID), []byte(chainid), []byte("1")}
	sProp, _ = utils.MockSignedEndorserProposalOrPanic(chainid, &peer2.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	sProp.Signature = sProp.ProposalBytes
//将aclprovider设置为失败
	resetProvider(resources.Qscc_GetTransactionByID, chainid, sProp, errors.New("Failed access control"))
	res = stub.MockInvokeWithSignedProposal("2", args, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "Qscc_GetTransactionByID must fail: %s", res.Message)
	assert.Contains(t, res.Message, "Failed access control")
//断言达到了预期
	mockAclProvider.AssertExpectations(t)
}

func TestQueryNonexistentFunction(t *testing.T) {
	chainid := "mytestchainid7"
	path := tempDir(t, "test7")
	defer os.RemoveAll(path)

	stub, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte("GetBlocks"), []byte(chainid), []byte("arg1")}
	prop := resetProvider("qscc/GetBlocks", chainid, &peer2.SignedProposal{}, nil)
	res := stub.MockInvokeWithSignedProposal("1", args, prop)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetBlocks should have failed because the function does not exist")
}

//testquerygeneratedblock测试新生成的块的各种查询
//包含两个事务
func TestQueryGeneratedBlock(t *testing.T) {
	chainid := "mytestchainid8"
	path := tempDir(t, "test8")
	defer os.RemoveAll(path)

	stub, err := setupTestLedger(chainid, path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	block1 := addBlockForTesting(t, chainid)

//块编号1现在应该存在
	args := [][]byte{[]byte(GetBlockByNumber), []byte(chainid), []byte("1")}
	prop := resetProvider(resources.Qscc_GetBlockByNumber, chainid, &peer2.SignedProposal{}, nil)
	res := stub.MockInvokeWithSignedProposal("1", args, prop)
	assert.Equal(t, int32(shim.OK), res.Status, "GetBlockByNumber should have succeeded for block number 1")

//1号方块
	args = [][]byte{[]byte(GetBlockByHash), []byte(chainid), []byte(block1.Header.Hash())}
	prop = resetProvider(resources.Qscc_GetBlockByHash, chainid, &peer2.SignedProposal{}, nil)
	res = stub.MockInvokeWithSignedProposal("2", args, prop)
	assert.Equal(t, int32(shim.OK), res.Status, "GetBlockByHash should have succeeded for block 1 hash")

//钻取该块以查找它包含的事务ID
	for _, d := range block1.Data.Data {
		ebytes := d
		if ebytes != nil {
			if env, err := utils.GetEnvelopeFromBlock(ebytes); err != nil {
				t.Fatalf("error getting envelope from block: %s", err)
			} else if env != nil {
				payload, err := utils.GetPayload(env)
				if err != nil {
					t.Fatalf("error extracting payload from envelope: %s", err)
				}
				chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
				if err != nil {
					t.Fatalf(err.Error())
				}
				if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {
					args = [][]byte{[]byte(GetBlockByTxID), []byte(chainid), []byte(chdr.TxId)}
					mockAclProvider.Reset()
					prop = resetProvider(resources.Qscc_GetBlockByTxID, chainid, &peer2.SignedProposal{}, nil)
					res = stub.MockInvokeWithSignedProposal("3", args, prop)
					assert.Equal(t, int32(shim.OK), res.Status, "GetBlockByTxId should have succeeded for txid: %s", chdr.TxId)

					args = [][]byte{[]byte(GetTransactionByID), []byte(chainid), []byte(chdr.TxId)}
					prop = resetProvider(resources.Qscc_GetTransactionByID, chainid, &peer2.SignedProposal{}, nil)
					res = stub.MockInvokeWithSignedProposal("4", args, prop)
					assert.Equal(t, int32(shim.OK), res.Status, "GetTransactionById should have succeeded for txid: %s", chdr.TxId)
				}
			}
		}
	}
}

func addBlockForTesting(t *testing.T, chainid string) *common.Block {
	ledger := peer.GetLedger(chainid)
	defer ledger.Close()

	txid1 := util.GenerateUUID()
	simulator, _ := ledger.NewTxSimulator(txid1)
	simulator.SetState("ns1", "key1", []byte("value1"))
	simulator.SetState("ns1", "key2", []byte("value2"))
	simulator.SetState("ns1", "key3", []byte("value3"))
	simulator.Done()
	simRes1, _ := simulator.GetTxSimulationResults()
	pubSimResBytes1, _ := simRes1.GetPubSimulationBytes()

	txid2 := util.GenerateUUID()
	simulator, _ = ledger.NewTxSimulator(txid2)
	simulator.SetState("ns2", "key4", []byte("value4"))
	simulator.SetState("ns2", "key5", []byte("value5"))
	simulator.SetState("ns2", "key6", []byte("value6"))
	simulator.Done()
	simRes2, _ := simulator.GetTxSimulationResults()
	pubSimResBytes2, _ := simRes2.GetPubSimulationBytes()

	bcInfo, err := ledger.GetBlockchainInfo()
	assert.NoError(t, err)
	block1 := testutil.ConstructBlock(t, 1, bcInfo.CurrentBlockHash, [][]byte{pubSimResBytes1, pubSimResBytes2}, false)
	ledger.CommitWithPvtData(&ledger2.BlockAndPvtData{Block: block1})
	return block1
}

var mockAclProvider *mocks.MockACLProvider

func TestMain(m *testing.M) {
	mockAclProvider = &mocks.MockACLProvider{}
	mockAclProvider.Reset()

	os.Exit(m.Run())
}
