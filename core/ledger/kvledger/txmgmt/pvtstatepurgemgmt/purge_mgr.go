
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


package pvtstatepurgemgmt

import (
	"math"
	"sync"

	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/util"
)

//purgemgr管理清除过期的pvtdata
type PurgeMgr interface {
//PrepareForExpiringKeys为purgemgr提供了一个提前进行后台工作的机会（如果有的话）。
	PrepareForExpiringKeys(expiringAtBlk uint64)
//waitForPrepareToFinish将保留调用方，直到完成由“PrepareForExpiringKeys”引发的后台goroutine。
	WaitForPrepareToFinish()
//DeleteExpiredAndUpdateBooking通过添加过期pvtData的删除来更新簿记并修改更新批。
	DeleteExpiredAndUpdateBookkeeping(
		pvtUpdates *privacyenabledstate.PvtUpdateBatch,
		hashedUpdates *privacyenabledstate.HashedUpdateBatch) error
//updateBookkeepingForpvtDataofolBlocks使用给定的pvTupDate更新簿记员中的现有到期条目
	UpdateBookkeepingForPvtDataOfOldBlocks(pvtUpdates *privacyenabledstate.PvtUpdateBatch) error
//blockcommitdone是将块提交到分类帐时对purgemgr的回调
	BlockCommitDone() error
}

type keyAndVersion struct {
	key             string
	committingBlock uint64
	purgeKeyOnly    bool
}

type expiryInfoMap map[privacyenabledstate.HashedCompositeKey]*keyAndVersion

type workingset struct {
	toPurge             expiryInfoMap
	toClearFromSchedule []*expiryInfoKey
	expiringBlk         uint64
	err                 error
}

type purgeMgr struct {
	btlPolicy pvtdatapolicy.BTLPolicy
	db        privacyenabledstate.DB
	expKeeper expiryKeeper

	lock    *sync.Mutex
	waitGrp *sync.WaitGroup

	workingset *workingset
}

//实例化pugemgr实例化一个pugemgr。
func InstantiatePurgeMgr(ledgerid string, db privacyenabledstate.DB, btlPolicy pvtdatapolicy.BTLPolicy, bookkeepingProvider bookkeeping.Provider) (PurgeMgr, error) {
	return &purgeMgr{
		btlPolicy: btlPolicy,
		db:        db,
		expKeeper: newExpiryKeeper(ledgerid, bookkeepingProvider),
		lock:      &sync.Mutex{},
		waitGrp:   &sync.WaitGroup{},
	}, nil
}

//PrepareForExpiringKeys在接口“pugemgr”中实现函数
func (p *purgeMgr) PrepareForExpiringKeys(expiringAtBlk uint64) {
	p.waitGrp.Add(1)
	go func() {
		p.lock.Lock()
		p.waitGrp.Done()
		defer p.lock.Unlock()
		p.workingset = p.prepareWorkingsetFor(expiringAtBlk)
	}()
	p.waitGrp.Wait()
}

//waitForPrepareToFinish在接口“pugemgr”中实现函数
func (p *purgeMgr) WaitForPrepareToFinish() {
	p.lock.Lock()
	p.lock.Unlock()
}

func (p *purgeMgr) UpdateBookkeepingForPvtDataOfOldBlocks(pvtUpdates *privacyenabledstate.PvtUpdateBatch) error {
	builder := newExpiryScheduleBuilder(p.btlPolicy)
	pvtUpdateCompositeKeyMap := pvtUpdates.ToCompositeKeyMap()
	for k, vv := range pvtUpdateCompositeKeyMap {
		builder.add(k.Namespace, k.CollectionName, k.Key, util.ComputeStringHash(k.Key), vv)
	}

	var updatedList []*expiryInfo
	for _, toAdd := range builder.getExpiryInfo() {
		toUpdate, err := p.expKeeper.retrieveByExpiryKey(toAdd.expiryInfoKey)
		if err != nil {
			return err
		}
//尽管我们可以更新现有条目（因为应该有一个
//为了简单起见
//昂贵，我们附加了一个新条目
		toUpdate.pvtdataKeys.addAll(toAdd.pvtdataKeys)
		updatedList = append(updatedList, toUpdate)
	}

//因为过期密钥列表可能是在最后一个
//常规块提交，我们需要更新列表。这是因为，
//正在提交的某些旧pvtdata可能会过期
//在下一个常规块提交期间。因此，相应的
//过期密钥列表中的hashedkey将缺少pvtdata。
	p.addMissingPvtDataToWorkingSet(pvtUpdateCompositeKeyMap)

	return p.expKeeper.updateBookkeeping(updatedList, nil)
}

