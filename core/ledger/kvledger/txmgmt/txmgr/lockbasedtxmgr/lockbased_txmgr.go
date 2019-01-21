
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
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/pvtstatepurgemgmt"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/queryutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/valimpl"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
)

var logger = flogging.MustGetLogger("lockbasedtxmgr")

//lockbasedtxmgr是接口“txmgmt.txmgr”的简单实现。
//此实现使用读写锁来防止事务模拟和提交之间的冲突
type LockBasedTxMgr struct {
	ledgerid        string
	db              privacyenabledstate.DB
	pvtdataPurgeMgr *pvtdataPurgeMgr
	validator       validator.Validator
	stateListeners  []ledger.StateListener
	ccInfoProvider  ledger.DeployedChaincodeInfoProvider
	commitRWLock    sync.RWMutex
	oldBlockCommit  sync.Mutex
	current         *current
}

type current struct {
	block     *common.Block
	batch     *privacyenabledstate.UpdateBatch
	listeners []ledger.StateListener
}

func (c *current) blockNum() uint64 {
	return c.block.Header.Number
}

func (c *current) maxTxNumber() uint64 {
	return uint64(len(c.block.Data.Data)) - 1
}

//newlockbasedtxmgr构造newlockbasedtxmgr的新实例
func NewLockBasedTxMgr(ledgerid string, db privacyenabledstate.DB, stateListeners []ledger.StateListener,
	btlPolicy pvtdatapolicy.BTLPolicy, bookkeepingProvider bookkeeping.Provider, ccInfoProvider ledger.DeployedChaincodeInfoProvider) (*LockBasedTxMgr, error) {
	db.Open()
	txmgr := &LockBasedTxMgr{
		ledgerid:       ledgerid,
		db:             db,
		stateListeners: stateListeners,
		ccInfoProvider: ccInfoProvider,
	}
	pvtstatePurgeMgr, err := pvtstatepurgemgmt.InstantiatePurgeMgr(ledgerid, db, btlPolicy, bookkeepingProvider)
	if err != nil {
		return nil, err
	}
	txmgr.pvtdataPurgeMgr = &pvtdataPurgeMgr{pvtstatePurgeMgr, false}
	txmgr.validator = valimpl.NewStatebasedValidator(txmgr, db)
	return txmgr, nil
}

//GetLastSavepoint返回在保存点中记录的块编号，
//如果找不到保存点，则返回0
func (txmgr *LockBasedTxMgr) GetLastSavepoint() (*version.Height, error) {
	return txmgr.db.GetLatestSavePoint()
}

//newQueryExecutor在接口“txmgmt.txmgr”中实现方法
func (txmgr *LockBasedTxMgr) NewQueryExecutor(txid string) (ledger.QueryExecutor, error) {
	qe := newQueryExecutor(txmgr, txid)
	txmgr.commitRWLock.RLock()
	return qe, nil
}

//newtxSimulator在接口“txmgmt.txmgr”中实现方法
func (txmgr *LockBasedTxMgr) NewTxSimulator(txid string) (ledger.TxSimulator, error) {
	logger.Debugf("constructing new tx simulator")
	s, err := newLockBasedTxSimulator(txmgr, txid)
	if err != nil {
		return nil, err
	}
	txmgr.commitRWLock.RLock()
	return s, nil
}

