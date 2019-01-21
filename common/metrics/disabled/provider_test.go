
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


package disabled_test

import (
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider", func() {
	var p metrics.Provider

	BeforeEach(func() {
		p = &disabled.Provider{}
	})

	Describe("NewCounter", func() {
		It("creates a no-op counter that doesn't blow up", func() {
			c := p.NewCounter(metrics.CounterOpts{})
			Expect(c).NotTo(BeNil())

			c.Add(1)
			c.With("whatever").Add(2)
		})
	})

	Describe("NewGauge", func() {
		It("creates a no-op gauge that doesn't blow up", func() {
			g := p.NewGauge(metrics.GaugeOpts{})
			Expect(g).NotTo(BeNil())

			g.Set(1)
			g.Add(1)
			g.With("whatever").Set(2)
			g.With("whatever").Add(2)
		})
	})

	Describe("NewHistogram", func() {
		It("creates a no-op histogram that doesn't blow up", func() {
			h := p.NewHistogram(metrics.HistogramOpts{})
			Expect(h).NotTo(BeNil())

			h.Observe(1)
			h.With("whatever").Observe(2)
		})
	})
})
