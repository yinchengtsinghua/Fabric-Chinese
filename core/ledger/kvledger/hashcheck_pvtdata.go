
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


package kvledger

import (
	"bytes"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/ledgerstorage"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/utils"
)

//ConstructValidAndInvalidPvtData计算有效的Pvt数据和哈希不匹配列表
//从接收到的旧块的pvt数据列表。
func ConstructValidAndInvalidPvtData(blocksPvtData []*ledger.BlockPvtData, blockStore *ledgerstorage.Store) (
	map[uint64][]*ledger.TxPvtData, []*ledger.PvtdataHashMismatch, error,
) {
//对于每个块，对于每个事务，检索txEnvelope到
//比较块中pvtrwset的哈希和接收的哈希
//TXPVTDATA。在不匹配的情况下，向hashmismatch列表中添加一个条目。
//在匹配项上，将pvtdata添加到validpvtdata列表中
	validPvtData := make(map[uint64][]*ledger.TxPvtData)
	var invalidPvtData []*ledger.PvtdataHashMismatch

	for _, blockPvtData := range blocksPvtData {
		validData, invalidData, err := findValidAndInvalidBlockPvtData(blockPvtData, blockStore)
		if err != nil {
			return nil, nil, err
		}
		if len(validData) > 0 {
			validPvtData[blockPvtData.BlockNum] = validData
		}
		invalidPvtData = append(invalidPvtData, invalidData...)
} //对于每个块的pvtdata
	return validPvtData, invalidPvtData, nil
}

func findValidAndInvalidBlockPvtData(blockPvtData *ledger.BlockPvtData, blockStore *ledgerstorage.Store) (
	[]*ledger.TxPvtData, []*ledger.PvtdataHashMismatch, error,
) {
	var validPvtData []*ledger.TxPvtData
	var invalidPvtData []*ledger.PvtdataHashMismatch
	for _, txPvtData := range blockPvtData.WriteSets {
//（1）从blockstore中检索txrwset
		logger.Debugf("Retrieving rwset of blockNum:[%d], txNum:[%d]", blockPvtData.BlockNum, txPvtData.SeqInBlock)
		txRWSet, err := retrieveRwsetForTx(blockPvtData.BlockNum, txPvtData.SeqInBlock, blockStore)
		if err != nil {
			return nil, nil, err
		}

//（2）根据Tx rwset中的pvtdata散列验证传递的pvtdata。
		logger.Debugf("Constructing valid and invalid pvtData using rwset of blockNum:[%d], txNum:[%d]",
			blockPvtData.BlockNum, txPvtData.SeqInBlock)
		validData, invalidData := findValidAndInvalidTxPvtData(txPvtData, txRWSet, blockPvtData.BlockNum)

//（3）将validdata附加到此块的validpvttatapvt列表中，然后
//无效数据到无效pvtdata列表
		if validData != nil {
			validPvtData = append(validPvtData, validData)
		}
		invalidPvtData = append(invalidPvtData, invalidData...)
} //对于每个Tx的pvtData
	return validPvtData, invalidPvtData, nil
}

func retrieveRwsetForTx(blkNum uint64, txNum uint64, blockStore *ledgerstorage.Store) (*rwsetutil.TxRwSet, error) {
//从块存储中检索txEnvelope，以便
//可以检索pvtdata进行比较
	txEnvelope, err := blockStore.RetrieveTxByBlockNumTranNum(blkNum, txNum)
	if err != nil {
		return nil, err
	}
//从txenvelope检索pvtrwset哈希
	responsePayload, err := utils.GetActionFromEnvelopeMsg(txEnvelope)
	if err != nil {
		return nil, err
	}
	txRWSet := &rwsetutil.TxRwSet{}
	if err := txRWSet.FromProtoBytes(responsePayload.Results); err != nil {
		return nil, err
	}
	return txRWSet, nil
}

func findValidAndInvalidTxPvtData(txPvtData *ledger.TxPvtData, txRWSet *rwsetutil.TxRwSet, blkNum uint64) (
	*ledger.TxPvtData, []*ledger.PvtdataHashMismatch,
) {
	var invalidPvtData []*ledger.PvtdataHashMismatch
	var toDeleteNsColl []*nsColl
//将pvtdata的哈希与rwset中的哈希进行比较
//查找有效和无效的pvt数据
	for _, nsRwset := range txPvtData.WriteSet.NsPvtRwset {
		txNum := txPvtData.SeqInBlock
		invalidData, invalidNsColl := findInvalidNsPvtData(nsRwset, txRWSet, blkNum, txNum)
		invalidPvtData = append(invalidPvtData, invalidData...)
		toDeleteNsColl = append(toDeleteNsColl, invalidNsColl...)
	}
	for _, nsColl := range toDeleteNsColl {
		txPvtData.WriteSet.Remove(nsColl.ns, nsColl.coll)
	}
	if len(txPvtData.WriteSet.NsPvtRwset) == 0 {
//表示所有命名空间
//无效的pvt数据
		return nil, invalidPvtData
	}
	return txPvtData, invalidPvtData
}

type nsColl struct {
	ns, coll string
}

func findInvalidNsPvtData(nsRwset *rwset.NsPvtReadWriteSet, txRWSet *rwsetutil.TxRwSet, blkNum, txNum uint64) (
	[]*ledger.PvtdataHashMismatch, []*nsColl,
) {
	var invalidPvtData []*ledger.PvtdataHashMismatch
	var invalidNsColl []*nsColl

	ns := nsRwset.Namespace
	for _, collPvtRwset := range nsRwset.CollectionPvtRwset {
		coll := collPvtRwset.CollectionName
		rwsetHash := txRWSet.GetPvtDataHash(ns, coll)
		if rwsetHash == nil {
			logger.Warningf("namespace: %s collection: %s was not accessed by txNum %d in BlkNum %d. "+
				"Unnecessary pvtdata has been passed", ns, coll, txNum, blkNum)
			invalidNsColl = append(invalidNsColl, &nsColl{ns, coll})
			continue
		}

		if !bytes.Equal(util.ComputeSHA256(collPvtRwset.Rwset), rwsetHash) {
			invalidPvtData = append(invalidPvtData, &ledger.PvtdataHashMismatch{
				BlockNum:     blkNum,
				TxNum:        txNum,
				Namespace:    ns,
				Collection:   coll,
				ExpectedHash: rwsetHash})
			invalidNsColl = append(invalidNsColl, &nsColl{ns, coll})
		}
	}
	return invalidPvtData, invalidNsColl
}
