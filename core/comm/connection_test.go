
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


package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	testpb "github.com/hyperledger/fabric/core/comm/testdata/grpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	numOrgs      = 2
	numChildOrgs = 2
)

//证书文件名字符串
var (
	orgCACert   = filepath.Join("testdata", "certs", "Org%d-cert.pem")
	childCACert = filepath.Join("testdata", "certs", "Org%d-child%d-cert.pem")
)

var badPEM = `-----BEGIN CERTIFICATE-----
MIICRDCCAemgAwIBAgIJALwW//DZ2ZBMAOGCCQGSM49BAMCM4XCZZAJBGNVBAYT公司
AlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNpc2Nv
MRgwFgYDVQQKDA9MaW51eEZvdW5kYXRpb24xFDASBgNVBAsMC0h5cGVybGVkZ2Vy
MRIwEAYDVQQDDAlsb2NhbGhvc3QwHhcNMTYxMjA0MjIzMDE4WhcNMjYxMjAyMjIz
MDE4WjB+MQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UE
BwwNU2FuIEZyYW5jaXNjbzEYMBYGA1UECgwPTGludXhGb3VuZGF0aW9uMRQwEgYD
VQQLDAtIeXBlcmxlZGdlcjESMBAGA1UEAwwJbG9jYWxob3N0MFkwEwYHKoZIzj0C
-----END CERTIFICATE-----
`

func TestClientConnections(t *testing.T) {
	t.Parallel()

//使用ORG1测试加密材料
	fileBase := "Org1"
	certPEMBlock, _ := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-server1-cert.pem"))
	keyPEMBlock, _ := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-server1-key.pem"))
	caPEMBlock, _ := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-cert.pem"))
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caPEMBlock)

	var tests = []struct {
		name       string
		sc         ServerConfig
		creds      credentials.TransportCredentials
		clientPort int
		fail       bool
	}{
		{
			name: "ValidConnection",
			sc: ServerConfig{
				SecOpts: &SecureOptions{
					UseTLS: false}},
		},
		{
			name: "InvalidConnection",
			sc: ServerConfig{
				SecOpts: &SecureOptions{
					UseTLS: false}},
			clientPort: 20040,
			fail:       true,
		},
		{
			name: "ValidConnectionTLS",
			sc: ServerConfig{
				SecOpts: &SecureOptions{
					UseTLS:      true,
					Certificate: certPEMBlock,
					Key:         keyPEMBlock}},
			creds: credentials.NewClientTLSFromCert(certPool, ""),
		},
		{
			name: "InvalidConnectionTLS",
			sc: ServerConfig{
				SecOpts: &SecureOptions{
					UseTLS:      true,
					Certificate: certPEMBlock,
					Key:         keyPEMBlock}},
			creds: credentials.NewClientTLSFromCert(nil, ""),
			fail:  true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("Running test %s ...", test.name)
			lis, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("failed to create listener for test server: %v", err)
			}
			clientAddress := lis.Addr().String()
			if test.clientPort > 0 {
				clientAddress = fmt.Sprintf("127.0.0.1:%d", test.clientPort)
			}
			srv, err := NewGRPCServerFromListener(lis, test.sc)
//检查错误
			if err != nil {
				t.Fatalf("Error [%s] creating test server for address [%s]",
					err, lis.Addr().String())
			}
//启动服务器
			go srv.Start()
			defer srv.Stop()
			testConn, err := NewClientConnectionWithAddress(clientAddress,
				true, test.sc.SecOpts.UseTLS, test.creds, nil)
			if test.fail {
				assert.Error(t, err)
			} else {
				testConn.Close()
				assert.NoError(t, err)
			}
		})
	}
}

//实用程序函数从testdata/certs加载测试根证书
func loadRootCAs() [][]byte {
	rootCAs := [][]byte{}
	for i := 1; i <= numOrgs; i++ {
		root, err := ioutil.ReadFile(fmt.Sprintf(orgCACert, i))
		if err != nil {
			return [][]byte{}
		}
		rootCAs = append(rootCAs, root)
		for j := 1; j <= numChildOrgs; j++ {
			root, err := ioutil.ReadFile(fmt.Sprintf(childCACert, i, j))
			if err != nil {
				return [][]byte{}
			}
			rootCAs = append(rootCAs, root)
		}
	}
	return rootCAs
}

