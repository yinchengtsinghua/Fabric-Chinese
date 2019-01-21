
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


package rwsetutil

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
)

var logger = flogging.MustGetLogger("rwsetutil")

//rwsetbuilder帮助构建读写集
type RWSetBuilder struct {
	pubRwBuilderMap map[string]*nsPubRwBuilder
	pvtRwBuilderMap map[string]*nsPvtRwBuilder
}

type nsPubRwBuilder struct {
	namespace         string
readMap           map[string]*kvrwset.KVRead //用于MVCC验证
	writeMap          map[string]*kvrwset.KVWrite
	metadataWriteMap  map[string]*kvrwset.KVMetadataWrite
rangeQueriesMap   map[rangeQueryKey]*kvrwset.RangeQueryInfo //用于幻像读取验证
	rangeQueriesKeys  []rangeQueryKey
	collHashRwBuilder map[string]*collHashRwBuilder
}

type collHashRwBuilder struct {
	collName         string
	readMap          map[string]*kvrwset.KVReadHash
	writeMap         map[string]*kvrwset.KVWriteHash
	metadataWriteMap map[string]*kvrwset.KVMetadataWriteHash
	pvtDataHash      []byte
}

type nsPvtRwBuilder struct {
	namespace         string
	collPvtRwBuilders map[string]*collPvtRwBuilder
}

type collPvtRwBuilder struct {
	collectionName   string
	writeMap         map[string]*kvrwset.KVWrite
	metadataWriteMap map[string]*kvrwset.KVMetadataWrite
}

type rangeQueryKey struct {
	startKey     string
	endKey       string
	itrExhausted bool
}

//new rwsetbuilder构造rwsetbuilder的新实例
func NewRWSetBuilder() *RWSetBuilder {
	return &RWSetBuilder{make(map[string]*nsPubRwBuilder), make(map[string]*nsPvtRwBuilder)}
}

//addtoreadset向读取集添加一个键和相应的版本
func (b *RWSetBuilder) AddToReadSet(ns string, key string, version *version.Height) {
	nsPubRwBuilder := b.getOrCreateNsPubRwBuilder(ns)
	nsPubRwBuilder.readMap[key] = NewKVRead(key, version)
}

//addToWriteset向写入集添加键和值
func (b *RWSetBuilder) AddToWriteSet(ns string, key string, value []byte) {
	nsPubRwBuilder := b.getOrCreateNsPubRwBuilder(ns)
	nsPubRwBuilder.writeMap[key] = newKVWrite(key, value)
}

//AddToMetadataWriteSet adds a metadata to a key in the write-set
//“metadata”参数的nil/empty映射指示删除元数据
func (b *RWSetBuilder) AddToMetadataWriteSet(ns, key string, metadata map[string][]byte) {
	b.getOrCreateNsPubRwBuilder(ns).
		metadataWriteMap[key] = mapToMetadataWrite(key, metadata)
}

//addtorangequeryset添加用于执行幻象读取验证的范围查询信息
func (b *RWSetBuilder) AddToRangeQuerySet(ns string, rqi *kvrwset.RangeQueryInfo) {
	nsPubRwBuilder := b.getOrCreateNsPubRwBuilder(ns)
	key := rangeQueryKey{rqi.StartKey, rqi.EndKey, rqi.ItrExhausted}
	_, ok := nsPubRwBuilder.rangeQueriesMap[key]
	if !ok {
		nsPubRwBuilder.rangeQueriesMap[key] = rqi
		nsPubRwBuilder.rangeQueriesKeys = append(nsPubRwBuilder.rangeQueriesKeys, key)
	}
}

//addtohashedreadset向散列读取集添加键和相应的版本
func (b *RWSetBuilder) AddToHashedReadSet(ns string, coll string, key string, version *version.Height) {
	kvReadHash := newPvtKVReadHash(key, version)
	b.getOrCreateCollHashedRwBuilder(ns, coll).readMap[key] = kvReadHash
}

