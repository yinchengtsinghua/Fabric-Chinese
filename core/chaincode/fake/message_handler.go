
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

	chaincode "github.com/hyperledger/fabric/core/chaincode"
	peer "github.com/hyperledger/fabric/protos/peer"
)

type MessageHandler struct {
	HandleStub        func(*peer.ChaincodeMessage, *chaincode.TransactionContext) (*peer.ChaincodeMessage, error)
	handleMutex       sync.RWMutex
	handleArgsForCall []struct {
		arg1 *peer.ChaincodeMessage
		arg2 *chaincode.TransactionContext
	}
	handleReturns struct {
		result1 *peer.ChaincodeMessage
		result2 error
	}
	handleReturnsOnCall map[int]struct {
		result1 *peer.ChaincodeMessage
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *MessageHandler) Handle(arg1 *peer.ChaincodeMessage, arg2 *chaincode.TransactionContext) (*peer.ChaincodeMessage, error) {
	fake.handleMutex.Lock()
	ret, specificReturn := fake.handleReturnsOnCall[len(fake.handleArgsForCall)]
	fake.handleArgsForCall = append(fake.handleArgsForCall, struct {
		arg1 *peer.ChaincodeMessage
		arg2 *chaincode.TransactionContext
	}{arg1, arg2})
	fake.recordInvocation("Handle", []interface{}{arg1, arg2})
	fake.handleMutex.Unlock()
	if fake.HandleStub != nil {
		return fake.HandleStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.handleReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *MessageHandler) HandleCallCount() int {
	fake.handleMutex.RLock()
	defer fake.handleMutex.RUnlock()
	return len(fake.handleArgsForCall)
}

func (fake *MessageHandler) HandleCalls(stub func(*peer.ChaincodeMessage, *chaincode.TransactionContext) (*peer.ChaincodeMessage, error)) {
	fake.handleMutex.Lock()
	defer fake.handleMutex.Unlock()
	fake.HandleStub = stub
}

func (fake *MessageHandler) HandleArgsForCall(i int) (*peer.ChaincodeMessage, *chaincode.TransactionContext) {
	fake.handleMutex.RLock()
	defer fake.handleMutex.RUnlock()
	argsForCall := fake.handleArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *MessageHandler) HandleReturns(result1 *peer.ChaincodeMessage, result2 error) {
	fake.handleMutex.Lock()
	defer fake.handleMutex.Unlock()
	fake.HandleStub = nil
	fake.handleReturns = struct {
		result1 *peer.ChaincodeMessage
		result2 error
	}{result1, result2}
}

func (fake *MessageHandler) HandleReturnsOnCall(i int, result1 *peer.ChaincodeMessage, result2 error) {
	fake.handleMutex.Lock()
	defer fake.handleMutex.Unlock()
	fake.HandleStub = nil
	if fake.handleReturnsOnCall == nil {
		fake.handleReturnsOnCall = make(map[int]struct {
			result1 *peer.ChaincodeMessage
			result2 error
		})
	}
	fake.handleReturnsOnCall[i] = struct {
		result1 *peer.ChaincodeMessage
		result2 error
	}{result1, result2}
}

func (fake *MessageHandler) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.handleMutex.RLock()
	defer fake.handleMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *MessageHandler) recordInvocation(key string, args []interface{}) {
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
