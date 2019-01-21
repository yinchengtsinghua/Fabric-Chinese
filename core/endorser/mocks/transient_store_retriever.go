
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//Mokery v1.0.0生成的代码
package mocks

import mock "github.com/stretchr/testify/mock"
import transientstore "github.com/hyperledger/fabric/core/transientstore"

//TransientStoreRetriever是TransientStoreRetriever类型的自动生成的模拟类型
type TransientStoreRetriever struct {
	mock.Mock
}

//storeforchannel为给定字段提供模拟函数：channel
func (_m *TransientStoreRetriever) StoreForChannel(channel string) transientstore.Store {
	ret := _m.Called(channel)

	var r0 transientstore.Store
	if rf, ok := ret.Get(0).(func(string) transientstore.Store); ok {
		r0 = rf(channel)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(transientstore.Store)
		}
	}

	return r0
}
