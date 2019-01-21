
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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	mc "github.com/hyperledger/fabric/common/mocks/config"
	mocklgr "github.com/hyperledger/fabric/common/mocks/ledger"
	mockpeer "github.com/hyperledger/fabric/common/mocks/peer"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/aclmgmt/mocks"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/chaincode/accesscontrol"
	"github.com/hyperledger/fabric/core/chaincode/mock"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/dockercontroller"
	"github.com/hyperledger/fabric/core/container/inproccontroller"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	cmp "github.com/hyperledger/fabric/core/mocks/peer"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/core/scc"
	"github.com/hyperledger/fabric/core/scc/lscc"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	plgr "github.com/hyperledger/fabric/protos/ledger/queryresult"
	pb "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

var globalBlockNum map[string]uint64

type mockResultsIterator struct {
	current int
	kvs     []*plgr.KV
}

func (mri *mockResultsIterator) Next() (commonledger.QueryResult, error) {
	if mri.current == len(mri.kvs) {
		return nil, nil
	}
	kv := mri.kvs[mri.current]
	mri.current = mri.current + 1

	return kv, nil
}

func (mri *mockResultsIterator) Close() {
	mri.current = len(mri.kvs)
}

type mockExecQuerySimulator struct {
	txsim ledger.TxSimulator
	mocklgr.MockQueryExecutor
	resultsIter map[string]map[string]*mockResultsIterator
}

func (meqe *mockExecQuerySimulator) GetHistoryForKey(namespace, query string) (commonledger.ResultsIterator, error) {
	return meqe.commonQuery(namespace, query)
}

func (meqe *mockExecQuerySimulator) ExecuteQuery(namespace, query string) (commonledger.ResultsIterator, error) {
	return meqe.commonQuery(namespace, query)
}

func (meqe *mockExecQuerySimulator) commonQuery(namespace, query string) (commonledger.ResultsIterator, error) {
	if meqe.resultsIter == nil {
		return nil, fmt.Errorf("query executor not initialized")
	}
	nsiter := meqe.resultsIter[namespace]
	if nsiter == nil {
		return nil, fmt.Errorf("namespace %v not found for %s", namespace, query)
	}
	iter := nsiter[query]
	if iter == nil {
		fmt.Printf("iter not found for query %s\n", query)
	}
	return iter, nil
}

func (meqe *mockExecQuerySimulator) SetState(namespace string, key string, value []byte) error {
	if meqe.txsim == nil {
		return fmt.Errorf("SetState txsimulator not initialed")
	}
	return meqe.txsim.SetState(namespace, key, value)
}

func (meqe *mockExecQuerySimulator) DeleteState(namespace string, key string) error {
	if meqe.txsim == nil {
		return fmt.Errorf("SetState txsimulator not initialed")
	}
	return meqe.txsim.DeleteState(namespace, key)
}

func (meqe *mockExecQuerySimulator) SetStateMultipleKeys(namespace string, kvs map[string][]byte) error {
	if meqe.txsim == nil {
		return fmt.Errorf("SetState txsimulator not initialed")
	}
	return meqe.txsim.SetStateMultipleKeys(namespace, kvs)
}

func (meqe *mockExecQuerySimulator) ExecuteUpdate(query string) error {
	if meqe.txsim == nil {
		return fmt.Errorf("SetState txsimulator not initialed")
	}
	return meqe.txsim.ExecuteUpdate(query)
}

func (meqe *mockExecQuerySimulator) GetTxSimulationResults() ([]byte, error) {
	if meqe.txsim == nil {
		return nil, fmt.Errorf("SetState txsimulator not initialed")
	}
	simRes, err := meqe.txsim.GetTxSimulationResults()
	if err != nil {
		return nil, err
	}
	return simRes.GetPubSimulationBytes()
}

var mockAclProvider *mocks.MockACLProvider

