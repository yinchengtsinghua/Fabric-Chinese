
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


package pvtstatepurgemgmt

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("pvtstatepurgemgmt,privacyenabledstate=debug")
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/pvtstatepurgemgmt")
	os.Exit(m.Run())
}

func TestPurgeMgr(t *testing.T) {
	dbEnvs := []privacyenabledstate.TestEnv{
		&privacyenabledstate.LevelDBCommonStorageTestEnv{},
		&privacyenabledstate.CouchDBCommonStorageTestEnv{},
	}
	for _, dbEnv := range dbEnvs {
		t.Run(dbEnv.GetName(), func(t *testing.T) { testPurgeMgr(t, dbEnv) })
	}
}

func testPurgeMgr(t *testing.T, dbEnv privacyenabledstate.TestEnv) {
	ledgerid := "testledger-perge-mgr"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns1", "coll1"}: 1,
			{"ns1", "coll2"}: 2,
			{"ns2", "coll3"}: 4,
			{"ns2", "coll4"}: 4,
		},
	)

	testHelper := &testHelper{}
	testHelper.init(t, ledgerid, btlPolicy, dbEnv)
	defer testHelper.cleanup()

	block1Updates := privacyenabledstate.NewUpdateBatch()
	block1Updates.PubUpdates.Put("ns1", "pubkey1", []byte("pubvalue1-1"), version.NewHeight(1, 1))
	putPvtAndHashUpdates(t, block1Updates, "ns1", "coll1", "pvtkey1", []byte("pvtvalue1-1"), version.NewHeight(1, 1))
	putPvtAndHashUpdates(t, block1Updates, "ns1", "coll2", "pvtkey2", []byte("pvtvalue2-1"), version.NewHeight(1, 1))
	putPvtAndHashUpdates(t, block1Updates, "ns2", "coll3", "pvtkey3", []byte("pvtvalue3-1"), version.NewHeight(1, 1))
	putPvtAndHashUpdates(t, block1Updates, "ns2", "coll4", "pvtkey4", []byte("pvtvalue4-1"), version.NewHeight(1, 1))
	testHelper.commitUpdatesForTesting(1, block1Updates)
	testHelper.checkPvtdataExists("ns1", "coll1", "pvtkey1", []byte("pvtvalue1-1"))
	testHelper.checkPvtdataExists("ns1", "coll2", "pvtkey2", []byte("pvtvalue2-1"))
	testHelper.checkPvtdataExists("ns2", "coll3", "pvtkey3", []byte("pvtvalue3-1"))
	testHelper.checkPvtdataExists("ns2", "coll4", "pvtkey4", []byte("pvtvalue4-1"))

	block2Updates := privacyenabledstate.NewUpdateBatch()
	putPvtAndHashUpdates(t, block2Updates, "ns1", "coll2", "pvtkey2", []byte("pvtvalue2-2"), version.NewHeight(2, 1))
	deletePvtAndHashUpdates(t, block2Updates, "ns2", "coll4", "pvtkey4", version.NewHeight(2, 1))
	testHelper.commitUpdatesForTesting(2, block2Updates)
	testHelper.checkPvtdataExists("ns1", "coll1", "pvtkey1", []byte("pvtvalue1-1"))
	testHelper.checkPvtdataExists("ns1", "coll2", "pvtkey2", []byte("pvtvalue2-2"))
	testHelper.checkPvtdataExists("ns2", "coll3", "pvtkey3", []byte("pvtvalue3-1"))
	testHelper.checkPvtdataDoesNotExist("ns1", "coll4", "pvtkey4")

	noPvtdataUpdates := privacyenabledstate.NewUpdateBatch()
	testHelper.commitUpdatesForTesting(3, noPvtdataUpdates)
	testHelper.checkPvtdataDoesNotExist("ns1", "coll1", "pvtkey1")
	testHelper.checkPvtdataExists("ns1", "coll2", "pvtkey2", []byte("pvtvalue2-2"))
	testHelper.checkPvtdataExists("ns2", "coll3", "pvtkey3", []byte("pvtvalue3-1"))
	testHelper.checkPvtdataDoesNotExist("ns1", "coll4", "pvtkey4")

	testHelper.commitUpdatesForTesting(4, noPvtdataUpdates)
	testHelper.checkPvtdataDoesNotExist("ns1", "coll1", "pvtkey1")
	testHelper.checkPvtdataExists("ns1", "coll2", "pvtkey2", []byte("pvtvalue2-2"))
	testHelper.checkPvtdataExists("ns2", "coll3", "pvtkey3", []byte("pvtvalue3-1"))
	testHelper.checkPvtdataDoesNotExist("ns1", "coll4", "pvtkey4")

	testHelper.commitUpdatesForTesting(5, noPvtdataUpdates)
	testHelper.checkPvtdataDoesNotExist("ns1", "coll1", "pvtkey1")
	testHelper.checkPvtdataDoesNotExist("ns1", "coll2", "pvtkey2")
	testHelper.checkPvtdataExists("ns2", "coll3", "pvtkey3", []byte("pvtvalue3-1"))
	testHelper.checkPvtdataDoesNotExist("ns1", "coll4", "pvtkey4")

	testHelper.commitUpdatesForTesting(6, noPvtdataUpdates)
	testHelper.checkPvtdataDoesNotExist("ns1", "coll1", "pvtkey1")
	testHelper.checkPvtdataDoesNotExist("ns1", "coll2", "pvtkey2")
	testHelper.checkPvtdataDoesNotExist("ns2", "coll3", "pvtkey3")
	testHelper.checkPvtdataDoesNotExist("ns1", "coll4", "pvtkey4")
}

