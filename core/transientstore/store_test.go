
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


package transientstore

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/core/transientdata")
	os.Exit(m.Run())
}

func TestPurgeIndexKeyCodingEncoding(t *testing.T) {
	assert := assert.New(t)
	blkHts := []uint64{0, 10, 20000}
	txids := []string{"txid", ""}
	uuids := []string{"uuid", ""}
	for _, blkHt := range blkHts {
		for _, txid := range txids {
			for _, uuid := range uuids {
				testCase := fmt.Sprintf("blkHt=%d,txid=%s,uuid=%s", blkHt, txid, uuid)
				t.Run(testCase, func(t *testing.T) {
					t.Logf("Running test case [%s]", testCase)
					purgeIndexKey := createCompositeKeyForPurgeIndexByHeight(blkHt, txid, uuid)
					txid1, uuid1, blkHt1 := splitCompositeKeyOfPurgeIndexByHeight(purgeIndexKey)
					assert.Equal(txid, txid1)
					assert.Equal(uuid, uuid1)
					assert.Equal(blkHt, blkHt1)
				})
			}
		}
	}
}

func TestRWSetKeyCodingEncoding(t *testing.T) {
	assert := assert.New(t)
	blkHts := []uint64{0, 10, 20000}
	txids := []string{"txid", ""}
	uuids := []string{"uuid", ""}
	for _, blkHt := range blkHts {
		for _, txid := range txids {
			for _, uuid := range uuids {
				testCase := fmt.Sprintf("blkHt=%d,txid=%s,uuid=%s", blkHt, txid, uuid)
				t.Run(testCase, func(t *testing.T) {
					t.Logf("Running test case [%s]", testCase)
					rwsetKey := createCompositeKeyForPvtRWSet(txid, uuid, blkHt)
					uuid1, blkHt1 := splitCompositeKeyOfPvtRWSet(rwsetKey)
					assert.Equal(uuid, uuid1)
					assert.Equal(blkHt, blkHt1)
				})
			}
		}
	}
}

func TestTransientStorePersistAndRetrieve(t *testing.T) {
	env := NewTestStoreEnv(t)
	assert := assert.New(t)
	txid := "txid-1"
	samplePvtRWSetWithConfig := samplePvtDataWithConfigInfo(t)

//为TXID-1创建私有模拟结果
	var endorsersResults []*EndorserPvtSimulationResultsWithConfig

//背书人1的结果
	endorser0SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser0SimulationResults)

//背书人2的结果
	endorser1SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser1SimulationResults)

//将模拟结果保存到存储中
	var err error
	for i := 0; i < len(endorsersResults); i++ {
		err = env.TestStore.PersistWithConfig(txid, endorsersResults[i].ReceivedAtBlockHeight,
			endorsersResults[i].PvtSimulationResultsWithConfig)
		assert.NoError(err)
	}

//从存储中检索TXID-1的模拟结果
	iter, err := env.TestStore.GetTxPvtRWSetByTxid(txid, nil)
	assert.NoError(err)

	var actualEndorsersResults []*EndorserPvtSimulationResultsWithConfig
	for {
		result, err := iter.NextWithConfig()
		assert.NoError(err)
		if result == nil {
			break
		}
		actualEndorsersResults = append(actualEndorsersResults, result)
	}
	iter.Close()
	sortResults(endorsersResults)
	sortResults(actualEndorsersResults)
	assert.Equal(endorsersResults, actualEndorsersResults)
}

