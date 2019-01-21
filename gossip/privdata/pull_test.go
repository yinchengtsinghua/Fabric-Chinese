
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
	"bytes"
	"crypto/rand"
	"sync"
	"testing"

	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	privdatacommon "github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/gossip/privdata/mocks"
	"github.com/hyperledger/fabric/gossip/util"
	fcommon "github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	policy2Filter = make(map[privdata.CollectionAccessPolicy]privdata.Filter)
}

//原匹配器用来测试一片质子等于另一片质子。
//这是必需的，因为一般来说reflect.equal（proto1，proto1）可能不是真的。
func protoMatcher(pvds ...*proto.PvtDataDigest) func([]*proto.PvtDataDigest) bool {
	return func(ipvds []*proto.PvtDataDigest) bool {
		if len(pvds) != len(ipvds) {
			return false
		}

		for i, pvd := range pvds {
			if !pb.Equal(pvd, ipvds[i]) {
				return false
			}
		}

		return true
	}
}

var policyLock sync.Mutex
var policy2Filter map[privdata.CollectionAccessPolicy]privdata.Filter

type mockCollectionStore struct {
	m            map[string]*mockCollectionAccess
	accessFilter privdata.Filter
}

func newCollectionStore() *mockCollectionStore {
	return &mockCollectionStore{
		m:            make(map[string]*mockCollectionAccess),
		accessFilter: nil,
	}
}

func (cs *mockCollectionStore) withPolicy(collection string, btl uint64) *mockCollectionAccess {
	coll := &mockCollectionAccess{cs: cs, btl: btl}
	cs.m[collection] = coll
	return coll
}

func (cs *mockCollectionStore) withAccessFilter(filter privdata.Filter) *mockCollectionStore {
	cs.accessFilter = filter
	return cs
}

func (cs mockCollectionStore) RetrieveCollectionAccessPolicy(cc fcommon.CollectionCriteria) (privdata.CollectionAccessPolicy, error) {
	return cs.m[cc.Collection], nil
}

func (cs mockCollectionStore) RetrieveCollection(fcommon.CollectionCriteria) (privdata.Collection, error) {
	panic("implement me")
}

func (cs mockCollectionStore) RetrieveCollectionConfigPackage(fcommon.CollectionCriteria) (*fcommon.CollectionConfigPackage, error) {
	panic("implement me")
}

func (cs mockCollectionStore) RetrieveCollectionPersistenceConfigs(cc fcommon.CollectionCriteria) (privdata.CollectionPersistenceConfigs, error) {
	return cs.m[cc.Collection], nil
}

func (cs mockCollectionStore) HasReadAccess(cc fcommon.CollectionCriteria, sp *peer.SignedProposal, qe ledger.QueryExecutor) (bool, error) {
	panic("implement me")
}

func (cs mockCollectionStore) AccessFilter(channelName string, collectionPolicyConfig *fcommon.CollectionPolicyConfig) (privdata.Filter, error) {
	if cs.accessFilter != nil {
		return cs.accessFilter, nil
	}
	panic("implement me")
}

type mockCollectionAccess struct {
	cs  *mockCollectionStore
	btl uint64
}

func (mc *mockCollectionAccess) BlockToLive() uint64 {
	return mc.btl
}

func (mc *mockCollectionAccess) thatMapsTo(peers ...string) *mockCollectionStore {
	policyLock.Lock()
	defer policyLock.Unlock()
	policy2Filter[mc] = func(sd fcommon.SignedData) bool {
		for _, peer := range peers {
			if bytes.Equal(sd.Identity, []byte(peer)) {
				return true
			}
		}
		return false
	}
	return mc.cs
}

func (mc *mockCollectionAccess) MemberOrgs() []string {
	return nil
}

func (mc *mockCollectionAccess) AccessFilter() privdata.Filter {
	policyLock.Lock()
	defer policyLock.Unlock()
	return policy2Filter[mc]
}

func (mc *mockCollectionAccess) RequiredPeerCount() int {
	return 0
}

func (mc *mockCollectionAccess) MaximumPeerCount() int {
	return 0
}

func (mc *mockCollectionAccess) IsMemberOnlyRead() bool {
	return false
}

type dataRetrieverMock struct {
	mock.Mock
}

func (dr *dataRetrieverMock) CollectionRWSet(dig []*proto.PvtDataDigest, blockNum uint64) (Dig2PvtRWSetWithConfig, bool, error) {
	args := dr.Called(dig, blockNum)
	return args.Get(0).(Dig2PvtRWSetWithConfig), args.Bool(1), args.Error(2)
}

