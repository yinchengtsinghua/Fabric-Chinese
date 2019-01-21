
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
	"bytes"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/util"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	putil "github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("fsblkstorage")

const (
	blockfilePrefix = "blockfile_"
)

var (
	blkMgrInfoKey = []byte("blkMgrInfo")
)

type blockfileMgr struct {
	rootDir           string
	conf              *Conf
	db                *leveldbhelper.DBHandle
	index             index
	cpInfo            *checkpointInfo
	cpInfoCond        *sync.Cond
	currentFileWriter *blockfileWriter
	bcInfo            atomic.Value
}

/*
创建一个新的管理器，用于管理用于块持久性的文件。
此管理器管理文件系统fs，包括
  --存储文件的目录
  --存储块的单个文件
  --跟踪要持久化的最新文件的检查点
  --跟踪什么块和事务在哪个文件中的索引
当新的blockfile管理器启动时（即仅在启动时），它会检查
如果此启动是系统首次启动，还是重新启动
系统的。

块文件管理器将数据块存储到文件系统中。该文件
存储是通过创建按配置大小顺序编号的文件来完成的。
即blockfile_000000、blockfile_000001等。

块中的每个事务都存储有关于
该事务中的字节数
 将txloc[filesufixnum=0，offset=3，byteslength=104]添加到索引中。
 将txloc[filesufixnum=0，offset=107，byteslength=104]添加到索引中。
每个块与该块的总编码长度以及
Tx位置偏移。

请记住，这些步骤仅在系统启动时执行一次。
启动时，新经理：
  *）检查用于存储文件的目录是否存在，如果不存在，则创建目录
  *）检查键值数据库是否存在，如果不存在，则创建一个键值数据库。
       （将创建一个db dir）
  *）确定用于存储的检查点信息（cpinfo）
  --从数据库加载（如果存在），如果不实例化新的cpinfo
  --如果cpinfo是从db加载的，则与fs比较
  --如果cpinfo和文件系统不同步，则从fs同步cpinfo
  *）启动新的文件编写器
  --根据cpinfo截断文件以删除超过最后一个块的任何多余部分
  *）确定用于查找Tx和块的索引信息
  文件blkstorage
  --实例化一个新的blockIDxinfo
  --从数据库加载索引（如果存在）
  --同步索引将索引的最后一个块与fs中的块进行比较
  --如果索引和文件系统不同步，则从fs同步索引
  *）更新API使用的区块链信息
**/

