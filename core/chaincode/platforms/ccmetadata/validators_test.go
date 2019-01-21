
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


package ccmetadata

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var packageTestDir = filepath.Join(os.TempDir(), "ccmetadata-validator-test")

func TestGoodIndexJSON(t *testing.T) {
	testDir := filepath.Join(packageTestDir, "GoodIndexJSON")
	cleanupDir(testDir)
	defer cleanupDir(testDir)

	fileName := "META-INF/statedb/couchdb/indexes/myIndex.json"
	fileBytes := []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err := ValidateMetadataFile(fileName, fileBytes)
	assert.NoError(t, err, "Error validating a good index")
}

func TestBadIndexJSON(t *testing.T) {
	testDir := filepath.Join(packageTestDir, "BadIndexJSON")
	cleanupDir(testDir)
	defer cleanupDir(testDir)

	fileName := "META-INF/statedb/couchdb/indexes/myIndex.json"
	fileBytes := []byte("invalid json")

	err := ValidateMetadataFile(fileName, fileBytes)

	assert.Error(t, err, "Should have received an InvalidIndexContentError")

//InvalidIndexContentError上的类型断言
	_, ok := err.(*InvalidIndexContentError)
	assert.True(t, ok, "Should have received an InvalidIndexContentError")

	t.Log("SAMPLE ERROR STRING:", err.Error())
}

func TestIndexWrongLocation(t *testing.T) {
	testDir := filepath.Join(packageTestDir, "IndexWrongLocation")
	cleanupDir(testDir)
	defer cleanupDir(testDir)

	fileName := "META-INF/statedb/couchdb/myIndex.json"
	fileBytes := []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err := ValidateMetadataFile(fileName, fileBytes)
	assert.Error(t, err, "Should have received an UnhandledDirectoryError")

//UnhandledDirectoryError上的类型断言
	_, ok := err.(*UnhandledDirectoryError)
	assert.True(t, ok, "Should have received an UnhandledDirectoryError")

	t.Log("SAMPLE ERROR STRING:", err.Error())
}

func TestInvalidMetadataType(t *testing.T) {
	testDir := filepath.Join(packageTestDir, "InvalidMetadataType")
	cleanupDir(testDir)
	defer cleanupDir(testDir)

	fileName := "myIndex.json"
	fileBytes := []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err := ValidateMetadataFile(fileName, fileBytes)
	assert.Error(t, err, "Should have received an UnhandledDirectoryError")

//UnhandledDirectoryError上的类型断言
	_, ok := err.(*UnhandledDirectoryError)
	assert.True(t, ok, "Should have received an UnhandledDirectoryError")
}

func TestBadMetadataExtension(t *testing.T) {
	testDir := filepath.Join(packageTestDir, "BadMetadataExtension")
	cleanupDir(testDir)
	defer cleanupDir(testDir)

	fileName := "myIndex.go"
	fileBytes := []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err := ValidateMetadataFile(fileName, fileBytes)
	assert.Error(t, err, "Should have received an error")

}

func TestBadFilePaths(t *testing.T) {
	testDir := filepath.Join(packageTestDir, "BadMetadataExtension")
	cleanupDir(testDir)
	defer cleanupDir(testDir)

//测试错误的META-INF
	fileName := "META-INF1/statedb/couchdb/indexes/test1.json"
	fileBytes := []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err := ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.Error(t, err, "Should have received an error for bad META-INF directory")

//
	fileName = "META-INF/statedb/test1.json"
	fileBytes = []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err = ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.Error(t, err, "Should have received an error for bad length")

//测试无效的数据库名称
	fileName = "META-INF/statedb/goleveldb/indexes/test1.json"
	fileBytes = []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err = ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.Error(t, err, "Should have received an error for invalid database")

//测试无效索引目录名
	fileName = "META-INF/statedb/couchdb/index/test1.json"
	fileBytes = []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err = ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.Error(t, err, "Should have received an error for invalid indexes directory")

//测试无效的集合目录名
	fileName = "META-INF/statedb/couchdb/collection/testcoll/indexes/test1.json"
	fileBytes = []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err = ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.Error(t, err, "Should have received an error for invalid collections directory")

//测试有效集合名称
	fileName = "META-INF/statedb/couchdb/collections/testcoll/indexes/test1.json"
	fileBytes = []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err = ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.NoError(t, err, "Error should not have been thrown for a valid collection name")

//测试无效集合名称
	fileName = "META-INF/statedb/couchdb/collections/#testcoll/indexes/test1.json"
	fileBytes = []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err = ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.Error(t, err, "Should have received an error for an invalid collection name")

//测试无效集合名称
	fileName = "META-INF/statedb/couchdb/collections/testcoll/indexes/test1.txt"
	fileBytes = []byte(`{"index":{"fields":["data.docType","data.owner"]},"name":"indexOwner","type":"json"}`)

	err = ValidateMetadataFile(fileName, fileBytes)
	fmt.Println(err)
	assert.Error(t, err, "Should have received an error for an invalid file name")

}

