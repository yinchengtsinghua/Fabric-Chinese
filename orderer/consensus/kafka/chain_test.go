
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
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/channelconfig"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/orderer/common/blockcutter"
	"github.com/hyperledger/fabric/orderer/common/msgprocessor"
	lmock "github.com/hyperledger/fabric/orderer/consensus/kafka/mock"
	mockblockcutter "github.com/hyperledger/fabric/orderer/mocks/common/blockcutter"
	mockmultichannel "github.com/hyperledger/fabric/orderer/mocks/common/multichannel"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	extraShortTimeout = 1 * time.Millisecond
	shortTimeout      = 1 * time.Second
	longTimeout       = 1 * time.Hour

	hitBranch = 50 * time.Millisecond
)

func TestChain(t *testing.T) {

	oldestOffset := int64(0)
	newestOffset := int64(5)
	lastOriginalOffsetProcessed := int64(0)
	lastResubmittedConfigOffset := int64(0)

	message := sarama.StringEncoder("messageFoo")

	newMocks := func(t *testing.T) (mockChannel channel, mockBroker *sarama.MockBroker, mockSupport *mockmultichannel.ConsenterSupport) {
		mockChannel = newChannel(channelNameForTest(t), defaultPartition)
		mockBroker = sarama.NewMockBroker(t, 0)
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
		mockSupport = &mockmultichannel.ConsenterSupport{
			ChainIDVal:      mockChannel.topic(),
			HeightVal:       uint64(3),
			SharedConfigVal: &mockconfig.Orderer{KafkaBrokersVal: []string{mockBroker.Addr()}},
		}
		return
	}

	t.Run("New", func(t *testing.T) {
		_, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, err := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

		assert.NoError(t, err, "Expected newChain to return without errors")
		select {
		case <-chain.Errored():
			logger.Debug("Errored() returned a closed channel as expected")
		default:
			t.Fatal("Errored() should have returned a closed channel")
		}

		select {
		case <-chain.haltChan:
			t.Fatal("haltChan should have been open")
		default:
			logger.Debug("haltChan is open as it should be")
		}

		select {
		case <-chain.startChan:
			t.Fatal("startChan should have been open")
		default:
			logger.Debug("startChan is open as it should be")
		}
	})

	t.Run("Start", func(t *testing.T) {
		_, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
//设置为-1，因为我们尚未发送连接消息
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

		chain.Start()
		select {
		case <-chain.startChan:
			logger.Debug("startChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("startChan should have been closed by now")
		}

//触发processmessagestoblocks goroutine中的haltchan子句
		close(chain.haltChan)
	})

	t.Run("Halt", func(t *testing.T) {
		_, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

		chain.Start()
		select {
		case <-chain.startChan:
			logger.Debug("startChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("startChan should have been closed by now")
		}

//等待开始阶段完成，然后：
		chain.Halt()

		select {
		case <-chain.haltChan:
			logger.Debug("haltChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("haltChan should have been closed")
		}

		select {
		case <-chain.errorChan:
			logger.Debug("errorChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("errorChan should have been closed")
		}
	})

	t.Run("DoubleHalt", func(t *testing.T) {
		_, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

		chain.Start()
		select {
		case <-chain.startChan:
			logger.Debug("startChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("startChan should have been closed by now")
		}

		chain.Halt()

		assert.NotPanics(t, func() { chain.Halt() }, "Calling Halt() more than once shouldn't panic")
	})

	t.Run("StartWithProducerForChannelError", func(t *testing.T) {
		_, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
//指向空的经纪人列表
		mockSupportCopy := *mockSupport
		mockSupportCopy.SharedConfigVal = &mockconfig.Orderer{KafkaBrokersVal: []string{}}

		chain, _ := newChain(mockConsenter, &mockSupportCopy, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

//生产路径将实际调用chain.start（）。这是
//在功能上相当，并且允许我们在上面运行断言。
		assert.Panics(t, func() { startThread(chain) }, "Expected the Start() call to panic")
	})

	t.Run("StartWithConnectMessageError", func(t *testing.T) {
//请注意，此测试受以下参数影响：
//-net.readtimeout
//-consumer.retry.backoff
//-metadata.retry.max
		mockChannel, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

//让代理返回ErrNotLeaderForPartition错误
		mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
			"MetadataRequest": sarama.NewMockMetadataResponse(t).
				SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
				SetLeader(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID()),
			"ProduceRequest": sarama.NewMockProduceResponse(t).
				SetError(mockChannel.topic(), mockChannel.partition(), sarama.ErrNotLeaderForPartition),
			"OffsetRequest": sarama.NewMockOffsetResponse(t).
				SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetOldest, oldestOffset).
				SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetNewest, newestOffset),
			"FetchRequest": sarama.NewMockFetchResponse(t, 1).
				SetMessage(mockChannel.topic(), mockChannel.partition(), newestOffset, message),
		})

		assert.Panics(t, func() { startThread(chain) }, "Expected the Start() call to panic")
	})

	t.Run("enqueueIfNotStarted", func(t *testing.T) {
		mockChannel, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

//与StartWithConnectMessageError一样，让代理返回
//errnotLeaderForPartition错误，即导致
//“连接后消息”步骤。
		mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
			"MetadataRequest": sarama.NewMockMetadataResponse(t).
				SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
				SetLeader(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID()),
			"ProduceRequest": sarama.NewMockProduceResponse(t).
				SetError(mockChannel.topic(), mockChannel.partition(), sarama.ErrNotLeaderForPartition),
			"OffsetRequest": sarama.NewMockOffsetResponse(t).
				SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetOldest, oldestOffset).
				SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetNewest, newestOffset),
			"FetchRequest": sarama.NewMockFetchResponse(t, 1).
				SetMessage(mockChannel.topic(), mockChannel.partition(), newestOffset, message),
		})

//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
		assert.False(t, chain.enqueue(newRegularMessage([]byte("fooMessage"))), "Expected enqueue call to return false")
	})

	t.Run("StartWithConsumerForChannelError", func(t *testing.T) {
//请注意，此测试受以下参数影响：
//-net.readtimeout
//-consumer.retry.backoff
//-metadata.retry.max

		mockChannel, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()

//提供超出范围的偏移
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

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

		assert.Panics(t, func() { startThread(chain) }, "Expected the Start() call to panic")
	})

	t.Run("enqueueProper", func(t *testing.T) {
		mockChannel, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

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

		chain.Start()
		select {
		case <-chain.startChan:
			logger.Debug("startChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("startChan should have been closed by now")
		}

//队列应该可以访问post路径，并且它的produceRequest应该没有错误地通过。
//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
		assert.True(t, chain.enqueue(newRegularMessage([]byte("fooMessage"))), "Expected enqueue call to return true")

		chain.Halt()
	})

	t.Run("enqueueIfHalted", func(t *testing.T) {
		mockChannel, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

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

		chain.Start()
		select {
		case <-chain.startChan:
			logger.Debug("startChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("startChan should have been closed by now")
		}
		chain.Halt()

//Haltchan应关闭对post路径的访问。
//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
		assert.False(t, chain.enqueue(newRegularMessage([]byte("fooMessage"))), "Expected enqueue call to return false")
	})

	t.Run("enqueueError", func(t *testing.T) {
		mockChannel, mockBroker, mockSupport := newMocks(t)
		defer func() { mockBroker.Close() }()
		chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

//使用“良好”处理程序映射，该映射允许在没有
//问题
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

		chain.Start()
		select {
		case <-chain.startChan:
			logger.Debug("startChan is closed as it should be")
		case <-time.After(shortTimeout):
			t.Fatal("startChan should have been closed by now")
		}
		defer chain.Halt()

//现在进行此操作，以便下一个ProduceRequest遇到错误
		mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
			"ProduceRequest": sarama.NewMockProduceResponse(t).
				SetError(mockChannel.topic(), mockChannel.partition(), sarama.ErrNotLeaderForPartition),
		})

//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
		assert.False(t, chain.enqueue(newRegularMessage([]byte("fooMessage"))), "Expected enqueue call to return false")
	})

	t.Run("Order", func(t *testing.T) {
		t.Run("ErrorIfNotStarted", func(t *testing.T) {
			_, mockBroker, mockSupport := newMocks(t)
			defer func() { mockBroker.Close() }()
			chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
			assert.Error(t, chain.Order(&cb.Envelope{}, uint64(0)))
		})

		t.Run("Proper", func(t *testing.T) {
			mockChannel, mockBroker, mockSupport := newMocks(t)
			defer func() { mockBroker.Close() }()
			chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

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

			chain.Start()
			defer chain.Halt()

			select {
			case <-chain.startChan:
				logger.Debug("startChan is closed as it should be")
			case <-time.After(shortTimeout):
				t.Fatal("startChan should have been closed by now")
			}

//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
			assert.NoError(t, chain.Order(&cb.Envelope{}, uint64(0)), "Expect Order successfully")
		})
	})

	t.Run("Configure", func(t *testing.T) {
		t.Run("ErrorIfNotStarted", func(t *testing.T) {
			_, mockBroker, mockSupport := newMocks(t)
			defer func() { mockBroker.Close() }()
			chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
			assert.Error(t, chain.Configure(&cb.Envelope{}, uint64(0)))
		})

		t.Run("Proper", func(t *testing.T) {
			mockChannel, mockBroker, mockSupport := newMocks(t)
			defer func() { mockBroker.Close() }()
			chain, _ := newChain(mockConsenter, mockSupport, newestOffset-1, lastOriginalOffsetProcessed, lastResubmittedConfigOffset)

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

			chain.Start()
			defer chain.Halt()

			select {
			case <-chain.startChan:
				logger.Debug("startChan is closed as it should be")
			case <-time.After(shortTimeout):
				t.Fatal("startChan should have been closed by now")
			}

//我们不需要在这里创建一个合法的信封，因为在这个测试中它没有被检查过。
			assert.NoError(t, chain.Configure(&cb.Envelope{}, uint64(0)), "Expect Configure successfully")
		})
	})
}

