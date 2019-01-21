
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


package chaincode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	mc "github.com/hyperledger/fabric/common/mocks/config"
	mockpolicies "github.com/hyperledger/fabric/common/mocks/policies"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/aclmgmt"
	aclmocks "github.com/hyperledger/fabric/core/aclmgmt/mocks"
	"github.com/hyperledger/fabric/core/chaincode/accesscontrol"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/dockercontroller"
	"github.com/hyperledger/fabric/core/container/inproccontroller"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	cut "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	cmp "github.com/hyperledger/fabric/core/mocks/peer"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/core/policy/mocks"
	"github.com/hyperledger/fabric/core/scc"
	"github.com/hyperledger/fabric/core/scc/lscc"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

//初始化对等机并启动。如果security==enabled，则以vp身份登录
func initPeer(chainIDs ...string) (net.Listener, *ChaincodeSupport, func(), error) {
//开始清洁
	finitPeer(nil, chainIDs...)

	msi := &cmp.MockSupportImpl{
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetApplicationConfigBoolRv: true,
	}

	ipRegistry := inproccontroller.NewRegistry()
	sccp := &scc.Provider{Peer: peer.Default, PeerSupport: msi, Registrar: ipRegistry}

	mockAclProvider = &aclmocks.MockACLProvider{}
	mockAclProvider.Reset()

	peer.MockInitialize()

	mspGetter := func(cid string) []string {
		return []string{"SampleOrg"}
	}

	peer.MockSetMSPIDGetter(mspGetter)

//对于单元测试，不需要TLS。
	viper.Set("peer.tls.enabled", false)

	var opts []grpc.ServerOption
	if viper.GetBool("peer.tls.enabled") {
		creds, err := credentials.NewServerTLSFromFile(config.GetPath("peer.tls.cert.file"), config.GetPath("peer.tls.key.file"))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)

	peerAddress, err := peer.GetLocalAddress()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error obtaining peer address: %s", err)
	}
	lis, err := net.Listen("tcp", peerAddress)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error starting peer listener %s", err)
	}

	ccprovider.SetChaincodesPath(ccprovider.GetChaincodeInstallPathFromViper())
	ca, _ := tlsgen.NewCA()
	certGenerator := accesscontrol.NewAuthenticator(ca)
	config := GlobalConfig()
	config.StartupTimeout = 3 * time.Minute
	pr := platforms.NewRegistry(&golang.Platform{})
	lsccImpl := lscc.New(sccp, mockAclProvider, pr)
	chaincodeSupport := NewChaincodeSupport(
		config,
		peerAddress,
		false,
		ca.CertBytes(),
		certGenerator,
		&ccprovider.CCInfoFSImpl{},
		lsccImpl,
		aclmgmt.NewACLProvider(func(string) channelconfig.Resources { return nil }),
		container.NewVMController(
			map[string]container.VMProvider{
				dockercontroller.ContainerType: dockercontroller.NewProvider("", "", &disabled.Provider{}),
				inproccontroller.ContainerType: ipRegistry,
			},
		),
		sccp,
		pr,
		peer.DefaultSupport,
		&disabled.Provider{},
	)
	ipRegistry.ChaincodeSupport = chaincodeSupport
	pb.RegisterChaincodeSupportServer(grpcServer, chaincodeSupport)

//模拟策略检查器
	policy.RegisterPolicyCheckerFactory(&mockPolicyCheckerFactory{})

	ccp := &CCProviderImpl{cs: chaincodeSupport}
	sccp.RegisterSysCC(lsccImpl)

	for _, id := range chainIDs {
		sccp.DeDeploySysCCs(id, ccp)
		if err = peer.MockCreateChain(id); err != nil {
			closeListenerAndSleep(lis)
			return nil, nil, nil, err
		}
		sccp.DeploySysCCs(id, ccp)
//除默认testchainid之外的任何链都没有设置msp->create one
		if id != util.GetTestChainID() {
			mspmgmt.XXXSetMSPManager(id, mspmgmt.GetManagerForChain(util.GetTestChainID()))
		}
	}

	go grpcServer.Serve(lis)

//在顶上超过零尼什
	return lis, chaincodeSupport, func() { finitPeer(lis, chainIDs...) }, nil
}

func finitPeer(lis net.Listener, chainIDs ...string) {
	if lis != nil {
		for _, c := range chainIDs {
			if lgr := peer.GetLedger(c); lgr != nil {
				lgr.Close()
			}
		}
		closeListenerAndSleep(lis)
	}
	ledgermgmt.CleanupTestEnv()
	ledgerPath := config.GetPath("peer.fileSystemPath")
	os.RemoveAll(ledgerPath)
	os.RemoveAll(filepath.Join(os.TempDir(), "hyperledger"))

//如果启用了couchdb，则清除测试couchdb
	if ledgerconfig.IsCouchDBEnabled() == true {

		chainID := util.GetTestChainID()

		connectURL := viper.GetString("ledger.state.couchDBConfig.couchDBAddress")
		username := viper.GetString("ledger.state.couchDBConfig.username")
		password := viper.GetString("ledger.state.couchDBConfig.password")
		maxRetries := viper.GetInt("ledger.state.couchDBConfig.maxRetries")
		maxRetriesOnStartup := viper.GetInt("ledger.state.couchDBConfig.maxRetriesOnStartup")
		requestTimeout := viper.GetDuration("ledger.state.couchDBConfig.requestTimeout")
		createGlobalChangesDB := viper.GetBool("ledger.state.couchDBConfig.createGlobalChangesDB")

		couchInstance, _ := couchdb.CreateCouchInstance(connectURL, username, password, maxRetries, maxRetriesOnStartup, requestTimeout, createGlobalChangesDB, &disabled.Provider{})
		db := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: chainID}
//删除测试数据库
		db.DropDatabase()

	}
}

func startTxSimulation(chainID string, txid string) (ledger.TxSimulator, ledger.HistoryQueryExecutor, error) {
	lgr := peer.GetLedger(chainID)
	txsim, err := lgr.NewTxSimulator(txid)
	if err != nil {
		return nil, nil, err
	}
	historyQueryExecutor, err := lgr.NewHistoryQueryExecutor()
	if err != nil {
		return nil, nil, err
	}

	return txsim, historyQueryExecutor, nil
}

