
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


package localconfig

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/viperutil"
	cf "github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/spf13/viper"
)

const (
//
	Prefix string = "CONFIGTX"
)

var logger = flogging.MustGetLogger("common.tools.configtxgen.localconfig")
var configName = strings.ToLower(Prefix)

const (
//testchainid是用于测试的通道名，当
//
	TestChainID = "testchainid"

//
//
	SampleInsecureSoloProfile = "SampleInsecureSolo"
//
//只有管理员权限的基本成员资格，并且使用SOLO进行订购。
	SampleDevModeSoloProfile = "SampleDevModeSolo"
//samplesinglemspssoloprofile引用示例配置文件，其中包括
//只有示例MSP和使用SOLO进行订购。
	SampleSingleMSPSoloProfile = "SampleSingleMSPSolo"

//sampleinsecurekafkaprofile引用的示例配置文件不是
//
	SampleInsecureKafkaProfile = "SampleInsecureKafka"
//
//管理特权的基本成员身份，并使用Kafka进行订购。
	SampleDevModeKafkaProfile = "SampleDevModeKafka"
//samplesinglemspkafkaprofile引用示例配置文件，其中包括
//
	SampleSingleMSPKafkaProfile = "SampleSingleMSPKafka"

//sampledevmodeetcdraftprofile引用用于测试的样本配置文件
//基于ETCD/RAFT的订购服务。
	SampleDevModeEtcdRaftProfile = "SampleDevModeEtcdRaft"

//samplesinglemschannelprofile引用示例配置文件，
//仅包括示例MSP，用于创建通道
	SampleSingleMSPChannelProfile = "SampleSingleMSPChannel"

//
//
	SampleConsortiumName = "SampleConsortium"
//
	SampleOrgName = "SampleOrg"

//adminroleadminprincipal设置为adminrole，以使
//键入要用作管理主体默认值的admin
	AdminRoleAdminPrincipal = "Role.ADMIN"
//MemberRoleADMinPrincipal设置为AdminRole，以导致的MSP角色
//要用作管理主体默认值的类型成员
	MemberRoleAdminPrincipal = "Role.MEMBER"
)

//顶层由configtxgen工具使用的结构组成。
type TopLevel struct {
	Profiles      map[string]*Profile        `yaml:"Profiles"`
	Organizations []*Organization            `yaml:"Organizations"`
	Channel       *Profile                   `yaml:"Channel"`
	Application   *Application               `yaml:"Application"`
	Orderer       *Orderer                   `yaml:"Orderer"`
	Capabilities  map[string]map[string]bool `yaml:"Capabilities"`
	Resources     *Resources                 `yaml:"Resources"`
}

//配置文件编码订购者/应用程序配置组合
//
type Profile struct {
	Consortium   string                 `yaml:"Consortium"`
	Application  *Application           `yaml:"Application"`
	Orderer      *Orderer               `yaml:"Orderer"`
	Consortiums  map[string]*Consortium `yaml:"Consortiums"`
	Capabilities map[string]bool        `yaml:"Capabilities"`
	Policies     map[string]*Policy     `yaml:"Policies"`
}

//策略对通道配置策略进行编码
type Policy struct {
	Type string `yaml:"Type"`
	Rule string `yaml:"Rule"`
}

//联合体代表一组可能创造渠道的组织。
//彼此
type Consortium struct {
	Organizations []*Organization `yaml:"Organizations"`
}

//应用程序编码配置中所需的应用程序级配置
//交易。
type Application struct {
	Organizations []*Organization    `yaml:"Organizations"`
	Capabilities  map[string]bool    `yaml:"Capabilities"`
	Resources     *Resources         `yaml:"Resources"`
	Policies      map[string]*Policy `yaml:"Policies"`
	ACLs          map[string]string  `yaml:"ACLs"`
}

//资源编码需要的应用程序级资源配置
//播种资源树
type Resources struct {
	DefaultModPolicy string
}

//组织编码中所需的组织级配置
//配置事务。
type Organization struct {
	Name     string             `yaml:"Name"`
	ID       string             `yaml:"ID"`
	MSPDir   string             `yaml:"MSPDir"`
	MSPType  string             `yaml:"MSPType"`
	Policies map[string]*Policy `yaml:"Policies"`

//注意：Viper反序列化似乎不关心
//嵌入类型，因此我们使用一个组织结构
//对于订购者和应用程序。
	AnchorPeers []*AnchorPeer `yaml:"AnchorPeers"`

//
//它用于修改默认策略生成，但是策略
//现在可以显式指定，因此它是多余的和不必要的
	AdminPrincipal string `yaml:"AdminPrincipal"`
}

