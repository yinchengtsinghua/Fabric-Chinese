
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
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	ledgerutil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	putil "github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func TestBlockfileMgrBlockReadWrite(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkfileMgrWrapper := newTestBlockfileWrapper(env, "testLedger")
	defer blkfileMgrWrapper.close()
	blocks := testutil.ConstructTestBlocks(t, 10)
	blkfileMgrWrapper.addBlocks(blocks)
	blkfileMgrWrapper.testGetBlockByHash(blocks)
	blkfileMgrWrapper.testGetBlockByNumber(blocks, 0)
}

func TestAddBlockWithWrongHash(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkfileMgrWrapper := newTestBlockfileWrapper(env, "testLedger")
	defer blkfileMgrWrapper.close()
	blocks := testutil.ConstructTestBlocks(t, 10)
	blkfileMgrWrapper.addBlocks(blocks[0:9])
	lastBlock := blocks[9]
lastBlock.Header.PreviousHash = []byte("someJunkHash") //将哈希设置为意外的值
	err := blkfileMgrWrapper.blockfileMgr.addBlock(lastBlock)
	assert.Error(t, err, "An error is expected when adding a block with some unexpected hash")
	assert.Contains(t, err.Error(), "unexpected Previous block hash. Expected PreviousHash")
	t.Logf("err = %s", err)
}

func TestBlockfileMgrCrashDuringWriting(t *testing.T) {
	testBlockfileMgrCrashDuringWriting(t, 10, 2, 1000, 10)
	testBlockfileMgrCrashDuringWriting(t, 10, 2, 1000, 1)
	testBlockfileMgrCrashDuringWriting(t, 10, 2, 1000, 0)
	testBlockfileMgrCrashDuringWriting(t, 0, 0, 1000, 10)
	testBlockfileMgrCrashDuringWriting(t, 0, 5, 1000, 10)
}

func testBlockfileMgrCrashDuringWriting(t *testing.T, numBlocksBeforeCheckpoint int,
	numBlocksAfterCheckpoint int, numLastBlockBytes int, numPartialBytesToWrite int) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	ledgerid := "testLedger"
	blkfileMgrWrapper := newTestBlockfileWrapper(env, ledgerid)
	bg, gb := testutil.NewBlockGenerator(t, ledgerid, false)

//创建所有必需的块
	totalBlocks := numBlocksBeforeCheckpoint + numBlocksAfterCheckpoint
	allBlocks := []*common.Block{gb}
	allBlocks = append(allBlocks, bg.NextTestBlocks(totalBlocks+1)...)

//标识要在cp之前、cp之后和重新启动之后添加的块
	blocksBeforeCP := []*common.Block{}
	blocksAfterCP := []*common.Block{}
	if numBlocksBeforeCheckpoint != 0 {
		blocksBeforeCP = allBlocks[0:numBlocksBeforeCheckpoint]
	}
	if numBlocksAfterCheckpoint != 0 {
		blocksAfterCP = allBlocks[numBlocksBeforeCheckpoint : numBlocksBeforeCheckpoint+numBlocksAfterCheckpoint]
	}
	blocksAfterRestart := allBlocks[numBlocksBeforeCheckpoint+numBlocksAfterCheckpoint:]

//在CP前添加块
	blkfileMgrWrapper.addBlocks(blocksBeforeCP)
	currentCPInfo := blkfileMgrWrapper.blockfileMgr.cpInfo
	cpInfo1 := &checkpointInfo{
		currentCPInfo.latestFileChunkSuffixNum,
		currentCPInfo.latestFileChunksize,
		currentCPInfo.isChainEmpty,
		currentCPInfo.lastBlockNumber}

//在cp后添加块
	blkfileMgrWrapper.addBlocks(blocksAfterCP)
	cpInfo2 := blkfileMgrWrapper.blockfileMgr.cpInfo

