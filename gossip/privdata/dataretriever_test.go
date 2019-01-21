
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
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/transientstore"
	privdatacommon "github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/gossip/privdata/mocks"
	"github.com/hyperledger/fabric/protos/common"
	gossip2 "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	transientstore2 "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

/*
 测试检查以下场景，它尝试获取
 给定的块序列大于可用分类帐高度，
 因此，应直接从瞬态存储器中查找数据。
**/

func TestNewDataRetriever_GetDataFromTransientStore(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	rwSetScanner := &mocks.RWSetScanner{}
	namespace := "testChaincodeName1"
	collectionName := "testCollectionName"

	rwSetScanner.On("Close")
	rwSetScanner.On("NextWithConfig").Return(&transientstore.EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          2,
		PvtSimulationResultsWithConfig: nil,
	}, nil).Once().On("NextWithConfig").Return(&transientstore.EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight: 2,
		PvtSimulationResultsWithConfig: &transientstore2.TxPvtReadWriteSetWithConfigInfo{
			PvtRwset: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(namespace, collectionName, []byte{1, 2}),
					pvtReadWriteSet(namespace, collectionName, []byte{3, 4}),
				},
			},
			CollectionConfigs: map[string]*common.CollectionConfigPackage{
				namespace: {
					Config: []*common.CollectionConfig{
						{
							Payload: &common.CollectionConfig_StaticCollectionConfig{
								StaticCollectionConfig: &common.StaticCollectionConfig{
									Name: collectionName,
								},
							},
						},
					},
				},
			},
		},
	}, nil).
Once(). //只返回一次结果，下次调用应返回并清空结果
		On("NextWithConfig").Return(nil, nil)

	dataStore.On("LedgerHeight").Return(uint64(1), nil)
	dataStore.On("GetTxPvtRWSetByTxid", "testTxID", mock.Anything).Return(rwSetScanner, nil)

	retriever := NewDataRetriever(dataStore)

//对大于当前分类帐高度的私有数据请求摘要
//使其查询丢失的私有数据的临时存储
	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   2,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 2)

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotEmpty(rwSets)
	dig2pvtRWSet := rwSets[privdatacommon.DigKey{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   2,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}]
	assertion.NotNil(dig2pvtRWSet)
	pvtRWSets := dig2pvtRWSet.RWSet
	assertion.Equal(2, len(pvtRWSets))

	var mergedRWSet []byte
	for _, rws := range pvtRWSets {
		mergedRWSet = append(mergedRWSet, rws...)
	}

	assertion.Equal([]byte{1, 2, 3, 4}, mergedRWSet)
}

/*
 可用分类帐高度大于的简单测试用例
 请求的块序列，因此将检索私有数据
 从分类帐而不是作为正在提交的数据的临时存储
**/

func TestNewDataRetriever_GetDataFromLedger(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	namespace := "testChaincodeName1"
	collectionName := "testCollectionName"

	result := []*ledger.TxPvtData{{
		WriteSet: &rwset.TxPvtReadWriteSet{
			DataModel: rwset.TxReadWriteSet_KV,
			NsPvtRwset: []*rwset.NsPvtReadWriteSet{
				pvtReadWriteSet(namespace, collectionName, []byte{1, 2}),
				pvtReadWriteSet(namespace, collectionName, []byte{3, 4}),
			},
		},
		SeqInBlock: 1,
	}}

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).Return(result, nil)

	historyRetreiver := &mocks.ConfigHistoryRetriever{}
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, namespace).Return(newCollectionConfig(collectionName), nil)
	dataStore.On("GetConfigHistoryRetriever").Return(historyRetreiver, nil)

	retriever := NewDataRetriever(dataStore)

//对大于当前分类帐高度的私有数据请求摘要
//使其查询丢失的私人数据的分类帐
	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, uint64(5))

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotEmpty(rwSets)
	pvtRWSet := rwSets[privdatacommon.DigKey{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   5,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}]
	assertion.NotEmpty(pvtRWSet)
	assertion.Equal(2, len(pvtRWSet.RWSet))

	var mergedRWSet []byte
	for _, rws := range pvtRWSet.RWSet {
		mergedRWSet = append(mergedRWSet, rws...)
	}

	assertion.Equal([]byte{1, 2, 3, 4}, mergedRWSet)
}

