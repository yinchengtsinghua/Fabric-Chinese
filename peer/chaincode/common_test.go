
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有Digital Asset Holdings，LLC。保留所有权利。
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package chaincode

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/tools/configtxgen/configtxgentest"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/peer/chaincode/mock"
	"github.com/hyperledger/fabric/peer/common"
	"github.com/hyperledger/fabric/peer/common/api"
	cmock "github.com/hyperledger/fabric/peer/common/mock"
	common2 "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckChaincodeCmdParamsWithNewCallingSchema(t *testing.T) {
	chaincodeCtorJSON = `{ "Args":["func", "param"] }`
	chaincodePath = "some/path"
	chaincodeName = "somename"
	require := require.New(t)
	result := checkChaincodeCmdParams(&cobra.Command{})

	require.Nil(result)
}

func TestCheckChaincodeCmdParamsWithOldCallingSchema(t *testing.T) {
	chaincodeCtorJSON = `{ "Function":"func", "Args":["param"] }`
	chaincodePath = "some/path"
	chaincodeName = "somename"
	require := require.New(t)
	result := checkChaincodeCmdParams(&cobra.Command{})

	require.Nil(result)
}

func TestCheckChaincodeCmdParamsWithoutName(t *testing.T) {
	chaincodeCtorJSON = `{ "Function":"func", "Args":["param"] }`
	chaincodePath = "some/path"
	chaincodeName = ""
	require := require.New(t)
	result := checkChaincodeCmdParams(&cobra.Command{})

	require.Error(result)
}

func TestCheckChaincodeCmdParamsWithFunctionOnly(t *testing.T) {
	chaincodeCtorJSON = `{ "Function":"func" }`
	chaincodePath = "some/path"
	chaincodeName = "somename"
	require := require.New(t)
	result := checkChaincodeCmdParams(&cobra.Command{})

	require.Error(result)
}

func TestCheckChaincodeCmdParamsEmptyCtor(t *testing.T) {
	chaincodeCtorJSON = `{}`
	chaincodePath = "some/path"
	chaincodeName = "somename"
	require := require.New(t)
	result := checkChaincodeCmdParams(&cobra.Command{})

	require.Error(result)
}

func TestCheckValidJSON(t *testing.T) {
	validJSON := `{"Args":["a","b","c"]}`
	input := &pb.ChaincodeInput{}
	if err := json.Unmarshal([]byte(validJSON), &input); err != nil {
		t.Fail()
		t.Logf("Chaincode argument error: %s", err)
		return
	}

	validJSON = `{"Function":"f", "Args":["a","b","c"]}`
	if err := json.Unmarshal([]byte(validJSON), &input); err != nil {
		t.Fail()
		t.Logf("Chaincode argument error: %s", err)
		return
	}

	validJSON = `{"Function":"f", "Args":[]}`
	if err := json.Unmarshal([]byte(validJSON), &input); err != nil {
		t.Fail()
		t.Logf("Chaincode argument error: %s", err)
		return
	}

	validJSON = `{"Function":"f"}`
	if err := json.Unmarshal([]byte(validJSON), &input); err != nil {
		t.Fail()
		t.Logf("Chaincode argument error: %s", err)
		return
	}
}

func TestCheckInvalidJSON(t *testing.T) {
	invalidJSON := `{["a","b","c"]}`
	input := &pb.ChaincodeInput{}
	if err := json.Unmarshal([]byte(invalidJSON), &input); err == nil {
		t.Fail()
		t.Logf("Bar argument error should have been caught: %s", invalidJSON)
		return
	}

	invalidJSON = `{"Function":}`
	if err := json.Unmarshal([]byte(invalidJSON), &input); err == nil {
		t.Fail()
		t.Logf("Chaincode argument error: %s", err)
		t.Logf("Bar argument error should have been caught: %s", invalidJSON)
		return
	}
}

func TestGetOrdererEndpointFromConfigTx(t *testing.T) {
	signer, err := common.GetDefaultSigner()
	assert.NoError(t, err)

	mockchain := "mockchain"
	factory.InitFactories(nil)
	config := configtxgentest.Load(genesisconfig.SampleInsecureSoloProfile)
	pgen := encoder.New(config)
	genesisBlock := pgen.GenesisBlockForChannel(mockchain)

	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(genesisBlock)},
		Endorsement: &pb.Endorsement{},
	}
	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)

	ordererEndpoints, err := common.GetOrdererEndpointOfChain(mockchain, signer, mockEndorserClient)
	assert.NoError(t, err, "GetOrdererEndpointOfChain from genesis block")

	assert.Equal(t, len(ordererEndpoints), 1)
	assert.Equal(t, ordererEndpoints[0], "127.0.0.1:7050")
}

