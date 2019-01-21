
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
	"sync"

	"github.com/hyperledger/fabric/gossip/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
)

//封装的MembershipStore结构
//成员身份消息存储抽象
type MembershipStore struct {
	m map[string]*proto.SignedGossipMessage
	sync.RWMutex
}

//NewMembershipStore创建新的成员存储实例
func NewMembershipStore() *MembershipStore {
	return &MembershipStore{m: make(map[string]*proto.SignedGossipMessage)}
}

//msgbyid返回由特定ID或nil存储的消息
//如果找不到这样的身份证
func (m *MembershipStore) MsgByID(pkiID common.PKIidType) *proto.SignedGossipMessage {
	m.RLock()
	defer m.RUnlock()
	if msg, exists := m.m[string(pkiID)]; exists {
		return msg
	}
	return nil
}

//会员店规模
func (m *MembershipStore) Size() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.m)
}

//将关联的MSG与给定的PKIID一起放置
func (m *MembershipStore) Put(pkiID common.PKIidType, msg *proto.SignedGossipMessage) {
	m.Lock()
	defer m.Unlock()
	m.m[string(pkiID)] = msg
}

//删除删除具有给定pkiid的邮件
func (m *MembershipStore) Remove(pkiID common.PKIidType) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, string(pkiID))
}

//ToSlice返回由元素支持的切片
//成员关系存储区的
func (m *MembershipStore) ToSlice() []*proto.SignedGossipMessage {
	m.RLock()
	defer m.RUnlock()
	members := make([]*proto.SignedGossipMessage, len(m.m))
	i := 0
	for _, member := range m.m {
		members[i] = member
		i++
	}
	return members
}