func newBlockfileMgr(id string, conf *Conf, indexConfig *blkstorage.IndexConfig, indexStore *leveldbhelper.DBHandle) *blockfileMgr {
	logger.Debugf("newBlockfileMgr() initializing file-based block storage for ledger: %s ", id)
//确定块文件存储的根目录，如果不存在，则创建它
	rootDir := conf.getLedgerBlockDir(id)
	_, err := util.CreateDirIfMissing(rootDir)
	if err != nil {
		panic(fmt.Sprintf("Error creating block storage root dir [%s]: %s", rootDir, err))
	}
//实例化管理器，即blockfilemgr结构
	mgr := &blockfileMgr{rootDir: rootDir, conf: conf, db: indexStore}

//cp=checkpointinfo，从数据库中检索存储块的文件后缀或数量。
//它还检索该文件的当前大小和写入该文件的最后一个块号。
//在init checkpointinfo:latestfilechunksuffixnum=[0]，latestfilechunksize=[0]，lastblocknumber=[0]
	cpInfo, err := mgr.loadCurrentInfo()
	if err != nil {
		panic(fmt.Sprintf("Could not get block file info for current block file from db: %s", err))
	}
	if cpInfo == nil {
		logger.Info(`Getting block information from block storage`)
		if cpInfo, err = constructCheckpointInfoFromBlockFiles(rootDir); err != nil {
			panic(fmt.Sprintf("Could not build checkpoint info from block files: %s", err))
		}
		logger.Debugf("Info constructed by scanning the blocks dir = %s", spew.Sdump(cpInfo))
	} else {
		logger.Debug(`Synching block information from block storage (if needed)`)
		syncCPInfoFromFS(rootDir, cpInfo)
	}
	err = mgr.saveCurrentInfo(cpInfo, true)
	if err != nil {
		panic(fmt.Sprintf("Could not save next block file info to db: %s", err))
	}

//打开由数字标识的文件的写入程序，并将其截断为只包含最新的块
//已完全保存（文件系统、索引、cpinfo等）
	currentFileWriter, err := newBlockfileWriter(deriveBlockfilePath(rootDir, cpInfo.latestFileChunkSuffixNum))
	if err != nil {
		panic(fmt.Sprintf("Could not open writer to current file: %s", err))
	}
//截断文件以删除超过最后一个块的部分
	err = currentFileWriter.truncateFile(cpInfo.latestFileChunksize)
	if err != nil {
		panic(fmt.Sprintf("Could not truncate current file to known size in db: %s", err))
	}

//为keyValue数据库中的块索引创建新的keyValue存储数据库处理程序
	if mgr.index, err = newBlockIndex(indexConfig, indexStore); err != nil {
		panic(fmt.Sprintf("error in block index: %s", err))
	}

//使用检查点信息和文件写入程序更新管理器
	mgr.cpInfo = cpInfo
	mgr.currentFileWriter = currentFileWriter
//为等待的goroutine创建检查点条件（事件）变量
//或宣布事件的发生。
	mgr.cpInfoCond = sync.NewCond(&sync.Mutex{})

//外部API的init blockchaininfo
	bcInfo := &common.BlockchainInfo{
		Height:            0,
		CurrentBlockHash:  nil,
		PreviousBlockHash: nil}

	if !cpInfo.isChainEmpty {
//如果启动是现有存储的重新启动，请从块存储同步索引并更新外部API的blockchaininfo
		mgr.syncIndex()
		lastBlockHeader, err := mgr.retrieveBlockHeaderByNumber(cpInfo.lastBlockNumber)
		if err != nil {
			panic(fmt.Sprintf("Could not retrieve header of the last block form file: %s", err))
		}
		lastBlockHash := lastBlockHeader.Hash()
		previousBlockHash := lastBlockHeader.PreviousHash
		bcInfo = &common.BlockchainInfo{
			Height:            cpInfo.lastBlockNumber + 1,
			CurrentBlockHash:  lastBlockHash,
			PreviousBlockHash: previousBlockHash}
	}
	mgr.bcInfo.Store(bcInfo)
	return mgr
}

//cp=checkpointinfo，从数据库获取文件后缀和
//写入最后一个块的文件。还检索包含
//写入的最后一个块号。在初始化时
//检查点信息：LatestFileChunkSuffixNum=[0]，LatestFileChunkSize=[0]，LastBlockNumber=[0]
func syncCPInfoFromFS(rootDir string, cpInfo *checkpointInfo) {
	logger.Debugf("Starting checkpoint=%s", cpInfo)
//检查写入最后一个块的文件后缀是否存在
	filePath := deriveBlockfilePath(rootDir, cpInfo.latestFileChunkSuffixNum)
	exists, size, err := util.FileExists(filePath)
	if err != nil {
		panic(fmt.Sprintf("Error in checking whether file [%s] exists: %s", filePath, err))
	}
	logger.Debugf("status of file [%s]: exists=[%t], size=[%d]", filePath, exists, size)
//测试是！存在，因为首次使用文件号时，文件尚不存在
//检查文件是否存在以及文件的大小是否与存储在cpinfo中的大小相同
//文件状态[/tmp/tests/ledger/blkstorage/fsblkstorage/blocks/blockfile_000000]：exists=[false]，size=[0]
	if !exists || int(size) == cpInfo.latestFileChunksize {
//检查点信息与磁盘上的文件同步
		return
	}
//扫描文件系统以验证存储在数据库中的检查点信息是否正确。
	_, endOffsetLastBlock, numBlocks, err := scanForLastCompleteBlock(
		rootDir, cpInfo.latestFileChunkSuffixNum, int64(cpInfo.latestFileChunksize))
	if err != nil {
		panic(fmt.Sprintf("Could not open current file for detecting last block in the file: %s", err))
	}
	cpInfo.latestFileChunksize = int(endOffsetLastBlock)
	if numBlocks == 0 {
		return
	}
//更新实际存储的最后一个块号及其结束位置的检查点信息
	if cpInfo.isChainEmpty {
		cpInfo.lastBlockNumber = uint64(numBlocks - 1)
	} else {
		cpInfo.lastBlockNumber += uint64(numBlocks)
	}
	cpInfo.isChainEmpty = false
	logger.Debugf("Checkpoint after updates by scanning the last file segment:%s", cpInfo)
}

