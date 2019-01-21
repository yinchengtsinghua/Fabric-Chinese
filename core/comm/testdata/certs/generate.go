
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


//+构建忽略

//go:generate-command gencerts go运行$gopath/src/github.com/hyperledger/fabric/core/comm/testdata/certs/generate.go
//go:生成gencerts-orgs 2-子orgs 2-服务器2-客户端2

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

//命令行标志
var (
	numOrgs        = flag.Int("orgs", 2, "number of unique organizations")
	numChildOrgs   = flag.Int("child-orgs", 2, "number of intermediaries per organization")
	numClientCerts = flag.Int("clients", 1, "number of client certificates per organization")
	numServerCerts = flag.Int("servers", 1, "number of server certificates per organization")
)

//X509主题的默认模板
func subjectTemplate() pkix.Name {
	return pkix.Name{
		Country:  []string{"US"},
		Locality: []string{"San Francisco"},
		Province: []string{"California"},
	}
}

//X509证书的默认模板
func x509Template() (x509.Certificate, error) {

//生成序列号
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return x509.Certificate{}, err
	}

	now := time.Now()
//要使用的基本模板
	x509 := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             now,
NotAfter:              now.Add(3650 * 24 * time.Hour), //十年
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	return x509, nil

}

//generate an EC private key (P256 curve)
func genKeyECDSA(name string) (*ecdsa.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
//将密钥写入文件
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	keyFile, err := os.OpenFile(name+"-key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	keyFile.Close()
	return priv, nil
}

//使用ECDSA生成已签名的X509证书
func genCertificateECDSA(name string, template, parent *x509.Certificate, pub *ecdsa.PublicKey,
	priv *ecdsa.PrivateKey) (*x509.Certificate, error) {

//创建X509公共证书
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		return nil, err
	}

//将证书写入文件
	certFile, err := os.Create(name + "-cert.pem")
	if err != nil {
		return nil, err
	}
//PEM编码证书
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certFile.Close()

	x509Cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}
	return x509Cert, nil
}

//生成适用于TLS服务器的EC证书
func genServerCertificateECDSA(name string, signKey *ecdsa.PrivateKey, signCert *x509.Certificate) error {
	fmt.Println(name)
	key, err := genKeyECDSA(name)
	template, err := x509Template()

	if err != nil {
		return err
	}

	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth,
		x509.ExtKeyUsageClientAuth}

//为主题设置组织
	subject := subjectTemplate()
	subject.Organization = []string{name}
	subject.CommonName = "localhost"

	template.Subject = subject
	template.DNSNames = []string{"localhost"}
	template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}

	_, err = genCertificateECDSA(name, &template, signCert, &key.PublicKey, signKey)

	if err != nil {
		return err
	}

	return nil
}

//生成适用于TLS服务器的EC证书
func genClientCertificateECDSA(name string, signKey *ecdsa.PrivateKey, signCert *x509.Certificate) error {
	fmt.Println(name)
	key, err := genKeyECDSA(name)
	template, err := x509Template()

	if err != nil {
		return err
	}

	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

//为主题设置组织
	subject := subjectTemplate()
	subject.Organization = []string{name}
	subject.CommonName = name

	template.Subject = subject

	_, err = genCertificateECDSA(name, &template, signCert, &key.PublicKey, signKey)

	if err != nil {
		return err
	}

	return nil
}

//生成EC证书签名（CA）密钥对并输出为
//PEM编码文件
func genCertificateAuthorityECDSA(name string) (*ecdsa.PrivateKey, *x509.Certificate, error) {

	key, err := genKeyECDSA(name)
	template, err := x509Template()

	if err != nil {
		return nil, nil, err
	}

//这是一个CA
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}

//为主题设置组织
	subject := subjectTemplate()
	subject.Organization = []string{name}
	subject.CommonName = name

	template.Subject = subject
	template.SubjectKeyId = []byte{1, 2, 3, 4}

	x509Cert, err := genCertificateECDSA(name, &template, &template, &key.PublicKey, key)

	if err != nil {
		return nil, nil, err
	}
	return key, x509Cert, nil
}

//生成适用于TLS服务器的EC证书
func genIntermediateCertificateAuthorityECDSA(name string, signKey *ecdsa.PrivateKey,
	signCert *x509.Certificate) (*ecdsa.PrivateKey, *x509.Certificate, error) {

	fmt.Println(name)
	key, err := genKeyECDSA(name)
	template, err := x509Template()

	if err != nil {
		return nil, nil, err
	}

//这是一个CA
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}

//为主题设置组织
	subject := subjectTemplate()
	subject.Organization = []string{name}
	subject.CommonName = name

	template.Subject = subject
	template.SubjectKeyId = []byte{1, 2, 3, 4}

	x509Cert, err := genCertificateECDSA(name, &template, signCert, &key.PublicKey, signKey)

	if err != nil {
		return nil, nil, err
	}
	return key, x509Cert, nil
}

func main() {

//分析命令行标志
	flag.Parse()

	fmt.Printf("Generating %d organizations each with %d server(s) and %d client(s)\n",
		*numOrgs, *numServerCerts, *numClientCerts)

	baseOrgName := "Org"
//生成组织/CA
	for i := 1; i <= *numOrgs; i++ {
		signKey, signCert, err := genCertificateAuthorityECDSA(fmt.Sprintf(baseOrgName+"%d", i))
		if err != nil {
			fmt.Printf("error generating CA %s%d : %s\n", baseOrgName, i, err.Error())
		}
//为组织生成服务器证书
		for j := 1; j <= *numServerCerts; j++ {
			err := genServerCertificateECDSA(fmt.Sprintf(baseOrgName+"%d-server%d", i, j), signKey, signCert)
			if err != nil {
				fmt.Printf("error generating server certificate for %s%d-server%d : %s\n",
					baseOrgName, i, j, err.Error())
			}
		}
//为组织生成客户端证书
		for k := 1; k <= *numClientCerts; k++ {
			err := genClientCertificateECDSA(fmt.Sprintf(baseOrgName+"%d-client%d", i, k), signKey, signCert)
			if err != nil {
				fmt.Printf("error generating client certificate for %s%d-client%d : %s\n",
					baseOrgName, i, k, err.Error())
			}
		}
//生成子组织（中介机构）
		for m := 1; m <= *numChildOrgs; m++ {
			childSignKey, childSignCert, err := genIntermediateCertificateAuthorityECDSA(
				fmt.Sprintf(baseOrgName+"%d-child%d", i, m), signKey, signCert)
			if err != nil {
				fmt.Printf("error generating CA %s%d-child%d : %s\n",
					baseOrgName, i, m, err.Error())
			}
//为子组织生成服务器证书
			for n := 1; n <= *numServerCerts; n++ {
				err := genServerCertificateECDSA(fmt.Sprintf(baseOrgName+"%d-child%d-server%d", i, m, n),
					childSignKey, childSignCert)
				if err != nil {
					fmt.Printf("error generating server certificate for %s%d-child%d-server%d : %s\n",
						baseOrgName, i, m, n, err.Error())
				}
			}
//为子组织生成客户端证书
			for p := 1; p <= *numClientCerts; p++ {
				err := genClientCertificateECDSA(fmt.Sprintf(baseOrgName+"%d-child%d-client%d", i, m, p),
					childSignKey, childSignCert)
				if err != nil {
					fmt.Printf("error generating server certificate for %s%d-child%d-client%d : %s\n",
						baseOrgName, i, m, p, err.Error())
				}
			}
		}
	}

}
