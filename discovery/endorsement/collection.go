
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


package endorsement

import (
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/gossip/api"
	. "github.com/hyperledger/fabric/protos/discovery"
	"github.com/pkg/errors"
)

func principalsFromCollectionConfig(configBytes []byte) (principalSetsByCollectionName, error) {
	principalSetsByCollections := make(principalSetsByCollectionName)
	if len(configBytes) == 0 {
		return principalSetsByCollections, nil
	}
	ccp, err := privdata.ParseCollectionConfig(configBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid collection bytes")
	}
	for _, colConfig := range ccp.Config {
		staticCol := colConfig.GetStaticCollectionConfig()
		if staticCol == nil {
//现在我们只支持静态集合，所以如果我们有其他的
//we should refuse to process further
			return nil, errors.Errorf("expected a static collection but got %v instead", colConfig)
		}
		if staticCol.MemberOrgsPolicy == nil {
			return nil, errors.Errorf("MemberOrgsPolicy of %s is nil", staticCol.Name)
		}
		pol := staticCol.MemberOrgsPolicy.GetSignaturePolicy()
		if pol == nil {
			return nil, errors.Errorf("policy of %s is nil", staticCol.Name)
		}
		var principals policies.PrincipalSet
//我们现在从策略中提取所有主体
		for _, principal := range pol.Identities {
			principals = append(principals, principal)
		}
		principalSetsByCollections[staticCol.Name] = principals
	}
	return principalSetsByCollections, nil
}

type principalSetsByCollectionName map[string]policies.PrincipalSet

//ToIdentityFilter将此PrincipalSetsByCollectionName映射转换为筛选器
//接受或拒绝同龄人的身份。
func (psbc principalSetsByCollectionName) toIdentityFilter(channel string, evaluator principalEvaluator, cc *ChaincodeCall) (identityFilter, error) {
	var principalSets policies.PrincipalSets
	for _, col := range cc.CollectionNames {
//我们感兴趣的每个集合都应该存在于PrincipalSetsByCollectionName映射中。
//否则，我们无法计算过滤器，因为我们找不到主体和对等身份
//need to satisfy.
		principalSet, exists := psbc[col]
		if !exists {
			return nil, errors.Errorf("collection %s doesn't exist in collection config for chaincode %s", col, cc.Name)
		}
		principalSets = append(principalSets, principalSet)
	}
	return filterForPrincipalSets(channel, evaluator, principalSets), nil
}

//filterForPrincipalSets creates a filter of peer identities out of the given PrincipalSets
func filterForPrincipalSets(channel string, evaluator principalEvaluator, sets policies.PrincipalSets) identityFilter {
	return func(identity api.PeerIdentityType) bool {
//遍历所有主体集并确保每个主体集
//authorizes the identity.
		for _, principalSet := range sets {
			if !isIdentityAuthorizedByPrincipalSet(channel, evaluator, principalSet, identity) {
				return false
			}
		}
		return true
	}
}

//IsIdentityAuthorizedByPrincipalset返回给定标识是否满足给定Principalset之外的某个主体
func isIdentityAuthorizedByPrincipalSet(channel string, evaluator principalEvaluator, principalSet policies.PrincipalSet, identity api.PeerIdentityType) bool {
//我们要找一个授权身份的委托人
//在Principalset的所有负责人中
	for _, principal := range principalSet {
		err := evaluator.SatisfiesPrincipal(channel, identity, principal)
		if err != nil {
			continue
		}
//否则，err为零，因此我们找到一个授权委托人
//给定的标识。
		return true
	}
	return false
}
