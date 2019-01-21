
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


package server

import (
	"context"

	"github.com/hyperledger/fabric/protos/token"
	"github.com/pkg/errors"
)

//go：生成仿冒者-o mock/access_control.go-forke name policychecker。策略检查器

//PolicyChecker负责执行基于策略的访问控制
//与令牌命令相关的检查。
type PolicyChecker interface {
	Check(sc *token.SignedCommand, c *token.Command) error
}

//go：生成伪造者-o mock/marshaler.go-伪造名称marshaler。封送者

//编组人员负责编组和签署命令响应。
type Marshaler interface {
	MarshalCommandResponse(command []byte, responsePayload interface{}) (*token.SignedCommandResponse, error)
}

//提供程序负责处理令牌命令。
type Prover struct {
	CapabilityChecker CapabilityChecker
	Marshaler         Marshaler
	PolicyChecker     PolicyChecker
	TMSManager        TMSManager
}

//newprover创建prover
func NewProver(policyChecker PolicyChecker, signingIdentity SignerIdentity) (*Prover, error) {
	responseMarshaler, err := NewResponseMarshaler(signingIdentity)
	if err != nil {
		return nil, err
	}

	return &Prover{
		Marshaler:     responseMarshaler,
		PolicyChecker: policyChecker,
		TMSManager: &Manager{
			LedgerManager: &PeerLedgerManager{},
		},
	}, nil
}

func (s *Prover) ProcessCommand(ctx context.Context, sc *token.SignedCommand) (*token.SignedCommandResponse, error) {
	command, err := UnmarshalCommand(sc.Command)
	if err != nil {
		return s.MarshalErrorResponse(sc.Command, err)
	}

	err = s.ValidateHeader(command.Header)
	if err != nil {
		return s.MarshalErrorResponse(sc.Command, err)
	}

//检查FabToken功能是否启用
	channelId := command.Header.ChannelId
	enabled, err := s.CapabilityChecker.FabToken(channelId)
	if err != nil {
		return s.MarshalErrorResponse(sc.Command, err)
	}
	if !enabled {
		return s.MarshalErrorResponse(sc.Command, errors.Errorf("FabToken capability not enabled for channel %s", channelId))
	}

	err = s.PolicyChecker.Check(sc, command)
	if err != nil {
		return s.MarshalErrorResponse(sc.Command, err)
	}

	var payload interface{}
	switch t := command.GetPayload().(type) {
	case *token.Command_ImportRequest:
		payload, err = s.RequestImport(ctx, command.Header, t.ImportRequest)
	case *token.Command_TransferRequest:
		payload, err = s.RequestTransfer(ctx, command.Header, t.TransferRequest)
	case *token.Command_RedeemRequest:
		payload, err = s.RequestRedeem(ctx, command.Header, t.RedeemRequest)
	case *token.Command_ListRequest:
		payload, err = s.ListUnspentTokens(ctx, command.Header, t.ListRequest)
	case *token.Command_ApproveRequest:
		payload, err = s.RequestApprove(ctx, command.Header, t.ApproveRequest)
	case *token.Command_TransferFromRequest:
		payload, err = s.RequestTransferFrom(ctx, command.Header, t.TransferFromRequest)
	case *token.Command_ExpectationRequest:
		payload, err = s.RequestExpectation(ctx, command.Header, t.ExpectationRequest)
	default:
		err = errors.Errorf("command type not recognized: %T", t)
	}

	if err != nil {
		payload = &token.CommandResponse_Err{
			Err: &token.Error{Message: err.Error()},
		}
	}

	return s.Marshaler.MarshalCommandResponse(sc.Command, payload)
}

func (s *Prover) RequestImport(ctx context.Context, header *token.Header, requestImport *token.ImportRequest) (*token.CommandResponse_TokenTransaction, error) {
	issuer, err := s.TMSManager.GetIssuer(header.ChannelId, requestImport.Credential, header.Creator)
	if err != nil {
		return nil, err
	}

	tokenTransaction, err := issuer.RequestImport(requestImport.TokensToIssue)
	if err != nil {
		return nil, err
	}

	return &token.CommandResponse_TokenTransaction{TokenTransaction: tokenTransaction}, nil
}

func (s *Prover) RequestTransfer(ctx context.Context, header *token.Header, request *token.TransferRequest) (*token.CommandResponse_TokenTransaction, error) {
	transactor, err := s.TMSManager.GetTransactor(header.ChannelId, request.Credential, header.Creator)
	if err != nil {
		return nil, err
	}
	defer transactor.Done()

	tokenTransaction, err := transactor.RequestTransfer(request)
	if err != nil {
		return nil, err
	}

	return &token.CommandResponse_TokenTransaction{TokenTransaction: tokenTransaction}, nil
}

