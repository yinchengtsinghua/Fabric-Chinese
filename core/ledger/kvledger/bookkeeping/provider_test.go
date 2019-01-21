
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


package bookkeeping

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/bookkeeping")
	os.Exit(m.Run())
}

func TestProvider(t *testing.T) {
	testEnv := NewTestEnv(t)
	defer testEnv.Cleanup()
	p := testEnv.TestProvider
	db := p.GetDBHandle("TestLedger", PvtdataExpiry)
	assert.NoError(t, db.Put([]byte("key"), []byte("value"), true))
	val, err := db.Get([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("value"), val)
}
