
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


package pvtdatastorage

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/stretchr/testify/assert"
)

//store env提供用于测试的store env
type StoreEnv struct {
	t                 testing.TB
	TestStoreProvider Provider
	TestStore         Store
	ledgerid          string
	btlPolicy         pvtdatapolicy.BTLPolicy
}

//newteststorenv构造用于测试的storeenv
func NewTestStoreEnv(t *testing.T, ledgerid string, btlPolicy pvtdatapolicy.BTLPolicy) *StoreEnv {
	removeStorePath(t)
	assert := assert.New(t)
	testStoreProvider := NewProvider()
	testStore, err := testStoreProvider.OpenStore(ledgerid)
	testStore.Init(btlPolicy)
	assert.NoError(err)
	return &StoreEnv{t, testStoreProvider, testStore, ledgerid, btlPolicy}
}

//关闭并打开存储提供程序
func (env *StoreEnv) CloseAndReopen() {
	var err error
	env.TestStoreProvider.Close()
	env.TestStoreProvider = NewProvider()
	env.TestStore, err = env.TestStoreProvider.OpenStore(env.ledgerid)
	env.TestStore.Init(env.btlPolicy)
	assert.NoError(env.t, err)
}

//清理测试后清理存储环境
func (env *StoreEnv) Cleanup() {
//env.teststoreprovider.close（）。
	removeStorePath(env.t)
}

func removeStorePath(t testing.TB) {
	dbPath := ledgerconfig.GetPvtdataStorePath()
	if err := os.RemoveAll(dbPath); err != nil {
		t.Fatalf("Err: %s", err)
		t.FailNow()
	}
}