func (s *Prover) RequestRedeem(ctx context.Context, header *token.Header, request *token.RedeemRequest) (*token.CommandResponse_TokenTransaction, error) {
	transactor, err := s.TMSManager.GetTransactor(header.ChannelId, request.Credential, header.Creator)
	if err != nil {
		return nil, err
	}
	defer transactor.Done()

	tokenTransaction, err := transactor.RequestRedeem(request)
	if err != nil {
		return nil, err
	}

	return &token.CommandResponse_TokenTransaction{TokenTransaction: tokenTransaction}, nil
}

func (s *Prover) ListUnspentTokens(ctxt context.Context, header *token.Header, listRequest *token.ListRequest) (*token.CommandResponse_UnspentTokens, error) {
	transactor, err := s.TMSManager.GetTransactor(header.ChannelId, listRequest.Credential, header.Creator)
	if err != nil {
		return nil, err
	}
	defer transactor.Done()

	tokens, err := transactor.ListTokens()
	if err != nil {
		return nil, err
	}

	return &token.CommandResponse_UnspentTokens{UnspentTokens: tokens}, nil
}

func (s *Prover) RequestApprove(ctx context.Context, header *token.Header, request *token.ApproveRequest) (*token.CommandResponse_TokenTransaction, error) {
	transactor, err := s.TMSManager.GetTransactor(header.ChannelId, request.Credential, header.Creator)
	if err != nil {
		return nil, err
	}
	defer transactor.Done()

	tokenTransaction, err := transactor.RequestApprove(request)
	if err != nil {
		return nil, err
	}

	return &token.CommandResponse_TokenTransaction{TokenTransaction: tokenTransaction}, nil
}

func (s *Prover) RequestTransferFrom(ctx context.Context, header *token.Header, request *token.TransferRequest) (*token.CommandResponse_TokenTransaction, error) {
	transactor, err := s.TMSManager.GetTransactor(header.ChannelId, request.Credential, header.Creator)
	if err != nil {
		return nil, err
	}
	defer transactor.Done()

	tokenTransaction, err := transactor.RequestTransferFrom(request)
	if err != nil {
		return nil, err
	}

	return &token.CommandResponse_TokenTransaction{TokenTransaction: tokenTransaction}, nil
}

//请求期望获取颁发者或事务处理者并创建令牌事务响应
//用于进口、转让或赎回。
func (s *Prover) RequestExpectation(ctx context.Context, header *token.Header, request *token.ExpectationRequest) (*token.CommandResponse_TokenTransaction, error) {
	if request.GetExpectation() == nil {
		return nil, errors.New("ExpectationRequest has nil Expectation")
	}
	plainExpectation := request.GetExpectation().GetPlainExpectation()
	if plainExpectation == nil {
		return nil, errors.New("ExpectationRequest has nil PlainExpectation")
	}

//根据请求中的有效负载类型获取颁发者或事务处理者
	var tokenTransaction *token.TokenTransaction
	switch t := plainExpectation.GetPayload().(type) {
	case *token.PlainExpectation_ImportExpectation:
		issuer, err := s.TMSManager.GetIssuer(header.ChannelId, request.Credential, header.Creator)
		if err != nil {
			return nil, err
		}
		tokenTransaction, err = issuer.RequestExpectation(request)
		if err != nil {
			return nil, err
		}
	case *token.PlainExpectation_TransferExpectation:
		transactor, err := s.TMSManager.GetTransactor(header.ChannelId, request.Credential, header.Creator)
		if err != nil {
			return nil, err
		}
		defer transactor.Done()

		tokenTransaction, err = transactor.RequestExpectation(request)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("expectation payload type not recognized: %T", t)
	}

	return &token.CommandResponse_TokenTransaction{TokenTransaction: tokenTransaction}, nil
}

func (s *Prover) ValidateHeader(header *token.Header) error {
	if header == nil {
		return errors.New("command header is required")
	}

	if header.ChannelId == "" {
		return errors.New("channel ID is required in header")
	}

	if len(header.Nonce) == 0 {
		return errors.New("nonce is required in header")
	}

	if len(header.Creator) == 0 {
		return errors.New("creator is required in header")
	}

	return nil
}

func (s *Prover) MarshalErrorResponse(command []byte, e error) (*token.SignedCommandResponse, error) {
	return s.Marshaler.MarshalCommandResponse(
		command,
		&token.CommandResponse_Err{
			Err: &token.Error{Message: e.Error()},
		})
}
