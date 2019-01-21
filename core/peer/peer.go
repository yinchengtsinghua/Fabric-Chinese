
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


package peer

import (
	"fmt"
	"net"
	"runtime"
	"sync"

	"github.com/hyperledger/fabric/common/channelconfig"
	cc "github.com/hyperledger/fabric/common/config"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/deliver"
	"github.com/hyperledger/fabric/common/flogging"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	fileledger "github.com/hyperledger/fabric/common/ledger/blockledger/file"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/customtx"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/hyperledger/fabric/token/tms/manager"
	"github.com/hyperledger/fabric/token/transaction"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/sync/semaphore"
)

var peerLogger = flogging.MustGetLogger("peer")

var peerServer *comm.GRPCServer

var configTxProcessor = newConfigTxProcessor()
var tokenTxProcessor = &transaction.Processor{
	TMSManager: &manager.Manager{
		IdentityDeserializerManager: &manager.FabricIdentityDeserializerManager{}}}
var ConfigTxProcessors = customtx.Processors{
	common.HeaderType_CONFIG:            configTxProcessor,
	common.HeaderType_TOKEN_TRANSACTION: tokenTxProcessor,
}

//用于跨通道配置更改管理对等机凭据的单实例
var credSupport = comm.GetCredentialSupport()

type gossipSupport struct {
	channelconfig.Application
	configtx.Validator
	channelconfig.Channel
}

type chainSupport struct {
	bundleSource *channelconfig.BundleSource
	channelconfig.Resources
	channelconfig.Application
	ledger ledger.PeerLedger
}

var TransientStoreFactory = &storeProvider{stores: make(map[string]transientstore.Store)}

type storeProvider struct {
	stores map[string]transientstore.Store
	transientstore.StoreProvider
	sync.RWMutex
}

func (sp *storeProvider) StoreForChannel(channel string) transientstore.Store {
	sp.RLock()
	defer sp.RUnlock()
	return sp.stores[channel]
}

func (sp *storeProvider) OpenStore(ledgerID string) (transientstore.Store, error) {
	sp.Lock()
	defer sp.Unlock()
	if sp.StoreProvider == nil {
		sp.StoreProvider = transientstore.NewStoreProvider()
	}
	store, err := sp.StoreProvider.OpenStore(ledgerID)
	if err == nil {
		sp.stores[ledgerID] = store
	}
	return store, err
}

func (cs *chainSupport) Apply(configtx *common.ConfigEnvelope) error {
	err := cs.ConfigtxValidator().Validate(configtx)
	if err != nil {
		return err
	}

//如果正在模拟链支撑，则此字段将为零。
	if cs.bundleSource != nil {
		bundle, err := channelconfig.NewBundle(cs.ConfigtxValidator().ChainID(), configtx.Config)
		if err != nil {
			return err
		}

		channelconfig.LogSanityChecks(bundle)

		err = cs.bundleSource.ValidateNew(bundle)
		if err != nil {
			return err
		}

		capabilitiesSupportedOrPanic(bundle)

		cs.bundleSource.Update(bundle)
	}
	return nil
}

func capabilitiesSupportedOrPanic(res channelconfig.Resources) {
	ac, ok := res.ApplicationConfig()
	if !ok {
		peerLogger.Panicf("[channel %s] does not have application config so is incompatible", res.ConfigtxValidator().ChainID())
	}

	if err := ac.Capabilities().Supported(); err != nil {
		peerLogger.Panicf("[channel %s] incompatible: %s", res.ConfigtxValidator().ChainID(), err)
	}

	if err := res.ChannelConfig().Capabilities().Supported(); err != nil {
		peerLogger.Panicf("[channel %s] incompatible: %s", res.ConfigtxValidator().ChainID(), err)
	}
}

func (cs *chainSupport) Ledger() ledger.PeerLedger {
	return cs.ledger
}

func (cs *chainSupport) GetMSPIDs(cid string) []string {
	return GetMSPIDs(cid)
}

//序列传递给基础configtx.validator
func (cs *chainSupport) Sequence() uint64 {
	sb := cs.bundleSource.StableBundle()
	return sb.ConfigtxValidator().Sequence()
}

