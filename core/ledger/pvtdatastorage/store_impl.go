
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
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/willf/bitset"
)

var logger = flogging.MustGetLogger("pvtdatastorage")

type provider struct {
	dbProvider *leveldbhelper.Provider
}

type store struct {
	db        *leveldbhelper.DBHandle
	ledgerid  string
	btlPolicy pvtdatapolicy.BTLPolicy

	isEmpty            bool
	lastCommittedBlock uint64
	batchPending       bool
	purgerLock         sync.Mutex
	collElgProcSync    *collElgProcSync
//在提交旧块的pvtdata之后，
//'islastupdatedodldblocksset'设置为true。
//一旦使用这些pvtdata更新statedb，
//'islastupdatedodldblocksset'设置为false。
//islastupdatedodldblocksset主要用于
//恢复过程。在对等机启动期间，如果
//IsLastUpdatedDoldBlocksSet设置为true，pvtData
//在状态数据库中，需要在完成
//恢复操作。
	isLastUpdatedOldBlocksSet bool
}

type blkTranNumKey []byte

type dataEntry struct {
	key   *dataKey
	value *rwset.CollectionPvtReadWriteSet
}

type expiryEntry struct {
	key   *expiryKey
	value *ExpiryData
}

type expiryKey struct {
	expiringBlk   uint64
	committingBlk uint64
}

type nsCollBlk struct {
	ns, coll string
	blkNum   uint64
}

type dataKey struct {
	nsCollBlk
	txNum uint64
}

type missingDataKey struct {
	nsCollBlk
	isEligible bool
}

type storeEntries struct {
	dataEntries        []*dataEntry
	expiryEntries      []*expiryEntry
	missingDataEntries map[missingDataKey]*bitset.BitSet
}

//lastupdatedodldblockslist保留上次更新的块的列表
//并存储为lastupdatedodldblockskey的值（以kv_encoding.go定义）
type lastUpdatedOldBlocksList []uint64

type entriesForPvtDataOfOldBlocks struct {
//对于每个<ns，coll，blknum，txnum>，存储数据条目，即pvtdata
	dataEntries map[dataKey]*rwset.CollectionPvtReadWriteSet
//将检索到的（&updated）expiryData存储在expiryEntries中
	expiryEntries map[expiryKey]*ExpiryData
//对于每个<ns，coll，blknum>，将检索的（更新的）位图存储在MissingDataEntries中
	missingDataEntries map[nsCollBlk]*bitset.BitSet
}

///////provider函数
///////////////////////////////////

//NewProvider实例化StoreProvider
func NewProvider() Provider {
	dbPath := ledgerconfig.GetPvtdataStorePath()
	dbProvider := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})
	return &provider{dbProvider: dbProvider}
}

//openstore将句柄返回到存储
func (p *provider) OpenStore(ledgerid string) (Store, error) {
	dbHandle := p.dbProvider.GetDBHandle(ledgerid)
	s := &store{db: dbHandle, ledgerid: ledgerid,
		collElgProcSync: &collElgProcSync{
			notification: make(chan bool, 1),
			procComplete: make(chan bool, 1),
		},
	}
	if err := s.initState(); err != nil {
		return nil, err
	}
	s.launchCollElgProc()
	logger.Debugf("Pvtdata store opened. Initial state: isEmpty [%t], lastCommittedBlock [%d], batchPending [%t]",
		s.isEmpty, s.lastCommittedBlock, s.batchPending)
	return s, nil
}

//关闭关闭商店
func (p *provider) Close() {
	p.dbProvider.Close()
}

///////store函数
///////////////////////////////////

func (s *store) initState() error {
	var err error
	var blist lastUpdatedOldBlocksList
	if s.isEmpty, s.lastCommittedBlock, err = s.getLastCommittedBlockNum(); err != nil {
		return err
	}
	if s.batchPending, err = s.hasPendingCommit(); err != nil {
		return err
	}
	if blist, err = s.getLastUpdatedOldBlocksList(); err != nil {
		return err
	}
	if len(blist) > 0 {
		s.isLastUpdatedOldBlocksSet = true
} //如果未设置为false

	return nil
}

func (s *store) Init(btlPolicy pvtdatapolicy.BTLPolicy) {
	s.btlPolicy = btlPolicy
}