func endTxSimulationCDS(chainID string, txid string, txsim ledger.TxSimulator, payload []byte, commit bool, cds *pb.ChaincodeDeploymentSpec, blockNumber uint64) error {
//获取签名者的序列化版本
	ss, err := signer.Serialize()
	if err != nil {
		return err
	}

//获取lscc chaincodeid
	lsccid := &pb.ChaincodeID{
		Name:    "lscc",
		Version: util.GetSysCCVersion(),
	}

//得到一个提议-我们需要它来达成交易
	prop, _, err := putils.CreateDeployProposalFromCDS(chainID, cds, ss, nil, nil, nil, nil)
	if err != nil {
		return err
	}

	return endTxSimulation(chainID, lsccid, txsim, payload, commit, prop, blockNumber)
}

func endTxSimulationCIS(chainID string, ccid *pb.ChaincodeID, txid string, txsim ledger.TxSimulator, payload []byte, commit bool, cis *pb.ChaincodeInvocationSpec, blockNumber uint64) error {
//获取签名者的序列化版本
	ss, err := signer.Serialize()
	if err != nil {
		return err
	}

//得到一个提议-我们需要它来达成交易
	prop, returnedTxid, err := putils.CreateProposalFromCISAndTxid(txid, common.HeaderType_ENDORSER_TRANSACTION, chainID, cis, ss)
	if err != nil {
		return err
	}
	if returnedTxid != txid {
		return errors.New("txids are not same")
	}

	return endTxSimulation(chainID, ccid, txsim, payload, commit, prop, blockNumber)
}

//从分类帐中获取崩溃。在执行并发调用时提交
//很可能是故意的，Ledger.commit是串行的（即
//提交者将在每个块上连续地调用这个函数）。在这里模仿
//通过强制对分类帐进行序列化。提交调用。
//
//注：这对旧的串行测试没有任何影响。
//这只影响并发\u test.go中调用这些的测试
//并发（100个并发调用，100个并发查询）
var _commitLock_ sync.Mutex

func endTxSimulation(chainID string, ccid *pb.ChaincodeID, txsim ledger.TxSimulator, _ []byte, commit bool, prop *pb.Proposal, blockNumber uint64) error {
	txsim.Done()
	if lgr := peer.GetLedger(chainID); lgr != nil {
		if commit {
			var txSimulationResults *ledger.TxSimulationResults
			var txSimulationBytes []byte
			var err error

			txsim.Done()

//获取模拟结果
			if txSimulationResults, err = txsim.GetTxSimulationResults(); err != nil {
				return err
			}
			if txSimulationBytes, err = txSimulationResults.GetPubSimulationBytes(); err != nil {
				return nil
			}
//汇编（签名）提案响应消息
			resp, err := putils.CreateProposalResponse(prop.Header, prop.Payload, &pb.Response{Status: 200},
				txSimulationBytes, nil, ccid, nil, signer)
			if err != nil {
				return err
			}

//拿到信封
			env, err := putils.CreateSignedTx(prop, signer, resp)
			if err != nil {
				return err
			}

			envBytes, err := putils.GetBytesEnvelope(env)
			if err != nil {
				return err
			}

//用1个事务创建块
			bcInfo, err := lgr.GetBlockchainInfo()
			if err != nil {
				return err
			}
			block := common.NewBlock(blockNumber, bcInfo.CurrentBlockHash)
			block.Data.Data = [][]byte{envBytes}
			txsFilter := cut.NewTxValidationFlagsSetValue(len(block.Data.Data), pb.TxValidationCode_VALID)
			block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter

//提交块

//请参阅对“CommitLock”的评论
			_commitLock_.Lock()
			defer _commitLock_.Unlock()

			blockAndPvtData := &ledger.BlockAndPvtData{
				Block:   block,
				PvtData: make(ledger.TxPvtDataMap),
			}

//所有测试在一个块中只使用一个事务来执行。
//因此，我们可以模拟
//用私有数据阻止。没有足够的必要
//在块中添加多个事务以测试链代码
//应用程序编程接口。

//假设：一个块中只有一个事务。
			seqInBlock := uint64(0)

			if txSimulationResults.PvtSimulationResults != nil {

				blockAndPvtData.PvtData[seqInBlock] = &ledger.TxPvtData{
					SeqInBlock: seqInBlock,
					WriteSet:   txSimulationResults.PvtSimulationResults,
				}
			}

			if err := lgr.CommitWithPvtData(blockAndPvtData); err != nil {
				return err
			}
		}
	}

	return nil
}

//构建一个链代码。
func getDeploymentSpec(spec *pb.ChaincodeSpec) (*pb.ChaincodeDeploymentSpec, error) {
	fmt.Printf("getting deployment spec for chaincode spec: %v\n", spec)
	codePackageBytes, err := container.GetChaincodePackageBytes(platforms.NewRegistry(&golang.Platform{}), spec)
	if err != nil {
		return nil, err
	}
	cdDeploymentSpec := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: codePackageBytes}
	return cdDeploymentSpec, nil
}

//getdeploylsccspec获取要发送到lscc的链代码部署的规范
func getDeployLSCCSpec(chainID string, cds *pb.ChaincodeDeploymentSpec, ccp *common.CollectionConfigPackage) (*pb.ChaincodeInvocationSpec, error) {
	b, err := proto.Marshal(cds)
	if err != nil {
		return nil, err
	}

	var ccpBytes []byte
	if ccp != nil {
		if ccpBytes, err = proto.Marshal(ccp); err != nil {
			return nil, err
		}
	}
	sysCCVers := util.GetSysCCVersion()

	invokeInput := &pb.ChaincodeInput{Args: [][]byte{
[]byte("deploy"), //函数名
[]byte(chainID),  //要部署的链代码名称
b,                //链码部署规范
	}}

	if ccpBytes != nil {
//SignaturePolicyInvelope、ESCC、VSCC、CollectionConfigPackage
		invokeInput.Args = append(invokeInput.Args, nil, nil, nil, ccpBytes)
	}

//将部署包装到lscc的调用规范中…
	lsccSpec := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        pb.ChaincodeSpec_GOLANG,
			ChaincodeId: &pb.ChaincodeID{Name: "lscc", Version: sysCCVers},
			Input:       invokeInput,
		}}

	return lsccSpec, nil
}

