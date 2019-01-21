
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

	grpclogging "github.com/hyperledger/fabric/common/grpclogging"
	zapcore "go.uber.org/zap/zapcore"
)

type Leveler struct {
	Stub        func(context.Context, string) zapcore.Level
	mutex       sync.RWMutex
	argsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	returns struct {
		result1 zapcore.Level
	}
	returnsOnCall map[int]struct {
		result1 zapcore.Level
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Leveler) Spy(arg1 context.Context, arg2 string) zapcore.Level {
	fake.mutex.Lock()
	ret, specificReturn := fake.returnsOnCall[len(fake.argsForCall)]
	fake.argsForCall = append(fake.argsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("LevelerFunc", []interface{}{arg1, arg2})
	fake.mutex.Unlock()
	if fake.Stub != nil {
		return fake.Stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.returns.result1
}

func (fake *Leveler) CallCount() int {
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	return len(fake.argsForCall)
}

func (fake *Leveler) Calls(stub func(context.Context, string) zapcore.Level) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = stub
}

func (fake *Leveler) ArgsForCall(i int) (context.Context, string) {
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	return fake.argsForCall[i].arg1, fake.argsForCall[i].arg2
}

func (fake *Leveler) Returns(result1 zapcore.Level) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = nil
	fake.returns = struct {
		result1 zapcore.Level
	}{result1}
}

func (fake *Leveler) ReturnsOnCall(i int, result1 zapcore.Level) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = nil
	if fake.returnsOnCall == nil {
		fake.returnsOnCall = make(map[int]struct {
			result1 zapcore.Level
		})
	}
	fake.returnsOnCall[i] = struct {
		result1 zapcore.Level
	}{result1}
}

func (fake *Leveler) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Leveler) recordInvocation(key string, args []interface{}) {
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

var _ grpclogging.LevelerFunc = new(Leveler).Spy