func TestTransientStorePersistAndRetrieveBothOldAndNewProto(t *testing.T) {
	env := NewTestStoreEnv(t)
	assert := assert.New(t)
	txid := "txid-1"
	var receivedAtBlockHeight uint64 = 10
	var err error

//使用旧协议为TXID-1创建和持久化私有模拟结果
	samplePvtRWSet := samplePvtData(t)
	err = env.TestStore.Persist(txid, receivedAtBlockHeight, samplePvtRWSet)
	assert.NoError(err)

//使用新的TxID-1协议创建和持久化私有模拟结果
	samplePvtRWSetWithConfig := samplePvtDataWithConfigInfo(t)
	err = env.TestStore.PersistWithConfig(txid, receivedAtBlockHeight, samplePvtRWSetWithConfig)
	assert.NoError(err)

//构建预期结果
	var expectedEndorsersResults []*EndorserPvtSimulationResultsWithConfig

	pvtRWSetWithConfigInfo := &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: samplePvtRWSet,
	}

	endorser0SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          receivedAtBlockHeight,
		PvtSimulationResultsWithConfig: pvtRWSetWithConfigInfo,
	}
	expectedEndorsersResults = append(expectedEndorsersResults, endorser0SimulationResults)

	endorser1SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          receivedAtBlockHeight,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	expectedEndorsersResults = append(expectedEndorsersResults, endorser1SimulationResults)

//从存储中检索TXID-1的模拟结果
	iter, err := env.TestStore.GetTxPvtRWSetByTxid(txid, nil)
	assert.NoError(err)

	var actualEndorsersResults []*EndorserPvtSimulationResultsWithConfig
	for {
		result, err := iter.NextWithConfig()
		assert.NoError(err)
		if result == nil {
			break
		}
		actualEndorsersResults = append(actualEndorsersResults, result)
	}
	iter.Close()
	sortResults(expectedEndorsersResults)
	sortResults(actualEndorsersResults)
	assert.Equal(expectedEndorsersResults, actualEndorsersResults)
}

func TestTransientStorePurgeByTxids(t *testing.T) {
	env := NewTestStoreEnv(t)
	assert := assert.New(t)

	var txids []string
	var endorsersResults []*EndorserPvtSimulationResultsWithConfig

	samplePvtRWSetWithConfig := samplePvtDataWithConfigInfo(t)

//为txid-1创建两个私有写入集条目
	txids = append(txids, "txid-1")
	endorser0SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser0SimulationResults)

	txids = append(txids, "txid-1")
	endorser1SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser1SimulationResults)

//为txid-2创建一个私有写入集条目
	txids = append(txids, "txid-2")
	endorser2SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser2SimulationResults)

//为txid-3创建三个私有写入集条目
	txids = append(txids, "txid-3")
	endorser3SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser3SimulationResults)

	txids = append(txids, "txid-3")
	endorser4SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser4SimulationResults)

	txids = append(txids, "txid-3")
	endorser5SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          13,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser5SimulationResults)

	var err error
	for i := 0; i < len(txids); i++ {
		err = env.TestStore.PersistWithConfig(txids[i], endorsersResults[i].ReceivedAtBlockHeight,
			endorsersResults[i].PvtSimulationResultsWithConfig)
		assert.NoError(err)
	}

//从存储库中检索TXID-2仿真结果
	iter, err := env.TestStore.GetTxPvtRWSetByTxid("txid-2", nil)
	assert.NoError(err)

//TXID-2的预期结果
	var expectedEndorsersResults []*EndorserPvtSimulationResultsWithConfig
	expectedEndorsersResults = append(expectedEndorsersResults, endorser2SimulationResults)

//Check whether actual results and expected results are same
	var actualEndorsersResults []*EndorserPvtSimulationResultsWithConfig
	for true {
		result, err := iter.NextWithConfig()
		assert.NoError(err)
		if result == nil {
			break
		}
		actualEndorsersResults = append(actualEndorsersResults, result)
	}
	iter.Close()

//请注意，实际值和预期值的顺序取决于UUID。因此，我们正在排序
//预期值和实际值。
	sortResults(expectedEndorsersResults)
	sortResults(actualEndorsersResults)

	assert.Equal(len(expectedEndorsersResults), len(actualEndorsersResults))
	for i, expected := range expectedEndorsersResults {
		assert.Equal(expected.ReceivedAtBlockHeight, actualEndorsersResults[i].ReceivedAtBlockHeight)
		assert.True(proto.Equal(expected.PvtSimulationResultsWithConfig, actualEndorsersResults[i].PvtSimulationResultsWithConfig))
	}