//部署一个链代码——即构建和初始化。
func deploy(chainID string, cccid *ccprovider.CCContext, spec *pb.ChaincodeSpec, blockNumber uint64, chaincodeSupport *ChaincodeSupport) (resp *pb.Response, err error) {
//首先构建并获取部署规范
	cdDeploymentSpec, err := getDeploymentSpec(spec)
	if err != nil {
		return nil, err
	}
	return deploy2(chainID, cccid, cdDeploymentSpec, nil, blockNumber, chaincodeSupport)
}

func deployWithCollectionConfigs(chainID string, cccid *ccprovider.CCContext, spec *pb.ChaincodeSpec,
	collectionConfigPkg *common.CollectionConfigPackage, blockNumber uint64, chaincodeSupport *ChaincodeSupport) (resp *pb.Response, err error) {
//首先构建并获取部署规范
	cdDeploymentSpec, err := getDeploymentSpec(spec)
	if err != nil {
		return nil, err
	}
	return deploy2(chainID, cccid, cdDeploymentSpec, collectionConfigPkg, blockNumber, chaincodeSupport)
}

func deploy2(chainID string, cccid *ccprovider.CCContext, chaincodeDeploymentSpec *pb.ChaincodeDeploymentSpec,
	collectionConfigPkg *common.CollectionConfigPackage, blockNumber uint64, chaincodeSupport *ChaincodeSupport) (resp *pb.Response, err error) {
	cis, err := getDeployLSCCSpec(chainID, chaincodeDeploymentSpec, collectionConfigPkg)
	if err != nil {
		return nil, fmt.Errorf("Error creating lscc spec : %s\n", err)
	}

	uuid := util.GenerateUUID()
	txsim, hqe, err := startTxSimulation(chainID, uuid)
	sprop, prop := putils.MockSignedEndorserProposal2OrPanic(chainID, cis.ChaincodeSpec, signer)
	txParams := &ccprovider.TransactionParams{
		TxID:                 uuid,
		ChannelID:            chainID,
		TXSimulator:          txsim,
		HistoryQueryExecutor: hqe,
		SignedProp:           sprop,
		Proposal:             prop,
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to get handle to simulator: %s ", err)
	}

	defer func() {
//没有错误，让我们尝试提交
		if err == nil {
//捕获从提交返回错误
			err = endTxSimulationCDS(chainID, uuid, txsim, []byte("deployed"), true, chaincodeDeploymentSpec, blockNumber)
		} else {
//
			endTxSimulationCDS(chainID, uuid, txsim, []byte("deployed"), false, chaincodeDeploymentSpec, blockNumber)
		}
	}()

//忽略存在错误
	ccprovider.PutChaincodeIntoFS(chaincodeDeploymentSpec)

	sysCCVers := util.GetSysCCVersion()
	lsccid := &ccprovider.CCContext{
		Name:    cis.ChaincodeSpec.ChaincodeId.Name,
		Version: sysCCVers,
	}

//写入LSCC
	if _, _, err = chaincodeSupport.Execute(txParams, lsccid, cis.ChaincodeSpec.Input); err != nil {
		return nil, fmt.Errorf("Error deploying chaincode (1): %s", err)
	}

	if resp, _, err = chaincodeSupport.ExecuteLegacyInit(txParams, cccid, chaincodeDeploymentSpec); err != nil {
		return nil, fmt.Errorf("Error deploying chaincode(2): %s", err)
	}

	return resp, nil
}

//
func invoke(chainID string, spec *pb.ChaincodeSpec, blockNumber uint64, creator []byte, chaincodeSupport *ChaincodeSupport) (ccevt *pb.ChaincodeEvent, uuid string, retval []byte, err error) {
	return invokeWithVersion(chainID, spec.GetChaincodeId().Version, spec, blockNumber, creator, chaincodeSupport)
}

//使用版本调用链代码（升级时需要）
func invokeWithVersion(chainID string, version string, spec *pb.ChaincodeSpec, blockNumber uint64, creator []byte, chaincodeSupport *ChaincodeSupport) (ccevt *pb.ChaincodeEvent, uuid string, retval []byte, err error) {
	cdInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

//现在创建事务消息并发送到对等机。
	uuid = util.GenerateUUID()

	txsim, hqe, err := startTxSimulation(chainID, uuid)
	if err != nil {
		return nil, uuid, nil, fmt.Errorf("Failed to get handle to simulator: %s ", err)
	}

	defer func() {
//没有错误，让我们尝试提交
		if err == nil {
//捕获从提交返回错误
			err = endTxSimulationCIS(chainID, spec.ChaincodeId, uuid, txsim, []byte("invoke"), true, cdInvocationSpec, blockNumber)
		} else {
//有一个错误，只需关闭模拟并返回
			endTxSimulationCIS(chainID, spec.ChaincodeId, uuid, txsim, []byte("invoke"), false, cdInvocationSpec, blockNumber)
		}
	}()

	if len(creator) == 0 {
		creator = []byte("Admin")
	}
	sprop, prop := putils.MockSignedEndorserProposalOrPanic(chainID, spec, creator, []byte("msg1"))
	cccid := &ccprovider.CCContext{
		Name:    cdInvocationSpec.ChaincodeSpec.ChaincodeId.Name,
		Version: version,
	}
	var resp *pb.Response
	txParams := &ccprovider.TransactionParams{
		TxID:                 uuid,
		ChannelID:            chainID,
		TXSimulator:          txsim,
		HistoryQueryExecutor: hqe,
		SignedProp:           sprop,
		Proposal:             prop,
	}

	resp, ccevt, err = chaincodeSupport.Execute(txParams, cccid, cdInvocationSpec.ChaincodeSpec.Input)
	if err != nil {
		return nil, uuid, nil, fmt.Errorf("Error invoking chaincode: %s", err)
	}
	if resp.Status != shim.OK {
		return nil, uuid, nil, fmt.Errorf("Error invoking chaincode: %s", resp.Message)
	}

	return ccevt, uuid, resp.Payload, err
}

func closeListenerAndSleep(l net.Listener) {
	if l != nil {
		l.Close()
		time.Sleep(2 * time.Second)
	}
}

