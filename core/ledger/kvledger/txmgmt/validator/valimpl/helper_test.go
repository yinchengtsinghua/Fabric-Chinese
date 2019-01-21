
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


package valimpl

import (
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/flogging/floggingtest"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/internal"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	lutils "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("internal=debug")
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/txmgmt/validator/internal")
	os.Exit(m.Run())
}

func TestValidateAndPreparePvtBatch(t *testing.T) {
	testDBEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	testDB := testDBEnv.GetDBHandle("emptydb")

	pubSimulationResults := [][]byte{}
	pvtDataMap := make(map[uint64]*ledger.TxPvtData)

	txids := []string{"tx1", "tx2", "tx3"}

//1。构建一个包含三个事务和
//通过调用PreprocessProtoBlock（）处理块
//得到一个预处理的块。

//德克萨斯州1
//获取tx1的模拟结果
	tx1SimulationResults := testutilSampleTxSimulationResults(t, "key1")
	res, err := tx1SimulationResults.GetPubSimulationBytes()
	assert.NoError(t, err)

//将tx1 public rwset添加到结果集
	pubSimulationResults = append(pubSimulationResults, res)

//将tx1 private rwset添加到私有数据映射
	tx1PvtData := &ledger.TxPvtData{SeqInBlock: 0, WriteSet: tx1SimulationResults.PvtSimulationResults}
	pvtDataMap[uint64(0)] = tx1PvtData

//德克萨斯州2
//获取tx2的模拟结果
	tx2SimulationResults := testutilSampleTxSimulationResults(t, "key2")
	res, err = tx2SimulationResults.GetPubSimulationBytes()
	assert.NoError(t, err)

//Add tx2 public rwset to the set of results
	pubSimulationResults = append(pubSimulationResults, res)

//由于tx2 private rwset不属于当前对等方拥有的集合，
//私有RWset未添加到私有数据映射中

//德克萨斯州3
//获取TX3的模拟结果
	tx3SimulationResults := testutilSampleTxSimulationResults(t, "key3")
	res, err = tx3SimulationResults.GetPubSimulationBytes()
	assert.NoError(t, err)

//将tx3 public rwset添加到结果集
	pubSimulationResults = append(pubSimulationResults, res)

//将tx3 private rwset添加到私有数据映射
	tx3PvtData := &ledger.TxPvtData{SeqInBlock: 2, WriteSet: tx3SimulationResults.PvtSimulationResults}
	pvtDataMap[uint64(2)] = tx3PvtData

//使用所有三个事务的模拟结果构造一个块
	block := testutil.ConstructBlockWithTxid(t, 10, testutil.ConstructRandomBytes(t, 32), pubSimulationResults, txids, false)

//从PreprocessProtoBlock（）构造预期的预处理块
	expectedPerProcessedBlock := &internal.Block{Num: 10}
	tx1TxRWSet, err := rwsetutil.TxRwSetFromProtoMsg(tx1SimulationResults.PubSimulationResults)
	assert.NoError(t, err)
	expectedPerProcessedBlock.Txs = append(expectedPerProcessedBlock.Txs, &internal.Transaction{IndexInBlock: 0, ID: "tx1", RWSet: tx1TxRWSet})

	tx2TxRWSet, err := rwsetutil.TxRwSetFromProtoMsg(tx2SimulationResults.PubSimulationResults)
	assert.NoError(t, err)
	expectedPerProcessedBlock.Txs = append(expectedPerProcessedBlock.Txs, &internal.Transaction{IndexInBlock: 1, ID: "tx2", RWSet: tx2TxRWSet})

	tx3TxRWSet, err := rwsetutil.TxRwSetFromProtoMsg(tx3SimulationResults.PubSimulationResults)
	assert.NoError(t, err)
	expectedPerProcessedBlock.Txs = append(expectedPerProcessedBlock.Txs, &internal.Transaction{IndexInBlock: 2, ID: "tx3", RWSet: tx3TxRWSet})
	alwaysValidKVFunc := func(key string, value []byte) error {
		return nil
	}
	actualPreProcessedBlock, _, err := preprocessProtoBlock(nil, alwaysValidKVFunc, block, false)
	assert.NoError(t, err)
	assert.Equal(t, expectedPerProcessedBlock, actualPreProcessedBlock)

//2。假设MVCC验证是在预处理的块上执行的，请设置适当的验证代码
//对于每个事务，然后调用validateAndPreparpvtBatch（）以获取经过验证的私有更新批。
//这里，validate是公共rwset中pvtrwset的散列与pvtrwset的实际散列的比较）

//为所有三个事务设置验证代码。三个事务之一被标记为无效
	mvccValidatedBlock := actualPreProcessedBlock
	mvccValidatedBlock.Txs[0].ValidationCode = peer.TxValidationCode_VALID
	mvccValidatedBlock.Txs[1].ValidationCode = peer.TxValidationCode_VALID
	mvccValidatedBlock.Txs[2].ValidationCode = peer.TxValidationCode_INVALID_OTHER_REASON

//构造预期的私有更新
	expectedPvtUpdates := privacyenabledstate.NewPvtUpdateBatch()
	tx1TxPvtRWSet, err := rwsetutil.TxPvtRwSetFromProtoMsg(tx1SimulationResults.PvtSimulationResults)
	assert.NoError(t, err)
	addPvtRWSetToPvtUpdateBatch(tx1TxPvtRWSet, expectedPvtUpdates, version.NewHeight(uint64(10), uint64(0)))

	actualPvtUpdates, err := validateAndPreparePvtBatch(mvccValidatedBlock, testDB, nil, pvtDataMap)
	assert.NoError(t, err)
	assert.Equal(t, expectedPvtUpdates, actualPvtUpdates)

	expectedtxsFilter := []uint8{uint8(peer.TxValidationCode_VALID), uint8(peer.TxValidationCode_VALID), uint8(peer.TxValidationCode_INVALID_OTHER_REASON)}

	postprocessProtoBlock(block, mvccValidatedBlock)
	assert.Equal(t, expectedtxsFilter, block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
}

func TestPreprocessProtoBlock(t *testing.T) {
	allwaysValidKVfunc := func(key string, value []byte) error {
		return nil
	}
//好块
//_u，gb：=testutil.newblockgenerator（t，“testledger”，false）
	gb := testutil.ConstructTestBlock(t, 10, 1, 1)
	_, _, err := preprocessProtoBlock(nil, allwaysValidKVfunc, gb, false)
	assert.NoError(t, err)
//坏信封
	gb = testutil.ConstructTestBlock(t, 11, 1, 1)
	gb.Data = &common.BlockData{Data: [][]byte{{123}}}
	gb.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] =
		lutils.NewTxValidationFlagsSetValue(len(gb.Data.Data), peer.TxValidationCode_VALID)
	_, _, err = preprocessProtoBlock(nil, allwaysValidKVfunc, gb, false)
	assert.Error(t, err)
	t.Log(err)
//不良有效载荷
	gb = testutil.ConstructTestBlock(t, 12, 1, 1)
	envBytes, _ := putils.GetBytesEnvelope(&common.Envelope{Payload: []byte{123}})
	gb.Data = &common.BlockData{Data: [][]byte{envBytes}}
	_, _, err = preprocessProtoBlock(nil, allwaysValidKVfunc, gb, false)
	assert.Error(t, err)
	t.Log(err)
//错误的频道标题
	gb = testutil.ConstructTestBlock(t, 13, 1, 1)
	payloadBytes, _ := putils.GetBytesPayload(&common.Payload{
		Header: &common.Header{ChannelHeader: []byte{123}},
	})
	envBytes, _ = putils.GetBytesEnvelope(&common.Envelope{Payload: payloadBytes})
	gb.Data = &common.BlockData{Data: [][]byte{envBytes}}
	_, _, err = preprocessProtoBlock(nil, allwaysValidKVfunc, gb, false)
	assert.Error(t, err)
	t.Log(err)

//无效筛选器集的错误通道头
	gb = testutil.ConstructTestBlock(t, 14, 1, 1)
	payloadBytes, _ = putils.GetBytesPayload(&common.Payload{
		Header: &common.Header{ChannelHeader: []byte{123}},
	})
	envBytes, _ = putils.GetBytesEnvelope(&common.Envelope{Payload: payloadBytes})
	gb.Data = &common.BlockData{Data: [][]byte{envBytes}}
	flags := lutils.NewTxValidationFlags(len(gb.Data.Data))
	flags.SetFlag(0, peer.TxValidationCode_BAD_CHANNEL_HEADER)
	gb.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = flags
	_, _, err = preprocessProtoBlock(nil, allwaysValidKVfunc, gb, false)
assert.NoError(t, err) //无效的筛选器应采用高级

//新区块
	var blockNum uint64 = 15
	txid := "testtxid1234"
	gb = testutil.ConstructBlockWithTxid(t, blockNum, []byte{123},
		[][]byte{{123}}, []string{txid}, false)
	flags = lutils.NewTxValidationFlags(len(gb.Data.Data))
	flags.SetFlag(0, peer.TxValidationCode_BAD_HEADER_EXTENSION)
	gb.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = flags

//测试记录器
	oldLogger := logger
	defer func() { logger = oldLogger }()
	l, recorder := floggingtest.NewTestLogger(t)
	logger = l

	_, _, err = preprocessProtoBlock(nil, allwaysValidKVfunc, gb, false)
	assert.NoError(t, err)
	expected := fmt.Sprintf(
		"Channel [%s]: Block [%d] Transaction index [%d] TxId [%s] marked as invalid by committer. Reason code [%s]",
		util.GetTestChainID(), blockNum, 0, txid, peer.TxValidationCode_BAD_HEADER_EXTENSION,
	)
	assert.NotEmpty(t, recorder.MessagesContaining(expected))
}

