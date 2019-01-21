
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


package protolator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/tools/protolator/testprotos"
	"github.com/stretchr/testify/assert"
)

type testProtoPlainFieldFactory struct {
	fromPrefix string
	toPrefix   string
	fromError  error
	toError    error
}

func (tpff *testProtoPlainFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	return fieldName == "plain_field"
}

func (tpff *testProtoPlainFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &plainField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: reflect.TypeOf(""),
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(source interface{}, destType reflect.Type) (reflect.Value, error) {
			sourceAsString := source.(string)
			return reflect.ValueOf(tpff.fromPrefix + sourceAsString), tpff.fromError
		},
		populateTo: func(source reflect.Value) (interface{}, error) {
			return tpff.toPrefix + source.Interface().(string), tpff.toError
		},
	}, nil
}

func TestSimpleMsgPlainField(t *testing.T) {
	fromPrefix := "from"
	toPrefix := "to"
	tppff := &testProtoPlainFieldFactory{
		fromPrefix: fromPrefix,
		toPrefix:   toPrefix,
	}

	fieldFactories = []protoFieldFactory{tppff}

	pfValue := "foo"
	startMsg := &testprotos.SimpleMsg{
		PlainField: pfValue,
		MapField:   map[string]string{"1": "2"},
		SliceField: []string{"a", "b"},
	}

	var buffer bytes.Buffer
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))

	newMsg := &testprotos.SimpleMsg{}
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))

	assert.Equal(t, startMsg.MapField, newMsg.MapField)
	assert.Equal(t, startMsg.SliceField, newMsg.SliceField)
	assert.Equal(t, fromPrefix+toPrefix+startMsg.PlainField, newMsg.PlainField)

	tppff.fromError = fmt.Errorf("Failing from intentionally")
	assert.Error(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))

	tppff.toError = fmt.Errorf("Failing to intentionally")
	assert.Error(t, DeepMarshalJSON(&buffer, startMsg))
}

type testProtoMapFieldFactory struct {
	fromPrefix string
	toPrefix   string
	fromError  error
	toError    error
}

func (tpff *testProtoMapFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	return fieldName == "map_field"
}

func (tpff *testProtoMapFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &mapField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: reflect.TypeOf(""),
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(key string, source interface{}, destType reflect.Type) (reflect.Value, error) {
			sourceAsString := source.(string)
			return reflect.ValueOf(tpff.fromPrefix + key + sourceAsString), tpff.fromError
		},
		populateTo: func(key string, source reflect.Value) (interface{}, error) {
			return tpff.toPrefix + key + source.Interface().(string), tpff.toError
		},
	}, nil
}

func TestSimpleMsgMapField(t *testing.T) {
	fromPrefix := "from"
	toPrefix := "to"
	tpmff := &testProtoMapFieldFactory{
		fromPrefix: fromPrefix,
		toPrefix:   toPrefix,
	}
	fieldFactories = []protoFieldFactory{tpmff}

	key := "foo"
	value := "bar"
	startMsg := &testprotos.SimpleMsg{
		PlainField: "1",
		MapField:   map[string]string{key: value},
		SliceField: []string{"a", "b"},
	}

	var buffer bytes.Buffer
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))

	newMsg := &testprotos.SimpleMsg{}
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))

	assert.Equal(t, startMsg.PlainField, newMsg.PlainField)
	assert.Equal(t, startMsg.SliceField, newMsg.SliceField)
	assert.Equal(t, fromPrefix+key+toPrefix+key+startMsg.MapField[key], newMsg.MapField[key])

	tpmff.fromError = fmt.Errorf("Failing from intentionally")
	assert.Error(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))

	tpmff.toError = fmt.Errorf("Failing to intentionally")
	assert.Error(t, DeepMarshalJSON(&buffer, startMsg))
}

type testProtoSliceFieldFactory struct {
	fromPrefix string
	toPrefix   string
	fromError  error
	toError    error
}

