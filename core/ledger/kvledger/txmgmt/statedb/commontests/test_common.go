
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


package commontests

import (
	"strings"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

//testgetstatemultiplekeys测试读取给定的多个键
func TestGetStateMultipleKeys(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testgetmultiplekeys")
	assert.NoError(t, err)

//测试新状态数据库的保存点是否为零
	sp, err := db.GetLatestSavePoint()
	assert.NoError(t, err, "Error upon GetLatestSavePoint()")
	assert.Nil(t, sp)

	batch := statedb.NewUpdateBatch()
	expectedValues := make([]*statedb.VersionedValue, 2)
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	expectedValues[0] = &vv1
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	expectedValues[1] = &vv2
	vv3 := statedb.VersionedValue{Value: []byte("value3"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte{}, Version: version.NewHeight(1, 4)}
	batch.Put("ns1", "key1", vv1.Value, vv1.Version)
	batch.Put("ns1", "key2", vv2.Value, vv2.Version)
	batch.Put("ns2", "key3", vv3.Value, vv3.Version)
	batch.Put("ns2", "key4", vv4.Value, vv4.Version)
	savePoint := version.NewHeight(2, 5)
	db.ApplyUpdates(batch, savePoint)

	actualValues, _ := db.GetStateMultipleKeys("ns1", []string{"key1", "key2"})
	assert.Equal(t, expectedValues, actualValues)
}

//测试基本CRW测试基本读写
func TestBasicRW(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testbasicrw")
	assert.NoError(t, err)

//测试新状态数据库的保存点是否为零
	sp, err := db.GetLatestSavePoint()
	assert.NoError(t, err, "Error upon GetLatestSavePoint()")
	assert.Nil(t, sp)

//测试检索不存在的密钥-返回零而不是错误
//有关更多详细信息，请参阅https://github.com/hyperledger-archives/fabric/issues/936。
	val, err := db.GetState("ns", "key1")
	assert.NoError(t, err, "Should receive nil rather than error upon reading non existent key")
	assert.Nil(t, val)

	batch := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	vv3 := statedb.VersionedValue{Value: []byte("value3"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte{}, Version: version.NewHeight(1, 4)}
	vv5 := statedb.VersionedValue{Value: []byte("null"), Version: version.NewHeight(1, 5)}
	batch.Put("ns1", "key1", vv1.Value, vv1.Version)
	batch.Put("ns1", "key2", vv2.Value, vv2.Version)
	batch.Put("ns2", "key3", vv3.Value, vv3.Version)
	batch.Put("ns2", "key4", vv4.Value, vv4.Version)
	batch.Put("ns2", "key5", vv5.Value, vv5.Version)
	savePoint := version.NewHeight(2, 5)
	db.ApplyUpdates(batch, savePoint)

	vv, _ := db.GetState("ns1", "key1")
	assert.Equal(t, &vv1, vv)

	vv, _ = db.GetState("ns2", "key4")
	assert.Equal(t, &vv4, vv)

	vv, _ = db.GetState("ns2", "key5")
	assert.Equal(t, &vv5, vv)

	sp, err = db.GetLatestSavePoint()
	assert.NoError(t, err)
	assert.Equal(t, savePoint, sp)

}

//testmultidbbasicrw在多个dbs上测试基本读写
func TestMultiDBBasicRW(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db1, err := dbProvider.GetDBHandle("testmultidbbasicrw")
	assert.NoError(t, err)

	db2, err := dbProvider.GetDBHandle("testmultidbbasicrw2")
	assert.NoError(t, err)

	batch1 := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1_db1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2_db1"), Version: version.NewHeight(1, 2)}
	batch1.Put("ns1", "key1", vv1.Value, vv1.Version)
	batch1.Put("ns1", "key2", vv2.Value, vv2.Version)
	savePoint1 := version.NewHeight(1, 2)
	db1.ApplyUpdates(batch1, savePoint1)

	batch2 := statedb.NewUpdateBatch()
	vv3 := statedb.VersionedValue{Value: []byte("value1_db2"), Version: version.NewHeight(1, 4)}
	vv4 := statedb.VersionedValue{Value: []byte("value2_db2"), Version: version.NewHeight(1, 5)}
	batch2.Put("ns1", "key1", vv3.Value, vv3.Version)
	batch2.Put("ns1", "key2", vv4.Value, vv4.Version)
	savePoint2 := version.NewHeight(1, 5)
	db2.ApplyUpdates(batch2, savePoint2)

	vv, _ := db1.GetState("ns1", "key1")
	assert.Equal(t, &vv1, vv)

	sp, err := db1.GetLatestSavePoint()
	assert.NoError(t, err)
	assert.Equal(t, savePoint1, sp)

	vv, _ = db2.GetState("ns1", "key1")
	assert.Equal(t, &vv3, vv)

	sp, err = db2.GetLatestSavePoint()
	assert.NoError(t, err)
	assert.Equal(t, savePoint2, sp)
}

//测试删除测试删除
func TestDeletes(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testdeletes")
	assert.NoError(t, err)

	batch := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	vv3 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 4)}

	batch.Put("ns", "key1", vv1.Value, vv1.Version)
	batch.Put("ns", "key2", vv2.Value, vv2.Version)
	batch.Put("ns", "key3", vv2.Value, vv3.Version)
	batch.Put("ns", "key4", vv2.Value, vv4.Version)
	batch.Delete("ns", "key3", version.NewHeight(1, 5))
	savePoint := version.NewHeight(1, 5)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)
	vv, _ := db.GetState("ns", "key2")
	assert.Equal(t, &vv2, vv)

	vv, err = db.GetState("ns", "key3")
	assert.NoError(t, err)
	assert.Nil(t, vv)

	batch = statedb.NewUpdateBatch()
	batch.Delete("ns", "key2", version.NewHeight(1, 6))
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)
	vv, err = db.GetState("ns", "key2")
	assert.NoError(t, err)
	assert.Nil(t, vv)
}

