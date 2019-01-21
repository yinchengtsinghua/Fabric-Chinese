
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


package privdata

import (
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"

	util2 "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/ledger"
	privdatacommon "github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/protos/common"
	gossip2 "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	reconcileSleepIntervalConfigKey = "peer.gossip.pvtData.reconcileSleepInterval"
	reconcileSleepIntervalDefault   = time.Minute * 1
	reconcileBatchSizeConfigKey     = "peer.gossip.pvtData.reconcileBatchSize"
	reconcileBatchSizeDefault       = 10
	reconciliationEnabledConfigKey  = "peer.gossip.pvtData.reconciliationEnabled"
)

//定义要获取的API的ReconciliationFetcher接口
//必须协调的私有数据元素
type ReconciliationFetcher interface {
	FetchReconciledItems(dig2collectionConfig privdatacommon.Dig2CollectionConfig) (*privdatacommon.FetchedPvtDataContainer, error)
}

//去：生成mokery-dir。-名称和解获取器-案例下划线-输出模拟/
//go:generate mokery-dir../../core/ledger/-name missingpvdtatatracker-case underline-output mocks/
//go:generate mokery-dir../../core/ledger/-name confighistoryretriever-case underline-output mocks/

//协调器完成提交期间不可用的私有数据的缺失部分。
//这是通过从分类帐中获取丢失的私有数据列表并从其他同行中提取该列表来完成的。
type PvtDataReconciler interface {
//start函数根据调度程序启动协调器，如创建协调器时配置的那样。
	Start()
//停止功能停止调节器
	Stop()
}

type Reconciler struct {
	config *ReconcilerConfig
	ReconciliationFetcher
	committer.Committer
	stopChan  chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

//要使用的noopecciler非功能调节器
//如果已禁用对帐
type NoOpReconciler struct {
}

func (*NoOpReconciler) Start() {
//什么也不做
	logger.Debug("Private data reconciliation has been disabled")
}

func (*NoOpReconciler) Stop() {
//什么也不做
}

//reconcilerconfig保存从core.yaml读取的配置标志
type ReconcilerConfig struct {
	sleepInterval time.Duration
	batchSize     int
	IsEnabled     bool
}

//此func从core.yaml读取协调器配置值并返回reconcilerconfig
func GetReconcilerConfig() *ReconcilerConfig {
	reconcileSleepInterval := viper.GetDuration(reconcileSleepIntervalConfigKey)
	if reconcileSleepInterval == 0 {
		logger.Warning("Configuration key", reconcileSleepIntervalConfigKey, "isn't set, defaulting to", reconcileSleepIntervalDefault)
		reconcileSleepInterval = reconcileSleepIntervalDefault
	}
	reconcileBatchSize := viper.GetInt(reconcileBatchSizeConfigKey)
	if reconcileBatchSize == 0 {
		logger.Warning("Configuration key", reconcileBatchSizeConfigKey, "isn't set, defaulting to", reconcileBatchSizeDefault)
		reconcileBatchSize = reconcileBatchSizeDefault
	}
	isEnabled := viper.GetBool(reconciliationEnabledConfigKey)
	return &ReconcilerConfig{sleepInterval: reconcileSleepInterval, batchSize: reconcileBatchSize, IsEnabled: isEnabled}
}

//NewReconciler创建Reconciler的新实例
func NewReconciler(c committer.Committer, fetcher ReconciliationFetcher, config *ReconcilerConfig) *Reconciler {
	logger.Debug("Private data reconciliation is enabled")
	return &Reconciler{
		config:                config,
		Committer:             c,
		ReconciliationFetcher: fetcher,
		stopChan:              make(chan struct{}),
	}
}

func (r *Reconciler) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopChan)
	})
}

func (r *Reconciler) Start() {
	r.startOnce.Do(func() {
		go r.run()
	})
}

func (r *Reconciler) run() {
	for {
		select {
		case <-r.stopChan:
			return
		case <-time.After(r.config.sleepInterval):
			logger.Debug("Start reconcile missing private info")
			if err := r.reconcile(); err != nil {
				logger.Error("Failed to reconcile missing private info, error: ", err.Error())
				break
			}
		}
	}
}

