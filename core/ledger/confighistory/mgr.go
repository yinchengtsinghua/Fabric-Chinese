
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


package confighistory

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("confighistory")

const (
collectionConfigNamespace = "lscc" //lscc名称空间是在1.2版中引入的，为了与现有数据兼容，我们继续使用它。
)

//管理器应注册为状态侦听器。状态侦听器构建历史记录，检索器帮助查询历史记录
type Mgr interface {
	ledger.StateListener
	GetRetriever(ledgerID string, ledgerInfoRetriever LedgerInfoRetriever) ledger.ConfigHistoryRetriever
	Close()
}

type mgr struct {
	ccInfoProvider ledger.DeployedChaincodeInfoProvider
	dbProvider     *dbProvider
}

//newmgr构造实现接口“mgr”的实例
func NewMgr(ccInfoProvider ledger.DeployedChaincodeInfoProvider) Mgr {
	return newMgr(ccInfoProvider, dbPath())
}

func newMgr(ccInfoProvider ledger.DeployedChaincodeInfoProvider, dbPath string) Mgr {
	return &mgr{ccInfoProvider, newDBProvider(dbPath)}
}

//InterestedInNamespaces从接口Ledger.StateListener实现函数
func (m *mgr) InterestedInNamespaces() []string {
	return m.ccInfoProvider.Namespaces()
}

//StateCommitDone从接口Ledger.StateListener实现函数
func (m *mgr) StateCommitDone(ledgerID string) {
//诺普
}

//handleStateUpdates从接口ledger.stateListener实现函数
//在此实现中，通过
//ledger.deployedchaincodeinfo提供程序，并作为单独的数据库中的单独条目持久化。
//条目的复合键是一个<blocknum，namespace，key>
func (m *mgr) HandleStateUpdates(trigger *ledger.StateUpdateTrigger) error {
	updatedCCs, err := m.ccInfoProvider.UpdatedChaincodes(convertToKVWrites(trigger.StateUpdates))
	if err != nil {
		return err
	}
	if len(updatedCCs) == 0 {
		logger.Errorf("Config history manager is expected to recieve events only if at least one chaincode is updated stateUpdates = %#v",
			trigger.StateUpdates)
		return nil
	}
	updatedCollConfigs := map[string]*common.CollectionConfigPackage{}
	for _, cc := range updatedCCs {
		ccInfo, err := m.ccInfoProvider.ChaincodeInfo(cc.Name, trigger.PostCommitQueryExecutor)
		if err != nil {
			return err
		}
		if ccInfo.CollectionConfigPkg == nil {
			continue
		}
		updatedCollConfigs[ccInfo.Name] = ccInfo.CollectionConfigPkg
	}
	if len(updatedCollConfigs) == 0 {
		return nil
	}
	batch, err := prepareDBBatch(updatedCollConfigs, trigger.CommittingBlockNum)
	if err != nil {
		return err
	}
	dbHandle := m.dbProvider.getDB(trigger.LedgerID)
	return dbHandle.writeBatch(batch, true)
}

//GetRetriever返回给定分类帐ID的“ledger.configHistoryRetriever”的实现。
func (m *mgr) GetRetriever(ledgerID string, ledgerInfoRetriever LedgerInfoRetriever) ledger.ConfigHistoryRetriever {
	return &retriever{dbHandle: m.dbProvider.getDB(ledgerID), ledgerInfoRetriever: ledgerInfoRetriever}
}

//close实现接口“mgr”中的函数
func (m *mgr) Close() {
	m.dbProvider.Close()
}

type retriever struct {
	ledgerInfoRetriever LedgerInfoRetriever
	dbHandle            *db
}

//mostrecentCollectionConfigBelow从接口ledger.configHistoryRetriever实现函数
func (r *retriever) MostRecentCollectionConfigBelow(blockNum uint64, chaincodeName string) (*ledger.CollectionConfigInfo, error) {
	compositeKV, err := r.dbHandle.mostRecentEntryBelow(blockNum, collectionConfigNamespace, constructCollectionConfigKey(chaincodeName))
	if err != nil || compositeKV == nil {
		return nil, err
	}
	return compositeKVToCollectionConfig(compositeKV)
}

//collectionconfigat从接口ledger.configHistoryRetriever实现函数
func (r *retriever) CollectionConfigAt(blockNum uint64, chaincodeName string) (*ledger.CollectionConfigInfo, error) {
	info, err := r.ledgerInfoRetriever.GetBlockchainInfo()
	if err != nil {
		return nil, err
	}
	maxCommittedBlockNum := info.Height - 1
	if maxCommittedBlockNum < blockNum {
		return nil, &ledger.ErrCollectionConfigNotYetAvailable{MaxBlockNumCommitted: maxCommittedBlockNum,
			Msg: fmt.Sprintf("The maximum block number committed [%d] is less than the requested block number [%d]", maxCommittedBlockNum, blockNum)}
	}

	compositeKV, err := r.dbHandle.entryAt(blockNum, collectionConfigNamespace, constructCollectionConfigKey(chaincodeName))
	if err != nil || compositeKV == nil {
		return nil, err
	}
	return compositeKVToCollectionConfig(compositeKV)
}

func prepareDBBatch(chaincodeCollConfigs map[string]*common.CollectionConfigPackage, committingBlockNum uint64) (*batch, error) {
	batch := newBatch()
	for ccName, collConfig := range chaincodeCollConfigs {
		key := constructCollectionConfigKey(ccName)
		var configBytes []byte
		var err error
		if configBytes, err = proto.Marshal(collConfig); err != nil {
			return nil, errors.WithStack(err)
		}
		batch.add(collectionConfigNamespace, key, committingBlockNum, configBytes)
	}
	return batch, nil
}

func compositeKVToCollectionConfig(compositeKV *compositeKV) (*ledger.CollectionConfigInfo, error) {
	conf := &common.CollectionConfigPackage{}
	if err := proto.Unmarshal(compositeKV.value, conf); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling compositeKV to collection config")
	}
	return &ledger.CollectionConfigInfo{CollectionConfig: conf, CommittingBlockNum: compositeKV.blockNum}, nil
}

func constructCollectionConfigKey(chaincodeName string) string {
return chaincodeName + "~collection" //与版本1.2相同的集合配置键，为了与现有数据兼容，我们继续使用它
}

func dbPath() string {
	return ledgerconfig.GetConfigHistoryPath()
}

func convertToKVWrites(stateUpdates ledger.StateUpdates) map[string][]*kvrwset.KVWrite {
	m := map[string][]*kvrwset.KVWrite{}
	for ns, updates := range stateUpdates {
		m[ns] = updates.([]*kvrwset.KVWrite)
	}
	return m
}

//LedgerForeTrier从Ledger中检索相关信息
type LedgerInfoRetriever interface {
	GetBlockchainInfo() (*common.BlockchainInfo, error)
}
