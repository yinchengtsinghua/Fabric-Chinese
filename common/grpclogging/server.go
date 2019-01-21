
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


package grpclogging

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

//Leveler返回从GRPC拦截器进行日志记录时要使用的ZAP级别。
type Leveler interface {
	Level(ctx context.Context, fullMethod string) zapcore.Level
}

//payloadLeveler获取在记录GRPC消息有效负载时要使用的级别。
type PayloadLeveler interface {
	PayloadLevel(ctx context.Context, fullMethod string) zapcore.Level
}

//go：生成仿冒者-o仿冒品/校平器。go——仿冒名称校平器。水准仪

type LevelerFunc func(ctx context.Context, fullMethod string) zapcore.Level

func (l LevelerFunc) Level(ctx context.Context, fullMethod string) zapcore.Level {
	return l(ctx, fullMethod)
}

func (l LevelerFunc) PayloadLevel(ctx context.Context, fullMethod string) zapcore.Level {
	return l(ctx, fullMethod)
}

//DefaultPayloadLevel是记录有效负载时使用的默认级别
const DefaultPayloadLevel = zapcore.Level(zapcore.DebugLevel - 1)

type options struct {
	Leveler
	PayloadLeveler
}

type Option func(o *options)

func WithLeveler(l Leveler) Option {
	return func(o *options) { o.Leveler = l }
}

func WithPayloadLeveler(l PayloadLeveler) Option {
	return func(o *options) { o.PayloadLeveler = l }
}

func applyOptions(opts ...Option) *options {
	o := &options{
		Leveler:        LevelerFunc(func(context.Context, string) zapcore.Level { return zapcore.InfoLevel }),
		PayloadLeveler: LevelerFunc(func(context.Context, string) zapcore.Level { return DefaultPayloadLevel }),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

//需要平地机，并应提供完整的方法信息。

func UnaryServerInterceptor(logger *zap.Logger, opts ...Option) grpc.UnaryServerInterceptor {
	o := applyOptions(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := logger
		startTime := time.Now()

		fields := getFields(ctx, info.FullMethod)
		logger = logger.With(fields...)
		ctx = WithFields(ctx, fields)

		payloadLogger := logger.Named("payload")
		payloadLevel := o.PayloadLevel(ctx, info.FullMethod)
		if ce := payloadLogger.Check(payloadLevel, "received unary request"); ce != nil {
			ce.Write(ProtoMessage("message", req))
		}

		resp, err := handler(ctx, req)

		if ce := payloadLogger.Check(payloadLevel, "sending unary response"); ce != nil && err == nil {
			ce.Write(ProtoMessage("message", resp))
		}

		if ce := logger.Check(o.Level(ctx, info.FullMethod), "unary call completed"); ce != nil {
			ce.Write(
				Error(err),
				zap.Stringer("grpc.code", grpc.Code(err)),
				zap.Duration("grpc.call_duration", time.Since(startTime)),
			)
		}

		return resp, err
	}
}

func StreamServerInterceptor(logger *zap.Logger, opts ...Option) grpc.StreamServerInterceptor {
	o := applyOptions(opts...)

	return func(service interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		logger := logger
		ctx := stream.Context()
		startTime := time.Now()

		fields := getFields(ctx, info.FullMethod)
		logger = logger.With(fields...)
		ctx = WithFields(ctx, fields)

		wrappedStream := &serverStream{
			ServerStream:  stream,
			context:       ctx,
			payloadLogger: logger.Named("payload"),
			payloadLevel:  o.PayloadLevel(ctx, info.FullMethod),
		}

		err := handler(service, wrappedStream)
		if ce := logger.Check(o.Level(ctx, info.FullMethod), "streaming call completed"); ce != nil {
			ce.Write(
				Error(err),
				zap.Stringer("grpc.code", grpc.Code(err)),
				zap.Duration("grpc.call_duration", time.Since(startTime)),
			)
		}
		return err
	}
}

func getFields(ctx context.Context, method string) []zapcore.Field {
	var fields []zap.Field
	if parts := strings.Split(method, "/"); len(parts) == 3 {
		fields = append(fields, zap.String("grpc.service", parts[1]), zap.String("grpc.method", parts[2]))
	}
	if deadline, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.Time("grpc.request_deadline", deadline))
	}
	if p, ok := peer.FromContext(ctx); ok {
		fields = append(fields, zap.String("grpc.peer_address", p.Addr.String()))
		if ti, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			if len(ti.State.PeerCertificates) > 0 {
				cert := ti.State.PeerCertificates[0]
				fields = append(fields, zap.String("grpc.peer_subject", cert.Subject.String()))
			}
		}
	}
	return fields
}

type serverStream struct {
	grpc.ServerStream
	context       context.Context
	payloadLogger *zap.Logger
	payloadLevel  zapcore.Level
}

func (ss *serverStream) Context() context.Context {
	return ss.context
}

func (ss *serverStream) SendMsg(msg interface{}) error {
	if ce := ss.payloadLogger.Check(ss.payloadLevel, "sending stream message"); ce != nil {
		ce.Write(ProtoMessage("message", msg))
	}
	return ss.ServerStream.SendMsg(msg)
}

func (ss *serverStream) RecvMsg(msg interface{}) error {
	err := ss.ServerStream.RecvMsg(msg)
	if ce := ss.payloadLogger.Check(ss.payloadLevel, "received stream message"); ce != nil {
		ce.Write(ProtoMessage("message", msg))
	}
	return err
}
