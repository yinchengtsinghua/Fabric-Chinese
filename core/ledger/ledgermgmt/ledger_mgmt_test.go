
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package ledgermgmt

import (
	"fmt"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"

	"github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/ledgermgmt")
	os.Exit(m.Run())
}

func TestLedgerMgmt(t *testing.T) {
//检查在未初始化的情况下创建/打开分类帐时是否出错。
	gb, _ := test.MakeGenesisBlock(constructTestLedgerID(0))
	l, err := CreateLedger(gb)
	assert.Nil(t, l)
	assert.Equal(t, ErrLedgerMgmtNotInitialized, err)

	ledgerID := constructTestLedgerID(2)
	l, err = OpenLedger(ledgerID)
	assert.Nil(t, l)
	assert.Equal(t, ErrLedgerMgmtNotInitialized, err)

	ids, err := GetLedgerIDs()
	assert.Nil(t, ids)
	assert.Equal(t, ErrLedgerMgmtNotInitialized, err)

	Close()

	InitializeTestEnv()
	defer CleanupTestEnv()

	numLedgers := 10
	ledgers := make([]ledger.PeerLedger, numLedgers)
	for i := 0; i < numLedgers; i++ {
		gb, _ := test.MakeGenesisBlock(constructTestLedgerID(i))
		l, _ := CreateLedger(gb)
		ledgers[i] = l
	}

	ids, _ = GetLedgerIDs()
	assert.Len(t, ids, numLedgers)
	for i := 0; i < numLedgers; i++ {
		assert.Equal(t, constructTestLedgerID(i), ids[i])
	}

	ledgerID = constructTestLedgerID(2)
	t.Logf("Ledger selected for test = %s", ledgerID)
	_, err = OpenLedger(ledgerID)
	assert.Equal(t, ErrLedgerAlreadyOpened, err)

	l = ledgers[2]
	l.Close()
	l, err = OpenLedger(ledgerID)
	assert.NoError(t, err)

	l, err = OpenLedger(ledgerID)
	assert.Equal(t, ErrLedgerAlreadyOpened, err)

//关闭所有已打开的分类帐和分类帐管理
	Close()

//使用现有分类帐重新启动分类帐管理
	Initialize(&Initializer{
		PlatformRegistry: platforms.NewRegistry(&golang.Platform{}),
		MetricsProvider:  &disabled.Provider{},
	})
	l, err = OpenLedger(ledgerID)
	assert.NoError(t, err)
	Close()
}

func TestChaincodeInfoProvider(t *testing.T) {
	InitializeTestEnv()
	defer CleanupTestEnv()
	gb, _ := test.MakeGenesisBlock("ledger1")
	CreateLedger(gb)

	mockDeployedCCInfoProvider := &mock.DeployedChaincodeInfoProvider{}
	mockDeployedCCInfoProvider.ChaincodeInfoStub = func(ccName string, qe ledger.SimpleQueryExecutor) (*ledger.DeployedChaincodeInfo, error) {
		return constructTestCCInfo(ccName, ccName, ccName), nil
	}

	ccInfoProvider := &chaincodeInfoProviderImpl{
		platforms.NewRegistry(&golang.Platform{}),
		mockDeployedCCInfoProvider,
	}
	_, err := ccInfoProvider.GetDeployedChaincodeInfo("ledger2", constructTestCCDef("cc2", "1.0", "cc2Hash"))
	t.Logf("Expected error received = %s", err)
	assert.Error(t, err)

	ccInfo, err := ccInfoProvider.GetDeployedChaincodeInfo("ledger1", constructTestCCDef("cc1", "non-matching-version", "cc1"))
	assert.NoError(t, err)
	assert.Nil(t, ccInfo)

	ccInfo, err = ccInfoProvider.GetDeployedChaincodeInfo("ledger1", constructTestCCDef("cc1", "cc1", "non-matching-hash"))
	assert.NoError(t, err)
	assert.Nil(t, ccInfo)

	ccInfo, err = ccInfoProvider.GetDeployedChaincodeInfo("ledger1", constructTestCCDef("cc1", "cc1", "cc1"))
	assert.NoError(t, err)
	assert.Equal(t, constructTestCCInfo("cc1", "cc1", "cc1"), ccInfo)
}

func constructTestLedgerID(i int) string {
	return fmt.Sprintf("ledger_%06d", i)
}

func constructTestCCInfo(ccName, version, hash string) *ledger.DeployedChaincodeInfo {
	return &ledger.DeployedChaincodeInfo{
		Name:    ccName,
		Hash:    []byte(hash),
		Version: version,
	}
}

func constructTestCCDef(ccName, version, hash string) *cceventmgmt.ChaincodeDefinition {
	return &cceventmgmt.ChaincodeDefinition{
		Name:    ccName,
		Hash:    []byte(hash),
		Version: version,
	}
}
