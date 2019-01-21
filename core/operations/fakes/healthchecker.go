
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//伪造者生成的代码。不要编辑。
package fakes

import (
	context "context"
	sync "sync"
)

type HealthChecker struct {
	HealthCheckStub        func(context.Context) error
	healthCheckMutex       sync.RWMutex
	healthCheckArgsForCall []struct {
		arg1 context.Context
	}
	healthCheckReturns struct {
		result1 error
	}
	healthCheckReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *HealthChecker) HealthCheck(arg1 context.Context) error {
	fake.healthCheckMutex.Lock()
	ret, specificReturn := fake.healthCheckReturnsOnCall[len(fake.healthCheckArgsForCall)]
	fake.healthCheckArgsForCall = append(fake.healthCheckArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	fake.recordInvocation("HealthCheck", []interface{}{arg1})
	fake.healthCheckMutex.Unlock()
	if fake.HealthCheckStub != nil {
		return fake.HealthCheckStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.healthCheckReturns
	return fakeReturns.result1
}

func (fake *HealthChecker) HealthCheckCallCount() int {
	fake.healthCheckMutex.RLock()
	defer fake.healthCheckMutex.RUnlock()
	return len(fake.healthCheckArgsForCall)
}

func (fake *HealthChecker) HealthCheckCalls(stub func(context.Context) error) {
	fake.healthCheckMutex.Lock()
	defer fake.healthCheckMutex.Unlock()
	fake.HealthCheckStub = stub
}

func (fake *HealthChecker) HealthCheckArgsForCall(i int) context.Context {
	fake.healthCheckMutex.RLock()
	defer fake.healthCheckMutex.RUnlock()
	argsForCall := fake.healthCheckArgsForCall[i]
	return argsForCall.arg1
}

func (fake *HealthChecker) HealthCheckReturns(result1 error) {
	fake.healthCheckMutex.Lock()
	defer fake.healthCheckMutex.Unlock()
	fake.HealthCheckStub = nil
	fake.healthCheckReturns = struct {
		result1 error
	}{result1}
}

func (fake *HealthChecker) HealthCheckReturnsOnCall(i int, result1 error) {
	fake.healthCheckMutex.Lock()
	defer fake.healthCheckMutex.Unlock()
	fake.HealthCheckStub = nil
	if fake.healthCheckReturnsOnCall == nil {
		fake.healthCheckReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.healthCheckReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *HealthChecker) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.healthCheckMutex.RLock()
	defer fake.healthCheckMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *HealthChecker) recordInvocation(key string, args []interface{}) {
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
