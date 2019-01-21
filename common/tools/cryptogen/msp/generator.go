
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
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/tools/cryptogen/ca"
	"github.com/hyperledger/fabric/common/tools/cryptogen/csp"
	fabricmsp "github.com/hyperledger/fabric/msp"
	"gopkg.in/yaml.v2"
)

const (
	CLIENT = iota
	ORDERER
	PEER
)

const (
	CLIENTOU = "client"
	PEEROU   = "peer"
)

var nodeOUMap = map[int]string{
	CLIENT: CLIENTOU,
	PEER:   PEEROU,
}

func GenerateLocalMSP(baseDir, name string, sans []string, signCA *ca.CA,
	tlsCA *ca.CA, nodeType int, nodeOUs bool) error {

//
	mspDir := filepath.Join(baseDir, "msp")
	tlsDir := filepath.Join(baseDir, "tls")

	err := createFolderStructure(mspDir, true)
	if err != nil {
		return err
	}

	err = os.MkdirAll(tlsDir, 0755)
	if err != nil {
		return err
	}

 /*
  创建MSP标识项目
 **/

//获取密钥库路径
	keystore := filepath.Join(mspDir, "keystore")

//生成私钥
	priv, _, err := csp.GeneratePrivateKey(keystore)
	if err != nil {
		return err
	}

//获取公钥
	ecPubKey, err := csp.GetECPublicKey(priv)
	if err != nil {
		return err
	}
//使用签名CA生成X509证书
	var ous []string
	if nodeOUs {
		ous = []string{nodeOUMap[nodeType]}
	}
	cert, err := signCA.SignCertificate(filepath.Join(mspDir, "signcerts"),
		name, ous, nil, ecPubKey, x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	if err != nil {
		return err
	}

//将项目写入MSP文件夹

//签名的CA证书进入cacerts
	err = x509Export(filepath.Join(mspDir, "cacerts", x509Filename(signCA.Name)), signCA.SignCert)
	if err != nil {
		return err
	}
//TLS CA证书进入TLSCACERT
	err = x509Export(filepath.Join(mspDir, "tlscacerts", x509Filename(tlsCA.Name)), tlsCA.SignCert)
	if err != nil {
		return err
	}

//如果需要，生成config.yaml
	if nodeOUs && nodeType == PEER {
		exportConfig(mspDir, "cacerts/"+x509Filename(signCA.Name), true)
	}

//
//这意味着签名身份
//此MSP的管理员也是此MSP的管理员
//注意：admincerts文件夹将是
//由copyadmincert清除，但是
//
//单元测试
	err = x509Export(filepath.Join(mspDir, "admincerts", x509Filename(name)), cert)
	if err != nil {
		return err
	}

 /*
  在tls文件夹中生成tls项目
 **/


//生成私钥
	tlsPrivKey, _, err := csp.GeneratePrivateKey(tlsDir)
	if err != nil {
		return err
	}
//获取公钥
	tlsPubKey, err := csp.GetECPublicKey(tlsPrivKey)
	if err != nil {
		return err
	}
//使用TLS CA生成X509证书
	_, err = tlsCA.SignCertificate(filepath.Join(tlsDir),
		name, nil, sans, tlsPubKey, x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth})
	if err != nil {
		return err
	}
	err = x509Export(filepath.Join(tlsDir, "ca.crt"), tlsCA.SignCert)
	if err != nil {
		return err
	}

//重命名生成的TLS X509证书
	tlsFilePrefix := "server"
	if nodeType == CLIENT {
		tlsFilePrefix = "client"
	}
	err = os.Rename(filepath.Join(tlsDir, x509Filename(name)),
		filepath.Join(tlsDir, tlsFilePrefix+".crt"))
	if err != nil {
		return err
	}

	err = keyExport(tlsDir, filepath.Join(tlsDir, tlsFilePrefix+".key"), tlsPrivKey)
	if err != nil {
		return err
	}

	return nil
}

func GenerateVerifyingMSP(baseDir string, signCA *ca.CA, tlsCA *ca.CA, nodeOUs bool) error {

//创建文件夹结构并将工件写入适当的位置
	err := createFolderStructure(baseDir, false)
	if err == nil {
//签名的CA证书进入cacerts
		err = x509Export(filepath.Join(baseDir, "cacerts", x509Filename(signCA.Name)), signCA.SignCert)
		if err != nil {
			return err
		}
//TLS CA证书进入TLSCACERT
		err = x509Export(filepath.Join(baseDir, "tlscacerts", x509Filename(tlsCA.Name)), tlsCA.SignCert)
		if err != nil {
			return err
		}
	}

//如果需要，生成config.yaml
	if nodeOUs {
		exportConfig(baseDir, "cacerts/"+x509Filename(signCA.Name), true)
	}

//创建一个一次性证书作为管理证书
//注意：admincerts文件夹将是
//由copyadmincert清除，但是
//为了方便，我们暂时离开一个有效的管理员
//单元测试
	factory.InitFactories(nil)
	bcsp := factory.GetDefault()
	priv, err := bcsp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{Temporary: true})
	ecPubKey, err := csp.GetECPublicKey(priv)
	if err != nil {
		return err
	}
	_, err = signCA.SignCertificate(filepath.Join(baseDir, "admincerts"), signCA.Name,
		nil, nil, ecPubKey, x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	if err != nil {
		return err
	}

	return nil
}

func createFolderStructure(rootDir string, local bool) error {

	var folders []string
//创建admincerts、cacerts、keystore和signcerts文件夹
	folders = []string{
		filepath.Join(rootDir, "admincerts"),
		filepath.Join(rootDir, "cacerts"),
		filepath.Join(rootDir, "tlscacerts"),
	}
	if local {
		folders = append(folders, filepath.Join(rootDir, "keystore"),
			filepath.Join(rootDir, "signcerts"))
	}

	for _, folder := range folders {
		err := os.MkdirAll(folder, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func x509Filename(name string) string {
	return name + "-cert.pem"
}

func x509Export(path string, cert *x509.Certificate) error {
	return pemExport(path, "CERTIFICATE", cert.Raw)
}

func keyExport(keystore, output string, key bccsp.Key) error {
	id := hex.EncodeToString(key.SKI())

	return os.Rename(filepath.Join(keystore, id+"_sk"), output)
}

func pemExport(path, pemType string, bytes []byte) error {
//将PEM写入文件
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{Type: pemType, Bytes: bytes})
}

func exportConfig(mspDir, caFile string, enable bool) error {
	var config = &fabricmsp.Configuration{
		NodeOUs: &fabricmsp.NodeOUs{
			Enable: enable,
			ClientOUIdentifier: &fabricmsp.OrganizationalUnitIdentifiersConfiguration{
				Certificate:                  caFile,
				OrganizationalUnitIdentifier: CLIENTOU,
			},
			PeerOUIdentifier: &fabricmsp.OrganizationalUnitIdentifiersConfiguration{
				Certificate:                  caFile,
				OrganizationalUnitIdentifier: PEEROU,
			},
		},
	}

	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(mspDir, "config.yaml"))
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString(string(configBytes))

	return err
}
