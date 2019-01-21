
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
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
)

//kvledgerlscstateListener侦听链代码生命周期的状态更改
type KVLedgerLSCCStateListener struct {
	DeployedChaincodeInfoProvider ledger.DeployedChaincodeInfoProvider
}

//handlestateupdates使用“deployedchaincodeinfo提供程序”查找链代码的部署
//并在chaincode事件管理器上调用“handlechaincodedeploy”函数（后者反过来负责创建statedb
//链码状态数据的工件）
func (listener *KVLedgerLSCCStateListener) HandleStateUpdates(trigger *ledger.StateUpdateTrigger) error {
	channelName, kvWrites, postCommitQE, deployCCInfoProvider :=
		trigger.LedgerID, convertToKVWrites(trigger.StateUpdates), trigger.PostCommitQueryExecutor, listener.DeployedChaincodeInfoProvider

	logger.Debugf("Channel [%s]: Handling state updates in LSCC namespace - stateUpdates=%#v", channelName, kvWrites)
	updatedChaincodes, err := deployCCInfoProvider.UpdatedChaincodes(kvWrites)
	if err != nil {
		return err
	}
	chaincodeDefs := []*ChaincodeDefinition{}
	for _, updatedChaincode := range updatedChaincodes {
		logger.Infof("Channel [%s]: Handling deploy or update of chaincode [%s]", channelName, updatedChaincode.Name)
		if updatedChaincode.Deleted {
//TODO在生命周期中实现删除时处理删除案例
			continue
		}
		deployedCCInfo, err := deployCCInfoProvider.ChaincodeInfo(updatedChaincode.Name, postCommitQE)
		if err != nil {
			return err
		}
		chaincodeDefs = append(chaincodeDefs, &ChaincodeDefinition{
			Name:              deployedCCInfo.Name,
			Hash:              deployedCCInfo.Hash,
			Version:           deployedCCInfo.Version,
			CollectionConfigs: deployedCCInfo.CollectionConfigPkg,
		})
	}
	return GetMgr().HandleChaincodeDeploy(channelName, chaincodeDefs)
}

//interestedInNamespaces从接口“ledger.stateListener”实现函数
func (listener *KVLedgerLSCCStateListener) InterestedInNamespaces() []string {
	return listener.DeployedChaincodeInfoProvider.Namespaces()
}

//statecommitdone从接口'ledger.statelistener'实现函数
func (listener *KVLedgerLSCCStateListener) StateCommitDone(channelName string) {
	GetMgr().ChaincodeDeployDone(channelName)
}

func convertToKVWrites(stateUpdates ledger.StateUpdates) map[string][]*kvrwset.KVWrite {
	m := map[string][]*kvrwset.KVWrite{}
	for ns, updates := range stateUpdates {
		m[ns] = updates.([]*kvrwset.KVWrite)
	}
	return m
}
