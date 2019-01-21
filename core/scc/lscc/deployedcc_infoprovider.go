
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


package lscc

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/pkg/errors"
)

const (
	lsccNamespace = "lscc"
)

//deployedccinfoprovider实现ineterface分类帐。deployedchaincodeinfo提供程序
type DeployedCCInfoProvider struct {
}

//命名空间实现接口ledger.deployedchaincodeinfo提供程序中的函数
func (p *DeployedCCInfoProvider) Namespaces() []string {
	return []string{lsccNamespace}
}

//updatedchaincodes实现接口分类帐中的功能。deployedchaincodeinfo提供程序
func (p *DeployedCCInfoProvider) UpdatedChaincodes(stateUpdates map[string][]*kvrwset.KVWrite) ([]*ledger.ChaincodeLifecycleInfo, error) {
	lsccUpdates := stateUpdates[lsccNamespace]
	lifecycleInfo := []*ledger.ChaincodeLifecycleInfo{}
	updatedCCNames := map[string]bool{}

	for _, kvWrite := range lsccUpdates {
		if kvWrite.IsDelete {
//LSCC命名空间不应有删除
			continue
		}
//chaincode和chaincode集合都有lscc条目。
//我们可以根据collectionseparator的存在来检测集合，
//链码名称中从未存在过。
		if privdata.IsCollectionConfigKey(kvWrite.Key) {
			ccname := privdata.GetCCNameFromCollectionConfigKey(kvWrite.Key)
			updatedCCNames[ccname] = true
			continue
		}
		updatedCCNames[kvWrite.Key] = true
	}

	for updatedCCNames := range updatedCCNames {
		lifecycleInfo = append(lifecycleInfo, &ledger.ChaincodeLifecycleInfo{Name: updatedCCNames})
	}
	return lifecycleInfo, nil
}

//chaincodeinfo在interface ledger.deployedchaincodeinfo提供程序中实现函数
func (p *DeployedCCInfoProvider) ChaincodeInfo(chaincodeName string, qe ledger.SimpleQueryExecutor) (*ledger.DeployedChaincodeInfo, error) {
	chaincodeDataBytes, err := qe.GetState(lsccNamespace, chaincodeName)
	if err != nil || chaincodeDataBytes == nil {
		return nil, err
	}
	chaincodeData := &ccprovider.ChaincodeData{}
	if err := proto.Unmarshal(chaincodeDataBytes, chaincodeData); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling chaincode state data")
	}
	collConfigPkg, err := fetchCollConfigPkg(chaincodeName, qe)
	if err != nil {
		return nil, err
	}
	return &ledger.DeployedChaincodeInfo{
		Name:                chaincodeName,
		Hash:                chaincodeData.Id,
		Version:             chaincodeData.Version,
		CollectionConfigPkg: collConfigPkg,
	}, nil
}

//collectioninfo实现接口分类帐中的功能。deployedchaincodeinfo提供程序
func (p *DeployedCCInfoProvider) CollectionInfo(chaincodeName, collectionName string, qe ledger.SimpleQueryExecutor) (*common.StaticCollectionConfig, error) {
	collConfigPkg, err := fetchCollConfigPkg(chaincodeName, qe)
	if err != nil || collConfigPkg == nil {
		return nil, err
	}
	for _, conf := range collConfigPkg.Config {
		staticCollConfig := conf.GetStaticCollectionConfig()
		if staticCollConfig != nil && staticCollConfig.Name == collectionName {
			return staticCollConfig, nil
		}
	}
	return nil, nil
}

func fetchCollConfigPkg(chaincodeName string, qe ledger.SimpleQueryExecutor) (*common.CollectionConfigPackage, error) {
	collKey := privdata.BuildCollectionKVSKey(chaincodeName)
	collectionConfigPkgBytes, err := qe.GetState(lsccNamespace, collKey)
	if err != nil || collectionConfigPkgBytes == nil {
		return nil, err
	}
	collectionConfigPkg := &common.CollectionConfigPackage{}
	if err := proto.Unmarshal(collectionConfigPkgBytes, collectionConfigPkg); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling chaincode collection config pkg")
	}
	return collectionConfigPkg, nil
}
