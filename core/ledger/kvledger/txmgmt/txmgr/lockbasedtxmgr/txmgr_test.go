
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


package lockbasedtxmgr

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	ledgertestutil "github.com/hyperledger/fabric/core/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	ledgertestutil.SetupCoreYAMLConfig()
	flogging.ActivateSpec("lockbasedtxmgr,statevalidator,statebasedval,statecouchdb,valimpl,pvtstatepurgemgmt,valinternal=debug")
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/txmgmt/txmgr/lockbasedtxmgr")
	os.Exit(m.Run())
}

func TestTxSimulatorWithNoExistingData(t *testing.T) {
//为pkg_test.go中配置的每个环境运行测试
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxsimulatorwithnoexistingdata"
		testEnv.init(t, testLedgerID, nil)
		testTxSimulatorWithNoExistingData(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxSimulatorWithNoExistingData(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	s, _ := txMgr.NewTxSimulator("test_txid")
	value, err := s.GetState("ns1", "key1")
	assert.NoErrorf(t, err, "Error in GetState(): %s", err)
	assert.Nil(t, value)

	s.SetState("ns1", "key1", []byte("value1"))
	s.SetState("ns1", "key2", []byte("value2"))
	s.SetState("ns2", "key3", []byte("value3"))
	s.SetState("ns2", "key4", []byte("value4"))

	value, _ = s.GetState("ns2", "key3")
	assert.Nil(t, value)

	simulationResults, err := s.GetTxSimulationResults()
	assert.NoError(t, err)
	assert.Nil(t, simulationResults.PvtSimulationResults)
}

func TestTxSimulatorGetResults(t *testing.T) {
	testEnv := testEnvsMap[levelDBtestEnvName]
	testEnv.init(t, "testLedger", nil)
	defer testEnv.cleanup()
	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll3"},
			{"ns2", "coll2"},
			{"ns3", "coll3"},
		},
		version.NewHeight(1, 1),
	)

	var err error

//在一个名称空间“ns1”中创建模拟器并获取/设置键
	simulator, _ := testEnv.getTxMgr().NewTxSimulator("test_txid1")
	simulator.GetState("ns1", "key1")
	_, err = simulator.GetPrivateData("ns1", "coll1", "key1")
	assert.NoError(t, err)
	simulator.SetState("ns1", "key1", []byte("value1"))
//get simulation results and verify that this contains rwset only for one namespace
	simulationResults1, err := simulator.GetTxSimulationResults()
	assert.NoError(t, err)
	assert.Len(t, simulationResults1.PubSimulationResults.NsRwset, 1)
//克隆冻结模拟结果1
	buff1 := new(bytes.Buffer)
	assert.NoError(t, gob.NewEncoder(buff1).Encode(simulationResults1))
	frozenSimulationResults1 := &ledger.TxSimulationResults{}
	assert.NoError(t, gob.NewDecoder(buff1).Decode(&frozenSimulationResults1))

//使用一个或多个命名空间“NS2”中的GET/SET键获得模拟结果后使用相同的模拟器
	simulator.GetState("ns2", "key2")
	simulator.GetPrivateData("ns2", "coll2", "key2")
	simulator.SetState("ns2", "key2", []byte("value2"))
//获取仿真结果，并验证在多次获得仿真结果时是否出现错误。
	_, err = simulator.GetTxSimulationResults()
assert.Error(t, err) //多次调用“GETTXSimultRESULTS（）”会引起错误
//现在，验证模拟器操作对先前获得的结果没有影响。
	assert.Equal(t, frozenSimulationResults1, simulationResults1)

//调用“done”后，所有数据获取/设置操作都应失败。
	simulator.Done()
	_, err = simulator.GetState("ns3", "key3")
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
	err = simulator.SetState("ns3", "key3", []byte("value3"))
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
	_, err = simulator.GetPrivateData("ns3", "coll3", "key3")
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
	err = simulator.SetPrivateData("ns3", "coll3", "key3", []byte("value3"))
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
}

func TestTxSimulatorWithExistingData(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Run(testEnv.getName(), func(t *testing.T) {
			testLedgerID := "testtxsimulatorwithexistingdata"
			testEnv.init(t, testLedgerID, nil)
			testTxSimulatorWithExistingData(t, testEnv)
			testEnv.cleanup()
		})
	}
}

func testTxSimulatorWithExistingData(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
//模拟TX1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	s1.SetState("ns1", "key1", []byte("value1"))
	s1.SetState("ns1", "key2", []byte("value2"))
	s1.SetState("ns2", "key3", []byte("value3"))
	s1.SetState("ns2", "key4", []byte("value4"))
	s1.Done()
//验证并提交rwset
	txRWSet1, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

//模拟对现有数据进行更改的tx2
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	value, _ := s2.GetState("ns1", "key1")
	assert.Equal(t, []byte("value1"), value)
	s2.SetState("ns1", "key1", []byte("value1_1"))
	s2.DeleteState("ns2", "key3")
	value, _ = s2.GetState("ns1", "key1")
	assert.Equal(t, []byte("value1"), value)
	s2.Done()
//验证并提交tx2的rwset
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

//模拟TX3
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	value, _ = s3.GetState("ns1", "key1")
	assert.Equal(t, []byte("value1_1"), value)
	value, _ = s3.GetState("ns2", "key3")
	assert.Nil(t, value)
	s3.Done()

//验证持久化中的密钥版本
	vv, _ := env.getVDB().GetState("ns1", "key1")
	assert.Equal(t, version.NewHeight(2, 0), vv.Version)
	vv, _ = env.getVDB().GetState("ns1", "key2")
	assert.Equal(t, version.NewHeight(1, 0), vv.Version)
}

func TestTxValidation(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxvalidation"
		testEnv.init(t, testLedgerID, nil)
		testTxValidation(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxValidation(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
//模拟TX1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	s1.SetState("ns1", "key1", []byte("value1"))
	s1.SetState("ns1", "key2", []byte("value2"))
	s1.SetState("ns2", "key3", []byte("value3"))
	s1.SetState("ns2", "key4", []byte("value4"))
	s1.Done()
//验证并提交rwset
	txRWSet1, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

//模拟对现有数据进行更改的tx2。
//tx2:读取/更新ns1:key1，删除ns2:key3。
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	value, _ := s2.GetState("ns1", "key1")
	assert.Equal(t, []byte("value1"), value)

	s2.SetState("ns1", "key1", []byte("value1_2"))
	s2.DeleteState("ns2", "key3")
	s2.Done()

//在提交tx2更改之前模拟tx3。读取并修改由tx2更改的密钥。
//TX3:读取/更新NS1:键1
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	s3.GetState("ns1", "key1")
	s3.SetState("ns1", "key1", []byte("value1_3"))
	s3.Done()

//在提交tx2更改之前模拟tx4。读取和删除由tx2更改的密钥
//TX4:读取/删除NS2:键3
	s4, _ := txMgr.NewTxSimulator("test_tx4")
	s4.GetState("ns2", "key3")
	s4.DeleteState("ns2", "key3")
	s4.Done()

//在提交TX2更改之前模拟TX5。修改然后读取由tx2更改的密钥并写入新密钥
//TX5:更新/读取NS1:键1
	s5, _ := txMgr.NewTxSimulator("test_tx5")
	s5.SetState("ns1", "key1", []byte("new_value"))
	s5.GetState("ns1", "key1")
	s5.Done()

//simulate tx6 before committing tx2 changes. Only writes a new key, does not reads/writes a key changed by tx2
//tx6:更新ns1:新密钥
	s6, _ := txMgr.NewTxSimulator("test_tx6")
	s6.SetState("ns1", "new_key", []byte("new_value"))
	s6.Done()

//模拟交易汇总
//tx2:读取/更新ns1:key1，删除ns2:key3。
//TX3:读取/更新NS1:键1
//TX4:读取/删除NS2:键3
//TX5:更新/读取NS1:键1
//tx6:更新ns1:新密钥

//验证并提交tx2的rwset
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

//由于读取冲突，tx3、tx4和tx5的rwset现在应该无效。
	txRWSet3, _ := s3.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet3.PubSimulationResults)

	txRWSet4, _ := s4.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet4.PubSimulationResults)

	txRWSet5, _ := s5.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet5.PubSimulationResults)

