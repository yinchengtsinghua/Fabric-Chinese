
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
	cb "github.com/hyperledger/fabric/protos/common"
)

//验证程序提供了一种机制来建议配置更新，请参见配置更新结果
//并验证配置更新的结果。
type Validator interface {
//验证应用configtx成为新配置的尝试
	Validate(configEnv *cb.ConfigEnvelope) error

//验证针对当前配置状态验证新configtx的尝试
	ProposeConfigUpdate(configtx *cb.Envelope) (*cb.ConfigEnvelope, error)

//chainID检索与此管理器关联的链ID
	ChainID() string

//config proto以proto的形式返回当前配置
	ConfigProto() *cb.Config

//Sequence返回配置的当前序列号
	Sequence() uint64
}
