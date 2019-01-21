
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


package msgstore

import (
	"sync"
	"time"

	"github.com/hyperledger/fabric/gossip/common"
)

var noopLock = func() {}

//noop是一个不做任何事情的函数
func Noop(_ interface{}) {

}

//对因添加消息而无效的每条消息调用InvalidationTrigger
//即：如果添加（0），则一个接一个调用添加（1），并且存储在调用序列之后只有1
//然后在添加1时调用0上的无效触发器。
type invalidationTrigger func(message interface{})

//new messagestore返回一个新的messagestore，并替换消息
//策略和无效触发已通过。
func NewMessageStore(pol common.MessageReplacingPolicy, trigger invalidationTrigger) MessageStore {
	return newMsgStore(pol, trigger)
}

//newmessagestoreexpireable返回一个新的messagestore，并替换消息
//策略和无效触发已通过。它支持在MSGTTL之后、在第一个外部过期期间过期的旧邮件
//锁定，调用过期回调，释放外部锁。回调和外部锁可以为零。
func NewMessageStoreExpirable(pol common.MessageReplacingPolicy, trigger invalidationTrigger, msgTTL time.Duration, externalLock func(), externalUnlock func(), externalExpire func(interface{})) MessageStore {
	store := newMsgStore(pol, trigger)
	store.msgTTL = msgTTL

	if externalLock != nil {
		store.externalLock = externalLock
	}

	if externalUnlock != nil {
		store.externalUnlock = externalUnlock
	}

	if externalExpire != nil {
		store.expireMsgCallback = externalExpire
	}

	go store.expirationRoutine()
	return store
}

func newMsgStore(pol common.MessageReplacingPolicy, trigger invalidationTrigger) *messageStoreImpl {
	return &messageStoreImpl{
		pol:        pol,
		messages:   make([]*msg, 0),
		invTrigger: trigger,

		externalLock:      noopLock,
		externalUnlock:    noopLock,
		expireMsgCallback: func(m interface{}) {},
		expiredCount:      0,

		doneCh: make(chan struct{}),
	}
}

//messagestore将消息添加到内部缓冲区。
//当收到消息时，它可能：
//-添加到缓冲区
//-由于缓冲区中已存在某些消息而被丢弃（已失效）
//-使缓冲区中已存在要丢弃的消息（无效）
//当消息无效时，将对该消息调用InvalidationTrigger。
type MessageStore interface {
//添加将消息添加到存储
//无论消息是否添加到存储区，都返回true或false
	Add(msg interface{}) bool

//检查消息是否可插入存储
//返回“真”或“假”消息是否可以添加到存储区
	CheckValid(msg interface{}) bool

//SIZE返回存储区中的邮件数量
	Size() int

//GET返回存储区中的所有消息
	Get() []interface{}

//停止所有相关的go例程
	Stop()

//清除清除接受的所有消息
//给定谓词
	Purge(func(interface{}) bool)
}

type messageStoreImpl struct {
	pol               common.MessageReplacingPolicy
	lock              sync.RWMutex
	messages          []*msg
	invTrigger        invalidationTrigger
	msgTTL            time.Duration
	expiredCount      int
	externalLock      func()
	externalUnlock    func()
	expireMsgCallback func(msg interface{})
	doneCh            chan struct{}
	stopOnce          sync.Once
}

type msg struct {
	data    interface{}
	created time.Time
	expired bool
}

//添加将消息添加到存储
func (s *messageStoreImpl) Add(message interface{}) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	n := len(s.messages)
	for i := 0; i < n; i++ {
		m := s.messages[i]
		switch s.pol(message, m.data) {
		case common.MessageInvalidated:
			return false
		case common.MessageInvalidates:
			s.invTrigger(m.data)
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			n--
			i--
		}
	}

	s.messages = append(s.messages, &msg{data: message, created: time.Now()})
	return true
}

func (s *messageStoreImpl) Purge(shouldBePurged func(interface{}) bool) {
	shouldMsgBePurged := func(m *msg) bool {
		return shouldBePurged(m.data)
	}
	if !s.isPurgeNeeded(shouldMsgBePurged) {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	n := len(s.messages)
	for i := 0; i < n; i++ {
		if !shouldMsgBePurged(s.messages[i]) {
			continue
		}
		s.invTrigger(s.messages[i].data)
		s.messages = append(s.messages[:i], s.messages[i+1:]...)
		n--
		i--
	}
}

//检查消息是否可插入存储
func (s *messageStoreImpl) CheckValid(message interface{}) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, m := range s.messages {
		if s.pol(message, m.data) == common.MessageInvalidated {
			return false
		}
	}
	return true
}

//SIZE返回存储区中的邮件数量
func (s *messageStoreImpl) Size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.messages) - s.expiredCount
}

//GET返回存储区中的所有消息
func (s *messageStoreImpl) Get() []interface{} {
	res := make([]interface{}, 0)

	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, msg := range s.messages {
		if !msg.expired {
			res = append(res, msg.data)
		}
	}
	return res
}

func (s *messageStoreImpl) expireMessages() {
	s.externalLock()
	s.lock.Lock()
	defer s.lock.Unlock()
	defer s.externalUnlock()

	n := len(s.messages)
	for i := 0; i < n; i++ {
		m := s.messages[i]
		if !m.expired {
			if time.Since(m.created) > s.msgTTL {
				m.expired = true
				s.expireMsgCallback(m.data)
				s.expiredCount++
			}
		} else {
			if time.Since(m.created) > (s.msgTTL * 2) {
				s.messages = append(s.messages[:i], s.messages[i+1:]...)
				n--
				i--
				s.expiredCount--
			}

		}
	}
}

func (s *messageStoreImpl) isPurgeNeeded(shouldBePurged func(*msg) bool) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, m := range s.messages {
		if shouldBePurged(m) {
			return true
		}
	}
	return false
}

func (s *messageStoreImpl) expirationRoutine() {
	for {
		select {
		case <-s.doneCh:
			return
		case <-time.After(s.expirationCheckInterval()):
			hasMessageExpired := func(m *msg) bool {
				if !m.expired && time.Since(m.created) > s.msgTTL {
					return true
				} else if time.Since(m.created) > (s.msgTTL * 2) {
					return true
				}
				return false
			}
			if s.isPurgeNeeded(hasMessageExpired) {
				s.expireMessages()
			}
		}
	}
}

func (s *messageStoreImpl) Stop() {
	stopFunc := func() {
		close(s.doneCh)
	}
	s.stopOnce.Do(stopFunc)
}

func (s *messageStoreImpl) expirationCheckInterval() time.Duration {
	return s.msgTTL / 100
}
