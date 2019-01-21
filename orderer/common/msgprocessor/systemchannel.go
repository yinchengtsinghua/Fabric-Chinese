
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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/policies"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

//ChannelConfigTemplator可用于生成配置模板。
type ChannelConfigTemplator interface {
//newchannelconfig创建新的模板配置管理器。
	NewChannelConfig(env *cb.Envelope) (channelconfig.Resources, error)
}

//SystemChannel实现系统通道的处理器接口。
type SystemChannel struct {
	*StandardChannel
	templator ChannelConfigTemplator
}

//NewSystemChannel创建新的系统通道消息处理器。
func NewSystemChannel(support StandardChannelSupport, templator ChannelConfigTemplator, filters *RuleSet) *SystemChannel {
	logger.Debugf("Creating system channel msg processor for channel %s", support.ChainID())
	return &SystemChannel{
		StandardChannel: NewStandardChannel(support, filters),
		templator:       templator,
	}
}

//CreateSystemChannelFilters为订购系统链创建一组筛选器。
func CreateSystemChannelFilters(chainCreator ChainCreator, ledgerResources channelconfig.Resources) *RuleSet {
	ordererConfig, ok := ledgerResources.OrdererConfig()
	if !ok {
		logger.Panicf("Cannot create system channel filters without orderer config")
	}
	return NewRuleSet([]Rule{
		EmptyRejectRule,
		NewExpirationRejectRule(ledgerResources),
		NewSizeFilter(ordererConfig),
		NewSigFilter(policies.ChannelWriters, ledgerResources),
		NewSystemChannelFilter(ledgerResources, chainCreator),
	})
}

//processnormalmsg处理普通消息，如果它们未绑定到系统通道ID，则拒绝它们。
//与errChannel无关。
func (s *SystemChannel) ProcessNormalMsg(msg *cb.Envelope) (configSeq uint64, err error) {
	channelID, err := utils.ChannelID(msg)
	if err != nil {
		return 0, err
	}

//对于标准通道消息处理，我们不会检查通道ID，
//因为消息处理器是按通道ID查找的。
//但是，系统通道消息处理器是捕获所有消息的工具。
//它与现存的通道不对应，所以我们必须在这里检查它。
	if channelID != s.support.ChainID() {
		return 0, ErrChannelDoesNotExist
	}

	return s.StandardChannel.ProcessNormalMsg(msg)
}

//processconfigupdatemsg处理系统通道本身的config_update类型的消息
//或者，用于创建通道。在通道创建案例中，配置更新被包装成
//订购方事务，在标准配置更新案例中，生成配置消息
func (s *SystemChannel) ProcessConfigUpdateMsg(envConfigUpdate *cb.Envelope) (config *cb.Envelope, configSeq uint64, err error) {
	channelID, err := utils.ChannelID(envConfigUpdate)
	if err != nil {
		return nil, 0, err
	}

	logger.Debugf("Processing config update tx with system channel message processor for channel ID %s", channelID)

	if channelID == s.support.ChainID() {
		return s.StandardChannel.ProcessConfigUpdateMsg(envConfigUpdate)
	}

//我们应该检查外层信封上的签名是否至少对系统通道中的某些MSP有效。

	logger.Debugf("Processing channel create tx for channel %s on system channel %s", channelID, s.support.ChainID())

//如果通道ID与系统通道不匹配，则这必须是通道创建事务。

	bundle, err := s.templator.NewChannelConfig(envConfigUpdate)
	if err != nil {
		return nil, 0, err
	}

	newChannelConfigEnv, err := bundle.ConfigtxValidator().ProposeConfigUpdate(envConfigUpdate)
	if err != nil {
		return nil, 0, err
	}

	newChannelEnvConfig, err := utils.CreateSignedEnvelope(cb.HeaderType_CONFIG, channelID, s.support.Signer(), newChannelConfigEnv, msgVersion, epoch)
	if err != nil {
		return nil, 0, err
	}

	wrappedOrdererTransaction, err := utils.CreateSignedEnvelope(cb.HeaderType_ORDERER_TRANSACTION, s.support.ChainID(), s.support.Signer(), newChannelEnvConfig, msgVersion, epoch)
	if err != nil {
		return nil, 0, err
	}

//我们在这里重新应用过滤器，特别是对于大小过滤器，以确保
//刚建造的对我们的同意者来说不算太大。它还重新应用了签名
//检查，虽然不是严格必要的，但它是一个良好的健全性检查，以防订购者
//尚未配置正确的证书材料。签名的额外开销
//检查是可忽略的，因为这是通道创建路径，而不是正常路径。
	err = s.StandardChannel.filters.Apply(wrappedOrdererTransaction)
	if err != nil {
		return nil, 0, err
	}

	return wrappedOrdererTransaction, s.support.Sequence(), nil
}

