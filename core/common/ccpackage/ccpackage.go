
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016-2017保留所有权利。

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


package ccpackage

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

//ExtractSignedCDepspec从信封中提取消息
func ExtractSignedCCDepSpec(env *common.Envelope) (*common.ChannelHeader, *peer.SignedChaincodeDeploymentSpec, error) {
	p := &common.Payload{}
	err := proto.Unmarshal(env.Payload, p)
	if err != nil {
		return nil, nil, err
	}
	if p.Header == nil {
		return nil, nil, errors.New("channel header cannot be nil")
	}
	ch := &common.ChannelHeader{}
	err = proto.Unmarshal(p.Header.ChannelHeader, ch)
	if err != nil {
		return nil, nil, err
	}

	sp := &peer.SignedChaincodeDeploymentSpec{}
	err = proto.Unmarshal(p.Data, sp)
	if err != nil {
		return nil, nil, err
	}

	return ch, sp, nil
}

//此文件提供用于帮助安装chaincode的功能
//包工作流。特别地
//ownerCreateSignedCDepspec-每个所有者使用相同的部署创建对包的签名
//CreateSignedCDepspecForInstall-管理员或所有者创建要安装的包
//使用OwnerCreateSignedCDepspec中的包

//validatecp根据基本包验证已背书的包
func ValidateCip(baseCip, otherCip *peer.SignedChaincodeDeploymentSpec) error {
	if baseCip == nil || otherCip == nil {
		panic("do not call with nil parameters")
	}

	if (baseCip.OwnerEndorsements == nil && otherCip.OwnerEndorsements != nil) || (baseCip.OwnerEndorsements != nil && otherCip.OwnerEndorsements == nil) {
		return fmt.Errorf("endorsements should either be both nil or not nil")
	}

	bN := len(baseCip.OwnerEndorsements)
	oN := len(otherCip.OwnerEndorsements)
	if bN > 1 || oN > 1 {
		return fmt.Errorf("expect utmost 1 endorsement from a owner")
	}

	if bN != oN {
		return fmt.Errorf("Rule-all packages should be endorsed or none should be endorsed failed for (%d, %d)", bN, oN)
	}

	if !bytes.Equal(baseCip.ChaincodeDeploymentSpec, otherCip.ChaincodeDeploymentSpec) {
		return fmt.Errorf("Rule-all deployment specs should match(%d, %d)", len(baseCip.ChaincodeDeploymentSpec), len(otherCip.ChaincodeDeploymentSpec))
	}

	if !bytes.Equal(baseCip.InstantiationPolicy, otherCip.InstantiationPolicy) {
		return fmt.Errorf("Rule-all instantiation policies should match(%d, %d)", len(baseCip.InstantiationPolicy), len(otherCip.InstantiationPolicy))
	}

	return nil
}

func createSignedCCDepSpec(cdsbytes []byte, instpolicybytes []byte, endorsements []*peer.Endorsement) (*common.Envelope, error) {
	if cdsbytes == nil {
		return nil, fmt.Errorf("nil chaincode deployment spec")
	}

	if instpolicybytes == nil {
		return nil, fmt.Errorf("nil instantiation policy")
	}

//创建SignedChaincodeDeploymentSpec…
	cip := &peer.SignedChaincodeDeploymentSpec{ChaincodeDeploymentSpec: cdsbytes, InstantiationPolicy: instpolicybytes, OwnerEndorsements: endorsements}

//…然后整理它
	cipbytes := utils.MarshalOrPanic(cip)

//使用默认值（对于安装包来说这绝对是正常的）
	msgVersion := int32(0)
	epoch := uint64(0)
	chdr := utils.MakeChannelHeader(common.HeaderType_CHAINCODE_PACKAGE, msgVersion, "", epoch)

//创建有效载荷
	payl := &common.Payload{Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(chdr)}, Data: cipbytes}
	paylBytes, err := utils.GetBytesPayload(payl)
	if err != nil {
		return nil, err
	}

//这是未签名的信封。如果签名，安装包将被背书！=零
	return &common.Envelope{Payload: paylBytes}, nil
}

