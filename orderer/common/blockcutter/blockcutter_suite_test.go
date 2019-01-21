
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


package blockcutter_test

import (
	"testing"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/orderer/common/blockcutter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go：生成伪造者-o mock/metrics_histogram.go——伪造名称metrics histogram。测量记录
type metricsHistogram interface {
	metrics.Histogram
}

//go：生成伪造者-o mock/metrics_provider.go-伪造名称metricsProvider。度量提供者
type metricsProvider interface {
	metrics.Provider
}

//go：生成仿冒者-o mock/config-fetcher.go--forke-name-orderconfigfetcher。订单控制器
type ordererConfigFetcher interface {
	blockcutter.OrdererConfigFetcher
}

//go：生成伪造者-o mock/order_config.go--forke name orderconfig。有序配置
type ordererConfig interface {
	channelconfig.Orderer
}

func TestBlockcutter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Blockcutter Suite")
}
