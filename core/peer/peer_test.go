
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
	"math/rand"
	"net"
	"testing"

	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	mscc "github.com/hyperledger/fabric/common/mocks/scc"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/deliverservice"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	ledgermocks "github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/core/mocks/ccprovider"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	peergossip "github.com/hyperledger/fabric/peer/gossip"
	"github.com/hyperledger/fabric/peer/gossip/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockDeliveryClient struct {
}

func (ds *mockDeliveryClient) UpdateEndpoints(chainID string, endpoints []string) error {
	return nil
}

//StartDeliverForChannel从订购服务动态启动新块的交付
//以引导同行。
func (ds *mockDeliveryClient) StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, f func()) error {
	return nil
}

//StopDeliverForChannel从订购服务动态停止新块的交付
//以引导同行。
func (ds *mockDeliveryClient) StopDeliverForChannel(chainID string) error {
	return nil
}

//停止终止传递服务并关闭连接
func (*mockDeliveryClient) Stop() {
}

type mockDeliveryClientFactory struct {
}

func (*mockDeliveryClientFactory) Service(g service.GossipService, endpoints []string, mcs api.MessageCryptoService) (deliverclient.DeliverService, error) {
	return &mockDeliveryClient{}, nil
}

func TestNewPeerServer(t *testing.T) {
	server, err := NewPeerServer(":4050", comm.ServerConfig{})
	assert.NoError(t, err, "NewPeerServer returned unexpected error")
	assert.Equal(t, "[::]:4050", server.Address(), "NewPeerServer returned the wrong address")
	server.Stop()

	_, err = NewPeerServer("", comm.ServerConfig{})
	assert.Error(t, err, "expected NewPeerServer to return error with missing address")
}

func TestInitChain(t *testing.T) {
	chainId := "testChain"
	chainInitializer = func(cid string) {
		assert.Equal(t, chainId, cid, "chainInitializer received unexpected cid")
	}
	InitChain(chainId)
}

func TestInitialize(t *testing.T) {
	cleanup := setupPeerFS(t)
	defer cleanup()

	Initialize(nil, &ccprovider.MockCcProviderImpl{}, (&mscc.MocksccProviderFactory{}).NewSystemChaincodeProvider(), txvalidator.MapBasedPluginMapper(map[string]validation.PluginFactory{}), nil, &ledgermocks.DeployedChaincodeInfoProvider{}, nil, &disabled.Provider{})
}

func TestCreateChainFromBlock(t *testing.T) {
	cleanup := setupPeerFS(t)
	defer cleanup()

	Initialize(nil, &ccprovider.MockCcProviderImpl{}, (&mscc.MocksccProviderFactory{}).NewSystemChaincodeProvider(), txvalidator.MapBasedPluginMapper(map[string]validation.PluginFactory{}), &platforms.Registry{}, &ledgermocks.DeployedChaincodeInfoProvider{}, nil, &disabled.Provider{})
	testChainID := fmt.Sprintf("mytestchainid-%d", rand.Int())
	block, err := configtxtest.MakeGenesisBlock(testChainID)
	if err != nil {
		fmt.Printf("Failed to create a config block, err %s\n", err)
		t.FailNow()
	}

//初始化八卦服务
	grpcServer := grpc.NewServer()
	socket, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	msptesttools.LoadMSPSetupForTesting()

	identity, _ := mgmt.GetLocalSigningIdentityOrPanic().Serialize()
	messageCryptoService := peergossip.NewMCS(&mocks.ChannelPolicyManagerGetter{}, localmsp.NewSigner(), mgmt.NewDeserializersManager())
	secAdv := peergossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())
	var defaultSecureDialOpts = func() []grpc.DialOption {
		var dialOpts []grpc.DialOption
		dialOpts = append(dialOpts, grpc.WithInsecure())
		return dialOpts
	}
	err = service.InitGossipServiceCustomDeliveryFactory(
		identity, socket.Addr().String(), grpcServer, nil,
		&mockDeliveryClientFactory{},
		messageCryptoService, secAdv, defaultSecureDialOpts)

	assert.NoError(t, err)

	go grpcServer.Serve(socket)
	defer grpcServer.Stop()

	err = CreateChainFromBlock(block, nil, nil)
	if err != nil {
		t.Fatalf("failed to create chain %s", err)
	}

//正确分类帐
	ledger := GetLedger(testChainID)
	if ledger == nil {
		t.Fatalf("failed to get correct ledger")
	}

//从分类帐获取配置块
	block, err = getCurrConfigBlockFromLedger(ledger)
	assert.NoError(t, err, "Failed to get config block from ledger")
	assert.NotNil(t, block, "Config block should not be nil")
	assert.Equal(t, uint64(0), block.Header.Number, "config block should have been block 0")

//坏账分类帐
	ledger = GetLedger("BogusChain")
	if ledger != nil {
		t.Fatalf("got a bogus ledger")
	}

//正确块
	block = GetCurrConfigBlock(testChainID)
	if block == nil {
		t.Fatalf("failed to get correct block")
	}

	cfgSupport := configSupport{}
	chCfg := cfgSupport.GetChannelConfig(testChainID)
	assert.NotNil(t, chCfg, "failed to get channel config")

//坏块
	block = GetCurrConfigBlock("BogusBlock")
	if block != nil {
		t.Fatalf("got a bogus block")
	}

//正确的策略管理器
	pmgr := GetPolicyManager(testChainID)
	if pmgr == nil {
		t.Fatal("failed to get PolicyManager")
	}

//坏的策略管理器
	pmgr = GetPolicyManager("BogusChain")
	if pmgr != nil {
		t.Fatal("got a bogus PolicyManager")
	}

//政策管理者
	pmg := NewChannelPolicyManagerGetter()
	assert.NotNil(t, pmg, "PolicyManagerGetter should not be nil")

	pmgr, ok := pmg.Manager(testChainID)
	assert.NotNil(t, pmgr, "PolicyManager should not be nil")
	assert.Equal(t, true, ok, "expected Manage() to return true")

	SetCurrConfigBlock(block, testChainID)

	channels := GetChannelsInfo()
	if len(channels) != 1 {
		t.Fatalf("incorrect number of channels")
	}

//清除链引用以启用-count n执行
	chains.Lock()
	chains.list = map[string]*chain{}
	chains.Unlock()
}

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	t.Log(ip)
}

func TestDeliverSupportManager(t *testing.T) {
//reset chains for testing
	MockInitialize()

	manager := &DeliverChainManager{}
	chainSupport := manager.GetChain("fake")
	assert.Nil(t, chainSupport, "chain support should be nil")

	MockCreateChain("testchain")
	chainSupport = manager.GetChain("testchain")
	assert.NotNil(t, chainSupport, "chain support should not be nil")
}
