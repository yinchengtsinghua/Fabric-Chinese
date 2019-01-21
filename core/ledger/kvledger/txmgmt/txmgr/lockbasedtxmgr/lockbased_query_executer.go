
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
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger"
)

//lockbasedqueryexecutor是“lockbasedtxmgr”中使用的查询执行器
type lockBasedQueryExecutor struct {
	helper *queryHelper
	txid   string
}

func newQueryExecutor(txmgr *LockBasedTxMgr, txid string) *lockBasedQueryExecutor {
	helper := newQueryHelper(txmgr, nil)
	logger.Debugf("constructing new query executor txid = [%s]", txid)
	return &lockBasedQueryExecutor{helper, txid}
}

//getstate在接口“ledger.queryexecutor”中实现方法
func (q *lockBasedQueryExecutor) GetState(ns string, key string) (val []byte, err error) {
	val, _, err = q.helper.getState(ns, key)
	return
}

//getStateMetadata在接口“ledger.queryExecutor”中实现方法
func (q *lockBasedQueryExecutor) GetStateMetadata(namespace, key string) (map[string][]byte, error) {
	return q.helper.getStateMetadata(namespace, key)
}

//getstatemultiplekeys在接口“ledger.queryexecutor”中实现方法
func (q *lockBasedQueryExecutor) GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error) {
	return q.helper.getStateMultipleKeys(namespace, keys)
}

//getStateRangeScanIterator在接口“ledger.queryExecutor”中实现方法
//结果中包含startkey，但不包括endkey。空的startkey引用第一个可用的key
//空的endkey指的是最后一个可用的key。用于扫描所有键，包括startkey和endkey
//can be supplied as empty strings. However, a full scan shuold be used judiciously for performance reasons.
func (q *lockBasedQueryExecutor) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (commonledger.ResultsIterator, error) {
	return q.helper.getStateRangeScanIterator(namespace, startKey, endKey)
}

//getStateRangeScanIteratorWithMetadata在接口“ledger.queryExecutor”中实现方法
//结果中包含startkey，但不包括endkey。空的startkey引用第一个可用的key
//空的endkey指的是最后一个可用的key。用于扫描所有键，包括startkey和endkey
//可以作为空字符串提供。但是，出于性能方面的考虑，应谨慎使用全扫描Shuold。
//元数据是附加查询参数的映射
func (q *lockBasedQueryExecutor) GetStateRangeScanIteratorWithMetadata(namespace string, startKey string, endKey string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	return q.helper.getStateRangeScanIteratorWithMetadata(namespace, startKey, endKey, metadata)
}

//executeQuery在接口“ledger.queryexecutor”中实现方法
func (q *lockBasedQueryExecutor) ExecuteQuery(namespace, query string) (commonledger.ResultsIterator, error) {
	return q.helper.executeQuery(namespace, query)
}

//Excel中的元数据实现了接口'Leiger-QueRealExtuor中的方法
func (q *lockBasedQueryExecutor) ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	return q.helper.executeQueryWithMetadata(namespace, query, metadata)
}

//getprivatedata在接口'ledger.queryexecutor'中实现方法
func (q *lockBasedQueryExecutor) GetPrivateData(namespace, collection, key string) ([]byte, error) {
	return q.helper.getPrivateData(namespace, collection, key)
}

//getprivatedatametadata在接口'ledger.queryexecutor'中实现方法
func (q *lockBasedQueryExecutor) GetPrivateDataMetadata(namespace, collection, key string) (map[string][]byte, error) {
	return q.helper.getPrivateDataMetadata(namespace, collection, key)
}

//getprivatedatametadatabyhash在接口“ledger.queryexecutor”中实现方法
func (q *lockBasedQueryExecutor) GetPrivateDataMetadataByHash(namespace, collection string, keyhash []byte) (map[string][]byte, error) {
	return q.helper.getPrivateDataMetadataByHash(namespace, collection, keyhash)
}

//GETPultReabaseMultIdKIKE在接口'Leiger-QueRealExtuor中实现方法
func (q *lockBasedQueryExecutor) GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([][]byte, error) {
	return q.helper.getPrivateDataMultipleKeys(namespace, collection, keys)
}

//GETPultRealAtRangeSCAN迭代在接口'Leiger-QueRealExtuor中实现方法
func (q *lockBasedQueryExecutor) GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (commonledger.ResultsIterator, error) {
	return q.helper.getPrivateDataRangeScanIterator(namespace, collection, startKey, endKey)
}

//ExecuteEqueryOnPrivateData在接口“ledger.queryExecutor”中实现方法
func (q *lockBasedQueryExecutor) ExecuteQueryOnPrivateData(namespace, collection, query string) (commonledger.ResultsIterator, error) {
	return q.helper.executeQueryOnPrivateData(namespace, collection, query)
}

//done在接口“ledger.queryexecutor”中实现方法
func (q *lockBasedQueryExecutor) Done() {
	logger.Debugf("Done with transaction simulation / query execution [%s]", q.txid)
	q.helper.done()
}
