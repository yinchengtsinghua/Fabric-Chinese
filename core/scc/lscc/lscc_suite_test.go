
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


package lscc_test

import (
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/scc/lscc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:生成仿冒者-o mock/sysccprovider.go--仿冒名称systemchaincodeprovider。系统链代码提供程序
type systemChaincodeProvider interface {
	sysccprovider.SystemChaincodeProvider
}

//go：生成伪造者-o mock/query-executor.go--forke-name-query executor。查询执行器
type queryExecutor interface {
	ledger.QueryExecutor
}

//go：生成伪造者-o mock/fs_support.go--forke name filesystemsupport。文件系统支持
type fileSystemSupport interface {
	lscc.FilesystemSupport
}

//go：生成伪造者-o mock/cc_package.go--forke name cc package。CCC软件包
type ccPackage interface {
	ccprovider.CCPackage
}

//go：生成伪造者-o mock/chaincode_stub.go——伪造名称chaincode stub。链状短截线
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

//go：生成伪造者-o mock/state_query_iterator.go——伪造名称state query iterator。状态查询迭代器
type stateQueryIterator interface {
	shim.StateQueryIteratorInterface
}

func TestLscc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lscc Suite")
}
