
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


package endorsement

import (
	"github.com/hyperledger/fabric/core/handlers/endorsement/api"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
)

//国家定义与世界国家的互动
type State interface {
//getprivatedatamultiplekeys获取单个调用中多个私有数据项的值
	GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([][]byte, error)

//GetStateMultipleKeys获取单个调用中多个键的值
	GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error)

//getTransientByXid获取与给定txID关联的私有数据值
	GetTransientByTXID(txID string) ([]*rwset.TxPvtReadWriteSet, error)

//完成释放国家占用的资源
	Done()
}

//StateFetcher检索状态的实例
type StateFetcher interface {
	endorsement.Dependency

//FetchState获取状态
	FetchState() (State, error)
}