//TX6应该仍然有效，因为它只写一个新的密钥
	txRWSet6, _ := s6.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet6.PubSimulationResults)
}

func TestTxPhantomValidation(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxphantomvalidation"
		testEnv.init(t, testLedgerID, nil)
		testTxPhantomValidation(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxPhantomValidation(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
//模拟TX1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	s1.SetState("ns", "key1", []byte("value1"))
	s1.SetState("ns", "key2", []byte("value2"))
	s1.SetState("ns", "key3", []byte("value3"))
	s1.SetState("ns", "key4", []byte("value4"))
	s1.SetState("ns", "key5", []byte("value5"))
	s1.SetState("ns", "key6", []byte("value6"))
//验证并提交rwset
	txRWSet1, _ := s1.GetTxSimulationResults()
s1.Done() //获取结果后显式调用Done以验证FAB-10788
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

//模拟TX2
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	itr2, _ := s2.GetStateRangeScanIterator("ns", "key2", "key5")
	for {
		if result, _ := itr2.Next(); result == nil {
			break
		}
	}
	s2.DeleteState("ns", "key3")
	txRWSet2, _ := s2.GetTxSimulationResults()
	s2.Done()

//模拟TX3
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	itr3, _ := s3.GetStateRangeScanIterator("ns", "key2", "key5")
	for {
		if result, _ := itr3.Next(); result == nil {
			break
		}
	}
	s3.SetState("ns", "key3", []byte("value3_new"))
	txRWSet3, _ := s3.GetTxSimulationResults()
	s3.Done()
//模拟TX4
	s4, _ := txMgr.NewTxSimulator("test_tx4")
	itr4, _ := s4.GetStateRangeScanIterator("ns", "key4", "key6")
	for {
		if result, _ := itr4.Next(); result == nil {
			break
		}
	}
	s4.SetState("ns", "key3", []byte("value3_new"))
	txRWSet4, _ := s4.GetTxSimulationResults()
	s4.Done()

//txrwset2应有效
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)
//txrwset2使txrwset3无效，因为它删除了范围内的键
	txMgrHelper.checkRWsetInvalid(txRWSet3.PubSimulationResults)
//TxRwset4应该有效，因为它在不同的范围内迭代
	txMgrHelper.validateAndCommitRWSet(txRWSet4.PubSimulationResults)
}

func TestIterator(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())

		testLedgerID := "testiterator.1"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 2, 7)
		testEnv.cleanup()

		testLedgerID = "testiterator.2"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 1, 11)
		testEnv.cleanup()

		testLedgerID = "testiterator.3"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 0, 0)
		testEnv.cleanup()

		testLedgerID = "testiterator.4"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 5, 0)
		testEnv.cleanup()

		testLedgerID = "testiterator.5"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 0, 5)
		testEnv.cleanup()
	}
}

func testIterator(t *testing.T, env testEnv, numKeys int, startKeyNum int, endKeyNum int) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	s, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= numKeys; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s.SetState(cID, k, v)
	}
	s.Done()
//验证并提交rwset
	txRWSet, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)

	var startKey string
	var endKey string
	var begin int
	var end int

	if startKeyNum != 0 {
		begin = startKeyNum
		startKey = createTestKey(startKeyNum)
	} else {
begin = 1 //数据库中的第一个键
		startKey = ""
	}

	if endKeyNum != 0 {
		endKey = createTestKey(endKeyNum)
		end = endKeyNum
	} else {
		endKey = ""
end = numKeys + 1 //数据库中的最后一个键
	}

	expectedCount := end - begin

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx2")
	itr, _ := queryExecuter.GetStateRangeScanIterator(cID, startKey, endKey)
	count := 0
	for {
		kv, _ := itr.Next()
		if kv == nil {
			break
		}
		keyNum := begin + count
		k := kv.(*queryresult.KV).Key
		v := kv.(*queryresult.KV).Value
		t.Logf("Retrieved k=%s, v=%s at count=%d start=%s end=%s", k, v, count, startKey, endKey)
		assert.Equal(t, createTestKey(keyNum), k)
		assert.Equal(t, createTestValue(keyNum), v)
		count++
	}
	assert.Equal(t, expectedCount, count)
}

func TestIteratorPaging(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())

//测试显式分页
		testLedgerID := "testiterator.1"
		testEnv.init(t, testLedgerID, nil)
		testIteratorPagingInit(t, testEnv, 10)
		returnKeys := []string{"key_002", "key_003"}
		nextStartKey := testIteratorPaging(t, testEnv, 10, "key_002", "key_007", int32(2), returnKeys)
		returnKeys = []string{"key_004", "key_005"}
		nextStartKey = testIteratorPaging(t, testEnv, 10, nextStartKey, "key_007", int32(2), returnKeys)
		returnKeys = []string{"key_006"}
		testIteratorPaging(t, testEnv, 10, nextStartKey, "key_007", int32(2), returnKeys)
		testEnv.cleanup()
	}
}

func testIteratorPagingInit(t *testing.T, env testEnv, numKeys int) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	s, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= numKeys; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s.SetState(cID, k, v)
	}
	s.Done()
//验证并提交rwset
	txRWSet, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)
}

func testIteratorPaging(t *testing.T, env testEnv, numKeys int, startKey, endKey string,
	limit int32, expectedKeys []string) string {
	cID := "cid"
	txMgr := env.getTxMgr()

	queryOptions := make(map[string]interface{})
	if limit != 0 {
		queryOptions["limit"] = limit
	}

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx2")
	itr, _ := queryExecuter.GetStateRangeScanIteratorWithMetadata(cID, startKey, endKey, queryOptions)

//验证返回的密钥
	testItrWithoutClose(t, itr, expectedKeys)

	returnBookmark := ""
	if limit > 0 {
		returnBookmark = itr.GetBookmarkAndClose()
	}

	return returnBookmark
}

//TestTrwithOutClose验证迭代器是否包含预期的键
func testItrWithoutClose(t *testing.T, itr ledger.QueryResultsIterator, expectedKeys []string) {
	for _, expectedKey := range expectedKeys {
		queryResult, err := itr.Next()
		assert.NoError(t, err, "An unexpected error was thrown during iterator Next()")
		vkv := queryResult.(*queryresult.KV)
		key := vkv.Key
		assert.Equal(t, expectedKey, key)
	}
	queryResult, err := itr.Next()
	assert.NoError(t, err, "An unexpected error was thrown during iterator Next()")
	assert.Nil(t, queryResult)
}

func TestIteratorWithDeletes(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testiteratorwithdeletes"
		testEnv.init(t, testLedgerID, nil)
		testIteratorWithDeletes(t, testEnv)
		testEnv.cleanup()
	}
}

func testIteratorWithDeletes(t *testing.T, env testEnv) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	s, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s.SetState(cID, k, v)
	}
	s.Done()
//验证并提交rwset
	txRWSet1, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

	s, _ = txMgr.NewTxSimulator("test_tx2")
	s.DeleteState(cID, createTestKey(4))
	s.Done()
//验证并提交rwset
	txRWSet2, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx3")
	itr, _ := queryExecuter.GetStateRangeScanIterator(cID, createTestKey(3), createTestKey(6))
	defer itr.Close()
	kv, _ := itr.Next()
	assert.Equal(t, createTestKey(3), kv.(*queryresult.KV).Key)
	kv, _ = itr.Next()
	assert.Equal(t, createTestKey(5), kv.(*queryresult.KV).Key)
}

