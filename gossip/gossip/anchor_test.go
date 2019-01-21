
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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type peerMock struct {
	pkiID                common.PKIidType
	selfCertHash         []byte
	gRGCserv             *grpc.Server
	lsnr                 net.Listener
	finishedSignal       sync.WaitGroup
	expectedMsgs2Receive uint32
	msgReceivedCount     uint32
	msgAssertions        []msgInspection
	t                    *testing.T
}

func (p *peerMock) GossipStream(stream proto.Gossip_GossipStreamServer) error {
	sessionCounter := 0
	for {
		envelope, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		gMsg, err := envelope.ToGossipMessage()
		if err != nil {
			panic(err)
		}
		if sessionCounter == 0 {
			connEstablishMsg := p.connEstablishMsg(p.pkiID, p.selfCertHash, api.PeerIdentityType(p.pkiID))
			stream.Send(connEstablishMsg.Envelope)
		}
		for _, assertion := range p.msgAssertions {
			assertion(p.t, sessionCounter, &receivedMsg{stream: stream, SignedGossipMessage: gMsg})
		}
		p.t.Log("sessionCounter:", sessionCounter, string(p.pkiID), "got msg:", gMsg)
		sessionCounter++
		atomic.AddUint32(&p.msgReceivedCount, uint32(1))
		if atomic.LoadUint32(&p.msgReceivedCount) == p.expectedMsgs2Receive {
			p.finishedSignal.Done()
		}
	}
}

func (p *peerMock) Ping(context.Context, *proto.Empty) (*proto.Empty, error) {
	return &proto.Empty{}, nil
}

func newPeerMock(port int, expectedMsgs2Receive int, t *testing.T, msgAssertions ...msgInspection) *peerMock {
	listenAddress := fmt.Sprintf(":%d", port)
	ll, err := net.Listen("tcp", listenAddress)
	if err != nil {
		fmt.Printf("Error listening on %v, %v", listenAddress, err)
	}
	s, selfCertHash := newGRPCServerWithTLS()
	p := &peerMock{
		lsnr:                 ll,
		gRGCserv:             s,
		msgAssertions:        msgAssertions,
		t:                    t,
		pkiID:                common.PKIidType(fmt.Sprintf("localhost:%d", port)),
		selfCertHash:         selfCertHash,
		expectedMsgs2Receive: uint32(expectedMsgs2Receive),
	}
	p.finishedSignal.Add(1)
	proto.RegisterGossipServer(s, p)
	go s.Serve(ll)
	return p
}

func newGRPCServerWithTLS() (*grpc.Server, []byte) {
	cert := comm.GenerateCertificatesOrPanic()
	tlsConf := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.RequestClientCert,
		InsecureSkipVerify: true,
	}
	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConf)))
	return s, util.ComputeSHA256(cert.Certificate[0])
}

func (p *peerMock) connEstablishMsg(pkiID common.PKIidType, hash []byte, cert api.PeerIdentityType) *proto.SignedGossipMessage {
	m := &proto.GossipMessage{
		Tag:   proto.GossipMessage_EMPTY,
		Nonce: 0,
		Content: &proto.GossipMessage_Conn{
			Conn: &proto.ConnEstablish{
				TlsCertHash: hash,
				Identity:    cert,
				PkiId:       pkiID,
			},
		},
	}
	gMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	gMsg.Sign((&configurableCryptoService{}).Sign)
	return gMsg
}

func (p *peerMock) stop() {
	p.lsnr.Close()
	p.gRGCserv.Stop()
}

type receivedMsg struct {
	*proto.SignedGossipMessage
	stream proto.Gossip_GossipStreamServer
}

func (msg *receivedMsg) respond(message *proto.SignedGossipMessage) {
	msg.stream.Send(message.Envelope)
}

