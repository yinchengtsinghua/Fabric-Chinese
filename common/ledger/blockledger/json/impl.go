
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


package jsonledger

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/golang/protobuf/jsonpb"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("common.ledger.blockledger.json")

var closedChan chan struct{}
var fileLock sync.Mutex

func init() {
	closedChan = make(chan struct{})
	close(closedChan)
}

const (
	blockFileFormatString      = "block_%020d.json"
	chainDirectoryFormatString = "chain_%s"
)

type cursor struct {
	jl          *jsonLedger
	blockNumber uint64
}

type jsonLedger struct {
	directory string
	height    uint64
	lastHash  []byte
	marshaler *jsonpb.Marshaler

	mutex  sync.Mutex
	signal chan struct{}
}

//readblock返回块或nil，无论是否找到块，（nil，true）通常表示一个不可恢复的问题。
func (jl *jsonLedger) readBlock(number uint64) (*cb.Block, bool) {
	name := jl.blockFilename(number)

//如果正在写入，读取块文件可能会导致“意外的EOF”错误。
//
	fileLock.Lock()
	defer fileLock.Unlock()

	file, err := os.Open(name)
	if err == nil {
		defer file.Close()
		block := &cb.Block{}
		err = jsonpb.Unmarshal(file, block)
		if err != nil {
			return nil, true
		}
		logger.Debugf("Read block %d", block.Header.Number)
		return block, true
	}
	return nil, false
}

//下一个块，直到有新的块可用，或者返回错误，如果
//下一个块不再可检索
func (cu *cursor) Next() (*cb.Block, cb.Status) {
//这只循环一次，作为信号读取
//
	for {
		block, found := cu.jl.readBlock(cu.blockNumber)
		if found {
			if block == nil {
				return nil, cb.Status_SERVICE_UNAVAILABLE
			}
			cu.blockNumber++
			return block, cb.Status_SUCCESS
		}

//将信号通道复制到锁定状态以避免比赛
//附加新的信号通道
		cu.jl.mutex.Lock()
		signal := cu.jl.signal
		cu.jl.mutex.Unlock()
		<-signal
	}
}

func (cu *cursor) Close() {}

//迭代器返回由ab.seekinfo消息指定的迭代器及其
//起始块编号
func (jl *jsonLedger) Iterator(startPosition *ab.SeekPosition) (blockledger.Iterator, uint64) {
	switch start := startPosition.Type.(type) {
	case *ab.SeekPosition_Oldest:
		return &cursor{jl: jl, blockNumber: 0}, 0
	case *ab.SeekPosition_Newest:
		high := jl.height - 1
		return &cursor{jl: jl, blockNumber: high}, high
	case *ab.SeekPosition_Specified:
		if start.Specified.Number > jl.height {
			return &blockledger.NotFoundErrorIterator{}, 0
		}
		return &cursor{jl: jl, blockNumber: start.Specified.Number}, start.Specified.Number
	default:
		return &blockledger.NotFoundErrorIterator{}, 0
	}
}

//height返回分类帐上的块数
func (jl *jsonLedger) Height() uint64 {
	return jl.height
}

//
func (jl *jsonLedger) Append(block *cb.Block) error {
	if block.Header.Number != jl.height {
		return errors.Errorf("block number should have been %d but was %d", jl.height, block.Header.Number)
	}

	if !bytes.Equal(block.Header.PreviousHash, jl.lastHash) {
		return errors.Errorf("block should have had previous hash of %x but was %x", jl.lastHash, block.Header.PreviousHash)
	}

	jl.writeBlock(block)
	jl.lastHash = block.Header.Hash()
	jl.height++

//
	jl.mutex.Lock()
	close(jl.signal)
	jl.signal = make(chan struct{})
	jl.mutex.Unlock()
	return nil
}

//
func (jl *jsonLedger) writeBlock(block *cb.Block) {
	name := jl.blockFilename(block.Header.Number)

	fileLock.Lock()
	defer fileLock.Unlock()

	file, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	err = jl.marshaler.Marshal(file, block)
	logger.Debugf("Wrote block %d", block.Header.Number)
	if err != nil {
		logger.Panicf("Error marshalling with block number [%d]: %s", block.Header.Number, err)
	}
}

//
//一个给定的数字应该存储在磁盘上
func (jl *jsonLedger) blockFilename(number uint64) string {
	return filepath.Join(jl.directory, fmt.Sprintf(blockFileFormatString, number))
}
