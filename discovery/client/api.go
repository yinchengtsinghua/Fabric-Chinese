
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
	"github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var (
//errnotfound定义了一个错误，这意味着找不到元素
	ErrNotFound = errors.New("not found")
)

//签名者在消息上签名并返回签名和nil，
//or nil and error on failure
type Signer func(msg []byte) ([]byte, error)

//拨号程序连接到服务器
type Dialer func() (*grpc.ClientConn, error)

//响应聚合了来自发现服务的多个响应
type Response interface {
//ForChannel返回给定通道上下文中的ChannelResponse
	ForChannel(string) ChannelResponse

//forlocal返回无通道上下文中的localresponse
	ForLocal() LocalResponse
}

//通道响应聚合给定通道的响应
type ChannelResponse interface {
//config返回对config查询的响应，或者在出现错误时返回错误
	Config() (*discovery.ConfigResult, error)

//对等方返回对等方成员身份查询的响应，或者在出现错误时返回错误
	Peers(invocationChain ...*discovery.ChaincodeCall) ([]*Peer, error)

//背书人返回对给定的背书人查询的响应
//给定通道上下文中的链代码，或者出错时出错。
//该方法返回一组随机的背书人，以便所有背书人的签名
//合起来，满足背书政策。
//选择基于给定的选择提示：
//筛选：筛选和排序背书人
//给定的invocationChain指定链码调用（以及集合）
//客户在构建请求期间通过
	Endorsers(invocationChain InvocationChain, f Filter) (Endorsers, error)
}

//LocalResponse聚合无通道作用域的响应
type LocalResponse interface {
//对等方返回本地对等方成员身份查询的响应，或者在出现错误时返回错误
	Peers() ([]*Peer, error)
}

//Endorsers defines a set of peers that are sufficient
//满足一些链码的认可政策
type Endorsers []*Peer

//对等机聚合身份、成员身份和通道范围的信息
//of a certain peer.
type Peer struct {
	MSPID            string
	AliveMessage     *gossip.SignedGossipMessage
	StateInfoMessage *gossip.SignedGossipMessage
	Identity         []byte
}
