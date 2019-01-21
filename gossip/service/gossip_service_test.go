
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
	"bytes"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/core/deliverservice"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/gossip/api"
	gossipCommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/election"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/state"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	peergossip "github.com/hyperledger/fabric/peer/gossip"
	"github.com/hyperledger/fabric/peer/gossip/mocks"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/peer"
	transientstore2 "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func init() {
	util.SetupTestLogging()
}

type mockTransientStore struct {
}

func (*mockTransientStore) PurgeByHeight(maxBlockNumToRetain uint64) error {
	return nil
}

func (*mockTransientStore) Persist(txid string, blockHeight uint64, privateSimulationResults *rwset.TxPvtReadWriteSet) error {
	panic("implement me")
}

func (*mockTransientStore) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore2.TxPvtReadWriteSetWithConfigInfo) error {
	panic("implement me")
}

func (*mockTransientStore) GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error) {
	panic("implement me")
}

func (*mockTransientStore) PurgeByTxids(txids []string) error {
	panic("implement me")
}

func TestInitGossipService(t *testing.T) {
//当八卦服务确实是单身时测试
	grpcServer := grpc.NewServer()
	socket, error := net.Listen("tcp", fmt.Sprintf("%s:%d", "", 5611))
	assert.NoError(t, error)

	msptesttools.LoadMSPSetupForTesting()
	identity, _ := mgmt.GetLocalSigningIdentityOrPanic().Serialize()

	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			messageCryptoService := peergossip.NewMCS(&mocks.ChannelPolicyManagerGetter{}, localmsp.NewSigner(), mgmt.NewDeserializersManager())
			secAdv := peergossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())
			err := InitGossipService(identity, "localhost:5611", grpcServer, nil, messageCryptoService,
				secAdv, nil)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	go grpcServer.Serve(socket)
	defer grpcServer.Stop()

	defer GetGossipService().Stop()
	gossip := GetGossipService()

	for i := 0; i < 10; i++ {
		go func(gossipInstance GossipService) {
			assert.Equal(t, gossip, GetGossipService())
		}(gossip)
	}

	time.Sleep(time.Second * 2)
}

//确保*joinchannelmessage实现api.joinchannelmessage
func TestJCMInterface(t *testing.T) {
	_ = api.JoinChannelMessage(&joinChannelMessage{})
	t.Parallel()
}

func TestLeaderElectionWithDeliverClient(t *testing.T) {
	t.Parallel()
//测试领导人选举是否与模拟交付服务实例一起工作
//配置设置为使用动态领导选择
//开始10个对等点，添加到通道，最后检查是否只有一个对等点
//调用了MockDeliverService.StartDeliverForChannel

	util.SetVal("peer.gossip.useLeaderElection", true)
	util.SetVal("peer.gossip.orgLeader", false)
	n := 10
	gossips := startPeers(t, n, 20100, 0, 1, 2, 3, 4)

	channelName := "chanA"
	peerIndexes := make([]int, n)
	for i := 0; i < n; i++ {
		peerIndexes[i] = i
	}
	addPeersToChannel(t, n, 20100, channelName, gossips, peerIndexes)

	waitForFullMembership(t, gossips, n, time.Second*20, time.Second*2)

	services := make([]*electionService, n)

	for i := 0; i < n; i++ {
		deliverServiceFactory := &mockDeliverServiceFactory{
			service: &mockDeliverService{
				running: make(map[string]bool),
			},
		}
		gossips[i].(*gossipServiceImpl).deliveryFactory = deliverServiceFactory
		deliverServiceFactory.service.running[channelName] = false

		gossips[i].InitializeChannel(channelName, []string{"localhost:5005"}, Support{
			Store:     &mockTransientStore{},
			Committer: &mockLedgerInfo{1},
		})
		service, exist := gossips[i].(*gossipServiceImpl).leaderElection[channelName]
		assert.True(t, exist, "Leader election service should be created for peer %d and channel %s", i, channelName)
		services[i] = &electionService{nil, false, 0}
		services[i].LeaderElectionService = service
	}

//是单任领导人当选的。
	assert.True(t, waitForLeaderElection(t, services, time.Second*30, time.Second*2), "One leader should be selected")

	startsNum := 0
	for i := 0; i < n; i++ {
//调用了特定通道的当前对等方中的mockDeliverService.StartDeliverForChannel
		if gossips[i].(*gossipServiceImpl).deliveryService[channelName].(*mockDeliverService).running[channelName] {
			startsNum++
		}
	}

	assert.Equal(t, 1, startsNum, "Only for one peer delivery client should start")

	stopPeers(gossips)
}

