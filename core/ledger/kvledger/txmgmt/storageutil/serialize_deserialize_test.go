
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


package storageutil

import (
	"testing"

	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/stretchr/testify/assert"
)

func TestSerializeDeSerialize(t *testing.T) {
	sampleMetadata := []*kvrwset.KVMetadataEntry{
		{Name: "metadata_1", Value: []byte("metadata_value_1")},
		{Name: "metadata_2", Value: []byte("metadata_value_2")},
		{Name: "metadata_3", Value: []byte("metadata_value_3")},
	}

	serializedMetadata, err := SerializeMetadata(sampleMetadata)
	assert.NoError(t, err)
	metadataMap, err := DeserializeMetadata(serializedMetadata)
	assert.NoError(t, err)
	assert.Len(t, metadataMap, 3)
	assert.Equal(t, []byte("metadata_value_1"), metadataMap["metadata_1"])
	assert.Equal(t, []byte("metadata_value_2"), metadataMap["metadata_2"])
	assert.Equal(t, []byte("metadata_value_3"), metadataMap["metadata_3"])
}
