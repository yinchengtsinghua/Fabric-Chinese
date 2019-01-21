
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


package cauthdsl

import (
	"fmt"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	mb "github.com/hyperledger/fabric/protos/msp"
	"go.uber.org/zap/zapcore"
)

var cauthdslLogger = flogging.MustGetLogger("cauthdsl")

//重复数据消除在保留标识顺序的同时删除所有重复的标识
func deduplicate(sds []IdentityAndSignature) []IdentityAndSignature {
	ids := make(map[string]struct{})
	result := make([]IdentityAndSignature, 0, len(sds))
	for i, sd := range sds {
		identity, err := sd.Identity()
		if err != nil {
			cauthdslLogger.Errorf("Principal deserialization failure (%s) for identity %d", err, i)
			continue
		}
		key := identity.GetIdentifier().Mspid + identity.GetIdentifier().Id

		if _, ok := ids[key]; ok {
			cauthdslLogger.Warningf("De-duplicating identity [%s] at index %d in signature set", key, i)
		} else {
			result = append(result, sd)
			ids[key] = struct{}{}
		}
	}
	return result
}

//递归编译生成一个对应于指定策略的go-evaluable函数，请记住在
//将它们传递给此函数进行评估
func compile(policy *cb.SignaturePolicy, identities []*mb.MSPPrincipal, deserializer msp.IdentityDeserializer) (func([]IdentityAndSignature, []bool) bool, error) {
	if policy == nil {
		return nil, fmt.Errorf("Empty policy element")
	}

	switch t := policy.Type.(type) {
	case *cb.SignaturePolicy_NOutOf_:
		policies := make([]func([]IdentityAndSignature, []bool) bool, len(t.NOutOf.Rules))
		for i, policy := range t.NOutOf.Rules {
			compiledPolicy, err := compile(policy, identities, deserializer)
			if err != nil {
				return nil, err
			}
			policies[i] = compiledPolicy

		}
		return func(signedData []IdentityAndSignature, used []bool) bool {
			grepKey := time.Now().UnixNano()
			cauthdslLogger.Debugf("%p gate %d evaluation starts", signedData, grepKey)
			verified := int32(0)
			_used := make([]bool, len(used))
			for _, policy := range policies {
				copy(_used, used)
				if policy(signedData, _used) {
					verified++
					copy(used, _used)
				}
			}

			if verified >= t.NOutOf.N {
				cauthdslLogger.Debugf("%p gate %d evaluation succeeds", signedData, grepKey)
			} else {
				cauthdslLogger.Debugf("%p gate %d evaluation fails", signedData, grepKey)
			}

			return verified >= t.NOutOf.N
		}, nil
	case *cb.SignaturePolicy_SignedBy:
		if t.SignedBy < 0 || t.SignedBy >= int32(len(identities)) {
			return nil, fmt.Errorf("identity index out of range, requested %v, but identies length is %d", t.SignedBy, len(identities))
		}
		signedByID := identities[t.SignedBy]
		return func(signedData []IdentityAndSignature, used []bool) bool {
			cauthdslLogger.Debugf("%p signed by %d principal evaluation starts (used %v)", signedData, t.SignedBy, used)
			for i, sd := range signedData {
				if used[i] {
					cauthdslLogger.Debugf("%p skipping identity %d because it has already been used", signedData, i)
					continue
				}
				if cauthdslLogger.IsEnabledFor(zapcore.DebugLevel) {
//与大多数地方不同，这是一个巨大的print语句，值得在创建垃圾之前检查日志级别。
					cauthdslLogger.Debugf("%p processing identity %d with bytes of %x", signedData, i, sd.Identity)
				}
				identity, err := sd.Identity()
				if err != nil {
					cauthdslLogger.Errorf("Principal deserialization failure (%s) for identity %d", err, i)
					continue
				}
				err = identity.SatisfiesPrincipal(signedByID)
				if err != nil {
					cauthdslLogger.Debugf("%p identity %d does not satisfy principal: %s", signedData, i, err)
					continue
				}
				cauthdslLogger.Debugf("%p principal matched by identity %d", signedData, i)
				err = sd.Verify()
				if err != nil {
					cauthdslLogger.Debugf("%p signature for identity %d is invalid: %s", signedData, i, err)
					continue
				}
				cauthdslLogger.Debugf("%p principal evaluation succeeds for identity %d", signedData, i)
				used[i] = true
				return true
			}
			cauthdslLogger.Debugf("%p principal evaluation fails", signedData)
			return false
		}, nil
	default:
		return nil, fmt.Errorf("Unknown type: %T:%v", t, t)
	}
}