func TestNewDataRetriever_FailGetPvtDataFromLedger(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	namespace := "testChaincodeName1"
	collectionName := "testCollectionName"

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).
		Return(nil, errors.New("failing retrieving private data"))

	retriever := NewDataRetriever(dataStore)

//对大于当前分类帐高度的私有数据请求摘要
//使其查询丢失的私有数据的临时存储
	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, uint64(5))

	assertion := assert.New(t)
	assertion.Error(err)
	assertion.Empty(rwSets)
}

func TestNewDataRetriever_GetOnlyRelevantPvtData(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	namespace := "testChaincodeName1"
	collectionName := "testCollectionName"

	result := []*ledger.TxPvtData{{
		WriteSet: &rwset.TxPvtReadWriteSet{
			DataModel: rwset.TxReadWriteSet_KV,
			NsPvtRwset: []*rwset.NsPvtReadWriteSet{
				pvtReadWriteSet(namespace, collectionName, []byte{1}),
				pvtReadWriteSet(namespace, collectionName, []byte{2}),
				pvtReadWriteSet("invalidNamespace", collectionName, []byte{0, 0}),
				pvtReadWriteSet(namespace, "invalidCollectionName", []byte{0, 0}),
			},
		},
		SeqInBlock: 1,
	}}

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).Return(result, nil)
	historyRetreiver := &mocks.ConfigHistoryRetriever{}
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, namespace).Return(newCollectionConfig(collectionName), nil)
	dataStore.On("GetConfigHistoryRetriever").Return(historyRetreiver, nil)

	retriever := NewDataRetriever(dataStore)

//对大于当前分类帐高度的私有数据请求摘要
//使其查询丢失的私有数据的临时存储
	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 5)

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotEmpty(rwSets)
	pvtRWSet := rwSets[privdatacommon.DigKey{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   5,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}]
	assertion.NotEmpty(pvtRWSet)
	assertion.Equal(2, len(pvtRWSet.RWSet))

	var mergedRWSet []byte
	for _, rws := range pvtRWSet.RWSet {
		mergedRWSet = append(mergedRWSet, rws...)
	}

	assertion.Equal([]byte{1, 2}, mergedRWSet)
}

func TestNewDataRetriever_GetMultipleDigests(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	ns1, ns2 := "testChaincodeName1", "testChaincodeName2"
	col1, col2 := "testCollectionName1", "testCollectionName2"

	result := []*ledger.TxPvtData{
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns1, col1, []byte{1}),
					pvtReadWriteSet(ns1, col1, []byte{2}),
					pvtReadWriteSet("invalidNamespace", col1, []byte{0, 0}),
					pvtReadWriteSet(ns1, "invalidCollectionName", []byte{0, 0}),
				},
			},
			SeqInBlock: 1,
		},
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns2, col2, []byte{3}),
					pvtReadWriteSet(ns2, col2, []byte{4}),
					pvtReadWriteSet("invalidNamespace", col2, []byte{0, 0}),
					pvtReadWriteSet(ns2, "invalidCollectionName", []byte{0, 0}),
				},
			},
			SeqInBlock: 2,
		},
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns1, col1, []byte{5}),
					pvtReadWriteSet(ns2, col2, []byte{6}),
					pvtReadWriteSet("invalidNamespace", col2, []byte{0, 0}),
					pvtReadWriteSet(ns2, "invalidCollectionName", []byte{0, 0}),
				},
			},
			SeqInBlock: 3,
		},
	}

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).Return(result, nil)
	historyRetreiver := &mocks.ConfigHistoryRetriever{}
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, ns1).Return(newCollectionConfig(col1), nil)
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, ns2).Return(newCollectionConfig(col2), nil)
	dataStore.On("GetConfigHistoryRetriever").Return(historyRetreiver, nil)

	retriever := NewDataRetriever(dataStore)