func TestPurgeMgrForCommittingPvtDataOfOldBlocks(t *testing.T) {
	dbEnvs := []privacyenabledstate.TestEnv{
		&privacyenabledstate.LevelDBCommonStorageTestEnv{},
		&privacyenabledstate.CouchDBCommonStorageTestEnv{},
	}
	for _, dbEnv := range dbEnvs {
		t.Run(dbEnv.GetName(), func(t *testing.T) { testPurgeMgrForCommittingPvtDataOfOldBlocks(t, dbEnv) })
	}
}

func testPurgeMgrForCommittingPvtDataOfOldBlocks(t *testing.T, dbEnv privacyenabledstate.TestEnv) {
	ledgerid := "testledger-purge-mgr-pvtdata-oldblocks"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns1", "coll1"}: 1,
		},
	)

	testHelper := &testHelper{}
	testHelper.init(t, ledgerid, btlPolicy, dbEnv)
	defer testHelper.cleanup()

//提交块1
	block1Updates := privacyenabledstate.NewUpdateBatch()
//缺少pvt数据pvtkey1，但存在pvtkey2。
//提交块3时，pvtkey1和pvtkey2都将过期并清除
	putHashUpdates(block1Updates, "ns1", "coll1", "pvtkey1", []byte("pvtvalue1-1"), version.NewHeight(1, 1))
	putPvtAndHashUpdates(t, block1Updates, "ns1", "coll1", "pvtkey2", []byte("pvtvalue1-2"), version.NewHeight(1, 1))
	testHelper.commitUpdatesForTesting(1, block1Updates)

//pvtkey1不应存在，但pvtkey2应存在
	testHelper.checkOnlyPvtKeyDoesNotExist("ns1", "coll1", "pvtkey1")
	testHelper.checkPvtdataExists("ns1", "coll1", "pvtkey2", []byte("pvtvalue1-2"))

//提交块2
	block2Updates := privacyenabledstate.NewUpdateBatch()
	testHelper.commitUpdatesForTesting(2, block2Updates)

//通过提交丢失的数据来提交pvtkey1，并将其添加到topurge列表中。
//提交块3时应删除
	block1PvtData := privacyenabledstate.NewUpdateBatch()
	putPvtUpdates(block1PvtData, "ns1", "coll1", "pvtkey1", []byte("pvtvalue1-1"), version.NewHeight(1, 1))
	testHelper.commitPvtDataOfOldBlocksForTesting(block1PvtData)

//pvtkey1和pvtkey1都应该存在
	testHelper.checkPvtdataExists("ns1", "coll1", "pvtkey1", []byte("pvtvalue1-1"))
	testHelper.checkPvtdataExists("ns1", "coll1", "pvtkey2", []byte("pvtvalue1-2"))

//提交块3
	block3Updates := privacyenabledstate.NewUpdateBatch()
	testHelper.commitUpdatesForTesting(3, block3Updates)

//pvtkey1和pvtkey1都不应该存在
	testHelper.checkPvtdataDoesNotExist("ns1", "coll1", "pvtkey1")
	testHelper.checkPvtdataDoesNotExist("ns1", "coll1", "pvtkey2")
}

func TestKeyUpdateBeforeExpiryBlock(t *testing.T) {
	dbEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	ledgerid := "testledger-perge-mgr"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
{"ns", "coll"}: 1, //失效块=提交块+2
		},
	)
	helper := &testHelper{}
	helper.init(t, ledgerid, btlPolicy, dbEnv)
	defer helper.cleanup()

//block-1更新：仅更新pvt密钥的哈希
	block1Updates := privacyenabledstate.NewUpdateBatch()
	putHashUpdates(block1Updates, "ns", "coll", "pvtkey", []byte("pvtvalue-1"), version.NewHeight(1, 1))
	helper.commitUpdatesForTesting(1, block1Updates)
	expInfo, _ := helper.purgeMgr.(*purgeMgr).expKeeper.retrieve(3)
	assert.Len(t, expInfo, 1)

