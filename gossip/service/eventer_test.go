
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


package service

import (
	"reflect"
	"testing"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/peer"
)

const testChainID = "foo"

func init() {
	util.SetupTestLogging()
}

type mockReceiver struct {
	orgs     map[string]channelconfig.ApplicationOrg
	sequence uint64
}

func (mr *mockReceiver) updateAnchors(config Config) {
	logger.Debugf("[TEST] Setting config to %d %v", config.Sequence(), config.Organizations())
	mr.orgs = config.Organizations()
	mr.sequence = config.Sequence()
}

func (mr *mockReceiver) updateEndpoints(chainID string, endpoints []string) {
}

type mockConfig mockReceiver

func (mc *mockConfig) OrdererAddresses() []string {
	return []string{"localhost:7050"}
}

func (mc *mockConfig) Sequence() uint64 {
	return mc.sequence
}

func (mc *mockConfig) Organizations() map[string]channelconfig.ApplicationOrg {
	return mc.orgs
}

func (mc *mockConfig) ChainID() string {
	return testChainID
}

const testOrgID = "testID"

func TestInitialUpdate(t *testing.T) {
	mc := &mockConfig{
		sequence: 7,
		orgs: map[string]channelconfig.ApplicationOrg{
			testOrgID: &appGrp{
				anchorPeers: []*peer.AnchorPeer{{Port: 9}},
			},
		},
	}

	mr := &mockReceiver{}

	ce := newConfigEventer(mr)
	ce.ProcessConfigUpdate(mc)

	if !reflect.DeepEqual(mc, (*mockConfig)(mr)) {
		t.Fatalf("Should have updated config on initial update but did not")
	}
}

func TestSecondUpdate(t *testing.T) {
	appGrps := map[string]channelconfig.ApplicationOrg{
		testOrgID: &appGrp{
			anchorPeers: []*peer.AnchorPeer{{Port: 9}},
		},
	}
	mc := &mockConfig{
		sequence: 7,
		orgs:     appGrps,
	}

	mr := &mockReceiver{}

	ce := newConfigEventer(mr)
	ce.ProcessConfigUpdate(mc)

	mc.sequence = 8
	appGrps[testOrgID] = &appGrp{
		anchorPeers: []*peer.AnchorPeer{{Port: 10}},
	}

	ce.ProcessConfigUpdate(mc)

	if !reflect.DeepEqual(mc, (*mockConfig)(mr)) {
		t.Fatal("Should have updated config on initial update but did not")
	}
}

func TestSecondSameUpdate(t *testing.T) {
	mc := &mockConfig{
		sequence: 7,
		orgs: map[string]channelconfig.ApplicationOrg{
			testOrgID: &appGrp{
				anchorPeers: []*peer.AnchorPeer{{Port: 9}},
			},
		},
	}

	mr := &mockReceiver{}

	ce := newConfigEventer(mr)
	ce.ProcessConfigUpdate(mc)
	mr.sequence = 0
	mr.orgs = nil
	ce.ProcessConfigUpdate(mc)

	if mr.sequence != 0 {
		t.Error("Should not have updated sequence when reprocessing same config")
	}

	if mr.orgs != nil {
		t.Error("Should not have updated anchor peers when reprocessing same config")
	}
}

func TestUpdatedSeqOnly(t *testing.T) {
	mc := &mockConfig{
		sequence: 7,
		orgs: map[string]channelconfig.ApplicationOrg{
			testOrgID: &appGrp{
				anchorPeers: []*peer.AnchorPeer{{Port: 9}},
			},
		},
	}

	mr := &mockReceiver{}

	ce := newConfigEventer(mr)
	ce.ProcessConfigUpdate(mc)
	mc.sequence = 9
	ce.ProcessConfigUpdate(mc)

	if mr.sequence != 7 {
		t.Errorf("Should not have updated sequence when reprocessing same config")
	}

	if !reflect.DeepEqual(mr.orgs, mc.orgs) {
		t.Errorf("Should not have cleared anchor peers when reprocessing newer config with higher sequence")
	}
}
