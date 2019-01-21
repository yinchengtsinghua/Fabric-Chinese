
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp，SecureKey Technologies Inc.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package library

import (
	"github.com/hyperledger/fabric/core/handlers/auth"
	"github.com/hyperledger/fabric/core/handlers/auth/filter"
	"github.com/hyperledger/fabric/core/handlers/decoration"
	"github.com/hyperledger/fabric/core/handlers/decoration/decorator"
	"github.com/hyperledger/fabric/core/handlers/endorsement/api"
	"github.com/hyperledger/fabric/core/handlers/endorsement/builtin"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	. "github.com/hyperledger/fabric/core/handlers/validation/builtin"
)

//handlerLibrary用于断言
//如何创建各种处理程序
type HandlerLibrary struct {
}

//Debug将创建默认的Auth.Futter。
//这不做任何访问控制检查-简单地
//进一步转发请求。
//它需要通过调用init（）进行初始化
//并通过peer.背书服务器
func (r *HandlerLibrary) DefaultAuth() auth.Filter {
	return filter.NewFilter()
}

//ExpirationCheck是一个阻止请求的身份验证筛选器
//来自具有过期X509证书的标识
func (r *HandlerLibrary) ExpirationCheck() auth.Filter {
	return filter.NewExpirationCheckFilter()
}

//DefaultDecorator创建默认Decorator
//这与输入无关，只是
//将输入作为输出返回。
func (r *HandlerLibrary) DefaultDecorator() decoration.Decorator {
	return decorator.NewDecorator()
}

func (r *HandlerLibrary) DefaultEndorsement() endorsement.PluginFactory {
	return &builtin.DefaultEndorsementFactory{}
}

func (r *HandlerLibrary) DefaultValidation() validation.PluginFactory {
	return &DefaultValidationFactory{}
}
