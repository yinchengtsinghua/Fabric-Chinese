
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//Mokery v1.0.0生成的代码
package mocks

import endorser "github.com/hyperledger/fabric/core/endorser"
import mock "github.com/stretchr/testify/mock"

//ChannelStateRetriever是ChannelStateRetriever类型的自动生成的模拟类型
type ChannelStateRetriever struct {
	mock.Mock
}

//newQueryCreator提供了一个具有给定字段的模拟函数：channel
func (_m *ChannelStateRetriever) NewQueryCreator(channel string) (endorser.QueryCreator, error) {
	ret := _m.Called(channel)

	var r0 endorser.QueryCreator
	if rf, ok := ret.Get(0).(func(string) endorser.QueryCreator); ok {
		r0 = rf(channel)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(endorser.QueryCreator)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(channel)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