//读卡器返回要从分类帐中读取的迭代器
func (cs *chainSupport) Reader() blockledger.Reader {
	return fileledger.NewFileLedger(fileLedgerBlockStore{cs.ledger})
}

//ERRORED返回一个可用于确定
//如果备份资源出错。在这个时候，
//对等端没有导致
//此函数表示发生了错误。
func (cs *chainSupport) Errored() <-chan struct{} {
	return nil
}

//链是管理链中对象的本地结构
type chain struct {
	cs        *chainSupport
	cb        *common.Block
	committer committer.Committer
}

//chains是chainID->chainObject的本地映射
var chains = struct {
	sync.RWMutex
	list map[string]*chain
}{list: make(map[string]*chain)}

var chainInitializer func(string)

var pluginMapper txvalidator.PluginMapper

var mockMSPIDGetter func(string) []string

func MockSetMSPIDGetter(mspIDGetter func(string) []string) {
	mockMSPIDGetter = mspIDGetter
}

//validationWorkersSemaphore是用于确保
//没有太多的并发Tx验证goroutine
var validationWorkersSemaphore *semaphore.Weighted

//初始化设置对等机从持久性中拥有的任何链。这个
//当分类帐和闲话发生时，应在启动时调用函数
//准备好的
func Initialize(init func(string), ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider,
	pm txvalidator.PluginMapper, pr *platforms.Registry, deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
	membershipProvider ledger.MembershipInfoProvider, metricsProvider metrics.Provider) {
	nWorkers := viper.GetInt("peer.validatorPoolSize")
	if nWorkers <= 0 {
		nWorkers = runtime.NumCPU()
	}
	validationWorkersSemaphore = semaphore.NewWeighted(int64(nWorkers))

	pluginMapper = pm
	chainInitializer = init

	var cb *common.Block
	var ledger ledger.PeerLedger
	ledgermgmt.Initialize(&ledgermgmt.Initializer{
		CustomTxProcessors:            ConfigTxProcessors,
		PlatformRegistry:              pr,
		DeployedChaincodeInfoProvider: deployedCCInfoProvider,
		MembershipInfoProvider:        membershipProvider,
		MetricsProvider:               metricsProvider,
	})
	ledgerIds, err := ledgermgmt.GetLedgerIDs()
	if err != nil {
		panic(fmt.Errorf("Error in initializing ledgermgmt: %s", err))
	}
	for _, cid := range ledgerIds {
		peerLogger.Infof("Loading chain %s", cid)
		if ledger, err = ledgermgmt.OpenLedger(cid); err != nil {
			peerLogger.Warningf("Failed to load ledger %s(%s)", cid, err)
			peerLogger.Debugf("Error while loading ledger %s with message %s. We continue to the next ledger rather than abort.", cid, err)
			continue
		}
		if cb, err = getCurrConfigBlockFromLedger(ledger); err != nil {
			peerLogger.Warningf("Failed to find config block on ledger %s(%s)", cid, err)
			peerLogger.Debugf("Error while looking for config block on ledger %s with message %s. We continue to the next ledger rather than abort.", cid, err)
			continue
		}
//如果使用配置块获得有效的分类帐，则创建链
		if err = createChain(cid, ledger, cb, ccp, sccp, pm); err != nil {
			peerLogger.Warningf("Failed to load chain %s(%s)", cid, err)
			peerLogger.Debugf("Error reloading chain %s with message %s. We continue to the next chain rather than abort.", cid, err)
			continue
		}

		InitChain(cid)
	}
}

//initchain负责在对等连接后初始化链，例如部署系统ccs
func InitChain(cid string) {
	if chainInitializer != nil {
//初始化链码，即部署系统CC
		peerLogger.Debugf("Initializing channel %s", cid)
		chainInitializer(cid)
	}
}

