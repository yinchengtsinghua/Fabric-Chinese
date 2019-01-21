
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
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const defaultTimeout = time.Second * 3

var commLogger = flogging.MustGetLogger("comm")
var credSupport *CredentialSupport
var once sync.Once

//casupport类型管理由通道作用域的证书颁发机构
type CASupport struct {
	sync.RWMutex
	AppRootCAsByChain     map[string][][]byte
	OrdererRootCAsByChain map[string][][]byte
	ClientRootCAs         [][]byte
	ServerRootCAs         [][]byte
}

//CredentialSupport类型管理用于GRPC客户端连接的凭据
type CredentialSupport struct {
	*CASupport
	clientCert tls.Certificate
}

//getCredentialSupport返回Singleton CredentialSupport实例
func GetCredentialSupport() *CredentialSupport {

	once.Do(func() {
		credSupport = &CredentialSupport{
			CASupport: &CASupport{
				AppRootCAsByChain:     make(map[string][][]byte),
				OrdererRootCAsByChain: make(map[string][][]byte),
			},
		}
	})
	return credSupport
}

//getserverrootcas返回所有
//为所有链定义的应用程序和订购方组织。根
//返回的证书应用于设置的受信任服务器根
//TLS客户端。
func (cas *CASupport) GetServerRootCAs() (appRootCAs, ordererRootCAs [][]byte) {
	cas.RLock()
	defer cas.RUnlock()

	appRootCAs = [][]byte{}
	ordererRootCAs = [][]byte{}

	for _, appRootCA := range cas.AppRootCAsByChain {
		appRootCAs = append(appRootCAs, appRootCA...)
	}

	for _, ordererRootCA := range cas.OrdererRootCAsByChain {
		ordererRootCAs = append(ordererRootCAs, ordererRootCA...)
	}

//还需要附加静态配置的根证书
	appRootCAs = append(appRootCAs, cas.ServerRootCAs...)
	return appRootCAs, ordererRootCAs
}

//getclientrootcas返回所有
//为所有链定义的应用程序和订购方组织。根
//返回的证书应用于设置的受信任客户端根
//TLS服务器。
func (cas *CASupport) GetClientRootCAs() (appRootCAs, ordererRootCAs [][]byte) {
	cas.RLock()
	defer cas.RUnlock()

	appRootCAs = [][]byte{}
	ordererRootCAs = [][]byte{}

	for _, appRootCA := range cas.AppRootCAsByChain {
		appRootCAs = append(appRootCAs, appRootCA...)
	}

	for _, ordererRootCA := range cas.OrdererRootCAsByChain {
		ordererRootCAs = append(ordererRootCAs, ordererRootCA...)
	}

//还需要附加静态配置的根证书
	appRootCAs = append(appRootCAs, cas.ClientRootCAs...)
	return appRootCAs, ordererRootCAs
}

//setclientcertificate设置要用于GRPC客户端的tls.certificate
//连接
func (cs *CredentialSupport) SetClientCertificate(cert tls.Certificate) {
	cs.clientCert = cert
}

//getclientCertificate返回凭证支持的客户端证书
func (cs *CredentialSupport) GetClientCertificate() tls.Certificate {
	return cs.clientCert
}

//GetDeliverServiceCredentials返回GRPC要使用的给定通道的GRPC传输凭据
//与订购服务端点通信的客户端。
//如果找不到通道，则返回错误。
func (cs *CredentialSupport) GetDeliverServiceCredentials(channelID string) (credentials.TransportCredentials, error) {
	cs.RLock()
	defer cs.RUnlock()

	var creds credentials.TransportCredentials
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cs.clientCert},
	}
	certPool := x509.NewCertPool()

	rootCACerts, exists := cs.OrdererRootCAsByChain[channelID]
	if !exists {
		commLogger.Errorf("Attempted to obtain root CA certs of a non existent channel: %s", channelID)
		return nil, fmt.Errorf("didn't find any root CA certs for channel %s", channelID)
	}

	for _, cert := range rootCACerts {
		block, _ := pem.Decode(cert)
		if block != nil {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				certPool.AddCert(cert)
			} else {
				commLogger.Warningf("Failed to add root cert to credentials (%s)", err)
			}
		} else {
			commLogger.Warning("Failed to add root cert to credentials")
		}
	}
	tlsConfig.RootCAs = certPool
	creds = credentials.NewTLS(tlsConfig)
	return creds, nil
}

//GetPeerCredentials返回GRPC使用的GRPC传输凭据
//与远程对等端点通信的客户端。
func (cs *CredentialSupport) GetPeerCredentials() credentials.TransportCredentials {
	var creds credentials.TransportCredentials
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cs.clientCert},
	}
	certPool := x509.NewCertPool()
//循环访问服务器根CA
	roots, _ := cs.GetServerRootCAs()
	for _, root := range roots {
		err := AddPemToCertPool(root, certPool)
		if err != nil {
			commLogger.Warningf("Failed adding certificates to peer's client TLS trust pool: %s", err)
		}
	}
	tlsConfig.RootCAs = certPool
	creds = credentials.NewTLS(tlsConfig)
	return creds
}

func getEnv(key, def string) string {
	val := os.Getenv(key)
	if len(val) > 0 {
		return val
	} else {
		return def
	}
}

//newclientconnectionwithaddress将新grpc.clientconn返回给给定地址
func NewClientConnectionWithAddress(peerAddress string, block bool, tslEnabled bool,
	creds credentials.TransportCredentials, ka *KeepaliveOptions) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if ka != nil {
		opts = ClientKeepaliveOptions(ka)
	} else {
//设置为默认选项
		opts = ClientKeepaliveOptions(DefaultKeepaliveOptions)
	}

	if tslEnabled {
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	if block {
		opts = append(opts, grpc.WithBlock())
	}
	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(MaxRecvMsgSize),
		grpc.MaxCallSendMsgSize(MaxSendMsgSize),
	))
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, peerAddress, opts...)
	if err != nil {
		return nil, err
	}
	return conn, err
}

func InitTLSForShim(key, certStr string) credentials.TransportCredentials {
	var sn string
	priv, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		commLogger.Panicf("failed decoding private key from base64, string: %s, error: %v", key, err)
	}
	pub, err := base64.StdEncoding.DecodeString(certStr)
	if err != nil {
		commLogger.Panicf("failed decoding public key from base64, string: %s, error: %v", certStr, err)
	}
	cert, err := tls.X509KeyPair(pub, priv)
	if err != nil {
		commLogger.Panicf("failed loading certificate: %v", err)
	}
	b, err := ioutil.ReadFile(config.GetPath("peer.tls.rootcert.file"))
	if err != nil {
		commLogger.Panicf("failed loading root ca cert: %v", err)
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		commLogger.Panicf("failed to append certificates")
	}
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      cp,
		ServerName:   sn,
	})
}
