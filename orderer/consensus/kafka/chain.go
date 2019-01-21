
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
	"strconv"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/protobuf/proto"
	localconfig "github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/common/msgprocessor"
	"github.com/hyperledger/fabric/orderer/consensus"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
)

//用于捕获度量--请参阅processmessagestoblocks
const (
	indexRecvError = iota
	indexUnmarshalError
	indexRecvPass
	indexProcessConnectPass
	indexProcessTimeToCutError
	indexProcessTimeToCutPass
	indexProcessRegularError
	indexProcessRegularPass
	indexSendTimeToCutError
	indexSendTimeToCutPass
	indexExitChanPass
)

func newChain(
	consenter commonConsenter,
	support consensus.ConsenterSupport,
	lastOffsetPersisted int64,
	lastOriginalOffsetProcessed int64,
	lastResubmittedConfigOffset int64,
) (*chainImpl, error) {
	lastCutBlockNumber := getLastCutBlockNumber(support.Height())
	logger.Infof("[channel: %s] Starting chain with last persisted offset %d and last recorded block %d",
		support.ChainID(), lastOffsetPersisted, lastCutBlockNumber)

	doneReprocessingMsgInFlight := make(chan struct{})
//在以下任何一种情况下，我们都应该取消阻止入口消息：
//-lastSubmittedConfigOffset==0，其中我们从未重新提交任何配置消息
//-lastSubmittedConfigOffset==lastOriginalOffsetProcessed，其中我们重新提交的最新配置消息
//已经处理过了
//-lastSubmittedConfigOffset<lastOriginalOffsetProcessed，其中我们已经处理了一个或多个重新提交的
//最新重新提交配置消息后的正常消息。（我们前进'lastsubmittedconfigoffset'以
//配置消息，但不是普通消息）
	if lastResubmittedConfigOffset == 0 || lastResubmittedConfigOffset <= lastOriginalOffsetProcessed {
//如果我们已经赶上重新处理重新提交的消息，请关闭频道以取消阻止广播
		close(doneReprocessingMsgInFlight)
	}

	return &chainImpl{
		consenter:                   consenter,
		ConsenterSupport:            support,
		channel:                     newChannel(support.ChainID(), defaultPartition),
		lastOffsetPersisted:         lastOffsetPersisted,
		lastOriginalOffsetProcessed: lastOriginalOffsetProcessed,
		lastResubmittedConfigOffset: lastResubmittedConfigOffset,
		lastCutBlockNumber:          lastCutBlockNumber,

		haltChan:                    make(chan struct{}),
		startChan:                   make(chan struct{}),
		doneReprocessingMsgInFlight: doneReprocessingMsgInFlight,
	}, nil
}

type chainImpl struct {
	consenter commonConsenter
	consensus.ConsenterSupport

	channel                     channel
	lastOffsetPersisted         int64
	lastOriginalOffsetProcessed int64
	lastResubmittedConfigOffset int64
	lastCutBlockNumber          uint64

	producer        sarama.SyncProducer
	parentConsumer  sarama.Consumer
	channelConsumer sarama.PartitionConsumer

//更改DoneReprocessingMsginFlight时使用的互斥体
	doneReprocessingMutex sync.Mutex
//有飞行信息需要等待的通知
	doneReprocessingMsgInFlight chan struct{}

//当分区使用者出错时，关闭通道。否则，使
//这是一个开放的，没有缓冲的通道。
	errorChan chan struct{}
//当一个halt（）请求出现时，关闭通道。与ErrorChan不同的是，
//通道关闭时不得重新打开。它的关闭会触发
//processmessagestoblock循环。
	haltChan chan struct{}
//通知链已停止将消息处理为块
	doneProcessingMessagesToBlocks chan struct{}
//启动中的可重试步骤完成后关闭。
	startChan chan struct{}
//计时器控制将挂起消息剪切到块中的批处理超时
	timer <-chan time.Time
}

//出错返回一个通道，当分区使用者出错时该通道将关闭。
//已经发生了。由deliver（）检查。
func (chain *chainImpl) Errored() <-chan struct{} {
	select {
	case <-chain.startChan:
		return chain.errorChan
	default:
//在同意者启动时，始终返回错误
		dummyError := make(chan struct{})
		close(dummyError)
		return dummyError
	}
}

//Start为保持最新状态分配必要的资源
//链。实现共识链接口。被称为
//consumeration.newManagerImpl（），当排序过程为
//在调用newserver（）之前启动。启动Goroutine以避免
//阻止共识管理器。
func (chain *chainImpl) Start() {
	go startThread(chain)
}

//HALT释放为此链分配的资源。实现
//共识。链接口。
func (chain *chainImpl) Halt() {
	select {
	case <-chain.startChan:
//链条启动完毕，我们可以停止它。
		select {
		case <-chain.haltChan:
//此构造非常有用，因为它允许调用halt（）。
//多次（通过一个线程）没有恐慌。瑞卡尔说
//从关闭的通道接收立即返回（零值）。
			logger.Warningf("[channel: %s] Halting of chain requested again", chain.ChainID())
		default:
			logger.Criticalf("[channel: %s] Halting of chain requested", chain.ChainID())
//链条静停
			close(chain.haltChan)
//等待消息处理到块以完成关闭
			<-chain.doneProcessingMessagesToBlocks
//关闭卡夫卡生产商和消费者
			chain.closeKafkaObjects()
			logger.Debugf("[channel: %s] Closed the haltChan", chain.ChainID())
		}
	default:
		logger.Warningf("[channel: %s] Waiting for chain to finish starting before halting", chain.ChainID())
		<-chain.startChan
		chain.Halt()
	}
}

