
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


package disabled

import (
	"github.com/hyperledger/fabric/common/metrics"
)

type Provider struct{}

func (p *Provider) NewCounter(o metrics.CounterOpts) metrics.Counter       { return &Counter{} }
func (p *Provider) NewGauge(o metrics.GaugeOpts) metrics.Gauge             { return &Gauge{} }
func (p *Provider) NewHistogram(o metrics.HistogramOpts) metrics.Histogram { return &Histogram{} }

type Counter struct{}

func (c *Counter) Add(delta float64) {}
func (c *Counter) With(labelValues ...string) metrics.Counter {
	return c
}

type Gauge struct{}

func (g *Gauge) Add(delta float64) {}
func (g *Gauge) Set(delta float64) {}
func (g *Gauge) With(labelValues ...string) metrics.Gauge {
	return g
}

type Histogram struct{}

func (h *Histogram) Observe(value float64) {}
func (h *Histogram) With(labelValues ...string) metrics.Histogram {
	return h
}
