
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


package historyleveldb

import (
	"bytes"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/util"
	"github.com/hyperledger/fabric/core/ledger/kvledger/history/historydb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

//LevelHistoryDBQueryExecutor是针对LevelDB历史数据库的查询执行器
type LevelHistoryDBQueryExecutor struct {
	historyDB  *historyDB
	blockStore blkstorage.BlockStore
}

//getHistoryForkey在接口“ledger.historyQueryExecutor”中实现方法
func (q *LevelHistoryDBQueryExecutor) GetHistoryForKey(namespace string, key string) (commonledger.ResultsIterator, error) {

	if ledgerconfig.IsHistoryDBEnabled() == false {
		return nil, errors.New("history database not enabled")
	}

	var compositeStartKey []byte
	var compositeEndKey []byte
	compositeStartKey = historydb.ConstructPartialCompositeHistoryKey(namespace, key, false)
	compositeEndKey = historydb.ConstructPartialCompositeHistoryKey(namespace, key, true)

//范围扫描以查找以命名空间~key开头的任何历史记录
	dbItr := q.historyDB.db.GetIterator(compositeStartKey, compositeEndKey)
	return newHistoryScanner(compositeStartKey, namespace, key, dbItr, q.blockStore), nil
}

//HistoryScanner实现resultsiterator，用于迭代历史结果
type historyScanner struct {
compositePartialKey []byte //CopyTeDealAlgKy包括命名空间~KEY
	namespace           string
	key                 string
	dbItr               iterator.Iterator
	blockStore          blkstorage.BlockStore
}

func newHistoryScanner(compositePartialKey []byte, namespace string, key string,
	dbItr iterator.Iterator, blockStore blkstorage.BlockStore) *historyScanner {
	return &historyScanner{compositePartialKey, namespace, key, dbItr, blockStore}
}

func (scanner *historyScanner) Next() (commonledger.QueryResult, error) {
	for {
		if !scanner.dbItr.Next() {
			return nil, nil
		}
historyKey := scanner.dbItr.Key() //历史键的格式为namespace~key~blocknum~trannum。

//splitcompositekey（namespace~key~blocknum~trannum，namespace~key~）将在第二个位置返回blocknum~trannum
		_, blockNumTranNumBytes := historydb.SplitCompositeHistoryKey(historyKey, scanner.compositePartialKey)

//检查blockNumTranNumBytes是否不包含nil字节（fab-11244）-除了最后一个字节。
//如果它包含一个nil字节，表示它是另一个键，而不是
//正在扫描历史记录。但是，即使对于有效密钥，最后一个字节也可以为零（表示事务编号为零）。
//这是因为，如果“blockNumTranNumBytes”确实是所需密钥的后缀，则只有可能包含nil字节
//是BlockNumTranNumBytes中事务号为零时的最后一个字节）。
//另一方面，如果“blockNumTranNumBytes”不是所需密钥的后缀，那么它必须是前缀。
//对于其他某个键（所需键除外），在这种情况下，必须至少有一个零字节（最后一个字节除外）。
//对于复合键中的“last”compositekeysep
//以名称空间ns中的两个键“key”和“key\x00”为例。这些键的条目将是
//分别为“ns-\x00 key-\x00 blknumTranNumBytes”和“ns-\x00 key-\x00-\x00 blknumTranNumBytes”类型。
//在上面的例子中，“-”只是为了可读性。此外，扫描范围时
//_ns-\x00 key-\x00-ns-\x00 key xff用于获取<ns，key>的历史记录，另一个key的条目
//属于范围，需要忽略
		if bytes.Contains(blockNumTranNumBytes[:len(blockNumTranNumBytes)-1], historydb.CompositeKeySep) {
			logger.Debugf("Some other key [%#v] found in the range while scanning history for key [%#v]. Skipping...",
				historyKey, scanner.key)
			continue
		}
		blockNum, bytesConsumed := util.DecodeOrderPreservingVarUint64(blockNumTranNumBytes[0:])
		tranNum, _ := util.DecodeOrderPreservingVarUint64(blockNumTranNumBytes[bytesConsumed:])
		logger.Debugf("Found history record for namespace:%s key:%s at blockNumTranNum %v:%v\n",
			scanner.namespace, scanner.key, blockNum, tranNum)

//从与此历史记录关联的块存储中获取事务
		tranEnvelope, err := scanner.blockStore.RetrieveTxByBlockNumTranNum(blockNum, tranNum)
		if err != nil {
			return nil, err
		}

//获取与此事务关联的TxID、键写入值、时间戳和删除指示器
		queryResult, err := getKeyModificationFromTran(tranEnvelope, scanner.namespace, scanner.key)
		if err != nil {
			return nil, err
		}
		logger.Debugf("Found historic key value for namespace:%s key:%s from transaction %s\n",
			scanner.namespace, scanner.key, queryResult.(*queryresult.KeyModification).TxId)
		return queryResult, nil
	}
}

func (scanner *historyScanner) Close() {
	scanner.dbItr.Release()
}

//GettXidAndKeyWriteValueFromTran检查事务是否写入给定的键
func getKeyModificationFromTran(tranEnvelope *common.Envelope, namespace string, key string) (commonledger.QueryResult, error) {
	logger.Debugf("Entering getKeyModificationFromTran()\n", namespace, key)

//从信封中提取操作
	payload, err := putils.GetPayload(tranEnvelope)
	if err != nil {
		return nil, err
	}

	tx, err := putils.GetTransaction(payload.Data)
	if err != nil {
		return nil, err
	}

	_, respPayload, err := putils.GetPayloads(tx.Actions[0])
	if err != nil {
		return nil, err
	}

	chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, err
	}

	txID := chdr.TxId
	timestamp := chdr.Timestamp

	txRWSet := &rwsetutil.TxRwSet{}

//从操作中获取结果，然后取消标记
//使用自定义反编排将其转换为txreadwriteset
	if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
		return nil, err
	}

//通过循环访问事务的读写集来查找命名空间和键
	for _, nsRWSet := range txRWSet.NsRwSets {
		if nsRWSet.NameSpace == namespace {
//得到了正确的名称空间，现在查找密钥写入
			for _, kvWrite := range nsRWSet.KvRwSet.Writes {
				if kvWrite.Key == key {
					return &queryresult.KeyModification{TxId: txID, Value: kvWrite.Value,
						Timestamp: timestamp, IsDelete: kvWrite.IsDelete}, nil
				}
} //结束键环
			return nil, errors.New("key not found in namespace's writeset")
} //结束如果
} //结束命名空间循环
	return nil, errors.New("namespace not found in transaction's ReadWriteSets")

}
