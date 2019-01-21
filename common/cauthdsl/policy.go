
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


package cauthdsl

import (
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	mspp "github.com/hyperledger/fabric/protos/msp"
)

type Identity interface {
//satisfiesprincipal检查此实例是否匹配
//mspprincipal中提供的说明。支票可以
//涉及逐字节比较（如果主体是
//或可能需要MSP验证
	SatisfiesPrincipal(principal *mspp.MSPPrincipal) error

//GetIdentifier返回该标识的标识符
	GetIdentifier() *msp.IdentityIdentifier
}

type IdentityAndSignature interface {
//标识返回与此实例关联的标识
	Identity() (Identity, error)

//verify返回此标识在消息上的签名的有效性状态
	Verify() error
}

type deserializeAndVerify struct {
	signedData           *cb.SignedData
	deserializer         msp.IdentityDeserializer
	deserializedIdentity msp.Identity
}

func (d *deserializeAndVerify) Identity() (Identity, error) {
	deserializedIdentity, err := d.deserializer.DeserializeIdentity(d.signedData.Identity)
	if err != nil {
		return nil, err
	}

	d.deserializedIdentity = deserializedIdentity
	return deserializedIdentity, nil
}

func (d *deserializeAndVerify) Verify() error {
	if d.deserializedIdentity == nil {
		cauthdslLogger.Panicf("programming error, Identity must be called prior to Verify")
	}
	return d.deserializedIdentity.Verify(d.signedData.Data, d.signedData.Signature)
}

type provider struct {
	deserializer msp.IdentityDeserializer
}

//NewProviderImpl为cauthDSL类型策略提供策略生成器
func NewPolicyProvider(deserializer msp.IdentityDeserializer) policies.Provider {
	return &provider{
		deserializer: deserializer,
	}
}

//new policy基于策略字节创建新策略
func (pr *provider) NewPolicy(data []byte) (policies.Policy, proto.Message, error) {
	sigPolicy := &cb.SignaturePolicyEnvelope{}
	if err := proto.Unmarshal(data, sigPolicy); err != nil {
		return nil, nil, fmt.Errorf("Error unmarshaling to SignaturePolicy: %s", err)
	}

	if sigPolicy.Version != 0 {
		return nil, nil, fmt.Errorf("This evaluator only understands messages of version 0, but version was %d", sigPolicy.Version)
	}

	compiled, err := compile(sigPolicy.Rule, sigPolicy.Identities, pr.deserializer)
	if err != nil {
		return nil, nil, err
	}

	return &policy{
		evaluator:    compiled,
		deserializer: pr.deserializer,
	}, sigPolicy, nil

}

type policy struct {
	evaluator    func([]IdentityAndSignature, []bool) bool
	deserializer msp.IdentityDeserializer
}

//Evaluate获取一组SignedData并评估该组签名是否满足策略
func (p *policy) Evaluate(signatureSet []*cb.SignedData) error {
	if p == nil {
		return fmt.Errorf("No such policy")
	}
	idAndS := make([]IdentityAndSignature, len(signatureSet))
	for i, sd := range signatureSet {
		idAndS[i] = &deserializeAndVerify{
			signedData:   sd,
			deserializer: p.deserializer,
		}
	}

	ok := p.evaluator(deduplicate(idAndS), make([]bool, len(signatureSet)))
	if !ok {
		return errors.New("signature set did not satisfy policy")
	}
	return nil
}
