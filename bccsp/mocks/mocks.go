
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


package mocks

import (
	"bytes"
	"crypto"
	"errors"
	"hash"
	"reflect"

	"github.com/hyperledger/fabric/bccsp"
)

type MockBCCSP struct {
	SignArgKey    bccsp.Key
	SignDigestArg []byte
	SignOptsArg   bccsp.SignerOpts

	SignValue []byte
	SignErr   error

	VerifyValue bool
	VerifyErr   error

	ExpectedSig []byte

	KeyImportValue bccsp.Key
	KeyImportErr   error

	EncryptError error
	DecryptError error

	HashVal []byte
	HashErr error
}

func (*MockBCCSP) KeyGen(opts bccsp.KeyGenOpts) (bccsp.Key, error) {
	panic("Not yet implemented")
}

func (*MockBCCSP) KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (bccsp.Key, error) {
	panic("Not yet implemented")
}

func (m *MockBCCSP) KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (bccsp.Key, error) {
	return m.KeyImportValue, m.KeyImportErr
}

func (*MockBCCSP) GetKey(ski []byte) (bccsp.Key, error) {
	panic("Not yet implemented")
}

func (m *MockBCCSP) Hash(msg []byte, opts bccsp.HashOpts) ([]byte, error) {
	return m.HashVal, m.HashErr
}

func (*MockBCCSP) GetHash(opts bccsp.HashOpts) (hash.Hash, error) {
	panic("Not yet implemented")
}

func (b *MockBCCSP) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) ([]byte, error) {
	if !reflect.DeepEqual(b.SignArgKey, k) {
		return nil, errors.New("invalid key")
	}
	if !reflect.DeepEqual(b.SignDigestArg, digest) {
		return nil, errors.New("invalid digest")
	}
	if !reflect.DeepEqual(b.SignOptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return b.SignValue, b.SignErr
}

func (b *MockBCCSP) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (bool, error) {
//我们想嘲笑成功
	if b.VerifyValue {
		return b.VerifyValue, nil
	}

//我们想嘲笑一个错误导致的失败
	if b.VerifyErr != nil {
		return b.VerifyValue, b.VerifyErr
	}

//在这两种情况下，将签名与预期的签名进行比较
	return bytes.Equal(b.ExpectedSig, signature), nil
}

func (m *MockBCCSP) Encrypt(k bccsp.Key, plaintext []byte, opts bccsp.EncrypterOpts) ([]byte, error) {
	if m.EncryptError == nil {
		return plaintext, nil
	} else {
		return nil, m.EncryptError
	}
}

func (m *MockBCCSP) Decrypt(k bccsp.Key, ciphertext []byte, opts bccsp.DecrypterOpts) ([]byte, error) {
	if m.DecryptError == nil {
		return ciphertext, nil
	} else {
		return nil, m.DecryptError
	}
}

type MockKey struct {
	BytesValue []byte
	BytesErr   error
	Symm       bool
	PK         bccsp.Key
	PKErr      error
	Pvt        bool
}

func (m *MockKey) Bytes() ([]byte, error) {
	return m.BytesValue, m.BytesErr
}

func (*MockKey) SKI() []byte {
	panic("Not yet implemented")
}

func (m *MockKey) Symmetric() bool {
	return m.Symm
}

func (m *MockKey) Private() bool {
	return m.Pvt
}

func (m *MockKey) PublicKey() (bccsp.Key, error) {
	return m.PK, m.PKErr
}

type SignerOpts struct {
	HashFuncValue crypto.Hash
}

func (o *SignerOpts) HashFunc() crypto.Hash {
	return o.HashFuncValue
}

type KeyGenOpts struct {
	EphemeralValue bool
}

func (*KeyGenOpts) Algorithm() string {
	return "Mock KeyGenOpts"
}

func (o *KeyGenOpts) Ephemeral() bool {
	return o.EphemeralValue
}

type KeyStore struct {
	GetKeyValue bccsp.Key
	GetKeyErr   error
	StoreKeyErr error
}

func (*KeyStore) ReadOnly() bool {
	panic("Not yet implemented")
}

func (ks *KeyStore) GetKey(ski []byte) (bccsp.Key, error) {
	return ks.GetKeyValue, ks.GetKeyErr
}

func (ks *KeyStore) StoreKey(k bccsp.Key) error {
	return ks.StoreKeyErr
}

type KeyImportOpts struct{}

func (*KeyImportOpts) Algorithm() string {
	return "Mock KeyImportOpts"
}

func (*KeyImportOpts) Ephemeral() bool {
	panic("Not yet implemented")
}

type EncrypterOpts struct{}
type DecrypterOpts struct{}

type HashOpts struct{}

func (HashOpts) Algorithm() string {
	return "Mock HashOpts"
}

type KeyDerivOpts struct {
	EphemeralValue bool
}

func (*KeyDerivOpts) Algorithm() string {
	return "Mock KeyDerivOpts"
}

func (o *KeyDerivOpts) Ephemeral() bool {
	return o.EphemeralValue
}
