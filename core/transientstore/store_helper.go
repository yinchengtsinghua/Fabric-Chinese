
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


package transientstore

import (
	"bytes"
	"errors"
	"path/filepath"

	"github.com/hyperledger/fabric/common/ledger/util"
	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
)

var (
prwsetPrefix             = []byte("P")[0] //用于在临时存储中存储私有写入集的密钥前缀。
purgeIndexByHeightPrefix = []byte("H")[0] //用于在块高度使用Received在专用写入集上存储索引的键前缀。
purgeIndexByTxidPrefix   = []byte("T")[0] //用于在使用txid的私有写入集上存储索引的键前缀
	compositeKeySep          = byte(0x00)
)

//CREATECOMPITEKEYFPVPVRTWSET创建用于存储私有写入集的密钥
//在临时存储中。键的结构是<prwsetprefix>~txid~uuid~ blockheight。
func createCompositeKeyForPvtRWSet(txid string, uuid string, blockHeight uint64) []byte {
	var compositeKey []byte
	compositeKey = append(compositeKey, prwsetPrefix)
	compositeKey = append(compositeKey, compositeKeySep)
	compositeKey = append(compositeKey, createCompositeKeyWithoutPrefixForTxid(txid, uuid, blockHeight)...)

	return compositeKey
}

//CreateCompositeKeyOrburgeIndexByXid创建一个索引私有写入集的键，该键基于
//这样就可以实现基于TXID的净化。结构
//其中键是<purgeindexbythidprefix>~txid~uuid~ blockheight。
func createCompositeKeyForPurgeIndexByTxid(txid string, uuid string, blockHeight uint64) []byte {
	var compositeKey []byte
	compositeKey = append(compositeKey, purgeIndexByTxidPrefix)
	compositeKey = append(compositeKey, compositeKeySep)
	compositeKey = append(compositeKey, createCompositeKeyWithoutPrefixForTxid(txid, uuid, blockHeight)...)

	return compositeKey
}

//CreateCompositeKeyWithOutPrefixForXid创建结构txID~uuid~blockHeight的复合键。
func createCompositeKeyWithoutPrefixForTxid(txid string, uuid string, blockHeight uint64) []byte {
	var compositeKey []byte
	compositeKey = append(compositeKey, []byte(txid)...)
	compositeKey = append(compositeKey, compositeKeySep)
	compositeKey = append(compositeKey, []byte(uuid)...)
	compositeKey = append(compositeKey, compositeKeySep)
	compositeKey = append(compositeKey, util.EncodeOrderPreservingVarUint64(blockHeight)...)

	return compositeKey
}

//CreateCompositeKeyOrburgeIndexByHeight创建索引私有写入集的键
//在块高度接收，以便根据块高度进行净化。结构
//其中键是<purgeindexbyheightprefix>~blockheight~txid~uuid。
func createCompositeKeyForPurgeIndexByHeight(blockHeight uint64, txid string, uuid string) []byte {
	var compositeKey []byte
	compositeKey = append(compositeKey, purgeIndexByHeightPrefix)
	compositeKey = append(compositeKey, compositeKeySep)
	compositeKey = append(compositeKey, util.EncodeOrderPreservingVarUint64(blockHeight)...)
	compositeKey = append(compositeKey, compositeKeySep)
	compositeKey = append(compositeKey, []byte(txid)...)
	compositeKey = append(compositeKey, compositeKeySep)
	compositeKey = append(compositeKey, []byte(uuid)...)

	return compositeKey
}

//splitcompositekeyofpvtrwset拆分compositekey（<prwsetprefix>~txid~uuid~blockheight）
//到uuid和blockheight。
func splitCompositeKeyOfPvtRWSet(compositeKey []byte) (uuid string, blockHeight uint64) {
	return splitCompositeKeyWithoutPrefixForTxid(compositeKey[2:])
}

//splitcompositekeyofpurgeindexbythid拆分compositekey（<purgeindexbythidprefix>~txid~uuid~blockheight）
//到uuid和blockheight。
func splitCompositeKeyOfPurgeIndexByTxid(compositeKey []byte) (uuid string, blockHeight uint64) {
	return splitCompositeKeyWithoutPrefixForTxid(compositeKey[2:])
}

//splitcompositekeyofpurgeindexbyheight拆分compositekey（<purgeindexbyheightprefix>~blockheight~txid~uuid）
//into txid, uuid and blockHeight.
func splitCompositeKeyOfPurgeIndexByHeight(compositeKey []byte) (txid string, uuid string, blockHeight uint64) {
	var n int
	blockHeight, n = util.DecodeOrderPreservingVarUint64(compositeKey[2:])
	splits := bytes.Split(compositeKey[n+3:], []byte{compositeKeySep})
	txid = string(splits[0])
	uuid = string(splits[1])
	return
}

//splitcompositekeywithoutprefixforxid将组合键txid~uuid~ blockheight拆分为
//UUID和块高度
func splitCompositeKeyWithoutPrefixForTxid(compositeKey []byte) (uuid string, blockHeight uint64) {
//跳过txid，因为所有需要拆分复合键的函数都已具有它
	firstSepIndex := bytes.IndexByte(compositeKey, compositeKeySep)
	secondSepIndex := firstSepIndex + bytes.IndexByte(compositeKey[firstSepIndex+1:], compositeKeySep) + 1
	uuid = string(compositeKey[firstSepIndex+1 : secondSepIndex])
	blockHeight, _ = util.DecodeOrderPreservingVarUint64(compositeKey[secondSepIndex+1:])
	return
}