func TestWithStaticDeliverClientLeader(t *testing.T) {
//测试检查静态引导标志是否正常工作。
//领袖选举旗设为假，静态领袖旗设为真
//创建了两个八卦服务实例（对等方）。
//每个对等端都被添加到通道中，并且应该运行模拟交付客户机。
//之后，每个对等端都添加到另一个客户机，它也应该为此通道运行deliver客户机。

	util.SetVal("peer.gossip.useLeaderElection", false)
	util.SetVal("peer.gossip.orgLeader", true)

	n := 2
	gossips := startPeers(t, n, 20200, 0, 1)

	channelName := "chanA"
	peerIndexes := make([]int, n)
	for i := 0; i < n; i++ {
		peerIndexes[i] = i
	}

	addPeersToChannel(t, n, 20200, channelName, gossips, peerIndexes)

	waitForFullMembership(t, gossips, n, time.Second*30, time.Second*2)

	deliverServiceFactory := &mockDeliverServiceFactory{
		service: &mockDeliverService{
			running: make(map[string]bool),
		},
	}

	for i := 0; i < n; i++ {
		gossips[i].(*gossipServiceImpl).deliveryFactory = deliverServiceFactory
		deliverServiceFactory.service.running[channelName] = false
		gossips[i].InitializeChannel(channelName, []string{"localhost:5005"}, Support{
			Committer: &mockLedgerInfo{1},
			Store:     &mockTransientStore{},
		})
	}

	for i := 0; i < n; i++ {
		assert.NotNil(t, gossips[i].(*gossipServiceImpl).deliveryService[channelName], "Delivery service for channel %s not initiated in peer %d", channelName, i)
		assert.True(t, gossips[i].(*gossipServiceImpl).deliveryService[channelName].(*mockDeliverService).running[channelName], "Block deliverer not started for peer %d", i)
	}

	channelName = "chanB"
	for i := 0; i < n; i++ {
		deliverServiceFactory.service.running[channelName] = false
		gossips[i].InitializeChannel(channelName, []string{"localhost:5005"}, Support{
			Committer: &mockLedgerInfo{1},
			Store:     &mockTransientStore{},
		})
	}

	for i := 0; i < n; i++ {
		assert.NotNil(t, gossips[i].(*gossipServiceImpl).deliveryService[channelName], "Delivery service for channel %s not initiated in peer %d", channelName, i)
		assert.True(t, gossips[i].(*gossipServiceImpl).deliveryService[channelName].(*mockDeliverService).running[channelName], "Block deliverer not started for peer %d", i)
	}

	stopPeers(gossips)
}

func TestWithStaticDeliverClientNotLeader(t *testing.T) {
	util.SetVal("peer.gossip.useLeaderElection", false)
	util.SetVal("peer.gossip.orgLeader", false)

	n := 2
	gossips := startPeers(t, n, 20300, 0, 1)

	channelName := "chanA"
	peerIndexes := make([]int, n)
	for i := 0; i < n; i++ {
		peerIndexes[i] = i
	}

	addPeersToChannel(t, n, 20300, channelName, gossips, peerIndexes)

	waitForFullMembership(t, gossips, n, time.Second*30, time.Second*2)

	deliverServiceFactory := &mockDeliverServiceFactory{
		service: &mockDeliverService{
			running: make(map[string]bool),
		},
	}

	for i := 0; i < n; i++ {
		gossips[i].(*gossipServiceImpl).deliveryFactory = deliverServiceFactory
		deliverServiceFactory.service.running[channelName] = false
		gossips[i].InitializeChannel(channelName, []string{"localhost:5005"}, Support{
			Committer: &mockLedgerInfo{1},
			Store:     &mockTransientStore{},
		})
	}

	for i := 0; i < n; i++ {
		assert.NotNil(t, gossips[i].(*gossipServiceImpl).deliveryService[channelName], "Delivery service for channel %s not initiated in peer %d", channelName, i)
		assert.False(t, gossips[i].(*gossipServiceImpl).deliveryService[channelName].(*mockDeliverService).running[channelName], "Block deliverer should not be started for peer %d", i)
	}

	stopPeers(gossips)
}

