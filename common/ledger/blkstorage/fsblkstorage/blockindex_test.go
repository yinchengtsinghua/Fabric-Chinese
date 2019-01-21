
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
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	putil "github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

type noopIndex struct {
}

func (i *noopIndex) getLastBlockIndexed() (uint64, error) {
	return 0, nil
}
func (i *noopIndex) indexBlock(blockIdxInfo *blockIdxInfo) error {
	return nil
}
func (i *noopIndex) getBlockLocByHash(blockHash []byte) (*fileLocPointer, error) {
	return nil, nil
}
func (i *noopIndex) getBlockLocByBlockNum(blockNum uint64) (*fileLocPointer, error) {
	return nil, nil
}
func (i *noopIndex) getTxLoc(txID string) (*fileLocPointer, error) {
	return nil, nil
}
func (i *noopIndex) getTXLocByBlockNumTranNum(blockNum uint64, tranNum uint64) (*fileLocPointer, error) {
	return nil, nil
}

func (i *noopIndex) getBlockLocByTxID(txID string) (*fileLocPointer, error) {
	return nil, nil
}

func (i *noopIndex) getTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	return peer.TxValidationCode(-1), nil
}

func TestBlockIndexSync(t *testing.T) {
	testBlockIndexSync(t, 10, 5, false)
	testBlockIndexSync(t, 10, 5, true)
	testBlockIndexSync(t, 10, 0, true)
	testBlockIndexSync(t, 10, 10, true)
}

func testBlockIndexSync(t *testing.T, numBlocks int, numBlocksToIndex int, syncByRestart bool) {
	testName := fmt.Sprintf("%v/%v/%v", numBlocks, numBlocksToIndex, syncByRestart)
	t.Run(testName, func(t *testing.T) {
		env := newTestEnv(t, NewConf(testPath(), 0))
		defer env.Cleanup()
		ledgerid := "testledger"
		blkfileMgrWrapper := newTestBlockfileWrapper(env, ledgerid)
		defer blkfileMgrWrapper.close()
		blkfileMgr := blkfileMgrWrapper.blockfileMgr
		origIndex := blkfileMgr.index
//构建测试块
		blocks := testutil.ConstructTestBlocks(t, numBlocks)
//添加几个块
		blkfileMgrWrapper.addBlocks(blocks[:numBlocksToIndex])

//插入noop索引并添加剩余块
		blkfileMgr.index = &noopIndex{}
		blkfileMgrWrapper.addBlocks(blocks[numBlocksToIndex:])

//重新插入原始索引
		blkfileMgr.index = origIndex
//第一组块应该出现在原始索引中
		for i := 0; i < numBlocksToIndex; i++ {
			block, err := blkfileMgr.retrieveBlockByNumber(uint64(i))
			assert.NoError(t, err, "block [%d] should have been present in the index", i)
			assert.Equal(t, blocks[i], block)
		}

//最后一组块不应出现在原始索引中
		for i := numBlocksToIndex + 1; i <= numBlocks; i++ {
			_, err := blkfileMgr.retrieveBlockByNumber(uint64(i))
			assert.Exactly(t, blkstorage.ErrNotFoundInIndex, err)
		}

//执行索引同步
		if syncByRestart {
			blkfileMgrWrapper.close()
			blkfileMgrWrapper = newTestBlockfileWrapper(env, ledgerid)
			defer blkfileMgrWrapper.close()
			blkfileMgr = blkfileMgrWrapper.blockfileMgr
		} else {
			blkfileMgr.syncIndex()
		}

//现在，最后一组块也应该出现在原始索引中
		for i := numBlocksToIndex; i < numBlocks; i++ {
			block, err := blkfileMgr.retrieveBlockByNumber(uint64(i))
			assert.NoError(t, err, "block [%d] should have been present in the index", i)
			assert.Equal(t, blocks[i], block)
		}
	})
}

func TestBlockIndexSelectiveIndexingWrongConfig(t *testing.T) {
	testBlockIndexSelectiveIndexingWrongConfig(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrBlockTxID})
	testBlockIndexSelectiveIndexingWrongConfig(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrTxValidationCode})
	testBlockIndexSelectiveIndexingWrongConfig(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrBlockTxID, blkstorage.IndexableAttrBlockNum})
	testBlockIndexSelectiveIndexingWrongConfig(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrTxValidationCode, blkstorage.IndexableAttrBlockNumTranNum})

}

func testBlockIndexSelectiveIndexingWrongConfig(t *testing.T, indexItems []blkstorage.IndexableAttr) {
	var testName string
	for _, s := range indexItems {
		testName = testName + string(s)
	}
	t.Run(testName, func(t *testing.T) {
		env := newTestEnvSelectiveIndexing(t, NewConf(testPath(), 0), indexItems)
		defer env.Cleanup()

		assert.Panics(t, func() {
			env.provider.OpenBlockStore("test-ledger")
		}, "A panic is expected when index is opened with wrong configs")
	})
}

