
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
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	util2 "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/transientstore"
	privdatacommon "github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/gossip/privdata/mocks"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/peer"
	transientstore2 "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	viper.Set("peer.gossip.pvtData.pullRetryThreshold", time.Second*3)
	factory.InitFactories(nil)
}

//集合标准集总标准
//藏书
type CollectionCriteria struct {
	Channel    string
	TxId       string
	Collection string
	Namespace  string
}

func fromCollectionCriteria(criteria common.CollectionCriteria) CollectionCriteria {
	return CollectionCriteria{
		TxId:       criteria.TxId,
		Collection: criteria.Collection,
		Namespace:  criteria.Namespace,
		Channel:    criteria.Channel,
	}
}

type persistCall struct {
	*mock.Call
	store *mockTransientStore
}

func (pc *persistCall) expectRWSet(namespace string, collection string, rws []byte) *persistCall {
	if pc.store.persists == nil {
		pc.store.persists = make(map[rwsTriplet]struct{})
	}
	pc.store.persists[rwsTriplet{
		namespace:  namespace,
		collection: collection,
		rwset:      hex.EncodeToString(rws),
	}] = struct{}{}
	return pc
}

type mockTransientStore struct {
	t *testing.T
	mock.Mock
	persists      map[rwsTriplet]struct{}
	lastReqTxID   string
	lastReqFilter map[string]ledger.PvtCollFilter
}

func (store *mockTransientStore) On(methodName string, arguments ...interface{}) *persistCall {
	return &persistCall{
		store: store,
		Call:  store.Mock.On(methodName, arguments...),
	}
}

func (store *mockTransientStore) PurgeByTxids(txids []string) error {
	args := store.Called(txids)
	return args.Error(0)
}

func (store *mockTransientStore) Persist(txid string, blockHeight uint64, res *rwset.TxPvtReadWriteSet) error {
	key := rwsTriplet{
		namespace:  res.NsPvtRwset[0].Namespace,
		collection: res.NsPvtRwset[0].CollectionPvtRwset[0].CollectionName,
		rwset:      hex.EncodeToString(res.NsPvtRwset[0].CollectionPvtRwset[0].Rwset)}
	if _, exists := store.persists[key]; !exists {
		store.t.Fatal("Shouldn't have persisted", res)
	}
	delete(store.persists, key)
	store.Called(txid, blockHeight, res)
	return nil
}

func (store *mockTransientStore) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore2.TxPvtReadWriteSetWithConfigInfo) error {
	res := privateSimulationResultsWithConfig.PvtRwset
	key := rwsTriplet{
		namespace:  res.NsPvtRwset[0].Namespace,
		collection: res.NsPvtRwset[0].CollectionPvtRwset[0].CollectionName,
		rwset:      hex.EncodeToString(res.NsPvtRwset[0].CollectionPvtRwset[0].Rwset)}
	if _, exists := store.persists[key]; !exists {
		store.t.Fatal("Shouldn't have persisted", res)
	}
	delete(store.persists, key)
	store.Called(txid, blockHeight, privateSimulationResultsWithConfig)
	return nil
}

func (store *mockTransientStore) PurgeByHeight(maxBlockNumToRetain uint64) error {
	return store.Called(maxBlockNumToRetain).Error(0)
}

func (store *mockTransientStore) GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error) {
	store.lastReqTxID = txid
	store.lastReqFilter = filter
	args := store.Called(txid, filter)
	if args.Get(1) == nil {
		return args.Get(0).(transientstore.RWSetScanner), nil
	}
	return nil, args.Get(1).(error)
}

type mockRWSetScanner struct {
	err     error
	results []*transientstore.EndorserPvtSimulationResultsWithConfig
}

func (scanner *mockRWSetScanner) withRWSet(ns string, col string) *mockRWSetScanner {
	scanner.results = append(scanner.results, &transientstore.EndorserPvtSimulationResultsWithConfig{
		PvtSimulationResultsWithConfig: &transientstore2.TxPvtReadWriteSetWithConfigInfo{
			PvtRwset: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					{
						Namespace: ns,
						CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
							{
								CollectionName: col,
								Rwset:          []byte("rws-pre-image"),
							},
						},
					},
				},
			},
			CollectionConfigs: map[string]*common.CollectionConfigPackage{
				ns: {
					Config: []*common.CollectionConfig{
						{
							Payload: &common.CollectionConfig_StaticCollectionConfig{
								StaticCollectionConfig: &common.StaticCollectionConfig{
									Name: col,
								},
							},
						},
					},
				},
			},
		},
	})
	return scanner
}

func (scanner *mockRWSetScanner) Next() (*transientstore.EndorserPvtSimulationResults, error) {
	panic("should not be used")
}

func (scanner *mockRWSetScanner) NextWithConfig() (*transientstore.EndorserPvtSimulationResultsWithConfig, error) {
	if scanner.err != nil {
		return nil, scanner.err
	}
	var res *transientstore.EndorserPvtSimulationResultsWithConfig
	if len(scanner.results) == 0 {
		return nil, nil
	}
	res, scanner.results = scanner.results[len(scanner.results)-1], scanner.results[:len(scanner.results)-1]
	return res, nil
}

func (*mockRWSetScanner) Close() {
}

type validatorMock struct {
	err error
}

func (v *validatorMock) Validate(block *common.Block) error {
	if v.err != nil {
		return v.err
	}
	return nil
}

type digests []privdatacommon.DigKey

func (d digests) Equal(other digests) bool {
	flatten := func(d digests) map[privdatacommon.DigKey]struct{} {
		m := map[privdatacommon.DigKey]struct{}{}
		for _, dig := range d {
			m[dig] = struct{}{}
		}
		return m
	}
	return reflect.DeepEqual(flatten(d), flatten(other))
}

type fetchCall struct {
	fetcher *fetcherMock
	*mock.Call
}

