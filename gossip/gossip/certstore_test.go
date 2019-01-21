
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


package gossip

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip/algo"
	"github.com/hyperledger/fabric/gossip/gossip/pull"
	"github.com/hyperledger/fabric/gossip/identity"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	util.SetupTestLogging()
	shortenedWaitTime := time.Millisecond * 300
	algo.SetDigestWaitTime(shortenedWaitTime / 2)
	algo.SetRequestWaitTime(shortenedWaitTime)
	algo.SetResponseWaitTime(shortenedWaitTime)
}

var (
	cs = &naiveCryptoService{
		revokedPkiIDS: make(map[string]struct{}),
	}
)

type pullerMock struct {
	mock.Mock
	pull.Mediator
}

type sentMsg struct {
	msg *proto.SignedGossipMessage
	mock.Mock
}

//GetSourceEnvelope返回接收到的消息的SignedGossipMessage
//建筑用
func (s *sentMsg) GetSourceEnvelope() *proto.Envelope {
	return nil
}

//ACK向发送方返回消息确认
func (s *sentMsg) Ack(err error) {

}

func (s *sentMsg) Respond(msg *proto.GossipMessage) {
	s.Called(msg)
}

func (s *sentMsg) GetGossipMessage() *proto.SignedGossipMessage {
	return s.msg
}

func (s *sentMsg) GetConnectionInfo() *proto.ConnectionInfo {
	return nil
}

type senderMock struct {
	mock.Mock
}

func (s *senderMock) Send(msg *proto.SignedGossipMessage, peers ...*comm.RemotePeer) {
	s.Called(msg, peers)
}

type membershipSvcMock struct {
	mock.Mock
}

func (m *membershipSvcMock) GetMembership() []discovery.NetworkMember {
	args := m.Called()
	return args.Get(0).([]discovery.NetworkMember)
}

func TestCertStoreBadSignature(t *testing.T) {
	badSignature := func(nonce uint64) proto.ReceivedMessage {
		return createUpdateMessage(nonce, createBadlySignedUpdateMessage())
	}
	pm, cs, _ := createObjects(badSignature, nil)
	defer pm.Stop()
	defer cs.stop()
	testCertificateUpdate(t, false, cs)
}

func TestCertStoreMismatchedIdentity(t *testing.T) {
	mismatchedIdentity := func(nonce uint64) proto.ReceivedMessage {
		return createUpdateMessage(nonce, createMismatchedUpdateMessage())
	}

	pm, cs, _ := createObjects(mismatchedIdentity, nil)
	defer pm.Stop()
	defer cs.stop()
	testCertificateUpdate(t, false, cs)
}

func TestCertStoreShouldSucceed(t *testing.T) {
	totallyFineIdentity := func(nonce uint64) proto.ReceivedMessage {
		return createUpdateMessage(nonce, createValidUpdateMessage())
	}

	pm, cs, _ := createObjects(totallyFineIdentity, nil)
	defer pm.Stop()
	defer cs.stop()
	testCertificateUpdate(t, true, cs)
}

func TestCertRevocation(t *testing.T) {
	defer func() {
		cs.revokedPkiIDS = map[string]struct{}{}
	}()

	totallyFineIdentity := func(nonce uint64) proto.ReceivedMessage {
		return createUpdateMessage(nonce, createValidUpdateMessage())
	}

	askedForIdentity := make(chan struct{}, 1)

	pm, cStore, sender := createObjects(totallyFineIdentity, func(message *proto.SignedGossipMessage) {
		askedForIdentity <- struct{}{}
	})
	defer cStore.stop()
	defer pm.Stop()
	testCertificateUpdate(t, true, cStore)
//第一次应该要身份证
	assert.Len(t, askedForIdentity, 1)
//排水渠
	<-askedForIdentity
//现在是0
	assert.Len(t, askedForIdentity, 0)

	sentHello := false
	l := sync.Mutex{}
	sender.Mock = mock.Mock{}
	sender.On("Send", mock.Anything, mock.Anything).Run(func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.SignedGossipMessage)
		l.Lock()
		defer l.Unlock()

		if hello := msg.GetHello(); hello != nil && !sentHello {
			sentHello = true
			dig := &proto.GossipMessage{
				Tag: proto.GossipMessage_EMPTY,
				Content: &proto.GossipMessage_DataDig{
					DataDig: &proto.DataDigest{
						Nonce:   hello.Nonce,
						MsgType: proto.PullMsgType_IDENTITY_MSG,
						Digests: [][]byte{[]byte("B")},
					},
				},
			}
			sMsg, _ := dig.NoopSign()
			go cStore.handleMessage(&sentMsg{msg: sMsg})
		}

		if dataReq := msg.GetDataReq(); dataReq != nil {
			askedForIdentity <- struct{}{}
		}
	})
	testCertificateUpdate(t, true, cStore)