func TestTxValidationWithItr(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxvalidationwithitr"
		testEnv.init(t, testLedgerID, nil)
		testTxValidationWithItr(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxValidationWithItr(t *testing.T, env testEnv) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

//模拟TX1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s1.SetState(cID, k, v)
	}
	s1.Done()
//验证并提交rwset
	txRWSet1, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

//模拟TX2，该TX2读到键“001”和键“002”。
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	itr, _ := s2.GetStateRangeScanIterator(cID, createTestKey(1), createTestKey(5))
//读取钥匙001和钥匙002
	itr.Next()
	itr.Next()
	itr.Close()
	s2.Done()

//模拟TX3，读取键_004和键_005
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	itr, _ = s3.GetStateRangeScanIterator(cID, createTestKey(4), createTestKey(6))
//读取钥匙001和钥匙002
	itr.Next()
	itr.Next()
	itr.Close()
	s3.Done()

//在提交tx2和tx3之前模拟tx4。修改TX3读取的密钥
	s4, _ := txMgr.NewTxSimulator("test_tx4")
	s4.DeleteState(cID, createTestKey(5))
	s4.Done()

//验证并提交tx4的rwset
	txRWSet4, _ := s4.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet4.PubSimulationResults)

//RWSet tx3 should be invalid now
	txRWSet3, _ := s3.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet3.PubSimulationResults)

//tx2仍然有效
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

}

func TestGetSetMultipeKeys(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testgetsetmultipekeys"
		testEnv.init(t, testLedgerID, nil)
		testGetSetMultipeKeys(t, testEnv)
		testEnv.cleanup()
	}
}

func testGetSetMultipeKeys(t *testing.T, env testEnv) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
//模拟TX1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	multipleKeyMap := make(map[string][]byte)
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		multipleKeyMap[k] = v
	}
	s1.SetStateMultipleKeys(cID, multipleKeyMap)
	s1.Done()
//验证并提交rwset
	txRWSet, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)
	qe, _ := txMgr.NewQueryExecutor("test_tx2")
	defer qe.Done()
	multipleKeys := []string{}
	for k := range multipleKeyMap {
		multipleKeys = append(multipleKeys, k)
	}
	values, _ := qe.GetStateMultipleKeys(cID, multipleKeys)
	assert.Len(t, values, 10)
	for i, v := range values {
		assert.Equal(t, multipleKeyMap[multipleKeys[i]], v)
	}

	s2, _ := txMgr.NewTxSimulator("test_tx3")
	defer s2.Done()
	values, _ = s2.GetStateMultipleKeys(cID, multipleKeys[5:7])
	assert.Len(t, values, 2)
	for i, v := range values {
		assert.Equal(t, multipleKeyMap[multipleKeys[i+5]], v)
	}
}

func createTestKey(i int) string {
	if i == 0 {
		return ""
	}
	return fmt.Sprintf("key_%03d", i)
}

func createTestValue(i int) []byte {
	return []byte(fmt.Sprintf("value_%03d", i))
}

//TestExecuteQueryQuery is only tested on the CouchDB testEnv
func TestExecuteQuery(t *testing.T) {

	for _, testEnv := range testEnvs {
//查询仅在couchdb testenv上受支持和测试
		if testEnv.getName() == couchDBtestEnvName {
			t.Logf("Running test for TestEnv = %s", testEnv.getName())
			testLedgerID := "testexecutequery"
			testEnv.init(t, testLedgerID, nil)
			testExecuteQuery(t, testEnv)
			testEnv.cleanup()
		}
	}
}

func testExecuteQuery(t *testing.T, env testEnv) {

	type Asset struct {
		ID        string `json:"_id"`
		Rev       string `json:"_rev"`
		AssetName string `json:"asset_name"`
		Color     string `json:"color"`
		Size      string `json:"size"`
		Owner     string `json:"owner"`
	}

	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

	s1, _ := txMgr.NewTxSimulator("test_tx1")

	s1.SetState("ns1", "key1", []byte("value1"))
	s1.SetState("ns1", "key2", []byte("value2"))
	s1.SetState("ns1", "key3", []byte("value3"))
	s1.SetState("ns1", "key4", []byte("value4"))
	s1.SetState("ns1", "key5", []byte("value5"))
	s1.SetState("ns1", "key6", []byte("value6"))
	s1.SetState("ns1", "key7", []byte("value7"))
	s1.SetState("ns1", "key8", []byte("value8"))

	s1.SetState("ns1", "key9", []byte(`{"asset_name":"marble1","color":"red","size":"25","owner":"jerry"}`))
	s1.SetState("ns1", "key10", []byte(`{"asset_name":"marble2","color":"blue","size":"10","owner":"bob"}`))
	s1.SetState("ns1", "key11", []byte(`{"asset_name":"marble3","color":"blue","size":"35","owner":"jerry"}`))
	s1.SetState("ns1", "key12", []byte(`{"asset_name":"marble4","color":"green","size":"15","owner":"bob"}`))
	s1.SetState("ns1", "key13", []byte(`{"asset_name":"marble5","color":"red","size":"35","owner":"jerry"}`))
	s1.SetState("ns1", "key14", []byte(`{"asset_name":"marble6","color":"blue","size":"25","owner":"bob"}`))

	s1.Done()

//验证并提交rwset
	txRWSet, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx2")
	queryString := "{\"selector\":{\"owner\": {\"$eq\": \"bob\"}},\"limit\": 10,\"skip\": 0}"

	itr, err := queryExecuter.ExecuteQuery("ns1", queryString)
	assert.NoError(t, err, "Error upon ExecuteQuery()")
	counter := 0
	for {
		queryRecord, _ := itr.Next()
		if queryRecord == nil {
			break
		}
//将文档取消标记为资产结构
		assetResp := &Asset{}
		json.Unmarshal(queryRecord.(*queryresult.KV).Value, &assetResp)
//验证所有者检索到的匹配项
		assert.Equal(t, "bob", assetResp.Owner)
		counter++
	}
//确保查询返回3个文档
	assert.Equal(t, 3, counter)
}

//仅在couchdb testenv上测试testExecutePagedQuery
func TestExecutePaginatedQuery(t *testing.T) {

	for _, testEnv := range testEnvs {
//查询仅在couchdb testenv上受支持和测试
		if testEnv.getName() == couchDBtestEnvName {
			t.Logf("Running test for TestEnv = %s", testEnv.getName())
			testLedgerID := "testexecutepaginatedquery"
			testEnv.init(t, testLedgerID, nil)
			testExecutePaginatedQuery(t, testEnv)
			testEnv.cleanup()
		}
	}
}

func testExecutePaginatedQuery(t *testing.T, env testEnv) {

	type Asset struct {
		ID        string `json:"_id"`
		Rev       string `json:"_rev"`
		AssetName string `json:"asset_name"`
		Color     string `json:"color"`
		Size      string `json:"size"`
		Owner     string `json:"owner"`
	}

	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

	s1, _ := txMgr.NewTxSimulator("test_tx1")

	s1.SetState("ns1", "key1", []byte(`{"asset_name":"marble1","color":"red","size":"25","owner":"jerry"}`))
	s1.SetState("ns1", "key2", []byte(`{"asset_name":"marble2","color":"blue","size":"10","owner":"bob"}`))
	s1.SetState("ns1", "key3", []byte(`{"asset_name":"marble3","color":"blue","size":"35","owner":"jerry"}`))
	s1.SetState("ns1", "key4", []byte(`{"asset_name":"marble4","color":"green","size":"15","owner":"bob"}`))
	s1.SetState("ns1", "key5", []byte(`{"asset_name":"marble5","color":"red","size":"35","owner":"jerry"}`))
	s1.SetState("ns1", "key6", []byte(`{"asset_name":"marble6","color":"blue","size":"25","owner":"bob"}`))

	s1.Done()

//验证并提交rwset
	txRWSet, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx2")
	queryString := `{"selector":{"owner":{"$eq":"bob"}}}`

	queryOptions := map[string]interface{}{
		"limit": int32(2),
	}

	itr, err := queryExecuter.ExecuteQueryWithMetadata("ns1", queryString, queryOptions)
	assert.NoError(t, err, "Error upon ExecuteQueryWithMetadata()")
	counter := 0
	for {
		queryRecord, _ := itr.Next()
		if queryRecord == nil {
			break
		}
//将文档取消标记为资产结构
		assetResp := &Asset{}
		json.Unmarshal(queryRecord.(*queryresult.KV).Value, &assetResp)
//验证所有者检索到的匹配项
		assert.Equal(t, "bob", assetResp.Owner)
		counter++
	}
//确保查询返回2个文档
	assert.Equal(t, 2, counter)

	bookmark := itr.GetBookmarkAndClose()

	queryOptions = map[string]interface{}{
		"limit": int32(2),
	}
	if bookmark != "" {
		queryOptions["bookmark"] = bookmark
	}

	itr, err = queryExecuter.ExecuteQueryWithMetadata("ns1", queryString, queryOptions)
	assert.NoError(t, err, "Error upon ExecuteQuery()")
	counter = 0
	for {
		queryRecord, _ := itr.Next()
		if queryRecord == nil {
			break
		}
//将文档取消标记为资产结构
		assetResp := &Asset{}
		json.Unmarshal(queryRecord.(*queryresult.KV).Value, &assetResp)
//验证所有者检索到的匹配项
		assert.Equal(t, "bob", assetResp.Owner)
		counter++
	}
//确保查询返回1个文档
	assert.Equal(t, 1, counter)
}

