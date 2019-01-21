
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
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/assert"
)

const (
	binaryTestFileName = "rwsetV1ProtoBytes"
)

//如果在最新版本中声明了“rwset”消息，则testrwsetv1backwardCompatible将通过
//能够取消标记中声明的“rwset”proto消息生成的protobytes
//V1.0。这是为了确保任何不兼容的更改不会被忽略。
func TestRWSetV1BackwardCompatible(t *testing.T) {
	protoBytes, err := ioutil.ReadFile(binaryTestFileName)
	assert.NoError(t, err)
	rwset1 := &rwset.TxReadWriteSet{}
	assert.NoError(t, proto.Unmarshal(protoBytes, rwset1))
	rwset2 := constructSampleRWSet()
	t.Logf("rwset1=%s, rwset2=%s", spew.Sdump(rwset1), spew.Sdump(rwset2))
	assert.Equal(t, rwset2, rwset1)
}

//testpreparebinarifilesamplerwsetv1为kvrwset构造一个proto消息，并将其字节封送到文件“rwsetvprotobytes”。
//此代码应该在结构版本1.0上运行，以便生成在v1中声明的proto消息的示例文件
//为了在v1代码上调用此函数，请将其复制到v1代码上，将第一个字母设为“t”，最后调用此函数。
//使用Golang测试框架
func testPrepareBinaryFileSampleRWSetV1(t *testing.T) {
	b, err := proto.Marshal(constructSampleRWSet())
	assert.NoError(t, err)
	assert.NoError(t, ioutil.WriteFile(binaryTestFileName, b, 0775))
}

func constructSampleRWSet() *rwset.TxReadWriteSet {
	rwset1 := &rwset.TxReadWriteSet{}
	rwset1.DataModel = rwset.TxReadWriteSet_KV
	rwset1.NsRwset = []*rwset.NsReadWriteSet{
		{Namespace: "ns-1", Rwset: []byte("ns-1-rwset")},
		{Namespace: "ns-2", Rwset: []byte("ns-2-rwset")},
	}
	return rwset1
}
