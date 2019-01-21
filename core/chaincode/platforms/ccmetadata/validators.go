
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
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

//filevalidator用作验证特定元数据目录的处理程序
type fileValidator func(fileName string, fileBytes []byte) error

//allowedCharsCollectionName捕获有效集合名称的regex模式
const AllowedCharsCollectionName = "[A-Za-z0-9_-]+"

//目前，唯一需要和允许的元数据是META-INF/statedb/couchdb/indexes。
var fileValidators = map[*regexp.Regexp]fileValidator{
	regexp.MustCompile("^META-INF/statedb/couchdb/indexes/.*[.]json"):                                                couchdbIndexFileValidator,
	regexp.MustCompile("^META-INF/statedb/couchdb/collections/" + AllowedCharsCollectionName + "/indexes/.*[.]json"): couchdbIndexFileValidator,
}

var collectionNameValid = regexp.MustCompile("^" + AllowedCharsCollectionName)

var fileNameValid = regexp.MustCompile("^.*[.]json")

var validDatabases = []string{"couchdb"}

//未处理目录中的元数据文件返回UnhandledDirectoryError
type UnhandledDirectoryError struct {
	err string
}

func (e *UnhandledDirectoryError) Error() string {
	return e.err
}

//对于包含无效内容的元数据文件，返回InvalidIndexContentError
type InvalidIndexContentError struct {
	err string
}

func (e *InvalidIndexContentError) Error() string {
	return e.err
}

//验证元数据文件检查元数据文件是否有效
//根据文件目录的验证规则
func ValidateMetadataFile(filePathName string, fileBytes []byte) error {
//获取元数据目录的验证程序处理程序
	fileValidator := selectFileValidator(filePathName)

//如果没有元数据目录的验证程序处理程序，则返回UnhandledDirectoryError
	if fileValidator == nil {
		return &UnhandledDirectoryError{buildMetadataFileErrorMessage(filePathName)}
	}

//如果文件对于给定的基于目录的验证器无效，则返回相应的错误。
	err := fileValidator(filePathName, fileBytes)
	if err != nil {
		return err
	}

//文件有效，返回零错误
	return nil
}

func buildMetadataFileErrorMessage(filePathName string) string {

	dir, filename := filepath.Split(filePathName)

	if !strings.HasPrefix(filePathName, "META-INF/statedb") {
		return fmt.Sprintf("metadata file path must begin with META-INF/statedb, found: %s", dir)
	}
	directoryArray := strings.Split(filepath.Clean(dir), "/")
//验证最小目录深度
	if len(directoryArray) < 4 {
		return fmt.Sprintf("metadata file path must include a database and index directory: %s", dir)
	}
//验证数据库类型
	if !contains(validDatabases, directoryArray[2]) {
		return fmt.Sprintf("database name [%s] is not supported, valid options: %s", directoryArray[2], validDatabases)
	}
//验证“indexes”是否在数据库名称下
	if len(directoryArray) == 4 && directoryArray[3] != "indexes" {
		return fmt.Sprintf("metadata file path does not have an indexes directory: %s", dir)
	}
//如果是集合，请检查路径长度
	if len(directoryArray) != 6 {
		return fmt.Sprintf("metadata file path for collections must include a collections and index directory: %s", dir)
	}
//
	if directoryArray[3] != "collections" || directoryArray[5] != "indexes" {
		return fmt.Sprintf("metadata file path for collections must have a collections and indexes directory: %s", dir)
	}
//验证集合名称
	if !collectionNameValid.MatchString(directoryArray[4]) {
		return fmt.Sprintf("collection name is not valid: %s", directoryArray[4])
	}

//验证文件名
	if !fileNameValid.MatchString(filename) {
		return fmt.Sprintf("artifact file name is not valid: %s", filename)
	}

	return fmt.Sprintf("metadata file path or name is not supported: %s", dir)

}

func contains(validStrings []string, target string) bool {
	for _, str := range validStrings {
		if str == target {
			return true
		}
	}
	return false
}

func selectFileValidator(filePathName string) fileValidator {
	for validateExp, fileValidator := range fileValidators {
		isValid := validateExp.MatchString(filePathName)
		if isValid {
			return fileValidator
		}
	}
	return nil
}

