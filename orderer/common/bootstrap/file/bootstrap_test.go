
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


package file

import (
	"bytes"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
)

const file = "./abc"

func TestNoFile(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("TestNoFile should have panicked")
		}
	}()

	helper := New(file)
	_ = helper.GenesisBlock()

} //试验材料

func TestBadBlock(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("TestBadBlock should have panicked")
		}
	}()

	testFile, _ := os.Create(file)
	defer os.Remove(file)
	testFile.Write([]byte("abc"))
	testFile.Close()
	helper := New(file)
	_ = helper.GenesisBlock()
} //测试块

func TestGenesisBlock(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestGenesisBlock: unexpected panic")
		}
	}()

	header := &cb.BlockHeader{
		Number:       0,
		PreviousHash: nil,
		DataHash:     []byte("abc"),
	}
	data := &cb.BlockData{
		Data: [][]byte{[]byte("abc")},
	}
	metadata := &cb.BlockMetadata{
		Metadata: [][]byte{[]byte("abc")},
	}
	block := &cb.Block{
		Header:   header,
		Data:     data,
		Metadata: metadata,
	}
	marshalledBlock, _ := proto.Marshal(block)

	testFile, _ := os.Create(file)
	defer os.Remove(file)
	testFile.Write(marshalledBlock)
	testFile.Close()

	helper := New(file)
	outBlock := helper.GenesisBlock()

	outHeader := outBlock.Header
	if outHeader.Number != 0 || outHeader.PreviousHash != nil || !bytes.Equal(outHeader.DataHash, []byte("abc")) {
		t.Errorf("block header not read correctly. Got %+v\n . Should have been %+v\n", outHeader, header)
	}
	outData := outBlock.Data
	if len(outData.Data) != 1 && !bytes.Equal(outData.Data[0], []byte("abc")) {
		t.Errorf("block data not read correctly. Got %+v\n . Should have been %+v\n", outData, data)
	}
	outMeta := outBlock.Metadata
	if len(outMeta.Metadata) != 1 && !bytes.Equal(outMeta.Metadata[0], []byte("abc")) {
		t.Errorf("Metadata data not read correctly. Got %+v\n . Should have been %+v\n", outMeta, metadata)
	}
} //测试生成模块
