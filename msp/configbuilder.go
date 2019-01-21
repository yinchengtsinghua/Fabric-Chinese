
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
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

//OrganizationalUnitIdentifiersConfiguration用于表示组织单位
//以及关联的可信证书
type OrganizationalUnitIdentifiersConfiguration struct {
//证书是指向根证书或中间证书的路径
	Certificate string `yaml:"Certificate,omitempty"`
//OrganizationalUnitIdentifier是组织单位的名称
	OrganizationalUnitIdentifier string `yaml:"OrganizationalUnitIdentifier,omitempty"`
}

//nodeous包含有关如何区分客户、对等方和订购方的信息
//基于U.如果强制执行检查，通过将启用设置为真，
//如果标识是客户机、对等机或
//订购者一个身份应该只有这些特殊的组织单位中的一个。
type NodeOUs struct {
//启用激活OU强制
	Enable bool `yaml:"Enable,omitempty"`
//clientouidentifier指定如何通过ou识别客户机
	ClientOUIdentifier *OrganizationalUnitIdentifiersConfiguration `yaml:"ClientOUIdentifier,omitempty"`
//peerouidentifier指定如何通过ou识别对等
	PeerOUIdentifier *OrganizationalUnitIdentifiersConfiguration `yaml:"PeerOUIdentifier,omitempty"`
}

//配置表示MSP可以配备的附件配置。
//默认情况下，此配置存储在yaml文件中
type Configuration struct {
//OrganizationalUnitIdentifiers是OU的列表。如果设置了此选项，则MSP
//将认为一个标识有效，但它至少包含一个这样的OU
	OrganizationalUnitIdentifiers []*OrganizationalUnitIdentifiersConfiguration `yaml:"OrganizationalUnitIdentifiers,omitempty"`
//nodeous使MSP能够区分基于客户机、对等机和订购方的
//关于你的身份。
	NodeOUs *NodeOUs `yaml:"NodeOUs,omitempty"`
}

func readFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file %s", file)
	}

	return fileCont, nil
}

func readPemFile(file string) ([]byte, error) {
	bytes, err := readFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading from file %s failed", file)
	}

	b, _ := pem.Decode(bytes)
if b == nil { //TODO:还要检查类型是否是我们期望的类型（cert与key..）。
		return nil, errors.Errorf("no pem content for file %s", file)
	}

	return bytes, nil
}

func getPemMaterialFromDir(dir string) ([][]byte, error) {
	mspLogger.Debugf("Reading directory %s", dir)

	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, err
	}

	content := make([][]byte, 0)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read directory %s", dir)
	}

	for _, f := range files {
		fullName := filepath.Join(dir, f.Name())

		f, err := os.Stat(fullName)
		if err != nil {
			mspLogger.Warningf("Failed to stat %s: %s", fullName, err)
			continue
		}
		if f.IsDir() {
			continue
		}

		mspLogger.Debugf("Inspecting file %s", fullName)

		item, err := readPemFile(fullName)
		if err != nil {
			mspLogger.Warningf("Failed reading file %s: %s", fullName, err)
			continue
		}

		content = append(content, item)
	}

	return content, nil
}

const (
	cacerts              = "cacerts"
	admincerts           = "admincerts"
	signcerts            = "signcerts"
	keystore             = "keystore"
	intermediatecerts    = "intermediatecerts"
	crlsfolder           = "crls"
	configfilename       = "config.yaml"
	tlscacerts           = "tlscacerts"
	tlsintermediatecerts = "tlsintermediatecerts"
)

func SetupBCCSPKeystoreConfig(bccspConfig *factory.FactoryOpts, keystoreDir string) *factory.FactoryOpts {
	if bccspConfig == nil {
		bccspConfig = factory.GetDefaultOpts()
	}

	if bccspConfig.ProviderName == "SW" {
		if bccspConfig.SwOpts == nil {
			bccspConfig.SwOpts = factory.GetDefaultOpts().SwOpts
		}

//仅当keystorepath为空时才重写它
		if bccspConfig.SwOpts.FileKeystore == nil ||
			bccspConfig.SwOpts.FileKeystore.KeyStorePath == "" {
			bccspConfig.SwOpts.Ephemeral = false
			bccspConfig.SwOpts.FileKeystore = &factory.FileKeystoreOpts{KeyStorePath: keystoreDir}
		}
	}

	return bccspConfig
}

//GetLocalMSPConfigWithType返回本地MSP
//在指定的
//目录，具有指定的ID和类型
func GetLocalMspConfigWithType(dir string, bccspConfig *factory.FactoryOpts, ID, mspType string) (*msp.MSPConfig, error) {
	switch mspType {
	case ProviderTypeToString(FABRIC):
		return GetLocalMspConfig(dir, bccspConfig, ID)
	case ProviderTypeToString(IDEMIX):
		return GetIdemixMspConfig(dir, ID)
	default:
		return nil, errors.Errorf("unknown MSP type '%s'", mspType)
	}
}