func deriveBlockfilePath(rootDir string, suffixNum int) string {
	return rootDir + "/" + blockfilePrefix + fmt.Sprintf("%06d", suffixNum)
}

func (mgr *blockfileMgr) close() {
	mgr.currentFileWriter.close()
}

func (mgr *blockfileMgr) moveToNextFile() {
	cpInfo := &checkpointInfo{
		latestFileChunkSuffixNum: mgr.cpInfo.latestFileChunkSuffixNum + 1,
		latestFileChunksize:      0,
		lastBlockNumber:          mgr.cpInfo.lastBlockNumber}

	nextFileWriter, err := newBlockfileWriter(
		deriveBlockfilePath(mgr.rootDir, cpInfo.latestFileChunkSuffixNum))

	if err != nil {
		panic(fmt.Sprintf("Could not open writer to next file: %s", err))
	}
	mgr.currentFileWriter.close()
	err = mgr.saveCurrentInfo(cpInfo, true)
	if err != nil {
		panic(fmt.Sprintf("Could not save next block file info to db: %s", err))
	}
	mgr.currentFileWriter = nextFileWriter
	mgr.updateCheckpoint(cpInfo)
}

func (mgr *blockfileMgr) addBlock(block *common.Block) error {
	bcInfo := mgr.getBlockchainInfo()
	if block.Header.Number != bcInfo.Height {
		return errors.Errorf(
			"block number should have been %d but was %d",
			mgr.getBlockchainInfo().Height, block.Header.Number,
		)
	}

//添加上一个哈希检查-虽然不是必需的，但对
//验证块中是否存在字段“block.header.previouushash”。
//此检查是一个简单的字节比较，因此不会造成任何可观察的性能损失。
//如果订购服务中存在任何错误，则可能有助于检测罕见的情况。
	if !bytes.Equal(block.Header.PreviousHash, bcInfo.CurrentBlockHash) {
		return errors.Errorf(
			"unexpected Previous block hash. Expected PreviousHash = [%x], PreviousHash referred in the latest block= [%x]",
			bcInfo.CurrentBlockHash, block.Header.PreviousHash,
		)
	}
	blockBytes, info, err := serializeBlock(block)
	if err != nil {
		return errors.WithMessage(err, "error serializing block")
	}
	blockHash := block.Header.Hash()
//获取每个事务在块中开始和块结束的位置/偏移量
	txOffsets := info.txOffsets
	currentOffset := mgr.cpInfo.latestFileChunksize

	blockBytesLen := len(blockBytes)
	blockBytesEncodedLen := proto.EncodeVarint(uint64(blockBytesLen))
	totalBytesToAppend := blockBytesLen + len(blockBytesEncodedLen)

//确定是否需要启动一个新文件，因为此块的大小
//超出当前文件中剩余的空间量
	if currentOffset+totalBytesToAppend > mgr.conf.maxBlockfileSize {
		mgr.moveToNextFile()
		currentOffset = 0
	}
//将BlockBytesEncodedLen附加到文件
	err = mgr.currentFileWriter.append(blockBytesEncodedLen, false)
	if err == nil {
//将实际块字节附加到文件
		err = mgr.currentFileWriter.append(blockBytes, true)
	}
	if err != nil {
		truncateErr := mgr.currentFileWriter.truncateFile(mgr.cpInfo.latestFileChunksize)
		if truncateErr != nil {
			panic(fmt.Sprintf("Could not truncate current file to known size after an error during block append: %s", err))
		}
		return errors.WithMessage(err, "error appending block to file")
	}

//使用添加新块的结果更新检查点信息
	currentCPInfo := mgr.cpInfo
	newCPInfo := &checkpointInfo{
		latestFileChunkSuffixNum: currentCPInfo.latestFileChunkSuffixNum,
		latestFileChunksize:      currentCPInfo.latestFileChunksize + totalBytesToAppend,
		isChainEmpty:             false,
		lastBlockNumber:          block.Header.Number}
//将检查点信息保存到数据库中
	if err = mgr.saveCurrentInfo(newCPInfo, false); err != nil {
		truncateErr := mgr.currentFileWriter.truncateFile(currentCPInfo.latestFileChunksize)
		if truncateErr != nil {
			panic(fmt.Sprintf("Error in truncating current file to known size after an error in saving checkpoint info: %s", err))
		}
		return errors.WithMessage(err, "error saving current file info to db")
	}

//索引块文件位置指针用文件suffex和新块的偏移量更新
	blockFLP := &fileLocPointer{fileSuffixNum: newCPInfo.latestFileChunkSuffixNum}
	blockFLP.offset = currentOffset
//移动txoffset，因为我们在块字节之前预留了字节长度
	for _, txOffset := range txOffsets {
		txOffset.loc.offset += len(blockBytesEncodedLen)
	}
//将索引保存在数据库中
	if err = mgr.index.indexBlock(&blockIdxInfo{
		blockNum: block.Header.Number, blockHash: blockHash,
		flp: blockFLP, txOffsets: txOffsets, metadata: block.Metadata}); err != nil {
		return err
	}

//在管理器中更新检查点信息（用于存储）和区块链信息（用于API）
	mgr.updateCheckpoint(newCPInfo)
	mgr.updateBlockchainInfo(blockHash, block)
	return nil
}

