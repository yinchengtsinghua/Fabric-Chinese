
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


package txvalidator

import (
	"fmt"
	"sync"

	"github.com/hyperledger/fabric/common/cauthdsl"
	ledger2 "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/capabilities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/identities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/state"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

//mapbasedpluginmapper将插件名称映射到相应的工厂
type MapBasedPluginMapper map[string]validation.PluginFactory

//PluginFactoryByName returns a plugin factory for the given plugin name, or nil if not found
func (m MapBasedPluginMapper) PluginFactoryByName(name PluginName) validation.PluginFactory {
	return m[string(name)]
}

//去：生成mokery-dir。-name pluginmapper-case underline-输出模拟/
//go:generate mokery-dir../../handlers/validation/api/-name pluginFactory-case underline-output mocks/
//go:generate mokery-dir../../handlers/validation/api/-name plugin-case underline-output mocks/

//pluginmapper将插件名称映射到相应的工厂实例。
//如果名称未与任何插件关联，则返回nil。
type PluginMapper interface {
	PluginFactoryByName(name PluginName) validation.PluginFactory
}

//去：生成mokery-dir。-name queryexecutorcreator-case underline-输出模拟/

//QueryExecutorCreator创建新的查询执行器
type QueryExecutorCreator interface {
	NewQueryExecutor() (ledger.QueryExecutor, error)
}

//上下文定义有关事务的信息
//正在验证
type Context struct {
	Seq       int
	Envelope  []byte
	TxID      string
	Channel   string
	VSCCName  string
	Policy    []byte
	Namespace string
	Block     *common.Block
}

//字符串返回此上下文的字符串表示形式
func (c Context) String() string {
	return fmt.Sprintf("Tx %s, seq %d out of %d in block %d for channel %s with validation plugin %s", c.TxID, c.Seq, len(c.Block.Data.Data), c.Block.Header.Number, c.Channel, c.VSCCName)
}

//PlugInvalidator为带有验证插件的事务赋值
type PluginValidator struct {
	sync.Mutex
	pluginChannelMapping map[PluginName]*pluginsByChannel
	PluginMapper
	QueryExecutorCreator
	msp.IdentityDeserializer
	capabilities Capabilities
}

//go:generate mokery-dir../../handlers/validation/api/capabilities/-name capabilities-case underline-output mocks/
//go:generate mokery-dir../../msp/-name identitydeserializer-case underline-output mocks/

//NewPlugInvalidator创建新的PlugInvalidator
func NewPluginValidator(pm PluginMapper, qec QueryExecutorCreator, deserializer msp.IdentityDeserializer, capabilities Capabilities) *PluginValidator {
	return &PluginValidator{
		capabilities:         capabilities,
		pluginChannelMapping: make(map[PluginName]*pluginsByChannel),
		PluginMapper:         pm,
		QueryExecutorCreator: qec,
		IdentityDeserializer: deserializer,
	}
}

func (pv *PluginValidator) ValidateWithPlugin(ctx *Context) error {
	plugin, err := pv.getOrCreatePlugin(ctx)
	if err != nil {
		return &validation.ExecutionFailureError{
			Reason: fmt.Sprintf("plugin with name %s couldn't be used: %v", ctx.VSCCName, err),
		}
	}
	err = plugin.Validate(ctx.Block, ctx.Namespace, ctx.Seq, 0, SerializedPolicy(ctx.Policy))
	validityStatus := "valid"
	if err != nil {
		validityStatus = fmt.Sprintf("invalid: %v", err)
	}
	logger.Debug("Transaction", ctx.TxID, "appears to be", validityStatus)
	return err
}

func (pv *PluginValidator) getOrCreatePlugin(ctx *Context) (validation.Plugin, error) {
	pluginFactory := pv.PluginFactoryByName(PluginName(ctx.VSCCName))
	if pluginFactory == nil {
		return nil, errors.Errorf("plugin with name %s wasn't found", ctx.VSCCName)
	}

	pluginsByChannel := pv.getOrCreatePluginChannelMapping(PluginName(ctx.VSCCName), pluginFactory)
	return pluginsByChannel.createPluginIfAbsent(ctx.Channel)

}

func (pv *PluginValidator) getOrCreatePluginChannelMapping(plugin PluginName, pf validation.PluginFactory) *pluginsByChannel {
	pv.Lock()
	defer pv.Unlock()
	endorserChannelMapping, exists := pv.pluginChannelMapping[PluginName(plugin)]
	if !exists {
		endorserChannelMapping = &pluginsByChannel{
			pluginFactory:    pf,
			channels2Plugins: make(map[string]validation.Plugin),
			pv:               pv,
		}
		pv.pluginChannelMapping[PluginName(plugin)] = endorserChannelMapping
	}
	return endorserChannelMapping
}

//plugin name定义插件在配置中出现的名称。
type PluginName string

type pluginsByChannel struct {
	sync.RWMutex
	pluginFactory    validation.PluginFactory
	channels2Plugins map[string]validation.Plugin
	pv               *PluginValidator
}

func (pbc *pluginsByChannel) createPluginIfAbsent(channel string) (validation.Plugin, error) {
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

func (pbc *pluginsByChannel) initPlugin(plugin validation.Plugin, channel string) (validation.Plugin, error) {
	pe := &PolicyEvaluator{IdentityDeserializer: pbc.pv.IdentityDeserializer}
	sf := &StateFetcherImpl{QueryExecutorCreator: pbc.pv}
	if err := plugin.Init(pe, sf, pbc.pv.capabilities); err != nil {
		return nil, errors.Wrap(err, "failed initializing plugin")
	}
	return plugin, nil
}

type PolicyEvaluator struct {
	msp.IdentityDeserializer
}

//Evaluate获取一组SignedData并评估该组签名是否满足策略
func (id *PolicyEvaluator) Evaluate(policyBytes []byte, signatureSet []*common.SignedData) error {
	pp := cauthdsl.NewPolicyProvider(id.IdentityDeserializer)
	policy, _, err := pp.NewPolicy(policyBytes)
	if err != nil {
		return err
	}
	return policy.Evaluate(signatureSet)
}

//将给定标识反序列化为msp.identity
func (id *PolicyEvaluator) DeserializeIdentity(serializedIdentity []byte) (Identity, error) {
	mspIdentity, err := id.IdentityDeserializer.DeserializeIdentity(serializedIdentity)
	if err != nil {
		return nil, err
	}
	return &identity{Identity: mspIdentity}, nil
}

type identity struct {
	msp.Identity
}

func (i *identity) GetIdentityIdentifier() *IdentityIdentifier {
	identifier := i.Identity.GetIdentifier()
	return &IdentityIdentifier{
		Id:    identifier.Id,
		Mspid: identifier.Mspid,
	}
}

type StateFetcherImpl struct {
	QueryExecutorCreator
}

func (sf *StateFetcherImpl) FetchState() (State, error) {
	qe, err := sf.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	return &StateImpl{qe}, nil
}

type StateImpl struct {
	ledger.QueryExecutor
}

func (s *StateImpl) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (ResultsIterator, error) {
	it, err := s.QueryExecutor.GetStateRangeScanIterator(namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}
	return &ResultsIteratorImpl{ResultsIterator: it}, nil
}

type ResultsIteratorImpl struct {
	ledger2.ResultsIterator
}

func (it *ResultsIteratorImpl) Next() (QueryResult, error) {
	return it.ResultsIterator.Next()
}

//序列化策略定义了已封送的策略
type SerializedPolicy []byte

//bytes返回序列化策略的te字节
func (sp SerializedPolicy) Bytes() []byte {
	return sp
}
