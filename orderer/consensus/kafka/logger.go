
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
	"sync"

	"github.com/Shopify/sarama"
	"github.com/hyperledger/fabric/common/flogging"
	"go.uber.org/zap"
)

var logger = flogging.MustGetLogger("orderer.consensus.kafka")
var saramaLogger eventLogger

//init初始化Samara记录器
func init() {
	loggingProvider := flogging.MustGetLogger("orderer.consensus.kafka.sarama")
	saramaEventLogger := &saramaLoggerImpl{
		logger: loggingProvider.WithOptions(zap.AddCallerSkip(3)),
		eventListenerSupport: &eventListenerSupport{
			listeners: make(map[string][]chan string),
		},
	}
	sarama.Logger = saramaEventLogger
	saramaLogger = saramaEventLogger
}

//init启动一个go例程，检测可能的配置问题
func init() {
	listener := saramaLogger.NewListener("insufficient data to decode packet")
	go func() {
		for {
			select {
			case <-listener:
				logger.Critical("Unable to decode a Kafka packet. Usually, this " +
					"indicates that the Kafka.Version specified in the orderer " +
					"configuration is incorrectly set to a version which is newer than " +
					"the actual Kafka broker version.")
			}
		}
	}()
}

//事件记录器将记录器适应sarama.logger接口。
//此外，可以注册侦听器，以便在子字符串
//已记录。
type eventLogger interface {
	sarama.StdLogger
	NewListener(substr string) <-chan string
	RemoveListener(substr string, listener <-chan string)
}

type debugger interface {
	Debug(...interface{})
}

type saramaLoggerImpl struct {
	logger               debugger
	eventListenerSupport *eventListenerSupport
}

func (l saramaLoggerImpl) Print(args ...interface{}) {
	l.print(fmt.Sprint(args...))
}

func (l saramaLoggerImpl) Printf(format string, args ...interface{}) {
	l.print(fmt.Sprintf(format, args...))
}

func (l saramaLoggerImpl) Println(args ...interface{}) {
	l.print(fmt.Sprintln(args...))
}

func (l saramaLoggerImpl) print(message string) {
	l.eventListenerSupport.fire(message)
	l.logger.Debug(message)
}

//对于一个行为良好的听众来说，这应该足够了。
const listenerChanSize = 100

func (l saramaLoggerImpl) NewListener(substr string) <-chan string {
	listener := make(chan string, listenerChanSize)
	l.eventListenerSupport.addListener(substr, listener)
	return listener
}

func (l saramaLoggerImpl) RemoveListener(substr string, listener <-chan string) {
	l.eventListenerSupport.removeListener(substr, listener)
}

//EventListenerSupport维护到侦听器列表的子字符串映射
//有兴趣在记录子字符串时接收通知。
type eventListenerSupport struct {
	sync.Mutex
	listeners map[string][]chan string
}

//addListener将侦听器添加到指定子字符串的侦听器列表中
func (b *eventListenerSupport) addListener(substr string, listener chan string) {
	b.Lock()
	defer b.Unlock()
	if listeners, ok := b.listeners[substr]; ok {
		b.listeners[substr] = append(listeners, listener)
	} else {
		b.listeners[substr] = []chan string{listener}
	}
}

//fire将指定的消息发送到注册到的每个侦听器
//包含在消息中的子字符串
func (b *eventListenerSupport) fire(message string) {
	b.Lock()
	defer b.Unlock()
	for substr, listeners := range b.listeners {
		if strings.Contains(message, substr) {
			for _, listener := range listeners {
				listener <- message
			}
		}
	}
}

//addListener从指定子字符串的侦听器列表中删除侦听器
func (b *eventListenerSupport) removeListener(substr string, listener <-chan string) {
	b.Lock()
	defer b.Unlock()
	if listeners, ok := b.listeners[substr]; ok {
		for i, l := range listeners {
			if l == listener {
				copy(listeners[i:], listeners[i+1:])
				listeners[len(listeners)-1] = nil
				b.listeners[substr] = listeners[:len(listeners)-1]
			}
		}
	}
}
