
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


package msp

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/stretchr/testify/assert"
)

func TestInvalidAdminNodeOU(t *testing.T) {
//测试数据/节点1：
//配置启用nodeous，但管理员不携带
//任何有效的结节。因此，MSP初始化必须失败
	thisMSP, err := getLocalMSPWithVersionAndError(t, "testdata/nodeous1", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)
	assert.Error(t, err)

//mspv1_0不应失败
	thisMSP, err = getLocalMSPWithVersionAndError(t, "testdata/nodeous1", MSPv1_0)
	assert.False(t, thisMSP.(*bccspmsp).ouEnforcement)
	assert.NoError(t, err)
}

func TestInvalidSigningIdentityNodeOU(t *testing.T) {
//测试数据/节点2：
//配置启用noduous，但签名标识不携带
//任何有效的结节。因此签名身份验证应该失败
	thisMSP := getLocalMSPWithVersion(t, "testdata/nodeous2", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)

	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	err = id.Validate()
	assert.Error(t, err)

//mspv1_0不应失败
	thisMSP, err = getLocalMSPWithVersionAndError(t, "testdata/nodeous1", MSPv1_0)
	assert.False(t, thisMSP.(*bccspmsp).ouEnforcement)
	assert.NoError(t, err)

	id, err = thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	err = id.Validate()
	assert.NoError(t, err)
}

func TestValidMSPWithNodeOU(t *testing.T) {
//测试数据/节点3：
//配置使节点和管理和签名标识有效
	thisMSP := getLocalMSPWithVersion(t, "testdata/nodeous3", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)

	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	err = id.Validate()
	assert.NoError(t, err)

//MSPV1_0也不应失败
	thisMSP = getLocalMSPWithVersion(t, "testdata/nodeous3", MSPv1_0)
	assert.False(t, thisMSP.(*bccspmsp).ouEnforcement)

	id, err = thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	err = id.Validate()
	assert.NoError(t, err)
}

func TestValidMSPWithNodeOUAndOrganizationalUnits(t *testing.T) {
//测试数据/节点6：
//配置允许节点和组织单元，并且管理员和签名标识有效。
	thisMSP := getLocalMSPWithVersion(t, "testdata/nodeous6", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)

	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	err = id.Validate()
	assert.NoError(t, err)

//MSPV1_0也不应失败
	thisMSP = getLocalMSPWithVersion(t, "testdata/nodeous6", MSPv1_0)
	assert.False(t, thisMSP.(*bccspmsp).ouEnforcement)

	id, err = thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	err = id.Validate()
	assert.NoError(t, err)
}

func TestInvalidMSPWithNodeOUAndOrganizationalUnits(t *testing.T) {
//测试数据/节点6：
//配置使节点和组织单元，
//管理和签名标识无效，因为它们没有
//你在他们的组织中很常见。
	thisMSP, err := getLocalMSPWithVersionAndError(t, "testdata/nodeous7", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "could not validate identity's OUs: none of the identity's organizational units")
	}

//MSPV1_0也应该失败
	thisMSP, err = getLocalMSPWithVersionAndError(t, "testdata/nodeous7", MSPv1_0)
	assert.False(t, thisMSP.(*bccspmsp).ouEnforcement)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "could not validate identity's OUs: none of the identity's organizational units")
	}
}