func (fc *fetchCall) expectingEndorsers(orgs ...string) *fetchCall {
	if fc.fetcher.expectedEndorsers == nil {
		fc.fetcher.expectedEndorsers = make(map[string]struct{})
	}
	for _, org := range orgs {
		sID := &msp.SerializedIdentity{Mspid: org, IdBytes: []byte(fmt.Sprintf("p0%s", org))}
		b, _ := pb.Marshal(sID)
		fc.fetcher.expectedEndorsers[string(b)] = struct{}{}
	}

	return fc
}

func (fc *fetchCall) expectingDigests(digests []privdatacommon.DigKey) *fetchCall {
	fc.fetcher.expectedDigests = digests
	return fc
}

func (fc *fetchCall) Return(returnArguments ...interface{}) *mock.Call {

	return fc.Call.Return(returnArguments...)
}

type fetcherMock struct {
	t *testing.T
	mock.Mock
	expectedDigests   []privdatacommon.DigKey
	expectedEndorsers map[string]struct{}
}

func (f *fetcherMock) On(methodName string, arguments ...interface{}) *fetchCall {
	return &fetchCall{
		fetcher: f,
		Call:    f.Mock.On(methodName, arguments...),
	}
}

func (f *fetcherMock) fetch(dig2src dig2sources) (*privdatacommon.FetchedPvtDataContainer, error) {
	for _, endorsements := range dig2src {
		for _, endorsement := range endorsements {
			_, exists := f.expectedEndorsers[string(endorsement.Endorser)]
			if !exists {
				f.t.Fatalf("Encountered a non-expected endorser: %s", string(endorsement.Endorser))
			}
//否则，它就存在了，所以请删除它，以便在调用结束时得到一个空的预期映射
			delete(f.expectedEndorsers, string(endorsement.Endorser))
		}
	}
	assert.True(f.t, digests(dig2src.keys()).Equal(digests(f.expectedDigests)))
	assert.Empty(f.t, f.expectedEndorsers)
	args := f.Called(dig2src)
	if args.Get(1) == nil {
		return args.Get(0).(*privdatacommon.FetchedPvtDataContainer), nil
	}
	return nil, args.Get(1).(error)
}

func createcollectionStore(expectedSignedData common.SignedData) *collectionStore {
	return &collectionStore{
		expectedSignedData: expectedSignedData,
		policies:           make(map[collectionAccessPolicy]CollectionCriteria),
		store:              make(map[CollectionCriteria]collectionAccessPolicy),
	}
}

type collectionStore struct {
	expectedSignedData common.SignedData
	acceptsAll         bool
	acceptsNone        bool
	lenient            bool
	store              map[CollectionCriteria]collectionAccessPolicy
	policies           map[collectionAccessPolicy]CollectionCriteria
}

func (cs *collectionStore) thatAcceptsAll() *collectionStore {
	cs.acceptsAll = true
	return cs
}

func (cs *collectionStore) thatAcceptsNone() *collectionStore {
	cs.acceptsNone = true
	return cs
}

func (cs *collectionStore) andIsLenient() *collectionStore {
	cs.lenient = true
	return cs
}

func (cs *collectionStore) thatAccepts(cc CollectionCriteria) *collectionStore {
	sp := collectionAccessPolicy{
		cs: cs,
		n:  util.RandomUInt64(),
	}
	cs.store[cc] = sp
	cs.policies[sp] = cc
	return cs
}

func (cs *collectionStore) RetrieveCollectionAccessPolicy(cc common.CollectionCriteria) (privdata.CollectionAccessPolicy, error) {
	if sp, exists := cs.store[fromCollectionCriteria(cc)]; exists {
		return &sp, nil
	}
	if cs.acceptsAll || cs.acceptsNone || cs.lenient {
		return &collectionAccessPolicy{
			cs: cs,
			n:  util.RandomUInt64(),
		}, nil
	}
	return nil, privdata.NoSuchCollectionError{}
}

func (cs *collectionStore) RetrieveCollection(common.CollectionCriteria) (privdata.Collection, error) {
	panic("implement me")
}

func (cs *collectionStore) HasReadAccess(cc common.CollectionCriteria, sp *peer.SignedProposal, qe ledger.QueryExecutor) (bool, error) {
	panic("implement me")
}

func (cs *collectionStore) RetrieveCollectionConfigPackage(cc common.CollectionCriteria) (*common.CollectionConfigPackage, error) {
	return &common.CollectionConfigPackage{
		Config: []*common.CollectionConfig{
			{
				Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name:              cc.Collection,
						MaximumPeerCount:  1,
						RequiredPeerCount: 1,
					},
				},
			},
		},
	}, nil
}

func (cs *collectionStore) RetrieveCollectionPersistenceConfigs(cc common.CollectionCriteria) (privdata.CollectionPersistenceConfigs, error) {
	panic("implement me")
}

func (cs *collectionStore) AccessFilter(channelName string, collectionPolicyConfig *common.CollectionPolicyConfig) (privdata.Filter, error) {
	panic("implement me")
}

type collectionAccessPolicy struct {
	cs *collectionStore
	n  uint64
}

func (cap *collectionAccessPolicy) MemberOrgs() []string {
	return []string{"org0", "org1"}
}

func (cap *collectionAccessPolicy) RequiredPeerCount() int {
	return 1
}

func (cap *collectionAccessPolicy) MaximumPeerCount() int {
	return 2
}

func (cap *collectionAccessPolicy) IsMemberOnlyRead() bool {
	return false
}

func (cap *collectionAccessPolicy) AccessFilter() privdata.Filter {
	return func(sd common.SignedData) bool {
		that, _ := asn1.Marshal(sd)
		this, _ := asn1.Marshal(cap.cs.expectedSignedData)
		if hex.EncodeToString(that) != hex.EncodeToString(this) {
			panic(fmt.Errorf("self signed data passed isn't equal to expected:%v, %v", sd, cap.cs.expectedSignedData))
		}

		if cap.cs.acceptsNone {
			return false
		} else if cap.cs.acceptsAll {
			return true
		}

		_, exists := cap.cs.policies[*cap]
		return exists
	}
}

