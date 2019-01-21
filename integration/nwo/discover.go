
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


package nwo

import (
	"encoding/json"

	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

//DiscoveredPeer定义了一个结构，用于使用发现服务发现对等点。
//结果中的每个对等方都将具有这些字段
type DiscoveredPeer struct {
	MSPID      string   `yaml:"mspid,omitempty"`
	Endpoint   string   `yaml:"endpoint,omitempty"`
	Identity   string   `yaml:"identity,omitempty"`
	Chaincodes []string `yaml:"chaincodes,omitempty"`
}

//按照中的指定，使用通道名和用户对对等机运行发现服务命令发现对等机
//函数参数。返回发现的对等点的切片
func DiscoverPeers(n *Network, p *Peer, user, channelName string) func() []DiscoveredPeer {
	return func() []DiscoveredPeer {
		peers := commands.Peers{
			UserCert: n.PeerUserCert(p, user),
			UserKey:  n.PeerUserKey(p, user),
			MSPID:    n.Organization(p.Organization).MSPID,
			Server:   n.PeerAddress(p, ListenPort),
			Channel:  channelName,
		}
		sess, err := n.Discover(peers)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))

		var discovered []DiscoveredPeer
		err = json.Unmarshal(sess.Out.Contents(), &discovered)
		Expect(err).NotTo(HaveOccurred())
		return discovered
	}
}
