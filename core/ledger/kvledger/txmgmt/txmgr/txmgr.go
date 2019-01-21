
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


package txmgr

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

//事务管理器应该实现的接口
type TxMgr interface {
	NewQueryExecutor(txid string) (ledger.QueryExecutor, error)
	NewTxSimulator(txid string) (ledger.TxSimulator, error)
	ValidateAndPrepare(blockAndPvtdata *ledger.BlockAndPvtData, doMVCCValidation bool) ([]*TxStatInfo, error)
	RemoveStaleAndCommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error
	GetLastSavepoint() (*version.Height, error)
	ShouldRecover(lastAvailableBlock uint64) (bool, uint64, error)
	CommitLostBlock(blockAndPvtdata *ledger.BlockAndPvtData) error
	Commit() error
	Rollback()
	Shutdown()
}

//txstatinfo封装有关事务的信息
type TxStatInfo struct {
	ValidationCode peer.TxValidationCode
	TxType         common.HeaderType
	ChaincodeID    *peer.ChaincodeID
	NumCollections int
}

//如果在更新事务中执行了不受支持的查询，则应引发errUnsupportedTransaction。
type ErrUnsupportedTransaction struct {
	Msg string
}

func (e *ErrUnsupportedTransaction) Error() string {
	return e.Msg
}

//当应用程序查找私有数据项时，将引发errpvdtatanovailable
//在模拟过程中，模拟器无法返回
//与暴露在模拟中的Snapshopt一致的私有数据项
type ErrPvtdataNotAvailable struct {
	Msg string
}

func (e *ErrPvtdataNotAvailable) Error() string {
	return e.Msg
}
