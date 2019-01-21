
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


package statecouchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("statecouchdb")

//通过查询分页实现queryskip以供将来使用
//当前默认为0，未使用
const querySkip = 0

//lsccachesize表示lsccstatecache中允许的条目数
const lsccCacheSize = 50

//versioneddbprovider实现接口versioneddbprovider
type VersionedDBProvider struct {
	couchInstance *couchdb.CouchInstance
	databases     map[string]*VersionedDB
	mux           sync.Mutex
	openCounts    uint64
}

//NewVersionedDBProvider实例化VersionedDBProvider
func NewVersionedDBProvider(metricsProvider metrics.Provider) (*VersionedDBProvider, error) {
	logger.Debugf("constructing CouchDB VersionedDBProvider")
	couchDBDef := couchdb.GetCouchDBDefinition()
	couchInstance, err := couchdb.CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, metricsProvider)
	if err != nil {
		return nil, err
	}
	return &VersionedDBProvider{couchInstance, make(map[string]*VersionedDB), sync.Mutex{}, 0}, nil
}

//getdbhandle获取命名数据库的句柄
func (provider *VersionedDBProvider) GetDBHandle(dbName string) (statedb.VersionedDB, error) {
	provider.mux.Lock()
	defer provider.mux.Unlock()
	vdb := provider.databases[dbName]
	if vdb == nil {
		var err error
		vdb, err = newVersionedDB(provider.couchInstance, dbName)
		if err != nil {
			return nil, err
		}
		provider.databases[dbName] = vdb
	}
	return vdb, nil
}

//关闭关闭基础数据库实例
func (provider *VersionedDBProvider) Close() {
//沙发上不需要关门
}

//健康检查检查同伴的沙发实例是否健康
func (provider *VersionedDBProvider) HealthCheck(ctx context.Context) error {
	return provider.couchInstance.HealthCheck(ctx)
}

//版本数据库实现版本数据库接口
type VersionedDB struct {
	couchInstance      *couchdb.CouchInstance
metadataDB         *couchdb.CouchDatabase            //每个通道用于存储元数据（如保存点）的数据库。
chainName          string                            //链/通道的名称。
namespaceDBs       map[string]*couchdb.CouchDatabase //每个部署的链代码一个数据库。
committedDataCache *versionsCache                    //在块的批量处理期间用作本地缓存。
	verCacheLock       sync.RWMutex
	mux                sync.RWMutex
	lsccStateCache     *lsccStateCache
}

type lsccStateCache struct {
	cache   map[string]*statedb.VersionedValue
	rwMutex sync.RWMutex
}

func (l *lsccStateCache) getState(key string) *statedb.VersionedValue {
	l.rwMutex.RLock()
	defer l.rwMutex.RUnlock()

	if versionedValue, ok := l.cache[key]; ok {
		logger.Debugf("key:[%s] found in the lsccStateCache", key)
		return versionedValue
	}
	return nil
}

func (l *lsccStateCache) updateState(key string, value *statedb.VersionedValue) {
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()

	if _, ok := l.cache[key]; ok {
		logger.Debugf("key:[%s] is updated in lsccStateCache", key)
		l.cache[key] = value
	}
}

func (l *lsccStateCache) setState(key string, value *statedb.VersionedValue) {
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()

	if l.isCacheFull() {
		l.evictARandomEntry()
	}

	logger.Debugf("key:[%s] is stoed in lsccStateCache", key)
	l.cache[key] = value
}

func (l *lsccStateCache) isCacheFull() bool {
	return len(l.cache) == lsccCacheSize
}

func (l *lsccStateCache) evictARandomEntry() {
	for key := range l.cache {
		delete(l.cache, key)
		return
	}
}

