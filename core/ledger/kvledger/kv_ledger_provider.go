
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
	"bytes"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/confighistory"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/history/historydb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/history/historydb/historyleveldb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/ledgerstorage"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
//如果具有给定ID的分类帐已存在，则由CreateLedger调用引发errledRidests
	ErrLedgerIDExists = errors.New("LedgerID already exists")
//如果具有给定ID的分类帐不存在，则OpenLedger调用将引发ErrUnexistingLedgerID。
	ErrNonExistingLedgerID = errors.New("LedgerID does not exist")
//如果具有给定ID的分类帐尚未打开，则由closeledger调用引发errlegornotopened。
	ErrLedgerNotOpened = errors.New("ledger is not opened yet")

	underConstructionLedgerKey = []byte("underConstructionLedgerKey")
	ledgerKeyPrefix            = []byte("l")
)

//提供程序实现接口Ledger.PeerledgerProvider
type Provider struct {
	idStore             *idStore
	ledgerStoreProvider *ledgerstorage.Provider
	vdbProvider         privacyenabledstate.DBProvider
	historydbProvider   historydb.HistoryDBProvider
	configHistoryMgr    confighistory.Mgr
	stateListeners      []ledger.StateListener
	bookkeepingProvider bookkeeping.Provider
	initializer         *ledger.Initializer
	collElgNotifier     *collElgNotifier
	stats               *stats
}

//NewProvider实例化新的提供程序。
//这不是线程安全的，假定是同步调用方
func NewProvider() (ledger.PeerLedgerProvider, error) {
	logger.Info("Initializing ledger provider")
//初始化ID存储（chainID/ledgerids的清单）
	idStore := openIDStore(ledgerconfig.GetLedgerProviderPath())
	ledgerStoreProvider := ledgerstorage.NewProvider()
//初始化历史数据库（按键对值的历史进行索引）
	historydbProvider := historyleveldb.NewHistoryDBProvider()
	logger.Info("ledger provider Initialized")
	provider := &Provider{idStore, ledgerStoreProvider,
		nil, historydbProvider, nil, nil, nil, nil, nil, nil}
	return provider, nil
}

//初始化从接口ledger.peerledgerprovider实现相应的方法
func (provider *Provider) Initialize(initializer *ledger.Initializer) error {
	var err error
	configHistoryMgr := confighistory.NewMgr(initializer.DeployedChaincodeInfoProvider)
	collElgNotifier := &collElgNotifier{
		initializer.DeployedChaincodeInfoProvider,
		initializer.MembershipInfoProvider,
		make(map[string]collElgListener),
	}
	stateListeners := initializer.StateListeners
	stateListeners = append(stateListeners, collElgNotifier)
	stateListeners = append(stateListeners, configHistoryMgr)

	provider.initializer = initializer
	provider.configHistoryMgr = configHistoryMgr
	provider.stateListeners = stateListeners
	provider.collElgNotifier = collElgNotifier
	provider.bookkeepingProvider = bookkeeping.NewProvider()
	provider.vdbProvider, err = privacyenabledstate.NewCommonStorageDBProvider(provider.bookkeepingProvider, initializer.MetricsProvider, initializer.HealthCheckRegistry)
	if err != nil {
		return err
	}
	provider.stats = newStats(initializer.MetricsProvider)
	provider.recoverUnderConstructionLedger()
	return nil
}

//创建从接口ledger.peerledgerprovider实现相应的方法
//此函数在执行与分类帐创建相关的任何操作之前设置正在构建的标志，以及
//在使用提交的Genesis块成功创建分类帐时，删除标记并将条目添加到
//创建分类帐列表（自动）。如果在这两者之间发生崩溃，“恢复到结构分类账”
//在声明提供程序可用之前调用函数
func (provider *Provider) Create(genesisBlock *common.Block) (ledger.PeerLedger, error) {
	ledgerID, err := utils.GetChainIDFromBlock(genesisBlock)
	if err != nil {
		return nil, err
	}
	exists, err := provider.idStore.ledgerIDExists(ledgerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrLedgerIDExists
	}
	if err = provider.idStore.setUnderConstructionFlag(ledgerID); err != nil {
		return nil, err
	}
	lgr, err := provider.openInternal(ledgerID)
	if err != nil {
		logger.Errorf("Error opening a new empty ledger. Unsetting under construction flag. Error: %+v", err)
		panicOnErr(provider.runCleanup(ledgerID), "Error running cleanup for ledger id [%s]", ledgerID)
		panicOnErr(provider.idStore.unsetUnderConstructionFlag(), "Error while unsetting under construction flag")
		return nil, err
	}
	if err := lgr.CommitWithPvtData(&ledger.BlockAndPvtData{
		Block: genesisBlock,
	}); err != nil {
		lgr.Close()
		return nil, err
	}
	panicOnErr(provider.idStore.createLedgerID(ledgerID, genesisBlock), "Error while marking ledger as created")
	return lgr, nil
}

