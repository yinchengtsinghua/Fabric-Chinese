
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


package discovery

import (
	"encoding/asn1"
	"errors"
	"sync"
	"testing"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSignedDataToKey(t *testing.T) {
	key1, err1 := signedDataToKey(common.SignedData{
		Data:      []byte{1, 2, 3, 4},
		Identity:  []byte{5, 6, 7},
		Signature: []byte{8, 9},
	})
	key2, err2 := signedDataToKey(common.SignedData{
		Data:      []byte{1, 2, 3},
		Identity:  []byte{4, 5, 6},
		Signature: []byte{7, 8, 9},
	})
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, key1, key2)
}

type mockAcSupport struct {
	mock.Mock
}

func (as *mockAcSupport) EligibleForService(channel string, data common.SignedData) error {
	return as.Called(channel, data).Error(0)
}

func (as *mockAcSupport) ConfigSequence(channel string) uint64 {
	return as.Called(channel).Get(0).(uint64)
}

func TestCacheDisabled(t *testing.T) {
	sd := common.SignedData{
		Data:      []byte{1, 2, 3},
		Identity:  []byte("authorizedIdentity"),
		Signature: []byte{1, 2, 3},
	}

	as := &mockAcSupport{}
	as.On("ConfigSequence", "foo").Return(uint64(0))
	as.On("EligibleForService", "foo", sd).Return(nil)
	cache := newAuthCache(as, authCacheConfig{maxCacheSize: 100, purgeRetentionRatio: 0.5})

//使用相同的参数调用缓存两次，并确保未缓存调用
	cache.EligibleForService("foo", sd)
	cache.EligibleForService("foo", sd)
	as.AssertNumberOfCalls(t, "EligibleForService", 2)
}

func TestCacheUsage(t *testing.T) {
	as := &mockAcSupport{}
	as.On("ConfigSequence", "foo").Return(uint64(0))
	as.On("ConfigSequence", "bar").Return(uint64(0))
	cache := newAuthCache(as, defaultConfig())

	sd1 := common.SignedData{
		Data:      []byte{1, 2, 3},
		Identity:  []byte("authorizedIdentity"),
		Signature: []byte{1, 2, 3},
	}

	sd2 := common.SignedData{
		Data:      []byte{1, 2, 3},
		Identity:  []byte("authorizedIdentity"),
		Signature: []byte{1, 2, 3},
	}

	sd3 := common.SignedData{
		Data:      []byte{1, 2, 3, 3},
		Identity:  []byte("unAuthorizedIdentity"),
		Signature: []byte{1, 2, 3},
	}

	testCases := []struct {
		channel     string
		expectedErr error
		sd          common.SignedData
	}{
		{
			sd:      sd1,
			channel: "foo",
		},
		{
			sd:      sd2,
			channel: "bar",
		},
		{
			channel:     "bar",
			sd:          sd3,
			expectedErr: errors.New("user revoked"),
		},
	}

	for _, tst := range testCases {
//场景一：调用没有缓存
		invoked := false
		as.On("EligibleForService", tst.channel, tst.sd).Return(tst.expectedErr).Once().Run(func(_ mock.Arguments) {
			invoked = true
		})
		t.Run("Not cached test", func(t *testing.T) {
			err := cache.EligibleForService(tst.channel, tst.sd)
			if tst.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tst.expectedErr.Error(), err.Error())
			}
			assert.True(t, invoked)
//为下次测试调用重置为false
			invoked = false
		})

//场景二：调用被缓存。
//我们不定义模拟调用，因为它应该与上次相同。
//如果不使用缓存，测试将失败，因为未定义模拟
		t.Run("Cached test", func(t *testing.T) {
			err := cache.EligibleForService(tst.channel, tst.sd)
			if tst.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tst.expectedErr.Error(), err.Error())
			}
			assert.False(t, invoked)
		})
	}
}

func TestCacheMarshalFailure(t *testing.T) {
	as := &mockAcSupport{}
	cache := newAuthCache(as, defaultConfig())
	asBytes = func(_ interface{}) ([]byte, error) {
		return nil, errors.New("failed marshaling ASN1")
	}
	defer func() {
		asBytes = asn1.Marshal
	}()
	err := cache.EligibleForService("mychannel", common.SignedData{})
	assert.Contains(t, err.Error(), "failed marshaling ASN1")
}

