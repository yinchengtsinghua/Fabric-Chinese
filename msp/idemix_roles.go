
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

package msp

import (
	m "github.com/hyperledger/fabric/protos/msp"
)

//角色：表示IDemixRole
type Role int32

//期望的角色是4；我们可以使用位掩码组合它们
const (
	MEMBER Role = 1
	ADMIN  Role = 2
	CLIENT Role = 4
	PEER   Role = 8
//下一个角色值：16、32、64…
)

func (role Role) getValue() int {
	return int(role)
}

//checkRole证明所需角色是否包含在位掩码中
func checkRole(bitmask int, role Role) bool {
	return (bitmask & role.getValue()) == role.getValue()
}

//GetRoleMaskFromIdeMixRoles接收要在单个位掩码中组合的角色列表
func getRoleMaskFromIdemixRoles(roles []Role) int {
	mask := 0
	for _, role := range roles {
		mask = mask | role.getValue()
	}
	return mask
}

//GetRoleMaskFromMSProles接收要在单个位掩码中组合的角色列表
func getRoleMaskFromMSPRoles(roles []*m.MSPRole) int {
	mask := 0
	for _, role := range roles {
		mask = mask | getIdemixRoleFromMSPRole(role)
	}
	return mask
}

//GetRoleMaskFromIdeMixRole返回一个角色的位掩码
func GetRoleMaskFromIdemixRole(role Role) int {
	return getRoleMaskFromIdemixRoles([]Role{role})
}

//GetRoleMaskFromMSProle返回一个角色的位掩码
func getRoleMaskFromMSPRole(role *m.MSPRole) int {
	return getRoleMaskFromMSPRoles([]*m.MSPRole{role})
}

//getidemixrolefrommsprole获取MSP角色类型并返回整数值
func getIdemixRoleFromMSPRole(role *m.MSPRole) int {
	return getIdemixRoleFromMSPRoleType(role.GetRole())
}

//GetIDemixRoleFromsRoleType获取MSP角色类型并返回整数值
func getIdemixRoleFromMSPRoleType(rtype m.MSPRole_MSPRoleType) int {
	return getIdemixRoleFromMSPRoleValue(int(rtype))
}

//GetIDemixRoleFromsProleValue接收MSP角色值并返回IDemixEquivalent
func getIdemixRoleFromMSPRoleValue(role int) int {
	switch role {
	case int(m.MSPRole_ADMIN):
		return ADMIN.getValue()
	case int(m.MSPRole_CLIENT):
		return CLIENT.getValue()
	case int(m.MSPRole_MEMBER):
		return MEMBER.getValue()
	case int(m.MSPRole_PEER):
		return PEER.getValue()
	default:
		return MEMBER.getValue()
	}
}
