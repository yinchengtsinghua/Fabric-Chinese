
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
	"github.com/hyperledger/fabric/core/common/ccprovider"
)

//deployysccs是系统链码的钩子，系统链码在结构中注册。
//注意，chaincode必须像用户chaincode一样部署和启动。
func (p *Provider) DeploySysCCs(chainID string, ccp ccprovider.ChaincodeProvider) {
	for _, sysCC := range p.SysCCs {
		deploySysCC(chainID, ccp, sysCC)
	}
}

//DEDEPLoySyscs用于单元测试中，在
//在同一进程中重新启动它们。这允许系统干净启动。
//在同一过程中
func (p *Provider) DeDeploySysCCs(chainID string, ccp ccprovider.ChaincodeProvider) {
	for _, sysCC := range p.SysCCs {
		deDeploySysCC(chainID, ccp, sysCC)
	}
}
