
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


package util

import (
	"testing"

	"github.com/hyperledger/fabric/gossip/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
)

func init() {
	SetupTestLogging()
}

func TestMembershipStore(t *testing.T) {
	membershipStore := NewMembershipStore()

	id1 := common.PKIidType("id1")
	id2 := common.PKIidType("id2")

	msg1 := &proto.SignedGossipMessage{}
	msg2 := &proto.SignedGossipMessage{Envelope: &proto.Envelope{}}

//最初创建的测试存储为空
	assert.Nil(t, membershipStore.MsgByID(id1))
	assert.Equal(t, membershipStore.Size(), 0)
//按预期进行测试
	membershipStore.Put(id1, msg1)
	assert.NotNil(t, membershipStore.MsgByID(id1))
//test msgbyid返回存储的正确实例
	membershipStore.Put(id2, msg2)
	assert.Equal(t, msg1, membershipStore.MsgByID(id1))
	assert.NotEqual(t, msg2, membershipStore.MsgByID(id1))
//测试容量增长
	assert.Equal(t, membershipStore.Size(), 2)
//试验拆除工程
	membershipStore.Remove(id1)
	assert.Nil(t, membershipStore.MsgByID(id1))
	assert.Equal(t, membershipStore.Size(), 1)
//测试返回的实例不是副本
	msg3 := &proto.SignedGossipMessage{GossipMessage: &proto.GossipMessage{}}
	msg3Clone := &proto.SignedGossipMessage{GossipMessage: &proto.GossipMessage{}}
	id3 := common.PKIidType("id3")
	membershipStore.Put(id3, msg3)
	assert.Equal(t, msg3Clone, msg3)
	membershipStore.MsgByID(id3).Channel = []byte{0, 1, 2, 3}
	assert.NotEqual(t, msg3Clone, msg3)
}

func TestToSlice(t *testing.T) {
	membershipStore := NewMembershipStore()
	id1 := common.PKIidType("id1")
	id2 := common.PKIidType("id2")
	id3 := common.PKIidType("id3")
	id4 := common.PKIidType("id4")

	msg1 := &proto.SignedGossipMessage{}
	msg2 := &proto.SignedGossipMessage{Envelope: &proto.Envelope{}}
	msg3 := &proto.SignedGossipMessage{GossipMessage: &proto.GossipMessage{}}
	msg4 := &proto.SignedGossipMessage{GossipMessage: &proto.GossipMessage{}, Envelope: &proto.Envelope{}}

	membershipStore.Put(id1, msg1)
	membershipStore.Put(id2, msg2)
	membershipStore.Put(id3, msg3)
	membershipStore.Put(id4, msg4)

	assert.Len(t, membershipStore.ToSlice(), 4)

	existsInSlice := func(slice []*proto.SignedGossipMessage, msg *proto.SignedGossipMessage) bool {
		for _, m := range slice {
			if assert.ObjectsAreEqual(m, msg) {
				return true
			}
		}
		return false
	}

	expectedMsgs := []*proto.SignedGossipMessage{msg1, msg2, msg3, msg4}
	for _, msg := range membershipStore.ToSlice() {
		assert.True(t, existsInSlice(expectedMsgs, msg))
	}

}
