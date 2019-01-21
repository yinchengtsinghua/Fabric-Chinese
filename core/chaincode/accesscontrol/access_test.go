
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


package accesscontrol

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/flogging/floggingtest"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ccSrv struct {
	l              net.Listener
	grpcSrv        *grpc.Server
	t              *testing.T
	cert           []byte
	expectedCCname string
}

func (cs *ccSrv) Register(stream pb.ChaincodeSupport_RegisterServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}

//第一条消息是注册消息
	assert.Equal(cs.t, pb.ChaincodeMessage_REGISTER.String(), msg.Type.String())
//它的链码名是预期的
	chaincodeID := &pb.ChaincodeID{}
	err = proto.Unmarshal(msg.Payload, chaincodeID)
	if err != nil {
		return err
	}
	assert.Equal(cs.t, cs.expectedCCname, chaincodeID.Name)
//
	for {
		msg, _ = stream.Recv()
		if err != nil {
			return err
		}
		err = stream.Send(msg)
		if err != nil {
			return err
		}
	}
}

func (cs *ccSrv) stop() {
	cs.grpcSrv.Stop()
	cs.l.Close()
}

func createTLSService(t *testing.T, ca tlsgen.CA, host string) *grpc.Server {
	keyPair, err := ca.NewServerCertKeyPair(host)
	cert, err := tls.X509KeyPair(keyPair.Cert, keyPair.Key)
	assert.NoError(t, err)
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    x509.NewCertPool(),
	}
	tlsConf.ClientCAs.AppendCertsFromPEM(ca.CertBytes())
	return grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConf)))
}

func newCCServer(t *testing.T, port int, expectedCCname string, withTLS bool, ca tlsgen.CA) *ccSrv {
	var s *grpc.Server
	if withTLS {
		s = createTLSService(t, ca, "localhost")
	} else {
		s = grpc.NewServer()
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "", port))
	assert.NoError(t, err, "%v", err)
	return &ccSrv{
		t:              t,
		expectedCCname: expectedCCname,
		l:              l,
		grpcSrv:        s,
	}
}

type ccClient struct {
	conn   *grpc.ClientConn
	stream pb.ChaincodeSupport_RegisterClient
}

func newClient(t *testing.T, port int, cert *tls.Certificate, peerCACert []byte) (*ccClient, error) {
	tlsCfg := &tls.Config{
		RootCAs: x509.NewCertPool(),
	}

	tlsCfg.RootCAs.AppendCertsFromPEM(peerCACert)
	if cert != nil {
		tlsCfg.Certificates = []tls.Certificate{*cert}
	}
	tlsOpts := grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%d", port), tlsOpts, grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	chaincodeSupportClient := pb.NewChaincodeSupportClient(conn)
	stream, err := chaincodeSupportClient.Register(context.Background())
	assert.NoError(t, err)
	return &ccClient{
		conn:   conn,
		stream: stream,
	}, nil
}

func (c *ccClient) close() {
	c.conn.Close()
}

func (c *ccClient) sendMsg(msg *pb.ChaincodeMessage) {
	c.stream.Send(msg)
}

func (c *ccClient) recv() *pb.ChaincodeMessage {
	msgs := make(chan *pb.ChaincodeMessage, 1)
	go func() {
		msg, _ := c.stream.Recv()
		if msg != nil {
			msgs <- msg
		}
	}()
	select {
	case <-time.After(time.Second):
		return nil
	case msg := <-msgs:
		return msg
	}
}

