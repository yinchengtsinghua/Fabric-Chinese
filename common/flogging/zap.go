
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


package flogging

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
)

//new zap logger围绕一个新的zap.core创建一个zap记录器。核心将使用
//提供的编码器和接收器以及与
//提供的记录器名称。返回的记录器将被命名为相同的
//作为记录器。
func NewZapLogger(core zapcore.Core, options ...zap.Option) *zap.Logger {
	return zap.New(
		core,
		append([]zap.Option{
			zap.AddCaller(),
			zap.AddStacktrace(zapcore.ErrorLevel),
		}, options...)...,
	)
}

//newgrpclogger创建委托给zap.logger的grpc.logger。
func NewGRPCLogger(l *zap.Logger) *zapgrpc.Logger {
	l = l.WithOptions(
		zap.AddCaller(),
		zap.AddCallerSkip(3),
	)
	return zapgrpc.NewLogger(l, zapgrpc.WithDebug())
}

//newFabricLogger创建一个委托给zap.sugaredLogger的记录器。
func NewFabricLogger(l *zap.Logger, options ...zap.Option) *FabricLogger {
	return &FabricLogger{
		s: l.WithOptions(append(options, zap.AddCallerSkip(1))...).Sugar(),
	}
}

//FabricLogger是围绕zap.sugaredLogger的适配器，它提供
//
//行为。
//
//FabricLogger和
//zap.sugaredlogger是没有格式后缀（f或w）构建的方法
//使用fmt.sprintln而不是fmt.sprint的日志条目消息。没有这个
//更改，参数不由空格分隔。
type FabricLogger struct{ s *zap.SugaredLogger }

func (f *FabricLogger) DPanic(args ...interface{})                    { f.s.DPanicf(formatArgs(args)) }
func (f *FabricLogger) DPanicf(template string, args ...interface{})  { f.s.DPanicf(template, args...) }
func (f *FabricLogger) DPanicw(msg string, kvPairs ...interface{})    { f.s.DPanicw(msg, kvPairs...) }
func (f *FabricLogger) Debug(args ...interface{})                     { f.s.Debugf(formatArgs(args)) }
func (f *FabricLogger) Debugf(template string, args ...interface{})   { f.s.Debugf(template, args...) }
func (f *FabricLogger) Debugw(msg string, kvPairs ...interface{})     { f.s.Debugw(msg, kvPairs...) }
func (f *FabricLogger) Error(args ...interface{})                     { f.s.Errorf(formatArgs(args)) }
func (f *FabricLogger) Errorf(template string, args ...interface{})   { f.s.Errorf(template, args...) }
func (f *FabricLogger) Errorw(msg string, kvPairs ...interface{})     { f.s.Errorw(msg, kvPairs...) }
func (f *FabricLogger) Fatal(args ...interface{})                     { f.s.Fatalf(formatArgs(args)) }
func (f *FabricLogger) Fatalf(template string, args ...interface{})   { f.s.Fatalf(template, args...) }
func (f *FabricLogger) Fatalw(msg string, kvPairs ...interface{})     { f.s.Fatalw(msg, kvPairs...) }
func (f *FabricLogger) Info(args ...interface{})                      { f.s.Infof(formatArgs(args)) }
func (f *FabricLogger) Infof(template string, args ...interface{})    { f.s.Infof(template, args...) }
func (f *FabricLogger) Infow(msg string, kvPairs ...interface{})      { f.s.Infow(msg, kvPairs...) }
func (f *FabricLogger) Panic(args ...interface{})                     { f.s.Panicf(formatArgs(args)) }
func (f *FabricLogger) Panicf(template string, args ...interface{})   { f.s.Panicf(template, args...) }
func (f *FabricLogger) Panicw(msg string, kvPairs ...interface{})     { f.s.Panicw(msg, kvPairs...) }
func (f *FabricLogger) Warn(args ...interface{})                      { f.s.Warnf(formatArgs(args)) }
func (f *FabricLogger) Warnf(template string, args ...interface{})    { f.s.Warnf(template, args...) }
func (f *FabricLogger) Warnw(msg string, kvPairs ...interface{})      { f.s.Warnw(msg, kvPairs...) }
func (f *FabricLogger) Warning(args ...interface{})                   { f.s.Warnf(formatArgs(args)) }
func (f *FabricLogger) Warningf(template string, args ...interface{}) { f.s.Warnf(template, args...) }

//为了向后兼容
func (f *FabricLogger) Critical(args ...interface{})                   { f.s.Errorf(formatArgs(args)) }
func (f *FabricLogger) Criticalf(template string, args ...interface{}) { f.s.Errorf(template, args...) }
func (f *FabricLogger) Notice(args ...interface{})                     { f.s.Infof(formatArgs(args)) }
func (f *FabricLogger) Noticef(template string, args ...interface{})   { f.s.Infof(template, args...) }

func (f *FabricLogger) Named(name string) *FabricLogger { return &FabricLogger{s: f.s.Named(name)} }
func (f *FabricLogger) Sync() error                     { return f.s.Sync() }
func (f *FabricLogger) Zap() *zap.Logger                { return f.s.Desugar() }

func (f *FabricLogger) IsEnabledFor(level zapcore.Level) bool {
	return f.s.Desugar().Core().Enabled(level)
}

func (f *FabricLogger) With(args ...interface{}) *FabricLogger {
	return &FabricLogger{s: f.s.With(args...)}
}

func (f *FabricLogger) WithOptions(opts ...zap.Option) *FabricLogger {
	l := f.s.Desugar().WithOptions(opts...)
	return &FabricLogger{s: l.Sugar()}
}

func formatArgs(args []interface{}) string { return strings.TrimSuffix(fmt.Sprintln(args...), "\n") }
