
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

package peer

import (
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestCacheConfigurationNegative(t *testing.T) {

//设置错误的peer.address
	viper.Set("peer.addressAutoDetect", true)
	viper.Set("peer.address", "testing.com")
	cacheConfiguration()
	err := CacheConfiguration()
	assert.Error(t, err, "Expected error for bad configuration")
}

func TestConfiguration(t *testing.T) {

	var ips []string
//获取接口地址
	if addresses, err := net.InterfaceAddrs(); err == nil {
		for _, address := range addresses {
//消除环回接口
			if ip, ok := address.(*net.IPNet); ok && !ip.IP.IsLoopback() {
				ips = append(ips, ip.IP.String()+":7051")
				t.Logf("found interface address [%s]", ip.IP.String())
			}
		}
	} else {
		t.Fatal("Failed to get interface addresses")
	}

	var tests = []struct {
		name             string
		settings         map[string]interface{}
		validAddresses   []string
		invalidAddresses []string
	}{
		{
			name: "test1",
			settings: map[string]interface{}{
				"peer.addressAutoDetect": false,
				"peer.address":           "testing.com:7051",
				"peer.id":                "testPeer",
			},
			validAddresses:   []string{"testing.com:7051"},
			invalidAddresses: ips,
		},
		{
			name: "test2",
			settings: map[string]interface{}{
				"peer.addressAutoDetect": true,
				"peer.address":           "testing.com:7051",
				"peer.id":                "testPeer",
			},
			validAddresses:   ips,
			invalidAddresses: []string{"testing.com:7051"},
		},
		{
			name: "test3",
			settings: map[string]interface{}{
				"peer.addressAutoDetect": false,
				"peer.address":           "0.0.0.0:7051",
				"peer.id":                "testPeer",
			},
			validAddresses:   []string{fmt.Sprintf("%s:7051", GetLocalIP())},
			invalidAddresses: []string{"0.0.0.0:7051"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.settings {
				viper.Set(k, v)
			}
//重置缓存
			configurationCached = false
//获取本地地址
			address, err := GetLocalAddress()
			assert.NoError(t, err, "GetLocalAddress returned unexpected error")
			assert.Contains(t, test.validAddresses, address,
				"GetLocalAddress returned unexpected address")
			assert.NotContains(t, test.invalidAddresses, address,
				"GetLocalAddress returned invalid address")
//重置缓存
			configurationCached = false
//GETPER端点
			pe, err := GetPeerEndpoint()
			assert.NoError(t, err, "GetPeerEndpoint returned unexpected error")
			assert.Equal(t, test.settings["peer.id"], pe.Id.Name,
				"GetPeerEndpoint returned the wrong peer ID")
			assert.Equal(t, address, pe.Address,
				"GetPeerEndpoint returned the wrong peer address")

//现在检查缓存配置
			err = CacheConfiguration()
			assert.NoError(t, err, "CacheConfiguration should not have returned an err")
//再次检查功能
//获取本地地址
			address, err = GetLocalAddress()
			assert.NoError(t, err, "GetLocalAddress should not have returned error")
			assert.Contains(t, test.validAddresses, address,
				"GetLocalAddress returned unexpected address")
			assert.NotContains(t, test.invalidAddresses, address,
				"GetLocalAddress returned invalid address")
//GETPER端点
			pe, err = GetPeerEndpoint()
			assert.NoError(t, err, "GetPeerEndpoint returned unexpected error")
			assert.Equal(t, test.settings["peer.id"], pe.Id.Name,
				"GetPeerEndpoint returned the wrong peer ID")
			assert.Equal(t, address, pe.Address,
				"GetPeerEndpoint returned the wrong peer address")
		})
	}
}

func TestGetServerConfig(t *testing.T) {

//没有TLS的良好配置
	viper.Set("peer.tls.enabled", false)
	sc, _ := GetServerConfig()
	assert.Equal(t, false, sc.SecOpts.UseTLS,
		"ServerConfig.SecOpts.UseTLS should be false")

//保留选项
	assert.Equal(t, comm.DefaultKeepaliveOptions, sc.KaOpts,
		"ServerConfig.KaOpts should be set to default values")
	viper.Set("peer.keepalive.minInterval", "2m")
	sc, _ = GetServerConfig()
	assert.Equal(t, time.Duration(2)*time.Minute, sc.KaOpts.ServerMinInterval,
		"ServerConfig.KaOpts.ServerMinInterval should be set to 2 min")

//TLS配置良好
	viper.Set("peer.tls.enabled", true)
	viper.Set("peer.tls.cert.file", filepath.Join("testdata", "Org1-server1-cert.pem"))
	viper.Set("peer.tls.key.file", filepath.Join("testdata", "Org1-server1-key.pem"))
	viper.Set("peer.tls.rootcert.file", filepath.Join("testdata", "Org1-cert.pem"))
	sc, _ = GetServerConfig()
	assert.Equal(t, true, sc.SecOpts.UseTLS, "ServerConfig.SecOpts.UseTLS should be true")
	assert.Equal(t, false, sc.SecOpts.RequireClientCert,
		"ServerConfig.SecOpts.RequireClientCert should be false")
	viper.Set("peer.tls.clientAuthRequired", true)
	viper.Set("peer.tls.clientRootCAs.files",
		[]string{filepath.Join("testdata", "Org1-cert.pem"),
			filepath.Join("testdata", "Org2-cert.pem")})
	sc, _ = GetServerConfig()
	assert.Equal(t, true, sc.SecOpts.RequireClientCert,
		"ServerConfig.SecOpts.RequireClientCert should be true")
	assert.Equal(t, 2, len(sc.SecOpts.ClientRootCAs),
		"ServerConfig.SecOpts.ClientRootCAs should contain 2 entries")

//TLS配置错误
	viper.Set("peer.tls.rootcert.file", filepath.Join("testdata", "Org11-cert.pem"))
	_, err := GetServerConfig()
	assert.Error(t, err, "GetServerConfig should return error with bad root cert path")
	viper.Set("peer.tls.cert.file", filepath.Join("testdata", "Org11-cert.pem"))
	_, err = GetServerConfig()
	assert.Error(t, err, "GetServerConfig should return error with bad tls cert path")

//禁用剩余测试的TLS
	viper.Set("peer.tls.enabled", false)
	viper.Set("peer.tls.clientAuthRequired", false)

}

func TestGetClientCertificate(t *testing.T) {
	viper.Set("peer.tls.key.file", "")
	viper.Set("peer.tls.cert.file", "")
	viper.Set("peer.tls.clientKey.file", "")
	viper.Set("peer.tls.clientCert.file", "")

//客户端和服务器密钥对均未设置-预期错误
	_, err := GetClientCertificate()
	assert.Error(t, err)

	viper.Set("peer.tls.key.file", "")
	viper.Set("peer.tls.cert.file",
		filepath.Join("testdata", "Org1-server1-cert.pem"))
//缺少服务器密钥文件-预期错误
	_, err = GetClientCertificate()
	assert.Error(t, err)

	viper.Set("peer.tls.key.file",
		filepath.Join("testdata", "Org1-server1-key.pem"))
	viper.Set("peer.tls.cert.file", "")
//缺少服务器证书文件-预期错误
	_, err = GetClientCertificate()
	assert.Error(t, err)

//设置服务器TLS设置以确保获得客户端TLS设置
//当它们设置正确时
	viper.Set("peer.tls.key.file",
		filepath.Join("testdata", "Org1-server1-key.pem"))
	viper.Set("peer.tls.cert.file",
		filepath.Join("testdata", "Org1-server1-cert.pem"))

//peer.tls.clientcert.file未设置-预期错误
	viper.Set("peer.tls.clientKey.file",
		filepath.Join("testdata", "Org2-server1-key.pem"))
	_, err = GetClientCertificate()
	assert.Error(t, err)

//peer.tls.clientkey.file未设置-预期错误
	viper.Set("peer.tls.clientKey.file", "")
	viper.Set("peer.tls.clientCert.file",
		filepath.Join("testdata", "Org2-server1-cert.pem"))
	_, err = GetClientCertificate()
	assert.Error(t, err)

//需要客户端身份验证，并且设置了clientkey/clientcert
	expected, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "Org2-server1-cert.pem"),
		filepath.Join("testdata", "Org2-server1-key.pem"))
	if err != nil {
		t.Fatalf("Failed to load test certificate (%s)", err)
	}
	viper.Set("peer.tls.clientKey.file",
		filepath.Join("testdata", "Org2-server1-key.pem"))
	cert, err := GetClientCertificate()
	assert.NoError(t, err)
	assert.Equal(t, expected, cert)

//需要客户端身份验证，但未设置clientkey/clientcert-预期
//客户端证书为服务器证书
	viper.Set("peer.tls.clientKey.file", "")
	viper.Set("peer.tls.clientCert.file", "")
	expected, err = tls.LoadX509KeyPair(
		filepath.Join("testdata", "Org1-server1-cert.pem"),
		filepath.Join("testdata", "Org1-server1-key.pem"))
	if err != nil {
		t.Fatalf("Failed to load test certificate (%s)", err)
	}
	cert, err = GetClientCertificate()
	assert.NoError(t, err)
	assert.Equal(t, expected, cert)
}