func (p *purgeMgr) addMissingPvtDataToWorkingSet(pvtKeys privacyenabledstate.PvtdataCompositeKeyMap) {
	if p.workingset == nil || len(p.workingset.toPurge) == 0 {
		return
	}

	for k := range pvtKeys {
		hashedCompositeKey := privacyenabledstate.HashedCompositeKey{
			Namespace:      k.Namespace,
			CollectionName: k.CollectionName,
			KeyHash:        string(util.ComputeStringHash(k.Key))}

		toPurgeKey, ok := p.workingset.toPurge[hashedCompositeKey]
		if !ok {
//相应的hashedkey不在
//过期密钥列表
			continue
		}

//如果设置了pugeKeyOnly，则表示pvtkey的版本
//存储在statedb中的版本早于hashedkey的版本。
//因此，只需清除pvtkey（过期块高度
//因为最近的hashedkey会更高）。如果最近
//相应hashedkey的pvtkey正在提交，我们需要
//从TopurgeList中移除pugeKeyOnly条目
//由缺少pvtdata的提交更新
		if toPurgeKey.purgeKeyOnly {
			delete(p.workingset.toPurge, hashedCompositeKey)
		} else {
			toPurgeKey.key = k.Key
		}
	}
}

//deleteExpiredAndUpdateBooking在接口“purgemgr”中实现函数
func (p *purgeMgr) DeleteExpiredAndUpdateBookkeeping(
	pvtUpdates *privacyenabledstate.PvtUpdateBatch,
	hashedUpdates *privacyenabledstate.HashedUpdateBatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.workingset.err != nil {
		return p.workingset.err
	}

	listExpiryInfo, err := buildExpirySchedule(p.btlPolicy, pvtUpdates, hashedUpdates)
	if err != nil {
		return err
	}

//对于选定要清除的每个键，检查当前块中的键是否未得到更新，
//将其删除添加到pvt和哈希更新的更新批中
	for compositeHashedKey, keyAndVersion := range p.workingset.toPurge {
		ns := compositeHashedKey.Namespace
		coll := compositeHashedKey.CollectionName
		keyHash := []byte(compositeHashedKey.KeyHash)
		key := keyAndVersion.key
		purgeKeyOnly := keyAndVersion.purgeKeyOnly
		hashUpdated := hashedUpdates.Contains(ns, coll, keyHash)
		pvtKeyUpdated := pvtUpdates.Contains(ns, coll, key)

		logger.Debugf("Checking whether the key [ns=%s, coll=%s, keyHash=%x, purgeKeyOnly=%t] "+
			"is updated in the update batch for the committing block - hashUpdated=%t, and pvtKeyUpdated=%t",
			ns, coll, keyHash, purgeKeyOnly, hashUpdated, pvtKeyUpdated)

		expiringTxVersion := version.NewHeight(p.workingset.expiringBlk, math.MaxUint64)
		if !hashUpdated && !purgeKeyOnly {
			logger.Debugf("Adding the hashed key to be purged to the delete list in the update batch")
			hashedUpdates.Delete(ns, coll, keyHash, expiringTxVersion)
		}
		if key != "" && !pvtKeyUpdated {
			logger.Debugf("Adding the pvt key to be purged to the delete list in the update batch")
			pvtUpdates.Delete(ns, coll, key, expiringTxVersion)
		}
	}
	return p.expKeeper.updateBookkeeping(listExpiryInfo, nil)
}

//blockcommitdone在接口“purgemgr”中实现函数
//清除计划的这些孤立条目也可以在单独的后台例程中批量清除。
//如果我们维护以下逻辑（即块提交后清除条目），我们需要一个TODO-
//我们需要在开始时执行一个签入，因为块提交和
//对该函数的调用导致为最后一个块计划的删除的孤立项
//另外，另一种方法是在同一批中删除这些条目，为未来到期添加条目-
//但是，这需要通过重放区块链中的最后一个块来更新到期存储，以维持
//条目更新和阻止提交
func (p *purgeMgr) BlockCommitDone() error {
	defer func() { p.workingset = nil }()
	return p.expKeeper.updateBookkeeping(nil, p.workingset.toClearFromSchedule)
}