func TestSetupTopicForChannel(t *testing.T) {

	mockChannel := newChannel(channelNameForTest(t), defaultPartition)
	haltChan := make(chan struct{})

	mockBrokerNoError := sarama.NewMockBroker(t, 0)
	defer mockBrokerNoError.Close()
	metadataResponse := sarama.NewMockMetadataResponse(t)
	metadataResponse.SetBroker(mockBrokerNoError.Addr(),
		mockBrokerNoError.BrokerID())
	metadataResponse.SetController(mockBrokerNoError.BrokerID())

	mdrUnknownTopicOrPartition := &sarama.MetadataResponse{
		Version:      1,
		Brokers:      []*sarama.Broker{sarama.NewBroker(mockBrokerNoError.Addr())},
		ControllerID: -1,
		Topics: []*sarama.TopicMetadata{
			{
				Err:  sarama.ErrUnknownTopicOrPartition,
				Name: mockChannel.topic(),
			},
		},
	}

	mockBrokerNoError.SetHandlerByMap(map[string]sarama.MockResponse{
		"CreateTopicsRequest": sarama.NewMockWrapper(
			&sarama.CreateTopicsResponse{
				TopicErrors: map[string]*sarama.TopicError{
					mockChannel.topic(): {
						Err: sarama.ErrNoError}}}),
		"MetadataRequest": sarama.NewMockWrapper(mdrUnknownTopicOrPartition)})

	mockBrokerTopicExists := sarama.NewMockBroker(t, 1)
	defer mockBrokerTopicExists.Close()
	mockBrokerTopicExists.SetHandlerByMap(map[string]sarama.MockResponse{
		"CreateTopicsRequest": sarama.NewMockWrapper(
			&sarama.CreateTopicsResponse{
				TopicErrors: map[string]*sarama.TopicError{
					mockChannel.topic(): {
						Err: sarama.ErrTopicAlreadyExists}}}),
		"MetadataRequest": sarama.NewMockWrapper(&sarama.MetadataResponse{
			Version: 1,
			Topics: []*sarama.TopicMetadata{
				{
					Name: channelNameForTest(t),
					Err:  sarama.ErrNoError}}})})

	mockBrokerInvalidTopic := sarama.NewMockBroker(t, 2)
	defer mockBrokerInvalidTopic.Close()
	metadataResponse = sarama.NewMockMetadataResponse(t)
	metadataResponse.SetBroker(mockBrokerInvalidTopic.Addr(),
		mockBrokerInvalidTopic.BrokerID())
	metadataResponse.SetController(mockBrokerInvalidTopic.BrokerID())
	mockBrokerInvalidTopic.SetHandlerByMap(map[string]sarama.MockResponse{
		"CreateTopicsRequest": sarama.NewMockWrapper(
			&sarama.CreateTopicsResponse{
				TopicErrors: map[string]*sarama.TopicError{
					mockChannel.topic(): {
						Err: sarama.ErrInvalidTopic}}}),
		"MetadataRequest": metadataResponse})

	mockBrokerInvalidTopic2 := sarama.NewMockBroker(t, 3)
	defer mockBrokerInvalidTopic2.Close()
	mockBrokerInvalidTopic2.SetHandlerByMap(map[string]sarama.MockResponse{
		"CreateTopicsRequest": sarama.NewMockWrapper(
			&sarama.CreateTopicsResponse{
				TopicErrors: map[string]*sarama.TopicError{
					mockChannel.topic(): {
						Err: sarama.ErrInvalidTopic}}}),
		"MetadataRequest": sarama.NewMockWrapper(&sarama.MetadataResponse{
			Version:      1,
			Brokers:      []*sarama.Broker{sarama.NewBroker(mockBrokerInvalidTopic2.Addr())},
			ControllerID: mockBrokerInvalidTopic2.BrokerID()})})

	closedBroker := sarama.NewMockBroker(t, 99)
	badAddress := closedBroker.Addr()
	closedBroker.Close()

	var tests = []struct {
		name         string
		brokers      []string
		brokerConfig *sarama.Config
		version      sarama.KafkaVersion
		expectErr    bool
		errorMsg     string
	}{
		{
			name:         "Unsupported Version",
			brokers:      []string{mockBrokerNoError.Addr()},
			brokerConfig: sarama.NewConfig(),
			version:      sarama.V0_9_0_0,
			expectErr:    false,
		},
		{
			name:         "No Error",
			brokers:      []string{mockBrokerNoError.Addr()},
			brokerConfig: sarama.NewConfig(),
			version:      sarama.V0_10_2_0,
			expectErr:    false,
		},
		{
			name:         "Topic Exists",
			brokers:      []string{mockBrokerTopicExists.Addr()},
			brokerConfig: sarama.NewConfig(),
			version:      sarama.V0_10_2_0,
			expectErr:    false,
		},
		{
			name:         "Invalid Topic",
			brokers:      []string{mockBrokerInvalidTopic.Addr()},
			brokerConfig: sarama.NewConfig(),
			version:      sarama.V0_10_2_0,
			expectErr:    true,
			errorMsg:     "process asked to exit",
		},
		{
			name:         "Multiple Brokers - One No Error",
			brokers:      []string{badAddress, mockBrokerNoError.Addr()},
			brokerConfig: sarama.NewConfig(),
			version:      sarama.V0_10_2_0,
			expectErr:    false,
		},
		{
			name:         "Multiple Brokers - All Errors",
			brokers:      []string{badAddress, badAddress},
			brokerConfig: sarama.NewConfig(),
			version:      sarama.V0_10_2_0,
			expectErr:    true,
			errorMsg:     "failed to retrieve metadata",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			test.brokerConfig.Version = test.version
			err := setupTopicForChannel(
				mockRetryOptions,
				haltChan,
				test.brokers,
				test.brokerConfig,
				&sarama.TopicDetail{
					NumPartitions:     1,
					ReplicationFactor: 2},
				mockChannel)
			if test.expectErr {
				assert.Contains(t, err.Error(), test.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func TestSetupProducerForChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	mockBroker := sarama.NewMockBroker(t, 0)
	defer mockBroker.Close()

	mockChannel := newChannel(channelNameForTest(t), defaultPartition)

	haltChan := make(chan struct{})

	t.Run("Proper", func(t *testing.T) {
		metadataResponse := new(sarama.MetadataResponse)
		metadataResponse.AddBroker(mockBroker.Addr(), mockBroker.BrokerID())
		metadataResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID(), nil, nil, sarama.ErrNoError)
		mockBroker.Returns(metadataResponse)

		producer, err := setupProducerForChannel(mockConsenter.retryOptions(), haltChan, []string{mockBroker.Addr()}, mockBrokerConfig, mockChannel)
		assert.NoError(t, err, "Expected the setupProducerForChannel call to return without errors")
		assert.NoError(t, producer.Close(), "Expected to close the producer without errors")
	})

	t.Run("WithError", func(t *testing.T) {
		_, err := setupProducerForChannel(mockConsenter.retryOptions(), haltChan, []string{}, mockBrokerConfig, mockChannel)
		assert.Error(t, err, "Expected the setupProducerForChannel call to return an error")
	})
}

func TestSetupConsumerForChannel(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 0)
	defer func() { mockBroker.Close() }()

	mockChannel := newChannel(channelNameForTest(t), defaultPartition)

	oldestOffset := int64(0)
	newestOffset := int64(5)

	startFrom := int64(3)
	message := sarama.StringEncoder("messageFoo")

	mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
			SetLeader(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetOldest, oldestOffset).
			SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetNewest, newestOffset),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(mockChannel.topic(), mockChannel.partition(), startFrom, message),
	})

	haltChan := make(chan struct{})

	t.Run("ProperParent", func(t *testing.T) {
		parentConsumer, err := setupParentConsumerForChannel(mockConsenter.retryOptions(), haltChan, []string{mockBroker.Addr()}, mockBrokerConfig, mockChannel)
		assert.NoError(t, err, "Expected the setupParentConsumerForChannel call to return without errors")
		assert.NoError(t, parentConsumer.Close(), "Expected to close the parentConsumer without errors")
	})

	t.Run("ProperChannel", func(t *testing.T) {
		parentConsumer, _ := setupParentConsumerForChannel(mockConsenter.retryOptions(), haltChan, []string{mockBroker.Addr()}, mockBrokerConfig, mockChannel)
		defer func() { parentConsumer.Close() }()
		channelConsumer, err := setupChannelConsumerForChannel(mockConsenter.retryOptions(), haltChan, parentConsumer, mockChannel, newestOffset)
		assert.NoError(t, err, "Expected the setupChannelConsumerForChannel call to return without errors")
		assert.NoError(t, channelConsumer.Close(), "Expected to close the channelConsumer without errors")
	})

	t.Run("WithParentConsumerError", func(t *testing.T) {
//提供空的经纪人名单
		_, err := setupParentConsumerForChannel(mockConsenter.retryOptions(), haltChan, []string{}, mockBrokerConfig, mockChannel)
		assert.Error(t, err, "Expected the setupParentConsumerForChannel call to return an error")
	})

	t.Run("WithChannelConsumerError", func(t *testing.T) {
//提供超出范围的偏移
		parentConsumer, _ := setupParentConsumerForChannel(mockConsenter.retryOptions(), haltChan, []string{mockBroker.Addr()}, mockBrokerConfig, mockChannel)
		_, err := setupChannelConsumerForChannel(mockConsenter.retryOptions(), haltChan, parentConsumer, mockChannel, newestOffset+1)
		defer func() { parentConsumer.Close() }()
		assert.Error(t, err, "Expected the setupChannelConsumerForChannel call to return an error")
	})
}

func TestCloseKafkaObjects(t *testing.T) {
	mockChannel := newChannel(channelNameForTest(t), defaultPartition)

	mockSupport := &mockmultichannel.ConsenterSupport{
		ChainIDVal: mockChannel.topic(),
	}

	oldestOffset := int64(0)
	newestOffset := int64(5)

	startFrom := int64(3)
	message := sarama.StringEncoder("messageFoo")

	mockBroker := sarama.NewMockBroker(t, 0)
	defer func() { mockBroker.Close() }()

	mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
			SetLeader(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetOldest, oldestOffset).
			SetOffset(mockChannel.topic(), mockChannel.partition(), sarama.OffsetNewest, newestOffset),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(mockChannel.topic(), mockChannel.partition(), startFrom, message),
	})

	haltChan := make(chan struct{})

	t.Run("Proper", func(t *testing.T) {
		producer, _ := setupProducerForChannel(mockConsenter.retryOptions(), haltChan, []string{mockBroker.Addr()}, mockBrokerConfig, mockChannel)
		parentConsumer, _ := setupParentConsumerForChannel(mockConsenter.retryOptions(), haltChan, []string{mockBroker.Addr()}, mockBrokerConfig, mockChannel)
		channelConsumer, _ := setupChannelConsumerForChannel(mockConsenter.retryOptions(), haltChan, parentConsumer, mockChannel, startFrom)

//设置一个仅实例化了最小必需字段的链，以便
//关于测试功能
		bareMinimumChain := &chainImpl{
			ConsenterSupport: mockSupport,
			producer:         producer,
			parentConsumer:   parentConsumer,
			channelConsumer:  channelConsumer,
		}

		errs := bareMinimumChain.closeKafkaObjects()

		assert.Len(t, errs, 0, "Expected zero errors")

		assert.NotPanics(t, func() {
			channelConsumer.Close()
		})

		assert.NotPanics(t, func() {
			parentConsumer.Close()
		})

//托多，出于某种原因，“断言”无法捕捉到这种恐慌。
//测试框架。不是交易破坏者，但需要进一步调查。
  /*断言.panics（t，func（）
   producer.close（）。
  }）*/

	})

	t.Run("ChannelConsumerError", func(t *testing.T) {
		producer, _ := sarama.NewSyncProducer([]string{mockBroker.Addr()}, mockBrokerConfig)

//与此文件中的所有其他测试不同，在
//channelConsumer.close（）调用使用mock更容易实现
//消费者。因此，我们绕过了对“setup*consumer”的调用。

//让消费者收到错误消息。
		mockParentConsumer := mocks.NewConsumer(t, nil)
		mockParentConsumer.ExpectConsumePartition(mockChannel.topic(), mockChannel.partition(), startFrom).YieldError(sarama.ErrOutOfBrokers)
		mockChannelConsumer, err := mockParentConsumer.ConsumePartition(mockChannel.topic(), mockChannel.partition(), startFrom)
		assert.NoError(t, err, "Expected no error when setting up the mock partition consumer")

		bareMinimumChain := &chainImpl{
			ConsenterSupport: mockSupport,
			producer:         producer,
			parentConsumer:   mockParentConsumer,
			channelConsumer:  mockChannelConsumer,
		}

		errs := bareMinimumChain.closeKafkaObjects()

		assert.Len(t, errs, 1, "Expected 1 error returned")

		assert.NotPanics(t, func() {
			mockChannelConsumer.Close()
		})

		assert.NotPanics(t, func() {
			mockParentConsumer.Close()
		})
	})
}

