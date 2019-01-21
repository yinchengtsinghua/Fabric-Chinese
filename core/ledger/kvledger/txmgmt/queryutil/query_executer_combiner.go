
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


package queryutil

import (
	"github.com/hyperledger/fabric/common/flogging"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

var logger = flogging.MustGetLogger("util")

//go：生成仿冒者-o mock/query_executer.go-forke name query executer。查询执行程序

//queryexecuter封装了查询函数
type QueryExecuter interface {
	GetState(namespace, key string) (*statedb.VersionedValue, error)
	GetStateRangeScanIterator(namespace, startKey, endKey string) (statedb.ResultsIterator, error)
}

//QeCombiner组合来自一个或多个基础“QueryExecutors”的查询结果
//在这种情况下，同一个键由多个“queryexecuters”返回，第一个“queryexecuter”
//在输入中被认为具有键的最新状态
type QECombiner struct {
QueryExecuters []QueryExecuter //按优先顺序递减的实际执行者
}

//GetState在接口Ledger.SimpleQueryExecutor中实现函数
func (c *QECombiner) GetState(namespace string, key string) ([]byte, error) {
	var vv *statedb.VersionedValue
	var val []byte
	var err error
	for _, qe := range c.QueryExecuters {
		if vv, err = qe.GetState(namespace, key); err != nil {
			return nil, err
		}
		if vv != nil {
			if !vv.IsDelete() {
				val = vv.Value
			}
			break
		}
	}
	return val, nil
}

//GetStateRangeScanIterator在接口Ledger.SimpleQueryExecutor中实现函数
func (c *QECombiner) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (commonledger.ResultsIterator, error) {
	var itrs []statedb.ResultsIterator
	for _, qe := range c.QueryExecuters {
		itr, err := qe.GetStateRangeScanIterator(namespace, startKey, endKey)
		if err != nil {
			for _, itr := range itrs {
				itr.Close()
			}
			return nil, err
		}
		itrs = append(itrs, itr)
	}
	itrCombiner, err := newItrCombiner(namespace, itrs)
	if err != nil {
		return nil, err
	}
	return itrCombiner, nil
}

//updateBatchBackedQueryExecutor包装更新批，以便在接口“queryExecutor”中提供函数
type UpdateBatchBackedQueryExecuter struct {
	UpdateBatch *statedb.UpdateBatch
}

//GetState implements function in interface 'queryExecuter'
func (qe *UpdateBatchBackedQueryExecuter) GetState(ns, key string) (*statedb.VersionedValue, error) {
	return qe.UpdateBatch.Get(ns, key), nil
}

//GetStateRangeScanIterator在接口“QueryExecuter”中实现函数
func (qe *UpdateBatchBackedQueryExecuter) GetStateRangeScanIterator(namespace, startKey, endKey string) (statedb.ResultsIterator, error) {
	return qe.UpdateBatch.GetRangeScanIterator(namespace, startKey, endKey), nil
}