//模拟崩溃场景
	lastBlockBytes := []byte{}
	encodedLen := proto.EncodeVarint(uint64(numLastBlockBytes))
	randomBytes := testutil.ConstructRandomBytes(t, numLastBlockBytes)
	lastBlockBytes = append(lastBlockBytes, encodedLen...)
	lastBlockBytes = append(lastBlockBytes, randomBytes...)
	partialBytes := lastBlockBytes[:numPartialBytesToWrite]
	blkfileMgrWrapper.blockfileMgr.currentFileWriter.append(partialBytes, true)
	blkfileMgrWrapper.blockfileMgr.saveCurrentInfo(cpInfo1, true)
	blkfileMgrWrapper.close()

//模拟碰撞后的启动
	blkfileMgrWrapper = newTestBlockfileWrapper(env, ledgerid)
	defer blkfileMgrWrapper.close()
	cpInfo3 := blkfileMgrWrapper.blockfileMgr.cpInfo
	assert.Equal(t, cpInfo2, cpInfo3)

//重新启动后添加新块
	blkfileMgrWrapper.addBlocks(blocksAfterRestart)
	testBlockfileMgrBlockIterator(t, blkfileMgrWrapper.blockfileMgr, 0, len(allBlocks)-1, allBlocks)
}

func TestBlockfileMgrBlockIterator(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkfileMgrWrapper := newTestBlockfileWrapper(env, "testLedger")
	defer blkfileMgrWrapper.close()
	blocks := testutil.ConstructTestBlocks(t, 10)
	blkfileMgrWrapper.addBlocks(blocks)
	testBlockfileMgrBlockIterator(t, blkfileMgrWrapper.blockfileMgr, 0, 7, blocks[0:8])
}

func testBlockfileMgrBlockIterator(t *testing.T, blockfileMgr *blockfileMgr,
	firstBlockNum int, lastBlockNum int, expectedBlocks []*common.Block) {
	itr, err := blockfileMgr.retrieveBlocks(uint64(firstBlockNum))
	defer itr.Close()
	assert.NoError(t, err, "Error while getting blocks iterator")
	numBlocksItrated := 0
	for {
		block, err := itr.Next()
		assert.NoError(t, err, "Error while getting block number [%d] from iterator", numBlocksItrated)
		assert.Equal(t, expectedBlocks[numBlocksItrated], block)
		numBlocksItrated++
		if numBlocksItrated == lastBlockNum-firstBlockNum+1 {
			break
		}
	}
	assert.Equal(t, lastBlockNum-firstBlockNum+1, numBlocksItrated)
}

func TestBlockfileMgrBlockchainInfo(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkfileMgrWrapper := newTestBlockfileWrapper(env, "testLedger")
	defer blkfileMgrWrapper.close()

	bcInfo := blkfileMgrWrapper.blockfileMgr.getBlockchainInfo()
	assert.Equal(t, &common.BlockchainInfo{Height: 0, CurrentBlockHash: nil, PreviousBlockHash: nil}, bcInfo)

	blocks := testutil.ConstructTestBlocks(t, 10)
	blkfileMgrWrapper.addBlocks(blocks)
	bcInfo = blkfileMgrWrapper.blockfileMgr.getBlockchainInfo()
	assert.Equal(t, uint64(10), bcInfo.Height)
}

func TestBlockfileMgrGetTxById(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkfileMgrWrapper := newTestBlockfileWrapper(env, "testLedger")
	defer blkfileMgrWrapper.close()
	blocks := testutil.ConstructTestBlocks(t, 2)
	blkfileMgrWrapper.addBlocks(blocks)
	for _, blk := range blocks {
		for j, txEnvelopeBytes := range blk.Data.Data {
//blocknum以0开头
			txID, err := extractTxID(blk.Data.Data[j])
			assert.NoError(t, err)
			txEnvelopeFromFileMgr, err := blkfileMgrWrapper.blockfileMgr.retrieveTransactionByID(txID)
			assert.NoError(t, err, "Error while retrieving tx from blkfileMgr")
			txEnvelope, err := putil.GetEnvelopeFromBlock(txEnvelopeBytes)
			assert.NoError(t, err, "Error while unmarshalling tx")
			assert.Equal(t, txEnvelope, txEnvelopeFromFileMgr)
		}
	}
}

