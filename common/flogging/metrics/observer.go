
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

import (
	"github.com/hyperledger/fabric/common/metrics"
	"go.uber.org/zap/zapcore"
)

var (
	CheckedCountOpts = metrics.CounterOpts{
		Namespace:    "logging",
		Name:         "entries_checked",
		Help:         "Number of log entries checked against the active logging level",
		LabelNames:   []string{"level"},
		StatsdFormat: "%{#fqname}.%{level}",
	}

	WriteCountOpts = metrics.CounterOpts{
		Namespace:    "logging",
		Name:         "entries_written",
		Help:         "Number of log entries that are written",
		LabelNames:   []string{"level"},
		StatsdFormat: "%{#fqname}.%{level}",
	}
)

type Observer struct {
	CheckedCounter metrics.Counter
	WrittenCounter metrics.Counter
}

func NewObserver(provider metrics.Provider) *Observer {
	return &Observer{
		CheckedCounter: provider.NewCounter(CheckedCountOpts),
		WrittenCounter: provider.NewCounter(WriteCountOpts),
	}
}

func (m *Observer) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) {
	m.CheckedCounter.With("level", e.Level.String()).Add(1)
}

func (m *Observer) WriteEntry(e zapcore.Entry, fields []zapcore.Field) {
	m.WrittenCounter.With("level", e.Level.String()).Add(1)
}
