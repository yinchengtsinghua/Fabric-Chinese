
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
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
)

//////////////////////////////////////////////
//与公共读写集相关的消息
//////////////////////////////////////////////

//txrwset充当'rwset.txreadwriteset'协议消息的代理，并帮助专门为kv数据模型构造读写集。
type TxRwSet struct {
	NsRwSets []*NsRwSet
}

//NSRwset为特定的名称空间（链码）封装“kvrwset.kvrwset”协议消息
type NsRwSet struct {
	NameSpace        string
	KvRwSet          *kvrwset.KVRWSet
	CollHashedRwSets []*CollHashedRwSet
}

//collhashedrwset封装特定集合的“kvrwset.hashedrwset”协议消息
type CollHashedRwSet struct {
	CollectionName string
	HashedRwSet    *kvrwset.HashedRWSet
	PvtRwSetHash   []byte
}

//getpvtdatahash返回给定命名空间和集合的pvtrwsethash
func (txRwSet *TxRwSet) GetPvtDataHash(ns, coll string) []byte {
//我们可以建立并使用一个映射来减少查找的次数
//以后再打电话。但是，我们决定推迟这种优化
//由于以下假设（主要是为了避免额外的LOC）。
//我们假设txrwset中的名称空间和集合数
//非常小（用一个数字表示）
	for _, nsRwSet := range txRwSet.NsRwSets {
		if nsRwSet.NameSpace != ns {
			continue
		}
		return nsRwSet.getPvtDataHash(coll)
	}
	return nil
}

func (nsRwSet *NsRwSet) getPvtDataHash(coll string) []byte {
	for _, collHashedRwSet := range nsRwSet.CollHashedRwSets {
		if collHashedRwSet.CollectionName != coll {
			continue
		}
		return collHashedRwSet.PvtRwSetHash
	}
	return nil
}

//////////////////////////////////////////////
//与专用读写集相关的消息
//////////////////////////////////////////////

//txpvtrwset表示“rwset.txpvtreadwriteset”协议消息
type TxPvtRwSet struct {
	NsPvtRwSet []*NsPvtRwSet
}

//nspvtrwset表示“rwset.nspvtreadwriteset”协议消息
type NsPvtRwSet struct {
	NameSpace     string
	CollPvtRwSets []*CollPvtRwSet
}

//collpvtrwset为特定集合的私有rwset封装“kvrwset.kvrwset”协议消息
//私有rwset中的kvrwset不应包含范围查询信息
type CollPvtRwSet struct {
	CollectionName string
	KvRwSet        *kvrwset.KVRWSet
}

//////////////////////////////////////////////
//用于将消息转换为/从协议字节转换的函数
//////////////////////////////////////////////

//toprotobytes构造txreadwriteset proto消息并使用protobuf marshal进行序列化
func (txRwSet *TxRwSet) ToProtoBytes() ([]byte, error) {
	var protoMsg *rwset.TxReadWriteSet
	var err error
	if protoMsg, err = txRwSet.toProtoMsg(); err != nil {
		return nil, err
	}
	return proto.Marshal(protoMsg)
}

//FromProtoBytes将ProtoBytes反序列化为TxReadWriteSet Proto消息并填充“TxRwset”
func (txRwSet *TxRwSet) FromProtoBytes(protoBytes []byte) error {
	protoMsg := &rwset.TxReadWriteSet{}
	var err error
	var txRwSetTemp *TxRwSet
	if err = proto.Unmarshal(protoBytes, protoMsg); err != nil {
		return err
	}
	if txRwSetTemp, err = TxRwSetFromProtoMsg(protoMsg); err != nil {
		return err
	}
	txRwSet.NsRwSets = txRwSetTemp.NsRwSets
	return nil
}

//toprotobytes使用protobuf marshal构造“txpvtreadwriteset”proto消息并序列化
func (txPvtRwSet *TxPvtRwSet) ToProtoBytes() ([]byte, error) {
	var protoMsg *rwset.TxPvtReadWriteSet
	var err error
	if protoMsg, err = txPvtRwSet.toProtoMsg(); err != nil {
		return nil, err
	}
	return proto.Marshal(protoMsg)
}

//FromProtoBytes将ProtoBytes反序列化为“TxPvTreadWriteSet”Proto消息并填充“TxPvTrwset”
func (txPvtRwSet *TxPvtRwSet) FromProtoBytes(protoBytes []byte) error {
	protoMsg := &rwset.TxPvtReadWriteSet{}
	var err error
	var txPvtRwSetTemp *TxPvtRwSet
	if err = proto.Unmarshal(protoBytes, protoMsg); err != nil {
		return err
	}
	if txPvtRwSetTemp, err = TxPvtRwSetFromProtoMsg(protoMsg); err != nil {
		return err
	}
	txPvtRwSet.NsPvtRwSet = txPvtRwSetTemp.NsPvtRwSet
	return nil
}

