
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

	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/deliverservice"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/gossip/api"
	gossipCommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/election"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/integration"
	privdata2 "github.com/hyperledger/fabric/gossip/privdata"
	"github.com/hyperledger/fabric/gossip/state"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	gproto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	gossipServiceInstance *gossipServiceImpl
	once                  sync.Once
)

type gossipSvc gossip.Gossip

//gossipservice将流言和状态功能封装到单个接口中
type GossipService interface {
	gossip.Gossip

//distributeprivatedata将私有数据分发给集合中的对等方
//根据policyStore和policyParser诱导的策略
	DistributePrivateData(chainID string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error
//newconfigeventer创建一个configprocessor，channelconfig.bundlesource最终可以将配置更新路由到
	NewConfigEventer() ConfigProcessor
//InitializeChannel分配状态提供程序，每次执行时应在每个通道调用一次。
	InitializeChannel(chainID string, endpoints []string, support Support)
//对于给定的链，addPayload将消息有效负载附加到
	AddPayload(chainID string, payload *gproto.Payload) error
}

//DeliveryServiceFactory工厂以创建和初始化传递服务实例
type DeliveryServiceFactory interface {
//返回传递客户端的实例
	Service(g GossipService, endpoints []string, msc api.MessageCryptoService) (deliverclient.DeliverService, error)
}

type deliveryFactoryImpl struct {
}

//返回传递客户端的实例
func (*deliveryFactoryImpl) Service(g GossipService, endpoints []string, mcs api.MessageCryptoService) (deliverclient.DeliverService, error) {
	return deliverclient.NewDeliverService(&deliverclient.Config{
		CryptoSvc:   mcs,
		Gossip:      g,
		Endpoints:   endpoints,
		ConnFactory: deliverclient.DefaultConnectionFactory,
		ABCFactory:  deliverclient.DefaultABCFactory,
	})
}

type privateHandler struct {
	support     Support
	coordinator privdata2.Coordinator
	distributor privdata2.PvtDataDistributor
	reconciler  privdata2.PvtDataReconciler
}

func (p privateHandler) close() {
	p.coordinator.Close()
	p.reconciler.Stop()
}

type gossipServiceImpl struct {
	gossipSvc
	privateHandlers map[string]privateHandler
	chains          map[string]state.GossipStateProvider
	leaderElection  map[string]election.LeaderElectionService
	deliveryService map[string]deliverclient.DeliverService
	deliveryFactory DeliveryServiceFactory
	lock            sync.RWMutex
	mcs             api.MessageCryptoService
	peerIdentity    []byte
	secAdv          api.SecurityAdvisor
}

//这是API.JoinChannelMessage的实现。
type joinChannelMessage struct {
	seqNum              uint64
	members2AnchorPeers map[string][]api.AnchorPeer
}

func (jcm *joinChannelMessage) SequenceNumber() uint64 {
	return jcm.seqNum
}

//成员返回频道的组织
func (jcm *joinChannelMessage) Members() []api.OrgIdentityType {
	members := make([]api.OrgIdentityType, len(jcm.members2AnchorPeers))
	i := 0
	for org := range jcm.members2AnchorPeers {
		members[i] = api.OrgIdentityType(org)
		i++
	}
	return members
}

//anchor peers of返回给定组织的锚定对等方
func (jcm *joinChannelMessage) AnchorPeersOf(org api.OrgIdentityType) []api.AnchorPeer {
	return jcm.members2AnchorPeers[string(org)]
}

var logger = util.GetLogger(util.ServiceLogger, "")

//initgossipservice初始化八卦服务
func InitGossipService(peerIdentity []byte, endpoint string, s *grpc.Server, certs *gossipCommon.TLSCertificates,
	mcs api.MessageCryptoService, secAdv api.SecurityAdvisor, secureDialOpts api.PeerSecureDialOpts, bootPeers ...string) error {
//TODO:删除此。
//TODO:这是一项临时工作，可以让八卦领袖选举模块在启动时加载其记录器。
//TODO:为了使Flogging包及时注册此记录器，以便它可以根据配置中的请求设置日志级别。
	util.GetLogger(util.ElectionLogger, "")
	return InitGossipServiceCustomDeliveryFactory(peerIdentity, endpoint, s, certs, &deliveryFactoryImpl{},
		mcs, secAdv, secureDialOpts, bootPeers...)
}

//initgossipserviceCustomDeliveryFactory使用自定义传递工厂初始化八卦服务
//实现，可能对测试和模拟有用
func InitGossipServiceCustomDeliveryFactory(peerIdentity []byte, endpoint string, s *grpc.Server,
	certs *gossipCommon.TLSCertificates, factory DeliveryServiceFactory, mcs api.MessageCryptoService,
	secAdv api.SecurityAdvisor, secureDialOpts api.PeerSecureDialOpts, bootPeers ...string) error {
	var err error
	var gossip gossip.Gossip
	once.Do(func() {
		if overrideEndpoint := viper.GetString("peer.gossip.endpoint"); overrideEndpoint != "" {
			endpoint = overrideEndpoint
		}

		logger.Info("Initialize gossip with endpoint", endpoint, "and bootstrap set", bootPeers)

		gossip, err = integration.NewGossipComponent(peerIdentity, endpoint, s, secAdv,
			mcs, secureDialOpts, certs, bootPeers...)
		gossipServiceInstance = &gossipServiceImpl{
			mcs:             mcs,
			gossipSvc:       gossip,
			privateHandlers: make(map[string]privateHandler),
			chains:          make(map[string]state.GossipStateProvider),
			leaderElection:  make(map[string]election.LeaderElectionService),
			deliveryService: make(map[string]deliverclient.DeliverService),
			deliveryFactory: factory,
			peerIdentity:    peerIdentity,
			secAdv:          secAdv,
		}
	})
	return errors.WithStack(err)
}

//GetGossipsService返回八卦服务的实例
func GetGossipService() GossipService {
	return gossipServiceInstance
}

//distributeprivatedata根据收集策略在通道内分发私有读写集
func (g *gossipServiceImpl) DistributePrivateData(chainID string, txID string, privData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error {
	g.lock.RLock()
	handler, exists := g.privateHandlers[chainID]
	g.lock.RUnlock()
	if !exists {
		return errors.Errorf("No private data handler for %s", chainID)
	}

	if err := handler.distributor.Distribute(txID, privData, blkHt); err != nil {
		logger.Error("Failed to distributed private collection, txID", txID, "channel", chainID, "due to", err)
		return err
	}

	if err := handler.coordinator.StorePvtData(txID, privData, blkHt); err != nil {
		logger.Error("Failed to store private data into transient store, txID",
			txID, "channel", chainID, "due to", err)
		return err
	}
	return nil
}

//newconfigeventer创建一个configprocessor，channelconfig.bundlesource最终可以将配置更新路由到
func (g *gossipServiceImpl) NewConfigEventer() ConfigProcessor {
	return newConfigEventer(g)
}

//支持聚合多个
//八卦服务所需的接口
type Support struct {
	Validator            txvalidator.Validator
	Committer            committer.Committer
	Store                privdata2.TransientStore
	Cs                   privdata.CollectionStore
	IdDeserializeFactory privdata2.IdentityDeserializerFactory
}

//数据存储支持聚合接口
//处理传入块或私有数据
type DataStoreSupport struct {
	committer.Committer
	privdata2.TransientStore
}

//InitializeChannel分配状态提供程序，每次执行时应在每个通道调用一次。
func (g *gossipServiceImpl) InitializeChannel(chainID string, endpoints []string, support Support) {
	g.lock.Lock()
	defer g.lock.Unlock()
//为给定提交者初始化新的状态提供程序
	logger.Debug("Creating state provider for chainID", chainID)
	servicesAdapter := &state.ServicesMediator{GossipAdapter: g, MCSAdapter: g.mcs}

//嵌入临时存储和提交者API以实现
//用于捕获检索能力的数据存储接口
//私人数据
	storeSupport := &DataStoreSupport{
		TransientStore: support.Store,
		Committer:      support.Committer,
	}
//初始化私有数据获取程序
	dataRetriever := privdata2.NewDataRetriever(storeSupport)
	collectionAccessFactory := privdata2.NewCollectionAccessFactory(support.IdDeserializeFactory)
	fetcher := privdata2.NewPuller(support.Cs, g.gossipSvc, dataRetriever, collectionAccessFactory, chainID)

	coordinator := privdata2.NewCoordinator(privdata2.Support{
		ChainID:         chainID,
		CollectionStore: support.Cs,
		Validator:       support.Validator,
		TransientStore:  support.Store,
		Committer:       support.Committer,
		Fetcher:         fetcher,
	}, g.createSelfSignedData())

	reconcilerConfig := privdata2.GetReconcilerConfig()
	var reconciler privdata2.PvtDataReconciler

	if reconcilerConfig.IsEnabled {
		reconciler = privdata2.NewReconciler(support.Committer, fetcher, reconcilerConfig)
	} else {
		reconciler = &privdata2.NoOpReconciler{}
	}

	g.privateHandlers[chainID] = privateHandler{
		support:     support,
		coordinator: coordinator,
		distributor: privdata2.NewDistributor(chainID, g, collectionAccessFactory),
		reconciler:  reconciler,
	}
	g.privateHandlers[chainID].reconciler.Start()

	g.chains[chainID] = state.NewGossipStateProvider(chainID, servicesAdapter, coordinator)
	if g.deliveryService[chainID] == nil {
		var err error
		g.deliveryService[chainID], err = g.deliveryFactory.Service(g, endpoints, g.mcs)
		if err != nil {
			logger.Warningf("Cannot create delivery client, due to %+v", errors.WithStack(err))
		}
	}

//仅当无法连接时，传递服务可能为零
//到订购服务
	if g.deliveryService[chainID] != nil {
//参数：
//-peer.八卦.useLeaderRelation
//-peer.八卦.orgleader
//
//是互斥的，不定义将两者都设置为真，因此
//同伴会恐慌并终止
		leaderElection := viper.GetBool("peer.gossip.useLeaderElection")
		isStaticOrgLeader := viper.GetBool("peer.gossip.orgLeader")

		if leaderElection && isStaticOrgLeader {
			logger.Panic("Setting both orgLeader and useLeaderElection to true isn't supported, aborting execution")
		}

		if leaderElection {
			logger.Debug("Delivery uses dynamic leader election mechanism, channel", chainID)
			g.leaderElection[chainID] = g.newLeaderElectionComponent(chainID, g.onStatusChangeFactory(chainID, support.Committer))
		} else if isStaticOrgLeader {
			logger.Debug("This peer is configured to connect to ordering service for blocks delivery, channel", chainID)
			g.deliveryService[chainID].StartDeliverForChannel(chainID, support.Committer, func() {})
		} else {
			logger.Debug("This peer is not configured to connect to ordering service for blocks delivery, channel", chainID)
		}
	} else {
		logger.Warning("Delivery client is down won't be able to pull blocks for chain", chainID)
	}
}

func (g *gossipServiceImpl) createSelfSignedData() common.SignedData {
	msg := make([]byte, 32)
	sig, err := g.mcs.Sign(msg)
	if err != nil {
		logger.Panicf("Failed creating self signed data because message signing failed: %v", err)
	}
	return common.SignedData{
		Data:      msg,
		Signature: sig,
		Identity:  g.peerIdentity,
	}
}

//updateCanchors构造一个joinChannelMessage并将其发送到gossipsvc
func (g *gossipServiceImpl) updateAnchors(config Config) {
	myOrg := string(g.secAdv.OrgByPeerIdentity(api.PeerIdentityType(g.peerIdentity)))
	if !g.amIinChannel(myOrg, config) {
		logger.Error("Tried joining channel", config.ChainID(), "but our org(", myOrg, "), isn't "+
			"among the orgs of the channel:", orgListFromConfig(config), ", aborting.")
		return
	}
	jcm := &joinChannelMessage{seqNum: config.Sequence(), members2AnchorPeers: map[string][]api.AnchorPeer{}}
	for _, appOrg := range config.Organizations() {
		logger.Debug(appOrg.MSPID(), "anchor peers:", appOrg.AnchorPeers())
		jcm.members2AnchorPeers[appOrg.MSPID()] = []api.AnchorPeer{}
		for _, ap := range appOrg.AnchorPeers() {
			anchorPeer := api.AnchorPeer{
				Host: ap.Host,
				Port: int(ap.Port),
			}
			jcm.members2AnchorPeers[appOrg.MSPID()] = append(jcm.members2AnchorPeers[appOrg.MSPID()], anchorPeer)
		}
	}

//为给定提交者初始化新的状态提供程序
	logger.Debug("Creating state provider for chainID", config.ChainID())
	g.JoinChan(jcm, gossipCommon.ChainID(config.ChainID()))
}

func (g *gossipServiceImpl) updateEndpoints(chainID string, endpoints []string) {
	if ds, ok := g.deliveryService[chainID]; ok {
		logger.Debugf("Updating endpoints for chainID", chainID)
		if err := ds.UpdateEndpoints(chainID, endpoints); err != nil {
//失败的唯一原因是缺少块提供程序
//对于给定的通道ID，因此打印警告就足够了
			logger.Warningf("Failed to update ordering service endpoints, due to %s", err)
		}
	}
}

//对于给定的链，addPayload将消息有效负载附加到
func (g *gossipServiceImpl) AddPayload(chainID string, payload *gproto.Payload) error {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.chains[chainID].AddPayload(payload)
}

//停止停止八卦组件
func (g *gossipServiceImpl) Stop() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for chainID := range g.chains {
		logger.Info("Stopping chain", chainID)
		if le, exists := g.leaderElection[chainID]; exists {
			logger.Infof("Stopping leader election for %s", chainID)
			le.Stop()
		}
		g.chains[chainID].Stop()
		g.privateHandlers[chainID].close()

		if g.deliveryService[chainID] != nil {
			g.deliveryService[chainID].Stop()
		}
	}
	g.gossipSvc.Stop()
}

func (g *gossipServiceImpl) newLeaderElectionComponent(chainID string, callback func(bool)) election.LeaderElectionService {
	PKIid := g.mcs.GetPKIidOfCert(g.peerIdentity)
	adapter := election.NewAdapter(g, PKIid, gossipCommon.ChainID(chainID))
	return election.NewLeaderElectionService(adapter, string(PKIid), callback)
}

func (g *gossipServiceImpl) amIinChannel(myOrg string, config Config) bool {
	for _, orgName := range orgListFromConfig(config) {
		if orgName == myOrg {
			return true
		}
	}
	return false
}

func (g *gossipServiceImpl) onStatusChangeFactory(chainID string, committer blocksprovider.LedgerInfo) func(bool) {
	return func(isLeader bool) {
		if isLeader {
			yield := func() {
				g.lock.RLock()
				le := g.leaderElection[chainID]
				g.lock.RUnlock()
				le.Yield()
			}
			logger.Info("Elected as a leader, starting delivery service for channel", chainID)
			if err := g.deliveryService[chainID].StartDeliverForChannel(chainID, committer, yield); err != nil {
				logger.Errorf("Delivery service is not able to start blocks delivery for chain, due to %+v", errors.WithStack(err))
			}
		} else {
			logger.Info("Renounced leadership, stopping delivery service for channel", chainID)
			if err := g.deliveryService[chainID].StopDeliverForChannel(chainID); err != nil {
				logger.Errorf("Delivery service is not able to stop blocks delivery for chain, due to %+v", errors.WithStack(err))
			}

		}

	}
}

func orgListFromConfig(config Config) []string {
	var orgList []string
	for _, appOrg := range config.Organizations() {
		orgList = append(orgList, appOrg.MSPID())
	}
	return orgList
}
