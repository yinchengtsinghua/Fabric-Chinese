
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


package statebasedval

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/internal"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("statebasedval")

//验证程序根据最新提交状态验证Tx
//与在同一个块中的前一个有效事务
type Validator struct {
	db privacyenabledstate.DB
}

//newvalidator构造Statevalidator
func NewValidator(db privacyenabledstate.DB) *Validator {
	return &Validator{db}
}

//preloadcommittedversionofrset加载每个键中所有键的提交版本
//事务的读取设置到缓存中。
func (v *Validator) preLoadCommittedVersionOfRSet(block *internal.Block) error {

//在给定块中所有事务的读取集中收集公钥和哈希键
	var pubKeys []*statedb.CompositeKey
	var hashedKeys []*privacyenabledstate.HashedCompositeKey

//pubkeysmap和hashedkeysmap用于避免
//PubKeys和HashedKeys。尽管地图可以单独用来收集钥匙
//在loadcommittedversionOfBubandHashedKeys（）中读取集并作为参数传递，
//数组用于更好的代码可读性。在消极方面，这种方法
//可能需要一些额外的内存。
	pubKeysMap := make(map[statedb.CompositeKey]interface{})
	hashedKeysMap := make(map[privacyenabledstate.HashedCompositeKey]interface{})

	for _, tx := range block.Txs {
		for _, nsRWSet := range tx.RWSet.NsRwSets {
			for _, kvRead := range nsRWSet.KvRwSet.Reads {
				compositeKey := statedb.CompositeKey{
					Namespace: nsRWSet.NameSpace,
					Key:       kvRead.Key,
				}
				if _, ok := pubKeysMap[compositeKey]; !ok {
					pubKeysMap[compositeKey] = nil
					pubKeys = append(pubKeys, &compositeKey)
				}

			}
			for _, colHashedRwSet := range nsRWSet.CollHashedRwSets {
				for _, kvHashedRead := range colHashedRwSet.HashedRwSet.HashedReads {
					hashedCompositeKey := privacyenabledstate.HashedCompositeKey{
						Namespace:      nsRWSet.NameSpace,
						CollectionName: colHashedRwSet.CollectionName,
						KeyHash:        string(kvHashedRead.KeyHash),
					}
					if _, ok := hashedKeysMap[hashedCompositeKey]; !ok {
						hashedKeysMap[hashedCompositeKey] = nil
						hashedKeys = append(hashedKeys, &hashedCompositeKey)
					}
				}
			}
		}
	}

//将所有密钥的提交版本加载到缓存中
	if len(pubKeys) > 0 || len(hashedKeys) > 0 {
		err := v.db.LoadCommittedVersionsOfPubAndHashedKeys(pubKeys, hashedKeys)
		if err != nil {
			return err
		}
	}

	return nil
}

//validateAndPreparatch在validator接口中实现方法
func (v *Validator) ValidateAndPrepareBatch(block *internal.Block, doMVCCValidation bool) (*internal.PubAndHashUpdates, error) {
//检查statedb是否实现了BulkOptimizable接口。现在，
//只有CouchDB实现了BulkOptimizable以减少剩余的数量
//从对等机到CouchDB实例的API调用。
	if v.db.IsBulkOptimizable() {
		err := v.preLoadCommittedVersionOfRSet(block)
		if err != nil {
			return nil, err
		}
	}

	updates := internal.NewPubAndHashUpdates()
	for _, tx := range block.Txs {
		var validationCode peer.TxValidationCode
		var err error
		if validationCode, err = v.validateEndorserTX(tx.RWSet, doMVCCValidation, updates); err != nil {
			return nil, err
		}

		tx.ValidationCode = validationCode
		if validationCode == peer.TxValidationCode_VALID {
			logger.Debugf("Block [%d] Transaction index [%d] TxId [%s] marked as valid by state validator", block.Num, tx.IndexInBlock, tx.ID)
			committingTxHeight := version.NewHeight(block.Num, uint64(tx.IndexInBlock))
			updates.ApplyWriteSet(tx.RWSet, committingTxHeight, v.db)
		} else {
			logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. Reason code [%s]",
				block.Num, tx.IndexInBlock, tx.ID, validationCode.String())
		}
	}
	return updates, nil
}

//validateEndorSertx验证背书人事务
func (v *Validator) validateEndorserTX(
	txRWSet *rwsetutil.TxRwSet,
	doMVCCValidation bool,
	updates *internal.PubAndHashUpdates) (peer.TxValidationCode, error) {

	var validationCode = peer.TxValidationCode_VALID
	var err error
//MVCCvalidation，可能使事务无效
	if doMVCCValidation {
		validationCode, err = v.validateTx(txRWSet, updates)
	}
	return validationCode, err
}