//块2更新：同时更新哈希和pvt数据
	block2Updates := privacyenabledstate.NewUpdateBatch()
	putPvtAndHashUpdates(t, block2Updates, "ns", "coll", "pvtkey", []byte("pvtvalue-2"), version.NewHeight(2, 1))
	helper.commitUpdatesForTesting(2, block2Updates)
	helper.checkExpiryEntryExistsForBlockNum(3, 1)
	helper.checkExpiryEntryExistsForBlockNum(4, 1)

//Block-3更新：无更新
	noPvtdataUpdates := privacyenabledstate.NewUpdateBatch()
	helper.commitUpdatesForTesting(3, noPvtdataUpdates)
	helper.checkPvtdataExists("ns", "coll", "pvtkey", []byte("pvtvalue-2"))
	helper.checkNoExpiryEntryExistsForBlockNum(3)
	helper.checkExpiryEntryExistsForBlockNum(4, 1)

//Block-4更新：无更新
	noPvtdataUpdates = privacyenabledstate.NewUpdateBatch()
	helper.commitUpdatesForTesting(4, noPvtdataUpdates)
	helper.checkPvtdataDoesNotExist("ns", "coll", "pvtkey")
	helper.checkNoExpiryEntryExistsForBlockNum(4)
}

func TestOnlyHashUpdateInExpiryBlock(t *testing.T) {
	dbEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	ledgerid := "testledger-perge-mgr"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
{"ns", "coll"}: 1, //失效块=提交块+2
		},
	)
	helper := &testHelper{}
	helper.init(t, ledgerid, btlPolicy, dbEnv)
	defer helper.cleanup()

//块1更新：添加pvt数据
	block1Updates := privacyenabledstate.NewUpdateBatch()
	putPvtAndHashUpdates(t, block1Updates,
		"ns", "coll", "pvtkey", []byte("pvtvalue-1"), version.NewHeight(1, 1))
	helper.commitUpdatesForTesting(1, block1Updates)
	helper.checkExpiryEntryExistsForBlockNum(3, 1)

//Block-2更新：无更新
	noPvtdataUpdates := privacyenabledstate.NewUpdateBatch()
	helper.commitUpdatesForTesting(2, noPvtdataUpdates)
	helper.checkPvtdataExists(
		"ns", "coll", "pvtkey", []byte("pvtvalue-1"))
	helper.checkExpiryEntryExistsForBlockNum(3, 1)

//块3更新：仅更新哈希
	block3Updates := privacyenabledstate.NewUpdateBatch()
	putHashUpdates(block3Updates,
		"ns", "coll", "pvtkey", []byte("pvtvalue-3"), version.NewHeight(3, 1))
	helper.commitUpdatesForTesting(3, block3Updates)
	helper.checkOnlyKeyHashExists("ns", "coll", "pvtkey")
	helper.checkNoExpiryEntryExistsForBlockNum(3)
	helper.checkExpiryEntryExistsForBlockNum(5, 1)

//Block-4更新：无更新
	noPvtdataUpdates = privacyenabledstate.NewUpdateBatch()
	helper.commitUpdatesForTesting(4, noPvtdataUpdates)
	helper.checkExpiryEntryExistsForBlockNum(5, 1)

//Block-5更新：无更新
	noPvtdataUpdates = privacyenabledstate.NewUpdateBatch()
	helper.commitUpdatesForTesting(5, noPvtdataUpdates)
	helper.checkPvtdataDoesNotExist("ns", "coll", "pvtkey")
	helper.checkNoExpiryEntryExistsForBlockNum(5)
}

func TestOnlyHashDeleteBeforeExpiryBlock(t *testing.T) {
	dbEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	ledgerid := "testledger-perge-mgr"
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
{"ns", "coll"}: 1, //失效块=提交块+2
		},
	)
	testHelper := &testHelper{}
	testHelper.init(t, ledgerid, btlPolicy, dbEnv)
	defer testHelper.cleanup()

//块1更新：添加pvt键
	block1Updates := privacyenabledstate.NewUpdateBatch()
	putPvtAndHashUpdates(t, block1Updates,
		"ns", "coll", "pvtkey", []byte("pvtvalue-1"), version.NewHeight(1, 1))
	testHelper.commitUpdatesForTesting(1, block1Updates)

//块2更新：仅删除哈希
	block2Updates := privacyenabledstate.NewUpdateBatch()
	deleteHashUpdates(block2Updates, "ns", "coll", "pvtkey", version.NewHeight(2, 1))
	testHelper.commitUpdatesForTesting(2, block2Updates)
	testHelper.checkOnlyPvtKeyExists("ns", "coll", "pvtkey", []byte("pvtvalue-1"))

