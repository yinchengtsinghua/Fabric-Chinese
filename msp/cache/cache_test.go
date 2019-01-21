
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


package cache

import (
	"sync"
	"testing"

	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mocks"
	msp2 "github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewCacheMsp(t *testing.T) {
	i, err := New(nil)
	assert.Error(t, err)
	assert.Nil(t, i)
	assert.Contains(t, err.Error(), "Invalid passed MSP. It must be different from nil.")

	i, err = New(&mocks.MockMSP{})
	assert.NoError(t, err)
	assert.NotNil(t, i)
}

func TestSetup(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

	mockMSP.On("Setup", (*msp2.MSPConfig)(nil)).Return(nil)
	err = i.Setup(nil)
	assert.NoError(t, err)
	mockMSP.AssertExpectations(t)
	assert.Equal(t, 0, i.(*cachedMSP).deserializeIdentityCache.len())
	assert.Equal(t, 0, i.(*cachedMSP).satisfiesPrincipalCache.len())
	assert.Equal(t, 0, i.(*cachedMSP).validateIdentityCache.len())
}

func TestGetType(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

	mockMSP.On("GetType").Return(msp.FABRIC)
	assert.Equal(t, msp.FABRIC, i.GetType())
	mockMSP.AssertExpectations(t)
}

func TestGetIdentifier(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

	mockMSP.On("GetIdentifier").Return("MSP", nil)
	id, err := i.GetIdentifier()
	assert.NoError(t, err)
	assert.Equal(t, "MSP", id)
	mockMSP.AssertExpectations(t)
}

func TestGetSigningIdentity(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

	mockIdentity := &mocks.MockSigningIdentity{Mock: mock.Mock{}, MockIdentity: &mocks.MockIdentity{ID: "Alice"}}
	identifier := &msp.IdentityIdentifier{Mspid: "MSP", Id: "Alice"}
	mockMSP.On("GetSigningIdentity", identifier).Return(mockIdentity, nil)
	id, err := i.GetSigningIdentity(identifier)
	assert.NoError(t, err)
	assert.Equal(t, mockIdentity, id)
	mockMSP.AssertExpectations(t)
}

func TestGetDefaultSigningIdentity(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

	mockIdentity := &mocks.MockSigningIdentity{Mock: mock.Mock{}, MockIdentity: &mocks.MockIdentity{ID: "Alice"}}
	mockMSP.On("GetDefaultSigningIdentity").Return(mockIdentity, nil)
	id, err := i.GetDefaultSigningIdentity()
	assert.NoError(t, err)
	assert.Equal(t, mockIdentity, id)
	mockMSP.AssertExpectations(t)
}

func TestGetTLSRootCerts(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

	expected := [][]byte{{1}, {2}}
	mockMSP.On("GetTLSRootCerts").Return(expected)
	certs := i.GetTLSRootCerts()
	assert.Equal(t, expected, certs)
}

func TestGetTLSIntermediateCerts(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

	expected := [][]byte{{1}, {2}}
	mockMSP.On("GetTLSIntermediateCerts").Return(expected)
	certs := i.GetTLSIntermediateCerts()
	assert.Equal(t, expected, certs)
}

func TestDeserializeIdentity(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	wrappedMSP, err := New(mockMSP)
	assert.NoError(t, err)

//已缓存检查ID
	mockIdentity := &mocks.MockIdentity{ID: "Alice"}
	mockIdentity2 := &mocks.MockIdentity{ID: "Bob"}
	serializedIdentity := []byte{1, 2, 3}
	serializedIdentity2 := []byte{4, 5, 6}
	mockMSP.On("DeserializeIdentity", serializedIdentity).Return(mockIdentity, nil)
	mockMSP.On("DeserializeIdentity", serializedIdentity2).Return(mockIdentity2, nil)
//主缓存
	wrappedMSP.DeserializeIdentity(serializedIdentity)
	wrappedMSP.DeserializeIdentity(serializedIdentity2)

//强调缓存并确保并发操作
//不会导致故障
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(m msp.MSP, i int) {
			sIdentity := serializedIdentity
			expectedIdentity := mockIdentity
			defer wg.Done()
			if i%2 == 0 {
				sIdentity = serializedIdentity2
				expectedIdentity = mockIdentity2
			}
			id, err := wrappedMSP.DeserializeIdentity(sIdentity)
			assert.NoError(t, err)
			assert.Equal(t, expectedIdentity, id.(*cachedIdentity).Identity)
		}(wrappedMSP, i)
	}
	wg.Wait()

	mockMSP.AssertExpectations(t)
