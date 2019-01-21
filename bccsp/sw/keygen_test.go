
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
	"crypto/elliptic"
	"errors"
	"reflect"
	"testing"

	mocks2 "github.com/hyperledger/fabric/bccsp/mocks"
	"github.com/hyperledger/fabric/bccsp/sw/mocks"
	"github.com/stretchr/testify/assert"
)

func TestKeyGen(t *testing.T) {
	t.Parallel()

	expectedOpts := &mocks2.KeyGenOpts{EphemeralValue: true}
	expectetValue := &mocks2.MockKey{}
	expectedErr := errors.New("Expected Error")

	keyGenerators := make(map[reflect.Type]KeyGenerator)
	keyGenerators[reflect.TypeOf(&mocks2.KeyGenOpts{})] = &mocks.KeyGenerator{
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     expectedErr,
	}
	csp := CSP{KeyGenerators: keyGenerators}
	value, err := csp.KeyGen(expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())

	keyGenerators = make(map[reflect.Type]KeyGenerator)
	keyGenerators[reflect.TypeOf(&mocks2.KeyGenOpts{})] = &mocks.KeyGenerator{
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     nil,
	}
	csp = CSP{KeyGenerators: keyGenerators}
	value, err = csp.KeyGen(expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)
}

func TestECDSAKeyGenerator(t *testing.T) {
	t.Parallel()

	kg := &ecdsaKeyGenerator{curve: elliptic.P256()}

	k, err := kg.KeyGen(nil)
	assert.NoError(t, err)

	ecdsaK, ok := k.(*ecdsaPrivateKey)
	assert.True(t, ok)
	assert.NotNil(t, ecdsaK.privKey)
	assert.Equal(t, ecdsaK.privKey.Curve, elliptic.P256())
}

func TestRSAKeyGenerator(t *testing.T) {
	t.Parallel()

	kg := &rsaKeyGenerator{length: 512}

	k, err := kg.KeyGen(nil)
	assert.NoError(t, err)

	rsaK, ok := k.(*rsaPrivateKey)
	assert.True(t, ok)
	assert.NotNil(t, rsaK.privKey)
	assert.Equal(t, rsaK.privKey.N.BitLen(), 512)
}

func TestAESKeyGenerator(t *testing.T) {
	t.Parallel()

	kg := &aesKeyGenerator{length: 32}

	k, err := kg.KeyGen(nil)
	assert.NoError(t, err)

	aesK, ok := k.(*aesPrivateKey)
	assert.True(t, ok)
	assert.NotNil(t, aesK.privKey)
	assert.Equal(t, len(aesK.privKey), 32)
}

func TestAESKeyGeneratorInvalidInputs(t *testing.T) {
	t.Parallel()

	kg := &aesKeyGenerator{length: -1}

	_, err := kg.KeyGen(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Len must be larger than 0")
}

func TestRSAKeyGeneratorInvalidInputs(t *testing.T) {
	t.Parallel()

	kg := &rsaKeyGenerator{length: -1}

	_, err := kg.KeyGen(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed generating RSA -1 key")
}
