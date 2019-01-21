
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


package inquire

import (
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/graph"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/protos/common"
)

var logger = flogging.MustGetLogger("policies.inquire")

type inquireableSignaturePolicy struct {
	sigPol *common.SignaturePolicyEnvelope
}

//NewInquireableSignaturePolicy创建可查询的签名策略，
//来自策略和签名策略。
func NewInquireableSignaturePolicy(sigPol *common.SignaturePolicyEnvelope) policies.InquireablePolicy {
	return &inquireableSignaturePolicy{
		sigPol: sigPol,
	}
}

//satisfiedby返回一部分原则集，其中每个原则集
//满足政策要求。
func (isp *inquireableSignaturePolicy) SatisfiedBy() []policies.PrincipalSet {
	rootId := fmt.Sprintf("%d", 0)
	root := graph.NewTreeVertex(rootId, isp.sigPol.Rule)
	computePolicyTree(root)
	var res []policies.PrincipalSet
	for _, perm := range root.ToTree().Permute() {
		principalSet := principalsOfTree(perm, isp.sigPol.Identities)
		if len(principalSet) == 0 {
			return nil
		}
		res = append(res, principalSet)
	}
	return res
}

func principalsOfTree(tree *graph.Tree, principals policies.PrincipalSet) policies.PrincipalSet {
	var principalSet policies.PrincipalSet
	i := tree.BFS()
	for {
		v := i.Next()
		if v == nil {
			break
		}
		if !v.IsLeaf() {
			continue
		}
		pol := v.Data.(*common.SignaturePolicy)
		switch principalIndex := pol.Type.(type) {
		case *common.SignaturePolicy_SignedBy:
			if len(principals) <= int(principalIndex.SignedBy) {
				logger.Warning("Failed computing principalsOfTree, index out of bounds")
				return nil
			}
			principal := principals[principalIndex.SignedBy]
			principalSet = append(principalSet, principal)
		default:
//叶顶点的类型不是SignedBy
			logger.Warning("Leaf vertex", v.Id, "is of type", pol.GetType())
			return nil
		}
	}
	return principalSet
}

func computePolicyTree(v *graph.TreeVertex) {
	sigPol := v.Data.(*common.SignaturePolicy)
	if p := sigPol.GetNOutOf(); p != nil {
		v.Threshold = int(p.N)
		for i, rule := range p.Rules {
			id := fmt.Sprintf("%s.%d", v.Id, i)
			u := v.AddDescendant(graph.NewTreeVertex(id, rule))
			computePolicyTree(u)
		}
	}
}
