
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


package broadcast

import "github.com/hyperledger/fabric/common/metrics"

var (
	validateDuration = metrics.HistogramOpts{
		Namespace:    "broadcast",
		Name:         "validate_duration",
		Help:         "The time to validate a transaction in seconds.",
		LabelNames:   []string{"channel", "type", "status"},
		StatsdFormat: "%{#fqname}.%{channel}.%{type}.%{status}",
	}
	enqueueDuration = metrics.HistogramOpts{
		Namespace:    "broadcast",
		Name:         "enqueue_duration",
		Help:         "The time to enqueue a transaction in seconds.",
		LabelNames:   []string{"channel", "type", "status"},
		StatsdFormat: "%{#fqname}.%{channel}.%{type}.%{status}",
	}
	processedCount = metrics.CounterOpts{
		Namespace:    "broadcast",
		Name:         "processed_count",
		Help:         "The number of transactions processed.",
		LabelNames:   []string{"channel", "type", "status"},
		StatsdFormat: "%{#fqname}.%{channel}.%{type}.%{status}",
	}
)

type Metrics struct {
	ValidateDuration metrics.Histogram
	EnqueueDuration  metrics.Histogram
	ProcessedCount   metrics.Counter
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		ValidateDuration: p.NewHistogram(validateDuration),
		EnqueueDuration:  p.NewHistogram(enqueueDuration),
		ProcessedCount:   p.NewCounter(processedCount),
	}
}