//couchdbindexfilevalidator实现filevalidator
func couchdbIndexFileValidator(fileName string, fileBytes []byte) error {

//如果内容未验证为JSON，则返回err以使文件无效。
	boolIsJSON, indexDefinition := isJSON(fileBytes)
	if !boolIsJSON {
		return &InvalidIndexContentError{fmt.Sprintf("Index metadata file [%s] is not a valid JSON", fileName)}
	}

//
	err := validateIndexJSON(indexDefinition)
	if err != nil {
		return &InvalidIndexContentError{fmt.Sprintf("Index metadata file [%s] is not a valid index definition: %s", fileName, err)}
	}

	return nil

}

//isjson测试一个字符串，以确定它是否可以被解析为有效的json。
func isJSON(s []byte) (bool, map[string]interface{}) {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil, js
}

func validateIndexJSON(indexDefinition map[string]interface{}) error {

//跟踪是否包含“索引”键的标志
	indexIncluded := false

//遍历JSON索引定义
	for jsonKey, jsonValue := range indexDefinition {

//为顶级条目创建案例
		switch jsonKey {

		case "index":

			if reflect.TypeOf(jsonValue).Kind() != reflect.Map {
				return fmt.Errorf("Invalid entry, \"index\" must be a JSON")
			}

			err := processIndexMap(jsonValue.(map[string]interface{}))
			if err != nil {
				return err
			}

			indexIncluded = true

		case "ddoc":

//
			if reflect.TypeOf(jsonValue).Kind() != reflect.String {
				return fmt.Errorf("Invalid entry, \"ddoc\" must be a string")
			}

			logger.Debugf("Found index object: \"%s\":\"%s\"", jsonKey, jsonValue)

		case "name":

//验证名称是否为字符串
			if reflect.TypeOf(jsonValue).Kind() != reflect.String {
				return fmt.Errorf("Invalid entry, \"name\" must be a string")
			}

			logger.Debugf("Found index object: \"%s\":\"%s\"", jsonKey, jsonValue)

		case "type":

			if jsonValue != "json" {
				return fmt.Errorf("Index type must be json")
			}

			logger.Debugf("Found index object: \"%s\":\"%s\"", jsonKey, jsonValue)

		default:

			return fmt.Errorf("Invalid Entry.  Entry %s", jsonKey)

		}

	}

	if !indexIncluded {
		return fmt.Errorf("Index definition must include a \"fields\" definition")
	}

	return nil

}

//processindexmap处理接口映射并包装字段名或遍历
//
func processIndexMap(jsonFragment map[string]interface{}) error {

//重复映射中的项
	for jsonKey, jsonValue := range jsonFragment {

		switch jsonKey {

		case "fields":

			switch jsonValueType := jsonValue.(type) {

			case []interface{}:

//迭代索引字段对象
				for _, itemValue := range jsonValueType {

					switch reflect.TypeOf(itemValue).Kind() {

					case reflect.String:
//字符串是有效的字段描述符，例如：“颜色”、“大小”
						logger.Debugf("Found index field name: \"%s\"", itemValue)

					case reflect.Map:
//处理包含排序的情况，例如：“size”：“asc”，“color”：“desc”
						err := validateFieldMap(itemValue.(map[string]interface{}))
						if err != nil {
							return err
						}

					}
				}

			default:
				return fmt.Errorf("Expecting a JSON array of fields")
			}

		case "partial_filter_selector":

//
//不采取其他行动，将被视为目前有效

		default:

//如果发现“字段”或“部分筛选”之外的任何内容，
//返回错误
			return fmt.Errorf("Invalid Entry.  Entry %s", jsonKey)

		}

	}

	return nil

}

//validateFieldMap验证字段对象列表
func validateFieldMap(jsonFragment map[string]interface{}) error {

//重复字段以验证排序条件
	for jsonKey, jsonValue := range jsonFragment {

		switch jsonValue.(type) {

		case string:
//确保排序为“asc”或“desc”
			if !(strings.ToLower(jsonValue.(string)) == "asc" || strings.ToLower(jsonValue.(string)) == "desc") {
				return fmt.Errorf("Sort must be either \"asc\" or \"desc\".  \"%s\" was found.", jsonValue)
			}
			logger.Debugf("Found index field name: \"%s\":\"%s\"", jsonKey, jsonValue)

		default:
			return fmt.Errorf("Invalid field definition, fields must be in the form \"fieldname\":\"sort\"")

		}
	}

	return nil

}
