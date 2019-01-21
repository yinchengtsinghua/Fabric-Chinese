
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


package historyleveldb

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/history/historydb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	putils "github.com/hyperledger/fabric/protos/utils"
)

var logger = flogging.MustGetLogger("historyleveldb")

var savePointKey = []byte{0x00}
var emptyValue = []byte{}

//HistoryDBProvider实现接口HistoryDBProvider
type HistoryDBProvider struct {
	dbProvider *leveldbhelper.Provider
}

//NewHistoryDBProvider实例化HistoryDBProvider
func NewHistoryDBProvider() *HistoryDBProvider {
	dbPath := ledgerconfig.GetHistoryLevelDBPath()
	logger.Debugf("constructing HistoryDBProvider dbPath=%s", dbPath)
	dbProvider := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})
	return &HistoryDBProvider{dbProvider}
}

//getdbhandle获取命名数据库的句柄
func (provider *HistoryDBProvider) GetDBHandle(dbName string) (historydb.HistoryDB, error) {
	return newHistoryDB(provider.dbProvider.GetDBHandle(dbName), dbName), nil
}

//关闭关闭基础数据库
func (provider *HistoryDBProvider) Close() {
	provider.dbProvider.Close()
}

//HistoryDB实现HistoryDB接口
type historyDB struct {
	db     *leveldbhelper.DBHandle
	dbName string
}

//newHistoryDB构造HistoryDB的实例
func newHistoryDB(db *leveldbhelper.DBHandle, dbName string) *historyDB {
	return &historyDB{db, dbName}
}

//在HistoryDB接口中打开实现方法
func (historyDB *historyDB) Open() error {
//不执行任何操作，因为使用了共享数据库
	return nil
}

//Close在HistoryDB接口中实现方法
func (historyDB *historyDB) Close() {
//不执行任何操作，因为使用了共享数据库
}

//提交在HistoryDB接口中实现方法
func (historyDB *historyDB) Commit(block *common.Block) error {

	blockNo := block.Header.Number
//将Starting Tranno设置为0
	var tranNo uint64

	dbBatch := leveldbhelper.NewUpdateBatch()

	logger.Debugf("Channel [%s]: Updating history database for blockNo [%v] with [%d] transactions",
		historyDB.dbName, blockNo, len(block.Data.Data))

//获取块的无效字节数组
	txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])

//将每个事务的写入集写入历史数据库
	for _, envBytes := range block.Data.Data {

//如果Tran标记为无效，则跳过它。
		if txsFilter.IsInvalid(int(tranNo)) {
			logger.Debugf("Channel [%s]: Skipping history write for invalid transaction number %d",
				historyDB.dbName, tranNo)
			tranNo++
			continue
		}

		env, err := putils.GetEnvelopeFromBlock(envBytes)
		if err != nil {
			return err
		}

		payload, err := putils.GetPayload(env)
		if err != nil {
			return err
		}

		chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			return err
		}

		if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {

//从信封消息中提取操作
			respPayload, err := putils.GetActionFromEnvelope(envBytes)
			if err != nil {
				return err
			}

//准备从事务中提取rwset
			txRWSet := &rwsetutil.TxRwSet{}

//从操作中获取结果，然后取消标记
//使用自定义反编排将其转换为txreadwriteset
			if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
				return err
			}
//对于每个事务，循环访问命名空间和写集
//并为每次写入添加历史记录
			for _, nsRWSet := range txRWSet.NsRwSets {
				ns := nsRWSet.NameSpace

				for _, kvWrite := range nsRWSet.KvRwSet.Writes {
					writeKey := kvWrite.Key

//历史记录的组合键格式为ns~key~blockno~tranno。
					compositeHistoryKey := historydb.ConstructCompositeHistoryKey(ns, writeKey, blockNo, tranNo)

//不需要任何值，请写入空字节数组（EmptyValue），因为不允许使用nil的put（）。
					dbBatch.Put(compositeHistoryKey, emptyValue)
				}
			}

		} else {
			logger.Debugf("Skipping transaction [%d] since it is not an endorsement transaction\n", tranNo)
		}
		tranNo++
	}

//为恢复添加保存点
	height := version.NewHeight(blockNo, tranNo)
	dbBatch.Put(savePointKey, height.ToBytes())

//将块的历史记录和保存点写入leveldb
//将snyc设置为true作为预防措施，在进一步测试之后，false可能是一个不错的优化。
	if err := historyDB.db.WriteBatch(dbBatch, true); err != nil {
		return err
	}

	logger.Debugf("Channel [%s]: Updates committed to history database for blockNo [%v]", historyDB.dbName, blockNo)
	return nil
}

//newHistoryQueryExecutor在HistoryDB接口中实现方法
func (historyDB *historyDB) NewHistoryQueryExecutor(blockStore blkstorage.BlockStore) (ledger.HistoryQueryExecutor, error) {
	return &LevelHistoryDBQueryExecutor{historyDB, blockStore}, nil
}

//GetBlockNumFromSavePoint在HistoryDB接口中实现方法
func (historyDB *historyDB) GetLastSavepoint() (*version.Height, error) {
	versionBytes, err := historyDB.db.Get(savePointKey)
	if err != nil || versionBytes == nil {
		return nil, err
	}
	height, _ := version.NewHeightFromBytes(versionBytes)
	return height, nil
}

//shouldRecover在接口kvledger.recoverer中实现方法
func (historyDB *historyDB) ShouldRecover(lastAvailableBlock uint64) (bool, uint64, error) {
	if !ledgerconfig.IsHistoryDBEnabled() {
		return false, 0, nil
	}
	savepoint, err := historyDB.GetLastSavepoint()
	if err != nil {
		return false, 0, err
	}
	if savepoint == nil {
		return true, 0, nil
	}
	return savepoint.BlockNum != lastAvailableBlock, savepoint.BlockNum + 1, nil
}

//commitListBlock在接口kvledger.recoverer中实现方法
func (historyDB *historyDB) CommitLostBlock(blockAndPvtdata *ledger.BlockAndPvtData) error {
	block := blockAndPvtdata.Block

//每隔1000个信息块记录一次，以便在生产环境中跟踪历史重建进度。
	if block.Header.Number%1000 == 0 {
		logger.Infof("Recommitting block [%d] to history database", block.Header.Number)
	} else {
		logger.Debugf("Recommitting block [%d] to history database", block.Header.Number)
	}

	if err := historyDB.Commit(block); err != nil {
		return err
	}
	return nil
}
