
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


package plain

import (
	"github.com/hyperledger/fabric/common/ledger"
)

//memory ledger是一个内存中的事务和未使用的输出分类账。
//此实现仅用于测试。
type MemoryLedger struct {
	entries map[string][]byte
}

//new memoryledger创建新的memoryledger
func NewMemoryLedger() *MemoryLedger {
	return &MemoryLedger{
		entries: make(map[string][]byte),
	}
}

//GetState获取给定命名空间和键的值。对于chaincode，命名空间对应于chaincodeid
func (p *MemoryLedger) GetState(namespace string, key string) ([]byte, error) {
	value := p.entries[key]

	return value, nil
}

//setState为给定的命名空间和键设置给定值。对于chaincode，命名空间对应于chaincodeid
func (p *MemoryLedger) SetState(namespace string, key string, value []byte) error {
	p.entries[key] = value

	return nil
}

//GetStateRangeScanIterator获取给定命名空间的值，该命名空间位于由StartKey和EndKey确定的间隔内。
//这是一个模拟函数。
func (p *MemoryLedger) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ledger.ResultsIterator, error) {
	return nil, nil
}

//完成释放由memoryleedger占用的资源
func (p *MemoryLedger) Done() {
//没有要为memoryleedger释放的资源
}
