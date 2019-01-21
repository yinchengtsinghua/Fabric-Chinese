
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


package statecouchdb

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
)

const (
	binaryWrapper = "valueBytes"
	idField       = "_id"
	revField      = "_rev"
	versionField  = "~version"
	deletedField  = "_deleted"
)

type keyValue struct {
	key string
	*statedb.VersionedValue
}

type jsonValue map[string]interface{}

func tryCastingToJSON(b []byte) (isJSON bool, val jsonValue) {
	var jsonVal map[string]interface{}
	err := json.Unmarshal(b, &jsonVal)
	return err == nil, jsonValue(jsonVal)
}

func castToJSON(b []byte) (jsonValue, error) {
	var jsonVal map[string]interface{}
	err := json.Unmarshal(b, &jsonVal)
	err = errors.Wrap(err, "error unmarshalling json data")
	return jsonVal, err
}

func (v jsonValue) checkReservedFieldsNotPresent() error {
	for fieldName := range v {
		if fieldName == versionField || strings.HasPrefix(fieldName, "_") {
			return errors.Errorf("field [%s] is not valid for the CouchDB state database", fieldName)
		}
	}
	return nil
}

func (v jsonValue) removeRevField() {
	delete(v, revField)
}

func (v jsonValue) toBytes() ([]byte, error) {
	jsonBytes, err := json.Marshal(v)
	err = errors.Wrap(err, "error marshalling json data")
	return jsonBytes, err
}

func couchDocToKeyValue(doc *couchdb.CouchDoc) (*keyValue, error) {
//初始化返回值
	var returnValue []byte
	var err error
//创建通用映射取消标记JSON
	jsonResult := make(map[string]interface{})
	decoder := json.NewDecoder(bytes.NewBuffer(doc.JSONValue))
	decoder.UseNumber()
	if err = decoder.Decode(&jsonResult); err != nil {
		return nil, err
	}
//验证版本字段是否存在
	if _, fieldFound := jsonResult[versionField]; !fieldFound {
		return nil, errors.Errorf("version field %s was not found", versionField)
	}
	key := jsonResult[idField].(string)
//从JSON中的版本字段创建返回版本

	returnVersion, returnMetadata, err := decodeVersionAndMetadata(jsonResult[versionField].(string))
	if err != nil {
		return nil, err
	}
//删除“ID”、“Rev”和“Version”字段
	delete(jsonResult, idField)
	delete(jsonResult, revField)
	delete(jsonResult, versionField)

//处理二进制或JSON数据
if doc.Attachments != nil { //二进制附件
//从附件获取二进制数据
		for _, attachment := range doc.Attachments {
			if attachment.Name == binaryWrapper {
				returnValue = attachment.AttachmentBytes
			}
		}
	} else {
//封送返回的JSON数据。
		if returnValue, err = json.Marshal(jsonResult); err != nil {
			return nil, err
		}
	}
	return &keyValue{key, &statedb.VersionedValue{
		Value:    returnValue,
		Metadata: returnMetadata,
		Version:  returnVersion},
	}, nil
}

func keyValToCouchDoc(kv *keyValue, revision string) (*couchdb.CouchDoc, error) {
	type kvType int32
	const (
		kvTypeDelete = iota
		kvTypeJSON
		kvTypeAttachment
	)
	key, value, metadata, version := kv.key, kv.Value, kv.Metadata, kv.Version
	jsonMap := make(jsonValue)

	var kvtype kvType
	switch {
	case value == nil:
		kvtype = kvTypeDelete
//check for the case where the jsonMap is nil,  this will indicate
//导致有效JSON返回nil的unmashal的特殊情况
	case json.Unmarshal(value, &jsonMap) == nil && jsonMap != nil:
		kvtype = kvTypeJSON
		if err := jsonMap.checkReservedFieldsNotPresent(); err != nil {
			return nil, err
		}
	default:
//如果映射为零，则创建空映射
		if jsonMap == nil {
			jsonMap = make(jsonValue)
		}
		kvtype = kvTypeAttachment
	}

	verAndMetadata, err := encodeVersionAndMetadata(version, metadata)
	if err != nil {
		return nil, err
	}
//add the (version + metadata), id, revision, and delete marker (if needed)
	jsonMap[versionField] = verAndMetadata
	jsonMap[idField] = key
	if revision != "" {
		jsonMap[revField] = revision
	}
	if kvtype == kvTypeDelete {
		jsonMap[deletedField] = true
	}
	jsonBytes, err := jsonMap.toBytes()
	if err != nil {
		return nil, err
	}
	couchDoc := &couchdb.CouchDoc{JSONValue: jsonBytes}
	if kvtype == kvTypeAttachment {
		attachment := &couchdb.AttachmentInfo{}
		attachment.AttachmentBytes = value
		attachment.ContentType = "application/octet-stream"
		attachment.Name = binaryWrapper
		attachments := append([]*couchdb.AttachmentInfo{}, attachment)
		couchDoc.Attachments = attachments
	}
	return couchDoc, nil
}

//couchsavepointdata用于couchdb
type couchSavepointData struct {
	BlockNum uint64 `json:"BlockNum"`
	TxNum    uint64 `json:"TxNum"`
}

func encodeSavepoint(height *version.Height) (*couchdb.CouchDoc, error) {
	var err error
	var savepointDoc couchSavepointData
//构造保存点文档
	savepointDoc.BlockNum = height.BlockNum
	savepointDoc.TxNum = height.TxNum
	savepointDocJSON, err := json.Marshal(savepointDoc)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal savepoint data")
		logger.Errorf("%+v", err)
		return nil, err
	}
	return &couchdb.CouchDoc{JSONValue: savepointDocJSON, Attachments: nil}, nil
}

func decodeSavepoint(couchDoc *couchdb.CouchDoc) (*version.Height, error) {
	savepointDoc := &couchSavepointData{}
	if err := json.Unmarshal(couchDoc.JSONValue, &savepointDoc); err != nil {
		err = errors.Wrap(err, "failed to unmarshal savepoint data")
		logger.Errorf("%+v", err)
		return nil, err
	}
	return &version.Height{BlockNum: savepointDoc.BlockNum, TxNum: savepointDoc.TxNum}, nil
}

func validateValue(value []byte) error {
	isJSON, jsonVal := tryCastingToJSON(value)
	if !isJSON {
		return nil
	}
	return jsonVal.checkReservedFieldsNotPresent()
}

func validateKey(key string) error {
	if !utf8.ValidString(key) {
		return errors.Errorf("invalid key [%x], must be a UTF-8 string", key)
	}
	if strings.HasPrefix(key, "_") {
		return errors.Errorf("invalid key [%s], cannot begin with \"_\"", key)
	}
	if key == "" {
		return errors.New("invalid key. Empty string is not supported as a key by couchdb")
	}
	return nil
}

//如果这是JSON，则removejsonRevision将删除“_rev”
func removeJSONRevision(jsonValue *[]byte) error {
	jsonVal, err := castToJSON(*jsonValue)
	if err != nil {
		logger.Errorf("Failed to unmarshal couchdb JSON data: %+v", err)
		return err
	}
	jsonVal.removeRevField()
	if *jsonValue, err = jsonVal.toBytes(); err != nil {
		logger.Errorf("Failed to marshal couchdb JSON data: %+v", err)
	}
	return err
}
