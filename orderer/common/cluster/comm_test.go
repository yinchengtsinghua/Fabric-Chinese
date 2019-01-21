
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package cluster_test

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/flogging"
	comm_utils "github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/cluster/mocks"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

const (
	testChannel  = "test"
	testChannel2 = "test2"
	timeout      = time.Second * 10
)

var (
//生成TLS密钥对的CA。
//我们只使用一个CA，因为身份验证
//基于TLS固定
	ca = createCAOrPanic()

	lastNodeID uint64

	testSubReq = &orderer.SubmitRequest{
		Channel: "test",
	}

	testStepReq = &orderer.StepRequest{
		Channel: "test",
		Payload: []byte("test"),
	}

	testStepRes = &orderer.StepResponse{
		Payload: []byte("test"),
	}

	fooReq = &orderer.StepRequest{
		Channel: "foo",
	}

	fooRes = &orderer.StepResponse{
		Payload: []byte("foo"),
	}

	barReq = &orderer.StepRequest{
		Channel: "bar",
	}

	barRes = &orderer.StepResponse{
		Payload: []byte("bar"),
	}

	channelExtractor = &mockChannelExtractor{}
)

func nextUnusedID() uint64 {
	return atomic.AddUint64(&lastNodeID, 1)
}

func createCAOrPanic() tlsgen.CA {
	ca, err := tlsgen.NewCA()
	if err != nil {
		panic(fmt.Sprintf("failed creating CA: %+v", err))
	}
	return ca
}

type mockChannelExtractor struct{}

func (*mockChannelExtractor) TargetChannel(msg proto.Message) string {
	if stepReq, isStepReq := msg.(*orderer.StepRequest); isStepReq {
		return stepReq.Channel
	}
	if subReq, isSubReq := msg.(*orderer.SubmitRequest); isSubReq {
		return string(subReq.Channel)
	}
	return ""
}

type clusterNode struct {
	dialer       *cluster.PredicateDialer
	handler      *mocks.Handler
	nodeInfo     cluster.RemoteNode
	srv          *comm_utils.GRPCServer
	bindAddress  string
	clientConfig comm_utils.ClientConfig
	serverConfig comm_utils.ServerConfig
	c            *cluster.Comm
}

func (cn *clusterNode) Submit(stream orderer.Cluster_SubmitServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	res, err := cn.c.DispatchSubmit(stream.Context(), req)
	if err != nil {
		return err
	}
	return stream.Send(res)
}

func (cn *clusterNode) Step(ctx context.Context, msg *orderer.StepRequest) (*orderer.StepResponse, error) {
	return cn.c.DispatchStep(ctx, msg)
}

func (cn *clusterNode) resurrect() {
	gRPCServer, err := comm_utils.NewGRPCServer(cn.bindAddress, cn.serverConfig)
	if err != nil {
		panic(fmt.Errorf("failed starting gRPC server: %v", err))
	}
	cn.srv = gRPCServer
	orderer.RegisterClusterServer(gRPCServer.Server(), cn)
	go cn.srv.Start()
}

func (cn *clusterNode) stop() {
	cn.srv.Stop()
	cn.c.Shutdown()
}

func (cn *clusterNode) renewCertificates() {
	clientKeyPair, err := ca.NewClientCertKeyPair()
	if err != nil {
		panic(fmt.Errorf("failed creating client certificate %v", err))
	}
	serverKeyPair, err := ca.NewServerCertKeyPair("127.0.0.1")
	if err != nil {
		panic(fmt.Errorf("failed creating server certificate %v", err))
	}

	cn.nodeInfo.ClientTLSCert = clientKeyPair.TLSCert.Raw
	cn.nodeInfo.ServerTLSCert = serverKeyPair.TLSCert.Raw

	cn.serverConfig.SecOpts.Certificate = serverKeyPair.Cert
	cn.serverConfig.SecOpts.Key = serverKeyPair.Key

	cn.clientConfig.SecOpts.Key = clientKeyPair.Key
	cn.clientConfig.SecOpts.Certificate = clientKeyPair.Cert
	cn.dialer.SetConfig(cn.clientConfig)
}