//初始化对等机并启动。如果security==enabled，则以vp身份登录
func initMockPeer(chainIDs ...string) (*ChaincodeSupport, error) {
	msi := &cmp.MockSupportImpl{
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetApplicationConfigBoolRv: true,
	}

	ipRegistry := inproccontroller.NewRegistry()
	sccp := &scc.Provider{Peer: peer.Default, PeerSupport: msi, Registrar: ipRegistry}

	mockAclProvider = &mocks.MockACLProvider{}
	mockAclProvider.Reset()

	peer.MockInitialize()

	mspGetter := func(cid string) []string {
		return []string{"SampleOrg"}
	}

	peer.MockSetMSPIDGetter(mspGetter)

	ccprovider.SetChaincodesPath(ccprovider.GetChaincodeInstallPathFromViper())
	ca, _ := tlsgen.NewCA()
	certGenerator := accesscontrol.NewAuthenticator(ca)
	config := GlobalConfig()
	config.StartupTimeout = 10 * time.Second
	config.ExecuteTimeout = 1 * time.Second
	pr := platforms.NewRegistry(&golang.Platform{})
	lsccImpl := lscc.New(sccp, mockAclProvider, pr)
	chaincodeSupport := NewChaincodeSupport(
		config,
		"0.0.0.0:7052",
		true,
		ca.CertBytes(),
		certGenerator,
		&ccprovider.CCInfoFSImpl{},
		lsccImpl,
		mockAclProvider,
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

//模拟策略检查器
	policy.RegisterPolicyCheckerFactory(&mockPolicyCheckerFactory{})

	ccp := &CCProviderImpl{cs: chaincodeSupport}

	sccp.RegisterSysCC(lsccImpl)

	globalBlockNum = make(map[string]uint64, len(chainIDs))
	for _, id := range chainIDs {
		if err := peer.MockCreateChain(id); err != nil {
			return nil, err
		}
		sccp.DeploySysCCs(id, ccp)
//除默认testchainid之外的任何链都没有设置msp->create one
		if id != util.GetTestChainID() {
			mspmgmt.XXXSetMSPManager(id, mspmgmt.GetManagerForChain(util.GetTestChainID()))
		}
		globalBlockNum[id] = 1
	}

	return chaincodeSupport, nil
}

func finitMockPeer(chainIDs ...string) {
	for _, c := range chainIDs {
		if lgr := peer.GetLedger(c); lgr != nil {
			lgr.Close()
		}
	}
	ledgermgmt.CleanupTestEnv()
	ledgerPath := config.GetPath("peer.fileSystemPath")
	os.RemoveAll(ledgerPath)
	os.RemoveAll(filepath.Join(os.TempDir(), "hyperledger"))
}

//在此处存储流CC映射
var mockPeerCCSupport = mockpeer.NewMockPeerSupport()

func mockChaincodeStreamGetter(name string) (shim.PeerChaincodeStream, error) {
	return mockPeerCCSupport.GetCC(name)
}

type mockCCLauncher struct {
	execTime *time.Duration
	resp     error
	retErr   error
	notfyb   bool
}

func (ccl *mockCCLauncher) launch(ctxt context.Context, notfy chan bool) error {
	if ccl.execTime != nil {
		time.Sleep(*ccl.execTime)
	}

//启动时无错误，通知
	if ccl.resp == nil {
		notfy <- ccl.notfyb
	}

	return ccl.retErr
}

func setupcc(name string) (*mockpeer.MockCCComm, *mockpeer.MockCCComm) {
	send := make(chan *pb.ChaincodeMessage)
	recv := make(chan *pb.ChaincodeMessage)
	peerSide, _ := mockPeerCCSupport.AddCC(name, recv, send)
	peerSide.SetName("peer")
	ccSide := mockPeerCCSupport.GetCCMirror(name)
	ccSide.SetPong(true)
	return peerSide, ccSide
}

//将此项分配给Done和FailNow并继续使用它们
func setuperror() chan error {
	return make(chan error)
}

func processDone(t *testing.T, done chan error, expecterr bool) {
	var err error
	if done != nil {
		err = <-done
	}
	if expecterr != (err != nil) {
		if err == nil {
			t.Fatalf("Expected error but got success")
		} else {
			t.Fatalf("Expected success but got error %s", err)
		}
	}
}

func startTx(t *testing.T, chainID string, cis *pb.ChaincodeInvocationSpec, txId string) (*ccprovider.TransactionParams, ledger.TxSimulator) {
	creator := []byte([]byte("Alice"))
	sprop, prop := putils.MockSignedEndorserProposalOrPanic(chainID, cis.ChaincodeSpec, creator, []byte("msg1"))
	txsim, hqe, err := startTxSimulation(chainID, txId)
	if err != nil {
		t.Fatalf("getting txsimulator failed %s", err)
	}

	txParams := &ccprovider.TransactionParams{
		ChannelID:            chainID,
		TxID:                 txId,
		Proposal:             prop,
		SignedProp:           sprop,
		TXSimulator:          txsim,
		HistoryQueryExecutor: hqe,
	}

	return txParams, txsim
}

func endTx(t *testing.T, txParams *ccprovider.TransactionParams, txsim ledger.TxSimulator, cis *pb.ChaincodeInvocationSpec) {
	if err := endTxSimulationCIS(txParams.ChannelID, cis.ChaincodeSpec.ChaincodeId, txParams.TxID, txsim, []byte("invoke"), true, cis, globalBlockNum[txParams.ChannelID]); err != nil {
		t.Fatalf("simulation failed with error %s", err)
	}
	globalBlockNum[txParams.ChannelID] = globalBlockNum[txParams.ChannelID] + 1
}

func execCC(t *testing.T, txParams *ccprovider.TransactionParams, ccSide *mockpeer.MockCCComm, cccid *ccprovider.CCContext, waitForERROR bool, expectExecErr bool, done chan error, cis *pb.ChaincodeInvocationSpec, respSet *mockpeer.MockResponseSet, chaincodeSupport *ChaincodeSupport) error {
	ccSide.SetResponses(respSet)

	resp, _, err := chaincodeSupport.Execute(txParams, cccid, cis.ChaincodeSpec.Input)
	if err == nil && resp.Status != shim.OK {
		err = errors.New(resp.Message)
	}

	if err == nil && expectExecErr {
		t.Fatalf("expected error but succeeded")
	} else if err != nil && !expectExecErr {
		t.Fatalf("exec failed with %s", err)
	}

//等待
	processDone(t, done, waitForERROR)

	return nil
}

//初始化cc-support-env并启动chaincode
func startCC(t *testing.T, channelID string, ccname string, chaincodeSupport *ChaincodeSupport) (*mockpeer.MockCCComm, *mockpeer.MockCCComm) {
	peerSide, ccSide := setupcc(ccname)
	defer mockPeerCCSupport.RemoveCC(ccname)

//向CCSUPPORT注册对等端
	go func() {
		chaincodeSupport.HandleChaincodeStream(peerSide)
	}()

	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	ccDone := make(chan struct{})
	defer close(ccDone)

//启动模拟对等机
	go func() {
		respSet := &mockpeer.MockResponseSet{
			DoneFunc:  errorFunc,
			ErrorFunc: nil,
			Responses: []*mockpeer.MockResponse{
				{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTERED}, RespMsg: nil},
				{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_READY}, RespMsg: nil},
			},
		}
		ccSide.SetResponses(respSet)
		ccSide.Run(ccDone)
	}()

	ccSide.Send(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_REGISTER, Payload: putils.MarshalOrPanic(&pb.ChaincodeID{Name: ccname + ":0"}), Txid: "0", ChannelId: channelID})

//等待初始化
	processDone(t, done, false)

	return peerSide, ccSide
}

func getTarGZ(t *testing.T, name string, contents []byte) []byte {
	startTime := time.Now()
	inputbuf := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(inputbuf)
	tr := tar.NewWriter(gw)
	size := int64(len(contents))

	tr.WriteHeader(&tar.Header{Name: name, Size: size, ModTime: startTime, AccessTime: startTime, ChangeTime: startTime})
	tr.Write(contents)
	tr.Close()
	gw.Close()
	ioutil.WriteFile("/tmp/t.gz", inputbuf.Bytes(), 0644)
	return inputbuf.Bytes()
}

