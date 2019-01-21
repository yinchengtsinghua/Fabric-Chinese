
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

	"github.com/hyperledger/fabric/bccsp/idemix/handlers"
)

type Ecp struct {
	BytesStub        func() ([]byte, error)
	bytesMutex       sync.RWMutex
	bytesArgsForCall []struct{}
	bytesReturns     struct {
		result1 []byte
		result2 error
	}
	bytesReturnsOnCall map[int]struct {
		result1 []byte
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Ecp) Bytes() ([]byte, error) {
	fake.bytesMutex.Lock()
	ret, specificReturn := fake.bytesReturnsOnCall[len(fake.bytesArgsForCall)]
	fake.bytesArgsForCall = append(fake.bytesArgsForCall, struct{}{})
	fake.recordInvocation("Bytes", []interface{}{})
	fake.bytesMutex.Unlock()
	if fake.BytesStub != nil {
		return fake.BytesStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.bytesReturns.result1, fake.bytesReturns.result2
}

func (fake *Ecp) BytesCallCount() int {
	fake.bytesMutex.RLock()
	defer fake.bytesMutex.RUnlock()
	return len(fake.bytesArgsForCall)
}

func (fake *Ecp) BytesReturns(result1 []byte, result2 error) {
	fake.BytesStub = nil
	fake.bytesReturns = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *Ecp) BytesReturnsOnCall(i int, result1 []byte, result2 error) {
	fake.BytesStub = nil
	if fake.bytesReturnsOnCall == nil {
		fake.bytesReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 error
		})
	}
	fake.bytesReturnsOnCall[i] = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *Ecp) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.bytesMutex.RLock()
	defer fake.bytesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Ecp) recordInvocation(key string, args []interface{}) {
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

var _ handlers.Ecp = new(Ecp)
