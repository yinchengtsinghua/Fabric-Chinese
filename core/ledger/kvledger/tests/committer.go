
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


package tests

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

//Committer帮助切割块并将块（带有pvt数据）提交到分类帐。
type committer struct {
	lgr    ledger.PeerLedger
	blkgen *blkGenerator
	assert *assert.Assertions
}

func newCommitter(lgr ledger.PeerLedger, t *testing.T) *committer {
	return &committer{lgr, newBlockGenerator(lgr, t), assert.New(t)}
}

//CutBlockAndCommitWithPvtData从给定的“TxandPvtData”中剪切下一个块，并将该块（带有Pvt数据）提交到分类帐。
//此函数返回已提交到分类帐以提交的“ledger.blockAndPvtData”的副本。
//返回副本而不是实际的副本，因为分类帐在提交前对提交的块进行了一些更改
//（例如设置元数据）和测试代码将希望具有提交到的块的准确副本
//分类帐
func (c *committer) cutBlockAndCommitWithPvtdata(trans ...*txAndPvtdata) *ledger.BlockAndPvtData {
	blk := c.blkgen.nextBlockAndPvtdata(trans...)
	blkCopy := c.copyOfBlockAndPvtdata(blk)
	c.assert.NoError(
		c.lgr.CommitWithPvtData(blk),
	)
	return blkCopy
}

func (c *committer) cutBlockAndCommitExpectError(trans ...*txAndPvtdata) (*ledger.BlockAndPvtData, error) {
	blk := c.blkgen.nextBlockAndPvtdata(trans...)
	blkCopy := c.copyOfBlockAndPvtdata(blk)
	err := c.lgr.CommitWithPvtData(blk)
	c.assert.Error(err)
	return blkCopy, err
}

func (c *committer) copyOfBlockAndPvtdata(blk *ledger.BlockAndPvtData) *ledger.BlockAndPvtData {
	blkBytes, err := proto.Marshal(blk.Block)
	c.assert.NoError(err)
	blkCopy := &common.Block{}
	c.assert.NoError(proto.Unmarshal(blkBytes, blkCopy))
	return &ledger.BlockAndPvtData{Block: blkCopy, PvtData: blk.PvtData,
		MissingPvtData: blk.MissingPvtData}
}

/////////////////block生成代码
//BLKGenerator帮助创建分类帐的下一个块
type blkGenerator struct {
	lastNum  uint64
	lastHash []byte
	assert   *assert.Assertions
}

//NewBlockGenerator构造“BlkGenerator”并初始化“BlkGenerator”
//从分类帐中可用的最后一个块开始，以便可以填充下一个块
//具有正确的块号和上一个块哈希
func newBlockGenerator(lgr ledger.PeerLedger, t *testing.T) *blkGenerator {
	assert := assert.New(t)
	info, err := lgr.GetBlockchainInfo()
	assert.NoError(err)
	return &blkGenerator{info.Height - 1, info.CurrentBlockHash, assert}
}

//NextBlock和PvtData将剪切下一个块
func (g *blkGenerator) nextBlockAndPvtdata(trans ...*txAndPvtdata) *ledger.BlockAndPvtData {
	block := common.NewBlock(g.lastNum+1, g.lastHash)
	blockPvtdata := make(map[uint64]*ledger.TxPvtData)
	for i, tran := range trans {
		seq := uint64(i)
		envelopeBytes, _ := proto.Marshal(tran.Envelope)
		block.Data.Data = append(block.Data.Data, envelopeBytes)
		if tran.Pvtws != nil {
			blockPvtdata[seq] = &ledger.TxPvtData{SeqInBlock: seq, WriteSet: tran.Pvtws}
		}
	}
	block.Header.DataHash = block.Data.Hash()
	g.lastNum++
	g.lastHash = block.Header.Hash()
	setBlockFlagsToValid(block)
	return &ledger.BlockAndPvtData{Block: block, PvtData: blockPvtdata,
		MissingPvtData: make(ledger.TxMissingPvtDataMap)}
}
