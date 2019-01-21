
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


package comm

import (
	"errors"
	"testing"
	"time"

	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
)

func TestInterceptAcks(t *testing.T) {
	pubsub := util.NewPubSub()
	pkiID := common.PKIidType("pkiID")
	msgs := make(chan *proto.SignedGossipMessage, 1)
	handlerFunc := func(message *proto.SignedGossipMessage) {
		msgs <- message
	}
	wrappedHandler := interceptAcks(handlerFunc, pkiID, pubsub)
	ack := &proto.SignedGossipMessage{
		GossipMessage: &proto.GossipMessage{
			Nonce: 1,
			Content: &proto.GossipMessage_Ack{
				Ack: &proto.Acknowledgement{},
			},
		},
	}
	sub := pubsub.Subscribe(topicForAck(1, pkiID), time.Second)
	wrappedHandler(ack)
//确保ACK已被消耗，且不会传递给包裹的处理者。
	assert.Len(t, msgs, 0)
	_, err := sub.Listen()
//确保ACK已发布
	assert.NoError(t, err)

//测试未转发ACK
	notAck := &proto.SignedGossipMessage{
		GossipMessage: &proto.GossipMessage{
			Nonce: 2,
			Content: &proto.GossipMessage_DataMsg{
				DataMsg: &proto.DataMessage{},
			},
		},
	}
	sub = pubsub.Subscribe(topicForAck(2, pkiID), time.Second)
	wrappedHandler(notAck)
//确保消息已传递到包装的处理程序
	assert.Len(t, msgs, 1)
	_, err = sub.Listen()
//确保ACK未发布
	assert.Error(t, err)
}

func TestAck(t *testing.T) {
	t.Parallel()

	comm1, _ := newCommInstance(14000, naiveSec)
	comm2, _ := newCommInstance(14001, naiveSec)
	defer comm2.Stop()
	comm3, _ := newCommInstance(14002, naiveSec)
	defer comm3.Stop()
	comm4, _ := newCommInstance(14003, naiveSec)
	defer comm4.Stop()

	acceptData := func(o interface{}) bool {
		return o.(proto.ReceivedMessage).GetGossipMessage().IsDataMsg()
	}

	ack := func(c <-chan proto.ReceivedMessage) {
		msg := <-c
		msg.Ack(nil)
	}

	nack := func(c <-chan proto.ReceivedMessage) {
		msg := <-c
		msg.Ack(errors.New("Failed processing message because reasons"))
	}

//让实例2和3订阅数据消息，并确认它们
	inc2 := comm2.Accept(acceptData)
	inc3 := comm3.Accept(acceptData)

//收集2个ACK中的2个-应成功
	go ack(inc2)
	go ack(inc3)
	res := comm1.SendWithAck(createGossipMsg(), time.Second*3, 2, remotePeer(14001), remotePeer(14002))
	assert.Len(t, res, 2)
	assert.Empty(t, res[0].Error())
	assert.Empty(t, res[1].Error())

//收集3个ACK中的2个-应成功
	t1 := time.Now()
	go ack(inc2)
	go ack(inc3)
	res = comm1.SendWithAck(createGossipMsg(), time.Second*10, 2, remotePeer(14001), remotePeer(14002), remotePeer(14003))
	elapsed := time.Since(t1)
	assert.Len(t, res, 2)
	assert.Empty(t, res[0].Error())
	assert.Empty(t, res[1].Error())
//Collection of 2 out of 3 acks should have taken much less than the timeout (10 seconds)
	assert.True(t, elapsed < time.Second*5)

//收集3个ACK中的2个-应该失败，因为Peer3现在与ACK一起发送了一个错误。
	go ack(inc2)
	go nack(inc3)
	res = comm1.SendWithAck(createGossipMsg(), time.Second*10, 2, remotePeer(14001), remotePeer(14002), remotePeer(14003))
	assert.Len(t, res, 3)
	assert.Contains(t, []string{res[0].Error(), res[1].Error(), res[2].Error()}, "Failed processing message because reasons")
	assert.Contains(t, []string{res[0].Error(), res[1].Error(), res[2].Error()}, "timed out")

//收集2个ACK中的2个-应该失败，因为COMM2和COMM3现在不确认消息
	res = comm1.SendWithAck(createGossipMsg(), time.Second*3, 2, remotePeer(14001), remotePeer(14002))
	assert.Len(t, res, 2)
	assert.Contains(t, res[0].Error(), "timed out")
	assert.Contains(t, res[1].Error(), "timed out")
//排出ACK消息以准备下一次齐射
	<-inc2
	<-inc3

//收集3个ACK中的2个-应失败
	go ack(inc2)
	go nack(inc3)
	res = comm1.SendWithAck(createGossipMsg(), time.Second*3, 2, remotePeer(14001), remotePeer(14002), remotePeer(14003))
	assert.Len(t, res, 3)
assert.Contains(t, []string{res[0].Error(), res[1].Error(), res[2].Error()}, "") //这是“成功确认”
	assert.Contains(t, []string{res[0].Error(), res[1].Error(), res[2].Error()}, "Failed processing message because reasons")
	assert.Contains(t, []string{res[0].Error(), res[1].Error(), res[2].Error()}, "timed out")
	assert.Contains(t, res.String(), "\"Failed processing message because reasons\":1")
	assert.Contains(t, res.String(), "\"timed out\":1")
	assert.Contains(t, res.String(), "\"successes\":1")
	assert.Equal(t, 2, res.NackCount())
	assert.Equal(t, 1, res.AckCount())

//不向任何人发送消息
	res = comm1.SendWithAck(createGossipMsg(), time.Second*3, 1)
	assert.Len(t, res, 0)

//停止时发送消息
	comm1.Stop()
	res = comm1.SendWithAck(createGossipMsg(), time.Second*3, 1, remotePeer(14001), remotePeer(14002), remotePeer(14003))
	assert.Len(t, res, 3)
	assert.Contains(t, res[0].Error(), "comm is stopping")
	assert.Contains(t, res[1].Error(), "comm is stopping")
	assert.Contains(t, res[2].Error(), "comm is stopping")
}
