
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


package deliverclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestTLSBinding(t *testing.T) {
	defer ensureNoGoroutineLeak(t)()

	requester := blocksRequester{
		tls:     true,
		chainID: "testchainid",
	}

//创建原子广播服务器
	serverCert, serverKey, caCert := loadCertificates(t)
	serverTLScert, err := tls.X509KeyPair(serverCert, serverKey)
	assert.NoError(t, err)
	comm.GetCredentialSupport().SetClientCertificate(serverTLScert)
	s, err := comm.NewGRPCServer("localhost:9435", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			RequireClientCert: true,
			Key:               serverKey,
			Certificate:       serverCert,
			ClientRootCAs:     [][]byte{caCert},
			UseTLS:            true,
		},
	})
	assert.NoError(t, err)

	orderer.RegisterAtomicBroadcastServer(s.Server(), &mockOrderer{t: t})
	go s.Start()
	defer s.Stop()
	time.Sleep(time.Second * 3)

//创建传递客户端并尝试请求块100
//从订购服务
	client := createClient(t, serverTLScert, caCert)
	requester.client = client

//测试来自Committer和Seekoldest的Seeklatest

//见提交人签名
	requester.seekLatestFromCommitter(100)
	resp, err := requester.client.Recv()
	assert.NoError(t, err)
	assert.Equal(t, 100, int(resp.GetBlock().Header.Number))
	client.conn.Close()

//寻找最老的人
	client = createClient(t, serverTLScert, caCert)
	requester.client = client
	requester.seekOldest()
	resp, err = requester.client.Recv()
	assert.NoError(t, err)
	assert.Equal(t, 100, int(resp.GetBlock().Header.Number))
	client.conn.Close()
}

func loadCertificates(t *testing.T) (cert []byte, key []byte, caCert []byte) {
	var err error
	caCertFile := filepath.Join("testdata", "ca.pem")
	certFile := filepath.Join("testdata", "cert.pem")
	keyFile := filepath.Join("testdata", "key.pem")

	cert, err = ioutil.ReadFile(certFile)
	assert.NoError(t, err)
	key, err = ioutil.ReadFile(keyFile)
	assert.NoError(t, err)
	caCert, err = ioutil.ReadFile(caCertFile)
	assert.NoError(t, err)
	return
}

type mockClient struct {
	blocksprovider.BlocksDeliverer
	conn *grpc.ClientConn
}

func createClient(t *testing.T, tlsCert tls.Certificate, caCert []byte) *mockClient {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      x509.NewCertPool(),
	}
	tlsConfig.RootCAs.AppendCertsFromPEM(caCert)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	dialOpts := []grpc.DialOption{grpc.WithBlock(), grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.DialContext(ctx, "localhost:9435", dialOpts...)
	assert.NoError(t, err)
	cl := orderer.NewAtomicBroadcastClient(conn)

	stream, err := cl.Deliver(context.Background())
	assert.NoError(t, err)
	return &mockClient{
		conn:            conn,
		BlocksDeliverer: stream,
	}
}

type mockOrderer struct {
	t *testing.T
}

func (*mockOrderer) Broadcast(orderer.AtomicBroadcast_BroadcastServer) error {
	panic("not implemented")
}

func (o *mockOrderer) Deliver(stream orderer.AtomicBroadcast_DeliverServer) error {
	env, _ := stream.Recv()
	inspectTLSBinding := comm.NewBindingInspector(true, func(msg proto.Message) []byte {
		env, isEnvelope := msg.(*common.Envelope)
		if !isEnvelope || env == nil {
			assert.Fail(o.t, "not an envelope")
		}
		ch, err := utils.ChannelHeader(env)
		assert.NoError(o.t, err)
		return ch.TlsCertHash
	})
	err := inspectTLSBinding(stream.Context(), env)
	assert.NoError(o.t, err, "orderer rejected TLS binding")

	stream.Send(&orderer.DeliverResponse{
		Type: &orderer.DeliverResponse_Block{
			Block: &common.Block{
				Header: &common.BlockHeader{
					Number: 100,
				},
			},
		},
	})
	return nil
}
