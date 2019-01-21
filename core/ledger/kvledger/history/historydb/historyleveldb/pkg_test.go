
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


package historyleveldb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/blkstorage/fsblkstorage"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/history/historydb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr/lockbasedtxmgr"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

/////levelblockbasedhistoryenv///

type levelDBLockBasedHistoryEnv struct {
	t                   testing.TB
	testBlockStorageEnv *testBlockStoreEnv

	testDBEnv          privacyenabledstate.TestEnv
	testBookkeepingEnv *bookkeeping.TestEnv

	txmgr txmgr.TxMgr

	testHistoryDBProvider historydb.HistoryDBProvider
	testHistoryDB         historydb.HistoryDB
}

func newTestHistoryEnv(t *testing.T) *levelDBLockBasedHistoryEnv {
	viper.Set("ledger.history.enableHistoryDatabase", "true")
	testLedgerID := "TestLedger"

	blockStorageTestEnv := newBlockStorageTestEnv(t)

	testDBEnv := &privacyenabledstate.LevelDBCommonStorageTestEnv{}
	testDBEnv.Init(t)
	testDB := testDBEnv.GetDBHandle(testLedgerID)
	testBookkeepingEnv := bookkeeping.NewTestEnv(t)

	txMgr, err := lockbasedtxmgr.NewLockBasedTxMgr(testLedgerID, testDB, nil, nil, testBookkeepingEnv.TestProvider, &mock.DeployedChaincodeInfoProvider{})
	assert.NoError(t, err)
	testHistoryDBProvider := NewHistoryDBProvider()
	testHistoryDB, err := testHistoryDBProvider.GetDBHandle("TestHistoryDB")
	assert.NoError(t, err)

	return &levelDBLockBasedHistoryEnv{t,
		blockStorageTestEnv, testDBEnv, testBookkeepingEnv,
		txMgr, testHistoryDBProvider, testHistoryDB}
}

func (env *levelDBLockBasedHistoryEnv) cleanup() {
	env.txmgr.Shutdown()
	env.testDBEnv.Cleanup()
	env.testBlockStorageEnv.cleanup()
	env.testBookkeepingEnv.Cleanup()
//清除历史记录
	env.testHistoryDBProvider.Close()
	removeDBPath(env.t)
}

func removeDBPath(t testing.TB) {
	removePath(t, ledgerconfig.GetHistoryLevelDBPath())
}

func removePath(t testing.TB, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Err: %s", err)
		t.FailNow()
	}
}

/////testblockstoreenv///

type testBlockStoreEnv struct {
	t               testing.TB
	provider        *fsblkstorage.FsBlockstoreProvider
	blockStorageDir string
}

func newBlockStorageTestEnv(t testing.TB) *testBlockStoreEnv {

	testPath, err := ioutil.TempDir("", "historyleveldb-")
	if err != nil {
		panic(err)
	}
	conf := fsblkstorage.NewConf(testPath, 0)

	attrsToIndex := []blkstorage.IndexableAttr{
		blkstorage.IndexableAttrBlockHash,
		blkstorage.IndexableAttrBlockNum,
		blkstorage.IndexableAttrTxID,
		blkstorage.IndexableAttrBlockNumTranNum,
	}
	indexConfig := &blkstorage.IndexConfig{AttrsToIndex: attrsToIndex}

	blockStorageProvider := fsblkstorage.NewProvider(conf, indexConfig).(*fsblkstorage.FsBlockstoreProvider)

	return &testBlockStoreEnv{t, blockStorageProvider, testPath}
}

func (env *testBlockStoreEnv) cleanup() {
	env.provider.Close()
	env.removeFSPath()
}

func (env *testBlockStoreEnv) removeFSPath() {
	fsPath := env.blockStorageDir
	os.RemoveAll(fsPath)
}
