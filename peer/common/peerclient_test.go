
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

package common_test

import (
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hyperledger/fabric/peer/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func initPeerTestEnv(t *testing.T) (cleanup func()) {
	t.Helper()
	cfgPath := "./testdata"
	os.Setenv("FABRIC_CFG_PATH", cfgPath)
	viper.Reset()
	_ = common.InitConfig("test")

	return func() {
		err := os.Unsetenv("FABRIC_CFG_PATH")
		assert.NoError(t, err)
		viper.Reset()
	}
}

func TestNewPeerClientFromEnv(t *testing.T) {
	cleanup := initPeerTestEnv(t)
	defer cleanup()

	pClient, err := common.NewPeerClientFromEnv()
	assert.NoError(t, err)
	assert.NotNil(t, pClient)

	viper.Set("peer.tls.enabled", true)
	pClient, err = common.NewPeerClientFromEnv()
	assert.NoError(t, err)
	assert.NotNil(t, pClient)

	viper.Set("peer.tls.enabled", true)
	viper.Set("peer.tls.clientAuthRequired", true)
	pClient, err = common.NewPeerClientFromEnv()
	assert.NoError(t, err)
	assert.NotNil(t, pClient)

//坏密钥文件
	badKeyFile := filepath.Join("certs", "bad.key")
	viper.Set("peer.tls.clientKey.file", badKeyFile)
	pClient, err = common.NewPeerClientFromEnv()
	assert.Contains(t, err.Error(), "failed to create PeerClient from config")
	assert.Nil(t, pClient)

//错误的证书文件路径
	viper.Set("peer.tls.clientCert.file", "./nocert.crt")
	pClient, err = common.NewPeerClientFromEnv()
	assert.Contains(t, err.Error(), "unable to load peer.tls.clientCert.file")
	assert.Contains(t, err.Error(), "failed to load config for PeerClient")
	assert.Nil(t, pClient)

//错误的密钥文件路径
	viper.Set("peer.tls.clientKey.file", "./nokey.key")
	pClient, err = common.NewPeerClientFromEnv()
	assert.Contains(t, err.Error(), "unable to load peer.tls.clientKey.file")
	assert.Nil(t, pClient)

//坏CA路径
	viper.Set("peer.tls.rootcert.file", "noroot.crt")
	pClient, err = common.NewPeerClientFromEnv()
	assert.Contains(t, err.Error(), "unable to load peer.tls.rootcert.file")
	assert.Nil(t, pClient)
}

func TestPeerClient(t *testing.T) {
	cleanup := initPeerTestEnv(t)
	defer cleanup()

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("error creating server for test: %v", err)
	}
	defer lis.Close()
	viper.Set("peer.address", lis.Addr().String())
	pClient1, err := common.NewPeerClientFromEnv()
	if err != nil {
		t.Fatalf("failed to create PeerClient for test: %v", err)
	}
	eClient, err := pClient1.Endorser()
	assert.NoError(t, err)
	assert.NotNil(t, eClient)
	eClient, err = common.GetEndorserClient("", "")
	assert.NoError(t, err)
	assert.NotNil(t, eClient)

	aClient, err := pClient1.Admin()
	assert.NoError(t, err)
	assert.NotNil(t, aClient)
	aClient, err = common.GetAdminClient()
	assert.NoError(t, err)
	assert.NotNil(t, aClient)

	dClient, err := pClient1.Deliver()
	assert.NoError(t, err)
	assert.NotNil(t, dClient)
	dClient, err = common.GetDeliverClient("", "")
	assert.NoError(t, err)
	assert.NotNil(t, dClient)
}

