
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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

type GRPCServer struct {
//侦听指定为hostname:port的服务器的地址
	address string
//处理网络请求的侦听器
	listener net.Listener
//GRPC服务器
	server *grpc.Server
//服务器提供的用于TLS通信的证书
//存储为原子引用
	serverCertificate atomic.Value
//服务器用于TLS通信的密钥
	serverKeyPEM []byte
//锁定以保护对追加/删除的并发访问
	lock *sync.Mutex
//用于填充的PEM编码的X509证书颁发机构集
//按主题索引的tlsconfig.clientcas
	clientRootCAs map[string]*x509.Certificate
//GRPC服务器使用的TLS配置
	tlsConfig *tls.Config
}

//new grpcserver在给定
//收听地址
func NewGRPCServer(address string, serverConfig ServerConfig) (*GRPCServer, error) {
	if address == "" {
		return nil, errors.New("Missing address parameter")
	}
//创造我们的听众
	lis, err := net.Listen("tcp", address)

	if err != nil {
		return nil, err
	}
	return NewGRPCServerFromListener(lis, serverConfig)
}

//newgrpcServerFromListener创建给定grpcServer的新实现
//使用默认keepalive的现有net.listener实例
func NewGRPCServerFromListener(listener net.Listener, serverConfig ServerConfig) (*GRPCServer, error) {
	grpcServer := &GRPCServer{
		address:  listener.Addr().String(),
		listener: listener,
		lock:     &sync.Mutex{},
	}

//设置服务器选项
	var serverOpts []grpc.ServerOption

//检查科普特
	var secureConfig SecureOptions
	if serverConfig.SecOpts != nil {
		secureConfig = *serverConfig.SecOpts
	}
	if secureConfig.UseTLS {
//密钥和证书都是必需的
		if secureConfig.Key != nil && secureConfig.Certificate != nil {
//加载服务器公钥和私钥
			cert, err := tls.X509KeyPair(secureConfig.Certificate, secureConfig.Key)
			if err != nil {
				return nil, err
			}
			grpcServer.serverCertificate.Store(cert)

//设置我们的TLS配置
			if len(secureConfig.CipherSuites) == 0 {
				secureConfig.CipherSuites = DefaultTLSCipherSuites
			}
			getCert := func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert := grpcServer.serverCertificate.Load().(tls.Certificate)
				return &cert, nil
			}
//基本服务器证书
			grpcServer.tlsConfig = &tls.Config{
				VerifyPeerCertificate:  secureConfig.VerifyCertificate,
				GetCertificate:         getCert,
				SessionTicketsDisabled: true,
				CipherSuites:           secureConfig.CipherSuites,
			}
			grpcServer.tlsConfig.ClientAuth = tls.RequestClientCert
//检查是否需要客户端身份验证
			if secureConfig.RequireClientCert {
//需要TLS客户端身份验证
				grpcServer.tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
//如果我们有客户机根CA，请创建一个证书池
				if len(secureConfig.ClientRootCAs) > 0 {
					grpcServer.clientRootCAs = make(map[string]*x509.Certificate)
					grpcServer.tlsConfig.ClientCAs = x509.NewCertPool()
					for _, clientRootCA := range secureConfig.ClientRootCAs {
						err = grpcServer.appendClientRootCA(clientRootCA)
						if err != nil {
							return nil, err
						}
					}
				}
			}

//创建凭据并添加到服务器选项
			creds := NewServerTransportCredentials(grpcServer.tlsConfig, serverConfig.Logger)
			serverOpts = append(serverOpts, grpc.Creds(creds))
		} else {
			return nil, errors.New("serverConfig.SecOpts must contain both Key and Certificate when UseTLS is true")
		}
	}
//设置最大发送和接收消息大小
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(MaxSendMsgSize))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(MaxRecvMsgSize))
//设置keepalive选项
	serverOpts = append(serverOpts, ServerKeepaliveOptions(serverConfig.KaOpts)...)
//设置连接超时
	if serverConfig.ConnectionTimeout <= 0 {
		serverConfig.ConnectionTimeout = DefaultConnectionTimeout
	}
	serverOpts = append(
		serverOpts,
		grpc.ConnectionTimeout(serverConfig.ConnectionTimeout))
//设置拦截器
	if len(serverConfig.StreamInterceptors) > 0 {
		serverOpts = append(
			serverOpts,
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(serverConfig.StreamInterceptors...)),
		)
	}
	if len(serverConfig.UnaryInterceptors) > 0 {
		serverOpts = append(
			serverOpts,
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(serverConfig.UnaryInterceptors...)),
		)
	}

	if serverConfig.MetricsProvider != nil {
		sh := NewServerStatsHandler(serverConfig.MetricsProvider)
		serverOpts = append(serverOpts, grpc.StatsHandler(sh))
	}

	grpcServer.server = grpc.NewServer(serverOpts...)

	return grpcServer, nil
}

//setServerCertificate将当前的TLS证书分配为对等方的服务器证书
func (gServer *GRPCServer) SetServerCertificate(cert tls.Certificate) {
	gServer.serverCertificate.Store(cert)
}

//地址返回此grpcserver实例的侦听地址
func (gServer *GRPCServer) Address() string {
	return gServer.address
}

