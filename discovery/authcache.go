
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


package discovery

import (
	"encoding/asn1"
	"encoding/hex"
	"sync"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

const (
	defaultMaxCacheSize   = 1000
	defaultRetentionRatio = 0.75
)

var (
//asbytes是用于将common.signeddata封送到bytes的函数。
	asBytes = asn1.Marshal
)

type acSupport interface {
//合格返回给定对等方是否有资格接收
//来自给定通道的发现服务的服务
	EligibleForService(channel string, data common.SignedData) error

//ConfigSequence returns the configuration sequence of the given channel
	ConfigSequence(channel string) uint64
}

type authCacheConfig struct {
	enabled bool
//MaxCacheSize是缓存的最大大小，之后
//进行净化
	maxCacheSize int
//purgeRetentionRatio is the % of entries that remain in the cache
//在清除缓存后，由于人口过多
	purgeRetentionRatio float64
}

//AuthCache定义了一个接口，用于对通道上下文中的请求进行身份验证，
//以及呼叫的记忆化
type authCache struct {
	credentialCache map[string]*accessCache
	acSupport
	sync.RWMutex
	conf authCacheConfig
}

func newAuthCache(s acSupport, conf authCacheConfig) *authCache {
	return &authCache{
		acSupport:       s,
		credentialCache: make(map[string]*accessCache),
		conf:            conf,
	}
}

//合格返回给定对等方是否有资格接收
//来自给定通道的发现服务的服务
func (ac *authCache) EligibleForService(channel string, data common.SignedData) error {
	if !ac.conf.enabled {
		return ac.acSupport.EligibleForService(channel, data)
	}
//检查是否已经有此通道的缓存
	ac.RLock()
	cache := ac.credentialCache[channel]
	ac.RUnlock()
	if cache == nil {
//找不到给定频道的缓存，请创建一个新频道
		ac.Lock()
		cache = ac.newAccessCache(channel)
//并存储缓存实例。
		ac.credentialCache[channel] = cache
		ac.Unlock()
	}
	return cache.EligibleForService(data)
}

type accessCache struct {
	sync.RWMutex
	channel      string
	ac           *authCache
	lastSequence uint64
	entries      map[string]error
}

func (ac *authCache) newAccessCache(channel string) *accessCache {
	return &accessCache{
		channel: channel,
		ac:      ac,
		entries: make(map[string]error),
	}
}

func (cache *accessCache) EligibleForService(data common.SignedData) error {
	key, err := signedDataToKey(data)
	if err != nil {
		logger.Warningf("Failed computing key of signed data: +%v", err)
		return errors.Wrap(err, "failed computing key of signed data")
	}
	currSeq := cache.ac.acSupport.ConfigSequence(cache.channel)
	if cache.isValid(currSeq) {
		foundInCache, isEligibleErr := cache.lookup(key)
		if foundInCache {
			return isEligibleErr
		}
	} else {
		cache.configChange(currSeq)
	}

//确保缓存不会过度填充。
//它可能会因为并发而过度增长最大大小。
//Goroutines在上面等待锁，但这是可以接受的。
	cache.purgeEntriesIfNeeded()

//计算客户的服务资格
	err = cache.ac.acSupport.EligibleForService(cache.channel, data)
	cache.Lock()
	defer cache.Unlock()
//检查序列是否自上次以来没有更改
	if currSeq != cache.ac.acSupport.ConfigSequence(cache.channel) {
//我们计算合格性的顺序可能已经改变了，
//所以我们不能把它放到缓存中，因为这是一个更新的计算结果
//现在可能已经存在于缓存中，我们不想覆盖它
//with a stale computation result, so just return the result.
		return err
	}
//否则，根据最新的顺序计算客户的资格，
//所以将计算结果存储在缓存中
	cache.entries[key] = err
	return err
}

func (cache *accessCache) isPurgeNeeded() bool {
	cache.RLock()
	defer cache.RUnlock()
	return len(cache.entries)+1 > cache.ac.conf.maxCacheSize
}

func (cache *accessCache) purgeEntriesIfNeeded() {
	if !cache.isPurgeNeeded() {
		return
	}

	cache.Lock()
	defer cache.Unlock()

	maxCacheSize := cache.ac.conf.maxCacheSize
	purgeRatio := cache.ac.conf.purgeRetentionRatio
	entries2evict := maxCacheSize - int(purgeRatio*float64(maxCacheSize))

	for key := range cache.entries {
		if entries2evict == 0 {
			return
		}
		entries2evict--
		delete(cache.entries, key)
	}
}

func (cache *accessCache) isValid(currSeq uint64) bool {
	cache.RLock()
	defer cache.RUnlock()
	return currSeq == cache.lastSequence
}

func (cache *accessCache) configChange(currSeq uint64) {
	cache.Lock()
	defer cache.Unlock()
	cache.lastSequence = currSeq
//使条目无效
	cache.entries = make(map[string]error)
}

func (cache *accessCache) lookup(key string) (cacheHit bool, lookupResult error) {
	cache.RLock()
	defer cache.RUnlock()

	lookupResult, cacheHit = cache.entries[key]
	return
}

func signedDataToKey(data common.SignedData) (string, error) {
	b, err := asBytes(data)
	if err != nil {
		return "", errors.Wrap(err, "failed marshaling signed data")
	}
	return hex.EncodeToString(util.ComputeSHA256(b)), nil
}
