
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
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

const (
//channelCreationPolicyKey是联合体配置中用于表示策略的密钥。
//用于评估通道创建请求是否被授权
	ChannelCreationPolicyKey = "ChannelCreationPolicy"
)

//联合体代理持有联合体配置的配置协议
type ConsortiumProtos struct {
	ChannelCreationPolicy *cb.Policy
}

//联合配置保存联合配置信息
type ConsortiumConfig struct {
	protos *ConsortiumProtos
	orgs   map[string]Org
}

//newconsortiumconfig创建联合体配置的新实例
func NewConsortiumConfig(consortiumGroup *cb.ConfigGroup, mspConfig *MSPConfigHandler) (*ConsortiumConfig, error) {
	cc := &ConsortiumConfig{
		protos: &ConsortiumProtos{},
		orgs:   make(map[string]Org),
	}

	if err := DeserializeProtoValuesFromGroup(consortiumGroup, cc.protos); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize values")
	}

	for orgName, orgGroup := range consortiumGroup.Groups {
		var err error
		if cc.orgs[orgName], err = NewOrganizationConfig(orgName, orgGroup, mspConfig); err != nil {
			return nil, err
		}
	}

	return cc, nil
}

//组织返回联合体中的一组组织
func (cc *ConsortiumConfig) Organizations() map[string]Org {
	return cc.orgs
}

//creationpolicy返回用于验证的策略结构
//渠道创建
func (cc *ConsortiumConfig) ChannelCreationPolicy() *cb.Policy {
	return cc.protos.ChannelCreationPolicy
}
