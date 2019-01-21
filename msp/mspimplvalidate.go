
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


package msp

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"reflect"
	"time"

	"github.com/pkg/errors"
)

func (msp *bccspmsp) validateIdentity(id *identity) error {
	validationChain, err := msp.getCertificationChainForBCCSPIdentity(id)
	if err != nil {
		return errors.WithMessage(err, "could not obtain certification chain")
	}

	err = msp.validateIdentityAgainstChain(id, validationChain)
	if err != nil {
		return errors.WithMessage(err, "could not validate identity against certification chain")
	}

	err = msp.internalValidateIdentityOusFunc(id)
	if err != nil {
		return errors.WithMessage(err, "could not validate identity's OUs")
	}

	return nil
}

func (msp *bccspmsp) validateCAIdentity(id *identity) error {
	if !id.cert.IsCA {
		return errors.New("Only CA identities can be validated")
	}

	validationChain, err := msp.getUniqueValidationChain(id.cert, msp.getValidityOptsForCert(id.cert))
	if err != nil {
		return errors.WithMessage(err, "could not obtain certification chain")
	}
	if len(validationChain) == 1 {
//validationchain[0]是根CA证书
		return nil
	}

	return msp.validateIdentityAgainstChain(id, validationChain)
}

func (msp *bccspmsp) validateTLSCAIdentity(cert *x509.Certificate, opts *x509.VerifyOptions) error {
	if !cert.IsCA {
		return errors.New("Only CA identities can be validated")
	}

	validationChain, err := msp.getUniqueValidationChain(cert, *opts)
	if err != nil {
		return errors.WithMessage(err, "could not obtain certification chain")
	}
	if len(validationChain) == 1 {
//validationchain[0]是根CA证书
		return nil
	}

	return msp.validateCertAgainstChain(cert, validationChain)
}

func (msp *bccspmsp) validateIdentityAgainstChain(id *identity, validationChain []*x509.Certificate) error {
	return msp.validateCertAgainstChain(id.cert, validationChain)
}

func (msp *bccspmsp) validateCertAgainstChain(cert *x509.Certificate, validationChain []*x509.Certificate) error {
//在这里，我们知道这个身份是有效的；现在我们必须检查它是否被撤销了。

//识别签署此证书的CA的SKI
	SKI, err := getSubjectKeyIdentifierFromCert(validationChain[1])
	if err != nil {
		return errors.WithMessage(err, "could not obtain Subject Key Identifier for signer cert")
	}

//检查我们有没有一个CRL有这个证书
//SKI作为其授权密钥标识符
	for _, crl := range msp.CRL {
		aki, err := getAuthorityKeyIdentifierFromCrl(crl)
		if err != nil {
			return errors.WithMessage(err, "could not obtain Authority Key Identifier for crl")
		}

//检查签名我们的证书的ski是否与任何crl的aki匹配
		if bytes.Equal(aki, SKI) {
//我们有一个CRL，检查序列号是否被撤销。
			for _, rc := range crl.TBSCertList.RevokedCertificates {
				if rc.SerialNumber.Cmp(cert.SerialNumber) == 0 {
//我们找到了一个crl，其aki与滑雪板的
//签名的CA（根或中间）
//正在验证的证书。作为一个
//预防措施，我们证实上述CA也是
//此CRL的签名者。
					err = validationChain[1].CheckCRLSignature(crl)
					if err != nil {
//签署证书的CA证书
//正在验证的未签署
//候选CRL-跳过
						mspLogger.Warningf("Invalid signature over the identified CRL, error %+v", err)
						continue
					}

//crl还包括一个撤销时间，以便
//CA可以说“此证书将从
//从这个时候开始，“然而，我们假设
//撤销立即生效
//MSP配置已提交并使用，因此我们不会
//利用这个领域
					return errors.New("The certificate has been revoked")
				}
			}
		}
	}

	return nil
}

func (msp *bccspmsp) validateIdentityOUsV1(id *identity) error {
//检查身份的OU是否与此MSP识别的那些一致，
//也就是说交叉口不是空的。
	if len(msp.ouIdentifiers) > 0 {
		found := false

		for _, OU := range id.GetOrganizationalUnits() {
			certificationIDs, exists := msp.ouIdentifiers[OU.OrganizationalUnitIdentifier]

			if exists {
				for _, certificationID := range certificationIDs {
					if bytes.Equal(certificationID, OU.CertifiersIdentifier) {
						found = true
						break
					}
				}
			}
		}

		if !found {
			if len(id.GetOrganizationalUnits()) == 0 {
				return errors.New("the identity certificate does not contain an Organizational Unit (OU)")
			}
			return errors.Errorf("none of the identity's organizational units [%v] are in MSP %s", id.GetOrganizationalUnits(), msp.name)
		}
	}

	return nil
}

