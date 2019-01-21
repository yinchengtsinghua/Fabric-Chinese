
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

package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

//go：生成伪造者-o mock/broadcast.go-伪造姓名广播。广播

//broadcast定义将GRPC调用抽象为将事务广播给订购方的接口
type Broadcast interface {
	Send(m *common.Envelope) error
	Recv() (*ab.BroadcastResponse, error)
	CloseSend() error
}

//go：生成伪造者-o mock/order_client.go-forke name orderclient。订单客户机

//orderclient定义用于创建广播的接口
type OrdererClient interface {
//Newbroadcast返回广播
	NewBroadcast(ctx context.Context, opts ...grpc.CallOption) (Broadcast, error)

//证书返回订购方客户端的TLS证书
	Certificate() *tls.Certificate
}

//orderclient实现orderclient接口
type ordererClient struct {
	ordererAddr        string
	serverNameOverride string
	grpcClient         *comm.GRPCClient
	conn               *grpc.ClientConn
}

func NewOrdererClient(config *ClientConfig) (OrdererClient, error) {
	grpcClient, err := createGrpcClient(&config.OrdererCfg, config.TlsEnabled)
	if err != nil {
		err = errors.WithMessage(err, fmt.Sprintf("failed to create a GRPCClient to orderer %s", config.OrdererCfg.Address))
		logger.Errorf("%s", err)
		return nil, err
	}
	conn, err := grpcClient.NewConnection(config.OrdererCfg.Address, config.OrdererCfg.ServerNameOverride)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to connect to orderer %s", config.OrdererCfg.Address))
	}

	return &ordererClient{
		ordererAddr:        config.OrdererCfg.Address,
		serverNameOverride: config.OrdererCfg.ServerNameOverride,
		grpcClient:         grpcClient,
		conn:               conn,
	}, nil
}

//Newbroadcast创建广播
func (oc *ordererClient) NewBroadcast(ctx context.Context, opts ...grpc.CallOption) (Broadcast, error) {
//重用现有连接以创建广播客户端
	broadcast, err := ab.NewAtomicBroadcastClient(oc.conn).Broadcast(ctx)
	if err == nil {
		return broadcast, nil
	}

//现有连接出错，因此创建一个到订购者的新连接
	oc.conn, err = oc.grpcClient.NewConnection(oc.ordererAddr, oc.serverNameOverride)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to connect to orderer %s", oc.ordererAddr))
	}

//创建新广播
	broadcast, err = ab.NewAtomicBroadcastClient(oc.conn).Broadcast(ctx)
	if err != nil {
		rpcStatus, _ := status.FromError(err)
		return nil, errors.Wrapf(err, "failed to new a broadcast, rpcStatus=%+v", rpcStatus)
	}
	return broadcast, nil
}

func (oc *ordererClient) Certificate() *tls.Certificate {
	cert := oc.grpcClient.Certificate()
	return &cert
}

//BroadcastSend向订购方服务发送事务信封
func BroadcastSend(broadcast Broadcast, addr string, envelope *common.Envelope) error {
	err := broadcast.Send(envelope)
	broadcast.CloseSend()

	if err != nil {
		return errors.Wrapf(err, "failed to send transaction to orderer %s", addr)
	}
	return nil
}

//BroadReceive等待，直到它收到广播流的响应
func BroadcastReceive(broadcast Broadcast, addr string, responses chan common.Status, errs chan error) {
	logger.Infof("calling OrdererClient.broadcastReceive")
	for {
		broadcastResponse, err := broadcast.Recv()
		if err == io.EOF {
			close(responses)
			return
		}

		if err != nil {
			rpcStatus, _ := status.FromError(err)
			errs <- errors.Wrapf(err, "broadcast recv error from orderer %s, rpcStatus=%+v", addr, rpcStatus)
			close(responses)
			return
		}

		if broadcastResponse.Status == common.Status_SUCCESS {
			responses <- broadcastResponse.Status
		} else {
			errs <- errors.Errorf("broadcast response error %d from orderer %s", int32(broadcastResponse.Status), addr)
		}
	}
}

//BroadcastWaitForResponse从响应和错误通道读取，直到响应通道关闭。
func BroadcastWaitForResponse(responses chan common.Status, errs chan error) (common.Status, error) {
	var status common.Status
	allErrs := make([]error, 0)

read:
	for {
		select {
		case s, ok := <-responses:
			if !ok {
				break read
			}
			status = s
		case e := <-errs:
			allErrs = append(allErrs, e)
		}
	}

//排出剩余错误
	for i := 0; i < len(errs); i++ {
		e := <-errs
		allErrs = append(allErrs, e)
	}
//关闭errs通道，因为我们已经读取了所有的错误。
	close(errs)
	return status, toError(allErrs)
}

//ToError将[]错误转换为错误
func toError(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	errmsgs := []string{fmt.Sprint("Multiple errors occurred in order broadcast stream: ")}
	for _, err := range errs {
		errmsgs = append(errmsgs, err.Error())
	}
	return errors.New(strings.Join(errmsgs, "\n"))
}