func TestPreprocessProtoBlockInvalidWriteset(t *testing.T) {
	kvValidationFunc := func(key string, value []byte) error {
		if value[0] == '_' {
			return fmt.Errorf("value [%s] found to be invalid by 'kvValidationFunc for testing'", value)
		}
		return nil
	}

	rwSetBuilder := rwsetutil.NewRWSetBuilder()
rwSetBuilder.AddToWriteSet("ns", "key", []byte("_invalidValue")) //坏值
	simulation1, err := rwSetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	simulation1Bytes, err := simulation1.GetPubSimulationBytes()
	assert.NoError(t, err)

	rwSetBuilder = rwsetutil.NewRWSetBuilder()
rwSetBuilder.AddToWriteSet("ns", "key", []byte("validValue")) //良好价值
	simulation2, err := rwSetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	simulation2Bytes, err := simulation2.GetPubSimulationBytes()
	assert.NoError(t, err)

	block := testutil.ConstructBlock(t, 1, testutil.ConstructRandomBytes(t, 32),
[][]byte{simulation1Bytes, simulation2Bytes}, false) //带两个TXS的块
	txfilter := lutils.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	assert.True(t, txfilter.IsValid(0))
assert.True(t, txfilter.IsValid(1)) //两种TXS最初在切割块时都有效。

	internalBlock, _, err := preprocessProtoBlock(nil, kvValidationFunc, block, false)
	assert.NoError(t, err)
assert.False(t, txfilter.IsValid(0)) //索引0处的Tx应标记为无效
assert.True(t, txfilter.IsValid(1))  //索引1处的Tx应标记为有效
	assert.Len(t, internalBlock.Txs, 1)
	assert.Equal(t, internalBlock.Txs[0].IndexInBlock, 1)
}

