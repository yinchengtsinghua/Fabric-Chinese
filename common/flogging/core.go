
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
	"go.uber.org/zap/zapcore"
)

type Encoding int8

const (
	CONSOLE = iota
	JSON
)

//encodingselector用于确定日志记录是否
//编码为JSON或人类可读的控制台格式。
type EncodingSelector interface {
	Encoding() Encoding
}

//core是zapcore.core的自定义实现。这是一个可怕的黑客
//只存在于与
//编码器、隐藏在zapcore中的实现以及隐式的即席记录器
//在结构中初始化。
//
//除了将日志条目和字段编码到缓冲区之外，zap编码器
//实现还需要维护字段状态。当zapcore.core.with为
//使用时，将克隆关联的编码器并将字段添加到
//编码器。这意味着编码器实例不能跨核心共享。
//
//在实现隐藏方面，我们的FormatEncoder很难
//将Zap中的JSON和控制台实现作为所有方法进行干净包装
//需要从zapcore.objectEncoder实现委托给
//正确的后端。
//
//此实现通过将多个编码器与一个核心关联来工作。什么时候？
//字段添加到核心，字段添加到所有编码器
//实施。核心还引用日志配置
//确定要使用的正确编码、要委托的编写器和
//启用级别。
type Core struct {
	zapcore.LevelEnabler
	Levels   *LoggerLevels
	Encoders map[Encoding]zapcore.Encoder
	Selector EncodingSelector
	Output   zapcore.WriteSyncer
	Observer Observer
}

//go：生成仿冒者-o mock/observer.go-forke name observer。观察者

type Observer interface {
	Check(e zapcore.Entry, ce *zapcore.CheckedEntry)
	WriteEntry(e zapcore.Entry, fields []zapcore.Field)
}

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	clones := map[Encoding]zapcore.Encoder{}
	for name, enc := range c.Encoders {
		clone := enc.Clone()
		addFields(clone, fields)
		clones[name] = clone
	}

	return &Core{
		LevelEnabler: c.LevelEnabler,
		Levels:       c.Levels,
		Encoders:     clones,
		Selector:     c.Selector,
		Output:       c.Output,
		Observer:     c.Observer,
	}
}

func (c *Core) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Observer != nil {
		c.Observer.Check(e, ce)
	}

	if c.Enabled(e.Level) && c.Levels.Level(e.LoggerName).Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

func (c *Core) Write(e zapcore.Entry, fields []zapcore.Field) error {
	encoding := c.Selector.Encoding()
	enc := c.Encoders[encoding]

	buf, err := enc.EncodeEntry(e, fields)
	if err != nil {
		return err
	}
	_, err = c.Output.Write(buf.Bytes())
	buf.Free()
	if err != nil {
		return err
	}

	if e.Level >= zapcore.PanicLevel {
		c.Sync()
	}

	if c.Observer != nil {
		c.Observer.WriteEntry(e, fields)
	}

	return nil
}

func (c *Core) Sync() error {
	return c.Output.Sync()
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
