
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


package grpcmetrics_test

import (
	"testing"

	"github.com/hyperledger/fabric/common/grpcmetrics/testpb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate protoc--proto_path=$go path/src/github.com/hyperledger/fabric/common/grpcmetrics/testpb--go_out=plugins=grpc:$go path/src$go path/src/github.com/hyperledger/fabric/common/grpcmetrics/testpb/echo.proto

func TestGrpcmetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Grpcmetrics Suite")
}

//go：生成仿冒者-o fakes/echo_service.go--echoserviceserver的假名。EchoServiceServer（EchoServiceServer）

type echoServiceServer interface {
	testpb.EchoServiceServer
}
