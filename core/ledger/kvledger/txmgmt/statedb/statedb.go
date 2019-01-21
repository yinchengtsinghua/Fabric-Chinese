
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


package statedb

import (
	"fmt"
	"sort"

	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util"
)

//go:生成伪造者-o mock/results_iterator.go-forke name results iterator。制作人
//go:generate counterfeiter -o mock/versioned_db.go -fake-name VersionedDB . 版本数据库

//VersionedDBProvider提供版本化数据库的实例
type VersionedDBProvider interface {
//getdbhandle将句柄返回到versioneddb
	GetDBHandle(id string) (VersionedDB, error)
//Close closes all the VersionedDB instances and releases any resources held by VersionedDBProvider
	Close()
}

//versionedDB列出了数据库应该实现的方法
type VersionedDB interface {
//GetState获取给定命名空间和键的值。对于chaincode，命名空间对应于chaincodeid
	GetState(namespace string, key string) (*VersionedValue, error)
//GetVersion获取给定命名空间和键的版本。对于chaincode，命名空间对应于chaincodeid
	GetVersion(namespace string, key string) (*version.Height, error)
//GetStateMultipleKeys获取单个调用中多个键的值
	GetStateMultipleKeys(namespace string, keys []string) ([]*VersionedValue, error)
//GetStateRangeScanIterator返回一个迭代器，该迭代器包含给定键范围之间的所有键值。
//startkey包含在内
//endkey是独占的
//The returned ResultsIterator contains results of type *VersionedKV
	GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ResultsIterator, error)
//GetStateRangeScanIteratorWithMetadata返回一个迭代器，该迭代器包含给定键范围之间的所有键值。
//startkey包含在内
//endkey是独占的
//元数据是附加查询参数的映射
//返回的resultsiterator包含类型为*versionedkv的结果
	GetStateRangeScanIteratorWithMetadata(namespace string, startKey string, endKey string, metadata map[string]interface{}) (QueryResultsIterator, error)
//ExecuteQuery executes the given query and returns an iterator that contains results of type *VersionedKV.
	ExecuteQuery(namespace, query string) (ResultsIterator, error)
//ExecuteEqueryWithMetadata使用关联的查询选项和
//返回一个迭代器，该迭代器包含类型为*versionedkv的结果。
//元数据是附加查询参数的映射
	ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (QueryResultsIterator, error)
//ApplyUpdates将批应用于基础数据库。
//height是批处理中最高事务的高度，
//状态数据库实现应作为保存点使用。
	ApplyUpdates(batch *UpdateBatch, height *version.Height) error
//GetLatestSavePoint returns the height of the highest transaction upto which
//状态db一致
	GetLatestSavePoint() (*version.Height, error)
//validateKeyValue测试DB实现是否支持键和值。
//例如，leveldb支持密钥的任何字节，而couchdb只支持有效的utf-8字符串
//要使函数validateKeyValue返回特定错误，请说errInvalidKeyValue
//然而，到目前为止，这个函数的两个实现（leveldb和couchdb）在重新计算错误时都是确定的。
//也就是说，只有在键值对基础数据库无效时才会返回错误。
	ValidateKeyValue(key string, value []byte) error
//如果实现（基础数据库）支持用作键的任何字节，则BytesKeySupported返回true。
//例如，leveldb支持密钥的任何字节，而couchdb只支持有效的utf-8字符串
	BytesKeySupported() bool
//打开打开数据库
	Open() error
//关闭关闭数据库
	Close()
}

//BulkOptimizable接口为
//能够进行批处理操作的数据库
type BulkOptimizable interface {
	LoadCommittedVersions(keys []*CompositeKey) error
	GetCachedVersion(namespace, key string) (*version.Height, bool)
	ClearCachedVersions()
}

//可索引接口为
//databases capable of index operations
type IndexCapable interface {
	GetDBType() string
	ProcessIndexesForChaincodeDeploy(namespace string, fileEntries []*ccprovider.TarFileEntry) error
}

//compositekey包含命名空间和关键组件
type CompositeKey struct {
	Namespace string
	Key       string
}

//versionedValue包含值和相应的版本
type VersionedValue struct {
	Value    []byte
	Metadata []byte
	Version  *version.Height
}

//如果此更新指示删除某个键，则isDelete返回true
func (vv *VersionedValue) IsDelete() bool {
	return vv.Value == nil
}

//versionedkv包含密钥和相应的versionedValue
type VersionedKV struct {
	CompositeKey
	VersionedValue
}

//resultsiterator迭代查询结果
type ResultsIterator interface {
	Next() (QueryResult, error)
	Close()
}

//queryresultsiterator添加getbookmarkandclose方法
type QueryResultsIterator interface {
	ResultsIterator
	GetBookmarkAndClose() string
}

//queryresult-支持不同类型查询结果的通用接口。不同查询的实际类型不同
type QueryResult interface{}

type nsUpdates struct {
	m map[string]*VersionedValue
}

func newNsUpdates() *nsUpdates {
	return &nsUpdates{make(map[string]*VersionedValue)}
}

//updateBatch包含多个“updates”的详细信息
type UpdateBatch struct {
	updates map[string]*nsUpdates
}

//NewUpdateBatch构造批处理的实例
func NewUpdateBatch() *UpdateBatch {
	return &UpdateBatch{make(map[string]*nsUpdates)}
}