//Prepare实现接口“store”中的函数
func (s *store) Prepare(blockNum uint64, pvtData []*ledger.TxPvtData, missingPvtData ledger.TxMissingPvtDataMap) error {
	if s.batchPending {
		return &ErrIllegalCall{`A pending batch exists as as result of last invoke to "Prepare" call.
			 Invoke "Commit" or "Rollback" on the pending batch before invoking "Prepare" function`}
	}
	expectedBlockNum := s.nextBlockNum()
	if expectedBlockNum != blockNum {
		return &ErrIllegalArgs{fmt.Sprintf("Expected block number=%d, recived block number=%d", expectedBlockNum, blockNum)}
	}

	batch := leveldbhelper.NewUpdateBatch()
	var err error
	var keyBytes, valBytes []byte

	storeEntries, err := prepareStoreEntries(blockNum, pvtData, s.btlPolicy, missingPvtData)
	if err != nil {
		return err
	}

	for _, dataEntry := range storeEntries.dataEntries {
		keyBytes = encodeDataKey(dataEntry.key)
		if valBytes, err = encodeDataValue(dataEntry.value); err != nil {
			return err
		}
		batch.Put(keyBytes, valBytes)
	}

	for _, expiryEntry := range storeEntries.expiryEntries {
		keyBytes = encodeExpiryKey(expiryEntry.key)
		if valBytes, err = encodeExpiryValue(expiryEntry.value); err != nil {
			return err
		}
		batch.Put(keyBytes, valBytes)
	}

	for missingDataKey, missingDataValue := range storeEntries.missingDataEntries {
		keyBytes = encodeMissingDataKey(&missingDataKey)
		if valBytes, err = encodeMissingDataValue(missingDataValue); err != nil {
			return err
		}
		batch.Put(keyBytes, valBytes)
	}

	batch.Put(pendingCommitKey, emptyValue)
	if err := s.db.WriteBatch(batch, true); err != nil {
		return err
	}
	s.batchPending = true
	logger.Debugf("Saved %d private data write sets for block [%d]", len(pvtData), blockNum)
	return nil
}

//commit实现接口“store”中的函数
func (s *store) Commit() error {
	if !s.batchPending {
		return &ErrIllegalCall{"No pending batch to commit"}
	}
	committingBlockNum := s.nextBlockNum()
	logger.Debugf("Committing private data for block [%d]", committingBlockNum)
	batch := leveldbhelper.NewUpdateBatch()
	batch.Delete(pendingCommitKey)
	batch.Put(lastCommittedBlkkey, encodeLastCommittedBlockVal(committingBlockNum))
	if err := s.db.WriteBatch(batch, true); err != nil {
		return err
	}
	s.batchPending = false
	s.isEmpty = false
	s.lastCommittedBlock = committingBlockNum
	logger.Debugf("Committed private data for block [%d]", committingBlockNum)
	s.performPurgeIfScheduled(committingBlockNum)
	return nil
}

//rollback实现接口“store”中的函数
//这将删除现有数据条目和符合条件的丢失数据条目。
//但是，这不会在下次尝试时删除不合格的丢失数据实体。
//将有完全相同的条目，并将覆盖这些条目。这也使得
//现有的失效实体是因为，它们很可能也会被覆盖
//每个新数据条目。即使某些过期条目没有被覆盖，
//（因为某些数据下次可能会丢失），额外的到期条目只是
//诺普
func (s *store) Rollback() error {
	if !s.batchPending {
		return &ErrIllegalCall{"No pending batch to rollback"}
	}
	blkNum := s.nextBlockNum()
	batch := leveldbhelper.NewUpdateBatch()
	itr := s.db.GetIterator(datakeyRange(blkNum))
	for itr.Next() {
		batch.Delete(itr.Key())
	}
	itr.Release()
	itr = s.db.GetIterator(eligibleMissingdatakeyRange(blkNum))
	for itr.Next() {
		batch.Delete(itr.Key())
	}
	itr.Release()
	batch.Delete(pendingCommitKey)
	if err := s.db.WriteBatch(batch, true); err != nil {
		return err
	}
	s.batchPending = false
	return nil
}