func newTestNode(t *testing.T) *clusterNode {
	serverKeyPair, err := ca.NewServerCertKeyPair("127.0.0.1")
	assert.NoError(t, err)

	clientKeyPair, _ := ca.NewClientCertKeyPair()

	handler := &mocks.Handler{}
	clientConfig := comm_utils.ClientConfig{
		AsyncConnect: true,
		Timeout:      time.Hour,
		SecOpts: &comm_utils.SecureOptions{
			RequireClientCert: true,
			Key:               clientKeyPair.Key,
			Certificate:       clientKeyPair.Cert,
			ServerRootCAs:     [][]byte{ca.CertBytes()},
			UseTLS:            true,
			ClientRootCAs:     [][]byte{ca.CertBytes()},
		},
	}

	dialer := cluster.NewTLSPinningDialer(clientConfig)

	srvConfig := comm_utils.ServerConfig{
		SecOpts: &comm_utils.SecureOptions{
			Key:         serverKeyPair.Key,
			Certificate: serverKeyPair.Cert,
			UseTLS:      true,
		},
	}
	gRPCServer, err := comm_utils.NewGRPCServer("127.0.0.1:", srvConfig)
	assert.NoError(t, err)

	tstSrv := &clusterNode{
		dialer:       dialer,
		clientConfig: clientConfig,
		serverConfig: srvConfig,
		bindAddress:  gRPCServer.Address(),
		handler:      handler,
		nodeInfo: cluster.RemoteNode{
			Endpoint:      gRPCServer.Address(),
			ID:            nextUnusedID(),
			ServerTLSCert: serverKeyPair.TLSCert.Raw,
			ClientTLSCert: clientKeyPair.TLSCert.Raw,
		},
		srv: gRPCServer,
	}

	tstSrv.c = &cluster.Comm{
		Logger:       flogging.MustGetLogger("test"),
		Chan2Members: make(cluster.MembersByChannel),
		H:            handler,
		ChanExt:      channelExtractor,
		Connections:  cluster.NewConnectionStore(dialer),
	}

	orderer.RegisterClusterServer(gRPCServer.Server(), tstSrv)
	go gRPCServer.Start()
	return tstSrv
}

func TestBasic(t *testing.T) {
	t.Parallel()
//场景：生成2个节点并互相发送的基本测试
//预期回送的消息

	node1 := newTestNode(t)
	node2 := newTestNode(t)

	defer node1.stop()
	defer node2.stop()

	node1.handler.On("OnStep", testChannel, node2.nodeInfo.ID, mock.Anything).Return(testStepRes, nil)
	node2.handler.On("OnStep", testChannel, node1.nodeInfo.ID, mock.Anything).Return(testStepRes, nil)

	config := []cluster.RemoteNode{node1.nodeInfo, node2.nodeInfo}
	node1.c.Configure(testChannel, config)
	node2.c.Configure(testChannel, config)

	assertBiDiCommunication(t, node1, node2, testStepReq)
}

func TestUnavailableHosts(t *testing.T) {
	t.Parallel()
//场景：节点被配置为连接
//给一个倒下的主人
	node1 := newTestNode(t)
	clientConfig, err := node1.dialer.ClientConfig()
	assert.NoError(t, err)
//以下超时确保连接建立完成
//异步。如果它是同步的，则远程（）调用将
//封锁了一个小时。
	clientConfig.Timeout = time.Hour
	node1.dialer.SetConfig(clientConfig)
	defer node1.stop()

	node2 := newTestNode(t)
	node2.stop()

	node1.c.Configure(testChannel, []cluster.RemoteNode{node2.nodeInfo})
	remote, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.NoError(t, err)
	assert.NotNil(t, remote)
	_, err = remote.Step(&orderer.StepRequest{})
	assert.Contains(t, err.Error(), "rpc error")
}