//不应该问，因为已经有身份了
	select {
	case <-time.After(time.Second * 5):
	case <-askedForIdentity:
		assert.Fail(t, "Shouldn't have asked for an identity, because we already have it")
	}
	assert.Len(t, askedForIdentity, 0)
//Revoke the identity
	cs.revoke(common.PKIidType("B"))
	cStore.suspectPeers(func(id api.PeerIdentityType) bool {
		return string(id) == "B"
	})

	l.Lock()
	sentHello = false
	l.Unlock()

	select {
	case <-time.After(time.Second * 5):
		assert.Fail(t, "Didn't ask for identity, but should have. Looks like identity hasn't expired")
	case <-askedForIdentity:
	}
}

func TestCertExpiration(t *testing.T) {
//Scenario: In this test we make sure that a peer may not expire
//它自己的身份。
//这一点很重要，因为身份的唯一途径就是八卦。
//通过牵引机构传递。
//如果一个同伴的身份从“拉”调解人中消失，
//它将永远不会传递给对等机。
//测试确保自我身份不会过期
//以下列方式：
//它启动一个对等机，然后休眠两倍的标识使用阈值，
//in order to make sure that its own identity should be expired.
//Then, it starts another peer, and listens to the messages sent
//在两个对等点之间，并查找第一个对等点的一些身份摘要。
//If such identity digest are detected, it means that the peer
//它自己的身份没有过期。

//备份原始usagethreshold值
	idUsageThreshold := identity.GetIdentityUsageThreshold()
	identity.SetIdentityUsageThreshold(time.Second)
//还原原始usagethreshold值
	defer identity.SetIdentityUsageThreshold(idUsageThreshold)

	g1 := newGossipInstance(4321, 0, 0, 1)
	defer g1.Stop()
	time.Sleep(identity.GetIdentityUsageThreshold() * 2)
	g2 := newGossipInstance(4322, 0, 0)
	defer g2.Stop()

	identities2Detect := 3
//使通道比需要的更大，这样Goroutines就不会被卡住。
	identitiesGotViaPull := make(chan struct{}, identities2Detect+100)
	acceptIdentityPullMsgs := func(o interface{}) bool {
		m := o.(proto.ReceivedMessage).GetGossipMessage()
		if m.IsPullMsg() && m.IsDigestMsg() {
			for _, dig := range m.GetDataDig().Digests {
				if bytes.Equal(dig, []byte("localhost:4321")) {
					identitiesGotViaPull <- struct{}{}
				}
			}
		}
		return false
	}
	g1.Accept(acceptIdentityPullMsgs, true)
	for i := 0; i < identities2Detect; i++ {
		select {
		case <-identitiesGotViaPull:
		case <-time.After(time.Second * 15):
			assert.Fail(t, "Didn't detect an identity gossiped via pull in a timely manner")
			return
		}
	}
}

func testCertificateUpdate(t *testing.T, shouldSucceed bool, certStore *certStore) {
	msg, _ := (&proto.GossipMessage{
		Channel: []byte(""),
		Tag:     proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_Hello{
			Hello: &proto.GossipHello{
				Nonce:    0,
				Metadata: nil,
				MsgType:  proto.PullMsgType_IDENTITY_MSG,
			},
		},
	}).NoopSign()
	hello := &sentMsg{
		msg: msg,
	}
	responseChan := make(chan *proto.GossipMessage, 1)
	hello.On("Respond", mock.Anything).Run(func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.GossipMessage)
		assert.NotNil(t, msg.GetDataDig())
		responseChan <- msg
	})
	certStore.handleMessage(hello)
	select {
	case msg := <-responseChan:
		if shouldSucceed {
			assert.Len(t, msg.GetDataDig().Digests, 2, "Valid identity hasn't entered the certStore")
		} else {
			assert.Len(t, msg.GetDataDig().Digests, 1, "Mismatched identity has been injected into certStore")
		}
	case <-time.After(time.Second):
		t.Fatal("Didn't respond with a digest message in a timely manner")
	}
}