func executeDeployTransaction(t *testing.T, chainID string, name string, url string) {
	_, chaincodeSupport, cleanup, err := initPeer(chainID)
	if err != nil {
		t.Fail()
		t.Logf("Error creating peer: %s", err)
	}

	defer cleanup()

	f := "init"
	args := util.ToChaincodeArgs(f, "a", "100", "b", "200")
	spec := &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: name, Path: url, Version: "0"}, Input: &pb.ChaincodeInput{Args: args}}

	cccid := &ccprovider.CCContext{
		Name:    name,
		Version: "0",
	}

	defer chaincodeSupport.Stop(
		ccprovider.DeploymentSpecToChaincodeContainerInfo(
			&pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec},
		),
	)

	_, err = deploy(chainID, cccid, spec, 0, chaincodeSupport)

	cID := spec.ChaincodeId.Name
	if err != nil {
		t.Fail()
		t.Logf("Error deploying <%s>: %s", cID, err)
		return
	}
}

//检查事务执行后最终状态的正确性。
func checkFinalState(chainID string, cccid *ccprovider.CCContext, a int, b int) error {
	txid := util.GenerateUUID()
	txsim, _, err := startTxSimulation(chainID, txid)
	if err != nil {
		return fmt.Errorf("Failed to get handle to simulator: %s ", err)
	}

	defer txsim.Done()

	cName := cccid.GetCanonicalName()

//调用分类帐以获取状态
	var Aval, Bval int
	resbytes, resErr := txsim.GetState(cccid.Name, "a")
	if resErr != nil {
		return fmt.Errorf("Error retrieving state from ledger for <%s>: %s", cName, resErr)
	}
	fmt.Printf("Got string: %s\n", string(resbytes))
	Aval, resErr = strconv.Atoi(string(resbytes))
	if resErr != nil {
		return fmt.Errorf("Error retrieving state from ledger for <%s>: %s", cName, resErr)
	}
	if Aval != a {
		return fmt.Errorf("Incorrect result. Aval %d != %d <%s>", Aval, a, cName)
	}

	resbytes, resErr = txsim.GetState(cccid.Name, "b")
	if resErr != nil {
		return fmt.Errorf("Error retrieving state from ledger for <%s>: %s", cName, resErr)
	}
	Bval, resErr = strconv.Atoi(string(resbytes))
	if resErr != nil {
		return fmt.Errorf("Error retrieving state from ledger for <%s>: %s", cName, resErr)
	}
	if Bval != b {
		return fmt.Errorf("Incorrect result. Bval %d != %d <%s>", Bval, b, cName)
	}

//成功
	fmt.Printf("Aval = %d, Bval = %d\n", Aval, Bval)
	return nil
}

const (
	chaincodeExample02GolangPath   = "github.com/hyperledger/fabric/examples/chaincode/go/example02/cmd"
	chaincodeExample04GolangPath   = "github.com/hyperledger/fabric/examples/chaincode/go/example04/cmd"
	chaincodeEventSenderGolangPath = "github.com/hyperledger/fabric/examples/chaincode/go/eventsender"
	chaincodePassthruGolangPath    = "github.com/hyperledger/fabric/examples/chaincode/go/passthru"
	chaincodeExample02JavaPath     = "../../examples/chaincode/java/chaincode_example02"
	chaincodeExample04JavaPath     = "../../examples/chaincode/java/chaincode_example04"
	chaincodeExample06JavaPath     = "../../examples/chaincode/java/chaincode_example06"
	chaincodeEventSenderJavaPath   = "../../examples/chaincode/java/eventsender"
)

