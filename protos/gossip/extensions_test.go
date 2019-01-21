
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package gossip

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/stretchr/testify/assert"
)

func TestToGossipMessageNilEnvelope(t *testing.T) {
	memReq := &MembershipRequest{}
	_, err := memReq.SelfInformation.ToGossipMessage()
	assert.EqualError(t, err, "nil envelope")
}

func TestToString(t *testing.T) {
//确保当我们
//日志消息。
//每个负载或签名都包含“2”，因此我们将记录
//如果不是用于在SignedGossipMessage中重载string（）方法，则返回它们

//下一行证明了本试验中构造的信封
//打印时有“2”
	assert.Contains(t, fmt.Sprintf("%v", envelopes()[0]), "2")
//对于有效载荷，以下内容同样适用：
	dMsg := &DataMessage{
		Payload: &Payload{
			SeqNum: 3,
			Data:   []byte{2, 2, 2, 2, 2},
		},
	}
	assert.Contains(t, fmt.Sprintf("%v", dMsg), "2")

//现在我们构建所有类型的消息，其中包含信封或有效负载
//即使找到“2”，也不要将其输出到其格式中
//作为外部消息的子消息。

	sMsg := &SignedGossipMessage{
		GossipMessage: &GossipMessage{
			Tag:     GossipMessage_EMPTY,
			Nonce:   5,
			Channel: []byte("A"),
			Content: &GossipMessage_DataMsg{
				DataMsg: &DataMessage{
					Payload: &Payload{
						SeqNum: 3,
						Data:   []byte{2, 2, 2, 2, 2},
					},
				},
			},
		},
		Envelope: &Envelope{
			Payload:   []byte{0, 1, 2, 3, 4, 5, 6},
			Signature: []byte{0, 1, 2},
			SecretEnvelope: &SecretEnvelope{
				Payload:   []byte{0, 1, 2, 3, 4, 5},
				Signature: []byte{0, 1, 2},
			},
		},
	}
	assert.NotContains(t, fmt.Sprintf("%v", sMsg), "2")
	sMsg.GetDataMsg().Payload = nil
	assert.NotPanics(t, func() {
		_ = sMsg.String()
	})

	sMsg = &SignedGossipMessage{
		GossipMessage: &GossipMessage{
			Channel: []byte("A"),
			Tag:     GossipMessage_EMPTY,
			Nonce:   5,
			Content: &GossipMessage_DataUpdate{
				DataUpdate: &DataUpdate{
					Nonce:   11,
					MsgType: PullMsgType_BLOCK_MSG,
					Data:    envelopes(),
				},
			},
		},
		Envelope: envelopes()[0],
	}
	assert.NotContains(t, fmt.Sprintf("%v", sMsg), "2")

	sMsg = &SignedGossipMessage{
		GossipMessage: &GossipMessage{
			Channel: []byte("A"),
			Tag:     GossipMessage_EMPTY,
			Nonce:   5,
			Content: &GossipMessage_MemRes{
				MemRes: &MembershipResponse{
					Alive: envelopes(),
					Dead:  envelopes(),
				},
			},
		},
		Envelope: envelopes()[0],
	}
	assert.NotContains(t, fmt.Sprintf("%v", sMsg), "2")

	sMsg = &SignedGossipMessage{
		GossipMessage: &GossipMessage{
			Channel: []byte("A"),
			Tag:     GossipMessage_EMPTY,
			Nonce:   5,
			Content: &GossipMessage_StateSnapshot{
				StateSnapshot: &StateInfoSnapshot{
					Elements: envelopes(),
				},
			},
		},
		Envelope: envelopes()[0],
	}
	assert.NotContains(t, fmt.Sprintf("%v", sMsg), "2")

	sMsg = &SignedGossipMessage{
		GossipMessage: &GossipMessage{
			Channel: []byte("A"),
			Tag:     GossipMessage_EMPTY,
			Nonce:   5,
			Content: &GossipMessage_StateResponse{
				StateResponse: &RemoteStateResponse{
					Payloads: []*Payload{
						{Data: []byte{2, 2, 2}},
					},
				},
			},
		},
		Envelope: envelopes()[0],
	}
	assert.NotContains(t, fmt.Sprintf("%v", sMsg), "2")
}

