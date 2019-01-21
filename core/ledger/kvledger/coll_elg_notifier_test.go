
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


package kvledger

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/stretchr/testify/assert"
)

func TestCollElgNotifier(t *testing.T) {
	mockDeployedChaincodeInfoProvider := &mock.DeployedChaincodeInfoProvider{}
	mockDeployedChaincodeInfoProvider.UpdatedChaincodesReturns([]*ledger.ChaincodeLifecycleInfo{
		{Name: "cc1"},
	}, nil)

//返回3个集合-bool值指示对等方是否有资格进行相应的集合
	mockDeployedChaincodeInfoProvider.ChaincodeInfoReturnsOnCall(0,
		&ledger.DeployedChaincodeInfo{
			CollectionConfigPkg: testutilPrepapreMockCollectionConfigPkg(
				map[string]bool{"coll1": true, "coll2": true, "coll3": false})}, nil)

//提交后-返回4个集合
	mockDeployedChaincodeInfoProvider.ChaincodeInfoReturnsOnCall(1,
		&ledger.DeployedChaincodeInfo{
			CollectionConfigPkg: testutilPrepapreMockCollectionConfigPkg(
				map[string]bool{"coll1": false, "coll2": true, "coll3": true, "coll4": true})}, nil)

	mockMembershipInfoProvider := &mock.MembershipInfoProvider{}
	mockMembershipInfoProvider.AmMemberOfStub = func(channel string, p *common.CollectionPolicyConfig) (bool, error) {
		return testutilIsEligibleForMockPolicy(p), nil
	}

	mockCollElgListener := &mockCollElgListener{}

	collElgNotifier := &collElgNotifier{
		mockDeployedChaincodeInfoProvider,
		mockMembershipInfoProvider,
		make(map[string]collElgListener),
	}
	collElgNotifier.registerListener("testLedger", mockCollElgListener)

	collElgNotifier.HandleStateUpdates(&ledger.StateUpdateTrigger{
		LedgerID:           "testLedger",
		CommittingBlockNum: uint64(500),
		StateUpdates: map[string]interface{}{
			"doesNotMatterNS": []*kvrwset.KVWrite{
				{
					Key:   "doesNotMatterKey",
					Value: []byte("doesNotMatterVal"),
				},
			},
		},
	})

//触发的事件只应包含“coll3”，因为这是唯一的集合
//通过升级tx，哪个对等机从不合格变为合格
	assert.Equal(t, uint64(500), mockCollElgListener.receivedCommittingBlk)
	assert.Equal(t,
		map[string][]string{
			"cc1": {"coll3"},
		},
		mockCollElgListener.receivedNsCollMap,
	)
}

type mockCollElgListener struct {
	receivedCommittingBlk uint64
	receivedNsCollMap     map[string][]string
}

func (m *mockCollElgListener) ProcessCollsEligibilityEnabled(commitingBlk uint64, nsCollMap map[string][]string) error {
	m.receivedCommittingBlk = commitingBlk
	m.receivedNsCollMap = nsCollMap
	return nil
}

func testutilPrepapreMockCollectionConfigPkg(collEligibilityMap map[string]bool) *common.CollectionConfigPackage {
	pkg := &common.CollectionConfigPackage{}
	for collName, isEligible := range collEligibilityMap {
		var version int32
		if isEligible {
			version = 1
		}
		policy := &common.CollectionPolicyConfig{
			Payload: &common.CollectionPolicyConfig_SignaturePolicy{
				SignaturePolicy: &common.SignaturePolicyEnvelope{Version: version},
			},
		}
		sCollConfig := &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:             collName,
				MemberOrgsPolicy: policy,
			},
		}
		config := &common.CollectionConfig{Payload: sCollConfig}
		pkg.Config = append(pkg.Config, config)
	}
	return pkg
}

func testutilIsEligibleForMockPolicy(p *common.CollectionPolicyConfig) bool {
	return p.GetSignaturePolicy().Version == 1
}
