
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+构建插件已启用，CGO
//+构建达尔文，go1.10 linux，go1.10 linux，go1.9，！PPC64

/*
版权所有SecureKey Technologies Inc.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package scc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	examplePluginPackage = "github.com/hyperledger/fabric/examples/plugins/scc"
	pluginName           = "testscc"
)

func TestLoadSCCPlugin(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "scc-plugin")
	require.NoError(t, err)

	pluginPath := filepath.Join(tmpdir, "scc-plugin.so")
	buildExamplePlugin(t, pluginPath, examplePluginPackage)
	defer os.RemoveAll(tmpdir)

	testConfig := fmt.Sprintf(`
  chaincode:
    systemPlugins:
      - enabled: true
        name: %s
        path: %s
        invokableExternal: true
        invokableCC2CC: true
  `, pluginName, pluginPath)
	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer([]byte(testConfig)))

	sccs := loadSysCCs(&Provider{})
	assert.Len(t, sccs, 1, "expected one SCC to be loaded")
	resp := sccs[0].Chaincode.Invoke(nil)
	assert.Equal(t, int32(shim.OK), resp.Status, "expected success response from scc")
}

func TestLoadSCCPluginInvalid(t *testing.T) {
	assert.Panics(t, func() { loadPlugin("missing.so") }, "expected panic with invalid path")
}

//启用race生成标记时，raceEnabled设置为true。
//参见race_test.go
var raceEnabled bool

func buildExamplePlugin(t *testing.T, path, pluginPackage string) {
	cmd := exec.Command("go", "build", "-o", path, "-buildmode=plugin")
	if raceEnabled {
		cmd.Args = append(cmd.Args, "-race")
	}
	cmd.Args = append(cmd.Args, pluginPackage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %s, Could not build plugin: %s", err, output)
	}
}
