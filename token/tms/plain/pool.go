
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


package plain

import (
	"fmt"
	"io"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/token/tms"
	"github.com/pkg/errors"
)

//在池中找不到条目时，将返回OutputNotFoundError。
type OutputNotFoundError struct {
	ID string
}

func (o *OutputNotFoundError) Error() string {
	return fmt.Sprintf("entry not found: %s", o.ID)
}

//在池中找不到事务时返回txnotfounderror。
type TxNotFoundError struct {
	TxID string
}

func (p *TxNotFoundError) Error() string {
	return fmt.Sprintf("transaction not found: %s", p.TxID)
}

//memoryPool是一个内存中的事务和未使用的输出分类账。
//此实现仅用于测试。
type MemoryPool struct {
	mutex   sync.RWMutex
	entries map[string]*token.PlainOutput
	history map[string]*token.TokenTransaction
}

//new memorypool创建新的memorypool
func NewMemoryPool() *MemoryPool {
	return &MemoryPool{
		entries: map[string]*token.PlainOutput{},
		history: map[string]*token.TokenTransaction{},
	}
}

//检查是否可以提交建议的更新。
func (p *MemoryPool) checkUpdate(transactionData []tms.TransactionData) error {
	for _, td := range transactionData {
		action := td.Tx.GetPlainAction()
		if action == nil {
			return errors.Errorf("check update failed for transaction '%s': missing token action", td.TxID)
		}

		err := p.checkAction(action, td.TxID)
		if err != nil {
			return errors.WithMessage(err, "check update failed")
		}

		if p.history[td.TxID] != nil {
			return errors.Errorf("transaction already exists: %s", td.TxID)
		}
	}

	return nil
}

//CommitUpdate将事务数据提交到池中。
func (p *MemoryPool) CommitUpdate(transactionData []tms.TransactionData) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.checkUpdate(transactionData)
	if err != nil {
		return err
	}

	for _, td := range transactionData {
		p.commitAction(td.Tx.GetPlainAction(), td.TxID)
		p.history[td.TxID] = cloneTransaction(td.Tx)
	}

	return nil
}

func (p *MemoryPool) checkAction(plainAction *token.PlainTokenAction, txID string) error {
	switch action := plainAction.Data.(type) {
	case *token.PlainTokenAction_PlainImport:
		return p.checkImportAction(action.PlainImport, txID)
	default:
		return errors.Errorf("unknown plain token action: %T", action)
	}
}

func (p *MemoryPool) commitAction(plainAction *token.PlainTokenAction, txID string) {
	switch action := plainAction.Data.(type) {
	case *token.PlainTokenAction_PlainImport:
		p.commitImportAction(action.PlainImport, txID)
	}
}

func (p *MemoryPool) checkImportAction(importAction *token.PlainImport, txID string) error {
	for i := range importAction.GetOutputs() {
		entryID := calculateOutputID(txID, i)
		if p.entries[entryID] != nil {
			return errors.Errorf("pool entry already exists: %s", entryID)
		}
	}
	return nil
}

func (p *MemoryPool) commitImportAction(importAction *token.PlainImport, txID string) {
	for i, entry := range importAction.GetOutputs() {
		entryID := calculateOutputID(txID, i)
		p.addEntry(entryID, entry)
	}
}

//向池中添加新条目。
func (p *MemoryPool) addEntry(entryID string, entry *token.PlainOutput) {
	p.entries[entryID] = cloneOutput(entry)
}

func cloneOutput(po *token.PlainOutput) *token.PlainOutput {
	clone := proto.Clone(po)
	return clone.(*token.PlainOutput)
}

func cloneTransaction(tt *token.TokenTransaction) *token.TokenTransaction {
	clone := proto.Clone(tt)
	return clone.(*token.TokenTransaction)
}

//OutputByID根据其ID获取输出。
func (p *MemoryPool) OutputByID(id string) (*token.PlainOutput, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	output := p.entries[id]
	if output == nil {
		return nil, &OutputNotFoundError{ID: id}
	}

	return output, nil
}

//txbyid通过其事务ID获取事务。
//如果不存在具有给定ID的事务，则返回txNotFoundError。
func (p *MemoryPool) TxByID(txID string) (*token.TokenTransaction, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tt := p.history[txID]
	if tt == nil {
		return nil, &TxNotFoundError{TxID: txID}
	}

	return tt, nil
}

//迭代器基于池的副本返回未占用输出的迭代器。
func (p *MemoryPool) Iterator() *PoolIterator {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	keys := make([]string, len(p.entries))
	i := 0
	for k := range p.entries {
		keys[i] = k
		i++
	}

	return &PoolIterator{pool: p, keys: keys, current: 0}
}

//poolLiterator是一个迭代器，用于迭代池中的项。
type PoolIterator struct {
	pool    *MemoryPool
	keys    []string
	current int
}

//next从池中获取下一个输出，如果没有更多输出，则获取io.eof。
func (it *PoolIterator) Next() (string, *token.PlainOutput, error) {
	if it.current >= len(it.keys) {
		return "", nil, io.EOF
	}

	entryID := it.keys[it.current]
	it.current++
	entry, err := it.pool.OutputByID(entryID)

	return entryID, entry, err
}

//HistoryIterator创建一个新的HistoryIterator，用于迭代事务历史记录。
func (p *MemoryPool) HistoryIterator() *HistoryIterator {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	keys := make([]string, len(p.history))
	i := 0
	for k := range p.history {
		keys[i] = k
		i++
	}

	return &HistoryIterator{pool: p, keys: keys, current: 0}
}

//HistoryIterator是用于迭代事务历史记录的迭代器。
type HistoryIterator struct {
	pool    *MemoryPool
	keys    []string
	current int
}

//next从历史记录中获取下一个事务，如果没有更多事务，则为io.eof。
func (it *HistoryIterator) Next() (string, *token.TokenTransaction, error) {
	if it.current >= len(it.keys) {
		return "", nil, io.EOF
	}

	txID := it.keys[it.current]
	it.current++
	tx, err := it.pool.TxByID(txID)

	return txID, tx, err
}

//计算事务中单个输出的ID，作为
//事务ID和输出的索引
func calculateOutputID(tmsTxID string, index int) string {
	return fmt.Sprintf("%s.%d", tmsTxID, index)
}
