
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

package ledger

import (
	"testing"

	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/assert"
)

func TestTxPvtData(t *testing.T) {
	txPvtData := &TxPvtData{}
	assert.False(t, txPvtData.Has("ns", "coll"))

	txPvtData.WriteSet = &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "ns",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "coll-1",
						Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll1"),
					},
					{
						CollectionName: "coll-2",
						Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll2"),
					},
				},
			},
		},
	}

	assert.True(t, txPvtData.Has("ns", "coll-1"))
	assert.True(t, txPvtData.Has("ns", "coll-2"))
	assert.False(t, txPvtData.Has("ns", "coll-3"))
	assert.False(t, txPvtData.Has("ns1", "coll-1"))
}

func TestPvtNsCollFilter(t *testing.T) {
	filter := NewPvtNsCollFilter()
	filter.Add("ns", "coll-1")
	filter.Add("ns", "coll-2")
	assert.True(t, filter.Has("ns", "coll-1"))
	assert.True(t, filter.Has("ns", "coll-2"))
	assert.False(t, filter.Has("ns", "coll-3"))
	assert.False(t, filter.Has("ns1", "coll-3"))
}
