
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


package msgprocessor

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//ChainCreator定义模拟通道创建所需的方法。
type ChainCreator interface {
//new channel config返回新通道的模板配置。
	NewChannelConfig(envConfigUpdate *cb.Envelope) (channelconfig.Resources, error)

//CreateBandle将配置解析为资源
	CreateBundle(channelID string, config *cb.Config) (channelconfig.Resources, error)

//channelsCount返回当前存在的通道数。
	ChannelsCount() int
}

//LimitedSupport定义SystemChannel筛选器所需的通道资源的子集。
type LimitedSupport interface {
	OrdererConfig() (channelconfig.Orderer, bool)
}

//SystemChainFilter实现了filter.rule接口。
type SystemChainFilter struct {
	cc      ChainCreator
	support LimitedSupport
}

//NewSystemChannelFilter返回*SystemChainFilter的新实例。
func NewSystemChannelFilter(ls LimitedSupport, cc ChainCreator) *SystemChainFilter {
	return &SystemChainFilter{
		support: ls,
		cc:      cc,
	}
}

//Apply拒绝错误消息。
func (scf *SystemChainFilter) Apply(env *cb.Envelope) error {
	msgData := &cb.Payload{}

	err := proto.Unmarshal(env.Payload, msgData)
	if err != nil {
		return errors.Errorf("bad payload: %s", err)
	}

	if msgData.Header == nil {
		return errors.Errorf("missing payload header")
	}

	chdr, err := utils.UnmarshalChannelHeader(msgData.Header.ChannelHeader)
	if err != nil {
		return errors.Errorf("bad channel header: %s", err)
	}

	if chdr.Type != int32(cb.HeaderType_ORDERER_TRANSACTION) {
		return nil
	}

	ordererConfig, ok := scf.support.OrdererConfig()
	if !ok {
		logger.Panicf("System channel does not have orderer config")
	}

	maxChannels := ordererConfig.MaxChannelsCount()
	if maxChannels > 0 {
//我们严格检查是否大于系统通道
		if uint64(scf.cc.ChannelsCount()) > maxChannels {
			return errors.Errorf("channel creation would exceed maximimum number of channels: %d", maxChannels)
		}
	}

	configTx := &cb.Envelope{}
	err = proto.Unmarshal(msgData.Data, configTx)
	if err != nil {
		return errors.Errorf("payload data error unmarshaling to envelope: %s", err)
	}

	return scf.authorizeAndInspect(configTx)
}

func (scf *SystemChainFilter) authorizeAndInspect(configTx *cb.Envelope) error {
	payload := &cb.Payload{}
	err := proto.Unmarshal(configTx.Payload, payload)
	if err != nil {
		return errors.Errorf("error unmarshaling wrapped configtx envelope payload: %s", err)
	}

	if payload.Header == nil {
		return errors.Errorf("wrapped configtx envelope missing header")
	}

	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return errors.Errorf("error unmarshaling wrapped configtx envelope channel header: %s", err)
	}

	if chdr.Type != int32(cb.HeaderType_CONFIG) {
		return errors.Errorf("wrapped configtx envelope not a config transaction")
	}

	configEnvelope := &cb.ConfigEnvelope{}
	err = proto.Unmarshal(payload.Data, configEnvelope)
	if err != nil {
		return errors.Errorf("error unmarshalling wrapped configtx config envelope from payload: %s", err)
	}

	if configEnvelope.LastUpdate == nil {
		return errors.Errorf("updated config does not include a config update")
	}

	res, err := scf.cc.NewChannelConfig(configEnvelope.LastUpdate)
	if err != nil {
		return errors.Errorf("error constructing new channel config from update: %s", err)
	}

//确保配置由适当的授权实体签名
	newChannelConfigEnv, err := res.ConfigtxValidator().ProposeConfigUpdate(configEnvelope.LastUpdate)
	if err != nil {
		return errors.Errorf("error proposing channel update to new channel config: %s", err)
	}

//reflect.deepequal在这里不起作用，因为它认为nil和空映射不相等
	if !proto.Equal(newChannelConfigEnv, configEnvelope) {
		return errors.Errorf("config proposed by the channel creation request did not match the config received with the channel creation request")
	}

	bundle, err := scf.cc.CreateBundle(res.ConfigtxValidator().ChainID(), newChannelConfigEnv.Config)
	if err != nil {
		return errors.Wrap(err, "config does not validly parse")
	}

	if err = res.ValidateNew(bundle); err != nil {
		return errors.Wrap(err, "new bundle invalid")
	}

	oc, ok := bundle.OrdererConfig()
	if !ok {
		return errors.New("config is missing orderer group")
	}

	if err = oc.Capabilities().Supported(); err != nil {
		return errors.Wrap(err, "config update is not compatible")
	}

	if err = bundle.ChannelConfig().Capabilities().Supported(); err != nil {
		return errors.Wrap(err, "config update is not compatible")
	}

	return nil
}
