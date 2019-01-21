
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


package policies

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
//
	PathSeparator = "/"

//
	ChannelPrefix = "Channel"

//applicationPrefix用于标准应用程序策略路径的路径
	ApplicationPrefix = "Application"

//orderPrefix用于标准排序器策略路径的路径中
	OrdererPrefix = "Orderer"

//channel readers是频道的读卡器策略的标签（包括订购者和应用程序读卡器）
	ChannelReaders = PathSeparator + ChannelPrefix + PathSeparator + "Readers"

//ChannelWriters是频道的写入程序策略的标签（包括订购程序和应用程序写入程序）
	ChannelWriters = PathSeparator + ChannelPrefix + PathSeparator + "Writers"

//channelApplicationReaders是频道的应用程序读取器策略的标签
	ChannelApplicationReaders = PathSeparator + ChannelPrefix + PathSeparator + ApplicationPrefix + PathSeparator + "Readers"

//channelApplicationWriters是频道的应用程序写入程序策略的标签
	ChannelApplicationWriters = PathSeparator + ChannelPrefix + PathSeparator + ApplicationPrefix + PathSeparator + "Writers"

//
	ChannelApplicationAdmins = PathSeparator + ChannelPrefix + PathSeparator + ApplicationPrefix + PathSeparator + "Admins"

//
	BlockValidation = PathSeparator + ChannelPrefix + PathSeparator + OrdererPrefix + PathSeparator + "BlockValidation"
)

var logger = flogging.MustGetLogger("policies")

//Principalset是mspprincipals的集合
type PrincipalSet []*msp.MSPPrincipal

//Principalsets聚合Principalsets
type PrincipalSets []PrincipalSet

//
func (psSets PrincipalSets) ContainingOnly(f func(*msp.MSPPrincipal) bool) PrincipalSets {
	var res PrincipalSets
	for _, set := range psSets {
		if !set.ContainingOnly(f) {
			continue
		}
		res = append(res, set)
	}
	return res
}

//
//满足给定谓词的
func (ps PrincipalSet) ContainingOnly(f func(*msp.MSPPrincipal) bool) bool {
	for _, principal := range ps {
		if !f(principal) {
			return false
		}
	}
	return true
}

//uniqueset返回由principalset诱导的直方图
func (ps PrincipalSet) UniqueSet() map[*msp.MSPPrincipal]int {
//创建一个包含mspprincipals并对其进行计数的柱状图
	histogram := make(map[struct {
		cls       int32
		principal string
	}]int)
//现在，填充柱状图
	for _, principal := range ps {
		key := struct {
			cls       int32
			principal string
		}{
			cls:       int32(principal.PrincipalClassification),
			principal: string(principal.Principal),
		}
		histogram[key]++
	}
//
	res := make(map[*msp.MSPPrincipal]int)
	for principal, count := range histogram {
		res[&msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_Classification(principal.cls),
			Principal:               []byte(principal.principal),
		}] = count
	}
	return res
}

//
type Policy interface {
//Evaluate获取一组SignedData并评估该组签名是否满足策略
	Evaluate(signatureSet []*cb.SignedData) error
}

//
type InquireablePolicy interface {
//satisfiedby返回一部分原则集，其中每个原则集
//满足政策要求。
	SatisfiedBy() []PrincipalSet
}

//
type Manager interface {
//
	GetPolicy(id string) (Policy, bool)

//管理器返回给定路径的子策略管理器以及它是否存在
	Manager(path []string) (Manager, bool)
}

//
type Provider interface {
//new policy基于策略字节创建新策略
	NewPolicy(data []byte) (Policy, proto.Message, error)
}

//
//
type ChannelPolicyManagerGetter interface {
//返回与传递的通道关联的策略管理器
//
	Manager(channelID string) (Manager, bool)
}

//managerimpl是manager和configtx.configHandler的实现
//通常，它只应作为configtx.configmanager的impl引用。
type ManagerImpl struct {
path     string //组级路径
	policies map[string]Policy
	managers map[string]*ManagerImpl
}