func GetLocalMspConfig(dir string, bccspConfig *factory.FactoryOpts, ID string) (*msp.MSPConfig, error) {
	signcertDir := filepath.Join(dir, signcerts)
	keystoreDir := filepath.Join(dir, keystore)
	bccspConfig = SetupBCCSPKeystoreConfig(bccspConfig, keystoreDir)

	err := factory.InitFactories(bccspConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "could not initialize BCCSP Factories")
	}

	signcert, err := getPemMaterialFromDir(signcertDir)
	if err != nil || len(signcert) == 0 {
		return nil, errors.Wrapf(err, "could not load a valid signer certificate from directory %s", signcertDir)
	}

 /*Fixme：目前我们正在做以下假设
 1）只有一个签名证书
 2）BCCSP的密钥库具有与滑雪板匹配的私钥
    签署证书
 **/


	sigid := &msp.SigningIdentityInfo{PublicSigner: signcert[0], PrivateSigner: nil}

	return getMspConfig(dir, ID, sigid)
}

//GetVerifyingMSPConfig返回给定目录、ID和类型的MSP配置
func GetVerifyingMspConfig(dir, ID, mspType string) (*msp.MSPConfig, error) {
	switch mspType {
	case ProviderTypeToString(FABRIC):
		return getMspConfig(dir, ID, nil)
	case ProviderTypeToString(IDEMIX):
		return GetIdemixMspConfig(dir, ID)
	default:
		return nil, errors.Errorf("unknown MSP type '%s'", mspType)
	}
}

func getMspConfig(dir string, ID string, sigid *msp.SigningIdentityInfo) (*msp.MSPConfig, error) {
	cacertDir := filepath.Join(dir, cacerts)
	admincertDir := filepath.Join(dir, admincerts)
	intermediatecertsDir := filepath.Join(dir, intermediatecerts)
	crlsDir := filepath.Join(dir, crlsfolder)
	configFile := filepath.Join(dir, configfilename)
	tlscacertDir := filepath.Join(dir, tlscacerts)
	tlsintermediatecertsDir := filepath.Join(dir, tlsintermediatecerts)

	cacerts, err := getPemMaterialFromDir(cacertDir)
	if err != nil || len(cacerts) == 0 {
		return nil, errors.WithMessage(err, fmt.Sprintf("could not load a valid ca certificate from directory %s", cacertDir))
	}

	admincert, err := getPemMaterialFromDir(admincertDir)
	if err != nil || len(admincert) == 0 {
		return nil, errors.WithMessage(err, fmt.Sprintf("could not load a valid admin certificate from directory %s", admincertDir))
	}

	intermediatecerts, err := getPemMaterialFromDir(intermediatecertsDir)
	if os.IsNotExist(err) {
		mspLogger.Debugf("Intermediate certs folder not found at [%s]. Skipping. [%s]", intermediatecertsDir, err)
	} else if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed loading intermediate ca certs at [%s]", intermediatecertsDir))
	}

	tlsCACerts, err := getPemMaterialFromDir(tlscacertDir)
	tlsIntermediateCerts := [][]byte{}
	if os.IsNotExist(err) {
		mspLogger.Debugf("TLS CA certs folder not found at [%s]. Skipping and ignoring TLS intermediate CA folder. [%s]", tlsintermediatecertsDir, err)
	} else if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed loading TLS ca certs at [%s]", tlsintermediatecertsDir))
	} else if len(tlsCACerts) != 0 {
		tlsIntermediateCerts, err = getPemMaterialFromDir(tlsintermediatecertsDir)
		if os.IsNotExist(err) {
			mspLogger.Debugf("TLS intermediate certs folder not found at [%s]. Skipping. [%s]", tlsintermediatecertsDir, err)
		} else if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("failed loading TLS intermediate ca certs at [%s]", tlsintermediatecertsDir))
		}
	} else {
		mspLogger.Debugf("TLS CA certs folder at [%s] is empty. Skipping.", tlsintermediatecertsDir)
	}

	crls, err := getPemMaterialFromDir(crlsDir)
	if os.IsNotExist(err) {
		mspLogger.Debugf("crls folder not found at [%s]. Skipping. [%s]", crlsDir, err)
	} else if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed loading crls at [%s]", crlsDir))
	}

//加载配置文件
//如果存在配置文件，则加载它
//否则跳过它
	var ouis []*msp.FabricOUIdentifier
	var nodeOUs *msp.FabricNodeOUs
	_, err = os.Stat(configFile)
	if err == nil {
//加载文件，如果加载失败，则
//返回错误
		raw, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, errors.Wrapf(err, "failed loading configuration file at [%s]", configFile)
		}

		configuration := Configuration{}
		err = yaml.Unmarshal(raw, &configuration)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unmarshalling configuration file at [%s]", configFile)
		}

