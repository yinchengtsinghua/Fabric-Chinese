
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/tools/configtxgen/configtxgentest"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/stretchr/testify/assert"
)

var tmpDir string

func TestMain(m *testing.M) {
	dir, err := ioutil.TempDir("", "configtxgen")
	if err != nil {
		panic("Error creating temp dir")
	}
	tmpDir = dir
	testResult := m.Run()
	os.RemoveAll(dir)

	os.Exit(testResult)
}

func TestInspectMissing(t *testing.T) {
	assert.Error(t, doInspectBlock("NonSenseBlockFileThatDoesn'tActuallyExist"), "Missing block")
}

func TestInspectBlock(t *testing.T) {
	blockDest := filepath.Join(tmpDir, "block")

	config := configtxgentest.Load(genesisconfig.SampleInsecureSoloProfile)

	assert.NoError(t, doOutputBlock(config, "foo", blockDest), "Good block generation request")
	assert.NoError(t, doInspectBlock(blockDest), "Good block inspection request")
}

func TestMissingOrdererSection(t *testing.T) {
	blockDest := filepath.Join(tmpDir, "block")

	config := configtxgentest.Load(genesisconfig.SampleInsecureSoloProfile)
	config.Orderer = nil

	assert.Panics(t, func() { doOutputBlock(config, "foo", blockDest) }, "Missing orderer section")
}

func TestMissingConsortiumSection(t *testing.T) {
	blockDest := filepath.Join(tmpDir, "block")

	config := configtxgentest.Load(genesisconfig.SampleInsecureSoloProfile)
	config.Consortiums = nil

	assert.NoError(t, doOutputBlock(config, "foo", blockDest), "Missing consortiums section")
}

func TestMissingConsortiumValue(t *testing.T) {
	configTxDest := filepath.Join(tmpDir, "configtx")

	config := configtxgentest.Load(genesisconfig.SampleSingleMSPChannelProfile)
	config.Consortium = ""

	assert.Error(t, doOutputChannelCreateTx(config, "foo", configTxDest), "Missing Consortium value in Application Profile definition")
}

func TestMissingApplicationValue(t *testing.T) {
	configTxDest := filepath.Join(tmpDir, "configtx")

	config := configtxgentest.Load(genesisconfig.SampleSingleMSPChannelProfile)
	config.Application = nil

	assert.Error(t, doOutputChannelCreateTx(config, "foo", configTxDest), "Missing Application value in Application Profile definition")
}

func TestInspectMissingConfigTx(t *testing.T) {
	assert.Error(t, doInspectChannelCreateTx("ChannelCreateTxFileWhichDoesn'tReallyExist"), "Missing channel create tx file")
}

func TestInspectConfigTx(t *testing.T) {
	configTxDest := filepath.Join(tmpDir, "configtx")

	config := configtxgentest.Load(genesisconfig.SampleSingleMSPChannelProfile)

	assert.NoError(t, doOutputChannelCreateTx(config, "foo", configTxDest), "Good outputChannelCreateTx generation request")
	assert.NoError(t, doInspectChannelCreateTx(configTxDest), "Good configtx inspection request")
}

func TestGenerateAnchorPeersUpdate(t *testing.T) {
	configTxDest := filepath.Join(tmpDir, "anchorPeerUpdate")

	config := configtxgentest.Load(genesisconfig.SampleSingleMSPChannelProfile)

	assert.NoError(t, doOutputAnchorPeersUpdate(config, "foo", configTxDest, genesisconfig.SampleOrgName), "Good anchorPeerUpdate request")
}

func TestBadAnchorPeersUpdates(t *testing.T) {
	configTxDest := filepath.Join(tmpDir, "anchorPeerUpdate")

	config := configtxgentest.Load(genesisconfig.SampleSingleMSPChannelProfile)

	assert.Error(t, doOutputAnchorPeersUpdate(config, "foo", configTxDest, ""), "Bad anchorPeerUpdate request - asOrg empty")

	backupApplication := config.Application
	config.Application = nil
	assert.Error(t, doOutputAnchorPeersUpdate(config, "foo", configTxDest, genesisconfig.SampleOrgName), "Bad anchorPeerUpdate request")
	config.Application = backupApplication

	config.Application.Organizations[0] = &genesisconfig.Organization{Name: "FakeOrg", ID: "FakeOrg"}
	assert.Error(t, doOutputAnchorPeersUpdate(config, "foo", configTxDest, genesisconfig.SampleOrgName), "Bad anchorPeerUpdate request - fake org")
}

func TestConfigTxFlags(t *testing.T) {
	configTxDest := filepath.Join(tmpDir, "configtx")
	configTxDestAnchorPeers := filepath.Join(tmpDir, "configtxAnchorPeers")

	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()
	devConfigDir, err := configtest.GetDevConfigDir()
	assert.NoError(t, err, "failed to get dev config dir")

	os.Args = []string{
		"cmd",
		"-outputCreateChannelTx=" + configTxDest,
		"-profile=" + genesisconfig.SampleSingleMSPChannelProfile,
		"-configPath=" + devConfigDir,
		"-inspectChannelCreateTx=" + configTxDest,
		"-outputAnchorPeersUpdate=" + configTxDestAnchorPeers,
		"-asOrg=" + genesisconfig.SampleOrgName,
	}

	main()

	_, err = os.Stat(configTxDest)
	assert.NoError(t, err, "Configtx file is written successfully")
	_, err = os.Stat(configTxDestAnchorPeers)
	assert.NoError(t, err, "Configtx anchor peers file is written successfully")
}

func TestBlockFlags(t *testing.T) {
	blockDest := filepath.Join(tmpDir, "block")
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()
	os.Args = []string{
		"cmd",
		"-profile=" + genesisconfig.SampleSingleMSPSoloProfile,
		"-outputBlock=" + blockDest,
		"-inspectBlock=" + blockDest,
	}
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	main()

	_, err := os.Stat(blockDest)
	assert.NoError(t, err, "Block file is written successfully")
}

func TestPrintOrg(t *testing.T) {
	factory.InitFactories(nil)
	config := configtxgentest.LoadTopLevel()

	assert.NoError(t, doPrintOrg(config, genesisconfig.SampleOrgName), "Good org to print")

	err := doPrintOrg(config, genesisconfig.SampleOrgName+".wrong")
	assert.Error(t, err, "Bad org name")
	assert.Regexp(t, "organization [^ ]* not found", err.Error())

	config.Organizations[0] = &genesisconfig.Organization{Name: "FakeOrg", ID: "FakeOrg"}
	err = doPrintOrg(config, "FakeOrg")
	assert.Error(t, err, "Fake org")
	assert.Regexp(t, "bad org definition", err.Error())
}