type receivedMsg struct {
	responseChan chan proto.ReceivedMessage
	*comm.RemotePeer
	*proto.SignedGossipMessage
}

func (msg *receivedMsg) Ack(_ error) {

}

func (msg *receivedMsg) Respond(message *proto.GossipMessage) {
	m, _ := message.NoopSign()
	msg.responseChan <- &receivedMsg{SignedGossipMessage: m, RemotePeer: &comm.RemotePeer{}}
}

func (msg *receivedMsg) GetGossipMessage() *proto.SignedGossipMessage {
	return msg.SignedGossipMessage
}

func (msg *receivedMsg) GetSourceEnvelope() *proto.Envelope {
	panic("implement me")
}

func (msg *receivedMsg) GetConnectionInfo() *proto.ConnectionInfo {
	return &proto.ConnectionInfo{
		Identity: api.PeerIdentityType(msg.RemotePeer.PKIID),
		Auth: &proto.AuthInfo{
			SignedData: []byte{},
			Signature:  []byte{},
		},
	}
}

type mockGossip struct {
	mock.Mock
	msgChan chan proto.ReceivedMessage
	id      *comm.RemotePeer
	network *gossipNetwork
}

func newMockGossip(id *comm.RemotePeer) *mockGossip {
	return &mockGossip{
		msgChan: make(chan proto.ReceivedMessage),
		id:      id,
	}
}

func (g *mockGossip) PeerFilter(channel common.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error) {
	for _, call := range g.Mock.ExpectedCalls {
		if call.Method == "PeerFilter" {
			args := g.Called(channel, messagePredicate)
			if args.Get(1) != nil {
				return nil, args.Get(1).(error)
			}
			return args.Get(0).(filter.RoutingFilter), nil
		}
	}
	return func(member discovery.NetworkMember) bool {
		return messagePredicate(api.PeerSignature{
			PeerIdentity: api.PeerIdentityType(member.PKIid),
		})
	}, nil
}

func (g *mockGossip) Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer) {
	sMsg, _ := msg.NoopSign()
	for _, peer := range g.network.peers {
		if bytes.Equal(peer.id.PKIID, peers[0].PKIID) {
			peer.msgChan <- &receivedMsg{
				RemotePeer:          g.id,
				SignedGossipMessage: sMsg,
				responseChan:        g.msgChan,
			}
			return
		}
	}
}

func (g *mockGossip) PeersOfChannel(common.ChainID) []discovery.NetworkMember {
	return g.Called().Get(0).([]discovery.NetworkMember)
}

func (g *mockGossip) Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage) {
	return nil, g.msgChan
}

type peerData struct {
	id           string
	ledgerHeight uint64
}

func membership(knownPeers ...peerData) []discovery.NetworkMember {
	var peers []discovery.NetworkMember
	for _, peer := range knownPeers {
		peers = append(peers, discovery.NetworkMember{
			Endpoint: peer.id,
			PKIid:    common.PKIidType(peer.id),
			Properties: &proto.Properties{
				LedgerHeight: peer.ledgerHeight,
			},
		})
	}
	return peers
}

type gossipNetwork struct {
	peers []*mockGossip
}

func (gn *gossipNetwork) newPuller(id string, ps privdata.CollectionStore, factory CollectionAccessFactory, knownMembers ...discovery.NetworkMember) *puller {
	g := newMockGossip(&comm.RemotePeer{PKIID: common.PKIidType(id), Endpoint: id})
	g.network = gn
	g.On("PeersOfChannel", mock.Anything).Return(knownMembers)

	p := NewPuller(ps, g, &dataRetrieverMock{}, factory, "A")
	gn.peers = append(gn.peers, g)
	return p
}

func newPRWSet() []util.PrivateRWSet {
	b1 := make([]byte, 10)
	b2 := make([]byte, 10)
	rand.Read(b1)
	rand.Read(b2)
	return []util.PrivateRWSet{util.PrivateRWSet(b1), util.PrivateRWSet(b2)}
}

