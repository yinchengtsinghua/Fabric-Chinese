
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2018保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package endorser

import (
	"fmt"
	"sync"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/handlers/endorsement/api"
	endorsement3 "github.com/hyperledger/fabric/core/handlers/endorsement/api/identities"
	"github.com/hyperledger/fabric/core/transientstore"
	pb "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//去：生成mokery-dir。-name transientstoreretriever-case underline-output mocks/
//go:generate mokery-dir../transientstore/-name store-case underline-output mocks/

//TransientStoreRetriever检索临时存储
type TransientStoreRetriever interface {
//store for channel返回给定通道的临时存储
	StoreForChannel(channel string) transientstore.Store
}

//去：生成mokery-dir。-name channelstateretriever-case underline-output mocks/

//ChannelStateRetriever检索通道状态
type ChannelStateRetriever interface {
//channelstate返回给定通道的querycreator
	NewQueryCreator(channel string) (QueryCreator, error)
}

//去：生成mokery-dir。-name pluginmapper-case underline-输出模拟/

//pluginmapper将插件名称映射到相应的工厂
type PluginMapper interface {
	PluginFactoryByName(name PluginName) endorsement.PluginFactory
}

//mapbasedpluginmapper将插件名称映射到相应的工厂
type MapBasedPluginMapper map[string]endorsement.PluginFactory

//pluginFactoryByName返回给定插件名称的插件工厂，如果找不到，则返回nil。
func (m MapBasedPluginMapper) PluginFactoryByName(name PluginName) endorsement.PluginFactory {
	return m[string(name)]
}

//上下文定义与飞行中背书相关的数据
type Context struct {
	PluginName     string
	Channel        string
	TxID           string
	Proposal       *pb.Proposal
	SignedProposal *pb.SignedProposal
	Visibility     []byte
	Response       *pb.Response
	Event          []byte
	ChaincodeID    *pb.ChaincodeID
	SimRes         []byte
}

//字符串返回此上下文的文本表示形式
func (c Context) String() string {
	return fmt.Sprintf("{plugin: %s, channel: %s, tx: %s, chaincode: %s}", c.PluginName, c.Channel, c.TxID, c.ChaincodeID.Name)
}

//PluginSupport聚合支持接口
//插件背书器操作所需的
type PluginSupport struct {
	ChannelStateRetriever
	endorsement3.SigningIdentityFetcher
	PluginMapper
	TransientStoreRetriever
}

//NewPluginendorser认可使用插件
func NewPluginEndorser(ps *PluginSupport) *PluginEndorser {
	return &PluginEndorser{
		SigningIdentityFetcher:  ps.SigningIdentityFetcher,
		PluginMapper:            ps.PluginMapper,
		pluginChannelMapping:    make(map[PluginName]*pluginsByChannel),
		ChannelStateRetriever:   ps.ChannelStateRetriever,
		TransientStoreRetriever: ps.TransientStoreRetriever,
	}
}

//plugin name定义插件在配置中出现的名称。
type PluginName string

type pluginsByChannel struct {
	sync.RWMutex
	pluginFactory    endorsement.PluginFactory
	channels2Plugins map[string]endorsement.Plugin
	pe               *PluginEndorser
}

func (pbc *pluginsByChannel) createPluginIfAbsent(channel string) (endorsement.Plugin, error) {
	pbc.RLock()
	plugin, exists := pbc.channels2Plugins[channel]
	pbc.RUnlock()
	if exists {
		return plugin, nil
	}

	pbc.Lock()
	defer pbc.Unlock()
	plugin, exists = pbc.channels2Plugins[channel]
	if exists {
		return plugin, nil
	}

	pluginInstance := pbc.pluginFactory.New()
	plugin, err := pbc.initPlugin(pluginInstance, channel)
	if err != nil {
		return nil, err
	}
	pbc.channels2Plugins[channel] = plugin
	return plugin, nil
}

func (pbc *pluginsByChannel) initPlugin(plugin endorsement.Plugin, channel string) (endorsement.Plugin, error) {
	var dependencies []endorsement.Dependency
	var err error
//如果这是渠道认可，请将渠道状态添加为依赖项
	if channel != "" {
		query, err := pbc.pe.NewQueryCreator(channel)
		if err != nil {
			return nil, errors.Wrap(err, "failed obtaining channel state")
		}
		store := pbc.pe.TransientStoreRetriever.StoreForChannel(channel)
		if store == nil {
			return nil, errors.Errorf("transient store for channel %s was not initialized", channel)
		}
		dependencies = append(dependencies, &ChannelState{QueryCreator: query, Store: store})
	}
//将SigningIdentityFetcher添加为依赖项
	dependencies = append(dependencies, pbc.pe.SigningIdentityFetcher)
	err = plugin.Init(dependencies...)
	if err != nil {
		return nil, err
	}
	return plugin, nil
}

//插件支持者使用插件的建议响应
type PluginEndorser struct {
	sync.Mutex
	PluginMapper
	pluginChannelMapping map[PluginName]*pluginsByChannel
	ChannelStateRetriever
	endorsement3.SigningIdentityFetcher
	TransientStoreRetriever
}

//背书WithPlugin用插件背书响应
func (pe *PluginEndorser) EndorseWithPlugin(ctx Context) (*pb.ProposalResponse, error) {
	endorserLogger.Debug("Entering endorsement for", ctx)

	if ctx.Response == nil {
		return nil, errors.New("response is nil")
	}

	if ctx.Response.Status >= shim.ERRORTHRESHOLD {
		return &pb.ProposalResponse{Response: ctx.Response}, nil
	}

	plugin, err := pe.getOrCreatePlugin(PluginName(ctx.PluginName), ctx.Channel)
	if err != nil {
		endorserLogger.Warning("Endorsement with plugin for", ctx, " failed:", err)
		return nil, errors.Errorf("plugin with name %s could not be used: %v", ctx.PluginName, err)
	}

	prpBytes, err := proposalResponsePayloadFromContext(ctx)
	if err != nil {
		endorserLogger.Warning("Endorsement with plugin for", ctx, " failed:", err)
		return nil, errors.Wrap(err, "failed assembling proposal response payload")
	}

	endorsement, prpBytes, err := plugin.Endorse(prpBytes, ctx.SignedProposal)
	if err != nil {
		endorserLogger.Warning("Endorsement with plugin for", ctx, " failed:", err)
		return nil, errors.WithStack(err)
	}

	resp := &pb.ProposalResponse{
		Version:     1,
		Endorsement: endorsement,
		Payload:     prpBytes,
		Response:    ctx.Response,
	}
	endorserLogger.Debug("Exiting", ctx)
	return resp, nil
}

//GetAndStorePlugin返回给定插件名称和通道的插件实例
func (pe *PluginEndorser) getOrCreatePlugin(plugin PluginName, channel string) (endorsement.Plugin, error) {
	pluginFactory := pe.PluginFactoryByName(plugin)
	if pluginFactory == nil {
		return nil, errors.Errorf("plugin with name %s wasn't found", plugin)
	}

	pluginsByChannel := pe.getOrCreatePluginChannelMapping(PluginName(plugin), pluginFactory)
	return pluginsByChannel.createPluginIfAbsent(channel)
}

func (pe *PluginEndorser) getOrCreatePluginChannelMapping(plugin PluginName, pf endorsement.PluginFactory) *pluginsByChannel {
	pe.Lock()
	defer pe.Unlock()
	endorserChannelMapping, exists := pe.pluginChannelMapping[PluginName(plugin)]
	if !exists {
		endorserChannelMapping = &pluginsByChannel{
			pluginFactory:    pf,
			channels2Plugins: make(map[string]endorsement.Plugin),
			pe:               pe,
		}
		pe.pluginChannelMapping[PluginName(plugin)] = endorserChannelMapping
	}
	return endorserChannelMapping
}

func proposalResponsePayloadFromContext(ctx Context) ([]byte, error) {
	hdr, err := putils.GetHeader(ctx.Proposal.Header)
	if err != nil {
		endorserLogger.Warning("Failed parsing header", err)
		return nil, errors.Wrap(err, "failed parsing header")
	}

	pHashBytes, err := putils.GetProposalHash1(hdr, ctx.Proposal.Payload, ctx.Visibility)
	if err != nil {
		endorserLogger.Warning("Failed computing proposal hash", err)
		return nil, errors.Wrap(err, "could not compute proposal hash")
	}

	prpBytes, err := putils.GetBytesProposalResponsePayload(pHashBytes, ctx.Response, ctx.SimRes, ctx.Event, ctx.ChaincodeID)
	if err != nil {
		endorserLogger.Warning("Failed marshaling the proposal response payload to bytes", err)
		return nil, errors.New("failure while marshaling the ProposalResponsePayload")
	}
	return prpBytes, nil
}
