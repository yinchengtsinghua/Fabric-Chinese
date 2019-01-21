
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


package privdata

import (
	"errors"
	"sync"
	"testing"
	"time"

	util2 "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger"
	privdatacommon "github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/gossip/privdata/mocks"
	"github.com/hyperledger/fabric/protos/common"
	gossip2 "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNoItemsToReconcile(t *testing.T) {
//方案：没有丢失的要协调的私有数据。
//协调器应该确定我们没有丢失的数据，它不需要调用ReconciliationFetcher
//提取缺少的项。
//协调器不应该出错。
	committer := &mocks.Committer{}
	fetcher := &mocks.ReconciliationFetcher{}
	missingPvtDataTracker := &mocks.MissingPvtDataTracker{}
	var missingInfo ledger.MissingPvtDataInfo
	missingInfo = make(map[uint64]ledger.MissingBlockPvtdataInfo)

	missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(missingInfo, nil)
	committer.On("GetMissingPvtDataTracker").Return(missingPvtDataTracker, nil)
	fetcher.On("FetchReconciledItems", mock.Anything).Return(nil, errors.New("this function shouldn't be called"))

	r := &Reconciler{config: &ReconcilerConfig{sleepInterval: time.Minute, batchSize: 1, IsEnabled: true}, ReconciliationFetcher: fetcher, Committer: committer}
	err := r.reconcile()

	assert.NoError(t, err)
}

func TestNotReconcilingWhenCollectionConfigNotAvailable(t *testing.T) {
//方案：协调器在尝试读取丢失的私有数据的集合配置时出错。
//结果，它移除了消化切片，并且没有要拉的消化。
//不应该出错。
	committer := &mocks.Committer{}
	fetcher := &mocks.ReconciliationFetcher{}
	configHistoryRetriever := &mocks.ConfigHistoryRetriever{}
	missingPvtDataTracker := &mocks.MissingPvtDataTracker{}
	var missingInfo ledger.MissingPvtDataInfo

	missingInfo = map[uint64]ledger.MissingBlockPvtdataInfo{
		1: map[uint64][]*ledger.MissingCollectionPvtDataInfo{
			1: {{Collection: "col1", Namespace: "chain1"}},
		},
	}

	var collectionConfigInfo ledger.CollectionConfigInfo

	missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(missingInfo, nil)
	configHistoryRetriever.On("MostRecentCollectionConfigBelow", mock.Anything, mock.Anything).Return(&collectionConfigInfo, errors.New("fail to get collection config"))
	committer.On("GetMissingPvtDataTracker").Return(missingPvtDataTracker, nil)
	committer.On("GetConfigHistoryRetriever").Return(configHistoryRetriever, nil)

	var fetchCalled bool
	fetcher.On("FetchReconciledItems", mock.Anything).Run(func(args mock.Arguments) {
		var dig2CollectionConfig = args.Get(0).(privdatacommon.Dig2CollectionConfig)
		assert.Equal(t, 0, len(dig2CollectionConfig))
		fetchCalled = true
	}).Return(nil, errors.New("called with no digests"))

	r := &Reconciler{config: &ReconcilerConfig{sleepInterval: time.Minute, batchSize: 1, IsEnabled: true}, ReconciliationFetcher: fetcher, Committer: committer}
	err := r.reconcile()

	assert.Error(t, err)
	assert.Equal(t, "called with no digests", err.Error())
	assert.True(t, fetchCalled)
}

