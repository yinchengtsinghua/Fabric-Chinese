
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
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	vsccErrors "github.com/hyperledger/fabric/common/errors"
	util2 "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/transientstore"
	privdatacommon "github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/peer"
	transientstore2 "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	pullRetrySleepInterval           = time.Second
	transientBlockRetentionConfigKey = "peer.gossip.pvtData.transientstoreMaxBlockRetention"
	transientBlockRetentionDefault   = 1000
)

var logger = util.GetLogger(util.PrivateDataLogger, "")

//go:generate mokery-dir../../core/common/privdata/-name collectionstore-case underline-output mocks/
//go:generate mokery-dir../../core/committer/-name committer-case underline-output mocks/

//TransientStore保留尚未提交到分类帐中的相应块的私有数据。
type TransientStore interface {
//PersisteWithConfig存储事务的私有写入集以及集合配置
//在基于txid和块高度的临时存储中，在
	PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore2.TxPvtReadWriteSetWithConfigInfo) error

//持久性将事务的私有写入集存储在临时存储中
	Persist(txid string, blockHeight uint64, privateSimulationResults *rwset.TxPvtReadWriteSet) error
//gettxpvtrwsetbytxid返回迭代器，因为txid可能有多个private
//来自不同代言人的写集（通过八卦）
	GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error)

//purgebytxids从
//瞬态存储
	PurgeByTxids(txids []string) error

//purgebyHeight删除块高度小于
//给定的最大阻塞数。换句话说，清除只保留私有写集。
//保留在MaxBlockNumtoretain或更高的块高度。尽管是私人的
//用PurgBytxIdx（）将存储在瞬态存储中的写集由协调器删除。
//块提交成功后，仍需要purgeByHeight（）来删除孤立的条目（如
//获得批准的事务不能由客户端提交以供提交）
	PurgeByHeight(maxBlockNumToRetain uint64) error
}

//协调器协调新的流程
//块到达和飞行中瞬态数据，负责
//完成给定块的瞬态数据缺失部分。
type Coordinator interface {
//StoreBlock提供带有下划线的私有数据的新块
//返回缺少的事务ID
	StoreBlock(block *common.Block, data util.PvtDataCollections) error

//storepvdtata用于将私有数据持久化到临时存储中
	StorePvtData(txid string, privData *transientstore2.TxPvtReadWriteSetWithConfigInfo, blckHeight uint64) error

//getpvtdata和blockbynum按编号获取block并返回所有相关的私有数据
//pvtDataCollections切片中私有数据的顺序并不意味着
//块中与这些私有数据相关的事务，以获得正确的位置
//需要读取txpvtdata.seqinblock字段
	GetPvtDataAndBlockByNum(seqNum uint64, peerAuth common.SignedData) (*common.Block, util.PvtDataCollections, error)

//获取最近的块序列号
	LedgerHeight() (uint64, error)

//关闭协调员，关闭协调员服务
	Close()
}

type dig2sources map[privdatacommon.DigKey][]*peer.Endorsement

func (d2s dig2sources) keys() []privdatacommon.DigKey {
	res := make([]privdatacommon.DigKey, 0, len(d2s))
	for dig := range d2s {
		res = append(res, dig)
	}
	return res
}

//缺少定义要获取的API的获取器接口
//私有数据元素
type Fetcher interface {
	fetch(dig2src dig2sources) (*privdatacommon.FetchedPvtDataContainer, error)
}

//支持将一组接口封装到
//按单个结构聚合所需功能
type Support struct {
	ChainID string
	privdata.CollectionStore
	txvalidator.Validator
	committer.Committer
	TransientStore
	Fetcher
}

type coordinator struct {
	selfSignedData common.SignedData
	Support
	transientBlockRetention uint64
}

//NewCoordinator创建Coordinator的新实例
func NewCoordinator(support Support, selfSignedData common.SignedData) Coordinator {
	transientBlockRetention := uint64(viper.GetInt(transientBlockRetentionConfigKey))
	if transientBlockRetention == 0 {
		logger.Warning("Configuration key", transientBlockRetentionConfigKey, "isn't set, defaulting to", transientBlockRetentionDefault)
		transientBlockRetention = transientBlockRetentionDefault
	}
	return &coordinator{Support: support, selfSignedData: selfSignedData, transientBlockRetention: transientBlockRetention}
}

