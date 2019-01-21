
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


package pkcs11

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"os"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

var (
	logger           = flogging.MustGetLogger("bccsp_p11")
	sessionCacheSize = 10
)

//New WithParams返回基于软件的BCCSP的新实例
//设置为通过的安全级别、哈希系列和密钥库。
func New(opts PKCS11Opts, keyStore bccsp.KeyStore) (bccsp.BCCSP, error) {
//初始化配置
	conf := &config{}
	err := conf.setSecurityLevel(opts.SecLevel, opts.HashFamily)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed initializing configuration")
	}

	swCSP, err := sw.NewWithParams(opts.SecLevel, opts.HashFamily, keyStore)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed initializing fallback SW BCCSP")
	}

//检查密钥库
	if keyStore == nil {
		return nil, errors.New("Invalid bccsp.KeyStore instance. It must be different from nil")
	}

	lib := opts.Library
	pin := opts.Pin
	label := opts.Label
	ctx, slot, session, err := loadLib(lib, pin, label)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed initializing PKCS11 library %s %s",
			lib, label)
	}

	sessions := make(chan pkcs11.SessionHandle, sessionCacheSize)
	csp := &impl{swCSP, conf, keyStore, ctx, sessions, slot, lib, opts.SoftVerify, opts.Immutable}
	csp.returnSession(*session)
	return csp, nil
}

type impl struct {
	bccsp.BCCSP

	conf *config
	ks   bccsp.KeyStore

	ctx      *pkcs11.Ctx
	sessions chan pkcs11.SessionHandle
	slot     uint

	lib        string
	softVerify bool
//不可变标志使对象不可变
	immutable bool
}

//keygen使用opts生成密钥。
func (csp *impl) KeyGen(opts bccsp.KeyGenOpts) (k bccsp.Key, err error) {
//验证参数
	if opts == nil {
		return nil, errors.New("Invalid Opts parameter. It must not be nil")
	}

//解析算法
	switch opts.(type) {
	case *bccsp.ECDSAKeyGenOpts:
		ski, pub, err := csp.generateECKey(csp.conf.ellipticCurve, opts.Ephemeral())
		if err != nil {
			return nil, errors.Wrapf(err, "Failed generating ECDSA key")
		}
		k = &ecdsaPrivateKey{ski, ecdsaPublicKey{ski, pub}}

	case *bccsp.ECDSAP256KeyGenOpts:
		ski, pub, err := csp.generateECKey(oidNamedCurveP256, opts.Ephemeral())
		if err != nil {
			return nil, errors.Wrapf(err, "Failed generating ECDSA P256 key")
		}

		k = &ecdsaPrivateKey{ski, ecdsaPublicKey{ski, pub}}

	case *bccsp.ECDSAP384KeyGenOpts:
		ski, pub, err := csp.generateECKey(oidNamedCurveP384, opts.Ephemeral())
		if err != nil {
			return nil, errors.Wrapf(err, "Failed generating ECDSA P384 key")
		}

		k = &ecdsaPrivateKey{ski, ecdsaPublicKey{ski, pub}}

	default:
		return csp.BCCSP.KeyGen(opts)
	}

	return k, nil
}

//keyimport使用opts从原始表示中导入密钥。
//opts参数应该适合使用的原语。
func (csp *impl) KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error) {
//验证参数
	if raw == nil {
		return nil, errors.New("Invalid raw. Cannot be nil")
	}

	if opts == nil {
		return nil, errors.New("Invalid Opts parameter. It must not be nil")
	}

	switch opts.(type) {

	case *bccsp.X509PublicKeyImportOpts:
		x509Cert, ok := raw.(*x509.Certificate)
		if !ok {
			return nil, errors.New("[X509PublicKeyImportOpts] Invalid raw material. Expected *x509.Certificate")
		}

		pk := x509Cert.PublicKey

		switch pk.(type) {
		case *ecdsa.PublicKey:
			return csp.KeyImport(pk, &bccsp.ECDSAGoPublicKeyImportOpts{Temporary: opts.Ephemeral()})
		case *rsa.PublicKey:
			return csp.KeyImport(pk, &bccsp.RSAGoPublicKeyImportOpts{Temporary: opts.Ephemeral()})
		default:
			return nil, errors.New("Certificate's public key type not recognized. Supported keys: [ECDSA, RSA]")
		}

	default:
		return csp.BCCSP.KeyImport(raw, opts)

	}
}

