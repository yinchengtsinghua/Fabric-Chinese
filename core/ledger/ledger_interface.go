
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


package ledger

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-lib-go/healthz"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/peer"
)

//初始值设定项封装PeerledgerProvider的依赖项
type Initializer struct {
	StateListeners                []StateListener
	DeployedChaincodeInfoProvider DeployedChaincodeInfoProvider
	MembershipInfoProvider        MembershipInfoProvider
	MetricsProvider               metrics.Provider
	HealthCheckRegistry           HealthCheckRegistry
}

//PeerledgerProvider提供对分类帐实例的处理
type PeerLedgerProvider interface {
	Initialize(initializer *Initializer) error
//创建一个具有给定Genesis块的新分类帐。
//此函数确保创建分类帐并提交Genesis块将是一个原子操作
//从Genesis块中检索到的链ID被视为分类帐ID。
	Create(genesisBlock *common.Block) (PeerLedger, error)
//打开打开已创建的分类帐
	Open(ledgerID string) (PeerLedger, error)
//exists指示具有给定ID的分类帐是否存在
	Exists(ledgerID string) (bool, error)
//列表列出现有分类帐的ID
	List() ([]string, error)
//关闭关闭PeerledgerProvider
	Close()
}

//Peerledger与Ordererledger的不同之处在于，Peerledger在本地维护位掩码
//区分有效事务和无效事务
type PeerLedger interface {
	commonledger.Ledger
//GetTransactionByID按ID检索事务
	GetTransactionByID(txID string) (*peer.ProcessedTransaction, error)
//GetBlockByHash返回给定哈希的块
	GetBlockByHash(blockHash []byte) (*common.Block, error)
//getblockbytxid返回包含事务的块
	GetBlockByTxID(txID string) (*common.Block, error)
//gettxvalidationcodebytxid返回事务验证的原因代码
	GetTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error)
//newtxsimulator提供事务模拟器的句柄。
//客户机可以获得多个“txsimulator”用于并行执行。
//如果需要，应在实现级别执行任何快照/同步
	NewTxSimulator(txid string) (TxSimulator, error)
//NewQueryExecutor向查询执行器提供句柄。
//客户端可以获取多个“queryexecutor”以进行并行执行。
//如果需要，应在实现级别执行任何同步
	NewQueryExecutor() (QueryExecutor, error)
//newHistoryQueryExecutor为历史查询执行器提供句柄。
//客户端可以获取多个“HistoryQueryExecutor”以进行并行执行。
//如果需要，应在实现级别执行任何同步
	NewHistoryQueryExecutor() (HistoryQueryExecutor, error)
//getpvtdataandblockbynum返回块和相应的pvt数据。
//pvt数据由提供的“ns/collections”列表筛选
//nil筛选器不筛选任何结果，并导致检索给定blocknum的所有pvt数据
	GetPvtDataAndBlockByNum(blockNum uint64, filter PvtNsCollFilter) (*BlockAndPvtData, error)
//getpvtdatabynum仅返回与给定块号对应的pvt数据
//pvt数据由筛选器中提供的“ns/collections”列表筛选。
//nil筛选器不筛选任何结果，并导致检索给定blocknum的所有pvt数据
	GetPvtDataByNum(blockNum uint64, filter PvtNsCollFilter) ([]*TxPvtData, error)
//commitWithpvtData在原子操作中提交块和相应的pvt数据
	CommitWithPvtData(blockAndPvtdata *BlockAndPvtData) error
//清除在块高度小于
//给定的MaxBlockNumtoretain。换句话说，purge只保留私有读写集
//在MaxBlockNumtoretain或更高的块高度生成。
	PurgePrivateData(maxBlockNumToRetain uint64) error
//privatedataminblocknum返回保留的最低认可块高度
	PrivateDataMinBlockNum() (uint64, error)
//prune删除满足给定策略的块/事务
	Prune(policy commonledger.PrunePolicy) error
