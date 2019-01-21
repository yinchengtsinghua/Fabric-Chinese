
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


package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalECDSASignature(t *testing.T) {
	_, _, err := UnmarshalECDSASignature(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed unmashalling signature [")

	_, _, err = UnmarshalECDSASignature([]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed unmashalling signature [")

	_, _, err = UnmarshalECDSASignature([]byte{0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed unmashalling signature [")

	sigma, err := MarshalECDSASignature(big.NewInt(-1), big.NewInt(1))
	assert.NoError(t, err)
	_, _, err = UnmarshalECDSASignature(sigma)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature, R must be larger than zero")

	sigma, err = MarshalECDSASignature(big.NewInt(0), big.NewInt(1))
	assert.NoError(t, err)
	_, _, err = UnmarshalECDSASignature(sigma)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature, R must be larger than zero")

	sigma, err = MarshalECDSASignature(big.NewInt(1), big.NewInt(0))
	assert.NoError(t, err)
	_, _, err = UnmarshalECDSASignature(sigma)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature, S must be larger than zero")

	sigma, err = MarshalECDSASignature(big.NewInt(1), big.NewInt(-1))
	assert.NoError(t, err)
	_, _, err = UnmarshalECDSASignature(sigma)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature, S must be larger than zero")

	sigma, err = MarshalECDSASignature(big.NewInt(1), big.NewInt(1))
	assert.NoError(t, err)
	R, S, err := UnmarshalECDSASignature(sigma)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(1), R)
	assert.Equal(t, big.NewInt(1), S)
}

func TestIsLowS(t *testing.T) {
	lowLevelKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	lowS, err := IsLowS(&lowLevelKey.PublicKey, big.NewInt(0))
	assert.NoError(t, err)
	assert.True(t, lowS)

	s := new(big.Int)
	s = s.Set(GetCurveHalfOrdersAt(elliptic.P256()))

	lowS, err = IsLowS(&lowLevelKey.PublicKey, s)
	assert.NoError(t, err)
	assert.True(t, lowS)

	s = s.Add(s, big.NewInt(1))
	lowS, err = IsLowS(&lowLevelKey.PublicKey, s)
	assert.NoError(t, err)
	assert.False(t, lowS)
	s, modified, err := ToLowS(&lowLevelKey.PublicKey, s)
	assert.NoError(t, err)
	assert.True(t, modified)
	lowS, err = IsLowS(&lowLevelKey.PublicKey, s)
	assert.NoError(t, err)
	assert.True(t, lowS)
}

func TestSignatureToLowS(t *testing.T) {
	lowLevelKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	s := new(big.Int)
	s = s.Set(GetCurveHalfOrdersAt(elliptic.P256()))
	s = s.Add(s, big.NewInt(1))

	lowS, err := IsLowS(&lowLevelKey.PublicKey, s)
	assert.NoError(t, err)
	assert.False(t, lowS)
	sigma, err := MarshalECDSASignature(big.NewInt(1), s)
	assert.NoError(t, err)
	sigma2, err := SignatureToLowS(&lowLevelKey.PublicKey, sigma)
	assert.NoError(t, err)
	_, s, err = UnmarshalECDSASignature(sigma2)
	assert.NoError(t, err)
	lowS, err = IsLowS(&lowLevelKey.PublicKey, s)
	assert.NoError(t, err)
	assert.True(t, lowS)
}