func TestReconciliationHappyPathWithoutScheduler(t *testing.T) {
//场景：尝试协调丢失的私有数据时的快乐路径。
	committer := &mocks.Committer{}
	fetcher := &mocks.ReconciliationFetcher{}
	configHistoryRetriever := &mocks.ConfigHistoryRetriever{}
	missingPvtDataTracker := &mocks.MissingPvtDataTracker{}
	var missingInfo ledger.MissingPvtDataInfo

	missingInfo = map[uint64]ledger.MissingBlockPvtdataInfo{
		3: map[uint64][]*ledger.MissingCollectionPvtDataInfo{
			1: {{Collection: "col1", Namespace: "ns1"}},
		},
	}

	collectionConfigInfo := ledger.CollectionConfigInfo{
		CollectionConfig: &common.CollectionConfigPackage{
			Config: []*common.CollectionConfig{
				{Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name: "col1",
					},
				}},
			},
		},
		CommittingBlockNum: 1,
	}

	missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(missingInfo, nil).Run(func(_ mock.Arguments) {
		missingPvtDataTracker.Mock = mock.Mock{}
		missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(nil, nil)
	})
	configHistoryRetriever.On("MostRecentCollectionConfigBelow", mock.Anything, mock.Anything).Return(&collectionConfigInfo, nil)
	committer.On("GetMissingPvtDataTracker").Return(missingPvtDataTracker, nil)
	committer.On("GetConfigHistoryRetriever").Return(configHistoryRetriever, nil)

	result := &privdatacommon.FetchedPvtDataContainer{}
	fetcher.On("FetchReconciledItems", mock.Anything).Run(func(args mock.Arguments) {
		var dig2CollectionConfig = args.Get(0).(privdatacommon.Dig2CollectionConfig)
		assert.Equal(t, 1, len(dig2CollectionConfig))
		for digest := range dig2CollectionConfig {
			hash := util2.ComputeSHA256([]byte("rws-pre-image"))
			element := &gossip2.PvtDataElement{
				Digest: &gossip2.PvtDataDigest{
					TxId:       digest.TxId,
					BlockSeq:   digest.BlockSeq,
					Collection: digest.Collection,
					Namespace:  digest.Namespace,
					SeqInBlock: digest.SeqInBlock,
				},
				Payload: [][]byte{hash},
			}
			result.AvailableElements = append(result.AvailableElements, element)
		}
	}).Return(result, nil)

	var commitPvtDataOfOldBlocksHappened bool
	var blockNum, seqInBlock uint64
	blockNum = 3
	seqInBlock = 1
	committer.On("CommitPvtDataOfOldBlocks", mock.Anything).Run(func(args mock.Arguments) {
		var blockPvtData = args.Get(0).([]*ledger.BlockPvtData)
		assert.Equal(t, 1, len(blockPvtData))
		assert.Equal(t, blockNum, blockPvtData[0].BlockNum)
		assert.Equal(t, seqInBlock, blockPvtData[0].WriteSets[1].SeqInBlock)
		assert.Equal(t, "ns1", blockPvtData[0].WriteSets[1].WriteSet.NsPvtRwset[0].Namespace)
		assert.Equal(t, "col1", blockPvtData[0].WriteSets[1].WriteSet.NsPvtRwset[0].CollectionPvtRwset[0].CollectionName)
		commitPvtDataOfOldBlocksHappened = true
	}).Return([]*ledger.PvtdataHashMismatch{}, nil)

	r := &Reconciler{config: &ReconcilerConfig{sleepInterval: time.Minute, batchSize: 1, IsEnabled: true}, ReconciliationFetcher: fetcher, Committer: committer}
	err := r.reconcile()

	assert.NoError(t, err)
	assert.True(t, commitPvtDataOfOldBlocksHappened)
}