//返回已协调的项数、MinBlock、MaxBlock（块范围）和错误
func (r *Reconciler) reconcile() error {
	missingPvtDataTracker, err := r.GetMissingPvtDataTracker()
	if err != nil {
		logger.Error("reconciliation error when trying to get missingPvtDataTracker:", err)
		return err
	}
	if missingPvtDataTracker == nil {
		logger.Error("got nil as MissingPvtDataTracker, exiting...")
		return errors.New("got nil as MissingPvtDataTracker, exiting...")
	}
	totalReconciled, minBlock, maxBlock := 0, uint64(math.MaxUint64), uint64(0)

	for {
		missingPvtDataInfo, err := missingPvtDataTracker.GetMissingPvtDataInfoForMostRecentBlocks(r.config.batchSize)
		if err != nil {
			logger.Error("reconciliation error when trying to get missing pvt data info recent blocks:", err)
			return err
		}
//如果MissingPvtDataInfo为零，Len将返回0
		if len(missingPvtDataInfo) == 0 {
			if totalReconciled > 0 {
				logger.Infof("Reconciliation cycle finished successfully. reconciled %d private data keys from blocks range [%d - %d]", totalReconciled, minBlock, maxBlock)
			} else {
				logger.Debug("Reconciliation cycle finished successfully. no items to reconcile")
			}
			return nil
		}

		logger.Debug("got from ledger", len(missingPvtDataInfo), "blocks with missing private data, trying to reconcile...")

		dig2collectionCfg, minB, maxB := r.getDig2CollectionConfig(missingPvtDataInfo)
		fetchedData, err := r.FetchReconciledItems(dig2collectionCfg)
		if err != nil {
			logger.Error("reconciliation error when trying to fetch missing items from different peers:", err)
			return err
		}
		if len(fetchedData.AvailableElements) == 0 {
			logger.Warning("missing private data is not available on other peers")
			return nil
		}

		pvtDataToCommit := r.preparePvtDataToCommit(fetchedData.AvailableElements)
//提交已协调和日志不匹配的丢失的私有数据
		pvtdataHashMismatch, err := r.CommitPvtDataOfOldBlocks(pvtDataToCommit)
		if err != nil {
			return errors.Wrap(err, "failed to commit private data")
		}
		r.logMismatched(pvtdataHashMismatch)
		if minB < minBlock {
			minBlock = minB
		}
		if maxB > maxBlock {
			maxBlock = maxB
		}
		totalReconciled += len(fetchedData.AvailableElements)
	}
}

type collectionConfigKey struct {
	chaincodeName, collectionName string
	blockNum                      uint64
}

func (r *Reconciler) getDig2CollectionConfig(missingPvtDataInfo ledger.MissingPvtDataInfo) (privdatacommon.Dig2CollectionConfig, uint64, uint64) {
	var minBlock, maxBlock uint64
	minBlock = math.MaxUint64
	maxBlock = 0
	collectionConfigCache := make(map[collectionConfigKey]*common.StaticCollectionConfig)
	dig2collectionCfg := make(map[privdatacommon.DigKey]*common.StaticCollectionConfig)
	for blockNum, blockPvtDataInfo := range missingPvtDataInfo {
		if blockNum < minBlock {
			minBlock = blockNum
		}
		if blockNum > maxBlock {
			maxBlock = blockNum
		}
		for seqInBlock, collectionPvtDataInfo := range blockPvtDataInfo {
			for _, pvtDataInfo := range collectionPvtDataInfo {
				collConfigKey := collectionConfigKey{
					chaincodeName:  pvtDataInfo.Namespace,
					collectionName: pvtDataInfo.Collection,
					blockNum:       blockNum,
				}
				if _, exists := collectionConfigCache[collConfigKey]; !exists {
					collectionConfig, err := r.getMostRecentCollectionConfig(pvtDataInfo.Namespace, pvtDataInfo.Collection, blockNum)
					if err != nil {
						logger.Debug(err)
						continue
					}
					collectionConfigCache[collConfigKey] = collectionConfig
				}
				digKey := privdatacommon.DigKey{
					SeqInBlock: seqInBlock,
					Collection: pvtDataInfo.Collection,
					Namespace:  pvtDataInfo.Namespace,
					BlockSeq:   blockNum,
				}
				dig2collectionCfg[digKey] = collectionConfigCache[collConfigKey]
			}
		}
	}
	return dig2collectionCfg, minBlock, maxBlock
}

