
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


package stateleveldb

import (
	"bytes"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

var logger = flogging.MustGetLogger("stateleveldb")

var compositeKeySep = []byte{0x00}
var lastKeyIndicator = byte(0x01)
var savePointKey = []byte{0x00}

//versioneddbprovider实现接口versioneddbprovider
type VersionedDBProvider struct {
	dbProvider *leveldbhelper.Provider
}

//NewVersionedDBProvider实例化VersionedDBProvider
func NewVersionedDBProvider() *VersionedDBProvider {
	dbPath := ledgerconfig.GetStateLevelDBPath()
	logger.Debugf("constructing VersionedDBProvider dbPath=%s", dbPath)
	dbProvider := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})
	return &VersionedDBProvider{dbProvider}
}

//getdbhandle获取命名数据库的句柄
func (provider *VersionedDBProvider) GetDBHandle(dbName string) (statedb.VersionedDB, error) {
	return newVersionedDB(provider.dbProvider.GetDBHandle(dbName), dbName), nil
}

//关闭关闭基础数据库
func (provider *VersionedDBProvider) Close() {
	provider.dbProvider.Close()
}

//版本数据库实现版本数据库接口
type versionedDB struct {
	db     *leveldbhelper.DBHandle
	dbName string
}

//newVersionedDB构造versionedDB的实例
func newVersionedDB(db *leveldbhelper.DBHandle, dbName string) *versionedDB {
	return &versionedDB{db, dbName}
}

//在版本数据库接口中打开实现方法
func (vdb *versionedDB) Open() error {
//不执行任何操作，因为使用了共享数据库
	return nil
}

//Close在VersionedDB接口中实现方法
func (vdb *versionedDB) Close() {
//不执行任何操作，因为使用了共享数据库
}

//validateKeyValue在versionedDB接口中实现方法
func (vdb *versionedDB) ValidateKeyValue(key string, value []byte) error {
	return nil
}

//BytesKeySupported在VersionedDB接口中实现方法
func (vdb *versionedDB) BytesKeySupported() bool {
	return true
}

//GetState在VersionedDB接口中实现方法
func (vdb *versionedDB) GetState(namespace string, key string) (*statedb.VersionedValue, error) {
	logger.Debugf("GetState(). ns=%s, key=%s", namespace, key)
	compositeKey := constructCompositeKey(namespace, key)
	dbVal, err := vdb.db.Get(compositeKey)
	if err != nil {
		return nil, err
	}
	if dbVal == nil {
		return nil, nil
	}
	return decodeValue(dbVal)
}

//GetVersion在VersionedDB接口中实现方法
func (vdb *versionedDB) GetVersion(namespace string, key string) (*version.Height, error) {
	versionedValue, err := vdb.GetState(namespace, key)
	if err != nil {
		return nil, err
	}
	if versionedValue == nil {
		return nil, nil
	}
	return versionedValue.Version, nil
}

