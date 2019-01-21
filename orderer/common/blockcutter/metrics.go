
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


package blockcutter

import "github.com/hyperledger/fabric/common/metrics"

var (
	blockFillDuration = metrics.HistogramOpts{
		Namespace:    "blockcutter",
		Name:         "block_fill_duration",
		Help:         "The time from first transaction enqueing to the block being cut in seconds.",
		LabelNames:   []string{"channel"},
		StatsdFormat: "%{#fqname}.%{channel}",
	}
)

type Metrics struct {
	BlockFillDuration metrics.Histogram
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		BlockFillDuration: p.NewHistogram(blockFillDuration),
	}
}
