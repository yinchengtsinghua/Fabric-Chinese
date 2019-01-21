
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


package api

import (
	"testing"

	"github.com/hyperledger/fabric/gossip/common"
	"github.com/stretchr/testify/assert"
)

func TestPeerIdentitySetByOrg(t *testing.T) {
	p1 := PeerIdentityInfo{
		Organization: OrgIdentityType("ORG1"),
		Identity:     PeerIdentityType("Peer1"),
	}
	p2 := PeerIdentityInfo{
		Organization: OrgIdentityType("ORG2"),
		Identity:     PeerIdentityType("Peer2"),
	}
	is := PeerIdentitySet{
		p1, p2,
	}
	m := is.ByOrg()
	assert.Len(t, m, 2)
	assert.Equal(t, PeerIdentitySet{p1}, m["ORG1"])
	assert.Equal(t, PeerIdentitySet{p2}, m["ORG2"])
}

func TestPeerIdentitySetByID(t *testing.T) {
	p1 := PeerIdentityInfo{
		Organization: OrgIdentityType("ORG1"),
		PKIId:        common.PKIidType("p1"),
	}
	p2 := PeerIdentityInfo{
		Organization: OrgIdentityType("ORG2"),
		PKIId:        common.PKIidType("p2"),
	}
	is := PeerIdentitySet{
		p1, p2,
	}
	assert.Equal(t, map[string]PeerIdentityInfo{
		"p1": p1,
		"p2": p2,
	}, is.ByID())
}