//准备组织初始化标识符
		if len(configuration.OrganizationalUnitIdentifiers) > 0 {
			for _, ouID := range configuration.OrganizationalUnitIdentifiers {
				f := filepath.Join(dir, ouID.Certificate)
				raw, err = readFile(f)
				if err != nil {
					return nil, errors.Wrapf(err, "failed loading OrganizationalUnit certificate at [%s]", f)
				}

				oui := &msp.FabricOUIdentifier{
					Certificate:                  raw,
					OrganizationalUnitIdentifier: ouID.OrganizationalUnitIdentifier,
				}
				ouis = append(ouis, oui)
			}
		}

//作怪
		if configuration.NodeOUs != nil && configuration.NodeOUs.Enable {
			mspLogger.Debug("Loading NodeOUs")
			if configuration.NodeOUs.ClientOUIdentifier == nil || len(configuration.NodeOUs.ClientOUIdentifier.OrganizationalUnitIdentifier) == 0 {
				return nil, errors.New("Failed loading NodeOUs. ClientOU must be different from nil.")
			}
			if configuration.NodeOUs.PeerOUIdentifier == nil || len(configuration.NodeOUs.PeerOUIdentifier.OrganizationalUnitIdentifier) == 0 {
				return nil, errors.New("Failed loading NodeOUs. PeerOU must be different from nil.")
			}

			nodeOUs = &msp.FabricNodeOUs{
				Enable:             configuration.NodeOUs.Enable,
				ClientOuIdentifier: &msp.FabricOUIdentifier{OrganizationalUnitIdentifier: configuration.NodeOUs.ClientOUIdentifier.OrganizationalUnitIdentifier},
				PeerOuIdentifier:   &msp.FabricOUIdentifier{OrganizationalUnitIdentifier: configuration.NodeOUs.PeerOUIdentifier.OrganizationalUnitIdentifier},
			}

//读取证书（如果已定义）

//克里门特
			f := filepath.Join(dir, configuration.NodeOUs.ClientOUIdentifier.Certificate)
			raw, err = readFile(f)
			if err != nil {
				mspLogger.Infof("Failed loading ClientOU certificate at [%s]: [%s]", f, err)
			} else {
				nodeOUs.ClientOuIdentifier.Certificate = raw
			}

//佩罗
			f = filepath.Join(dir, configuration.NodeOUs.PeerOUIdentifier.Certificate)
			raw, err = readFile(f)
			if err != nil {
				mspLogger.Debugf("Failed loading PeerOU certificate at [%s]: [%s]", f, err)
			} else {
				nodeOUs.PeerOuIdentifier.Certificate = raw
			}
		}
	} else {
		mspLogger.Debugf("MSP configuration file not found at [%s]: [%s]", configFile, err)
	}

//设置FabricCryptoConfig
	cryptoConfig := &msp.FabricCryptoConfig{
		SignatureHashFamily:            bccsp.SHA2,
		IdentityIdentifierHashFunction: bccsp.SHA256,
	}

//撰写fabricmspconfig
	fmspconf := &msp.FabricMSPConfig{
		Admins:                        admincert,
		RootCerts:                     cacerts,
		IntermediateCerts:             intermediatecerts,
		SigningIdentity:               sigid,
		Name:                          ID,
		OrganizationalUnitIdentifiers: ouis,
		RevocationList:                crls,
		CryptoConfig:                  cryptoConfig,
		TlsRootCerts:                  tlsCACerts,
		TlsIntermediateCerts:          tlsIntermediateCerts,
		FabricNodeOus:                 nodeOUs,
	}

	fmpsjs, _ := proto.Marshal(fmspconf)

	mspconf := &msp.MSPConfig{Config: fmpsjs, Type: int32(FABRIC)}

	return mspconf, nil
}

const (
	IdemixConfigDirMsp                  = "msp"
	IdemixConfigDirUser                 = "user"
	IdemixConfigFileIssuerPublicKey     = "IssuerPublicKey"
	IdemixConfigFileRevocationPublicKey = "RevocationPublicKey"
	IdemixConfigFileSigner              = "SignerConfig"
)

//getidemixmspconfig返回idemix msp的配置
func GetIdemixMspConfig(dir string, ID string) (*msp.MSPConfig, error) {
	ipkBytes, err := readFile(filepath.Join(dir, IdemixConfigDirMsp, IdemixConfigFileIssuerPublicKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read issuer public key file")
	}

	revocationPkBytes, err := readFile(filepath.Join(dir, IdemixConfigDirMsp, IdemixConfigFileRevocationPublicKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read revocation public key file")
	}

	idemixConfig := &msp.IdemixMSPConfig{
		Name:         ID,
		Ipk:          ipkBytes,
		RevocationPk: revocationPkBytes,
	}

	signerBytes, err := readFile(filepath.Join(dir, IdemixConfigDirUser, IdemixConfigFileSigner))
	if err == nil {
		signerConfig := &msp.IdemixMSPSignerConfig{}
		err = proto.Unmarshal(signerBytes, signerConfig)
		if err != nil {
			return nil, err
		}
		idemixConfig.Signer = signerConfig
	}

	confBytes, err := proto.Marshal(idemixConfig)
	if err != nil {
		return nil, err
	}

	return &msp.MSPConfig{Config: confBytes, Type: int32(IDEMIX)}, nil
}