func (mgr *blockfileMgr) syncIndex() error {
	var lastBlockIndexed uint64
	var indexEmpty bool
	var err error
//从数据库中，获取索引的最后一个块
	if lastBlockIndexed, err = mgr.index.getLastBlockIndexed(); err != nil {
		if err != errIndexEmpty {
			return err
		}
		indexEmpty = true
	}

//将索引初始化为文件号：零、偏移量：零和blocknum:0
	startFileNum := 0
	startOffset := 0
	skipFirstBlock := false
//获取使用检查点信息添加块的最后一个文件
	endFileNum := mgr.cpInfo.latestFileChunkSuffixNum
	startingBlockNum := uint64(0)

//如果存储在数据库中的索引具有值，请使用这些值更新索引信息。
	if !indexEmpty {
		if lastBlockIndexed == mgr.cpInfo.lastBlockNumber {
			logger.Debug("Both the block files and indices are in sync.")
			return nil
		}
		logger.Debugf("Last block indexed [%d], Last block present in block files [%d]", lastBlockIndexed, mgr.cpInfo.lastBlockNumber)
		var flp *fileLocPointer
		if flp, err = mgr.index.getBlockLocByBlockNum(lastBlockIndexed); err != nil {
			return err
		}
		startFileNum = flp.fileSuffixNum
		startOffset = flp.locPointer.offset
		skipFirstBlock = true
		startingBlockNum = lastBlockIndexed + 1
	} else {
		logger.Debugf("No block indexed, Last block present in block files=[%d]", mgr.cpInfo.lastBlockNumber)
	}

	logger.Infof("Start building index from block [%d] to last block [%d]", startingBlockNum, mgr.cpInfo.lastBlockNumber)

//打开块流到存储在索引中的文件位置
	var stream *blockStream
	if stream, err = newBlockStream(mgr.rootDir, startFileNum, int64(startOffset), endFileNum); err != nil {
		return err
	}
	var blockBytes []byte
	var blockPlacementInfo *blockPlacementInfo

	if skipFirstBlock {
		if blockBytes, _, err = stream.nextBlockBytesAndPlacementInfo(); err != nil {
			return err
		}
		if blockBytes == nil {
			return errors.Errorf("block bytes for block num = [%d] should not be nil here. The indexes for the block are already present",
				lastBlockIndexed)
		}
	}

//应该已经在最后一个块了，但是继续循环查找下一个块字节。
//如果有其他块，请将其添加到索引中。
//这将确保块索引是正确的，例如，如果对等机在索引更新之前崩溃。
	blockIdxInfo := &blockIdxInfo{}
	for {
		if blockBytes, blockPlacementInfo, err = stream.nextBlockBytesAndPlacementInfo(); err != nil {
			return err
		}
		if blockBytes == nil {
			break
		}
		info, err := extractSerializedBlockInfo(blockBytes)
		if err != nil {
			return err
		}

//blockStartOffset将在indexBlock（）内建立索引之前应用于txOffset，
//因此，只需通过blockbytesoffset和blockstartoffset之间的差异进行移位
		numBytesToShift := int(blockPlacementInfo.blockBytesOffset - blockPlacementInfo.blockStartOffset)
		for _, offset := range info.txOffsets {
			offset.loc.offset += numBytesToShift
		}

//使用文件系统中实际存储的内容更新blockindexinfo
		blockIdxInfo.blockHash = info.blockHeader.Hash()
		blockIdxInfo.blockNum = info.blockHeader.Number
		blockIdxInfo.flp = &fileLocPointer{fileSuffixNum: blockPlacementInfo.fileNum,
			locPointer: locPointer{offset: int(blockPlacementInfo.blockStartOffset)}}
		blockIdxInfo.txOffsets = info.txOffsets
		blockIdxInfo.metadata = info.metadata

		logger.Debugf("syncIndex() indexing block [%d]", blockIdxInfo.blockNum)
		if err = mgr.index.indexBlock(blockIdxInfo); err != nil {
			return err
		}
		if blockIdxInfo.blockNum%10000 == 0 {
			logger.Infof("Indexed block number [%d]", blockIdxInfo.blockNum)
		}
	}
	logger.Infof("Finished building index. Last block indexed [%d]", blockIdxInfo.blockNum)
	return nil
}

