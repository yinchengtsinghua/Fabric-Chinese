
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


package testdata

import (
	goo "github.com/hyperledger/fabric/common/metrics"
)

//这些变量应该被发现为有效的度量选项。
//即使使用了命名导入。

var (
	NamedCounter = goo.CounterOpts{
		Namespace:    "namespace",
		Subsystem:    "counter",
		Name:         "name",
		Help:         "This is some help text",
		LabelNames:   []string{"label_one", "label_two"},
		StatsdFormat: "%{#fqname}.%{label_one}.%{label_two}",
	}

	NamedGauge = goo.GaugeOpts{
		Namespace:    "namespace",
		Subsystem:    "gauge",
		Name:         "name",
		Help:         "This is some help text",
		LabelNames:   []string{"label_one", "label_two"},
		StatsdFormat: "%{#fqname}.%{label_one}.%{label_two}",
	}

	NamedHistogram = goo.HistogramOpts{
		Namespace:    "namespace",
		Subsystem:    "histogram",
		Name:         "name",
		Help:         "This is some help text",
		LabelNames:   []string{"label_one", "label_two"},
		StatsdFormat: "%{#fqname}.%{label_one}.%{label_two}",
	}
)
