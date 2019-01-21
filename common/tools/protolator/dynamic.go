
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
	"reflect"

	"github.com/golang/protobuf/proto"
)

func dynamicFrom(dynamicMsg func(underlying proto.Message) (proto.Message, error), value interface{}, destType reflect.Type) (reflect.Value, error) {
tree := value.(map[string]interface{}) //安全，已检查
	uMsg := reflect.New(destType.Elem())
nMsg, err := dynamicMsg(uMsg.Interface().(proto.Message)) //安全，已检查
	if err != nil {
		return reflect.Value{}, err
	}
	if err := recursivelyPopulateMessageFromTree(tree, nMsg); err != nil {
		return reflect.Value{}, err
	}
	return uMsg, nil
}

func dynamicTo(dynamicMsg func(underlying proto.Message) (proto.Message, error), value reflect.Value) (interface{}, error) {
nMsg, err := dynamicMsg(value.Interface().(proto.Message)) //安全，已检查
	if err != nil {
		return nil, err
	}
	return recursivelyCreateTreeFromMessage(nMsg)
}

type dynamicFieldFactory struct{}

func (dff dynamicFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	dynamicProto, ok := msg.(DynamicFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, dynamicProto.DynamicFields())
}

func (dff dynamicFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
dynamicProto, _ := msg.(DynamicFieldProto) //键入签入句柄

	return &plainField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(v interface{}, dT reflect.Type) (reflect.Value, error) {
			return dynamicFrom(func(underlying proto.Message) (proto.Message, error) {
				return dynamicProto.DynamicFieldProto(fieldName, underlying)
			}, v, dT)
		},
		populateTo: func(v reflect.Value) (interface{}, error) {
			return dynamicTo(func(underlying proto.Message) (proto.Message, error) {
				return dynamicProto.DynamicFieldProto(fieldName, underlying)
			}, v)
		},
	}, nil
}

type dynamicMapFieldFactory struct{}

func (dmff dynamicMapFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	dynamicProto, ok := msg.(DynamicMapFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, dynamicProto.DynamicMapFields())
}

func (dmff dynamicMapFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
dynamicProto := msg.(DynamicMapFieldProto) //类型由句柄检查

	return &mapField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(k string, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return dynamicFrom(func(underlying proto.Message) (proto.Message, error) {
				return dynamicProto.DynamicMapFieldProto(fieldName, k, underlying)
			}, v, dT)
		},
		populateTo: func(k string, v reflect.Value) (interface{}, error) {
			return dynamicTo(func(underlying proto.Message) (proto.Message, error) {
				return dynamicProto.DynamicMapFieldProto(fieldName, k, underlying)
			}, v)
		},
	}, nil
}

type dynamicSliceFieldFactory struct{}

func (dmff dynamicSliceFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	dynamicProto, ok := msg.(DynamicSliceFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, dynamicProto.DynamicSliceFields())
}

func (dmff dynamicSliceFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
dynamicProto := msg.(DynamicSliceFieldProto) //类型由句柄检查

	return &sliceField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(i int, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return dynamicFrom(func(underlying proto.Message) (proto.Message, error) {
				return dynamicProto.DynamicSliceFieldProto(fieldName, i, underlying)
			}, v, dT)
		},
		populateTo: func(i int, v reflect.Value) (interface{}, error) {
			return dynamicTo(func(underlying proto.Message) (proto.Message, error) {
				return dynamicProto.DynamicSliceFieldProto(fieldName, i, underlying)
			}, v)
		},
	}, nil
}
