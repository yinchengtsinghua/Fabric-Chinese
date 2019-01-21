
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


package policies

import (
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

//
type ConfigPolicy interface {
//key是密钥该值应存储在*cb.configggroup.policies映射中。
	Key() string

//值是此配置策略的支持策略实现
	Value() *cb.Policy
}

//StandardConfigValue实现ConfigValue接口。
type StandardConfigPolicy struct {
	key   string
	value *cb.Policy
}

//key是键，该值应存储在*cb.configggroup.values映射中。
func (scv *StandardConfigPolicy) Key() string {
	return scv.key
}

//
func (scv *StandardConfigPolicy) Value() *cb.Policy {
	return scv.value
}

func makeImplicitMetaPolicy(subPolicyName string, rule cb.ImplicitMetaPolicy_Rule) *cb.Policy {
	return &cb.Policy{
		Type: int32(cb.Policy_IMPLICIT_META),
		Value: utils.MarshalOrPanic(&cb.ImplicitMetaPolicy{
			Rule:      rule,
			SubPolicy: subPolicyName,
		}),
	}
}

//implicit meta all policy定义了一个隐式元策略，其子策略和密钥是policyName和rule all。
func ImplicitMetaAllPolicy(policyName string) *StandardConfigPolicy {
	return &StandardConfigPolicy{
		key:   policyName,
		value: makeImplicitMetaPolicy(policyName, cb.ImplicitMetaPolicy_ALL),
	}
}

//implicit meta any policy定义了一个隐式元策略，其子策略和密钥为policyName，规则为any。
func ImplicitMetaAnyPolicy(policyName string) *StandardConfigPolicy {
	return &StandardConfigPolicy{
		key:   policyName,
		value: makeImplicitMetaPolicy(policyName, cb.ImplicitMetaPolicy_ANY),
	}
}

//
func ImplicitMetaMajorityPolicy(policyName string) *StandardConfigPolicy {
	return &StandardConfigPolicy{
		key:   policyName,
		value: makeImplicitMetaPolicy(policyName, cb.ImplicitMetaPolicy_MAJORITY),
	}
}

//ImplicitMetamajorityPolicy定义具有密钥policyName和给定签名策略的策略。
func SignaturePolicy(policyName string, sigPolicy *cb.SignaturePolicyEnvelope) *StandardConfigPolicy {
	return &StandardConfigPolicy{
		key: policyName,
		value: &cb.Policy{
			Type:  int32(cb.Policy_SIGNATURE),
			Value: utils.MarshalOrPanic(sigPolicy),
		},
	}
}