//getconfigHistoryRetriever返回configHistoryRetriever
	GetConfigHistoryRetriever() (ConfigHistoryRetriever, error)
//commitpvtdataofoldblocks提交与已提交的块对应的私有数据
//如果此函数中提供的某些私有数据的哈希不匹配
//块中存在相应的哈希，不匹配的私有数据不是
//提交并返回不匹配信息
	CommitPvtDataOfOldBlocks(blockPvtData []*BlockPvtData) ([]*PvtdataHashMismatch, error)
//GetMissingPvtDataTracker返回MissingPvtDataTracker
	GetMissingPvtDataTracker() (MissingPvtDataTracker, error)
}

//validatedLedger表示从对等分类帐中筛选出无效交易后的“最终分类帐”。
//V1后
type ValidatedLedger interface {
	commonledger.Ledger
}

//SimpleQueryExecutor封装了基本函数
type SimpleQueryExecutor interface {
//GetState获取给定命名空间和键的值。对于chaincode，命名空间对应于chaincodeid
	GetState(namespace string, key string) ([]byte, error)
//GetStateRangeScanIterator返回一个迭代器，该迭代器包含给定键范围之间的所有键值。
//结果中包含startkey，但不包括endkey。空的startkey引用第一个可用的key
//空的endkey指的是最后一个可用的key。用于扫描所有键，包括startkey和endkey
//可以作为空字符串提供。但是，出于性能原因，应该明智地使用完整扫描。
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	GetStateRangeScanIterator(namespace string, startKey string, endKey string) (commonledger.ResultsIterator, error)
}

//QueryExecutor执行查询
//get*方法用于支持基于kv的数据模型。ExecuteQuery方法支持丰富的数据模型和查询支持
//
//对于富数据模型，ExecuteQuery方法应支持对
//最新状态、历史状态以及状态与事务的交叉点
type QueryExecutor interface {
	SimpleQueryExecutor
//GetStateMetadata返回给定命名空间和键的元数据
	GetStateMetadata(namespace, key string) (map[string][]byte, error)
//GetStateMultipleKeys获取单个调用中多个键的值
	GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error)
//GetStateRangeScanIteratorWithMetadata返回一个迭代器，该迭代器包含给定键范围之间的所有键值。
//结果中包含startkey，但不包括endkey。空的startkey引用第一个可用的key
//空的endkey指的是最后一个可用的key。用于扫描所有键，包括startkey和endkey
//可以作为空字符串提供。但是，出于性能原因，应该明智地使用完整扫描。
//元数据是附加查询参数的映射
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	GetStateRangeScanIteratorWithMetadata(namespace string, startKey, endKey string, metadata map[string]interface{}) (QueryResultsIterator, error)
//ExecuteQuery执行给定的查询并返回一个迭代器，该迭代器包含特定于基础数据存储区的类型的结果。
//仅用于支持查询的状态数据库
//对于chaincode，命名空间对应于chaincodeid
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	ExecuteQuery(namespace, query string) (commonledger.ResultsIterator, error)
//ExecuteEqueryWithMetadata执行给定的查询并返回一个迭代器，该迭代器包含特定于基础数据存储区的类型的结果。
//元数据是附加查询参数的映射
//仅用于支持查询的状态数据库
//对于chaincode，命名空间对应于chaincodeid
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (QueryResultsIterator, error)
//getprivatedata获取由元组标识的私有数据项的值<namespace，collection，key>
	GetPrivateData(namespace, collection, key string) ([]byte, error)
//getprivatedatametadata获取由tuple<namespace，collection，key>
	GetPrivateDataMetadata(namespace, collection, key string) (map[string][]byte, error)
//getprivatedatametadatabyhash获取由元组标识的私有数据项的元数据<namespace，collection，keyhash>
	GetPrivateDataMetadataByHash(namespace, collection string, keyhash []byte) (map[string][]byte, error)