//anchor peer对必要字段进行编码，以标识锚定对等。
type AnchorPeer struct {
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
}

//
//临时引导程序引导订购程序。
type Orderer struct {
	OrdererType   string             `yaml:"OrdererType"`
	Addresses     []string           `yaml:"Addresses"`
	BatchTimeout  time.Duration      `yaml:"BatchTimeout"`
	BatchSize     BatchSize          `yaml:"BatchSize"`
	Kafka         Kafka              `yaml:"Kafka"`
	EtcdRaft      *etcdraft.Metadata `yaml:"EtcdRaft"`
	Organizations []*Organization    `yaml:"Organizations"`
	MaxChannels   uint64             `yaml:"MaxChannels"`
	Capabilities  map[string]bool    `yaml:"Capabilities"`
	Policies      map[string]*Policy `yaml:"Policies"`
}

//
type BatchSize struct {
	MaxMessageCount   uint32 `yaml:"MaxMessageCount"`
	AbsoluteMaxBytes  uint32 `yaml:"AbsoluteMaxBytes"`
	PreferredMaxBytes uint32 `yaml:"PreferredMaxBytes"`
}

//
type Kafka struct {
	Brokers []string `yaml:"Brokers"`
}

var genesisDefaults = TopLevel{
	Orderer: &Orderer{
		OrdererType:  "solo",
		Addresses:    []string{"127.0.0.1:7050"},
		BatchTimeout: 2 * time.Second,
		BatchSize: BatchSize{
			MaxMessageCount:   10,
			AbsoluteMaxBytes:  10 * 1024 * 1024,
			PreferredMaxBytes: 512 * 1024,
		},
		Kafka: Kafka{
			Brokers: []string{"127.0.0.1:9092"},
		},
		EtcdRaft: &etcdraft.Metadata{
			Options: &etcdraft.Options{
				TickInterval:    100,
				ElectionTick:    10,
				HeartbeatTick:   1,
				MaxInflightMsgs: 256,
				MaxSizePerMsg:   1048576,
			},
		},
	},
}

//loadtoplevel只需将configtx.yaml文件加载到上面的结构中，然后
//完成初始化。可以选择提供配置路径，并且
//将用于替代结构的路径env变量。
//
//注意，要使环境覆盖在配置文件中正常工作，请加载
//
func LoadTopLevel(configPaths ...string) *TopLevel {
	config := viper.New()
	if len(configPaths) > 0 {
		for _, p := range configPaths {
			config.AddConfigPath(p)
		}
		config.SetConfigName(configName)
	} else {
		cf.InitViper(config, configName)
	}

//对于环境变量
	config.SetEnvPrefix(Prefix)
	config.AutomaticEnv()

	replacer := strings.NewReplacer(".", "_")
	config.SetEnvKeyReplacer(replacer)

	err := config.ReadInConfig()
	if err != nil {
		logger.Panic("Error reading configuration: ", err)
	}
	logger.Debugf("Using config file: %s", config.ConfigFileUsed())

	var uconf TopLevel
	err = viperutil.EnhancedExactUnmarshal(config, &uconf)
	if err != nil {
		logger.Panic("Error unmarshaling config into struct: ", err)
	}

	(&uconf).completeInitialization(filepath.Dir(config.ConfigFileUsed()))

	logger.Infof("Loaded configuration: %s", config.ConfigFileUsed())

	return &uconf
}

//LOAD返回对应于
//
//代替结构的路径env变量。
func Load(profile string, configPaths ...string) *Profile {
	config := viper.New()
	if len(configPaths) > 0 {
		for _, p := range configPaths {
			config.AddConfigPath(p)
		}
		config.SetConfigName(configName)
	} else {
		cf.InitViper(config, configName)
	}

//对于环境变量
	config.SetEnvPrefix(Prefix)
	config.AutomaticEnv()

//此替换程序允许在特定配置文件中替换
//必须完全限定名称
	replacer := strings.NewReplacer(strings.ToUpper(fmt.Sprintf("profiles.%s.", profile)), "", ".", "_")
	config.SetEnvKeyReplacer(replacer)

	err := config.ReadInConfig()
	if err != nil {
		logger.Panic("Error reading configuration: ", err)
	}
	logger.Debugf("Using config file: %s", config.ConfigFileUsed())

	var uconf TopLevel
	err = viperutil.EnhancedExactUnmarshal(config, &uconf)
	if err != nil {
		logger.Panic("Error unmarshaling config into struct: ", err)
	}

	result, ok := uconf.Profiles[profile]
	if !ok {
		logger.Panic("Could not find profile: ", profile)
	}

	result.completeInitialization(filepath.Dir(config.ConfigFileUsed()))

	logger.Infof("Loaded configuration: %s", config.ConfigFileUsed())

	return result
}