//new managerimpl使用给定的cryptoHelper创建新的managerimpl
func NewManagerImpl(path string, providers map[int32]Provider, root *cb.ConfigGroup) (*ManagerImpl, error) {
	var err error
	_, ok := providers[int32(cb.Policy_IMPLICIT_META)]
	if ok {
		logger.Panicf("ImplicitMetaPolicy type must be provider by the policy manager")
	}

	managers := make(map[string]*ManagerImpl)

	for groupName, group := range root.Groups {
		managers[groupName], err = NewManagerImpl(path+PathSeparator+groupName, providers, group)
		if err != nil {
			return nil, err
		}
	}

	policies := make(map[string]Policy)
	for policyName, configPolicy := range root.Policies {
		policy := configPolicy.Policy
		if policy == nil {
			return nil, fmt.Errorf("policy %s at path %s was nil", policyName, path)
		}

		var cPolicy Policy

		if policy.Type == int32(cb.Policy_IMPLICIT_META) {
			imp, err := newImplicitMetaPolicy(policy.Value, managers)
			if err != nil {
				return nil, errors.Wrapf(err, "implicit policy %s at path %s did not compile", policyName, path)
			}
			cPolicy = imp
		} else {
			provider, ok := providers[int32(policy.Type)]
			if !ok {
				return nil, fmt.Errorf("policy %s at path %s has unknown policy type: %v", policyName, path, policy.Type)
			}

			var err error
			cPolicy, _, err = provider.NewPolicy(policy.Value)
			if err != nil {
				return nil, errors.Wrapf(err, "policy %s at path %s did not compile", policyName, path)
			}
		}

		policies[policyName] = cPolicy

		logger.Debugf("Proposed new policy %s for %s", policyName, path)
	}

	for groupName, manager := range managers {
		for policyName, policy := range manager.policies {
			policies[groupName+PathSeparator+policyName] = policy
		}
	}

	return &ManagerImpl{
		path:     path,
		policies: policies,
		managers: managers,
	}, nil
}

type rejectPolicy string

func (rp rejectPolicy) Evaluate(signedData []*cb.SignedData) error {
	return fmt.Errorf("No such policy: '%s'", rp)
}

//管理器返回给定路径的子策略管理器以及它是否存在
func (pm *ManagerImpl) Manager(path []string) (Manager, bool) {
	logger.Debugf("Manager %s looking up path %v", pm.path, path)
	for manager := range pm.managers {
		logger.Debugf("Manager %s has managers %s", pm.path, manager)
	}
	if len(path) == 0 {
		return pm, true
	}

	m, ok := pm.managers[path[0]]
	if !ok {
		return nil, false
	}

	return m.Manager(path[1:])
}

type policyLogger struct {
	policy     Policy
	policyName string
}

func (pl *policyLogger) Evaluate(signatureSet []*cb.SignedData) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("== Evaluating %T Policy %s ==", pl.policy, pl.policyName)
		defer logger.Debugf("== Done Evaluating %T Policy %s", pl.policy, pl.policyName)
	}

	err := pl.policy.Evaluate(signatureSet)
	if err != nil {
		logger.Debugf("Signature set did not satisfy policy %s", pl.policyName)
	} else {
		logger.Debugf("Signature set satisfies policy %s", pl.policyName)
	}
	return err
}

//
func (pm *ManagerImpl) GetPolicy(id string) (Policy, bool) {
	if id == "" {
		logger.Errorf("Returning dummy reject all policy because no policy ID supplied")
		return rejectPolicy(id), false
	}
	var relpath string

	if strings.HasPrefix(id, PathSeparator) {
		if !strings.HasPrefix(id, PathSeparator+pm.path) {
			logger.Debugf("Requested absolute policy %s from %s, returning rejectAll", id, pm.path)
			return rejectPolicy(id), false
		}
//去掉前斜杠、路径和后斜杠
		relpath = id[1+len(pm.path)+1:]
	} else {
		relpath = id
	}

	policy, ok := pm.policies[relpath]
	if !ok {
		logger.Debugf("Returning dummy reject all policy because %s could not be found in %s/%s", id, pm.path, relpath)
		return rejectPolicy(relpath), false
	}

	return &policyLogger{
		policy:     policy,
		policyName: PathSeparator + pm.path + PathSeparator + relpath,
	}, true
}
