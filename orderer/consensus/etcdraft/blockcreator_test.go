
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
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func getSeedBlock() *cb.Block {
	seedBlock := cb.NewBlock(0, []byte("firsthash"))
	seedBlock.Data.Data = [][]byte{[]byte("somebytes")}
	return seedBlock
}

func TestValidCreatedBlocksQueue(t *testing.T) {
	seedBlock := getSeedBlock()
	logger := flogging.NewFabricLogger(zap.NewNop())
	bc := newBlockCreator(seedBlock, logger)

	t.Run("correct creation of a single block", func(t *testing.T) {
		block := bc.createNextBlock([]*cb.Envelope{{Payload: []byte("some other bytes")}})

		assert.Equal(t, seedBlock.Header.Number+1, block.Header.Number)
		assert.Equal(t, block.Data.Hash(), block.Header.DataHash)
		assert.Equal(t, seedBlock.Header.Hash(), block.Header.PreviousHash)
//此创建的块应是创建的块队列的一部分
		assert.Len(t, bc.CreatedBlocks, 1)

		bc.commitBlock(block)

		assert.Empty(t, bc.CreatedBlocks)
		assert.Equal(t, bc.LastCommittedBlock.Header.Hash(), block.Header.Hash())
	})

	t.Run("ResetCreatedBlocks resets the queue of created blocks", func(t *testing.T) {
		numBlocks := 10
		blocks := []*cb.Block{}
		for i := 0; i < numBlocks; i++ {
			blocks = append(blocks, bc.createNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}

		bc.resetCreatedBlocks()
		assert.True(t,
			bytes.Equal(bc.LastCommittedBlock.Header.Bytes(), bc.LastCreatedBlock.Header.Bytes()),
			"resetting the created blocks queue should leave the lastCommittedBlock and the lastCreatedBlock equal",
		)
		assert.Empty(t, bc.CreatedBlocks)
	})

	t.Run("commit of block without any created blocks sets the lastCreatedBlock correctly", func(t *testing.T) {
		block := bc.createNextBlock([]*cb.Envelope{{Payload: []byte("some other bytes")}})
		bc.resetCreatedBlocks()

		bc.commitBlock(block)

		assert.True(t,
			bytes.Equal(block.Header.Bytes(), bc.LastCommittedBlock.Header.Bytes()),
			"resetting the created blocks queue should leave the lastCommittedBlock and the lastCreatedBlock equal",
		)
		assert.True(t,
			bytes.Equal(block.Header.Bytes(), bc.LastCreatedBlock.Header.Bytes()),
			"resetting the created blocks queue should leave the lastCommittedBlock and the lastCreatedBlock equal",
		)
		assert.Empty(t, bc.CreatedBlocks)
	})
	t.Run("propose multiple blocks without having to commit them; tests the optimistic block creation", func(t *testing.T) {
  /*
   *情景：
   *我们最初创建五个块，然后只提交其中的两个。我们再创造五个街区
   *并提交建议堆栈中剩余的8个块。这应该是成功的，因为
   *块与创建的块没有差异。
   **/

		blocks := []*cb.Block{}
//创建五个块而不将其写出
		for i := 0; i < 5; i++ {
			blocks = append(blocks, bc.createNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}
		assert.Len(t, bc.CreatedBlocks, 5)

//把这两个写出来
		for i := 0; i < 2; i++ {
			bc.commitBlock(blocks[i])
		}
		assert.Len(t, bc.CreatedBlocks, 3)

//再创建五个块；应在之前创建的五个块上创建这些块
		for i := 0; i < 5; i++ {
			blocks = append(blocks, bc.createNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}
		assert.Len(t, bc.CreatedBlocks, 8)

//写出剩余的8个块；只有在所有块都在一个堆栈中创建时才能成功，否则会死机
		for i := 2; i < 10; i++ {
			bc.commitBlock(blocks[i])
		}
		assert.Empty(t, bc.CreatedBlocks)

//断言块确实是以正确的顺序创建的
		for i := 0; i < 9; i++ {
			assertNextBlock(t, blocks[i], blocks[i+1])
		}
	})

	t.Run("createNextBlock returns nil after createdBlocksBuffersize blocks have been created", func(t *testing.T) {
		numBlocks := createdBlocksBuffersize
		blocks := []*cb.Block{}

		for i := 0; i < numBlocks; i++ {
			blocks = append(blocks, bc.createNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}

		block := bc.createNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}})

		assert.Nil(t, block)
	})

	t.Run("created blocks should always be over committed blocks", func(t *testing.T) {
  /*
   *情景：
   *我们将创造
   * 1。在baselastcreatedblock上具有5个块的建议堆栈，以及
   * 2。baselastcreatedblock上的备用块。
   *我们将写出这个备用块，并验证是否在这个备用块上创建了后续块。
   *不在现有的建议堆栈上。
   *如果写入的块与
   *创建的块。
   **/


		baseLastCreatedBlock := bc.LastCreatedBlock

//创建五个块的堆栈而不将它们写出
		blocks := []*cb.Block{}
		for i := 0; i < 5; i++ {
			blocks = append(blocks, bc.createNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}

//创建并写出备用块
		alternateBlock := createBlockOverSpecifiedBlock(baseLastCreatedBlock, []*cb.Envelope{{Payload: []byte("alternate test envelope")}})
		bc.commitBlock(alternateBlock)

//断言CreateNextBlock在此AlternateBlock上创建下一个块
		createdBlockOverAlternateBlock := bc.createNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}})
		synthesizedBlockOverAlternateBlock := createBlockOverSpecifiedBlock(alternateBlock, []*cb.Envelope{{Payload: []byte("test envelope")}})
		assert.True(t,
			bytes.Equal(createdBlockOverAlternateBlock.Header.Bytes(), synthesizedBlockOverAlternateBlock.Header.Bytes()),
			"created and synthesized blocks should be equal",
		)
		bc.commitBlock(createdBlockOverAlternateBlock)
	})

}

func createBlockOverSpecifiedBlock(baseBlock *cb.Block, messages []*cb.Envelope) *cb.Block {
	previousBlockHash := baseBlock.Header.Hash()

	data := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	var err error
	for i, msg := range messages {
		data.Data[i], err = proto.Marshal(msg)
		if err != nil {
			panic(fmt.Sprintf("Could not marshal envelope: %s", err))
		}
	}

	block := cb.NewBlock(baseBlock.Header.Number+1, previousBlockHash)
	block.Header.DataHash = data.Hash()
	block.Data = data

	return block
}

//如果“next block”正确格式为下一个块，则isNextBlock返回true
//在链中跟在“block”后面。
func assertNextBlock(t *testing.T, block, nextBlock *cb.Block) {
	assert.Equal(t, block.Header.Number+1, nextBlock.Header.Number)
	assert.Equal(t, block.Header.Hash(), nextBlock.Header.PreviousHash)
}
