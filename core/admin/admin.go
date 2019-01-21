
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


package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = flogging.MustGetLogger("server")

type requestValidator interface {
	validate(ctx context.Context, env *common.Envelope) (*pb.AdminOperation, error)
}

//AccessControlEvaluator评估给定SignedData的创建者
//有资格使用管理服务
type AccessControlEvaluator interface {
//评估评估给定签名数据的创建者的资格
//由行政服务部提供服务
	Evaluate(signatureSet []*common.SignedData) error
}

//newadminserver创建并返回一个管理服务实例。
func NewAdminServer(ace AccessControlEvaluator) *ServerAdmin {
	s := &ServerAdmin{
		v: &validator{
			ace: ace,
		},
		specAtStartup: flogging.Global.Spec(),
	}
	return s
}

//ServerAdmin对等端管理服务的实现
type ServerAdmin struct {
	v requestValidator

	specAtStartup string
}

func (s *ServerAdmin) GetStatus(ctx context.Context, env *common.Envelope) (*pb.ServerStatus, error) {
	if _, err := s.v.validate(ctx, env); err != nil {
		return nil, err
	}
	status := &pb.ServerStatus{Status: pb.ServerStatus_STARTED}
	logger.Debugf("returning status: %s", status)
	return status, nil
}

func (s *ServerAdmin) StartServer(ctx context.Context, env *common.Envelope) (*pb.ServerStatus, error) {
	if _, err := s.v.validate(ctx, env); err != nil {
		return nil, err
	}
	status := &pb.ServerStatus{Status: pb.ServerStatus_STARTED}
	logger.Debugf("returning status: %s", status)
	return status, nil
}

func (s *ServerAdmin) GetModuleLogLevel(ctx context.Context, env *common.Envelope) (*pb.LogLevelResponse, error) {
	op, err := s.v.validate(ctx, env)
	if err != nil {
		return nil, err
	}
	request := op.GetLogReq()
	if request == nil {
		return nil, errors.New("request is nil")
	}
	logLevelString := flogging.GetLoggerLevel(request.LogModule)
	logResponse := &pb.LogLevelResponse{LogModule: request.LogModule, LogLevel: logLevelString}
	return logResponse, nil
}

func (s *ServerAdmin) SetModuleLogLevel(ctx context.Context, env *common.Envelope) (*pb.LogLevelResponse, error) {
	op, err := s.v.validate(ctx, env)
	if err != nil {
		return nil, err
	}
	request := op.GetLogReq()
	if request == nil {
		return nil, errors.New("request is nil")
	}

	spec := fmt.Sprintf("%s:%s=%s", flogging.Global.Spec(), request.LogModule, request.LogLevel)
	err = flogging.Global.ActivateSpec(spec)
	if err != nil {
		err = status.Errorf(codes.InvalidArgument, "error setting log spec to '%s': %s", spec, err.Error())
		return nil, err
	}

	logResponse := &pb.LogLevelResponse{LogModule: request.LogModule, LogLevel: strings.ToUpper(request.LogLevel)}
	return logResponse, nil
}

func (s *ServerAdmin) RevertLogLevels(ctx context.Context, env *common.Envelope) (*empty.Empty, error) {
	if _, err := s.v.validate(ctx, env); err != nil {
		return nil, err
	}
	flogging.ActivateSpec(s.specAtStartup)
	return &empty.Empty{}, nil
}

func (s *ServerAdmin) GetLogSpec(ctx context.Context, env *common.Envelope) (*pb.LogSpecResponse, error) {
	if _, err := s.v.validate(ctx, env); err != nil {
		return nil, err
	}
	logSpec := flogging.Global.Spec()
	logResponse := &pb.LogSpecResponse{LogSpec: logSpec}
	return logResponse, nil
}

func (s *ServerAdmin) SetLogSpec(ctx context.Context, env *common.Envelope) (*pb.LogSpecResponse, error) {
	op, err := s.v.validate(ctx, env)
	if err != nil {
		return nil, err
	}
	request := op.GetLogSpecReq()
	if request == nil {
		return nil, errors.New("request is nil")
	}
	err = flogging.Global.ActivateSpec(request.LogSpec)
	logResponse := &pb.LogSpecResponse{
		LogSpec: request.LogSpec,
	}
	if err != nil {
		logResponse.Error = err.Error()
	}
	return logResponse, nil
}
