
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//Mokery v1.0.0生成的代码
package mocks

import api "github.com/hyperledger/fabric/gossip/api"
import common "github.com/hyperledger/fabric/gossip/common"

import gossipdiscovery "github.com/hyperledger/fabric/gossip/discovery"
import mock "github.com/stretchr/testify/mock"

//gossipsupport是为gossipsupport类型自动生成的模拟类型
type GossipSupport struct {
	mock.Mock
}

//channelexists为给定字段提供模拟函数：channel
func (_m *GossipSupport) ChannelExists(channel string) bool {
	ret := _m.Called(channel)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(channel)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

//IdentityInfo提供具有给定字段的模拟函数：
func (_m *GossipSupport) IdentityInfo() api.PeerIdentitySet {
	ret := _m.Called()

	var r0 api.PeerIdentitySet
	if rf, ok := ret.Get(0).(func() api.PeerIdentitySet); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.PeerIdentitySet)
		}
	}

	return r0
}

//对等端为给定字段提供模拟函数：
func (_m *GossipSupport) Peers() gossipdiscovery.Members {
	ret := _m.Called()

	var r0 gossipdiscovery.Members
	if rf, ok := ret.Get(0).(func() gossipdiscovery.Members); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(gossipdiscovery.Members)
		}
	}

	return r0
}

//peersofchannel提供了一个具有给定字段的模拟函数：a0
func (_m *GossipSupport) PeersOfChannel(_a0 common.ChainID) gossipdiscovery.Members {
	ret := _m.Called(_a0)

	var r0 gossipdiscovery.Members
	if rf, ok := ret.Get(0).(func(common.ChainID) gossipdiscovery.Members); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(gossipdiscovery.Members)
		}
	}

	return r0
}