//PrepareWorkingSetfor返回给定过期块“expiringatblk”的工作集。
//此工作集包含pvt数据键，这些数据键将随着块“expiringatblk”的提交而过期。
func (p *purgeMgr) prepareWorkingsetFor(expiringAtBlk uint64) *workingset {
	logger.Debugf("Preparing potential purge list working-set for expiringAtBlk [%d]", expiringAtBlk)
	workingset := &workingset{expiringBlk: expiringAtBlk}
//从簿记员那里取回钥匙
	expiryInfo, err := p.expKeeper.retrieve(expiringAtBlk)
	if err != nil {
		workingset.err = err
		return workingset
	}
//将密钥转换为表单，以便符合清除条件的每个散列密钥都显示在“topurge”中。
	toPurge := transformToExpiryInfoMap(expiryInfo)
//加载哈希键的最新版本
	p.preloadCommittedVersionsInCache(toPurge)
	var expiryInfoKeysToClear []*expiryInfoKey

	if len(toPurge) == 0 {
		logger.Debugf("No expiry entry found for expiringAtBlk [%d]", expiringAtBlk)
		return workingset
	}
	logger.Debugf("Total [%d] expiring entries found. Evaluaitng whether some of these keys have been overwritten in later blocks...", len(toPurge))

	for purgeEntryK, purgeEntryV := range toPurge {
		logger.Debugf("Evaluating for hashedKey [%s]", purgeEntryK)
		expiryInfoKeysToClear = append(expiryInfoKeysToClear, &expiryInfoKey{committingBlk: purgeEntryV.committingBlock, expiryBlk: expiringAtBlk})
		currentVersion, err := p.db.GetKeyHashVersion(purgeEntryK.Namespace, purgeEntryK.CollectionName, []byte(purgeEntryK.KeyHash))
		if err != nil {
			workingset.err = err
			return workingset
		}

		if sameVersion(currentVersion, purgeEntryV.committingBlock) {
			logger.Debugf(
				"The version of the hashed key in the committed state and in the expiry entry is same " +
					"hence, keeping the entry in the purge list")
			continue
		}

		logger.Debugf("The version of the hashed key in the committed state and in the expiry entry is different")
		if purgeEntryV.key != "" {
			logger.Debugf("The expiry entry also contains the raw key along with the key hash")
			committedPvtVerVal, err := p.db.GetPrivateData(purgeEntryK.Namespace, purgeEntryK.CollectionName, purgeEntryV.key)
			if err != nil {
				workingset.err = err
				return workingset
			}

			if sameVersionFromVal(committedPvtVerVal, purgeEntryV.committingBlock) {
				logger.Debugf(
					"The version of the pvt key in the committed state and in the expiry entry is same" +
						"Including only key in the purge list and not the hashed key")
				purgeEntryV.purgeKeyOnly = true
				continue
			}
		}

//如果我们到达这里，keyhash和private密钥（如果存在，在expiry条目中）将在后面的块中更新，因此从当前清除列表中删除。
		logger.Debugf("Removing from purge list - the key hash and key (if present, in the expiry entry)")
		delete(toPurge, purgeEntryK)
	}
//清除状态的最终键
	workingset.toPurge = toPurge
//从簿记员处清除的钥匙
	workingset.toClearFromSchedule = expiryInfoKeysToClear
	return workingset
}

func (p *purgeMgr) preloadCommittedVersionsInCache(expInfoMap expiryInfoMap) {
	if !p.db.IsBulkOptimizable() {
		return
	}
	var hashedKeys []*privacyenabledstate.HashedCompositeKey
	for k := range expInfoMap {
		hashedKeys = append(hashedKeys, &k)
	}
	p.db.LoadCommittedVersionsOfPubAndHashedKeys(nil, hashedKeys)
}

func transformToExpiryInfoMap(expiryInfo []*expiryInfo) expiryInfoMap {
	expinfoMap := make(expiryInfoMap)
	for _, expinfo := range expiryInfo {
		for ns, colls := range expinfo.pvtdataKeys.Map {
			for coll, keysAndHashes := range colls.Map {
				for _, keyAndHash := range keysAndHashes.List {
					compositeKey := privacyenabledstate.HashedCompositeKey{Namespace: ns, CollectionName: coll, KeyHash: string(keyAndHash.Hash)}
					expinfoMap[compositeKey] = &keyAndVersion{key: keyAndHash.Key, committingBlock: expinfo.expiryInfoKey.committingBlk}
				}
			}
		}
	}
	return expinfoMap
}

func sameVersion(version *version.Height, blockNum uint64) bool {
	return version != nil && version.BlockNum == blockNum
}

func sameVersionFromVal(vv *statedb.VersionedValue, blockNum uint64) bool {
	return vv != nil && sameVersion(vv.Version, blockNum)
}
