
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
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	m "github.com/hyperledger/fabric/protos/msp"
	errors "github.com/pkg/errors"
)

func (msp *bccspmsp) getCertifiersIdentifier(certRaw []byte) ([]byte, error) {
//1。检查证书是否已在msp.rootcerts或msp.intermediatecerts中注册。
	cert, err := msp.getCertFromPem(certRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed getting certificate for [%v]: [%s]", certRaw, err)
	}

//2。对其进行消毒以确保相似比较
	cert, err = msp.sanitizeCert(cert)
	if err != nil {
		return nil, fmt.Errorf("sanitizeCert failed %s", err)
	}

	found := false
	root := false
//在根证书中搜索
	for _, v := range msp.rootCerts {
		if v.(*identity).cert.Equal(cert) {
			found = true
			root = true
			break
		}
	}
	if !found {
//在根中间证书中搜索
		for _, v := range msp.intermediateCerts {
			if v.(*identity).cert.Equal(cert) {
				found = true
				break
			}
		}
	}
	if !found {
//证书无效，拒绝配置
		return nil, fmt.Errorf("Failed adding OU. Certificate [%v] not in root or intermediate certs.", cert)
	}

//三。获取它的证书路径
	var certifiersIdentifier []byte
	var chain []*x509.Certificate
	if root {
		chain = []*x509.Certificate{cert}
	} else {
		chain, err = msp.getValidationChain(cert, true)
		if err != nil {
			return nil, fmt.Errorf("Failed computing validation chain for [%v]. [%s]", cert, err)
		}
	}

//4。计算证书路径的哈希值
	certifiersIdentifier, err = msp.getCertificationChainIdentifierFromChain(chain)
	if err != nil {
		return nil, fmt.Errorf("Failed computing Certifiers Identifier for [%v]. [%s]", certRaw, err)
	}

	return certifiersIdentifier, nil

}

func (msp *bccspmsp) setupCrypto(conf *m.FabricMSPConfig) error {
	msp.cryptoConfig = conf.CryptoConfig
	if msp.cryptoConfig == nil {
//移到默认值
		msp.cryptoConfig = &m.FabricCryptoConfig{
			SignatureHashFamily:            bccsp.SHA2,
			IdentityIdentifierHashFunction: bccsp.SHA256,
		}
		mspLogger.Debugf("CryptoConfig was nil. Move to defaults.")
	}
	if msp.cryptoConfig.SignatureHashFamily == "" {
		msp.cryptoConfig.SignatureHashFamily = bccsp.SHA2
		mspLogger.Debugf("CryptoConfig.SignatureHashFamily was nil. Move to defaults.")
	}
	if msp.cryptoConfig.IdentityIdentifierHashFunction == "" {
		msp.cryptoConfig.IdentityIdentifierHashFunction = bccsp.SHA256
		mspLogger.Debugf("CryptoConfig.IdentityIdentifierHashFunction was nil. Move to defaults.")
	}

	return nil
}

func (msp *bccspmsp) setupCAs(conf *m.FabricMSPConfig) error {
//制作并填写一套CA证书-我们希望它们在那里
	if len(conf.RootCerts) == 0 {
		return errors.New("expected at least one CA certificate")
	}

//使用根和中间文件预创建验证选项。
//这需要使证书卫生起作用。
//回想一下，消毒也适用于根CA和中间物。
//CA证书。卫生处理完成后，门诊部
//将使用已清理的证书重新创建。
	msp.opts = &x509.VerifyOptions{Roots: x509.NewCertPool(), Intermediates: x509.NewCertPool()}
	for _, v := range conf.RootCerts {
		cert, err := msp.getCertFromPem(v)
		if err != nil {
			return err
		}
		msp.opts.Roots.AddCert(cert)
	}
	for _, v := range conf.IntermediateCerts {
		cert, err := msp.getCertFromPem(v)
		if err != nil {
			return err
		}
		msp.opts.Intermediates.AddCert(cert)
	}

//加载根和中间CA标识
//回想一下，当创建标识时，其证书会被清理
	msp.rootCerts = make([]Identity, len(conf.RootCerts))
	for i, trustedCert := range conf.RootCerts {
		id, _, err := msp.getIdentityFromConf(trustedCert)
		if err != nil {
			return err
		}

		msp.rootCerts[i] = id
	}

//制作并填写一套中间证书（如有）
	msp.intermediateCerts = make([]Identity, len(conf.IntermediateCerts))
	for i, trustedCert := range conf.IntermediateCerts {
		id, _, err := msp.getIdentityFromConf(trustedCert)
		if err != nil {
			return err
		}

		msp.intermediateCerts[i] = id
	}

//根CA和中间CA证书已清理，可以重新导入
	msp.opts = &x509.VerifyOptions{Roots: x509.NewCertPool(), Intermediates: x509.NewCertPool()}
	for _, id := range msp.rootCerts {
		msp.opts.Roots.AddCert(id.(*identity).cert)
	}
	for _, id := range msp.intermediateCerts {
		msp.opts.Intermediates.AddCert(id.(*identity).cert)
	}

	return nil
}