func TestGetOrdererEndpointFail(t *testing.T) {
	signer, err := common.GetDefaultSigner()
	assert.NoError(t, err)

	mockchain := "mockchain"
	factory.InitFactories(nil)

	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 404, Payload: []byte{}},
		Endorsement: &pb.Endorsement{},
	}
	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)

	_, err = common.GetOrdererEndpointOfChain(mockchain, signer, mockEndorserClient)
	assert.Error(t, err, "GetOrdererEndpointOfChain from invalid response")
}

const sampleCollectionConfigGood = `[
	{
		"name": "foo",
		"policy": "OR('A.member', 'B.member')",
		"requiredPeerCount": 3,
		"maxPeerCount": 483279847,
		"blockToLive":10,
		"memberOnlyRead": true
	}
]`

const sampleCollectionConfigBad = `[
	{
		"name": "foo",
		"policy": "barf",
		"requiredPeerCount": 3,
		"maxPeerCount": 483279847
	}
]`

func TestCollectionParsing(t *testing.T) {
	cc, err := getCollectionConfigFromBytes([]byte(sampleCollectionConfigGood))
	assert.NoError(t, err)
	assert.NotNil(t, cc)
	ccp := &common2.CollectionConfigPackage{}
	proto.Unmarshal(cc, ccp)
	conf := ccp.Config[0].GetStaticCollectionConfig()
	pol, _ := cauthdsl.FromString("OR('A.member', 'B.member')")
	assert.Equal(t, 3, int(conf.RequiredPeerCount))
	assert.Equal(t, 483279847, int(conf.MaximumPeerCount))
	assert.Equal(t, "foo", conf.Name)
	assert.Equal(t, pol, conf.MemberOrgsPolicy.GetSignaturePolicy())
	assert.Equal(t, 10, int(conf.BlockToLive))
	assert.Equal(t, true, conf.MemberOnlyRead)
	t.Logf("conf=%s", conf)

	cc, err = getCollectionConfigFromBytes([]byte(sampleCollectionConfigBad))
	assert.Error(t, err)
	assert.Nil(t, cc)

	cc, err = getCollectionConfigFromBytes([]byte("barf"))
	assert.Error(t, err)
	assert.Nil(t, cc)
}

func TestValidatePeerConnectionParams(t *testing.T) {
	defer resetFlags()
	defer viper.Reset()
	assert := assert.New(t)
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

//禁用TLS
	viper.Set("peer.tls.enabled", false)

//失败-多个对等端和TLS根证书-未调用
	resetFlags()
	peerAddresses = []string{"peer0", "peer1"}
	tlsRootCertFiles = []string{"cert0", "cert1"}
	err := validatePeerConnectionParameters("query")
	assert.Error(err)
	assert.Contains(err.Error(), "command can only be executed against one peer")

//成功-提供了对等端，但没有TLS根证书
//禁用TLS
	resetFlags()
	peerAddresses = []string{"peer0"}
	err = validatePeerConnectionParameters("query")
	assert.NoError(err)
	assert.Nil(tlsRootCertFiles)

//成功-比对等端更多的TLS根证书
//禁用TLS
	resetFlags()
	peerAddresses = []string{"peer0"}
	tlsRootCertFiles = []string{"cert0", "cert1"}
	err = validatePeerConnectionParameters("invoke")
	assert.NoError(err)
	assert.Nil(tlsRootCertFiles)

//成功-多个对等机，没有TLS根证书-调用
//禁用TLS
	resetFlags()
	peerAddresses = []string{"peer0", "peer1"}
	err = validatePeerConnectionParameters("invoke")
	assert.NoError(err)
	assert.Nil(tlsRootCertFiles)

//启用TLS
	viper.Set("peer.tls.enabled", true)

//失败-对等端和TLS根证书的数量不均匀-调用
//启用TLS
	resetFlags()
	peerAddresses = []string{"peer0", "peer1"}
	tlsRootCertFiles = []string{"cert0"}
	err = validatePeerConnectionParameters("invoke")
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("number of peer addresses (%d) does not match the number of TLS root cert files (%d)", len(peerAddresses), len(tlsRootCertFiles)))

//成功-多个对等端和TLS根证书-调用
//启用TLS
	resetFlags()
	peerAddresses = []string{"peer0", "peer1"}
	tlsRootCertFiles = []string{"cert0", "cert1"}
	err = validatePeerConnectionParameters("invoke")
	assert.NoError(err)

