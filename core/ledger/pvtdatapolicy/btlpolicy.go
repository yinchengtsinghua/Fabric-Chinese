
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


package pvtdatapolicy

import (
	"math"
	"sync"

	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/protos/common"
)

var defaultBTL uint64 = math.MaxUint64

//btlpolicy blocktoLive策略用于pvt数据
type BTLPolicy interface {
//getbtl返回给定命名空间和集合的blocktoLive
	GetBTL(ns string, coll string) (uint64, error)
//GetExpiringBlock返回给定命名空间、集合和CommitingBlock的pvtData的到期块号。
	GetExpiringBlock(namesapce string, collection string, committingBlock uint64) (uint64, error)
}

//lsccbasedbtlpolicy实现接口btlpolicy。
//此实现从已填充的lscc命名空间加载btl策略
//链码初始化期间的集合配置
type LSCCBasedBTLPolicy struct {
	collInfoProvider collectionInfoProvider
	cache            map[btlkey]uint64
	lock             sync.Mutex
}

type btlkey struct {
	ns   string
	coll string
}

//constructbtlpolicy构造lsccbasedbtlpolicy的实例
func ConstructBTLPolicy(collInfoProvider collectionInfoProvider) BTLPolicy {
	return &LSCCBasedBTLPolicy{
		collInfoProvider: collInfoProvider,
		cache:            make(map[btlkey]uint64),
	}
}

//getbtl在接口“btlpolicymgr”中实现相应的函数
func (p *LSCCBasedBTLPolicy) GetBTL(namesapce string, collection string) (uint64, error) {
	var btl uint64
	var ok bool
	key := btlkey{namesapce, collection}
	p.lock.Lock()
	defer p.lock.Unlock()
	btl, ok = p.cache[key]
	if !ok {
		collConfig, err := p.collInfoProvider.CollectionInfo(namesapce, collection)
		if err != nil {
			return 0, err
		}
		if collConfig == nil {
			return 0, privdata.NoSuchCollectionError{Namespace: namesapce, Collection: collection}
		}
		btlConfigured := collConfig.BlockToLive
		if btlConfigured > 0 {
			btl = uint64(btlConfigured)
		} else {
			btl = defaultBTL
		}
		p.cache[key] = btl
	}
	return btl, nil
}

//GetExpiringBlock从接口“btlpolicy”实现函数
func (p *LSCCBasedBTLPolicy) GetExpiringBlock(namesapce string, collection string, committingBlock uint64) (uint64, error) {
	btl, err := p.GetBTL(namesapce, collection)
	if err != nil {
		return 0, err
	}
	expiryBlk := committingBlock + btl + uint64(1)
if expiryBlk <= committingBlock { //committingblk+btl溢出uint64 max
		expiryBlk = math.MaxUint64
	}
	return expiryBlk, nil
}

type collectionInfoProvider interface {
	CollectionInfo(chaincodeName, collectionName string) (*common.StaticCollectionConfig, error)
}

//go：生成仿冒者-o mock/coll_info_provider.go-forke name collectioninfooprovider。集合信息提供者
