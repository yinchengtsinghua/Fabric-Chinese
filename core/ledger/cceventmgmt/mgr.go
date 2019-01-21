
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


package cceventmgmt

import (
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
)

var logger = flogging.MustGetLogger("cceventmgmt")

var mgr *Mgr

//初始化初始化事件管理
func Initialize(ccInfoProvider ChaincodeInfoProvider) {
	initialize(ccInfoProvider)
}

func initialize(ccInfoProvider ChaincodeInfoProvider) {
	mgr = newMgr(ccInfoProvider)
}

//getmgr返回对singleton事件管理器的引用
func GetMgr() *Mgr {
	return mgr
}

//经理概括了与分类账利息相关的事件的重要交互作用。
type Mgr struct {
//rwlock主要用于跨部署事务、链代码安装和通道创建进行同步。
//理想情况下，对等端中的不同服务的设计应该使它们为不同的重要对象公开锁。
//事件，以便在需要时，顶部的代码可以跨同步。然而，由于缺乏此类全系统设计，
//我们将此锁用于上下文使用
	rwlock               sync.RWMutex
	infoProvider         ChaincodeInfoProvider
	ccLifecycleListeners map[string][]ChaincodeLifecycleEventListener
	callbackStatus       *callbackStatus
}

func newMgr(chaincodeInfoProvider ChaincodeInfoProvider) *Mgr {
	return &Mgr{
		infoProvider:         chaincodeInfoProvider,
		ccLifecycleListeners: make(map[string][]ChaincodeLifecycleEventListener),
		callbackStatus:       newCallbackStatus()}
}

//寄存器为给定的LedgerID注册chaincodeLifecycleEventListener
//因为，在创建/打开分类帐实例时应调用“register”
func (m *Mgr) Register(ledgerid string, l ChaincodeLifecycleEventListener) {
//写锁以同步“chaincode安装”操作与分类帐创建/打开
	m.rwlock.Lock()
	defer m.rwlock.Unlock()
	m.ccLifecycleListeners[ledgerid] = append(m.ccLifecycleListeners[ledgerid], l)
}

//当通过部署事务部署链代码时，应调用handlechaincodedeploy。
//'chaincodedefinitions'参数包含块中部署的所有链码
//我们需要存储上一次收到的“chaincodedefinitions”，因为此函数将被调用
//在执行部署事务之后，验证尚未提交到分类帐。此外，我们
//完成此功能后释放读取锁定。当可能发生“chaincode install”时，这会留下一个小窗口。
//在提交部署事务之前，因此函数“handlechaincodeinstall”可能会错过查找
//已部署的链代码。因此，在函数“handlechaincodeinstall”中，我们显式检查是否部署了链代码
//在存储的“chaincodedefinitions”中
func (m *Mgr) HandleChaincodeDeploy(chainid string, chaincodeDefinitions []*ChaincodeDefinition) error {
	logger.Debugf("Channel [%s]: Handling chaincode deploy event for chaincode [%s]", chainid, chaincodeDefinitions)
//读取锁定以允许在多个通道上并发部署，但同步并发的“chaincode install”操作
	m.rwlock.RLock()
	for _, chaincodeDefinition := range chaincodeDefinitions {
		installed, dbArtifacts, err := m.infoProvider.RetrieveChaincodeArtifacts(chaincodeDefinition)
		if err != nil {
			return err
		}
		if !installed {
			logger.Infof("Channel [%s]: Chaincode [%s] is not installed hence no need to create chaincode artifacts for endorsement",
				chainid, chaincodeDefinition)
			continue
		}
		m.callbackStatus.setDeployPending(chainid)
		if err := m.invokeHandler(chainid, chaincodeDefinition, dbArtifacts); err != nil {
			logger.Warningf("Channel [%s]: Error while invoking a listener for handling chaincode install event: %s", chainid, err)
			return err
		}
		logger.Debugf("Channel [%s]: Handled chaincode deploy event for chaincode [%s]", chainid, chaincodeDefinitions)
	}
	return nil
}

