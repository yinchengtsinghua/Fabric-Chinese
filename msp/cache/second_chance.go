
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


package cache

import (
	"sync"
	"sync/atomic"
)

//该软件包实现了二次机会算法，一种近似的LRU算法。
//网址：https://www.cs.jhu.edu/~yairamir/cs418/os6/tsld023.htm

//secondchanceCache保存大小有限的关键值项。
//当缓存项目数超过限制时，将根据
//二次机会算法和清除
type secondChanceCache struct {
//管理键和项之间的映射
	table map[string]*cacheItem

//保存缓存项列表。
	items []*cacheItem

//指示项目列表中受害者的下一个候选项
	position int

//读取get的锁，写入add的锁
	rwlock sync.RWMutex
}

type cacheItem struct {
	key   string
	value interface{}
//调用get（）时设置为1。当受害者扫描时设置为0
	referenced int32
}

func newSecondChanceCache(cacheSize int) *secondChanceCache {
	var cache secondChanceCache
	cache.position = 0
	cache.items = make([]*cacheItem, cacheSize)
	cache.table = make(map[string]*cacheItem)

	return &cache
}

func (cache *secondChanceCache) len() int {
	cache.rwlock.RLock()
	defer cache.rwlock.RUnlock()

	return len(cache.table)
}

func (cache *secondChanceCache) get(key string) (interface{}, bool) {
	cache.rwlock.RLock()
	defer cache.rwlock.RUnlock()

	item, ok := cache.table[key]
	if !ok {
		return nil, false
	}

//引用位设置为真，表示最近访问过该项。
	atomic.StoreInt32(&item.referenced, 1)

	return item.value, true
}

func (cache *secondChanceCache) add(key string, value interface{}) {
	cache.rwlock.Lock()
	defer cache.rwlock.Unlock()

	if old, ok := cache.table[key]; ok {
		old.value = value
		atomic.StoreInt32(&old.referenced, 1)
		return
	}

	var item cacheItem
	item.key = key
	item.value = value
	atomic.StoreInt32(&item.referenced, 1)

	size := len(cache.items)
	num := len(cache.table)
	if num < size {
//缓存未满，请将新项目存储在列表末尾
		cache.table[key] = &item
		cache.items[num] = &item
		return
	}

//启动受害者扫描，因为缓存已满
	for {
//检查此项目是否最近被接受
		victim := cache.items[cache.position]
		if atomic.LoadInt32(&victim.referenced) == 0 {
//找到了一个受害者。删除它，并将新项目存储在此处。
			delete(cache.table, victim.key)
			cache.table[key] = &item
			cache.items[cache.position] = &item
			cache.position = (cache.position + 1) % size
			return
		}

//引用的位设置为false，以便清除此项
//除非在下一次受害者扫描前能进入
		atomic.StoreInt32(&victim.referenced, 0)
		cache.position = (cache.position + 1) % size
	}
}
