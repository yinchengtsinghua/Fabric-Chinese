
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


package lockbasedtxmgr

import (
	"fmt"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	ledger "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/storageutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/pkg/errors"
)

type queryHelper struct {
	txmgr             *LockBasedTxMgr
	collNameValidator *collNameValidator
	rwsetBuilder      *rwsetutil.RWSetBuilder
	itrs              []*resultsItr
	err               error
	doneInvoked       bool
}

func newQueryHelper(txmgr *LockBasedTxMgr, rwsetBuilder *rwsetutil.RWSetBuilder) *queryHelper {
	helper := &queryHelper{txmgr: txmgr, rwsetBuilder: rwsetBuilder}
	validator := newCollNameValidator(txmgr.ccInfoProvider, &lockBasedQueryExecutor{helper: helper})
	helper.collNameValidator = validator
	return helper
}

func (h *queryHelper) getState(ns string, key string) ([]byte, []byte, error) {
	if err := h.checkDone(); err != nil {
		return nil, nil, err
	}
	versionedValue, err := h.txmgr.db.GetState(ns, key)
	if err != nil {
		return nil, nil, err
	}
	val, metadata, ver := decomposeVersionedValue(versionedValue)
	if h.rwsetBuilder != nil {
		h.rwsetBuilder.AddToReadSet(ns, key, ver)
	}
	return val, metadata, nil
}

func (h *queryHelper) getStateMultipleKeys(namespace string, keys []string) ([][]byte, error) {
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	versionedValues, err := h.txmgr.db.GetStateMultipleKeys(namespace, keys)
	if err != nil {
		return nil, nil
	}
	values := make([][]byte, len(versionedValues))
	for i, versionedValue := range versionedValues {
		val, _, ver := decomposeVersionedValue(versionedValue)
		if h.rwsetBuilder != nil {
			h.rwsetBuilder.AddToReadSet(namespace, keys[i], ver)
		}
		values[i] = val
	}
	return values, nil
}

func (h *queryHelper) getStateRangeScanIterator(namespace string, startKey string, endKey string) (ledger.QueryResultsIterator, error) {
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	itr, err := newResultsItr(namespace, startKey, endKey, nil, h.txmgr.db, h.rwsetBuilder,
		ledgerconfig.IsQueryReadsHashingEnabled(), ledgerconfig.GetMaxDegreeQueryReadsHashing())
	if err != nil {
		return nil, err
	}
	h.itrs = append(h.itrs, itr)
	return itr, nil
}

func (h *queryHelper) getStateRangeScanIteratorWithMetadata(namespace string, startKey string, endKey string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	itr, err := newResultsItr(namespace, startKey, endKey, metadata, h.txmgr.db, h.rwsetBuilder,
		ledgerconfig.IsQueryReadsHashingEnabled(), ledgerconfig.GetMaxDegreeQueryReadsHashing())
	if err != nil {
		return nil, err
	}
	h.itrs = append(h.itrs, itr)
	return itr, nil
}

func (h *queryHelper) executeQuery(namespace, query string) (commonledger.ResultsIterator, error) {
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	dbItr, err := h.txmgr.db.ExecuteQuery(namespace, query)
	if err != nil {
		return nil, err
	}
	return &queryResultsItr{DBItr: dbItr, RWSetBuilder: h.rwsetBuilder}, nil
}

func (h *queryHelper) executeQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	dbItr, err := h.txmgr.db.ExecuteQueryWithMetadata(namespace, query, metadata)
	if err != nil {
		return nil, err
	}
	return &queryResultsItr{DBItr: dbItr, RWSetBuilder: h.rwsetBuilder}, nil
}

func (h *queryHelper) getPrivateData(ns, coll, key string) ([]byte, error) {
	if err := h.validateCollName(ns, coll); err != nil {
		return nil, err
	}
	if err := h.checkDone(); err != nil {
		return nil, err
	}

	var err error
	var hashVersion *version.Height
	var versionedValue *statedb.VersionedValue

	if versionedValue, err = h.txmgr.db.GetPrivateData(ns, coll, key); err != nil {
		return nil, err
	}

//元数据对于私有数据始终是零，因为元数据是散列密钥的一部分（而不是原始密钥）。
	val, _, ver := decomposeVersionedValue(versionedValue)

	keyHash := util.ComputeStringHash(key)
	if hashVersion, err = h.txmgr.db.GetKeyHashVersion(ns, coll, keyHash); err != nil {
		return nil, err
	}
	if !version.AreSame(hashVersion, ver) {
		return nil, &txmgr.ErrPvtdataNotAvailable{Msg: fmt.Sprintf(
			"private data matching public hash version is not available. Public hash version = %#v, Private data version = %#v",
			hashVersion, ver)}
	}
	if h.rwsetBuilder != nil {
		h.rwsetBuilder.AddToHashedReadSet(ns, coll, key, ver)
	}
	return val, nil
}

