
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package cauthdsl

import (
	"reflect"
	"testing"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func TestOutOf1(t *testing.T) {
	p1, err := FromString("OutOf(1, 'A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*common.SignaturePolicy{SignedBy(0), SignedBy(1)}),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestOutOf2(t *testing.T) {
	p1, err := FromString("OutOf(2, 'A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(2, []*common.SignaturePolicy{SignedBy(0), SignedBy(1)}),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestAnd(t *testing.T) {
	p1, err := FromString("AND('A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       And(SignedBy(0), SignedBy(1)),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestAndClientPeerOrderer(t *testing.T) {
	p1, err := FromString("AND('A.client', 'B.peer')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_CLIENT, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_PEER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       And(SignedBy(0), SignedBy(1)),
		Identities: principals,
	}

	assert.True(t, reflect.DeepEqual(p1, p2))

}

func TestOr(t *testing.T) {
	p1, err := FromString("OR('A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       Or(SignedBy(0), SignedBy(1)),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestComplex1(t *testing.T) {
	p1, err := FromString("OR('A.member', AND('B.member', 'C.member'))")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "C"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       Or(SignedBy(2), And(SignedBy(0), SignedBy(1))),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestComplex2(t *testing.T) {
	p1, err := FromString("OR(AND('A.member', 'B.member'), OR('C.admin', 'D.member'))")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_ADMIN, MspIdentifier: "C"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "D"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       Or(And(SignedBy(0), SignedBy(1)), Or(SignedBy(2), SignedBy(3))),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestMSPIDWIthSpecialChars(t *testing.T) {
	p1, err := FromString("OR('MSP.member', 'MSP.WITH.DOTS.member', 'MSP-WITH-DASHES.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "MSP"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "MSP.WITH.DOTS"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "MSP-WITH-DASHES"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*common.SignaturePolicy{SignedBy(0), SignedBy(1), SignedBy(2)}),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestBadStringsNoPanic(t *testing.T) {
_, err := FromString("OR('A.member', Bmember)") //第一次计算（）后出错
	assert.EqualError(t, err, "unrecognized token 'Bmember' in policy string")

_, err = FromString("OR('A.member', 'Bmember')") //第二个值（）后出错
	assert.EqualError(t, err, "unrecognized token 'Bmember' in policy string")

_, err = FromString(`OR('A.member', '\'Bmember\'')`) //第三个值（）后出错
	assert.EqualError(t, err, "unrecognized token 'Bmember' in policy string")
}

func TestOutOfNumIsString(t *testing.T) {
	p1, err := FromString("OutOf('1', 'A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*common.SignaturePolicy{SignedBy(0), SignedBy(1)}),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestOutOfErrorCase(t *testing.T) {
p1, err1 := FromString("") //1st newEvaluableExpressionWithFunctions（）返回错误
	assert.Nil(t, p1)
	assert.EqualError(t, err1, "Unexpected end of expression")

p2, err2 := FromString("OutOf(1)") //out of（）if len（args）<2
	assert.Nil(t, p2)
	assert.EqualError(t, err2, "Expected at least two arguments to NOutOf. Given 1")

p3, err3 := FromString("OutOf(true, 'A.member')") //outof（）其他。第一个参数不是float、int、string
	assert.Nil(t, p3)
	assert.EqualError(t, err3, "Unexpected type bool")

p4, err4 := FromString("OutOf(1, 2)") //oufof（）开关默认值。第二个参数不是字符串。
	assert.Nil(t, p4)
	assert.EqualError(t, err4, "Unexpected type float64")

p5, err5 := FromString("OutOf(1, 'true')") //firstpass（）开关默认值
	assert.Nil(t, p5)
	assert.EqualError(t, err5, "Unexpected type bool")

p6, err6 := FromString(`OutOf('\'\\\'A\\\'\'', 'B.member')`) //secondpass（）开关参数[1]。（类型）默认值
	assert.Nil(t, p6)
	assert.EqualError(t, err6, "Unrecognized type, expected a number, got string")

p7, err7 := FromString(`OutOf(1, '\'1\'')`) //secondpass（）开关参数[1]。（类型）默认值
	assert.Nil(t, p7)
	assert.EqualError(t, err7, "Unrecognized type, expected a principal or a policy, got float64")

p8, err8 := FromString(`''`) //第二个newEvaluateExpressionWithFunction（）返回错误
	assert.Nil(t, p8)
	assert.EqualError(t, err8, "Unexpected end of expression")

p9, err9 := FromString(`'\'\''`) //第3个newEvaluateExpressionWithFunction（）返回错误
	assert.Nil(t, p9)
	assert.EqualError(t, err9, "Unexpected end of expression")
}

func TestBadStringBeforeFAB11404_ThisCanDeleteAfterFAB11404HasMerged(t *testing.T) {
s1 := "1" //字符串中的Ineger
	p1, err1 := FromString(s1)
	assert.Nil(t, p1)
	assert.EqualError(t, err1, `invalid policy string '1'`)

s2 := "'1'" //字符串中带引号的ineger
	p2, err2 := FromString(s2)
	assert.Nil(t, p2)
	assert.EqualError(t, err2, `invalid policy string ''1''`)

s3 := `'\'1\''` //字符串中嵌套的带引号的ineger
	p3, err3 := FromString(s3)
	assert.Nil(t, p3)
	assert.EqualError(t, err3, `invalid policy string ''\'1\'''`)
}

func TestSecondPassBoundaryCheck(t *testing.T) {
//检查下边界
//禁止T＜0
	p0, err0 := FromString("OutOf(-1, 'A.member', 'B.member')")
	assert.Nil(t, p0)
	assert.EqualError(t, err0, "Invalid t-out-of-n predicate, t -1, n 2")

//允许t==0:始终满足策略
//没有明确的t=0的用例，但可能有人已经使用了它，所以我们不会将其视为错误。
	p1, err1 := FromString("OutOf(0, 'A.member', 'B.member')")
	assert.NoError(t, err1)
	principals := make([]*msp.MSPPrincipal, 0)
	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})
	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})
	expected1 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(0, []*common.SignaturePolicy{SignedBy(0), SignedBy(1)}),
		Identities: principals,
	}
	assert.Equal(t, expected1, p1)

//检查上边界
//允许t==n+1：不满足政策
//用例：创建不可变的分类帐键
	p2, err2 := FromString("OutOf(3, 'A.member', 'B.member')")
	assert.NoError(t, err2)
	expected2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(3, []*common.SignaturePolicy{SignedBy(0), SignedBy(1)}),
		Identities: principals,
	}
	assert.Equal(t, expected2, p2)

//禁止t>n+1
	p3, err3 := FromString("OutOf(4, 'A.member', 'B.member')")
	assert.Nil(t, p3)
	assert.EqualError(t, err3, "Invalid t-out-of-n predicate, t 4, n 2")
}
