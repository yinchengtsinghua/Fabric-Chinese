
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
	"math"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/willf/bitset"
)

func prepareStoreEntries(blockNum uint64, pvtData []*ledger.TxPvtData, btlPolicy pvtdatapolicy.BTLPolicy,
	missingPvtData ledger.TxMissingPvtDataMap) (*storeEntries, error) {
	dataEntries := prepareDataEntries(blockNum, pvtData)

	missingDataEntries := prepareMissingDataEntries(blockNum, missingPvtData)

	expiryEntries, err := prepareExpiryEntries(blockNum, dataEntries, missingDataEntries, btlPolicy)
	if err != nil {
		return nil, err
	}

	return &storeEntries{
		dataEntries:        dataEntries,
		expiryEntries:      expiryEntries,
		missingDataEntries: missingDataEntries}, nil
}

func prepareDataEntries(blockNum uint64, pvtData []*ledger.TxPvtData) []*dataEntry {
	var dataEntries []*dataEntry
	for _, txPvtdata := range pvtData {
		for _, nsPvtdata := range txPvtdata.WriteSet.NsPvtRwset {
			for _, collPvtdata := range nsPvtdata.CollectionPvtRwset {
				txnum := txPvtdata.SeqInBlock
				ns := nsPvtdata.Namespace
				coll := collPvtdata.CollectionName
				dataKey := &dataKey{nsCollBlk{ns, coll, blockNum}, txnum}
				dataEntries = append(dataEntries, &dataEntry{key: dataKey, value: collPvtdata})
			}
		}
	}
	return dataEntries
}

func prepareMissingDataEntries(committingBlk uint64, missingPvtData ledger.TxMissingPvtDataMap) map[missingDataKey]*bitset.BitSet {
	missingDataEntries := make(map[missingDataKey]*bitset.BitSet)

	for txNum, missingData := range missingPvtData {
		for _, nsColl := range missingData {
			key := missingDataKey{nsCollBlk{nsColl.Namespace, nsColl.Collection, committingBlk},
				nsColl.IsEligible}

			if _, ok := missingDataEntries[key]; !ok {
				missingDataEntries[key] = &bitset.BitSet{}
			}
			bitmap := missingDataEntries[key]

			bitmap.Set(uint(txNum))
		}
	}

	return missingDataEntries
}

//PrepareExpiryEntries返回committingBLK中存在的两个私有数据的到期条目
//和失踪的私人。
func prepareExpiryEntries(committingBlk uint64, dataEntries []*dataEntry, missingDataEntries map[missingDataKey]*bitset.BitSet,
	btlPolicy pvtdatapolicy.BTLPolicy) ([]*expiryEntry, error) {

	var expiryEntries []*expiryEntry
	mapByExpiringBlk := make(map[uint64]*ExpiryData)

//1。为未丢失的数据准备ExpiryData
	for _, dataEntry := range dataEntries {
		prepareExpiryEntriesForPresentData(mapByExpiringBlk, dataEntry.key, btlPolicy)
	}

//2。为丢失的数据准备ExpiryData
	for missingDataKey := range missingDataEntries {
		prepareExpiryEntriesForMissingData(mapByExpiringBlk, &missingDataKey, btlPolicy)
	}

	for expiryBlk, expiryData := range mapByExpiringBlk {
		expiryKey := &expiryKey{expiringBlk: expiryBlk, committingBlk: committingBlk}
		expiryEntries = append(expiryEntries, &expiryEntry{key: expiryKey, value: expiryData})
	}

	return expiryEntries, nil
}

//PrepareExpiryDataforPresentData为未丢失的pvt数据创建ExpiryData
func prepareExpiryEntriesForPresentData(mapByExpiringBlk map[uint64]*ExpiryData, dataKey *dataKey, btlPolicy pvtdatapolicy.BTLPolicy) error {
	expiringBlk, err := btlPolicy.GetExpiringBlock(dataKey.ns, dataKey.coll, dataKey.blkNum)
	if err != nil {
		return err
	}
	if neverExpires(expiringBlk) {
		return nil
	}

	expiryData := getOrCreateExpiryData(mapByExpiringBlk, expiringBlk)

	expiryData.addPresentData(dataKey.ns, dataKey.coll, dataKey.txNum)
	return nil
}

