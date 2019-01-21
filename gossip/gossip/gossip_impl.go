
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
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	protoG "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	"github.com/hyperledger/fabric/gossip/gossip/channel"
	"github.com/hyperledger/fabric/gossip/gossip/msgstore"
	"github.com/hyperledger/fabric/gossip/gossip/pull"
	"github.com/hyperledger/fabric/gossip/identity"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	presumedDeadChanSize = 100
	acceptChanSize       = 100
)

type channelRoutingFilterFactory func(channel.GossipChannel) filter.RoutingFilter

type gossipServiceImpl struct {
	selfIdentity          api.PeerIdentityType
	includeIdentityPeriod time.Time
	certStore             *certStore
	idMapper              identity.Mapper
	presumedDead          chan common.PKIidType
	disc                  discovery.Discovery
	comm                  comm.Comm
	incTime               time.Time
	selfOrg               api.OrgIdentityType
	*comm.ChannelDeMultiplexer
	logger            util.Logger
	stopSignal        *sync.WaitGroup
	conf              *Config
	toDieChan         chan struct{}
	stopFlag          int32
	emitter           batchingEmitter
	discAdapter       *discoveryAdapter
	secAdvisor        api.SecurityAdvisor
	chanState         *channelState
	disSecAdap        *discoverySecurityAdapter
	mcs               api.MessageCryptoService
	stateInfoMsgStore msgstore.MessageStore
	certPuller        pull.Mediator
}

//newgossipservice创建附加到GRPC服务器的八卦实例
func NewGossipService(conf *Config, s *grpc.Server, sa api.SecurityAdvisor,
	mcs api.MessageCryptoService, selfIdentity api.PeerIdentityType,
	secureDialOpts api.PeerSecureDialOpts) Gossip {
	var err error

	lgr := util.GetLogger(util.GossipLogger, conf.ID)

	g := &gossipServiceImpl{
		selfOrg:               sa.OrgByPeerIdentity(selfIdentity),
		secAdvisor:            sa,
		selfIdentity:          selfIdentity,
		presumedDead:          make(chan common.PKIidType, presumedDeadChanSize),
		disc:                  nil,
		mcs:                   mcs,
		conf:                  conf,
		ChannelDeMultiplexer:  comm.NewChannelDemultiplexer(),
		logger:                lgr,
		toDieChan:             make(chan struct{}, 1),
		stopFlag:              int32(0),
		stopSignal:            &sync.WaitGroup{},
		includeIdentityPeriod: time.Now().Add(conf.PublishCertPeriod),
	}
	g.stateInfoMsgStore = g.newStateInfoMsgStore()

	g.idMapper = identity.NewIdentityMapper(mcs, selfIdentity, func(pkiID common.PKIidType, identity api.PeerIdentityType) {
		g.comm.CloseConn(&comm.RemotePeer{PKIID: pkiID})
		g.certPuller.Remove(string(pkiID))
	}, sa)

	if s == nil {
		g.comm, err = createCommWithServer(conf.BindPort, g.idMapper, selfIdentity, secureDialOpts, sa)
	} else {
		g.comm, err = createCommWithoutServer(s, conf.TLSCerts, g.idMapper, selfIdentity, secureDialOpts, sa)
	}

	if err != nil {
		lgr.Error("Failed instntiating communication layer:", err)
		return nil
	}

	g.chanState = newChannelState(g)
	g.emitter = newBatchingEmitter(conf.PropagateIterations,
		conf.MaxPropagationBurstSize, conf.MaxPropagationBurstLatency,
		g.sendGossipBatch)

	g.discAdapter = g.newDiscoveryAdapter()
	g.disSecAdap = g.newDiscoverySecurityAdapter()
	g.disc = discovery.NewDiscoveryService(g.selfNetworkMember(), g.discAdapter, g.disSecAdap, g.disclosurePolicy)
	g.logger.Infof("Creating gossip service with self membership of %s", g.selfNetworkMember())

	g.certPuller = g.createCertStorePuller()
	g.certStore = newCertStore(g.certPuller, g.idMapper, selfIdentity, mcs)

	if g.conf.ExternalEndpoint == "" {
		g.logger.Warning("External endpoint is empty, peer will not be accessible outside of its organization")
	}
//为handlePreumedDead和添加delta
//在等待时接受要阻止的消息goroutine
	g.stopSignal.Add(2)
	go g.start()
	go g.connect2BootstrapPeers()

	return g
}

func (g *gossipServiceImpl) newStateInfoMsgStore() msgstore.MessageStore {
	pol := proto.NewGossipMessageComparator(0)
	return msgstore.NewMessageStoreExpirable(pol,
		msgstore.Noop,
		g.conf.PublishStateInfoInterval*100,
		nil,
		nil,
		msgstore.Noop)
}

func (g *gossipServiceImpl) selfNetworkMember() discovery.NetworkMember {
	self := discovery.NetworkMember{
		Endpoint:         g.conf.ExternalEndpoint,
		PKIid:            g.comm.GetPKIid(),
		Metadata:         []byte{},
		InternalEndpoint: g.conf.InternalEndpoint,
	}
	if g.disc != nil {
		self.Metadata = g.disc.Self().Metadata
	}
	return self
}

func newChannelState(g *gossipServiceImpl) *channelState {
	return &channelState{
		stopping: int32(0),
		channels: make(map[string]channel.GossipChannel),
		g:        g,
	}
}

func createCommWithoutServer(s *grpc.Server, certs *common.TLSCertificates, idStore identity.Mapper,
	identity api.PeerIdentityType, secureDialOpts api.PeerSecureDialOpts, sa api.SecurityAdvisor) (comm.Comm, error) {
	return comm.NewCommInstance(s, certs, idStore, identity, secureDialOpts, sa)
}

//newgossipserviceWithServer使用GRPC服务器创建新的八卦实例
func NewGossipServiceWithServer(conf *Config, secAdvisor api.SecurityAdvisor, mcs api.MessageCryptoService,
	identity api.PeerIdentityType, secureDialOpts api.PeerSecureDialOpts) Gossip {
	return NewGossipService(conf, nil, secAdvisor, mcs, identity, secureDialOpts)
}

func createCommWithServer(port int, idStore identity.Mapper, identity api.PeerIdentityType,
	secureDialOpts api.PeerSecureDialOpts, sa api.SecurityAdvisor) (comm.Comm, error) {
	return comm.NewCommInstanceWithServer(port, idStore, identity, secureDialOpts, sa)
}

