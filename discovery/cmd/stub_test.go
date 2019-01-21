
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


package discovery

import (
	"fmt"
	"net"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/cmd/common"
	"github.com/hyperledger/fabric/cmd/common/comm"
	"github.com/hyperledger/fabric/cmd/common/signer"
	c "github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/discovery/client"
	"github.com/stretchr/testify/assert"
)

func TestClientStub(t *testing.T) {
	srv, err := c.NewGRPCServer("127.0.0.1:", c.ServerConfig{
		SecOpts: &c.SecureOptions{},
	})
	assert.NoError(t, err)
	go srv.Start()
	defer srv.Stop()

	_, portStr, _ := net.SplitHostPort(srv.Address())
	endpoint := fmt.Sprintf("localhost:%s", portStr)
	stub := &ClientStub{}

	req := discovery.NewRequest()

	_, err = stub.Send(endpoint, common.Config{
		SignerConfig: signer.Config{
			MSPID:        "Org1MSP",
			KeyPath:      filepath.Join("testdata", "8150cb2d09628ccc89727611ebb736189f6482747eff9b8aaaa27e9a382d2e93_sk"),
			IdentityPath: filepath.Join("testdata", "cert.pem"),
		},
		TLSConfig: comm.Config{},
	}, req)
	assert.Contains(t, err.Error(), "Unimplemented desc = unknown service discovery.Discovery")
}

func TestRawStub(t *testing.T) {
	srv, err := c.NewGRPCServer("127.0.0.1:", c.ServerConfig{
		SecOpts: &c.SecureOptions{},
	})
	assert.NoError(t, err)
	go srv.Start()
	defer srv.Stop()

	_, portStr, _ := net.SplitHostPort(srv.Address())
	endpoint := fmt.Sprintf("localhost:%s", portStr)
	stub := &RawStub{}

	req := discovery.NewRequest()

	_, err = stub.Send(endpoint, common.Config{
		SignerConfig: signer.Config{
			MSPID:        "Org1MSP",
			KeyPath:      filepath.Join("testdata", "8150cb2d09628ccc89727611ebb736189f6482747eff9b8aaaa27e9a382d2e93_sk"),
			IdentityPath: filepath.Join("testdata", "cert.pem"),
		},
		TLSConfig: comm.Config{},
	}, req)
	assert.Contains(t, err.Error(), "Unimplemented desc = unknown service discovery.Discovery")
}
