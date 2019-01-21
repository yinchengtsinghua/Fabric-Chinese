
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


package deliverclient

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/core/deliverservice/mocks"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func init() {
	msptesttools.LoadMSPSetupForTesting()
}

const (
	goRoutineTestWaitTimeout = time.Second * 15
)

var (
	lock = sync.Mutex{}
)

type mockBlocksDelivererFactory struct {
	mockCreate func() (blocksprovider.BlocksDeliverer, error)
}

func (mock *mockBlocksDelivererFactory) Create() (blocksprovider.BlocksDeliverer, error) {
	return mock.mockCreate()
}

type mockMCS struct {
}

func (*mockMCS) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	return time.Now().Add(time.Hour), nil
}

func (*mockMCS) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return common.PKIidType("pkiID")
}

func (*mockMCS) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

func (*mockMCS) Sign(msg []byte) ([]byte, error) {
	return msg, nil
}

func (*mockMCS) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return nil
}

func (*mockMCS) VerifyByChannel(chainID common.ChainID, peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return nil
}

func (*mockMCS) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	return nil
}

func TestNewDeliverService(t *testing.T) {
	defer ensureNoGoroutineLeak(t)()
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64, 1)}
	factory := &struct{ mockBlocksDelivererFactory }{}

	blocksDeliverer := &mocks.MockBlocksDeliverer{}
	blocksDeliverer.MockRecv = mocks.MockRecv

	factory.mockCreate = func() (blocksprovider.BlocksDeliverer, error) {
		return blocksDeliverer, nil
	}
	abcf := func(*grpc.ClientConn) orderer.AtomicBroadcastClient {
		return &mocks.MockAtomicBroadcastClient{
			BD: blocksDeliverer,
		}
	}

	connFactory := func(_ string) func(string) (*grpc.ClientConn, error) {
		return func(endpoint string) (*grpc.ClientConn, error) {
			lock.Lock()
			defer lock.Unlock()
			return newConnection(), nil
		}
	}
	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"a"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  abcf,
		ConnFactory: connFactory,
	})
	assert.NoError(t, err)
	assert.NoError(t, service.StartDeliverForChannel("TEST_CHAINID", &mocks.MockLedgerInfo{Height: 0}, func() {}))

//让我们开始交付两次
	assert.Error(t, service.StartDeliverForChannel("TEST_CHAINID", &mocks.MockLedgerInfo{Height: 0}, func() {}), "can't start delivery")
//让我们停止交付未启动的
	assert.Error(t, service.StopDeliverForChannel("TEST_CHAINID2"), "can't stop delivery")

//让它尝试模拟一些recv->八卦轮
	time.Sleep(time.Second)
	assert.NoError(t, service.StopDeliverForChannel("TEST_CHAINID"))
	time.Sleep(time.Duration(10) * time.Millisecond)
//确保停止所有块提供程序
	service.Stop()
	time.Sleep(time.Duration(500) * time.Millisecond)
	connWG.Wait()

	assertBlockDissemination(0, gossipServiceAdapter.GossipBlockDisseminations, t)
	assert.Equal(t, blocksDeliverer.RecvCount(), gossipServiceAdapter.AddPayloadCount())
	assert.Error(t, service.StartDeliverForChannel("TEST_CHAINID", &mocks.MockLedgerInfo{Height: 0}, func() {}), "Delivery service is stopping")
	assert.Error(t, service.StopDeliverForChannel("TEST_CHAINID"), "Delivery service is stopping")
}

func TestDeliverServiceRestart(t *testing.T) {
	defer ensureNoGoroutineLeak(t)()
//场景：打开订购服务实例，然后关闭它，然后恢复它。
//客户机需要重新连接到它，并请求下一个块的块序列
//在最后一个块之后，它从以前的订购服务的体现中获得。

	os := mocks.NewOrderer(5611, t)

	time.Sleep(time.Second)
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}

	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"localhost:5611"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.NoError(t, err)

	li := &mocks.MockLedgerInfo{Height: uint64(100)}
	os.SetNextExpectedSeek(uint64(100))

	err = service.StartDeliverForChannel("TEST_CHAINID", li, func() {})
	assert.NoError(t, err, "can't start delivery")
