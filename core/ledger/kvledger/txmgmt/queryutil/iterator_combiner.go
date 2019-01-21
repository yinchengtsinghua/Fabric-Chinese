
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
	"fmt"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
)

type itrCombiner struct {
	namespace string
	holders   []*itrHolder
}

func newItrCombiner(namespace string, baseIterators []statedb.ResultsIterator) (*itrCombiner, error) {
	var holders []*itrHolder
	for _, itr := range baseIterators {
		res, err := itr.Next()
		if err != nil {
			for _, holder := range holders {
				holder.itr.Close()
			}
			return nil, err
		}
		if res != nil {
			holders = append(holders, &itrHolder{itr, res.(*statedb.VersionedKV)})
		}
	}
	return &itrCombiner{namespace, holders}, nil
}

//Next从基础迭代器返回下一个符合条件的项。
//此函数计算底层迭代器，并选择
//给出词典最小的键。然后，它保存该值，并推进所选迭代器。
//如果所选的迭代器中没有元素，则该迭代器将关闭，并从迭代器列表中删除。
func (combiner *itrCombiner) Next() (commonledger.QueryResult, error) {
	logger.Debugf("Iterators position at beginning: %s", combiner.holders)
	if len(combiner.holders) == 0 {
		return nil, nil
	}
	smallestHolderIndex := 0
	for i := 1; i < len(combiner.holders); i++ {
		smallestKey, holderKey := combiner.keyAt(smallestHolderIndex), combiner.keyAt(i)
		switch {
case holderKey == smallestKey: //我们在低阶迭代器（键的过时值）中找到了相同的键；
//我们已经有了这个密钥的最新值（在smallestholder中）。忽略此值并移动迭代器
//到下一个项（到更大的键），以便在下一轮键选择中，不再考虑此键
			removed, err := combiner.moveItrAndRemoveIfExhausted(i)
			if err != nil {
				return nil, err
			}
if removed { //如果使用当前迭代器并因此删除，则减小索引
//因为剩余迭代器的索引减量为1
				i--
			}
		case holderKey < smallestKey:
			smallestHolderIndex = i
		default:
//正在评估的当前密钥大于smallestkey-不执行任何操作
		}
	}
	kv := combiner.kvAt(smallestHolderIndex)
	combiner.moveItrAndRemoveIfExhausted(smallestHolderIndex)
	if kv.IsDelete() {
		return combiner.Next()
	}
	logger.Debugf("Key [%s] selected from iterator at index [%d]", kv.Key, smallestHolderIndex)
	logger.Debugf("Iterators position at end: %s", combiner.holders)
	return &queryresult.KV{Namespace: combiner.namespace, Key: kv.Key, Value: kv.Value}, nil
}

//moveitrandemoveifexhasted将索引i处的迭代器移动到下一项。如果迭代器耗尽
//然后迭代器从底层切片中移除
func (combiner *itrCombiner) moveItrAndRemoveIfExhausted(i int) (removed bool, err error) {
	holder := combiner.holders[i]
	exhausted, err := holder.moveToNext()
	if err != nil {
		return false, err
	}
	if exhausted {
		combiner.holders[i].itr.Close()
		combiner.holders = append(combiner.holders[:i], combiner.holders[i+1:]...)

	}
	return exhausted, nil
}

//kv at返回索引i处迭代器的可用kv
func (combiner *itrCombiner) kvAt(i int) *statedb.VersionedKV {
	return combiner.holders[i].kv
}

//key at返回索引i处迭代器的可用键
func (combiner *itrCombiner) keyAt(i int) string {
	return combiner.kvAt(i).Key
}

//关闭关闭所有底层迭代器
func (combiner *itrCombiner) Close() {
	for _, holder := range combiner.holders {
		holder.itr.Close()
	}
}

//itrholder包含一个迭代器，并将迭代器中的下一个项保持在缓冲区中
type itrHolder struct {
	itr statedb.ResultsIterator
	kv  *statedb.VersionedKV
}

//moveToNext获取要保留在缓冲区中的下一个项，如果迭代器已用完，则返回true。
func (holder *itrHolder) moveToNext() (exhausted bool, err error) {
	var res statedb.QueryResult
	if res, err = holder.itr.Next(); err != nil {
		return false, err
	}
	if res != nil {
		holder.kv = res.(*statedb.VersionedKV)
	}
	return res == nil, nil
}

//字符串返回持有者在缓冲区中用作下一个键的键
func (holder *itrHolder) String() string {
	return fmt.Sprintf("{%s}", holder.kv.Key)
}
