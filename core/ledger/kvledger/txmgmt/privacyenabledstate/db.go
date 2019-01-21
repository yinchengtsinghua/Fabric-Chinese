
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
	"fmt"

	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
)

//dbprovider提供pvtversioneddb的句柄
type DBProvider interface {
//getdbhandle返回pvtversioneddb的句柄
	GetDBHandle(id string) (DB, error)
//close关闭所有pvversioneddb实例并释放versioneddbprovider持有的任何资源
	Close()
}

//数据库扩展版本数据库接口。此接口提供用于管理私有数据状态的附加功能
type DB interface {
	statedb.VersionedDB
	IsBulkOptimizable() bool
	LoadCommittedVersionsOfPubAndHashedKeys(pubKeys []*statedb.CompositeKey, hashedKeys []*HashedCompositeKey) error
	GetCachedKeyHashVersion(namespace, collection string, keyHash []byte) (*version.Height, bool)
	ClearCachedVersions()
	GetChaincodeEventListener() cceventmgmt.ChaincodeLifecycleEventListener
	GetPrivateData(namespace, collection, key string) (*statedb.VersionedValue, error)
	GetValueHash(namespace, collection string, keyHash []byte) (*statedb.VersionedValue, error)
	GetKeyHashVersion(namespace, collection string, keyHash []byte) (*version.Height, error)
	GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([]*statedb.VersionedValue, error)
	GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (statedb.ResultsIterator, error)
	GetStateMetadata(namespace, key string) ([]byte, error)
	GetPrivateDataMetadataByHash(namespace, collection string, keyHash []byte) ([]byte, error)
	ExecuteQueryOnPrivateData(namespace, collection, query string) (statedb.ResultsIterator, error)
	ApplyPrivacyAwareUpdates(updates *UpdateBatch, height *version.Height) error
}

//pvtDataCompositeKey包含命名空间、集合名和密钥组件
type PvtdataCompositeKey struct {
	Namespace      string
	CollectionName string
	Key            string
}

//HashedCompositeKey包含命名空间、CollectionName和KeyHash组件
type HashedCompositeKey struct {
	Namespace      string
	CollectionName string
	KeyHash        string
}

//pvtkvwrite包含密钥、isDelete、值和版本组件
type PvtKVWrite struct {
	Key      string
	IsDelete bool
	Value    []byte
	Version  *version.Height
}

//updateBatch封装对公共、私有和哈希数据的更新。
//这将包含一组一致的更新
type UpdateBatch struct {
	PubUpdates  *PubUpdateBatch
	HashUpdates *HashedUpdateBatch
	PvtUpdates  *PvtUpdateBatch
}

//pubUpdateBatch包含公共数据的更新
type PubUpdateBatch struct {
	*statedb.UpdateBatch
}

//hashedupdatebatch包含对私有数据哈希的更新
type HashedUpdateBatch struct {
	UpdateMap
}

//pvtupdateBatch包含私有数据的更新
type PvtUpdateBatch struct {
	UpdateMap
}

//updateMap维护tuple<namespace，updatesForNamespace>
type UpdateMap map[string]nsBatch

//nsbatch包含与一个命名空间相关的更新
type nsBatch struct {
	*statedb.UpdateBatch
}

//newupdateBatch创建并清空updateBatch
func NewUpdateBatch() *UpdateBatch {
	return &UpdateBatch{NewPubUpdateBatch(), NewHashedUpdateBatch(), NewPvtUpdateBatch()}
}

//NewPubUpdateBatch创建空的PubUpdateBatch
func NewPubUpdateBatch() *PubUpdateBatch {
	return &PubUpdateBatch{statedb.NewUpdateBatch()}
}

//NewHashEdupDateBatch创建空的HashEdupDateBatch
func NewHashedUpdateBatch() *HashedUpdateBatch {
	return &HashedUpdateBatch{make(map[string]nsBatch)}
}

//newpvtupdateBatch创建空pvtupdateBatch
func NewPvtUpdateBatch() *PvtUpdateBatch {
	return &PvtUpdateBatch{make(map[string]nsBatch)}
}

//如果存在任何更新，isEmpty将返回true
func (b UpdateMap) IsEmpty() bool {
	return len(b) == 0
}