//storepvdtdata用于将私有日期持久化到临时存储中
func (c *coordinator) StorePvtData(txID string, privData *transientstore2.TxPvtReadWriteSetWithConfigInfo, blkHeight uint64) error {
	return c.TransientStore.PersistWithConfig(txID, blkHeight, privData)
}

//存储块将包含私有数据的块存储到分类帐中
func (c *coordinator) StoreBlock(block *common.Block, privateDataSets util.PvtDataCollections) error {
	if block.Data == nil {
		return errors.New("Block data is empty")
	}
	if block.Header == nil {
		return errors.New("Block header is nil")
	}

	logger.Infof("[%s] Received block [%d] from buffer", c.ChainID, block.Header.Number)

	logger.Debugf("[%s] Validating block [%d]", c.ChainID, block.Header.Number)
	err := c.Validator.Validate(block)
	if err != nil {
		logger.Errorf("Validation failed: %+v", err)
		return err
	}

	blockAndPvtData := &ledger.BlockAndPvtData{
		Block:          block,
		PvtData:        make(ledger.TxPvtDataMap),
		MissingPvtData: make(ledger.TxMissingPvtDataMap),
	}

	ownedRWsets, err := computeOwnedRWsets(block, privateDataSets)
	if err != nil {
		logger.Warning("Failed computing owned RWSets", err)
		return err
	}

	privateInfo, err := c.listMissingPrivateData(block, ownedRWsets)
	if err != nil {
		logger.Warning(err)
		return err
	}

	retryThresh := viper.GetDuration("peer.gossip.pvtData.pullRetryThreshold")
var bFetchFromPeers bool //默认为假
	if len(privateInfo.missingKeys) == 0 {
		logger.Debugf("[%s] No missing collection private write sets to fetch from remote peers", c.ChainID)
	} else {
		bFetchFromPeers = true
		logger.Debugf("[%s] Could not find all collection private write sets in local peer transient store for block [%d].", c.ChainID, block.Header.Number)
		logger.Debugf("[%s] Fetching %d collection private write sets from remote peers for a maximum duration of %s", c.ChainID, len(privateInfo.missingKeys), retryThresh)
	}
	startPull := time.Now()
	limit := startPull.Add(retryThresh)
	for len(privateInfo.missingKeys) > 0 && time.Now().Before(limit) {
		c.fetchFromPeers(block.Header.Number, ownedRWsets, privateInfo)
//如果成功地获取了所有信息，就不需要睡觉了。
//重试
		if len(privateInfo.missingKeys) == 0 {
			break
		}
		time.Sleep(pullRetrySleepInterval)
	}
elapsedPull := int64(time.Since(startPull) / time.Millisecond) //MS持续时间

//仅记录实际尝试获取的结果
	if bFetchFromPeers {
		if len(privateInfo.missingKeys) == 0 {
			logger.Infof("[%s] Fetched all missing collection private write sets from remote peers for block [%d] (%dms)", c.ChainID, block.Header.Number, elapsedPull)
		} else {
			logger.Warningf("[%s] Could not fetch all missing collection private write sets from remote peers. Will commit block [%d] with missing private write sets:[%v]",
				c.ChainID, block.Header.Number, privateInfo.missingKeys)
		}
	}

//填充传递到分类帐的私有RWset
	for seqInBlock, nsRWS := range ownedRWsets.bySeqsInBlock() {
		rwsets := nsRWS.toRWSet()
		logger.Debugf("[%s] Added %d namespace private write sets for block [%d], tran [%d]", c.ChainID, len(rwsets.NsPvtRwset), block.Header.Number, seqInBlock)
		blockAndPvtData.PvtData[seqInBlock] = &ledger.TxPvtData{
			SeqInBlock: seqInBlock,
			WriteSet:   rwsets,
		}
	}

//填充要传递到分类帐的缺少的RWset
	for missingRWS := range privateInfo.missingKeys {
		blockAndPvtData.MissingPvtData.Add(missingRWS.seqInBlock, missingRWS.namespace, missingRWS.collection, true)
	}

//为要传递到分类帐的不合格集合填充缺少的rwset
	for _, missingRWS := range privateInfo.missingRWSButIneligible {
		blockAndPvtData.MissingPvtData.Add(missingRWS.seqInBlock, missingRWS.namespace, missingRWS.collection, false)
	}

//提交块和私有数据
	err = c.CommitWithPvtData(blockAndPvtData)
	if err != nil {
		return errors.Wrap(err, "commit failed")
	}

	if len(blockAndPvtData.PvtData) > 0 {
//最后，清除块中的所有事务-有效或无效。
		if err := c.PurgeByTxids(privateInfo.txns); err != nil {
			logger.Error("Purging transactions", privateInfo.txns, "failed:", err)
		}
	}

	seq := block.Header.Number
	if seq%c.transientBlockRetention == 0 && seq > c.transientBlockRetention {
		err := c.PurgeByHeight(seq - c.transientBlockRetention)
		if err != nil {
			logger.Error("Failed purging data from transient store at block", seq, ":", err)
		}
	}

	return nil
}

