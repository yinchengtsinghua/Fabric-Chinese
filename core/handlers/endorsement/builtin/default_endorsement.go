
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


package builtin

import (
	. "github.com/hyperledger/fabric/core/handlers/endorsement/api"
	. "github.com/hyperledger/fabric/core/handlers/endorsement/api/identities"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//默认认可工厂返回认可插件工厂，认可插件工厂返回插件
//作为默认的背书系统链码
type DefaultEndorsementFactory struct {
}

//new返回一个作为默认认可系统链码的认可插件
func (*DefaultEndorsementFactory) New() Plugin {
	return &DefaultEndorsement{}
}

//默认认可是一个认可插件，作为默认认可系统链码。
type DefaultEndorsement struct {
	SigningIdentityFetcher
}

//认可对给定的有效负载（proposalResponsePayLoad字节）进行签名，并可选地对其进行变异。
//返回：
//背书：有效载荷上的签名，以及用于验证签名的标识。
//作为输入给出的有效负载（可以在此函数中修改）
//或失败时出错
func (e *DefaultEndorsement) Endorse(prpBytes []byte, sp *peer.SignedProposal) (*peer.Endorsement, []byte, error) {
	signer, err := e.SigningIdentityForRequest(sp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed fetching signing identity")
	}
//序列化签名标识
	identityBytes, err := signer.Serialize()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not serialize the signing identity")
	}

//用此背书人的密钥签署提案响应和序列化背书人标识的串联
	signature, err := signer.Sign(append(prpBytes, identityBytes...))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not sign the proposal response payload")
	}
	endorsement := &peer.Endorsement{Signature: signature, Endorser: identityBytes}
	return endorsement, prpBytes, nil
}

//init将依赖项插入插件的实例中
func (e *DefaultEndorsement) Init(dependencies ...Dependency) error {
	for _, dep := range dependencies {
		sIDFetcher, isSigningIdentityFetcher := dep.(SigningIdentityFetcher)
		if !isSigningIdentityFetcher {
			continue
		}
		e.SigningIdentityFetcher = sIDFetcher
		return nil
	}
	return errors.New("could not find SigningIdentityFetcher in dependencies")
}
