
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


package main

//IDemixGen是一个命令行工具，用于生成CA的密钥和
//生成用于签名和验证的MSP配置
//此工具可用于设置对等端和CA以支持
//身份混音器MSP

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/tools/idemixgen/idemixca"
	"github.com/hyperledger/fabric/common/tools/idemixgen/metadata"
	"github.com/hyperledger/fabric/idemix"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	IdemixDirIssuer             = "ca"
	IdemixConfigIssuerSecretKey = "IssuerSecretKey"
	IdemixConfigRevocationKey   = "RevocationKey"
)

//命令行标志
var (
	app = kingpin.New("idemixgen", "Utility for generating key material to be used with the Identity Mixer MSP in Hyperledger Fabric")

	outputDir = app.Flag("output", "The output directory in which to place artifacts").Default("idemix-config").String()

	genIssuerKey            = app.Command("ca-keygen", "Generate CA key material")
	genSignerConfig         = app.Command("signerconfig", "Generate a default signer for this Idemix MSP")
	genCredOU               = genSignerConfig.Flag("org-unit", "The Organizational Unit of the default signer").Short('u').String()
	genCredIsAdmin          = genSignerConfig.Flag("admin", "Make the default signer admin").Short('a').Bool()
	genCredEnrollmentId     = genSignerConfig.Flag("enrollmentId", "The enrollment id of the default signer").Short('e').String()
	genCredRevocationHandle = genSignerConfig.Flag("revocationHandle", "The handle used to revoke this signer").Short('r').Int()

	version = app.Command("version", "Show version information")
)

func main() {
	app.HelpFlag.Short('h')

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case genIssuerKey.FullCommand():
		isk, ipk, err := idemixca.GenerateIssuerKey()
		handleError(err)

		revocationKey, err := idemix.GenerateLongTermRevocationKey()
		handleError(err)
		encodedRevocationSK, err := x509.MarshalECPrivateKey(revocationKey)
		handleError(err)
		pemEncodedRevocationSK := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedRevocationSK})
		handleError(err)
		encodedRevocationPK, err := x509.MarshalPKIXPublicKey(revocationKey.Public())
		handleError(err)
		pemEncodedRevocationPK := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encodedRevocationPK})

//防止覆盖现有密钥
		path := filepath.Join(*outputDir, IdemixDirIssuer)
		checkDirectoryNotExists(path, fmt.Sprintf("Directory %s already exists", path))

		path = filepath.Join(*outputDir, msp.IdemixConfigDirMsp)
		checkDirectoryNotExists(path, fmt.Sprintf("Directory %s already exists", path))

//将私钥和公钥写入文件
		handleError(os.MkdirAll(filepath.Join(*outputDir, IdemixDirIssuer), 0770))
		handleError(os.MkdirAll(filepath.Join(*outputDir, msp.IdemixConfigDirMsp), 0770))
		writeFile(filepath.Join(*outputDir, IdemixDirIssuer, IdemixConfigIssuerSecretKey), isk)
		writeFile(filepath.Join(*outputDir, IdemixDirIssuer, IdemixConfigRevocationKey), pemEncodedRevocationSK)
		writeFile(filepath.Join(*outputDir, IdemixDirIssuer, msp.IdemixConfigFileIssuerPublicKey), ipk)
		writeFile(filepath.Join(*outputDir, msp.IdemixConfigDirMsp, msp.IdemixConfigFileRevocationPublicKey), pemEncodedRevocationPK)
		writeFile(filepath.Join(*outputDir, msp.IdemixConfigDirMsp, msp.IdemixConfigFileIssuerPublicKey), ipk)

	case genSignerConfig.FullCommand():
		roleMask := 0
		if *genCredIsAdmin {
			roleMask = msp.GetRoleMaskFromIdemixRole(msp.ADMIN)
		} else {
			roleMask = msp.GetRoleMaskFromIdemixRole(msp.MEMBER)
		}
		config, err := idemixca.GenerateSignerConfig(roleMask, *genCredOU, *genCredEnrollmentId, *genCredRevocationHandle, readIssuerKey(), readRevocationKey())
		handleError(err)

		path := filepath.Join(*outputDir, msp.IdemixConfigDirUser)
		checkDirectoryNotExists(path, fmt.Sprintf("This MSP config already contains a directory \"%s\"", path))

//将配置写入文件
		handleError(os.Mkdir(filepath.Join(*outputDir, msp.IdemixConfigDirUser), 0770))
		writeFile(filepath.Join(*outputDir, msp.IdemixConfigDirUser, msp.IdemixConfigFileSigner), config)

	case version.FullCommand():
		printVersion()
	}
}

func printVersion() {
	fmt.Println(metadata.GetVersionInfo())
}

//WRITEFILE将字节写入文件并在出现错误时死机
func writeFile(path string, contents []byte) {
	handleError(ioutil.WriteFile(path, contents, 0640))
}

//readissuerkey从当前目录中读取颁发者密钥
func readIssuerKey() *idemix.IssuerKey {
	path := filepath.Join(*outputDir, IdemixDirIssuer, IdemixConfigIssuerSecretKey)
	isk, err := ioutil.ReadFile(path)
	if err != nil {
		handleError(errors.Wrapf(err, "failed to open issuer secret key file: %s", path))
	}
	path = filepath.Join(*outputDir, IdemixDirIssuer, msp.IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		handleError(errors.Wrapf(err, "failed to open issuer public key file: %s", path))
	}
	ipk := &idemix.IssuerPublicKey{}
	handleError(proto.Unmarshal(ipkBytes, ipk))
	key := &idemix.IssuerKey{Isk: isk, Ipk: ipk}

	return key
}

func readRevocationKey() *ecdsa.PrivateKey {
	path := filepath.Join(*outputDir, IdemixDirIssuer, IdemixConfigRevocationKey)
	keyBytes, err := ioutil.ReadFile(path)
	if err != nil {
		handleError(errors.Wrapf(err, "failed to open revocation secret key file: %s", path))
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		handleError(errors.Errorf("failed to decode ECDSA private key"))
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	handleError(err)

	return key
}

//checkdirectorynotexists检查具有给定路径的目录是否已存在，如果是这种情况，则退出
func checkDirectoryNotExists(path string, errorMessage string) {
	_, err := os.Stat(path)
	if err == nil {
		handleError(errors.New(errorMessage))
	}
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