func TestValidateKey(t *testing.T) {
	nonUTF8Key := string([]byte{0xff, 0xff})
	dummyValue := []byte("dummyValue")
	for _, testEnv := range testEnvs {
		testLedgerID := "test.validate.key"
		testEnv.init(t, testLedgerID, nil)
		txSimulator, _ := testEnv.getTxMgr().NewTxSimulator("test_tx1")
		err := txSimulator.SetState("ns1", nonUTF8Key, dummyValue)
		if testEnv.getName() == levelDBtestEnvName {
			assert.NoError(t, err)
		}
		if testEnv.getName() == couchDBtestEnvName {
			assert.Error(t, err)
		}
		testEnv.cleanup()
	}
}

//TestTxSimulatorUnsupportedTx验证模拟在不支持的事务时是否必须引发错误
//是Perfromed-在只读事务中支持对私有数据的查询
func TestTxSimulatorUnsupportedTx(t *testing.T) {
	testEnv := testEnvs[0]
	testEnv.init(t, "testtxsimulatorunsupportedtx", nil)
	defer testEnv.cleanup()
	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll2"},
			{"ns1", "coll3"},
			{"ns1", "coll4"},
		},
		version.NewHeight(1, 1))

	simulator, _ := txMgr.NewTxSimulator("txid1")
	err := simulator.SetState("ns", "key", []byte("value"))
	assert.NoError(t, err)
	_, err = simulator.GetPrivateDataRangeScanIterator("ns1", "coll1", "startKey", "endKey")
	_, ok := err.(*txmgr.ErrUnsupportedTransaction)
	assert.True(t, ok)

	simulator, _ = txMgr.NewTxSimulator("txid2")
	_, err = simulator.GetPrivateDataRangeScanIterator("ns1", "coll1", "startKey", "endKey")
	assert.NoError(t, err)
	err = simulator.SetState("ns", "key", []byte("value"))
	_, ok = err.(*txmgr.ErrUnsupportedTransaction)
	assert.True(t, ok)

	queryOptions := map[string]interface{}{
		"limit": int32(2),
	}

	simulator, _ = txMgr.NewTxSimulator("txid3")
	err = simulator.SetState("ns", "key", []byte("value"))
	assert.NoError(t, err)
	_, err = simulator.GetStateRangeScanIteratorWithMetadata("ns1", "startKey", "endKey", queryOptions)
	_, ok = err.(*txmgr.ErrUnsupportedTransaction)
	assert.True(t, ok)

	simulator, _ = txMgr.NewTxSimulator("txid4")
	_, err = simulator.GetStateRangeScanIteratorWithMetadata("ns1", "startKey", "endKey", queryOptions)
	assert.NoError(t, err)
	err = simulator.SetState("ns", "key", []byte("value"))
	_, ok = err.(*txmgr.ErrUnsupportedTransaction)
	assert.True(t, ok)

}

//testxtSimulatoryUnsupportedTx仅在couchdb testenv上测试
func TestTxSimulatorQueryUnsupportedTx(t *testing.T) {

	for _, testEnv := range testEnvs {
//查询仅在couchdb testenv上受支持和测试
		if testEnv.getName() == couchDBtestEnvName {
			t.Logf("Running test for TestEnv = %s", testEnv.getName())
			testLedgerID := "testtxsimulatorunsupportedtxqueries"
			testEnv.init(t, testLedgerID, nil)
			testTxSimulatorQueryUnsupportedTx(t, testEnv)
			testEnv.cleanup()
		}
	}
}

func testTxSimulatorQueryUnsupportedTx(t *testing.T, env testEnv) {

	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

	s1, _ := txMgr.NewTxSimulator("test_tx1")

	s1.SetState("ns1", "key1", []byte(`{"asset_name":"marble1","color":"red","size":"25","owner":"jerry"}`))

	s1.Done()

//验证并提交rwset
	txRWSet, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)

	queryString := `{"selector":{"owner":{"$eq":"bob"}}}`
	queryOptions := map[string]interface{}{
		"limit": int32(2),
	}

	simulator, _ := txMgr.NewTxSimulator("txid1")
	err := simulator.SetState("ns1", "key1", []byte(`{"asset_name":"marble1","color":"red","size":"25","owner":"jerry"}`))
	assert.NoError(t, err)
	_, err = simulator.ExecuteQueryWithMetadata("ns1", queryString, queryOptions)
	_, ok := err.(*txmgr.ErrUnsupportedTransaction)
	assert.True(t, ok)

	simulator, _ = txMgr.NewTxSimulator("txid2")
	_, err = simulator.ExecuteQueryWithMetadata("ns1", queryString, queryOptions)
	assert.NoError(t, err)
	err = simulator.SetState("ns1", "key1", []byte(`{"asset_name":"marble1","color":"red","size":"25","owner":"jerry"}`))
	_, ok = err.(*txmgr.ErrUnsupportedTransaction)
	assert.True(t, ok)

}

