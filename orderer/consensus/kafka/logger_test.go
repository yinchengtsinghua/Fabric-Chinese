
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
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/onsi/gomega/gbytes"
	"github.com/stretchr/testify/assert"
)

func TestLoggerInit(t *testing.T) {
	assert.IsType(t, &saramaLoggerImpl{}, sarama.Logger, "Sarama logger (sarama.Logger) is not properly initialized.")
	assert.NotNil(t, saramaLogger, "Event logger (saramaLogger) is not properly initialized, it's Nil.")
	assert.Equal(t, sarama.Logger, saramaLogger, "Sarama logger (sarama.Logger) and Event logger (saramaLogger) should be the same.")
}

func TestEventLogger(t *testing.T) {
	eventMessage := "this message contains an interesting string within"
	substr := "interesting string"

//注册侦听器
	eventChan := saramaLogger.NewListener(substr)

//注册第二个侦听器（同一订阅）
	eventChan2 := saramaLogger.NewListener(substr)
	defer saramaLogger.RemoveListener(substr, eventChan2)

//调用记录器
	saramaLogger.Print("this message is not interesting")
	saramaLogger.Print(eventMessage)

	select {
//预期第一个侦听器的事件
	case receivedEvent := <-eventChan:
		assert.Equal(t, eventMessage, receivedEvent, "")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected event on eventChan")
	}

//Sesond Listener的Expect事件
	select {
	case receivedEvent := <-eventChan2:
		assert.Equal(t, eventMessage, receivedEvent, "")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected event on eventChan2")
	}

//注销第一个侦听器
	saramaLogger.RemoveListener(substr, eventChan)

//调用记录器
	saramaLogger.Print(eventMessage)

//不期望第一个侦听器发生任何事件
	select {
	case <-eventChan:
		t.Fatal("did not expect an event")
	case <-time.After(10 * time.Millisecond):
	}

//Sesond Listener的Expect事件
	select {
	case receivedEvent := <-eventChan2:
		assert.Equal(t, eventMessage, receivedEvent, "")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected event on eventChan2")
	}

}

func TestEventListener(t *testing.T) {
	topic := channelNameForTest(t)
	partition := int32(0)

	subscription := fmt.Sprintf("added subscription to %s/%d", topic, partition)
	listenerChan := saramaLogger.NewListener(subscription)
	defer saramaLogger.RemoveListener(subscription, listenerChan)

	go func() {
		event := <-listenerChan
		t.Logf("GOT: %s", event)
	}()

	broker := sarama.NewMockBroker(t, 500)
	defer broker.Close()

	config := sarama.NewConfig()
	config.ClientID = t.Name()
	config.Metadata.Retry.Max = 0
	config.Metadata.Retry.Backoff = 250 * time.Millisecond
	config.Net.ReadTimeout = 100 * time.Millisecond

	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(topic, partition, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(topic, partition, sarama.OffsetNewest, 1000).
			SetOffset(topic, partition, sarama.OffsetOldest, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(topic, partition, 0, sarama.StringEncoder("MSG 00")).
			SetMessage(topic, partition, 1, sarama.StringEncoder("MSG 01")).
			SetMessage(topic, partition, 2, sarama.StringEncoder("MSG 02")).
			SetMessage(topic, partition, 3, sarama.StringEncoder("MSG 03")),
	})

	consumer, err := sarama.NewConsumer([]string{broker.Addr()}, config)
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer partitionConsumer.Close()

	for i := 0; i < 3; i++ {
		select {
		case <-partitionConsumer.Messages():
		case <-time.After(shortTimeout):
			t.Fatalf("timed out waiting for messages (receieved %d messages)", i)
		}
	}
}

func TestLogPossibleKafkaVersionMismatch(t *testing.T) {
	topic := channelNameForTest(t)
	partition := int32(0)

	buf := gbytes.NewBuffer()
	flogging.Global.SetWriter(buf)
	defer flogging.Global.SetWriter(os.Stderr)

	broker := sarama.NewMockBroker(t, 500)
	defer broker.Close()

	config := sarama.NewConfig()
	config.ClientID = t.Name()
	config.Metadata.Retry.Max = 0
	config.Metadata.Retry.Backoff = 250 * time.Millisecond
	config.Net.ReadTimeout = 100 * time.Millisecond
	config.Version = sarama.V0_10_0_0

	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(topic, partition, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(topic, partition, sarama.OffsetNewest, 1000).
			SetOffset(topic, partition, sarama.OffsetOldest, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(topic, partition, 0, sarama.StringEncoder("MSG 00")),
	})

	consumer, err := sarama.NewConsumer([]string{broker.Addr()}, config)
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer partitionConsumer.Close()

	select {
	case <-partitionConsumer.Messages():
		t.Fatalf("did not expect to receive message")
	case <-time.After(shortTimeout):
		t.Logf("buffer:\n%s", buf.Contents())
		assert.Contains(t, string(buf.Contents()), "Kafka.Version specified in the orderer configuration is incorrectly set")
	}
}