func TestStreamAbort(t *testing.T) {
	t.Parallel()

//场景：节点1通过2个通道连接到节点2，
//通信呼叫的消费者接收。
//两个子场景发生：
//1）节点2的服务器证书在第一个通道中更改
//2）节点2从第一个通道的成员中退出
//在这两种情况下，应中止recv（）调用

	node2 := newTestNode(t)
	defer node2.stop()

	invalidNodeInfo := cluster.RemoteNode{
		ID:            node2.nodeInfo.ID,
		ServerTLSCert: []byte{1, 2, 3},
		ClientTLSCert: []byte{1, 2, 3},
	}

	for _, tst := range []struct {
		testName      string
		membership    []cluster.RemoteNode
		expectedError string
	}{
		{
			testName:      "Evicted from membership",
			membership:    nil,
			expectedError: "rpc error",
		},
		{
			testName:      "Changed TLS certificate",
			membership:    []cluster.RemoteNode{invalidNodeInfo},
			expectedError: "rpc error",
		},
	} {
		t.Run(tst.testName, func(t *testing.T) {
			testStreamAbort(t, node2, tst.membership, tst.expectedError)
		})
	}
	node2.handler.AssertNumberOfCalls(t, "OnSubmit", 2)
}

func testStreamAbort(t *testing.T, node2 *clusterNode, newMembership []cluster.RemoteNode, expectedError string) {
	node1 := newTestNode(t)
	defer node1.stop()

	node1.c.Configure(testChannel, []cluster.RemoteNode{node2.nodeInfo})
	node2.c.Configure(testChannel, []cluster.RemoteNode{node1.nodeInfo})
	node1.c.Configure(testChannel2, []cluster.RemoteNode{node2.nodeInfo})
	node2.c.Configure(testChannel2, []cluster.RemoteNode{node1.nodeInfo})

	var waitForReconfigWG sync.WaitGroup
	waitForReconfigWG.Add(1)

	var streamCreated sync.WaitGroup
	streamCreated.Add(1)

	node2.handler.On("OnSubmit", testChannel, node1.nodeInfo.ID, mock.Anything).Once().Run(func(_ mock.Arguments) {
//通知流已创建
		streamCreated.Done()
//在返回之前等待重新配置，以便
//recv（）将在重新配置后发生
		waitForReconfigWG.Wait()
	}).Return(nil, nil).Once()

	rm1, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.NoError(t, err)

	errorChan := make(chan error)

	go func() {
		stream, err := rm1.SubmitStream()
		assert.NoError(t, err)
//发出重新配置信号
		err = stream.Send(&orderer.SubmitRequest{
			Channel: testChannel,
		})
		assert.NoError(t, err)
		_, err = stream.Recv()
		assert.Contains(t, err.Error(), expectedError)
		errorChan <- err
	}()

	go func() {
//等待获取流引用
		streamCreated.Wait()
//重新配置通道成员身份
		node1.c.Configure(testChannel, newMembership)
		waitForReconfigWG.Done()
	}()

	<-errorChan
}

func TestDoubleReconfigure(t *testing.T) {
	t.Parallel()
//场景：生成2个节点的基本测试
//对节点1进行两次配置，并检查
//节点1的远程存根未在第二个节点中重新创建
//配置，因为它已经存在

	node1 := newTestNode(t)
	node2 := newTestNode(t)

	defer node1.stop()
	defer node2.stop()

	node1.c.Configure(testChannel, []cluster.RemoteNode{node2.nodeInfo})
	rm1, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.NoError(t, err)

	node1.c.Configure(testChannel, []cluster.RemoteNode{node2.nodeInfo})
	rm2, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.NoError(t, err)
//确保引用相等
	assert.True(t, rm1 == rm2)
}

