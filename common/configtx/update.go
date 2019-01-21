
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
	"strings"

	"github.com/hyperledger/fabric/common/policies"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

func (vi *ValidatorImpl) verifyReadSet(readSet map[string]comparable) error {
	for key, value := range readSet {
		existing, ok := vi.configMap[key]
		if !ok {
			return errors.Errorf("existing config does not contain element for %s but was in the read set", key)
		}

		if existing.version() != value.version() {
			return errors.Errorf("readset expected key %s at version %d, but got version %d", key, value.version(), existing.version())
		}
	}
	return nil
}

func computeDeltaSet(readSet, writeSet map[string]comparable) map[string]comparable {
	result := make(map[string]comparable)
	for key, value := range writeSet {
		readVal, ok := readSet[key]

		if ok && readVal.version() == value.version() {
			continue
		}

//如果readset中的键是不同的版本，我们将其包括在内
//根据配置检查更新的健全性时出错。
		result[key] = value
	}
	return result
}

func validateModPolicy(modPolicy string) error {
	if modPolicy == "" {
		return errors.Errorf("mod_policy not set")
	}

	trimmed := modPolicy
	if modPolicy[0] == '/' {
		trimmed = modPolicy[1:]
	}

	for i, pathElement := range strings.Split(trimmed, pathSeparator) {
		err := validateConfigID(pathElement)
		if err != nil {
			return errors.Wrapf(err, "path element at %d is invalid", i)
		}
	}
	return nil

}

func (vi *ValidatorImpl) verifyDeltaSet(deltaSet map[string]comparable, signedData []*cb.SignedData) error {
	if len(deltaSet) == 0 {
		return errors.Errorf("delta set was empty -- update would have no effect")
	}

	for key, value := range deltaSet {
		logger.Debugf("Processing change to key: %s", key)
		if err := validateModPolicy(value.modPolicy()); err != nil {
			return errors.Wrapf(err, "invalid mod_policy for element %s", key)
		}

		existing, ok := vi.configMap[key]
		if !ok {
			if value.version() != 0 {
				return errors.Errorf("attempted to set key %s to version %d, but key does not exist", key, value.version())
			}

			continue
		}
		if value.version() != existing.version()+1 {
			return errors.Errorf("attempt to set key %s to version %d, but key is at version %d", key, value.version(), existing.version())
		}

		policy, ok := vi.policyForItem(existing)
		if !ok {
			return errors.Errorf("unexpected missing policy %s for item %s", existing.modPolicy(), key)
		}

//确保政策得到满足
		if err := policy.Evaluate(signedData); err != nil {
			return errors.Wrapf(err, "policy for %s not satisfied", key)
		}
	}
	return nil
}

func verifyFullProposedConfig(writeSet, fullProposedConfig map[string]comparable) error {
	for key := range writeSet {
		if _, ok := fullProposedConfig[key]; !ok {
			return errors.Errorf("writeset contained key %s which did not appear in proposed config", key)
		}
	}
	return nil
}

//authorizeupdate验证所有已修改的配置是否具有签名集所满足的相应修改策略。
//它返回已修改配置的映射
func (vi *ValidatorImpl) authorizeUpdate(configUpdateEnv *cb.ConfigUpdateEnvelope) (map[string]comparable, error) {
	if configUpdateEnv == nil {
		return nil, errors.Errorf("cannot process nil ConfigUpdateEnvelope")
	}

	configUpdate, err := UnmarshalConfigUpdate(configUpdateEnv.ConfigUpdate)
	if err != nil {
		return nil, err
	}

	if configUpdate.ChannelId != vi.channelID {
		return nil, errors.Errorf("ConfigUpdate for channel '%s' but envelope for channel '%s'", configUpdate.ChannelId, vi.channelID)
	}

	readSet, err := mapConfig(configUpdate.ReadSet, vi.namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "error mapping ReadSet")
	}
	err = vi.verifyReadSet(readSet)
	if err != nil {
		return nil, errors.Wrapf(err, "error validating ReadSet")
	}

	writeSet, err := mapConfig(configUpdate.WriteSet, vi.namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "error mapping WriteSet")
	}

	deltaSet := computeDeltaSet(readSet, writeSet)
	signedData, err := configUpdateEnv.AsSignedData()
	if err != nil {
		return nil, err
	}

	if err = vi.verifyDeltaSet(deltaSet, signedData); err != nil {
		return nil, errors.Wrapf(err, "error validating DeltaSet")
	}

	fullProposedConfig := vi.computeUpdateResult(deltaSet)
	if err := verifyFullProposedConfig(writeSet, fullProposedConfig); err != nil {
		return nil, errors.Wrapf(err, "full config did not verify")
	}

	return fullProposedConfig, nil
}

func (vi *ValidatorImpl) policyForItem(item comparable) (policies.Policy, bool) {
	manager := vi.pm

	modPolicy := item.modPolicy()
	logger.Debugf("Getting policy for item %s with mod_policy %s", item.key, modPolicy)

//如果modeu策略路径是相对路径，请获取上下文的正确管理器
//如果项的路径长度为零，则它是根组，请使用基本策略管理器
//如果modeu policy路径是绝对路径（以/开头），也可以使用基本策略管理器
	if len(modPolicy) > 0 && modPolicy[0] != policies.PathSeparator[0] && len(item.path) != 0 {
		var ok bool

		manager, ok = manager.Manager(item.path[1:])
		if !ok {
			logger.Debugf("Could not find manager at path: %v", item.path[1:])
			return nil, ok
		}

//对于组类型，其键是其路径的一部分，用于查找策略管理器
		if item.ConfigGroup != nil {
			manager, ok = manager.Manager([]string{item.key})
		}
		if !ok {
			logger.Debugf("Could not find group at subpath: %v", item.key)
			return nil, ok
		}
	}

	return manager.GetPolicy(item.modPolicy())
}

//ComputeUpdateResult获取由更新生成的configmap，并生成一个新的configmap，将其覆盖到旧配置上
func (vi *ValidatorImpl) computeUpdateResult(updatedConfig map[string]comparable) map[string]comparable {
	newConfigMap := make(map[string]comparable)
	for key, value := range vi.configMap {
		newConfigMap[key] = value
	}

	for key, value := range updatedConfig {
		newConfigMap[key] = value
	}
	return newConfigMap
}
