
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//伪造者生成的代码。不要编辑。
package fake

import (
	sync "sync"

	channelconfig "github.com/hyperledger/fabric/common/channelconfig"
)

type ApplicationConfigRetriever struct {
	GetApplicationConfigStub        func(string) (channelconfig.Application, bool)
	getApplicationConfigMutex       sync.RWMutex
	getApplicationConfigArgsForCall []struct {
		arg1 string
	}
	getApplicationConfigReturns struct {
		result1 channelconfig.Application
		result2 bool
	}
	getApplicationConfigReturnsOnCall map[int]struct {
		result1 channelconfig.Application
		result2 bool
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *ApplicationConfigRetriever) GetApplicationConfig(arg1 string) (channelconfig.Application, bool) {
	fake.getApplicationConfigMutex.Lock()
	ret, specificReturn := fake.getApplicationConfigReturnsOnCall[len(fake.getApplicationConfigArgsForCall)]
	fake.getApplicationConfigArgsForCall = append(fake.getApplicationConfigArgsForCall, struct {
		arg1 string
	}{arg1})
	fake.recordInvocation("GetApplicationConfig", []interface{}{arg1})
	fake.getApplicationConfigMutex.Unlock()
	if fake.GetApplicationConfigStub != nil {
		return fake.GetApplicationConfigStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getApplicationConfigReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ApplicationConfigRetriever) GetApplicationConfigCallCount() int {
	fake.getApplicationConfigMutex.RLock()
	defer fake.getApplicationConfigMutex.RUnlock()
	return len(fake.getApplicationConfigArgsForCall)
}

func (fake *ApplicationConfigRetriever) GetApplicationConfigCalls(stub func(string) (channelconfig.Application, bool)) {
	fake.getApplicationConfigMutex.Lock()
	defer fake.getApplicationConfigMutex.Unlock()
	fake.GetApplicationConfigStub = stub
}

func (fake *ApplicationConfigRetriever) GetApplicationConfigArgsForCall(i int) string {
	fake.getApplicationConfigMutex.RLock()
	defer fake.getApplicationConfigMutex.RUnlock()
	argsForCall := fake.getApplicationConfigArgsForCall[i]
	return argsForCall.arg1
}

func (fake *ApplicationConfigRetriever) GetApplicationConfigReturns(result1 channelconfig.Application, result2 bool) {
	fake.getApplicationConfigMutex.Lock()
	defer fake.getApplicationConfigMutex.Unlock()
	fake.GetApplicationConfigStub = nil
	fake.getApplicationConfigReturns = struct {
		result1 channelconfig.Application
		result2 bool
	}{result1, result2}
}

func (fake *ApplicationConfigRetriever) GetApplicationConfigReturnsOnCall(i int, result1 channelconfig.Application, result2 bool) {
	fake.getApplicationConfigMutex.Lock()
	defer fake.getApplicationConfigMutex.Unlock()
	fake.GetApplicationConfigStub = nil
	if fake.getApplicationConfigReturnsOnCall == nil {
		fake.getApplicationConfigReturnsOnCall = make(map[int]struct {
			result1 channelconfig.Application
			result2 bool
		})
	}
	fake.getApplicationConfigReturnsOnCall[i] = struct {
		result1 channelconfig.Application
		result2 bool
	}{result1, result2}
}

func (fake *ApplicationConfigRetriever) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getApplicationConfigMutex.RLock()
	defer fake.getApplicationConfigMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *ApplicationConfigRetriever) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}
