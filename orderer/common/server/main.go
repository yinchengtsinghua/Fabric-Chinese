
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


package server

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
_ "net/http/pprof" //这基本上是订购方的主包
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	floggingmetrics "github.com/hyperledger/fabric/common/flogging/metrics"
	"github.com/hyperledger/fabric/common/grpclogging"
	"github.com/hyperledger/fabric/common/grpcmetrics"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/operations"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/bootstrap/file"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/common/metadata"
	"github.com/hyperledger/fabric/orderer/common/multichannel"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/orderer/consensus/etcdraft"
	"github.com/hyperledger/fabric/orderer/consensus/kafka"
	"github.com/hyperledger/fabric/orderer/consensus/solo"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
)

var logger = flogging.MustGetLogger("orderer.common.server")

//命令行标志
var (
	app = kingpin.New("orderer", "Hyperledger Fabric orderer node")

	start     = app.Command("start", "Start the orderer node").Default()
	version   = app.Command("version", "Show version information")
	benchmark = app.Command("benchmark", "Run orderer in benchmark mode")

	clusterTypes = map[string]struct{}{"etcdraft": {}}
)

//主要是订购流程的入口点
func Main() {
	fullCmd := kingpin.MustParse(app.Parse(os.Args[1:]))

//“版本”命令
	if fullCmd == version.FullCommand() {
		fmt.Println(metadata.GetVersionInfo())
		return
	}

	conf, err := localconfig.Load()
	if err != nil {
		logger.Error("failed to parse config: ", err)
		os.Exit(1)
	}
	initializeLogging()
	initializeLocalMsp(conf)

	prettyPrintStruct(conf)
	Start(fullCmd, conf)
}

//Start为基准测试提供了一个抽象层
func Start(cmd string, conf *localconfig.TopLevel) {
	bootstrapBlock := extractBootstrapBlock(conf)
	clusterType := isClusterType(bootstrapBlock)
	signer := localmsp.NewSigner()

	lf, _ := createLedgerFactory(conf)

	clusterDialer := &cluster.PredicateDialer{}
	clusterConfig := initializeClusterConfig(conf)
	clusterDialer.SetConfig(clusterConfig)

//只有配备了最新配置块的集群才能复制。
	if clusterType && conf.General.GenesisMethod == "file" {
		r := &replicationInitiator{
			logger:         logger,
			secOpts:        clusterConfig.SecOpts,
			bootstrapBlock: bootstrapBlock,
			conf:           conf,
			lf:             &ledgerFactory{lf},
			signer:         signer,
		}
		r.replicateIfNeeded()
	}

	opsSystem := newOperationsSystem(conf.Operations, conf.Metrics)
	err := opsSystem.Start()
	if err != nil {
		logger.Panicf("failed to initialize operations subsystem: %s", err)
	}
	defer opsSystem.Stop()
	metricsProvider := opsSystem.Provider
	logObserver := floggingmetrics.NewObserver(metricsProvider)
	flogging.Global.SetObserver(logObserver)

	serverConfig := initializeServerConfig(conf, metricsProvider)
	grpcServer := initializeGrpcServer(conf, serverConfig)
	caSupport := &comm.CASupport{
		AppRootCAsByChain:     make(map[string][][]byte),
		OrdererRootCAsByChain: make(map[string][][]byte),
		ClientRootCAs:         serverConfig.SecOpts.ClientRootCAs,
	}

	tlsCallback := func(bundle *channelconfig.Bundle) {
//仅当需要相互TLS或者订购方节点是集群的一部分时才需要这样做。
		if grpcServer.MutualTLSRequired() || clusterType {
			logger.Debug("Executing callback to update root CAs")
			updateTrustedRoots(grpcServer, caSupport, bundle)
			if clusterType {
				updateClusterDialer(caSupport, clusterDialer, clusterConfig.SecOpts.ServerRootCAs)
			}
		}
	}

	manager := initializeMultichannelRegistrar(bootstrapBlock, clusterDialer, serverConfig, grpcServer, conf, signer, metricsProvider, lf, tlsCallback)
	mutualTLS := serverConfig.SecOpts.UseTLS && serverConfig.SecOpts.RequireClientCert
	server := NewServer(manager, metricsProvider, &conf.Debug, conf.General.Authentication.TimeWindow, mutualTLS)

	logger.Infof("Starting %s", metadata.GetVersionInfo())
	go handleSignals(addPlatformSignals(map[os.Signal]func(){
		syscall.SIGTERM: func() { grpcServer.Stop() },
	}))
	initializeProfilingService(conf)
	ab.RegisterAtomicBroadcastServer(grpcServer.Server(), server)
	logger.Info("Beginning to serve requests")
	grpcServer.Start()
}