func TestGetLastCutBlockNumber(t *testing.T) {
	testCases := []struct {
		name     string
		input    uint64
		expected uint64
	}{
		{"Proper", uint64(2), uint64(1)},
		{"Zero", uint64(1), uint64(0)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, getLastCutBlockNumber(tc.input))
		})
	}
}

func TestGetLastOffsetPersisted(t *testing.T) {
	mockChannel := newChannel(channelNameForTest(t), defaultPartition)
	mockMetadata := &cb.Metadata{Value: utils.MarshalOrPanic(&ab.KafkaMetadata{
		LastOffsetPersisted:         int64(5),
		LastOriginalOffsetProcessed: int64(3),
		LastResubmittedConfigOffset: int64(4),
	})}

	testCases := []struct {
		name                string
		md                  []byte
		expectedPersisted   int64
		expectedProcessed   int64
		expectedResubmitted int64
		panics              bool
	}{
		{"Proper", mockMetadata.Value, int64(5), int64(3), int64(4), false},
		{"Empty", nil, sarama.OffsetOldest - 1, int64(0), int64(0), false},
		{"Panics", tamperBytes(mockMetadata.Value), sarama.OffsetOldest - 1, int64(0), int64(0), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.panics {
				persisted, processed, resubmitted := getOffsets(tc.md, mockChannel.String())
				assert.Equal(t, tc.expectedPersisted, persisted)
				assert.Equal(t, tc.expectedProcessed, processed)
				assert.Equal(t, tc.expectedResubmitted, resubmitted)
			} else {
				assert.Panics(t, func() {
					getOffsets(tc.md, mockChannel.String())
				}, "Expected getOffsets call to panic")
			}
		})
	}
}

func TestSendConnectMessage(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 0)
	defer func() { mockBroker.Close() }()

	mockChannel := newChannel("mockChannelFoo", defaultPartition)

	metadataResponse := new(sarama.MetadataResponse)
	metadataResponse.AddBroker(mockBroker.Addr(), mockBroker.BrokerID())
	metadataResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID(), nil, nil, sarama.ErrNoError)
	mockBroker.Returns(metadataResponse)

	producer, _ := sarama.NewSyncProducer([]string{mockBroker.Addr()}, mockBrokerConfig)
	defer func() { producer.Close() }()

	haltChan := make(chan struct{})

	t.Run("Proper", func(t *testing.T) {
		successResponse := new(sarama.ProduceResponse)
		successResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), sarama.ErrNoError)
		mockBroker.Returns(successResponse)

		assert.NoError(t, sendConnectMessage(mockConsenter.retryOptions(), haltChan, producer, mockChannel), "Expected the sendConnectMessage call to return without errors")
	})

	t.Run("WithError", func(t *testing.T) {
//请注意，此测试受以下参数影响：
//-net.readtimeout
//-consumer.retry.backoff
//-metadata.retry.max

//让代理返回errnotenoughreplicas错误
		failureResponse := new(sarama.ProduceResponse)
		failureResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), sarama.ErrNotEnoughReplicas)
		mockBroker.Returns(failureResponse)

		assert.Error(t, sendConnectMessage(mockConsenter.retryOptions(), haltChan, producer, mockChannel), "Expected the sendConnectMessage call to return an error")
	})
}

func TestSendTimeToCut(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 0)
	defer func() { mockBroker.Close() }()

	mockChannel := newChannel("mockChannelFoo", defaultPartition)

	metadataResponse := new(sarama.MetadataResponse)
	metadataResponse.AddBroker(mockBroker.Addr(), mockBroker.BrokerID())
	metadataResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID(), nil, nil, sarama.ErrNoError)
	mockBroker.Returns(metadataResponse)

	producer, err := sarama.NewSyncProducer([]string{mockBroker.Addr()}, mockBrokerConfig)
	assert.NoError(t, err, "Expected no error when setting up the sarama SyncProducer")
	defer func() { producer.Close() }()

	timeToCutBlockNumber := uint64(3)
	var timer <-chan time.Time

	t.Run("Proper", func(t *testing.T) {
		successResponse := new(sarama.ProduceResponse)
		successResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), sarama.ErrNoError)
		mockBroker.Returns(successResponse)

		timer = time.After(longTimeout)

		assert.NoError(t, sendTimeToCut(producer, mockChannel, timeToCutBlockNumber, &timer), "Expected the sendTimeToCut call to return without errors")
		assert.Nil(t, timer, "Expected the sendTimeToCut call to nil the timer")
	})

	t.Run("WithError", func(t *testing.T) {
//请注意，此测试受以下参数影响：
//-net.readtimeout
//-consumer.retry.backoff
//-metadata.retry.max
		failureResponse := new(sarama.ProduceResponse)
		failureResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), sarama.ErrNotEnoughReplicas)
		mockBroker.Returns(failureResponse)

		timer = time.After(longTimeout)

		assert.Error(t, sendTimeToCut(producer, mockChannel, timeToCutBlockNumber, &timer), "Expected the sendTimeToCut call to return an error")
		assert.Nil(t, timer, "Expected the sendTimeToCut call to nil the timer")
	})
}

func TestProcessMessagesToBlocks(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 0)
	defer func() { mockBroker.Close() }()

	mockChannel := newChannel("mockChannelFoo", defaultPartition)

	metadataResponse := new(sarama.MetadataResponse)
	metadataResponse.AddBroker(mockBroker.Addr(), mockBroker.BrokerID())
	metadataResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), mockBroker.BrokerID(), nil, nil, sarama.ErrNoError)
	mockBroker.Returns(metadataResponse)

	producer, _ := sarama.NewSyncProducer([]string{mockBroker.Addr()}, mockBrokerConfig)

	mockBrokerConfigCopy := *mockBrokerConfig
	mockBrokerConfigCopy.ChannelBufferSize = 0

	mockParentConsumer := mocks.NewConsumer(t, &mockBrokerConfigCopy)
	mpc := mockParentConsumer.ExpectConsumePartition(mockChannel.topic(), mockChannel.partition(), int64(0))
	mockChannelConsumer, err := mockParentConsumer.ConsumePartition(mockChannel.topic(), mockChannel.partition(), int64(0))
	assert.NoError(t, err, "Expected no error when setting up the mock partition consumer")

	t.Run("TimeToCut", func(t *testing.T) {
		t.Run("PendingMsgToCutProper", func(t *testing.T) {
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:          make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal:  mockblockcutter.NewReceiver(),
				ChainIDVal:      mockChannel.topic(),
HeightVal:       lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: shortTimeout / 2},
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				producer:        producer,
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

//我们需要模拟块切割机来运送一批非空的
			go func() {
mockSupport.BlockCutterVal.Block <- struct{}{} //让下面的“mockblockcutter.ordered”调用返回
				logger.Debugf("Mock blockcutter's Ordered call has returned")
			}()
//我们正在“植入”一个信息，直接向模拟块切割机
			mockSupport.BlockCutterVal.Ordered(newMockEnvelope("fooMessage"))

			done := make(chan struct{})

			go func() {
				bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//割断祖先
			mockSupport.BlockCutterVal.CutAncestors = true

//此信封将添加到挂起列表中，等待计时器触发时被剪切。
			mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))

			go func() {
				mockSupport.BlockCutterVal.Block <- struct{}{}
				logger.Debugf("Mock blockcutter's Ordered call has returned")
			}()

<-mockSupport.Blocks //等待第一个街区

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			if bareMinimumChain.timer != nil {
				go func() {
<-bareMinimumChain.timer //启动垃圾收集计时器
				}()
			}

			assert.NotEmpty(t, mockSupport.BlockCutterVal.CurBatch, "Expected the blockCutter to be non-empty")
			assert.NotNil(t, bareMinimumChain.timer, "Expected the cutTimer to be non-nil when there are pending envelopes")

		})

		t.Run("ReceiveTimeToCutProper", func(t *testing.T) {
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

//我们需要模拟块切割机来运送一批非空的
			go func() {
mockSupport.BlockCutterVal.Block <- struct{}{} //让下面的“mockblockcutter.ordered”调用返回
				logger.Debugf("Mock blockcutter's Ordered call has returned")
			}()
//我们正在“植入”一个信息，直接向模拟块切割机
			mockSupport.BlockCutterVal.Ordered(newMockEnvelope("fooMessage"))

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的wrappedMessage
			mpc.YieldMessage(newMockConsumerMessage(newTimeToCutMessage(lastCutBlockNumber + 1)))

<-mockSupport.Blocks //让“mockConsentersupport.writeBlock”继续

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessTimeToCutPass], "Expected 1 TIMETOCUT message processed")
			assert.Equal(t, lastCutBlockNumber+1, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to be bumped up by one")
		})

		t.Run("ReceiveTimeToCutZeroBatch", func(t *testing.T) {
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的wrappedMessage
			mpc.YieldMessage(newMockConsumerMessage(newTimeToCutMessage(lastCutBlockNumber + 1)))

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			assert.Error(t, err, "Expected the processMessagesToBlocks call to return an error")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessTimeToCutError], "Expected 1 faulty TIMETOCUT message processed")
			assert.Equal(t, lastCutBlockNumber, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to stay the same")
		})

		t.Run("ReceiveTimeToCutLargerThanExpected", func(t *testing.T) {
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的wrappedMessage
			mpc.YieldMessage(newMockConsumerMessage(newTimeToCutMessage(lastCutBlockNumber + 2)))

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			assert.Error(t, err, "Expected the processMessagesToBlocks call to return an error")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessTimeToCutError], "Expected 1 faulty TIMETOCUT message processed")
			assert.Equal(t, lastCutBlockNumber, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to stay the same")
		})

		t.Run("ReceiveTimeToCutStale", func(t *testing.T) {
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的wrappedMessage
			mpc.YieldMessage(newMockConsumerMessage(newTimeToCutMessage(lastCutBlockNumber)))

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessTimeToCutPass], "Expected 1 TIMETOCUT message processed")
			assert.Equal(t, lastCutBlockNumber, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to stay the same")
		})
	})

	t.Run("Connect", func(t *testing.T) {
		t.Run("ReceiveConnect", func(t *testing.T) {
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			mockSupport := &mockmultichannel.ConsenterSupport{
				ChainIDVal: mockChannel.topic(),
			}

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:          mockChannel,
				ConsenterSupport: mockSupport,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的wrappedMessage
			mpc.YieldMessage(newMockConsumerMessage(newConnectMessage()))

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessConnectPass], "Expected 1 CONNECT message processed")
		})
	})

	t.Run("Regular", func(t *testing.T) {
		t.Run("Error", func(t *testing.T) {
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			mockSupport := &mockmultichannel.ConsenterSupport{
				ChainIDVal: mockChannel.topic(),
			}

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:          mockChannel,
				ConsenterSupport: mockSupport,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的wrappedMessage
			mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(tamperBytes(utils.MarshalOrPanic(newMockEnvelope("fooMessage"))))))

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 damaged REGULAR message processed")
		})