//validateAndPrepare在接口“txmgmt.txmgr”中实现方法
func (txmgr *LockBasedTxMgr) ValidateAndPrepare(blockAndPvtdata *ledger.BlockAndPvtData, doMVCCValidation bool) (
	[]*txmgr.TxStatInfo, error,
) {
//在validateAndPrepare（）、PrepareExpiringKeys（）和
//removestaleandcommitpvttataofoldBlocks（），我们只能允许一个
//一次执行的函数。原因是每个函数调用
//loadcommittedversions（），它将清除
//临时缓冲区和加载新条目（这样的临时缓冲区不是
//适用于goleveldb）。因此，这三个函数可以
//交错并取消批量读取API提供的优化。
//Once the ledger cache (FAB-103) is introduced and existing
//loadcommittedversions（）被重构以返回映射，我们可以允许
//这三个函数是并行执行的。
	logger.Debugf("Waiting for purge mgr to finish the background job of computing expirying keys for the block")
	txmgr.pvtdataPurgeMgr.WaitForPrepareToFinish()
	txmgr.oldBlockCommit.Lock()
	defer txmgr.oldBlockCommit.Unlock()
	logger.Debug("lock acquired on oldBlockCommit for validating read set version against the committed version")

	block := blockAndPvtdata.Block
	logger.Debugf("Validating new block with num trans = [%d]", len(block.Data.Data))
	batch, txstatsInfo, err := txmgr.validator.ValidateAndPrepareBatch(blockAndPvtdata, doMVCCValidation)
	if err != nil {
		txmgr.reset()
		return nil, err
	}
	txmgr.current = &current{block: block, batch: batch}
	if err := txmgr.invokeNamespaceListeners(); err != nil {
		txmgr.reset()
		return nil, err
	}
	return txstatsInfo, nil
}

//RemoveStaleAndCommitPvtDataOfOldBlocks implements method in interface `txmgmt.TxMgr`
//执行以下六项操作：
//（1）从传递的blockspvtdata构造唯一的pvt数据
//（2）获取oldblockcommit上的锁
//（3）通过比较[version，valuehash]检查过时的pvtdata并删除过时的数据
//（4）从非过时的pvtdata创建更新批
//（5）更新由清除管理器管理的BTL记账，并更新过期密钥。
//（6）将未过时的pvt数据提交到statedb
//此函数假定传递的输入仅包含
//标记为“有效”。在当前的设计中，当我们存储
//缺少有关仅有效事务的数据信息。此外，八卦只提供
//缺少有效事务的pvtdata。如果这两个假设由于某些错误而被破坏，
//从数据一致性的角度来看，我们仍然是安全的，因为我们匹配版本和
//值哈希在提交值之前存储在statedb中。但是，如果
//a tuple <ns, Coll, key> is passed for two (or more) transactions with one as valid and
//另一个是无效事务，如果
//无效Tx的版本大于有效Tx（根据我们在
//构造唯一的pvtData（）。除了bug之外，还有另一种情况
//函数可能会同时接收有效和无效Tx的pvtData。这样的情况将得到解释。
//在FAB-12924中，与状态叉和重建分类帐状态相关。
func (txmgr *LockBasedTxMgr) RemoveStaleAndCommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error {
//（0）在validateAndPrepare（）、PrepareExpiringKeys（）和
//removestaleandcommitpvttataofoldBlocks（），我们只能允许一个
//一次执行的函数。原因是每个函数调用
//loadcommittedversions（），它将清除
//临时缓冲区和加载新条目（这样的临时缓冲区不是
//适用于goleveldb）。因此，这三个函数可以
//交错并取消批量读取API提供的优化。
//一旦引入并存在分类账缓存（FAB-103）
//loadcommittedversions（）被重构以返回映射，我们可以允许
//这三个函数是并行执行的。但是，我们不能移除
//oldblockcommit上的锁，因为它还用于避免交错
//在commit（）和执行此函数之间，以确保正确性。
	logger.Debug("Waiting for purge mgr to finish the background job of computing expirying keys for the block")
	txmgr.pvtdataPurgeMgr.WaitForPrepareToFinish()
	txmgr.oldBlockCommit.Lock()
	defer txmgr.oldBlockCommit.Unlock()
	logger.Debug("lock acquired on oldBlockCommit for committing pvtData of old blocks to state database")

//（1）由于blockSPVTData可以包含用于
//a给定的<ns，coll，key>，我们需要找到具有不同
//版本和使用更高版本的版本
	logger.Debug("Constructing unique pvtData by removing duplicate entries")
	uniquePvtData, err := constructUniquePvtData(blocksPvtData)
	if len(uniquePvtData) == 0 || err != nil {
		return err
	}

//（3）删除与哈希不匹配的pvt数据
//存储在公共状态中的值
	logger.Debug("Finding and removing stale pvtData")
	if err := uniquePvtData.findAndRemoveStalePvtData(txmgr.db); err != nil {
		return err
	}

//（4）从uniquepvtdata创建更新批
	batch := uniquePvtData.transformToUpdateBatch()

//（5）更新purge manager中的簿记并更新topurgelist
//（即失效密钥列表）。就像过期的钥匙
//在提交的最后一个PrepareExpiringKeys期间构造，我们需要
//更新列表。这是因为删除块的vestale和commitpvtdata
//可能添加了新的数据，这些数据可能在
//下一个常规块提交。
	logger.Debug("Updating bookkeeping info in the purge manager")
	if err := txmgr.pvtdataPurgeMgr.UpdateBookkeepingForPvtDataOfOldBlocks(batch.PvtUpdates); err != nil {
		return err
	}

//（6）将PVT数据提交到STATEDB
	logger.Debug("Committing updates to state database")
	if err := txmgr.db.ApplyPrivacyAwareUpdates(batch, nil); err != nil {
		return err
	}
	return nil
}

