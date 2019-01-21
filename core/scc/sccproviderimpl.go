
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


package scc

import (
	"fmt"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer"
)

//NewProvider创建新的提供程序实例
func NewProvider(pOps peer.Operations, pSup peer.Support, r Registrar) *Provider {
	return &Provider{
		Peer:        pOps,
		PeerSupport: pSup,
		Registrar:   r,
	}
}

//提供程序实现SysCCProvider.SystemChainCodeProvider
type Provider struct {
	Peer        peer.Operations
	PeerSupport peer.Support
	Registrar   Registrar
	SysCCs      []SelfDescribingSysCC
}

//registeryscc向syscc提供程序注册系统链码。
func (p *Provider) RegisterSysCC(scc SelfDescribingSysCC) {
	p.SysCCs = append(p.SysCCs, scc)
	_, err := p.registerSysCC(scc)
	if err != nil {
		sysccLogger.Panicf("Could not register system chaincode: %s", err)
	}
}

//如果提供的链码是系统链码，ISSysCC将返回true。
func (p *Provider) IsSysCC(name string) bool {
	for _, sysCC := range p.SysCCs {
		if sysCC.Name() == name {
			return true
		}
	}
	if isDeprecatedSysCC(name) {
		return true
	}
	return false
}

//如果链码返回，则ISSyCtoNoToNoKabcLC2CC返回true。
//是系统链代码，不能通过调用*调用
//a cc2cc invocation
func (p *Provider) IsSysCCAndNotInvokableCC2CC(name string) bool {
	for _, sysCC := range p.SysCCs {
		if sysCC.Name() == name {
			return !sysCC.InvokableCC2CC()
		}
	}

	if isDeprecatedSysCC(name) {
		return true
	}

	return false
}

//GetQueryExecutorForLedger返回指定通道的查询执行器
func (p *Provider) GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error) {
	l := p.Peer.GetLedger(cid)
	if l == nil {
		return nil, fmt.Errorf("Could not retrieve ledger for channel %s", cid)
	}

	return l.NewQueryExecutor()
}

//如果链码为
//是系统链代码，不能通过调用*调用
//向这位同行提出的建议
func (p *Provider) IsSysCCAndNotInvokableExternal(name string) bool {
	for _, sysCC := range p.SysCCs {
		if sysCC.Name() == name {
			return !sysCC.InvokableExternal()
		}
	}

	if isDeprecatedSysCC(name) {
		return true
	}

	return false
}

//GETAPPLICATIOFIGG返回通道的CONTXXAPPLATION.SARDCONFIGG
//以及应用程序配置是否存在
func (p *Provider) GetApplicationConfig(cid string) (channelconfig.Application, bool) {
	return p.PeerSupport.GetApplicationConfig(cid)
}

//返回与传递的通道关联的策略管理器
//以及策略管理器是否存在
func (p *Provider) PolicyManager(channelID string) (policies.Manager, bool) {
	m := p.Peer.GetPolicyManager(channelID)
	return m, (m != nil)
}

func isDeprecatedSysCC(name string) bool {
	return name == "vscc" || name == "escc"
}