func (chain *chainImpl) WaitReady() error {
	select {
case <-chain.startChan: //启动阶段已完成
		select {
case <-chain.haltChan: //链条已经停了，停在这里
			return fmt.Errorf("consenter for this channel has been halted")
case <-chain.doneReprocessing(): //阻止等待所有重新提交的邮件重新处理
			return nil
		}
default: //还没有准备好
		return fmt.Errorf("backing Kafka cluster has not completed booting; try again later")
	}
}

func (chain *chainImpl) doneReprocessing() <-chan struct{} {
	chain.doneReprocessingMutex.Lock()
	defer chain.doneReprocessingMutex.Unlock()
	return chain.doneReprocessingMsgInFlight
}

func (chain *chainImpl) reprocessConfigComplete() {
	chain.doneReprocessingMutex.Lock()
	defer chain.doneReprocessingMutex.Unlock()
	close(chain.doneReprocessingMsgInFlight)
}

func (chain *chainImpl) reprocessConfigPending() {
	chain.doneReprocessingMutex.Lock()
	defer chain.doneReprocessingMutex.Unlock()
	chain.doneReprocessingMsgInFlight = make(chan struct{})
}

//实现共识链接口。由广播（）调用。
func (chain *chainImpl) Order(env *cb.Envelope, configSeq uint64) error {
	return chain.order(env, configSeq, int64(0))
}

func (chain *chainImpl) order(env *cb.Envelope, configSeq uint64, originalOffset int64) error {
	marshaledEnv, err := utils.Marshal(env)
	if err != nil {
		return fmt.Errorf("cannot enqueue, unable to marshal envelope because = %s", err)
	}
	if !chain.enqueue(newNormalMessage(marshaledEnv, configSeq, originalOffset)) {
		return fmt.Errorf("cannot enqueue")
	}
	return nil
}

//实现共识链接口。由广播（）调用。
func (chain *chainImpl) Configure(config *cb.Envelope, configSeq uint64) error {
	return chain.configure(config, configSeq, int64(0))
}

func (chain *chainImpl) configure(config *cb.Envelope, configSeq uint64, originalOffset int64) error {
	marshaledConfig, err := utils.Marshal(config)
	if err != nil {
		return fmt.Errorf("cannot enqueue, unable to marshal config because %s", err)
	}
	if !chain.enqueue(newConfigMessage(marshaledConfig, configSeq, originalOffset)) {
		return fmt.Errorf("cannot enqueue")
	}
	return nil
}

//排队接受消息并在接受时返回true，或返回false otheriwse。
func (chain *chainImpl) enqueue(kafkaMsg *ab.KafkaMessage) bool {
	logger.Debugf("[channel: %s] Enqueueing envelope...", chain.ChainID())
	select {
case <-chain.startChan: //启动阶段已完成
		select {
case <-chain.haltChan: //链条已经停了，停在这里
			logger.Warningf("[channel: %s] consenter for this channel has been halted", chain.ChainID())
			return false
default: //邮路
			payload, err := utils.Marshal(kafkaMsg)
			if err != nil {
				logger.Errorf("[channel: %s] unable to marshal Kafka message because = %s", chain.ChainID(), err)
				return false
			}
			message := newProducerMessage(chain.channel, payload)
			if _, _, err = chain.producer.SendMessage(message); err != nil {
				logger.Errorf("[channel: %s] cannot enqueue envelope because = %s", chain.ChainID(), err)
				return false
			}
			logger.Debugf("[channel: %s] Envelope enqueued successfully", chain.ChainID())
			return true
		}
default: //还没有准备好
		logger.Warningf("[channel: %s] Will not enqueue, consenter for this channel hasn't started yet", chain.ChainID())
		return false
	}
}

//由Start（）调用。
func startThread(chain *chainImpl) {
	var err error

//如果不存在则创建主题（需要kafka v0.10.1.0）
	err = setupTopicForChannel(chain.consenter.retryOptions(), chain.haltChan, chain.SharedConfig().KafkaBrokers(), chain.consenter.brokerConfig(), chain.consenter.topicDetail(), chain.channel)
	if err != nil {
//立即登录并回退到自动创建代理主题设置
		logger.Infof("[channel: %s]: failed to create Kafka topic = %s", chain.channel.topic(), err)
	}

//设置生产商
	chain.producer, err = setupProducerForChannel(chain.consenter.retryOptions(), chain.haltChan, chain.SharedConfig().KafkaBrokers(), chain.consenter.brokerConfig(), chain.channel)
	if err != nil {
		logger.Panicf("[channel: %s] Cannot set up producer = %s", chain.channel.topic(), err)
	}
	logger.Infof("[channel: %s] Producer set up successfully", chain.ChainID())

//让生产商发布连接消息
	if err = sendConnectMessage(chain.consenter.retryOptions(), chain.haltChan, chain.producer, chain.channel); err != nil {
		logger.Panicf("[channel: %s] Cannot post CONNECT message = %s", chain.channel.topic(), err)
	}
	logger.Infof("[channel: %s] CONNECT message posted successfully", chain.channel.topic())

//设置父使用者
	chain.parentConsumer, err = setupParentConsumerForChannel(chain.consenter.retryOptions(), chain.haltChan, chain.SharedConfig().KafkaBrokers(), chain.consenter.brokerConfig(), chain.channel)
	if err != nil {
		logger.Panicf("[channel: %s] Cannot set up parent consumer = %s", chain.channel.topic(), err)
	}
	logger.Infof("[channel: %s] Parent consumer set up successfully", chain.channel.topic())

//设置频道消费者
	chain.channelConsumer, err = setupChannelConsumerForChannel(chain.consenter.retryOptions(), chain.haltChan, chain.parentConsumer, chain.channel, chain.lastOffsetPersisted+1)
	if err != nil {
		logger.Panicf("[channel: %s] Cannot set up channel consumer = %s", chain.channel.topic(), err)
	}
	logger.Infof("[channel: %s] Channel consumer set up successfully", chain.channel.topic())

	chain.doneProcessingMessagesToBlocks = make(chan struct{})

chain.errorChan = make(chan struct{}) //交付请求也将通过
close(chain.startChan)                //广播请求现在将通过

	logger.Infof("[channel: %s] Start phase completed successfully", chain.channel.topic())

chain.processMessagesToBlocks() //与频道保持同步
}

