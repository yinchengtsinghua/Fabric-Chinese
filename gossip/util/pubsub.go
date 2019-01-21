
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


package util

import (
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	subscriptionBuffSize = 50
)

//pubsub定义了一个可以用来：
//-将项目发布到多个订阅服务器的主题
//-订阅主题中的项目
//订阅有一个TTL，当它通过时将被清除。
type PubSub struct {
	sync.RWMutex

//从主题到订阅集的映射
	subscriptions map[string]*Set
}

//订阅定义对主题的订阅
//可用于接收发布
type Subscription interface {
//在发布之前侦听块
//订阅，或者如果
//订阅的TTL已通过
	Listen() (interface{}, error)
}

type subscription struct {
	top string
	ttl time.Duration
	c   chan interface{}
}

//在发布之前侦听块
//订阅，或者如果
//订阅的TTL已通过
func (s *subscription) Listen() (interface{}, error) {
	select {
	case <-time.After(s.ttl):
		return nil, errors.New("timed out")
	case item := <-s.c:
		return item, nil
	}
}

//new pubsub创建一个新的pubsub，其中
//订阅集
func NewPubSub() *PubSub {
	return &PubSub{
		subscriptions: make(map[string]*Set),
	}
}

//发布将项目发布到主题上的所有订阅服务器
func (ps *PubSub) Publish(topic string, item interface{}) error {
	ps.RLock()
	defer ps.RUnlock()
	s, subscribed := ps.subscriptions[topic]
	if !subscribed {
		return errors.New("no subscribers")
	}
	for _, sub := range s.ToArray() {
		c := sub.(*subscription).c
//缓冲区中没有足够的空间，请继续以不阻止发布服务器
		if len(c) == subscriptionBuffSize {
			continue
		}
		c <- item
	}
	return nil
}

//订阅返回对给定TTL通过时过期的主题的订阅
func (ps *PubSub) Subscribe(topic string, ttl time.Duration) Subscription {
	sub := &subscription{
		top: topic,
		ttl: ttl,
		c:   make(chan interface{}, subscriptionBuffSize),
	}

	ps.Lock()
//向订阅映射添加订阅
	s, exists := ps.subscriptions[topic]
//如果不存在该主题的订阅集，请创建一个
	if !exists {
		s = NewSet()
		ps.subscriptions[topic] = s
	}
	ps.Unlock()

//添加订阅
	s.Add(sub)

//当超时到期时，删除订阅
	time.AfterFunc(ttl, func() {
		ps.unSubscribe(sub)
	})
	return sub
}

func (ps *PubSub) unSubscribe(sub *subscription) {
	ps.Lock()
	defer ps.Unlock()
	ps.subscriptions[sub.top].Remove(sub)
	if ps.subscriptions[sub.top].Size() != 0 {
		return
	}
//否则，这是该主题的最后一次订阅。
//从订阅映射中删除集合
	delete(ps.subscriptions, sub.top)

}