func initializeLogging() {
	loggingSpec := os.Getenv("FABRIC_LOGGING_SPEC")
	loggingFormat := os.Getenv("FABRIC_LOGGING_FORMAT")
	flogging.Init(flogging.Config{
		Format:  loggingFormat,
		Writer:  os.Stderr,
		LogSpec: loggingSpec,
	})
}

//如果启用，则启动分析服务。
func initializeProfilingService(conf *localconfig.TopLevel) {
	if conf.General.Profile.Enabled {
		go func() {
			logger.Info("Starting Go pprof profiling service on:", conf.General.Profile.Address)
//除非发生错误，否则listenandserve（）调用不会返回。
			logger.Panic("Go pprof service failed:", http.ListenAndServe(conf.General.Profile.Address, nil))
		}()
	}
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

func initializeClusterConfig(conf *localconfig.TopLevel) comm.ClientConfig {
	cc := comm.ClientConfig{
		AsyncConnect: true,
		KaOpts:       comm.DefaultKeepaliveOptions,
		Timeout:      conf.General.Cluster.DialTimeout,
		SecOpts:      &comm.SecureOptions{},
	}

	if (!conf.General.TLS.Enabled) || conf.General.Cluster.ClientCertificate == "" {
		return cc
	}

	certFile := conf.General.Cluster.ClientCertificate
	certBytes, err := ioutil.ReadFile(certFile)
	if err != nil {
		logger.Fatalf("Failed to load client TLS certificate file '%s' (%s)", certFile, err)
	}

	keyFile := conf.General.Cluster.ClientPrivateKey
	keyBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		logger.Fatalf("Failed to load client TLS key file '%s' (%s)", keyFile, err)
	}

	var serverRootCAs [][]byte
	for _, serverRoot := range conf.General.Cluster.RootCAs {
		rootCACert, err := ioutil.ReadFile(serverRoot)
		if err != nil {
			logger.Fatalf("Failed to load ServerRootCAs file '%s' (%s)",
				err, serverRoot)
		}
		serverRootCAs = append(serverRootCAs, rootCACert)
	}

	cc.SecOpts = &comm.SecureOptions{
		RequireClientCert: true,
		CipherSuites:      comm.DefaultTLSCipherSuites,
		ServerRootCAs:     serverRootCAs,
		Certificate:       certBytes,
		Key:               keyBytes,
		UseTLS:            true,
	}

	return cc
}

