
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


package fsblkstorage

import (
	"github.com/golang/protobuf/proto"
	ledgerutil "github.com/hyperledger/fabric/common/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

type serializedBlockInfo struct {
	blockHeader *common.BlockHeader
	txOffsets   []*txindexInfo
	metadata    *common.BlockMetadata
}

//必须为历史记录保留交易顺序
type txindexInfo struct {
	txID        string
	loc         *locPointer
	isDuplicate bool
}

func serializeBlock(block *common.Block) ([]byte, *serializedBlockInfo, error) {
	buf := proto.NewBuffer(nil)
	var err error
	info := &serializedBlockInfo{}
	info.blockHeader = block.Header
	info.metadata = block.Metadata
	if err = addHeaderBytes(block.Header, buf); err != nil {
		return nil, nil, err
	}
	if info.txOffsets, err = addDataBytes(block.Data, buf); err != nil {
		return nil, nil, err
	}
	if err = addMetadataBytes(block.Metadata, buf); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), info, nil
}

func deserializeBlock(serializedBlockBytes []byte) (*common.Block, error) {
	block := &common.Block{}
	var err error
	b := ledgerutil.NewBuffer(serializedBlockBytes)
	if block.Header, err = extractHeader(b); err != nil {
		return nil, err
	}
	if block.Data, _, err = extractData(b); err != nil {
		return nil, err
	}
	if block.Metadata, err = extractMetadata(b); err != nil {
		return nil, err
	}
	return block, nil
}

func extractSerializedBlockInfo(serializedBlockBytes []byte) (*serializedBlockInfo, error) {
	info := &serializedBlockInfo{}
	var err error
	b := ledgerutil.NewBuffer(serializedBlockBytes)
	info.blockHeader, err = extractHeader(b)
	if err != nil {
		return nil, err
	}
	_, info.txOffsets, err = extractData(b)
	if err != nil {
		return nil, err
	}

	info.metadata, err = extractMetadata(b)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func addHeaderBytes(blockHeader *common.BlockHeader, buf *proto.Buffer) error {
	if err := buf.EncodeVarint(blockHeader.Number); err != nil {
		return err
	}
	if err := buf.EncodeRawBytes(blockHeader.DataHash); err != nil {
		return err
	}
	if err := buf.EncodeRawBytes(blockHeader.PreviousHash); err != nil {
		return err
	}
	return nil
}

func addDataBytes(blockData *common.BlockData, buf *proto.Buffer) ([]*txindexInfo, error) {
	var txOffsets []*txindexInfo

	if err := buf.EncodeVarint(uint64(len(blockData.Data))); err != nil {
		return nil, err
	}
	for _, txEnvelopeBytes := range blockData.Data {
		offset := len(buf.Bytes())
		txid, err := extractTxID(txEnvelopeBytes)
		if err != nil {
			return nil, err
		}
		if err := buf.EncodeRawBytes(txEnvelopeBytes); err != nil {
			return nil, err
		}
		idxInfo := &txindexInfo{txID: txid, loc: &locPointer{offset, len(buf.Bytes()) - offset}}
		txOffsets = append(txOffsets, idxInfo)
	}
	return txOffsets, nil
}

func addMetadataBytes(blockMetadata *common.BlockMetadata, buf *proto.Buffer) error {
	numItems := uint64(0)
	if blockMetadata != nil {
		numItems = uint64(len(blockMetadata.Metadata))
	}
	if err := buf.EncodeVarint(numItems); err != nil {
		return err
	}
	for _, b := range blockMetadata.Metadata {
		if err := buf.EncodeRawBytes(b); err != nil {
			return err
		}
	}
	return nil
}

func extractHeader(buf *ledgerutil.Buffer) (*common.BlockHeader, error) {
	header := &common.BlockHeader{}
	var err error
	if header.Number, err = buf.DecodeVarint(); err != nil {
		return nil, err
	}
	if header.DataHash, err = buf.DecodeRawBytes(false); err != nil {
		return nil, err
	}
	if header.PreviousHash, err = buf.DecodeRawBytes(false); err != nil {
		return nil, err
	}
	if len(header.PreviousHash) == 0 {
		header.PreviousHash = nil
	}
	return header, nil
}

func extractData(buf *ledgerutil.Buffer) (*common.BlockData, []*txindexInfo, error) {
	data := &common.BlockData{}
	var txOffsets []*txindexInfo
	var numItems uint64
	var err error

	if numItems, err = buf.DecodeVarint(); err != nil {
		return nil, nil, err
	}
	for i := uint64(0); i < numItems; i++ {
		var txEnvBytes []byte
		var txid string
		txOffset := buf.GetBytesConsumed()
		if txEnvBytes, err = buf.DecodeRawBytes(false); err != nil {
			return nil, nil, err
		}
		if txid, err = extractTxID(txEnvBytes); err != nil {
			return nil, nil, err
		}
		data.Data = append(data.Data, txEnvBytes)
		idxInfo := &txindexInfo{txID: txid, loc: &locPointer{txOffset, buf.GetBytesConsumed() - txOffset}}
		txOffsets = append(txOffsets, idxInfo)
	}
	return data, txOffsets, nil
}

func extractMetadata(buf *ledgerutil.Buffer) (*common.BlockMetadata, error) {
	metadata := &common.BlockMetadata{}
	var numItems uint64
	var metadataEntry []byte
	var err error
	if numItems, err = buf.DecodeVarint(); err != nil {
		return nil, err
	}
	for i := uint64(0); i < numItems; i++ {
		if metadataEntry, err = buf.DecodeRawBytes(false); err != nil {
			return nil, err
		}
		metadata.Metadata = append(metadata.Metadata, metadataEntry)
	}
	return metadata, nil
}

func extractTxID(txEnvelopBytes []byte) (string, error) {
	txEnvelope, err := utils.GetEnvelopeFromBlock(txEnvelopBytes)
	if err != nil {
		return "", err
	}
	txPayload, err := utils.GetPayload(txEnvelope)
	if err != nil {
		return "", nil
	}
	chdr, err := utils.UnmarshalChannelHeader(txPayload.Header.ChannelHeader)
	if err != nil {
		return "", err
	}
	return chdr.TxId, nil
}