//commitpvtdataofoldblocks提交旧块的pvtdata（即以前丢失的数据）。
//参数'block s pvtdata'引用pvtstore中缺少的旧块的pvtdata列表。
//给定旧块的pvtdata列表，“commitpvtdataofoldblocks”将执行以下四项操作
//操作
//（1）为所有pvtdata构建数据条目
//（2）构建更新条目（即，数据条目、ExpiryEntries、MissingDataEntries，以及
//最后更新的doldblockslist）来自上面创建的数据项
//（3）从更新条目创建数据库更新批处理
//（4）将更新条目提交到pvtstore
func (s *store) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error {
	if s.isLastUpdatedOldBlocksSet {
		return &ErrIllegalCall{`The lastUpdatedOldBlocksList is set. It means that the
		stateDB may not be in sync with the pvtStore`}
	}

//（1）为所有pvtdata构建数据条目
	dataEntries := constructDataEntriesFromBlocksPvtData(blocksPvtData)

//（2）从上面创建的数据条目构造更新条目（即，数据条目、ExpireyEntries、MissingDataEntries）。
	logger.Debugf("Constructing pvtdatastore entries for pvtData of [%d] old blocks", len(blocksPvtData))
	updateEntries, err := s.constructUpdateEntriesFromDataEntries(dataEntries)
	if err != nil {
		return err
	}

//（3）从更新条目创建数据库更新批处理
	logger.Debug("Constructing update batch from pvtdatastore entries")
	batch, err := constructUpdateBatchFromUpdateEntries(updateEntries)
	if err != nil {
		return err
	}

//（4）将更新条目提交到pvtstore
	logger.Debug("Committing the update batch to pvtdatastore")
	if err := s.commitBatch(batch); err != nil {
		return err
	}
	s.isLastUpdatedOldBlocksSet = true

	return nil
}

func constructDataEntriesFromBlocksPvtData(blocksPvtData map[uint64][]*ledger.TxPvtData) []*dataEntry {
//为所有pvtdata构造数据项
	var dataEntries []*dataEntry
	for blkNum, pvtData := range blocksPvtData {
//准备pvtdata的数据项
		dataEntries = append(dataEntries, prepareDataEntries(blkNum, pvtData)...)
	}
	return dataEntries
}

func (s *store) constructUpdateEntriesFromDataEntries(dataEntries []*dataEntry) (*entriesForPvtDataOfOldBlocks, error) {
	updateEntries := &entriesForPvtDataOfOldBlocks{
		dataEntries:        make(map[dataKey]*rwset.CollectionPvtReadWriteSet),
		expiryEntries:      make(map[expiryKey]*ExpiryData),
		missingDataEntries: make(map[nsCollBlk]*bitset.BitSet)}

//for each data entry, first, get the expiryData and missingData from the pvtStore.
//然后，根据数据输入更新expireydata和missingdata。最后，添加
//数据项以及更新的ExpiryData和将数据发送到更新项
	for _, dataEntry := range dataEntries {
//获取expireyblk编号以构造expireykey
		expiryKey, err := s.constructExpiryKeyFromDataEntry(dataEntry)
		if err != nil {
			return nil, err
		}

//获取现有ExpiryData ntry
		var expiryData *ExpiryData
		if !neverExpires(expiryKey.expiringBlk) {
			if expiryData, err = s.getExpiryDataFromUpdateEntriesOrStore(updateEntries, expiryKey); err != nil {
				return nil, err
			}
			if expiryData == nil {
//数据输入已过期
//和净化（一种罕见的情况）
				continue
			}
		}

//获取现有的MissingData条目
		var missingData *bitset.BitSet
		nsCollBlk := dataEntry.key.nsCollBlk
		if missingData, err = s.getMissingDataFromUpdateEntriesOrStore(updateEntries, nsCollBlk); err != nil {
			return nil, err
		}
		if missingData == nil {
//数据输入已过期
//和净化（一种罕见的情况）
			continue
		}

		updateEntries.addDataEntry(dataEntry)
if expiryData != nil { //对于永不过期的条目将是nill
			expiryEntry := &expiryEntry{&expiryKey, expiryData}
			updateEntries.updateAndAddExpiryEntry(expiryEntry, dataEntry.key)
		}
		updateEntries.updateAndAddMissingDataEntry(missingData, dataEntry.key)
	}
	return updateEntries, nil
}

func (s *store) constructExpiryKeyFromDataEntry(dataEntry *dataEntry) (expiryKey, error) {
//获取expireyblk编号以构造expireykey
	nsCollBlk := dataEntry.key.nsCollBlk
	expiringBlk, err := s.btlPolicy.GetExpiringBlock(nsCollBlk.ns, nsCollBlk.coll, nsCollBlk.blkNum)
	if err != nil {
		return expiryKey{}, err
	}
	return expiryKey{expiringBlk, nsCollBlk.blkNum}, nil
}

func (s *store) getExpiryDataFromUpdateEntriesOrStore(updateEntries *entriesForPvtDataOfOldBlocks, expiryKey expiryKey) (*ExpiryData, error) {
	expiryData, ok := updateEntries.expiryEntries[expiryKey]
	if !ok {
		var err error
		expiryData, err = s.getExpiryDataOfExpiryKey(&expiryKey)
		if err != nil {
			return nil, err
		}
	}
	return expiryData, nil
}