func TestWithStaticDeliverClientBothStaticAndLeaderElection(t *testing.T) {
	util.SetVal("peer.gossip.useLeaderElection", true)
	util.SetVal("peer.gossip.orgLeader", true)

	n := 2
	gossips := startPeers(t, n, 20400, 0, 1)

	channelName := "chanA"
	peerIndexes := make([]int, n)
	for i := 0; i < n; i++ {
		peerIndexes[i] = i
	}

	addPeersToChannel(t, n, 20400, channelName, gossips, peerIndexes)

	waitForFullMembership(t, gossips, n, time.Second*30, time.Second*2)

	deliverServiceFactory := &mockDeliverServiceFactory{
		service: &mockDeliverService{
			running: make(map[string]bool),
		},
	}

	for i := 0; i < n; i++ {
		gossips[i].(*gossipServiceImpl).deliveryFactory = deliverServiceFactory
		assert.Panics(t, func() {
			gossips[i].InitializeChannel(channelName, []string{"localhost:5005"}, Support{
				Committer: &mockLedgerInfo{1},
				Store:     &mockTransientStore{},
			})
		}, "Dynamic leader election based and static connection to ordering service can't exist simultaneously")
	}

	stopPeers(gossips)
}

type mockDeliverServiceFactory struct {
	service *mockDeliverService
}

func (mf *mockDeliverServiceFactory) Service(g GossipService, endpoints []string, mcs api.MessageCryptoService) (deliverclient.DeliverService, error) {
	return mf.service, nil
}

type mockDeliverService struct {
	running map[string]bool
}

func (ds *mockDeliverService) UpdateEndpoints(chainID string, endpoints []string) error {
	panic("implement me")
}

func (ds *mockDeliverService) StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, finalizer func()) error {
	ds.running[chainID] = true
	return nil
}

func (ds *mockDeliverService) StopDeliverForChannel(chainID string) error {
	ds.running[chainID] = false
	return nil
}

func (ds *mockDeliverService) Stop() {
}

type mockLedgerInfo struct {
	Height uint64
}

func (li *mockLedgerInfo) GetConfigHistoryRetriever() (ledger.ConfigHistoryRetriever, error) {
	panic("implement me")
}

func (li *mockLedgerInfo) GetMissingPvtDataTracker() (ledger.MissingPvtDataTracker, error) {
	panic("implement me")
}

func (li *mockLedgerInfo) GetPvtDataByNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	panic("implement me")
}

func (li *mockLedgerInfo) CommitWithPvtData(blockAndPvtData *ledger.BlockAndPvtData) error {
	panic("implement me")
}

func (li *mockLedgerInfo) CommitPvtDataOfOldBlocks(blockPvtData []*ledger.BlockPvtData) ([]*ledger.PvtdataHashMismatch, error) {
	panic("implement me")
}

func (li *mockLedgerInfo) GetPvtDataAndBlockByNum(seqNum uint64) (*ledger.BlockAndPvtData, error) {
	panic("implement me")
}

//LedgerHeight将模拟值返回到分类帐高度
func (li *mockLedgerInfo) LedgerHeight() (uint64, error) {
	return li.Height, nil
}

//将块提交到分类帐
func (li *mockLedgerInfo) Commit(block *common.Block) error {
	return nil
}