func runChaincodeInvokeChaincode(t *testing.T, channel1 string, channel2 string, tc tcicTc, cccid1 *ccprovider.CCContext, expectedA int, expectedB int, nextBlockNumber1, nextBlockNumber2 uint64, chaincodeSupport *ChaincodeSupport) (uint64, uint64) {
	var ctxt = context.Background()

//chaincode2:将由chaincode1调用的chaincode
	chaincode2Name := generateChaincodeName(tc.chaincodeType)
	chaincode2Version := "0"
	chaincode2Type := tc.chaincodeType
	chaincode2Path := tc.chaincodePath
	chaincode2InitArgs := util.ToChaincodeArgs("init", "e", "0")
	chaincode2Creator := []byte([]byte("Alice"))

//在通道1上部署第二个链代码
	_, cccid2, err := deployChaincode(
		ctxt,
		chaincode2Name,
		chaincode2Version,
		chaincode2Type,
		chaincode2Path,
		chaincode2InitArgs,
		chaincode2Creator,
		channel1,
		nextBlockNumber1,
		chaincodeSupport,
	)
	if err != nil {
		stopChaincode(ctxt, cccid1, chaincodeSupport)
		stopChaincode(ctxt, cccid2, chaincodeSupport)
		t.Fatalf("Error initializing chaincode %s(%s)", chaincode2Name, err)
		return nextBlockNumber1, nextBlockNumber2
	}
	nextBlockNumber1++

	time.Sleep(time.Second)

//调用第二个chaincode，将第一个chaincode的名称作为第一个参数传递，
//它将通过直觉调用第一个链代码
	chaincode2InvokeSpec := &pb.ChaincodeSpec{
		Type: chaincode2Type,
		ChaincodeId: &pb.ChaincodeID{
			Name:    chaincode2Name,
			Version: chaincode2Version,
		},
		Input: &pb.ChaincodeInput{
			Args: util.ToChaincodeArgs("invoke", cccid1.Name, "e", "1"),
		},
	}
//调用链代码
	_, _, _, err = invoke(channel1, chaincode2InvokeSpec, nextBlockNumber1, []byte("Alice"), chaincodeSupport)
	if err != nil {
		stopChaincode(ctxt, cccid1, chaincodeSupport)
		stopChaincode(ctxt, cccid2, chaincodeSupport)
		t.Fatalf("Error invoking <%s>: %s", chaincode2Name, err)
		return nextBlockNumber1, nextBlockNumber2
	}
	nextBlockNumber1++

//检查分类帐中的状态
	err = checkFinalState(channel1, cccid1, expectedA, expectedB)
	if err != nil {
		stopChaincode(ctxt, cccid1, chaincodeSupport)
		stopChaincode(ctxt, cccid2, chaincodeSupport)
		t.Fatalf("Incorrect final state after transaction for <%s>: %s", cccid1.Name, err)
		return nextBlockNumber1, nextBlockNumber2
	}

//通过以下方式改变两个渠道的政策：
//1。爱丽丝有两个频道的读卡器。
//2。Bob只能访问chaineid2。
//因此，链式代码调用应该失败。
	pm := peer.GetPolicyManager(channel1)
	pm.(*mockpolicies.Manager).PolicyMap = map[string]policies.Policy{
		policies.ChannelApplicationWriters: &CreatorPolicy{Creators: [][]byte{[]byte("Alice")}},
	}

	pm = peer.GetPolicyManager(channel2)
	pm.(*mockpolicies.Manager).PolicyMap = map[string]policies.Policy{
		policies.ChannelApplicationWriters: &CreatorPolicy{Creators: [][]byte{[]byte("Alice"), []byte("Bob")}},
	}

//在通道2上部署链代码2
	_, cccid3, err := deployChaincode(
		ctxt,
		chaincode2Name,
		chaincode2Version,
		chaincode2Type,
		chaincode2Path,
		chaincode2InitArgs,
		chaincode2Creator,
		channel2,
		nextBlockNumber2,
		chaincodeSupport,
	)

	if err != nil {
		stopChaincode(ctxt, cccid1, chaincodeSupport)
		stopChaincode(ctxt, cccid2, chaincodeSupport)
		stopChaincode(ctxt, cccid3, chaincodeSupport)
		t.Fatalf("Error initializing chaincode %s/%s: %s", chaincode2Name, channel2, err)
		return nextBlockNumber1, nextBlockNumber2
	}
	nextBlockNumber2++
	time.Sleep(time.Second)

	chaincode2InvokeSpec = &pb.ChaincodeSpec{
		Type: chaincode2Type,
		ChaincodeId: &pb.ChaincodeID{
			Name:    chaincode2Name,
			Version: chaincode2Version,
		},
		Input: &pb.ChaincodeInput{
			Args: util.ToChaincodeArgs("invoke", cccid1.Name, "e", "1", channel1),
		},
	}

//作为bob，在channel2上调用chaincode2，以便在channel1上调用chaincode1
	_, _, _, err = invoke(channel2, chaincode2InvokeSpec, nextBlockNumber2, []byte("Bob"), chaincodeSupport)
	if err == nil {
//鲍勃不应该打电话来
		stopChaincode(ctxt, cccid1, chaincodeSupport)
		stopChaincode(ctxt, cccid2, chaincodeSupport)
		stopChaincode(ctxt, cccid3, chaincodeSupport)
		nextBlockNumber2++
		t.Fatalf("As Bob, invoking <%s/%s> via <%s/%s> should fail, but it succeeded.", cccid1.Name, channel1, chaincode2Name, channel2)
		return nextBlockNumber1, nextBlockNumber2
	}
	assert.True(t, strings.Contains(err.Error(), "[Creator not recognized [Bob]]"))

//作为Alice，在通道2上调用chaincode2，以便在通道1上调用chaincode1
	_, _, _, err = invoke(channel2, chaincode2InvokeSpec, nextBlockNumber2, []byte("Alice"), chaincodeSupport)
	if err != nil {
//爱丽丝应该可以打电话
		stopChaincode(ctxt, cccid1, chaincodeSupport)
		stopChaincode(ctxt, cccid2, chaincodeSupport)
		stopChaincode(ctxt, cccid3, chaincodeSupport)
		t.Fatalf("As Alice, invoking <%s/%s> via <%s/%s> should should of succeeded, but it failed: %s", cccid1.Name, channel1, chaincode2Name, channel2, err)
		return nextBlockNumber1, nextBlockNumber2
	}
	nextBlockNumber2++

	stopChaincode(ctxt, cccid1, chaincodeSupport)
	stopChaincode(ctxt, cccid2, chaincodeSupport)
	stopChaincode(ctxt, cccid3, chaincodeSupport)

	return nextBlockNumber1, nextBlockNumber2
}

//测试无效事务的执行。
//TestChaincodeInvokeChaincode的测试用例参数
type tcicTc struct {
	chaincodeType pb.ChaincodeSpec_Type
	chaincodePath string
}

