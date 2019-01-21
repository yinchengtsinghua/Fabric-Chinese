
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


package sysccprovider

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/ledger"
)

//StaskChanoCeDev提供提供了一个抽象层
//用于不同的包与中的代码交互
//system chaincode package without importing it; more methods
//如有必要，应在下面添加
type SystemChaincodeProvider interface {
//如果提供的链码是系统链码，ISSysCC将返回true。
	IsSysCC(name string) bool

//如果提供的链码为
//是系统链代码，不能通过cc2cc调用调用
	IsSysCCAndNotInvokableCC2CC(name string) bool

//如果提供的链代码为
//是一个系统链码，并不是通过提案不可接受的。
	IsSysCCAndNotInvokableExternal(name string) bool

//GetQueryExecutorForLedger返回
//所提供频道的分类帐。
//这对于需要不受约束的系统链代码很有用
//访问分类帐
	GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error)

//GETAPPLICATIOFIGG返回通道的CONTXXAPPLATION.SARDCONFIGG
//以及应用程序配置是否存在
	GetApplicationConfig(cid string) (channelconfig.Application, bool)

//返回与传递的通道关联的策略管理器
//以及策略管理器是否存在
	PolicyManager(channelID string) (policies.Manager, bool)
}

//chaincode instance是chaincode实例的唯一标识符
type ChaincodeInstance struct {
	ChainID          string
	ChaincodeName    string
	ChaincodeVersion string
}

func (ci *ChaincodeInstance) String() string {
	return ci.ChainID + "." + ci.ChaincodeName + "#" + ci.ChaincodeVersion
}
