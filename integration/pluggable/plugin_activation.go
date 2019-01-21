
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


package pluggable

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	EndorsementPluginEnvVar = "ENDORSEMENT_PLUGIN_ENV_VAR"
	ValidationPluginEnvVar  = "VALIDATION_PLUGIN_ENV_VAR"
)

//背书PlugInactivationFolder返回如果
//其中存在对等方ID的文件-表示已激活认可插件
//为了那个同伴
func EndorsementPluginActivationFolder() string {
	return os.Getenv(EndorsementPluginEnvVar)
}

//set背书pluginactivationfolder设置文件夹的名称
//如果对等方ID的文件存在，则表示认可插件已激活
//为了那个同伴
func SetEndorsementPluginActivationFolder(path string) {
	os.Setenv(EndorsementPluginEnvVar, path)
}

//validationpluginactivationfilepath返回如果
//其中存在对等方ID的文件-表示验证插件已激活
//为了那个同伴
func ValidationPluginActivationFolder() string {
	return os.Getenv(ValidationPluginEnvVar)
}

//setvalidationpluginactivationfolder设置文件夹的名称
//如果对等方ID的文件存在，则表示验证插件已激活
//为了那个同伴
func SetValidationPluginActivationFolder(path string) {
	os.Setenv(ValidationPluginEnvVar, path)
}

func markPluginActivation(dir string) {
	fileName := filepath.Join(dir, viper.GetString("peer.id"))
	_, err := os.Create(fileName)
	if err != nil {
		panic(fmt.Sprintf("failed to create file %s: %v", fileName, err))
	}
}

//PublishRemarkementPlugInactivation使其知道认可插件
//已为正在调用此函数的对等机激活
func PublishEndorsementPluginActivation() {
	markPluginActivation(EndorsementPluginActivationFolder())
}

//PublishValidationPlugInactivation使其知道验证插件
//已为正在调用此函数的对等机激活
func PublishValidationPluginActivation() {
	markPluginActivation(ValidationPluginActivationFolder())
}

//CountRemarkementPlugInactivations返回激活的对等方数
//背书插件
func CountEndorsementPluginActivations() int {
	return listDir(EndorsementPluginActivationFolder())
}

//CountValidationPlugInactivations返回激活的对等机数
//验证插件
func CountValidationPluginActivations() int {
	return listDir(ValidationPluginActivationFolder())
}

func listDir(d string) int {
	dir, err := ioutil.ReadDir(d)
	if err != nil {
		panic(fmt.Sprintf("failed listing directory %s: %v", d, err))
	}
	return len(dir)
}
