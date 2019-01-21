
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


package rwsetutil

import (
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flogging.ActivateSpec("rwsetutil=debug")
	os.Exit(m.Run())
}

func TestTxSimulationResultWithOnlyPubData(t *testing.T) {
	rwSetBuilder := NewRWSetBuilder()

	rwSetBuilder.AddToReadSet("ns1", "key2", version.NewHeight(1, 2))
	rwSetBuilder.AddToReadSet("ns1", "key1", version.NewHeight(1, 1))
	rwSetBuilder.AddToWriteSet("ns1", "key2", []byte("value2"))

	rqi1 := &kvrwset.RangeQueryInfo{StartKey: "bKey", EndKey: "", ItrExhausted: false, ReadsInfo: nil}
	rqi1.EndKey = "eKey"
	rqi1.SetRawReads([]*kvrwset.KVRead{NewKVRead("bKey1", version.NewHeight(2, 3)), NewKVRead("bKey2", version.NewHeight(2, 4))})
	rqi1.ItrExhausted = true
	rwSetBuilder.AddToRangeQuerySet("ns1", rqi1)

	rqi2 := &kvrwset.RangeQueryInfo{StartKey: "bKey", EndKey: "", ItrExhausted: false, ReadsInfo: nil}
	rqi2.EndKey = "eKey"
	rqi2.SetRawReads([]*kvrwset.KVRead{NewKVRead("bKey1", version.NewHeight(2, 3)), NewKVRead("bKey2", version.NewHeight(2, 4))})
	rqi2.ItrExhausted = true
	rwSetBuilder.AddToRangeQuerySet("ns1", rqi2)

	rqi3 := &kvrwset.RangeQueryInfo{StartKey: "bKey", EndKey: "", ItrExhausted: true, ReadsInfo: nil}
	rqi3.EndKey = "eKey1"
	rqi3.SetRawReads([]*kvrwset.KVRead{NewKVRead("bKey1", version.NewHeight(2, 3)), NewKVRead("bKey2", version.NewHeight(2, 4))})
	rwSetBuilder.AddToRangeQuerySet("ns1", rqi3)

	rwSetBuilder.AddToReadSet("ns2", "key2", version.NewHeight(1, 2))
	rwSetBuilder.AddToWriteSet("ns2", "key3", []byte("value3"))

	txSimulationResults, err := rwSetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)

	ns1KVRWSet := &kvrwset.KVRWSet{
		Reads:            []*kvrwset.KVRead{NewKVRead("key1", version.NewHeight(1, 1)), NewKVRead("key2", version.NewHeight(1, 2))},
		RangeQueriesInfo: []*kvrwset.RangeQueryInfo{rqi1, rqi3},
		Writes:           []*kvrwset.KVWrite{newKVWrite("key2", []byte("value2"))}}

	ns1RWSet := &rwset.NsReadWriteSet{
		Namespace: "ns1",
		Rwset:     serializeTestProtoMsg(t, ns1KVRWSet),
	}

	ns2KVRWSet := &kvrwset.KVRWSet{
		Reads:            []*kvrwset.KVRead{NewKVRead("key2", version.NewHeight(1, 2))},
		RangeQueriesInfo: nil,
		Writes:           []*kvrwset.KVWrite{newKVWrite("key3", []byte("value3"))}}

	ns2RWSet := &rwset.NsReadWriteSet{
		Namespace: "ns2",
		Rwset:     serializeTestProtoMsg(t, ns2KVRWSet),
	}

	expectedTxRWSet := &rwset.TxReadWriteSet{NsRwset: []*rwset.NsReadWriteSet{ns1RWSet, ns2RWSet}}
	assert.Equal(t, expectedTxRWSet, txSimulationResults.PubSimulationResults)
	assert.Nil(t, txSimulationResults.PvtSimulationResults)
	assert.Nil(t, txSimulationResults.PubSimulationResults.NsRwset[0].CollectionHashedRwset)
}