func (msp *bccspmsp) validateIdentityOUsV11(id *identity) error {
//按照v1执行相同的检查
	err := msp.validateIdentityOUsV1(id)
	if err != nil {
		return err
	}

//执行v1_1附加检查：
//
//--检查OU执行情况
	if !msp.ouEnforcement {
//不需要强制执行
		return nil
	}

//确保身份只有一个特殊的OU
//用于区分客户、对等方和订购方。
	counter := 0
	for _, OU := range id.GetOrganizationalUnits() {
//组织身份识别器是一个特殊的组织身份识别器吗？
		var nodeOU *OUIdentifier
		switch OU.OrganizationalUnitIdentifier {
		case msp.clientOU.OrganizationalUnitIdentifier:
			nodeOU = msp.clientOU
		case msp.peerOU.OrganizationalUnitIdentifier:
			nodeOU = msp.peerOU
		default:
			continue
		}

//对。然后，强制执行证明者标识符是否指定了该标识符。
//没有指定，这意味着任何认证路径都可以。
		if len(nodeOU.CertifiersIdentifier) != 0 && !bytes.Equal(nodeOU.CertifiersIdentifier, OU.CertifiersIdentifier) {
			return errors.Errorf("certifiersIdentifier does not match: [%v], MSP: [%s]", id.GetOrganizationalUnits(), msp.name)
		}
		counter++
		if counter > 1 {
			break
		}
	}
	if counter != 1 {
		return errors.Errorf("the identity must be a client, a peer or an orderer identity to be valid, not a combination of them. OUs: [%v], MSP: [%s]", id.GetOrganizationalUnits(), msp.name)
	}

	return nil
}

func (msp *bccspmsp) getValidityOptsForCert(cert *x509.Certificate) x509.VerifyOptions {
//首先复制opts以覆盖当前时间字段
//为了使证书通过过期测试
//独立于真实的本地电流时间。
//这是FAB-3678的临时解决方案

	var tempOpts x509.VerifyOptions
	tempOpts.Roots = msp.opts.Roots
	tempOpts.DNSName = msp.opts.DNSName
	tempOpts.Intermediates = msp.opts.Intermediates
	tempOpts.KeyUsages = msp.opts.KeyUsages
	tempOpts.CurrentTime = cert.NotBefore.Add(time.Second)

	return tempOpts
}

/*
   这是授权密钥标识符的ASN.1编组的定义
   来自https://www.ietf.org/rfc/rfc5280.txt

   权限键标识符：：=序列
      keyIdentifier[0]keyIdentifier可选，
      authoritycertissuer[1]常规名称可选，
      授权人序列号[2]证书序列号可选

   键标识符：：=octet字符串

   证书序列号：：=整数

**/


type authorityKeyIdentifier struct {
	KeyIdentifier             []byte  `asn1:"optional,tag:0"`
	AuthorityCertIssuer       []byte  `asn1:"optional,tag:1"`
	AuthorityCertSerialNumber big.Int `asn1:"optional,tag:2"`
}

//GetAuthorityKeyIdentifierFromCrl返回颁发机构密钥标识符
//用于提供的CRL。授权密钥标识符可用于标识
//与用于签署CRL的私钥相对应的公钥。
func getAuthorityKeyIdentifierFromCrl(crl *pkix.CertificateList) ([]byte, error) {
	aki := authorityKeyIdentifier{}

	for _, ext := range crl.TBSCertList.Extensions {
//授权密钥标识符由以下ASN.1标记标识
//授权密钥标识符（2 5 29 35）（请参阅https://tools.ietf.org/html/rfc3280.html）
		if reflect.DeepEqual(ext.Id, asn1.ObjectIdentifier{2, 5, 29, 35}) {
			_, err := asn1.Unmarshal(ext.Value, &aki)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal AKI")
			}

			return aki.KeyIdentifier, nil
		}
	}

	return nil, errors.New("authorityKeyIdentifier not found in certificate")
}

//GetSubjectKeyIdentifierFromCert返回所提供证书的主题密钥标识符
//使用者密钥标识符是此证书的公钥的标识符
func getSubjectKeyIdentifierFromCert(cert *x509.Certificate) ([]byte, error) {
	var SKI []byte

	for _, ext := range cert.Extensions {
//主题密钥标识符由以下ASN.1标记标识
//SubjectKeyIdentifier（2 5 29 14）（请参阅https://tools.ietf.org/html/rfc3280.html）
		if reflect.DeepEqual(ext.Id, asn1.ObjectIdentifier{2, 5, 29, 14}) {
			_, err := asn1.Unmarshal(ext.Value, &SKI)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal Subject Key Identifier")
			}

			return SKI, nil
		}
	}

	return nil, errors.New("subjectKeyIdentifier not found in certificate")
}
