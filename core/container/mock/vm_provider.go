
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//伪造者生成的代码。不要编辑。
package mock

import (
	sync "sync"

	container "github.com/hyperledger/fabric/core/container"
)

type VMProvider struct {
	NewVMStub        func() container.VM
	newVMMutex       sync.RWMutex
	newVMArgsForCall []struct {
	}
	newVMReturns struct {
		result1 container.VM
	}
	newVMReturnsOnCall map[int]struct {
		result1 container.VM
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *VMProvider) NewVM() container.VM {
	fake.newVMMutex.Lock()
	ret, specificReturn := fake.newVMReturnsOnCall[len(fake.newVMArgsForCall)]
	fake.newVMArgsForCall = append(fake.newVMArgsForCall, struct {
	}{})
	fake.recordInvocation("NewVM", []interface{}{})
	fake.newVMMutex.Unlock()
	if fake.NewVMStub != nil {
		return fake.NewVMStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.newVMReturns
	return fakeReturns.result1
}

func (fake *VMProvider) NewVMCallCount() int {
	fake.newVMMutex.RLock()
	defer fake.newVMMutex.RUnlock()
	return len(fake.newVMArgsForCall)
}

func (fake *VMProvider) NewVMCalls(stub func() container.VM) {
	fake.newVMMutex.Lock()
	defer fake.newVMMutex.Unlock()
	fake.NewVMStub = stub
}

func (fake *VMProvider) NewVMReturns(result1 container.VM) {
	fake.newVMMutex.Lock()
	defer fake.newVMMutex.Unlock()
	fake.NewVMStub = nil
	fake.newVMReturns = struct {
		result1 container.VM
	}{result1}
}

func (fake *VMProvider) NewVMReturnsOnCall(i int, result1 container.VM) {
	fake.newVMMutex.Lock()
	defer fake.newVMMutex.Unlock()
	fake.NewVMStub = nil
	if fake.newVMReturnsOnCall == nil {
		fake.newVMReturnsOnCall = make(map[int]struct {
			result1 container.VM
		})
	}
	fake.newVMReturnsOnCall[i] = struct {
		result1 container.VM
	}{result1}
}

func (fake *VMProvider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.newVMMutex.RLock()
	defer fake.newVMMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *VMProvider) recordInvocation(key string, args []interface{}) {
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