func TestTxSimulationResultWithPvtData(t *testing.T) {
	rwSetBuilder := NewRWSetBuilder()
//公共RWS NS1+NS2
	rwSetBuilder.AddToReadSet("ns1", "key1", version.NewHeight(1, 1))
	rwSetBuilder.AddToReadSet("ns2", "key1", version.NewHeight(1, 1))
	rwSetBuilder.AddToWriteSet("ns2", "key1", []byte("ns2-key1-value"))

//PVT RWSET NS1
	rwSetBuilder.AddToHashedReadSet("ns1", "coll1", "key1", version.NewHeight(1, 1))
	rwSetBuilder.AddToHashedReadSet("ns1", "coll2", "key1", version.NewHeight(1, 1))
	rwSetBuilder.AddToPvtAndHashedWriteSet("ns1", "coll2", "key1", []byte("pvt-ns1-coll2-key1-value"))

//PVT RWSET NS2
	rwSetBuilder.AddToHashedReadSet("ns2", "coll1", "key1", version.NewHeight(1, 1))
	rwSetBuilder.AddToHashedReadSet("ns2", "coll1", "key2", version.NewHeight(1, 1))
	rwSetBuilder.AddToPvtAndHashedWriteSet("ns2", "coll2", "key1", []byte("pvt-ns2-coll2-key1-value"))

	actualSimRes, err := rwSetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)

///////////////////////////////////////////
//构建预期的pvt rwset并与txSimulationResults中存在的pvt rwset进行比较
///////////////////////////////////////////
	pvt_Ns1_Coll2 := &kvrwset.KVRWSet{
		Writes: []*kvrwset.KVWrite{newKVWrite("key1", []byte("pvt-ns1-coll2-key1-value"))},
	}

	pvt_Ns2_Coll2 := &kvrwset.KVRWSet{
		Writes: []*kvrwset.KVWrite{newKVWrite("key1", []byte("pvt-ns2-coll2-key1-value"))},
	}

	expectedPvtRWSet := &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "ns1",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "coll2",
						Rwset:          serializeTestProtoMsg(t, pvt_Ns1_Coll2),
					},
				},
			},

			{
				Namespace: "ns2",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "coll2",
						Rwset:          serializeTestProtoMsg(t, pvt_Ns2_Coll2),
					},
				},
			},
		},
	}
	assert.Equal(t, expectedPvtRWSet, actualSimRes.PvtSimulationResults)

