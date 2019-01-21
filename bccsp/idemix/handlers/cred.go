
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

package handlers

import (
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

//CredentialRequestSigner生成凭据请求
type CredentialRequestSigner struct {
//CredRequest实现基础加密算法
	CredRequest CredRequest
}

func (c *CredentialRequestSigner) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) ([]byte, error) {
	userSecretKey, ok := k.(*userSecretKey)
	if !ok {
		return nil, errors.New("invalid key, expected *userSecretKey")
	}
	credentialRequestSignerOpts, ok := opts.(*bccsp.IdemixCredentialRequestSignerOpts)
	if !ok {
		return nil, errors.New("invalid options, expected *IdemixCredentialRequestSignerOpts")
	}
	if credentialRequestSignerOpts.IssuerPK == nil {
		return nil, errors.New("invalid options, missing issuer public key")
	}
	issuerPK, ok := credentialRequestSignerOpts.IssuerPK.(*issuerPublicKey)
	if !ok {
		return nil, errors.New("invalid options, expected IssuerPK as *issuerPublicKey")
	}

	return c.CredRequest.Sign(userSecretKey.sk, issuerPK.pk, credentialRequestSignerOpts.IssuerNonce)
}

//CredentialRequestVerifier验证凭据请求
type CredentialRequestVerifier struct {
//CredRequest实现基础加密算法
	CredRequest CredRequest
}

func (c *CredentialRequestVerifier) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (bool, error) {
	issuerPublicKey, ok := k.(*issuerPublicKey)
	if !ok {
		return false, errors.New("invalid key, expected *issuerPublicKey")
	}
	credentialRequestSignerOpts, ok := opts.(*bccsp.IdemixCredentialRequestSignerOpts)
	if !ok {
		return false, errors.New("invalid options, expected *IdemixCredentialRequestSignerOpts")
	}

	err := c.CredRequest.Verify(signature, issuerPublicKey.pk, credentialRequestSignerOpts.IssuerNonce)
	if err != nil {
		return false, err
	}

	return true, nil
}

type CredentialSigner struct {
	Credential Credential
}

func (s *CredentialSigner) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
	issuerSecretKey, ok := k.(*issuerSecretKey)
	if !ok {
		return nil, errors.New("invalid key, expected *issuerSecretKey")
	}
	credOpts, ok := opts.(*bccsp.IdemixCredentialSignerOpts)
	if !ok {
		return nil, errors.New("invalid options, expected *IdemixCredentialSignerOpts")
	}

	signature, err = s.Credential.Sign(issuerSecretKey.sk, digest, credOpts.Attributes)
	if err != nil {
		return nil, err
	}

	return
}

type CredentialVerifier struct {
	Credential Credential
}

func (v *CredentialVerifier) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
	userSecretKey, ok := k.(*userSecretKey)
	if !ok {
		return false, errors.New("invalid key, expected *userSecretKey")
	}
	credOpts, ok := opts.(*bccsp.IdemixCredentialSignerOpts)
	if !ok {
		return false, errors.New("invalid options, expected *IdemixCredentialSignerOpts")
	}
	if credOpts.IssuerPK == nil {
		return false, errors.New("invalid options, missing issuer public key")
	}
	ipk, ok := credOpts.IssuerPK.(*issuerPublicKey)
	if !ok {
		return false, errors.New("invalid issuer public key, expected *issuerPublicKey")
	}
	if len(signature) == 0 {
		return false, errors.New("invalid signature, it must not be empty")
	}

	err = v.Credential.Verify(userSecretKey.sk, ipk.pk, signature, credOpts.Attributes)
	if err != nil {
		return false, err
	}

	return true, nil
}
