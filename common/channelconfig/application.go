
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package channelconfig

import (
	"github.com/hyperledger/fabric/common/capabilities"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

const (
//applicationGroupKey是应用程序配置的组名
	ApplicationGroupKey = "Application"

//aclsky是acls配置的名称
	ACLsKey = "ACLs"
)

//applicationprotos用作applicationconfig的源
type ApplicationProtos struct {
	ACLs         *pb.ACLs
	Capabilities *cb.Capabilities
}

//applicationconfig实现应用程序接口
type ApplicationConfig struct {
	applicationOrgs map[string]ApplicationOrg
	protos          *ApplicationProtos
}

//
func NewApplicationConfig(appGroup *cb.ConfigGroup, mspConfig *MSPConfigHandler) (*ApplicationConfig, error) {
	ac := &ApplicationConfig{
		applicationOrgs: make(map[string]ApplicationOrg),
		protos:          &ApplicationProtos{},
	}

	if err := DeserializeProtoValuesFromGroup(appGroup, ac.protos); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize values")
	}

	if !ac.Capabilities().ACLs() {
		if _, ok := appGroup.Values[ACLsKey]; ok {
			return nil, errors.New("ACLs may not be specified without the required capability")
		}
	}

	var err error
	for orgName, orgGroup := range appGroup.Groups {
		ac.applicationOrgs[orgName], err = NewApplicationOrgConfig(orgName, orgGroup, mspConfig)
		if err != nil {
			return nil, err
		}
	}

	return ac, nil
}

//组织将组织ID的映射返回到ApplicationOrg
func (ac *ApplicationConfig) Organizations() map[string]ApplicationOrg {
	return ac.applicationOrgs
}

//能力返回能力名称到能力的映射
func (ac *ApplicationConfig) Capabilities() ApplicationCapabilities {
	return capabilities.NewApplicationProvider(ac.protos.Capabilities.Capabilities)
}

//api policymapper返回将api名称映射到策略的policymapper
func (ac *ApplicationConfig) APIPolicyMapper() PolicyMapper {
	pm := newAPIsProvider(ac.protos.ACLs.Acls)

	return pm
}
