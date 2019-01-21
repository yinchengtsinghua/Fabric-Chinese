
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


package pull

import (
	"sync"
	"time"

	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip/algo"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

//常量放在这里。
const (
	HelloMsgType MsgType = iota
	DigestMsgType
	RequestMsgType
	ResponseMsgType
)

//msgtype定义发送到pullstore的消息的类型
type MsgType int

//messagehook定义一个函数，该函数将在收到某个pull消息后运行
type MessageHook func(itemIDs []string, items []*proto.SignedGossipMessage, msg proto.ReceivedMessage)

//发送方向远程对等方发送消息
type Sender interface {
//发送向远程对等方列表发送消息
	Send(msg *proto.SignedGossipMessage, peers ...*comm.RemotePeer)
}

//成员身份服务获取活动对等方的成员身份信息
type MembershipService interface {
//GetMembership返回的成员身份
	GetMembership() []discovery.NetworkMember
}

//config定义pull中介的配置
type Config struct {
	ID                string
PullInterval      time.Duration //请求调用之间的持续时间
	Channel           common.ChainID
PeerCountToSelect int //开始拉的对等数
	Tag               proto.GossipMessage_Tag
	MsgType           proto.PullMsgType
}

//IngressDigestFilter筛选出从远程对等方接收的摘要中的实体
type IngressDigestFilter func(digestMsg *proto.DataDigest) *proto.DataDigest

//要发送到远程对等机的egersdigestfilter筛选器摘要，
//向您发送了以下信息
type EgressDigestFilter func(helloMsg proto.ReceivedMessage) func(digestItem string) bool

//ByContext将此EgressDigFilter转换为algo.DigestFilter
func (df EgressDigestFilter) byContext() algo.DigestFilter {
	return func(context interface{}) func(digestItem string) bool {
		return func(digestItem string) bool {
			return df(context.(proto.ReceivedMessage))(digestItem)
		}
	}
}

//pulladapter定义要交互的pullstore方法
//各种各样的八卦模块
type PullAdapter struct {
	Sndr             Sender
	MemSvc           MembershipService
	IdExtractor      proto.IdentifierExtractor
	MsgCons          proto.MsgConsumer
	EgressDigFilter  EgressDigestFilter
	IngressDigFilter IngressDigestFilter
}

//中介器是一个组件，它包装一个pullengine并提供方法
//它需要执行拉同步。
//拉中介器对特定类型消息的专门化是
//通过配置完成，IdentifierExtractor、IdentifierExtractor
//在施工时提供，也可以为每个挂钩注册
//pullmsgtype的类型（hello、digest、req、res）。
type Mediator interface {
//停止调解人
	Stop()

//registermsghook将消息挂钩注册到特定类型的拉消息
	RegisterMsgHook(MsgType, MessageHook)

//添加向中介添加八卦消息
	Add(*proto.SignedGossipMessage)

//移除从具有匹配摘要的中介移除八卦消息，
//如果这样的信息退出
	Remove(digest string)

//handleMessage处理来自某个远程对等机的消息
	HandleMessage(msg proto.ReceivedMessage)
}

//pullmediateimpl是中介的一个实现
type pullMediatorImpl struct {
	sync.RWMutex
	*PullAdapter
	msgType2Hook map[MsgType][]MessageHook
	config       Config
	logger       util.Logger
	itemID2Msg   map[string]*proto.SignedGossipMessage
	engine       *algo.PullEngine
}

//newpullmediator返回新的中介
func NewPullMediator(config Config, adapter *PullAdapter) Mediator {
	egressDigFilter := adapter.EgressDigFilter

	acceptAllFilter := func(_ proto.ReceivedMessage) func(string) bool {
		return func(_ string) bool {
			return true
		}
	}

	if egressDigFilter == nil {
		egressDigFilter = acceptAllFilter
	}

	p := &pullMediatorImpl{
		PullAdapter:  adapter,
		msgType2Hook: make(map[MsgType][]MessageHook),
		config:       config,
		logger:       util.GetLogger(util.PullLogger, config.ID),
		itemID2Msg:   make(map[string]*proto.SignedGossipMessage),
	}

	p.engine = algo.NewPullEngineWithFilter(p, config.PullInterval, egressDigFilter.byContext())

	if adapter.IngressDigFilter == nil {
//创建接受所有筛选器
		adapter.IngressDigFilter = func(digestMsg *proto.DataDigest) *proto.DataDigest {
			return digestMsg
		}
	}
	return p

}

func (p *pullMediatorImpl) HandleMessage(m proto.ReceivedMessage) {
	if m.GetGossipMessage() == nil || !m.GetGossipMessage().IsPullMsg() {
		return
	}
	msg := m.GetGossipMessage()
	msgType := msg.GetPullMsgType()
	if msgType != p.config.MsgType {
		return
	}

	p.logger.Debug(msg)

	itemIDs := []string{}
	items := []*proto.SignedGossipMessage{}
	var pullMsgType MsgType

	if helloMsg := msg.GetHello(); helloMsg != nil {
		pullMsgType = HelloMsgType
		p.engine.OnHello(helloMsg.Nonce, m)
	}
	if digest := msg.GetDataDig(); digest != nil {
		d := p.PullAdapter.IngressDigFilter(digest)
		itemIDs = util.BytesToStrings(d.Digests)
		pullMsgType = DigestMsgType
		p.engine.OnDigest(itemIDs, d.Nonce, m)
	}
	if req := msg.GetDataReq(); req != nil {
		itemIDs = util.BytesToStrings(req.Digests)
		pullMsgType = RequestMsgType
		p.engine.OnReq(itemIDs, req.Nonce, m)
	}
	if res := msg.GetDataUpdate(); res != nil {
		itemIDs = make([]string, len(res.Data))
		items = make([]*proto.SignedGossipMessage, len(res.Data))
		pullMsgType = ResponseMsgType
		for i, pulledMsg := range res.Data {
			msg, err := pulledMsg.ToGossipMessage()
			if err != nil {
				p.logger.Warningf("Data update contains an invalid message: %+v", errors.WithStack(err))
				return
			}
			p.MsgCons(msg)
			itemIDs[i] = p.IdExtractor(msg)
			items[i] = msg
			p.Lock()
			p.itemID2Msg[itemIDs[i]] = msg
			p.Unlock()
		}
		p.engine.OnRes(itemIDs, res.Nonce)
	}

//调用相关消息类型的挂钩
	for _, h := range p.hooksByMsgType(pullMsgType) {
		h(itemIDs, items, m)
	}
}

func (p *pullMediatorImpl) Stop() {
	p.engine.Stop()
}

//registermsghook将消息挂钩注册到特定类型的拉消息
func (p *pullMediatorImpl) RegisterMsgHook(pullMsgType MsgType, hook MessageHook) {
	p.Lock()
	defer p.Unlock()
	p.msgType2Hook[pullMsgType] = append(p.msgType2Hook[pullMsgType], hook)

}

//添加将八卦消息添加到存储
func (p *pullMediatorImpl) Add(msg *proto.SignedGossipMessage) {
	p.Lock()
	defer p.Unlock()
	itemID := p.IdExtractor(msg)
	p.itemID2Msg[itemID] = msg
	p.engine.Add(itemID)
}

//移除从具有匹配摘要的中介移除八卦消息，
//如果这样的信息退出
func (p *pullMediatorImpl) Remove(digest string) {
	p.Lock()
	defer p.Unlock()
	delete(p.itemID2Msg, digest)
	p.engine.Remove(digest)
}

//selectpeers返回一个对等机切片，引擎将使用该切片启动协议
func (p *pullMediatorImpl) SelectPeers() []string {
	remotePeers := SelectEndpoints(p.config.PeerCountToSelect, p.MemSvc.GetMembership())
	endpoints := make([]string, len(remotePeers))
	for i, peer := range remotePeers {
		endpoints[i] = peer.Endpoint
	}
	return endpoints
}

//hello发送hello消息以启动协议
//并返回一个期望返回的nonce
//在摘要消息中。
func (p *pullMediatorImpl) Hello(dest string, nonce uint64) {
	helloMsg := &proto.GossipMessage{
		Channel: p.config.Channel,
		Tag:     p.config.Tag,
		Content: &proto.GossipMessage_Hello{
			Hello: &proto.GossipHello{
				Nonce:    nonce,
				Metadata: nil,
				MsgType:  p.config.MsgType,
			},
		},
	}

	p.logger.Debug("Sending", p.config.MsgType, "hello to", dest)
	sMsg, err := helloMsg.NoopSign()
	if err != nil {
		p.logger.Errorf("Failed creating SignedGossipMessage: %+v", errors.WithStack(err))
		return
	}
	p.Sndr.Send(sMsg, p.peersWithEndpoints(dest)...)
}

//sendDigest将摘要发送到远程pullengine。
//上下文参数指定要发送到的远程引擎。
func (p *pullMediatorImpl) SendDigest(digest []string, nonce uint64, context interface{}) {
	digMsg := &proto.GossipMessage{
		Channel: p.config.Channel,
		Tag:     p.config.Tag,
		Nonce:   0,
		Content: &proto.GossipMessage_DataDig{
			DataDig: &proto.DataDigest{
				MsgType: p.config.MsgType,
				Nonce:   nonce,
				Digests: util.StringsToBytes(digest),
			},
		},
	}
	remotePeer := context.(proto.ReceivedMessage).GetConnectionInfo()
	if p.logger.IsEnabledFor(zapcore.DebugLevel) {
		p.logger.Debug("Sending", p.config.MsgType, "digest:", digMsg.GetDataDig().FormattedDigests(), "to", remotePeer)
	}

	context.(proto.ReceivedMessage).Respond(digMsg)
}

//sendreq将一组项目发送到某个已标识的远程拉器引擎
//用绳子
func (p *pullMediatorImpl) SendReq(dest string, items []string, nonce uint64) {
	req := &proto.GossipMessage{
		Channel: p.config.Channel,
		Tag:     p.config.Tag,
		Nonce:   0,
		Content: &proto.GossipMessage_DataReq{
			DataReq: &proto.DataRequest{
				MsgType: p.config.MsgType,
				Nonce:   nonce,
				Digests: util.StringsToBytes(items),
			},
		},
	}
	if p.logger.IsEnabledFor(zapcore.DebugLevel) {
		p.logger.Debug("Sending", req.GetDataReq().FormattedDigests(), "to", dest)
	}
	sMsg, err := req.NoopSign()
	if err != nil {
		p.logger.Warningf("Failed creating SignedGossipMessage: %+v", errors.WithStack(err))
		return
	}
	p.Sndr.Send(sMsg, p.peersWithEndpoints(dest)...)
}

//sendres将一组项目发送到由上下文标识的远程pullengine。
func (p *pullMediatorImpl) SendRes(items []string, context interface{}, nonce uint64) {
	items2return := []*proto.Envelope{}
	p.RLock()
	defer p.RUnlock()
	for _, item := range items {
		if msg, exists := p.itemID2Msg[item]; exists {
			items2return = append(items2return, msg.Envelope)
		}
	}
	returnedUpdate := &proto.GossipMessage{
		Channel: p.config.Channel,
		Tag:     p.config.Tag,
		Nonce:   0,
		Content: &proto.GossipMessage_DataUpdate{
			DataUpdate: &proto.DataUpdate{
				MsgType: p.config.MsgType,
				Nonce:   nonce,
				Data:    items2return,
			},
		},
	}
	remotePeer := context.(proto.ReceivedMessage).GetConnectionInfo()
	p.logger.Debug("Sending", len(returnedUpdate.GetDataUpdate().Data), p.config.MsgType, "items to", remotePeer)
	context.(proto.ReceivedMessage).Respond(returnedUpdate)
}

func (p *pullMediatorImpl) peersWithEndpoints(endpoints ...string) []*comm.RemotePeer {
	peers := []*comm.RemotePeer{}
	for _, member := range p.MemSvc.GetMembership() {
		for _, endpoint := range endpoints {
			if member.PreferredEndpoint() == endpoint {
				peers = append(peers, &comm.RemotePeer{Endpoint: member.PreferredEndpoint(), PKIID: member.PKIid})
			}
		}
	}
	return peers
}

func (p *pullMediatorImpl) hooksByMsgType(msgType MsgType) []MessageHook {
	p.RLock()
	defer p.RUnlock()
	returnedHooks := []MessageHook{}
	for _, h := range p.msgType2Hook[msgType] {
		returnedHooks = append(returnedHooks, h)
	}
	return returnedHooks
}

//selectendpoints从对等池中选择k个对等点并返回它们。
func SelectEndpoints(k int, peerPool []discovery.NetworkMember) []*comm.RemotePeer {
	if len(peerPool) < k {
		k = len(peerPool)
	}

	indices := util.GetRandomIndices(k, len(peerPool)-1)
	endpoints := make([]*comm.RemotePeer, len(indices))
	for i, j := range indices {
		endpoints[i] = &comm.RemotePeer{Endpoint: peerPool[j].PreferredEndpoint(), PKIID: peerPool[j].PKIid}
	}
	return endpoints
}
