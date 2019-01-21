
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


package ccpackage

import (
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

func ownerCreateCCDepSpec(codepackage []byte, sigpolicy *common.SignaturePolicyEnvelope, owner msp.SigningIdentity) (*common.Envelope, error) {
	cds := &peer.ChaincodeDeploymentSpec{CodePackage: codepackage}
	return OwnerCreateSignedCCDepSpec(cds, sigpolicy, owner)
}

//仅使用本地MSP管理员创建实例化策略
func createInstantiationPolicy(mspid string, role mspprotos.MSPRole_MSPRoleType) *common.SignaturePolicyEnvelope {
	principals := []*mspprotos.MSPPrincipal{{
		PrincipalClassification: mspprotos.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&mspprotos.MSPRole{Role: role, MspIdentifier: mspid})}}
	sigspolicy := []*common.SignaturePolicy{cauthdsl.SignedBy(int32(0))}

//创建策略：它只需要来自任何主体的1个签名
	p := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       cauthdsl.NOutOf(1, sigspolicy),
		Identities: principals,
	}

	return p
}

func TestOwnerCreateSignedCCDepSpec(t *testing.T) {
	mspid, _ := localmsp.GetIdentifier()
	sigpolicy := createInstantiationPolicy(mspid, mspprotos.MSPRole_ADMIN)
	env, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy, signer)
	if err != nil || env == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}
}

func TestAddSignature(t *testing.T) {
	mspid, _ := localmsp.GetIdentifier()
	sigpolicy := createInstantiationPolicy(mspid, mspprotos.MSPRole_ADMIN)
	env, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy, signer)
	if err != nil || env == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}
//用同一个签名者再添加一个签名者（我们没有要测试的其他签名者）
	env, err = SignExistingPackage(env, signer)
	if err != nil || env == nil {
		t.Fatalf("error signing existing package %s", err)
		return
	}
//…然后在那里签名祝你好运
	env, err = SignExistingPackage(env, signer)
	if err != nil || env == nil {
		t.Fatalf("error signing existing package %s", err)
		return
	}

	p := &common.Payload{}
	if err = proto.Unmarshal(env.Payload, p); err != nil {
		t.Fatalf("fatal error unmarshal payload")
		return
	}

	sigdepspec := &peer.SignedChaincodeDeploymentSpec{}
	if err = proto.Unmarshal(p.Data, sigdepspec); err != nil || sigdepspec == nil {
		t.Fatalf("fatal error unmarshal sigdepspec")
		return
	}

	if len(sigdepspec.OwnerEndorsements) != 3 {
		t.Fatalf("invalid number of endorsements %d", len(sigdepspec.OwnerEndorsements))
		return
	}
}

func TestMissingSigaturePolicy(t *testing.T) {
	env, err := ownerCreateCCDepSpec([]byte("codepackage"), nil, signer)
	if err == nil || env != nil {
		t.Fatalf("expected error on missing signature policy")
		return
	}
}

func TestCreateSignedCCDepSpecForInstall(t *testing.T) {
	mspid, _ := localmsp.GetIdentifier()
	sigpolicy := createInstantiationPolicy(mspid, mspprotos.MSPRole_ADMIN)
	env1, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy, nil)
	if err != nil || env1 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}

	env2, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy, nil)
	if err != nil || env2 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}

	pack := []*common.Envelope{env1, env2}
	env, err := CreateSignedCCDepSpecForInstall(pack)
	if err != nil || env == nil {
		t.Fatalf("error creating install package %s", err)
		return
	}

	p := &common.Payload{}
	if err = proto.Unmarshal(env.Payload, p); err != nil {
		t.Fatalf("fatal error unmarshal payload")
		return
	}

	cip2 := &peer.SignedChaincodeDeploymentSpec{}
	if err = proto.Unmarshal(p.Data, cip2); err != nil {
		t.Fatalf("fatal error unmarshal cip")
		return
	}

	p = &common.Payload{}
	if err = proto.Unmarshal(env1.Payload, p); err != nil {
		t.Fatalf("fatal error unmarshal payload")
		return
	}

	cip1 := &peer.SignedChaincodeDeploymentSpec{}
	if err = proto.Unmarshal(p.Data, cip1); err != nil {
		t.Fatalf("fatal error unmarshal cip")
		return
	}

	if err = ValidateCip(cip1, cip2); err != nil {
		t.Fatalf("fatal error validating cip1 (%v) against cip2(%v)", cip1, cip2)
		return
	}
}

func TestMismatchedCodePackages(t *testing.T) {
	mspid, _ := localmsp.GetIdentifier()
	sigpolicy := createInstantiationPolicy(mspid, mspprotos.MSPRole_ADMIN)
	env1, err := ownerCreateCCDepSpec([]byte("codepackage1"), sigpolicy, nil)
	if err != nil || env1 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}

	env2, err := ownerCreateCCDepSpec([]byte("codepackage2"), sigpolicy, nil)
	if err != nil || env2 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}
	pack := []*common.Envelope{env1, env2}
	env, err := CreateSignedCCDepSpecForInstall(pack)
	if err == nil || env != nil {
		t.Fatalf("expected error creating install from mismatched code package but succeeded")
		return
	}
}

func TestMismatchedEndorsements(t *testing.T) {
	mspid, _ := localmsp.GetIdentifier()
	sigpolicy := createInstantiationPolicy(mspid, mspprotos.MSPRole_ADMIN)
	env1, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy, signer)
	if err != nil || env1 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}

	env2, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy, nil)
	if err != nil || env2 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}
	pack := []*common.Envelope{env1, env2}
	env, err := CreateSignedCCDepSpecForInstall(pack)
	if err == nil || env != nil {
		t.Fatalf("expected error creating install from mismatched endorsed package but succeeded")
		return
	}
}

func TestMismatchedSigPolicy(t *testing.T) {
	sigpolicy1 := createInstantiationPolicy("mspid1", mspprotos.MSPRole_ADMIN)
	env1, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy1, signer)
	if err != nil || env1 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}

	sigpolicy2 := createInstantiationPolicy("mspid2", mspprotos.MSPRole_ADMIN)
	env2, err := ownerCreateCCDepSpec([]byte("codepackage"), sigpolicy2, signer)
	if err != nil || env2 == nil {
		t.Fatalf("error owner creating package %s", err)
		return
	}
	pack := []*common.Envelope{env1, env2}
	env, err := CreateSignedCCDepSpecForInstall(pack)
	if err == nil || env != nil {
		t.Fatalf("expected error creating install from mismatched signature policies but succeeded")
		return
	}
}

var localmsp msp.MSP
var signer msp.SigningIdentity
var signerSerialized []byte

func TestMain(m *testing.M) {
//设置MSP管理器，以便我们可以签名/验证
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		os.Exit(-1)
		fmt.Printf("Could not initialize msp")
		return
	}
	localmsp = mspmgmt.GetLocalMSP()
	if localmsp == nil {
		os.Exit(-1)
		fmt.Printf("Could not get msp")
		return
	}
	signer, err = localmsp.GetDefaultSigningIdentity()
	if err != nil {
		os.Exit(-1)
		fmt.Printf("Could not get signer")
		return
	}

	signerSerialized, err = signer.Serialize()
	if err != nil {
		os.Exit(-1)
		fmt.Printf("Could not serialize identity")
		return
	}

	os.Exit(m.Run())
}
