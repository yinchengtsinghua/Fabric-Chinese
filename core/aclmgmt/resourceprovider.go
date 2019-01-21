
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

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

//-------错误------

//资源的PolicyNotFound缓存
type PolicyNotFound string

func (e PolicyNotFound) Error() string {
	return fmt.Sprintf("policy %s not found", string(e))
}

//无效信息
type InvalidIdInfo string

func (e InvalidIdInfo) Error() string {
	return fmt.Sprintf("Invalid id for policy [%s]", string(e))
}

//-------策略评估器------

//policyEvaluor接口为策略评估提供接口
type policyEvaluator interface {
	PolicyRefForAPI(resName string) string
	Evaluate(polName string, id []*common.SignedData) error
}

//PolicyEvaluator Impl实现PolicyEvaluator
type policyEvaluatorImpl struct {
	bundle channelconfig.Resources
}

func (pe *policyEvaluatorImpl) PolicyRefForAPI(resName string) string {
	app, exists := pe.bundle.ApplicationConfig()
	if !exists {
		return ""
	}

	pm := app.APIPolicyMapper()
	if pm == nil {
		return ""
	}

	return pm.PolicyRefForAPI(resName)
}

func (pe *policyEvaluatorImpl) Evaluate(polName string, sd []*common.SignedData) error {
	policy, ok := pe.bundle.PolicyManager().GetPolicy(polName)
	if !ok {
		return PolicyNotFound(polName)
	}

	return policy.Evaluate(sd)
}

//------资源策略提供程序----------

//aclmgmtpolicyProvider是基于资源的acl实现的接口。
type aclmgmtPolicyProvider interface {
//GetPolicyName返回给定资源名称的策略名称
	GetPolicyName(resName string) string

//checkacl备份aclprovider接口
	CheckACL(polName string, idinfo interface{}) error
}

//aclmgmtpolicyproviderimpl保存来自分类帐状态的字节
type aclmgmtPolicyProviderImpl struct {
	pEvaluator policyEvaluator
}

//GetPolicyName返回给定资源字符串的策略名称
func (rp *aclmgmtPolicyProviderImpl) GetPolicyName(resName string) string {
	return rp.pEvaluator.PolicyRefForAPI(resName)
}

//checkacl实现aclprovider的checkacl接口，以便可以注册它
//作为具有aclmgmt的提供程序
func (rp *aclmgmtPolicyProviderImpl) CheckACL(polName string, idinfo interface{}) error {
	aclLogger.Debugf("acl check(%s)", polName)

//我们将实现其他标识符。最后我们只需要一个签名数据
	var sd []*common.SignedData
	var err error
	switch idinfo.(type) {
	case *pb.SignedProposal:
		signedProp, _ := idinfo.(*pb.SignedProposal)
//准备签名数据
		proposal, err := utils.GetProposal(signedProp.ProposalBytes)
		if err != nil {
			return fmt.Errorf("Failing extracting proposal during check policy with policy [%s]: [%s]", polName, err)
		}

		header, err := utils.GetHeader(proposal.Header)
		if err != nil {
			return fmt.Errorf("Failing extracting header during check policy [%s]: [%s]", polName, err)
		}

		shdr, err := utils.GetSignatureHeader(header.SignatureHeader)
		if err != nil {
			return fmt.Errorf("Invalid Proposal's SignatureHeader during check policy [%s]: [%s]", polName, err)
		}

		sd = []*common.SignedData{{
			Data:      signedProp.ProposalBytes,
			Identity:  shdr.Creator,
			Signature: signedProp.Signature,
		}}
	case *common.Envelope:
		sd, err = idinfo.(*common.Envelope).AsSignedData()
		if err != nil {
			return err
		}
	default:
		return InvalidIdInfo(polName)
	}

	err = rp.pEvaluator.Evaluate(polName, sd)
	if err != nil {
		return fmt.Errorf("failed evaluating policy on signed data during check policy [%s]: [%s]", polName, err)
	}

	return nil
}

//--------资源提供程序-AclMgmtImpl用于执行基于资源的ACL的入口点API------

//资源getter获取channelconfig。给定通道ID的资源
type ResourceGetter func(channelID string) channelconfig.Resources

//使用资源配置信息提供ACL支持的资源提供程序
type resourceProvider struct {
//资源吸气剂
	resGetter ResourceGetter

//用于未定义资源的默认提供程序
	defaultProvider ACLProvider
}

//创建新的资源提供程序
func newResourceProvider(rg ResourceGetter, defprov ACLProvider) *resourceProvider {
	return &resourceProvider{rg, defprov}
}

//checkacl实现acl
func (rp *resourceProvider) CheckACL(resName string, channelID string, idinfo interface{}) error {
	resCfg := rp.resGetter(channelID)

	if resCfg != nil {
		pp := &aclmgmtPolicyProviderImpl{&policyEvaluatorImpl{resCfg}}
		policyName := pp.GetPolicyName(resName)
		if policyName != "" {
			aclLogger.Debugf("acl policy %s found in config for resource %s", policyName, resName)
			return pp.CheckACL(policyName, idinfo)
		}
		aclLogger.Debugf("acl policy not found in config for resource %s", resName)
	}

	return rp.defaultProvider.CheckACL(resName, channelID, idinfo)
}