func memResp(nonce uint64, endpoint string) *proto.SignedGossipMessage {
	fakePeerAliveMsg := &proto.SignedGossipMessage{
		GossipMessage: &proto.GossipMessage{
			Tag: proto.GossipMessage_EMPTY,
			Content: &proto.GossipMessage_AliveMsg{
				AliveMsg: &proto.AliveMessage{
					Membership: &proto.Member{
						Endpoint: endpoint,
						PkiId:    []byte(endpoint),
					},
					Identity: []byte(endpoint),
					Timestamp: &proto.PeerTime{
						IncNum: uint64(time.Now().UnixNano()),
						SeqNum: 0,
					},
				},
			},
		},
	}

	m, _ := fakePeerAliveMsg.Sign((&configurableCryptoService{}).Sign)
	sMsg, _ := (&proto.SignedGossipMessage{
		GossipMessage: &proto.GossipMessage{
			Tag:   proto.GossipMessage_EMPTY,
			Nonce: nonce,
			Content: &proto.GossipMessage_MemRes{
				MemRes: &proto.MembershipResponse{
					Alive: []*proto.Envelope{m},
					Dead:  []*proto.Envelope{},
				},
			},
		},
	}).NoopSign()
	return sMsg
}

type msgInspection func(t *testing.T, index int, m *receivedMsg)

func TestAnchorPeer(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
//演员：
//Orga：{
//P：一个真正的流言蜚语实例
//AP1:peermock类型的锚定对等
//PM1:A*Peermock公司
//}
//OrgB：{
//AP2:peermock类型的锚定对等
//PM2:A*Peermock公司
//}
//脚本：
//生成将用于连接到2个定位点对等点的对等点（P）。
//5秒后，生成锚节点。
//查看发送到定位点对等端的MembershipRequest消息
//仅包含ORG中的锚定对等端的内部终结点。
//每个锚定对等机都会告诉自己组织中的对等机（PM1或PM2）。
//等待直到“P”向每个锚定对等发送3条消息（握手、握手+memreq）
//向PM1和PM2中的每一个发送一条消息（用于握手），以证明成员响应
//已成功从定位点对等端发送到P。

	cs := &configurableCryptoService{m: make(map[string]api.OrgIdentityType)}
	portPrefix := 13610
	orgA := "orgA"
	orgB := "orgB"
cs.putInOrg(portPrefix, orgA)   //真实对等体
cs.putInOrg(portPrefix+1, orgA) //主播对等模拟
cs.putInOrg(portPrefix+2, orgB) //主播对等模拟
cs.putInOrg(portPrefix+3, orgA) //同侪模仿我
cs.putInOrg(portPrefix+4, orgB) //同辈模拟II

//创建断言
	handshake := func(t *testing.T, index int, m *receivedMsg) {
		if index != 0 {
			return
		}
		assert.NotNil(t, m.GetConn())
	}

	memReqWithInternalEndpoint := func(t *testing.T, index int, m *receivedMsg) {
		if m.GetMemReq() == nil {
			return
		}
		assert.True(t, index > 0)
		req := m.GetMemReq()
		am, err := req.SelfInformation.ToGossipMessage()
		assert.NoError(t, err)
		assert.NotEmpty(t, am.GetSecretEnvelope().InternalEndpoint())
		m.respond(memResp(m.Nonce, fmt.Sprintf("localhost:%d", portPrefix+3)))
	}

	memReqWithoutInternalEndpoint := func(t *testing.T, index int, m *receivedMsg) {
		if m.GetMemReq() == nil {
			return
		}
		assert.True(t, index > 0)
		req := m.GetMemReq()
		am, err := req.SelfInformation.ToGossipMessage()
		assert.NoError(t, err)
		assert.Nil(t, am.GetSecretEnvelope())
		m.respond(memResp(m.Nonce, fmt.Sprintf("localhost:%d", portPrefix+4)))
	}

//创建对等模拟
	pm1 := newPeerMock(portPrefix+3, 1, t, handshake)
	defer pm1.stop()
	pm2 := newPeerMock(portPrefix+4, 1, t, handshake)
	defer pm2.stop()
	jcm := &joinChanMsg{
		members2AnchorPeers: map[string][]api.AnchorPeer{
			orgA: {
				{Host: "localhost", Port: portPrefix + 1},
			},
			orgB: {
				{Host: "localhost", Port: portPrefix + 2},
			},
		},
	}
	channel := common.ChainID("TEST")
	endpoint := fmt.Sprintf("localhost:%d", portPrefix)
//创建八卦实例（连接到锚定对等机的对等机）
	p := newGossipInstanceWithExternalEndpoint(portPrefix, 0, cs, endpoint)
	defer p.Stop()
	p.JoinChan(jcm, channel)
	p.UpdateLedgerHeight(1, channel)

	time.Sleep(time.Second * 5)

//创建定位点对等点
	ap1 := newPeerMock(portPrefix+1, 3, t, handshake, memReqWithInternalEndpoint)
	defer ap1.stop()
	ap2 := newPeerMock(portPrefix+2, 3, t, handshake, memReqWithoutInternalEndpoint)
	defer ap2.stop()

//等待，直到收到来自八卦实例的所有预期消息
	ap1.finishedSignal.Wait()
	ap2.finishedSignal.Wait()
	pm1.finishedSignal.Wait()
	pm2.finishedSignal.Wait()
}

