
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


package lifecycle_test

import (
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/lifecycle"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go：生成伪造者-o mock/chaincode_stub.go——伪造名称chaincode stub。链状短截线
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

//go：生成仿冒者-o mock/chaincode\store.go——仿冒名称chaincode store。链式存储器
type chaincodeStore interface {
	lifecycle.ChaincodeStore
}

//go：生成伪造者-o mock/package_parser.go--forke name package parser。打包分析器
type packageParser interface {
	lifecycle.PackageParser
}

//go：生成仿冒者-o mock/scc_functions.go--fake name scc functions。SCC函数
type sccFunctions interface {
	lifecycle.SCCFunctions
}

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lifecycle Suite")
}
