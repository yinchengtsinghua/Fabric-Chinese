
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


package channel

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	common_utils "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/election"
	"github.com/hyperledger/fabric/gossip/filter"
	"github.com/hyperledger/fabric/gossip/gossip/msgstore"
	"github.com/hyperledger/fabric/gossip/gossip/pull"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
)

//配置是一个配置项
//频道商店的
type Config struct {
	ID                          string
	PublishStateInfoInterval    time.Duration
	MaxBlockCountToStore        int
	PullPeerNum                 int
	PullInterval                time.Duration
	RequestStateInfoInterval    time.Duration
	BlockExpirationInterval     time.Duration
	StateInfoCacheSweepInterval time.Duration
	TimeForMembershipTracker    time.Duration
}

//gossipchannel定义处理所有与通道相关的消息的对象
type GossipChannel interface {
//self返回有关对等机的stateInfoMessage
	Self() *proto.SignedGossipMessage

//getpeers返回由其发布元数据的对等方列表
	GetPeers() []discovery.NetworkMember

//PeerFilter接收子通道SelectionCriteria并返回一个选择
//仅匹配给定条件的对等标识
	PeerFilter(api.SubChannelSelectionCriteria) filter.RoutingFilter

//ismemberinchan检查给定成员是否有资格加入频道
	IsMemberInChan(member discovery.NetworkMember) bool

//更新LedgerHeight更新Ledger Height the Peer
//发布到频道中的其他对等端
	UpdateLedgerHeight(height uint64)

//更新链码更新对等发布的链码
//到渠道中的其他同行
	UpdateChaincodes(chaincode []*proto.Chaincode)

//isorginchannel返回给定组织是否在通道中
	IsOrgInChannel(membersOrg api.OrgIdentityType) bool

//eligibleForChannel返回给定成员是否应获取块
//对于这个频道
	EligibleForChannel(member discovery.NetworkMember) bool

//handlemessage处理远程对等机发送的消息
	HandleMessage(proto.ReceivedMessage)

//addtomsgsstore向消息存储添加给定的gossipmessage
	AddToMsgStore(msg *proto.SignedGossipMessage)

//configurechannel（re）配置组织列表
//有资格进入频道的
	ConfigureChannel(joinMsg api.JoinChannelMessage)

//LeaveChannel使对等方离开通道
	LeaveChannel()

//停止停止停止频道的活动
	Stop()
}

//适配器启用gossipchannel
//与gossipserviceimpl通信。
type Adapter interface {
	Sign(msg *proto.GossipMessage) (*proto.SignedGossipMessage, error)

//getconf返回此gossipchannel将拥有的配置
	GetConf() Config

//流言蜚语在频道里传递信息
	Gossip(message *proto.SignedGossipMessage)

//转发将消息发送到下一个跃点
	Forward(message proto.ReceivedMessage)

//多路解复用将项目多路复用到订阅服务器
	DeMultiplex(interface{})

//getmembership返回已知的活动对等点及其信息
	GetMembership() []discovery.NetworkMember

//查找返回网络成员，如果未找到，则返回nil
	Lookup(PKIID common.PKIidType) *discovery.NetworkMember

//发送向对等方列表发送消息
	Send(msg *proto.SignedGossipMessage, peers ...*comm.RemotePeer)

//如果消息
//签名不正确，否则为零。
	ValidateStateInfoMessage(message *proto.SignedGossipMessage) error

//getorgofpeer返回给定对等pki-id的组织ID
	GetOrgOfPeer(pkiID common.PKIidType) api.OrgIdentityType

//GetIdentityByPkiid返回具有特定
//pkiid，如果找不到则为nil
	GetIdentityByPKIID(pkiID common.PKIidType) api.PeerIdentityType
}

type gossipChannel struct {
	Adapter
	sync.RWMutex
	shouldGossipStateInfo     int32
	mcs                       api.MessageCryptoService
	pkiID                     common.PKIidType
	selfOrg                   api.OrgIdentityType
	stopChan                  chan struct{}
	stateInfoMsg              *proto.SignedGossipMessage
	orgs                      []api.OrgIdentityType
	joinMsg                   api.JoinChannelMessage
	blockMsgStore             msgstore.MessageStore
	stateInfoMsgStore         *stateInfoCache
	leaderMsgStore            msgstore.MessageStore
	chainID                   common.ChainID
	blocksPuller              pull.Mediator
	logger                    util.Logger
	stateInfoPublishScheduler *time.Ticker
	stateInfoRequestScheduler *time.Ticker
	memFilter                 *membershipFilter
	ledgerHeight              uint64
	incTime                   uint64
	leftChannel               int32
	membershipTracker         *membershipTracker
}

type membershipFilter struct {
	adapter Adapter
	*gossipChannel
}

