
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


package ledgerstorage

import (
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/blkstorage/fsblkstorage"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	lutil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("ledgerstorage,pvtdatastorage=debug")
	viper.Set("peer.fileSystemPath", "/tmp/fabric/core/ledger/ledgerstorage")
	os.Exit(m.Run())
}

func TestStore(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.cleanup()
	provider := NewProvider()
	defer provider.Close()
	store, err := provider.Open("testLedger")
	store.Init(btlPolicyForSampleData())
	defer store.Shutdown()

	assert.NoError(t, err)
	sampleData := sampleDataWithPvtdataForSelectiveTx(t)
	for _, sampleDatum := range sampleData {
		assert.NoError(t, store.CommitWithPvtData(sampleDatum))
	}

//块1没有pvt数据
	pvtdata, err := store.GetPvtDataByNum(1, nil)
	assert.NoError(t, err)
	assert.Nil(t, pvtdata)

//块4没有pvt数据
	pvtdata, err = store.GetPvtDataByNum(4, nil)
	assert.NoError(t, err)
	assert.Nil(t, pvtdata)

//块2具有Tx 3、5和6的pvt数据。然而，TX 6
//在块中标记为无效，因此不应
//已存储
	pvtdata, err = store.GetPvtDataByNum(2, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pvtdata))
	assert.Equal(t, uint64(3), pvtdata[0].SeqInBlock)
	assert.Equal(t, uint64(5), pvtdata[1].SeqInBlock)

//块3只有tx 4和6的pvt数据
	pvtdata, err = store.GetPvtDataByNum(3, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pvtdata))
	assert.Equal(t, uint64(4), pvtdata[0].SeqInBlock)
	assert.Equal(t, uint64(6), pvtdata[1].SeqInBlock)

	blockAndPvtdata, err := store.GetPvtDataAndBlockByNum(2, nil)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(sampleData[2].Block, blockAndPvtdata.Block))

	blockAndPvtdata, err = store.GetPvtDataAndBlockByNum(3, nil)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(sampleData[3].Block, blockAndPvtdata.Block))

//带过滤器的块3的pvt数据检索应返回过滤后的pvt data
	filter := ledger.NewPvtNsCollFilter()
	filter.Add("ns-1", "coll-1")
	blockAndPvtdata, err = store.GetPvtDataAndBlockByNum(3, filter)
	assert.NoError(t, err)
	assert.Equal(t, sampleData[3].Block, blockAndPvtdata.Block)
//应存在两个交易
	assert.Equal(t, 2, len(blockAndPvtdata.PvtData))
//由于筛选的原因，Tran编号4和6都应该只有一个集合
	assert.Equal(t, 1, len(blockAndPvtdata.PvtData[4].WriteSet.NsPvtRwset))
	assert.Equal(t, 1, len(blockAndPvtdata.PvtData[6].WriteSet.NsPvtRwset))
//任何其他交易条目都应为零。
	assert.Nil(t, blockAndPvtdata.PvtData[2])

//在存在无效Tx的情况下测试缺失数据检索。块5
//缺少数据（用于tx4和tx5）。但是，tx5被标记为无效tx。
//因此，只应返回tx4丢失的数据
	expectedMissingDataInfo := make(ledger.MissingPvtDataInfo)
	expectedMissingDataInfo.Add(5, 4, "ns-4", "coll-4")
	missingDataInfo, err := store.GetMissingPvtDataInfoForMostRecentBlocks(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedMissingDataInfo, missingDataInfo)
}

