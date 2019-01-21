
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


package common

import (
	"io/ioutil"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

//networkconfig提供了超级账本结构网络的静态定义
type NetworkConfig struct {
	Name                   string                          `yaml:"name"`
	Xtype                  string                          `yaml:"x-type"`
	Description            string                          `yaml:"description"`
	Version                string                          `yaml:"version"`
	Channels               map[string]ChannelNetworkConfig `yaml:"channels"`
	Organizations          map[string]OrganizationConfig   `yaml:"organizations"`
	Peers                  map[string]PeerConfig           `yaml:"peers"`
	Client                 ClientConfig                    `yaml:"client"`
	Orderers               map[string]OrdererConfig        `yaml:"orderers"`
	CertificateAuthorities map[string]CAConfig             `yaml:"certificateAuthorities"`
}

//clientconfig-当前未被cli使用
type ClientConfig struct {
	Organization    string              `yaml:"organization"`
	Logging         LoggingType         `yaml:"logging"`
	CryptoConfig    CCType              `yaml:"cryptoconfig"`
	TLS             TLSType             `yaml:"tls"`
	CredentialStore CredentialStoreType `yaml:"credentialStore"`
}

//cli当前未使用loggingtype
type LoggingType struct {
	Level string `yaml:"level"`
}

//cctype-当前未由cli使用
type CCType struct {
	Path string `yaml:"path"`
}

//tlstype-当前未被cli使用
type TLSType struct {
	Enabled bool `yaml:"enabled"`
}

//CredentialStoreType-当前未由CLI使用
type CredentialStoreType struct {
	Path        string `yaml:"path"`
	CryptoStore struct {
		Path string `yaml:"path"`
	}
	Wallet string `yaml:"wallet"`
}

//channelnetworkconfig提供网络的通道定义
type ChannelNetworkConfig struct {
//订购者订购服务节点列表
	Orderers []string `yaml:"orderers"`
//对等端此组织中的对等通道列表
//要获取真正的对等配置对象，请使用名称字段并获取networkconfig.peers[名称]
	Peers map[string]PeerChannelConfig `yaml:"peers"`
//链码服务清单
	Chaincodes []string `yaml:"chaincodes"`
}

//peerchannelconfig定义对等功能
type PeerChannelConfig struct {
	EndorsingPeer  bool `yaml:"endorsingPeer"`
	ChaincodeQuery bool `yaml:"chaincodeQuery"`
	LedgerQuery    bool `yaml:"ledgerQuery"`
	EventSource    bool `yaml:"eventSource"`
}

//OrganizationConfig提供网络中组织的定义
//当前未被CLI使用
type OrganizationConfig struct {
	MspID                  string    `yaml:"mspid"`
	Peers                  []string  `yaml:"peers"`
	CryptoPath             string    `yaml:"cryptoPath"`
	CertificateAuthorities []string  `yaml:"certificateAuthorities"`
	AdminPrivateKey        TLSConfig `yaml:"adminPrivateKey"`
	SignedCert             TLSConfig `yaml:"signedCert"`
}

//orderconfig定义了一个order配置
//当前未被CLI使用
type OrdererConfig struct {
	URL         string                 `yaml:"url"`
	GrpcOptions map[string]interface{} `yaml:"grpcOptions"`
	TLSCACerts  TLSConfig              `yaml:"tlsCACerts"`
}

//peerconfig定义对等配置
type PeerConfig struct {
	URL         string                 `yaml:"url"`
	EventURL    string                 `yaml:"eventUrl"`
	GRPCOptions map[string]interface{} `yaml:"grpcOptions"`
	TLSCACerts  TLSConfig              `yaml:"tlsCACerts"`
}

//caconfig定义CA配置
//当前未被CLI使用
type CAConfig struct {
	URL         string                 `yaml:"url"`
	HTTPOptions map[string]interface{} `yaml:"httpOptions"`
	TLSCACerts  MutualTLSConfig        `yaml:"tlsCACerts"`
	Registrar   EnrollCredentials      `yaml:"registrar"`
	CaName      string                 `yaml:"caName"`
}

//注册凭据保留用于注册的凭据
//当前未被CLI使用
type EnrollCredentials struct {
	EnrollID     string `yaml:"enrollId"`
	EnrollSecret string `yaml:"enrollSecret"`
}

//tlsconfig tls配置
type TLSConfig struct {
//以下两个字段可以互换。
//如果路径可用，则它将用于加载证书
//如果PEM可用，则它具有证书的原始数据，将按原样使用。
//证书根证书路径
	Path string `yaml:"path"`
//证书实际内容
	Pem string `yaml:"pem"`
}

//MutualTlsConfig相互TLS配置
//当前未被CLI使用
type MutualTLSConfig struct {
	Pem []string `yaml:"pem"`

//certfiles用于TLS验证的根证书（逗号分隔的路径列表）
	Path string `yaml:"path"`

//客户端TLS信息
	Client TLSKeyPair `yaml:"client"`
}

//tlskeypair包含用于tls加密的私钥和证书
//当前未被CLI使用
type TLSKeyPair struct {
	Key  TLSConfig `yaml:"key"`
	Cert TLSConfig `yaml:"cert"`
}

//getconfig将提供的连接配置文件取消标记到网络中
//配置结构
func GetConfig(fileName string) (*NetworkConfig, error) {
	if fileName == "" {
		return nil, errors.New("filename cannot be empty")
	}

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "error reading connection profile")
	}

	configData := string(data)
	config := &NetworkConfig{}
	err = yaml.Unmarshal([]byte(configData), &config)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling YAML")
	}

	return config, nil
}