func TestBlockIndexSelectiveIndexing(t *testing.T) {
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrBlockHash})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrBlockNum})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrTxID})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrBlockNumTranNum})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrBlockHash, blkstorage.IndexableAttrBlockNum})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrTxID, blkstorage.IndexableAttrBlockNumTranNum})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrTxID, blkstorage.IndexableAttrBlockTxID})
	testBlockIndexSelectiveIndexing(t, []blkstorage.IndexableAttr{blkstorage.IndexableAttrTxID, blkstorage.IndexableAttrTxValidationCode})
}

func testBlockIndexSelectiveIndexing(t *testing.T, indexItems []blkstorage.IndexableAttr) {
	var testName string
	for _, s := range indexItems {
		testName = testName + string(s)
	}
	t.Run(testName, func(t *testing.T) {
		env := newTestEnvSelectiveIndexing(t, NewConf(testPath(), 0), indexItems)
		defer env.Cleanup()
		blkfileMgrWrapper := newTestBlockfileWrapper(env, "testledger")
		defer blkfileMgrWrapper.close()

		blocks := testutil.ConstructTestBlocks(t, 3)
//添加测试块
		blkfileMgrWrapper.addBlocks(blocks)
		blockfileMgr := blkfileMgrWrapper.blockfileMgr

//如果已为index item配置索引，则应为该项编制索引，否则不
//测试“retrieveBlockByHash”
		block, err := blockfileMgr.retrieveBlockByHash(blocks[0].Header.Hash())
		if containsAttr(indexItems, blkstorage.IndexableAttrBlockHash) {
			assert.NoError(t, err, "Error while retrieving block by hash")
			assert.Equal(t, blocks[0], block)
		} else {
			assert.Exactly(t, blkstorage.ErrAttrNotIndexed, err)
		}

//测试“retrieveBlockByNumber”
		block, err = blockfileMgr.retrieveBlockByNumber(0)
		if containsAttr(indexItems, blkstorage.IndexableAttrBlockNum) {
			assert.NoError(t, err, "Error while retrieving block by number")
			assert.Equal(t, blocks[0], block)
		} else {
			assert.Exactly(t, blkstorage.ErrAttrNotIndexed, err)
		}

//测试“RetrieveTransactionByID”
		txid, err := extractTxID(blocks[0].Data.Data[0])
		assert.NoError(t, err)
		txEnvelope, err := blockfileMgr.retrieveTransactionByID(txid)
		if containsAttr(indexItems, blkstorage.IndexableAttrTxID) {
			assert.NoError(t, err, "Error while retrieving tx by id")
			txEnvelopeBytes := blocks[0].Data.Data[0]
			txEnvelopeOrig, err := putil.GetEnvelopeFromBlock(txEnvelopeBytes)
			assert.NoError(t, err)
			assert.Equal(t, txEnvelopeOrig, txEnvelope)
		} else {
			assert.Exactly(t, blkstorage.ErrAttrNotIndexed, err)
		}

//测试“retrieveTransActionsByBlockNumTranNum”
		txEnvelope2, err := blockfileMgr.retrieveTransactionByBlockNumTranNum(0, 0)
		if containsAttr(indexItems, blkstorage.IndexableAttrBlockNumTranNum) {
			assert.NoError(t, err, "Error while retrieving tx by blockNum and tranNum")
			txEnvelopeBytes2 := blocks[0].Data.Data[0]
			txEnvelopeOrig2, err2 := putil.GetEnvelopeFromBlock(txEnvelopeBytes2)
			assert.NoError(t, err2)
			assert.Equal(t, txEnvelopeOrig2, txEnvelope2)
		} else {
			assert.Exactly(t, blkstorage.ErrAttrNotIndexed, err)
		}

//测试“retrieveblockbytxid”
		txid, err = extractTxID(blocks[0].Data.Data[0])
		assert.NoError(t, err)
		block, err = blockfileMgr.retrieveBlockByTxID(txid)
		if containsAttr(indexItems, blkstorage.IndexableAttrBlockTxID) {
			assert.NoError(t, err, "Error while retrieving block by txID")
			assert.Equal(t, block, blocks[0])
		} else {
			assert.Exactly(t, blkstorage.ErrAttrNotIndexed, err)
		}

		for _, block := range blocks {
			flags := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])

			for idx, d := range block.Data.Data {
				txid, err = extractTxID(d)
				assert.NoError(t, err)

				reason, err := blockfileMgr.retrieveTxValidationCodeByTxID(txid)

				if containsAttr(indexItems, blkstorage.IndexableAttrTxValidationCode) {
					assert.NoError(t, err, "Error while retrieving tx validation code by txID")

					reasonFromFlags := flags.Flag(idx)

					assert.Equal(t, reasonFromFlags, reason)
				} else {
					assert.Exactly(t, blkstorage.ErrAttrNotIndexed, err)
				}
			}
		}
	})
}

func containsAttr(indexItems []blkstorage.IndexableAttr, attr blkstorage.IndexableAttr) bool {
	for _, element := range indexItems {
		if element == attr {
			return true
		}
	}
	return false
}
