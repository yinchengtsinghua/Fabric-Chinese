
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


package transientstore

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

var logger = flogging.MustGetLogger("transientstore")

var emptyValue = []byte{}
var nilByte = byte('\x00')

//errstoreEmpty用于指示临时存储中没有条目
var ErrStoreEmpty = errors.New("Transient store is empty")

/////////////////////////////////////
//接口和数据类型
////

//StoreProvider提供TransientStore的实例
type StoreProvider interface {
	OpenStore(ledgerID string) (Store, error)
	Close()
}

//rwsetscanner为背书器vt模拟结果提供迭代器
type RWSetScanner interface {
//下一步返回RWSetscanner的下一个背书器vt模拟结果。
//如果没有进一步的数据，它可以返回nil，也可以返回错误。
//关于失败
	Next() (*EndorserPvtSimulationResults, error)
//NextWithConfig返回RWSetscanner中的下一个RemarkerPvSimulationResultsWithConfig。
//如果没有进一步的数据，它可以返回nil，也可以返回错误。
//关于失败
//TODO:一旦根据FAB-5096进行了相关的八卦更改，请删除上述功能
//并将下面的函数重命名为下一个表单NextWithConfig。
	NextWithConfig() (*EndorserPvtSimulationResultsWithConfig, error)
//关闭释放与此rwsetscanner关联的资源
	Close()
}

//存储管理LedgerID的私有写入集的存储。
//理想情况下，分类帐可以在将数据提交给
//某些数据项的永久存储或修剪由策略强制执行
type Store interface {
//持久性将事务的私有写入集存储在临时存储中
//根据TxID和块高度，在
	Persist(txid string, blockHeight uint64, privateSimulationResults *rwset.TxPvtReadWriteSet) error
//TODO:一旦根据FAB-5096进行了相关的八卦更改，请删除上述功能
//并将下面的函数重命名为persistewithconfig。
//PersisteWithConfig存储事务的私有写入集以及集合配置
//在基于txid和块高度的临时存储中，在
	PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo) error
//gettxpvtrwsetbytxid返回迭代器，因为txid可能有多个private
//来自不同代言人的写集（通过八卦）
	GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (RWSetScanner, error)
//PurgeByTxids removes private write sets of a given set of transactions from the
//瞬态存储
	PurgeByTxids(txids []string) error
//purgebyHeight删除块高度小于
//给定的最大阻塞数。换句话说，清除只保留私有写集。
//保留在MaxBlockNumtoretain或更高的块高度。尽管是私人的
//用PurgBytxIdx（）将存储在瞬态存储中的写集由协调器删除。
//块提交成功后，仍需要purgeByHeight（）来删除孤立的条目（如
//transaction that gets endorsed may not be submitted by the client for commit)
	PurgeByHeight(maxBlockNumToRetain uint64) error
//GetMinTransientBlkht返回瞬态存储中剩余的最低块高度
	GetMinTransientBlkHt() (uint64, error)
	Shutdown()
}

//背书人vt模拟结果捕获特定于背书人的模拟结果的详细信息
//TODO:一旦根据FAB-5096进行了相关的八卦更改，请删除此结构
type EndorserPvtSimulationResults struct {
	ReceivedAtBlockHeight uint64
	PvtSimulationResults  *rwset.TxPvtReadWriteSet
}

//背书人vt模拟结果thconfig捕获特定于背书人的模拟结果的详细信息
type EndorserPvtSimulationResultsWithConfig struct {
	ReceivedAtBlockHeight          uint64
	PvtSimulationResultsWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo
}

/////////////////////////////////////
//实施
////

//StoreProvider封装用于存储的LevelDB提供程序
//private write sets of simulated transactions, and implements TransientStoreProvider
//接口。
type storeProvider struct {
	dbProvider *leveldbhelper.Provider
}

//存储包含一个LevelDB实例。
type store struct {
	db       *leveldbhelper.DBHandle
	ledgerID string
}

