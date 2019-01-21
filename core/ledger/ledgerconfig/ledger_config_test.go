
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


package ledgerconfig

import (
	"testing"

	ledgertestutil "github.com/hyperledger/fabric/core/ledger/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestIsCouchDBEnabledDefault(t *testing.T) {
	setUpCoreYAMLConfig()
//在生成期间，默认值应为假。

//If the  ledger test are run with CouchDb enabled, need to provide a mechanism
//允许此测试运行但仍测试默认值。
	if IsCouchDBEnabled() == true {
		ledgertestutil.ResetConfigToDefaultValues()
		defer viper.Set("ledger.state.stateDatabase", "CouchDB")
	}
	defaultValue := IsCouchDBEnabled()
assert.False(t, defaultValue) //测试默认配置为假
}

func TestIsCouchDBEnabled(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.stateDatabase", "CouchDB")
	updatedValue := IsCouchDBEnabled()
assert.True(t, updatedValue) //测试配置返回true
}

func TestLedgerConfigPathDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	assert.Equal(t, "/var/hyperledger/production/ledgersData", GetRootPath())
	assert.Equal(t, "/var/hyperledger/production/ledgersData/ledgerProvider", GetLedgerProviderPath())
	assert.Equal(t, "/var/hyperledger/production/ledgersData/stateLeveldb", GetStateLevelDBPath())
	assert.Equal(t, "/var/hyperledger/production/ledgersData/historyLeveldb", GetHistoryLevelDBPath())
	assert.Equal(t, "/var/hyperledger/production/ledgersData/chains", GetBlockStorePath())
	assert.Equal(t, "/var/hyperledger/production/ledgersData/pvtdataStore", GetPvtdataStorePath())
	assert.Equal(t, "/var/hyperledger/production/ledgersData/bookkeeper", GetInternalBookkeeperPath())
}

func TestLedgerConfigPath(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("peer.fileSystemPath", "/tmp/hyperledger/production")
	assert.Equal(t, "/tmp/hyperledger/production/ledgersData", GetRootPath())
	assert.Equal(t, "/tmp/hyperledger/production/ledgersData/ledgerProvider", GetLedgerProviderPath())
	assert.Equal(t, "/tmp/hyperledger/production/ledgersData/stateLeveldb", GetStateLevelDBPath())
	assert.Equal(t, "/tmp/hyperledger/production/ledgersData/historyLeveldb", GetHistoryLevelDBPath())
	assert.Equal(t, "/tmp/hyperledger/production/ledgersData/chains", GetBlockStorePath())
	assert.Equal(t, "/tmp/hyperledger/production/ledgersData/pvtdataStore", GetPvtdataStorePath())
	assert.Equal(t, "/tmp/hyperledger/production/ledgersData/bookkeeper", GetInternalBookkeeperPath())
}

func TestGetTotalLimitDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetTotalQueryLimit()
assert.Equal(t, 10000, defaultValue) //测试默认配置为1000
}

func TestGetTotalLimitUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetTotalQueryLimit()
assert.Equal(t, 10000, defaultValue) //测试默认配置为1000
}

func TestGetTotalLimit(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.totalQueryLimit", 5000)
	updatedValue := GetTotalQueryLimit()
assert.Equal(t, 5000, updatedValue) //test config returns 5000
}

func TestGetQueryLimitDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetInternalQueryLimit()
assert.Equal(t, 1000, defaultValue) //测试默认配置为1000
}

func TestGetQueryLimitUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetInternalQueryLimit()
assert.Equal(t, 1000, defaultValue) //测试默认配置为1000
}

func TestGetQueryLimit(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.internalQueryLimit", 5000)
	updatedValue := GetInternalQueryLimit()
assert.Equal(t, 5000, updatedValue) //测试配置返回5000
}

func TestMaxBatchUpdateSizeDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetMaxBatchUpdateSize()
assert.Equal(t, 1000, defaultValue) //测试默认配置为1000
}

func TestMaxBatchUpdateSizeUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetMaxBatchUpdateSize()
assert.Equal(t, 500, defaultValue) //如果未设置maxbatchupdatesize，则为500
}

func TestMaxBatchUpdateSize(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.maxBatchUpdateSize", 2000)
	updatedValue := GetMaxBatchUpdateSize()
assert.Equal(t, 2000, updatedValue) //测试配置返回2000
}

func TestPvtdataStorePurgeIntervalDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetPvtdataStorePurgeInterval()
assert.Equal(t, uint64(100), defaultValue) //测试默认配置为100
}

func TestPvtdataStorePurgeIntervalUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetPvtdataStorePurgeInterval()
assert.Equal(t, uint64(100), defaultValue) //如果未设置purgeinterval，则为100
}

func TestIsQueryReadHasingEnabled(t *testing.T) {
	assert.True(t, IsQueryReadsHashingEnabled())
}

func TestGetMaxDegreeQueryReadsHashing(t *testing.T) {
	assert.Equal(t, uint32(50), GetMaxDegreeQueryReadsHashing())
}

func TestPvtdataStorePurgeInterval(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.pvtdataStore.purgeInterval", 1000)
	updatedValue := GetPvtdataStorePurgeInterval()
assert.Equal(t, uint64(1000), updatedValue) //测试配置返回1000
}

func TestPvtdataStoreCollElgProcMaxDbBatchSize(t *testing.T) {
	defaultVal := confCollElgProcMaxDbBatchSize.DefaultVal
	testVal := defaultVal + 1
	assert.Equal(t, defaultVal, GetPvtdataStoreCollElgProcMaxDbBatchSize())
	viper.Set("ledger.pvtdataStore.collElgProcMaxDbBatchSize", testVal)
	assert.Equal(t, testVal, GetPvtdataStoreCollElgProcMaxDbBatchSize())
}

func TestCollElgProcDbBatchesInterval(t *testing.T) {
	defaultVal := confCollElgProcDbBatchesInterval.DefaultVal
	testVal := defaultVal + 1
	assert.Equal(t, defaultVal, GetPvtdataStoreCollElgProcDbBatchesInterval())
	viper.Set("ledger.pvtdataStore.collElgProcDbBatchesInterval", testVal)
	assert.Equal(t, testVal, GetPvtdataStoreCollElgProcDbBatchesInterval())
}

func TestIsHistoryDBEnabledDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := IsHistoryDBEnabled()
assert.False(t, defaultValue) //测试默认配置为假
}

func TestIsHistoryDBEnabledTrue(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.history.enableHistoryDatabase", true)
	updatedValue := IsHistoryDBEnabled()
assert.True(t, updatedValue) //测试配置返回true
}

func TestIsHistoryDBEnabledFalse(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.history.enableHistoryDatabase", false)
	updatedValue := IsHistoryDBEnabled()
assert.False(t, updatedValue) //测试配置返回假
}

func TestIsAutoWarmIndexesEnabledDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := IsAutoWarmIndexesEnabled()
assert.True(t, defaultValue) //测试默认配置为true
}

func TestIsAutoWarmIndexesEnabledUnset(t *testing.T) {
	viper.Reset()
	defaultValue := IsAutoWarmIndexesEnabled()
assert.True(t, defaultValue) //测试默认配置为true
}

func TestIsAutoWarmIndexesEnabledTrue(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.autoWarmIndexes", true)
	updatedValue := IsAutoWarmIndexesEnabled()
assert.True(t, updatedValue) //测试配置返回true
}

func TestIsAutoWarmIndexesEnabledFalse(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.autoWarmIndexes", false)
	updatedValue := IsAutoWarmIndexesEnabled()
assert.False(t, updatedValue) //测试配置返回假
}

func TestGetWarmIndexesAfterNBlocksDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetWarmIndexesAfterNBlocks()
assert.Equal(t, 1, defaultValue) //测试默认配置为true
}

func TestGetWarmIndexesAfterNBlocksUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetWarmIndexesAfterNBlocks()
assert.Equal(t, 1, defaultValue) //测试默认配置为true
}

func TestGetWarmIndexesAfterNBlocks(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.warmIndexesAfterNBlocks", 10)
	updatedValue := GetWarmIndexesAfterNBlocks()
	assert.Equal(t, 10, updatedValue)
}

func TestGetMaxBlockfileSize(t *testing.T) {
	assert.Equal(t, 67108864, GetMaxBlockfileSize())
}

func setUpCoreYAMLConfig() {
//调用helper方法加载core.yaml
	ledgertestutil.SetupCoreYAMLConfig()
}