type uniquePvtDataMap map[privacyenabledstate.HashedCompositeKey]*privacyenabledstate.PvtKVWrite

func constructUniquePvtData(blocksPvtData map[uint64][]*ledger.TxPvtData) (uniquePvtDataMap, error) {
	uniquePvtData := make(uniquePvtDataMap)
//浏览blockspvtdata以查找重复的<ns，coll，key>
//in the pvtWrites and use the one with the higher version number
	for blkNum, blockPvtData := range blocksPvtData {
		if err := uniquePvtData.updateUsingBlockPvtData(blockPvtData, blkNum); err != nil {
			return nil, err
		}
} //对于每个块
	return uniquePvtData, nil
}

func (uniquePvtData uniquePvtDataMap) updateUsingBlockPvtData(blockPvtData []*ledger.TxPvtData, blkNum uint64) error {
	for _, txPvtData := range blockPvtData {
		ver := version.NewHeight(blkNum, txPvtData.SeqInBlock)
		if err := uniquePvtData.updateUsingTxPvtData(txPvtData, ver); err != nil {
			return err
		}
} //对于每个TX
	return nil
}
func (uniquePvtData uniquePvtDataMap) updateUsingTxPvtData(txPvtData *ledger.TxPvtData, ver *version.Height) error {
	for _, nsPvtData := range txPvtData.WriteSet.NsPvtRwset {
		if err := uniquePvtData.updateUsingNsPvtData(nsPvtData, ver); err != nil {
			return err
		}
} //对于每个ns
	return nil
}
func (uniquePvtData uniquePvtDataMap) updateUsingNsPvtData(nsPvtData *rwset.NsPvtReadWriteSet, ver *version.Height) error {
	for _, collPvtData := range nsPvtData.CollectionPvtRwset {
		if err := uniquePvtData.updateUsingCollPvtData(collPvtData, nsPvtData.Namespace, ver); err != nil {
			return err
		}
} //为每个科尔
	return nil
}

func (uniquePvtData uniquePvtDataMap) updateUsingCollPvtData(collPvtData *rwset.CollectionPvtReadWriteSet,
	ns string, ver *version.Height) error {

	kvRWSet := &kvrwset.KVRWSet{}
	if err := proto.Unmarshal(collPvtData.Rwset, kvRWSet); err != nil {
		return err
	}

	hashedCompositeKey := privacyenabledstate.HashedCompositeKey{
		Namespace:      ns,
		CollectionName: collPvtData.CollectionName,
	}

for _, kvWrite := range kvRWSet.Writes { //每对千伏
		hashedCompositeKey.KeyHash = string(util.ComputeStringHash(kvWrite.Key))
		uniquePvtData.updateUsingPvtWrite(kvWrite, hashedCompositeKey, ver)
} //每对千伏

	return nil
}

func (uniquePvtData uniquePvtDataMap) updateUsingPvtWrite(pvtWrite *kvrwset.KVWrite,
	hashedCompositeKey privacyenabledstate.HashedCompositeKey, ver *version.Height) {

	pvtData, ok := uniquePvtData[hashedCompositeKey]
	if !ok || pvtData.Version.Compare(ver) < 0 {
		uniquePvtData[hashedCompositeKey] =
			&privacyenabledstate.PvtKVWrite{
				Key:      pvtWrite.Key,
				IsDelete: pvtWrite.IsDelete,
				Value:    pvtWrite.Value,
				Version:  ver,
			}
	}
}

