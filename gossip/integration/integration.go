
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


package integration

import (
	"net"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

//此文件用于引导八卦实例和/或领导人选举服务实例

func newConfig(selfEndpoint string, externalEndpoint string, certs *common.TLSCertificates, bootPeers ...string) (*gossip.Config, error) {
	_, p, err := net.SplitHostPort(selfEndpoint)

	if err != nil {
		return nil, errors.Wrapf(err, "misconfigured endpoint %s", selfEndpoint)
	}

	port, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "misconfigured endpoint %s, failed to parse port number", selfEndpoint)
	}

	conf := &gossip.Config{
		BindPort:                   int(port),
		BootstrapPeers:             bootPeers,
		ID:                         selfEndpoint,
		MaxBlockCountToStore:       util.GetIntOrDefault("peer.gossip.maxBlockCountToStore", 100),
		MaxPropagationBurstLatency: util.GetDurationOrDefault("peer.gossip.maxPropagationBurstLatency", 10*time.Millisecond),
		MaxPropagationBurstSize:    util.GetIntOrDefault("peer.gossip.maxPropagationBurstSize", 10),
		PropagateIterations:        util.GetIntOrDefault("peer.gossip.propagateIterations", 1),
		PropagatePeerNum:           util.GetIntOrDefault("peer.gossip.propagatePeerNum", 3),
		PullInterval:               util.GetDurationOrDefault("peer.gossip.pullInterval", 4*time.Second),
		PullPeerNum:                util.GetIntOrDefault("peer.gossip.pullPeerNum", 3),
		InternalEndpoint:           selfEndpoint,
		ExternalEndpoint:           externalEndpoint,
		PublishCertPeriod:          util.GetDurationOrDefault("peer.gossip.publishCertPeriod", 10*time.Second),
		RequestStateInfoInterval:   util.GetDurationOrDefault("peer.gossip.requestStateInfoInterval", 4*time.Second),
		PublishStateInfoInterval:   util.GetDurationOrDefault("peer.gossip.publishStateInfoInterval", 4*time.Second),
		SkipBlockVerification:      viper.GetBool("peer.gossip.skipBlockVerification"),
		TLSCerts:                   certs,
		TimeForMembershipTracker:   util.GetDurationOrDefault("peer.gossip.membershipTrackerInterval", 5*time.Second),
	}

	return conf, nil
}

//newgossipcomponent创建一个八卦组件，它将自己连接到给定的GRPC服务器上。
func NewGossipComponent(peerIdentity []byte, endpoint string, s *grpc.Server,
	secAdv api.SecurityAdvisor, cryptSvc api.MessageCryptoService,
	secureDialOpts api.PeerSecureDialOpts, certs *common.TLSCertificates, bootPeers ...string) (gossip.Gossip, error) {

	externalEndpoint := viper.GetString("peer.gossip.externalEndpoint")

	conf, err := newConfig(endpoint, externalEndpoint, certs, bootPeers...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	gossipInstance := gossip.NewGossipService(conf, s, secAdv, cryptSvc,
		peerIdentity, secureDialOpts)

	return gossipInstance, nil
}