func (mgr *blockfileMgr) getBlockchainInfo() *common.BlockchainInfo {
	return mgr.bcInfo.Load().(*common.BlockchainInfo)
}

func (mgr *blockfileMgr) updateCheckpoint(cpInfo *checkpointInfo) {
	mgr.cpInfoCond.L.Lock()
	defer mgr.cpInfoCond.L.Unlock()
	mgr.cpInfo = cpInfo
	logger.Debugf("Broadcasting about update checkpointInfo: %s", cpInfo)
	mgr.cpInfoCond.Broadcast()
}

func (mgr *blockfileMgr) updateBlockchainInfo(latestBlockHash []byte, latestBlock *common.Block) {
	currentBCInfo := mgr.getBlockchainInfo()
	newBCInfo := &common.BlockchainInfo{
		Height:            currentBCInfo.Height + 1,
		CurrentBlockHash:  latestBlockHash,
		PreviousBlockHash: latestBlock.Header.PreviousHash}

	mgr.bcInfo.Store(newBCInfo)
}

func (mgr *blockfileMgr) retrieveBlockByHash(blockHash []byte) (*common.Block, error) {
	logger.Debugf("retrieveBlockByHash() - blockHash = [%#v]", blockHash)
	loc, err := mgr.index.getBlockLocByHash(blockHash)
	if err != nil {
		return nil, err
	}
	return mgr.fetchBlock(loc)
}

func (mgr *blockfileMgr) retrieveBlockByNumber(blockNum uint64) (*common.Block, error) {
	logger.Debugf("retrieveBlockByNumber() - blockNum = [%d]", blockNum)

//
	if blockNum == math.MaxUint64 {
		blockNum = mgr.getBlockchainInfo().Height - 1
	}

	loc, err := mgr.index.getBlockLocByBlockNum(blockNum)
	if err != nil {
		return nil, err
	}
	return mgr.fetchBlock(loc)
}