func TestInvalidChannel(t *testing.T) {
	t.Parallel()
//场景：它命令节点1在通道上发送消息
//它不存在，也接收消息，但是
//无法从消息中提取通道。

	t.Run("channel doesn't exist", func(t *testing.T) {
		t.Parallel()
		node1 := newTestNode(t)
		defer node1.stop()

		_, err := node1.c.Remote(testChannel, 0)
		assert.EqualError(t, err, "channel test doesn't exist")
	})

	t.Run("channel cannot be extracted", func(t *testing.T) {
		t.Parallel()
		node1 := newTestNode(t)
		defer node1.stop()

		node1.c.Configure(testChannel, []cluster.RemoteNode{node1.nodeInfo})
		gt := gomega.NewGomegaWithT(t)
		gt.Eventually(func() (bool, error) {
			_, err := node1.c.Remote(testChannel, node1.nodeInfo.ID)
			return true, err
		}, time.Minute).Should(gomega.BeTrue())

		stub, err := node1.c.Remote(testChannel, node1.nodeInfo.ID)
		assert.NoError(t, err)
//空的stepRequest具有无效的空通道
		_, err = stub.Step(&orderer.StepRequest{})
		assert.EqualError(t, err, "rpc error: code = Unknown desc = badly formatted message, cannot extract channel")

//空SubmitRequest具有无效的空通道。
		_, err = node1.c.DispatchSubmit(context.Background(), &orderer.SubmitRequest{})
		assert.EqualError(t, err, "badly formatted message, cannot extract channel")
	})
}

func TestAbortRPC(t *testing.T) {
	t.Parallel()
//情节：
//（i）节点调用RPC，并在远程上下文上调用abort（）。
//并行地。即使服务器端调用尚未完成，RPC也应该返回。
//（ii）节点调用RPC，但服务器端处理时间太长，
//rpc调用提前返回。

	testCases := []struct {
		name       string
		abortFunc  func(*cluster.RemoteContext)
		rpcTimeout time.Duration
	}{
		{
			name:       "Abort() called",
			rpcTimeout: 0,
			abortFunc: func(rc *cluster.RemoteContext) {
				rc.Abort()
			},
		},
		{
			name:       "RPC timeout",
			rpcTimeout: time.Millisecond * 100,
			abortFunc:  func(*cluster.RemoteContext) {},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testAbort(t, testCase.abortFunc, testCase.rpcTimeout)
		})
	}
}

func testAbort(t *testing.T, abortFunc func(*cluster.RemoteContext), rpcTimeout time.Duration) {
	node1 := newTestNode(t)
	node1.c.RPCTimeout = rpcTimeout
	defer node1.stop()

	node2 := newTestNode(t)
	defer node2.stop()

	config := []cluster.RemoteNode{node1.nodeInfo, node2.nodeInfo}
	node1.c.Configure(testChannel, config)
	node2.c.Configure(testChannel, config)
	var onStepCalled sync.WaitGroup
	onStepCalled.Add(1)

//stuckCall确保onstep（）调用在整个测试过程中被卡住
	var stuckCall sync.WaitGroup
	stuckCall.Add(1)
//在测试结束时，释放服务器端资源
	defer stuckCall.Done()

	node2.handler.On("OnStep", testChannel, node1.nodeInfo.ID, mock.Anything).Return(testStepRes, nil).Once().Run(func(_ mock.Arguments) {
		onStepCalled.Done()
		stuckCall.Wait()
	}).Once()

	rm, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.NoError(t, err)

	go func() {
		onStepCalled.Wait()
		abortFunc(rm)
	}()
	rm.Step(testStepReq)
	node2.handler.AssertNumberOfCalls(t, "OnStep", 1)
}

func TestNoTLSCertificate(t *testing.T) {
	t.Parallel()
//场景：节点由另一个不发送消息的节点发送
//使用相互的TLS连接，因此不提供TLS证书
	node1 := newTestNode(t)
	defer node1.stop()

	node1.c.Configure(testChannel, []cluster.RemoteNode{node1.nodeInfo})

	clientConfig := comm_utils.ClientConfig{
		AsyncConnect: true,
		Timeout:      time.Millisecond * 100,
		SecOpts: &comm_utils.SecureOptions{
			ServerRootCAs: [][]byte{ca.CertBytes()},
			UseTLS:        true,
		},
	}
	cl, err := comm_utils.NewGRPCClient(clientConfig)
	assert.NoError(t, err)

	var conn *grpc.ClientConn
	gt := gomega.NewGomegaWithT(t)
	gt.Eventually(func() (bool, error) {
		conn, err = cl.NewConnection(node1.srv.Address(), "")
		return true, err
	}, time.Minute).Should(gomega.BeTrue())

	echoClient := orderer.NewClusterClient(conn)
	_, err = echoClient.Step(context.Background(), testStepReq)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = no TLS certificate sent")
}

