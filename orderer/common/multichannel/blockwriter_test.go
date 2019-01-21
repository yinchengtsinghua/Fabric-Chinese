
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


package multichannel

import (
	"testing"

	newchannelconfig "github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	mockconfigtx "github.com/hyperledger/fabric/common/mocks/configtx"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

type mockBlockWriterSupport struct {
	*mockconfigtx.Validator
	crypto.LocalSigner
	blockledger.ReadWriter
}

func (mbws mockBlockWriterSupport) Update(bundle *newchannelconfig.Bundle) {}

func (mbws mockBlockWriterSupport) CreateBundle(channelID string, config *cb.Config) (*newchannelconfig.Bundle, error) {
	return nil, nil
}

func TestCreateBlock(t *testing.T) {
	seedBlock := cb.NewBlock(7, []byte("lasthash"))
	seedBlock.Data.Data = [][]byte{[]byte("somebytes")}

	bw := &BlockWriter{lastBlock: seedBlock}
	block := bw.CreateNextBlock([]*cb.Envelope{
		{Payload: []byte("some other bytes")},
	})

	assert.Equal(t, seedBlock.Header.Number+1, block.Header.Number)
	assert.Equal(t, block.Data.Hash(), block.Header.DataHash)
	assert.Equal(t, seedBlock.Header.Hash(), block.Header.PreviousHash)
}

func TestBlockSignature(t *testing.T) {
	bw := &BlockWriter{
		support: &mockBlockWriterSupport{
			LocalSigner: mockCrypto(),
		},
	}

	block := cb.NewBlock(7, []byte("foo"))
	bw.addBlockSignature(block)

	md := utils.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_SIGNATURES)
	assert.Nil(t, md.Value, "Value is empty in this case")
	assert.NotNil(t, md.Signatures, "Should have signature")
}

func TestBlockLastConfig(t *testing.T) {
	lastConfigSeq := uint64(6)
	newConfigSeq := lastConfigSeq + 1
	newBlockNum := uint64(9)

	bw := &BlockWriter{
		support: &mockBlockWriterSupport{
			LocalSigner: mockCrypto(),
			Validator: &mockconfigtx.Validator{
				SequenceVal: newConfigSeq,
			},
		},
		lastConfigSeq: lastConfigSeq,
	}

	block := cb.NewBlock(newBlockNum, []byte("foo"))
	bw.addLastConfigSignature(block)

	assert.Equal(t, newBlockNum, bw.lastConfigBlockNum)
	assert.Equal(t, newConfigSeq, bw.lastConfigSeq)

	md := utils.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_LAST_CONFIG)
	assert.NotNil(t, md.Value, "Value not be empty in this case")
	assert.NotNil(t, md.Signatures, "Should have signature")

	lc := utils.GetLastConfigIndexFromBlockOrPanic(block)
	assert.Equal(t, newBlockNum, lc)
}

func TestWriteConfigBlock(t *testing.T) {
//TODO，使用assert.panicsWithValue一次可用
	t.Run("EmptyBlock", func(t *testing.T) {
		assert.Panics(t, func() { (&BlockWriter{}).WriteConfigBlock(&cb.Block{}, nil) })
	})
	t.Run("BadPayload", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{Payload: []byte("bad")}),
					},
				},
			}, nil)
		})
	})
	t.Run("MissingHeader", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{
							Payload: utils.MarshalOrPanic(&cb.Payload{}),
						}),
					},
				},
			}, nil)
		})
	})
	t.Run("BadChannelHeader", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{
							Payload: utils.MarshalOrPanic(&cb.Payload{
								Header: &cb.Header{
									ChannelHeader: []byte("bad"),
								},
							}),
						}),
					},
				},
			}, nil)
		})
	})
	t.Run("BadChannelHeaderType", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{
							Payload: utils.MarshalOrPanic(&cb.Payload{
								Header: &cb.Header{
									ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{}),
								},
							}),
						}),
					},
				},
			}, nil)
		})
	})
}

func TestGoodWriteConfig(t *testing.T) {
	l := NewRAMLedger(10)

	bw := &BlockWriter{
		support: &mockBlockWriterSupport{
			LocalSigner: mockCrypto(),
			ReadWriter:  l,
			Validator:   &mockconfigtx.Validator{},
		},
	}

	ctx := makeConfigTx(genesisconfig.TestChainID, 1)
	block := cb.NewBlock(1, genesisBlock.Header.Hash())
	block.Data.Data = [][]byte{utils.MarshalOrPanic(ctx)}
	consenterMetadata := []byte("foo")
	bw.WriteConfigBlock(block, consenterMetadata)

//等待提交完成
	bw.committingBlock.Lock()
	bw.committingBlock.Unlock()

	cBlock := blockledger.GetBlock(l, block.Header.Number)
	assert.Equal(t, block.Header, cBlock.Header)
	assert.Equal(t, block.Data, cBlock.Data)

	omd := utils.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_ORDERER)
	assert.Equal(t, consenterMetadata, omd.Value)
}
