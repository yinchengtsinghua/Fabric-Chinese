
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


package configtx

import (
	"testing"

	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestCompareConfigValue(t *testing.T) {
//正常等式
	assert.True(t, comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "foo",
			Value:     []byte("bar"),
		}}.equals(comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "foo",
			Value:     []byte("bar"),
		}}), "Should have found identical config values to be identical")

//不同的国防部政策
	assert.False(t, comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "foo",
			Value:     []byte("bar"),
		}}.equals(comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "bar",
			Value:     []byte("bar"),
		}}), "Should have detected different mod policy")

//不同值
	assert.False(t, comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "foo",
			Value:     []byte("bar"),
		}}.equals(comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "foo",
			Value:     []byte("foo"),
		}}), "Should have detected different value")

//不同版本
	assert.False(t, comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "foo",
			Value:     []byte("bar"),
		}}.equals(comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   1,
			ModPolicy: "foo",
			Value:     []byte("bar"),
		}}), "Should have detected different version")

//一个零值
	assert.False(t, comparable{
		ConfigValue: &cb.ConfigValue{
			Version:   0,
			ModPolicy: "foo",
			Value:     []byte("bar"),
		}}.equals(comparable{}), "Should have detected nil other value")

}

func TestCompareConfigPolicy(t *testing.T) {
//正常等式
	assert.True(t, comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}.equals(comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}), "Should have found identical config policies to be identical")

//不同的国防部政策
	assert.False(t, comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}.equals(comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "bar",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}), "Should have detected different mod policy")

//不同版本
	assert.False(t, comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}.equals(comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   1,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}), "Should have detected different version")

//不同的策略类型
	assert.False(t, comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}.equals(comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  2,
				Value: []byte("foo"),
			},
		}}), "Should have detected different policy type")

//不同的政策价值
	assert.False(t, comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}.equals(comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("bar"),
			},
		}}), "Should have detected different policy value")

//一个零值
	assert.False(t, comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}.equals(comparable{}), "Should have detected one nil value")

//一项零保单
	assert.False(t, comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type:  1,
				Value: []byte("foo"),
			},
		}}.equals(comparable{
		ConfigPolicy: &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: "foo",
			Policy: &cb.Policy{
				Type: 1,
			},
		}}), "Should have detected one nil policy")
}

func TestCompareConfigGroup(t *testing.T) {
//正常等式
	assert.True(t, comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil, "Bar1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil, "Bar2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar3": nil},
		}}.equals(comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil, "Bar1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil, "Bar2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar3": nil},
		}}), "Should have found identical config groups to be identical")

//不同的国防部政策
	assert.False(t, comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
		}}.equals(comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "bar",
		}}), "Should have detected different mod policy")

//不同版本
	assert.False(t, comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
		}}.equals(comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   1,
			ModPolicy: "foo",
		}}), "Should have detected different version")

//不同的组
	assert.False(t, comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil, "Bar2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar3": nil},
		}}.equals(comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil, "Bar1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil, "Bar2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar3": nil},
		}}), "Should have detected different groups entries")

//不同的值
	assert.False(t, comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil, "Bar1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil, "Bar2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar3": nil},
		}}.equals(comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil, "Bar1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar3": nil},
		}}), "Should have detected fifferent values entries")

//不同的政策
	assert.False(t, comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil, "Bar1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil, "Bar2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar3": nil},
		}}.equals(comparable{
		ConfigGroup: &cb.ConfigGroup{
			Version:   0,
			ModPolicy: "foo",
			Groups:    map[string]*cb.ConfigGroup{"Foo1": nil, "Bar1": nil},
			Values:    map[string]*cb.ConfigValue{"Foo2": nil, "Bar2": nil},
			Policies:  map[string]*cb.ConfigPolicy{"Foo3": nil, "Bar4": nil},
		}}), "Should have detected fifferent policies entries")
}
