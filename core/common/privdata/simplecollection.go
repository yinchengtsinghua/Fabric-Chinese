
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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	m "github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

//SimpleCollection实现具有静态属性的集合
//和一个公共成员集
type SimpleCollection struct {
	name         string
	accessPolicy policies.Policy
	memberOrgs   []string
	conf         common.StaticCollectionConfig
}

type SimpleCollectionPersistenceConfigs struct {
	blockToLive uint64
}

//collection id返回集合的ID
func (sc *SimpleCollection) CollectionID() string {
	return sc.name
}

//memberOrgs返回属于此集合的MSP ID
func (sc *SimpleCollection) MemberOrgs() []string {
	return sc.memberOrgs
}

//RequiredPeerCount返回最小对等数
//需要将私人数据发送到
func (sc *SimpleCollection) RequiredPeerCount() int {
	return int(sc.conf.RequiredPeerCount)
}

func (sc *SimpleCollection) MaximumPeerCount() int {
	return int(sc.conf.MaximumPeerCount)
}

//accessfilter返回计算签名数据的成员筛选器函数
//根据此集合的成员访问策略
func (sc *SimpleCollection) AccessFilter() Filter {
	return func(sd common.SignedData) bool {
		if err := sc.accessPolicy.Evaluate([]*common.SignedData{&sd}); err != nil {
			return false
		}
		return true
	}
}

func (sc *SimpleCollection) IsMemberOnlyRead() bool {
	return sc.conf.MemberOnlyRead
}

//安装程序基于给定的
//具有所有必要信息的StaticCollectionConfig协议
func (sc *SimpleCollection) Setup(collectionConfig *common.StaticCollectionConfig, deserializer msp.IdentityDeserializer) error {
	if collectionConfig == nil {
		return errors.New("Nil config passed to collection setup")
	}
	sc.conf = *collectionConfig
	sc.name = collectionConfig.GetName()

//获取访问签名策略信封
	collectionPolicyConfig := collectionConfig.GetMemberOrgsPolicy()
	if collectionPolicyConfig == nil {
		return errors.New("Collection config policy is nil")
	}
	accessPolicyEnvelope := collectionPolicyConfig.GetSignaturePolicy()
	if accessPolicyEnvelope == nil {
		return errors.New("Collection config access policy is nil")
	}

	err := sc.setupAccessPolicy(collectionPolicyConfig, deserializer)
	if err != nil {
		return err
	}

//get member org MSP IDs from the envelope
	for _, principal := range accessPolicyEnvelope.Identities {
		switch principal.PrincipalClassification {
		case m.MSPPrincipal_ROLE:
//主体包含MSP角色
			mspRole := &m.MSPRole{}
			err := proto.Unmarshal(principal.Principal, mspRole)
			if err != nil {
				return errors.Wrap(err, "Could not unmarshal MSPRole from principal")
			}
			sc.memberOrgs = append(sc.memberOrgs, mspRole.MspIdentifier)
		case m.MSPPrincipal_IDENTITY:
			principalId, err := deserializer.DeserializeIdentity(principal.Principal)
			if err != nil {
				return errors.Wrap(err, "Invalid identity principal, not a certificate")
			}
			sc.memberOrgs = append(sc.memberOrgs, principalId.GetMSPIdentifier())
		case m.MSPPrincipal_ORGANIZATION_UNIT:
			OU := &m.OrganizationUnit{}
			err := proto.Unmarshal(principal.Principal, OU)
			if err != nil {
				return errors.Wrap(err, "Could not unmarshal OrganizationUnit from principal")
			}
			sc.memberOrgs = append(sc.memberOrgs, OU.MspIdentifier)
		default:
			return errors.New(fmt.Sprintf("Invalid principal type %d", int32(principal.PrincipalClassification)))
		}
	}

	return nil
}

//安装程序基于给定的
//具有所有必要信息的StaticCollectionConfig协议
func (sc *SimpleCollection) setupAccessPolicy(collectionPolicyConfig *common.CollectionPolicyConfig, deserializer msp.IdentityDeserializer) error {
	var err error
	sc.accessPolicy, err = getPolicy(collectionPolicyConfig, deserializer)
	return err
}

//BlockToLive返回集合的块到活动配置
func (s *SimpleCollectionPersistenceConfigs) BlockToLive() uint64 {
	return s.blockToLive
}
