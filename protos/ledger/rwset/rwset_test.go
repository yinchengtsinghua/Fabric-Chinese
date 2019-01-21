
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

package rwset

import (
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTxPvtRwsetTrim(t *testing.T) {
	txpvtrwset := testutilConstructSampleTxPvtRwset(
		[]*testNsColls{
			{ns: "ns-1", colls: []string{"coll-1", "coll-2"}},
			{ns: "ns-2", colls: []string{"coll-3", "coll-4"}},
		},
	)

	txpvtrwset.Remove("ns-1", "coll-1")
	assert.Equal(
		t,
		testutilConstructSampleTxPvtRwset(
			[]*testNsColls{
				{ns: "ns-1", colls: []string{"coll-2"}},
				{ns: "ns-2", colls: []string{"coll-3", "coll-4"}},
			},
		),
		txpvtrwset,
	)

	txpvtrwset.Remove("ns-1", "coll-2")
	assert.Equal(
		t,
		testutilConstructSampleTxPvtRwset(
			[]*testNsColls{
				{ns: "ns-2", colls: []string{"coll-3", "coll-4"}},
			},
		),
		txpvtrwset,
	)
}

func testutilConstructSampleTxPvtRwset(nsCollsList []*testNsColls) *TxPvtReadWriteSet {
	txPvtRwset := &TxPvtReadWriteSet{}
	for _, nsColls := range nsCollsList {
		ns := nsColls.ns
		nsdata := &NsPvtReadWriteSet{
			Namespace:          ns,
			CollectionPvtRwset: []*CollectionPvtReadWriteSet{},
		}
		txPvtRwset.NsPvtRwset = append(txPvtRwset.NsPvtRwset, nsdata)
		for _, coll := range nsColls.colls {
			nsdata.CollectionPvtRwset = append(nsdata.CollectionPvtRwset,
				&CollectionPvtReadWriteSet{
					CollectionName: coll,
					Rwset:          []byte(fmt.Sprintf("pvtrwset-for-%s-%s", ns, coll)),
				},
			)
		}
	}
	return txPvtRwset
}

type testNsColls struct {
	ns    string
	colls []string
}
