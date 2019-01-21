
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


package privdata

import (
	"strings"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//集合定义集合的公共接口
type Collection interface {
//settxContext配置特定于Tx的临时收集信息，例如
//作为txid、nonce、creator--供将来使用
//settxContext（参数…接口）

//collection id返回此集合的ID
	CollectionID() string

//Get背书策略返回用于验证的背书策略--
//未来用益权
//get背书policy（）字符串

//memberOrgs以MSP ID的形式返回集合的成员。这是
//一种人类可读的快速识别谁是集合的一部分的方法。
	MemberOrgs() []string
}

//collection access policy封装集合访问策略的函数
type CollectionAccessPolicy interface {
//accessfilter返回集合的成员筛选器函数
	AccessFilter() Filter

//将发送到的对等机私有数据的最小数目
//背书。如果至少传播到
//没有达到这一数量的同龄人。
	RequiredPeerCount() int

//将私人数据发送到的对等机的最大数目
//背书后。此数字必须大于RequiredPeerCount（）。
	MaximumPeerCount() int

//memberOrgs以MSP ID的形式返回集合的成员。这是
//一种人类可读的快速识别谁是集合的一部分的方法。
	MemberOrgs() []string

//IsMemberOnlyLead如果只有集合成员可以读取，则返回true
//私人数据
	IsMemberOnlyRead() bool
}

//CollectionPersistenceConfigs封装与集合的Persistece相关的配置
type CollectionPersistenceConfigs interface {
//BlockToLive返回收集数据过期后的块数。
//例如，如果该值设置为10，则最后由块编号100修改的键
//将在111号块处清除。零值的处理方式与maxuint64相同。
	BlockToLive() uint64
}

//筛选器定义一个规则，根据由其签名的数据筛选对等方。
//SignedData中的标识是对等机的序列化实体。
//数据是对等签名的消息，签名是对应的
//在数据上签名。
//如果策略保留给定的签名数据，则返回：true。
//否则假
type Filter func(common.SignedData) bool

//CollectionStore提供各种API来检索存储的集合并执行
//基于集合属性的成员资格检查和读取权限检查。
//TODO:重构集合存储-FAB-13082
//（1）RetrieveCollection（）和RetrieveCollectionConfigPackage（）等函数是
//除非在模拟和测试文件中，否则不得使用。
//（2）在八卦中，至少在7个不同的地方，以下3个操作
//可以通过引入名为isamberof（）的API来避免重复。
//（i）通过调用RetrieveCollectionAccessPolicy（）检索集合访问策略
//（ii）从收集访问策略中获取访问筛选器func
//（三）制定评估政策，检查会员资格
//（3）我们需要在收集存储中有一个缓存，以避免重复的加密操作。
//当我们引入isamberof（）API时，这很容易实现。
type CollectionStore interface {
//getCollection按以下方式检索集合：
//如果txid存在于分类帐中，则返回的集合具有
//在此txid之前提交到分类帐的最新配置
//承诺。
//Else - it's the latest configuration for the collection.
	RetrieveCollection(common.CollectionCriteria) (Collection, error)

//getCollectionAccessPolicy检索集合的访问策略
	RetrieveCollectionAccessPolicy(common.CollectionCriteria) (CollectionAccessPolicy, error)

//RetrieveCollectionConfigPackage检索整个配置包
//对于具有提供标准的链码
	RetrieveCollectionConfigPackage(common.CollectionCriteria) (*common.CollectionConfigPackage, error)

//RetrieveCollectionPersistenceConfigs检索集合的与持久性相关的配置
	RetrieveCollectionPersistenceConfigs(common.CollectionCriteria) (CollectionPersistenceConfigs, error)

//HasReadAccess检查已签名的Proposal的创建者是否对
//给定集合
	HasReadAccess(common.CollectionCriteria, *pb.SignedProposal, ledger.QueryExecutor) (bool, error)

	CollectionFilter
}

type CollectionFilter interface {
//AccessFilter检索与给定通道和CollectionPolicyConfig匹配的集合筛选器
	AccessFilter(channelName string, collectionPolicyConfig *common.CollectionPolicyConfig) (Filter, error)
}

const (
//Collecion特定常数

//collectionseparator是用于构建kvs的分隔符
//存储链码集合的键；请注意
//用作分隔符的字符对于
//链码的名称或版本，因此不能有任何
//选择名称时发生冲突
	collectionSeparator = "~"
//collectionsuffix是存储
//链式码的集合
	collectionSuffix = "collection"
)

//buildCollectionKvsky为给定的链码名称构造集合配置键
func BuildCollectionKVSKey(ccname string) string {
	return ccname + collectionSeparator + collectionSuffix
}

//iscollectionconfigkey检测密钥是否是集合密钥
func IsCollectionConfigKey(key string) bool {
	return strings.Contains(key, collectionSeparator)
}

//getccnamefromcollectionconfigkey返回给定集合配置键的链码名称
func GetCCNameFromCollectionConfigKey(key string) string {
	splittedKey := strings.Split(key, collectionSeparator)
	return splittedKey[0]
}
