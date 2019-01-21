
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

SPDX许可证标识符：Apache-2.0
*/


package encoder

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/genesis"
	"github.com/hyperledger/fabric/common/policies"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/common/tools/configtxlator/update"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

const (
	ordererAdminsPolicyName = "/Channel/Orderer/Admins"

	msgVersion = int32(0)
	epoch      = 0
)

var logger = flogging.MustGetLogger("common.tools.configtxgen.encoder")

const (
//Consensistypesolo确定了单独共识的实施。
	ConsensusTypeSolo = "solo"
//一致同意卡夫卡确定了基于卡夫卡的共识实施。
	ConsensusTypeKafka = "kafka"

//
	BlockValidationPolicyKey = "BlockValidation"

//orderAdminPolicy是医嘱管理策略的绝对路径
	OrdererAdminsPolicy = "/Channel/Orderer/Admins"

//SignaturePolicyType是签名策略的“类型”字符串
	SignaturePolicyType = "Signature"

//implicitMetapolicyType是隐式元策略的“type”字符串
	ImplicitMetaPolicyType = "ImplicitMeta"
)

func addValue(cg *cb.ConfigGroup, value channelconfig.ConfigValue, modPolicy string) {
	cg.Values[value.Key()] = &cb.ConfigValue{
		Value:     utils.MarshalOrPanic(value.Value()),
		ModPolicy: modPolicy,
	}
}

func addPolicy(cg *cb.ConfigGroup, policy policies.ConfigPolicy, modPolicy string) {
	cg.Policies[policy.Key()] = &cb.ConfigPolicy{
		Policy:    policy.Value(),
		ModPolicy: modPolicy,
	}
}

func addPolicies(cg *cb.ConfigGroup, policyMap map[string]*genesisconfig.Policy, modPolicy string) error {
	for policyName, policy := range policyMap {
		switch policy.Type {
		case ImplicitMetaPolicyType:
			imp, err := policies.ImplicitMetaFromString(policy.Rule)
			if err != nil {
				return errors.Wrapf(err, "invalid implicit meta policy rule '%s'", policy.Rule)
			}
			cg.Policies[policyName] = &cb.ConfigPolicy{
				ModPolicy: modPolicy,
				Policy: &cb.Policy{
					Type:  int32(cb.Policy_IMPLICIT_META),
					Value: utils.MarshalOrPanic(imp),
				},
			}
		case SignaturePolicyType:
			sp, err := cauthdsl.FromString(policy.Rule)
			if err != nil {
				return errors.Wrapf(err, "invalid signature policy rule '%s'", policy.Rule)
			}
			cg.Policies[policyName] = &cb.ConfigPolicy{
				ModPolicy: modPolicy,
				Policy: &cb.Policy{
					Type:  int32(cb.Policy_SIGNATURE),
					Value: utils.MarshalOrPanic(sp),
				},
			}
		default:
			return errors.Errorf("unknown policy type: %s", policy.Type)
		}
	}
	return nil
}

//addimplicitmetapolicyDefaults使用任意/任意/多数规则分别添加读卡器/编写器/管理员策略。
func addImplicitMetaPolicyDefaults(cg *cb.ConfigGroup) {
	addPolicy(cg, policies.ImplicitMetaMajorityPolicy(channelconfig.AdminsPolicyKey), channelconfig.AdminsPolicyKey)
	addPolicy(cg, policies.ImplicitMetaAnyPolicy(channelconfig.ReadersPolicyKey), channelconfig.AdminsPolicyKey)
	addPolicy(cg, policies.ImplicitMetaAnyPolicy(channelconfig.WritersPolicyKey), channelconfig.AdminsPolicyKey)
}

//
//如果devmode设置为true，则admins策略将接受管理函数的任意用户证书，否则它需要满足证书
//
func addSignaturePolicyDefaults(cg *cb.ConfigGroup, mspID string, devMode bool) {
	if devMode {
		logger.Warningf("Specifying AdminPrincipal is deprecated and will be removed in a future release, override the admin principal with explicit policies.")
		addPolicy(cg, policies.SignaturePolicy(channelconfig.AdminsPolicyKey, cauthdsl.SignedByMspMember(mspID)), channelconfig.AdminsPolicyKey)
	} else {
		addPolicy(cg, policies.SignaturePolicy(channelconfig.AdminsPolicyKey, cauthdsl.SignedByMspAdmin(mspID)), channelconfig.AdminsPolicyKey)
	}
	addPolicy(cg, policies.SignaturePolicy(channelconfig.ReadersPolicyKey, cauthdsl.SignedByMspMember(mspID)), channelconfig.AdminsPolicyKey)
	addPolicy(cg, policies.SignaturePolicy(channelconfig.WritersPolicyKey, cauthdsl.SignedByMspMember(mspID)), channelconfig.AdminsPolicyKey)
}

