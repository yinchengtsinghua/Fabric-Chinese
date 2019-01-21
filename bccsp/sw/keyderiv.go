
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package sw

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"errors"
	"fmt"
	"math/big"

	"github.com/hyperledger/fabric/bccsp"
)

type ecdsaPublicKeyKeyDeriver struct{}

func (kd *ecdsaPublicKeyKeyDeriver) KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (bccsp.Key, error) {
//验证选择
	if opts == nil {
		return nil, errors.New("Invalid opts parameter. It must not be nil.")
	}

	ecdsaK := k.(*ecdsaPublicKey)

	switch opts.(type) {
//重新随机分配ECDSA私钥
	case *bccsp.ECDSAReRandKeyOpts:
		reRandOpts := opts.(*bccsp.ECDSAReRandKeyOpts)
		tempSK := &ecdsa.PublicKey{
			Curve: ecdsaK.pubKey.Curve,
			X:     new(big.Int),
			Y:     new(big.Int),
		}

		var k = new(big.Int).SetBytes(reRandOpts.ExpansionValue())
		var one = new(big.Int).SetInt64(1)
		n := new(big.Int).Sub(ecdsaK.pubKey.Params().N, one)
		k.Mod(k, n)
		k.Add(k, one)

//计算临时公钥
		tempX, tempY := ecdsaK.pubKey.ScalarBaseMult(k.Bytes())
		tempSK.X, tempSK.Y = tempSK.Add(
			ecdsaK.pubKey.X, ecdsaK.pubKey.Y,
			tempX, tempY,
		)

//验证临时公钥是否为参考曲线上的有效点
		isOn := tempSK.Curve.IsOnCurve(tempSK.X, tempSK.Y)
		if !isOn {
			return nil, errors.New("Failed temporary public key IsOnCurve check.")
		}

		return &ecdsaPublicKey{tempSK}, nil
	default:
		return nil, fmt.Errorf("Unsupported 'KeyDerivOpts' provided [%v]", opts)
	}
}

type ecdsaPrivateKeyKeyDeriver struct{}

func (kd *ecdsaPrivateKeyKeyDeriver) KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (bccsp.Key, error) {
//验证选择
	if opts == nil {
		return nil, errors.New("Invalid opts parameter. It must not be nil.")
	}

	ecdsaK := k.(*ecdsaPrivateKey)

	switch opts.(type) {
//重新随机分配ECDSA私钥
	case *bccsp.ECDSAReRandKeyOpts:
		reRandOpts := opts.(*bccsp.ECDSAReRandKeyOpts)
		tempSK := &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: ecdsaK.privKey.Curve,
				X:     new(big.Int),
				Y:     new(big.Int),
			},
			D: new(big.Int),
		}

		var k = new(big.Int).SetBytes(reRandOpts.ExpansionValue())
		var one = new(big.Int).SetInt64(1)
		n := new(big.Int).Sub(ecdsaK.privKey.Params().N, one)
		k.Mod(k, n)
		k.Add(k, one)

		tempSK.D.Add(ecdsaK.privKey.D, k)
		tempSK.D.Mod(tempSK.D, ecdsaK.privKey.PublicKey.Params().N)

//计算临时公钥
		tempX, tempY := ecdsaK.privKey.PublicKey.ScalarBaseMult(k.Bytes())
		tempSK.PublicKey.X, tempSK.PublicKey.Y =
			tempSK.PublicKey.Add(
				ecdsaK.privKey.PublicKey.X, ecdsaK.privKey.PublicKey.Y,
				tempX, tempY,
			)

//验证临时公钥是否为参考曲线上的有效点
		isOn := tempSK.Curve.IsOnCurve(tempSK.PublicKey.X, tempSK.PublicKey.Y)
		if !isOn {
			return nil, errors.New("Failed temporary public key IsOnCurve check.")
		}

		return &ecdsaPrivateKey{tempSK}, nil
	default:
		return nil, fmt.Errorf("Unsupported 'KeyDerivOpts' provided [%v]", opts)
	}
}

type aesPrivateKeyKeyDeriver struct {
	conf *config
}

func (kd *aesPrivateKeyKeyDeriver) KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (bccsp.Key, error) {
//验证选择
	if opts == nil {
		return nil, errors.New("Invalid opts parameter. It must not be nil.")
	}

	aesK := k.(*aesPrivateKey)

	switch opts.(type) {
	case *bccsp.HMACTruncated256AESDeriveKeyOpts:
		hmacOpts := opts.(*bccsp.HMACTruncated256AESDeriveKeyOpts)

		mac := hmac.New(kd.conf.hashFunction, aesK.privKey)
		mac.Write(hmacOpts.Argument())
		return &aesPrivateKey{mac.Sum(nil)[:kd.conf.aesBitLength], false}, nil

	case *bccsp.HMACDeriveKeyOpts:
		hmacOpts := opts.(*bccsp.HMACDeriveKeyOpts)

		mac := hmac.New(kd.conf.hashFunction, aesK.privKey)
		mac.Write(hmacOpts.Argument())
		return &aesPrivateKey{mac.Sum(nil), true}, nil
	default:
		return nil, fmt.Errorf("Unsupported 'KeyDerivOpts' provided [%v]", opts)
	}
}