//processmessagestoblocks为给定通道排出Kafka使用者，以及
//负责将有序消息流转换为
//频道分类帐。
func (chain *chainImpl) processMessagesToBlocks() ([]uint64, error) {
counts := make([]uint64, 11) //用于度量和测试
	msg := new(ab.KafkaMessage)

	defer func() {
//通知我们没有处理要阻止的消息
		close(chain.doneProcessingMessagesToBlocks)
	}()

defer func() { //当调用halt（）时
		select {
case <-chain.errorChan: //如果已经关闭，不要做任何事情
		default:
			close(chain.errorChan)
		}
	}()

	subscription := fmt.Sprintf("added subscription to %s/%d", chain.channel.topic(), chain.channel.partition())
	var topicPartitionSubscriptionResumed <-chan string
	var deliverSessionTimer *time.Timer
	var deliverSessionTimedOut <-chan time.Time

	for {
		select {
		case <-chain.haltChan:
			logger.Warningf("[channel: %s] Consenter for channel exiting", chain.ChainID())
			counts[indexExitChanPass]++
			return counts, nil
		case kafkaErr := <-chain.channelConsumer.Errors():
			logger.Errorf("[channel: %s] Error during consumption: %s", chain.ChainID(), kafkaErr)
			counts[indexRecvError]++
			select {
case <-chain.errorChan: //如果已经关闭，不要做任何事情
			default:

				switch kafkaErr.Err {
				case sarama.ErrOffsetOutOfRange:
//除erroffsetoutofrange之外，Kafka使用者将自动重试所有错误。
					logger.Errorf("[channel: %s] Unrecoverable error during consumption: %s", chain.ChainID(), kafkaErr)
					close(chain.errorChan)
				default:
					if topicPartitionSubscriptionResumed == nil {
//注册侦听器
						topicPartitionSubscriptionResumed = saramaLogger.NewListener(subscription)
//启动会话超时计时器
						deliverSessionTimer = time.NewTimer(chain.consenter.retryOptions().NetworkTimeouts.ReadTimeout)
						deliverSessionTimedOut = deliverSessionTimer.C
					}
				}
			}
			select {
case <-chain.errorChan: //我们不会忽略错误
				logger.Warningf("[channel: %s] Closed the errorChan", chain.ChainID())
//这涵盖了边缘情况，其中（1）消耗错误
//关闭了错误通道，因此使链不可用于
//交付客户，（2）我们已经在最新的抵消，以及（3）
//没有新的广播请求进入。在这种情况下，
//没有触发器可以重新创建ErrorChan，并且
//将链条标记为可用，因此我们必须通过
//发出连接信息。TODO考虑速率限制
				go sendConnectMessage(chain.consenter.retryOptions(), chain.haltChan, chain.producer, chain.channel)
default: //我们忽略了这个错误
				logger.Warningf("[channel: %s] Deliver sessions will be dropped if consumption errors continue.", chain.ChainID())
			}
		case <-topicPartitionSubscriptionResumed:
//停止侦听订阅消息
			saramaLogger.RemoveListener(subscription, topicPartitionSubscriptionResumed)
//禁用订阅事件chan
			topicPartitionSubscriptionResumed = nil

//停止超时计时器
			if !deliverSessionTimer.Stop() {
				<-deliverSessionTimer.C
			}
			logger.Warningf("[channel: %s] Consumption will resume.", chain.ChainID())

		case <-deliverSessionTimedOut:
//停止侦听订阅消息
			saramaLogger.RemoveListener(subscription, topicPartitionSubscriptionResumed)
//禁用订阅事件chan
			topicPartitionSubscriptionResumed = nil

			close(chain.errorChan)
			logger.Warningf("[channel: %s] Closed the errorChan", chain.ChainID())

//通过连接消息触发器使链再次可用
			go sendConnectMessage(chain.consenter.retryOptions(), chain.haltChan, chain.producer, chain.channel)

		case in, ok := <-chain.channelConsumer.Messages():
			if !ok {
				logger.Criticalf("[channel: %s] Kafka consumer closed.", chain.ChainID())
				return counts, nil
			}

//抓住我们之前错过主题订阅事件的可能性
//我们注册了事件侦听器
			if topicPartitionSubscriptionResumed != nil {
//停止侦听订阅消息
				saramaLogger.RemoveListener(subscription, topicPartitionSubscriptionResumed)
//禁用订阅事件chan
				topicPartitionSubscriptionResumed = nil
//停止超时计时器
				if !deliverSessionTimer.Stop() {
					<-deliverSessionTimer.C
				}
			}

			select {
case <-chain.errorChan: //如果这个频道被关闭…
chain.errorChan = make(chan struct{}) //…做一个新的。
				logger.Infof("[channel: %s] Marked consenter as available again", chain.ChainID())
			default:
			}
			if err := proto.Unmarshal(in.Value, msg); err != nil {
//这不应该发生，应该在入口过滤
				logger.Criticalf("[channel: %s] Unable to unmarshal consumed message = %s", chain.ChainID(), err)
				counts[indexUnmarshalError]++
				continue
			} else {
				logger.Debugf("[channel: %s] Successfully unmarshalled consumed message, offset is %d. Inspecting type...", chain.ChainID(), in.Offset)
				counts[indexRecvPass]++
			}
			switch msg.Type.(type) {
			case *ab.KafkaMessage_Connect:
				_ = chain.processConnect(chain.ChainID())
				counts[indexProcessConnectPass]++
			case *ab.KafkaMessage_TimeToCut:
				if err := chain.processTimeToCut(msg.GetTimeToCut(), in.Offset); err != nil {
					logger.Warningf("[channel: %s] %s", chain.ChainID(), err)
					logger.Criticalf("[channel: %s] Consenter for channel exiting", chain.ChainID())
					counts[indexProcessTimeToCutError]++
return counts, err //重温一下我们是否真的应该在此时停止处理链
				}
				counts[indexProcessTimeToCutPass]++
			case *ab.KafkaMessage_Regular:
				if err := chain.processRegular(msg.GetRegular(), in.Offset); err != nil {
					logger.Warningf("[channel: %s] Error when processing incoming message of type REGULAR = %s", chain.ChainID(), err)
					counts[indexProcessRegularError]++
				} else {
					counts[indexProcessRegularPass]++
				}
			}
		case <-chain.timer:
			if err := sendTimeToCut(chain.producer, chain.channel, chain.lastCutBlockNumber+1, &chain.timer); err != nil {
				logger.Errorf("[channel: %s] cannot post time-to-cut message = %s", chain.ChainID(), err)
//但是不要回来
				counts[indexSendTimeToCutError]++
			} else {
				counts[indexSendTimeToCutPass]++
			}
		}
	}
}

