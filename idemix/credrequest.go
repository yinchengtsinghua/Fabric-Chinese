
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


package idemix

import (
	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/pkg/errors"
)

//CredRequestLabel是零知识证明（zkp）中用于标识此zkp是凭据请求的标签。
const credRequestLabel = "credRequest"

//凭证颁发是用户和颁发者之间的交互协议
//颁发者将其密钥、公钥和用户属性值作为输入
//用户以颁发者公钥和用户秘密作为输入
//发布协议包括以下步骤：
//1）颁发者向用户发送一个随机的nonce
//2）用户使用颁发者的公钥、用户机密和nonce作为输入创建凭证请求
//请求包括对用户秘密的承诺（可以看作是公钥）和零知识证明
//用户密钥知识
//用户向颁发者发送凭证请求
//3）颁发者通过验证零知识证明来验证凭证请求
//如果请求有效，则颁发者通过签署对密钥的承诺向用户颁发凭证。
//以及属性值，并将凭证发送回用户
//4）用户验证颁发者的签名并存储由
//签名值、用于创建签名的随机性、用户机密和属性值

//NewCredRequest创建新的凭证请求，这是交互式凭证颁发协议的第一条消息
//（从用户到颁发者）
func NewCredRequest(sk *FP256BN.BIG, IssuerNonce []byte, ipk *IssuerPublicKey, rng *amcl.RAND) *CredRequest {
//将nym设置为h_sk ^ sk
	HSk := EcpFromProto(ipk.HSk)
	Nym := HSk.Mul(sk)

//生成密钥的零知识证明（zk pok）

//抽样证明所需的随机性
	rSk := RandModOrder(rng)

//步骤1：第一条消息（t值）
t := HSk.Mul(rSk) //t=h_sk ^ r_sk，封面名称

//第二步：计算fiat-shamir散列，形成zkp的挑战。
//ProofData是要散列的数据，它包括：
//凭证请求标签
//g1的3个元素，每个元素取2*fieldbytes+1个字节
//长度为fieldbytes的颁发者公钥的哈希
//长度为fieldbytes的颁发者nonce
	proofData := make([]byte, len([]byte(credRequestLabel))+3*(2*FieldBytes+1)+2*FieldBytes)
	index := 0
	index = appendBytesString(proofData, index, credRequestLabel)
	index = appendBytesG1(proofData, index, t)
	index = appendBytesG1(proofData, index, HSk)
	index = appendBytesG1(proofData, index, Nym)
	index = appendBytes(proofData, index, IssuerNonce)
	copy(proofData[index:], ipk.Hash)
	proofC := HashModOrder(proofData)

//步骤3：回复挑战消息（S值）
proofS := Modadd(FP256BN.Modmul(proofC, sk, GroupOrder), rSk, GroupOrder) //s=r_sk+c\cdot sk

//多恩
	return &CredRequest{
		Nym:         EcpToProto(Nym),
		IssuerNonce: IssuerNonce,
		ProofC:      BigToBytes(proofC),
		ProofS:      BigToBytes(proofS)}
}

//通过密码检查验证凭证请求
func (m *CredRequest) Check(ipk *IssuerPublicKey) error {
	Nym := EcpFromProto(m.GetNym())
	IssuerNonce := m.GetIssuerNonce()
	ProofC := FP256BN.FromBytes(m.GetProofC())
	ProofS := FP256BN.FromBytes(m.GetProofS())

	HSk := EcpFromProto(ipk.HSk)

	if Nym == nil || IssuerNonce == nil || ProofC == nil || ProofS == nil {
		return errors.Errorf("one of the proof values is undefined")
	}

//验证证明

//使用s值重新计算t值
	t := HSk.Mul(ProofS)
t.Sub(Nym.Mul(ProofC)) //t=h_sk_s/名称

//重新计算挑战
	proofData := make([]byte, len([]byte(credRequestLabel))+3*(2*FieldBytes+1)+2*FieldBytes)
	index := 0
	index = appendBytesString(proofData, index, credRequestLabel)
	index = appendBytesG1(proofData, index, t)
	index = appendBytesG1(proofData, index, HSk)
	index = appendBytesG1(proofData, index, Nym)
	index = appendBytes(proofData, index, IssuerNonce)
	copy(proofData[index:], ipk.Hash)

	if *ProofC != *HashModOrder(proofData) {
		return errors.Errorf("zero knowledge proof is invalid")
	}

	return nil
}
