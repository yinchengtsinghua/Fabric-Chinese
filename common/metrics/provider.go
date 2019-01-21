
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


package metrics

//
//
type Provider interface {
//
	NewCounter(CounterOpts) Counter
//NewGauge创建仪表的新实例。
	NewGauge(GaugeOpts) Gauge
//
	NewHistogram(HistogramOpts) Histogram
}

//
type Counter interface {
//
//用于为提供给counteropts的所有labelname提供值。
	With(labelValues ...string) Counter

//
	Add(delta float64)
}

//
//度量子系统。
type CounterOpts struct {
//命名空间、子系统和名称是完全限定名的组件
//
//
//
	Namespace string
	Subsystem string
	Name      string

//
	Help string

//
//
//
	LabelNames []string

//
//
//
//
//
//
//
//
//
//-label_name-与命名标签关联的值
//
//
	StatsdFormat string
}

//仪表是表示某个度量的当前值的仪表。
type Gauge interface {
//用于在记录仪表值时提供标签值。这个
//
	With(labelValues ...string) Gauge

//
Add(delta float64) //

//set用于更新与仪表关联的当前值。
	Set(value float64)
}

//
//度量子系统。
type GaugeOpts struct {
//命名空间、子系统和名称是完全限定名的组件
//度量的。完全合格的名称是通过加入这些
//
//
	Namespace string
	Subsystem string
	Name      string

//
	Help string

//
//
//
	LabelNames []string

//statsdformat确定完全限定的statsd bucket名称是如何
//
//在`%引用`转义序列中包括字段引用。
//
//
//
//
//-name-name的值
//-fqname-完全限定的度量名称
//-label_name-与命名标签关联的值
//
//
	StatsdFormat string
}

//
//
type Histogram interface {
//
//
//
	With(labelValues ...string) Histogram
	Observe(value float64)
}

//
//度量子系统。
type HistogramOpts struct {
//命名空间、子系统和名称是完全限定名的组件
//度量的。完全合格的名称是通过加入这些
//
//其他人只是帮助组织这个名字。
	Namespace string
	Subsystem string
	Name      string

//帮助提供有关此度量的信息。
	Help string

//桶可以用来为普罗米修斯提供桶边界。什么时候？
//
	Buckets []float64

//
//
//
	LabelNames []string

//statsdformat确定完全限定的statsd bucket名称是如何
//从名称空间、子系统、名称和标签构建。这是通过
//在`%引用`转义序列中包括字段引用。
//
//
//
//-子系统-子系统的值
//-name-name的值
//-fqname-完全限定的度量名称
//-label_name-与命名标签关联的值
//
//
	StatsdFormat string
}
