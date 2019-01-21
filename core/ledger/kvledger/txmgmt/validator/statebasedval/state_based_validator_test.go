
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

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


package statebasedval

import (
	"fmt"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/internal"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type keyValue struct {
	namespace  string
	collection string
	key        string
	keyHash    []byte
	value      []byte
	version    *version.Height
}

func TestMain(m *testing.M) {
	flogging.ActivateSpec("statevalidator,statebasedval,statecouchdb=debug")
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/txmgmt/validator/statebasedval")
	os.Exit(m.Run())
}

func TestValidatorBulkLoadingOfCache(t *testing.T) {
	testDBEnv := privacyenabledstate.CouchDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	db := testDBEnv.GetDBHandle("testdb")

	validator := NewValidator(db)

//用初始数据填充数据库
	batch := privacyenabledstate.NewUpdateBatch()

//创建两个公用千伏对
	pubKV1 := keyValue{namespace: "ns1", key: "key1", value: []byte("value1"), version: version.NewHeight(1, 0)}
	pubKV2 := keyValue{namespace: "ns1", key: "key2", value: []byte("value2"), version: version.NewHeight(1, 1)}

//创建两个哈希kv对
	hashedKV1 := keyValue{namespace: "ns2", collection: "col1", key: "hashedPvtKey1",
		keyHash: util.ComputeStringHash("hashedPvtKey1"), value: []byte("value1"),
		version: version.NewHeight(1, 2)}
	hashedKV2 := keyValue{namespace: "ns2", collection: "col2", key: "hashedPvtKey2",
		keyHash: util.ComputeStringHash("hashedPvtKey2"), value: []byte("value2"),
		version: version.NewHeight(1, 3)}

//将公用和哈希的kv对存储到db
	batch.PubUpdates.Put(pubKV1.namespace, pubKV1.key, pubKV1.value, pubKV1.version)
	batch.PubUpdates.Put(pubKV2.namespace, pubKV2.key, pubKV2.value, pubKV2.version)
	batch.HashUpdates.Put(hashedKV1.namespace, hashedKV1.collection, hashedKV1.keyHash, hashedKV1.value, hashedKV1.version)
	batch.HashUpdates.Put(hashedKV2.namespace, hashedKV2.collection, hashedKV2.keyHash, hashedKV2.value, hashedKV2.version)

	db.ApplyPrivacyAwareUpdates(batch, version.NewHeight(1, 4))

//构造事务1的读取集。它包含两个公用千伏对（pubkv1、pubkv2）和两个
//哈希kv对（hashedkv1，hashedkv2）。
	rwsetBuilder1 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder1.AddToReadSet(pubKV1.namespace, pubKV1.key, pubKV1.version)
	rwsetBuilder1.AddToReadSet(pubKV2.namespace, pubKV2.key, pubKV2.version)
	rwsetBuilder1.AddToHashedReadSet(hashedKV1.namespace, hashedKV1.collection, hashedKV1.key, hashedKV1.version)
	rwsetBuilder1.AddToHashedReadSet(hashedKV2.namespace, hashedKV2.collection, hashedKV2.key, hashedKV2.version)

//构造事务1的读取集。它包含不在状态db中的kv对。
	rwsetBuilder2 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder2.AddToReadSet("ns3", "key1", nil)
	rwsetBuilder2.AddToHashedReadSet("ns3", "col1", "hashedPvtKey1", nil)

//Construct internal block
	transRWSets := getTestPubSimulationRWSet(t, rwsetBuilder1, rwsetBuilder2)
	var trans []*internal.Transaction
	for i, tranRWSet := range transRWSets {
		tx := &internal.Transaction{
			ID:             fmt.Sprintf("txid-%d", i),
			IndexInBlock:   i,
			ValidationCode: peer.TxValidationCode_VALID,
			RWSet:          tranRWSet,
		}
		trans = append(trans, tx)
	}
	block := &internal.Block{Num: 1, Txs: trans}

	if validator.db.IsBulkOptimizable() {

		commonStorageDB := validator.db.(*privacyenabledstate.CommonStorageDB)
		bulkOptimizable, _ := commonStorageDB.VersionedDB.(statedb.BulkOptimizable)

//清除在ApplyDriveAcyaWareUpdates（）期间加载的缓存
		validator.db.ClearCachedVersions()

		validator.preLoadCommittedVersionOfRSet(block)

//应在缓存中找到pubkv1
		version, keyFound := bulkOptimizable.GetCachedVersion(pubKV1.namespace, pubKV1.key)
		assert.True(t, keyFound)
		assert.Equal(t, pubKV1.version, version)

//应该在缓存中找到PUBKv2
		version, keyFound = bulkOptimizable.GetCachedVersion(pubKV2.namespace, pubKV2.key)
		assert.True(t, keyFound)
		assert.Equal(t, pubKV2.version, version)

//[ns3，key1]应该在缓存中找到，因为它在事务1的readset中，尽管它是
//不在状态数据库中，但版本将为零
		version, keyFound = bulkOptimizable.GetCachedVersion("ns3", "key1")
		assert.True(t, keyFound)
		assert.Nil(t, version)

//不应在缓存中找到[NS4，key1]，因为它未加载
		version, keyFound = bulkOptimizable.GetCachedVersion("ns4", "key1")
		assert.False(t, keyFound)
		assert.Nil(t, version)

//应在缓存中找到hashedkv1
		version, keyFound = validator.db.GetCachedKeyHashVersion(hashedKV1.namespace,
			hashedKV1.collection, hashedKV1.keyHash)
		assert.True(t, keyFound)
		assert.Equal(t, hashedKV1.version, version)

//应在缓存中找到hashedkv2
		version, keyFound = validator.db.GetCachedKeyHashVersion(hashedKV2.namespace,
			hashedKV2.collection, hashedKV2.keyHash)
		assert.True(t, keyFound)
		assert.Equal(t, hashedKV2.version, version)

//应在缓存中找到[ns3，col1，hashedpvtkey1]，就像在事务2的readset中一样，尽管它是
//不在状态数据库中
		version, keyFound = validator.db.GetCachedKeyHashVersion("ns3", "col1", util.ComputeStringHash("hashedPvtKey1"))
		assert.True(t, keyFound)
		assert.Nil(t, version)

//[ns4, col, key1] should not be found in cache as it was not loaded
		version, keyFound = validator.db.GetCachedKeyHashVersion("ns4", "col1", util.ComputeStringHash("key1"))
		assert.False(t, keyFound)
		assert.Nil(t, version)

//清除缓存
		validator.db.ClearCachedVersions()

//当cahce被清空时，不应在缓存中找到pubkv1
		version, keyFound = bulkOptimizable.GetCachedVersion(pubKV1.namespace, pubKV1.key)
		assert.False(t, keyFound)
		assert.Nil(t, version)

//当cahce清空时，不应在缓存中找到[ns3，col1，key1]
		version, keyFound = validator.db.GetCachedKeyHashVersion("ns3", "col1", util.ComputeStringHash("hashedPvtKey1"))
		assert.False(t, keyFound)
		assert.Nil(t, version)
	}
}

