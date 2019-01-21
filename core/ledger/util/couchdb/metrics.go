
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


package couchdb

import (
	"time"

	"github.com/hyperledger/fabric/common/metrics"
)

var (
	apiProcessingTimeOpts = metrics.HistogramOpts{
		Namespace:    "couchdb",
		Subsystem:    "",
		Name:         "processing_time",
		Help:         "Time taken in seconds for the function to complete request to CouchDB",
		LabelNames:   []string{"database", "function_name", "result"},
		StatsdFormat: "%{#fqname}.%{database}.%{function_name}.%{result}",
	}
)

type stats struct {
	apiProcessingTime metrics.Histogram
}

func newStats(metricsProvider metrics.Provider) *stats {
	return &stats{
		apiProcessingTime: metricsProvider.NewHistogram(apiProcessingTimeOpts),
	}
}

func (s *stats) observeProcessingTime(startTime time.Time, dbName, functionName, result string) {
	s.apiProcessingTime.With(
		"database", dbName,
		"function_name", functionName,
		"result", result,
	).Observe(time.Since(startTime).Seconds())
}
