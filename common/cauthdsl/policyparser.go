
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


package cauthdsl

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/utils"
)

//门值
const (
	GateAnd   = "And"
	GateOr    = "Or"
	GateOutOf = "OutOf"
)

//主体的角色值
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleClient = "client"
	RolePeer   = "peer"
//roleoorder=“order”待办事项
)

var (
	regex = regexp.MustCompile(
		fmt.Sprintf("^([[:alnum:].-]+)([.])(%s|%s|%s|%s)$",
			RoleAdmin, RoleMember, RoleClient, RolePeer),
	)
	regexErr = regexp.MustCompile("^No parameter '([^']+)' found[.]$")
)

//一个存根函数-它返回与传递的字符串相同的字符串。
//这将通过第二次/第三次传递进行评估，以转换为协议策略
func outof(args ...interface{}) (interface{}, error) {
	toret := "outof("
	if len(args) < 2 {
		return nil, fmt.Errorf("Expected at least two arguments to NOutOf. Given %d", len(args))
	}

	arg0 := args[0]
//政府只将所有数字视为float64。但和/或可以传递int/string。允许int/string以实现调用者的灵活性
	if n, ok := arg0.(float64); ok {
		toret += strconv.Itoa(int(n))
	} else if n, ok := arg0.(int); ok {
		toret += strconv.Itoa(n)
	} else if n, ok := arg0.(string); ok {
		toret += n
	} else {
		return nil, fmt.Errorf("Unexpected type %s", reflect.TypeOf(arg0))
	}

	for _, arg := range args[1:] {
		toret += ", "
		switch t := arg.(type) {
		case string:
			if regex.MatchString(t) {
				toret += "'" + t + "'"
			} else {
				toret += t
			}
		default:
			return nil, fmt.Errorf("Unexpected type %s", reflect.TypeOf(arg))
		}
	}
	return toret + ")", nil
}

func and(args ...interface{}) (interface{}, error) {
	args = append([]interface{}{len(args)}, args...)
	return outof(args...)
}

func or(args ...interface{}) (interface{}, error) {
	args = append([]interface{}{1}, args...)
	return outof(args...)
}

func firstPass(args ...interface{}) (interface{}, error) {
	toret := "outof(ID"
	for _, arg := range args {
		toret += ", "
		switch t := arg.(type) {
		case string:
			if regex.MatchString(t) {
				toret += "'" + t + "'"
			} else {
				toret += t
			}
		case float32:
		case float64:
			toret += strconv.Itoa(int(t))
		default:
			return nil, fmt.Errorf("Unexpected type %s", reflect.TypeOf(arg))
		}
	}

	return toret + ")", nil
}

func secondPass(args ...interface{}) (interface{}, error) {
 /*一般的健康检查，我们预计至少3个参数*/
	if len(args) < 3 {
		return nil, fmt.Errorf("At least 3 arguments expected, got %d", len(args))
	}

 /*获取第一个参数，我们期望它是上下文*/
	var ctx *context
	switch v := args[0].(type) {
	case *context:
		ctx = v
	default:
		return nil, fmt.Errorf("Unrecognized type, expected the context, got %s", reflect.TypeOf(args[0]))
	}

 /*得到第二个参数，我们期望一个整数告诉我们
    我们还剩下多少*/

	var t int
	switch arg := args[1].(type) {
	case float64:
		t = int(arg)
	default:
		return nil, fmt.Errorf("Unrecognized type, expected a number, got %s", reflect.TypeOf(args[1]))
	}

 /*从n中取出t中的n*/
	var n int = len(args) - 2

 /*健全性检查-t应为正，允许等于n+1，但不允许超过n+1。*/
	if t < 0 || t > n+1 {
		return nil, fmt.Errorf("Invalid t-out-of-n predicate, t %d, n %d", t, n)
	}

	policies := make([]*common.SignaturePolicy, 0)

 /*处理其余参数*/
	for _, principal := range args[2:] {
		switch t := principal.(type) {
  /*如果它是一个字符串，我们希望它的形式为
     <MSPYID><role>，其中msp_id是msp标识符
     角色可以是成员、管理员、客户机、对等机或订单机*/

		case string:
   /*拆分字符串*/
			subm := regex.FindAllStringSubmatch(t, -1)
			if subm == nil || len(subm) != 1 || len(subm[0]) != 4 {
				return nil, fmt.Errorf("Error parsing principal %s", t)
			}

   /*找到合适的角色*/
			var r msp.MSPRole_MSPRoleType
			switch subm[0][3] {
			case RoleMember:
				r = msp.MSPRole_MEMBER
			case RoleAdmin:
				r = msp.MSPRole_ADMIN
			case RoleClient:
				r = msp.MSPRole_CLIENT
			case RolePeer:
				r = msp.MSPRole_PEER
			default:
				return nil, fmt.Errorf("Error parsing role %s", t)
			}

   /*建立我们被告知的校长*/
			p := &msp.MSPPrincipal{
				PrincipalClassification: msp.MSPPrincipal_ROLE,
				Principal:               utils.MarshalOrPanic(&msp.MSPRole{MspIdentifier: subm[0][1], Role: r})}
			ctx.principals = append(ctx.principals, p)

   /*创建需要签名的签名策略
      我们刚建立的校长*/

			dapolicy := SignedBy(int32(ctx.IDNum))
			policies = append(policies, dapolicy)

   /*增加标识计数器。注意这是
      次优，因为我们不重用身份。我们
      可以很容易地将它们除尘，使这只小狗
      更小。不过现在还可以*/

//TODO:消除重复主体
			ctx.IDNum++

  /*如果我们已经有了一个政策，那就附加它吧*/
		case *common.SignaturePolicy:
			policies = append(policies, t)

		default:
			return nil, fmt.Errorf("Unrecognized type, expected a principal or a policy, got %s", reflect.TypeOf(principal))
		}
	}

	return NOutOf(int32(t), policies), nil
}

