
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

func opaqueFrom(opaqueType func() (proto.Message, error), value interface{}, destType reflect.Type) (reflect.Value, error) {
tree := value.(map[string]interface{}) //安全，已检查
	nMsg, err := opaqueType()
	if err != nil {
		return reflect.Value{}, err
	}
	if err := recursivelyPopulateMessageFromTree(tree, nMsg); err != nil {
		return reflect.Value{}, err
	}
	mMsg, err := MostlyDeterministicMarshal(nMsg)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(mMsg), nil
}

func opaqueTo(opaqueType func() (proto.Message, error), value reflect.Value) (interface{}, error) {
	nMsg, err := opaqueType()
	if err != nil {
		return nil, err
	}
mMsg := value.Interface().([]byte) //安全，已检查
	if err = proto.Unmarshal(mMsg, nMsg); err != nil {
		return nil, err
	}
	return recursivelyCreateTreeFromMessage(nMsg)
}

type staticallyOpaqueFieldFactory struct{}

func (soff staticallyOpaqueFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	opaqueProto, ok := msg.(StaticallyOpaqueFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, opaqueProto.StaticallyOpaqueFields())
}

func (soff staticallyOpaqueFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
opaqueProto := msg.(StaticallyOpaqueFieldProto) //键入签入句柄

	return &plainField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: bytesType,
			value: fieldValue,
		},
		populateFrom: func(v interface{}, dT reflect.Type) (reflect.Value, error) {
			return opaqueFrom(func() (proto.Message, error) { return opaqueProto.StaticallyOpaqueFieldProto(fieldName) }, v, dT)
		},
		populateTo: func(v reflect.Value) (interface{}, error) {
			return opaqueTo(func() (proto.Message, error) { return opaqueProto.StaticallyOpaqueFieldProto(fieldName) }, v)
		},
	}, nil
}

type staticallyOpaqueMapFieldFactory struct{}

func (soff staticallyOpaqueMapFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	opaqueProto, ok := msg.(StaticallyOpaqueMapFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, opaqueProto.StaticallyOpaqueMapFields())
}

func (soff staticallyOpaqueMapFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
opaqueProto := msg.(StaticallyOpaqueMapFieldProto) //键入签入句柄

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
				return opaqueProto.StaticallyOpaqueMapFieldProto(fieldName, key)
			}, v, dT)
		},
		populateTo: func(key string, v reflect.Value) (interface{}, error) {
			return opaqueTo(func() (proto.Message, error) {
				return opaqueProto.StaticallyOpaqueMapFieldProto(fieldName, key)
			}, v)
		},
	}, nil
}

type staticallyOpaqueSliceFieldFactory struct{}

func (soff staticallyOpaqueSliceFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	opaqueProto, ok := msg.(StaticallyOpaqueSliceFieldProto)
	if !ok {
		return false
	}

	return stringInSlice(fieldName, opaqueProto.StaticallyOpaqueSliceFields())
}

func (soff staticallyOpaqueSliceFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
opaqueProto := msg.(StaticallyOpaqueSliceFieldProto) //键入签入句柄

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
				return opaqueProto.StaticallyOpaqueSliceFieldProto(fieldName, index)
			}, v, dT)
		},
		populateTo: func(index int, v reflect.Value) (interface{}, error) {
			return opaqueTo(func() (proto.Message, error) {
				return opaqueProto.StaticallyOpaqueSliceFieldProto(fieldName, index)
			}, v)
		},
	}, nil
}
