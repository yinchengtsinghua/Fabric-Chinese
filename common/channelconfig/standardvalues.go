
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


package channelconfig

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
)

//反序列化组反序列化配置组中所有值的值
func DeserializeProtoValuesFromGroup(group *cb.ConfigGroup, protosStructs ...interface{}) error {
	sv, err := NewStandardValues(protosStructs...)
	if err != nil {
		logger.Panicf("This is a compile time bug only, the proto structures are somehow invalid: %s", err)
	}

	for key, value := range group.Values {
		if _, err := sv.Deserialize(key, value.Value); err != nil {
			return err
		}
	}
	return nil
}

type StandardValues struct {
	lookup map[string]proto.Message
}

//newStandardValues接受只能包含protobuf消息的结构
//类型。该结构可以嵌入满足以下条件的其他（非指针）结构
//同样的条件。新标准值将为所有协议实例化内存
//消息并构建从结构字段名到原型消息实例的查找映射
//这是一种很容易实现值接口的有用方法
func NewStandardValues(protosStructs ...interface{}) (*StandardValues, error) {
	sv := &StandardValues{
		lookup: make(map[string]proto.Message),
	}

	for _, protosStruct := range protosStructs {
		logger.Debugf("Initializing protos for %T\n", protosStruct)
		if err := sv.initializeProtosStruct(reflect.ValueOf(protosStruct)); err != nil {
			return nil, err
		}
	}

	return sv, nil
}

//反序列化查找给定名称的支持值proto，取消对给定字节的标记
//填充支持消息结构，并返回对保留的反序列化的引用
//消息（或错误，原因可能是密钥不存在，或者解组时出错）
func (sv *StandardValues) Deserialize(key string, value []byte) (proto.Message, error) {
	msg, ok := sv.lookup[key]
	if !ok {
		return nil, fmt.Errorf("Unexpected key %s", key)
	}

	err := proto.Unmarshal(value, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (sv *StandardValues) initializeProtosStruct(objValue reflect.Value) error {
	objType := objValue.Type()
	if objType.Kind() != reflect.Ptr {
		return fmt.Errorf("Non pointer type")
	}
	if objType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Non struct type")
	}

	numFields := objValue.Elem().NumField()
	for i := 0; i < numFields; i++ {
		structField := objType.Elem().Field(i)
		logger.Debugf("Processing field: %s\n", structField.Name)
		switch structField.Type.Kind() {
		case reflect.Ptr:
			fieldPtr := objValue.Elem().Field(i)
			if !fieldPtr.CanSet() {
				return fmt.Errorf("Cannot set structure field %s (unexported?)", structField.Name)
			}
			fieldPtr.Set(reflect.New(structField.Type.Elem()))
		default:
			return fmt.Errorf("Bad type supplied: %s", structField.Type.Kind())
		}

		proto, ok := objValue.Elem().Field(i).Interface().(proto.Message)
		if !ok {
			return fmt.Errorf("Field type %T does not implement proto.Message", objValue.Elem().Field(i))
		}

		_, ok = sv.lookup[structField.Name]
		if ok {
			return fmt.Errorf("Ambiguous field name specified, multiple occurrences of %s", structField.Name)
		}

		sv.lookup[structField.Name] = proto
	}

	return nil
}
