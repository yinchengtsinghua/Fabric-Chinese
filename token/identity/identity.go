
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


package identity

import (
	"github.com/hyperledger/fabric/msp"
)

//IssuingValidator用于确定创建者是否可以颁发所传递类型的令牌。
type IssuingValidator interface {
//如果传递的创建者可以颁发传递类型的令牌，则validate返回no error，否则返回错误。
	Validate(creator PublicInfo, tokenType string) error
}

//publicInfo用于标识令牌所有者。
type PublicInfo interface {
	Public() []byte
}

//DeserializerManager返回反序列化程序的实例
type DeserializerManager interface {
//反序列化程序返回事务的实例。为传递的通道反序列化程序
//如果通道存在
	Deserializer(channel string) (Deserializer, error)
}

//解串器
type Deserializer interface {
//反序列化反序列化标识。
//如果标识与关联，则反序列化将失败
//与正在执行的MSP不同的MSP
//反序列化。
	DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error)
}