func TestPullerFromOnly1Peer(t *testing.T) {
	t.Parallel()
//场景：p1从p2拉而不是从p3拉
//并且成功-p1从p2请求（而不是从p3请求！）对于
//预期摘要
	gn := &gossipNetwork{}
	policyStore := newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2")
	factoryMock1 := &collectionAccessFactoryMock{}
	policyMock1 := &collectionAccessPolicyMock{}
	policyMock1.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2"))
	}, []string{"org1", "org2"}, false)
	factoryMock1.On("AccessPolicy", mock.Anything, mock.Anything).Return(policyMock1, nil)
	p1 := gn.newPuller("p1", policyStore, factoryMock1, membership(peerData{"p2", uint64(1)}, peerData{"p3", uint64(1)})...)

	p2TransientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col1",
				},
			},
		},
	}
	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p1")
	factoryMock2 := &collectionAccessFactoryMock{}
	policyMock2 := &collectionAccessPolicyMock{}
	policyMock2.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)
	factoryMock2.On("AccessPolicy", mock.Anything, mock.Anything).Return(policyMock2, nil)

	p2 := gn.newPuller("p2", policyStore, factoryMock2)
	dig := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col1",
		Namespace:  "ns1",
	}

	store := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: p2TransientStore,
	}

	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), uint64(0)).Return(store, true, nil)

	factoryMock3 := &collectionAccessFactoryMock{}
	policyMock3 := &collectionAccessPolicyMock{}
	policyMock3.Setup(1, 2, func(data fcommon.SignedData) bool {
		return false
	}, []string{"org1", "org2"}, false)
	factoryMock3.On("AccessPolicy", mock.Anything, mock.Anything).Return(policyMock3, nil)

	p3 := gn.newPuller("p3", newCollectionStore(), factoryMock3)
	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), uint64(0)).Run(func(_ mock.Arguments) {
		t.Fatal("p3 shouldn't have been selected for pull")
	})

	dasf := &digestsAndSourceFactory{}

	fetchedMessages, err := p1.fetch(dasf.mapDigest(toDigKey(dig)).toSources().create())
	rws1 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[0])
	rws2 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[1])
	fetched := []util.PrivateRWSet{rws1, rws2}
	assert.NoError(t, err)
	assert.Equal(t, p2TransientStore.RWSet, fetched)
}

func TestPullerDataNotAvailable(t *testing.T) {
	t.Parallel()
//场景：p1从p2拉而不是从p3拉
//但是p2中的数据不存在
	gn := &gossipNetwork{}
	policyStore := newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2")
	factoryMock := &collectionAccessFactoryMock{}
	factoryMock.On("AccessPolicy", mock.Anything, mock.Anything).Return(&collectionAccessPolicyMock{}, nil)

	p1 := gn.newPuller("p1", policyStore, factoryMock, membership(peerData{"p2", uint64(1)}, peerData{"p3", uint64(1)})...)

	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p1")
	p2 := gn.newPuller("p2", policyStore, factoryMock)
	dig := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col1",
		Namespace:  "ns1",
	}

	store := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: &util.PrivateRWSetWithConfig{
			RWSet: []util.PrivateRWSet{},
		},
	}

	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), mock.Anything).Return(store, true, nil)

	p3 := gn.newPuller("p3", newCollectionStore(), factoryMock)
	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), mock.Anything).Run(func(_ mock.Arguments) {
		t.Fatal("p3 shouldn't have been selected for pull")
	})

	dasf := &digestsAndSourceFactory{}
	fetchedMessages, err := p1.fetch(dasf.mapDigest(toDigKey(dig)).toSources().create())
	assert.Empty(t, fetchedMessages.AvailableElements)
	assert.NoError(t, err)
}

