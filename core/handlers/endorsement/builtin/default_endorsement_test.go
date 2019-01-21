
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


package builtin_test

import (
	"testing"

	"github.com/hyperledger/fabric/core/endorser/mocks"
	"github.com/hyperledger/fabric/core/handlers/endorsement/builtin"
	mocks2 "github.com/hyperledger/fabric/core/handlers/endorsement/builtin/mocks"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefaultEndorsement(t *testing.T) {
	factory := &builtin.DefaultEndorsementFactory{}
	endorser := factory.New()

//场景一：不要传递任何依赖项，并观察初始化失败
	err := endorser.Init()
	assert.Equal(t, "could not find SigningIdentityFetcher in dependencies", err.Error())

//场景二：传入init a signingIdentityFetcher
	sif := &mocks.SigningIdentityFetcher{}
//同时传递另一项以确保忽略它
	err = endorser.Init("foo", sif)
	assert.NoError(t, err)

//场景三：获取签名身份失败
	sif.On("SigningIdentityForRequest", mock.Anything).Return(nil, errors.New("foo")).Once()
	_, _, err = endorser.Endorse(nil, nil)
	assert.Contains(t, err.Error(), "foo")

//场景四：获取签名标识成功，但将标识序列化失败
	sid := &mocks2.SigningIdentity{}
	sid.On("Serialize").Return(nil, errors.New("bar")).Once()
	sif.On("SigningIdentityForRequest", mock.Anything).Return(sid, nil)
	_, _, err = endorser.Endorse(nil, nil)
	assert.Contains(t, err.Error(), "bar")

//场景五：序列化身份成功，但签名失败
	sid.On("Serialize").Return([]byte{1, 2, 3}, nil)
	sid.On("Sign", mock.Anything).Return(nil, errors.New("baz")).Once()
	_, _, err = endorser.Endorse([]byte{1, 1, 1, 1, 1}, nil)
	assert.Contains(t, err.Error(), "baz")

//场景六：签名成功
	sid.On("Serialize").Return([]byte{1, 2, 3}, nil)
	sid.On("Sign", mock.Anything).Return([]byte{10, 20, 30}, nil).Once()
	endorsement, resp, err := endorser.Endorse([]byte{1, 1, 1, 1, 1}, nil)
	assert.NoError(t, err)
	assert.Equal(t, resp, []byte{1, 1, 1, 1, 1})
	assert.Equal(t, &peer.Endorsement{
		Signature: []byte{10, 20, 30},
		Endorser:  []byte{1, 2, 3},
	}, endorsement)
}
