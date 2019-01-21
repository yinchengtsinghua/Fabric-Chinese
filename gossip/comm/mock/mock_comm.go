
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
	"time"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

//模拟插座
type socketMock struct {
//套接字端点
	endpoint string

//模拟简单TCP套接字
	socket chan interface{}
}

//原始TCP包结构的模拟
type packetMock struct {
//发件人频道消息从发送
	src *socketMock

//目标频道发送到
	dst *socketMock

	msg interface{}
}

type channelMock struct {
	accept common.MessageAcceptor

	channel chan proto.ReceivedMessage
}

type commMock struct {
	id string

	members map[string]*socketMock

	acceptors []*channelMock

	deadChannel chan common.PKIidType

	done chan struct{}
}

var logger = util.GetLogger(util.CommMockLogger, "")

//NewcommMock创建模拟通信对象
func NewCommMock(id string, members map[string]*socketMock) comm.Comm {
	res := &commMock{
		id: id,

		members: members,

		acceptors: make([]*channelMock, 0),

		done: make(chan struct{}),

		deadChannel: make(chan common.PKIidType),
	}
//启动通信服务
	go res.start()

	return res
}

//response将一条八卦消息发送到发送此接收消息的来源。
func (packet *packetMock) Respond(msg *proto.GossipMessage) {
	sMsg, _ := msg.NoopSign()
	packet.src.socket <- &packetMock{
		src: packet.dst,
		dst: packet.src,
		msg: sMsg,
	}
}

//ACK向发送方返回消息确认
func (packet *packetMock) Ack(err error) {

}

//GetSourceEnvelope返回接收到的消息所在的信封
//建筑用
func (packet *packetMock) GetSourceEnvelope() *proto.Envelope {
	return nil
}

//GetGossipMessage返回基础的GossipMessage
func (packet *packetMock) GetGossipMessage() *proto.SignedGossipMessage {
	return packet.msg.(*proto.SignedGossipMessage)
}

//getConnectionInfo返回有关远程对等机的信息
//发出信息的
func (packet *packetMock) GetConnectionInfo() *proto.ConnectionInfo {
	return nil
}

func (mock *commMock) start() {
	logger.Debug("Starting communication mock module...")
	for {
		select {
		case <-mock.done:
			{
//收到最后信号，正在退出…
				logger.Debug("Exiting...")
				return
			}
		case msg := <-mock.members[mock.id].socket:
			{
				logger.Debug("Got new message", msg)
				packet := msg.(*packetMock)
				for _, channel := range mock.acceptors {
//如果消息接受者同意
//新消息将其转发到接收的
//消息通道
					if channel.accept(packet) {
						channel.channel <- packet
					}
				}
			}
		}
	}
}

//getpkiid返回此实例的pki id
func (mock *commMock) GetPKIid() common.PKIidType {
	return common.PKIidType(mock.id)
}

//发送向远程对等发送消息
func (mock *commMock) Send(msg *proto.SignedGossipMessage, peers ...*comm.RemotePeer) {
	for _, peer := range peers {
		logger.Debug("Sending message to peer ", peer.Endpoint, "from ", mock.id)
		mock.members[peer.Endpoint].socket <- &packetMock{
			src: mock.members[mock.id],
			dst: mock.members[peer.Endpoint],
			msg: msg,
		}
	}
}

func (mock *commMock) SendWithAck(_ *proto.SignedGossipMessage, _ time.Duration, _ int, _ ...*comm.RemotePeer) comm.AggregatedSendResult {
	panic("not implemented")
}

//Probe探测一个远程节点，如果响应为零，则返回nil。
//如果不是的话也会出错。
func (mock *commMock) Probe(peer *comm.RemotePeer) error {
	return nil
}

//握手验证远程对等机并返回
//（其身份，无）成功和（无，错误）
func (mock *commMock) Handshake(peer *comm.RemotePeer) (api.PeerIdentityType, error) {
	return nil, nil
}

//accept返回与某个谓词匹配的其他节点发送的消息的专用只读通道。
//来自通道的每条消息都可用于向发送者发送回复。
func (mock *commMock) Accept(accept common.MessageAcceptor) <-chan proto.ReceivedMessage {
	ch := make(chan proto.ReceivedMessage)
	mock.acceptors = append(mock.acceptors, &channelMock{accept, ch})
	return ch
}

//ExpertedHead返回怀疑处于脱机状态的节点终结点的只读通道
func (mock *commMock) PresumedDead() <-chan common.PKIidType {
	return mock.deadChannel
}

//closeconn关闭到某个端点的连接
func (mock *commMock) CloseConn(peer *comm.RemotePeer) {
//诺普
}

//停止停止模块
func (mock *commMock) Stop() {
	logger.Debug("Stopping communication module, closing all accepting channels.")
	for _, accept := range mock.acceptors {
		close(accept.channel)
	}
	logger.Debug("[XXX]: Sending done signal to close the module.")
	mock.done <- struct{}{}
}