//这样可以确保正确处理未知类型的常规Kafka消息
		t.Run("Unknown", func(t *testing.T) {
			t.Run("Enqueue", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

//这是for循环将处理的wrappedMessage
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))

mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回
				logger.Debugf("Mock blockcutter's Ordered call has returned")

				logger.Debug("Closing haltChan to exit the infinite for-loop")
//我们保证至少打过一次正规分支机构后就可以打到浩昌分支机构。
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
			})

			t.Run("CutBlock", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{})}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				mockSupport.BlockCutterVal.CutNext = true

//这是for循环将处理的wrappedMessage
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))

mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回
				logger.Debugf("Mock blockcutter's Ordered call has returned")
<-mockSupport.Blocks //让“mockConsentersupport.writeBlock”继续

				logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
				assert.Equal(t, lastCutBlockNumber+1, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to be bumped up by one")
			})

//此测试确保处理FAB-5709中的角箱
			t.Run("SecondTxOverflows", func(t *testing.T) {
				if testing.Short() {
					t.Skip("Skipping test in short mode")
				}

				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				var block1, block2 *cb.Block

				block1LastOffset := mpc.HighWaterMarkOffset()
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))
mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回

//将CutAncestors设置为true，以便第二条消息溢出接收器批处理
				mockSupport.BlockCutterVal.CutAncestors = true
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))
				mockSupport.BlockCutterVal.Block <- struct{}{}

				select {
case block1 = <-mockSupport.Blocks: //让“mockConsentersupport.writeBlock”继续
				case <-time.After(shortTimeout):
					logger.Fatalf("Did not receive a block from the blockcutter as expected")
				}

//将cutnext设置为true以刷新所有挂起的消息
				mockSupport.BlockCutterVal.CutAncestors = false
				mockSupport.BlockCutterVal.CutNext = true
				block2LastOffset := mpc.HighWaterMarkOffset()
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))
				mockSupport.BlockCutterVal.Block <- struct{}{}

				select {
				case block2 = <-mockSupport.Blocks:
				case <-time.After(shortTimeout):
					logger.Fatalf("Did not receive a block from the blockcutter as expected")
				}

				logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(3), counts[indexRecvPass], "Expected 2 messages received and unmarshaled")
				assert.Equal(t, uint64(3), counts[indexProcessRegularPass], "Expected 2 REGULAR messages processed")
				assert.Equal(t, lastCutBlockNumber+2, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to be bumped up by two")
				assert.Equal(t, block1LastOffset, extractEncodedOffset(block1.GetMetadata().Metadata[cb.BlockMetadataIndex_ORDERER]), "Expected encoded offset in first block to be %d", block1LastOffset)
				assert.Equal(t, block2LastOffset, extractEncodedOffset(block2.GetMetadata().Metadata[cb.BlockMetadataIndex_ORDERER]), "Expected encoded offset in second block to be %d", block2LastOffset)
			})

			t.Run("InvalidConfigEnv", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:              make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal:      mockblockcutter.NewReceiver(),
					ChainIDVal:          mockChannel.topic(),
HeightVal:           lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					ClassifyMsgVal:      msgprocessor.ConfigMsg,
					ProcessConfigMsgErr: fmt.Errorf("Invalid config message"),
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

//这是for循环将处理的config wrappedMessage。
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockConfigEnvelope()))))

				logger.Debug("Closing haltChan to exit the infinite for-loop")
//我们保证至少打过一次正规分支机构后就可以打到浩昌分支机构。
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 REGULAR message error")
				assert.Equal(t, lastCutBlockNumber, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber not to be incremented")
			})

			t.Run("InvalidOrdererTxEnv", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:              make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal:      mockblockcutter.NewReceiver(),
					ChainIDVal:          mockChannel.topic(),
HeightVal:           lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					ClassifyMsgVal:      msgprocessor.ConfigMsg,
					ProcessConfigMsgErr: fmt.Errorf("Invalid config message"),
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

//这是for循环将处理的config wrappedMessage。
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockOrdererTxEnvelope()))))

				logger.Debug("Closing haltChan to exit the infinite for-loop")
//我们保证至少打过一次正规分支机构后就可以打到浩昌分支机构。
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 REGULAR message error")
				assert.Equal(t, lastCutBlockNumber, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber not to be incremented")
			})

			t.Run("InvalidNormalEnv", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
					ProcessNormalMsgErr: fmt.Errorf("Invalid normal message"),
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

//这是for循环将处理的wrappedMessage
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))

close(haltChan) //与chain.halt（）相同
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 REGULAR message processed")
			})

			t.Run("CutConfigEnv", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
					ClassifyMsgVal: msgprocessor.ConfigMsg,
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				configBlkOffset := mpc.HighWaterMarkOffset()
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockConfigEnvelope()))))

				var configBlk *cb.Block

				select {
				case configBlk = <-mockSupport.Blocks:
				case <-time.After(shortTimeout):
					logger.Fatalf("Did not receive a config block from the blockcutter as expected")
				}

close(haltChan) //与chain.halt（）相同
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
				assert.Equal(t, lastCutBlockNumber+1, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to be incremented by 1")
				assert.Equal(t, configBlkOffset, extractEncodedOffset(configBlk.GetMetadata().Metadata[cb.BlockMetadataIndex_ORDERER]), "Expected encoded offset in second block to be %d", configBlkOffset)
			})

//我们不期望卡夫卡发出这种类型的消息
			t.Run("ConfigUpdateEnv", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
					},
					ClassifyMsgVal: msgprocessor.ConfigUpdateMsg,
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("FooMessage")))))

close(haltChan) //与chain.halt（）相同
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 REGULAR message processed")
			})

			t.Run("SendTimeToCut", func(t *testing.T) {
				t.Skip("Skipping test as it introduces a race condition")

//注意，我们没有为模拟代理设置handlermap，因此我们需要设置
//生产响应
				successResponse := new(sarama.ProduceResponse)
				successResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), sarama.ErrNoError)
				mockBroker.Returns(successResponse)

				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
BatchTimeoutVal: extraShortTimeout, //阿特恩
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					producer:        producer,
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

//这是for循环将处理的wrappedMessage
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))

mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回
				logger.Debugf("Mock blockcutter's Ordered call has returned")

//睡眠，以便在退出前激活计时器分支。
//TODO这是一个争用条件，将在后续变更集中修复
				time.Sleep(hitBranch)

				logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
				assert.Equal(t, uint64(1), counts[indexSendTimeToCutPass], "Expected 1 TIMER event processed")
				assert.Equal(t, lastCutBlockNumber, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to stay the same")
			})

			t.Run("SendTimeToCutError", func(t *testing.T) {
//请注意，此测试受以下参数影响：
//-net.readtimeout
//-consumer.retry.backoff
//-metadata.retry.max

				t.Skip("Skipping test as it introduces a race condition")

//与receiveregularandsendtimetocut完全相同的测试。
//唯一不同的是，生产者试图发送一个TTC将
//失败，出现errnotenoughreplicas错误。
				failureResponse := new(sarama.ProduceResponse)
				failureResponse.AddTopicPartition(mockChannel.topic(), mockChannel.partition(), sarama.ErrNotEnoughReplicas)
				mockBroker.Returns(failureResponse)

				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
BatchTimeoutVal: extraShortTimeout, //阿特恩
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					producer:        producer,
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

//这是for循环将处理的wrappedMessage
				mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")))))

mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回
				logger.Debugf("Mock blockcutter's Ordered call has returned")

//睡眠，以便在退出前激活计时器分支。
//TODO这是一个争用条件，将在后续变更集中修复
				time.Sleep(hitBranch)

				logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
				assert.Equal(t, uint64(1), counts[indexSendTimeToCutError], "Expected 1 faulty TIMER event processed")
				assert.Equal(t, lastCutBlockNumber, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to stay the same")
			})
		})

//这样可以确保正常类型的常规Kafka消息得到正确处理。
		t.Run("Normal", func(t *testing.T) {
			lastOriginalOffsetProcessed := int64(3)

			t.Run("ReceiveTwoRegularAndCutTwoBlocks", func(t *testing.T) {
				if testing.Short() {
					t.Skip("Skipping test in short mode")
				}

				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
						CapabilitiesVal: &mockconfig.OrdererCapabilities{
							ResubmissionVal: false,
						},
					},
					SequenceVal: uint64(0),
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:                     mockChannel,
					ConsenterSupport:            mockSupport,
					lastCutBlockNumber:          lastCutBlockNumber,
					lastOriginalOffsetProcessed: lastOriginalOffsetProcessed,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				var block1, block2 *cb.Block

//这是for循环将处理的第一个wrappedMessage
				block1LastOffset := mpc.HighWaterMarkOffset()
				mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(0))))
mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回
				logger.Debugf("Mock blockcutter's Ordered call has returned")

				mockSupport.BlockCutterVal.IsolatedTx = true

//这是for循环将处理的第一个wrappedMessage
				block2LastOffset := mpc.HighWaterMarkOffset()
				mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(0))))
				mockSupport.BlockCutterVal.Block <- struct{}{}
				logger.Debugf("Mock blockcutter's Ordered call has returned for the second time")

				select {
case block1 = <-mockSupport.Blocks: //让“mockConsentersupport.writeBlock”继续
				case <-time.After(shortTimeout):
					logger.Fatalf("Did not receive a block from the blockcutter as expected")
				}

				select {
				case block2 = <-mockSupport.Blocks:
				case <-time.After(shortTimeout):
					logger.Fatalf("Did not receive a block from the blockcutter as expected")
				}

				logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(2), counts[indexRecvPass], "Expected 2 messages received and unmarshaled")
				assert.Equal(t, uint64(2), counts[indexProcessRegularPass], "Expected 2 REGULAR messages processed")
				assert.Equal(t, lastCutBlockNumber+2, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to be bumped up by two")
				assert.Equal(t, block1LastOffset, extractEncodedOffset(block1.GetMetadata().Metadata[cb.BlockMetadataIndex_ORDERER]), "Expected encoded offset in first block to be %d", block1LastOffset)
				assert.Equal(t, block2LastOffset, extractEncodedOffset(block2.GetMetadata().Metadata[cb.BlockMetadataIndex_ORDERER]), "Expected encoded offset in second block to be %d", block2LastOffset)
			})

			t.Run("ReceiveRegularAndQueue", func(t *testing.T) {
				if testing.Short() {
					t.Skip("Skipping test in short mode")
				}

				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
						CapabilitiesVal: &mockconfig.OrdererCapabilities{
							ResubmissionVal: false,
						},
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:                     mockChannel,
					ConsenterSupport:            mockSupport,
					lastCutBlockNumber:          lastCutBlockNumber,
					lastOriginalOffsetProcessed: lastOriginalOffsetProcessed,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				mockSupport.BlockCutterVal.CutNext = true

				mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(0))))
mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回
				<-mockSupport.Blocks

				close(haltChan)
				logger.Debug("haltChan closed")
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 2 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
			})
		})

//这样可以确保正确处理常规的config类型的kafka消息。
		t.Run("Config", func(t *testing.T) {
//此测试发送一个正常的Tx，然后发送一个配置Tx。它应该
//立刻把它们切成两块。
			t.Run("ReceiveConfigEnvelopeAndCut", func(t *testing.T) {
				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
						CapabilitiesVal: &mockconfig.OrdererCapabilities{
							ResubmissionVal: false,
						},
					},
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				normalBlkOffset := mpc.HighWaterMarkOffset()
				mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(0))))
mockSupport.BlockCutterVal.Block <- struct{}{} //让“mockblockcutter.ordered”调用返回

				configBlkOffset := mpc.HighWaterMarkOffset()
				mockSupport.ClassifyMsgVal = msgprocessor.ConfigMsg
				mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(
					utils.MarshalOrPanic(newMockConfigEnvelope()),
					uint64(0),
					int64(0))))

				var normalBlk, configBlk *cb.Block
				select {
				case normalBlk = <-mockSupport.Blocks:
				case <-time.After(shortTimeout):
					logger.Fatalf("Did not receive a normal block from the blockcutter as expected")
				}

				select {
				case configBlk = <-mockSupport.Blocks:
				case <-time.After(shortTimeout):
					logger.Fatalf("Did not receive a config block from the blockcutter as expected")
				}

close(haltChan) //与chain.halt（）相同
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(2), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(2), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
				assert.Equal(t, lastCutBlockNumber+2, bareMinimumChain.lastCutBlockNumber, "Expected lastCutBlockNumber to be incremented by 2")
				assert.Equal(t, normalBlkOffset, extractEncodedOffset(normalBlk.GetMetadata().Metadata[cb.BlockMetadataIndex_ORDERER]), "Expected encoded offset in first block to be %d", normalBlkOffset)
				assert.Equal(t, configBlkOffset, extractEncodedOffset(configBlk.GetMetadata().Metadata[cb.BlockMetadataIndex_ORDERER]), "Expected encoded offset in second block to be %d", configBlkOffset)
			})

//这可以确保配置消息在config seq高级时重新验证。
			t.Run("RevalidateConfigEnvInvalid", func(t *testing.T) {
				if testing.Short() {
					t.Skip("Skipping test in short mode")
				}

				errorChan := make(chan struct{})
				close(errorChan)
				haltChan := make(chan struct{})

				lastCutBlockNumber := uint64(3)

				mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
					BlockCutterVal: mockblockcutter.NewReceiver(),
					ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
					ClassifyMsgVal: msgprocessor.ConfigMsg,
					SharedConfigVal: &mockconfig.Orderer{
						BatchTimeoutVal: longTimeout,
						CapabilitiesVal: &mockconfig.OrdererCapabilities{
							ResubmissionVal: false,
						},
					},
					SequenceVal:         uint64(1),
					ProcessConfigMsgErr: fmt.Errorf("Invalid config message"),
				}
				defer close(mockSupport.BlockCutterVal.Block)

				bareMinimumChain := &chainImpl{
					parentConsumer:  mockParentConsumer,
					channelConsumer: mockChannelConsumer,

					channel:            mockChannel,
					ConsenterSupport:   mockSupport,
					lastCutBlockNumber: lastCutBlockNumber,

					errorChan:                      errorChan,
					haltChan:                       haltChan,
					doneProcessingMessagesToBlocks: make(chan struct{}),
				}

				var counts []uint64
				done := make(chan struct{})

				go func() {
					counts, err = bareMinimumChain.processMessagesToBlocks()
					done <- struct{}{}
				}()

				mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(
					utils.MarshalOrPanic(newMockConfigEnvelope()),
					uint64(0),
					int64(0))))
				select {
				case <-mockSupport.Blocks:
					t.Fatalf("Expected no block being cut given invalid config message")
				case <-time.After(shortTimeout):
				}

close(haltChan) //与chain.halt（）相同
				<-done

				assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
				assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
				assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 REGULAR message error")
			})
		})
	})

	t.Run("KafkaError", func(t *testing.T) {
		t.Run("ReceiveKafkaErrorAndCloseErrorChan", func(t *testing.T) {
//如果我们设置模拟代理以便它返回响应，如果
//在接收到sendconnectMessage goroutine之前，测试完成
//对此，我们将失败（“并非所有的期望都是
//满意”）来自模拟经纪人。所以我们破坏了制片人。
			failedProducer, _ := sarama.NewSyncProducer([]string{}, mockBrokerConfig)

//我们需要让sendconnectmessage goroutine立即消失，
//否则，我们将得到一个零指针取消引用恐慌。我们是
//利用公认的黑客快捷方式进行可重审程序
//当给定nil time.duration值时立即返回
//滴答声。
			zeroRetryConsenter := &consenterImpl{}

//让我们假设一个开放的错误，即
//用户和对应于通道的Kafka分区
			errorChan := make(chan struct{})

			haltChan := make(chan struct{})

			mockSupport := &mockmultichannel.ConsenterSupport{
				ChainIDVal: mockChannel.topic(),
			}

			bareMinimumChain := &chainImpl{
consenter:       zeroRetryConsenter, //对于sendconnectmessage
producer:        failedProducer,     //对于sendconnectmessage
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:          mockChannel,
				ConsenterSupport: mockSupport,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的内容
			mpc.YieldError(fmt.Errorf("fooError"))

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvError], "Expected 1 Kafka error received")

			select {
			case <-bareMinimumChain.errorChan:
				logger.Debug("errorChan is closed as it should be")
			default:
				t.Fatal("errorChan should have been closed")
			}
		})

		t.Run("ReceiveKafkaErrorAndThenReceiveRegularMessage", func(t *testing.T) {
			t.Skip("Skipping test as it introduces a race condition")

//如果我们设置模拟代理以便它返回响应，如果
//在接收到sendconnectMessage goroutine之前，测试完成
//对此，我们将失败（“并非所有的期望都是
//满意”）来自模拟经纪人。所以我们破坏了制片人。
			failedProducer, _ := sarama.NewSyncProducer([]string{}, mockBrokerConfig)

//我们需要让sendconnectmessage goroutine立即消失，
//否则，我们将得到一个零指针取消引用恐慌。我们是
//利用公认的黑客快捷方式进行可重审程序
//当给定nil time.duration值时立即返回
//滴答声。
			zeroRetryConsenter := &consenterImpl{}

//如果ErrorChan已经关闭，Kafkarr分支机构不应该
//触摸它
			errorChan := make(chan struct{})
			close(errorChan)

			haltChan := make(chan struct{})

			mockSupport := &mockmultichannel.ConsenterSupport{
				ChainIDVal: mockChannel.topic(),
			}

			bareMinimumChain := &chainImpl{
consenter:       zeroRetryConsenter, //对于sendconnectmessage
producer:        failedProducer,     //对于sendconnectmessage
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:          mockChannel,
				ConsenterSupport: mockSupport,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			done := make(chan struct{})

			go func() {
				_, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//这是for循环将处理的内容
			mpc.YieldError(fmt.Errorf("foo"))

//我们在ReceiveKafkarror和CloseErrorChan中对此进行了测试，因此此检查
//在这方面是多余的。但是我们使用它来确保
//在继续推动
//常规消息。
			select {
			case <-bareMinimumChain.errorChan:
				logger.Debug("errorChan is closed as it should be")
			case <-time.After(shortTimeout):
				t.Fatal("errorChan should have been closed by now")
			}

//这是for循环将处理的wrappedMessage。我们使用
//因为这是最短的消息，所以这里有一条中断的定期消息。
//它允许我们测试我们想要的东西。
			mpc.YieldMessage(newMockConsumerMessage(newRegularMessage(tamperBytes(utils.MarshalOrPanic(newMockEnvelope("fooMessage"))))))

//休眠以便激活messages/errorchan分支。
//todo hacky方法，最终需要修改
			time.Sleep(hitBranch)

//检查是否重新创建了错误通道
			select {
			case <-bareMinimumChain.errorChan:
				t.Fatal("errorChan should have been open")
			default:
				logger.Debug("errorChan is open as it should be")
			}

			logger.Debug("Closing haltChan to exit the infinite for-loop")
close(haltChan) //与chain.halt（）相同
			logger.Debug("haltChan closed")
			<-done
		})
	})
}

//这可确保在config seq已高级时重新验证消息。
func TestResubmission(t *testing.T) {
	blockIngressMsg := func(t *testing.T, block bool, fn func() error) {
		wait := make(chan struct{})
		go func() {
			fn()
			wait <- struct{}{}
		}()

		select {
		case <-wait:
			if block {
				t.Fatalf("Expected WaitReady to block")
			}
		case <-time.After(100 * time.Millisecond):
			if !block {
				t.Fatalf("Expected WaitReady not to block")
			}
		}
	}

	mockBroker := sarama.NewMockBroker(t, 0)
	defer func() { mockBroker.Close() }()

	mockChannel := newChannel(channelNameForTest(t), defaultPartition)
	mockBrokerConfigCopy := *mockBrokerConfig
	mockBrokerConfigCopy.ChannelBufferSize = 0

	mockParentConsumer := mocks.NewConsumer(t, &mockBrokerConfigCopy)
	mpc := mockParentConsumer.ExpectConsumePartition(mockChannel.topic(), mockChannel.partition(), int64(0))
	mockChannelConsumer, err := mockParentConsumer.ConsumePartition(mockChannel.topic(), mockChannel.partition(), int64(0))
	assert.NoError(t, err, "Expected no error when setting up the mock partition consumer")

	t.Run("Normal", func(t *testing.T) {
//此测试允许卡夫卡发出模拟重新提交的消息，不需要重新处理。
//（通过设置originaloffset<=lastoriginaloffsetprocessed）
		t.Run("AlreadyProcessedDiscard", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)
			lastOriginalOffsetProcessed := int64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:                     mockChannel,
				ConsenterSupport:            mockSupport,
				lastCutBlockNumber:          lastCutBlockNumber,
				lastOriginalOffsetProcessed: lastOriginalOffsetProcessed,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

			mockSupport.BlockCutterVal.CutNext = true

			mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(2))))

			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut")
			case <-time.After(shortTimeout):
			}

			close(haltChan)
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 2 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
		})