func TestReconnect(t *testing.T) {
	t.Parallel()
//场景：节点1和节点2连接，
//节点2离线。
//节点1尝试向节点2发送消息，但失败，
//然后将节点2带回，之后
//节点1发送更多消息，应该成功
//最终向节点2发送消息。

	node1 := newTestNode(t)
	defer node1.stop()
	conf, err := node1.dialer.ClientConfig()
	assert.NoError(t, err)
	conf.Timeout = time.Hour
	node1.dialer.SetConfig(conf)

	node2 := newTestNode(t)
	node2.handler.On("OnStep", testChannel, node1.nodeInfo.ID, mock.Anything).Return(testStepRes, nil)
	defer node2.stop()

	config := []cluster.RemoteNode{node1.nodeInfo, node2.nodeInfo}
	node1.c.Configure(testChannel, config)
	node2.c.Configure(testChannel, config)

//通过关闭节点2的GRPC服务使其脱机
	node2.srv.Stop()
//获取节点2的存根。
//应该成功，因为连接是在配置时创建的
	gt := gomega.NewGomegaWithT(t)
	gt.Eventually(func() (bool, error) {
		_, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
		return true, err
	}, time.Minute).Should(gomega.BeTrue())
	stub, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
//从节点1向节点2发送消息。
//应该失败。
	_, err = stub.Step(testStepReq)
	assert.Contains(t, err.Error(), "rpc error: code = Unavailable")
//恢复节点2
	node2.resurrect()
//从节点1向节点2发送消息。
//最终会成功的
	assertEventuallyConnect(t, stub, testStepReq)
}

func TestRenewCertificates(t *testing.T) {
	t.Parallel()
//场景：节点1和节点2连接，
//两个节点的证书都更新了
//同时。
//他们应该互相联系。
//重新配置之后。

	node1 := newTestNode(t)
	defer node1.stop()

	node2 := newTestNode(t)
	defer node2.stop()

	node1.handler.On("OnStep", testChannel, node2.nodeInfo.ID, mock.Anything).Return(testStepRes, nil)
	node2.handler.On("OnStep", testChannel, node1.nodeInfo.ID, mock.Anything).Return(testStepRes, nil)

	config := []cluster.RemoteNode{node1.nodeInfo, node2.nodeInfo}
	node1.c.Configure(testChannel, config)
	node2.c.Configure(testChannel, config)

	assertBiDiCommunication(t, node1, node2, testStepReq)

//现在，更新两个节点的证书
	node1.renewCertificates()
	node2.renewCertificates()

//重新配置它们
	config = []cluster.RemoteNode{node1.nodeInfo, node2.nodeInfo}
	node1.c.Configure(testChannel, config)
	node2.c.Configure(testChannel, config)

//W.L.O.G，尝试将消息从node1发送到node2
//它应该会失败，因为node2的服务器证书现在已经更改，
//所以它关闭了到远程节点的连接
	remote, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.NoError(t, err)
	assert.NotNil(t, remote)
	_, err = remote.Step(&orderer.StepRequest{})
	assert.Contains(t, err.Error(), "rpc error")

//在两个节点上重新启动GRPC服务，以加载新的TLS证书
	node1.srv.Stop()
	node1.resurrect()
	node2.srv.Stop()
	node2.resurrect()

//最后，检查节点是否可以再次通信
	assertBiDiCommunication(t, node1, node2, testStepReq)
}

