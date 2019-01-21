
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

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

package signer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/hyperledger/fabric/bccsp/mocks"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/stretchr/testify/assert"
)

func TestInitFailures(t *testing.T) {
	_, err := New(nil, &mocks.MockKey{})
	assert.Error(t, err)

	_, err = New(&mocks.MockBCCSP{}, nil)
	assert.Error(t, err)

	_, err = New(&mocks.MockBCCSP{}, &mocks.MockKey{Symm: true})
	assert.Error(t, err)

	_, err = New(&mocks.MockBCCSP{}, &mocks.MockKey{PKErr: errors.New("No PK")})
	assert.Error(t, err)
	assert.Equal(t, "failed getting public key: No PK", err.Error())

	_, err = New(&mocks.MockBCCSP{}, &mocks.MockKey{PK: &mocks.MockKey{BytesErr: errors.New("No bytes")}})
	assert.Error(t, err)
	assert.Equal(t, "failed marshalling public key: No bytes", err.Error())

	_, err = New(&mocks.MockBCCSP{}, &mocks.MockKey{PK: &mocks.MockKey{BytesValue: []byte{0, 1, 2, 3}}})
	assert.Error(t, err)
}

func TestInit(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)
	pkRaw, err := utils.PublicKeyToDER(&k.PublicKey)
	assert.NoError(t, err)

	signer, err := New(&mocks.MockBCCSP{}, &mocks.MockKey{PK: &mocks.MockKey{BytesValue: pkRaw}})
	assert.NoError(t, err)
	assert.NotNil(t, signer)

//测试公钥
	R, S, err := ecdsa.Sign(rand.Reader, k, []byte{0, 1, 2, 3})
	assert.NoError(t, err)

	assert.True(t, ecdsa.Verify(signer.Public().(*ecdsa.PublicKey), []byte{0, 1, 2, 3}, R, S))
}

func TestPublic(t *testing.T) {
	pk := &mocks.MockKey{}
	signer := &bccspCryptoSigner{pk: pk}

	pk2 := signer.Public()
	assert.NotNil(t, pk, pk2)
}

func TestSign(t *testing.T) {
	expectedSig := []byte{0, 1, 2, 3, 4}
	expectedKey := &mocks.MockKey{}
	expectedDigest := []byte{0, 1, 2, 3, 4, 5}
	expectedOpts := &mocks.SignerOpts{}

	signer := &bccspCryptoSigner{
		key: expectedKey,
		csp: &mocks.MockBCCSP{
			SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts,
			SignValue: expectedSig}}
	signature, err := signer.Sign(nil, expectedDigest, expectedOpts)
	assert.NoError(t, err)
	assert.Equal(t, expectedSig, signature)

	signer = &bccspCryptoSigner{
		key: expectedKey,
		csp: &mocks.MockBCCSP{
			SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts,
			SignErr: errors.New("no signature")}}
	signature, err = signer.Sign(nil, expectedDigest, expectedOpts)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "no signature")

	signer = &bccspCryptoSigner{
		key: nil,
		csp: &mocks.MockBCCSP{SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts}}
	_, err = signer.Sign(nil, expectedDigest, expectedOpts)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid key")

	signer = &bccspCryptoSigner{
		key: expectedKey,
		csp: &mocks.MockBCCSP{SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts}}
	_, err = signer.Sign(nil, nil, expectedOpts)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid digest")

	signer = &bccspCryptoSigner{
		key: expectedKey,
		csp: &mocks.MockBCCSP{SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts}}
	_, err = signer.Sign(nil, expectedDigest, nil)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid opts")
}