//此测试允许卡夫卡发出模拟重新提交的消息，需要重新处理。
//（通过设置OriginalOffset>LastOriginalOffsetProcessed）
//在此测试用例中，有两条正常消息排队：重新编译的正常消息，其中
//'originaloffset'不是0，后跟一个普通的msg，其中'originaloffset'是0。
//它测试的情况是，即使没有块被剪切，“LastOriginalOffsetProcessed”仍然是
//更新。我们检查块以验证
//卡夫卡元数据。
		t.Run("ResubmittedMsgEnqueue", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)
			lastOriginalOffsetProcessed := int64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
				SequenceVal: uint64(0),
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:                     mockChannel,
				ConsenterSupport:            mockSupport,
				lastCutBlockNumber:          lastCutBlockNumber,
				lastOriginalOffsetProcessed: lastOriginalOffsetProcessed,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

			mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(4))))
			mockSupport.BlockCutterVal.Block <- struct{}{}

			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block to be cut")
			case <-time.After(shortTimeout):
			}

			mockSupport.BlockCutterVal.CutNext = true
			mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(0))))
			mockSupport.BlockCutterVal.Block <- struct{}{}

			select {
			case block := <-mockSupport.Blocks:
				metadata := &cb.Metadata{}
				proto.Unmarshal(block.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER], metadata)
				kafkaMetadata := &ab.KafkaMetadata{}
				proto.Unmarshal(metadata.Value, kafkaMetadata)
				assert.Equal(t, kafkaMetadata.LastOriginalOffsetProcessed, int64(4))
			case <-time.After(shortTimeout):
				t.Fatalf("Expected one block being cut")
			}

			close(haltChan)
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(2), counts[indexRecvPass], "Expected 2 message received and unmarshaled")
			assert.Equal(t, uint64(2), counts[indexProcessRegularPass], "Expected 2 REGULAR message processed")
		})

		t.Run("InvalidDiscard", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
				SequenceVal:         uint64(1),
				ProcessNormalMsgErr: fmt.Errorf("Invalid normal message"),
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

			mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(
				utils.MarshalOrPanic(newMockNormalEnvelope(t)),
				uint64(0),
				int64(0))))
			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut given invalid config message")
			case <-time.After(shortTimeout):
			}

close(haltChan) //与chain.halt（）相同
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 REGULAR message error")
		})

//此测试通过以下步骤重新提交路径：
//1）Kafka发出带有滞后配置序列的消息，同意者需要重新处理和
//重新提交邮件。但是，“waitready”不应为正常消息而被阻止
//2）在config seq被高级捕获的情况下，kafka将接收生产者消息。
//向上显示当前配置序列，并且originaloffset不为零以捕获
//以前从卡夫卡收到的同意书
//3）当同意人收到2）中提交的kafka消息时，其中config seq同步，
//它为它切了一个街区。
		t.Run("ValidResubmit", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			startChan := make(chan struct{})
			close(startChan)
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})
			doneReprocessing := make(chan struct{})
			close(doneReprocessing)

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
				SequenceVal:  uint64(1),
				ConfigSeqVal: uint64(1),
			}
			defer close(mockSupport.BlockCutterVal.Block)

			expectedKafkaMsgCh := make(chan *ab.KafkaMessage, 1)
			producer := mocks.NewSyncProducer(t, mockBrokerConfig)
			producer.ExpectSendMessageWithCheckerFunctionAndSucceed(func(val []byte) error {
				defer close(expectedKafkaMsgCh)

				expectedKafkaMsg := &ab.KafkaMessage{}
				if err := proto.Unmarshal(val, expectedKafkaMsg); err != nil {
					return err
				}

				regular := expectedKafkaMsg.GetRegular()
				if regular == nil {
					return fmt.Errorf("Expect message type to be regular")
				}

				if regular.ConfigSeq != mockSupport.Sequence() {
					return fmt.Errorf("Expect new config seq to be %d, got %d", mockSupport.Sequence(), regular.ConfigSeq)
				}

				if regular.OriginalOffset == 0 {
					return fmt.Errorf("Expect Original Offset to be non-zero if resubmission")
				}

				expectedKafkaMsgCh <- expectedKafkaMsg
				return nil
			})

			bareMinimumChain := &chainImpl{
				producer:        producer,
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				startChan:                      startChan,
				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
				doneReprocessingMsgInFlight:    doneReprocessing,
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

			mockSupport.BlockCutterVal.CutNext = true

			mpc.YieldMessage(newMockConsumerMessage(newNormalMessage(
				utils.MarshalOrPanic(newMockNormalEnvelope(t)),
				uint64(0),
				int64(0))))
			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut given invalid config message")
			case <-time.After(shortTimeout):
			}

//检查waitready是否未被正常类型的飞行中重新处理消息阻止。
			waitReady := make(chan struct{})
			go func() {
				bareMinimumChain.WaitReady()
				waitReady <- struct{}{}
			}()

			select {
			case <-waitReady:
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Expected WaitReady call to be unblock because all reprocessed messages are consumed")
			}

//发出同意者生成的卡夫卡消息
			select {
			case expectedKafkaMsg := <-expectedKafkaMsgCh:
				require.NotNil(t, expectedKafkaMsg)
				mpc.YieldMessage(newMockConsumerMessage(expectedKafkaMsg))
				mockSupport.BlockCutterVal.Block <- struct{}{}
			case <-time.After(shortTimeout):
				t.Fatalf("Expected to receive kafka message")
			}

			select {
			case <-mockSupport.Blocks:
			case <-time.After(shortTimeout):
				t.Fatalf("Expected one block being cut")
			}

close(haltChan) //与chain.halt（）相同
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(2), counts[indexRecvPass], "Expected 2 message received and unmarshaled")
			assert.Equal(t, uint64(2), counts[indexProcessRegularPass], "Expected 2 REGULAR message error")
		})
	})

	t.Run("Config", func(t *testing.T) {
//此测试允许卡夫卡发出模拟重新提交的消息，不需要重新处理。
//（通过设置originaloffset<=lastoriginaloffsetprocessed）
		t.Run("AlreadyProcessedDiscard", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)
			lastOriginalOffsetProcessed := int64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:                     mockChannel,
				ConsenterSupport:            mockSupport,
				lastCutBlockNumber:          lastCutBlockNumber,
				lastOriginalOffsetProcessed: lastOriginalOffsetProcessed,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

			mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(0), int64(2))))

			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut")
			case <-time.After(shortTimeout):
			}

			close(haltChan)
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 2 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
		})

//此测试模拟了非确定性情况，其中有人在偏移量x处重新提交了消息，
//然而，我们没有。在重新验证期间，我们认为该消息无效，但是有人
//否则视为有效，并重新提交。
		t.Run("Non-determinism", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			startChan := make(chan struct{})
			close(startChan)
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})
			doneReprocessing := make(chan struct{})
			close(doneReprocessing)

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
				SequenceVal:         uint64(1),
				ConfigSeqVal:        uint64(1),
				ProcessConfigMsgVal: newMockConfigEnvelope(),
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				lastResubmittedConfigOffset: int64(0),

				startChan:                      startChan,
				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
				doneReprocessingMsgInFlight:    doneReprocessing,
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//检查waitready是否在开始时被阻止
			blockIngressMsg(t, false, bareMinimumChain.WaitReady)

//消息应该重新验证，但被认为无效，因此我们不重新提交它
			mockSupport.ProcessConfigMsgErr = fmt.Errorf("invalid message found during revalidation")

//使用滞后的配置序列发出配置消息
			mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(
				utils.MarshalOrPanic(newMockConfigEnvelope()),
				uint64(0),
				int64(0))))
			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut")
			case <-time.After(shortTimeout):
			}

//检查waitready是否仍然没有被阻止，因为我们没有重新提交任何内容
			blockIngressMsg(t, false, bareMinimumChain.WaitReady)

//有人重新提交了我们认为无效的消息
//我们故意保持processconfigmsgerr不变，以便
//我们肯定不会遇到再验证的途径。
			mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(
				utils.MarshalOrPanic(newMockConfigEnvelope()),
				uint64(1),
				int64(5))))

			select {
			case block := <-mockSupport.Blocks:
				metadata, err := utils.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
				assert.NoError(t, err, "Failed to get metadata from block")
				kafkaMetadata := &ab.KafkaMetadata{}
				err = proto.Unmarshal(metadata.Value, kafkaMetadata)
				assert.NoError(t, err, "Failed to unmarshal metadata")

				assert.Equal(t, kafkaMetadata.LastResubmittedConfigOffset, int64(5), "LastResubmittedConfigOffset didn't catch up")
				assert.Equal(t, kafkaMetadata.LastOriginalOffsetProcessed, int64(5), "LastOriginalOffsetProcessed doesn't match")
			case <-time.After(shortTimeout):
				t.Fatalf("Expected one block being cut")
			}

close(haltChan) //与chain.halt（）相同
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(2), counts[indexRecvPass], "Expected 2 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 2 REGULAR message error")
		})

//这个测试让卡夫卡发出一个模拟的重新提交的消息，其config seq仍然落后。
		t.Run("ResubmittedMsgStillBehind", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			startChan := make(chan struct{})
			close(startChan)
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)
			lastOriginalOffsetProcessed := int64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
				SequenceVal:         uint64(2),
				ProcessConfigMsgVal: newMockConfigEnvelope(),
			}
			defer close(mockSupport.BlockCutterVal.Block)

			producer := mocks.NewSyncProducer(t, mockBrokerConfig)
			producer.ExpectSendMessageWithCheckerFunctionAndSucceed(func(val []byte) error {
				return nil
			})

			bareMinimumChain := &chainImpl{
				producer:        producer,
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:                     mockChannel,
				ConsenterSupport:            mockSupport,
				lastCutBlockNumber:          lastCutBlockNumber,
				lastOriginalOffsetProcessed: lastOriginalOffsetProcessed,

				startChan:                      startChan,
				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
				doneReprocessingMsgInFlight:    make(chan struct{}),
			}

//由于我们正在进行再处理，waitready应该在开始时阻塞
			blockIngressMsg(t, true, bareMinimumChain.WaitReady)

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

			mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(utils.MarshalOrPanic(newMockEnvelope("fooMessage")), uint64(1), int64(4))))
			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut")
			case <-time.After(shortTimeout):
			}

//waitready仍应阻止，因为重新提交的配置消息仍在当前配置序列之后
			blockIngressMsg(t, true, bareMinimumChain.WaitReady)

			close(haltChan)
			logger.Debug("haltChan closed")
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessRegularPass], "Expected 1 REGULAR message processed")
		})

		t.Run("InvalidDiscard", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
				SequenceVal:         uint64(1),
				ProcessConfigMsgErr: fmt.Errorf("Invalid config message"),
			}
			defer close(mockSupport.BlockCutterVal.Block)

			bareMinimumChain := &chainImpl{
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

			mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(
				utils.MarshalOrPanic(newMockNormalEnvelope(t)),
				uint64(0),
				int64(0))))
			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut given invalid config message")
			case <-time.After(shortTimeout):
			}