//newVersionedDB构造versionedDB的实例
func newVersionedDB(couchInstance *couchdb.CouchInstance, dbName string) (*VersionedDB, error) {
//createCouchDatabase创建一个couchdb数据库对象，以及基础数据库（如果它不存在的话）
	chainName := dbName
	dbName = couchdb.ConstructMetadataDBName(dbName)

	metadataDB, err := couchdb.CreateCouchDatabase(couchInstance, dbName)
	if err != nil {
		return nil, err
	}
	namespaceDBMap := make(map[string]*couchdb.CouchDatabase)
	return &VersionedDB{
		couchInstance:      couchInstance,
		metadataDB:         metadataDB,
		chainName:          chainName,
		namespaceDBs:       namespaceDBMap,
		committedDataCache: newVersionCache(),
		lsccStateCache: &lsccStateCache{
			cache: make(map[string]*statedb.VersionedValue),
		},
	}, nil
}

//getNamespaceDBHandle gets the handle to a named chaincode database
func (vdb *VersionedDB) getNamespaceDBHandle(namespace string) (*couchdb.CouchDatabase, error) {
	vdb.mux.RLock()
	db := vdb.namespaceDBs[namespace]
	vdb.mux.RUnlock()
	if db != nil {
		return db, nil
	}
	namespaceDBName := couchdb.ConstructNamespaceDBName(vdb.chainName, namespace)
	vdb.mux.Lock()
	defer vdb.mux.Unlock()
	db = vdb.namespaceDBs[namespace]
	if db == nil {
		var err error
		db, err = couchdb.CreateCouchDatabase(vdb.couchInstance, namespaceDBName)
		if err != nil {
			return nil, err
		}
		vdb.namespaceDBs[namespace] = db
	}
	return db, nil
}

//processindexesforchaincodedeploy为指定的命名空间创建索引
func (vdb *VersionedDB) ProcessIndexesForChaincodeDeploy(namespace string, fileEntries []*ccprovider.TarFileEntry) error {
	db, err := vdb.getNamespaceDBHandle(namespace)
	if err != nil {
		return err
	}
	for _, fileEntry := range fileEntries {
		indexData := fileEntry.FileContent
		filename := fileEntry.FileHeader.Name
		_, err = db.CreateIndex(string(indexData))
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf(
				"error creating index from file [%s] for channel [%s]", filename, namespace))
		}
	}
	return nil
}

//GetDBType返回托管的StateDB
func (vdb *VersionedDB) GetDBType() string {
	return "couchdb"
}

//加载委托版本将提交的版本和修订号填充到缓存中。
//从couchdb进行大容量检索用于填充缓存。
//CommittedVersions缓存将用于读取集的状态验证
//RevisionNumbers缓存将在提交阶段用于CouchDB批量更新
func (vdb *VersionedDB) LoadCommittedVersions(keys []*statedb.CompositeKey) error {
	nsKeysMap := map[string][]string{}
	committedDataCache := newVersionCache()
	for _, compositeKey := range keys {
		ns, key := compositeKey.Namespace, compositeKey.Key
		committedDataCache.setVerAndRev(ns, key, nil, "")
		logger.Debugf("Load into version cache: %s~%s", ns, key)
		nsKeysMap[ns] = append(nsKeysMap[ns], key)
	}
	nsMetadataMap, err := vdb.retrieveMetadata(nsKeysMap)
	logger.Debugf("nsKeysMap=%s", nsKeysMap)
	logger.Debugf("nsMetadataMap=%s", nsMetadataMap)
	if err != nil {
		return err
	}
	for ns, nsMetadata := range nsMetadataMap {
		for _, keyMetadata := range nsMetadata {
//TODO-如果从数据库加载，为什么版本永远为零？
			if len(keyMetadata.Version) != 0 {
				version, _, err := decodeVersionAndMetadata(keyMetadata.Version)
				if err != nil {
					return err
				}
				committedDataCache.setVerAndRev(ns, keyMetadata.ID, version, keyMetadata.Rev)
			}
		}
	}
	vdb.verCacheLock.Lock()
	defer vdb.verCacheLock.Unlock()
	vdb.committedDataCache = committedDataCache
	return nil
}

