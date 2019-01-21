
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


package valimpl

import (
	"bytes"
	"fmt"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/customtx"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/internal"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

//validateAndPreparpVTbatch为标记为有效的事务提取专用写入集
//通过内部公共数据验证器。最后，它根据
//corresponding hash present in the public rwset
func validateAndPreparePvtBatch(block *internal.Block, db privacyenabledstate.DB,
	pubAndHashUpdates *internal.PubAndHashUpdates, pvtdata map[uint64]*ledger.TxPvtData) (*privacyenabledstate.PvtUpdateBatch, error) {
	pvtUpdates := privacyenabledstate.NewPvtUpdateBatch()
	metadataUpdates := metadataUpdates{}
	for _, tx := range block.Txs {
		if tx.ValidationCode != peer.TxValidationCode_VALID {
			continue
		}
		if !tx.ContainsPvtWrites() {
			continue
		}
		txPvtdata := pvtdata[uint64(tx.IndexInBlock)]
		if txPvtdata == nil {
			continue
		}
		if requiresPvtdataValidation(txPvtdata) {
			if err := validatePvtdata(tx, txPvtdata); err != nil {
				return nil, err
			}
		}
		var pvtRWSet *rwsetutil.TxPvtRwSet
		var err error
		if pvtRWSet, err = rwsetutil.TxPvtRwSetFromProtoMsg(txPvtdata.WriteSet); err != nil {
			return nil, err
		}
		addPvtRWSetToPvtUpdateBatch(pvtRWSet, pvtUpdates, version.NewHeight(block.Num, uint64(tx.IndexInBlock)))
		addEntriesToMetadataUpdates(metadataUpdates, pvtRWSet)
	}
	if err := incrementPvtdataVersionIfNeeded(metadataUpdates, pvtUpdates, pubAndHashUpdates, db); err != nil {
		return nil, err
	}
	return pvtUpdates, nil
}

//RequiresPvtDataValidation返回是否应计算集合的哈希值
//对于存在于私有数据中的集合
//Todo现在总是返回true。添加检查此数据是否由
//在模拟过程中验证对等体本身，在这种情况下返回false
func requiresPvtdataValidation(tx *ledger.TxPvtData) bool {
	return true
}

//如果pvt数据中存在所有集合的散列，则validpvtdata返回true。
//与公共读写集中存在的相应哈希匹配
func validatePvtdata(tx *internal.Transaction, pvtdata *ledger.TxPvtData) error {
	if pvtdata.WriteSet == nil {
		return nil
	}

	for _, nsPvtdata := range pvtdata.WriteSet.NsPvtRwset {
		for _, collPvtdata := range nsPvtdata.CollectionPvtRwset {
			collPvtdataHash := util.ComputeHash(collPvtdata.Rwset)
			hashInPubdata := tx.RetrieveHash(nsPvtdata.Namespace, collPvtdata.CollectionName)
			if !bytes.Equal(collPvtdataHash, hashInPubdata) {
				return &validator.ErrPvtdataHashMissmatch{
					Msg: fmt.Sprintf(`Hash of pvt data for collection [%s:%s] does not match with the corresponding hash in the public data.
					public hash = [%#v], pvt data hash = [%#v]`, nsPvtdata.Namespace, collPvtdata.CollectionName, hashInPubdata, collPvtdataHash),
				}
			}
		}
	}
	return nil
}

//预处理ProtoBlock将块的Proto实例解析为“Block”结构。
//“重新调整的”block“结构仅包含属于背书人交易且未被标记为无效的交易。”
func preprocessProtoBlock(txMgr txmgr.TxMgr,
	validateKVFunc func(key string, value []byte) error,
	block *common.Block, doMVCCValidation bool,
) (*internal.Block, []*txmgr.TxStatInfo, error) {
	b := &internal.Block{Num: block.Header.Number}
	txsStatInfo := []*txmgr.TxStatInfo{}
//提交者验证程序已根据格式良好的事务检查设置验证标志
	txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	for txIndex, envBytes := range block.Data.Data {
		var env *common.Envelope
		var chdr *common.ChannelHeader
		var payload *common.Payload
		var err error
		txStatInfo := &txmgr.TxStatInfo{TxType: -1}
		txsStatInfo = append(txsStatInfo, txStatInfo)
		if env, err = utils.GetEnvelopeFromBlock(envBytes); err == nil {
			if payload, err = utils.GetPayload(env); err == nil {
				chdr, err = utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
			}
		}
		if txsFilter.IsInvalid(txIndex) {
//跳过无效事务
			logger.Warningf("Channel [%s]: Block [%d] Transaction index [%d] TxId [%s]"+
				" marked as invalid by committer. Reason code [%s]",
				chdr.GetChannelId(), block.Header.Number, txIndex, chdr.GetTxId(),
				txsFilter.Flag(txIndex).String())
			continue
		}
		if err != nil {
			return nil, nil, err
		}

		var txRWSet *rwsetutil.TxRwSet
		txType := common.HeaderType(chdr.Type)
		logger.Debugf("txType=%s", txType)
		txStatInfo.TxType = txType
		if txType == common.HeaderType_ENDORSER_TRANSACTION {
//从信封消息中提取操作
			respPayload, err := utils.GetActionFromEnvelope(envBytes)
			if err != nil {
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_NIL_TXACTION)
				continue
			}
			txStatInfo.ChaincodeID = respPayload.ChaincodeId
			txRWSet = &rwsetutil.TxRwSet{}
			if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_OTHER_REASON)
				continue
			}
		} else {
			rwsetProto, err := processNonEndorserTx(env, chdr.TxId, txType, txMgr, !doMVCCValidation)
			if _, ok := err.(*customtx.InvalidTxError); ok {
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_OTHER_REASON)
				continue
			}
			if err != nil {
				return nil, nil, err
			}
			if rwsetProto != nil {
				if txRWSet, err = rwsetutil.TxRwSetFromProtoMsg(rwsetProto); err != nil {
					return nil, nil, err
				}
			}
		}
		if txRWSet != nil {
			txStatInfo.NumCollections = txRWSet.NumCollections()
			if err := validateWriteset(txRWSet, validateKVFunc); err != nil {
				logger.Warningf("Channel [%s]: Block [%d] Transaction index [%d] TxId [%s]"+
					" marked as invalid. Reason code [%s]",
					chdr.GetChannelId(), block.Header.Number, txIndex, chdr.GetTxId(), peer.TxValidationCode_INVALID_WRITESET)
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_WRITESET)
				continue
			}
			b.Txs = append(b.Txs, &internal.Transaction{IndexInBlock: txIndex, ID: chdr.TxId, RWSet: txRWSet})
		}
	}
	return b, txsStatInfo, nil
}