func TestValidator(t *testing.T) {
	testDBEnv := privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	db := testDBEnv.GetDBHandle("TestDB")

//用初始数据填充数据库
	batch := privacyenabledstate.NewUpdateBatch()
	batch.PubUpdates.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 0))
	batch.PubUpdates.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 1))
	batch.PubUpdates.Put("ns1", "key3", []byte("value3"), version.NewHeight(1, 2))
	batch.PubUpdates.Put("ns1", "key4", []byte("value4"), version.NewHeight(1, 3))
	batch.PubUpdates.Put("ns1", "key5", []byte("value5"), version.NewHeight(1, 4))
	db.ApplyPrivacyAwareUpdates(batch, version.NewHeight(1, 4))

	validator := NewValidator(db)

//rwset1应该有效
	rwsetBuilder1 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder1.AddToReadSet("ns1", "key1", version.NewHeight(1, 0))
	rwsetBuilder1.AddToReadSet("ns2", "key2", nil)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder1), []int{})

//rwset2不应有效
	rwsetBuilder2 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder2.AddToReadSet("ns1", "key1", version.NewHeight(1, 1))
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder2), []int{0})

//rwset3不应有效
	rwsetBuilder3 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder3.AddToReadSet("ns1", "key1", nil)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder3), []int{0})

//同一块内的rwset4和rwset5-rwset4应有效并使rwset5无效
	rwsetBuilder4 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder4.AddToReadSet("ns1", "key1", version.NewHeight(1, 0))
	rwsetBuilder4.AddToWriteSet("ns1", "key1", []byte("value1_new"))

	rwsetBuilder5 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder5.AddToReadSet("ns1", "key1", version.NewHeight(1, 0))
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder4, rwsetBuilder5), []int{1})
}

