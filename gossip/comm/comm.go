
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
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

//comm是一个能够与其他对等方通信的对象。
//它还嵌入了一个通信模块。
type Comm interface {

//getpkiid返回此实例的pki id
	GetPKIid() common.PKIidType

//发送向远程对等发送消息
	Send(msg *proto.SignedGossipMessage, peers ...*RemotePeer)

//sendwithack向远程对等端发送消息，等待来自它们的minack的确认，或者直到某个超时结束。
	SendWithAck(msg *proto.SignedGossipMessage, timeout time.Duration, minAck int, peers ...*RemotePeer) AggregatedSendResult

//Probe探测一个远程节点，如果响应为零，则返回nil。
//如果不是的话也会出错。
	Probe(peer *RemotePeer) error

//握手验证远程对等机并返回
//（其身份，无）成功和（无，错误）
	Handshake(peer *RemotePeer) (api.PeerIdentityType, error)

//accept返回与某个谓词匹配的其他节点发送的消息的专用只读通道。
//来自通道的每条消息都可用于向发送者发送回复。
	Accept(common.MessageAcceptor) <-chan proto.ReceivedMessage

//ExpertedHead返回怀疑处于脱机状态的节点终结点的只读通道
	PresumedDead() <-chan common.PKIidType

//closeconn关闭到某个端点的连接
	CloseConn(peer *RemotePeer)

//停止停止模块
	Stop()
}

//远程对等定义对等端的端点及其pkiid
type RemotePeer struct {
	Endpoint string
	PKIID    common.PKIidType
}

//sendResult定义发送到远程对等机的结果
type SendResult struct {
	error
	RemotePeer
}

//错误返回sendResult的错误或空字符串
//如果没有发生错误
func (sr SendResult) Error() string {
	if sr.error != nil {
		return sr.error.Error()
	}
	return ""
}

//AggregatedSendResult表示一个sendResults切片
type AggregatedSendResult []SendResult

//AckCount返回成功确认的次数
func (ar AggregatedSendResult) AckCount() int {
	c := 0
	for _, ack := range ar {
		if ack.error == nil {
			c++
		}
	}
	return c
}

//nackcount返回未成功确认的数目
func (ar AggregatedSendResult) NackCount() int {
	return len(ar) - ar.AckCount()
}

//字符串返回JSED字符串表示形式
//聚合发送结果的
func (ar AggregatedSendResult) String() string {
	errMap := map[string]int{}
	for _, ack := range ar {
		if ack.error == nil {
			continue
		}
		errMap[ack.Error()]++
	}

	ackCount := ar.AckCount()
	output := map[string]interface{}{}
	if ackCount > 0 {
		output["successes"] = ackCount
	}
	if ackCount < len(ar) {
		output["failures"] = errMap
	}
	b, _ := json.Marshal(output)
	return string(b)
}

//字符串将远程对等机转换为字符串
func (p *RemotePeer) String() string {
	return fmt.Sprintf("%s, PKIid:%v", p.Endpoint, p.PKIID)
}
