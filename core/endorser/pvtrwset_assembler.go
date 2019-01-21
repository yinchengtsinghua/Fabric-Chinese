
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *
 *版权所有IBM公司。保留所有权利。
 *
 *SPDX许可证标识符：Apache-2.0
 */
 *
 **/


package endorser

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
)

//pvtrwsetassembler组装用于分发的专用读写集
//如果需要，可添加其他信息
type PvtRWSetAssembler interface {
//assemblepvtrwset为分发准备txpvtreadwriteset
//使用configinfo将其添加到txpvtreadwritesets中
//有关可用集合配置的信息
//到专用读写集
	AssemblePvtRWSet(privData *rwset.TxPvtReadWriteSet, txsim CollectionConfigRetriever) (*transientstore.TxPvtReadWriteSetWithConfigInfo, error)
}

//collectionconfigretriever封装了ledger.txsimulator的子功能
//提取所需的最小函数集
type CollectionConfigRetriever interface {
//GetState获取给定命名空间和键的值。对于chaincode，命名空间对应于chaincodeid
	GetState(namespace string, key string) ([]byte, error)
}

type rwSetAssembler struct {
}

//assemblepvtrwset为分发准备txpvtreadwriteset
//使用configinfo将其添加到txpvtreadwritesets中
//有关可用集合配置的信息
//到专用读写集
func (as *rwSetAssembler) AssemblePvtRWSet(privData *rwset.TxPvtReadWriteSet, txsim CollectionConfigRetriever) (*transientstore.TxPvtReadWriteSetWithConfigInfo, error) {
	txPvtRwSetWithConfig := &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset:          privData,
		CollectionConfigs: make(map[string]*common.CollectionConfigPackage),
	}

	for _, pvtRwset := range privData.NsPvtRwset {
		namespace := pvtRwset.Namespace
		if _, found := txPvtRwSetWithConfig.CollectionConfigs[namespace]; !found {
			cb, err := txsim.GetState("lscc", privdata.BuildCollectionKVSKey(namespace))
			if err != nil {
				return nil, errors.WithMessage(err, fmt.Sprintf("error while retrieving collection config for chaincode %#v", namespace))
			}
			if cb == nil {
				return nil, errors.New(fmt.Sprintf("no collection config for chaincode %#v", namespace))
			}

			colCP := &common.CollectionConfigPackage{}
			err = proto.Unmarshal(cb, colCP)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid configuration for collection criteria %#v", namespace)
			}

			txPvtRwSetWithConfig.CollectionConfigs[namespace] = colCP
		}
	}
	as.trimCollectionConfigs(txPvtRwSetWithConfig)
	return txPvtRwSetWithConfig, nil
}

func (as *rwSetAssembler) trimCollectionConfigs(pvtData *transientstore.TxPvtReadWriteSetWithConfigInfo) {
	flags := make(map[string]map[string]struct{})
	for _, pvtRWset := range pvtData.PvtRwset.NsPvtRwset {
		namespace := pvtRWset.Namespace
		for _, col := range pvtRWset.CollectionPvtRwset {
			if _, found := flags[namespace]; !found {
				flags[namespace] = make(map[string]struct{})
			}
			flags[namespace][col.CollectionName] = struct{}{}
		}
	}

	filteredConfigs := make(map[string]*common.CollectionConfigPackage)
	for namespace, configs := range pvtData.CollectionConfigs {
		filteredConfigs[namespace] = &common.CollectionConfigPackage{}
		for _, conf := range configs.Config {
			if colConf := conf.GetStaticCollectionConfig(); colConf != nil {
				if _, found := flags[namespace][colConf.Name]; found {
					filteredConfigs[namespace].Config = append(filteredConfigs[namespace].Config, conf)
				}
			}
		}
	}
	pvtData.CollectionConfigs = filteredConfigs
}
