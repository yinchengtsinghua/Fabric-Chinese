
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package msgprocessor

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/policies"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

//StandardChannel支持包括StandardChannel处理器所需的资源。
type StandardChannelSupport interface {
//序列应返回当前配置eq
	Sequence() uint64

//chainID返回通道ID
	ChainID() string

//签名者返回此订购者的签名者
	Signer() crypto.LocalSigner

//ProposeConfigUpdate接受config_update类型的信封，并生成
//将用作配置消息的信封有效负载数据的configendevelope
	ProposeConfigUpdate(configtx *cb.Envelope) (*cb.ConfigEnvelope, error)
}

//StandardChannel为标准现有通道实现处理器接口
type StandardChannel struct {
	support StandardChannelSupport
	filters *RuleSet
}

//新建标准消息处理器
func NewStandardChannel(support StandardChannelSupport, filters *RuleSet) *StandardChannel {
	return &StandardChannel{
		filters: filters,
		support: support,
	}
}

//CreateStandardChannelFilters为普通（非系统）链创建一组筛选器
func CreateStandardChannelFilters(filterSupport channelconfig.Resources) *RuleSet {
	ordererConfig, ok := filterSupport.OrdererConfig()
	if !ok {
		logger.Panicf("Missing orderer config")
	}
	return NewRuleSet([]Rule{
		EmptyRejectRule,
		NewExpirationRejectRule(filterSupport),
		NewSizeFilter(ordererConfig),
		NewSigFilter(policies.ChannelWriters, filterSupport),
	})
}

//ClassifyMSG检查消息以确定需要哪种类型的处理
func (s *StandardChannel) ClassifyMsg(chdr *cb.ChannelHeader) Classification {
	switch chdr.Type {
	case int32(cb.HeaderType_CONFIG_UPDATE):
		return ConfigUpdateMsg
	case int32(cb.HeaderType_ORDERER_TRANSACTION):
//为了保持向后兼容性，我们必须对这些消息进行分类
		return ConfigMsg
	case int32(cb.HeaderType_CONFIG):
//为了保持向后兼容性，我们必须对这些消息进行分类
		return ConfigMsg
	default:
		return NormalMsg
	}
}

//processnormalmsg将根据当前配置检查消息的有效性。它返回电流
//配置序列号，成功时为零，如果消息无效则为错误
func (s *StandardChannel) ProcessNormalMsg(env *cb.Envelope) (configSeq uint64, err error) {
	configSeq = s.support.Sequence()
	err = s.filters.Apply(env)
	return
}

//processconfigupdatemsg将尝试将config formost msg应用于当前配置，如果成功
//返回生成的配置消息和从中计算配置的configseq。如果配置推动消息
//无效，返回错误。
func (s *StandardChannel) ProcessConfigUpdateMsg(env *cb.Envelope) (config *cb.Envelope, configSeq uint64, err error) {
	logger.Debugf("Processing config update message for channel %s", s.support.ChainID())

//先调用序列。如果Seq在提议和接受之间进展顺利，这将导致再处理
//然而，如果序列被称为最后一个，那么成功可能被错误地归因于新的configseq
	seq := s.support.Sequence()
	err = s.filters.Apply(env)
	if err != nil {
		return nil, 0, err
	}

	configEnvelope, err := s.support.ProposeConfigUpdate(env)
	if err != nil {
		return nil, 0, err
	}

	config, err = utils.CreateSignedEnvelope(cb.HeaderType_CONFIG, s.support.ChainID(), s.support.Signer(), configEnvelope, msgVersion, epoch)
	if err != nil {
		return nil, 0, err
	}

//我们在这里重新应用过滤器，特别是对于大小过滤器，以确保
//刚建造的对我们的同意者来说不算太大。它还重新应用了签名
//检查，虽然不是严格必要的，但它是一个良好的健全性检查，以防订购者
//尚未配置正确的证书材料。签名的额外开销
//检查是可忽略的，因为这是重新配置路径，而不是正常路径。
	err = s.filters.Apply(config)
	if err != nil {
		return nil, 0, err
	}

	return config, seq, nil
}

//processconfigmsg接受类型为“headertype_config”的信封，并从中解压缩“configendevelope”
//从“lastupdate”字段中提取“configupdate”，并对其调用“processconfigupdatemsg”。
func (s *StandardChannel) ProcessConfigMsg(env *cb.Envelope) (config *cb.Envelope, configSeq uint64, err error) {
	logger.Debugf("Processing config message for channel %s", s.support.ChainID())

	configEnvelope := &cb.ConfigEnvelope{}
	_, err = utils.UnmarshalEnvelopeOfType(env, cb.HeaderType_CONFIG, configEnvelope)
	if err != nil {
		return
	}

	return s.ProcessConfigUpdateMsg(configEnvelope.LastUpdate)
}
