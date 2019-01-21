
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


package ledger

import "github.com/hyperledger/fabric/common/ledger"

//go：生成伪造者-o mock/ledger-reader.go-fake-name-ledger reader。书页阅读器
//go：生成伪造者-o mock/ledger\manager.go-伪造名称ledger manager。分类管理器

//LedgerManager提供对分类帐基础结构的访问
type LedgerManager interface {
//返回已传递通道的LedgerReader，否则返回错误
	GetLedgerReader(channel string) (LedgerReader, error)
}

//LedgerReader接口，用于从分类帐中读取。
type LedgerReader interface {
//GetState获取给定命名空间和键的值。对于chaincode，命名空间对应于chaincodeid
	GetState(namespace string, key string) ([]byte, error)

//GetStateRangeScanIterator返回一个迭代器，该迭代器包含给定键范围之间的所有键值。
//结果中包含startkey，但不包括endkey。空的startkey引用第一个可用的key
//空的endkey指的是最后一个可用的key。用于扫描所有键，包括startkey和endkey
//可以作为空字符串提供。但是，出于性能原因，应该明智地使用完整扫描。
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ledger.ResultsIterator, error)

//完成释放LedgerReader占用的资源
	Done()
}

//go：生成伪造者-o mock/ledger-writer.go-伪造名称ledger writer。莱德格林特

//LedgerWriter接口，用于读取和写入分类帐。
type LedgerWriter interface {
	LedgerReader
//setState为给定的命名空间和键设置给定值。对于chaincode，命名空间对应于chaincodeid
	SetState(namespace string, key string, value []byte) error
}

//go:生成伪造者-o mock/results_iterator.go-forke name results iterator。制作人

//resultsiterator-查询结果集的迭代器
type ResultsIterator interface {
//Next返回结果集中的下一项。当
//迭代器耗尽
	Next() (ledger.QueryResult, error)
//close释放迭代器占用的资源
	Close()
}
