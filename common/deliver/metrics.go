
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
*/


package deliver

import (
	"github.com/hyperledger/fabric/common/metrics"
)

var (
	streamsOpened = metrics.CounterOpts{
		Namespace: "deliver",
		Name:      "streams_opened",
		Help:      "The number of GRPC streams that have been opened for the deliver service.",
	}
	streamsClosed = metrics.CounterOpts{
		Namespace: "deliver",
		Name:      "streams_closed",
		Help:      "The number of GRPC streams that have been closed for the deliver service.",
	}

	requestsReceived = metrics.CounterOpts{
		Namespace:    "deliver",
		Name:         "requests_received",
		Help:         "The number of deliver requests that have been received.",
		LabelNames:   []string{"channel", "filtered"},
		StatsdFormat: "%{#fqname}.%{channel}.%{filtered}",
	}
	requestsCompleted = metrics.CounterOpts{
		Namespace:    "deliver",
		Name:         "requests_completed",
		Help:         "The number of deliver requests that have been completed.",
		LabelNames:   []string{"channel", "filtered", "success"},
		StatsdFormat: "%{#fqname}.%{channel}.%{filtered}.%{success}",
	}

	blocksSent = metrics.CounterOpts{
		Namespace:    "deliver",
		Name:         "blocks_sent",
		Help:         "The number of blocks sent by the deliver service.",
		LabelNames:   []string{"channel", "filtered"},
		StatsdFormat: "%{#fqname}.%{channel}.%{filtered}",
	}
)

type Metrics struct {
	StreamsOpened     metrics.Counter
	StreamsClosed     metrics.Counter
	RequestsReceived  metrics.Counter
	RequestsCompleted metrics.Counter
	BlocksSent        metrics.Counter
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		StreamsOpened:     p.NewCounter(streamsOpened),
		StreamsClosed:     p.NewCounter(streamsClosed),
		RequestsReceived:  p.NewCounter(requestsReceived),
		RequestsCompleted: p.NewCounter(requestsCompleted),
		BlocksSent:        p.NewCounter(blocksSent),
	}
}
