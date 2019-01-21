
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


package tests

import (
	"testing"
)

func TestRebuildComponents(t *testing.T) {
	env := newEnv(defaultConfig, t)
	defer env.cleanup()

	h1, h2 := newTestHelperCreateLgr("ledger1", t), newTestHelperCreateLgr("ledger2", t)
	dataHelper := newSampleDataHelper(t)

	dataHelper.populateLedger(h1)
	dataHelper.populateLedger(h2)

	dataHelper.verifyLedgerContent(h1)
	dataHelper.verifyLedgerContent(h2)

	t.Run("rebuild only statedb",
		func(t *testing.T) {
			env.closeAllLedgersAndDrop(rebuildableStatedb)
			h1, h2 := newTestHelperOpenLgr("ledger1", t), newTestHelperOpenLgr("ledger2", t)
			dataHelper.verifyLedgerContent(h1)
			dataHelper.verifyLedgerContent(h2)
		},
	)

	t.Run("rebuild statedb and config history",
		func(t *testing.T) {
			env.closeAllLedgersAndDrop(rebuildableStatedb + rebuildableConfigHistory)
			h1, h2 := newTestHelperOpenLgr("ledger1", t), newTestHelperOpenLgr("ledger2", t)
			dataHelper.verifyLedgerContent(h1)
			dataHelper.verifyLedgerContent(h2)
		},
	)

	t.Run("rebuild statedb and block index",
		func(t *testing.T) {
			env.closeAllLedgersAndDrop(rebuildableStatedb + rebuildableBlockIndex)
			h1, h2 := newTestHelperOpenLgr("ledger1", t), newTestHelperOpenLgr("ledger2", t)
			dataHelper.verifyLedgerContent(h1)
			dataHelper.verifyLedgerContent(h2)
		},
	)
}