//listener返回grpcserver实例的net.listener
func (gServer *GRPCServer) Listener() net.Listener {
	return gServer.listener
}

//server返回grpc server实例的grpc.server
func (gServer *GRPCServer) Server() *grpc.Server {
	return gServer.server
}

//serverCertificate返回grpc.server使用的tls.certificate
func (gServer *GRPCServer) ServerCertificate() tls.Certificate {
	return gServer.serverCertificate.Load().(tls.Certificate)
}

//TLSEnabled是一个标志，指示是否为
//grpcserver实例
func (gServer *GRPCServer) TLSEnabled() bool {
	return gServer.tlsConfig != nil
}

//mutualTlsRequired是一个标志，指示客户端证书是否
//对于此grpcserver实例是必需的
func (gServer *GRPCServer) MutualTLSRequired() bool {
	return gServer.tlsConfig != nil &&
		gServer.tlsConfig.ClientAuth ==
			tls.RequireAndVerifyClientCert
}

//启动启动基础grpc.server
func (gServer *GRPCServer) Start() error {
	return gServer.server.Serve(gServer.listener)
}

//停止停止基础grpc.server
func (gServer *GRPCServer) Stop() {
	gServer.server.Stop()
}

//AppendClientRootcas将PEM编码的X509证书颁发机构附加到
//用于验证客户端证书的权限列表
func (gServer *GRPCServer) AppendClientRootCAs(clientRoots [][]byte) error {
	gServer.lock.Lock()
	defer gServer.lock.Unlock()
	for _, clientRoot := range clientRoots {
		err := gServer.appendClientRootCA(clientRoot)
		if err != nil {
			return err
		}
	}
	return nil
}

//用于添加PEM编码的clientrootca的内部函数
func (gServer *GRPCServer) appendClientRootCA(clientRoot []byte) error {

	errMsg := "Failed to append client root certificate(s): %s"
//转换为X509
	certs, subjects, err := pemToX509Certs(clientRoot)
	if err != nil {
		return fmt.Errorf(errMsg, err.Error())
	}

	if len(certs) < 1 {
		return fmt.Errorf(errMsg, "No client root certificates found")
	}

	for i, cert := range certs {
//首先添加到客户端
		gServer.tlsConfig.ClientCAs.AddCert(cert)
//将它添加到我们的客户端rootcas映射中，使用subject作为键
		gServer.clientRootCAs[subjects[i]] = cert
	}
	return nil
}

//removeclientrootcas从中删除PEM编码的X509证书颁发机构
//用于验证客户端证书的权限列表
func (gServer *GRPCServer) RemoveClientRootCAs(clientRoots [][]byte) error {
	gServer.lock.Lock()
	defer gServer.lock.Unlock()
//从内部映射中删除
	for _, clientRoot := range clientRoots {
		err := gServer.removeClientRootCA(clientRoot)
		if err != nil {
			return err
		}
	}

//创建新的证书池并用当前的clientrootcas填充
	certPool := x509.NewCertPool()
	for _, clientRoot := range gServer.clientRootCAs {
		certPool.AddCert(clientRoot)
	}

//替换当前的clientcas池
	gServer.tlsConfig.ClientCAs = certPool
	return nil
}

//用于删除PEM编码的clientrootca的内部函数
func (gServer *GRPCServer) removeClientRootCA(clientRoot []byte) error {

	errMsg := "Failed to remove client root certificate(s): %s"
//转换为X509
	certs, subjects, err := pemToX509Certs(clientRoot)
	if err != nil {
		return fmt.Errorf(errMsg, err.Error())
	}

	if len(certs) < 1 {
		return fmt.Errorf(errMsg, "No client root certificates found")
	}

	for i, subject := range subjects {
//使用Subject作为键从客户端rootcas映射中删除它
//看看我们是否匹配
		if certs[i].Equal(gServer.clientRootCAs[subject]) {
			delete(gServer.clientRootCAs, subject)
		}
	}
	return nil
}

//setclientrootcas设置用于验证客户端的权限列表
//基于PEM编码的X509证书颁发机构列表的证书
func (gServer *GRPCServer) SetClientRootCAs(clientRoots [][]byte) error {
	gServer.lock.Lock()
	defer gServer.lock.Unlock()

	errMsg := "Failed to set client root certificate(s): %s"

//创建新映射和证书池
	clientRootCAs := make(map[string]*x509.Certificate)
	for _, clientRoot := range clientRoots {
		certs, subjects, err := pemToX509Certs(clientRoot)
		if err != nil {
			return fmt.Errorf(errMsg, err.Error())
		}
		if len(certs) >= 1 {
			for i, cert := range certs {
//将它添加到我们的客户端rootcas映射中，使用subject作为键
				clientRootCAs[subjects[i]] = cert
			}
		}
	}

//创建新的证书池并用新的clientrootcas填充
	certPool := x509.NewCertPool()
	for _, clientRoot := range clientRootCAs {
		certPool.AddCert(clientRoot)
	}
//替换内部映射
	gServer.clientRootCAs = clientRootCAs
//替换当前的clientcas池
	gServer.tlsConfig.ClientCAs = certPool
	return nil
}
