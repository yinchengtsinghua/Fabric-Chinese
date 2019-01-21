
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
	"crypto/tls"
	"crypto/x509"

	localconfig "github.com/hyperledger/fabric/orderer/common/localconfig"

	"github.com/Shopify/sarama"
)

func newBrokerConfig(
	tlsConfig localconfig.TLS,
	saslPlain localconfig.SASLPlain,
	retryOptions localconfig.Retry,
	kafkaVersion sarama.KafkaVersion,
	chosenStaticPartition int32) *sarama.Config {

//以字节为单位设置的请求头等的最大大小。故意太大。
	paddingDelta := 1 * 1024 * 1024

	brokerConfig := sarama.NewConfig()

	brokerConfig.Consumer.Retry.Backoff = retryOptions.Consumer.RetryBackoff

//允许我们检索使用通道时发生的错误
	brokerConfig.Consumer.Return.Errors = true

	brokerConfig.Metadata.Retry.Backoff = retryOptions.Metadata.RetryBackoff
	brokerConfig.Metadata.Retry.Max = retryOptions.Metadata.RetryMax

	brokerConfig.Net.DialTimeout = retryOptions.NetworkTimeouts.DialTimeout
	brokerConfig.Net.ReadTimeout = retryOptions.NetworkTimeouts.ReadTimeout
	brokerConfig.Net.WriteTimeout = retryOptions.NetworkTimeouts.WriteTimeout

	brokerConfig.Net.TLS.Enable = tlsConfig.Enabled
	if brokerConfig.Net.TLS.Enable {
//创建公钥/私钥对结构
		keyPair, err := tls.X509KeyPair([]byte(tlsConfig.Certificate), []byte(tlsConfig.PrivateKey))
		if err != nil {
			logger.Panic("Unable to decode public/private key pair:", err)
		}
//创建根CA池
		rootCAs := x509.NewCertPool()
		for _, certificate := range tlsConfig.RootCAs {
			if !rootCAs.AppendCertsFromPEM([]byte(certificate)) {
				logger.Panic("Unable to parse the root certificate authority certificates (Kafka.Tls.RootCAs)")
			}
		}
		brokerConfig.Net.TLS.Config = &tls.Config{
			Certificates: []tls.Certificate{keyPair},
			RootCAs:      rootCAs,
			MinVersion:   tls.VersionTLS12,
MaxVersion:   0, //最新支持的TLS版本
		}
	}
	brokerConfig.Net.SASL.Enable = saslPlain.Enabled
	if brokerConfig.Net.SASL.Enable {
		brokerConfig.Net.SASL.User = saslPlain.User
		brokerConfig.Net.SASL.Password = saslPlain.Password
	}

//将相当于kafka producer config max.request.bytes的值设置为默认值
//kafka代理的socket.request.max.bytes属性的值（100 mib）。
	brokerConfig.Producer.MaxMessageBytes = int(sarama.MaxRequestSize) - paddingDelta

	brokerConfig.Producer.Retry.Backoff = retryOptions.Producer.RetryBackoff
	brokerConfig.Producer.Retry.Max = retryOptions.Producer.RetryMax

//我们现在做事的方式实际上不需要一个分裂者，
//但我们现在添加它是为了在未来提供灵活性。
	brokerConfig.Producer.Partitioner = newStaticPartitioner(chosenStaticPartition)
//设置代理所需的确认可靠性级别。
//waitforall意味着分区领导将等待所有ISR
//在向发送方发送ACK之前的消息。
	brokerConfig.Producer.RequiredAcks = sarama.WaitForAll
//Sarama图书馆要求的深奥设置，见：
//https://github.com/shopify/sarama/issues/816
	brokerConfig.Producer.Return.Successes = true

	brokerConfig.Version = kafkaVersion

	return brokerConfig
}
