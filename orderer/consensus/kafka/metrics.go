
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


package kafka

import (
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/metrics"

	gometrics "github.com/rcrowley/go-metrics"
)

/*

 根据以下文档：https://godoc.org/github.com/shopify/sarama sarama公开了以下度量集：

 +————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
 名称类型说明
 +————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
 传入字节速率米字节/秒读取所有代理
 broker的传入字节速率-<broker id>meter bytes/second read off a given broker
 传出字节速率米字节/秒注销所有代理
 代理的传出字节速率-<broker id>meter bytes/second已注销给定的代理
 请求率米请求数/秒发送给所有经纪人
 代理请求率-<broker id>meter requests/second sent to a given broker_
 请求大小柱状图所有代理的请求大小的字节分布
 请求代理大小-<broker id>柱状图指定代理请求大小的字节分布
 请求延迟（毫秒）柱状图所有代理的请求延迟（毫秒）分布
 针对代理的请求延迟（ms）-<broker id>柱状图针对给定代理的请求延迟（ms）分布
 响应率米响应/秒从所有经纪人处收到
 代理的响应率-<broker id>meter responses/second received from a given broker_
 响应大小柱状图所有代理的响应大小的字节分布
 代理的响应大小-<broker id>柱状图给定代理的响应大小分布（以字节为单位）
 +————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————

 +——————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
 名称类型说明
 +——————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
 批量大小柱状图所有主题每个分区请求发送的字节数分布
 Topic的批大小-<Topic>Histogram Distribution of the number of bytes sented per partition per request for a given topic
 记录发送速率米记录/秒发送到所有主题
 记录发送速率为topic-<topic>meter records/second sent to a given topic_
 每个请求的记录柱状图所有主题每个请求发送的记录数分布
 每个主题请求的记录-<topic>柱状图为给定主题每个请求发送的记录数分布
 压缩比直方图所有主题压缩比乘以100个记录批的分布
 主题压缩比-<topic>柱状图给定主题压缩比乘以100个记录批的分布
 +——————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————————
**/


const (
	IncomingByteRateName  = "incoming-byte-rate-for-broker-"
	OutgoingByteRateName  = "outgoing-byte-rate-for-broker-"
	RequestRateName       = "request-rate-for-broker-"
	RequestSizeName       = "request-size-for-broker-"
	RequestLatencyName    = "request-latency-in-ms-for-broker-"
	ResponseRateName      = "response-rate-for-broker-"
	ResponseSizeName      = "response-size-for-broker-"
	BatchSizeName         = "batch-size-for-topic-"
	RecordSendRateName    = "record-send-rate-for-topic-"
	RecordsPerRequestName = "records-per-request-for-topic-"
	CompressionRatioName  = "compression-ratio-for-topic-"
)

var (
	incomingByteRate = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "incoming_byte_rate",
		Help:         "Bytes/second read off brokers.",
		LabelNames:   []string{"broker_id"},
		StatsdFormat: "%{#fqname}.%{broker_id}",
	}

	outgoingByteRate = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "outgoing_byte_rate",
		Help:         "Bytes/second written to brokers.",
		LabelNames:   []string{"broker_id"},
		StatsdFormat: "%{#fqname}.%{broker_id}",
	}

	requestRate = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "request_rate",
		Help:         "Requests/second sent to brokers.",
		LabelNames:   []string{"broker_id"},
		StatsdFormat: "%{#fqname}.%{broker_id}",
	}

	requestSize = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "request_size",
		Help:         "The mean request size in bytes to brokers.",
		LabelNames:   []string{"broker_id"},
		StatsdFormat: "%{#fqname}.%{broker_id}",
	}

	requestLatency = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "request_latency",
		Help:         "The mean request latency in ms to brokers.",
		LabelNames:   []string{"broker_id"},
		StatsdFormat: "%{#fqname}.%{broker_id}",
	}

	responseRate = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "response_rate",
		Help:         "Requests/second sent to brokers.",
		LabelNames:   []string{"broker_id"},
		StatsdFormat: "%{#fqname}.%{broker_id}",
	}

	responseSize = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "response_size",
		Help:         "The mean response size in bytes from brokers.",
		LabelNames:   []string{"broker_id"},
		StatsdFormat: "%{#fqname}.%{broker_id}",
	}

	batchSize = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "batch_size",
		Help:         "The mean batch size in bytes sent to topics.",
		LabelNames:   []string{"topic"},
		StatsdFormat: "%{#fqname}.%{topic}",
	}

	recordSendRate = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "record_send_rate",
		Help:         "The number of records per second sent to topics.",
		LabelNames:   []string{"topic"},
		StatsdFormat: "%{#fqname}.%{topic}",
	}

	recordsPerRequest = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "records_per_request",
		Help:         "The mean number of records sent per request to topics.",
		LabelNames:   []string{"topic"},
		StatsdFormat: "%{#fqname}.%{topic}",
	}

	compressionRatio = metrics.GaugeOpts{
		Namespace:    "consensus",
		Subsystem:    "kafka",
		Name:         "compression_ratio",
		Help:         "The mean compression ratio (as percentage) for topics.",
		LabelNames:   []string{"topic"},
		StatsdFormat: "%{#fqname}.%{topic}",
	}
)

