
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


package confighistory

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("confighistory=debug")
	os.Exit(m.Run())
}

func TestWithNoCollectionConfig(t *testing.T) {
	dbPath := "/tmp/fabric/core/ledger/confighistory"
	mockCCInfoProvider := &mock.DeployedChaincodeInfoProvider{}
	env := newTestEnv(t, dbPath, mockCCInfoProvider)
	mgr := env.mgr
	defer env.cleanup()
	testutilEquipMockCCInfoProviderToReturnDesiredCollConfig(mockCCInfoProvider, "chaincode1", nil)
	err := mgr.HandleStateUpdates(&ledger.StateUpdateTrigger{
		LedgerID:           "ledger1",
		CommittingBlockNum: 50},
	)
	assert.NoError(t, err)
	dummyLedgerInfoRetriever := &dummyLedgerInfoRetriever{info: &common.BlockchainInfo{Height: 100}}
	retriever := mgr.GetRetriever("ledger1", dummyLedgerInfoRetriever)
	collConfig, err := retriever.MostRecentCollectionConfigBelow(90, "chaincode1")
	assert.NoError(t, err)
	assert.Nil(t, collConfig)
}

func TestMgr(t *testing.T) {
	dbPath := "/tmp/fabric/core/ledger/confighistory"
	mockCCInfoProvider := &mock.DeployedChaincodeInfoProvider{}
	env := newTestEnv(t, dbPath, mockCCInfoProvider)
	mgr := env.mgr
	defer env.cleanup()
	chaincodeName := "chaincode1"
	maxBlockNumberInLedger := uint64(2000)
	dummyLedgerInfoRetriever := &dummyLedgerInfoRetriever{info: &common.BlockchainInfo{Height: maxBlockNumberInLedger + 1}}
	configCommittingBlockNums := []uint64{5, 10, 15, 100}
	ledgerIds := []string{"ledgerid1", "ledger2"}

//填充集合配置版本
	for _, ledgerid := range ledgerIds {
		for _, committingBlockNum := range configCommittingBlockNums {
//对于每个ledgerid和commitheight组合，构建一个独特的collconfigpackage并引入一个stateupdate
			collConfigPackage := sampleCollectionConfigPackage(ledgerid, committingBlockNum)
			testutilEquipMockCCInfoProviderToReturnDesiredCollConfig(mockCCInfoProvider, chaincodeName, collConfigPackage)
			mgr.HandleStateUpdates(&ledger.StateUpdateTrigger{
				LedgerID:           ledgerid,
				CommittingBlockNum: committingBlockNum},
			)
		}
	}

	t.Run("test-api-MostRecentCollectionConfigBelow()", func(t *testing.T) {
//一种包含条目的映射，对于类型为<k，v>的每个条目，
//我们检索“k”的“mostrecentcollectionconfigbelow”和预期值
//应在“v”提交配置
		m := map[uint64]uint64{math.MaxUint64: 100, 1000: 100, 50: 15, 12: 10, 7: 5}
		for _, ledgerid := range ledgerIds {
			retriever := mgr.GetRetriever(ledgerid, dummyLedgerInfoRetriever)
			for testHeight, expectedHeight := range m {
				retrievedConfig, err := retriever.MostRecentCollectionConfigBelow(testHeight, chaincodeName)
				assert.NoError(t, err)
				expectedConfig := sampleCollectionConfigPackage(ledgerid, expectedHeight)
				assert.Equal(t, expectedConfig, retrievedConfig.CollectionConfig)
				assert.Equal(t, expectedHeight, retrievedConfig.CommittingBlockNum)
			}

			retrievedConfig, err := retriever.MostRecentCollectionConfigBelow(5, chaincodeName)
			assert.NoError(t, err)
			assert.Nil(t, retrievedConfig)
		}
	})

	t.Run("test-api-CollectionConfigAt()", func(t *testing.T) {
		for _, ledgerid := range ledgerIds {
			retriever := mgr.GetRetriever(ledgerid, dummyLedgerInfoRetriever)
			for _, commitHeight := range configCommittingBlockNums {
				retrievedConfig, err := retriever.CollectionConfigAt(commitHeight, chaincodeName)
				assert.NoError(t, err)
				expectedConfig := sampleCollectionConfigPackage(ledgerid, commitHeight)
				assert.Equal(t, expectedConfig, retrievedConfig.CollectionConfig)
				assert.Equal(t, commitHeight, retrievedConfig.CommittingBlockNum)
			}
		}
	})

	t.Run("test-api-CollectionConfigAt-BoundaryCases()", func(t *testing.T) {
		retriever := mgr.GetRetriever("ledgerid1", dummyLedgerInfoRetriever)
		retrievedConfig, err := retriever.CollectionConfigAt(4, chaincodeName)
		assert.NoError(t, err)
		assert.Nil(t, retrievedConfig)

		retrievedConfig, err = retriever.CollectionConfigAt(5000, chaincodeName)
		typedErr, ok := err.(*ledger.ErrCollectionConfigNotYetAvailable)
		assert.True(t, ok)
		assert.Equal(t, maxBlockNumberInLedger, typedErr.MaxBlockNumCommitted)
	})
}

type testEnv struct {
	dbPath string
	mgr    Mgr
	t      *testing.T
}

func newTestEnv(t *testing.T, dbPath string, ccInfoProvider ledger.DeployedChaincodeInfoProvider) *testEnv {
	env := &testEnv{dbPath: dbPath, t: t}
	env.cleanup()
	env.mgr = newMgr(ccInfoProvider, dbPath)
	return env
}

func (env *testEnv) cleanup() {
	err := os.RemoveAll(env.dbPath)
	assert.NoError(env.t, err)
}

func sampleCollectionConfigPackage(collNamePart1 string, collNamePart2 uint64) *common.CollectionConfigPackage {
	collName := fmt.Sprintf("%s-%d", collNamePart1, collNamePart2)
	collConfig := &common.CollectionConfig{Payload: &common.CollectionConfig_StaticCollectionConfig{
		StaticCollectionConfig: &common.StaticCollectionConfig{Name: collName}}}
	collConfigPackage := &common.CollectionConfigPackage{Config: []*common.CollectionConfig{collConfig}}
	return collConfigPackage
}

func testutilEquipMockCCInfoProviderToReturnDesiredCollConfig(
	mockCCInfoProvider *mock.DeployedChaincodeInfoProvider,
	chaincodeName string,
	collConfigPackage *common.CollectionConfigPackage) {
	mockCCInfoProvider.UpdatedChaincodesReturns(
		[]*ledger.ChaincodeLifecycleInfo{
			{Name: chaincodeName},
		},
		nil,
	)
	mockCCInfoProvider.ChaincodeInfoReturns(
		&ledger.DeployedChaincodeInfo{Name: chaincodeName, CollectionConfigPkg: collConfigPackage},
		nil,
	)
}

type dummyLedgerInfoRetriever struct {
	info *common.BlockchainInfo
}

func (d *dummyLedgerInfoRetriever) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	return d.info, nil
}
