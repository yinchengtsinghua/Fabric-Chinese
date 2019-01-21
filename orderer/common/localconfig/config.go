
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有IBM公司。保留所有权利。
//SPDX许可证标识符：Apache-2.0

package localconfig

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	bccsp "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/viperutil"
	coreconfig "github.com/hyperledger/fabric/core/config"
	"github.com/spf13/viper"
)

//环境变量的前缀。
const Prefix = "ORDERER"

var logger = flogging.MustGetLogger("localconfig")

//顶层直接对应于订购者配置yaml。
//注意，对于非1-1映射，可以附加
//类似于'mapstructure:“weirdformat”`to
//修改默认映射，请参见“取消标记”
//更多信息，请访问https://github.com/spf13/viper。
type TopLevel struct {
	General    General
	FileLedger FileLedger
	RAMLedger  RAMLedger
	Kafka      Kafka
	Debug      Debug
	Consensus  interface{}
	Operations Operations
	Metrics    Metrics
}

//General包含在所有排序器类型中应该是通用的配置。
type General struct {
	LedgerType     string
	ListenAddress  string
	ListenPort     uint16
	TLS            TLS
	Cluster        Cluster
	Keepalive      Keepalive
	GenesisMethod  string
	GenesisProfile string
	SystemChannel  string
	GenesisFile    string
	Profile        Profile
	LocalMSPDir    string
	LocalMSPID     string
	BCCSP          *bccsp.FactoryOpts
	Authentication Authentication
}

type Cluster struct {
	RootCAs                 []string
	ClientCertificate       string
	ClientPrivateKey        string
	DialTimeout             time.Duration
	RPCTimeout              time.Duration
	ReplicationBufferSize   int
	ReplicationPullTimeout  time.Duration
	ReplicationRetryTimeout time.Duration
}

//keepalive包含GRPC服务器的配置。
type Keepalive struct {
	ServerMinInterval time.Duration
	ServerInterval    time.Duration
	ServerTimeout     time.Duration
}

//TLS包含TLS连接的配置。
type TLS struct {
	Enabled            bool
	PrivateKey         string
	Certificate        string
	RootCAs            []string
	ClientAuthRequired bool
	ClientRootCAs      []string
}

//sasl plain包含sasl/plain身份验证的配置
type SASLPlain struct {
	Enabled  bool
	User     string
	Password string
}

//身份验证包含与身份验证相关的配置参数
//客户端消息。
type Authentication struct {
	TimeWindow time.Duration
}

//配置文件包含Go-PProf配置文件的配置。
type Profile struct {
	Enabled bool
	Address string
}

//file ledger包含基于文件的分类帐的配置。
type FileLedger struct {
	Location string
	Prefix   string
}

//ram ledger包含RAM分类账的配置。
type RAMLedger struct {
	HistorySize uint
}

//Kafka包含基于Kafka的医嘱者的配置。
type Kafka struct {
	Retry     Retry
	Verbose   bool
Version   sarama.KafkaVersion //TODO将此移动到全局配置
	TLS       TLS
	SASLPlain SASLPlain
	Topic     Topic
}

//重试包含与重试和超时相关的配置，当
//无法建立到Kafka群集的连接，或者当元数据
//需要重复请求（因为集群位于
//领袖选举）。
type Retry struct {
	ShortInterval   time.Duration
	ShortTotal      time.Duration
	LongInterval    time.Duration
	LongTotal       time.Duration
	NetworkTimeouts NetworkTimeouts
	Metadata        Metadata
	Producer        Producer
	Consumer        Consumer
}

//NetworkTimeouts包含网络请求的套接字超时
//卡夫卡簇。
type NetworkTimeouts struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

//元数据包含对Kafka的元数据请求的配置
//集群。
type Metadata struct {
	RetryMax     int
	RetryBackoff time.Duration
}

//Producer包含失败时Producer重试的配置
//将消息发送到Kafka分区。
type Producer struct {
	RetryMax     int
	RetryBackoff time.Duration
}

//使用者包含使用者在失败时重试的配置
//从kafa分区读取。
type Consumer struct {
	RetryBackoff time.Duration
}

//主题包含创建卡夫卡主题时要使用的设置
type Topic struct {
	ReplicationFactor int16
}

