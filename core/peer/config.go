
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


//配置处理的“viper”包非常灵活，但是
//当配置值为
//重复访问。此处定义的函数cacheconfiguration（）将缓存
//经常访问的所有配置值。这些参数
//现在显示为访问本地配置的函数调用
//变量。这似乎是最有力的方法来表示这些
//面对配置文件的多种方式的参数
//加载和使用（例如，正常使用与测试用例）。

//允许全局调用cacheconfiguration（）函数
//确保始终缓存正确的值；请参见
//某些参数在main.go的“chaincodedevmode”中被强制使用。

package peer

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/config"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//配置是否已缓存？
var configurationCached = false

//计算常量getLocalAddress（）的缓存值和错误值，
//getValidatorStreamAddress（）和getPeerEndpoint（）。
var localAddress string
var localAddressError error
var peerEndpoint *pb.PeerEndpoint
var peerEndpointError error

//常用配置常量的缓存值。

//cacheconfiguration计算并缓存常用常量和
//计算常量作为包变量。以前的程序
//这里嵌入了全局以保留原始抽象。
func CacheConfiguration() (err error) {
//getlocaladdress返回本地对等机操作的地址：端口。受env影响：peer.addressautodetect
	getLocalAddress := func() (string, error) {
		peerAddress := viper.GetString("peer.address")
		if peerAddress == "" {
			return "", fmt.Errorf("peer.address isn't set")
		}
		host, port, err := net.SplitHostPort(peerAddress)
		if err != nil {
			return "", errors.Errorf("peer.address isn't in host:port format: %s", peerAddress)
		}

		autoDetectedIPAndPort := net.JoinHostPort(GetLocalIP(), port)
		peerLogger.Info("Auto-detected peer address:", autoDetectedIPAndPort)
//如果主机是IPv4地址“0.0.0.0”或IPv6地址“：”，
//然后回退到自动检测到的地址
		if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
			peerLogger.Info("Host is", host, ", falling back to auto-detected address:", autoDetectedIPAndPort)
			return autoDetectedIPAndPort, nil
		}

		if viper.GetBool("peer.addressAutoDetect") {
			peerLogger.Info("Auto-detect flag is set, returning", autoDetectedIPAndPort)
			return autoDetectedIPAndPort, nil
		}
		peerLogger.Info("Returning", peerAddress)
		return peerAddress, nil

	}

//GetPeerEndpoint返回此对等实例的对等端点。受env影响：peer.addressautodetect
	getPeerEndpoint := func() (*pb.PeerEndpoint, error) {
		var peerAddress string
		peerAddress, err := getLocalAddress()
		if err != nil {
			return nil, err
		}
		return &pb.PeerEndpoint{Id: &pb.PeerID{Name: viper.GetString("peer.id")}, Address: peerAddress}, nil
	}

	localAddress, localAddressError = getLocalAddress()
	peerEndpoint, _ = getPeerEndpoint()

	configurationCached = true

	if localAddressError != nil {
		return localAddressError
	}
	return
}

//如果错误检查失败，cacheconfiguration会记录一个错误。
func cacheConfiguration() {
	if err := CacheConfiguration(); err != nil {
		peerLogger.Errorf("Execution continues after CacheConfiguration() failure : %s", err)
	}
}

//功能形式

//GetLocalAddress返回peer.address属性
func GetLocalAddress() (string, error) {
	if !configurationCached {
		cacheConfiguration()
	}
	return localAddress, localAddressError
}

//GetPeerEndpoint从缓存配置返回PeerEndpoint
func GetPeerEndpoint() (*pb.PeerEndpoint, error) {
	if !configurationCached {
		cacheConfiguration()
	}
	return peerEndpoint, peerEndpointError
}