func (g *gossipServiceImpl) toDie() bool {
	return atomic.LoadInt32(&g.stopFlag) == int32(1)
}

func (g *gossipServiceImpl) JoinChan(joinMsg api.JoinChannelMessage, chainID common.ChainID) {
//joinmsg应该已经过验证
	g.chanState.joinChannel(joinMsg, chainID)

	g.logger.Info("Joining gossip network of channel", string(chainID), "with", len(joinMsg.Members()), "organizations")
	for _, org := range joinMsg.Members() {
		g.learnAnchorPeers(string(chainID), org, joinMsg.AnchorPeersOf(org))
	}
}

func (g *gossipServiceImpl) LeaveChan(chainID common.ChainID) {
	gc := g.chanState.getGossipChannelByChainID(chainID)
	if gc == nil {
		g.logger.Debug("No such channel", chainID)
		return
	}
	gc.LeaveChannel()
}

//SuspectPeers使八卦实例验证可疑对等的身份，并关闭
//与标识无效的对等方的任何连接
func (g *gossipServiceImpl) SuspectPeers(isSuspected api.PeerSuspector) {
	g.certStore.suspectPeers(isSuspected)
}

func (g *gossipServiceImpl) periodicalIdentityValidation(suspectFunc api.PeerSuspector, interval time.Duration) {
	for {
		select {
		case s := <-g.toDieChan:
			g.toDieChan <- s
			return
		case <-time.After(interval):
			g.SuspectPeers(suspectFunc)
		}
	}
}

func (g *gossipServiceImpl) learnAnchorPeers(channel string, orgOfAnchorPeers api.OrgIdentityType, anchorPeers []api.AnchorPeer) {
	if len(anchorPeers) == 0 {
		g.logger.Info("No configured anchor peers of", string(orgOfAnchorPeers), "for channel", channel, "to learn about")
		return
	}
	g.logger.Info("Learning about the configured anchor peers of", string(orgOfAnchorPeers), "for channel", channel, ":", anchorPeers)
	for _, ap := range anchorPeers {
		if ap.Host == "" {
			g.logger.Warning("Got empty hostname, skipping connecting to anchor peer", ap)
			continue
		}
		if ap.Port == 0 {
			g.logger.Warning("Got invalid port (0), skipping connecting to anchor peer", ap)
			continue
		}
		endpoint := fmt.Sprintf("%s:%d", ap.Host, ap.Port)
//跳过连接到自身
		if g.selfNetworkMember().Endpoint == endpoint || g.selfNetworkMember().InternalEndpoint == endpoint {
			g.logger.Info("Anchor peer with same endpoint, skipping connecting to myself")
			continue
		}

		inOurOrg := bytes.Equal(g.selfOrg, orgOfAnchorPeers)
		if !inOurOrg && g.selfNetworkMember().Endpoint == "" {
			g.logger.Infof("Anchor peer %s:%d isn't in our org(%v) and we have no external endpoint, skipping", ap.Host, ap.Port, string(orgOfAnchorPeers))
			continue
		}
		identifier := func() (*discovery.PeerIdentification, error) {
			remotePeerIdentity, err := g.comm.Handshake(&comm.RemotePeer{Endpoint: endpoint})
			if err != nil {
				err = errors.WithStack(err)
				g.logger.Warningf("Deep probe of %s failed: %+v", endpoint, err)
				return nil, err
			}
			isAnchorPeerInMyOrg := bytes.Equal(g.selfOrg, g.secAdvisor.OrgByPeerIdentity(remotePeerIdentity))
			if bytes.Equal(orgOfAnchorPeers, g.selfOrg) && !isAnchorPeerInMyOrg {
				err := errors.Errorf("Anchor peer %s isn't in our org, but is claimed to be", endpoint)
				g.logger.Warningf("%+v", err)
				return nil, err
			}
			pkiID := g.mcs.GetPKIidOfCert(remotePeerIdentity)
			if len(pkiID) == 0 {
				return nil, errors.Errorf("Wasn't able to extract PKI-ID of remote peer with identity of %v", remotePeerIdentity)
			}
			return &discovery.PeerIdentification{
				ID:      pkiID,
				SelfOrg: isAnchorPeerInMyOrg,
			}, nil
		}

		g.disc.Connect(discovery.NetworkMember{
			InternalEndpoint: endpoint, Endpoint: endpoint}, identifier)
	}
}

func (g *gossipServiceImpl) handlePresumedDead() {
	defer g.logger.Debug("Exiting")
	defer g.stopSignal.Done()
	for {
		select {
		case s := <-g.toDieChan:
			g.toDieChan <- s
			return
		case deadEndpoint := <-g.comm.PresumedDead():
			g.presumedDead <- deadEndpoint
		}
	}
}

func (g *gossipServiceImpl) syncDiscovery() {
	g.logger.Debug("Entering discovery sync with interval", g.conf.PullInterval)
	defer g.logger.Debug("Exiting discovery sync loop")
	for !g.toDie() {
		g.disc.InitiateSync(g.conf.PullPeerNum)
		time.Sleep(g.conf.PullInterval)
	}
}

func (g *gossipServiceImpl) start() {
	go g.syncDiscovery()
	go g.handlePresumedDead()

	msgSelector := func(msg interface{}) bool {
		gMsg, isGossipMsg := msg.(proto.ReceivedMessage)
		if !isGossipMsg {
			return false
		}

		isConn := gMsg.GetGossipMessage().GetConn() != nil
		isEmpty := gMsg.GetGossipMessage().GetEmpty() != nil
		isPrivateData := gMsg.GetGossipMessage().IsPrivateDataMsg()

		return !(isConn || isEmpty || isPrivateData)
	}

	incMsgs := g.comm.Accept(msgSelector)

	go g.acceptMessages(incMsgs)

	g.logger.Info("Gossip instance", g.conf.ID, "started")
}

func (g *gossipServiceImpl) acceptMessages(incMsgs <-chan proto.ReceivedMessage) {
	defer g.logger.Debug("Exiting")
	defer g.stopSignal.Done()
	for {
		select {
		case s := <-g.toDieChan:
			g.toDieChan <- s
			return
		case msg := <-incMsgs:
			g.handleMessage(msg)
		}
	}
}

