
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


package broadcast_test

import (
	"testing"

	"github.com/hyperledger/fabric/common/metrics"
	ab "github.com/hyperledger/fabric/protos/orderer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go：生成伪造者-o mock/ab_server.go--forke name ab server。弃权者
type abServer interface {
	ab.AtomicBroadcast_BroadcastServer
}

//go：生成伪造者-o mock/metrics_histogram.go——伪造名称metrics histogram。测量记录
type metricsHistogram interface {
	metrics.Histogram
}

//go：生成伪造者-o mock/metrics_counter.go——伪造名称metrics counter。计量计数器
type metricsCounter interface {
	metrics.Counter
}

//go：生成伪造者-o mock/metrics_provider.go-伪造名称metricsProvider。度量提供者
type metricsProvider interface {
	metrics.Provider
}

func TestBroadcast(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broadcast Suite")
}
