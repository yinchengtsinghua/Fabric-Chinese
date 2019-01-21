
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

import endorsement "github.com/hyperledger/fabric/core/handlers/endorsement/api/identities"
import mock "github.com/stretchr/testify/mock"
import peer "github.com/hyperledger/fabric/protos/peer"

//SigningIdentityFetcher是为SigningIdentityFetcher类型自动生成的模拟类型
type SigningIdentityFetcher struct {
	mock.Mock
}

//SigningIdentityForRequest提供了一个具有给定字段的模拟函数：a0
func (_m *SigningIdentityFetcher) SigningIdentityForRequest(_a0 *peer.SignedProposal) (endorsement.SigningIdentity, error) {
	ret := _m.Called(_a0)

	var r0 endorsement.SigningIdentity
	if rf, ok := ret.Get(0).(func(*peer.SignedProposal) endorsement.SigningIdentity); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(endorsement.SigningIdentity)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*peer.SignedProposal) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
