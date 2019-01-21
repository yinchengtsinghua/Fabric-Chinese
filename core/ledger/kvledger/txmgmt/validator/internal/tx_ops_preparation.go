
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


package internal

import (
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/storageutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
)

func prepareTxOps(rwset *rwsetutil.TxRwSet, txht *version.Height,
	precedingUpdates *PubAndHashUpdates, db privacyenabledstate.DB) (txOps, error) {
	txops := txOps{}
	txops.applyTxRwset(rwset)
//logger.debugf（“应用raw rwset=%v”，spew.sdump（txops））后的preparetxops（）txops）
	for ck, keyop := range txops {
//检查键、值和元数据的最终状态是否已存在于事务中，然后跳过
//否则，我们需要检索最新状态并合并到当前值或元数据更新中。
		if keyop.isDelete() || keyop.isUpsertAndMetadataUpdate() {
			continue
		}

//检查当前事务中是否只更新了值，然后从上次提交的状态合并元数据
		if keyop.isOnlyUpsert() {
			latestMetadata, err := retrieveLatestMetadata(ck.ns, ck.coll, ck.key, precedingUpdates, db)
			if err != nil {
				return nil, err
			}
			keyop.metadata = latestMetadata
			continue
		}

//当前事务中只更新元数据。合并上次提交状态的值
//如果密钥在最后一个状态下不存在，则将此密钥设置为当前事务中的noop
		latestVal, err := retrieveLatestState(ck.ns, ck.coll, ck.key, precedingUpdates, db)
		if err != nil {
			return nil, err
		}
		if latestVal != nil {
			keyop.value = latestVal.Value
		} else {
			delete(txops, ck)
		}
	}
//logger.debugf（“PrepareTxOps（）最终处理后的txOps=%v”，spew.sdump（txOps））。
	return txops, nil
}

//applyTxRwset记录了kv的上升/删除和上升/删除
//TxRwset中存在的关联元数据
func (txops txOps) applyTxRwset(rwset *rwsetutil.TxRwSet) error {
	for _, nsRWSet := range rwset.NsRwSets {
		ns := nsRWSet.NameSpace
		for _, kvWrite := range nsRWSet.KvRwSet.Writes {
			txops.applyKVWrite(ns, "", kvWrite)
		}
		for _, kvMetadataWrite := range nsRWSet.KvRwSet.MetadataWrites {
			txops.applyMetadata(ns, "", kvMetadataWrite)
		}

//应用收集级别kvwrite和kvmetadatawrite
		for _, collHashRWset := range nsRWSet.CollHashedRwSets {
			coll := collHashRWset.CollectionName
			for _, hashedWrite := range collHashRWset.HashedRwSet.HashedWrites {
				txops.applyKVWrite(ns, coll,
					&kvrwset.KVWrite{
						Key:      string(hashedWrite.KeyHash),
						Value:    hashedWrite.ValueHash,
						IsDelete: hashedWrite.IsDelete,
					},
				)
			}

			for _, metadataWrite := range collHashRWset.HashedRwSet.MetadataWrites {
				txops.applyMetadata(ns, coll,
					&kvrwset.KVMetadataWrite{
						Key:     string(metadataWrite.KeyHash),
						Entries: metadataWrite.Entries,
					},
				)
			}
		}
	}
	return nil
}

//applykvwrite记录kvwrite的更新/删除
func (txops txOps) applyKVWrite(ns, coll string, kvWrite *kvrwset.KVWrite) {
	if kvWrite.IsDelete {
		txops.delete(compositeKey{ns, coll, kvWrite.Key})
	} else {
		txops.upsert(compositeKey{ns, coll, kvWrite.Key}, kvWrite.Value)
	}
}