//测试调用另一个链代码的链代码的执行。
func TestChaincodeInvokeChaincode(t *testing.T) {
	channel := util.GetTestChainID()
	channel2 := channel + "2"
	lis, chaincodeSupport, cleanup, err := initPeer(channel, channel2)
	if err != nil {
		t.Fail()
		t.Logf("Error creating peer: %s", err)
	}
	defer cleanup()

	mockAclProvider.On("CheckACL", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	testCases := []tcicTc{
		{pb.ChaincodeSpec_GOLANG, chaincodeExample04GolangPath},
	}

	ctx := context.Background()

	var nextBlockNumber1 uint64 = 1
	var nextBlockNumber2 uint64 = 1

//部署将由第二个链代码调用的链代码
	chaincode1Name := generateChaincodeName(pb.ChaincodeSpec_GOLANG)
	chaincode1Version := "0"
	chaincode1Type := pb.ChaincodeSpec_GOLANG
	chaincode1Path := chaincodeExample02GolangPath
	initialA := 100
	initialB := 200
	chaincode1InitArgs := util.ToChaincodeArgs("init", "a", strconv.Itoa(initialA), "b", strconv.Itoa(initialB))
	chaincode1Creator := []byte([]byte("Alice"))

//
	_, chaincodeCtx, err := deployChaincode(
		ctx,
		chaincode1Name,
		chaincode1Version,
		chaincode1Type,
		chaincode1Path,
		chaincode1InitArgs,
		chaincode1Creator,
		channel,
		nextBlockNumber1,
		chaincodeSupport,
	)
	if err != nil {
		stopChaincode(ctx, chaincodeCtx, chaincodeSupport)
		t.Fatalf("Error initializing chaincode %s: %s", chaincodeCtx.Name, err)
	}
	nextBlockNumber1++
	time.Sleep(time.Second)

	expectedA := initialA
	expectedB := initialB

	for _, tc := range testCases {
		t.Run(tc.chaincodeType.String(), func(t *testing.T) {
			expectedA = expectedA - 10
			expectedB = expectedB + 10
			nextBlockNumber1, nextBlockNumber2 = runChaincodeInvokeChaincode(
				t,
				channel,
				channel2,
				tc,
				chaincodeCtx,
				expectedA,
				expectedB,
				nextBlockNumber1,
				nextBlockNumber2,
				chaincodeSupport,
			)
		})
	}

	closeListenerAndSleep(lis)
}

func stopChaincode(ctx context.Context, chaincodeCtx *ccprovider.CCContext, chaincodeSupport *ChaincodeSupport) {
	chaincodeSupport.Stop(&ccprovider.ChaincodeContainerInfo{
		Name:          chaincodeCtx.Name,
		Version:       chaincodeCtx.Version,
		ContainerType: "DOCKER",
		Type:          "GOLANG",
	})
}

//测试使用错误参数调用另一个链代码的链代码的执行。应该从接收错误
//从被调用的链代码
func TestChaincodeInvokeChaincodeErrorCase(t *testing.T) {
	chainID := util.GetTestChainID()

	_, chaincodeSupport, cleanup, err := initPeer(chainID)
	if err != nil {
		t.Fail()
		t.Logf("Error creating peer: %s", err)
	}
	defer cleanup()

	mockAclProvider.On("CheckACL", mock.Anything, mock.Anything, mock.Anything).Return(nil)

//
	cID1 := &pb.ChaincodeID{Name: "example02", Path: chaincodeExample02GolangPath, Version: "0"}
	f := "init"
	args := util.ToChaincodeArgs(f, "a", "100", "b", "200")

	spec1 := &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID1, Input: &pb.ChaincodeInput{Args: args}}

	cccid1 := &ccprovider.CCContext{
		Name:    "example02",
		Version: "0",
	}

	var nextBlockNumber uint64 = 1
	defer chaincodeSupport.Stop(&ccprovider.ChaincodeContainerInfo{
		Name:          cID1.Name,
		Version:       cID1.Version,
		Path:          cID1.Path,
		ContainerType: "DOCKER",
		Type:          "GOLANG",
	})

	_, err = deploy(chainID, cccid1, spec1, nextBlockNumber, chaincodeSupport)
	nextBlockNumber++
	ccID1 := spec1.ChaincodeId.Name
	if err != nil {
		t.Fail()
		t.Logf("Error initializing chaincode %s(%s)", ccID1, err)
		return
	}

	time.Sleep(time.Second)

//部署第二个链代码
	cID2 := &pb.ChaincodeID{Name: "pthru", Path: chaincodePassthruGolangPath, Version: "0"}
	f = "init"
	args = util.ToChaincodeArgs(f)

	spec2 := &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID2, Input: &pb.ChaincodeInput{Args: args}}

	cccid2 := &ccprovider.CCContext{
		Name:    "pthru",
		Version: "0",
	}

	defer chaincodeSupport.Stop(&ccprovider.ChaincodeContainerInfo{
		Name:          cID2.Name,
		Version:       cID2.Version,
		Path:          cID2.Path,
		ContainerType: "DOCKER",
		Type:          "GOLANG",
	})
	_, err = deploy(chainID, cccid2, spec2, nextBlockNumber, chaincodeSupport)
	nextBlockNumber++
	ccID2 := spec2.ChaincodeId.Name
	if err != nil {
		t.Fail()
		t.Logf("Error initializing chaincode %s(%s)", ccID2, err)
		return
	}

	time.Sleep(time.Second)

//
	f = ccID1
	args = util.ToChaincodeArgs(f, "invoke", "a")

	spec2 = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID2, Input: &pb.ChaincodeInput{Args: args}}
//调用链代码
	_, _, _, err = invoke(chainID, spec2, nextBlockNumber, []byte("Alice"), chaincodeSupport)

	if err == nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID2, err)
		return
	}

	if strings.Index(err.Error(), "Error invoking chaincode: Incorrect number of arguments. Expecting 3") < 0 {
		t.Fail()
		t.Logf("Unexpected error %s", err)
		return
	}
}

func TestChaincodeInit(t *testing.T) {
	chainID := util.GetTestChainID()

	_, chaincodeSupport, cleanup, err := initPeer(chainID)
	if err != nil {
		t.Fail()
		t.Logf("Error creating peer: %s", err)
	}

	defer cleanup()

	url := "github.com/hyperledger/fabric/core/chaincode/testdata/chaincode/init_private_data"
	cID := &pb.ChaincodeID{Name: "init_pvtdata", Path: url, Version: "0"}

	f := "init"
	args := util.ToChaincodeArgs(f)

	spec := &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}

	cccid := &ccprovider.CCContext{
		Name:    "tmap",
		Version: "0",
	}

	defer chaincodeSupport.Stop(&ccprovider.ChaincodeContainerInfo{
		Name:          cID.Name,
		Version:       cID.Version,
		Path:          cID.Path,
		Type:          "GOLANG",
		ContainerType: "DOCKER",
	})

	var nextBlockNumber uint64 = 1
	_, err = deploy(chainID, cccid, spec, nextBlockNumber, chaincodeSupport)
	assert.Contains(t, err.Error(), "private data APIs are not allowed in chaincode Init")

	url = "github.com/hyperledger/fabric/core/chaincode/testdata/chaincode/init_public_data"
	cID = &pb.ChaincodeID{Name: "init_public_data", Path: url, Version: "0"}

	f = "init"
	args = util.ToChaincodeArgs(f)

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}

	cccid = &ccprovider.CCContext{
		Name:    "tmap",
		Version: "0",
	}

	defer chaincodeSupport.Stop(&ccprovider.ChaincodeContainerInfo{
		Name:          cID.Name,
		Version:       cID.Version,
		Path:          cID.Path,
		Type:          "GOLANG",
		ContainerType: "DOCKER",
	})

	resp, err := deploy(chainID, cccid, spec, nextBlockNumber, chaincodeSupport)
	assert.NoError(t, err)
//当状态代码为
//定义为int（即常量）
	assert.Equal(t, int32(shim.OK), resp.Status)
}

//测试事务的调用。
func TestQueries(t *testing.T) {
//允许单独进行查询测试，以便可以执行端到端测试。不到5秒。
//TestFurkip（t）

	chainID := util.GetTestChainID()

	_, chaincodeSupport, cleanup, err := initPeer(chainID)
	if err != nil {
		t.Fail()
		t.Logf("Error creating peer: %s", err)
	}

	defer cleanup()

	url := "github.com/hyperledger/fabric/examples/chaincode/go/map"
	cID := &pb.ChaincodeID{Name: "tmap", Path: url, Version: "0"}

	f := "init"
	args := util.ToChaincodeArgs(f)

	spec := &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}

	cccid := &ccprovider.CCContext{
		Name:    "tmap",
		Version: "0",
	}

	defer chaincodeSupport.Stop(&ccprovider.ChaincodeContainerInfo{
		Name:          cID.Name,
		Version:       cID.Version,
		Path:          cID.Path,
		Type:          "GOLANG",
		ContainerType: "DOCKER",
	})

	var nextBlockNumber uint64 = 1
	_, err = deploy(chainID, cccid, spec, nextBlockNumber, chaincodeSupport)
	nextBlockNumber++
	ccID := spec.ChaincodeId.Name
	if err != nil {
		t.Fail()
		t.Logf("Error initializing chaincode %s(%s)", ccID, err)
		return
	}

	var keys []interface{}
