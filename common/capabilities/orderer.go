
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
	ordererTypeName = "Orderer"

//orderv1_1是用于标准新的非向后兼容结构v1.1订购方功能的功能字符串。
	OrdererV1_1 = "V1_1"
)

//OrdererProvider为订购方级别配置提供功能信息。
type OrdererProvider struct {
	*registry
	v11BugFixes bool
}

//NewOrdererProvider创建一个Orderer功能提供程序。
func NewOrdererProvider(capabilities map[string]*cb.Capability) *OrdererProvider {
	cp := &OrdererProvider{}
	cp.registry = newRegistry(cp, capabilities)
	_, cp.v11BugFixes = capabilities[OrdererV1_1]
	return cp
}

//类型返回用于日志记录的描述性字符串。
func (cp *OrdererProvider) Type() string {
	return ordererTypeName
}

//如果此二进制文件支持此功能，则HasCapability返回true。
func (cp *OrdererProvider) HasCapability(capability string) bool {
	switch capability {
//在此处添加新功能名称
	case OrdererV1_1:
		return true
	default:
		return false
	}
}

//predictablechanneltemplate指定v1.0版中设置/channel的不良行为
//组的mod_策略为“”并从通道配置复制版本应该是固定的或不固定的。
func (cp *OrdererProvider) PredictableChannelTemplate() bool {
	return cp.v11BugFixes
}

//重新提交指定是否应通过重新提交来修复Tx的v1.0非确定性承诺
//重新验证的Tx。
func (cp *OrdererProvider) Resubmission() bool {
	return cp.v11BugFixes
}

//ExpirationCheck指定订购方是否检查标识过期检查
//验证消息时
func (cp *OrdererProvider) ExpirationCheck() bool {
	return cp.v11BugFixes
}
