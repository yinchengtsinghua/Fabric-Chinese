
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


package historyleveldb

import (
	"os"
	"strconv"
	"testing"

	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	util2 "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/history/historydb/historyleveldb")
	flogging.ActivateSpec("leveldbhelper,historyleveldb=debug")
	os.Exit(m.Run())
}

//testsavepoint在每个块后写入保存点并通过getblocknumfromsavepoint返回的测试
func TestSavepoint(t *testing.T) {
	env := newTestHistoryEnv(t)
	defer env.cleanup()

//读取保存点，它不应存在，并且应返回零高度对象
	savepoint, err := env.testHistoryDB.GetLastSavepoint()
	assert.NoError(t, err, "Error upon historyDatabase.GetLastSavepoint()")
	assert.Nil(t, savepoint)

//当找不到保存点并从块0恢复时，shouldRecover应返回true
	status, blockNum, err := env.testHistoryDB.ShouldRecover(0)
	assert.NoError(t, err, "Error upon historyDatabase.ShouldRecover()")
	assert.True(t, status)
	assert.Equal(t, uint64(0), blockNum)

	bg, gb := testutil.NewBlockGenerator(t, "testLedger", false)
	assert.NoError(t, env.testHistoryDB.Commit(gb))
//读取保存点，它现在应该存在并返回一个高度为blocknum 0的对象
	savepoint, err = env.testHistoryDB.GetLastSavepoint()
	assert.NoError(t, err, "Error upon historyDatabase.GetLastSavepoint()")
	assert.Equal(t, uint64(0), savepoint.BlockNum)

//创建下一个块（块1）
	txid := util2.GenerateUUID()
	simulator, _ := env.txmgr.NewTxSimulator(txid)
	simulator.SetState("ns1", "key1", []byte("value1"))
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimResBytes, _ := simRes.GetPubSimulationBytes()
	block1 := bg.NextBlock([][]byte{pubSimResBytes})
	assert.NoError(t, env.testHistoryDB.Commit(block1))
	savepoint, err = env.testHistoryDB.GetLastSavepoint()
	assert.NoError(t, err, "Error upon historyDatabase.GetLastSavepoint()")
	assert.Equal(t, uint64(1), savepoint.BlockNum)

//应恢复应返回假
	status, blockNum, err = env.testHistoryDB.ShouldRecover(1)
	assert.NoError(t, err, "Error upon historyDatabase.ShouldRecover()")
	assert.False(t, status)
	assert.Equal(t, uint64(2), blockNum)

//创建下一个块（块2）
	txid = util2.GenerateUUID()
	simulator, _ = env.txmgr.NewTxSimulator(txid)
	simulator.SetState("ns1", "key1", []byte("value2"))
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	pubSimResBytes, _ = simRes.GetPubSimulationBytes()
	block2 := bg.NextBlock([][]byte{pubSimResBytes})

//假设对等机未能将此块提交到HistoryDB，现在正在恢复。
	env.testHistoryDB.CommitLostBlock(&ledger.BlockAndPvtData{Block: block2})
	savepoint, err = env.testHistoryDB.GetLastSavepoint()
	assert.NoError(t, err, "Error upon historyDatabase.GetLastSavepoint()")
	assert.Equal(t, uint64(2), savepoint.BlockNum)

//通过高位blocknum，should recover应返回true，3作为blocknum进行恢复
	status, blockNum, err = env.testHistoryDB.ShouldRecover(10)
	assert.NoError(t, err, "Error upon historyDatabase.ShouldRecover()")
	assert.True(t, status)
	assert.Equal(t, uint64(3), blockNum)
}

