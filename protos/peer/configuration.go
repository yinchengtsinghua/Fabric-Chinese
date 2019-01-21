
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


package peer

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
)

func init() {
	common.ChannelGroupMap["Application"] = DynamicApplicationGroupFactory{}
}

type DynamicApplicationGroupFactory struct{}

func (dagf DynamicApplicationGroupFactory) DynamicConfigGroup(cg *common.ConfigGroup) proto.Message {
	return &DynamicApplicationGroup{
		ConfigGroup: cg,
	}
}

type DynamicApplicationGroup struct {
	*common.ConfigGroup
}

func (dag *DynamicApplicationGroup) DynamicMapFieldProto(name string, key string, base proto.Message) (proto.Message, error) {
	switch name {
	case "groups":
		cg, ok := base.(*common.ConfigGroup)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup groups can only contain ConfigGroup messages")
		}

		return &DynamicApplicationOrgGroup{
			ConfigGroup: cg,
		}, nil
	case "values":
		cv, ok := base.(*common.ConfigValue)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup values can only contain ConfigValue messages")
		}
		return &DynamicApplicationConfigValue{
			ConfigValue: cv,
			name:        key,
		}, nil
	default:
		return nil, fmt.Errorf("ConfigGroup does not have a dynamic field: %s", name)
	}
}

type DynamicApplicationOrgGroup struct {
	*common.ConfigGroup
}

func (dag *DynamicApplicationOrgGroup) DynamicMapFieldProto(name string, key string, base proto.Message) (proto.Message, error) {
	switch name {
	case "groups":
		return nil, fmt.Errorf("The application orgs do not support sub-groups")
	case "values":
		cv, ok := base.(*common.ConfigValue)
		if !ok {
			return nil, fmt.Errorf("ConfigGroup values can only contain ConfigValue messages")
		}

		return &DynamicApplicationOrgConfigValue{
			ConfigValue: cv,
			name:        key,
		}, nil
	default:
		return nil, fmt.Errorf("Not a dynamic application map field: %s", name)
	}
}

type DynamicApplicationConfigValue struct {
	*common.ConfigValue
	name string
}

func (ccv *DynamicApplicationConfigValue) VariablyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != ccv.VariablyOpaqueFields()[0] {
		return nil, fmt.Errorf("Not a marshaled field: %s", name)
	}
	switch ccv.name {
	case "Capabilities":
		return &common.Capabilities{}, nil
	case "ACLs":
		return &ACLs{}, nil
	default:
		return nil, fmt.Errorf("Unknown Application ConfigValue name: %s", ccv.name)
	}
}

type DynamicApplicationOrgConfigValue struct {
	*common.ConfigValue
	name string
}

func (daocv *DynamicApplicationOrgConfigValue) VariablyOpaqueFieldProto(name string) (proto.Message, error) {
	if name != daocv.VariablyOpaqueFields()[0] {
		return nil, fmt.Errorf("Not a marshaled field: %s", name)
	}
	switch daocv.name {
	case "MSP":
		return &msp.MSPConfig{}, nil
	case "AnchorPeers":
		return &AnchorPeers{}, nil
	default:
		return nil, fmt.Errorf("Unknown Application Org ConfigValue name: %s", daocv.name)
	}
}