//getprivatedatamultiplekeys获取单个调用中多个私有数据项的值
	GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([][]byte, error)
//GetPrivateDataRangeScanIterator返回一个迭代器，该迭代器包含给定键范围之间的所有键值。
//结果中包含startkey，但不包括endkey。空的startkey引用第一个可用的key
//空的endkey指的是最后一个可用的key。用于扫描所有键，包括startkey和endkey
//可以作为空字符串提供。但是，出于性能方面的考虑，应谨慎使用全扫描Shuold。
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (commonledger.ResultsIterator, error)
//ExecuteQuery执行给定的查询并返回一个迭代器，该迭代器包含特定于基础数据存储区的类型的结果。
//仅用于支持查询的状态数据库
//对于chaincode，命名空间对应于chaincodeid
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	ExecuteQueryOnPrivateData(namespace, collection, query string) (commonledger.ResultsIterator, error)
//完成释放由QueryExecutor占用的资源
	Done()
}

//HistoryQueryExecutor执行历史查询
type HistoryQueryExecutor interface {
//GetHistoryForkey检索键值的历史记录。
//返回的resultsiterator包含protos/ledger/queryresult中定义的*keymodification类型的结果。
	GetHistoryForKey(namespace string, key string) (commonledger.ResultsIterator, error)
}

//TxSimulator在“尽可能最近的状态”的一致快照上模拟事务。
//set*方法用于支持基于kv的数据模型。ExecuteUpdate方法支持丰富的数据模型和查询支持
type TxSimulator interface {
	QueryExecutor
//setState为给定的命名空间和键设置给定值。对于chaincode，命名空间对应于chaincodeid
	SetState(namespace string, key string, value []byte) error
//DeleteState删除给定的命名空间和键
	DeleteState(namespace string, key string) error
//setmultiplekeys设置单个调用中多个键的值
	SetStateMultipleKeys(namespace string, kvs map[string][]byte) error
//setStateMetadata设置与现有键元组关联的元数据<namespace，key>
	SetStateMetadata(namespace, key string, metadata map[string][]byte) error
//deleteStateMetadata删除与现有键元组关联的元数据（如果有）<namespace，key>
	DeleteStateMetadata(namespace, key string) error
//用于支持富数据模型的ExecuteUpdate（请参见上面关于QueryExecutor的注释）
	ExecuteUpdate(query string) error
//setprivatedata将给定值设置为由tuple<namespace，collection，key>
	SetPrivateData(namespace, collection, key string, value []byte) error
//setprivatedatamultiplekeys在单个调用中为私有数据空间中的多个键设置值
	SetPrivateDataMultipleKeys(namespace, collection string, kvs map[string][]byte) error
//deleteprivatedata从私有数据中删除给定的tuple<namespace，collection，key>
	DeletePrivateData(namespace, collection, key string) error
//setprivatedatametadata设置与现有密钥元组关联的元数据<namespace，collection，key>
	SetPrivateDataMetadata(namespace, collection, key string, metadata map[string][]byte) error
//deleteprivatedatametadata删除与现有密钥元组关联的元数据<namespace，collection，key>
	DeletePrivateDataMetadata(namespace, collection, key string) error
//GetTxSimulationResults封装了事务模拟的结果。
//这应该包含足够的细节
//-如果要提交事务，将导致的状态更新
//-执行事务的环境，以便能够决定环境的有效性。
//（稍后在另一个对等机上）在提交事务期间
//不同的分类帐实现（或单个实现的配置）可能需要表示以上两个部分
//以不同的方式支持不同的数据模型或优化信息表示。
//返回的类型“txSimulationResults”包含公共数据和私有数据的模拟结果。
//公共数据模拟结果预计将在v1中使用，而私有数据模拟结果预计将在v1中使用。
//用于八卦传播给其他背书人（在SideDB第2阶段）
	GetTxSimulationResults() (*TxSimulationResults, error)
}

