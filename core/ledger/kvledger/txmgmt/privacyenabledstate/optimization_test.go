
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


package privacyenabledstate

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/bookkeeping"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/mock"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/stretchr/testify/assert"
)

func TestMetadataHintCorrectness(t *testing.T) {
	bookkeepingTestEnv := bookkeeping.NewTestEnv(t)
	defer bookkeepingTestEnv.Cleanup()
	bookkeeper := bookkeepingTestEnv.TestProvider.GetDBHandle("ledger1", bookkeeping.MetadataPresenceIndicator)

	metadataHint := newMetadataHint(bookkeeper)
	assert.False(t, metadataHint.metadataEverUsedFor("ns1"))

	updates := NewUpdateBatch()
	updates.PubUpdates.PutValAndMetadata("ns1", "key", []byte("value"), []byte("metadata"), version.NewHeight(1, 1))
	updates.PubUpdates.PutValAndMetadata("ns2", "key", []byte("value"), []byte("metadata"), version.NewHeight(1, 2))
	updates.PubUpdates.PutValAndMetadata("ns3", "key", []byte("value"), nil, version.NewHeight(1, 3))
	updates.HashUpdates.PutValAndMetadata("ns1_pvt", "key", "coll", []byte("value"), []byte("metadata"), version.NewHeight(1, 1))
	updates.HashUpdates.PutValAndMetadata("ns2_pvt", "key", "coll", []byte("value"), []byte("metadata"), version.NewHeight(1, 3))
	updates.HashUpdates.PutValAndMetadata("ns3_pvt", "key", "coll", []byte("value"), nil, version.NewHeight(1, 3))
	metadataHint.setMetadataUsedFlag(updates)

	t.Run("MetadataAddedInCurrentSession", func(t *testing.T) {
		assert.True(t, metadataHint.metadataEverUsedFor("ns1"))
		assert.True(t, metadataHint.metadataEverUsedFor("ns2"))
		assert.True(t, metadataHint.metadataEverUsedFor("ns1_pvt"))
		assert.True(t, metadataHint.metadataEverUsedFor("ns2_pvt"))
		assert.False(t, metadataHint.metadataEverUsedFor("ns3"))
		assert.False(t, metadataHint.metadataEverUsedFor("ns4"))
	})

	t.Run("MetadataFromPersistence", func(t *testing.T) {
		metadataHintFromPersistence := newMetadataHint(bookkeeper)
		assert.True(t, metadataHintFromPersistence.metadataEverUsedFor("ns1"))
		assert.True(t, metadataHintFromPersistence.metadataEverUsedFor("ns2"))
		assert.True(t, metadataHintFromPersistence.metadataEverUsedFor("ns1_pvt"))
		assert.True(t, metadataHintFromPersistence.metadataEverUsedFor("ns2_pvt"))
		assert.False(t, metadataHintFromPersistence.metadataEverUsedFor("ns3"))
		assert.False(t, metadataHintFromPersistence.metadataEverUsedFor("ns4"))
	})
}

func TestMetadataHintOptimizationSkippingGoingToDB(t *testing.T) {
	bookkeepingTestEnv := bookkeeping.NewTestEnv(t)
	defer bookkeepingTestEnv.Cleanup()
	bookkeeper := bookkeepingTestEnv.TestProvider.GetDBHandle("ledger1", bookkeeping.MetadataPresenceIndicator)

	mockVersionedDB := &mock.VersionedDB{}
	db, err := NewCommonStorageDB(mockVersionedDB, "testledger", newMetadataHint(bookkeeper))
	assert.NoError(t, err)
	updates := NewUpdateBatch()
	updates.PubUpdates.PutValAndMetadata("ns1", "key", []byte("value"), []byte("metadata"), version.NewHeight(1, 1))
	updates.PubUpdates.PutValAndMetadata("ns2", "key", []byte("value"), nil, version.NewHeight(1, 2))
	db.ApplyPrivacyAwareUpdates(updates, version.NewHeight(1, 3))

	db.GetStateMetadata("ns1", "randomkey")
	assert.Equal(t, 1, mockVersionedDB.GetStateCallCount())
	db.GetPrivateDataMetadataByHash("ns1", "randomColl", []byte("randomKeyhash"))
	assert.Equal(t, 2, mockVersionedDB.GetStateCallCount())

	db.GetStateMetadata("ns2", "randomkey")
	db.GetPrivateDataMetadataByHash("ns2", "randomColl", []byte("randomKeyhash"))
	assert.Equal(t, 2, mockVersionedDB.GetStateCallCount())

	db.GetStateMetadata("randomeNs", "randomkey")
	db.GetPrivateDataMetadataByHash("randomeNs", "randomColl", []byte("randomKeyhash"))
	assert.Equal(t, 2, mockVersionedDB.GetStateCallCount())
}
