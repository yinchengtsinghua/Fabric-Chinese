
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


package acl

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

var (
	logger = flogging.MustGetLogger("discovery.acl")
)

//channelconfiggetter允许检索频道配置资源
type ChannelConfigGetter interface {
//getchannelconfig返回通道配置的资源
	GetChannelConfig(cid string) channelconfig.Resources
}

//channelconfiggetterfunc返回通道配置的资源
type ChannelConfigGetterFunc func(cid string) channelconfig.Resources

//getchannelconfig返回通道配置的资源
func (f ChannelConfigGetterFunc) GetChannelConfig(cid string) channelconfig.Resources {
	return f(cid)
}

//验证程序验证签名和消息
type Verifier interface {
//VerifyByChannel检查签名是否为消息的有效签名
//在对等机的验证密钥下，也在特定通道的上下文中。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为零，则验证失败。
	VerifyByChannel(channel string, sd *cb.SignedData) error
}

//评估器评估签名。
//它用于评估本地MSP的签名
type Evaluator interface {
//Evaluate获取一组SignedData并评估该组签名是否满足策略
	Evaluate(signatureSet []*cb.SignedData) error
}

//DiscoverySupport实现用于服务发现的支持
//与访问控制有关
type DiscoverySupport struct {
	ChannelConfigGetter
	Verifier
	Evaluator
}

//新建DiscoverySupport创建新的DiscoverySupport
func NewDiscoverySupport(v Verifier, e Evaluator, chanConf ChannelConfigGetter) *DiscoverySupport {
	return &DiscoverySupport{Verifier: v, Evaluator: e, ChannelConfigGetter: chanConf}
}

//合格返回给定对等方是否有资格接收
//来自给定通道的发现服务的服务
func (s *DiscoverySupport) EligibleForService(channel string, data cb.SignedData) error {
	if channel == "" {
		return s.Evaluate([]*cb.SignedData{&data})
	}
	return s.VerifyByChannel(channel, &data)
}

//configSequence返回给定通道的配置序列
func (s *DiscoverySupport) ConfigSequence(channel string) uint64 {
//如果通道为空，则没有序列
	if channel == "" {
		return 0
	}
	conf := s.GetChannelConfig(channel)
	if conf == nil {
		logger.Panic("Failed obtaining channel config for channel", channel)
	}
	v := conf.ConfigtxValidator()
	if v == nil {
		logger.Panic("ConfigtxValidator for channel", channel, "is nil")
	}
	return v.Sequence()
}

func (s *DiscoverySupport) SatisfiesPrincipal(channel string, rawIdentity []byte, principal *msp.MSPPrincipal) error {
	conf := s.GetChannelConfig(channel)
	if conf == nil {
		return errors.Errorf("channel %s doesn't exist", channel)
	}
	mspMgr := conf.MSPManager()
	if mspMgr == nil {
		return errors.Errorf("could not find MSP manager for channel %s", channel)
	}
	identity, err := mspMgr.DeserializeIdentity(rawIdentity)
	if err != nil {
		return errors.Wrap(err, "failed deserializing identity")
	}
	return identity.SatisfiesPrincipal(principal)
}

//go:generate mokery-name channelpolicymanagergetter-case underline-output../mocks/

//ChannelPolicyManagerGetter是一个支持接口
//访问给定通道的策略管理器
type ChannelPolicyManagerGetter interface {
//返回与传递的通道关联的策略管理器
//如果是经理请求的，则为true；如果是默认经理，则为false。
	Manager(channelID string) (policies.Manager, bool)
}

//NewChannelVerifier从给定的策略和策略管理器getter返回新的通道验证程序
func NewChannelVerifier(policy string, polMgr policies.ChannelPolicyManagerGetter) *ChannelVerifier {
	return &ChannelVerifier{
		Policy:                     policy,
		ChannelPolicyManagerGetter: polMgr,
	}
}

//ChannelVerifier在通道上下文中验证签名和消息
type ChannelVerifier struct {
	policies.ChannelPolicyManagerGetter
	Policy string
}

//VerifyByChannel检查签名是否为消息的有效签名
//在对等机的验证密钥下，也在特定通道的上下文中。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peerIdentity为零，则验证失败。
func (cv *ChannelVerifier) VerifyByChannel(channel string, sd *cb.SignedData) error {
	mgr, _ := cv.Manager(channel)
	if mgr == nil {
		return errors.Errorf("policy manager for channel %s doesn't exist", channel)
	}
	pol, _ := mgr.GetPolicy(cv.Policy)
	if pol == nil {
		return errors.New("failed obtaining channel application writers policy")
	}
	return pol.Evaluate([]*cb.SignedData{sd})
}