//调试包含订购方调试参数的配置。
type Debug struct {
	BroadcastTraceDir string
	DeliverTraceDir   string
}

//操作为医嘱者配置操作endpont。
type Operations struct {
	ListenAddress string
	TLS           TLS
}

//操作配置医嘱者的度量提供程序。
type Metrics struct {
	Provider string
	Statsd   Statsd
}

//statsd提供从订购方发出statsd度量所需的配置。
type Statsd struct {
	Network       string
	Address       string
	WriteInterval time.Duration
	Prefix        string
}

//默认值包含默认的订购者配置值。
var Defaults = TopLevel{
	General: General{
		LedgerType:     "file",
		ListenAddress:  "127.0.0.1",
		ListenPort:     7050,
		GenesisMethod:  "provisional",
		GenesisProfile: "SampleSingleMSPSolo",
		SystemChannel:  "test-system-channel-name",
		GenesisFile:    "genesisblock",
		Profile: Profile{
			Enabled: false,
			Address: "0.0.0.0:6060",
		},
		LocalMSPDir: "msp",
		LocalMSPID:  "SampleOrg",
		BCCSP:       bccsp.GetDefaultOpts(),
		Authentication: Authentication{
			TimeWindow: time.Duration(15 * time.Minute),
		},
	},
	RAMLedger: RAMLedger{
		HistorySize: 10000,
	},
	FileLedger: FileLedger{
		Location: "/var/hyperledger/production/orderer",
		Prefix:   "hyperledger-fabric-ordererledger",
	},
	Kafka: Kafka{
		Retry: Retry{
			ShortInterval: 1 * time.Minute,
			ShortTotal:    10 * time.Minute,
			LongInterval:  10 * time.Minute,
			LongTotal:     12 * time.Hour,
			NetworkTimeouts: NetworkTimeouts{
				DialTimeout:  30 * time.Second,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			Metadata: Metadata{
				RetryBackoff: 250 * time.Millisecond,
				RetryMax:     3,
			},
			Producer: Producer{
				RetryBackoff: 100 * time.Millisecond,
				RetryMax:     3,
			},
			Consumer: Consumer{
				RetryBackoff: 2 * time.Second,
			},
		},
		Verbose: false,
		Version: sarama.V0_10_2_0,
		TLS: TLS{
			Enabled: false,
		},
		Topic: Topic{
			ReplicationFactor: 3,
		},
	},
	Debug: Debug{
		BroadcastTraceDir: "",
		DeliverTraceDir:   "",
	},
	Operations: Operations{
		ListenAddress: "127.0.0.1:0",
	},
	Metrics: Metrics{
		Provider: "disabled",
	},
}

//加载解析排序器yaml文件和环境，生成
//适用于配置使用的结构，失败时返回错误。
func Load() (*TopLevel, error) {
	config := viper.New()
	coreconfig.InitViper(config, "orderer")
	config.SetEnvPrefix(Prefix)
	config.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	config.SetEnvKeyReplacer(replacer)

	if err := config.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error reading configuration: %s", err)
	}

	var uconf TopLevel
	if err := viperutil.EnhancedExactUnmarshal(config, &uconf); err != nil {
		return nil, fmt.Errorf("Error unmarshaling config into struct: %s", err)
	}

	uconf.completeInitialization(filepath.Dir(config.ConfigFileUsed()))
	return &uconf, nil
}