func TestAliveMessageNoActionTaken(t *testing.T) {
	comparator := NewGossipMessageComparator(1)

	sMsg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{
			Membership: &Member{
				Endpoint: "localhost",
				Metadata: []byte{1, 2, 3, 4, 5},
				PkiId:    []byte{17},
			},
			Timestamp: &PeerTime{
				IncNum: 1,
				SeqNum: 1,
			},
			Identity: []byte("peerID1"),
		},
	})

	sMsg2 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{
			Membership: &Member{
				Endpoint: "localhost",
				Metadata: []byte{1, 2, 3, 4, 5},
				PkiId:    []byte{15},
			},
			Timestamp: &PeerTime{
				IncNum: 2,
				SeqNum: 2,
			},
			Identity: []byte("peerID1"),
		},
	})

	assert.Equal(t, comparator(sMsg1, sMsg2), common.MessageNoAction)
}

func TestStateInfoMessageNoActionTaken(t *testing.T) {
	comparator := NewGossipMessageComparator(1)

//msg1和msg2具有相同的通道MAC，而pkid不同，而
//msg和msg3相同的pkid和不同的通道mac

	sMsg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY,
		stateInfoMessage(1, 1, []byte{17}, []byte{17, 13}))
	sMsg2 := signedGossipMessage("testChannel", GossipMessage_EMPTY,
		stateInfoMessage(1, 1, []byte{13}, []byte{17, 13}))

//我们只应比较可比较的消息，例如来自同一对等方的消息
//在其他情况下，不应采取无效宣告。
	assert.Equal(t, comparator(sMsg1, sMsg2), common.MessageNoAction)
}

func TestStateInfoMessagesInvalidation(t *testing.T) {
	comparator := NewGossipMessageComparator(1)

	sMsg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY,
		stateInfoMessage(1, 1, []byte{17}, []byte{17}))
	sMsg2 := signedGossipMessage("testChannel", GossipMessage_EMPTY,
		stateInfoMessage(1, 1, []byte{17}, []byte{17}))
	sMsg3 := signedGossipMessage("testChannel", GossipMessage_EMPTY,
		stateInfoMessage(1, 2, []byte{17}, []byte{17}))
	sMsg4 := signedGossipMessage("testChannel", GossipMessage_EMPTY,
		stateInfoMessage(2, 1, []byte{17}, []byte{17}))

	assert.Equal(t, comparator(sMsg1, sMsg2), common.MessageInvalidated)

	assert.Equal(t, comparator(sMsg1, sMsg3), common.MessageInvalidated)
	assert.Equal(t, comparator(sMsg3, sMsg1), common.MessageInvalidates)

	assert.Equal(t, comparator(sMsg1, sMsg4), common.MessageInvalidated)
	assert.Equal(t, comparator(sMsg4, sMsg1), common.MessageInvalidates)

	assert.Equal(t, comparator(sMsg3, sMsg4), common.MessageInvalidated)
	assert.Equal(t, comparator(sMsg4, sMsg3), common.MessageInvalidates)
}

func TestAliveMessageInvalidation(t *testing.T) {
	comparator := NewGossipMessageComparator(1)

	sMsg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{
			Membership: &Member{
				Endpoint: "localhost",
				Metadata: []byte{1, 2, 3, 4, 5},
				PkiId:    []byte{17},
			},
			Timestamp: &PeerTime{
				IncNum: 1,
				SeqNum: 1,
			},
			Identity: []byte("peerID1"),
		},
	})

	sMsg2 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{
			Membership: &Member{
				Endpoint: "localhost",
				Metadata: []byte{1, 2, 3, 4, 5},
				PkiId:    []byte{17},
			},
			Timestamp: &PeerTime{
				IncNum: 2,
				SeqNum: 2,
			},
			Identity: []byte("peerID1"),
		},
	})

	sMsg3 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{
			Membership: &Member{
				Endpoint: "localhost",
				Metadata: []byte{1, 2, 3, 4, 5},
				PkiId:    []byte{17},
			},
			Timestamp: &PeerTime{
				IncNum: 1,
				SeqNum: 2,
			},
			Identity: []byte("peerID1"),
		},
	})

	assert.Equal(t, comparator(sMsg1, sMsg2), common.MessageInvalidated)
	assert.Equal(t, comparator(sMsg2, sMsg1), common.MessageInvalidates)
	assert.Equal(t, comparator(sMsg1, sMsg3), common.MessageInvalidated)
	assert.Equal(t, comparator(sMsg3, sMsg1), common.MessageInvalidates)
}

