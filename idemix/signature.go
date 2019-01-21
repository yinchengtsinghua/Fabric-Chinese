
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
	"crypto/ecdsa"
	"sort"

	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
)

var idemixLogger = flogging.MustGetLogger("idemix")

//signlabel是零知识证明（zkp）中用来标识该zkp是知识的签名的标签。
const signLabel = "sign"

//使用身份混合凭证生成的签名是所谓的知识签名
//（有关详细信息，请参阅C.P.Schnorr“智能卡的有效识别和签名”）。
//身份混合签名是对消息进行签名并证明（零知识）的知识签名。
//在凭证内签名的用户秘密（以及可能的属性）的知识
//由某个发行者发行的（与发行者公钥一起引用）
//使用正在签名的消息和颁发者的公钥验证签名。
//凭证中的一些属性可以选择性地公开，或者可以证明不同的语句
//凭证属性不会在清除时丢失它们
//使用X.509证书的标准签名和身份混合签名的区别是
//Identity Mixer提供的高级隐私功能（由于零知识证明）：
//-使用同一凭证生成的签名的不可链接性
//-选择性属性公开和属性上的谓词

//对所有不公开的属性索引进行切片
func hiddenIndices(Disclosure []byte) []int {
	HiddenIndices := make([]int, 0)
	for index, disclose := range Disclosure {
		if disclose == 0 {
			HiddenIndices = append(HiddenIndices, index)
		}
	}
	return HiddenIndices
}