func (s *store) getMissingDataFromUpdateEntriesOrStore(updateEntries *entriesForPvtDataOfOldBlocks, nsCollBlk nsCollBlk) (*bitset.BitSet, error) {
	missingData, ok := updateEntries.missingDataEntries[nsCollBlk]
	if !ok {
		var err error
		missingDataKey := &missingDataKey{nsCollBlk, true}
		missingData, err = s.getBitmapOfMissingDataKey(missingDataKey)
		if err != nil {
			return nil, err
		}
	}
	return missingData, nil
}

func (updateEntries *entriesForPvtDataOfOldBlocks) addDataEntry(dataEntry *dataEntry) {
	dataKey := dataKey{dataEntry.key.nsCollBlk, dataEntry.key.txNum}
	updateEntries.dataEntries[dataKey] = dataEntry.value
}

func (updateEntries *entriesForPvtDataOfOldBlocks) updateAndAddExpiryEntry(expiryEntry *expiryEntry, dataKey *dataKey) {
	txNum := dataKey.txNum
	nsCollBlk := dataKey.nsCollBlk
//更新
	expiryEntry.value.addPresentData(nsCollBlk.ns, nsCollBlk.coll, txNum)
//无法从MissingDatamap中删除条目，因为
//每个丢失的<ns col>
//不考虑txnum的数目。

//添加
	expiryKey := expiryKey{expiryEntry.key.expiringBlk, expiryEntry.key.committingBlk}
	updateEntries.expiryEntries[expiryKey] = expiryEntry.value
}

func (updateEntries *entriesForPvtDataOfOldBlocks) updateAndAddMissingDataEntry(missingData *bitset.BitSet, dataKey *dataKey) {

	txNum := dataKey.txNum
	nsCollBlk := dataKey.nsCollBlk
//更新
	missingData.Clear(uint(txNum))
//添加
	updateEntries.missingDataEntries[nsCollBlk] = missingData
}

func constructUpdateBatchFromUpdateEntries(updateEntries *entriesForPvtDataOfOldBlocks) (*leveldbhelper.UpdateBatch, error) {
	batch := leveldbhelper.NewUpdateBatch()

//将以下四种类型的条目添加到更新批处理中：（1）新数据条目
//（即，PVTDATA），（2）更新过期条目，（3）更新缺失数据条目，以及
//（4）更新的阻止列表

//（1）向批次添加新数据条目
	if err := addNewDataEntriesToUpdateBatch(batch, updateEntries); err != nil {
		return nil, err
	}

//（2）将更新的expiryEntry添加到批处理中
	if err := addUpdatedExpiryEntriesToUpdateBatch(batch, updateEntries); err != nil {
		return nil, err
	}

//（3）将更新的MissingData添加到批处理中
	if err := addUpdatedMissingDataEntriesToUpdateBatch(batch, updateEntries); err != nil {
		return nil, err
	}

//（4）将lastupdatedDoldBlocksList添加到批处理中
	addLastUpdatedOldBlocksList(batch, updateEntries)

	return batch, nil
}

func addNewDataEntriesToUpdateBatch(batch *leveldbhelper.UpdateBatch, entries *entriesForPvtDataOfOldBlocks) error {
	var keyBytes, valBytes []byte
	var err error
	for dataKey, pvtData := range entries.dataEntries {
		keyBytes = encodeDataKey(&dataKey)
		if valBytes, err = encodeDataValue(pvtData); err != nil {
			return err
		}
		batch.Put(keyBytes, valBytes)
	}
	return nil
}

func addUpdatedExpiryEntriesToUpdateBatch(batch *leveldbhelper.UpdateBatch, entries *entriesForPvtDataOfOldBlocks) error {
	var keyBytes, valBytes []byte
	var err error
	for expiryKey, expiryData := range entries.expiryEntries {
		keyBytes = encodeExpiryKey(&expiryKey)
		if valBytes, err = encodeExpiryValue(expiryData); err != nil {
			return err
		}
		batch.Put(keyBytes, valBytes)
	}
	return nil
}

func addUpdatedMissingDataEntriesToUpdateBatch(batch *leveldbhelper.UpdateBatch, entries *entriesForPvtDataOfOldBlocks) error {
	var keyBytes, valBytes []byte
	var err error
	for nsCollBlk, missingData := range entries.missingDataEntries {
		keyBytes = encodeMissingDataKey(&missingDataKey{nsCollBlk, true})
//如果MissingData为空，则需要删除MissingDataKey
		if missingData.None() {
			batch.Delete(keyBytes)
			continue
		}
		if valBytes, err = encodeMissingDataValue(missingData); err != nil {
			return err
		}
		batch.Put(keyBytes, valBytes)
	}
	return nil
}

