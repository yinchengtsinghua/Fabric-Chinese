
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package cluster

import (
	"context"
	"sync"

	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

//去：生成mokery-dir。-name submitclient-case underline-output./mocks/

//submitclient是提交GRPC流
type SubmitClient interface {
	Send(request *orderer.SubmitRequest) error
	Recv() (*orderer.SubmitResponse, error)
	grpc.ClientStream
}

//去：生成mokery-dir。-name client-case underline-output./mocks/

//客户机是群集GRPC服务的操作定义。
//向群集节点公开。
type Client interface {
//提交向群集成员提交事务
	Submit(ctx context.Context, opts ...grpc.CallOption) (orderer.Cluster_SubmitClient, error)
//步骤将特定于实现的消息传递给另一个集群成员。
	Step(ctx context.Context, in *orderer.StepRequest, opts ...grpc.CallOption) (*orderer.StepResponse, error)
}

//RPC执行对远程群集节点的远程过程调用。
type RPC struct {
	Channel             string
	Comm                Communicator
	lock                sync.RWMutex
	DestinationToStream map[uint64]orderer.Cluster_SubmitClient
}

//step向给定的目标节点发送stepRequest并返回响应
func (s *RPC) Step(destination uint64, msg *orderer.StepRequest) (*orderer.StepResponse, error) {
	stub, err := s.Comm.Remote(s.Channel, destination)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return stub.Step(msg)
}

//sendsubmit向给定的目标节点发送submitRequest
func (s *RPC) SendSubmit(destination uint64, request *orderer.SubmitRequest) error {
	stream, err := s.getProposeStream(destination)
	if err != nil {
		return err
	}
	err = stream.Send(request)
	if err != nil {
		s.unMapStream(destination)
	}
	return err
}

//ReceiveSubmitResponse从给定的目标节点接收SubmitResponse
func (s *RPC) ReceiveSubmitResponse(destination uint64) (*orderer.SubmitResponse, error) {
	stream, err := s.getProposeStream(destination)
	if err != nil {
		return nil, err
	}
	msg, err := stream.Recv()
	if err != nil {
		s.unMapStream(destination)
	}
	return msg, err
}

//GetProposeStream获取给定目标节点的提交流
func (s *RPC) getProposeStream(destination uint64) (orderer.Cluster_SubmitClient, error) {
	stream := s.getStream(destination)
	if stream != nil {
		return stream, nil
	}
	stub, err := s.Comm.Remote(s.Channel, destination)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	stream, err = stub.SubmitStream()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	s.mapStream(destination, stream)
	return stream, nil
}

func (s *RPC) getStream(destination uint64) orderer.Cluster_SubmitClient {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.DestinationToStream[destination]
}

func (s *RPC) mapStream(destination uint64, stream orderer.Cluster_SubmitClient) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.DestinationToStream[destination] = stream
}

func (s *RPC) unMapStream(destination uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.DestinationToStream, destination)
}