func (g *gossipServiceImpl) handleMessage(m proto.ReceivedMessage) {
	if g.toDie() {
		return
	}

	if m == nil || m.GetGossipMessage() == nil {
		return
	}

	msg := m.GetGossipMessage()

	g.logger.Debug("Entering,", m.GetConnectionInfo(), "sent us", msg)
	defer g.logger.Debug("Exiting")

	if !g.validateMsg(m) {
		g.logger.Warning("Message", msg, "isn't valid")
		return
	}

	if msg.IsChannelRestricted() {
		if gc := g.chanState.lookupChannelForMsg(m); gc == nil {
//如果我们不在这个频道，我们还是应该转发给我们组织的同行。
//以防是StateInfo消息
			if g.isInMyorg(discovery.NetworkMember{PKIid: m.GetConnectionInfo().ID}) && msg.IsStateInfoMsg() {
				if g.stateInfoMsgStore.Add(msg) {
					g.emitter.Add(&emittedGossipMessage{
						SignedGossipMessage: msg,
						filter:              m.GetConnectionInfo().ID.IsNotSameFilter,
					})
				}
			}
			if !g.toDie() {
				g.logger.Debug("No such channel", msg.Channel, "discarding message", msg)
			}
		} else {
			if m.GetGossipMessage().IsLeadershipMsg() {
				if err := g.validateLeadershipMessage(m.GetGossipMessage()); err != nil {
					g.logger.Warningf("Failed validating LeaderElection message: %+v", errors.WithStack(err))
					return
				}
			}
			gc.HandleMessage(m)
		}
		return
	}

	if selectOnlyDiscoveryMessages(m) {
//这是一个会员申请，检查它的个人信息
//与发件人匹配
		if m.GetGossipMessage().GetMemReq() != nil {
			sMsg, err := m.GetGossipMessage().GetMemReq().SelfInformation.ToGossipMessage()
			if err != nil {
				g.logger.Warningf("Got membership request with invalid selfInfo: %+v", errors.WithStack(err))
				return
			}
			if !sMsg.IsAliveMsg() {
				g.logger.Warning("Got membership request with selfInfo that isn't an AliveMessage")
				return
			}
			if !bytes.Equal(sMsg.GetAliveMsg().Membership.PkiId, m.GetConnectionInfo().ID) {
				g.logger.Warning("Got membership request with selfInfo that doesn't match the handshake")
				return
			}
		}
		g.forwardDiscoveryMsg(m)
	}

	if msg.IsPullMsg() && msg.GetPullMsgType() == proto.PullMsgType_IDENTITY_MSG {
		g.certStore.handleMessage(m)
	}
}

func (g *gossipServiceImpl) forwardDiscoveryMsg(msg proto.ReceivedMessage) {
defer func() { //关闭时可以关闭
		recover()
	}()

	g.discAdapter.incChan <- msg
}

//validatemsg检查消息的签名（如果存在），
//并检查标签是否与消息类型匹配
func (g *gossipServiceImpl) validateMsg(msg proto.ReceivedMessage) bool {
	if err := msg.GetGossipMessage().IsTagLegal(); err != nil {
		g.logger.Warningf("Tag of %v isn't legal: %v", msg.GetGossipMessage(), errors.WithStack(err))
		return false
	}

	if msg.GetGossipMessage().IsStateInfoMsg() {
		if err := g.validateStateInfoMsg(msg.GetGossipMessage()); err != nil {
			g.logger.Warningf("StateInfo message %v is found invalid: %v", msg, err)
			return false
		}
	}
	return true
}

func (g *gossipServiceImpl) sendGossipBatch(a []interface{}) {
	msgs2Gossip := make([]*emittedGossipMessage, len(a))
	for i, e := range a {
		msgs2Gossip[i] = e.(*emittedGossipMessage)
	}
	g.gossipBatch(msgs2Gossip)
}

//gossippatch-这是一种方法，它实际决定要向哪个对等方传递消息。
//我们拥有的一批。
//为了提高效率，我们首先隔离具有相同路由策略的所有消息
//然后把它们一起发送，然后才转到下一组消息。
//即：我们将信道C的所有块发送到同一组对等端，
//并将所有stateinfo消息发送到同一组对等机等。
//当我们发送数据块时，我们只发送给在通道中做广告的对等端。
//当我们发送StateInfo消息时，我们会发送到通道中的对等端。
//当我们发送标记为只在组织内发送的消息时，我们发送所有这些消息
//同一组同龄人。
//可以发送对其目的地没有限制的其余消息
//任何一组同龄人。
func (g *gossipServiceImpl) gossipBatch(msgs []*emittedGossipMessage) {
	if g.disc == nil {
		g.logger.Error("Discovery has not been initialized yet, aborting!")
		return
	}

	var blocks []*emittedGossipMessage
	var stateInfoMsgs []*emittedGossipMessage
	var orgMsgs []*emittedGossipMessage
	var leadershipMsgs []*emittedGossipMessage

	isABlock := func(o interface{}) bool {
		return o.(*emittedGossipMessage).IsDataMsg()
	}
	isAStateInfoMsg := func(o interface{}) bool {
		return o.(*emittedGossipMessage).IsStateInfoMsg()
	}
	aliveMsgsWithNoEndpointAndInOurOrg := func(o interface{}) bool {
		msg := o.(*emittedGossipMessage)
		if !msg.IsAliveMsg() {
			return false
		}
		member := msg.GetAliveMsg().Membership
		return member.Endpoint == "" && g.isInMyorg(discovery.NetworkMember{PKIid: member.PkiId})
	}
	isOrgRestricted := func(o interface{}) bool {
		return aliveMsgsWithNoEndpointAndInOurOrg(o) || o.(*emittedGossipMessage).IsOrgRestricted()
	}
	isLeadershipMsg := func(o interface{}) bool {
		return o.(*emittedGossipMessage).IsLeadershipMsg()
	}

//闲话街区
	blocks, msgs = partitionMessages(isABlock, msgs)
	g.gossipInChan(blocks, func(gc channel.GossipChannel) filter.RoutingFilter {
		return filter.CombineRoutingFilters(gc.EligibleForChannel, gc.IsMemberInChan, g.isInMyorg)
	})

//八卦领导力信息
	leadershipMsgs, msgs = partitionMessages(isLeadershipMsg, msgs)
	g.gossipInChan(leadershipMsgs, func(gc channel.GossipChannel) filter.RoutingFilter {
		return filter.CombineRoutingFilters(gc.EligibleForChannel, gc.IsMemberInChan, g.isInMyorg)
	})

//八卦状态信息
	stateInfoMsgs, msgs = partitionMessages(isAStateInfoMsg, msgs)
	for _, stateInfMsg := range stateInfoMsgs {
		peerSelector := g.isInMyorg
		gc := g.chanState.lookupChannelForGossipMsg(stateInfMsg.GossipMessage)
		if gc != nil && g.hasExternalEndpoint(stateInfMsg.GossipMessage.GetStateInfo().PkiId) {
			peerSelector = gc.IsMemberInChan
		}

		peerSelector = filter.CombineRoutingFilters(peerSelector, func(member discovery.NetworkMember) bool {
			return stateInfMsg.filter(member.PKIid)
		})

		peers2Send := filter.SelectPeers(g.conf.PropagatePeerNum, g.disc.GetMembership(), peerSelector)
		g.comm.Send(stateInfMsg.SignedGossipMessage, peers2Send...)
	}

//仅限于我们组织的八卦消息
	orgMsgs, msgs = partitionMessages(isOrgRestricted, msgs)
	peers2Send := filter.SelectPeers(g.conf.PropagatePeerNum, g.disc.GetMembership(), g.isInMyorg)
	for _, msg := range orgMsgs {
		g.comm.Send(msg.SignedGossipMessage, g.removeSelfLoop(msg, peers2Send)...)
	}

//最后，闲聊剩下的信息
	for _, msg := range msgs {
		if !msg.IsAliveMsg() {
			g.logger.Error("Unknown message type", msg)
			continue
		}
		selectByOriginOrg := g.peersByOriginOrgPolicy(discovery.NetworkMember{PKIid: msg.GetAliveMsg().Membership.PkiId})
		selector := filter.CombineRoutingFilters(selectByOriginOrg, func(member discovery.NetworkMember) bool {
			return msg.filter(member.PKIid)
		})
		peers2Send := filter.SelectPeers(g.conf.PropagatePeerNum, g.disc.GetMembership(), selector)
		g.sendAndFilterSecrets(msg.SignedGossipMessage, peers2Send...)
	}
}

