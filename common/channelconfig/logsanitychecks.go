
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


package channelconfig

import (
	"github.com/hyperledger/fabric/common/policies"
)

func LogSanityChecks(res Resources) {
	pm := res.PolicyManager()
	for _, policyName := range []string{policies.ChannelReaders, policies.ChannelWriters} {
		_, ok := pm.GetPolicy(policyName)
		if !ok {
			logger.Warningf("Current configuration has no policy '%s', this will likely cause problems in production systems", policyName)
		} else {
			logger.Debugf("As expected, current configuration has policy '%s'", policyName)
		}
	}
	if _, ok := pm.Manager([]string{policies.ApplicationPrefix}); ok {
//
		for _, policyName := range []string{
			policies.ChannelApplicationReaders,
			policies.ChannelApplicationWriters,
			policies.ChannelApplicationAdmins} {
			_, ok := pm.GetPolicy(policyName)
			if !ok {
				logger.Warningf("Current configuration has no policy '%s', this will likely cause problems in production systems", policyName)
			} else {
				logger.Debugf("As expected, current configuration has policy '%s'", policyName)
			}
		}
	}
	if _, ok := pm.Manager([]string{policies.OrdererPrefix}); ok {
		for _, policyName := range []string{policies.BlockValidation} {
			_, ok := pm.GetPolicy(policyName)
			if !ok {
				logger.Warningf("Current configuration has no policy '%s', this will likely cause problems in production systems", policyName)
			} else {
				logger.Debugf("As expected, current configuration has policy '%s'", policyName)
			}
		}
	}
}