func TestHistory(t *testing.T) {
	env := newTestHistoryEnv(t)
	defer env.cleanup()
	provider := env.testBlockStorageEnv.provider
	ledger1id := "ledger1"
	store1, err := provider.OpenBlockStore(ledger1id)
	assert.NoError(t, err, "Error upon provider.OpenBlockStore()")
	defer store1.Shutdown()

	bg, gb := testutil.NewBlockGenerator(t, ledger1id, false)
	assert.NoError(t, store1.AddBlock(gb))
	assert.NoError(t, env.testHistoryDB.Commit(gb))

//BLUT1
	txid := util2.GenerateUUID()
	simulator, _ := env.txmgr.NewTxSimulator(txid)
	value1 := []byte("value1")
	simulator.SetState("ns1", "key7", value1)
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimResBytes, _ := simRes.GetPubSimulationBytes()
	block1 := bg.NextBlock([][]byte{pubSimResBytes})
	err = store1.AddBlock(block1)
	assert.NoError(t, err)
	err = env.testHistoryDB.Commit(block1)
	assert.NoError(t, err)

//BLUT2 TRA1
	simulationResults := [][]byte{}
	txid = util2.GenerateUUID()
	simulator, _ = env.txmgr.NewTxSimulator(txid)
	value2 := []byte("value2")
	simulator.SetState("ns1", "key7", value2)
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	pubSimResBytes, _ = simRes.GetPubSimulationBytes()
	simulationResults = append(simulationResults, pubSimResBytes)
//BLUT2转铁蛋白2
	txid2 := util2.GenerateUUID()
	simulator2, _ := env.txmgr.NewTxSimulator(txid2)
	value3 := []byte("value3")
	simulator2.SetState("ns1", "key7", value3)
	simulator2.Done()
	simRes2, _ := simulator2.GetTxSimulationResults()
	pubSimResBytes2, _ := simRes2.GetPubSimulationBytes()
	simulationResults = append(simulationResults, pubSimResBytes2)
	block2 := bg.NextBlock(simulationResults)
	err = store1.AddBlock(block2)
	assert.NoError(t, err)
	err = env.testHistoryDB.Commit(block2)
	assert.NoError(t, err)

//方块3
	txid = util2.GenerateUUID()
	simulator, _ = env.txmgr.NewTxSimulator(txid)
	simulator.DeleteState("ns1", "key7")
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	pubSimResBytes, _ = simRes.GetPubSimulationBytes()
	block3 := bg.NextBlock([][]byte{pubSimResBytes})
	err = store1.AddBlock(block3)
	assert.NoError(t, err)
	err = env.testHistoryDB.Commit(block3)
	assert.NoError(t, err)
	t.Logf("Inserted all 3 blocks")

	qhistory, err := env.testHistoryDB.NewHistoryQueryExecutor(store1)
	assert.NoError(t, err, "Error upon NewHistoryQueryExecutor")

	itr, err2 := qhistory.GetHistoryForKey("ns1", "key7")
	assert.NoError(t, err2, "Error upon GetHistoryForKey()")

	count := 0
	for {
		kmod, _ := itr.Next()
		if kmod == nil {
			break
		}
		txid = kmod.(*queryresult.KeyModification).TxId
		retrievedValue := kmod.(*queryresult.KeyModification).Value
		retrievedTimestamp := kmod.(*queryresult.KeyModification).Timestamp
		retrievedIsDelete := kmod.(*queryresult.KeyModification).IsDelete
		t.Logf("Retrieved history record for key=key7 at TxId=%s with value %v and timestamp %v",
			txid, retrievedValue, retrievedTimestamp)
		count++
		if count != 4 {
			expectedValue := []byte("value" + strconv.Itoa(count))
			assert.Equal(t, expectedValue, retrievedValue)
			assert.NotNil(t, retrievedTimestamp)
			assert.False(t, retrievedIsDelete)
		} else {
			assert.Equal(t, []uint8(nil), retrievedValue)
			assert.NotNil(t, retrievedTimestamp)
			assert.True(t, retrievedIsDelete)
		}
	}
	assert.Equal(t, 4, count)
}