func (mgr *blockfileMgr) retrieveBlockByTxID(txID string) (*common.Block, error) {
	logger.Debugf("retrieveBlockByTxID() - txID = [%s]", txID)

	loc, err := mgr.index.getBlockLocByTxID(txID)

	if err != nil {
		return nil, err
	}
	return mgr.fetchBlock(loc)
}

func (mgr *blockfileMgr) retrieveTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	logger.Debugf("retrieveTxValidationCodeByTxID() - txID = [%s]", txID)
	return mgr.index.getTxValidationCodeByTxID(txID)
}

func (mgr *blockfileMgr) retrieveBlockHeaderByNumber(blockNum uint64) (*common.BlockHeader, error) {
	logger.Debugf("retrieveBlockHeaderByNumber() - blockNum = [%d]", blockNum)
	loc, err := mgr.index.getBlockLocByBlockNum(blockNum)
	if err != nil {
		return nil, err
	}
	blockBytes, err := mgr.fetchBlockBytes(loc)
	if err != nil {
		return nil, err
	}
	info, err := extractSerializedBlockInfo(blockBytes)
	if err != nil {
		return nil, err
	}
	return info.blockHeader, nil
}

func (mgr *blockfileMgr) retrieveBlocks(startNum uint64) (*blocksItr, error) {
	return newBlockItr(mgr, startNum), nil
}

func (mgr *blockfileMgr) retrieveTransactionByID(txID string) (*common.Envelope, error) {
	logger.Debugf("retrieveTransactionByID() - txId = [%s]", txID)
	loc, err := mgr.index.getTxLoc(txID)
	if err != nil {
		return nil, err
	}
	return mgr.fetchTransactionEnvelope(loc)
}

func (mgr *blockfileMgr) retrieveTransactionByBlockNumTranNum(blockNum uint64, tranNum uint64) (*common.Envelope, error) {
	logger.Debugf("retrieveTransactionByBlockNumTranNum() - blockNum = [%d], tranNum = [%d]", blockNum, tranNum)
	loc, err := mgr.index.getTXLocByBlockNumTranNum(blockNum, tranNum)
	if err != nil {
		return nil, err
	}
	return mgr.fetchTransactionEnvelope(loc)
}