close(haltChan) //与chain.halt（）相同
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(1), counts[indexRecvPass], "Expected 1 message received and unmarshaled")
			assert.Equal(t, uint64(1), counts[indexProcessRegularError], "Expected 1 REGULAR message error")
		})

//此测试通过以下步骤重新提交路径：
//1）Kafka发出带有滞后配置序列的消息，同意者需要重新处理和
//重新提交邮件，并阻止“waitready”api
//2）在config seq被高级捕获的情况下，kafka将接收生产者消息。
//向上显示当前配置序列，并且originaloffset不为零以捕获
//以前从卡夫卡收到的同意书
//3）当同意人收到2）中提交的kafka消息时，其中config seq同步，
//它为它切了一个街区，然后在“waitready”上提起街区。
		t.Run("ValidResubmit", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			startChan := make(chan struct{})
			close(startChan)
			errorChan := make(chan struct{})
			close(errorChan)
			haltChan := make(chan struct{})
			doneReprocessing := make(chan struct{})
			close(doneReprocessing)

			lastCutBlockNumber := uint64(3)

			mockSupport := &mockmultichannel.ConsenterSupport{
Blocks:         make(chan *cb.Block), //WriteBlock将在此处发布
				BlockCutterVal: mockblockcutter.NewReceiver(),
				ChainIDVal:     mockChannel.topic(),
HeightVal:      lastCutBlockNumber, //在WRITEBLOCK调用期间递增
				SharedConfigVal: &mockconfig.Orderer{
					BatchTimeoutVal: longTimeout,
					CapabilitiesVal: &mockconfig.OrdererCapabilities{
						ResubmissionVal: true,
					},
				},
				SequenceVal:         uint64(1),
				ConfigSeqVal:        uint64(1),
				ProcessConfigMsgVal: newMockConfigEnvelope(),
			}
			defer close(mockSupport.BlockCutterVal.Block)

			expectedKafkaMsgCh := make(chan *ab.KafkaMessage, 1)
			producer := mocks.NewSyncProducer(t, mockBrokerConfig)
			producer.ExpectSendMessageWithCheckerFunctionAndSucceed(func(val []byte) error {
				defer close(expectedKafkaMsgCh)

				expectedKafkaMsg := &ab.KafkaMessage{}
				if err := proto.Unmarshal(val, expectedKafkaMsg); err != nil {
					return err
				}

				regular := expectedKafkaMsg.GetRegular()
				if regular == nil {
					return fmt.Errorf("Expect message type to be regular")
				}

				if regular.ConfigSeq != mockSupport.Sequence() {
					return fmt.Errorf("Expect new config seq to be %d, got %d", mockSupport.Sequence(), regular.ConfigSeq)
				}

				if regular.OriginalOffset == 0 {
					return fmt.Errorf("Expect Original Offset to be non-zero if resubmission")
				}

				expectedKafkaMsgCh <- expectedKafkaMsg
				return nil
			})

			bareMinimumChain := &chainImpl{
				producer:        producer,
				parentConsumer:  mockParentConsumer,
				channelConsumer: mockChannelConsumer,

				channel:            mockChannel,
				ConsenterSupport:   mockSupport,
				lastCutBlockNumber: lastCutBlockNumber,

				startChan:                      startChan,
				errorChan:                      errorChan,
				haltChan:                       haltChan,
				doneProcessingMessagesToBlocks: make(chan struct{}),
				doneReprocessingMsgInFlight:    doneReprocessing,
			}

			var counts []uint64
			done := make(chan struct{})

			go func() {
				counts, err = bareMinimumChain.processMessagesToBlocks()
				done <- struct{}{}
			}()

//检查waitready是否在开始时被阻止
			blockIngressMsg(t, false, bareMinimumChain.WaitReady)

//使用滞后的配置序列发出配置消息
			mpc.YieldMessage(newMockConsumerMessage(newConfigMessage(
				utils.MarshalOrPanic(newMockConfigEnvelope()),
				uint64(0),
				int64(0))))
			select {
			case <-mockSupport.Blocks:
				t.Fatalf("Expected no block being cut given lagged config message")
			case <-time.After(shortTimeout):
			}

//检查waitready是否因飞行中重新处理的消息而被实际阻止
			blockIngressMsg(t, true, bareMinimumChain.WaitReady)

			select {
			case expectedKafkaMsg := <-expectedKafkaMsgCh:
				require.NotNil(t, expectedKafkaMsg)
//发出同意者生成的卡夫卡消息
				mpc.YieldMessage(newMockConsumerMessage(expectedKafkaMsg))
			case <-time.After(shortTimeout):
				t.Fatalf("Expected to receive kafka message")
			}

			select {
			case <-mockSupport.Blocks:
			case <-time.After(shortTimeout):
				t.Fatalf("Expected one block being cut")
			}

//现在应该取消阻止“waitready”
			blockIngressMsg(t, false, bareMinimumChain.WaitReady)

close(haltChan) //与chain.halt（）相同
			<-done

			assert.NoError(t, err, "Expected the processMessagesToBlocks call to return without errors")
			assert.Equal(t, uint64(2), counts[indexRecvPass], "Expected 2 message received and unmarshaled")
			assert.Equal(t, uint64(2), counts[indexProcessRegularPass], "Expected 2 REGULAR message error")
		})
	})
}

//这里是测试助手函数。

func newRegularMessage(payload []byte) *ab.KafkaMessage {
	return &ab.KafkaMessage{
		Type: &ab.KafkaMessage_Regular{
			Regular: &ab.KafkaMessageRegular{
				Payload: payload,
			},
		},
	}
}

func newMockNormalEnvelope(t *testing.T) *cb.Envelope {
	return &cb.Envelope{Payload: utils.MarshalOrPanic(&cb.Payload{
		Header: &cb.Header{ChannelHeader: utils.MarshalOrPanic(
			&cb.ChannelHeader{Type: int32(cb.HeaderType_MESSAGE), ChannelId: channelNameForTest(t)})},
		Data: []byte("Foo"),
	})}
}

func newMockConfigEnvelope() *cb.Envelope {
	return &cb.Envelope{Payload: utils.MarshalOrPanic(&cb.Payload{
		Header: &cb.Header{ChannelHeader: utils.MarshalOrPanic(
			&cb.ChannelHeader{Type: int32(cb.HeaderType_CONFIG), ChannelId: "foo"})},
		Data: utils.MarshalOrPanic(&cb.ConfigEnvelope{}),
	})}
}

func newMockOrdererTxEnvelope() *cb.Envelope {
	return &cb.Envelope{Payload: utils.MarshalOrPanic(&cb.Payload{
		Header: &cb.Header{ChannelHeader: utils.MarshalOrPanic(
			&cb.ChannelHeader{Type: int32(cb.HeaderType_ORDERER_TRANSACTION), ChannelId: "foo"})},
		Data: utils.MarshalOrPanic(newMockConfigEnvelope()),
	})}
}

func TestDeliverSession(t *testing.T) {

	type testEnvironment struct {
		channelID  string
		topic      string
		partition  int32
		height     int64
		nextOffset int64
		support    *mockConsenterSupport
		broker0    *sarama.MockBroker
		broker1    *sarama.MockBroker
		broker2    *sarama.MockBroker
		testMsg    sarama.Encoder
	}

//初始化测试环境
	newTestEnvironment := func(t *testing.T) *testEnvironment {

		channelID := channelNameForTest(t)
		topic := channelID
		partition := int32(defaultPartition)
		height := int64(100)
		nextOffset := height + 1
		broker0 := sarama.NewMockBroker(t, 0)
		broker1 := sarama.NewMockBroker(t, 1)
		broker2 := sarama.NewMockBroker(t, 2)

//broker0将输入有关其他代理和分区负责人的信息
		broker0.SetHandlerByMap(map[string]sarama.MockResponse{
			"MetadataRequest": sarama.NewMockMetadataResponse(t).
				SetBroker(broker1.Addr(), broker1.BrokerID()).
				SetBroker(broker2.Addr(), broker2.BrokerID()).
				SetLeader(topic, partition, broker1.BrokerID()),
		})

//使用启动所需的响应配置broker1
		broker1.SetHandlerByMap(map[string]sarama.MockResponse{
//连接ProduceRequest
			"ProduceRequest": sarama.NewMockProduceResponse(t).
				SetError(topic, partition, sarama.ErrNoError),
//响应主题偏移请求
			"OffsetRequest": sarama.NewMockOffsetResponse(t).
				SetOffset(topic, partition, sarama.OffsetOldest, 0).
				SetOffset(topic, partition, sarama.OffsetNewest, nextOffset),
//启动时用空响应响应响应提取请求
			"FetchRequest": sarama.NewMockFetchResponse(t, 1),
		})

//使用默认提取请求响应配置代理2
		broker2.SetHandlerByMap(map[string]sarama.MockResponse{
//启动时用空响应响应响应提取请求
			"FetchRequest": sarama.NewMockFetchResponse(t, 1),
		})

//设置模拟块切割机
		blockcutter := &mockReceiver{}
		blockcutter.On("Ordered", mock.Anything).Return([][]*cb.Envelope{{&cb.Envelope{}}}, false)

//设置模拟链支持和模拟方法调用
		support := &mockConsenterSupport{}
		support.On("Height").Return(uint64(height))
		support.On("ChainID").Return(topic)
		support.On("Sequence").Return(uint64(0))
		support.On("SharedConfig").Return(&mockconfig.Orderer{KafkaBrokersVal: []string{broker0.Addr()}})
		support.On("ClassifyMsg", mock.Anything).Return(msgprocessor.NormalMsg, nil)
		support.On("ProcessNormalMsg", mock.Anything).Return(uint64(0), nil)
		support.On("BlockCutter").Return(blockcutter)
		support.On("CreateNextBlock", mock.Anything).Return(&cb.Block{})

//模拟代理将返回的测试消息
		testMsg := sarama.ByteEncoder(utils.MarshalOrPanic(
			newRegularMessage(utils.MarshalOrPanic(&cb.Envelope{
				Payload: utils.MarshalOrPanic(&cb.Payload{
					Header: &cb.Header{
						ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{
							ChannelId: topic,
						}),
					},
					Data: []byte("TEST_DATA"),
				})})),
		))

		return &testEnvironment{
			channelID:  channelID,
			topic:      topic,
			partition:  partition,
			height:     height,
			nextOffset: nextOffset,
			support:    support,
			broker0:    broker0,
			broker1:    broker1,
			broker2:    broker2,
			testMsg:    testMsg,
		}
	}