func (msp *bccspmsp) setupAdmins(conf *m.FabricMSPConfig) error {
//制作并填写管理证书集（如果有）
	msp.admins = make([]Identity, len(conf.Admins))
	for i, admCert := range conf.Admins {
		id, _, err := msp.getIdentityFromConf(admCert)
		if err != nil {
			return err
		}

		msp.admins[i] = id
	}

	return nil
}

func (msp *bccspmsp) setupCRLs(conf *m.FabricMSPConfig) error {
//设置CRL（如果存在）
	msp.CRL = make([]*pkix.CertificateList, len(conf.RevocationList))
	for i, crlbytes := range conf.RevocationList {
		crl, err := x509.ParseCRL(crlbytes)
		if err != nil {
			return errors.Wrap(err, "could not parse RevocationList")
		}

//TODO:预验证CRL上的签名并创建映射
//CA认证到各自的CRL，以便以后
//验证我们已经可以在给定的
//待验证证书链

		msp.CRL[i] = crl
	}

	return nil
}

func (msp *bccspmsp) finalizeSetupCAs() error {
//确保我们的CA格式正确并且有效
	for _, id := range append(append([]Identity{}, msp.rootCerts...), msp.intermediateCerts...) {
		if !id.(*identity).cert.IsCA {
			return errors.Errorf("CA Certificate did not have the CA attribute, (SN: %x)", id.(*identity).cert.SerialNumber)
		}
		if _, err := getSubjectKeyIdentifierFromCert(id.(*identity).cert); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("CA Certificate problem with Subject Key Identifier extension, (SN: %x)", id.(*identity).cert.SerialNumber))
		}

		if err := msp.validateCAIdentity(id.(*identity)); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("CA Certificate is not valid, (SN: %s)", id.(*identity).cert.SerialNumber))
		}
	}

//填充certificationtreeinternalNodesMap以标记
//认证树
	msp.certificationTreeInternalNodesMap = make(map[string]bool)
	for _, id := range append([]Identity{}, msp.intermediateCerts...) {
		chain, err := msp.getUniqueValidationChain(id.(*identity).cert, msp.getValidityOptsForCert(id.(*identity).cert))
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("failed getting validation chain, (SN: %s)", id.(*identity).cert.SerialNumber))
		}

//调用链[0]是id.（*identity）.id，因此它不算作父级
		for i := 1; i < len(chain); i++ {
			msp.certificationTreeInternalNodesMap[string(chain[i].Raw)] = true
		}
	}

	return nil
}

func (msp *bccspmsp) setupNodeOUs(config *m.FabricMSPConfig) error {
	if config.FabricNodeOus != nil {

		msp.ouEnforcement = config.FabricNodeOus.Enable

//克里门特
		msp.clientOU = &OUIdentifier{OrganizationalUnitIdentifier: config.FabricNodeOus.ClientOuIdentifier.OrganizationalUnitIdentifier}
		if len(config.FabricNodeOus.ClientOuIdentifier.Certificate) != 0 {
			certifiersIdentifier, err := msp.getCertifiersIdentifier(config.FabricNodeOus.ClientOuIdentifier.Certificate)
			if err != nil {
				return err
			}
			msp.clientOU.CertifiersIdentifier = certifiersIdentifier
		}

//佩罗
		msp.peerOU = &OUIdentifier{OrganizationalUnitIdentifier: config.FabricNodeOus.PeerOuIdentifier.OrganizationalUnitIdentifier}
		if len(config.FabricNodeOus.PeerOuIdentifier.Certificate) != 0 {
			certifiersIdentifier, err := msp.getCertifiersIdentifier(config.FabricNodeOus.PeerOuIdentifier.Certificate)
			if err != nil {
				return err
			}
			msp.peerOU.CertifiersIdentifier = certifiersIdentifier
		}

	} else {
		msp.ouEnforcement = false
	}

	return nil
}

func (msp *bccspmsp) setupSigningIdentity(conf *m.FabricMSPConfig) error {
	if conf.SigningIdentity != nil {
		sid, err := msp.getSigningIdentityFromConf(conf.SigningIdentity)
		if err != nil {
			return err
		}

		expirationTime := sid.ExpiresAt()
		now := time.Now()
		if expirationTime.After(now) {
			mspLogger.Debug("Signing identity expires at", expirationTime)
		} else if expirationTime.IsZero() {
			mspLogger.Debug("Signing identity has no known expiration time")
		} else {
			return errors.Errorf("signing identity expired %v ago", now.Sub(expirationTime))
		}

		msp.signer = sid
	}

	return nil
}

func (msp *bccspmsp) setupOUs(conf *m.FabricMSPConfig) error {
	msp.ouIdentifiers = make(map[string][][]byte)
	for _, ou := range conf.OrganizationalUnitIdentifiers {

		certifiersIdentifier, err := msp.getCertifiersIdentifier(ou.Certificate)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("failed getting certificate for [%v]", ou))
		}