func TestDataMessageInvalidation(t *testing.T) {
	comparator := NewGossipMessageComparator(5)

	data := []byte{1, 1, 1}
	sMsg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY, dataMessage(1, data))
	sMsg1Clone := signedGossipMessage("testChannel", GossipMessage_EMPTY, dataMessage(1, data))
	sMsg3 := signedGossipMessage("testChannel", GossipMessage_EMPTY, dataMessage(2, data))
	sMsg4 := signedGossipMessage("testChannel", GossipMessage_EMPTY, dataMessage(7, data))

	assert.Equal(t, comparator(sMsg1, sMsg1Clone), common.MessageInvalidated)
	assert.Equal(t, comparator(sMsg1, sMsg3), common.MessageNoAction)
	assert.Equal(t, comparator(sMsg1, sMsg4), common.MessageInvalidated)
	assert.Equal(t, comparator(sMsg4, sMsg1), common.MessageInvalidates)
}

func TestIdentityMessagesInvalidation(t *testing.T) {
	comparator := NewGossipMessageComparator(5)

	msg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_PeerIdentity{
		PeerIdentity: &PeerIdentity{
			PkiId:    []byte{17},
			Cert:     []byte{1, 2, 3, 4},
			Metadata: nil,
		},
	})

	msg2 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_PeerIdentity{
		PeerIdentity: &PeerIdentity{
			PkiId:    []byte{17},
			Cert:     []byte{1, 2, 3, 4},
			Metadata: nil,
		},
	})

	msg3 := signedGossipMessage("testChannel", GossipMessage_EMPTY, &GossipMessage_PeerIdentity{
		PeerIdentity: &PeerIdentity{
			PkiId:    []byte{11},
			Cert:     []byte{11, 21, 31, 41},
			Metadata: nil,
		},
	})

	assert.Equal(t, comparator(msg1, msg2), common.MessageInvalidated)
	assert.Equal(t, comparator(msg1, msg3), common.MessageNoAction)
}

func TestLeadershipMessagesNoAction(t *testing.T) {
	comparator := NewGossipMessageComparator(5)

	msg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY, leadershipMessage(1, 1, []byte{17}))
	msg2 := signedGossipMessage("testChannel", GossipMessage_EMPTY, leadershipMessage(1, 1, []byte{11}))

//如果消息具有不同的pkid，则不应采取任何操作
	assert.Equal(t, comparator(msg1, msg2), common.MessageNoAction)
}

func TestLeadershipMessagesInvalidation(t *testing.T) {
	comparator := NewGossipMessageComparator(5)

	pkiID := []byte{17}
	msg1 := signedGossipMessage("testChannel", GossipMessage_EMPTY, leadershipMessage(1, 1, pkiID))
	msg2 := signedGossipMessage("testChannel", GossipMessage_EMPTY, leadershipMessage(1, 2, pkiID))
	msg3 := signedGossipMessage("testChannel", GossipMessage_EMPTY, leadershipMessage(2, 1, pkiID))

//如果消息具有不同的pkid，则不应采取任何操作
	assert.Equal(t, comparator(msg1, msg2), common.MessageInvalidated)
	assert.Equal(t, comparator(msg2, msg1), common.MessageInvalidates)
	assert.Equal(t, comparator(msg1, msg3), common.MessageInvalidated)
	assert.Equal(t, comparator(msg3, msg1), common.MessageInvalidates)
	assert.Equal(t, comparator(msg2, msg3), common.MessageInvalidated)
	assert.Equal(t, comparator(msg3, msg2), common.MessageInvalidates)
}

func TestCheckGossipMessageTypes(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

//创建状态信息拉请求
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_StateInfoPullReq{
		StateInfoPullReq: &StateInfoPullRequest{
			Channel_MAC: []byte{17},
		},
	})

	assert.True(t, msg.IsStateInfoPullRequestMsg())

//创建活动消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{
			Identity: []byte("peerID"),
			Membership: &Member{
				PkiId:    []byte("pkiID"),
				Metadata: []byte{17},
				Endpoint: "localhost",
			},
			Timestamp: &PeerTime{
				SeqNum: 1,
				IncNum: 1,
			},
		},
	})

	assert.True(t, msg.IsAliveMsg())

//创建八卦数据消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, dataMessage(1, []byte{1, 2, 3, 4, 5}))
	assert.True(t, msg.IsDataMsg())

