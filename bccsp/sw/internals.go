
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


package sw

import (
	"hash"

	"github.com/hyperledger/fabric/bccsp"
)

//keyGenerator是一个类似bccsp的接口，提供密钥生成算法
type KeyGenerator interface {

//keygen使用opts生成密钥。
	KeyGen(opts bccsp.KeyGenOpts) (k bccsp.Key, err error)
}

//keyDeriver是一个类似bccsp的接口，提供密钥派生算法
type KeyDeriver interface {

//keyderive使用opts从k派生一个键。
//opts参数应该适合使用的原语。
	KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (dk bccsp.Key, err error)
}

//keyImporter是一个类似bccsp的接口，提供密钥导入算法
type KeyImporter interface {

//keyimport使用opts从原始表示中导入密钥。
//opts参数应该适合使用的原语。
	KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error)
}

//Encryptor是一个类似bccsp的接口，提供加密算法
type Encryptor interface {

//加密使用密钥K加密明文。
//opts参数应该适合所使用的算法。
	Encrypt(k bccsp.Key, plaintext []byte, opts bccsp.EncrypterOpts) (ciphertext []byte, err error)
}

//解密器是一个类似bccsp的接口，提供解密算法
type Decryptor interface {

//解密使用密钥k解密密文。
//opts参数应该适合所使用的算法。
	Decrypt(k bccsp.Key, ciphertext []byte, opts bccsp.DecrypterOpts) (plaintext []byte, err error)
}

//签名者是一个类似bccsp的接口，提供签名算法
type Signer interface {

//
//opts参数应该适合所使用的算法。
//
//请注意，当需要较大消息的哈希签名时，
//调用者负责散列较大的消息并传递
//散列（作为摘要）。
	Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error)
}

//Verifier是一个类似bccsp的接口，提供验证算法
type Verifier interface {

//验证根据密钥k和摘要验证签名
//opts参数应该适合所使用的算法。
	Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error)
}

//hasher是一个类似bccsp的接口，提供哈希算法
type Hasher interface {

//哈希使用选项opts散列消息msg。
//如果opts为nil，将使用默认的哈希函数。
	Hash(msg []byte, opts bccsp.HashOpts) (hash []byte, err error)

//gethash返回hash.hash的实例，使用选项opts。
//如果opts为nil，则返回默认的哈希函数。
	GetHash(opts bccsp.HashOpts) (h hash.Hash, err error)
}
