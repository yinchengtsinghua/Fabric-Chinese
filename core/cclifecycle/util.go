
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
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/pkg/errors"
)

var (
//AcceptAll返回接受所有元数据的谓词
	AcceptAll ChaincodePredicate = func(cc chaincode.Metadata) bool {
		return true
	}
)

//chaincodePredicate根据其元数据接受或拒绝chaincode
type ChaincodePredicate func(cc chaincode.Metadata) bool

//deployed chaincodes检索给定已部署链代码的元数据
func DeployedChaincodes(q Query, filter ChaincodePredicate, loadCollections bool, chaincodes ...string) (chaincode.MetadataSet, error) {
	defer q.Done()

	var res chaincode.MetadataSet
	for _, cc := range chaincodes {
		data, err := q.GetState("lscc", cc)
		if err != nil {
			Logger.Error("Failed querying lscc namespace:", err)
			return nil, errors.WithStack(err)
		}
		if len(data) == 0 {
			Logger.Info("Chaincode", cc, "isn't instantiated")
			continue
		}
		ccInfo, err := extractCCInfo(data)
		if err != nil {
			Logger.Error("Failed extracting chaincode info about", cc, "from LSCC returned payload. Error:", err)
			continue
		}
		if ccInfo.Name != cc {
			Logger.Error("Chaincode", cc, "is listed in LSCC as", ccInfo.Name)
			continue
		}

		instCC := chaincode.Metadata{
			Name:    ccInfo.Name,
			Version: ccInfo.Version,
			Id:      ccInfo.Id,
			Policy:  ccInfo.Policy,
		}

		if !filter(instCC) {
			Logger.Debug("Filtered out", instCC)
			continue
		}

		if loadCollections {
			key := privdata.BuildCollectionKVSKey(cc)
			collectionData, err := q.GetState("lscc", key)
			if err != nil {
				Logger.Errorf("Failed querying lscc namespace for %s: %v", key, err)
				return nil, errors.WithStack(err)
			}
			instCC.CollectionsConfig = collectionData
			Logger.Debug("Retrieved collection config for", cc, "from", key)
		}

		res = append(res, instCC)
	}
	Logger.Debug("Returning", res)
	return res, nil
}

func deployedCCToNameVersion(cc chaincode.Metadata) nameVersion {
	return nameVersion{
		name:    cc.Name,
		version: cc.Version,
	}
}

func extractCCInfo(data []byte) (*ccprovider.ChaincodeData, error) {
	cd := &ccprovider.ChaincodeData{}
	if err := proto.Unmarshal(data, cd); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling lscc read value into ChaincodeData")
	}
	return cd, nil
}

type nameVersion struct {
	name    string
	version string
}

func installedCCToNameVersion(cc chaincode.InstalledChaincode) nameVersion {
	return nameVersion{
		name:    cc.Name,
		version: cc.Version,
	}
}

func names(installedChaincodes []chaincode.InstalledChaincode) []string {
	var ccs []string
	for _, cc := range installedChaincodes {
		ccs = append(ccs, cc.Name)
	}
	return ccs
}