//检查传递客户端请求是否按顺序阻塞
	go os.SendBlock(uint64(100))
	assertBlockDissemination(100, gossipServiceAdapter.GossipBlockDisseminations, t)
	go os.SendBlock(uint64(101))
	assertBlockDissemination(101, gossipServiceAdapter.GossipBlockDisseminations, t)
	go os.SendBlock(uint64(102))
	assertBlockDissemination(102, gossipServiceAdapter.GossipBlockDisseminations, t)
	os.Shutdown()
	time.Sleep(time.Second * 3)
	os = mocks.NewOrderer(5611, t)
	atomic.StoreUint64(&li.Height, uint64(103))
	os.SetNextExpectedSeek(uint64(103))
	go os.SendBlock(uint64(103))
	assertBlockDissemination(103, gossipServiceAdapter.GossipBlockDisseminations, t)
	service.Stop()
	os.Shutdown()
}

func TestDeliverServiceFailover(t *testing.T) {
	defer ensureNoGoroutineLeak(t)()
//场景：调出2个订购服务实例，
//关闭客户端连接到的实例。
//客户机应该连接到另一个实例，并请求下一个块的块序列
//在最后一个块之后，它从关闭的订购服务获得。
//然后，关闭另一个节点，并返回第一个节点（即先关闭的节点）。

	os1 := mocks.NewOrderer(5612, t)
	os2 := mocks.NewOrderer(5613, t)

	time.Sleep(time.Second)
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}

	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"localhost:5612", "localhost:5613"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.NoError(t, err)
	li := &mocks.MockLedgerInfo{Height: uint64(100)}
	os1.SetNextExpectedSeek(uint64(100))
	os2.SetNextExpectedSeek(uint64(100))

	err = service.StartDeliverForChannel("TEST_CHAINID", li, func() {})
	assert.NoError(t, err, "can't start delivery")
//我们需要发现客户机连接到哪个实例
	go os1.SendBlock(uint64(100))
	instance2fail := os1
	reincarnatedNodePort := 5612
	instance2failSecond := os2
	select {
	case seq := <-gossipServiceAdapter.GossipBlockDisseminations:
		assert.Equal(t, uint64(100), seq)
	case <-time.After(time.Second * 2):
//关闭第一个实例并替换它，以便创建一个实例
//发送通道为空
		os1.Shutdown()
		time.Sleep(time.Second)
		os1 = mocks.NewOrderer(5612, t)
		instance2fail = os2
		instance2failSecond = os1
		reincarnatedNodePort = 5613
//确保我们确实连接到第二个实例，
//让它发送一个块
		go os2.SendBlock(uint64(100))
		assertBlockDissemination(100, gossipServiceAdapter.GossipBlockDisseminations, t)
	}

	atomic.StoreUint64(&li.Height, uint64(101))
	os1.SetNextExpectedSeek(uint64(101))
	os2.SetNextExpectedSeek(uint64(101))
//客户机连接到的医嘱者节点失败
	instance2fail.Shutdown()
	time.Sleep(time.Second)
//确保客户端从其他订购服务节点请求块
	go instance2failSecond.SendBlock(uint64(101))
	assertBlockDissemination(101, gossipServiceAdapter.GossipBlockDisseminations, t)
	atomic.StoreUint64(&li.Height, uint64(102))
//现在关闭第二个节点
	instance2failSecond.Shutdown()
	time.Sleep(time.Second * 1)
//把第一个带上来
	os := mocks.NewOrderer(reincarnatedNodePort, t)
	os.SetNextExpectedSeek(102)
	go os.SendBlock(uint64(102))
	assertBlockDissemination(102, gossipServiceAdapter.GossipBlockDisseminations, t)
	os.Shutdown()
	service.Stop()
}

