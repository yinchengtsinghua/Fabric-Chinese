
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
	"sync"
	"sync/atomic"

	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

//payloadsbuffer用于存储用于
//根据
//序列号。它还将提供
//当预期的数据块到达时发出信号。
type PayloadsBuffer interface {
//将新块添加到缓冲区中
	Push(payload *proto.Payload)

//返回下一个预期序列号
	Next() uint64

//移除并返回具有给定序列号的有效载荷
	Pop() *proto.Payload

//获取当前缓冲区大小
	Size() int

//按顺序推送新有效负载时指示事件的通道
//数字等于下一个期望值。
	Ready() chan struct{}

	Close()
}

//用于实现payloadsbuffer接口的payloadsbufferimpl结构
//存储可用有效载荷和序列号的内部状态
type PayloadsBufferImpl struct {
	next uint64

	buf map[uint64]*proto.Payload

	readyChan chan struct{}

	mutex sync.RWMutex

	logger util.Logger
}

//new payloads buffer是工厂函数，用于创建新的payloads缓冲区
func NewPayloadsBuffer(next uint64) PayloadsBuffer {
	return &PayloadsBufferImpl{
		buf:       make(map[uint64]*proto.Payload),
		readyChan: make(chan struct{}, 1),
		next:      next,
		logger:    util.GetLogger(util.StateLogger, ""),
	}
}

//就绪函数返回指示预期时间的通道
//下一个街区已经到了，可以安全地跳出来
//下一个块序列
func (b *PayloadsBufferImpl) Ready() chan struct{} {
	return b.readyChan
}

//将新的有效载荷推入缓冲结构，以防新的到达有效载荷
//序列号低于预期的下一个块号负载将
//扔掉。
//TODO返回bool以指示是否添加了有效负载，以便调用方可以记录结果。
func (b *PayloadsBufferImpl) Push(payload *proto.Payload) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	seqNum := payload.SeqNum

	if seqNum < b.next || b.buf[seqNum] != nil {
		logger.Debugf("Payload with sequence number = %d has been already processed", payload.SeqNum)
		return
	}

	b.buf[seqNum] = payload

//发送下一个序列已到达的通知
	if seqNum == b.next && len(b.readyChan) == 0 {
		b.readyChan <- struct{}{}
	}
}

//next函数提供下一个预期块的编号
func (b *PayloadsBufferImpl) Next() uint64 {
//自动读取顶部序列号的值
	return atomic.LoadUint64(&b.next)
}

//pop函数根据下一个预期块提取有效负载
//如果还没有下一个块到达，则函数返回nil。
func (b *PayloadsBufferImpl) Pop() *proto.Payload {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	result := b.buf[b.Next()]

	if result != nil {
//如果缓冲区中有这样的序列，需要删除它
		delete(b.buf, b.Next())
//增加下一个预期块索引
		atomic.AddUint64(&b.next, 1)

		b.drainReadChannel()

	}

	return result
}

//DRAINREADCHANNEL清空准备就绪的通道，以防最后
//有效载荷已弹出，仍在等待
//频道中的通知
func (b *PayloadsBufferImpl) drainReadChannel() {
	if len(b.buf) == 0 {
		for {
			if len(b.readyChan) > 0 {
				<-b.readyChan
			} else {
				break
			}
		}
	}
}

//SIZE返回当前存储在缓冲区中的有效负载数
func (b *PayloadsBufferImpl) Size() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return len(b.buf)
}

//在维护中关闭清理资源和渠道
func (b *PayloadsBufferImpl) Close() {
	close(b.readyChan)
}
