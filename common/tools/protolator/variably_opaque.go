
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

type variablyOpaqueFieldFactory struct{}

func (soff variablyOpaqueFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	opaqueProto, ok := msg.(VariablyOpaqueFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, opaqueProto.VariablyOpaqueFields())
}

func (soff variablyOpaqueFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
opaqueProto := msg.(VariablyOpaqueFieldProto) //键入签入句柄

	return &plainField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: bytesType,
			value: fieldValue,
		},
		populateFrom: func(v interface{}, dT reflect.Type) (reflect.Value, error) {
			return opaqueFrom(func() (proto.Message, error) { return opaqueProto.VariablyOpaqueFieldProto(fieldName) }, v, dT)
		},
		populateTo: func(v reflect.Value) (interface{}, error) {
			return opaqueTo(func() (proto.Message, error) { return opaqueProto.VariablyOpaqueFieldProto(fieldName) }, v)
		},
	}, nil
}

type variablyOpaqueMapFieldFactory struct{}

func (soff variablyOpaqueMapFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	opaqueProto, ok := msg.(VariablyOpaqueMapFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, opaqueProto.VariablyOpaqueMapFields())
}

func (soff variablyOpaqueMapFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
opaqueProto := msg.(VariablyOpaqueMapFieldProto) //键入签入句柄

	return &mapField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(key string, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return opaqueFrom(func() (proto.Message, error) {
				return opaqueProto.VariablyOpaqueMapFieldProto(fieldName, key)
			}, v, dT)
		},
		populateTo: func(key string, v reflect.Value) (interface{}, error) {
			return opaqueTo(func() (proto.Message, error) {
				return opaqueProto.VariablyOpaqueMapFieldProto(fieldName, key)
			}, v)
		},
	}, nil
}

type variablyOpaqueSliceFieldFactory struct{}

func (soff variablyOpaqueSliceFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	opaqueProto, ok := msg.(VariablyOpaqueSliceFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, opaqueProto.VariablyOpaqueSliceFields())
}

func (soff variablyOpaqueSliceFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
opaqueProto := msg.(VariablyOpaqueSliceFieldProto) //键入签入句柄

	return &sliceField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(index int, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return opaqueFrom(func() (proto.Message, error) {
				return opaqueProto.VariablyOpaqueSliceFieldProto(fieldName, index)
			}, v, dT)
		},
		populateTo: func(index int, v reflect.Value) (interface{}, error) {
			return opaqueTo(func() (proto.Message, error) {
				return opaqueProto.VariablyOpaqueSliceFieldProto(fieldName, index)
			}, v)
		},
	}, nil
}
