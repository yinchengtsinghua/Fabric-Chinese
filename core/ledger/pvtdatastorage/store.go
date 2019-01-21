
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


package pvtdatastorage

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
)

//提供程序提供特定“存储”的句柄，而该存储反过来管理
//分类帐的专用写入集
type Provider interface {
	OpenStore(id string) (Store, error)
	Close()
}

//存储管理分类帐专用写入集的永久存储
//因为pvt数据应该与
//分类帐，两者都应该在逻辑上发生在原子操作中。整齐
//为了实现这一点，该存储的实现应该提供
//支持类似于两阶段的提交/回滚功能。
//预期用途是-首先将私有数据提供给
//此存储（通过“准备”功能），然后将块附加到块存储。
//最后，在此基于存储的存储上调用“commit”或“rollback”函数之一
//关于块是否成功写入。商店实施
//在调用“prepare”和“commit”/“rollback”之间的服务器崩溃后，应该可以存活
type Store interface {
//init初始化存储。此函数应在使用存储之前调用
	Init(btlPolicy pvtdatapolicy.BTLPolicy)
//initlastcommittedblockheight将最后一个提交的块高度设置到pvt数据存储中
//此功能用于对等端以区块链启动的特殊情况。
//当pvt数据特性（因此存储）不是
//可用。此函数只能在这种情况下调用，因此
//如果存储不为空，则应引发错误。从这里成功返回
//功能存储区的状态应与调用Prepare/Commit的状态相同。
//块“0”到“blocknum”的函数，没有pvt数据
	InitLastCommittedBlock(blockNum uint64) error
//getpvtdatabyblocknum仅返回与给定块号对应的pvt数据
//pvt数据由筛选器中提供的“ns/collections”列表筛选。
//nil筛选器不筛选任何结果
	GetPvtDataByBlockNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error)
//GetMissingPvtDataInfoFormsToRecentBlocks返回缺少的
//最新的'maxblock'块，它至少丢失符合条件的集合的私有数据。
	GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error)
//准备为提交pvt数据和存储合格和不合格做好准备。
//缺少私有数据--“合格”表示缺少的私有数据属于集合
//此对等方是其成员；“不合格”表示缺少的私有数据属于
//此对等方不是其成员的集合。
//此调用不会提交pvt数据并存储丢失的私有数据。随后，呼叫者
//应调用“commit”或“rollback”函数。从此处返回应确保
//已经做了足够的准备，以便随后调用的“commit”函数可以提交
//数据和存储能够在这个函数调用和下一个函数调用之间的崩溃中幸存
//调用“commit”
	Prepare(blockNum uint64, pvtData []*ledger.TxPvtData, missingPvtData ledger.TxMissingPvtDataMap) error
//commit将上一次调用中传递的pvt数据提交给“prepare”函数
	Commit() error
//回滚将上一次调用中传递的pvt数据回滚到“prepare”函数
	Rollback() error
//当对等方有资格接收
//现有集合。参数“committingblk”引用包含相应的
//集合升级事务和参数“nscollmap”包含对等方的集合
//现在有资格接收pvt数据
	ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error
//commitpvtdataofoldblocks提交旧块的pvtdata（即以前丢失的数据）。
//参数'block s pvtdata'引用pvtstore中缺少的旧块的pvtdata列表。
//此调用存储了一个名为“lastupdatedodldblockslist”的附加项，该项保留准确的列表
//更新的块。此列表将在恢复过程中使用。一旦statedb更新为
//必须删除这些pvtdata和“lastupdatedodldblockslist”。在对等机启动期间，
//如果存在“lastupdatedodldblockslist”，则需要使用适当的pvtdata更新statedb。
	CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error
//getlastupdatedoldblockspvtdata返回“lastupdatedoldblockslist”中列出的块的pvtdata
	GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error)
//resetlastupdatedodldblockslist从存储中删除“lastupdatedodldblockslist”条目
	ResetLastUpdatedOldBlocksList() error
//如果存储区尚未提交任何块，则IsEmpty返回true
	IsEmpty() (bool, error)
//last committed block height返回上一个提交块的高度
	LastCommittedBlockHeight() (uint64, error)
//如果存储有挂起的批，则返回hasPendingBatch
	HasPendingBatch() (bool, error)
//关机停止存储
	Shutdown()
}

//如果存储不希望调用Prepare/Commit/Rollback/InitLastCommittedBlock，则由存储impl引发errillegalCall。
type ErrIllegalCall struct {
	msg string
}

func (err *ErrIllegalCall) Error() string {
	return err.msg
}

//如果不允许传递的参数，则由store impl引发errilegalargs。
type ErrIllegalArgs struct {
	msg string
}

func (err *ErrIllegalArgs) Error() string {
	return err.msg
}

//对于尚未提交的数据请求，将引发erAutoFrange。
type ErrOutOfRange struct {
	msg string
}

func (err *ErrOutOfRange) Error() string {
	return err.msg
}