type RwsetScanner struct {
	txid   string
	dbItr  iterator.Iterator
	filter ledger.PvtNsCollFilter
}

//Newstoreprovider实例化TransientStoreProvider
func NewStoreProvider() StoreProvider {
	dbProvider := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: GetTransientStorePath()})
	return &storeProvider{dbProvider: dbProvider}
}

//OpenStore将句柄返回到Store中的LedgerID
func (provider *storeProvider) OpenStore(ledgerID string) (Store, error) {
	dbHandle := provider.dbProvider.GetDBHandle(ledgerID)
	return &store{db: dbHandle, ledgerID: ledgerID}, nil
}

//Close closes the TransientStoreProvider
func (provider *storeProvider) Close() {
	provider.dbProvider.Close()
}

//持久性将事务的私有写入集存储在临时存储中
//根据TxID和块高度，在
//TODO:一旦根据FAB-5096进行了相关的八卦更改，请删除此功能。
func (s *store) Persist(txid string, blockHeight uint64,
	privateSimulationResults *rwset.TxPvtReadWriteSet) error {

	logger.Debugf("Persisting private data to transient store for txid [%s] at block height [%d]", txid, blockHeight)

	dbBatch := leveldbhelper.NewUpdateBatch()

//使用适当的前缀、txid、uuid和blockheight创建compositekey
//由于txid可能有多个私有写入集，这些写入集是从不同的
//代言人（通过八卦），我们发布一个UUID和TXID，以避免冲突。
	uuid := util.GenerateUUID()
	compositeKeyPvtRWSet := createCompositeKeyForPvtRWSet(txid, uuid, blockHeight)
	privateSimulationResultsBytes, err := proto.Marshal(privateSimulationResults)
	if err != nil {
		return err
	}
	dbBatch.Put(compositeKeyPvtRWSet, privateSimulationResultsBytes)

//创建两个索引：（i）按txid，（i i）按高度

//使用适当的前缀、blockheight、创建用于按高度清除索引的compositekey。
//txid、uuid并以nil字节作为值存储compositekey（清除索引）。注意
//清除索引用于删除临时存储中的孤立项（未删除
//by PurgeTxids()) using BTL policy by PurgeByHeight(). Note that orphan entries are due to transaction
//由客户批准但未提交提交提交的）
	compositeKeyPurgeIndexByHeight := createCompositeKeyForPurgeIndexByHeight(blockHeight, txid, uuid)
	dbBatch.Put(compositeKeyPurgeIndexByHeight, emptyValue)

//使用适当的前缀、txid、uuid、创建用于清除索引的compositekey。
//blockHeight and store the compositeKey (purge index) with a nil byte as value.
//虽然compositekeypvtrwset本身可以用于清除txid设置的私有写入，
//我们创建一个单独的复合键，值为零字节。原因是
//如果我们使用compositekeypvtrwset，我们就不必要地读取（可能是大的）私有写
//与数据库中的键关联的集。请注意，此清除索引用于删除非孤立的
//临时存储中的条目，由purgetxids（）使用
//注意：只需替换compositekeypvtrwset的前缀，就可以创建compositekeyprugeindexbytxid。
//具有purgeindexBythIdPrefix。为了代码的可读性和表达性，我们使用
//而创建compositekeycorpurgeindexbytxid（）。
	compositeKeyPurgeIndexByTxid := createCompositeKeyForPurgeIndexByTxid(txid, uuid, blockHeight)
	dbBatch.Put(compositeKeyPurgeIndexByTxid, emptyValue)

	return s.db.WriteBatch(dbBatch, true)
}

