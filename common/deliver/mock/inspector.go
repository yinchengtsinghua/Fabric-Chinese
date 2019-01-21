
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
	context "context"
	sync "sync"

	proto "github.com/golang/protobuf/proto"
	deliver "github.com/hyperledger/fabric/common/deliver"
)

type Inspector struct {
	InspectStub        func(context.Context, proto.Message) error
	inspectMutex       sync.RWMutex
	inspectArgsForCall []struct {
		arg1 context.Context
		arg2 proto.Message
	}
	inspectReturns struct {
		result1 error
	}
	inspectReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Inspector) Inspect(arg1 context.Context, arg2 proto.Message) error {
	fake.inspectMutex.Lock()
	ret, specificReturn := fake.inspectReturnsOnCall[len(fake.inspectArgsForCall)]
	fake.inspectArgsForCall = append(fake.inspectArgsForCall, struct {
		arg1 context.Context
		arg2 proto.Message
	}{arg1, arg2})
	fake.recordInvocation("Inspect", []interface{}{arg1, arg2})
	fake.inspectMutex.Unlock()
	if fake.InspectStub != nil {
		return fake.InspectStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.inspectReturns
	return fakeReturns.result1
}

func (fake *Inspector) InspectCallCount() int {
	fake.inspectMutex.RLock()
	defer fake.inspectMutex.RUnlock()
	return len(fake.inspectArgsForCall)
}

func (fake *Inspector) InspectCalls(stub func(context.Context, proto.Message) error) {
	fake.inspectMutex.Lock()
	defer fake.inspectMutex.Unlock()
	fake.InspectStub = stub
}

func (fake *Inspector) InspectArgsForCall(i int) (context.Context, proto.Message) {
	fake.inspectMutex.RLock()
	defer fake.inspectMutex.RUnlock()
	argsForCall := fake.inspectArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *Inspector) InspectReturns(result1 error) {
	fake.inspectMutex.Lock()
	defer fake.inspectMutex.Unlock()
	fake.InspectStub = nil
	fake.inspectReturns = struct {
		result1 error
	}{result1}
}

func (fake *Inspector) InspectReturnsOnCall(i int, result1 error) {
	fake.inspectMutex.Lock()
	defer fake.inspectMutex.Unlock()
	fake.InspectStub = nil
	if fake.inspectReturnsOnCall == nil {
		fake.inspectReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.inspectReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *Inspector) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.inspectMutex.RLock()
	defer fake.inspectMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Inspector) recordInvocation(key string, args []interface{}) {
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

var _ deliver.Inspector = new(Inspector)
