
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
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("lockbasedtxmgr,statevalidator,statebasedval,statecouchdb,valimpl,pvtstatepurgemgmt,confighistory,kvledger=debug")
	os.Exit(m.Run())
}

func TestLedgerAPIs(t *testing.T) {
	env := newEnv(defaultConfig, t)
	defer env.cleanup()

//创建两个分类帐
	h1 := newTestHelperCreateLgr("ledger1", t)
	h2 := newTestHelperCreateLgr("ledger2", t)

//用示例数据填充分类帐
	dataHelper := newSampleDataHelper(t)
	dataHelper.populateLedger(h1)
	dataHelper.populateLedger(h2)

//验证两个分类帐中的内容
	dataHelper.verifyLedgerContent(h1)
	dataHelper.verifyLedgerContent(h2)
}