///////////////////////////////////////////
//构建公共RWset（将是块的一部分），并与TxSimulationResults中存在的RWset进行比较
///////////////////////////////////////////
	pub_Ns1 := &kvrwset.KVRWSet{
		Reads: []*kvrwset.KVRead{NewKVRead("key1", version.NewHeight(1, 1))},
	}

	pub_Ns2 := &kvrwset.KVRWSet{
		Reads:  []*kvrwset.KVRead{NewKVRead("key1", version.NewHeight(1, 1))},
		Writes: []*kvrwset.KVWrite{newKVWrite("key1", []byte("ns2-key1-value"))},
	}

	hashed_Ns1_Coll1 := &kvrwset.HashedRWSet{
		HashedReads: []*kvrwset.KVReadHash{
			constructTestPvtKVReadHash(t, "key1", version.NewHeight(1, 1))},
	}

	hashed_Ns1_Coll2 := &kvrwset.HashedRWSet{
		HashedReads: []*kvrwset.KVReadHash{
			constructTestPvtKVReadHash(t, "key1", version.NewHeight(1, 1))},
		HashedWrites: []*kvrwset.KVWriteHash{
			constructTestPvtKVWriteHash(t, "key1", []byte("pvt-ns1-coll2-key1-value")),
		},
	}

	hashed_Ns2_Coll1 := &kvrwset.HashedRWSet{
		HashedReads: []*kvrwset.KVReadHash{
			constructTestPvtKVReadHash(t, "key1", version.NewHeight(1, 1)),
			constructTestPvtKVReadHash(t, "key2", version.NewHeight(1, 1)),
		},
	}

	hashed_Ns2_Coll2 := &kvrwset.HashedRWSet{
		HashedWrites: []*kvrwset.KVWriteHash{
			constructTestPvtKVWriteHash(t, "key1", []byte("pvt-ns2-coll2-key1-value")),
		},
	}

	combined_Ns1 := &rwset.NsReadWriteSet{
		Namespace: "ns1",
		Rwset:     serializeTestProtoMsg(t, pub_Ns1),
		CollectionHashedRwset: []*rwset.CollectionHashedReadWriteSet{
			{
				CollectionName: "coll1",
				HashedRwset:    serializeTestProtoMsg(t, hashed_Ns1_Coll1),
			},
			{
				CollectionName: "coll2",
				HashedRwset:    serializeTestProtoMsg(t, hashed_Ns1_Coll2),
				PvtRwsetHash:   util.ComputeHash(serializeTestProtoMsg(t, pvt_Ns1_Coll2)),
			},
		},
	}
	assert.Equal(t, combined_Ns1, actualSimRes.PubSimulationResults.NsRwset[0])

	combined_Ns2 := &rwset.NsReadWriteSet{
		Namespace: "ns2",
		Rwset:     serializeTestProtoMsg(t, pub_Ns2),
		CollectionHashedRwset: []*rwset.CollectionHashedReadWriteSet{
			{
				CollectionName: "coll1",
				HashedRwset:    serializeTestProtoMsg(t, hashed_Ns2_Coll1),
			},
			{
				CollectionName: "coll2",
				HashedRwset:    serializeTestProtoMsg(t, hashed_Ns2_Coll2),
				PvtRwsetHash:   util.ComputeHash(serializeTestProtoMsg(t, pvt_Ns2_Coll2)),
			},
		},
	}
	assert.Equal(t, combined_Ns2, actualSimRes.PubSimulationResults.NsRwset[1])
	expectedPubRWSet := &rwset.TxReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsRwset:   []*rwset.NsReadWriteSet{combined_Ns1, combined_Ns2},
	}
	assert.Equal(t, expectedPubRWSet, actualSimRes.PubSimulationResults)
}

func TestTxSimulationResultWithMetadata(t *testing.T) {
	rwSetBuilder := NewRWSetBuilder()
//公共RWS NS1
	rwSetBuilder.AddToReadSet("ns1", "key1", version.NewHeight(1, 1))
	rwSetBuilder.AddToMetadataWriteSet("ns1", "key1",
		map[string][]byte{"metadata2": []byte("ns1-key1-metadata2"), "metadata1": []byte("ns1-key1-metadata1")},
	)
//公共RWS NS2
	rwSetBuilder.AddToWriteSet("ns2", "key1", []byte("ns2-key1-value"))
rwSetBuilder.AddToMetadataWriteSet("ns2", "key1", map[string][]byte{}) //零/空映射表示元数据删除

//pvt rwset<ns1，coll1>
	rwSetBuilder.AddToPvtAndHashedWriteSet("ns1", "coll1", "key1", []byte("pvt-ns1-coll1-key1-value"))
	rwSetBuilder.AddToHashedMetadataWriteSet("ns1", "coll1", "key1",
		map[string][]byte{"metadata1": []byte("ns1-coll1-key1-metadata1")})

//pvt rwset<ns1，coll2>
rwSetBuilder.AddToHashedMetadataWriteSet("ns1", "coll2", "key1", nil) //pvt数据元数据删除

	actualSimRes, err := rwSetBuilder.GetTxSimulationResults()
	assert.NoError(t, err)

//构建预期的pvt rwset并与txSimulationResults中存在的pvt rwset进行比较
	pvtNs1Coll1 := &kvrwset.KVRWSet{
		Writes:         []*kvrwset.KVWrite{newKVWrite("key1", []byte("pvt-ns1-coll1-key1-value"))},
		MetadataWrites: []*kvrwset.KVMetadataWrite{{Key: "key1"}},
	}

	pvtNs1Coll2 := &kvrwset.KVRWSet{
		MetadataWrites: []*kvrwset.KVMetadataWrite{{Key: "key1"}},
	}

	expectedPvtRWSet := &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "ns1",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "coll1",
						Rwset:          serializeTestProtoMsg(t, pvtNs1Coll1),
					},
					{
						CollectionName: "coll2",
						Rwset:          serializeTestProtoMsg(t, pvtNs1Coll2),
					},
				},
			},
		},
	}
	assert.Equal(t, expectedPvtRWSet, actualSimRes.PvtSimulationResults)
