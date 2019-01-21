
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	protoG "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/gossip/msgstore"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

var timeout = time.Second * time.Duration(15)

func init() {
	util.SetupTestLogging()
	aliveTimeInterval := time.Duration(time.Millisecond * 300)
	SetAliveTimeInterval(aliveTimeInterval)
	SetAliveExpirationTimeout(10 * aliveTimeInterval)
	SetAliveExpirationCheckInterval(aliveTimeInterval)
	SetReconnectInterval(10 * aliveTimeInterval)
	maxConnectionAttempts = 10000
}

type dummyReceivedMessage struct {
	msg  *proto.SignedGossipMessage
	info *proto.ConnectionInfo
}

func (*dummyReceivedMessage) Respond(msg *proto.GossipMessage) {
	panic("implement me")
}

func (rm *dummyReceivedMessage) GetGossipMessage() *proto.SignedGossipMessage {
	return rm.msg
}

func (*dummyReceivedMessage) GetSourceEnvelope() *proto.Envelope {
	panic("implement me")
}

func (rm *dummyReceivedMessage) GetConnectionInfo() *proto.ConnectionInfo {
	return rm.info
}

func (*dummyReceivedMessage) Ack(err error) {
	panic("implement me")
}

type dummyCommModule struct {
	validatedMessages chan *proto.SignedGossipMessage
	msgsReceived      uint32
	msgsSent          uint32
	id                string
	presumeDead       chan common.PKIidType
	detectedDead      chan string
	streams           map[string]proto.Gossip_GossipStreamClient
	conns             map[string]*grpc.ClientConn
	lock              *sync.RWMutex
	incMsgs           chan proto.ReceivedMessage
	lastSeqs          map[string]uint64
	shouldGossip      bool
	mock              *mock.Mock
}

type gossipInstance struct {
	msgInterceptor func(*proto.SignedGossipMessage)
	comm           *dummyCommModule
	Discovery
	gRGCserv      *grpc.Server
	lsnr          net.Listener
	shouldGossip  bool
	syncInitiator *time.Ticker
	stopChan      chan struct{}
	port          int
}

func (comm *dummyCommModule) ValidateAliveMsg(am *proto.SignedGossipMessage) bool {
	comm.lock.RLock()
	c := comm.validatedMessages
	comm.lock.RUnlock()

	if c != nil {
		c <- am
	}
	return true
}

func (comm *dummyCommModule) recordValidation(validatedMessages chan *proto.SignedGossipMessage) {
	comm.lock.Lock()
	defer comm.lock.Unlock()
	comm.validatedMessages = validatedMessages
}

func (comm *dummyCommModule) SignMessage(am *proto.GossipMessage, internalEndpoint string) *proto.Envelope {
	am.NoopSign()

	secret := &proto.Secret{
		Content: &proto.Secret_InternalEndpoint{
			InternalEndpoint: internalEndpoint,
		},
	}
	signer := func(msg []byte) ([]byte, error) {
		return nil, nil
	}
	s, _ := am.NoopSign()
	env := s.Envelope
	env.SignSecret(signer, secret)
	return env
}

func (comm *dummyCommModule) Gossip(msg *proto.SignedGossipMessage) {
	if !comm.shouldGossip {
		return
	}
	comm.lock.Lock()
	defer comm.lock.Unlock()
	for _, conn := range comm.streams {
		conn.Send(msg.Envelope)
	}
}

func (comm *dummyCommModule) Forward(msg proto.ReceivedMessage) {
	if !comm.shouldGossip {
		return
	}
	comm.lock.Lock()
	defer comm.lock.Unlock()
	for _, conn := range comm.streams {
		conn.Send(msg.GetGossipMessage().Envelope)
	}
}

func (comm *dummyCommModule) SendToPeer(peer *NetworkMember, msg *proto.SignedGossipMessage) {
	comm.lock.RLock()
	_, exists := comm.streams[peer.Endpoint]
	mock := comm.mock
	comm.lock.RUnlock()

	if mock != nil {
		mock.Called(peer, msg)
	}

	if !exists {
		if comm.Ping(peer) == false {
			fmt.Printf("Ping to %v failed\n", peer.Endpoint)
			return
		}
	}
	comm.lock.Lock()
	s, _ := msg.NoopSign()
	comm.streams[peer.Endpoint].Send(s.Envelope)
	comm.lock.Unlock()
	atomic.AddUint32(&comm.msgsSent, 1)
}

func (comm *dummyCommModule) Ping(peer *NetworkMember) bool {
	comm.lock.Lock()
	defer comm.lock.Unlock()

	if comm.mock != nil {
		comm.mock.Called()
	}

	_, alreadyExists := comm.streams[peer.Endpoint]
	if !alreadyExists {
		newConn, err := grpc.Dial(peer.Endpoint, grpc.WithInsecure())
		if err != nil {
			return false
		}
		if stream, err := proto.NewGossipClient(newConn).GossipStream(context.Background()); err == nil {
			comm.conns[peer.Endpoint] = newConn
			comm.streams[peer.Endpoint] = stream
			return true
		}
		return false
	}
	conn := comm.conns[peer.Endpoint]
	if _, err := proto.NewGossipClient(conn).Ping(context.Background(), &proto.Empty{}); err != nil {
		return false
	}
	return true
}

func (comm *dummyCommModule) Accept() <-chan proto.ReceivedMessage {
	return comm.incMsgs
}

func (comm *dummyCommModule) PresumedDead() <-chan common.PKIidType {
	return comm.presumeDead
}

func (comm *dummyCommModule) CloseConn(peer *NetworkMember) {
	comm.lock.Lock()
	defer comm.lock.Unlock()

	if _, exists := comm.streams[peer.Endpoint]; !exists {
		return
	}

	comm.streams[peer.Endpoint].CloseSend()
	comm.conns[peer.Endpoint].Close()
}

func (g *gossipInstance) receivedMsgCount() int {
	return int(atomic.LoadUint32(&g.comm.msgsReceived))
}

func (g *gossipInstance) sentMsgCount() int {
	return int(atomic.LoadUint32(&g.comm.msgsSent))
}

func (g *gossipInstance) discoveryImpl() *gossipDiscoveryImpl {
	return g.Discovery.(*gossipDiscoveryImpl)
}

func (g *gossipInstance) initiateSync(frequency time.Duration, peerNum int) {
	g.syncInitiator = time.NewTicker(frequency)
	g.stopChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-g.syncInitiator.C:
				g.Discovery.InitiateSync(peerNum)
			case <-g.stopChan:
				g.syncInitiator.Stop()
				return
			}
		}
	}()
}

func (g *gossipInstance) GossipStream(stream proto.Gossip_GossipStreamServer) error {
	for {
		envelope, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		lgr := g.Discovery.(*gossipDiscoveryImpl).logger
		gMsg, err := envelope.ToGossipMessage()
		if err != nil {
			lgr.Warning("Failed deserializing GossipMessage from envelope:", err)
			continue
		}
		g.msgInterceptor(gMsg)

		lgr.Debug(g.Discovery.Self().Endpoint, "Got message:", gMsg)
		g.comm.incMsgs <- &dummyReceivedMessage{
			msg: gMsg,
			info: &proto.ConnectionInfo{
				ID: common.PKIidType("testID"),
			},
		}
		atomic.AddUint32(&g.comm.msgsReceived, 1)

		if aliveMsg := gMsg.GetAliveMsg(); aliveMsg != nil {
			g.tryForwardMessage(gMsg)
		}
	}
}

