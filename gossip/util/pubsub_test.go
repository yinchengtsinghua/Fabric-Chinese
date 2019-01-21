
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPubsub(t *testing.T) {
	ps := NewPubSub()
//检查订阅成功的主题的发布
	sub1 := ps.Subscribe("test", time.Second)
	sub2 := ps.Subscribe("test2", time.Second)
	assert.NotNil(t, sub1)
	go func() {
		err := ps.Publish("test", 5)
		assert.NoError(t, err)
	}()
	item, err := sub1.Listen()
	assert.NoError(t, err)
	assert.Equal(t, 5, item)
//检查没有订阅服务器的主题的发布是否失败
	err = ps.Publish("test3", 5)
	assert.Error(t, err)
	assert.Contains(t, "no subscribers", err.Error())
//检查是否对其发布太迟、超时的主题进行侦听
//并返回一个错误
	go func() {
		time.Sleep(time.Second * 2)
		ps.Publish("test2", 10)
	}()
	item, err = sub2.Listen()
	assert.Error(t, err)
	assert.Contains(t, "timed out", err.Error())
	assert.Nil(t, item)
//让多个订户订阅同一主题
	subscriptions := []Subscription{}
	n := 100
	for i := 0; i < n; i++ {
		subscriptions = append(subscriptions, ps.Subscribe("test4", time.Second))
	}
	go func() {
//发送项目并填充缓冲区和溢出
//1项
		for i := 0; i <= subscriptionBuffSize; i++ {
			err := ps.Publish("test4", 100+i)
			assert.NoError(t, err)
		}
	}()
	wg := sync.WaitGroup{}
	wg.Add(n)
	for _, s := range subscriptions {
		go func(s Subscription) {
			time.Sleep(time.Second)
			defer wg.Done()
			for i := 0; i < subscriptionBuffSize; i++ {
				item, err := s.Listen()
				assert.NoError(t, err)
				assert.Equal(t, 100+i, item)
			}
//我们发布的最后一个项目被删除
//因为缓冲区已满
			item, err := s.Listen()
			assert.Nil(t, item)
			assert.Error(t, err)
		}(s)
	}
	wg.Wait()

//确保订阅在使用后被清除
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		ps.Lock()
		empty := len(ps.subscriptions) == 0
		ps.Unlock()
		if empty {
			break
		}
	}
	ps.Lock()
	defer ps.Unlock()
	assert.Empty(t, ps.subscriptions)
}