//创建数据请求消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataReq{
		DataReq: &DataRequest{
			MsgType: PullMsgType_UNDEFINED,
			Nonce:   0,
			Digests: [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")},
		},
	})
	assert.True(t, msg.IsDataReq())
	assert.True(t, msg.IsPullMsg())

//创建数据请求消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataDig{
		DataDig: &DataDigest{
			MsgType: PullMsgType_UNDEFINED,
			Nonce:   0,
			Digests: [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")},
		},
	})
	assert.True(t, msg.IsDigestMsg())
	assert.True(t, msg.IsPullMsg())

//创建数据更新消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataUpdate{
		DataUpdate: &DataUpdate{
			MsgType: PullMsgType_UNDEFINED,
			Nonce:   0,
			Data:    []*Envelope{envelopes()[0]},
		},
	})
	assert.True(t, msg.IsDataUpdate())
	assert.True(t, msg.IsPullMsg())

//创建八卦Hello消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_Hello{
		Hello: &GossipHello{
			MsgType: PullMsgType_UNDEFINED,
			Nonce:   0,
		},
	})
	assert.True(t, msg.IsHelloMsg())
	assert.True(t, msg.IsPullMsg())

//创建状态请求消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_StateRequest{
		StateRequest: &RemoteStateRequest{
			StartSeqNum: 1,
			EndSeqNum:   10,
		},
	})
	assert.True(t, msg.IsRemoteStateMessage())

//创建状态响应消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_StateResponse{
		StateResponse: &RemoteStateResponse{
			Payloads: []*Payload{{
				SeqNum: 1,
				Data:   []byte{1, 2, 3, 4, 5},
			}},
		},
	})
	assert.True(t, msg.IsRemoteStateMessage())
}

func TestGossipPullMessageType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

//创建八卦Hello消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_Hello{
		Hello: &GossipHello{
			MsgType: PullMsgType_BLOCK_MSG,
			Nonce:   0,
		},
	})

	assert.True(t, msg.IsHelloMsg())
	assert.True(t, msg.IsPullMsg())
	assert.Equal(t, msg.GetPullMsgType(), PullMsgType_BLOCK_MSG)

//创建数据请求消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataDig{
		DataDig: &DataDigest{
			MsgType: PullMsgType_IDENTITY_MSG,
			Nonce:   0,
			Digests: [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")},
		},
	})
	assert.True(t, msg.IsDigestMsg())
	assert.True(t, msg.IsPullMsg())
	assert.Equal(t, msg.GetPullMsgType(), PullMsgType_IDENTITY_MSG)

//创建数据请求消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataReq{
		DataReq: &DataRequest{
			MsgType: PullMsgType_BLOCK_MSG,
			Nonce:   0,
			Digests: [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")},
		},
	})
	assert.True(t, msg.IsDataReq())
	assert.True(t, msg.IsPullMsg())
	assert.Equal(t, msg.GetPullMsgType(), PullMsgType_BLOCK_MSG)

//创建数据更新消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataUpdate{
		DataUpdate: &DataUpdate{
			MsgType: PullMsgType_IDENTITY_MSG,
			Nonce:   0,
			Data:    []*Envelope{envelopes()[0]},
		},
	})
	assert.True(t, msg.IsDataUpdate())
	assert.True(t, msg.IsPullMsg())
	assert.Equal(t, msg.GetPullMsgType(), PullMsgType_IDENTITY_MSG)

//创建八卦数据消息
	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, dataMessage(1, []byte{1, 2, 3, 4, 5}))
	assert.True(t, msg.IsDataMsg())
	assert.Equal(t, msg.GetPullMsgType(), PullMsgType_UNDEFINED)
}

func TestGossipMessageDataMessageTagType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_AND_ORG, dataMessage(1, []byte{1}))
	assert.True(t, msg.IsChannelRestricted())
	assert.True(t, msg.IsOrgRestricted())
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, dataMessage(1, []byte{1}))
	assert.Error(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_UNDEFINED, dataMessage(1, []byte{1}))
	assert.Error(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_ORG_ONLY, dataMessage(1, []byte{1}))
	assert.False(t, msg.IsChannelRestricted())
	assert.True(t, msg.IsOrgRestricted())

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, dataMessage(1, []byte{1}))
	assert.True(t, msg.IsChannelRestricted())
	assert.False(t, msg.IsOrgRestricted())

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, dataMessage(1, []byte{1}))
	assert.False(t, msg.IsChannelRestricted())
	assert.False(t, msg.IsOrgRestricted())

	msg = signedGossipMessage(channelID, GossipMessage_UNDEFINED, dataMessage(1, []byte{1}))
	assert.False(t, msg.IsChannelRestricted())
	assert.False(t, msg.IsOrgRestricted())
}

