
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+build go1.9，linux，cgo go1.10，达尔文，cgo
//+建设！PPC64

/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/

package factory

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

//启用race生成标记时，raceEnabled设置为true。
//参见race_test.go
var raceEnabled bool

func buildPlugin(lib string, t *testing.T) {
	t.Helper()
//检查示例插件是否存在
	if _, err := os.Stat(lib); err != nil {
//构建示例插件
		cmd := exec.Command("go", "build", "-buildmode=plugin")
		if raceEnabled {
			cmd.Args = append(cmd.Args, "-race")
		}
		cmd.Args = append(cmd.Args, "github.com/hyperledger/fabric/examples/plugins/bccsp")
		err := cmd.Run()
		if err != nil {
			t.Fatalf("Could not build plugin: [%s]", err)
		}
	}
}

func TestPluginFactoryName(t *testing.T) {
	f := &PluginFactory{}
	assert.Equal(t, f.Name(), PluginFactoryName)
}

func TestPluginFactoryInvalidConfig(t *testing.T) {
	f := &PluginFactory{}
	opts := &FactoryOpts{}

	_, err := f.Get(nil)
	assert.Error(t, err)

	_, err = f.Get(opts)
	assert.Error(t, err)

	opts.PluginOpts = &PluginOpts{}
	_, err = f.Get(opts)
	assert.Error(t, err)
}

func TestPluginFactoryValidConfig(t *testing.T) {
//构建插件
	lib := "./bccsp.so"
	defer os.Remove(lib)
	buildPlugin(lib, t)

	f := &PluginFactory{}
	opts := &FactoryOpts{
		PluginOpts: &PluginOpts{
			Library: lib,
		},
	}

	csp, err := f.Get(opts)
	assert.NoError(t, err)
	assert.NotNil(t, csp)

	_, err = csp.GetKey([]byte{123})
	assert.NoError(t, err)
}

func TestPluginFactoryFromOpts(t *testing.T) {
//构建插件
	lib := "./bccsp.so"
	defer os.Remove(lib)
	buildPlugin(lib, t)

	opts := &FactoryOpts{
		ProviderName: "PLUGIN",
		PluginOpts: &PluginOpts{
			Library: lib,
		},
	}
	csp, err := GetBCCSPFromOpts(opts)
	assert.NoError(t, err)
	assert.NotNil(t, csp)

	_, err = csp.GetKey([]byte{123})
	assert.NoError(t, err)
}
