
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
*/


package channelconfig

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/capabilities"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
)

const (
//
	OrdererGroupKey = "Orderer"
)

const (
//consensustype key是consensustype消息的cb.configitem类型密钥名称
	ConsensusTypeKey = "ConsensusType"

//batchsize key是batchsize消息的cb.configitem类型键名称
	BatchSizeKey = "BatchSize"

//batchTimeoutKey是batchTimeout消息的cb.configitem类型键名称
	BatchTimeoutKey = "BatchTimeout"

//ChannelRestrictionsKey是ChannelRestrictions消息的密钥名称
	ChannelRestrictionsKey = "ChannelRestrictions"

//KafkAbrokersKey是KafkAbrokers消息的cb.configitem类型密钥名称
	KafkaBrokersKey = "KafkaBrokers"
)

//OrdererProtos用作OrdererConfig的源
type OrdererProtos struct {
	ConsensusType       *ab.ConsensusType
	BatchSize           *ab.BatchSize
	BatchTimeout        *ab.BatchTimeout
	KafkaBrokers        *ab.KafkaBrokers
	ChannelRestrictions *ab.ChannelRestrictions
	Capabilities        *cb.Capabilities
}

//orderconfig保存订购者配置信息
type OrdererConfig struct {
	protos *OrdererProtos
	orgs   map[string]Org

	batchTimeout time.Duration
}

//neworderconfig创建order配置的新实例
func NewOrdererConfig(ordererGroup *cb.ConfigGroup, mspConfig *MSPConfigHandler) (*OrdererConfig, error) {
	oc := &OrdererConfig{
		protos: &OrdererProtos{},
		orgs:   make(map[string]Org),
	}

	if err := DeserializeProtoValuesFromGroup(ordererGroup, oc.protos); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize values")
	}

	if err := oc.Validate(); err != nil {
		return nil, err
	}

	for orgName, orgGroup := range ordererGroup.Groups {
		var err error
		if oc.orgs[orgName], err = NewOrganizationConfig(orgName, orgGroup, mspConfig); err != nil {
			return nil, err
		}
	}
	return oc, nil
}

//ConsenseStype返回配置的共识类型
func (oc *OrdererConfig) ConsensusType() string {
	return oc.protos.ConsensusType.Type
}

//ConsensusMetadata返回与共识类型关联的元数据。
func (oc *OrdererConfig) ConsensusMetadata() []byte {
	return oc.protos.ConsensusType.Metadata
}

//batchsize返回块中要包含的最大消息数
func (oc *OrdererConfig) BatchSize() *ab.BatchSize {
	return oc.protos.BatchSize
}

//batchTimeout返回创建批之前等待的时间量
func (oc *OrdererConfig) BatchTimeout() time.Duration {
	return oc.batchTimeout
}

//KafkAbrokers返回一组“引导”的地址（IP:端口表示法）
//卡夫卡经纪人，也就是说，这不一定是卡夫卡经纪人的全部。
//用于订购
func (oc *OrdererConfig) KafkaBrokers() []string {
	return oc.protos.KafkaBrokers.Brokers
}

//
func (oc *OrdererConfig) MaxChannelsCount() uint64 {
	return oc.protos.ChannelRestrictions.MaxCount
}

//组织返回渠道中组织的地图
func (oc *OrdererConfig) Organizations() map[string]Org {
	return oc.orgs
}

//功能返回订购网络对此频道的功能
func (oc *OrdererConfig) Capabilities() OrdererCapabilities {
	return capabilities.NewOrdererProvider(oc.protos.Capabilities.Capabilities)
}

func (oc *OrdererConfig) Validate() error {
	for _, validator := range []func() error{
		oc.validateBatchSize,
		oc.validateBatchTimeout,
		oc.validateKafkaBrokers,
	} {
		if err := validator(); err != nil {
			return err
		}
	}

	return nil
}

func (oc *OrdererConfig) validateBatchSize() error {
	if oc.protos.BatchSize.MaxMessageCount == 0 {
		return fmt.Errorf("Attempted to set the batch size max message count to an invalid value: 0")
	}
	if oc.protos.BatchSize.AbsoluteMaxBytes == 0 {
		return fmt.Errorf("Attempted to set the batch size absolute max bytes to an invalid value: 0")
	}
	if oc.protos.BatchSize.PreferredMaxBytes == 0 {
		return fmt.Errorf("Attempted to set the batch size preferred max bytes to an invalid value: 0")
	}
	if oc.protos.BatchSize.PreferredMaxBytes > oc.protos.BatchSize.AbsoluteMaxBytes {
		return fmt.Errorf("Attempted to set the batch size preferred max bytes (%v) greater than the absolute max bytes (%v).", oc.protos.BatchSize.PreferredMaxBytes, oc.protos.BatchSize.AbsoluteMaxBytes)
	}
	return nil
}

func (oc *OrdererConfig) validateBatchTimeout() error {
	var err error
	oc.batchTimeout, err = time.ParseDuration(oc.protos.BatchTimeout.Timeout)
	if err != nil {
		return fmt.Errorf("Attempted to set the batch timeout to a invalid value: %s", err)
	}
	if oc.batchTimeout <= 0 {
		return fmt.Errorf("Attempted to set the batch timeout to a non-positive value: %s", oc.batchTimeout)
	}
	return nil
}

func (oc *OrdererConfig) validateKafkaBrokers() error {
	for _, broker := range oc.protos.KafkaBrokers.Brokers {
		if !brokerEntrySeemsValid(broker) {
			return fmt.Errorf("Invalid broker entry: %s", broker)
		}
	}
	return nil
}

//这只是一个准健全的检查。
func brokerEntrySeemsValid(broker string) bool {
	if !strings.Contains(broker, ":") {
		return false
	}

	parts := strings.Split(broker, ":")
	if len(parts) > 2 {
		return false
	}

	host := parts[0]
	port := parts[1]

	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return false
	}

//有效主机名只能包含ASCII字母“a”到“z”（在
//不区分大小写的方式）、数字“0”到“9”以及连字符。知识产权
//v4地址用点-十进制表示法表示，包括
//四个十进制数字，每个数字的范围从0到255，用点分隔，
//例如，172.16.254.1
//以下正则表达式：
//1。只允许a-z（不区分大小写）、0-9和点和连字符
//2。不允许前导尾随点或连字符
	re, _ := regexp.Compile("^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9])$")
	matched := re.FindString(host)
	return len(matched) == len(host)
}
