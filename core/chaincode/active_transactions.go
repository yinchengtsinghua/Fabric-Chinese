
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


package chaincode

import "sync"

func NewTxKey(channelID, txID string) string { return channelID + txID }

type ActiveTransactions struct {
	mutex sync.Mutex
	ids   map[string]struct{}
}

func NewActiveTransactions() *ActiveTransactions {
	return &ActiveTransactions{
		ids: map[string]struct{}{},
	}
}

func (a *ActiveTransactions) Add(channelID, txID string) bool {
	key := NewTxKey(channelID, txID)
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if _, ok := a.ids[key]; ok {
		return false
	}

	a.ids[key] = struct{}{}
	return true
}

func (a *ActiveTransactions) Remove(channelID, txID string) {
	key := NewTxKey(channelID, txID)
	a.mutex.Lock()
	delete(a.ids, key)
	a.mutex.Unlock()
}