//processconfigmsg采用以下两种类型的信封：
//-`headertype_config`：系统通道本身是config的目标，我们只需解包'configupdate`。
//信封来自'lastupdate'字段，并在基础标准频道上调用'processconfigupdatemsg'
//-`headertype_order_transaction`：这是一条创建频道的消息，我们打开'configupdate'信封
//并在上面运行'processconfigupdatemsg'
func (s *SystemChannel) ProcessConfigMsg(env *cb.Envelope) (*cb.Envelope, uint64, error) {
	payload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, 0, err
	}

	if payload.Header == nil {
		return nil, 0, fmt.Errorf("Abort processing config msg because no head was set")
	}

	if payload.Header.ChannelHeader == nil {
		return nil, 0, fmt.Errorf("Abort processing config msg because no channel header was set")
	}

	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, 0, fmt.Errorf("Abort processing config msg because channel header unmarshalling error: %s", err)
	}

	switch chdr.Type {
	case int32(cb.HeaderType_CONFIG):
		configEnvelope := &cb.ConfigEnvelope{}
		if err = proto.Unmarshal(payload.Data, configEnvelope); err != nil {
			return nil, 0, err
		}

		return s.StandardChannel.ProcessConfigUpdateMsg(configEnvelope.LastUpdate)

	case int32(cb.HeaderType_ORDERER_TRANSACTION):
		env, err := utils.UnmarshalEnvelope(payload.Data)
		if err != nil {
			return nil, 0, fmt.Errorf("Abort processing config msg because payload data unmarshalling error: %s", err)
		}

		configEnvelope := &cb.ConfigEnvelope{}
		_, err = utils.UnmarshalEnvelopeOfType(env, cb.HeaderType_CONFIG, configEnvelope)
		if err != nil {
			return nil, 0, fmt.Errorf("Abort processing config msg because payload data unmarshalling error: %s", err)
		}

		return s.ProcessConfigUpdateMsg(configEnvelope.LastUpdate)

	default:
		return nil, 0, fmt.Errorf("Panic processing config msg due to unexpected envelope type %s", cb.HeaderType_name[chdr.Type])
	}
}

//DefaultTemplatorSupport是DefaultTemplator所需的通道配置的子集。
type DefaultTemplatorSupport interface {
//consortiums config返回排序系统通道的consortiums配置。
	ConsortiumsConfig() (channelconfig.Consortiums, bool)

//orderconfig返回排序配置以及配置是否存在
	OrdererConfig() (channelconfig.Orderer, bool)

//configtxvalidator返回与系统通道当前配置相对应的configtx管理器。
	ConfigtxValidator() configtx.Validator

//signer返回适合对转发消息进行签名的本地签名者。
	Signer() crypto.LocalSigner
}

//DefaultTemplator实现了ChannelConfigTemplator接口，是在生产部署中使用的接口。
type DefaultTemplator struct {
	support DefaultTemplatorSupport
}

//NewDefaultTemplator返回DefaultTemplator的实例。
func NewDefaultTemplator(support DefaultTemplatorSupport) *DefaultTemplator {
	return &DefaultTemplator{
		support: support,
	}
}