//testblockfilemgrgettxbyidduplicatetxid测试具有现有txid的事务
//（在同一块或不同块内）不应按txid（fab-8557）重写索引。
func TestBlockfileMgrGetTxByIdDuplicateTxid(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkStore, err := env.provider.OpenBlockStore("testLedger")
	assert.NoError(env.t, err)
	blkFileMgr := blkStore.(*fsBlockStore).fileMgr
	bg, gb := testutil.NewBlockGenerator(t, "testLedger", false)
	assert.NoError(t, blkFileMgr.addBlock(gb))

	block1 := bg.NextBlockWithTxid(
		[][]byte{
			[]byte("tx with id=txid-1"),
			[]byte("tx with id=txid-2"),
			[]byte("another tx with existing id=txid-1"),
		},
		[]string{"txid-1", "txid-2", "txid-1"},
	)
	txValidationFlags := ledgerutil.NewTxValidationFlags(3)
	txValidationFlags.SetFlag(0, peer.TxValidationCode_VALID)
	txValidationFlags.SetFlag(1, peer.TxValidationCode_INVALID_OTHER_REASON)
	txValidationFlags.SetFlag(2, peer.TxValidationCode_DUPLICATE_TXID)
	block1.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txValidationFlags
	assert.NoError(t, blkFileMgr.addBlock(block1))

	block2 := bg.NextBlockWithTxid(
		[][]byte{
			[]byte("tx with id=txid-3"),
			[]byte("yet another tx with existing id=txid-1"),
		},
		[]string{"txid-3", "txid-1"},
	)
	txValidationFlags = ledgerutil.NewTxValidationFlags(2)
	txValidationFlags.SetFlag(0, peer.TxValidationCode_VALID)
	txValidationFlags.SetFlag(1, peer.TxValidationCode_DUPLICATE_TXID)
	block2.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txValidationFlags
	assert.NoError(t, blkFileMgr.addBlock(block2))

	txenvp1, err := putil.GetEnvelopeFromBlock(block1.Data.Data[0])
	assert.NoError(t, err)
	txenvp2, err := putil.GetEnvelopeFromBlock(block1.Data.Data[1])
	assert.NoError(t, err)
	txenvp3, err := putil.GetEnvelopeFromBlock(block2.Data.Data[0])
	assert.NoError(t, err)

	indexedTxenvp, _ := blkFileMgr.retrieveTransactionByID("txid-1")
	assert.Equal(t, txenvp1, indexedTxenvp)
	indexedTxenvp, _ = blkFileMgr.retrieveTransactionByID("txid-2")
	assert.Equal(t, txenvp2, indexedTxenvp)
	indexedTxenvp, _ = blkFileMgr.retrieveTransactionByID("txid-3")
	assert.Equal(t, txenvp3, indexedTxenvp)

	blk, _ := blkFileMgr.retrieveBlockByTxID("txid-1")
	assert.Equal(t, block1, blk)
	blk, _ = blkFileMgr.retrieveBlockByTxID("txid-2")
	assert.Equal(t, block1, blk)
	blk, _ = blkFileMgr.retrieveBlockByTxID("txid-3")
	assert.Equal(t, block2, blk)

	validationCode, _ := blkFileMgr.retrieveTxValidationCodeByTxID("txid-1")
	assert.Equal(t, peer.TxValidationCode_VALID, validationCode)
	validationCode, _ = blkFileMgr.retrieveTxValidationCodeByTxID("txid-2")
	assert.Equal(t, peer.TxValidationCode_INVALID_OTHER_REASON, validationCode)
	validationCode, _ = blkFileMgr.retrieveTxValidationCodeByTxID("txid-3")
	assert.Equal(t, peer.TxValidationCode_VALID, validationCode)
}