func getCurrConfigBlockFromLedger(ledger ledger.PeerLedger) (*common.Block, error) {
	peerLogger.Debugf("Getting config block")

//获取最后一个块。最后一个块号是height-1
	blockchainInfo, err := ledger.GetBlockchainInfo()
	if err != nil {
		return nil, err
	}
	lastBlock, err := ledger.GetBlockByNumber(blockchainInfo.Height - 1)
	if err != nil {
		return nil, err
	}

//从最后一个块元数据获取最新的配置块位置
	configBlockIndex, err := utils.GetLastConfigIndexFromBlock(lastBlock)
	if err != nil {
		return nil, err
	}

//获取最新配置块
	configBlock, err := ledger.GetBlockByNumber(configBlockIndex)
	if err != nil {
		return nil, err
	}

	peerLogger.Debugf("Got config block[%d]", configBlockIndex)
	return configBlock, nil
}

//CreateChain创建新的链对象并将其插入链中
func createChain(cid string, ledger ledger.PeerLedger, cb *common.Block, ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider, pm txvalidator.PluginMapper) error {
	chanConf, err := retrievePersistedChannelConfig(ledger)
	if err != nil {
		return err
	}

	var bundle *channelconfig.Bundle

	if chanConf != nil {
		bundle, err = channelconfig.NewBundle(cid, chanConf)
		if err != nil {
			return err
		}
	} else {
//配置只存储在从v1.1二进制文件开始的statedb中
//因此，如果在那里找不到配置，请从配置块手动提取它
		envelopeConfig, err := utils.ExtractEnvelope(cb, 0)
		if err != nil {
			return err
		}

		bundle, err = channelconfig.NewBundleFromEnvelope(envelopeConfig)
		if err != nil {
			return err
		}
	}

	capabilitiesSupportedOrPanic(bundle)

	channelconfig.LogSanityChecks(bundle)

	gossipEventer := service.GetGossipService().NewConfigEventer()

	gossipCallbackWrapper := func(bundle *channelconfig.Bundle) {
		ac, ok := bundle.ApplicationConfig()
		if !ok {
//TODO，更优雅地处理丢失的applicationconfig
			ac = nil
		}
		gossipEventer.ProcessConfigUpdate(&gossipSupport{
			Validator:   bundle.ConfigtxValidator(),
			Application: ac,
			Channel:     bundle.ChannelConfig(),
		})
		service.GetGossipService().SuspectPeers(func(identity api.PeerIdentityType) bool {
//托多：这是一个以某种方式使MSP层可疑的位置持有者。
//给定的证书被吊销，或其中间CA被吊销。
//同时，在我们拥有这样的能力之前，我们会依次回归
//怀疑所有身份以验证所有身份。
			return true
		})
	}

	trustedRootsCallbackWrapper := func(bundle *channelconfig.Bundle) {
		updateTrustedRoots(bundle)
	}

	mspCallback := func(bundle *channelconfig.Bundle) {
//当所有对mspmgmt的引用从对等代码中消失后，TODO将删除
		mspmgmt.XXXSetMSPManager(cid, bundle.MSPManager())
	}

	ac, ok := bundle.ApplicationConfig()
	if !ok {
		ac = nil
	}

	cs := &chainSupport{
Application: ac, //TODO，重构，因为可以通过管理器访问它
		ledger:      ledger,
	}

	peerSingletonCallback := func(bundle *channelconfig.Bundle) {
		ac, ok := bundle.ApplicationConfig()
		if !ok {
			ac = nil
		}
		cs.Application = ac
		cs.Resources = bundle
	}

	cs.bundleSource = channelconfig.NewBundleSource(
		bundle,
		gossipCallbackWrapper,
		trustedRootsCallbackWrapper,
		mspCallback,
		peerSingletonCallback,
	)

	vcs := struct {
		*chainSupport
		*semaphore.Weighted
	}{cs, validationWorkersSemaphore}
	validator := txvalidator.NewTxValidator(cid, vcs, sccp, pm)
	c := committer.NewLedgerCommitterReactive(ledger, func(block *common.Block) error {
		chainID, err := utils.GetChainIDFromBlock(block)
		if err != nil {
			return err
		}
		return SetCurrConfigBlock(block, chainID)
	})

	ordererAddresses := bundle.ChannelConfig().OrdererAddresses()
	if len(ordererAddresses) == 0 {
		return errors.New("no ordering service endpoint provided in configuration block")
	}

//TODO:有人需要在对等机关闭时对TransientStoreFactory调用close（）吗？
	store, err := TransientStoreFactory.OpenStore(bundle.ConfigtxValidator().ChainID())
	if err != nil {
		return errors.Wrapf(err, "[channel %s] failed opening transient store", bundle.ConfigtxValidator().ChainID())
	}
	csStoreSupport := &CollectionSupport{
		PeerLedger: ledger,
	}
	simpleCollectionStore := privdata.NewSimpleCollectionStore(csStoreSupport)

	service.GetGossipService().InitializeChannel(bundle.ConfigtxValidator().ChainID(), ordererAddresses, service.Support{
		Validator:            validator,
		Committer:            c,
		Store:                store,
		Cs:                   simpleCollectionStore,
		IdDeserializeFactory: csStoreSupport,
	})

	chains.Lock()
	defer chains.Unlock()
	chains.list[cid] = &chain{
		cs:        cs,
		cb:        cb,
		committer: c,
	}

	return nil
}

