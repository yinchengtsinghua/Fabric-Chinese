
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package etcdraft

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	cb "github.com/hyperledger/fabric/protos/common"
)

//这控制飞行中创建的块的最大数量；即块
//是创造出来的，但不是写出来的。
//一旦此数量的块在运行中，createnextblock将返回nil。
const createdBlocksBuffersize = 20

//BlockCreator优化地在链中创建块。创造的
//块最终可能无法写出。这使我们能够销售
//创建块并就其达成共识，从而
//性能改进。如果
//已提交分流块
//为了安全地使用BlockCreator，应该只有一个线程与之交互。
type blockCreator struct {
	CreatedBlocks      chan *cb.Block
	LastCreatedBlock   *cb.Block
	LastCommittedBlock *cb.Block
	logger             *flogging.FabricLogger
}

func newBlockCreator(lastBlock *cb.Block, logger *flogging.FabricLogger) *blockCreator {
	if lastBlock == nil {
		logger.Panic("block creator initialized with nil last block")
	}
	bc := &blockCreator{
		CreatedBlocks:      make(chan *cb.Block, createdBlocksBuffersize),
		LastCreatedBlock:   lastBlock,
		LastCommittedBlock: lastBlock,
		logger:             logger,
	}

	logger.Debugf("Initialized block creator with (lastblockNumber=%d)", lastBlock.Header.Number)
	return bc
}

//createnextblock创建一个具有下一个块号的新块，并且
//给定内容。
//如果可以创建块，则返回创建的块，否则为零。
//当已经存在“createdblocksbuffersize”块时，它可能会失败
//“createdBlocks”通道中已创建且不能容纳更多内容。
func (bc *blockCreator) createNextBlock(messages []*cb.Envelope) *cb.Block {
	previousBlockHash := bc.LastCreatedBlock.Header.Hash()

	data := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	var err error
	for i, msg := range messages {
		data.Data[i], err = proto.Marshal(msg)
		if err != nil {
			bc.logger.Panicf("Could not marshal envelope: %s", err)
		}
	}

	block := cb.NewBlock(bc.LastCreatedBlock.Header.Number+1, previousBlockHash)
	block.Header.DataHash = data.Hash()
	block.Data = data

	select {
	case bc.CreatedBlocks <- block:
		bc.LastCreatedBlock = block
		bc.logger.Debugf("Created block %d", bc.LastCreatedBlock.Header.Number)
		return block
	default:
		return nil
	}
}

//ResetCreatedBlocks重置已创建块的队列。
//随后的块将在上次提交的块上创建
//使用CommitBlock。
func (bc *blockCreator) resetCreatedBlocks() {
//我们不应该重新创建createdBlocks通道，因为它会导致
//访问数据竞赛
	for len(bc.CreatedBlocks) > 0 {
//清空通道
		<-bc.CreatedBlocks
	}
	bc.LastCreatedBlock = bc.LastCommittedBlock
	bc.logger.Debug("Reset created blocks")
}

//应该为所有块调用commitBlock，以便让blockCreator知道
//已提交哪些块。如果承诺区块偏离
//创建的块的堆栈，然后重置堆栈。
func (bc *blockCreator) commitBlock(block *cb.Block) {
	bc.LastCommittedBlock = block

//检查提交的块是否与创建的块分离
	select {
	case b := <-bc.CreatedBlocks:
		if !bytes.Equal(b.Header.Bytes(), block.Header.Bytes()) {
//写入的块正从CreateBlocks堆栈中分离
//丢弃创建的块
			bc.resetCreatedBlocks()
		}
//否则，写入的块是createBlocks堆栈的一部分；不执行任何操作
	default:
//没有创建的块；请将此块设置为最后创建的块。
//当调用WriteBlock而不调用CreateNextBlock时，会发生这种情况。
//例如，在ETCD/RAFT的情况下，领导者提出块和追随者
//只写建议的块。
		bc.LastCreatedBlock = block
	}
}