type Metrics struct {
	IncomingByteRate  metrics.Gauge
	OutgoingByteRate  metrics.Gauge
	RequestRate       metrics.Gauge
	RequestSize       metrics.Gauge
	RequestLatency    metrics.Gauge
	ResponseRate      metrics.Gauge
	ResponseSize      metrics.Gauge
	BatchSize         metrics.Gauge
	RecordSendRate    metrics.Gauge
	RecordsPerRequest metrics.Gauge
	CompressionRatio  metrics.Gauge

	GoMetricsRegistry gometrics.Registry
}

func NewMetrics(p metrics.Provider, registry gometrics.Registry) *Metrics {
	return &Metrics{
		IncomingByteRate:  p.NewGauge(incomingByteRate),
		OutgoingByteRate:  p.NewGauge(outgoingByteRate),
		RequestRate:       p.NewGauge(requestRate),
		RequestSize:       p.NewGauge(requestSize),
		RequestLatency:    p.NewGauge(requestLatency),
		ResponseRate:      p.NewGauge(responseRate),
		ResponseSize:      p.NewGauge(responseSize),
		BatchSize:         p.NewGauge(batchSize),
		RecordSendRate:    p.NewGauge(recordSendRate),
		RecordsPerRequest: p.NewGauge(recordsPerRequest),
		CompressionRatio:  p.NewGauge(compressionRatio),

		GoMetricsRegistry: registry,
	}
}

//Pollgometrics从Go度量中获取当前度量值并将其发布到
//通过Go Kit的度量标准暴露的仪表。
func (m *Metrics) PollGoMetrics() {
	m.GoMetricsRegistry.Each(func(name string, value interface{}) {
		recordMeter := func(prefix, label string, gauge metrics.Gauge) bool {
			if !strings.HasPrefix(name, prefix) {
				return false
			}

			meter, ok := value.(gometrics.Meter)
			if !ok {
				logger.Panicf("Expected metric with name %s to be of type Meter but was of type %T", name, value)
			}

			labelValue := name[len(prefix):]
			gauge.With(label, labelValue).Set(meter.Snapshot().Rate1())

			return true
		}

		recordHistogram := func(prefix, label string, gauge metrics.Gauge) bool {
			if !strings.HasPrefix(name, prefix) {
				return false
			}

			histogram, ok := value.(gometrics.Histogram)
			if !ok {
				logger.Panicf("Expected metric with name %s to be of type Histogram but was of type %T", name, value)
			}

			labelValue := name[len(prefix):]
			gauge.With(label, labelValue).Set(histogram.Snapshot().Mean())

			return true
		}

		switch {
		case recordMeter(IncomingByteRateName, "broker_id", m.IncomingByteRate):
		case recordMeter(OutgoingByteRateName, "broker_id", m.OutgoingByteRate):
		case recordMeter(RequestRateName, "broker_id", m.RequestRate):
		case recordHistogram(RequestSizeName, "broker_id", m.RequestSize):
		case recordHistogram(RequestLatencyName, "broker_id", m.RequestLatency):
		case recordMeter(ResponseRateName, "broker_id", m.ResponseRate):
		case recordHistogram(ResponseSizeName, "broker_id", m.ResponseSize):
		case recordHistogram(BatchSizeName, "topic", m.BatchSize):
		case recordMeter(RecordSendRateName, "topic", m.RecordSendRate):
		case recordHistogram(RecordsPerRequestName, "topic", m.RecordsPerRequest):
		case recordHistogram(CompressionRatioName, "topic", m.CompressionRatio):
		default:
//忽略未知指标
		}
	})
}

//pollgometricsuntilstop通常应在专用的go例程上调用。这个例行程序
//然后以指定的频率调用pollgometrics，直到stopChannel关闭。
func (m *Metrics) PollGoMetricsUntilStop(frequency time.Duration, stopChannel <-chan struct{}) {
	timer := time.NewTimer(frequency)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			m.PollGoMetrics()
			timer.Reset(frequency)
		case <-stopChannel:
			return
		}
	}
}