//失败-连接配置文件不存在
	resetFlags()
	connectionProfile = "blah"
	err = validatePeerConnectionParameters("invoke")
	assert.Error(err)
	assert.Contains(err.Error(), "error reading connection profile")

//失败-连接配置文件在通道配置中定义了对等，但
//不在对等配置中
	resetFlags()
	channelID = "mychannel"
	connectionProfile = "../common/testdata/connectionprofile-uneven.yaml"
	err = validatePeerConnectionParameters("invoke")
	assert.Error(err)
	assert.Contains(err.Error(), "defined in the channel config but doesn't have associated peer config")

//成功-连接配置文件存在
	resetFlags()
	channelID = "mychannel"
	connectionProfile = "../common/testdata/connectionprofile.yaml"
	err = validatePeerConnectionParameters("invoke")
	assert.NoError(err)
}

func TestInitCmdFactoryFailures(t *testing.T) {
	defer resetFlags()
	assert := assert.New(t)

//验证对等连接参数失败
	resetFlags()
	peerAddresses = []string{"peer0", "peer1"}
	tlsRootCertFiles = []string{"cert0", "cert1"}
	cf, err := InitCmdFactory("query", true, false)
	assert.Error(err)
	assert.Contains(err.Error(), "error validating peer connection parameters: 'query' command can only be executed against one peer")
	assert.Nil(cf)

//失败-没有提供对等机，需要背书客户端
	resetFlags()
	peerAddresses = []string{}
	cf, err = InitCmdFactory("query", true, false)
	assert.Error(err)
	assert.Contains(err.Error(), "no endorser clients retrieved")
	assert.Nil(cf)

//失败-需要订购方客户端，订购端点为空，没有
//背书人客户提供
	resetFlags()
	peerAddresses = nil
	cf, err = InitCmdFactory("invoke", false, true)
	assert.Error(err)
	assert.Contains(err.Error(), "no ordering endpoint or endorser client supplied")
	assert.Nil(cf)
}

