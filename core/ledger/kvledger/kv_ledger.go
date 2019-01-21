
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


package kvledger

import (
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/core/ledger/confighistory"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/history/historydb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr/lockbasedtxmgr"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/ledgerstorage"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("kvledger")

//kvledger提供了“ledger.peerledger”的实现。
//此实现提供了一个基于关键值的数据模型
type kvLedger struct {
	ledgerID               string
	blockStore             *ledgerstorage.Store
	txtmgmt                txmgr.TxMgr
	historyDB              historydb.HistoryDB
	configHistoryRetriever ledger.ConfigHistoryRetriever
	blockAPIsRWLock        *sync.RWMutex
	stats                  *ledgerStats
}

//new kvledger构造新的“kvledger”
func newKVLedger(
	ledgerID string,
	blockStore *ledgerstorage.Store,
	versionedDB privacyenabledstate.DB,
	historyDB historydb.HistoryDB,
	configHistoryMgr confighistory.Mgr,
	stateListeners []ledger.StateListener,
	bookkeeperProvider bookkeeping.Provider,
	ccInfoProvider ledger.DeployedChaincodeInfoProvider,
	stats *ledgerStats,
) (*kvLedger, error) {
	logger.Debugf("Creating KVLedger ledgerID=%s: ", ledgerID)
//为此链/分类帐创建一个kvledger，它封装了基础
//id store、blockstore、txmgr（状态数据库）、history数据库
	l := &kvLedger{ledgerID: ledgerID, blockStore: blockStore, historyDB: historyDB, blockAPIsRWLock: &sync.RWMutex{}}

//TODO将函数“getchaincodeventlistener”移动到分类帐界面，然后
//此功能将事件注册到Ledgermgmt包，以便
//在其他未来的分类帐实现中重复使用
	ccEventListener := versionedDB.GetChaincodeEventListener()
	logger.Debugf("Register state db for chaincode lifecycle events: %t", ccEventListener != nil)
	if ccEventListener != nil {
		cceventmgmt.GetMgr().Register(ledgerID, ccEventListener)
	}
	btlPolicy := pvtdatapolicy.ConstructBTLPolicy(&collectionInfoRetriever{l, ccInfoProvider})
	if err := l.initTxMgr(versionedDB, stateListeners, btlPolicy, bookkeeperProvider, ccInfoProvider); err != nil {
		return nil, err
	}
	l.initBlockStore(btlPolicy)
//如果状态数据库和历史数据库与块存储不同步，则恢复它们
	if err := l.recoverDBs(); err != nil {
		panic(errors.WithMessage(err, "error during state DB recovery"))
	}
	l.configHistoryRetriever = configHistoryMgr.GetRetriever(ledgerID, l)

	info, err := l.GetBlockchainInfo()
	if err != nil {
		return nil, err
	}
//用当前高度初始化stat
	stats.updateBlockchainHeight(info.Height)
	l.stats = stats
	return l, nil
}

func (l *kvLedger) initTxMgr(versionedDB privacyenabledstate.DB, stateListeners []ledger.StateListener,
	btlPolicy pvtdatapolicy.BTLPolicy, bookkeeperProvider bookkeeping.Provider, ccInfoProvider ledger.DeployedChaincodeInfoProvider) error {
	var err error
	l.txtmgmt, err = lockbasedtxmgr.NewLockBasedTxMgr(l.ledgerID, versionedDB, stateListeners, btlPolicy, bookkeeperProvider, ccInfoProvider)
	return err
}

func (l *kvLedger) initBlockStore(btlPolicy pvtdatapolicy.BTLPolicy) {
	l.blockStore.Init(btlPolicy)
}

//恢复状态数据库和历史数据库（如果存在）
//通过重新提交最后一个有效块
func (l *kvLedger) recoverDBs() error {
	logger.Debugf("Entering recoverDB()")
	if err := l.syncStateAndHistoryDBWithBlockstore(); err != nil {
		return err
	}
	if err := l.syncStateDBWithPvtdatastore(); err != nil {
		return err
	}
	return nil
}

func (l *kvLedger) syncStateAndHistoryDBWithBlockstore() error {
//如果blockstorage中没有块，则没有要恢复的块。
	info, _ := l.blockStore.GetBlockchainInfo()
	if info.Height == 0 {
		logger.Debug("Block storage is empty.")
		return nil
	}
	lastAvailableBlockNum := info.Height - 1
	recoverables := []recoverable{l.txtmgmt, l.historyDB}
	recoverers := []*recoverer{}
	for _, recoverable := range recoverables {
		recoverFlag, firstBlockNum, err := recoverable.ShouldRecover(lastAvailableBlockNum)
		if err != nil {
			return err
		}
		if recoverFlag {
			recoverers = append(recoverers, &recoverer{firstBlockNum, recoverable})
		}
	}
	if len(recoverers) == 0 {
		return nil
	}
	if len(recoverers) == 1 {
		return l.recommitLostBlocks(recoverers[0].firstBlockNum, lastAvailableBlockNum, recoverers[0].recoverable)
	}

//两个DBS都需要恢复
	if recoverers[0].firstBlockNum > recoverers[1].firstBlockNum {
//交换（将lagger db设为0索引）
		recoverers[0], recoverers[1] = recoverers[1], recoverers[0]
	}
	if recoverers[0].firstBlockNum != recoverers[1].firstBlockNum {
//使拉格朗日分贝等于另一分贝
		if err := l.recommitLostBlocks(recoverers[0].firstBlockNum, recoverers[1].firstBlockNum-1,
			recoverers[0].recoverable); err != nil {
			return err
		}
	}
//获取数据库到块存储
	return l.recommitLostBlocks(recoverers[1].firstBlockNum, lastAvailableBlockNum,
		recoverers[0].recoverable, recoverers[1].recoverable)
}

func (l *kvLedger) syncStateDBWithPvtdatastore() error {
//托多：到目前为止，设计理念是块存储的范围是
//仅限于存储和检索具有一定保证的块数据，statedb是
//为了国家管理。上层“kvledger”协调
//二。但是，在保持块的消耗状态（即，
//用于pvtstore调节的lastupdatedodldblocklist）在专用数据块存储中
//打破了这个假设。知道为了这个目的消耗了哪些模块
//状态更新不应与源（即pvtdata存储）有关。潜在的固定
//在FAB-12731中提到
	blocksPvtData, err := l.blockStore.GetLastUpdatedOldBlocksPvtData()
	if err != nil {
		return err
	}
	if err := l.txtmgmt.RemoveStaleAndCommitPvtDataOfOldBlocks(blocksPvtData); err != nil {
		return err
	}
	if err := l.blockStore.ResetLastUpdatedOldBlocksList(); err != nil {
		return err
	}

	return nil
}

//RecommitListBlocks检索指定范围内的块，并将写入集提交给
//状态数据库或历史数据库或两者
func (l *kvLedger) recommitLostBlocks(firstBlockNum uint64, lastBlockNum uint64, recoverables ...recoverable) error {
	logger.Infof("Recommitting lost blocks - firstBlockNum=%d, lastBlockNum=%d, recoverables=%#v", firstBlockNum, lastBlockNum, recoverables)
	var err error
	var blockAndPvtdata *ledger.BlockAndPvtData
	for blockNumber := firstBlockNum; blockNumber <= lastBlockNum; blockNumber++ {
		if blockAndPvtdata, err = l.GetPvtDataAndBlockByNum(blockNumber, nil); err != nil {
			return err
		}
		for _, r := range recoverables {
			if err := r.CommitLostBlock(blockAndPvtdata); err != nil {
				return err
			}
		}
	}
	logger.Infof("Recommitted lost blocks - firstBlockNum=%d, lastBlockNum=%d, recoverables=%#v", firstBlockNum, lastBlockNum, recoverables)
	return nil
}

//GetTransactionByID按ID检索事务
func (l *kvLedger) GetTransactionByID(txID string) (*peer.ProcessedTransaction, error) {
	tranEnv, err := l.blockStore.RetrieveTxByID(txID)
	if err != nil {
		return nil, err
	}
	txVResult, err := l.blockStore.RetrieveTxValidationCodeByTxID(txID)
	if err != nil {
		return nil, err
	}
	processedTran := &peer.ProcessedTransaction{TransactionEnvelope: tranEnv, ValidationCode: int32(txVResult)}
	l.blockAPIsRWLock.RLock()
	l.blockAPIsRWLock.RUnlock()
	return processedTran, nil
}

//GetBlockChainInfo返回区块链的基本信息
func (l *kvLedger) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	bcInfo, err := l.blockStore.GetBlockchainInfo()
	l.blockAPIsRWLock.RLock()
	defer l.blockAPIsRWLock.RUnlock()
	return bcInfo, err
}

