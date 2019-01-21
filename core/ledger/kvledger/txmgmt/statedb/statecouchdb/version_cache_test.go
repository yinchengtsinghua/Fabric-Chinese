
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


package statecouchdb

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/stretchr/testify/assert"
)

func TestVersionCache(t *testing.T) {
	verCache := newVersionCache()
	ver1 := version.NewHeight(1, 1)
	ver2 := version.NewHeight(2, 2)
	verCache.setVerAndRev("ns1", "key1", version.NewHeight(1, 1), "rev1")
	verCache.setVerAndRev("ns2", "key2", version.NewHeight(2, 2), "rev2")

	ver, found := verCache.getVersion("ns1", "key1")
	assert.True(t, found)
	assert.Equal(t, ver1, ver)

	ver, found = verCache.getVersion("ns2", "key2")
	assert.True(t, found)
	assert.Equal(t, ver2, ver)

	ver, found = verCache.getVersion("ns1", "key3")
	assert.False(t, found)
	assert.Nil(t, ver)

	ver, found = verCache.getVersion("ns3", "key4")
	assert.False(t, found)
	assert.Nil(t, ver)
}
