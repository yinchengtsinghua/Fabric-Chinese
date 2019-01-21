
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


package cc

import (
	"sync"

	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/pkg/errors"
)

var (
//Logger是此包的日志记录实例。
//它被导出是因为测试会覆盖其后端
	Logger = flogging.MustGetLogger("discovery.lifecycle")
)

//Lifecycle管理有关链码生命周期的信息
type Lifecycle struct {
	sync.RWMutex
	listeners              []LifeCycleChangeListener
	installedCCs           []chaincode.InstalledChaincode
	deployedCCsByChannel   map[string]*chaincode.MetadataMapping
	queryCreatorsByChannel map[string]QueryCreator
}

//LifecycleChangeListener在元数据发生更改时运行
//特定通道上下文中的链代码
type LifeCycleChangeListener interface {
	LifeCycleChangeListener(channel string, chaincodes chaincode.MetadataSet)
}

//handleMetadataUpdate在链代码生命周期更改时触发
type HandleMetadataUpdate func(channel string, chaincodes chaincode.MetadataSet)

//去：生成mokery-dir。-name lifecyclechangelistener-case underline-output mocks/

//LifecycleChangeListener在元数据发生更改时运行
//
func (mdUpdate HandleMetadataUpdate) LifeCycleChangeListener(channel string, chaincodes chaincode.MetadataSet) {
	mdUpdate(channel, chaincodes)
}

//去：生成mokery-dir。-名称枚举器-大小写下划线-输出模拟/

//枚举器枚举链代码
type Enumerator interface {
//
	Enumerate() ([]chaincode.InstalledChaincode, error)
}

//
type Enumerate func() ([]chaincode.InstalledChaincode, error)

//枚举枚举链代码
func (listCCs Enumerate) Enumerate() ([]chaincode.InstalledChaincode, error) {
	return listCCs()
}

//

//查询查询状态
type Query interface {
//GetState获取给定命名空间和键的值。对于chaincode，命名空间对应于chaincodeid
	GetState(namespace string, key string) ([]byte, error)

//完成释放由QueryExecutor占用的资源
	Done()
}

//去：生成mokery-dir。-name querycreator-case underline-输出模拟/

//
type QueryCreator interface {
//
	NewQuery() (Query, error)
}

//querycreatorfunc创建新查询
type QueryCreatorFunc func() (Query, error)

//new query创建新查询，或失败时出错
func (qc QueryCreatorFunc) NewQuery() (Query, error) {
	return qc()
}

//NewLifecycle创建新的生命周期实例
func NewLifeCycle(installedChaincodes Enumerator) (*Lifecycle, error) {
	installedCCs, err := installedChaincodes.Enumerate()
	if err != nil {
		return nil, errors.Wrap(err, "failed listing installed chaincodes")
	}

	lc := &Lifecycle{
		installedCCs:           installedCCs,
		deployedCCsByChannel:   make(map[string]*chaincode.MetadataMapping),
		queryCreatorsByChannel: make(map[string]QueryCreator),
	}

	return lc, nil
}

//元数据返回给定通道上链码的元数据，
//如果找不到或检索时出错，则为零。
func (lc *Lifecycle) Metadata(channel string, cc string, collections bool) *chaincode.Metadata {
	queryCreator := lc.queryCreatorsByChannel[channel]
	if queryCreator == nil {
		Logger.Warning("Requested Metadata for non-existent channel", channel)
		return nil
	}
//
//调用中未指定任何集合。
	if md, found := lc.deployedCCsByChannel[channel].Lookup(cc); found && !collections {
		Logger.Debug("Returning metadata for channel", channel, ", chaincode", cc, ":", md)
		return &md
	}
	query, err := queryCreator.NewQuery()
	if err != nil {
		Logger.Error("Failed obtaining new query for channel", channel, ":", err)
		return nil
	}
	md, err := DeployedChaincodes(query, AcceptAll, collections, cc)
	if err != nil {
		Logger.Error("Failed querying LSCC for channel", channel, ":", err)
		return nil
	}
	if len(md) == 0 {
		Logger.Info("Chaincode", cc, "isn't defined in channel", channel)
		return nil
	}

	return &md[0]
}

func (lc *Lifecycle) initMetadataForChannel(channel string, queryCreator QueryCreator) error {
	if lc.isChannelMetadataInitialized(channel) {
		return nil
	}
//为通道创建新的元数据映射
	query, err := queryCreator.NewQuery()
	if err != nil {
		return errors.WithStack(err)
	}
	ccs, err := queryChaincodeDefinitions(query, lc.installedCCs, DeployedChaincodes)
	if err != nil {
		return errors.WithStack(err)
	}
	lc.createMetadataForChannel(channel, queryCreator)
	lc.updateState(channel, ccs)
	return nil
}

func (lc *Lifecycle) createMetadataForChannel(channel string, newQuery QueryCreator) {
	lc.Lock()
	defer lc.Unlock()
	lc.deployedCCsByChannel[channel] = chaincode.NewMetadataMapping()
	lc.queryCreatorsByChannel[channel] = newQuery
}

func (lc *Lifecycle) isChannelMetadataInitialized(channel string) bool {
	lc.RLock()
	defer lc.RUnlock()
	_, exists := lc.deployedCCsByChannel[channel]
	return exists
}

func (lc *Lifecycle) updateState(channel string, ccUpdate chaincode.MetadataSet) {
	lc.RLock()
	defer lc.RUnlock()
	for _, cc := range ccUpdate {
		lc.deployedCCsByChannel[channel].Update(cc)
	}
}

func (lc *Lifecycle) fireChangeListeners(channel string) {
	lc.RLock()
	md := lc.deployedCCsByChannel[channel]
	lc.RUnlock()
	for _, listener := range lc.listeners {
		aggregatedMD := md.Aggregate()
		listener.LifeCycleChangeListener(channel, aggregatedMD)
	}
	Logger.Debug("Listeners for channel", channel, "invoked")
}

//newchannelsubscription订阅通道
func (lc *Lifecycle) NewChannelSubscription(channel string, queryCreator QueryCreator) (*Subscription, error) {
	sub := &Subscription{
		lc:             lc,
		channel:        channel,
		queryCreator:   queryCreator,
		pendingUpdates: make(chan *cceventmgmt.ChaincodeDefinition, 1),
	}
//初始化通道的元数据。
//这将加载有关所有已安装的链代码的元数据
	if err := lc.initMetadataForChannel(channel, queryCreator); err != nil {
		return nil, errors.WithStack(err)
	}
	lc.fireChangeListeners(channel)
	return sub, nil
}

//
func (lc *Lifecycle) AddListener(listener LifeCycleChangeListener) {
	lc.Lock()
	defer lc.Unlock()
	lc.listeners = append(lc.listeners, listener)
}