//Remove all private write set of txid-2 and txid-3
	toRemoveTxids := []string{"txid-2", "txid-3"}
	err = env.TestStore.PurgeByTxids(toRemoveTxids)
	assert.NoError(err)

	for _, txid := range toRemoveTxids {

//检查TXID-2的私有写入集是否被删除
		var expectedEndorsersResults *EndorserPvtSimulationResultsWithConfig
		expectedEndorsersResults = nil
		iter, err = env.TestStore.GetTxPvtRWSetByTxid(txid, nil)
		assert.NoError(err)
//应返回零，零
		result, err := iter.NextWithConfig()
		assert.NoError(err)
		assert.Equal(expectedEndorsersResults, result)
	}

//从存储中检索TXID-1的模拟结果
	iter, err = env.TestStore.GetTxPvtRWSetByTxid("txid-1", nil)
	assert.NoError(err)

//TXID-1的预期结果
	expectedEndorsersResults = nil
	expectedEndorsersResults = append(expectedEndorsersResults, endorser0SimulationResults)
	expectedEndorsersResults = append(expectedEndorsersResults, endorser1SimulationResults)

//检查实际结果与预期结果是否一致
	actualEndorsersResults = nil
	for true {
		result, err := iter.NextWithConfig()
		assert.NoError(err)
		if result == nil {
			break
		}
		actualEndorsersResults = append(actualEndorsersResults, result)
	}
	iter.Close()

//请注意，实际值和预期值的顺序取决于UUID。因此，我们正在排序
//预期值和实际值。
	sortResults(expectedEndorsersResults)
	sortResults(actualEndorsersResults)

	assert.Equal(len(expectedEndorsersResults), len(actualEndorsersResults))
	for i, expected := range expectedEndorsersResults {
		assert.Equal(expected.ReceivedAtBlockHeight, actualEndorsersResults[i].ReceivedAtBlockHeight)
		assert.True(proto.Equal(expected.PvtSimulationResultsWithConfig, actualEndorsersResults[i].PvtSimulationResultsWithConfig))
	}

	toRemoveTxids = []string{"txid-1"}
	err = env.TestStore.PurgeByTxids(toRemoveTxids)
	assert.NoError(err)

	for _, txid := range toRemoveTxids {

//检查TXID-1的私有写入集是否被删除
		var expectedEndorsersResults *EndorserPvtSimulationResultsWithConfig
		expectedEndorsersResults = nil
		iter, err = env.TestStore.GetTxPvtRWSetByTxid(txid, nil)
		assert.NoError(err)
//应返回零，零
		result, err := iter.NextWithConfig()
		assert.NoError(err)
		assert.Equal(expectedEndorsersResults, result)
	}

//存储区中不应该有条目
	_, err = env.TestStore.GetMinTransientBlkHt()
	assert.Equal(err, ErrStoreEmpty)
}

func TestTransientStorePurgeByHeight(t *testing.T) {
	env := NewTestStoreEnv(t)
	assert := assert.New(t)

	txid := "txid-1"
	samplePvtRWSetWithConfig := samplePvtDataWithConfigInfo(t)

//为TXID-1创建私有模拟结果
	var endorsersResults []*EndorserPvtSimulationResultsWithConfig

//背书人1的结果
	endorser0SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser0SimulationResults)

//背书人2的结果
	endorser1SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser1SimulationResults)

//背书人3的结果
	endorser2SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser2SimulationResults)

//背书人3的结果
	endorser3SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser3SimulationResults)

//背书人3的结果
	endorser4SimulationResults := &EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          13,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser4SimulationResults)

//将模拟结果保存到存储中
	var err error
	for i := 0; i < 5; i++ {
		err = env.TestStore.PersistWithConfig(txid, endorsersResults[i].ReceivedAtBlockHeight,
			endorsersResults[i].PvtSimulationResultsWithConfig)
		assert.NoError(err)
	}

//块高度大于或等于12时生成的保留结果
	minTransientBlkHtToRetain := uint64(12)
	err = env.TestStore.PurgeByHeight(minTransientBlkHtToRetain)
	assert.NoError(err)

//从存储中检索TXID-1的模拟结果
	iter, err := env.TestStore.GetTxPvtRWSetByTxid(txid, nil)
	assert.NoError(err)

//TXID-1的预期结果
	var expectedEndorsersResults []*EndorserPvtSimulationResultsWithConfig
