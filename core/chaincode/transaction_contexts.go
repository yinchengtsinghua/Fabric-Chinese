
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
	"context"
	"sync"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

type key string

const (
//txSimulatorKey是用于提供分类帐的上下文键。txSimulator
//从背书人到链码。
	TXSimulatorKey key = "txsimulatorkey"

//HistoryQueryExecutorKey是用于提供
//从背书人到链码的ledger.historyqueryexecutor。
	HistoryQueryExecutorKey key = "historyqueryexecutorkey"
)

//TransactionContexts维护处理程序的活动事务上下文。
type TransactionContexts struct {
	mutex    sync.Mutex
	contexts map[string]*TransactionContext
}

//NewTransactionContexts为活动事务上下文创建注册表。
func NewTransactionContexts() *TransactionContexts {
	return &TransactionContexts{
		contexts: map[string]*TransactionContext{},
	}
}

//ContextID创建了一个范围为链的事务标识符。
func contextID(chainID, txID string) string {
	return chainID + txID
}

//创建为指定链创建新的TransactionContext，并
//事务ID。当事务上下文已经
//已为指定的链和事务ID创建。
func (c *TransactionContexts) Create(txParams *ccprovider.TransactionParams) (*TransactionContext, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ctxID := contextID(txParams.ChannelID, txParams.TxID)
	if c.contexts[ctxID] != nil {
		return nil, errors.Errorf("txid: %s(%s) exists", txParams.TxID, txParams.ChannelID)
	}

	txctx := &TransactionContext{
		ChainID:              txParams.ChannelID,
		SignedProp:           txParams.SignedProp,
		Proposal:             txParams.Proposal,
		ResponseNotifier:     make(chan *pb.ChaincodeMessage, 1),
		TXSimulator:          txParams.TXSimulator,
		HistoryQueryExecutor: txParams.HistoryQueryExecutor,
		CollectionStore:      txParams.CollectionStore,
		IsInitTransaction:    txParams.IsInitTransaction,

		queryIteratorMap:    map[string]commonledger.ResultsIterator{},
		pendingQueryResults: map[string]*PendingQueryResult{},

		AllowedCollectionAccess: make(map[string]bool),
	}
	c.contexts[ctxID] = txctx

	return txctx, nil
}

func getTxSimulator(ctx context.Context) ledger.TxSimulator {
	if txsim, ok := ctx.Value(TXSimulatorKey).(ledger.TxSimulator); ok {
		return txsim
	}
	return nil
}

func getHistoryQueryExecutor(ctx context.Context) ledger.HistoryQueryExecutor {
	if historyQueryExecutor, ok := ctx.Value(HistoryQueryExecutorKey).(ledger.HistoryQueryExecutor); ok {
		return historyQueryExecutor
	}
	return nil
}

//get检索与链关联的事务上下文，并
//事务ID。
func (c *TransactionContexts) Get(chainID, txID string) *TransactionContext {
	ctxID := contextID(chainID, txID)
	c.mutex.Lock()
	tc := c.contexts[ctxID]
	c.mutex.Unlock()
	return tc
}

//删除删除与指定链关联的事务上下文
//和事务ID。
func (c *TransactionContexts) Delete(chainID, txID string) {
	ctxID := contextID(chainID, txID)
	c.mutex.Lock()
	delete(c.contexts, ctxID)
	c.mutex.Unlock()
}

//关闭关闭与上下文关联的所有查询迭代器。
func (c *TransactionContexts) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, txctx := range c.contexts {
		txctx.CloseQueryIterators()
	}
}
