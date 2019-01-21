
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
*/


package inquire

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/protos/msp"
)

//
type ComparablePrincipal struct {
	principal *msp.MSPPrincipal
	ou        *msp.OrganizationUnit
	role      *msp.MSPRole
	mspID     string
}

//NewComparablePrincipal根据给定的mspprincipal创建了一个可比较的主体。
//
func NewComparablePrincipal(principal *msp.MSPPrincipal) *ComparablePrincipal {
	if principal == nil {
		logger.Warning("Principal is nil")
		return nil
	}
	cp := &ComparablePrincipal{
		principal: principal,
	}
	switch principal.PrincipalClassification {
	case msp.MSPPrincipal_ROLE:
		return cp.ToRole()
	case msp.MSPPrincipal_ORGANIZATION_UNIT:
		return cp.ToOURole()
	}
	mapping := msp.MSPPrincipal_Classification_name[int32(principal.PrincipalClassification)]
	logger.Warning("Received an unsupported principal type:", principal.PrincipalClassification, "mapped to", mapping)
	return nil
}

//
//
//
func (cp *ComparablePrincipal) IsFound(set ...*ComparablePrincipal) bool {
	for _, cp2 := range set {
		if cp.IsA(cp2) {
			return true
		}
	}
	return false
}

//
//
//
//
//
//
func (cp *ComparablePrincipal) IsA(other *ComparablePrincipal) bool {
	this := cp

	if other == nil {
		return false
	}
	if this.principal == nil || other.principal == nil {
		logger.Warning("Used an un-initialized ComparablePrincipal")
		return false
	}
//
	if this.mspID != other.mspID {
		return false
	}

//
//
	if other.role != nil && other.role.Role == msp.MSPRole_MEMBER {
		return true
	}

//
	if this.ou != nil && other.ou != nil {
		sameOU := this.ou.OrganizationalUnitIdentifier == other.ou.OrganizationalUnitIdentifier
		sameIssuer := bytes.Equal(this.ou.CertifiersIdentifier, other.ou.CertifiersIdentifier)
		return sameOU && sameIssuer
	}

//
	if this.role != nil && other.role != nil {
		return this.role.Role == other.role.Role
	}

//
//
	return false
}

//
func (cp *ComparablePrincipal) ToOURole() *ComparablePrincipal {
	ouRole := &msp.OrganizationUnit{}
	err := proto.Unmarshal(cp.principal.Principal, ouRole)
	if err != nil {
		logger.Warning("Failed unmarshaling principal:", err)
		return nil
	}
	cp.mspID = ouRole.MspIdentifier
	cp.ou = ouRole
	return cp
}

//
func (cp *ComparablePrincipal) ToRole() *ComparablePrincipal {
	mspRole := &msp.MSPRole{}
	err := proto.Unmarshal(cp.principal.Principal, mspRole)
	if err != nil {
		logger.Warning("Failed unmarshaling principal:", err)
		return nil
	}
	cp.mspID = mspRole.MspIdentifier
	cp.role = mspRole
	return cp
}

//
type ComparablePrincipalSet []*ComparablePrincipal

//
func (cps ComparablePrincipalSet) ToPrincipalSet() policies.PrincipalSet {
	var res policies.PrincipalSet
	for _, cp := range cps {
		res = append(res, cp.principal)
	}
	return res
}

//
func (cps ComparablePrincipalSet) String() string {
	buff := bytes.Buffer{}
	buff.WriteString("[")
	for i, cp := range cps {
		buff.WriteString(cp.mspID)
		buff.WriteString(".")
		if cp.role != nil {
			buff.WriteString(fmt.Sprintf("%v", cp.role.Role))
		}
		if cp.ou != nil {
			buff.WriteString(fmt.Sprintf("%v", cp.ou.OrganizationalUnitIdentifier))
		}
		if i < len(cps)-1 {
			buff.WriteString(", ")
		}
	}
	buff.WriteString("]")
	return buff.String()
}

//
func NewComparablePrincipalSet(set policies.PrincipalSet) ComparablePrincipalSet {
	var res ComparablePrincipalSet
	for _, principal := range set {
		cp := NewComparablePrincipal(principal)
		if cp == nil {
			return nil
		}
		res = append(res, cp)
	}
	return res
}

//
func (cps ComparablePrincipalSet) Clone() ComparablePrincipalSet {
	res := make(ComparablePrincipalSet, len(cps))
	for i, cp := range cps {
		res[i] = cp
	}
	return res
}