func TestPvtDataCollections_FailOnEmptyPayload(t *testing.T) {
	collection := &util.PvtDataCollections{
		&ledger.TxPvtData{
			SeqInBlock: uint64(1),
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					{
						Namespace: "ns1",
						CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
							{
								CollectionName: "secretCollection",
								Rwset:          []byte{1, 2, 3, 4, 5, 6, 7},
							},
						},
					},
				},
			},
		},

		nil,
	}

	_, err := collection.Marshal()
	assertion := assert.New(t)
	assertion.Error(err, "Expected to fail since second item has nil payload")
	assertion.Equal("Mallformed private data payload, rwset index 1 is nil", fmt.Sprintf("%s", err))
}

func TestPvtDataCollections_FailMarshalingWriteSet(t *testing.T) {
	collection := &util.PvtDataCollections{
		&ledger.TxPvtData{
			SeqInBlock: uint64(1),
			WriteSet:   nil,
		},
	}

	_, err := collection.Marshal()
	assertion := assert.New(t)
	assertion.Error(err, "Expected to fail since first item has nil writeset")
	assertion.Contains(fmt.Sprintf("%s", err), "Could not marshal private rwset index 0")
}

func TestPvtDataCollections_Marshal(t *testing.T) {
	collection := &util.PvtDataCollections{
		&ledger.TxPvtData{
			SeqInBlock: uint64(1),
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					{
						Namespace: "ns1",
						CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
							{
								CollectionName: "secretCollection",
								Rwset:          []byte{1, 2, 3, 4, 5, 6, 7},
							},
						},
					},
				},
			},
		},

		&ledger.TxPvtData{
			SeqInBlock: uint64(2),
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					{
						Namespace: "ns1",
						CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
							{
								CollectionName: "secretCollection",
								Rwset:          []byte{42, 42, 42, 42, 42, 42, 42},
							},
						},
					},
					{
						Namespace: "ns2",
						CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
							{
								CollectionName: "otherCollection",
								Rwset:          []byte{10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
							},
						},
					},
				},
			},
		},
	}

	bytes, err := collection.Marshal()

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotNil(bytes)
	assertion.Equal(2, len(bytes))
}

func TestPvtDataCollections_Unmarshal(t *testing.T) {
	collection := util.PvtDataCollections{
		&ledger.TxPvtData{
			SeqInBlock: uint64(1),
			WriteSet: &rwset.TxPvtReadWriteSet{
				DataModel: rwset.TxReadWriteSet_KV,
				NsPvtRwset: []*rwset.NsPvtReadWriteSet{
					{
						Namespace: "ns1",
						CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
							{
								CollectionName: "secretCollection",
								Rwset:          []byte{1, 2, 3, 4, 5, 6, 7},
							},
						},
					},
				},
			},
		},
	}

	bytes, err := collection.Marshal()

	assertion := assert.New(t)
	assertion.NoError(err)
	assertion.NotNil(bytes)
	assertion.Equal(1, len(bytes))

	var newCol util.PvtDataCollections

	err = newCol.Unmarshal(bytes)
	assertion.NoError(err)
	assertion.Equal(1, len(newCol))
	assertion.Equal(newCol[0].SeqInBlock, collection[0].SeqInBlock)
	assertion.True(pb.Equal(newCol[0].WriteSet, collection[0].WriteSet))
}

type rwsTriplet struct {
	namespace  string
	collection string
	rwset      string
}

func flattenTxPvtDataMap(pd ledger.TxPvtDataMap) map[uint64]map[rwsTriplet]struct{} {
	m := make(map[uint64]map[rwsTriplet]struct{})
	for seqInBlock, namespaces := range pd {
		triplets := make(map[rwsTriplet]struct{})
		for _, namespace := range namespaces.WriteSet.NsPvtRwset {
			for _, col := range namespace.CollectionPvtRwset {
				triplets[rwsTriplet{
					namespace:  namespace.Namespace,
					collection: col.CollectionName,
					rwset:      hex.EncodeToString(col.Rwset),
				}] = struct{}{}
			}
		}
		m[seqInBlock] = triplets
	}
	return m
}

var expectedCommittedPrivateData1 = map[uint64]*ledger.TxPvtData{
	0: {SeqInBlock: 0, WriteSet: &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "ns1",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "c1",
						Rwset:          []byte("rws-pre-image"),
					},
					{
						CollectionName: "c2",
						Rwset:          []byte("rws-pre-image"),
					},
				},
			},
		},
	}},
	1: {SeqInBlock: 1, WriteSet: &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "ns2",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "c1",
						Rwset:          []byte("rws-pre-image"),
					},
				},
			},
		},
	}},
}

var expectedCommittedPrivateData2 = map[uint64]*ledger.TxPvtData{
	0: {SeqInBlock: 0, WriteSet: &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "ns3",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "c3",
						Rwset:          []byte("rws-pre-image"),
					},
				},
			},
		},
	}},
}

var expectedCommittedPrivateData3 = map[uint64]*ledger.TxPvtData{}

