
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


package cscc

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/config"
	"github.com/hyperledger/fabric/common/configtx"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/genesis"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/common/mocks/scc"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/tools/configtxgen/configtxgentest"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/core/aclmgmt"
	aclmocks "github.com/hyperledger/fabric/core/aclmgmt/mocks"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/chaincode/accesscontrol"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/inproccontroller"
	"github.com/hyperledger/fabric/core/deliverservice"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	ccprovidermocks "github.com/hyperledger/fabric/core/mocks/ccprovider"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	policymocks "github.com/hyperledger/fabric/core/policy/mocks"
	"github.com/hyperledger/fabric/core/scc/cscc/mock"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	peergossip "github.com/hyperledger/fabric/peer/gossip"
	"github.com/hyperledger/fabric/peer/gossip/mocks"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

//go：生成伪造者-o mock/configmanager.go--forke name config manager。配置管理器
type configManager interface {
	config.Manager
}

//go：生成仿冒者-o mock/acl_provider.go——仿冒名称acl provider。ACL提供者
type aclProvider interface {
	aclmgmt.ACLProvider
}

//go：生成伪造者-o mock/configtx-validator.go——伪造名称configtx validator。配置X验证程序
type configtxValidator interface {
	configtx.Validator
}

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

var mockAclProvider *aclmocks.MockACLProvider

func TestMain(m *testing.M) {
	msptesttools.LoadMSPSetupForTesting()

	mockAclProvider = &aclmocks.MockACLProvider{}
	mockAclProvider.Reset()

	os.Exit(m.Run())
}

func TestConfigerInit(t *testing.T) {
	e := New(nil, nil, mockAclProvider)
	stub := shim.NewMockStub("PeerConfiger", e)

	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}
}

func TestConfigerInvokeInvalidParameters(t *testing.T) {
	e := New(nil, nil, mockAclProvider)
	stub := shim.NewMockStub("PeerConfiger", e)

	res := stub.MockInit("1", nil)
	assert.Equal(t, res.Status, int32(shim.OK), "Init failed")

	res = stub.MockInvoke("2", nil)
	assert.Equal(t, res.Status, int32(shim.ERROR), "CSCC invoke expected to fail having zero arguments")
	assert.Equal(t, res.Message, "Incorrect number of arguments, 0")

	args := [][]byte{[]byte("GetChannels")}
	res = stub.MockInvokeWithSignedProposal("3", args, nil)
	assert.Equal(t, res.Status, int32(shim.ERROR), "CSCC invoke expected to fail no signed proposal provided")
	assert.Contains(t, res.Message, "access denied for [GetChannels]")

	args = [][]byte{[]byte("fooFunction"), []byte("testChainID")}
	res = stub.MockInvoke("5", args)
	assert.Equal(t, res.Status, int32(shim.ERROR), "CSCC invoke expected wrong function name provided")
	assert.Equal(t, res.Message, "Requested function fooFunction not found.")

	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Cscc_GetConfigBlock, "testChainID", (*pb.SignedProposal)(nil)).Return(errors.New("Nil SignedProposal"))
	args = [][]byte{[]byte("GetConfigBlock"), []byte("testChainID")}
	res = stub.MockInvokeWithSignedProposal("4", args, nil)
	assert.Equal(t, res.Status, int32(shim.ERROR), "CSCC invoke expected to fail no signed proposal provided")
	assert.Contains(t, res.Message, "Nil SignedProposal")
	mockAclProvider.AssertExpectations(t)
}

func TestConfigerInvokeJoinChainMissingParams(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/tmp/hyperledgertest/")
	os.Mkdir("/tmp/hyperledgertest", 0755)
	defer os.RemoveAll("/tmp/hyperledgertest/")

	e := New(nil, nil, mockAclProvider)
	stub := shim.NewMockStub("PeerConfiger", e)

	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}

//失败的路径：应至少有一个参数
	args := [][]byte{[]byte("JoinChain")}
	if res := stub.MockInvoke("2", args); res.Status == shim.OK {
		t.Fatalf("cscc invoke JoinChain should have failed with invalid number of args: %v", args)
	}
}

func TestConfigerInvokeJoinChainWrongParams(t *testing.T) {

	viper.Set("peer.fileSystemPath", "/tmp/hyperledgertest/")
	os.Mkdir("/tmp/hyperledgertest", 0755)
	defer os.RemoveAll("/tmp/hyperledgertest/")

	e := New(nil, nil, mockAclProvider)
	stub := shim.NewMockStub("PeerConfiger", e)

	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}