func addLastUpdatedOldBlocksList(batch *leveldbhelper.UpdateBatch, entries *entriesForPvtDataOfOldBlocks) {
//创建正在存储的块的pvtdata列表。如果这个列表是
//在恢复过程中发现，statedb可能与pvtdata不同步
//需要恢复。在正常流中，一旦StateDB同步，则
//阻止列表将被删除。
	updatedBlksListMap := make(map[uint64]bool)

	for dataKey := range entries.dataEntries {
		updatedBlksListMap[dataKey.blkNum] = true
	}

	var updatedBlksList lastUpdatedOldBlocksList
	for blkNum := range updatedBlksListMap {
		updatedBlksList = append(updatedBlksList, blkNum)
	}

//最好存储为排序列表
	sort.SliceStable(updatedBlksList, func(i, j int) bool {
		return updatedBlksList[i] < updatedBlksList[j]
	})

	buf := proto.NewBuffer(nil)
	buf.EncodeVarint(uint64(len(updatedBlksList)))
	for _, blkNum := range updatedBlksList {
		buf.EncodeVarint(blkNum)
	}

	batch.Put(lastUpdatedOldBlocksKey, buf.Bytes())
}

func (s *store) commitBatch(batch *leveldbhelper.UpdateBatch) error {
//将批提交到存储区
	if err := s.db.WriteBatch(batch, true); err != nil {
		return err
	}

	return nil
}

//getlastupdatedodldblockspvtdata实现接口“store”中的函数
func (s *store) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	if !s.isLastUpdatedOldBlocksSet {
		return nil, nil
	}

	updatedBlksList, err := s.getLastUpdatedOldBlocksList()
	if err != nil {
		return nil, err
	}

	blksPvtData := make(map[uint64][]*ledger.TxPvtData)
	for _, blkNum := range updatedBlksList {
		if blksPvtData[blkNum], err = s.GetPvtDataByBlockNum(blkNum, nil); err != nil {
			return nil, err
		}
	}
	return blksPvtData, nil
}

func (s *store) getLastUpdatedOldBlocksList() ([]uint64, error) {
	var v []byte
	var err error
	if v, err = s.db.Get(lastUpdatedOldBlocksKey); err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	var updatedBlksList []uint64
	buf := proto.NewBuffer(v)
	numBlks, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(numBlks); i++ {
		blkNum, err := buf.DecodeVarint()
		if err != nil {
			return nil, err
		}
		updatedBlksList = append(updatedBlksList, blkNum)
	}
	return updatedBlksList, nil
}

//ReStLaStudiDeDeloBobsStList实现接口“存储”中的函数
func (s *store) ResetLastUpdatedOldBlocksList() error {
	batch := leveldbhelper.NewUpdateBatch()
	batch.Delete(lastUpdatedOldBlocksKey)
	if err := s.db.WriteBatch(batch, true); err != nil {
		return err
	}
	s.isLastUpdatedOldBlocksSet = false
	return nil
}

//getpvtdatabyblocknum实现接口“store”中的函数。
//如果存储为空或最后提交的块号较小，则
//请求的块编号，引发了“erroutoFrange”
func (s *store) GetPvtDataByBlockNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	logger.Debugf("Get private data for block [%d], filter=%#v", blockNum, filter)
	if s.isEmpty {
		return nil, &ErrOutOfRange{"The store is empty"}
	}
	if blockNum > s.lastCommittedBlock {
		return nil, &ErrOutOfRange{fmt.Sprintf("Last committed block=%d, block requested=%d", s.lastCommittedBlock, blockNum)}
	}
	startKey, endKey := getDataKeysForRangeScanByBlockNum(blockNum)
	logger.Debugf("Querying private data storage for write sets using startKey=%#v, endKey=%#v", startKey, endKey)
	itr := s.db.GetIterator(startKey, endKey)
	defer itr.Release()

	var blockPvtdata []*ledger.TxPvtData
	var currentTxNum uint64
	var currentTxWsetAssember *txPvtdataAssembler
	firstItr := true

	for itr.Next() {
		dataKeyBytes := itr.Key()
		if v11Format(dataKeyBytes) {
			return v11RetrievePvtdata(itr, filter)
		}
		dataValueBytes := itr.Value()
		dataKey := decodeDatakey(dataKeyBytes)
		expired, err := isExpired(dataKey.nsCollBlk, s.btlPolicy, s.lastCommittedBlock)
		if err != nil {
			return nil, err
		}
		if expired || !passesFilter(dataKey, filter) {
			continue
		}
		dataValue, err := decodeDataValue(dataValueBytes)
		if err != nil {
			return nil, err
		}

		if firstItr {
			currentTxNum = dataKey.txNum
			currentTxWsetAssember = newTxPvtdataAssembler(blockNum, currentTxNum)
			firstItr = false
		}

		if dataKey.txNum != currentTxNum {
			blockPvtdata = append(blockPvtdata, currentTxWsetAssember.getTxPvtdata())
			currentTxNum = dataKey.txNum
			currentTxWsetAssember = newTxPvtdataAssembler(blockNum, currentTxNum)
		}
		currentTxWsetAssember.add(dataKey.ns, dataValue)
	}
	if currentTxWsetAssember != nil {
		blockPvtdata = append(blockPvtdata, currentTxWsetAssember.getTxPvtdata())
	}
	return blockPvtdata, nil
}