//PersisteWithConfig存储事务的私有写入集以及集合配置
//在基于txid和块高度的临时存储中，在
//TODO:一旦根据FAB-5096进行了相关的八卦更改，请重命名此函数以保持
//表单持久化配置。
func (s *store) PersistWithConfig(txid string, blockHeight uint64,
	privateSimulationResultsWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo) error {

	logger.Debugf("Persisting private data to transient store for txid [%s] at block height [%d]", txid, blockHeight)

	dbBatch := leveldbhelper.NewUpdateBatch()

//使用适当的前缀、txid、uuid和blockheight创建compositekey
//由于txid可能有多个私有写入集，这些写入集是从不同的
//代言人（通过八卦），我们发布一个UUID和TXID，以避免冲突。
	uuid := util.GenerateUUID()
	compositeKeyPvtRWSet := createCompositeKeyForPvtRWSet(txid, uuid, blockHeight)
	privateSimulationResultsWithConfigBytes, err := proto.Marshal(privateSimulationResultsWithConfig)
	if err != nil {
		return err
	}

//注意一些rwset.txpvtreadwriteset可能在之后立即存在于临时存储中。
//将对等机升级到v1.2。为了区分新原型和旧原型
//retrieving, a nil byte is prepended to the new proto, i.e., privateSimulationResultsWithConfigBytes,
//因为封送的消息不能以零字节开头。在v1.3中，我们可以避免
//零字节。
	value := append([]byte{nilByte}, privateSimulationResultsWithConfigBytes...)
	dbBatch.Put(compositeKeyPvtRWSet, value)

//创建两个索引：（i）按txid，（i i）按高度

//使用适当的前缀、blockheight、创建用于按高度清除索引的compositekey。
//txid、uuid并以nil字节作为值存储compositekey（清除索引）。注意
//清除索引用于删除临时存储中的孤立项（未删除
//通过purgeByHeight（）使用btl策略。请注意，孤立条目是由事务引起的
//由客户批准但未提交提交提交的）
	compositeKeyPurgeIndexByHeight := createCompositeKeyForPurgeIndexByHeight(blockHeight, txid, uuid)
	dbBatch.Put(compositeKeyPurgeIndexByHeight, emptyValue)

//使用适当的前缀、txid、uuid、创建用于清除索引的compositekey。
//blockheight并以nil字节作为值存储compositekey（清除索引）。
//虽然compositekeypvtrwset本身可以用于清除txid设置的私有写入，
//我们创建一个单独的复合键，值为零字节。原因是
//如果我们使用compositekeypvtrwset，我们就不必要地读取（可能是大的）私有写
//与数据库中的键关联的集。请注意，此清除索引用于删除非孤立的
//临时存储中的条目，由purgetxids（）使用
//注意：只需替换compositekeypvtrwset的前缀，就可以创建compositekeyprugeindexbytxid。
//具有purgeindexBythIdPrefix。为了代码的可读性和表达性，我们使用
//而创建compositekeycorpurgeindexbytxid（）。
	compositeKeyPurgeIndexByTxid := createCompositeKeyForPurgeIndexByTxid(txid, uuid, blockHeight)
	dbBatch.Put(compositeKeyPurgeIndexByTxid, emptyValue)

	return s.db.WriteBatch(dbBatch, true)
}

//gettxpvtrwsetbytxid返回迭代器，因为txid可能有多个private
//来自不同背书人的写入集保持不变。
func (s *store) GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (RWSetScanner, error) {

	logger.Debugf("Getting private data from transient store for transaction %s", txid)

//Construct startKey and endKey to do an range query
	startKey := createTxidRangeStartKey(txid)
	endKey := createTxidRangeEndKey(txid)

	iter := s.db.GetIterator(startKey, endKey)
	return &RwsetScanner{txid, iter, filter}, nil
}