func initializeServerConfig(conf *localconfig.TopLevel, metricsProvider metrics.Provider) comm.ServerConfig {
//安全服务器配置
	secureOpts := &comm.SecureOptions{
		UseTLS:            conf.General.TLS.Enabled,
		RequireClientCert: conf.General.TLS.ClientAuthRequired,
	}
//检查是否启用了TLS
	if secureOpts.UseTLS {
		msg := "TLS"
//从文件加载加密材料
		serverCertificate, err := ioutil.ReadFile(conf.General.TLS.Certificate)
		if err != nil {
			logger.Fatalf("Failed to load server Certificate file '%s' (%s)",
				conf.General.TLS.Certificate, err)
		}
		serverKey, err := ioutil.ReadFile(conf.General.TLS.PrivateKey)
		if err != nil {
			logger.Fatalf("Failed to load PrivateKey file '%s' (%s)",
				conf.General.TLS.PrivateKey, err)
		}
		var serverRootCAs, clientRootCAs [][]byte
		for _, serverRoot := range conf.General.TLS.RootCAs {
			root, err := ioutil.ReadFile(serverRoot)
			if err != nil {
				logger.Fatalf("Failed to load ServerRootCAs file '%s' (%s)",
					err, serverRoot)
			}
			serverRootCAs = append(serverRootCAs, root)
		}
		if secureOpts.RequireClientCert {
			for _, clientRoot := range conf.General.TLS.ClientRootCAs {
				root, err := ioutil.ReadFile(clientRoot)
				if err != nil {
					logger.Fatalf("Failed to load ClientRootCAs file '%s' (%s)",
						err, clientRoot)
				}
				clientRootCAs = append(clientRootCAs, root)
			}
			msg = "mutual TLS"
		}
		secureOpts.Key = serverKey
		secureOpts.Certificate = serverCertificate
		secureOpts.ClientRootCAs = clientRootCAs
		logger.Infof("Starting orderer with %s enabled", msg)
	}
	kaOpts := comm.DefaultKeepaliveOptions
//保持连接设置
//ServerMinInterval必须大于0
	if conf.General.Keepalive.ServerMinInterval > time.Duration(0) {
		kaOpts.ServerMinInterval = conf.General.Keepalive.ServerMinInterval
	}
	kaOpts.ServerInterval = conf.General.Keepalive.ServerInterval
	kaOpts.ServerTimeout = conf.General.Keepalive.ServerTimeout

	commLogger := flogging.MustGetLogger("core.comm").With("server", "Orderer")
	if metricsProvider == nil {
		metricsProvider = &disabled.Provider{}
	}

	return comm.ServerConfig{
		SecOpts:         secureOpts,
		KaOpts:          kaOpts,
		Logger:          commLogger,
		MetricsProvider: metricsProvider,
		StreamInterceptors: []grpc.StreamServerInterceptor{
			grpcmetrics.StreamServerInterceptor(grpcmetrics.NewStreamMetrics(metricsProvider)),
			grpclogging.StreamServerInterceptor(flogging.MustGetLogger("comm.grpc.server").Zap()),
		},
		UnaryInterceptors: []grpc.UnaryServerInterceptor{
			grpcmetrics.UnaryServerInterceptor(grpcmetrics.NewUnaryMetrics(metricsProvider)),
			grpclogging.UnaryServerInterceptor(
				flogging.MustGetLogger("comm.grpc.server").Zap(),
				grpclogging.WithLeveler(grpclogging.LevelerFunc(grpcLeveler)),
			),
		},
	}
}

func grpcLeveler(ctx context.Context, fullMethod string) zapcore.Level {
	switch fullMethod {
	case "/orderer.Cluster/Step":
		return flogging.DisabledLevel
	default:
		return zapcore.InfoLevel
	}
}

func extractBootstrapBlock(conf *localconfig.TopLevel) *cb.Block {
	var bootstrapBlock *cb.Block

//选择引导机制
	switch conf.General.GenesisMethod {
	case "provisional":
		bootstrapBlock = encoder.New(genesisconfig.Load(conf.General.GenesisProfile)).GenesisBlockForChannel(conf.General.SystemChannel)
	case "file":
		bootstrapBlock = file.New(conf.General.GenesisFile).GenesisBlock()
	default:
		logger.Panic("Unknown genesis method:", conf.General.GenesisMethod)
	}

	return bootstrapBlock
}

func initializeBootstrapChannel(genesisBlock *cb.Block, lf blockledger.Factory) {
	chainID, err := utils.GetChainIDFromBlock(genesisBlock)
	if err != nil {
		logger.Fatal("Failed to parse chain ID from genesis block:", err)
	}
	gl, err := lf.GetOrCreate(chainID)
	if err != nil {
		logger.Fatal("Failed to create the system chain:", err)
	}

	if err := gl.Append(genesisBlock); err != nil {
		logger.Fatal("Could not write genesis block to ledger:", err)
	}
}