//GetBlockByNumber返回给定高度的块
//math.maxuint64的blocknumber将返回最后一个块
func (l *kvLedger) GetBlockByNumber(blockNumber uint64) (*common.Block, error) {
	block, err := l.blockStore.RetrieveBlockByNumber(blockNumber)
	l.blockAPIsRWLock.RLock()
	l.blockAPIsRWLock.RUnlock()
	return block, err
}

//GetBlocksiterator返回一个以“startBlockNumber”（含）开头的迭代器。
//迭代器是一个阻塞迭代器，也就是说，它阻塞直到下一个块在分类帐中可用为止。
//resultsiterator包含类型blockholder
func (l *kvLedger) GetBlocksIterator(startBlockNumber uint64) (commonledger.ResultsIterator, error) {
	blkItr, err := l.blockStore.RetrieveBlocks(startBlockNumber)
	if err != nil {
		return nil, err
	}
	return &blocksItr{l.blockAPIsRWLock, blkItr}, nil
}

//GetBlockByHash返回给定哈希的块
func (l *kvLedger) GetBlockByHash(blockHash []byte) (*common.Block, error) {
	block, err := l.blockStore.RetrieveBlockByHash(blockHash)
	l.blockAPIsRWLock.RLock()
	l.blockAPIsRWLock.RUnlock()
	return block, err
}