func TestIncrementPvtdataVersionIfNeeded(t *testing.T) {
	testDBEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	testDB := testDBEnv.GetDBHandle("testdb")
	updateBatch := privacyenabledstate.NewUpdateBatch()
//用一些pvt数据填充db
	updateBatch.PvtUpdates.Put("ns", "coll1", "key1", []byte("value1"), version.NewHeight(1, 1))
	updateBatch.PvtUpdates.Put("ns", "coll2", "key2", []byte("value2"), version.NewHeight(1, 2))
	updateBatch.PvtUpdates.Put("ns", "coll3", "key3", []byte("value3"), version.NewHeight(1, 3))
	updateBatch.PvtUpdates.Put("ns", "col4", "key4", []byte("value4"), version.NewHeight(1, 4))
	testDB.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(1, 4))

//for the current block, mimic the resultant hashed updates
	hashUpdates := privacyenabledstate.NewHashedUpdateBatch()
	hashUpdates.PutValHashAndMetadata("ns", "coll1", lutils.ComputeStringHash("key1"),
lutils.ComputeStringHash("value1_set_by_tx1"), []byte("metadata1_set_by_tx2"), version.NewHeight(2, 2)) //模拟Tx1设置的情况值和Tx2设置的元数据
	hashUpdates.PutValHashAndMetadata("ns", "coll2", lutils.ComputeStringHash("key2"),
lutils.ComputeStringHash("value2"), []byte("metadata2_set_by_tx4"), version.NewHeight(2, 4)) //仅限tx4设置的元数据
	hashUpdates.PutValHashAndMetadata("ns", "coll3", lutils.ComputeStringHash("key3"),
lutils.ComputeStringHash("value3_set_by_tx6"), []byte("metadata3"), version.NewHeight(2, 6)) //仅限tx6设置的值
	pubAndHashedUpdatesBatch := &internal.PubAndHashUpdates{HashUpdates: hashUpdates}