//open从接口ledger.peerledgerprovider实现相应的方法
func (provider *Provider) Open(ledgerID string) (ledger.PeerLedger, error) {
	logger.Debugf("Open() opening kvledger: %s", ledgerID)
//检查ID存储以确保chainID/ledgeID存在
	exists, err := provider.idStore.ledgerIDExists(ledgerID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNonExistingLedgerID
	}
	return provider.openInternal(ledgerID)
}

func (provider *Provider) openInternal(ledgerID string) (ledger.PeerLedger, error) {
//获取连锁店/分类帐的区块商店
	blockStore, err := provider.ledgerStoreProvider.Open(ledgerID)
	if err != nil {
		return nil, err
	}
	provider.collElgNotifier.registerListener(ledgerID, blockStore)

//获取链/分类帐的版本数据库（状态数据库）
	vDB, err := provider.vdbProvider.GetDBHandle(ledgerID)
	if err != nil {
		return nil, err
	}

//获取链/分类帐的历史数据库（按键的值历史索引）
	historyDB, err := provider.historydbProvider.GetDBHandle(ledgerID)
	if err != nil {
		return nil, err
	}

//Create a kvLedger for this chain/ledger, which encasulates the underlying data stores
//（ID存储、块存储、状态数据库、历史数据库）
	l, err := newKVLedger(
		ledgerID, blockStore, vDB, historyDB, provider.configHistoryMgr,
		provider.stateListeners, provider.bookkeepingProvider,
		provider.initializer.DeployedChaincodeInfoProvider,
		provider.stats.ledgerStats(ledgerID),
	)
	if err != nil {
		return nil, err
	}
	return l, nil
}

//exists从接口ledger.peerledgerprovider实现相应的方法
func (provider *Provider) Exists(ledgerID string) (bool, error) {
	return provider.idStore.ledgerIDExists(ledgerID)
}

//列表从接口ledger.peerledgerprovider实现相应的方法
func (provider *Provider) List() ([]string, error) {
	return provider.idStore.getAllLedgerIds()
}

//close从接口ledger.peerledgerprovider实现相应的方法
func (provider *Provider) Close() {
	provider.idStore.close()
	provider.ledgerStoreProvider.Close()
	provider.vdbProvider.Close()
	provider.historydbProvider.Close()
	provider.bookkeepingProvider.Close()
	provider.configHistoryMgr.Close()
}

//recoverUnderConstructionLedger检查是否设置了“在建”标志-这种情况下
//如果在创建分类帐的过程中发生崩溃，则分类帐创建可能会保留在中间
//状态。恢复检查是否创建了分类帐并且成功提交了Genesis块，然后完成
//将分类帐ID添加到已创建分类帐列表的最后一步。否则，它将清除正在构建的标志
func (provider *Provider) recoverUnderConstructionLedger() {
	logger.Debugf("Recovering under construction ledger")
	ledgerID, err := provider.idStore.getUnderConstructionFlag()
	panicOnErr(err, "Error while checking whether the under construction flag is set")
	if ledgerID == "" {
		logger.Debugf("No under construction ledger found. Quitting recovery")
		return
	}
	logger.Infof("ledger [%s] found as under construction", ledgerID)
	ledger, err := provider.openInternal(ledgerID)
	panicOnErr(err, "Error while opening under construction ledger [%s]", ledgerID)
	bcInfo, err := ledger.GetBlockchainInfo()
	panicOnErr(err, "Error while getting blockchain info for the under construction ledger [%s]", ledgerID)
	ledger.Close()

	switch bcInfo.Height {
	case 0:
		logger.Infof("Genesis block was not committed. Hence, the peer ledger not created. unsetting the under construction flag")
		panicOnErr(provider.runCleanup(ledgerID), "Error while running cleanup for ledger id [%s]", ledgerID)
		panicOnErr(provider.idStore.unsetUnderConstructionFlag(), "Error while unsetting under construction flag")
	case 1:
		logger.Infof("Genesis block was committed. Hence, marking the peer ledger as created")
		genesisBlock, err := ledger.GetBlockByNumber(0)
		panicOnErr(err, "Error while retrieving genesis block from blockchain for ledger [%s]", ledgerID)
		panicOnErr(provider.idStore.createLedgerID(ledgerID, genesisBlock), "Error while adding ledgerID [%s] to created list", ledgerID)
	default:
		panic(errors.Errorf(
			"data inconsistency: under construction flag is set for ledger [%s] while the height of the blockchain is [%d]",
			ledgerID, bcInfo.Height))
	}
	return
}