func (g *gossipInstance) tryForwardMessage(msg *proto.SignedGossipMessage) {
	g.comm.lock.Lock()

	aliveMsg := msg.GetAliveMsg()

	forward := false
	id := string(aliveMsg.Membership.PkiId)
	seqNum := aliveMsg.Timestamp.SeqNum
	if last, exists := g.comm.lastSeqs[id]; exists {
		if last < seqNum {
			g.comm.lastSeqs[id] = seqNum
			forward = true
		}
	} else {
		g.comm.lastSeqs[id] = seqNum
		forward = true
	}

	g.comm.lock.Unlock()

	if forward {
		g.comm.Gossip(msg)
	}
}

func (g *gossipInstance) Stop() {
	if g.syncInitiator != nil {
		g.stopChan <- struct{}{}
	}
	g.gRGCserv.Stop()
	g.lsnr.Close()
	g.comm.lock.Lock()
	for _, stream := range g.comm.streams {
		stream.CloseSend()
	}
	g.comm.lock.Unlock()
	for _, conn := range g.comm.conns {
		conn.Close()
	}
	g.Discovery.Stop()
}

func (g *gossipInstance) Ping(context.Context, *proto.Empty) (*proto.Empty, error) {
	return &proto.Empty{}, nil
}

var noopPolicy = func(remotePeer *NetworkMember) (Sieve, EnvelopeFilter) {
	return func(msg *proto.SignedGossipMessage) bool {
			return true
		}, func(message *proto.SignedGossipMessage) *proto.Envelope {
			return message.Envelope
		}
}

func createDiscoveryInstance(port int, id string, bootstrapPeers []string) *gossipInstance {
	return createDiscoveryInstanceThatGossips(port, id, bootstrapPeers, true, noopPolicy)
}

func createDiscoveryInstanceWithNoGossip(port int, id string, bootstrapPeers []string) *gossipInstance {
	return createDiscoveryInstanceThatGossips(port, id, bootstrapPeers, false, noopPolicy)
}

func createDiscoveryInstanceWithNoGossipWithDisclosurePolicy(port int, id string, bootstrapPeers []string, pol DisclosurePolicy) *gossipInstance {
	return createDiscoveryInstanceThatGossips(port, id, bootstrapPeers, false, pol)
}

func createDiscoveryInstanceThatGossips(port int, id string, bootstrapPeers []string, shouldGossip bool, pol DisclosurePolicy) *gossipInstance {
	return createDiscoveryInstanceThatGossipsWithInterceptors(port, id, bootstrapPeers, shouldGossip, pol, func(_ *proto.SignedGossipMessage) {})
}

func createDiscoveryInstanceThatGossipsWithInterceptors(port int, id string, bootstrapPeers []string, shouldGossip bool, pol DisclosurePolicy, f func(*proto.SignedGossipMessage)) *gossipInstance {
	comm := &dummyCommModule{
		conns:        make(map[string]*grpc.ClientConn),
		streams:      make(map[string]proto.Gossip_GossipStreamClient),
		incMsgs:      make(chan proto.ReceivedMessage, 1000),
		presumeDead:  make(chan common.PKIidType, 10000),
		id:           id,
		detectedDead: make(chan string, 10000),
		lock:         &sync.RWMutex{},
		lastSeqs:     make(map[string]uint64),
		shouldGossip: shouldGossip,
	}

	endpoint := fmt.Sprintf("localhost:%d", port)
	self := NetworkMember{
		Metadata:         []byte{},
		PKIid:            []byte(endpoint),
		Endpoint:         endpoint,
		InternalEndpoint: endpoint,
	}

	listenAddress := fmt.Sprintf("%s:%d", "", port)
	ll, err := net.Listen("tcp", listenAddress)
	if err != nil {
		fmt.Printf("Error listening on %v, %v", listenAddress, err)
	}
	s := grpc.NewServer()

	discSvc := NewDiscoveryService(self, comm, comm, pol)
	for _, bootPeer := range bootstrapPeers {
		bp := bootPeer
		discSvc.Connect(NetworkMember{Endpoint: bp, InternalEndpoint: bootPeer}, func() (*PeerIdentification, error) {
			return &PeerIdentification{SelfOrg: true, ID: common.PKIidType(bp)}, nil
		})
	}

	gossInst := &gossipInstance{comm: comm, gRGCserv: s, Discovery: discSvc, lsnr: ll, shouldGossip: shouldGossip, port: port, msgInterceptor: f}

	proto.RegisterGossipServer(s, gossInst)
	go s.Serve(ll)

	return gossInst
}

func bootPeer(port int) string {
	return fmt.Sprintf("localhost:%d", port)
}

func TestHasExternalEndpoints(t *testing.T) {
	memberWithEndpoint := NetworkMember{Endpoint: "foo"}
	memberWithoutEndpoint := NetworkMember{}

	assert.True(t, HasExternalEndpoint(memberWithEndpoint))
	assert.False(t, HasExternalEndpoint(memberWithoutEndpoint))
}

func TestToString(t *testing.T) {
	nm := NetworkMember{
		Endpoint:         "a",
		InternalEndpoint: "b",
	}
	assert.Equal(t, "b", nm.PreferredEndpoint())
	nm = NetworkMember{
		Endpoint: "a",
	}
	assert.Equal(t, "a", nm.PreferredEndpoint())

	now := time.Now()
	ts := &timestamp{
		incTime: now,
		seqNum:  uint64(42),
	}
	assert.Equal(t, fmt.Sprintf("%d, %d", now.UnixNano(), 42), fmt.Sprint(ts))
}

func TestNetworkMemberString(t *testing.T) {
	tests := []struct {
		input    NetworkMember
		expected string
	}{
		{
			input:    NetworkMember{Endpoint: "endpoint", InternalEndpoint: "internal-endpoint", PKIid: common.PKIidType{0, 1, 2, 3}, Metadata: nil},
			expected: "Endpoint: endpoint, InternalEndpoint: internal-endpoint, PKI-ID: 00010203, Metadata: ",
		},
		{
			input:    NetworkMember{Endpoint: "endpoint", InternalEndpoint: "internal-endpoint", PKIid: common.PKIidType{0, 1, 2, 3}, Metadata: []byte{4, 5, 6, 7}},
			expected: "Endpoint: endpoint, InternalEndpoint: internal-endpoint, PKI-ID: 00010203, Metadata: 04050607",
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.input.String())
	}
}

func TestBadInput(t *testing.T) {
	inst := createDiscoveryInstance(2048, fmt.Sprintf("d%d", 0), []string{})
	inst.Discovery.(*gossipDiscoveryImpl).handleMsgFromComm(nil)
	s, _ := (&proto.GossipMessage{
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{},
		},
	}).NoopSign()
	inst.Discovery.(*gossipDiscoveryImpl).handleMsgFromComm(&dummyReceivedMessage{
		msg: s,
		info: &proto.ConnectionInfo{
			ID: common.PKIidType("testID"),
		},
	})
}

