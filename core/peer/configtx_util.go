
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
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
)

//computefullconfig计算给定当前资源束和事务（包含增量）的完整资源配置。
func computeFullConfig(currentConfigBundle *channelconfig.Bundle, channelConfTx *common.Envelope) (*common.Config, error) {
	fullChannelConfigEnv, err := currentConfigBundle.ConfigtxValidator().ProposeConfigUpdate(channelConfTx)
	if err != nil {
		return nil, err
	}
	return fullChannelConfigEnv.Config, nil
}

//TODO最好使序列化/反序列化具有确定性
func serialize(resConfig *common.Config) ([]byte, error) {
	return proto.Marshal(resConfig)
}

func deserialize(serializedConf []byte) (*common.Config, error) {
	conf := &common.Config{}
	if err := proto.Unmarshal(serializedConf, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

//RetrievePersistedChannelConfig从StateDB检索持久化通道配置
func retrievePersistedChannelConfig(ledger ledger.PeerLedger) (*common.Config, error) {
	qe, err := ledger.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()
	return retrievePersistedConf(qe, channelConfigKey)
}
