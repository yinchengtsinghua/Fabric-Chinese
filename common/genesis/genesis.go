
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package genesis

import (
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

const (
	msgVersion = int32(1)

//对于Genesis区块，这些值是固定的。
	epoch = 0
)

//工厂有助于创建Genesis块。
type Factory interface {
//块返回给定通道ID的Genesis块。
	Block(channelID string) (*cb.Block, error)
}

type factory struct {
	channelGroup *cb.ConfigGroup
}

//NewFactoryImpl创建新工厂。
func NewFactoryImpl(channelGroup *cb.ConfigGroup) Factory {
	return &factory{channelGroup: channelGroup}
}

//块构造并返回给定通道ID的Genesis块。
func (f *factory) Block(channelID string) (*cb.Block, error) {
	payloadChannelHeader := utils.MakeChannelHeader(cb.HeaderType_CONFIG, msgVersion, channelID, epoch)
	payloadSignatureHeader := utils.MakeSignatureHeader(nil, utils.CreateNonceOrPanic())
	utils.SetTxID(payloadChannelHeader, payloadSignatureHeader)
	payloadHeader := utils.MakePayloadHeader(payloadChannelHeader, payloadSignatureHeader)
	payload := &cb.Payload{Header: payloadHeader, Data: utils.MarshalOrPanic(&cb.ConfigEnvelope{Config: &cb.Config{ChannelGroup: f.channelGroup}})}
	envelope := &cb.Envelope{Payload: utils.MarshalOrPanic(payload), Signature: nil}

	block := cb.NewBlock(0, nil)
	block.Data = &cb.BlockData{Data: [][]byte{utils.MarshalOrPanic(envelope)}}
	block.Header.DataHash = block.Data.Hash()
	block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = utils.MarshalOrPanic(&cb.Metadata{
		Value: utils.MarshalOrPanic(&cb.LastConfig{Index: 0}),
	})
	return block, nil
}
