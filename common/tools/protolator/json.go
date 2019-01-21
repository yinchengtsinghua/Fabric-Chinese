
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
	"io"
	"io/ioutil"
	"reflect"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

//大多数情况下，TerministicMarshal不是您要寻找的功能。
//它使Protobuf序列化在单个构建中保持一致。它
//不能保证序列化在proto中是确定性的
//版本或原型实现。它适用于以下情况：
//同一进程希望比较二进制消息是否相等
//需要先解组，但一般不应使用。
func MostlyDeterministicMarshal(msg proto.Message) ([]byte, error) {
	buffer := proto.NewBuffer(make([]byte, 0))
	buffer.SetDeterministic(true)
	if err := buffer.Marshal(msg); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type protoFieldFactory interface {
//句柄应返回此特定的ProtoFieldFactory实例
//负责给定的原型场
	Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool

//NewProtoField应该创建一个支持ProtoField实现器
//请注意，FieldValue可能表示nil，因此FieldType也是
//包括（反映零值的类型会引起恐慌）
	NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error)
}

type protoField interface {
//name返回字段的原名称
	Name() string

//PopulateFrom通过采用中间的JSON表示来改变底层对象
//并将其转换为原型表示，然后将其分配给支持值
//通过反射
	PopulateFrom(source interface{}) error

//Populatto不改变基础对象，而是转换它
//进入中间JSON表示（即结构->映射[string]接口
//或结构切片到[]映射[string]接口
	PopulateTo() (interface{}, error)
}

var (
	protoMsgType           = reflect.TypeOf((*proto.Message)(nil)).Elem()
	mapStringInterfaceType = reflect.TypeOf(map[string]interface{}{})
	bytesType              = reflect.TypeOf([]byte{})
)

type baseField struct {
	msg   proto.Message
	name  string
	fType reflect.Type
	vType reflect.Type
	value reflect.Value
}

func (bf *baseField) Name() string {
	return bf.name
}

type plainField struct {
	baseField
	populateFrom func(source interface{}, destType reflect.Type) (reflect.Value, error)
	populateTo   func(source reflect.Value) (interface{}, error)
}

func (pf *plainField) PopulateFrom(source interface{}) error {
	if source == nil {
		return nil
	}

	if !reflect.TypeOf(source).AssignableTo(pf.fType) {
		return fmt.Errorf("expected field %s for message %T to be assignable from %v but was not.  Is %T", pf.name, pf.msg, pf.fType, source)
	}
	value, err := pf.populateFrom(source, pf.vType)
	if err != nil {
		return fmt.Errorf("error in PopulateFrom for field %s for message %T: %s", pf.name, pf.msg, err)
	}
	pf.value.Set(value)
	return nil
}

func (pf *plainField) PopulateTo() (interface{}, error) {
	if !pf.value.Type().AssignableTo(pf.vType) {
		return nil, fmt.Errorf("expected field %s for message %T to be assignable to %v but was not. Got %T.", pf.name, pf.msg, pf.fType, pf.value)
	}

	kind := pf.value.Type().Kind()
//不要试图对零字段进行深度编码，因为没有正确的类型信息等。
//可能返回错误
	if (kind == reflect.Ptr || kind == reflect.Slice || kind == reflect.Map) && pf.value.IsNil() {
		return nil, nil
	}

	value, err := pf.populateTo(pf.value)
	if err != nil {
		return nil, fmt.Errorf("error in PopulateTo for field %s for message %T: %s", pf.name, pf.msg, err)
	}
	return value, nil
}

type mapField struct {
	baseField
	populateFrom func(key string, value interface{}, destType reflect.Type) (reflect.Value, error)
	populateTo   func(key string, value reflect.Value) (interface{}, error)
}

func (mf *mapField) PopulateFrom(source interface{}) error {
	tree, ok := source.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map field %s for message %T to be assignable from map[string]interface{} but was not. Got %T", mf.name, mf.msg, source)
	}

	result := reflect.MakeMap(mf.vType)

	for k, v := range tree {
		if !reflect.TypeOf(v).AssignableTo(mf.fType) {
			return fmt.Errorf("expected map field %s value for %s for message %T to be assignable from %v but was not.  Is %T", mf.name, k, mf.msg, mf.fType, v)
		}
		newValue, err := mf.populateFrom(k, v, mf.vType.Elem())
		if err != nil {
			return fmt.Errorf("error in PopulateFrom for map field %s with key %s for message %T: %s", mf.name, k, mf.msg, err)
		}
		result.SetMapIndex(reflect.ValueOf(k), newValue)
	}

	mf.value.Set(result)
	return nil
}

