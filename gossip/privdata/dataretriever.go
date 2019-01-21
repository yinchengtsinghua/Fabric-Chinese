
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


package privdata

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/gossip/util"
	gossip2 "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/pkg/errors"
)

//StorageDataRetriever定义从存储中检索私有日期的API
type StorageDataRetriever interface {
//collectionrwset检索give digest相关的私有数据，如果
//否则，available将返回nil、bool，如果从分类帐中提取数据，则返回true；如果从临时存储中提取数据，则返回false，并返回错误。
	CollectionRWSet(dig []*gossip2.PvtDataDigest, blockNum uint64) (Dig2PvtRWSetWithConfig, bool, error)
}

//去：生成mokery-dir。-名称数据存储-大小写下划线-输出模拟/
//go:generate mokery-dir../../core/transientstore/-name rwsetscanner-case underline-output mocks/
//go:generate mokery-dir../../core/ledger/-name confighistoryretriever-case underline-output mocks/

//数据存储定义需要获取私有数据的一组API
//来自带下划线的数据存储
type DataStore interface {
//gettxpvtrwsetbytxid返回迭代器，因为txid可能有多个private
//RWSets坚持来自不同的支持者（通过八卦）
	GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error)

//getpvdtatabynum从分类帐返回一部分私有数据
//对于给定的块并基于指示
//要检索的私有数据的集合和命名空间
	GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error)

//getconfigHistoryRetriever返回configHistoryRetriever
	GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error)

//获取最近的块序列号
	LedgerHeight() (uint64, error)
}

type dataRetriever struct {
	store DataStore
}

//NewDataRetriever为实现
//存储数据检索器接口
func NewDataRetriever(store DataStore) StorageDataRetriever {
	return &dataRetriever{store: store}
}

//collectionrwset检索give digest相关的私有数据，如果
//否则，available将返回nil、bool，如果从分类帐中提取数据，则返回true；如果从临时存储中提取数据，则返回false，并返回错误。
func (dr *dataRetriever) CollectionRWSet(digests []*gossip2.PvtDataDigest, blockNum uint64) (Dig2PvtRWSetWithConfig, bool, error) {
	height, err := dr.store.LedgerHeight()
	if err != nil {
//如果从分类帐中获取信息时出错，我们需要尝试从临时存储中读取信息。
		return nil, false, errors.Wrap(err, "wasn't able to read ledger height")
	}
	if height <= blockNum {
		logger.Debug("Current ledger height ", height, "is below requested block sequence number",
			blockNum, "retrieving private data from transient store")
	}

if height <= blockNum { //当当前分类帐高度等于或低于块序列号时进行检查。
		results := make(Dig2PvtRWSetWithConfig)
		for _, dig := range digests {
			filter := map[string]ledger.PvtCollFilter{
				dig.Namespace: map[string]bool{
					dig.Collection: true,
				},
			}
			pvtRWSet, err := dr.fromTransientStore(dig, filter)
			if err != nil {
				logger.Errorf("couldn't read from transient store private read-write set, "+
					"digest %+v, because of %s", dig, err)
				continue
			}
			results[common.DigKey{
				Namespace:  dig.Namespace,
				Collection: dig.Collection,
				TxId:       dig.TxId,
				BlockSeq:   dig.BlockSeq,
				SeqInBlock: dig.SeqInBlock,
			}] = pvtRWSet
		}

		return results, false, nil
	}
//由于分类帐高度高于块序列号，因此分类帐中可能提供私有数据。
	results, err := dr.fromLedger(digests, blockNum)
	return results, true, err
}