func TestPullerNoPeersKnown(t *testing.T) {
	t.Parallel()
//场景：P1不知道任何对等点，因此无法获取
	gn := &gossipNetwork{}
	policyStore := newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2", "p3")
	factoryMock := &collectionAccessFactoryMock{}
	factoryMock.On("AccessPolicy", mock.Anything, mock.Anything).Return(&collectionAccessPolicyMock{}, nil)

	p1 := gn.newPuller("p1", policyStore, factoryMock)
	dasf := &digestsAndSourceFactory{}
	d2s := dasf.mapDigest(&privdatacommon.DigKey{Collection: "col1", TxId: "txID1", Namespace: "ns1"}).toSources().create()
	fetchedMessages, err := p1.fetch(d2s)
	assert.Empty(t, fetchedMessages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Empty membership")
}

func TestPullPeerFilterError(t *testing.T) {
	t.Parallel()
//场景：P1尝试获取错误通道
	gn := &gossipNetwork{}
	policyStore := newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2")
	factoryMock := &collectionAccessFactoryMock{}
	factoryMock.On("AccessPolicy", mock.Anything, mock.Anything).Return(&collectionAccessPolicyMock{}, nil)

	p1 := gn.newPuller("p1", policyStore, factoryMock)
	gn.peers[0].On("PeerFilter", mock.Anything, mock.Anything).Return(nil, errors.New("Failed obtaining filter"))
	dasf := &digestsAndSourceFactory{}
	d2s := dasf.mapDigest(&privdatacommon.DigKey{Collection: "col1", TxId: "txID1", Namespace: "ns1"}).toSources().create()
	fetchedMessages, err := p1.fetch(d2s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed obtaining filter")
	assert.Empty(t, fetchedMessages)
}

func TestPullerPeerNotEligible(t *testing.T) {
	t.Parallel()
//场景：p1从p2或p3拉出
//但不能从p2或p3中提取数据
	gn := &gossipNetwork{}
	policyStore := newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2", "p3")
	factoryMock1 := &collectionAccessFactoryMock{}
	accessPolicyMock1 := &collectionAccessPolicyMock{}
	accessPolicyMock1.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2")) || bytes.Equal(data.Identity, []byte("p3"))
	}, []string{"org1", "org2"}, false)
	factoryMock1.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock1, nil)

	p1 := gn.newPuller("p1", policyStore, factoryMock1, membership(peerData{"p2", uint64(1)}, peerData{"p3", uint64(1)})...)

	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2")
	factoryMock2 := &collectionAccessFactoryMock{}
	accessPolicyMock2 := &collectionAccessPolicyMock{}
	accessPolicyMock2.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2"))
	}, []string{"org1", "org2"}, false)
	factoryMock2.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock2, nil)

	p2 := gn.newPuller("p2", policyStore, factoryMock2)

	dig := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col1",
		Namespace:  "ns1",
	}

	store := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: &util.PrivateRWSetWithConfig{
			RWSet: newPRWSet(),
			CollectionConfig: &fcommon.CollectionConfig{
				Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &fcommon.StaticCollectionConfig{
						Name: "col1",
					},
				},
			},
		},
	}

	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), mock.Anything).Return(store, true, nil)

	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p3")
	factoryMock3 := &collectionAccessFactoryMock{}
	accessPolicyMock3 := &collectionAccessPolicyMock{}
	accessPolicyMock3.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p3"))
	}, []string{"org1", "org2"}, false)
	factoryMock3.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock1, nil)

	p3 := gn.newPuller("p3", policyStore, factoryMock3)
	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), mock.Anything).Return(store, true, nil)
	dasf := &digestsAndSourceFactory{}
	d2s := dasf.mapDigest(&privdatacommon.DigKey{Collection: "col1", TxId: "txID1", Namespace: "ns1"}).toSources().create()
	fetchedMessages, err := p1.fetch(d2s)
	assert.Empty(t, fetchedMessages.AvailableElements)
	assert.NoError(t, err)
}

func TestPullerDifferentPeersDifferentCollections(t *testing.T) {
	t.Parallel()
//场景：p1从p2和p3拉出
//每个都有不同的收藏
	gn := &gossipNetwork{}
	factoryMock1 := &collectionAccessFactoryMock{}
	accessPolicyMock1 := &collectionAccessPolicyMock{}
	accessPolicyMock1.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2")) || bytes.Equal(data.Identity, []byte("p3"))
	}, []string{"org1", "org2"}, false)
	factoryMock1.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock1, nil)

	policyStore := newCollectionStore().withPolicy("col2", uint64(100)).thatMapsTo("p2").withPolicy("col3", uint64(100)).thatMapsTo("p3")
	p1 := gn.newPuller("p1", policyStore, factoryMock1, membership(peerData{"p2", uint64(1)}, peerData{"p3", uint64(1)})...)

	p2TransientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col2",
				},
			},
		},
	}

	policyStore = newCollectionStore().withPolicy("col2", uint64(100)).thatMapsTo("p1")
	factoryMock2 := &collectionAccessFactoryMock{}
	accessPolicyMock2 := &collectionAccessPolicyMock{}
	accessPolicyMock2.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)
	factoryMock2.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock2, nil)

	p2 := gn.newPuller("p2", policyStore, factoryMock2)
	dig1 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col2",
		Namespace:  "ns1",
	}

	store1 := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col2",
			Namespace:  "ns1",
		}: p2TransientStore,
	}

	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig1)), mock.Anything).Return(store1, true, nil)

	p3TransientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col3",
				},
			},
		},
	}

	store2 := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col3",
			Namespace:  "ns1",
		}: p3TransientStore,
	}
	policyStore = newCollectionStore().withPolicy("col3", uint64(100)).thatMapsTo("p1")
	factoryMock3 := &collectionAccessFactoryMock{}
	accessPolicyMock3 := &collectionAccessPolicyMock{}
	accessPolicyMock3.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)
	factoryMock3.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock3, nil)

	p3 := gn.newPuller("p3", policyStore, factoryMock3)
	dig2 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col3",
		Namespace:  "ns1",
	}

	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig2)), mock.Anything).Return(store2, true, nil)

	dasf := &digestsAndSourceFactory{}
	fetchedMessages, err := p1.fetch(dasf.mapDigest(toDigKey(dig1)).toSources().mapDigest(toDigKey(dig2)).toSources().create())
	assert.NoError(t, err)
	rws1 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[0])
	rws2 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[1])
	rws3 := util.PrivateRWSet(fetchedMessages.AvailableElements[1].Payload[0])
	rws4 := util.PrivateRWSet(fetchedMessages.AvailableElements[1].Payload[1])
	fetched := []util.PrivateRWSet{rws1, rws2, rws3, rws4}
	assert.Contains(t, fetched, p2TransientStore.RWSet[0])
	assert.Contains(t, fetched, p2TransientStore.RWSet[1])
	assert.Contains(t, fetched, p3TransientStore.RWSet[0])
	assert.Contains(t, fetched, p3TransientStore.RWSet[1])
}

