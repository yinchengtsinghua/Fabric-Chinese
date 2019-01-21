
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


package grpcmetrics

import "github.com/hyperledger/fabric/common/metrics"

var (
	unaryRequestDuration = metrics.HistogramOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "unary_request_duration",
		Help:         "The time to complete a unary request.",
		LabelNames:   []string{"service", "method", "code"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}.%{code}",
	}
	unaryRequestsReceived = metrics.CounterOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "unary_requests_received",
		Help:         "The number of unary requests received.",
		LabelNames:   []string{"service", "method"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}",
	}
	unaryRequestsCompleted = metrics.CounterOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "unary_requests_completed",
		Help:         "The number of unary requests completed.",
		LabelNames:   []string{"service", "method", "code"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}.%{code}",
	}

	streamRequestDuration = metrics.HistogramOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "stream_request_duration",
		Help:         "The time to complete a stream request.",
		LabelNames:   []string{"service", "method", "code"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}.%{code}",
	}
	streamRequestsReceived = metrics.CounterOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "stream_requests_received",
		Help:         "The number of stream requests received.",
		LabelNames:   []string{"service", "method"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}",
	}
	streamRequestsCompleted = metrics.CounterOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "stream_requests_completed",
		Help:         "The number of stream requests completed.",
		LabelNames:   []string{"service", "method", "code"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}.%{code}",
	}
	streamMessagesReceived = metrics.CounterOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "stream_messages_received",
		Help:         "The number of stream messages received.",
		LabelNames:   []string{"service", "method"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}",
	}
	streamMessagesSent = metrics.CounterOpts{
		Namespace:    "grpc",
		Subsystem:    "server",
		Name:         "stream_messages_sent",
		Help:         "The number of stream messages sent.",
		LabelNames:   []string{"service", "method"},
		StatsdFormat: "%{#fqname}.%{service}.%{method}",
	}
)

func NewUnaryMetrics(p metrics.Provider) *UnaryMetrics {
	return &UnaryMetrics{
		RequestDuration:   p.NewHistogram(unaryRequestDuration),
		RequestsReceived:  p.NewCounter(unaryRequestsReceived),
		RequestsCompleted: p.NewCounter(unaryRequestsCompleted),
	}
}

func NewStreamMetrics(p metrics.Provider) *StreamMetrics {
	return &StreamMetrics{
		RequestDuration:   p.NewHistogram(streamRequestDuration),
		RequestsReceived:  p.NewCounter(streamRequestsReceived),
		RequestsCompleted: p.NewCounter(streamRequestsCompleted),
		MessagesSent:      p.NewCounter(streamMessagesSent),
		MessagesReceived:  p.NewCounter(streamMessagesReceived),
	}
}
