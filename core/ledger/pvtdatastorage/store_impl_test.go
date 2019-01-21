
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


package pvtdatastorage

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("pvtdatastorage=debug")
	viper.Set("peer.fileSystemPath", "/tmp/fabric/core/ledger/pvtdatastorage")
	os.Exit(m.Run())
}

func TestEmptyStore(t *testing.T) {
	env := NewTestStoreEnv(t, "TestEmptyStore", nil)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore
	testEmpty(true, assert, store)
	testPendingBatch(false, assert, store)
}

func TestStoreBasicCommitAndRetrieval(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
			{"ns-2", "coll-1"}: 0,
			{"ns-2", "coll-2"}: 0,
			{"ns-3", "coll-1"}: 0,
			{"ns-4", "coll-1"}: 0,
			{"ns-4", "coll-2"}: 0,
		},
	)

	env := NewTestStoreEnv(t, "TestStoreBasicCommitAndRetrieval", btlPolicy)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore
	testData := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}

//construct missing data for block 1
	blk1MissingData := make(ledger.TxMissingPvtDataMap)

//TX1中符合条件的丢失数据
	blk1MissingData.Add(1, "ns-1", "coll-1", true)
	blk1MissingData.Add(1, "ns-1", "coll-2", true)
	blk1MissingData.Add(1, "ns-2", "coll-1", true)
	blk1MissingData.Add(1, "ns-2", "coll-2", true)
//TX2中符合条件的丢失数据
	blk1MissingData.Add(2, "ns-3", "coll-1", true)
//tx4中缺少数据不合格
	blk1MissingData.Add(4, "ns-4", "coll-1", false)
	blk1MissingData.Add(4, "ns-4", "coll-2", false)

//构造块2的缺失数据
	blk2MissingData := make(ledger.TxMissingPvtDataMap)
//TX1中符合条件的丢失数据
	blk2MissingData.Add(1, "ns-1", "coll-1", true)
	blk2MissingData.Add(1, "ns-1", "coll-2", true)
//TX3中符合条件的丢失数据
	blk2MissingData.Add(3, "ns-1", "coll-1", true)

//没有块0的pvt数据
	assert.NoError(store.Prepare(0, nil, nil))
	assert.NoError(store.Commit())

//带块1的pvt数据-提交
	assert.NoError(store.Prepare(1, testData, blk1MissingData))
	assert.NoError(store.Commit())

//带块2的pvt数据-回滚
	assert.NoError(store.Prepare(2, testData, nil))
	assert.NoError(store.Rollback())

//块0的PVT数据检索应该返回零
	var nilFilter ledger.PvtNsCollFilter
	retrievedData, err := store.GetPvtDataByBlockNum(0, nilFilter)
	assert.NoError(err)
	assert.Nil(retrievedData)

//块1的pvt数据检索应返回完整的pvt data
	retrievedData, err = store.GetPvtDataByBlockNum(1, nilFilter)
	assert.NoError(err)
	for i, data := range retrievedData {
		assert.Equal(data.SeqInBlock, testData[i].SeqInBlock)
		assert.True(proto.Equal(data.WriteSet, testData[i].WriteSet))
	}

//带过滤器的块1的pvt数据检索应返回过滤后的pvt data
	filter := ledger.NewPvtNsCollFilter()
	filter.Add("ns-1", "coll-1")
	filter.Add("ns-2", "coll-2")
	retrievedData, err = store.GetPvtDataByBlockNum(1, filter)
	expectedRetrievedData := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-2:coll-2"}),
	}
	for i, data := range retrievedData {
		assert.Equal(data.SeqInBlock, expectedRetrievedData[i].SeqInBlock)
		assert.True(proto.Equal(data.WriteSet, expectedRetrievedData[i].WriteSet))
	}

//块2的pvt数据检索应返回erAutoFrange
	retrievedData, err = store.GetPvtDataByBlockNum(2, nilFilter)
	_, ok := err.(*ErrOutOfRange)
	assert.True(ok)
	assert.Nil(retrievedData)

//带块2的pvt数据-提交
	assert.NoError(store.Prepare(2, testData, blk2MissingData))
	assert.NoError(store.Commit())

//使用GetMissingPvtDataInfoFormsToRecentBlocks检索存储的缺少的条目
//只有符合条件的条目的代码路径才会包含在这个单元测试中。为了
//不合格条目，代码路径将包含在FAB-11437中。

	expectedMissingPvtDataInfo := make(ledger.MissingPvtDataInfo)
//块2中缺少数据，tx1
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-2")
	expectedMissingPvtDataInfo.Add(2, 3, "ns-1", "coll-1")

	missingPvtDataInfo, err := store.GetMissingPvtDataInfoForMostRecentBlocks(1)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