//addtopvtandhashedwriteset向私有和哈希写入集添加键和值
func (b *RWSetBuilder) AddToPvtAndHashedWriteSet(ns string, coll string, key string, value []byte) {
	kvWrite, kvWriteHash := newPvtKVWriteAndHash(key, value)
	b.getOrCreateCollPvtRwBuilder(ns, coll).writeMap[key] = kvWrite
	b.getOrCreateCollHashedRwBuilder(ns, coll).writeMap[key] = kvWriteHash
}

//addToHashedMetadataWriteSet向哈希写入集中的键添加元数据
func (b *RWSetBuilder) AddToHashedMetadataWriteSet(ns, coll, key string, metadata map[string][]byte) {
//pvt写入集只需要密钥，而不是整个元数据。元数据仅存储
//通过哈希键。pvt写入集需要知道处理特殊情况的键
//元数据已更新，因此pvt数据中存在的密钥版本应递增。
	b.getOrCreateCollPvtRwBuilder(ns, coll).
		metadataWriteMap[key] = &kvrwset.KVMetadataWrite{Key: key, Entries: nil}
	b.getOrCreateCollHashedRwBuilder(ns, coll).
		metadataWriteMap[key] = mapToMetadataWriteHash(key, metadata)
}

//GetTxSimulationResults返回公共RWset的协议字节
//（公共数据+私有数据散列）和事务的私有RWset
func (b *RWSetBuilder) GetTxSimulationResults() (*ledger.TxSimulationResults, error) {
	pvtData := b.getTxPvtReadWriteSet()
	var err error

	var pubDataProto *rwset.TxReadWriteSet
	var pvtDataProto *rwset.TxPvtReadWriteSet

//将集合级散列填充到pub rwset中，并计算pvt rwset的协议字节。
	if pvtData != nil {
		if pvtDataProto, err = pvtData.toProtoMsg(); err != nil {
			return nil, err
		}
		for _, ns := range pvtDataProto.NsPvtRwset {
			for _, coll := range ns.CollectionPvtRwset {
				b.setPvtCollectionHash(ns.Namespace, coll.CollectionName, coll.Rwset)
			}
		}
	}
//计算pub rwset的协议字节
	pubSet := b.GetTxReadWriteSet()
	if pubSet != nil {
		if pubDataProto, err = b.GetTxReadWriteSet().toProtoMsg(); err != nil {
			return nil, err
		}
	}
	return &ledger.TxSimulationResults{
		PubSimulationResults: pubDataProto,
		PvtSimulationResults: pvtDataProto,
	}, nil
}

func (b *RWSetBuilder) setPvtCollectionHash(ns string, coll string, pvtDataProto []byte) {
	collHashedBuilder := b.getOrCreateCollHashedRwBuilder(ns, coll)
	collHashedBuilder.pvtDataHash = util.ComputeHash(pvtDataProto)
}

//gettxreadwriteset返回读写集
//todo在txmgr开始使用此处介绍的新函数“gettxSimulationResults”时将此函数设为私有
func (b *RWSetBuilder) GetTxReadWriteSet() *TxRwSet {
	sortedNsPubBuilders := []*nsPubRwBuilder{}
	util.GetValuesBySortedKeys(&(b.pubRwBuilderMap), &sortedNsPubBuilders)

	var nsPubRwSets []*NsRwSet
	for _, nsPubRwBuilder := range sortedNsPubBuilders {
		nsPubRwSets = append(nsPubRwSets, nsPubRwBuilder.build())
	}
	return &TxRwSet{NsRwSets: nsPubRwSets}
}

//gettxpvtreadwriteset返回私有读写集
func (b *RWSetBuilder) getTxPvtReadWriteSet() *TxPvtRwSet {
	sortedNsPvtBuilders := []*nsPvtRwBuilder{}
	util.GetValuesBySortedKeys(&(b.pvtRwBuilderMap), &sortedNsPvtBuilders)

	var nsPvtRwSets []*NsPvtRwSet
	for _, nsPvtRwBuilder := range sortedNsPvtBuilders {
		nsPvtRwSets = append(nsPvtRwSets, nsPvtRwBuilder.build())
	}
	if len(nsPvtRwSets) == 0 {
		return nil
	}
	return &TxPvtRwSet{NsPvtRwSet: nsPvtRwSets}
}

