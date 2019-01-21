
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


package state

import (
	"crypto/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
)

func init() {
	util.SetupTestLogging()
}

func randomPayloadWithSeqNum(seqNum uint64) (*proto.Payload, error) {
	data := make([]byte, 64)
	_, err := rand.Read(data)
	if err != nil {
		return nil, err
	}
	return &proto.Payload{
		SeqNum: seqNum,
		Data:   data,
	}, nil
}

func TestNewPayloadsBuffer(t *testing.T) {
	payloadsBuffer := NewPayloadsBuffer(10)
	assert.Equal(t, payloadsBuffer.Next(), uint64(10))
}

func TestPayloadsBufferImpl_Push(t *testing.T) {
	buffer := NewPayloadsBuffer(5)

	payload, err := randomPayloadWithSeqNum(4)

	if err != nil {
		t.Fatal("Wasn't able to generate random payload for test")
	}

	t.Log("Pushing new payload into buffer")
	buffer.Push(payload)

//序列号小于缓冲区顶部的有效载荷
//不应接受索引
	t.Log("Getting next block sequence number")
	assert.Equal(t, buffer.Next(), uint64(5))
	t.Log("Check block buffer size")
	assert.Equal(t, buffer.Size(), 0)

//使用seq添加新的有效负载。数字等于顶部
//不应添加有效负载
	payload, err = randomPayloadWithSeqNum(5)
	if err != nil {
		t.Fatal("Wasn't able to generate random payload for test")
	}

	t.Log("Pushing new payload into buffer")
	buffer.Push(payload)
	t.Log("Getting next block sequence number")
	assert.Equal(t, buffer.Next(), uint64(5))
	t.Log("Check block buffer size")
	assert.Equal(t, buffer.Size(), 1)
}

func TestPayloadsBufferImpl_Ready(t *testing.T) {
	fin := make(chan struct{})
	buffer := NewPayloadsBuffer(1)
	assert.Equal(t, buffer.Next(), uint64(1))

	go func() {
		<-buffer.Ready()
		fin <- struct{}{}
	}()

	time.AfterFunc(100*time.Millisecond, func() {
		payload, err := randomPayloadWithSeqNum(1)

		if err != nil {
			t.Fatal("Wasn't able to generate random payload for test")
		}
		buffer.Push(payload)
	})

	select {
	case <-fin:
		payload := buffer.Pop()
		assert.Equal(t, payload.SeqNum, uint64(1))
	case <-time.After(500 * time.Millisecond):
		t.Fail()
	}
}

//将几个并发块推入缓冲区的测试
//如果序列号相同，则只有一个期望成功
func TestPayloadsBufferImpl_ConcurrentPush(t *testing.T) {

//测试设置，预期的下一个块编号和
//要模拟多少并发推
	nextSeqNum := uint64(7)
	concurrency := 10

	buffer := NewPayloadsBuffer(nextSeqNum)
	assert.Equal(t, buffer.Next(), uint64(nextSeqNum))

	startWG := sync.WaitGroup{}
	startWG.Add(1)

	finishWG := sync.WaitGroup{}
	finishWG.Add(concurrency)

	payload, err := randomPayloadWithSeqNum(nextSeqNum)
	assert.NoError(t, err)

	ready := int32(0)
	readyWG := sync.WaitGroup{}
	readyWG.Add(1)
	go func() {
//等待下一个预期块到达
		<-buffer.Ready()
		atomic.AddInt32(&ready, 1)
		readyWG.Done()
	}()

	for i := 0; i < concurrency; i++ {
		go func() {
			buffer.Push(payload)
			startWG.Wait()
			finishWG.Done()
		}()
	}
	startWG.Done()
	finishWG.Wait()

	readyWG.Wait()
	assert.Equal(t, int32(1), atomic.LoadInt32(&ready))
//缓冲区大小必须只有一个
	assert.Equal(t, 1, buffer.Size())
}

//测试负载推送和POP在ready（）信号后交错的场景。
func TestPayloadsBufferImpl_Interleave(t *testing.T) {
	buffer := NewPayloadsBuffer(1)
	assert.Equal(t, buffer.Next(), uint64(1))

//
//前两个序列到达，缓冲区被清空，没有交错。
//
//这也是织物生产/消费模式的一个例子。
//生产者：
//
//有效载荷被推入缓冲器。这些有效载荷可能不正常。
//当缓冲区有一系列有效负载就绪（按顺序）时，它会触发一个信号
//在它的ready（）通道上。
//
//使用者等待信号，然后耗尽所有准备好的有效负载。

	payload, err := randomPayloadWithSeqNum(1)
	assert.NoError(t, err, "generating random payload failed")
	buffer.Push(payload)

	payload, err = randomPayloadWithSeqNum(2)
	assert.NoError(t, err, "generating random payload failed")
	buffer.Push(payload)

	select {
	case <-buffer.Ready():
	case <-time.After(500 * time.Millisecond):
		t.Error("buffer wasn't ready after 500 ms for first sequence")
	}

//消费者清空缓冲区。
	for payload := buffer.Pop(); payload != nil; payload = buffer.Pop() {
	}

//缓冲区没有准备好，因为清空缓冲区后没有新的序列。
	select {
	case <-buffer.Ready():
		t.Error("buffer should not be ready as no new sequences have come")
	case <-time.After(500 * time.Millisecond):
	}

//
//下一个序列在消费者清空缓冲区的同时进入。
//
	payload, err = randomPayloadWithSeqNum(3)
	assert.NoError(t, err, "generating random payload failed")
	buffer.Push(payload)

	select {
	case <-buffer.Ready():
	case <-time.After(500 * time.Millisecond):
		t.Error("buffer wasn't ready after 500 ms for second sequence")
	}
	payload = buffer.Pop()
	assert.NotNil(t, payload, "payload should not be nil")

//…块处理现在在序列3上进行。

//同时，序列4被推入队列。
	payload, err = randomPayloadWithSeqNum(4)
	assert.NoError(t, err, "generating random payload failed")
	buffer.Push(payload)

//…块处理在序列3上完成，消费循环抓取下一个（4）。
	payload = buffer.Pop()
	assert.NotNil(t, payload, "payload should not be nil")

//同时，序列5被推入队列。
	payload, err = randomPayloadWithSeqNum(5)
	assert.NoError(t, err, "generating random payload failed")
	buffer.Push(payload)

//…块处理在序列4上完成，消费循环抓取下一个（5）。
	payload = buffer.Pop()
	assert.NotNil(t, payload, "payload should not be nil")

//
//现在我们看到Goroutines由于上面的交错推送和弹出而正在建立。
//
	select {
	case <-buffer.Ready():
//
//应该是错误的-没有有效载荷准备就绪
//
		t.Log("buffer ready (1) -- should be error")
		t.Fail()
	case <-time.After(500 * time.Millisecond):
		t.Log("buffer not ready (1)")
	}
	payload = buffer.Pop()
	t.Logf("payload: %v", payload)
	assert.Nil(t, payload, "payload should be nil")

	select {
	case <-buffer.Ready():
//
//应该是错误的-没有有效载荷准备就绪
//
		t.Log("buffer ready (2) -- should be error")
		t.Fail()
	case <-time.After(500 * time.Millisecond):
		t.Log("buffer not ready (2)")
	}
	payload = buffer.Pop()
	assert.Nil(t, payload, "payload should be nil")
	t.Logf("payload: %v", payload)

	select {
	case <-buffer.Ready():
		t.Error("buffer ready (3)")
	case <-time.After(500 * time.Millisecond):
		t.Log("buffer not ready (3) -- good")
	}
}
