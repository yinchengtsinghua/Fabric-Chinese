
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package policy

import (
	"errors"
	"fmt"

	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

//PolicyChecker提供了根据特定策略检查已签名建议的方法
//是否在通道中定义。
type PolicyChecker interface {
//检查策略检查通过的签名建议是否对
//已通过通道上的策略。
//如果没有传递任何通道，则直接调用checkPolicyNoChannel。
	CheckPolicy(channelID, policyName string, signedProp *pb.SignedProposal) error

//checkpolicyBySignedData检查传递的签名数据相对于
//已通过通道上的策略。
//如果没有传递任何通道，该方法将失败。
	CheckPolicyBySignedData(channelID, policyName string, sd []*common.SignedData) error

//checkpolicynochannel检查通过的签名建议是否对
//已通过本地MSP上的策略。
	CheckPolicyNoChannel(policyName string, signedProp *pb.SignedProposal) error
}

type policyChecker struct {
	channelPolicyManagerGetter policies.ChannelPolicyManagerGetter
	localMSP                   msp.IdentityDeserializer
	principalGetter            mgmt.MSPPrincipalGetter
}

//NewPolicyChecker创建新的PolicyChecker实例
func NewPolicyChecker(channelPolicyManagerGetter policies.ChannelPolicyManagerGetter, localMSP msp.IdentityDeserializer, principalGetter mgmt.MSPPrincipalGetter) PolicyChecker {
	return &policyChecker{channelPolicyManagerGetter, localMSP, principalGetter}
}

//检查策略检查通过的签名建议是否对
//已通过通道上的策略。
func (p *policyChecker) CheckPolicy(channelID, policyName string, signedProp *pb.SignedProposal) error {
	if channelID == "" {
		return p.CheckPolicyNoChannel(policyName, signedProp)
	}

	if policyName == "" {
		return fmt.Errorf("Invalid policy name during check policy on channel [%s]. Name must be different from nil.", channelID)
	}

	if signedProp == nil {
		return fmt.Errorf("Invalid signed proposal during check policy on channel [%s] with policy [%s]", channelID, policyName)
	}

//获取策略
	policyManager, _ := p.channelPolicyManagerGetter.Manager(channelID)
	if policyManager == nil {
		return fmt.Errorf("Failed to get policy manager for channel [%s]", channelID)
	}

//准备签名数据
	proposal, err := utils.GetProposal(signedProp.ProposalBytes)
	if err != nil {
		return fmt.Errorf("Failing extracting proposal during check policy on channel [%s] with policy [%s]: [%s]", channelID, policyName, err)
	}

	header, err := utils.GetHeader(proposal.Header)
	if err != nil {
		return fmt.Errorf("Failing extracting header during check policy on channel [%s] with policy [%s]: [%s]", channelID, policyName, err)
	}

	shdr, err := utils.GetSignatureHeader(header.SignatureHeader)
	if err != nil {
		return fmt.Errorf("Invalid Proposal's SignatureHeader during check policy on channel [%s] with policy [%s]: [%s]", channelID, policyName, err)
	}

	sd := []*common.SignedData{{
		Data:      signedProp.ProposalBytes,
		Identity:  shdr.Creator,
		Signature: signedProp.Signature,
	}}

	return p.CheckPolicyBySignedData(channelID, policyName, sd)
}

//checkpolicynochannel检查通过的签名建议是否对
//已通过本地MSP上的策略。
func (p *policyChecker) CheckPolicyNoChannel(policyName string, signedProp *pb.SignedProposal) error {
	if policyName == "" {
		return errors.New("Invalid policy name during channelless check policy. Name must be different from nil.")
	}

	if signedProp == nil {
		return fmt.Errorf("Invalid signed proposal during channelless check policy with policy [%s]", policyName)
	}

	proposal, err := utils.GetProposal(signedProp.ProposalBytes)
	if err != nil {
		return fmt.Errorf("Failing extracting proposal during channelless check policy with policy [%s]: [%s]", policyName, err)
	}

	header, err := utils.GetHeader(proposal.Header)
	if err != nil {
		return fmt.Errorf("Failing extracting header during channelless check policy with policy [%s]: [%s]", policyName, err)
	}

	shdr, err := utils.GetSignatureHeader(header.SignatureHeader)
	if err != nil {
		return fmt.Errorf("Invalid Proposal's SignatureHeader during channelless check policy with policy [%s]: [%s]", policyName, err)
	}

//用本地MSP反序列化建议的创建者
	id, err := p.localMSP.DeserializeIdentity(shdr.Creator)
	if err != nil {
		return fmt.Errorf("Failed deserializing proposal creator during channelless check policy with policy [%s]: [%s]", policyName, err)
	}

//为策略加载mspprincipal
	principal, err := p.principalGetter.Get(policyName)
	if err != nil {
		return fmt.Errorf("Failed getting local MSP principal during channelless check policy with policy [%s]: [%s]", policyName, err)
	}

//验证提案的创建者是否满足委托人的要求
	err = id.SatisfiesPrincipal(principal)
	if err != nil {
		return fmt.Errorf("Failed verifying that proposal's creator satisfies local MSP principal during channelless check policy with policy [%s]: [%s]", policyName, err)
	}

//验证签名
	return id.Verify(signedProp.ProposalBytes, signedProp.Signature)
}

//checkpolicyBySignedData检查传递的签名数据相对于
//已通过通道上的策略。
func (p *policyChecker) CheckPolicyBySignedData(channelID, policyName string, sd []*common.SignedData) error {
	if channelID == "" {
		return errors.New("Invalid channel ID name during check policy on signed data. Name must be different from nil.")
	}

	if policyName == "" {
		return fmt.Errorf("Invalid policy name during check policy on signed data on channel [%s]. Name must be different from nil.", channelID)
	}

	if sd == nil {
		return fmt.Errorf("Invalid signed data during check policy on channel [%s] with policy [%s]", channelID, policyName)
	}

//获取策略
	policyManager, _ := p.channelPolicyManagerGetter.Manager(channelID)
	if policyManager == nil {
		return fmt.Errorf("Failed to get policy manager for channel [%s]", channelID)
	}

//调用get-policy始终返回策略对象
	policy, _ := policyManager.GetPolicy(policyName)

//评估政策
	err := policy.Evaluate(sd)
	if err != nil {
		return fmt.Errorf("Failed evaluating policy on signed data during check policy on channel [%s] with policy [%s]: [%s]", channelID, policyName, err)
	}

	return nil
}

var pcFactory PolicyCheckerFactory

//policyCheckerFactory定义工厂接口，因此
//可以注入实际的实现
type PolicyCheckerFactory interface {
	NewPolicyChecker() PolicyChecker
}

//将调用一次RegisterPolicyCheckerFactory以设置
//将用于获取PolicyChecker实例的工厂
func RegisterPolicyCheckerFactory(f PolicyCheckerFactory) {
	pcFactory = f
}

//getpolicychecker返回policychecker的实例；
//实际实施由工厂控制
//通过RegisterPolicyCheckerFactory注册
func GetPolicyChecker() PolicyChecker {
	if pcFactory == nil {
		panic("The factory must be set first via RegisterPolicyCheckerFactory")
	}
	return pcFactory.NewPolicyChecker()
}
