
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


package comm_test

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/comm"
	grpc_testdata "github.com/hyperledger/fabric/core/comm/testdata/grpc"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func TestExtractCertificateHashFromContext(t *testing.T) {
	t.Parallel()
	assert.Nil(t, comm.ExtractCertificateHashFromContext(context.Background()))

	p := &peer.Peer{}
	ctx := peer.NewContext(context.Background(), p)
	assert.Nil(t, comm.ExtractCertificateHashFromContext(ctx))

	p.AuthInfo = &nonTLSConnection{}
	ctx = peer.NewContext(context.Background(), p)
	assert.Nil(t, comm.ExtractCertificateHashFromContext(ctx))

	p.AuthInfo = credentials.TLSInfo{}
	ctx = peer.NewContext(context.Background(), p)
	assert.Nil(t, comm.ExtractCertificateHashFromContext(ctx))

	p.AuthInfo = credentials.TLSInfo{
		State: tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				{Raw: []byte{1, 2, 3}},
			},
		},
	}
	ctx = peer.NewContext(context.Background(), p)
	h := sha256.New()
	h.Write([]byte{1, 2, 3})
	assert.Equal(t, h.Sum(nil), comm.ExtractCertificateHashFromContext(ctx))
}

type nonTLSConnection struct {
}

func (*nonTLSConnection) AuthType() string {
	return ""
}

func TestBindingInspectorBadInit(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		comm.NewBindingInspector(false, nil)
	})
}

func TestNoopBindingInspector(t *testing.T) {
	t.Parallel()
	extract := func(msg proto.Message) []byte {
		return nil
	}
	assert.Nil(t, comm.NewBindingInspector(false, extract)(context.Background(), &common.Envelope{}))
	err := comm.NewBindingInspector(false, extract)(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, "message is nil", err.Error())
}

func TestBindingInspector(t *testing.T) {
	t.Parallel()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener for test server: %v", err)
	}

	extract := func(msg proto.Message) []byte {
		env, isEnvelope := msg.(*common.Envelope)
		if !isEnvelope || env == nil {
			return nil
		}
		ch, err := utils.ChannelHeader(env)
		if err != nil {
			return nil
		}
		return ch.TlsCertHash
	}
	srv := newInspectingServer(lis, comm.NewBindingInspector(true, extract))
	go srv.Start()
	defer srv.Stop()
	time.Sleep(time.Second)

//场景一：发送的标题无效
	err = srv.newInspection(t).inspectBinding(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client didn't include its TLS cert hash")

//场景二：通道头无效
	ch, _ := proto.Marshal(utils.MakeChannelHeader(common.HeaderType_CONFIG, 0, "test", 0))
//损坏的通道头
	ch = append(ch, 0)
	err = srv.newInspection(t).inspectBinding(envelopeWithChannelHeader(ch))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client didn't include its TLS cert hash")

//场景三：信封中没有TLS证书哈希
	chanHdr := utils.MakeChannelHeader(common.HeaderType_CONFIG, 0, "test", 0)
	ch, _ = proto.Marshal(chanHdr)
	err = srv.newInspection(t).inspectBinding(envelopeWithChannelHeader(ch))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client didn't include its TLS cert hash")

//场景四：客户端根据需要发送其TLS证书哈希，但不使用相互TLS
	cert, _ := tls.X509KeyPair([]byte(selfSignedCertPEM), []byte(selfSignedKeyPEM))
	h := sha256.New()
	h.Write([]byte(cert.Certificate[0]))
	chanHdr.TlsCertHash = h.Sum(nil)
	ch, _ = proto.Marshal(chanHdr)
	err = srv.newInspection(t).inspectBinding(envelopeWithChannelHeader(ch))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client didn't send a TLS certificate")

//方案五：客户端使用相互的TLS，但发送错误的TLS证书哈希
	chanHdr.TlsCertHash = []byte{1, 2, 3}
	chHdrWithWrongTLSCertHash, _ := proto.Marshal(chanHdr)
	err = srv.newInspection(t).withMutualTLS().inspectBinding(envelopeWithChannelHeader(chHdrWithWrongTLSCertHash))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claimed TLS cert hash is [1 2 3] but actual TLS cert hash is")

//场景六：客户端使用相互的TLS，并发送正确的TLS证书哈希
	err = srv.newInspection(t).withMutualTLS().inspectBinding(envelopeWithChannelHeader(ch))
	assert.NoError(t, err)
}

type inspectingServer struct {
	addr string
	*comm.GRPCServer
	lastContext atomic.Value
	inspector   comm.BindingInspector
}

func (is *inspectingServer) EmptyCall(ctx context.Context, _ *grpc_testdata.Empty) (*grpc_testdata.Empty, error) {
	is.lastContext.Store(ctx)
	return &grpc_testdata.Empty{}, nil
}

func (is *inspectingServer) inspect(envelope *common.Envelope) error {
	return is.inspector(is.lastContext.Load().(context.Context), envelope)
}

func newInspectingServer(listener net.Listener, inspector comm.BindingInspector) *inspectingServer {
	srv, err := comm.NewGRPCServerFromListener(listener, comm.ServerConfig{
		ConnectionTimeout: 250 * time.Millisecond,
		SecOpts: &comm.SecureOptions{
			UseTLS:      true,
			Certificate: []byte(selfSignedCertPEM),
			Key:         []byte(selfSignedKeyPEM),
		}})
	if err != nil {
		panic(err)
	}
	is := &inspectingServer{
		addr:       listener.Addr().String(),
		GRPCServer: srv,
		inspector:  inspector,
	}
	grpc_testdata.RegisterTestServiceServer(srv.Server(), is)
	return is
}

type inspection struct {
	tlsConfig *tls.Config
	server    *inspectingServer
	creds     credentials.TransportCredentials
	t         *testing.T
}

func (is *inspectingServer) newInspection(t *testing.T) *inspection {
	tlsConfig := &tls.Config{
		RootCAs: x509.NewCertPool(),
	}
	tlsConfig.RootCAs.AppendCertsFromPEM([]byte(selfSignedCertPEM))
	return &inspection{
		server:    is,
		creds:     credentials.NewTLS(tlsConfig),
		t:         t,
		tlsConfig: tlsConfig,
	}
}

func (ins *inspection) withMutualTLS() *inspection {
	cert, err := tls.X509KeyPair([]byte(selfSignedCertPEM), []byte(selfSignedKeyPEM))
	assert.NoError(ins.t, err)
	ins.tlsConfig.Certificates = []tls.Certificate{cert}
	ins.creds = credentials.NewTLS(ins.tlsConfig)
	return ins
}

func (ins *inspection) inspectBinding(envelope *common.Envelope) error {
	ctx := context.Background()
	ctx, c := context.WithTimeout(ctx, time.Second*3)
	defer c()
	conn, err := grpc.DialContext(ctx, ins.server.addr, grpc.WithTransportCredentials(ins.creds), grpc.WithBlock())
	defer conn.Close()
	assert.NoError(ins.t, err)
	_, err = grpc_testdata.NewTestServiceClient(conn).EmptyCall(context.Background(), &grpc_testdata.Empty{})
	return ins.server.inspect(envelope)
}

func envelopeWithChannelHeader(ch []byte) *common.Envelope {
	pl := &common.Payload{
		Header: &common.Header{
			ChannelHeader: ch,
		},
	}
	payload, _ := proto.Marshal(pl)
	return &common.Envelope{
		Payload: payload,
	}
}