func TestAccessControl(t *testing.T) {
	backupTTL := ttl
	defer func() {
		ttl = backupTTL
	}()
	ttl = time.Second * 3

	oldLogger := logger
	l, recorder := floggingtest.NewTestLogger(t, floggingtest.AtLevel(zapcore.InfoLevel))
	logger = l
	defer func() { logger = oldLogger }()

	chaincodeID := &pb.ChaincodeID{Name: "example02"}
	payload, err := proto.Marshal(chaincodeID)
	registerMsg := &pb.ChaincodeMessage{
		Type:    pb.ChaincodeMessage_REGISTER,
		Payload: payload,
	}
	putStateMsg := &pb.ChaincodeMessage{
		Type: pb.ChaincodeMessage_PUT_STATE,
	}

	ca, _ := tlsgen.NewCA()
	srv := newCCServer(t, 7052, "example02", true, ca)
	auth := NewAuthenticator(ca)
	pb.RegisterChaincodeSupportServer(srv.grpcSrv, auth.Wrap(srv))
	go srv.grpcSrv.Serve(srv.l)
	defer srv.stop()

//创建没有TLS证书的攻击者
	_, err = newClient(t, 7052, nil, ca.CertBytes())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")

//使用自己的TLS证书创建攻击者
	maliciousCA, _ := tlsgen.NewCA()
	keyPair, err := maliciousCA.NewClientCertKeyPair()
	cert, err := tls.X509KeyPair(keyPair.Cert, keyPair.Key)
	assert.NoError(t, err)
	_, err = newClient(t, 7052, &cert, ca.CertBytes())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")

//为尝试模拟example02的example01创建链代码
	kp, err := auth.Generate("example01")
	assert.NoError(t, err)
	keyBytes, err := base64.StdEncoding.DecodeString(kp.Key)
	assert.NoError(t, err)
	certBytes, err := base64.StdEncoding.DecodeString(kp.Cert)
	assert.NoError(t, err)
	cert, err = tls.X509KeyPair(certBytes, keyBytes)
	assert.NoError(t, err)
	mismatchedShim, err := newClient(t, 7052, &cert, ca.CertBytes())
	assert.NoError(t, err)
	defer mismatchedShim.close()
	mismatchedShim.sendMsg(registerMsg)
	mismatchedShim.sendMsg(putStateMsg)
//不匹配的链码无法返回任何内容
	assert.Nil(t, mismatchedShim.recv())
	assertLogContains(t, recorder, "with given certificate hash", "belongs to a different chaincode")

//创建真正的链码，它的证书由我们生成，应该通过安全检查
	kp, err = auth.Generate("example02")
	assert.NoError(t, err)
	keyBytes, err = base64.StdEncoding.DecodeString(kp.Key)
	assert.NoError(t, err)
	certBytes, err = base64.StdEncoding.DecodeString(kp.Cert)
	assert.NoError(t, err)
	cert, err = tls.X509KeyPair(certBytes, keyBytes)
	assert.NoError(t, err)
	realCC, err := newClient(t, 7052, &cert, ca.CertBytes())
	assert.NoError(t, err)
	defer realCC.close()
	realCC.sendMsg(registerMsg)
	realCC.sendMsg(putStateMsg)
	echoMsg := realCC.recv()
//真正的链码应该回显它的消息
	assert.NotNil(t, echoMsg)
	assert.Equal(t, pb.ChaincodeMessage_PUT_STATE, echoMsg.Type)
//日志不应抱怨任何事情
	assert.Empty(t, recorder.Messages())

//创建其证书由我们生成的真正链代码
//但它发送的第一条消息不是注册消息。
//发送的第二条消息是一条注册消息，但它“太迟了”
//流已被拒绝。
	kp, err = auth.Generate("example02")
	assert.NoError(t, err)
	keyBytes, err = base64.StdEncoding.DecodeString(kp.Key)
	assert.NoError(t, err)
	certBytes, err = base64.StdEncoding.DecodeString(kp.Cert)
	assert.NoError(t, err)
	cert, err = tls.X509KeyPair(certBytes, keyBytes)
	assert.NoError(t, err)
	confusedCC, err := newClient(t, 7052, &cert, ca.CertBytes())
	assert.NoError(t, err)
	defer confusedCC.close()
	confusedCC.sendMsg(putStateMsg)
	confusedCC.sendMsg(registerMsg)
	confusedCC.sendMsg(putStateMsg)
	assert.Nil(t, confusedCC.recv())
	assertLogContains(t, recorder, "expected a ChaincodeMessage_REGISTER message")

//创建一个真正的链代码，它的证书是由我们生成的
//但它发送了一个格式错误的第一条消息
	kp, err = auth.Generate("example02")
	assert.NoError(t, err)
	keyBytes, err = base64.StdEncoding.DecodeString(kp.Key)
	assert.NoError(t, err)
	certBytes, err = base64.StdEncoding.DecodeString(kp.Cert)
	assert.NoError(t, err)
	cert, err = tls.X509KeyPair(certBytes, keyBytes)
	assert.NoError(t, err)
	malformedMessageCC, err := newClient(t, 7052, &cert, ca.CertBytes())
	assert.NoError(t, err)
	defer malformedMessageCC.close()
//保存旧负载
	originalPayload := registerMsg.Payload
	registerMsg.Payload = append(registerMsg.Payload, 0)
	malformedMessageCC.sendMsg(registerMsg)
	malformedMessageCC.sendMsg(putStateMsg)
	assert.Nil(t, malformedMessageCC.recv())
	assertLogContains(t, recorder, "Failed unmarshaling message")
//恢复旧的有效载荷
	registerMsg.Payload = originalPayload

//创建一个真正的链代码，它的证书是由我们生成的
//但要在太长时间后重新连接。
//
//而且CC已经被泄露了。我们不希望它能够
//重新连接我们。
	kp, err = auth.Generate("example02")
	assert.NoError(t, err)
	keyBytes, err = base64.StdEncoding.DecodeString(kp.Key)
	assert.NoError(t, err)
	certBytes, err = base64.StdEncoding.DecodeString(kp.Cert)
	assert.NoError(t, err)
	cert, err = tls.X509KeyPair(certBytes, keyBytes)
	assert.NoError(t, err)
	lateCC, err := newClient(t, 7052, &cert, ca.CertBytes())
	assert.NoError(t, err)
	defer lateCC.close()
	time.Sleep(ttl + time.Second*2)
	lateCC.sendMsg(registerMsg)
	lateCC.sendMsg(putStateMsg)
	echoMsg = lateCC.recv()
	assert.Nil(t, echoMsg)
	assertLogContains(t, recorder, "with given certificate hash", "not found in registry")
}

func assertLogContains(t *testing.T, r *floggingtest.Recorder, ss ...string) {
	defer r.Reset()
	for _, s := range ss {
		assert.NotEmpty(t, r.MessagesContaining(s))
	}
}