func (mf *mapField) PopulateTo() (interface{}, error) {
	result := make(map[string]interface{})
	keys := mf.value.MapKeys()
	for _, key := range keys {
		k, ok := key.Interface().(string)
		if !ok {
			return nil, fmt.Errorf("expected map field %s for message %T to have string keys, but did not.", mf.name, mf.msg)
		}

		subValue := mf.value.MapIndex(key)
		kind := subValue.Type().Kind()
		if (kind == reflect.Ptr || kind == reflect.Slice || kind == reflect.Map) && subValue.IsNil() {
			continue
		}

		if !subValue.Type().AssignableTo(mf.vType.Elem()) {
			return nil, fmt.Errorf("expected map field %s with key %s for message %T to be assignable to %v but was not. Got %v.", mf.name, k, mf.msg, mf.vType.Elem(), subValue.Type())
		}

		value, err := mf.populateTo(k, subValue)
		if err != nil {
			return nil, fmt.Errorf("error in PopulateTo for map field %s and key %s for message %T: %s", mf.name, k, mf.msg, err)
		}
		result[k] = value
	}

	return result, nil
}

type sliceField struct {
	baseField
	populateTo   func(i int, source reflect.Value) (interface{}, error)
	populateFrom func(i int, source interface{}, destType reflect.Type) (reflect.Value, error)
}

func (sf *sliceField) PopulateFrom(source interface{}) error {
	slice, ok := source.([]interface{})
	if !ok {
		return fmt.Errorf("expected slice field %s for message %T to be assignable from []interface{} but was not. Got %T", sf.name, sf.msg, source)
	}

	result := reflect.MakeSlice(sf.vType, len(slice), len(slice))

	for i, v := range slice {
		if !reflect.TypeOf(v).AssignableTo(sf.fType) {
			return fmt.Errorf("expected slice field %s value at index %d for message %T to be assignable from %v but was not.  Is %T", sf.name, i, sf.msg, sf.fType, v)
		}
		subValue, err := sf.populateFrom(i, v, sf.vType.Elem())
		if err != nil {
			return fmt.Errorf("error in PopulateFrom for slice field %s at index %d for message %T: %s", sf.name, i, sf.msg, err)
		}
		result.Index(i).Set(subValue)
	}

	sf.value.Set(result)
	return nil
}

func (sf *sliceField) PopulateTo() (interface{}, error) {
	result := make([]interface{}, sf.value.Len())
	for i := range result {
		subValue := sf.value.Index(i)
		kind := subValue.Type().Kind()
		if (kind == reflect.Ptr || kind == reflect.Slice || kind == reflect.Map) && subValue.IsNil() {
			continue
		}

		if !subValue.Type().AssignableTo(sf.vType.Elem()) {
			return nil, fmt.Errorf("expected slice field %s at index %d for message %T to be assignable to %v but was not. Got %v.", sf.name, i, sf.msg, sf.vType.Elem(), subValue.Type())
		}

		value, err := sf.populateTo(i, subValue)
		if err != nil {
			return nil, fmt.Errorf("error in PopulateTo for slice field %s at index %d for message %T: %s", sf.name, i, sf.msg, err)
		}
		result[i] = value
	}

	return result, nil
}

func stringInSlice(target string, slice []string) bool {
	for _, name := range slice {
		if name == target {
			return true
		}
	}
	return false
}

//Protojson是围绕Protojson封送器的简单快捷包装器。
func protoToJSON(msg proto.Message) ([]byte, error) {
	if reflect.ValueOf(msg).IsNil() {
		panic("We're nil here")
	}
	var b bytes.Buffer
	m := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: true,
		Indent:       "  ",
		OrigName:     true,
	}
	err := m.Marshal(&b, msg)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func mapToProto(tree map[string]interface{}, msg proto.Message) error {
	jsonOut, err := json.Marshal(tree)
	if err != nil {
		return err
	}

	return jsonpb.UnmarshalString(string(jsonOut), msg)
}

//
//然后返回，或者错误
func jsonToMap(marshaled []byte) (map[string]interface{}, error) {
	tree := make(map[string]interface{})
	d := json.NewDecoder(bytes.NewReader(marshaled))
	d.UseNumber()
	err := d.Decode(&tree)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling intermediate JSON: %s", err)
	}
	return tree, nil
}

//工厂实现，按最贪婪到最少的顺序列出。
//列在较低位置的工厂，可能取决于列在较高位置的工厂
//先评估。
var fieldFactories = []protoFieldFactory{
	dynamicSliceFieldFactory{},
	dynamicMapFieldFactory{},
	dynamicFieldFactory{},
	variablyOpaqueSliceFieldFactory{},
	variablyOpaqueMapFieldFactory{},
	variablyOpaqueFieldFactory{},
	staticallyOpaqueSliceFieldFactory{},
	staticallyOpaqueMapFieldFactory{},
	staticallyOpaqueFieldFactory{},
	nestedSliceFieldFactory{},
	nestedMapFieldFactory{},
	nestedFieldFactory{},
}

