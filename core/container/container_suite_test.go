
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

package container_test

import (
	"testing"

	"github.com/hyperledger/fabric/core/container"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go：生成伪造者-o mock/vm_provider.go--forke name vm provider。VM提供者
type vmProvider interface {
	container.VMProvider
}

//go：生成仿冒者-o mock/vm.go——仿冒名称vm。虚拟机
type vm interface {
	container.VM
}

//go：生成伪造者-o mock/vm req.go--forke name vmcreq。VMCREQ
type vmcReq interface {
	container.VMCReq
}

//go：生成伪造者-o mock/builder.go——伪造名称builder。建设者
type builder interface {
	container.Builder
}

func TestContainer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Container Suite")
}