//initlastcommittedblock实现接口“store”中的函数
func (s *store) InitLastCommittedBlock(blockNum uint64) error {
	if !(s.isEmpty && !s.batchPending) {
		return &ErrIllegalCall{"The private data store is not empty. InitLastCommittedBlock() function call is not allowed"}
	}
	batch := leveldbhelper.NewUpdateBatch()
	batch.Put(lastCommittedBlkkey, encodeLastCommittedBlockVal(blockNum))
	if err := s.db.WriteBatch(batch, true); err != nil {
		return err
	}
	s.isEmpty = false
	s.lastCommittedBlock = blockNum
	logger.Debugf("InitLastCommittedBlock set to block [%d]", blockNum)
	return nil
}

//getMissingPvtDataInfoFormsToRecentBlocks实现接口“store”中的函数
func (s *store) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
//我们假设只有在处理完
//上次检索到缺少的pvtdata信息并提交相同的信息。
	if maxBlock < 1 {
		return nil, nil
	}

	missingPvtDataInfo := make(ledger.MissingPvtDataInfo)
	numberOfBlockProcessed := 0
	lastProcessedBlock := uint64(0)
	isMaxBlockLimitReached := false
//由于我们不获取读取锁，因此可以在
//构造MissingPvtDataInfo。因此，lastcommittedblock可以
//改变。为了确保一致性，我们自动加载lastcommittedblock值
	lastCommittedBlock := atomic.LoadUint64(&s.lastCommittedBlock)

	startKey, endKey := createRangeScanKeysForEligibleMissingDataEntries(lastCommittedBlock)
	dbItr := s.db.GetIterator(startKey, endKey)
	defer dbItr.Release()

	for dbItr.Next() {
		missingDataKeyBytes := dbItr.Key()
		missingDataKey := decodeMissingDataKey(missingDataKeyBytes)

		if isMaxBlockLimitReached && (missingDataKey.blkNum != lastProcessedBlock) {
//esnure就是maxblock数
//处理块的条目数
			break
		}

//检查条目是否过期。如果是，转到下一项。
//因为我们可以使用旧的lastcommittedblock值，所以有可能
//丢失的数据实际上已过期，但我们可能会得到过时的信息。
//虽然它可能会导致额外的工作来提取过期的数据，但它不会
//影响正确性。此外，当我们试图取回最近丢失的
//数据（现在到期的可能性较小），这种情况很少见。在
//最好的情况是，我们可以在这里自动加载最新的lastcommittedblock值
//使这种情况非常罕见。
		lastCommittedBlock = atomic.LoadUint64(&s.lastCommittedBlock)
		expired, err := isExpired(missingDataKey.nsCollBlk, s.btlPolicy, lastCommittedBlock)
		if err != nil {
			return nil, err
		}
		if expired {
			continue
		}

//检查MissingPvtDataInfo中的blknum的现有条目。
//如果不存在这样的条目，请创建一个。同时，跟踪
//由于MaxBlock限制，已处理块。
		if _, ok := missingPvtDataInfo[missingDataKey.blkNum]; !ok {
			numberOfBlockProcessed++
			if numberOfBlockProcessed == maxBlock {
				isMaxBlockLimitReached = true
//因为这个块可以有多个条目，
//我们不能在这里“中断”
				lastProcessedBlock = missingDataKey.blkNum
			}
		}

		valueBytes := dbItr.Value()
		bitmap, err := decodeMissingDataValue(valueBytes)
		if err != nil {
			return nil, err
		}

//对于错过私有数据的每个事务，在MissingBlockPvtDataInfo中进行输入
		for index, isSet := bitmap.NextSet(0); isSet; index, isSet = bitmap.NextSet(index + 1) {
			txNum := uint64(index)
			missingPvtDataInfo.Add(missingDataKey.blkNum, txNum, missingDataKey.ns, missingDataKey.coll)
		}
	}

	return missingPvtDataInfo, nil
}

