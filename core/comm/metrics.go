
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


package comm

import "github.com/hyperledger/fabric/common/metrics"

var (
	openConnCounterOpts = metrics.CounterOpts{
		Namespace: "grpc",
		Subsystem: "comm",
		Name:      "conn_opened",
		Help:      "gRPC connections opened. Open minus closed is the active number of connections.",
	}

	closedConnCounterOpts = metrics.CounterOpts{
		Namespace: "grpc",
		Subsystem: "comm",
		Name:      "conn_closed",
		Help:      "gRPC connections closed. Open minus closed is the active number of connections.",
	}
)

func NewServerStatsHandler(p metrics.Provider) *ServerStatsHandler {
	return &ServerStatsHandler{
		OpenConnCounter:   p.NewCounter(openConnCounterOpts),
		ClosedConnCounter: p.NewCounter(closedConnCounterOpts),
	}
}
