
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

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/mocks"
	mocks2 "github.com/hyperledger/fabric/bccsp/sw/mocks"
	"github.com/stretchr/testify/assert"
)

func TestKeyGenInvalidInputs(t *testing.T) {
//用返回存储错误的密钥存储初始化bccsp实例
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{StoreKeyErr: errors.New("cannot store key")})
	assert.NoError(t, err)

	_, err = csp.KeyGen(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Opts parameter. It must not be nil.")

	_, err = csp.KeyGen(&mocks.KeyGenOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyGenOpts' provided [")

	_, err = csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{})
	assert.Error(t, err, "Generation of a non-ephemeral key must fail. KeyStore is programmed to fail.")
	assert.Contains(t, err.Error(), "cannot store key")
}

func TestKeyDerivInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{StoreKeyErr: errors.New("cannot store key")})
	assert.NoError(t, err)

	_, err = csp.KeyDeriv(nil, &bccsp.ECDSAReRandKeyOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.KeyDeriv(&mocks.MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.KeyDeriv(&mocks.MockKey{}, &bccsp.ECDSAReRandKeyOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'Key' provided [")

	keyDerivers := make(map[reflect.Type]KeyDeriver)
	keyDerivers[reflect.TypeOf(&mocks.MockKey{})] = &mocks2.KeyDeriver{
		KeyArg:  &mocks.MockKey{},
		OptsArg: &mocks.KeyDerivOpts{EphemeralValue: false},
		Value:   nil,
		Err:     nil,
	}
	csp.(*CSP).KeyDerivers = keyDerivers
	_, err = csp.KeyDeriv(&mocks.MockKey{}, &mocks.KeyDerivOpts{EphemeralValue: false})
	assert.Error(t, err, "KeyDerivation of a non-ephemeral key must fail. KeyStore is programmed to fail.")
	assert.Contains(t, err.Error(), "cannot store key")
}

func TestKeyImportInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{})
	assert.NoError(t, err)

	_, err = csp.KeyImport(nil, &bccsp.AES256ImportKeyOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw. It must not be nil.")

	_, err = csp.KeyImport([]byte{0, 1, 2, 3, 4}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.KeyImport([]byte{0, 1, 2, 3, 4}, &mocks.KeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyImportOpts' provided [")
}

func TestGetKeyInvalidInputs(t *testing.T) {
//初始化一个bccsp实例，该实例的密钥存储区在get时返回错误
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{GetKeyErr: errors.New("cannot get key")})
	assert.NoError(t, err)

	_, err = csp.GetKey(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot get key")

//用返回给定密钥的密钥存储初始化bccsp实例
	k := &mocks.MockKey{}
	csp, err = NewWithParams(256, "SHA2", &mocks.KeyStore{GetKeyValue: k})
	assert.NoError(t, err)
//这里不需要滑雪
	k2, err := csp.GetKey(nil)
	assert.NoError(t, err)
	assert.Equal(t, k, k2, "Keys must be the same.")
}

func TestSignInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{})
	assert.NoError(t, err)

	_, err = csp.Sign(nil, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Sign(&mocks.MockKey{}, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid digest. Cannot be empty.")

	_, err = csp.Sign(&mocks.MockKey{}, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'SignKey' provided [")
}

func TestVerifyInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{})
	assert.NoError(t, err)

	_, err = csp.Verify(nil, []byte{1, 2, 3, 5}, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Verify(&mocks.MockKey{}, nil, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid signature. Cannot be empty.")

	_, err = csp.Verify(&mocks.MockKey{}, []byte{1, 2, 3, 5}, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid digest. Cannot be empty.")

	_, err = csp.Verify(&mocks.MockKey{}, []byte{1, 2, 3, 5}, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'VerifyKey' provided [")
}

func TestEncryptInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{})
	assert.NoError(t, err)

	_, err = csp.Encrypt(nil, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Encrypt(&mocks.MockKey{}, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'EncryptKey' provided [")
}

func TestDecryptInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{})
	assert.NoError(t, err)

	_, err = csp.Decrypt(nil, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Decrypt(&mocks.MockKey{}, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'DecryptKey' provided [")
}

func TestHashInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{})
	assert.NoError(t, err)

	_, err = csp.Hash(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.Hash(nil, &mocks.HashOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'HashOpt' provided [")
}

func TestGetHashInvalidInputs(t *testing.T) {
	csp, err := NewWithParams(256, "SHA2", &mocks.KeyStore{})
	assert.NoError(t, err)

	_, err = csp.GetHash(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.GetHash(&mocks.HashOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'HashOpt' provided [")
}
