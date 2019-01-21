
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

package idemix

import (
	"reflect"

	"github.com/hyperledger/fabric/bccsp/idemix/bridge"

	"github.com/hyperledger/fabric/bccsp/idemix/handlers"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/pkg/errors"
)

type csp struct {
	*sw.CSP
}

func New(keyStore bccsp.KeyStore) (*csp, error) {
	base, err := sw.New(keyStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed instantiating base bccsp")
	}

	csp := &csp{CSP: base}

//密钥生成器
	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixIssuerKeyGenOpts{}), &handlers.IssuerKeyGen{Issuer: &bridge.Issuer{NewRand: bridge.NewRandOrPanic}})
	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixUserSecretKeyGenOpts{}), &handlers.UserKeyGen{User: &bridge.User{NewRand: bridge.NewRandOrPanic}})
	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixRevocationKeyGenOpts{}), &handlers.RevocationKeyGen{Revocation: &bridge.Revocation{}})

//密钥导出程序
	base.AddWrapper(reflect.TypeOf(handlers.NewUserSecretKey(nil, false)), &handlers.NymKeyDerivation{
		User: &bridge.User{NewRand: bridge.NewRandOrPanic},
	})

//签署者
	base.AddWrapper(reflect.TypeOf(handlers.NewUserSecretKey(nil, false)), &userSecreKeySignerMultiplexer{
		signer:                  &handlers.Signer{SignatureScheme: &bridge.SignatureScheme{NewRand: bridge.NewRandOrPanic}},
		nymSigner:               &handlers.NymSigner{NymSignatureScheme: &bridge.NymSignatureScheme{NewRand: bridge.NewRandOrPanic}},
		credentialRequestSigner: &handlers.CredentialRequestSigner{CredRequest: &bridge.CredRequest{NewRand: bridge.NewRandOrPanic}},
	})
	base.AddWrapper(reflect.TypeOf(handlers.NewIssuerSecretKey(nil, false)), &handlers.CredentialSigner{
		Credential: &bridge.Credential{NewRand: bridge.NewRandOrPanic},
	})
	base.AddWrapper(reflect.TypeOf(handlers.NewRevocationSecretKey(nil, false)), &handlers.CriSigner{
		Revocation: &bridge.Revocation{},
	})

//验证者
	base.AddWrapper(reflect.TypeOf(handlers.NewIssuerPublicKey(nil)), &issuerPublicKeyVerifierMultiplexer{
		verifier:                  &handlers.Verifier{SignatureScheme: &bridge.SignatureScheme{NewRand: bridge.NewRandOrPanic}},
		credentialRequestVerifier: &handlers.CredentialRequestVerifier{CredRequest: &bridge.CredRequest{NewRand: bridge.NewRandOrPanic}},
	})
	base.AddWrapper(reflect.TypeOf(handlers.NewNymPublicKey(nil)), &handlers.NymVerifier{
		NymSignatureScheme: &bridge.NymSignatureScheme{NewRand: bridge.NewRandOrPanic},
	})
	base.AddWrapper(reflect.TypeOf(handlers.NewUserSecretKey(nil, false)), &handlers.CredentialVerifier{
		Credential: &bridge.Credential{NewRand: bridge.NewRandOrPanic},
	})
	base.AddWrapper(reflect.TypeOf(handlers.NewRevocationPublicKey(nil)), &handlers.CriVerifier{
		Revocation: &bridge.Revocation{},
	})

//进口商
	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixUserSecretKeyImportOpts{}), &handlers.UserKeyImporter{
		User: &bridge.User{},
	})
	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixIssuerPublicKeyImportOpts{}), &handlers.IssuerPublicKeyImporter{
		Issuer: &bridge.Issuer{},
	})
	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixNymPublicKeyImportOpts{}), &handlers.NymPublicKeyImporter{
		User: &bridge.User{},
	})
	base.AddWrapper(reflect.TypeOf(&bccsp.IdemixRevocationPublicKeyImportOpts{}), &handlers.RevocationPublicKeyImporter{})

	return csp, nil
}

//用K键签署符号摘要。
//opts参数应该适合使用的原语。
//
//请注意，当需要较大消息的哈希签名时，
//调用者负责散列较大的消息并传递
//散列（作为摘要）。
//请注意，这将覆盖sw impl的sign方法。以避免进行摘要检查。
func (csp *csp) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
//验证参数
	if k == nil {
		return nil, errors.New("Invalid Key. It must not be nil.")
	}
//不检查摘要

	keyType := reflect.TypeOf(k)
	signer, found := csp.Signers[keyType]
	if !found {
		return nil, errors.Errorf("Unsupported 'SignKey' provided [%s]", keyType)
	}

	signature, err = signer.Sign(k, digest, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed signing with opts [%v]", opts)
	}

	return
}

//验证根据密钥k和摘要验证签名
//请注意，这将覆盖sw impl的sign方法。以避免进行摘要检查。
func (csp *csp) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
//验证参数
	if k == nil {
		return false, errors.New("Invalid Key. It must not be nil.")
	}
	if len(signature) == 0 {
		return false, errors.New("Invalid signature. Cannot be empty.")
	}
//不检查摘要

	verifier, found := csp.Verifiers[reflect.TypeOf(k)]
	if !found {
		return false, errors.Errorf("Unsupported 'VerifyKey' provided [%v]", k)
	}

	valid, err = verifier.Verify(k, signature, digest, opts)
	if err != nil {
		return false, errors.Wrapf(err, "Failed verifing with opts [%v]", opts)
	}

	return
}

type userSecreKeySignerMultiplexer struct {
	signer                  *handlers.Signer
	nymSigner               *handlers.NymSigner
	credentialRequestSigner *handlers.CredentialRequestSigner
}

func (s *userSecreKeySignerMultiplexer) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
	switch opts.(type) {
	case *bccsp.IdemixSignerOpts:
		return s.signer.Sign(k, digest, opts)
	case *bccsp.IdemixNymSignerOpts:
		return s.nymSigner.Sign(k, digest, opts)
	case *bccsp.IdemixCredentialRequestSignerOpts:
		return s.credentialRequestSigner.Sign(k, digest, opts)
	default:
		return nil, errors.New("invalid opts, expected *bccsp.IdemixSignerOpt or *bccsp.IdemixNymSignerOpts or *bccsp.IdemixCredentialRequestSignerOpts")
	}
}

type issuerPublicKeyVerifierMultiplexer struct {
	verifier                  *handlers.Verifier
	credentialRequestVerifier *handlers.CredentialRequestVerifier
}

func (v *issuerPublicKeyVerifierMultiplexer) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
	switch opts.(type) {
	case *bccsp.IdemixSignerOpts:
		return v.verifier.Verify(k, signature, digest, opts)
	case *bccsp.IdemixCredentialRequestSignerOpts:
		return v.credentialRequestVerifier.Verify(k, signature, digest, opts)
	default:
		return false, errors.New("invalid opts, expected *bccsp.IdemixSignerOpts or *bccsp.IdemixCredentialRequestSignerOpts")
	}
}