func (chain *chainImpl) closeKafkaObjects() []error {
	var errs []error

	err := chain.channelConsumer.Close()
	if err != nil {
		logger.Errorf("[channel: %s] could not close channelConsumer cleanly = %s", chain.ChainID(), err)
		errs = append(errs, err)
	} else {
		logger.Debugf("[channel: %s] Closed the channel consumer", chain.ChainID())
	}

	err = chain.parentConsumer.Close()
	if err != nil {
		logger.Errorf("[channel: %s] could not close parentConsumer cleanly = %s", chain.ChainID(), err)
		errs = append(errs, err)
	} else {
		logger.Debugf("[channel: %s] Closed the parent consumer", chain.ChainID())
	}

	err = chain.producer.Close()
	if err != nil {
		logger.Errorf("[channel: %s] could not close producer cleanly = %s", chain.ChainID(), err)
		errs = append(errs, err)
	} else {
		logger.Debugf("[channel: %s] Closed the producer", chain.ChainID())
	}

	return errs
}

//帮助程序函数

func getLastCutBlockNumber(blockchainHeight uint64) uint64 {
	return blockchainHeight - 1
}

func getOffsets(metadataValue []byte, chainID string) (persisted int64, processed int64, resubmitted int64) {
	if metadataValue != nil {
//首先从分类帐的尖端提取与医嘱者相关的元数据
		kafkaMetadata := &ab.KafkaMetadata{}
		if err := proto.Unmarshal(metadataValue, kafkaMetadata); err != nil {
			logger.Panicf("[channel: %s] Ledger may be corrupted:"+
				"cannot unmarshal orderer metadata in most recent block", chainID)
		}
		return kafkaMetadata.LastOffsetPersisted,
			kafkaMetadata.LastOriginalOffsetProcessed,
			kafkaMetadata.LastResubmittedConfigOffset
	}
return sarama.OffsetOldest - 1, int64(0), int64(0) //违约
}

func newConnectMessage() *ab.KafkaMessage {
	return &ab.KafkaMessage{
		Type: &ab.KafkaMessage_Connect{
			Connect: &ab.KafkaMessageConnect{
				Payload: nil,
			},
		},
	}
}

func newNormalMessage(payload []byte, configSeq uint64, originalOffset int64) *ab.KafkaMessage {
	return &ab.KafkaMessage{
		Type: &ab.KafkaMessage_Regular{
			Regular: &ab.KafkaMessageRegular{
				Payload:        payload,
				ConfigSeq:      configSeq,
				Class:          ab.KafkaMessageRegular_NORMAL,
				OriginalOffset: originalOffset,
			},
		},
	}
}

func newConfigMessage(config []byte, configSeq uint64, originalOffset int64) *ab.KafkaMessage {
	return &ab.KafkaMessage{
		Type: &ab.KafkaMessage_Regular{
			Regular: &ab.KafkaMessageRegular{
				Payload:        config,
				ConfigSeq:      configSeq,
				Class:          ab.KafkaMessageRegular_CONFIG,
				OriginalOffset: originalOffset,
			},
		},
	}
}

func newTimeToCutMessage(blockNumber uint64) *ab.KafkaMessage {
	return &ab.KafkaMessage{
		Type: &ab.KafkaMessage_TimeToCut{
			TimeToCut: &ab.KafkaMessageTimeToCut{
				BlockNumber: blockNumber,
			},
		},
	}
}

func newProducerMessage(channel channel, pld []byte) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{
		Topic: channel.topic(),
Key:   sarama.StringEncoder(strconv.Itoa(int(channel.partition()))), //要考虑写一个意图编码器吗？
		Value: sarama.ByteEncoder(pld),
	}
}

func (chain *chainImpl) processConnect(channelName string) error {
	logger.Debugf("[channel: %s] It's a connect message - ignoring", channelName)
	return nil
}