//测试迭代器
func TestIterator(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testiterator")
	assert.NoError(t, err)
	db.Open()
	defer db.Close()
	batch := statedb.NewUpdateBatch()
	batch.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 1))
	batch.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 2))
	batch.Put("ns1", "key3", []byte("value3"), version.NewHeight(1, 3))
	batch.Put("ns1", "key4", []byte("value4"), version.NewHeight(1, 4))
	batch.Put("ns2", "key5", []byte("value5"), version.NewHeight(1, 5))
	batch.Put("ns2", "key6", []byte("value6"), version.NewHeight(1, 6))
	batch.Put("ns3", "key7", []byte("value7"), version.NewHeight(1, 7))
	savePoint := version.NewHeight(2, 5)
	db.ApplyUpdates(batch, savePoint)
	itr1, _ := db.GetStateRangeScanIterator("ns1", "key1", "")
	testItr(t, itr1, []string{"key1", "key2", "key3", "key4"})

	itr2, _ := db.GetStateRangeScanIterator("ns1", "key2", "key3")
	testItr(t, itr2, []string{"key2"})

	itr3, _ := db.GetStateRangeScanIterator("ns1", "", "")
	testItr(t, itr3, []string{"key1", "key2", "key3", "key4"})

	itr4, _ := db.GetStateRangeScanIterator("ns2", "", "")
	testItr(t, itr4, []string{"key5", "key6"})

}

func testItr(t *testing.T, itr statedb.ResultsIterator, expectedKeys []string) {
	defer itr.Close()
	for _, expectedKey := range expectedKeys {
		queryResult, _ := itr.Next()
		vkv := queryResult.(*statedb.VersionedKV)
		key := vkv.Key
		assert.Equal(t, expectedKey, key)
	}
	_, err := itr.Next()
	assert.NoError(t, err)
}

//测试查询测试查询
func TestQuery(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testquery")
	assert.NoError(t, err)
	db.Open()
	defer db.Close()
	batch := statedb.NewUpdateBatch()
	jsonValue1 := `{"asset_name": "marble1","color": "blue","size": 1,"owner": "tom"}`
	batch.Put("ns1", "key1", []byte(jsonValue1), version.NewHeight(1, 1))
	jsonValue2 := `{"asset_name": "marble2","color": "blue","size": 2,"owner": "jerry"}`
	batch.Put("ns1", "key2", []byte(jsonValue2), version.NewHeight(1, 2))
	jsonValue3 := `{"asset_name": "marble3","color": "blue","size": 3,"owner": "fred"}`
	batch.Put("ns1", "key3", []byte(jsonValue3), version.NewHeight(1, 3))
	jsonValue4 := `{"asset_name": "marble4","color": "blue","size": 4,"owner": "martha"}`
	batch.Put("ns1", "key4", []byte(jsonValue4), version.NewHeight(1, 4))
	jsonValue5 := `{"asset_name": "marble5","color": "blue","size": 5,"owner": "fred"}`
	batch.Put("ns1", "key5", []byte(jsonValue5), version.NewHeight(1, 5))
	jsonValue6 := `{"asset_name": "marble6","color": "blue","size": 6,"owner": "elaine"}`
	batch.Put("ns1", "key6", []byte(jsonValue6), version.NewHeight(1, 6))
	jsonValue7 := `{"asset_name": "marble7","color": "blue","size": 7,"owner": "fred"}`
	batch.Put("ns1", "key7", []byte(jsonValue7), version.NewHeight(1, 7))
	jsonValue8 := `{"asset_name": "marble8","color": "blue","size": 8,"owner": "elaine"}`
	batch.Put("ns1", "key8", []byte(jsonValue8), version.NewHeight(1, 8))
	jsonValue9 := `{"asset_name": "marble9","color": "green","size": 9,"owner": "fred"}`
	batch.Put("ns1", "key9", []byte(jsonValue9), version.NewHeight(1, 9))
	jsonValue10 := `{"asset_name": "marble10","color": "green","size": 10,"owner": "mary"}`
	batch.Put("ns1", "key10", []byte(jsonValue10), version.NewHeight(1, 10))
	jsonValue11 := `{"asset_name": "marble11","color": "cyan","size": 1000007,"owner": "joe"}`
	batch.Put("ns1", "key11", []byte(jsonValue11), version.NewHeight(1, 11))

//为单独的命名空间添加键
	batch.Put("ns2", "key1", []byte(jsonValue1), version.NewHeight(1, 12))
	batch.Put("ns2", "key2", []byte(jsonValue2), version.NewHeight(1, 13))
	batch.Put("ns2", "key3", []byte(jsonValue3), version.NewHeight(1, 14))
	batch.Put("ns2", "key4", []byte(jsonValue4), version.NewHeight(1, 15))
	batch.Put("ns2", "key5", []byte(jsonValue5), version.NewHeight(1, 16))
	batch.Put("ns2", "key6", []byte(jsonValue6), version.NewHeight(1, 17))
	batch.Put("ns2", "key7", []byte(jsonValue7), version.NewHeight(1, 18))
	batch.Put("ns2", "key8", []byte(jsonValue8), version.NewHeight(1, 19))
	batch.Put("ns2", "key9", []byte(jsonValue9), version.NewHeight(1, 20))
	batch.Put("ns2", "key10", []byte(jsonValue10), version.NewHeight(1, 21))

	savePoint := version.NewHeight(2, 22)
	db.ApplyUpdates(batch, savePoint)

//查询owner=jerry，使用名称空间“ns1”
	itr, err := db.ExecuteQuery("ns1", `{"selector":{"owner":"jerry"}}`)
	assert.NoError(t, err)

//验证一个Jerry结果
	queryResult1, err := itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord := queryResult1.(*statedb.VersionedKV)
	stringRecord := string(versionedQueryRecord.Value)
	bFoundRecord := strings.Contains(stringRecord, "jerry")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult2, err := itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult2)

