
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


package kvledger

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
)

//CollelgNotifier侦听链码事件并确定对等方是否符合一个或多个现有的条件
//私有数据收集并通知注册的侦听器
type collElgNotifier struct {
	deployedChaincodeInfoProvider ledger.DeployedChaincodeInfoProvider
	membershipInfoProvider        ledger.MembershipInfoProvider
	listeners                     map[string]collElgListener
}

//InterestedInNamespaces在接口Ledger.StateListener中实现函数
func (n *collElgNotifier) InterestedInNamespaces() []string {
	return n.deployedChaincodeInfoProvider.Namespaces()
}

//handleStateUpdates在接口ledger.stateListener中实现函数
//当一个块部署或升级一个或多个链码时，将调用此函数。
//对于每个升级的链码，此函数执行以下操作
//1）检索现有的收集配置和新的收集配置
//2）根据现有集合配置计算对等方不符合条件的集合
//但符合新集合配置的条件
//最后，它使用一个map ns:colls在分类帐存储上调用函数“processcollseligibilityEnabled”。
//that contains the details of <ns, coll> combination for which the eligibility of the peer is switched on.
func (n *collElgNotifier) HandleStateUpdates(trigger *ledger.StateUpdateTrigger) error {
	nsCollMap := map[string][]string{}
	qe := trigger.CommittedStateQueryExecutor
	postCommitQE := trigger.PostCommitQueryExecutor

	stateUpdates := convertToKVWrites(trigger.StateUpdates)
	ccLifecycleInfo, err := n.deployedChaincodeInfoProvider.UpdatedChaincodes(stateUpdates)
	if err != nil {
		return err
	}
	var existingCCInfo, postCommitCCInfo *ledger.DeployedChaincodeInfo
	for _, ccInfo := range ccLifecycleInfo {
		ledgerid := trigger.LedgerID
		ccName := ccInfo.Name
		if existingCCInfo, err = n.deployedChaincodeInfoProvider.ChaincodeInfo(ccName, qe); err != nil {
			return err
		}
if existingCCInfo == nil { //不是升级事务
			continue
		}
		if postCommitCCInfo, err = n.deployedChaincodeInfoProvider.ChaincodeInfo(ccName, postCommitQE); err != nil {
			return err
		}
		elgEnabledCollNames, err := n.elgEnabledCollNames(
			ledgerid,
			existingCCInfo.CollectionConfigPkg,
			postCommitCCInfo.CollectionConfigPkg,
		)
		if err != nil {
			return err
		}
		logger.Debugf("[%s] collections of chaincode [%s] for which peer was not eligible before and now the eligiblity is enabled - [%s]",
			ledgerid, ccName, elgEnabledCollNames,
		)
		if len(elgEnabledCollNames) > 0 {
			nsCollMap[ccName] = elgEnabledCollNames
		}
	}
	if len(nsCollMap) > 0 {
		n.invokeLedgerSpecificNotifier(trigger.LedgerID, trigger.CommittingBlockNum, nsCollMap)
	}
	return nil
}

func (n *collElgNotifier) registerListener(ledgerID string, listener collElgListener) {
	n.listeners[ledgerID] = listener
}

func (n *collElgNotifier) invokeLedgerSpecificNotifier(ledgerID string, commtingBlk uint64, nsCollMap map[string][]string) {
	listener := n.listeners[ledgerID]
	listener.ProcessCollsEligibilityEnabled(commtingBlk, nsCollMap)
}

//elgenabledcollnames返回对等方不符合“existingpkg”条件且符合“postcommitpkg”条件的集合的名称。
func (n *collElgNotifier) elgEnabledCollNames(ledgerID string,
	existingPkg, postCommitPkg *common.CollectionConfigPackage) ([]string, error) {

	collectionNames := []string{}
	exisingConfs := retrieveCollConfs(existingPkg)
	postCommitConfs := retrieveCollConfs(postCommitPkg)
	existingConfMap := map[string]*common.StaticCollectionConfig{}
	for _, existingConf := range exisingConfs {
		existingConfMap[existingConf.Name] = existingConf
	}

	for _, postCommitConf := range postCommitConfs {
		collName := postCommitConf.Name
		existingConf, ok := existingConfMap[collName]
if !ok { //全新系列
			continue
		}
		membershipEnabled, err := n.elgEnabled(ledgerID, existingConf.MemberOrgsPolicy, postCommitConf.MemberOrgsPolicy)
		if err != nil {
			return nil, err
		}
		if !membershipEnabled {
			continue
		}
//不是现有成员，现在添加
		collectionNames = append(collectionNames, collName)
	}
	return collectionNames, nil
}

//如果对等方不符合“existingpolicy”的收集条件，并且符合“postcommitpolicy”的条件，则elgenaled返回true。
func (n *collElgNotifier) elgEnabled(ledgerID string, existingPolicy, postCommitPolicy *common.CollectionPolicyConfig) (bool, error) {
	existingMember, err := n.membershipInfoProvider.AmMemberOf(ledgerID, existingPolicy)
	if err != nil || existingMember {
		return false, err
	}
	return n.membershipInfoProvider.AmMemberOf(ledgerID, postCommitPolicy)
}

func convertToKVWrites(stateUpdates ledger.StateUpdates) map[string][]*kvrwset.KVWrite {
	m := map[string][]*kvrwset.KVWrite{}
	for ns, updates := range stateUpdates {
		m[ns] = updates.([]*kvrwset.KVWrite)
	}
	return m
}

//StateCommitDone在接口Ledger.StateListener中实现函数
func (n *collElgNotifier) StateCommitDone(ledgerID string) {
//诺普
}

type collElgListener interface {
	ProcessCollsEligibilityEnabled(commitingBlk uint64, nsCollMap map[string][]string) error
}

func retrieveCollConfs(collConfPkg *common.CollectionConfigPackage) []*common.StaticCollectionConfig {
	if collConfPkg == nil {
		return nil
	}
	var staticCollConfs []*common.StaticCollectionConfig
	protoConfArray := collConfPkg.Config
	for _, protoConf := range protoConfArray {
		staticCollConfs = append(staticCollConfs, protoConf.GetStaticCollectionConfig())
	}
	return staticCollConfs
}
