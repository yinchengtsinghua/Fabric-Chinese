
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
	"github.com/hyperledger/fabric/protos/msp"
)

type DynamicConfigGroupFactory interface {
	DynamicConfigGroup(cg *ConfigGroup) proto.Message
}

//channelgroupmap是打破依赖循环的一个稍微有点笨拙的方法，这将
//如果protos/common包导入protos/order或protos/peer，则创建
//包装。这些程序包将在
//当它们被加载时，在两者之间创建一个运行时链接
var ChannelGroupMap = map[string]DynamicConfigGroupFactory{
	"Consortiums": DynamicConsortiumsGroupFactory{},
}

type DynamicChannelGroup struct {
	*ConfigGroup
}

func (dcg *DynamicChannelGroup) DynamicMapFieldProto(name string, key string, base proto.Message) (proto.Message, error) {
	switch name {
	case "groups":
		cg, ok := base.(*ConfigGroup)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup groups can only contain ConfigGroup messages")
		}

		dcgf, ok := ChannelGroupMap[key]
		if !ok {
			return nil, fmt.Errorf("unknown channel ConfigGroup sub-group: %s", key)
		}
		return dcgf.DynamicConfigGroup(cg), nil
	case "values":
		cv, ok := base.(*ConfigValue)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup values can only contain ConfigValue messages")
		}
		return &DynamicChannelConfigValue{
			ConfigValue: cv,
			name:        key,
		}, nil
	default:
		return nil, fmt.Errorf("ConfigGroup does not have a dynamic field: %s", name)
	}
}

type DynamicChannelConfigValue struct {
	*ConfigValue
	name string
}

func (dccv *DynamicChannelConfigValue) VariablyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != dccv.VariablyOpaqueFields()[0] {
		return nil, fmt.Errorf("not a marshaled field: %s", name)
	}
	switch dccv.name {
	case "HashingAlgorithm":
		return &HashingAlgorithm{}, nil
	case "BlockDataHashingStructure":
		return &BlockDataHashingStructure{}, nil
	case "OrdererAddresses":
		return &OrdererAddresses{}, nil
	case "Consortium":
		return &Consortium{}, nil
	case "Capabilities":
		return &Capabilities{}, nil
	default:
		return nil, fmt.Errorf("unknown Channel ConfigValue name: %s", dccv.name)
	}
}

func (dccv *DynamicChannelConfigValue) Underlying() proto.Message {
	return dccv.ConfigValue
}

type DynamicConsortiumsGroupFactory struct{}

func (dogf DynamicConsortiumsGroupFactory) DynamicConfigGroup(cg *ConfigGroup) proto.Message {
	return &DynamicConsortiumsGroup{
		ConfigGroup: cg,
	}
}

type DynamicConsortiumsGroup struct {
	*ConfigGroup
}

func (dcg *DynamicConsortiumsGroup) DynamicMapFieldProto(name string, key string, base proto.Message) (proto.Message, error) {
	switch name {
	case "groups":
		cg, ok := base.(*ConfigGroup)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup groups can only contain ConfigGroup messages")
		}

		return &DynamicConsortiumGroup{
			ConfigGroup: cg,
		}, nil
	case "values":
		return nil, fmt.Errorf("Consortiums currently support no config values")
	default:
		return nil, fmt.Errorf("ConfigGroup does not have a dynamic field: %s", name)
	}
}

func (dcg *DynamicConsortiumsGroup) Underlying() proto.Message {
	return dcg.ConfigGroup
}

type DynamicConsortiumGroup struct {
	*ConfigGroup
}

func (dcg *DynamicConsortiumGroup) DynamicMapFieldProto(name string, key string, base proto.Message) (proto.Message, error) {
	switch name {
	case "groups":
		cg, ok := base.(*ConfigGroup)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup groups can only contain ConfigGroup messages")
		}
		return &DynamicConsortiumOrgGroup{
			ConfigGroup: cg,
		}, nil
	case "values":
		cv, ok := base.(*ConfigValue)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup values can only contain ConfigValue messages")
		}

		return &DynamicConsortiumConfigValue{
			ConfigValue: cv,
			name:        key,
		}, nil
	default:
		return nil, fmt.Errorf("not a dynamic orderer map field: %s", name)
	}
}

func (dcg *DynamicConsortiumGroup) Underlying() proto.Message {
	return dcg.ConfigGroup
}

type DynamicConsortiumConfigValue struct {
	*ConfigValue
	name string
}

func (dccv *DynamicConsortiumConfigValue) VariablyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != dccv.VariablyOpaqueFields()[0] {
		return nil, fmt.Errorf("not a marshaled field: %s", name)
	}
	switch dccv.name {
	case "ChannelCreationPolicy":
		return &Policy{}, nil
	default:
		return nil, fmt.Errorf("unknown Consortium ConfigValue name: %s", dccv.name)
	}
}

type DynamicConsortiumOrgGroup struct {
	*ConfigGroup
}

func (dcg *DynamicConsortiumOrgGroup) DynamicMapFieldProto(name string, key string, base proto.Message) (proto.Message, error) {
	switch name {
	case "groups":
		return nil, fmt.Errorf("ConsortiumOrg groups do not support sub groups")
	case "values":
		cv, ok := base.(*ConfigValue)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup values can only contain ConfigValue messages")
		}

		return &DynamicConsortiumOrgConfigValue{
			ConfigValue: cv,
			name:        key,
		}, nil
	default:
		return nil, fmt.Errorf("not a dynamic orderer map field: %s", name)
	}
}

type DynamicConsortiumOrgConfigValue struct {
	*ConfigValue
	name string
}

func (dcocv *DynamicConsortiumOrgConfigValue) VariablyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != dcocv.VariablyOpaqueFields()[0] {
		return nil, fmt.Errorf("not a marshaled field: %s", name)
	}
	switch dcocv.name {
	case "MSP":
		return &msp.MSPConfig{}, nil
	default:
		return nil, fmt.Errorf("unknown Consortium Org ConfigValue name: %s", dcocv.name)
	}
}
