
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


package update

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
)

func computePoliciesMapUpdate(original, updated map[string]*cb.ConfigPolicy) (readSet, writeSet, sameSet map[string]*cb.ConfigPolicy, updatedMembers bool) {
	readSet = make(map[string]*cb.ConfigPolicy)
	writeSet = make(map[string]*cb.ConfigPolicy)

//所有修改后的配置都进入读/写集，但是如果映射成员身份发生更改，我们将保留
//配置与添加到读/写集的配置相同
	sameSet = make(map[string]*cb.ConfigPolicy)

	for policyName, originalPolicy := range original {
		updatedPolicy, ok := updated[policyName]
		if !ok {
			updatedMembers = true
			continue
		}

		if originalPolicy.ModPolicy == updatedPolicy.ModPolicy && proto.Equal(originalPolicy.Policy, updatedPolicy.Policy) {
			sameSet[policyName] = &cb.ConfigPolicy{
				Version: originalPolicy.Version,
			}
			continue
		}

		writeSet[policyName] = &cb.ConfigPolicy{
			Version:   originalPolicy.Version + 1,
			ModPolicy: updatedPolicy.ModPolicy,
			Policy:    updatedPolicy.Policy,
		}
	}

	for policyName, updatedPolicy := range updated {
		if _, ok := original[policyName]; ok {
//如果updatedPolicy在原始策略集中，则它已被处理
			continue
		}
		updatedMembers = true
		writeSet[policyName] = &cb.ConfigPolicy{
			Version:   0,
			ModPolicy: updatedPolicy.ModPolicy,
			Policy:    updatedPolicy.Policy,
		}
	}

	return
}

func computeValuesMapUpdate(original, updated map[string]*cb.ConfigValue) (readSet, writeSet, sameSet map[string]*cb.ConfigValue, updatedMembers bool) {
	readSet = make(map[string]*cb.ConfigValue)
	writeSet = make(map[string]*cb.ConfigValue)

//所有修改后的配置都进入读/写集，但是如果映射成员身份发生更改，我们将保留
//配置与添加到读/写集的配置相同
	sameSet = make(map[string]*cb.ConfigValue)

	for valueName, originalValue := range original {
		updatedValue, ok := updated[valueName]
		if !ok {
			updatedMembers = true
			continue
		}

		if originalValue.ModPolicy == updatedValue.ModPolicy && bytes.Equal(originalValue.Value, updatedValue.Value) {
			sameSet[valueName] = &cb.ConfigValue{
				Version: originalValue.Version,
			}
			continue
		}

		writeSet[valueName] = &cb.ConfigValue{
			Version:   originalValue.Version + 1,
			ModPolicy: updatedValue.ModPolicy,
			Value:     updatedValue.Value,
		}
	}

	for valueName, updatedValue := range updated {
		if _, ok := original[valueName]; ok {
//如果updatedValue在原始值集中，则它已被处理
			continue
		}
		updatedMembers = true
		writeSet[valueName] = &cb.ConfigValue{
			Version:   0,
			ModPolicy: updatedValue.ModPolicy,
			Value:     updatedValue.Value,
		}
	}

	return
}

func computeGroupsMapUpdate(original, updated map[string]*cb.ConfigGroup) (readSet, writeSet, sameSet map[string]*cb.ConfigGroup, updatedMembers bool) {
	readSet = make(map[string]*cb.ConfigGroup)
	writeSet = make(map[string]*cb.ConfigGroup)

//所有修改后的配置都进入读/写集，但是如果映射成员身份发生更改，我们将保留
//配置与添加到读/写集的配置相同
	sameSet = make(map[string]*cb.ConfigGroup)

	for groupName, originalGroup := range original {
		updatedGroup, ok := updated[groupName]
		if !ok {
			updatedMembers = true
			continue
		}

		groupReadSet, groupWriteSet, groupUpdated := computeGroupUpdate(originalGroup, updatedGroup)
		if !groupUpdated {
			sameSet[groupName] = groupReadSet
			continue
		}

		readSet[groupName] = groupReadSet
		writeSet[groupName] = groupWriteSet

	}

	for groupName, updatedGroup := range updated {
		if _, ok := original[groupName]; ok {
//如果updatedGroup在原始组集中，则它已被处理
			continue
		}
		updatedMembers = true
		_, groupWriteSet, _ := computeGroupUpdate(cb.NewConfigGroup(), updatedGroup)
		writeSet[groupName] = &cb.ConfigGroup{
			Version:   0,
			ModPolicy: updatedGroup.ModPolicy,
			Policies:  groupWriteSet.Policies,
			Values:    groupWriteSet.Values,
			Groups:    groupWriteSet.Groups,
		}
	}

	return
}