//queryresultsiterator-查询结果集的迭代器
type QueryResultsIterator interface {
	commonledger.ResultsIterator
//getbookmarkandclose返回分页书签并释放迭代器占用的资源
	GetBookmarkAndClose() string
}

//txpvtdata封装事务编号和事务的pvt写入集
type TxPvtData struct {
	SeqInBlock uint64
	WriteSet   *rwset.TxPvtReadWriteSet
}

//txpvdtatamap是从txnum到pvdtata的映射
type TxPvtDataMap map[uint64]*TxPvtData

//MissingPvtData包含的命名空间和集合
//其中pvtdata不存在。它也表示
//丢失的pvtdata是否合格（即
//对等方是[namespace，collection]的成员。
type MissingPvtData struct {
	Namespace  string
	Collection string
	IsEligible bool
}

//txmissingpvdtatamap是从txnum到列表的映射
//缺失PVTDATA
type TxMissingPvtDataMap map[uint64][]*MissingPvtData

//blockAndpvtData封装块和包含元组的映射<seqinblock，*txpvtdata>
//映射应该只包含与pvt数据关联的事务的条目。
type BlockAndPvtData struct {
	Block          *common.Block
	PvtData        TxPvtDataMap
	MissingPvtData TxMissingPvtDataMap
}

//blockpvdtata包含块的私有数据
type BlockPvtData struct {
	BlockNum  uint64
	WriteSets TxPvtDataMap
}

//添加在MissingPrivateDataList中添加给定的缺少的私有数据
func (txMissingPvtData TxMissingPvtDataMap) Add(txNum uint64, ns, coll string, isEligible bool) {
	txMissingPvtData[txNum] = append(txMissingPvtData[txNum], &MissingPvtData{ns, coll, isEligible})
}

//pvtcollfilter表示集合名称集（作为值为“true”的映射键）
type PvtCollFilter map[string]bool

//pvtnscollfilter指定tuple<namespace，pvtcollfilter>
type PvtNsCollFilter map[string]PvtCollFilter

//newpvtnscollfilter构造空pvtnscollfilter
func NewPvtNsCollFilter() PvtNsCollFilter {
	return make(map[string]PvtCollFilter)
}

//如果pvtdata包含用于收集的数据<ns，coll>
func (pvtdata *TxPvtData) Has(ns string, coll string) bool {
	if pvtdata.WriteSet == nil {
		return false
	}
	for _, nsdata := range pvtdata.WriteSet.NsPvtRwset {
		if nsdata.Namespace == ns {
			for _, colldata := range nsdata.CollectionPvtRwset {
				if colldata.CollectionName == coll {
					return true
				}
			}
		}
	}
	return false
}

//添加将命名空间集合元组添加到筛选器
func (filter PvtNsCollFilter) Add(ns string, coll string) {
	collFilter, ok := filter[ns]
	if !ok {
		collFilter = make(map[string]bool)
		filter[ns] = collFilter
	}
	collFilter[coll] = true
}

//如果筛选器具有元组命名空间集合的条目，则返回true
func (filter PvtNsCollFilter) Has(ns string, coll string) bool {
	collFilter, ok := filter[ns]
	if !ok {
		return false
	}
	return collFilter[coll]
}

//TXSimulationResults捕获模拟结果的详细信息
type TxSimulationResults struct {
	PubSimulationResults *rwset.TxReadWriteSet
	PvtSimulationResults *rwset.TxPvtReadWriteSet
}

//GetPubSimulationBytes返回公共读写集的序列化字节
func (txSim *TxSimulationResults) GetPubSimulationBytes() ([]byte, error) {
	return proto.Marshal(txSim.PubSimulationResults)
}

//getpvtsimulationBytes返回私有读写集的序列化字节
func (txSim *TxSimulationResults) GetPvtSimulationBytes() ([]byte, error) {
	if !txSim.ContainsPvtWrites() {
		return nil, nil
	}
	return proto.Marshal(txSim.PvtSimulationResults)
}