func TestCoordinatorStoreInvalidBlock(t *testing.T) {
	peerSelfSignedData := common.SignedData{
		Identity:  []byte{0, 1, 2},
		Signature: []byte{3, 4, 5},
		Data:      []byte{6, 7, 8},
	}
	hash := util2.ComputeSHA256([]byte("rws-pre-image"))
	committer := &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		t.Fatal("Shouldn't have committed")
	}).Return(nil)
	cs := createcollectionStore(peerSelfSignedData).thatAcceptsAll()
	purgedTxns := make(map[string]struct{})
	store := &mockTransientStore{t: t}
	assertPurged := func(txns ...string) {
		for _, txn := range txns {
			_, exists := purgedTxns[txn]
			assert.True(t, exists)
			delete(purgedTxns, txn)
		}
		assert.Len(t, purgedTxns, 0)
	}
	store.On("PurgeByTxids", mock.Anything).Run(func(args mock.Arguments) {
		for _, txn := range args.Get(0).([]string) {
			purgedTxns[txn] = struct{}{}
		}
	}).Return(nil)
	fetcher := &fetcherMock{t: t}
	pdFactory := &pvtDataFactory{}
	bf := &blockFactory{
		channelID: "test",
	}

	block := bf.withoutMetadata().create()
//场景一：我们得到的块没有任何元数据
	pvtData := pdFactory.create()
	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err := coordinator.StoreBlock(block, pvtData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Block.Metadata is nil or Block.Metadata lacks a Tx filter bitmap")

//场景二：验证程序在验证块时出错
	block = bf.create()
	pvtData = pdFactory.create()
	coordinator = NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{fmt.Errorf("failed validating block")},
	}, peerSelfSignedData)
	err = coordinator.StoreBlock(block, pvtData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed validating block")

//场景三：我们得到的块在元数据中包含的Tx过滤器长度不足
	block = bf.withMetadataSize(100).create()
	pvtData = pdFactory.create()
	coordinator = NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err = coordinator.StoreBlock(block, pvtData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Block data size")
	assert.Contains(t, err.Error(), "is different from Tx filter size")

//场景四：我们得到的块中的第二个事务是无效的，我们没有用于此的私有数据。
//如果协调者试图获取私有数据，那么测试将失败，因为我们没有定义
//此测试中临时存储（或八卦）的模拟操作。
	var commitHappened bool
	assertCommitHappened := func() {
		assert.True(t, commitHappened)
		commitHappened = false
	}
	committer = &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		privateDataPassed2Ledger := args.Get(0).(*ledger.BlockAndPvtData).PvtData
		commitHappened = true
//只有第一个交易的私有数据传递到分类帐
		assert.Len(t, privateDataPassed2Ledger, 1)
		assert.Equal(t, 0, int(privateDataPassed2Ledger[0].SeqInBlock))
//传递到分类帐的私有数据包含“NS1”，其中包含2个集合
		assert.Len(t, privateDataPassed2Ledger[0].WriteSet.NsPvtRwset, 1)
		assert.Equal(t, "ns1", privateDataPassed2Ledger[0].WriteSet.NsPvtRwset[0].Namespace)
		assert.Len(t, privateDataPassed2Ledger[0].WriteSet.NsPvtRwset[0].CollectionPvtRwset, 2)
	}).Return(nil)
	block = bf.withInvalidTxns(1).AddTxn("tx1", "ns1", hash, "c1", "c2").AddTxn("tx2", "ns2", hash, "c1").create()
	pvtData = pdFactory.addRWSet().addNSRWSet("ns1", "c1", "c2").create()
	coordinator = NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err = coordinator.StoreBlock(block, pvtData)
	assert.NoError(t, err)
	assertCommitHappened()
//确保第二个事务无效且未提交-仍被清除。
//这样一来，如果我们通过背书人的传播获得交易，我们就将其清除。
//当它的障碍物到来时。
	assertPurged("tx1", "tx2")

//场景V:块不包含标题
	block.Header = nil
	err = coordinator.StoreBlock(block, pvtData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Block header is nil")

//场景六：块不包含数据
	block.Data = nil
	err = coordinator.StoreBlock(block, pvtData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Block data is empty")
}

func TestCoordinatorToFilterOutPvtRWSetsWithWrongHash(t *testing.T) {
 /*
  测试用例，其中对等端接收用于提交的新块
  它在瞬态存储器中有NS1:C1，但它有错误
  哈希，因此它将从其他对等端获取ns1:c1
 **/

	peerSelfSignedData := common.SignedData{
		Identity:  []byte{0, 1, 2},
		Signature: []byte{3, 4, 5},
		Data:      []byte{6, 7, 8},
	}

	expectedPvtData := map[uint64]*ledger.TxPvtData{
		0: {SeqInBlock: 0, WriteSet: &rwset.TxPvtReadWriteSet{
			DataModel: rwset.TxReadWriteSet_KV,
			NsPvtRwset: []*rwset.NsPvtReadWriteSet{
				{
					Namespace: "ns1",
					CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
						{
							CollectionName: "c1",
							Rwset:          []byte("rws-original"),
						},
					},
				},
			},
		}},
	}

	cs := createcollectionStore(peerSelfSignedData).thatAcceptsAll()
	committer := &mocks.Committer{}
	store := &mockTransientStore{t: t}

	fetcher := &fetcherMock{t: t}

	var commitHappened bool

	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		privateDataPassed2Ledger := args.Get(0).(*ledger.BlockAndPvtData).PvtData
		assert.True(t, reflect.DeepEqual(flattenTxPvtDataMap(privateDataPassed2Ledger),
			flattenTxPvtDataMap(expectedPvtData)))
		commitHappened = true
	}).Return(nil)

	hash := util2.ComputeSHA256([]byte("rws-original"))
	bf := &blockFactory{
		channelID: "test",
	}

	block := bf.AddTxnWithEndorsement("tx1", "ns1", hash, "org1", true, "c1").create()
	store.On("GetTxPvtRWSetByTxid", "tx1", mock.Anything).Return((&mockRWSetScanner{}).withRWSet("ns1", "c1"), nil)

	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)

	fetcher.On("fetch", mock.Anything).expectingDigests([]privdatacommon.DigKey{
		{
			TxId: "tx1", Namespace: "ns1", Collection: "c1", BlockSeq: 1,
		},
	}).expectingEndorsers("org1").Return(&privdatacommon.FetchedPvtDataContainer{
		AvailableElements: []*proto.PvtDataElement{
			{
				Digest: &proto.PvtDataDigest{
					BlockSeq:   1,
					Collection: "c1",
					Namespace:  "ns1",
					TxId:       "tx1",
				},
				Payload: [][]byte{[]byte("rws-original")},
			},
		},
	}, nil)
	store.On("Persist", mock.Anything, uint64(1), mock.Anything).
		expectRWSet("ns1", "c1", []byte("rws-original")).Return(nil)

	purgedTxns := make(map[string]struct{})
	store.On("PurgeByTxids", mock.Anything).Run(func(args mock.Arguments) {
		for _, txn := range args.Get(0).([]string) {
			purgedTxns[txn] = struct{}{}
		}
	}).Return(nil)

	coordinator.StoreBlock(block, nil)