//对于当前块，模拟生成的pvt更新（不考虑元数据）。假设tx6 pvt数据丢失
	pvtUpdateBatch := privacyenabledstate.NewPvtUpdateBatch()
	pvtUpdateBatch.Put("ns", "coll1", "key1", []byte("value1_set_by_tx1"), version.NewHeight(2, 1))
	pvtUpdateBatch.Put("ns", "coll3", "key3", []byte("value3_set_by_tx5"), version.NewHeight(2, 5))
//更新了key1和key3的元数据
	metadataUpdates := metadataUpdates{collKey{"ns", "coll1", "key1"}: true, collKey{"ns", "coll2", "key2"}: true}

//调用函数和测试结果
	err := incrementPvtdataVersionIfNeeded(metadataUpdates, pvtUpdateBatch, pubAndHashedUpdatesBatch, testDB)
	assert.NoError(t, err)

	assert.Equal(t,
&statedb.VersionedValue{Value: []byte("value1_set_by_tx1"), Version: version.NewHeight(2, 2)}, //key1值应相同，版本应升级到（2,2）
		pvtUpdateBatch.Get("ns", "coll1", "key1"),
	)

	assert.Equal(t,
&statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(2, 4)}, //key2条目应该在db和version（2,4）中添加值
		pvtUpdateBatch.Get("ns", "coll2", "key2"),
	)

	assert.Equal(t,
&statedb.VersionedValue{Value: []byte("value3_set_by_tx5"), Version: version.NewHeight(2, 5)}, //键3应该不受影响，因为tx6在pvt数据中丢失。
		pvtUpdateBatch.Get("ns", "coll3", "key3"),
	)
}

func TestTxStatsInfoWithConfigTx(t *testing.T) {
	testDBEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	testDB := testDBEnv.GetDBHandle("emptydb")
	v := NewStatebasedValidator(nil, testDB)

	gb := testutil.ConstructTestBlocks(t, 1)[0]
	_, txStatsInfo, err := v.ValidateAndPrepareBatch(&ledger.BlockAndPvtData{Block: gb}, true)
	assert.NoError(t, err)
	expectedTxStatInfo := []*txmgr.TxStatInfo{
		{
			TxType:         common.HeaderType_CONFIG,
			ValidationCode: peer.TxValidationCode_VALID,
		},
	}
	t.Logf("txStatsInfo=%s\n", spew.Sdump(txStatsInfo))
	assert.Equal(t, expectedTxStatInfo, txStatsInfo)
}

