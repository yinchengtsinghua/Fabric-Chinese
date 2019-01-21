
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//伪造者生成的代码。不要编辑。
package mocks

import (
	"sync"

	"github.com/hyperledger/fabric/orderer/consensus/etcdraft"
	"github.com/hyperledger/fabric/protos/common"
)

type FakeBlockPuller struct {
	PullBlockStub        func(seq uint64) *common.Block
	pullBlockMutex       sync.RWMutex
	pullBlockArgsForCall []struct {
		seq uint64
	}
	pullBlockReturns struct {
		result1 *common.Block
	}
	pullBlockReturnsOnCall map[int]struct {
		result1 *common.Block
	}
	CloseStub        func()
	closeMutex       sync.RWMutex
	closeArgsForCall []struct{}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeBlockPuller) PullBlock(seq uint64) *common.Block {
	fake.pullBlockMutex.Lock()
	ret, specificReturn := fake.pullBlockReturnsOnCall[len(fake.pullBlockArgsForCall)]
	fake.pullBlockArgsForCall = append(fake.pullBlockArgsForCall, struct {
		seq uint64
	}{seq})
	fake.recordInvocation("PullBlock", []interface{}{seq})
	fake.pullBlockMutex.Unlock()
	if fake.PullBlockStub != nil {
		return fake.PullBlockStub(seq)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.pullBlockReturns.result1
}

func (fake *FakeBlockPuller) PullBlockCallCount() int {
	fake.pullBlockMutex.RLock()
	defer fake.pullBlockMutex.RUnlock()
	return len(fake.pullBlockArgsForCall)
}

func (fake *FakeBlockPuller) PullBlockArgsForCall(i int) uint64 {
	fake.pullBlockMutex.RLock()
	defer fake.pullBlockMutex.RUnlock()
	return fake.pullBlockArgsForCall[i].seq
}

func (fake *FakeBlockPuller) PullBlockReturns(result1 *common.Block) {
	fake.PullBlockStub = nil
	fake.pullBlockReturns = struct {
		result1 *common.Block
	}{result1}
}

func (fake *FakeBlockPuller) PullBlockReturnsOnCall(i int, result1 *common.Block) {
	fake.PullBlockStub = nil
	if fake.pullBlockReturnsOnCall == nil {
		fake.pullBlockReturnsOnCall = make(map[int]struct {
			result1 *common.Block
		})
	}
	fake.pullBlockReturnsOnCall[i] = struct {
		result1 *common.Block
	}{result1}
}

func (fake *FakeBlockPuller) Close() {
	fake.closeMutex.Lock()
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct{}{})
	fake.recordInvocation("Close", []interface{}{})
	fake.closeMutex.Unlock()
	if fake.CloseStub != nil {
		fake.CloseStub()
	}
}

func (fake *FakeBlockPuller) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *FakeBlockPuller) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.pullBlockMutex.RLock()
	defer fake.pullBlockMutex.RUnlock()
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeBlockPuller) recordInvocation(key string, args []interface{}) {
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

var _ etcdraft.BlockPuller = new(FakeBlockPuller)