//
//
//
//配置。此组的所有mod_policy值都设置为“admins”，但orderAddresses除外。
//
func NewChannelGroup(conf *genesisconfig.Profile) (*cb.ConfigGroup, error) {
	if conf.Orderer == nil {
		return nil, errors.New("missing orderer config section")
	}

	channelGroup := cb.NewConfigGroup()
	if len(conf.Policies) == 0 {
		logger.Warningf("Default policy emission is deprecated, please include policy specifications for the channel group in configtx.yaml")
		addImplicitMetaPolicyDefaults(channelGroup)
	} else {
		if err := addPolicies(channelGroup, conf.Policies, channelconfig.AdminsPolicyKey); err != nil {
			return nil, errors.Wrapf(err, "error adding policies to channel group")
		}
	}

	addValue(channelGroup, channelconfig.HashingAlgorithmValue(), channelconfig.AdminsPolicyKey)
	addValue(channelGroup, channelconfig.BlockDataHashingStructureValue(), channelconfig.AdminsPolicyKey)
	addValue(channelGroup, channelconfig.OrdererAddressesValue(conf.Orderer.Addresses), ordererAdminsPolicyName)

	if conf.Consortium != "" {
		addValue(channelGroup, channelconfig.ConsortiumValue(conf.Consortium), channelconfig.AdminsPolicyKey)
	}

	if len(conf.Capabilities) > 0 {
		addValue(channelGroup, channelconfig.CapabilitiesValue(conf.Capabilities), channelconfig.AdminsPolicyKey)
	}

	var err error
	channelGroup.Groups[channelconfig.OrdererGroupKey], err = NewOrdererGroup(conf.Orderer)
	if err != nil {
		return nil, errors.Wrap(err, "could not create orderer group")
	}

	if conf.Application != nil {
		channelGroup.Groups[channelconfig.ApplicationGroupKey], err = NewApplicationGroup(conf.Application)
		if err != nil {
			return nil, errors.Wrap(err, "could not create application group")
		}
	}

	if conf.Consortiums != nil {
		channelGroup.Groups[channelconfig.ConsortiumsGroupKey], err = NewConsortiumsGroup(conf.Consortiums)
		if err != nil {
			return nil, errors.Wrap(err, "could not create consortiums group")
		}
	}

	channelGroup.ModPolicy = channelconfig.AdminsPolicyKey
	return channelGroup, nil
}