func (g *gossipServiceImpl) sendAndFilterSecrets(msg *proto.SignedGossipMessage, peers ...*comm.RemotePeer) {
	for _, peer := range peers {
//阻止转发外部组织的活动消息
//到没有外部终结点的对等机
		aliveMsgFromDiffOrg := msg.IsAliveMsg() && !g.isInMyorg(discovery.NetworkMember{PKIid: msg.GetAliveMsg().Membership.PkiId})
		if aliveMsgFromDiffOrg && !g.hasExternalEndpoint(peer.PKIID) {
			continue
		}
//不要八卦秘密
		if !g.isInMyorg(discovery.NetworkMember{PKIid: peer.PKIID}) {
			msg.Envelope.SecretEnvelope = nil
		}

		g.comm.Send(msg, peer)
	}
}

//根据信道的路由策略，gossipinchan gossip是一个给定的gossipMessage切片。
func (g *gossipServiceImpl) gossipInChan(messages []*emittedGossipMessage, chanRoutingFactory channelRoutingFilterFactory) {
	if len(messages) == 0 {
		return
	}
	totalChannels := extractChannels(messages)
	var channel common.ChainID
	var messagesOfChannel []*emittedGossipMessage
	for len(totalChannels) > 0 {
//走第一个通道
		channel, totalChannels = totalChannels[0], totalChannels[1:]
//提取该频道的所有消息
		grabMsgs := func(o interface{}) bool {
			return bytes.Equal(o.(*emittedGossipMessage).Channel, channel)
		}
		messagesOfChannel, messages = partitionMessages(grabMsgs, messages)
		if len(messagesOfChannel) == 0 {
			continue
		}
//为该通道获取通道对象
		gc := g.chanState.getGossipChannelByChainID(channel)
		if gc == nil {
			g.logger.Warning("Channel", channel, "wasn't found")
			continue
		}
//选择要向其发送消息的对等方
//对于领导信息，我们将选择通过路由工厂的所有对等方，例如渠道和组织中的所有对等方。
		membership := g.disc.GetMembership()
		var peers2Send []*comm.RemotePeer
		if messagesOfChannel[0].IsLeadershipMsg() {
			peers2Send = filter.SelectPeers(len(membership), membership, chanRoutingFactory(gc))
		} else {
			peers2Send = filter.SelectPeers(g.conf.PropagatePeerNum, membership, chanRoutingFactory(gc))
		}

//将消息发送到远程对等端
		for _, msg := range messagesOfChannel {
			filteredPeers := g.removeSelfLoop(msg, peers2Send)
			g.comm.Send(msg.SignedGossipMessage, filteredPeers...)
		}
	}
}

//removeselflop从发送消息的对等机列表中删除
func (g *gossipServiceImpl) removeSelfLoop(msg *emittedGossipMessage, peers []*comm.RemotePeer) []*comm.RemotePeer {
	var result []*comm.RemotePeer
	for _, peer := range peers {
		if msg.filter(peer.PKIID) {
			result = append(result, peer)
		}
	}
	return result
}

//IdentityInfo返回已知对等标识的信息
func (g *gossipServiceImpl) IdentityInfo() api.PeerIdentitySet {
	return g.idMapper.IdentityInfo()
}

//sendbyCriteria将给定消息发送给与给定发送条件匹配的所有对等方
func (g *gossipServiceImpl) SendByCriteria(msg *proto.SignedGossipMessage, criteria SendCriteria) error {
	if criteria.MaxPeers == 0 {
		return nil
	}
	if criteria.Timeout == 0 {
		return errors.New("Timeout should be specified")
	}

	if criteria.IsEligible == nil {
		criteria.IsEligible = filter.SelectAllPolicy
	}

	membership := g.disc.GetMembership()

	if len(criteria.Channel) > 0 {
		gc := g.chanState.getGossipChannelByChainID(criteria.Channel)
		if gc == nil {
			return fmt.Errorf("Requested to Send for channel %s, but no such channel exists", string(criteria.Channel))
		}
		membership = gc.GetPeers()
	}

	peers2send := filter.SelectPeers(criteria.MaxPeers, membership, criteria.IsEligible)
	if len(peers2send) < criteria.MinAck {
		return fmt.Errorf("Requested to send to at least %d peers, but know only of %d suitable peers", criteria.MinAck, len(peers2send))
	}

	results := g.comm.SendWithAck(msg, criteria.Timeout, criteria.MinAck, peers2send...)

	for _, res := range results {
		if res.Error() == "" {
			continue
		}
		g.logger.Warning("Failed sending to", res.Endpoint, "error:", res.Error())
	}

	if results.AckCount() < criteria.MinAck {
		return errors.New(results.String())
	}
	return nil
}

