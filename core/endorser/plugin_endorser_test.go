
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package endorser_test

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/mocks/ledger"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/endorser"
	"github.com/hyperledger/fabric/core/endorser/mocks"
	"github.com/hyperledger/fabric/core/handlers/endorsement/api"
	. "github.com/hyperledger/fabric/core/handlers/endorsement/api/state"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/peer"
	transientstore2 "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	mockTransientStoreRetriever = transientStoreRetriever()
	mockTransientStore          = &mocks.Store{}
)

func TestPluginEndorserNotFound(t *testing.T) {
	pluginMapper := &mocks.PluginMapper{}
	pluginMapper.On("PluginFactoryByName", endorser.PluginName("notfound")).Return(nil)
	pluginEndorser := endorser.NewPluginEndorser(&endorser.PluginSupport{
		PluginMapper: pluginMapper,
	})
	resp, err := pluginEndorser.EndorseWithPlugin(endorser.Context{
		Response:   &peer.Response{},
		PluginName: "notfound",
	})
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "plugin with name notfound wasn't found")
}

func TestPluginEndorserGreenPath(t *testing.T) {
	proposal, _, err := utils.CreateChaincodeProposal(common.HeaderType_ENDORSER_TRANSACTION, "mychannel", &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: "mycc"},
		},
	}, []byte{1, 2, 3})
	assert.NoError(t, err)
	expectedSignature := []byte{5, 4, 3, 2, 1}
	expectedProposalResponsePayload := []byte{1, 2, 3}
	pluginMapper := &mocks.PluginMapper{}
	pluginFactory := &mocks.PluginFactory{}
	plugin := &mocks.Plugin{}
	plugin.On("Endorse", mock.Anything, mock.Anything).Return(&peer.Endorsement{Signature: expectedSignature}, expectedProposalResponsePayload, nil)
	pluginMapper.On("PluginFactoryByName", endorser.PluginName("plugin")).Return(pluginFactory)
//期望插件在这个测试中只被实例化一次，因为我们用相同的参数调用背书器。
	plugin.On("Init", mock.Anything, mock.Anything).Return(nil).Once()
	pluginFactory.On("New").Return(plugin).Once()
	sif := &mocks.SigningIdentityFetcher{}
	cs := &mocks.ChannelStateRetriever{}
	queryCreator := &mocks.QueryCreator{}
	cs.On("NewQueryCreator", "mychannel").Return(queryCreator, nil)
	pluginEndorser := endorser.NewPluginEndorser(&endorser.PluginSupport{
		ChannelStateRetriever:   cs,
		SigningIdentityFetcher:  sif,
		PluginMapper:            pluginMapper,
		TransientStoreRetriever: mockTransientStoreRetriever,
	})
	ctx := endorser.Context{
		Response:   &peer.Response{},
		PluginName: "plugin",
		Proposal:   proposal,
		ChaincodeID: &peer.ChaincodeID{
			Name: "mycc",
		},
		Channel: "mychannel",
	}

//场景一：第一次呼吁认可
	resp, err := pluginEndorser.EndorseWithPlugin(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedSignature, resp.Endorsement.Signature)
	assert.Equal(t, expectedProposalResponsePayload, resp.Payload)
//确保State和SigningIdentityFetcher都传递给了Init（）。
	plugin.AssertCalled(t, "Init", &endorser.ChannelState{QueryCreator: queryCreator, Store: mockTransientStore}, sif)

//场景二：再次呼叫背书。
//确保插件没有再次实例化-这意味着相同的实例
//用于服务请求。
//另外-检查插件上是否多次调用了init（）。
	resp, err = pluginEndorser.EndorseWithPlugin(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedSignature, resp.Endorsement.Signature)
	assert.Equal(t, expectedProposalResponsePayload, resp.Payload)
	pluginFactory.AssertNumberOfCalls(t, "New", 1)
	plugin.AssertNumberOfCalls(t, "Init", 1)