func (v *Validator) validateTx(txRWSet *rwsetutil.TxRwSet, updates *internal.PubAndHashUpdates) (peer.TxValidationCode, error) {
//仅为本地调试取消对以下内容的注释。不想在生产中打印日志中的数据
//logger.debugf（“validatetx-验证txrwset:%s”，spew.sdump（txrwset））。
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace
//验证公共读取
		if valid, err := v.validateReadSet(ns, nsRWSet.KvRwSet.Reads, updates.PubUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
//Validate range queries for phantom items
		if valid, err := v.validateRangeQueries(ns, nsRWSet.KvRwSet.RangeQueriesInfo, updates.PubUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_PHANTOM_READ_CONFLICT, nil
		}
//为私有读取验证哈希
		if valid, err := v.validateNsHashedReadSets(ns, nsRWSet.CollHashedRwSets, updates.HashUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
	}
	return peer.TxValidationCode_VALID, nil
}

//////////////////////////////////////////////////
/////公共读取集的验证
//////////////////////////////////////////////////
func (v *Validator) validateReadSet(ns string, kvReads []*kvrwset.KVRead, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	for _, kvRead := range kvReads {
		if valid, err := v.validateKVRead(ns, kvRead, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

//ValueKvRead在事务模拟期间执行密钥读取的MVCC检查。
//例如，它检查statedb中的键/版本组合是否已更新（由已提交的块更新）
//或在更新中（由当前块中的先前有效事务）
func (v *Validator) validateKVRead(ns string, kvRead *kvrwset.KVRead, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	if updates.Exists(ns, kvRead.Key) {
		return false, nil
	}
	committedVersion, err := v.db.GetVersion(ns, kvRead.Key)
	if err != nil {
		return false, err
	}

	logger.Debugf("Comparing versions for key [%s]: committed version=%#v and read version=%#v",
		kvRead.Key, committedVersion, rwsetutil.NewVersion(kvRead.Version))
	if !version.AreSame(committedVersion, rwsetutil.NewVersion(kvRead.Version)) {
		logger.Debugf("Version mismatch for key [%s:%s]. Committed version = [%#v], Version in readSet [%#v]",
			ns, kvRead.Key, committedVersion, kvRead.Version)
		return false, nil
	}
	return true, nil
}

//////////////////////////////////////////////////
/////范围查询的验证
//////////////////////////////////////////////////
func (v *Validator) validateRangeQueries(ns string, rangeQueriesInfo []*kvrwset.RangeQueryInfo, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	for _, rqi := range rangeQueriesInfo {
		if valid, err := v.validateRangeQuery(ns, rqi, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

//validaterangequery执行幻象读取检查，即
//检查在上执行时范围查询的结果是否仍然相同
//statedb (latest state as of last committed block) + updates (prepared by the writes of preceding valid transactions
//在当前块中，但在块验证结束时作为组提交的一部分提交）
func (v *Validator) validateRangeQuery(ns string, rangeQueryInfo *kvrwset.RangeQueryInfo, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	logger.Debugf("validateRangeQuery: ns=%s, rangeQueryInfo=%s", ns, rangeQueryInfo)

//如果在模拟过程中，调用方没有耗尽迭代器，那么
//rangequeryinfo.endkey不是范围查询中调用方给定的实际endkey
//但是它是调用方看到的最后一个键，因此combineditor应该在结果中包含endkey。
	includeEndKey := !rangeQueryInfo.ItrExhausted

	combinedItr, err := newCombinedIterator(v.db, updates.UpdateBatch,
		ns, rangeQueryInfo.StartKey, rangeQueryInfo.EndKey, includeEndKey)
	if err != nil {
		return false, err
	}
	defer combinedItr.Close()
	var validator rangeQueryValidator
	if rangeQueryInfo.GetReadsMerkleHashes() != nil {
		logger.Debug(`Hashing results are present in the range query info hence, initiating hashing based validation`)
		validator = &rangeQueryHashValidator{}
	} else {
		logger.Debug(`Hashing results are not present in the range query info hence, initiating raw KVReads based validation`)
		validator = &rangeQueryResultsValidator{}
	}
	validator.init(rangeQueryInfo, combinedItr)
	return validator.validate()
}

//////////////////////////////////////////////////
/////哈希读取集的验证
//////////////////////////////////////////////////
func (v *Validator) validateNsHashedReadSets(ns string, collHashedRWSets []*rwsetutil.CollHashedRwSet,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	for _, collHashedRWSet := range collHashedRWSets {
		if valid, err := v.validateCollHashedReadSet(ns, collHashedRWSet.CollectionName, collHashedRWSet.HashedRwSet.HashedReads, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

func (v *Validator) validateCollHashedReadSet(ns, coll string, kvReadHashes []*kvrwset.KVReadHash,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	for _, kvReadHash := range kvReadHashes {
		if valid, err := v.validateKVReadHash(ns, coll, kvReadHash, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

//validatekvreadhash对私有数据空间中存在的密钥的哈希执行MVCC检查
//例如，它检查statedb中的键/版本组合是否已更新（由已提交的块更新）
//或在更新中（由当前块中的先前有效事务）
func (v *Validator) validateKVReadHash(ns, coll string, kvReadHash *kvrwset.KVReadHash,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	if updates.Contains(ns, coll, kvReadHash.KeyHash) {
		return false, nil
	}
	committedVersion, err := v.db.GetKeyHashVersion(ns, coll, kvReadHash.KeyHash)
	if err != nil {
		return false, err
	}

	if !version.AreSame(committedVersion, rwsetutil.NewVersion(kvReadHash.Version)) {
		logger.Debugf("Version mismatch for key hash [%s:%s:%#v]. Committed version = [%s], Version in hashedReadSet [%s]",
			ns, coll, kvReadHash.KeyHash, committedVersion, kvReadHash.Version)
		return false, nil
	}
	return true, nil
}