//块1、tx1中缺少数据
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-2")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-2", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-2", "coll-2")

//块1、tx2中缺少数据
	expectedMissingPvtDataInfo.Add(1, 2, "ns-3", "coll-1")

	missingPvtDataInfo, err = store.GetMissingPvtDataInfoForMostRecentBlocks(2)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

	missingPvtDataInfo, err = store.GetMissingPvtDataInfoForMostRecentBlocks(10)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)
}

func TestCommitPvtDataOfOldBlocks(t *testing.T) {
	viper.Set("ledger.pvtdataStore.purgeInterval", 2)
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 3,
			{"ns-1", "coll-2"}: 1,
			{"ns-2", "coll-1"}: 0,
			{"ns-2", "coll-2"}: 1,
			{"ns-3", "coll-1"}: 0,
			{"ns-3", "coll-2"}: 3,
			{"ns-4", "coll-1"}: 0,
			{"ns-4", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, "TestCommitPvtDataOfOldBlocks", btlPolicy)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore

	testData := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}

//构造块1的缺失数据
	blk1MissingData := make(ledger.TxMissingPvtDataMap)

//TX1中符合条件的丢失数据
	blk1MissingData.Add(1, "ns-1", "coll-1", true)
	blk1MissingData.Add(1, "ns-1", "coll-2", true)
	blk1MissingData.Add(1, "ns-2", "coll-1", true)
	blk1MissingData.Add(1, "ns-2", "coll-2", true)
//TX2中符合条件的丢失数据
	blk1MissingData.Add(2, "ns-1", "coll-1", true)
	blk1MissingData.Add(2, "ns-1", "coll-2", true)
	blk1MissingData.Add(2, "ns-3", "coll-1", true)
	blk1MissingData.Add(2, "ns-3", "coll-2", true)

//构造块2的缺失数据
	blk2MissingData := make(ledger.TxMissingPvtDataMap)
//TX1中符合条件的丢失数据
	blk2MissingData.Add(1, "ns-1", "coll-1", true)
	blk2MissingData.Add(1, "ns-1", "coll-2", true)
//TX3中符合条件的丢失数据
	blk2MissingData.Add(3, "ns-1", "coll-1", true)

//提交没有数据的块0
	assert.NoError(store.Prepare(0, nil, nil))
	assert.NoError(store.Commit())

//使用pvtdata和missingdata提交块1
	assert.NoError(store.Prepare(1, testData, blk1MissingData))
	assert.NoError(store.Commit())

//使用pvtdata和missingdata提交块2
	assert.NoError(store.Prepare(2, nil, blk2MissingData))
	assert.NoError(store.Commit())

//检查丢失的数据条目是否正确存储
	expectedMissingPvtDataInfo := make(ledger.MissingPvtDataInfo)
//块1、tx1中缺少数据
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-2")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-2", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-2", "coll-2")

//块1、tx2中缺少数据
	expectedMissingPvtDataInfo.Add(1, 2, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 2, "ns-1", "coll-2")
	expectedMissingPvtDataInfo.Add(1, 2, "ns-3", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 2, "ns-3", "coll-2")

//块2中缺少数据，tx1
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-2")
//块2中缺少数据，tx3
	expectedMissingPvtDataInfo.Add(2, 3, "ns-1", "coll-1")

	missingPvtDataInfo, err := store.GetMissingPvtDataInfoForMostRecentBlocks(2)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

//COMMIT THE MISSINGDATA IN BLOCK 1 AND BLOCK 2
	oldBlocksPvtData := make(map[uint64][]*ledger.TxPvtData)
	oldBlocksPvtData[1] = []*ledger.TxPvtData{
		produceSamplePvtdata(t, 1, []string{"ns-1:coll-1", "ns-2:coll-1"}),
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-3:coll-1"}),
	}
	oldBlocksPvtData[2] = []*ledger.TxPvtData{
		produceSamplePvtdata(t, 3, []string{"ns-1:coll-1"}),
	}

	err = store.CommitPvtDataOfOldBlocks(oldBlocksPvtData)
	assert.NoError(err)

//确保存储区中存在先前丢失的块1和块2的pvtdata
	ns1Coll1Blk1Tx1 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-1", blkNum: 1}, txNum: 1}
	ns2Coll1Blk1Tx1 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-2", coll: "coll-1", blkNum: 1}, txNum: 1}
	ns1Coll1Blk1Tx2 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-1", blkNum: 1}, txNum: 2}
	ns3Coll1Blk1Tx2 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-3", coll: "coll-1", blkNum: 1}, txNum: 2}
	ns1Coll1Blk2Tx3 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-1", blkNum: 2}, txNum: 3}

	assert.True(testDataKeyExists(t, store, ns1Coll1Blk1Tx1))
	assert.True(testDataKeyExists(t, store, ns2Coll1Blk1Tx1))
	assert.True(testDataKeyExists(t, store, ns1Coll1Blk1Tx2))
	assert.True(testDataKeyExists(t, store, ns3Coll1Blk1Tx2))
	assert.True(testDataKeyExists(t, store, ns1Coll1Blk2Tx3))

