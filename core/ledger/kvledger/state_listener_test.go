
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package kvledger

import (
	"testing"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/stretchr/testify/assert"
)

func TestStateListener(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()
	provider, _ := NewProvider()
	defer provider.Close()

//创建一个侦听器并注册它以侦听命名空间中的状态更改
	channelid := "testLedger"
	namespace := "testchaincode"
	mockListener := &mockStateListener{namespace: namespace}
	provider.Initialize(&ledger.Initializer{
		DeployedChaincodeInfoProvider: &mock.DeployedChaincodeInfoProvider{},
		StateListeners:                []ledger.StateListener{mockListener},
		MetricsProvider:               &disabled.Provider{},
	})

	bg, gb := testutil.NewBlockGenerator(t, channelid, false)
	lgr, err := provider.Create(gb)
	defer lgr.Close()

//模拟TX1
	sim1, err := lgr.NewTxSimulator("test_tx_1")
	assert.NoError(t, err)
	sim1.GetState(namespace, "key1")
	sim1.SetState(namespace, "key1", []byte("value1"))
	sim1.SetState(namespace, "key2", []byte("value2"))
	sim1.Done()

//模拟tx2-这与tx1有冲突，因为它读取“key1”
	sim2, err := lgr.NewTxSimulator("test_tx_2")
	assert.NoError(t, err)
	sim2.GetState(namespace, "key1")
	sim2.SetState(namespace, "key3", []byte("value3"))
	sim2.Done()

//模拟tx3-这与tx1或tx2稍有冲突
	sim3, err := lgr.NewTxSimulator("test_tx_3")
	assert.NoError(t, err)
	sim3.SetState(namespace, "key4", []byte("value4"))
	sim3.Done()

//提交tx1，这将导致模拟侦听器接收tx1所做的状态更改。
	mockListener.reset()
	sim1Res, _ := sim1.GetTxSimulationResults()
	sim1ResBytes, _ := sim1Res.GetPubSimulationBytes()
	assert.NoError(t, err)
	blk1 := bg.NextBlock([][]byte{sim1ResBytes})
	assert.NoError(t, lgr.CommitWithPvtData(&ledger.BlockAndPvtData{Block: blk1}))
	assert.Equal(t, channelid, mockListener.channelName)
	assert.Contains(t, mockListener.kvWrites, &kvrwset.KVWrite{Key: "key1", Value: []byte("value1")})
	assert.Contains(t, mockListener.kvWrites, &kvrwset.KVWrite{Key: "key2", Value: []byte("value2")})
//提交tx2，这不会导致模拟侦听器接收tx2所做的状态更改。
//（因为，tx2应该是无效的）
	mockListener.reset()
	sim2Res, _ := sim2.GetTxSimulationResults()
	sim2ResBytes, _ := sim2Res.GetPubSimulationBytes()
	assert.NoError(t, err)
	blk2 := bg.NextBlock([][]byte{sim2ResBytes})
	assert.NoError(t, lgr.CommitWithPvtData(&ledger.BlockAndPvtData{Block: blk2}))
	assert.Equal(t, "", mockListener.channelName)
	assert.Nil(t, mockListener.kvWrites)

//提交tx3和thsi应使模拟侦听器接收tx3所做的更改
	mockListener.reset()
	sim3Res, _ := sim3.GetTxSimulationResults()
	sim3ResBytes, _ := sim3Res.GetPubSimulationBytes()
	assert.NoError(t, err)
	blk3 := bg.NextBlock([][]byte{sim3ResBytes})
	assert.NoError(t, lgr.CommitWithPvtData(&ledger.BlockAndPvtData{Block: blk3}))
	assert.Equal(t, channelid, mockListener.channelName)
	assert.Equal(t, []*kvrwset.KVWrite{
		{Key: "key4", Value: []byte("value4")},
	}, mockListener.kvWrites)
}

type mockStateListener struct {
	channelName string
	namespace   string
	kvWrites    []*kvrwset.KVWrite
}

func (l *mockStateListener) InterestedInNamespaces() []string {
	return []string{l.namespace}
}

func (l *mockStateListener) HandleStateUpdates(trigger *ledger.StateUpdateTrigger) error {
	channelName, stateUpdates := trigger.LedgerID, trigger.StateUpdates
	l.channelName = channelName
	l.kvWrites = stateUpdates[l.namespace].([]*kvrwset.KVWrite)
	return nil
}

func (l *mockStateListener) StateCommitDone(channelID string) {
//诺普
}

func (l *mockStateListener) reset() {
	l.channelName = ""
	l.kvWrites = nil
}
