
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
	"fmt"
	"time"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

//八卦是八卦组件的接口
type Gossip interface {

//selfmembershipinfo返回对等方的成员信息
	SelfMembershipInfo() discovery.NetworkMember

//selfchannelinfo返回给定通道的对等端的最新stateinfo消息
	SelfChannelInfo(common.ChainID) *proto.SignedGossipMessage

//发送向远程对等发送消息
	Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer)

//sendbyCriteria将给定消息发送给与给定发送条件匹配的所有对等方
	SendByCriteria(*proto.SignedGossipMessage, SendCriteria) error

//getpeers返回被认为是活动的网络成员
	Peers() []discovery.NetworkMember

//peersofchannel返回被认为是活动的网络成员
//也订阅了给定的频道
	PeersOfChannel(common.ChainID) []discovery.NetworkMember

//updateMetadata更新发现层的自身元数据
//对等发布到其他对等
	UpdateMetadata(metadata []byte)

//更新LedgerHeight更新Ledger Height the Peer
//发布到频道中的其他对等端
	UpdateLedgerHeight(height uint64, chainID common.ChainID)

//更新链码更新对等发布的链码
//到渠道中的其他同行
	UpdateChaincodes(chaincode []*proto.Chaincode, chainID common.ChainID)

//八卦消息向网络中的其他对等方发送消息
	Gossip(msg *proto.GossipMessage)

//PeerFilter接收子通道SelectionCriteria并返回一个选择
//只有符合给定标准的对等身份，并且他们发布了自己的渠道参与
	PeerFilter(channel common.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error)

//accept返回与某个谓词匹配的其他节点发送的消息的专用只读通道。
//如果passthrough为false，则消息将由八卦层预先处理。
//如果passthrough是真的，那么八卦层不会介入，消息也不会
//可用于将答复发送回发件人
	Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage)

//JoinChan使八卦实例加入频道
	JoinChan(joinMsg api.JoinChannelMessage, chainID common.ChainID)

//LeaveChan让八卦实例离开一个频道。
//它仍然传播状态信息消息，但不参与。
//在拉块中，不能再返回一个对等列表
//在通道中。
	LeaveChan(chainID common.ChainID)

//SuspectPeers使八卦实例验证可疑对等的身份，并关闭
//与标识无效的对等方的任何连接
	SuspectPeers(s api.PeerSuspector)

//IdentityInfo returns information known peer identities
	IdentityInfo() api.PeerIdentitySet

//停止停止八卦组件
	Stop()
}

//EmittedGossipMessage封装签名的八卦消息以撰写
//with routing filter to be used while message is forwarded
type emittedGossipMessage struct {
	*proto.SignedGossipMessage
	filter func(id common.PKIidType) bool
}

//sendCriteria定义如何发送特定消息
type SendCriteria struct {
Timeout    time.Duration        //超时定义等待确认的时间
MinAck     int                  //MinAck defines the amount of peers to collect acknowledgements from
MaxPeers   int                  //maxpeers定义将消息发送到的最大对等数
IsEligible filter.RoutingFilter //is eligible定义特定对等方是否有资格接收消息
Channel    common.ChainID       //通道指定发送此消息的通道。\
//只有加入通道的对等方才会收到此消息
}

//字符串返回此发送条件的字符串表示形式
func (sc SendCriteria) String() string {
	return fmt.Sprintf("channel: %s, tout: %v, minAck: %d, maxPeers: %d", sc.Channel, sc.Timeout, sc.MinAck, sc.MaxPeers)
}

//config是八卦组件的配置
type Config struct {
BindPort            int      //我们绑定到的端口，仅用于测试
ID                  string   //此实例的ID
BootstrapPeers      []string //我们在启动时连接到的对等机
PropagateIterations int      //将消息推送到远程对等机的次数
PropagatePeerNum    int      //选择将消息推送到的对等机的数目

MaxBlockCountToStore int //存储在内存中的最大块数

MaxPropagationBurstSize    int           //在触发向远程对等机推送之前存储的最大消息数
MaxPropagationBurstLatency time.Duration //连续消息推送之间的最长时间

PullInterval time.Duration //确定拉相频率
PullPeerNum  int           //要从中提取的对等数

SkipBlockVerification bool //我们是否应该跳过验证阻塞消息

PublishCertPeriod        time.Duration //从启动证书开始的时间包含在活动消息中
PublishStateInfoInterval time.Duration //确定向对等端推送状态信息消息的频率
RequestStateInfoInterval time.Duration //确定从对等端提取状态信息消息的频率

TLSCerts *common.TLSCertificates //对等端的TLS证书

InternalEndpoint         string        //我们向组织中的对等方发布的端点
ExternalEndpoint         string        //对等端向外部组织发布此终结点而不是SelfEndpoint
TimeForMembershipTracker time.Duration //确定使用MembershipTracker进行轮询的时间
}
