
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

package ca_test

import (
	"crypto/ecdsa"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/common/tools/cryptogen/ca"
	"github.com/hyperledger/fabric/common/tools/cryptogen/csp"
	"github.com/stretchr/testify/assert"
)

const (
	testCAName             = "root0"
	testCA2Name            = "root1"
	testCA3Name            = "root2"
	testName               = "cert0"
	testName2              = "cert1"
	testName3              = "cert2"
	testIP                 = "172.16.10.31"
	testCountry            = "US"
	testProvince           = "California"
	testLocality           = "San Francisco"
	testOrganizationalUnit = "Hyperledger Fabric"
	testStreetAddress      = "testStreetAddress"
	testPostalCode         = "123456"
)

var testDir = filepath.Join(os.TempDir(), "ca-test")

func TestLoadCertificateECDSA(t *testing.T) {
	caDir := filepath.Join(testDir, "ca")
	certDir := filepath.Join(testDir, "certs")
//生成私钥
	priv, _, err := csp.GeneratePrivateKey(certDir)
	assert.NoError(t, err, "Failed to generate signed certificate")

//获取EC公钥
	ecPubKey, err := csp.GetECPublicKey(priv)
	assert.NoError(t, err, "Failed to generate signed certificate")
	assert.NotNil(t, ecPubKey, "Failed to generate signed certificate")

//创造我们的CA
	rootCA, err := ca.NewCA(caDir, testCA3Name, testCA3Name, testCountry, testProvince, testLocality, testOrganizationalUnit, testStreetAddress, testPostalCode)
	assert.NoError(t, err, "Error generating CA")

	cert, err := rootCA.SignCertificate(certDir, testName3, nil, nil, ecPubKey,
		x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageAny})
	assert.NoError(t, err, "Failed to generate signed certificate")
//keyUsage应为x509.keyUsageDigitalSignature x509.keyUsageKeyEncipherment
	assert.Equal(t, x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,
		cert.KeyUsage)
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageAny)

	loadedCert, err := ca.LoadCertificateECDSA(certDir)
	assert.NotNil(t, loadedCert, "Should load cert")
	assert.Equal(t, cert.SerialNumber, loadedCert.SerialNumber, "Should have same serial number")
	assert.Equal(t, cert.Subject.CommonName, loadedCert.Subject.CommonName, "Should have same CN")
	cleanup(testDir)
}

func TestNewCA(t *testing.T) {

	caDir := filepath.Join(testDir, "ca")
	rootCA, err := ca.NewCA(caDir, testCAName, testCAName, testCountry, testProvince, testLocality, testOrganizationalUnit, testStreetAddress, testPostalCode)
	assert.NoError(t, err, "Error generating CA")
	assert.NotNil(t, rootCA, "Failed to return CA")
	assert.NotNil(t, rootCA.Signer,
		"rootCA.Signer should not be empty")
	assert.IsType(t, &x509.Certificate{}, rootCA.SignCert,
		"rootCA.SignCert should be type x509.Certificate")

//检查以确保已存储根公钥
	pemFile := filepath.Join(caDir, testCAName+"-cert.pem")
	assert.Equal(t, true, checkForFile(pemFile),
		"Expected to find file "+pemFile)

	assert.NotEmpty(t, rootCA.SignCert.Subject.Country, "country cannot be empty.")
	assert.Equal(t, testCountry, rootCA.SignCert.Subject.Country[0], "Failed to match country")
	assert.NotEmpty(t, rootCA.SignCert.Subject.Province, "province cannot be empty.")
	assert.Equal(t, testProvince, rootCA.SignCert.Subject.Province[0], "Failed to match province")
	assert.NotEmpty(t, rootCA.SignCert.Subject.Locality, "locality cannot be empty.")
	assert.Equal(t, testLocality, rootCA.SignCert.Subject.Locality[0], "Failed to match locality")
	assert.NotEmpty(t, rootCA.SignCert.Subject.OrganizationalUnit, "organizationalUnit cannot be empty.")
	assert.Equal(t, testOrganizationalUnit, rootCA.SignCert.Subject.OrganizationalUnit[0], "Failed to match organizationalUnit")
	assert.NotEmpty(t, rootCA.SignCert.Subject.StreetAddress, "streetAddress cannot be empty.")
	assert.Equal(t, testStreetAddress, rootCA.SignCert.Subject.StreetAddress[0], "Failed to match streetAddress")
	assert.NotEmpty(t, rootCA.SignCert.Subject.PostalCode, "postalCode cannot be empty.")
	assert.Equal(t, testPostalCode, rootCA.SignCert.Subject.PostalCode[0], "Failed to match postalCode")

	cleanup(testDir)

}

