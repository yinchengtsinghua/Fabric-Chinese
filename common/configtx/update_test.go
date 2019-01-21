
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


package configtx

import (
	"fmt"
	"testing"

	mockpolicies "github.com/hyperledger/fabric/common/mocks/policies"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestReadSetNotPresent(t *testing.T) {
	vi := &ValidatorImpl{
		configMap: make(map[string]comparable),
	}

	vi.configMap["1"] = comparable{}
	vi.configMap["2"] = comparable{}

	readSet := make(map[string]comparable)
	readSet["1"] = comparable{}
	readSet["3"] = comparable{}

	assert.Error(t, vi.verifyReadSet(readSet), "ReadSet contained '3', not in config")
}

func TestReadSetBackVersioned(t *testing.T) {
	vi := &ValidatorImpl{
		configMap: make(map[string]comparable),
	}

	vi.configMap["1"] = comparable{ConfigValue: &cb.ConfigValue{Version: 1}}
	vi.configMap["2"] = comparable{}

	readSet := make(map[string]comparable)
	readSet["1"] = comparable{}

	assert.Error(t, vi.verifyReadSet(readSet), "ReadSet contained '1', at old version")
}

func TestComputeDeltaSet(t *testing.T) {
	readSet := make(map[string]comparable)
	readSet["1"] = comparable{}
	readSet["2"] = comparable{}

	writeSet := make(map[string]comparable)
	writeSet["1"] = comparable{}
	writeSet["2"] = comparable{ConfigValue: &cb.ConfigValue{Version: 1}}
	writeSet["3"] = comparable{}

	result := computeDeltaSet(readSet, writeSet)
	assert.Len(t, result, 2, "Should have two values in the delta set")
	assert.NotNil(t, result["2"], "Element had version increased")
	assert.NotNil(t, result["3"], "Element was new")
}

func TestVerifyDeltaSet(t *testing.T) {
	vi := &ValidatorImpl{
		pm: &mockpolicies.Manager{
			Policy: &mockpolicies.Policy{},
		},
		configMap: make(map[string]comparable),
	}

	vi.configMap["foo"] = comparable{path: []string{"foo"}}

	t.Run("Green path", func(t *testing.T) {
		deltaSet := make(map[string]comparable)

		deltaSet["foo"] = comparable{ConfigValue: &cb.ConfigValue{Version: 1, ModPolicy: "foo"}}

		assert.NoError(t, vi.verifyDeltaSet(deltaSet, nil), "Good update")
	})

	t.Run("Bad mod policy", func(t *testing.T) {
		deltaSet := make(map[string]comparable)

		deltaSet["foo"] = comparable{ConfigValue: &cb.ConfigValue{Version: 1}}

		assert.Regexp(t, "invalid mod_policy for element", vi.verifyDeltaSet(deltaSet, nil))
	})

	t.Run("Big Skip", func(t *testing.T) {
		deltaSet := make(map[string]comparable)

		deltaSet["foo"] = comparable{ConfigValue: &cb.ConfigValue{Version: 2, ModPolicy: "foo"}}

		assert.Error(t, vi.verifyDeltaSet(deltaSet, nil), "Version skip from 0 to 2")
	})

	t.Run("New item high version", func(t *testing.T) {
		deltaSet := make(map[string]comparable)

		deltaSet["bar"] = comparable{ConfigValue: &cb.ConfigValue{Version: 1, ModPolicy: "foo"}}

		assert.Error(t, vi.verifyDeltaSet(deltaSet, nil), "New key not at version 0")
	})

	t.Run("Policy evalaution to false", func(t *testing.T) {
		deltaSet := make(map[string]comparable)

		deltaSet["foo"] = comparable{ConfigValue: &cb.ConfigValue{Version: 1, ModPolicy: "foo"}}
		vi.pm.(*mockpolicies.Manager).Policy = &mockpolicies.Policy{Err: fmt.Errorf("Err")}

		assert.Error(t, vi.verifyDeltaSet(deltaSet, nil), "Policy evaluation should have failed")
	})

	t.Run("Empty delta set", func(t *testing.T) {
		err := (&ValidatorImpl{}).verifyDeltaSet(map[string]comparable{}, nil)
		assert.Error(t, err, "Empty delta set should be rejected")
		assert.Contains(t, err.Error(), "delta set was empty -- update would have no effect")
	})
}

func TestPolicyForItem(t *testing.T) {
//策略被设置为不同的错误值，以便在相等的断言中区分它们
	rootPolicy := &mockpolicies.Policy{Err: fmt.Errorf("rootPolicy")}
	fooPolicy := &mockpolicies.Policy{Err: fmt.Errorf("fooPolicy")}

	vi := &ValidatorImpl{
		pm: &mockpolicies.Manager{
			Policy: rootPolicy,
			SubManagersMap: map[string]*mockpolicies.Manager{
				"foo": {
					Policy: fooPolicy,
				},
			},
		},
	}

	t.Run("Root manager", func(t *testing.T) {
		policy, ok := vi.policyForItem(comparable{
			path: []string{"root"},
			ConfigValue: &cb.ConfigValue{
				ModPolicy: "rootPolicy",
			},
		})
		assert.True(t, ok)
		assert.Equal(t, policy, rootPolicy, "Should have found relative policy off the root manager")
	})

	t.Run("Nonexistent manager", func(t *testing.T) {
		_, ok := vi.policyForItem(comparable{
			path: []string{"root", "wrong"},
			ConfigValue: &cb.ConfigValue{
				ModPolicy: "rootPolicy",
			},
		})
		assert.False(t, ok, "Should not have found rootPolicy off a nonexistent manager")
	})

	t.Run("Foo manager", func(t *testing.T) {
		policy, ok := vi.policyForItem(comparable{
			path: []string{"root", "foo"},
			ConfigValue: &cb.ConfigValue{
				ModPolicy: "foo",
			},
		})
		assert.True(t, ok)
		assert.Equal(t, policy, fooPolicy, "Should have found relative foo policy off the foo manager")
	})

	t.Run("Foo group", func(t *testing.T) {
		policy, ok := vi.policyForItem(comparable{
			key:  "foo",
			path: []string{"root"},
			ConfigGroup: &cb.ConfigGroup{
				ModPolicy: "foo",
			},
		})
		assert.True(t, ok)
		assert.Equal(t, policy, fooPolicy, "Should have found relative foo policy for foo group")
	})

	t.Run("Root group manager", func(t *testing.T) {
		policy, ok := vi.policyForItem(comparable{
			path: []string{},
			key:  "root",
			ConfigGroup: &cb.ConfigGroup{
				ModPolicy: "rootPolicy",
			},
		})
		assert.True(t, ok)
		assert.Equal(t, policy, rootPolicy, "Should have found relative policy off the root manager")
	})
}

func TestValidateModPolicy(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		assert.Nil(t, validateModPolicy("/foo/bar"))
	})
	t.Run("Empty", func(t *testing.T) {
		assert.Regexp(t, "mod_policy not set", validateModPolicy(""))
	})
	t.Run("InvalidFirstChar", func(t *testing.T) {
		assert.Regexp(t, "path element at 0 is invalid", validateModPolicy("^foo"))
	})
	t.Run("InvalidRootPath", func(t *testing.T) {
		assert.Regexp(t, "path element at 0 is invalid", validateModPolicy("/"))
	})
	t.Run("InvalidSubPath", func(t *testing.T) {
assert.Regexp(t, "path element at 1 is invalid", validateModPolicy("foo//“酒吧”
	})
}
