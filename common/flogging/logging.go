
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
	"io"
	"os"
	"sync"

	"github.com/hyperledger/fabric/common/flogging/fabenc"
	logging "github.com/op/go-logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//config用于提供日志记录实例的依赖项。
type Config struct {
//格式是日志记录实例的日志记录格式说明符。如果
//spec是字符串“json”，日志记录将格式化为json。任何
//其他字符串将提供给格式化编码器。请看
//fabenc.parseformat获取有关支持的谓词的详细信息。
//
//如果未提供格式，则提供基本信息的默认格式将
//被使用。
	Format string

//logspec确定为日志系统启用的日志级别。这个
//规范必须采用可由ActivateSpec处理的格式。
//
//如果未提供logspec，将在信息级别启用记录器。
	LogSpec string

//Writer是编码和格式化日志记录的接收器。
//
//如果未提供写入程序，则将使用os.stderr作为日志接收器。
	Writer io.Writer
}

//日志记录维护与结构日志记录系统关联的状态。它是
//
//Go日志和Zap提供的结构化、级别日志。
type Logging struct {
	*LoggerLevels

	mutex          sync.RWMutex
	encoding       Encoding
	encoderConfig  zapcore.EncoderConfig
	multiFormatter *fabenc.MultiFormatter
	writer         zapcore.WriteSyncer
	observer       Observer
}

//新建创建一个新的日志记录系统，并用提供的
//配置。
func New(c Config) (*Logging, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.NameKey = "name"

	s := &Logging{
		LoggerLevels: &LoggerLevels{
			defaultLevel: defaultLevel,
		},
		encoderConfig:  encoderConfig,
		multiFormatter: fabenc.NewMultiFormatter(),
	}

	err := s.Apply(c)
	if err != nil {
		return nil, err
	}
	return s, nil
}

//应用将提供的配置应用于日志记录系统。
func (s *Logging) Apply(c Config) error {
	err := s.SetFormat(c.Format)
	if err != nil {
		return err
	}

	if c.LogSpec == "" {
		c.LogSpec = os.Getenv("FABRIC_LOGGING_SPEC")
	}
	if c.LogSpec == "" {
		c.LogSpec = defaultLevel.String()
	}

	err = s.LoggerLevels.ActivateSpec(c.LogSpec)
	if err != nil {
		return err
	}

	if c.Writer == nil {
		c.Writer = os.Stderr
	}
	s.SetWriter(c.Writer)

	var formatter logging.Formatter
	if s.Encoding() == JSON {
		formatter = SetFormat(defaultFormat)
	} else {
		formatter = SetFormat(c.Format)
	}

	InitBackend(formatter, c.Writer)

	return nil
}

//setformat更新日志记录的格式和编码方式。日志条目
//此方法完成后创建的将使用新格式。
//
//如果无法分析日志格式规范，则返回错误。
func (s *Logging) SetFormat(format string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if format == "" {
		format = defaultFormat
	}

	if format == "json" {
		s.encoding = JSON
		return nil
	}

	formatters, err := fabenc.ParseFormat(format)
	if err != nil {
		return err
	}
	s.multiFormatter.SetFormatters(formatters)
	s.encoding = CONSOLE

	return nil
}

//setwriter控制写入哪些编写器格式的日志记录。
//除了*os.file之外，编写器需要对并发安全。
//由多个go例程使用。
func (s *Logging) SetWriter(w io.Writer) {
	var sw zapcore.WriteSyncer
	switch t := w.(type) {
	case *os.File:
		sw = zapcore.Lock(t)
	case zapcore.WriteSyncer:
		sw = t
	default:
		sw = zapcore.AddSync(w)
	}

	s.mutex.Lock()
	s.writer = sw
	s.mutex.Unlock()
}

//setobserver用于提供将被称为日志的日志观察器
//检查或书写级别。只支持一个观察者。
func (s *Logging) SetObserver(observer Observer) {
	s.mutex.Lock()
	s.observer = observer
	s.mutex.Unlock()
}

//写满足IO。写合同。它委托给作者论点
//设置编写器或配置的编写器字段。核心在编码时使用这个
//日志记录。
func (s *Logging) Write(b []byte) (int, error) {
	s.mutex.RLock()
	w := s.writer
	s.mutex.RUnlock()

	return w.Write(b)
}

//同步满足zapcore.writesyncer接口。它被核心用来
//终止进程前刷新日志记录。
func (s *Logging) Sync() error {
	s.mutex.RLock()
	w := s.writer
	s.mutex.RUnlock()

	return w.Sync()
}

//编码满足编码接口。它决定JSON还是
//写日志记录时，内核应该使用控制台编码器。
func (s *Logging) Encoding() Encoding {
	s.mutex.RLock()
	e := s.encoding
	s.mutex.RUnlock()
	return e
}

//zap logger用指定的名称实例化新的zap.logger。名字是
//用于确定启用了哪些日志级别。
func (s *Logging) ZapLogger(name string) *zap.Logger {
	if !isValidLoggerName(name) {
		panic(fmt.Sprintf("invalid logger name: %s", name))
	}

//始终在此处返回true，因为核心的检查（）。
//方法计算基于日志程序名称的级别
//在活动日志规范上
	levelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool { return true })

	s.mutex.RLock()
	core := &Core{
		LevelEnabler: levelEnabler,
		Levels:       s.LoggerLevels,
		Encoders: map[Encoding]zapcore.Encoder{
			JSON:    zapcore.NewJSONEncoder(s.encoderConfig),
			CONSOLE: fabenc.NewFormatEncoder(s.multiFormatter),
		},
		Selector: s,
		Output:   s,
		Observer: s,
	}
	s.mutex.RUnlock()

	return NewZapLogger(core).Named(name)
}

func (s *Logging) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) {
	s.mutex.RLock()
	observer := s.observer
	s.mutex.RUnlock()

	if observer != nil {
		observer.Check(e, ce)
	}
}

func (s *Logging) WriteEntry(e zapcore.Entry, fields []zapcore.Field) {
	s.mutex.RLock()
	observer := s.observer
	s.mutex.RUnlock()

	if observer != nil {
		observer.WriteEntry(e, fields)
	}
}

//Logger用指定的名称实例化新的FabricLogger。名字是
//用于确定启用了哪些日志级别。
func (s *Logging) Logger(name string) *FabricLogger {
	zl := s.ZapLogger(name)
	return NewFabricLogger(zl)
}
