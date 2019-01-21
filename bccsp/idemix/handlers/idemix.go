
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

package handlers

import (
	"crypto/ecdsa"

	"github.com/hyperledger/fabric/bccsp"
)

//IssuerPublickey是颁发者公钥
type IssuerPublicKey interface {

//bytes返回此键的字节表示形式
	Bytes() ([]byte, error)

//哈希返回此键的哈希表示形式。
//输出应该是抗碰撞的
	Hash() []byte
}

//IssuerPublickey是颁发者密钥
type IssuerSecretKey interface {

//bytes返回此键的字节表示形式
	Bytes() ([]byte, error)

//public返回对应的公钥
	Public() IssuerPublicKey
}

//Issuer是一个本地接口，用于与IDemix实现分离。
type Issuer interface {
//new key生成新的IDemix颁发者密钥w.r.t传递的属性名。
	NewKey(AttributeNames []string) (IssuerSecretKey, error)

//NewPublicKeyFromBytes将传递的字节转换为颁发者公钥
//它确保这样获得的公钥具有传递的属性（如果指定的话）
	NewPublicKeyFromBytes(raw []byte, attributes []string) (IssuerPublicKey, error)
}

//大表示一个大整数
type Big interface {
//bytes返回此键的字节表示形式
	Bytes() ([]byte, error)
}

//ECP表示一个椭圆曲线点
type Ecp interface {
//bytes返回此键的字节表示形式
	Bytes() ([]byte, error)
}

//用户是从IDemix实现中分离出来的本地接口
type User interface {
//new key生成新的用户密钥
	NewKey() (Big, error)

//newkeyfrombytes将传递的字节转换为用户密钥
	NewKeyFromBytes(raw []byte) (Big, error)

//makenym创建一个新的不可链接的笔名
	MakeNym(sk Big, key IssuerPublicKey) (Ecp, Big, error)

//NewPublicNymFromBytes将传递的字节转换为公共名称
	NewPublicNymFromBytes(raw []byte) (Ecp, error)
}

//CredRequest是一个本地接口，用于与IDemix实现分离。
//发出凭证请求。
type CredRequest interface {
//签名创建一个新的凭证请求，这是交互式凭证颁发协议的第一条消息
//（从用户到颁发者）
	Sign(sk Big, ipk IssuerPublicKey, nonce []byte) ([]byte, error)

//验证验证验证凭据请求
	Verify(credRequest []byte, ipk IssuerPublicKey, nonce []byte) error
}

//CredRequest是一个本地接口，用于与IDemix实现分离。
//证书的颁发。
type Credential interface {

//签名颁发新的凭据，这是交互式颁发协议的最后一步
//在此步骤中，所有属性值都由颁发者添加，然后与承诺一起签名
//来自凭据请求的用户密钥
	Sign(key IssuerSecretKey, credentialRequest []byte, attributes []bccsp.IdemixAttribute) ([]byte, error)

//通过验证签名以加密方式验证凭据
//属性值和用户密钥
	Verify(sk Big, ipk IssuerPublicKey, credential []byte, attributes []bccsp.IdemixAttribute) error
}

//吊销是一个本地接口，用于与IDemix实现分离
//与撤销相关的操作
type Revocation interface {

//newkey生成用于吊销的长期签名密钥
	NewKey() (*ecdsa.PrivateKey, error)

//sign创建特定时间段（epoch）的凭证吊销信息。
//用户可以使用CRI来证明他们没有被撤销。
//注意，当不使用撤销（即alg=alg_no_撤销）时，不使用输入的未撤销数据，
//由此产生的CRI可以被任何签名者使用。
	Sign(key *ecdsa.PrivateKey, unrevokedHandles [][]byte, epoch int, alg bccsp.RevocationAlgorithm) ([]byte, error)

//验证验证特定时期的吊销pk是否有效，
//通过检查它是否使用长期吊销密钥签名。
//注意，即使我们不使用撤销（即alg=alg_no_撤销），我们也需要
//验证签名以确保颁发者确实签署了没有吊销的签名
//在这个时代使用。
	Verify(pk *ecdsa.PublicKey, cri []byte, epoch int, alg bccsp.RevocationAlgorithm) error
}

//SignatureScheme是一个本地接口，用于与IDemix实现分离。
//标志相关操作
type SignatureScheme interface {
//签名创建新的IDemix签名（schnorr类型签名）。
//属性切片控制公开哪些属性：
//如果attributes[i].type==bccsp.idemixHiddenAttribute，则attribute i保持隐藏状态，否则将被公开。
//我们要求撤销句柄保持未公开（即，属性[rhindex]==bccsp.idemixhiddenattribute）。
//参数应理解为：
//cred：IDemix凭证的序列化版本；
//sk：用户密钥；
//（nym，rnym）：nym密钥对；
//ipk：发行人公钥；
//属性：如上所述；
//msg：要签名的消息；
//Rhindex：与属性相关的撤销句柄索引；
//
//是在引用中创建的）。
	Sign(cred []byte, sk Big, Nym Ecp, RNym Big, ipk IssuerPublicKey, attributes []bccsp.IdemixAttribute,
		msg []byte, rhIndex int, cri []byte) ([]byte, error)

//verify验证IDemix签名。
//属性切片控制它希望公开的属性
//如果属性[i].type==bccsp.idemixHiddenAttribute，则属性i保持隐藏状态，否则
//属性[i].值应包含所公开的属性值。
//换句话说，此函数将检查如果属性i被公开，则第i个属性等于属性[i].value。
//参数应理解为：
//ipk：发行人公钥；
//签名：签名验证；
//消息：消息已签名；
//属性：如上所述；
//Rhindex：与属性相关的撤销句柄索引；
//吊销公钥：吊销公钥；
//时代：撤销时代。
	Verify(ipk IssuerPublicKey, signature, msg []byte, attributes []bccsp.IdemixAttribute, rhIndex int, revocationPublicKey *ecdsa.PublicKey, epoch int) error
}

//NymSignatureScheme是一个本地接口，用于与IDemix实现分离。
//与纽约商品交易所标志相关的操作
type NymSignatureScheme interface {
//签名创建新的IDemix假名签名
	Sign(sk Big, Nym Ecp, RNym Big, ipk IssuerPublicKey, digest []byte) ([]byte, error)

//验证验证Idemix NymSignature
	Verify(pk IssuerPublicKey, Nym Ecp, signature, digest []byte) error
}