//NewSignature创建新的Idemix签名（Schnorr类型签名）
//[]字节的公开控制公开哪些属性：
//如果disclosure[i]=0，则属性i保持隐藏，否则将被公开。
//我们要求撤销处理保持未公开（即披露[Rhindex]=0）。
//我们使用http://eprint.iacr.org/2016/663.pdf，sec.提供的零知识证明。4.5证明对BBS+签名的了解
func NewSignature(cred *Credential, sk *FP256BN.BIG, Nym *FP256BN.ECP, RNym *FP256BN.BIG, ipk *IssuerPublicKey, Disclosure []byte, msg []byte, rhIndex int, cri *CredentialRevocationInformation, rng *amcl.RAND) (*Signature, error) {
//验证输入
	if cred == nil || sk == nil || Nym == nil || RNym == nil || ipk == nil || rng == nil || cri == nil {
		return nil, errors.Errorf("cannot create idemix signature: received nil input")
	}

	if rhIndex < 0 || rhIndex >= len(ipk.AttributeNames) || len(Disclosure) != len(ipk.AttributeNames) {
		return nil, errors.Errorf("cannot create idemix signature: received invalid input")
	}

	if cri.RevocationAlg != int32(ALG_NO_REVOCATION) && Disclosure[rhIndex] == 1 {
		return nil, errors.Errorf("Attribute %d is disclosed but also used as revocation handle attribute, which should remain hidden.", rhIndex)
	}

//定位要隐藏的属性的索引并为其随机抽样
	HiddenIndices := hiddenIndices(Disclosure)

//生成所需的随机性r_1，r_2
	r1 := RandModOrder(rng)
	r2 := RandModOrder(rng)
//将r_3设为\分数1 r_1
	r3 := FP256BN.NewBIGcopy(r1)
	r3.Invmodp(GroupOrder)

//采样随机数
	Nonce := RandModOrder(rng)

//分析凭据
	A := EcpFromProto(cred.A)
	B := EcpFromProto(cred.B)

//随机化凭证

//计算a'作为^ r ！}
	APrime := FP256BN.G1mul(A, r1)

//将abar计算为‘^-e b ^ r1
	ABar := FP256BN.G1mul(B, r1)
	ABar.Sub(FP256BN.G1mul(APrime, FP256BN.FromBytes(cred.E)))

//计算b'为b^ r1/h_r2，其中h_r是h_r
	BPrime := FP256BN.G1mul(B, r1)
	HRand := EcpFromProto(ipk.HRand)
//从IPK分析h_sk_
	HSk := EcpFromProto(ipk.HSk)

	BPrime.Sub(FP256BN.G1mul(HRand, r2))

	S := FP256BN.FromBytes(cred.S)
	E := FP256BN.FromBytes(cred.E)

//计算s'as s-r_2\cdot r_3
	sPrime := Modsub(S, FP256BN.Modmul(r2, r3, GroupOrder), GroupOrder)

//此函数的其余部分构造非交互式零知识证明
//链接签名、未公开的属性和名称。

//示例用于计算zkp的承诺值（即t值）的随机性
	rSk := RandModOrder(rng)
	re := RandModOrder(rng)
	rR2 := RandModOrder(rng)
	rR3 := RandModOrder(rng)
	rSPrime := RandModOrder(rng)
	rRNym := RandModOrder(rng)

	rAttrs := make([]*FP256BN.BIG, len(HiddenIndices))
	for i := range HiddenIndices {
		rAttrs[i] = RandModOrder(rng)
	}

//首先计算非撤销证明。
//ZKP的挑战也需要依赖于它。
	prover, err := getNonRevocationProver(RevocationAlgorithm(cri.RevocationAlg))
	if err != nil {
		return nil, err
	}
	nonRevokedProofHashData, err := prover.getFSContribution(
		FP256BN.FromBytes(cred.Attrs[rhIndex]),
		rAttrs[sort.SearchInts(HiddenIndices, rhIndex)],
		cri,
		rng,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute non-revoked proof")
	}

//步骤1：第一条消息（t值）

//T1与认证知识有关（召回，它是BBS+签名）
t1 := APrime.Mul2(re, HRand, rR2) //A’^ R E \ D不H R R R R2

//t2：与登录（a、b、s、e）的未披露属性相关。
t2 := FP256BN.G1mul(HRand, rSPrime) //Hyr{{r{{'}}
t2.Add(BPrime.Mul2(rR3, HSk, rSk))  //B’^ R R3 \ D不H SK ^ R SK
	for i := 0; i < len(HiddenIndices)/2; i++ {
		t2.Add(
//\cdot h_2 \cdot i ^ r_attrs，i_
			EcpFromProto(ipk.HAttrs[HiddenIndices[2*i]]).Mul2(
				rAttrs[2*i],
				EcpFromProto(ipk.HAttrs[HiddenIndices[2*i+1]]),
				rAttrs[2*i+1],
			),
		)
	}
	if len(HiddenIndices)%2 != 0 {
		t2.Add(FP256BN.G1mul(EcpFromProto(ipk.HAttrs[HiddenIndices[len(HiddenIndices)-1]]), rAttrs[len(HiddenIndices)-1]))
	}

//t3与笔名背后的秘密知识有关，笔名也在（a、b、s、e）中签名。
t3 := HSk.Mul2(rSk, HRand, rRNym) //H SK R SK \ D不H R R RNYM

//第二步：计算fiat-shamir散列，形成zkp的挑战。

//计算fiat-shamir散列，形成zkp的挑战。
//ProofData是要散列的数据，它包括：
//签名标签
//g1的7个元素，每个元素取2*fieldbytes+1个字节
//长度为fieldbytes的一个bigint（颁发者公钥的哈希）
//公开的属性
//正在签名的邮件
//不可撤消证明所需的字节数
	proofData := make([]byte, len([]byte(signLabel))+7*(2*FieldBytes+1)+FieldBytes+len(Disclosure)+len(msg)+ProofBytes[RevocationAlgorithm(cri.RevocationAlg)])
	index := 0
	index = appendBytesString(proofData, index, signLabel)
	index = appendBytesG1(proofData, index, t1)
	index = appendBytesG1(proofData, index, t2)
	index = appendBytesG1(proofData, index, t3)
	index = appendBytesG1(proofData, index, APrime)
	index = appendBytesG1(proofData, index, ABar)
	index = appendBytesG1(proofData, index, BPrime)
	index = appendBytesG1(proofData, index, Nym)
	index = appendBytes(proofData, index, nonRevokedProofHashData)
	copy(proofData[index:], ipk.Hash)
	index = index + FieldBytes
	copy(proofData[index:], Disclosure)
	index = index + len(Disclosure)
	copy(proofData[index:], msg)
	c := HashModOrder(proofData)

//再次添加上一个哈希和nonce和hash以计算第二个哈希（C值）
	index = 0
	proofData = proofData[:2*FieldBytes]
	index = appendBytesBig(proofData, index, c)
	index = appendBytesBig(proofData, index, Nonce)
	ProofC := HashModOrder(proofData)

//步骤3：回复挑战消息（S值）
ProofSSk := Modadd(rSk, FP256BN.Modmul(ProofC, sk, GroupOrder), GroupOrder)             //S_sk=rsk+c\cdot sk
ProofSE := Modsub(re, FP256BN.Modmul(ProofC, E, GroupOrder), GroupOrder)                //Seu e=re+c\cdot e
ProofSR2 := Modadd(rR2, FP256BN.Modmul(ProofC, r2, GroupOrder), GroupOrder)             //S_r2=rr2+c\cdot r2
ProofSR3 := Modsub(rR3, FP256BN.Modmul(ProofC, r3, GroupOrder), GroupOrder)             //S r3=rr3+c\cdot r3
ProofSSPrime := Modadd(rSPrime, FP256BN.Modmul(ProofC, sPrime, GroupOrder), GroupOrder) //s_s'=rsprime+c\cdot sprime
ProofSRNym := Modadd(rRNym, FP256BN.Modmul(ProofC, RNym, GroupOrder), GroupOrder)       //S诳nym=rrnym+c\cdot rnym
	ProofSAttrs := make([][]byte, len(HiddenIndices))
	for i, j := range HiddenIndices {
		ProofSAttrs[i] = BigToBytes(
//S attrsi=rattrsi+c\cdot cred.attrs[j]
			Modadd(rAttrs[i], FP256BN.Modmul(ProofC, FP256BN.FromBytes(cred.Attrs[j]), GroupOrder), GroupOrder),
		)
	}

//计算吊销部分
	nonRevokedProof, err := prover.getNonRevokedProof(ProofC)
	if err != nil {
		return nil, err
	}

//我们完了。返回签名
	return &Signature{
			APrime:             EcpToProto(APrime),
			ABar:               EcpToProto(ABar),
			BPrime:             EcpToProto(BPrime),
			ProofC:             BigToBytes(ProofC),
			ProofSSk:           BigToBytes(ProofSSk),
			ProofSE:            BigToBytes(ProofSE),
			ProofSR2:           BigToBytes(ProofSR2),
			ProofSR3:           BigToBytes(ProofSR3),
			ProofSSPrime:       BigToBytes(ProofSSPrime),
			ProofSAttrs:        ProofSAttrs,
			Nonce:              BigToBytes(Nonce),
			Nym:                EcpToProto(Nym),
			ProofSRNym:         BigToBytes(ProofSRNym),
			RevocationEpochPk:  cri.EpochPk,
			RevocationPkSig:    cri.EpochPkSig,
			Epoch:              cri.Epoch,
			NonRevocationProof: nonRevokedProof},
		nil
}

