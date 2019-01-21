
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


package kvledger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/ledger/blkstorage/fsblkstorage"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/common/util"
	lgr "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLedgerProvider(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	numLedgers := 10
	provider := testutilNewProvider(t)
	existingLedgerIDs, err := provider.List()
	assert.NoError(t, err)
	assert.Len(t, existingLedgerIDs, 0)
	genesisBlocks := make([]*common.Block, numLedgers)
	for i := 0; i < numLedgers; i++ {
		genesisBlock, _ := configtxtest.MakeGenesisBlock(constructTestLedgerID(i))
		genesisBlocks[i] = genesisBlock
		provider.Create(genesisBlock)
	}
	existingLedgerIDs, err = provider.List()
	assert.NoError(t, err)
	assert.Len(t, existingLedgerIDs, numLedgers)

	provider.Close()

	provider = testutilNewProvider(t)
	defer provider.Close()
	ledgerIds, _ := provider.List()
	assert.Len(t, ledgerIds, numLedgers)
	t.Logf("ledgerIDs=%#v", ledgerIds)
	for i := 0; i < numLedgers; i++ {
		assert.Equal(t, constructTestLedgerID(i), ledgerIds[i])
	}
	for i := 0; i < numLedgers; i++ {
		ledgerid := constructTestLedgerID(i)
		status, _ := provider.Exists(ledgerid)
		assert.True(t, status)
		ledger, err := provider.Open(ledgerid)
		assert.NoError(t, err)
		bcInfo, err := ledger.GetBlockchainInfo()
		ledger.Close()
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), bcInfo.Height)

//检查Genesis块是否保留在提供程序的数据库中
		s := provider.(*Provider).idStore
		gbBytesInProviderStore, err := s.db.Get(s.encodeLedgerKey(ledgerid))
		assert.NoError(t, err)
		gb := &common.Block{}
		assert.NoError(t, proto.Unmarshal(gbBytesInProviderStore, gb))
		assert.True(t, proto.Equal(gb, genesisBlocks[i]), "proto messages are not equal")
	}
	gb, _ := configtxtest.MakeGenesisBlock(constructTestLedgerID(2))
	_, err = provider.Create(gb)
	assert.Equal(t, ErrLedgerIDExists, err)

	status, err := provider.Exists(constructTestLedgerID(numLedgers))
	assert.NoError(t, err, "Failed to check for ledger existence")
	assert.Equal(t, status, false)

	_, err = provider.Open(constructTestLedgerID(numLedgers))
	assert.Equal(t, ErrNonExistingLedgerID, err)
}

func TestRecovery(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	provider := testutilNewProvider(t)

//现在创建Genesis块
	genesisBlock, _ := configtxtest.MakeGenesisBlock(constructTestLedgerID(1))
	ledger, err := provider.(*Provider).openInternal(constructTestLedgerID(1))
	ledger.CommitWithPvtData(&lgr.BlockAndPvtData{Block: genesisBlock})
	ledger.Close()

//案例1：假设发生了碰撞，强制将Unconstruction标志设置为Simulate
//创建LedgerID时出现故障，即写块但未取消标记。
	provider.(*Provider).idStore.setUnderConstructionFlag(constructTestLedgerID(1))
	provider.Close()

//构造新的提供程序以调用恢复
	provider = testutilNewProvider(t)
//验证下面的代码标记并打开分类帐
	flag, err := provider.(*Provider).idStore.getUnderConstructionFlag()
	assert.NoError(t, err, "Failed to read the underconstruction flag")
	assert.Equal(t, "", flag)
	ledger, err = provider.Open(constructTestLedgerID(1))
	assert.NoError(t, err, "Failed to open the ledger")
	ledger.Close()

//案例0：假设在提交Ledger 2的Genesis块之前发生崩溃
//打开ID存储（链ID/LedgerID的库存）
	provider.(*Provider).idStore.setUnderConstructionFlag(constructTestLedgerID(2))
	provider.Close()

//构造新的提供程序以调用恢复
	provider = testutilNewProvider(t)
	assert.NoError(t, err, "Provider failed to recover an underConstructionLedger")
	flag, err = provider.(*Provider).idStore.getUnderConstructionFlag()
	assert.NoError(t, err, "Failed to read the underconstruction flag")
	assert.Equal(t, "", flag)

}

