
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


package pvtstatepurgemgmt

import (
	proto "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
)

var logger = flogging.MustGetLogger("pvtstatepurgemgmt")

const (
	expiryPrefix = '1'
)

//expiryinfokey用作簿记员中某个条目的键（由leveldb实例支持）
type expiryInfoKey struct {
	committingBlk uint64
	expiryBlk     uint64
}

//expiryinfo封装“expiryinfokey”和相应的私有数据密钥。
//换句话说，此结构封装由提交的键和键哈希
//块号“expiryinfokey.committingblk”应过期（因此清除）
//提交块号为“expiryinfokey.expiryblk”的
type expiryInfo struct {
	expiryInfoKey *expiryInfoKey
	pvtdataKeys   *PvtdataKeys
}

//ExpireyKeeper用于跟踪pvtData空间中的过期项目。
type expiryKeeper interface {
//更新簿记跟踪密钥列表及其相应的到期块编号。
	updateBookkeeping(toTrack []*expiryInfo, toClear []*expiryInfoKey) error
//retrieve返回按给定的块号应该过期的键信息
	retrieve(expiringAtBlkNum uint64) ([]*expiryInfo, error)
//RetrieveByExpiryKey检索给定ExpiryKey的ExpiryInfo
	retrieveByExpiryKey(expiryKey *expiryInfoKey) (*expiryInfo, error)
}

func newExpiryKeeper(ledgerid string, provider bookkeeping.Provider) expiryKeeper {
	return &expKeeper{provider.GetDBHandle(ledgerid, bookkeeping.PvtdataExpiry)}
}

type expKeeper struct {
	db *leveldbhelper.DBHandle
}

//更新簿记更新簿记员存储的信息
//“totrack”参数导致簿记员中出现新条目，“toclear”参数包含
//从簿记员那里移走。这个函数是通过每个块的提交来调用的。作为一个
//例如，块号为50的块的提交，“totrack”参数可能包含以下两个条目：
//（1）&expiryinfo&expiryinfokey承诺blk:50，expiryblk:55，pvtd数据键….和
//（2）&expiryinfo&expiryinfokey承诺blk:50，expiryblk:60，pvtd数据键….
//第一个条目中的“pvtdatakeys”包含在块55处过期的所有键（和键散列）（即，这些集合的btl配置为4）
//第二个条目中的“pvtdatakeys”包含在块60过期的所有键（和键散列）（即，这些集合的btl配置为9）。
//同样，继续上述示例，参数“toclear”可能包含以下两个条目
//（1）&expiryinfokey承诺blk:45，expiryinfokey:50和（2）&expiryinfokey承诺blk:40，expiryinfokey:50。第一个条目已创建
//然而，在块号45的提交时和第二个条目是在块号40的提交时创建的。
//两者都将在提交块50时过期。
func (ek *expKeeper) updateBookkeeping(toTrack []*expiryInfo, toClear []*expiryInfoKey) error {
	updateBatch := leveldbhelper.NewUpdateBatch()
	for _, expinfo := range toTrack {
		k, v, err := encodeKV(expinfo)
		if err != nil {
			return err
		}
		updateBatch.Put(k, v)
	}
	for _, expinfokey := range toClear {
		updateBatch.Delete(encodeExpiryInfoKey(expinfokey))
	}
	return ek.db.WriteBatch(updateBatch, true)
}

func (ek *expKeeper) retrieve(expiringAtBlkNum uint64) ([]*expiryInfo, error) {
	startKey := encodeExpiryInfoKey(&expiryInfoKey{expiryBlk: expiringAtBlkNum, committingBlk: 0})
	endKey := encodeExpiryInfoKey(&expiryInfoKey{expiryBlk: expiringAtBlkNum + 1, committingBlk: 0})
	itr := ek.db.GetIterator(startKey, endKey)
	defer itr.Release()

	var listExpinfo []*expiryInfo
	for itr.Next() {
		expinfo, err := decodeExpiryInfo(itr.Key(), itr.Value())
		if err != nil {
			return nil, err
		}
		listExpinfo = append(listExpinfo, expinfo)
	}
	return listExpinfo, nil
}

func (ek *expKeeper) retrieveByExpiryKey(expiryKey *expiryInfoKey) (*expiryInfo, error) {
	key := encodeExpiryInfoKey(expiryKey)
	value, err := ek.db.Get(key)
	if err != nil {
		return nil, err
	}
	return decodeExpiryInfo(key, value)
}

func encodeKV(expinfo *expiryInfo) (key []byte, value []byte, err error) {
	key = encodeExpiryInfoKey(expinfo.expiryInfoKey)
	value, err = encodeExpiryInfoValue(expinfo.pvtdataKeys)
	return
}

func encodeExpiryInfoKey(expinfoKey *expiryInfoKey) []byte {
	key := append([]byte{expiryPrefix}, util.EncodeOrderPreservingVarUint64(expinfoKey.expiryBlk)...)
	return append(key, util.EncodeOrderPreservingVarUint64(expinfoKey.committingBlk)...)
}

func encodeExpiryInfoValue(pvtdataKeys *PvtdataKeys) ([]byte, error) {
	return proto.Marshal(pvtdataKeys)
}

func decodeExpiryInfo(key []byte, value []byte) (*expiryInfo, error) {
	expiryBlk, n := util.DecodeOrderPreservingVarUint64(key[1:])
	committingBlk, _ := util.DecodeOrderPreservingVarUint64(key[n+1:])
	pvtdataKeys := &PvtdataKeys{}
	if err := proto.Unmarshal(value, pvtdataKeys); err != nil {
		return nil, err
	}
	return &expiryInfo{
			expiryInfoKey: &expiryInfoKey{committingBlk: committingBlk, expiryBlk: expiryBlk},
			pvtdataKeys:   pvtdataKeys},
		nil
}
