
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


package election

import (
	"bytes"
	"sync"
	"time"

	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

type msgImpl struct {
	msg *proto.GossipMessage
}

func (mi *msgImpl) SenderID() peerID {
	return mi.msg.GetLeadershipMsg().PkiId
}

func (mi *msgImpl) IsProposal() bool {
	return !mi.IsDeclaration()
}

func (mi *msgImpl) IsDeclaration() bool {
	return mi.msg.GetLeadershipMsg().IsDeclaration
}

type peerImpl struct {
	member discovery.NetworkMember
}

func (pi *peerImpl) ID() peerID {
	return peerID(pi.member.PKIid)
}

type gossip interface {
//对等方返回被认为是活动的网络成员
	Peers() []discovery.NetworkMember

//accept返回与某个谓词匹配的其他节点发送的消息的专用只读通道。
//如果passthrough为false，则消息将由八卦层预先处理。
//如果passthrough是真的，那么八卦层不会介入，消息也不会
//可用于将答复发送回发件人
	Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage)

//八卦消息向网络中的其他对等方发送消息
	Gossip(msg *proto.GossipMessage)
}

type adapterImpl struct {
	gossip    gossip
	selfPKIid common.PKIidType

	incTime uint64
	seqNum  uint64

	channel common.ChainID

	logger util.Logger

	doneCh   chan struct{}
	stopOnce *sync.Once
}

//NewAdapter创建新的Leader Election适配器
func NewAdapter(gossip gossip, pkiid common.PKIidType, channel common.ChainID) LeaderElectionAdapter {
	return &adapterImpl{
		gossip:    gossip,
		selfPKIid: pkiid,

		incTime: uint64(time.Now().UnixNano()),
		seqNum:  uint64(0),

		channel: channel,

		logger: util.GetLogger(util.ElectionLogger, ""),

		doneCh:   make(chan struct{}),
		stopOnce: &sync.Once{},
	}
}

func (ai *adapterImpl) Gossip(msg Msg) {
	ai.gossip.Gossip(msg.(*msgImpl).msg)
}

func (ai *adapterImpl) Accept() <-chan Msg {
	adapterCh, _ := ai.gossip.Accept(func(message interface{}) bool {
//仅获取领导组织和渠道消息
		return message.(*proto.GossipMessage).Tag == proto.GossipMessage_CHAN_AND_ORG &&
			message.(*proto.GossipMessage).IsLeadershipMsg() &&
			bytes.Equal(message.(*proto.GossipMessage).Channel, ai.channel)
	}, false)

	msgCh := make(chan Msg)

	go func(inCh <-chan *proto.GossipMessage, outCh chan Msg, stopCh chan struct{}) {
		for {
			select {
			case <-stopCh:
				return
			case gossipMsg, ok := <-inCh:
				if ok {
					outCh <- &msgImpl{gossipMsg}
				} else {
					return
				}
			}
		}
	}(adapterCh, msgCh, ai.doneCh)
	return msgCh
}

func (ai *adapterImpl) CreateMessage(isDeclaration bool) Msg {
	ai.seqNum++
	seqNum := ai.seqNum

	leadershipMsg := &proto.LeadershipMessage{
		PkiId:         ai.selfPKIid,
		IsDeclaration: isDeclaration,
		Timestamp: &proto.PeerTime{
			IncNum: ai.incTime,
			SeqNum: seqNum,
		},
	}

	msg := &proto.GossipMessage{
		Nonce:   0,
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Content: &proto.GossipMessage_LeadershipMsg{LeadershipMsg: leadershipMsg},
		Channel: ai.channel,
	}
	return &msgImpl{msg}
}

func (ai *adapterImpl) Peers() []Peer {
	peers := ai.gossip.Peers()

	var res []Peer
	for _, peer := range peers {
		res = append(res, &peerImpl{peer})
	}

	return res
}

func (ai *adapterImpl) Stop() {
	stopFunc := func() {
		close(ai.doneCh)
	}
	ai.stopOnce.Do(stopFunc)
}