//getmembership返回已知的活动对等点及其信息
func (mf *membershipFilter) GetMembership() []discovery.NetworkMember {
	if mf.hasLeftChannel() {
		return nil
	}

	var members []discovery.NetworkMember
	for _, mem := range mf.adapter.GetMembership() {
		if mf.eligibleForChannelAndSameOrg(mem) {
			members = append(members, mem)
		}
	}
	return members
}

//new gossipchannel创建新的gossipchannel
func NewGossipChannel(pkiID common.PKIidType, org api.OrgIdentityType, mcs api.MessageCryptoService,
	chainID common.ChainID, adapter Adapter, joinMsg api.JoinChannelMessage) GossipChannel {
	gc := &gossipChannel{
		incTime:                   uint64(time.Now().UnixNano()),
		selfOrg:                   org,
		pkiID:                     pkiID,
		mcs:                       mcs,
		Adapter:                   adapter,
		logger:                    util.GetLogger(util.ChannelLogger, adapter.GetConf().ID),
		stopChan:                  make(chan struct{}, 1),
		shouldGossipStateInfo:     int32(0),
		stateInfoPublishScheduler: time.NewTicker(adapter.GetConf().PublishStateInfoInterval),
		stateInfoRequestScheduler: time.NewTicker(adapter.GetConf().RequestStateInfoInterval),
		orgs:                      []api.OrgIdentityType{},
		chainID:                   chainID,
	}

	gc.memFilter = &membershipFilter{adapter: gc.Adapter, gossipChannel: gc}

	comparator := proto.NewGossipMessageComparator(adapter.GetConf().MaxBlockCountToStore)

	gc.blocksPuller = gc.createBlockPuller()

	seqNumFromMsg := func(m interface{}) string {
		return fmt.Sprintf("%d", m.(*proto.SignedGossipMessage).GetDataMsg().Payload.SeqNum)
	}
	gc.blockMsgStore = msgstore.NewMessageStoreExpirable(comparator, func(m interface{}) {
		gc.blocksPuller.Remove(seqNumFromMsg(m))
	}, gc.GetConf().BlockExpirationInterval, nil, nil, func(m interface{}) {
		gc.blocksPuller.Remove(seqNumFromMsg(m))
	})

	hashPeerExpiredInMembership := func(o interface{}) bool {
		pkiID := o.(*proto.SignedGossipMessage).GetStateInfo().PkiId
		return gc.Lookup(pkiID) == nil
	}
	verifyStateInfoMsg := func(msg *proto.SignedGossipMessage, orgs ...api.OrgIdentityType) bool {
		si := msg.GetStateInfo()
//证明自己没有意义
		if bytes.Equal(gc.pkiID, si.PkiId) {
			return true
		}
		peerIdentity := adapter.GetIdentityByPKIID(si.PkiId)
		if len(peerIdentity) == 0 {
			gc.logger.Warning("Identity for peer", si.PkiId, "doesn't exist")
			return false
		}
		isOrgInChan := func(org api.OrgIdentityType) bool {
			if len(orgs) == 0 {
				if !gc.IsOrgInChannel(org) {
					return false
				}
			} else {
				found := false
				for _, chanMember := range orgs {
					if bytes.Equal(chanMember, org) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		}

		org := gc.GetOrgOfPeer(si.PkiId)
		if !isOrgInChan(org) {
			gc.logger.Warning("peer", peerIdentity, "'s organization(", string(org), ") isn't in the channel", string(chainID))
			return false
		}
		if err := gc.mcs.VerifyByChannel(chainID, peerIdentity, msg.Signature, msg.Payload); err != nil {
			gc.logger.Warningf("Peer %v isn't eligible for channel %s : %+v", peerIdentity, string(chainID), errors.WithStack(err))
			return false
		}
		return true
	}
	gc.stateInfoMsgStore = newStateInfoCache(gc.GetConf().StateInfoCacheSweepInterval, hashPeerExpiredInMembership, verifyStateInfoMsg)

	ttl := election.GetMsgExpirationTimeout()
	pol := proto.NewGossipMessageComparator(0)

	gc.leaderMsgStore = msgstore.NewMessageStoreExpirable(pol, msgstore.Noop, ttl, nil, nil, nil)

	gc.ConfigureChannel(joinMsg)

//定期发布状态信息
	go gc.periodicalInvocation(gc.publishStateInfo, gc.stateInfoPublishScheduler.C)
//定期请求状态信息
	go gc.periodicalInvocation(gc.requestStateInfo, gc.stateInfoRequestScheduler.C)

	ticker := time.NewTicker(gc.GetConf().TimeForMembershipTracker)
	gc.membershipTracker = &membershipTracker{
		getPeersToTrack: gc.GetPeers,
		report:          gc.reportMembershipChanges,
		stopChan:        make(chan struct{}, 1),
		tickerChannel:   ticker.C,
	}

	go gc.membershipTracker.trackMembershipChanges()
	return gc
}

func (gc *gossipChannel) reportMembershipChanges(input ...interface{}) {
	gc.logger.Info(input...)
}

//停止停止频道操作
func (gc *gossipChannel) Stop() {
	gc.stopChan <- struct{}{}
	gc.membershipTracker.stopChan <- struct{}{}
	gc.blocksPuller.Stop()
	gc.stateInfoPublishScheduler.Stop()
	gc.stateInfoRequestScheduler.Stop()
	gc.leaderMsgStore.Stop()
	gc.stateInfoMsgStore.Stop()
	gc.blockMsgStore.Stop()
}

func (gc *gossipChannel) periodicalInvocation(fn func(), c <-chan time.Time) {
	for {
		select {
		case <-c:
			fn()
		case <-gc.stopChan:
			gc.stopChan <- struct{}{}
			return
		}
	}
}

//self返回有关对等机的stateInfoMessage
func (gc *gossipChannel) Self() *proto.SignedGossipMessage {
	gc.RLock()
	defer gc.RUnlock()
	return gc.stateInfoMsg
}

//LeaveChannel使对等方离开通道
func (gc *gossipChannel) LeaveChannel() {
	gc.Lock()
	defer gc.Unlock()

	atomic.StoreInt32(&gc.leftChannel, 1)

	var chaincodes []*proto.Chaincode
	var height uint64
	if prevMsg := gc.stateInfoMsg; prevMsg != nil {
		chaincodes = prevMsg.GetStateInfo().Properties.Chaincodes
		height = prevMsg.GetStateInfo().Properties.LedgerHeight
	}
	gc.updateProperties(height, chaincodes, true)
}

func (gc *gossipChannel) hasLeftChannel() bool {
	return atomic.LoadInt32(&gc.leftChannel) == 1
}

//getpeers返回由其发布元数据的对等方列表
func (gc *gossipChannel) GetPeers() []discovery.NetworkMember {
	var members []discovery.NetworkMember
	if gc.hasLeftChannel() {
		return members
	}

	for _, member := range gc.GetMembership() {
		if !gc.EligibleForChannel(member) {
			continue
		}
		stateInf := gc.stateInfoMsgStore.MsgByID(member.PKIid)
		if stateInf == nil {
			continue
		}
		props := stateInf.GetStateInfo().Properties
		if props != nil && props.LeftChannel {
			continue
		}
		member.Properties = stateInf.GetStateInfo().Properties
		member.Envelope = stateInf.Envelope
		members = append(members, member)
	}
	return members
}

func (gc *gossipChannel) requestStateInfo() {
	req, err := gc.createStateInfoRequest()
	if err != nil {
		gc.logger.Warningf("Failed creating SignedGossipMessage: %+v", errors.WithStack(err))
		return
	}
	endpoints := filter.SelectPeers(gc.GetConf().PullPeerNum, gc.GetMembership(), gc.IsMemberInChan)
	gc.Send(req, endpoints...)
}

func (gc *gossipChannel) eligibleForChannelAndSameOrg(member discovery.NetworkMember) bool {
	sameOrg := func(networkMember discovery.NetworkMember) bool {
		return bytes.Equal(gc.GetOrgOfPeer(networkMember.PKIid), gc.selfOrg)
	}
	return filter.CombineRoutingFilters(gc.EligibleForChannel, sameOrg)(member)
}

func (gc *gossipChannel) publishStateInfo() {
	if atomic.LoadInt32(&gc.shouldGossipStateInfo) == int32(0) {
		return
	}
	gc.RLock()
	stateInfoMsg := gc.stateInfoMsg
	gc.RUnlock()
	gc.Gossip(stateInfoMsg)
	if len(gc.GetMembership()) > 0 {
		atomic.StoreInt32(&gc.shouldGossipStateInfo, int32(0))
	}
}

func (gc *gossipChannel) createBlockPuller() pull.Mediator {
	conf := pull.Config{
		MsgType:           proto.PullMsgType_BLOCK_MSG,
		Channel:           []byte(gc.chainID),
		ID:                gc.GetConf().ID,
		PeerCountToSelect: gc.GetConf().PullPeerNum,
		PullInterval:      gc.GetConf().PullInterval,
		Tag:               proto.GossipMessage_CHAN_AND_ORG,
	}
	seqNumFromMsg := func(msg *proto.SignedGossipMessage) string {
		dataMsg := msg.GetDataMsg()
		if dataMsg == nil || dataMsg.Payload == nil {
			gc.logger.Warning("Non-data block or with no payload")
			return ""
		}
		return fmt.Sprintf("%d", dataMsg.Payload.SeqNum)
	}
	adapter := &pull.PullAdapter{
		Sndr:        gc,
		MemSvc:      gc.memFilter,
		IdExtractor: seqNumFromMsg,
		MsgCons: func(msg *proto.SignedGossipMessage) {
			gc.DeMultiplex(msg)
		},
	}

	adapter.IngressDigFilter = func(digestMsg *proto.DataDigest) *proto.DataDigest {
		gc.RLock()
		height := gc.ledgerHeight
		gc.RUnlock()
		digests := digestMsg.Digests
		digestMsg.Digests = nil
		for i := range digests {
			seqNum, err := strconv.ParseUint(string(digests[i]), 10, 64)
			if err != nil {
				gc.logger.Warningf("Can't parse digest %s : %+v", digests[i], errors.WithStack(err))
				continue
			}
			if seqNum >= height {
				digestMsg.Digests = append(digestMsg.Digests, digests[i])
			}

		}
		return digestMsg
	}

	return pull.NewPullMediator(conf, adapter)
}

//ismemberinchan检查给定成员是否有资格加入频道
func (gc *gossipChannel) IsMemberInChan(member discovery.NetworkMember) bool {
	org := gc.GetOrgOfPeer(member.PKIid)
	if org == nil {
		return false
	}

	return gc.IsOrgInChannel(org)
}

//PeerFilter接收子通道SelectionCriteria并返回一个选择
//仅匹配给定条件的对等标识
func (gc *gossipChannel) PeerFilter(messagePredicate api.SubChannelSelectionCriteria) filter.RoutingFilter {
	return func(member discovery.NetworkMember) bool {
		peerIdentity := gc.GetIdentityByPKIID(member.PKIid)
		if len(peerIdentity) == 0 {
			return false
		}
		msg := gc.stateInfoMsgStore.MembershipStore.MsgByID(member.PKIid)
		if msg == nil {
			return false
		}

		return messagePredicate(api.PeerSignature{
			Message:      msg.Payload,
			Signature:    msg.Signature,
			PeerIdentity: peerIdentity,
		})
	}
}

//isorginchannel返回给定组织是否在通道中
func (gc *gossipChannel) IsOrgInChannel(membersOrg api.OrgIdentityType) bool {
	gc.RLock()
	defer gc.RUnlock()
	for _, orgOfChan := range gc.orgs {
		if bytes.Equal(orgOfChan, membersOrg) {
			return true
		}
	}
	return false
}

//eligibleForChannel返回给定成员是否应获取块
//对于这个频道
func (gc *gossipChannel) EligibleForChannel(member discovery.NetworkMember) bool {
	peerIdentity := gc.GetIdentityByPKIID(member.PKIid)
	if len(peerIdentity) == 0 {
		gc.logger.Warning("Identity for peer", member.PKIid, "doesn't exist")
		return false
	}
	msg := gc.stateInfoMsgStore.MsgByID(member.PKIid)
	if msg == nil {
		return false
	}
	return true
}

//addtomsgsstore向消息存储添加给定的gossipmessage
func (gc *gossipChannel) AddToMsgStore(msg *proto.SignedGossipMessage) {
	if msg.IsDataMsg() {
		gc.blockMsgStore.Add(msg)
		gc.blocksPuller.Add(msg)
	}

	if msg.IsStateInfoMsg() {
		gc.stateInfoMsgStore.Add(msg)
	}
}

//configurechannel（re）配置组织列表
//有资格进入频道的
func (gc *gossipChannel) ConfigureChannel(joinMsg api.JoinChannelMessage) {
	gc.Lock()
	defer gc.Unlock()

	if len(joinMsg.Members()) == 0 {
		gc.logger.Warning("Received join channel message with empty set of members")
		return
	}

	if gc.joinMsg == nil {
		gc.joinMsg = joinMsg
	}

	if gc.joinMsg.SequenceNumber() > (joinMsg.SequenceNumber()) {
		gc.logger.Warning("Already have a more updated JoinChannel message(", gc.joinMsg.SequenceNumber(), ") than", joinMsg.SequenceNumber())
		return
	}

	gc.orgs = joinMsg.Members()
	gc.joinMsg = joinMsg
	gc.stateInfoMsgStore.validate(joinMsg.Members())
}

//handlemessage处理远程对等机发送的消息
func (gc *gossipChannel) HandleMessage(msg proto.ReceivedMessage) {
	if !gc.verifyMsg(msg) {
		gc.logger.Warning("Failed verifying message:", msg.GetGossipMessage().GossipMessage)
		return
	}
	m := msg.GetGossipMessage()
	if !m.IsChannelRestricted() {
		gc.logger.Warning("Got message", msg.GetGossipMessage(), "but it's not a per-channel message, discarding it")
		return
	}
	orgID := gc.GetOrgOfPeer(msg.GetConnectionInfo().ID)
	if len(orgID) == 0 {
		gc.logger.Debug("Couldn't find org identity of peer", msg.GetConnectionInfo())
		return
	}
	if !gc.IsOrgInChannel(orgID) {
		gc.logger.Warning("Point to point message came from", msg.GetConnectionInfo(),
			", org(", string(orgID), ") but it's not eligible for the channel", string(gc.chainID))
		return
	}

	if m.IsStateInfoPullRequestMsg() {
		msg.Respond(gc.createStateInfoSnapshot(orgID))
		return
	}

	if m.IsStateInfoSnapshot() {
		gc.handleStateInfSnapshot(m.GossipMessage, msg.GetConnectionInfo().ID)
		return
	}

	if m.IsDataMsg() || m.IsStateInfoMsg() {
		added := false

		if m.IsDataMsg() {
			if m.GetDataMsg().Payload == nil {
				gc.logger.Warning("Payload is empty, got it from", msg.GetConnectionInfo().ID)
				return
			}
//如果验证了这个块，它会进入消息存储吗？
			if !gc.blockMsgStore.CheckValid(msg.GetGossipMessage()) {
				return
			}
			if !gc.verifyBlock(m.GossipMessage, msg.GetConnectionInfo().ID) {
				gc.logger.Warning("Failed verifying block", m.GetDataMsg().Payload.SeqNum)
				return
			}
			added = gc.blockMsgStore.Add(msg.GetGossipMessage())
} else { //stateinfomsg验证应在上面的层中处理
//因为我们无法访问这里的ID映射器
			added = gc.stateInfoMsgStore.Add(msg.GetGossipMessage())
		}

		if added {
//转发消息
			gc.Forward(msg)
//解复用到本地用户
			gc.DeMultiplex(m)

			if m.IsDataMsg() {
				gc.blocksPuller.Add(msg.GetGossipMessage())
			}
		}
		return
	}

	if m.IsPullMsg() && m.GetPullMsgType() == proto.PullMsgType_BLOCK_MSG {
		if gc.hasLeftChannel() {
			gc.logger.Info("Received Pull message from", msg.GetConnectionInfo().Endpoint, "but left the channel", string(gc.chainID))
			return
		}
//如果我们没有来自对等端的stateinfo消息，
//无法在频道中验证其资格。
		if gc.stateInfoMsgStore.MsgByID(msg.GetConnectionInfo().ID) == nil {
			gc.logger.Debug("Don't have StateInfo message of peer", msg.GetConnectionInfo())
			return
		}
		if !gc.eligibleForChannelAndSameOrg(discovery.NetworkMember{PKIid: msg.GetConnectionInfo().ID}) {
			gc.logger.Warning(msg.GetConnectionInfo(), "isn't eligible for pulling blocks of", string(gc.chainID))
			return
		}
		if m.IsDataUpdate() {
//迭代信封，并筛选出块
//我们已经在BlockMsgstore中拥有的，或是在BlockMsgstore中
//在过去太远了。
			filteredEnvelopes := []*proto.Envelope{}
			for _, item := range m.GetDataUpdate().Data {
				gMsg, err := item.ToGossipMessage()
				if err != nil {
					gc.logger.Warningf("Data update contains an invalid message: %+v", errors.WithStack(err))
					return
				}
				if !bytes.Equal(gMsg.Channel, []byte(gc.chainID)) {
					gc.logger.Warning("DataUpdate message contains item with channel", gMsg.Channel, "but should be", gc.chainID)
					return
				}
//如果验证了这个块，它会进入消息存储吗？
				if !gc.blockMsgStore.CheckValid(msg.GetGossipMessage()) {
					return
				}
				if !gc.verifyBlock(gMsg.GossipMessage, msg.GetConnectionInfo().ID) {
					return
				}
				added := gc.blockMsgStore.Add(gMsg)
				if !added {
//如果不需要添加此块，则表示它已经
//存在于记忆中或过去太远
					continue
				}
				filteredEnvelopes = append(filteredEnvelopes, item)
			}
//仅用应处理的块替换更新消息
			m.GetDataUpdate().Data = filteredEnvelopes
		}
		gc.blocksPuller.HandleMessage(msg)
	}

	if m.IsLeadershipMsg() {
//处理领导信息
		added := gc.leaderMsgStore.Add(m)
		if added {
			gc.DeMultiplex(m)
		}
	}
}

func (gc *gossipChannel) handleStateInfSnapshot(m *proto.GossipMessage, sender common.PKIidType) {
	chanName := string(gc.chainID)
	for _, envelope := range m.GetStateSnapshot().Elements {
		stateInf, err := envelope.ToGossipMessage()
		if err != nil {
			gc.logger.Warningf("Channel %s : StateInfo snapshot contains an invalid message: %+v", chanName, errors.WithStack(err))
			return
		}
		if !stateInf.IsStateInfoMsg() {
			gc.logger.Warning("Channel", chanName, ": Element of StateInfoSnapshot isn't a StateInfoMessage:",
				stateInf, "message sent from", sender)
			return
		}
		si := stateInf.GetStateInfo()
		orgID := gc.GetOrgOfPeer(si.PkiId)
		if orgID == nil {
			gc.logger.Debug("Channel", chanName, ": Couldn't find org identity of peer",
				string(si.PkiId), "message sent from", string(sender))
			return
		}

		if !gc.IsOrgInChannel(orgID) {
			gc.logger.Warning("Channel", chanName, ": Peer", stateInf.GetStateInfo().PkiId,
				"is not in an eligible org, can't process a stateInfo from it, sent from", sender)
			return
		}

		expectedMAC := GenerateMAC(si.PkiId, gc.chainID)
		if !bytes.Equal(si.Channel_MAC, expectedMAC) {
			gc.logger.Warning("Channel", chanName, ": StateInfo message", stateInf,
				", has an invalid MAC. Expected", expectedMAC, ", got", si.Channel_MAC, ", sent from", sender)
			return
		}
		err = gc.ValidateStateInfoMessage(stateInf)
		if err != nil {
			gc.logger.Warningf("Channel %s: Failed validating state info message: %v sent from %v : %+v", chanName, stateInf, sender, errors.WithStack(err))
			return
		}

		if gc.Lookup(si.PkiId) == nil {
//跳过属于对等方的StateInfo消息
//已经过期了
			continue
		}

		gc.stateInfoMsgStore.Add(stateInf)
	}
}

func (gc *gossipChannel) verifyBlock(msg *proto.GossipMessage, sender common.PKIidType) bool {
	if !msg.IsDataMsg() {
		gc.logger.Warning("Received from ", sender, "a DataUpdate message that contains a non-block GossipMessage:", msg)
		return false
	}
	payload := msg.GetDataMsg().Payload
	if payload == nil {
		gc.logger.Warning("Received empty payload from", sender)
		return false
	}
	seqNum := payload.SeqNum
	rawBlock := payload.Data
	err := gc.mcs.VerifyBlock(msg.Channel, seqNum, rawBlock)
	if err != nil {
		gc.logger.Warningf("Received fabricated block from %v in DataUpdate: %+v", sender, errors.WithStack(err))
		return false
	}
	return true
}

func (gc *gossipChannel) createStateInfoSnapshot(requestersOrg api.OrgIdentityType) *proto.GossipMessage {
	sameOrg := bytes.Equal(gc.selfOrg, requestersOrg)
	rawElements := gc.stateInfoMsgStore.Get()
	elements := []*proto.Envelope{}
	for _, rawEl := range rawElements {
		msg := rawEl.(*proto.SignedGossipMessage)
		orgOfCurrentMsg := gc.GetOrgOfPeer(msg.GetStateInfo().PkiId)
//如果我们和请求者在同一个组织，或者消息属于一个外国组织
//不进行任何筛选
		if sameOrg || !bytes.Equal(orgOfCurrentMsg, gc.selfOrg) {
			elements = append(elements, msg.Envelope)
			continue
		}
//否则，请求者在不同的组织中，因此只公开stateinfo消息，
//相应的AliveMessage具有外部终结点
		if netMember := gc.Lookup(msg.GetStateInfo().PkiId); netMember == nil || netMember.Endpoint == "" {
			continue
		}
		elements = append(elements, msg.Envelope)
	}

	return &proto.GossipMessage{
		Channel: gc.chainID,
		Tag:     proto.GossipMessage_CHAN_OR_ORG,
		Nonce:   0,
		Content: &proto.GossipMessage_StateSnapshot{
			StateSnapshot: &proto.StateInfoSnapshot{
				Elements: elements,
			},
		},
	}
}

func (gc *gossipChannel) verifyMsg(msg proto.ReceivedMessage) bool {
	if msg == nil {
		gc.logger.Warning("Messsage is nil")
		return false
	}
	m := msg.GetGossipMessage()
	if m == nil {
		gc.logger.Warning("Message content is empty")
		return false
	}

	if msg.GetConnectionInfo().ID == nil {
		gc.logger.Warning("Message has nil PKI-ID")
		return false
	}

	if m.IsStateInfoMsg() {
		si := m.GetStateInfo()
		expectedMAC := GenerateMAC(si.PkiId, gc.chainID)
		if !bytes.Equal(expectedMAC, si.Channel_MAC) {
			gc.logger.Warning("Message contains wrong channel MAC(", si.Channel_MAC, "), expected", expectedMAC)
			return false
		}
		return true
	}

	if m.IsStateInfoPullRequestMsg() {
		sipr := m.GetStateInfoPullReq()
		expectedMAC := GenerateMAC(msg.GetConnectionInfo().ID, gc.chainID)
		if !bytes.Equal(expectedMAC, sipr.Channel_MAC) {
			gc.logger.Warning("Message contains wrong channel MAC(", sipr.Channel_MAC, "), expected", expectedMAC)
			return false
		}
		return true
	}

	if !bytes.Equal(m.Channel, []byte(gc.chainID)) {
		gc.logger.Warning("Message contains wrong channel(", m.Channel, "), expected", gc.chainID)
		return false
	}
	return true
}

func (gc *gossipChannel) createStateInfoRequest() (*proto.SignedGossipMessage, error) {
	return (&proto.GossipMessage{
		Tag:   proto.GossipMessage_CHAN_OR_ORG,
		Nonce: 0,
		Content: &proto.GossipMessage_StateInfoPullReq{
			StateInfoPullReq: &proto.StateInfoPullRequest{
				Channel_MAC: GenerateMAC(gc.pkiID, gc.chainID),
			},
		},
	}).NoopSign()
}

//更新LedgerHeight更新Ledger Height the Peer
//发布到频道中的其他对等端
func (gc *gossipChannel) UpdateLedgerHeight(height uint64) {
	gc.Lock()
	defer gc.Unlock()

	var chaincodes []*proto.Chaincode
	var leftChannel bool
	if prevMsg := gc.stateInfoMsg; prevMsg != nil {
		leftChannel = prevMsg.GetStateInfo().Properties.LeftChannel
		chaincodes = prevMsg.GetStateInfo().Properties.Chaincodes
	}
	gc.updateProperties(height, chaincodes, leftChannel)
}

//更新链码更新对等发布的链码
//到渠道中的其他同行
func (gc *gossipChannel) UpdateChaincodes(chaincodes []*proto.Chaincode) {
	gc.Lock()
	defer gc.Unlock()

	var ledgerHeight uint64 = 1
	var leftChannel bool
	if prevMsg := gc.stateInfoMsg; prevMsg != nil {
		ledgerHeight = prevMsg.GetStateInfo().Properties.LedgerHeight
		leftChannel = prevMsg.GetStateInfo().Properties.LeftChannel
	}
	gc.updateProperties(ledgerHeight, chaincodes, leftChannel)
}

//UpdateStateInfo更新此频道的StateInfo消息
//定期出版的
func (gc *gossipChannel) updateStateInfo(msg *proto.SignedGossipMessage) {
	gc.stateInfoMsgStore.Add(msg)
	gc.ledgerHeight = msg.GetStateInfo().Properties.LedgerHeight
	gc.stateInfoMsg = msg
	atomic.StoreInt32(&gc.shouldGossipStateInfo, int32(1))
}

func (gc *gossipChannel) updateProperties(ledgerHeight uint64, chaincodes []*proto.Chaincode, leftChannel bool) {
	stateInfMsg := &proto.StateInfo{
		Channel_MAC: GenerateMAC(gc.pkiID, gc.chainID),
		PkiId:       gc.pkiID,
		Timestamp: &proto.PeerTime{
			IncNum: gc.incTime,
			SeqNum: uint64(time.Now().UnixNano()),
		},
		Properties: &proto.Properties{
			LeftChannel:  leftChannel,
			LedgerHeight: ledgerHeight,
			Chaincodes:   chaincodes,
		},
	}
	m := &proto.GossipMessage{
		Nonce: 0,
		Tag:   proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_StateInfo{
			StateInfo: stateInfMsg,
		},
	}

	msg, err := gc.Sign(m)
	if err != nil {
		gc.logger.Error("Failed signing message:", err)
		return
	}
	gc.updateStateInfo(msg)
}

func newStateInfoCache(sweepInterval time.Duration, hasExpired func(interface{}) bool, verifyFunc membershipPredicate) *stateInfoCache {
	membershipStore := util.NewMembershipStore()
	pol := proto.NewGossipMessageComparator(0)

	s := &stateInfoCache{
		verify:          verifyFunc,
		MembershipStore: membershipStore,
		stopChan:        make(chan struct{}),
	}
	invalidationTrigger := func(m interface{}) {
		pkiID := m.(*proto.SignedGossipMessage).GetStateInfo().PkiId
		membershipStore.Remove(pkiID)
	}
	s.MessageStore = msgstore.NewMessageStore(pol, invalidationTrigger)

	go func() {
		for {
			select {
			case <-s.stopChan:
				return
			case <-time.After(sweepInterval):
				s.Purge(hasExpired)
			}
		}
	}()
	return s
}

//membershipPredicate接收StateInfoMessage和可选的组织标识符切片
//并返回签名给定StateInfoMessage的对等方是否符合条件
//不到频道
type membershipPredicate func(msg *proto.SignedGossipMessage, orgs ...api.OrgIdentityType) bool

//StateInfoCache实际上是一个消息存储
//它还索引添加的消息
//以便以后提取
type stateInfoCache struct {
	verify membershipPredicate
	*util.MembershipStore
	msgstore.MessageStore
	stopChan chan struct{}
}

func (cache *stateInfoCache) validate(orgs []api.OrgIdentityType) {
	for _, m := range cache.Get() {
		msg := m.(*proto.SignedGossipMessage)
		if !cache.verify(msg, orgs...) {
			cache.delete(msg)
		}
	}
}

//添加将给定消息添加到StateInfoCache的尝试，
//如果添加了消息，也会对其进行索引。
//消息必须是StateInfo消息。
func (cache *stateInfoCache) Add(msg *proto.SignedGossipMessage) bool {
	if !cache.MessageStore.CheckValid(msg) {
		return false
	}
	if !cache.verify(msg) {
		return false
	}
	added := cache.MessageStore.Add(msg)
	if added {
		pkiID := msg.GetStateInfo().PkiId
		cache.MembershipStore.Put(pkiID, msg)
	}
	return added
}

func (cache *stateInfoCache) delete(msg *proto.SignedGossipMessage) {
	cache.Purge(func(o interface{}) bool {
		pkiID := o.(*proto.SignedGossipMessage).GetStateInfo().PkiId
		return bytes.Equal(pkiID, msg.GetStateInfo().PkiId)
	})
	cache.Remove(msg.GetStateInfo().PkiId)
}

func (cache *stateInfoCache) Stop() {
	cache.stopChan <- struct{}{}
}

//generatemac返回从对等机的pki-id派生的字节片
//还有一个频道名
func GenerateMAC(pkiID common.PKIidType, channelID common.ChainID) []byte {
//哈希计算基于（pki-id通道id）
	preImage := append([]byte(pkiID), []byte(channelID)...)
	return common_utils.ComputeSHA256(preImage)
}

//MembershipTracker是用于跟踪通道对等端更改的结构。
type membershipTracker struct {
	getPeersToTrack func() []discovery.NetworkMember
	report          func(...interface{})
	stopChan        chan struct{}
	tickerChannel   <-chan time.Time
}

//终结点按终结点返回所有对等点
func endpoints(members discovery.Members) [][]string {
	var currView [][]string
	for _, member := range members {
		ep := member.Endpoint
		epi := member.InternalEndpoint
		var endPoints []string
		if ep != epi {
			endPoints = append(endPoints, ep, epi)
		} else {
			endPoints = append(endPoints, ep)
		}
		currView = append(currView, endPoints)
	}
	return currView
}

//checkifpeershaged检查哪些对等机处于脱机状态，哪些对等机处于通道的联机状态
func (mt *membershipTracker) checkIfPeersChanged(prevPeers discovery.Members, currPeers discovery.Members,
	prevSetPeers map[string]struct{}, currSetPeers map[string]struct{}) {
	var currView [][]string

	wereInPrev := endpoints(prevPeers.Filter(func(member discovery.NetworkMember) bool {
		_, exists := currSetPeers[string(member.PKIid)]
		return !exists
	}))
	newInCurr := endpoints(currPeers.Filter(func(member discovery.NetworkMember) bool {
		_, exists := prevSetPeers[string(member.PKIid)]
		return !exists
	}))
	currView = endpoints(currPeers)

	if !reflect.DeepEqual(wereInPrev, newInCurr) {
		if len(wereInPrev) == 0 {
			mt.report("Membership view has changed. peers went online: ", newInCurr, ", current view: ", currView)
		} else if len(newInCurr) == 0 {
			mt.report("Membership view has changed. peers went offline: ", wereInPrev, ", current view: ", currView)
		} else {
			mt.report("Membership view has changed. peers went offline: ", wereInPrev, ", peers went online: ", newInCurr, ", current view: ", currView)
		}
	}
}

func (mt *membershipTracker) createSetOfPeers(peersToMakeSet []discovery.NetworkMember) map[string]struct{} {
	setPeers := make(map[string]struct{})
	for _, prevPeer := range peersToMakeSet {
		prevPeerID := string(prevPeer.PKIid)
		setPeers[prevPeerID] = struct{}{}
	}
	return setPeers
}

func (mt *membershipTracker) trackMembershipChanges() {
	prevSetPeers := make(map[string]struct{})
	prev := mt.getPeersToTrack()
	prevSetPeers = mt.createSetOfPeers(prev)
	for {
		currSetPeers := make(map[string]struct{})
//检查对等机中的更改超时
		select {
		case <-mt.stopChan:
			return
		case <-mt.tickerChannel:
//获取当前对等方
			currPeers := mt.getPeersToTrack()
			currSetPeers = mt.createSetOfPeers(currPeers)
			mt.checkIfPeersChanged(prev, currPeers, prevSetPeers, currSetPeers)
			prev = currPeers
			prevSetPeers = map[string]struct{}{}
			prevSetPeers = mt.createSetOfPeers(prev)
		}
	}
}
