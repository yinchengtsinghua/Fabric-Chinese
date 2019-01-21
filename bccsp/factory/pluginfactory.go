
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

package factory

import (
	"errors"
	"fmt"
	"os"
	"plugin"

	"github.com/hyperledger/fabric/bccsp"
)

const (
//PlugInfactoryName是BCCSP插件的工厂名称
	PluginFactoryName = "PLUGIN"
)

//PluginOpts包含PluginFactory的选项
type PluginOpts struct {
//插件库路径
	Library string
//插件库的配置映射
	Config map[string]interface{}
}

//PlugInfectory是BCCSP插件的工厂
type PluginFactory struct{}

//name返回此工厂的名称
func (f *PluginFactory) Name() string {
	return PluginFactoryName
}

//get返回使用opts的bccsp实例。
func (f *PluginFactory) Get(config *FactoryOpts) (bccsp.BCCSP, error) {
//检查有效配置
	if config == nil || config.PluginOpts == nil {
		return nil, errors.New("Invalid config. It must not be nil.")
	}

//库是必需属性
	if config.PluginOpts.Library == "" {
		return nil, errors.New("Invalid config: missing property 'Library'")
	}

//确保库存在
	if _, err := os.Stat(config.PluginOpts.Library); err != nil {
		return nil, fmt.Errorf("Could not find library '%s' [%s]", config.PluginOpts.Library, err)
	}

//尝试将库作为插件加载
	plug, err := plugin.Open(config.PluginOpts.Library)
	if err != nil {
		return nil, fmt.Errorf("Failed to load plugin '%s' [%s]", config.PluginOpts.Library, err)
	}

//查找所需符号“new”
	sym, err := plug.Lookup("New")
	if err != nil {
		return nil, fmt.Errorf("Could not find required symbol 'CryptoServiceProvider' [%s]", err)
	}

//检查以确保新符号符合所需的功能签名
	new, ok := sym.(func(config map[string]interface{}) (bccsp.BCCSP, error))
	if !ok {
		return nil, fmt.Errorf("Plugin does not implement the required function signature for 'New'")
	}

	return new(config.PluginOpts.Config)
}