//getblockbytxid返回包含事务的块
func (l *kvLedger) GetBlockByTxID(txID string) (*common.Block, error) {
	block, err := l.blockStore.RetrieveBlockByTxID(txID)
	l.blockAPIsRWLock.RLock()
	l.blockAPIsRWLock.RUnlock()
	return block, err
}

func (l *kvLedger) GetTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	txValidationCode, err := l.blockStore.RetrieveTxValidationCodeByTxID(txID)
	l.blockAPIsRWLock.RLock()
	l.blockAPIsRWLock.RUnlock()
	return txValidationCode, err
}

//prune删除满足给定策略的块/事务
func (l *kvLedger) Prune(policy commonledger.PrunePolicy) error {
	return errors.New("not yet implemented")
}

//newtxtsimulator返回new`ledger.txtsimulator`
func (l *kvLedger) NewTxSimulator(txid string) (ledger.TxSimulator, error) {
	return l.txtmgmt.NewTxSimulator(txid)
}

//NewQueryExecutor向查询执行器提供句柄。
//客户端可以获取多个“queryexecutor”以进行并行执行。
//如果需要，应在实现级别执行任何同步
func (l *kvLedger) NewQueryExecutor() (ledger.QueryExecutor, error) {
	return l.txtmgmt.NewQueryExecutor(util.GenerateUUID())
}