func (c *TopLevel) completeInitialization(configDir string) {
	defer func() {
//为群集TLS配置转换任何路径（如果适用）
		if c.General.Cluster.ClientPrivateKey != "" {
			coreconfig.TranslatePathInPlace(configDir, &c.General.Cluster.ClientPrivateKey)
		}
		if c.General.Cluster.ClientCertificate != "" {
			coreconfig.TranslatePathInPlace(configDir, &c.General.Cluster.ClientCertificate)
		}
		c.General.Cluster.RootCAs = translateCAs(configDir, c.General.Cluster.RootCAs)
//转换常规TLS配置的任何路径
		c.General.TLS.RootCAs = translateCAs(configDir, c.General.TLS.RootCAs)
		c.General.TLS.ClientRootCAs = translateCAs(configDir, c.General.TLS.ClientRootCAs)
		coreconfig.TranslatePathInPlace(configDir, &c.General.TLS.PrivateKey)
		coreconfig.TranslatePathInPlace(configDir, &c.General.TLS.Certificate)
		coreconfig.TranslatePathInPlace(configDir, &c.General.GenesisFile)
		coreconfig.TranslatePathInPlace(configDir, &c.General.LocalMSPDir)
	}()

	for {
		switch {
		case c.General.LedgerType == "":
			logger.Infof("General.LedgerType unset, setting to %s", Defaults.General.LedgerType)
			c.General.LedgerType = Defaults.General.LedgerType

		case c.General.ListenAddress == "":
			logger.Infof("General.ListenAddress unset, setting to %s", Defaults.General.ListenAddress)
			c.General.ListenAddress = Defaults.General.ListenAddress
		case c.General.ListenPort == 0:
			logger.Infof("General.ListenPort unset, setting to %v", Defaults.General.ListenPort)
			c.General.ListenPort = Defaults.General.ListenPort

		case c.General.GenesisMethod == "":
			c.General.GenesisMethod = Defaults.General.GenesisMethod
		case c.General.GenesisFile == "":
			c.General.GenesisFile = Defaults.General.GenesisFile
		case c.General.GenesisProfile == "":
			c.General.GenesisProfile = Defaults.General.GenesisProfile
		case c.General.SystemChannel == "":
			c.General.SystemChannel = Defaults.General.SystemChannel

		case c.Kafka.TLS.Enabled && c.Kafka.TLS.Certificate == "":
			logger.Panicf("General.Kafka.TLS.Certificate must be set if General.Kafka.TLS.Enabled is set to true.")
		case c.Kafka.TLS.Enabled && c.Kafka.TLS.PrivateKey == "":
			logger.Panicf("General.Kafka.TLS.PrivateKey must be set if General.Kafka.TLS.Enabled is set to true.")
		case c.Kafka.TLS.Enabled && c.Kafka.TLS.RootCAs == nil:
			logger.Panicf("General.Kafka.TLS.CertificatePool must be set if General.Kafka.TLS.Enabled is set to true.")

		case c.Kafka.SASLPlain.Enabled && c.Kafka.SASLPlain.User == "":
			logger.Panic("General.Kafka.SASLPlain.User must be set if General.Kafka.SASLPlain.Enabled is set to true.")
		case c.Kafka.SASLPlain.Enabled && c.Kafka.SASLPlain.Password == "":
			logger.Panic("General.Kafka.SASLPlain.Password must be set if General.Kafka.SASLPlain.Enabled is set to true.")

		case c.General.Profile.Enabled && c.General.Profile.Address == "":
			logger.Infof("Profiling enabled and General.Profile.Address unset, setting to %s", Defaults.General.Profile.Address)
			c.General.Profile.Address = Defaults.General.Profile.Address

		case c.General.LocalMSPDir == "":
			logger.Infof("General.LocalMSPDir unset, setting to %s", Defaults.General.LocalMSPDir)
			c.General.LocalMSPDir = Defaults.General.LocalMSPDir
		case c.General.LocalMSPID == "":
			logger.Infof("General.LocalMSPID unset, setting to %s", Defaults.General.LocalMSPID)
			c.General.LocalMSPID = Defaults.General.LocalMSPID

		case c.General.Authentication.TimeWindow == 0:
			logger.Infof("General.Authentication.TimeWindow unset, setting to %s", Defaults.General.Authentication.TimeWindow)
			c.General.Authentication.TimeWindow = Defaults.General.Authentication.TimeWindow

		case c.FileLedger.Prefix == "":
			logger.Infof("FileLedger.Prefix unset, setting to %s", Defaults.FileLedger.Prefix)
			c.FileLedger.Prefix = Defaults.FileLedger.Prefix

		case c.Kafka.Retry.ShortInterval == 0:
			logger.Infof("Kafka.Retry.ShortInterval unset, setting to %v", Defaults.Kafka.Retry.ShortInterval)
			c.Kafka.Retry.ShortInterval = Defaults.Kafka.Retry.ShortInterval
		case c.Kafka.Retry.ShortTotal == 0:
			logger.Infof("Kafka.Retry.ShortTotal unset, setting to %v", Defaults.Kafka.Retry.ShortTotal)
			c.Kafka.Retry.ShortTotal = Defaults.Kafka.Retry.ShortTotal
		case c.Kafka.Retry.LongInterval == 0:
			logger.Infof("Kafka.Retry.LongInterval unset, setting to %v", Defaults.Kafka.Retry.LongInterval)
			c.Kafka.Retry.LongInterval = Defaults.Kafka.Retry.LongInterval
		case c.Kafka.Retry.LongTotal == 0:
			logger.Infof("Kafka.Retry.LongTotal unset, setting to %v", Defaults.Kafka.Retry.LongTotal)
			c.Kafka.Retry.LongTotal = Defaults.Kafka.Retry.LongTotal

		case c.Kafka.Retry.NetworkTimeouts.DialTimeout == 0:
			logger.Infof("Kafka.Retry.NetworkTimeouts.DialTimeout unset, setting to %v", Defaults.Kafka.Retry.NetworkTimeouts.DialTimeout)
			c.Kafka.Retry.NetworkTimeouts.DialTimeout = Defaults.Kafka.Retry.NetworkTimeouts.DialTimeout
		case c.Kafka.Retry.NetworkTimeouts.ReadTimeout == 0:
			logger.Infof("Kafka.Retry.NetworkTimeouts.ReadTimeout unset, setting to %v", Defaults.Kafka.Retry.NetworkTimeouts.ReadTimeout)
			c.Kafka.Retry.NetworkTimeouts.ReadTimeout = Defaults.Kafka.Retry.NetworkTimeouts.ReadTimeout
		case c.Kafka.Retry.NetworkTimeouts.WriteTimeout == 0:
			logger.Infof("Kafka.Retry.NetworkTimeouts.WriteTimeout unset, setting to %v", Defaults.Kafka.Retry.NetworkTimeouts.WriteTimeout)
			c.Kafka.Retry.NetworkTimeouts.WriteTimeout = Defaults.Kafka.Retry.NetworkTimeouts.WriteTimeout

		case c.Kafka.Retry.Metadata.RetryBackoff == 0:
			logger.Infof("Kafka.Retry.Metadata.RetryBackoff unset, setting to %v", Defaults.Kafka.Retry.Metadata.RetryBackoff)
			c.Kafka.Retry.Metadata.RetryBackoff = Defaults.Kafka.Retry.Metadata.RetryBackoff
		case c.Kafka.Retry.Metadata.RetryMax == 0:
			logger.Infof("Kafka.Retry.Metadata.RetryMax unset, setting to %v", Defaults.Kafka.Retry.Metadata.RetryMax)
			c.Kafka.Retry.Metadata.RetryMax = Defaults.Kafka.Retry.Metadata.RetryMax

		case c.Kafka.Retry.Producer.RetryBackoff == 0:
			logger.Infof("Kafka.Retry.Producer.RetryBackoff unset, setting to %v", Defaults.Kafka.Retry.Producer.RetryBackoff)
			c.Kafka.Retry.Producer.RetryBackoff = Defaults.Kafka.Retry.Producer.RetryBackoff
		case c.Kafka.Retry.Producer.RetryMax == 0:
			logger.Infof("Kafka.Retry.Producer.RetryMax unset, setting to %v", Defaults.Kafka.Retry.Producer.RetryMax)
			c.Kafka.Retry.Producer.RetryMax = Defaults.Kafka.Retry.Producer.RetryMax

		case c.Kafka.Retry.Consumer.RetryBackoff == 0:
			logger.Infof("Kafka.Retry.Consumer.RetryBackoff unset, setting to %v", Defaults.Kafka.Retry.Consumer.RetryBackoff)
			c.Kafka.Retry.Consumer.RetryBackoff = Defaults.Kafka.Retry.Consumer.RetryBackoff

		case c.Kafka.Version == sarama.KafkaVersion{}:
			logger.Infof("Kafka.Version unset, setting to %v", Defaults.Kafka.Version)
			c.Kafka.Version = Defaults.Kafka.Version

		default:
			return
		}
	}
}

func translateCAs(configDir string, certificateAuthorities []string) []string {
	var results []string
	for _, ca := range certificateAuthorities {
		result := coreconfig.TranslatePath(configDir, ca)
		results = append(results, result)
	}
	return results
}
