
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


package common

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

func NewConfigGroup() *ConfigGroup {
	return &ConfigGroup{
		Groups:   make(map[string]*ConfigGroup),
		Values:   make(map[string]*ConfigValue),
		Policies: make(map[string]*ConfigPolicy),
	}
}

func (cue *ConfigUpdateEnvelope) StaticallyOpaqueFields() []string {
	return []string{"config_update"}
}

func (cue *ConfigUpdateEnvelope) StaticallyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != cue.StaticallyOpaqueFields()[0] {
		return nil, fmt.Errorf("Not a marshaled field: %s", name)
	}
	return &ConfigUpdate{}, nil
}

func (cs *ConfigSignature) StaticallyOpaqueFields() []string {
	return []string{"signature_header"}
}

func (cs *ConfigSignature) StaticallyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != cs.StaticallyOpaqueFields()[0] {
		return nil, fmt.Errorf("Not a marshaled field: %s", name)
	}
	return &SignatureHeader{}, nil
}

func (c *Config) DynamicFields() []string {
	return []string{"channel_group"}
}

func (c *Config) DynamicFieldProto(name string, base proto.Message) (proto.Message, error) {
	if name != c.DynamicFields()[0] {
		return nil, fmt.Errorf("Not a dynamic field: %s", name)
	}

	cg, ok := base.(*ConfigGroup)
	if !ok {
		return nil, fmt.Errorf("Config must embed a config group as its dynamic field")
	}

	return &DynamicChannelGroup{ConfigGroup: cg}, nil
}

//configupdateisolateddatatypes允许其他proto包为
//独立的_数据字段。这是打破进口循环的必要条件。
var ConfigUpdateIsolatedDataTypes = map[string]func(string) proto.Message{}

func (c *ConfigUpdate) StaticallyOpaqueMapFields() []string {
	return []string{"isolated_data"}
}

func (c *ConfigUpdate) StaticallyOpaqueMapFieldProto(name string, key string) (proto.Message, error) {
	if name != c.StaticallyOpaqueMapFields()[0] {
		return nil, fmt.Errorf("Not a statically opaque map field: %s", name)
	}

	mf, ok := ConfigUpdateIsolatedDataTypes[key]
	if !ok {
		return nil, fmt.Errorf("Unknown map key: %s", key)
	}

	return mf(key), nil
}

func (c *ConfigUpdate) DynamicFields() []string {
	return []string{"read_set", "write_set"}
}

func (c *ConfigUpdate) DynamicFieldProto(name string, base proto.Message) (proto.Message, error) {
	if name != c.DynamicFields()[0] && name != c.DynamicFields()[1] {
		return nil, fmt.Errorf("Not a dynamic field: %s", name)
	}

	cg, ok := base.(*ConfigGroup)
	if !ok {
		return nil, fmt.Errorf("Expected base to be *ConfigGroup, got %T", base)
	}

	return &DynamicChannelGroup{ConfigGroup: cg}, nil
}

func (cv *ConfigValue) VariablyOpaqueFields() []string {
	return []string{"value"}
}

func (cv *ConfigValue) Underlying() proto.Message {
	return cv
}

func (cg *ConfigGroup) DynamicMapFields() []string {
	return []string{"groups", "values"}
}

func (cg *ConfigGroup) Underlying() proto.Message {
	return cg
}