func TestReconciliationHappyPathWithScheduler(t *testing.T) {
//场景：尝试协调丢失的私有数据时的快乐路径。
	committer := &mocks.Committer{}
	fetcher := &mocks.ReconciliationFetcher{}
	configHistoryRetriever := &mocks.ConfigHistoryRetriever{}
	missingPvtDataTracker := &mocks.MissingPvtDataTracker{}
	var missingInfo ledger.MissingPvtDataInfo

	missingInfo = map[uint64]ledger.MissingBlockPvtdataInfo{
		3: map[uint64][]*ledger.MissingCollectionPvtDataInfo{
			1: {{Collection: "col1", Namespace: "ns1"}},
		},
	}

	collectionConfigInfo := ledger.CollectionConfigInfo{
		CollectionConfig: &common.CollectionConfigPackage{
			Config: []*common.CollectionConfig{
				{Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name: "col1",
					},
				}},
			},
		},
		CommittingBlockNum: 1,
	}

	missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(missingInfo, nil).Run(func(_ mock.Arguments) {
		missingPvtDataTracker.Mock = mock.Mock{}
		missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(nil, nil)
	})
	configHistoryRetriever.On("MostRecentCollectionConfigBelow", mock.Anything, mock.Anything).Return(&collectionConfigInfo, nil)
	committer.On("GetMissingPvtDataTracker").Return(missingPvtDataTracker, nil)
	committer.On("GetConfigHistoryRetriever").Return(configHistoryRetriever, nil)

	result := &privdatacommon.FetchedPvtDataContainer{}
	fetcher.On("FetchReconciledItems", mock.Anything).Run(func(args mock.Arguments) {
		var dig2CollectionConfig = args.Get(0).(privdatacommon.Dig2CollectionConfig)
		assert.Equal(t, 1, len(dig2CollectionConfig))
		for digest := range dig2CollectionConfig {
			hash := util2.ComputeSHA256([]byte("rws-pre-image"))
			element := &gossip2.PvtDataElement{
				Digest: &gossip2.PvtDataDigest{
					TxId:       digest.TxId,
					BlockSeq:   digest.BlockSeq,
					Collection: digest.Collection,
					Namespace:  digest.Namespace,
					SeqInBlock: digest.SeqInBlock,
				},
				Payload: [][]byte{hash},
			}
			result.AvailableElements = append(result.AvailableElements, element)
		}
	}).Return(result, nil)

	var wg sync.WaitGroup
	wg.Add(1)

	var commitPvtDataOfOldBlocksHappened bool
	var blockNum, seqInBlock uint64
	blockNum = 3
	seqInBlock = 1
	committer.On("CommitPvtDataOfOldBlocks", mock.Anything).Run(func(args mock.Arguments) {
		var blockPvtData = args.Get(0).([]*ledger.BlockPvtData)
		assert.Equal(t, 1, len(blockPvtData))
		assert.Equal(t, blockNum, blockPvtData[0].BlockNum)
		assert.Equal(t, seqInBlock, blockPvtData[0].WriteSets[1].SeqInBlock)
		assert.Equal(t, "ns1", blockPvtData[0].WriteSets[1].WriteSet.NsPvtRwset[0].Namespace)
		assert.Equal(t, "col1", blockPvtData[0].WriteSets[1].WriteSet.NsPvtRwset[0].CollectionPvtRwset[0].CollectionName)
		commitPvtDataOfOldBlocksHappened = true
		wg.Done()
	}).Return([]*ledger.PvtdataHashMismatch{}, nil)

	r := NewReconciler(committer, fetcher, &ReconcilerConfig{sleepInterval: time.Millisecond * 100, batchSize: 1, IsEnabled: true})
	r.Start()
	wg.Wait()
	r.Stop()

	assert.True(t, commitPvtDataOfOldBlocksHappened)
}

func TestReconciliationPullingMissingPrivateDataAtOnePass(t *testing.T) {
//方案：定义批大小以将丢失的私有数据检索到1
//确保即使两个数据块丢失了数据
//他们一枪就能和好。
	committer := &mocks.Committer{}
	fetcher := &mocks.ReconciliationFetcher{}
	configHistoryRetriever := &mocks.ConfigHistoryRetriever{}
	missingPvtDataTracker := &mocks.MissingPvtDataTracker{}

	missingInfo := ledger.MissingPvtDataInfo{
		4: ledger.MissingBlockPvtdataInfo{
			1: {{Collection: "col1", Namespace: "ns1"}},
		},
	}

	collectionConfigInfo := &ledger.CollectionConfigInfo{
		CollectionConfig: &common.CollectionConfigPackage{
			Config: []*common.CollectionConfig{
				{Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name: "col1",
					},
				}},
				{Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name: "col2",
					},
				}},
			},
		},
		CommittingBlockNum: 1,
	}

	stopC := make(chan struct{})
	nextC := make(chan struct{})

	missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).
		Return(missingInfo, nil).Run(func(_ mock.Arguments) {
		missingInfo := ledger.MissingPvtDataInfo{
			3: ledger.MissingBlockPvtdataInfo{
				2: {{Collection: "col2", Namespace: "ns2"}},
			},
		}
		missingPvtDataTracker.Mock = mock.Mock{}
		missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).
			Return(missingInfo, nil).Run(func(_ mock.Arguments) {
//在这里，我们要确保先停下来
//和解，因此下次调用GetMissingPvtDataInfoFormsToRecentBlocks
//将进入同一轮
			<-nextC
			missingPvtDataTracker.Mock = mock.Mock{}
			missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).
				Return(nil, nil)
		})