func (txRwSet *TxRwSet) toProtoMsg() (*rwset.TxReadWriteSet, error) {
	protoMsg := &rwset.TxReadWriteSet{DataModel: rwset.TxReadWriteSet_KV}
	var nsRwSetProtoMsg *rwset.NsReadWriteSet
	var err error
	for _, nsRwSet := range txRwSet.NsRwSets {
		if nsRwSetProtoMsg, err = nsRwSet.toProtoMsg(); err != nil {
			return nil, err
		}
		protoMsg.NsRwset = append(protoMsg.NsRwset, nsRwSetProtoMsg)
	}
	return protoMsg, nil
}

func TxRwSetFromProtoMsg(protoMsg *rwset.TxReadWriteSet) (*TxRwSet, error) {
	txRwSet := &TxRwSet{}
	var nsRwSet *NsRwSet
	var err error
	for _, nsRwSetProtoMsg := range protoMsg.NsRwset {
		if nsRwSet, err = nsRwSetFromProtoMsg(nsRwSetProtoMsg); err != nil {
			return nil, err
		}
		txRwSet.NsRwSets = append(txRwSet.NsRwSets, nsRwSet)
	}
	return txRwSet, nil
}

func (nsRwSet *NsRwSet) toProtoMsg() (*rwset.NsReadWriteSet, error) {
	var err error
	protoMsg := &rwset.NsReadWriteSet{Namespace: nsRwSet.NameSpace}
	if protoMsg.Rwset, err = proto.Marshal(nsRwSet.KvRwSet); err != nil {
		return nil, err
	}

	var collHashedRwSetProtoMsg *rwset.CollectionHashedReadWriteSet
	for _, collHashedRwSet := range nsRwSet.CollHashedRwSets {
		if collHashedRwSetProtoMsg, err = collHashedRwSet.toProtoMsg(); err != nil {
			return nil, err
		}
		protoMsg.CollectionHashedRwset = append(protoMsg.CollectionHashedRwset, collHashedRwSetProtoMsg)
	}
	return protoMsg, nil
}

func nsRwSetFromProtoMsg(protoMsg *rwset.NsReadWriteSet) (*NsRwSet, error) {
	nsRwSet := &NsRwSet{NameSpace: protoMsg.Namespace, KvRwSet: &kvrwset.KVRWSet{}}
	if err := proto.Unmarshal(protoMsg.Rwset, nsRwSet.KvRwSet); err != nil {
		return nil, err
	}
	var err error
	var collHashedRwSet *CollHashedRwSet
	for _, collHashedRwSetProtoMsg := range protoMsg.CollectionHashedRwset {
		if collHashedRwSet, err = collHashedRwSetFromProtoMsg(collHashedRwSetProtoMsg); err != nil {
			return nil, err
		}
		nsRwSet.CollHashedRwSets = append(nsRwSet.CollHashedRwSets, collHashedRwSet)
	}
	return nsRwSet, nil
}

func (collHashedRwSet *CollHashedRwSet) toProtoMsg() (*rwset.CollectionHashedReadWriteSet, error) {
	var err error
	protoMsg := &rwset.CollectionHashedReadWriteSet{
		CollectionName: collHashedRwSet.CollectionName,
		PvtRwsetHash:   collHashedRwSet.PvtRwSetHash,
	}
	if protoMsg.HashedRwset, err = proto.Marshal(collHashedRwSet.HashedRwSet); err != nil {
		return nil, err
	}
	return protoMsg, nil
}

func collHashedRwSetFromProtoMsg(protoMsg *rwset.CollectionHashedReadWriteSet) (*CollHashedRwSet, error) {
	colHashedRwSet := &CollHashedRwSet{
		CollectionName: protoMsg.CollectionName,
		PvtRwSetHash:   protoMsg.PvtRwsetHash,
		HashedRwSet:    &kvrwset.HashedRWSet{},
	}
	if err := proto.Unmarshal(protoMsg.HashedRwset, colHashedRwSet.HashedRwSet); err != nil {
		return nil, err
	}
	return colHashedRwSet, nil
}

func (txRwSet *TxRwSet) NumCollections() int {
	if txRwSet == nil {
		return 0
	}
	numColls := 0
	for _, nsRwset := range txRwSet.NsRwSets {
		for range nsRwset.CollHashedRwSets {
			numColls++
		}
	}
	return numColls
}

//////////////////////////////////////////////////
//专用读写集函数
//////////////////////////////////////////////////

func (txPvtRwSet *TxPvtRwSet) toProtoMsg() (*rwset.TxPvtReadWriteSet, error) {
	protoMsg := &rwset.TxPvtReadWriteSet{DataModel: rwset.TxReadWriteSet_KV}
	var nsProtoMsg *rwset.NsPvtReadWriteSet
	var err error
	for _, nsPvtRwSet := range txPvtRwSet.NsPvtRwSet {
		if nsProtoMsg, err = nsPvtRwSet.toProtoMsg(); err != nil {
			return nil, err
		}
		protoMsg.NsPvtRwset = append(protoMsg.NsPvtRwset, nsProtoMsg)
	}
	return protoMsg, nil
}