func TestIndexValidation(t *testing.T) {

//
	indexDef := []byte(`{"index":{"fields":[{"size":"desc"}, {"color":"desc"}]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition := isJSON(indexDef)
	err := validateIndexJSON(indexDefinition)
	assert.NoError(t, err)

//测试没有字段排序的有效索引
	indexDef = []byte(`{"index":{"fields":["size","color"]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.NoError(t, err)

//测试没有设计文档、名称和类型的有效索引
	indexDef = []byte(`{"index":{"fields":["size","color"]}}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.NoError(t, err)

//使用部分筛选选择器测试有效索引（仅测试包含该索引时不会返回错误）
	indexDef = []byte(`{
		  "index": {
		    "partial_filter_selector": {
		      "status": {
		        "$ne": "archived"
		      }
		    },
		    "fields": ["type"]
		  },
		  "ddoc" : "type-not-archived",
		  "type" : "json"
		}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.NoError(t, err)

}

func TestIndexValidationInvalidParameters(t *testing.T) {

//测试为参数传入的数值
	indexDef := []byte(`{"index":{"fields":[{"size":"desc"}, {"color":"desc"}]},"ddoc":1, "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition := isJSON(indexDef)
	err := validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for numeric design doc")

//测试无效的设计文档参数
	indexDef = []byte(`{"index":{"fields":["size","color"]},"ddoc1":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for invalid design doc parameter")

//测试无效的名称参数
	indexDef = []byte(`{"index":{"fields":["size","color"]},"ddoc":"indexSizeSortName", "name1":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for invalid name parameter")

//测试无效的类型参数，数字
	indexDef = []byte(`{"index":{"fields":["size","color"]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":1}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for numeric type parameter")

//测试无效的类型参数
	indexDef = []byte(`{"index":{"fields":["size","color"]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"text"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for invalid type parameter")

//测试无效的索引参数
	indexDef = []byte(`{"index1":{"fields":["size","color"]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for invalid index parameter")

//测试缺少索引参数
	indexDef = []byte(`{"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for missing index parameter")

}

func TestIndexValidationInvalidFields(t *testing.T) {

//测试无效字段参数
	indexDef := []byte(`{"index":{"fields1":[{"size":"desc"}, {"color":"desc"}]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition := isJSON(indexDef)
	err := validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for invalid fields parameter")

//测试无效的字段名（数字）
	indexDef = []byte(`{"index":{"fields":["size", 1]},"ddoc1":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for field name defined as numeric")

//测试无效字段排序
	indexDef = []byte(`{"index":{"fields":[{"size":"desc1"}, {"color":"desc"}]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for invalid field sort")

//排序时测试数字
	indexDef = []byte(`{"index":{"fields":[{"size":1}, {"color":"desc"}]},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for a numeric in field sort")

//测试字段的无效JSON
	indexDef = []byte(`{"index":{"fields":"size"},"ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for invalid field json")

//测试字段缺少JSON
	indexDef = []byte(`{"index":"fields","ddoc":"indexSizeSortName", "name":"indexSizeSortName","type":"json"}`)
	_, indexDefinition = isJSON(indexDef)
	err = validateIndexJSON(indexDefinition)
	assert.Error(t, err, "Error should have been thrown for missing JSON for fields")

}

func cleanupDir(dir string) error {
//清除以前的所有文件
	err := os.RemoveAll(dir)
	if err != nil {
		return nil
	}
	return os.Mkdir(dir, os.ModePerm)
}

func writeToFile(filename string, bytes []byte) error {
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, bytes, 0644)
}