func TestPullerRetries(t *testing.T) {
	t.Parallel()
//场景：p1从p2、p3、p4和p5拉出。
//只有P3认为P1有资格接收数据。
//其他人认为P1不合格。
	gn := &gossipNetwork{}
	factoryMock1 := &collectionAccessFactoryMock{}
	accessPolicyMock1 := &collectionAccessPolicyMock{}
	accessPolicyMock1.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2")) || bytes.Equal(data.Identity, []byte("p3")) ||
			bytes.Equal(data.Identity, []byte("p4")) ||
			bytes.Equal(data.Identity, []byte("p5"))
	}, []string{"org1", "org2"}, false)
	factoryMock1.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock1, nil)

//P1
	policyStore := newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2", "p3", "p4", "p5")
	p1 := gn.newPuller("p1", policyStore, factoryMock1, membership(peerData{"p2", uint64(1)},
		peerData{"p3", uint64(1)}, peerData{"p4", uint64(1)}, peerData{"p5", uint64(1)})...)

//p2、p3、p4和p5具有相同的瞬态存储
	transientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col1",
				},
			},
		},
	}

	dig := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col1",
		Namespace:  "ns1",
	}

	store := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: transientStore,
	}

//P2
	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p2")
	factoryMock2 := &collectionAccessFactoryMock{}
	accessPolicyMock2 := &collectionAccessPolicyMock{}
	accessPolicyMock2.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2"))
	}, []string{"org1", "org2"}, false)
	factoryMock2.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock2, nil)

	p2 := gn.newPuller("p2", policyStore, factoryMock2)
	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), uint64(0)).Return(store, true, nil)

//P3
	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p1")
	factoryMock3 := &collectionAccessFactoryMock{}
	accessPolicyMock3 := &collectionAccessPolicyMock{}
	accessPolicyMock3.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)
	factoryMock3.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock3, nil)

	p3 := gn.newPuller("p3", policyStore, factoryMock3)
	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), uint64(0)).Return(store, true, nil)

//P4
	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p4")
	factoryMock4 := &collectionAccessFactoryMock{}
	accessPolicyMock4 := &collectionAccessPolicyMock{}
	accessPolicyMock4.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p4"))
	}, []string{"org1", "org2"}, false)
	factoryMock4.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock4, nil)

	p4 := gn.newPuller("p4", policyStore, factoryMock4)
	p4.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), uint64(0)).Return(store, true, nil)

//P5
	policyStore = newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p5")
	factoryMock5 := &collectionAccessFactoryMock{}
	accessPolicyMock5 := &collectionAccessPolicyMock{}
	accessPolicyMock5.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p5"))
	}, []string{"org1", "org2"}, false)
	factoryMock5.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock5, nil)

	p5 := gn.newPuller("p5", policyStore, factoryMock5)
	p5.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig)), uint64(0)).Return(store, true, nil)

//从某人处获取
	dasf := &digestsAndSourceFactory{}
	fetchedMessages, err := p1.fetch(dasf.mapDigest(toDigKey(dig)).toSources().create())
	assert.NoError(t, err)
	rws1 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[0])
	rws2 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[1])
	fetched := []util.PrivateRWSet{rws1, rws2}
	assert.NoError(t, err)
	assert.Equal(t, transientStore.RWSet, fetched)
}