//部署一个链代码——即构建和初始化。
func deployCC(t *testing.T, txParams *ccprovider.TransactionParams, cccid *ccprovider.CCContext, spec *pb.ChaincodeSpec, chaincodeSupport *ChaincodeSupport) {
//首先构建并获取部署规范
	code := getTarGZ(t, "src/dummy/dummy.go", []byte("code"))
	cds := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: code}

//忽略存在错误
	ccprovider.PutChaincodeIntoFS(cds)

	b := putils.MarshalOrPanic(cds)

	sysCCVers := util.GetSysCCVersion()

//将部署包装到lscc的调用规范中…
	lsccSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_GOLANG, ChaincodeId: &pb.ChaincodeID{Name: "lscc", Version: sysCCVers}, Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("deploy"), []byte(txParams.ChannelID), b}}}}

	lsccid := &ccprovider.CCContext{
		Name:    lsccSpec.ChaincodeSpec.ChaincodeId.Name,
		Version: sysCCVers,
	}

//写入LSCC
	if _, _, err := chaincodeSupport.Execute(txParams, lsccid, lsccSpec.ChaincodeSpec.Input); err != nil {
		t.Fatalf("Error deploying chaincode %v (err: %s)", cccid, err)
	}
}

func initializeCC(t *testing.T, chainID, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) error {
	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	chaincodeID := &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("init"), []byte("A"), []byte("100"), []byte("B"), []byte("200")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}

	txid := util.GenerateUUID()
	txParams, txsim := startTx(t, chainID, cis, txid)

//设置checkacl调用
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

//响应中的TxID错误（应为“1”），应失败
	resp := &mockpeer.MockResponse{
		RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION},
		RespMsg: &pb.ChaincodeMessage{
			Type:      pb.ChaincodeMessage_COMPLETED,
			Payload:   putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("init succeeded")}),
			Txid:      "unknowntxid",
			ChannelId: chainID,
		},
	}
	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{resp},
	}

	cccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)

//立即设置正确的TxID响应
	resp = &mockpeer.MockResponse{
		RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION},
		RespMsg: &pb.ChaincodeMessage{
			Type:      pb.ChaincodeMessage_COMPLETED,
			Payload:   putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("init succeeded")}),
			Txid:      txid,
			ChannelId: chainID,
		},
	}
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{resp},
	}

	badcccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "unknownver",
	}

//我们无法访问链码，因此无法从中得到响应。ProcessDone不会
//由链码流触发。我们只是希望面料出错。因此，完成时通过零
	execCC(t, txParams, ccSide, badcccid, false, true, nil, cis, respSet, chaincodeSupport)

//-------最后尝试成功初始化------
//一切都排好了
//正确注册的链码版本
//匹配TXID
//TXSIM上下文
//完全反应
//结束SIM卡的正确块号

	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "", Key: "A", Value: []byte("100")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "", Key: "B", Value: []byte("200")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), ChaincodeEvent: &pb.ChaincodeEvent{ChaincodeId: ccname}, Txid: txid, ChannelId: chainID}},
		},
	}

	execCC(t, txParams, ccSide, cccid, false, false, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)

	return nil
}

func invokeCC(t *testing.T, chainID, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) error {
	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	chaincodeID := &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid := util.GenerateUUID()
	txParams, txsim := startTx(t, chainID, cis, txid)

//设置checkacl调用
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: putils.MarshalOrPanic(&pb.GetState{Collection: "", Key: "A"}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: putils.MarshalOrPanic(&pb.GetState{Collection: "", Key: "B"}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "", Key: "A", Value: []byte("90")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "", Key: "B", Value: []byte("210")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "", Key: "TODEL", Value: []byte("-to-be-deleted-")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: txid, ChannelId: chainID}},
		},
	}

	cccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	execCC(t, txParams, ccSide, cccid, false, false, done, cis, respSet, chaincodeSupport)

//删除额外的变量
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: putils.MarshalOrPanic(&pb.GetState{Collection: "", Key: "TODEL"}), Txid: "3", ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_DEL_STATE, Payload: putils.MarshalOrPanic(&pb.DelState{Collection: "", Key: "TODEL"}), Txid: "3", ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: "3", ChannelId: chainID}},
		},
	}

	txParams.TxID = "3"
	execCC(t, txParams, ccSide, cccid, false, false, done, cis, respSet, chaincodeSupport)

//获取额外的var并删除它
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: putils.MarshalOrPanic(&pb.GetState{Collection: "", Key: "TODEL"}), Txid: "4", ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.ERROR, Message: "variable not found"}), Txid: "4", ChannelId: chainID}},
		},
	}

	txParams.TxID = "4"
	execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)

	return nil
}

func invokePrivateDataGetPutDelCC(t *testing.T, chainID, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) error {
	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	chaincodeID := &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("invokePrivateData")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid := util.GenerateUUID()
	txParams, txsim := startTx(t, chainID, cis, txid)

	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: putils.MarshalOrPanic(&pb.GetState{Collection: "", Key: "C"}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.ERROR, Message: "variable not found"}), Txid: txid, ChannelId: chainID}},
		},
	}

	cccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)

	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: putils.MarshalOrPanic(&pb.GetState{Collection: "c1", Key: "C"}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "c1", Key: "C", Value: []byte("310")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "c1", Key: "A", Value: []byte("100")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: putils.MarshalOrPanic(&pb.PutState{Collection: "c1", Key: "B", Value: []byte("100")}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_DEL_STATE, Payload: putils.MarshalOrPanic(&pb.DelState{Collection: "c2", Key: "C"}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: txid, ChannelId: chainID}},
		},
	}

	cccid = &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	execCC(t, txParams, ccSide, cccid, false, false, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)

	txid = util.GenerateUUID()
	txParams, txsim = startTx(t, chainID, cis, txid)

	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: putils.MarshalOrPanic(&pb.GetState{Collection: "c2", Key: "C"}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.ERROR, Message: "variable not found"}), Txid: txid, ChannelId: chainID}},
		},
	}

	cccid = &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)

	return nil
}

