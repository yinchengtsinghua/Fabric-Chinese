
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
	"time"

	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//组织存储公共组织配置
type Org interface {
//name返回这个组织在config中引用的名称
	Name() string

//msp id返回与此组织关联的msp id
	MSPID() string
}

//application org存储每个组织的应用程序配置
type ApplicationOrg interface {
	Org

//主持人返回八卦主持人名单
	AnchorPeers() []*pb.AnchorPeer
}

//应用程序存储公共共享应用程序配置
type Application interface {
//组织将组织ID的映射返回到ApplicationOrg
	Organizations() map[string]ApplicationOrg

//api policymapper返回将api名称映射到策略的policymapper
	APIPolicyMapper() PolicyMapper

//功能定义通道应用程序部分的功能
	Capabilities() ApplicationCapabilities
}

//通道提供对通道配置的只读访问
type Channel interface {
//哈希算法返回哈希时使用的默认算法
//例如计算块散列和创建策略摘要
	HashingAlgorithm() func(input []byte) []byte

//BlockDataHashingStructureWidth返回构造
//计算blockdata散列的merkle树
	BlockDataHashingStructureWidth() uint32

//orderAddresses返回要连接以调用广播/传递的有效订购者地址列表
	OrdererAddresses() []string

//功能定义通道的功能
	Capabilities() ChannelCapabilities
}

//联合体表示由订购服务提供服务的一组联合体。
type Consortiums interface {
//联合体返回一组联合体
	Consortiums() map[string]Consortium
}

//联合体代表一组可以共同创建渠道的组织。
type Consortium interface {
//channelcreationpolicy返回为该联合体实例化通道时要检查的策略
	ChannelCreationPolicy() *cb.Policy

//组织返回此联合体的组织
	Organizations() map[string]Org
}

//排序器存储公共共享排序器配置
type Orderer interface {
//ConsenseStype返回配置的共识类型
	ConsensusType() string

//ConsensusMetadata返回与共识类型关联的元数据。
	ConsensusMetadata() []byte

//batchsize返回块中要包含的最大消息数
	BatchSize() *ab.BatchSize

//batchTimeout返回创建批之前等待的时间量
	BatchTimeout() time.Duration

//maxchannelscount返回允许订购网络使用的最大通道数。
	MaxChannelsCount() uint64

//KafkAbrokers返回一组“引导”的地址（IP:端口表示法）
//卡夫卡经纪人，也就是说，这不一定是卡夫卡经纪人的全部。
//用于订购
	KafkaBrokers() []string

//组织返回订购服务的组织
	Organizations() map[string]Org

//功能定义通道的医嘱者部分的功能
	Capabilities() OrdererCapabilities
}

//
type ChannelCapabilities interface {
//如果此通道中存在所需的未知功能，则SUPPORTED返回错误
	Supported() error

//msp version指定此通道必须了解的msp版本，包括msp类型
//和MSP主体类型。
	MSPVersion() msp.MSPVersion
}

//应用程序功能定义通道应用程序部分的功能
type ApplicationCapabilities interface {
//如果此通道中存在所需的未知功能，则SUPPORTED返回错误
	Supported() error

//ForbidDuplicateXdinBlock指定是否允许两个具有相同TxID的事务
//在同一个块中，或者是否将第二个块标记为txvalidationcode_duplicate_txid
	ForbidDuplicateTXIdInBlock() bool

//acls返回true是否可以在配置树的应用程序部分指定acls
	ACLs() bool

//如果启用了对专用通道数据（也称为集合）的支持，则private channel data返回true。
//在v1.1中，专用通道数据是实验性的，必须显式启用。
//在v1.2中，默认情况下启用专用通道数据。
	PrivateChannelData() bool

//如果将此通道配置为允许更新到
//现有集合或通过chaincode升级添加新集合（如v1.2所述）
	CollectionUpgrade() bool

//v11validation返回true是否将此通道配置为执行更严格的验证
//事务数（如v1.1中介绍的）。
	V1_1Validation() bool

//v12validation返回true是否将此通道配置为执行更严格的验证
//事务数（如v1.2所述）。
	V1_2Validation() bool

//v13validation如果此通道支持事务验证，则返回true
//如v1.3所述。这包括：
//-如FAB-8812所述，以分类帐键粒度表示的政策
//-新的链码生命周期，如FAB-11237所述
	V1_3Validation() bool

//MetadataLifecycle指示对等端是否应使用已弃用和有问题的
//v1.0/v1.1生命周期，或者是否应该使用更新的每通道对等本地链代码
//计划使用Fabric v1.2发布的元数据包方法
	MetadataLifecycle() bool

//如果此通道支持认可，则keyLevelndOrsement返回true
//如FAB-8812所述，以分类帐键粒度表示的策略
	KeyLevelEndorsement() bool

//如果此通道支持fabtoken函数，fabtoken返回true
	FabToken() bool
}

//orderCapabilities定义通道的order部分的功能
type OrdererCapabilities interface {
//predictablechanneltemplate指定v1.0版中设置/channel的不良行为
//组的mod_policy to“”和copy versions from the order system channel config应为fixed或not。
	PredictableChannelTemplate() bool

//重新提交指定是否应通过重新提交来修复Tx的v1.0非确定性承诺
//重新验证的Tx。
	Resubmission() bool

//如果此通道中存在所需的未知功能，则SUPPORTED返回错误
	Supported() error

//ExpirationCheck指定订购方是否检查标识过期检查
//验证消息时
	ExpirationCheck() bool
}

//policyMapper是的接口
type PolicyMapper interface {
//policyRefForAPI使用API的名称，并返回策略名称
//如果找不到API，则返回空字符串
	PolicyRefForAPI(apiName string) string
}

//资源是所有通道的通用配置资源集
//根据是在订购方还是在对等方使用链，其他
//配置资源可能可用
type Resources interface {
//configtx validator返回通道的configtx.validator
	ConfigtxValidator() configtx.Validator

//policyManager返回通道的policies.manager
	PolicyManager() policies.Manager

//channel config返回链的config.channel
	ChannelConfig() Channel

//orderconfig返回通道的config.order
//以及医嘱者配置是否存在
	OrdererConfig() (Orderer, bool)

//consortiums config（）返回通道的config.consortiums
//以及联合体配置是否存在
	ConsortiumsConfig() (Consortiums, bool)

//applicationconfig返回通道的configtxapplication.sharedconfig
//以及应用程序配置是否存在
	ApplicationConfig() (Application, bool)

//mspmanager返回链的msp.mspmanager
	MSPManager() msp.MSPManager

//如果新的配置资源集与当前的配置资源集不兼容，则validateNew应返回一个错误。
	ValidateNew(resources Resources) error
}
