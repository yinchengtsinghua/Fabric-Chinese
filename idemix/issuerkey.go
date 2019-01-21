
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
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/pkg/errors"
)

//颁发者密钥isk和公共IPK密钥用于颁发凭据和
//验证使用凭据创建的签名

//颁发者密钥是随机指数（从z*u p随机生成）

//颁发者公钥由几个椭圆曲线点（ecp）组成，
//其中，索引1对应于组g1，2对应于组g2）
//hsk、hrand、barg1、barg2、ecp阵列hattrs和ecp2 w，
//以及相应密钥的知识证明

//NewIssuerKey创建一个新的具有属性名称数组的Issuer密钥对
//将包含在此颁发者认证的凭据中（凭据规范）
//参见http://eprint.iacr.org/2016/663.pdf第4.3，供参考。
func NewIssuerKey(AttributeNames []string, rng *amcl.RAND) (*IssuerKey, error) {
//验证输入

//检查重复的属性
	attributeNamesMap := map[string]bool{}
	for _, name := range AttributeNames {
		if attributeNamesMap[name] {
			return nil, errors.Errorf("attribute %s appears multiple times in AttributeNames", name)
		}
		attributeNamesMap[name] = true
	}

	key := new(IssuerKey)

//生成颁发者密钥
	ISk := RandModOrder(rng)
	key.Isk = BigToBytes(ISk)

//生成相应的公钥
	key.Ipk = new(IssuerPublicKey)
	key.Ipk.AttributeNames = AttributeNames

	W := GenG2.Mul(ISk)
	key.Ipk.W = Ecp2ToProto(W)

//生成与属性对应的基
	key.Ipk.HAttrs = make([]*ECP, len(AttributeNames))
	for i := 0; i < len(AttributeNames); i++ {
		key.Ipk.HAttrs[i] = EcpToProto(GenG1.Mul(RandModOrder(rng)))
	}

//为密钥生成基
	HSk := GenG1.Mul(RandModOrder(rng))
	key.Ipk.HSk = EcpToProto(HSk)

//为随机性生成基础
	HRand := GenG1.Mul(RandModOrder(rng))
	key.Ipk.HRand = EcpToProto(HRand)

	BarG1 := GenG1.Mul(RandModOrder(rng))
	key.Ipk.BarG1 = EcpToProto(BarG1)

	BarG2 := BarG1.Mul(ISk)
	key.Ipk.BarG2 = EcpToProto(BarG2)

//生成密钥的零知识证明（zk pok），其中
//在w和barg2中。

//抽样证明所需的随机性
	r := RandModOrder(rng)

//步骤1：第一条消息（t值）
t1 := GenG2.Mul(r) //T1=G_2^r，封面W
t2 := BarG1.Mul(r) //t2=（\bar g_1）^r，盖barg2

//第二步：计算fiat-shamir散列，形成zkp的挑战。
	proofData := make([]byte, 18*FieldBytes+3)
	index := 0
	index = appendBytesG2(proofData, index, t1)
	index = appendBytesG1(proofData, index, t2)
	index = appendBytesG2(proofData, index, GenG2)
	index = appendBytesG1(proofData, index, BarG1)
	index = appendBytesG2(proofData, index, W)
	index = appendBytesG1(proofData, index, BarG2)

	proofC := HashModOrder(proofData)
	key.Ipk.ProofC = BigToBytes(proofC)

//步骤3：回复挑战消息（S值）
proofS := Modadd(FP256BN.Modmul(proofC, ISk, GroupOrder), r, GroupOrder) ////s=r+c\cdot isk
	key.Ipk.ProofS = BigToBytes(proofS)

//哈希公钥
	serializedIPk, err := proto.Marshal(key.Ipk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal issuer public key")
	}
	key.Ipk.Hash = BigToBytes(HashModOrder(serializedIPk))

//我们完成了
	return key, nil
}

//检查此颁发者公钥是否有效，即
//所有组件都存在，并且一个zk证明可以验证
func (IPk *IssuerPublicKey) Check() error {
//取消标记公钥
	NumAttrs := len(IPk.GetAttributeNames())
	HSk := EcpFromProto(IPk.GetHSk())
	HRand := EcpFromProto(IPk.GetHRand())
	HAttrs := make([]*FP256BN.ECP, len(IPk.GetHAttrs()))
	for i := 0; i < len(IPk.GetHAttrs()); i++ {
		HAttrs[i] = EcpFromProto(IPk.GetHAttrs()[i])
	}
	BarG1 := EcpFromProto(IPk.GetBarG1())
	BarG2 := EcpFromProto(IPk.GetBarG2())
	W := Ecp2FromProto(IPk.GetW())
	ProofC := FP256BN.FromBytes(IPk.GetProofC())
	ProofS := FP256BN.FromBytes(IPk.GetProofS())

//检查公钥的格式是否正确
	if NumAttrs < 0 ||
		HSk == nil ||
		HRand == nil ||
		BarG1 == nil ||
		BarG1.Is_infinity() ||
		BarG2 == nil ||
		HAttrs == nil ||
		len(IPk.HAttrs) < NumAttrs {
		return errors.Errorf("some part of the public key is undefined")
	}
	for i := 0; i < NumAttrs; i++ {
		if IPk.HAttrs[i] == nil {
			return errors.Errorf("some part of the public key is undefined")
		}
	}

//验证证明

//重新计算挑战
	proofData := make([]byte, 18*FieldBytes+3)
	index := 0

//使用s值重新计算t值
	t1 := GenG2.Mul(ProofS)
t1.Add(W.Mul(FP256BN.Modneg(ProofC, GroupOrder))) //T1=G-C

	t2 := BarG1.Mul(ProofS)
t2.Add(BarG2.Mul(FP256BN.Modneg(ProofC, GroupOrder))) //t2=\bar g_1 ^s\cdot \bar g_2 ^c

	index = appendBytesG2(proofData, index, t1)
	index = appendBytesG1(proofData, index, t2)
	index = appendBytesG2(proofData, index, GenG2)
	index = appendBytesG1(proofData, index, BarG1)
	index = appendBytesG2(proofData, index, W)
	index = appendBytesG1(proofData, index, BarG2)

//验证挑战是否相同
	if *ProofC != *HashModOrder(proofData) {
		return errors.Errorf("zero knowledge proof in public key invalid")
	}

	return IPk.SetHash()
}

//sethash附加序列化公钥的哈希
func (IPk *IssuerPublicKey) SetHash() error {
	IPk.Hash = nil
	serializedIPk, err := proto.Marshal(IPk)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal issuer public key")
	}
	IPk.Hash = BigToBytes(HashModOrder(serializedIPk))
	return nil
}
