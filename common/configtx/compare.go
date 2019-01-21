
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
	"bytes"

	cb "github.com/hyperledger/fabric/protos/common"
)

type comparable struct {
	*cb.ConfigGroup
	*cb.ConfigValue
	*cb.ConfigPolicy
	key  string
	path []string
}

func (cg comparable) equals(other comparable) bool {
	switch {
	case cg.ConfigGroup != nil:
		if other.ConfigGroup == nil {
			return false
		}
		return equalConfigGroup(cg.ConfigGroup, other.ConfigGroup)
	case cg.ConfigValue != nil:
		if other.ConfigValue == nil {
			return false
		}
		return equalConfigValues(cg.ConfigValue, other.ConfigValue)
	case cg.ConfigPolicy != nil:
		if other.ConfigPolicy == nil {
			return false
		}
		return equalConfigPolicies(cg.ConfigPolicy, other.ConfigPolicy)
	}

//不可达
	return false
}

func (cg comparable) version() uint64 {
	switch {
	case cg.ConfigGroup != nil:
		return cg.ConfigGroup.Version
	case cg.ConfigValue != nil:
		return cg.ConfigValue.Version
	case cg.ConfigPolicy != nil:
		return cg.ConfigPolicy.Version
	}

//不可达
	return 0
}

func (cg comparable) modPolicy() string {
	switch {
	case cg.ConfigGroup != nil:
		return cg.ConfigGroup.ModPolicy
	case cg.ConfigValue != nil:
		return cg.ConfigValue.ModPolicy
	case cg.ConfigPolicy != nil:
		return cg.ConfigPolicy.ModPolicy
	}

//不可达
	return ""
}

func equalConfigValues(lhs, rhs *cb.ConfigValue) bool {
	return lhs.Version == rhs.Version &&
		lhs.ModPolicy == rhs.ModPolicy &&
		bytes.Equal(lhs.Value, rhs.Value)
}

func equalConfigPolicies(lhs, rhs *cb.ConfigPolicy) bool {
	if lhs.Version != rhs.Version ||
		lhs.ModPolicy != rhs.ModPolicy {
		return false
	}

	if lhs.Policy == nil || rhs.Policy == nil {
		return lhs.Policy == rhs.Policy
	}

	return lhs.Policy.Type == rhs.Policy.Type &&
		bytes.Equal(lhs.Policy.Value, rhs.Policy.Value)
}

//子集函数检查内部是否是外部的子集
//Todo，尝试将这三个方法合并为一个，作为代码
//
func subsetOfGroups(inner, outer map[string]*cb.ConfigGroup) bool {
//空集是所有集的子集
	if len(inner) == 0 {
		return true
	}

//如果内部元素多于外部元素，则不能是子集
	if len(inner) > len(outer) {
		return false
	}

//如果内部的任何元素不在外部，则它不是子集
	for key := range inner {
		if _, ok := outer[key]; !ok {
			return false
		}
	}

	return true
}

func subsetOfPolicies(inner, outer map[string]*cb.ConfigPolicy) bool {
//空集是所有集的子集
	if len(inner) == 0 {
		return true
	}

//如果内部元素多于外部元素，则不能是子集
	if len(inner) > len(outer) {
		return false
	}

//如果内部的任何元素不在外部，则它不是子集
	for key := range inner {
		if _, ok := outer[key]; !ok {
			return false
		}
	}

	return true
}

func subsetOfValues(inner, outer map[string]*cb.ConfigValue) bool {
//空集是所有集的子集
	if len(inner) == 0 {
		return true
	}

//如果内部元素多于外部元素，则不能是子集
	if len(inner) > len(outer) {
		return false
	}

//如果内部的任何元素不在外部，则它不是子集
	for key := range inner {
		if _, ok := outer[key]; !ok {
			return false
		}
	}

	return true
}

func equalConfigGroup(lhs, rhs *cb.ConfigGroup) bool {
	if lhs.Version != rhs.Version ||
		lhs.ModPolicy != rhs.ModPolicy {
		return false
	}

	if !subsetOfGroups(lhs.Groups, rhs.Groups) ||
		!subsetOfGroups(rhs.Groups, lhs.Groups) ||
		!subsetOfPolicies(lhs.Policies, rhs.Policies) ||
		!subsetOfPolicies(rhs.Policies, lhs.Policies) ||
		!subsetOfValues(lhs.Values, rhs.Values) ||
		!subsetOfValues(rhs.Values, lhs.Values) {
		return false
	}

	return true
}