func TestTxStatsInfo(t *testing.T) {
	testDBEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	testDB := testDBEnv.GetDBHandle("emptydb")
	v := NewStatebasedValidator(nil, testDB)

//用4个背书人交易创建一个区块
	tx1SimulationResults, _ := testutilGenerateTxSimulationResultsAsBytes(t,
		&testRwset{
			writes: []*testKeyWrite{
				{ns: "ns1", key: "key1", val: "val1"},
			},
		},
	)
	tx2SimulationResults, _ := testutilGenerateTxSimulationResultsAsBytes(t,
		&testRwset{
			reads: []*testKeyRead{
{ns: "ns1", key: "key1", version: nil}, //应导致MVCC读取与TX1冲突
			},
		},
	)
	tx3SimulationResults, _ := testutilGenerateTxSimulationResultsAsBytes(t,
		&testRwset{
			writes: []*testKeyWrite{
				{ns: "ns1", key: "key2", val: "val2"},
			},
		},
	)
	tx4SimulationResults, _ := testutilGenerateTxSimulationResultsAsBytes(t,
		&testRwset{
			writes: []*testKeyWrite{
				{ns: "ns1", coll: "coll1", key: "key1", val: "val1"},
				{ns: "ns1", coll: "coll2", key: "key1", val: "val1"},
			},
		},
	)

	blockDetails := &testutil.BlockDetails{
		BlockNum:     5,
		PreviousHash: []byte("previousHash"),
		Txs: []*testutil.TxDetails{
			{
				TxID:              "tx_1",
				ChaincodeName:     "cc_1",
				ChaincodeVersion:  "cc_1_v1",
				SimulationResults: tx1SimulationResults,
			},
			{
				TxID:              "tx_2",
				ChaincodeName:     "cc_2",
				ChaincodeVersion:  "cc_2_v1",
				SimulationResults: tx2SimulationResults,
			},
			{
				TxID:              "tx_3",
				ChaincodeName:     "cc_3",
				ChaincodeVersion:  "cc_3_v1",
				SimulationResults: tx3SimulationResults,
			},
			{
				TxID:              "tx_4",
				ChaincodeName:     "cc_4",
				ChaincodeVersion:  "cc_4_v1",
				SimulationResults: tx4SimulationResults,
			},
		},
	}

	block := testutil.ConstructBlockFromBlockDetails(t, blockDetails, false)
	txsFilter := lutils.NewTxValidationFlags(4)
	txsFilter.SetFlag(0, peer.TxValidationCode_VALID)
	txsFilter.SetFlag(1, peer.TxValidationCode_VALID)
	txsFilter.SetFlag(2, peer.TxValidationCode_BAD_PAYLOAD)
	txsFilter.SetFlag(3, peer.TxValidationCode_VALID)
	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter

//收集块的验证状态并对照预期状态进行检查
	_, txStatsInfo, err := v.ValidateAndPrepareBatch(&ledger.BlockAndPvtData{Block: block}, true)
	assert.NoError(t, err)
	expectedTxStatInfo := []*txmgr.TxStatInfo{
		{
			TxType:         common.HeaderType_ENDORSER_TRANSACTION,
			ValidationCode: peer.TxValidationCode_VALID,
			ChaincodeID:    &peer.ChaincodeID{Name: "cc_1", Version: "cc_1_v1"},
		},
		{
			TxType:         common.HeaderType_ENDORSER_TRANSACTION,
			ValidationCode: peer.TxValidationCode_MVCC_READ_CONFLICT,
			ChaincodeID:    &peer.ChaincodeID{Name: "cc_2", Version: "cc_2_v1"},
		},
		{
			TxType:         -1,
			ValidationCode: peer.TxValidationCode_BAD_PAYLOAD,
		},
		{
			TxType:         common.HeaderType_ENDORSER_TRANSACTION,
			ValidationCode: peer.TxValidationCode_VALID,
			ChaincodeID:    &peer.ChaincodeID{Name: "cc_4", Version: "cc_4_v1"},
			NumCollections: 2,
		},
	}
	t.Logf("txStatsInfo=%s\n", spew.Sdump(txStatsInfo))
	assert.Equal(t, expectedTxStatInfo, txStatsInfo)
}

