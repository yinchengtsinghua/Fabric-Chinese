
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *
 *版权所有IBM公司。保留所有权利。
 *
 *SPDX许可证标识符：Apache-2.0
 */
 *
 **/


package endorser

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

//CollectionConfigRetriever是CollectionConfigRetriever类型的模拟类型
type mockCollectionConfigRetriever struct {
	mock.Mock
}

//GetState提供了一个具有给定字段的模拟函数：命名空间、键
func (_m *mockCollectionConfigRetriever) GetState(namespace string, key string) ([]byte, error) {
	result := _m.Called(namespace, key)
	return result.Get(0).([]byte), result.Error(1)
}

func TestAssemblePvtRWSet(t *testing.T) {
	collectionsConfigCC1 := &common.CollectionConfigPackage{
		Config: []*common.CollectionConfig{
			{
				Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name: "mycollection-1",
					},
				},
			},
			{
				Payload: &common.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &common.StaticCollectionConfig{
						Name: "mycollection-2",
					},
				},
			},
		},
	}
	colB, err := proto.Marshal(collectionsConfigCC1)
	assert.NoError(t, err)

	configRetriever := &mockCollectionConfigRetriever{}
	configRetriever.On("GetState", "lscc", privdata.BuildCollectionKVSKey("myCC")).Return(colB, nil)

	assembler := rwSetAssembler{}

	privData := &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsPvtRwset: []*rwset.NsPvtReadWriteSet{
			{
				Namespace: "myCC",
				CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
					{
						CollectionName: "mycollection-1",
						Rwset:          []byte{1, 2, 3, 4, 5, 6, 7, 8},
					},
				},
			},
		},
	}

	pvtReadWriteSetWithConfigInfo, err := assembler.AssemblePvtRWSet(privData, configRetriever)
	assert.NoError(t, err)
	assert.NotNil(t, pvtReadWriteSetWithConfigInfo)
	assert.NotNil(t, pvtReadWriteSetWithConfigInfo.PvtRwset)
	configPackages := pvtReadWriteSetWithConfigInfo.CollectionConfigs
	assert.NotNil(t, configPackages)
	configs, found := configPackages["myCC"]
	assert.True(t, found)
	assert.Equal(t, 1, len(configs.Config))
	assert.NotNil(t, configs.Config[0])
	assert.NotNil(t, configs.Config[0].GetStaticCollectionConfig())
	assert.Equal(t, "mycollection-1", configs.Config[0].GetStaticCollectionConfig().Name)
	assert.Equal(t, 1, len(pvtReadWriteSetWithConfigInfo.PvtRwset.NsPvtRwset))

}
