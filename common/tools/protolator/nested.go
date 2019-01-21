
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
	"github.com/golang/protobuf/ptypes/timestamp"
)

func nestedFrom(value interface{}, destType reflect.Type) (reflect.Value, error) {
tree := value.(map[string]interface{}) //安全，已检查
	result := reflect.New(destType.Elem())
nMsg := result.Interface().(proto.Message) //安全，已检查
	if err := recursivelyPopulateMessageFromTree(tree, nMsg); err != nil {
		return reflect.Value{}, err
	}
	return result, nil
}

func nestedTo(value reflect.Value) (interface{}, error) {
nMsg := value.Interface().(proto.Message) //安全，已检查
	return recursivelyCreateTreeFromMessage(nMsg)
}

var timestampType = reflect.TypeOf(&timestamp.Timestamp{})

type nestedFieldFactory struct{}

func (nff nestedFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
//注意，如果字段是原生时间戳，我们将跳过递归到该字段中，因为存在与此冲突的其他自定义封送处理
//应该更广泛地重新访问这一点，以防止对“已知消息”进行自定义封送处理。
	return fieldType.Kind() == reflect.Ptr && fieldType.AssignableTo(protoMsgType) && !fieldType.AssignableTo(timestampType)
}

func (nff nestedFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &plainField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: nestedFrom,
		populateTo:   nestedTo,
	}, nil
}

type nestedMapFieldFactory struct{}

func (nmff nestedMapFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	return fieldType.Kind() == reflect.Map && fieldType.Elem().AssignableTo(protoMsgType) && !fieldType.Elem().AssignableTo(timestampType) && fieldType.Key().Kind() == reflect.String
}

func (nmff nestedMapFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &mapField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(k string, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return nestedFrom(v, dT)
		},
		populateTo: func(k string, v reflect.Value) (interface{}, error) {
			return nestedTo(v)
		},
	}, nil
}

type nestedSliceFieldFactory struct{}

func (nmff nestedSliceFieldFactory) Handles(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) bool {
	return fieldType.Kind() == reflect.Slice && fieldType.Elem().AssignableTo(protoMsgType) && !fieldType.Elem().AssignableTo(timestampType)
}

func (nmff nestedSliceFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &sliceField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(i int, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return nestedFrom(v, dT)
		},
		populateTo: func(i int, v reflect.Value) (interface{}, error) {
			return nestedTo(v)
		},
	}, nil
}