//断言块最终被提交
	assert.True(t, commitHappened)

//断言事务已清除
	_, exists := purgedTxns["tx1"]
	assert.True(t, exists)
}

func TestCoordinatorStoreBlock(t *testing.T) {
	peerSelfSignedData := common.SignedData{
		Identity:  []byte{0, 1, 2},
		Signature: []byte{3, 4, 5},
		Data:      []byte{6, 7, 8},
	}
//绿色路径测试，应成功获取所有私有数据

	cs := createcollectionStore(peerSelfSignedData).thatAcceptsAll()

	var commitHappened bool
	assertCommitHappened := func() {
		assert.True(t, commitHappened)
		commitHappened = false
	}
	committer := &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		privateDataPassed2Ledger := args.Get(0).(*ledger.BlockAndPvtData).PvtData
		assert.True(t, reflect.DeepEqual(flattenTxPvtDataMap(privateDataPassed2Ledger),
			flattenTxPvtDataMap(expectedCommittedPrivateData1)))
		commitHappened = true
	}).Return(nil)
	purgedTxns := make(map[string]struct{})
	store := &mockTransientStore{t: t}
	store.On("PurgeByTxids", mock.Anything).Run(func(args mock.Arguments) {
		for _, txn := range args.Get(0).([]string) {
			purgedTxns[txn] = struct{}{}
		}
	}).Return(nil)
	assertPurged := func(txns ...string) {
		for _, txn := range txns {
			_, exists := purgedTxns[txn]
			assert.True(t, exists)
			delete(purgedTxns, txn)
		}
		assert.Len(t, purgedTxns, 0)
	}

	fetcher := &fetcherMock{t: t}

	hash := util2.ComputeSHA256([]byte("rws-pre-image"))
	pdFactory := &pvtDataFactory{}
	bf := &blockFactory{
		channelID: "test",
	}
	block := bf.AddTxnWithEndorsement("tx1", "ns1", hash, "org1", true, "c1", "c2").
		AddTxnWithEndorsement("tx2", "ns2", hash, "org2", true, "c1").create()

	fmt.Println("Scenario I")
//场景一：我们得到的块旁边有足够的私有数据。
//如果协调者试图从转瞬即逝的商店或同伴那里获取信息，会导致恐慌，
//因为我们还没有定义临时存储或其他对等机的“on（…）”调用。
	pvtData := pdFactory.addRWSet().addNSRWSet("ns1", "c1", "c2").addRWSet().addNSRWSet("ns2", "c1").create()
	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err := coordinator.StoreBlock(block, pvtData)
	assert.NoError(t, err)
	assertCommitHappened()
	assertPurged("tx1", "tx2")

	fmt.Println("Scenario II")
//场景二：我们得到的数据块没有足够的私有数据，
//它缺少NS1:C2，但数据存在于临时存储中
	store.On("GetTxPvtRWSetByTxid", "tx1", mock.Anything).Return((&mockRWSetScanner{}).withRWSet("ns1", "c2"), nil)
	store.On("GetTxPvtRWSetByTxid", "tx2", mock.Anything).Return(&mockRWSetScanner{}, nil)
	pvtData = pdFactory.addRWSet().addNSRWSet("ns1", "c1").addRWSet().addNSRWSet("ns2", "c1").create()
	err = coordinator.StoreBlock(block, pvtData)
	assert.NoError(t, err)
	assertCommitHappened()
	assertPurged("tx1", "tx2")
	assert.Equal(t, "tx1", store.lastReqTxID)
	assert.Equal(t, map[string]ledger.PvtCollFilter{
		"ns1": map[string]bool{
			"c2": true,
		},
	}, store.lastReqFilter)

	fmt.Println("Scenario III")
//场景三：块旁边没有足够的私有数据，
//缺少NS1:C2，数据存在于瞬态存储器中，
//但它也缺少ns2:c1，并且数据不存在于临时存储中，而是存在于对等存储中。
//此外，协调器应该传递ORG1的背书人标识，而不是ORG2的背书人标识，因为
//memberorgs（）调用不返回org2，只返回org0和org1。
	fetcher.On("fetch", mock.Anything).expectingDigests([]privdatacommon.DigKey{
		{
			TxId: "tx1", Namespace: "ns1", Collection: "c2", BlockSeq: 1,
		},
		{
			TxId: "tx2", Namespace: "ns2", Collection: "c1", BlockSeq: 1, SeqInBlock: 1,
		},
	}).expectingEndorsers("org1").Return(&privdatacommon.FetchedPvtDataContainer{
		AvailableElements: []*proto.PvtDataElement{
			{
				Digest: &proto.PvtDataDigest{
					BlockSeq:   1,
					Collection: "c2",
					Namespace:  "ns1",
					TxId:       "tx1",
				},
				Payload: [][]byte{[]byte("rws-pre-image")},
			},
			{
				Digest: &proto.PvtDataDigest{
					SeqInBlock: 1,
					BlockSeq:   1,
					Collection: "c1",
					Namespace:  "ns2",
					TxId:       "tx2",
				},
				Payload: [][]byte{[]byte("rws-pre-image")},
			},
		},
	}, nil)
	store.On("Persist", mock.Anything, uint64(1), mock.Anything).
		expectRWSet("ns1", "c2", []byte("rws-pre-image")).
		expectRWSet("ns2", "c1", []byte("rws-pre-image")).Return(nil)
	pvtData = pdFactory.addRWSet().addNSRWSet("ns1", "c1").create()
	err = coordinator.StoreBlock(block, pvtData)
	assertPurged("tx1", "tx2")
	assert.NoError(t, err)
	assertCommitHappened()

	fmt.Println("Scenario IV")