func (c *coordinator) fetchFromPeers(blockSeq uint64, ownedRWsets map[rwSetKey][]byte, privateInfo *privateDataInfo) {
	dig2src := make(map[privdatacommon.DigKey][]*peer.Endorsement)
	privateInfo.missingKeys.foreach(func(k rwSetKey) {
		logger.Debug("Fetching", k, "from peers")
		dig := privdatacommon.DigKey{
			TxId:       k.txID,
			SeqInBlock: k.seqInBlock,
			Collection: k.collection,
			Namespace:  k.namespace,
			BlockSeq:   blockSeq,
		}
		dig2src[dig] = privateInfo.sources[k]
	})
	fetchedData, err := c.fetch(dig2src)
	if err != nil {
		logger.Warning("Failed fetching private data for block", blockSeq, "from peers:", err)
		return
	}

//迭代从对等端获取的数据
	for _, element := range fetchedData.AvailableElements {
		dig := element.Digest
		for _, rws := range element.Payload {
			hash := hex.EncodeToString(util2.ComputeSHA256(rws))
			key := rwSetKey{
				txID:       dig.TxId,
				namespace:  dig.Namespace,
				collection: dig.Collection,
				seqInBlock: dig.SeqInBlock,
				hash:       hash,
			}
			if _, isMissing := privateInfo.missingKeys[key]; !isMissing {
				logger.Debug("Ignoring", key, "because it wasn't found in the block")
				continue
			}
			ownedRWsets[key] = rws
			delete(privateInfo.missingKeys, key)
//如果我们获取与块I关联的私有数据，那么最后一个持久化的块必须是I-1
//所以我们的分类帐高度是i，因为块从0开始。
			c.TransientStore.Persist(dig.TxId, blockSeq, key.toTxPvtReadWriteSet(rws))
			logger.Debug("Fetched", key)
		}
	}
//重复清除的数据
	for _, dig := range fetchedData.PurgedElements {
//从丢失的键中删除清除的键
		for missingPvtRWKey := range privateInfo.missingKeys {
			if missingPvtRWKey.namespace == dig.Namespace &&
				missingPvtRWKey.collection == dig.Collection &&
				missingPvtRWKey.txID == dig.TxId {
				delete(privateInfo.missingKeys, missingPvtRWKey)
				logger.Warningf("Missing key because was purged or will soon be purged, "+
					"continue block commit without [%+v] in private rwset", missingPvtRWKey)
			}
		}
	}
}

func (c *coordinator) fetchMissingFromTransientStore(missing rwSetKeysByTxIDs, ownedRWsets map[rwSetKey][]byte) {
//检查瞬态存储
	for txAndSeq, filter := range missing.FiltersByTxIDs() {
		c.fetchFromTransientStore(txAndSeq, filter, ownedRWsets)
	}
}

