
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


package ledgermgmt

import (
	"fmt"
	"os"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/mock"
)

//initializetestenv初始化测试的ledgermgmt
func InitializeTestEnv() {
	remove()
	InitializeTestEnvWithInitializer(nil)
}

//initializeTestInvWithInitializer使用提供的初始值设定项初始化测试的ledgermgmt
func InitializeTestEnvWithInitializer(initializer *Initializer) {
	remove()
	InitializeExistingTestEnvWithInitializer(initializer)
}

//InitializeExistingTestenvWithInitializer为具有现有分类帐的测试初始化LedgerMgmt
//此功能不会删除现有分类帐，并在升级测试中使用。
//Todo Ledgermgmt应重新编写，以将包范围的函数移动到结构
func InitializeExistingTestEnvWithInitializer(initializer *Initializer) {
	if initializer == nil {
		initializer = &Initializer{}
	}
	if initializer.DeployedChaincodeInfoProvider == nil {
		initializer.DeployedChaincodeInfoProvider = &mock.DeployedChaincodeInfoProvider{}
	}
	if initializer.MetricsProvider == nil {
		initializer.MetricsProvider = &disabled.Provider{}
	}
	if initializer.PlatformRegistry == nil {
		initializer.PlatformRegistry = platforms.NewRegistry(&golang.Platform{})
	}
	initialize(initializer)
}

//cleanuptestenv关闭ledgermagmt并删除存储目录
func CleanupTestEnv() {
	Close()
	remove()
}

func remove() {
	path := ledgerconfig.GetRootPath()
	fmt.Printf("removing dir = %s\n", path)
	err := os.RemoveAll(path)
	if err != nil {
		logger.Errorf("Error: %s", err)
	}
}