//场景三：使用无通道上下文调用认可。
//应再次调用init方法，但这次是通道状态对象
//不应传入init。
	ctx.Channel = ""
	pluginFactory.On("New").Return(plugin).Once()
	plugin.On("Init", mock.Anything).Return(nil).Once()
	resp, err = pluginEndorser.EndorseWithPlugin(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedSignature, resp.Endorsement.Signature)
	assert.Equal(t, expectedProposalResponsePayload, resp.Payload)
	plugin.AssertCalled(t, "Init", sif)
}

func TestPluginEndorserErrors(t *testing.T) {
	pluginMapper := &mocks.PluginMapper{}
	pluginFactory := &mocks.PluginFactory{}
	plugin := &mocks.Plugin{}
	plugin.On("Endorse", mock.Anything, mock.Anything)
	pluginMapper.On("PluginFactoryByName", endorser.PluginName("plugin")).Return(pluginFactory)
	pluginFactory.On("New").Return(plugin)
	sif := &mocks.SigningIdentityFetcher{}
	cs := &mocks.ChannelStateRetriever{}
	queryCreator := &mocks.QueryCreator{}
	cs.On("NewQueryCreator", "mychannel").Return(queryCreator, nil)
	pluginEndorser := endorser.NewPluginEndorser(&endorser.PluginSupport{
		ChannelStateRetriever:   cs,
		SigningIdentityFetcher:  sif,
		PluginMapper:            pluginMapper,
		TransientStoreRetriever: mockTransientStoreRetriever,
	})

//方案一：插件初始化失败
	t.Run("PluginInitializationFailure", func(t *testing.T) {
		plugin.On("Init", mock.Anything, mock.Anything).Return(errors.New("plugin initialization failed")).Once()
		resp, err := pluginEndorser.EndorseWithPlugin(endorser.Context{
			PluginName: "plugin",
			Channel:    "mychannel",
			Response:   &peer.Response{},
		})
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "plugin initialization failed")
	})

//场景二：在上下文中传递空建议，解析失败
	t.Run("EmptyProposal", func(t *testing.T) {
		plugin.On("Init", mock.Anything, mock.Anything).Return(nil).Once()
		ctx := endorser.Context{
			Response:   &peer.Response{},
			PluginName: "plugin",
			ChaincodeID: &peer.ChaincodeID{
				Name: "mycc",
			},
			Proposal: &peer.Proposal{},
			Channel:  "mychannel",
		}
		resp, err := pluginEndorser.EndorseWithPlugin(ctx)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "could not compute proposal hash")
	})

//方案三：提案的标题无效
	t.Run("InvalidHeader in the proposal", func(t *testing.T) {
		ctx := endorser.Context{
			Response:   &peer.Response{},
			PluginName: "plugin",
			ChaincodeID: &peer.ChaincodeID{
				Name: "mycc",
			},
			Proposal: &peer.Proposal{
				Header: []byte{1, 2, 3},
			},
			Channel: "mychannel",
		}
		resp, err := pluginEndorser.EndorseWithPlugin(ctx)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed parsing header")
	})

//方案四：提案的响应状态代码指示错误
	t.Run("ResponseStatusContainsError", func(t *testing.T) {
		r := &peer.Response{
			Status:  shim.ERRORTHRESHOLD,
			Payload: []byte{1, 2, 3},
			Message: "bla bla",
		}
		resp, err := pluginEndorser.EndorseWithPlugin(endorser.Context{
			Response: r,
		})
		assert.Equal(t, &peer.ProposalResponse{Response: r}, resp)
		assert.NoError(t, err)
	})

//方案五：提案的答复为零
	t.Run("ResponseIsNil", func(t *testing.T) {
		resp, err := pluginEndorser.EndorseWithPlugin(endorser.Context{})
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "response is nil")
	})
}