func (t *TopLevel) completeInitialization(configDir string) {
	for _, org := range t.Organizations {
		org.completeInitialization(configDir)
	}

	if t.Orderer != nil {
		t.Orderer.completeInitialization(configDir)
	}
}

func (p *Profile) completeInitialization(configDir string) {
	if p.Application != nil {
		for _, org := range p.Application.Organizations {
			org.completeInitialization(configDir)
		}
		if p.Application.Resources != nil {
			p.Application.Resources.completeInitialization()
		}
	}

	if p.Consortiums != nil {
		for _, consortium := range p.Consortiums {
			for _, org := range consortium.Organizations {
				org.completeInitialization(configDir)
			}
		}
	}

	if p.Orderer != nil {
		for _, org := range p.Orderer.Organizations {
			org.completeInitialization(configDir)
		}
//某些配置文件将不定义医嘱者参数
		p.Orderer.completeInitialization(configDir)
	}
}

func (r *Resources) completeInitialization() {
	for {
		switch {
		case r.DefaultModPolicy == "":
			r.DefaultModPolicy = policies.ChannelApplicationAdmins
		default:
			return
		}
	}
}

func (org *Organization) completeInitialization(configDir string) {
//设置msp类型；如果未指定，则假定为bccsp
	if org.MSPType == "" {
		org.MSPType = msp.ProviderTypeToString(msp.FABRIC)
	}

	if org.AdminPrincipal == "" {
		org.AdminPrincipal = AdminRoleAdminPrincipal
	}
	translatePaths(configDir, org)
}

