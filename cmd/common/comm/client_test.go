
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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/stretchr/testify/assert"
)

func TestTLSClient(t *testing.T) {
	srv, err := comm.NewGRPCServer("127.0.0.1:", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{
			UseTLS:      true,
			Key:         loadFileOrDie(filepath.Join("testdata", "server", "key.pem")),
			Certificate: loadFileOrDie(filepath.Join("testdata", "server", "cert.pem")),
		},
	})
	assert.NoError(t, err)
	go srv.Start()
	defer srv.Stop()
	conf := Config{}
	cl, err := NewClient(conf)
	assert.NoError(t, err)
	_, port, _ := net.SplitHostPort(srv.Address())
	dial := cl.NewDialer(net.JoinHostPort("localhost", port))
	conn, err := dial()
	assert.NoError(t, err)
	conn.Close()
}

func TestDialBadEndpoint(t *testing.T) {
	conf := Config{
		PeerCACertPath: filepath.Join("testdata", "server", "ca.pem"),
		Timeout:        100 * time.Millisecond,
	}
	cl, err := NewClient(conf)
	assert.NoError(t, err)
	dial := cl.NewDialer("non_existent_host.xyz.blabla:9999")
	_, err = dial()
	assert.Error(t, err)
}

func TestNonTLSClient(t *testing.T) {
	srv, err := comm.NewGRPCServer("127.0.0.1:", comm.ServerConfig{
		SecOpts: &comm.SecureOptions{},
	})
	assert.NoError(t, err)
	go srv.Start()
	defer srv.Stop()
	conf := Config{}
	cl, err := NewClient(conf)
	assert.NoError(t, err)
	_, port, _ := net.SplitHostPort(srv.Address())
	dial := cl.NewDialer(fmt.Sprintf("localhost:%s", port))
	conn, err := dial()
	assert.NoError(t, err)
	conn.Close()
}

func TestClientBadConfig(t *testing.T) {
	conf := Config{
		PeerCACertPath: filepath.Join("testdata", "server", "non_existent_file"),
	}
	cl, err := NewClient(conf)
	assert.Nil(t, cl)
	assert.Contains(t, err.Error(), "open testdata/server/non_existent_file: no such file or directory")

	conf = Config{
		PeerCACertPath: filepath.Join("testdata", "server", "ca.pem"),
		KeyPath:        "non_existent_file",
		CertPath:       "non_existent_file",
	}
	cl, err = NewClient(conf)
	assert.Nil(t, cl)
	assert.Contains(t, err.Error(), "open non_existent_file: no such file or directory")

	conf = Config{
		PeerCACertPath: filepath.Join("testdata", "server", "ca.pem"),
		KeyPath:        filepath.Join("testdata", "client", "key.pem"),
		CertPath:       "non_existent_file",
	}
	cl, err = NewClient(conf)
	assert.Nil(t, cl)
	assert.Contains(t, err.Error(), "open non_existent_file: no such file or directory")
}

func loadFileOrDie(path string) []byte {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Failed opening file", path, ":", err)
		os.Exit(1)
	}
	return b
}