//Block-3更新：无更新
	noPvtdataUpdates := privacyenabledstate.NewUpdateBatch()
	testHelper.commitUpdatesForTesting(3, noPvtdataUpdates)
	testHelper.checkPvtdataDoesNotExist("ns", "coll", "pvtkey")
}

type testHelper struct {
	t              *testing.T
	bookkeepingEnv *bookkeeping.TestEnv
	dbEnv          privacyenabledstate.TestEnv

	db             privacyenabledstate.DB
	purgeMgr       PurgeMgr
	purgerUsedOnce bool
}

func (h *testHelper) init(t *testing.T, ledgerid string, btlPolicy pvtdatapolicy.BTLPolicy, dbEnv privacyenabledstate.TestEnv) {
	h.t = t
	h.bookkeepingEnv = bookkeeping.NewTestEnv(t)
	dbEnv.Init(t)
	h.dbEnv = dbEnv
	h.db = h.dbEnv.GetDBHandle(ledgerid)
	var err error
	if h.purgeMgr, err = InstantiatePurgeMgr(ledgerid, h.db, btlPolicy, h.bookkeepingEnv.TestProvider); err != nil {
		t.Fatalf("err:%s", err)
	}
}

func (h *testHelper) cleanup() {
	h.bookkeepingEnv.Cleanup()
	h.dbEnv.Cleanup()
}

func (h *testHelper) commitUpdatesForTesting(blkNum uint64, updates *privacyenabledstate.UpdateBatch) {
	h.purgeMgr.PrepareForExpiringKeys(blkNum)
	assert.NoError(h.t, h.purgeMgr.DeleteExpiredAndUpdateBookkeeping(updates.PvtUpdates, updates.HashUpdates))
	assert.NoError(h.t, h.db.ApplyPrivacyAwareUpdates(updates, version.NewHeight(blkNum, 1)))
	h.db.ClearCachedVersions()
	h.purgeMgr.BlockCommitDone()
}

func (h *testHelper) commitPvtDataOfOldBlocksForTesting(updates *privacyenabledstate.UpdateBatch) {
	assert.NoError(h.t, h.purgeMgr.UpdateBookkeepingForPvtDataOfOldBlocks(updates.PvtUpdates))
	assert.NoError(h.t, h.db.ApplyPrivacyAwareUpdates(updates, nil))
}

func (h *testHelper) checkPvtdataExists(ns, coll, key string, value []byte) {
	vv, _ := h.fetchPvtdataFronDB(ns, coll, key)
	vv, hashVersion := h.fetchPvtdataFronDB(ns, coll, key)
	assert.NotNil(h.t, vv)
	assert.Equal(h.t, value, vv.Value)
	assert.Equal(h.t, vv.Version, hashVersion)
}

func (h *testHelper) checkPvtdataDoesNotExist(ns, coll, key string) {
	vv, hashVersion := h.fetchPvtdataFronDB(ns, coll, key)
	assert.Nil(h.t, vv)
	assert.Nil(h.t, hashVersion)
}

func (h *testHelper) checkOnlyPvtKeyExists(ns, coll, key string, value []byte) {
	vv, hashVersion := h.fetchPvtdataFronDB(ns, coll, key)
	assert.NotNil(h.t, vv)
	assert.Nil(h.t, hashVersion)
	assert.Equal(h.t, value, vv.Value)
}

func (h *testHelper) checkOnlyPvtKeyDoesNotExist(ns, coll, key string) {
	kv, err := h.db.GetPrivateData(ns, coll, key)
	assert.Nil(h.t, err)
	assert.Nil(h.t, kv)
}

func (h *testHelper) checkOnlyKeyHashExists(ns, coll, key string) {
	vv, hashVersion := h.fetchPvtdataFronDB(ns, coll, key)
	assert.Nil(h.t, vv)
	assert.NotNil(h.t, hashVersion)
}

func (h *testHelper) fetchPvtdataFronDB(ns, coll, key string) (kv *statedb.VersionedValue, hashVersion *version.Height) {
	var err error
	kv, err = h.db.GetPrivateData(ns, coll, key)
	assert.NoError(h.t, err)
	hashVersion, err = h.db.GetKeyHashVersion(ns, coll, util.ComputeStringHash(key))
	assert.NoError(h.t, err)
	return
}

func (h *testHelper) checkExpiryEntryExistsForBlockNum(expiringBlk uint64, expectedNumEntries int) {
	expInfo, err := h.purgeMgr.(*purgeMgr).expKeeper.retrieve(expiringBlk)
	assert.NoError(h.t, err)
	assert.Len(h.t, expInfo, expectedNumEntries)
}

func (h *testHelper) checkNoExpiryEntryExistsForBlockNum(expiringBlk uint64) {
	expInfo, err := h.purgeMgr.(*purgeMgr).expKeeper.retrieve(expiringBlk)
	assert.NoError(h.t, err)
	assert.Len(h.t, expInfo, 0)
}
