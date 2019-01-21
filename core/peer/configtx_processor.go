
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


package peer

import (
	"fmt"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/customtx"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

const (
	channelConfigKey = "resourcesconfigtx.CHANNEL_CONFIG_KEY"
	peerNamespace    = ""
)

//txprocessor实现接口“github.com/hyperledger/fabric/core/ledger/customtx/processor”
type configtxProcessor struct {
}

//new txprocessor构造txprocessor的新实例
func newConfigTxProcessor() customtx.Processor {
	return &configtxProcessor{}
}

//GenerateSimulationResults实现接口“github.com/hyperledger/fabric/core/ledger/customtx/processor”中的功能。
//这一实现过程遵循两种类型的事务。
//配置-只将配置存储在statedb中。此外，如果事务来自Genesis块，则存储资源配置种子。
//对等资源更新-在正常过程中，这将根据当前资源包验证事务，
//计算完整配置，如果发现事务有效，则存储完整配置。
//但是，如果“initializingledger”为真（即，分类账是从Genesis块创建的
//或者分类账正在同步状态与区块链，在启动期间），使用
//statedb的最新配置
func (tp *configtxProcessor) GenerateSimulationResults(txEnv *common.Envelope, simulator ledger.TxSimulator, initializingLedger bool) error {
	payload := utils.UnmarshalPayloadOrPanic(txEnv.Payload)
	channelHdr := utils.UnmarshalChannelHeaderOrPanic(payload.Header.ChannelHeader)
	txType := common.HeaderType(channelHdr.GetType())

	switch txType {
	case common.HeaderType_CONFIG:
		peerLogger.Debugf("Processing CONFIG")
		return processChannelConfigTx(txEnv, simulator)

	default:
		return fmt.Errorf("tx type [%s] is not expected", txType)
	}
}

func processChannelConfigTx(txEnv *common.Envelope, simulator ledger.TxSimulator) error {
	configEnvelope := &common.ConfigEnvelope{}
	if _, err := utils.UnmarshalEnvelopeOfType(txEnv, common.HeaderType_CONFIG, configEnvelope); err != nil {
		return err
	}
	channelConfig := configEnvelope.Config

	if err := persistConf(simulator, channelConfigKey, channelConfig); err != nil {
		return err
	}

	peerLogger.Debugf("channelConfig=%s", channelConfig)
	if channelConfig == nil {
		return fmt.Errorf("Channel config found nil")
	}

	return nil
}

func persistConf(simulator ledger.TxSimulator, key string, config *common.Config) error {
	serializedConfig, err := serialize(config)
	if err != nil {
		return err
	}
	return simulator.SetState(peerNamespace, key, serializedConfig)
}

func retrievePersistedConf(queryExecuter ledger.QueryExecutor, key string) (*common.Config, error) {
	serializedConfig, err := queryExecuter.GetState(peerNamespace, key)
	if err != nil {
		return nil, err
	}
	if serializedConfig == nil {
		return nil, nil
	}
	return deserialize(serializedConfig)
}