func TestPhantomValidation(t *testing.T) {
	testDBEnv := privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	db := testDBEnv.GetDBHandle("TestDB")

//用初始数据填充数据库
	batch := privacyenabledstate.NewUpdateBatch()
	batch.PubUpdates.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 0))
	batch.PubUpdates.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 1))
	batch.PubUpdates.Put("ns1", "key3", []byte("value3"), version.NewHeight(1, 2))
	batch.PubUpdates.Put("ns1", "key4", []byte("value4"), version.NewHeight(1, 3))
	batch.PubUpdates.Put("ns1", "key5", []byte("value5"), version.NewHeight(1, 4))
	db.ApplyPrivacyAwareUpdates(batch, version.NewHeight(1, 4))

	validator := NewValidator(db)

//rwset1应该有效
	rwsetBuilder1 := rwsetutil.NewRWSetBuilder()
	rqi1 := &kvrwset.RangeQueryInfo{StartKey: "key2", EndKey: "key4", ItrExhausted: true}
	rqi1.SetRawReads([]*kvrwset.KVRead{
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key3", version.NewHeight(1, 2))})
	rwsetBuilder1.AddToRangeQuerySet("ns1", rqi1)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder1), []int{})

//rwset2不应有效-密钥4的版本已更改
	rwsetBuilder2 := rwsetutil.NewRWSetBuilder()
	rqi2 := &kvrwset.RangeQueryInfo{StartKey: "key2", EndKey: "key4", ItrExhausted: false}
	rqi2.SetRawReads([]*kvrwset.KVRead{
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key3", version.NewHeight(1, 2)),
		rwsetutil.NewKVRead("key4", version.NewHeight(1, 2))})
	rwsetBuilder2.AddToRangeQuerySet("ns1", rqi2)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder2), []int{0})

//rwset3不应有效-仿真键3已提交到数据库
	rwsetBuilder3 := rwsetutil.NewRWSetBuilder()
	rqi3 := &kvrwset.RangeQueryInfo{StartKey: "key2", EndKey: "key4", ItrExhausted: false}
	rqi3.SetRawReads([]*kvrwset.KVRead{
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key4", version.NewHeight(1, 3))})
	rwsetBuilder3.AddToRangeQuerySet("ns1", rqi3)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder3), []int{0})

////删除rwset4和rwset5中的键应该无效
	rwsetBuilder4 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder4.AddToWriteSet("ns1", "key3", nil)
	rwsetBuilder5 := rwsetutil.NewRWSetBuilder()
	rqi5 := &kvrwset.RangeQueryInfo{StartKey: "key2", EndKey: "key4", ItrExhausted: false}
	rqi5.SetRawReads([]*kvrwset.KVRead{
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key3", version.NewHeight(1, 2)),
		rwsetutil.NewKVRead("key4", version.NewHeight(1, 3))})
	rwsetBuilder5.AddToRangeQuerySet("ns1", rqi5)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder4, rwsetBuilder5), []int{1})

//在rwset6中添加一个键，rwset7应该无效
	rwsetBuilder6 := rwsetutil.NewRWSetBuilder()
	rwsetBuilder6.AddToWriteSet("ns1", "key2_1", []byte("value2_1"))

	rwsetBuilder7 := rwsetutil.NewRWSetBuilder()
	rqi7 := &kvrwset.RangeQueryInfo{StartKey: "key2", EndKey: "key4", ItrExhausted: false}
	rqi7.SetRawReads([]*kvrwset.KVRead{
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key3", version.NewHeight(1, 2)),
		rwsetutil.NewKVRead("key4", version.NewHeight(1, 3))})
	rwsetBuilder7.AddToRangeQuerySet("ns1", rqi7)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder6, rwsetBuilder7), []int{1})
}

