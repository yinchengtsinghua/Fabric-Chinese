
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

package aclmgmt

import (
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func newPolicyProvider(pEvaluator policyEvaluator) aclmgmtPolicyProvider {
	return &aclmgmtPolicyProviderImpl{pEvaluator}
}

//-------模拟------

//MockPolicyEvaluator Impl实现PolicyEvaluator
type mockPolicyEvaluatorImpl struct {
	pmap  map[string]string
	peval map[string]error
}

func (pe *mockPolicyEvaluatorImpl) PolicyRefForAPI(resName string) string {
	return pe.pmap[resName]
}

func (pe *mockPolicyEvaluatorImpl) Evaluate(polName string, sd []*common.SignedData) error {
	err, ok := pe.peval[polName]
	if !ok {
		return PolicyNotFound(polName)
	}

//这可能是非零或某些错误
	return err
}

func TestPolicyBase(t *testing.T) {
	peval := &mockPolicyEvaluatorImpl{pmap: map[string]string{"res": "pol"}, peval: map[string]error{"pol": nil}}
	pprov := newPolicyProvider(peval)
	sProp, _ := utils.MockSignedEndorserProposalOrPanic("A", &peer.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	err := pprov.CheckACL("pol", sProp)
	assert.NoError(t, err)

	env, err := utils.CreateSignedEnvelope(common.HeaderType_CONFIG, "myc", localmsp.NewSigner(), &common.ConfigEnvelope{}, 0, 0)
	assert.NoError(t, err)
	err = pprov.CheckACL("pol", env)
	assert.NoError(t, err)
}

func TestPolicyBad(t *testing.T) {
	peval := &mockPolicyEvaluatorImpl{pmap: map[string]string{"res": "pol"}, peval: map[string]error{"pol": nil}}
	pprov := newPolicyProvider(peval)

//坏政策
	err := pprov.CheckACL("pol", []byte("not a signed proposal"))
	assert.Error(t, err, InvalidIdInfo("pol").Error())

	sProp, _ := utils.MockSignedEndorserProposalOrPanic("A", &peer.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	err = pprov.CheckACL("badpolicy", sProp)
	assert.Error(t, err)

	sProp, _ = utils.MockSignedEndorserProposalOrPanic("A", &peer.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	sProp.ProposalBytes = []byte("bad proposal bytes")
	err = pprov.CheckACL("res", sProp)
	assert.Error(t, err)

	sProp, _ = utils.MockSignedEndorserProposalOrPanic("A", &peer.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	prop := &peer.Proposal{}
	if proto.Unmarshal(sProp.ProposalBytes, prop) != nil {
		t.FailNow()
	}
	prop.Header = []byte("bad hdr")
	sProp.ProposalBytes = utils.MarshalOrPanic(prop)
	err = pprov.CheckACL("res", sProp)
	assert.Error(t, err)
}

func init() {
	var err error
//设置MSP管理器，以便我们可以签名/验证
	err = msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		fmt.Printf("Could not load msp config, err %s", err)
		os.Exit(-1)
		return
	}
}
