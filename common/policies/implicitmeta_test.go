
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


package policies

import (
	"fmt"
	"testing"

	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

const TestPolicyName = "TestPolicyName"

type acceptPolicy struct{}

func (rp acceptPolicy) Evaluate(signedData []*cb.SignedData) error {
	return nil
}

func TestImplicitMarshalError(t *testing.T) {
	_, err := newImplicitMetaPolicy([]byte("GARBAGE"), nil)
	assert.Error(t, err, "Should have errored unmarshaling garbage")
}

func makeManagers(count, passing int) map[string]*ManagerImpl {
	result := make(map[string]*ManagerImpl)
	remaining := passing
	for i := 0; i < count; i++ {
		policyMap := make(map[string]Policy)
		if remaining > 0 {
			policyMap[TestPolicyName] = acceptPolicy{}
		}
		remaining--

		result[fmt.Sprintf("%d", i)] = &ManagerImpl{
			policies: policyMap,
		}
	}
	return result
}

//makepolicytest使用一组
func runPolicyTest(rule cb.ImplicitMetaPolicy_Rule, managerCount int, passingCount int) error {
	imp, err := newImplicitMetaPolicy(utils.MarshalOrPanic(&cb.ImplicitMetaPolicy{
		Rule:      rule,
		SubPolicy: TestPolicyName,
	}), makeManagers(managerCount, passingCount))
	if err != nil {
		panic(err)
	}

	return imp.Evaluate(nil)
}

func TestImplicitMetaAny(t *testing.T) {
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_ANY, 1, 1))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_ANY, 10, 1))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_ANY, 10, 8))
	assert.Error(t, runPolicyTest(cb.ImplicitMetaPolicy_ANY, 10, 0))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_ANY, 0, 0))
}

func TestImplicitMetaAll(t *testing.T) {
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_ALL, 1, 1))
	assert.Error(t, runPolicyTest(cb.ImplicitMetaPolicy_ALL, 10, 1))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_ALL, 10, 10))
	assert.Error(t, runPolicyTest(cb.ImplicitMetaPolicy_ALL, 10, 0))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_ALL, 0, 0))
}

func TestImplicitMetaMajority(t *testing.T) {
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_MAJORITY, 1, 1))
	assert.Error(t, runPolicyTest(cb.ImplicitMetaPolicy_MAJORITY, 10, 5))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_MAJORITY, 10, 6))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_MAJORITY, 3, 2))
	assert.Error(t, runPolicyTest(cb.ImplicitMetaPolicy_MAJORITY, 10, 0))
	assert.NoError(t, runPolicyTest(cb.ImplicitMetaPolicy_MAJORITY, 0, 0))
}
