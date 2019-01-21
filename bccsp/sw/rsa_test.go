
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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/bccsp/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRSAPrivateKey(t *testing.T) {
	t.Parallel()

	lowLevelKey, err := rsa.GenerateKey(rand.Reader, 512)
	assert.NoError(t, err)
	k := &rsaPrivateKey{lowLevelKey}

	assert.False(t, k.Symmetric())
	assert.True(t, k.Private())

	_, err = k.Bytes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Not supported.")

	k.privKey = nil
	ski := k.SKI()
	assert.Nil(t, ski)

	k.privKey = lowLevelKey
	ski = k.SKI()
	raw, _ := asn1.Marshal(rsaPublicKeyASN{N: k.privKey.N, E: k.privKey.E})
	hash := sha256.New()
	hash.Write(raw)
	ski2 := hash.Sum(nil)
	assert.Equal(t, ski2, ski, "SKI is not computed in the right way.")

	pk, err := k.PublicKey()
	assert.NoError(t, err)
	assert.NotNil(t, pk)
	ecdsaPK, ok := pk.(*rsaPublicKey)
	assert.True(t, ok)
	assert.Equal(t, &lowLevelKey.PublicKey, ecdsaPK.pubKey)
}

func TestRSAPublicKey(t *testing.T) {
	t.Parallel()

	lowLevelKey, err := rsa.GenerateKey(rand.Reader, 512)
	assert.NoError(t, err)
	k := &rsaPublicKey{&lowLevelKey.PublicKey}

	assert.False(t, k.Symmetric())
	assert.False(t, k.Private())

	k.pubKey = nil
	ski := k.SKI()
	assert.Nil(t, ski)

	k.pubKey = &lowLevelKey.PublicKey
	ski = k.SKI()
	raw, _ := asn1.Marshal(rsaPublicKeyASN{N: k.pubKey.N, E: k.pubKey.E})
	hash := sha256.New()
	hash.Write(raw)
	ski2 := hash.Sum(nil)
	assert.Equal(t, ski, ski2, "SKI is not computed in the right way.")

	pk, err := k.PublicKey()
	assert.NoError(t, err)
	assert.Equal(t, k, pk)

	bytes, err := k.Bytes()
	assert.NoError(t, err)
	bytes2, err := x509.MarshalPKIXPublicKey(k.pubKey)
	assert.Equal(t, bytes2, bytes, "bytes are not computed in the right way.")
}

func TestRSASignerSign(t *testing.T) {
	t.Parallel()

	signer := &rsaSigner{}
	verifierPrivateKey := &rsaPrivateKeyVerifier{}
	verifierPublicKey := &rsaPublicKeyKeyVerifier{}

//生成密钥
	lowLevelKey, err := rsa.GenerateKey(rand.Reader, 1024)
	assert.NoError(t, err)
	k := &rsaPrivateKey{lowLevelKey}
	pk, err := k.PublicKey()
	assert.NoError(t, err)

//符号
	msg := []byte("Hello World!!!")

	_, err = signer.Sign(k, msg, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid options. Must be different from nil.")

	_, err = signer.Sign(k, msg, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	assert.Error(t, err)

	hf := sha256.New()
	hf.Write(msg)
	digest := hf.Sum(nil)
	sigma, err := signer.Sign(k, digest, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	assert.NoError(t, err)

	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256}
//根据消息验证，必须失败
	err = rsa.VerifyPSS(&lowLevelKey.PublicKey, crypto.SHA256, msg, sigma, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "crypto/rsa: verification error")

//根据摘要验证，必须成功
	err = rsa.VerifyPSS(&lowLevelKey.PublicKey, crypto.SHA256, digest, sigma, opts)
	assert.NoError(t, err)

	valid, err := verifierPrivateKey.Verify(k, sigma, msg, opts)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "crypto/rsa: verification error"))

	valid, err = verifierPrivateKey.Verify(k, sigma, digest, opts)
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = verifierPublicKey.Verify(pk, sigma, msg, opts)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "crypto/rsa: verification error"))

	valid, err = verifierPublicKey.Verify(pk, sigma, digest, opts)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestRSAVerifiersInvalidInputs(t *testing.T) {
	t.Parallel()

	verifierPrivate := &rsaPrivateKeyVerifier{}
	_, err := verifierPrivate.Verify(nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Invalid options. It must not be nil."))

	_, err = verifierPrivate.Verify(nil, nil, nil, &mocks.SignerOpts{})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Opts type not recognized ["))

	verifierPublic := &rsaPublicKeyKeyVerifier{}
	_, err = verifierPublic.Verify(nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Invalid options. It must not be nil."))

	_, err = verifierPublic.Verify(nil, nil, nil, &mocks.SignerOpts{})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Opts type not recognized ["))
}
