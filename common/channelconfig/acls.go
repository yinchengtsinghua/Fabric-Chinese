
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有State Street Corp.2018保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package channelconfig

import (
	pb "github.com/hyperledger/fabric/protos/peer"
)

//aclsprovider提供资源到策略名称的映射
type aclsProvider struct {
	aclPolicyRefs map[string]string
}

func (ag *aclsProvider) PolicyRefForAPI(aclName string) string {
	return ag.aclPolicyRefs[aclName]
}

//如果需要，这会将策略转换为绝对路径
func newAPIsProvider(acls map[string]*pb.APIResource) *aclsProvider {
	aclPolicyRefs := make(map[string]string)

	for key, acl := range acls {
//如果策略是完全限定的，即到/频道/应用程序/读卡器，则不要使用它。
//否则，将其完全限定为引用/channel/application/policyname
		if '/' != acl.PolicyRef[0] {
			aclPolicyRefs[key] = "/" + ChannelGroupKey + "/" + ApplicationGroupKey + "/" + acl.PolicyRef
		} else {
			aclPolicyRefs[key] = acl.PolicyRef
		}
	}

	return &aclsProvider{
		aclPolicyRefs: aclPolicyRefs,
	}
}