//开始记录内存\test.go
func memoryRecordN(b *logging.MemoryBackend, n int) *logging.Record {
	node := b.Head()
	for i := 0; i < n; i++ {
		if node == nil {
			break
		}
		node = node.Next()
	}
	if node == nil {
		return nil
	}
	return node.Record
}

func testutilSampleTxSimulationResults(t *testing.T, key string) *ledger.TxSimulationResults {
	rwSetBuilder := rwsetutil.NewRWSetBuilder()
//公共RWS NS1+NS2
	rwSetBuilder.AddToReadSet("ns1", key, version.NewHeight(1, 1))
	rwSetBuilder.AddToReadSet("ns2", key, version.NewHeight(1, 1))
	rwSetBuilder.AddToWriteSet("ns2", key, []byte("ns2-key1-value"))

//PVT RWSET NS1
	rwSetBuilder.AddToHashedReadSet("ns1", "coll1", key, version.NewHeight(1, 1))
	rwSetBuilder.AddToHashedReadSet("ns1", "coll2", key, version.NewHeight(1, 1))
	rwSetBuilder.AddToPvtAndHashedWriteSet("ns1", "coll2", key, []byte("pvt-ns1-coll2-key1-value"))

//PVT RWSET NS2
	rwSetBuilder.AddToHashedReadSet("ns2", "coll1", key, version.NewHeight(1, 1))
	rwSetBuilder.AddToHashedReadSet("ns2", "coll2", key, version.NewHeight(1, 1))
	rwSetBuilder.AddToPvtAndHashedWriteSet("ns2", "coll2", key, []byte("pvt-ns2-coll2-key1-value"))
	rwSetBuilder.AddToPvtAndHashedWriteSet("ns2", "coll3", key, nil)

	rwSetBuilder.AddToHashedReadSet("ns3", "coll1", key, version.NewHeight(1, 1))

	pubAndPvtSimulationResults, err := rwSetBuilder.GetTxSimulationResults()
	if err != nil {
		t.Fatalf("ConstructSimulationResultsWithPvtData failed while getting simulation results, err %s", err)
	}

	return pubAndPvtSimulationResults
}

type testKeyRead struct {
	ns, coll, key string
	version       *version.Height
}
type testKeyWrite struct {
	ns, coll, key string
	val           string
}
type testRwset struct {
	reads  []*testKeyRead
	writes []*testKeyWrite
}

func testutilGenerateTxSimulationResults(t *testing.T, rwsetInfo *testRwset) *ledger.TxSimulationResults {
	rwSetBuilder := rwsetutil.NewRWSetBuilder()
	for _, r := range rwsetInfo.reads {
		if r.coll == "" {
			rwSetBuilder.AddToReadSet(r.ns, r.key, r.version)
		} else {
			rwSetBuilder.AddToHashedReadSet(r.ns, r.coll, r.key, r.version)
		}
	}

	for _, w := range rwsetInfo.writes {
		if w.coll == "" {
			rwSetBuilder.AddToWriteSet(w.ns, w.key, []byte(w.val))
		} else {
			rwSetBuilder.AddToPvtAndHashedWriteSet(w.ns, w.coll, w.key, []byte(w.val))
		}
	}
	simulationResults, err := rwSetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)
	return simulationResults
}

func testutilGenerateTxSimulationResultsAsBytes(
	t *testing.T, rwsetInfo *testRwset) (
	publicSimulationRes []byte, pvtWS []byte,
) {
	simulationRes := testutilGenerateTxSimulationResults(t, rwsetInfo)
	pub, err := simulationRes.GetPubSimulationBytes()
	assert.NoError(t, err)
	pvt, err := simulationRes.GetPvtSimulationBytes()
	assert.NoError(t, err)
	return pub, pvt
}