//八卦消息向网络中的其他对等方发送消息
func (g *gossipServiceImpl) Gossip(msg *proto.GossipMessage) {
//教育开发人员使用正确的标签来闲聊信息。
//有关所需行为，请参阅istagegal（）。
	if err := msg.IsTagLegal(); err != nil {
		panic(errors.WithStack(err))
	}

	sMsg := &proto.SignedGossipMessage{
		GossipMessage: msg,
	}

	var err error
	if sMsg.IsDataMsg() {
		sMsg, err = sMsg.NoopSign()
	} else {
		_, err = sMsg.Sign(func(msg []byte) ([]byte, error) {
			return g.mcs.Sign(msg)
		})
	}

	if err != nil {
		g.logger.Warningf("Failed signing message: %+v", errors.WithStack(err))
		return
	}

	if msg.IsChannelRestricted() {
		gc := g.chanState.getGossipChannelByChainID(msg.Channel)
		if gc == nil {
			g.logger.Warning("Failed obtaining gossipChannel of", msg.Channel, "aborting")
			return
		}
		if msg.IsDataMsg() {
			gc.AddToMsgStore(sMsg)
		}
	}

	if g.conf.PropagateIterations == 0 {
		return
	}
	g.emitter.Add(&emittedGossipMessage{
		SignedGossipMessage: sMsg,
		filter: func(_ common.PKIidType) bool {
			return true
		},
	})
}

//发送向远程对等发送消息
func (g *gossipServiceImpl) Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer) {
	m, err := msg.NoopSign()
	if err != nil {
		g.logger.Warningf("Failed creating SignedGossipMessage: %+v", errors.WithStack(err))
		return
	}
	g.comm.Send(m, peers...)
}

//getpeers返回endpoint-->[]discovery.networkmember的映射
func (g *gossipServiceImpl) Peers() []discovery.NetworkMember {
	return g.disc.GetMembership()
}

//peersofchannel返回被认为是活动的网络成员
//也订阅了给定的频道
func (g *gossipServiceImpl) PeersOfChannel(channel common.ChainID) []discovery.NetworkMember {
	gc := g.chanState.getGossipChannelByChainID(channel)
	if gc == nil {
		g.logger.Debug("No such channel", channel)
		return nil
	}

	return gc.GetPeers()
}

//selfmembershipinfo返回对等方的成员信息
func (g *gossipServiceImpl) SelfMembershipInfo() discovery.NetworkMember {
	return g.disc.Self()
}

//selfchannelinfo返回给定通道的对等端的最新stateinfo消息
func (g *gossipServiceImpl) SelfChannelInfo(chain common.ChainID) *proto.SignedGossipMessage {
	ch := g.chanState.getGossipChannelByChainID(chain)
	if ch == nil {
		return nil
	}
	return ch.Self()
}

//PeerFilter接收子通道SelectionCriteria并返回一个选择
//只有符合给定标准的对等身份，并且他们发布了自己的渠道参与
func (g *gossipServiceImpl) PeerFilter(channel common.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error) {
	gc := g.chanState.getGossipChannelByChainID(channel)
	if gc == nil {
		return nil, errors.Errorf("Channel %s doesn't exist", string(channel))
	}
	return gc.PeerFilter(messagePredicate), nil
}

//停止停止八卦组件
func (g *gossipServiceImpl) Stop() {
	if g.toDie() {
		return
	}
	atomic.StoreInt32(&g.stopFlag, int32(1))
	g.logger.Info("Stopping gossip")
	g.chanState.stop()
	g.discAdapter.close()
	g.disc.Stop()
	g.certStore.stop()
	g.toDieChan <- struct{}{}
	g.emitter.Stop()
	g.ChannelDeMultiplexer.Close()
	g.stateInfoMsgStore.Stop()
	g.stopSignal.Wait()
	g.comm.Stop()
}

func (g *gossipServiceImpl) UpdateMetadata(md []byte) {
	g.disc.UpdateMetadata(md)
}

//更新LedgerHeight更新Ledger Height the Peer
//发布到频道中的其他对等端
func (g *gossipServiceImpl) UpdateLedgerHeight(height uint64, chainID common.ChainID) {
	gc := g.chanState.getGossipChannelByChainID(chainID)
	if gc == nil {
		g.logger.Warning("No such channel", chainID)
		return
	}
	gc.UpdateLedgerHeight(height)
}

//更新链码更新对等发布的链码
//到渠道中的其他同行
func (g *gossipServiceImpl) UpdateChaincodes(chaincodes []*proto.Chaincode, chainID common.ChainID) {
	gc := g.chanState.getGossipChannelByChainID(chainID)
	if gc == nil {
		g.logger.Warning("No such channel", chainID)
		return
	}
	gc.UpdateChaincodes(chaincodes)
}

//accept返回与某个谓词匹配的其他节点发送的消息的专用只读通道。
//如果passthrough为false，则消息将由八卦层预先处理。
//如果passthrough是真的，那么八卦层不会介入，消息也不会
//可用于将答复发送回发件人
func (g *gossipServiceImpl) Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage) {
	if passThrough {
		return nil, g.comm.Accept(acceptor)
	}
	acceptByType := func(o interface{}) bool {
		if o, isGossipMsg := o.(*proto.GossipMessage); isGossipMsg {
			return acceptor(o)
		}
		if o, isSignedMsg := o.(*proto.SignedGossipMessage); isSignedMsg {
			sMsg := o
			return acceptor(sMsg.GossipMessage)
		}
		g.logger.Warning("Message type:", reflect.TypeOf(o), "cannot be evaluated")
		return false
	}
	inCh := g.AddChannel(acceptByType)
	outCh := make(chan *proto.GossipMessage, acceptChanSize)
	go func() {
		for {
			select {
			case s := <-g.toDieChan:
				g.toDieChan <- s
				return
			case m := <-inCh:
				if m == nil {
					return
				}
				outCh <- m.(*proto.SignedGossipMessage).GossipMessage
			}
		}
	}()
	return outCh, nil
}