//Put为给定的命名空间和集合名称组合设置批处理中的值
func (b UpdateMap) Put(ns, coll, key string, value []byte, version *version.Height) {
	b.PutValAndMetadata(ns, coll, key, value, nil, version)
}

//PutValandMetadata添加了一个带有值和元数据的键
func (b UpdateMap) PutValAndMetadata(ns, coll, key string, value []byte, metadata []byte, version *version.Height) {
	b.getOrCreateNsBatch(ns).PutValAndMetadata(coll, key, value, metadata, version)
}

//delete为给定的命名空间和集合名称组合在批处理中添加删除标记
func (b UpdateMap) Delete(ns, coll, key string, version *version.Height) {
	b.getOrCreateNsBatch(ns).Delete(coll, key, version)
}

//get从批中检索给定命名空间和集合名称组合的值
func (b UpdateMap) Get(ns, coll, key string) *statedb.VersionedValue {
	nsPvtBatch, ok := b[ns]
	if !ok {
		return nil
	}
	return nsPvtBatch.Get(coll, key)
}

//如果批处理中存在给定的<ns，coll，key>元组，则contains返回true
func (b UpdateMap) Contains(ns, coll, key string) bool {
	nsBatch, ok := b[ns]
	if !ok {
		return false
	}
	return nsBatch.Exists(coll, key)
}

func (nsb nsBatch) GetCollectionNames() []string {
	return nsb.GetUpdatedNamespaces()
}

func (b UpdateMap) getOrCreateNsBatch(ns string) nsBatch {
	batch, ok := b[ns]
	if !ok {
		batch = nsBatch{statedb.NewUpdateBatch()}
		b[ns] = batch
	}
	return batch
}

//如果批处理中存在给定的<ns、coll、keyhash>元组，则包含返回true
func (h HashedUpdateBatch) Contains(ns, coll string, keyHash []byte) bool {
	return h.UpdateMap.Contains(ns, coll, string(keyHash))
}

//Put重写updateMap中的函数，以允许键为[]字节而不是字符串
func (h HashedUpdateBatch) Put(ns, coll string, key []byte, value []byte, version *version.Height) {
	h.PutValHashAndMetadata(ns, coll, key, value, nil, version)
}

//putvalhashandmetadata添加一个带有值和元数据的键
//要引入新函数以限制重构。稍后在单独的CR中，应删除上面的“Put”函数
func (h HashedUpdateBatch) PutValHashAndMetadata(ns, coll string, key []byte, value []byte, metadata []byte, version *version.Height) {
	h.UpdateMap.PutValAndMetadata(ns, coll, string(key), value, metadata, version)
}

//delete重写updateMap中的函数，以允许键为[]字节而不是字符串
func (h HashedUpdateBatch) Delete(ns, coll string, key []byte, version *version.Height) {
	h.UpdateMap.Delete(ns, coll, string(key), version)
}

//若要组合键映射，请以单个映射的形式重新排列更新批处理数据。
func (h HashedUpdateBatch) ToCompositeKeyMap() map[HashedCompositeKey]*statedb.VersionedValue {
	m := make(map[HashedCompositeKey]*statedb.VersionedValue)
	for ns, nsBatch := range h.UpdateMap {
		for _, coll := range nsBatch.GetCollectionNames() {
			for key, vv := range nsBatch.GetUpdates(coll) {
				m[HashedCompositeKey{ns, coll, key}] = vv
			}
		}
	}
	return m
}

//pvtdatacompositekey map是pvtdatacompositekey到versionedValue的映射
type PvtdataCompositeKeyMap map[PvtdataCompositeKey]*statedb.VersionedValue

//若要组合键映射，请以单个映射的形式重新排列更新批处理数据。
func (p PvtUpdateBatch) ToCompositeKeyMap() PvtdataCompositeKeyMap {
	m := make(PvtdataCompositeKeyMap)
	for ns, nsBatch := range p.UpdateMap {
		for _, coll := range nsBatch.GetCollectionNames() {
			for key, vv := range nsBatch.GetUpdates(coll) {
				m[PvtdataCompositeKey{ns, coll, key}] = vv
			}
		}
	}
	return m
}

//字符串返回哈希复合键的打印友好形式
func (hck *HashedCompositeKey) String() string {
	return fmt.Sprintf("ns=%s, collection=%s, keyHash=%x", hck.Namespace, hck.CollectionName, hck.KeyHash)
}