func TestInvalidAdminOU(t *testing.T) {
//测试数据/节点4：
//配置启用nodeous，并且admin与config中指定的证明者链不匹配
	thisMSP, err := getLocalMSPWithVersionAndError(t, "testdata/nodeous4", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin 0 is invalid: The identity is not valid under this MSP [SampleOrg]: could not validate identity's OUs: certifiersIdentifier does not match")

//MSPV1_0也不应失败
	thisMSP, err = getLocalMSPWithVersionAndError(t, "testdata/nodeous4", MSPv1_0)
	assert.False(t, thisMSP.(*bccspmsp).ouEnforcement)
	assert.NoError(t, err)
}

func TestInvalidAdminOUNotAClient(t *testing.T) {
//测试数据/节点4：
//配置启用nodeous，而admin不是客户机
	thisMSP, err := getLocalMSPWithVersionAndError(t, "testdata/nodeous8", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "The identity does not contain OU [CLIENT]")

//mspv1_0不应失败
	thisMSP, err = getLocalMSPWithVersionAndError(t, "testdata/nodeous8", MSPv1_0)
	assert.False(t, thisMSP.(*bccspmsp).ouEnforcement)
	assert.NoError(t, err)
}

func TestSatisfiesPrincipalPeer(t *testing.T) {
//测试数据/节点3：
//配置使节点和管理和签名标识有效
	thisMSP := getLocalMSPWithVersion(t, "testdata/nodeous3", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)

//默认签名标识是对等机
	id, err := thisMSP.GetDefaultSigningIdentity()
	assert.NoError(t, err)

	err = id.Validate()
	assert.NoError(t, err)

	assert.True(t, t.Run("Check that id is a peer", func(t *testing.T) {
//检查ID是否为对等
		mspID, err := thisMSP.GetIdentifier()
		assert.NoError(t, err)
		principalBytes, err := proto.Marshal(&msp.MSPRole{Role: msp.MSPRole_PEER, MspIdentifier: mspID})
		assert.NoError(t, err)
		principal := &msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_ROLE,
			Principal:               principalBytes}
		err = id.SatisfiesPrincipal(principal)
		assert.NoError(t, err)
	}))

	assert.True(t, t.Run("Check that id is not a client", func(t *testing.T) {
//检查ID是否不是客户端
		mspID, err := thisMSP.GetIdentifier()
		assert.NoError(t, err)
		principalBytes, err := proto.Marshal(&msp.MSPRole{Role: msp.MSPRole_CLIENT, MspIdentifier: mspID})
		assert.NoError(t, err)
		principal := &msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_ROLE,
			Principal:               principalBytes}
		err = id.SatisfiesPrincipal(principal)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "The identity is not a [CLIENT] under this MSP [SampleOrg]")
	}))
}

func TestSatisfiesPrincipalClient(t *testing.T) {
//测试数据/节点3：
//配置使节点和管理和签名标识有效
	thisMSP := getLocalMSPWithVersion(t, "testdata/nodeous3", MSPv1_1)
	assert.True(t, thisMSP.(*bccspmsp).ouEnforcement)

//此MSP的管理员是客户端
	assert.Equal(t, 1, len(thisMSP.(*bccspmsp).admins))
	id := thisMSP.(*bccspmsp).admins[0]

	err := id.Validate()
	assert.NoError(t, err)

//检查ID是否为客户端
	assert.True(t, t.Run("Check that id is a client", func(t *testing.T) {
		mspID, err := thisMSP.GetIdentifier()
		assert.NoError(t, err)
		principalBytes, err := proto.Marshal(&msp.MSPRole{Role: msp.MSPRole_CLIENT, MspIdentifier: mspID})
		assert.NoError(t, err)
		principal := &msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_ROLE,
			Principal:               principalBytes}
		err = id.SatisfiesPrincipal(principal)
		assert.NoError(t, err)
	}))

	assert.True(t, t.Run("Check that id is not a peer", func(t *testing.T) {
//检查ID是否不是对等的
		mspID, err := thisMSP.GetIdentifier()
		assert.NoError(t, err)
		principalBytes, err := proto.Marshal(&msp.MSPRole{Role: msp.MSPRole_PEER, MspIdentifier: mspID})
		assert.NoError(t, err)
		principal := &msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_ROLE,
			Principal:               principalBytes}
		err = id.SatisfiesPrincipal(principal)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "The identity is not a [PEER] under this MSP [SampleOrg]")
	}))
}
