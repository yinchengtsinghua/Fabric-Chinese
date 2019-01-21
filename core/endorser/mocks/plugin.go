
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
import peer "github.com/hyperledger/fabric/protos/peer"

//插件是插件类型的自动生成的模拟类型
type Plugin struct {
	mock.Mock
}

//认可提供了一个具有给定字段的模拟函数：有效载荷，sp
func (_m *Plugin) Endorse(payload []byte, sp *peer.SignedProposal) (*peer.Endorsement, []byte, error) {
	ret := _m.Called(payload, sp)

	var r0 *peer.Endorsement
	if rf, ok := ret.Get(0).(func([]byte, *peer.SignedProposal) *peer.Endorsement); ok {
		r0 = rf(payload, sp)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*peer.Endorsement)
		}
	}

	var r1 []byte
	if rf, ok := ret.Get(1).(func([]byte, *peer.SignedProposal) []byte); ok {
		r1 = rf(payload, sp)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func([]byte, *peer.SignedProposal) error); ok {
		r2 = rf(payload, sp)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

//init提供了一个具有给定字段的模拟函数：依赖项
func (_m *Plugin) Init(dependencies ...endorsement.Dependency) error {
	_va := make([]interface{}, len(dependencies))
	for _i := range dependencies {
		_va[_i] = dependencies[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(...endorsement.Dependency) error); ok {
		r0 = rf(dependencies...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