func (ord *Orderer) completeInitialization(configDir string) {
loop:
	for {
		switch {
		case ord.OrdererType == "":
			logger.Infof("Orderer.OrdererType unset, setting to %v", genesisDefaults.Orderer.OrdererType)
			ord.OrdererType = genesisDefaults.Orderer.OrdererType
		case ord.Addresses == nil:
			logger.Infof("Orderer.Addresses unset, setting to %s", genesisDefaults.Orderer.Addresses)
			ord.Addresses = genesisDefaults.Orderer.Addresses
		case ord.BatchTimeout == 0:
			logger.Infof("Orderer.BatchTimeout unset, setting to %s", genesisDefaults.Orderer.BatchTimeout)
			ord.BatchTimeout = genesisDefaults.Orderer.BatchTimeout
		case ord.BatchSize.MaxMessageCount == 0:
			logger.Infof("Orderer.BatchSize.MaxMessageCount unset, setting to %v", genesisDefaults.Orderer.BatchSize.MaxMessageCount)
			ord.BatchSize.MaxMessageCount = genesisDefaults.Orderer.BatchSize.MaxMessageCount
		case ord.BatchSize.AbsoluteMaxBytes == 0:
			logger.Infof("Orderer.BatchSize.AbsoluteMaxBytes unset, setting to %v", genesisDefaults.Orderer.BatchSize.AbsoluteMaxBytes)
			ord.BatchSize.AbsoluteMaxBytes = genesisDefaults.Orderer.BatchSize.AbsoluteMaxBytes
		case ord.BatchSize.PreferredMaxBytes == 0:
			logger.Infof("Orderer.BatchSize.PreferredMaxBytes unset, setting to %v", genesisDefaults.Orderer.BatchSize.PreferredMaxBytes)
			ord.BatchSize.PreferredMaxBytes = genesisDefaults.Orderer.BatchSize.PreferredMaxBytes
		default:
			break loop
		}
	}

	logger.Infof("orderer type: %s", ord.OrdererType)
//另外，这里是共识类型相关的初始化
//也可以用它来死机处理未知的排序器类型。
	switch ord.OrdererType {
	case "solo":
//这里什么都不做
	case "kafka":
		if ord.Kafka.Brokers == nil {
			logger.Infof("Orderer.Kafka unset, setting to %v", genesisDefaults.Orderer.Kafka.Brokers)
			ord.Kafka.Brokers = genesisDefaults.Orderer.Kafka.Brokers
		}
	case etcdraft.TypeKey:
		if ord.EtcdRaft == nil {
			logger.Panicf("%s raft configuration missing", etcdraft.TypeKey)
		}
		if ord.EtcdRaft.Options == nil {
			logger.Infof("Orderer.EtcdRaft.Options unset, setting to %v", genesisDefaults.Orderer.EtcdRaft.Options)
			ord.EtcdRaft.Options = genesisDefaults.Orderer.EtcdRaft.Options
		}
	second_loop:
		for {
			switch {
			case ord.EtcdRaft.Options.TickInterval == 0:
				logger.Infof("Orderer.EtcdRaft.Options.TickInterval unset, setting to %v", genesisDefaults.Orderer.EtcdRaft.Options.TickInterval)
				ord.EtcdRaft.Options.TickInterval = genesisDefaults.Orderer.EtcdRaft.Options.TickInterval

			case ord.EtcdRaft.Options.ElectionTick == 0:
				logger.Infof("Orderer.EtcdRaft.Options.ElectionTick unset, setting to %v", genesisDefaults.Orderer.EtcdRaft.Options.ElectionTick)
				ord.EtcdRaft.Options.ElectionTick = genesisDefaults.Orderer.EtcdRaft.Options.ElectionTick

			case ord.EtcdRaft.Options.HeartbeatTick == 0:
				logger.Infof("Orderer.EtcdRaft.Options.HeartbeatTick unset, setting to %v", genesisDefaults.Orderer.EtcdRaft.Options.HeartbeatTick)
				ord.EtcdRaft.Options.HeartbeatTick = genesisDefaults.Orderer.EtcdRaft.Options.HeartbeatTick

			case ord.EtcdRaft.Options.MaxInflightMsgs == 0:
				logger.Infof("Orderer.EtcdRaft.Options.MaxInflightMsgs unset, setting to %v", genesisDefaults.Orderer.EtcdRaft.Options.MaxInflightMsgs)
				ord.EtcdRaft.Options.MaxInflightMsgs = genesisDefaults.Orderer.EtcdRaft.Options.MaxInflightMsgs

			case ord.EtcdRaft.Options.MaxSizePerMsg == 0:
				logger.Infof("Orderer.EtcdRaft.Options.MaxSizePerMsg unset, setting to %v", genesisDefaults.Orderer.EtcdRaft.Options.MaxSizePerMsg)
				ord.EtcdRaft.Options.MaxSizePerMsg = genesisDefaults.Orderer.EtcdRaft.Options.MaxSizePerMsg

			case len(ord.EtcdRaft.Consenters) == 0:
				logger.Panicf("%s configuration did not specify any consenter", etcdraft.TypeKey)

			default:
				break second_loop
			}
		}

//为选项验证指定的成员
		if ord.EtcdRaft.Options.ElectionTick <= ord.EtcdRaft.Options.HeartbeatTick {
			logger.Panicf("election tick must be greater than heartbeat tick")
		}

		for _, c := range ord.EtcdRaft.GetConsenters() {
			if c.Host == "" {
				logger.Panicf("consenter info in %s configuration did not specify host", etcdraft.TypeKey)
			}
			if c.Port == 0 {
				logger.Panicf("consenter info in %s configuration did not specify port", etcdraft.TypeKey)
			}
			if c.ClientTlsCert == nil {
				logger.Panicf("consenter info in %s configuration did not specify client TLS cert", etcdraft.TypeKey)
			}
			if c.ServerTlsCert == nil {
				logger.Panicf("consenter info in %s configuration did not specify server TLS cert", etcdraft.TypeKey)
			}
			clientCertPath := string(c.GetClientTlsCert())
			cf.TranslatePathInPlace(configDir, &clientCertPath)
			c.ClientTlsCert = []byte(clientCertPath)
			serverCertPath := string(c.GetServerTlsCert())
			cf.TranslatePathInPlace(configDir, &serverCertPath)
			c.ServerTlsCert = []byte(serverCertPath)
		}
	default:
		logger.Panicf("unknown orderer type: %s", ord.OrdererType)
	}
}

func translatePaths(configDir string, org *Organization) {
	cf.TranslatePathInPlace(configDir, &org.MSPDir)
}