func TestBootstrapPeerMisConfiguration(t *testing.T) {
	t.Parallel()
	defer testWG.Done()
//脚本：
//同伴“p”是ORGA中的同伴。
//对等端BS1和BS2是引导对等端。
//bs1在orgb中，所以p不应该连接到它。
//BS2在ORGA中，所以P应该连接到它。
//我们通过截取BS1和BS2从P获取的*所有*消息来测试：
//1）至少有3次连接尝试从P发送到BS1
//2）P向BS2发送了会员申请。

	cs := &configurableCryptoService{m: make(map[string]api.OrgIdentityType)}
	portPrefix := 43478
	orgA := "orgA"
	orgB := "orgB"
	cs.putInOrg(portPrefix, orgA)
	cs.putInOrg(portPrefix+1, orgB)
	cs.putInOrg(portPrefix+2, orgA)

	onlyHandshakes := func(t *testing.T, index int, m *receivedMsg) {
//确保发送的所有消息都是连接建立消息
//那是试探性的尝试
		assert.NotNil(t, m.GetConn())
//如果我们在这个测试中测试的逻辑失败，
//第一条信息是会员申请，
//所以这个断言将捕获它并打印相应的失败
		assert.Nil(t, m.GetMemReq())
	}
//初始化对等模拟，以等待发送给它的3条消息
	bs1 := newPeerMock(portPrefix+1, 3, t, onlyHandshakes)
	defer bs1.stop()

	membershipRequestsSent := make(chan struct{}, 100)
	detectMembershipRequest := func(t *testing.T, index int, m *receivedMsg) {
		if m.GetMemReq() != nil {
			membershipRequestsSent <- struct{}{}
		}
	}

	bs2 := newPeerMock(portPrefix+2, 0, t, detectMembershipRequest)
	defer bs2.stop()

	p := newGossipInstanceWithExternalEndpoint(portPrefix, 0, cs, fmt.Sprintf("localhost:%d", portPrefix), 1, 2)
	defer p.Stop()

//等待来自orgb的引导程序对等端的3次握手尝试，
//为了证明对等机确实尝试从orgb探测引导对等机
	got3Handshakes := make(chan struct{})
	go func() {
		bs1.finishedSignal.Wait()
		got3Handshakes <- struct{}{}
	}()

	select {
	case <-got3Handshakes:
	case <-time.After(time.Second * 15):
		assert.Fail(t, "Didn't detect 3 handshake attempts to the bootstrap peer from orgB")
	}

	select {
	case <-membershipRequestsSent:
	case <-time.After(time.Second * 15):
		assert.Fail(t, "Bootstrap peer didn't receive a membership request from the peer within a timely manner")
	}
}
