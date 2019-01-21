
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


package kvrwset

import (
	"io/ioutil"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/stretchr/testify/assert"
)

const (
	binaryTestFileName = "kvrwsetV1ProtoBytes"
)

//如果在最新版本中声明了“kvrwset”消息，则testkvrwsetv1backwardCompatible将通过
//能够取消标记中声明的“kvrwset”proto消息生成的protobytes
//V1.0。这是为了确保任何不兼容的更改不会被忽略。
func TestKVRWSetV1BackwardCompatible(t *testing.T) {
	protoBytes, err := ioutil.ReadFile(binaryTestFileName)
	assert.NoError(t, err)
	kvrwset1 := &kvrwset.KVRWSet{}
	assert.NoError(t, proto.Unmarshal(protoBytes, kvrwset1))
	kvrwset2 := constructSampleKVRWSet()
	t.Logf("kvrwset1=%s, kvrwset2=%s", spew.Sdump(kvrwset1), spew.Sdump(kvrwset2))
	assert.Equal(t, kvrwset2, kvrwset1)
}

//testpreparebinarifilesamplekvrwsetv1为kvrwset构造一个协议消息，并将其字节封送到文件“kvrwsetvprotobytes”。
//此代码应该在结构版本1.0上运行，以便生成在v1中声明的proto消息的示例文件
//为了在v1代码上调用此函数，请将其复制到v1代码上，将第一个字母设为“t”，最后调用此函数。
//使用Golang测试框架
func testPrepareBinaryFileSampleKVRWSetV1(t *testing.T) {
	b, err := proto.Marshal(constructSampleKVRWSet())
	assert.NoError(t, err)
	assert.NoError(t, ioutil.WriteFile(binaryTestFileName, b, 0775))
}

func constructSampleKVRWSet() *kvrwset.KVRWSet {
	rqi1 := &kvrwset.RangeQueryInfo{StartKey: "k0", EndKey: "k9", ItrExhausted: true}
	rqi1.SetRawReads([]*kvrwset.KVRead{
		{Key: "k1", Version: &kvrwset.Version{BlockNum: 1, TxNum: 1}},
		{Key: "k2", Version: &kvrwset.Version{BlockNum: 1, TxNum: 2}},
	})

	rqi2 := &kvrwset.RangeQueryInfo{StartKey: "k00", EndKey: "k90", ItrExhausted: true}
	rqi2.SetMerkelSummary(&kvrwset.QueryReadsMerkleSummary{MaxDegree: 5, MaxLevel: 4, MaxLevelHashes: [][]byte{[]byte("Hash-1"), []byte("Hash-2")}})
	return &kvrwset.KVRWSet{
		Reads:            []*kvrwset.KVRead{{Key: "key1", Version: &kvrwset.Version{BlockNum: 1, TxNum: 1}}},
		RangeQueriesInfo: []*kvrwset.RangeQueryInfo{rqi1, rqi2},
		Writes:           []*kvrwset.KVWrite{{Key: "key2", IsDelete: false, Value: []byte("value2")}},
	}
}
