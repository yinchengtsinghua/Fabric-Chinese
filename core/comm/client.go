
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
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type GRPCClient struct {
//grpc.clientconn使用的tls配置
	tlsConfig *tls.Config
//设置新连接的选项
	dialOpts []grpc.DialOption
//建立新连接时阻塞的持续时间
	timeout time.Duration
//客户端可以接收的最大消息大小
	maxRecvMsgSize int
//客户端可以发送的最大消息大小
	maxSendMsgSize int
}

//
//
func NewGRPCClient(config ClientConfig) (*GRPCClient, error) {
	client := &GRPCClient{}

//
	err := client.parseSecureOptions(config.SecOpts)
	if err != nil {
		return client, err
	}

//
	var kap keepalive.ClientParameters
	if config.KaOpts != nil {
		kap = keepalive.ClientParameters{
			Time:    config.KaOpts.ClientInterval,
			Timeout: config.KaOpts.ClientTimeout}
	} else {
//
		kap = keepalive.ClientParameters{
			Time:    DefaultKeepaliveOptions.ClientInterval,
			Timeout: DefaultKeepaliveOptions.ClientTimeout}
	}
	kap.PermitWithoutStream = true
//设置保持状态
	client.dialOpts = append(client.dialOpts, grpc.WithKeepaliveParams(kap))
//除非设置了异步连接，否则将连接建立阻塞。
	if !config.AsyncConnect {
		client.dialOpts = append(client.dialOpts, grpc.WithBlock())
	}
	client.timeout = config.Timeout
//将发送/接收消息大小设置为包默认值
	client.maxRecvMsgSize = MaxRecvMsgSize
	client.maxSendMsgSize = MaxSendMsgSize

	return client, nil
}

func (client *GRPCClient) parseSecureOptions(opts *SecureOptions) error {

	if opts == nil || !opts.UseTLS {
		return nil
	}
	client.tlsConfig = &tls.Config{
		VerifyPeerCertificate: opts.VerifyCertificate,
MinVersion:            tls.VersionTLS12} //仅限TLS 1.2
	if len(opts.ServerRootCAs) > 0 {
		client.tlsConfig.RootCAs = x509.NewCertPool()
		for _, certBytes := range opts.ServerRootCAs {
			err := AddPemToCertPool(certBytes, client.tlsConfig.RootCAs)
			if err != nil {
				commLogger.Debugf("error adding root certificate: %v", err)
				return errors.WithMessage(err,
					"error adding root certificate")
			}
		}
	}
	if opts.RequireClientCert {
//确保我们有钥匙和证书
		if opts.Key != nil &&
			opts.Certificate != nil {
			cert, err := tls.X509KeyPair(opts.Certificate,
				opts.Key)
			if err != nil {
				return errors.WithMessage(err, "failed to "+
					"load client certificate")
			}
			client.tlsConfig.Certificates = append(
				client.tlsConfig.Certificates, cert)
		} else {
			return errors.New("both Key and Certificate " +
				"are required when using mutual TLS")
		}
	}
	return nil
}

//证书返回用于建立TLS连接的TLS证书
//当服务器需要客户端证书时
func (client *GRPCClient) Certificate() tls.Certificate {
	cert := tls.Certificate{}
	if client.tlsConfig != nil && len(client.tlsConfig.Certificates) > 0 {
		cert = client.tlsConfig.Certificates[0]
	}
	return cert
}

//tlsenabled是一个标志，指示是否对客户端使用tls
//连接
func (client *GRPCClient) TLSEnabled() bool {
	return client.tlsConfig != nil
}

//mutualTlsRequired是一个标志，指示客户端
//进行TLS连接时必须发送证书
func (client *GRPCClient) MutualTLSRequired() bool {
	return client.tlsConfig != nil &&
		len(client.tlsConfig.Certificates) > 0
}

//setmaxrecvmsgsize设置客户端可以接收的最大消息大小
func (client *GRPCClient) SetMaxRecvMsgSize(size int) {
	client.maxRecvMsgSize = size
}

//setmaxsendmsgsize设置客户端可以发送的最大消息大小
func (client *GRPCClient) SetMaxSendMsgSize(size int) {
	client.maxSendMsgSize = size
}

//setserverrootcas设置用于验证服务器的权限列表
//基于PEM编码的X509证书颁发机构列表的证书
func (client *GRPCClient) SetServerRootCAs(serverRoots [][]byte) error {

//注意：如果未指定ServerRoots，则当前证书池将
//替换为空的
	certPool := x509.NewCertPool()
	for _, root := range serverRoots {
		err := AddPemToCertPool(root, certPool)
		if err != nil {
			return errors.WithMessage(err, "error adding root certificate")
		}
	}
	client.tlsConfig.RootCAs = certPool
	return nil
}

//NewConnection返回目标地址的grpc.clientconn，并
//覆盖用于验证主机名的服务器名
//使用TLS时服务器返回的证书
func (client *GRPCClient) NewConnection(address string, serverNameOverride string) (
	*grpc.ClientConn, error) {

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, client.dialOpts...)

//设置传输凭据和最大发送/接收消息大小
//在创建连接之前立即允许
//设置服务器rootcas/setmaxrecvmsgsize/setmaxsendmsgsize
//按连接生效
	if client.tlsConfig != nil {
		client.tlsConfig.ServerName = serverNameOverride
		dialOpts = append(dialOpts,
			grpc.WithTransportCredentials(
				credentials.NewTLS(client.tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(client.maxRecvMsgSize),
		grpc.MaxCallSendMsgSize(client.maxSendMsgSize)))

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, address, dialOpts...)
	if err != nil {
		return nil, errors.WithMessage(errors.WithStack(err),
			"failed to create new connection")
	}
	return conn, nil
}