func TestMultipleLedgerBasicRW(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	numLedgers := 10
	provider := testutilNewProvider(t)
	ledgers := make([]lgr.PeerLedger, numLedgers)
	for i := 0; i < numLedgers; i++ {
		bg, gb := testutil.NewBlockGenerator(t, constructTestLedgerID(i), false)
		l, err := provider.Create(gb)
		assert.NoError(t, err)
		ledgers[i] = l
		txid := util.GenerateUUID()
		s, _ := l.NewTxSimulator(txid)
		err = s.SetState("ns", "testKey", []byte(fmt.Sprintf("testValue_%d", i)))
		s.Done()
		assert.NoError(t, err)
		res, err := s.GetTxSimulationResults()
		assert.NoError(t, err)
		pubSimBytes, _ := res.GetPubSimulationBytes()
		b := bg.NextBlock([][]byte{pubSimBytes})
		err = l.CommitWithPvtData(&lgr.BlockAndPvtData{Block: b})
		l.Close()
		assert.NoError(t, err)
	}

	provider.Close()

	provider = testutilNewProvider(t)
	defer provider.Close()
	ledgers = make([]lgr.PeerLedger, numLedgers)
	for i := 0; i < numLedgers; i++ {
		l, err := provider.Open(constructTestLedgerID(i))
		assert.NoError(t, err)
		ledgers[i] = l
	}

	for i, l := range ledgers {
		q, _ := l.NewQueryExecutor()
		val, err := q.GetState("ns", "testKey")
		q.Done()
		assert.NoError(t, err)
		assert.Equal(t, []byte(fmt.Sprintf("testValue_%d", i)), val)
		l.Close()
	}
}

func TestLedgerBackup(t *testing.T) {
	ledgerid := "TestLedger"
	originalPath := "/tmp/fabric/ledgertests/kvledger1"
	restorePath := "/tmp/fabric/ledgertests/kvledger2"
	viper.Set("ledger.history.enableHistoryDatabase", true)

//在原始环境中创建和填充分类帐
	env := createTestEnv(t, originalPath)
	provider := testutilNewProvider(t)
	bg, gb := testutil.NewBlockGenerator(t, ledgerid, false)
	gbHash := gb.Header.Hash()
	ledger, _ := provider.Create(gb)

	txid := util.GenerateUUID()
	simulator, _ := ledger.NewTxSimulator(txid)
	simulator.SetState("ns1", "key1", []byte("value1"))
	simulator.SetState("ns1", "key2", []byte("value2"))
	simulator.SetState("ns1", "key3", []byte("value3"))
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimBytes, _ := simRes.GetPubSimulationBytes()
	block1 := bg.NextBlock([][]byte{pubSimBytes})
	ledger.CommitWithPvtData(&lgr.BlockAndPvtData{Block: block1})

	txid = util.GenerateUUID()
	simulator, _ = ledger.NewTxSimulator(txid)
	simulator.SetState("ns1", "key1", []byte("value4"))
	simulator.SetState("ns1", "key2", []byte("value5"))
	simulator.SetState("ns1", "key3", []byte("value6"))
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	pubSimBytes, _ = simRes.GetPubSimulationBytes()
	block2 := bg.NextBlock([][]byte{pubSimBytes})
	ledger.CommitWithPvtData(&lgr.BlockAndPvtData{Block: block2})

	ledger.Close()
	provider.Close()

//创建还原环境
	env = createTestEnv(t, restorePath)

//删除statedb、historydb和block索引（它们应该在打开现有分类帐时自动创建）
//并将原始路径重命名为restorepath
	assert.NoError(t, os.RemoveAll(ledgerconfig.GetStateLevelDBPath()))
	assert.NoError(t, os.RemoveAll(ledgerconfig.GetHistoryLevelDBPath()))
	assert.NoError(t, os.RemoveAll(filepath.Join(ledgerconfig.GetBlockStorePath(), fsblkstorage.IndexDir)))
	assert.NoError(t, os.Rename(originalPath, restorePath))
	defer env.cleanup()

//从还原环境实例化分类帐，它的行为应该与在原始环境中完全相同。
	provider = testutilNewProvider(t)
	defer provider.Close()

	_, err := provider.Create(gb)
	assert.Equal(t, ErrLedgerIDExists, err)

	ledger, _ = provider.Open(ledgerid)
	defer ledger.Close()

	block1Hash := block1.Header.Hash()
	block2Hash := block2.Header.Hash()
	bcInfo, _ := ledger.GetBlockchainInfo()
	assert.Equal(t, &common.BlockchainInfo{
		Height: 3, CurrentBlockHash: block2Hash, PreviousBlockHash: block1Hash,
	}, bcInfo)

	b0, _ := ledger.GetBlockByHash(gbHash)
	assert.True(t, proto.Equal(b0, gb), "proto messages are not equal")

	b1, _ := ledger.GetBlockByHash(block1Hash)
	assert.True(t, proto.Equal(b1, block1), "proto messages are not equal")

	b2, _ := ledger.GetBlockByHash(block2Hash)
	assert.True(t, proto.Equal(b2, block2), "proto messages are not equal")

	b0, _ = ledger.GetBlockByNumber(0)
	assert.True(t, proto.Equal(b0, gb), "proto messages are not equal")

	b1, _ = ledger.GetBlockByNumber(1)
	assert.True(t, proto.Equal(b1, block1), "proto messages are not equal")

	b2, _ = ledger.GetBlockByNumber(2)
	assert.True(t, proto.Equal(b2, block2), "proto messages are not equal")

//从第2个块获取tran id，然后使用它测试getTransactionByID（）。
	txEnvBytes2 := block1.Data.Data[0]
	txEnv2, err := putils.GetEnvelopeFromBlock(txEnvBytes2)
	assert.NoError(t, err, "Error upon GetEnvelopeFromBlock")
	payload2, err := putils.GetPayload(txEnv2)
	assert.NoError(t, err, "Error upon GetPayload")
	chdr, err := putils.UnmarshalChannelHeader(payload2.Header.ChannelHeader)
	assert.NoError(t, err, "Error upon GetChannelHeaderFromBytes")
	txID2 := chdr.TxId
	processedTran2, err := ledger.GetTransactionByID(txID2)
	assert.NoError(t, err, "Error upon GetTransactionByID")
//从检索的processedTransaction获取Tran信封
	retrievedTxEnv2 := processedTran2.TransactionEnvelope
	assert.Equal(t, txEnv2, retrievedTxEnv2)

	qe, _ := ledger.NewQueryExecutor()
	value1, _ := qe.GetState("ns1", "key1")
	assert.Equal(t, []byte("value4"), value1)

	hqe, err := ledger.NewHistoryQueryExecutor()
	assert.NoError(t, err)
	itr, err := hqe.GetHistoryForKey("ns1", "key1")
	assert.NoError(t, err)
	defer itr.Close()

	result1, err := itr.Next()
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), result1.(*queryresult.KeyModification).Value)
	result2, err := itr.Next()
	assert.NoError(t, err)
	assert.Equal(t, []byte("value4"), result2.(*queryresult.KeyModification).Value)
}