func (uniquePvtData uniquePvtDataMap) findAndRemoveStalePvtData(db privacyenabledstate.DB) error {
//（1）加载所有提交的版本
	if err := uniquePvtData.loadCommittedVersionIntoCache(db); err != nil {
		return err
	}

//（2）查找并删除陈旧数据
	for hashedCompositeKey, pvtWrite := range uniquePvtData {
		isStale, err := checkIfPvtWriteIsStale(&hashedCompositeKey, pvtWrite, db)
		if err != nil {
			return err
		}
		if isStale {
			delete(uniquePvtData, hashedCompositeKey)
		}
	}
	return nil
}

func (uniquePvtData uniquePvtDataMap) loadCommittedVersionIntoCache(db privacyenabledstate.DB) error {
//注意，在我们验证并提交这些文件之前，不会调用ClearCachedVersions。
//旧区块的PVT数据。这是因为只有在独占锁定期间，我们才
//清除缓存，我们在到达这里之前已经获得了一个缓存。
	var hashedCompositeKeys []*privacyenabledstate.HashedCompositeKey
	for hashedCompositeKey := range uniquePvtData {
//tempKey ensures a different pointer is added to the slice for each key
		tempKey := hashedCompositeKey
		hashedCompositeKeys = append(hashedCompositeKeys, &tempKey)
	}

	err := db.LoadCommittedVersionsOfPubAndHashedKeys(nil, hashedCompositeKeys)
	if err != nil {
		return err
	}
	return nil
}

func checkIfPvtWriteIsStale(hashedKey *privacyenabledstate.HashedCompositeKey,
	kvWrite *privacyenabledstate.PvtKVWrite, db privacyenabledstate.DB) (bool, error) {

	ns := hashedKey.Namespace
	coll := hashedKey.CollectionName
	keyHashBytes := []byte(hashedKey.KeyHash)
	committedVersion, err := db.GetKeyHashVersion(ns, coll, keyHashBytes)
	if err != nil {
		return true, err
	}

//对于已删除的hashedkey，我们将得到一个nil提交版本。注意
//哈希键已被删除，因为它已过期或被删除
//链码本身。
	if committedVersion == nil && kvWrite.IsDelete {
		return false, nil
	}

 /*
  待办事项：FAB-12922
  在第一轮中，我们需要检查传递的pvtdata的版本
  与存储在statedb中的pvtdata版本相对应。在第二轮比赛中，
  对于剩余的pvtdata，我们需要使用hashed检查是否有过时
  版本。在第三轮，对于剩下的pvtdata，我们需要
  检查哈希值。在每个阶段，我们都需要
  从statedb执行相关数据的批量加载。
  committedPvtData, err := db.GetPrivateData(ns, coll, kvWrite.Key)
  如果犯错！= nIL{
   返回false，err
  }
  如果是提交的PVTDATA。版本（比较）（KvWrord.Valuy）> 0 {
   返回假，零
  }
 **/

	if version.AreSame(committedVersion, kvWrite.Version) {
		return false, nil
	}

//由于元数据更新，我们可以得到一个版本
//pvt kv写入与提交的不匹配
//哈斯凯奇在这种情况下，我们必须比较哈希
//价值的。如果哈希匹配，我们应该更新
//pvt kv写入和返回中的版本号
//作为验证结果为真
	vv, err := db.GetValueHash(ns, coll, keyHashBytes)
	if err != nil {
		return true, err
	}
	if bytes.Equal(vv.Value, util.ComputeHash(kvWrite.Value)) {
//if hash of value matches, update version
//回归真实
kvWrite.Version = vv.Version //副作用
//（checkifpvtwriteisstale不应更新状态）
		return false, nil
	}
	return true, nil
}

