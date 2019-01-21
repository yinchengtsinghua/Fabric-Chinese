
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


package committer

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
)

//committer是由committer支持的接口
//唯一的提交者是noopsSingleChain提交者。
//界面有意与鞋底稀疏
//目标是“暂时把一切交给委员”。
//当我们巩固引导过程和我们添加
//更多支持（如八卦）此接口将
//改变
type Committer interface {

//将私有数据块和私有数据提交到分类帐中
	CommitWithPvtData(blockAndPvtData *ledger.BlockAndPvtData) error

//GetPvtDataAndBlockByNum使用给定的私有数据检索块
//序列号
	GetPvtDataAndBlockByNum(seqNum uint64) (*ledger.BlockAndPvtData, error)

//getpvdtatabynum从分类帐返回一部分私有数据
//对于给定的块并基于指示
//要检索的私有数据的集合和命名空间
	GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error)

//获取最近的块序列号
	LedgerHeight() (uint64, error)

//获取切片中提供序列号的块
	GetBlocks(blockSeqs []uint64) []*common.Block

//getconfigHistoryRetriever返回configHistoryRetriever
	GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error)

//commitpvtdataofoldblocks提交与已提交的块对应的私有数据
//如果此函数中提供的某些私有数据的哈希不匹配
//块中存在相应的哈希，不匹配的私有数据不是
//提交并返回不匹配信息
	CommitPvtDataOfOldBlocks(blockPvtData []*ledger.BlockPvtData) ([]*ledger.PvtdataHashMismatch, error)

//GetMissingPvtDataTracker返回MissingPvtDataTracker
	GetMissingPvtDataTracker() (ledger.MissingPvtDataTracker, error)

//关闭提交服务
	Close()
}
