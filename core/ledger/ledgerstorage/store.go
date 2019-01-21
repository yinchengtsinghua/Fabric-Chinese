
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
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/blkstorage/fsblkstorage"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	lutil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ledgerstorage")

//提供程序封装两个提供程序1）块存储提供程序和2）和pvt数据存储提供程序
type Provider struct {
	blkStoreProvider     blkstorage.BlockStoreProvider
	pvtdataStoreProvider pvtdatastorage.Provider
}

//存储封装了两个存储1）块存储和pvt数据存储
type Store struct {
	blkstorage.BlockStore
	pvtdataStore pvtdatastorage.Store
	rwlock       *sync.RWMutex
}

//NewProvider将句柄返回给提供程序
func NewProvider() *Provider {
//初始化块存储
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

	pvtStoreProvider := pvtdatastorage.NewProvider()
	return &Provider{blockStoreProvider, pvtStoreProvider}
}

//打开打开打开商店
func (p *Provider) Open(ledgerid string) (*Store, error) {
	var blockStore blkstorage.BlockStore
	var pvtdataStore pvtdatastorage.Store
	var err error

	if blockStore, err = p.blkStoreProvider.OpenBlockStore(ledgerid); err != nil {
		return nil, err
	}
	if pvtdataStore, err = p.pvtdataStoreProvider.OpenStore(ledgerid); err != nil {
		return nil, err
	}
	store := &Store{blockStore, pvtdataStore, &sync.RWMutex{}}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
}

//关闭关闭提供程序
func (p *Provider) Close() {
	p.blkStoreProvider.Close()
	p.pvtdataStoreProvider.Close()
}

//init用基本配置初始化存储
func (s *Store) Init(btlPolicy pvtdatapolicy.BTLPolicy) {
	s.pvtdataStore.Init(btlPolicy)
}

//commitWithpvtData在原子操作中提交块和相应的pvt数据
func (s *Store) CommitWithPvtData(blockAndPvtdata *ledger.BlockAndPvtData) error {
	blockNum := blockAndPvtdata.Block.Header.Number
	s.rwlock.Lock()
	defer s.rwlock.Unlock()

	pvtBlkStoreHt, err := s.pvtdataStore.LastCommittedBlockHeight()
	if err != nil {
		return err
	}

	writtenToPvtStore := false
if pvtBlkStoreHt < blockNum+1 { //pvt数据存储健全性检查不允许重写pvt数据。
//当重新处理块（重新连接通道或重新获取最后几个块）时，
//跳过pvt数据提交到pvt data块存储
		logger.Debugf("Writing block [%d] to pvt block store", blockNum)
//由于分类帐已经验证了此块中的所有TXS，因此我们需要
//使用验证的信息仅提交有效Tx的pvtdata。
//托多：FAB-12924已经说过了，有一个角落的案子，我们
//需要考虑。如果在常规块提交期间发生状态分叉，
//我们有一种机制，可以放下所有的块，然后重新蚀刻块。
//并重新处理它们。按照目前的方式，我们只会放弃
//块文件（和相关工件），但我们不会删除/覆盖
//pvtdata存储-因为目前的假设是存储完整的数据
//（对于有效和无效的事务）。现在，我们必须允许空投
//以及pvt数据存储。然而，问题是，它在
//通道（与块文件不同）。
//不删除pvtdata存储的副作用是我们可能
//在pvtdatastore中有一些缺少的数据项
//打破了我们只存储有效Tx的pvtData的目标的事务。
//我们还可能错过有效事务的pvtdata。请注意
//StuteDB TXMGR中的RevVestAldand CdvpTdAdOfOLDBog（））仅期望
//有效事务的pvtdata。因此，有必要重建pvtdatastore
//与blockstore一起在pvtdatastore中只保留有效的tx数据。
		validTxPvtData, validTxMissingPvtData := constructValidTxPvtDataAndMissingData(blockAndPvtdata)
		if err := s.pvtdataStore.Prepare(blockAndPvtdata.Block.Header.Number, validTxPvtData, validTxMissingPvtData); err != nil {
			return err
		}
		writtenToPvtStore = true
	} else {
		logger.Debugf("Skipping writing block [%d] to pvt block store as the store height is [%d]", blockNum, pvtBlkStoreHt)
	}

	if err := s.AddBlock(blockAndPvtdata.Block); err != nil {
		s.pvtdataStore.Rollback()
		return err
	}

	if writtenToPvtStore {
		return s.pvtdataStore.Commit()
	}
	return nil
}