func TestDeliverServiceUpdateEndpoints(t *testing.T) {
//TODO:添加测试用例以检查端点更新
//案例：使用给定的订购服务端点启动服务
//发送一个块，切换到新端点并发送一个新块
//预期：交付服务应能够切换到
//及时更新端点并接收第二个块。
	defer ensureNoGoroutineLeak(t)()

	os1 := mocks.NewOrderer(5612, t)

	time.Sleep(time.Second)
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}

	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"localhost:5612"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	defer service.Stop()

	assert.NoError(t, err)
	li := &mocks.MockLedgerInfo{Height: uint64(100)}
	os1.SetNextExpectedSeek(uint64(100))

	err = service.StartDeliverForChannel("TEST_CHAINID", li, func() {})
	assert.NoError(t, err, "can't start delivery")

	go os1.SendBlock(uint64(100))
	assertBlockDissemination(100, gossipServiceAdapter.GossipBlockDisseminations, t)

	os2 := mocks.NewOrderer(5613, t)
	defer os2.Shutdown()
	os2.SetNextExpectedSeek(uint64(101))

	service.UpdateEndpoints("TEST_CHAINID", []string{"localhost:5613"})
//关闭旧的订购服务以确保块现在来自
//更新的订购服务
	os1.Shutdown()

	atomic.StoreUint64(&li.Height, uint64(101))
	go os2.SendBlock(uint64(101))
	assertBlockDissemination(101, gossipServiceAdapter.GossipBlockDisseminations, t)
}

