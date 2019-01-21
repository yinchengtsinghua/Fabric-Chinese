
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有greg haskins<gregory.haskins@gmail.com>2017，保留所有权利。
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func AddConfigPath(v *viper.Viper, p string) {
	if v != nil {
		v.AddConfigPath(p)
	} else {
		viper.AddConfigPath(p)
	}
}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//转换词（）
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//将相对路径转换为与配置相对应的完全限定路径。
//指定它的文件。绝对路径是毫发无损地通过的。
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
func TranslatePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}

	return filepath.Join(base, p)
}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//翻译位置（）
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//将相对路径转换为就地完全限定路径（更新
//指针）相对于指定它的配置文件。绝对路径是
//顺利通过。
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
func TranslatePathInPlace(base string, p *string) {
	*p = TranslatePath(base, *p)
}

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//GETPATH（）
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//GetPath允许指定（配置文件）相对路径的配置字符串
//
//例如：假设我们的配置位于/etc/hyperledger/fabric/core.yaml中，
//一个密钥“MSP.CONTROPATION”=“MSP/CONFIG.YAML”。
//
//此函数将返回：
//getpath（“msp.configpath”）->/etc/hyperledger/fabric/msp/config.yaml
//
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
func GetPath(key string) string {
	p := viper.GetString(key)
	if p == "" {
		return ""
	}

	return TranslatePath(filepath.Dir(viper.ConfigFileUsed()), p)
}

const OfficialPath = "/etc/hyperledger/fabric"

//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//Invivior（）
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
//执行基于viper的配置层的基本初始化。
//主要的推动力是建立应该寻找的路径。
//我们需要的配置。如果v==nil，我们将初始化全局
//蝰蛇实例
//——————————————————————————————————————————————————————————————————————————————————————————————————————————————————
func InitViper(v *viper.Viper, configName string) error {
	var altPath = os.Getenv("FABRIC_CFG_PATH")
	if altPath != "" {
//如果用户用envvar覆盖了路径，则它是唯一的路径
//我们会考虑

		if !dirExists(altPath) {
			return fmt.Errorf("FABRIC_CFG_PATH %s does not exist", altPath)
		}

		AddConfigPath(v, altPath)
	} else {
//如果到达这里，我们应该优先使用默认路径：
//
//＊CWD
//*）/ETC /超分类帐/织物

//随钻测井
		AddConfigPath(v, "./")

//最后，官方途径
		if dirExists(OfficialPath) {
			AddConfigPath(v, OfficialPath)
		}
	}

//现在设置配置文件。
	if v != nil {
		v.SetConfigName(configName)
	} else {
		viper.SetConfigName(configName)
	}

	return nil
}