//块2的pvt数据检索应返回刚提交的pvt data
	var nilFilter ledger.PvtNsCollFilter
	retrievedData, err := store.GetPvtDataByBlockNum(2, nilFilter)
	assert.NoError(err)
	for i, data := range retrievedData {
		assert.Equal(data.SeqInBlock, oldBlocksPvtData[2][i].SeqInBlock)
		assert.True(proto.Equal(data.WriteSet, oldBlocksPvtData[2][i].WriteSet))
	}

	expectedMissingPvtDataInfo = make(ledger.MissingPvtDataInfo)
//块1、tx1中缺少数据
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-2")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-2", "coll-2")

//块1、tx2中缺少数据
	expectedMissingPvtDataInfo.Add(1, 2, "ns-1", "coll-2")
	expectedMissingPvtDataInfo.Add(1, 2, "ns-3", "coll-2")

//块2中缺少数据，tx1
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-2")

	missingPvtDataInfo, err = store.GetMissingPvtDataInfoForMostRecentBlocks(2)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

//blkspvdtata返回已为其提交任何pvdtata的块的所有pvt数据。
//使用commitpvtdataofoldblocks
	blksPvtData, err := store.GetLastUpdatedOldBlocksPvtData()
	assert.NoError(err)

	expectedLastupdatedPvtdata := make(map[uint64][]*ledger.TxPvtData)
	expectedLastupdatedPvtdata[1] = []*ledger.TxPvtData{
		produceSamplePvtdata(t, 1, []string{"ns-1:coll-1", "ns-2:coll-1"}),
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-2:coll-1", "ns-2:coll-2", "ns-3:coll-1"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}
	expectedLastupdatedPvtdata[2] = []*ledger.TxPvtData{
		produceSamplePvtdata(t, 3, []string{"ns-1:coll-1"}),
	}
	assert.Equal(expectedLastupdatedPvtdata, blksPvtData)

	err = store.ResetLastUpdatedOldBlocksList()
	assert.NoError(err)

	blksPvtData, err = store.GetLastUpdatedOldBlocksPvtData()
	assert.NoError(err)
	assert.Nil(blksPvtData)

//提交块3，不带pvtdata
	assert.NoError(store.Prepare(3, nil, nil))
	assert.NoError(store.Commit())

//在块1中，ns-1:coll-2和ns-2:coll-2应该过期，但不清除
//因此，下面的提交应该在存储中创建条目
	oldBlocksPvtData = make(map[uint64][]*ledger.TxPvtData)
	oldBlocksPvtData[1] = []*ledger.TxPvtData{
produceSamplePvtdata(t, 1, []string{"ns-1:coll-2"}), //虽然过期了，
//将提交到存储，因为它尚未清除
produceSamplePvtdata(t, 2, []string{"ns-3:coll-2"}), //永不过期
	}

	err = store.CommitPvtDataOfOldBlocks(oldBlocksPvtData)
	assert.NoError(err)

	ns1Coll2Blk1Tx1 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-2", blkNum: 1}, txNum: 1}
	ns2Coll2Blk1Tx1 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-2", coll: "coll-2", blkNum: 1}, txNum: 1}
	ns1Coll2Blk1Tx2 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-2", blkNum: 1}, txNum: 2}
	ns3Coll2Blk1Tx2 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-3", coll: "coll-2", blkNum: 1}, txNum: 2}

//虽然pvtdata已过期但尚未清除，但我们确实
//提交数据，因此条目将存在于
//商店
assert.True(testDataKeyExists(t, store, ns1Coll2Blk1Tx1))  //已过期但已提交
assert.False(testDataKeyExists(t, store, ns2Coll2Blk1Tx1)) //已过期但仍然丢失
assert.False(testDataKeyExists(t, store, ns1Coll2Blk1Tx2)) //过期仍然丢失
assert.True(testDataKeyExists(t, store, ns3Coll2Blk1Tx2))  //永不过期

	err = store.ResetLastUpdatedOldBlocksList()
	assert.NoError(err)

//提交块4，不带pvtdata
	assert.NoError(store.Prepare(4, nil, nil))
	assert.NoError(store.Commit())

	testWaitForPurgerRoutineToFinish(store)

//在块1中，ns-1:coll-2和ns-2:coll-2应该过期，但不清除
//因此，以下提交不应在存储区中创建条目
	oldBlocksPvtData = make(map[uint64][]*ledger.TxPvtData)
	oldBlocksPvtData[1] = []*ledger.TxPvtData{
//这两个数据都已过期并清除。因此，它不会
//已提交到存储区
		produceSamplePvtdata(t, 1, []string{"ns-2:coll-2"}),
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-2"}),
	}

	err = store.CommitPvtDataOfOldBlocks(oldBlocksPvtData)
	assert.NoError(err)

	ns1Coll2Blk1Tx1 = &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-2", blkNum: 1}, txNum: 1}
	ns2Coll2Blk1Tx1 = &dataKey{nsCollBlk: nsCollBlk{ns: "ns-2", coll: "coll-2", blkNum: 1}, txNum: 1}
	ns1Coll2Blk1Tx2 = &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-2", blkNum: 1}, txNum: 2}
	ns3Coll2Blk1Tx2 = &dataKey{nsCollBlk: nsCollBlk{ns: "ns-3", coll: "coll-2", blkNum: 1}, txNum: 2}

assert.False(testDataKeyExists(t, store, ns1Coll2Blk1Tx1)) //清除的
assert.False(testDataKeyExists(t, store, ns2Coll2Blk1Tx1)) //清除的
assert.False(testDataKeyExists(t, store, ns1Coll2Blk1Tx2)) //清除的
assert.True(testDataKeyExists(t, store, ns3Coll2Blk1Tx2))  //永不过期
}