func (chain *chainImpl) processRegular(regularMessage *ab.KafkaMessageRegular, receivedOffset int64) error {
//在提交普通消息时，我们还使用“newoffset”更新“lastoriginaloffsetprocessed”。
//调用者有责任根据以下规则推断“newoffset”的正确值：
//-如果重新提交已关闭，则应始终为零。
//-如果消息在第一次传递时提交，意味着它没有重新验证和重新排序，则此值
//应与当前“lastoriginaloffsetprocessed”相同
//-如果消息被重新验证和排序，则此值应为该消息的“原始偏移量”
//Kafka消息，因此“lastoriginaloffsetprocessed”是高级的
	commitNormalMsg := func(message *cb.Envelope, newOffset int64) {
		batches, pending := chain.BlockCutter().Ordered(message)
		logger.Debugf("[channel: %s] Ordering results: items in batch = %d, pending = %v", chain.ChainID(), len(batches), pending)

		switch {
		case chain.timer != nil && !pending:
//计时器已在运行，但没有挂起的消息，请停止计时器
			chain.timer = nil
		case chain.timer == nil && pending:
//计时器尚未运行，并且有消息挂起，请启动它。
			chain.timer = time.After(chain.SharedConfig().BatchTimeout())
			logger.Debugf("[channel: %s] Just began %s batch timer", chain.ChainID(), chain.SharedConfig().BatchTimeout().String())
		default:
//在以下情况下不执行任何操作：
//1。计时器已在运行，有消息挂起
//2。没有设置计时器，也没有消息挂起。
		}

		if len(batches) == 0 {
//如果未切割任何块，则更新“lastoriginaloffsetprocessed”，必要时启动计时器并返回
			chain.lastOriginalOffsetProcessed = newOffset
			return
		}

		offset := receivedOffset
		if pending || len(batches) == 2 {
//如果最新的信封没有封装到第一批中，
//“lastoffsetpersisted”应为“receivedoffset”-1。
			offset--
		} else {
//我们只切割了一个块，因此可以安全地更新
//`lastOriginalOffsetProcessed`在此处使用'newOffset`处理，然后
//将其封装到此块中。否则，如果我们切两个
//块，第一个块应使用当前的“lastOriginalOffsetProcessed”
//第二个应该使用“newoffset”，它也用于
//更新“lastoriginaloffsetprocessed”
			chain.lastOriginalOffsetProcessed = newOffset
		}

//提交第一个块
		block := chain.CreateNextBlock(batches[0])
		metadata := utils.MarshalOrPanic(&ab.KafkaMetadata{
			LastOffsetPersisted:         offset,
			LastOriginalOffsetProcessed: chain.lastOriginalOffsetProcessed,
			LastResubmittedConfigOffset: chain.lastResubmittedConfigOffset,
		})
		chain.WriteBlock(block, metadata)
		chain.lastCutBlockNumber++
		logger.Debugf("[channel: %s] Batch filled, just cut block %d - last persisted offset is now %d", chain.ChainID(), chain.lastCutBlockNumber, offset)

//提交第二个块（如果存在）
		if len(batches) == 2 {
			chain.lastOriginalOffsetProcessed = newOffset
			offset++

			block := chain.CreateNextBlock(batches[1])
			metadata := utils.MarshalOrPanic(&ab.KafkaMetadata{
				LastOffsetPersisted:         offset,
				LastOriginalOffsetProcessed: newOffset,
				LastResubmittedConfigOffset: chain.lastResubmittedConfigOffset,
			})
			chain.WriteBlock(block, metadata)
			chain.lastCutBlockNumber++
			logger.Debugf("[channel: %s] Batch filled, just cut block %d - last persisted offset is now %d", chain.ChainID(), chain.lastCutBlockNumber, offset)
		}
	}

//提交配置消息时，我们还使用“newoffset”更新“lastoriginaloffsetprocessed”。
//调用者有责任根据以下规则推断“newoffset”的正确值：
//-如果重新提交已关闭，则应始终为零。
//-如果消息在第一次传递时提交，意味着它没有重新验证和重新排序，则此值
//应与当前“lastoriginaloffsetprocessed”相同
//-如果消息被重新验证和排序，则此值应为该消息的“原始偏移量”
//Kafka消息，因此“lastoriginaloffsetprocessed”是高级的
	commitConfigMsg := func(message *cb.Envelope, newOffset int64) {
		logger.Debugf("[channel: %s] Received config message", chain.ChainID())
		batch := chain.BlockCutter().Cut()

		if batch != nil {
			logger.Debugf("[channel: %s] Cut pending messages into block", chain.ChainID())
			block := chain.CreateNextBlock(batch)
			metadata := utils.MarshalOrPanic(&ab.KafkaMetadata{
				LastOffsetPersisted:         receivedOffset - 1,
				LastOriginalOffsetProcessed: chain.lastOriginalOffsetProcessed,
				LastResubmittedConfigOffset: chain.lastResubmittedConfigOffset,
			})
			chain.WriteBlock(block, metadata)
			chain.lastCutBlockNumber++
		}

		logger.Debugf("[channel: %s] Creating isolated block for config message", chain.ChainID())
		chain.lastOriginalOffsetProcessed = newOffset
		block := chain.CreateNextBlock([]*cb.Envelope{message})
		metadata := utils.MarshalOrPanic(&ab.KafkaMetadata{
			LastOffsetPersisted:         receivedOffset,
			LastOriginalOffsetProcessed: chain.lastOriginalOffsetProcessed,
			LastResubmittedConfigOffset: chain.lastResubmittedConfigOffset,
		})
		chain.WriteConfigBlock(block, metadata)
		chain.lastCutBlockNumber++
		chain.timer = nil
	}

	seq := chain.Sequence()

	env := &cb.Envelope{}
	if err := proto.Unmarshal(regularMessage.Payload, env); err != nil {
//这不应该发生，应该在入口过滤
		return fmt.Errorf("failed to unmarshal payload of regular message because = %s", err)
	}

	logger.Debugf("[channel: %s] Processing regular Kafka message of type %s", chain.ChainID(), regularMessage.Class.String())

//如果我们从1.1版之前的订购者那里收到消息，或者重新提交被显式禁用，那么每个订购者
//应该像1.1版之前的版本一样运行：再次验证，不要尝试重新排序。那是因为
//1.1版之前的订购者无法识别重新订购的消息，重新提交可能导致提交
//同样的信息两次。
//
//这里的隐式假设是，仅当不再存在时才设置重新提交功能标志。
//网络上的1.1版之前的订购者。否则它是未设置的，这就是我们所说的兼容模式。
	if regularMessage.Class == ab.KafkaMessageRegular_UNKNOWN || !chain.SharedConfig().Capabilities().Resubmission() {
//接收到类型未知的常规消息或关闭后重新提交，表示OSN网络带有v1.0.x订购程序
		logger.Warningf("[channel: %s] This orderer is running in compatibility mode", chain.ChainID())

		chdr, err := utils.ChannelHeader(env)
		if err != nil {
			return fmt.Errorf("discarding bad config message because of channel header unmarshalling error = %s", err)
		}

		class := chain.ClassifyMsg(chdr)
		switch class {
		case msgprocessor.ConfigMsg:
			if _, _, err := chain.ProcessConfigMsg(env); err != nil {
				return fmt.Errorf("discarding bad config message because = %s", err)
			}

			commitConfigMsg(env, chain.lastOriginalOffsetProcessed)

		case msgprocessor.NormalMsg:
			if _, err := chain.ProcessNormalMsg(env); err != nil {
				return fmt.Errorf("discarding bad normal message because = %s", err)
			}

			commitNormalMsg(env, chain.lastOriginalOffsetProcessed)

		case msgprocessor.ConfigUpdateMsg:
			return fmt.Errorf("not expecting message of type ConfigUpdate")

		default:
			logger.Panicf("[channel: %s] Unsupported message classification: %v", chain.ChainID(), class)
		}

		return nil
	}

	switch regularMessage.Class {
	case ab.KafkaMessageRegular_UNKNOWN:
		logger.Panicf("[channel: %s] Kafka message of type UNKNOWN should have been processed already", chain.ChainID())

	case ab.KafkaMessageRegular_NORMAL:
//这是一条重新验证和重新订购的消息
		if regularMessage.OriginalOffset != 0 {
			logger.Debugf("[channel: %s] Received re-submitted normal message with original offset %d", chain.ChainID(), regularMessage.OriginalOffset)

//但我们已经重新处理过了
			if regularMessage.OriginalOffset <= chain.lastOriginalOffsetProcessed {
				logger.Debugf(
					"[channel: %s] OriginalOffset(%d) <= LastOriginalOffsetProcessed(%d), message has been consumed already, discard",
					chain.ChainID(), regularMessage.OriginalOffset, chain.lastOriginalOffsetProcessed)
				return nil
			}

			logger.Debugf(
				"[channel: %s] OriginalOffset(%d) > LastOriginalOffsetProcessed(%d), "+
					"this is the first time we receive this re-submitted normal message",
				chain.ChainID(), regularMessage.OriginalOffset, chain.lastOriginalOffsetProcessed)

//如果我们没有重新处理消息，就不需要将其与那些消息区分开来。
//将首次处理的消息。
		}

//配置序列已高级
		if regularMessage.ConfigSeq < seq {
			logger.Debugf("[channel: %s] Config sequence has advanced since this normal message got validated, re-validating", chain.ChainID())
			configSeq, err := chain.ProcessNormalMsg(env)
			if err != nil {
				return fmt.Errorf("discarding bad normal message because = %s", err)
			}

			logger.Debugf("[channel: %s] Normal message is still valid, re-submit", chain.ChainID())

//对于第一次订购或重新订购的两条消息，我们设置原始偏移量
//到当前接收的偏移量并重新订购。
			if err := chain.order(env, configSeq, receivedOffset); err != nil {
				return fmt.Errorf("error re-submitting normal message because = %s", err)
			}

			return nil
		}

//任何进入此处的消息可能已重新验证，也可能未重新验证。
//重新订购，但在这里绝对有效

//高级原始偏移处理的iff消息重新验证和重新排序
		offset := regularMessage.OriginalOffset
		if offset == 0 {
			offset = chain.lastOriginalOffsetProcessed
		}

		commitNormalMsg(env, offset)

	case ab.KafkaMessageRegular_CONFIG:
//这是一条重新验证和重新订购的消息
		if regularMessage.OriginalOffset != 0 {
			logger.Debugf("[channel: %s] Received re-submitted config message with original offset %d", chain.ChainID(), regularMessage.OriginalOffset)

//但我们已经重新处理过了
			if regularMessage.OriginalOffset <= chain.lastOriginalOffsetProcessed {
				logger.Debugf(
					"[channel: %s] OriginalOffset(%d) <= LastOriginalOffsetProcessed(%d), message has been consumed already, discard",
					chain.ChainID(), regularMessage.OriginalOffset, chain.lastOriginalOffsetProcessed)
				return nil
			}

			logger.Debugf(
				"[channel: %s] OriginalOffset(%d) > LastOriginalOffsetProcessed(%d), "+
					"this is the first time we receive this re-submitted config message",
				chain.ChainID(), regularMessage.OriginalOffset, chain.lastOriginalOffsetProcessed)

if regularMessage.OriginalOffset == chain.lastResubmittedConfigOffset && //这是最后一次重新提交配置消息
regularMessage.ConfigSeq == seq { //我们不需要再重新提交
				logger.Debugf("[channel: %s] Config message with original offset %d is the last in-flight resubmitted message"+
					"and it does not require revalidation, unblock ingress messages now", chain.ChainID(), regularMessage.OriginalOffset)
chain.reprocessConfigComplete() //因此，我们最终可以解除广播的阻塞
			}

//有人在偏移量x处重新提交了消息，而我们没有。这是由于不确定性，其中
//在重新验证期间，我们认为该消息无效，但其他人认为它无效
//有效，然后重新提交。在这种情况下，我们需要提前最后重新提交configoffset
//在整个网络中强制实现一致性。
			if chain.lastResubmittedConfigOffset < regularMessage.OriginalOffset {
				chain.lastResubmittedConfigOffset = regularMessage.OriginalOffset
			}
		}

//配置序列已高级
		if regularMessage.ConfigSeq < seq {
			logger.Debugf("[channel: %s] Config sequence has advanced since this config message got validated, re-validating", chain.ChainID())
			configEnv, configSeq, err := chain.ProcessConfigMsg(env)
			if err != nil {
				return fmt.Errorf("rejecting config message because = %s", err)
			}

//对于第一次订购或重新订购的两条消息，我们设置原始偏移量
//到当前接收的偏移量并重新订购。
			if err := chain.configure(configEnv, configSeq, receivedOffset); err != nil {
				return fmt.Errorf("error re-submitting config message because = %s", err)
			}

			logger.Debugf("[channel: %s] Resubmitted config message with offset %d, block ingress messages", chain.ChainID(), receivedOffset)
chain.lastResubmittedConfigOffset = receivedOffset //跟踪上次重新提交的消息偏移量
chain.reprocessConfigPending()                     //开始阻止进入消息

			return nil
		}

//任何进入此处的消息可能已重新验证，也可能未重新验证。
//重新订购，但在这里绝对有效

//高级原始偏移处理的iff消息重新验证和重新排序
		offset := regularMessage.OriginalOffset
		if offset == 0 {
			offset = chain.lastOriginalOffsetProcessed
		}

		commitConfigMsg(env, offset)

	default:
		return fmt.Errorf("unsupported regular kafka message type: %v", regularMessage.Class.String())
	}

	return nil
}

