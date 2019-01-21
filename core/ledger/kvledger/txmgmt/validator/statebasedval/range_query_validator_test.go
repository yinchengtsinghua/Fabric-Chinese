
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package statebasedval

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/stateleveldb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/stretchr/testify/assert"
)

func TestRangeQueryBoundaryConditions(t *testing.T) {
	batch := statedb.NewUpdateBatch()
	batch.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 0))
	batch.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 1))
	batch.Put("ns1", "key3", []byte("value3"), version.NewHeight(1, 2))
	batch.Put("ns1", "key4", []byte("value4"), version.NewHeight(1, 3))
	batch.Put("ns1", "key5", []byte("value5"), version.NewHeight(1, 4))

	testcase1 := "NoResults"
	rqi1 := &kvrwset.RangeQueryInfo{StartKey: "key7", EndKey: "key10", ItrExhausted: true}
	rqi1.SetRawReads([]*kvrwset.KVRead{})
	testRangeQuery(t, testcase1, batch, version.NewHeight(1, 4), "ns1", rqi1, true)

	testcase2 := "NoResultsDuringValidation"
	rqi2 := &kvrwset.RangeQueryInfo{StartKey: "key7", EndKey: "key10", ItrExhausted: true}
	rqi2.SetRawReads([]*kvrwset.KVRead{rwsetutil.NewKVRead("key8", version.NewHeight(1, 8))})
	testRangeQuery(t, testcase2, batch, version.NewHeight(1, 4), "ns1", rqi2, false)

	testcase3 := "OneExtraTailingResultsDuringValidation"
	rqi3 := &kvrwset.RangeQueryInfo{StartKey: "key1", EndKey: "key4", ItrExhausted: true}
	rqi3.SetRawReads([]*kvrwset.KVRead{
		rwsetutil.NewKVRead("key1", version.NewHeight(1, 0)),
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
	})
	testRangeQuery(t, testcase3, batch, version.NewHeight(1, 4), "ns1", rqi3, false)

	testcase4 := "TwoExtraTailingResultsDuringValidation"
	rqi4 := &kvrwset.RangeQueryInfo{StartKey: "key1", EndKey: "key5", ItrExhausted: true}
	rqi4.SetRawReads([]*kvrwset.KVRead{
		rwsetutil.NewKVRead("key1", version.NewHeight(1, 0)),
		rwsetutil.NewKVRead("key2", version.NewHeight(1, 1)),
	})
	testRangeQuery(t, testcase4, batch, version.NewHeight(1, 4), "ns1", rqi4, false)
}

func testRangeQuery(t *testing.T, testcase string, stateData *statedb.UpdateBatch, savepoint *version.Height,
	ns string, rqi *kvrwset.RangeQueryInfo, expectedResult bool) {
	t.Run(testcase, func(t *testing.T) {
		testDBEnv := stateleveldb.NewTestVDBEnv(t)
		defer testDBEnv.Cleanup()
		db, err := testDBEnv.DBProvider.GetDBHandle("TestDB")
		assert.NoError(t, err)
		if stateData != nil {
			db.ApplyUpdates(stateData, savepoint)
		}

		itr, err := db.GetStateRangeScanIterator(ns, rqi.StartKey, rqi.EndKey)
		assert.NoError(t, err)
		validator := &rangeQueryResultsValidator{}
		validator.init(rqi, itr)
		isValid, err := validator.validate()
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, isValid)
	})
}