func TestDeliverGroupConnect(t *testing.T) {
	defer resetFlags()
	g := NewGomegaWithT(t)

//成功
	mockDeliverClients := []*deliverClient{
		{
			Client:  getMockDeliverClientResponseWithTxID("txid0"),
			Address: "peer0",
		},
		{
			Client:  getMockDeliverClientResponseWithTxID("txid0"),
			Address: "peer1",
		},
	}
	dg := deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err := dg.Connect(context.Background())
	g.Expect(err).To(BeNil())

//失败-DeliverFiltered返回错误
	mockDC := &cmock.PeerDeliverClient{}
	mockDC.DeliverFilteredReturns(nil, errors.New("icecream"))
	mockDeliverClients = []*deliverClient{
		{
			Client:  mockDC,
			Address: "peer0",
		},
	}
	dg = deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err = dg.Connect(context.Background())
	g.Expect(err.Error()).To(ContainSubstring("error connecting to deliver filtered"))
	g.Expect(err.Error()).To(ContainSubstring("icecream"))

//失败-发送返回错误
	mockD := &mock.Deliver{}
	mockD.SendReturns(errors.New("blah"))
	mockDC.DeliverFilteredReturns(mockD, nil)
	mockDeliverClients = []*deliverClient{
		{
			Client:  mockDC,
			Address: "peer0",
		},
	}
	dg = deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err = dg.Connect(context.Background())
	g.Expect(err.Error()).To(ContainSubstring("error sending deliver seek info"))
	g.Expect(err.Error()).To(ContainSubstring("blah"))

//失败-传递注册超时
	delayChan := make(chan struct{})
	mockDCDelay := getMockDeliverClientRegisterAfterDelay(delayChan)
	mockDeliverClients = []*deliverClient{
		{
			Client:  mockDCDelay,
			Address: "peer0",
		},
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelFunc()
	dg = deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err = dg.Connect(ctx)
	g.Expect(err.Error()).To(ContainSubstring("timed out waiting for connection to deliver on all peers"))
	close(delayChan)
}

func TestDeliverGroupWait(t *testing.T) {
	defer resetFlags()
	g := NewGomegaWithT(t)

//成功
	mockConn := getMockDeliverConnectionResponseWithTxID("txid0")
	mockDeliverClients := []*deliverClient{
		{
			Connection: mockConn,
			Address:    "peer0",
		},
	}
	dg := deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err := dg.Wait(context.Background())
	g.Expect(err).To(BeNil())

//失败-接收返回错误
	mockConn = &mock.Deliver{}
	mockConn.RecvReturns(nil, errors.New("avocado"))
	mockDeliverClients = []*deliverClient{
		{
			Connection: mockConn,
			Address:    "peer0",
		},
	}
	dg = deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err = dg.Wait(context.Background())
	g.Expect(err.Error()).To(ContainSubstring("error receiving from deliver filtered"))
	g.Expect(err.Error()).To(ContainSubstring("avocado"))

//失败-recv返回意外类型
	mockConn = &mock.Deliver{}
	resp := &pb.DeliverResponse{
		Type: &pb.DeliverResponse_Block{},
	}
	mockConn.RecvReturns(resp, nil)
	mockDeliverClients = []*deliverClient{
		{
			Connection: mockConn,
			Address:    "peer0",
		},
	}
	dg = deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err = dg.Wait(context.Background())
	g.Expect(err.Error()).To(ContainSubstring("unexpected response type"))

//失败-两个连接都返回错误
	mockConn = &mock.Deliver{}
	mockConn.RecvReturns(nil, errors.New("barbeque"))
	mockConn2 := &mock.Deliver{}
	mockConn2.RecvReturns(nil, errors.New("tofu"))
	mockDeliverClients = []*deliverClient{
		{
			Connection: mockConn,
			Address:    "peerBBQ",
		},
		{
			Connection: mockConn2,
			Address:    "peerTOFU",
		},
	}
	dg = deliverGroup{
		Clients:   mockDeliverClients,
		ChannelID: "testchannel",
		Certificate: tls.Certificate{
			Certificate: [][]byte{[]byte("test")},
		},
		TxID: "txid0",
	}
	err = dg.Wait(context.Background())
	g.Expect(err.Error()).To(SatisfyAny(
		ContainSubstring("barbeque"),
		ContainSubstring("tofu")))
}

func TestChaincodeInvokeOrQuery_waitForEvent(t *testing.T) {
	defer resetFlags()

//成功-传递带有预期txid的客户端返回事件
	waitForEvent = true
	mockCF, err := getMockChaincodeCmdFactory()
	assert.NoError(t, err)
	peerAddresses = []string{"peer0", "peer1"}
	channelID := "testchannel"
	txID := "txid0"

	_, err = ChaincodeInvokeOrQuery(
		&pb.ChaincodeSpec{},
		channelID,
		txID,
		true,
		mockCF.Signer,
		mockCF.Certificate,
		mockCF.EndorserClients,
		mockCF.DeliverClients,
		mockCF.BroadcastClient,
	)
	assert.NoError(t, err)

//成功-一个交付客户端首先接收不带txid的块，然后
//然后是一个有txid的
	filteredBlocks := []*pb.FilteredBlock{
		createFilteredBlock("theseare", "notthetxidsyouarelookingfor"),
		createFilteredBlock("txid0"),
	}
	mockDCTwoBlocks := getMockDeliverClientRespondsWithFilteredBlocks(filteredBlocks)
	mockDC := getMockDeliverClientResponseWithTxID("txid0")
	mockDeliverClients := []api.PeerDeliverClient{mockDCTwoBlocks, mockDC}

	_, err = ChaincodeInvokeOrQuery(
		&pb.ChaincodeSpec{},
		channelID,
		txID,
		true,
		mockCF.Signer,
		mockCF.Certificate,
		mockCF.EndorserClients,
		mockDeliverClients,
		mockCF.BroadcastClient,
	)
	assert.NoError(t, err)

//失败-其中一个传递客户端返回错误
	mockDCErr := getMockDeliverClientWithErr("moist")
	mockDC = getMockDeliverClient()
	mockDeliverClients = []api.PeerDeliverClient{mockDCErr, mockDC}

	_, err = ChaincodeInvokeOrQuery(
		&pb.ChaincodeSpec{},
		channelID,
		txID,
		true,
		mockCF.Signer,
		mockCF.Certificate,
		mockCF.EndorserClients,
		mockDeliverClients,
		mockCF.BroadcastClient,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "moist")

//失败-发生超时-两个传递客户端都不返回事件
//具有预期的txid
	mockDC = getMockDeliverClientResponseWithTxID("garbage")
	delayChan := make(chan struct{})
	mockDCDelay := getMockDeliverClientRespondAfterDelay(delayChan)
	mockDeliverClients = []api.PeerDeliverClient{mockDC, mockDCDelay}
	waitForEventTimeout = 10 * time.Millisecond

	_, err = ChaincodeInvokeOrQuery(
		&pb.ChaincodeSpec{},
		channelID,
		txID,
		true,
		mockCF.Signer,
		mockCF.Certificate,
		mockCF.EndorserClients,
		mockDeliverClients,
		mockCF.BroadcastClient,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	close(delayChan)
}