func TestDeliverServiceServiceUnavailable(t *testing.T) {
	orgEndpointDisableInterval := comm.EndpointDisableInterval
	comm.EndpointDisableInterval = time.Millisecond * 1500
	defer func() { comm.EndpointDisableInterval = orgEndpointDisableInterval }()
	defer ensureNoGoroutineLeak(t)()
//场景：调出2个订购服务实例，
//使客户端连接的实例在块传递后失败，发送服务不可用
//无论何时向其发送后续请求。
//客户机应该连接到另一个实例，并请求下一个块的块序列
//在最后一个块之后，它从第一个订购服务节点获得。
//等待终结点禁用间隔
//之后，恢复失败的节点（第一个节点）和失败的实例客户端当前连接-发送服务不可用
//客户端应该重新连接到原始实例并请求下一个块。

	os1 := mocks.NewOrderer(5615, t)
	os2 := mocks.NewOrderer(5616, t)

	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}

	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"localhost:5615", "localhost:5616"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.NoError(t, err)
	li := &mocks.MockLedgerInfo{Height: 100}
	os1.SetNextExpectedSeek(li.Height)
	os2.SetNextExpectedSeek(li.Height)

	err = service.StartDeliverForChannel("TEST_CHAINID", li, func() {})
	assert.NoError(t, err, "can't start delivery")

	waitForConnectionToSomeOSN := func() (*mocks.Orderer, *mocks.Orderer) {
		for {
			if os1.ConnCount() > 0 {
				return os1, os2
			}
			if os2.ConnCount() > 0 {
				return os2, os1
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	activeInstance, backupInstance := waitForConnectionToSomeOSN()
	assert.NotNil(t, activeInstance)
	assert.NotNil(t, backupInstance)
//检查传递客户端是否连接到活动
	assert.Equal(t, activeInstance.ConnCount(), 1)
//且未连接到备份实例
	assert.Equal(t, backupInstance.ConnCount(), 0)

//发送第一个块
	go activeInstance.SendBlock(li.Height)

	assertBlockDissemination(li.Height, gossipServiceAdapter.GossipBlockDisseminations, t)
	li.Height++

//备份实例应该期望查找101，因为我们得到100
	backupInstance.SetNextExpectedSeek(li.Height)
//让备份实例准备发送块
	backupInstance.SendBlock(li.Height)

//连接到的实例传递客户端失败
	activeInstance.Fail()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func(ctx context.Context) {
		defer wg.Done()
		for {
			select {
			case <-time.After(time.Millisecond * 100):
				if backupInstance.ConnCount() > 0 {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	wg.Wait()
	assert.NoError(t, ctx.Err(), "Delivery client has not failed over to alive ordering service")
//检查交付客户端是否确实已连接
	assert.Equal(t, backupInstance.ConnCount(), 1)
//确保客户端从其他订购服务节点请求块
	assertBlockDissemination(li.Height, gossipServiceAdapter.GossipBlockDisseminations, t)

//等待第一个端点再次启用
	time.Sleep(time.Millisecond * 1600)

	li.Height++
	activeInstance.Resurrect()
	backupInstance.Fail()

	resurrectCtx, resCancel := context.WithTimeout(context.Background(), time.Second)
	defer resCancel()

	go func() {
//复活的实例应该期望得到102，因为我们得到101
		activeInstance.SetNextExpectedSeek(li.Height)
//已恢复实例准备发送块
		activeInstance.SendBlock(li.Height)

	}()

	reswg := sync.WaitGroup{}
	reswg.Add(1)

	go func() {
		defer reswg.Done()
		for {
			select {
			case <-time.After(time.Millisecond * 100):
				if activeInstance.ConnCount() > 0 {
					return
				}
			case <-resurrectCtx.Done():
				return
			}
		}
	}()

	reswg.Wait()

	assert.NoError(t, resurrectCtx.Err(), "Delivery client has not failed over to alive ordering service")
//检查交付客户端是否确实已连接
	assert.Equal(t, activeInstance.ConnCount(), 1)
//确保客户端从其他订购服务节点请求块
	assertBlockDissemination(li.Height, gossipServiceAdapter.GossipBlockDisseminations, t)

//清理
	os1.Shutdown()
	os2.Shutdown()
	service.Stop()
}

func TestDeliverServiceAbruptStop(t *testing.T) {
	defer ensureNoGoroutineLeak(t)()
//场景：交付服务启动并突然停止。
//块提供程序实例在单独的goroutine中运行，因此
//它可能是在传递客户端停止之后安排的。
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}
	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"a"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.NoError(t, err)

	li := &mocks.MockLedgerInfo{Height: uint64(100)}
	service.StartDeliverForChannel("mychannel", li, func() {})
	service.StopDeliverForChannel("mychannel")
}

func TestDeliverServiceShutdown(t *testing.T) {
	defer ensureNoGoroutineLeak(t)()
//场景：启动一个订购服务节点，让客户机拉一些块。
//然后，关闭客户机，并检查它是否不再获取块。
	os := mocks.NewOrderer(5614, t)

	time.Sleep(time.Second)
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}

	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"localhost:5614"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.NoError(t, err)

	li := &mocks.MockLedgerInfo{Height: uint64(100)}
	os.SetNextExpectedSeek(uint64(100))
	err = service.StartDeliverForChannel("TEST_CHAINID", li, func() {})
	assert.NoError(t, err, "can't start delivery")

//检查交货服务请求块是否有序
	go os.SendBlock(uint64(100))
	assertBlockDissemination(100, gossipServiceAdapter.GossipBlockDisseminations, t)
	go os.SendBlock(uint64(101))
	assertBlockDissemination(101, gossipServiceAdapter.GossipBlockDisseminations, t)
	atomic.StoreUint64(&li.Height, uint64(102))
	os.SetNextExpectedSeek(uint64(102))
//现在停止送货服务，确保我们不会传播一个区块
	service.Stop()
	go os.SendBlock(uint64(102))
	select {
	case <-gossipServiceAdapter.GossipBlockDisseminations:
		assert.Fail(t, "Disseminated a block after shutting down the delivery service")
	case <-time.After(time.Second * 2):
	}
	os.Shutdown()
	time.Sleep(time.Second)
}

func TestDeliverServiceShutdownRespawn(t *testing.T) {
//场景：启动一个订购服务节点，让客户机拉一些块。
//然后，等待几秒钟，不要发送任何块。
//之后-启动一个新实例并关闭旧实例。
	viper.Set("peer.deliveryclient.reconnectTotalTimeThreshold", time.Second)
	defer viper.Reset()
	defer ensureNoGoroutineLeak(t)()

	osn1 := mocks.NewOrderer(5614, t)

	time.Sleep(time.Second)
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}

	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"localhost:5614", "localhost:5615"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.NoError(t, err)

	li := &mocks.MockLedgerInfo{Height: uint64(100)}
	osn1.SetNextExpectedSeek(uint64(100))
	err = service.StartDeliverForChannel("TEST_CHAINID", li, func() {})
	assert.NoError(t, err, "can't start delivery")

//检查交货服务请求块是否有序
	go osn1.SendBlock(uint64(100))
	assertBlockDissemination(100, gossipServiceAdapter.GossipBlockDisseminations, t)
	go osn1.SendBlock(uint64(101))
	assertBlockDissemination(101, gossipServiceAdapter.GossipBlockDisseminations, t)
	atomic.StoreUint64(&li.Height, uint64(102))
//现在等几秒钟
	time.Sleep(time.Second * 2)
//现在启动新实例
	osn2 := mocks.NewOrderer(5615, t)
//现在停止旧实例
	osn1.Shutdown()
//从OSN2发送块
	osn2.SetNextExpectedSeek(uint64(102))
	go osn2.SendBlock(uint64(102))
//确保收到
	assertBlockDissemination(102, gossipServiceAdapter.GossipBlockDisseminations, t)
	service.Stop()
	osn2.Shutdown()
}

func TestDeliverServiceDisconnectReconnect(t *testing.T) {
//场景：启动一个订购服务节点，让客户机拉一些块。
//停止订购服务，等待-模拟断开连接并重新启动。
//等待一段时间，不发送数据块-模拟接收在空通道上等待。
//多次重复停止/启动序列，以确保总重试时间将过去。
//GetReconnectTotalTimeThreshold返回的值-在测试中将其设置为2秒
//（0.5s+1s+2s+4s）>2s。
//发送新的块并检查交付客户端是否收到了它。
//所以，我们可以看到在空通道中等待recv会重置重新连接所花费的总时间。
	viper.Set("peer.deliveryclient.reconnectTotalTimeThreshold", time.Second*2)
	defer viper.Reset()
	defer ensureNoGoroutineLeak(t)()

	osn := mocks.NewOrderer(5614, t)

	time.Sleep(time.Second)
	gossipServiceAdapter := &mocks.MockGossipServiceAdapter{GossipBlockDisseminations: make(chan uint64)}

	service, err := NewDeliverService(&Config{
		Endpoints:   []string{"localhost:5614"},
		Gossip:      gossipServiceAdapter,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.NoError(t, err)

	li := &mocks.MockLedgerInfo{Height: uint64(100)}
	osn.SetNextExpectedSeek(uint64(100))
	err = service.StartDeliverForChannel("TEST_CHAINID", li, func() {})
	assert.NoError(t, err, "can't start delivery")

//检查交货服务请求块是否有序
	go osn.SendBlock(uint64(100))
	assertBlockDissemination(100, gossipServiceAdapter.GossipBlockDisseminations, t)
	go osn.SendBlock(uint64(101))
	assertBlockDissemination(101, gossipServiceAdapter.GossipBlockDisseminations, t)
	atomic.StoreUint64(&li.Height, uint64(102))

	for i := 0; i < 5; i += 1 {
//关闭订购程序，模拟网络断开
		osn.Shutdown()
//现在等待发现断开连接
		assert.True(t, waitForConnectionCount(osn, 0), "deliverService can't disconnect from orderer")
//重新创建医嘱者，模拟网络恢复
		osn = mocks.NewOrderer(5614, t)
		osn.SetNextExpectedSeek(atomic.LoadUint64(&li.Height))
//现在，等待一段时间，以便客户端重新连接并模拟空通道
		assert.True(t, waitForConnectionCount(osn, 1), "deliverService can't reconnect to orderer")
	}

//从订购者发送块
	go osn.SendBlock(uint64(102))
//确保收到
	assertBlockDissemination(102, gossipServiceAdapter.GossipBlockDisseminations, t)
	service.Stop()
	osn.Shutdown()
}

func TestDeliverServiceBadConfig(t *testing.T) {
//空端点
	service, err := NewDeliverService(&Config{
		Endpoints:   []string{},
		Gossip:      &mocks.MockGossipServiceAdapter{},
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.Error(t, err)
	assert.Nil(t, service)

//无流言适配器
	service, err = NewDeliverService(&Config{
		Endpoints:   []string{"a"},
		Gossip:      nil,
		CryptoSvc:   &mockMCS{},
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.Error(t, err)
	assert.Nil(t, service)

//无加密服务
	service, err = NewDeliverService(&Config{
		Endpoints:   []string{"a"},
		Gossip:      &mocks.MockGossipServiceAdapter{},
		CryptoSvc:   nil,
		ABCFactory:  DefaultABCFactory,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.Error(t, err)
	assert.Nil(t, service)

//零abcPrim工厂
	service, err = NewDeliverService(&Config{
		Endpoints:   []string{"a"},
		Gossip:      &mocks.MockGossipServiceAdapter{},
		CryptoSvc:   &mockMCS{},
		ABCFactory:  nil,
		ConnFactory: DefaultConnectionFactory,
	})
	assert.Error(t, err)
	assert.Nil(t, service)

//零连接
	service, err = NewDeliverService(&Config{
		Endpoints:  []string{"a"},
		Gossip:     &mocks.MockGossipServiceAdapter{},
		CryptoSvc:  &mockMCS{},
		ABCFactory: DefaultABCFactory,
	})
	assert.Error(t, err)
	assert.Nil(t, service)
}

func TestRetryPolicyOverflow(t *testing.T) {
	connFactory := func(channelID string) func(endpoint string) (*grpc.ClientConn, error) {
		return func(_ string) (*grpc.ClientConn, error) {
			return nil, errors.New("")
		}
	}
	client := (&deliverServiceImpl{conf: &Config{ConnFactory: connFactory}}).newClient("TEST", &mocks.MockLedgerInfo{Height: uint64(100)})
	assert.NotNil(t, client.shouldRetry)
	for i := 0; i < 100; i++ {
		retryTime, _ := client.shouldRetry(i, time.Second)
		assert.True(t, retryTime <= time.Hour && retryTime > 0)
	}
}

func assertBlockDissemination(expectedSeq uint64, ch chan uint64, t *testing.T) {
	select {
	case seq := <-ch:
		assert.Equal(t, expectedSeq, seq)
	case <-time.After(time.Second * 5):
		assert.FailNow(t, fmt.Sprintf("Didn't gossip a new block with seq num %d within a timely manner", expectedSeq))
		t.Fatal()
	}
}

func ensureNoGoroutineLeak(t *testing.T) func() {
	goroutineCountAtStart := runtime.NumGoroutine()
	return func() {
		start := time.Now()
		timeLimit := start.Add(goRoutineTestWaitTimeout)
		for time.Now().Before(timeLimit) {
			time.Sleep(time.Millisecond * 500)
			if goroutineCountAtStart >= runtime.NumGoroutine() {
				return
			}
		}
		assert.Fail(t, "Some goroutine(s) didn't finish: %s", getStackTrace())
	}
}

func getStackTrace() string {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	return string(buf)
}

func waitForConnectionCount(orderer *mocks.Orderer, connCount int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	for {
		select {
		case <-time.After(time.Millisecond * 100):
			if orderer.ConnCount() == connCount {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}
