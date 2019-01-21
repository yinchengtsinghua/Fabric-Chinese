
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


package multichannel

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/orderer/common/blockcutter"
	"github.com/hyperledger/fabric/orderer/common/msgprocessor"
	mockblockcutter "github.com/hyperledger/fabric/orderer/mocks/common/blockcutter"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

//ConsenterSupport用于模拟多通道。ConsenterSupport接口
//每次写入块时，它都会写入批处理通道以允许同步。
type ConsenterSupport struct {
//sharedconfigval是sharedconfig（）返回的值
	SharedConfigVal *mockconfig.Orderer

//BlockCutterVal是BlockCutter（）返回的值
	BlockCutterVal *mockblockcutter.Receiver

//blockbyindex将块编号映射到这些块的检索值
	BlockByIndex map[uint64]*cb.Block

//blocks是writeblock写入最近创建的块的通道，
	Blocks chan *cb.Block

//chainIdVal是chainID（）返回的值。
	ChainIDVal string

//heightval是height（）返回的值
	HeightVal uint64

//nextBlockVal存储由最新的createnextBlock（）调用创建的块
	NextBlockVal *cb.Block

//ClassifyMsgVal由ClassifyMsg返回
	ClassifyMsgVal msgprocessor.Classification

//configseqval作为process*msg的configseq返回
	ConfigSeqVal uint64

//processnormalmsgerr作为processnormalmsg的错误返回
	ProcessNormalMsgErr error

//processconfigupdatemsgval作为processconfigupdatemsg的错误返回
	ProcessConfigUpdateMsgVal *cb.Envelope

//processconfigupdatemsgerr作为processconfigupdatemsg的错误返回
	ProcessConfigUpdateMsgErr error

//processconfigmsgval作为processconfigmsg的错误返回
	ProcessConfigMsgVal *cb.Envelope

//processconfigmsgerr由processconfigmsg返回
	ProcessConfigMsgErr error

//SequenceVal按序列返回
	SequenceVal uint64

//通过VerifyBlockSignature返回BlockVerificationer
	BlockVerificationErr error
}

//块返回具有给定数字的块，如果找不到，则返回零。
func (mcs *ConsenterSupport) Block(number uint64) *cb.Block {
	return mcs.BlockByIndex[number]
}

//断块刀具返回断块刀具值
func (mcs *ConsenterSupport) BlockCutter() blockcutter.Receiver {
	return mcs.BlockCutterVal
}

//sharedconfig返回sharedconfigval
func (mcs *ConsenterSupport) SharedConfig() channelconfig.Orderer {
	return mcs.SharedConfigVal
}

//createnextblock使用给定的数据创建一个简单的块结构
func (mcs *ConsenterSupport) CreateNextBlock(data []*cb.Envelope) *cb.Block {
	block := cb.NewBlock(0, nil)
	mtxs := make([][]byte, len(data))
	for i := range data {
		mtxs[i] = utils.MarshalOrPanic(data[i])
	}
	block.Data = &cb.BlockData{Data: mtxs}
	mcs.NextBlockVal = block
	return block
}

//writeblock将数据写入blocks通道
func (mcs *ConsenterSupport) WriteBlock(block *cb.Block, encodedMetadataValue []byte) {
	if encodedMetadataValue != nil {
		block.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = utils.MarshalOrPanic(&cb.Metadata{Value: encodedMetadataValue})
	}
	mcs.HeightVal++
	mcs.Blocks <- block
}

//WriteConfigBlock调用WriteBlock
func (mcs *ConsenterSupport) WriteConfigBlock(block *cb.Block, encodedMetadataValue []byte) {
	mcs.WriteBlock(block, encodedMetadataValue)
}

//chainID返回与此特定同意者实例关联的链ID
func (mcs *ConsenterSupport) ChainID() string {
	return mcs.ChainIDVal
}

//height返回与此特定同意者实例关联的链的块数
func (mcs *ConsenterSupport) Height() uint64 {
	return mcs.HeightVal
}

//sign返回传入的字节
func (mcs *ConsenterSupport) Sign(message []byte) ([]byte, error) {
	return message, nil
}

//NewSignatureHeader返回空签名头
func (mcs *ConsenterSupport) NewSignatureHeader() (*cb.SignatureHeader, error) {
	return &cb.SignatureHeader{}, nil
}

//ClassifyMsg返回ClassifyMsgVal、ClassifyMsgErr
func (mcs *ConsenterSupport) ClassifyMsg(chdr *cb.ChannelHeader) msgprocessor.Classification {
	return mcs.ClassifyMsgVal
}

//processnormalmsg返回configseqval，processnormalmsgerr
func (mcs *ConsenterSupport) ProcessNormalMsg(env *cb.Envelope) (configSeq uint64, err error) {
	return mcs.ConfigSeqVal, mcs.ProcessNormalMsgErr
}

//processconfigupdatemsg返回processconfigupdatemsgval、configseqval、processconfigupdatemsgerr
func (mcs *ConsenterSupport) ProcessConfigUpdateMsg(env *cb.Envelope) (config *cb.Envelope, configSeq uint64, err error) {
	return mcs.ProcessConfigUpdateMsgVal, mcs.ConfigSeqVal, mcs.ProcessConfigUpdateMsgErr
}

//processconfigmsg返回processconfigmsgval、configseqval、processconfigmsgerr
func (mcs *ConsenterSupport) ProcessConfigMsg(env *cb.Envelope) (*cb.Envelope, uint64, error) {
	return mcs.ProcessConfigMsgVal, mcs.ConfigSeqVal, mcs.ProcessConfigMsgErr
}

//序列返回SequenceVal
func (mcs *ConsenterSupport) Sequence() uint64 {
	return mcs.SequenceVal
}

//verifyblocksignature验证块的签名
func (mcs *ConsenterSupport) VerifyBlockSignature(_ []*cb.SignedData, _ *cb.ConfigEnvelope) error {
	return mcs.BlockVerificationErr
}
