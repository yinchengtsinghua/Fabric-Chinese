
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

package sw

import (
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/sha512"
	"reflect"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

//NewDefaultSecurityLevel返回基于软件的BCCSP的新实例
//在安全级别256，散列家族sha2，并使用folderbasedKeystore作为密钥库。
func NewDefaultSecurityLevel(keyStorePath string) (bccsp.BCCSP, error) {
	ks := &fileBasedKeyStore{}
	if err := ks.Init(nil, keyStorePath, false); err != nil {
		return nil, errors.Wrapf(err, "Failed initializing key store at [%v]", keyStorePath)
	}

	return NewWithParams(256, "SHA2", ks)
}

//NewDefaultSecurityLevel返回基于软件的BCCSP的新实例
//在安全级别256上，散列家族sha2并使用传递的密钥库。
func NewDefaultSecurityLevelWithKeystore(keyStore bccsp.KeyStore) (bccsp.BCCSP, error) {
	return NewWithParams(256, "SHA2", keyStore)
}

//newwithparams返回基于软件的BCCSP的新实例
//设置为通过的安全级别、哈希系列和密钥库。
func NewWithParams(securityLevel int, hashFamily string, keyStore bccsp.KeyStore) (bccsp.BCCSP, error) {
//初始化配置
	conf := &config{}
	err := conf.setSecurityLevel(securityLevel, hashFamily)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed initializing configuration at [%v,%v]", securityLevel, hashFamily)
	}

	swbccsp, err := New(keyStore)
	if err != nil {
		return nil, err
	}

//注意这里忽略了错误，因为如果有一个测试失败了
//以下调用失败。

//设置加密机
	swbccsp.AddWrapper(reflect.TypeOf(&aesPrivateKey{}), &aescbcpkcs7Encryptor{})

//设置解密程序
	swbccsp.AddWrapper(reflect.TypeOf(&aesPrivateKey{}), &aescbcpkcs7Decryptor{})

//设置签名者
	swbccsp.AddWrapper(reflect.TypeOf(&ecdsaPrivateKey{}), &ecdsaSigner{})
	swbccsp.AddWrapper(reflect.TypeOf(&rsaPrivateKey{}), &rsaSigner{})

//设置验证程序
	swbccsp.AddWrapper(reflect.TypeOf(&ecdsaPrivateKey{}), &ecdsaPrivateKeyVerifier{})
	swbccsp.AddWrapper(reflect.TypeOf(&ecdsaPublicKey{}), &ecdsaPublicKeyKeyVerifier{})
	swbccsp.AddWrapper(reflect.TypeOf(&rsaPrivateKey{}), &rsaPrivateKeyVerifier{})
	swbccsp.AddWrapper(reflect.TypeOf(&rsaPublicKey{}), &rsaPublicKeyKeyVerifier{})

//设置哈希
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.SHAOpts{}), &hasher{hash: conf.hashFunction})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.SHA256Opts{}), &hasher{hash: sha256.New})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.SHA384Opts{}), &hasher{hash: sha512.New384})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.SHA3_256Opts{}), &hasher{hash: sha3.New256})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.SHA3_384Opts{}), &hasher{hash: sha3.New384})

//设置密钥生成器
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.ECDSAKeyGenOpts{}), &ecdsaKeyGenerator{curve: conf.ellipticCurve})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.ECDSAP256KeyGenOpts{}), &ecdsaKeyGenerator{curve: elliptic.P256()})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.ECDSAP384KeyGenOpts{}), &ecdsaKeyGenerator{curve: elliptic.P384()})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.AESKeyGenOpts{}), &aesKeyGenerator{length: conf.aesBitLength})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.AES256KeyGenOpts{}), &aesKeyGenerator{length: 32})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.AES192KeyGenOpts{}), &aesKeyGenerator{length: 24})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.AES128KeyGenOpts{}), &aesKeyGenerator{length: 16})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.RSAKeyGenOpts{}), &rsaKeyGenerator{length: conf.rsaBitLength})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.RSA1024KeyGenOpts{}), &rsaKeyGenerator{length: 1024})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.RSA2048KeyGenOpts{}), &rsaKeyGenerator{length: 2048})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.RSA3072KeyGenOpts{}), &rsaKeyGenerator{length: 3072})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.RSA4096KeyGenOpts{}), &rsaKeyGenerator{length: 4096})

//设置密钥生成器
	swbccsp.AddWrapper(reflect.TypeOf(&ecdsaPrivateKey{}), &ecdsaPrivateKeyKeyDeriver{})
	swbccsp.AddWrapper(reflect.TypeOf(&ecdsaPublicKey{}), &ecdsaPublicKeyKeyDeriver{})
	swbccsp.AddWrapper(reflect.TypeOf(&aesPrivateKey{}), &aesPrivateKeyKeyDeriver{conf: conf})

//设置密钥导入程序
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.AES256ImportKeyOpts{}), &aes256ImportKeyOptsKeyImporter{})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.HMACImportKeyOpts{}), &hmacImportKeyOptsKeyImporter{})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.ECDSAPKIXPublicKeyImportOpts{}), &ecdsaPKIXPublicKeyImportOptsKeyImporter{})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.ECDSAPrivateKeyImportOpts{}), &ecdsaPrivateKeyImportOptsKeyImporter{})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.ECDSAGoPublicKeyImportOpts{}), &ecdsaGoPublicKeyImportOptsKeyImporter{})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.RSAGoPublicKeyImportOpts{}), &rsaGoPublicKeyImportOptsKeyImporter{})
	swbccsp.AddWrapper(reflect.TypeOf(&bccsp.X509PublicKeyImportOpts{}), &x509PublicKeyImportOptsKeyImporter{bccsp: swbccsp})

	return swbccsp, nil
}