//CreateSignedCDepspecForInstall从一组由签名的包创建最终包
//业主。这类似于SDK如何从各种方案组装TX。
//签名回复。
func CreateSignedCCDepSpecForInstall(pack []*common.Envelope) (*common.Envelope, error) {
	if len(pack) == 0 {
		return nil, errors.New("no packages provided to collate")
	}

//规则。。。
//所有包裹必须背书，否则所有包裹都不应背书。
//链代码部署规范应该相同
	var baseCip *peer.SignedChaincodeDeploymentSpec
	var err error
	var endorsementExists bool
	var endorsements []*peer.Endorsement
	for n, r := range pack {
		p := &common.Payload{}
		if err = proto.Unmarshal(r.Payload, p); err != nil {
			return nil, err
		}

		cip := &peer.SignedChaincodeDeploymentSpec{}
		if err = proto.Unmarshal(p.Data, cip); err != nil {
			return nil, err
		}

//如果它是第一个元素，检查它是否有背书，这样我们可以
//执行背书规则
		if n == 0 {
			baseCip = cip
//如果有背书，所有其他所有人也应该签字。
			if len(cip.OwnerEndorsements) > 0 {
				endorsements = make([]*peer.Endorsement, len(pack))
			}

		} else if err = ValidateCip(baseCip, cip); err != nil {
			return nil, err
		}

		if endorsementExists {
			endorsements[n] = cip.OwnerEndorsements[0]
		}
	}

	return createSignedCCDepSpec(baseCip.ChaincodeDeploymentSpec, baseCip.InstantiationPolicy, endorsements)
}

//ownerCreateSignedCDepspec从chaincodeDeploymentsPec创建包，然后
//可选择背书
func OwnerCreateSignedCCDepSpec(cds *peer.ChaincodeDeploymentSpec, instPolicy *common.SignaturePolicyEnvelope, owner msp.SigningIdentity) (*common.Envelope, error) {
	if cds == nil {
		return nil, fmt.Errorf("invalid chaincode deployment spec")
	}

	if instPolicy == nil {
		return nil, fmt.Errorf("must provide an instantiation policy")
	}

	cdsbytes := utils.MarshalOrPanic(cds)

	instpolicybytes := utils.MarshalOrPanic(instPolicy)

	var endorsements []*peer.Endorsement
//不强制（在这个实用程序级别）具有签名
//这在开发/测试期间特别方便
//可能需要通过更高级别的政策来实施
	if owner != nil {
//序列化签名标识
		endorser, err := owner.Serialize()
		if err != nil {
			return nil, fmt.Errorf("Could not serialize the signing identity for %s, err %s", owner.GetIdentifier(), err)
		}

//用此背书人的密钥签署CDS、instpolicy和序列化背书人标识的串联
		signature, err := owner.Sign(append(cdsbytes, append(instpolicybytes, endorser...)...))
		if err != nil {
			return nil, fmt.Errorf("Could not sign the ccpackage, err %s", err)
		}

//每个所有者都以一个元素开始背书。所有此类背书
//包将由CreateSignedCDepspecForInstall收集到最终包中。
//当背书将有所有条目时
		endorsements = make([]*peer.Endorsement, 1)

		endorsements[0] = &peer.Endorsement{Signature: signature, Endorser: endorser}
	}

	return createSignedCCDepSpec(cdsbytes, instpolicybytes, endorsements)
}

//SigneXistingPackage向已签名的包添加签名。
func SignExistingPackage(env *common.Envelope, owner msp.SigningIdentity) (*common.Envelope, error) {
	if owner == nil {
		return nil, fmt.Errorf("owner not provided")
	}

	ch, sdepspec, err := ExtractSignedCCDepSpec(env)
	if err != nil {
		return nil, err
	}

	if ch == nil {
		return nil, fmt.Errorf("channel header not found in the envelope")
	}

	if sdepspec == nil || sdepspec.ChaincodeDeploymentSpec == nil || sdepspec.InstantiationPolicy == nil || sdepspec.OwnerEndorsements == nil {
		return nil, fmt.Errorf("invalid signed deployment spec")
	}

//序列化签名标识
	endorser, err := owner.Serialize()
	if err != nil {
		return nil, fmt.Errorf("Could not serialize the signing identity for %s, err %s", owner.GetIdentifier(), err)
	}

//用此背书人的密钥签署CDS、instpolicy和序列化背书人标识的串联
	signature, err := owner.Sign(append(sdepspec.ChaincodeDeploymentSpec, append(sdepspec.InstantiationPolicy, endorser...)...))
	if err != nil {
		return nil, fmt.Errorf("Could not sign the ccpackage, err %s", err)
	}

	endorsements := append(sdepspec.OwnerEndorsements, &peer.Endorsement{Signature: signature, Endorser: endorser})

	return createSignedCCDepSpec(sdepspec.ChaincodeDeploymentSpec, sdepspec.InstantiationPolicy, endorsements)
}