func (uniquePvtData uniquePvtDataMap) transformToUpdateBatch() *privacyenabledstate.UpdateBatch {
	batch := privacyenabledstate.NewUpdateBatch()
	for hashedCompositeKey, pvtWrite := range uniquePvtData {
		ns := hashedCompositeKey.Namespace
		coll := hashedCompositeKey.CollectionName
		if pvtWrite.IsDelete {
			batch.PvtUpdates.Delete(ns, coll, pvtWrite.Key, pvtWrite.Version)
		} else {
			batch.PvtUpdates.Put(ns, coll, pvtWrite.Key, pvtWrite.Value, pvtWrite.Version)
		}
	}
	return batch
}

func (txmgr *LockBasedTxMgr) invokeNamespaceListeners() error {
	for _, listener := range txmgr.stateListeners {
		stateUpdatesForListener := extractStateUpdates(txmgr.current.batch, listener.InterestedInNamespaces())
		if len(stateUpdatesForListener) == 0 {
			continue
		}
		txmgr.current.listeners = append(txmgr.current.listeners, listener)

		committedStateQueryExecuter := &queryutil.QECombiner{
			QueryExecuters: []queryutil.QueryExecuter{txmgr.db}}

		postCommitQueryExecuter := &queryutil.QECombiner{
			QueryExecuters: []queryutil.QueryExecuter{
				&queryutil.UpdateBatchBackedQueryExecuter{UpdateBatch: txmgr.current.batch.PubUpdates.UpdateBatch},
				txmgr.db,
			},
		}

		trigger := &ledger.StateUpdateTrigger{
			LedgerID:                    txmgr.ledgerid,
			StateUpdates:                stateUpdatesForListener,
			CommittingBlockNum:          txmgr.current.blockNum(),
			CommittedStateQueryExecutor: committedStateQueryExecuter,
			PostCommitQueryExecutor:     postCommitQueryExecuter,
		}
		if err := listener.HandleStateUpdates(trigger); err != nil {
			return err
		}
		logger.Debugf("Invoking listener for state changes:%s", listener)
	}
	return nil
}

//shutdown在接口“txmgmt.txmgr”中实现方法
func (txmgr *LockBasedTxMgr) Shutdown() {
//等待后台执行例程完成，否则计时问题会导致goleveldb代码中出现nil指针。
//见FAB-11974
	txmgr.pvtdataPurgeMgr.WaitForPrepareToFinish()
	txmgr.db.Close()
}

//commit在接口“txmgmt.txmgr”中实现方法
func (txmgr *LockBasedTxMgr) Commit() error {
//我们需要在oldblockcommit上获取一个锁。以下是两个原因：
//（1）如果
//TopurgeList由removestaleandcommitpvtdataofoldblocks（）更新。
//（2）removestale和commitpvttataofoldblocks计算更新
//基于当前状态的批处理，如果我们允许在
//same time, the former may overwrite the newer versions of the data and we may
//以错误的更新批结束。
	txmgr.oldBlockCommit.Lock()
	defer txmgr.oldBlockCommit.Unlock()
	logger.Debug("lock acquired on oldBlockCommit for committing regular updates to state database")

//在对等机启动后为第一个块提交使用清除管理器时，异步函数
//'PrepareForExpiringKeys' is invoked in-line. However, for the subsequent blocks commits, this function is invoked
//在下一个街区之前
	if !txmgr.pvtdataPurgeMgr.usedOnce {
		txmgr.pvtdataPurgeMgr.PrepareForExpiringKeys(txmgr.current.blockNum())
		txmgr.pvtdataPurgeMgr.usedOnce = true
	}
	defer func() {
		txmgr.pvtdataPurgeMgr.PrepareForExpiringKeys(txmgr.current.blockNum() + 1)
		logger.Debugf("launched the background routine for preparing keys to purge with the next block")
		txmgr.reset()
	}()

	logger.Debugf("Committing updates to state database")
	if txmgr.current == nil {
		panic("validateAndPrepare() method should have been called before calling commit()")
	}

	if err := txmgr.pvtdataPurgeMgr.DeleteExpiredAndUpdateBookkeeping(
		txmgr.current.batch.PvtUpdates, txmgr.current.batch.HashUpdates); err != nil {
		return err
	}

	commitHeight := version.NewHeight(txmgr.current.blockNum(), txmgr.current.maxTxNumber())
	txmgr.commitRWLock.Lock()
	logger.Debugf("Write lock acquired for committing updates to state database")
	if err := txmgr.db.ApplyPrivacyAwareUpdates(txmgr.current.batch, commitHeight); err != nil {
		txmgr.commitRWLock.Unlock()
		return err
	}
	txmgr.commitRWLock.Unlock()
//只有在对oldblockcommit持有锁的情况下，我们才应该将缓存清除为
//旧的PVTDATA提交器使用缓存来加载版本
//哈氏钥匙。另外，请注意PrepareForExpiringKeys使用缓存。
	txmgr.clearCache()
	logger.Debugf("Updates committed to state database and the write lock is released")

//purge manager should be called (in this call the purge mgr removes the expiry entries from schedules) after committing to statedb
	if err := txmgr.pvtdataPurgeMgr.BlockCommitDone(); err != nil {
		return err
	}
//如果出现错误状态，侦听器将不会接收此调用，而是在接收时由分类帐引起对等机恐慌。
//此函数出错
	txmgr.updateStateListeners()
	return nil
}