//createTxidRangeStartKey returns a startKey to do a range query on transient store using txid
func createTxidRangeStartKey(txid string) []byte {
	var startKey []byte
	startKey = append(startKey, prwsetPrefix)
	startKey = append(startKey, compositeKeySep)
	startKey = append(startKey, []byte(txid)...)
	startKey = append(startKey, compositeKeySep)
	return startKey
}

//createtxidrangeendkey返回一个endkey，以使用txid在临时存储上执行范围查询。
func createTxidRangeEndKey(txid string) []byte {
	var endKey []byte
	endKey = append(endKey, prwsetPrefix)
	endKey = append(endKey, compositeKeySep)
	endKey = append(endKey, []byte(txid)...)
//由于txid是一个固定长度的字符串（即128位长的uuid），因此0xff可以用作阻止器。
//否则，给定txid的超级字符串也将落在范围查询的结束键下。
	endKey = append(endKey, byte(0xff))
	return endKey
}

//CreatePurgeIndexByHeightRangeStartKey返回StartKey以对存储在临时存储区中的索引执行范围查询
//使用块高度
func createPurgeIndexByHeightRangeStartKey(blockHeight uint64) []byte {
	var startKey []byte
	startKey = append(startKey, purgeIndexByHeightPrefix)
	startKey = append(startKey, compositeKeySep)
	startKey = append(startKey, util.EncodeOrderPreservingVarUint64(blockHeight)...)
	startKey = append(startKey, compositeKeySep)
	return startKey
}

//CreatePurgeIndexByHeightRangeEndKey返回一个EndKey，以便对存储在临时存储区中的索引执行范围查询。
//使用块高度
func createPurgeIndexByHeightRangeEndKey(blockHeight uint64) []byte {
	var endKey []byte
	endKey = append(endKey, purgeIndexByHeightPrefix)
	endKey = append(endKey, compositeKeySep)
	endKey = append(endKey, util.EncodeOrderPreservingVarUint64(blockHeight)...)
	endKey = append(endKey, byte(0xff))
	return endKey
}

//CreatePurgeIndexByXidRangeStartKey返回StartKey以对存储在临时存储区中的索引执行范围查询
//使用TXID
func createPurgeIndexByTxidRangeStartKey(txid string) []byte {
	var startKey []byte
	startKey = append(startKey, purgeIndexByTxidPrefix)
	startKey = append(startKey, compositeKeySep)
	startKey = append(startKey, []byte(txid)...)
	startKey = append(startKey, compositeKeySep)
	return startKey
}

//CreatePurgeIndexByXidRangeEndKey返回一个EndKey，以便对存储在临时存储区中的索引执行范围查询。
//使用TXID
func createPurgeIndexByTxidRangeEndKey(txid string) []byte {
	var endKey []byte
	endKey = append(endKey, purgeIndexByTxidPrefix)
	endKey = append(endKey, compositeKeySep)
	endKey = append(endKey, []byte(txid)...)
//由于txid是一个固定长度的字符串（即128位长的uuid），因此0xff可以用作阻止器。
//否则，给定txid的超级字符串也将落在范围查询的结束键下。
	endKey = append(endKey, byte(0xff))
	return endKey
}

//getTransientStorePath返回临时存储私有Rwset的文件系统路径
func GetTransientStorePath() string {
	sysPath := config.GetPath("peer.fileSystemPath")
	return filepath.Join(sysPath, "transientStore")
}

//trimpvtwset返回一个“txpvtreadwriteset”，它只保留在筛选器中提供的“ns/collections”列表。
//nil筛选器不筛选任何结果，并按原样返回原始的“pvtwset”
func trimPvtWSet(pvtWSet *rwset.TxPvtReadWriteSet, filter ledger.PvtNsCollFilter) *rwset.TxPvtReadWriteSet {
	if filter == nil {
		return pvtWSet
	}

	var filteredNsRwSet []*rwset.NsPvtReadWriteSet
	for _, ns := range pvtWSet.NsPvtRwset {
		var filteredCollRwSet []*rwset.CollectionPvtReadWriteSet
		for _, coll := range ns.CollectionPvtRwset {
			if filter.Has(ns.Namespace, coll.CollectionName) {
				filteredCollRwSet = append(filteredCollRwSet, coll)
			}
		}
		if filteredCollRwSet != nil {
			filteredNsRwSet = append(filteredNsRwSet,
				&rwset.NsPvtReadWriteSet{
					Namespace:          ns.Namespace,
					CollectionPvtRwset: filteredCollRwSet,
				},
			)
		}
	}
	var filteredTxPvtRwSet *rwset.TxPvtReadWriteSet
	if filteredNsRwSet != nil {
		filteredTxPvtRwSet = &rwset.TxPvtReadWriteSet{
			DataModel:  pvtWSet.GetDataModel(),
			NsPvtRwset: filteredNsRwSet,
		}
	}
	return filteredTxPvtRwSet
}

func trimPvtCollectionConfigs(configs map[string]*common.CollectionConfigPackage,
	filter ledger.PvtNsCollFilter) (map[string]*common.CollectionConfigPackage, error) {
	if filter == nil {
		return configs, nil
	}
	result := make(map[string]*common.CollectionConfigPackage)

	for ns, pkg := range configs {
		result[ns] = &common.CollectionConfigPackage{}
		for _, colConf := range pkg.GetConfig() {
			switch cconf := colConf.Payload.(type) {
			case *common.CollectionConfig_StaticCollectionConfig:
				if filter.Has(ns, cconf.StaticCollectionConfig.Name) {
					result[ns].Config = append(result[ns].Config, colConf)
				}
			default:
				return nil, errors.New("unexpected collection type")
			}
		}
	}
	return result, nil
}