func getQueryStateByRange(t *testing.T, collection, chainID, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) error {
	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	chaincodeID := &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid := util.GenerateUUID()
	txParams, txsim := startTx(t, chainID, cis, txid)

//设置checkacl调用
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

//创建响应
	queryStateNextFunc := func(reqMsg *pb.ChaincodeMessage) *pb.ChaincodeMessage {
		qr := &pb.QueryResponse{}
		proto.Unmarshal(reqMsg.Payload, qr)
		return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Payload: putils.MarshalOrPanic(&pb.QueryStateNext{Id: qr.Id}), Txid: txid, ChannelId: chainID}
	}
	queryStateCloseFunc := func(reqMsg *pb.ChaincodeMessage) *pb.ChaincodeMessage {
		qr := &pb.QueryResponse{}
		proto.Unmarshal(reqMsg.Payload, qr)
		return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Payload: putils.MarshalOrPanic(&pb.QueryStateClose{Id: qr.Id}), Txid: txid, ChannelId: chainID}
	}

	var mkpeer []*mockpeer.MockResponse

	mkpeer = append(mkpeer, &mockpeer.MockResponse{
		RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION},
		RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE_BY_RANGE, Payload: putils.MarshalOrPanic(&pb.GetStateByRange{Collection: collection, StartKey: "A", EndKey: "B"}), Txid: txid, ChannelId: chainID},
	})

	if collection == "" {
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE},
			RespMsg: queryStateNextFunc,
		})
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR},
			RespMsg: queryStateCloseFunc,
		})
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE},
			RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: txid, ChannelId: chainID},
		})
	} else {
//尚未实现对私有数据的范围查询。
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR},
			RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.ERROR, Message: "Not Yet Supported"}), Txid: txid, ChannelId: chainID},
		})
	}

	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: mkpeer,
	}

	cccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	if collection == "" {
		execCC(t, txParams, ccSide, cccid, false, false, done, cis, respSet, chaincodeSupport)
	} else {
		execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)
	}

	endTx(t, txParams, txsim, cis)

	return nil
}

func cc2cc(t *testing.T, chainID, chainID2, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) error {
	calledCC := "calledCC"
//启动并注册CC
	_, calledCCSide := startCC(t, chainID2, calledCC, chaincodeSupport)
	if calledCCSide == nil {
		t.Fatalf("start up failed for called CC")
	}
	defer calledCCSide.Quit()

	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	chaincodeID := &pb.ChaincodeID{Name: calledCC, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("deploycc")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid := util.GenerateUUID()
//首先将新CC部署到LSCC
	txParams, txsim := startTx(t, chainID, cis, txid)

//设置checkacl调用
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

	cccid := &ccprovider.CCContext{
		Name:    calledCC,
		Version: "0",
	}

	deployCC(t, txParams, cccid, cis.ChaincodeSpec, chaincodeSupport)

//犯罪
	endTx(t, txParams, txsim, cis)

//现在做cc2cc
	chaincodeID = &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("invokecc")}, Decorations: nil}
	cis = &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid = util.GenerateUUID()
	txParams, txsim = startTx(t, chainID, cis, txid)

//我们要两个链子的钩子
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

	sysCCVers := util.GetSysCCVersion()
//调用可调用系统CC、常规CC、不同链上常规但不同的CC、不同链上常规但相同的CC和不可调用系统CC，并期望在最后一个系统中出现错误
	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: putils.MarshalOrPanic(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: "lscc:" + sysCCVers}, Input: &pb.ChaincodeInput{Args: [][]byte{{}}}}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: putils.MarshalOrPanic(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: "calledCC:0/" + chainID}, Input: &pb.ChaincodeInput{Args: [][]byte{{}}}}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: putils.MarshalOrPanic(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: "calledCC:0/" + chainID2}, Input: &pb.ChaincodeInput{Args: [][]byte{{}}}}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: putils.MarshalOrPanic(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: "vscc:" + sysCCVers}, Input: &pb.ChaincodeInput{Args: [][]byte{{}}}}), Txid: txid, ChannelId: chainID}},
		},
	}

	respSet2 := &mockpeer.MockResponseSet{
		DoneFunc:  nil,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: txid, ChannelId: chainID}},
		},
	}
	calledCCSide.SetResponses(respSet2)

	cccid = &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}

	execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)

//最后，让我们用CC调用CC来尝试一个坏的ACL。
	chaincodeID = &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("invokecc")}, Decorations: nil}
	cis = &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid = util.GenerateUUID()
	txParams, txsim = startTx(t, chainID, cis, txid)

	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID, txParams.SignedProp).Return(errors.New("Bad ACL calling CC"))
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)
//调用常规CC，但在被调用CC上不使用ACL
	respSet = &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: putils.MarshalOrPanic(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: "calledCC:0/" + chainID}, Input: &pb.ChaincodeInput{Args: [][]byte{{}}}}), Txid: txid, ChannelId: chainID}},
		},
	}

	respSet2 = &mockpeer.MockResponseSet{
		DoneFunc:  nil,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: txid, ChannelId: chainID}},
		},
	}

	calledCCSide.SetResponses(respSet2)

	cccid = &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}

	execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)

	return nil
}

