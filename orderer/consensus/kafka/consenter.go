
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


package kafka

import (
	"github.com/hyperledger/fabric/common/metrics"
	localconfig "github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/consensus"
	cb "github.com/hyperledger/fabric/protos/common"

	"github.com/Shopify/sarama"
	logging "github.com/op/go-logging"
)

//New创建了一个基于Kafka的同意者。由订购方的main.go调用。
func New(config localconfig.Kafka, metricsProvider metrics.Provider) (consensus.Consenter, *Metrics) {
	if config.Verbose {
		logging.SetLevel(logging.DEBUG, "orderer.consensus.kafka.sarama")
	}

	brokerConfig := newBrokerConfig(
		config.TLS,
		config.SASLPlain,
		config.Retry,
		config.Version,
		defaultPartition)

	return &consenterImpl{
		brokerConfigVal: brokerConfig,
		tlsConfigVal:    config.TLS,
		retryOptionsVal: config.Retry,
		kafkaVersionVal: config.Version,
		topicDetailVal: &sarama.TopicDetail{
			NumPartitions:     1,
			ReplicationFactor: config.Topic.ReplicationFactor,
		},
	}, NewMetrics(metricsProvider, brokerConfig.MetricRegistry)
}

//ConsenterImpl持有满足
//协商一致。同意人界面——根据handlechain合同的要求——以及
//共同同意者一。
type consenterImpl struct {
	brokerConfigVal *sarama.Config
	tlsConfigVal    localconfig.TLS
	retryOptionsVal localconfig.Retry
	kafkaVersionVal sarama.KafkaVersion
	topicDetailVal  *sarama.TopicDetail
	metricsProvider metrics.Provider
}

//handlechain创建/返回对
//给定的一组支持资源。执行协商一致。同意人
//接口。由共识调用。newChainSupport（），它本身由
//在分类帐目录中查找时使用multichannel.newManagerImpl（）。
//存在的枷锁。
func (consenter *consenterImpl) HandleChain(support consensus.ConsenterSupport, metadata *cb.Metadata) (consensus.Chain, error) {
	lastOffsetPersisted, lastOriginalOffsetProcessed, lastResubmittedConfigOffset := getOffsets(metadata.Value, support.ChainID())
	return newChain(consenter, support, lastOffsetPersisted, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)
}

//CommonConsent允许我们检索
//同意人反对。这些将在所有由
//这个同意者。它们使用本地配置设置进行设置。这个
//同意模板满足接口要求。
type commonConsenter interface {
	brokerConfig() *sarama.Config
	retryOptions() localconfig.Retry
	topicDetail() *sarama.TopicDetail
}

func (consenter *consenterImpl) brokerConfig() *sarama.Config {
	return consenter.brokerConfigVal
}

func (consenter *consenterImpl) retryOptions() localconfig.Retry {
	return consenter.retryOptionsVal
}

func (consenter *consenterImpl) topicDetail() *sarama.TopicDetail {
	return consenter.topicDetailVal
}

//Closeable允许关闭调用资源。
type closeable interface {
	close() error
}