//对大于当前分类帐高度的私有数据请求摘要
//使其查询丢失的私有数据的临时存储
	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  ns1,
		Collection: col1,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}, {
		Namespace:  ns2,
		Collection: col2,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 2,
	}}, 5)

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotEmpty(rwSets)
	assertion.Equal(2, len(rwSets))

	pvtRWSet := rwSets[privdatacommon.DigKey{
		Namespace:  ns1,
		Collection: col1,
		BlockSeq:   5,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}]
	assertion.NotEmpty(pvtRWSet)
	assertion.Equal(2, len(pvtRWSet.RWSet))

	var mergedRWSet []byte
	for _, rws := range pvtRWSet.RWSet {
		mergedRWSet = append(mergedRWSet, rws...)
	}

	pvtRWSet = rwSets[privdatacommon.DigKey{
		Namespace:  ns2,
		Collection: col2,
		BlockSeq:   5,
		TxId:       "testTxID",
		SeqInBlock: 2,
	}]
	assertion.NotEmpty(pvtRWSet)
	assertion.Equal(2, len(pvtRWSet.RWSet))
	for _, rws := range pvtRWSet.RWSet {
		mergedRWSet = append(mergedRWSet, rws...)
	}

	assertion.Equal([]byte{1, 2, 3, 4}, mergedRWSet)
}

func TestNewDataRetriever_EmptyWriteSet(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	ns1 := "testChaincodeName1"
	col1 := "testCollectionName1"

	result := []*ledger.TxPvtData{
		{
			SeqInBlock: 1,
		},
	}

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).Return(result, nil)
	historyRetreiver := &mocks.ConfigHistoryRetriever{}
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, ns1).Return(newCollectionConfig(col1), nil)
	dataStore.On("GetConfigHistoryRetriever").Return(historyRetreiver, nil)

	retriever := NewDataRetriever(dataStore)

	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  ns1,
		Collection: col1,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 5)

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotEmpty(rwSets)

	pvtRWSet := rwSets[privdatacommon.DigKey{
		Namespace:  ns1,
		Collection: col1,
		BlockSeq:   5,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}]
	assertion.NotEmpty(pvtRWSet)
	assertion.Empty(pvtRWSet.RWSet)

}

func TestNewDataRetriever_FailedObtainConfigHistoryRetriever(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	ns1 := "testChaincodeName1"
	col1 := "testCollectionName1"

	result := []*ledger.TxPvtData{
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns1, col1, []byte{1}),
					pvtReadWriteSet(ns1, col1, []byte{2}),
				},
			},
			SeqInBlock: 1,
		},
	}

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).Return(result, nil)
	dataStore.On("GetConfigHistoryRetriever").Return(nil, errors.New("failed to obtain ConfigHistoryRetriever"))

	retriever := NewDataRetriever(dataStore)

	_, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  ns1,
		Collection: col1,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 5)

	assertion := assert.New(t)
	assertion.Contains(err.Error(), "failed to obtain ConfigHistoryRetriever")
}

func TestNewDataRetriever_NoCollectionConfig(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	ns1, ns2 := "testChaincodeName1", "testChaincodeName2"
	col1, col2 := "testCollectionName1", "testCollectionName2"

	result := []*ledger.TxPvtData{
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns1, col1, []byte{1}),
					pvtReadWriteSet(ns1, col1, []byte{2}),
				},
			},
			SeqInBlock: 1,
		},
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns2, col2, []byte{3}),
					pvtReadWriteSet(ns2, col2, []byte{4}),
				},
			},
			SeqInBlock: 2,
		},
	}

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).Return(result, nil)
	historyRetreiver := &mocks.ConfigHistoryRetriever{}
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, ns1).
		Return(newCollectionConfig(col1), errors.New("failed to obtain collection config"))
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, ns2).
		Return(nil, nil)
	dataStore.On("GetConfigHistoryRetriever").Return(historyRetreiver, nil)

	retriever := NewDataRetriever(dataStore)
	assertion := assert.New(t)

	_, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  ns1,
		Collection: col1,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 5)
	assertion.Error(err)
	assertion.Contains(err.Error(), "cannot find recent collection config update below block sequence")

	_, _, err = retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  ns2,
		Collection: col2,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 2,
	}}, 5)
	assertion.Error(err)
	assertion.Contains(err.Error(), "no collection config update below block sequence")
}