func TestConnect(t *testing.T) {
	t.Parallel()
	nodeNum := 10
	instances := []*gossipInstance{}
	firstSentMemReqMsgs := make(chan *proto.SignedGossipMessage, nodeNum)
	for i := 0; i < nodeNum; i++ {
		inst := createDiscoveryInstance(7611+i, fmt.Sprintf("d%d", i), []string{})

		inst.comm.lock.Lock()
		inst.comm.mock = &mock.Mock{}
		inst.comm.mock.On("SendToPeer", mock.Anything, mock.Anything).Run(func(arguments mock.Arguments) {
			inst := inst
			msg := arguments.Get(1).(*proto.SignedGossipMessage)
			if req := msg.GetMemReq(); req != nil {
				selfMsg, _ := req.SelfInformation.ToGossipMessage()
				firstSentMemReqMsgs <- selfMsg
				inst.comm.lock.Lock()
				inst.comm.mock = nil
				inst.comm.lock.Unlock()
			}
		})
		inst.comm.mock.On("Ping", mock.Anything)
		inst.comm.lock.Unlock()
		instances = append(instances, inst)
		j := (i + 1) % 10
		endpoint := fmt.Sprintf("localhost:%d", 7611+j)
		netMember2Connect2 := NetworkMember{Endpoint: endpoint, PKIid: []byte(endpoint)}
		inst.Connect(netMember2Connect2, func() (identification *PeerIdentification, err error) {
			return &PeerIdentification{SelfOrg: false, ID: nil}, nil
		})
	}

	time.Sleep(time.Second * 3)
	fullMembership := func() bool {
		return nodeNum-1 == len(instances[nodeNum-1].GetMembership())
	}
	waitUntilOrFail(t, fullMembership)

	discInst := instances[rand.Intn(len(instances))].Discovery.(*gossipDiscoveryImpl)
	mr, _ := discInst.createMembershipRequest(true)
	am, _ := mr.GetMemReq().SelfInformation.ToGossipMessage()
	assert.NotNil(t, am.SecretEnvelope)
	mr2, _ := discInst.createMembershipRequest(false)
	am, _ = mr2.GetMemReq().SelfInformation.ToGossipMessage()
	assert.Nil(t, am.SecretEnvelope)
	stopInstances(t, instances)
	assert.Len(t, firstSentMemReqMsgs, 10)
	close(firstSentMemReqMsgs)
	for firstSentSelfMsg := range firstSentMemReqMsgs {
		assert.Nil(t, firstSentSelfMsg.Envelope.SecretEnvelope)
	}
}

