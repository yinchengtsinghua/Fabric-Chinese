
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


package pvtdatapolicy

import (
	"math"
	"testing"

	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/mock"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestBTLPolicy(t *testing.T) {
	btlPolicy := testutilSampleBTLPolicy()
	btl1, err := btlPolicy.GetBTL("ns1", "coll1")
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), btl1)

	btl2, err := btlPolicy.GetBTL("ns1", "coll2")
	assert.NoError(t, err)
	assert.Equal(t, uint64(200), btl2)

	btl3, err := btlPolicy.GetBTL("ns1", "coll3")
	assert.NoError(t, err)
	assert.Equal(t, defaultBTL, btl3)

	_, err = btlPolicy.GetBTL("ns1", "coll4")
	_, ok := err.(privdata.NoSuchCollectionError)
	assert.True(t, ok)
}

func TestExpiringBlock(t *testing.T) {
	btlPolicy := testutilSampleBTLPolicy()
	expiringBlk, err := btlPolicy.GetExpiringBlock("ns1", "coll1", 50)
	assert.NoError(t, err)
	assert.Equal(t, uint64(151), expiringBlk)

	expiringBlk, err = btlPolicy.GetExpiringBlock("ns1", "coll2", 50)
	assert.NoError(t, err)
	assert.Equal(t, uint64(251), expiringBlk)

	expiringBlk, err = btlPolicy.GetExpiringBlock("ns1", "coll3", 50)
	assert.NoError(t, err)
	assert.Equal(t, uint64(math.MaxUint64), expiringBlk)

	expiringBlk, err = btlPolicy.GetExpiringBlock("ns1", "coll4", 50)
	_, ok := err.(privdata.NoSuchCollectionError)
	assert.True(t, ok)
}

func testutilSampleBTLPolicy() BTLPolicy {
	ccInfoRetriever := &mock.CollectionInfoProvider{}
	ccInfoRetriever.CollectionInfoStub = func(ccName, collName string) (*common.StaticCollectionConfig, error) {
		collConfig := &common.StaticCollectionConfig{}
		var err error
		switch collName {
		case "coll1":
			collConfig.BlockToLive = 100
		case "coll2":
			collConfig.BlockToLive = 200
		case "coll3":
			collConfig.BlockToLive = 0
		default:
			err = privdata.NoSuchCollectionError{}
		}
		return collConfig, err
	}
	return ConstructBTLPolicy(ccInfoRetriever)
}