//processcollseligibilityEnabled实现接口“store”中的函数
func (s *store) ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error {
	key := encodeCollElgKey(committingBlk)
	m := newCollElgInfo(nsCollMap)
	val, err := encodeCollElgVal(m)
	if err != nil {
		return err
	}
	batch := leveldbhelper.NewUpdateBatch()
	batch.Put(key, val)
	if err = s.db.WriteBatch(batch, true); err != nil {
		return err
	}
	s.collElgProcSync.notify()
	return nil
}

func (s *store) performPurgeIfScheduled(latestCommittedBlk uint64) {
	if latestCommittedBlk%ledgerconfig.GetPvtdataStorePurgeInterval() != 0 {
		return
	}
	go func() {
		s.purgerLock.Lock()
		logger.Debugf("Purger started: Purging expired private data till block number [%d]", latestCommittedBlk)
		defer s.purgerLock.Unlock()
		err := s.purgeExpiredData(0, latestCommittedBlk)
		if err != nil {
			logger.Warningf("Could not purge data from pvtdata store:%s", err)
		}
		logger.Debug("Purger finished")
	}()
}

func (s *store) purgeExpiredData(minBlkNum, maxBlkNum uint64) error {
	batch := leveldbhelper.NewUpdateBatch()
	expiryEntries, err := s.retrieveExpiryEntries(minBlkNum, maxBlkNum)
	if err != nil || len(expiryEntries) == 0 {
		return err
	}
	for _, expiryEntry := range expiryEntries {
//如果函数retrieveExpiryEntries也返回编码的到期键，则可以保存此编码。
//但是，为了更好的可读性而保留它
		batch.Delete(encodeExpiryKey(expiryEntry.key))
		dataKeys, missingDataKeys := deriveKeys(expiryEntry)
		for _, dataKey := range dataKeys {
			batch.Delete(encodeDataKey(dataKey))
		}
		for _, missingDataKey := range missingDataKeys {
			batch.Delete(encodeMissingDataKey(missingDataKey))
		}
		s.db.WriteBatch(batch, false)
	}
	logger.Infof("[%s] - [%d] Entries purged from private data storage till block number [%d]", s.ledgerid, len(expiryEntries), maxBlkNum)
	return nil
}

func (s *store) retrieveExpiryEntries(minBlkNum, maxBlkNum uint64) ([]*expiryEntry, error) {
	startKey, endKey := getExpiryKeysForRangeScan(minBlkNum, maxBlkNum)
	logger.Debugf("retrieveExpiryEntries(): startKey=%#v, endKey=%#v", startKey, endKey)
	itr := s.db.GetIterator(startKey, endKey)
	defer itr.Release()

	var expiryEntries []*expiryEntry
	for itr.Next() {
		expiryKeyBytes := itr.Key()
		expiryValueBytes := itr.Value()
		expiryKey := decodeExpiryKey(expiryKeyBytes)
		expiryValue, err := decodeExpiryValue(expiryValueBytes)
		if err != nil {
			return nil, err
		}
		expiryEntries = append(expiryEntries, &expiryEntry{key: expiryKey, value: expiryValue})
	}
	return expiryEntries, nil
}

func (s *store) launchCollElgProc() {
	maxBatchSize := ledgerconfig.GetPvtdataStoreCollElgProcMaxDbBatchSize()
	batchesInterval := ledgerconfig.GetPvtdataStoreCollElgProcDbBatchesInterval()
	go func() {
s.processCollElgEvents(maxBatchSize, batchesInterval) //打开存储时处理收集合格性事件-以防上次运行中有未处理的事件
		for {
			logger.Debugf("Waiting for collection eligibility event")
			s.collElgProcSync.waitForNotification()
			s.processCollElgEvents(maxBatchSize, batchesInterval)
			s.collElgProcSync.done()
		}
	}()
}