func TestPeerClientTimeout(t *testing.T) {
	t.Run("PeerClient.GetEndorser() timeout", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		viper.Set("peer.client.connTimeout", 10*time.Millisecond)
		defer cleanup()
		pClient, err := common.NewPeerClientFromEnv()
		if err != nil {
			t.Fatalf("failed to create PeerClient for test: %v", err)
		}
		_, err = pClient.Endorser()
		assert.Contains(t, err.Error(), "endorser client failed to connect")
	})
	t.Run("GetEndorserClient() timeout", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		viper.Set("peer.client.connTimeout", 10*time.Millisecond)
		defer cleanup()
		_, err := common.GetEndorserClient("", "")
		assert.Contains(t, err.Error(), "endorser client failed to connect")
	})
	t.Run("PeerClient.GetAdmin() timeout", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		viper.Set("peer.client.connTimeout", 10*time.Millisecond)
		defer cleanup()
		pClient, err := common.NewPeerClientFromEnv()
		if err != nil {
			t.Fatalf("failed to create PeerClient for test: %v", err)
		}
		_, err = pClient.Admin()
		assert.Contains(t, err.Error(), "admin client failed to connect")
	})
	t.Run("GetAdminClient() timeout", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		viper.Set("peer.client.connTimeout", 10*time.Millisecond)
		defer cleanup()
		_, err := common.GetAdminClient()
		assert.Contains(t, err.Error(), "admin client failed to connect")
	})
	t.Run("PeerClient.Deliver() timeout", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		viper.Set("peer.client.connTimeout", 10*time.Millisecond)
		defer cleanup()
		pClient, err := common.NewPeerClientFromEnv()
		if err != nil {
			t.Fatalf("failed to create PeerClient for test: %v", err)
		}
		_, err = pClient.Deliver()
		assert.Contains(t, err.Error(), "deliver client failed to connect")
	})
	t.Run("GetDeliverClient() timeout", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		viper.Set("peer.client.connTimeout", 10*time.Millisecond)
		defer cleanup()
		_, err := common.GetDeliverClient("", "")
		assert.Contains(t, err.Error(), "deliver client failed to connect")
	})
	t.Run("PeerClient.Certificate()", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		defer cleanup()
		pClient, err := common.NewPeerClientFromEnv()
		if err != nil {
			t.Fatalf("failed to create PeerClient for test: %v", err)
		}
		cert := pClient.Certificate()
		assert.NotNil(t, cert)
	})
	t.Run("GetCertificate()", func(t *testing.T) {
		cleanup := initPeerTestEnv(t)
		defer cleanup()
		cert, err := common.GetCertificate()
		assert.NotEqual(t, cert, &tls.Certificate{})
		assert.NoError(t, err)
	})
}

func TestNewPeerClientForAddress(t *testing.T) {
	cleanup := initPeerTestEnv(t)
	defer cleanup()

//禁用TLS
	viper.Set("peer.tls.enabled", false)

//成功案例
	pClient, err := common.NewPeerClientForAddress("testPeer", "")
	assert.NoError(t, err)
	assert.NotNil(t, pClient)

//失败-未提供对等地址
	pClient, err = common.NewPeerClientForAddress("", "")
	assert.Contains(t, err.Error(), "peer address must be set")
	assert.Nil(t, pClient)

//启用TLS
	viper.Set("peer.tls.enabled", true)

//成功案例
	pClient, err = common.NewPeerClientForAddress("tlsPeer", "./testdata/certs/ca.crt")
	assert.NoError(t, err)
	assert.NotNil(t, pClient)

//失败-错误的TLS根证书文件
	pClient, err = common.NewPeerClientForAddress("badPeer", "bad.crt")
	assert.Contains(t, err.Error(), "unable to load TLS root cert file from bad.crt")
	assert.Nil(t, pClient)

//失败-空的TLS根证书文件
	pClient, err = common.NewPeerClientForAddress("badPeer", "")
	assert.Contains(t, err.Error(), "tls root cert file must be set")
	assert.Nil(t, pClient)
}

func TestGetClients_AddressError(t *testing.T) {
	cleanup := initPeerTestEnv(t)
	defer cleanup()

	viper.Set("peer.tls.enabled", true)

//失败
	eClient, err := common.GetEndorserClient("peer0", "")
	assert.Contains(t, err.Error(), "tls root cert file must be set")
	assert.Nil(t, eClient)

	dClient, err := common.GetDeliverClient("peer0", "")
	assert.Contains(t, err.Error(), "tls root cert file must be set")
	assert.Nil(t, dClient)
}
