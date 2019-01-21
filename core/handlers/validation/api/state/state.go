
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


package validation

import (
	"github.com/hyperledger/fabric/core/handlers/validation/api"
)

//国家定义与世界国家的互动
type State interface {
//GetStateMultipleKeys获取单个调用中多个键的值
	GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error)

//GetStateRangeScanIterator返回一个迭代器，该迭代器包含给定键范围之间的所有键值。
//结果中包含startkey，但不包括endkey。空的startkey引用第一个可用的key
//空的endkey指的是最后一个可用的key。用于扫描所有键，包括startkey和endkey
//可以作为空字符串提供。但是，出于性能原因，应该明智地使用完整扫描。
//返回的resultsiterator包含在protos/ledger/queryresult中定义的*kv类型的结果。
	GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ResultsIterator, error)

//GetStateMetadata返回给定命名空间和键的元数据
	GetStateMetadata(namespace, key string) (map[string][]byte, error)

//getprivatedatametadatabyhash获取由元组标识的私有数据项的元数据<namespace，collection，keyhash>
	GetPrivateDataMetadataByHash(namespace, collection string, keyhash []byte) (map[string][]byte, error)

//完成释放国家占用的资源
	Done()
}

//StateFetcher检索状态的实例
type StateFetcher interface {
	validation.Dependency

//FetchState获取状态
	FetchState() (State, error)
}

//resultsiterator-查询结果集的迭代器
type ResultsIterator interface {
//Next返回结果集中的下一项。当
//迭代器耗尽
	Next() (QueryResult, error)
//close释放迭代器占用的资源
	Close()
}

//queryresult-支持不同类型查询结果的通用接口。不同查询的实际类型不同
type QueryResult interface{}
