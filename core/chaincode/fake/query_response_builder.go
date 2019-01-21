
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

	ledger "github.com/hyperledger/fabric/common/ledger"
	chaincode "github.com/hyperledger/fabric/core/chaincode"
	peer "github.com/hyperledger/fabric/protos/peer"
)

type QueryResponseBuilder struct {
	BuildQueryResponseStub        func(*chaincode.TransactionContext, ledger.ResultsIterator, string, bool, int32) (*peer.QueryResponse, error)
	buildQueryResponseMutex       sync.RWMutex
	buildQueryResponseArgsForCall []struct {
		arg1 *chaincode.TransactionContext
		arg2 ledger.ResultsIterator
		arg3 string
		arg4 bool
		arg5 int32
	}
	buildQueryResponseReturns struct {
		result1 *peer.QueryResponse
		result2 error
	}
	buildQueryResponseReturnsOnCall map[int]struct {
		result1 *peer.QueryResponse
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *QueryResponseBuilder) BuildQueryResponse(arg1 *chaincode.TransactionContext, arg2 ledger.ResultsIterator, arg3 string, arg4 bool, arg5 int32) (*peer.QueryResponse, error) {
	fake.buildQueryResponseMutex.Lock()
	ret, specificReturn := fake.buildQueryResponseReturnsOnCall[len(fake.buildQueryResponseArgsForCall)]
	fake.buildQueryResponseArgsForCall = append(fake.buildQueryResponseArgsForCall, struct {
		arg1 *chaincode.TransactionContext
		arg2 ledger.ResultsIterator
		arg3 string
		arg4 bool
		arg5 int32
	}{arg1, arg2, arg3, arg4, arg5})
	fake.recordInvocation("BuildQueryResponse", []interface{}{arg1, arg2, arg3, arg4, arg5})
	fake.buildQueryResponseMutex.Unlock()
	if fake.BuildQueryResponseStub != nil {
		return fake.BuildQueryResponseStub(arg1, arg2, arg3, arg4, arg5)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.buildQueryResponseReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryResponseBuilder) BuildQueryResponseCallCount() int {
	fake.buildQueryResponseMutex.RLock()
	defer fake.buildQueryResponseMutex.RUnlock()
	return len(fake.buildQueryResponseArgsForCall)
}

func (fake *QueryResponseBuilder) BuildQueryResponseCalls(stub func(*chaincode.TransactionContext, ledger.ResultsIterator, string, bool, int32) (*peer.QueryResponse, error)) {
	fake.buildQueryResponseMutex.Lock()
	defer fake.buildQueryResponseMutex.Unlock()
	fake.BuildQueryResponseStub = stub
}

func (fake *QueryResponseBuilder) BuildQueryResponseArgsForCall(i int) (*chaincode.TransactionContext, ledger.ResultsIterator, string, bool, int32) {
	fake.buildQueryResponseMutex.RLock()
	defer fake.buildQueryResponseMutex.RUnlock()
	argsForCall := fake.buildQueryResponseArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4, argsForCall.arg5
}

func (fake *QueryResponseBuilder) BuildQueryResponseReturns(result1 *peer.QueryResponse, result2 error) {
	fake.buildQueryResponseMutex.Lock()
	defer fake.buildQueryResponseMutex.Unlock()
	fake.BuildQueryResponseStub = nil
	fake.buildQueryResponseReturns = struct {
		result1 *peer.QueryResponse
		result2 error
	}{result1, result2}
}

func (fake *QueryResponseBuilder) BuildQueryResponseReturnsOnCall(i int, result1 *peer.QueryResponse, result2 error) {
	fake.buildQueryResponseMutex.Lock()
	defer fake.buildQueryResponseMutex.Unlock()
	fake.BuildQueryResponseStub = nil
	if fake.buildQueryResponseReturnsOnCall == nil {
		fake.buildQueryResponseReturnsOnCall = make(map[int]struct {
			result1 *peer.QueryResponse
			result2 error
		})
	}
	fake.buildQueryResponseReturnsOnCall[i] = struct {
		result1 *peer.QueryResponse
		result2 error
	}{result1, result2}
}

func (fake *QueryResponseBuilder) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.buildQueryResponseMutex.RLock()
	defer fake.buildQueryResponseMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *QueryResponseBuilder) recordInvocation(key string, args []interface{}) {
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
