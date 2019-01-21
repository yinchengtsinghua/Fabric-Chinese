
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
*/


package ramledger

import (
	"bytes"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("common.ledger.blockledger.ram")

type cursor struct {
	list *simpleList
}

type simpleList struct {
	lock   sync.RWMutex
	next   *simpleList
	signal chan struct{}
	block  *cb.Block
}

func (s *simpleList) getNext() *simpleList {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.next
}

func (s *simpleList) setNext(n *simpleList) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.next = n
}

type ramLedger struct {
	lock    sync.RWMutex
	maxSize int
	size    int
	oldest  *simpleList
	newest  *simpleList
}

//下一个块，直到有新的块可用，或者返回错误，如果
//下一个块不再可检索
func (cu *cursor) Next() (*cb.Block, cb.Status) {
//
	for {
		next := cu.list.getNext()
		if next != nil {
			cu.list = next
			return cu.list.block, cb.Status_SUCCESS
		}
		<-cu.list.signal
	}
}

//
func (cu *cursor) Close() {}

//迭代器返回由ab.seekinfo消息指定的迭代器及其
//起始块编号
func (rl *ramLedger) Iterator(startPosition *ab.SeekPosition) (blockledger.Iterator, uint64) {
	rl.lock.RLock()
	defer rl.lock.RUnlock()

	var list *simpleList
	switch start := startPosition.Type.(type) {
	case *ab.SeekPosition_Oldest:
		oldest := rl.oldest
		list = &simpleList{
			block:  &cb.Block{Header: &cb.BlockHeader{Number: oldest.block.Header.Number - 1}},
			next:   oldest,
			signal: make(chan struct{}),
		}
		close(list.signal)
	case *ab.SeekPosition_Newest:
		newest := rl.newest
		list = &simpleList{
			block:  &cb.Block{Header: &cb.BlockHeader{Number: newest.block.Header.Number - 1}},
			next:   newest,
			signal: make(chan struct{}),
		}
		close(list.signal)
	case *ab.SeekPosition_Specified:
		oldest := rl.oldest
		specified := start.Specified.Number
		logger.Debugf("Attempting to return block %d", specified)

//
		if specified+1 < oldest.block.Header.Number+1 || specified > rl.newest.block.Header.Number+1 {
			logger.Debugf("Returning error iterator because specified seek was %d with oldest %d and newest %d",
				specified, rl.oldest.block.Header.Number, rl.newest.block.Header.Number)
			return &blockledger.NotFoundErrorIterator{}, 0
		}

		if specified == oldest.block.Header.Number {
			list = &simpleList{
				block:  &cb.Block{Header: &cb.BlockHeader{Number: oldest.block.Header.Number - 1}},
				next:   oldest,
				signal: make(chan struct{}),
			}
			close(list.signal)
			break
		}

		list = oldest
		for {
			if list.block.Header.Number == specified-1 {
				break
			}
list = list.getNext() //
		}
	}
	cursor := &cursor{list: list}
	blockNum := list.block.Header.Number + 1

//
	if blockNum == ^uint64(0) {
		cursor.Next()
		blockNum++
	}

	return cursor, blockNum
}

//height返回分类帐上的块数
func (rl *ramLedger) Height() uint64 {
	rl.lock.RLock()
	defer rl.lock.RUnlock()
	return rl.newest.block.Header.Number + 1
}

//
func (rl *ramLedger) Append(block *cb.Block) error {
	rl.lock.Lock()
	defer rl.lock.Unlock()

	if block.Header.Number != rl.newest.block.Header.Number+1 {
		return errors.Errorf("block number should have been %d but was %d",
			rl.newest.block.Header.Number+1, block.Header.Number)
	}

if rl.newest.block.Header.Number+1 != 0 { //
		if !bytes.Equal(block.Header.PreviousHash, rl.newest.block.Header.Hash()) {
			return errors.Errorf("block should have had previous hash of %x but was %x",
				rl.newest.block.Header.Hash(), block.Header.PreviousHash)
		}
	}

	rl.appendBlock(block)
	return nil
}

func (rl *ramLedger) appendBlock(block *cb.Block) {
	next := &simpleList{
		signal: make(chan struct{}),
		block:  block,
	}
	rl.newest.setNext(next)

	lastSignal := rl.newest.signal
	logger.Debugf("Sending signal that block %d has a successor", rl.newest.block.Header.Number)
	rl.newest = rl.newest.getNext()
	close(lastSignal)

	rl.size++

	if rl.size > rl.maxSize {
		logger.Debugf("RAM ledger max size about to be exceeded, removing oldest item: %d",
			rl.oldest.block.Header.Number)
		rl.oldest = rl.oldest.getNext()
		rl.size--
	}
}