//失败路径：错误参数类型
	args := [][]byte{[]byte("JoinChain"), []byte("action")}
	if res := stub.MockInvoke("2", args); res.Status == shim.OK {
		t.Fatalf("cscc invoke JoinChain should have failed with null genesis block.  args: %v", args)
	}
}

func TestConfigerInvokeJoinChainCorrectParams(t *testing.T) {
	mp := (&scc.MocksccProviderFactory{}).NewSystemChaincodeProvider()
	ccp := &ccprovidermocks.MockCcProviderImpl{}

	viper.Set("peer.fileSystemPath", "/tmp/hyperledgertest/")
	viper.Set("chaincode.executetimeout", "3s")
	os.Mkdir("/tmp/hyperledgertest", 0755)

	peer.MockInitialize()
	ledgermgmt.InitializeTestEnv()
	defer ledgermgmt.CleanupTestEnv()
	defer os.RemoveAll("/tmp/hyperledgertest/")

	e := New(ccp, mp, mockAclProvider)
	stub := shim.NewMockStub("PeerConfiger", e)

	peerEndpoint := "localhost:13611"

	ca, _ := tlsgen.NewCA()
	certGenerator := accesscontrol.NewAuthenticator(ca)
	config := chaincode.GlobalConfig()
	config.StartupTimeout = 30 * time.Second
	chaincode.NewChaincodeSupport(
		config,
		peerEndpoint,
		false,
		ca.CertBytes(),
		certGenerator,
		&ccprovider.CCInfoFSImpl{},
		nil,
		mockAclProvider,
		container.NewVMController(
			map[string]container.VMProvider{
				inproccontroller.ContainerType: inproccontroller.NewRegistry(),
			},
		),
		mp,
		platforms.NewRegistry(&golang.Platform{}),
		peer.DefaultSupport,
		&disabled.Provider{},
	)

//初始化策略检查器
	policyManagerGetter := &policymocks.MockChannelPolicyManagerGetter{
		Managers: map[string]policies.Manager{
			"mytestchainid": &policymocks.MockChannelPolicyManager{
				MockPolicy: &policymocks.MockPolicy{
					Deserializer: &policymocks.MockIdentityDeserializer{
						Identity: []byte("Alice"),
						Msg:      []byte("msg1"),
					},
				},
			},
		},
	}

	identityDeserializer := &policymocks.MockIdentityDeserializer{
		Identity: []byte("Alice"),
		Msg:      []byte("msg1"),
	}

	e.policyChecker = policy.NewPolicyChecker(
		policyManagerGetter,
		identityDeserializer,
		&policymocks.MockMSPPrincipalGetter{Principal: []byte("Alice")},
	)

	identity, _ := mgmt.GetLocalSigningIdentityOrPanic().Serialize()
	messageCryptoService := peergossip.NewMCS(&mocks.ChannelPolicyManagerGetter{}, localmsp.NewSigner(), mgmt.NewDeserializersManager())
	secAdv := peergossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())
	err := service.InitGossipServiceCustomDeliveryFactory(identity, peerEndpoint, nil, nil, &mockDeliveryClientFactory{}, messageCryptoService, secAdv, nil)
	assert.NoError(t, err)

//joinchain的成功路径
	blockBytes := mockConfigBlock()
	if blockBytes == nil {
		t.Fatalf("cscc invoke JoinChain failed because invalid block")
	}
	args := [][]byte{[]byte("JoinChain"), blockBytes}
	sProp, _ := utils.MockSignedEndorserProposalOrPanic("", &pb.ChaincodeSpec{}, []byte("Alice"), []byte("msg1"))
	identityDeserializer.Msg = sProp.ProposalBytes
	sProp.Signature = sProp.ProposalBytes

//尝试使用零块的失败路径
	res := stub.MockInvokeWithSignedProposal("2", [][]byte{[]byte("JoinChain"), nil}, sProp)
	assert.Equal(t, res.Status, int32(shim.ERROR))

//尝试使用块和零有效负载头的失败路径
	payload, _ := proto.Marshal(&cb.Payload{})
	env, _ := proto.Marshal(&cb.Envelope{
		Payload: payload,
	})
	badBlock := &cb.Block{
		Data: &cb.BlockData{
			Data: [][]byte{env},
		},
	}
	badBlockBytes := utils.MarshalOrPanic(badBlock)
	res = stub.MockInvokeWithSignedProposal("2", [][]byte{[]byte("JoinChain"), badBlockBytes}, sProp)
	assert.Equal(t, res.Status, int32(shim.ERROR))

