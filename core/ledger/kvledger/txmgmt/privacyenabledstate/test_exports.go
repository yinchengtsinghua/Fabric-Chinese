
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


package privacyenabledstate

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/integration/runner"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

//测试环境实现的接口
type TestEnv interface {
	Init(t testing.TB)
	GetDBHandle(id string) DB
	GetName() string
	Cleanup()
}

//将对此数组中的每个环境运行测试
//例如，要跳过couchdb测试，请删除&couchdblockbasedenv
//var testenvs=[]testenv&leveldbcommonstoragetestenv，&couchdbcommonstoragetestenv
var testEnvs = []TestEnv{&LevelDBCommonStorageTestEnv{}, &CouchDBCommonStorageTestEnv{}}

///////////leveldb环境

//leveldbcommonstoragetestenv为基于leveldb的存储实现testenv接口
type LevelDBCommonStorageTestEnv struct {
	t                 testing.TB
	provider          DBProvider
	bookkeeperTestEnv *bookkeeping.TestEnv
}

//init从接口testenv实现相应的函数
func (env *LevelDBCommonStorageTestEnv) Init(t testing.TB) {
	viper.Set("ledger.state.stateDatabase", "")
	removeDBPath(t)
	env.bookkeeperTestEnv = bookkeeping.NewTestEnv(t)
	dbProvider, err := NewCommonStorageDBProvider(env.bookkeeperTestEnv.TestProvider, &disabled.Provider{}, &mock.HealthCheckRegistry{})
	assert.NoError(t, err)
	env.t = t
	env.provider = dbProvider
}

//getdbhandle从接口testenv实现相应的函数
func (env *LevelDBCommonStorageTestEnv) GetDBHandle(id string) DB {
	db, err := env.provider.GetDBHandle(id)
	assert.NoError(env.t, err)
	return db
}

//getname从接口testenv实现相应的函数
func (env *LevelDBCommonStorageTestEnv) GetName() string {
	return "levelDBCommonStorageTestEnv"
}

//cleanup从接口testenv实现相应的函数
func (env *LevelDBCommonStorageTestEnv) Cleanup() {
	env.provider.Close()
	env.bookkeeperTestEnv.Cleanup()
	removeDBPath(env.t)
}

///////////couchdb环境

//couchdbcommonstoragetestenv为基于couchdb的存储实现testenv接口
type CouchDBCommonStorageTestEnv struct {
	t                 testing.TB
	provider          DBProvider
	bookkeeperTestEnv *bookkeeping.TestEnv
	couchCleanup      func()
}

func (env *CouchDBCommonStorageTestEnv) setupCouch() string {
	externalCouch, set := os.LookupEnv("COUCHDB_ADDR")
	if set {
		env.couchCleanup = func() {}
		return externalCouch
	}

	couchDB := &runner.CouchDB{}
	if err := couchDB.Start(); err != nil {
		err := fmt.Errorf("failed to start couchDB: %s", err)
		panic(err)
	}
	env.couchCleanup = func() { couchDB.Stop() }
	return couchDB.Address()
}

//init从接口testenv实现相应的函数
func (env *CouchDBCommonStorageTestEnv) Init(t testing.TB) {
	viper.Set("ledger.state.stateDatabase", "CouchDB")
	couchAddr := env.setupCouch()
	viper.Set("ledger.state.couchDBConfig.couchDBAddress", couchAddr)
//替换为正确的用户名/密码，例如
//如果在CouchDB上启用了用户安全性，则为admin/admin。
	viper.Set("ledger.state.couchDBConfig.username", "")
	viper.Set("ledger.state.couchDBConfig.password", "")
	viper.Set("ledger.state.couchDBConfig.maxRetries", 3)
	viper.Set("ledger.state.couchDBConfig.maxRetriesOnStartup", 20)
	viper.Set("ledger.state.couchDBConfig.requestTimeout", time.Second*35)

	env.bookkeeperTestEnv = bookkeeping.NewTestEnv(t)
	dbProvider, err := NewCommonStorageDBProvider(env.bookkeeperTestEnv.TestProvider, &disabled.Provider{}, &mock.HealthCheckRegistry{})
	assert.NoError(t, err)
	env.t = t
	env.provider = dbProvider
}

//getdbhandle从接口testenv实现相应的函数
func (env *CouchDBCommonStorageTestEnv) GetDBHandle(id string) DB {
	db, err := env.provider.GetDBHandle(id)
	assert.NoError(env.t, err)
	return db
}

//getname从接口testenv实现相应的函数
func (env *CouchDBCommonStorageTestEnv) GetName() string {
	return "couchDBCommonStorageTestEnv"
}

//cleanup从接口testenv实现相应的函数
func (env *CouchDBCommonStorageTestEnv) Cleanup() {
	csdbProvider, _ := env.provider.(*CommonStorageDBProvider)
	statecouchdb.CleanupDB(env.t, csdbProvider.VersionedDBProvider)

	env.bookkeeperTestEnv.Cleanup()
	env.provider.Close()
	env.couchCleanup()
}

func removeDBPath(t testing.TB) {
	dbPath := ledgerconfig.GetStateLevelDBPath()
	if err := os.RemoveAll(dbPath); err != nil {
		t.Fatalf("Err: %s", err)
		t.FailNow()
	}
}