//Ver验证IDemix签名
//披露控制其期望披露的属性
//attributeValues包含所需的属性值。
//此函数将检查如果属性i被公开，则第i个属性等于属性值[i]。
func (sig *Signature) Ver(Disclosure []byte, ipk *IssuerPublicKey, msg []byte, attributeValues []*FP256BN.BIG, rhIndex int, revPk *ecdsa.PublicKey, epoch int) error {
//验证输入
	if ipk == nil || revPk == nil {
		return errors.Errorf("cannot verify idemix signature: received nil input")
	}

	if rhIndex < 0 || rhIndex >= len(ipk.AttributeNames) || len(Disclosure) != len(ipk.AttributeNames) {
		return errors.Errorf("cannot verify idemix signature: received invalid input")
	}

	if sig.NonRevocationProof.RevocationAlg != int32(ALG_NO_REVOCATION) && Disclosure[rhIndex] == 1 {
		return errors.Errorf("Attribute %d is disclosed but is also used as revocation handle, which should remain hidden.", rhIndex)
	}

	HiddenIndices := hiddenIndices(Disclosure)

//解析签名
	APrime := EcpFromProto(sig.GetAPrime())
	ABar := EcpFromProto(sig.GetABar())
	BPrime := EcpFromProto(sig.GetBPrime())
	Nym := EcpFromProto(sig.GetNym())
	ProofC := FP256BN.FromBytes(sig.GetProofC())
	ProofSSk := FP256BN.FromBytes(sig.GetProofSSk())
	ProofSE := FP256BN.FromBytes(sig.GetProofSE())
	ProofSR2 := FP256BN.FromBytes(sig.GetProofSR2())
	ProofSR3 := FP256BN.FromBytes(sig.GetProofSR3())
	ProofSSPrime := FP256BN.FromBytes(sig.GetProofSSPrime())
	ProofSRNym := FP256BN.FromBytes(sig.GetProofSRNym())
	ProofSAttrs := make([]*FP256BN.BIG, len(sig.GetProofSAttrs()))

	if len(sig.ProofSAttrs) != len(HiddenIndices) {
		return errors.Errorf("signature invalid: incorrect amount of s-values for AttributeProofSpec")
	}
	for i, b := range sig.ProofSAttrs {
		ProofSAttrs[i] = FP256BN.FromBytes(b)
	}
	Nonce := FP256BN.FromBytes(sig.GetNonce())

//分析颁发者公钥
	W := Ecp2FromProto(ipk.W)
	HRand := EcpFromProto(ipk.HRand)
	HSk := EcpFromProto(ipk.HSk)

//验证签名
	if APrime.Is_infinity() {
		return errors.Errorf("signature invalid: APrime = 1")
	}
	temp1 := FP256BN.Ate(W, APrime)
	temp2 := FP256BN.Ate(GenG2, ABar)
	temp2.Inverse()
	temp1.Mul(temp2)
	if !FP256BN.Fexp(temp1).Isunity() {
		return errors.Errorf("signature invalid: APrime and ABar don't have the expected structure")
	}

//验证ZK证明

//恢复t值

//重新计算T1
	t1 := APrime.Mul2(ProofSE, HRand, ProofSR2)
	temp := FP256BN.NewECP()
	temp.Copy(ABar)
	temp.Sub(BPrime)
	t1.Sub(FP256BN.G1mul(temp, ProofC))

//重新计算T2
	t2 := FP256BN.G1mul(HRand, ProofSSPrime)
	t2.Add(BPrime.Mul2(ProofSR3, HSk, ProofSSk))
	for i := 0; i < len(HiddenIndices)/2; i++ {
		t2.Add(EcpFromProto(ipk.HAttrs[HiddenIndices[2*i]]).Mul2(ProofSAttrs[2*i], EcpFromProto(ipk.HAttrs[HiddenIndices[2*i+1]]), ProofSAttrs[2*i+1]))
	}
	if len(HiddenIndices)%2 != 0 {
		t2.Add(FP256BN.G1mul(EcpFromProto(ipk.HAttrs[HiddenIndices[len(HiddenIndices)-1]]), ProofSAttrs[len(HiddenIndices)-1]))
	}
	temp = FP256BN.NewECP()
	temp.Copy(GenG1)
	for index, disclose := range Disclosure {
		if disclose != 0 {
			temp.Add(FP256BN.G1mul(EcpFromProto(ipk.HAttrs[index]), attributeValues[index]))
		}
	}
	t2.Add(FP256BN.G1mul(temp, ProofC))

//重新计算T3
	t3 := HSk.Mul2(ProofSSk, HRand, ProofSRNym)
	t3.Sub(Nym.Mul(ProofC))

//从非撤销证明中添加贡献
	nonRevokedVer, err := getNonRevocationVerifier(RevocationAlgorithm(sig.NonRevocationProof.RevocationAlg))
	if err != nil {
		return err
	}

	i := sort.SearchInts(HiddenIndices, rhIndex)
	proofSRh := ProofSAttrs[i]
	nonRevokedProofBytes, err := nonRevokedVer.recomputeFSContribution(sig.NonRevocationProof, ProofC, Ecp2FromProto(sig.RevocationEpochPk), proofSRh)
	if err != nil {
		return err
	}

//重新计算挑战
//ProofData是要散列的数据，它包括：
//签名标签
//g1的7个元素，每个元素取2*fieldbytes+1个字节
//长度为fieldbytes的一个bigint（颁发者公钥的哈希）
//公开的属性
//已签名的消息
	proofData := make([]byte, len([]byte(signLabel))+7*(2*FieldBytes+1)+FieldBytes+len(Disclosure)+len(msg)+ProofBytes[RevocationAlgorithm(sig.NonRevocationProof.RevocationAlg)])
	index := 0
	index = appendBytesString(proofData, index, signLabel)
	index = appendBytesG1(proofData, index, t1)
	index = appendBytesG1(proofData, index, t2)
	index = appendBytesG1(proofData, index, t3)
	index = appendBytesG1(proofData, index, APrime)
	index = appendBytesG1(proofData, index, ABar)
	index = appendBytesG1(proofData, index, BPrime)
	index = appendBytesG1(proofData, index, Nym)
	index = appendBytes(proofData, index, nonRevokedProofBytes)
	copy(proofData[index:], ipk.Hash)
	index = index + FieldBytes
	copy(proofData[index:], Disclosure)
	index = index + len(Disclosure)
	copy(proofData[index:], msg)

	c := HashModOrder(proofData)
	index = 0
	proofData = proofData[:2*FieldBytes]
	index = appendBytesBig(proofData, index, c)
	index = appendBytesBig(proofData, index, Nonce)

	if *ProofC != *HashModOrder(proofData) {
//此调试行有助于确定发生不匹配的位置
		idemixLogger.Debugf("Signature Verification : \n"+
			"	[t1:%v]\n,"+
			"	[t2:%v]\n,"+
			"	[t3:%v]\n,"+
			"	[APrime:%v]\n,"+
			"	[ABar:%v]\n,"+
			"	[BPrime:%v]\n,"+
			"	[Nym:%v]\n,"+
			"	[nonRevokedProofBytes:%v]\n,"+
			"	[ipk.Hash:%v]\n,"+
			"	[Disclosure:%v]\n,"+
			"	[msg:%v]\n,",
			EcpToBytes(t1),
			EcpToBytes(t2),
			EcpToBytes(t3),
			EcpToBytes(APrime),
			EcpToBytes(ABar),
			EcpToBytes(BPrime),
			EcpToBytes(Nym),
			nonRevokedProofBytes,
			ipk.Hash,
			Disclosure,
			msg)
		return errors.Errorf("signature invalid: zero-knowledge proof is invalid")
	}

//签名有效
	return nil
}
