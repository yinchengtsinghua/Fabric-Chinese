
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


package aclmgmt

import (
	"fmt"

	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

const (
	CHANNELREADERS = policies.ChannelApplicationReaders
	CHANNELWRITERS = policies.ChannelApplicationWriters
)

//如果未提供基于资源的ACL提供程序或
//如果它不包含命名资源的策略
type defaultACLProvider struct {
	policyChecker policy.PolicyChecker

//对等策略（当前未使用）
	pResourcePolicyMap map[string]string

//通道特定策略
	cResourcePolicyMap map[string]string
}

func NewDefaultACLProvider() ACLProvider {
	d := &defaultACLProvider{}
	d.initialize()

	return d
}

func (d *defaultACLProvider) initialize() {
	d.policyChecker = policy.NewPolicyChecker(
		peer.NewChannelPolicyManagerGetter(),
		mgmt.GetLocalMSP(),
		mgmt.NewLocalMSPPrincipalGetter(),
	)

	d.pResourcePolicyMap = make(map[string]string)
	d.cResourcePolicyMap = make(map[string]string)

//----------LSCC----------
//P资源（目前由chaincode实现）
	d.pResourcePolicyMap[resources.Lscc_Install] = ""
	d.pResourcePolicyMap[resources.Lscc_GetInstalledChaincodes] = ""

//C资源
d.cResourcePolicyMap[resources.Lscc_Deploy] = ""  //提案中包含的ACL检查
d.cResourcePolicyMap[resources.Lscc_Upgrade] = "" //提案中包含的ACL检查
	d.cResourcePolicyMap[resources.Lscc_ChaincodeExists] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Lscc_GetDeploymentSpec] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Lscc_GetChaincodeData] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Lscc_GetInstantiatedChaincodes] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Lscc_GetCollectionsConfig] = CHANNELREADERS

//————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//P资源（无）

//C资源
	d.cResourcePolicyMap[resources.Qscc_GetChainInfo] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Qscc_GetBlockByNumber] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Qscc_GetBlockByHash] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Qscc_GetTransactionByID] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Qscc_GetBlockByTxID] = CHANNELREADERS

//-----------CSCC资源-----------
//P资源（目前由chaincode实现）
	d.pResourcePolicyMap[resources.Cscc_JoinChain] = ""
	d.pResourcePolicyMap[resources.Cscc_GetChannels] = ""

//C资源
	d.cResourcePolicyMap[resources.Cscc_GetConfigBlock] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Cscc_GetConfigTree] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Cscc_SimulateConfigTreeUpdate] = CHANNELWRITERS

//------------非SCC资源-----------
//对等资源
	d.cResourcePolicyMap[resources.Peer_Propose] = CHANNELWRITERS
	d.cResourcePolicyMap[resources.Peer_ChaincodeToChaincode] = CHANNELWRITERS
	d.cResourcePolicyMap[resources.Token_Issue] = CHANNELWRITERS
	d.cResourcePolicyMap[resources.Token_Transfer] = CHANNELWRITERS
	d.cResourcePolicyMap[resources.Token_List] = CHANNELREADERS

//事件资源
	d.cResourcePolicyMap[resources.Event_Block] = CHANNELREADERS
	d.cResourcePolicyMap[resources.Event_FilteredBlock] = CHANNELREADERS
}

//这应该包括从对等方调用的所有内容的详尽列表。
func (d *defaultACLProvider) defaultPolicy(resName string, cprovider bool) string {
	var pol string
	if cprovider {
		pol = d.cResourcePolicyMap[resName]
	} else {
		pol = d.pResourcePolicyMap[resName]
	}
	return pol
}

//checkacl通过将资源映射到其通道的acl来提供默认（v 1.0）行为
func (d *defaultACLProvider) CheckACL(resName string, channelID string, idinfo interface{}) error {
	policy := d.defaultPolicy(resName, true)
	if policy == "" {
		aclLogger.Errorf("Unmapped policy for %s", resName)
		return fmt.Errorf("Unmapped policy for %s", resName)
	}

	switch typedData := idinfo.(type) {
	case *pb.SignedProposal:
		return d.policyChecker.CheckPolicy(channelID, policy, typedData)
	case *common.Envelope:
		sd, err := typedData.AsSignedData()
		if err != nil {
			return err
		}
		return d.policyChecker.CheckPolicyBySignedData(channelID, policy, sd)
	case []*common.SignedData:
		return d.policyChecker.CheckPolicyBySignedData(channelID, policy, typedData)
	default:
		aclLogger.Errorf("Unmapped id on checkACL %s", resName)
		return fmt.Errorf("Unknown id on checkACL %s", resName)
	}
}