//GetVersion在VersionedDB接口中实现方法
func (vdb *VersionedDB) GetVersion(namespace string, key string) (*version.Height, error) {
	returnVersion, keyFound := vdb.GetCachedVersion(namespace, key)
	if !keyFound {
//此if块仅在模拟期间执行，因为在提交期间
//在调用“getversion”之前，我们总是先调用“loadcommittedversions”
		vv, err := vdb.GetState(namespace, key)
		if err != nil || vv == nil {
			return nil, err
		}
		returnVersion = vv.Version
	}
	return returnVersion, nil
}

//getcachedversion从缓存返回版本。`loadcommittedversions`函数填充缓存
func (vdb *VersionedDB) GetCachedVersion(namespace string, key string) (*version.Height, bool) {
	logger.Debugf("Retrieving cached version: %s~%s", key, namespace)
	vdb.verCacheLock.RLock()
	defer vdb.verCacheLock.RUnlock()
	return vdb.committedDataCache.getVersion(namespace, key)
}

//validateKeyValue在versionedDB接口中实现方法
func (vdb *VersionedDB) ValidateKeyValue(key string, value []byte) error {
	err := validateKey(key)
	if err != nil {
		return err
	}
	return validateValue(value)
}

//BytesKeySupported在VersionedDB接口中实现方法
func (vdb *VersionedDB) BytesKeySupported() bool {
	return false
}

//GetState在VersionedDB接口中实现方法
func (vdb *VersionedDB) GetState(namespace string, key string) (*statedb.VersionedValue, error) {
	logger.Debugf("GetState(). ns=%s, key=%s", namespace, key)
	if namespace == "lscc" {
		if value := vdb.lsccStateCache.getState(key); value != nil {
			return value, nil
		}
	}

	db, err := vdb.getNamespaceDBHandle(namespace)
	if err != nil {
		return nil, err
	}
	couchDoc, _, err := db.ReadDoc(key)
	if err != nil {
		return nil, err
	}
	if couchDoc == nil {
		return nil, nil
	}
	kv, err := couchDocToKeyValue(couchDoc)
	if err != nil {
		return nil, err
	}

	if namespace == "lscc" {
		vdb.lsccStateCache.setState(key, kv.VersionedValue)
	}

	return kv.VersionedValue, nil
}