//添加101个大理石用于测试范围查询和丰富的查询（对于有能力的分类账）
//
	for i := 1; i <= 101; i++ {
		f = "put"

//51归汤姆所有，50归杰瑞所有
		owner := "tom"
		if i%2 == 0 {
			owner = "jerry"
		}

//一个大理石颜色是红色，100个是蓝色
		color := "blue"
		if i == 12 {
			color = "red"
		}

		key := fmt.Sprintf("marble%03d", i)
		argsString := fmt.Sprintf("{\"docType\":\"marble\",\"name\":\"%s\",\"color\":\"%s\",\"size\":35,\"owner\":\"%s\"}", key, color, owner)
		args = util.ToChaincodeArgs(f, key, argsString)
		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

	}

//“Marble001”到“Marble011”的以下范围查询应返回10个Marbles
	f = "keys"
	args = util.ToChaincodeArgs(f, "marble001", "marble011")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err := invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	err = json.Unmarshal(retval, &keys)
	if len(keys) != 10 {
		t.Fail()
		t.Logf("Error detected with the range query, should have returned 10 but returned %v", len(keys))
		return
	}

//FAB-1163-以下范围查询应超时并产生错误
//同伴应该优雅地处理这个问题，而不是死去。

//保存原始超时并将新超时设置为1秒
	origTimeout := chaincodeSupport.ExecuteTimeout
	chaincodeSupport.ExecuteTimeout = time.Duration(1) * time.Second

//链码休眠2秒，超时1
	args = util.ToChaincodeArgs(f, "marble001", "marble002", "2000")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	if err == nil {
		t.Fail()
		t.Logf("expected timeout error but succeeded")
		return
	}

//恢复超时
	chaincodeSupport.ExecuteTimeout = origTimeout

//查询所有大理石将返回101个大理石
//此查询应返回正好101个结果（一个对next（）的调用）
//“marble001”到“marble102”的以下范围查询应返回101个marbles。
	f = "keys"
	args = util.ToChaincodeArgs(f, "marble001", "marble102")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//取消标记结果
	err = json.Unmarshal(retval, &keys)

//检查是否有101个值
//使用默认查询限制10000，此查询实际上是不受限制的
	if len(keys) != 101 {
		t.Fail()
		t.Logf("Error detected with the range query, should have returned 101 but returned %v", len(keys))
		return
	}

//正在查询所有简单键。此查询应返回正好101个简单键（一个
//
//以下“to”的开放式范围查询应返回101个大理石
	f = "keys"
	args = util.ToChaincodeArgs(f, "", "")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//取消标记结果
	err = json.Unmarshal(retval, &keys)

//检查是否有101个值
//使用默认查询限制10000，此查询实际上是不受限制的
	if len(keys) != 101 {
		t.Fail()
		t.Logf("Error detected with the range query, should have returned 101 but returned %v", len(keys))
		return
	}

	type PageResponse struct {
		Bookmark string   `json:"bookmark"`
		Keys     []string `json:"keys"`
	}

//“Marble001”到“Marble011”的以下范围查询应返回10个Marbles，共2页
	f = "keysByPage"
	args = util.ToChaincodeArgs(f, "marble001", "marble011", "2", "")

	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}
	queryPage := &PageResponse{}

	json.Unmarshal(retval, &queryPage)

	expectedResult := []string{"marble001", "marble002"}

	if !reflect.DeepEqual(expectedResult, queryPage.Keys) {
		t.Fail()
		t.Logf("Error detected with the paginated range query. Returned: %v  should have returned: %v", queryPage.Keys, expectedResult)
		return
	}

//查询下一页
	args = util.ToChaincodeArgs(f, "marble001", "marble011", "2", queryPage.Bookmark)
	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	json.Unmarshal(retval, &queryPage)

	expectedResult = []string{"marble003", "marble004"}

	if !reflect.DeepEqual(expectedResult, queryPage.Keys) {
		t.Fail()
		t.Logf("Error detected with the paginated range query second page. Returned: %v  should have returned: %v    %v", queryPage.Keys, expectedResult, queryPage.Bookmark)
		return
	}

//只有CouchDB和
//查询限制仅适用于CouchDB范围和富查询
	if ledgerconfig.IsCouchDBEnabled() == true {

//用于垫片配料的角箱。电流垫片批量为100
//此查询应返回100个结果（不调用next（））
		f = "query"
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"color\":\"blue\"}}")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)

//检查是否有100个值
		if len(keys) != 100 {
			t.Fail()
			t.Logf("Error detected with the rich query, should have returned 100 but returned %v %s", len(keys), keys)
			return
		}
//将查询限制重置为5
		viper.Set("ledger.state.queryLimit", 5)

//由于查询限制，“Marble01”到“Marble11”的以下范围查询应返回5个Marbles。
		f = "keys"
		args = util.ToChaincodeArgs(f, "marble001", "marble011")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err := invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++
		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)
//检查是否有5个值
		if len(keys) != 5 {
			t.Fail()
			t.Logf("Error detected with the range query, should have returned 5 but returned %v", len(keys))
			return
		}

//将查询限制重置为10000
		viper.Set("ledger.state.queryLimit", 10000)

//以下丰富的查询应返回50个大理石
		f = "query"
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"owner\":\"jerry\"}}")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++

		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)

//检查是否有50个值
//使用默认查询限制10000，此查询实际上是不受限制的
		if len(keys) != 50 {
			t.Fail()
			t.Logf("Error detected with the rich query, should have returned 50 but returned %v", len(keys))
			return
		}

//将查询限制重置为5
		viper.Set("ledger.state.queryLimit", 5)

//由于查询限制，下面的富查询应返回5个大理石。
		f = "query"
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"owner\":\"jerry\"}}")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++
		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

//取消标记结果
		err = json.Unmarshal(retval, &keys)