//提交部署事务状态时，应调用ChaincodeDeploydOne。
func (m *Mgr) ChaincodeDeployDone(chainid string) {
//释放“handlechaincodedeploy”函数中的锁
	defer m.rwlock.RUnlock()
	if m.callbackStatus.isDeployPending(chainid) {
		m.invokeDoneOnHandlers(chainid, true)
		m.callbackStatus.unsetDeployPending(chainid)
	}
}

//handlechaincodeinstall应该在chaincode包的安装过程中被调用。
func (m *Mgr) HandleChaincodeInstall(chaincodeDefinition *ChaincodeDefinition, dbArtifacts []byte) error {
	logger.Debugf("HandleChaincodeInstall() - chaincodeDefinition=%#v", chaincodeDefinition)
//写锁防止并发部署操作
	m.rwlock.Lock()
	for chainid := range m.ccLifecycleListeners {
		logger.Debugf("Channel [%s]: Handling chaincode install event for chaincode [%s]", chainid, chaincodeDefinition)
		var deployedCCInfo *ledger.DeployedChaincodeInfo
		var err error
		if deployedCCInfo, err = m.infoProvider.GetDeployedChaincodeInfo(chainid, chaincodeDefinition); err != nil {
			logger.Warningf("Channel [%s]: Error while getting the deployment status of chaincode: %s", chainid, err)
			return err
		}
		if deployedCCInfo == nil {
			logger.Debugf("Channel [%s]: Chaincode [%s] is not deployed on channel hence not creating chaincode artifacts.",
				chainid, chaincodeDefinition)
			continue
		}
		m.callbackStatus.setInstallPending(chainid)
		chaincodeDefinition.CollectionConfigs = deployedCCInfo.CollectionConfigPkg
		if err := m.invokeHandler(chainid, chaincodeDefinition, dbArtifacts); err != nil {
			logger.Warningf("Channel [%s]: Error while invoking a listener for handling chaincode install event: %s", chainid, err)
			return err
		}
		logger.Debugf("Channel [%s]: Handled chaincode install event for chaincode [%s]", chainid, chaincodeDefinition)
	}
	return nil
}

//当chaincode安装完成时，应调用chaincodeinstalldone。
func (m *Mgr) ChaincodeInstallDone(succeeded bool) {
//释放在函数“handlechaincodeinstall”中获取的锁
	defer m.rwlock.Unlock()
	for chainid := range m.callbackStatus.installPending {
		m.invokeDoneOnHandlers(chainid, succeeded)
		m.callbackStatus.unsetInstallPending(chainid)
	}
}

func (m *Mgr) invokeHandler(chainid string, chaincodeDefinition *ChaincodeDefinition, dbArtifactsTar []byte) error {
	listeners := m.ccLifecycleListeners[chainid]
	for _, listener := range listeners {
		if err := listener.HandleChaincodeDeploy(chaincodeDefinition, dbArtifactsTar); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mgr) invokeDoneOnHandlers(chainid string, succeeded bool) {
	listeners := m.ccLifecycleListeners[chainid]
	for _, listener := range listeners {
		listener.ChaincodeDeployDone(succeeded)
	}
}

type callbackStatus struct {
	l              sync.Mutex
	deployPending  map[string]bool
	installPending map[string]bool
}

func newCallbackStatus() *callbackStatus {
	return &callbackStatus{
		deployPending:  make(map[string]bool),
		installPending: make(map[string]bool)}
}

func (s *callbackStatus) setDeployPending(channelID string) {
	s.l.Lock()
	defer s.l.Unlock()
	s.deployPending[channelID] = true
}

func (s *callbackStatus) unsetDeployPending(channelID string) {
	s.l.Lock()
	defer s.l.Unlock()
	delete(s.deployPending, channelID)
}

func (s *callbackStatus) isDeployPending(channelID string) bool {
	s.l.Lock()
	defer s.l.Unlock()
	return s.deployPending[channelID]
}

func (s *callbackStatus) setInstallPending(channelID string) {
	s.l.Lock()
	defer s.l.Unlock()
	s.installPending[channelID] = true
}

func (s *callbackStatus) unsetInstallPending(channelID string) {
	s.l.Lock()
	defer s.l.Unlock()
	delete(s.installPending, channelID)
}

func (s *callbackStatus) isInstallPending(channelID string) bool {
	s.l.Lock()
	defer s.l.Unlock()
	return s.installPending[channelID]
}