func TestCASupport(t *testing.T) {
	t.Parallel()
	rootCAs := loadRootCAs()
	t.Logf("loaded %d root certificates", len(rootCAs))
	if len(rootCAs) != 6 {
		t.Fatalf("failed to load root certificates")
	}

	cas := &CASupport{
		AppRootCAsByChain:     make(map[string][][]byte),
		OrdererRootCAsByChain: make(map[string][][]byte),
	}
	cas.AppRootCAsByChain["channel1"] = [][]byte{rootCAs[0]}
	cas.AppRootCAsByChain["channel2"] = [][]byte{rootCAs[1]}
	cas.AppRootCAsByChain["channel3"] = [][]byte{rootCAs[2]}
	cas.OrdererRootCAsByChain["channel1"] = [][]byte{rootCAs[3]}
	cas.OrdererRootCAsByChain["channel2"] = [][]byte{rootCAs[4]}
	cas.ServerRootCAs = [][]byte{rootCAs[5]}
	cas.ClientRootCAs = [][]byte{rootCAs[5]}

	appServerRoots, ordererServerRoots := cas.GetServerRootCAs()
	t.Logf("%d appServerRoots | %d ordererServerRoots", len(appServerRoots),
		len(ordererServerRoots))
	assert.Equal(t, 4, len(appServerRoots), "Expected 4 app server root CAs")
	assert.Equal(t, 2, len(ordererServerRoots), "Expected 2 orderer server root CAs")

	appClientRoots, ordererClientRoots := cas.GetClientRootCAs()
	t.Logf("%d appClientRoots | %d ordererClientRoots", len(appClientRoots),
		len(ordererClientRoots))
	assert.Equal(t, 4, len(appClientRoots), "Expected 4 app client root CAs")
	assert.Equal(t, 2, len(ordererClientRoots), "Expected 4 orderer client root CAs")
}

func TestCredentialSupport(t *testing.T) {
	t.Parallel()
	rootCAs := loadRootCAs()
	t.Logf("loaded %d root certificates", len(rootCAs))
	if len(rootCAs) != 6 {
		t.Fatalf("failed to load root certificates")
	}

	cs := &CredentialSupport{
		CASupport: &CASupport{
			AppRootCAsByChain:     make(map[string][][]byte),
			OrdererRootCAsByChain: make(map[string][][]byte),
		},
	}
	cert := tls.Certificate{Certificate: [][]byte{}}
	cs.SetClientCertificate(cert)
	assert.Equal(t, cert, cs.clientCert)
	assert.Equal(t, cert, cs.GetClientCertificate())

	cs.AppRootCAsByChain["channel1"] = [][]byte{rootCAs[0]}
	cs.AppRootCAsByChain["channel2"] = [][]byte{rootCAs[1]}
	cs.AppRootCAsByChain["channel3"] = [][]byte{rootCAs[2]}
	cs.OrdererRootCAsByChain["channel1"] = [][]byte{rootCAs[3]}
	cs.OrdererRootCAsByChain["channel2"] = [][]byte{rootCAs[4]}
	cs.ServerRootCAs = [][]byte{rootCAs[5]}
	cs.ClientRootCAs = [][]byte{rootCAs[5]}

	appServerRoots, ordererServerRoots := cs.GetServerRootCAs()
	t.Logf("%d appServerRoots | %d ordererServerRoots", len(appServerRoots),
		len(ordererServerRoots))
	assert.Equal(t, 4, len(appServerRoots), "Expected 4 app server root CAs")
	assert.Equal(t, 2, len(ordererServerRoots), "Expected 2 orderer server root CAs")

	appClientRoots, ordererClientRoots := cs.GetClientRootCAs()
	t.Logf("%d appClientRoots | %d ordererClientRoots", len(appClientRoots),
		len(ordererClientRoots))
	assert.Equal(t, 4, len(appClientRoots), "Expected 4 app client root CAs")
	assert.Equal(t, 2, len(ordererClientRoots), "Expected 4 orderer client root CAs")

	creds, _ := cs.GetDeliverServiceCredentials("channel1")
	assert.Equal(t, "1.2", creds.Info().SecurityVersion,
		"Expected Security version to be 1.2")
	creds = cs.GetPeerCredentials()
	assert.Equal(t, "1.2", creds.Info().SecurityVersion,
		"Expected Security version to be 1.2")

//附加一些坏证书并确保事情仍然有效
	cs.ServerRootCAs = append(cs.ServerRootCAs, []byte("badcert"))
	cs.ServerRootCAs = append(cs.ServerRootCAs, []byte(badPEM))
	creds, _ = cs.GetDeliverServiceCredentials("channel1")
	assert.Equal(t, "1.2", creds.Info().SecurityVersion,
		"Expected Security version to be 1.2")
	creds = cs.GetPeerCredentials()
	assert.Equal(t, "1.2", creds.Info().SecurityVersion,
		"Expected Security version to be 1.2")

//测试单
	singleton := GetCredentialSupport()
	clone := GetCredentialSupport()
	assert.Exactly(t, clone, singleton, "Expected GetCredentialSupport to be a singleton")
}

