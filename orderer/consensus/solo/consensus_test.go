
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


package solo

import (
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	mockblockcutter "github.com/hyperledger/fabric/orderer/mocks/common/blockcutter"
	mockmultichannel "github.com/hyperledger/fabric/orderer/mocks/common/multichannel"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func init() {
	flogging.ActivateSpec("orderer.consensus.solo=DEBUG")
}

var testMessage = &cb.Envelope{
	Payload: utils.MarshalOrPanic(&cb.Payload{
		Header: &cb.Header{ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{ChannelId: "foo"})},
		Data:   []byte("TEST_MESSAGE"),
	}),
}

func syncQueueMessage(msg *cb.Envelope, chain *chain, bc *mockblockcutter.Receiver) {
	chain.Order(msg, 0)
	bc.Block <- struct{}{}
}

type waitableGo struct {
	done chan struct{}
}

func goWithWait(target func()) *waitableGo {
	wg := &waitableGo{
		done: make(chan struct{}),
	}
	go func() {
		target()
		close(wg.done)
	}()
	return wg
}

//这个测试检查如果同意者在计时器触发之前被停止，实际上什么都不会被写入。
func TestHaltBeforeTimeout(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("10ms")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	defer close(support.BlockCutterVal.Block)
	bs := newChain(support)
	wg := goWithWait(bs.main)
	defer bs.Halt()

	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	bs.Halt()
	select {
	case <-support.Blocks:
		t.Fatalf("Expected no invocations of Append")
	case <-wg.done:
	}
}

func TestStart(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1ms")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	close(support.BlockCutterVal.Block)
	bs, _ := New().HandleChain(support, nil)
	bs.Start()
	defer bs.Halt()

	support.BlockCutterVal.CutNext = true
	assert.Nil(t, bs.Order(testMessage, 0))
	select {
	case <-support.Blocks:
	case <-bs.Errored():
		t.Fatalf("Expected not to exit")
	}
}

func TestOrderAfterHalt(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1ms")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	defer close(support.BlockCutterVal.Block)
	bs := newChain(support)
	bs.Halt()
	assert.NotNil(t, bs.Order(testMessage, 0), "Order should not be accepted after halt")
	select {
	case <-bs.Errored():
	default:
		t.Fatalf("Expected Errored to be closed by halt")
	}
}

func TestBatchTimer(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1ms")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	defer close(support.BlockCutterVal.Block)
	bs := newChain(support)
	wg := goWithWait(bs.main)
	defer bs.Halt()

	syncQueueMessage(testMessage, bs, support.BlockCutterVal)

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected a block to be cut because of batch timer expiration but did not")
	}

	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Did not create the second batch, indicating that the timer was not appropriately reset")
	}

	support.SharedConfigVal.BatchTimeoutVal, _ = time.ParseDuration("10s")
	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	select {
	case <-support.Blocks:
		t.Fatalf("Created another batch, indicating that the timer was not appropriately re-read")
	case <-time.After(100 * time.Millisecond):
	}

	bs.Halt()
	select {
	case <-support.Blocks:
		t.Fatalf("Expected no invocations of Append")
	case <-wg.done:
	}
}

func TestBatchTimerHaltOnFilledBatch(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1h")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	defer close(support.BlockCutterVal.Block)

	bs := newChain(support)
	wg := goWithWait(bs.main)
	defer bs.Halt()

	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	support.BlockCutterVal.CutNext = true
	syncQueueMessage(testMessage, bs, support.BlockCutterVal)

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected a block to be cut because the batch was filled, but did not")
	}

//将批处理超时更改为接近即时，如果计时器未重置，它仍将等待一小时。
	support.SharedConfigVal.BatchTimeoutVal = time.Millisecond

	support.BlockCutterVal.CutNext = false
	syncQueueMessage(testMessage, bs, support.BlockCutterVal)

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Did not create the second batch, indicating that the old timer was still running")
	}

	bs.Halt()
	select {
	case <-time.After(time.Second):
		t.Fatalf("Should have exited")
	case <-wg.done:
	}
}

func TestLargeMsgStyleMultiBatch(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1h")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	defer close(support.BlockCutterVal.Block)
	bs := newChain(support)
	wg := goWithWait(bs.main)
	defer bs.Halt()

	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	support.BlockCutterVal.IsolatedTx = true
	syncQueueMessage(testMessage, bs, support.BlockCutterVal)

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected two blocks to be cut but never got the first")
	}

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected the config type tx to create two blocks, but only go the first")
	}

	bs.Halt()
	select {
	case <-time.After(time.Second):
		t.Fatalf("Should have exited")
	case <-wg.done:
	}
}

