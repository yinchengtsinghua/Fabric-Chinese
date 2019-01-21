
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
	"fmt"
	"math"

	"github.com/hyperledger/fabric/common/util"
)

//newblock构造一个没有数据和元数据的块。
func NewBlock(seqNum uint64, previousHash []byte) *Block {
	block := &Block{}
	block.Header = &BlockHeader{}
	block.Header.Number = seqNum
	block.Header.PreviousHash = previousHash
	block.Data = &BlockData{}

	var metadataContents [][]byte
	for i := 0; i < len(BlockMetadataIndex_name); i++ {
		metadataContents = append(metadataContents, []byte{})
	}
	block.Metadata = &BlockMetadata{Metadata: metadataContents}

	return block
}

type asn1Header struct {
	Number       int64
	PreviousHash []byte
	DataHash     []byte
}

//字节返回块头的ASN.1封送表示。
func (b *BlockHeader) Bytes() []byte {
	asn1Header := asn1Header{
		PreviousHash: b.PreviousHash,
		DataHash:     b.DataHash,
	}
	if b.Number > uint64(math.MaxInt64) {
		panic(fmt.Errorf("Golang does not currently support encoding uint64 to asn1"))
	} else {
		asn1Header.Number = int64(b.Number)
	}
	result, err := asn1.Marshal(asn1Header)
	if err != nil {
//只有无法编码的类型才会出现错误，因为
//blockheader类型已知为a-priori，只包含可编码的类型，
//这里的错误是致命的，不应该被宣扬。
		panic(err)
	}
	return result
}

//哈希返回块头的哈希。
//此方法将很快删除，以允许可配置的哈希算法
func (b *BlockHeader) Hash() []byte {
	return util.ComputeSHA256(b.Bytes())
}

//字节返回BlockData的确定序列化版本
//最终，这应该被真正的梅克尔树结构所取代，
//但目前，我们假设一个无限宽的梅克尔树（uint32_max）
//它会退化成扁平的散列
func (b *BlockData) Bytes() []byte {
	return util.ConcatenateBytes(b.Data...)
}

//哈希返回块数据的封送表示形式的哈希。
func (b *BlockData) Hash() []byte {
	return util.ComputeSHA256(b.Bytes())
}
