
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

	"github.com/davecgh/go-spew/spew"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

//test v11 v12测试我们是否能够读取混合格式的数据（对于v11和v12）
//从pvtdata商店。此测试使用了在
//升级测试。这家商店总共有15个街区。一到九号街区没有
//pvt数据是因为，当时的对等代码是v1.0，因此没有pvt数据。块10包含
//来自Peer v1.1的pvtData。块11-13没有pvt数据。块14具有来自对等机v1.2的pvt数据
func TestV11v12(t *testing.T) {
	testWorkingDir := "test-working-dir"
	testutil.CopyDir("testdata/v11_v12/ledgersData", testWorkingDir)
	defer os.RemoveAll(testWorkingDir)

	viper.Set("peer.fileSystemPath", testWorkingDir)
	defer viper.Reset()

	ledgerid := "ch1"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"marbles_private", "collectionMarbles"}:              0,
			{"marbles_private", "collectionMarblePrivateDetails"}: 0,
		},
	)
	p := NewProvider()
	defer p.Close()
	s, err := p.OpenStore(ledgerid)
	assert.NoError(t, err)
	s.Init(btlPolicy)

	for blk := 0; blk < 10; blk++ {
		checkDataNotExists(t, s, blk)
	}
	checkDataExists(t, s, 10)
	for blk := 11; blk < 14; blk++ {
		checkDataNotExists(t, s, blk)
	}
	checkDataExists(t, s, 14)

	_, err = s.GetPvtDataByBlockNum(uint64(15), nil)
	_, ok := err.(*ErrOutOfRange)
	assert.True(t, ok)
}

func checkDataNotExists(t *testing.T, s Store, blkNum int) {
	data, err := s.GetPvtDataByBlockNum(uint64(blkNum), nil)
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func checkDataExists(t *testing.T, s Store, blkNum int) {
	data, err := s.GetPvtDataByBlockNum(uint64(blkNum), nil)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	t.Logf("pvtdata = %s\n", spew.Sdump(data))
}
