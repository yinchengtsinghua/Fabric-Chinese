
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

//Identity Mixer凭证是由颁发者认证（签名）的属性列表。
//凭证还包含由颁发者盲目签名的用户密钥。
//如果没有密钥，则无法使用凭据

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

//NewCredential颁发新的凭据，这是交互式颁发协议的最后一步
//在此步骤中，所有属性值都由颁发者添加，然后与承诺一起签名
//来自凭据请求的用户密钥
func NewCredential(key *IssuerKey, m *CredRequest, attrs []*FP256BN.BIG, rng *amcl.RAND) (*Credential, error) {
//检查包含
	err := m.Check(key.Ipk)
	if err != nil {
		return nil, err
	}

	if len(attrs) != len(key.Ipk.AttributeNames) {
		return nil, errors.Errorf("incorrect number of attribute values passed")
	}

//在用户密钥和属性值上放置bbs+签名
//（对于BBS+，请参见Man Ho Au、Willy Susilo、Yi Mu的“恒定尺寸动态K-TAA”）。
//或http://eprint.iacr.org/2016/663.pdf，第4.3。

//对于凭证，BBS+签名由以下三个元素组成：
//1。e，适当组中的随机值
//2。s，适当组中的随机值
//三。a作为b^exp，其中b=g_1 \cdot h_s \cdot h_sk ^sk \cdot \prod_i=1_l h_i m_i_exp=\frac_1_123;e+x_
//注意：
//h_r是h_0，见http://eprint.iacr.org/2016/663.pdf，第2节。4.3。

//选择随机性E和S
	E := RandModOrder(rng)
	S := RandModOrder(rng)

//将b设置为g_1 \cdot h_r^s \cdot h_sk ^sk \cdot \prod_i=1_l h_i m_i_exp=\frac 1_123;e+x_
	B := FP256BN.NewECP()
B.Copy(GenG1) //GY1
	Nym := EcpFromProto(m.Nym)
B.Add(Nym)                                //在这种情况下，调用nym=h_sk^sk
B.Add(EcpFromProto(key.Ipk.HRand).Mul(S)) //HyrRs

//附加属性
//出于效率原因，尽可能使用mul2而不是mul
	for i := 0; i < len(attrs)/2; i++ {
		B.Add(
//一次添加两个属性
			EcpFromProto(key.Ipk.HAttrs[2*i]).Mul2(
				attrs[2*i],
				EcpFromProto(key.Ipk.HAttrs[2*i+1]),
				attrs[2*i+1],
			),
		)
	}
//检查len（attrs）为奇数时的残留物
	if len(attrs)%2 != 0 {
		B.Add(EcpFromProto(key.Ipk.HAttrs[len(attrs)-1]).Mul(attrs[len(attrs)-1]))
	}

//将exp设置为\frac 1 e+x
	Exp := Modadd(FP256BN.FromBytes(key.GetIsk()), E, GroupOrder)
	Exp.Invmodp(GroupOrder)
//最终确定A为B^exp
	A := B.Mul(Exp)
//现在生成签名。

//注意这里我们也释放B，这不会伤害安全原因
//它可以从bbs+签名本身公开计算。
	CredAttrs := make([][]byte, len(attrs))
	for index, attribute := range attrs {
		CredAttrs[index] = BigToBytes(attribute)
	}

	return &Credential{
		A:     EcpToProto(A),
		B:     EcpToProto(B),
		E:     BigToBytes(E),
		S:     BigToBytes(S),
		Attrs: CredAttrs}, nil
}

//Ver通过验证签名以加密方式验证凭证
//属性值和用户密钥
func (cred *Credential) Ver(sk *FP256BN.BIG, ipk *IssuerPublicKey) error {
//验证输入

//-分析凭证
	A := EcpFromProto(cred.GetA())
	B := EcpFromProto(cred.GetB())
	E := FP256BN.FromBytes(cred.GetE())
	S := FP256BN.FromBytes(cred.GetS())

//-验证是否存在所有属性值
	for i := 0; i < len(cred.GetAttrs()); i++ {
		if cred.Attrs[i] == nil {
			return errors.Errorf("credential has no value for attribute %s", ipk.AttributeNames[i])
		}
	}

//-验证属性和用户密钥上的加密签名
	BPrime := FP256BN.NewECP()
	BPrime.Copy(GenG1)
	BPrime.Add(EcpFromProto(ipk.HSk).Mul2(sk, EcpFromProto(ipk.HRand), S))
	for i := 0; i < len(cred.Attrs)/2; i++ {
		BPrime.Add(
			EcpFromProto(ipk.HAttrs[2*i]).Mul2(
				FP256BN.FromBytes(cred.Attrs[2*i]),
				EcpFromProto(ipk.HAttrs[2*i+1]),
				FP256BN.FromBytes(cred.Attrs[2*i+1]),
			),
		)
	}
	if len(cred.Attrs)%2 != 0 {
		BPrime.Add(EcpFromProto(ipk.HAttrs[len(cred.Attrs)-1]).Mul(FP256BN.FromBytes(cred.Attrs[len(cred.Attrs)-1])))
	}
	if !B.Equals(BPrime) {
		return errors.Errorf("b-value from credential does not match the attribute values")
	}

//验证bbs+签名。即：e（w\cdot g_2^e，a）=？e（g2，b）
	a := GenG2.Mul(E)
	a.Add(Ecp2FromProto(ipk.W))
	a.Affine()

	left := FP256BN.Fexp(FP256BN.Ate(a, A))
	right := FP256BN.Fexp(FP256BN.Ate(GenG2, B))

	if !left.Equals(right) {
		return errors.Errorf("credential is not cryptographically valid")
	}

	return nil
}
