
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


package committer

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("committer")

//--------！！！！重要！！-！重要！！-！重要！！----------
//这仅用于完成“骨架”的循环
//路径，以便我们能够推理和修改提交者组件
//更有效地使用代码。

//PeerledgerSupport抽象出ledger.peerledger接口的API
//需要执行LedgerCommitter
type PeerLedgerSupport interface {
	GetPvtDataAndBlockByNum(blockNum uint64, filter ledger.PvtNsCollFilter) (*ledger.BlockAndPvtData, error)

	GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error)

	CommitWithPvtData(blockAndPvtdata *ledger.BlockAndPvtData) error

	CommitPvtDataOfOldBlocks(blockPvtData []*ledger.BlockPvtData) ([]*ledger.PvtdataHashMismatch, error)

	GetBlockchainInfo() (*common.BlockchainInfo, error)

	GetBlockByNumber(blockNumber uint64) (*common.Block, error)

	GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error)

	GetMissingPvtDataTracker() (ledger.MissingPvtDataTracker, error)

	Close()
}

//LedgerCommitter是Committer接口的实现
//它保持对分类帐的引用，以提交块并检索
//链信息
type LedgerCommitter struct {
	PeerLedgerSupport
	eventer ConfigBlockEventer
}

//要定义操作的configBlockEventer回调函数原型类型
//到达新的配置更新块时
type ConfigBlockEventer func(block *common.Block) error

//newledgercommitter是一个工厂函数，用于创建committer的实例
//它通过验证传递传入块并将其提交到分类帐中。
func NewLedgerCommitter(ledger PeerLedgerSupport) *LedgerCommitter {
	return NewLedgerCommitterReactive(ledger, func(_ *common.Block) error { return nil })
}

//newledgercommitterreactive是一个工厂函数，用于创建提交者的实例
//与newledgercommitter的方法相同，同时还提供了一个指定回调的选项
//在新配置块到达并提交事件时调用
func NewLedgerCommitterReactive(ledger PeerLedgerSupport, eventer ConfigBlockEventer) *LedgerCommitter {
	return &LedgerCommitter{PeerLedgerSupport: ledger, eventer: eventer}
}

//预调试负责验证块并根据其更新
//内容
func (lc *LedgerCommitter) preCommit(block *common.Block) error {
//用新的配置块更新CSCC
	if utils.IsConfigBlock(block) {
		logger.Debug("Received configuration update, calling CSCC ConfigUpdate")
		if err := lc.eventer(block); err != nil {
			return errors.WithMessage(err, "could not update CSCC with new configuration update")
		}
	}
	return nil
}

//COMMITTHPVTData使用私有数据自动提交块
func (lc *LedgerCommitter) CommitWithPvtData(blockAndPvtData *ledger.BlockAndPvtData) error {
//做验证和之前需要做的任何事情
//提交新块
	if err := lc.preCommit(blockAndPvtData.Block); err != nil {
		return err
	}

//提交新块
	if err := lc.PeerLedgerSupport.CommitWithPvtData(blockAndPvtData); err != nil {
		return err
	}

	return nil
}

//getpvtdata和blockbynum检索给定序列号的私有数据和块
func (lc *LedgerCommitter) GetPvtDataAndBlockByNum(seqNum uint64) (*ledger.BlockAndPvtData, error) {
	return lc.PeerLedgerSupport.GetPvtDataAndBlockByNum(seqNum, nil)
}

//LedgerHeight返回最近提交的块序列号
func (lc *LedgerCommitter) LedgerHeight() (uint64, error) {
	var info *common.BlockchainInfo
	var err error
	if info, err = lc.GetBlockchainInfo(); err != nil {
		logger.Errorf("Cannot get blockchain info, %s", info)
		return uint64(0), err
	}

	return info.Height, nil
}

//用于检索切片中提供序列号的块的GetBlocks
func (lc *LedgerCommitter) GetBlocks(blockSeqs []uint64) []*common.Block {
	var blocks []*common.Block

	for _, seqNum := range blockSeqs {
		if blck, err := lc.GetBlockByNumber(seqNum); err != nil {
			logger.Errorf("Not able to acquire block num %d, from the ledger skipping...", seqNum)
			continue
		} else {
			logger.Debug("Appending next block with seqNum = ", seqNum, " to the resulting set")
			blocks = append(blocks, blck)
		}
	}

	return blocks
}