func (s *store) processCollElgEvents(maxBatchSize, batchesInterval int) {
	logger.Debugf("Starting to process collection eligibility events")
	s.purgerLock.Lock()
	defer s.purgerLock.Unlock()
	collElgStartKey, collElgEndKey := createRangeScanKeysForCollElg()
	eventItr := s.db.GetIterator(collElgStartKey, collElgEndKey)
	defer eventItr.Release()
	batch := leveldbhelper.NewUpdateBatch()
	totalEntriesConverted := 0

	for eventItr.Next() {
		collElgKey, collElgVal := eventItr.Key(), eventItr.Value()
		blkNum := decodeCollElgKey(collElgKey)
		CollElgInfo, err := decodeCollElgVal(collElgVal)
		logger.Debugf("Processing collection eligibility event [blkNum=%d], CollElgInfo=%s", blkNum, CollElgInfo)
		if err != nil {
			logger.Errorf("This error is not expected %s", err)
			continue
		}
		for ns, colls := range CollElgInfo.NsCollMap {
			var coll string
			for _, coll = range colls.Entries {
				logger.Infof("Converting missing data entries from ineligible to eligible for [ns=%s, coll=%s]", ns, coll)
				startKey, endKey := createRangeScanKeysForIneligibleMissingData(blkNum, ns, coll)
				collItr := s.db.GetIterator(startKey, endKey)
				collEntriesConverted := 0

for collItr.Next() { //每个条目
					originalKey, originalVal := collItr.Key(), collItr.Value()
					modifiedKey := decodeMissingDataKey(originalKey)
					modifiedKey.isEligible = true
					batch.Delete(originalKey)
					copyVal := make([]byte, len(originalVal))
					copy(copyVal, originalVal)
					batch.Put(encodeMissingDataKey(modifiedKey), copyVal)
					collEntriesConverted++
					if batch.Len() > maxBatchSize {
						s.db.WriteBatch(batch, true)
						batch = leveldbhelper.NewUpdateBatch()
						sleepTime := time.Duration(batchesInterval)
						logger.Infof("Going to sleep for %d milliseconds between batches. Entries for [ns=%s, coll=%s] converted so far = %d",
							sleepTime, ns, coll, collEntriesConverted)
						s.purgerLock.Unlock()
						time.Sleep(sleepTime * time.Millisecond)
						s.purgerLock.Lock()
					}
} //入口回路

				collItr.Release()
				logger.Infof("Converted all [%d] entries for [ns=%s, coll=%s]", collEntriesConverted, ns, coll)
				totalEntriesConverted += collEntriesConverted
} //科尔环
} //NS环
batch.Delete(collElgKey) //同时删除收集资格事件密钥
} //事件循环

	s.db.WriteBatch(batch, true)
	logger.Debugf("Converted [%d] inelligible mising data entries to elligible", totalEntriesConverted)
}

//lastcommittedblockheight实现接口“store”中的函数
func (s *store) LastCommittedBlockHeight() (uint64, error) {
	if s.isEmpty {
		return 0, nil
	}
	return s.lastCommittedBlock + 1, nil
}

//haspendingbatch实现接口“store”中的函数
func (s *store) HasPendingBatch() (bool, error) {
	return s.batchPending, nil
}

//isEmpty实现接口“store”中的函数
func (s *store) IsEmpty() (bool, error) {
	return s.isEmpty, nil
}

//shutdown实现接口“store”中的函数
func (s *store) Shutdown() {
//什么也不做
}

func (s *store) nextBlockNum() uint64 {
	if s.isEmpty {
		return 0
	}
	return s.lastCommittedBlock + 1
}

func (s *store) hasPendingCommit() (bool, error) {
	var v []byte
	var err error
	if v, err = s.db.Get(pendingCommitKey); err != nil {
		return false, err
	}
	return v != nil, nil
}

func (s *store) getLastCommittedBlockNum() (bool, uint64, error) {
	var v []byte
	var err error
	if v, err = s.db.Get(lastCommittedBlkkey); v == nil || err != nil {
		return true, 0, err
	}
	return false, decodeLastCommittedBlockVal(v), nil
}

type collElgProcSync struct {
	notification, procComplete chan bool
}

func (sync *collElgProcSync) notify() {
	select {
	case sync.notification <- true:
		logger.Debugf("Signaled to collection eligibility processing routine")
default: //诺普
		logger.Debugf("Previous signal still pending. Skipping new signal")
	}
}

func (sync *collElgProcSync) waitForNotification() {
	<-sync.notification
}

func (sync *collElgProcSync) done() {
	select {
	case sync.procComplete <- true:
	default:
	}
}

func (sync *collElgProcSync) waitForDone() {
	<-sync.procComplete
}

func (s *store) getBitmapOfMissingDataKey(missingDataKey *missingDataKey) (*bitset.BitSet, error) {
	var v []byte
	var err error
	if v, err = s.db.Get(encodeMissingDataKey(missingDataKey)); err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return decodeMissingDataValue(v)
}

func (s *store) getExpiryDataOfExpiryKey(expiryKey *expiryKey) (*ExpiryData, error) {
	var v []byte
	var err error
	if v, err = s.db.Get(encodeExpiryKey(expiryKey)); err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return decodeExpiryValue(v)
}