//brokerPath模拟分区领导死亡和
//在交付会话超时之前成为领导者的第二个代理。
	t.Run("BrokerDeath", func(t *testing.T) {

//初始化测试环境
		env := newTestEnvironment(t)

//经纪人1将在测试中关闭
		defer env.broker0.Close()
		defer env.broker2.Close()

//初始化同意者
		consenter, _ := New(mockLocalConfig.Kafka, &lmock.MetricsProvider{})

//初始化链
		metadata := &cb.Metadata{Value: utils.MarshalOrPanic(&ab.KafkaMetadata{LastOffsetPersisted: env.height})}
		chain, err := consenter.HandleChain(env.support, metadata)
		if err != nil {
			t.Fatal(err)
		}

//启动链条，等待它稳定下来。
		chain.Start()
		select {
		case <-chain.(*chainImpl).startChan:
			logger.Debug("chain started")
		case <-time.After(shortTimeout):
			t.Fatal("chain should have started by now")
		}

//直接阻止到此频道
		blocks := make(chan *cb.Block, 1)
		env.support.On("WriteBlock", mock.Anything, mock.Anything).Return().Run(func(arg1 mock.Arguments) {
			blocks <- arg1.Get(0).(*cb.Block)
		})

//从代理1发送一些消息
		fetchResponse1 := sarama.NewMockFetchResponse(t, 1)
		for i := 0; i < 5; i++ {
			fetchResponse1.SetMessage(env.topic, env.partition, env.nextOffset, env.testMsg)
			env.nextOffset++
		}
		env.broker1.SetHandlerByMap(map[string]sarama.MockResponse{
			"FetchRequest": fetchResponse1,
		})

		logger.Debug("Waiting for messages from broker1")
		for i := 0; i < 5; i++ {
			select {
			case <-blocks:
			case <-time.After(shortTimeout):
				t.Fatalf("timed out waiting for messages (receieved %d messages)", i)
			}
		}

//准备broker2发送一些消息
		fetchResponse2 := sarama.NewMockFetchResponse(t, 1)
		for i := 0; i < 5; i++ {
			fetchResponse2.SetMessage(env.topic, env.partition, env.nextOffset, env.testMsg)
			env.nextOffset++
		}

		env.broker2.SetHandlerByMap(map[string]sarama.MockResponse{
			"ProduceRequest": sarama.NewMockProduceResponse(t).
				SetError(env.topic, env.partition, sarama.ErrNoError),
			"FetchRequest": fetchResponse2,
		})

//停机断路器1
		env.broker1.Close()

//准备broker0回应broker2现在是领导者
		env.broker0.SetHandlerByMap(map[string]sarama.MockResponse{
			"MetadataRequest": sarama.NewMockMetadataResponse(t).
				SetLeader(env.topic, env.partition, env.broker2.BrokerID()),
		})

		logger.Debug("Waiting for messages from broker2")
		for i := 0; i < 5; i++ {
			select {
			case <-blocks:
			case <-time.After(shortTimeout):
				t.Fatalf("timed out waiting for messages (receieved %d messages)", i)
			}
		}

		chain.Halt()
	})

//erroffsetoutofrange不可恢复
	t.Run("ErrOffsetOutOfRange", func(t *testing.T) {

//初始化测试环境
		env := newTestEnvironment(t)

//经纪人清理
		defer env.broker2.Close()
		defer env.broker1.Close()
		defer env.broker0.Close()

//初始化同意者
		consenter, _ := New(mockLocalConfig.Kafka, &lmock.MetricsProvider{})

//初始化链
		metadata := &cb.Metadata{Value: utils.MarshalOrPanic(&ab.KafkaMetadata{LastOffsetPersisted: env.height})}
		chain, err := consenter.HandleChain(env.support, metadata)
		if err != nil {
			t.Fatal(err)
		}

//启动链条，等待它稳定下来。
		chain.Start()
		select {
		case <-chain.(*chainImpl).startChan:
			logger.Debug("chain started")
		case <-time.After(shortTimeout):
			t.Fatal("chain should have started by now")
		}

//直接阻止到此频道
		blocks := make(chan *cb.Block, 1)
		env.support.On("WriteBlock", mock.Anything, mock.Anything).Return().Run(func(arg1 mock.Arguments) {
			blocks <- arg1.Get(0).(*cb.Block)
		})

//将broker1设置为响应两个fetch请求：
//-第一个提取请求将得到一个erroffsetoutofrange错误响应。
//-第二个提取请求将得到有效（即非错误）响应。
		fetchResponse := &sarama.FetchResponse{}
		fetchResponse.AddError(env.topic, env.partition, sarama.ErrOffsetOutOfRange)
		fetchResponse.AddMessage(env.topic, env.partition, nil, env.testMsg, env.nextOffset)
		env.nextOffset++
		env.broker1.SetHandlerByMap(map[string]sarama.MockResponse{
			"FetchRequest": sarama.NewMockWrapper(fetchResponse),
//连接消息的答案
			"ProduceRequest": sarama.NewMockProduceResponse(t).
				SetError(env.topic, env.partition, sarama.ErrNoError),
		})

		select {
		case <-blocks:
//不应提取有效的提取响应
			t.Fatal("Did not expect new blocks")
		case <-time.After(shortTimeout):
			t.Fatal("Errored() should have closed by now")
		case <-chain.Errored():
		}

		chain.Halt()
	})

//测试链超时
	t.Run("DeliverSessionTimedOut", func(t *testing.T) {

//初始化测试环境
		env := newTestEnvironment(t)

//经纪人清理
		defer env.broker2.Close()
		defer env.broker1.Close()
		defer env.broker0.Close()

//初始化同意者
		consenter, _ := New(mockLocalConfig.Kafka, &lmock.MetricsProvider{})

//初始化链
		metadata := &cb.Metadata{Value: utils.MarshalOrPanic(&ab.KafkaMetadata{LastOffsetPersisted: env.height})}
		chain, err := consenter.HandleChain(env.support, metadata)
		if err != nil {
			t.Fatal(err)
		}

//启动链条，等待它稳定下来。
		chain.Start()
		select {
		case <-chain.(*chainImpl).startChan:
			logger.Debug("chain started")
		case <-time.After(shortTimeout):
			t.Fatal("chain should have started by now")
		}

//直接阻止到此频道
		blocks := make(chan *cb.Block, 1)
		env.support.On("WriteBlock", mock.Anything, mock.Anything).Return().Run(func(arg1 mock.Arguments) {
			blocks <- arg1.Get(0).(*cb.Block)
		})

		metadataResponse := new(sarama.MetadataResponse)
		metadataResponse.AddTopicPartition(env.topic, env.partition, -1, []int32{}, []int32{}, sarama.ErrBrokerNotAvailable)

//将seed broker配置为在元数据请求时返回错误，否则
//使用者客户端将继续成功地“订阅”主题/分区
		env.broker0.SetHandlerByMap(map[string]sarama.MockResponse{
			"MetadataRequest": sarama.NewMockWrapper(metadataResponse),
		})

//将broker1设置为返回错误。
//请注意，以下不是来自Sarama客户的错误
//消费者的观点：
//-错误未知的吸收
//-不合格铅吸收
//-错误领导不可用
//-可复制的错误：
		fetchResponse := &sarama.FetchResponse{}
		fetchResponse.AddError(env.topic, env.partition, sarama.ErrUnknown)
		env.broker1.SetHandlerByMap(map[string]sarama.MockResponse{
			"FetchRequest": sarama.NewMockWrapper(fetchResponse),
//连接消息的答案
			"ProduceRequest": sarama.NewMockProduceResponse(t).
				SetError(env.topic, env.partition, sarama.ErrNoError),
		})

		select {
		case <-blocks:
			t.Fatal("Did not expect new blocks")
		case <-time.After(mockRetryOptions.NetworkTimeouts.ReadTimeout + shortTimeout):
			t.Fatal("Errored() should have closed by now")
		case <-chain.Errored():
			t.Log("Errored() closed")
		}

		chain.Halt()
	})

}

type mockReceiver struct {
	mock.Mock
}

func (r *mockReceiver) Ordered(msg *cb.Envelope) (messageBatches [][]*cb.Envelope, pending bool) {
	args := r.Called(msg)
	return args.Get(0).([][]*cb.Envelope), args.Bool(1)
}

func (r *mockReceiver) Cut() []*cb.Envelope {
	args := r.Called()
	return args.Get(0).([]*cb.Envelope)
}

type mockConsenterSupport struct {
	mock.Mock
}

func (c *mockConsenterSupport) Block(seq uint64) *cb.Block {
	return nil
}

func (c *mockConsenterSupport) VerifyBlockSignature([]*cb.SignedData, *cb.ConfigEnvelope) error {
	return nil
}

func (c *mockConsenterSupport) NewSignatureHeader() (*cb.SignatureHeader, error) {
	args := c.Called()
	return args.Get(0).(*cb.SignatureHeader), args.Error(1)
}

func (c *mockConsenterSupport) Sign(message []byte) ([]byte, error) {
	args := c.Called(message)
	return args.Get(0).([]byte), args.Error(1)
}

func (c *mockConsenterSupport) ClassifyMsg(chdr *cb.ChannelHeader) msgprocessor.Classification {
	args := c.Called(chdr)
	return args.Get(0).(msgprocessor.Classification)
}

func (c *mockConsenterSupport) ProcessNormalMsg(env *cb.Envelope) (configSeq uint64, err error) {
	args := c.Called(env)
	return args.Get(0).(uint64), args.Error(1)
}

func (c *mockConsenterSupport) ProcessConfigUpdateMsg(env *cb.Envelope) (config *cb.Envelope, configSeq uint64, err error) {
	args := c.Called(env)
	return args.Get(0).(*cb.Envelope), args.Get(1).(uint64), args.Error(2)
}

func (c *mockConsenterSupport) ProcessConfigMsg(env *cb.Envelope) (config *cb.Envelope, configSeq uint64, err error) {
	args := c.Called(env)
	return args.Get(0).(*cb.Envelope), args.Get(1).(uint64), args.Error(2)
}

func (c *mockConsenterSupport) BlockCutter() blockcutter.Receiver {
	args := c.Called()
	return args.Get(0).(blockcutter.Receiver)
}

func (c *mockConsenterSupport) SharedConfig() channelconfig.Orderer {
	args := c.Called()
	return args.Get(0).(channelconfig.Orderer)
}

func (c *mockConsenterSupport) CreateNextBlock(messages []*cb.Envelope) *cb.Block {
	args := c.Called(messages)
	return args.Get(0).(*cb.Block)
}

func (c *mockConsenterSupport) WriteBlock(block *cb.Block, encodedMetadataValue []byte) {
	c.Called(block, encodedMetadataValue)
	return
}

func (c *mockConsenterSupport) WriteConfigBlock(block *cb.Block, encodedMetadataValue []byte) {
	c.Called(block, encodedMetadataValue)
	return
}

func (c *mockConsenterSupport) Sequence() uint64 {
	args := c.Called()
	return args.Get(0).(uint64)
}

func (c *mockConsenterSupport) ChainID() string {
	args := c.Called()
	return args.String(0)
}

func (c *mockConsenterSupport) Height() uint64 {
	args := c.Called()
	return args.Get(0).(uint64)
}