//purgebytxids从
//瞬态存储。PurgeByTxids() is expected to be called by coordinator after
//将块提交到分类帐。
func (s *store) PurgeByTxids(txids []string) error {

	logger.Debug("Purging private data from transient store for committed txids")

	dbBatch := leveldbhelper.NewUpdateBatch()

	for _, txid := range txids {
//构造startkey和endkey以执行范围查询
		startKey := createPurgeIndexByTxidRangeStartKey(txid)
		endKey := createPurgeIndexByTxidRangeEndKey(txid)

		iter := s.db.GetIterator(startKey, endKey)

//从上面的结果中获取所有txid和uuid，并将其从临时存储中删除（两者都是
//写入集和相应的索引。
		for iter.Next() {
//对于每个条目，删除私有读写集和相关索引

//删除私有写入集
			compositeKeyPurgeIndexByTxid := iter.Key()
//注意：只需替换compositekeypurgeindexbytxid的前缀，就可以创建compositekeypvtrwset。
//带prwsetPrefix。为了提高代码的可读性和表达能力，我们重新进行了拆分和创建。
			uuid, blockHeight := splitCompositeKeyOfPurgeIndexByTxid(compositeKeyPurgeIndexByTxid)
			compositeKeyPvtRWSet := createCompositeKeyForPvtRWSet(txid, uuid, blockHeight)
			dbBatch.Delete(compositeKeyPvtRWSet)

//Remove purge index -- purgeIndexByHeight
			compositeKeyPurgeIndexByHeight := createCompositeKeyForPurgeIndexByHeight(blockHeight, txid, uuid)
			dbBatch.Delete(compositeKeyPurgeIndexByHeight)

//删除清除索引--purgeindexbytxid
			dbBatch.Delete(compositeKeyPurgeIndexByTxid)
		}
		iter.Release()
	}
//如果对等机在将批写入goleveldb之前/期间失败，则这些条目将
//removed as per BTL policy later by PurgeByHeight()
	return s.db.WriteBatch(dbBatch, true)
}

//purgebyHeight删除块高度小于
//给定的最大阻塞数。换句话说，清除只保留私有写集。
//保留在MaxBlockNumtoretain或更高的块高度。尽管是私人的
//用PurgBytxIdx（）将存储在瞬态存储中的写集由协调器删除。
//块提交成功后，仍需要purgeByHeight（）来删除孤立的条目（如
//获得批准的事务不能由客户端提交以供提交）
func (s *store) PurgeByHeight(maxBlockNumToRetain uint64) error {

	logger.Debugf("Purging orphaned private data from transient store received prior to block [%d]", maxBlockNumToRetain)

//使用0作为startkey和maxblocknumtoretain-1作为endkey执行范围查询
	startKey := createPurgeIndexByHeightRangeStartKey(0)
	endKey := createPurgeIndexByHeightRangeEndKey(maxBlockNumToRetain - 1)
	iter := s.db.GetIterator(startKey, endKey)

	dbBatch := leveldbhelper.NewUpdateBatch()

//从上面的结果中获取所有txid和uuid，并将其从临时存储中删除（两者都是
//写入集和相应的索引。
	for iter.Next() {
//对于每个条目，删除私有读写集和相关索引

//删除私有写入集
		compositeKeyPurgeIndexByHeight := iter.Key()
		txid, uuid, blockHeight := splitCompositeKeyOfPurgeIndexByHeight(compositeKeyPurgeIndexByHeight)
		logger.Debugf("Purging from transient store private data simulated at block [%d]: txid [%s] uuid [%s]", blockHeight, txid, uuid)

		compositeKeyPvtRWSet := createCompositeKeyForPvtRWSet(txid, uuid, blockHeight)
		dbBatch.Delete(compositeKeyPvtRWSet)

//删除清除索引--purgeindexbytxid
		compositeKeyPurgeIndexByTxid := createCompositeKeyForPurgeIndexByTxid(txid, uuid, blockHeight)
		dbBatch.Delete(compositeKeyPurgeIndexByTxid)

//删除清除索引--purgeindexbyheight
		dbBatch.Delete(compositeKeyPurgeIndexByHeight)
	}
	iter.Release()

	return s.db.WriteBatch(dbBatch, true)
}