func TestBlockfileMgrGetTxByBlockNumTranNum(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkfileMgrWrapper := newTestBlockfileWrapper(env, "testLedger")
	defer blkfileMgrWrapper.close()
	blocks := testutil.ConstructTestBlocks(t, 10)
	blkfileMgrWrapper.addBlocks(blocks)
	for blockIndex, blk := range blocks {
		for tranIndex, txEnvelopeBytes := range blk.Data.Data {
//blocknum和trannum都以0开头
			txEnvelopeFromFileMgr, err := blkfileMgrWrapper.blockfileMgr.retrieveTransactionByBlockNumTranNum(uint64(blockIndex), uint64(tranIndex))
			assert.NoError(t, err, "Error while retrieving tx from blkfileMgr")
			txEnvelope, err := putil.GetEnvelopeFromBlock(txEnvelopeBytes)
			assert.NoError(t, err, "Error while unmarshalling tx")
			assert.Equal(t, txEnvelope, txEnvelopeFromFileMgr)
		}
	}
}

func TestBlockfileMgrRestart(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	ledgerid := "testLedger"
	blkfileMgrWrapper := newTestBlockfileWrapper(env, ledgerid)
	blocks := testutil.ConstructTestBlocks(t, 10)
	blkfileMgrWrapper.addBlocks(blocks)
	expectedHeight := uint64(10)
	assert.Equal(t, expectedHeight, blkfileMgrWrapper.blockfileMgr.getBlockchainInfo().Height)
	blkfileMgrWrapper.close()

	blkfileMgrWrapper = newTestBlockfileWrapper(env, ledgerid)
	defer blkfileMgrWrapper.close()
	assert.Equal(t, 9, int(blkfileMgrWrapper.blockfileMgr.cpInfo.lastBlockNumber))
	blkfileMgrWrapper.testGetBlockByHash(blocks)
	assert.Equal(t, expectedHeight, blkfileMgrWrapper.blockfileMgr.getBlockchainInfo().Height)
}

func TestBlockfileMgrFileRolling(t *testing.T) {
	blocks := testutil.ConstructTestBlocks(t, 200)
	size := 0
	for _, block := range blocks[:100] {
		by, _, err := serializeBlock(block)
		assert.NoError(t, err, "Error while serializing block")
		blockBytesSize := len(by)
		encodedLen := proto.EncodeVarint(uint64(blockBytesSize))
		size += blockBytesSize + len(encodedLen)
	}

	maxFileSie := int(0.75 * float64(size))
	env := newTestEnv(t, NewConf(testPath(), maxFileSie))
	defer env.Cleanup()
	ledgerid := "testLedger"
	blkfileMgrWrapper := newTestBlockfileWrapper(env, ledgerid)
	blkfileMgrWrapper.addBlocks(blocks[:100])
	assert.Equal(t, 1, blkfileMgrWrapper.blockfileMgr.cpInfo.latestFileChunkSuffixNum)
	blkfileMgrWrapper.testGetBlockByHash(blocks[:100])
	blkfileMgrWrapper.close()

	blkfileMgrWrapper = newTestBlockfileWrapper(env, ledgerid)
	defer blkfileMgrWrapper.close()
	blkfileMgrWrapper.addBlocks(blocks[100:])
	assert.Equal(t, 2, blkfileMgrWrapper.blockfileMgr.cpInfo.latestFileChunkSuffixNum)
	blkfileMgrWrapper.testGetBlockByHash(blocks[100:])
}

func TestBlockfileMgrGetBlockByTxID(t *testing.T) {
	env := newTestEnv(t, NewConf(testPath(), 0))
	defer env.Cleanup()
	blkfileMgrWrapper := newTestBlockfileWrapper(env, "testLedger")
	defer blkfileMgrWrapper.close()
	blocks := testutil.ConstructTestBlocks(t, 10)
	blkfileMgrWrapper.addBlocks(blocks)
	for _, blk := range blocks {
		for j := range blk.Data.Data {
//blocknum以1开头
			txID, err := extractTxID(blk.Data.Data[j])
			assert.NoError(t, err)

			blockFromFileMgr, err := blkfileMgrWrapper.blockfileMgr.retrieveBlockByTxID(txID)
			assert.NoError(t, err, "Error while retrieving block from blkfileMgr")
			assert.Equal(t, blk, blockFromFileMgr)
		}
	}
}