func (c *coordinator) fetchFromTransientStore(txAndSeq txAndSeqInBlock, filter ledger.PvtNsCollFilter, ownedRWsets map[rwSetKey][]byte) {
	iterator, err := c.TransientStore.GetTxPvtRWSetByTxid(txAndSeq.txID, filter)
	if err != nil {
		logger.Warning("Failed obtaining iterator from transient store:", err)
		return
	}
	defer iterator.Close()
	for {
		res, err := iterator.NextWithConfig()
		if err != nil {
			logger.Error("Failed iterating:", err)
			break
		}
		if res == nil {
//迭代结束
			break
		}
		if res.PvtSimulationResultsWithConfig == nil {
			logger.Warning("Resultset's PvtSimulationResultsWithConfig for", txAndSeq.txID, "is nil, skipping")
			continue
		}
		simRes := res.PvtSimulationResultsWithConfig
		if simRes.PvtRwset == nil {
			logger.Warning("The PvtRwset of PvtSimulationResultsWithConfig for", txAndSeq.txID, "is nil, skipping")
			continue
		}
		for _, ns := range simRes.PvtRwset.NsPvtRwset {
			for _, col := range ns.CollectionPvtRwset {
				key := rwSetKey{
					txID:       txAndSeq.txID,
					seqInBlock: txAndSeq.seqInBlock,
					collection: col.CollectionName,
					namespace:  ns.Namespace,
					hash:       hex.EncodeToString(util2.ComputeSHA256(col.Rwset)),
				}
//用临时存储中的rw集填充ownedrwset
				ownedRWsets[key] = col.Rwset
} //迭代所有集合
} //遍历所有命名空间
} //迭代txpvtrwset结果
}

//ComputeOwnedRwsets标识我们已经拥有的块私有数据
func computeOwnedRWsets(block *common.Block, blockPvtData util.PvtDataCollections) (rwsetByKeys, error) {
	lastBlockSeq := len(block.Data.Data) - 1

	ownedRWsets := make(map[rwSetKey][]byte)
	for _, txPvtData := range blockPvtData {
		if lastBlockSeq < int(txPvtData.SeqInBlock) {
			logger.Warningf("Claimed SeqInBlock %d but block has only %d transactions", txPvtData.SeqInBlock, len(block.Data.Data))
			continue
		}
		env, err := utils.GetEnvelopeFromBlock(block.Data.Data[txPvtData.SeqInBlock])
		if err != nil {
			return nil, err
		}
		payload, err := utils.GetPayload(env)
		if err != nil {
			return nil, err
		}

		chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			return nil, err
		}
		for _, ns := range txPvtData.WriteSet.NsPvtRwset {
			for _, col := range ns.CollectionPvtRwset {
				computedHash := hex.EncodeToString(util2.ComputeSHA256(col.Rwset))
				ownedRWsets[rwSetKey{
					txID:       chdr.TxId,
					seqInBlock: txPvtData.SeqInBlock,
					collection: col.CollectionName,
					namespace:  ns.Namespace,
					hash:       computedHash,
				}] = col.Rwset
} //迭代命名空间中的集合
} //循环访问wset中的命名空间
} //迭代块中的事务
	return ownedRWsets, nil
}

type readWriteSets []readWriteSet

func (s readWriteSets) toRWSet() *rwset.TxPvtReadWriteSet {
	namespaces := make(map[string]*rwset.NsPvtReadWriteSet)
	dataModel := rwset.TxReadWriteSet_KV
	for _, rws := range s {
		if _, exists := namespaces[rws.namespace]; !exists {
			namespaces[rws.namespace] = &rwset.NsPvtReadWriteSet{
				Namespace: rws.namespace,
			}
		}
		col := &rwset.CollectionPvtReadWriteSet{
			CollectionName: rws.collection,
			Rwset:          rws.rws,
		}
		namespaces[rws.namespace].CollectionPvtRwset = append(namespaces[rws.namespace].CollectionPvtRwset, col)
	}

	var namespaceSlice []*rwset.NsPvtReadWriteSet
	for _, nsRWset := range namespaces {
		namespaceSlice = append(namespaceSlice, nsRWset)
	}

	return &rwset.TxPvtReadWriteSet{
		DataModel:  dataModel,
		NsPvtRwset: namespaceSlice,
	}
}

type readWriteSet struct {
	rwSetKey
	rws []byte
}

type rwsetByKeys map[rwSetKey][]byte

