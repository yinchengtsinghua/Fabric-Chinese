
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
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

type sendFunc func(peer *RemotePeer, msg *proto.SignedGossipMessage)
type waitFunc func(*RemotePeer) error

type ackSendOperation struct {
	snd        sendFunc
	waitForAck waitFunc
}

func newAckSendOperation(snd sendFunc, waitForAck waitFunc) *ackSendOperation {
	return &ackSendOperation{
		snd:        snd,
		waitForAck: waitForAck,
	}
}

func (aso *ackSendOperation) send(msg *proto.SignedGossipMessage, minAckNum int, peers ...*RemotePeer) []SendResult {
	successAcks := 0
	results := []SendResult{}

	acks := make(chan SendResult, len(peers))
//将消息发送给所有对等方
	for _, p := range peers {
		go func(p *RemotePeer) {
//将消息发送到“P”
			aso.snd(p, msg)
//等待“p”的确认，或者在超时时出错。
			err := aso.waitForAck(p)
			acks <- SendResult{
				RemotePeer: *p,
				error:      err,
			}
		}(p)
	}
	for {
		ack := <-acks
		results = append(results, SendResult{
			error:      ack.error,
			RemotePeer: ack.RemotePeer,
		})
		if ack.error == nil {
			successAcks++
		}
		if successAcks == minAckNum || len(results) == len(peers) {
			break
		}
	}
	return results
}

func interceptAcks(nextHandler handler, remotePeerID common.PKIidType, pubSub *util.PubSub) func(*proto.SignedGossipMessage) {
	return func(m *proto.SignedGossipMessage) {
		if m.IsAck() {
			topic := topicForAck(m.Nonce, remotePeerID)
			pubSub.Publish(topic, m.GetAck())
			return
		}
		nextHandler(m)
	}
}