//场景四：块附带了足够多的私有数据，其中一些是冗余的。
	pvtData = pdFactory.addRWSet().addNSRWSet("ns1", "c1", "c2", "c3").
		addRWSet().addNSRWSet("ns2", "c1", "c3").addRWSet().addNSRWSet("ns1", "c4").create()
	err = coordinator.StoreBlock(block, pvtData)
	assertPurged("tx1", "tx2")
	assert.NoError(t, err)
	assertCommitHappened()

	fmt.Println("Scenario V")
//场景五：我们得到的块旁边有私有数据，但协调器无法检索收集访问权
//由于数据库不可用错误而导致的收集策略。
//我们验证错误是否正确传播。
	mockCs := &mocks.CollectionStore{}
	mockCs.On("RetrieveCollectionAccessPolicy", mock.Anything).Return(nil, errors.New("test error"))
	coordinator = NewCoordinator(Support{
		CollectionStore: mockCs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err = coordinator.StoreBlock(block, nil)
	assert.Error(t, err)
	assert.Equal(t, "test error", err.Error())

	fmt.Println("Scenario VI")
//场景六：block没有得到任何私有数据，以及临时存储
//有点问题。
//在这种情况下，我们应该尝试从对等端获取数据。
	block = bf.AddTxn("tx3", "ns3", hash, "c3").create()
	fetcher = &fetcherMock{t: t}
	fetcher.On("fetch", mock.Anything).expectingDigests([]privdatacommon.DigKey{
		{
			TxId: "tx3", Namespace: "ns3", Collection: "c3", BlockSeq: 1,
		},
	}).Return(&privdatacommon.FetchedPvtDataContainer{
		AvailableElements: []*proto.PvtDataElement{
			{
				Digest: &proto.PvtDataDigest{
					BlockSeq:   1,
					Collection: "c3",
					Namespace:  "ns3",
					TxId:       "tx3",
				},
				Payload: [][]byte{[]byte("rws-pre-image")},
			},
		},
	}, nil)
	store = &mockTransientStore{t: t}
	store.On("PurgeByTxids", mock.Anything).Run(func(args mock.Arguments) {
		for _, txn := range args.Get(0).([]string) {
			purgedTxns[txn] = struct{}{}
		}
	}).Return(nil)
	store.On("GetTxPvtRWSetByTxid", "tx3", mock.Anything).Return(&mockRWSetScanner{err: errors.New("uh oh")}, nil)
	store.On("Persist", mock.Anything, uint64(1), mock.Anything).expectRWSet("ns3", "c3", []byte("rws-pre-image"))
	committer = &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		privateDataPassed2Ledger := args.Get(0).(*ledger.BlockAndPvtData).PvtData
		assert.True(t, reflect.DeepEqual(flattenTxPvtDataMap(privateDataPassed2Ledger),
			flattenTxPvtDataMap(expectedCommittedPrivateData2)))
		commitHappened = true
	}).Return(nil)
	coordinator = NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err = coordinator.StoreBlock(block, nil)
	assertPurged("tx3")
	assert.NoError(t, err)
	assertCommitHappened()

	fmt.Println("Scenario VII")
//场景七：数据块包含2个事务，对等端仅适用于TX3-NS3-C3。
//此外，这些块还附带一个用于tx3-ns3-c3的私有数据，这样对等端就不必获取
//来自临时存储或对等端的私有数据，实际上-如果它试图获取数据，则不符合条件
//对于来自临时存储或对等存储的-测试将失败，因为模拟未初始化。
	block = bf.AddTxn("tx3", "ns3", hash, "c3", "c2", "c1").AddTxn("tx1", "ns1", hash, "c1").create()
	cs = createcollectionStore(peerSelfSignedData).thatAccepts(CollectionCriteria{
		TxId:       "tx3",
		Collection: "c3",
		Namespace:  "ns3",
		Channel:    "test",
	})
	store = &mockTransientStore{t: t}
	store.On("PurgeByTxids", mock.Anything).Run(func(args mock.Arguments) {
		for _, txn := range args.Get(0).([]string) {
			purgedTxns[txn] = struct{}{}
		}
	}).Return(nil)
	fetcher = &fetcherMock{t: t}
	committer = &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		privateDataPassed2Ledger := args.Get(0).(*ledger.BlockAndPvtData).PvtData
		assert.True(t, reflect.DeepEqual(flattenTxPvtDataMap(privateDataPassed2Ledger),
			flattenTxPvtDataMap(expectedCommittedPrivateData2)))
		commitHappened = true
	}).Return(nil)
	coordinator = NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)

	pvtData = pdFactory.addRWSet().addNSRWSet("ns3", "c3").create()
	err = coordinator.StoreBlock(block, pvtData)
	assert.NoError(t, err)
	assertCommitHappened()
//在任何情况下，块中的所有事务都将从临时存储中清除。
	assertPurged("tx3", "tx1")
}

