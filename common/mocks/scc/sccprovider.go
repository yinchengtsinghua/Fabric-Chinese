
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
*/


package scc

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	lm "github.com/hyperledger/fabric/common/mocks/ledger"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger"
)

type MocksccProviderFactory struct {
	Qe                    *lm.MockQueryExecutor
	QErr                  error
	ApplicationConfigRv   channelconfig.Application
	ApplicationConfigBool bool
	PolicyManagerRv       policies.Manager
	PolicyManagerBool     bool
}

func (c *MocksccProviderFactory) NewSystemChaincodeProvider() sysccprovider.SystemChaincodeProvider {
	return &MocksccProviderImpl{
		Qe:                    c.Qe,
		QErr:                  c.QErr,
		ApplicationConfigRv:   c.ApplicationConfigRv,
		ApplicationConfigBool: c.ApplicationConfigBool,
		PolicyManagerBool:     c.PolicyManagerBool,
		PolicyManagerRv:       c.PolicyManagerRv,
	}
}

type MocksccProviderImpl struct {
	Qe                    *lm.MockQueryExecutor
	QErr                  error
	ApplicationConfigRv   channelconfig.Application
	ApplicationConfigBool bool
	PolicyManagerRv       policies.Manager
	PolicyManagerBool     bool
	SysCCMap              map[string]bool
}

func (c *MocksccProviderImpl) IsSysCC(name string) bool {
	if c.SysCCMap != nil {
		return c.SysCCMap[name]
	}

	return (name == "lscc") || (name == "escc") || (name == "vscc") || (name == "notext")
}

func (c *MocksccProviderImpl) IsSysCCAndNotInvokableCC2CC(name string) bool {
	return (name == "escc") || (name == "vscc")
}

func (c *MocksccProviderImpl) IsSysCCAndNotInvokableExternal(name string) bool {
	return (name == "escc") || (name == "vscc") || (name == "notext")
}

func (c *MocksccProviderImpl) GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error) {
	return c.Qe, c.QErr
}

func (c *MocksccProviderImpl) GetApplicationConfig(cid string) (channelconfig.Application, bool) {
	return c.ApplicationConfigRv, c.ApplicationConfigBool
}

func (c *MocksccProviderImpl) PolicyManager(channelID string) (policies.Manager, bool) {
	return c.PolicyManagerRv, c.PolicyManagerBool
}