//GetStateMultipleKeys在VersionedDB接口中实现方法
func (vdb *versionedDB) GetStateMultipleKeys(namespace string, keys []string) ([]*statedb.VersionedValue, error) {
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
func (vdb *versionedDB) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (statedb.ResultsIterator, error) {
	return vdb.GetStateRangeScanIteratorWithMetadata(namespace, startKey, endKey, nil)
}

const optionLimit = "limit"

//GetStateRangeScanIteratorWithMetadata在VersionedDB接口中实现方法
func (vdb *versionedDB) GetStateRangeScanIteratorWithMetadata(namespace string, startKey string, endKey string, metadata map[string]interface{}) (statedb.QueryResultsIterator, error) {

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

//注意：元数据不用于范围查询的goleveldb实现
	compositeStartKey := constructCompositeKey(namespace, startKey)
	compositeEndKey := constructCompositeKey(namespace, endKey)
	if endKey == "" {
		compositeEndKey[len(compositeEndKey)-1] = lastKeyIndicator
	}
	dbItr := vdb.db.GetIterator(compositeStartKey, compositeEndKey)

	return newKVScanner(namespace, dbItr, requestedLimit), nil

}

//ExecuteQuery在VersionedDB接口中实现方法
func (vdb *versionedDB) ExecuteQuery(namespace, query string) (statedb.ResultsIterator, error) {
	return nil, errors.New("ExecuteQuery not supported for leveldb")
}

//ExecuteEqueryWithMetadata在VersionedDB接口中实现方法
func (vdb *versionedDB) ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (statedb.QueryResultsIterator, error) {
	return nil, errors.New("ExecuteQueryWithMetadata not supported for leveldb")
}

//ApplyUpdates在VersionedDB接口中实现方法
func (vdb *versionedDB) ApplyUpdates(batch *statedb.UpdateBatch, height *version.Height) error {
	dbBatch := leveldbhelper.NewUpdateBatch()
	namespaces := batch.GetUpdatedNamespaces()
	for _, ns := range namespaces {
		updates := batch.GetUpdates(ns)
		for k, vv := range updates {
			compositeKey := constructCompositeKey(ns, k)
			logger.Debugf("Channel [%s]: Applying key(string)=[%s] key(bytes)=[%#v]", vdb.dbName, string(compositeKey), compositeKey)

			if vv.Value == nil {
				dbBatch.Delete(compositeKey)
			} else {
				encodedVal, err := encodeValue(vv)
				if err != nil {
					return err
				}
				dbBatch.Put(compositeKey, encodedVal)
			}
		}
	}
//在给定高度记录保存点
//如果给定高度为零，则表示我们正在提交旧块的pvt数据。
//在这种情况下，我们不应该存储用于恢复的保存点。最新更新的DoldBlockList
//在pvtstore中用作pvt数据的保存点。
	if height != nil {
		dbBatch.Put(savePointKey, height.ToBytes())
	}
//将snyc设置为true作为预防措施，在进一步测试之后，false可能是一个不错的优化。
	if err := vdb.db.WriteBatch(dbBatch, true); err != nil {
		return err
	}
	return nil
}

//GetLatestSavePoint在VersionedDB接口中实现方法
func (vdb *versionedDB) GetLatestSavePoint() (*version.Height, error) {
	versionBytes, err := vdb.db.Get(savePointKey)
	if err != nil {
		return nil, err
	}
	if versionBytes == nil {
		return nil, nil
	}
	version, _ := version.NewHeightFromBytes(versionBytes)
	return version, nil
}

func constructCompositeKey(ns string, key string) []byte {
	return append(append([]byte(ns), compositeKeySep...), []byte(key)...)
}

func splitCompositeKey(compositeKey []byte) (string, string) {
	split := bytes.SplitN(compositeKey, compositeKeySep, 2)
	return string(split[0]), string(split[1])
}

type kvScanner struct {
	namespace            string
	dbItr                iterator.Iterator
	requestedLimit       int32
	totalRecordsReturned int32
}

func newKVScanner(namespace string, dbItr iterator.Iterator, requestedLimit int32) *kvScanner {
	return &kvScanner{namespace, dbItr, requestedLimit, 0}
}

func (scanner *kvScanner) Next() (statedb.QueryResult, error) {

	if scanner.requestedLimit > 0 && scanner.totalRecordsReturned >= scanner.requestedLimit {
		return nil, nil
	}

	if !scanner.dbItr.Next() {
		return nil, nil
	}

	dbKey := scanner.dbItr.Key()
	dbVal := scanner.dbItr.Value()
	dbValCopy := make([]byte, len(dbVal))
	copy(dbValCopy, dbVal)
	_, key := splitCompositeKey(dbKey)
	vv, err := decodeValue(dbValCopy)
	if err != nil {
		return nil, err
	}

	scanner.totalRecordsReturned++

	return &statedb.VersionedKV{
		CompositeKey: statedb.CompositeKey{Namespace: scanner.namespace, Key: key},
//TODO通过更改字段类型删除下面的取消引用
//'statedb.versionedkv'中的'versionedValue'指向指针
		VersionedValue: *vv}, nil
}

func (scanner *kvScanner) Close() {
	scanner.dbItr.Release()
}

func (scanner *kvScanner) GetBookmarkAndClose() string {
	retval := ""
	if scanner.dbItr.Next() {
		dbKey := scanner.dbItr.Key()
		_, key := splitCompositeKey(dbKey)
		retval = key
	}
	scanner.Close()
	return retval
}
