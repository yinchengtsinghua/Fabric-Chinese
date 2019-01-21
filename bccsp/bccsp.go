
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


package bccsp

import (
	"crypto"
	"hash"
)

//密钥表示加密密钥
type Key interface {

//字节将此键转换为其字节表示形式，
//如果允许此操作。
	Bytes() ([]byte, error)

//ski返回此密钥的主题密钥标识符。
	SKI() []byte

//如果此密钥是对称密钥，则对称返回true，
//假是这把钥匙是不对称的
	Symmetric() bool

//如果此密钥是私钥，则private返回true，
//否则为假。
	Private() bool

//public key返回非对称公钥/私钥对的相应公钥部分。
//此方法返回对称密钥方案中的错误。
	PublicKey() (Key, error)
}

//keygenopts包含使用CSP生成密钥的选项。
type KeyGenOpts interface {

//算法返回密钥生成算法标识符（要使用）。
	Algorithm() string

//如果要生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
	Ephemeral() bool
}

//keyDerivaOpts包含使用CSP进行密钥派生的选项。
type KeyDerivOpts interface {

//算法返回密钥派生算法标识符（要使用）。
	Algorithm() string

//如果派生的键必须是短暂的，则短暂返回true，
//否则为假。
	Ephemeral() bool
}

//keyimportopts包含用于导入具有CSP的密钥的原材料的选项。
type KeyImportOpts interface {

//算法返回密钥导入算法标识符（要使用）。
	Algorithm() string

//如果生成的密钥必须是短暂的，则短暂返回true，
//否则为假。
	Ephemeral() bool
}

//hashopts包含使用csp进行哈希的选项。
type HashOpts interface {

//Algorithm返回哈希算法标识符（要使用）。
	Algorithm() string
}

//SignerOpts包含使用CSP签名的选项。
type SignerOpts interface {
	crypto.SignerOpts
}

//encrypteropts包含使用CSP加密的选项。
type EncrypterOpts interface{}

//DecrypterOpts包含使用CSP进行解密的选项。
type DecrypterOpts interface{}

//BCCSP是提供
//密码标准和算法的实施。
type BCCSP interface {

//keygen使用opts生成密钥。
	KeyGen(opts KeyGenOpts) (k Key, err error)

//keyderive使用opts从k派生一个键。
//opts参数应该适合使用的原语。
	KeyDeriv(k Key, opts KeyDerivOpts) (dk Key, err error)

//keyimport使用opts从原始表示中导入密钥。
//opts参数应该适合使用的原语。
	KeyImport(raw interface{}, opts KeyImportOpts) (k Key, err error)

//GetKey返回此CSP关联的密钥
//主题键标识符ski。
	GetKey(ski []byte) (k Key, err error)

//哈希使用选项opts散列消息msg。
//如果opts为nil，将使用默认的哈希函数。
	Hash(msg []byte, opts HashOpts) (hash []byte, err error)

//gethash返回hash.hash的实例，使用选项opts。
//如果opts为nil，则返回默认的哈希函数。
	GetHash(opts HashOpts) (h hash.Hash, err error)

//用K键签署符号摘要。
//
//
//请注意，当需要较大消息的哈希签名时，
//调用者负责散列较大的消息并传递
//散列（作为摘要）。
	Sign(k Key, digest []byte, opts SignerOpts) (signature []byte, err error)

//验证根据密钥k和摘要验证签名
//opts参数应该适合所使用的算法。
	Verify(k Key, signature, digest []byte, opts SignerOpts) (valid bool, err error)

//加密使用密钥K加密明文。
//opts参数应该适合所使用的算法。
	Encrypt(k Key, plaintext []byte, opts EncrypterOpts) (ciphertext []byte, err error)

//解密使用密钥k解密密文。
//opts参数应该适合所使用的算法。
	Decrypt(k Key, ciphertext []byte, opts DecrypterOpts) (plaintext []byte, err error)
}
