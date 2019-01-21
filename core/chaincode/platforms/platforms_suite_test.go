
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


package platforms_test

import (
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/platforms"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go：生成仿冒者-o mock/platform.go——仿冒名称平台。平台
type platform interface {
	platforms.Platform
}

//go：生成伪造者-o mock/package-writer.go——伪造名称package writer。包装作家
type packageWriter interface {
	platforms.PackageWriter
}

func TestPlatforms(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Platforms Suite")
}