func TestStoreWithExistingBlockchain(t *testing.T) {
	testLedgerid := "test-ledger"
	testEnv := newTestEnv(t)
	defer testEnv.cleanup()

//构建块存储
	attrsToIndex := []blkstorage.IndexableAttr{
		blkstorage.IndexableAttrBlockHash,
		blkstorage.IndexableAttrBlockNum,
		blkstorage.IndexableAttrTxID,
		blkstorage.IndexableAttrBlockNumTranNum,
		blkstorage.IndexableAttrBlockTxID,
		blkstorage.IndexableAttrTxValidationCode,
	}
	indexConfig := &blkstorage.IndexConfig{AttrsToIndex: attrsToIndex}
	blockStoreProvider := fsblkstorage.NewProvider(
		fsblkstorage.NewConf(ledgerconfig.GetBlockStorePath(), ledgerconfig.GetMaxBlockfileSize()),
		indexConfig)

	blkStore, err := blockStoreProvider.OpenBlockStore(testLedgerid)
	assert.NoError(t, err)
	testBlocks := testutil.ConstructTestBlocks(t, 10)

	existingBlocks := testBlocks[0:9]
	blockToAdd := testBlocks[9:][0]

//直接将现有块添加到块存储，而不涉及pvtdata存储并关闭块存储
	for _, blk := range existingBlocks {
		assert.NoError(t, blkStore.AddBlock(blk))
	}
	blockStoreProvider.Close()

//模拟从1.0升级的情况：
//打开分类帐存储-使用现有块存储首次打开pvtdata存储
	provider := NewProvider()
	defer provider.Close()
	store, err := provider.Open(testLedgerid)
	store.Init(btlPolicyForSampleData())
	defer store.Shutdown()

//用现有块存储中的信息更新pvtdata存储的测试
	pvtdataBlockHt, err := store.pvtdataStore.LastCommittedBlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(9), pvtdataBlockHt)

//再添加一个带有与某个事务相关的ovtdata的块，然后在正常过程中提交
	pvtdata := samplePvtData(t, []uint64{0})
	assert.NoError(t, store.CommitWithPvtData(&ledger.BlockAndPvtData{Block: blockToAdd, PvtData: pvtdata}))
	pvtdataBlockHt, err = store.pvtdataStore.LastCommittedBlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), pvtdataBlockHt)
}

func TestCrashAfterPvtdataStorePreparation(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.cleanup()
	provider := NewProvider()
	defer provider.Close()
	store, err := provider.Open("testLedger")
	store.Init(btlPolicyForSampleData())
	defer store.Shutdown()
	assert.NoError(t, err)

	sampleData := sampleDataWithPvtdataForAllTxs(t)
	dataBeforeCrash := sampleData[0:3]
	dataAtCrash := sampleData[3]

	for _, sampleDatum := range dataBeforeCrash {
		assert.NoError(t, store.CommitWithPvtData(sampleDatum))
	}
	blokNumAtCrash := dataAtCrash.Block.Header.Number
	var pvtdataAtCrash []*ledger.TxPvtData
	for _, p := range dataAtCrash.PvtData {
		pvtdataAtCrash = append(pvtdataAtCrash, p)
	}
//仅在pvt数据存储上调用Prepare并模拟崩溃
	store.pvtdataStore.Prepare(blokNumAtCrash, pvtdataAtCrash, nil)
	store.Shutdown()
	provider.Close()
	provider = NewProvider()
	store, err = provider.Open("testLedger")
	assert.NoError(t, err)
	store.Init(btlPolicyForSampleData())

//在崩溃后启动存储时，不应该有最后一个块写入的跟踪
	_, err = store.GetPvtDataByNum(blokNumAtCrash, nil)
	_, ok := err.(*pvtdatastorage.ErrOutOfRange)
	assert.True(t, ok)

//我们应该可以再写最后一个块
	assert.NoError(t, store.CommitWithPvtData(dataAtCrash))
	pvtdata, err := store.GetPvtDataByNum(blokNumAtCrash, nil)
	assert.NoError(t, err)
	constructed := constructPvtdataMap(pvtdata)
	for k, v := range dataAtCrash.PvtData {
		ov, ok := constructed[k]
		assert.True(t, ok)
		assert.Equal(t, v.SeqInBlock, ov.SeqInBlock)
		assert.True(t, proto.Equal(v.WriteSet, ov.WriteSet))
	}
	for k, v := range constructed {
		ov, ok := dataAtCrash.PvtData[k]
		assert.True(t, ok)
		assert.Equal(t, v.SeqInBlock, ov.SeqInBlock)
		assert.True(t, proto.Equal(v.WriteSet, ov.WriteSet))
	}
}