//CreateChainFromBlock从配置块创建新链
func CreateChainFromBlock(cb *common.Block, ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider) error {
	cid, err := utils.GetChainIDFromBlock(cb)
	if err != nil {
		return err
	}

	var l ledger.PeerLedger
	if l, err = ledgermgmt.CreateLedger(cb); err != nil {
		return errors.WithMessage(err, "cannot create ledger from genesis block")
	}

	return createChain(cid, l, cb, ccp, sccp, pluginMapper)
}

//GetLedger返回具有链ID的链的分类帐。请注意
//如果尚未创建链cid，则调用返回nil。
func GetLedger(cid string) ledger.PeerLedger {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cs.ledger
	}
	return nil
}

//getstablechannelconfig返回具有通道ID的链的稳定通道配置。
//注意，如果尚未创建链cid，则此调用返回nil。
func GetStableChannelConfig(cid string) channelconfig.Resources {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cs.bundleSource.StableBundle()
	}
	return nil
}

//getchannelconfig返回具有通道ID的链的通道配置。请注意
//如果尚未创建链cid，则调用返回nil。
func GetChannelConfig(cid string) channelconfig.Resources {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cs
	}
	return nil
}

//GetPolicyManager返回具有链ID的链的策略管理器。请注意
//如果尚未创建链cid，则调用返回nil。
func GetPolicyManager(cid string) policies.Manager {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cs.PolicyManager()
	}
	return nil
}

//getcurrconfigblock返回指定链的缓存配置块。
//注意，如果尚未创建链cid，则此调用返回nil。
func GetCurrConfigBlock(cid string) *common.Block {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cb
	}
	return nil
}

//根据对通道的更新更新更新对等端的受信任根目录
func updateTrustedRoots(cm channelconfig.Resources) {
//这是根据每个通道触发的，因此首先更新通道的根
	peerLogger.Debugf("Updating trusted root authorities for channel %s", cm.ConfigtxValidator().ChainID())
	var serverConfig comm.ServerConfig
	var err error
//只有运行时启用了TLS
	serverConfig, err = GetServerConfig()
	if err == nil && serverConfig.SecOpts.UseTLS {
		buildTrustedRootsForChain(cm)

//现在遍历所有应用程序和订购程序链的所有根
		trustedRoots := [][]byte{}
		credSupport.RLock()
		defer credSupport.RUnlock()
		for _, roots := range credSupport.AppRootCAsByChain {
			trustedRoots = append(trustedRoots, roots...)
		}
//还需要附加静态配置的根证书
		if len(serverConfig.SecOpts.ClientRootCAs) > 0 {
			trustedRoots = append(trustedRoots, serverConfig.SecOpts.ClientRootCAs...)
		}
		if len(serverConfig.SecOpts.ServerRootCAs) > 0 {
			trustedRoots = append(trustedRoots, serverConfig.SecOpts.ServerRootCAs...)
		}

		server := peerServer
//现在更新对等服务器的客户机根目录
		if server != nil {
			err := server.SetClientRootCAs(trustedRoots)
			if err != nil {
				msg := "Failed to update trusted roots for peer from latest config " +
					"block.  This peer may not be able to communicate " +
					"with members of channel %s (%s)"
				peerLogger.Warningf(msg, cm.ConfigtxValidator().ChainID(), err)
			}
		}
	}
}

