
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


package common

import (
	"encoding/asn1"
	"math"
	"testing"

	"github.com/hyperledger/fabric/common/util"
	"github.com/stretchr/testify/assert"
)

func TestBlock(t *testing.T) {
	var block *Block
	assert.Nil(t, block.GetHeader())
	assert.Nil(t, block.GetData())
	assert.Nil(t, block.GetMetadata())

	data := &BlockData{
		Data: [][]byte{{0, 1, 2}},
	}
	block = NewBlock(uint64(0), []byte("datahash"))
	assert.Equal(t, []byte("datahash"), block.Header.PreviousHash, "Incorrect previous hash")
	assert.NotNil(t, block.GetData())
	assert.NotNil(t, block.GetMetadata())
	block.GetHeader().DataHash = data.Hash()

	asn1Bytes, err := asn1.Marshal(asn1Header{
		Number:       int64(uint64(0)),
		DataHash:     data.Hash(),
		PreviousHash: []byte("datahash"),
	})
	headerHash := util.ComputeSHA256(asn1Bytes)
	assert.NoError(t, err)
	assert.Equal(t, asn1Bytes, block.Header.Bytes(), "Incorrect marshaled blockheader bytes")
	assert.Equal(t, headerHash, block.Header.Hash(), "Incorrect blockheader hash")

}

func TestGoodBlockHeaderBytes(t *testing.T) {
	goodBlockHeader := &BlockHeader{
		Number:       1,
		PreviousHash: []byte("foo"),
		DataHash:     []byte("bar"),
	}

_ = goodBlockHeader.Bytes() //不应该惊慌
}

func TestBadBlockHeaderBytes(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("Should have panicked on block number too high to encode as int64")
		}
	}()

	badBlockHeader := &BlockHeader{
		Number:       math.MaxUint64,
		PreviousHash: []byte("foo"),
		DataHash:     []byte("bar"),
	}

_ = badBlockHeader.Bytes() //应该恐慌
}
