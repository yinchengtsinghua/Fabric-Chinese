
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


package ledgermgmt

import (
	"bytes"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/core/ledger/customtx"
	"github.com/hyperledger/fabric/core/ledger/kvledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ledgermgmt")

//如果已打开具有给定ID的分类帐，则由CreateLedger调用引发errlegralreadyOpened。
var ErrLedgerAlreadyOpened = errors.New("ledger already opened")

//当使用分类帐管理时，将在初始化此项之前引发errlegrmgmtnotinitialized。
var ErrLedgerMgmtNotInitialized = errors.New("ledger mgmt should be initialized before using")

var openedLedgers map[string]ledger.PeerLedger
var ledgerProvider ledger.PeerLedgerProvider
var lock sync.Mutex
var initialized bool
var once sync.Once

//初始值设定项封装了分类帐模块的所有外部依赖项
type Initializer struct {
	CustomTxProcessors            customtx.Processors
	PlatformRegistry              *platforms.Registry
	DeployedChaincodeInfoProvider ledger.DeployedChaincodeInfoProvider
	MembershipInfoProvider        ledger.MembershipInfoProvider
	MetricsProvider               metrics.Provider
	HealthCheckRegistry           ledger.HealthCheckRegistry
}

//初始化初始化Ledgermgmt
func Initialize(initializer *Initializer) {
	once.Do(func() {
		initialize(initializer)
	})
}

func initialize(initializer *Initializer) {
	logger.Info("Initializing ledger mgmt")
	lock.Lock()
	defer lock.Unlock()
	initialized = true
	openedLedgers = make(map[string]ledger.PeerLedger)
	customtx.Initialize(initializer.CustomTxProcessors)
	cceventmgmt.Initialize(&chaincodeInfoProviderImpl{
		initializer.PlatformRegistry,
		initializer.DeployedChaincodeInfoProvider,
	})
	finalStateListeners := addListenerForCCEventsHandler(initializer.DeployedChaincodeInfoProvider, []ledger.StateListener{})
	provider, err := kvledger.NewProvider()
	if err != nil {
		panic(errors.WithMessage(err, "Error in instantiating ledger provider"))
	}
	provider.Initialize(&ledger.Initializer{
		StateListeners:                finalStateListeners,
		DeployedChaincodeInfoProvider: initializer.DeployedChaincodeInfoProvider,
		MembershipInfoProvider:        initializer.MembershipInfoProvider,
		MetricsProvider:               initializer.MetricsProvider,
		HealthCheckRegistry:           initializer.HealthCheckRegistry,
	})
	ledgerProvider = provider
	logger.Info("ledger mgmt initialized")
}

//CreateLedger使用给定的Genesis块创建一个新的分类帐。
//此函数确保创建分类帐并提交Genesis块将是一个原子操作
//从Genesis块中检索到的链ID被视为分类帐ID。
func CreateLedger(genesisBlock *common.Block) (ledger.PeerLedger, error) {
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return nil, ErrLedgerMgmtNotInitialized
	}
	id, err := utils.GetChainIDFromBlock(genesisBlock)
	if err != nil {
		return nil, err
	}

	logger.Infof("Creating ledger [%s] with genesis block", id)
	l, err := ledgerProvider.Create(genesisBlock)
	if err != nil {
		return nil, err
	}
	l = wrapLedger(id, l)
	openedLedgers[id] = l
	logger.Infof("Created ledger [%s] with genesis block", id)
	return l, nil
}

//OpenLedger返回给定ID的分类帐
func OpenLedger(id string) (ledger.PeerLedger, error) {
	logger.Infof("Opening ledger with id = %s", id)
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return nil, ErrLedgerMgmtNotInitialized
	}
	l, ok := openedLedgers[id]
	if ok {
		return nil, ErrLedgerAlreadyOpened
	}
	l, err := ledgerProvider.Open(id)
	if err != nil {
		return nil, err
	}
	l = wrapLedger(id, l)
	openedLedgers[id] = l
	logger.Infof("Opened ledger with id = %s", id)
	return l, nil
}

//GetLedgerIDs returns the ids of the ledgers created
func GetLedgerIDs() ([]string, error) {
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return nil, ErrLedgerMgmtNotInitialized
	}
	return ledgerProvider.List()
}

//关闭关闭所有打开的分类帐和为分类帐管理保留的任何资源
func Close() {
	logger.Infof("Closing ledger mgmt")
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return
	}
	for _, l := range openedLedgers {
		l.(*closableLedger).closeWithoutLock()
	}
	ledgerProvider.Close()
	openedLedgers = nil
	logger.Infof("ledger mgmt closed")
}

func wrapLedger(id string, l ledger.PeerLedger) ledger.PeerLedger {
	return &closableLedger{id, l}
}

//Closableledger从实际验证的分类帐扩展并覆盖Close方法
type closableLedger struct {
	id string
	ledger.PeerLedger
}

//关闭关闭实际分类帐并从打开的分类帐映射中删除条目
func (l *closableLedger) Close() {
	lock.Lock()
	defer lock.Unlock()
	l.closeWithoutLock()
}

func (l *closableLedger) closeWithoutLock() {
	l.PeerLedger.Close()
	delete(openedLedgers, l.id)
}

//用于链代码实例化事务的lscc命名空间侦听器（它在“lscc”命名空间中操作数据）
//此代码稍后应移到对等端，并通过ledgermgmt的“initialize”函数传递
func addListenerForCCEventsHandler(
	deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
	stateListeners []ledger.StateListener) []ledger.StateListener {
	return append(stateListeners, &cceventmgmt.KVLedgerLSCCStateListener{DeployedChaincodeInfoProvider: deployedCCInfoProvider})
}

//chaincodeInfoProviderImpl实现接口cceventmgmt.chaincodeInfoProvider
type chaincodeInfoProviderImpl struct {
	pr                     *platforms.Registry
	deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider
}

//GetDeployedChaincodeInfo在接口cceventmgmt.chaincodeInfoProvider中实现函数
func (p *chaincodeInfoProviderImpl) GetDeployedChaincodeInfo(chainid string,
	chaincodeDefinition *cceventmgmt.ChaincodeDefinition) (*ledger.DeployedChaincodeInfo, error) {
	lock.Lock()
	ledger := openedLedgers[chainid]
	lock.Unlock()
	if ledger == nil {
		return nil, errors.Errorf("Ledger not opened [%s]", chainid)
	}
	qe, err := ledger.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()
	deployedChaincodeInfo, err := p.deployedCCInfoProvider.ChaincodeInfo(chaincodeDefinition.Name, qe)
	if err != nil || deployedChaincodeInfo == nil {
		return nil, err
	}
	if deployedChaincodeInfo.Version != chaincodeDefinition.Version ||
		!bytes.Equal(deployedChaincodeInfo.Hash, chaincodeDefinition.Hash) {
//如果已部署的具有给定名称的链码具有不同的版本或哈希，则返回nil
		return nil, nil
	}
	return deployedChaincodeInfo, nil
}

//RetrieveChaincodeArtifacts implements function in the interface cceventmgmt.ChaincodeInfoProvider
func (p *chaincodeInfoProviderImpl) RetrieveChaincodeArtifacts(chaincodeDefinition *cceventmgmt.ChaincodeDefinition) (installed bool, dbArtifactsTar []byte, err error) {
	return ccprovider.ExtractStatedbArtifactsForChaincode(chaincodeDefinition.Name, chaincodeDefinition.Version, p.pr)
}
