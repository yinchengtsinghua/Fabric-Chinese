
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


package fsblkstorage

import (
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/ledger/util"
	"github.com/stretchr/testify/assert"
)

func TestConstructCheckpointInfoFromBlockFiles(t *testing.T) {
	testPath := "/tmp/tests/fabric/common/ledger/blkstorage/fsblkstorage"
	ledgerid := "testLedger"
	conf := NewConf(testPath, 0)
	blkStoreDir := conf.getLedgerBlockDir(ledgerid)
	env := newTestEnv(t, conf)
	util.CreateDirIfMissing(blkStoreDir)
	defer env.Cleanup()

//在空块文件夹上构建的检查点应返回cpinfo，ischainempty:true
	cpInfo, err := constructCheckpointInfoFromBlockFiles(blkStoreDir)
	assert.NoError(t, err)
	assert.Equal(t, &checkpointInfo{isChainEmpty: true, lastBlockNumber: 0, latestFileChunksize: 0, latestFileChunkSuffixNum: 0}, cpInfo)

	w := newTestBlockfileWrapper(env, ledgerid)
	defer w.close()
	blockfileMgr := w.blockfileMgr
	bg, gb := testutil.NewBlockGenerator(t, ledgerid, false)

//添加几个块，并验证从文件系统派生的cpinfo应该与从块文件管理器派生的cpinfo相同。
	blockfileMgr.addBlock(gb)
	for _, blk := range bg.NextTestBlocks(3) {
		blockfileMgr.addBlock(blk)
	}
	checkCPInfoFromFile(t, blkStoreDir, blockfileMgr.cpInfo)

//将链移动到新文件并检查从文件系统派生的cpinfo
	blockfileMgr.moveToNextFile()
	checkCPInfoFromFile(t, blkStoreDir, blockfileMgr.cpInfo)

//添加几个将转到新文件的块，并验证从文件系统派生的cpinfo应与从块文件管理器派生的cpinfo相同。
	for _, blk := range bg.NextTestBlocks(3) {
		blockfileMgr.addBlock(blk)
	}
	checkCPInfoFromFile(t, blkStoreDir, blockfileMgr.cpInfo)

//编写一个部分块（模拟崩溃），并验证从文件系统派生的cpinfo应该与从块文件管理器派生的cpinfo相同。
	lastTestBlk := bg.NextTestBlocks(1)[0]
	blockBytes, _, err := serializeBlock(lastTestBlk)
	assert.NoError(t, err)
	partialByte := append(proto.EncodeVarint(uint64(len(blockBytes))), blockBytes[len(blockBytes)/2:]...)
	blockfileMgr.currentFileWriter.append(partialByte, true)
	checkCPInfoFromFile(t, blkStoreDir, blockfileMgr.cpInfo)

//关闭块存储，删除索引并重新启动并验证
	cpInfoBeforeClose := blockfileMgr.cpInfo
	w.close()
	env.provider.Close()
	indexFolder := conf.getIndexDir()
	assert.NoError(t, os.RemoveAll(indexFolder))

	env = newTestEnv(t, conf)
	w = newTestBlockfileWrapper(env, ledgerid)
	blockfileMgr = w.blockfileMgr
	assert.Equal(t, cpInfoBeforeClose, blockfileMgr.cpInfo)

	lastBlkIndexed, err := blockfileMgr.index.getLastBlockIndexed()
	assert.NoError(t, err)
	assert.Equal(t, uint64(6), lastBlkIndexed)

//启动后再次添加最后一个块，再次检查cpinfo
	assert.NoError(t, blockfileMgr.addBlock(lastTestBlk))
	checkCPInfoFromFile(t, blkStoreDir, blockfileMgr.cpInfo)
}

func checkCPInfoFromFile(t *testing.T, blkStoreDir string, expectedCPInfo *checkpointInfo) {
	cpInfo, err := constructCheckpointInfoFromBlockFiles(blkStoreDir)
	assert.NoError(t, err)
	assert.Equal(t, expectedCPInfo, cpInfo)
}