//newHistoryQueryExecutor为历史查询执行器提供句柄。
//客户端可以获取多个“HistoryQueryExecutor”以进行并行执行。
//如果需要，应在实现级别执行任何同步
//传递分类帐块库，以便可以从链中查找历史值
func (l *kvLedger) NewHistoryQueryExecutor() (ledger.HistoryQueryExecutor, error) {
	return l.historyDB.NewHistoryQueryExecutor(l.blockStore)
}

//commitWithpvtData在原子操作中提交块和相应的pvt数据
func (l *kvLedger) CommitWithPvtData(pvtdataAndBlock *ledger.BlockAndPvtData) error {
	var err error
	block := pvtdataAndBlock.Block
	blockNo := pvtdataAndBlock.Block.Header.Number

	startBlockProcessing := time.Now()
	logger.Debugf("[%s] Validating state for block [%d]", l.ledgerID, blockNo)
	txstatsInfo, err := l.txtmgmt.ValidateAndPrepare(pvtdataAndBlock, true)
	if err != nil {
		return err
	}
	elapsedBlockProcessing := time.Since(startBlockProcessing)

	startCommitBlockStorage := time.Now()
	logger.Debugf("[%s] Committing block [%d] to storage", l.ledgerID, blockNo)
	l.blockAPIsRWLock.Lock()
	defer l.blockAPIsRWLock.Unlock()
	if err = l.blockStore.CommitWithPvtData(pvtdataAndBlock); err != nil {
		return err
	}
	elapsedCommitBlockStorage := time.Since(startCommitBlockStorage)

	startCommitState := time.Now()
	logger.Debugf("[%s] Committing block [%d] transactions to state database", l.ledgerID, blockNo)
	if err = l.txtmgmt.Commit(); err != nil {
		panic(errors.WithMessage(err, "error during commit to txmgr"))
	}
	elapsedCommitState := time.Since(startCommitState)

//历史数据库可以与状态和/或异步并行写入，作为将来的优化，
//虽然它不是一个瓶颈……但不需要在经过的时间内使日志混乱。
	if ledgerconfig.IsHistoryDBEnabled() {
		logger.Debugf("[%s] Committing block [%d] transactions to history database", l.ledgerID, blockNo)
		if err := l.historyDB.Commit(block); err != nil {
			panic(errors.WithMessage(err, "Error during commit to history db"))
		}
	}

	elapsedCommitWithPvtData := time.Since(startBlockProcessing)

	logger.Infof("[%s] Committed block [%d] with %d transaction(s) in %dms (state_validation=%dms block_commit=%dms state_commit=%dms)",
		l.ledgerID, block.Header.Number, len(block.Data.Data),
		elapsedCommitWithPvtData/time.Millisecond,
		elapsedBlockProcessing/time.Millisecond,
		elapsedCommitBlockStorage/time.Millisecond,
		elapsedCommitState/time.Millisecond,
	)
	l.updateBlockStats(blockNo,
		elapsedBlockProcessing,
		elapsedCommitBlockStorage,
		elapsedCommitState,
		txstatsInfo,
	)
	return nil
}

func (l *kvLedger) updateBlockStats(
	blockNum uint64,
	blockProcessingTime time.Duration,
	blockstorageCommitTime time.Duration,
	statedbCommitTime time.Duration,
	txstatsInfo []*txmgr.TxStatInfo,
) {
	l.stats.updateBlockchainHeight(blockNum + 1)
	l.stats.updateBlockProcessingTime(blockProcessingTime)
	l.stats.updateBlockstorageCommitTime(blockstorageCommitTime)
	l.stats.updateStatedbCommitTime(statedbCommitTime)
	l.stats.updateTransactionsStats(txstatsInfo)
}

//GetMissingPvtDataInfoFormsToRecentBlocks返回缺少的
//最新的'maxblock'块，它至少丢失符合条件的集合的私有数据。
func (l *kvLedger) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
	return l.blockStore.GetMissingPvtDataInfoForMostRecentBlocks(maxBlock)
}

