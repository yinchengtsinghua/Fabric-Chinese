
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


package service

import (
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/deliverservice"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	transientstore2 "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type transientStoreMock struct {
}

func (*transientStoreMock) PurgeByHeight(maxBlockNumToRetain uint64) error {
	return nil
}

func (*transientStoreMock) Persist(txid string, blockHeight uint64, privateSimulationResults *rwset.TxPvtReadWriteSet) error {
	panic("implement me")
}

func (*transientStoreMock) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore2.TxPvtReadWriteSetWithConfigInfo) error {
	panic("implement me")
}

func (*transientStoreMock) GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error) {
	panic("implement me")
}

func (*transientStoreMock) PurgeByTxids(txids []string) error {
	panic("implement me")
}

type embeddingDeliveryService struct {
	startOnce sync.Once
	stopOnce  sync.Once
	deliverclient.DeliverService
	startSignal sync.WaitGroup
	stopSignal  sync.WaitGroup
}

func newEmbeddingDeliveryService(ds deliverclient.DeliverService) *embeddingDeliveryService {
	eds := &embeddingDeliveryService{
		DeliverService: ds,
	}
	eds.startSignal.Add(1)
	eds.stopSignal.Add(1)
	return eds
}

func (eds *embeddingDeliveryService) waitForDeliveryServiceActivation() {
	eds.startSignal.Wait()
}

func (eds *embeddingDeliveryService) waitForDeliveryServiceTermination() {
	eds.stopSignal.Wait()
}

func (eds *embeddingDeliveryService) StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, finalizer func()) error {
	eds.startOnce.Do(func() {
		eds.startSignal.Done()
	})
	return eds.DeliverService.StartDeliverForChannel(chainID, ledgerInfo, finalizer)
}

func (eds *embeddingDeliveryService) StopDeliverForChannel(chainID string) error {
	eds.stopOnce.Do(func() {
		eds.stopSignal.Done()
	})
	return eds.DeliverService.StopDeliverForChannel(chainID)
}

func (eds *embeddingDeliveryService) Stop() {
	eds.DeliverService.Stop()
}

type embeddingDeliveryServiceFactory struct {
	DeliveryServiceFactory
}

func (edsf *embeddingDeliveryServiceFactory) Service(g GossipService, endpoints []string, mcs api.MessageCryptoService) (deliverclient.DeliverService, error) {
	ds, _ := edsf.DeliveryServiceFactory.Service(g, endpoints, mcs)
	return newEmbeddingDeliveryService(ds), nil
}

func TestLeaderYield(t *testing.T) {
//场景：生成2个同伴，等待第一个成为领导者
//没有任何订购者在场，因此领导同级无法
//连接到订购方，并应在一段时间后放弃其领导。
//确保对方很快宣布自己是领导者。
	takeOverMaxTimeout := time.Minute
	viper.Set("peer.gossip.election.leaderAliveThreshold", time.Second*5)
//测试用例只有两个实例+仅在成员身份视图之后声明
//是稳定的，因此选举时间可能更短
	viper.Set("peer.gossip.election.leaderElectionDuration", time.Millisecond*500)
//这就足以让单身人士重新尝试了。
	viper.Set("peer.deliveryclient.reconnectTotalTimeThreshold", time.Second*1)
//既然我们保证八卦有稳定的会员资格，就没有必要
//等待稳定的领导人选举
	viper.Set("peer.gossip.election.membershipSampleInterval", time.Millisecond*100)
//没有可用的订购服务，因此连接超时
//可能更短
	viper.Set("peer.deliveryclient.connTimeout", time.Millisecond*100)
	viper.Set("peer.gossip.useLeaderElection", true)
	viper.Set("peer.gossip.orgLeader", false)
	n := 2
	portPrefix := 30000
	gossips := startPeers(t, n, portPrefix, 0, 1)
	defer stopPeers(gossips)
	channelName := "channelA"
	peerIndexes := []int{0, 1}
//将对等端添加到通道
	addPeersToChannel(t, n, portPrefix, channelName, gossips, peerIndexes)
//激发同龄人的会员观
	waitForFullMembership(t, gossips, n, time.Second*30, time.Millisecond*100)
//创建gossipservice实例的helper函数
	newGossipService := func(i int) *gossipServiceImpl {
		gs := gossips[i].(*gossipServiceImpl)
		gs.deliveryFactory = &embeddingDeliveryServiceFactory{&deliveryFactoryImpl{}}
		gossipServiceInstance = gs
		gs.InitializeChannel(channelName, []string{"localhost:7050"}, Support{
			Committer: &mockLedgerInfo{1},
			Store:     &transientStoreMock{},
		})
		return gs
	}

	p0 := newGossipService(0)
	p1 := newGossipService(1)

//返回领导的索引，如果没有选择领导，则返回-1
	getLeader := func() int {
		p0.lock.RLock()
		p1.lock.RLock()
		defer p0.lock.RUnlock()
		defer p1.lock.RUnlock()

		if p0.leaderElection[channelName].IsLeader() {
			return 0
		}
		if p1.leaderElection[channelName].IsLeader() {
			return 1
		}
		return -1
	}

	ds0 := p0.deliveryService[channelName].(*embeddingDeliveryService)

//等待p0连接到订购服务
	ds0.waitForDeliveryServiceActivation()
	t.Log("p0 started its delivery service")
//确保它是领导者
	assert.Equal(t, 0, getLeader())
//等待p0失去领导权
	ds0.waitForDeliveryServiceTermination()
	t.Log("p0 stopped its delivery service")
//确保p0不是领导者
	assert.NotEqual(t, 0, getLeader())
//等待P1接管。它应该在时间到达时间限制之前接管
	timeLimit := time.Now().Add(takeOverMaxTimeout)
	for getLeader() != 1 && time.Now().Before(timeLimit) {
		time.Sleep(100 * time.Millisecond)
	}
	if time.Now().After(timeLimit) {
		util.PrintStackTrace()
		t.Fatalf("p1 hasn't taken over leadership within %v: %d", takeOverMaxTimeout, getLeader())
	}
	t.Log("p1 has taken over leadership")
	p0.chains[channelName].Stop()
	p1.chains[channelName].Stop()
	p0.deliveryService[channelName].Stop()
	p1.deliveryService[channelName].Stop()
}