//查询owner=jerry，使用名称空间“ns2”
	itr, err = db.ExecuteQuery("ns2", `{"selector":{"owner":"jerry"}}`)
	assert.NoError(t, err)

//验证一个Jerry结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "jerry")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult2)

//查询owner=jerry，使用名称空间“ns3”
	itr, err = db.ExecuteQuery("ns3", `{"selector":{"owner":"jerry"}}`)
	assert.NoError(t, err)

//验证结果-应无记录
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult1)

//使用错误的查询字符串进行查询
	itr, err = db.ExecuteQuery("ns1", "this is an invalid query string")
	assert.Error(t, err, "Should have received an error for invalid query string")

//查询返回0条记录
	itr, err = db.ExecuteQuery("ns1", `{"selector":{"owner":"not_a_valid_name"}}`)
	assert.NoError(t, err)

//验证无结果
	queryResult3, err := itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult3)

//带字段的查询，命名空间“ns1”
	itr, err = db.ExecuteQuery("ns1", `{"selector":{"owner":"jerry"},"fields": ["owner", "asset_name", "color", "size"]}`)
	assert.NoError(t, err)

//验证一个Jerry结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "jerry")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult2)

//带字段的查询，命名空间“ns2”
	itr, err = db.ExecuteQuery("ns2", `{"selector":{"owner":"jerry"},"fields": ["owner", "asset_name", "color", "size"]}`)
	assert.NoError(t, err)

//验证一个Jerry结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "jerry")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult2)

//带字段的查询，命名空间“ns3”
	itr, err = db.ExecuteQuery("ns3", `{"selector":{"owner":"jerry"},"fields": ["owner", "asset_name", "color", "size"]}`)
	assert.NoError(t, err)

//验证无结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult1)

//带有复杂选择器的查询，命名空间“ns1”
	itr, err = db.ExecuteQuery("ns1", `{"selector":{"$and":[{"size":{"$gt": 5}},{"size":{"$lt":8}},{"$not":{"size":6}}]}}`)
	assert.NoError(t, err)

//验证一个FRED结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "fred")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult2)

//带有复杂选择器的查询，命名空间“ns2”
	itr, err = db.ExecuteQuery("ns2", `{"selector":{"$and":[{"size":{"$gt": 5}},{"size":{"$lt":8}},{"$not":{"size":6}}]}}`)
	assert.NoError(t, err)

//验证一个FRED结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "fred")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult2)

//带有复杂选择器的查询，命名空间“ns3”
	itr, err = db.ExecuteQuery("ns3", `{"selector":{"$and":[{"size":{"$gt": 5}},{"size":{"$lt":8}},{"$not":{"size":6}}]}}`)
	assert.NoError(t, err)

//不再验证结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult1)

//带嵌入隐式“和”和显式“或”的查询，命名空间为“ns1”
	itr, err = db.ExecuteQuery("ns1", `{"selector":{"color":"green","$or":[{"owner":"fred"},{"owner":"mary"}]}}`)
	assert.NoError(t, err)

//验证一个绿色结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "green")
	assert.True(t, bFoundRecord)

//验证另一个绿色结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult2)
	versionedQueryRecord = queryResult2.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "green")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult3, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult3)

//使用嵌入的隐式“and”和显式“or”查询，命名空间为“ns2”
	itr, err = db.ExecuteQuery("ns2", `{"selector":{"color":"green","$or":[{"owner":"fred"},{"owner":"mary"}]}}`)
	assert.NoError(t, err)

//验证一个绿色结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "green")
	assert.True(t, bFoundRecord)

//验证另一个绿色结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult2)
	versionedQueryRecord = queryResult2.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "green")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult3, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult3)

//使用嵌入的隐式“and”和显式“or”查询，命名空间为“ns3”
	itr, err = db.ExecuteQuery("ns3", `{"selector":{"color":"green","$or":[{"owner":"fred"},{"owner":"mary"}]}}`)
	assert.NoError(t, err)