func isClusterType(_ *cb.Block) bool {
	return false
}

func initializeGrpcServer(conf *localconfig.TopLevel, serverConfig comm.ServerConfig) *comm.GRPCServer {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort))
	if err != nil {
		logger.Fatal("Failed to listen:", err)
	}

//创建GRPC服务器-发生错误时返回
	grpcServer, err := comm.NewGRPCServerFromListener(lis, serverConfig)
	if err != nil {
		logger.Fatal("Failed to return new GRPC server:", err)
	}

	return grpcServer
}

func initializeLocalMsp(conf *localconfig.TopLevel) {
//加载本地MSP
	err := mspmgmt.LoadLocalMsp(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID)
if err != nil { //处理读取配置文件时的错误
		logger.Fatal("Failed to initialize local MSP:", err)
	}
}

func initializeMultichannelRegistrar(bootstrapBlock *cb.Block,
	clusterDialer *cluster.PredicateDialer,
	srvConf comm.ServerConfig,
	srv *comm.GRPCServer,
	conf *localconfig.TopLevel,
	signer crypto.LocalSigner,
	metricsProvider metrics.Provider,
	lf blockledger.Factory,
	callbacks ...func(bundle *channelconfig.Bundle)) *multichannel.Registrar {
	genesisBlock := extractBootstrapBlock(conf)
//我们是在引导吗？
	if len(lf.ChainIDs()) == 0 {
		initializeBootstrapChannel(genesisBlock, lf)
	} else {
		logger.Info("Not bootstrapping because of existing chains")
	}

	consenters := make(map[string]consensus.Consenter)

	registrar := multichannel.NewRegistrar(lf, signer, metricsProvider, callbacks...)

	consenters["solo"] = solo.New()
	var kafkaMetrics *kafka.Metrics
	consenters["kafka"], kafkaMetrics = kafka.New(conf.Kafka, metricsProvider)
//注意，我们在这里通过一个“零”频道，我们可以通过一个
//如果希望在退出时清除此例程，则关闭。
	go kafkaMetrics.PollGoMetricsUntilStop(time.Minute, nil)
	if isClusterType(bootstrapBlock) {
		raftConsenter := etcdraft.New(clusterDialer, conf, srvConf, srv, registrar)
		consenters["etcdraft"] = raftConsenter
	}
	registrar.Initialize(consenters)
	return registrar
}

func newOperationsSystem(ops localconfig.Operations, metrics localconfig.Metrics) *operations.System {
	return operations.NewSystem(operations.Options{
		Logger:        flogging.MustGetLogger("orderer.operations"),
		ListenAddress: ops.ListenAddress,
		Metrics: operations.MetricsOptions{
			Provider: metrics.Provider,
			Statsd: &operations.Statsd{
				Network:       metrics.Statsd.Network,
				Address:       metrics.Statsd.Address,
				WriteInterval: metrics.Statsd.WriteInterval,
				Prefix:        metrics.Statsd.Prefix,
			},
		},
		TLS: operations.TLS{
			Enabled:            ops.TLS.Enabled,
			CertFile:           ops.TLS.Certificate,
			KeyFile:            ops.TLS.PrivateKey,
			ClientCertRequired: ops.TLS.ClientAuthRequired,
			ClientCACertFiles:  ops.TLS.ClientRootCAs,
		},
		Version: metadata.Version,
	})
}