func (chain *chainImpl) processTimeToCut(ttcMessage *ab.KafkaMessageTimeToCut, receivedOffset int64) error {
	ttcNumber := ttcMessage.GetBlockNumber()
	logger.Debugf("[channel: %s] It's a time-to-cut message for block %d", chain.ChainID(), ttcNumber)
	if ttcNumber == chain.lastCutBlockNumber+1 {
		chain.timer = nil
		logger.Debugf("[channel: %s] Nil'd the timer", chain.ChainID())
		batch := chain.BlockCutter().Cut()
		if len(batch) == 0 {
			return fmt.Errorf("got right time-to-cut message (for block %d),"+
				" no pending requests though; this might indicate a bug", chain.lastCutBlockNumber+1)
		}
		block := chain.CreateNextBlock(batch)
		metadata := utils.MarshalOrPanic(&ab.KafkaMetadata{
			LastOffsetPersisted:         receivedOffset,
			LastOriginalOffsetProcessed: chain.lastOriginalOffsetProcessed,
		})
		chain.WriteBlock(block, metadata)
		chain.lastCutBlockNumber++
		logger.Debugf("[channel: %s] Proper time-to-cut received, just cut block %d", chain.ChainID(), chain.lastCutBlockNumber)
		return nil
	} else if ttcNumber > chain.lastCutBlockNumber+1 {
		return fmt.Errorf("got larger time-to-cut message (%d) than allowed/expected (%d)"+
			" - this might indicate a bug", ttcNumber, chain.lastCutBlockNumber+1)
	}
	logger.Debugf("[channel: %s] Ignoring stale time-to-cut-message for block %d", chain.ChainID(), ttcNumber)
	return nil
}