func processNonEndorserTx(txEnv *common.Envelope, txid string, txType common.HeaderType, txmgr txmgr.TxMgr, synchingState bool) (*rwset.TxReadWriteSet, error) {
	logger.Debugf("Performing custom processing for transaction [txid=%s], [txType=%s]", txid, txType)
	processor := customtx.GetProcessor(txType)
	logger.Debugf("Processor for custom tx processing:%#v", processor)
	if processor == nil {
		return nil, nil
	}

	var err error
	var sim ledger.TxSimulator
	var simRes *ledger.TxSimulationResults
	if sim, err = txmgr.NewTxSimulator(txid); err != nil {
		return nil, err
	}
	defer sim.Done()
	if err = processor.GenerateSimulationResults(txEnv, sim, synchingState); err != nil {
		return nil, err
	}
	if simRes, err = sim.GetTxSimulationResults(); err != nil {
		return nil, err
	}
	return simRes.PubSimulationResults, nil
}

func validateWriteset(txRWSet *rwsetutil.TxRwSet, validateKVFunc func(key string, value []byte) error) error {
	for _, nsRwSet := range txRWSet.NsRwSets {
		pubWriteset := nsRwSet.KvRwSet
		if pubWriteset == nil {
			continue
		}
		for _, kvwrite := range pubWriteset.Writes {
			if err := validateKVFunc(kvwrite.Key, kvwrite.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

//后处理ProtoBlock通过验证过程的结果更新ProtoBlock的验证标志（在元数据中）
func postprocessProtoBlock(block *common.Block, validatedBlock *internal.Block) {
	txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	for _, tx := range validatedBlock.Txs {
		txsFilter.SetFlag(tx.IndexInBlock, tx.ValidationCode)
	}
	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
}

func addPvtRWSetToPvtUpdateBatch(pvtRWSet *rwsetutil.TxPvtRwSet, pvtUpdateBatch *privacyenabledstate.PvtUpdateBatch, ver *version.Height) {
	for _, ns := range pvtRWSet.NsPvtRwSet {
		for _, coll := range ns.CollPvtRwSets {
			for _, kvwrite := range coll.KvRwSet.Writes {
				if !kvwrite.IsDelete {
					pvtUpdateBatch.Put(ns.NameSpace, coll.CollectionName, kvwrite.Key, kvwrite.Value, ver)
				} else {
					pvtUpdateBatch.Delete(ns.NameSpace, coll.CollectionName, kvwrite.Key, ver)
				}
			}
		}
	}
}

//如果相应哈希键的版本
//已经升级了。仅元数据更新的事务类型可能导致哈希空间中现有值的版本更改。
//迭代所有元数据写入，并尝试获取这些密钥，并在私有写入中增加与哈希密钥版本相同的版本-如果是最新的
//键的值可用。否则，在这个场景中，我们最终得到的是私有状态下的最新值，但是版本
//因为错误地假设我们有过时的值，所以会导致模拟失败。
func incrementPvtdataVersionIfNeeded(
	metadataUpdates metadataUpdates,
	pvtUpdateBatch *privacyenabledstate.PvtUpdateBatch,
	pubAndHashUpdates *internal.PubAndHashUpdates,
	db privacyenabledstate.DB) error {

	for collKey := range metadataUpdates {
		ns, coll, key := collKey.ns, collKey.coll, collKey.key
		keyHash := util.ComputeStringHash(key)
		hashedVal := pubAndHashUpdates.HashUpdates.Get(ns, coll, string(keyHash))
		if hashedVal == nil {
//这个密钥最终不会被这个块中的散列空间更新。
//元数据更新位于不存在的键上，或者该键被块中的后一个事务删除。
//忽略此密钥的元数据更新
			continue
		}
		latestVal, err := retrieveLatestVal(ns, coll, key, pvtUpdateBatch, db)
		if err != nil {
			return err
		}
if latestVal == nil || //在数据库或pvt更新中找不到最新值（由缺少数据的提交导致）
version.AreSame(latestVal.Version, hashedVal.Version) { //版本已与哈希空间中的版本相同-由于只发生元数据事务，因此没有版本增量
			continue
		}
//todo-可以避免计算哈希。在散列更新中，我们可以增加
//更新了哪个原始版本
		latestValHash := util.ComputeHash(latestVal.Value)
if bytes.Equal(latestValHash, hashedVal.Value) { //由于我们允许块提交缺少pvt数据，因此可用的私有值可能已过时。
//仅当pvt值与哈希空间中的相应哈希匹配时升级版本
			pvtUpdateBatch.Put(ns, coll, key, latestVal.Value, hashedVal.Version)
		}
	}
	return nil
}

type collKey struct {
	ns, coll, key string
}

type metadataUpdates map[collKey]bool

func addEntriesToMetadataUpdates(metadataUpdates metadataUpdates, pvtRWSet *rwsetutil.TxPvtRwSet) {
	for _, ns := range pvtRWSet.NsPvtRwSet {
		for _, coll := range ns.CollPvtRwSets {
			for _, metadataWrite := range coll.KvRwSet.MetadataWrites {
				ns, coll, key := ns.NameSpace, coll.CollectionName, metadataWrite.Key
				metadataUpdates[collKey{ns, coll, key}] = true
			}
		}
	}
}

func retrieveLatestVal(ns, coll, key string, pvtUpdateBatch *privacyenabledstate.PvtUpdateBatch,
	db privacyenabledstate.DB) (val *statedb.VersionedValue, err error) {
	val = pvtUpdateBatch.Get(ns, coll, key)
	if val == nil {
		val, err = db.GetPrivateData(ns, coll, key)
	}
	return
}