//检查缓存
	_, ok := wrappedMSP.(*cachedMSP).deserializeIdentityCache.get(string(serializedIdentity))
	assert.True(t, ok)

//检查返回的对象是否相同
	id, err := wrappedMSP.DeserializeIdentity(serializedIdentity)
	assert.NoError(t, err)
	assert.True(t, mockIdentity == id.(*cachedIdentity).Identity)
	mockMSP.AssertExpectations(t)

//未缓存检查ID
	mockIdentity = &mocks.MockIdentity{ID: "Bob"}
	serializedIdentity = []byte{1, 2, 3, 4}
	mockMSP.On("DeserializeIdentity", serializedIdentity).Return(mockIdentity, errors.New("Invalid identity"))
	id, err = wrappedMSP.DeserializeIdentity(serializedIdentity)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid identity")
	mockMSP.AssertExpectations(t)

	_, ok = wrappedMSP.(*cachedMSP).deserializeIdentityCache.get(string(serializedIdentity))
	assert.False(t, ok)
}

func TestValidate(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

//已缓存检查验证
	mockIdentity := &mocks.MockIdentity{ID: "Alice"}
	mockIdentity.On("GetIdentifier").Return(&msp.IdentityIdentifier{Mspid: "MSP", Id: "Alice"})
	mockMSP.On("Validate", mockIdentity).Return(nil)
	err = i.Validate(mockIdentity)
	assert.NoError(t, err)
	mockIdentity.AssertExpectations(t)
	mockMSP.AssertExpectations(t)
//检查缓存
	identifier := mockIdentity.GetIdentifier()
	key := string(identifier.Mspid + ":" + identifier.Id)
	v, ok := i.(*cachedMSP).validateIdentityCache.get(string(key))
	assert.True(t, ok)
	assert.True(t, v.(bool))

//复查
	err = i.Validate(mockIdentity)
	assert.NoError(t, err)

//未缓存检查验证
	mockIdentity = &mocks.MockIdentity{ID: "Bob"}
	mockIdentity.On("GetIdentifier").Return(&msp.IdentityIdentifier{Mspid: "MSP", Id: "Bob"})
	mockMSP.On("Validate", mockIdentity).Return(errors.New("Invalid identity"))
	err = i.Validate(mockIdentity)
	assert.Error(t, err)
	mockIdentity.AssertExpectations(t)
	mockMSP.AssertExpectations(t)
//检查缓存
	identifier = mockIdentity.GetIdentifier()
	key = string(identifier.Mspid + ":" + identifier.Id)
	_, ok = i.(*cachedMSP).validateIdentityCache.get(string(key))
	assert.False(t, ok)
}

func TestSatisfiesValidateIndirectCall(t *testing.T) {
	mockMSP := &mocks.MockMSP{}

	mockIdentity := &mocks.MockIdentity{ID: "Alice"}
	mockIdentity.On("Validate").Run(func(_ mock.Arguments) {
		panic("shouldn't have invoked the identity method")
	})
	mockMSP.On("DeserializeIdentity", mock.Anything).Return(mockIdentity, nil).Once()
	mockIdentity.On("GetIdentifier").Return(&msp.IdentityIdentifier{Mspid: "MSP", Id: "Alice"})

	cache, err := New(mockMSP)
	assert.NoError(t, err)

	mockMSP.On("Validate", mockIdentity).Return(nil)

//测试缓存是否返回正确的值，并使用该值来启动缓存
	err = cache.Validate(mockIdentity)
	mockMSP.AssertNumberOfCalls(t, "Validate", 1)
	assert.NoError(t, err)
//获取我们测试缓存的身份
	identity, err := cache.DeserializeIdentity([]byte{1, 2, 3})
	assert.NoError(t, err)
//确保返回的标识回答缓存的MSP所回答的问题。
	err = identity.Validate()
	assert.NoError(t, err)
//确保尽管调用了一个验证调用，但未将该调用传递给支持的MSP
	mockMSP.AssertNumberOfCalls(t, "Validate", 1)
}

