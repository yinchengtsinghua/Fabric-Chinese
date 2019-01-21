
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


package statedb

import (
	"sort"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/stretchr/testify/assert"
)

func TestPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Nil value to Put() did not panic\n")
		}
	}()

	batch := NewUpdateBatch()
//下面的put（）调用应导致死机
	batch.Put("ns1", "key1", nil, nil)
}

//test put（）、get（）和delete（）。
func TestPutGetDeleteExistsGetUpdates(t *testing.T) {
	batch := NewUpdateBatch()
	batch.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 1))

//get（）应返回以上inserted<k，v>对
	actualVersionedValue := batch.Get("ns1", "key1")
	assert.Equal(t, &VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}, actualVersionedValue)
//exists（）应返回false，因为key2不存在
	actualResult := batch.Exists("ns1", "key2")
	expectedResult := false
	assert.Equal(t, expectedResult, actualResult)

//exists（）应返回false，因为NS3不存在
	actualResult = batch.Exists("ns3", "key2")
	expectedResult = false
	assert.Equal(t, expectedResult, actualResult)

//get（）应返回nill，因为key2不存在
	actualVersionedValue = batch.Get("ns1", "key2")
	assert.Nil(t, actualVersionedValue)
//get（）应返回nill，因为ns3不存在
	actualVersionedValue = batch.Get("ns3", "key2")
	assert.Nil(t, actualVersionedValue)

	batch.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 2))
//exists（）应返回true，因为key2存在
	actualResult = batch.Exists("ns1", "key2")
	expectedResult = true
	assert.Equal(t, expectedResult, actualResult)

//GETUpDATED命名空间应该返回3个命名空间
	batch.Put("ns2", "key2", []byte("value2"), version.NewHeight(1, 2))
	batch.Put("ns3", "key2", []byte("value2"), version.NewHeight(1, 2))
	actualNamespaces := batch.GetUpdatedNamespaces()
	sort.Strings(actualNamespaces)
	expectedNamespaces := []string{"ns1", "ns2", "ns3"}
	assert.Equal(t, expectedNamespaces, actualNamespaces)

//getupdates应为命名空间ns1返回两个versionedValues
	expectedUpdates := make(map[string]*VersionedValue)
	expectedUpdates["key1"] = &VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	expectedUpdates["key2"] = &VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	actualUpdates := batch.GetUpdates("ns1")
	assert.Equal(t, expectedUpdates, actualUpdates)

	actualUpdates = batch.GetUpdates("ns4")
	assert.Nil(t, actualUpdates)

//删除上面插入的<k，v>对
	batch.Delete("ns1", "key2", version.NewHeight(1, 2))
//exists（）应在删除key2后返回true
//exists（）应返回true如果该键在此批处理中有操作（放置/删除）
	actualResult = batch.Exists("ns1", "key2")
	expectedResult = true
	assert.Equal(t, expectedResult, actualResult)

}

func TestUpdateBatchIterator(t *testing.T) {
	batch := NewUpdateBatch()
	batch.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 1))
	batch.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 2))
	batch.Put("ns1", "key3", []byte("value3"), version.NewHeight(1, 3))

	batch.Put("ns2", "key6", []byte("value6"), version.NewHeight(2, 3))
	batch.Put("ns2", "key5", []byte("value5"), version.NewHeight(2, 2))
	batch.Put("ns2", "key4", []byte("value4"), version.NewHeight(2, 1))

	checkItrResults(t, batch.GetRangeScanIterator("ns1", "key2", "key3"), []*VersionedKV{
		{CompositeKey{"ns1", "key2"}, VersionedValue{[]byte("value2"), nil, version.NewHeight(1, 2)}},
	})

	checkItrResults(t, batch.GetRangeScanIterator("ns2", "key0", "key8"), []*VersionedKV{
		{CompositeKey{"ns2", "key4"}, VersionedValue{[]byte("value4"), nil, version.NewHeight(2, 1)}},
		{CompositeKey{"ns2", "key5"}, VersionedValue{[]byte("value5"), nil, version.NewHeight(2, 2)}},
		{CompositeKey{"ns2", "key6"}, VersionedValue{[]byte("value6"), nil, version.NewHeight(2, 3)}},
	})

	checkItrResults(t, batch.GetRangeScanIterator("ns2", "", ""), []*VersionedKV{
		{CompositeKey{"ns2", "key4"}, VersionedValue{[]byte("value4"), nil, version.NewHeight(2, 1)}},
		{CompositeKey{"ns2", "key5"}, VersionedValue{[]byte("value5"), nil, version.NewHeight(2, 2)}},
		{CompositeKey{"ns2", "key6"}, VersionedValue{[]byte("value6"), nil, version.NewHeight(2, 3)}},
	})

	checkItrResults(t, batch.GetRangeScanIterator("non-existing-ns", "", ""), nil)
}

func checkItrResults(t *testing.T, itr QueryResultsIterator, expectedResults []*VersionedKV) {
	for i := 0; i < len(expectedResults); i++ {
		res, _ := itr.Next()
		assert.Equal(t, expectedResults[i], res)
	}
	lastRes, err := itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, lastRes)
	itr.Close()
}

//testpaginatedrangevalidation使用分页测试查询
func TestPaginatedRangeValidation(t *testing.T) {

	queryOptions := make(map[string]interface{})
	queryOptions["limit"] = int32(10)

	err := ValidateRangeMetadata(queryOptions)
	assert.NoError(t, err, "An error was thrown for a valid option")

	queryOptions = make(map[string]interface{})
	queryOptions["limit"] = float32(10.2)

	err = ValidateRangeMetadata(queryOptions)
	assert.Error(t, err, "An should have been thrown for an invalid option")

	queryOptions = make(map[string]interface{})
	queryOptions["limit"] = "10"

	err = ValidateRangeMetadata(queryOptions)
	assert.Error(t, err, "An should have been thrown for an invalid option")

	queryOptions = make(map[string]interface{})
	queryOptions["limit1"] = int32(10)

	err = ValidateRangeMetadata(queryOptions)
	assert.Error(t, err, "An should have been thrown for an invalid option")

}