func TestMembershipReconfiguration(t *testing.T) {
	t.Parallel()
//场景：节点1和节点2启动
//节点2配置为了解节点1，
//无需节点1了解节点2。
//他们之间的交流只能起作用
//将节点1配置为了解节点2之后。

	node1 := newTestNode(t)
	defer node1.stop()

	node2 := newTestNode(t)
	defer node2.stop()

	node1.c.Configure(testChannel, []cluster.RemoteNode{})
	node2.c.Configure(testChannel, []cluster.RemoteNode{node1.nodeInfo})

//节点1无法连接到节点2，因为它还不知道其TLS证书
	_, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.EqualError(t, err, fmt.Sprintf("node %d doesn't exist in channel test's membership", node2.nodeInfo.ID))
//节点2可以连接到节点1，但它不能发送消息，因为节点1还不知道节点2。

	gt := gomega.NewGomegaWithT(t)
	gt.Eventually(func() (bool, error) {
		_, err := node2.c.Remote(testChannel, node1.nodeInfo.ID)
		return true, err
	}, time.Minute).Should(gomega.BeTrue())

	stub, err := node2.c.Remote(testChannel, node1.nodeInfo.ID)
	_, err = stub.Step(testStepReq)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = certificate extracted from TLS connection isn't authorized")

//接下来，配置节点1以了解节点2
	node1.handler.On("OnStep", testChannel, node2.nodeInfo.ID, mock.Anything).Return(testStepRes, nil)
	node2.handler.On("OnStep", testChannel, node1.nodeInfo.ID, mock.Anything).Return(testStepRes, nil)
	node1.c.Configure(testChannel, []cluster.RemoteNode{node2.nodeInfo})

//检查两个节点之间的通信是否正常工作
	assertBiDiCommunication(t, node1, node2, testStepReq)
	assertBiDiCommunication(t, node2, node1, testStepReq)

//重新配置节点2以忽略节点1
	node2.c.Configure(testChannel, []cluster.RemoteNode{})
//节点1仍然可以连接到节点2
	stub, err = node1.c.Remote(testChannel, node2.nodeInfo.ID)
//但无法发送消息，因为节点2现在未授权节点1
	_, err = stub.Step(testStepReq)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = certificate extracted from TLS connection isn't authorized")
}

func TestShutdown(t *testing.T) {
	t.Parallel()
//场景：节点1关闭，因此无法
//向任何人发送消息，也不能重新配置

	node1 := newTestNode(t)
	defer node1.stop()

	node1.c.Shutdown()

//无法成功获取远程上下文，因为以前调用了Shutdown
	_, err := node1.c.Remote(testChannel, node1.nodeInfo.ID)
	assert.EqualError(t, err, "communication has been shut down")

	node2 := newTestNode(t)
	defer node2.stop()

	node2.c.Configure(testChannel, []cluster.RemoteNode{node1.nodeInfo})
//节点配置未发生
	node1.c.Configure(testChannel, []cluster.RemoteNode{node2.nodeInfo})
	gt := gomega.NewGomegaWithT(t)
	gt.Eventually(func() (bool, error) {
		_, err := node2.c.Remote(testChannel, node1.nodeInfo.ID)
		return true, err
	}, time.Minute).Should(gomega.BeTrue())

	stub, err := node2.c.Remote(testChannel, node1.nodeInfo.ID)
//因此，发送消息失败，因为节点1拒绝了配置更改
	_, err = stub.Step(testStepReq)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = channel test doesn't exist")
}