func TestValidation(t *testing.T) {
	t.Parallel()

//场景：此测试包含以下子测试：
//1）活动消息验证：消息被验证<==>进入消息存储
//2）请求/响应消息验证：
//2.1）验证来自成员请求/响应的活动消息。
//2.2）一旦活动消息进入消息存储，通过成员响应接收它们。
//不会触发验证，而是通过成员请求-do。

	wrapReceivedMessage := func(msg *proto.SignedGossipMessage) proto.ReceivedMessage {
		return &dummyReceivedMessage{
			msg: msg,
			info: &proto.ConnectionInfo{
				ID: common.PKIidType("testID"),
			},
		}
	}

	requestMessagesReceived := make(chan *proto.SignedGossipMessage, 100)
	responseMessagesReceived := make(chan *proto.SignedGossipMessage, 100)
	aliveMessagesReceived := make(chan *proto.SignedGossipMessage, 5000)

	var membershipRequest atomic.Value
	var membershipResponseWithAlivePeers atomic.Value
	var membershipResponseWithDeadPeers atomic.Value

	recordMembershipRequest := func(req *proto.SignedGossipMessage) {
		msg, _ := req.GetMemReq().SelfInformation.ToGossipMessage()
		membershipRequest.Store(req)
		requestMessagesReceived <- msg
	}

	recordMembershipResponse := func(res *proto.SignedGossipMessage) {
		memRes := res.GetMemRes()
		if len(memRes.GetAlive()) > 0 {
			membershipResponseWithAlivePeers.Store(res)
		}
		if len(memRes.GetDead()) > 0 {
			membershipResponseWithDeadPeers.Store(res)
		}
		responseMessagesReceived <- res
	}

	interceptor := func(msg *proto.SignedGossipMessage) {
		if memReq := msg.GetMemReq(); memReq != nil {
			recordMembershipRequest(msg)
			return
		}

		if memRes := msg.GetMemRes(); memRes != nil {
			recordMembershipResponse(msg)
			return
		}
//否则，这是一个活生生的信息
		aliveMessagesReceived <- msg
	}

//p3是p1的引导对等机，p1是p2的引导对等机。
//p1向p3发送一个（成员）请求，并接收一个（成员）响应。
//P2向P1发送（成员资格）请求。
//因此，p1同时接收成员请求和响应。
	p1 := createDiscoveryInstanceThatGossipsWithInterceptors(4675, "p1", []string{bootPeer(4677)}, true, noopPolicy, interceptor)
	p2 := createDiscoveryInstance(4676, "p2", []string{bootPeer(4675)})
	p3 := createDiscoveryInstance(4677, "p3", nil)
	instances := []*gossipInstance{p1, p2, p3}

	assertMembership(t, instances, 2)

	instances = []*gossipInstance{p1, p2}
//停止P3并等待其死亡被检测到
	p3.Stop()
	assertMembership(t, instances, 1)
//强制p1发送成员请求，以便它可以接收回响应
//与死掉的同龄人。
	p1.InitiateSync(1)

//等待直到接收到带死点的响应
	waitUntilOrFail(t, func() bool {
		return membershipResponseWithDeadPeers.Load() != nil
	})

	p1.Stop()
	p2.Stop()

	close(aliveMessagesReceived)
	t.Log("Recorded", len(aliveMessagesReceived), "alive messages")
	t.Log("Recorded", len(requestMessagesReceived), "request messages")
	t.Log("Recorded", len(responseMessagesReceived), "response messages")

//确保从成员请求和成员响应中获取活动消息
	assert.NotNil(t, membershipResponseWithAlivePeers.Load())
	assert.NotNil(t, membershipRequest.Load())

	t.Run("alive message", func(t *testing.T) {
		t.Parallel()
//生成新的对等-P4
		p4 := createDiscoveryInstance(4678, "p1", nil)
		defer p4.Stop()
//已验证记录消息
		validatedMessages := make(chan *proto.SignedGossipMessage, 5000)
		p4.comm.recordValidation(validatedMessages)
		tmpMsgs := make(chan *proto.SignedGossipMessage, 5000)
//将发送给p1的消息重放到p4中，并将其保存到临时通道中
		for msg := range aliveMessagesReceived {
			p4.comm.incMsgs <- wrapReceivedMessage(msg)
			tmpMsgs <- msg
		}

//模拟P4接收到的消息进入消息存储
		policy := proto.NewGossipMessageComparator(0)
		msgStore := msgstore.NewMessageStore(policy, func(_ interface{}) {})
		close(tmpMsgs)
		for msg := range tmpMsgs {
			if msgStore.Add(msg) {
//如果可以将消息添加到消息存储中，请确保已验证该消息
				expectedMessage := <-validatedMessages
				assert.Equal(t, expectedMessage, msg)
			}
		}
//确保我们没有验证任何其他消息。
		assert.Empty(t, validatedMessages)
	})

	req := membershipRequest.Load().(*proto.SignedGossipMessage)
	res := membershipResponseWithDeadPeers.Load().(*proto.SignedGossipMessage)
//确保成员响应中同时包含活动对等端和死对等端
	assert.Len(t, res.GetMemRes().GetAlive(), 2)
	assert.Len(t, res.GetMemRes().GetDead(), 1)

	for _, testCase := range []struct {
		name                  string
		expectedAliveMessages int
		port                  int
		message               *proto.SignedGossipMessage
		shouldBeReValidated   bool
	}{
		{
			name:                  "membership request",
			expectedAliveMessages: 1,
			message:               req,
			port:                  4679,
			shouldBeReValidated:   true,
		},
		{
			name:                  "membership response",
			expectedAliveMessages: 3,
			message:               res,
			port:                  4680,
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			p := createDiscoveryInstance(testCase.port, "p", nil)
			defer p.Stop()
//已验证记录消息
			validatedMessages := make(chan *proto.SignedGossipMessage, testCase.expectedAliveMessages)
			p.comm.recordValidation(validatedMessages)

			p.comm.incMsgs <- wrapReceivedMessage(testCase.message)
//Ensure all messages were validated
			for i := 0; i < testCase.expectedAliveMessages; i++ {
				validatedMsg := <-validatedMessages
//直接发送要包含在消息存储中的消息
				p.comm.incMsgs <- wrapReceivedMessage(validatedMsg)
			}
//等待消息验证
			for i := 0; i < testCase.expectedAliveMessages; i++ {
				<-validatedMessages
			}
//应验证不超过testcase.expectedAliveMessages
			assert.Empty(t, validatedMessages)

			if !testCase.shouldBeReValidated {
//重新提交消息两次，并确保消息未经验证。
//如果验证了它，则会出现恐慌，因为排队到validatemessages通道
//将尝试并关闭通道。
				close(validatedMessages)
			}
			p.comm.incMsgs <- wrapReceivedMessage(testCase.message)
			p.comm.incMsgs <- wrapReceivedMessage(testCase.message)
//等待通道大小为零。这意味着至少处理了一条消息。
			waitUntilOrFail(t, func() bool {
				return len(p.comm.incMsgs) == 0
			})
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	nodeNum := 5
	bootPeers := []string{bootPeer(6611), bootPeer(6612)}
	instances := []*gossipInstance{}

	inst := createDiscoveryInstance(6611, "d1", bootPeers)
	instances = append(instances, inst)

	inst = createDiscoveryInstance(6612, "d2", bootPeers)
	instances = append(instances, inst)

	for i := 3; i <= nodeNum; i++ {
		id := fmt.Sprintf("d%d", i)
		inst = createDiscoveryInstance(6610+i, id, bootPeers)
		instances = append(instances, inst)
	}

	fullMembership := func() bool {
		return nodeNum-1 == len(instances[nodeNum-1].GetMembership())
	}

	waitUntilOrFail(t, fullMembership)

	instances[0].UpdateMetadata([]byte("bla bla"))
	instances[nodeNum-1].UpdateEndpoint("localhost:5511")

	checkMembership := func() bool {
		for _, member := range instances[nodeNum-1].GetMembership() {
			if string(member.PKIid) == instances[0].comm.id {
				if "bla bla" != string(member.Metadata) {
					return false
				}
			}
		}

		for _, member := range instances[0].GetMembership() {
			if string(member.PKIid) == instances[nodeNum-1].comm.id {
				if "localhost:5511" != string(member.Endpoint) {
					return false
				}
			}
		}
		return true
	}

	waitUntilOrFail(t, checkMembership)
	stopInstances(t, instances)
}

func TestInitiateSync(t *testing.T) {
	t.Parallel()
	nodeNum := 10
	bootPeers := []string{bootPeer(3611), bootPeer(3612)}
	instances := []*gossipInstance{}

	toDie := int32(0)
	for i := 1; i <= nodeNum; i++ {
		id := fmt.Sprintf("d%d", i)
		inst := createDiscoveryInstanceWithNoGossip(3610+i, id, bootPeers)
		instances = append(instances, inst)
		go func() {
			for {
				if atomic.LoadInt32(&toDie) == int32(1) {
					return
				}
				time.Sleep(getAliveExpirationTimeout() / 3)
				inst.InitiateSync(9)
			}
		}()
	}
	time.Sleep(getAliveExpirationTimeout() * 4)
	assertMembership(t, instances, nodeNum-1)
	atomic.StoreInt32(&toDie, int32(1))
	stopInstances(t, instances)
}

func TestSelf(t *testing.T) {
	t.Parallel()
	inst := createDiscoveryInstance(13463, "d1", []string{})
	defer inst.Stop()
	env := inst.Self().Envelope
	sMsg, err := env.ToGossipMessage()
	assert.NoError(t, err)
	member := sMsg.GetAliveMsg().Membership
	assert.Equal(t, "localhost:13463", member.Endpoint)
	assert.Equal(t, []byte("localhost:13463"), member.PkiId)

	assert.Equal(t, "localhost:13463", inst.Self().Endpoint)
	assert.Equal(t, common.PKIidType("localhost:13463"), inst.Self().PKIid)
}

func TestExpiration(t *testing.T) {
	t.Parallel()
	nodeNum := 5
	bootPeers := []string{bootPeer(2611), bootPeer(2612)}
	instances := []*gossipInstance{}

	inst := createDiscoveryInstance(2611, "d1", bootPeers)
	instances = append(instances, inst)

	inst = createDiscoveryInstance(2612, "d2", bootPeers)
	instances = append(instances, inst)

	for i := 3; i <= nodeNum; i++ {
		id := fmt.Sprintf("d%d", i)
		inst = createDiscoveryInstance(2610+i, id, bootPeers)
		instances = append(instances, inst)
	}

	assertMembership(t, instances, nodeNum-1)

	waitUntilOrFailBlocking(t, instances[nodeNum-1].Stop)
	waitUntilOrFailBlocking(t, instances[nodeNum-2].Stop)

	assertMembership(t, instances[:len(instances)-2], nodeNum-3)

	stopAction := &sync.WaitGroup{}
	for i, inst := range instances {
		if i+2 == nodeNum {
			break
		}
		stopAction.Add(1)
		go func(inst *gossipInstance) {
			defer stopAction.Done()
			inst.Stop()
		}(inst)
	}

	waitUntilOrFailBlocking(t, stopAction.Wait)
}

func TestGetFullMembership(t *testing.T) {
	t.Parallel()
	nodeNum := 15
	bootPeers := []string{bootPeer(5511), bootPeer(5512)}
	instances := []*gossipInstance{}
	var inst *gossipInstance

	for i := 3; i <= nodeNum; i++ {
		id := fmt.Sprintf("d%d", i)
		inst = createDiscoveryInstance(5510+i, id, bootPeers)
		instances = append(instances, inst)
	}

	time.Sleep(time.Second)

	inst = createDiscoveryInstance(5511, "d1", bootPeers)
	instances = append(instances, inst)

	inst = createDiscoveryInstance(5512, "d2", bootPeers)
	instances = append(instances, inst)

	assertMembership(t, instances, nodeNum-1)

//确保将内部终结点传播给所有人
	for _, inst := range instances {
		for _, member := range inst.GetMembership() {
			assert.NotEmpty(t, member.InternalEndpoint)
			assert.NotEmpty(t, member.Endpoint)
		}
	}

//检查lookup（）是否有效
	for _, inst := range instances {
		for _, member := range inst.GetMembership() {
			assert.Equal(t, string(member.PKIid), inst.Lookup(member.PKIid).Endpoint)
			assert.Equal(t, member.PKIid, inst.Lookup(member.PKIid).PKIid)
		}
	}

	stopInstances(t, instances)
}

func TestGossipDiscoveryStopping(t *testing.T) {
	t.Parallel()
	inst := createDiscoveryInstance(9611, "d1", []string{bootPeer(9611)})
	time.Sleep(time.Second)
	waitUntilOrFailBlocking(t, inst.Stop)
}

func TestGossipDiscoverySkipConnectingToLocalhostBootstrap(t *testing.T) {
	t.Parallel()
	inst := createDiscoveryInstance(11611, "d1", []string{"localhost:11611", "127.0.0.1:11611"})
	inst.comm.lock.Lock()
	inst.comm.mock = &mock.Mock{}
	inst.comm.mock.On("SendToPeer", mock.Anything, mock.Anything).Run(func(mock.Arguments) {
		t.Fatal("Should not have connected to any peer")
	})
	inst.comm.mock.On("Ping", mock.Anything).Run(func(mock.Arguments) {
		t.Fatal("Should not have connected to any peer")
	})
	inst.comm.lock.Unlock()
	time.Sleep(time.Second * 3)
	waitUntilOrFailBlocking(t, inst.Stop)
}

func TestConvergence(t *testing.T) {
	t.Parallel()
//脚本：
//{Booer-Peer-:[Peer-List] }
//d1:d2、d3、d4_
//d5:d6、d7、d8_
//D9:D10、D11、D12_
//将所有引导节点与D13连接
//取下D13
//确保仍然是完全会员
	instances := []*gossipInstance{}
	for _, i := range []int{1, 5, 9} {
		bootPort := 4610 + i
		id := fmt.Sprintf("d%d", i)
		leader := createDiscoveryInstance(bootPort, id, []string{})
		instances = append(instances, leader)
		for minionIndex := 1; minionIndex <= 3; minionIndex++ {
			id := fmt.Sprintf("d%d", i+minionIndex)
			minion := createDiscoveryInstance(4610+minionIndex+i, id, []string{bootPeer(bootPort)})
			instances = append(instances, minion)
		}
	}

	assertMembership(t, instances, 3)
	connector := createDiscoveryInstance(4623, "d13", []string{bootPeer(4611), bootPeer(4615), bootPeer(4619)})
	instances = append(instances, connector)
	assertMembership(t, instances, 12)
	connector.Stop()
	instances = instances[:len(instances)-1]
	assertMembership(t, instances, 11)
	stopInstances(t, instances)
}

func TestDisclosurePolicyWithPull(t *testing.T) {
	t.Parallel()
//场景：运行两组模拟两个组织的对等方：
//p0，p1，p2，p3，p4_
//P5、P6、P7、P8、P9_
//只有具有偶数ID的对等机才具有外部地址
//只有这些同龄人才能被公布给另一组同龄人，
//而唯一需要了解的是同龄人
//他们自己有一个偶数身份。
//此外，不同组中的对等点不应该知道
//其他同行。

//这是一个引导映射，它为每个对等机匹配自己的引导对等机。
//在实践（生产）中，对等方应该只使用其组织的对等方作为引导对等方，
//但是发现层对组织一无所知。
	bootPeerMap := map[int]int{
		8610: 8616,
		8611: 8610,
		8612: 8610,
		8613: 8610,
		8614: 8610,
		8615: 8616,
		8616: 8610,
		8617: 8616,
		8618: 8616,
		8619: 8616,
	}

//这个映射匹配每个对等点，它应该在测试场景中知道的对等点。
	peersThatShouldBeKnownToPeers := map[int][]int{
		8610: {8611, 8612, 8613, 8614, 8616, 8618},
		8611: {8610, 8612, 8613, 8614},
		8612: {8610, 8611, 8613, 8614, 8616, 8618},
		8613: {8610, 8611, 8612, 8614},
		8614: {8610, 8611, 8612, 8613, 8616, 8618},
		8615: {8616, 8617, 8618, 8619},
		8616: {8610, 8612, 8614, 8615, 8617, 8618, 8619},
		8617: {8615, 8616, 8618, 8619},
		8618: {8610, 8612, 8614, 8615, 8616, 8617, 8619},
		8619: {8615, 8616, 8617, 8618},
	}
//在两个组中创建对等
	instances1, instances2 := createDisjointPeerGroupsWithNoGossip(bootPeerMap)
//睡一会儿，让他们成为会员。这次应该足够了
//因为实例被配置为从
//最多10个同龄人（这会导致从每个人身上拉下来）
	waitUntilOrFail(t, func() bool {
		for _, inst := range append(instances1, instances2...) {
//确保预期成员的大小与实际成员的大小相同
//每个对等体。
			portsOfKnownMembers := portsOfMembers(inst.GetMembership())
			if len(peersThatShouldBeKnownToPeers[inst.port]) != len(portsOfKnownMembers) {
				return false
			}
		}
		return true
	})
	for _, inst := range append(instances1, instances2...) {
		portsOfKnownMembers := portsOfMembers(inst.GetMembership())
//确保预期成员资格等于实际成员资格
//每个对等体。portsofmembers返回已排序的切片，因此assert.equal执行作业。
		assert.Equal(t, peersThatShouldBeKnownToPeers[inst.port], portsOfKnownMembers)
//接下来，检查内部端点是否没有跨组泄漏，
		for _, knownPeer := range inst.GetMembership() {
//如果知道内部端点，请确保对等端位于同一组中。
//除非所讨论的对等方是具有公共地址的对等方。
//当我们发送会员资格请求时，我们无法控制自己所披露的内容。
			if len(knownPeer.InternalEndpoint) > 0 && inst.port%2 != 0 {
				bothInGroup1 := portOfEndpoint(knownPeer.Endpoint) < 8615 && inst.port < 8615
				bothInGroup2 := portOfEndpoint(knownPeer.Endpoint) >= 8615 && inst.port >= 8615
				assert.True(t, bothInGroup1 || bothInGroup2, "%v knows about %v's internal endpoint", inst.port, knownPeer.InternalEndpoint)
			}
		}
	}

	t.Log("Shutting down instance 0...")
//现在，我们关闭实例0并确保不应该知道它的对等机，
//通过会员申请不知道
	stopInstances(t, []*gossipInstance{instances1[0]})
	time.Sleep(time.Second * 6)
	for _, inst := range append(instances1[1:], instances2...) {
		if peersThatShouldBeKnownToPeers[inst.port][0] == 8610 {
			assert.Equal(t, 1, inst.Discovery.(*gossipDiscoveryImpl).deadMembership.Size())
		} else {
			assert.Equal(t, 0, inst.Discovery.(*gossipDiscoveryImpl).deadMembership.Size())
		}
	}
	stopInstances(t, instances1[1:])
	stopInstances(t, instances2)
}

func createDisjointPeerGroupsWithNoGossip(bootPeerMap map[int]int) ([]*gossipInstance, []*gossipInstance) {
	instances1 := []*gossipInstance{}
	instances2 := []*gossipInstance{}
	for group := 0; group < 2; group++ {
		for i := 0; i < 5; i++ {
			group := group
			id := fmt.Sprintf("id%d", group*5+i)
			port := 8610 + group*5 + i
			bootPeers := []string{bootPeer(bootPeerMap[port])}
			pol := discPolForPeer(port)
			inst := createDiscoveryInstanceWithNoGossipWithDisclosurePolicy(8610+group*5+i, id, bootPeers, pol)
			inst.initiateSync(getAliveExpirationTimeout()/3, 10)
			if group == 0 {
				instances1 = append(instances1, inst)
			} else {
				instances2 = append(instances2, inst)
			}
		}
	}
	return instances1, instances2
}

func discPolForPeer(selfPort int) DisclosurePolicy {
	return func(remotePeer *NetworkMember) (Sieve, EnvelopeFilter) {
		targetPortStr := strings.Split(remotePeer.Endpoint, ":")[1]
		targetPort, _ := strconv.ParseInt(targetPortStr, 10, 64)
		return func(msg *proto.SignedGossipMessage) bool {
				portOfAliveMsgStr := strings.Split(msg.GetAliveMsg().Membership.Endpoint, ":")[1]
				portOfAliveMsg, _ := strconv.ParseInt(portOfAliveMsgStr, 10, 64)

				if portOfAliveMsg < 8615 && targetPort < 8615 {
					return true
				}
				if portOfAliveMsg >= 8615 && targetPort >= 8615 {
					return true
				}

//否则，将具有偶数ID的对等端公开给具有偶数ID的其他对等端
				return portOfAliveMsg%2 == 0 && targetPort%2 == 0
			}, func(msg *proto.SignedGossipMessage) *proto.Envelope {
				envelope := protoG.Clone(msg.Envelope).(*proto.Envelope)
				if selfPort < 8615 && targetPort >= 8615 {
					envelope.SecretEnvelope = nil
				}

				if selfPort >= 8615 && targetPort < 8615 {
					envelope.SecretEnvelope = nil
				}

				return envelope
			}
	}
}

func TestConfigFromFile(t *testing.T) {
	preAliveTimeInterval := getAliveTimeInterval()
	preAliveExpirationTimeout := getAliveExpirationTimeout()
	preAliveExpirationCheckInterval := getAliveExpirationCheckInterval()
	preReconnectInterval := getReconnectInterval()

//恢复配置值以避免影响其他测试
	defer func() {
		SetAliveTimeInterval(preAliveTimeInterval)
		SetAliveExpirationTimeout(preAliveExpirationTimeout)
		SetAliveExpirationCheckInterval(preAliveExpirationCheckInterval)
		SetReconnectInterval(preReconnectInterval)
	}()

//验证在缺少配置时是否使用默认值
	viper.Reset()
	aliveExpirationCheckInterval = 0 * time.Second
	assert.Equal(t, time.Duration(5)*time.Second, getAliveTimeInterval())
	assert.Equal(t, time.Duration(25)*time.Second, getAliveExpirationTimeout())
	assert.Equal(t, time.Duration(25)*time.Second/10, getAliveExpirationCheckInterval())
	assert.Equal(t, time.Duration(25)*time.Second, getReconnectInterval())

//验证是否从配置文件读取值
	viper.Reset()
	aliveExpirationCheckInterval = 0 * time.Second
	viper.SetConfigName("core")
	viper.SetEnvPrefix("CORE")
	configtest.AddDevConfigPath(nil)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(5)*time.Second, getAliveTimeInterval())
	assert.Equal(t, time.Duration(25)*time.Second, getAliveExpirationTimeout())
	assert.Equal(t, time.Duration(25)*time.Second/10, getAliveExpirationCheckInterval())
	assert.Equal(t, time.Duration(25)*time.Second, getReconnectInterval())
}

func TestMsgStoreExpiration(t *testing.T) {
//启动4个实例，等待成员身份生成，停止2个实例
//检查2个正在运行的实例中的成员身份是否变为2
//等待到期，并检查在运行的实例中是否删除了映射中的活动消息和相关实体
	t.Parallel()
	nodeNum := 4
	bootPeers := []string{bootPeer(12611), bootPeer(12612)}
	instances := []*gossipInstance{}

	inst := createDiscoveryInstance(12611, "d1", bootPeers)
	instances = append(instances, inst)

	inst = createDiscoveryInstance(12612, "d2", bootPeers)
	instances = append(instances, inst)

	for i := 3; i <= nodeNum; i++ {
		id := fmt.Sprintf("d%d", i)
		inst = createDiscoveryInstance(12610+i, id, bootPeers)
		instances = append(instances, inst)
	}

	assertMembership(t, instances, nodeNum-1)

	waitUntilOrFailBlocking(t, instances[nodeNum-1].Stop)
	waitUntilOrFailBlocking(t, instances[nodeNum-2].Stop)

	assertMembership(t, instances[:len(instances)-2], nodeNum-3)

	checkMessages := func() bool {
		for _, inst := range instances[:len(instances)-2] {
			for _, downInst := range instances[len(instances)-2:] {
				downCastInst := inst.discoveryImpl()
				downCastInst.lock.RLock()
				if _, exist := downCastInst.aliveLastTS[string(downInst.discoveryImpl().self.PKIid)]; exist {
					downCastInst.lock.RUnlock()
					return false
				}
				if _, exist := downCastInst.deadLastTS[string(downInst.discoveryImpl().self.PKIid)]; exist {
					downCastInst.lock.RUnlock()
					return false
				}
				if _, exist := downCastInst.id2Member[string(downInst.discoveryImpl().self.PKIid)]; exist {
					downCastInst.lock.RUnlock()
					return false
				}
				if downCastInst.aliveMembership.MsgByID(downInst.discoveryImpl().self.PKIid) != nil {
					downCastInst.lock.RUnlock()
					return false
				}
				if downCastInst.deadMembership.MsgByID(downInst.discoveryImpl().self.PKIid) != nil {
					downCastInst.lock.RUnlock()
					return false
				}
				for _, am := range downCastInst.msgStore.Get() {
					m := am.(*proto.SignedGossipMessage).GetAliveMsg()
					if bytes.Equal(m.Membership.PkiId, downInst.discoveryImpl().self.PKIid) {
						downCastInst.lock.RUnlock()
						return false
					}
				}
				downCastInst.lock.RUnlock()
			}
		}
		return true
	}

	waitUntilTimeoutOrFail(t, checkMessages, getAliveExpirationTimeout()*(msgExpirationFactor+5))

	assertMembership(t, instances[:len(instances)-2], nodeNum-3)

	stopInstances(t, instances[:len(instances)-2])
}

func TestMsgStoreExpirationWithMembershipMessages(t *testing.T) {
//创建3个发现实例，而不进行流言通信
//Generates MembershipRequest msg for each instance using createMembershipRequest
//使用CreateAliveMessage为每个实例生成活动消息
//使用动态MSGS构建成员资格
//检查msgstore和相关地图
//使用CreateMembershipResponse为每个实例生成membershipResponse MSG
//生成新的活动MSG集并处理它们
//检查msgstore和相关地图
//等待到期并检查msgstore和相关地图
//处理存储的membershiprequest msg并检查msgstore和相关映射
//处理存储的membershipresponse msg并检查msgstore和相关映射

	t.Parallel()
	bootPeers := []string{}
	peersNum := 3
	instances := []*gossipInstance{}
	aliveMsgs := []*proto.SignedGossipMessage{}
	newAliveMsgs := []*proto.SignedGossipMessage{}
	memReqMsgs := []*proto.SignedGossipMessage{}
	memRespMsgs := make(map[int][]*proto.MembershipResponse)

	for i := 0; i < peersNum; i++ {
		id := fmt.Sprintf("d%d", i)
		inst := createDiscoveryInstanceWithNoGossip(22610+i, id, bootPeers)
		instances = append(instances, inst)
	}

//正在创建成员资格请求消息
	for i := 0; i < peersNum; i++ {
		memReqMsg, _ := instances[i].discoveryImpl().createMembershipRequest(true)
		sMsg, _ := memReqMsg.NoopSign()
		memReqMsgs = append(memReqMsgs, sMsg)
	}
//正在创建活动消息
	for i := 0; i < peersNum; i++ {
		aliveMsg, _ := instances[i].discoveryImpl().createSignedAliveMessage(true)
		aliveMsgs = append(aliveMsgs, aliveMsg)
	}

	repeatForFiltered := func(n int, filter func(i int) bool, action func(i int)) {
		for i := 0; i < n; i++ {
			if filter(i) {
				continue
			}
			action(i)
		}
	}

//活体处理
	for i := 0; i < peersNum; i++ {
		for k := 0; k < peersNum; k++ {
			instances[i].discoveryImpl().handleMsgFromComm(&dummyReceivedMessage{
				msg: aliveMsgs[k],
				info: &proto.ConnectionInfo{
					ID: common.PKIidType(fmt.Sprintf("d%d", i)),
				},
			})
		}
	}

	checkExistence := func(instances []*gossipInstance, msgs []*proto.SignedGossipMessage, index int, i int, step string) {
		_, exist := instances[index].discoveryImpl().aliveLastTS[string(instances[i].discoveryImpl().self.PKIid)]
		assert.True(t, exist, fmt.Sprint(step, " Data from alive msg ", i, " doesn't exist in aliveLastTS of discovery inst ", index))

		_, exist = instances[index].discoveryImpl().id2Member[string(string(instances[i].discoveryImpl().self.PKIid))]
		assert.True(t, exist, fmt.Sprint(step, " id2Member mapping doesn't exist for alive msg ", i, " of discovery inst ", index))

		assert.NotNil(t, instances[index].discoveryImpl().aliveMembership.MsgByID(instances[i].discoveryImpl().self.PKIid), fmt.Sprint(step, " Alive msg", i, " not exist in aliveMembership of discovery inst ", index))

		assert.Contains(t, instances[index].discoveryImpl().msgStore.Get(), msgs[i], fmt.Sprint(step, " Alive msg ", i, "not stored in store of discovery inst ", index))
	}

	checkAliveMsgExist := func(instances []*gossipInstance, msgs []*proto.SignedGossipMessage, index int, step string) {
		instances[index].discoveryImpl().lock.RLock()
		defer instances[index].discoveryImpl().lock.RUnlock()
		repeatForFiltered(peersNum,
			func(k int) bool {
				return k == index
			},
			func(k int) {
				checkExistence(instances, msgs, index, k, step)
			})
	}

//Checking is Alive was processed
	for i := 0; i < peersNum; i++ {
		checkAliveMsgExist(instances, aliveMsgs, i, "[Step 1 - processing aliveMsg]")
	}

//在所有实例都具有完全成员身份时创建MembershipResponse
	for i := 0; i < peersNum; i++ {
		peerToResponse := &NetworkMember{
			Metadata:         []byte{},
			PKIid:            []byte(fmt.Sprintf("localhost:%d", 22610+i)),
			Endpoint:         fmt.Sprintf("localhost:%d", 22610+i),
			InternalEndpoint: fmt.Sprintf("localhost:%d", 22610+i),
		}
		memRespMsgs[i] = []*proto.MembershipResponse{}
		repeatForFiltered(peersNum,
			func(k int) bool {
				return k == i
			},
			func(k int) {
				aliveMsg, _ := instances[k].discoveryImpl().createSignedAliveMessage(true)
				memResp := instances[k].discoveryImpl().createMembershipResponse(aliveMsg, peerToResponse)
				memRespMsgs[i] = append(memRespMsgs[i], memResp)
			})
	}

//重新创建具有最高序列号的活动msg，以确保memreq和memresp中的活动msg较旧。
	for i := 0; i < peersNum; i++ {
		aliveMsg, _ := instances[i].discoveryImpl().createSignedAliveMessage(true)
		newAliveMsgs = append(newAliveMsgs, aliveMsg)
	}

//处理新的活动集
	for i := 0; i < peersNum; i++ {
		for k := 0; k < peersNum; k++ {
			instances[i].discoveryImpl().handleMsgFromComm(&dummyReceivedMessage{
				msg: newAliveMsgs[k],
				info: &proto.ConnectionInfo{
					ID: common.PKIidType(fmt.Sprintf("d%d", i)),
				},
			})
		}
	}

//正在检查是否已处理新的活动
	for i := 0; i < peersNum; i++ {
		checkAliveMsgExist(instances, newAliveMsgs, i, "[Step 2 - proccesing aliveMsg]")
	}

	checkAliveMsgNotExist := func(instances []*gossipInstance, msgs []*proto.SignedGossipMessage, index int, step string) {
		instances[index].discoveryImpl().lock.RLock()
		defer instances[index].discoveryImpl().lock.RUnlock()
		assert.Empty(t, instances[index].discoveryImpl().aliveLastTS, fmt.Sprint(step, " Data from alive msg still exists in aliveLastTS of discovery inst ", index))
		assert.Empty(t, instances[index].discoveryImpl().deadLastTS, fmt.Sprint(step, " Data from alive msg still exists in deadLastTS of discovery inst ", index))
		assert.Empty(t, instances[index].discoveryImpl().id2Member, fmt.Sprint(step, " id2Member mapping still still contains data related to Alive msg: discovery inst ", index))
		assert.Empty(t, instances[index].discoveryImpl().msgStore.Get(), fmt.Sprint(step, " Expired Alive msg still stored in store of discovery inst ", index))
		assert.Zero(t, instances[index].discoveryImpl().aliveMembership.Size(), fmt.Sprint(step, " Alive membership list is not empty, discovery instance", index))
		assert.Zero(t, instances[index].discoveryImpl().deadMembership.Size(), fmt.Sprint(step, " Dead membership list is not empty, discovery instance", index))
	}

//睡眠到过期
	time.Sleep(getAliveExpirationTimeout() * (msgExpirationFactor + 5))

//Checking Alive expired
	for i := 0; i < peersNum; i++ {
		checkAliveMsgNotExist(instances, newAliveMsgs, i, "[Step3 - expiration in msg store]")
	}

//正在处理旧的成员身份请求
	for i := 0; i < peersNum; i++ {
		repeatForFiltered(peersNum,
			func(k int) bool {
				return k == i
			},
			func(k int) {
				instances[i].discoveryImpl().handleMsgFromComm(&dummyReceivedMessage{
					msg: memReqMsgs[k],
					info: &proto.ConnectionInfo{
						ID: common.PKIidType(fmt.Sprintf("d%d", i)),
					},
				})
			})
	}

//MembershipRequest处理未更改任何内容
	for i := 0; i < peersNum; i++ {
		checkAliveMsgNotExist(instances, aliveMsgs, i, "[Step4 - memReq processing after expiration]")
	}

//处理旧的（以后的）活动消息
	for i := 0; i < peersNum; i++ {
		for k := 0; k < peersNum; k++ {
			instances[i].discoveryImpl().handleMsgFromComm(&dummyReceivedMessage{
				msg: aliveMsgs[k],
				info: &proto.ConnectionInfo{
					ID: common.PKIidType(fmt.Sprintf("d%d", i)),
				},
			})
		}
	}

//活动消息处理没有更改任何内容
	for i := 0; i < peersNum; i++ {
		checkAliveMsgNotExist(instances, aliveMsgs, i, "[Step5.1 - after lost old aliveMsg process]")
		checkAliveMsgNotExist(instances, newAliveMsgs, i, "[Step5.2 - after lost new aliveMsg process]")
	}

//处理旧的成员身份响应消息
	for i := 0; i < peersNum; i++ {
		respForPeer := memRespMsgs[i]
		for _, msg := range respForPeer {
			sMsg, _ := (&proto.GossipMessage{
				Tag:   proto.GossipMessage_EMPTY,
				Nonce: uint64(0),
				Content: &proto.GossipMessage_MemRes{
					MemRes: msg,
				},
			}).NoopSign()
			instances[i].discoveryImpl().handleMsgFromComm(&dummyReceivedMessage{
				msg: sMsg,
				info: &proto.ConnectionInfo{
					ID: common.PKIidType(fmt.Sprintf("d%d", i)),
				},
			})
		}
	}

//MembershipResponse msg processing didn't change anything
	for i := 0; i < peersNum; i++ {
		checkAliveMsgNotExist(instances, aliveMsgs, i, "[Step6 - after lost MembershipResp process]")
	}

	for i := 0; i < peersNum; i++ {
		instances[i].Stop()
	}

}

func TestAliveMsgStore(t *testing.T) {
	t.Parallel()

	bootPeers := []string{}
	peersNum := 2
	instances := []*gossipInstance{}
	aliveMsgs := []*proto.SignedGossipMessage{}
	memReqMsgs := []*proto.SignedGossipMessage{}

	for i := 0; i < peersNum; i++ {
		id := fmt.Sprintf("d%d", i)
		inst := createDiscoveryInstanceWithNoGossip(32610+i, id, bootPeers)
		instances = append(instances, inst)
	}

//正在创建成员资格请求消息
	for i := 0; i < peersNum; i++ {
		memReqMsg, _ := instances[i].discoveryImpl().createMembershipRequest(true)
		sMsg, _ := memReqMsg.NoopSign()
		memReqMsgs = append(memReqMsgs, sMsg)
	}
//正在创建活动消息
	for i := 0; i < peersNum; i++ {
		aliveMsg, _ := instances[i].discoveryImpl().createSignedAliveMessage(true)
		aliveMsgs = append(aliveMsgs, aliveMsg)
	}

//检查新的活动MSG
	for _, msg := range aliveMsgs {
		assert.True(t, instances[0].discoveryImpl().msgStore.CheckValid(msg), "aliveMsgStore CheckValid returns false on new AliveMsg")
	}

//添加新的活动MSG
	for _, msg := range aliveMsgs {
		assert.True(t, instances[0].discoveryImpl().msgStore.Add(msg), "aliveMsgStore Add returns false on new AliveMsg")
	}

//检查是否存在活动的MSG
	for _, msg := range aliveMsgs {
		assert.False(t, instances[0].discoveryImpl().msgStore.CheckValid(msg), "aliveMsgStore CheckValid returns true on existing AliveMsg")
	}

//检查非活动MSG
	for _, msg := range memReqMsgs {
		assert.Panics(t, func() { instances[1].discoveryImpl().msgStore.CheckValid(msg) }, "aliveMsgStore CheckValid should panic on new MembershipRequest msg")
		assert.Panics(t, func() { instances[1].discoveryImpl().msgStore.Add(msg) }, "aliveMsgStore Add should panic on new MembershipRequest msg")
	}
}

func TestMemRespDisclosurePol(t *testing.T) {
	t.Parallel()
	pol := func(remotePeer *NetworkMember) (Sieve, EnvelopeFilter) {
		return func(_ *proto.SignedGossipMessage) bool {
				return remotePeer.Endpoint == "localhost:7880"
			}, func(m *proto.SignedGossipMessage) *proto.Envelope {
				return m.Envelope
			}
	}
	d1 := createDiscoveryInstanceThatGossips(7878, "d1", []string{}, true, pol)
	defer d1.Stop()
	d2 := createDiscoveryInstanceThatGossips(7879, "d2", []string{"localhost:7878"}, true, noopPolicy)
	defer d2.Stop()
	d3 := createDiscoveryInstanceThatGossips(7880, "d3", []string{"localhost:7878"}, true, noopPolicy)
	defer d3.Stop()
//d1和d3都互相了解，也了解d2
	assertMembership(t, []*gossipInstance{d1, d3}, 2)
//d2不知道任何一个，因为由于自定义策略，引导对等机忽略了它。
	assertMembership(t, []*gossipInstance{d2}, 0)
	assert.Zero(t, d2.receivedMsgCount())
	assert.NotZero(t, d2.sentMsgCount())
}

func TestMembersByID(t *testing.T) {
	members := Members{
		{PKIid: common.PKIidType("p0"), Endpoint: "p0"},
		{PKIid: common.PKIidType("p1"), Endpoint: "p1"},
	}
	byID := members.ByID()
	assert.Len(t, byID, 2)
	assert.Equal(t, "p0", byID["p0"].Endpoint)
	assert.Equal(t, "p1", byID["p1"].Endpoint)
}

func TestFilter(t *testing.T) {
	members := Members{
		{PKIid: common.PKIidType("p0"), Endpoint: "p0", Properties: &proto.Properties{
			Chaincodes: []*proto.Chaincode{{Name: "cc", Version: "1.0"}},
		}},
		{PKIid: common.PKIidType("p1"), Endpoint: "p1", Properties: &proto.Properties{
			Chaincodes: []*proto.Chaincode{{Name: "cc", Version: "2.0"}},
		}},
	}
	res := members.Filter(func(member NetworkMember) bool {
		cc := member.Properties.Chaincodes[0]
		return cc.Version == "2.0" && cc.Name == "cc"
	})
	assert.Equal(t, Members{members[1]}, res)
}

func TestMap(t *testing.T) {
	members := Members{
		{PKIid: common.PKIidType("p0"), Endpoint: "p0"},
		{PKIid: common.PKIidType("p1"), Endpoint: "p1"},
	}
	expectedMembers := Members{
		{PKIid: common.PKIidType("p0"), Endpoint: "p0", Properties: &proto.Properties{LedgerHeight: 2}},
		{PKIid: common.PKIidType("p1"), Endpoint: "p1", Properties: &proto.Properties{LedgerHeight: 2}},
	}

	addProperty := func(member NetworkMember) NetworkMember {
		member.Properties = &proto.Properties{
			LedgerHeight: 2,
		}
		return member
	}

	assert.Equal(t, expectedMembers, members.Map(addProperty))
//确保原始成员没有更改
	assert.Nil(t, members[0].Properties)
	assert.Nil(t, members[1].Properties)
}

func TestMembersIntersect(t *testing.T) {
	members1 := Members{
		{PKIid: common.PKIidType("p0"), Endpoint: "p0"},
		{PKIid: common.PKIidType("p1"), Endpoint: "p1"},
	}
	members2 := Members{
		{PKIid: common.PKIidType("p1"), Endpoint: "p1"},
		{PKIid: common.PKIidType("p2"), Endpoint: "p2"},
	}
	assert.Equal(t, Members{{PKIid: common.PKIidType("p1"), Endpoint: "p1"}}, members1.Intersect(members2))
}

func waitUntilOrFail(t *testing.T, pred func() bool) {
	waitUntilTimeoutOrFail(t, pred, timeout)
}

func waitUntilTimeoutOrFail(t *testing.T, pred func() bool, timeout time.Duration) {
	start := time.Now()
	limit := start.UnixNano() + timeout.Nanoseconds()
	for time.Now().UnixNano() < limit {
		if pred() {
			return
		}
		time.Sleep(timeout / 10)
	}
	assert.Fail(t, "Timeout expired!")
}

func waitUntilOrFailBlocking(t *testing.T, f func()) {
	successChan := make(chan struct{}, 1)
	go func() {
		f()
		successChan <- struct{}{}
	}()
	select {
	case <-time.NewTimer(timeout).C:
		break
	case <-successChan:
		return
	}
	assert.Fail(t, "Timeout expired!")
}

func stopInstances(t *testing.T, instances []*gossipInstance) {
	stopAction := &sync.WaitGroup{}
	for _, inst := range instances {
		stopAction.Add(1)
		go func(inst *gossipInstance) {
			defer stopAction.Done()
			inst.Stop()
		}(inst)
	}

	waitUntilOrFailBlocking(t, stopAction.Wait)
}

func assertMembership(t *testing.T, instances []*gossipInstance, expectedNum int) {
	wg := sync.WaitGroup{}
	wg.Add(len(instances))

	ctx, cancelation := context.WithTimeout(context.Background(), timeout)
	defer cancelation()

	for _, inst := range instances {
		go func(ctx context.Context, i *gossipInstance) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(timeout / 10):
					if len(i.GetMembership()) == expectedNum {
						return
					}
				}
			}
		}(ctx, inst)
	}

	wg.Wait()
	assert.NoError(t, ctx.Err(), "Timeout expired!")
}

func portsOfMembers(members []NetworkMember) []int {
	ports := make([]int, len(members))
	for i := range members {
		ports[i] = portOfEndpoint(members[i].Endpoint)
	}
	sort.Ints(ports)
	return ports
}

func portOfEndpoint(endpoint string) int {
	port, _ := strconv.ParseInt(strings.Split(endpoint, ":")[1], 10, 64)
	return int(port)
}
