
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


package lockbasedtxmgr

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/assert"
)

type testEnv interface {
	init(t *testing.T, testLedgerID string, btlPolicy pvtdatapolicy.BTLPolicy)
	getName() string
	getTxMgr() txmgr.TxMgr
	getVDB() privacyenabledstate.DB
	cleanup()
}

const (
	levelDBtestEnvName = "levelDB_LockBasedTxMgr"
	couchDBtestEnvName = "couchDB_LockBasedTxMgr"
)

//将对此数组中的每个环境运行测试
//例如，要跳过couchdb测试，请删除coach环境的条目。
var testEnvs = []testEnv{
	&lockBasedEnv{name: levelDBtestEnvName, testDBEnv: &privacyenabledstate.LevelDBCommonStorageTestEnv{}},
	&lockBasedEnv{name: couchDBtestEnvName, testDBEnv: &privacyenabledstate.CouchDBCommonStorageTestEnv{}},
}

var testEnvsMap = map[string]testEnv{
	levelDBtestEnvName: testEnvs[0],
	couchDBtestEnvName: testEnvs[1],
}

///////////leveldb环境

type lockBasedEnv struct {
	t            testing.TB
	name         string
	testLedgerID string

	testDBEnv privacyenabledstate.TestEnv
	testDB    privacyenabledstate.DB

	testBookkeepingEnv *bookkeeping.TestEnv

	txmgr txmgr.TxMgr
}

func (env *lockBasedEnv) getName() string {
	return env.name
}

func (env *lockBasedEnv) init(t *testing.T, testLedgerID string, btlPolicy pvtdatapolicy.BTLPolicy) {
	var err error
	env.t = t
	env.testDBEnv.Init(t)
	env.testDB = env.testDBEnv.GetDBHandle(testLedgerID)
	assert.NoError(t, err)
	env.testBookkeepingEnv = bookkeeping.NewTestEnv(t)
	env.txmgr, err = NewLockBasedTxMgr(
		testLedgerID, env.testDB, nil,
		btlPolicy, env.testBookkeepingEnv.TestProvider,
		&mock.DeployedChaincodeInfoProvider{})
	assert.NoError(t, err)

}

func (env *lockBasedEnv) getTxMgr() txmgr.TxMgr {
	return env.txmgr
}

func (env *lockBasedEnv) getVDB() privacyenabledstate.DB {
	return env.testDB
}

func (env *lockBasedEnv) cleanup() {
	env.txmgr.Shutdown()
	env.testDBEnv.Cleanup()
	env.testBookkeepingEnv.Cleanup()
}

//////////// txMgrTestHelper /////////////

type txMgrTestHelper struct {
	t     *testing.T
	txMgr txmgr.TxMgr
	bg    *testutil.BlockGenerator
}

func newTxMgrTestHelper(t *testing.T, txMgr txmgr.TxMgr) *txMgrTestHelper {
	bg, _ := testutil.NewBlockGenerator(t, "testLedger", false)
	return &txMgrTestHelper{t, txMgr, bg}
}

func (h *txMgrTestHelper) validateAndCommitRWSet(txRWSet *rwset.TxReadWriteSet) {
	rwSetBytes, _ := proto.Marshal(txRWSet)
	block := h.bg.NextBlock([][]byte{rwSetBytes})
	_, err := h.txMgr.ValidateAndPrepare(&ledger.BlockAndPvtData{Block: block, PvtData: nil}, true)
	assert.NoError(h.t, err)
	txsFltr := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	invalidTxNum := 0
	for i := 0; i < len(block.Data.Data); i++ {
		if txsFltr.IsInvalid(i) {
			invalidTxNum++
		}
	}
	assert.Equal(h.t, 0, invalidTxNum)
	err = h.txMgr.Commit()
	assert.NoError(h.t, err)
}

func (h *txMgrTestHelper) checkRWsetInvalid(txRWSet *rwset.TxReadWriteSet) {
	rwSetBytes, _ := proto.Marshal(txRWSet)
	block := h.bg.NextBlock([][]byte{rwSetBytes})
	_, err := h.txMgr.ValidateAndPrepare(&ledger.BlockAndPvtData{Block: block, PvtData: nil}, true)
	assert.NoError(h.t, err)
	txsFltr := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	invalidTxNum := 0
	for i := 0; i < len(block.Data.Data); i++ {
		if txsFltr.IsInvalid(i) {
			invalidTxNum++
		}
	}
	assert.Equal(h.t, 1, invalidTxNum)
}

func populateCollConfigForTest(t *testing.T, txMgr *LockBasedTxMgr, nsColls []collConfigkey, ht *version.Height) {
	m := map[string]*common.CollectionConfigPackage{}
	for _, nsColl := range nsColls {
		ns, coll := nsColl.ns, nsColl.coll
		pkg, ok := m[ns]
		if !ok {
			pkg = &common.CollectionConfigPackage{}
			m[ns] = pkg
		}
		sCollConfig := &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name: coll,
			},
		}
		pkg.Config = append(pkg.Config, &common.CollectionConfig{Payload: sCollConfig})
	}
	ccInfoProvider := &mock.DeployedChaincodeInfoProvider{}
	ccInfoProvider.ChaincodeInfoStub = func(ccName string, qe ledger.SimpleQueryExecutor) (*ledger.DeployedChaincodeInfo, error) {
		fmt.Printf("retrieveing info for [%s] from [%s]\n", ccName, m)
		return &ledger.DeployedChaincodeInfo{Name: ccName, CollectionConfigPkg: m[ccName]}, nil
	}
	txMgr.ccInfoProvider = ccInfoProvider
}

func testutilPopulateDB(t *testing.T, txMgr *LockBasedTxMgr, ns string, data []*queryresult.KV, version *version.Height) {
	updates := privacyenabledstate.NewUpdateBatch()
	for _, kv := range data {
		updates.PubUpdates.Put(ns, kv.Key, kv.Value, version)
	}
	txMgr.db.ApplyPrivacyAwareUpdates(updates, version)
}
