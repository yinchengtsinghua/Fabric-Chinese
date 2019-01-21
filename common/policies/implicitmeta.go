
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
*/


package policies

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	"go.uber.org/zap/zapcore"
)

type implicitMetaPolicy struct {
	threshold   int
	subPolicies []Policy

//
	managers      map[string]*ManagerImpl
	subPolicyName string
}

//new policy基于策略字节创建新策略
func newImplicitMetaPolicy(data []byte, managers map[string]*ManagerImpl) (*implicitMetaPolicy, error) {
	definition := &cb.ImplicitMetaPolicy{}
	if err := proto.Unmarshal(data, definition); err != nil {
		return nil, fmt.Errorf("Error unmarshaling to ImplicitMetaPolicy: %s", err)
	}

	subPolicies := make([]Policy, len(managers))

	i := 0
	for _, manager := range managers {
		subPolicies[i], _ = manager.GetPolicy(definition.SubPolicy)
		i++
	}

	var threshold int

	switch definition.Rule {
	case cb.ImplicitMetaPolicy_ANY:
		threshold = 1
	case cb.ImplicitMetaPolicy_ALL:
		threshold = len(subPolicies)
	case cb.ImplicitMetaPolicy_MAJORITY:
		threshold = len(subPolicies)/2 + 1
	}

//
	if len(subPolicies) == 0 {
		threshold = 0
	}

	return &implicitMetaPolicy{
		subPolicies:   subPolicies,
		threshold:     threshold,
		managers:      managers,
		subPolicyName: definition.SubPolicy,
	}, nil
}

//Evaluate获取一组SignedData并评估该组签名是否满足策略
func (imp *implicitMetaPolicy) Evaluate(signatureSet []*cb.SignedData) error {
	logger.Debugf("This is an implicit meta policy, it will trigger other policy evaluations, whose failures may be benign")
	remaining := imp.threshold

	defer func() {
		if remaining != 0 {
//
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				var b bytes.Buffer
				b.WriteString(fmt.Sprintf("Evaluation Failed: Only %d policies were satisfied, but needed %d of [ ", imp.threshold-remaining, imp.threshold))
				for m := range imp.managers {
					b.WriteString(m)
					b.WriteString(".")
					b.WriteString(imp.subPolicyName)
					b.WriteString(" ")
				}
				b.WriteString("]")
				logger.Debugf(b.String())
			}
		}
	}()

	for _, policy := range imp.subPolicies {
		if policy.Evaluate(signatureSet) == nil {
			remaining--
			if remaining == 0 {
				return nil
			}
		}
	}
	if remaining == 0 {
		return nil
	}
	return fmt.Errorf("Failed to reach implicit threshold of %d sub-policies, required %d remaining", imp.threshold, remaining)
}