func (b *nsPubRwBuilder) build() *NsRwSet {
	var readSet []*kvrwset.KVRead
	var writeSet []*kvrwset.KVWrite
	var metadataWriteSet []*kvrwset.KVMetadataWrite
	var rangeQueriesInfo []*kvrwset.RangeQueryInfo
	var collHashedRwSet []*CollHashedRwSet
//添加读集
	util.GetValuesBySortedKeys(&(b.readMap), &readSet)
//添加写集
	util.GetValuesBySortedKeys(&(b.writeMap), &writeSet)
	util.GetValuesBySortedKeys(&(b.metadataWriteMap), &metadataWriteSet)
//添加范围查询信息
	for _, key := range b.rangeQueriesKeys {
		rangeQueriesInfo = append(rangeQueriesInfo, b.rangeQueriesMap[key])
	}
//为私有集合添加哈希RW
	sortedCollBuilders := []*collHashRwBuilder{}
	util.GetValuesBySortedKeys(&(b.collHashRwBuilder), &sortedCollBuilders)
	for _, collBuilder := range sortedCollBuilders {
		collHashedRwSet = append(collHashedRwSet, collBuilder.build())
	}
	return &NsRwSet{
		NameSpace: b.namespace,
		KvRwSet: &kvrwset.KVRWSet{
			Reads:            readSet,
			Writes:           writeSet,
			MetadataWrites:   metadataWriteSet,
			RangeQueriesInfo: rangeQueriesInfo,
		},
		CollHashedRwSets: collHashedRwSet,
	}
}

func (b *nsPvtRwBuilder) build() *NsPvtRwSet {
	sortedCollBuilders := []*collPvtRwBuilder{}
	util.GetValuesBySortedKeys(&(b.collPvtRwBuilders), &sortedCollBuilders)

	var collPvtRwSets []*CollPvtRwSet
	for _, collBuilder := range sortedCollBuilders {
		collPvtRwSets = append(collPvtRwSets, collBuilder.build())
	}
	return &NsPvtRwSet{NameSpace: b.namespace, CollPvtRwSets: collPvtRwSets}
}

func (b *collHashRwBuilder) build() *CollHashedRwSet {
	var readSet []*kvrwset.KVReadHash
	var writeSet []*kvrwset.KVWriteHash
	var metadataWriteSet []*kvrwset.KVMetadataWriteHash

	util.GetValuesBySortedKeys(&(b.readMap), &readSet)
	util.GetValuesBySortedKeys(&(b.writeMap), &writeSet)
	util.GetValuesBySortedKeys(&(b.metadataWriteMap), &metadataWriteSet)
	return &CollHashedRwSet{
		CollectionName: b.collName,
		HashedRwSet: &kvrwset.HashedRWSet{
			HashedReads:    readSet,
			HashedWrites:   writeSet,
			MetadataWrites: metadataWriteSet,
		},
		PvtRwSetHash: b.pvtDataHash,
	}
}

func (b *collPvtRwBuilder) build() *CollPvtRwSet {
	var writeSet []*kvrwset.KVWrite
	var metadataWriteSet []*kvrwset.KVMetadataWrite
	util.GetValuesBySortedKeys(&(b.writeMap), &writeSet)
	util.GetValuesBySortedKeys(&(b.metadataWriteMap), &metadataWriteSet)
	return &CollPvtRwSet{
		CollectionName: b.collectionName,
		KvRwSet: &kvrwset.KVRWSet{
			Writes:         writeSet,
			MetadataWrites: metadataWriteSet,
		},
	}
}

