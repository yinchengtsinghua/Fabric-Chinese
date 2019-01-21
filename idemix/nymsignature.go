
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

//NewSignature创建新的Idemix假名签名
func NewNymSignature(sk *FP256BN.BIG, Nym *FP256BN.ECP, RNym *FP256BN.BIG, ipk *IssuerPublicKey, msg []byte, rng *amcl.RAND) (*NymSignature, error) {
//验证输入
	if sk == nil || Nym == nil || RNym == nil || ipk == nil || rng == nil {
		return nil, errors.Errorf("cannot create NymSignature: received nil input")
	}

	Nonce := RandModOrder(rng)

	HRand := EcpFromProto(ipk.HRand)
	HSk := EcpFromProto(ipk.HSk)

//该函数的其余部分构造了非交互式零知识证明，证明
//签名者“拥有”这个笔名，也就是说，它知道它所基于的秘密密钥和随机性。
//回想一下（nym，rnym）是makenym的输出。因此，nym=h_sk_sk\cdot h_r^r

//抽样证明所需的随机性
	rSk := RandModOrder(rng)
	rRNym := RandModOrder(rng)

//步骤1：第一条消息（t值）
t := HSk.Mul2(rSk, HRand, rRNym) //t=h_sk r_sk \cdot h_r r_rnym

//第二步：计算fiat-shamir散列，形成zkp的挑战。
//校对数据将保存正在散列的数据，它包括：
//-签名标签
//-g1的2个元素，每个元素取2*fieldbytes+1个字节
//-长度为fieldbytes的一个bigint（颁发者公钥的哈希）
//-披露的属性
//-正在签名的消息
	proofData := make([]byte, len([]byte(signLabel))+2*(2*FieldBytes+1)+FieldBytes+len(msg))
	index := 0
	index = appendBytesString(proofData, index, signLabel)
	index = appendBytesG1(proofData, index, t)
	index = appendBytesG1(proofData, index, Nym)
	copy(proofData[index:], ipk.Hash)
	index = index + FieldBytes
	copy(proofData[index:], msg)
	c := HashModOrder(proofData)
//将前一个哈希与nonce和hash再次组合，以计算最终fiat shamir值“proofc”
	index = 0
	proofData = proofData[:2*FieldBytes]
	index = appendBytesBig(proofData, index, c)
	index = appendBytesBig(proofData, index, Nonce)
	ProofC := HashModOrder(proofData)

//步骤3：回复挑战消息（S值）
ProofSSk := Modadd(rSk, FP256BN.Modmul(ProofC, sk, GroupOrder), GroupOrder)       //S sk=R sk+C\cdot sk
ProofSRNym := Modadd(rRNym, FP256BN.Modmul(ProofC, RNym, GroupOrder), GroupOrder) //S RNYM=R RNYM+C \ CDOT RNYM

//签名由fiat shamir散列（proofc）、s值（proofssk、proofsrynm）和nonce组成。
	return &NymSignature{
		ProofC:     BigToBytes(ProofC),
		ProofSSk:   BigToBytes(ProofSSk),
		ProofSRNym: BigToBytes(ProofSRNym),
		Nonce:      BigToBytes(Nonce)}, nil
}

//验证IDemix NymSignature
func (sig *NymSignature) Ver(nym *FP256BN.ECP, ipk *IssuerPublicKey, msg []byte) error {
	ProofC := FP256BN.FromBytes(sig.GetProofC())
	ProofSSk := FP256BN.FromBytes(sig.GetProofSSk())
	ProofSRNym := FP256BN.FromBytes(sig.GetProofSRNym())
	Nonce := FP256BN.FromBytes(sig.GetNonce())

	HRand := EcpFromProto(ipk.HRand)
	HSk := EcpFromProto(ipk.HSk)

//验证证明

//使用s值重新计算t值
	t := HSk.Mul2(ProofSSk, HRand, ProofSRNym)
t.Sub(nym.Mul(ProofC)) //t=h_sk s_sk \cdot h_r s_rnym

//重新计算挑战
	proofData := make([]byte, len([]byte(signLabel))+2*(2*FieldBytes+1)+FieldBytes+len(msg))
	index := 0
	index = appendBytesString(proofData, index, signLabel)
	index = appendBytesG1(proofData, index, t)
	index = appendBytesG1(proofData, index, nym)
	copy(proofData[index:], ipk.Hash)
	index = index + FieldBytes
	copy(proofData[index:], msg)
	c := HashModOrder(proofData)
	index = 0
	proofData = proofData[:2*FieldBytes]
	index = appendBytesBig(proofData, index, c)
	index = appendBytesBig(proofData, index, Nonce)

	if *ProofC != *HashModOrder(proofData) {
		return errors.Errorf("pseudonym signature invalid: zero-knowledge proof is invalid")
	}

	return nil
}
