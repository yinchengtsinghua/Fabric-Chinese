
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


//Mokery v1.0.0生成的代码
package mocks

import endorsement "github.com/hyperledger/fabric/core/handlers/endorsement/api"
import mock "github.com/stretchr/testify/mock"

//PlugInfectory是PlugInfectory类型的自动生成的模拟类型
type PluginFactory struct {
	mock.Mock
}

//new为给定字段提供模拟函数：
func (_m *PluginFactory) New() endorsement.Plugin {
	ret := _m.Called()

	var r0 endorsement.Plugin
	if rf, ok := ret.Get(0).(func() endorsement.Plugin); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(endorsement.Plugin)
		}
	}

	return r0
}
