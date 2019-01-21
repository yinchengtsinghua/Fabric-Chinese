
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


package support

import "github.com/hyperledger/fabric/discovery"

//DiscoverySupport聚合了发现服务所需的所有支持
type DiscoverySupport struct {
	discovery.AccessControlSupport
	discovery.GossipSupport
	discovery.EndorsementSupport
	discovery.ConfigSupport
	discovery.ConfigSequenceSupport
}

//NewDiscoverySupport返回聚合发现支持
func NewDiscoverySupport(
	access discovery.AccessControlSupport,
	gossip discovery.GossipSupport,
	endorsement discovery.EndorsementSupport,
	config discovery.ConfigSupport,
	sequence discovery.ConfigSequenceSupport,
) *DiscoverySupport {
	return &DiscoverySupport{
		AccessControlSupport:  access,
		GossipSupport:         gossip,
		EndorsementSupport:    endorsement,
		ConfigSupport:         config,
		ConfigSequenceSupport: sequence,
	}
}
