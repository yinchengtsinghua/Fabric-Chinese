
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

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


package cauthdsl

import (
	"sort"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
)

//接受策略始终评估为真
var AcceptAllPolicy *cb.SignaturePolicyEnvelope

//MarshaledAcceptAllPolicy是已封送的AcceptAllPolicy版本
var MarshaledAcceptAllPolicy []byte

//RejectAllPolicy的计算结果始终为False
var RejectAllPolicy *cb.SignaturePolicyEnvelope

//MarshaledRejectAllPolicy是RejectAllPolicy的已封送版本
var MarshaledRejectAllPolicy []byte

func init() {
	var err error

	AcceptAllPolicy = Envelope(NOutOf(0, []*cb.SignaturePolicy{}), [][]byte{})
	MarshaledAcceptAllPolicy, err = proto.Marshal(AcceptAllPolicy)
	if err != nil {
		panic("Error marshaling trueEnvelope")
	}

	RejectAllPolicy = Envelope(NOutOf(1, []*cb.SignaturePolicy{}), [][]byte{})
	MarshaledRejectAllPolicy, err = proto.Marshal(RejectAllPolicy)
	if err != nil {
		panic("Error marshaling falseEnvelope")
	}
}

//信封生成一个信封消息，嵌入一个签名策略
func Envelope(policy *cb.SignaturePolicy, identities [][]byte) *cb.SignaturePolicyEnvelope {
	ids := make([]*msp.MSPPrincipal, len(identities))
	for i := range ids {
		ids[i] = &msp.MSPPrincipal{PrincipalClassification: msp.MSPPrincipal_IDENTITY, Principal: identities[i]}
	}

	return &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       policy,
		Identities: ids,
	}
}

//SignedBy创建需要给定签名者签名的签名策略
func SignedBy(index int32) *cb.SignaturePolicy {
	return &cb.SignaturePolicy{
		Type: &cb.SignaturePolicy_SignedBy{
			SignedBy: index,
		},
	}
}

//SignedByMspMember创建SignaturePolicyInvelope
//要求指定MSP的任何成员签名1次
func SignedByMspMember(mspId string) *cb.SignaturePolicyEnvelope {
	return signedByFabricEntity(mspId, msp.MSPRole_MEMBER)
}

//SignedByMspClient创建SignaturePolicyInvelope
//要求指定MSP的任何客户端提供1个签名
func SignedByMspClient(mspId string) *cb.SignaturePolicyEnvelope {
	return signedByFabricEntity(mspId, msp.MSPRole_CLIENT)
}

//SignedByMspPeer创建SignaturePolicyInvelope
//要求指定MSP的任何对等方提供1个签名
func SignedByMspPeer(mspId string) *cb.SignaturePolicyEnvelope {
	return signedByFabricEntity(mspId, msp.MSPRole_PEER)
}

//SignedByFabriceEntity创建SignaturePolicyInvelope
//需要指定MSP的具有传递角色的任何结构实体的1个签名
func signedByFabricEntity(mspId string, role msp.MSPRole_MSPRoleType) *cb.SignaturePolicyEnvelope {
//指定主体：它是我们刚找到的MSP的成员
	principal := &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: role, MspIdentifier: mspId})}

//创建策略：它只需要来自第一个（并且只有一个）主体的1个签名
	p := &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*cb.SignaturePolicy{SignedBy(0)}),
		Identities: []*msp.MSPPrincipal{principal},
	}

	return p
}

//SignedByMspadmin创建SignaturePolicyInvelope
//需要指定MSP的任何管理员的1个签名
func SignedByMspAdmin(mspId string) *cb.SignaturePolicyEnvelope {
//指定主体：它是我们刚找到的MSP的成员
	principal := &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_ADMIN, MspIdentifier: mspId})}

//创建策略：它只需要来自第一个（并且只有一个）主体的1个签名
	p := &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*cb.SignaturePolicy{SignedBy(0)}),
		Identities: []*msp.MSPPrincipal{principal},
	}

	return p
}

//用于生成“任意给定角色”类型策略的包装器
func signedByAnyOfGivenRole(role msp.MSPRole_MSPRoleType, ids []string) *cb.SignaturePolicyEnvelope {
//我们创建一个主体数组，一个主体
//在此链上定义的每个应用程序MSP
	sort.Strings(ids)
	principals := make([]*msp.MSPPrincipal, len(ids))
	sigspolicy := make([]*cb.SignaturePolicy, len(ids))
	for i, id := range ids {
		principals[i] = &msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_ROLE,
			Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: role, MspIdentifier: id})}
		sigspolicy[i] = SignedBy(int32(i))
	}

//创建策略：它只需要来自任何主体的1个签名
	p := &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, sigspolicy),
		Identities: principals,
	}

	return p
}

//SignedByAnyMember返回需要一个有效策略的策略
//任何组织成员的签名，其ID为
//在提供的字符串数组中列出
func SignedByAnyMember(ids []string) *cb.SignaturePolicyEnvelope {
	return signedByAnyOfGivenRole(msp.MSPRole_MEMBER, ids)
}

//SignedByAnyclient返回需要一个有效策略的策略
//来自ID为
//在提供的字符串数组中列出
func SignedByAnyClient(ids []string) *cb.SignaturePolicyEnvelope {
	return signedByAnyOfGivenRole(msp.MSPRole_CLIENT, ids)
}

//SignedByAnyper返回一个策略，该策略需要一个有效的
//来自ID为
//在提供的字符串数组中列出
func SignedByAnyPeer(ids []string) *cb.SignaturePolicyEnvelope {
	return signedByAnyOfGivenRole(msp.MSPRole_PEER, ids)
}

//signedByyanyadmin返回一个策略，该策略需要一个有效的
//ID为的任何组织的管理员的签名
//在提供的字符串数组中列出
func SignedByAnyAdmin(ids []string) *cb.SignaturePolicyEnvelope {
	return signedByAnyOfGivenRole(msp.MSPRole_ADMIN, ids)
}

//是一种利用无糖产生等效行为的简便方法。
func And(lhs, rhs *cb.SignaturePolicy) *cb.SignaturePolicy {
	return NOutOf(2, []*cb.SignaturePolicy{lhs, rhs})
}

//或者是一种方便的方法，利用noutof产生或等效的行为
func Or(lhs, rhs *cb.SignaturePolicy) *cb.SignaturePolicy {
	return NOutOf(1, []*cb.SignaturePolicy{lhs, rhs})
}

//n out of创建一个策略，该策略要求n个策略片中的n个值为true。
func NOutOf(n int32, policies []*cb.SignaturePolicy) *cb.SignaturePolicy {
	return &cb.SignaturePolicy{
		Type: &cb.SignaturePolicy_NOutOf_{
			NOutOf: &cb.SignaturePolicy_NOutOf{
				N:     n,
				Rules: policies,
			},
		},
	}
}