//通过获取
//与mspmanager关联的所有msp的根证书和中间证书
func buildTrustedRootsForChain(cm channelconfig.Resources) {
	credSupport.Lock()
	defer credSupport.Unlock()

	appRootCAs := [][]byte{}
	ordererRootCAs := [][]byte{}
	appOrgMSPs := make(map[string]struct{})
	ordOrgMSPs := make(map[string]struct{})

	if ac, ok := cm.ApplicationConfig(); ok {
//循环应用程序组织并构建MSPIDS地图
		for _, appOrg := range ac.Organizations() {
			appOrgMSPs[appOrg.MSPID()] = struct{}{}
		}
	}

	if ac, ok := cm.OrdererConfig(); ok {
//通过订购者组织循环并构建MSPIDS地图
		for _, ordOrg := range ac.Organizations() {
			ordOrgMSPs[ordOrg.MSPID()] = struct{}{}
		}
	}

	cid := cm.ConfigtxValidator().ChainID()
	peerLogger.Debugf("updating root CAs for channel [%s]", cid)
	msps, err := cm.MSPManager().GetMSPs()
	if err != nil {
		peerLogger.Errorf("Error getting root CAs for channel %s (%s)", cid, err)
	}
	if err == nil {
		for k, v := range msps {
//检查这是否是织物MSP
			if v.GetType() == msp.FABRIC {
				for _, root := range v.GetTLSRootCerts() {
//查看这是一个应用程序组织MSP
					if _, ok := appOrgMSPs[k]; ok {
						peerLogger.Debugf("adding app root CAs for MSP [%s]", k)
						appRootCAs = append(appRootCAs, root)
					}
//查看这是一个订购者组织MSP
					if _, ok := ordOrgMSPs[k]; ok {
						peerLogger.Debugf("adding orderer root CAs for MSP [%s]", k)
						ordererRootCAs = append(ordererRootCAs, root)
					}
				}
				for _, intermediate := range v.GetTLSIntermediateCerts() {
//查看这是一个应用程序组织MSP
					if _, ok := appOrgMSPs[k]; ok {
						peerLogger.Debugf("adding app root CAs for MSP [%s]", k)
						appRootCAs = append(appRootCAs, intermediate)
					}
//查看这是一个订购者组织MSP
					if _, ok := ordOrgMSPs[k]; ok {
						peerLogger.Debugf("adding orderer root CAs for MSP [%s]", k)
						ordererRootCAs = append(ordererRootCAs, intermediate)
					}
				}
			}
		}
		credSupport.AppRootCAsByChain[cid] = appRootCAs
		credSupport.OrdererRootCAsByChain[cid] = ordererRootCAs
	}
}

//getmspids返回在此链上定义的每个应用程序msp的ID
func GetMSPIDs(cid string) []string {
	chains.RLock()
	defer chains.RUnlock()

//如果设置了mock，则使用它返回mspids
//用于没有正确连接的测试
	if mockMSPIDGetter != nil {
		return mockMSPIDGetter(cid)
	}
	if c, ok := chains.list[cid]; ok {
		if c == nil || c.cs == nil {
			return nil
		}
		ac, ok := c.cs.ApplicationConfig()
		if !ok || ac.Organizations() == nil {
			return nil
		}

		orgs := ac.Organizations()
		toret := make([]string, len(orgs))
		i := 0
		for _, org := range orgs {
			toret[i] = org.MSPID()
			i++
		}

		return toret
	}
	return nil
}

//setcurrconfigblock设置指定通道的当前配置块
func SetCurrConfigBlock(block *common.Block, cid string) error {
	chains.Lock()
	defer chains.Unlock()
	if c, ok := chains.list[cid]; ok {
		c.cb = block
		return nil
	}
	return errors.Errorf("[channel %s] channel not associated with this peer", cid)
}