func TestProceedWithoutPrivateData(t *testing.T) {
//场景：我们缺少私有数据（NS3中的C2），无法从任何对等机获取。
//需要使用缺少的私有数据提交块。
	peerSelfSignedData := common.SignedData{
		Identity:  []byte{0, 1, 2},
		Signature: []byte{3, 4, 5},
		Data:      []byte{6, 7, 8},
	}
	cs := createcollectionStore(peerSelfSignedData).thatAcceptsAll()
	var commitHappened bool
	assertCommitHappened := func() {
		assert.True(t, commitHappened)
		commitHappened = false
	}
	committer := &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		blockAndPrivateData := args.Get(0).(*ledger.BlockAndPvtData)
		privateDataPassed2Ledger := blockAndPrivateData.PvtData
		assert.True(t, reflect.DeepEqual(flattenTxPvtDataMap(privateDataPassed2Ledger),
			flattenTxPvtDataMap(expectedCommittedPrivateData2)))
		missingPrivateData := blockAndPrivateData.MissingPvtData
		expectedMissingPvtData := make(ledger.TxMissingPvtDataMap)
		expectedMissingPvtData.Add(0, "ns3", "c2", true)
		assert.Equal(t, expectedMissingPvtData, missingPrivateData)
		commitHappened = true
	}).Return(nil)
	purgedTxns := make(map[string]struct{})
	store := &mockTransientStore{t: t}
	store.On("GetTxPvtRWSetByTxid", "tx1", mock.Anything).Return(&mockRWSetScanner{}, nil)
	store.On("PurgeByTxids", mock.Anything).Run(func(args mock.Arguments) {
		for _, txn := range args.Get(0).([]string) {
			purgedTxns[txn] = struct{}{}
		}
	}).Return(nil)
	assertPurged := func(txns ...string) {
		for _, txn := range txns {
			_, exists := purgedTxns[txn]
			assert.True(t, exists)
			delete(purgedTxns, txn)
		}
		assert.Len(t, purgedTxns, 0)
	}

	fetcher := &fetcherMock{t: t}
//让对等方返回以响应pull，这是一个带有不匹配哈希的私有数据
	fetcher.On("fetch", mock.Anything).expectingDigests([]privdatacommon.DigKey{
		{
			TxId: "tx1", Namespace: "ns3", Collection: "c2", BlockSeq: 1,
		},
	}).Return(&privdatacommon.FetchedPvtDataContainer{
		AvailableElements: []*proto.PvtDataElement{
			{
				Digest: &proto.PvtDataDigest{
					BlockSeq:   1,
					Collection: "c2",
					Namespace:  "ns3",
					TxId:       "tx1",
				},
				Payload: [][]byte{[]byte("wrong pre-image")},
			},
		},
	}, nil)

	hash := util2.ComputeSHA256([]byte("rws-pre-image"))
	pdFactory := &pvtDataFactory{}
	bf := &blockFactory{
		channelID: "test",
	}

	block := bf.AddTxn("tx1", "ns3", hash, "c3", "c2").create()
	pvtData := pdFactory.addRWSet().addNSRWSet("ns3", "c3").create()
	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err := coordinator.StoreBlock(block, pvtData)
	assert.NoError(t, err)
	assertCommitHappened()
	assertPurged("tx1")
}

func TestProceedWithInEligiblePrivateData(t *testing.T) {
//场景：我们缺少私有数据（NS3中的C2），无法从任何对等机获取。
//需要使用缺少的私有数据提交块。
	peerSelfSignedData := common.SignedData{
		Identity:  []byte{0, 1, 2},
		Signature: []byte{3, 4, 5},
		Data:      []byte{6, 7, 8},
	}

	cs := createcollectionStore(peerSelfSignedData).thatAcceptsNone()

	var commitHappened bool
	assertCommitHappened := func() {
		assert.True(t, commitHappened)
		commitHappened = false
	}
	committer := &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		blockAndPrivateData := args.Get(0).(*ledger.BlockAndPvtData)
		privateDataPassed2Ledger := blockAndPrivateData.PvtData
		assert.True(t, reflect.DeepEqual(flattenTxPvtDataMap(privateDataPassed2Ledger),
			flattenTxPvtDataMap(expectedCommittedPrivateData3)))
		missingPrivateData := blockAndPrivateData.MissingPvtData
		expectedMissingPvtData := make(ledger.TxMissingPvtDataMap)
		expectedMissingPvtData.Add(0, "ns3", "c2", false)
		assert.Equal(t, expectedMissingPvtData, missingPrivateData)
		commitHappened = true
	}).Return(nil)

	hash := util2.ComputeSHA256([]byte("rws-pre-image"))
	bf := &blockFactory{
		channelID: "test",
	}

	block := bf.AddTxn("tx1", "ns3", hash, "c2").create()

	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         nil,
		TransientStore:  nil,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
	err := coordinator.StoreBlock(block, nil)
	assert.NoError(t, err)
	assertCommitHappened()
}

func TestCoordinatorGetBlocks(t *testing.T) {
	sd := common.SignedData{
		Identity:  []byte{0, 1, 2},
		Signature: []byte{3, 4, 5},
		Data:      []byte{6, 7, 8},
	}
	cs := createcollectionStore(sd).thatAcceptsAll()
	committer := &mocks.Committer{}
	store := &mockTransientStore{t: t}
	fetcher := &fetcherMock{t: t}
	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, sd)

	hash := util2.ComputeSHA256([]byte("rws-pre-image"))
	bf := &blockFactory{
		channelID: "test",
	}
	block := bf.AddTxn("tx1", "ns1", hash, "c1", "c2").AddTxn("tx2", "ns2", hash, "c1").create()

//绿色路径-返回块和私有数据，但请求者不具备所有私有数据的资格，
//但只限于其中的一个子集。
	cs = createcollectionStore(sd).thatAccepts(CollectionCriteria{
		Namespace:  "ns1",
		Collection: "c2",
		TxId:       "tx1",
		Channel:    "test",
	})
	committer.Mock = mock.Mock{}
	committer.On("GetPvtDataAndBlockByNum", mock.Anything).Return(&ledger.BlockAndPvtData{
		Block:   block,
		PvtData: expectedCommittedPrivateData1,
	}, nil)
	coordinator = NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, sd)
	expectedPrivData := (&pvtDataFactory{}).addRWSet().addNSRWSet("ns1", "c2").create()
	block2, returnedPrivateData, err := coordinator.GetPvtDataAndBlockByNum(1, sd)
	assert.NoError(t, err)
	assert.Equal(t, block, block2)
	assert.Equal(t, expectedPrivData, []*ledger.TxPvtData(returnedPrivateData))