func TestConstructUniquePvtData(t *testing.T) {
	v1 := []byte{1}
//NS1-coll1-key1应该被拒绝，因为它将来会被blk2tx1更新。
	pvtDataBlk1Tx1 := producePvtdata(t, 1, []string{"ns1:coll1"}, []string{"key1"}, [][]byte{v1})
//ns1-coll2-key3 should be accepted but ns1-coll1-key2 as it is updated in the future by Blk2Tx2
	pvtDataBlk1Tx2 := producePvtdata(t, 2, []string{"ns1:coll1", "ns1:coll2"}, []string{"key2", "key3"}, [][]byte{v1, v1})
//应接受NS1-coll2-key4
	pvtDataBlk1Tx3 := producePvtdata(t, 3, []string{"ns1:coll2"}, []string{"key4"}, [][]byte{v1})

	v2 := []byte{2}
//NS1-coll1-key1应该被拒绝，因为它将来会被blk3tx1更新。
	pvtDataBlk2Tx1 := producePvtdata(t, 1, []string{"ns1:coll1"}, []string{"key1"}, [][]byte{v2})
//NS1-COL1-KEY2应被接受
	pvtDataBlk2Tx2 := producePvtdata(t, 2, []string{"ns1:coll1"}, []string{"key2"}, [][]byte{nil})

	v3 := []byte{3}
//应接受NS1-coll1-key1
	pvtDataBlk3Tx1 := producePvtdata(t, 1, []string{"ns1:coll1"}, []string{"key1"}, [][]byte{v3})

	blocksPvtData := map[uint64][]*ledger.TxPvtData{
		1: {
			pvtDataBlk1Tx1,
			pvtDataBlk1Tx2,
			pvtDataBlk1Tx3,
		},
		2: {
			pvtDataBlk2Tx1,
			pvtDataBlk2Tx2,
		},
		3: {
			pvtDataBlk3Tx1,
		},
	}

	hashedCompositeKeyNs1Coll2Key3 := privacyenabledstate.HashedCompositeKey{Namespace: "ns1", CollectionName: "coll2", KeyHash: string(util.ComputeStringHash("key3"))}
	pvtKVWriteNs1Coll2Key3 := &privacyenabledstate.PvtKVWrite{Key: "key3", IsDelete: false, Value: v1, Version: version.NewHeight(1, 2)}

	hashedCompositeKeyNs1Coll2Key4 := privacyenabledstate.HashedCompositeKey{Namespace: "ns1", CollectionName: "coll2", KeyHash: string(util.ComputeStringHash("key4"))}
	pvtKVWriteNs1Coll2Key4 := &privacyenabledstate.PvtKVWrite{Key: "key4", IsDelete: false, Value: v1, Version: version.NewHeight(1, 3)}

	hashedCompositeKeyNs1Coll1Key2 := privacyenabledstate.HashedCompositeKey{Namespace: "ns1", CollectionName: "coll1", KeyHash: string(util.ComputeStringHash("key2"))}
	pvtKVWriteNs1Coll1Key2 := &privacyenabledstate.PvtKVWrite{Key: "key2", IsDelete: true, Value: nil, Version: version.NewHeight(2, 2)}

	hashedCompositeKeyNs1Coll1Key1 := privacyenabledstate.HashedCompositeKey{Namespace: "ns1", CollectionName: "coll1", KeyHash: string(util.ComputeStringHash("key1"))}
	pvtKVWriteNs1Coll1Key1 := &privacyenabledstate.PvtKVWrite{Key: "key1", IsDelete: false, Value: v3, Version: version.NewHeight(3, 1)}

	expectedUniquePvtData := uniquePvtDataMap{
		hashedCompositeKeyNs1Coll2Key3: pvtKVWriteNs1Coll2Key3,
		hashedCompositeKeyNs1Coll2Key4: pvtKVWriteNs1Coll2Key4,
		hashedCompositeKeyNs1Coll1Key2: pvtKVWriteNs1Coll1Key2,
		hashedCompositeKeyNs1Coll1Key1: pvtKVWriteNs1Coll1Key1,
	}

	uniquePvtData, err := constructUniquePvtData(blocksPvtData)
	assert.NoError(t, err)
	assert.Equal(t, expectedUniquePvtData, uniquePvtData)
}

func TestFindAndRemoveStalePvtData(t *testing.T) {
	ledgerid := "TestFindAndRemoveStalePvtData"
	testEnv := testEnvs[0]
	testEnv.init(t, ledgerid, nil)
	defer testEnv.cleanup()
	db := testEnv.getVDB()

	batch := privacyenabledstate.NewUpdateBatch()
	batch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("value_1_1_1"), version.NewHeight(1, 1))
	batch.HashUpdates.Put("ns1", "coll2", util.ComputeStringHash("key2"), util.ComputeStringHash("value_1_2_2"), version.NewHeight(1, 2))
	batch.HashUpdates.Put("ns2", "coll1", util.ComputeStringHash("key2"), util.ComputeStringHash("value_2_1_2"), version.NewHeight(2, 1))
	batch.HashUpdates.Put("ns2", "coll2", util.ComputeStringHash("key3"), util.ComputeStringHash("value_2_2_3"), version.NewHeight(10, 10))

//与哈希更新关联的所有pvt数据都丢失
	db.ApplyPrivacyAwareUpdates(batch, version.NewHeight(11, 1))

//为上述缺少的一些数据构造pvt数据。注意没有
//需要重复条目
//旧值，因此不应被接受
	hashedCompositeKeyNs1Coll1Key1 := privacyenabledstate.HashedCompositeKey{Namespace: "ns1", CollectionName: "coll1", KeyHash: string(util.ComputeStringHash("key1"))}
	pvtKVWriteNs1Coll1Key1 := &privacyenabledstate.PvtKVWrite{Key: "key1", IsDelete: false, Value: []byte("old_value_1_1_1"), Version: version.NewHeight(1, 0)}

//新的价值，因此应该被接受
	hashedCompositeKeyNs2Coll1Key2 := privacyenabledstate.HashedCompositeKey{Namespace: "ns2", CollectionName: "coll1", KeyHash: string(util.ComputeStringHash("key2"))}
	pvtKVWriteNs2Coll1Key2 := &privacyenabledstate.PvtKVWrite{Key: "key2", IsDelete: false, Value: []byte("value_2_1_2"), Version: version.NewHeight(2, 1)}

//删除--应该被接受
	hashedCompositeKeyNs1Coll3Key3 := privacyenabledstate.HashedCompositeKey{Namespace: "ns1", CollectionName: "coll3", KeyHash: string(util.ComputeStringHash("key3"))}
	pvtKVWriteNs1Coll3Key3 := &privacyenabledstate.PvtKVWrite{Key: "key3", IsDelete: true, Value: nil, Version: version.NewHeight(2, 3)}

//版本不匹配，但哈希值必须相同。因此，
//这也应该被接受
	hashedCompositeKeyNs2Coll2Key3 := privacyenabledstate.HashedCompositeKey{Namespace: "ns2", CollectionName: "coll2", KeyHash: string(util.ComputeStringHash("key3"))}
	pvtKVWriteNs2Coll2Key3 := &privacyenabledstate.PvtKVWrite{Key: "key3", IsDelete: false, Value: []byte("value_2_2_3"), Version: version.NewHeight(9, 9)}

	uniquePvtData := uniquePvtDataMap{
		hashedCompositeKeyNs1Coll1Key1: pvtKVWriteNs1Coll1Key1,
		hashedCompositeKeyNs2Coll1Key2: pvtKVWriteNs2Coll1Key2,
		hashedCompositeKeyNs1Coll3Key3: pvtKVWriteNs1Coll3Key3,
		hashedCompositeKeyNs2Coll2Key3: pvtKVWriteNs2Coll2Key3,
	}

//已从validateAndPreparaTchForpvtDataofOldBlocks创建预期批处理
	expectedBatch := privacyenabledstate.NewUpdateBatch()
	expectedBatch.PvtUpdates.Put("ns2", "coll1", "key2", []byte("value_2_1_2"), version.NewHeight(2, 1))
	expectedBatch.PvtUpdates.Delete("ns1", "coll3", "key3", version.NewHeight(2, 3))
	expectedBatch.PvtUpdates.Put("ns2", "coll2", "key3", []byte("value_2_2_3"), version.NewHeight(10, 10))

	err := uniquePvtData.findAndRemoveStalePvtData(db)
	assert.NoError(t, err, "uniquePvtData.findAndRemoveStatePvtData resulted in an error")
	batch = uniquePvtData.transformToUpdateBatch()
	assert.Equal(t, expectedBatch.PvtUpdates, batch.PvtUpdates)
}

func producePvtdata(t *testing.T, txNum uint64, nsColls []string, keys []string, values [][]byte) *ledger.TxPvtData {
	builder := rwsetutil.NewRWSetBuilder()
	for index, nsColl := range nsColls {
		nsCollSplit := strings.Split(nsColl, ":")
		ns := nsCollSplit[0]
		coll := nsCollSplit[1]
		key := keys[index]
		value := values[index]
		builder.AddToPvtAndHashedWriteSet(ns, coll, key, value)
	}
	simRes, err := builder.GetTxSimulationResults()
	assert.NoError(t, err)
	return &ledger.TxPvtData{
		SeqInBlock: txNum,
		WriteSet:   simRes.PvtSimulationResults,
	}
}

func TestRemoveStaleAndCommitPvtDataOfOldBlocks(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testvalidationandcommitofoldpvtdata"
		testEnv.init(t, testLedgerID, nil)
		testValidationAndCommitOfOldPvtData(t, testEnv)
		testEnv.cleanup()
	}
}

