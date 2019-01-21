
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
	"errors"
	"net"

	"github.com/hyperledger/fabric/common/flogging"
	"google.golang.org/grpc/credentials"
)

var (
	ClientHandshakeNotImplError = errors.New("core/comm: Client handshakes" +
		"are not implemented with serverCreds")
	OverrrideHostnameNotSupportedError = errors.New(
		"core/comm: OverrideServerName is " +
			"not supported")
	MissingServerConfigError = errors.New(
		"core/comm: `serverConfig` cannot be nil")
//alpnrotostr是GRPC的指定应用程序级协议。
	alpnProtoStr = []string{"h2"}
)

//NewServerTransportCredentials返回新的已初始化
//GRPC/凭证.运输凭证
func NewServerTransportCredentials(
	serverConfig *tls.Config,
	logger *flogging.FabricLogger) credentials.TransportCredentials {

//注意：与默认的GRPC/Credentials实现不同，我们没有
//克隆tls.config，它允许我们动态更新
	serverConfig.NextProtos = alpnProtoStr
//覆盖TLS版本并确保其为1.2
	serverConfig.MinVersion = tls.VersionTLS12
	serverConfig.MaxVersion = tls.VersionTLS12
	return &serverCreds{
		serverConfig: serverConfig,
		logger:       logger}
}

//servercreds是grpc/credentials.transportCredentials的实现。
type serverCreds struct {
	serverConfig *tls.Config
	logger       *flogging.FabricLogger
}

//“servercreds”未实现客户端握手。
func (sc *serverCreds) ClientHandshake(context.Context,
	string, net.Conn) (net.Conn, credentials.AuthInfo, error) {
	return nil, nil, ClientHandshakeNotImplError
}

//ServerHandshake does the authentication handshake for servers.
func (sc *serverCreds) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn := tls.Server(rawConn, sc.serverConfig)
	if err := conn.Handshake(); err != nil {
		if sc.logger != nil {
			sc.logger.With("remote address",
				conn.RemoteAddr().String()).Errorf("TLS handshake failed with error %s", err)
		}
		return nil, nil, err
	}
	return conn, credentials.TLSInfo{State: conn.ConnectionState()}, nil
}

//INFO提供此传输凭据的协议信息。
func (sc *serverCreds) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "tls",
		SecurityVersion:  "1.2",
	}
}

//克隆会复制此TransportCredentials。
func (sc *serverCreds) Clone() credentials.TransportCredentials {
	creds := NewServerTransportCredentials(sc.serverConfig, sc.logger)
	return creds
}

//OverrideServerName overrides the server name used to verify the hostname
//从服务器返回的证书。
func (sc *serverCreds) OverrideServerName(string) error {
	return OverrrideHostnameNotSupportedError
}