//如果模拟结果包括私有写入，则containsPvTwrites返回true
func (txSim *TxSimulationResults) ContainsPvtWrites() bool {
	return txSim.PvtSimulationResults != nil
}

//go：生成仿冒者-o mock/state_listener.go-forke name state listener。他汀类药物

//StateListener允许自定义代码在状态更改时执行其他操作
//用于侦听器注册所依据的特定命名空间。
//这有助于执行除状态更新以外的自定义任务。
//分类帐实现应在每个块和
//传递给函数的“stateupdates”参数捕获由块引起的状态更改
//对于命名空间。状态更新的实际数据类型取决于启用的数据模型。
//例如，对于kv数据模型，实际类型是proto message
//`github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset.kvwrite`
//应在提交块之前调用函数“handlestateupdates”，如果
//函数返回错误，分类帐实现应停止块提交操作
//导致恐慌
type StateListener interface {
	InterestedInNamespaces() []string
	HandleStateUpdates(trigger *StateUpdateTrigger) error
	StateCommitDone(channelID string)
}

//StateUpdateTrigger封装了StateListener可能使用的信息和帮助工具
type StateUpdateTrigger struct {
	LedgerID                    string
	StateUpdates                StateUpdates
	CommittingBlockNum          uint64
	CommittedStateQueryExecutor SimpleQueryExecutor
	PostCommitQueryExecutor     SimpleQueryExecutor
}

//StateUpdates是表示状态更新的通用类型
type StateUpdates map[string]interface{}

//configHistoryRetriever允许检索集合配置的历史记录
type ConfigHistoryRetriever interface {
	CollectionConfigAt(blockNum uint64, chaincodeName string) (*CollectionConfigInfo, error)
	MostRecentCollectionConfigBelow(blockNum uint64, chaincodeName string) (*CollectionConfigInfo, error)
}

//MissingPvtDataTracker允许获取有关对等机上未丢失的私有数据的信息
type MissingPvtDataTracker interface {
	GetMissingPvtDataInfoForMostRecentBlocks(maxBlocks int) (MissingPvtDataInfo, error)
}

//MissingPvtDataInfo是块号到MissingBlockPvTDataInfo的映射
type MissingPvtDataInfo map[uint64]MissingBlockPvtdataInfo

//MissingBlockPvtDataInfo是事务编号（在块内）到MissingCollectionPvTDataInfo的映射
type MissingBlockPvtdataInfo map[uint64][]*MissingCollectionPvtDataInfo

//MissingCollectionPvtDataInfo包含缺少私有数据的链代码和集合的名称
type MissingCollectionPvtDataInfo struct {
	Namespace, Collection string
}

//collectionconfiginfo封装链码的集合配置及其提交块号
type CollectionConfigInfo struct {
	CollectionConfig   *common.CollectionConfigPackage
	CommittingBlockNum uint64
}

//添加将丢失的数据项添加到MissingPvtDataInfo映射
func (missingPvtDataInfo MissingPvtDataInfo) Add(blkNum, txNum uint64, ns, coll string) {
	missingBlockPvtDataInfo, ok := missingPvtDataInfo[blkNum]
	if !ok {
		missingBlockPvtDataInfo = make(MissingBlockPvtdataInfo)
		missingPvtDataInfo[blkNum] = missingBlockPvtDataInfo
	}

	if _, ok := missingBlockPvtDataInfo[txNum]; !ok {
		missingBlockPvtDataInfo[txNum] = []*MissingCollectionPvtDataInfo{}
	}

	missingBlockPvtDataInfo[txNum] = append(missingBlockPvtDataInfo[txNum],
		&MissingCollectionPvtDataInfo{
			Namespace:  ns,
			Collection: coll})
}

