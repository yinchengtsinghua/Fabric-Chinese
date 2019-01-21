
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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/core/comm"
	testpb "github.com/hyperledger/fabric/core/comm/testdata/grpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

//嵌入式测试证书
//自签名证书将于2028年到期。
var selfSignedKeyPEM = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMLemLh3+uDzww1pvqP6Xj2Z0Kc6yqf3RxyfTBNwRuuyoAoGCCqGSM49
AwEHoUQDQgAEDB3l94vM7EqKr2L/vhqU5IsEub0rviqCAaWGiVAPp3orb/LJqFLS
yo/k60rhUiir6iD4S4pb5TEb2ouWylQI3A==
-----END EC PRIVATE KEY-----
`
var selfSignedCertPEM = `-----BEGIN CERTIFICATE-----
MIIBdDCCARqgAwIBAgIRAKCiW5r6W32jGUn+l9BORMAwCgYIKoZIzj0EAwIwEjEQ
MA4GA1UEChMHQWNtZSBDbzAeFw0xODA4MjExMDI1MzJaFw0yODA4MTgxMDI1MzJa
MBIxEDAOBgNVBAoTB0FjbWUgQ28wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQM
HeX3i8zsSoqvYv++GpTkiwS5vSu+KoIBpYaJUA+neitv8smoUtLKj+TrSuFSKKvq
IPhLilvlMRvai5bKVAjco1EwTzAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYI
KwYBBQUHAwEwDAYDVR0TAQH/BAIwADAaBgNVHREEEzARgglsb2NhbGhvc3SHBH8A
AAEwCgYIKoZIzj0EAwIDSAAwRQIgOaYc3pdGf2j0uXRyvdBJq2PlK9FkgvsUjXOT
bQ9fWRkCIQCr1FiRRzapgtrnttDn3O2fhLlbrw67kClzY8pIIN42Qw==
-----END CERTIFICATE-----
`

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

var pemNoCertificateHeader = `-----BEGIN NOCERT-----
MIICRDCCAemgAwIBAgIJALwW//DZ2ZBMAOGCCQGSM49BAMCM4XCZZAJBGNVBAYT公司
AlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNpc2Nv
MRgwFgYDVQQKDA9MaW51eEZvdW5kYXRpb24xFDASBgNVBAsMC0h5cGVybGVkZ2Vy
MRIwEAYDVQQDDAlsb2NhbGhvc3QwHhcNMTYxMjA0MjIzMDE4WhcNMjYxMjAyMjIz
MDE4WjB+MQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UE
BwwNU2FuIEZyYW5jaXNjbzEYMBYGA1UECgwPTGludXhGb3VuZGF0aW9uMRQwEgYD
VQQLDAtIeXBlcmxlZGdlcjESMBAGA1UEAwwJbG9jYWxob3N0MFkwEwYHKoZIzj0C
AQYIKoZIzj0DAQcDQgAEu2FEZVSr30Afey6dwcypeg5P+BuYx5JSYdG0/KJIBjWK
nzYo7FEmgMir7GbNh4pqA8KFrJZkPuxMgnEJBZTv+6NQME4wHQYDVR0OBBYEFAWO
4bfTEr2R6VYzQYrGk/2VWmtYMB8GA1UdIwQYMBaAFAWO4bfTEr2R6VYzQYrGk/2V
WmtYMAwGA1UdEwQFMAMBAf8wCgYIKoZIzj0EAwIDSQAwRgIhAIelqGdxPMHmQqRF
zA85vv7JhfMkvZYGPELC7I2K8V7ZAiEA9KcthV3HtDXKNDsA6ULT+qUkyoHRzCzr
A4QaL2VU6i4=
-----END NOCERT-----
`

var timeout = time.Second * 1
var testOrgs = []testOrg{}

func init() {
//加载测试组织的加密材料
	for i := 1; i <= numOrgs; i++ {
		testOrg, err := loadOrg(i)
		if err != nil {
			log.Fatalf("Failed to load test organizations due to error: %s", err.Error())
		}
		testOrgs = append(testOrgs, testOrg)
	}
}

//要在GRPCServer中注册的测试服务器
type emptyServiceServer struct{}

func (ess *emptyServiceServer) EmptyCall(context.Context, *testpb.Empty) (*testpb.Empty, error) {
	return new(testpb.Empty), nil
}

func (esss *emptyServiceServer) EmptyStream(stream testpb.EmptyService_EmptyStreamServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&testpb.Empty{}); err != nil {
			return err
		}

	}
}

//调用EmptyCall RPC
func invokeEmptyCall(address string, dialOptions []grpc.DialOption) (*testpb.Empty, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
//创建GRPC客户端连接
	clientConn, err := grpc.DialContext(ctx, address, dialOptions...)
	if err != nil {
		return nil, err
	}
	defer clientConn.Close()

//创建GRPC客户端
	client := testpb.NewEmptyServiceClient(clientConn)

//调用服务
	empty, err := client.EmptyCall(context.Background(), new(testpb.Empty))
	if err != nil {
		return nil, err
	}

	return empty, nil
}

//调用EmptyStream RPC
func invokeEmptyStream(address string, dialOptions []grpc.DialOption) (*testpb.Empty, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
//创建GRPC客户端连接
	clientConn, err := grpc.DialContext(ctx, address, dialOptions...)
	if err != nil {
		return nil, err
	}
	defer clientConn.Close()

	stream, err := testpb.NewEmptyServiceClient(clientConn).EmptyStream(ctx)
	if err != nil {
		return nil, err
	}

	var msg *testpb.Empty
	var streamErr error

	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				streamErr = err
				close(waitc)
				return
			}
			msg = in
		}
	}()

//testserverinterceptors添加了一个不调用目标的拦截器
//流处理程序并返回一个错误，因此send可以使用io.eof返回，因为
//服务器端已终止。是否出错
//取决于时间。
	err = stream.Send(&testpb.Empty{})
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("stream send failed: %s", err)
	}

	stream.CloseSend()
	<-waitc
	return msg, streamErr
}

const (
	numOrgs        = 2
	numChildOrgs   = 2
	numClientCerts = 2
	numServerCerts = 2
)

//证书文件名字符串
var (
	orgCAKey        = filepath.Join("testdata", "certs", "Org%d-key.pem")
	orgCACert       = filepath.Join("testdata", "certs", "Org%d-cert.pem")
	orgServerKey    = filepath.Join("testdata", "certs", "Org%d-server%d-key.pem")
	orgServerCert   = filepath.Join("testdata", "certs", "Org%d-server%d-cert.pem")
	orgClientKey    = filepath.Join("testdata", "certs", "Org%d-client%d-key.pem")
	orgClientCert   = filepath.Join("testdata", "certs", "Org%d-client%d-cert.pem")
	childCAKey      = filepath.Join("testdata", "certs", "Org%d-child%d-key.pem")
	childCACert     = filepath.Join("testdata", "certs", "Org%d-child%d-cert.pem")
	childServerKey  = filepath.Join("testdata", "certs", "Org%d-child%d-server%d-key.pem")
	childServerCert = filepath.Join("testdata", "certs", "Org%d-child%d-server%d-cert.pem")
	childClientKey  = filepath.Join("testdata", "certs", "Org%d-child%d-client%d-key.pem")
	childClientCert = filepath.Join("testdata", "certs", "Org%d-child%d-client%d-cert.pem")
)

type testServer struct {
	config comm.ServerConfig
}

type serverCert struct {
	keyPEM  []byte
	certPEM []byte
}

type testOrg struct {
	rootCA      []byte
	serverCerts []serverCert
	clientCerts []tls.Certificate
	childOrgs   []testOrg
}

//为组织的rootca返回*x509.certpool
func (org *testOrg) rootCertPool() *x509.CertPool {
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(org.rootCA)
	return certPool
}

//返回组织的测试服务器
func (org *testOrg) testServers(clientRootCAs [][]byte) []testServer {

	var testServers = []testServer{}
	clientRootCAs = append(clientRootCAs, org.rootCA)
//循环访问服务器证书并创建测试服务器
	for _, serverCert := range org.serverCerts {
		testServer := testServer{
			comm.ServerConfig{
				ConnectionTimeout: 250 * time.Millisecond,
				SecOpts: &comm.SecureOptions{
					UseTLS:            true,
					Certificate:       serverCert.certPEM,
					Key:               serverCert.keyPEM,
					RequireClientCert: true,
					ClientRootCAs:     clientRootCAs,
				},
			},
		}
		testServers = append(testServers, testServer)
	}
	return testServers
}

//返回组织的受信任客户端
func (org *testOrg) trustedClients(serverRootCAs [][]byte) []*tls.Config {

	var trustedClients = []*tls.Config{}
//如果我们有任何额外的服务器根CA，请将它们添加到证书池
	certPool := org.rootCertPool()
	for _, serverRootCA := range serverRootCAs {
		certPool.AppendCertsFromPEM(serverRootCA)
	}

//循环访问客户端证书并创建tls.configs
	for _, clientCert := range org.clientCerts {
		trustedClient := &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      certPool,
		}
		trustedClients = append(trustedClients, trustedClient)
	}
	return trustedClients
}

//CreateCertPool从PEM编码的证书数组创建一个x509.certpool
func createCertPool(rootCAs [][]byte) (*x509.CertPool, error) {

	certPool := x509.NewCertPool()
	for _, rootCA := range rootCAs {
		if !certPool.AppendCertsFromPEM(rootCA) {
			return nil, errors.New("Failed to load root certificates")
		}
	}
	return certPool, nil
}

//为组织加载加密材料的实用功能
func loadOrg(parent int) (testOrg, error) {

	var org = testOrg{}
//加载CA
	caPEM, err := ioutil.ReadFile(fmt.Sprintf(orgCACert, parent))
	if err != nil {
		return org, err
	}
//循环和加载服务器
	var serverCerts = []serverCert{}
	for i := 1; i <= numServerCerts; i++ {
		keyPEM, err := ioutil.ReadFile(fmt.Sprintf(orgServerKey, parent, i))
		if err != nil {
			return org, err
		}
		certPEM, err := ioutil.ReadFile(fmt.Sprintf(orgServerCert, parent, i))
		if err != nil {
			return org, err
		}
		serverCerts = append(serverCerts, serverCert{keyPEM, certPEM})
	}
//循环并加载客户端
	var clientCerts = []tls.Certificate{}
	for j := 1; j <= numServerCerts; j++ {
		clientCert, err := loadTLSKeyPairFromFile(fmt.Sprintf(orgClientKey, parent, j),
			fmt.Sprintf(orgClientCert, parent, j))
		if err != nil {
			return org, err
		}
		clientCerts = append(clientCerts, clientCert)
	}
//循环并加载子组织
	var childOrgs = []testOrg{}

	for k := 1; k <= numChildOrgs; k++ {
		childOrg, err := loadChildOrg(parent, k)
		if err != nil {
			return org, err
		}
		childOrgs = append(childOrgs, childOrg)
	}

	return testOrg{caPEM, serverCerts, clientCerts, childOrgs}, nil
}

//为子组织加载加密材料的实用功能
func loadChildOrg(parent, child int) (testOrg, error) {

	var org = testOrg{}
//加载CA
	caPEM, err := ioutil.ReadFile(fmt.Sprintf(childCACert, parent, child))
	if err != nil {
		return org, err
	}
//循环和加载服务器
	var serverCerts = []serverCert{}
	for i := 1; i <= numServerCerts; i++ {
		keyPEM, err := ioutil.ReadFile(fmt.Sprintf(childServerKey, parent, child, i))
		if err != nil {
			return org, err
		}
		certPEM, err := ioutil.ReadFile(fmt.Sprintf(childServerCert, parent, child, i))
		if err != nil {
			return org, err
		}
		serverCerts = append(serverCerts, serverCert{keyPEM, certPEM})
	}
//循环并加载客户端
	var clientCerts = []tls.Certificate{}
	for j := 1; j <= numServerCerts; j++ {
		clientCert, err := loadTLSKeyPairFromFile(fmt.Sprintf(childClientKey, parent, child, j),
			fmt.Sprintf(childClientCert, parent, child, j))
		if err != nil {
			return org, err
		}
		clientCerts = append(clientCerts, clientCert)
	}
	return testOrg{caPEM, serverCerts, clientCerts, []testOrg{}}, nil
}

//loadtlskeypairpromfile从pem编码的密钥和证书文件创建一个tls.certificate
func loadTLSKeyPairFromFile(keyFile, certFile string) (tls.Certificate, error) {

	certPEMBlock, err := ioutil.ReadFile(certFile)
	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)

	if err != nil {
		return tls.Certificate{}, err
	}
	return cert, nil
}

func TestNewGRPCServerInvalidParameters(t *testing.T) {

	t.Parallel()
//遗漏地址
	_, err := comm.NewGRPCServer("", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{UseTLS: false}})
//检查错误
	msg := "Missing address parameter"
	assert.EqualError(t, err, msg)
	if err != nil {
		t.Log(err.Error())
	}

//丢失端口
	_, err = comm.NewGRPCServer("abcdef", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{UseTLS: false}})
//检查错误
	assert.Error(t, err, "Expected error with missing port")
	msg = "missing port in address"
	assert.Contains(t, err.Error(), msg)

//坏端口
	_, err = comm.NewGRPCServer("localhost:1BBB", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{UseTLS: false}})
//基于平台检查可能的错误并发布
	msgs := []string{
		"listen tcp: lookup tcp/1BBB: nodename nor servname provided, or not known",
		"listen tcp: unknown port tcp/1BBB",
		"listen tcp: address tcp/1BBB: unknown port",
		"listen tcp: lookup tcp/1BBB: Servname not supported for ai_socktype",
	}

	if assert.Error(t, err, fmt.Sprintf("[%s], [%s] [%s] or [%s] expected", msgs[0], msgs[1], msgs[2], msgs[3])) {
		assert.Contains(t, msgs, err.Error())
	}
	if err != nil {
		t.Log(err.Error())
	}

//坏主机名
	_, err = comm.NewGRPCServer("hostdoesnotexist.localdomain:9050",
		comm.ServerConfig{SecOpts: &comm.SecureOptions{UseTLS: false}})
 /*
  我们无法检查特定的错误消息，因为
  系统将自动将未知主机名解析为“搜索”
  地址，所以我们只是检查以确保返回错误
 **/

	assert.Error(t, err, fmt.Sprintf("%s error expected", msg))
	if err != nil {
		t.Log(err.Error())
	}

//使用地址
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener [%s]", err)
	}
	defer lis.Close()
	_, err = comm.NewGRPCServerFromListener(
		lis,
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{UseTLS: false}})
	if err != nil {
		t.Fatalf("Failed to create GRPCServer [%s]", err)
	}
	_, err = comm.NewGRPCServer(
		lis.Addr().String(),
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{UseTLS: false}},
	)
//检查错误
	if err != nil {
		t.Log(err.Error())
	}
	assert.Contains(t, err.Error(), "address already in use")

//缺少服务器证书
	_, err = comm.NewGRPCServerFromListener(
		lis,
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{
				UseTLS: true,
				Key:    []byte{}},
		},
	)
//检查错误
	msg = "serverConfig.SecOpts must contain both Key and " +
		"Certificate when UseTLS is true"
	assert.EqualError(t, err, msg)
	if err != nil {
		t.Log(err.Error())
	}

//缺少服务器密钥
	_, err = comm.NewGRPCServerFromListener(
		lis,
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{
				UseTLS:      true,
				Certificate: []byte{}},
		},
	)
//检查错误
	assert.EqualError(t, err, msg)
	if err != nil {
		t.Log(err.Error())
	}

//坏服务器密钥
	_, err = comm.NewGRPCServerFromListener(
		lis,
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{
				UseTLS:      true,
				Certificate: []byte(selfSignedCertPEM),
				Key:         []byte{}},
		},
	)

//检查错误
	msg = "tls: failed to find any PEM data in key input"
	assert.EqualError(t, err, msg)
	if err != nil {
		t.Log(err.Error())
	}

//错误的服务器证书
	_, err = comm.NewGRPCServerFromListener(
		lis,
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{
				UseTLS:      true,
				Certificate: []byte{},
				Key:         []byte(selfSignedKeyPEM)},
		},
	)
//检查错误
	msg = "tls: failed to find any PEM data in certificate input"
	assert.EqualError(t, err, msg)
	if err != nil {
		t.Log(err.Error())
	}

	srv, err := comm.NewGRPCServerFromListener(
		lis,
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{
				UseTLS:            true,
				Certificate:       []byte(selfSignedCertPEM),
				Key:               []byte(selfSignedKeyPEM),
				RequireClientCert: true},
		},
	)
	badRootCAs := [][]byte{[]byte(badPEM)}
	err = srv.SetClientRootCAs(badRootCAs)
//检查错误
	msg = "Failed to set client root certificate(s): " +
		"asn1: syntax error: data truncated"
	assert.EqualError(t, err, msg)
	if err != nil {
		t.Log(err.Error())
	}
}

func TestNewGRPCServer(t *testing.T) {

	t.Parallel()
	testAddress := "localhost:9053"
	srv, err := comm.NewGRPCServer(
		testAddress,
		comm.ServerConfig{SecOpts: &comm.SecureOptions{UseTLS: false}},
	)
//检查错误
	if err != nil {
		t.Fatalf("Failed to return new GRPC server: %v", err)
	}

//确保我们的财产符合预期
//解析地址
	addr, err := net.ResolveTCPAddr("tcp", testAddress)
	assert.Equal(t, srv.Address(), addr.String())
	assert.Equal(t, srv.Listener().Addr().String(), addr.String())

//tlsenabled应为false
	assert.Equal(t, srv.TLSEnabled(), false)
//mutualTlsRequired应为false
	assert.Equal(t, srv.MutualTLSRequired(), false)

//注册GRPC测试服务器
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})

//启动服务器
	go srv.Start()

	defer srv.Stop()
//不需要
	time.Sleep(10 * time.Millisecond)

//GRPC客户端选项
	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithInsecure())

//调用EmptyCall服务
	_, err = invokeEmptyCall(testAddress, dialOptions)

	if err != nil {
		t.Fatalf("GRPC client failed to invoke the EmptyCall service on %s: %v",
			testAddress, err)
	} else {
		t.Log("GRPC client successfully invoked the EmptyCall service: " + testAddress)
	}

}

func TestNewGRPCServerFromListener(t *testing.T) {

	t.Parallel()

//创造我们的听众
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testAddress := lis.Addr().String()

	srv, err := comm.NewGRPCServerFromListener(
		lis,
		comm.ServerConfig{
			SecOpts: &comm.SecureOptions{UseTLS: false}},
	)
//检查错误
	if err != nil {
		t.Fatalf("Failed to return new GRPC server: %v", err)
	}

//确保我们的财产符合预期
//解析地址
	addr, err := net.ResolveTCPAddr("tcp", testAddress)
	assert.Equal(t, srv.Address(), addr.String())
	assert.Equal(t, srv.Listener().Addr().String(), addr.String())

//tlsenabled应为false
	assert.Equal(t, srv.TLSEnabled(), false)
//mutualTlsRequired应为false
	assert.Equal(t, srv.MutualTLSRequired(), false)

//注册GRPC测试服务器
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})

//启动服务器
	go srv.Start()

	defer srv.Stop()
//不需要
	time.Sleep(10 * time.Millisecond)

//GRPC客户端选项
	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithInsecure())

//调用EmptyCall服务
	_, err = invokeEmptyCall(testAddress, dialOptions)

	if err != nil {
		t.Fatalf("GRPC client failed to invoke the EmptyCall service on %s: %v", testAddress, err)
	} else {
		t.Log("GRPC client successfully invoked the EmptyCall service: " + testAddress)
	}
}

func TestNewSecureGRPCServer(t *testing.T) {

	t.Parallel()
//创造我们的听众
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testAddress := lis.Addr().String()
	srv, err := comm.NewGRPCServerFromListener(lis, comm.ServerConfig{
		ConnectionTimeout: 250 * time.Millisecond,
		SecOpts: &comm.SecureOptions{
			UseTLS:      true,
			Certificate: []byte(selfSignedCertPEM),
			Key:         []byte(selfSignedKeyPEM)}})
//检查错误
	if err != nil {
		t.Fatalf("Failed to return new GRPC server: %v", err)
	}

//确保我们的财产符合预期
//解析地址
	addr, err := net.ResolveTCPAddr("tcp", testAddress)
	assert.Equal(t, srv.Address(), addr.String())
	assert.Equal(t, srv.Listener().Addr().String(), addr.String())

//检查服务器证书
	cert, _ := tls.X509KeyPair([]byte(selfSignedCertPEM), []byte(selfSignedKeyPEM))
	assert.Equal(t, srv.ServerCertificate(), cert)

//tlsenabled应为true
	assert.Equal(t, srv.TLSEnabled(), true)
//mutualTlsRequired应为false
	assert.Equal(t, srv.MutualTLSRequired(), false)

//注册GRPC测试服务器
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})

//启动服务器
	go srv.Start()

	defer srv.Stop()
//不需要
	time.Sleep(10 * time.Millisecond)

//创建客户端凭据
	certPool := x509.NewCertPool()

	if !certPool.AppendCertsFromPEM([]byte(selfSignedCertPEM)) {

		t.Fatal("Failed to append certificate to client credentials")
	}

	creds := credentials.NewClientTLSFromCert(certPool, "")

//GRPC客户端选项
	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))

//调用EmptyCall服务
	_, err = invokeEmptyCall(testAddress, dialOptions)

	if err != nil {
		t.Fatalf("GRPC client failed to invoke the EmptyCall service on %s: %v",
			testAddress, err)
	} else {
		t.Log("GRPC client successfully invoked the EmptyCall service: " + testAddress)
	}

	tlsVersions := []string{"SSL30", "TLS10", "TLS11"}
	for counter, tlsVersion := range []uint16{tls.VersionSSL30, tls.VersionTLS10, tls.VersionTLS11} {
		tlsVersion := tlsVersion
		t.Run(tlsVersions[counter], func(t *testing.T) {
			t.Parallel()
			_, err := invokeEmptyCall(testAddress,
				[]grpc.DialOption{grpc.WithTransportCredentials(
					credentials.NewTLS(&tls.Config{
						RootCAs:    certPool,
						MinVersion: tlsVersion,
						MaxVersion: tlsVersion,
					})),
					grpc.WithBlock()})
			t.Logf("TLSVersion [%d] failed with [%s]", tlsVersion, err)
			assert.Error(t, err, "Should not have been able to connect with TLS version < 1.2")
			assert.Contains(t, err.Error(), "context deadline exceeded")
		})
	}
}

func TestVerifyCertificateCallback(t *testing.T) {
	t.Parallel()

	ca, err := tlsgen.NewCA()
	assert.NoError(t, err)

	authorizedClientKeyPair, err := ca.NewClientCertKeyPair()
	assert.NoError(t, err)

	notAuthorizedClientKeyPair, err := ca.NewClientCertKeyPair()
	assert.NoError(t, err)

	serverKeyPair, err := ca.NewServerCertKeyPair("127.0.0.1")
	assert.NoError(t, err)

	verifyFunc := func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if bytes.Equal(rawCerts[0], authorizedClientKeyPair.TLSCert.Raw) {
			return nil
		}
		return errors.New("certificate mismatch")
	}

	probeTLS := func(endpoint string, clientKeyPair *tlsgen.CertKeyPair) error {
		cert, err := tls.X509KeyPair(clientKeyPair.Cert, clientKeyPair.Key)
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      x509.NewCertPool(),
		}
		tlsCfg.RootCAs.AppendCertsFromPEM(ca.CertBytes())

		conn, err := tls.Dial("tcp", endpoint, tlsCfg)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}

	gRPCServer, err := comm.NewGRPCServer("127.0.0.1:", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			ClientRootCAs:     [][]byte{ca.CertBytes()},
			Key:               serverKeyPair.Key,
			Certificate:       serverKeyPair.Cert,
			UseTLS:            true,
			VerifyCertificate: verifyFunc,
		},
	})
	go gRPCServer.Start()
	defer gRPCServer.Stop()

	t.Run("Success path", func(t *testing.T) {
		err = probeTLS(gRPCServer.Address(), authorizedClientKeyPair)
		assert.NoError(t, err)
	})

	t.Run("Failure path", func(t *testing.T) {
		err = probeTLS(gRPCServer.Address(), notAuthorizedClientKeyPair)
		assert.EqualError(t, err, "remote error: tls: bad certificate")
	})

}

//以前的测试使用由GRPCServer和测试客户端加载的自签名证书
//在这里，我们将使用由证书颁发机构签名的证书
func TestWithSignedRootCertificates(t *testing.T) {

	t.Parallel()
//使用ORG1测试数据
	fileBase := "Org1"
	certPEMBlock, err := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-server1-cert.pem"))
	if err != nil {
		t.Fatalf("Failed to load test certificates: %v", err)
	}
	keyPEMBlock, err := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-server1-key.pem"))
	if err != nil {
		t.Fatalf("Failed to load test certificates: %v", err)
	}
	caPEMBlock, err := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-cert.pem"))
	if err != nil {
		t.Fatalf("Failed to load test certificates: %v", err)
	}

//创造我们的听众
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testAddress := lis.Addr().String()

	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	srv, err := comm.NewGRPCServerFromListener(lis, comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			UseTLS:      true,
			Certificate: certPEMBlock,
			Key:         keyPEMBlock}})
//检查错误
	if err != nil {
		t.Fatalf("Failed to return new GRPC server: %v", err)
	}

//注册GRPC测试服务器
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})

//启动服务器
	go srv.Start()

	defer srv.Stop()
//不需要
	time.Sleep(10 * time.Millisecond)

//创建一个证书池供客户端仅与服务器证书一起使用
	certPoolServer, err := createCertPool([][]byte{certPEMBlock})
	if err != nil {
		t.Fatalf("Failed to load root certificates into pool: %v", err)
	}
//创建客户端凭据
	creds := credentials.NewClientTLSFromCert(certPoolServer, "")

//GRPC客户端选项
	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))

//调用EmptyCall服务
	_, err = invokeEmptyCall(testAddress, dialOptions)

//客户端应该能够与Go 1.9连接
	assert.NoError(t, err, "Expected client to connect with server cert only")

//现在使用CA证书
	certPoolCA := x509.NewCertPool()
	if !certPoolCA.AppendCertsFromPEM(caPEMBlock) {
		t.Fatal("Failed to append certificate to client credentials")
	}
	creds = credentials.NewClientTLSFromCert(certPoolCA, "")
	var dialOptionsCA []grpc.DialOption
	dialOptionsCA = append(dialOptionsCA, grpc.WithTransportCredentials(creds))

//调用EmptyCall服务
	_, err2 := invokeEmptyCall(testAddress, dialOptionsCA)

	if err2 != nil {
		t.Fatalf("GRPC client failed to invoke the EmptyCall service on %s: %v",
			testAddress, err2)
	} else {
		t.Log("GRPC client successfully invoked the EmptyCall service: " + testAddress)
	}
}

//这里我们将使用由中级证书颁发机构签名的证书
func TestWithSignedIntermediateCertificates(t *testing.T) {

	t.Parallel()
//使用ORG1测试数据
	fileBase := "Org1"
	certPEMBlock, err := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-child1-server1-cert.pem"))
	keyPEMBlock, err := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-child1-server1-key.pem"))
	intermediatePEMBlock, err := ioutil.ReadFile(filepath.Join("testdata", "certs", fileBase+"-child1-cert.pem"))

	if err != nil {
		t.Fatalf("Failed to load test certificates: %v", err)
	}
//创造我们的听众
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testAddress := lis.Addr().String()

	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	srv, err := comm.NewGRPCServerFromListener(lis, comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			UseTLS:      true,
			Certificate: certPEMBlock,
			Key:         keyPEMBlock}})
//检查错误
	if err != nil {
		t.Fatalf("Failed to return new GRPC server: %v", err)
	}

//注册GRPC测试服务器
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})

//启动服务器
	go srv.Start()

	defer srv.Stop()
//不需要
	time.Sleep(10 * time.Millisecond)

//创建一个证书池供客户端仅与服务器证书一起使用
	certPoolServer, err := createCertPool([][]byte{certPEMBlock})
	if err != nil {
		t.Fatalf("Failed to load root certificates into pool: %v", err)
	}
//创建客户端凭据
	creds := credentials.NewClientTLSFromCert(certPoolServer, "")

//GRPC客户端选项
	var dialOptions []grpc.DialOption
	dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))

//调用EmptyCall服务
	_, err = invokeEmptyCall(testAddress, dialOptions)

//客户端应该能够与Go 1.9连接
	assert.NoError(t, err, "Expected client to connect with server cert only")

//现在使用CA证书
//创建一个证书池供具有中间根CA的客户端使用
	certPoolCA, err := createCertPool([][]byte{intermediatePEMBlock})
	if err != nil {
		t.Fatalf("Failed to load root certificates into pool: %v", err)
	}

	creds = credentials.NewClientTLSFromCert(certPoolCA, "")
	var dialOptionsCA []grpc.DialOption
	dialOptionsCA = append(dialOptionsCA, grpc.WithTransportCredentials(creds))

//调用EmptyCall服务
	_, err2 := invokeEmptyCall(testAddress, dialOptionsCA)

	if err2 != nil {
		t.Fatalf("GRPC client failed to invoke the EmptyCall service on %s: %v",
			testAddress, err2)
	} else {
		t.Log("GRPC client successfully invoked the EmptyCall service: " + testAddress)
	}
}

//使用tls测试客户机/服务器通信的实用功能
func runMutualAuth(t *testing.T, servers []testServer, trustedClients, unTrustedClients []*tls.Config) error {

//循环访问所有测试服务器
	for i := 0; i < len(servers); i++ {
//创建侦听器
		lis, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			return err
		}
		srvAddr := lis.Addr().String()

//创建GRPCServer
		srv, err := comm.NewGRPCServerFromListener(lis, servers[i].config)
		if err != nil {
			return err
		}

//mutualTlsRequired应为true
		assert.Equal(t, srv.MutualTLSRequired(), true)

//注册GRPC测试服务器并启动GRPCServer
		testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
		go srv.Start()
		defer srv.Stop()
//不需要，只是以防万一
		time.Sleep(10 * time.Millisecond)

//循环访问所有受信任的客户端
		for j := 0; j < len(trustedClients); j++ {
//调用EmptyCall服务
			_, err = invokeEmptyCall(srvAddr,
				[]grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(trustedClients[j]))})
//我们期待来自可靠客户的成功
			if err != nil {
				return err
			} else {
				t.Logf("Trusted client%d successfully connected to %s", j, srvAddr)
			}
		}
//循环访问所有不受信任的客户端
		for k := 0; k < len(unTrustedClients); k++ {
//调用EmptyCall服务
			_, err = invokeEmptyCall(
				srvAddr,
				[]grpc.DialOption{
					grpc.WithTransportCredentials(
						credentials.NewTLS(unTrustedClients[k]))})
//我们期望不受信任的客户失败
			if err != nil {
				t.Logf("Untrusted client%d was correctly rejected by %s", k, srvAddr)
			} else {
				return fmt.Errorf("Untrusted client %d should not have been able to connect to %s", k,
					srvAddr)
			}
		}
	}

	return nil
}

func TestMutualAuth(t *testing.T) {

	t.Parallel()
	var tests = []struct {
		name             string
		servers          []testServer
		trustedClients   []*tls.Config
		unTrustedClients []*tls.Config
	}{
		{
			name:             "ClientAuthRequiredWithSingleOrg",
			servers:          testOrgs[0].testServers([][]byte{}),
			trustedClients:   testOrgs[0].trustedClients([][]byte{}),
			unTrustedClients: testOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA}),
		},
		{
			name:             "ClientAuthRequiredWithChildClientOrg",
			servers:          testOrgs[0].testServers([][]byte{testOrgs[0].childOrgs[0].rootCA}),
			trustedClients:   testOrgs[0].childOrgs[0].trustedClients([][]byte{testOrgs[0].rootCA}),
			unTrustedClients: testOrgs[0].childOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA}),
		},
		{
			name: "ClientAuthRequiredWithMultipleChildClientOrgs",
			servers: testOrgs[0].testServers(append([][]byte{},
				testOrgs[0].childOrgs[0].rootCA, testOrgs[0].childOrgs[1].rootCA)),
			trustedClients: append(append([]*tls.Config{},
				testOrgs[0].childOrgs[0].trustedClients([][]byte{testOrgs[0].rootCA})...),
				testOrgs[0].childOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA})...),
			unTrustedClients: testOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA}),
		},
		{
			name:             "ClientAuthRequiredWithDifferentServerAndClientOrgs",
			servers:          testOrgs[0].testServers([][]byte{testOrgs[1].rootCA}),
			trustedClients:   testOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA}),
			unTrustedClients: testOrgs[0].childOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA}),
		},
		{
			name:             "ClientAuthRequiredWithDifferentServerAndChildClientOrgs",
			servers:          testOrgs[1].testServers([][]byte{testOrgs[0].childOrgs[0].rootCA}),
			trustedClients:   testOrgs[0].childOrgs[0].trustedClients([][]byte{testOrgs[1].rootCA}),
			unTrustedClients: testOrgs[1].childOrgs[0].trustedClients([][]byte{testOrgs[1].rootCA}),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("Running test %s ...", test.name)
			testErr := runMutualAuth(t, test.servers, test.trustedClients, test.unTrustedClients)
			if testErr != nil {
				t.Fatalf("%s failed with error: %s", test.name, testErr.Error())
			}
		})
	}

}

func TestAppendRemoveWithInvalidBytes(t *testing.T) {

//TODO:解决不带PEM类型的MSP序列化时重新访问
	t.Skip()
	t.Parallel()

	noPEMData := [][]byte{[]byte("badcert1"), []byte("badCert2")}

//获取我们的ORG1测试服务器之一的配置
	serverConfig := testOrgs[0].testServers([][]byte{})[0].config
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener [%s]", err)
	}
	defer lis.Close()

//创建GRPCServer
	srv, err := comm.NewGRPCServerFromListener(lis, serverConfig)
	if err != nil {
		t.Fatalf("Failed to create GRPCServer due to: %s", err.Error())
	}

//附加/删除非项目数据
	noCertsFound := "No client root certificates found"
	err = srv.AppendClientRootCAs(noPEMData)
	if err == nil {
		t.Fatalf("Expected error: %s", noCertsFound)
	}
	err = srv.RemoveClientRootCAs(noPEMData)
	if err == nil {
		t.Fatalf("Expected error: %s", noCertsFound)
	}

//删除没有证书头的PEM
	err = srv.AppendClientRootCAs([][]byte{[]byte(pemNoCertificateHeader)})
	if err == nil {
		t.Fatalf("Expected error: %s", noCertsFound)
	}

	err = srv.RemoveClientRootCAs([][]byte{[]byte(pemNoCertificateHeader)})
	if err == nil {
		t.Fatalf("Expected error: %s", noCertsFound)
	}

//附加/删除错误的PEM数据
	err = srv.AppendClientRootCAs([][]byte{[]byte(badPEM)})
	if err == nil {
		t.Fatalf("Expected error parsing bad PEM data")
	}

	err = srv.RemoveClientRootCAs([][]byte{[]byte(badPEM)})
	if err == nil {
		t.Fatalf("Expected error parsing bad PEM data")
	}

}

func TestAppendClientRootCAs(t *testing.T) {

	t.Parallel()
//获取我们的ORG1测试服务器之一的配置
	serverConfig := testOrgs[0].testServers([][]byte{})[0].config
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener [%s]", err)
	}
	defer lis.Close()
	address := lis.Addr().String()

//创建GRPCServer
	srv, err := comm.NewGRPCServerFromListener(lis, serverConfig)
	if err != nil {
		t.Fatalf("Failed to create GRPCServer due to: %s", err.Error())
	}

//注册GRPC测试服务器并启动GRPCServer
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
	go srv.Start()
	defer srv.Stop()
//不需要，只是以防万一
	time.Sleep(10 * time.Millisecond)

//尝试连接来自org2子级的不受信任的客户端
	clientConfig1 := testOrgs[1].childOrgs[0].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfig2 := testOrgs[1].childOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfigs := []*tls.Config{clientConfig1, clientConfig2}

	for i, clientConfig := range clientConfigs {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})
//我们期望失败，因为这些客户端当前不受信任
		if err != nil {
			t.Logf("Untrusted client%d was correctly rejected by %s", i, address)
		} else {
			t.Fatalf("Untrusted client %d should not have been able to connect to %s", i,
				address)
		}
	}

//现在为不受信任的客户机附加根CA
	err = srv.AppendClientRootCAs([][]byte{testOrgs[1].childOrgs[0].rootCA,
		testOrgs[1].childOrgs[1].rootCA})
	if err != nil {
		t.Fatal("Failed to append client root CAs")
	}

//现在尝试再次连接
	for j, clientConfig := range clientConfigs {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})
//我们期待成功，因为这些客户现在都是值得信赖的客户。
		if err != nil {
			t.Fatalf("Now trusted client%d failed to connect to %s with error: %s",
				j, address, err.Error())
		} else {
			t.Logf("Now trusted client%d successfully connected to %s", j, address)
		}
	}

}

func TestRemoveClientRootCAs(t *testing.T) {

	t.Parallel()
//获取org1测试服务器之一的配置，并从中包含客户端CA
//Org2儿童组织
	testServers := testOrgs[0].testServers(
		[][]byte{testOrgs[1].childOrgs[0].rootCA,
			testOrgs[1].childOrgs[1].rootCA},
	)
	serverConfig := testServers[0].config
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener [%s]", err)
	}
	defer lis.Close()
	address := lis.Addr().String()

//创建GRPCServer
	srv, err := comm.NewGRPCServerFromListener(lis, serverConfig)
	if err != nil {
		t.Fatalf("Failed to create GRPCServer due to: %s", err.Error())
	}

//注册GRPC测试服务器并启动GRPCServer
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
	go srv.Start()
	defer srv.Stop()
//不需要，只是以防万一
	time.Sleep(10 * time.Millisecond)

//尝试连接来自org2子级的受信任客户端
	clientConfig1 := testOrgs[1].childOrgs[0].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfig2 := testOrgs[1].childOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfigs := []*tls.Config{clientConfig1, clientConfig2}

	for i, clientConfig := range clientConfigs {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})

//我们期待成功，因为他们是值得信赖的客户
		if err != nil {
			t.Fatalf("Trusted client%d failed to connect to %s with error: %s",
				i, address, err.Error())
		} else {
			t.Logf("Trusted client%d successfully connected to %s", i, address)
		}
	}

//现在删除不受信任客户端的根CA
	err = srv.RemoveClientRootCAs([][]byte{testOrgs[1].childOrgs[0].rootCA,
		testOrgs[1].childOrgs[1].rootCA})
	if err != nil {
		t.Fatal("Failed to remove client root CAs")
	}

//现在尝试再次连接
	for j, clientConfig := range clientConfigs {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})
//我们期望失败，因为这些客户现在不受信任
		if err != nil {
			t.Logf("Now untrusted client%d was correctly rejected by %s", j, address)
		} else {
			t.Fatalf("Now untrusted client %d should not have been able to connect to %s", j,
				address)
		}
	}

}

//比赛条件测试-使用“go test-race-run testconcurrentappendremoveset”进行本地测试
func TestConcurrentAppendRemoveSet(t *testing.T) {

	t.Parallel()
//获取org1测试服务器之一的配置，并从中包含客户端CA
//Org2儿童组织
	testServers := testOrgs[0].testServers(
		[][]byte{testOrgs[1].childOrgs[0].rootCA,
			testOrgs[1].childOrgs[1].rootCA},
	)
	serverConfig := testServers[0].config
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener [%s]", err)
	}
	defer lis.Close()

//创建GRPCServer
	srv, err := comm.NewGRPCServerFromListener(lis, serverConfig)
	if err != nil {
		t.Fatalf("Failed to create GRPCServer due to: %s", err.Error())
	}

//注册GRPC测试服务器并启动GRPCServer
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
	go srv.Start()
	defer srv.Stop()

//需要等待以下go例程完成
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
//现在删除不受信任客户端的根CA
		err := srv.RemoveClientRootCAs([][]byte{testOrgs[1].childOrgs[0].rootCA,
			testOrgs[1].childOrgs[1].rootCA})
		if err != nil {
			t.Fatal("Failed to remove client root CAs")
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
//设置客户端根CA
		err := srv.SetClientRootCAs([][]byte{testOrgs[1].childOrgs[0].rootCA,
			testOrgs[1].childOrgs[1].rootCA})
		if err != nil {
			t.Fatal("Failed to set client root CAs")
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
//现在为不受信任的客户机附加根CA
		err := srv.AppendClientRootCAs([][]byte{testOrgs[1].childOrgs[0].rootCA,
			testOrgs[1].childOrgs[1].rootCA})
		if err != nil {
			t.Fatal("Failed to append client root CAs")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
//设置客户端根CA
		err := srv.SetClientRootCAs([][]byte{testOrgs[1].childOrgs[0].rootCA,
			testOrgs[1].childOrgs[1].rootCA})
		if err != nil {
			t.Fatal("Failed to set client root CAs")
		}

	}()

	wg.Wait()

}

func TestSetClientRootCAs(t *testing.T) {

	t.Parallel()

//获取我们的ORG1测试服务器之一的配置
	serverConfig := testOrgs[0].testServers([][]byte{})[0].config
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener [%s]", err)
	}
	defer lis.Close()
	address := lis.Addr().String()

//创建GRPCServer
	srv, err := comm.NewGRPCServerFromListener(lis, serverConfig)
	if err != nil {
		t.Fatalf("Failed to create GRPCServer due to: %s", err.Error())
	}

//注册GRPC测试服务器并启动GRPCServer
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
	go srv.Start()
	defer srv.Stop()
//不需要，只是以防万一
	time.Sleep(10 * time.Millisecond)

//设置测试客户端
//Org1
	clientConfigOrg1Child1 := testOrgs[0].childOrgs[0].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfigOrg1Child2 := testOrgs[0].childOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfigsOrg1Children := []*tls.Config{clientConfigOrg1Child1, clientConfigOrg1Child2}
	org1ChildRootCAs := [][]byte{testOrgs[0].childOrgs[0].rootCA,
		testOrgs[0].childOrgs[1].rootCA}
//Org2
	clientConfigOrg2Child1 := testOrgs[1].childOrgs[0].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfigOrg2Child2 := testOrgs[1].childOrgs[1].trustedClients([][]byte{testOrgs[0].rootCA})[0]
	clientConfigsOrg2Children := []*tls.Config{clientConfigOrg2Child1, clientConfigOrg2Child2}
	org2ChildRootCAs := [][]byte{testOrgs[1].childOrgs[0].rootCA,
		testOrgs[1].childOrgs[1].rootCA}

//最初将客户端CA设置为org1子级
	err = srv.SetClientRootCAs(org1ChildRootCAs)
	if err != nil {
		t.Fatalf("SetClientRootCAs failed due to: %s", err.Error())
	}

//当前信任ClientConfigsOrg1Children
	for i, clientConfig := range clientConfigsOrg1Children {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})

//我们期待成功，因为他们是值得信赖的客户
		if err != nil {
			t.Fatalf("Trusted client%d failed to connect to %s with error: %s",
				i, address, err.Error())
		} else {
			t.Logf("Trusted client%d successfully connected to %s", i, address)
		}
	}

//当前不信任ClientConfigsOrg2Children
	for j, clientConfig := range clientConfigsOrg2Children {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})
//我们期望失败，因为这些客户现在不受信任
		if err != nil {
			t.Logf("Untrusted client%d was correctly rejected by %s", j, address)
		} else {
			t.Fatalf("Untrusted client %d should not have been able to connect to %s", j,
				address)
		}
	}

//现在将客户端CA设置为org2子级
	err = srv.SetClientRootCAs(org2ChildRootCAs)
	if err != nil {
		t.Fatalf("SetClientRootCAs failed due to: %s", err.Error())
	}

//现在反转受信任和不受信任
//当前信任ClientConfigsOrg1Children
	for i, clientConfig := range clientConfigsOrg2Children {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})

//我们期待成功，因为他们是值得信赖的客户
		if err != nil {
			t.Fatalf("Trusted client%d failed to connect to %s with error: %s",
				i, address, err.Error())
		} else {
			t.Logf("Trusted client%d successfully connected to %s", i, address)
		}
	}

//当前不信任ClientConfigsOrg2Children
	for j, clientConfig := range clientConfigsOrg1Children {
//调用EmptyCall服务
		_, err = invokeEmptyCall(address, []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(clientConfig))})
//我们期望失败，因为这些客户现在不受信任
		if err != nil {
			t.Logf("Untrusted client%d was correctly rejected by %s", j, address)
		} else {
			t.Fatalf("Untrusted client %d should not have been able to connect to %s", j,
				address)
		}
	}

}

func TestKeepaliveNoClientResponse(t *testing.T) {
	t.Parallel()
//设置grpcserver实例
	kap := &comm.KeepaliveOptions{
		ServerInterval: 2 * time.Second,
		ServerTimeout:  1 * time.Second,
	}
//创造我们的听众
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testAddress := lis.Addr().String()
	srv, err := comm.NewGRPCServerFromListener(lis, comm.ServerConfig{KaOpts: kap})
	assert.NoError(t, err, "Unexpected error starting GRPCServer")
	go srv.Start()
	defer srv.Stop()

//如果客户端不响应ping，则测试连接关闭
//NET客户端不响应keepalive
	client, err := net.Dial("tcp", testAddress)
	assert.NoError(t, err, "Unexpected error dialing GRPCServer")
	defer client.Close()
//睡眠超过keepalive超时
	time.Sleep(4 * time.Second)
	data := make([]byte, 24)
	for {
		_, err = client.Read(data)
		if err == nil {
			continue
		}
		assert.EqualError(t, err, io.EOF.Error(), "Expected io.EOF")
		break
	}
}

func TestKeepaliveClientResponse(t *testing.T) {
	t.Parallel()
//设置grpcserver实例
	kap := &comm.KeepaliveOptions{
		ServerInterval: 1 * time.Second,
		ServerTimeout:  1 * time.Second,
	}
//创造我们的听众
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testAddress := lis.Addr().String()
	srv, err := comm.NewGRPCServerFromListener(lis, comm.ServerConfig{KaOpts: kap})
	if err != nil {
		t.Fatalf("Failed to create GRPCServer [%s]", err)
	}
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
	go srv.Start()
	defer srv.Stop()

//创建GRPC客户端连接
	clientCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	clientConn, err := grpc.DialContext(
		clientCtx,
		testAddress,
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create gRPC client conn [%s]", err)
	}
	defer clientConn.Close()

	stream, err := testpb.NewEmptyServiceClient(clientConn).EmptyStream(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("Failed to create EmptyServiceClient [%s]", err)
	}
	err = stream.Send(new(testpb.Empty))
	assert.NoError(t, err, "failed to send message")

//睡眠超过keepalive超时
	time.Sleep(1500 * time.Millisecond)
	err = stream.Send(new(testpb.Empty))
	assert.NoError(t, err, "failed to send message")

}

func TestUpdateTLSCert(t *testing.T) {
	t.Parallel()

	readFile := func(path string) []byte {
		fName := filepath.Join("testdata", "dynamic_cert_update", path)
		data, err := ioutil.ReadFile(fName)
		if err != nil {
			panic(fmt.Errorf("Failed reading %s: %v", fName, err))
		}
		return data
	}
	loadBytes := func(prefix string) (key, cert, caCert []byte) {
		cert = readFile(filepath.Join(prefix, "server.crt"))
		key = readFile(filepath.Join(prefix, "server.key"))
		caCert = readFile(filepath.Join("ca.crt"))
		return
	}

	key, cert, caCert := loadBytes("notlocalhost")

	cfg := comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			UseTLS:      true,
			Key:         key,
			Certificate: cert,
		},
	}
//创造我们的听众
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testAddress := lis.Addr().String()
	srv, err := comm.NewGRPCServerFromListener(lis, cfg)
	assert.NoError(t, err)
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
	go srv.Start()
	defer srv.Stop()

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	probeServer := func() error {
		_, err = invokeEmptyCall(testAddress,
			[]grpc.DialOption{grpc.WithTransportCredentials(
				credentials.NewTLS(&tls.Config{
					RootCAs: certPool})),
				grpc.WithBlock()})
		return err
	}

//引导TLS证书具有“notlocalhost”的SAN，因此它应该失败。
	err = probeServer()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")

//新的TLS证书的SAN为“127.0.0.1”，因此应该成功
	certPath := filepath.Join("testdata", "dynamic_cert_update", "localhost", "server.crt")
	keyPath := filepath.Join("testdata", "dynamic_cert_update", "localhost", "server.key")
	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	assert.NoError(t, err)
	srv.SetServerCertificate(tlsCert)
	err = probeServer()
	assert.NoError(t, err)

//恢复到旧证书，应失败。
	certPath = filepath.Join("testdata", "dynamic_cert_update", "notlocalhost", "server.crt")
	keyPath = filepath.Join("testdata", "dynamic_cert_update", "notlocalhost", "server.key")
	tlsCert, err = tls.LoadX509KeyPair(certPath, keyPath)
	assert.NoError(t, err)
	srv.SetServerCertificate(tlsCert)

	err = probeServer()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestCipherSuites(t *testing.T) {
	t.Parallel()

//默认密码套件
	defaultCipherSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	}
//Go支持的其他密码套件
	otherCipherSuites := []uint16{
		tls.TLS_RSA_WITH_RC4_128_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	}
	certPEM, err := ioutil.ReadFile(filepath.Join("testdata", "certs",
		"Org1-server1-cert.pem"))
	assert.NoError(t, err)
	keyPEM, err := ioutil.ReadFile(filepath.Join("testdata", "certs",
		"Org1-server1-key.pem"))
	assert.NoError(t, err)
	caPEM, err := ioutil.ReadFile(filepath.Join("testdata", "certs",
		"Org1-cert.pem"))
	assert.NoError(t, err)
	certPool, err := createCertPool([][]byte{caPEM})
	assert.NoError(t, err)

	serverConfig := comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			Certificate: certPEM,
			Key:         keyPEM,
			UseTLS:      true,
		}}

	var tests = []struct {
		name          string
		clientCiphers []uint16
		success       bool
	}{
		{
			name:    "server default / client all",
			success: true,
		},
		{
			name:          "server default / client match",
			clientCiphers: defaultCipherSuites,
			success:       true,
		},
		{
			name:          "server default / client no match",
			clientCiphers: otherCipherSuites,
			success:       false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("Running test %s ...", test.name)
//创造我们的听众
			lis, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("Failed to create listener: %v", err)
			}
			testAddress := lis.Addr().String()
			srv, err := comm.NewGRPCServerFromListener(lis, serverConfig)
			assert.NoError(t, err)
			go srv.Start()
			defer srv.Stop()
			tlsConfig := &tls.Config{
				RootCAs:      certPool,
				CipherSuites: test.clientCiphers,
			}
			_, err = tls.Dial("tcp", testAddress, tlsConfig)
			if test.success {
				assert.NoError(t, err)
			} else {
				t.Log(err)
				assert.Contains(t, err.Error(), "handshake failure")
			}
		})
	}
}

func TestServerInterceptors(t *testing.T) {

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: [%s]", err)
	}
	msg := "error from interceptor"

//设置拦截器
	usiCount := uint32(0)
	ssiCount := uint32(0)
	usi1 := func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {

		atomic.AddUint32(&usiCount, 1)
		return handler(ctx, req)
	}
	usi2 := func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {

		atomic.AddUint32(&usiCount, 1)
		return nil, status.Error(codes.Aborted, msg)
	}
	ssi1 := func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {

		atomic.AddUint32(&ssiCount, 1)
		return handler(srv, ss)
	}
	ssi2 := func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {

		atomic.AddUint32(&ssiCount, 1)
		return status.Error(codes.Aborted, msg)
	}

	srvConfig := comm.ServerConfig{}
	srvConfig.UnaryInterceptors = append(srvConfig.UnaryInterceptors, usi1)
	srvConfig.UnaryInterceptors = append(srvConfig.UnaryInterceptors, usi2)
	srvConfig.StreamInterceptors = append(srvConfig.StreamInterceptors, ssi1)
	srvConfig.StreamInterceptors = append(srvConfig.StreamInterceptors, ssi2)

	srv, err := comm.NewGRPCServerFromListener(lis, srvConfig)
	if err != nil {
		t.Fatalf("failed to create gRPC server: [%s]", err)
	}
	testpb.RegisterEmptyServiceServer(srv.Server(), &emptyServiceServer{})
	defer srv.Stop()
	go srv.Start()

	_, err = invokeEmptyCall(lis.Addr().String(),
		[]grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithInsecure()})
	assert.Equal(t, grpc.ErrorDesc(err), msg, "Expected error from second usi")
	assert.Equal(t, uint32(2), atomic.LoadUint32(&usiCount), "Expected both usi handlers to be invoked")

	_, err = invokeEmptyStream(lis.Addr().String(),
		[]grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithInsecure()})
	assert.Equal(t, grpc.ErrorDesc(err), msg, "Expected error from second ssi")
	assert.Equal(t, uint32(2), atomic.LoadUint32(&ssiCount), "Expected both ssi handlers to be invoked")
}
