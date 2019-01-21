
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


package statecouchdb

import (
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
)

type versions map[string]nsVersions
type revisions map[string]nsRevisions
type nsRevisions map[string]string
type nsVersions map[string]*version.Height

//versionscache包含版本和修订的映射。
//在块的批量处理期间用作本地缓存。
//版本-包含提交的版本并用于读取集的状态验证
//修订-包含已提交的修订，并在提交阶段用于couchdb批量更新
type versionsCache struct {
	vers versions
	revs revisions
}

func newVersionCache() *versionsCache {
	return &versionsCache{make(versions), make(revisions)}
}

func (c *versionsCache) getVersion(ns, key string) (*version.Height, bool) {
	ver, ok := c.vers[ns][key]
	if ok {
		return ver, true
	}
	return nil, false
}

//setverandrev为给定的ns/key将给定的版本和沙发版本设置为缓存
//在批量加载版本以进行读集验证期间调用此函数。
//验证不需要这些修订，但在提交期间使用它们
//写字板放在沙发上。在
//因为在一个典型的工作负载中，预期会有一个很好的重叠
//between the read-set and the write-set. During the commit, we load missing revisions for
//与读取集中没有读取对应的写入集中的任何其他写入
func (c *versionsCache) setVerAndRev(ns, key string, ver *version.Height, rev string) {
	_, ok := c.vers[ns]
	if !ok {
		c.vers[ns] = make(nsVersions)
		c.revs[ns] = make(nsRevisions)
	}
	c.vers[ns][key] = ver
	c.revs[ns][key] = rev
}
