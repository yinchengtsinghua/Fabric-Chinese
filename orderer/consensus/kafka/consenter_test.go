
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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	localconfig "github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/orderer/consensus/kafka/mock"
	mockmultichannel "github.com/hyperledger/fabric/orderer/mocks/common/multichannel"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"

	"github.com/Shopify/sarama"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

var mockRetryOptions = localconfig.Retry{
	ShortInterval: 50 * time.Millisecond,
	ShortTotal:    100 * time.Millisecond,
	LongInterval:  60 * time.Millisecond,
	LongTotal:     120 * time.Millisecond,
	NetworkTimeouts: localconfig.NetworkTimeouts{
		DialTimeout:  40 * time.Millisecond,
		ReadTimeout:  40 * time.Millisecond,
		WriteTimeout: 40 * time.Millisecond,
	},
	Metadata: localconfig.Metadata{
		RetryMax:     2,
		RetryBackoff: 40 * time.Millisecond,
	},
	Producer: localconfig.Producer{
		RetryMax:     2,
		RetryBackoff: 40 * time.Millisecond,
	},
	Consumer: localconfig.Consumer{
		RetryBackoff: 40 * time.Millisecond,
	},
}

func init() {
	mockLocalConfig = newMockLocalConfig(
		false,
		localconfig.SASLPlain{Enabled: false},
		mockRetryOptions,
		false)
	mockBrokerConfig = newMockBrokerConfig(
		mockLocalConfig.General.TLS,
		mockLocalConfig.Kafka.SASLPlain,
		mockLocalConfig.Kafka.Retry,
		mockLocalConfig.Kafka.Version,
		defaultPartition)
	mockConsenter = newMockConsenter(
		mockBrokerConfig,
		mockLocalConfig.General.TLS,
		mockLocalConfig.Kafka.Retry,
		mockLocalConfig.Kafka.Version)
	setupTestLogging("ERROR")
}

func TestNew(t *testing.T) {
	c, _ := New(mockLocalConfig.Kafka, &mock.MetricsProvider{})
	_ = consensus.Consenter(c)
}

func TestHandleChain(t *testing.T) {
	consenter, _ := New(mockLocalConfig.Kafka, &mock.MetricsProvider{})

	oldestOffset := int64(0)
	newestOffset := int64(5)
	message := sarama.StringEncoder("messageFoo")

	mockChannel := newChannel(channelNameForTest(t), defaultPartition)

	mockBroker := sarama.NewMockBroker(t, 0)
	mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
			SetLeader(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID()),
		"ProduceRequest": sarama.NewMockProduceResponse(t).
			SetError(mockChannel.topic(), mockChannel.partition(), sarama.ErrNoError),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetOldest, oldestOffset).
			SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetNewest, newestOffset),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(mockChannel.topic(), mockChannel.partition(), newestOffset, message),
	})

	mockSupport := &mockmultichannel.ConsenterSupport{
		ChainIDVal: mockChannel.topic(),
		SharedConfigVal: &mockconfig.Orderer{
			KafkaBrokersVal: []string{mockBroker.Addr()},
		},
	}

	mockMetadata := &cb.Metadata{Value: utils.MarshalOrPanic(&ab.KafkaMetadata{LastOffsetPersisted: newestOffset - 1})}
	_, err := consenter.HandleChain(mockSupport, mockMetadata)
	assert.NoError(t, err, "Expected the HandleChain call to return without errors")
}

//此处定义的测试助手函数和模拟对象

var mockConsenter commonConsenter
var mockLocalConfig *localconfig.TopLevel
var mockBrokerConfig *sarama.Config

func extractEncodedOffset(marshalledOrdererMetadata []byte) int64 {
	omd := &cb.Metadata{}
	_ = proto.Unmarshal(marshalledOrdererMetadata, omd)
	kmd := &ab.KafkaMetadata{}
	_ = proto.Unmarshal(omd.GetValue(), kmd)
	return kmd.LastOffsetPersisted
}

func newMockBrokerConfig(
	tlsConfig localconfig.TLS,
	saslPlain localconfig.SASLPlain,
	retryOptions localconfig.Retry,
	kafkaVersion sarama.KafkaVersion,
	chosenStaticPartition int32) *sarama.Config {

	brokerConfig := newBrokerConfig(
		tlsConfig,
		saslPlain,
		retryOptions,
		kafkaVersion,
		chosenStaticPartition)
	brokerConfig.ClientID = "test"
	return brokerConfig
}

func newMockConsenter(brokerConfig *sarama.Config, tlsConfig localconfig.TLS, retryOptions localconfig.Retry, kafkaVersion sarama.KafkaVersion) *consenterImpl {
	return &consenterImpl{
		brokerConfigVal: brokerConfig,
		tlsConfigVal:    tlsConfig,
		retryOptionsVal: retryOptions,
		kafkaVersionVal: kafkaVersion,
	}
}

func newMockConsumerMessage(wrappedMessage *ab.KafkaMessage) *sarama.ConsumerMessage {
	return &sarama.ConsumerMessage{
		Value: sarama.ByteEncoder(utils.MarshalOrPanic(wrappedMessage)),
	}
}

func newMockEnvelope(content string) *cb.Envelope {
	return &cb.Envelope{Payload: utils.MarshalOrPanic(&cb.Payload{
		Header: &cb.Header{ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{ChannelId: "foo"})},
		Data:   []byte(content),
	})}
}

func newMockLocalConfig(
	enableTLS bool,
	saslPlain localconfig.SASLPlain,
	retryOptions localconfig.Retry,
	verboseLog bool) *localconfig.TopLevel {

	return &localconfig.TopLevel{
		General: localconfig.General{
			TLS: localconfig.TLS{
				Enabled: enableTLS,
			},
		},
		Kafka: localconfig.Kafka{
			TLS: localconfig.TLS{
				Enabled: enableTLS,
			},
			SASLPlain: saslPlain,
			Retry:     retryOptions,
			Verbose:   verboseLog,
Version:   sarama.V0_9_0_1, //sarama.mockbroker仅生成与版本<0.10兼容的消息
		},
	}
}

func setupTestLogging(logLevel string) {
//此调用允许我们（a）获取日志后端初始化
//发生在“鞭打”包中，并且（b）调整
//在此包上运行测试时的日志。
	spec := fmt.Sprintf("orderer.consensus.kafka=%s", logLevel)
	flogging.ActivateSpec(spec)
}

func tamperBytes(original []byte) []byte {
	byteCount := len(original)
	return original[:byteCount-1]
}

func channelNameForTest(t *testing.T) string {
	return fmt.Sprintf("%s.channel", strings.Replace(strings.ToLower(t.Name()), "/", ".", -1))
}