func getQueryResult(t *testing.T, collection, chainID, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) error {
	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	chaincodeID := &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid := util.GenerateUUID()
	txParams, txsim := startTx(t, chainID, cis, txid)

	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

	kvs := make([]*plgr.KV, 1000)
	for i := 0; i < 1000; i++ {
		kvs[i] = &plgr.KV{Namespace: chainID, Key: fmt.Sprintf("%d", i), Value: []byte(fmt.Sprintf("%d", i))}
	}

	queryExec := &mockExecQuerySimulator{resultsIter: make(map[string]map[string]*mockResultsIterator)}
	queryExec.resultsIter[ccname] = map[string]*mockResultsIterator{"goodquery": {kvs: kvs}}

	queryExec.txsim = txParams.TXSimulator

//创建响应
	queryStateNextFunc := func(reqMsg *pb.ChaincodeMessage) *pb.ChaincodeMessage {
		qr := &pb.QueryResponse{}
		proto.Unmarshal(reqMsg.Payload, qr)
		return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Payload: putils.MarshalOrPanic(&pb.QueryStateNext{Id: qr.Id}), Txid: txid, ChannelId: chainID}
	}
	queryStateCloseFunc := func(reqMsg *pb.ChaincodeMessage) *pb.ChaincodeMessage {
		qr := &pb.QueryResponse{}
		proto.Unmarshal(reqMsg.Payload, qr)
		return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Payload: putils.MarshalOrPanic(&pb.QueryStateClose{Id: qr.Id}), Txid: txid, ChannelId: chainID}
	}

	var mkpeer []*mockpeer.MockResponse

	mkpeer = append(mkpeer, &mockpeer.MockResponse{
		RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION},
		RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_QUERY_RESULT, Payload: putils.MarshalOrPanic(&pb.GetQueryResult{Collection: "", Query: "goodquery"}), Txid: txid, ChannelId: chainID},
	})

	if collection == "" {
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE},
			RespMsg: queryStateNextFunc,
		})
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR},
			RespMsg: queryStateCloseFunc,
		})
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE},
			RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: txid, ChannelId: chainID},
		})
	} else {
//尚未实现对私有数据的get查询结果。
		mkpeer = append(mkpeer, &mockpeer.MockResponse{
			RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR},
			RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.ERROR, Message: "Not Yet Supported"}), Txid: txid, ChannelId: chainID},
		})
	}

	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: mkpeer,
	}

	cccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	if collection == "" {
		execCC(t, txParams, ccSide, cccid, false, false, done, cis, respSet, chaincodeSupport)
	} else {
		execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)
	}

	endTx(t, txParams, txsim, cis)

	return nil
}

func getHistory(t *testing.T, chainID, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) error {
	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

	chaincodeID := &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid := util.GenerateUUID()
	txParams, txsim := startTx(t, chainID, cis, txid)

	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

	kvs := make([]*plgr.KV, 1000)
	for i := 0; i < 1000; i++ {
		kvs[i] = &plgr.KV{Namespace: chainID, Key: fmt.Sprintf("%d", i), Value: []byte(fmt.Sprintf("%d", i))}
	}

	queryExec := &mockExecQuerySimulator{resultsIter: make(map[string]map[string]*mockResultsIterator)}
	queryExec.resultsIter[ccname] = map[string]*mockResultsIterator{"goodquery": {kvs: kvs}}

	queryExec.txsim = txParams.TXSimulator

//创建响应
	queryStateNextFunc := func(reqMsg *pb.ChaincodeMessage) *pb.ChaincodeMessage {
		qr := &pb.QueryResponse{}
		proto.Unmarshal(reqMsg.Payload, qr)
		return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_NEXT, Payload: putils.MarshalOrPanic(&pb.QueryStateNext{Id: qr.Id}), Txid: txid}
	}
	queryStateCloseFunc := func(reqMsg *pb.ChaincodeMessage) *pb.ChaincodeMessage {
		qr := &pb.QueryResponse{}
		proto.Unmarshal(reqMsg.Payload, qr)
		return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_QUERY_STATE_CLOSE, Payload: putils.MarshalOrPanic(&pb.QueryStateClose{Id: qr.Id}), Txid: txid}
	}

	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_HISTORY_FOR_KEY, Payload: putils.MarshalOrPanic(&pb.GetQueryResult{Query: "goodquery"}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: queryStateNextFunc},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: queryStateNextFunc},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR}, RespMsg: queryStateCloseFunc},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: putils.MarshalOrPanic(&pb.Response{Status: shim.OK, Payload: []byte("OK")}), Txid: txid, ChannelId: chainID}},
		},
	}

	cccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}
	execCC(t, txParams, ccSide, cccid, false, false, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)

	return nil
}

func getLaunchConfigs(t *testing.T, cr *ContainerRuntime) {
	gt := NewGomegaWithT(t)
	ccContext := &ccprovider.CCContext{
		Name:    "mycc",
		Version: "v0",
	}
	lc, err := cr.LaunchConfig(ccContext.GetCanonicalName(), pb.ChaincodeSpec_GOLANG.String())
	if err != nil {
		t.Fatalf("calling getLaunchConfigs() failed with error %s", err)
	}
	args := lc.Args
	envs := lc.Envs
	filesToUpload := lc.Files

	if len(args) != 2 {
		t.Fatalf("calling getLaunchConfigs() for golang chaincode should have returned an array of 2 elements for Args, but got %v", args)
	}
	if args[0] != "chaincode" || !strings.HasPrefix(args[1], "-peer.address") {
		t.Fatalf("calling getLaunchConfigs() should have returned the start command for golang chaincode, but got %v", args)
	}

	if len(envs) != 8 {
		t.Fatalf("calling getLaunchConfigs() with TLS enabled should have returned an array of 8 elements for Envs, but got %v", envs)
	}
	gt.Expect(envs).To(ContainElement("CORE_CHAINCODE_LOGGING_LEVEL=info"))
	gt.Expect(envs).To(ContainElement("CORE_CHAINCODE_LOGGING_SHIM=warning"))
	gt.Expect(envs).To(ContainElement("CORE_CHAINCODE_ID_NAME=mycc:v0"))
	gt.Expect(envs).To(ContainElement("CORE_PEER_TLS_ENABLED=true"))
	gt.Expect(envs).To(ContainElement("CORE_TLS_CLIENT_KEY_PATH=/etc/hyperledger/fabric/client.key"))
	gt.Expect(envs).To(ContainElement("CORE_TLS_CLIENT_CERT_PATH=/etc/hyperledger/fabric/client.crt"))
	gt.Expect(envs).To(ContainElement("CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/peer.crt"))

	if len(filesToUpload) != 3 {
		t.Fatalf("calling getLaunchConfigs() with TLS enabled should have returned an array of 3 elements for filesToUpload, but got %v", len(filesToUpload))
	}

cr.CertGenerator = nil //禁用TLS
	lc, err = cr.LaunchConfig(ccContext.GetCanonicalName(), pb.ChaincodeSpec_NODE.String())
	assert.NoError(t, err)
	args = lc.Args

	if len(args) != 3 {
		t.Fatalf("calling getLaunchConfigs() for node chaincode should have returned an array of 3 elements for Args, but got %v", args)
	}

	if args[0] != "/bin/sh" || args[1] != "-c" || !strings.HasPrefix(args[2], "cd /usr/local/src; npm start -- --peer.address") {
		t.Fatalf("calling getLaunchConfigs() should have returned the start command for node.js chaincode, but got %v", args)
	}

	lc, err = cr.LaunchConfig(ccContext.GetCanonicalName(), pb.ChaincodeSpec_GOLANG.String())
	assert.NoError(t, err)

	envs = lc.Envs
	if len(envs) != 5 {
		t.Fatalf("calling getLaunchConfigs() with TLS disabled should have returned an array of 4 elements for Envs, but got %v", envs)
	}
	gt.Expect(envs).To(ContainElement("CORE_CHAINCODE_LOGGING_LEVEL=info"))
	gt.Expect(envs).To(ContainElement("CORE_CHAINCODE_LOGGING_SHIM=warning"))
	gt.Expect(envs).To(ContainElement("CORE_CHAINCODE_ID_NAME=mycc:v0"))
	gt.Expect(envs).To(ContainElement("CORE_PEER_TLS_ENABLED=false"))
}