func testValidationAndCommitOfOldPvtData(t *testing.T, env testEnv) {
	ledgerid := "testvalidationandcommitofoldpvtdata"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns1", "coll1"}: 0,
			{"ns1", "coll2"}: 0,
		},
	)
	env.init(t, ledgerid, btlPolicy)
	txMgr := env.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll2"},
		},
		version.NewHeight(1, 1),
	)

	db := env.getVDB()
	updateBatch := privacyenabledstate.NewUpdateBatch()
//所有pvt数据丢失
updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("value1"), version.NewHeight(1, 1)) //E1
updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key2"), util.ComputeStringHash("value2"), version.NewHeight(1, 2)) //E2
updateBatch.HashUpdates.Put("ns1", "coll2", util.ComputeStringHash("key3"), util.ComputeStringHash("value3"), version.NewHeight(1, 2)) //E3
updateBatch.HashUpdates.Put("ns1", "coll2", util.ComputeStringHash("key4"), util.ComputeStringHash("value4"), version.NewHeight(1, 3)) //E4
	db.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(1, 2))

	updateBatch = privacyenabledstate.NewUpdateBatch()
updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("new-value1"), version.NewHeight(2, 1)) //E1被更新
updateBatch.HashUpdates.Delete("ns1", "coll1", util.ComputeStringHash("key2"), version.NewHeight(2, 2))                                    //正在删除e2
	db.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(2, 2))

	updateBatch = privacyenabledstate.NewUpdateBatch()
updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("another-new-value1"), version.NewHeight(3, 1)) //e1再次更新
updateBatch.HashUpdates.Put("ns1", "coll2", util.ComputeStringHash("key3"), util.ComputeStringHash("value3"), version.NewHeight(3, 2))             //E3 gets only metadata update
	db.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(3, 2))

	v1 := []byte("value1")
//NS1-coll1-key1应该被拒绝，因为它将来会被blk2tx1更新。
	pvtDataBlk1Tx1 := producePvtdata(t, 1, []string{"ns1:coll1"}, []string{"key1"}, [][]byte{v1})
//应接受NS1-coll2-key3，但NS1-coll1-key2
//should be rejected as it is updated in the future by Blk2Tx2
	v2 := []byte("value2")
	v3 := []byte("value3")
	pvtDataBlk1Tx2 := producePvtdata(t, 2, []string{"ns1:coll1", "ns1:coll2"}, []string{"key2", "key3"}, [][]byte{v2, v3})
//应接受NS1-coll2-key4
	v4 := []byte("value4")
	pvtDataBlk1Tx3 := producePvtdata(t, 3, []string{"ns1:coll2"}, []string{"key4"}, [][]byte{v4})

	nv1 := []byte("new-value1")
//NS1-coll1-key1应该被拒绝，因为它将来会被blk3tx1更新。
	pvtDataBlk2Tx1 := producePvtdata(t, 1, []string{"ns1:coll1"}, []string{"key1"}, [][]byte{nv1})
//应接受ns1-coll1-key2——删除操作
	pvtDataBlk2Tx2 := producePvtdata(t, 2, []string{"ns1:coll1"}, []string{"key2"}, [][]byte{nil})

	anv1 := []byte("another-new-value1")
//应接受NS1-coll1-key1
	pvtDataBlk3Tx1 := producePvtdata(t, 1, []string{"ns1:coll1"}, []string{"key1"}, [][]byte{anv1})
//NS1-coll2-key3应该被接受——假设只更新元数据
	pvtDataBlk3Tx2 := producePvtdata(t, 2, []string{"ns1:coll2"}, []string{"key3"}, [][]byte{v3})

	blocksPvtData := map[uint64][]*ledger.TxPvtData{
		1: {
			pvtDataBlk1Tx1,
			pvtDataBlk1Tx2,
			pvtDataBlk1Tx3,
		},
		2: {
			pvtDataBlk2Tx1,
			pvtDataBlk2Tx2,
		},
		3: {
			pvtDataBlk3Tx1,
			pvtDataBlk3Tx2,
		},
	}

	err := txMgr.RemoveStaleAndCommitPvtDataOfOldBlocks(blocksPvtData)
	assert.NoError(t, err)

	vv, err := db.GetPrivateData("ns1", "coll1", "key1")
	assert.NoError(t, err)
assert.Equal(t, anv1, vv.Value) //上次更新的值

	vv, err = db.GetPrivateData("ns1", "coll1", "key2")
	assert.NoError(t, err)
assert.Equal(t, nil, nil) //删除

	vv, err = db.GetPrivateData("ns1", "coll2", "key3")
	assert.NoError(t, err)
	assert.Equal(t, v3, vv.Value)
assert.Equal(t, version.NewHeight(3, 2), vv.Version) //though we passed with version {1,2}, we should get {3,2} due to metadata update

	vv, err = db.GetPrivateData("ns1", "coll2", "key4")
	assert.NoError(t, err)
	assert.Equal(t, v4, vv.Value)
}

func TestTxSimulatorMissingPvtdata(t *testing.T) {
	testEnv := testEnvs[0]
	testEnv.init(t, "TestTxSimulatorUnsupportedTxQueries", nil)
	defer testEnv.cleanup()

	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll2"},
			{"ns1", "coll3"},
			{"ns1", "coll4"},
		},
		version.NewHeight(1, 1),
	)

	db := testEnv.getVDB()
	updateBatch := privacyenabledstate.NewUpdateBatch()
	updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("value1"), version.NewHeight(1, 1))
	updateBatch.PvtUpdates.Put("ns1", "coll1", "key1", []byte("value1"), version.NewHeight(1, 1))
	db.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(1, 1))

	assert.True(t, testPvtValueEqual(t, txMgr, "ns1", "coll1", "key1", []byte("value1")))

	updateBatch = privacyenabledstate.NewUpdateBatch()
	updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("value1"), version.NewHeight(2, 1))
	updateBatch.HashUpdates.Put("ns1", "coll2", util.ComputeStringHash("key2"), util.ComputeStringHash("value2"), version.NewHeight(2, 1))
	updateBatch.HashUpdates.Put("ns1", "coll3", util.ComputeStringHash("key3"), util.ComputeStringHash("value3"), version.NewHeight(2, 1))
	updateBatch.PvtUpdates.Put("ns1", "coll3", "key3", []byte("value3"), version.NewHeight(2, 1))
	db.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(2, 1))

	assert.False(t, testPvtKeyExist(t, txMgr, "ns1", "coll1", "key1"))

	assert.False(t, testPvtKeyExist(t, txMgr, "ns1", "coll2", "key2"))

	assert.True(t, testPvtValueEqual(t, txMgr, "ns1", "coll3", "key3", []byte("value3")))

	assert.True(t, testPvtValueEqual(t, txMgr, "ns1", "coll4", "key4", nil))
}

func TestRemoveStaleAndCommitPvtDataOfOldBlocksWithExpiry(t *testing.T) {
	ledgerid := "TestTxSimulatorMissingPvtdataExpiry"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns", "coll"}: 1,
		},
	)
	testEnv := testEnvs[0]
	testEnv.init(t, ledgerid, btlPolicy)
	defer testEnv.cleanup()

	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns", "coll"},
		},
		version.NewHeight(1, 1),
	)

	bg, _ := testutil.NewBlockGenerator(t, ledgerid, false)

//存储哈希数据，但缺少pvt密钥
//提交块3时，存储的pvt密钥将过期并被清除
	blkAndPvtdata := prepareNextBlockForTest(t, txMgr, bg, "txid-1",
		map[string]string{"pubkey1": "pub-value1"}, map[string]string{"pvtkey1": "pvt-value1"}, true)
	_, err := txMgr.ValidateAndPrepare(blkAndPvtdata, true)
	assert.NoError(t, err)
//提交块1
	assert.NoError(t, txMgr.Commit())

//pvt数据不应存在
	assert.False(t, testPvtKeyExist(t, txMgr, "ns", "coll", "pvtkey1"))

