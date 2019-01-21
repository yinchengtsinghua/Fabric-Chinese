
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


package stateleveldb

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/commontests"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/txmgmt/statedb/stateleveldb")
	os.Exit(m.Run())
}

func TestBasicRW(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestBasicRW(t, env.DBProvider)
}

func TestMultiDBBasicRW(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestMultiDBBasicRW(t, env.DBProvider)
}

func TestDeletes(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestDeletes(t, env.DBProvider)
}

func TestIterator(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestIterator(t, env.DBProvider)
}

func TestCompositeKey(t *testing.T) {
	testCompositeKey(t, "ledger1", "ns", "key")
	testCompositeKey(t, "ledger2", "ns", "")
}

func testCompositeKey(t *testing.T, dbName string, ns string, key string) {
	compositeKey := constructCompositeKey(ns, key)
	t.Logf("compositeKey=%#v", compositeKey)
	ns1, key1 := splitCompositeKey(compositeKey)
	assert.Equal(t, ns, ns1)
	assert.Equal(t, key, key1)
}

//testqueryonleveldb在leveldb上测试查询。
func TestQueryOnLevelDB(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	db, err := env.DBProvider.GetDBHandle("testquery")
	assert.NoError(t, err)
	db.Open()
	defer db.Close()
	batch := statedb.NewUpdateBatch()
	jsonValue1 := `{"asset_name": "marble1","color": "blue","size": 1,"owner": "tom"}`
	batch.Put("ns1", "key1", []byte(jsonValue1), version.NewHeight(1, 1))

	savePoint := version.NewHeight(2, 22)
	db.ApplyUpdates(batch, savePoint)

//查询owner=jerry，使用名称空间“ns1”
//由于LevelDB不支持查询，因此调用ExecuteQuery（）。
//应返回错误消息
	itr, err := db.ExecuteQuery("ns1", `{"selector":{"owner":"jerry"}}`)
	assert.Error(t, err, "ExecuteQuery not supported for leveldb")
	assert.Nil(t, itr)
}

func TestGetStateMultipleKeys(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestGetStateMultipleKeys(t, env.DBProvider)
}

func TestGetVersion(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestGetVersion(t, env.DBProvider)
}

func TestUtilityFunctions(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()

	db, err := env.DBProvider.GetDBHandle("testutilityfunctions")
	assert.NoError(t, err)

//支持BythKEY的GeleVeldB应该是真的
	byteKeySupported := db.BytesKeySupported()
	assert.True(t, byteKeySupported)

//对于有效的键和值，validateKeyValue应返回nil
	assert.NoError(t, db.ValidateKeyValue("testKey", []byte("testValue")), "leveldb should accept all key-values")
}

func TestValueAndMetadataWrites(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestValueAndMetadataWrites(t, env.DBProvider)
}

func TestPaginatedRangeQuery(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestPaginatedRangeQuery(t, env.DBProvider)
}

func TestApplyUpdatesWithNilHeight(t *testing.T) {
	env := NewTestVDBEnv(t)
	defer env.Cleanup()
	commontests.TestApplyUpdatesWithNilHeight(t, env.DBProvider)
}