//现在，继续使用有效的执行路径
	if res := stub.MockInvokeWithSignedProposal("2", args, sProp); res.Status != shim.OK {
		t.Fatalf("cscc invoke JoinChain failed with: %v", res.Message)
	}

//这个呼叫一定失败了
	sProp.Signature = nil
	res = stub.MockInvokeWithSignedProposal("3", args, sProp)
	if res.Status == shim.OK {
		t.Fatalf("cscc invoke JoinChain must fail : %v", res.Message)
	}
	assert.Contains(t, res.Message, "access denied for [JoinChain][mytestchainid]")
	sProp.Signature = sProp.ProposalBytes

//查询配置块
//chainID：=[]字节143、222、22、192、73、145、76、110、167、154、118、66、132、204、113、168
	chainID, err := utils.GetChainIDFromBlockBytes(blockBytes)
	if err != nil {
		t.Fatalf("cscc invoke JoinChain failed with: %v", err)
	}

//测试getconfigblock上的acl故障
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Cscc_GetConfigBlock, "mytestchainid", sProp).Return(errors.New("Failed authorization"))
	args = [][]byte{[]byte("GetConfigBlock"), []byte(chainID)}
	res = stub.MockInvokeWithSignedProposal("2", args, sProp)
	if res.Status == shim.OK {
		t.Fatalf("cscc invoke GetConfigBlock should have failed: %v", res.Message)
	}
	assert.Contains(t, res.Message, "Failed authorization")
	mockAclProvider.AssertExpectations(t)

//用acl测试好吗
	mockAclProvider.Reset()
	mockAclProvider.On("CheckACL", resources.Cscc_GetConfigBlock, "mytestchainid", sProp).Return(nil)
	if res := stub.MockInvokeWithSignedProposal("2", args, sProp); res.Status != shim.OK {
		t.Fatalf("cscc invoke GetConfigBlock failed with: %v", res.Message)
	}

//为对等端获取频道
	args = [][]byte{[]byte(GetChannels)}
	res = stub.MockInvokeWithSignedProposal("2", args, sProp)
	if res.Status != shim.OK {
		t.FailNow()
	}

	cqr := &pb.ChannelQueryResponse{}
	err = proto.Unmarshal(res.Payload, cqr)
	if err != nil {
		t.FailNow()
	}

//对等联接了一个通道，因此查询应返回具有一个通道的数组
	if len(cqr.GetChannels()) != 1 {
		t.FailNow()
	}
}