//获取切片中提供序列号的块
func (li *mockLedgerInfo) GetBlocks(blockSeqs []uint64) []*common.Block {
	return make([]*common.Block, 0)
}

//关闭提交服务
func (li *mockLedgerInfo) Close() {
}

func TestLeaderElectionWithRealGossip(t *testing.T) {
	t.Parallel()
//在同一个组织内用单通道生成10个八卦实例
//在每个八卦实例的顶部运行领导人选举，并检查是否只选择了一个领导人
//创建另一个通道，包括相同八卦实例1、3、5、7上的对等子集
//为新频道运行额外的领导人选举服务
//检查第一频道是否仍存在正确的主持人，第二频道是否选择了新的正确主持人。
//停止两个渠道的领导者同行的流言蜚语，并看到为两个渠道都选择了新的领导者。

//为对等端创建八卦服务实例
	n := 10
	gossips := startPeers(t, n, 20500, 0, 1, 2, 3, 4)

//将所有对等端加入第一个通道
	channelName := "chanA"
	peerIndexes := make([]int, n)
	for i := 0; i < n; i++ {
		peerIndexes[i] = i
	}
	addPeersToChannel(t, n, 20500, channelName, gossips, peerIndexes)

	waitForFullMembership(t, gossips, n, time.Second*30, time.Second*2)

	logger.Warning("Starting leader election services")

//开始领导人选举服务
	services := make([]*electionService, n)

	for i := 0; i < n; i++ {
		services[i] = &electionService{nil, false, 0}
		services[i].LeaderElectionService = gossips[i].(*gossipServiceImpl).newLeaderElectionComponent(channelName, services[i].callback)
	}

	logger.Warning("Waiting for leader election")

	assert.True(t, waitForLeaderElection(t, services, time.Second*30, time.Second*2), "One leader should be selected")

	startsNum := 0
	for i := 0; i < n; i++ {
//此领导人选举服务实例是否调用了回调函数
		if services[i].callbackInvokeRes {
			startsNum++
		}
	}
//只有leader才应该调用回调函数，所以需要反复检查是否只有一个leader
	assert.Equal(t, 1, startsNum, "Only for one peer callback function should be called - chanA")

//在新渠道中增加一些同行，为新渠道中的同行创建领袖选举服务
//期望对等1（选举服务列表中的第一个）成为第二个频道的领导者
	secondChannelPeerIndexes := []int{1, 3, 5, 7}
	secondChannelName := "chanB"
	secondChannelServices := make([]*electionService, len(secondChannelPeerIndexes))
	addPeersToChannel(t, n, 20500, secondChannelName, gossips, secondChannelPeerIndexes)

	for idx, i := range secondChannelPeerIndexes {
		secondChannelServices[idx] = &electionService{nil, false, 0}
		secondChannelServices[idx].LeaderElectionService = gossips[i].(*gossipServiceImpl).newLeaderElectionComponent(secondChannelName, secondChannelServices[idx].callback)
	}

	assert.True(t, waitForLeaderElection(t, secondChannelServices, time.Second*30, time.Second*2), "One leader should be selected for chanB")
	assert.True(t, waitForLeaderElection(t, services, time.Second*30, time.Second*2), "One leader should be selected for chanA")

	startsNum = 0
	for i := 0; i < n; i++ {
		if services[i].callbackInvokeRes {
			startsNum++
		}
	}
	assert.Equal(t, 1, startsNum, "Only for one peer callback function should be called - chanA")

	startsNum = 0
	for i := 0; i < len(secondChannelServices); i++ {
		if secondChannelServices[i].callbackInvokeRes {
			startsNum++
		}
	}
	assert.Equal(t, 1, startsNum, "Only for one peer callback function should be called - chanB")

//停止2个八卦实例（对等0和对等1），应初始化重新选择
//现在，Peer 2成为第一个通道的领导者，Peer 3成为第二个通道的领导者。

	logger.Warning("Killing 2 peers, initiation new leader election")

	stopPeers(gossips[:2])

	waitForFullMembership(t, gossips[2:], n-2, time.Second*30, time.Second*2)

	assert.True(t, waitForLeaderElection(t, services[2:], time.Second*30, time.Second*2), "One leader should be selected after re-election - chanA")
	assert.True(t, waitForLeaderElection(t, secondChannelServices[1:], time.Second*30, time.Second*2), "One leader should be selected after re-election - chanB")

	startsNum = 0
	for i := 2; i < n; i++ {
		if services[i].callbackInvokeRes {
			startsNum++
		}
	}
	assert.Equal(t, 1, startsNum, "Only for one peer callback function should be called after re-election - chanA")

	startsNum = 0
	for i := 1; i < len(secondChannelServices); i++ {
		if secondChannelServices[i].callbackInvokeRes {
			startsNum++
		}
	}
	assert.Equal(t, 1, startsNum, "Only for one peer callback function should be called after re-election - chanB")

	stopServices(secondChannelServices)
	stopServices(services)
	stopPeers(gossips[2:])
}