func (h *queryHelper) getPrivateDataValueHash(ns, coll, key string) (valueHash, metadataBytes []byte, err error) {
	if err := h.validateCollName(ns, coll); err != nil {
		return nil, nil, err
	}
	if err := h.checkDone(); err != nil {
		return nil, nil, err
	}
	var versionedValue *statedb.VersionedValue

	keyHash := util.ComputeStringHash(key)
	if versionedValue, err = h.txmgr.db.GetValueHash(ns, coll, keyHash); err != nil {
		return nil, nil, err
	}
	valHash, metadata, ver := decomposeVersionedValue(versionedValue)
	if h.rwsetBuilder != nil {
		h.rwsetBuilder.AddToHashedReadSet(ns, coll, key, ver)
	}
	return valHash, metadata, nil
}

func (h *queryHelper) getPrivateDataMultipleKeys(ns, coll string, keys []string) ([][]byte, error) {
	if err := h.validateCollName(ns, coll); err != nil {
		return nil, err
	}
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	versionedValues, err := h.txmgr.db.GetPrivateDataMultipleKeys(ns, coll, keys)
	if err != nil {
		return nil, nil
	}
	values := make([][]byte, len(versionedValues))
	for i, versionedValue := range versionedValues {
		val, _, ver := decomposeVersionedValue(versionedValue)
		if h.rwsetBuilder != nil {
			h.rwsetBuilder.AddToHashedReadSet(ns, coll, keys[i], ver)
		}
		values[i] = val
	}
	return values, nil
}

func (h *queryHelper) getPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (commonledger.ResultsIterator, error) {
	if err := h.validateCollName(namespace, collection); err != nil {
		return nil, err
	}
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	dbItr, err := h.txmgr.db.GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey)
	if err != nil {
		return nil, err
	}
	return &pvtdataResultsItr{namespace, collection, dbItr}, nil
}

func (h *queryHelper) executeQueryOnPrivateData(namespace, collection, query string) (commonledger.ResultsIterator, error) {
	if err := h.validateCollName(namespace, collection); err != nil {
		return nil, err
	}
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	dbItr, err := h.txmgr.db.ExecuteQueryOnPrivateData(namespace, collection, query)
	if err != nil {
		return nil, err
	}
	return &pvtdataResultsItr{namespace, collection, dbItr}, nil
}

func (h *queryHelper) getStateMetadata(ns string, key string) (map[string][]byte, error) {
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	var metadataBytes []byte
	var err error
	if h.rwsetBuilder == nil {
//未记录读取版本，通过优化路径检索元数据值
		if metadataBytes, err = h.txmgr.db.GetStateMetadata(ns, key); err != nil {
			return nil, err
		}
	} else {
		if _, metadataBytes, err = h.getState(ns, key); err != nil {
			return nil, err
		}
	}
	return storageutil.DeserializeMetadata(metadataBytes)
}

func (h *queryHelper) getPrivateDataMetadata(ns, coll, key string) (map[string][]byte, error) {
	if h.rwsetBuilder == nil {
//未记录读取版本，通过优化路径检索元数据值
		return h.getPrivateDataMetadataByHash(ns, coll, util.ComputeStringHash(key))
	}
	if err := h.validateCollName(ns, coll); err != nil {
		return nil, err
	}
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	_, metadataBytes, err := h.getPrivateDataValueHash(ns, coll, key)
	if err != nil {
		return nil, err
	}
	return storageutil.DeserializeMetadata(metadataBytes)
}