func (s rwsetByKeys) bySeqsInBlock() map[uint64]readWriteSets {
	res := make(map[uint64]readWriteSets)
	for k, rws := range s {
		res[k.seqInBlock] = append(res[k.seqInBlock], readWriteSet{
			rws:      rws,
			rwSetKey: k,
		})
	}
	return res
}

type rwsetKeys map[rwSetKey]struct{}

//string返回rwsetkeys的字符串表示形式
func (s rwsetKeys) String() string {
	var buffer bytes.Buffer
	for k := range s {
		buffer.WriteString(fmt.Sprintf("%s\n", k.String()))
	}
	return buffer.String()
}

//foreach调用每个键中的给定函数
func (s rwsetKeys) foreach(f func(key rwSetKey)) {
	for k := range s {
		f(k)
	}
}

//排除删除给定谓词接受的所有键。
func (s rwsetKeys) exclude(exists func(key rwSetKey) bool) {
	for k := range s {
		if exists(k) {
			delete(s, k)
		}
	}
}

type txAndSeqInBlock struct {
	txID       string
	seqInBlock uint64
}

type rwSetKeysByTxIDs map[txAndSeqInBlock][]rwSetKey

func (s rwSetKeysByTxIDs) flatten() rwsetKeys {
	m := make(map[rwSetKey]struct{})
	for _, keys := range s {
		for _, k := range keys {
			m[k] = struct{}{}
		}
	}
	return m
}

func (s rwSetKeysByTxIDs) FiltersByTxIDs() map[txAndSeqInBlock]ledger.PvtNsCollFilter {
	filters := make(map[txAndSeqInBlock]ledger.PvtNsCollFilter)
	for txAndSeq, rwsKeys := range s {
		filter := ledger.NewPvtNsCollFilter()
		for _, rwskey := range rwsKeys {
			filter.Add(rwskey.namespace, rwskey.collection)
		}
		filters[txAndSeq] = filter
	}

	return filters
}

type rwSetKey struct {
	txID       string
	seqInBlock uint64
	namespace  string
	collection string
	hash       string
}

//string返回rwsetkey的字符串表示形式
func (k *rwSetKey) String() string {
	return fmt.Sprintf("txID: %s, seq: %d, namespace: %s, collection: %s, hash: %s", k.txID, k.seqInBlock, k.namespace, k.collection, k.hash)
}

func (k *rwSetKey) toTxPvtReadWriteSet(rws []byte) *rwset.TxPvtReadWriteSet {
	return &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: k.namespace,
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: k.collection,
						Rwset:          rws,
					},
				},
			},
		},
	}
}

type txns []string
type blockData [][]byte
type blockConsumer func(seqInBlock uint64, chdr *common.ChannelHeader, txRWSet *rwsetutil.TxRwSet, endorsers []*peer.Endorsement) error

func (data blockData) forEachTxn(txsFilter txValidationFlags, consumer blockConsumer) (txns, error) {
	var txList []string
	for seqInBlock, envBytes := range data {
		env, err := utils.GetEnvelopeFromBlock(envBytes)
		if err != nil {
			logger.Warning("Invalid envelope:", err)
			continue
		}

		payload, err := utils.GetPayload(env)
		if err != nil {
			logger.Warning("Invalid payload:", err)
			continue
		}

		chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			logger.Warning("Invalid channel header:", err)
			continue
		}

		if chdr.Type != int32(common.HeaderType_ENDORSER_TRANSACTION) {
			continue
		}

		txList = append(txList, chdr.TxId)

		if txsFilter[seqInBlock] != uint8(peer.TxValidationCode_VALID) {
			logger.Debug("Skipping Tx", seqInBlock, "because it's invalid. Status is", txsFilter[seqInBlock])
			continue
		}

		respPayload, err := utils.GetActionFromEnvelope(envBytes)
		if err != nil {
			logger.Warning("Failed obtaining action from envelope", err)
			continue
		}

		tx, err := utils.GetTransaction(payload.Data)
		if err != nil {
			logger.Warning("Invalid transaction in payload data for tx ", chdr.TxId, ":", err)
			continue
		}

		ccActionPayload, err := utils.GetChaincodeActionPayload(tx.Actions[0].Payload)
		if err != nil {
			logger.Warning("Invalid chaincode action in payload for tx", chdr.TxId, ":", err)
			continue
		}

		if ccActionPayload.Action == nil {
			logger.Warning("Action in ChaincodeActionPayload for", chdr.TxId, "is nil")
			continue
		}

		txRWSet := &rwsetutil.TxRwSet{}
		if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
			logger.Warning("Failed obtaining TxRwSet from ChaincodeAction's results", err)
			continue
		}
		err = consumer(uint64(seqInBlock), chdr, txRWSet, ccActionPayload.Action.Endorsements)
		if err != nil {
			return txList, err
		}
	}
	return txList, nil
}