type electionService struct {
	election.LeaderElectionService
	callbackInvokeRes   bool
	callbackInvokeCount int
}

func (es *electionService) callback(isLeader bool) {
	es.callbackInvokeRes = isLeader
	es.callbackInvokeCount = es.callbackInvokeCount + 1
}

type joinChanMsg struct {
}

//sequence number返回此joinchanmsg块的序列号
//来源于
func (jmc *joinChanMsg) SequenceNumber() uint64 {
	return uint64(time.Now().UnixNano())
}

//成员返回频道的组织
func (jmc *joinChanMsg) Members() []api.OrgIdentityType {
	return []api.OrgIdentityType{orgInChannelA}
}

//anchor peers of返回给定组织的锚定对等方
func (jmc *joinChanMsg) AnchorPeersOf(org api.OrgIdentityType) []api.AnchorPeer {
	return []api.AnchorPeer{}
}

func waitForFullMembership(t *testing.T, gossips []GossipService, peersNum int, timeout time.Duration, testPollInterval time.Duration) bool {
	end := time.Now().Add(timeout)
	var correctPeers int
	for time.Now().Before(end) {
		correctPeers = 0
		for _, g := range gossips {
			if len(g.Peers()) == (peersNum - 1) {
				correctPeers++
			}
		}
		if correctPeers == peersNum {
			return true
		}
		time.Sleep(testPollInterval)
	}
	logger.Warningf("Only %d peers have full membership", correctPeers)
	return false
}

func waitForMultipleLeadersElection(t *testing.T, services []*electionService, leadersNum int, timeout time.Duration, testPollInterval time.Duration) bool {
	logger.Warning("Waiting for", leadersNum, "leaders")
	end := time.Now().Add(timeout)
	correctNumberOfLeadersFound := false
	leaders := 0
	for time.Now().Before(end) {
		leaders = 0
		for _, s := range services {
			if s.IsLeader() {
				leaders++
			}
		}
		if leaders == leadersNum {
			if correctNumberOfLeadersFound {
				return true
			}
			correctNumberOfLeadersFound = true
		} else {
			correctNumberOfLeadersFound = false
		}
		time.Sleep(testPollInterval)
	}
	logger.Warning("Incorrect number of leaders", leaders)
	for i, s := range services {
		logger.Warning("Peer at index", i, "is leader", s.IsLeader())
	}
	return false
}

func waitForLeaderElection(t *testing.T, services []*electionService, timeout time.Duration, testPollInterval time.Duration) bool {
	return waitForMultipleLeadersElection(t, services, 1, timeout, testPollInterval)
}

func waitUntilOrFailBlocking(t *testing.T, f func(), timeout time.Duration) {
	successChan := make(chan struct{}, 1)
	go func() {
		f()
		successChan <- struct{}{}
	}()
	select {
	case <-time.NewTimer(timeout).C:
		break
	case <-successChan:
		return
	}
	util.PrintStackTrace()
	assert.Fail(t, "Timeout expired!")
}