func TestExpiryDataNotIncluded(t *testing.T) {
	ledgerid := "TestExpiryDataNotIncluded"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 1,
			{"ns-1", "coll-2"}: 0,
			{"ns-2", "coll-1"}: 0,
			{"ns-2", "coll-2"}: 2,
			{"ns-3", "coll-1"}: 1,
			{"ns-3", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, ledgerid, btlPolicy)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore

//构造块1的缺失数据
	blk1MissingData := make(ledger.TxMissingPvtDataMap)
//TX1中符合条件的丢失数据
	blk1MissingData.Add(1, "ns-1", "coll-1", true)
	blk1MissingData.Add(1, "ns-1", "coll-2", true)
//tx4中缺少数据不合格
	blk1MissingData.Add(4, "ns-3", "coll-1", false)
	blk1MissingData.Add(4, "ns-3", "coll-2", false)

//构造块2的缺失数据
	blk2MissingData := make(ledger.TxMissingPvtDataMap)
//TX1中符合条件的丢失数据
	blk2MissingData.Add(1, "ns-1", "coll-1", true)
	blk2MissingData.Add(1, "ns-1", "coll-2", true)

//没有块0的pvt数据
	assert.NoError(store.Prepare(0, nil, nil))
	assert.NoError(store.Commit())

//为块1写入pvt数据
	testDataForBlk1 := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}
	assert.NoError(store.Prepare(1, testDataForBlk1, blk1MissingData))
	assert.NoError(store.Commit())

//为块2写入pvt数据
	testDataForBlk2 := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 3, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 5, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}
	assert.NoError(store.Prepare(2, testDataForBlk2, blk2MissingData))
	assert.NoError(store.Commit())

	retrievedData, _ := store.GetPvtDataByBlockNum(1, nil)
//块1数据仍不应过期
	for i, data := range retrievedData {
		assert.Equal(data.SeqInBlock, testDataForBlk1[i].SeqInBlock)
		assert.True(proto.Equal(data.WriteSet, testDataForBlk1[i].WriteSet))
	}

//丢失的数据项都不会过期
	expectedMissingPvtDataInfo := make(ledger.MissingPvtDataInfo)
//块2中缺少数据，tx1
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-2")

//块1、tx1中缺少数据
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-2")

	missingPvtDataInfo, err := store.GetMissingPvtDataInfoForMostRecentBlocks(10)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

//Commit block 3 with no pvtdata
	assert.NoError(store.Prepare(3, nil, nil))
	assert.NoError(store.Commit())

//提交块3后，块1的“ns-1:coll1”的数据应该已经过期，不应该由存储返回。
	expectedPvtdataFromBlock1 := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}
	retrievedData, _ = store.GetPvtDataByBlockNum(1, nil)
	assert.Equal(expectedPvtdataFromBlock1, retrievedData)

//提交block 3后，block1-tx1中“ns1-coll1”的缺失数据应该已经过期
	expectedMissingPvtDataInfo = make(ledger.MissingPvtDataInfo)
