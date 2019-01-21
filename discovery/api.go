
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
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	common2 "github.com/hyperledger/fabric/protos/common"
	discprotos "github.com/hyperledger/fabric/protos/discovery"
)

//访问控制支持检查客户是否有资格接受服务
type AccessControlSupport interface {
//合格返回给定对等方是否有资格接收
//service from the discovery service for a given channel
	EligibleForService(channel string, data common2.SignedData) error
}

//configSequenceSupport返回给定通道的配置序列
type ConfigSequenceSupport interface {
//configSequence返回给定通道的配置序列
	ConfigSequence(channel string) uint64
}

//go:generate mockery -name GossipSupport -case underscore -output ../support/mocks/

//八卦支持聚合八卦模块
//向发现服务提供，例如了解有关对等方的信息
type GossipSupport interface {
//channel exists返回给定通道是否存在
	ChannelExists(channel string) bool

//peersofchannel返回被认为是活动的网络成员
//and also subscribed to the channel given
	PeersOfChannel(common.ChainID) discovery.Members

//对等方返回被认为是活动的网络成员
	Peers() discovery.Members

//IdentityInfo返回有关对等方的标识信息
	IdentityInfo() api.PeerIdentitySet
}

//EndorsementSupport provides knowledge of endorsement policy selection
//链码
type EndorsementSupport interface {
//PeersforElement返回给定对等方、通道和链码集的认可描述符
	PeersForEndorsement(channel common.ChainID, interest *discprotos.ChaincodeInterest) (*discprotos.EndorsementDescriptor, error)

//PeersAuthorizedByCriteria returns the peers of the channel that are authorized by the given chaincode interest
//这是考虑到，如果感兴趣的链码安装在对等端上，并且
//考虑到对等端是否是链码集合的一部分。
//如果传递了零利息或空利息，则不进行过滤。
	PeersAuthorizedByCriteria(chainID common.ChainID, interest *discprotos.ChaincodeInterest) (discovery.Members, error)
}

//配置支持提供对通道配置的访问
type ConfigSupport interface {
//config返回通道的配置
	Config(channel string) (*discprotos.ConfigResult, error)
}

//Support defines an interface that allows the discovery service
//获取其他对等组件拥有的信息
type Support interface {
	AccessControlSupport
	GossipSupport
	EndorsementSupport
	ConfigSupport
	ConfigSequenceSupport
}
