
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


package token

//go：生成伪造者-o客户端/mock/identity.go-伪造名称标识。身份

//身份是指TX的创建者；
type Identity interface {
	Serialize() ([]byte, error)
}

//go：生成伪造者-o客户端/mock/signing_identity.go-伪造名称signingIdentity。签名身份

//SigningIdentity定义签名
//字节数组；需要对传输到的命令进行签名
//提供程序对等服务。
type SigningIdentity interface {
Identity //扩展标识

	Sign(msg []byte) ([]byte, error)

	GetPublicVersion() Identity
}
