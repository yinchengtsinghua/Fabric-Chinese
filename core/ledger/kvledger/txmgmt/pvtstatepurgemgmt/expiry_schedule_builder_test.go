
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
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/stretchr/testify/assert"
)

func TestBuildExpirySchedule(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns1", "coll1"}: 1,
			{"ns1", "coll2"}: 2,
			{"ns2", "coll3"}: 3,
			{"ns3", "coll4"}: 0,
		},
	)
	updates := privacyenabledstate.NewUpdateBatch()
	updates.PubUpdates.Put("ns1", "pubkey1", []byte("pubvalue1"), version.NewHeight(1, 1))
	putPvtAndHashUpdates(t, updates, "ns1", "coll1", "pvtkey1", []byte("pvtvalue1"), version.NewHeight(1, 1))
	putPvtAndHashUpdates(t, updates, "ns1", "coll2", "pvtkey2", []byte("pvtvalue2"), version.NewHeight(2, 1))
	putPvtAndHashUpdates(t, updates, "ns2", "coll3", "pvtkey3", []byte("pvtvalue3"), version.NewHeight(3, 1))
	putPvtAndHashUpdates(t, updates, "ns3", "coll4", "pvtkey4", []byte("pvtvalue4"), version.NewHeight(4, 1))

	listExpinfo, err := buildExpirySchedule(btlPolicy, updates.PvtUpdates, updates.HashUpdates)
	assert.NoError(t, err)
	t.Logf("listExpinfo=%s", spew.Sdump(listExpinfo))

	pvtdataKeys1 := newPvtdataKeys()
	pvtdataKeys1.add("ns1", "coll1", "pvtkey1", util.ComputeStringHash("pvtkey1"))

	pvtdataKeys2 := newPvtdataKeys()
	pvtdataKeys2.add("ns1", "coll2", "pvtkey2", util.ComputeStringHash("pvtkey2"))

	pvtdataKeys3 := newPvtdataKeys()
	pvtdataKeys3.add("ns2", "coll3", "pvtkey3", util.ComputeStringHash("pvtkey3"))

	expectedListExpInfo := []*expiryInfo{
		{expiryInfoKey: &expiryInfoKey{expiryBlk: 3, committingBlk: 1}, pvtdataKeys: pvtdataKeys1},
		{expiryInfoKey: &expiryInfoKey{expiryBlk: 5, committingBlk: 2}, pvtdataKeys: pvtdataKeys2},
		{expiryInfoKey: &expiryInfoKey{expiryBlk: 7, committingBlk: 3}, pvtdataKeys: pvtdataKeys3},
	}

	assert.Len(t, listExpinfo, 3)
	assert.ElementsMatch(t, expectedListExpInfo, listExpinfo)
}

func TestBuildExpiryScheduleWithMissingPvtdata(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns1", "coll1"}: 1,
			{"ns1", "coll2"}: 2,
			{"ns2", "coll3"}: 3,
			{"ns3", "coll4"}: 0,
			{"ns3", "coll5"}: 20,
		},
	)

	updates := privacyenabledstate.NewUpdateBatch()

//此更新应在到期计划中同时显示密钥和哈希
	putPvtAndHashUpdates(t, updates, "ns1", "coll1", "pvtkey1", []byte("pvtvalue1"), version.NewHeight(50, 1))

//此更新应仅在密钥哈希的到期计划中出现
	putHashUpdates(updates, "ns1", "coll2", "pvtkey2", []byte("pvtvalue2"), version.NewHeight(50, 2))

//此更新应仅在密钥哈希的到期计划中出现
	putHashUpdates(updates, "ns2", "coll3", "pvtkey3", []byte("pvtvalue3"), version.NewHeight(50, 3))

//此更新预计不会出现在到期计划中，因为此集合已配置为到期-“从不”
	putPvtAndHashUpdates(t, updates, "ns3", "coll4", "pvtkey4", []byte("pvtvalue4"), version.NewHeight(50, 4))

//以下两个更新不应出现在到期计划中，因为它们是删除的
	deletePvtAndHashUpdates(t, updates, "ns3", "coll5", "pvtkey5", version.NewHeight(50, 5))
	deleteHashUpdates(updates, "ns3", "coll5", "pvtkey6", version.NewHeight(50, 6))

	listExpinfo, err := buildExpirySchedule(btlPolicy, updates.PvtUpdates, updates.HashUpdates)
	assert.NoError(t, err)
	t.Logf("listExpinfo=%s", spew.Sdump(listExpinfo))

	pvtdataKeys1 := newPvtdataKeys()
	pvtdataKeys1.add("ns1", "coll1", "pvtkey1", util.ComputeStringHash("pvtkey1"))
	pvtdataKeys2 := newPvtdataKeys()
	pvtdataKeys2.add("ns1", "coll2", "", util.ComputeStringHash("pvtkey2"))
	pvtdataKeys3 := newPvtdataKeys()
	pvtdataKeys3.add("ns2", "coll3", "", util.ComputeStringHash("pvtkey3"))

	expectedListExpInfo := []*expiryInfo{
		{expiryInfoKey: &expiryInfoKey{expiryBlk: 52, committingBlk: 50}, pvtdataKeys: pvtdataKeys1},
		{expiryInfoKey: &expiryInfoKey{expiryBlk: 53, committingBlk: 50}, pvtdataKeys: pvtdataKeys2},
		{expiryInfoKey: &expiryInfoKey{expiryBlk: 54, committingBlk: 50}, pvtdataKeys: pvtdataKeys3},
	}

	assert.Len(t, listExpinfo, 3)
	assert.ElementsMatch(t, expectedListExpInfo, listExpinfo)
}

func putPvtAndHashUpdates(t *testing.T, updates *privacyenabledstate.UpdateBatch, ns, coll, key string, value []byte, ver *version.Height) {
	putPvtUpdates(updates, ns, coll, key, value, ver)
	putHashUpdates(updates, ns, coll, key, value, ver)
}

func deletePvtAndHashUpdates(t *testing.T, updates *privacyenabledstate.UpdateBatch, ns, coll, key string, ver *version.Height) {
	updates.PvtUpdates.Delete(ns, coll, key, ver)
	deleteHashUpdates(updates, ns, coll, key, ver)
}

func putHashUpdates(updates *privacyenabledstate.UpdateBatch, ns, coll, key string, value []byte, ver *version.Height) {
	updates.HashUpdates.Put(ns, coll, util.ComputeStringHash(key), util.ComputeHash(value), ver)
}

func putPvtUpdates(updates *privacyenabledstate.UpdateBatch, ns, coll, key string, value []byte, ver *version.Height) {
	updates.PvtUpdates.Put(ns, coll, key, value, ver)
}

func deleteHashUpdates(updates *privacyenabledstate.UpdateBatch, ns, coll, key string, ver *version.Height) {
	updates.HashUpdates.Delete(ns, coll, util.ComputeStringHash(key), ver)
}