func (tpff *testProtoSliceFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	return fieldName == "slice_field"
}

func (tpff *testProtoSliceFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &sliceField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: reflect.TypeOf(""),
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(index int, source interface{}, destType reflect.Type) (reflect.Value, error) {
			sourceAsString := source.(string)
			return reflect.ValueOf(tpff.fromPrefix + fmt.Sprintf("%d", index) + sourceAsString), tpff.fromError
		},
		populateTo: func(index int, source reflect.Value) (interface{}, error) {
			return tpff.toPrefix + fmt.Sprintf("%d", index) + source.Interface().(string), tpff.toError
		},
	}, nil
}

func TestSimpleMsgSliceField(t *testing.T) {
	fromPrefix := "from"
	toPrefix := "to"
	tpsff := &testProtoSliceFieldFactory{
		fromPrefix: fromPrefix,
		toPrefix:   toPrefix,
	}
	fieldFactories = []protoFieldFactory{tpsff}

	value := "foo"
	startMsg := &testprotos.SimpleMsg{
		PlainField: "1",
		MapField:   map[string]string{"a": "b"},
		SliceField: []string{value},
	}

	var buffer bytes.Buffer
	assert.NoError(t, DeepMarshalJSON(&buffer, startMsg))

	newMsg := &testprotos.SimpleMsg{}
	assert.NoError(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))

	assert.Equal(t, startMsg.PlainField, newMsg.PlainField)
	assert.Equal(t, startMsg.MapField, newMsg.MapField)
	assert.Equal(t, fromPrefix+"0"+toPrefix+"0"+startMsg.SliceField[0], newMsg.SliceField[0])

	tpsff.fromError = fmt.Errorf("Failing from intentionally")
	assert.Error(t, DeepUnmarshalJSON(bytes.NewReader(buffer.Bytes()), newMsg))

	tpsff.toError = fmt.Errorf("Failing to intentionally")
	assert.Error(t, DeepMarshalJSON(&buffer, startMsg))
}

type testProtoFailFactory struct{}

func (tpff testProtoFailFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	return true
}

func (tpff testProtoFailFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return nil, fmt.Errorf("Intentionally failing")
}

func TestFailFactory(t *testing.T) {
	fieldFactories = []protoFieldFactory{&testProtoFailFactory{}}

	var buffer bytes.Buffer
	assert.Error(t, DeepMarshalJSON(&buffer, &testprotos.SimpleMsg{}))
}

func TestJSONUnmarshalMaxUint32(t *testing.T) {
	fieldName := "numField"
	jsonString := fmt.Sprintf("{\"%s\":%d}", fieldName, math.MaxUint32)
	m, err := jsonToMap([]byte(jsonString))
	assert.NoError(t, err)
	assert.IsType(t, json.Number(""), m[fieldName])
}

func TestMostlyDeterministicMarshal(t *testing.T) {
	multiKeyMap := &testprotos.SimpleMsg{
		MapField: map[string]string{
			"a": "b",
			"c": "d",
			"e": "f",
			"g": "h",
			"i": "j",
			"k": "l",
			"m": "n",
			"o": "p",
			"q": "r",
			"s": "t",
			"u": "v",
			"w": "x",
			"y": "z",
		},
	}

	result, err := MostlyDeterministicMarshal(multiKeyMap)
	assert.NoError(t, err)
	assert.NotNil(t, result)

//
//同一条消息与嵌入的映射多次，我们应该
//如果默认行为持续存在，则检测不匹配。即使有3张地图
//元素，通常在2-3次迭代中不匹配，因此13
//条目和10次迭代似乎是一个合理的检查。
	for i := 0; i < 10; i++ {
		newResult, err := MostlyDeterministicMarshal(multiKeyMap)
		assert.NoError(t, err)
		assert.Equal(t, result, newResult)
	}

	unmarshaled := &testprotos.SimpleMsg{}
	err = proto.Unmarshal(result, unmarshaled)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(unmarshaled, multiKeyMap))
}
