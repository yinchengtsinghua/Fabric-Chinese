
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
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

//配置默认值
var (
//GRPC客户端和服务器的最大发送和接收字节数
	MaxRecvMsgSize = 100 * 1024 * 1024
	MaxSendMsgSize = 100 * 1024 * 1024
//默认对等保留选项
	DefaultKeepaliveOptions = &KeepaliveOptions{
ClientInterval:    time.Duration(1) * time.Minute,  //1分钟
ClientTimeout:     time.Duration(20) * time.Second, //20秒-GRPC默认值
ServerInterval:    time.Duration(2) * time.Hour,    //2小时-GRPC默认值
ServerTimeout:     time.Duration(20) * time.Second, //20秒-GRPC默认值
ServerMinInterval: time.Duration(1) * time.Minute,  //匹配客户端间隔
	}
//强TLS密码套件
	DefaultTLSCipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	}
//默认连接超时
	DefaultConnectionTimeout = 5 * time.Second
)

//serverconfig定义用于配置grpcserver实例的参数
type ServerConfig struct {
//connectionTimeout指定建立连接的超时
//对于所有新连接
	ConnectionTimeout time.Duration
//secopts定义安全参数
	SecOpts *SecureOptions
//kaopts定义了keepalive参数
	KaOpts *KeepaliveOptions
//流拦截器指定要应用于的拦截器列表
//流式RPC。它们是按顺序执行的。
	StreamInterceptors []grpc.StreamServerInterceptor
//一元拦截器指定要应用于一元的拦截器列表
//RPCs。它们是按顺序执行的。
	UnaryInterceptors []grpc.UnaryServerInterceptor
//logger指定服务器将使用的记录器
	Logger *flogging.FabricLogger
//度量提供程序
	MetricsProvider metrics.Provider
}

//clientconfig定义用于配置grpcclient实例的参数
type ClientConfig struct {
//secopts定义安全参数
	SecOpts *SecureOptions
//kaopts定义了keepalive参数
	KaOpts *KeepaliveOptions
//Timeout指定客户端在尝试阻止时阻止的时间
//建立连接
	Timeout time.Duration
//AsyncConnect使连接创建无阻塞
	AsyncConnect bool
}

//SecureOptions定义
//grpcserver或grpcclient实例
type SecureOptions struct {
//verifycertificate，如果不是nil，则在正常之后调用
//由TLS客户端或服务器进行证书验证。
//如果返回非零错误，则终止握手并导致该错误。
	VerifyCertificate func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error
//用于TLS通信的PEM编码的X509公钥
	Certificate []byte
//用于TLS通信的PEM编码私钥
	Key []byte
//客户端使用的一组PEM编码的X509证书颁发机构
//验证服务器证书
	ServerRootCAs [][]byte
//服务器使用的一组PEM编码的X509证书颁发机构
//验证客户端证书
	ClientRootCAs [][]byte
//是否使用TLS进行通信
	UseTLS bool
//TLS客户端是否必须提供用于身份验证的证书
	RequireClientCert bool
//CipherSuites是受支持的TLS密码套件列表
	CipherSuites []uint16
}

//keepaliveoptions用于为两个设置grpc keepalive设置
//客户端和服务器
type KeepaliveOptions struct {
//clientInterval是当客户端没有看到
//它对服务器执行ping操作以查看服务器是否处于活动状态的服务器上的任何活动
	ClientInterval time.Duration
//clientTimeout是客户端等待响应的持续时间
//在关闭连接前发送ping之后从服务器发送
	ClientTimeout time.Duration
//ServerInterval是如果服务器没有看到
//它从客户机ping客户机以查看其是否活动的任何活动
	ServerInterval time.Duration
//ServerTimeout是服务器等待响应的持续时间
//在关闭连接前发送ping之后从客户端发送
	ServerTimeout time.Duration
//ServerMinInterval是客户端Ping之间允许的最短时间。
//如果客户机更频繁地发送ping，服务器将断开它们的连接。
	ServerMinInterval time.Duration
}

type Metrics struct {
//OpenConncounter跟踪打开的连接数
	OpenConnCounter metrics.Counter
//closedcanncounter跟踪关闭的连接数
	ClosedConnCounter metrics.Counter
}

//server keepalive options返回服务器的grpc keepalive选项。如果
//opts为nil，返回默认的keepalive选项
func ServerKeepaliveOptions(ka *KeepaliveOptions) []grpc.ServerOption {
//如果为空，则使用默认的保留选项
	if ka == nil {
		ka = DefaultKeepaliveOptions
	}
	var serverOpts []grpc.ServerOption
	kap := keepalive.ServerParameters{
		Time:    ka.ServerInterval,
		Timeout: ka.ServerTimeout,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(kap))
	kep := keepalive.EnforcementPolicy{
		MinTime: ka.ServerMinInterval,
//允许不带RPC的Keepalive
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))
	return serverOpts
}

//clientkeepaliveoptions为客户端返回grpc keepalive选项。如果
//opts为nil，返回默认的keepalive选项
func ClientKeepaliveOptions(ka *KeepaliveOptions) []grpc.DialOption {
//如果为空，则使用默认的保留选项
	if ka == nil {
		ka = DefaultKeepaliveOptions
	}

	var dialOpts []grpc.DialOption
	kap := keepalive.ClientParameters{
		Time:                ka.ClientInterval,
		Timeout:             ka.ClientTimeout,
		PermitWithoutStream: true,
	}
	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(kap))
	return dialOpts
}