func TestPullerPreferEndorsers(t *testing.T) {
	t.Parallel()
//场景：p1从p2、p3、p4、p5拉出
//而col1唯一的代言人是p3，所以应该选择它
//列1的最高优先级。
//对于col2，只有p2应该有数据，但它不是数据的代言人。
	gn := &gossipNetwork{}
	factoryMock := &collectionAccessFactoryMock{}
	accessPolicyMock2 := &collectionAccessPolicyMock{}
	accessPolicyMock2.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2")) || bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)
	factoryMock.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock2, nil)

	policyStore := newCollectionStore().
		withPolicy("col1", uint64(100)).
		thatMapsTo("p1", "p2", "p3", "p4", "p5").
		withPolicy("col2", uint64(100)).
		thatMapsTo("p1", "p2")
	p1 := gn.newPuller("p1", policyStore, factoryMock, membership(peerData{"p2", uint64(1)},
		peerData{"p3", uint64(1)}, peerData{"p4", uint64(1)}, peerData{"p5", uint64(1)})...)

	p3TransientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col2",
				},
			},
		},
	}

	p2TransientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col2",
				},
			},
		},
	}

	p2 := gn.newPuller("p2", policyStore, factoryMock)
	p3 := gn.newPuller("p3", policyStore, factoryMock)
	gn.newPuller("p4", policyStore, factoryMock)
	gn.newPuller("p5", policyStore, factoryMock)

	dig1 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col1",
		Namespace:  "ns1",
	}

	dig2 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col2",
		Namespace:  "ns1",
	}

	store := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: p3TransientStore,
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col2",
			Namespace:  "ns1",
		}: p2TransientStore,
	}

//我们只为p2上的dig2定义了一个操作，如果有其他同行被要求，测试将因恐慌而失败。
//Dig2上的私人RWset
	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig2)), uint64(0)).Return(store, true, nil)

//我们只为p3上的dig1定义了一个操作，如果有其他同行被要求，测试将因恐慌而失败。
//Dig1上的私人RWset
	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig1)), uint64(0)).Return(store, true, nil)

	dasf := &digestsAndSourceFactory{}
	d2s := dasf.mapDigest(toDigKey(dig1)).toSources("p3").mapDigest(toDigKey(dig2)).toSources().create()
	fetchedMessages, err := p1.fetch(d2s)
	assert.NoError(t, err)
	rws1 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[0])
	rws2 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[1])
	rws3 := util.PrivateRWSet(fetchedMessages.AvailableElements[1].Payload[0])
	rws4 := util.PrivateRWSet(fetchedMessages.AvailableElements[1].Payload[1])
	fetched := []util.PrivateRWSet{rws1, rws2, rws3, rws4}
	assert.Contains(t, fetched, p3TransientStore.RWSet[0])
	assert.Contains(t, fetched, p3TransientStore.RWSet[1])
	assert.Contains(t, fetched, p2TransientStore.RWSet[0])
	assert.Contains(t, fetched, p2TransientStore.RWSet[1])
}

func TestPullerFetchReconciledItemsPreferPeersFromOriginalConfig(t *testing.T) {
	t.Parallel()
//场景：p1从p2、p3、p4、p5拉出
//为col1创建数据时，集合配置中唯一的对等点是p3，因此应该选择它。
//列1的最高优先级。
//对于第2列，在创建数据时，p3处于收集配置中，但已从收集中删除，现在只有p2应该具有数据。
//所以显然，应该为第2列选择p2。
	gn := &gossipNetwork{}
	factoryMock := &collectionAccessFactoryMock{}
	accessPolicyMock2 := &collectionAccessPolicyMock{}
	accessPolicyMock2.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p2")) || bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)
	factoryMock.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock2, nil)

	policyStore := newCollectionStore().
		withPolicy("col1", uint64(100)).
		thatMapsTo("p1", "p2", "p3", "p4", "p5").
		withPolicy("col2", uint64(100)).
		thatMapsTo("p1", "p2").
		withAccessFilter(func(data fcommon.SignedData) bool {
			return bytes.Equal(data.Identity, []byte("p3"))
		})

	p1 := gn.newPuller("p1", policyStore, factoryMock, membership(peerData{"p2", uint64(1)},
		peerData{"p3", uint64(1)}, peerData{"p4", uint64(1)}, peerData{"p5", uint64(1)})...)

	p3TransientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col2",
				},
			},
		},
	}

	p2TransientStore := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col2",
				},
			},
		},
	}

	p2 := gn.newPuller("p2", policyStore, factoryMock)
	p3 := gn.newPuller("p3", policyStore, factoryMock)
	gn.newPuller("p4", policyStore, factoryMock)
	gn.newPuller("p5", policyStore, factoryMock)

	dig1 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col1",
		Namespace:  "ns1",
	}

	dig2 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col2",
		Namespace:  "ns1",
	}

	store := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: p3TransientStore,
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col2",
			Namespace:  "ns1",
		}: p2TransientStore,
	}