func endorsersFromOrgs(ns string, col string, endorsers []*peer.Endorsement, orgs []string) []*peer.Endorsement {
	var res []*peer.Endorsement
	for _, e := range endorsers {
		sID := &msp.SerializedIdentity{}
		err := proto.Unmarshal(e.Endorser, sID)
		if err != nil {
			logger.Warning("Failed unmarshalling endorser:", err)
			continue
		}
		if !util.Contains(sID.Mspid, orgs) {
			logger.Debug(sID.Mspid, "isn't among the collection's orgs:", orgs, "for namespace", ns, ",collection", col)
			continue
		}
		res = append(res, e)
	}
	return res
}

type privateDataInfo struct {
	sources                 map[rwSetKey][]*peer.Endorsement
	missingKeysByTxIDs      rwSetKeysByTxIDs
	missingKeys             rwsetKeys
	txns                    txns
	missingRWSButIneligible []rwSetKey
}

//listmissingprivatedata标识缺少的私有写入集，并尝试从本地临时存储中检索它们
func (c *coordinator) listMissingPrivateData(block *common.Block, ownedRWsets map[rwSetKey][]byte) (*privateDataInfo, error) {
	if block.Metadata == nil || len(block.Metadata.Metadata) <= int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		return nil, errors.New("Block.Metadata is nil or Block.Metadata lacks a Tx filter bitmap")
	}
	txsFilter := txValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	if len(txsFilter) != len(block.Data.Data) {
		return nil, errors.Errorf("Block data size(%d) is different from Tx filter size(%d)", len(block.Data.Data), len(txsFilter))
	}

	sources := make(map[rwSetKey][]*peer.Endorsement)
	privateRWsetsInBlock := make(map[rwSetKey]struct{})
	missing := make(rwSetKeysByTxIDs)
	data := blockData(block.Data.Data)
	bi := &transactionInspector{
		sources:              sources,
		missingKeys:          missing,
		ownedRWsets:          ownedRWsets,
		privateRWsetsInBlock: privateRWsetsInBlock,
		coordinator:          c,
	}
	txList, err := data.forEachTxn(txsFilter, bi.inspectTransaction)
	if err != nil {
		return nil, err
	}

	privateInfo := &privateDataInfo{
		sources:                 sources,
		missingKeysByTxIDs:      missing,
		txns:                    txList,
		missingRWSButIneligible: bi.missingRWSButIneligible,
	}

	logger.Debug("Retrieving private write sets for", len(privateInfo.missingKeysByTxIDs), "transactions from transient store")

//放入丢失并在临时存储中找到的ownedrwsets rw sets
	c.fetchMissingFromTransientStore(privateInfo.missingKeysByTxIDs, ownedRWsets)

//最后，循环访问ownedrwset，如果密钥不存在于
//privaterwsetsinblock-将其从ownedrwset中删除
	for k := range ownedRWsets {
		if _, exists := privateRWsetsInBlock[k]; !exists {
			logger.Warning("Removed", k.namespace, k.collection, "hash", k.hash, "from the data passed to the ledger")
			delete(ownedRWsets, k)
		}
	}

	privateInfo.missingKeys = privateInfo.missingKeysByTxIDs.flatten()
//删除我们已经拥有的所有密钥
	privateInfo.missingKeys.exclude(func(key rwSetKey) bool {
		_, exists := ownedRWsets[key]
		return exists
	})

	return privateInfo, nil
}

type transactionInspector struct {
	*coordinator
	privateRWsetsInBlock    map[rwSetKey]struct{}
	missingKeys             rwSetKeysByTxIDs
	sources                 map[rwSetKey][]*peer.Endorsement
	ownedRWsets             map[rwSetKey][]byte
	missingRWSButIneligible []rwSetKey
}