func TestCacheConfigChange(t *testing.T) {
	as := &mockAcSupport{}
	sd := common.SignedData{
		Data:      []byte{1, 2, 3},
		Identity:  []byte("identity"),
		Signature: []byte{1, 2, 3},
	}

	cache := newAuthCache(as, defaultConfig())

//场景一：首先，身份被授权
	as.On("EligibleForService", "mychannel", sd).Return(nil).Once()
	as.On("ConfigSequence", "mychannel").Return(uint64(0)).Times(2)
	err := cache.EligibleForService("mychannel", sd)
	assert.NoError(t, err)

//场景二：身份仍然被授权，配置还没有改变。
//应缓存结果
	as.On("ConfigSequence", "mychannel").Return(uint64(0)).Once()
	err = cache.EligibleForService("mychannel", sd)
	assert.NoError(t, err)

//场景三：发生配置更改，应忽略缓存
	as.On("ConfigSequence", "mychannel").Return(uint64(1)).Times(2)
	as.On("EligibleForService", "mychannel", sd).Return(errors.New("unauthorized")).Once()
	err = cache.EligibleForService("mychannel", sd)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestCachePurgeCache(t *testing.T) {
	as := &mockAcSupport{}
	cache := newAuthCache(as, authCacheConfig{maxCacheSize: 4, purgeRetentionRatio: 0.75, enabled: true})
	as.On("ConfigSequence", "mychannel").Return(uint64(0))

//预热缓存-尝试放置4个标识以填充缓存
	for _, id := range []string{"identity1", "identity2", "identity3", "identity4"} {
		sd := common.SignedData{
			Data:      []byte{1, 2, 3},
			Identity:  []byte(id),
			Signature: []byte{1, 2, 3},
		}
//首先，所有身份都符合该服务的条件。
		as.On("EligibleForService", "mychannel", sd).Return(nil).Once()
		err := cache.EligibleForService("mychannel", sd)
		assert.NoError(t, err)
	}

//现在，确保至少有1个标识从缓存中被逐出，但不是全部
	var evicted int
	for _, id := range []string{"identity5", "identity1", "identity2"} {
		sd := common.SignedData{
			Data:      []byte{1, 2, 3},
			Identity:  []byte(id),
			Signature: []byte{1, 2, 3},
		}
		as.On("EligibleForService", "mychannel", sd).Return(errors.New("unauthorized")).Once()
		err := cache.EligibleForService("mychannel", sd)
		if err != nil {
			evicted++
		}
	}
	assert.True(t, evicted > 0 && evicted < 4, "evicted: %d, but expected between 1 and 3 evictions", evicted)
}

func TestCacheConcurrentConfigUpdate(t *testing.T) {
//Scenario: 2 requests for the same identity are made concurrently.
//两者都没有缓存，因此它们的计算结果都可能进入缓存。
//当配置序列为0且配置更新时，第一个请求进入。
//这将在评估第一个请求的访问控制检查的同时撤销标识。
//第一个请求的计算由于调度而暂停，并在第二个请求之后完成，
//这发生在配置更新之后。
//第二个请求的计算结果不应被计算结果覆盖。
//尽管第一个请求的计算在第二个请求之后完成。

	as := &mockAcSupport{}
	sd := common.SignedData{
		Data:      []byte{1, 2, 3},
		Identity:  []byte{1, 2, 3},
		Signature: []byte{1, 2, 3},
	}
	var firstRequestInvoked sync.WaitGroup
	firstRequestInvoked.Add(1)
	var firstRequestFinished sync.WaitGroup
	firstRequestFinished.Add(1)
	var secondRequestFinished sync.WaitGroup
	secondRequestFinished.Add(1)
	cache := newAuthCache(as, defaultConfig())
//首先，身份是合格的。
	as.On("EligibleForService", "mychannel", mock.Anything).Return(nil).Once().Run(func(_ mock.Arguments) {
		firstRequestInvoked.Done()
		secondRequestFinished.Wait()
	})
//但是在配置更改之后，它不是
	as.On("EligibleForService", "mychannel", mock.Anything).Return(errors.New("unauthorized")).Once()
//第一个请求看到的配置序列
	as.On("ConfigSequence", "mychannel").Return(uint64(0)).Once()
	as.On("ConfigSequence", "mychannel").Return(uint64(1)).Once()
//第二个请求看到的配置序列
	as.On("ConfigSequence", "mychannel").Return(uint64(1)).Times(2)

//第一个请求返回OK
	go func() {
		defer firstRequestFinished.Done()
		firstResult := cache.EligibleForService("mychannel", sd)
		assert.NoError(t, firstResult)
	}()
	firstRequestInvoked.Wait()
//第二个请求返回身份未经授权
	secondResult := cache.EligibleForService("mychannel", sd)
//将第二个请求标记为“完成”，以向第一个请求发出信号，以完成其计算。
	secondRequestFinished.Done()
//等待第一个请求返回
	firstRequestFinished.Wait()
	assert.Contains(t, secondResult.Error(), "unauthorized")

//现在再提出一个请求并确保缓存第二个请求的结果（授权的结果），
//即使它在第一次请求之前完成。
	as.On("ConfigSequence", "mychannel").Return(uint64(1)).Once()
	cachedResult := cache.EligibleForService("mychannel", sd)
	assert.Contains(t, cachedResult.Error(), "unauthorized")
}

func defaultConfig() authCacheConfig {
	return authCacheConfig{maxCacheSize: defaultMaxCacheSize, purgeRetentionRatio: defaultRetentionRatio, enabled: true}
}