func createMismatchedUpdateMessage() *proto.SignedGossipMessage {
	peeridentity := &proto.PeerIdentity{
//This PKI-ID is different than the cert, and the mapping between
//在这个测试中，到pki-id的证书只是标识函数。
		PkiId: []byte("A"),
		Cert:  []byte("D"),
	}

	signer := func(msg []byte) ([]byte, error) {
		return (&naiveCryptoService{}).Sign(msg)
	}
	m := &proto.GossipMessage{
		Channel: nil,
		Nonce:   0,
		Tag:     proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_PeerIdentity{
			PeerIdentity: peeridentity,
		},
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	sMsg.Sign(signer)
	return sMsg
}

func createBadlySignedUpdateMessage() *proto.SignedGossipMessage {
	peeridentity := &proto.PeerIdentity{
		PkiId: []byte("C"),
		Cert:  []byte("C"),
	}

	signer := func(msg []byte) ([]byte, error) {
		return (&naiveCryptoService{}).Sign(msg)
	}

	m := &proto.GossipMessage{
		Channel: nil,
		Nonce:   0,
		Tag:     proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_PeerIdentity{
			PeerIdentity: peeridentity,
		},
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	sMsg.Sign(signer)
//这会模拟出一个错误的信号
	if sMsg.Envelope.Signature[0] == 0 {
		sMsg.Envelope.Signature[0] = 1
	} else {
		sMsg.Envelope.Signature[0] = 0
	}
	return sMsg
}

func createValidUpdateMessage() *proto.SignedGossipMessage {
	peeridentity := &proto.PeerIdentity{
		PkiId: []byte("B"),
		Cert:  []byte("B"),
	}

	signer := func(msg []byte) ([]byte, error) {
		return (&naiveCryptoService{}).Sign(msg)
	}
	m := &proto.GossipMessage{
		Channel: nil,
		Nonce:   0,
		Tag:     proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_PeerIdentity{
			PeerIdentity: peeridentity,
		},
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	sMsg.Sign(signer)
	return sMsg
}

func createUpdateMessage(nonce uint64, idMsg *proto.SignedGossipMessage) proto.ReceivedMessage {
	update := &proto.GossipMessage{
		Tag: proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_DataUpdate{
			DataUpdate: &proto.DataUpdate{
				MsgType: proto.PullMsgType_IDENTITY_MSG,
				Nonce:   nonce,
				Data:    []*proto.Envelope{idMsg.Envelope},
			},
		},
	}
	sMsg, _ := update.NoopSign()
	return &sentMsg{msg: sMsg}
}

func createDigest(nonce uint64) proto.ReceivedMessage {
	digest := &proto.GossipMessage{
		Tag: proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_DataDig{
			DataDig: &proto.DataDigest{
				Nonce:   nonce,
				MsgType: proto.PullMsgType_IDENTITY_MSG,
				Digests: [][]byte{[]byte("A"), []byte("C")},
			},
		},
	}
	sMsg, _ := digest.NoopSign()
	return &sentMsg{msg: sMsg}
}

func createObjects(updateFactory func(uint64) proto.ReceivedMessage, msgCons proto.MsgConsumer) (pull.Mediator, *certStore, *senderMock) {
	if msgCons == nil {
		msgCons = func(_ *proto.SignedGossipMessage) {}
	}
	config := pull.Config{
		MsgType:           proto.PullMsgType_IDENTITY_MSG,
		PeerCountToSelect: 1,
		PullInterval:      time.Second,
		Tag:               proto.GossipMessage_EMPTY,
		Channel:           nil,
		ID:                "id1",
	}
	sender := &senderMock{}
	memberSvc := &membershipSvcMock{}
	memberSvc.On("GetMembership").Return([]discovery.NetworkMember{{PKIid: []byte("bla bla"), Endpoint: "localhost:5611"}})

	var certStore *certStore
	adapter := &pull.PullAdapter{
		Sndr: sender,
		MsgCons: func(msg *proto.SignedGossipMessage) {
			certStore.idMapper.Put(msg.GetPeerIdentity().PkiId, msg.GetPeerIdentity().Cert)
			msgCons(msg)
		},
		IdExtractor: func(msg *proto.SignedGossipMessage) string {
			return string(msg.GetPeerIdentity().PkiId)
		},
		MemSvc: memberSvc,
	}
	pullMediator := pull.NewPullMediator(config, adapter)
	selfIdentity := api.PeerIdentityType("SELF")
	certStore = newCertStore(&pullerMock{
		Mediator: pullMediator,
	}, identity.NewIdentityMapper(cs, selfIdentity, func(pkiID common.PKIidType, _ api.PeerIdentityType) {
		pullMediator.Remove(string(pkiID))
	}, cs), selfIdentity, cs)

	wg := sync.WaitGroup{}
	wg.Add(1)
	sentHello := false
	sentDataReq := false
	l := sync.Mutex{}
	sender.On("Send", mock.Anything, mock.Anything).Run(func(arg mock.Arguments) {
		msg := arg.Get(0).(*proto.SignedGossipMessage)
		l.Lock()
		defer l.Unlock()

		if hello := msg.GetHello(); hello != nil && !sentHello {
			sentHello = true
			go certStore.handleMessage(createDigest(hello.Nonce))
		}

		if dataReq := msg.GetDataReq(); dataReq != nil && !sentDataReq {
			sentDataReq = true
			certStore.handleMessage(updateFactory(dataReq.Nonce))
			wg.Done()
		}
	})
	wg.Wait()
	return pullMediator, certStore, sender
}
