
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
	"context"

	"github.com/hyperledger/fabric/cmd/common"
	"github.com/hyperledger/fabric/cmd/common/comm"
	"github.com/hyperledger/fabric/cmd/common/signer"
	"github.com/hyperledger/fabric/discovery/client"
	. "github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//go:generate mokery-dir../client/-name localresponse-case underline-output mocks/
//go:generate mokery-dir../client/-name channelresponse-case underline-output mocks/
//去：生成mokery-dir。-name serviceresponse-case underline-输出模拟/

//ServiceResponse表示从发现服务发送的响应
type ServiceResponse interface {
//ForChannel返回给定通道上下文中的ChannelResponse
	ForChannel(string) discovery.ChannelResponse

//forlocal返回无通道上下文中的localresponse
	ForLocal() discovery.LocalResponse

//RAW返回来自服务器的原始响应
	Raw() *Response
}

type response struct {
	raw *Response
	discovery.Response
}

func (r *response) Raw() *Response {
	return r.raw
}

//ClientStub是一个与发现服务通信的存根
//using the discovery client implementation
type ClientStub struct {
}

//发送发送请求并接收响应
func (stub *ClientStub) Send(server string, conf common.Config, req *discovery.Request) (ServiceResponse, error) {
	comm, err := comm.NewClient(conf.TLSConfig)
	if err != nil {
		return nil, err
	}
	signer, err := signer.NewSigner(conf.SignerConfig)
	if err != nil {
		return nil, err
	}
	timeout, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	disc := discovery.NewClient(comm.NewDialer(server), signer.Sign, 0)

	resp, err := disc.Send(timeout, req, &AuthInfo{
		ClientIdentity:    signer.Creator,
		ClientTlsCertHash: comm.TLSCertHash,
	})
	if err != nil {
		return nil, errors.Errorf("failed connecting to %s: %v", server, err)
	}
	return &response{
		Response: resp,
	}, nil
}

//rawstub是一个与发现服务通信的存根
//没有任何中间人。
type RawStub struct {
}

//发送发送请求并接收响应
func (stub *RawStub) Send(server string, conf common.Config, req *discovery.Request) (ServiceResponse, error) {
	comm, err := comm.NewClient(conf.TLSConfig)
	if err != nil {
		return nil, err
	}
	signer, err := signer.NewSigner(conf.SignerConfig)
	if err != nil {
		return nil, err
	}
	timeout, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	req.Authentication = &AuthInfo{
		ClientIdentity:    signer.Creator,
		ClientTlsCertHash: comm.TLSCertHash,
	}

	payload := utils.MarshalOrPanic(req.Request)
	sig, err := signer.Sign(payload)
	if err != nil {
		return nil, err
	}

	cc, err := comm.NewDialer(server)()
	if err != nil {
		return nil, err
	}
	resp, err := NewDiscoveryClient(cc).Discover(timeout, &SignedRequest{
		Payload:   payload,
		Signature: sig,
	})

	if err != nil {
		return nil, err
	}

	return &response{
		raw: resp,
	}, nil
}
