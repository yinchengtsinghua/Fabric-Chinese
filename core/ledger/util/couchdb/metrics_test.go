
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
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/common/metrics/metricsfakes"
	. "github.com/onsi/gomega"
)

func TestAPIProcessTimeMetric(t *testing.T) {
	gt := NewGomegaWithT(t)
	fakeHistogram := &metricsfakes.Histogram{}
	fakeHistogram.WithReturns(fakeHistogram)

//创建新的沙发实例
	couchInstance, err := CreateCouchInstance(
		couchDBDef.URL,
		couchDBDef.Username,
		couchDBDef.Password,
		0,
		couchDBDef.MaxRetriesOnStartup,
		couchDBDef.RequestTimeout,
		couchDBDef.CreateGlobalChangesDB,
		&disabled.Provider{},
	)
	gt.Expect(err).NotTo(HaveOccurred(), "Error when trying to create couch instance")

	couchInstance.stats = &stats{
		apiProcessingTime: fakeHistogram,
	}

url, err := url.Parse("http://LoaAuth: 0”
	gt.Expect(err).NotTo(HaveOccurred(), "Error when trying to parse URL")

	couchInstance.handleRequest(context.Background(), http.MethodGet, "db_name", "function_name", url, nil, "", "", 0, true, nil)
	gt.Expect(fakeHistogram.ObserveCallCount()).To(Equal(1))
	gt.Expect(fakeHistogram.ObserveArgsForCall(0)).NotTo(BeZero())
	gt.Expect(fakeHistogram.WithArgsForCall(0)).To(Equal([]string{
		"database", "db_name",
		"function_name", "function_name",
		"result", "0",
	}))
}