//块2中缺少数据，tx1
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-2")
//块1、tx1中缺少数据
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-2")

	missingPvtDataInfo, err = store.GetMissingPvtDataInfoForMostRecentBlocks(10)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

//提交块4，不带pvtdata
	assert.NoError(store.Prepare(4, nil, nil))
	assert.NoError(store.Commit())

//提交块4后，块1的“ns-2:coll2”的数据也应过期，不应由存储返回。
	expectedPvtdataFromBlock1 = []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-2", "ns-2:coll-1"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-2", "ns-2:coll-1"}),
	}
	retrievedData, _ = store.GetPvtDataByBlockNum(1, nil)
	assert.Equal(expectedPvtdataFromBlock1, retrievedData)

//现在，对于块2，“ns-1:coll1”也应该已经过期
	expectedPvtdataFromBlock2 := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 3, []string{"ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 5, []string{"ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}
	retrievedData, _ = store.GetPvtDataByBlockNum(2, nil)
	assert.Equal(expectedPvtdataFromBlock2, retrievedData)

//提交块4后，块2-tx1中“ns1-coll1”的缺失数据应该已经过期
	expectedMissingPvtDataInfo = make(ledger.MissingPvtDataInfo)
//块2中缺少数据，tx1
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-2")

//块1、tx1中缺少数据
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-2")

	missingPvtDataInfo, err = store.GetMissingPvtDataInfoForMostRecentBlocks(10)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

}

func TestStorePurge(t *testing.T) {
	ledgerid := "TestStorePurge"
	viper.Set("ledger.pvtdataStore.purgeInterval", 2)
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 1,
			{"ns-1", "coll-2"}: 0,
			{"ns-2", "coll-1"}: 0,
			{"ns-2", "coll-2"}: 4,
			{"ns-3", "coll-1"}: 1,
			{"ns-3", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, ledgerid, btlPolicy)
	defer env.Cleanup()
	assert := assert.New(t)
	s := env.TestStore

//没有块0的pvt数据
	assert.NoError(s.Prepare(0, nil, nil))
	assert.NoError(s.Commit())

//构造块1的缺失数据
	blk1MissingData := make(ledger.TxMissingPvtDataMap)
//TX1中符合条件的丢失数据
	blk1MissingData.Add(1, "ns-1", "coll-1", true)
	blk1MissingData.Add(1, "ns-1", "coll-2", true)
//tx4中缺少数据不合格
	blk1MissingData.Add(4, "ns-3", "coll-1", false)
	blk1MissingData.Add(4, "ns-3", "coll-2", false)

//为块1写入pvt数据
	testDataForBlk1 := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}
	assert.NoError(s.Prepare(1, testDataForBlk1, blk1MissingData))
	assert.NoError(s.Commit())

//为块2写入pvt数据
	assert.NoError(s.Prepare(2, nil, nil))
	assert.NoError(s.Commit())
//存储中应存在ns-1:coll-1和ns-2:coll-2的数据
	ns1Coll1 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-1", blkNum: 1}, txNum: 2}
	ns2Coll2 := &dataKey{nsCollBlk: nsCollBlk{ns: "ns-2", coll: "coll-2", blkNum: 1}, txNum: 2}

//NS-1的合格MISGISDATA条目：COL-1，NS-1：COL-2（NURExcel）应该存在于商店中
	ns1Coll1elgMD := &missingDataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-1", blkNum: 1}, isEligible: true}
	ns1Coll2elgMD := &missingDataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-2", blkNum: 1}, isEligible: true}

//ineligible missingData entries for ns-3:col-1, ns-3:coll-2 (neverExpires) should exist in store
	ns3Coll1inelgMD := &missingDataKey{nsCollBlk: nsCollBlk{ns: "ns-3", coll: "coll-1", blkNum: 1}, isEligible: false}
	ns3Coll2inelgMD := &missingDataKey{nsCollBlk: nsCollBlk{ns: "ns-3", coll: "coll-2", blkNum: 1}, isEligible: false}

	testWaitForPurgerRoutineToFinish(s)
	assert.True(testDataKeyExists(t, s, ns1Coll1))
	assert.True(testDataKeyExists(t, s, ns2Coll2))

	assert.True(testMissingDataKeyExists(t, s, ns1Coll1elgMD))
	assert.True(testMissingDataKeyExists(t, s, ns1Coll2elgMD))

	assert.True(testMissingDataKeyExists(t, s, ns3Coll1inelgMD))
	assert.True(testMissingDataKeyExists(t, s, ns3Coll2inelgMD))

//为块3写入pvt数据
	assert.NoError(s.Prepare(3, nil, nil))
	assert.NoError(s.Commit())