func updateTrustedRoots(srv *comm.GRPCServer, rootCASupport *comm.CASupport,
	cm channelconfig.Resources) {
	rootCASupport.Lock()
	defer rootCASupport.Unlock()

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

	if cc, ok := cm.ConsortiumsConfig(); ok {
		for _, consortium := range cc.Consortiums() {
//通过联合体组织循环并构建MSPIDS地图
			for _, consortiumOrg := range consortium.Organizations() {
				appOrgMSPs[consortiumOrg.MSPID()] = struct{}{}
			}
		}
	}

	cid := cm.ConfigtxValidator().ChainID()
	logger.Debugf("updating root CAs for channel [%s]", cid)
	msps, err := cm.MSPManager().GetMSPs()
	if err != nil {
		logger.Errorf("Error getting root CAs for channel %s (%s)", cid, err)
		return
	}
	for k, v := range msps {
//检查这是否是织物MSP
		if v.GetType() == msp.FABRIC {
			for _, root := range v.GetTLSRootCerts() {
//查看这是一个应用程序组织MSP
				if _, ok := appOrgMSPs[k]; ok {
					logger.Debugf("adding app root CAs for MSP [%s]", k)
					appRootCAs = append(appRootCAs, root)
				}
//查看这是一个订购者组织MSP
				if _, ok := ordOrgMSPs[k]; ok {
					logger.Debugf("adding orderer root CAs for MSP [%s]", k)
					ordererRootCAs = append(ordererRootCAs, root)
				}
			}
			for _, intermediate := range v.GetTLSIntermediateCerts() {
//查看这是一个应用程序组织MSP
				if _, ok := appOrgMSPs[k]; ok {
					logger.Debugf("adding app root CAs for MSP [%s]", k)
					appRootCAs = append(appRootCAs, intermediate)
				}
//查看这是一个订购者组织MSP
				if _, ok := ordOrgMSPs[k]; ok {
					logger.Debugf("adding orderer root CAs for MSP [%s]", k)
					ordererRootCAs = append(ordererRootCAs, intermediate)
				}
			}
		}
	}
	rootCASupport.AppRootCAsByChain[cid] = appRootCAs
	rootCASupport.OrdererRootCAsByChain[cid] = ordererRootCAs

//现在遍历所有应用程序和订购程序链的所有根
	trustedRoots := [][]byte{}
	for _, roots := range rootCASupport.AppRootCAsByChain {
		trustedRoots = append(trustedRoots, roots...)
	}
	for _, roots := range rootCASupport.OrdererRootCAsByChain {
		trustedRoots = append(trustedRoots, roots...)
	}
//还需要附加静态配置的根证书
	if len(rootCASupport.ClientRootCAs) > 0 {
		trustedRoots = append(trustedRoots, rootCASupport.ClientRootCAs...)
	}

//现在更新GRPC服务器的客户机根目录
	err = srv.SetClientRootCAs(trustedRoots)
	if err != nil {
		msg := "Failed to update trusted roots for orderer from latest config " +
			"block.  This orderer may not be able to communicate " +
			"with members of channel %s (%s)"
		logger.Warningf(msg, cm.ConfigtxValidator().ChainID(), err)
	}
}

func updateClusterDialer(rootCASupport *comm.CASupport, clusterDialer *cluster.PredicateDialer, localClusterRootCAs [][]byte) {
	rootCASupport.Lock()
	defer rootCASupport.Unlock()

//遍历所有链的所有订购者根CA并添加它们
//到根CAS
	var clusterRootCAs [][]byte
	for _, roots := range rootCASupport.OrdererRootCAsByChain {
		clusterRootCAs = append(clusterRootCAs, roots...)
	}

//也添加本地根CA
	clusterRootCAs = append(clusterRootCAs, localClusterRootCAs...)
//用新的根CA更新集群配置
	clusterConfig := clusterDialer.Config.Load().(comm.ClientConfig)
	clusterConfig.SecOpts.ServerRootCAs = clusterRootCAs
	clusterDialer.SetConfig(clusterConfig)
}

func prettyPrintStruct(i interface{}) {
	params := util.Flatten(i)
	var buffer bytes.Buffer
	for i := range params {
		buffer.WriteString("\n\t")
		buffer.WriteString(params[i])
	}
	logger.Infof("Orderer config values:%s\n", buffer.String())
}