expectedEndorsersResults = append(expectedEndorsersResults, endorser2SimulationResults) //高度12背书
expectedEndorsersResults = append(expectedEndorsersResults, endorser3SimulationResults) //高度12背书
expectedEndorsersResults = append(expectedEndorsersResults, endorser4SimulationResults) //高度13背书

//检查实际结果与预期结果是否一致
	var actualEndorsersResults []*EndorserPvtSimulationResultsWithConfig
	for true {
		result, err := iter.NextWithConfig()
		assert.NoError(err)
		if result == nil {
			break
		}
		actualEndorsersResults = append(actualEndorsersResults, result)
	}
	iter.Close()

//请注意，实际值和预期值的顺序取决于UUID。因此，我们正在排序
//预期值和实际值。
	sortResults(expectedEndorsersResults)
	sortResults(actualEndorsersResults)

	assert.Equal(len(expectedEndorsersResults), len(actualEndorsersResults))
	for i, expected := range expectedEndorsersResults {
		assert.Equal(expected.ReceivedAtBlockHeight, actualEndorsersResults[i].ReceivedAtBlockHeight)
		assert.True(proto.Equal(expected.PvtSimulationResultsWithConfig, actualEndorsersResults[i].PvtSimulationResultsWithConfig))
	}

//获取临时存储中剩余的最小块高度
	var actualMinTransientBlkHt uint64
	actualMinTransientBlkHt, err = env.TestStore.GetMinTransientBlkHt()
	assert.NoError(err)
	assert.Equal(minTransientBlkHtToRetain, actualMinTransientBlkHt)

//保持块高度大于或等于15的结果
	minTransientBlkHtToRetain = uint64(15)
	err = env.TestStore.PurgeByHeight(minTransientBlkHtToRetain)
	assert.NoError(err)

//存储区中不应该有条目
	actualMinTransientBlkHt, err = env.TestStore.GetMinTransientBlkHt()
	assert.Equal(err, ErrStoreEmpty)

//保持块高度大于或等于15的结果
	minTransientBlkHtToRetain = uint64(15)
	err = env.TestStore.PurgeByHeight(minTransientBlkHtToRetain)
//Should not return any error
	assert.NoError(err)

	env.Cleanup()
}

func TestTransientStoreRetrievalWithFilter(t *testing.T) {
	env := NewTestStoreEnv(t)
	store := env.TestStore

	samplePvtSimResWithConfig := samplePvtDataWithConfigInfo(t)

	testTxid := "testTxid"
	numEntries := 5
	for i := 0; i < numEntries; i++ {
		store.PersistWithConfig(testTxid, uint64(i), samplePvtSimResWithConfig)
	}

	filter := ledger.NewPvtNsCollFilter()
	filter.Add("ns-1", "coll-1")
	filter.Add("ns-2", "coll-2")

	itr, err := store.GetTxPvtRWSetByTxid(testTxid, filter)
	assert.NoError(t, err)

	var actualRes []*EndorserPvtSimulationResultsWithConfig
	for {
		res, err := itr.NextWithConfig()
		if res == nil || err != nil {
			assert.NoError(t, err)
			break
		}
		actualRes = append(actualRes, res)
	}

//手动准备修剪过的pvtrwset-仅保留“ns-1/coll-1”和“ns-2/coll-2”
	expectedSimulationRes := samplePvtSimResWithConfig
	expectedSimulationRes.GetPvtRwset().NsPvtRwset[0].CollectionPvtRwset = expectedSimulationRes.GetPvtRwset().NsPvtRwset[0].CollectionPvtRwset[0:1]
	expectedSimulationRes.GetPvtRwset().NsPvtRwset[1].CollectionPvtRwset = expectedSimulationRes.GetPvtRwset().NsPvtRwset[1].CollectionPvtRwset[1:]
	expectedSimulationRes.CollectionConfigs, err = trimPvtCollectionConfigs(expectedSimulationRes.CollectionConfigs, filter)
	assert.NoError(t, err)
	for ns, colName := range map[string]string{"ns-1": "coll-1", "ns-2": "coll-2"} {
		config := expectedSimulationRes.CollectionConfigs[ns]
		assert.NotNil(t, config)
		ns1Config := config.Config
		assert.Equal(t, len(ns1Config), 1)
		ns1ColConfig := ns1Config[0].GetStaticCollectionConfig()
		assert.NotNil(t, ns1ColConfig.Name, colName)
	}

	var expectedRes []*EndorserPvtSimulationResultsWithConfig
	for i := 0; i < numEntries; i++ {
		expectedRes = append(expectedRes, &EndorserPvtSimulationResultsWithConfig{uint64(i), expectedSimulationRes})
	}

//请注意，实际值和预期值的顺序取决于UUID。因此，我们正在排序
//预期值和实际值。
	sortResults(expectedRes)
	sortResults(actualRes)
	assert.Equal(t, len(expectedRes), len(actualRes))
	for i, expected := range expectedRes {
		assert.Equal(t, expected.ReceivedAtBlockHeight, actualRes[i].ReceivedAtBlockHeight)
		assert.True(t, proto.Equal(expected.PvtSimulationResultsWithConfig, actualRes[i].PvtSimulationResultsWithConfig))
	}

}