func TestPhantomHashBasedValidation(t *testing.T) {
	testDBEnv := privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	defer testDBEnv.Cleanup()
	db := testDBEnv.GetDBHandle("TestDB")

//用初始数据填充数据库
	batch := privacyenabledstate.NewUpdateBatch()
	batch.PubUpdates.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 0))
	batch.PubUpdates.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 1))
	batch.PubUpdates.Put("ns1", "key3", []byte("value3"), version.NewHeight(1, 2))
	batch.PubUpdates.Put("ns1", "key4", []byte("value4"), version.NewHeight(1, 3))
	batch.PubUpdates.Put("ns1", "key5", []byte("value5"), version.NewHeight(1, 4))
	batch.PubUpdates.Put("ns1", "key6", []byte("value6"), version.NewHeight(1, 5))
	batch.PubUpdates.Put("ns1", "key7", []byte("value7"), version.NewHeight(1, 6))
	batch.PubUpdates.Put("ns1", "key8", []byte("value8"), version.NewHeight(1, 7))
	batch.PubUpdates.Put("ns1", "key9", []byte("value9"), version.NewHeight(1, 8))
	db.ApplyPrivacyAwareUpdates(batch, version.NewHeight(1, 8))

	validator := NewValidator(db)

	rwsetBuilder1 := rwsetutil.NewRWSetBuilder()
	rqi1 := &kvrwset.RangeQueryInfo{StartKey: "key2", EndKey: "key9", ItrExhausted: true}
	kvReadsDuringSimulation1 := []*kvrwset.KVRead{
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key3", version.NewHeight(1, 2)),
		rwsetutil.NewKVRead("key4", version.NewHeight(1, 3)),
		rwsetutil.NewKVRead("key5", version.NewHeight(1, 4)),
		rwsetutil.NewKVRead("key6", version.NewHeight(1, 5)),
		rwsetutil.NewKVRead("key7", version.NewHeight(1, 6)),
		rwsetutil.NewKVRead("key8", version.NewHeight(1, 7)),
	}
	rqi1.SetMerkelSummary(buildTestHashResults(t, 2, kvReadsDuringSimulation1))
	rwsetBuilder1.AddToRangeQuerySet("ns1", rqi1)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder1), []int{})

	rwsetBuilder2 := rwsetutil.NewRWSetBuilder()
	rqi2 := &kvrwset.RangeQueryInfo{StartKey: "key1", EndKey: "key9", ItrExhausted: false}
	kvReadsDuringSimulation2 := []*kvrwset.KVRead{
		rwsetutil.NewKVRead("key1", version.NewHeight(1, 0)),
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key3", version.NewHeight(1, 1)),
		rwsetutil.NewKVRead("key4", version.NewHeight(1, 3)),
		rwsetutil.NewKVRead("key5", version.NewHeight(1, 4)),
		rwsetutil.NewKVRead("key6", version.NewHeight(1, 5)),
		rwsetutil.NewKVRead("key7", version.NewHeight(1, 6)),
		rwsetutil.NewKVRead("key8", version.NewHeight(1, 7)),
		rwsetutil.NewKVRead("key9", version.NewHeight(1, 8)),
	}
	rqi2.SetMerkelSummary(buildTestHashResults(t, 2, kvReadsDuringSimulation2))
	rwsetBuilder2.AddToRangeQuerySet("ns1", rqi2)
	checkValidation(t, validator, getTestPubSimulationRWSet(t, rwsetBuilder2), []int{0})
}

func checkValidation(t *testing.T, val *Validator, transRWSets []*rwsetutil.TxRwSet, expectedInvalidTxIndexes []int) {
	var trans []*internal.Transaction
	for i, tranRWSet := range transRWSets {
		tx := &internal.Transaction{
			ID:             fmt.Sprintf("txid-%d", i),
			IndexInBlock:   i,
			ValidationCode: peer.TxValidationCode_VALID,
			RWSet:          tranRWSet,
		}
		trans = append(trans, tx)
	}
	block := &internal.Block{Num: 1, Txs: trans}
	_, err := val.ValidateAndPrepareBatch(block, true)
	assert.NoError(t, err)
	t.Logf("block.Txs[0].ValidationCode = %d", block.Txs[0].ValidationCode)
	var invalidTxs []int
	for _, tx := range block.Txs {
		if tx.ValidationCode != peer.TxValidationCode_VALID {
			invalidTxs = append(invalidTxs, tx.IndexInBlock)
		}
	}
	assert.Equal(t, len(expectedInvalidTxIndexes), len(invalidTxs))
	assert.ElementsMatch(t, invalidTxs, expectedInvalidTxIndexes)
}

func buildTestHashResults(t *testing.T, maxDegree int, kvReads []*kvrwset.KVRead) *kvrwset.QueryReadsMerkleSummary {
	if len(kvReads) <= maxDegree {
		t.Fatal("This method should be called with number of KVReads more than maxDegree; Else, hashing won't be performedrwset")
	}
	helper, _ := rwsetutil.NewRangeQueryResultsHelper(true, uint32(maxDegree))
	for _, kvRead := range kvReads {
		helper.AddResult(kvRead)
	}
	_, h, err := helper.Done()
	assert.NoError(t, err)
	assert.NotNil(t, h)
	return h
}

func getTestPubSimulationRWSet(t *testing.T, builders ...*rwsetutil.RWSetBuilder) []*rwsetutil.TxRwSet {
	var pubRWSets []*rwsetutil.TxRwSet
	for _, b := range builders {
		s, e := b.GetTxSimulationResults()
		assert.NoError(t, e)
		sBytes, err := s.GetPubSimulationBytes()
		assert.NoError(t, err)
		pubRWSet := &rwsetutil.TxRwSet{}
		assert.NoError(t, pubRWSet.FromProtoBytes(sBytes))
		pubRWSets = append(pubRWSets, pubRWSet)
	}
	return pubRWSets
}
