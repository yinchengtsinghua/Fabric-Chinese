
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
	"time"

	"github.com/pkg/errors"
)

type emitBatchCallback func([]interface{})

//batchingemitter用于八卦推送/转发阶段。
//消息被添加到batchingemitter中，它们被周期性地分批转发T次，然后丢弃。
//如果batchingemitter的存储消息计数达到某个容量，也会触发消息调度
type batchingEmitter interface {
//添加添加要批处理的消息
	Add(interface{})

//Stop stops the component
	Stop()

//大小返回要发出的挂起消息的数量
	Size() int
}

//NewPatchingEmitter接受以下参数：
//迭代：每个消息转发的次数
//BurstSize：由于消息计数而触发转发的阈值。
//延迟：每个消息在不转发的情况下可以存储的最大延迟
//cb：为进行转发而调用的回调。
func newBatchingEmitter(iterations, burstSize int, latency time.Duration, cb emitBatchCallback) batchingEmitter {
	if iterations < 0 {
		panic(errors.Errorf("Got a negative iterations number"))
	}

	p := &batchingEmitterImpl{
		cb:         cb,
		delay:      latency,
		iterations: iterations,
		burstSize:  burstSize,
		lock:       &sync.Mutex{},
		buff:       make([]*batchedMessage, 0),
		stopFlag:   int32(0),
	}

	if iterations != 0 {
		go p.periodicEmit()
	}

	return p
}

func (p *batchingEmitterImpl) periodicEmit() {
	for !p.toDie() {
		time.Sleep(p.delay)
		p.lock.Lock()
		p.emit()
		p.lock.Unlock()
	}
}

func (p *batchingEmitterImpl) emit() {
	if p.toDie() {
		return
	}
	if len(p.buff) == 0 {
		return
	}
	msgs2beEmitted := make([]interface{}, len(p.buff))
	for i, v := range p.buff {
		msgs2beEmitted[i] = v.data
	}

	p.cb(msgs2beEmitted)
	p.decrementCounters()
}

func (p *batchingEmitterImpl) decrementCounters() {
	n := len(p.buff)
	for i := 0; i < n; i++ {
		msg := p.buff[i]
		msg.iterationsLeft--
		if msg.iterationsLeft == 0 {
			p.buff = append(p.buff[:i], p.buff[i+1:]...)
			n--
			i--
		}
	}
}

func (p *batchingEmitterImpl) toDie() bool {
	return atomic.LoadInt32(&(p.stopFlag)) == int32(1)
}

type batchingEmitterImpl struct {
	iterations int
	burstSize  int
	delay      time.Duration
	cb         emitBatchCallback
	lock       *sync.Mutex
	buff       []*batchedMessage
	stopFlag   int32
}

type batchedMessage struct {
	data           interface{}
	iterationsLeft int
}

func (p *batchingEmitterImpl) Stop() {
	atomic.StoreInt32(&(p.stopFlag), int32(1))
}

func (p *batchingEmitterImpl) Size() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	return len(p.buff)
}

func (p *batchingEmitterImpl) Add(message interface{}) {
	if p.iterations == 0 {
		return
	}
	p.lock.Lock()
	defer p.lock.Unlock()

	p.buff = append(p.buff, &batchedMessage{data: message, iterationsLeft: p.iterations})

	if len(p.buff) >= p.burstSize {
		p.emit()
	}
}