//GetMinTransientBlkht返回瞬态存储中剩余的最低块高度
func (s *store) GetMinTransientBlkHt() (uint64, error) {
//Current approach performs a range query on purgeIndex with startKey
//为0（即blockheight），并返回第一个键，表示
//暂时存储中剩余的最低块高度。另一种方法
//在TransientStore中显式存储minBlockHeight。
	startKey := createPurgeIndexByHeightRangeStartKey(0)
	iter := s.db.GetIterator(startKey, nil)
	defer iter.Release()
//获取最小瞬态块高度
	if iter.Next() {
		dbKey := iter.Key()
		_, _, blockHeight := splitCompositeKeyOfPurgeIndexByHeight(dbKey)
		return blockHeight, nil
	}
//返回错误可能不是正确的操作。可能是
//返回一个布尔。-1不可能，因为unsigned int是第一个
//返回值
	return 0, ErrStoreEmpty
}

func (s *store) Shutdown() {
//不执行任何操作，因为使用了共享数据库
}

//下一步将迭代器移动到下一个键/值对。
//它返回迭代器是否耗尽。
//TODO: Once the related gossip changes are made as per FAB-5096, remove this function
func (scanner *RwsetScanner) Next() (*EndorserPvtSimulationResults, error) {
	if !scanner.dbItr.Next() {
		return nil, nil
	}
	dbKey := scanner.dbItr.Key()
	dbVal := scanner.dbItr.Value()
	_, blockHeight := splitCompositeKeyOfPvtRWSet(dbKey)

	txPvtRWSet := &rwset.TxPvtReadWriteSet{}
	if err := proto.Unmarshal(dbVal, txPvtRWSet); err != nil {
		return nil, err
	}
	filteredTxPvtRWSet := trimPvtWSet(txPvtRWSet, scanner.filter)

	return &EndorserPvtSimulationResults{
		ReceivedAtBlockHeight: blockHeight,
		PvtSimulationResults:  filteredTxPvtRWSet,
	}, nil
}

//下一步将迭代器移动到下一个键/值对。
//它返回迭代器是否耗尽。
//TODO:根据FAB-5096对相关的八卦进行更改后，将此函数重命名为Next
func (scanner *RwsetScanner) NextWithConfig() (*EndorserPvtSimulationResultsWithConfig, error) {
	if !scanner.dbItr.Next() {
		return nil, nil
	}
	dbKey := scanner.dbItr.Key()
	dbVal := scanner.dbItr.Value()
	_, blockHeight := splitCompositeKeyOfPvtRWSet(dbKey)

	txPvtRWSet := &rwset.TxPvtReadWriteSet{}
	filteredTxPvtRWSet := &rwset.TxPvtReadWriteSet{}
	txPvtRWSetWithConfig := &transientstore.TxPvtReadWriteSetWithConfigInfo{}

	if dbVal[0] == nilByte {
//新协议，即txpvtreadwritesetwithconfiginfo
		if err := proto.Unmarshal(dbVal[1:], txPvtRWSetWithConfig); err != nil {
			return nil, err
		}

		filteredTxPvtRWSet = trimPvtWSet(txPvtRWSetWithConfig.GetPvtRwset(), scanner.filter)
		configs, err := trimPvtCollectionConfigs(txPvtRWSetWithConfig.CollectionConfigs, scanner.filter)
		if err != nil {
			return nil, err
		}
		txPvtRWSetWithConfig.CollectionConfigs = configs
	} else {
//旧协议，即txpvtreadwriteset
		if err := proto.Unmarshal(dbVal, txPvtRWSet); err != nil {
			return nil, err
		}
		filteredTxPvtRWSet = trimPvtWSet(txPvtRWSet, scanner.filter)
	}

	txPvtRWSetWithConfig.PvtRwset = filteredTxPvtRWSet

	return &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          blockHeight,
		PvtSimulationResultsWithConfig: txPvtRWSetWithConfig,
	}, nil
}

//close释放迭代器持有的资源
func (scanner *RwsetScanner) Close() {
	scanner.dbItr.Release()
}
