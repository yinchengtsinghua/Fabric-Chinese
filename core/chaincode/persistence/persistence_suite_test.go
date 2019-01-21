
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


package persistence_test

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//
type ioReadWriter interface {
	persistence.IOReadWriter
}

//go：生成伪造者-o mock/osfileinfo.go-forke name osfileinfo。OsFielFig
type osFileInfo interface {
	os.FileInfo
}

//go:生成仿冒者-o mock/store_package_provider.go-fake name store package provider。存储包提供程序
type storePackageProvider interface {
	persistence.StorePackageProvider
}

//go：生成仿冒者-o mock/legacy package provider.go-仿冒名称legacy package provider。LegacyPackageProvider（legacyPackageProvider）
type legacyPackageProvider interface {
	persistence.LegacyPackageProvider
}

//go：生成仿冒者-o mock/package_parser.go-forke name package parser。打包分析器
type packageParser interface {
	persistence.PackageParser
}

func TestPersistence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}