func sortResults(res []*EndorserPvtSimulationResultsWithConfig) {
//结果按块高度接收的升序排序。当块
//高度相同，我们通过比较私有写入集的散列进行排序。
	var sortCondition = func(i, j int) bool {
		if res[i].ReceivedAtBlockHeight == res[j].ReceivedAtBlockHeight {
			res_i, _ := proto.Marshal(res[i].PvtSimulationResultsWithConfig)
			res_j, _ := proto.Marshal(res[j].PvtSimulationResultsWithConfig)
//如果哈希值相同，任何顺序都可以工作。
			return string(util.ComputeHash(res_i)) < string(util.ComputeHash(res_j))
		}
		return res[i].ReceivedAtBlockHeight < res[j].ReceivedAtBlockHeight
	}
	sort.SliceStable(res, sortCondition)
}

func samplePvtData(t *testing.T) *rwset.TxPvtReadWriteSet {
	pvtWriteSet := &rwset.TxPvtReadWriteSet{DataModel: rwset.TxReadWriteSet_KV}
	pvtWriteSet.NsPvtRwset = []*rwset.NsPvtReadWriteSet{
		{
			Namespace: "ns-1",
			CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
				{
					CollectionName: "coll-1",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll1"),
				},
				{
					CollectionName: "coll-2",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll2"),
				},
			},
		},

		{
			Namespace: "ns-2",
			CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
				{
					CollectionName: "coll-1",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns2-coll1"),
				},
				{
					CollectionName: "coll-2",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns2-coll2"),
				},
			},
		},
	}
	return pvtWriteSet
}

func samplePvtDataWithConfigInfo(t *testing.T) *transientstore.TxPvtReadWriteSetWithConfigInfo {
	pvtWriteSet := samplePvtData(t)
	pvtRWSetWithConfigInfo := &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: pvtWriteSet,
		CollectionConfigs: map[string]*common.CollectionConfigPackage{
			"ns-1": {
				Config: []*common.CollectionConfig{
					sampleCollectionConfigPackage("coll-1"),
					sampleCollectionConfigPackage("coll-2"),
				},
			},
			"ns-2": {
				Config: []*common.CollectionConfig{
					sampleCollectionConfigPackage("coll-1"),
					sampleCollectionConfigPackage("coll-2"),
				},
			},
		},
	}
	return pvtRWSetWithConfigInfo
}

func createCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32,
) *common.CollectionConfig {
	signaturePolicy := &common.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}
	accessPolicy := &common.CollectionPolicyConfig{
		Payload: signaturePolicy,
	}

	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collectionName,
				MemberOrgsPolicy:  accessPolicy,
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maximumPeerCount,
			},
		},
	}
}

func sampleCollectionConfigPackage(colName string) *common.CollectionConfig {
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)

	var requiredPeerCount, maximumPeerCount int32
	requiredPeerCount = 1
	maximumPeerCount = 2

	return createCollectionConfig(colName, policyEnvelope, requiredPeerCount, maximumPeerCount)
}