func selectOnlyDiscoveryMessages(m interface{}) bool {
	msg, isGossipMsg := m.(proto.ReceivedMessage)
	if !isGossipMsg {
		return false
	}
	alive := msg.GetGossipMessage().GetAliveMsg()
	memRes := msg.GetGossipMessage().GetMemRes()
	memReq := msg.GetGossipMessage().GetMemReq()

	selected := alive != nil || memReq != nil || memRes != nil

	return selected
}

func (g *gossipServiceImpl) newDiscoveryAdapter() *discoveryAdapter {
	return &discoveryAdapter{
		c:        g.comm,
		stopping: int32(0),
		gossipFunc: func(msg *proto.SignedGossipMessage) {
			if g.conf.PropagateIterations == 0 {
				return
			}
			g.emitter.Add(&emittedGossipMessage{
				SignedGossipMessage: msg,
				filter: func(_ common.PKIidType) bool {
					return true
				},
			})
		},
		forwardFunc: func(message proto.ReceivedMessage) {
			if g.conf.PropagateIterations == 0 {
				return
			}
			g.emitter.Add(&emittedGossipMessage{
				SignedGossipMessage: message.GetGossipMessage(),
				filter:              message.GetConnectionInfo().ID.IsNotSameFilter,
			})
		},
		incChan:          make(chan proto.ReceivedMessage),
		presumedDead:     g.presumedDead,
		disclosurePolicy: g.disclosurePolicy,
	}
}

//DiscoveryAdapter用于为发现模块提供所需的功能
//发现模块中的通信接口声明
type discoveryAdapter struct {
	stopping         int32
	c                comm.Comm
	presumedDead     chan common.PKIidType
	incChan          chan proto.ReceivedMessage
	gossipFunc       func(message *proto.SignedGossipMessage)
	forwardFunc      func(message proto.ReceivedMessage)
	disclosurePolicy discovery.DisclosurePolicy
}

func (da *discoveryAdapter) close() {
	atomic.StoreInt32(&da.stopping, int32(1))
	close(da.incChan)
}

func (da *discoveryAdapter) toDie() bool {
	return atomic.LoadInt32(&da.stopping) == int32(1)
}

func (da *discoveryAdapter) Gossip(msg *proto.SignedGossipMessage) {
	if da.toDie() {
		return
	}

	da.gossipFunc(msg)
}

func (da *discoveryAdapter) Forward(msg proto.ReceivedMessage) {
	if da.toDie() {
		return
	}

	da.forwardFunc(msg)
}

func (da *discoveryAdapter) SendToPeer(peer *discovery.NetworkMember, msg *proto.SignedGossipMessage) {
	if da.toDie() {
		return
	}
//检查我们知道其PKI-ID的对等方的成员请求。
//我们不知道他们的pki ID的唯一对等体是引导程序对等体。
	if memReq := msg.GetMemReq(); memReq != nil && len(peer.PKIid) != 0 {
		selfMsg, err := memReq.SelfInformation.ToGossipMessage()
		if err != nil {
//不应该发生
			panic(errors.Wrapf(err, "Tried to send a membership request with a malformed AliveMessage"))
		}
//应用披露政策的信封筛选器
//成员请求的self-info字段的活动消息
		_, omitConcealedFields := da.disclosurePolicy(peer)
		selfMsg.Envelope = omitConcealedFields(selfMsg)
//备份旧已知字段
		oldKnown := memReq.Known
//用更新的信封覆盖新的SelfInfo消息
		memReq = &proto.MembershipRequest{
			SelfInformation: selfMsg.Envelope,
			Known:           oldKnown,
		}
		msgCopy := protoG.Clone(msg.GossipMessage).(*proto.GossipMessage)

//更新原始邮件
		msgCopy.Content = &proto.GossipMessage_MemReq{
			MemReq: memReq,
		}
//更新外部消息的信封，无需签名（点2）
		msg, err = (&proto.SignedGossipMessage{
			GossipMessage: msgCopy,
		}).NoopSign()

		if err != nil {
			return
		}
		da.c.Send(msg, &comm.RemotePeer{PKIID: peer.PKIid, Endpoint: peer.PreferredEndpoint()})
		return
	}
	da.c.Send(msg, &comm.RemotePeer{PKIID: peer.PKIid, Endpoint: peer.PreferredEndpoint()})
}

func (da *discoveryAdapter) Ping(peer *discovery.NetworkMember) bool {
	err := da.c.Probe(&comm.RemotePeer{Endpoint: peer.PreferredEndpoint(), PKIID: peer.PKIid})
	return err == nil
}

func (da *discoveryAdapter) Accept() <-chan proto.ReceivedMessage {
	return da.incChan
}

func (da *discoveryAdapter) PresumedDead() <-chan common.PKIidType {
	return da.presumedDead
}

func (da *discoveryAdapter) CloseConn(peer *discovery.NetworkMember) {
	da.c.CloseConn(&comm.RemotePeer{PKIID: peer.PKIid})
}

type discoverySecurityAdapter struct {
	identity              api.PeerIdentityType
	includeIdentityPeriod time.Time
	idMapper              identity.Mapper
	sa                    api.SecurityAdvisor
	mcs                   api.MessageCryptoService
	c                     comm.Comm
	logger                util.Logger
}

func (g *gossipServiceImpl) newDiscoverySecurityAdapter() *discoverySecurityAdapter {
	return &discoverySecurityAdapter{
		sa:                    g.secAdvisor,
		idMapper:              g.idMapper,
		mcs:                   g.mcs,
		c:                     g.comm,
		logger:                g.logger,
		includeIdentityPeriod: g.includeIdentityPeriod,
		identity:              g.selfIdentity,
	}
}

