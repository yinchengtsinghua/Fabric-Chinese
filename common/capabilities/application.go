
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


package capabilities

import (
	cb "github.com/hyperledger/fabric/protos/common"
)

const (
	applicationTypeName = "Application"

//application v1_1是标准新的非向后兼容结构v1.1应用程序功能的功能字符串。
	ApplicationV1_1 = "V1_1"

//application v1_2是标准新的非向后兼容结构v1.2应用程序功能的功能字符串。
	ApplicationV1_2 = "V1_2"

//application v1_3是标准新的非向后兼容结构v1.3应用程序功能的功能字符串。
	ApplicationV1_3 = "V1_3"

//applicationpvdtataExperimental是使用collections/sidedb的实验功能的私有数据的功能字符串。
	ApplicationPvtDataExperimental = "V1_1_PVTDATA_EXPERIMENTAL"

//applicationResourceStreeExperimental是使用collections/sidedb的实验功能的私有数据的功能字符串。
	ApplicationResourcesTreeExperimental = "V1_1_RESOURCETREE_EXPERIMENTAL"

	ApplicationFabTokenExperimental = "V1_4_FABTOKEN_EXPERIMENTAL"
)

//ApplicationProvider为应用程序级配置提供功能信息。
type ApplicationProvider struct {
	*registry
	v11                     bool
	v12                     bool
	v13                     bool
	v11PvtDataExperimental  bool
	v14FabTokenExperimental bool
}

//NewApplicationProvider创建应用程序功能提供程序。
func NewApplicationProvider(capabilities map[string]*cb.Capability) *ApplicationProvider {
	ap := &ApplicationProvider{}
	ap.registry = newRegistry(ap, capabilities)
	_, ap.v11 = capabilities[ApplicationV1_1]
	_, ap.v12 = capabilities[ApplicationV1_2]
	_, ap.v13 = capabilities[ApplicationV1_3]
	_, ap.v11PvtDataExperimental = capabilities[ApplicationPvtDataExperimental]
	_, ap.v14FabTokenExperimental = capabilities[ApplicationFabTokenExperimental]
	return ap
}

//类型返回用于日志记录的描述性字符串。
func (ap *ApplicationProvider) Type() string {
	return applicationTypeName
}

//acls返回是否可以在通道应用程序配置中指定acls
func (ap *ApplicationProvider) ACLs() bool {
	return ap.v12 || ap.v13
}

//ForbidDuplicateXdinBlock指定是否允许两个具有相同TxID的事务
//在同一个块中，或者是否将第二个块标记为txvalidationcode_duplicate_txid
func (ap *ApplicationProvider) ForbidDuplicateTXIdInBlock() bool {
	return ap.v11 || ap.v12 || ap.v13
}

//如果启用了对专用通道数据（也称为集合）的支持，则private channel data返回true。
//在v1.1中，专用通道数据是实验性的，必须显式启用。
//在v1.2中，默认情况下启用专用通道数据。
func (ap *ApplicationProvider) PrivateChannelData() bool {
	return ap.v11PvtDataExperimental || ap.v12 || ap.v13
}

//如果将此通道配置为允许更新到
//现有集合或通过chaincode升级添加新集合（如v1.2所述）
func (ap ApplicationProvider) CollectionUpgrade() bool {
	return ap.v12 || ap.v13
}

//v11validation返回true是否将此通道配置为执行更严格的验证
//事务数（如v1.1中介绍的）。
func (ap *ApplicationProvider) V1_1Validation() bool {
	return ap.v11 || ap.v12 || ap.v13
}

//如果此通道配置为执行更严格的验证，则v12validation返回true
//事务数（如v1.2所述）。
func (ap *ApplicationProvider) V1_2Validation() bool {
	return ap.v12 || ap.v13
}

//如果此通道配置为执行更严格的验证，则v13validation返回true
//事务数（如v1.3所述）。
func (ap *ApplicationProvider) V1_3Validation() bool {
	return ap.v13
}

//MetadataLifecycle指示对等端是否应使用已弃用和有问题的
//v1.0/v1.1/v1.2生命周期，或者它是否应该使用更新的每通道对等本地链代码
//计划使用Fabric v1.3发布的元数据包方法
func (ap *ApplicationProvider) MetadataLifecycle() bool {
	return false
}

//如果此通道支持认可，则keyLevelndOrsement返回true
//如FAB-8812所述，以分类帐键粒度表示的策略
func (ap *ApplicationProvider) KeyLevelEndorsement() bool {
	return ap.v13
}

//如果启用了对结构令牌函数的支持，fabtoken将返回true。
func (ap *ApplicationProvider) FabToken() bool {
	return ap.v14FabTokenExperimental
}

//如果此二进制文件支持此功能，则HasCapability返回true。
func (ap *ApplicationProvider) HasCapability(capability string) bool {
	switch capability {
//在此处添加新功能名称
	case ApplicationV1_1:
		return true
	case ApplicationV1_2:
		return true
	case ApplicationV1_3:
		return true
	case ApplicationPvtDataExperimental:
		return true
	case ApplicationResourcesTreeExperimental:
		return true
	case ApplicationFabTokenExperimental:
		return true
	default:
		return false
	}
}