//errcollectionconfignityetavailable是从函数返回的错误
//configHistoryRetriever.collectionconfigat（）如果提交了最新的块号
//低于请求中指定的块号。
type ErrCollectionConfigNotYetAvailable struct {
	MaxBlockNumCommitted uint64
	Msg                  string
}

func (e *ErrCollectionConfigNotYetAvailable) Error() string {
	return e.Msg
}

//notfoundinindexerr用于指示索引中缺少条目。
type NotFoundInIndexErr string

func (NotFoundInIndexErr) Error() string {
	return "Entry not found in index"
}

//每次操作时都返回collconfignotdefinedError
//在尚未定义配置的集合上请求
type CollConfigNotDefinedError struct {
	Ns string
}

func (e *CollConfigNotDefinedError) Error() string {
	return fmt.Sprintf("collection config not defined for chaincode [%s], pass the collection configuration upon chaincode definition/instantiation", e.Ns)
}

//每当操作执行时，都返回InvalidColNameError
//在名称无效的集合上请求
type InvalidCollNameError struct {
	Ns, Coll string
}

func (e *InvalidCollNameError) Error() string {
	return fmt.Sprintf("collection [%s] not defined in the collection config for chaincode [%s]", e.Coll, e.Ns)
}

//当私有写入集的哈希值为
//与块中存在的相应哈希不匹配
//有关用法，请参见函数“peerledger.commitpvtdata”
type PvtdataHashMismatch struct {
	BlockNum, TxNum       uint64
	Namespace, Collection string
	ExpectedHash          []byte
}

//DeployedChaincodeInfo提供程序是分类帐用于生成集合配置历史记录的依赖项
//LSCC模块需要为这个依赖项提供一个实现
type DeployedChaincodeInfoProvider interface {
	Namespaces() []string
	UpdatedChaincodes(stateUpdates map[string][]*kvrwset.KVWrite) ([]*ChaincodeLifecycleInfo, error)
	ChaincodeInfo(chaincodeName string, qe SimpleQueryExecutor) (*DeployedChaincodeInfo, error)
	CollectionInfo(chaincodeName, collectionName string, qe SimpleQueryExecutor) (*common.StaticCollectionConfig, error)
}

//deployedchaincodeinfo封装已部署链码中的链码信息
type DeployedChaincodeInfo struct {
	Name                string
	Hash                []byte
	Version             string
	CollectionConfigPkg *common.CollectionConfigPackage
}

//chaincodelifecycleinfo捕获chaincode的更新信息
type ChaincodeLifecycleInfo struct {
	Name    string
	Deleted bool
Details *ChaincodeLifecycleDetails //可以包含有关可用于特定优化的生命周期事件的更详细信息
}

//ChaincodeLifecycleDetails捕获ChaincodeLifecycle事件的更详细信息
type ChaincodeLifecycleDetails struct {
Updated bool //如果更新了现有链码，则为true（对于新部署的链码，则为false）。
//只有当“updated”为true时，以下属性才有意义
HashChanged        bool     //如果更改了chaincode代码包，则返回true。
CollectionsUpdated []string //添加或更新的集合的名称
CollectionsRemoved []string //删除的集合的名称
}

//MembershipInfoProvider是一个依赖项，分类帐使用它来确定当前对等方是否是
//集合的成员。八卦模块将提供对分类帐的依赖关系
type MembershipInfoProvider interface {
//amMemberOf检查当前对等方是否为给定集合的成员
	AmMemberOf(channelName string, collectionPolicyConfig *common.CollectionPolicyConfig) (bool, error)
}

//go：生成仿冒者-o mock/deployed_cinfo_provider.go-fake name deployedchaincodeinfo provider。部署caincodeinfo提供程序
//go:生成伪造者-o mock/membership_info_provider.go-forke name membership info provider。成员身份信息提供程序

//go：生成伪造者-o mock/health_check_registry.go-fake name health check registry。健康检查注册表

type HealthCheckRegistry interface {
	RegisterChecker(string, healthz.HealthChecker) error
}
