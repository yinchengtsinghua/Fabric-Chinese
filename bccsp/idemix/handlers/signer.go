
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

type Signer struct {
	SignatureScheme SignatureScheme
}

func (s *Signer) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) ([]byte, error) {
	userSecretKey, ok := k.(*userSecretKey)
	if !ok {
		return nil, errors.New("invalid key, expected *userSecretKey")
	}

	signerOpts, ok := opts.(*bccsp.IdemixSignerOpts)
	if !ok {
		return nil, errors.New("invalid options, expected *IdemixSignerOpts")
	}

//颁发者公钥
	if signerOpts.IssuerPK == nil {
		return nil, errors.New("invalid options, missing issuer public key")
	}
	ipk, ok := signerOpts.IssuerPK.(*issuerPublicKey)
	if !ok {
		return nil, errors.New("invalid issuer public key, expected *issuerPublicKey")
	}

//尼姆
	if signerOpts.Nym == nil {
		return nil, errors.New("invalid options, missing nym key")
	}
	nymSk, ok := signerOpts.Nym.(*nymSecretKey)
	if !ok {
		return nil, errors.New("invalid nym key, expected *nymSecretKey")
	}

	sigma, err := s.SignatureScheme.Sign(
		signerOpts.Credential,
		userSecretKey.sk,
		nymSk.pk, nymSk.sk,
		ipk.pk,
		signerOpts.Attributes,
		digest,
		signerOpts.RhIndex,
		signerOpts.CRI,
	)
	if err != nil {
		return nil, err
	}

	return sigma, nil
}

type Verifier struct {
	SignatureScheme SignatureScheme
}

func (v *Verifier) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (bool, error) {
	issuerPublicKey, ok := k.(*issuerPublicKey)
	if !ok {
		return false, errors.New("invalid key, expected *issuerPublicKey")
	}

	signerOpts, ok := opts.(*bccsp.IdemixSignerOpts)
	if !ok {
		return false, errors.New("invalid options, expected *IdemixSignerOpts")
	}

	rpk, ok := signerOpts.RevocationPublicKey.(*revocationPublicKey)
	if !ok {
		return false, errors.New("invalid options, expected *revocationPublicKey")
	}

	if len(signature) == 0 {
		return false, errors.New("invalid signature, it must not be empty")
	}

	err := v.SignatureScheme.Verify(
		issuerPublicKey.pk,
		signature,
		digest,
		signerOpts.Attributes,
		signerOpts.RhIndex,
		rpk.pubKey,
		signerOpts.Epoch,
	)
	if err != nil {
		return false, err
	}

	return true, nil
}
