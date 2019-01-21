
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
	"time"

	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const defaultTimeout = time.Second * 5

//客户端处理TLS连接
//到发现服务器
type Client struct {
	TLSCertHash []byte
	*comm.GRPCClient
}

//new client根据给定的配置创建新的通信客户端
func NewClient(conf Config) (*Client, error) {
	if conf.Timeout == time.Duration(0) {
		conf.Timeout = defaultTimeout
	}
	sop, err := conf.ToSecureOptions(newSelfSignedTLSCert)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cl, err := comm.NewGRPCClient(comm.ClientConfig{
		SecOpts: sop,
		Timeout: conf.Timeout,
	})
	if err != nil {
		return nil, err
	}
	return &Client{GRPCClient: cl, TLSCertHash: util.ComputeSHA256(sop.Certificate)}, nil
}

//NewDialer从给定的端点创建一个新的拨号程序
func (c *Client) NewDialer(endpoint string) func() (*grpc.ClientConn, error) {
	return func() (*grpc.ClientConn, error) {
		conn, err := c.NewConnection(endpoint, "")
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return conn, nil
	}
}

func newSelfSignedTLSCert() (*tlsgen.CertKeyPair, error) {
	ca, err := tlsgen.NewCA()
	if err != nil {
		return nil, err
	}
	return ca.NewClientCertKeyPair()
}