//提交块1的pvt数据
	v1 := []byte("pvt-value1")
	pvtDataBlk1Tx1 := producePvtdata(t, 1, []string{"ns:coll"}, []string{"pvtkey1"}, [][]byte{v1})
	blocksPvtData := map[uint64][]*ledger.TxPvtData{
		1: {
			pvtDataBlk1Tx1,
		},
	}
	err = txMgr.RemoveStaleAndCommitPvtDataOfOldBlocks(blocksPvtData)
	assert.NoError(t, err)

//pvt data should exist
	assert.True(t, testPvtValueEqual(t, txMgr, "ns", "coll", "pvtkey1", v1))

//存储哈希数据，但缺少pvt密钥
//提交块4时，存储的pvt密钥将过期并被清除
	blkAndPvtdata = prepareNextBlockForTest(t, txMgr, bg, "txid-2",
		map[string]string{"pubkey2": "pub-value2"}, map[string]string{"pvtkey2": "pvt-value2"}, true)
	_, err = txMgr.ValidateAndPrepare(blkAndPvtdata, true)
	assert.NoError(t, err)
//提交块2
	assert.NoError(t, txMgr.Commit())

//pvt数据不应存在
	assert.False(t, testPvtKeyExist(t, txMgr, "ns", "coll", "pvtkey2"))

	blkAndPvtdata = prepareNextBlockForTest(t, txMgr, bg, "txid-3",
		map[string]string{"pubkey3": "pub-value3"}, nil, false)
	_, err = txMgr.ValidateAndPrepare(blkAndPvtdata, true)
	assert.NoError(t, err)
//提交块3
	assert.NoError(t, txMgr.Commit())

//PrepareForExpiringKey必须像选择pvtkey2一样选择了pvtkey2
//在下一个块提交期间过期

//提交块2的pvt数据
	v2 := []byte("pvt-value2")
	pvtDataBlk2Tx1 := producePvtdata(t, 1, []string{"ns:coll"}, []string{"pvtkey2"}, [][]byte{v2})
	blocksPvtData = map[uint64][]*ledger.TxPvtData{
		2: {
			pvtDataBlk2Tx1,
		},
	}

	err = txMgr.RemoveStaleAndCommitPvtDataOfOldBlocks(blocksPvtData)
	assert.NoError(t, err)

//应存在pvt数据
	assert.True(t, testPvtValueEqual(t, txMgr, "ns", "coll", "pvtkey2", v2))

	blkAndPvtdata = prepareNextBlockForTest(t, txMgr, bg, "txid-4",
		map[string]string{"pubkey4": "pub-value4"}, nil, false)
	_, err = txMgr.ValidateAndPrepare(blkAndPvtdata, true)
	assert.NoError(t, err)
//提交块4，应清除pvtkey2
	assert.NoError(t, txMgr.Commit())

	assert.True(t, testPvtValueEqual(t, txMgr, "ns", "coll", "pvtkey2", nil))
}

func testPvtKeyExist(t *testing.T, txMgr txmgr.TxMgr, ns, coll, key string) bool {
	simulator, _ := txMgr.NewTxSimulator("tx-tmp")
	defer simulator.Done()
	_, err := simulator.GetPrivateData(ns, coll, key)
	_, ok := err.(*txmgr.ErrPvtdataNotAvailable)
	return !ok
}

func testPvtValueEqual(t *testing.T, txMgr txmgr.TxMgr, ns, coll, key string, value []byte) bool {
	simulator, _ := txMgr.NewTxSimulator("tx-tmp")
	defer simulator.Done()
	pvtValue, err := simulator.GetPrivateData(ns, coll, key)
	assert.NoError(t, err)
	if bytes.Compare(pvtValue, value) == 0 {
		return true
	}
	return false
}

func TestDeleteOnCursor(t *testing.T) {
	cID := "cid"
	env := testEnvs[0]
	env.init(t, "TestDeleteOnCursor", nil)
	defer env.cleanup()

	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

//模拟并提交tx1以填充示例数据（key_001到key_010）
	s, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s.SetState(cID, k, v)
	}
	s.Done()
	txRWSet1, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

//模拟并提交Tx2，该Tx2读取键_001到键_004并逐个删除它们（在循环中-itr.next（）后跟delete（））
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	itr2, _ := s2.GetStateRangeScanIterator(cID, createTestKey(1), createTestKey(5))
	for i := 1; i <= 4; i++ {
		kv, err := itr2.Next()
		assert.NoError(t, err)
		assert.NotNil(t, kv)
		key := kv.(*queryresult.KV).Key
		s2.DeleteState(cID, key)
	}
	itr2.Close()
	s2.Done()
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

//simulate tx3 to verify that the keys key_001 through key_004 got deleted
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	itr3, _ := s3.GetStateRangeScanIterator(cID, createTestKey(1), createTestKey(10))
	kv, err := itr3.Next()
	assert.NoError(t, err)
	assert.NotNil(t, kv)
	key := kv.(*queryresult.KV).Key
	assert.Equal(t, "key_005", key)
	itr3.Close()
	s3.Done()
}

func TestTxSimulatorMissingPvtdataExpiry(t *testing.T) {
	ledgerid := "TestTxSimulatorMissingPvtdataExpiry"
	testEnv := testEnvs[0]
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns", "coll"}: 1,
		},
	)
	testEnv.init(t, ledgerid, btlPolicy)
	defer testEnv.cleanup()

	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr), []collConfigkey{{"ns", "coll"}}, version.NewHeight(1, 1))

	viper.Set(fmt.Sprintf("ledger.pvtdata.btlpolicy.%s.ns.coll", ledgerid), 1)
	bg, _ := testutil.NewBlockGenerator(t, ledgerid, false)

	blkAndPvtdata := prepareNextBlockForTest(t, txMgr, bg, "txid-1",
		map[string]string{"pubkey1": "pub-value1"}, map[string]string{"pvtkey1": "pvt-value1"}, false)
	_, err := txMgr.ValidateAndPrepare(blkAndPvtdata, true)
	assert.NoError(t, err)
	assert.NoError(t, txMgr.Commit())

	assert.True(t, testPvtValueEqual(t, txMgr, "ns", "coll", "pvtkey1", []byte("pvt-value1")))

	blkAndPvtdata = prepareNextBlockForTest(t, txMgr, bg, "txid-2",

		map[string]string{"pubkey1": "pub-value2"}, map[string]string{"pvtkey2": "pvt-value2"}, false)
	_, err = txMgr.ValidateAndPrepare(blkAndPvtdata, true)
	assert.NoError(t, err)
	assert.NoError(t, txMgr.Commit())

	assert.True(t, testPvtValueEqual(t, txMgr, "ns", "coll", "pvtkey1", []byte("pvt-value1")))

	blkAndPvtdata = prepareNextBlockForTest(t, txMgr, bg, "txid-2",
		map[string]string{"pubkey1": "pub-value3"}, map[string]string{"pvtkey3": "pvt-value3"}, false)
	_, err = txMgr.ValidateAndPrepare(blkAndPvtdata, true)
	assert.NoError(t, err)
	assert.NoError(t, txMgr.Commit())

	assert.True(t, testPvtValueEqual(t, txMgr, "ns", "coll", "pvtkey1", nil))
}

