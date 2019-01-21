
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


package metrics_test

import (
	"testing"

	"github.com/hyperledger/fabric/common/flogging/metrics"
	commonmetrics "github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/metricsfakes"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestNewObserver(t *testing.T) {
	provider := &metricsfakes.Provider{}
	checkedCounter := &metricsfakes.Counter{}
	writtenCounter := &metricsfakes.Counter{}
	provider.NewCounterStub = func(c commonmetrics.CounterOpts) commonmetrics.Counter {
		switch c.Name {
		case "entries_checked":
			assert.Equal(t, metrics.CheckedCountOpts, c)
			return checkedCounter
		case "entries_written":
			assert.Equal(t, metrics.WriteCountOpts, c)
			return writtenCounter
		default:
			return nil
		}
	}

	expectedObserver := &metrics.Observer{
		CheckedCounter: checkedCounter,
		WrittenCounter: writtenCounter,
	}
	m := metrics.NewObserver(provider)
	assert.Equal(t, expectedObserver, m)
	assert.Equal(t, 2, provider.NewCounterCallCount())
}

func TestCheck(t *testing.T) {
	counter := &metricsfakes.Counter{}
	counter.WithReturns(counter)

	m := metrics.Observer{CheckedCounter: counter}
	entry := zapcore.Entry{Level: zapcore.DebugLevel}
	checkedEntry := &zapcore.CheckedEntry{}
	m.Check(entry, checkedEntry)

	assert.Equal(t, 1, counter.WithCallCount())
	assert.Equal(t, []string{"level", "debug"}, counter.WithArgsForCall(0))

	assert.Equal(t, 1, counter.AddCallCount())
	assert.Equal(t, float64(1), counter.AddArgsForCall(0))
}

func TestWrite(t *testing.T) {
	counter := &metricsfakes.Counter{}
	counter.WithReturns(counter)

	m := metrics.Observer{WrittenCounter: counter}
	entry := zapcore.Entry{Level: zapcore.DebugLevel}
	m.WriteEntry(entry, nil)

	assert.Equal(t, 1, counter.WithCallCount())
	assert.Equal(t, []string{"level", "debug"}, counter.WithArgsForCall(0))

	assert.Equal(t, 1, counter.AddCallCount())
	assert.Equal(t, float64(1), counter.AddArgsForCall(0))
}