func (dr *dataRetriever) fromLedger(digests []*gossip2.PvtDataDigest, blockNum uint64) (Dig2PvtRWSetWithConfig, error) {
	filter := make(map[string]ledger.PvtCollFilter)
	for _, dig := range digests {
		if _, ok := filter[dig.Namespace]; !ok {
			filter[dig.Namespace] = make(ledger.PvtCollFilter)
		}
		filter[dig.Namespace][dig.Collection] = true
	}

	pvtData, err := dr.store.GetPvtDataByNum(blockNum, filter)
	if err != nil {
		return nil, errors.Errorf("wasn't able to obtain private data, block sequence number %d, due to %s", blockNum, err)
	}

	results := make(Dig2PvtRWSetWithConfig)
	for _, dig := range digests {
		dig := dig
		pvtRWSetWithConfig := &util.PrivateRWSetWithConfig{}
		for _, data := range pvtData {
			if data.WriteSet == nil {
				logger.Warning("Received nil write set for collection tx in block", data.SeqInBlock, "block number", blockNum)
				continue
			}

//私有数据不包含命名空间和集合的rwset，或者
//属于不同的交易
			if !data.Has(dig.Namespace, dig.Collection) || data.SeqInBlock != dig.SeqInBlock {
				continue
			}

			pvtRWSet := dr.extractPvtRWsets(data.WriteSet.NsPvtRwset, dig.Namespace, dig.Collection)
			pvtRWSetWithConfig.RWSet = append(pvtRWSetWithConfig.RWSet, pvtRWSet...)
		}

		confHistoryRetriever, err := dr.store.GetConfigHistoryRetriever()
		if err != nil {
			return nil, errors.Errorf("cannot obtain configuration history retriever, for collection <%s>"+
				" txID <%s> block sequence number <%d> due to <%s>", dig.Collection, dig.TxId, dig.BlockSeq, err)
		}

		configInfo, err := confHistoryRetriever.MostRecentCollectionConfigBelow(dig.BlockSeq, dig.Namespace)
		if err != nil {
			return nil, errors.Errorf("cannot find recent collection config update below block sequence = %d,"+
				" collection name = <%s> for chaincode <%s>", dig.BlockSeq, dig.Collection, dig.Namespace)
		}

		if configInfo == nil {
			return nil, errors.Errorf("no collection config update below block sequence = <%d>"+
				" collection name = <%s> for chaincode <%s> is available ", dig.BlockSeq, dig.Collection, dig.Namespace)
		}
		configs := extractCollectionConfig(configInfo.CollectionConfig, dig.Collection)
		if configs == nil {
			return nil, errors.Errorf("no collection config was found for collection <%s>"+
				" namespace <%s> txID <%s>", dig.Collection, dig.Namespace, dig.TxId)
		}
		pvtRWSetWithConfig.CollectionConfig = configs
		results[common.DigKey{
			Namespace:  dig.Namespace,
			Collection: dig.Collection,
			TxId:       dig.TxId,
			BlockSeq:   dig.BlockSeq,
			SeqInBlock: dig.SeqInBlock,
		}] = pvtRWSetWithConfig
	}

	return results, nil
}

func (dr *dataRetriever) fromTransientStore(dig *gossip2.PvtDataDigest, filter map[string]ledger.PvtCollFilter) (*util.PrivateRWSetWithConfig, error) {
	results := &util.PrivateRWSetWithConfig{}
	it, err := dr.store.GetTxPvtRWSetByTxid(dig.TxId, filter)
	if err != nil {
		return nil, errors.Errorf("was not able to retrieve private data from transient store, namespace <%s>"+
			", collection name %s, txID <%s>, due to <%s>", dig.Namespace, dig.Collection, dig.TxId, err)
	}
	defer it.Close()

	maxEndorsedAt := uint64(0)
	for {
		res, err := it.NextWithConfig()
		if err != nil {
			return nil, errors.Errorf("error getting next element out of private data iterator, namespace <%s>"+
				", collection name <%s>, txID <%s>, due to <%s>", dig.Namespace, dig.Collection, dig.TxId, err)
		}
		if res == nil {
			return results, nil
		}
		rws := res.PvtSimulationResultsWithConfig
		if rws == nil {
			logger.Debug("Skipping nil PvtSimulationResultsWithConfig received at block height", res.ReceivedAtBlockHeight)
			continue
		}
		txPvtRWSet := rws.PvtRwset
		if txPvtRWSet == nil {
			logger.Debug("Skipping empty PvtRwset of PvtSimulationResultsWithConfig received at block height", res.ReceivedAtBlockHeight)
			continue
		}

		colConfigs, found := rws.CollectionConfigs[dig.Namespace]
		if !found {
			logger.Error("No collection config was found for chaincode", dig.Namespace, "collection name",
				dig.Namespace, "txID", dig.TxId)
			continue
		}

		configs := extractCollectionConfig(colConfigs, dig.Collection)
		if configs == nil {
			logger.Error("No collection config was found for collection", dig.Collection,
				"namespace", dig.Namespace, "txID", dig.TxId)
			continue
		}

		pvtRWSet := dr.extractPvtRWsets(txPvtRWSet.NsPvtRwset, dig.Namespace, dig.Collection)
		if rws.EndorsedAt >= maxEndorsedAt {
			maxEndorsedAt = rws.EndorsedAt
			results.CollectionConfig = configs
		}
		results.RWSet = append(results.RWSet, pvtRWSet...)
	}
}

func (dr *dataRetriever) extractPvtRWsets(pvtRWSets []*rwset.NsPvtReadWriteSet, namespace string, collectionName string) []util.PrivateRWSet {
	pRWsets := []util.PrivateRWSet{}

//遍历所有命名空间
	for _, nsws := range pvtRWSets {
//在每个命名空间中-遍历所有集合
		if nsws.Namespace != namespace {
			logger.Debug("Received private data namespace ", nsws.Namespace, " instead of ", namespace, " skipping...")
			continue
		}
		for _, col := range nsws.CollectionPvtRwset {
//这不是我们要找的收藏品
			if col.CollectionName != collectionName {
				logger.Debug("Received private data collection ", col.CollectionName, " instead of ", collectionName, " skipping...")
				continue
			}
//将集合prwset添加到累计集
			pRWsets = append(pRWsets, col.Rwset)
		}
	}

	return pRWsets
}