func TestTxWithPubMetadata(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxwithpubmetadata"
		testEnv.init(t, testLedgerID, nil)
		testTxWithPubMetadata(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxWithPubMetadata(t *testing.T, env testEnv) {
	namespace := "testns"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

//模拟并提交tx1-设置key1和key2的val和元数据。仅为键3设置元数据
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	key1, value1, metadata1 := "key1", []byte("value1"), map[string][]byte{"entry1": []byte("meatadata1-entry1")}
	key2, value2, metadata2 := "key2", []byte("value2"), map[string][]byte{"entry1": []byte("meatadata2-entry1")}
	key3, metadata3 := "key3", map[string][]byte{"entry1": []byte("meatadata3-entry")}

	s1.SetState(namespace, key1, value1)
	s1.SetStateMetadata(namespace, key1, metadata1)
	s1.SetState(namespace, key2, value2)
	s1.SetStateMetadata(namespace, key2, metadata2)
	s1.SetStateMetadata(namespace, key3, metadata3)
	s1.Done()
	txRWSet1, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

//运行查询-key1和key2应该同时返回值和元数据。key3在db中仍应是不存在的
	qe, _ := txMgr.NewQueryExecutor("test_tx2")
	checkTestQueryResults(t, qe, namespace, key1, value1, metadata1)
	checkTestQueryResults(t, qe, namespace, key2, value2, metadata2)
	checkTestQueryResults(t, qe, namespace, key3, nil, nil)
	qe.Done()

//模拟并提交tx3-更新key1的元数据并删除key2的元数据
	updatedMetadata1 := map[string][]byte{"entry1": []byte("meatadata1-entry1"), "entry2": []byte("meatadata1-entry2")}
	s2, _ := txMgr.NewTxSimulator("test_tx3")
	s2.SetStateMetadata(namespace, key1, updatedMetadata1)
	s2.DeleteStateMetadata(namespace, key2)
	s2.Done()
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

//运行查询-key1应返回更新的元数据。key2应返回“nil”元数据
	qe, _ = txMgr.NewQueryExecutor("test_tx4")
	checkTestQueryResults(t, qe, namespace, key1, value1, updatedMetadata1)
	checkTestQueryResults(t, qe, namespace, key2, value2, nil)
	qe.Done()
}

func TestTxWithPvtdataMetadata(t *testing.T) {
	ledgerid, ns, coll := "testtxwithpvtdatametadata", "ns", "coll"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns", "coll"}: 1000,
		},
	)
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testEnv.init(t, ledgerid, btlPolicy)
		testTxWithPvtdataMetadata(t, testEnv, ns, coll)
		testEnv.cleanup()
	}
}

func testTxWithPvtdataMetadata(t *testing.T, env testEnv, ns, coll string) {
	ledgerid := "testtxwithpvtdatametadata"
	txMgr := env.getTxMgr()
	bg, _ := testutil.NewBlockGenerator(t, ledgerid, false)

	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr), []collConfigkey{{"ns", "coll"}}, version.NewHeight(1, 1))

//模拟并提交tx1-设置key1和key2的val和元数据。仅为键3设置元数据
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	key1, value1, metadata1 := "key1", []byte("value1"), map[string][]byte{"entry1": []byte("meatadata1-entry1")}
	key2, value2, metadata2 := "key2", []byte("value2"), map[string][]byte{"entry1": []byte("meatadata2-entry1")}
	key3, metadata3 := "key3", map[string][]byte{"entry1": []byte("meatadata3-entry")}
	s1.SetPrivateData(ns, coll, key1, value1)
	s1.SetPrivateDataMetadata(ns, coll, key1, metadata1)
	s1.SetPrivateData(ns, coll, key2, value2)
	s1.SetPrivateDataMetadata(ns, coll, key2, metadata2)
	s1.SetPrivateDataMetadata(ns, coll, key3, metadata3)
	s1.Done()

	blkAndPvtdata1 := prepareNextBlockForTestFromSimulator(t, bg, s1)
	_, err := txMgr.ValidateAndPrepare(blkAndPvtdata1, true)
	assert.NoError(t, err)
	assert.NoError(t, txMgr.Commit())

//运行查询-key1和key2应该同时返回值和元数据。key3在db中仍应是不存在的
	qe, _ := txMgr.NewQueryExecutor("test_tx2")
	checkPvtdataTestQueryResults(t, qe, ns, coll, key1, value1, metadata1)
	checkPvtdataTestQueryResults(t, qe, ns, coll, key2, value2, metadata2)
	checkPvtdataTestQueryResults(t, qe, ns, coll, key3, nil, nil)
	qe.Done()

//模拟并提交tx3-更新key1的元数据并删除key2的元数据
	updatedMetadata1 := map[string][]byte{"entry1": []byte("meatadata1-entry1"), "entry2": []byte("meatadata1-entry2")}
	s2, _ := txMgr.NewTxSimulator("test_tx3")
	s2.SetPrivateDataMetadata(ns, coll, key1, updatedMetadata1)
	s2.DeletePrivateDataMetadata(ns, coll, key2)
	s2.Done()

	blkAndPvtdata2 := prepareNextBlockForTestFromSimulator(t, bg, s2)
	_, err = txMgr.ValidateAndPrepare(blkAndPvtdata2, true)
	assert.NoError(t, err)
	assert.NoError(t, txMgr.Commit())

//运行查询-key1应返回更新的元数据。key2应返回“nil”元数据
	qe, _ = txMgr.NewQueryExecutor("test_tx4")
	checkPvtdataTestQueryResults(t, qe, ns, coll, key1, value1, updatedMetadata1)
	checkPvtdataTestQueryResults(t, qe, ns, coll, key2, value2, nil)
	qe.Done()
}

func prepareNextBlockForTest(t *testing.T, txMgr txmgr.TxMgr, bg *testutil.BlockGenerator,
	txid string, pubKVs map[string]string, pvtKVs map[string]string, isMissing bool) *ledger.BlockAndPvtData {
	simulator, _ := txMgr.NewTxSimulator(txid)
//模拟事务
	for k, v := range pubKVs {
		simulator.SetState("ns", k, []byte(v))
	}
	for k, v := range pvtKVs {
		simulator.SetPrivateData("ns", "coll", k, []byte(v))
	}
	simulator.Done()
	if isMissing {
		return prepareNextBlockForTestFromSimulatorWithMissingData(t, bg, simulator, txid, 1, "ns", "coll", true)
	}
	return prepareNextBlockForTestFromSimulator(t, bg, simulator)
}

func prepareNextBlockForTestFromSimulator(t *testing.T, bg *testutil.BlockGenerator, simulator ledger.TxSimulator) *ledger.BlockAndPvtData {
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimBytes, _ := simRes.GetPubSimulationBytes()
	block := bg.NextBlock([][]byte{pubSimBytes})
	return &ledger.BlockAndPvtData{Block: block,
		PvtData: ledger.TxPvtDataMap{0: {SeqInBlock: 0, WriteSet: simRes.PvtSimulationResults}},
	}
}

func prepareNextBlockForTestFromSimulatorWithMissingData(t *testing.T, bg *testutil.BlockGenerator, simulator ledger.TxSimulator,
	txid string, txNum uint64, ns, coll string, isEligible bool) *ledger.BlockAndPvtData {
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimBytes, _ := simRes.GetPubSimulationBytes()
	block := bg.NextBlock([][]byte{pubSimBytes})
	missingData := make(ledger.TxMissingPvtDataMap)
	missingData.Add(txNum, ns, coll, isEligible)
	return &ledger.BlockAndPvtData{Block: block, MissingPvtData: missingData}
}

func checkTestQueryResults(t *testing.T, qe ledger.QueryExecutor, ns, key string,
	expectedVal []byte, expectedMetadata map[string][]byte) {
	committedVal, err := qe.GetState(ns, key)
	assert.NoError(t, err)
	assert.Equal(t, expectedVal, committedVal)

	committedMetadata, err := qe.GetStateMetadata(ns, key)
	assert.NoError(t, err)
	assert.Equal(t, expectedMetadata, committedMetadata)
	t.Logf("key=%s, value=%s, metadata=%s", key, committedVal, committedMetadata)
}

func checkPvtdataTestQueryResults(t *testing.T, qe ledger.QueryExecutor, ns, coll, key string,
	expectedVal []byte, expectedMetadata map[string][]byte) {
	committedVal, err := qe.GetPrivateData(ns, coll, key)
	assert.NoError(t, err)
	assert.Equal(t, expectedVal, committedVal)

	committedMetadata, err := qe.GetPrivateDataMetadata(ns, coll, key)
	assert.NoError(t, err)
	assert.Equal(t, expectedMetadata, committedMetadata)
	t.Logf("key=%s, value=%s, metadata=%s", key, committedVal, committedMetadata)
}