//使用给定的重试选项将连接消息发布到通道。这个
//防止恐慌，如果我们要建立一个消费者和
//在尚未写入的分区上查找。
func sendConnectMessage(retryOptions localconfig.Retry, exitChan chan struct{}, producer sarama.SyncProducer, channel channel) error {
	logger.Infof("[channel: %s] About to post the CONNECT message...", channel.topic())

	payload := utils.MarshalOrPanic(newConnectMessage())
	message := newProducerMessage(channel, payload)

	retryMsg := "Attempting to post the CONNECT message..."
	postConnect := newRetryProcess(retryOptions, exitChan, channel, retryMsg, func() error {
		select {
		case <-exitChan:
			logger.Debugf("[channel: %s] Consenter for channel exiting, aborting retry", channel)
			return nil
		default:
			_, _, err := producer.SendMessage(message)
			return err
		}
	})

	return postConnect.retry()
}

func sendTimeToCut(producer sarama.SyncProducer, channel channel, timeToCutBlockNumber uint64, timer *<-chan time.Time) error {
	logger.Debugf("[channel: %s] Time-to-cut block %d timer expired", channel.topic(), timeToCutBlockNumber)
	*timer = nil
	payload := utils.MarshalOrPanic(newTimeToCutMessage(timeToCutBlockNumber))
	message := newProducerMessage(channel, payload)
	_, _, err := producer.SendMessage(message)
	return err
}