//验证无结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult1)

//数字计数为7的整数查询，并接收到响应
//数字计数相同且不存在浮点转换
	itr, err = db.ExecuteQuery("ns1", `{"selector":{"$and":[{"size":{"$eq": 1000007}}]}}`)
	assert.NoError(t, err)

//验证一个Jerry结果
	queryResult1, err = itr.Next()
	assert.NoError(t, err)
	assert.NotNil(t, queryResult1)
	versionedQueryRecord = queryResult1.(*statedb.VersionedKV)
	stringRecord = string(versionedQueryRecord.Value)
	bFoundRecord = strings.Contains(stringRecord, "joe")
	assert.True(t, bFoundRecord)
	bFoundRecord = strings.Contains(stringRecord, "1000007")
	assert.True(t, bFoundRecord)

//不再验证结果
	queryResult2, err = itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, queryResult2)

}

//TestGetVersion测试按命名空间和键检索版本
func TestGetVersion(t *testing.T, dbProvider statedb.VersionedDBProvider) {

	db, err := dbProvider.GetDBHandle("testgetversion")
	assert.NoError(t, err)

	batch := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	vv3 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 4)}

	batch.Put("ns", "key1", vv1.Value, vv1.Version)
	batch.Put("ns", "key2", vv2.Value, vv2.Version)
	batch.Put("ns", "key3", vv2.Value, vv3.Version)
	batch.Put("ns", "key4", vv2.Value, vv4.Version)
	savePoint := version.NewHeight(1, 5)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//检查是否支持批量优化接口（couchdb）
	if bulkdb, ok := db.(statedb.BulkOptimizable); ok {
//清除缓存的版本，这将在调用GetVerion时强制读取
		bulkdb.ClearCachedVersions()
	}

//按命名空间和键检索版本
	resp, err := db.GetVersion("ns", "key2")
	assert.NoError(t, err)
	assert.Equal(t, version.NewHeight(1, 2), resp)

//尝试检索不存在的命名空间和键
	resp, err = db.GetVersion("ns2", "key2")
	assert.NoError(t, err)
	assert.Nil(t, resp)

//检查是否支持批量优化接口（couchdb）
	if bulkdb, ok := db.(statedb.BulkOptimizable); ok {

//清除缓存的版本，这将在调用GetVerion时强制读取
		bulkdb.ClearCachedVersions()

//初始化密钥列表
		loadKeys := []*statedb.CompositeKey{}
//创建复合键并添加到键列表
		compositeKey := statedb.CompositeKey{Namespace: "ns", Key: "key3"}
		loadKeys = append(loadKeys, &compositeKey)
//加载提交的版本
		bulkdb.LoadCommittedVersions(loadKeys)

//按命名空间和键检索版本
		resp, err := db.GetVersion("ns", "key3")
		assert.NoError(t, err)
		assert.Equal(t, version.NewHeight(1, 3), resp)

	}
}

//testsmallbatchsize测试多个更新批
func TestSmallBatchSize(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testsmallbatchsize")
	assert.NoError(t, err)
	db.Open()
	defer db.Close()
	batch := statedb.NewUpdateBatch()
	jsonValue1 := []byte(`{"asset_name": "marble1","color": "blue","size": 1,"owner": "tom"}`)
	batch.Put("ns1", "key1", jsonValue1, version.NewHeight(1, 1))
	jsonValue2 := []byte(`{"asset_name": "marble2","color": "blue","size": 2,"owner": "jerry"}`)
	batch.Put("ns1", "key2", jsonValue2, version.NewHeight(1, 2))
	jsonValue3 := []byte(`{"asset_name": "marble3","color": "blue","size": 3,"owner": "fred"}`)
	batch.Put("ns1", "key3", jsonValue3, version.NewHeight(1, 3))
	jsonValue4 := []byte(`{"asset_name": "marble4","color": "blue","size": 4,"owner": "martha"}`)
	batch.Put("ns1", "key4", jsonValue4, version.NewHeight(1, 4))
	jsonValue5 := []byte(`{"asset_name": "marble5","color": "blue","size": 5,"owner": "fred"}`)
	batch.Put("ns1", "key5", jsonValue5, version.NewHeight(1, 5))
	jsonValue6 := []byte(`{"asset_name": "marble6","color": "blue","size": 6,"owner": "elaine"}`)
	batch.Put("ns1", "key6", jsonValue6, version.NewHeight(1, 6))
	jsonValue7 := []byte(`{"asset_name": "marble7","color": "blue","size": 7,"owner": "fred"}`)
	batch.Put("ns1", "key7", jsonValue7, version.NewHeight(1, 7))
	jsonValue8 := []byte(`{"asset_name": "marble8","color": "blue","size": 8,"owner": "elaine"}`)
	batch.Put("ns1", "key8", jsonValue8, version.NewHeight(1, 8))
	jsonValue9 := []byte(`{"asset_name": "marble9","color": "green","size": 9,"owner": "fred"}`)
	batch.Put("ns1", "key9", jsonValue9, version.NewHeight(1, 9))
	jsonValue10 := []byte(`{"asset_name": "marble10","color": "green","size": 10,"owner": "mary"}`)
	batch.Put("ns1", "key10", jsonValue10, version.NewHeight(1, 10))
	jsonValue11 := []byte(`{"asset_name": "marble11","color": "cyan","size": 1000007,"owner": "joe"}`)
	batch.Put("ns1", "key11", jsonValue11, version.NewHeight(1, 11))

	savePoint := version.NewHeight(1, 12)
	db.ApplyUpdates(batch, savePoint)

//验证是否添加了所有大理石

	vv, _ := db.GetState("ns1", "key1")
	assert.JSONEq(t, string(jsonValue1), string(vv.Value))

	vv, _ = db.GetState("ns1", "key2")
	assert.JSONEq(t, string(jsonValue2), string(vv.Value))

	vv, _ = db.GetState("ns1", "key3")
	assert.JSONEq(t, string(jsonValue3), string(vv.Value))

	vv, _ = db.GetState("ns1", "key4")
	assert.JSONEq(t, string(jsonValue4), string(vv.Value))

	vv, _ = db.GetState("ns1", "key5")
	assert.JSONEq(t, string(jsonValue5), string(vv.Value))

	vv, _ = db.GetState("ns1", "key6")
	assert.JSONEq(t, string(jsonValue6), string(vv.Value))

	vv, _ = db.GetState("ns1", "key7")
	assert.JSONEq(t, string(jsonValue7), string(vv.Value))

	vv, _ = db.GetState("ns1", "key8")
	assert.JSONEq(t, string(jsonValue8), string(vv.Value))

	vv, _ = db.GetState("ns1", "key9")
	assert.JSONEq(t, string(jsonValue9), string(vv.Value))

	vv, _ = db.GetState("ns1", "key10")
	assert.JSONEq(t, string(jsonValue10), string(vv.Value))

	vv, _ = db.GetState("ns1", "key11")
	assert.JSONEq(t, string(jsonValue11), string(vv.Value))
}

