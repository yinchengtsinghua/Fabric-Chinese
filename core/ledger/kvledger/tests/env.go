
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


package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"

	"github.com/hyperledger/fabric/common/ledger/blkstorage/fsblkstorage"
	"github.com/hyperledger/fabric/common/ledger/util"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/scc/lscc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type config map[string]interface{}
type rebuildable uint8

const (
	rebuildableStatedb       rebuildable = 1
	rebuildableBlockIndex    rebuildable = 2
	rebuildableConfigHistory rebuildable = 4
	rebuildableHistoryDB     rebuildable = 8
)

var (
	defaultConfig = config{
		"peer.fileSystemPath":        "/tmp/fabric/ledgertests",
		"ledger.state.stateDatabase": "goleveldb",
	}
)

type env struct {
	assert *assert.Assertions
}

func newEnv(conf config, t *testing.T) *env {
	setupConfigs(conf)
	env := &env{assert.New(t)}
	initLedgerMgmt()
	return env
}

func (e *env) cleanup() {
	closeLedgerMgmt()
	e.assert.NoError(os.RemoveAll(getLedgerRootPath()))
}

func (e *env) closeAllLedgersAndDrop(flags rebuildable) {
	closeLedgerMgmt()
	defer initLedgerMgmt()

	if flags&rebuildableBlockIndex == rebuildableBlockIndex {
		indexPath := getBlockIndexDBPath()
		logger.Infof("Deleting blockstore indexdb path [%s]", indexPath)
		e.verifyNonEmptyDirExists(indexPath)
		e.assert.NoError(os.RemoveAll(indexPath))
	}

	if flags&rebuildableStatedb == rebuildableStatedb {
		statedbPath := getLevelstateDBPath()
		logger.Infof("Deleting statedb path [%s]", statedbPath)
		e.verifyNonEmptyDirExists(statedbPath)
		e.assert.NoError(os.RemoveAll(statedbPath))
	}

	if flags&rebuildableConfigHistory == rebuildableConfigHistory {
		configHistory := getConfigHistoryDBPath()
		logger.Infof("Deleting configHistory db path [%s]", configHistory)
		e.verifyNonEmptyDirExists(configHistory)
		e.assert.NoError(os.RemoveAll(configHistory))
	}
}

func (e *env) verifyRebuilablesExist(flags rebuildable) {
	if flags&rebuildableStatedb == rebuildableBlockIndex {
		e.verifyNonEmptyDirExists(getBlockIndexDBPath())
	}
	if flags&rebuildableBlockIndex == rebuildableStatedb {
		e.verifyNonEmptyDirExists(getLevelstateDBPath())
	}
	if flags&rebuildableConfigHistory == rebuildableConfigHistory {
		e.verifyNonEmptyDirExists(getConfigHistoryDBPath())
	}
}

func (e *env) verifyRebuilableDoesNotExist(flags rebuildable) {
	if flags&rebuildableStatedb == rebuildableStatedb {
		e.verifyDirDoesNotExist(getLevelstateDBPath())
	}
	if flags&rebuildableStatedb == rebuildableBlockIndex {
		e.verifyDirDoesNotExist(getBlockIndexDBPath())
	}
	if flags&rebuildableConfigHistory == rebuildableConfigHistory {
		e.verifyDirDoesNotExist(getConfigHistoryDBPath())
	}
}

func (e *env) verifyNonEmptyDirExists(path string) {
	empty, err := util.DirEmpty(path)
	e.assert.NoError(err)
	e.assert.False(empty)
}

func (e *env) verifyDirDoesNotExist(path string) {
	exists, _, err := util.FileExists(path)
	e.assert.NoError(err)
	e.assert.False(exists)
}

//########################### ledgermgmt and ledgerconfig related functions wrappers #############################
//在当前代码中，ledgermgmt和ledgerconfigs是打包的作用域API，因此下面是
//包装器API。作为TODO，可以将ledgermgmt和ledgerconfig重构为单独的对象，然后
//这两个实例将包装在上面的“env”结构中。
//#################################################################################################################
func setupConfigs(conf config) {
	for c, v := range conf {
		viper.Set(c, v)
	}
}

func initLedgerMgmt() {
	identityDeserializerFactory := func(chainID string) msp.IdentityDeserializer {
		return mgmt.GetManagerForChain(chainID)
	}
	membershipInfoProvider := privdata.NewMembershipInfoProvider(createSelfSignedData(), identityDeserializerFactory)

	ledgermgmt.InitializeExistingTestEnvWithInitializer(
		&ledgermgmt.Initializer{
			CustomTxProcessors:            peer.ConfigTxProcessors,
			DeployedChaincodeInfoProvider: &lscc.DeployedCCInfoProvider{},
			MembershipInfoProvider:        membershipInfoProvider,
			MetricsProvider:               &disabled.Provider{},
		},
	)
}

func createSelfSignedData() common.SignedData {
	sID := mgmt.GetLocalSigningIdentityOrPanic()
	msg := make([]byte, 32)
	sig, err := sID.Sign(msg)
	if err != nil {
		logger.Panicf("Failed creating self signed data because message signing failed: %v", err)
	}
	peerIdentity, err := sID.Serialize()
	if err != nil {
		logger.Panicf("Failed creating self signed data because peer identity couldn't be serialized: %v", err)
	}
	return common.SignedData{
		Data:      msg,
		Signature: sig,
		Identity:  peerIdentity,
	}
}

func closeLedgerMgmt() {
	ledgermgmt.Close()
}

func getLedgerRootPath() string {
	return ledgerconfig.GetRootPath()
}

func getLevelstateDBPath() string {
	return ledgerconfig.GetStateLevelDBPath()
}

func getBlockIndexDBPath() string {
	return filepath.Join(ledgerconfig.GetBlockStorePath(), fsblkstorage.IndexDir)
}

func getConfigHistoryDBPath() string {
	return ledgerconfig.GetConfigHistoryPath()
}
