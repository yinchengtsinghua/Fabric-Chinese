
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


package statebasedval

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/stateleveldb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/stretchr/testify/assert"
)

func TestCombinedIterator(t *testing.T) {
	testDBEnv := stateleveldb.NewTestVDBEnv(t)
	defer testDBEnv.Cleanup()

	db, err := testDBEnv.DBProvider.GetDBHandle("TestDB")
	assert.NoError(t, err)

//用初始数据填充数据库
	batch := statedb.NewUpdateBatch()
	batch.Put("ns", "key1", []byte("value1"), version.NewHeight(1, 1))
	batch.Put("ns", "key4", []byte("value4"), version.NewHeight(1, 1))
	batch.Put("ns", "key6", []byte("value6"), version.NewHeight(1, 1))
	batch.Put("ns", "key8", []byte("value8"), version.NewHeight(1, 1))
	db.ApplyUpdates(batch, version.NewHeight(1, 5))

//准备批量
	batch1 := statedb.NewUpdateBatch()
	batch1.Put("ns", "key3", []byte("value3"), version.NewHeight(1, 1))
	batch1.Delete("ns", "key5", version.NewHeight(1, 1))
	batch1.Put("ns", "key6", []byte("value6_new"), version.NewHeight(1, 1))
	batch1.Put("ns", "key7", []byte("value7"), version.NewHeight(1, 1))

//准备第2批（空）
	batch2 := statedb.NewUpdateBatch()

//测试数据库+批处理1更新（不包括endkey）
	itr1, _ := newCombinedIterator(db, batch1, "ns", "key2", "key8", false)
	defer itr1.Close()
	checkItrResults(t, "ExcludeEndKey", itr1, []*statedb.VersionedKV{
		constructVersionedKV("ns", "key3", []byte("value3"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key4", []byte("value4"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key6", []byte("value6_new"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key7", []byte("value7"), version.NewHeight(1, 1)),
	})

//Test db + batch1 updates (include endKey)
	itr1WithEndKey, _ := newCombinedIterator(db, batch1, "ns", "key2", "key8", true)
	defer itr1WithEndKey.Close()
	checkItrResults(t, "IncludeEndKey", itr1WithEndKey, []*statedb.VersionedKV{
		constructVersionedKV("ns", "key3", []byte("value3"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key4", []byte("value4"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key6", []byte("value6_new"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key7", []byte("value7"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key8", []byte("value8"), version.NewHeight(1, 1)),
	})

//为额外范围测试db+batch1更新（包括endkey）
	itr1WithEndKeyExtraRange, _ := newCombinedIterator(db, batch1, "ns", "key0", "key9", true)
	defer itr1WithEndKeyExtraRange.Close()
	checkItrResults(t, "IncludeEndKey_ExtraRange", itr1WithEndKeyExtraRange, []*statedb.VersionedKV{
		constructVersionedKV("ns", "key1", []byte("value1"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key3", []byte("value3"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key4", []byte("value4"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key6", []byte("value6_new"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key7", []byte("value7"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key8", []byte("value8"), version.NewHeight(1, 1)),
	})

//使用全范围查询测试数据库+批处理1更新
	itr3, _ := newCombinedIterator(db, batch1, "ns", "", "", false)
	defer itr3.Close()
	checkItrResults(t, "ExcludeEndKey_FullRange", itr3, []*statedb.VersionedKV{
		constructVersionedKV("ns", "key1", []byte("value1"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key3", []byte("value3"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key4", []byte("value4"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key6", []byte("value6_new"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key7", []byte("value7"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key8", []byte("value8"), version.NewHeight(1, 1)),
	})

//测试数据库+第2批更新
	itr2, _ := newCombinedIterator(db, batch2, "ns", "key2", "key8", false)
	defer itr2.Close()
	checkItrResults(t, "ExcludeEndKey_EmptyUpdates", itr2, []*statedb.VersionedKV{
		constructVersionedKV("ns", "key4", []byte("value4"), version.NewHeight(1, 1)),
		constructVersionedKV("ns", "key6", []byte("value6"), version.NewHeight(1, 1)),
	})
}

func checkItrResults(t *testing.T, testName string, itr statedb.ResultsIterator, expectedResults []*statedb.VersionedKV) {
	t.Run(testName, func(t *testing.T) {
		for i := 0; i < len(expectedResults); i++ {
			res, _ := itr.Next()
			assert.Equal(t, expectedResults[i], res)
		}
		lastRes, err := itr.Next()
		assert.NoError(t, err)
		assert.Nil(t, lastRes)
	})
}

func constructVersionedKV(ns string, key string, value []byte, version *version.Height) *statedb.VersionedKV {
	return &statedb.VersionedKV{
		CompositeKey:   statedb.CompositeKey{Namespace: ns, Key: key},
		VersionedValue: statedb.VersionedValue{Value: value, Version: version}}
}
