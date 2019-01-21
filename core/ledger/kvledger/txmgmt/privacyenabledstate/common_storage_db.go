
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


package privacyenabledstate

import (
	"encoding/base64"
	"strings"

	"github.com/hyperledger/fabric-lib-go/healthz"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/stateleveldb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("privacyenabledstate")

const (
	nsJoiner       = "$$"
	pvtDataPrefix  = "p"
	hashDataPrefix = "h"
)

//CommonStorageDBProvider实现接口DBProvider
type CommonStorageDBProvider struct {
	statedb.VersionedDBProvider
	HealthCheckRegistry ledger.HealthCheckRegistry
	bookkeepingProvider bookkeeping.Provider
}

//newcommonstoragedbprovider构造dbprovider的实例
func NewCommonStorageDBProvider(bookkeeperProvider bookkeeping.Provider, metricsProvider metrics.Provider, healthCheckRegistry ledger.HealthCheckRegistry) (DBProvider, error) {
	var vdbProvider statedb.VersionedDBProvider
	var err error
	if ledgerconfig.IsCouchDBEnabled() {
		if vdbProvider, err = statecouchdb.NewVersionedDBProvider(metricsProvider); err != nil {
			return nil, err
		}
	} else {
		vdbProvider = stateleveldb.NewVersionedDBProvider()
	}

	dbProvider := &CommonStorageDBProvider{vdbProvider, healthCheckRegistry, bookkeeperProvider}

	err = dbProvider.RegisterHealthChecker()
	if err != nil {
		return nil, err
	}

	return dbProvider, nil
}

func (p *CommonStorageDBProvider) RegisterHealthChecker() error {
	if healthChecker, ok := p.VersionedDBProvider.(healthz.HealthChecker); ok {
		return p.HealthCheckRegistry.RegisterChecker("couchdb", healthChecker)
	}
	return nil
}

//GetDBHandle从接口DBProvider实现函数
func (p *CommonStorageDBProvider) GetDBHandle(id string) (DB, error) {
	vdb, err := p.VersionedDBProvider.GetDBHandle(id)
	if err != nil {
		return nil, err
	}
	bookkeeper := p.bookkeepingProvider.GetDBHandle(id, bookkeeping.MetadataPresenceIndicator)
	metadataHint := newMetadataHint(bookkeeper)
	return NewCommonStorageDB(vdb, id, metadataHint)
}

//close从接口dbprovider实现函数
func (p *CommonStorageDBProvider) Close() {
	p.VersionedDBProvider.Close()
}

//CommonStorageDB实现接口DB。此实现使用单个数据库来维护
//公共和私人数据
type CommonStorageDB struct {
	statedb.VersionedDB
	metadataHint *metadataHint
}

//newcommonstoragedb包装一个versioneddb实例。公共数据由封装的版本数据库直接管理。
//为了管理散列数据和私有数据，此实现在包装的数据库中创建单独的命名空间。
func NewCommonStorageDB(vdb statedb.VersionedDB, ledgerid string, metadataHint *metadataHint) (DB, error) {
	return &CommonStorageDB{vdb, metadataHint}, nil
}

//isBulkOptimizable在接口db中实现相应的功能
func (s *CommonStorageDB) IsBulkOptimizable() bool {
	_, ok := s.VersionedDB.(statedb.BulkOptimizable)
	return ok
}

//LoadCommittedVersionsOfPubandHashedKeys在接口DB中实现了相应的功能
func (s *CommonStorageDB) LoadCommittedVersionsOfPubAndHashedKeys(pubKeys []*statedb.CompositeKey,
	hashedKeys []*HashedCompositeKey) error {

	bulkOptimizable, ok := s.VersionedDB.(statedb.BulkOptimizable)
	if !ok {
		return nil
	}
//在这里，hashedkeys被合并到pubkeys中，以获得组合加载的组合键集。
	for _, key := range hashedKeys {
		ns := deriveHashedDataNs(key.Namespace, key.CollectionName)
//无需检查重复项，因为哈希键位于单独的命名空间中。
		var keyHashStr string
		if !s.BytesKeySupported() {
			keyHashStr = base64.StdEncoding.EncodeToString([]byte(key.KeyHash))
		} else {
			keyHashStr = key.KeyHash
		}
		pubKeys = append(pubKeys, &statedb.CompositeKey{
			Namespace: ns,
			Key:       keyHashStr,
		})
	}

	err := bulkOptimizable.LoadCommittedVersions(pubKeys)
	if err != nil {
		return err
	}

	return nil
}

//ClearCachedVersions在接口数据库中实现相应的功能
func (s *CommonStorageDB) ClearCachedVersions() {
	bulkOptimizable, ok := s.VersionedDB.(statedb.BulkOptimizable)
	if ok {
		bulkOptimizable.ClearCachedVersions()
	}
}

//getchaincodeEventListener在接口db中实现相应功能
func (s *CommonStorageDB) GetChaincodeEventListener() cceventmgmt.ChaincodeLifecycleEventListener {
	_, ok := s.VersionedDB.(statedb.IndexCapable)
	if ok {
		return s
	}
	return nil
}

//getprivatedata在接口db中实现相应的函数
func (s *CommonStorageDB) GetPrivateData(namespace, collection, key string) (*statedb.VersionedValue, error) {
	return s.GetState(derivePvtDataNs(namespace, collection), key)
}

//getValueHash在接口db中实现相应的函数
func (s *CommonStorageDB) GetValueHash(namespace, collection string, keyHash []byte) (*statedb.VersionedValue, error) {
	keyHashStr := string(keyHash)
	if !s.BytesKeySupported() {
		keyHashStr = base64.StdEncoding.EncodeToString(keyHash)
	}
	return s.GetState(deriveHashedDataNs(namespace, collection), keyHashStr)
}

//GetKeyHashVersion在接口数据库中实现相应的函数
func (s *CommonStorageDB) GetKeyHashVersion(namespace, collection string, keyHash []byte) (*version.Height, error) {
	keyHashStr := string(keyHash)
	if !s.BytesKeySupported() {
		keyHashStr = base64.StdEncoding.EncodeToString(keyHash)
	}
	return s.GetVersion(deriveHashedDataNs(namespace, collection), keyHashStr)
}

//getcachedKeyHashVersion从缓存中检索keyHash版本
func (s *CommonStorageDB) GetCachedKeyHashVersion(namespace, collection string, keyHash []byte) (*version.Height, bool) {
	bulkOptimizable, ok := s.VersionedDB.(statedb.BulkOptimizable)
	if !ok {
		return nil, false
	}

	keyHashStr := string(keyHash)
	if !s.BytesKeySupported() {
		keyHashStr = base64.StdEncoding.EncodeToString(keyHash)
	}
	return bulkOptimizable.GetCachedVersion(deriveHashedDataNs(namespace, collection), keyHashStr)
}

//getprivatedatamultiplekeys在接口db中实现相应的功能
func (s *CommonStorageDB) GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([]*statedb.VersionedValue, error) {
	return s.GetStateMultipleKeys(derivePvtDataNs(namespace, collection), keys)
}

//getprivatedatarangescaniterator在接口db中实现相应的函数
func (s *CommonStorageDB) GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (statedb.ResultsIterator, error) {
	return s.GetStateRangeScanIterator(derivePvtDataNs(namespace, collection), startKey, endKey)
}

//ExecuteEqueryOnPrivateData在接口数据库中实现相应功能
func (s CommonStorageDB) ExecuteQueryOnPrivateData(namespace, collection, query string) (statedb.ResultsIterator, error) {
	return s.ExecuteQuery(derivePvtDataNs(namespace, collection), query)
}

//ApplyUpdates重写statedb.versioneddb中的函数，并引发相应的错误消息
//否则，在代码的某个地方，使用此函数可能只会导致更新公共数据。
func (s *CommonStorageDB) ApplyUpdates(batch *statedb.UpdateBatch, height *version.Height) error {
	return errors.New("this function should not be invoked on this type. Please invoke function ApplyPrivacyAwareUpdates")
}

//applyprivacyawareupdates在接口db中实现相应功能
func (s *CommonStorageDB) ApplyPrivacyAwareUpdates(updates *UpdateBatch, height *version.Height) error {
//CombinedUpdates包括对公共数据库和私有数据库的更新，它们由单独的命名空间进行分区。
	combinedUpdates := updates.PubUpdates
	addPvtUpdates(combinedUpdates, updates.PvtUpdates)
	addHashedUpdates(combinedUpdates, updates.HashUpdates, !s.BytesKeySupported())
	s.metadataHint.setMetadataUsedFlag(updates)
	return s.VersionedDB.ApplyUpdates(combinedUpdates.UpdateBatch, height)
}

//getStateMetadata在接口db中实现相应的函数。此实现提供
//一种优化，以便跟踪命名空间是否从未为任何
//其项，返回值“nil”，而不转到db。这是有意调用的
//在验证和提交路径中。这样可以避免链式代码支付不必要的性能。
//如果他们不使用利用元数据的功能（如关键级别的认可），则将受到惩罚，
func (s *CommonStorageDB) GetStateMetadata(namespace, key string) ([]byte, error) {
	if !s.metadataHint.metadataEverUsedFor(namespace) {
		return nil, nil
	}
	vv, err := s.GetState(namespace, key)
	if err != nil || vv == nil {
		return nil, err
	}
	return vv.Metadata, nil
}

//getprivatedatametadatabyhash在接口db中实现相应的函数。有关更多详细信息，请参阅
//类似函数“getStateMetadata”的说明
func (s *CommonStorageDB) GetPrivateDataMetadataByHash(namespace, collection string, keyHash []byte) ([]byte, error) {
	if !s.metadataHint.metadataEverUsedFor(namespace) {
		return nil, nil
	}
	vv, err := s.GetValueHash(namespace, collection, keyHash)
	if err != nil || vv == nil {
		return nil, err
	}
	return vv.Metadata, nil
}

//handlechaincodedeploy初始化与命名空间关联的数据库的数据库项目
//此函数可以动态地抑制在couchdb上创建索引期间发生的错误。
//这是因为，在目前的代码中，由于couchdb交互，我们不区分错误。
//以及由于索引文件不好而导致的错误——后者是管理员无法修复的。注意错误抑制
//是可接受的，因为对等方可以在没有索引的情况下继续执行提交角色。但是，执行链码查询
//可能会受到影响，直到安装并实例化具有固定索引的新链码
func (s *CommonStorageDB) HandleChaincodeDeploy(chaincodeDefinition *cceventmgmt.ChaincodeDefinition, dbArtifactsTar []byte) error {
//检查是否实现了可索引的接口
	indexCapable, ok := s.VersionedDB.(statedb.IndexCapable)
	if !ok {
		return nil
	}
	if chaincodeDefinition == nil {
		return errors.New("chaincode definition not found while creating couchdb index")
	}
	dbArtifacts, err := ccprovider.ExtractFileEntries(dbArtifactsTar, indexCapable.GetDBType())
	if err != nil {
		logger.Errorf("Index creation: error extracting db artifacts from tar for chaincode [%s]: %s", chaincodeDefinition.Name, err)
		return nil
	}

	collectionConfigMap, err := extractCollectionNames(chaincodeDefinition)
	if err != nil {
		logger.Errorf("Error while retrieving collection config for chaincode=[%s]: %s",
			chaincodeDefinition.Name, err)
		return nil
	}

	for directoryPath, archiveDirectoryEntries := range dbArtifacts {
//拆分目录名
		directoryPathArray := strings.Split(directoryPath, "/")
//处理链的索引
		if directoryPathArray[3] == "indexes" {
			err := indexCapable.ProcessIndexesForChaincodeDeploy(chaincodeDefinition.Name, archiveDirectoryEntries)
			if err != nil {
				logger.Errorf("Error processing index for chaincode [%s]: %s", chaincodeDefinition.Name, err)
			}
			continue
		}
//检查集合的索引目录
		if directoryPathArray[3] == "collections" && directoryPathArray[5] == "indexes" {
			collectionName := directoryPathArray[4]
			_, ok := collectionConfigMap[collectionName]
			if !ok {
				logger.Errorf("Error processing index for chaincode [%s]: cannot create an index for an undefined collection=[%s]", chaincodeDefinition.Name, collectionName)
			} else {
				err := indexCapable.ProcessIndexesForChaincodeDeploy(derivePvtDataNs(chaincodeDefinition.Name, collectionName),
					archiveDirectoryEntries)
				if err != nil {
					logger.Errorf("Error processing collection index for chaincode [%s]: %s", chaincodeDefinition.Name, err)
				}
			}
		}
	}
	return nil
}

//chaincodedeploydone是couchdb state impl的noop
func (s *CommonStorageDB) ChaincodeDeployDone(succeeded bool) {
//诺普
}

func derivePvtDataNs(namespace, collection string) string {
	return namespace + nsJoiner + pvtDataPrefix + collection
}

func deriveHashedDataNs(namespace, collection string) string {
	return namespace + nsJoiner + hashDataPrefix + collection
}

func addPvtUpdates(pubUpdateBatch *PubUpdateBatch, pvtUpdateBatch *PvtUpdateBatch) {
	for ns, nsBatch := range pvtUpdateBatch.UpdateMap {
		for _, coll := range nsBatch.GetCollectionNames() {
			for key, vv := range nsBatch.GetUpdates(coll) {
				pubUpdateBatch.Update(derivePvtDataNs(ns, coll), key, vv)
			}
		}
	}
}

func addHashedUpdates(pubUpdateBatch *PubUpdateBatch, hashedUpdateBatch *HashedUpdateBatch, base64Key bool) {
	for ns, nsBatch := range hashedUpdateBatch.UpdateMap {
		for _, coll := range nsBatch.GetCollectionNames() {
			for key, vv := range nsBatch.GetUpdates(coll) {
				if base64Key {
					key = base64.StdEncoding.EncodeToString([]byte(key))
				}
				pubUpdateBatch.Update(deriveHashedDataNs(ns, coll), key, vv)
			}
		}
	}
}

func extractCollectionNames(chaincodeDefinition *cceventmgmt.ChaincodeDefinition) (map[string]bool, error) {
	collectionConfigs := chaincodeDefinition.CollectionConfigs
	collectionConfigsMap := make(map[string]bool)
	if collectionConfigs != nil {
		for _, config := range collectionConfigs.Config {
			sConfig := config.GetStaticCollectionConfig()
			if sConfig == nil {
				continue
			}
			collectionConfigsMap[sConfig.Name] = true
		}
	}
	return collectionConfigsMap, nil
}