type context struct {
	IDNum      int
	principals []*msp.MSPPrincipal
}

func newContext() *context {
	return &context{IDNum: 0, principals: make([]*msp.MSPPrincipal, 0)}
}

//fromString接受策略的字符串表示，
//分析它并返回一个SignaturePolicyInvelope
//执行该策略。支持的语言如下：
//
//门（P[，P]）
//
//在哪里？
//-门是“和”或“或”
//-p是主体或对gate的另一个嵌套调用
//
//委托人的定义如下：
//
//组织角色
//
//在哪里？
//-org是一个字符串（表示MSP标识符）
//-role取表示的任何rolexxx常量的值
//所需角色
func FromString(policy string) (*common.SignaturePolicyEnvelope, error) {
//首先，我们将和/或业务转化为外部业务
	intermediate, err := govaluate.NewEvaluableExpressionWithFunctions(
		policy, map[string]govaluate.ExpressionFunction{
			GateAnd:                    and,
			strings.ToLower(GateAnd):   and,
			strings.ToUpper(GateAnd):   and,
			GateOr:                     or,
			strings.ToLower(GateOr):    or,
			strings.ToUpper(GateOr):    or,
			GateOutOf:                  outof,
			strings.ToLower(GateOutOf): outof,
			strings.ToUpper(GateOutOf): outof,
		},
	)
	if err != nil {
		return nil, err
	}

	intermediateRes, err := intermediate.Evaluate(map[string]interface{}{})
	if err != nil {
//尝试产生有意义的错误
		if regexErr.MatchString(err.Error()) {
			sm := regexErr.FindStringSubmatch(err.Error())
			if len(sm) == 2 {
				return nil, fmt.Errorf("unrecognized token '%s' in policy string", sm[1])
			}
		}

		return nil, err
	}
	resStr, ok := intermediateRes.(string)
	if !ok {
		return nil, fmt.Errorf("invalid policy string '%s'", policy)
	}

//我们还需要两张通行证。第一次传球就多了一次
//每个outof调用的参数ID。这是
//要求，因为政府没有提供背景的手段
//到用户实现的函数，而不是通过参数。
//我们需要这个论点，因为我们需要一个全球性的地方
//我们把政策要求的身份
	exp, err := govaluate.NewEvaluableExpressionWithFunctions(resStr, map[string]govaluate.ExpressionFunction{"outof": firstPass})
	if err != nil {
		return nil, err
	}

	res, err := exp.Evaluate(map[string]interface{}{})
	if err != nil {
//尝试产生有意义的错误
		if regexErr.MatchString(err.Error()) {
			sm := regexErr.FindStringSubmatch(err.Error())
			if len(sm) == 2 {
				return nil, fmt.Errorf("unrecognized token '%s' in policy string", sm[1])
			}
		}

		return nil, err
	}
	resStr, ok = res.(string)
	if !ok {
		return nil, fmt.Errorf("invalid policy string '%s'", policy)
	}

	ctx := newContext()
	parameters := make(map[string]interface{}, 1)
	parameters["ID"] = ctx

	exp, err = govaluate.NewEvaluableExpressionWithFunctions(resStr, map[string]govaluate.ExpressionFunction{"outof": secondPass})
	if err != nil {
		return nil, err
	}

	res, err = exp.Evaluate(parameters)
	if err != nil {
//尝试产生有意义的错误
		if regexErr.MatchString(err.Error()) {
			sm := regexErr.FindStringSubmatch(err.Error())
			if len(sm) == 2 {
				return nil, fmt.Errorf("unrecognized token '%s' in policy string", sm[1])
			}
		}

		return nil, err
	}
	rule, ok := res.(*common.SignaturePolicy)
	if !ok {
		return nil, fmt.Errorf("invalid policy string '%s'", policy)
	}

	p := &common.SignaturePolicyEnvelope{
		Identities: ctx.principals,
		Version:    0,
		Rule:       rule,
	}

	return p, nil
}
