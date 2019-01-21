
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


package chaincode

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/policies/inquire"
	common2 "github.com/hyperledger/fabric/protos/common"
)

var logger = flogging.MustGetLogger("discovery.DiscoverySupport")

type MetadataRetriever interface {
	Metadata(channel string, cc string, loadCollections bool) *chaincode.Metadata
}

//DiscoverySupport实现用于服务发现的支持
//与链码有关
type DiscoverySupport struct {
	ci MetadataRetriever
}

//新建DiscoverySupport创建新的DiscoverySupport
func NewDiscoverySupport(ci MetadataRetriever) *DiscoverySupport {
	s := &DiscoverySupport{
		ci: ci,
	}
	return s
}

func (s *DiscoverySupport) PolicyByChaincode(channel string, cc string) policies.InquireablePolicy {
	chaincodeData := s.ci.Metadata(channel, cc, false)
	if chaincodeData == nil {
		logger.Info("Chaincode", cc, "wasn't found")
		return nil
	}
	pol := &common2.SignaturePolicyEnvelope{}
	if err := proto.Unmarshal(chaincodeData.Policy, pol); err != nil {
		logger.Warning("Failed unmarshaling policy for chaincode", cc, ":", err)
		return nil
	}
	if len(pol.Identities) == 0 || pol.Rule == nil {
		logger.Warningf("Invalid policy, either Identities(%v) or Rule(%v) are empty:", pol.Identities, pol.Rule)
		return nil
	}
	return inquire.NewInquireableSignaturePolicy(pol)
}