func transientStoreRetriever() *mocks.TransientStoreRetriever {
	storeRetriever := &mocks.TransientStoreRetriever{}
	storeRetriever.On("StoreForChannel", mock.Anything).Return(mockTransientStore)
	return storeRetriever
}

type fakeEndorsementPlugin struct {
	StateFetcher
}

func (fep *fakeEndorsementPlugin) Endorse(payload []byte, sp *peer.SignedProposal) (*peer.Endorsement, []byte, error) {
	state, _ := fep.StateFetcher.FetchState()
	txrws, _ := state.GetTransientByTXID("tx")
	b, _ := proto.Marshal(txrws[0])
	return nil, b, nil
}

func (fep *fakeEndorsementPlugin) Init(dependencies ...endorsement.Dependency) error {
	for _, dep := range dependencies {
		if state, isState := dep.(StateFetcher); isState {
			fep.StateFetcher = state
			return nil
		}
	}
	panic("could not find State dependency")
}

type rwsetScanner struct {
	mock.Mock
	data []*rwset.TxPvtReadWriteSet
}

func (*rwsetScanner) Next() (*transientstore.EndorserPvtSimulationResults, error) {
	panic("implement me")
}

func (rws *rwsetScanner) NextWithConfig() (*transientstore.EndorserPvtSimulationResultsWithConfig, error) {
	if len(rws.data) == 0 {
		return nil, nil
	}
	res := rws.data[0]
	rws.data = rws.data[1:]
	return &transientstore.EndorserPvtSimulationResultsWithConfig{
		PvtSimulationResultsWithConfig: &transientstore2.TxPvtReadWriteSetWithConfigInfo{
			PvtRwset: res,
		},
	}, nil
}

func (rws *rwsetScanner) Close() {
	rws.Called()
}

func TestTransientStore(t *testing.T) {
	plugin := &fakeEndorsementPlugin{}
	factory := &mocks.PluginFactory{}
	factory.On("New").Return(plugin)
	sif := &mocks.SigningIdentityFetcher{}
	cs := &mocks.ChannelStateRetriever{}
	queryCreator := &mocks.QueryCreator{}
	queryCreator.On("NewQueryExecutor").Return(&ledger.MockQueryExecutor{}, nil)
	cs.On("NewQueryCreator", "mychannel").Return(queryCreator, nil)

	transientStore := &mocks.Store{}
	storeRetriever := &mocks.TransientStoreRetriever{}
	storeRetriever.On("StoreForChannel", mock.Anything).Return(transientStore)

	pluginEndorser := endorser.NewPluginEndorser(&endorser.PluginSupport{
		ChannelStateRetriever:  cs,
		SigningIdentityFetcher: sif,
		PluginMapper: endorser.MapBasedPluginMapper{
			"plugin": factory,
		},
		TransientStoreRetriever: storeRetriever,
	})

	proposal, _, err := utils.CreateChaincodeProposal(common.HeaderType_ENDORSER_TRANSACTION, "mychannel", &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: "mycc"},
		},
	}, []byte{1, 2, 3})
	assert.NoError(t, err)
	ctx := endorser.Context{
		Response:   &peer.Response{},
		PluginName: "plugin",
		Proposal:   proposal,
		ChaincodeID: &peer.ChaincodeID{
			Name: "mycc",
		},
		Channel: "mychannel",
	}

	rws := &rwset.TxPvtReadWriteSet{
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "ns",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "col",
					},
				},
			},
		},
	}
	scanner := &rwsetScanner{
		data: []*rwset.TxPvtReadWriteSet{rws},
	}
	scanner.On("Close")

	transientStore.On("GetTxPvtRWSetByTxid", mock.Anything, mock.Anything).Return(scanner, nil)

	resp, err := pluginEndorser.EndorseWithPlugin(ctx)
	assert.NoError(t, err)

	txrws := &rwset.TxPvtReadWriteSet{}
	err = proto.Unmarshal(resp.Payload, txrws)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(rws, txrws))
	scanner.AssertCalled(t, "Close")
}