//检查是否有5个值
		if len(keys) != 5 {
			t.Fail()
			t.Logf("Error detected with the rich query, should have returned 5 but returned %v", len(keys))
			return
		}

//由于页面大小的原因，下面的富查询应返回2个大理石。
		f = "queryByPage"
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"owner\":\"jerry\"}}", "2", "")

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++
		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

		queryPage := &PageResponse{}

		json.Unmarshal(retval, &queryPage)

		expectedResult := []string{"marble001", "marble003"}

		if !reflect.DeepEqual(expectedResult, queryPage.Keys) {
			t.Fail()
			t.Logf("Error detected with the paginated range query. Returned: %v  should have returned: %v", queryPage.Keys, expectedResult)
			return
		}

//为下一页设置参数
		args = util.ToChaincodeArgs(f, "{\"selector\":{\"owner\":\"jerry\"}}", "2", queryPage.Bookmark)

		spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
		_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
		nextBlockNumber++
		if err != nil {
			t.Fail()
			t.Logf("Error invoking <%s>: %s", ccID, err)
			return
		}

		queryPage = &PageResponse{}

		json.Unmarshal(retval, &queryPage)

		expectedResult = []string{"marble005", "marble007"}

		if !reflect.DeepEqual(expectedResult, queryPage.Keys) {
			t.Fail()
			t.Logf("Error detected with the paginated range query. Returned: %v  should have returned: %v", queryPage.Keys, expectedResult)
			return
		}

	}

//历史查询修改
	f = "put"
	args = util.ToChaincodeArgs(f, "marble012", "{\"docType\":\"marble\",\"name\":\"marble012\",\"color\":\"red\",\"size\":30,\"owner\":\"jerry\"}")
	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	f = "put"
	args = util.ToChaincodeArgs(f, "marble012", "{\"docType\":\"marble\",\"name\":\"marble012\",\"color\":\"red\",\"size\":30,\"owner\":\"jerry\"}")
	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, _, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

//“Marble12”的以下历史查询应返回3条记录
	f = "history"
	args = util.ToChaincodeArgs(f, "marble012")
	spec = &pb.ChaincodeSpec{Type: 1, ChaincodeId: cID, Input: &pb.ChaincodeInput{Args: args}}
	_, _, retval, err = invoke(chainID, spec, nextBlockNumber, nil, chaincodeSupport)
	nextBlockNumber++
	if err != nil {
		t.Fail()
		t.Logf("Error invoking <%s>: %s", ccID, err)
		return
	}

	var history []interface{}
	err = json.Unmarshal(retval, &history)
	if len(history) != 3 {
		t.Fail()
		t.Logf("Error detected with the history query, should have returned 3 but returned %v", len(history))
		return
	}
}

func TestMain(m *testing.M) {
	var err error

	msptesttools.LoadMSPSetupForTesting()
	signer, err = mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		fmt.Print("Could not initialize msp/signer")
		os.Exit(-1)
		return
	}

	setupTestConfig()
	flogging.ActivateSpec("chaincode=debug")
	os.Exit(m.Run())
}

func setupTestConfig() {
	flag.Parse()

//现在设置配置文件
	viper.SetEnvPrefix("CORE")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
viper.SetConfigName("chaincodetest") //配置文件名（不带扩展名）
viper.AddConfigPath("./")            //查找配置文件的路径
err := viper.ReadInConfig()          //查找并读取配置文件
if err != nil {                      //处理读取配置文件时的错误
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

//设置maxprocs的数目
	var numProcsDesired = viper.GetInt("peer.gomaxprocs")
	chaincodeLogger.Debugf("setting Number of procs to %d, was %d\n", numProcsDesired, runtime.GOMAXPROCS(numProcsDesired))

//初始化BCCSP
	err = factory.InitFactories(nil)
	if err != nil {
		panic(fmt.Errorf("Could not initialize BCCSP Factories [%s]", err))
	}
}

func deployChaincode(ctx context.Context, name string, version string, chaincodeType pb.ChaincodeSpec_Type, path string, args [][]byte, creator []byte, channel string, nextBlockNumber uint64, chaincodeSupport *ChaincodeSupport) (*pb.Response, *ccprovider.CCContext, error) {
	chaincodeSpec := &pb.ChaincodeSpec{
		ChaincodeId: &pb.ChaincodeID{
			Name:    name,
			Version: version,
			Path:    path,
		},
		Type: chaincodeType,
		Input: &pb.ChaincodeInput{
			Args: args,
		},
	}

	chaincodeCtx := &ccprovider.CCContext{
		Name:    name,
		Version: version,
	}

	result, err := deploy(channel, chaincodeCtx, chaincodeSpec, nextBlockNumber, chaincodeSupport)
	if err != nil {
		return nil, chaincodeCtx, fmt.Errorf("Error deploying <%s:%s>: %s", name, version, err)
	}
	return result, chaincodeCtx, nil
}

var signer msp.SigningIdentity

var rng *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateChaincodeName(chaincodeType pb.ChaincodeSpec_Type) string {
	prefix := "cc_"
	switch chaincodeType {
	case pb.ChaincodeSpec_GOLANG:
		prefix = "cc_go_"
	case pb.ChaincodeSpec_JAVA:
		prefix = "cc_java_"
	case pb.ChaincodeSpec_NODE:
		prefix = "cc_js_"
	}
	return fmt.Sprintf("%s%06d", prefix, rng.Intn(999999))
}

type CreatorPolicy struct {
	Creators [][]byte
}

//Evaluate获取一组SignedData并评估该组签名是否满足策略
func (c *CreatorPolicy) Evaluate(signatureSet []*common.SignedData) error {
	for _, value := range c.Creators {
		if bytes.Compare(signatureSet[0].Identity, value) == 0 {
			return nil
		}
	}
	return fmt.Errorf("Creator not recognized [%s]", string(signatureSet[0].Identity))
}

type mockPolicyCheckerFactory struct{}

func (f *mockPolicyCheckerFactory) NewPolicyChecker() policy.PolicyChecker {
	return policy.NewPolicyChecker(
		peer.NewChannelPolicyManagerGetter(),
		&mocks.MockIdentityDeserializer{
			Identity: []byte("Admin"),
			Msg:      []byte("msg1"),
		},
		&mocks.MockMSPPrincipalGetter{Principal: []byte("Admin")},
	)
}