//我们只为p2上的dig2定义了一个操作，如果有其他同行被要求，测试将因恐慌而失败。
//Dig2上的私人RWset
	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig2)), uint64(0)).Return(store, true, nil)

//我们只为p3上的dig1定义了一个操作，如果有其他同行被要求，测试将因恐慌而失败。
//Dig1上的私人RWset
	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig1)), uint64(0)).Return(store, true, nil)

	d2cc := privdatacommon.Dig2CollectionConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: &fcommon.StaticCollectionConfig{
			Name: "col1",
		},
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col2",
			Namespace:  "ns1",
		}: &fcommon.StaticCollectionConfig{
			Name: "col2",
		},
	}

	fetchedMessages, err := p1.FetchReconciledItems(d2cc)
	assert.NoError(t, err)
	rws1 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[0])
	rws2 := util.PrivateRWSet(fetchedMessages.AvailableElements[0].Payload[1])
	rws3 := util.PrivateRWSet(fetchedMessages.AvailableElements[1].Payload[0])
	rws4 := util.PrivateRWSet(fetchedMessages.AvailableElements[1].Payload[1])
	fetched := []util.PrivateRWSet{rws1, rws2, rws3, rws4}
	assert.Contains(t, fetched, p3TransientStore.RWSet[0])
	assert.Contains(t, fetched, p3TransientStore.RWSet[1])
	assert.Contains(t, fetched, p2TransientStore.RWSet[0])
	assert.Contains(t, fetched, p2TransientStore.RWSet[1])
}

func TestPullerAvoidPullingPurgedData(t *testing.T) {
//场景：p1缺少col1的私有数据
//假设p2和p3有，而p3有更高级的
//分类帐和基于BTL的已清除数据，因此p1
//假设只从p2获取数据

	t.Parallel()
	gn := &gossipNetwork{}
	factoryMock := &collectionAccessFactoryMock{}
	accessPolicyMock2 := &collectionAccessPolicyMock{}
	accessPolicyMock2.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)
	factoryMock.On("AccessPolicy", mock.Anything, mock.Anything).Return(accessPolicyMock2, nil)

	policyStore := newCollectionStore().withPolicy("col1", uint64(100)).thatMapsTo("p1", "p2", "p3").
		withPolicy("col2", uint64(1000)).thatMapsTo("p1", "p2", "p3")

//P2位于分类帐高度1处，而P2位于111处，超出为第1列（100）定义的BTL。
	p1 := gn.newPuller("p1", policyStore, factoryMock, membership(peerData{"p2", uint64(1)},
		peerData{"p3", uint64(111)})...)

	privateData1 := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col1",
				},
			},
		},
	}
	privateData2 := &util.PrivateRWSetWithConfig{
		RWSet: newPRWSet(),
		CollectionConfig: &fcommon.CollectionConfig{
			Payload: &fcommon.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &fcommon.StaticCollectionConfig{
					Name: "col2",
				},
			},
		},
	}

	p2 := gn.newPuller("p2", policyStore, factoryMock)
	p3 := gn.newPuller("p3", policyStore, factoryMock)

	dig1 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col1",
		Namespace:  "ns1",
	}

	dig2 := &proto.PvtDataDigest{
		TxId:       "txID1",
		Collection: "col2",
		Namespace:  "ns1",
	}

	store := Dig2PvtRWSetWithConfig{
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col1",
			Namespace:  "ns1",
		}: privateData1,
		privdatacommon.DigKey{
			TxId:       "txID1",
			Collection: "col2",
			Namespace:  "ns1",
		}: privateData2,
	}

	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig1)), 0).Return(store, true, nil)
	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig1)), 0).Return(store, true, nil).
		Run(
			func(arg mock.Arguments) {
				assert.Fail(t, "we should not fetch private data from peers where it was purged")
			},
		)

	p3.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig2)), uint64(0)).Return(store, true, nil)
	p2.PrivateDataRetriever.(*dataRetrieverMock).On("CollectionRWSet", mock.MatchedBy(protoMatcher(dig2)), uint64(0)).Return(store, true, nil).
		Run(
			func(mock.Arguments) {
				assert.Fail(t, "we should not fetch private data of collection2 from peer 2")

			},
		)

	dasf := &digestsAndSourceFactory{}
	d2s := dasf.mapDigest(toDigKey(dig1)).toSources("p3", "p2").mapDigest(toDigKey(dig2)).toSources("p3").create()
