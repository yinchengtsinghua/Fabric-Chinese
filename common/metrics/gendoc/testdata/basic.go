
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


package testdata

import (
	"time"

	"github.com/hyperledger/fabric/common/metrics"
)

//

var (
	Counter = metrics.CounterOpts{
		Namespace:    "fixtures",
		Name:         "counter",
		Help:         "This is some help text that is more than a few words long. It really can be quite long. Really long.",
		LabelNames:   []string{"label_one", "label_two"},
		StatsdFormat: "%{#fqname}.%{label_one}.%{label_two}",
	}

	Gauge = metrics.GaugeOpts{
		Namespace:    "fixtures",
		Name:         "gauge",
		Help:         "This is some help text",
		LabelNames:   []string{"label_one", "label_two"},
		StatsdFormat: "%{#fqname}.%{label_one}.%{label_two}",
	}

	Histogram = metrics.HistogramOpts{
		Namespace:    "fixtures",
		Name:         "histogram",
		Help:         "This is some help text",
		LabelNames:   []string{"label_one", "label_two"},
		StatsdFormat: "%{#fqname}.%{label_one}.%{label_two}",
	}

	ignoredStruct = struct{}{}

	ignoredInt = 0

	ignoredTime = time.Now()
)
