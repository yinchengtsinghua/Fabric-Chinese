
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


package ledgerconfig

import (
	"path/filepath"

	"github.com/hyperledger/fabric/core/config"
	"github.com/spf13/viper"
)

//iscouchdbenabled公开usecouchdb变量
func IsCouchDBEnabled() bool {
	stateDatabase := viper.GetString("ledger.state.stateDatabase")
	if stateDatabase == "CouchDB" {
		return true
	}
	return false
}

const confPeerFileSystemPath = "peer.fileSystemPath"
const confLedgersData = "ledgersData"
const confLedgerProvider = "ledgerProvider"
const confStateleveldb = "stateLeveldb"
const confHistoryLeveldb = "historyLeveldb"
const confBookkeeper = "bookkeeper"
const confConfigHistory = "configHistory"
const confChains = "chains"
const confPvtdataStore = "pvtdataStore"
const confTotalQueryLimit = "ledger.state.totalQueryLimit"
const confInternalQueryLimit = "ledger.state.couchDBConfig.internalQueryLimit"
const confEnableHistoryDatabase = "ledger.history.enableHistoryDatabase"
const confMaxBatchSize = "ledger.state.couchDBConfig.maxBatchUpdateSize"
const confAutoWarmIndexes = "ledger.state.couchDBConfig.autoWarmIndexes"
const confWarmIndexesAfterNBlocks = "ledger.state.couchDBConfig.warmIndexesAfterNBlocks"

var confCollElgProcMaxDbBatchSize = &conf{"ledger.pvtdataStore.collElgProcMaxDbBatchSize", 5000}
var confCollElgProcDbBatchesInterval = &conf{"ledger.pvtdataStore.collElgProcDbBatchesInterval", 1000}

//getrootpath返回文件系统路径。
//所有与分类帐相关的内容都应存储在此路径下
func GetRootPath() string {
	sysPath := config.GetPath(confPeerFileSystemPath)
	return filepath.Join(sysPath, confLedgersData)
}

//GETLeDeGeValpRealPATH返回用于存储分类帐提供程序内容的文件系统路径
func GetLedgerProviderPath() string {
	return filepath.Join(GetRootPath(), confLedgerProvider)
}

//GetStateLevelDBPath返回用于维护状态级别数据库的文件系统路径
func GetStateLevelDBPath() string {
	return filepath.Join(GetRootPath(), confStateleveldb)
}

//GetHistoryLevelDBPath returns the filesystem path that is used to maintain the history level db
func GetHistoryLevelDBPath() string {
	return filepath.Join(GetRootPath(), confHistoryLeveldb)
}

//GetBlockStorePath返回用于链块存储的文件系统路径
func GetBlockStorePath() string {
	return filepath.Join(GetRootPath(), confChains)
}

//getpvtdatastorepath返回用于永久存储专用写入集的文件系统路径
func GetPvtdataStorePath() string {
	return filepath.Join(GetRootPath(), confPvtdataStore)
}

//GetInternalBookKeepPath返回kvledger用于记帐内部资料的文件系统路径（例如pvt的过期时间）
func GetInternalBookkeeperPath() string {
	return filepath.Join(GetRootPath(), confBookkeeper)
}

//getconfigHistoryPath返回用于维护链码收集配置历史记录的文件系统路径
func GetConfigHistoryPath() string {
	return filepath.Join(GetRootPath(), confConfigHistory)
}

//GetMaxBlockFileSize返回块文件的最大大小
func GetMaxBlockfileSize() int {
	return 64 * 1024 * 1024
}

//gettotalquerylimit公开totallimit变量
func GetTotalQueryLimit() int {
	totalQueryLimit := viper.GetInt(confTotalQueryLimit)
//如果未设置querylimit，则默认为10000
	if !viper.IsSet(confTotalQueryLimit) {
		totalQueryLimit = 10000
	}
	return totalQueryLimit
}

//GetInternalQueryLimit公开QueryLimit变量
func GetInternalQueryLimit() int {
	internalQueryLimit := viper.GetInt(confInternalQueryLimit)
//如果未设置querylimit，则默认为1000
	if !viper.IsSet(confInternalQueryLimit) {
		internalQueryLimit = 1000
	}
	return internalQueryLimit
}

//getmaxbatchupdatesize公开maxbatchupdatesize变量
func GetMaxBatchUpdateSize() int {
	maxBatchUpdateSize := viper.GetInt(confMaxBatchSize)
//如果未设置Max ButcUpDebug，默认为500
	if !viper.IsSet(confMaxBatchSize) {
		maxBatchUpdateSize = 500
	}
	return maxBatchUpdateSize
}

//getpvtdatastorepurgeinterval返回以块数表示的间隔
//when the purge for the expired data would be performed
func GetPvtdataStorePurgeInterval() uint64 {
	purgeInterval := viper.GetInt("ledger.pvtdataStore.purgeInterval")
	if purgeInterval <= 0 {
		purgeInterval = 100
	}
	return uint64(purgeInterval)
}

//getpvtdatastorecollelgprocmaxdbbatchsize返回用于转换的最大db批大小
//不符合条件的丢失数据项到符合条件的丢失数据项
func GetPvtdataStoreCollElgProcMaxDbBatchSize() int {
	collElgProcMaxDbBatchSize := viper.GetInt(confCollElgProcMaxDbBatchSize.Name)
	if collElgProcMaxDbBatchSize <= 0 {
		collElgProcMaxDbBatchSize = confCollElgProcMaxDbBatchSize.DefaultVal
	}
	return collElgProcMaxDbBatchSize
}

//GetPvtDatastoreCollelgProcDBbatchesInterval返回写入之间的最短持续时间（毫秒）
//两个连续的数据库批，用于将不合格的缺失数据项转换为合格的缺失数据项
func GetPvtdataStoreCollElgProcDbBatchesInterval() int {
	collElgProcDbBatchesInterval := viper.GetInt(confCollElgProcDbBatchesInterval.Name)
	if collElgProcDbBatchesInterval <= 0 {
		collElgProcDbBatchesInterval = confCollElgProcDbBatchesInterval.DefaultVal
	}
	return collElgProcDbBatchesInterval
}

//IsHistoryDBEnabled公开HistoryDatabase变量
func IsHistoryDBEnabled() bool {
	return viper.GetBool(confEnableHistoryDatabase)
}

//IsQueryReadsShashingEnabled启用或禁用哈希计算
//虚项验证的范围查询结果
func IsQueryReadsHashingEnabled() bool {
	return true
}

//GetMaxDegreeQueryReadShashing返回的哈希的Merkle树的最大度数
//虚项验证的范围查询结果
//有关更多详细信息，请参阅kvledger/txmgmt/rwset/query_results_helper.go中的描述。
func GetMaxDegreeQueryReadsHashing() uint32 {
	return 50
}

//IsAutowarMindExesEnabled公开AutowarMindExes变量
func IsAutoWarmIndexesEnabled() bool {
//返回core.yaml中设置的值，如果未设置，则返回true
	if viper.IsSet(confAutoWarmIndexes) {
		return viper.GetBool(confAutoWarmIndexes)
	}
	return true

}

//getwarmindexesafternblocks公开warmindexesafternblocks变量
func GetWarmIndexesAfterNBlocks() int {
	warmAfterNBlocks := viper.GetInt(confWarmIndexesAfterNBlocks)
//如果warmindexesafternblock未设置，则默认为1
	if !viper.IsSet(confWarmIndexesAfterNBlocks) {
		warmAfterNBlocks = 1
	}
	return warmAfterNBlocks
}

type conf struct {
	Name       string
	DefaultVal int
}