func TestMultiChannelConfig(t *testing.T) {
	t.Parallel()
//场景：节点1只知道通道“foo”中的节点2
//并且只知道通道“bar”中的节点3。
//接收到的消息将根据其相应的通道进行路由。
//当节点2向节点1发送信道“bar”的消息时，该消息被拒绝。
//同样的事情也适用于向通道“foo”中的节点1发送消息的节点3。

	node1 := newTestNode(t)
	defer node1.stop()

	node2 := newTestNode(t)
	defer node2.stop()

	node3 := newTestNode(t)
	defer node3.stop()

	node1.c.Configure("foo", []cluster.RemoteNode{node2.nodeInfo})
	node1.c.Configure("bar", []cluster.RemoteNode{node3.nodeInfo})
	node2.c.Configure("foo", []cluster.RemoteNode{node1.nodeInfo})
	node3.c.Configure("bar", []cluster.RemoteNode{node1.nodeInfo})

	t.Run("Correct channel", func(t *testing.T) {
		node1.handler.On("OnStep", "foo", node2.nodeInfo.ID, mock.Anything).Return(fooRes, nil)
		node1.handler.On("OnStep", "bar", node3.nodeInfo.ID, mock.Anything).Return(barRes, nil)

		node2toNode1, err := node2.c.Remote("foo", node1.nodeInfo.ID)
		assert.NoError(t, err)
		node3toNode1, err := node3.c.Remote("bar", node1.nodeInfo.ID)
		assert.NoError(t, err)

		res, err := node2toNode1.Step(fooReq)
		assert.NoError(t, err)
		assert.Equal(t, string(res.Payload), fooReq.Channel)

		res, err = node3toNode1.Step(barReq)
		assert.NoError(t, err)
		assert.Equal(t, string(res.Payload), barReq.Channel)
	})

	t.Run("Incorrect channel", func(t *testing.T) {
		node2toNode1, err := node2.c.Remote("foo", node1.nodeInfo.ID)
		assert.NoError(t, err)
		node3toNode1, err := node3.c.Remote("bar", node1.nodeInfo.ID)
		assert.NoError(t, err)

		_, err = node2toNode1.Step(barReq)
		assert.EqualError(t, err, "rpc error: code = Unknown desc = certificate extracted from TLS connection isn't authorized")

		_, err = node3toNode1.Step(fooReq)
		assert.EqualError(t, err, "rpc error: code = Unknown desc = certificate extracted from TLS connection isn't authorized")
	})
}

func TestConnectionFailure(t *testing.T) {
	t.Parallel()
//场景：节点1无法连接到节点2。

	node1 := newTestNode(t)
	defer node1.stop()

	node2 := newTestNode(t)
	defer node2.stop()

	dialer := &mocks.SecureDialer{}
	dialer.On("Dial", mock.Anything, mock.Anything).Return(nil, errors.New("oops"))
	node1.c.Connections = cluster.NewConnectionStore(dialer)
	node1.c.Configure(testChannel, []cluster.RemoteNode{node2.nodeInfo})

	_, err := node1.c.Remote(testChannel, node2.nodeInfo.ID)
	assert.EqualError(t, err, "oops")
}

func assertBiDiCommunication(t *testing.T, node1, node2 *clusterNode, msgToSend *orderer.StepRequest) {
	for _, tst := range []struct {
		label    string
		sender   *clusterNode
		receiver *clusterNode
		target   uint64
	}{
		{label: "1->2", sender: node1, target: node2.nodeInfo.ID, receiver: node2},
		{label: "2->1", sender: node2, target: node1.nodeInfo.ID, receiver: node1},
	} {
		t.Run(tst.label, func(t *testing.T) {
			stub, err := tst.sender.c.Remote(testChannel, tst.target)
			assert.NoError(t, err)
			assertEventuallyConnect(t, stub, msgToSend)
			msg, err := stub.Step(msgToSend)
			assert.NoError(t, err)
			assert.Equal(t, msg.Payload, msgToSend.Payload)

			expectedRes := &orderer.SubmitResponse{}
			tst.receiver.handler.On("OnSubmit", testChannel, tst.sender.nodeInfo.ID, mock.Anything).Return(expectedRes, nil).Once()
			stub, err = tst.sender.c.Remote(testChannel, tst.target)
			assert.NoError(t, err)
			stream, err := stub.SubmitStream()
			assert.NoError(t, err)

			err = stream.Send(testSubReq)
			assert.NoError(t, err)

			res, err := stream.Recv()
			assert.NoError(t, err)

			assert.Equal(t, expectedRes, res)
		})
	}
}

func assertEventuallyConnect(t *testing.T, rpc *cluster.RemoteContext, req *orderer.StepRequest) {
	gt := gomega.NewGomegaWithT(t)
	gt.Eventually(func() (bool, error) {
		res, err := rpc.Step(req)
		if err != nil {
			return false, err
		}
		return bytes.Equal(res.Payload, req.Payload), nil
	}, timeout).Should(gomega.BeTrue())
}
