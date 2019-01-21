
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


package blockledger

import (
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
)

var closedChan chan struct{}

func init() {
	closedChan = make(chan struct{})
	close(closedChan)
}

//NotFoundErrorIterator只返回cb.status的错误“未找到”，
//
type NotFoundErrorIterator struct{}

//
func (nfei *NotFoundErrorIterator) Next() (*cb.Block, cb.Status) {
	return nil, cb.Status_NOT_FOUND
}

//
func (nfei *NotFoundErrorIterator) ReadyChan() <-chan struct{} {
	return closedChan
}

//
func (nfei *NotFoundErrorIterator) Close() {}

//
//
//
//适应非确定性编组
func CreateNextBlock(rl Reader, messages []*cb.Envelope) *cb.Block {
	var nextBlockNumber uint64
	var previousBlockHash []byte

	if rl.Height() > 0 {
		it, _ := rl.Iterator(&ab.SeekPosition{
			Type: &ab.SeekPosition_Newest{
				Newest: &ab.SeekNewest{},
			},
		})
		block, status := it.Next()
		if status != cb.Status_SUCCESS {
			panic("Error seeking to newest block for chain with non-zero height")
		}
		nextBlockNumber = block.Header.Number + 1
		previousBlockHash = block.Header.Hash()
	}

	data := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	var err error
	for i, msg := range messages {
		data.Data[i], err = proto.Marshal(msg)
		if err != nil {
			panic(err)
		}
	}

	block := cb.NewBlock(nextBlockNumber, previousBlockHash)
	block.Header.DataHash = data.Hash()
	block.Data = data

	return block
}

//
func GetBlock(rl Reader, index uint64) *cb.Block {
	iterator, _ := rl.Iterator(&ab.SeekPosition{
		Type: &ab.SeekPosition_Specified{
			Specified: &ab.SeekSpecified{Number: index},
		},
	})
	if iterator == nil {
		return nil
	}
	defer iterator.Close()
	block, status := iterator.Next()
	if status != cb.Status_SUCCESS {
		return nil
	}
	return block
}