//存储中应存在NS-1:coll-1和NS-2:coll-2的数据（因为不应在块3上启动净化程序）
	testWaitForPurgerRoutineToFinish(s)
	assert.True(testDataKeyExists(t, s, ns1Coll1))
	assert.True(testDataKeyExists(t, s, ns2Coll2))
//NS-1的合格MISGISDATA条目：COL-1，NS-1：COL-2（NURExcel）应该存在于商店中
	assert.True(testMissingDataKeyExists(t, s, ns1Coll1elgMD))
	assert.True(testMissingDataKeyExists(t, s, ns1Coll2elgMD))
//存储区中应存在NS-3:col-1、NS-3:coll-2（neverexpires）的不合格MissingData条目。
	assert.True(testMissingDataKeyExists(t, s, ns3Coll1inelgMD))
	assert.True(testMissingDataKeyExists(t, s, ns3Coll2inelgMD))

//为块4写入pvt数据
	assert.NoError(s.Prepare(4, nil, nil))
	assert.NoError(s.Commit())
//存储中不应存在NS-1的数据：coll-1（因为清除器应在块4启动）
//但是NS-2:coll-2应该存在，因为它在块5过期
	testWaitForPurgerRoutineToFinish(s)
	assert.False(testDataKeyExists(t, s, ns1Coll1))
	assert.True(testDataKeyExists(t, s, ns2Coll2))
//NS-1:coll-1的合格MissingData条目应已过期，而NS-1:coll-2（NeverExpires）应存在于存储区中。
	assert.False(testMissingDataKeyExists(t, s, ns1Coll1elgMD))
	assert.True(testMissingDataKeyExists(t, s, ns1Coll2elgMD))
//不合格的NS-3的MissingData条目：col-1应已过期，而ns-3:coll-2（neverexpires）应存在于存储区中。
	assert.False(testMissingDataKeyExists(t, s, ns3Coll1inelgMD))
	assert.True(testMissingDataKeyExists(t, s, ns3Coll2inelgMD))

//为块5写入pvt数据
	assert.NoError(s.Prepare(5, nil, nil))
	assert.NoError(s.Commit())
//NS-2:coll-2应该存在，因为尽管数据在块5过期，但清除器每秒钟启动一次。
	testWaitForPurgerRoutineToFinish(s)
	assert.False(testDataKeyExists(t, s, ns1Coll1))
	assert.True(testDataKeyExists(t, s, ns2Coll2))

//为块6写入pvt数据
	assert.NoError(s.Prepare(6, nil, nil))
	assert.NoError(s.Commit())
//NS-2:coll-2现在不应该存在（因为清洗机应该在6区启动）
	testWaitForPurgerRoutineToFinish(s)
	assert.False(testDataKeyExists(t, s, ns1Coll1))
	assert.False(testDataKeyExists(t, s, ns2Coll2))

//“NS-2:coll-1”不应该被清除（因为没有为此声明BTL）
	assert.True(testDataKeyExists(t, s, &dataKey{nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-2", blkNum: 1}, txNum: 2}))
}

func TestStoreState(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, "TestStoreState", btlPolicy)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore
	testData := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 0, []string{"ns-1:coll-1", "ns-1:coll-2"}),
	}
	_, ok := store.Prepare(1, testData, nil).(*ErrIllegalArgs)
	assert.True(ok)

	assert.Nil(store.Prepare(0, testData, nil))
	assert.NoError(store.Commit())

	assert.Nil(store.Prepare(1, testData, nil))
	_, ok = store.Prepare(2, testData, nil).(*ErrIllegalCall)
	assert.True(ok)
}

func TestInitLastCommittedBlock(t *testing.T) {
	env := NewTestStoreEnv(t, "TestStoreState", nil)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore
	existingLastBlockNum := uint64(25)
	assert.NoError(store.InitLastCommittedBlock(existingLastBlockNum))

	testEmpty(false, assert, store)
	testPendingBatch(false, assert, store)
	testLastCommittedBlockHeight(existingLastBlockNum+1, assert, store)

	env.CloseAndReopen()
	testEmpty(false, assert, store)
	testPendingBatch(false, assert, store)
	testLastCommittedBlockHeight(existingLastBlockNum+1, assert, store)

	err := store.InitLastCommittedBlock(30)
	_, ok := err.(*ErrIllegalCall)
	assert.True(ok)
}

func TestCollElgEnabled(t *testing.T) {
	testCollElgEnabled(t)
	defaultValBatchSize := ledgerconfig.GetPvtdataStoreCollElgProcMaxDbBatchSize()
	defaultValInterval := ledgerconfig.GetPvtdataStoreCollElgProcDbBatchesInterval()
	defer func() {
		viper.Set("ledger.pvtdataStore.collElgProcMaxDbBatchSize", defaultValBatchSize)
		viper.Set("ledger.pvtdataStore.collElgProcMaxDbBatchSize", defaultValInterval)
	}()
	viper.Set("ledger.pvtdataStore.collElgProcMaxDbBatchSize", 1)
	viper.Set("ledger.pvtdataStore.collElgProcDbBatchesInterval", 1)
	testCollElgEnabled(t)
}

