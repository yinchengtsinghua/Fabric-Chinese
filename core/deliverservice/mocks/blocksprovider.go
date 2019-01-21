
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


package mocks

import (
	"context"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	gossip_common "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/protos/common"
	gossip_proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"google.golang.org/grpc"
)

//用于初始化八卦服务的MockgossipsServiceAdapter模拟结构
//块提供程序实现并断言数字
//使用了个函数调用。
type MockGossipServiceAdapter struct {
	addPayloadCnt int32

	GossipBlockDisseminations chan uint64
}

type MockAtomicBroadcastClient struct {
	BD *MockBlocksDeliverer
}

func (mabc *MockAtomicBroadcastClient) Broadcast(ctx context.Context, opts ...grpc.CallOption) (orderer.AtomicBroadcast_BroadcastClient, error) {
	panic("Should not be used")
}
func (mabc *MockAtomicBroadcastClient) Deliver(ctx context.Context, opts ...grpc.CallOption) (orderer.AtomicBroadcast_DeliverClient, error) {
	return mabc.BD, nil
}

//peersofchannel返回具有参与给定通道的对等方的切片
func (*MockGossipServiceAdapter) PeersOfChannel(gossip_common.ChainID) []discovery.NetworkMember {
	return []discovery.NetworkMember{}
}

//addPayload将八卦负载添加到本地状态传输缓冲区
func (mock *MockGossipServiceAdapter) AddPayload(chainID string, payload *gossip_proto.Payload) error {
	atomic.AddInt32(&mock.addPayloadCnt, 1)
	return nil
}

//addpayloadcount返回调用recv的次数。
func (mock *MockGossipServiceAdapter) AddPayloadCount() int32 {
	return atomic.LoadInt32(&mock.addPayloadCnt)
}

//向所有同龄人传递八卦信息
func (mock *MockGossipServiceAdapter) Gossip(msg *gossip_proto.GossipMessage) {
	mock.GossipBlockDisseminations <- msg.GetDataMsg().Payload.SeqNum
}

//要初始化的BlocksDeliverer接口的Mocking结构
//块提供程序实现
type MockBlocksDeliverer struct {
	DisconnectCalled           chan struct{}
	DisconnectAndDisableCalled chan struct{}
	CloseCalled                chan struct{}
	Pos                        uint64
	grpc.ClientStream
	recvCnt  int32
	MockRecv func(mock *MockBlocksDeliverer) (*orderer.DeliverResponse, error)
}

//recv从订购服务获取响应，当前模拟返回
//只有一个响应带有空块。
func (mock *MockBlocksDeliverer) Recv() (*orderer.DeliverResponse, error) {
	atomic.AddInt32(&mock.recvCnt, 1)
	return mock.MockRecv(mock)
}

//recvcount返回调用recv的次数。
func (mock *MockBlocksDeliverer) RecvCount() int32 {
	return atomic.LoadInt32(&mock.recvCnt)
}

//mock recv mock for the recv函数
func MockRecv(mock *MockBlocksDeliverer) (*orderer.DeliverResponse, error) {
	pos := mock.Pos

//下一个呼叫的提前位置
	mock.Pos++
	return &orderer.DeliverResponse{
		Type: &orderer.DeliverResponse_Block{
			Block: &common.Block{
				Header: &common.BlockHeader{
					Number:       pos,
					DataHash:     []byte{},
					PreviousHash: []byte{},
				},
				Data: &common.BlockData{
					Data: [][]byte{},
				},
			}},
	}, nil
}

//send发送信封并请求订购服务块
//现在被嘲笑，什么都不做
func (mock *MockBlocksDeliverer) Send(env *common.Envelope) error {
	payload, _ := utils.GetPayload(env)
	seekInfo := &orderer.SeekInfo{}

	proto.Unmarshal(payload.Data, seekInfo)

//读取起始位置
	switch t := seekInfo.Start.Type.(type) {
	case *orderer.SeekPosition_Oldest:
		mock.Pos = 0
	case *orderer.SeekPosition_Specified:
		mock.Pos = t.Specified.Number
	}
	return nil
}

func (mock *MockBlocksDeliverer) Disconnect(disableEndpoint bool) {
	if disableEndpoint {
		mock.DisconnectAndDisableCalled <- struct{}{}
	} else {
		mock.DisconnectCalled <- struct{}{}
	}
}

func (mock *MockBlocksDeliverer) Close() {
	if mock.CloseCalled == nil {
		return
	}
	mock.CloseCalled <- struct{}{}
}

func (mock *MockBlocksDeliverer) UpdateEndpoints(endpoints []string) {

}

func (mock *MockBlocksDeliverer) GetEndpoints() []string {
return []string{} //空切片
}

//mockledgerinfo需要的ledgerinfo接口的mocking实现
//用于测试初始化
type MockLedgerInfo struct {
	Height uint64
}

//LedgerHeight将模拟值返回到分类帐高度
func (li *MockLedgerInfo) LedgerHeight() (uint64, error) {
	return atomic.LoadUint64(&li.Height), nil
}