//applyMetadata records updatation/deletion of a metadataWrite
func (txops txOps) applyMetadata(ns, coll string, metadataWrite *kvrwset.KVMetadataWrite) error {
	if metadataWrite.Entries == nil {
		txops.metadataDelete(compositeKey{ns, coll, metadataWrite.Key})
	} else {
		metadataBytes, err := storageutil.SerializeMetadata(metadataWrite.Entries)
		if err != nil {
			return err
		}
		txops.metadataUpdate(compositeKey{ns, coll, metadataWrite.Key}, metadataBytes)
	}
	return nil
}

//RetrieveLatestState从预处理更新中返回键的值（如果该键由块中的前一个事务操作）。
//如果预处理更新中不存在该键，则此函数将从statedb中提取最新的值
//TODOFAB-11328，从州政府撤出（尤其是CouchDB）将支付大量的性能罚款，因此批量加载将是有帮助的。
//此外，所有被写入的密钥将被VSCC从statedb中提取，以进行背书策略检查（在密钥级别的情况下）
//endorsement) and hence, the bulkload should be combined
func retrieveLatestState(ns, coll, key string,
	precedingUpdates *PubAndHashUpdates, db privacyenabledstate.DB) (*statedb.VersionedValue, error) {
	var vv *statedb.VersionedValue
	var err error
	if coll == "" {
		vv := precedingUpdates.PubUpdates.Get(ns, key)
		if vv == nil {
			vv, err = db.GetState(ns, key)
		}
		return vv, err
	}

	vv = precedingUpdates.HashUpdates.Get(ns, coll, key)
	if vv == nil {
		vv, err = db.GetValueHash(ns, coll, []byte(key))
	}
	return vv, err
}

func retrieveLatestMetadata(ns, coll, key string,
	precedingUpdates *PubAndHashUpdates, db privacyenabledstate.DB) ([]byte, error) {
	if coll == "" {
		vv := precedingUpdates.PubUpdates.Get(ns, key)
		if vv != nil {
			return vv.Metadata, nil
		}
		return db.GetStateMetadata(ns, key)
	}
	vv := precedingUpdates.HashUpdates.Get(ns, coll, key)
	if vv != nil {
		return vv.Metadata, nil
	}
	return db.GetPrivateDataMetadataByHash(ns, coll, []byte(key))
}

type keyOpsFlag uint8

const (
upsertVal      keyOpsFlag = 1 //1＜0
metadataUpdate            = 2 //1＜1
metadataDelete            = 4 //1＜2
keyDelete                 = 8 //1＜3
)

type compositeKey struct {
	ns, coll, key string
}

type txOps map[compositeKey]*keyOps

type keyOps struct {
	flag     keyOpsFlag
	value    []byte
	metadata []byte
}

/////////////////txops函数

func (txops txOps) upsert(k compositeKey, val []byte) {
	keyops := txops.getOrCreateKeyEntry(k)
	keyops.flag += upsertVal
	keyops.value = val
}

func (txops txOps) delete(k compositeKey) {
	keyops := txops.getOrCreateKeyEntry(k)
	keyops.flag += keyDelete
}

func (txops txOps) metadataUpdate(k compositeKey, metadata []byte) {
	keyops := txops.getOrCreateKeyEntry(k)
	keyops.flag += metadataUpdate
	keyops.metadata = metadata
}

func (txops txOps) metadataDelete(k compositeKey) {
	keyops := txops.getOrCreateKeyEntry(k)
	keyops.flag += metadataDelete
}

func (txops txOps) getOrCreateKeyEntry(k compositeKey) *keyOps {
	keyops, ok := txops[k]
	if !ok {
		keyops = &keyOps{}
		txops[k] = keyops
	}
	return keyops
}

////////////////// keyOps functions

func (keyops keyOps) isDelete() bool {
	return keyops.flag&(keyDelete) == keyDelete
}

func (keyops keyOps) isUpsertAndMetadataUpdate() bool {
	if keyops.flag&upsertVal == upsertVal {
		return keyops.flag&metadataUpdate == metadataUpdate ||
			keyops.flag&metadataDelete == metadataDelete
	}
	return false
}

func (keyops keyOps) isOnlyUpsert() bool {
	return keyops.flag|upsertVal == upsertVal
}
