
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
*/


package fsblkstorage

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func TestBlockSerialization(t *testing.T) {
	block := testutil.ConstructTestBlock(t, 1, 10, 100)
	bb, _, err := serializeBlock(block)
	assert.NoError(t, err)
	deserializedBlock, err := deserializeBlock(bb)
	assert.NoError(t, err)
	assert.Equal(t, block, deserializedBlock)
}

func TestExtractTxid(t *testing.T) {
	txEnv, txid, _ := testutil.ConstructTransaction(t, testutil.ConstructRandomBytes(t, 50), "", false)
	txEnvBytes, _ := putils.GetBytesEnvelope(txEnv)
	extractedTxid, err := extractTxID(txEnvBytes)
	assert.NoError(t, err)
	assert.Equal(t, txid, extractedTxid)
}

func TestSerializedBlockInfo(t *testing.T) {
	block := testutil.ConstructTestBlock(t, 1, 10, 100)
	bb, info, err := serializeBlock(block)
	assert.NoError(t, err)
	infoFromBB, err := extractSerializedBlockInfo(bb)
	assert.NoError(t, err)
	assert.Equal(t, info, infoFromBB)
	assert.Equal(t, len(block.Data.Data), len(info.txOffsets))
	for txIndex, txEnvBytes := range block.Data.Data {
		txid, err := extractTxID(txEnvBytes)
		assert.NoError(t, err)

		indexInfo := info.txOffsets[txIndex]
		indexTxID := indexInfo.txID
		indexOffset := indexInfo.loc

		assert.Equal(t, indexTxID, txid)
		b := bb[indexOffset.offset:]
		length, num := proto.DecodeVarint(b)
		txEnvBytesFromBB := b[num : num+int(length)]
		assert.Equal(t, txEnvBytes, txEnvBytesFromBB)
	}
}
