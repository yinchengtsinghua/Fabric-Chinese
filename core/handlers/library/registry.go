
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp，SecureKey Technologies Inc.保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package library

import (
	"fmt"
	"os"
	"plugin"
	"reflect"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/handlers/auth"
	"github.com/hyperledger/fabric/core/handlers/decoration"
	endorsement2 "github.com/hyperledger/fabric/core/handlers/endorsement/api"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
)

var logger = flogging.MustGetLogger("core.handlers")

//注册表定义查找的对象
//按名称处理程序
type Registry interface {
//查找返回具有给定
//注册名称，如果不存在则为零
	Lookup(HandlerType) interface{}
}

//handlerType定义可以筛选和变异的自定义处理程序
//在对等机中传递的对象
type HandlerType int

const (
//授权处理程序-拒绝或转发来自客户端的建议
	Auth HandlerType = iota
//装饰处理程序-附加或修改链码输入
//传递到链码
	Decoration
	Endorsement
	Validation

	authPluginFactory      = "NewFilter"
	decoratorPluginFactory = "NewDecorator"
	pluginFactory          = "NewPluginFactory"
)

type registry struct {
	filters    []auth.Filter
	decorators []decoration.Decorator
	endorsers  map[string]endorsement2.PluginFactory
	validators map[string]validation.PluginFactory
}

var once sync.Once
var reg registry

//配置配置工厂方法
//和注册表插件
type Config struct {
	AuthFilters []*HandlerConfig `mapstructure:"authFilters" yaml:"authFilters"`
	Decorators  []*HandlerConfig `mapstructure:"decorators" yaml:"decorators"`
	Endorsers   PluginMapping    `mapstructure:"endorsers" yaml:"endorsers"`
	Validators  PluginMapping    `mapstructure:"validators" yaml:"validators"`
}

type PluginMapping map[string]*HandlerConfig

//handlerconfig定义插件或已编译处理程序的配置
type HandlerConfig struct {
	Name    string `mapstructure:"name" yaml:"name"`
	Library string `mapstructure:"library" yaml:"library"`
}

//InitRegistry创建（唯一）实例
//注册表
func InitRegistry(c Config) Registry {
	once.Do(func() {
		reg = registry{
			endorsers:  make(map[string]endorsement2.PluginFactory),
			validators: make(map[string]validation.PluginFactory),
		}
		reg.loadHandlers(c)
	})
	return &reg
}

//加载处理程序加载配置的处理程序
func (r *registry) loadHandlers(c Config) {
	for _, config := range c.AuthFilters {
		r.evaluateModeAndLoad(config, Auth)
	}
	for _, config := range c.Decorators {
		r.evaluateModeAndLoad(config, Decoration)
	}

	for chaincodeID, config := range c.Endorsers {
		r.evaluateModeAndLoad(config, Endorsement, chaincodeID)
	}

	for chaincodeID, config := range c.Validators {
		r.evaluateModeAndLoad(config, Validation, chaincodeID)
	}
}

//评估模式和加载如果提供了库路径，则加载共享对象
func (r *registry) evaluateModeAndLoad(c *HandlerConfig, handlerType HandlerType, extraArgs ...string) {
	if c.Library != "" {
		r.loadPlugin(c.Library, handlerType, extraArgs...)
	} else {
		r.loadCompiled(c.Name, handlerType, extraArgs...)
	}
}

//LoadCompiled加载静态编译的处理程序
func (r *registry) loadCompiled(handlerFactory string, handlerType HandlerType, extraArgs ...string) {
	registryMD := reflect.ValueOf(&HandlerLibrary{})

	o := registryMD.MethodByName(handlerFactory)
	if !o.IsValid() {
		logger.Panicf(fmt.Sprintf("Method %s isn't a method of HandlerLibrary", handlerFactory))
	}

	inst := o.Call(nil)[0].Interface()

	if handlerType == Auth {
		r.filters = append(r.filters, inst.(auth.Filter))
	} else if handlerType == Decoration {
		r.decorators = append(r.decorators, inst.(decoration.Decorator))
	} else if handlerType == Endorsement {
		if len(extraArgs) != 1 {
			logger.Panicf("expected 1 argument in extraArgs")
		}
		r.endorsers[extraArgs[0]] = inst.(endorsement2.PluginFactory)
	} else if handlerType == Validation {
		if len(extraArgs) != 1 {
			logger.Panicf("expected 1 argument in extraArgs")
		}
		r.validators[extraArgs[0]] = inst.(validation.PluginFactory)
	}
}