func (bi *transactionInspector) inspectTransaction(seqInBlock uint64, chdr *common.ChannelHeader, txRWSet *rwsetutil.TxRwSet, endorsers []*peer.Endorsement) error {
	for _, ns := range txRWSet.NsRwSets {
		for _, hashedCollection := range ns.CollHashedRwSets {
			if !containsWrites(chdr.TxId, ns.NameSpace, hashedCollection) {
				continue
			}

//如果由于数据库不可用而发生错误，我们应该停止提交
//关联链的块。对于有效集合，策略不能为零。
//对于从未定义的集合，策略将为零，我们可以安全地
//继续下一个收藏集。
			policy, err := bi.accessPolicyForCollection(chdr, ns.NameSpace, hashedCollection.CollectionName)
			if err != nil {
				return &vsccErrors.VSCCExecutionFailureError{Err: err}
			}

			if policy == nil {
				logger.Errorf("Failed to retrieve collection config for channel [%s], chaincode [%s], collection name [%s] for txID [%s]. Skipping.",
					chdr.ChannelId, ns.NameSpace, hashedCollection.CollectionName, chdr.TxId)
				continue
			}

			key := rwSetKey{
				txID:       chdr.TxId,
				seqInBlock: seqInBlock,
				hash:       hex.EncodeToString(hashedCollection.PvtRwSetHash),
				namespace:  ns.NameSpace,
				collection: hashedCollection.CollectionName,
			}

			if !bi.isEligible(policy, ns.NameSpace, hashedCollection.CollectionName) {
				logger.Debugf("Peer is not eligible for collection, channel [%s], chaincode [%s], "+
					"collection name [%s], txID [%s] the policy is [%#v]. Skipping.",
					chdr.ChannelId, ns.NameSpace, hashedCollection.CollectionName, chdr.TxId, policy)
				bi.missingRWSButIneligible = append(bi.missingRWSButIneligible, key)
				continue
			}

			bi.privateRWsetsInBlock[key] = struct{}{}
			if _, exists := bi.ownedRWsets[key]; !exists {
				txAndSeq := txAndSeqInBlock{
					txID:       chdr.TxId,
					seqInBlock: seqInBlock,
				}
				bi.missingKeys[txAndSeq] = append(bi.missingKeys[txAndSeq], key)
				bi.sources[key] = endorsersFromOrgs(ns.NameSpace, hashedCollection.CollectionName, endorsers, policy.MemberOrgs())
			}
} //对于所有哈希rw集
} //对于所有RW集
	return nil
}

//AccessPolicyForCollection检索给定命名空间、集合名称的CollectionAccessPolicy
//对应于给定的channelheader
func (c *coordinator) accessPolicyForCollection(chdr *common.ChannelHeader, namespace string, col string) (privdata.CollectionAccessPolicy, error) {
	cp := common.CollectionCriteria{
		Channel:    chdr.ChannelId,
		Namespace:  namespace,
		Collection: col,
		TxId:       chdr.TxId,
	}
	sp, err := c.CollectionStore.RetrieveCollectionAccessPolicy(cp)

	if _, isNoSuchCollectionError := err.(privdata.NoSuchCollectionError); err != nil && !isNoSuchCollectionError {
		logger.Warning("Failed obtaining policy for", cp, "due to database unavailability:", err)
		return nil, err
	}
	return sp, nil
}

//ISeligible检查此对等方是否符合给定CollectionAccessPolicy的条件
func (c *coordinator) isEligible(ap privdata.CollectionAccessPolicy, namespace string, col string) bool {
	filt := ap.AccessFilter()
	eligible := filt(c.selfSignedData)
	if !eligible {
		logger.Debug("Skipping namespace", namespace, "collection", col, "because we're not eligible for the private data")
	}
	return eligible
}

type seqAndDataModel struct {
	seq       uint64
	dataModel rwset.TxReadWriteSet_DataModel
}

//从SeqandDataModel映射到：
//MAAP从命名空间到[]*rwset.collectionpvtreadwriteset
type aggregatedCollections map[seqAndDataModel]map[string][]*rwset.CollectionPvtReadWriteSet