func stopServices(services []*electionService) {
	stoppingWg := sync.WaitGroup{}
	stoppingWg.Add(len(services))
	for i, sI := range services {
		go func(i int, s_i election.LeaderElectionService) {
			defer stoppingWg.Done()
			s_i.Stop()
		}(i, sI)
	}
	stoppingWg.Wait()
	time.Sleep(time.Second * time.Duration(2))
}

func stopPeers(peers []GossipService) {
	stoppingWg := sync.WaitGroup{}
	stoppingWg.Add(len(peers))
	for i, pI := range peers {
		go func(i int, p_i GossipService) {
			defer stoppingWg.Done()
			p_i.Stop()
		}(i, pI)
	}
	stoppingWg.Wait()
	time.Sleep(time.Second * time.Duration(2))
}

func addPeersToChannel(t *testing.T, n int, portPrefix int, channel string, peers []GossipService, peerIndexes []int) {
	jcm := &joinChanMsg{}

	wg := sync.WaitGroup{}
	for _, i := range peerIndexes {
		wg.Add(1)
		go func(i int) {
			peers[i].JoinChan(jcm, gossipCommon.ChainID(channel))
			peers[i].UpdateLedgerHeight(0, gossipCommon.ChainID(channel))
			wg.Done()
		}(i)
	}
	waitUntilOrFailBlocking(t, wg.Wait, time.Second*10)
}

func startPeers(t *testing.T, n int, portPrefix int, boot ...int) []GossipService {

	peers := make([]GossipService, n)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {

			peers[i] = newGossipInstance(portPrefix, i, 100, boot...)
			wg.Done()
		}(i)
	}
	waitUntilOrFailBlocking(t, wg.Wait, time.Second*10)

	return peers
}

func newGossipInstance(portPrefix int, id int, maxMsgCount int, boot ...int) GossipService {
	port := id + portPrefix
	conf := &gossip.Config{
		BindPort:                   port,
		BootstrapPeers:             bootPeers(portPrefix, boot...),
		ID:                         fmt.Sprintf("p%d", id),
		MaxBlockCountToStore:       maxMsgCount,
		MaxPropagationBurstLatency: time.Duration(500) * time.Millisecond,
		MaxPropagationBurstSize:    20,
		PropagateIterations:        1,
		PropagatePeerNum:           3,
		PullInterval:               time.Duration(2) * time.Second,
		PullPeerNum:                5,
		InternalEndpoint:           fmt.Sprintf("localhost:%d", port),
		ExternalEndpoint:           fmt.Sprintf("1.2.3.4:%d", port),
		PublishCertPeriod:          time.Duration(4) * time.Second,
		PublishStateInfoInterval:   time.Duration(1) * time.Second,
		RequestStateInfoInterval:   time.Duration(1) * time.Second,
		TimeForMembershipTracker:   time.Second * 5,
	}
	selfID := api.PeerIdentityType(conf.InternalEndpoint)
	cryptoService := &naiveCryptoService{}

	gossip := gossip.NewGossipServiceWithServer(conf, &orgCryptoService{}, cryptoService,
		selfID, nil)

	gossipService := &gossipServiceImpl{
		mcs:             cryptoService,
		gossipSvc:       gossip,
		chains:          make(map[string]state.GossipStateProvider),
		leaderElection:  make(map[string]election.LeaderElectionService),
		privateHandlers: make(map[string]privateHandler),
		deliveryService: make(map[string]deliverclient.DeliverService),
		deliveryFactory: &deliveryFactoryImpl{},
		peerIdentity:    api.PeerIdentityType(conf.InternalEndpoint),
	}

	return gossipService
}

func bootPeers(portPrefix int, ids ...int) []string {
	peers := []string{}
	for _, id := range ids {
		peers = append(peers, fmt.Sprintf("localhost:%d", id+portPrefix))
	}
	return peers
}

type naiveCryptoService struct {
}

type orgCryptoService struct {
}

//orgByPeerIdentity返回orgIdentityType
//给定对等身份的
func (*orgCryptoService) OrgByPeerIdentity(identity api.PeerIdentityType) api.OrgIdentityType {
	return orgInChannelA
}