func testCollElgEnabled(t *testing.T) {
	ledgerid := "TestCollElgEnabled"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
			{"ns-2", "coll-1"}: 0,
			{"ns-2", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, ledgerid, btlPolicy)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore

//初始状态：符合NS-1:coll-1和ns-2:coll-1

//没有块0的pvt数据
	assert.NoError(store.Prepare(0, nil, nil))
	assert.NoError(store.Commit())

//构造并提交块1
	blk1MissingData := make(ledger.TxMissingPvtDataMap)
	blk1MissingData.Add(1, "ns-1", "coll-1", true)
	blk1MissingData.Add(1, "ns-2", "coll-1", true)
	blk1MissingData.Add(4, "ns-1", "coll-2", false)
	blk1MissingData.Add(4, "ns-2", "coll-2", false)
	testDataForBlk1 := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1"}),
	}
	assert.NoError(store.Prepare(1, testDataForBlk1, blk1MissingData))
	assert.NoError(store.Commit())

//构建并提交块2
	blk2MissingData := make(ledger.TxMissingPvtDataMap)
//tx1中缺少数据不合格
	blk2MissingData.Add(1, "ns-1", "coll-2", false)
	blk2MissingData.Add(1, "ns-2", "coll-2", false)
	testDataForBlk2 := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 3, []string{"ns-1:coll-1"}),
	}
	assert.NoError(store.Prepare(2, testDataForBlk2, blk2MissingData))
	assert.NoError(store.Commit())

//检索并验证报告的丢失数据
//预期丢失的数据应仅为blk1-tx1（因为其他丢失的数据标记为不可识别）
	expectedMissingPvtDataInfo := make(ledger.MissingPvtDataInfo)
	expectedMissingPvtDataInfo.Add(1, 1, "ns-1", "coll-1")
	expectedMissingPvtDataInfo.Add(1, 1, "ns-2", "coll-1")
	missingPvtDataInfo, err := store.GetMissingPvtDataInfoForMostRecentBlocks(10)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

//启用NS-1:coll2的资格
	store.ProcessCollsEligibilityEnabled(
		5,
		map[string][]string{
			"ns-1": {"coll-2"},
		},
	)
	testutilWaitForCollElgProcToFinish(store)

//检索并验证报告的丢失数据
//预期缺少的数据应包括新的可识别集合
	expectedMissingPvtDataInfo.Add(1, 4, "ns-1", "coll-2")
	expectedMissingPvtDataInfo.Add(2, 1, "ns-1", "coll-2")
	missingPvtDataInfo, err = store.GetMissingPvtDataInfoForMostRecentBlocks(10)
	assert.NoError(err)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)

//启用{NS-2：Cul2}的资格
	store.ProcessCollsEligibilityEnabled(6,
		map[string][]string{
			"ns-2": {"coll-2"},
		},
	)
	testutilWaitForCollElgProcToFinish(store)

//检索并验证报告的丢失数据
//预期缺少的数据应包括新的可识别集合
	expectedMissingPvtDataInfo.Add(1, 4, "ns-2", "coll-2")
	expectedMissingPvtDataInfo.Add(2, 1, "ns-2", "coll-2")
	missingPvtDataInfo, err = store.GetMissingPvtDataInfoForMostRecentBlocks(10)
	assert.Equal(expectedMissingPvtDataInfo, missingPvtDataInfo)
}

func TestRollBack(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, "TestRollBack", btlPolicy)
	defer env.Cleanup()
	assert := assert.New(t)
	store := env.TestStore
	assert.NoError(store.Prepare(0, nil, nil))
	assert.NoError(store.Commit())

	pvtdata := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 0, []string{"ns-1:coll-1", "ns-1:coll-2"}),
		produceSamplePvtdata(t, 5, []string{"ns-1:coll-1", "ns-1:coll-2"}),
	}
	missingData := make(ledger.TxMissingPvtDataMap)
	missingData.Add(1, "ns-1", "coll-1", true)
	missingData.Add(5, "ns-1", "coll-1", true)
	missingData.Add(5, "ns-2", "coll-2", false)

	for i := 1; i <= 9; i++ {
		assert.NoError(store.Prepare(uint64(i), pvtdata, missingData))
		assert.NoError(store.Commit())
	}

	datakeyTx0 := &dataKey{
		nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-1"},
		txNum:     0,
	}
	datakeyTx5 := &dataKey{
		nsCollBlk: nsCollBlk{ns: "ns-1", coll: "coll-1"},
		txNum:     5,
	}
	eligibleMissingdatakey := &missingDataKey{
		nsCollBlk:  nsCollBlk{ns: "ns-1", coll: "coll-1"},
		isEligible: true,
	}