func (mgr *blockfileMgr) fetchBlock(lp *fileLocPointer) (*common.Block, error) {
	blockBytes, err := mgr.fetchBlockBytes(lp)
	if err != nil {
		return nil, err
	}
	block, err := deserializeBlock(blockBytes)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (mgr *blockfileMgr) fetchTransactionEnvelope(lp *fileLocPointer) (*common.Envelope, error) {
	logger.Debugf("Entering fetchTransactionEnvelope() %v\n", lp)
	var err error
	var txEnvelopeBytes []byte
	if txEnvelopeBytes, err = mgr.fetchRawBytes(lp); err != nil {
		return nil, err
	}
	_, n := proto.DecodeVarint(txEnvelopeBytes)
	return putil.GetEnvelopeFromBlock(txEnvelopeBytes[n:])
}

func (mgr *blockfileMgr) fetchBlockBytes(lp *fileLocPointer) ([]byte, error) {
	stream, err := newBlockfileStream(mgr.rootDir, lp.fileSuffixNum, int64(lp.offset))
	if err != nil {
		return nil, err
	}
	defer stream.close()
	b, err := stream.nextBlockBytes()
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (mgr *blockfileMgr) fetchRawBytes(lp *fileLocPointer) ([]byte, error) {
	filePath := deriveBlockfilePath(mgr.rootDir, lp.fileSuffixNum)
	reader, err := newBlockfileReader(filePath)
	if err != nil {
		return nil, err
	}
	defer reader.close()
	b, err := reader.read(lp.offset, lp.bytesLength)
	if err != nil {
		return nil, err
	}
	return b, nil
}

//获取存储在数据库中的当前检查点信息
func (mgr *blockfileMgr) loadCurrentInfo() (*checkpointInfo, error) {
	var b []byte
	var err error
	if b, err = mgr.db.Get(blkMgrInfoKey); b == nil || err != nil {
		return nil, err
	}
	i := &checkpointInfo{}
	if err = i.unmarshal(b); err != nil {
		return nil, err
	}
	logger.Debugf("loaded checkpointInfo:%s", i)
	return i, nil
}

func (mgr *blockfileMgr) saveCurrentInfo(i *checkpointInfo, sync bool) error {
	b, err := i.marshal()
	if err != nil {
		return err
	}
	if err = mgr.db.Put(blkMgrInfoKey, b, sync); err != nil {
		return err
	}
	return nil
}

//scanforlastcompleteblock扫描给定的块文件并检测文件中的最后一个偏移量
//在这之后，可能会有一个部分写入的块（在崩溃场景中接近文件的末尾）。
func scanForLastCompleteBlock(rootDir string, fileNum int, startingOffset int64) ([]byte, int64, int, error) {
//从传递的偏移量开始扫描传递的文件编号后缀，以查找最后完成的块。
	numBlocks := 0
	var lastBlockBytes []byte
	blockStream, errOpen := newBlockfileStream(rootDir, fileNum, startingOffset)
	if errOpen != nil {
		return nil, 0, 0, errOpen
	}
	defer blockStream.close()
	var errRead error
	var blockBytes []byte
	for {
		blockBytes, errRead = blockStream.nextBlockBytes()
		if blockBytes == nil || errRead != nil {
			break
		}
		lastBlockBytes = blockBytes
		numBlocks++
	}
	if errRead == ErrUnexpectedEndOfBlockfile {
		logger.Debugf(`Error:%s
		The error may happen if a crash has happened during block appending.
		Resetting error to nil and returning current offset as a last complete block's end offset`, errRead)
		errRead = nil
	}
	logger.Debugf("scanForLastCompleteBlock(): last complete block ends at offset=[%d]", blockStream.currentOffset)
	return lastBlockBytes, blockStream.currentOffset, numBlocks, errRead
}

//检查点信息
type checkpointInfo struct {
	latestFileChunkSuffixNum int
	latestFileChunksize      int
	isChainEmpty             bool
	lastBlockNumber          uint64
}

func (i *checkpointInfo) marshal() ([]byte, error) {
	buffer := proto.NewBuffer([]byte{})
	var err error
	if err = buffer.EncodeVarint(uint64(i.latestFileChunkSuffixNum)); err != nil {
		return nil, err
	}
	if err = buffer.EncodeVarint(uint64(i.latestFileChunksize)); err != nil {
		return nil, err
	}
	if err = buffer.EncodeVarint(i.lastBlockNumber); err != nil {
		return nil, err
	}
	var chainEmptyMarker uint64
	if i.isChainEmpty {
		chainEmptyMarker = 1
	}
	if err = buffer.EncodeVarint(chainEmptyMarker); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (i *checkpointInfo) unmarshal(b []byte) error {
	buffer := proto.NewBuffer(b)
	var val uint64
	var chainEmptyMarker uint64
	var err error

	if val, err = buffer.DecodeVarint(); err != nil {
		return err
	}
	i.latestFileChunkSuffixNum = int(val)

	if val, err = buffer.DecodeVarint(); err != nil {
		return err
	}
	i.latestFileChunksize = int(val)

	if val, err = buffer.DecodeVarint(); err != nil {
		return err
	}
	i.lastBlockNumber = val
	if chainEmptyMarker, err = buffer.DecodeVarint(); err != nil {
		return err
	}
	i.isChainEmpty = chainEmptyMarker == 1
	return nil
}

func (i *checkpointInfo) String() string {
	return fmt.Sprintf("latestFileChunkSuffixNum=[%d], latestFileChunksize=[%d], isChainEmpty=[%t], lastBlockNumber=[%d]",
		i.latestFileChunkSuffixNum, i.latestFileChunksize, i.isChainEmpty, i.lastBlockNumber)
}