func TestCrashBeforePvtdataStoreCommit(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.cleanup()
	provider := NewProvider()
	defer provider.Close()
	store, err := provider.Open("testLedger")
	store.Init(btlPolicyForSampleData())
	defer store.Shutdown()
	assert.NoError(t, err)

	sampleData := sampleDataWithPvtdataForAllTxs(t)
	dataBeforeCrash := sampleData[0:3]
	dataAtCrash := sampleData[3]

	for _, sampleDatum := range dataBeforeCrash {
		assert.NoError(t, store.CommitWithPvtData(sampleDatum))
	}
	blokNumAtCrash := dataAtCrash.Block.Header.Number
	var pvtdataAtCrash []*ledger.TxPvtData
	for _, p := range dataAtCrash.PvtData {
		pvtdataAtCrash = append(pvtdataAtCrash, p)
	}

//模拟在pvtdata存储上调用最终提交之前发生的崩溃
//再次启动存储后，块和pvtdata应可用。
	store.pvtdataStore.Prepare(blokNumAtCrash, pvtdataAtCrash, nil)
	store.BlockStore.AddBlock(dataAtCrash.Block)
	store.Shutdown()
	provider.Close()
	provider = NewProvider()
	store, err = provider.Open("testLedger")
	assert.NoError(t, err)
	store.Init(btlPolicyForSampleData())
	blkAndPvtdata, err := store.GetPvtDataAndBlockByNum(blokNumAtCrash, nil)
	assert.NoError(t, err)
	assert.Equal(t, dataAtCrash.MissingPvtData, blkAndPvtdata.MissingPvtData)
	assert.True(t, proto.Equal(dataAtCrash.Block, blkAndPvtdata.Block))
}

func TestAddAfterPvtdataStoreError(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.cleanup()
	provider := NewProvider()
	defer provider.Close()
	store, err := provider.Open("testLedger")
	store.Init(btlPolicyForSampleData())
	defer store.Shutdown()
	assert.NoError(t, err)

	sampleData := sampleDataWithPvtdataForAllTxs(t)
	for _, d := range sampleData[0:9] {
		assert.NoError(t, store.CommitWithPvtData(d))
	}
//再次尝试写入最后一个块。函数应跳过向私有存储区添加块
//作为pvt存储，但块存储应返回错误
	assert.Error(t, store.CommitWithPvtData(sampleData[8]))

//最后，pvt存储状态不应更改
	pvtStoreCommitHt, err := store.pvtdataStore.LastCommittedBlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(9), pvtStoreCommitHt)
	pvtStorePndingBatch, err := store.pvtdataStore.HasPendingBatch()
	assert.NoError(t, err)
	assert.False(t, pvtStorePndingBatch)

//提交正确的下一个块
	assert.NoError(t, store.CommitWithPvtData(sampleData[9]))
	pvtStoreCommitHt, err = store.pvtdataStore.LastCommittedBlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), pvtStoreCommitHt)

	pvtStorePndingBatch, err = store.pvtdataStore.HasPendingBatch()
	assert.NoError(t, err)
	assert.False(t, pvtStorePndingBatch)
}