func constructTestLedgerID(i int) string {
	return fmt.Sprintf("ledger_%06d", i)
}

func testutilNewProvider(t *testing.T) lgr.PeerLedgerProvider {
	provider, err := NewProvider()
	assert.NoError(t, err)
	provider.Initialize(&lgr.Initializer{
		DeployedChaincodeInfoProvider: &mock.DeployedChaincodeInfoProvider{},
		MetricsProvider:               &disabled.Provider{},
	})
	return provider
}

func testutilNewProviderWithCollectionConfig(t *testing.T, namespace string, btlConfigs map[string]uint64) lgr.PeerLedgerProvider {
	provider := testutilNewProvider(t)
	mockCCInfoProvider := provider.(*Provider).initializer.DeployedChaincodeInfoProvider.(*mock.DeployedChaincodeInfoProvider)
	collMap := map[string]*common.StaticCollectionConfig{}
	var conf []*common.CollectionConfig
	for collName, btl := range btlConfigs {
		staticConf := &common.StaticCollectionConfig{Name: collName, BlockToLive: btl}
		collMap[collName] = staticConf
		collectionConf := &common.CollectionConfig{}
		collectionConf.Payload = &common.CollectionConfig_StaticCollectionConfig{StaticCollectionConfig: staticConf}
		conf = append(conf, collectionConf)
	}
	collectionConfPkg := &common.CollectionConfigPackage{Config: conf}

	mockCCInfoProvider.ChaincodeInfoStub = func(ccName string, qe lgr.SimpleQueryExecutor) (*lgr.DeployedChaincodeInfo, error) {
		if ccName == namespace {
			return &lgr.DeployedChaincodeInfo{
				Name: namespace, CollectionConfigPkg: collectionConfPkg}, nil
		}
		return nil, nil
	}

	mockCCInfoProvider.CollectionInfoStub = func(ccName, collName string, qe lgr.SimpleQueryExecutor) (*common.StaticCollectionConfig, error) {
		if ccName == namespace {
			return collMap[collName], nil
		}
		return nil, nil
	}
	return provider
}
