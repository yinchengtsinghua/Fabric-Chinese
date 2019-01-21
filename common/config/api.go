
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

package config

import (
	cb "github.com/hyperledger/fabric/protos/common"
)

//config封装config（通道或资源）树
type Config interface {
//configproto返回当前配置
	ConfigProto() *cb.Config

//ProposeConfigUpdate尝试根据当前配置状态验证新的configtx
	ProposeConfigUpdate(configtx *cb.Envelope) (*cb.ConfigEnvelope, error)
}

//管理器提供对资源配置的访问
type Manager interface {
//getchannelconfig定义与通道配置相关的方法
	GetChannelConfig(channel string) Config
}