//构造public和hashed rwset（将是块的一部分），并与txSimulationResults中存在的rwset进行比较
	pubNs1 := &kvrwset.KVRWSet{
		Reads: []*kvrwset.KVRead{NewKVRead("key1", version.NewHeight(1, 1))},
		MetadataWrites: []*kvrwset.KVMetadataWrite{
			{
				Key: "key1",
				Entries: []*kvrwset.KVMetadataEntry{
					{Name: "metadata1", Value: []byte("ns1-key1-metadata1")},
					{Name: "metadata2", Value: []byte("ns1-key1-metadata2")},
				},
			},
		},
	}

	pubNs2 := &kvrwset.KVRWSet{
		Writes: []*kvrwset.KVWrite{newKVWrite("key1", []byte("ns2-key1-value"))},
		MetadataWrites: []*kvrwset.KVMetadataWrite{
			{
				Key:     "key1",
				Entries: nil,
			},
		},
	}

	hashedNs1Coll1 := &kvrwset.HashedRWSet{
		HashedWrites: []*kvrwset.KVWriteHash{
			constructTestPvtKVWriteHash(t, "key1", []byte("pvt-ns1-coll1-key1-value")),
		},
		MetadataWrites: []*kvrwset.KVMetadataWriteHash{
			{
				KeyHash: util.ComputeStringHash("key1"),
				Entries: []*kvrwset.KVMetadataEntry{
					{Name: "metadata1", Value: []byte("ns1-coll1-key1-metadata1")},
				},
			},
		},
	}

	hashedNs1Coll2 := &kvrwset.HashedRWSet{
		MetadataWrites: []*kvrwset.KVMetadataWriteHash{
			{
				KeyHash: util.ComputeStringHash("key1"),
				Entries: nil,
			},
		},
	}

	pubAndHashCombinedNs1 := &rwset.NsReadWriteSet{
		Namespace: "ns1",
		Rwset:     serializeTestProtoMsg(t, pubNs1),
		CollectionHashedRwset: []*rwset.CollectionHashedReadWriteSet{
			{
				CollectionName: "coll1",
				HashedRwset:    serializeTestProtoMsg(t, hashedNs1Coll1),
				PvtRwsetHash:   util.ComputeHash(serializeTestProtoMsg(t, pvtNs1Coll1)),
			},
			{
				CollectionName: "coll2",
				HashedRwset:    serializeTestProtoMsg(t, hashedNs1Coll2),
				PvtRwsetHash:   util.ComputeHash(serializeTestProtoMsg(t, pvtNs1Coll2)),
			},
		},
	}
	assert.Equal(t, pubAndHashCombinedNs1, actualSimRes.PubSimulationResults.NsRwset[0])
	pubAndHashCombinedNs2 := &rwset.NsReadWriteSet{
		Namespace:             "ns2",
		Rwset:                 serializeTestProtoMsg(t, pubNs2),
		CollectionHashedRwset: nil,
	}
	expectedPubRWSet := &rwset.TxReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsRwset:   []*rwset.NsReadWriteSet{pubAndHashCombinedNs1, pubAndHashCombinedNs2},
	}
	assert.Equal(t, expectedPubRWSet, actualSimRes.PubSimulationResults)
}

func constructTestPvtKVReadHash(t *testing.T, key string, version *version.Height) *kvrwset.KVReadHash {
	kvReadHash := newPvtKVReadHash(key, version)
	return kvReadHash
}

func constructTestPvtKVWriteHash(t *testing.T, key string, value []byte) *kvrwset.KVWriteHash {
	_, kvWriteHash := newPvtKVWriteAndHash(key, value)
	return kvWriteHash
}

func serializeTestProtoMsg(t *testing.T, protoMsg proto.Message) []byte {
	msgBytes, err := proto.Marshal(protoMsg)
	assert.NoError(t, err)
	return msgBytes
}
