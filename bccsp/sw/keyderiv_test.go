
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
	"errors"
	"reflect"
	"testing"

	mocks2 "github.com/hyperledger/fabric/bccsp/mocks"
	"github.com/hyperledger/fabric/bccsp/sw/mocks"
	"github.com/stretchr/testify/assert"
)

func TestKeyDeriv(t *testing.T) {
	t.Parallel()

	expectedKey := &mocks2.MockKey{BytesValue: []byte{1, 2, 3}}
	expectedOpts := &mocks2.KeyDerivOpts{EphemeralValue: true}
	expectetValue := &mocks2.MockKey{BytesValue: []byte{1, 2, 3, 4, 5}}
	expectedErr := errors.New("Expected Error")

	keyDerivers := make(map[reflect.Type]KeyDeriver)
	keyDerivers[reflect.TypeOf(&mocks2.MockKey{})] = &mocks.KeyDeriver{
		KeyArg:  expectedKey,
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     expectedErr,
	}
	csp := CSP{KeyDerivers: keyDerivers}
	value, err := csp.KeyDeriv(expectedKey, expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())

	keyDerivers = make(map[reflect.Type]KeyDeriver)
	keyDerivers[reflect.TypeOf(&mocks2.MockKey{})] = &mocks.KeyDeriver{
		KeyArg:  expectedKey,
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     nil,
	}
	csp = CSP{KeyDerivers: keyDerivers}
	value, err = csp.KeyDeriv(expectedKey, expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)
}

func TestECDSAPublicKeyKeyDeriver(t *testing.T) {
	t.Parallel()

	kd := ecdsaPublicKeyKeyDeriver{}

	_, err := kd.KeyDeriv(&mocks2.MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts parameter. It must not be nil.")

	_, err = kd.KeyDeriv(&ecdsaPublicKey{}, &mocks2.KeyDerivOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyDerivOpts' provided [")
}

func TestECDSAPrivateKeyKeyDeriver(t *testing.T) {
	t.Parallel()

	kd := ecdsaPrivateKeyKeyDeriver{}

	_, err := kd.KeyDeriv(&mocks2.MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts parameter. It must not be nil.")

	_, err = kd.KeyDeriv(&ecdsaPrivateKey{}, &mocks2.KeyDerivOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyDerivOpts' provided [")
}

func TestAESPrivateKeyKeyDeriver(t *testing.T) {
	t.Parallel()

	kd := aesPrivateKeyKeyDeriver{}

	_, err := kd.KeyDeriv(&mocks2.MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts parameter. It must not be nil.")

	_, err = kd.KeyDeriv(&aesPrivateKey{}, &mocks2.KeyDerivOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyDerivOpts' provided [")
}