//
//关于应该有多大的数据块，发送的频率，等等，以及订购网络的组织。
//它将所有元素的mod_策略设置为“admins”。此组始终存在于任何通道配置中。
func NewOrdererGroup(conf *genesisconfig.Orderer) (*cb.ConfigGroup, error) {
	ordererGroup := cb.NewConfigGroup()
	if len(conf.Policies) == 0 {
		logger.Warningf("Default policy emission is deprecated, please include policy specifications for the orderer group in configtx.yaml")
		addImplicitMetaPolicyDefaults(ordererGroup)
	} else {
		if err := addPolicies(ordererGroup, conf.Policies, channelconfig.AdminsPolicyKey); err != nil {
			return nil, errors.Wrapf(err, "error adding policies to orderer group")
		}
	}
	ordererGroup.Policies[BlockValidationPolicyKey] = &cb.ConfigPolicy{
		Policy:    policies.ImplicitMetaAnyPolicy(channelconfig.WritersPolicyKey).Value(),
		ModPolicy: channelconfig.AdminsPolicyKey,
	}
	addValue(ordererGroup, channelconfig.BatchSizeValue(
		conf.BatchSize.MaxMessageCount,
		conf.BatchSize.AbsoluteMaxBytes,
		conf.BatchSize.PreferredMaxBytes,
	), channelconfig.AdminsPolicyKey)
	addValue(ordererGroup, channelconfig.BatchTimeoutValue(conf.BatchTimeout.String()), channelconfig.AdminsPolicyKey)
	addValue(ordererGroup, channelconfig.ChannelRestrictionsValue(conf.MaxChannels), channelconfig.AdminsPolicyKey)

	if len(conf.Capabilities) > 0 {
		addValue(ordererGroup, channelconfig.CapabilitiesValue(conf.Capabilities), channelconfig.AdminsPolicyKey)
	}

	var consensusMetadata []byte
	var err error

	switch conf.OrdererType {
	case ConsensusTypeSolo:
	case ConsensusTypeKafka:
		addValue(ordererGroup, channelconfig.KafkaBrokersValue(conf.Kafka.Brokers), channelconfig.AdminsPolicyKey)
	case etcdraft.TypeKey:
		if consensusMetadata, err = etcdraft.Marshal(conf.EtcdRaft); err != nil {
			return nil, errors.Errorf("cannot marshal metadata for orderer type %s: %s", etcdraft.TypeKey, err)
		}
	default:
		return nil, errors.Errorf("unknown orderer type: %s", conf.OrdererType)
	}

	addValue(ordererGroup, channelconfig.ConsensusTypeValue(conf.OrdererType, consensusMetadata), channelconfig.AdminsPolicyKey)

	for _, org := range conf.Organizations {
		var err error
		ordererGroup.Groups[org.Name], err = NewOrdererOrgGroup(org)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create orderer org")
		}
	}

	ordererGroup.ModPolicy = channelconfig.AdminsPolicyKey
	return ordererGroup, nil
}

//neworderorgggroup返回通道配置的order org组件。它定义了
//组织（其MSP）。它将所有元素的mod_策略设置为“admins”。
func NewOrdererOrgGroup(conf *genesisconfig.Organization) (*cb.ConfigGroup, error) {
	mspConfig, err := msp.GetVerifyingMspConfig(conf.MSPDir, conf.ID, conf.MSPType)
	if err != nil {
		return nil, errors.Wrapf(err, "1 - Error loading MSP configuration for org: %s", conf.Name)
	}

	ordererOrgGroup := cb.NewConfigGroup()
	if len(conf.Policies) == 0 {
		logger.Warningf("Default policy emission is deprecated, please include policy specifications for the orderer org group %s in configtx.yaml", conf.Name)
		addSignaturePolicyDefaults(ordererOrgGroup, conf.ID, conf.AdminPrincipal != genesisconfig.AdminRoleAdminPrincipal)
	} else {
		if err := addPolicies(ordererOrgGroup, conf.Policies, channelconfig.AdminsPolicyKey); err != nil {
			return nil, errors.Wrapf(err, "error adding policies to orderer org group '%s'", conf.Name)
		}
	}

	addValue(ordererOrgGroup, channelconfig.MSPValue(mspConfig), channelconfig.AdminsPolicyKey)

	ordererOrgGroup.ModPolicy = channelconfig.AdminsPolicyKey
	return ordererOrgGroup, nil
}

//newApplicationGroup返回通道配置的应用程序组件。它定义了涉及的组织
//
func NewApplicationGroup(conf *genesisconfig.Application) (*cb.ConfigGroup, error) {
	applicationGroup := cb.NewConfigGroup()
	if len(conf.Policies) == 0 {
		logger.Warningf("Default policy emission is deprecated, please include policy specifications for the application group in configtx.yaml")
		addImplicitMetaPolicyDefaults(applicationGroup)
	} else {
		if err := addPolicies(applicationGroup, conf.Policies, channelconfig.AdminsPolicyKey); err != nil {
			return nil, errors.Wrapf(err, "error adding policies to application group")
		}
	}

	if len(conf.ACLs) > 0 {
		addValue(applicationGroup, channelconfig.ACLValues(conf.ACLs), channelconfig.AdminsPolicyKey)
	}

	if len(conf.Capabilities) > 0 {
		addValue(applicationGroup, channelconfig.CapabilitiesValue(conf.Capabilities), channelconfig.AdminsPolicyKey)
	}

	for _, org := range conf.Organizations {
		var err error
		applicationGroup.Groups[org.Name], err = NewApplicationOrgGroup(org)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create application org")
		}
	}

	applicationGroup.ModPolicy = channelconfig.AdminsPolicyKey
	return applicationGroup, nil
}