func TestConfigMsg(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1h")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	defer close(support.BlockCutterVal.Block)
	bs := newChain(support)
	wg := goWithWait(bs.main)
	defer bs.Halt()

	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	assert.Nil(t, bs.Configure(testMessage, 0))

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected two blocks to be cut but never got the first")
	}

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected the config type tx to create two blocks, but only go the first")
	}

	bs.Halt()
	select {
	case <-time.After(time.Second):
		t.Fatalf("Should have exited")
	case <-wg.done:
	}
}

//此测试检查单独同意者是否可以从错误情况中恢复。
//切割空批次的位置
func TestRecoverFromError(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1ms")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
	}
	defer close(support.BlockCutterVal.Block)
	bs := newChain(support)
	go bs.main()
	defer bs.Halt()

	support.BlockCutterVal.SkipAppendCurBatch = true
	syncQueueMessage(testMessage, bs, support.BlockCutterVal)

	select {
	case <-support.Blocks:
		t.Fatalf("Expected no invocations of Append")
	case <-time.After(100 * time.Millisecond):
	}

	support.BlockCutterVal.SkipAppendCurBatch = false
	support.BlockCutterVal.CutNext = true
	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected block to be cut")
	}
}

//此测试检查如果配置序列高级，单独同意者是否重新验证消息
func TestRevalidation(t *testing.T) {
	batchTimeout, _ := time.ParseDuration("1h")
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: batchTimeout},
		SequenceVal:     uint64(1),
	}
	defer close(support.BlockCutterVal.Block)
	bs := newChain(support)
	wg := goWithWait(bs.main)
	defer bs.Halt()

	t.Run("ConfigMsg", func(t *testing.T) {
		support.ProcessConfigMsgVal = testMessage

		t.Run("Valid", func(t *testing.T) {
			assert.Nil(t, bs.Configure(testMessage, 0))

			select {
			case <-support.Blocks:
			case <-time.After(time.Second):
				t.Fatalf("Expected one block to be cut but never got it")
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			support.ProcessConfigMsgErr = fmt.Errorf("Config message is not valid")
			assert.Nil(t, bs.Configure(testMessage, 0))

			select {
			case <-support.Blocks:
				t.Fatalf("Expected no block to be cut")
			case <-time.After(100 * time.Millisecond):
			}
		})

	})

	t.Run("NormalMsg", func(t *testing.T) {
		support.BlockCutterVal.CutNext = true

		t.Run("Valid", func(t *testing.T) {
			syncQueueMessage(testMessage, bs, support.BlockCutterVal)

			select {
			case <-support.Blocks:
			case <-time.After(time.Second):
				t.Fatalf("Expected one block to be cut but never got it")
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			support.ProcessNormalMsgErr = fmt.Errorf("Normal message is not valid")
//我们不会在此处调用'syncqueuemessage'，因为我们不希望
//“ordered”在这种情况下完全可以调用，因此我们不需要
//在“support.blockcutterval.block”上同步。
			assert.Nil(t, bs.Order(testMessage, 0))

			select {
			case <-support.Blocks:
				t.Fatalf("Expected no block to be cut")
			case <-time.After(100 * time.Millisecond):
			}
		})
	})

	bs.Halt()
	select {
	case <-time.After(time.Second):
		t.Fatalf("Should have exited")
	case <-wg.done:
	}
}

func TestPendingMsgCutByTimeout(t *testing.T) {
	support := &mockmultichannel.ConsenterSupport{
		Blocks:          make(chan *cb.Block),
		BlockCutterVal:  mockblockcutter.NewReceiver(),
		SharedConfigVal: &mockconfig.Orderer{BatchTimeoutVal: 500 * time.Millisecond},
	}
	defer close(support.BlockCutterVal.Block)

	bs := newChain(support)
	wg := goWithWait(bs.main)
	defer bs.Halt()

	syncQueueMessage(testMessage, bs, support.BlockCutterVal)
	support.BlockCutterVal.CutAncestors = true
	syncQueueMessage(testMessage, bs, support.BlockCutterVal)

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected first block to be cut")
	}

	select {
	case <-support.Blocks:
	case <-time.After(time.Second):
		t.Fatalf("Expected second block to be cut because of batch timer expiration but did not")
	}

	bs.Halt()
	select {
	case <-time.After(time.Second):
		t.Fatalf("Should have exited")
	case <-wg.done:
	}
}