//new channel config根据订购系统通道中的当前配置创建新的模板通道配置。
func (dt *DefaultTemplator) NewChannelConfig(envConfigUpdate *cb.Envelope) (channelconfig.Resources, error) {
	configUpdatePayload, err := utils.UnmarshalPayload(envConfigUpdate.Payload)
	if err != nil {
		return nil, fmt.Errorf("Failing initial channel config creation because of payload unmarshaling error: %s", err)
	}

	configUpdateEnv, err := configtx.UnmarshalConfigUpdateEnvelope(configUpdatePayload.Data)
	if err != nil {
		return nil, fmt.Errorf("Failing initial channel config creation because of config update envelope unmarshaling error: %s", err)
	}

	if configUpdatePayload.Header == nil {
		return nil, fmt.Errorf("Failed initial channel config creation because config update header was missing")
	}

	channelHeader, err := utils.UnmarshalChannelHeader(configUpdatePayload.Header.ChannelHeader)
	if err != nil {
		return nil, fmt.Errorf("Failed initial channel config creation because channel header was malformed: %s", err)
	}

	configUpdate, err := configtx.UnmarshalConfigUpdate(configUpdateEnv.ConfigUpdate)
	if err != nil {
		return nil, fmt.Errorf("Failing initial channel config creation because of config update unmarshaling error: %s", err)
	}

	if configUpdate.ChannelId != channelHeader.ChannelId {
		return nil, fmt.Errorf("Failing initial channel config creation: mismatched channel IDs: '%s' != '%s'", configUpdate.ChannelId, channelHeader.ChannelId)
	}

	if configUpdate.WriteSet == nil {
		return nil, fmt.Errorf("Config update has an empty writeset")
	}

	if configUpdate.WriteSet.Groups == nil || configUpdate.WriteSet.Groups[channelconfig.ApplicationGroupKey] == nil {
		return nil, fmt.Errorf("Config update has missing application group")
	}

	if uv := configUpdate.WriteSet.Groups[channelconfig.ApplicationGroupKey].Version; uv != 1 {
		return nil, fmt.Errorf("Config update for channel creation does not set application group version to 1, was %d", uv)
	}

	consortiumConfigValue, ok := configUpdate.WriteSet.Values[channelconfig.ConsortiumKey]
	if !ok {
		return nil, fmt.Errorf("Consortium config value missing")
	}

	consortium := &cb.Consortium{}
	err = proto.Unmarshal(consortiumConfigValue.Value, consortium)
	if err != nil {
		return nil, fmt.Errorf("Error reading unmarshaling consortium name: %s", err)
	}

	applicationGroup := cb.NewConfigGroup()
	consortiumsConfig, ok := dt.support.ConsortiumsConfig()
	if !ok {
		return nil, fmt.Errorf("The ordering system channel does not appear to support creating channels")
	}

	consortiumConf, ok := consortiumsConfig.Consortiums()[consortium.Name]
	if !ok {
		return nil, fmt.Errorf("Unknown consortium name: %s", consortium.Name)
	}

	applicationGroup.Policies[channelconfig.ChannelCreationPolicyKey] = &cb.ConfigPolicy{
		Policy: consortiumConf.ChannelCreationPolicy(),
	}
	applicationGroup.ModPolicy = channelconfig.ChannelCreationPolicyKey

//获取当前系统通道配置
	systemChannelGroup := dt.support.ConfigtxValidator().ConfigProto().ChannelGroup

//如果联合体集团没有成员，则允许源请求没有成员。然而，
//如果联合体集团有任何成员，则源请求中必须至少有一个成员。
	if len(systemChannelGroup.Groups[channelconfig.ConsortiumsGroupKey].Groups[consortium.Name].Groups) > 0 &&
		len(configUpdate.WriteSet.Groups[channelconfig.ApplicationGroupKey].Groups) == 0 {
		return nil, fmt.Errorf("Proposed configuration has no application group members, but consortium contains members")
	}

//如果联合体没有成员，则允许源请求包含任意成员
//否则，要求提供的成员是联合体成员的一个子集。
	if len(systemChannelGroup.Groups[channelconfig.ConsortiumsGroupKey].Groups[consortium.Name].Groups) > 0 {
		for orgName := range configUpdate.WriteSet.Groups[channelconfig.ApplicationGroupKey].Groups {
			consortiumGroup, ok := systemChannelGroup.Groups[channelconfig.ConsortiumsGroupKey].Groups[consortium.Name].Groups[orgName]
			if !ok {
				return nil, fmt.Errorf("Attempted to include a member which is not in the consortium")
			}
			applicationGroup.Groups[orgName] = proto.Clone(consortiumGroup).(*cb.ConfigGroup)
		}
	}

	channelGroup := cb.NewConfigGroup()

//将系统通道级别配置复制到新配置
	for key, value := range systemChannelGroup.Values {
		channelGroup.Values[key] = proto.Clone(value).(*cb.ConfigValue)
		if key == channelconfig.ConsortiumKey {
//不要设置联合体名称，我们稍后再设置。
			continue
		}
	}

	for key, policy := range systemChannelGroup.Policies {
		channelGroup.Policies[key] = proto.Clone(policy).(*cb.ConfigPolicy)
	}

//将新配置医嘱者组设置为系统通道医嘱者组，将应用程序组设置为新应用程序组
	channelGroup.Groups[channelconfig.OrdererGroupKey] = proto.Clone(systemChannelGroup.Groups[channelconfig.OrdererGroupKey]).(*cb.ConfigGroup)
	channelGroup.Groups[channelconfig.ApplicationGroupKey] = applicationGroup
	channelGroup.Values[channelconfig.ConsortiumKey] = &cb.ConfigValue{
		Value:     utils.MarshalOrPanic(channelconfig.ConsortiumValue(consortium.Name).Value()),
		ModPolicy: channelconfig.AdminsPolicyKey,
	}

//v1.1中引入了不向后兼容的错误修复程序
//一旦v1.0被否决，就应该删除功能检查
	if oc, ok := dt.support.OrdererConfig(); ok && oc.Capabilities().PredictableChannelTemplate() {
		channelGroup.ModPolicy = systemChannelGroup.ModPolicy
		zeroVersions(channelGroup)
	}

	bundle, err := channelconfig.NewBundle(channelHeader.ChannelId, &cb.Config{
		ChannelGroup: channelGroup,
	})

	if err != nil {
		return nil, err
	}

	return bundle, nil
}

//ZeroVersions在配置树上递归迭代，将所有版本设置为零
func zeroVersions(cg *cb.ConfigGroup) {
	cg.Version = 0

	for _, value := range cg.Values {
		value.Version = 0
	}

	for _, policy := range cg.Policies {
		policy.Version = 0
	}

	for _, group := range cg.Groups {
		zeroVersions(group)
	}
}
