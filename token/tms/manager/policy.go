
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


package manager

import (
	"github.com/hyperledger/fabric/token/identity"
	"github.com/pkg/errors"
)

//allissuingvalidator允许通道的所有成员颁发新令牌。
type AllIssuingValidator struct {
	Deserializer identity.Deserializer
}

//如果传递的创建者可以颁发传递类型的令牌，则validate返回no error，否则返回错误。
func (p *AllIssuingValidator) Validate(creator identity.PublicInfo, tokenType string) error {
//反序列化标识
	identity, err := p.Deserializer.DeserializeIdentity(creator.Public())
	if err != nil {
		return errors.Wrapf(err, "identity [0x%x] cannot be deserialised", creator.Public())
	}

//检查身份有效性-在这个简单的策略中，所有有效的身份都是颁发者。
	if err := identity.Validate(); err != nil {
		return errors.Wrapf(err, "identity [0x%x] cannot be validated", creator.Public())
	}

	return nil
}