func TestGossipMessageAliveMessageTagType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_ORG_ONLY, &GossipMessage_AliveMsg{
		AliveMsg: &AliveMessage{},
	})
	assert.Error(t, msg.IsTagLegal())
}

func TestGossipMessageMembershipMessageTagType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_MemReq{
		MemReq: &MembershipRequest{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_MemRes{
		MemRes: &MembershipResponse{},
	})
	assert.NoError(t, msg.IsTagLegal())
}

func TestGossipMessageIdentityMessageTagType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

	msg = signedGossipMessage(channelID, GossipMessage_ORG_ONLY, &GossipMessage_PeerIdentity{
		PeerIdentity: &PeerIdentity{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_PeerIdentity{
		PeerIdentity: &PeerIdentity{},
	})
	assert.Error(t, msg.IsTagLegal())
}

func TestGossipMessagePullMessageTagType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_AND_ORG, &GossipMessage_DataReq{
		DataReq: &DataRequest{
			MsgType: PullMsgType_BLOCK_MSG,
		},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataReq{
		DataReq: &DataRequest{
			MsgType: PullMsgType_BLOCK_MSG,
		},
	})
	assert.Error(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataDig{
		DataDig: &DataDigest{
			MsgType: PullMsgType_IDENTITY_MSG,
		},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_ORG_ONLY, &GossipMessage_DataDig{
		DataDig: &DataDigest{
			MsgType: PullMsgType_IDENTITY_MSG,
		},
	})
	assert.Error(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_ORG_ONLY, &GossipMessage_DataDig{
		DataDig: &DataDigest{
			MsgType: PullMsgType_UNDEFINED,
		},
	})
	assert.Error(t, msg.IsTagLegal())
}

func TestGossipMessageStateInfoMessageTagType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, &GossipMessage_StateInfo{
		StateInfo: &StateInfo{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, &GossipMessage_StateInfoPullReq{
		StateInfoPullReq: &StateInfoPullRequest{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, &GossipMessage_StateResponse{
		StateResponse: &RemoteStateResponse{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, &GossipMessage_StateRequest{
		StateRequest: &RemoteStateRequest{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, &GossipMessage_StateSnapshot{
		StateSnapshot: &StateInfoSnapshot{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_StateInfo{
		StateInfo: &StateInfo{},
	})
	assert.Error(t, msg.IsTagLegal())
}

func TestGossipMessageLeadershipMessageTagType(t *testing.T) {
	var msg *SignedGossipMessage
	channelID := "testID1"

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_AND_ORG, &GossipMessage_LeadershipMsg{
		LeadershipMsg: &LeadershipMessage{},
	})
	assert.NoError(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, &GossipMessage_LeadershipMsg{
		LeadershipMsg: &LeadershipMessage{},
	})
	assert.Error(t, msg.IsTagLegal())

	msg = signedGossipMessage(channelID, GossipMessage_CHAN_OR_ORG, &GossipMessage_Empty{})
	assert.Error(t, msg.IsTagLegal())
}

func TestGossipMessageSign(t *testing.T) {
	idSigner := func(msg []byte) ([]byte, error) {
		return msg, nil
	}

	errSigner := func(msg []byte) ([]byte, error) {
		return nil, errors.New("Error")
	}

	msg := signedGossipMessage("testChannelID", GossipMessage_EMPTY, &GossipMessage_DataMsg{
		DataMsg: &DataMessage{},
	})

	signedMsg, _ := msg.Sign(idSigner)

//由于检查身份签名者，签名将与有效负载相同
	assert.Equal(t, signedMsg.Payload, signedMsg.Signature)

	env, err := msg.Sign(errSigner)
	assert.Error(t, err)
	assert.Nil(t, env)
}

func TestEnvelope_NoopSign(t *testing.T) {
	channelID := "testChannelID"
	msg := signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataMsg{
		DataMsg: &DataMessage{},
	})

	signedMsg, err := msg.NoopSign()

//由于检查身份签名者，签名将与有效负载相同
	assert.Nil(t, signedMsg.Signature)
	assert.NoError(t, err)
}

func TestSignedGossipMessage_Verify(t *testing.T) {
	channelID := "testChannelID"
	peerID := []byte("peer")
	msg := signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataMsg{
		DataMsg: &DataMessage{},
	})

	assert.True(t, msg.IsSigned())

	verifier := func(peerIdentity []byte, signature, message []byte) error {
		return nil
	}

	res := msg.Verify(peerID, verifier)
	assert.Nil(t, res)

	msg = signedGossipMessage(channelID, GossipMessage_EMPTY, &GossipMessage_DataMsg{
		DataMsg: &DataMessage{},
	})

	env := msg.Envelope
	msg.Envelope = nil
	res = msg.Verify(peerID, verifier)
	assert.Error(t, res)

	msg.Envelope = env
	payload := msg.Envelope.Payload
	msg.Envelope.Payload = nil
	res = msg.Verify(peerID, verifier)
	assert.Error(t, res)

	msg.Envelope.Payload = payload
	sig := msg.Signature
	msg.Signature = nil
	res = msg.Verify(peerID, verifier)
	assert.Error(t, res)
	msg.Signature = sig

	errVerifier := func(peerIdentity []byte, signature, message []byte) error {
		return errors.New("Test")
	}

	res = msg.Verify(peerID, errVerifier)
	assert.Error(t, res)
}

func TestEnvelope(t *testing.T) {
	dataMsg := &GossipMessage{
		Content: dataMessage(1, []byte("data")),
	}
	bytes, err := proto.Marshal(dataMsg)
	assert.NoError(t, err)

	env := envelopes()[0]
	env.Payload = bytes

	msg, err := env.ToGossipMessage()
	assert.NoError(t, err)
	assert.NotNil(t, msg)

	assert.True(t, msg.IsDataMsg())
}

func TestEnvelope_SignSecret(t *testing.T) {
	dataMsg := &GossipMessage{
		Content: dataMessage(1, []byte("data")),
	}
	bytes, err := proto.Marshal(dataMsg)
	assert.NoError(t, err)

	env := envelopes()[0]
	env.Payload = bytes
	env.SecretEnvelope = nil

	env.SignSecret(func(message []byte) ([]byte, error) {
		return message, nil
	}, &Secret{
		Content: &Secret_InternalEndpoint{
			InternalEndpoint: "localhost:5050",
		},
	})

	assert.NotNil(t, env.SecretEnvelope)
	assert.Equal(t, env.SecretEnvelope.InternalEndpoint(), "localhost:5050")
}

func envelopes() []*Envelope {
	return []*Envelope{
		{Payload: []byte{2, 2, 2},
			Signature: []byte{2, 2, 2},
			SecretEnvelope: &SecretEnvelope{
				Payload:   []byte{2, 2, 2},
				Signature: []byte{2, 2, 2},
			},
		},
	}
}

func leadershipMessage(incNum uint64, seqNum uint64, pkid []byte) *GossipMessage_LeadershipMsg {
	return &GossipMessage_LeadershipMsg{
		LeadershipMsg: &LeadershipMessage{
			PkiId:         pkid,
			IsDeclaration: false,
			Timestamp: &PeerTime{
				IncNum: incNum,
				SeqNum: seqNum,
			},
		},
	}
}

func stateInfoMessage(incNum uint64, seqNum uint64, pkid []byte, mac []byte) *GossipMessage_StateInfo {
	return &GossipMessage_StateInfo{
		StateInfo: &StateInfo{
			Timestamp: &PeerTime{
				IncNum: incNum,
				SeqNum: seqNum,
			},
			PkiId:       pkid,
			Channel_MAC: mac,
		},
	}
}

func dataMessage(seqNum uint64, data []byte) *GossipMessage_DataMsg {
	return &GossipMessage_DataMsg{
		DataMsg: &DataMessage{
			Payload: &Payload{
				SeqNum: seqNum,
				Data:   data,
			},
		},
	}
}

func signedGossipMessage(channelID string, tag GossipMessage_Tag, content isGossipMessage_Content) *SignedGossipMessage {
	return &SignedGossipMessage{
		GossipMessage: &GossipMessage{
			Channel: []byte(channelID),
			Tag:     tag,
			Nonce:   0,
			Content: content,
		},
		Envelope: envelopes()[0],
	}
}
