
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


package privacyenabledstate

import (
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
)

type metadataHint struct {
	cache      map[string]bool
	bookkeeper *leveldbhelper.DBHandle
}

func newMetadataHint(bookkeeper *leveldbhelper.DBHandle) *metadataHint {
	cache := map[string]bool{}
	itr := bookkeeper.GetIterator(nil, nil)
	defer itr.Release()
	for itr.Next() {
		namespace := string(itr.Key())
		cache[namespace] = true
	}
	return &metadataHint{cache, bookkeeper}
}

func (h *metadataHint) metadataEverUsedFor(namespace string) bool {
	return h.cache[namespace]
}

func (h *metadataHint) setMetadataUsedFlag(updates *UpdateBatch) {
	batch := leveldbhelper.NewUpdateBatch()
	for ns := range filterNamespacesThatHasMetadata(updates) {
		if h.cache[ns] {
			continue
		}
		h.cache[ns] = true
		batch.Put([]byte(ns), []byte{})
	}
	h.bookkeeper.WriteBatch(batch, true)
}

func filterNamespacesThatHasMetadata(updates *UpdateBatch) map[string]bool {
	namespaces := map[string]bool{}
	pubUpdates, hashUpdates := updates.PubUpdates, updates.HashUpdates
//为公共数据添加NS
	for _, ns := range pubUpdates.GetUpdatedNamespaces() {
		for _, vv := range updates.PubUpdates.GetUpdates(ns) {
			if vv.Metadata == nil {
				continue
			}
			namespaces[ns] = true
		}
	}
//为专用哈希添加ns
	for ns, nsBatch := range hashUpdates.UpdateMap {
		for _, coll := range nsBatch.GetCollectionNames() {
			for _, vv := range nsBatch.GetUpdates(coll) {
				if vv.Metadata == nil {
					continue
				}
				namespaces[ns] = true
			}
		}
	}
	return namespaces
}
