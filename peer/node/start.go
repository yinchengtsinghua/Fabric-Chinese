
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


package node

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	ccdef "github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/deliver"
	"github.com/hyperledger/fabric/common/flogging"
	floggingmetrics "github.com/hyperledger/fabric/common/flogging/metrics"
	"github.com/hyperledger/fabric/common/grpclogging"
	"github.com/hyperledger/fabric/common/grpcmetrics"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/metadata"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/viperutil"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/admin"
	cc "github.com/hyperledger/fabric/core/cclifecycle"
	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/chaincode/accesscontrol"
	"github.com/hyperledger/fabric/core/chaincode/lifecycle"
	"github.com/hyperledger/fabric/core/chaincode/persistence"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/car"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/chaincode/platforms/java"
	"github.com/hyperledger/fabric/core/chaincode/platforms/node"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/dockercontroller"
	"github.com/hyperledger/fabric/core/container/inproccontroller"
	"github.com/hyperledger/fabric/core/endorser"
	authHandler "github.com/hyperledger/fabric/core/handlers/auth"
	endorsement2 "github.com/hyperledger/fabric/core/handlers/endorsement/api"
	endorsement3 "github.com/hyperledger/fabric/core/handlers/endorsement/api/identities"
	"github.com/hyperledger/fabric/core/handlers/library"
	validation "github.com/hyperledger/fabric/core/handlers/validation/api"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/operations"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/scc"
	"github.com/hyperledger/fabric/core/scc/cscc"
	"github.com/hyperledger/fabric/core/scc/lscc"
	"github.com/hyperledger/fabric/core/scc/qscc"
	"github.com/hyperledger/fabric/discovery"
	"github.com/hyperledger/fabric/discovery/endorsement"
	discsupport "github.com/hyperledger/fabric/discovery/support"
	discacl "github.com/hyperledger/fabric/discovery/support/acl"
	ccsupport "github.com/hyperledger/fabric/discovery/support/chaincode"
	"github.com/hyperledger/fabric/discovery/support/config"
	"github.com/hyperledger/fabric/discovery/support/gossip"
	gossipcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	peergossip "github.com/hyperledger/fabric/peer/gossip"
	"github.com/hyperledger/fabric/peer/version"
	cb "github.com/hyperledger/fabric/protos/common"
	common2 "github.com/hyperledger/fabric/protos/common"
	discprotos "github.com/hyperledger/fabric/protos/discovery"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/token"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/hyperledger/fabric/token/server"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const (
	chaincodeAddrKey       = "peer.chaincodeAddress"
	chaincodeListenAddrKey = "peer.chaincodeListenAddress"
	defaultChaincodePort   = 7052
	grpcMaxConcurrency     = 2500
)

var chaincodeDevMode bool

func startCmd() *cobra.Command {
//在node start命令上设置标志。
	flags := nodeStartCmd.Flags()
	flags.BoolVarP(&chaincodeDevMode, "peer-chaincodedev", "", false,
		"Whether peer in chaincode development mode")

	return nodeStartCmd
}

var nodeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the node.",
	Long:  `Starts a node that interacts with the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
//对命令行的分析已完成，因此沉默命令用法
		cmd.SilenceUsage = true
		return serve(args)
	},
}

func serve(args []string) error {
//目前，对等机只与标准MSP一起工作
//因为在某些情况下，MSP必须确保
//从一个凭证中，您只有一个“标识”。
//Idemix还不支持这个*但是它很容易
//固定以支撑它。现在，我们只需确保
//对等端只提供标准MSP
	mspType := mgmt.GetLocalMSP().GetType()
	if mspType != msp.FABRIC {
		panic("Unsupported msp type " + msp.ProviderTypeToString(mspType))
	}

//使用golang.org/x/net/trace包跟踪rpcs。这是搬出去的
//交付服务连接工厂具有流程范围的含义
//在初始化GRPC客户机和服务器方面也很活跃。
	grpc.EnableTracing = true

	logger.Infof("Starting %s", version.GetInfo())

//使用默认acl提供程序（基于资源和默认1.0策略）启动aclmgmt。
//用户可以将自己的aclprovider传递给registeraclprovider（当前单元测试执行此操作）
	aclProvider := aclmgmt.NewACLProvider(
		aclmgmt.ResourceGetter(peer.GetStableChannelConfig),
	)

	pr := platforms.NewRegistry(
		&golang.Platform{},
		&node.Platform{},
		&java.Platform{},
		&car.Platform{},
	)

	deployedCCInfoProvider := &lscc.DeployedCCInfoProvider{}

	identityDeserializerFactory := func(chainID string) msp.IdentityDeserializer {
		return mgmt.GetManagerForChain(chainID)
	}

	opsSystem := newOperationsSystem()
	err := opsSystem.Start()
	if err != nil {
		return errors.WithMessage(err, "failed to initialize operations subystems")
	}
	defer opsSystem.Stop()

	metricsProvider := opsSystem.Provider
	logObserver := floggingmetrics.NewObserver(metricsProvider)
	flogging.Global.SetObserver(logObserver)

	membershipInfoProvider := privdata.NewMembershipInfoProvider(createSelfSignedData(), identityDeserializerFactory)
//初始化资源管理出口
	ledgermgmt.Initialize(
		&ledgermgmt.Initializer{
			CustomTxProcessors:            peer.ConfigTxProcessors,
			PlatformRegistry:              pr,
			DeployedChaincodeInfoProvider: deployedCCInfoProvider,
			MembershipInfoProvider:        membershipInfoProvider,
			MetricsProvider:               metricsProvider,
			HealthCheckRegistry:           opsSystem,
		},
	)

//必须先处理参数重写，然后才能处理任何参数
//缓存。缓存失败会导致服务器立即终止。
	if chaincodeDevMode {
		logger.Info("Running in chaincode development mode")
		logger.Info("Disable loading validity system chaincode")

		viper.Set("chaincode.mode", chaincode.DevModeUserRunsChaincode)
	}

	if err := peer.CacheConfiguration(); err != nil {
		return err
	}

	peerEndpoint, err := peer.GetPeerEndpoint()
	if err != nil {
		err = fmt.Errorf("Failed to get Peer Endpoint: %s", err)
		return err
	}

	peerHost, _, err := net.SplitHostPort(peerEndpoint.Address)
	if err != nil {
		return fmt.Errorf("peer address is not in the format of host:port: %v", err)
	}

	listenAddr := viper.GetString("peer.listenAddress")
	serverConfig, err := peer.GetServerConfig()
	if err != nil {
		logger.Fatalf("Error loading secure config for peer (%s)", err)
	}

	throttle := comm.NewThrottle(grpcMaxConcurrency)
	serverConfig.Logger = flogging.MustGetLogger("core.comm").With("server", "PeerServer")
	serverConfig.MetricsProvider = metricsProvider
	serverConfig.UnaryInterceptors = append(
		serverConfig.UnaryInterceptors,
		grpcmetrics.UnaryServerInterceptor(grpcmetrics.NewUnaryMetrics(metricsProvider)),
		grpclogging.UnaryServerInterceptor(flogging.MustGetLogger("comm.grpc.server").Zap()),
		throttle.UnaryServerIntercptor,
	)
	serverConfig.StreamInterceptors = append(
		serverConfig.StreamInterceptors,
		grpcmetrics.StreamServerInterceptor(grpcmetrics.NewStreamMetrics(metricsProvider)),
		grpclogging.StreamServerInterceptor(flogging.MustGetLogger("comm.grpc.server").Zap()),
		throttle.StreamServerInterceptor,
	)

	peerServer, err := peer.NewPeerServer(listenAddr, serverConfig)
	if err != nil {
		logger.Fatalf("Failed to create peer server (%s)", err)
	}

	if serverConfig.SecOpts.UseTLS {
		logger.Info("Starting peer with TLS enabled")
//设置凭据支持
		cs := comm.GetCredentialSupport()
		cs.ServerRootCAs = serverConfig.SecOpts.ServerRootCAs

//如果远程终结点请求客户端身份验证，请设置要使用的证书
		clientCert, err := peer.GetClientCertificate()
		if err != nil {
			logger.Fatalf("Failed to set TLS client certificate (%s)", err)
		}
		comm.GetCredentialSupport().SetClientCertificate(clientCert)
	}

	mutualTLS := serverConfig.SecOpts.UseTLS && serverConfig.SecOpts.RequireClientCert
	policyCheckerProvider := func(resourceName string) deliver.PolicyCheckerFunc {
		return func(env *cb.Envelope, channelID string) error {
			return aclProvider.CheckACL(resourceName, channelID, env)
		}
	}

	abServer := peer.NewDeliverEventsServer(mutualTLS, policyCheckerProvider, &peer.DeliverChainManager{}, metricsProvider)
	pb.RegisterDeliverServer(peerServer.Server(), abServer)

//初始化链码服务
	chaincodeSupport, ccp, sccp, packageProvider := startChaincodeServer(peerHost, aclProvider, pr, opsSystem)

	logger.Debugf("Running peer")

//启动管理服务器
	startAdminServer(listenAddr, peerServer.Server(), metricsProvider)

	privDataDist := func(channel string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error {
		return service.GetGossipService().DistributePrivateData(channel, txID, privateData, blkHt)
	}

	signingIdentity := mgmt.GetLocalSigningIdentityOrPanic()
	serializedIdentity, err := signingIdentity.Serialize()
	if err != nil {
		logger.Panicf("Failed serializing self identity: %v", err)
	}

	libConf := library.Config{}
	if err = viperutil.EnhancedExactUnmarshalKey("peer.handlers", &libConf); err != nil {
		return errors.WithMessage(err, "could not load YAML config")
	}
	reg := library.InitRegistry(libConf)

	authFilters := reg.Lookup(library.Auth).([]authHandler.Filter)
	endorserSupport := &endorser.SupportImpl{
		SignerSupport:    signingIdentity,
		Peer:             peer.Default,
		PeerSupport:      peer.DefaultSupport,
		ChaincodeSupport: chaincodeSupport,
		SysCCProvider:    sccp,
		ACLProvider:      aclProvider,
	}
	endorsementPluginsByName := reg.Lookup(library.Endorsement).(map[string]endorsement2.PluginFactory)
	validationPluginsByName := reg.Lookup(library.Validation).(map[string]validation.PluginFactory)
	signingIdentityFetcher := (endorsement3.SigningIdentityFetcher)(endorserSupport)
	channelStateRetriever := endorser.ChannelStateRetriever(endorserSupport)
	pluginMapper := endorser.MapBasedPluginMapper(endorsementPluginsByName)
	pluginEndorser := endorser.NewPluginEndorser(&endorser.PluginSupport{
		ChannelStateRetriever:   channelStateRetriever,
		TransientStoreRetriever: peer.TransientStoreFactory,
		PluginMapper:            pluginMapper,
		SigningIdentityFetcher:  signingIdentityFetcher,
	})
	endorserSupport.PluginEndorser = pluginEndorser
	serverEndorser := endorser.NewEndorserServer(privDataDist, endorserSupport, pr, metricsProvider)
	auth := authHandler.ChainFilters(serverEndorser, authFilters...)
//注册背书服务器
	pb.RegisterEndorserServer(peerServer.Server(), auth)

	policyMgr := peer.NewChannelPolicyManagerGetter()

//初始化八卦组件
	err = initGossipService(policyMgr, peerServer, serializedIdentity, peerEndpoint.Address)
	if err != nil {
		return err
	}
	defer service.GetGossipService().Stop()

//注册验证程序GRPC服务
//FAB-12971在v1.4切割前禁用校准服务。将在v1.4剪切后取消注释
//错误=RegisterProverService（PeerServer、AclProvider、SigningIdentity）
//如果犯错！= nIL{
//返回错误
//}

//初始化系统链代码

//部署系统链代码
	sccp.DeploySysCCs("", ccp)
	logger.Infof("Deployed system chaincodes")

	installedCCs := func() ([]ccdef.InstalledChaincode, error) {
		return packageProvider.ListInstalledChaincodes()
	}
	lifecycle, err := cc.NewLifeCycle(cc.Enumerate(installedCCs))
	if err != nil {
		logger.Panicf("Failed creating lifecycle: +%v", err)
	}
	onUpdate := cc.HandleMetadataUpdate(func(channel string, chaincodes ccdef.MetadataSet) {
		service.GetGossipService().UpdateChaincodes(chaincodes.AsChaincodes(), gossipcommon.ChainID(channel))
	})
	lifecycle.AddListener(onUpdate)

//这会打开所有频道
	peer.Initialize(func(cid string) {
		logger.Debugf("Deploying system CC, for channel <%s>", cid)
		sccp.DeploySysCCs(cid, ccp)
		sub, err := lifecycle.NewChannelSubscription(cid, cc.QueryCreatorFunc(func() (cc.Query, error) {
			return peer.GetLedger(cid).NewQueryExecutor()
		}))
		if err != nil {
			logger.Panicf("Failed subscribing to chaincode lifecycle updates")
		}
		cceventmgmt.GetMgr().Register(cid, sub)
	}, ccp, sccp, txvalidator.MapBasedPluginMapper(validationPluginsByName),
		pr, deployedCCInfoProvider, membershipInfoProvider, metricsProvider)

	if viper.GetBool("peer.discovery.enabled") {
		registerDiscoveryService(peerServer, policyMgr, lifecycle)
	}

	networkID := viper.GetString("peer.networkId")

	logger.Infof("Starting peer with ID=[%s], network ID=[%s], address=[%s]", peerEndpoint.Id, networkID, peerEndpoint.Address)

//在启动Go例程之前获取配置以避免
//赛车试验
	profileEnabled := viper.GetBool("peer.profile.enabled")
	profileListenAddress := viper.GetString("peer.profile.listenAddress")

//启动GRPC服务器。在Goroutine中完成，以便我们可以部署
//如果需要的话，创世块。
	serve := make(chan error)

	go func() {
		var grpcErr error
		if grpcErr = peerServer.Start(); grpcErr != nil {
			grpcErr = fmt.Errorf("grpc server exited with error: %s", grpcErr)
		} else {
			logger.Info("peer server exited")
		}
		serve <- grpcErr
	}()

//如果启用，则开始分析HTTP端点
	if profileEnabled {
		go func() {
			logger.Infof("Starting profiling server with listenAddress = %s", profileListenAddress)
			if profileErr := http.ListenAndServe(profileListenAddress, nil); profileErr != nil {
				logger.Errorf("Error starting profiler: %s", profileErr)
			}
		}()
	}

	go handleSignals(addPlatformSignals(map[os.Signal]func(){
		syscall.SIGINT:  func() { serve <- nil },
		syscall.SIGTERM: func() { serve <- nil },
	}))

	logger.Infof("Started peer with ID=[%s], network ID=[%s], address=[%s]", peerEndpoint.Id, networkID, peerEndpoint.Address)

//阻止，直到GRPC服务器退出
	return <-serve
}

func handleSignals(handlers map[os.Signal]func()) {
	var signals []os.Signal
	for sig := range handlers {
		signals = append(signals, sig)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, signals...)

	for sig := range signalChan {
		logger.Infof("Received signal: %d (%s)", sig, sig)
		handlers[sig]()
	}
}

func localPolicy(policyObject proto.Message) policies.Policy {
	localMSP := mgmt.GetLocalMSP()
	pp := cauthdsl.NewPolicyProvider(localMSP)
	policy, _, err := pp.NewPolicy(utils.MarshalOrPanic(policyObject))
	if err != nil {
		logger.Panicf("Failed creating local policy: +%v", err)
	}
	return policy
}

func createSelfSignedData() common2.SignedData {
	sId := mgmt.GetLocalSigningIdentityOrPanic()
	msg := make([]byte, 32)
	sig, err := sId.Sign(msg)
	if err != nil {
		logger.Panicf("Failed creating self signed data because message signing failed: %v", err)
	}
	peerIdentity, err := sId.Serialize()
	if err != nil {
		logger.Panicf("Failed creating self signed data because peer identity couldn't be serialized: %v", err)
	}
	return common2.SignedData{
		Data:      msg,
		Signature: sig,
		Identity:  peerIdentity,
	}
}

func registerDiscoveryService(peerServer *comm.GRPCServer, polMgr policies.ChannelPolicyManagerGetter, lc *cc.Lifecycle) {
	mspID := viper.GetString("peer.localMspId")
	localAccessPolicy := localPolicy(cauthdsl.SignedByAnyAdmin([]string{mspID}))
	if viper.GetBool("peer.discovery.orgMembersAllowedAccess") {
		localAccessPolicy = localPolicy(cauthdsl.SignedByAnyMember([]string{mspID}))
	}
	channelVerifier := discacl.NewChannelVerifier(policies.ChannelApplicationWriters, polMgr)
	acl := discacl.NewDiscoverySupport(channelVerifier, localAccessPolicy, discacl.ChannelConfigGetterFunc(peer.GetStableChannelConfig))
	gSup := gossip.NewDiscoverySupport(service.GetGossipService())
	ccSup := ccsupport.NewDiscoverySupport(lc)
	ea := endorsement.NewEndorsementAnalyzer(gSup, ccSup, acl, lc)
	confSup := config.NewDiscoverySupport(config.CurrentConfigBlockGetterFunc(peer.GetCurrConfigBlock))
	support := discsupport.NewDiscoverySupport(acl, gSup, ea, confSup, acl)
	svc := discovery.NewService(discovery.Config{
		TLS:                          peerServer.TLSEnabled(),
		AuthCacheEnabled:             viper.GetBool("peer.discovery.authCacheEnabled"),
		AuthCacheMaxSize:             viper.GetInt("peer.discovery.authCacheMaxSize"),
		AuthCachePurgeRetentionRatio: viper.GetFloat64("peer.discovery.authCachePurgeRetentionRatio"),
	}, support)
	logger.Info("Discovery service activated")
	discprotos.RegisterDiscoveryServer(peerServer.Server(), svc)
}

//使用peer.chaincodelistenaddress创建CC侦听器（如果未设置，则使用peer.peeraddress）
func createChaincodeServer(ca tlsgen.CA, peerHostname string) (srv *comm.GRPCServer, ccEndpoint string, err error) {
//在可能设置chaincodelistenaddress之前，首先计算chaincode端点
	ccEndpoint, err = computeChaincodeEndpoint(peerHostname)
	if err != nil {
		if chaincode.IsDevMode() {
//如果dev模式有任何错误，我们使用0.0.0.0:7052
			ccEndpoint = fmt.Sprintf("%s:%d", "0.0.0.0", defaultChaincodePort)
			logger.Warningf("use %s as chaincode endpoint because of error in computeChaincodeEndpoint: %s", ccEndpoint, err)
		} else {
//对于非开发模式，我们必须返回错误
			logger.Errorf("Error computing chaincode endpoint: %s", err)
			return nil, "", err
		}
	}

	host, _, err := net.SplitHostPort(ccEndpoint)
	if err != nil {
		logger.Panic("Chaincode service host", ccEndpoint, "isn't a valid hostname:", err)
	}

	cclistenAddress := viper.GetString(chaincodeListenAddrKey)
	if cclistenAddress == "" {
		cclistenAddress = fmt.Sprintf("%s:%d", peerHostname, defaultChaincodePort)
		logger.Warningf("%s is not set, using %s", chaincodeListenAddrKey, cclistenAddress)
		viper.Set(chaincodeListenAddrKey, cclistenAddress)
	}

	config, err := peer.GetServerConfig()
	if err != nil {
		logger.Errorf("Error getting server config: %s", err)
		return nil, "", err
	}

//为服务器设置记录器
	config.Logger = flogging.MustGetLogger("core.comm").With("server", "ChaincodeServer")

//如果tls适用，则覆盖tls配置
	if config.SecOpts.UseTLS {
//使用与计算的链码端点匹配的SAN创建自签名的TLS证书
		certKeyPair, err := ca.NewServerCertKeyPair(host)
		if err != nil {
			logger.Panicf("Failed generating TLS certificate for chaincode service: +%v", err)
		}
		config.SecOpts = &comm.SecureOptions{
			UseTLS: true,
//需要链码填充程序进行自身身份验证
			RequireClientCert: true,
//仅信任自己签名的客户端证书
			ClientRootCAs: [][]byte{ca.CertBytes()},
//使用我们自己的自签名TLS证书和密钥
			Certificate: certKeyPair.Cert,
			Key:         certKeyPair.Key,
//由于此tls配置仅用于
//GRPC服务器而不是客户机
			ServerRootCAs: nil,
		}
	}

//chaincode keepalive选项-静态
	chaincodeKeepaliveOptions := &comm.KeepaliveOptions{
ServerInterval:    time.Duration(2) * time.Hour,    //2小时-GRPC默认值
ServerTimeout:     time.Duration(20) * time.Second, //20秒-GRPC默认值
ServerMinInterval: time.Duration(1) * time.Minute,  //匹配客户端间隔
	}
	config.KaOpts = chaincodeKeepaliveOptions

	srv, err = comm.NewGRPCServer(cclistenAddress, config)
	if err != nil {
		logger.Errorf("Error creating GRPC server: %s", err)
		return nil, "", err
	}

	return srv, ccEndpoint, nil
}

//ComputechaincodeEndpoint将使用链码地址，链码侦听
//地址（这两个来自viper）和计算链码端点的对等地址。
//计算链码端点可能有以下情况：
//案例A：如果设置了chaincodeAddRkey，则在未设置“0.0.0.0”（或“：：”时使用它）
//案例B：否则，如果设置了chaincodeListenAddRkey而不是“0.0.0.0”或（“：：”），则使用它。
//案例C：如果不是“0.0.0.0”（或“：：”，则使用对等地址）
//案例D：否则返回错误
func computeChaincodeEndpoint(peerHostname string) (ccEndpoint string, err error) {
	logger.Infof("Entering computeChaincodeEndpoint with peerHostname: %s", peerHostname)
//将此设置为链码将解析为的主机/IP。它可能是
//与对等机相同的地址（例如在示例Docker env中使用
//容器名作为主板上的主机名）
	ccEndpoint = viper.GetString(chaincodeAddrKey)
	if ccEndpoint == "" {
//未设置chaincodeAddRkey，请尝试从侦听器获取地址
//（可最终使用对等地址）
		ccEndpoint = viper.GetString(chaincodeListenAddrKey)
		if ccEndpoint == "" {
//案例C：未设置chaincodelistenaddrkey，请使用对等地址
			peerIp := net.ParseIP(peerHostname)
			if peerIp != nil && peerIp.IsUnspecified() {
//案例D：我们只有“0.0.0.0”或“：：”链接代码无法连接到
				logger.Errorf("ChaincodeAddress and chaincodeListenAddress are nil and peerIP is %s", peerIp)
				return "", errors.New("invalid endpoint for chaincode to connect")
			}

//使用对等地址：defaultchaincodeport
			ccEndpoint = fmt.Sprintf("%s:%d", peerHostname, defaultChaincodePort)

		} else {
//案例B：设置了chaincodeListenAddRkey
			host, port, err := net.SplitHostPort(ccEndpoint)
			if err != nil {
				logger.Errorf("ChaincodeAddress is nil and fail to split chaincodeListenAddress: %s", err)
				return "", err
			}

			ccListenerIp := net.ParseIP(host)
//忽略其他值，如多播地址等…如服务器
//无论如何都不会用这个地址开始
			if ccListenerIp != nil && ccListenerIp.IsUnspecified() {
//案例C：如果“0.0.0.0”或“：：”，则必须将对等地址与侦听端口一起使用。
				peerIp := net.ParseIP(peerHostname)
				if peerIp != nil && peerIp.IsUnspecified() {
//案例D：我们只有“0.0.0.0”或“：：”链接代码无法连接到
					logger.Error("ChaincodeAddress is nil while both chaincodeListenAddressIP and peerIP are 0.0.0.0")
					return "", errors.New("invalid endpoint for chaincode to connect")
				}
				ccEndpoint = fmt.Sprintf("%s:%s", peerHostname, port)
			}

		}

	} else {
//案例A：设置了chaincodeaddrkey
		if host, _, err := net.SplitHostPort(ccEndpoint); err != nil {
			logger.Errorf("Fail to split chaincodeAddress: %s", err)
			return "", err
		} else {
			ccIP := net.ParseIP(host)
			if ccIP != nil && ccIP.IsUnspecified() {
				logger.Errorf("ChaincodeAddress' IP cannot be %s in non-dev mode", ccIP)
				return "", errors.New("invalid endpoint for chaincode to connect")
			}
		}
	}

	logger.Infof("Exit with ccEndpoint: %s", ccEndpoint)
	return ccEndpoint, nil
}

//注意-当我们实现join时，我们将不再将chainID作为参数传递。
//链码支持将在不注册系统链码的情况下出现
//只在连接阶段注册。
func registerChaincodeSupport(
	grpcServer *comm.GRPCServer,
	ccEndpoint string,
	ca tlsgen.CA,
	packageProvider *persistence.PackageProvider,
	aclProvider aclmgmt.ACLProvider,
	pr *platforms.Registry,
	lifecycleSCC *lifecycle.SCC,
	ops *operations.System,
) (*chaincode.ChaincodeSupport, ccprovider.ChaincodeProvider, *scc.Provider) {
//获取用户模式
	userRunsCC := chaincode.IsDevMode()
	tlsEnabled := viper.GetBool("peer.tls.enabled")

	authenticator := accesscontrol.NewAuthenticator(ca)
	ipRegistry := inproccontroller.NewRegistry()

	sccp := scc.NewProvider(peer.Default, peer.DefaultSupport, ipRegistry)
	lsccInst := lscc.New(sccp, aclProvider, pr)

	dockerProvider := dockercontroller.NewProvider(
		viper.GetString("peer.id"),
		viper.GetString("peer.networkId"),
		ops.Provider,
	)
	dockerVM := dockercontroller.NewDockerVM(
		dockerProvider.PeerID,
		dockerProvider.NetworkID,
		dockerProvider.BuildMetrics,
	)

	err := ops.RegisterChecker("docker", dockerVM)
	if err != nil {
		logger.Panicf("failed to register docker health check: %s", err)
	}

	chaincodeSupport := chaincode.NewChaincodeSupport(
		chaincode.GlobalConfig(),
		ccEndpoint,
		userRunsCC,
		ca.CertBytes(),
		authenticator,
		packageProvider,
		lsccInst,
		aclProvider,
		container.NewVMController(
			map[string]container.VMProvider{
				dockercontroller.ContainerType: dockerProvider,
				inproccontroller.ContainerType: ipRegistry,
			},
		),
		sccp,
		pr,
		peer.DefaultSupport,
		ops.Provider,
	)
	ipRegistry.ChaincodeSupport = chaincodeSupport
	ccp := chaincode.NewProvider(chaincodeSupport)

	ccSrv := pb.ChaincodeSupportServer(chaincodeSupport)
	if tlsEnabled {
		ccSrv = authenticator.Wrap(ccSrv)
	}

	csccInst := cscc.New(ccp, sccp, aclProvider)
	qsccInst := qscc.New(aclProvider)

//既然链码已经初始化，那么就注册所有系统链码。
	sccs := scc.CreatePluginSysCCs(sccp)
	for _, cc := range append([]scc.SelfDescribingSysCC{lsccInst, csccInst, qsccInst, lifecycleSCC}, sccs...) {
		sccp.RegisterSysCC(cc)
	}
	pb.RegisterChaincodeSupportServer(grpcServer.Server(), ccSrv)

	return chaincodeSupport, ccp, sccp
}

//StartChaincodeServer将完成与Chaincode相关的初始化，包括：
//1）设置本地链码安装路径
//2）创建链码特定的TLS CA
//3）启动链码特定的GRPC监听服务
func startChaincodeServer(
	peerHost string,
	aclProvider aclmgmt.ACLProvider,
	pr *platforms.Registry,
	ops *operations.System,
) (*chaincode.ChaincodeSupport, ccprovider.ChaincodeProvider, *scc.Provider, *persistence.PackageProvider) {
//设置链码路径
	chaincodeInstallPath := ccprovider.GetChaincodeInstallPathFromViper()
	ccprovider.SetChaincodesPath(chaincodeInstallPath)

	ccPackageParser := &persistence.ChaincodePackageParser{}
	ccStore := &persistence.Store{
		Path:       chaincodeInstallPath,
		ReadWriter: &persistence.FilesystemIO{},
	}

	packageProvider := &persistence.PackageProvider{
		LegacyPP: &ccprovider.CCInfoFSImpl{},
		Store:    ccStore,
	}

	lifecycleSCC := &lifecycle.SCC{
		Protobuf: &lifecycle.ProtobufImpl{},
		Functions: &lifecycle.Lifecycle{
			PackageParser:  ccPackageParser,
			ChaincodeStore: ccStore,
		},
	}

//为链码服务创建自签名CA
	ca, err := tlsgen.NewCA()
	if err != nil {
		logger.Panic("Failed creating authentication layer:", err)
	}
	ccSrv, ccEndpoint, err := createChaincodeServer(ca, peerHost)
	if err != nil {
		logger.Panicf("Failed to create chaincode server: %s", err)
	}
	chaincodeSupport, ccp, sccp := registerChaincodeSupport(
		ccSrv,
		ccEndpoint,
		ca,
		packageProvider,
		aclProvider,
		pr,
		lifecycleSCC,
		ops,
	)
	go ccSrv.Start()
	return chaincodeSupport, ccp, sccp, packageProvider
}

func adminHasSeparateListener(peerListenAddr string, adminListenAddress string) bool {
//默认情况下，管理员在与对等数据服务相同的端口上侦听
	if adminListenAddress == "" {
		return false
	}
	_, peerPort, err := net.SplitHostPort(peerListenAddr)
	if err != nil {
		logger.Panicf("Failed parsing peer listen address")
	}

	_, adminPort, err := net.SplitHostPort(adminListenAddress)
	if err != nil {
		logger.Panicf("Failed parsing admin listen address")
	}
//管理服务有一个单独的侦听器，以防它与对等服务器的侦听器不匹配。
//配置的服务
	return adminPort != peerPort
}

func startAdminServer(peerListenAddr string, peerServer *grpc.Server, metricsProvider metrics.Provider) {
	adminListenAddress := viper.GetString("peer.adminService.listenAddress")
	separateLsnrForAdmin := adminHasSeparateListener(peerListenAddr, adminListenAddress)
	mspID := viper.GetString("peer.localMspId")
	adminPolicy := localPolicy(cauthdsl.SignedByAnyAdmin([]string{mspID}))
	gRPCService := peerServer
	if separateLsnrForAdmin {
		logger.Info("Creating gRPC server for admin service on", adminListenAddress)
		serverConfig, err := peer.GetServerConfig()
		if err != nil {
			logger.Fatalf("Error loading secure config for admin service (%s)", err)
		}
		throttle := comm.NewThrottle(grpcMaxConcurrency)
		serverConfig.Logger = flogging.MustGetLogger("core.comm").With("server", "AdminServer")
		serverConfig.MetricsProvider = metricsProvider
		serverConfig.UnaryInterceptors = append(
			serverConfig.UnaryInterceptors,
			grpcmetrics.UnaryServerInterceptor(grpcmetrics.NewUnaryMetrics(metricsProvider)),
			grpclogging.UnaryServerInterceptor(flogging.MustGetLogger("comm.grpc.server").Zap()),
			throttle.UnaryServerIntercptor,
		)
		serverConfig.StreamInterceptors = append(
			serverConfig.StreamInterceptors,
			grpcmetrics.StreamServerInterceptor(grpcmetrics.NewStreamMetrics(metricsProvider)),
			grpclogging.StreamServerInterceptor(flogging.MustGetLogger("comm.grpc.server").Zap()),
			throttle.StreamServerInterceptor,
		)
		adminServer, err := peer.NewPeerServer(adminListenAddress, serverConfig)
		if err != nil {
			logger.Fatalf("Failed to create admin server (%s)", err)
		}
		gRPCService = adminServer.Server()
		defer func() {
			go adminServer.Start()
		}()
	}

	pb.RegisterAdminServer(gRPCService, admin.NewAdminServer(adminPolicy))
}

//SecureDialOpts是用于八卦服务的安全拨号选项的回调函数
func secureDialOpts() []grpc.DialOption {
	var dialOpts []grpc.DialOption
//设置最大发送/接收消息大小
	dialOpts = append(
		dialOpts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(comm.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(comm.MaxSendMsgSize)))
//设置keepalive选项
	kaOpts := comm.DefaultKeepaliveOptions
	if viper.IsSet("peer.keepalive.client.interval") {
		kaOpts.ClientInterval = viper.GetDuration("peer.keepalive.client.interval")
	}
	if viper.IsSet("peer.keepalive.client.timeout") {
		kaOpts.ClientTimeout = viper.GetDuration("peer.keepalive.client.timeout")
	}
	dialOpts = append(dialOpts, comm.ClientKeepaliveOptions(kaOpts)...)

	if viper.GetBool("peer.tls.enabled") {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(comm.GetCredentialSupport().GetPeerCredentials()))
	} else {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}
	return dialOpts
}

//initgossipservice将通过以下方式初始化八卦服务：
//1。启用TLS（如果配置）；
//2。初始化消息加密服务；
//三。启动安全顾问；
//4。初始化八卦相关结构。
func initGossipService(policyMgr policies.ChannelPolicyManagerGetter, peerServer *comm.GRPCServer, serializedIdentity []byte, peerAddr string) error {
	var certs *gossipcommon.TLSCertificates
	if peerServer.TLSEnabled() {
		serverCert := peerServer.ServerCertificate()
		clientCert, err := peer.GetClientCertificate()
		if err != nil {
			return errors.Wrap(err, "failed obtaining client certificates")
		}
		certs = &gossipcommon.TLSCertificates{}
		certs.TLSServerCert.Store(&serverCert)
		certs.TLSClientCert.Store(&clientCert)
	}

	messageCryptoService := peergossip.NewMCS(
		policyMgr,
		localmsp.NewSigner(),
		mgmt.NewDeserializersManager(),
	)
	secAdv := peergossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())
	bootstrap := viper.GetStringSlice("peer.gossip.bootstrap")

	return service.InitGossipService(
		serializedIdentity,
		peerAddr,
		peerServer.Server(),
		certs,
		messageCryptoService,
		secAdv,
		secureDialOpts,
		bootstrap...,
	)
}

func newOperationsSystem() *operations.System {
	return operations.NewSystem(operations.Options{
		Logger:        flogging.MustGetLogger("peer.operations"),
		ListenAddress: viper.GetString("operations.listenAddress"),
		Metrics: operations.MetricsOptions{
			Provider: viper.GetString("metrics.provider"),
			Statsd: &operations.Statsd{
				Network:       viper.GetString("metrics.statsd.network"),
				Address:       viper.GetString("metrics.statsd.address"),
				WriteInterval: viper.GetDuration("metrics.statsd.writeInterval"),
				Prefix:        viper.GetString("metrics.statsd.prefix"),
			},
		},
		TLS: operations.TLS{
			Enabled:            viper.GetBool("operations.tls.enabled"),
			CertFile:           viper.GetString("operations.tls.cert.file"),
			KeyFile:            viper.GetString("operations.tls.key.file"),
			ClientCertRequired: viper.GetBool("operations.tls.clientAuthRequired"),
			ClientCACertFiles:  viper.GetStringSlice("operations.tls.clientRootCAs.files"),
		},
		Version: metadata.Version,
	})
}

func registerProverService(peerServer *comm.GRPCServer, aclProvider aclmgmt.ACLProvider, signingIdentity msp.SigningIdentity) error {
	policyChecker := &server.PolicyBasedAccessControl{
		ACLProvider: aclProvider,
		ACLResources: &server.ACLResources{
			IssueTokens:    resources.Token_Issue,
			TransferTokens: resources.Token_Transfer,
			ListTokens:     resources.Token_List,
		},
	}

	responseMarshaler, err := server.NewResponseMarshaler(signingIdentity)
	if err != nil {
		logger.Errorf("Failed to create prover service: %s", err)
		return err
	}

	prover := &server.Prover{
		CapabilityChecker: &server.TokenCapabilityChecker{
			PeerOps: peer.Default,
		},
		Marshaler:     responseMarshaler,
		PolicyChecker: policyChecker,
		TMSManager: &server.Manager{
			LedgerManager: &server.PeerLedgerManager{},
		},
	}
	token.RegisterProverServer(peerServer.Server(), prover)
	return nil
}
