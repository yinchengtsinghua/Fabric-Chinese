
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
*/


package config

import (
	"github.com/hyperledger/fabric/common/channelconfig"
)

type MockApplication struct {
	CapabilitiesRv channelconfig.ApplicationCapabilities
	Acls           map[string]string
}

func (m *MockApplication) Organizations() map[string]channelconfig.ApplicationOrg {
	return nil
}

func (m *MockApplication) Capabilities() channelconfig.ApplicationCapabilities {
	return m.CapabilitiesRv
}

func (m *MockApplication) PolicyRefForAPI(apiName string) string {
	if m.Acls == nil {
		return ""
	}
	return m.Acls[apiName]
}

//
func (m *MockApplication) APIPolicyMapper() channelconfig.PolicyMapper {
	return m
}

type MockApplicationCapabilities struct {
	SupportedRv                  error
	ForbidDuplicateTXIdInBlockRv bool
	ACLsRv                       bool
	PrivateChannelDataRv         bool
	CollectionUpgradeRv          bool
	V1_1ValidationRv             bool
	V1_2ValidationRv             bool
	MetadataLifecycleRv          bool
	KeyLevelEndorsementRv        bool
	V1_3ValidationRv             bool
	FabTokenRv                   bool
}

func (mac *MockApplicationCapabilities) Supported() error {
	return mac.SupportedRv
}

func (mac *MockApplicationCapabilities) ForbidDuplicateTXIdInBlock() bool {
	return mac.ForbidDuplicateTXIdInBlockRv
}

func (mac *MockApplicationCapabilities) ACLs() bool {
	return mac.ACLsRv
}

func (mac *MockApplicationCapabilities) PrivateChannelData() bool {
	return mac.PrivateChannelDataRv
}

func (mac *MockApplicationCapabilities) CollectionUpgrade() bool {
	return mac.CollectionUpgradeRv
}

func (mac *MockApplicationCapabilities) V1_1Validation() bool {
	return mac.V1_1ValidationRv
}

func (mac *MockApplicationCapabilities) V1_2Validation() bool {
	return mac.V1_2ValidationRv
}

func (mac *MockApplicationCapabilities) MetadataLifecycle() bool {
	return mac.MetadataLifecycleRv
}

func (mac *MockApplicationCapabilities) KeyLevelEndorsement() bool {
	return mac.KeyLevelEndorsementRv
}

func (mac *MockApplicationCapabilities) V1_3Validation() bool {
	return mac.V1_3ValidationRv
}

func (mac *MockApplicationCapabilities) FabToken() bool {
	return mac.FabTokenRv
}