//validateAliveMsg验证活动消息是否可信
func (sa *discoverySecurityAdapter) ValidateAliveMsg(m *proto.SignedGossipMessage) bool {
	am := m.GetAliveMsg()
	if am == nil || am.Membership == nil || am.Membership.PkiId == nil || !m.IsSigned() {
		sa.logger.Warning("Invalid alive message:", m)
		return false
	}

	var identity api.PeerIdentityType

//如果alivemessage中包含身份信息
	if am.Identity != nil {
		identity = api.PeerIdentityType(am.Identity)
		claimedPKIID := am.Membership.PkiId
		err := sa.idMapper.Put(claimedPKIID, identity)
		if err != nil {
			sa.logger.Debugf("Failed validating identity of %v reason: %+v", am, errors.WithStack(err))
			return false
		}
	} else {
		identity, _ = sa.idMapper.Get(am.Membership.PkiId)
		if identity != nil {
			sa.logger.Debug("Fetched identity of", am.Membership.PkiId, "from identity store")
		}
	}

	if identity == nil {
		sa.logger.Debug("Don't have certificate for", am)
		return false
	}

	return sa.validateAliveMsgSignature(m, identity)
}

//signmessage对alivemessage进行签名并更新其签名字段
func (sa *discoverySecurityAdapter) SignMessage(m *proto.GossipMessage, internalEndpoint string) *proto.Envelope {
	signer := func(msg []byte) ([]byte, error) {
		return sa.mcs.Sign(msg)
	}
	if m.IsAliveMsg() && time.Now().Before(sa.includeIdentityPeriod) {
		m.GetAliveMsg().Identity = sa.identity
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	e, err := sMsg.Sign(signer)
	if err != nil {
		sa.logger.Warningf("Failed signing message: %+v", errors.WithStack(err))
		return nil
	}

	if internalEndpoint == "" {
		return e
	}
	e.SignSecret(signer, &proto.Secret{
		Content: &proto.Secret_InternalEndpoint{
			InternalEndpoint: internalEndpoint,
		},
	})
	return e
}

func (sa *discoverySecurityAdapter) validateAliveMsgSignature(m *proto.SignedGossipMessage, identity api.PeerIdentityType) bool {
	am := m.GetAliveMsg()
//此时，我们得到了对等机的证书，继续验证alivemessage
	verifier := func(peerIdentity []byte, signature, message []byte) error {
		return sa.mcs.Verify(api.PeerIdentityType(peerIdentity), signature, message)
	}

//我们核实了邮件上的签名
	err := m.Verify(identity, verifier)
	if err != nil {
		sa.logger.Warningf("Failed verifying: %v: %+v", am, errors.WithStack(err))
		return false
	}

	return true
}

func (g *gossipServiceImpl) createCertStorePuller() pull.Mediator {
	conf := pull.Config{
		MsgType:           proto.PullMsgType_IDENTITY_MSG,
		Channel:           []byte(""),
		ID:                g.conf.InternalEndpoint,
		PeerCountToSelect: g.conf.PullPeerNum,
		PullInterval:      g.conf.PullInterval,
		Tag:               proto.GossipMessage_EMPTY,
	}
	pkiIDFromMsg := func(msg *proto.SignedGossipMessage) string {
		identityMsg := msg.GetPeerIdentity()
		if identityMsg == nil || identityMsg.PkiId == nil {
			return ""
		}
		return fmt.Sprintf("%s", string(identityMsg.PkiId))
	}
	certConsumer := func(msg *proto.SignedGossipMessage) {
		idMsg := msg.GetPeerIdentity()
		if idMsg == nil || idMsg.Cert == nil || idMsg.PkiId == nil {
			g.logger.Warning("Invalid PeerIdentity:", idMsg)
			return
		}
		err := g.idMapper.Put(common.PKIidType(idMsg.PkiId), api.PeerIdentityType(idMsg.Cert))
		if err != nil {
			g.logger.Warningf("Failed associating PKI-ID with certificate: %+v", errors.WithStack(err))
		}
		g.logger.Debug("Learned of a new certificate:", idMsg.Cert)
	}
	adapter := &pull.PullAdapter{
		Sndr:            g.comm,
		MemSvc:          g.disc,
		IdExtractor:     pkiIDFromMsg,
		MsgCons:         certConsumer,
		EgressDigFilter: g.sameOrgOrOurOrgPullFilter,
	}
	return pull.NewPullMediator(conf, adapter)
}

func (g *gossipServiceImpl) sameOrgOrOurOrgPullFilter(msg proto.ReceivedMessage) func(string) bool {
	peersOrg := g.secAdvisor.OrgByPeerIdentity(msg.GetConnectionInfo().Identity)
	if len(peersOrg) == 0 {
		g.logger.Warning("Failed determining organization of", msg.GetConnectionInfo())
		return func(_ string) bool {
			return false
		}
	}

//如果同伴来自我们的组织，八卦所有身份
	if bytes.Equal(g.selfOrg, peersOrg) {
		return func(_ string) bool {
			return true
		}
	}
//否则，同行来自不同的组织
	return func(item string) bool {
		pkiID := common.PKIidType(item)
		msgsOrg := g.getOrgOfPeer(pkiID)
		if len(msgsOrg) == 0 {
			g.logger.Warning("Failed determining organization of", pkiID)
			return false
		}
//不要八卦死去的同龄人或同龄人的身份
//没有外部终点，对外国组织的同行。
		if !g.hasExternalEndpoint(pkiID) {
			return false
		}
//来自我们组织的对等或来自我们组织的身份或来自对等组织的身份
		return bytes.Equal(msgsOrg, g.selfOrg) || bytes.Equal(msgsOrg, peersOrg)
	}
}

func (g *gossipServiceImpl) connect2BootstrapPeers() {
	for _, endpoint := range g.conf.BootstrapPeers {
		endpoint := endpoint
		identifier := func() (*discovery.PeerIdentification, error) {
			remotePeerIdentity, err := g.comm.Handshake(&comm.RemotePeer{Endpoint: endpoint})
			if err != nil {
				return nil, errors.WithStack(err)
			}
			sameOrg := bytes.Equal(g.selfOrg, g.secAdvisor.OrgByPeerIdentity(remotePeerIdentity))
			if !sameOrg {
				return nil, errors.Errorf("%s isn't in our organization, cannot be a bootstrap peer", endpoint)
			}
			pkiID := g.mcs.GetPKIidOfCert(remotePeerIdentity)
			if len(pkiID) == 0 {
				return nil, errors.Errorf("Wasn't able to extract PKI-ID of remote peer with identity of %v", remotePeerIdentity)
			}
			return &discovery.PeerIdentification{ID: pkiID, SelfOrg: sameOrg}, nil
		}
		g.disc.Connect(discovery.NetworkMember{
			InternalEndpoint: endpoint,
			Endpoint:         endpoint,
		}, identifier)
	}

}

func (g *gossipServiceImpl) hasExternalEndpoint(PKIID common.PKIidType) bool {
	if nm := g.disc.Lookup(PKIID); nm != nil {
		return nm.Endpoint != ""
	}
	return false
}

func (g *gossipServiceImpl) isInMyorg(member discovery.NetworkMember) bool {
	if member.PKIid == nil {
		return false
	}
	if org := g.getOrgOfPeer(member.PKIid); org != nil {
		return bytes.Equal(g.selfOrg, org)
	}
	return false
}

func (g *gossipServiceImpl) getOrgOfPeer(PKIID common.PKIidType) api.OrgIdentityType {
	cert, err := g.idMapper.Get(PKIID)
	if err != nil {
		return nil
	}

	return g.secAdvisor.OrgByPeerIdentity(cert)
}

func (g *gossipServiceImpl) validateLeadershipMessage(msg *proto.SignedGossipMessage) error {
	pkiID := msg.GetLeadershipMsg().PkiId
	if len(pkiID) == 0 {
		return errors.New("Empty PKI-ID")
	}
	identity, err := g.idMapper.Get(pkiID)
	if err != nil {
		return errors.Wrap(err, "Unable to fetch PKI-ID from id-mapper")
	}
	return msg.Verify(identity, func(peerIdentity []byte, signature, message []byte) error {
		return g.mcs.Verify(identity, signature, message)
	})
}

func (g *gossipServiceImpl) validateStateInfoMsg(msg *proto.SignedGossipMessage) error {
	verifier := func(identity []byte, signature, message []byte) error {
		pkiID := g.idMapper.GetPKIidOfCert(api.PeerIdentityType(identity))
		if pkiID == nil {
			return errors.New("PKI-ID not found in identity mapper")
		}
		return g.idMapper.Verify(pkiID, signature, message)
	}
	identity, err := g.idMapper.Get(msg.GetStateInfo().PkiId)
	if err != nil {
		return errors.WithStack(err)
	}
	return msg.Verify(identity, verifier)
}

func (g *gossipServiceImpl) disclosurePolicy(remotePeer *discovery.NetworkMember) (discovery.Sieve, discovery.EnvelopeFilter) {
	remotePeerOrg := g.getOrgOfPeer(remotePeer.PKIid)

	if len(remotePeerOrg) == 0 {
		g.logger.Warning("Cannot determine organization of", remotePeer)
		return func(msg *proto.SignedGossipMessage) bool {
				return false
			}, func(msg *proto.SignedGossipMessage) *proto.Envelope {
				return msg.Envelope
			}
	}

	return func(msg *proto.SignedGossipMessage) bool {
			if !msg.IsAliveMsg() {
				g.logger.Panic("Programming error, this should be used only on alive messages")
			}
			org := g.getOrgOfPeer(msg.GetAliveMsg().Membership.PkiId)
			if len(org) == 0 {
				g.logger.Warning("Unable to determine org of message", msg.GossipMessage)
//不要传播消息谁的来源组织未知
				return false
			}

//目标组织和消息来自同一个组织
			fromSameForeignOrg := bytes.Equal(remotePeerOrg, org)
//消息来自我的组织
			fromMyOrg := bytes.Equal(g.selfOrg, org)
//仅转发来自我们组织或目标组织本身的目标组织消息。
			if !(fromSameForeignOrg || fromMyOrg) {
				return false
			}

//仅当活动消息与远程对等机在同一组织中时才传递活动消息
//或者消息有一个外部端点，远程对等端也有一个
			return bytes.Equal(org, remotePeerOrg) || msg.GetAliveMsg().Membership.Endpoint != "" && remotePeer.Endpoint != ""
		}, func(msg *proto.SignedGossipMessage) *proto.Envelope {
			envelope := protoG.Clone(msg.Envelope).(*proto.Envelope)
			if !bytes.Equal(g.selfOrg, remotePeerOrg) {
				envelope.SecretEnvelope = nil
			}
			return envelope
		}
}

func (g *gossipServiceImpl) peersByOriginOrgPolicy(peer discovery.NetworkMember) filter.RoutingFilter {
	peersOrg := g.getOrgOfPeer(peer.PKIid)
	if len(peersOrg) == 0 {
		g.logger.Warning("Unable to determine organization of peer", peer)
//不要传播消息谁的来源组织未确定
		return filter.SelectNonePolicy
	}

	if bytes.Equal(g.selfOrg, peersOrg) {
//从我们的组织向所有已知的组织传播消息。
//重要提示：当前对等机无法取消加入通道，因此唯一的方法是
//让流言蜚语停止与组织交谈的方法是让MSP
//拒绝验证来自它的消息。
		return filter.SelectAllPolicy
	}

//否则，从源站的组织中选择对等点，
//以及来自我们自己组织的同行
	return func(member discovery.NetworkMember) bool {
		memberOrg := g.getOrgOfPeer(member.PKIid)
		if len(memberOrg) == 0 {
			return false
		}
		isFromMyOrg := bytes.Equal(g.selfOrg, memberOrg)
		return isFromMyOrg || bytes.Equal(memberOrg, peersOrg)
	}
}

//分区消息接收一个谓词和一段八卦消息
//并返回由两个切片组成的元组：为谓词保留的消息
//其余的
func partitionMessages(pred common.MessageAcceptor, a []*emittedGossipMessage) ([]*emittedGossipMessage, []*emittedGossipMessage) {
	s1 := []*emittedGossipMessage{}
	s2 := []*emittedGossipMessage{}
	for _, m := range a {
		if pred(m) {
			s1 = append(s1, m)
		} else {
			s2 = append(s2, m)
		}
	}
	return s1, s2
}

//ExtractChannels返回具有所有通道的切片
//在所有给定的八卦消息中
func extractChannels(a []*emittedGossipMessage) []common.ChainID {
	channels := []common.ChainID{}
	for _, m := range a {
		if len(m.Channel) == 0 {
			continue
		}
		sameChan := func(a interface{}, b interface{}) bool {
			return bytes.Equal(a.(common.ChainID), b.(common.ChainID))
		}
		if util.IndexInSlice(channels, common.ChainID(m.Channel), sameChan) == -1 {
			channels = append(channels, common.ChainID(m.Channel))
		}
	}
	return channels
}