func computeGroupUpdate(original, updated *cb.ConfigGroup) (readSet, writeSet *cb.ConfigGroup, updatedGroup bool) {
	readSetPolicies, writeSetPolicies, sameSetPolicies, policiesMembersUpdated := computePoliciesMapUpdate(original.Policies, updated.Policies)
	readSetValues, writeSetValues, sameSetValues, valuesMembersUpdated := computeValuesMapUpdate(original.Values, updated.Values)
	readSetGroups, writeSetGroups, sameSetGroups, groupsMembersUpdated := computeGroupsMapUpdate(original.Groups, updated.Groups)

//如果更新后的组与更新后的组“相等”（成员和mod策略均未更改）
	if !(policiesMembersUpdated || valuesMembersUpdated || groupsMembersUpdated || original.ModPolicy != updated.ModPolicy) {

//如果在任何策略/值/组映射中没有修改的条目
		if len(readSetPolicies) == 0 &&
			len(writeSetPolicies) == 0 &&
			len(readSetValues) == 0 &&
			len(writeSetValues) == 0 &&
			len(readSetGroups) == 0 &&
			len(writeSetGroups) == 0 {
			return &cb.ConfigGroup{
					Version: original.Version,
				}, &cb.ConfigGroup{
					Version: original.Version,
				}, false
		}

		return &cb.ConfigGroup{
				Version:  original.Version,
				Policies: readSetPolicies,
				Values:   readSetValues,
				Groups:   readSetGroups,
			}, &cb.ConfigGroup{
				Version:  original.Version,
				Policies: writeSetPolicies,
				Values:   writeSetValues,
				Groups:   writeSetGroups,
			}, true
	}

	for k, samePolicy := range sameSetPolicies {
		readSetPolicies[k] = samePolicy
		writeSetPolicies[k] = samePolicy
	}

	for k, sameValue := range sameSetValues {
		readSetValues[k] = sameValue
		writeSetValues[k] = sameValue
	}

	for k, sameGroup := range sameSetGroups {
		readSetGroups[k] = sameGroup
		writeSetGroups[k] = sameGroup
	}

	return &cb.ConfigGroup{
			Version:  original.Version,
			Policies: readSetPolicies,
			Values:   readSetValues,
			Groups:   readSetGroups,
		}, &cb.ConfigGroup{
			Version:   original.Version + 1,
			Policies:  writeSetPolicies,
			Values:    writeSetValues,
			Groups:    writeSetGroups,
			ModPolicy: updated.ModPolicy,
		}, true
}

func Compute(original, updated *cb.Config) (*cb.ConfigUpdate, error) {
	if original.ChannelGroup == nil {
		return nil, fmt.Errorf("no channel group included for original config")
	}

	if updated.ChannelGroup == nil {
		return nil, fmt.Errorf("no channel group included for updated config")
	}

	readSet, writeSet, groupUpdated := computeGroupUpdate(original.ChannelGroup, updated.ChannelGroup)
	if !groupUpdated {
		return nil, fmt.Errorf("no differences detected between original and updated config")
	}
	return &cb.ConfigUpdate{
		ReadSet:  readSet,
		WriteSet: writeSet,
	}, nil
}