//runcleanup清理blockstorage、statedb和historydb
//可能是在完整的分类帐创建过程中创建的
func (provider *Provider) runCleanup(ledgerID string) error {
//尽管如此，没有这是无害的千伏分类账。
//如果我们愿意，可以做到以下几点：
//-blockstorage可以删除空文件夹
//-couchdb backed statedb如果创建了数据库，则可以删除该数据库
//-支持leveldb的statedb和history db不需要执行任何操作，因为它使用跨分类账共享的单个db。
	return nil
}

func panicOnErr(err error, mgsFormat string, args ...interface{}) {
	if err == nil {
		return
	}
	args = append(args, err)
	panic(fmt.Sprintf(mgsFormat+" Error: %s", args...))
}

//////////////////////////////////////////////////
//分类帐ID持久性相关代码
////////////////////////////////////////////////
type idStore struct {
	db *leveldbhelper.DB
}

func openIDStore(path string) *idStore {
	db := leveldbhelper.CreateDB(&leveldbhelper.Conf{DBPath: path})
	db.Open()
	return &idStore{db}
}

func (s *idStore) setUnderConstructionFlag(ledgerID string) error {
	return s.db.Put(underConstructionLedgerKey, []byte(ledgerID), true)
}

func (s *idStore) unsetUnderConstructionFlag() error {
	return s.db.Delete(underConstructionLedgerKey, true)
}

func (s *idStore) getUnderConstructionFlag() (string, error) {
	val, err := s.db.Get(underConstructionLedgerKey)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func (s *idStore) createLedgerID(ledgerID string, gb *common.Block) error {
	key := s.encodeLedgerKey(ledgerID)
	var val []byte
	var err error
	if val, err = s.db.Get(key); err != nil {
		return err
	}
	if val != nil {
		return ErrLedgerIDExists
	}
	if val, err = proto.Marshal(gb); err != nil {
		return err
	}
	batch := &leveldb.Batch{}
	batch.Put(key, val)
	batch.Delete(underConstructionLedgerKey)
	return s.db.WriteBatch(batch, true)
}

func (s *idStore) ledgerIDExists(ledgerID string) (bool, error) {
	key := s.encodeLedgerKey(ledgerID)
	val := []byte{}
	err := error(nil)
	if val, err = s.db.Get(key); err != nil {
		return false, err
	}
	return val != nil, nil
}

func (s *idStore) getAllLedgerIds() ([]string, error) {
	var ids []string
	itr := s.db.GetIterator(nil, nil)
	defer itr.Release()
	itr.First()
	for itr.Valid() {
		if bytes.Equal(itr.Key(), underConstructionLedgerKey) {
			continue
		}
		id := string(s.decodeLedgerID(itr.Key()))
		ids = append(ids, id)
		itr.Next()
	}
	return ids, nil
}

func (s *idStore) close() {
	s.db.Close()
}

func (s *idStore) encodeLedgerKey(ledgerID string) []byte {
	return append(ledgerKeyPrefix, []byte(ledgerID)...)
}

func (s *idStore) decodeLedgerID(key []byte) string {
	return string(key[len(ledgerKeyPrefix):])
}
