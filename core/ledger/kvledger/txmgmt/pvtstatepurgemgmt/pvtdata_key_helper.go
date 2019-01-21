
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


package pvtstatepurgemgmt

func (pvtdataKeys *PvtdataKeys) addAll(toAdd *PvtdataKeys) {
	for ns, colls := range toAdd.Map {
		for coll, keysAndHashes := range colls.Map {
			for _, k := range keysAndHashes.List {
				pvtdataKeys.add(ns, coll, k.Key, k.Hash)
			}
		}
	}
}

func (pvtdataKeys *PvtdataKeys) add(ns string, coll string, key string, keyhash []byte) {
	colls := pvtdataKeys.getOrCreateCollections(ns)
	keysAndHashes := colls.getOrCreateKeysAndHashes(coll)
	keysAndHashes.List = append(keysAndHashes.List, &KeyAndHash{Key: key, Hash: keyhash})
}

func (pvtdataKeys *PvtdataKeys) getOrCreateCollections(ns string) *Collections {
	colls, ok := pvtdataKeys.Map[ns]
	if !ok {
		colls = newCollections()
		pvtdataKeys.Map[ns] = colls
	}
	return colls
}

func (colls *Collections) getOrCreateKeysAndHashes(coll string) *KeysAndHashes {
	keysAndHashes, ok := colls.Map[coll]
	if !ok {
		keysAndHashes = &KeysAndHashes{}
		colls.Map[coll] = keysAndHashes
	}
	return keysAndHashes
}

func newPvtdataKeys() *PvtdataKeys {
	return &PvtdataKeys{Map: make(map[string]*Collections)}
}

func newCollections() *Collections {
	return &Collections{Map: make(map[string]*KeysAndHashes)}
}