//确保我们调用了停止调节循环，因此
//在这次测试中，我们不能进入第二轮，不过要确保
//我们正在单程找回
		stopC <- struct{}{}
	})

	configHistoryRetriever.
		On("MostRecentCollectionConfigBelow", mock.Anything, mock.Anything).
		Return(collectionConfigInfo, nil)

	committer.On("GetMissingPvtDataTracker").Return(missingPvtDataTracker, nil)
	committer.On("GetConfigHistoryRetriever").Return(configHistoryRetriever, nil)

	result := &privdatacommon.FetchedPvtDataContainer{}
	fetcher.On("FetchReconciledItems", mock.Anything).Run(func(args mock.Arguments) {
		result.AvailableElements = make([]*gossip2.PvtDataElement, 0)
		var dig2CollectionConfig = args.Get(0).(privdatacommon.Dig2CollectionConfig)
		assert.Equal(t, 1, len(dig2CollectionConfig))
		for digest := range dig2CollectionConfig {
			hash := util2.ComputeSHA256([]byte("rws-pre-image"))
			element := &gossip2.PvtDataElement{
				Digest: &gossip2.PvtDataDigest{
					TxId:       digest.TxId,
					BlockSeq:   digest.BlockSeq,
					Collection: digest.Collection,
					Namespace:  digest.Namespace,
					SeqInBlock: digest.SeqInBlock,
				},
				Payload: [][]byte{hash},
			}
			result.AvailableElements = append(result.AvailableElements, element)
		}
	}).Return(result, nil)

	var wg sync.WaitGroup
	wg.Add(2)

	var commitPvtDataOfOldBlocksHappened bool
	pvtDataStore := make([][]*ledger.BlockPvtData, 0)
	committer.On("CommitPvtDataOfOldBlocks", mock.Anything).Run(func(args mock.Arguments) {
		var blockPvtData = args.Get(0).([]*ledger.BlockPvtData)
		assert.Equal(t, 1, len(blockPvtData))
		pvtDataStore = append(pvtDataStore, blockPvtData)
		commitPvtDataOfOldBlocksHappened = true
		wg.Done()
	}).Return([]*ledger.PvtdataHashMismatch{}, nil)

	r := NewReconciler(committer, fetcher, &ReconcilerConfig{sleepInterval: time.Millisecond * 100, batchSize: 1, IsEnabled: true})
	r.Start()
	<-stopC
	r.Stop()
	nextC <- struct{}{}
	wg.Wait()

	assert.Equal(t, 2, len(pvtDataStore))
	assert.Equal(t, uint64(4), pvtDataStore[0][0].BlockNum)
	assert.Equal(t, uint64(3), pvtDataStore[1][0].BlockNum)

	assert.Equal(t, uint64(1), pvtDataStore[0][0].WriteSets[1].SeqInBlock)
	assert.Equal(t, uint64(2), pvtDataStore[1][0].WriteSets[2].SeqInBlock)

	assert.Equal(t, "ns1", pvtDataStore[0][0].WriteSets[1].WriteSet.NsPvtRwset[0].Namespace)
	assert.Equal(t, "ns2", pvtDataStore[1][0].WriteSets[2].WriteSet.NsPvtRwset[0].Namespace)

	assert.Equal(t, "col1", pvtDataStore[0][0].WriteSets[1].WriteSet.NsPvtRwset[0].CollectionPvtRwset[0].CollectionName)
	assert.Equal(t, "col2", pvtDataStore[1][0].WriteSets[2].WriteSet.NsPvtRwset[0].CollectionPvtRwset[0].CollectionName)

	assert.True(t, commitPvtDataOfOldBlocksHappened)
}