//rollback在接口“txmgmt.txmgr”中实现方法
func (txmgr *LockBasedTxMgr) Rollback() {
	txmgr.reset()
}

//ClearCache清空由StateDB实现维护的缓存
func (txmgr *LockBasedTxMgr) clearCache() {
	if txmgr.db.IsBulkOptimizable() {
		txmgr.db.ClearCachedVersions()
	}
}

//shouldRecover在接口kvledger.recoverer中实现方法
func (txmgr *LockBasedTxMgr) ShouldRecover(lastAvailableBlock uint64) (bool, uint64, error) {
	savepoint, err := txmgr.GetLastSavepoint()
	if err != nil {
		return false, 0, err
	}
	if savepoint == nil {
		return true, 0, nil
	}
	return savepoint.BlockNum != lastAvailableBlock, savepoint.BlockNum + 1, nil
}

//commitListBlock在接口kvledger.recoverer中实现方法
func (txmgr *LockBasedTxMgr) CommitLostBlock(blockAndPvtdata *ledger.BlockAndPvtData) error {
	block := blockAndPvtdata.Block
	logger.Debugf("Constructing updateSet for the block %d", block.Header.Number)
	if _, err := txmgr.ValidateAndPrepare(blockAndPvtdata, false); err != nil {
		return err
	}

//在信息级别记录每1000个块，以便在生产环境中跟踪statedb重建进度。
	if block.Header.Number%1000 == 0 {
		logger.Infof("Recommitting block [%d] to state database", block.Header.Number)
	} else {
		logger.Debugf("Recommitting block [%d] to state database", block.Header.Number)
	}

	return txmgr.Commit()
}

func extractStateUpdates(batch *privacyenabledstate.UpdateBatch, namespaces []string) ledger.StateUpdates {
	stateupdates := make(ledger.StateUpdates)
	for _, namespace := range namespaces {
		updatesMap := batch.PubUpdates.GetUpdates(namespace)
		var kvwrites []*kvrwset.KVWrite
		for key, versionedValue := range updatesMap {
			kvwrites = append(kvwrites, &kvrwset.KVWrite{Key: key, IsDelete: versionedValue.Value == nil, Value: versionedValue.Value})
			if len(kvwrites) > 0 {
				stateupdates[namespace] = kvwrites
			}
		}
	}
	return stateupdates
}

func (txmgr *LockBasedTxMgr) updateStateListeners() {
	for _, l := range txmgr.current.listeners {
		l.StateCommitDone(txmgr.ledgerid)
	}
}

func (txmgr *LockBasedTxMgr) reset() {
	txmgr.current = nil
}

//PVTDATAPUGEMEGR封装了实际吹扫管理器和附加标志“使用一次”。
//有关此附加标志的用法，请参阅上面txmgr.commit（）函数中的相关注释。
type pvtdataPurgeMgr struct {
	pvtstatepurgemgmt.PurgeMgr
	usedOnce bool
}