//使用给定的重试选项为通道设置分区使用者。
func setupChannelConsumerForChannel(retryOptions localconfig.Retry, haltChan chan struct{}, parentConsumer sarama.Consumer, channel channel, startFrom int64) (sarama.PartitionConsumer, error) {
	var err error
	var channelConsumer sarama.PartitionConsumer

	logger.Infof("[channel: %s] Setting up the channel consumer for this channel (start offset: %d)...", channel.topic(), startFrom)

	retryMsg := "Connecting to the Kafka cluster"
	setupChannelConsumer := newRetryProcess(retryOptions, haltChan, channel, retryMsg, func() error {
		channelConsumer, err = parentConsumer.ConsumePartition(channel.topic(), channel.partition(), startFrom)
		return err
	})

	return channelConsumer, setupChannelConsumer.retry()
}

//使用给定的重试选项为通道设置父使用者。
func setupParentConsumerForChannel(retryOptions localconfig.Retry, haltChan chan struct{}, brokers []string, brokerConfig *sarama.Config, channel channel) (sarama.Consumer, error) {
	var err error
	var parentConsumer sarama.Consumer

	logger.Infof("[channel: %s] Setting up the parent consumer for this channel...", channel.topic())

	retryMsg := "Connecting to the Kafka cluster"
	setupParentConsumer := newRetryProcess(retryOptions, haltChan, channel, retryMsg, func() error {
		parentConsumer, err = sarama.NewConsumer(brokers, brokerConfig)
		return err
	})

	return parentConsumer, setupParentConsumer.retry()
}

//使用给定的重试选项设置频道的编写器/制作人。
func setupProducerForChannel(retryOptions localconfig.Retry, haltChan chan struct{}, brokers []string, brokerConfig *sarama.Config, channel channel) (sarama.SyncProducer, error) {
	var err error
	var producer sarama.SyncProducer

	logger.Infof("[channel: %s] Setting up the producer for this channel...", channel.topic())

	retryMsg := "Connecting to the Kafka cluster"
	setupProducer := newRetryProcess(retryOptions, haltChan, channel, retryMsg, func() error {
		producer, err = sarama.NewSyncProducer(brokers, brokerConfig)
		return err
	})

	return producer, setupProducer.retry()
}

//为频道创建Kafka主题（如果尚未存在）
func setupTopicForChannel(retryOptions localconfig.Retry, haltChan chan struct{}, brokers []string, brokerConfig *sarama.Config, topicDetail *sarama.TopicDetail, channel channel) error {

//需要Kafka v0.10.1.0或更高版本
	if !brokerConfig.Version.IsAtLeast(sarama.V0_10_1_0) {
		return nil
	}

	logger.Infof("[channel: %s] Setting up the topic for this channel...",
		channel.topic())

	retryMsg := fmt.Sprintf("Creating Kafka topic [%s] for channel [%s]",
		channel.topic(), channel.String())

	setupTopic := newRetryProcess(
		retryOptions,
		haltChan,
		channel,
		retryMsg,
		func() error {

			var err error
			clusterMembers := map[int32]*sarama.Broker{}
			var controllerId int32

//通过代理循环访问元数据
			for _, address := range brokers {
				broker := sarama.NewBroker(address)
				err = broker.Open(brokerConfig)

				if err != nil {
					continue
				}

				var ok bool
				ok, err = broker.Connected()
				if !ok {
					continue
				}
				defer broker.Close()

//包含主题的元数据请求
				var apiVersion int16
				if brokerConfig.Version.IsAtLeast(sarama.V0_11_0_0) {
//使用API版本4禁用自动创建主题
//元数据请求
					apiVersion = 4
				} else {
					apiVersion = 1
				}
				metadata, err := broker.GetMetadata(&sarama.MetadataRequest{
					Version:                apiVersion,
					Topics:                 []string{channel.topic()},
					AllowAutoTopicCreation: false})

				if err != nil {
					continue
				}

				controllerId = metadata.ControllerID
				for _, broker := range metadata.Brokers {
					clusterMembers[broker.ID()] = broker
				}

				for _, topic := range metadata.Topics {
					if topic.Name == channel.topic() {
						if topic.Err != sarama.ErrUnknownTopicOrPartition {
//必须启用自动创建主题，因此返回
							return nil
						}
					}
				}
				break
			}

//检查我们是否从列表中的任何代理获取了任何元数据
			if len(clusterMembers) == 0 {
				return fmt.Errorf(
					"error creating topic [%s]; failed to retrieve metadata for the cluster",
					channel.topic())
			}

//找到控制器
			controller := clusterMembers[controllerId]
			err = controller.Open(brokerConfig)

			if err != nil {
				return err
			}

			var ok bool
			ok, err = controller.Connected()
			if !ok {
				return err
			}
			defer controller.Close()

//创建主题
			req := &sarama.CreateTopicsRequest{
				Version: 0,
				TopicDetails: map[string]*sarama.TopicDetail{
					channel.topic(): topicDetail},
				Timeout: 3 * time.Second}
			resp := &sarama.CreateTopicsResponse{}
			resp, err = controller.CreateTopics(req)
			if err != nil {
				return err
			}

//检查响应
			if topicErr, ok := resp.TopicErrors[channel.topic()]; ok {
//将无错误和主题存在错误视为成功
				if topicErr.Err == sarama.ErrNoError ||
					topicErr.Err == sarama.ErrTopicAlreadyExists {
					return nil
				}
				if topicErr.Err == sarama.ErrInvalidTopic {
//主题无效，因此中止
					logger.Warningf("[channel: %s] Failed to set up topic = %s",
						channel.topic(), topicErr.Err.Error())
					go func() {
						haltChan <- struct{}{}
					}()
				}
				return fmt.Errorf("error creating topic: [%s]",
					topicErr.Err.Error())
			}

			return nil
		})

	return setupTopic.retry()
}
