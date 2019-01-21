
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

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


package blockledger_test

import (
	. "github.com/hyperledger/fabric/common/ledger/blockledger"
	ramledger "github.com/hyperledger/fabric/common/ledger/blockledger/ram"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
)

func init() {
	testables = append(testables, &ramLedgerTestEnv{})
}

type ramledgerTestFactory struct{}

type ramLedgerTestEnv struct{}

func (env *ramLedgerTestEnv) Initialize() (ledgerTestFactory, error) {
	return &ramledgerTestFactory{}, nil
}

func (env *ramLedgerTestEnv) Name() string {
	return "ramledger"
}

func (env *ramledgerTestFactory) Destroy() error {
	return nil
}

func (env *ramledgerTestFactory) Persistent() bool {
	return false
}

func (env *ramledgerTestFactory) New() (Factory, ReadWriter) {
	historySize := 10
	rlf := ramledger.New(historySize)
	rl, err := rlf.GetOrCreate(genesisconfig.TestChainID)
	if err != nil {
		panic(err)
	}
	err = rl.Append(genesisBlock)
	if err != nil {
		panic(err)
	}
	return rlf, rl
}
