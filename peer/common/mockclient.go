
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package common

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	grpc "google.golang.org/grpc"
)

//GetMock背书客户端返回背书客户端返回指定的ProposalResponse和Err（零或错误）
func GetMockEndorserClient(response *pb.ProposalResponse, err error) pb.EndorserClient {
	return &mockEndorserClient{
		response: response,
		err:      err,
	}
}

type mockEndorserClient struct {
	response *pb.ProposalResponse
	err      error
}

func (m *mockEndorserClient) ProcessProposal(ctx context.Context, in *pb.SignedProposal, opts ...grpc.CallOption) (*pb.ProposalResponse, error) {
	return m.response, m.err
}

func GetMockBroadcastClient(err error) BroadcastClient {
	return &mockBroadcastClient{err: err}
}

//mockbroadcastclient立即返回成功
type mockBroadcastClient struct {
	err error
}

func (m *mockBroadcastClient) Send(env *cb.Envelope) error {
	return m.err
}

func (m *mockBroadcastClient) Close() error {
	return nil
}

func GetMockAdminClient(err error) pb.AdminClient {
	return &mockAdminClient{err: err}
}

type mockAdminClient struct {
	status *pb.ServerStatus
	err    error
}

func (m *mockAdminClient) GetStatus(ctx context.Context, in *cb.Envelope, opts ...grpc.CallOption) (*pb.ServerStatus, error) {
	return m.status, m.err
}

func (m *mockAdminClient) DumpStackTrace(ctx context.Context, in *cb.Envelope, opts ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (m *mockAdminClient) StartServer(ctx context.Context, in *cb.Envelope, opts ...grpc.CallOption) (*pb.ServerStatus, error) {
	m.status = &pb.ServerStatus{Status: pb.ServerStatus_STARTED}
	return m.status, m.err
}

func (m *mockAdminClient) GetModuleLogLevel(ctx context.Context, env *cb.Envelope, opts ...grpc.CallOption) (*pb.LogLevelResponse, error) {
	op := &pb.AdminOperation{}
	pl := &cb.Payload{}
	proto.Unmarshal(env.Payload, pl)
	proto.Unmarshal(pl.Data, op)
	response := &pb.LogLevelResponse{LogModule: op.GetLogReq().LogModule, LogLevel: "INFO"}
	return response, m.err
}

func (m *mockAdminClient) SetModuleLogLevel(ctx context.Context, env *cb.Envelope, opts ...grpc.CallOption) (*pb.LogLevelResponse, error) {
	op := &pb.AdminOperation{}
	pl := &cb.Payload{}
	proto.Unmarshal(env.Payload, pl)
	proto.Unmarshal(pl.Data, op)
	response := &pb.LogLevelResponse{LogModule: op.GetLogReq().LogModule, LogLevel: op.GetLogReq().LogLevel}
	return response, m.err
}

func (m *mockAdminClient) RevertLogLevels(ctx context.Context, in *cb.Envelope, opts ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, m.err
}

func (m *mockAdminClient) GetLogSpec(ctx context.Context, in *cb.Envelope, opts ...grpc.CallOption) (*pb.LogSpecResponse, error) {
	response := &pb.LogSpecResponse{LogSpec: "info"}
	return response, m.err
}

func (m *mockAdminClient) SetLogSpec(ctx context.Context, in *cb.Envelope, opts ...grpc.CallOption) (*pb.LogSpecResponse, error) {
	response := &pb.LogSpecResponse{LogSpec: "info"}
	return response, m.err
}