func (r *Reconciler) getMostRecentCollectionConfig(chaincodeName string, collectionName string, blockNum uint64) (*common.StaticCollectionConfig, error) {
	configHistoryRetriever, err := r.GetConfigHistoryRetriever()
	if err != nil {
		return nil, errors.Wrap(err, "configHistoryRetriever is not available")
	}

	configInfo, err := configHistoryRetriever.MostRecentCollectionConfigBelow(blockNum, chaincodeName)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("cannot find recent collection config update below block sequence = %d for chaincode %s", blockNum, chaincodeName))
	}
	if configInfo == nil {
		return nil, errors.New(fmt.Sprintf("no collection config update below block sequence = %d for chaincode %s is available", blockNum, chaincodeName))
	}

	collectionConfig := extractCollectionConfig(configInfo.CollectionConfig, collectionName)
	if collectionConfig == nil {
		return nil, errors.New(fmt.Sprintf("no collection config was found for collection %s for chaincode %s", collectionName, chaincodeName))
	}

	staticCollectionConfig, wasCastingSuccessful := collectionConfig.Payload.(*common.CollectionConfig_StaticCollectionConfig)
	if !wasCastingSuccessful {
		return nil, errors.New(fmt.Sprintf("expected collection config of type CollectionConfig_StaticCollectionConfig for collection %s for chaincode %s, while got different config type...", collectionName, chaincodeName))
	}
	return staticCollectionConfig.StaticCollectionConfig, nil
}

func (r *Reconciler) preparePvtDataToCommit(elements []*gossip2.PvtDataElement) []*ledger.BlockPvtData {
	rwSetByBlockByKeys := r.groupRwsetByBlock(elements)

//填充传递到分类帐的私有RWset
	var pvtDataToCommit []*ledger.BlockPvtData

	for blockNum, rwSetKeys := range rwSetByBlockByKeys {
		blockPvtData := &ledger.BlockPvtData{
			BlockNum:  blockNum,
			WriteSets: make(map[uint64]*ledger.TxPvtData),
		}
		for seqInBlock, nsRWS := range rwSetKeys.bySeqsInBlock() {
			rwsets := nsRWS.toRWSet()
			logger.Debugf("Preparing to commit [%d] private write set, missed from transaction index [%d] of block number [%d]", len(rwsets.NsPvtRwset), seqInBlock, blockNum)
			blockPvtData.WriteSets[seqInBlock] = &ledger.TxPvtData{
				SeqInBlock: seqInBlock,
				WriteSet:   rwsets,
			}
		}
		pvtDataToCommit = append(pvtDataToCommit, blockPvtData)
	}
	return pvtDataToCommit
}

func (r *Reconciler) logMismatched(pvtdataMismatched []*ledger.PvtdataHashMismatch) {
	if len(pvtdataMismatched) > 0 {
		for _, hashMismatch := range pvtdataMismatched {
			logger.Warningf("failed to reconcile pvtdata chaincode %s, collection %s, block num %d, tx num %d due to hash mismatch",
				hashMismatch.Namespace, hashMismatch.Collection, hashMismatch.BlockNum, hashMismatch.TxNum)
		}
	}
}

//返回从block num到rwsetbykeys的映射
func (r *Reconciler) groupRwsetByBlock(elements []*gossip2.PvtDataElement) map[uint64]rwsetByKeys {
rwSetByBlockByKeys := make(map[uint64]rwsetByKeys) //从块编号映射到rwsetbykeys

//迭代从对等端获取的数据
	for _, element := range elements {
		dig := element.Digest
		if _, exists := rwSetByBlockByKeys[dig.BlockSeq]; !exists {
			rwSetByBlockByKeys[dig.BlockSeq] = make(map[rwSetKey][]byte)
		}
		for _, rws := range element.Payload {
			hash := hex.EncodeToString(util2.ComputeSHA256(rws))
			key := rwSetKey{
				txID:       dig.TxId,
				namespace:  dig.Namespace,
				collection: dig.Collection,
				seqInBlock: dig.SeqInBlock,
				hash:       hash,
			}
			rwSetByBlockByKeys[dig.BlockSeq][key] = rws
		}
	}
	return rwSetByBlockByKeys
}
