
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


package customtx

import (
	"sync"

	"github.com/hyperledger/fabric/protos/common"
)

var processors Processors
var once sync.Once

//处理器维护自定义事务类型与其对应的Tx处理器之间的关联
type Processors map[common.HeaderType]Processor

//初始化设置自定义处理器。此函数只能在ledgermgmt.initialize（）函数期间调用。
func Initialize(customTxProcessors Processors) {
	once.Do(func() {
		initialize(customTxProcessors)
	})
}

func initialize(customTxProcessors Processors) {
	processors = customTxProcessors
}

//GetProcessor返回与txtype关联的处理器
func GetProcessor(txType common.HeaderType) Processor {
	return processors[txType]
}