//正在尝试获取块seq 1缺少的pvt数据
	fetchedMessages, err := p1.fetch(d2s)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(fetchedMessages.PurgedElements))
	assert.Equal(t, dig1, fetchedMessages.PurgedElements[0])
	p3.PrivateDataRetriever.(*dataRetrieverMock).AssertNumberOfCalls(t, "CollectionRWSet", 1)

}

type counterDataRetreiver struct {
	numberOfCalls int
	PrivateDataRetriever
}

func (c *counterDataRetreiver) CollectionRWSet(dig []*proto.PvtDataDigest, blockNum uint64) (Dig2PvtRWSetWithConfig, bool, error) {
	c.numberOfCalls += 1
	return c.PrivateDataRetriever.CollectionRWSet(dig, blockNum)
}

func (c *counterDataRetreiver) getNumberOfCalls() int {
	return c.numberOfCalls
}

func TestPullerIntegratedWithDataRetreiver(t *testing.T) {
	t.Parallel()
	gn := &gossipNetwork{}

	ns1, ns2 := "testChaincodeName1", "testChaincodeName2"
	col1, col2 := "testCollectionName1", "testCollectionName2"

	ap := &collectionAccessPolicyMock{}
	ap.Setup(1, 2, func(data fcommon.SignedData) bool {
		return bytes.Equal(data.Identity, []byte("p1"))
	}, []string{"org1", "org2"}, false)

	factoryMock := &collectionAccessFactoryMock{}
	factoryMock.On("AccessPolicy", mock.Anything, mock.Anything).Return(ap, nil)

	policyStore := newCollectionStore().withPolicy(col1, uint64(1000)).thatMapsTo("p1", "p2").
		withPolicy(col2, uint64(1000)).thatMapsTo("p1", "p2")

	p1 := gn.newPuller("p1", policyStore, factoryMock, membership(peerData{"p2", uint64(10)})...)
	p2 := gn.newPuller("p2", policyStore, factoryMock, membership(peerData{"p1", uint64(1)})...)

	dataStore := &mocks.DataStore{}
	result := []*ledger.TxPvtData{
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns1, col1, []byte{1}),
					pvtReadWriteSet(ns1, col1, []byte{2}),
				},
			},
			SeqInBlock: 1,
		},
		{
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					pvtReadWriteSet(ns2, col2, []byte{3}),
					pvtReadWriteSet(ns2, col2, []byte{4}),
				},
			},
			SeqInBlock: 2,
		},
	}

	dataStore.On("LedgerHeight").Return(uint64(10), nil)
	dataStore.On("GetPvtDataByNum", uint64(5), mock.Anything).Return(result, nil)
	historyRetreiver := &mocks.ConfigHistoryRetriever{}
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, ns1).Return(newCollectionConfig(col1), nil)
	historyRetreiver.On("MostRecentCollectionConfigBelow", mock.Anything, ns2).Return(newCollectionConfig(col2), nil)
	dataStore.On("GetConfigHistoryRetriever").Return(historyRetreiver, nil)

	dataRetreiver := &counterDataRetreiver{PrivateDataRetriever: NewDataRetriever(dataStore), numberOfCalls: 0}
	p2.PrivateDataRetriever = dataRetreiver

	dig1 := &privdatacommon.DigKey{
		TxId:       "txID1",
		Collection: col1,
		Namespace:  ns1,
		BlockSeq:   5,
		SeqInBlock: 1,
	}

	dig2 := &privdatacommon.DigKey{
		TxId:       "txID1",
		Collection: col2,
		Namespace:  ns2,
		BlockSeq:   5,
		SeqInBlock: 2,
	}

	dasf := &digestsAndSourceFactory{}
	d2s := dasf.mapDigest(dig1).toSources("p2").mapDigest(dig2).toSources("p2").create()
	fetchedMessages, err := p1.fetch(d2s)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fetchedMessages.AvailableElements))
	assert.Equal(t, 1, dataRetreiver.getNumberOfCalls())
	assert.Equal(t, 2, len(fetchedMessages.AvailableElements[0].Payload))
	assert.Equal(t, 2, len(fetchedMessages.AvailableElements[1].Payload))
}

func toDigKey(dig *proto.PvtDataDigest) *privdatacommon.DigKey {
	return &privdatacommon.DigKey{
		TxId:       dig.TxId,
		BlockSeq:   dig.BlockSeq,
		SeqInBlock: dig.SeqInBlock,
		Namespace:  dig.Namespace,
		Collection: dig.Collection,
	}
}