func (h *queryHelper) getPrivateDataMetadataByHash(ns, coll string, keyhash []byte) (map[string][]byte, error) {
	if err := h.validateCollName(ns, coll); err != nil {
		return nil, err
	}
	if err := h.checkDone(); err != nil {
		return nil, err
	}
	if h.rwsetBuilder != nil {
//这需要改进rwset生成器以接受keyhash
		return nil, errors.New("retrieving private data metadata by keyhash is not supported in simulation. This function is only available for query as yet")
	}
	metadataBytes, err := h.txmgr.db.GetPrivateDataMetadataByHash(ns, coll, keyhash)
	if err != nil {
		return nil, err
	}
	return storageutil.DeserializeMetadata(metadataBytes)
}

func (h *queryHelper) done() {
	if h.doneInvoked {
		return
	}

	defer func() {
		h.txmgr.commitRWLock.RUnlock()
		h.doneInvoked = true
		for _, itr := range h.itrs {
			itr.Close()
		}
	}()
}

func (h *queryHelper) addRangeQueryInfo() {
	for _, itr := range h.itrs {
		if h.rwsetBuilder != nil {
			results, hash, err := itr.rangeQueryResultsHelper.Done()
			if err != nil {
				h.err = err
				return
			}
			if results != nil {
				itr.rangeQueryInfo.SetRawReads(results)
			}
			if hash != nil {
				itr.rangeQueryInfo.SetMerkelSummary(hash)
			}
			h.rwsetBuilder.AddToRangeQuerySet(itr.ns, itr.rangeQueryInfo)
		}
	}
}

func (h *queryHelper) checkDone() error {
	if h.doneInvoked {
		return errors.New("this instance should not be used after calling Done()")
	}
	return nil
}

func (h *queryHelper) validateCollName(ns, coll string) error {
	return h.collNameValidator.validateCollName(ns, coll)
}

//ResultSitr实现接口Ledger.ResultSiterator
//这将包装实际的db迭代器并拦截调用
//在使用的readwriteset中生成rangequeryinfo
//用于在提交期间执行幻象读取验证
type resultsItr struct {
	ns                      string
	endKey                  string
	dbItr                   statedb.ResultsIterator
	rwSetBuilder            *rwsetutil.RWSetBuilder
	rangeQueryInfo          *kvrwset.RangeQueryInfo
	rangeQueryResultsHelper *rwsetutil.RangeQueryResultsHelper
}

func newResultsItr(ns string, startKey string, endKey string, metadata map[string]interface{},
	db statedb.VersionedDB, rwsetBuilder *rwsetutil.RWSetBuilder, enableHashing bool, maxDegree uint32) (*resultsItr, error) {
	var err error
	var dbItr statedb.ResultsIterator
	if metadata == nil {
		dbItr, err = db.GetStateRangeScanIterator(ns, startKey, endKey)
	} else {
		dbItr, err = db.GetStateRangeScanIteratorWithMetadata(ns, startKey, endKey, metadata)
	}
	if err != nil {
		return nil, err
	}
	itr := &resultsItr{ns: ns, dbItr: dbItr}
//这是一个模拟请求，因此，启用范围查询信息的捕获
	if rwsetBuilder != nil {
		itr.rwSetBuilder = rwsetBuilder
		itr.endKey = endKey
//只需设置启动键…在后面的NEXT（）方法中设置EntKEY。
		itr.rangeQueryInfo = &kvrwset.RangeQueryInfo{StartKey: startKey}
		resultsHelper, err := rwsetutil.NewRangeQueryResultsHelper(enableHashing, maxDegree)
		if err != nil {
			return nil, err
		}
		itr.rangeQueryResultsHelper = resultsHelper
	}
	return itr, nil
}

//next在interface ledger.resultsiterator中实现方法
//在返回下一个结果之前，请更新rangequeryinfo中的endkey和itretired
//如果我们将构造函数中的endkey设置为
//在原始查询中提供，如果
//调用者决定在某个中间点停止迭代。或者，我们可以
//在close（）函数中设置endkey和itrexed，但可能不希望更改
//基于是否调用了close（）的事务行为
func (itr *resultsItr) Next() (commonledger.QueryResult, error) {
	queryResult, err := itr.dbItr.Next()
	if err != nil {
		return nil, err
	}
	itr.updateRangeQueryInfo(queryResult)
	if queryResult == nil {
		return nil, nil
	}
	versionedKV := queryResult.(*statedb.VersionedKV)
	return &queryresult.KV{Namespace: versionedKV.Namespace, Key: versionedKV.Key, Value: versionedKV.Value}, nil
}

