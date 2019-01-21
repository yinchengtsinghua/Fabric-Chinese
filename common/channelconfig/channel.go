
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
	"fmt"
	"math"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/common/capabilities"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

//频道配置键
const (
//联合体密钥是联合体消息的cb.configValue的密钥
	ConsortiumKey = "Consortium"

//hashingalgorithm key是hashingalgorithm消息的cb.configitem类型密钥名称
	HashingAlgorithmKey = "HashingAlgorithm"

//blockdatahashingstructure key是blockdatahashingstructure消息的cb.configitem类型键名称
	BlockDataHashingStructureKey = "BlockDataHashingStructure"

//orderAddresseskey是orderAddresses消息的cb.configitem类型键名称
	OrdererAddressesKey = "OrdererAddresses"

//
	ChannelGroupKey = "Channel"

//capabilities key是指功能的键的名称，它出现在通道中，
//应用程序和订购程序级别以及这个常量用于这三个级别。
	CapabilitiesKey = "Capabilities"
)

//channelValues提供对通道配置的只读访问
type ChannelValues interface {
//哈希算法返回哈希时使用的默认算法
//例如计算块散列和创建策略摘要
	HashingAlgorithm() func(input []byte) []byte

//BlockDataHashingStructureWidth返回构造
//计算blockdata散列的merkle树
	BlockDataHashingStructureWidth() uint32

//orderAddresses返回要连接以调用广播/传递的有效订购者地址列表
	OrdererAddresses() []string
}

//channelprotos是将建议的配置分解为
type ChannelProtos struct {
	HashingAlgorithm          *cb.HashingAlgorithm
	BlockDataHashingStructure *cb.BlockDataHashingStructure
	OrdererAddresses          *cb.OrdererAddresses
	Consortium                *cb.Consortium
	Capabilities              *cb.Capabilities
}

//channelconfig存储通道配置
type ChannelConfig struct {
	protos *ChannelProtos

	hashingAlgorithm func(input []byte) []byte

	mspManager msp.MSPManager

	appConfig         *ApplicationConfig
	ordererConfig     *OrdererConfig
	consortiumsConfig *ConsortiumsConfig
}

//new channelconfig创建新的channelconfig
func NewChannelConfig(channelGroup *cb.ConfigGroup) (*ChannelConfig, error) {
	cc := &ChannelConfig{
		protos: &ChannelProtos{},
	}

	if err := DeserializeProtoValuesFromGroup(channelGroup, cc.protos); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize values")
	}

	if err := cc.Validate(); err != nil {
		return nil, err
	}

	capabilities := cc.Capabilities()
	mspConfigHandler := NewMSPConfigHandler(capabilities.MSPVersion())

	var err error
	for groupName, group := range channelGroup.Groups {
		switch groupName {
		case ApplicationGroupKey:
			cc.appConfig, err = NewApplicationConfig(group, mspConfigHandler)
		case OrdererGroupKey:
			cc.ordererConfig, err = NewOrdererConfig(group, mspConfigHandler)
		case ConsortiumsGroupKey:
			cc.consortiumsConfig, err = NewConsortiumsConfig(group, mspConfigHandler)
		default:
			return nil, fmt.Errorf("Disallowed channel group: %s", group)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "could not create channel %s sub-group config", groupName)
		}
	}

	if cc.mspManager, err = mspConfigHandler.CreateMSPManager(); err != nil {
		return nil, err
	}

	return cc, nil
}

//msp manager返回此配置的msp管理器
func (cc *ChannelConfig) MSPManager() msp.MSPManager {
	return cc.mspManager
}

//orderconfig返回与此通道关联的order配置
func (cc *ChannelConfig) OrdererConfig() *OrdererConfig {
	return cc.ordererConfig
}

//application config返回与此通道关联的应用程序配置
func (cc *ChannelConfig) ApplicationConfig() *ApplicationConfig {
	return cc.appConfig
}

//concertiumsConfig返回与此通道关联的联合体配置（如果存在）
func (cc *ChannelConfig) ConsortiumsConfig() *ConsortiumsConfig {
	return cc.consortiumsConfig
}

//hashingalgorithm返回指向链hashing algorithm的函数指针
func (cc *ChannelConfig) HashingAlgorithm() func(input []byte) []byte {
	return cc.hashingAlgorithm
}

//BlockDataHashingStructure返回形成块数据哈希结构时使用的宽度
func (cc *ChannelConfig) BlockDataHashingStructureWidth() uint32 {
	return cc.protos.BlockDataHashingStructure.Width
}

//orderAddresses返回要连接以调用广播/传递的有效订购者地址列表
func (cc *ChannelConfig) OrdererAddresses() []string {
	return cc.protos.OrdererAddresses.Addresses
}

//ConsortiumName返回创建此通道所用的联合体的名称
func (cc *ChannelConfig) ConsortiumName() string {
	return cc.protos.Consortium.Name
}

//功能返回有关此通道可用功能的信息
func (cc *ChannelConfig) Capabilities() ChannelCapabilities {
	return capabilities.NewChannelProvider(cc.protos.Capabilities.Capabilities)
}

//验证检查生成的配置协议并确保值正确。
func (cc *ChannelConfig) Validate() error {
	for _, validator := range []func() error{
		cc.validateHashingAlgorithm,
		cc.validateBlockDataHashingStructure,
		cc.validateOrdererAddresses,
	} {
		if err := validator(); err != nil {
			return err
		}
	}

	return nil
}

func (cc *ChannelConfig) validateHashingAlgorithm() error {
	switch cc.protos.HashingAlgorithm.Name {
	case bccsp.SHA256:
		cc.hashingAlgorithm = util.ComputeSHA256
	case bccsp.SHA3_256:
		cc.hashingAlgorithm = util.ComputeSHA3256
	default:
		return fmt.Errorf("Unknown hashing algorithm type: %s", cc.protos.HashingAlgorithm.Name)
	}

	return nil
}

func (cc *ChannelConfig) validateBlockDataHashingStructure() error {
	if cc.protos.BlockDataHashingStructure.Width != math.MaxUint32 {
		return fmt.Errorf("BlockDataHashStructure width only supported at MaxUint32 in this version")
	}
	return nil
}

func (cc *ChannelConfig) validateOrdererAddresses() error {
	if len(cc.protos.OrdererAddresses.Addresses) == 0 {
		return fmt.Errorf("Must set some OrdererAddresses")
	}
	return nil
}