//
//（它的MSP）以及供八卦网络使用的锚定对等点。它将所有元素的mod_策略设置为“admins”。
func NewApplicationOrgGroup(conf *genesisconfig.Organization) (*cb.ConfigGroup, error) {
	mspConfig, err := msp.GetVerifyingMspConfig(conf.MSPDir, conf.ID, conf.MSPType)
	if err != nil {
		return nil, errors.Wrapf(err, "1 - Error loading MSP configuration for org %s", conf.Name)
	}

	applicationOrgGroup := cb.NewConfigGroup()
	if len(conf.Policies) == 0 {
		logger.Warningf("Default policy emission is deprecated, please include policy specifications for the application org group %s in configtx.yaml", conf.Name)
		addSignaturePolicyDefaults(applicationOrgGroup, conf.ID, conf.AdminPrincipal != genesisconfig.AdminRoleAdminPrincipal)
	} else {
		if err := addPolicies(applicationOrgGroup, conf.Policies, channelconfig.AdminsPolicyKey); err != nil {
			return nil, errors.Wrapf(err, "error adding policies to application org group %s", conf.Name)
		}
	}
	addValue(applicationOrgGroup, channelconfig.MSPValue(mspConfig), channelconfig.AdminsPolicyKey)

	var anchorProtos []*pb.AnchorPeer
	for _, anchorPeer := range conf.AnchorPeers {
		anchorProtos = append(anchorProtos, &pb.AnchorPeer{
			Host: anchorPeer.Host,
			Port: int32(anchorPeer.Port),
		})
	}
	addValue(applicationOrgGroup, channelconfig.AnchorPeersValue(anchorProtos), channelconfig.AdminsPolicyKey)

	applicationOrgGroup.ModPolicy = channelconfig.AdminsPolicyKey
	return applicationOrgGroup, nil
}

//newconsortiumsgroup返回通道配置的联合组件。此元素仅为订购系统通道定义。
//它将所有元素的mod_策略设置为“/channel/order/admins”。
func NewConsortiumsGroup(conf map[string]*genesisconfig.Consortium) (*cb.ConfigGroup, error) {
	consortiumsGroup := cb.NewConfigGroup()
//此策略未在任何地方引用，它仅用作通道级别的隐式元策略规则的一部分，因此此设置
//有效地降低了订购系统对订购管理员的通道控制
	addPolicy(consortiumsGroup, policies.SignaturePolicy(channelconfig.AdminsPolicyKey, cauthdsl.AcceptAllPolicy), ordererAdminsPolicyName)

	for consortiumName, consortium := range conf {
		var err error
		consortiumsGroup.Groups[consortiumName], err = NewConsortiumGroup(consortium)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create consortium %s", consortiumName)
		}
	}

	consortiumsGroup.ModPolicy = ordererAdminsPolicyName
	return consortiumsGroup, nil
}

//newconsortiums返回通道配置的联合组件。每个联合体定义可能参与渠道的组织
//创建，以及订购方在创建通道时检查的通道创建策略，以授权操作。它设定了所有人的国防政策
//元素到“/channel/order/admins”。
func NewConsortiumGroup(conf *genesisconfig.Consortium) (*cb.ConfigGroup, error) {
	consortiumGroup := cb.NewConfigGroup()

	for _, org := range conf.Organizations {
		var err error
//注意，neworderorgggroup在这里是正确的，因为结构是相同的
		consortiumGroup.Groups[org.Name], err = NewOrdererOrgGroup(org)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create consortium org")
		}
	}

	addValue(consortiumGroup, channelconfig.ChannelCreationPolicyValue(policies.ImplicitMetaAnyPolicy(channelconfig.AdminsPolicyKey).Value()), ordererAdminsPolicyName)

	consortiumGroup.ModPolicy = ordererAdminsPolicyName
	return consortiumGroup, nil
}