//getbookmarkandclose实现接口ledger.resultsiterator中的方法
func (itr *resultsItr) GetBookmarkAndClose() string {
	returnBookmark := ""
	if queryResultIterator, ok := itr.dbItr.(statedb.QueryResultsIterator); ok {
		returnBookmark = queryResultIterator.GetBookmarkAndClose()
	}
	return returnBookmark
}

//更新rangequeryinfo更新rangequeryinfo的两个属性
//1）endkey-设置为要返回给调用方的a）最新密钥（如果迭代器没有用完）
//因为，我们不知道调用者是否会再次调用next（）。
//或b）原始查询中提供的最后一个键（如果迭代器已用完）
//2）ItRexHuff-设置为TRUE，如果迭代器将返回NIL作为下一次（）调用的结果
func (itr *resultsItr) updateRangeQueryInfo(queryResult statedb.QueryResult) {
	if itr.rwSetBuilder == nil {
		return
	}

	if queryResult == nil {
//调用方已扫描，直到迭代器耗尽。
//So, set the endKey to the actual endKey supplied in the query
		itr.rangeQueryInfo.ItrExhausted = true
		itr.rangeQueryInfo.EndKey = itr.endKey
		return
	}
	versionedKV := queryResult.(*statedb.VersionedKV)
	itr.rangeQueryResultsHelper.AddResult(rwsetutil.NewKVRead(versionedKV.Key, versionedKV.Version))
//将结束键设置为调用方检索到的最新键。
//因为，调用方实际上可能不会再次调用next（）函数
	itr.rangeQueryInfo.EndKey = versionedKV.Key
}

//接口目录中的关闭实现方法
func (itr *resultsItr) Close() {
	itr.dbItr.Close()
}

type queryResultsItr struct {
	DBItr        statedb.ResultsIterator
	RWSetBuilder *rwsetutil.RWSetBuilder
}

//next在interface ledger.resultsiterator中实现方法
func (itr *queryResultsItr) Next() (commonledger.QueryResult, error) {

	queryResult, err := itr.DBItr.Next()
	if err != nil {
		return nil, err
	}
	if queryResult == nil {
		return nil, nil
	}
	versionedQueryRecord := queryResult.(*statedb.VersionedKV)
	logger.Debugf("queryResultsItr.Next() returned a record:%s", string(versionedQueryRecord.Value))

	if itr.RWSetBuilder != nil {
		itr.RWSetBuilder.AddToReadSet(versionedQueryRecord.Namespace, versionedQueryRecord.Key, versionedQueryRecord.Version)
	}
	return &queryresult.KV{Namespace: versionedQueryRecord.Namespace, Key: versionedQueryRecord.Key, Value: versionedQueryRecord.Value}, nil
}

//接口目录中的关闭实现方法
func (itr *queryResultsItr) Close() {
	itr.DBItr.Close()
}

func (itr *queryResultsItr) GetBookmarkAndClose() string {
	returnBookmark := ""
	if queryResultIterator, ok := itr.DBItr.(statedb.QueryResultsIterator); ok {
		returnBookmark = queryResultIterator.GetBookmarkAndClose()
	}
	return returnBookmark
}

func decomposeVersionedValue(versionedValue *statedb.VersionedValue) ([]byte, []byte, *version.Height) {
	var value []byte
	var metadata []byte
	var ver *version.Height
	if versionedValue != nil {
		value = versionedValue.Value
		ver = versionedValue.Version
		metadata = versionedValue.Metadata
	}
	return value, metadata, ver
}

//PVTDATARESULTSIRTR对PVT数据查询结果进行迭代
type pvtdataResultsItr struct {
	ns    string
	coll  string
	dbItr statedb.ResultsIterator
}

//next在interface ledger.resultsiterator中实现方法
func (itr *pvtdataResultsItr) Next() (commonledger.QueryResult, error) {
	queryResult, err := itr.dbItr.Next()
	if err != nil {
		return nil, err
	}
	if queryResult == nil {
		return nil, nil
	}
	versionedQueryRecord := queryResult.(*statedb.VersionedKV)
	return &queryresult.KV{
		Namespace: itr.ns,
		Key:       versionedQueryRecord.Key,
		Value:     versionedQueryRecord.Value,
	}, nil
}

//接口目录中的关闭实现方法
func (itr *pvtdataResultsItr) Close() {
	itr.dbItr.Close()
}
