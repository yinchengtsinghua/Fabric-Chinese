
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package fsblkstorage

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestBlockFileScanSmallTxOnly(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	ledgerid := "testLedger"
	blkfileMgrWrapper := newTestBlockfileWrapper(env, ledgerid)
	bg, gb := testutil.NewBlockGenerator(t, ledgerid, false)
	blocks := []*common.Block{gb}
	blocks = append(blocks, bg.NextTestBlock(0, 0))
	blocks = append(blocks, bg.NextTestBlock(0, 0))
	blocks = append(blocks, bg.NextTestBlock(0, 0))
	blkfileMgrWrapper.addBlocks(blocks)
	blkfileMgrWrapper.close()

	filePath := deriveBlockfilePath(env.provider.conf.getLedgerBlockDir(ledgerid), 0)
	_, fileSize, err := util.FileExists(filePath)
	assert.NoError(t, err)

	lastBlockBytes, endOffsetLastBlock, numBlocks, err := scanForLastCompleteBlock(env.provider.conf.getLedgerBlockDir(ledgerid), 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), numBlocks)
	assert.Equal(t, fileSize, endOffsetLastBlock)

	expectedLastBlockBytes, _, err := serializeBlock(blocks[len(blocks)-1])
	assert.Equal(t, expectedLastBlockBytes, lastBlockBytes)
}

func TestBlockFileScanSmallTxLastTxIncomplete(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	ledgerid := "testLedger"
	blkfileMgrWrapper := newTestBlockfileWrapper(env, ledgerid)
	bg, gb := testutil.NewBlockGenerator(t, ledgerid, false)
	blocks := []*common.Block{gb}
	blocks = append(blocks, bg.NextTestBlock(0, 0))
	blocks = append(blocks, bg.NextTestBlock(0, 0))
	blocks = append(blocks, bg.NextTestBlock(0, 0))
	blkfileMgrWrapper.addBlocks(blocks)
	blkfileMgrWrapper.close()

	filePath := deriveBlockfilePath(env.provider.conf.getLedgerBlockDir(ledgerid), 0)
	_, fileSize, err := util.FileExists(filePath)
	assert.NoError(t, err)

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	defer file.Close()
	assert.NoError(t, err)
	err = file.Truncate(fileSize - 1)
	assert.NoError(t, err)

	lastBlockBytes, _, numBlocks, err := scanForLastCompleteBlock(env.provider.conf.getLedgerBlockDir(ledgerid), 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, len(blocks)-1, numBlocks)

	expectedLastBlockBytes, _, err := serializeBlock(blocks[len(blocks)-2])
	assert.Equal(t, expectedLastBlockBytes, lastBlockBytes)
}