//NewChannelCreateConfigUpdate生成一个ConfigUpdate，可以将其发送给订购方以创建新的通道。（可选）的通道组
//可能会传入排序系统通道，生成的configupdate将从此文件中提取适当的版本。
func NewChannelCreateConfigUpdate(channelID string, conf *genesisconfig.Profile) (*cb.ConfigUpdate, error) {
	if conf.Application == nil {
		return nil, errors.New("cannot define a new channel with no Application section")
	}

	if conf.Consortium == "" {
		return nil, errors.New("cannot define a new channel with no Consortium value")
	}

//只解析应用程序部分，并将其封装到通道组中
	ag, err := NewApplicationGroup(conf.Application)
	if err != nil {
		return nil, errors.Wrapf(err, "could not turn channel application profile into application group")
	}

	var template, newChannelGroup *cb.ConfigGroup

	newChannelGroup = &cb.ConfigGroup{
		Groups: map[string]*cb.ConfigGroup{
			channelconfig.ApplicationGroupKey: ag,
		},
	}

//假设组织未被修改
	template = proto.Clone(newChannelGroup).(*cb.ConfigGroup)
	template.Groups[channelconfig.ApplicationGroupKey].Values = nil
	template.Groups[channelconfig.ApplicationGroupKey].Policies = nil

	updt, err := update.Compute(&cb.Config{ChannelGroup: template}, &cb.Config{ChannelGroup: newChannelGroup})
	if err != nil {
		return nil, errors.Wrapf(err, "could not compute update")
	}

//根据需要添加联合体名称以将的通道创建到写入集中。
	updt.ChannelId = channelID
	updt.ReadSet.Values[channelconfig.ConsortiumKey] = &cb.ConfigValue{Version: 0}
	updt.WriteSet.Values[channelconfig.ConsortiumKey] = &cb.ConfigValue{
		Version: 0,
		Value: utils.MarshalOrPanic(&cb.Consortium{
			Name: conf.Consortium,
		}),
	}

	return updt, nil
}

//MakeChannelCreationTransaction是一个方便的实用程序函数，用于创建用于创建通道的事务。
func MakeChannelCreationTransaction(channelID string, signer crypto.LocalSigner, conf *genesisconfig.Profile) (*cb.Envelope, error) {
	newChannelConfigUpdate, err := NewChannelCreateConfigUpdate(channelID, conf)
	if err != nil {
		return nil, errors.Wrap(err, "config update generation failure")
	}

	newConfigUpdateEnv := &cb.ConfigUpdateEnvelope{
		ConfigUpdate: utils.MarshalOrPanic(newChannelConfigUpdate),
	}

	if signer != nil {
		sigHeader, err := signer.NewSignatureHeader()
		if err != nil {
			return nil, errors.Wrap(err, "creating signature header failed")
		}

		newConfigUpdateEnv.Signatures = []*cb.ConfigSignature{{
			SignatureHeader: utils.MarshalOrPanic(sigHeader),
		}}

		newConfigUpdateEnv.Signatures[0].Signature, err = signer.Sign(util.ConcatenateBytes(newConfigUpdateEnv.Signatures[0].SignatureHeader, newConfigUpdateEnv.ConfigUpdate))
		if err != nil {
			return nil, errors.Wrap(err, "signature failure over config update")
		}

	}

	return utils.CreateSignedEnvelope(cb.HeaderType_CONFIG_UPDATE, channelID, signer, newConfigUpdateEnv, msgVersion, epoch)
}

//
type Bootstrapper struct {
	channelGroup *cb.ConfigGroup
}

//new为生成genesis块创建一个新的引导程序
func New(config *genesisconfig.Profile) *Bootstrapper {
	channelGroup, err := NewChannelGroup(config)
	if err != nil {
		logger.Panicf("Error creating channel group: %s", err)
	}
	return &Bootstrapper{
		channelGroup: channelGroup,
	}
}

//genesis block为默认测试链ID生成一个genesis块
func (bs *Bootstrapper) GenesisBlock() *cb.Block {
	block, err := genesis.NewFactoryImpl(bs.channelGroup).Block(genesisconfig.TestChainID)
	if err != nil {
		logger.Panicf("Error creating genesis block from channel group: %s", err)
	}
	return block
}

//genesis block for channel为给定的通道ID生成一个genesis块
func (bs *Bootstrapper) GenesisBlockForChannel(channelID string) *cb.Block {
	block, err := genesis.NewFactoryImpl(bs.channelGroup).Block(channelID)
	if err != nil {
		logger.Panicf("Error creating genesis block from channel group: %s", err)
	}
	return block
}
