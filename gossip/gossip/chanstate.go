
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
	"bytes"
	"sync"
	"sync/atomic"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip/channel"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

type channelState struct {
	stopping int32
	sync.RWMutex
	channels map[string]channel.GossipChannel
	g        *gossipServiceImpl
}

func (cs *channelState) stop() {
	if cs.isStopping() {
		return
	}
	atomic.StoreInt32(&cs.stopping, int32(1))
	cs.Lock()
	defer cs.Unlock()
	for _, gc := range cs.channels {
		gc.Stop()
	}
}

func (cs *channelState) isStopping() bool {
	return atomic.LoadInt32(&cs.stopping) == int32(1)
}

func (cs *channelState) lookupChannelForMsg(msg proto.ReceivedMessage) channel.GossipChannel {
	if msg.GetGossipMessage().IsStateInfoPullRequestMsg() {
		sipr := msg.GetGossipMessage().GetStateInfoPullReq()
		mac := sipr.Channel_MAC
		pkiID := msg.GetConnectionInfo().ID
		return cs.getGossipChannelByMAC(mac, pkiID)
	}
	return cs.lookupChannelForGossipMsg(msg.GetGossipMessage().GossipMessage)
}

func (cs *channelState) lookupChannelForGossipMsg(msg *proto.GossipMessage) channel.GossipChannel {
	if !msg.IsStateInfoMsg() {
//如果我们到达这里，则消息不是：
//1）状态信息请求
//2）状态信息
//因此，它已经被发送到一个证明它知道频道名称的对等端（美国）。
//过去正在发送StateInfo消息。
//因此，我们使用消息本身的通道名称。
		return cs.getGossipChannelByChainID(msg.Channel)
	}

//否则，这是一条StateInfo消息。
	stateInfMsg := msg.GetStateInfo()
	return cs.getGossipChannelByMAC(stateInfMsg.Channel_MAC, stateInfMsg.PkiId)
}

func (cs *channelState) getGossipChannelByMAC(receivedMAC []byte, pkiID common.PKIidType) channel.GossipChannel {
//迭代这些通道，并尝试找到一个
//MAC等于消息上的MAC。
//如果是，则签署消息的对等方知道通道的名称。
//因为它的pki-id是在验证消息时检查的。
	cs.RLock()
	defer cs.RUnlock()
	for chanName, gc := range cs.channels {
		mac := channel.GenerateMAC(pkiID, common.ChainID(chanName))
		if bytes.Equal(mac, receivedMAC) {
			return gc
		}
	}
	return nil
}

func (cs *channelState) getGossipChannelByChainID(chainID common.ChainID) channel.GossipChannel {
	if cs.isStopping() {
		return nil
	}
	cs.RLock()
	defer cs.RUnlock()
	return cs.channels[string(chainID)]
}

func (cs *channelState) joinChannel(joinMsg api.JoinChannelMessage, chainID common.ChainID) {
	if cs.isStopping() {
		return
	}
	cs.Lock()
	defer cs.Unlock()
	if gc, exists := cs.channels[string(chainID)]; !exists {
		pkiID := cs.g.comm.GetPKIid()
		ga := &gossipAdapterImpl{gossipServiceImpl: cs.g, Discovery: cs.g.disc}
		gc := channel.NewGossipChannel(pkiID, cs.g.selfOrg, cs.g.mcs, chainID, ga, joinMsg)
		cs.channels[string(chainID)] = gc
	} else {
		gc.ConfigureChannel(joinMsg)
	}
}

type gossipAdapterImpl struct {
	*gossipServiceImpl
	discovery.Discovery
}

func (ga *gossipAdapterImpl) GetConf() channel.Config {
	return channel.Config{
		ID:                          ga.conf.ID,
		MaxBlockCountToStore:        ga.conf.MaxBlockCountToStore,
		PublishStateInfoInterval:    ga.conf.PublishStateInfoInterval,
		PullInterval:                ga.conf.PullInterval,
		PullPeerNum:                 ga.conf.PullPeerNum,
		RequestStateInfoInterval:    ga.conf.RequestStateInfoInterval,
		BlockExpirationInterval:     ga.conf.PullInterval * 100,
		StateInfoCacheSweepInterval: ga.conf.PullInterval * 5,
		TimeForMembershipTracker:    ga.conf.TimeForMembershipTracker,
	}
}

func (ga *gossipAdapterImpl) Sign(msg *proto.GossipMessage) (*proto.SignedGossipMessage, error) {
	signer := func(msg []byte) ([]byte, error) {
		return ga.mcs.Sign(msg)
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: msg,
	}
	e, err := sMsg.Sign(signer)
	if err != nil {
		return nil, err
	}
	return &proto.SignedGossipMessage{
		Envelope:      e,
		GossipMessage: msg,
	}, nil
}

//流言蜚语
func (ga *gossipAdapterImpl) Gossip(msg *proto.SignedGossipMessage) {
	ga.gossipServiceImpl.emitter.Add(&emittedGossipMessage{
		SignedGossipMessage: msg,
		filter: func(_ common.PKIidType) bool {
			return true
		},
	})
}

//转发将消息发送到下一个跃点
func (ga *gossipAdapterImpl) Forward(msg proto.ReceivedMessage) {
	ga.gossipServiceImpl.emitter.Add(&emittedGossipMessage{
		SignedGossipMessage: msg.GetGossipMessage(),
		filter:              msg.GetConnectionInfo().ID.IsNotSameFilter,
	})
}

func (ga *gossipAdapterImpl) Send(msg *proto.SignedGossipMessage, peers ...*comm.RemotePeer) {
	ga.gossipServiceImpl.comm.Send(msg, peers...)
}

//如果消息无效，则validateInfoMessage返回错误
//否则为零
func (ga *gossipAdapterImpl) ValidateStateInfoMessage(msg *proto.SignedGossipMessage) error {
	return ga.gossipServiceImpl.validateStateInfoMsg(msg)
}

//GetOrgofPeer返回某个对等方的组织标识符
func (ga *gossipAdapterImpl) GetOrgOfPeer(PKIID common.PKIidType) api.OrgIdentityType {
	return ga.gossipServiceImpl.getOrgOfPeer(PKIID)
}

//GetIdentityByPkiid返回具有特定
//pkiid，如果找不到则为nil
func (ga *gossipAdapterImpl) GetIdentityByPKIID(pkiID common.PKIidType) api.PeerIdentityType {
	identity, err := ga.idMapper.Get(pkiID)
	if err != nil {
		return nil
	}
	return identity
}
