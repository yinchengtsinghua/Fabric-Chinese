
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


package bookkeeping

import (
	"fmt"

	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
)

//Category是一个枚举类型，用于表示不同类型的簿记。
type Category int

const (
//pvtdata expiry根据btl策略重新记录与pvtdata到期相关的簿记。
	PvtdataExpiry Category = iota
//MetadataPresenceIndicator维护有关是否为命名空间设置元数据的簿记。
	MetadataPresenceIndicator
)

//提供者为给定的分类帐向不同的簿记员提供处理
type Provider interface {
//getdbhandle返回可用于维护给定类别的簿记的db handle
	GetDBHandle(ledgerID string, cat Category) *leveldbhelper.DBHandle
//关闭关闭BookkeepProvider
	Close()
}

type provider struct {
	dbProvider *leveldbhelper.Provider
}

//NewProvider实例化新的提供程序
func NewProvider() Provider {
	dbProvider := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: getInternalBookkeeperPath()})
	return &provider{dbProvider: dbProvider}
}

//GetDBHandle实现接口“BookKeepProvider”中的函数
func (provider *provider) GetDBHandle(ledgerID string, cat Category) *leveldbhelper.DBHandle {
	return provider.dbProvider.GetDBHandle(fmt.Sprintf(ledgerID+"/%d", cat))
}

//close实现接口“bookkeeprovider”中的函数
func (provider *provider) Close() {
	provider.dbProvider.Close()
}

func getInternalBookkeeperPath() string {
	return ledgerconfig.GetInternalBookkeeperPath()
}
