
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


package configtest_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConfig_AddDevConfigPath(t *testing.T) {
//案例1：使用蝰蛇实例调用AdvDebug配置路径
	v := viper.New()
	err := configtest.AddDevConfigPath(v)
	assert.NoError(t, err, "Error while adding dev config path to viper")

//案例2：调用adddevconfigpath的默认viper实例
	err = configtest.AddDevConfigPath(nil)
	assert.NoError(t, err, "Error while adding dev config path to default viper")

//Error case: GOPATH is empty
	gopath := os.Getenv("GOPATH")
	os.Setenv("GOPATH", "")
	defer os.Setenv("GOPATH", gopath)
	err = configtest.AddDevConfigPath(v)
	assert.Error(t, err, "GOPATH is empty, expected error from AddDevConfigPath")
}

func TestConfig_GetDevMspDir(t *testing.T) {
//成功案例
	_, err := configtest.GetDevMspDir()
	assert.NoError(t, err)

//错误情况：gopath为空
	gopath := os.Getenv("GOPATH")
	os.Setenv("GOPATH", "")
	defer os.Setenv("GOPATH", gopath)
	_, err = configtest.GetDevMspDir()
	assert.Error(t, err, "GOPATH is empty, expected error from GetDevMspDir")

//Error case: GOPATH is set to temp dir
	dir, err1 := ioutil.TempDir("/tmp", "devmspdir")
	assert.NoError(t, err1)
	defer os.RemoveAll(dir)
	os.Setenv("GOPATH", dir)
	_, err = configtest.GetDevMspDir()
	assert.Error(t, err, "GOPATH is set to temp dir, expected error from GetDevMspDir")
}

func TestConfig_GetDevConfigDir(t *testing.T) {
	gopath := os.Getenv("GOPATH")
	os.Setenv("GOPATH", "")
	defer os.Setenv("GOPATH", gopath)
	_, err := configtest.GetDevConfigDir()
	assert.Error(t, err, "GOPATH is empty, expected error from GetDevConfigDir")
}