//getpvtdataandblockbynum返回块和相应的pvt数据。
//pvt数据由提供的“集合”列表筛选
func (l *kvLedger) GetPvtDataAndBlockByNum(blockNum uint64, filter ledger.PvtNsCollFilter) (*ledger.BlockAndPvtData, error) {
	blockAndPvtdata, err := l.blockStore.GetPvtDataAndBlockByNum(blockNum, filter)
	l.blockAPIsRWLock.RLock()
	l.blockAPIsRWLock.RUnlock()
	return blockAndPvtdata, err
}

//getpvtdatabynum仅返回与给定块号对应的pvt数据
//pvt数据由提供的“集合”列表筛选
func (l *kvLedger) GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	pvtdata, err := l.blockStore.GetPvtDataByNum(blockNum, filter)
	l.blockAPIsRWLock.RLock()
	l.blockAPIsRWLock.RUnlock()
	return pvtdata, err
}

//清除在块高度小于
//给定的MaxBlockNumtoretain。换句话说，purge只保留私有读写集
//在MaxBlockNumtoretain或更高的块高度生成。
func (l *kvLedger) PurgePrivateData(maxBlockNumToRetain uint64) error {
	return errors.New("not yet implemented")
}

//privatedataminblocknum返回保留的最低认可块高度
func (l *kvLedger) PrivateDataMinBlockNum() (uint64, error) {
	return 0, errors.New("not yet implemented")
}

func (l *kvLedger) GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error) {
	return l.configHistoryRetriever, nil
}

func (l *kvLedger) CommitPvtDataOfOldBlocks(pvtData []*ledger.BlockPvtData) ([]*ledger.PvtdataHashMismatch, error) {
	logger.Debugf("[%s:] Comparing pvtData of [%d] old blocks against the hashes in transaction's rwset to find valid and invalid data",
		l.ledgerID, len(pvtData))
	validPvtData, hashMismatches, err := ConstructValidAndInvalidPvtData(pvtData, l.blockStore)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s:] Committing pvtData of [%d] old blocks to the pvtdatastore", l.ledgerID, len(pvtData))
	err = l.blockStore.CommitPvtDataOfOldBlocks(validPvtData)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s:] Committing pvtData of [%d] old blocks to the stateDB", l.ledgerID, len(pvtData))
	err = l.txtmgmt.RemoveStaleAndCommitPvtDataOfOldBlocks(validPvtData)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s:] Clearing the bookkeeping information from pvtdatastore", l.ledgerID)
	if err := l.blockStore.ResetLastUpdatedOldBlocksList(); err != nil {
		return nil, err
	}

	return hashMismatches, nil
}

func (l *kvLedger) GetMissingPvtDataTracker() (ledger.MissingPvtDataTracker, error) {
	return l, nil
}

//关闭关闭“kvledger”
func (l *kvLedger) Close() {
	l.blockStore.Shutdown()
	l.txtmgmt.Shutdown()
}

type blocksItr struct {
	blockAPIsRWLock *sync.RWMutex
	blocksItr       commonledger.ResultsIterator
}

func (itr *blocksItr) Next() (commonledger.QueryResult, error) {
	block, err := itr.blocksItr.Next()
	if err != nil {
		return nil, err
	}
	itr.blockAPIsRWLock.RLock()
	itr.blockAPIsRWLock.RUnlock()
	return block, nil
}

func (itr *blocksItr) Close() {
	itr.blocksItr.Close()
}

type collectionInfoRetriever struct {
	ledger       ledger.PeerLedger
	infoProvider ledger.DeployedChaincodeInfoProvider
}

func (r *collectionInfoRetriever) CollectionInfo(chaincodeName, collectionName string) (*common.StaticCollectionConfig, error) {
	qe, err := r.ledger.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()
	return r.infoProvider.CollectionInfo(chaincodeName, collectionName, qe)
}