//GetServerConfig返回对等服务器的GRPC服务器配置
func GetServerConfig() (comm.ServerConfig, error) {
	secureOptions := &comm.SecureOptions{
		UseTLS: viper.GetBool("peer.tls.enabled"),
	}
	serverConfig := comm.ServerConfig{SecOpts: secureOptions}
	if secureOptions.UseTLS {
//从文件系统获取证书
		serverKey, err := ioutil.ReadFile(config.GetPath("peer.tls.key.file"))
		if err != nil {
			return serverConfig, fmt.Errorf("error loading TLS key (%s)", err)
		}
		serverCert, err := ioutil.ReadFile(config.GetPath("peer.tls.cert.file"))
		if err != nil {
			return serverConfig, fmt.Errorf("error loading TLS certificate (%s)", err)
		}
		secureOptions.Certificate = serverCert
		secureOptions.Key = serverKey
		secureOptions.RequireClientCert = viper.GetBool("peer.tls.clientAuthRequired")
		if secureOptions.RequireClientCert {
			var clientRoots [][]byte
			for _, file := range viper.GetStringSlice("peer.tls.clientRootCAs.files") {
				clientRoot, err := ioutil.ReadFile(
					config.TranslatePath(filepath.Dir(viper.ConfigFileUsed()), file))
				if err != nil {
					return serverConfig,
						fmt.Errorf("error loading client root CAs (%s)", err)
				}
				clientRoots = append(clientRoots, clientRoot)
			}
			secureOptions.ClientRootCAs = clientRoots
		}
//检查根证书
		if config.GetPath("peer.tls.rootcert.file") != "" {
			rootCert, err := ioutil.ReadFile(config.GetPath("peer.tls.rootcert.file"))
			if err != nil {
				return serverConfig, fmt.Errorf("error loading TLS root certificate (%s)", err)
			}
			secureOptions.ServerRootCAs = [][]byte{rootCert}
		}
	}
//获取默认的keepalive选项
	serverConfig.KaOpts = comm.DefaultKeepaliveOptions
//检查是否为env设置了mininterval
	if viper.IsSet("peer.keepalive.minInterval") {
		serverConfig.KaOpts.ServerMinInterval = viper.GetDuration("peer.keepalive.minInterval")
	}
	return serverConfig, nil
}

//getclientCertificate返回要用于GRPC客户端的TLS证书
//连接
func GetClientCertificate() (tls.Certificate, error) {
	cert := tls.Certificate{}

	keyPath := viper.GetString("peer.tls.clientKey.file")
	certPath := viper.GetString("peer.tls.clientCert.file")

	if keyPath != "" || certPath != "" {
//需要同时设置keypath和certpath
		if keyPath == "" || certPath == "" {
			return cert, errors.New("peer.tls.clientKey.file and " +
				"peer.tls.clientCert.file must both be set or must both be empty")
		}
		keyPath = config.GetPath("peer.tls.clientKey.file")
		certPath = config.GetPath("peer.tls.clientCert.file")

	} else {
//使用TLS服务器密钥对
		keyPath = viper.GetString("peer.tls.key.file")
		certPath = viper.GetString("peer.tls.cert.file")

		if keyPath != "" || certPath != "" {
//需要同时设置keypath和certpath
			if keyPath == "" || certPath == "" {
				return cert, errors.New("peer.tls.key.file and " +
					"peer.tls.cert.file must both be set or must both be empty")
			}
			keyPath = config.GetPath("peer.tls.key.file")
			certPath = config.GetPath("peer.tls.cert.file")
		} else {
			return cert, errors.New("must set either " +
				"[peer.tls.key.file and peer.tls.cert.file] or " +
				"[peer.tls.clientKey.file and peer.tls.clientCert.file]" +
				"when peer.tls.clientAuthEnabled is set to true")
		}
	}
//从文件系统获取密钥对
	clientKey, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return cert, errors.WithMessage(err,
			"error loading client TLS key")
	}
	clientCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return cert, errors.WithMessage(err,
			"error loading client TLS certificate")
	}
	cert, err = tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return cert, errors.WithMessage(err,
			"error parsing client TLS key pair")
	}
	return cert, nil
}