//verify验证joinchanmessage，成功时返回nil，
//失败时出错
func (*orgCryptoService) Verify(joinChanMsg api.JoinChannelMessage) error {
	return nil
}

func (naiveCryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	return time.Now().Add(time.Hour), nil
}

//verifybychannel验证上下文中消息上对等方的签名
//特定通道的
func (*naiveCryptoService) VerifyByChannel(_ gossipCommon.ChainID, _ api.PeerIdentityType, _, _ []byte) error {
	return nil
}

func (*naiveCryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	return nil
}

//getpkiidofcert返回对等身份的pki-id
func (*naiveCryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) gossipCommon.PKIidType {
	return gossipCommon.PKIidType(peerIdentity)
}

//verifyblock如果块被正确签名，则返回nil，
//else返回错误
func (*naiveCryptoService) VerifyBlock(chainID gossipCommon.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

//用该对等方的签名密钥和输出对消息进行签名
//如果没有出现错误，则返回签名。
func (*naiveCryptoService) Sign(msg []byte) ([]byte, error) {
	return msg, nil
}

//验证检查签名是否是对等验证密钥下消息的有效签名。
//如果验证成功，则verify返回nil，表示没有发生错误。
//如果peercert为nil，则根据该对等方的验证密钥验证签名。
func (*naiveCryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	equal := bytes.Equal(signature, message)
	if !equal {
		return fmt.Errorf("Wrong signature:%v, %v", signature, message)
	}
	return nil
}

var orgInChannelA = api.OrgIdentityType("ORG1")

func TestInvalidInitialization(t *testing.T) {
//当八卦服务确实是单身时测试
	grpcServer := grpc.NewServer()
	socket, error := net.Listen("tcp", fmt.Sprintf("%s:%d", "", 7611))
	assert.NoError(t, error)

	go grpcServer.Serve(socket)
	defer grpcServer.Stop()

	secAdv := peergossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())
	err := InitGossipService(api.PeerIdentityType("IDENTITY"), "localhost:7611", grpcServer, nil,
		&naiveCryptoService{}, secAdv, nil)
	assert.NoError(t, err)
	gService := GetGossipService().(*gossipServiceImpl)
	defer gService.Stop()

	dc, err := gService.deliveryFactory.Service(gService, []string{}, &naiveCryptoService{})
	assert.Nil(t, dc)
	assert.Error(t, err)

	dc, err = gService.deliveryFactory.Service(gService, []string{"localhost:1984"}, &naiveCryptoService{})
	assert.NotNil(t, dc)
	assert.NoError(t, err)
}

func TestChannelConfig(t *testing.T) {
//当八卦服务确实是单身时测试
	grpcServer := grpc.NewServer()
	socket, error := net.Listen("tcp", fmt.Sprintf("%s:%d", "", 6611))
	assert.NoError(t, error)

	go grpcServer.Serve(socket)
	defer grpcServer.Stop()

	secAdv := peergossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())
	error = InitGossipService(api.PeerIdentityType("IDENTITY"), "localhost:6611", grpcServer, nil,
		&naiveCryptoService{}, secAdv, nil)
	assert.NoError(t, error)
	gService := GetGossipService().(*gossipServiceImpl)
	defer gService.Stop()

	jcm := &joinChannelMessage{seqNum: 1, members2AnchorPeers: map[string][]api.AnchorPeer{
		"A": {{Host: "host", Port: 5000}},
	}}

	assert.Equal(t, uint64(1), jcm.SequenceNumber())

	mc := &mockConfig{
		sequence: 1,
		orgs: map[string]channelconfig.ApplicationOrg{
			string(orgInChannelA): &appGrp{
				mspID:       string(orgInChannelA),
				anchorPeers: []*peer.AnchorPeer{},
			},
		},
	}
	gService.JoinChan(jcm, gossipCommon.ChainID("A"))
	gService.updateAnchors(mc)
	assert.True(t, gService.amIinChannel(string(orgInChannelA), mc))
}
