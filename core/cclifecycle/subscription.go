
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
	"bytes"
	"sync"

	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
)

//订阅通道信息流
//关于进入生命周期的特定渠道
type Subscription struct {
	sync.Mutex
	lc             *Lifecycle
	channel        string
	queryCreator   QueryCreator
	pendingUpdates chan *cceventmgmt.ChaincodeDefinition
}

type depCCsRetriever func(Query, ChaincodePredicate, bool, ...string) (chaincode.MetadataSet, error)

//当通过部署事务部署链代码并且chaicndoe已经
//安装在对等机上。当对等端上安装了已部署的链代码时，也会调用此函数。
func (sub *Subscription) HandleChaincodeDeploy(chaincodeDefinition *cceventmgmt.ChaincodeDefinition, dbArtifactsTar []byte) error {
	Logger.Debug("Channel", sub.channel, "got a new deployment:", chaincodeDefinition)
	sub.pendingUpdates <- chaincodeDefinition
	return nil
}

func (sub *Subscription) processPendingUpdate(ccDef *cceventmgmt.ChaincodeDefinition) {
	query, err := sub.queryCreator.NewQuery()
	if err != nil {
		Logger.Errorf("Failed creating a new query for channel %s: %v", sub.channel, err)
		return
	}
	installedCC := []chaincode.InstalledChaincode{{
		Name:    ccDef.Name,
		Version: ccDef.Version,
		Id:      ccDef.Hash,
	}}
	ccs, err := queryChaincodeDefinitions(query, installedCC, DeployedChaincodes)
	if err != nil {
		Logger.Errorf("Query for channel %s for %v failed with error %v", sub.channel, ccDef, err)
		return
	}
	Logger.Debug("Updating channel", sub.channel, "with", ccs.AsChaincodes())
	sub.lc.updateState(sub.channel, ccs)
	sub.lc.fireChangeListeners(sub.channel)
}

//当chaincode deploy事务或chaincode安装时调用chaincodedeploydone
//
func (sub *Subscription) ChaincodeDeployDone(succeeded bool) {
//
//这是为了防止在状态查询期间获取任何分类帐锁。
//影响分类帐本身调用此方法时保留的锁。
//我们首先锁定，然后获取挂起的更新，以保留订单。
	sub.Lock()
	go func() {
		defer sub.Unlock()
		update := <-sub.pendingUpdates
//如果我们还没有成功部署链码，只需跳过更新
		if !succeeded {
			Logger.Error("Chaincode deploy for", update.Name, "failed")
			return
		}
		sub.processPendingUpdate(update)
	}()
}

func queryChaincodeDefinitions(query Query, ccs []chaincode.InstalledChaincode, deployedCCs depCCsRetriever) (chaincode.MetadataSet, error) {
//从字符串和版本映射到链代码ID
	installedCCsToIDs := make(map[nameVersion][]byte)
//填充地图
	for _, cc := range ccs {
		Logger.Debug("Chaincode", cc, "'s version is", cc.Version, "and Id is", cc.Id)
		installedCCsToIDs[installedCCToNameVersion(cc)] = cc.Id
	}

	filter := func(cc chaincode.Metadata) bool {
		installedID, exists := installedCCsToIDs[deployedCCToNameVersion(cc)]
		if !exists {
			Logger.Debug("Chaincode", cc, "is instantiated but a different version is installed")
			return false
		}
		if !bytes.Equal(installedID, cc.Id) {
			Logger.Debug("ID of chaincode", cc, "on filesystem doesn't match ID in ledger")
			return false
		}
		return true
	}

	return deployedCCs(query, filter, false, names(ccs)...)
}