func TestGetConfigTree(t *testing.T) {
	aclProvider := &mock.ACLProvider{}
	configMgr := &mock.ConfigManager{}
	pc := &PeerConfiger{
		aclProvider: aclProvider,
		configMgr:   configMgr,
	}

	args := [][]byte{[]byte("GetConfigTree"), []byte("testchan")}

	t.Run("Success", func(t *testing.T) {
		ctxv := &mock.ConfigtxValidator{}
		configMgr.GetChannelConfigReturns(ctxv)
		testConfig := &cb.Config{
			ChannelGroup: &cb.ConfigGroup{
				Values: map[string]*cb.ConfigValue{
					"foo": {
						Value: []byte("bar"),
					},
				},
			},
		}
		ctxv.ConfigProtoReturns(testConfig)
		res := pc.InvokeNoShim(args, nil)
		assert.Equal(t, int32(shim.OK), res.Status)
		checkConfig := &pb.ConfigTree{}
		err := proto.Unmarshal(res.Payload, checkConfig)
		assert.NoError(t, err)
		assert.True(t, proto.Equal(testConfig, checkConfig.ChannelConfig))
	})

	t.Run("MissingConfig", func(t *testing.T) {
		ctxv := &mock.ConfigtxValidator{}
		configMgr.GetChannelConfigReturns(ctxv)
		res := pc.InvokeNoShim(args, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Equal(t, "Unknown chain ID, testchan", res.Message)
	})

	t.Run("NilChannel", func(t *testing.T) {
		ctxv := &mock.ConfigtxValidator{}
		configMgr.GetChannelConfigReturns(ctxv)
		res := pc.InvokeNoShim([][]byte{[]byte("GetConfigTree"), nil}, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Equal(t, "Chain ID must not be nil", res.Message)
	})

	t.Run("BadACL", func(t *testing.T) {
		aclProvider.CheckACLReturns(fmt.Errorf("fake-error"))
		res := pc.InvokeNoShim(args, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Equal(t, "access denied for [GetConfigTree][testchan]: fake-error", res.Message)
	})
}

func TestSimulateConfigTreeUpdate(t *testing.T) {
	aclProvider := &mock.ACLProvider{}
	configMgr := &mock.ConfigManager{}
	pc := &PeerConfiger{
		aclProvider: aclProvider,
		configMgr:   configMgr,
	}

	testUpdate := &cb.Envelope{
		Payload: utils.MarshalOrPanic(&cb.Payload{
			Header: &cb.Header{
				ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{
					Type: int32(cb.HeaderType_CONFIG_UPDATE),
				}),
			},
		}),
	}

	args := [][]byte{[]byte("SimulateConfigTreeUpdate"), []byte("testchan"), utils.MarshalOrPanic(testUpdate)}

	t.Run("Success", func(t *testing.T) {
		ctxv := &mock.ConfigtxValidator{}
		configMgr.GetChannelConfigReturns(ctxv)
		res := pc.InvokeNoShim(args, nil)
		assert.Equal(t, int32(shim.OK), res.Status, res.Message)
	})

	t.Run("BadUpdate", func(t *testing.T) {
		ctxv := &mock.ConfigtxValidator{}
		configMgr.GetChannelConfigReturns(ctxv)
		ctxv.ProposeConfigUpdateReturns(nil, fmt.Errorf("fake-error"))
		res := pc.InvokeNoShim(args, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Equal(t, "fake-error", res.Message)
	})

	t.Run("BadType", func(t *testing.T) {
		res := pc.InvokeNoShim([][]byte{
			args[0],
			args[1],
			utils.MarshalOrPanic(&cb.Envelope{
				Payload: utils.MarshalOrPanic(&cb.Payload{
					Header: &cb.Header{
						ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{
							Type: int32(cb.HeaderType_ENDORSER_TRANSACTION),
						}),
					},
				}),
			}),
		}, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Equal(t, "invalid payload header type: 3", res.Message)
	})

	t.Run("BadEnvelope", func(t *testing.T) {
		res := pc.InvokeNoShim([][]byte{
			args[0],
			args[1],
			[]byte("garbage"),
		}, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Contains(t, res.Message, "proto:")
	})

	t.Run("NilChainID", func(t *testing.T) {
		res := pc.InvokeNoShim([][]byte{
			args[0],
			nil,
			args[2],
		}, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Equal(t, "Chain ID must not be nil", res.Message)
	})

	t.Run("BadACL", func(t *testing.T) {
		aclProvider.CheckACLReturns(fmt.Errorf("fake-error"))
		res := pc.InvokeNoShim(args, nil)
		assert.NotEqual(t, int32(shim.OK), res.Status)
		assert.Equal(t, "access denied for [SimulateConfigTreeUpdate][testchan]: fake-error", res.Message)
	})
}

func TestPeerConfiger_SubmittingOrdererGenesis(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/tmp/hyperledgertest/")
	os.Mkdir("/tmp/hyperledgertest", 0755)
	defer os.RemoveAll("/tmp/hyperledgertest/")

	e := New(nil, nil, nil)
	stub := shim.NewMockStub("PeerConfiger", e)

	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}
	conf := configtxgentest.Load(genesisconfig.SampleSingleMSPSoloProfile)
	conf.Application = nil
	cg, err := encoder.NewChannelGroup(conf)
	assert.NoError(t, err)
	block, err := genesis.NewFactoryImpl(cg).Block("mytestchainid")
	assert.NoError(t, err)
	blockBytes := utils.MarshalOrPanic(block)

//失败路径：错误参数类型
	args := [][]byte{[]byte("JoinChain"), []byte(blockBytes)}
	if res := stub.MockInvoke("2", args); res.Status == shim.OK {
		t.Fatalf("cscc invoke JoinChain should have failed with wrong genesis block.  args: %v", args)
	} else {
		assert.Contains(t, res.Message, "missing Application configuration group")
	}
}

func mockConfigBlock() []byte {
	var blockBytes []byte = nil
	block, err := configtxtest.MakeGenesisBlock("mytestchainid")
	if err == nil {
		blockBytes = utils.MarshalOrPanic(block)
	}
	return blockBytes
}