func (ac aggregatedCollections) addCollection(seqInBlock uint64, dm rwset.TxReadWriteSet_DataModel, namespace string, col *rwset.CollectionPvtReadWriteSet) {
	seq := seqAndDataModel{
		dataModel: dm,
		seq:       seqInBlock,
	}
	if _, exists := ac[seq]; !exists {
		ac[seq] = make(map[string][]*rwset.CollectionPvtReadWriteSet)
	}
	ac[seq][namespace] = append(ac[seq][namespace], col)
}

func (ac aggregatedCollections) asPrivateData() []*ledger.TxPvtData {
	var data []*ledger.TxPvtData
	for seq, ns := range ac {
		txPrivateData := &ledger.TxPvtData{
			SeqInBlock: seq.seq,
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: seq.dataModel,
			},
		}
		for namespaceName, cols := range ns {
			txPrivateData.WriteSet.NsPvtRwset = append(txPrivateData.WriteSet.NsPvtRwset, &rwset.NsPvtReadWriteSet{
				Namespace:          namespaceName,
				CollectionPvtRwset: cols,
			})
		}
		data = append(data, txPrivateData)
	}
	return data
}

//getpvtdata和blockbynum按编号获取block并返回所有相关的私有数据
//pvtDataCollections切片中私有数据的顺序并不意味着
//块中与这些私有数据相关的事务，以获得正确的位置
//需要读取txpvtdata.seqinblock字段
func (c *coordinator) GetPvtDataAndBlockByNum(seqNum uint64, peerAuthInfo common.SignedData) (*common.Block, util.PvtDataCollections, error) {
	blockAndPvtData, err := c.Committer.GetPvtDataAndBlockByNum(seqNum)
	if err != nil {
		return nil, nil, err
	}

	seqs2Namespaces := aggregatedCollections(make(map[seqAndDataModel]map[string][]*rwset.CollectionPvtReadWriteSet))
	data := blockData(blockAndPvtData.Block.Data.Data)
	data.forEachTxn(make(txValidationFlags, len(data)), func(seqInBlock uint64, chdr *common.ChannelHeader, txRWSet *rwsetutil.TxRwSet, _ []*peer.Endorsement) error {
		item, exists := blockAndPvtData.PvtData[seqInBlock]
		if !exists {
			return nil
		}

		for _, ns := range item.WriteSet.NsPvtRwset {
			for _, col := range ns.CollectionPvtRwset {
				cc := common.CollectionCriteria{
					Channel:    chdr.ChannelId,
					TxId:       chdr.TxId,
					Namespace:  ns.Namespace,
					Collection: col.CollectionName,
				}
				sp, err := c.CollectionStore.RetrieveCollectionAccessPolicy(cc)
				if err != nil {
					logger.Warning("Failed obtaining policy for", cc, ":", err)
					continue
				}
				isAuthorized := sp.AccessFilter()
				if isAuthorized == nil {
					logger.Warning("Failed obtaining filter for", cc)
					continue
				}
				if !isAuthorized(peerAuthInfo) {
					logger.Debug("Skipping", cc, "because peer isn't authorized")
					continue
				}
				seqs2Namespaces.addCollection(seqInBlock, item.WriteSet.DataModel, ns.Namespace, col)
			}
		}
		return nil
	})

	return blockAndPvtData.Block, seqs2Namespaces.asPrivateData(), nil
}

//containsWrites检查给定的collHashedRwset是否包含写入
func containsWrites(txID string, namespace string, colHashedRWSet *rwsetutil.CollHashedRwSet) bool {
	if colHashedRWSet.HashedRwSet == nil {
		logger.Warningf("HashedRWSet of tx %s, namespace %s, collection %s is nil", txID, namespace, colHashedRWSet.CollectionName)
		return false
	}
	if len(colHashedRWSet.HashedRwSet.HashedWrites) == 0 && len(colHashedRWSet.HashedRwSet.MetadataWrites) == 0 {
		logger.Debugf("HashedRWSet of tx %s, namespace %s, collection %s doesn't contain writes", txID, namespace, colHashedRWSet.CollectionName)
		return false
	}
	return true
}