//成功案例
func TestStartAndWaitSuccess(t *testing.T) {
	handlerRegistry := NewHandlerRegistry(false)
	fakeRuntime := &mock.Runtime{}
	fakeRuntime.StartStub = func(_ *ccprovider.ChaincodeContainerInfo, _ []byte) error {
		handlerRegistry.Ready("testcc:0")
		return nil
	}

	code := getTarGZ(t, "src/dummy/dummy.go", []byte("code"))
	fakePackageProvider := &mock.PackageProvider{}
	fakePackageProvider.GetChaincodeCodePackageReturns(code, nil)

	launcher := &RuntimeLauncher{
		Runtime:         fakeRuntime,
		Registry:        handlerRegistry,
		StartupTimeout:  10 * time.Second,
		PackageProvider: fakePackageProvider,
		Metrics:         NewLaunchMetrics(&disabled.Provider{}),
	}

	ccci := &ccprovider.ChaincodeContainerInfo{
		Type:          "GOLANG",
		Name:          "testcc",
		Version:       "0",
		ContainerType: "DOCKER",
	}

//实际测试-一切正常
	err := launcher.Launch(ccci)
	if err != nil {
		t.Fatalf("expected success but failed with error %s", err)
	}
}

//测试超时错误
func TestStartAndWaitTimeout(t *testing.T) {
	fakeRuntime := &mock.Runtime{}
	fakeRuntime.StartStub = func(_ *ccprovider.ChaincodeContainerInfo, _ []byte) error {
		time.Sleep(time.Second)
		return nil
	}

	code := getTarGZ(t, "src/dummy/dummy.go", []byte("code"))
	fakePackageProvider := &mock.PackageProvider{}
	fakePackageProvider.GetChaincodeCodePackageReturns(code, nil)

	launcher := &RuntimeLauncher{
		Runtime:         fakeRuntime,
		Registry:        NewHandlerRegistry(false),
		StartupTimeout:  500 * time.Millisecond,
		PackageProvider: fakePackageProvider,
		Metrics:         NewLaunchMetrics(&disabled.Provider{}),
	}

	ccci := &ccprovider.ChaincodeContainerInfo{
		Type:          "GOLANG",
		Name:          "testcc",
		Version:       "0",
		ContainerType: "DOCKER",
	}

//实际测试-超时1000>500
	err := launcher.Launch(ccci)
	if err == nil {
		t.Fatalf("expected error but succeeded")
	}
}

//测试容器返回错误
func TestStartAndWaitLaunchError(t *testing.T) {
	fakeRuntime := &mock.Runtime{}
	fakeRuntime.StartStub = func(_ *ccprovider.ChaincodeContainerInfo, _ []byte) error {
		return errors.New("Bad lunch; upset stomach")
	}

	code := getTarGZ(t, "src/dummy/dummy.go", []byte("code"))
	fakePackageProvider := &mock.PackageProvider{}
	fakePackageProvider.GetChaincodeCodePackageReturns(code, nil)

	launcher := &RuntimeLauncher{
		Runtime:         fakeRuntime,
		Registry:        NewHandlerRegistry(false),
		StartupTimeout:  10 * time.Second,
		PackageProvider: fakePackageProvider,
		Metrics:         NewLaunchMetrics(&disabled.Provider{}),
	}

	ccci := &ccprovider.ChaincodeContainerInfo{
		Type:          "GOLANG",
		Name:          "testcc",
		Version:       "0",
		ContainerType: "DOCKER",
	}

//实际测试-容器启动出错
	err := launcher.Launch(ccci)
	if err == nil {
		t.Fatalf("expected error but succeeded")
	}
	assert.EqualError(t, err, "error starting container: Bad lunch; upset stomach")
}

