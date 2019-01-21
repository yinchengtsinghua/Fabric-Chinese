
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
	math "math"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/util"
)

type expiryScheduleBuilder struct {
	btlPolicy       pvtdatapolicy.BTLPolicy
	scheduleEntries map[expiryInfoKey]*PvtdataKeys
}

func newExpiryScheduleBuilder(btlPolicy pvtdatapolicy.BTLPolicy) *expiryScheduleBuilder {
	return &expiryScheduleBuilder{btlPolicy, make(map[expiryInfoKey]*PvtdataKeys)}
}

func (builder *expiryScheduleBuilder) add(ns, coll, key string, keyHash []byte, versionedValue *statedb.VersionedValue) error {
	committingBlk := versionedValue.Version.BlockNum
	expiryBlk, err := builder.btlPolicy.GetExpiringBlock(ns, coll, committingBlk)
	if err != nil {
		return err
	}
	if isDelete(versionedValue) || neverExpires(expiryBlk) {
		return nil
	}
	expinfoKey := expiryInfoKey{committingBlk: committingBlk, expiryBlk: expiryBlk}
	pvtdataKeys, ok := builder.scheduleEntries[expinfoKey]
	if !ok {
		pvtdataKeys = newPvtdataKeys()
		builder.scheduleEntries[expinfoKey] = pvtdataKeys
	}
	pvtdataKeys.add(ns, coll, key, keyHash)
	return nil
}

func isDelete(versionedValue *statedb.VersionedValue) bool {
	return versionedValue.Value == nil
}

func neverExpires(expiryBlk uint64) bool {
	return expiryBlk == math.MaxUint64
}

func (builder *expiryScheduleBuilder) getExpiryInfo() []*expiryInfo {
	var listExpinfo []*expiryInfo
	for expinfoKey, pvtdataKeys := range builder.scheduleEntries {
		expinfoKeyCopy := expinfoKey
		listExpinfo = append(listExpinfo, &expiryInfo{expiryInfoKey: &expinfoKeyCopy, pvtdataKeys: pvtdataKeys})
	}
	return listExpinfo
}

func buildExpirySchedule(
	btlPolicy pvtdatapolicy.BTLPolicy,
	pvtUpdates *privacyenabledstate.PvtUpdateBatch,
	hashedUpdates *privacyenabledstate.HashedUpdateBatch) ([]*expiryInfo, error) {

	hashedUpdateKeys := hashedUpdates.ToCompositeKeyMap()
	expiryScheduleBuilder := newExpiryScheduleBuilder(btlPolicy)

	logger.Debugf("Building the expiry schedules based on the update batch")

//循环访问私有数据更新，并将每个密钥添加到到期计划中
//也就是说，当这些私有数据密钥和哈希密钥即将过期时
//请注意，“hashedupdatekeys”可能是pvtupdates的超集。这是因为，
//对等端可能无法接收所有私有数据，这可能是因为对等端不符合某些私有数据的条件。
//或者因为我们允许继续处理丢失的私有数据
	for pvtUpdateKey, vv := range pvtUpdates.ToCompositeKeyMap() {
		keyHash := util.ComputeStringHash(pvtUpdateKey.Key)
		hashedCompisiteKey := privacyenabledstate.HashedCompositeKey{
			Namespace:      pvtUpdateKey.Namespace,
			CollectionName: pvtUpdateKey.CollectionName,
			KeyHash:        string(keyHash),
		}
		logger.Debugf("Adding expiry schedule for key and key hash [%s]", &hashedCompisiteKey)
		if err := expiryScheduleBuilder.add(pvtUpdateKey.Namespace, pvtUpdateKey.CollectionName, pvtUpdateKey.Key, keyHash, vv); err != nil {
			return nil, err
		}
		delete(hashedUpdateKeys, hashedCompisiteKey)
	}

//为剩余的密钥散列添加条目，即不存在私钥对应的散列
	for hashedUpdateKey, vv := range hashedUpdateKeys {
		logger.Debugf("Adding expiry schedule for key hash [%s]", &hashedUpdateKey)
		if err := expiryScheduleBuilder.add(hashedUpdateKey.Namespace, hashedUpdateKey.CollectionName, "", []byte(hashedUpdateKey.KeyHash), vv); err != nil {
			return nil, err
		}
	}
	return expiryScheduleBuilder.getExpiryInfo(), nil
}
