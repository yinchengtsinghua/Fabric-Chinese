
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


package crypto

import (
	cb "github.com/hyperledger/fabric/protos/common"
)

//localsigner是一个临时存根接口，将由本地MSP实现。
type LocalSigner interface {
	SignatureHeaderMaker
	Signer
}

//签名者在邮件上签名
type Signer interface {
//在消息上签名并返回摘要上的签名，或者失败时出错
	Sign(message []byte) ([]byte, error)
}

//IdentitySerializer序列化标识
type IdentitySerializer interface {
//序列化将标识转换为字节
	Serialize() ([]byte, error)
}

//SignatureHeaderMaker创建新的SignatureHeader
type SignatureHeaderMaker interface {
//
	NewSignatureHeader() (*cb.SignatureHeader, error)
}

//SignatureHeaderCreator创建签名头
type SignatureHeaderCreator struct {
	SignerSupport
}

//SignerSupport实现对LocalSigner所需的支持
type SignerSupport interface {
	Signer
	IdentitySerializer
}

//NewSignatureHeaderCreator创建新的签名头
func NewSignatureHeaderCreator(ss SignerSupport) *SignatureHeaderCreator {
	return &SignatureHeaderCreator{ss}
}

//NewSignatureHeader创建具有正确签名标识和有效nonce的SignatureHeader
func (bs *SignatureHeaderCreator) NewSignatureHeader() (*cb.SignatureHeader, error) {
	creator, err := bs.Serialize()
	if err != nil {
		return nil, err
	}
	nonce, err := GetRandomNonce()
	if err != nil {
		return nil, err
	}

	return &cb.SignatureHeader{
		Creator: creator,
		Nonce:   nonce,
	}, nil
}