func TestNewDataRetriever_FailedGetLedgerHeight(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	ns1 := "testChaincodeName1"
	col1 := "testCollectionName1"

	dataStore.On("LedgerHeight").Return(uint64(0), errors.New("failed to read ledger height"))
	retriever := NewDataRetriever(dataStore)

	_, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  ns1,
		Collection: col1,
		BlockSeq:   uint64(5),
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 5)

	assertion := assert.New(t)
	assertion.Error(err)
	assertion.Contains(err.Error(), "failed to read ledger height")
}

func TestNewDataRetriever_FailToReadFromTransientStore(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	namespace := "testChaincodeName1"
	collectionName := "testCollectionName"

	dataStore.On("LedgerHeight").Return(uint64(1), nil)
	dataStore.On("GetTxPvtRWSetByTxid", "testTxID", mock.Anything).
		Return(nil, errors.New("fail to read form transient store"))

	retriever := NewDataRetriever(dataStore)

	rwset, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   2,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 2)

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.Empty(rwset)
}

func TestNewDataRetriever_FailedToReadNext(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	rwSetScanner := &mocks.RWSetScanner{}
	namespace := "testChaincodeName1"
	collectionName := "testCollectionName"

	rwSetScanner.On("Close")
	rwSetScanner.On("NextWithConfig").Return(nil, errors.New("failed to read next"))

	dataStore.On("LedgerHeight").Return(uint64(1), nil)
	dataStore.On("GetTxPvtRWSetByTxid", "testTxID", mock.Anything).Return(rwSetScanner, nil)

	retriever := NewDataRetriever(dataStore)

//对大于当前分类帐高度的私有数据请求摘要
//使其查询丢失的私有数据的临时存储
	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   2,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 2)

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.Empty(rwSets)
}

func TestNewDataRetriever_EmptyPvtRWSetInTransientStore(t *testing.T) {
	t.Parallel()
	dataStore := &mocks.DataStore{}

	rwSetScanner := &mocks.RWSetScanner{}
	namespace := "testChaincodeName1"
	collectionName := "testCollectionName"

	rwSetScanner.On("Close")
	rwSetScanner.On("NextWithConfig").Return(&transientstore.EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight: 2,
		PvtSimulationResultsWithConfig: &transientstore2.TxPvtReadWriteSetWithConfigInfo{
			CollectionConfigs: map[string]*common.CollectionConfigPackage{
				namespace: {
					Config: []*common.CollectionConfig{
						{
							Payload: &common.CollectionConfig_StaticCollectionConfig{
								StaticCollectionConfig: &common.StaticCollectionConfig{
									Name: collectionName,
								},
							},
						},
					},
				},
			},
		},
	}, nil).
Once(). //只返回一次结果，下次调用应返回并清空结果
		On("NextWithConfig").Return(nil, nil)

	dataStore.On("LedgerHeight").Return(uint64(1), nil)
	dataStore.On("GetTxPvtRWSetByTxid", "testTxID", mock.Anything).Return(rwSetScanner, nil)

	retriever := NewDataRetriever(dataStore)

	rwSets, _, err := retriever.CollectionRWSet([]*gossip2.PvtDataDigest{{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   2,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}}, 2)

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotEmpty(rwSets)
	assertion.Empty(rwSets[privdatacommon.DigKey{
		Namespace:  namespace,
		Collection: collectionName,
		BlockSeq:   2,
		TxId:       "testTxID",
		SeqInBlock: 1,
	}])
}

func newCollectionConfig(collectionName string) *ledger.CollectionConfigInfo {
	return &ledger.CollectionConfigInfo{
		CollectionConfig: &common.CollectionConfigPackage{
			Config: []*common.CollectionConfig{
				{
					Payload: &common.CollectionConfig_StaticCollectionConfig{
						StaticCollectionConfig: &common.StaticCollectionConfig{
							Name: collectionName,
						},
					},
				},
			},
		},
	}
}

func pvtReadWriteSet(ns string, collectionName string, data []byte) *rwset.NsPvtReadWriteSet {
	return &rwset.NsPvtReadWriteSet{
		Namespace: ns,
		CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{{
			CollectionName: collectionName,
			Rwset:          data,
		}},
	}
}