func constructValidTxPvtDataAndMissingData(blockAndPvtData *ledger.BlockAndPvtData) ([]*ledger.TxPvtData,
	ledger.TxMissingPvtDataMap) {

	var validTxPvtData []*ledger.TxPvtData
	validTxMissingPvtData := make(ledger.TxMissingPvtDataMap)

	txsFilter := lutil.TxValidationFlags(blockAndPvtData.Block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	numTxs := uint64(len(blockAndPvtData.Block.Data.Data))

//对于所有有效的Tx，构造pvtdata和缺少的pvtdata列表
	for txNum := uint64(0); txNum < numTxs; txNum++ {
		if txsFilter.IsInvalid(int(txNum)) {
			continue
		}

		if pvtdata, ok := blockAndPvtData.PvtData[txNum]; ok {
			validTxPvtData = append(validTxPvtData, pvtdata)
		}

		if missingPvtData, ok := blockAndPvtData.MissingPvtData[txNum]; ok {
			for _, missing := range missingPvtData {
				validTxMissingPvtData.Add(txNum, missing.Namespace,
					missing.Collection, missing.IsEligible)
			}
		}
	}
	return validTxPvtData, validTxMissingPvtData
}

//commitpvtdataofoldblocks提交旧块的pvtdata
func (s *Store) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error {
	err := s.pvtdataStore.CommitPvtDataOfOldBlocks(blocksPvtData)
	if err != nil {
		return err
	}
	return nil
}

//getpvtdataandblockbynum返回块和相应的pvt数据。
//pvt数据由提供的“集合”列表筛选
func (s *Store) GetPvtDataAndBlockByNum(blockNum uint64, filter ledger.PvtNsCollFilter) (*ledger.BlockAndPvtData, error) {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var block *common.Block
	var pvtdata []*ledger.TxPvtData
	var err error
	if block, err = s.RetrieveBlockByNumber(blockNum); err != nil {
		return nil, err
	}
	if pvtdata, err = s.getPvtDataByNumWithoutLock(blockNum, filter); err != nil {
		return nil, err
	}
	return &ledger.BlockAndPvtData{Block: block, PvtData: constructPvtdataMap(pvtdata)}, nil
}

//getpvtdatabynum仅返回与给定块号对应的pvt数据
//The pvt data is filtered by the list of 'ns/collections' supplied in the filter
//nil筛选器不筛选任何结果
func (s *Store) GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	return s.getPvtDataByNumWithoutLock(blockNum, filter)
}

//getpvtdatabynumwithoutlock仅返回与给定块号对应的pvt数据。
//此函数不获取readlock，在大多数情况下，调用方
//在's.rwlock'上具有读取锁
func (s *Store) getPvtDataByNumWithoutLock(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	var pvtdata []*ledger.TxPvtData
	var err error
	if pvtdata, err = s.pvtdataStore.GetPvtDataByBlockNum(blockNum, filter); err != nil {
		return nil, err
	}
	return pvtdata, nil
}

//GetMissingPvtDataInfoFormsToRecentBlocks调用底层PvTData存储上的函数
func (s *Store) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
//不获取s.rwlock上的读锁是安全的。如果没有锁，则
//由于新的块提交，LastCommittedBlock无法更改。因此，我们可能不会
//能够获取最新块的缺失数据信息。这个
//决定确保不影响常规块提交率。
	return s.pvtdataStore.GetMissingPvtDataInfoForMostRecentBlocks(maxBlock)
}

