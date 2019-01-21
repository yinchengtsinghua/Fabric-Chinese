
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


package channelconfig

import (
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("common.channelconfig")

//rootgroupkey是用于通道配置的名称间距的键，特别是用于
//政策评估。
const RootGroupKey = "Channel"

//bundle是一组始终具有一致性的资源
//通道配置视图。特别是，对于给定的束引用，
//配置序列、策略管理器等始终返回
//同样的价值。束结构是不可变的，将始终在其
//整体，带有新的后备存储器。
type Bundle struct {
	policyManager   policies.Manager
	mspManager      msp.MSPManager
	channelConfig   *ChannelConfig
	configtxManager configtx.Validator
}

//policyManager返回为此配置构造的策略管理器
func (b *Bundle) PolicyManager() policies.Manager {
	return b.policyManager
}

//msp manager返回为此配置构造的msp管理器
func (b *Bundle) MSPManager() msp.MSPManager {
	return b.channelConfig.MSPManager()
}

//channel config返回链的config.channel
func (b *Bundle) ChannelConfig() Channel {
	return b.channelConfig
}

//orderconfig返回通道的config.order
//以及医嘱者配置是否存在
func (b *Bundle) OrdererConfig() (Orderer, bool) {
	result := b.channelConfig.OrdererConfig()
	return result, result != nil
}

//consortiums config（）返回通道的config.consortiums
//以及联合体配置是否存在
func (b *Bundle) ConsortiumsConfig() (Consortiums, bool) {
	result := b.channelConfig.ConsortiumsConfig()
	return result, result != nil
}

//applicationconfig返回通道的configtxapplication.sharedconfig
//以及应用程序配置是否存在
func (b *Bundle) ApplicationConfig() (Application, bool) {
	result := b.channelConfig.ApplicationConfig()
	return result, result != nil
}

//configtx validator返回通道的configtx.validator
func (b *Bundle) ConfigtxValidator() configtx.Validator {
	return b.configtxManager
}

//validateNew检查从当前包派生的新包包含的配置是否有效。
//这允许对性质进行检查“确保共识类型没有改变”，否则
func (b *Bundle) ValidateNew(nb Resources) error {
	if oc, ok := b.OrdererConfig(); ok {
		noc, ok := nb.OrdererConfig()
		if !ok {
			return errors.New("Current config has orderer section, but new config does not")
		}

		if oc.ConsensusType() != noc.ConsensusType() {
			return errors.Errorf("Attempted to change consensus type from %s to %s", oc.ConsensusType(), noc.ConsensusType())
		}

		for orgName, org := range oc.Organizations() {
			norg, ok := noc.Organizations()[orgName]
			if !ok {
				continue
			}
			mspID := org.MSPID()
			if mspID != norg.MSPID() {
				return errors.Errorf("Orderer org %s attempted to change MSP ID from %s to %s", orgName, mspID, norg.MSPID())
			}
		}
	}

	if ac, ok := b.ApplicationConfig(); ok {
		nac, ok := nb.ApplicationConfig()
		if !ok {
			return errors.New("Current config has application section, but new config does not")
		}

		for orgName, org := range ac.Organizations() {
			norg, ok := nac.Organizations()[orgName]
			if !ok {
				continue
			}
			mspID := org.MSPID()
			if mspID != norg.MSPID() {
				return errors.Errorf("Application org %s attempted to change MSP ID from %s to %s", orgName, mspID, norg.MSPID())
			}
		}
	}

	if cc, ok := b.ConsortiumsConfig(); ok {
		ncc, ok := nb.ConsortiumsConfig()
		if !ok {
			return errors.Errorf("Current config has consortiums section, but new config does not")
		}

		for consortiumName, consortium := range cc.Consortiums() {
			nconsortium, ok := ncc.Consortiums()[consortiumName]
			if !ok {
				continue
			}

			for orgName, org := range consortium.Organizations() {
				norg, ok := nconsortium.Organizations()[orgName]
				if !ok {
					continue
				}
				mspID := org.MSPID()
				if mspID != norg.MSPID() {
					return errors.Errorf("Consortium %s org %s attempted to change MSP ID from %s to %s", consortiumName, orgName, mspID, norg.MSPID())
				}
			}
		}
	}

	return nil
}

//newbundleFromEnvelope包装newbundle函数，提取所需的
//来自完整配置X的信息
func NewBundleFromEnvelope(env *cb.Envelope) (*Bundle, error) {
	payload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal payload from envelope")
	}

	configEnvelope, err := configtx.UnmarshalConfigEnvelope(payload.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config envelope from payload")
	}

	if payload.Header == nil {
		return nil, errors.Errorf("envelope header cannot be nil")
	}

	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal channel header")
	}

	return NewBundle(chdr.ChannelId, configEnvelope.Config)
}

//Newbundle创建了一个新的不变配置包
func NewBundle(channelID string, config *cb.Config) (*Bundle, error) {
	if err := preValidate(config); err != nil {
		return nil, err
	}

	channelConfig, err := NewChannelConfig(config.ChannelGroup)
	if err != nil {
		return nil, errors.Wrap(err, "initializing channelconfig failed")
	}

	policyProviderMap := make(map[int32]policies.Provider)
	for pType := range cb.Policy_PolicyType_name {
		rtype := cb.Policy_PolicyType(pType)
		switch rtype {
		case cb.Policy_UNKNOWN:
//不注册处理程序
		case cb.Policy_SIGNATURE:
			policyProviderMap[pType] = cauthdsl.NewPolicyProvider(channelConfig.MSPManager())
		case cb.Policy_MSP:
//在此处添加MSP处理程序的钩子
		}
	}

	policyManager, err := policies.NewManagerImpl(RootGroupKey, policyProviderMap, config.ChannelGroup)
	if err != nil {
		return nil, errors.Wrap(err, "initializing policymanager failed")
	}

	configtxManager, err := configtx.NewValidatorImpl(channelID, config, RootGroupKey, policyManager)
	if err != nil {
		return nil, errors.Wrap(err, "initializing configtx manager failed")
	}

	return &Bundle{
		policyManager:   policyManager,
		channelConfig:   channelConfig,
		configtxManager: configtxManager,
	}, nil
}

func preValidate(config *cb.Config) error {
	if config == nil {
		return errors.New("channelconfig Config cannot be nil")
	}

	if config.ChannelGroup == nil {
		return errors.New("config must contain a channel group")
	}

	if og, ok := config.ChannelGroup.Groups[OrdererGroupKey]; ok {
		if _, ok := og.Values[CapabilitiesKey]; !ok {
			if _, ok := config.ChannelGroup.Values[CapabilitiesKey]; ok {
				return errors.New("cannot enable channel capabilities without orderer support first")
			}

			if ag, ok := config.ChannelGroup.Groups[ApplicationGroupKey]; ok {
				if _, ok := ag.Values[CapabilitiesKey]; ok {
					return errors.New("cannot enable application capabilities without orderer support first")
				}
			}
		}
	}

	return nil
}