func TestAddAfterBlkStoreError(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.cleanup()
	provider := NewProvider()
	defer provider.Close()
	store, err := provider.Open("testLedger")
	store.Init(btlPolicyForSampleData())
	defer store.Shutdown()
	assert.NoError(t, err)

	sampleData := sampleDataWithPvtdataForAllTxs(t)
	for _, d := range sampleData[0:9] {
		assert.NoError(t, store.CommitWithPvtData(d))
	}
	lastBlkAndPvtData := sampleData[9]
//直接将块添加到BlockStore
	store.BlockStore.AddBlock(lastBlkAndPvtData.Block)
//添加相同的块将导致传递由块存储引起的错误
	assert.Error(t, store.CommitWithPvtData(lastBlkAndPvtData))
//最后，pvt存储状态不应更改
	pvtStoreCommitHt, err := store.pvtdataStore.LastCommittedBlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(9), pvtStoreCommitHt)

	pvtStorePndingBatch, err := store.pvtdataStore.HasPendingBatch()
	assert.NoError(t, err)
	assert.False(t, pvtStorePndingBatch)
}

func TestConstructPvtdataMap(t *testing.T) {
	assert.Nil(t, constructPvtdataMap(nil))
}

func sampleDataWithPvtdataForSelectiveTx(t *testing.T) []*ledger.BlockAndPvtData {
	var blockAndpvtdata []*ledger.BlockAndPvtData
	blocks := testutil.ConstructTestBlocks(t, 10)
	for i := 0; i < 10; i++ {
		blockAndpvtdata = append(blockAndpvtdata, &ledger.BlockAndPvtData{Block: blocks[i]})
	}

//块2中的txnum 3、5、6具有pvtdata，但txnum 6无效
	blockAndpvtdata[2].PvtData = samplePvtData(t, []uint64{3, 5, 6})
	txFilter := lutil.TxValidationFlags(blockAndpvtdata[2].Block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	txFilter.SetFlag(6, pb.TxValidationCode_INVALID_WRITESET)
	blockAndpvtdata[2].Block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txFilter

//块3中的txnum 4、6具有pvtdata
	blockAndpvtdata[3].PvtData = samplePvtData(t, []uint64{4, 6})

//块5中的txnum 4、5缺少pvt数据，但txnum 5无效
	missingData := make(ledger.TxMissingPvtDataMap)
	missingData.Add(4, "ns-4", "coll-4", true)
	missingData.Add(5, "ns-5", "coll-5", true)
	blockAndpvtdata[5].MissingPvtData = missingData
	txFilter = lutil.TxValidationFlags(blockAndpvtdata[5].Block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	txFilter.SetFlag(5, pb.TxValidationCode_INVALID_WRITESET)
	blockAndpvtdata[5].Block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txFilter

	return blockAndpvtdata
}

func sampleDataWithPvtdataForAllTxs(t *testing.T) []*ledger.BlockAndPvtData {
	var blockAndpvtdata []*ledger.BlockAndPvtData
	blocks := testutil.ConstructTestBlocks(t, 10)
	for i := 0; i < 10; i++ {
		blockAndpvtdata = append(blockAndpvtdata,
			&ledger.BlockAndPvtData{
				Block:   blocks[i],
				PvtData: samplePvtData(t, []uint64{uint64(i), uint64(i + 1)}),
			},
		)
	}
	return blockAndpvtdata
}

func samplePvtData(t *testing.T, txNums []uint64) map[uint64]*ledger.TxPvtData {
	pvtWriteSet := &rwset.TxPvtReadWriteSet{DataModel: rwset.TxReadWriteSet_KV}
	pvtWriteSet.NsPvtRwset = []*rwset.NsPvtReadWriteSet{
		{
			Namespace: "ns-1",
			CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
				{
					CollectionName: "coll-1",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll1"),
				},
				{
					CollectionName: "coll-2",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll2"),
				},
			},
		},
	}
	var pvtData []*ledger.TxPvtData
	for _, txNum := range txNums {
		pvtData = append(pvtData, &ledger.TxPvtData{SeqInBlock: txNum, WriteSet: pvtWriteSet})
	}
	return constructPvtdataMap(pvtData)
}

func btlPolicyForSampleData() pvtdatapolicy.BTLPolicy {
	return btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
		},
	)
}
