
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


package common

import (
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/gossip"
)

//privdata_common保存privdata和mocks包中使用的类型。
//为了避免循环依赖性而需要

//Digkey定义了一个摘要
//指定特定的哈希rwset
type DigKey struct {
	TxId       string
	Namespace  string
	Collection string
	BlockSeq   uint64
	SeqInBlock uint64
}

type Dig2CollectionConfig map[DigKey]*common.StaticCollectionConfig

//获取pvt数据元素的dpvttatacontainer容器
//回卷人退回
type FetchedPvtDataContainer struct {
	AvailableElements []*gossip.PvtDataElement
	PurgedElements    []*gossip.PvtDataDigest
}