//GetKey返回此CSP关联的密钥
//主题键标识符ski。
func (csp *impl) GetKey(ski []byte) (bccsp.Key, error) {
	pubKey, isPriv, err := csp.getECKey(ski)
	if err == nil {
		if isPriv {
			return &ecdsaPrivateKey{ski, ecdsaPublicKey{ski, pubKey}}, nil
		}
		return &ecdsaPublicKey{ski, pubKey}, nil
	}
	return csp.BCCSP.GetKey(ski)
}

//用K键签署符号摘要。
//opts参数应该适合使用的原语。
//
//请注意，当需要较大消息的哈希签名时，
//调用者负责散列较大的消息并传递
//散列（作为摘要）。
func (csp *impl) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) ([]byte, error) {
//验证参数
	if k == nil {
		return nil, errors.New("Invalid Key. It must not be nil")
	}
	if len(digest) == 0 {
		return nil, errors.New("Invalid digest. Cannot be empty")
	}

//检查键类型
	switch k.(type) {
	case *ecdsaPrivateKey:
		return csp.signECDSA(*k.(*ecdsaPrivateKey), digest, opts)
	default:
		return csp.BCCSP.Sign(k, digest, opts)
	}
}

//验证根据密钥k和摘要验证签名
func (csp *impl) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (bool, error) {
//验证参数
	if k == nil {
		return false, errors.New("Invalid Key. It must not be nil")
	}
	if len(signature) == 0 {
		return false, errors.New("Invalid signature. Cannot be empty")
	}
	if len(digest) == 0 {
		return false, errors.New("Invalid digest. Cannot be empty")
	}

//检查键类型
	switch k.(type) {
	case *ecdsaPrivateKey:
		return csp.verifyECDSA(k.(*ecdsaPrivateKey).pub, signature, digest, opts)
	case *ecdsaPublicKey:
		return csp.verifyECDSA(*k.(*ecdsaPublicKey), signature, digest, opts)
	default:
		return csp.BCCSP.Verify(k, signature, digest, opts)
	}
}

//加密使用密钥K加密明文。
//opts参数应该适合使用的原语。
func (csp *impl) Encrypt(k bccsp.Key, plaintext []byte, opts bccsp.EncrypterOpts) ([]byte, error) {
//TODO:添加对加密的PKCS11支持，当结构开始需要它时
	return csp.BCCSP.Encrypt(k, plaintext, opts)
}

//解密使用密钥k解密密文。
//opts参数应该适合使用的原语。
func (csp *impl) Decrypt(k bccsp.Key, ciphertext []byte, opts bccsp.DecrypterOpts) ([]byte, error) {
	return csp.BCCSP.Decrypt(k, ciphertext, opts)
}

//findpkcs11lib仅用于测试
//这是一个方便的功能。对于自配置很有用，对于通常配置不为
//可获得的
func FindPKCS11Lib() (lib, pin, label string) {
//Fixme：在我们练习配置之前，在熟悉的地方寻找库
	lib = os.Getenv("PKCS11_LIB")
	if lib == "" {
		pin = "98765432"
		label = "ForFabric"
		possibilities := []string{
"/usr/lib/softhsm/libsofthsm2.so",                            //德比
"/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so",           //乌邦图
"/usr/lib/s390x-linux-gnu/softhsm/libsofthsm2.so",            //乌邦图
"/usr/lib/powerpc64le-linux-gnu/softhsm/libsofthsm2.so",      //功率
"/usr/local/Cellar/softhsm/2.1.0/lib/softhsm/libsofthsm2.so", //马科斯
		}
		for _, path := range possibilities {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				lib = path
				break
			}
		}
	} else {
		pin = os.Getenv("PKCS11_PIN")
		label = os.Getenv("PKCS11_LABEL")
	}
	return lib, pin, label
}