func TestHistoryForInvalidTran(t *testing.T) {
	env := newTestHistoryEnv(t)
	defer env.cleanup()
	provider := env.testBlockStorageEnv.provider
	ledger1id := "ledger1"
	store1, err := provider.OpenBlockStore(ledger1id)
	assert.NoError(t, err, "Error upon provider.OpenBlockStore()")
	defer store1.Shutdown()

	bg, gb := testutil.NewBlockGenerator(t, ledger1id, false)
	assert.NoError(t, store1.AddBlock(gb))
	assert.NoError(t, env.testHistoryDB.Commit(gb))

//BLUT1
	txid := util2.GenerateUUID()
	simulator, _ := env.txmgr.NewTxSimulator(txid)
	value1 := []byte("value1")
	simulator.SetState("ns1", "key7", value1)
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimResBytes, _ := simRes.GetPubSimulationBytes()
	block1 := bg.NextBlock([][]byte{pubSimResBytes})

//对于此无效的Tran测试，请将事务设置为无效
	txsFilter := util.TxValidationFlags(block1.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	txsFilter.SetFlag(0, peer.TxValidationCode_INVALID_OTHER_REASON)
	block1.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter

	err = store1.AddBlock(block1)
	assert.NoError(t, err)
	err = env.testHistoryDB.Commit(block1)
	assert.NoError(t, err)

	qhistory, err := env.testHistoryDB.NewHistoryQueryExecutor(store1)
	assert.NoError(t, err, "Error upon NewHistoryQueryExecutor")

	itr, err2 := qhistory.GetHistoryForKey("ns1", "key7")
	assert.NoError(t, err2, "Error upon GetHistoryForKey()")

//测试是否没有历史值，因为Tran被标记为无效
	kmod, _ := itr.Next()
	assert.Nil(t, kmod)
}

//testsavepoint在每个块后写入保存点并通过getblocknumfromsavepoint返回的测试
func TestHistoryDisabled(t *testing.T) {
	env := newTestHistoryEnv(t)
	defer env.cleanup()
	viper.Set("ledger.history.enableHistoryDatabase", "false")
//不需要将blockstore传递给历史执行器，它不会在这个测试中使用。
	qhistory, err := env.testHistoryDB.NewHistoryQueryExecutor(nil)
	assert.NoError(t, err, "Error upon NewHistoryQueryExecutor")
	_, err2 := qhistory.GetHistoryForKey("ns1", "key7")
	assert.Error(t, err2, "Error should have been returned for GetHistoryForKey() when history disabled")
}

//testgenesBlockNoError测试Genesis块被历史处理忽略
//因为我们只保存链式代码密钥写入的历史记录
func TestGenesisBlockNoError(t *testing.T) {
	env := newTestHistoryEnv(t)
	defer env.cleanup()
	block, err := configtxtest.MakeGenesisBlock("test_chainid")
	assert.NoError(t, err)
	err = env.testHistoryDB.Commit(block)
	assert.NoError(t, err)
}

//TestHistoryWithKeyContainingNilbytes在密钥包含零字节时测试HistoryDB（FAB-11244）-
//在为HistoryDB中的条目形成的复合键中，它正好用作分隔符。
func TestHistoryWithKeyContainingNilBytes(t *testing.T) {
	env := newTestHistoryEnv(t)
	defer env.cleanup()
	provider := env.testBlockStorageEnv.provider
	ledger1id := "ledger1"
	store1, err := provider.OpenBlockStore(ledger1id)
	assert.NoError(t, err, "Error upon provider.OpenBlockStore()")
	defer store1.Shutdown()

	bg, gb := testutil.NewBlockGenerator(t, ledger1id, false)
	assert.NoError(t, store1.AddBlock(gb))
	assert.NoError(t, env.testHistoryDB.Commit(gb))

//BLUT1
	txid := util2.GenerateUUID()
	simulator, _ := env.txmgr.NewTxSimulator(txid)
simulator.SetState("ns1", "key", []byte("value1")) //添加一个不包含零字节的键
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimResBytes, _ := simRes.GetPubSimulationBytes()
	block1 := bg.NextBlock([][]byte{pubSimResBytes})
	err = store1.AddBlock(block1)
	assert.NoError(t, err)
	err = env.testHistoryDB.Commit(block1)
	assert.NoError(t, err)

//BLUT2 TRA1
	simulationResults := [][]byte{}
	txid = util2.GenerateUUID()
	simulator, _ = env.txmgr.NewTxSimulator(txid)
simulator.SetState("ns1", "key", []byte("value2")) //为键添加另一个值<key>
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	pubSimResBytes, _ = simRes.GetPubSimulationBytes()
	simulationResults = append(simulationResults, pubSimResBytes)

//BLUT2转铁蛋白2
	txid2 := util2.GenerateUUID()
	simulator2, _ := env.txmgr.NewTxSimulator(txid2)
//添加另一个包含空字节的键<key\x00\x01\x01\x15>—这样，当使用旧格式形成范围查询时，该键将落在范围内。
	simulator2.SetState("ns1", "key\x00\x01\x01\x15", []byte("dummyVal1"))
	simulator2.SetState("ns1", "\x00key\x00\x01\x01\x15", []byte("dummyVal2"))
	simulator2.Done()
	simRes2, _ := simulator2.GetTxSimulationResults()
	pubSimResBytes2, _ := simRes2.GetPubSimulationBytes()
	simulationResults = append(simulationResults, pubSimResBytes2)
	block2 := bg.NextBlock(simulationResults)
	err = store1.AddBlock(block2)
	assert.NoError(t, err)
	err = env.testHistoryDB.Commit(block2)
	assert.NoError(t, err)

	qhistory, err := env.testHistoryDB.NewHistoryQueryExecutor(store1)
	assert.NoError(t, err, "Error upon NewHistoryQueryExecutor")
	testutilVerifyResults(t, qhistory, "ns1", "key", []string{"value1", "value2"})
	testutilVerifyResults(t, qhistory, "ns1", "key\x00\x01\x01\x15", []string{"dummyVal1"})
	testutilVerifyResults(t, qhistory, "ns1", "\x00key\x00\x01\x01\x15", []string{"dummyVal2"})
}

func testutilVerifyResults(t *testing.T, hqe ledger.HistoryQueryExecutor, ns, key string, expectedVals []string) {
	itr, err := hqe.GetHistoryForKey(ns, key)
	assert.NoError(t, err, "Error upon GetHistoryForKey()")
	retrievedVals := []string{}
	for {
		kmod, _ := itr.Next()
		if kmod == nil {
			break
		}
		txid := kmod.(*queryresult.KeyModification).TxId
		retrievedValue := string(kmod.(*queryresult.KeyModification).Value)
		retrievedVals = append(retrievedVals, retrievedValue)
		t.Logf("Retrieved history record at TxId=%s with value %s", txid, retrievedValue)
	}
	assert.Equal(t, expectedVals, retrievedVals)
}
