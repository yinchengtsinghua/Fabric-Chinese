
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


package configtx

import (
	"regexp"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("common.configtx")

//有效通道和配置ID的约束
var (
	channelAllowedChars = "[a-z][a-z0-9.-]*"
	configAllowedChars  = "[a-zA-Z0-9.-]+"
	maxLength           = 249
	illegalNames        = map[string]struct{}{
		".":  {},
		"..": {},
	}
)

//validatorimpl实现了validator接口
type ValidatorImpl struct {
	channelID   string
	sequence    uint64
	configMap   map[string]comparable
	configProto *cb.Config
	namespace   string
	pm          policies.Manager
}

//validateconfigid确保配置元素名称（即
//
//
//2。少于250个字符。
//
func validateConfigID(configID string) error {
	re, _ := regexp.Compile(configAllowedChars)
//长度
	if len(configID) <= 0 {
		return errors.New("config ID illegal, cannot be empty")
	}
	if len(configID) > maxLength {
		return errors.Errorf("config ID illegal, cannot be longer than %d", maxLength)
	}
//
	if _, ok := illegalNames[configID]; ok {
		return errors.Errorf("name '%s' for config ID is not allowed", configID)
	}
//非法字符
	matched := re.FindString(configID)
	if len(matched) != len(configID) {
		return errors.Errorf("config ID '%s' contains illegal characters", configID)
	}

	return nil
}

//validatechannelid确保建议的通道ID符合
//以下限制：
//1。只包含小写的ASCII字母数字、点“.”和短划线“-”
//2。少于250个字符。
//三。从一封信开始
//
//这是kafka限制和couchdb限制的交叉点
//出现以下异常：“.”在couchdb命名中转换为“u”
//这是为了适应现有频道名称与“.”，特别是在
//对依赖于点符号的慢化的测试进行操作。
func validateChannelID(channelID string) error {
	re, _ := regexp.Compile(channelAllowedChars)
//长度
	if len(channelID) <= 0 {
		return errors.Errorf("channel ID illegal, cannot be empty")
	}
	if len(channelID) > maxLength {
		return errors.Errorf("channel ID illegal, cannot be longer than %d", maxLength)
	}

//非法字符
	matched := re.FindString(channelID)
	if len(matched) != len(channelID) {
		return errors.Errorf("channel ID '%s' contains illegal characters", channelID)
	}

	return nil
}

//newvalidatorimpl构造了一个新的validator接口实现。
func NewValidatorImpl(channelID string, config *cb.Config, namespace string, pm policies.Manager) (*ValidatorImpl, error) {
	if config == nil {
		return nil, errors.Errorf("nil config parameter")
	}

	if config.ChannelGroup == nil {
		return nil, errors.Errorf("nil channel group")
	}

	if err := validateChannelID(channelID); err != nil {
		return nil, errors.Errorf("bad channel ID: %s", err)
	}

	configMap, err := mapConfig(config.ChannelGroup, namespace)
	if err != nil {
		return nil, errors.Errorf("error converting config to map: %s", err)
	}

	return &ValidatorImpl{
		namespace:   namespace,
		pm:          pm,
		sequence:    config.Sequence,
		configMap:   configMap,
		channelID:   channelID,
		configProto: config,
	}, nil
}

//ProposeConfigUpdate接受config_update类型的信封，并生成
//将用作配置消息的信封有效负载数据的configendevelope
func (vi *ValidatorImpl) ProposeConfigUpdate(configtx *cb.Envelope) (*cb.ConfigEnvelope, error) {
	return vi.proposeConfigUpdate(configtx)
}

func (vi *ValidatorImpl) proposeConfigUpdate(configtx *cb.Envelope) (*cb.ConfigEnvelope, error) {
	configUpdateEnv, err := utils.EnvelopeToConfigUpdate(configtx)
	if err != nil {
		return nil, errors.Errorf("error converting envelope to config update: %s", err)
	}

	configMap, err := vi.authorizeUpdate(configUpdateEnv)
	if err != nil {
		return nil, errors.Errorf("error authorizing update: %s", err)
	}

	channelGroup, err := configMapToConfig(configMap, vi.namespace)
	if err != nil {
		return nil, errors.Errorf("could not turn configMap back to channelGroup: %s", err)
	}

	return &cb.ConfigEnvelope{
		Config: &cb.Config{
			Sequence:     vi.sequence + 1,
			ChannelGroup: channelGroup,
		},
		LastUpdate: configtx,
	}, nil
}

//验证模拟应用configendevelope成为新配置
func (vi *ValidatorImpl) Validate(configEnv *cb.ConfigEnvelope) error {
	if configEnv == nil {
		return errors.Errorf("config envelope is nil")
	}

	if configEnv.Config == nil {
		return errors.Errorf("config envelope has nil config")
	}

	if configEnv.Config.Sequence != vi.sequence+1 {
		return errors.Errorf("config currently at sequence %d, cannot validate config at sequence %d", vi.sequence, configEnv.Config.Sequence)
	}

	configUpdateEnv, err := utils.EnvelopeToConfigUpdate(configEnv.LastUpdate)
	if err != nil {
		return err
	}

	configMap, err := vi.authorizeUpdate(configUpdateEnv)
	if err != nil {
		return err
	}

	channelGroup, err := configMapToConfig(configMap, vi.namespace)
	if err != nil {
		return errors.Errorf("could not turn configMap back to channelGroup: %s", err)
	}

//reflect.equal在这里不起作用，因为它认为nil和空映射不同
	if !proto.Equal(channelGroup, configEnv.Config.ChannelGroup) {
		return errors.Errorf("ConfigEnvelope LastUpdate did not produce the supplied config result")
	}

	return nil
}

//chainID检索与此管理器关联的链ID
func (vi *ValidatorImpl) ChainID() string {
	return vi.channelID
}

//Sequence返回配置的序列号
func (vi *ValidatorImpl) Sequence() uint64 {
	return vi.sequence
}

//config proto返回初始化此验证器的config proto
func (vi *ValidatorImpl) ConfigProto() *cb.Config {
	return vi.configProto
}
