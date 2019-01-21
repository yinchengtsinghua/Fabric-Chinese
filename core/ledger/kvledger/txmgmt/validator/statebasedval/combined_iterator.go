
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


package statebasedval

import (
	"strings"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

//combineditor实现接口statedb.resultsiterator。
//Internally, it maintains two iterators
//—（1）dbitr—对数据库中存在的键进行迭代的迭代器。
//—（2）updatesitr—迭代更新批处理中存在的键的迭代器。
//（即通过前面的有效事务插入/更新/删除的键
//在块中，作为块提交操作的一部分提交到数据库）
//
//在调用者希望看到最终结果的地方可以使用
//如果修改了前面的有效事务，则对键范围进行迭代
//将应用于数据库
//
//This can be used to perfrom validation for phantom reads in a transactions rwset
type combinedIterator struct {
//输入
	ns            string
	db            statedb.VersionedDB
	updates       *statedb.UpdateBatch
	endKey        string
	includeEndKey bool

//内部状态
	dbItr        statedb.ResultsIterator
	updatesItr   statedb.ResultsIterator
	dbItem       statedb.QueryResult
	updatesItem  statedb.QueryResult
	endKeyServed bool
}

func newCombinedIterator(db statedb.VersionedDB, updates *statedb.UpdateBatch,
	ns string, startKey string, endKey string, includeEndKey bool) (*combinedIterator, error) {

	var dbItr statedb.ResultsIterator
	var updatesItr statedb.ResultsIterator
	var err error
	if dbItr, err = db.GetStateRangeScanIterator(ns, startKey, endKey); err != nil {
		return nil, err
	}
	updatesItr = updates.GetRangeScanIterator(ns, startKey, endKey)
	var dbItem, updatesItem statedb.QueryResult
	if dbItem, err = dbItr.Next(); err != nil {
		return nil, err
	}
	if updatesItem, err = updatesItr.Next(); err != nil {
		return nil, err
	}
	logger.Debugf("Combined iterator initialized. dbItem=%#v, updatesItem=%#v", dbItem, updatesItem)
	return &combinedIterator{ns, db, updates, endKey, includeEndKey,
		dbItr, updatesItr, dbItem, updatesItem, false}, nil
}

//next返回dbitr或updatesitr的kv，后者给出下一个较小的键
//如果两者都给出相同的键，那么它将从updatesir返回kv。
func (itr *combinedIterator) Next() (statedb.QueryResult, error) {
	if itr.dbItem == nil && itr.updatesItem == nil {
		logger.Debugf("dbItem and updatesItem both are nil.")
		return itr.serveEndKeyIfNeeded()
	}
	var moveDBItr bool
	var moveUpdatesItr bool
	var selectedItem statedb.QueryResult
	compResult := compareKeys(itr.dbItem, itr.updatesItem)
	logger.Debugf("compResult=%d", compResult)
	switch compResult {
	case -1:
//dbitem较小
		selectedItem = itr.dbItem
		moveDBItr = true
	case 0:
//两个项目相同，因此，请选择更新站点m（最新）
		selectedItem = itr.updatesItem
		moveUpdatesItr = true
		moveDBItr = true
	case 1:
//更新站点m较小
		selectedItem = itr.updatesItem
		moveUpdatesItr = true
	}
	var err error
	if moveDBItr {
		if itr.dbItem, err = itr.dbItr.Next(); err != nil {
			return nil, err
		}
	}

	if moveUpdatesItr {
		if itr.updatesItem, err = itr.updatesItr.Next(); err != nil {
			return nil, err
		}
	}
	if isDelete(selectedItem) {
		return itr.Next()
	}
	logger.Debugf("Returning item=%#v. Next dbItem=%#v, Next updatesItem=%#v", selectedItem, itr.dbItem, itr.updatesItem)
	return selectedItem, nil
}

func (itr *combinedIterator) Close() {
	itr.dbItr.Close()
}

func (itr *combinedIterator) GetBookmarkAndClose() string {
	itr.Close()
	return ""
}

//ServeEndKeyIfNeeded仅返回一次EndKey，并且仅当IncludeEndKey设置为true时返回。
//in the constructor of combinedIterator.
func (itr *combinedIterator) serveEndKeyIfNeeded() (statedb.QueryResult, error) {
	if !itr.includeEndKey || itr.endKeyServed {
		logger.Debugf("Endkey not to be served. Returning nil... [toInclude=%t, alreadyServed=%t]",
			itr.includeEndKey, itr.endKeyServed)
		return nil, nil
	}
	logger.Debug("Serving the endKey")
	var vv *statedb.VersionedValue
	var err error
	vv = itr.updates.Get(itr.ns, itr.endKey)
	logger.Debugf("endKey value from updates:%s", vv)
	if vv == nil {
		if vv, err = itr.db.GetState(itr.ns, itr.endKey); err != nil {
			return nil, err
		}
		logger.Debugf("endKey value from stateDB:%s", vv)
	}
	itr.endKeyServed = true
	if vv == nil {
		return nil, nil
	}
	vkv := &statedb.VersionedKV{
		CompositeKey:   statedb.CompositeKey{Namespace: itr.ns, Key: itr.endKey},
		VersionedValue: statedb.VersionedValue{Value: vv.Value, Version: vv.Version}}

	if isDelete(vkv) {
		return nil, nil
	}
	return vkv, nil
}

func compareKeys(item1 statedb.QueryResult, item2 statedb.QueryResult) int {
	if item1 == nil {
		if item2 == nil {
			return 0
		}
		return 1
	}
	if item2 == nil {
		return -1
	}
//在这个阶段，这两项都不是零
	return strings.Compare(item1.(*statedb.VersionedKV).Key, item2.(*statedb.VersionedKV).Key)
}

func isDelete(item statedb.QueryResult) bool {
	return item.(*statedb.VersionedKV).Value == nil
}
