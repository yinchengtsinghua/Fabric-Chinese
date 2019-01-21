
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
	"testing"

	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/policy/mocks"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckPolicyInvalidArgs(t *testing.T) {
	policyManagerGetter := &mocks.MockChannelPolicyManagerGetter{
		Managers: map[string]policies.Manager{
			"A": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{
					Deserializer: &mocks.MockIdentityDeserializer{
						Identity: []byte("Alice"),
						Msg:      []byte("msg1"),
					},
				},
			},
		},
	}
	pc := &policyChecker{channelPolicyManagerGetter: policyManagerGetter}

	err := pc.CheckPolicy("B", "admin", &peer.SignedProposal{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to get policy manager for channel [B]")
}

func TestRegisterPolicyCheckerFactoryInvalidArgs(t *testing.T) {
	RegisterPolicyCheckerFactory(nil)
	assert.Panics(t, func() {
		GetPolicyChecker()
	})

	RegisterPolicyCheckerFactory(nil)
}

func TestRegisterPolicyCheckerFactory(t *testing.T) {
	policyManagerGetter := &mocks.MockChannelPolicyManagerGetter{
		Managers: map[string]policies.Manager{
			"A": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{
					Deserializer: &mocks.MockIdentityDeserializer{
						Identity: []byte("Alice"),
						Msg:      []byte("msg1"),
					},
				},
			},
		},
	}
	pc := &policyChecker{channelPolicyManagerGetter: policyManagerGetter}

	factory := &MockPolicyCheckerFactory{}
	factory.On("NewPolicyChecker").Return(pc)

	RegisterPolicyCheckerFactory(factory)
	pc2 := GetPolicyChecker()
	assert.Equal(t, pc, pc2)
}

func TestCheckPolicyBySignedDataInvalidArgs(t *testing.T) {
	policyManagerGetter := &mocks.MockChannelPolicyManagerGetter{
		Managers: map[string]policies.Manager{
			"A": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{
					Deserializer: &mocks.MockIdentityDeserializer{
						Identity: []byte("Alice"),
						Msg:      []byte("msg1"),
					}},
			},
		},
	}
	pc := &policyChecker{channelPolicyManagerGetter: policyManagerGetter}

	err := pc.CheckPolicyBySignedData("", "admin", []*common.SignedData{{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid channel ID name during check policy on signed data. Name must be different from nil.")

	err = pc.CheckPolicyBySignedData("A", "", []*common.SignedData{{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid policy name during check policy on signed data on channel [A]. Name must be different from nil.")

	err = pc.CheckPolicyBySignedData("A", "admin", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid signed data during check policy on channel [A] with policy [admin]")

	err = pc.CheckPolicyBySignedData("B", "admin", []*common.SignedData{{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to get policy manager for channel [B]")

	err = pc.CheckPolicyBySignedData("A", "admin", []*common.SignedData{{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed evaluating policy on signed data during check policy on channel [A] with policy [admin]")
}

func TestPolicyCheckerInvalidArgs(t *testing.T) {
	policyManagerGetter := &mocks.MockChannelPolicyManagerGetter{
		Managers: map[string]policies.Manager{
			"A": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{Deserializer: &mocks.MockIdentityDeserializer{
					Identity: []byte("Alice"),
					Msg:      []byte("msg1"),
				}},
			},
			"B": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{Deserializer: &mocks.MockIdentityDeserializer{
					Identity: []byte("Bob"),
					Msg:      []byte("msg2"),
				}},
			},
			"C": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{Deserializer: &mocks.MockIdentityDeserializer{
					Identity: []byte("Alice"),
					Msg:      []byte("msg3"),
				}},
			},
		},
	}
	identityDeserializer := &mocks.MockIdentityDeserializer{
		Identity: []byte("Alice"),
		Msg:      []byte("msg1"),
	}
	pc := NewPolicyChecker(
		policyManagerGetter,
		identityDeserializer,
		&mocks.MockMSPPrincipalGetter{Principal: []byte("Alice")},
	)

//检查（非空通道、空策略）是否失败
	err := pc.CheckPolicy("A", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid policy name during check policy on channel [A]. Name must be different from nil.")

//检查（空通道、空策略）是否失败
	err = pc.CheckPolicy("", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid policy name during channelless check policy. Name must be different from nil.")

//检查（非空通道、非空策略、无建议）是否失败
	err = pc.CheckPolicy("A", "A", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid signed proposal during check policy on channel [A] with policy [A]")

//检查（空通道、非空策略、无建议）是否失败
	err = pc.CheckPolicy("", "A", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid signed proposal during channelless check policy with policy [A]")
}

func TestPolicyChecker(t *testing.T) {
	policyManagerGetter := &mocks.MockChannelPolicyManagerGetter{
		Managers: map[string]policies.Manager{
			"A": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{
					Deserializer: &mocks.MockIdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1")},
				},
			},
			"B": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{
					Deserializer: &mocks.MockIdentityDeserializer{
						Identity: []byte("Bob"),
						Msg:      []byte("msg2"),
					},
				},
			},
			"C": &mocks.MockChannelPolicyManager{
				MockPolicy: &mocks.MockPolicy{
					Deserializer: &mocks.MockIdentityDeserializer{
						Identity: []byte("Alice"),
						Msg:      []byte("msg3"),
					},
				},
			},
		},
	}
	identityDeserializer := &mocks.MockIdentityDeserializer{
		Identity: []byte("Alice"),
		Msg:      []byte("msg1"),
	}
	pc := NewPolicyChecker(
		policyManagerGetter,
		identityDeserializer,
		&mocks.MockMSPPrincipalGetter{Principal: []byte("Alice")},
	)

//根据A频道的读者验证Alice签名
	sProp, _ := utils.MockSignedEndorserProposalOrPanic("A", &peer.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	policyManagerGetter.Managers["A"].(*mocks.MockChannelPolicyManager).MockPolicy.(*mocks.MockPolicy).Deserializer.(*mocks.MockIdentityDeserializer).Msg = sProp.ProposalBytes
	sProp.Signature = sProp.ProposalBytes
	err := pc.CheckPolicy("A", "readers", sProp)
	assert.NoError(t, err)

//爱丽丝提出的A频道的建议不应通过B频道，因为不涉及爱丽丝。
	err = pc.CheckPolicy("B", "readers", sProp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed evaluating policy on signed data during check policy on channel [B] with policy [readers]: [Invalid Identity]")

//爱丽丝提出的A频道的建议应与C频道不符，因为C频道涉及爱丽丝，但签名无效。
	err = pc.CheckPolicy("C", "readers", sProp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed evaluating policy on signed data during check policy on channel [C] with policy [readers]: [Invalid Signature]")

//爱丽丝是本地MSP的成员，策略检查必须成功。
	identityDeserializer.Msg = sProp.ProposalBytes
	err = pc.CheckPolicyNoChannel(mgmt.Members, sProp)
	assert.NoError(t, err)

	sProp, _ = utils.MockSignedEndorserProposalOrPanic("A", &peer.ChaincodeSpec{}, []byte("Bob"), []byte("msg2"))
//Bob不是本地MSP的成员，策略检查必须失败
	err = pc.CheckPolicyNoChannel(mgmt.Members, sProp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed deserializing proposal creator during channelless check policy with policy [Members]: [Invalid Identity]")
}

type MockPolicyCheckerFactory struct {
	mock.Mock
}

func (m *MockPolicyCheckerFactory) NewPolicyChecker() PolicyChecker {
	args := m.Called()
	return args.Get(0).(PolicyChecker)
}
