
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

//wbbkeygen创建了一个新的弱Boneh-Boyen签名密钥对（http://ia.cr/2004/171）
func WBBKeyGen(rng *amcl.RAND) (*FP256BN.BIG, *FP256BN.ECP2) {
//来自ZQ的样品sk均匀
	sk := RandModOrder(rng)
//设置PK＝G2 ^ SK
	pk := GenG2.Mul(sk)
	return sk, pk
}

//wbbsign使用密钥sk在消息m上放置一个弱的boneh-boyen签名
func WBBSign(sk *FP256BN.BIG, m *FP256BN.BIG) *FP256BN.ECP {
//计算exp=1/（m+sk）mod q
	exp := Modadd(sk, m, GroupOrder)
	exp.Invmodp(GroupOrder)

//返回签名sig=g1^（1/（m+sk））
	return GenG1.Mul(exp)
}

//wbbverify用公钥pk验证消息m上的弱Boneh Boyen签名sig
func WBBVerify(pk *FP256BN.ECP2, sig *FP256BN.ECP, m *FP256BN.BIG) error {
	if pk == nil || sig == nil || m == nil {
		return errors.Errorf("Weak-BB signature invalid: received nil input")
	}
//设置p=pk*g2^m
	P := FP256BN.NewECP2()
	P.Copy(pk)
	P.Add(GenG2.Mul(m))
	P.Affine()
//检查e（sig，pk*g2^m）=e（g1，g2）
	if !FP256BN.Fexp(FP256BN.Ate(P, sig)).Equals(GenGT) {
		return errors.Errorf("Weak-BB signature is invalid")
	}
	return nil
}