//错误路径-尝试检索块和私有数据时出错
	committer.Mock = mock.Mock{}
	committer.On("GetPvtDataAndBlockByNum", mock.Anything).Return(nil, errors.New("uh oh"))
	block2, returnedPrivateData, err = coordinator.GetPvtDataAndBlockByNum(1, sd)
	assert.Nil(t, block2)
	assert.Empty(t, returnedPrivateData)
	assert.Error(t, err)
}

func TestPurgeByHeight(t *testing.T) {
//场景：提交3000个块并确保调用purgeByHeight
//在提交块2000和3000时，最大块保留值为1000和2000
	peerSelfSignedData := common.SignedData{}
	cs := createcollectionStore(peerSelfSignedData).thatAcceptsAll()

	var purgeHappened bool
	assertPurgeHappened := func() {
		assert.True(t, purgeHappened)
		purgeHappened = false
	}
	committer := &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Return(nil)
	store := &mockTransientStore{t: t}
	store.On("PurgeByHeight", uint64(1000)).Return(nil).Once().Run(func(_ mock.Arguments) {
		purgeHappened = true
	})
	store.On("PurgeByHeight", uint64(2000)).Return(nil).Once().Run(func(_ mock.Arguments) {
		purgeHappened = true
	})
	store.On("PurgeByTxids", mock.Anything).Return(nil)
	fetcher := &fetcherMock{t: t}

	bf := &blockFactory{
		channelID: "test",
	}
	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)

	for i := 0; i <= 3000; i++ {
		block := bf.create()
		block.Header.Number = uint64(i)
		err := coordinator.StoreBlock(block, nil)
		assert.NoError(t, err)
		if i != 2000 && i != 3000 {
			assert.False(t, purgeHappened)
		} else {
			assertPurgeHappened()
		}
	}
}

func TestCoordinatorStorePvtData(t *testing.T) {
	cs := createcollectionStore(common.SignedData{}).thatAcceptsAll()
	committer := &mocks.Committer{}
	store := &mockTransientStore{t: t}
	store.On("PersistWithConfig", mock.Anything, uint64(5), mock.Anything).
		expectRWSet("ns1", "c1", []byte("rws-pre-image")).Return(nil)
	fetcher := &fetcherMock{t: t}
	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, common.SignedData{})
	pvtData := (&pvtDataFactory{}).addRWSet().addNSRWSet("ns1", "c1").create()
//绿色路径：可以从分类帐/提交人检索分类帐高度
	err := coordinator.StorePvtData("tx1", &transientstore2.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset:          pvtData[0].WriteSet,
		CollectionConfigs: make(map[string]*common.CollectionConfigPackage),
	}, uint64(5))
	assert.NoError(t, err)
}

func TestContainsWrites(t *testing.T) {
//场景一：集合中无哈希Rwset
	col := &rwsetutil.CollHashedRwSet{
		CollectionName: "col1",
	}
	assert.False(t, containsWrites("tx", "ns", col))

//场景二：集合中没有写入
	col.HashedRwSet = &kvrwset.HashedRWSet{}
	assert.False(t, containsWrites("tx", "ns", col))

//场景三：集合中的一些写入
	col.HashedRwSet.HashedWrites = append(col.HashedRwSet.HashedWrites, &kvrwset.KVWriteHash{})
	assert.True(t, containsWrites("tx", "ns", col))
}

func TestIgnoreReadOnlyColRWSets(t *testing.T) {
//场景：事务有一些仅具有读和写的colrwset，
//这些应该被忽略，而不是认为缺少需要检索的私有数据。
//来自临时存储或其他对等端。
//此测试中的八卦和临时存储模拟未初始化为
//操作，因此如果协调器尝试从
//暂时存储或其他对等，测试将失败。
//另外-我们在提交时检查-协调器得出结论
//找不到丢失的私人数据。
	peerSelfSignedData := common.SignedData{
		Identity:  []byte{0, 1, 2},
		Signature: []byte{3, 4, 5},
		Data:      []byte{6, 7, 8},
	}
	cs := createcollectionStore(peerSelfSignedData).thatAcceptsAll()
	var commitHappened bool
	assertCommitHappened := func() {
		assert.True(t, commitHappened)
		commitHappened = false
	}
	committer := &mocks.Committer{}
	committer.On("CommitWithPvtData", mock.Anything).Run(func(args mock.Arguments) {
		blockAndPrivateData := args.Get(0).(*ledger.BlockAndPvtData)
//确保没有要提交的私有数据
		assert.Empty(t, blockAndPrivateData.PvtData)
//确保没有丢失的私有数据
		assert.Empty(t, blockAndPrivateData.MissingPvtData)
		commitHappened = true
	}).Return(nil)
	store := &mockTransientStore{t: t}

	fetcher := &fetcherMock{t: t}
	hash := util2.ComputeSHA256([]byte("rws-pre-image"))
	bf := &blockFactory{
		channelID: "test",
	}
//该块包含只读私有数据事务
	block := bf.AddReadOnlyTxn("tx1", "ns3", hash, "c3", "c2").create()
	coordinator := NewCoordinator(Support{
		CollectionStore: cs,
		Committer:       committer,
		Fetcher:         fetcher,
		TransientStore:  store,
		Validator:       &validatorMock{},
	}, peerSelfSignedData)
//我们传递一个零的私有数据切片，以指示尽管块包含
//私人数据读取。
	err := coordinator.StoreBlock(block, nil)
	assert.NoError(t, err)
	assertCommitHappened()
}
