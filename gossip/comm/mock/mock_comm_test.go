
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


package mock

import (
	"testing"

	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
)

func TestMockComm(t *testing.T) {
	first := &socketMock{"first", make(chan interface{})}
	second := &socketMock{"second", make(chan interface{})}
	members := make(map[string]*socketMock)

	members[first.endpoint] = first
	members[second.endpoint] = second

	comm1 := NewCommMock(first.endpoint, members)
	defer comm1.Stop()

	msgCh := comm1.Accept(func(message interface{}) bool {
		return message.(proto.ReceivedMessage).GetGossipMessage().GetStateRequest() != nil ||
			message.(proto.ReceivedMessage).GetGossipMessage().GetStateResponse() != nil
	})

	comm2 := NewCommMock(second.endpoint, members)
	defer comm2.Stop()

	sMsg, _ := (&proto.GossipMessage{
		Content: &proto.GossipMessage_StateRequest{StateRequest: &proto.RemoteStateRequest{
			StartSeqNum: 1,
			EndSeqNum:   3,
		}},
	}).NoopSign()
	comm2.Send(sMsg, &comm.RemotePeer{Endpoint: "first", PKIID: common.PKIidType("first")})

	msg := <-msgCh

	assert.NotNil(t, msg.GetGossipMessage().GetStateRequest())
	assert.Equal(t, "first", string(comm1.GetPKIid()))
}

func TestMockComm_PingPong(t *testing.T) {
	members := make(map[string]*socketMock)

	members["peerA"] = &socketMock{"peerA", make(chan interface{})}
	members["peerB"] = &socketMock{"peerB", make(chan interface{})}

	peerA := NewCommMock("peerA", members)
	peerB := NewCommMock("peerB", members)

	all := func(interface{}) bool {
		return true
	}

	rcvChA := peerA.Accept(all)
	rcvChB := peerB.Accept(all)

	sMsg, _ := (&proto.GossipMessage{
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{
				Payload: &proto.Payload{
					SeqNum: 1,
					Data:   []byte("Ping"),
				},
			}},
	}).NoopSign()
	peerA.Send(sMsg, &comm.RemotePeer{Endpoint: "peerB", PKIID: common.PKIidType("peerB")})

	msg := <-rcvChB
	dataMsg := msg.GetGossipMessage().GetDataMsg()
	data := string(dataMsg.Payload.Data)
	assert.Equal(t, "Ping", data)

	msg.Respond(&proto.GossipMessage{
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{
				Payload: &proto.Payload{
					SeqNum: 1,
					Data:   []byte("Pong"),
				},
			}},
	})

	msg = <-rcvChA
	dataMsg = msg.GetGossipMessage().GetDataMsg()
	data = string(dataMsg.Payload.Data)
	assert.Equal(t, "Pong", data)

}