//LoadPlugin加载Pluggable处理程序
func (r *registry) loadPlugin(pluginPath string, handlerType HandlerType, extraArgs ...string) {
	if _, err := os.Stat(pluginPath); err != nil {
		logger.Panicf(fmt.Sprintf("Could not find plugin at path %s: %s", pluginPath, err))
	}
	p, err := plugin.Open(pluginPath)
	if err != nil {
		logger.Panicf(fmt.Sprintf("Error opening plugin at path %s: %s", pluginPath, err))
	}

	if handlerType == Auth {
		r.initAuthPlugin(p)
	} else if handlerType == Decoration {
		r.initDecoratorPlugin(p)
	} else if handlerType == Endorsement {
		r.initEndorsementPlugin(p, extraArgs...)
	} else if handlerType == Validation {
		r.initValidationPlugin(p, extraArgs...)
	}
}

//initauthplugin从给定的插件构造身份验证筛选器
func (r *registry) initAuthPlugin(p *plugin.Plugin) {
	constructorSymbol, err := p.Lookup(authPluginFactory)
	if err != nil {
		panicWithLookupError(authPluginFactory, err)
	}
	constructor, ok := constructorSymbol.(func() auth.Filter)
	if !ok {
		panicWithDefinitionError(authPluginFactory)
	}

	filter := constructor()
	if filter != nil {
		r.filters = append(r.filters, filter)
	}
}

//initdecoratorPlugin从给定的插件构造一个decorator
func (r *registry) initDecoratorPlugin(p *plugin.Plugin) {
	constructorSymbol, err := p.Lookup(decoratorPluginFactory)
	if err != nil {
		panicWithLookupError(decoratorPluginFactory, err)
	}
	constructor, ok := constructorSymbol.(func() decoration.Decorator)
	if !ok {
		panicWithDefinitionError(decoratorPluginFactory)
	}
	decorator := constructor()
	if decorator != nil {
		r.decorators = append(r.decorators, constructor())
	}
}

func (r *registry) initEndorsementPlugin(p *plugin.Plugin, extraArgs ...string) {
	if len(extraArgs) != 1 {
		logger.Panicf("expected 1 argument in extraArgs")
	}
	factorySymbol, err := p.Lookup(pluginFactory)
	if err != nil {
		panicWithLookupError(pluginFactory, err)
	}

	constructor, ok := factorySymbol.(func() endorsement2.PluginFactory)
	if !ok {
		panicWithDefinitionError(pluginFactory)
	}
	factory := constructor()
	if factory == nil {
		logger.Panicf("factory instance returned nil")
	}
	r.endorsers[extraArgs[0]] = factory
}

func (r *registry) initValidationPlugin(p *plugin.Plugin, extraArgs ...string) {
	if len(extraArgs) != 1 {
		logger.Panicf("expected 1 argument in extraArgs")
	}
	factorySymbol, err := p.Lookup(pluginFactory)
	if err != nil {
		panicWithLookupError(pluginFactory, err)
	}

	constructor, ok := factorySymbol.(func() validation.PluginFactory)
	if !ok {
		panicWithDefinitionError(pluginFactory)
	}
	factory := constructor()
	if factory == nil {
		logger.Panicf("factory instance returned nil")
	}
	r.validators[extraArgs[0]] = factory
}

//当处理程序构造函数查找失败时，PanicWithLookupError将暂停
func panicWithLookupError(factory string, err error) {
	logger.Panicf(fmt.Sprintf("Plugin must contain constructor with name %s. Error from lookup: %s",
		factory, err))
}

//处理程序构造函数不匹配时的panicWithDefinitionError panics
//预期函数定义
func panicWithDefinitionError(factory string) {
	logger.Panicf(fmt.Sprintf("Constructor method %s does not match expected definition",
		factory))
}

//查找返回具有给定
//给定类型，如果不存在则为零
func (r *registry) Lookup(handlerType HandlerType) interface{} {
	if handlerType == Auth {
		return r.filters
	} else if handlerType == Decoration {
		return r.decorators
	} else if handlerType == Endorsement {
		return r.endorsers
	} else if handlerType == Validation {
		return r.validators
	}

	return nil
}
