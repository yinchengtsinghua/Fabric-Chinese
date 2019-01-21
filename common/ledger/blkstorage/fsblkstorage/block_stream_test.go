
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
*/


package fsblkstorage

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestBlockfileStream(t *testing.T) {
	testBlockfileStream(t, 0)
	testBlockfileStream(t, 1)
	testBlockfileStream(t, 10)
}

func testBlockfileStream(t *testing.T, numBlocks int) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	ledgerid := "testledger"
	w := newTestBlockfileWrapper(env, ledgerid)
	blocks := testutil.ConstructTestBlocks(t, numBlocks)
	w.addBlocks(blocks)
	w.close()

	s, err := newBlockfileStream(w.blockfileMgr.rootDir, 0, 0)
	defer s.close()
	assert.NoError(t, err, "Error in constructing blockfile stream")

	blockCount := 0
	for {
		blockBytes, err := s.nextBlockBytes()
		assert.NoError(t, err, "Error in getting next block")
		if blockBytes == nil {
			break
		}
		blockCount++
	}
//
	blockBytes, err := s.nextBlockBytes()
	assert.Nil(t, blockBytes)
	assert.NoError(t, err, "Error in getting next block after exhausting the file")
	assert.Equal(t, numBlocks, blockCount)
}

func TestBlockFileStreamUnexpectedEOF(t *testing.T) {
	partialBlockBytes := []byte{}
	dummyBlockBytes := testutil.ConstructRandomBytes(t, 100)
	lenBytes := proto.EncodeVarint(uint64(len(dummyBlockBytes)))
	partialBlockBytes = append(partialBlockBytes, lenBytes...)
	partialBlockBytes = append(partialBlockBytes, dummyBlockBytes...)
	testBlockFileStreamUnexpectedEOF(t, 10, partialBlockBytes[:1])
	testBlockFileStreamUnexpectedEOF(t, 10, partialBlockBytes[:2])
	testBlockFileStreamUnexpectedEOF(t, 10, partialBlockBytes[:5])
	testBlockFileStreamUnexpectedEOF(t, 10, partialBlockBytes[:20])
}

func testBlockFileStreamUnexpectedEOF(t *testing.T, numBlocks int, partialBlockBytes []byte) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	w := newTestBlockfileWrapper(env, "testLedger")
	blockfileMgr := w.blockfileMgr
	blocks := testutil.ConstructTestBlocks(t, numBlocks)
	w.addBlocks(blocks)
	blockfileMgr.currentFileWriter.append(partialBlockBytes, true)
	w.close()
	s, err := newBlockfileStream(blockfileMgr.rootDir, 0, 0)
	defer s.close()
	assert.NoError(t, err, "Error in constructing blockfile stream")

	for i := 0; i < numBlocks; i++ {
		blockBytes, err := s.nextBlockBytes()
		assert.NotNil(t, blockBytes)
		assert.NoError(t, err, "Error in getting next block")
	}
	blockBytes, err := s.nextBlockBytes()
	assert.Nil(t, blockBytes)
	assert.Exactly(t, ErrUnexpectedEndOfBlockfile, err)
}

func TestBlockStream(t *testing.T) {
	testBlockStream(t, 1)
	testBlockStream(t, 2)
	testBlockStream(t, 10)
}

func testBlockStream(t *testing.T, numFiles int) {
	ledgerID := "testLedger"
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	w := newTestBlockfileWrapper(env, ledgerID)
	defer w.close()
	blockfileMgr := w.blockfileMgr

	numBlocksInEachFile := 10
	bg, gb := testutil.NewBlockGenerator(t, ledgerID, false)
	w.addBlocks([]*common.Block{gb})
	for i := 0; i < numFiles; i++ {
		numBlocks := numBlocksInEachFile
		if i == 0 {
//
			numBlocks = numBlocksInEachFile - 1
		}
		blocks := bg.NextTestBlocks(numBlocks)
		w.addBlocks(blocks)
		blockfileMgr.moveToNextFile()
	}
	s, err := newBlockStream(blockfileMgr.rootDir, 0, 0, numFiles-1)
	defer s.close()
	assert.NoError(t, err, "Error in constructing new block stream")
	blockCount := 0
	for {
		blockBytes, err := s.nextBlockBytes()
		assert.NoError(t, err, "Error in getting next block")
		if blockBytes == nil {
			break
		}
		blockCount++
	}
//
	blockBytes, err := s.nextBlockBytes()
	assert.Nil(t, blockBytes)
	assert.NoError(t, err, "Error in getting next block after exhausting the file")
	assert.Equal(t, numFiles*numBlocksInEachFile, blockCount)
}