func TxPvtRwSetFromProtoMsg(protoMsg *rwset.TxPvtReadWriteSet) (*TxPvtRwSet, error) {
	txPvtRwset := &TxPvtRwSet{}
	var nsPvtRwSet *NsPvtRwSet
	var err error
	for _, nsRwSetProtoMsg := range protoMsg.NsPvtRwset {
		if nsPvtRwSet, err = nsPvtRwSetFromProtoMsg(nsRwSetProtoMsg); err != nil {
			return nil, err
		}
		txPvtRwset.NsPvtRwSet = append(txPvtRwset.NsPvtRwSet, nsPvtRwSet)
	}
	return txPvtRwset, nil
}

func (nsPvtRwSet *NsPvtRwSet) toProtoMsg() (*rwset.NsPvtReadWriteSet, error) {
	protoMsg := &rwset.NsPvtReadWriteSet{Namespace: nsPvtRwSet.NameSpace}
	var err error
	var collPvtRwSetProtoMsg *rwset.CollectionPvtReadWriteSet
	for _, collPvtRwSet := range nsPvtRwSet.CollPvtRwSets {
		if collPvtRwSetProtoMsg, err = collPvtRwSet.toProtoMsg(); err != nil {
			return nil, err
		}
		protoMsg.CollectionPvtRwset = append(protoMsg.CollectionPvtRwset, collPvtRwSetProtoMsg)
	}
	return protoMsg, err
}

func nsPvtRwSetFromProtoMsg(protoMsg *rwset.NsPvtReadWriteSet) (*NsPvtRwSet, error) {
	nsPvtRwSet := &NsPvtRwSet{NameSpace: protoMsg.Namespace}
	for _, collPvtRwSetProtoMsg := range protoMsg.CollectionPvtRwset {
		var err error
		var collPvtRwSet *CollPvtRwSet
		if collPvtRwSet, err = collPvtRwSetFromProtoMsg(collPvtRwSetProtoMsg); err != nil {
			return nil, err
		}
		nsPvtRwSet.CollPvtRwSets = append(nsPvtRwSet.CollPvtRwSets, collPvtRwSet)
	}
	return nsPvtRwSet, nil
}

func (collPvtRwSet *CollPvtRwSet) toProtoMsg() (*rwset.CollectionPvtReadWriteSet, error) {
	var err error
	protoMsg := &rwset.CollectionPvtReadWriteSet{CollectionName: collPvtRwSet.CollectionName}
	if protoMsg.Rwset, err = proto.Marshal(collPvtRwSet.KvRwSet); err != nil {
		return nil, err
	}
	return protoMsg, nil
}

func collPvtRwSetFromProtoMsg(protoMsg *rwset.CollectionPvtReadWriteSet) (*CollPvtRwSet, error) {
	collPvtRwSet := &CollPvtRwSet{CollectionName: protoMsg.CollectionName, KvRwSet: &kvrwset.KVRWSet{}}
	if err := proto.Unmarshal(protoMsg.Rwset, collPvtRwSet.KvRwSet); err != nil {
		return nil, err
	}
	return collPvtRwSet, nil
}

//newkvread有助于构造原型消息kvrwset.kvread
func NewKVRead(key string, version *version.Height) *kvrwset.KVRead {
	return &kvrwset.KVRead{Key: key, Version: newProtoVersion(version)}
}

//newversion有助于将原型消息kvrwset.version转换为version.height
func NewVersion(protoVersion *kvrwset.Version) *version.Height {
	if protoVersion == nil {
		return nil
	}
	return version.NewHeight(protoVersion.BlockNum, protoVersion.TxNum)
}

func newProtoVersion(height *version.Height) *kvrwset.Version {
	if height == nil {
		return nil
	}
	return &kvrwset.Version{BlockNum: height.BlockNum, TxNum: height.TxNum}
}

func newKVWrite(key string, value []byte) *kvrwset.KVWrite {
	return &kvrwset.KVWrite{Key: key, IsDelete: value == nil, Value: value}
}

func newPvtKVReadHash(key string, version *version.Height) *kvrwset.KVReadHash {
	return &kvrwset.KVReadHash{KeyHash: util.ComputeStringHash(key), Version: newProtoVersion(version)}
}

func newPvtKVWriteAndHash(key string, value []byte) (*kvrwset.KVWrite, *kvrwset.KVWriteHash) {
	kvWrite := newKVWrite(key, value)
	var keyHash, valueHash []byte
	keyHash = util.ComputeStringHash(key)
	if !kvWrite.IsDelete {
		valueHash = util.ComputeHash(value)
	}
	return kvWrite, &kvrwset.KVWriteHash{KeyHash: keyHash, IsDelete: kvWrite.IsDelete, ValueHash: valueHash}
}