//getlocalip返回主机的非环回本地IP
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
//检查地址类型，如果不是环回，则显示它
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

//getchannelsinfo返回一个数组，其中包含有关
//这个对等体
func GetChannelsInfo() []*pb.ChannelInfo {
//存储所有通道元数据的数组
	var channelInfoArray []*pb.ChannelInfo

	chains.RLock()
	defer chains.RUnlock()
	for key := range chains.list {
		channelInfo := &pb.ChannelInfo{ChannelId: key}

//将此特定链码的元数据添加到所有链码的数组中
		channelInfoArray = append(channelInfoArray, channelInfo)
	}

	return channelInfoArray
}

//NewChannelPolicyManagergetter返回ChannelPolicyManagergetter的新实例
func NewChannelPolicyManagerGetter() policies.ChannelPolicyManagerGetter {
	return &channelPolicyManagerGetter{}
}

type channelPolicyManagerGetter struct{}

func (c *channelPolicyManagerGetter) Manager(channelID string) (policies.Manager, bool) {
	policyManager := GetPolicyManager(channelID)
	return policyManager, policyManager != nil
}

//newpeerserver创建comm.grpcserver的实例
//此服务器用于对等通信
func NewPeerServer(listenAddress string, serverConfig comm.ServerConfig) (*comm.GRPCServer, error) {
	var err error
	peerServer, err = comm.NewGRPCServer(listenAddress, serverConfig)
	if err != nil {
		peerLogger.Errorf("Failed to create peer server (%s)", err)
		return nil, err
	}
	return peerServer, nil
}

//TODO:删除集合支持及其各自的方法数。
//CollectionSupport是按链创建的，并传递给Simple
//collection store and the gossip. As it is created per chain, there
//不需要为getQueryExecutorForLedger（）传递channelID
//和GetIdentityDeserializer（）。注意，CID传递给
//从未使用GetQueryExecutorForLedger。相反，我们可以直接
//将ledger.peerledger和msp.identityDeserializer传递给
//SimpleCollectionStore并仅将msp.IdentityDeserializer传递给
//createchain（）中的流言--fab-13037
type CollectionSupport struct {
	ledger.PeerLedger
}

func (cs *CollectionSupport) GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error) {
	return cs.NewQueryExecutor()
}

func (*CollectionSupport) GetIdentityDeserializer(chainID string) msp.IdentityDeserializer {
	return mspmgmt.GetManagerForChain(chainID)
}

//
//为对等端提供服务支持结构
//

//DeliverChainManager提供对执行传递的通道的访问
type DeliverChainManager struct {
}

func (DeliverChainManager) GetChain(chainID string) deliver.Chain {
	channel, ok := chains.list[chainID]
	if !ok {
		return nil
	}
	return channel.cs
}

//FileLedgerBlockStore实现预期的接口
//公用/分类帐/分块分类帐/与文件分类帐交互以进行交付的文件
type fileLedgerBlockStore struct {
	ledger.PeerLedger
}

func (flbs fileLedgerBlockStore) AddBlock(*common.Block) error {
	return nil
}

func (flbs fileLedgerBlockStore) RetrieveBlocks(startBlockNumber uint64) (commonledger.ResultsIterator, error) {
	return flbs.GetBlocksIterator(startBlockNumber)
}

//NewConfigSupport返回
func NewConfigSupport() cc.Manager {
	return &configSupport{}
}

type configSupport struct {
}

//getchannelconfig返回表示
//指定通道的当前通道配置树。这个
//返回对象的configProto方法可用于获取
//表示通道配置的proto。
func (*configSupport) GetChannelConfig(channel string) cc.Config {
	chains.RLock()
	defer chains.RUnlock()
	chain := chains.list[channel]
	if chain == nil {
		peerLogger.Errorf("[channel %s] channel not associated with this peer", channel)
		return nil
	}
	return chain.cs.bundleSource.ConfigtxValidator()
}
