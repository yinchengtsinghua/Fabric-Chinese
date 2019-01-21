
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


package main

import (
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	"github.com/hyperledger/fabric/protos/common"
)

//noopvalidator用于测试验证插件基础结构
type NoOpValidator struct {
}

//验证使用给定数据验证事务
func (*NoOpValidator) Validate(_ *common.Block, _ string, _ int, _ int, _ ...validation.ContextDatum) error {
	return nil
}

//init用给定的依赖项初始化插件
func (*NoOpValidator) Init(dependencies ...validation.Dependency) error {
	return nil
}

//noopvalidatorfactory创建新的noopvalidators
type NoOpValidatorFactory struct {
}

//new返回noopvalidator的实例
func (*NoOpValidatorFactory) New() validation.Plugin {
	return &NoOpValidator{}
}

//验证插件框架调用NewPluginFactory以获取实例
//工厂的
func NewPluginFactory() validation.PluginFactory {
	return &NoOpValidatorFactory{}
}