func TestReconciliationFailedToCommit(t *testing.T) {
	committer := &mocks.Committer{}
	fetcher := &mocks.ReconciliationFetcher{}
	configHistoryRetriever := &mocks.ConfigHistoryRetriever{}
	missingPvtDataTracker := &mocks.MissingPvtDataTracker{}
	var missingInfo ledger.MissingPvtDataInfo

	missingInfo = map[uint64]ledger.MissingBlockPvtdataInfo{
		3: map[uint64][]*ledger.MissingCollectionPvtDataInfo{
			1: {{Collection: "col1", Namespace: "ns1"}},
		},
	}

	collectionConfigInfo := ledger.CollectionConfigInfo{
		CollectionConfig: &common.CollectionConfigPackage{
			Config: []*common.CollectionConfig{
				{Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name: "col1",
					},
				}},
			},
		},
		CommittingBlockNum: 1,
	}

	missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(missingInfo, nil).Run(func(_ mock.Arguments) {
		missingPvtDataTracker.Mock = mock.Mock{}
		missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(nil, nil)
	})
	configHistoryRetriever.On("MostRecentCollectionConfigBelow", mock.Anything, mock.Anything).Return(&collectionConfigInfo, nil)
	committer.On("GetMissingPvtDataTracker").Return(missingPvtDataTracker, nil)
	committer.On("GetConfigHistoryRetriever").Return(configHistoryRetriever, nil)

	result := &privdatacommon.FetchedPvtDataContainer{}
	fetcher.On("FetchReconciledItems", mock.Anything).Run(func(args mock.Arguments) {
		var dig2CollectionConfig = args.Get(0).(privdatacommon.Dig2CollectionConfig)
		assert.Equal(t, 1, len(dig2CollectionConfig))
		for digest := range dig2CollectionConfig {
			hash := util2.ComputeSHA256([]byte("rws-pre-image"))
			element := &gossip2.PvtDataElement{
				Digest: &gossip2.PvtDataDigest{
					TxId:       digest.TxId,
					BlockSeq:   digest.BlockSeq,
					Collection: digest.Collection,
					Namespace:  digest.Namespace,
					SeqInBlock: digest.SeqInBlock,
				},
				Payload: [][]byte{hash},
			}
			result.AvailableElements = append(result.AvailableElements, element)
		}
	}).Return(result, nil)

	committer.On("CommitPvtDataOfOldBlocks", mock.Anything).Return(nil, errors.New("failed to commit"))

	r := &Reconciler{config: &ReconcilerConfig{sleepInterval: time.Minute, batchSize: 1, IsEnabled: true}, ReconciliationFetcher: fetcher, Committer: committer}
	err := r.reconcile()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit")
}

func TestFailuresWhileReconcilingMissingPvtData(t *testing.T) {
	committer := &mocks.Committer{}
	fetcher := &mocks.ReconciliationFetcher{}
	committer.On("GetMissingPvtDataTracker").Return(nil, errors.New("failed to obtain missing pvt data tracker"))

	r := NewReconciler(committer, fetcher, &ReconcilerConfig{sleepInterval: time.Millisecond * 100, batchSize: 1, IsEnabled: true})
	err := r.reconcile()
	assert.Error(t, err)
	assert.Contains(t, "failed to obtain missing pvt data tracker", err.Error())

	committer.Mock = mock.Mock{}
	committer.On("GetMissingPvtDataTracker").Return(nil, nil)
	r = NewReconciler(committer, fetcher, &ReconcilerConfig{sleepInterval: time.Millisecond * 100, batchSize: 1, IsEnabled: true})
	err = r.reconcile()
	assert.Error(t, err)
	assert.Contains(t, "got nil as MissingPvtDataTracker, exiting...", err.Error())

	missingPvtDataTracker := &mocks.MissingPvtDataTracker{}
	missingPvtDataTracker.On("GetMissingPvtDataInfoForMostRecentBlocks", mock.Anything).Return(nil, errors.New("failed get missing pvt data for recent blocks"))

	committer.Mock = mock.Mock{}
	committer.On("GetMissingPvtDataTracker").Return(missingPvtDataTracker, nil)
	r = NewReconciler(committer, fetcher, &ReconcilerConfig{sleepInterval: time.Millisecond * 100, batchSize: 1, IsEnabled: true})
	err = r.reconcile()
	assert.Error(t, err)
	assert.Contains(t, "failed get missing pvt data for recent blocks", err.Error())
}
