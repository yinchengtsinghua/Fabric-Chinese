
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


package chaincode

import (
	"sync"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type TransactionContext struct {
	ChainID              string
	SignedProp           *pb.SignedProposal
	Proposal             *pb.Proposal
	ResponseNotifier     chan *pb.ChaincodeMessage
	TXSimulator          ledger.TxSimulator
	HistoryQueryExecutor ledger.HistoryQueryExecutor
	CollectionStore      privdata.CollectionStore
	IsInitTransaction    bool

//跟踪用于范围查询的打开迭代器
	queryMutex          sync.Mutex
	queryIteratorMap    map[string]commonledger.ResultsIterator
	pendingQueryResults map[string]*PendingQueryResult
	totalReturnCount    map[string]*int32

//用于保存集合ACL结果的缓存
//当为每个链代码创建TransactionContext时
//调用（即使在调用链码的情况下，
//我们不需要将命名空间存储在映射中，
//仅收集就足够了。
	AllowedCollectionAccess map[string]bool
}

func (t *TransactionContext) InitializeQueryContext(queryID string, iter commonledger.ResultsIterator) {
	t.queryMutex.Lock()
	if t.queryIteratorMap == nil {
		t.queryIteratorMap = map[string]commonledger.ResultsIterator{}
	}
	if t.pendingQueryResults == nil {
		t.pendingQueryResults = map[string]*PendingQueryResult{}
	}
	if t.totalReturnCount == nil {
		t.totalReturnCount = map[string]*int32{}
	}
	t.queryIteratorMap[queryID] = iter
	t.pendingQueryResults[queryID] = &PendingQueryResult{}
	zeroValue := int32(0)
	t.totalReturnCount[queryID] = &zeroValue
	t.queryMutex.Unlock()
}

func (t *TransactionContext) GetQueryIterator(queryID string) commonledger.ResultsIterator {
	t.queryMutex.Lock()
	iter := t.queryIteratorMap[queryID]
	t.queryMutex.Unlock()
	return iter
}

func (t *TransactionContext) GetPendingQueryResult(queryID string) *PendingQueryResult {
	t.queryMutex.Lock()
	result := t.pendingQueryResults[queryID]
	t.queryMutex.Unlock()
	return result
}

func (t *TransactionContext) GetTotalReturnCount(queryID string) *int32 {
	t.queryMutex.Lock()
	result := t.totalReturnCount[queryID]
	t.queryMutex.Unlock()
	return result
}

func (t *TransactionContext) CleanupQueryContext(queryID string) {
	t.queryMutex.Lock()
	defer t.queryMutex.Unlock()
	iter := t.queryIteratorMap[queryID]
	if iter != nil {
		iter.Close()
	}
	delete(t.queryIteratorMap, queryID)
	delete(t.pendingQueryResults, queryID)
	delete(t.totalReturnCount, queryID)
}

func (t *TransactionContext) CleanupQueryContextWithBookmark(queryID string) string {
	t.queryMutex.Lock()
	defer t.queryMutex.Unlock()
	iter := t.queryIteratorMap[queryID]
	bookmark := ""
	if iter != nil {
		if queryResultIterator, ok := iter.(commonledger.QueryResultsIterator); ok {
			bookmark = queryResultIterator.GetBookmarkAndClose()
		}
	}
	delete(t.queryIteratorMap, queryID)
	delete(t.pendingQueryResults, queryID)
	delete(t.totalReturnCount, queryID)
	return bookmark
}

func (t *TransactionContext) CloseQueryIterators() {
	t.queryMutex.Lock()
	defer t.queryMutex.Unlock()
	for _, iter := range t.queryIteratorMap {
		iter.Close()
	}
}