//PrepareExpiryDataforMissingData为缺少的pvt数据创建ExpiryData
func prepareExpiryEntriesForMissingData(mapByExpiringBlk map[uint64]*ExpiryData, missingKey *missingDataKey, btlPolicy pvtdatapolicy.BTLPolicy) error {
	expiringBlk, err := btlPolicy.GetExpiringBlock(missingKey.ns, missingKey.coll, missingKey.blkNum)
	if err != nil {
		return err
	}
	if neverExpires(expiringBlk) {
		return nil
	}

	expiryData := getOrCreateExpiryData(mapByExpiringBlk, expiringBlk)

	expiryData.addMissingData(missingKey.ns, missingKey.coll)
	return nil
}

func getOrCreateExpiryData(mapByExpiringBlk map[uint64]*ExpiryData, expiringBlk uint64) *ExpiryData {
	expiryData, ok := mapByExpiringBlk[expiringBlk]
	if !ok {
		expiryData = newExpiryData()
		mapByExpiringBlk[expiringBlk] = expiryData
	}
	return expiryData
}

//DeriveKeys从ExpireEntry构造数据键并忽略数据键
func deriveKeys(expiryEntry *expiryEntry) (dataKeys []*dataKey, missingDataKeys []*missingDataKey) {
	for ns, colls := range expiryEntry.value.Map {
//1。构造过期的现有pvt数据的数据键
		for coll, txNums := range colls.Map {
			for _, txNum := range txNums.List {
				dataKeys = append(dataKeys,
					&dataKey{nsCollBlk{ns, coll, expiryEntry.key.committingBlk}, txNum})
			}
		}
//2。构造缺少过期pvt数据的MissingDataKeys
		for coll := range colls.MissingDataMap {
//一个键用于符合条件的条目，另一个键用于符合条件的条目
			missingDataKeys = append(missingDataKeys,
				&missingDataKey{nsCollBlk{ns, coll, expiryEntry.key.committingBlk}, true})
			missingDataKeys = append(missingDataKeys,
				&missingDataKey{nsCollBlk{ns, coll, expiryEntry.key.committingBlk}, false})

		}
	}
	return
}

func passesFilter(dataKey *dataKey, filter ledger.PvtNsCollFilter) bool {
	return filter == nil || filter.Has(dataKey.ns, dataKey.coll)
}

func isExpired(key nsCollBlk, btl pvtdatapolicy.BTLPolicy, latestBlkNum uint64) (bool, error) {
	expiringBlk, err := btl.GetExpiringBlock(key.ns, key.coll, key.blkNum)
	if err != nil {
		return false, err
	}

	return latestBlkNum >= expiringBlk, nil
}

func neverExpires(expiringBlkNum uint64) bool {
	return expiringBlkNum == math.MaxUint64
}

type txPvtdataAssembler struct {
	blockNum, txNum uint64
	txWset          *rwset.TxPvtReadWriteSet
	currentNsWSet   *rwset.NsPvtReadWriteSet
	firstCall       bool
}

func newTxPvtdataAssembler(blockNum, txNum uint64) *txPvtdataAssembler {
	return &txPvtdataAssembler{blockNum, txNum, &rwset.TxPvtReadWriteSet{}, nil, true}
}

func (a *txPvtdataAssembler) add(ns string, collPvtWset *rwset.CollectionPvtReadWriteSet) {
//启动NSWSET
	if a.firstCall {
		a.currentNsWSet = &rwset.NsPvtReadWriteSet{Namespace: ns}
		a.firstCall = false
	}

//如果新的NS已启动，请将现有NSWSET添加到TXWSET并启动新的NSWSET。
	if a.currentNsWSet.Namespace != ns {
		a.txWset.NsPvtRwset = append(a.txWset.NsPvtRwset, a.currentNsWSet)
		a.currentNsWSet = &rwset.NsPvtReadWriteSet{Namespace: ns}
	}
//将collwset添加到当前nswset
	a.currentNsWSet.CollectionPvtRwset = append(a.currentNsWSet.CollectionPvtRwset, collPvtWset)
}

func (a *txPvtdataAssembler) done() {
	if a.currentNsWSet != nil {
		a.txWset.NsPvtRwset = append(a.txWset.NsPvtRwset, a.currentNsWSet)
	}
	a.currentNsWSet = nil
}

func (a *txPvtdataAssembler) getTxPvtdata() *ledger.TxPvtData {
	a.done()
	return &ledger.TxPvtData{SeqInBlock: a.txNum, WriteSet: a.txWset}
}