//get返回给定命名空间和键的versionedValue
func (batch *UpdateBatch) Get(ns string, key string) *VersionedValue {
	nsUpdates, ok := batch.updates[ns]
	if !ok {
		return nil
	}
	vv, ok := nsUpdates.m[key]
	if !ok {
		return nil
	}
	return vv
}

//Put只添加一个值为的键。元数据假定为零。
func (batch *UpdateBatch) Put(ns string, key string, value []byte, version *version.Height) {
	batch.PutValAndMetadata(ns, key, value, nil, version)
}

//PutValandMetadata添加了一个带有值和元数据的键
//要引入新函数以限制重构。稍后在单独的CR中，应删除上面的“Put”函数
func (batch *UpdateBatch) PutValAndMetadata(ns string, key string, value []byte, metadata []byte, version *version.Height) {
	if value == nil {
		panic("Nil value not allowed. Instead call 'Delete' function")
	}
	batch.Update(ns, key, &VersionedValue{value, metadata, version})
}

//删除删除键和关联值
func (batch *UpdateBatch) Delete(ns string, key string, version *version.Height) {
	batch.Update(ns, key, &VersionedValue{nil, nil, version})
}

//exists检查批处理中是否存在给定的密钥
func (batch *UpdateBatch) Exists(ns string, key string) bool {
	nsUpdates, ok := batch.updates[ns]
	if !ok {
		return false
	}
	_, ok = nsUpdates.m[key]
	return ok
}

//GetUpdatedNamespaces返回更新的命名空间的名称
func (batch *UpdateBatch) GetUpdatedNamespaces() []string {
	namespaces := make([]string, len(batch.updates))
	i := 0
	for ns := range batch.updates {
		namespaces[i] = ns
		i++
	}
	return namespaces
}

//更新使用命名空间和键的最新条目更新批处理
func (batch *UpdateBatch) Update(ns string, key string, vv *VersionedValue) {
	batch.getOrCreateNsUpdates(ns).m[key] = vv
}

//GetUpdates返回命名空间的所有更新
func (batch *UpdateBatch) GetUpdates(ns string) map[string]*VersionedValue {
	nsUpdates, ok := batch.updates[ns]
	if !ok {
		return nil
	}
	return nsUpdates.m
}

//GetRangeScanIterator返回按排序顺序在特定命名空间的键上迭代的迭代器
//换言之，“updateBatch”中内容的功能与
//`versioneddb.getStateRangeScanIterator（）`方法给出statedb中的内容
//此函数可用于在将UpdateBatch中的内容提交给StateDB之前查询这些内容。
//For instance, a validator implementation can used this to verify the validity of a range query of a transaction
//其中，updateBatch表示由同一块中前面的有效事务执行的修改的联合
//（假设使用group commit方法，在这里我们一起提交由块引起的所有更新）。
func (batch *UpdateBatch) GetRangeScanIterator(ns string, startKey string, endKey string) QueryResultsIterator {
	return newNsIterator(ns, startKey, endKey, batch)
}

func (batch *UpdateBatch) getOrCreateNsUpdates(ns string) *nsUpdates {
	nsUpdates := batch.updates[ns]
	if nsUpdates == nil {
		nsUpdates = newNsUpdates()
		batch.updates[ns] = nsUpdates
	}
	return nsUpdates
}

type nsIterator struct {
	ns         string
	nsUpdates  *nsUpdates
	sortedKeys []string
	nextIndex  int
	lastIndex  int
}

func newNsIterator(ns string, startKey string, endKey string, batch *UpdateBatch) *nsIterator {
	nsUpdates, ok := batch.updates[ns]
	if !ok {
		return &nsIterator{}
	}
	sortedKeys := util.GetSortedKeys(nsUpdates.m)
	var nextIndex int
	var lastIndex int
	if startKey == "" {
		nextIndex = 0
	} else {
		nextIndex = sort.SearchStrings(sortedKeys, startKey)
	}
	if endKey == "" {
		lastIndex = len(sortedKeys)
	} else {
		lastIndex = sort.SearchStrings(sortedKeys, endKey)
	}
	return &nsIterator{ns, nsUpdates, sortedKeys, nextIndex, lastIndex}
}

//下一个给出下一个键和版本化值。精疲力尽就归零
func (itr *nsIterator) Next() (QueryResult, error) {
	if itr.nextIndex >= itr.lastIndex {
		return nil, nil
	}
	key := itr.sortedKeys[itr.nextIndex]
	vv := itr.nsUpdates.m[key]
	itr.nextIndex++
	return &VersionedKV{CompositeKey{itr.ns, key}, VersionedValue{vv.Value, vv.Metadata, vv.Version}}, nil
}

//close从queryresult接口实现方法
func (itr *nsIterator) Close() {
//什么也不做
}

//getbookmarkandclose从queryresult接口实现方法
func (itr *nsIterator) GetBookmarkAndClose() string {
//什么也不做
	return ""
}

const optionLimit = "limit"

//validateRangeMetadata验证包含范围查询属性的JSON
func ValidateRangeMetadata(metadata map[string]interface{}) error {
	for key, keyVal := range metadata {
		switch key {

		case optionLimit:
//验证pageSize是否为整数
			if _, ok := keyVal.(int32); ok {
				continue
			}
			return fmt.Errorf("Invalid entry, \"limit\" must be a int32")

		default:
			return fmt.Errorf("Invalid entry, option %s not recognized", key)
		}
	}
	return nil
}
