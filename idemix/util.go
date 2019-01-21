
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
	"crypto/rand"
	"crypto/sha256"

	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/pkg/errors"
)

//geng1是g1组的生成器
var GenG1 = FP256BN.NewECPbigs(
	FP256BN.NewBIGints(FP256BN.CURVE_Gx),
	FP256BN.NewBIGints(FP256BN.CURVE_Gy))

//geng2是g2组的生成器
var GenG2 = FP256BN.NewECP2fp2s(
	FP256BN.NewFP2bigs(FP256BN.NewBIGints(FP256BN.CURVE_Pxa), FP256BN.NewBIGints(FP256BN.CURVE_Pxb)),
	FP256BN.NewFP2bigs(FP256BN.NewBIGints(FP256BN.CURVE_Pya), FP256BN.NewBIGints(FP256BN.CURVE_Pyb)))

//gengt是gt组的生成器
var GenGT = FP256BN.Fexp(FP256BN.Ate(GenG2, GenG1))

//GroupOrder是组的顺序
var GroupOrder = FP256BN.NewBIGints(FP256BN.CURVE_Order)

//FieldBytes是组顺序的字节长度
var FieldBytes = int(FP256BN.MODBYTES)

//randmodorder返回0，…，grouporder-1中的随机元素
func RandModOrder(rng *amcl.RAND) *FP256BN.BIG {
//曲线阶数q
	q := FP256BN.NewBIGints(FP256BN.CURVE_Order)

//取ZQ中的随机元素
	return FP256BN.Randomnum(q, rng)
}

//hashmodorder将数据散列为0，…，grouporder-1
func HashModOrder(data []byte) *FP256BN.BIG {
	digest := sha256.Sum256(data)
	digestBig := FP256BN.FromBytes(digest[:])
	digestBig.Mod(GroupOrder)
	return digestBig
}

func appendBytes(data []byte, index int, bytesToAdd []byte) int {
	copy(data[index:], bytesToAdd)
	return index + len(bytesToAdd)
}
func appendBytesG1(data []byte, index int, E *FP256BN.ECP) int {
	length := 2*FieldBytes + 1
	E.ToBytes(data[index:index+length], false)
	return index + length
}
func EcpToBytes(E *FP256BN.ECP) []byte {
	length := 2*FieldBytes + 1
	res := make([]byte, length)
	E.ToBytes(res, false)
	return res
}
func appendBytesG2(data []byte, index int, E *FP256BN.ECP2) int {
	length := 4 * FieldBytes
	E.ToBytes(data[index : index+length])
	return index + length
}
func appendBytesBig(data []byte, index int, B *FP256BN.BIG) int {
	length := FieldBytes
	B.ToBytes(data[index : index+length])
	return index + length
}
func appendBytesString(data []byte, index int, s string) int {
	bytes := []byte(s)
	copy(data[index:], bytes)
	return index + len(bytes)
}

//makenym创建一个新的不可链接的笔名
func MakeNym(sk *FP256BN.BIG, IPk *IssuerPublicKey, rng *amcl.RAND) (*FP256BN.ECP, *FP256BN.BIG) {
//构建对sk的承诺
//nym=h_sk_sk\cdot h_r^r
	RandNym := RandModOrder(rng)
	Nym := EcpFromProto(IPk.HSk).Mul2(sk, EcpFromProto(IPk.HRand), RandNym)
	return Nym, RandNym
}

//bigtobytes接受一个*amcl.big并返回一个[]字节的表示形式
func BigToBytes(big *FP256BN.BIG) []byte {
	ret := make([]byte, FieldBytes)
	big.ToBytes(ret)
	return ret
}

//ecptoproto将a*amcl.ecp转换为proto结构*ecp
func EcpToProto(p *FP256BN.ECP) *ECP {
	return &ECP{
		X: BigToBytes(p.GetX()),
		Y: BigToBytes(p.GetY())}
}

//ecpfromproto将proto结构*ecp转换为*amcl.ecp
func EcpFromProto(p *ECP) *FP256BN.ECP {
	return FP256BN.NewECPbigs(FP256BN.FromBytes(p.GetX()), FP256BN.FromBytes(p.GetY()))
}

//ecp2toproto将a*amcl.ecp2转换为proto结构*ecp2
func Ecp2ToProto(p *FP256BN.ECP2) *ECP2 {
	return &ECP2{
		Xa: BigToBytes(p.GetX().GetA()),
		Xb: BigToBytes(p.GetX().GetB()),
		Ya: BigToBytes(p.GetY().GetA()),
		Yb: BigToBytes(p.GetY().GetB())}
}

//ecp2fromproto将proto结构*ecp2转换为*amcl.ecp2
func Ecp2FromProto(p *ECP2) *FP256BN.ECP2 {
	return FP256BN.NewECP2fp2s(
		FP256BN.NewFP2bigs(FP256BN.FromBytes(p.GetXa()), FP256BN.FromBytes(p.GetXb())),
		FP256BN.NewFP2bigs(FP256BN.FromBytes(p.GetYa()), FP256BN.FromBytes(p.GetYb())))
}

//getrand返回一个新的带有新种子的amcl.rand
func GetRand() (*amcl.RAND, error) {
	seedLength := 32
	b := make([]byte, seedLength)
	_, err := rand.Read(b)
	if err != nil {
		return nil, errors.Wrap(err, "error getting randomness for seed")
	}
	rng := amcl.NewRAND()
	rng.Clean()
	rng.Seed(seedLength, b)
	return rng, nil
}

//modadd接受输入bigs A、B、M，并返回A+B模M
func Modadd(a, b, m *FP256BN.BIG) *FP256BN.BIG {
	c := a.Plus(b)
	c.Mod(m)
	return c
}

//modsub接受输入bigs a、b、m并返回a-b模m
func Modsub(a, b, m *FP256BN.BIG) *FP256BN.BIG {
	return Modadd(a, FP256BN.Modneg(b, m), m)
}
