
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/

package signer

import (
	"crypto"
	"io"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/pkg/errors"
)

//bccsp crypto signer是基于bccsp的crypto.signer实现
type bccspCryptoSigner struct {
	csp bccsp.BCCSP
	key bccsp.Key
	pk  interface{}
}

//new返回基于bccsp的新crypto.signer
//对于给定的BCCSP实例和密钥。
func New(csp bccsp.BCCSP, key bccsp.Key) (crypto.Signer, error) {
//验证参数
	if csp == nil {
		return nil, errors.New("bccsp instance must be different from nil.")
	}
	if key == nil {
		return nil, errors.New("key must be different from nil.")
	}
	if key.Symmetric() {
		return nil, errors.New("key must be asymmetric.")
	}

//将bccsp公钥马歇尔为crypto.public key
	pub, err := key.PublicKey()
	if err != nil {
		return nil, errors.Wrap(err, "failed getting public key")
	}

	raw, err := pub.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling public key")
	}

	pk, err := utils.DERToPublicKey(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling der to public key")
	}

	return &bccspCryptoSigner{csp, key, pk}, nil
}

//public返回与opaque对应的公钥，
//私钥。
func (s *bccspCryptoSigner) Public() crypto.PublicKey {
	return s.pk
}

//使用私钥进行符号摘要，可能使用来自的熵
//兰德对于RSA密钥，生成的签名应该是
//
//密钥，它应该是一个序列化的ASN.1签名结构。
//
//哈希实现了signeropts接口，在大多数情况下，可以
//只需传入用作opt的哈希函数。标志也可以尝试
//为了获得算法，将opts类型断言为其他类型
//特定值。有关详细信息，请参阅每个包中的文档。
//
//请注意，当需要较大消息的哈希签名时，
//调用者负责散列较大的消息并传递
//要签名的哈希（作为摘要）和哈希函数（作为opts）。
func (s *bccspCryptoSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.csp.Sign(s.key, digest, opts)
}