func TestGenerateSignCertificate(t *testing.T) {

	caDir := filepath.Join(testDir, "ca")
	certDir := filepath.Join(testDir, "certs")
//生成私钥
	priv, _, err := csp.GeneratePrivateKey(certDir)
	assert.NoError(t, err, "Failed to generate signed certificate")

//获取EC公钥
	ecPubKey, err := csp.GetECPublicKey(priv)
	assert.NoError(t, err, "Failed to generate signed certificate")
	assert.NotNil(t, ecPubKey, "Failed to generate signed certificate")

//创造我们的CA
	rootCA, err := ca.NewCA(caDir, testCA2Name, testCA2Name, testCountry, testProvince, testLocality, testOrganizationalUnit, testStreetAddress, testPostalCode)
	assert.NoError(t, err, "Error generating CA")

	cert, err := rootCA.SignCertificate(certDir, testName, nil, nil, ecPubKey,
		x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageAny})
	assert.NoError(t, err, "Failed to generate signed certificate")
//keyUsage应为x509.keyUsageDigitalSignature x509.keyUsageKeyEncipherment
	assert.Equal(t, x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,
		cert.KeyUsage)
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageAny)

	cert, err = rootCA.SignCertificate(certDir, testName, nil, nil, ecPubKey,
		x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	assert.NoError(t, err, "Failed to generate signed certificate")
	assert.Equal(t, 0, len(cert.ExtKeyUsage))

//确保正确设置OU
	ous := []string{"TestOU", "PeerOU"}
	cert, err = rootCA.SignCertificate(certDir, testName, ous, nil, ecPubKey,
		x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	assert.Contains(t, cert.Subject.OrganizationalUnit, ous[0])
	assert.Contains(t, cert.Subject.OrganizationalUnit, ous[1])

//确保SAN设置正确
	sans := []string{testName2, testIP}
	cert, err = rootCA.SignCertificate(certDir, testName, nil, sans, ecPubKey,
		x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	assert.Contains(t, cert.DNSNames, testName2)
	assert.Contains(t, cert.IPAddresses, net.ParseIP(testIP).To4())

//检查以确保已签名的公钥已存储
	pemFile := filepath.Join(certDir, testName+"-cert.pem")
	assert.Equal(t, true, checkForFile(pemFile),
		"Expected to find file "+pemFile)

	_, err = rootCA.SignCertificate(certDir, "empty/CA", nil, nil, ecPubKey,
		x509.KeyUsageKeyEncipherment, []x509.ExtKeyUsage{x509.ExtKeyUsageAny})
	assert.Error(t, err, "Bad name should fail")

//使用空CA测试错误路径
	badCA := &ca.CA{
		Name:     "badCA",
		SignCert: &x509.Certificate{},
	}
	_, err = badCA.SignCertificate(certDir, testName, nil, nil, &ecdsa.PublicKey{},
		x509.KeyUsageKeyEncipherment, []x509.ExtKeyUsage{x509.ExtKeyUsageAny})
	assert.Error(t, err, "Empty CA should not be able to sign")
	cleanup(testDir)

}

func cleanup(dir string) {
	os.RemoveAll(dir)
}

func checkForFile(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}
