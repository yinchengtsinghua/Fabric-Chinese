
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
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/msp"
	pmsp "github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

const (
	deserializeIdentityCacheSize = 100
	validateIdentityCacheSize    = 100
	satisfiesPrincipalCacheSize  = 100
)

var mspLogger = flogging.MustGetLogger("msp")

func New(o msp.MSP) (msp.MSP, error) {
	mspLogger.Debugf("Creating Cache-MSP instance")
	if o == nil {
		return nil, errors.Errorf("Invalid passed MSP. It must be different from nil.")
	}

	theMsp := &cachedMSP{MSP: o}
	theMsp.deserializeIdentityCache = newSecondChanceCache(deserializeIdentityCacheSize)
	theMsp.satisfiesPrincipalCache = newSecondChanceCache(satisfiesPrincipalCacheSize)
	theMsp.validateIdentityCache = newSecondChanceCache(validateIdentityCacheSize)

	return theMsp, nil
}

type cachedMSP struct {
	msp.MSP

//用于反序列化IDentity的缓存。
	deserializeIdentityCache *secondChanceCache

//validateIdentity的缓存
	validateIdentityCache *secondChanceCache

//基本上是主体的映射=>身份=>字符串化为布尔值
//指定此标识是否满足此主体
	satisfiesPrincipalCache *secondChanceCache
}

type cachedIdentity struct {
	msp.Identity
	cache *cachedMSP
}

func (id *cachedIdentity) SatisfiesPrincipal(principal *pmsp.MSPPrincipal) error {
	return id.cache.SatisfiesPrincipal(id.Identity, principal)
}

func (id *cachedIdentity) Validate() error {
	return id.cache.Validate(id.Identity)
}

func (c *cachedMSP) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	id, ok := c.deserializeIdentityCache.get(string(serializedIdentity))
	if ok {
		return &cachedIdentity{
			cache:    c,
			Identity: id.(msp.Identity),
		}, nil
	}

	id, err := c.MSP.DeserializeIdentity(serializedIdentity)
	if err == nil {
		c.deserializeIdentityCache.add(string(serializedIdentity), id)
		return &cachedIdentity{
			cache:    c,
			Identity: id.(msp.Identity),
		}, nil
	}
	return nil, err
}

func (c *cachedMSP) Setup(config *pmsp.MSPConfig) error {
	c.cleanCash()

	return c.MSP.Setup(config)
}

func (c *cachedMSP) Validate(id msp.Identity) error {
	identifier := id.GetIdentifier()
	key := string(identifier.Mspid + ":" + identifier.Id)

	_, ok := c.validateIdentityCache.get(key)
	if ok {
//仅当标识有效时才存储缓存。
		return nil
	}

	err := c.MSP.Validate(id)
	if err == nil {
		c.validateIdentityCache.add(key, true)
	}

	return err
}

func (c *cachedMSP) SatisfiesPrincipal(id msp.Identity, principal *pmsp.MSPPrincipal) error {
	identifier := id.GetIdentifier()
	identityKey := string(identifier.Mspid + ":" + identifier.Id)
	principalKey := string(principal.PrincipalClassification) + string(principal.Principal)
	key := identityKey + principalKey

	v, ok := c.satisfiesPrincipalCache.get(key)
	if ok {
		if v == nil {
			return nil
		}

		return v.(error)
	}

	err := c.MSP.SatisfiesPrincipal(id, principal)

	c.satisfiesPrincipalCache.add(key, err)
	return err
}

func (c *cachedMSP) cleanCash() error {
	c.deserializeIdentityCache = newSecondChanceCache(deserializeIdentityCacheSize)
	c.satisfiesPrincipalCache = newSecondChanceCache(satisfiesPrincipalCacheSize)
	c.validateIdentityCache = newSecondChanceCache(validateIdentityCacheSize)

	return nil
}
