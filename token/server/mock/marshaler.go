
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
	"sync"

	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/token/server"
)

type Marshaler struct {
	MarshalCommandResponseStub        func(command []byte, responsePayload interface{}) (*token.SignedCommandResponse, error)
	marshalCommandResponseMutex       sync.RWMutex
	marshalCommandResponseArgsForCall []struct {
		command         []byte
		responsePayload interface{}
	}
	marshalCommandResponseReturns struct {
		result1 *token.SignedCommandResponse
		result2 error
	}
	marshalCommandResponseReturnsOnCall map[int]struct {
		result1 *token.SignedCommandResponse
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Marshaler) MarshalCommandResponse(command []byte, responsePayload interface{}) (*token.SignedCommandResponse, error) {
	var commandCopy []byte
	if command != nil {
		commandCopy = make([]byte, len(command))
		copy(commandCopy, command)
	}
	fake.marshalCommandResponseMutex.Lock()
	ret, specificReturn := fake.marshalCommandResponseReturnsOnCall[len(fake.marshalCommandResponseArgsForCall)]
	fake.marshalCommandResponseArgsForCall = append(fake.marshalCommandResponseArgsForCall, struct {
		command         []byte
		responsePayload interface{}
	}{commandCopy, responsePayload})
	fake.recordInvocation("MarshalCommandResponse", []interface{}{commandCopy, responsePayload})
	fake.marshalCommandResponseMutex.Unlock()
	if fake.MarshalCommandResponseStub != nil {
		return fake.MarshalCommandResponseStub(command, responsePayload)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.marshalCommandResponseReturns.result1, fake.marshalCommandResponseReturns.result2
}

func (fake *Marshaler) MarshalCommandResponseCallCount() int {
	fake.marshalCommandResponseMutex.RLock()
	defer fake.marshalCommandResponseMutex.RUnlock()
	return len(fake.marshalCommandResponseArgsForCall)
}

func (fake *Marshaler) MarshalCommandResponseArgsForCall(i int) ([]byte, interface{}) {
	fake.marshalCommandResponseMutex.RLock()
	defer fake.marshalCommandResponseMutex.RUnlock()
	return fake.marshalCommandResponseArgsForCall[i].command, fake.marshalCommandResponseArgsForCall[i].responsePayload
}

func (fake *Marshaler) MarshalCommandResponseReturns(result1 *token.SignedCommandResponse, result2 error) {
	fake.MarshalCommandResponseStub = nil
	fake.marshalCommandResponseReturns = struct {
		result1 *token.SignedCommandResponse
		result2 error
	}{result1, result2}
}

func (fake *Marshaler) MarshalCommandResponseReturnsOnCall(i int, result1 *token.SignedCommandResponse, result2 error) {
	fake.MarshalCommandResponseStub = nil
	if fake.marshalCommandResponseReturnsOnCall == nil {
		fake.marshalCommandResponseReturnsOnCall = make(map[int]struct {
			result1 *token.SignedCommandResponse
			result2 error
		})
	}
	fake.marshalCommandResponseReturnsOnCall[i] = struct {
		result1 *token.SignedCommandResponse
		result2 error
	}{result1, result2}
}

func (fake *Marshaler) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.marshalCommandResponseMutex.RLock()
	defer fake.marshalCommandResponseMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Marshaler) recordInvocation(key string, args []interface{}) {
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

var _ server.Marshaler = new(Marshaler)