func (b *RWSetBuilder) getOrCreateNsPubRwBuilder(ns string) *nsPubRwBuilder {
	nsPubRwBuilder, ok := b.pubRwBuilderMap[ns]
	if !ok {
		nsPubRwBuilder = newNsPubRwBuilder(ns)
		b.pubRwBuilderMap[ns] = nsPubRwBuilder
	}
	return nsPubRwBuilder
}

func (b *RWSetBuilder) getOrCreateNsPvtRwBuilder(ns string) *nsPvtRwBuilder {
	nsPvtRwBuilder, ok := b.pvtRwBuilderMap[ns]
	if !ok {
		nsPvtRwBuilder = newNsPvtRwBuilder(ns)
		b.pvtRwBuilderMap[ns] = nsPvtRwBuilder
	}
	return nsPvtRwBuilder
}

func (b *RWSetBuilder) getOrCreateCollHashedRwBuilder(ns string, coll string) *collHashRwBuilder {
	nsPubRwBuilder := b.getOrCreateNsPubRwBuilder(ns)
	collHashRwBuilder, ok := nsPubRwBuilder.collHashRwBuilder[coll]
	if !ok {
		collHashRwBuilder = newCollHashRwBuilder(coll)
		nsPubRwBuilder.collHashRwBuilder[coll] = collHashRwBuilder
	}
	return collHashRwBuilder
}

func (b *RWSetBuilder) getOrCreateCollPvtRwBuilder(ns string, coll string) *collPvtRwBuilder {
	nsPvtRwBuilder := b.getOrCreateNsPvtRwBuilder(ns)
	collPvtRwBuilder, ok := nsPvtRwBuilder.collPvtRwBuilders[coll]
	if !ok {
		collPvtRwBuilder = newCollPvtRwBuilder(coll)
		nsPvtRwBuilder.collPvtRwBuilders[coll] = collPvtRwBuilder
	}
	return collPvtRwBuilder
}

func newNsPubRwBuilder(namespace string) *nsPubRwBuilder {
	return &nsPubRwBuilder{
		namespace,
		make(map[string]*kvrwset.KVRead),
		make(map[string]*kvrwset.KVWrite),
		make(map[string]*kvrwset.KVMetadataWrite),
		make(map[rangeQueryKey]*kvrwset.RangeQueryInfo),
		nil,
		make(map[string]*collHashRwBuilder),
	}
}

func newNsPvtRwBuilder(namespace string) *nsPvtRwBuilder {
	return &nsPvtRwBuilder{namespace, make(map[string]*collPvtRwBuilder)}
}

func newCollHashRwBuilder(collName string) *collHashRwBuilder {
	return &collHashRwBuilder{
		collName,
		make(map[string]*kvrwset.KVReadHash),
		make(map[string]*kvrwset.KVWriteHash),
		make(map[string]*kvrwset.KVMetadataWriteHash),
		nil,
	}
}

func newCollPvtRwBuilder(collName string) *collPvtRwBuilder {
	return &collPvtRwBuilder{
		collName,
		make(map[string]*kvrwset.KVWrite),
		make(map[string]*kvrwset.KVMetadataWrite),
	}
}

func mapToMetadataWrite(key string, m map[string][]byte) *kvrwset.KVMetadataWrite {
	proto := &kvrwset.KVMetadataWrite{Key: key}
	names := util.GetSortedKeys(m)
	for _, name := range names {
		proto.Entries = append(proto.Entries,
			&kvrwset.KVMetadataEntry{Name: name, Value: m[name]},
		)
	}
	return proto
}

func mapToMetadataWriteHash(key string, m map[string][]byte) *kvrwset.KVMetadataWriteHash {
	proto := &kvrwset.KVMetadataWriteHash{KeyHash: util.ComputeStringHash(key)}
	names := util.GetSortedKeys(m)
	for _, name := range names {
		proto.Entries = append(proto.Entries,
			&kvrwset.KVMetadataEntry{Name: name, Value: m[name]},
		)
	}
	return proto
}
