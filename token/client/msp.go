
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

package client

//go：生成伪造者-o mock/signer_identity.go-fake name signer identity。签名身份

type Signer interface {
//sign对给定的有效负载进行签名并返回签名
	Sign([]byte) ([]byte, error)
}

//SignerIdentity对消息进行签名并将其公共标识序列化为字节
type SignerIdentity interface {
	Signer

//serialize返回用于验证的此标识的字节表示形式
//此SignerIdentity签名的邮件
	Serialize() ([]byte, error)
}