func protoFields(msg proto.Message, uMsg proto.Message) ([]protoField, error) {
	var result []protoField

	pmVal := reflect.ValueOf(uMsg)
	if pmVal.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("expected proto.Message %T to be pointer kind", msg)
	}

	if pmVal.IsNil() {
		return nil, nil
	}

	mVal := pmVal.Elem()
	if mVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected proto.Message %T ptr value to be struct, was %v", uMsg, mVal.Kind())
	}

	iResult := make([][]protoField, len(fieldFactories))

	protoProps := proto.GetProperties(mVal.Type())
//TODO，这将跳过其中一个字段，这应该处理
//在某一点上是正确的
	for _, prop := range protoProps.Prop {
		fieldName := prop.OrigName
		fieldValue := mVal.FieldByName(prop.Name)
		fieldTypeStruct, ok := mVal.Type().FieldByName(prop.Name)
		if !ok {
			return nil, fmt.Errorf("programming error: proto does not have field advertised by proto package")
		}
		fieldType := fieldTypeStruct.Type

		for i, factory := range fieldFactories {
			if !factory.Handles(msg, fieldName, fieldType, fieldValue) {
				continue
			}

			field, err := factory.NewProtoField(msg, fieldName, fieldType, fieldValue)
			if err != nil {
				return nil, err
			}
			iResult[i] = append(iResult[i], field)
			break
		}
	}

//按相反顺序循环收集的字段，以便将它们收集到
//按照FieldFactories中的指定更正依赖项顺序
	for i := len(iResult) - 1; i >= 0; i-- {
		result = append(result, iResult[i]...)
	}

	return result, nil
}

func recursivelyCreateTreeFromMessage(msg proto.Message) (tree map[string]interface{}, err error) {
	defer func() {
//因为这个函数是递归的，所以很难确定哪个级别
//在产生错误的原型中，这个包装器留下面包屑用于调试
		if err != nil {
			err = fmt.Errorf("%T: %s", msg, err)
		}
	}()

	uMsg := msg
	decorated, ok := msg.(DecoratedProto)
	if ok {
		uMsg = decorated.Underlying()
	}

	fields, err := protoFields(msg, uMsg)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := protoToJSON(uMsg)
	if err != nil {
		return nil, err
	}

	tree, err = jsonToMap(jsonBytes)
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if _, ok := tree[field.Name()]; !ok {
			continue
		}
		delete(tree, field.Name())
		tree[field.Name()], err = field.PopulateTo()
		if err != nil {
			return nil, err
		}
	}

	return tree, nil
}

//deepmarshaljson将msg封送到w作为json，但不封送包含嵌套的字节字段
//将消息封送为base64（与标准协议编码类似），这些嵌套的消息将被重新标记
//作为这些消息的JSON表示。这样做是为了让JSON表示为非二进制的
//尽可能让人可读。
func DeepMarshalJSON(w io.Writer, msg proto.Message) error {
	root, err := recursivelyCreateTreeFromMessage(msg)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "\t")
	return encoder.Encode(root)
}

func recursivelyPopulateMessageFromTree(tree map[string]interface{}, msg proto.Message) (err error) {
	defer func() {
//因为这个函数是递归的，所以很难确定哪个级别
//
		if err != nil {
			err = fmt.Errorf("%T: %s", msg, err)
		}
	}()

	uMsg := msg
	decorated, ok := msg.(DecoratedProto)
	if ok {
		uMsg = decorated.Underlying()
	}

	fields, err := protoFields(msg, uMsg)
	if err != nil {
		return err
	}

	specialFieldsMap := make(map[string]interface{})

	for _, field := range fields {
		specialField, ok := tree[field.Name()]
		if !ok {
			continue
		}
		specialFieldsMap[field.Name()] = specialField
		delete(tree, field.Name())
	}

	if err = mapToProto(tree, uMsg); err != nil {
		return err
	}

	for _, field := range fields {
		specialField, ok := specialFieldsMap[field.Name()]
		if !ok {
			continue
		}
		if err := field.PopulateFrom(specialField); err != nil {
			return err
		}
	}

	return nil
}

//deepunmarshaljson接受deepmarshaljson生成的json输出并将其解码为msg
//这包括将扩展的嵌套元素重新封送为二进制形式
func DeepUnmarshalJSON(r io.Reader, msg proto.Message) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	root, err := jsonToMap(b)
	if err != nil {
		return err
	}

	return recursivelyPopulateMessageFromTree(root, msg)
}
