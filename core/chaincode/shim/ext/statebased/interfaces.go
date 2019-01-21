
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


package statebased

import "fmt"

//背书政策标识的角色类型
type RoleType string

const (
//roletypember标识组织的成员标识
	RoleTypeMember = RoleType("MEMBER")
//roletypeer标识组织的对等身份
	RoleTypePeer = RoleType("PEER")
)

//
//
//
type RoleTypeDoesNotExistError struct {
	RoleType RoleType
}

func (r *RoleTypeDoesNotExistError) Error() string {
	return fmt.Sprintf("role type %s does not exist", r.RoleType)
}

//
//
//
//
type KeyEndorsementPolicy interface {
//
	Policy() ([]byte, error)

//
//
//在第一个参数中指定。在其他方面，期望的角色
//取决于通道的配置：如果它支持节点OU，则为
//
//如果不是的话。
	AddOrgs(roleType RoleType, organizations ...string) error

//
//
	DelOrgs(organizations ...string)

//
	ListOrgs() []string
}
