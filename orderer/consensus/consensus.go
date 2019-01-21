
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package consensus

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/orderer/common/blockcutter"
	"github.com/hyperledger/fabric/orderer/common/msgprocessor"
	cb "github.com/hyperledger/fabric/protos/common"
)

//同意者定义支持顺序机制。
type Consenter interface {
//handlechain应该为给定的资源集创建并返回对链的引用。
//对于给定的链，每个进程只能调用一次。一般来说，错误将被处理
//无法恢复并导致系统关闭。详见链条说明
//handlechain的第二个参数是指向存储在的“order”槽中的元数据的指针。
//提交到此链的分类帐的最后一个块。对于新的链，此元数据将
//没有，因为这个区域不在Genesis块上
	HandleChain(support ConsenterSupport, metadata *cb.Metadata) (Chain, error)
}

//链定义了一种注入消息以便排序的方法。
//注意，为了实现灵活性，实现者的职责是
//要接收命令的信息，请通过BlockCutter发送。通过HandleChain提供的接收器用于切割块，
//并最终写下账本也通过手工提供。这种设计允许两个主要流量
//1。消息被排序成流，流被切割成块，块被提交（solo，kafka）
//2。消息被切割成块，对块进行排序，然后提交块（sbft）
type Chain interface {
//订单接受在给定configseq处处理的消息。
//如果configseq提前，则由同意人负责。
//重新验证并可能丢弃消息
//同意人可以返回一个错误，表明消息未被接受。
	Order(env *cb.Envelope, configSeq uint64) error

//configure接受一条消息，该消息将重新配置通道并将
//如果提交，则触发对configseq的更新。配置必须具有
//由configupdate消息触发。如果配置序列前进，
//同意人有责任重新计算生成的配置，
//如果重新配置不再有效，则丢弃消息。
//同意人可以返回一个错误，表明消息未被接受。
	Configure(config *cb.Envelope, configSeq uint64) error

//waitready阻止等待同意者准备接受新消息。
//当同意者需要临时阻止进入消息时，这很有用，因此
//飞行中的信息可以被消耗掉。如果同意者是
//处于错误状态。如果不需要这种阻塞行为，同意者可以
//只需返回零。
	WaitReady() error

//ERRORED返回一个在发生错误时将关闭的通道。
//这对于必须终止等待的交付客户特别有用
//当同意人不是最新的客户。
	Errored() <-chan struct{}

//Start应该分配保持链的最新所需的任何资源。
//通常，这涉及到创建一个线程，该线程从排序源读取并传递
//将消息发送给块切割器，并将生成的块写入分类帐。
	Start()

//HALT释放为此链分配的资源。
	Halt()
}

//go：生成伪造者-o mocks/mock同意者\u support.go。同意人支持

//同意者支持为同意者实施提供可用的资源。
type ConsenterSupport interface {
	crypto.LocalSigner
	msgprocessor.Processor

//verifyblocksignature使用给定的可选项验证块的签名
//配置（可以为零）。
	VerifyBlockSignature([]*cb.SignedData, *cb.ConfigEnvelope) error

//BlockCutter返回此通道的块切割助手。
	BlockCutter() blockcutter.Receiver

//shared config提供来自通道当前配置块的共享配置。
	SharedConfig() channelconfig.Orderer

//createnextblock获取消息列表，并根据提交给分类帐的块号最高的块创建下一个块。
//请注意，在再次调用此方法之前，必须调用WriteBlock或WriteConfigBlock。
	CreateNextBlock(messages []*cb.Envelope) *cb.Block

//block返回具有给定数字的块，
//如果不存在这样的块，则为零。
	Block(number uint64) *cb.Block

//WriteBlock将块提交到分类帐。
	WriteBlock(block *cb.Block, encodedMetadataValue []byte)

//WriteConfigBlock向分类帐提交一个块，并在内部应用配置更新。
	WriteConfigBlock(block *cb.Block, encodedMetadataValue []byte)

//Sequence返回当前的配置噪声。
	Sequence() uint64

//chainID返回与此支持相关联的通道ID。
	ChainID() string

//height返回与此通道关联的链中的块数。
	Height() uint64
}