//检查重复项
		found := false
		for _, id := range msp.ouIdentifiers[ou.OrganizationalUnitIdentifier] {
			if bytes.Equal(id, certifiersIdentifier) {
				mspLogger.Warningf("Duplicate found in ou identifiers [%s, %v]", ou.OrganizationalUnitIdentifier, id)
				found = true
				break
			}
		}

		if !found {
//找不到重复项，请添加
			msp.ouIdentifiers[ou.OrganizationalUnitIdentifier] = append(
				msp.ouIdentifiers[ou.OrganizationalUnitIdentifier],
				certifiersIdentifier,
			)
		}
	}

	return nil
}

func (msp *bccspmsp) setupTLSCAs(conf *m.FabricMSPConfig) error {

	opts := &x509.VerifyOptions{Roots: x509.NewCertPool(), Intermediates: x509.NewCertPool()}

//加载TLS根和中间CA标识
	msp.tlsRootCerts = make([][]byte, len(conf.TlsRootCerts))
	rootCerts := make([]*x509.Certificate, len(conf.TlsRootCerts))
	for i, trustedCert := range conf.TlsRootCerts {
		cert, err := msp.getCertFromPem(trustedCert)
		if err != nil {
			return err
		}

		rootCerts[i] = cert
		msp.tlsRootCerts[i] = trustedCert
		opts.Roots.AddCert(cert)
	}

//制作并填写一套中间证书（如有）
	msp.tlsIntermediateCerts = make([][]byte, len(conf.TlsIntermediateCerts))
	intermediateCerts := make([]*x509.Certificate, len(conf.TlsIntermediateCerts))
	for i, trustedCert := range conf.TlsIntermediateCerts {
		cert, err := msp.getCertFromPem(trustedCert)
		if err != nil {
			return err
		}

		intermediateCerts[i] = cert
		msp.tlsIntermediateCerts[i] = trustedCert
		opts.Intermediates.AddCert(cert)
	}

//确保我们的CA格式正确并且有效
	for _, cert := range append(append([]*x509.Certificate{}, rootCerts...), intermediateCerts...) {
		if cert == nil {
			continue
		}

		if !cert.IsCA {
			return errors.Errorf("CA Certificate did not have the CA attribute, (SN: %x)", cert.SerialNumber)
		}
		if _, err := getSubjectKeyIdentifierFromCert(cert); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("CA Certificate problem with Subject Key Identifier extension, (SN: %x)", cert.SerialNumber))
		}

		if err := msp.validateTLSCAIdentity(cert, opts); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("CA Certificate is not valid, (SN: %s)", cert.SerialNumber))
		}
	}

	return nil
}

func (msp *bccspmsp) setupV1(conf1 *m.FabricMSPConfig) error {
	err := msp.preSetupV1(conf1)
	if err != nil {
		return err
	}

	err = msp.postSetupV1(conf1)
	if err != nil {
		return err
	}

	return nil
}

func (msp *bccspmsp) preSetupV1(conf *m.FabricMSPConfig) error {
//设置加密配置
	if err := msp.setupCrypto(conf); err != nil {
		return err
	}

//设置CAS
	if err := msp.setupCAs(conf); err != nil {
		return err
	}

//设置管理员
	if err := msp.setupAdmins(conf); err != nil {
		return err
	}

//设置CRL
	if err := msp.setupCRLs(conf); err != nil {
		return err
	}

//完成CA的设置
	if err := msp.finalizeSetupCAs(); err != nil {
		return err
	}

//设置签名者（如果存在）
	if err := msp.setupSigningIdentity(conf); err != nil {
		return err
	}

//安装TLS CAs
	if err := msp.setupTLSCAs(conf); err != nil {
		return err
	}

//设置好
	if err := msp.setupOUs(conf); err != nil {
		return err
	}

	return nil
}

func (msp *bccspmsp) postSetupV1(conf *m.FabricMSPConfig) error {
//确保管理员也是有效的成员
//这样，当我们验证管理MSP主体时
//我们可以简单地检查证书是否完全匹配
	for i, admin := range msp.admins {
		err := admin.Validate()
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("admin %d is invalid", i))
		}
	}

	return nil
}

func (msp *bccspmsp) setupV11(conf *m.FabricMSPConfig) error {
	err := msp.preSetupV1(conf)
	if err != nil {
		return err
	}

//设置不良
	if err := msp.setupNodeOUs(conf); err != nil {
		return err
	}

	err = msp.postSetupV11(conf)
	if err != nil {
		return err
	}

	return nil
}

func (msp *bccspmsp) postSetupV11(conf *m.FabricMSPConfig) error {
//检查OU执行情况
	if !msp.ouEnforcement {
//不需要强制执行。根据v1设置呼叫岗
		return msp.postSetupV1(conf)
	}

//检查管理员是否为客户端
	principalBytes, err := proto.Marshal(&m.MSPRole{Role: m.MSPRole_CLIENT, MspIdentifier: msp.name})
	if err != nil {
		return errors.Wrapf(err, "failed creating MSPRole_CLIENT")
	}
	principal := &m.MSPPrincipal{
		PrincipalClassification: m.MSPPrincipal_ROLE,
		Principal:               principalBytes}
	for i, admin := range msp.admins {
		err = admin.SatisfiesPrincipal(principal)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("admin %d is invalid", i))
		}
	}

	return nil
}