type srv struct {
	port    int
	address string
	*GRPCServer
	caCert   []byte
	serviced uint32
}

func (s *srv) assertServiced(t *testing.T) {
	assert.Equal(t, uint32(1), atomic.LoadUint32(&s.serviced))
	atomic.StoreUint32(&s.serviced, 0)
}

func (s *srv) EmptyCall(context.Context, *testpb.Empty) (*testpb.Empty, error) {
	atomic.StoreUint32(&s.serviced, 1)
	return &testpb.Empty{}, nil
}

func newServer(org string) *srv {
	certs := map[string][]byte{
		"ca.crt":     nil,
		"server.crt": nil,
		"server.key": nil,
	}
	for suffix := range certs {
		fName := filepath.Join("testdata", "impersonation", org, suffix)
		cert, err := ioutil.ReadFile(fName)
		if err != nil {
			panic(fmt.Errorf("Failed reading %s: %v", fName, err))
		}
		certs[suffix] = cert
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Errorf("Failed to create listener: %v", err))
	}
	gSrv, err := NewGRPCServerFromListener(l, ServerConfig{
		ConnectionTimeout: 250 * time.Millisecond,
		SecOpts: &SecureOptions{
			Certificate: certs["server.crt"],
			Key:         certs["server.key"],
			UseTLS:      true,
		},
	})
	if err != nil {
		panic(fmt.Errorf("Failed starting gRPC server: %v", err))
	}
	s := &srv{
		address:    l.Addr().String(),
		caCert:     certs["ca.crt"],
		GRPCServer: gSrv,
	}
	testpb.RegisterTestServiceServer(gSrv.Server(), s)
	go s.Start()
	return s
}

func TestImpersonation(t *testing.T) {
	t.Parallel()
//场景：我们有两个组织：Orga，Orgb
//他们每个人都有各自受人尊敬的渠道——A，B。
//测试将通过调用GetDeliverServiceCredentials来获取Credentials.TransportCredentials。
//每个组织都有自己的GRPC服务器（SRVA和SRVB）和一个TLS证书
//由根CA签名，SAN条目为“127.0.0.1”。
//我们测试以下断言：
//1）使用GetDeliverServiceCredentials（“A”）调用SRVA成功
//2）使用GetDeliverServiceCredentials（“B”）调用SRVB成功
//
//4）使用GetDeliverServiceCredentials（“B”）调用SRVA失败

	osA := newServer("orgA")
	defer osA.Stop()
	osB := newServer("orgB")
	defer osB.Stop()
	time.Sleep(time.Second)

	cs := &CredentialSupport{
		CASupport: &CASupport{
			AppRootCAsByChain:     make(map[string][][]byte),
			OrdererRootCAsByChain: make(map[string][][]byte),
		},
	}
	_, err := cs.GetDeliverServiceCredentials("C")
	assert.Error(t, err)

	cs.OrdererRootCAsByChain["A"] = [][]byte{osA.caCert}
	cs.OrdererRootCAsByChain["B"] = [][]byte{osB.caCert}

	testInvoke(t, "A", osA, cs, true)
	testInvoke(t, "B", osB, cs, true)
	testInvoke(t, "A", osB, cs, false)
	testInvoke(t, "B", osA, cs, false)

}

func testInvoke(
	t *testing.T,
	channelID string,
	s *srv,
	cs *CredentialSupport,
	shouldSucceed bool) {

	creds, err := cs.GetDeliverServiceCredentials(channelID)
	assert.NoError(t, err)

	endpoint := s.address
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(creds), grpc.WithBlock())
	if shouldSucceed {
		assert.NoError(t, err)
		defer conn.Close()
	} else {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
		return
	}
	client := testpb.NewTestServiceClient(conn)
	_, err = client.EmptyCall(context.Background(), &testpb.Empty{})
	assert.NoError(t, err)
	s.assertServiced(t)
}
