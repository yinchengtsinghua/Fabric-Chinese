
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
	"math"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	cb "github.com/hyperledger/fabric/protos/common"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
	ab "github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"
)

const (
//readerspolicykey是用于读取策略的密钥
	ReadersPolicyKey = "Readers"

//WritersPolicyKey是用于读取策略的密钥
	WritersPolicyKey = "Writers"

//adminspolicykey是用于读取策略的密钥
	AdminsPolicyKey = "Admins"

	defaultHashingAlgorithm = bccsp.SHA256

	defaultBlockDataHashingStructureWidth = math.MaxUint32
)

//configvalue为不同的*cb.configvalue值定义了一个公共表示。
type ConfigValue interface {
//key是键，该值应存储在*cb.configggroup.values映射中。
	Key() string

//value是应为*cb.configvalue.value封送到不透明字节的消息。
	Value() proto.Message
}

//StandardConfigValue实现ConfigValue接口。
type StandardConfigValue struct {
	key   string
	value proto.Message
}

//key是键，该值应存储在*cb.configggroup.values映射中。
func (scv *StandardConfigValue) Key() string {
	return scv.key
}

//value是应为*cb.configvalue.value封送到不透明字节的消息。
func (scv *StandardConfigValue) Value() proto.Message {
	return scv.value
}

//联合体值返回联合体名称的配置定义。
//它是通道组的值。
func ConsortiumValue(name string) *StandardConfigValue {
	return &StandardConfigValue{
		key: ConsortiumKey,
		value: &cb.Consortium{
			Name: name,
		},
	}
}

//hashing algorithm返回当前唯一有效的哈希算法。
//它是/通道组的值。
func HashingAlgorithmValue() *StandardConfigValue {
	return &StandardConfigValue{
		key: HashingAlgorithmKey,
		value: &cb.HashingAlgorithm{
			Name: defaultHashingAlgorithm,
		},
	}
}

//BlockDataHashingStructureValue返回当前唯一有效的块数据哈希结构。
//它是/通道组的值。
func BlockDataHashingStructureValue() *StandardConfigValue {
	return &StandardConfigValue{
		key: BlockDataHashingStructureKey,
		value: &cb.BlockDataHashingStructure{
			Width: defaultBlockDataHashingStructureWidth,
		},
	}
}

//orderAddressesValue返回排序器地址的配置定义。
//它是/通道组的值。
func OrdererAddressesValue(addresses []string) *StandardConfigValue {
	return &StandardConfigValue{
		key: OrdererAddressesKey,
		value: &cb.OrdererAddresses{
			Addresses: addresses,
		},
	}
}

//consensustypevalue返回医嘱者共识类型的配置定义。
//它是/channel/order组的值。
func ConsensusTypeValue(consensusType string, consensusMetadata []byte) *StandardConfigValue {
	return &StandardConfigValue{
		key: ConsensusTypeKey,
		value: &ab.ConsensusType{
			Type:     consensusType,
			Metadata: consensusMetadata,
		},
	}
}

//batchSizeValue返回医嘱者批次大小的配置定义。
//它是/channel/order组的值。
func BatchSizeValue(maxMessages, absoluteMaxBytes, preferredMaxBytes uint32) *StandardConfigValue {
	return &StandardConfigValue{
		key: BatchSizeKey,
		value: &ab.BatchSize{
			MaxMessageCount:   maxMessages,
			AbsoluteMaxBytes:  absoluteMaxBytes,
			PreferredMaxBytes: preferredMaxBytes,
		},
	}
}

//batchTimeoutValue返回医嘱者批处理超时的配置定义。
//它是/channel/order组的值。
func BatchTimeoutValue(timeout string) *StandardConfigValue {
	return &StandardConfigValue{
		key: BatchTimeoutKey,
		value: &ab.BatchTimeout{
			Timeout: timeout,
		},
	}
}

//channelRestrictionsValue返回医嘱者通道限制的配置定义。
//它是/channel/order组的值。
func ChannelRestrictionsValue(maxChannelCount uint64) *StandardConfigValue {
	return &StandardConfigValue{
		key: ChannelRestrictionsKey,
		value: &ab.ChannelRestrictions{
			MaxCount: maxChannelCount,
		},
	}
}

//kafkabrokersvalue返回订购服务的kafka代理地址的配置定义。
//它是/channel/order组的值。
func KafkaBrokersValue(brokers []string) *StandardConfigValue {
	return &StandardConfigValue{
		key: KafkaBrokersKey,
		value: &ab.KafkaBrokers{
			Brokers: brokers,
		},
	}
}

//mspvalue返回msp的配置定义。
/*它是/channel/order/*、/channel/application/*和/channel/consortiums/*/*组的值。
func mspvalue（mspdef*mspprotos.mspconfig）*标准配置值
 返回&标准配置值
  MSPKey：
  值：MSPDEF，
 }
}

//capabilitiesValue返回一组功能的配置定义。
//它是/channel/order、channel/application/和/channel组的值。
func capabilities值（capabilities map[string]bool）*标准配置值
 C：=&CB.能力
  能力：制造（map[string]*cb.capability）
 }

 对于能力，要求：=范围能力
  如果！要求{
   持续
  }
  c.能力[能力]=和cb.能力
 }

 返回&标准配置值
  关键：能力，
  值：C，
 }
}

//anchorpeersvalue返回组织的锚对等的配置定义。
//它是/channel/application/*的值。
func anchorpeersvalue（anchorpeers[]*pb.anchorpeer）*standardconfigvalue_
 返回&标准配置值
  密钥：AnchorPeersky，
  值：&pb.anchorpeers anchorpeers:anchorpeers，
 }
}

//channelCreationPolicyValue返回联合体的频道创建策略的配置定义
//它是/channel/联合体的值*/*.

func ChannelCreationPolicyValue(policy *cb.Policy) *StandardConfigValue {
	return &StandardConfigValue{
		key:   ChannelCreationPolicyKey,
		value: policy,
	}
}

//aclsvalues返回基于资源的应用程序acl定义的配置定义。
//它是/channel/application/的值。
func ACLValues(acls map[string]string) *StandardConfigValue {
	a := &pb.ACLs{
		Acls: make(map[string]*pb.APIResource),
	}

	for apiResource, policyRef := range acls {
		a.Acls[apiResource] = &pb.APIResource{PolicyRef: policyRef}
	}

	return &StandardConfigValue{
		key:   ACLsKey,
		value: a,
	}
}
