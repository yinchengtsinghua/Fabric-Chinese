
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


package gossip

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/gossip/util"
	"github.com/stretchr/testify/assert"
)

func init() {
	util.SetupTestLogging()
}

func TestBatchingEmitterAddAndSize(t *testing.T) {
	emitter := newBatchingEmitter(1, 10, time.Second, func(a []interface{}) {})
	defer emitter.Stop()
	emitter.Add(1)
	emitter.Add(2)
	emitter.Add(3)
	assert.Equal(t, 3, emitter.Size())
}

func TestBatchingEmitterStop(t *testing.T) {
//In this test we make sure the emitter doesn't do anything after it's stopped
	disseminationAttempts := int32(0)
	cb := func(a []interface{}) {
		atomic.AddInt32(&disseminationAttempts, int32(1))
	}

	emitter := newBatchingEmitter(10, 1, time.Duration(100)*time.Millisecond, cb)
	emitter.Add(1)
	time.Sleep(time.Duration(100) * time.Millisecond)
	emitter.Stop()
	time.Sleep(time.Duration(1000) * time.Millisecond)
	assert.True(t, atomic.LoadInt32(&disseminationAttempts) < int32(5))
}

func TestBatchingEmitterExpiration(t *testing.T) {
//在此测试中，我们确保消息已过期，并在足够长的时间后丢弃。
//它被转发了足够的次数
	disseminationAttempts := int32(0)
	cb := func(a []interface{}) {
		atomic.AddInt32(&disseminationAttempts, int32(1))
	}

	emitter := newBatchingEmitter(10, 1, time.Duration(10)*time.Millisecond, cb)
	defer emitter.Stop()

	emitter.Add(1)
	time.Sleep(time.Duration(500) * time.Millisecond)
	assert.Equal(t, int32(10), atomic.LoadInt32(&disseminationAttempts), "Inadequate amount of dissemination attempts detected")
	assert.Equal(t, 0, emitter.Size())
}

func TestBatchingEmitterCounter(t *testing.T) {
//在这个测试中，我们计算每条消息转发的次数，与传递的时间有关。
	counters := make(map[int]int)
	lock := &sync.Mutex{}
	cb := func(a []interface{}) {
		lock.Lock()
		defer lock.Unlock()
		for _, e := range a {
			n := e.(int)
			if _, exists := counters[n]; !exists {
				counters[n] = 0
			} else {
				counters[n]++
			}
		}
	}

	emitter := newBatchingEmitter(5, 100, time.Duration(500)*time.Millisecond, cb)
	defer emitter.Stop()

	for i := 1; i <= 5; i++ {
		emitter.Add(i)
		if i == 5 {
			break
		}
		time.Sleep(time.Duration(600) * time.Millisecond)
	}
	emitter.Stop()

	lock.Lock()
	assert.Equal(t, 0, counters[4])
	assert.Equal(t, 1, counters[3])
	assert.Equal(t, 2, counters[2])
	assert.Equal(t, 3, counters[1])
	lock.Unlock()
}

//testbatchingemitterburstsizecap测试发射器
func TestBatchingEmitterBurstSizeCap(t *testing.T) {
	disseminationAttempts := int32(0)
	cb := func(a []interface{}) {
		atomic.AddInt32(&disseminationAttempts, int32(1))
	}
	emitter := newBatchingEmitter(1, 10, time.Duration(800)*time.Millisecond, cb)
	defer emitter.Stop()

	for i := 0; i < 50; i++ {
		emitter.Add(i)
	}
	assert.Equal(t, int32(5), atomic.LoadInt32(&disseminationAttempts))
}