//准备块10前测试存储状态
	testPendingBatch(false, assert, store)
	testLastCommittedBlockHeight(10, assert, store)

//准备布洛克10并测试存储是否存在数据键和可交付文件丢失的数据键
	assert.NoError(store.Prepare(10, pvtdata, missingData))
	testPendingBatch(true, assert, store)
	testLastCommittedBlockHeight(10, assert, store)

	datakeyTx0.blkNum = 10
	datakeyTx5.blkNum = 10
	eligibleMissingdatakey.blkNum = 10
	assert.True(testDataKeyExists(t, store, datakeyTx0))
	assert.True(testDataKeyExists(t, store, datakeyTx5))
	assert.True(testMissingDataKeyExists(t, store, eligibleMissingdatakey))

//回滚上一个准备好的块和测试存储区，以防缺少数据键和可交付文件缺少数据键。
	store.Rollback()
	testPendingBatch(false, assert, store)
	testLastCommittedBlockHeight(10, assert, store)
	assert.False(testDataKeyExists(t, store, datakeyTx0))
	assert.False(testDataKeyExists(t, store, datakeyTx5))
	assert.False(testMissingDataKeyExists(t, store, eligibleMissingdatakey))

//For previously committed blocks the datakeys and eligibile missingdatakeys should still be present
	for i := 1; i <= 9; i++ {
		datakeyTx0.blkNum = uint64(i)
		datakeyTx5.blkNum = uint64(i)
		eligibleMissingdatakey.blkNum = uint64(i)
		assert.True(testDataKeyExists(t, store, datakeyTx0))
		assert.True(testDataKeyExists(t, store, datakeyTx5))
		assert.True(testMissingDataKeyExists(t, store, eligibleMissingdatakey))
	}
}

//TODO添加用于模拟调用“prepare”和“commit”/“rollback”之间崩溃的测试-[fab-13099]

func testEmpty(expectedEmpty bool, assert *assert.Assertions, store Store) {
	isEmpty, err := store.IsEmpty()
	assert.NoError(err)
	assert.Equal(expectedEmpty, isEmpty)
}

func testPendingBatch(expectedPending bool, assert *assert.Assertions, store Store) {
	hasPendingBatch, err := store.HasPendingBatch()
	assert.NoError(err)
	assert.Equal(expectedPending, hasPendingBatch)
}

func testLastCommittedBlockHeight(expectedBlockHt uint64, assert *assert.Assertions, store Store) {
	blkHt, err := store.LastCommittedBlockHeight()
	assert.NoError(err)
	assert.Equal(expectedBlockHt, blkHt)
}

func testDataKeyExists(t *testing.T, s Store, dataKey *dataKey) bool {
	dataKeyBytes := encodeDataKey(dataKey)
	val, err := s.(*store).db.Get(dataKeyBytes)
	assert.NoError(t, err)
	return len(val) != 0
}

func testMissingDataKeyExists(t *testing.T, s Store, missingDataKey *missingDataKey) bool {
	dataKeyBytes := encodeMissingDataKey(missingDataKey)
	val, err := s.(*store).db.Get(dataKeyBytes)
	assert.NoError(t, err)
	return len(val) != 0
}

func testWaitForPurgerRoutineToFinish(s Store) {
	time.Sleep(1 * time.Second)
	s.(*store).purgerLock.Lock()
	s.(*store).purgerLock.Unlock()
}

func testutilWaitForCollElgProcToFinish(s Store) {
	s.(*store).collElgProcSync.waitForDone()
}

func produceSamplePvtdata(t *testing.T, txNum uint64, nsColls []string) *ledger.TxPvtData {
	builder := rwsetutil.NewRWSetBuilder()
	for _, nsColl := range nsColls {
		nsCollSplit := strings.Split(nsColl, ":")
		ns := nsCollSplit[0]
		coll := nsCollSplit[1]
		builder.AddToPvtAndHashedWriteSet(ns, coll, fmt.Sprintf("key-%s-%s", ns, coll), []byte(fmt.Sprintf("value-%s-%s", ns, coll)))
	}
	simRes, err := builder.GetTxSimulationResults()
	assert.NoError(t, err)
	return &ledger.TxPvtData{SeqInBlock: txNum, WriteSet: simRes.PvtSimulationResults}
}
