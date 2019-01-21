
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
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
)

const (
	groupPrefix  = "[Group]  "
	valuePrefix  = "[Value]  "
	policyPrefix = "[Policy] "

	pathSeparator = "/"

//hacky fix常量，用于recurseconfigmap
	hackyFixOrdererCapabilities = "[Value]  /Channel/Orderer/Capabilities"
	hackyFixNewModPolicy        = "Admins"
)

//mapconfig将在此文件外部调用
//它接受一个configgroup并生成一个到可比较项的fqpath映射（或对无效键出错）
func mapConfig(channelGroup *cb.ConfigGroup, rootGroupKey string) (map[string]comparable, error) {
	result := make(map[string]comparable)
	if channelGroup != nil {
		err := recurseConfig(result, []string{rootGroupKey}, channelGroup)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

//addtomap仅由mapconfig内部使用
func addToMap(cg comparable, result map[string]comparable) error {
	var fqPath string

	switch {
	case cg.ConfigGroup != nil:
		fqPath = groupPrefix
	case cg.ConfigValue != nil:
		fqPath = valuePrefix
	case cg.ConfigPolicy != nil:
		fqPath = policyPrefix
	}

	if err := validateConfigID(cg.key); err != nil {
		return fmt.Errorf("Illegal characters in key: %s", fqPath)
	}

	if len(cg.path) == 0 {
		fqPath += pathSeparator + cg.key
	} else {
		fqPath += pathSeparator + strings.Join(cg.path, pathSeparator) + pathSeparator + cg.key
	}

	logger.Debugf("Adding to config map: %s", fqPath)

	result[fqPath] = cg

	return nil
}

//recurseconfig仅由mapconfig内部使用
func recurseConfig(result map[string]comparable, path []string, group *cb.ConfigGroup) error {
	if err := addToMap(comparable{key: path[len(path)-1], path: path[:len(path)-1], ConfigGroup: group}, result); err != nil {
		return err
	}

	for key, group := range group.Groups {
		nextPath := make([]string, len(path)+1)
		copy(nextPath, path)
		nextPath[len(nextPath)-1] = key
		if err := recurseConfig(result, nextPath, group); err != nil {
			return err
		}
	}

	for key, value := range group.Values {
		if err := addToMap(comparable{key: key, path: path, ConfigValue: value}, result); err != nil {
			return err
		}
	}

	for key, policy := range group.Policies {
		if err := addToMap(comparable{key: key, path: path, ConfigPolicy: policy}, result); err != nil {
			return err
		}
	}

	return nil
}

//configmaptoconfig用于从此文件外部调用
//它接受configmap并将其转换回*cb.configgroup结构
func configMapToConfig(configMap map[string]comparable, rootGroupKey string) (*cb.ConfigGroup, error) {
	rootPath := pathSeparator + rootGroupKey
	return recurseConfigMap(rootPath, configMap)
}

//RecurseConfigMap仅由ConfigMapConfig内部使用
//注意，此函数不再改变configmap中的cb.config*项
func recurseConfigMap(path string, configMap map[string]comparable) (*cb.ConfigGroup, error) {
	groupPath := groupPrefix + path
	group, ok := configMap[groupPath]
	if !ok {
		return nil, fmt.Errorf("Missing group at path: %s", groupPath)
	}

	if group.ConfigGroup == nil {
		return nil, fmt.Errorf("ConfigGroup not found at group path: %s", groupPath)
	}

	newConfigGroup := cb.NewConfigGroup()
	proto.Merge(newConfigGroup, group.ConfigGroup)

	for key := range group.Groups {
		updatedGroup, err := recurseConfigMap(path+pathSeparator+key, configMap)
		if err != nil {
			return nil, err
		}
		newConfigGroup.Groups[key] = updatedGroup
	}

	for key := range group.Values {
		valuePath := valuePrefix + path + pathSeparator + key
		value, ok := configMap[valuePath]
		if !ok {
			return nil, fmt.Errorf("Missing value at path: %s", valuePath)
		}
		if value.ConfigValue == nil {
			return nil, fmt.Errorf("ConfigValue not found at value path: %s", valuePath)
		}
		newConfigGroup.Values[key] = proto.Clone(value.ConfigValue).(*cb.ConfigValue)
	}

	for key := range group.Policies {
		policyPath := policyPrefix + path + pathSeparator + key
		policy, ok := configMap[policyPath]
		if !ok {
			return nil, fmt.Errorf("Missing policy at path: %s", policyPath)
		}
		if policy.ConfigPolicy == nil {
			return nil, fmt.Errorf("ConfigPolicy not found at policy path: %s", policyPath)
		}
		newConfigGroup.Policies[key] = proto.Clone(policy.ConfigPolicy).(*cb.ConfigPolicy)
		logger.Debugf("Setting policy for key %s to %+v", key, group.Policies[key])
	}

//这是一个非常黑客的修复，以方便升级已构建的通道
//使用v1.0版本中的通道生成，其中包含bug fab-5309和fab-6080。
//总之，这些通道是用一个bug构建的，在某些情况下，这个bug使mod_策略不设置。
//如果未设置mod-policy，则无法修改元素，当前代码不允许
//取消设置mod_策略值。这个黑客用空的mod_策略值“修复”现有的配置。
//如果功能框架处于打开状态，它会将任何未设置的mod_策略设置为“admins”。
//此代码需要一直存在，直到从代码库中否决v1.0链的验证。
	if _, ok := configMap[hackyFixOrdererCapabilities]; ok {
//hacky fix常量，用于recurseconfigmap
		if newConfigGroup.ModPolicy == "" {
			logger.Debugf("Performing upgrade of group %s empty mod_policy", groupPath)
			newConfigGroup.ModPolicy = hackyFixNewModPolicy
		}

		for key, value := range newConfigGroup.Values {
			if value.ModPolicy == "" {
				logger.Debugf("Performing upgrade of value %s empty mod_policy", valuePrefix+path+pathSeparator+key)
				value.ModPolicy = hackyFixNewModPolicy
			}
		}

		for key, policy := range newConfigGroup.Policies {
			if policy.ModPolicy == "" {
				logger.Debugf("Performing upgrade of policy %s empty mod_policy", policyPrefix+path+pathSeparator+key)

				policy.ModPolicy = hackyFixNewModPolicy
			}
		}
	}

	return newConfigGroup, nil
}
