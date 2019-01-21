
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


package privdata

import (
	"errors"
	"testing"

	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/gossip/api"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	gossip2 "github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type collectionAccessFactoryMock struct {
	mock.Mock
}

func (mock *collectionAccessFactoryMock) AccessPolicy(config *common.CollectionConfig, chainID string) (privdata.CollectionAccessPolicy, error) {
	res := mock.Called(config, chainID)
	return res.Get(0).(privdata.CollectionAccessPolicy), res.Error(1)
}

type collectionAccessPolicyMock struct {
	mock.Mock
}

func (mock *collectionAccessPolicyMock) AccessFilter() privdata.Filter {
	args := mock.Called()
	return args.Get(0).(privdata.Filter)
}

func (mock *collectionAccessPolicyMock) RequiredPeerCount() int {
	args := mock.Called()
	return args.Int(0)
}

func (mock *collectionAccessPolicyMock) MaximumPeerCount() int {
	args := mock.Called()
	return args.Int(0)
}

func (mock *collectionAccessPolicyMock) MemberOrgs() []string {
	args := mock.Called()
	return args.Get(0).([]string)
}

func (mock *collectionAccessPolicyMock) IsMemberOnlyRead() bool {
	args := mock.Called()
	return args.Get(0).(bool)
}

func (mock *collectionAccessPolicyMock) Setup(requiredPeerCount int, maxPeerCount int,
	accessFilter privdata.Filter, orgs []string, memberOnlyRead bool) {
	mock.On("AccessFilter").Return(accessFilter)
	mock.On("RequiredPeerCount").Return(requiredPeerCount)
	mock.On("MaximumPeerCount").Return(maxPeerCount)
	mock.On("MemberOrgs").Return(orgs)
	mock.On("IsMemberOnlyRead").Return(memberOnlyRead)
}

type gossipMock struct {
	err error
	mock.Mock
	api.PeerSignature
}

func (g *gossipMock) SendByCriteria(message *proto.SignedGossipMessage, criteria gossip2.SendCriteria) error {
	args := g.Called(message, criteria)
	if args.Get(0) != nil {
		return args.Get(0).(error)
	}
	return nil
}

func (g *gossipMock) PeerFilter(channel gcommon.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error) {
	if g.err != nil {
		return nil, g.err
	}
	return func(member discovery.NetworkMember) bool {
		return messagePredicate(g.PeerSignature)
	}, nil
}

func TestDistributor(t *testing.T) {
	g := &gossipMock{
		Mock: mock.Mock{},
		PeerSignature: api.PeerSignature{
			Signature:    []byte{3, 4, 5},
			Message:      []byte{6, 7, 8},
			PeerIdentity: []byte{0, 1, 2},
		},
	}
	sendings := make(chan struct {
		*proto.PrivatePayload
		gossip2.SendCriteria
	}, 8)
	g.On("SendByCriteria", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		msg := args.Get(0).(*proto.SignedGossipMessage)
		sendCriteria := args.Get(1).(gossip2.SendCriteria)
		sendings <- struct {
			*proto.PrivatePayload
			gossip2.SendCriteria
		}{
			PrivatePayload: msg.GetPrivateData().Payload,
			SendCriteria:   sendCriteria,
		}
	}).Return(nil)
	accessFactoryMock := &collectionAccessFactoryMock{}
	c1ColConfig := &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              "c1",
				RequiredPeerCount: 1,
				MaximumPeerCount:  1,
			},
		},
	}

	c2ColConfig := &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              "c2",
				RequiredPeerCount: 1,
				MaximumPeerCount:  1,
			},
		},
	}

	policyMock := &collectionAccessPolicyMock{}
	policyMock.Setup(1, 2, func(_ common.SignedData) bool {
		return true
	}, []string{"org1", "org2"}, false)
	accessFactoryMock.On("AccessPolicy", c1ColConfig, "test").Return(policyMock, nil)
	accessFactoryMock.On("AccessPolicy", c2ColConfig, "test").Return(policyMock, nil)

	d := NewDistributor("test", g, accessFactoryMock)
	pdFactory := &pvtDataFactory{}
	pvtData := pdFactory.addRWSet().addNSRWSet("ns1", "c1", "c2").addRWSet().addNSRWSet("ns2", "c1", "c2").create()
	err := d.Distribute("tx1", &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: pvtData[0].WriteSet,
		CollectionConfigs: map[string]*common.CollectionConfigPackage{
			"ns1": {
				Config: []*common.CollectionConfig{c1ColConfig, c2ColConfig},
			},
		},
	}, 0)
	assert.NoError(t, err)
	err = d.Distribute("tx2", &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: pvtData[1].WriteSet,
		CollectionConfigs: map[string]*common.CollectionConfigPackage{
			"ns2": {
				Config: []*common.CollectionConfig{c1ColConfig, c2ColConfig},
			},
		},
	}, 0)
	assert.NoError(t, err)

	assertACL := func(pp *proto.PrivatePayload, sc gossip2.SendCriteria) {
		eligible := pp.Namespace == "ns1" && pp.CollectionName == "c1"
		eligible = eligible || (pp.Namespace == "ns2" && pp.CollectionName == "c2")
//模拟收集存储返回NS1和C1，或NS2和C2的策略返回为真，无论
//对于网络成员以及任何其他集合和命名空间组合-返回false

//确保MaxPeers是MaxInternalPeers，即2
//minack是mininternalpeers，是1
		assert.Equal(t, 2, sc.MaxPeers)
		assert.Equal(t, 1, sc.MinAck)
	}
	i := 0
	for dis := range sendings {
		assertACL(dis.PrivatePayload, dis.SendCriteria)
		i++
		if i == 4 {
			break
		}
	}
//我们读了8遍之后频道就空了
	assert.Len(t, sendings, 0)

//坏路径：依赖（八卦和其他）不能正常工作
	g.err = errors.New("failed obtaining filter")
	err = d.Distribute("tx1", &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: pvtData[0].WriteSet,
		CollectionConfigs: map[string]*common.CollectionConfigPackage{
			"ns1": {
				Config: []*common.CollectionConfig{c1ColConfig, c2ColConfig},
			},
		},
	}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed obtaining filter")

	g.Mock = mock.Mock{}
	g.On("SendByCriteria", mock.Anything, mock.Anything).Return(errors.New("failed sending"))
	g.err = nil
	err = d.Distribute("tx1", &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: pvtData[0].WriteSet,
		CollectionConfigs: map[string]*common.CollectionConfigPackage{
			"ns1": {
				Config: []*common.CollectionConfig{c1ColConfig, c2ColConfig},
			},
		},
	}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed disseminating 2 out of 2 private RWSets")
}