//testbatchWithIndividualRetry测试批中的单个故障
func TestBatchWithIndividualRetry(t *testing.T, dbProvider statedb.VersionedDBProvider) {

	db, err := dbProvider.GetDBHandle("testbatchretry")
	assert.NoError(t, err)

	batch := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	vv3 := statedb.VersionedValue{Value: []byte("value3"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte("value4"), Version: version.NewHeight(1, 4)}

	batch.Put("ns", "key1", vv1.Value, vv1.Version)
	batch.Put("ns", "key2", vv2.Value, vv2.Version)
	batch.Put("ns", "key3", vv3.Value, vv3.Version)
	batch.Put("ns", "key4", vv4.Value, vv4.Version)
	savePoint := version.NewHeight(1, 5)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//清除下一批的缓存，而不是模拟
	if bulkdb, ok := db.(statedb.BulkOptimizable); ok {
//清除缓存的版本，这将在调用GetVerion时强制读取
		bulkdb.ClearCachedVersions()
	}

	batch = statedb.NewUpdateBatch()
	batch.Put("ns", "key1", vv1.Value, vv1.Version)
	batch.Put("ns", "key2", vv2.Value, vv2.Version)
	batch.Put("ns", "key3", vv3.Value, vv3.Version)
	batch.Put("ns", "key4", vv4.Value, vv4.Version)
	savePoint = version.NewHeight(1, 6)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//更新文档键3
	batch = statedb.NewUpdateBatch()
	batch.Delete("ns", "key2", vv2.Version)
	batch.Put("ns", "key3", vv3.Value, vv3.Version)
	savePoint = version.NewHeight(1, 7)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//这将强制对删除和更新的couchdb修订冲突重试
//重试逻辑应更正更新并防止删除引发错误。
	batch = statedb.NewUpdateBatch()
	batch.Delete("ns", "key2", vv2.Version)
	batch.Put("ns", "key3", vv3.Value, vv3.Version)
	savePoint = version.NewHeight(1, 8)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//创建一组使用JSON而不是二进制的新值
	jsonValue5 := []byte(`{"asset_name": "marble5","color": "blue","size": 5,"owner": "fred"}`)
	jsonValue6 := []byte(`{"asset_name": "marble6","color": "blue","size": 6,"owner": "elaine"}`)
	jsonValue7 := []byte(`{"asset_name": "marble7","color": "blue","size": 7,"owner": "fred"}`)
	jsonValue8 := []byte(`{"asset_name": "marble8","color": "blue","size": 8,"owner": "elaine"}`)

//清除下一批的缓存，而不是模拟
	if bulkdb, ok := db.(statedb.BulkOptimizable); ok {
//清除缓存的版本，这将在调用GetVersion时强制读取
		bulkdb.ClearCachedVersions()
	}

	batch = statedb.NewUpdateBatch()
	batch.Put("ns1", "key5", jsonValue5, version.NewHeight(1, 9))
	batch.Put("ns1", "key6", jsonValue6, version.NewHeight(1, 10))
	batch.Put("ns1", "key7", jsonValue7, version.NewHeight(1, 11))
	batch.Put("ns1", "key8", jsonValue8, version.NewHeight(1, 12))
	savePoint = version.NewHeight(1, 6)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//清除下一批的缓存，而不是模拟
	if bulkdb, ok := db.(statedb.BulkOptimizable); ok {
//清除缓存的版本，这将在调用GetVersion时强制读取
		bulkdb.ClearCachedVersions()
	}

//再次发送批处理以测试更新
	batch = statedb.NewUpdateBatch()
	batch.Put("ns1", "key5", jsonValue5, version.NewHeight(1, 9))
	batch.Put("ns1", "key6", jsonValue6, version.NewHeight(1, 10))
	batch.Put("ns1", "key7", jsonValue7, version.NewHeight(1, 11))
	batch.Put("ns1", "key8", jsonValue8, version.NewHeight(1, 12))
	savePoint = version.NewHeight(1, 6)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//更新文档键3
//这将导致连接DB2的缓存条目不一致
	batch = statedb.NewUpdateBatch()
	batch.Delete("ns1", "key6", version.NewHeight(1, 13))
	batch.Put("ns1", "key7", jsonValue7, version.NewHeight(1, 14))
	savePoint = version.NewHeight(1, 15)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

//这将强制对删除和更新的couchdb修订冲突重试
//重试逻辑应更正更新并防止删除引发错误。
	batch = statedb.NewUpdateBatch()
	batch.Delete("ns1", "key6", version.NewHeight(1, 16))
	batch.Put("ns1", "key7", jsonValue7, version.NewHeight(1, 17))
	savePoint = version.NewHeight(1, 18)
	err = db.ApplyUpdates(batch, savePoint)
	assert.NoError(t, err)

}

//测试值和元数据写入测试StateDB的值和元数据读取写入
func TestValueAndMetadataWrites(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testvalueandmetadata")
	assert.NoError(t, err)
	batch := statedb.NewUpdateBatch()

	vv1 := statedb.VersionedValue{Value: []byte("value1"), Metadata: []byte("metadata1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Metadata: []byte("metadata2"), Version: version.NewHeight(1, 2)}
	vv3 := statedb.VersionedValue{Value: []byte("value3"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte{}, Metadata: []byte("metadata4"), Version: version.NewHeight(1, 4)}

	batch.PutValAndMetadata("ns1", "key1", vv1.Value, vv1.Metadata, vv1.Version)
	batch.PutValAndMetadata("ns1", "key2", vv2.Value, vv2.Metadata, vv2.Version)
	batch.PutValAndMetadata("ns2", "key3", vv3.Value, vv3.Metadata, vv3.Version)
	batch.PutValAndMetadata("ns2", "key4", vv4.Value, vv4.Metadata, vv4.Version)
	db.ApplyUpdates(batch, version.NewHeight(2, 5))

	vv, _ := db.GetState("ns1", "key1")
	assert.Equal(t, &vv1, vv)

	vv, _ = db.GetState("ns1", "key2")
	assert.Equal(t, &vv2, vv)

	vv, _ = db.GetState("ns2", "key3")
	assert.Equal(t, &vv3, vv)

	vv, _ = db.GetState("ns2", "key4")
	assert.Equal(t, &vv4, vv)
}

//testpaginatedrangequery测试带有分页的范围查询
func TestPaginatedRangeQuery(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("testpaginatedrangequery")
	assert.NoError(t, err)
	db.Open()
	defer db.Close()
	batch := statedb.NewUpdateBatch()
	jsonValue1 := `{"asset_name": "marble1","color": "blue","size": 1,"owner": "tom"}`
	batch.Put("ns1", "key1", []byte(jsonValue1), version.NewHeight(1, 1))
	jsonValue2 := `{"asset_name": "marble2","color": "red","size": 2,"owner": "jerry"}`
	batch.Put("ns1", "key2", []byte(jsonValue2), version.NewHeight(1, 2))
	jsonValue3 := `{"asset_name": "marble3","color": "red","size": 3,"owner": "fred"}`
	batch.Put("ns1", "key3", []byte(jsonValue3), version.NewHeight(1, 3))
	jsonValue4 := `{"asset_name": "marble4","color": "red","size": 4,"owner": "martha"}`
	batch.Put("ns1", "key4", []byte(jsonValue4), version.NewHeight(1, 4))
	jsonValue5 := `{"asset_name": "marble5","color": "blue","size": 5,"owner": "fred"}`
	batch.Put("ns1", "key5", []byte(jsonValue5), version.NewHeight(1, 5))
	jsonValue6 := `{"asset_name": "marble6","color": "red","size": 6,"owner": "elaine"}`
	batch.Put("ns1", "key6", []byte(jsonValue6), version.NewHeight(1, 6))
	jsonValue7 := `{"asset_name": "marble7","color": "blue","size": 7,"owner": "fred"}`
	batch.Put("ns1", "key7", []byte(jsonValue7), version.NewHeight(1, 7))
	jsonValue8 := `{"asset_name": "marble8","color": "red","size": 8,"owner": "elaine"}`
	batch.Put("ns1", "key8", []byte(jsonValue8), version.NewHeight(1, 8))
	jsonValue9 := `{"asset_name": "marble9","color": "green","size": 9,"owner": "fred"}`
	batch.Put("ns1", "key9", []byte(jsonValue9), version.NewHeight(1, 9))
	jsonValue10 := `{"asset_name": "marble10","color": "green","size": 10,"owner": "mary"}`
	batch.Put("ns1", "key10", []byte(jsonValue10), version.NewHeight(1, 10))

	jsonValue11 := `{"asset_name": "marble11","color": "cyan","size": 8,"owner": "joe"}`
	batch.Put("ns1", "key11", []byte(jsonValue11), version.NewHeight(1, 11))
	jsonValue12 := `{"asset_name": "marble12","color": "red","size": 4,"owner": "martha"}`
	batch.Put("ns1", "key12", []byte(jsonValue12), version.NewHeight(1, 4))
	jsonValue13 := `{"asset_name": "marble13","color": "red","size": 6,"owner": "james"}`
	batch.Put("ns1", "key13", []byte(jsonValue13), version.NewHeight(1, 4))
	jsonValue14 := `{"asset_name": "marble14","color": "red","size": 10,"owner": "fred"}`
	batch.Put("ns1", "key14", []byte(jsonValue14), version.NewHeight(1, 4))
	jsonValue15 := `{"asset_name": "marble15","color": "red","size": 8,"owner": "mary"}`
	batch.Put("ns1", "key15", []byte(jsonValue15), version.NewHeight(1, 4))
	jsonValue16 := `{"asset_name": "marble16","color": "red","size": 4,"owner": "robert"}`
	batch.Put("ns1", "key16", []byte(jsonValue16), version.NewHeight(1, 4))
	jsonValue17 := `{"asset_name": "marble17","color": "red","size": 2,"owner": "alan"}`
	batch.Put("ns1", "key17", []byte(jsonValue17), version.NewHeight(1, 4))
	jsonValue18 := `{"asset_name": "marble18","color": "red","size": 10,"owner": "elaine"}`
	batch.Put("ns1", "key18", []byte(jsonValue18), version.NewHeight(1, 4))
	jsonValue19 := `{"asset_name": "marble19","color": "red","size": 2,"owner": "alan"}`
	batch.Put("ns1", "key19", []byte(jsonValue19), version.NewHeight(1, 4))
	jsonValue20 := `{"asset_name": "marble20","color": "red","size": 10,"owner": "elaine"}`
	batch.Put("ns1", "key20", []byte(jsonValue20), version.NewHeight(1, 4))

	jsonValue21 := `{"asset_name": "marble21","color": "cyan","size": 1000007,"owner": "joe"}`
	batch.Put("ns1", "key21", []byte(jsonValue21), version.NewHeight(1, 11))
	jsonValue22 := `{"asset_name": "marble22","color": "red","size": 4,"owner": "martha"}`
	batch.Put("ns1", "key22", []byte(jsonValue22), version.NewHeight(1, 4))
	jsonValue23 := `{"asset_name": "marble23","color": "blue","size": 6,"owner": "james"}`
	batch.Put("ns1", "key23", []byte(jsonValue23), version.NewHeight(1, 4))
	jsonValue24 := `{"asset_name": "marble24","color": "red","size": 10,"owner": "fred"}`
	batch.Put("ns1", "key24", []byte(jsonValue24), version.NewHeight(1, 4))
	jsonValue25 := `{"asset_name": "marble25","color": "red","size": 8,"owner": "mary"}`
	batch.Put("ns1", "key25", []byte(jsonValue25), version.NewHeight(1, 4))
	jsonValue26 := `{"asset_name": "marble26","color": "red","size": 4,"owner": "robert"}`
	batch.Put("ns1", "key26", []byte(jsonValue26), version.NewHeight(1, 4))
	jsonValue27 := `{"asset_name": "marble27","color": "green","size": 2,"owner": "alan"}`
	batch.Put("ns1", "key27", []byte(jsonValue27), version.NewHeight(1, 4))
	jsonValue28 := `{"asset_name": "marble28","color": "red","size": 10,"owner": "elaine"}`
	batch.Put("ns1", "key28", []byte(jsonValue28), version.NewHeight(1, 4))
	jsonValue29 := `{"asset_name": "marble29","color": "red","size": 2,"owner": "alan"}`
	batch.Put("ns1", "key29", []byte(jsonValue29), version.NewHeight(1, 4))
	jsonValue30 := `{"asset_name": "marble30","color": "red","size": 10,"owner": "elaine"}`
	batch.Put("ns1", "key30", []byte(jsonValue30), version.NewHeight(1, 4))

	jsonValue31 := `{"asset_name": "marble31","color": "cyan","size": 1000007,"owner": "joe"}`
	batch.Put("ns1", "key31", []byte(jsonValue31), version.NewHeight(1, 11))
	jsonValue32 := `{"asset_name": "marble32","color": "red","size": 4,"owner": "martha"}`
	batch.Put("ns1", "key32", []byte(jsonValue32), version.NewHeight(1, 4))
	jsonValue33 := `{"asset_name": "marble33","color": "red","size": 6,"owner": "james"}`
	batch.Put("ns1", "key33", []byte(jsonValue33), version.NewHeight(1, 4))
	jsonValue34 := `{"asset_name": "marble34","color": "red","size": 10,"owner": "fred"}`
	batch.Put("ns1", "key34", []byte(jsonValue34), version.NewHeight(1, 4))
	jsonValue35 := `{"asset_name": "marble35","color": "red","size": 8,"owner": "mary"}`
	batch.Put("ns1", "key35", []byte(jsonValue35), version.NewHeight(1, 4))
	jsonValue36 := `{"asset_name": "marble36","color": "orange","size": 4,"owner": "robert"}`
	batch.Put("ns1", "key36", []byte(jsonValue36), version.NewHeight(1, 4))
	jsonValue37 := `{"asset_name": "marble37","color": "red","size": 2,"owner": "alan"}`
	batch.Put("ns1", "key37", []byte(jsonValue37), version.NewHeight(1, 4))
	jsonValue38 := `{"asset_name": "marble38","color": "yellow","size": 10,"owner": "elaine"}`
	batch.Put("ns1", "key38", []byte(jsonValue38), version.NewHeight(1, 4))
	jsonValue39 := `{"asset_name": "marble39","color": "red","size": 2,"owner": "alan"}`
	batch.Put("ns1", "key39", []byte(jsonValue39), version.NewHeight(1, 4))
	jsonValue40 := `{"asset_name": "marble40","color": "red","size": 10,"owner": "elaine"}`
	batch.Put("ns1", "key40", []byte(jsonValue40), version.NewHeight(1, 4))

	savePoint := version.NewHeight(2, 22)
	db.ApplyUpdates(batch, savePoint)

//无分页的测试范围查询
	returnKeys := []string{}
	_, err = executeRangeQuery(t, db, "ns1", "key1", "key15", int32(0), returnKeys)
	assert.NoError(t, err)

//大页面测试范围查询（单页返回）
	returnKeys = []string{"key1", "key10", "key11", "key12", "key13", "key14"}
	_, err = executeRangeQuery(t, db, "ns1", "key1", "key15", int32(10), returnKeys)
	assert.NoError(t, err)

//测试显式分页
//多页测试范围查询
	returnKeys = []string{"key1", "key10"}
	nextStartKey, err := executeRangeQuery(t, db, "ns1", "key1", "key22", int32(2), returnKeys)
	assert.NoError(t, err)

//NextStartKey现在作为StartKey传入，请验证pageSize是否正常工作。
	returnKeys = []string{"key11", "key12"}
	_, err = executeRangeQuery(t, db, "ns1", nextStartKey, "key22", int32(2), returnKeys)
	assert.NoError(t, err)

//将querylimit设置为2
	viper.Set("ledger.state.couchDBConfig.internalQueryLimit", 2)

//测试隐式分页
//无页面大小和小查询限制的测试范围查询
	returnKeys = []string{}
	_, err = executeRangeQuery(t, db, "ns1", "key1", "key15", int32(0), returnKeys)
	assert.NoError(t, err)

//具有大于查询限制的页面范围查询
	returnKeys = []string{"key1", "key10", "key11", "key12"}
	_, err = executeRangeQuery(t, db, "ns1", "key1", "key15", int32(4), returnKeys)
	assert.NoError(t, err)

//将querylimit重置为1000
	viper.Set("ledger.state.couchDBConfig.internalQueryLimit", 1000)
}

func executeRangeQuery(t *testing.T, db statedb.VersionedDB, namespace, startKey, endKey string, limit int32, returnKeys []string) (string, error) {

	var itr statedb.ResultsIterator
	var err error

	if limit == 0 {

		itr, err = db.GetStateRangeScanIterator(namespace, startKey, endKey)
		if err != nil {
			return "", err
		}

	} else {

		queryOptions := make(map[string]interface{})
		if limit != 0 {
			queryOptions["limit"] = limit
		}
		itr, err = db.GetStateRangeScanIteratorWithMetadata(namespace, startKey, endKey, queryOptions)
		if err != nil {
			return "", err
		}

//验证返回的密钥
		if limit > 0 {
			TestItrWithoutClose(t, itr, returnKeys)
		}

	}

	returnBookmark := ""
	if limit > 0 {
		if queryResultItr, ok := itr.(statedb.QueryResultsIterator); ok {
			returnBookmark = queryResultItr.GetBookmarkAndClose()
		}
	}

	return returnBookmark, nil
}

//TestTrwithOutClose验证迭代器是否包含预期的键
func TestItrWithoutClose(t *testing.T, itr statedb.ResultsIterator, expectedKeys []string) {
	for _, expectedKey := range expectedKeys {
		queryResult, err := itr.Next()
		assert.NoError(t, err, "An unexpected error was thrown during iterator Next()")
		vkv := queryResult.(*statedb.VersionedKV)
		key := vkv.Key
		assert.Equal(t, expectedKey, key)
	}
	queryResult, err := itr.Next()
	assert.NoError(t, err, "An unexpected error was thrown during iterator Next()")
	assert.Nil(t, queryResult)
}

func TestApplyUpdatesWithNilHeight(t *testing.T, dbProvider statedb.VersionedDBProvider) {
	db, err := dbProvider.GetDBHandle("test-apply-updates-with-nil-height")
	assert.NoError(t, err)

	batch1 := statedb.NewUpdateBatch()
	batch1.Put("ns", "key1", []byte("value1"), version.NewHeight(1, 4))
	savePoint := version.NewHeight(1, 5)
	assert.NoError(t, db.ApplyUpdates(batch1, savePoint))

	batch2 := statedb.NewUpdateBatch()
	batch2.Put("ns", "key1", []byte("value2"), version.NewHeight(1, 1))
	assert.NoError(t, db.ApplyUpdates(batch2, nil))

	ht, err := db.GetLatestSavePoint()
	assert.NoError(t, err)
assert.Equal(t, savePoint, ht) //保存点应该仍然是用batch1设置的
//（因为batch2调用保存点为nil的applyUpdates）
}
