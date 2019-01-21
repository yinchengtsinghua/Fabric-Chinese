
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


package discovery

import (
	"fmt"

	"github.com/hyperledger/fabric/gossip/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

//CryptoService是发现期望在创建时实现和传递的接口。
type CryptoService interface {
//validateAliveMsg验证活动消息是否可信
	ValidateAliveMsg(message *proto.SignedGossipMessage) bool

//签署消息签署消息
	SignMessage(m *proto.GossipMessage, internalEndpoint string) *proto.Envelope
}

//信封过滤器可以或不能移除信封的一部分。
//that the given SignedGossipMessage originates from.
type EnvelopeFilter func(message *proto.SignedGossipMessage) *proto.Envelope

//筛选定义允许发送到某个远程对等机的消息，
//基于某些标准。
//返回筛选器是否允许发送给定的消息。
type Sieve func(message *proto.SignedGossipMessage) bool

//disclosurepolicy定义给定远程对等端的消息
//有资格知道，也有资格知道什么是合格的
//从给定的已签名的ossipMessage中了解。
//返回：
//1）给定远程对等机的筛选。
//筛选应用于问题和输出中的每个对等点。
//消息是否应向远程对等方公开。
//2）给定已签名的ossipMessage的信封筛选器，它可以删除
//签名的ossipMessage源自的信封的一部分
type DisclosurePolicy func(remotePeer *NetworkMember) (Sieve, EnvelopeFilter)

//commservice是发现期望在创建时实现和传递的接口。
type CommService interface {
//流言蜚语
	Gossip(msg *proto.SignedGossipMessage)

//sendtopeer向给定的对等端发送消息。
//由于通信模块本身处理nonce，因此nonce可以是任何东西。
	SendToPeer(peer *NetworkMember, msg *proto.SignedGossipMessage)

//ping探测远程对等机并返回是否响应
	Ping(peer *NetworkMember) bool

//accept返回从远程对等方发送的成员身份消息的只读通道
	Accept() <-chan proto.ReceivedMessage

//ExpertedHead为假定已死亡的对等端返回只读通道
	PresumedDead() <-chan common.PKIidType

//关闭CONNEN命令以关闭与某个对等体的连接
	CloseConn(peer *NetworkMember)

//转发将消息发送到下一个跃点，不包括跃点
//最初接收消息的来源
	Forward(msg proto.ReceivedMessage)
}

//networkmember是对等方的表示
type NetworkMember struct {
	Endpoint         string
	Metadata         []byte
	PKIid            common.PKIidType
	InternalEndpoint string
	Properties       *proto.Properties
	*proto.Envelope
}

//string返回networkmember的字符串表示形式
func (n NetworkMember) String() string {
	return fmt.Sprintf("Endpoint: %s, InternalEndpoint: %s, PKI-ID: %s, Metadata: %x", n.Endpoint, n.InternalEndpoint, n.PKIid, n.Metadata)
}

//PreferredEndpoint计算要连接的端点，
//在首选内部终结点而不是标准终结点时
//端点
func (n NetworkMember) PreferredEndpoint() string {
	if n.InternalEndpoint != "" {
		return n.InternalEndpoint
	}
	return n.Endpoint
}

//对等身份包括远程对等的
//PKI-ID and whether its in the same org as the current
//同侪与否
type PeerIdentification struct {
	ID      common.PKIidType
	SelfOrg bool
}

type identifier func() (*PeerIdentification, error)

//发现是表示发现模块的接口
type Discovery interface {
//Lookup returns a network member, or nil if not found
	Lookup(PKIID common.PKIidType) *NetworkMember

//self返回此实例的成员身份信息
	Self() NetworkMember

//updateMetadata更新此实例的元数据
	UpdateMetadata([]byte)

//updateEndpoint更新此实例的终结点
	UpdateEndpoint(string)

//停止此实例
	Stop()

//GetMembership返回视图中的活动成员
	GetMembership() []NetworkMember

//InitiateSync使实例询问给定数量的对等方
//他们的会员信息
	InitiateSync(peerNum int)

//Connect使此实例连接到远程实例
//标识符参数是一个可用于标识
//以及断言其pki-id，无论是否在对等组织中，
//行动是否成功
	Connect(member NetworkMember, id identifier)
}

//成员表示网络成员的聚合
type Members []NetworkMember

//byid返回pki id的映射（字符串形式）
//到网络成员
func (members Members) ByID() map[string]NetworkMember {
	res := make(map[string]NetworkMember, len(members))
	for _, peer := range members {
		res[string(peer.PKIid)] = peer
	}
	return res
}

//intersect返回两个成员的交集
func (members Members) Intersect(otherMembers Members) Members {
	var res Members
	m := otherMembers.ByID()
	for _, member := range members {
		if _, exists := m[string(member.PKIid)]; exists {
			res = append(res, member)
		}
	}
	return res
}

//筛选器只返回满足给定筛选器的成员
func (members Members) Filter(filter func(member NetworkMember) bool) Members {
	var res Members
	for _, member := range members {
		if filter(member) {
			res = append(res, member)
		}
	}
	return res
}

//map向成员中的每个networkmember调用给定函数
func (members Members) Map(f func(member NetworkMember) NetworkMember) Members {
	var res Members
	for _, m := range members {
		res = append(res, f(m))
	}
	return res
}

//HaveExternalEndpoints selects network members that have external endpoints
func HasExternalEndpoint(member NetworkMember) bool {
	return member.Endpoint != ""
}