func TestGetTxContextFromHandler(t *testing.T) {
	h := Handler{TXContexts: NewTransactionContexts(), SystemCCProvider: &scc.Provider{Peer: peer.Default, PeerSupport: peer.DefaultSupport, Registrar: inproccontroller.NewRegistry()}}

	chnl := "test"
	txid := "1"
//测试通道的test gettxContext，tx=1，msgtype=ivnoke_chaincode和empty payload-empty payload=>预期返回空txContext
	txContext, _ := h.getTxContextForInvoke(chnl, "1", []byte(""), "[%s]No ledger context for %s. Sending %s", 12345, "TestCC", pb.ChaincodeMessage_ERROR)
	if txContext != nil {
		t.Fatalf("expected empty txContext for empty payload")
	}

//为我们的频道模仿同龄人
	peer.MockInitialize()

	err := peer.MockCreateChain(chnl)
	if err != nil {
		t.Fatalf("failed to create Peer Ledger %s", err)
	}

	pldgr := peer.GetLedger(chnl)

//准备一个有效负载并在处理程序中生成一个txContext，该处理程序将在下面的getXContextFromMessage中使用一个普通的UCC
	txCtxGenerated, payload := genNewPldAndCtxFromLdgr(t, "shimTestCC", chnl, txid, pldgr, &h)

//测试通道的test gettxContext，tx=1，msgtype=ivnoke_chaincode and non empty payload=>必须返回非空的txContext
	txContext, ccMsg := h.getTxContextForInvoke(chnl, txid, payload, "[%s]No ledger context for %s. Sending %s", 12345, pb.ChaincodeMessage_INVOKE_CHAINCODE, pb.ChaincodeMessage_ERROR)
	if txContext == nil || ccMsg != nil || txContext != txCtxGenerated {
		t.Fatalf("expected successful txContext for non empty payload and INVOKE_CHAINCODE msgType. triggerNextStateMsg: %s.", ccMsg)
	}

//测试具有相同负载的另一个msgtype（put_state）==>必须返回非空txContext
	txContext, ccMsg = h.getTxContextForInvoke(chnl, txid, payload, "[%s]No ledger context for %s. Sending %s", 12345, pb.ChaincodeMessage_PUT_STATE, pb.ChaincodeMessage_ERROR)
	if txContext == nil || ccMsg != nil || txContext != txCtxGenerated {
		t.Fatalf("expected successful txContext for non empty payload and PUT_STATE msgType. triggerNextStateMsg: %s.", ccMsg)
	}

//为我们的SCC测试获取新的TxContext
	txid = "2"
//将通道重置为“”以测试获取没有通道的SCC的上下文
	chnl = ""
	txCtxGenerated, payload = genNewPldAndCtxFromLdgr(t, "lscc", chnl, txid, pldgr, &h)

//测试获取不带通道的SCC的TxContext=>应返回非空的TxContext
	txContext, ccMsg = h.getTxContextForInvoke(chnl, txid, payload,
		"[%s]No ledger context for %s. Sending %s", 12345, pb.ChaincodeMessage_INVOKE_CHAINCODE, pb.ChaincodeMessage_ERROR)
	if txContext == nil || ccMsg != nil || txContext != txCtxGenerated {
		t.Fatalf("expected successful txContext for non empty payload and INVOKE_CHAINCODE msgType. triggerNextStateMsg: %s.", ccMsg)
	}

//现在重置为非空通道，并使用SCC进行测试
	txid = "3"
	chnl = "TEST"
	txCtxGenerated, payload = genNewPldAndCtxFromLdgr(t, "lscc", chnl, txid, pldgr, &h)

//测试获取带有scc的txContext和channel test=>预期返回非空的txContext
	txContext, ccMsg = h.getTxContextForInvoke(chnl, txid, payload,
		"[%s]No ledger context for %s. Sending %s", 12345, pb.ChaincodeMessage_INVOKE_CHAINCODE, pb.ChaincodeMessage_ERROR)
	if txContext == nil || ccMsg != nil || txContext != txCtxGenerated {
		t.Fatalf("expected successful txContext for non empty payload and INVOKE_CHAINCODE msgType. triggerNextStateMsg: %s.", ccMsg)
	}

//现在测试获取一个带有空通道和UCC而不是SCC的上下文
	txid = "4"
	chnl = ""
	txCtxGenerated, payload = genNewPldAndCtxFromLdgr(t, "shimTestCC", chnl, txid, pldgr, &h)
//测试获取带有scc的txContext和channel test=>预期返回非空的txContext
	txContext, ccMsg = h.getTxContextForInvoke(chnl, txid, payload,
		"[%s]No ledger context for %s. Sending %s", 12345, pb.ChaincodeMessage_INVOKE_CHAINCODE, pb.ChaincodeMessage_ERROR)
	if txContext == nil || ccMsg != nil || txContext != txCtxGenerated {
		t.Fatalf("expected successful txContext for non empty payload and INVOKE_CHAINCODE msgType. triggerNextStateMsg: %s.", ccMsg)
	}

//新测试获取带有空通道的上下文，而不使用分类帐为UCC创建新上下文
	txid = "5"
	payload = genNewPld(t, "shimTestCC")
//测试获取带有scc的txContext和channel test=>预期返回非空的txContext
	txContext, ccMsg = h.getTxContextForInvoke(chnl, txid, payload,
		"[%s]No ledger context for %s. Sending %s", 12345, pb.ChaincodeMessage_INVOKE_CHAINCODE, pb.ChaincodeMessage_ERROR)
	if txContext != nil || ccMsg == nil {
		t.Fatal("expected nil txContext for non empty payload and INVOKE_CHAINCODE msgType without the ledger generating a TxContext . unexpected non nil tcContext")
	}

//测试与上述相同的场景，但这次测试SCC
	txid = "6"
	payload = genNewPld(t, "lscc")
//测试获取带有scc的txContext和channel test=>预期返回非空的txContext
	txContext, ccMsg = h.getTxContextForInvoke(chnl, txid, payload,
		"[%s]No ledger context for %s. Sending %s", 12345, pb.ChaincodeMessage_INVOKE_CHAINCODE, pb.ChaincodeMessage_ERROR)
	if txContext != nil || ccMsg == nil {
		t.Fatal("expected nil txContext for non empty payload and INVOKE_CHAINCODE msgType without the ledger generating a TxContext . unexpected non nil tcContext")
	}
}

func genNewPldAndCtxFromLdgr(t *testing.T, ccName string, chnl string, txid string, pldgr ledger.PeerLedger, h *Handler) (*TransactionContext, []byte) {
//为接收到的TxID创建新的TxSimulator
	txsim, err := pldgr.NewTxSimulator(txid)
	if err != nil {
		t.Fatalf("failed to create TxSimulator %s", err)
	}
	txParams := &ccprovider.TransactionParams{
		TxID:        txid,
		ChannelID:   chnl,
		TXSimulator: txsim,
	}
	newTxCtxt, err := h.TXContexts.Create(txParams)
	if err != nil {
		t.Fatalf("Error creating TxContext by the handler for cc %s and channel '%s': %s", ccName, chnl, err)
	}
	if newTxCtxt == nil {
		t.Fatalf("Error creating TxContext: newTxCtxt created by the handler is nil for cc %s and channel '%s'.", ccName, chnl)
	}
//为CC构建一个名为ccname的新CD和有效负载
	payload := genNewPld(t, ccName)
	return newTxCtxt, payload
}

