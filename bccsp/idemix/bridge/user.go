
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

package bridge

import (
	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric/bccsp/idemix/handlers"
	cryptolib "github.com/hyperledger/fabric/idemix"
	"github.com/pkg/errors"
)

//用户封装IDemix算法以生成用户密钥和假名。
type User struct {
	NewRand func() *amcl.RAND
}

//newkey生成IDemix用户密钥
func (u *User) NewKey() (res handlers.Big, err error) {
	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	res = &Big{E: cryptolib.RandModOrder(u.NewRand())}

	return
}

func (*User) NewKeyFromBytes(raw []byte) (res handlers.Big, err error) {
	if len(raw) != int(FP256BN.MODBYTES) {
		return nil, errors.Errorf("invalid length, expected [%d], got [%d]", FP256BN.MODBYTES, len(raw))
	}

	res = &Big{E: FP256BN.FromBytes(raw)}

	return
}

//makenym从传递的用户密钥（sk）和颁发者公钥（ipk）生成一个新的假名密钥对。
func (u *User) MakeNym(sk handlers.Big, ipk handlers.IssuerPublicKey) (r1 handlers.Ecp, r2 handlers.Big, err error) {
	defer func() {
		if r := recover(); r != nil {
			r1 = nil
			r2 = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

	isk, ok := sk.(*Big)
	if !ok {
		return nil, nil, errors.Errorf("invalid user secret key, expected *Big, got [%T]", sk)
	}
	iipk, ok := ipk.(*IssuerPublicKey)
	if !ok {
		return nil, nil, errors.Errorf("invalid issuer public key, expected *IssuerPublicKey, got [%T]", ipk)
	}

	ecp, big := cryptolib.MakeNym(isk.E, iipk.PK, u.NewRand())

	r1 = &Ecp{E: ecp}
	r2 = &Big{E: big}

	return
}

func (*User) NewPublicNymFromBytes(raw []byte) (r handlers.Ecp, err error) {
	defer func() {
		if r := recover(); r != nil {
			r = nil
			err = errors.Errorf("failure [%s]", r)
		}
	}()

//raw是两个大整数的串联
	lHalve := len(raw) / 2

	r = &Ecp{E: FP256BN.NewECPbigs(FP256BN.FromBytes(raw[:lHalve]), FP256BN.FromBytes(raw[lHalve:]))}

	return
}
