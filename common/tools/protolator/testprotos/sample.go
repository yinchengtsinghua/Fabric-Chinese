
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


package testprotos

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

func (som *StaticallyOpaqueMsg) StaticallyOpaqueFields() []string {
	return []string{"plain_opaque_field"}
}

func (som *StaticallyOpaqueMsg) StaticallyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != som.StaticallyOpaqueFields()[0] {
		return nil, fmt.Errorf("not a statically opaque field: %s", name)
	}

	return &SimpleMsg{}, nil
}

func (som *StaticallyOpaqueMsg) StaticallyOpaqueMapFields() []string {
	return []string{"map_opaque_field"}
}

func (som *StaticallyOpaqueMsg) StaticallyOpaqueMapFieldProto(name string, key string) (proto.Message, error) {
	if name != som.StaticallyOpaqueMapFields()[0] {
		return nil, fmt.Errorf("not a statically opaque field: %s", name)
	}

	return &SimpleMsg{}, nil
}

func (som *StaticallyOpaqueMsg) StaticallyOpaqueSliceFields() []string {
	return []string{"slice_opaque_field"}
}

func (som *StaticallyOpaqueMsg) StaticallyOpaqueSliceFieldProto(name string, index int) (proto.Message, error) {
	if name != som.StaticallyOpaqueSliceFields()[0] {
		return nil, fmt.Errorf("not a statically opaque field: %s", name)
	}

	return &SimpleMsg{}, nil
}

func typeSwitch(typeName string) (proto.Message, error) {
	switch typeName {
	case "SimpleMsg":
		return &SimpleMsg{}, nil
	case "NestedMsg":
		return &NestedMsg{}, nil
	case "StaticallyOpaqueMsg":
		return &StaticallyOpaqueMsg{}, nil
	case "VariablyOpaqueMsg":
		return &VariablyOpaqueMsg{}, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", typeName)
	}
}

func (vom *VariablyOpaqueMsg) VariablyOpaqueFields() []string {
	return []string{"plain_opaque_field"}
}

func (vom *VariablyOpaqueMsg) VariablyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != vom.VariablyOpaqueFields()[0] {
		return nil, fmt.Errorf("not a statically opaque field: %s", name)
	}

	return typeSwitch(vom.OpaqueType)
}

func (vom *VariablyOpaqueMsg) VariablyOpaqueMapFields() []string {
	return []string{"map_opaque_field"}
}

func (vom *VariablyOpaqueMsg) VariablyOpaqueMapFieldProto(name string, key string) (proto.Message, error) {
	if name != vom.VariablyOpaqueMapFields()[0] {
		return nil, fmt.Errorf("not a statically opaque field: %s", name)
	}

	return typeSwitch(vom.OpaqueType)
}

func (vom *VariablyOpaqueMsg) VariablyOpaqueSliceFields() []string {
	return []string{"slice_opaque_field"}
}

func (vom *VariablyOpaqueMsg) VariablyOpaqueSliceFieldProto(name string, index int) (proto.Message, error) {
	if name != vom.VariablyOpaqueSliceFields()[0] {
		return nil, fmt.Errorf("not a statically opaque field: %s", name)
	}

	return typeSwitch(vom.OpaqueType)
}

func (cm *ContextlessMsg) VariablyOpaqueFields() []string {
	return []string{"opaque_field"}
}

type DynamicMessageWrapper struct {
	*ContextlessMsg
	typeName string
}

func (dmw *DynamicMessageWrapper) VariablyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != dmw.ContextlessMsg.VariablyOpaqueFields()[0] {
		return nil, fmt.Errorf("not a statically opaque field: %s", name)
	}

	return typeSwitch(dmw.typeName)
}

func (dmw *DynamicMessageWrapper) Underlying() proto.Message {
	return dmw.ContextlessMsg
}

func wrapContextless(underlying proto.Message, typeName string) (*DynamicMessageWrapper, error) {
	cm, ok := underlying.(*ContextlessMsg)
	if !ok {
		return nil, fmt.Errorf("unknown dynamic message to wrap (%T) requires *ContextlessMsg", underlying)
	}

	return &DynamicMessageWrapper{
		ContextlessMsg: cm,
		typeName:       typeName,
	}, nil
}

func (vom *DynamicMsg) DynamicFields() []string {
	return []string{"plain_dynamic_field"}
}

func (vom *DynamicMsg) DynamicFieldProto(name string, underlying proto.Message) (proto.Message, error) {
	if name != vom.DynamicFields()[0] {
		return nil, fmt.Errorf("not a dynamic field: %s", name)
	}

	return wrapContextless(underlying, vom.DynamicType)
}

func (vom *DynamicMsg) DynamicMapFields() []string {
	return []string{"map_dynamic_field"}
}

func (vom *DynamicMsg) DynamicMapFieldProto(name string, key string, underlying proto.Message) (proto.Message, error) {
	if name != vom.DynamicMapFields()[0] {
		return nil, fmt.Errorf("not a dynamic map field: %s", name)
	}

	return wrapContextless(underlying, vom.DynamicType)
}

func (vom *DynamicMsg) DynamicSliceFields() []string {
	return []string{"slice_dynamic_field"}
}

func (vom *DynamicMsg) DynamicSliceFieldProto(name string, index int, underlying proto.Message) (proto.Message, error) {
	if name != vom.DynamicSliceFields()[0] {
		return nil, fmt.Errorf("not a dynamic slice field: %s", name)
	}

	return wrapContextless(underlying, vom.DynamicType)
}

func (udf *UnmarshalableDeepFields) StaticallyOpaqueFields() []string {
	return []string{"plain_opaque_field"}
}

func (udf *UnmarshalableDeepFields) StaticallyOpaqueFieldProto(name string) (proto.Message, error) {
	return nil, fmt.Errorf("intentional error")
}

func (udf *UnmarshalableDeepFields) StaticallyOpaqueMapFields() []string {
	return []string{"map_opaque_field"}
}

func (udf *UnmarshalableDeepFields) StaticallyOpaqueMapFieldProto(name, key string) (proto.Message, error) {
	return nil, fmt.Errorf("intentional error")
}

func (udf *UnmarshalableDeepFields) StaticallyOpaqueSliceFields() []string {
	return []string{"slice_opaque_field"}
}

func (udf *UnmarshalableDeepFields) StaticallyOpaqueSliceFieldProto(name string, index int) (proto.Message, error) {
	return nil, fmt.Errorf("intentional error")
}