func genNewPld(t *testing.T, ccName string) []byte {
//为CC构建一个名为ccname的新CD和有效负载
	chaincodeID := &pb.ChaincodeID{Name: ccName, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("deploycc")}, Decorations: nil}
	cds := &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}
	payload, err := proto.Marshal(cds)
	if err != nil {
		t.Fatalf("failed to marshal CDS %s", err)
	}
	return payload
}

func cc2SameCC(t *testing.T, chainID, chainID2, ccname string, ccSide *mockpeer.MockCCComm, chaincodeSupport *ChaincodeSupport) {
//首先在chaineid2上部署CC
	chaincodeID := &pb.ChaincodeID{Name: ccname, Version: "0"}
	ci := &pb.ChaincodeInput{Args: [][]byte{[]byte("deploycc")}, Decorations: nil}
	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}

	txid := util.GenerateUUID()
	txParams, txsim := startTx(t, chainID2, cis, txid)

//设置checkacl调用
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID2, txParams.SignedProp).Return(nil)

	cccid := &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}

	deployCC(t, txParams, cccid, cis.ChaincodeSpec, chaincodeSupport)

//犯罪
	endTx(t, txParams, txsim, cis)

	done := setuperror()

	errorFunc := func(ind int, err error) {
		done <- err
	}

//现在测试-在不同的通道上调用相同的CC（应该成功），在相同的通道上调用相同的CC（应该失败）
//注意错误“此txid的另一个请求挂起。无法处理。“在tx”cctosamecctx下的日志中”
	ci = &pb.ChaincodeInput{Args: [][]byte{[]byte("invoke"), []byte("A"), []byte("B"), []byte("10")}, Decorations: nil}
	cis = &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Type: pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]), ChaincodeId: chaincodeID, Input: ci}}
	txid = util.GenerateUUID()
	txParams, txsim = startTx(t, chainID, cis, txid)

	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID, txParams.SignedProp).Return(nil)

	mockAclProvider.On("CheckACL", resources.Peer_ChaincodeToChaincode, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetDeploymentSpec, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Lscc_GetChaincodeData, chainID2, txParams.SignedProp).Return(nil)
	mockAclProvider.On("CheckACL", resources.Peer_Propose, chainID2, txParams.SignedProp).Return(nil)

	txid = "cctosamecctx"
	respSet := &mockpeer.MockResponseSet{
		DoneFunc:  errorFunc,
		ErrorFunc: nil,
		Responses: []*mockpeer.MockResponse{
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_TRANSACTION}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: putils.MarshalOrPanic(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: ccname + ":0/" + chainID2}, Input: &pb.ChaincodeInput{Args: [][]byte{{}}}}), Txid: txid, ChannelId: chainID}},
			{RecvMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_RESPONSE}, RespMsg: &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: putils.MarshalOrPanic(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: ccname + ":0/" + chainID}, Input: &pb.ChaincodeInput{Args: [][]byte{{}}}}), Txid: txid, ChannelId: chainID}},
		},
	}

	cccid = &ccprovider.CCContext{
		Name:    ccname,
		Version: "0",
	}

	execCC(t, txParams, ccSide, cccid, false, true, done, cis, respSet, chaincodeSupport)

	endTx(t, txParams, txsim, cis)
}

func TestCCFramework(t *testing.T) {
//注册2个频道
	chainID := "mockchainid"
	chainID2 := "secondchain"
	chaincodeSupport, err := initMockPeer(chainID, chainID2)
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer finitMockPeer(chainID, chainID2)
//创建链代码
	ccname := "shimTestCC"

//启动并注册CC
	_, ccSide := startCC(t, chainID, ccname, chaincodeSupport)
	if ccSide == nil {
		t.Fatalf("start up failed")
	}
	defer ccSide.Quit()

//调用的init并执行一些Put（在执行一些负测试之后）
	initializeCC(t, chainID, ccname, ccSide, chaincodeSupport)

//链码支持不允许DUP
	handler := &Handler{chaincodeID: &pb.ChaincodeID{Name: ccname + ":0"}, SystemCCProvider: chaincodeSupport.SystemCCProvider}
	if err := chaincodeSupport.HandlerRegistry.Register(handler); err == nil {
		t.Fatalf("expected re-register to fail")
	}

//调用的init并执行一些Put（在执行一些负测试之后）
	initializeCC(t, chainID2, ccname, ccSide, chaincodeSupport)

//调用的调用并执行一些get
	invokeCC(t, chainID, ccname, ccSide, chaincodeSupport)

//以下私有数据调用被禁用，因为
//这需要上的专用数据通道功能，因此应该存在
//在专用测试中。文件中存在这样一个测试-executeTransaction_pvtdata_test.go
//调用并对私有数据执行一些get/put/del操作
//invokeprivatedatagetputdelcc（t，chainID，ccname，ccside）

//调用的查询状态范围
	getQueryStateByRange(t, "", chainID, ccname, ccSide, chaincodeSupport)

//对同一个chaincode调用的cc2cc只应成功调用chainid2
	cc2SameCC(t, chainID, chainID2, ccname, ccSide, chaincodeSupport)

//调用的cc2cc（与syscc调用的变体）
	cc2cc(t, chainID, chainID2, ccname, ccSide, chaincodeSupport)

//调用的查询结果
	getQueryResult(t, "", chainID, ccname, ccSide, chaincodeSupport)

//呼叫历史记录结果
	getHistory(t, chainID, ccname, ccSide, chaincodeSupport)

//只需使用以前的证书生成器生成TLS密钥/对
	cr := chaincodeSupport.Runtime.(*ContainerRuntime)
	getLaunchConfigs(t, cr)

	ccSide.Quit()
}