//GetStateMultipleKeys在VersionedDB接口中实现方法
func (vdb *VersionedDB) GetStateMultipleKeys(namespace string, keys []string) ([]*statedb.VersionedValue, error) {
	vals := make([]*statedb.VersionedValue, len(keys))
	for i, key := range keys {
		val, err := vdb.GetState(namespace, key)
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return vals, nil
}

//GetStateRangeScanIterator在VersionedDB接口中实现方法
//startkey包含在内
//endkey是独占的
func (vdb *VersionedDB) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (statedb.ResultsIterator, error) {
	return vdb.GetStateRangeScanIteratorWithMetadata(namespace, startKey, endKey, nil)
}

const optionBookmark = "bookmark"
const optionLimit = "limit"
const returnCount = "count"

//GetStateRangeScanIteratorWithMetadata在VersionedDB接口中实现方法
//startkey包含在内
//endkey是独占的
//元数据包含其他查询选项的映射
func (vdb *VersionedDB) GetStateRangeScanIteratorWithMetadata(namespace string, startKey string, endKey string, metadata map[string]interface{}) (statedb.QueryResultsIterator, error) {
	logger.Debugf("Entering GetStateRangeScanIteratorWithMetadata  namespace: %s  startKey: %s  endKey: %s  metadata: %v", namespace, startKey, endKey, metadata)
//Get the internalQueryLimit from core.yaml
	internalQueryLimit := int32(ledgerconfig.GetInternalQueryLimit())
	requestedLimit := int32(0)
//如果提供了元数据，请验证并应用选项
	if metadata != nil {
//验证元数据
		err := statedb.ValidateRangeMetadata(metadata)
		if err != nil {
			return nil, err
		}
		if limitOption, ok := metadata[optionLimit]; ok {
			requestedLimit = limitOption.(int32)
		}
	}
	db, err := vdb.getNamespaceDBHandle(namespace)
	if err != nil {
		return nil, err
	}
	return newQueryScanner(namespace, db, "", internalQueryLimit, requestedLimit, "", startKey, endKey)
}

func (scanner *queryScanner) getNextStateRangeScanResults() error {
	queryLimit := scanner.queryDefinition.internalQueryLimit
	if scanner.paginationInfo.requestedLimit > 0 {
		moreResultsNeeded := scanner.paginationInfo.requestedLimit - scanner.resultsInfo.totalRecordsReturned
		if moreResultsNeeded < scanner.queryDefinition.internalQueryLimit {
			queryLimit = moreResultsNeeded
		}
	}
	queryResult, nextStartKey, err := rangeScanFilterCouchInternalDocs(scanner.db,
		scanner.queryDefinition.startKey, scanner.queryDefinition.endKey, queryLimit)
	if err != nil {
		return err
	}
	scanner.resultsInfo.results = queryResult
	scanner.queryDefinition.startKey = nextStartKey
	scanner.paginationInfo.cursor = 0
	return nil
}

func rangeScanFilterCouchInternalDocs(db *couchdb.CouchDatabase,
	startKey, endKey string, queryLimit int32,
) ([]*couchdb.QueryResult, string, error) {
	var finalResults []*couchdb.QueryResult
	var finalNextStartKey string
	for {
		results, nextStartKey, err := db.ReadDocRange(startKey, endKey, queryLimit)
		if err != nil {
			logger.Debugf("Error calling ReadDocRange(): %s\n", err.Error())
			return nil, "", err
		}
		var filteredResults []*couchdb.QueryResult
		for _, doc := range results {
			if !isCouchInternalKey(doc.ID) {
				filteredResults = append(filteredResults, doc)
			}
		}

		finalResults = append(finalResults, filteredResults...)
		finalNextStartKey = nextStartKey
		queryLimit = int32(len(results) - len(filteredResults))
		if queryLimit == 0 || finalNextStartKey == "" {
			break
		}
		startKey = finalNextStartKey
	}
	var err error
	for i := 0; isCouchInternalKey(finalNextStartKey); i++ {
		_, finalNextStartKey, err = db.ReadDocRange(finalNextStartKey, endKey, 1)
		logger.Debugf("i=%d, finalNextStartKey=%s", i, finalNextStartKey)
		if err != nil {
			return nil, "", err
		}
	}
	return finalResults, finalNextStartKey, nil
}

func isCouchInternalKey(key string) bool {
	return len(key) != 0 && key[0] == '_'
}

//ExecuteQuery在VersionedDB接口中实现方法
func (vdb *VersionedDB) ExecuteQuery(namespace, query string) (statedb.ResultsIterator, error) {
	queryResult, err := vdb.ExecuteQueryWithMetadata(namespace, query, nil)
	if err != nil {
		return nil, err
	}
	return queryResult, nil
}

//ExecuteEqueryWithMetadata在VersionedDB接口中实现方法
func (vdb *VersionedDB) ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (statedb.QueryResultsIterator, error) {
	logger.Debugf("Entering ExecuteQueryWithMetadata  namespace: %s,  query: %s,  metadata: %v", namespace, query, metadata)
//从core.yaml获取querylimit
	internalQueryLimit := int32(ledgerconfig.GetInternalQueryLimit())
	bookmark := ""
	requestedLimit := int32(0)
//如果提供了元数据，则验证并设置提供的选项
	if metadata != nil {
		err := validateQueryMetadata(metadata)
		if err != nil {
			return nil, err
		}
		if limitOption, ok := metadata[optionLimit]; ok {
			requestedLimit = limitOption.(int32)
		}
		if bookmarkOption, ok := metadata[optionBookmark]; ok {
			bookmark = bookmarkOption.(string)
		}
	}
	queryString, err := applyAdditionalQueryOptions(query, internalQueryLimit, bookmark)
	if err != nil {
		logger.Errorf("Error calling applyAdditionalQueryOptions(): %s", err.Error())
		return nil, err
	}
	db, err := vdb.getNamespaceDBHandle(namespace)
	if err != nil {
		return nil, err
	}
	return newQueryScanner(namespace, db, queryString, internalQueryLimit, requestedLimit, bookmark, "", "")
}

//ExecuteEqueryWithBookmark使用书签执行“分页”查询，此方法允许
//未返回新查询迭代器的分页查询
func (scanner *queryScanner) executeQueryWithBookmark() error {
	queryLimit := scanner.queryDefinition.internalQueryLimit
	if scanner.paginationInfo.requestedLimit > 0 {
		if scanner.paginationInfo.requestedLimit-scanner.resultsInfo.totalRecordsReturned < scanner.queryDefinition.internalQueryLimit {
			queryLimit = scanner.paginationInfo.requestedLimit - scanner.resultsInfo.totalRecordsReturned
		}
	}
	queryString, err := applyAdditionalQueryOptions(scanner.queryDefinition.query,
		queryLimit, scanner.paginationInfo.bookmark)
	if err != nil {
		logger.Debugf("Error calling applyAdditionalQueryOptions(): %s\n", err.Error())
		return err
	}
	queryResult, bookmark, err := scanner.db.QueryDocuments(queryString)
	if err != nil {
		logger.Debugf("Error calling QueryDocuments(): %s\n", err.Error())
		return err
	}
	scanner.resultsInfo.results = queryResult
	scanner.paginationInfo.bookmark = bookmark
	scanner.paginationInfo.cursor = 0
	return nil
}

func validateQueryMetadata(metadata map[string]interface{}) error {
	for key, keyVal := range metadata {
		switch key {
		case optionBookmark:
//验证书签是否为字符串
			if _, ok := keyVal.(string); ok {
				continue
			}
			return fmt.Errorf("Invalid entry, \"bookmark\" must be a string")

		case optionLimit:
//验证限制是否为整数
			if _, ok := keyVal.(int32); ok {
				continue
			}
			return fmt.Errorf("Invalid entry, \"limit\" must be an int32")

		default:
			return fmt.Errorf("Invalid entry, option %s not recognized", key)
		}
	}
	return nil
}

//ApplyUpdates在VersionedDB接口中实现方法
func (vdb *VersionedDB) ApplyUpdates(updates *statedb.UpdateBatch, height *version.Height) error {
//关于https://jira.hyperledger.org/browse/fab-8622的说明
//函数'apply update可分为三个函数。每一个都执行以下三个阶段中的一个。
//只有阶段2需要写锁。

//阶段1-PrepareForUpdates-db以底层db的形式转换给定批
//把它留在记忆中
	var updateBatches []batch
	var err error
	if updateBatches, err = vdb.buildCommitters(updates); err != nil {
		return err
	}
//阶段2-应用程序更新将更改推送到数据库
	if err = executeBatches(updateBatches); err != nil {
		return err
	}

//stgae 3-postupdateprocessing-flush和record savepoint。
	namespaces := updates.GetUpdatedNamespaces()
//在给定高度记录保存点
	if err = vdb.ensureFullCommitAndRecordSavepoint(height, namespaces); err != nil {
		logger.Errorf("Error during recordSavepoint: %s", err.Error())
		return err
	}

	lsccUpdates := updates.GetUpdates("lscc")
	for key, value := range lsccUpdates {
		vdb.lsccStateCache.updateState(key, value)
	}

	return nil
}

//ClearCachedVersions清除委员会版本和修订号
func (vdb *VersionedDB) ClearCachedVersions() {
	logger.Debugf("Clear Cache")
	vdb.verCacheLock.Lock()
	defer vdb.verCacheLock.Unlock()
	vdb.committedDataCache = newVersionCache()
}

//在版本数据库接口中打开实现方法
func (vdb *VersionedDB) Open() error {
//no need to open db since a shared couch instance is used
	return nil
}

//Close在VersionedDB接口中实现方法
func (vdb *VersionedDB) Close() {
//不需要关闭数据库，因为使用了共享的coach实例
}

//couchdb的保存点docid（key）
const savepointDocID = "statedb_savepoint"

//确保refullcommitandrecordsavepoint将所有dbs（对应于“namespaces”）刷新到磁盘
//并在元数据数据库中记录保存点。
//coach在集群或分片设置中并行写入，并且不保证排序。
//Hence we need to fence the savepoint with sync. So ensure_full_commit on all updated
//在保存点之前调用命名空间dbs以确保刷新所有块写入。保存点
//将自身刷新到元数据数据库。
func (vdb *VersionedDB) ensureFullCommitAndRecordSavepoint(height *version.Height, namespaces []string) error {
//确保完全提交将更新的命名空间上的所有更改刷新到磁盘
//命名空间还包括空的命名空间，它只是元数据数据库。
	var dbs []*couchdb.CouchDatabase
	for _, ns := range namespaces {
		db, err := vdb.getNamespaceDBHandle(ns)
		if err != nil {
			return err
		}
		dbs = append(dbs, db)
	}
	if err := vdb.ensureFullCommit(dbs); err != nil {
		return err
	}

//如果给定高度为零，则表示我们正在提交旧块的pvt数据。
//在这种情况下，我们不应该存储用于恢复的保存点。最新更新的DoldBlockList
//在pvtstore中用作pvt数据的保存点。
	if height == nil {
		return nil
	}

//构造保存点文档并保存
	savepointCouchDoc, err := encodeSavepoint(height)
	if err != nil {
		return err
	}
	_, err = vdb.metadataDB.SaveDoc(savepointDocID, "", savepointCouchDoc)
	if err != nil {
		logger.Errorf("Failed to save the savepoint to DB %s", err.Error())
		return err
	}
//注意：在存储保存点之后，确保完全提交metadatadb是不必要的。
//因为couchdb定期（每1秒）将状态同步到磁盘。如果之前对等方失败
//将保存点同步到磁盘，将启动分类帐恢复过程以确保一致性。
//对等重新启动时，在CouchDB和块存储之间
	return nil
}

//GetLatestSavePoint在VersionedDB接口中实现方法
func (vdb *VersionedDB) GetLatestSavePoint() (*version.Height, error) {
	var err error
	couchDoc, _, err := vdb.metadataDB.ReadDoc(savepointDocID)
	if err != nil {
		logger.Errorf("Failed to read savepoint data %s", err.Error())
		return nil, err
	}
//未找到readDoc（）（404）将导致nil响应，在这些情况下返回高度nil
	if couchDoc == nil || couchDoc.JSONValue == nil {
		return nil, nil
	}
	return decodeSavepoint(couchDoc)
}

//ApplyAdditionalQueryOptions将向查询处理所需的查询添加其他字段
func applyAdditionalQueryOptions(queryString string, queryLimit int32, queryBookmark string) (string, error) {
	const jsonQueryFields = "fields"
	const jsonQueryLimit = "limit"
	const jsonQueryBookmark = "bookmark"
//为查询json创建通用映射
	jsonQueryMap := make(map[string]interface{})
//将选择器JSON取消标记到通用映射中
	decoder := json.NewDecoder(bytes.NewBuffer([]byte(queryString)))
	decoder.UseNumber()
	err := decoder.Decode(&jsonQueryMap)
	if err != nil {
		return "", err
	}
	if fieldsJSONArray, ok := jsonQueryMap[jsonQueryFields]; ok {
		switch fieldsJSONArray.(type) {
		case []interface{}:
//添加“_id”和“version”字段，默认情况下需要这些字段
			jsonQueryMap[jsonQueryFields] = append(fieldsJSONArray.([]interface{}),
				idField, versionField)
		default:
			return "", errors.New("fields definition must be an array")
		}
	}
//添加限制
//这将覆盖查询中传递的任何限制。
//尚不支持显式分页。
	jsonQueryMap[jsonQueryLimit] = queryLimit
//添加书签（如果提供）
	if queryBookmark != "" {
		jsonQueryMap[jsonQueryBookmark] = queryBookmark
	}
//整理更新的JSON查询
	editedQuery, err := json.Marshal(jsonQueryMap)
	if err != nil {
		return "", err
	}
	logger.Debugf("Rewritten query: %s", editedQuery)
	return string(editedQuery), nil
}

type queryScanner struct {
	namespace       string
	db              *couchdb.CouchDatabase
	queryDefinition *queryDefinition
	paginationInfo  *paginationInfo
	resultsInfo     *resultsInfo
}

type queryDefinition struct {
	startKey           string
	endKey             string
	query              string
	internalQueryLimit int32
}

type paginationInfo struct {
	cursor         int32
	requestedLimit int32
	bookmark       string
}

type resultsInfo struct {
	totalRecordsReturned int32
	results              []*couchdb.QueryResult
}

func newQueryScanner(namespace string, db *couchdb.CouchDatabase, query string, internalQueryLimit,
	limit int32, bookmark, startKey, endKey string) (*queryScanner, error) {
	scanner := &queryScanner{namespace, db, &queryDefinition{startKey, endKey, query, internalQueryLimit}, &paginationInfo{-1, limit, bookmark}, &resultsInfo{0, nil}}
	var err error
//定义查询，然后执行查询并返回记录和书签
	if scanner.queryDefinition.query != "" {
		err = scanner.executeQueryWithBookmark()
	} else {
		err = scanner.getNextStateRangeScanResults()
	}
	if err != nil {
		return nil, err
	}
	scanner.paginationInfo.cursor = -1
	return scanner, nil
}

func (scanner *queryScanner) Next() (statedb.QueryResult, error) {
//无结果测试案例
	if len(scanner.resultsInfo.results) == 0 {
		return nil, nil
	}
//增加光标
	scanner.paginationInfo.cursor++
//检查是否需要附加记录
//如果光标超过InternalQueryLimit，则重新查询
	if scanner.paginationInfo.cursor >= scanner.queryDefinition.internalQueryLimit {
		var err error
//定义查询，然后执行查询并返回记录和书签
		if scanner.queryDefinition.query != "" {
			err = scanner.executeQueryWithBookmark()
		} else {
			err = scanner.getNextStateRangeScanResults()
		}
		if err != nil {
			return nil, err
		}
//如果没有更多结果，则返回
		if len(scanner.resultsInfo.results) == 0 {
			return nil, nil
		}
	}
//如果光标大于或等于结果记录数，则返回
	if scanner.paginationInfo.cursor >= int32(len(scanner.resultsInfo.results)) {
		return nil, nil
	}
	selectedResultRecord := scanner.resultsInfo.results[scanner.paginationInfo.cursor]
	key := selectedResultRecord.ID
//从couchdb json中删除保留字段并返回值和版本
	kv, err := couchDocToKeyValue(&couchdb.CouchDoc{JSONValue: selectedResultRecord.Value, Attachments: selectedResultRecord.Attachments})
	if err != nil {
		return nil, err
	}
	scanner.resultsInfo.totalRecordsReturned++
	return &statedb.VersionedKV{
		CompositeKey:   statedb.CompositeKey{Namespace: scanner.namespace, Key: key},
		VersionedValue: *kv.VersionedValue}, nil
}

func (scanner *queryScanner) Close() {
	scanner = nil
}

func (scanner *queryScanner) GetBookmarkAndClose() string {
	retval := ""
	if scanner.queryDefinition.query != "" {
		retval = scanner.paginationInfo.bookmark
	} else {
		retval = scanner.queryDefinition.startKey
	}
	scanner.Close()
	return retval
}