//processcollseligibilityEnabled调用底层pvtdata存储上的函数
func (s *Store) ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error {
	return s.pvtdataStore.ProcessCollsEligibilityEnabled(committingBlk, nsCollMap)
}

//GetLastUpdatedDoldBlockspvtData调用底层pvtData存储上的函数
func (s *Store) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	return s.pvtdataStore.GetLastUpdatedOldBlocksPvtData()
}

//resetlastupdatedodldblockslist调用底层pvtdata存储上的函数
func (s *Store) ResetLastUpdatedOldBlocksList() error {
	return s.pvtdataStore.ResetLastUpdatedOldBlocksList()
}

//init首先调用函数“initfromexistingBlockchain”
//为了检查pvtdata存储是否由于升级而存在
//来自1.0的对等，需要用现有的区块链更新。如果这是
//不是这样的，那么这个init将调用函数'syncpvtdatastorewithblockstore'
//遵循正常的路线
func (s *Store) init() error {
	var initialized bool
	var err error
	if initialized, err = s.initPvtdataStoreFromExistingBlockchain(); err != nil || initialized {
		return err
	}
	return s.syncPvtdataStoreWithBlockStore()
}

//initpvdtatastorefromexistingBlockchain更新pvdtata存储的初始状态
//如果现有的块存储区具有区块链且pvtdata存储区为空。
//当对等机从版本1.0升级时，预计会发生这种情况。
//存在一个使用1.0版本生成的现有区块链。
//在这种情况下，pvtdata存储会像
//已处理没有pvt数据的现有块。如果
//发现上述条件为真，并成功更新pvtdata存储
func (s *Store) initPvtdataStoreFromExistingBlockchain() (bool, error) {
	var bcInfo *common.BlockchainInfo
	var pvtdataStoreEmpty bool
	var err error

	if bcInfo, err = s.BlockStore.GetBlockchainInfo(); err != nil {
		return false, err
	}
	if pvtdataStoreEmpty, err = s.pvtdataStore.IsEmpty(); err != nil {
		return false, err
	}
	if pvtdataStoreEmpty && bcInfo.Height > 0 {
		if err = s.pvtdataStore.InitLastCommittedBlock(bcInfo.Height - 1); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

//syncpvdtatastorewithblockstore检查块存储和pvt数据存储是否同步
//当构造存储实例并将其移交使用时，将调用此函数。
//此检查是否存在挂起的批（可能来自以前的系统崩溃）
//未提交的pvt数据。如果存在挂起的批，则进行检查
//关联块是否在块存储中成功提交（崩溃前）
//或者没有。如果块已提交，则提交私有数据批处理
//否则，将回滚pvt数据批。
func (s *Store) syncPvtdataStoreWithBlockStore() error {
	var pendingPvtbatch bool
	var err error
	if pendingPvtbatch, err = s.pvtdataStore.HasPendingBatch(); err != nil {
		return err
	}
	if !pendingPvtbatch {
		return nil
	}
	var bcInfo *common.BlockchainInfo
	var pvtdataStoreHt uint64

	if bcInfo, err = s.GetBlockchainInfo(); err != nil {
		return err
	}
	if pvtdataStoreHt, err = s.pvtdataStore.LastCommittedBlockHeight(); err != nil {
		return err
	}

	if bcInfo.Height == pvtdataStoreHt {
		return s.pvtdataStore.Rollback()
	}

	if bcInfo.Height == pvtdataStoreHt+1 {
		return s.pvtdataStore.Commit()
	}

	return errors.Errorf("This is not expected. blockStoreHeight=%d, pvtdataStoreHeight=%d", bcInfo.Height, pvtdataStoreHt)
}

func constructPvtdataMap(pvtdata []*ledger.TxPvtData) map[uint64]*ledger.TxPvtData {
	if pvtdata == nil {
		return nil
	}
	m := make(map[uint64]*ledger.TxPvtData)
	for _, pvtdatum := range pvtdata {
		m[pvtdatum.SeqInBlock] = pvtdatum
	}
	return m
}