func TestSatisfiesPrincipalIndirectCall(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	mockMSPPrincipal := &msp2.MSPPrincipal{PrincipalClassification: msp2.MSPPrincipal_IDENTITY, Principal: []byte{1, 2, 3}}

	mockIdentity := &mocks.MockIdentity{ID: "Alice"}
	mockIdentity.On("SatisfiesPrincipal", mockMSPPrincipal).Run(func(_ mock.Arguments) {
		panic("shouldn't have invoked the identity method")
	})
	mockMSP.On("DeserializeIdentity", mock.Anything).Return(mockIdentity, nil).Once()
	mockIdentity.On("GetIdentifier").Return(&msp.IdentityIdentifier{Mspid: "MSP", Id: "Alice"})

	cache, err := New(mockMSP)
	assert.NoError(t, err)

//第一次调用satisfiesprincipal时返回错误
	mockMSP.On("SatisfiesPrincipal", mockIdentity, mockMSPPrincipal).Return(errors.New("error: foo")).Once()
//第二次调用返回nil
	mockMSP.On("SatisfiesPrincipal", mockIdentity, mockMSPPrincipal).Return(nil).Once()

//测试缓存是否返回正确的值
	err = cache.SatisfiesPrincipal(mockIdentity, mockMSPPrincipal)
	assert.Equal(t, "error: foo", err.Error())
//获取我们测试缓存的身份
	identity, err := cache.DeserializeIdentity([]byte{1, 2, 3})
	assert.NoError(t, err)
//确保返回的标识回答缓存的MSP所回答的问题。
//如果调用没有命中缓存，它将返回nil而不是错误。
	err = identity.SatisfiesPrincipal(mockMSPPrincipal)
	assert.Equal(t, "error: foo", err.Error())
}

func TestSatisfiesPrincipal(t *testing.T) {
	mockMSP := &mocks.MockMSP{}
	i, err := New(mockMSP)
	assert.NoError(t, err)

//已缓存检查验证
	mockIdentity := &mocks.MockIdentity{ID: "Alice"}
	mockIdentity.On("GetIdentifier").Return(&msp.IdentityIdentifier{Mspid: "MSP", Id: "Alice"})
	mockMSPPrincipal := &msp2.MSPPrincipal{PrincipalClassification: msp2.MSPPrincipal_IDENTITY, Principal: []byte{1, 2, 3}}
	mockMSP.On("SatisfiesPrincipal", mockIdentity, mockMSPPrincipal).Return(nil)
	mockMSP.SatisfiesPrincipal(mockIdentity, mockMSPPrincipal)
	err = i.SatisfiesPrincipal(mockIdentity, mockMSPPrincipal)
	assert.NoError(t, err)
	mockIdentity.AssertExpectations(t)
	mockMSP.AssertExpectations(t)
//检查缓存
	identifier := mockIdentity.GetIdentifier()
	identityKey := string(identifier.Mspid + ":" + identifier.Id)
	principalKey := string(mockMSPPrincipal.PrincipalClassification) + string(mockMSPPrincipal.Principal)
	key := identityKey + principalKey
	v, ok := i.(*cachedMSP).satisfiesPrincipalCache.get(key)
	assert.True(t, ok)
	assert.Nil(t, v)

//复查
	err = i.SatisfiesPrincipal(mockIdentity, mockMSPPrincipal)
	assert.NoError(t, err)

//未缓存检查验证
	mockIdentity = &mocks.MockIdentity{ID: "Bob"}
	mockIdentity.On("GetIdentifier").Return(&msp.IdentityIdentifier{Mspid: "MSP", Id: "Bob"})
	mockMSPPrincipal = &msp2.MSPPrincipal{PrincipalClassification: msp2.MSPPrincipal_IDENTITY, Principal: []byte{1, 2, 3, 4}}
	mockMSP.On("SatisfiesPrincipal", mockIdentity, mockMSPPrincipal).Return(errors.New("Invalid"))
	mockMSP.SatisfiesPrincipal(mockIdentity, mockMSPPrincipal)
	err = i.SatisfiesPrincipal(mockIdentity, mockMSPPrincipal)
	assert.Error(t, err)
	mockIdentity.AssertExpectations(t)
	mockMSP.AssertExpectations(t)
//检查缓存
	identifier = mockIdentity.GetIdentifier()
	identityKey = string(identifier.Mspid + ":" + identifier.Id)
	principalKey = string(mockMSPPrincipal.PrincipalClassification) + string(mockMSPPrincipal.Principal)
	key = identityKey + principalKey
	v, ok = i.(*cachedMSP).satisfiesPrincipalCache.get(key)
	assert.True(t, ok)
	assert.NotNil(t, v)
	assert.Contains(t, "Invalid", v.(error).Error())
}
