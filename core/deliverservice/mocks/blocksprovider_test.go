
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

                 http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package mocks

import (
	"context"
	"math"
	"testing"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/stretchr/testify/assert"
)

func TestMockBlocksDeliverer(t *testing.T) {
//确保它实现BlocksDeliverer
	var bd blocksprovider.BlocksDeliverer
	bd = &MockBlocksDeliverer{}
	_ = bd

	assert.Panics(t, func() {
		bd.Recv()
	})
	bd.(*MockBlocksDeliverer).MockRecv = func(mock *MockBlocksDeliverer) (*orderer.DeliverResponse, error) {
		return &orderer.DeliverResponse{
			Type: &orderer.DeliverResponse_Status{
				Status: common.Status_FORBIDDEN,
			},
		}, nil
	}
	status, err := bd.Recv()
	assert.Nil(t, err)
	assert.Equal(t, common.Status_FORBIDDEN, status.GetStatus())
	bd.(*MockBlocksDeliverer).MockRecv = MockRecv
	block, err := bd.Recv()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), block.GetBlock().Header.Number)

	bd.(*MockBlocksDeliverer).Close()

	seekInfo := &orderer.SeekInfo{
		Start:    &orderer.SeekPosition{Type: &orderer.SeekPosition_Oldest{Oldest: &orderer.SeekOldest{}}},
		Stop:     &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: math.MaxUint64}}},
		Behavior: orderer.SeekInfo_BLOCK_UNTIL_READY,
	}
	si, err := pb.Marshal(seekInfo)
	assert.NoError(t, err)

	payload := &common.Payload{}

	payload.Data = si
	b, err := pb.Marshal(payload)
	assert.NoError(t, err)
	assert.Nil(t, bd.Send(&common.Envelope{Payload: b}))
}

func TestMockGossipServiceAdapter(t *testing.T) {
//确保它实现了GossipServiceAdapter
	var gsa blocksprovider.GossipServiceAdapter
	seqNums := make(chan uint64, 1)
	gsa = &MockGossipServiceAdapter{GossipBlockDisseminations: seqNums}
	_ = gsa

//试探流言蜚语
	msg := &proto.GossipMessage{
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{
				Payload: &proto.Payload{
					SeqNum: uint64(100),
				},
			},
		},
	}
	gsa.Gossip(msg)
	select {
	case seq := <-seqNums:
		assert.Equal(t, uint64(100), seq)
	case <-time.After(time.Second):
		assert.Fail(t, "Didn't gossip within a timely manner")
	}

//测试附加负载
	gsa.AddPayload("TEST", msg.GetDataMsg().Payload)
	assert.Equal(t, int32(1), gsa.(*MockGossipServiceAdapter).AddPayloadCount())

//测试对等通道
	assert.Len(t, gsa.PeersOfChannel(nil), 0)
}

func TestMockAtomicBroadcastClient(t *testing.T) {
//确保它实现mockatomicbroadcastclient
	var abc orderer.AtomicBroadcastClient
	abc = &MockAtomicBroadcastClient{BD: &MockBlocksDeliverer{}}

	assert.Panics(t, func() {
		abc.Broadcast(context.Background())
	})
	c, err := abc.Deliver(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, c)
}

func TestMockLedgerInfo(t *testing.T) {
	var li blocksprovider.LedgerInfo
	li = &MockLedgerInfo{uint64(8)}
	_ = li

	height, err := li.LedgerHeight()
	assert.Equal(t, uint64(8), height)
	assert.NoError(t, err)
}
