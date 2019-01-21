
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
	"bytes"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
)

type rangeQueryValidator interface {
	init(rqInfo *kvrwset.RangeQueryInfo, itr statedb.ResultsIterator) error
	validate() (bool, error)
}

type rangeQueryResultsValidator struct {
	rqInfo *kvrwset.RangeQueryInfo
	itr    statedb.ResultsIterator
}

func (v *rangeQueryResultsValidator) init(rqInfo *kvrwset.RangeQueryInfo, itr statedb.ResultsIterator) error {
	v.rqInfo = rqInfo
	v.itr = itr
	return nil
}

//验证迭代范围查询信息中的结果（在模拟过程中捕获）
//和迭代器（提交状态的最新视图，即db+更新）。最初结果不匹配
//从两个源（范围查询信息和迭代器）中，validate返回false。
func (v *rangeQueryResultsValidator) validate() (bool, error) {
	rqResults := v.rqInfo.GetRawReads().GetKvReads()
	itr := v.itr
	var result statedb.QueryResult
	var err error
	if result, err = itr.Next(); err != nil {
		return false, err
	}
	if len(rqResults) == 0 {
		return result == nil, nil
	}
	for i := 0; i < len(rqResults); i++ {
		kvRead := rqResults[i]
		logger.Debugf("comparing kvRead=[%#v] to queryResponse=[%#v]", kvRead, result)
		if result == nil {
			logger.Debugf("Query response nil. Key [%s] got deleted", kvRead.Key)
			return false, nil
		}
		versionedKV := result.(*statedb.VersionedKV)
		if versionedKV.Key != kvRead.Key {
			logger.Debugf("key name mismatch: Key in rwset = [%s], key in query results = [%s]", kvRead.Key, versionedKV.Key)
			return false, nil
		}
		if !version.AreSame(versionedKV.Version, convertToVersionHeight(kvRead.Version)) {
			logger.Debugf(`Version mismatch for key [%s]: Version in rwset = [%#v], latest version = [%#v]`,
				versionedKV.Key, versionedKV.Version, kvRead.Version)
			return false, nil
		}
		if result, err = itr.Next(); err != nil {
			return false, err
		}
	}
	if result != nil {
//迭代器没有用完-这意味着在给定的范围内有额外的结果
		logger.Debugf("Extra result = [%#v]", result)
		return false, nil
	}
	return true, nil
}

type rangeQueryHashValidator struct {
	rqInfo        *kvrwset.RangeQueryInfo
	itr           statedb.ResultsIterator
	resultsHelper *rwsetutil.RangeQueryResultsHelper
}

func (v *rangeQueryHashValidator) init(rqInfo *kvrwset.RangeQueryInfo, itr statedb.ResultsIterator) error {
	v.rqInfo = rqInfo
	v.itr = itr
	var err error
	v.resultsHelper, err = rwsetutil.NewRangeQueryResultsHelper(true, rqInfo.GetReadsMerkleHashes().MaxDegree)
	return err
}

//通过迭代器验证迭代（“提交状态”的最新视图，即Db+更新）
//并开始逐步构建Merkle树（在模拟和验证期间，
//rwset.rangequeryresultshelper用于增量构建merkle树）。
//
//此函数还将正在构建的Merkle树与Merkle树进行比较
//范围查询信息中存在的摘要（在模拟过程中生成）。
//此函数在两个merkle树的节点之间第一次不匹配时返回false
//在所需级别（范围查询信息中Merkle树的最大级别）。
func (v *rangeQueryHashValidator) validate() (bool, error) {
	itr := v.itr
	lastMatchedIndex := -1
	inMerkle := v.rqInfo.GetReadsMerkleHashes()
	var merkle *kvrwset.QueryReadsMerkleSummary
	logger.Debugf("inMerkle: %#v", inMerkle)
	for {
		var result statedb.QueryResult
		var err error
		if result, err = itr.Next(); err != nil {
			return false, err
		}
		logger.Debugf("Processing result = %#v", result)
		if result == nil {
			if _, merkle, err = v.resultsHelper.Done(); err != nil {
				return false, err
			}
			equals := inMerkle.Equal(merkle)
			logger.Debugf("Combined iterator exhausted. merkle=%#v, equals=%t", merkle, equals)
			return equals, nil
		}
		versionedKV := result.(*statedb.VersionedKV)
		v.resultsHelper.AddResult(rwsetutil.NewKVRead(versionedKV.Key, versionedKV.Version))
		merkle := v.resultsHelper.GetMerkleSummary()

		if merkle.MaxLevel < inMerkle.MaxLevel {
			logger.Debugf("Hashes still under construction. Noting to compare yet. Need more results. Continuing...")
			continue
		}
		if lastMatchedIndex == len(merkle.MaxLevelHashes)-1 {
			logger.Debugf("Need more results to build next entry [index=%d] at level [%d]. Continuing...",
				lastMatchedIndex+1, merkle.MaxLevel)
			continue
		}
		if len(merkle.MaxLevelHashes) > len(inMerkle.MaxLevelHashes) {
			logger.Debugf("Entries exceeded from what are present in the incoming merkleSummary. Validation failed")
			return false, nil
		}
		lastMatchedIndex++
		if !bytes.Equal(merkle.MaxLevelHashes[lastMatchedIndex], inMerkle.MaxLevelHashes[lastMatchedIndex]) {
			logger.Debugf("Hashes does not match at index [%d]. Validation failed", lastMatchedIndex)
			return false, nil
		}
	}
}

func convertToVersionHeight(v *kvrwset.Version) *version.Height {
	return version.NewHeight(v.BlockNum, v.TxNum)
}
