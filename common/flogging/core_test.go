
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


package flogging_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/flogging/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func TestCoreWith(t *testing.T) {
	core := &flogging.Core{
		Encoders: map[flogging.Encoding]zapcore.Encoder{},
		Observer: &mock.Observer{},
	}
	clone := core.With([]zapcore.Field{zap.String("key", "value")})
	assert.Equal(t, core, clone)

	jsonEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
	consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{})
	core = &flogging.Core{
		Encoders: map[flogging.Encoding]zapcore.Encoder{
			flogging.JSON:    jsonEncoder,
			flogging.CONSOLE: consoleEncoder,
		},
	}
	decorated := core.With([]zapcore.Field{zap.String("key", "value")})

//验证对象是否不同
	assert.NotEqual(t, core, decorated)

//验证对象仅与编码字段不同
	jsonEncoder.AddString("key", "value")
	consoleEncoder.AddString("key", "value")
	assert.Equal(t, core, decorated)
}

func TestCoreCheck(t *testing.T) {
	var enabledArgs []zapcore.Level
	levels := &flogging.LoggerLevels{}
	err := levels.ActivateSpec("warning")
	assert.NoError(t, err)
	core := &flogging.Core{
		LevelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool {
			enabledArgs = append(enabledArgs, l)
			return l >= zapcore.WarnLevel
		}),
		Levels: levels,
	}

//未启用
	ce := core.Check(zapcore.Entry{Level: zapcore.DebugLevel}, nil)
	assert.Nil(t, ce)
	ce = core.Check(zapcore.Entry{Level: zapcore.InfoLevel}, nil)
	assert.Nil(t, ce)

//启用
	ce = core.Check(zapcore.Entry{Level: zapcore.WarnLevel}, nil)
	assert.NotNil(t, ce)

	assert.Equal(t, enabledArgs, []zapcore.Level{zapcore.DebugLevel, zapcore.InfoLevel, zapcore.WarnLevel})
}

type sw struct {
	bytes.Buffer
	writeErr   error
	syncCalled bool
	syncErr    error
}

func (s *sw) Sync() error {
	s.syncCalled = true
	return s.syncErr
}

func (s *sw) Write(b []byte) (int, error) {
	if s.writeErr != nil {
		return 0, s.writeErr
	}
	return s.Buffer.Write(b)
}

func (s *sw) Encoding() flogging.Encoding {
	return flogging.CONSOLE
}

func TestCoreWrite(t *testing.T) {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = nil

	output := &sw{}
	core := &flogging.Core{
		Encoders: map[flogging.Encoding]zapcore.Encoder{
			flogging.CONSOLE: zapcore.NewConsoleEncoder(encoderConfig),
		},
		Selector: output,
		Output:   output,
	}

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "this is a message",
	}
	err := core.Write(entry, nil)
	assert.NoError(t, err)
	assert.Equal(t, "INFO\tthis is a message\n", output.String())

	output.writeErr = errors.New("super-loose")
	err = core.Write(entry, nil)
	assert.EqualError(t, err, "super-loose")
}

func TestCoreWriteSync(t *testing.T) {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = nil

	output := &sw{}
	core := &flogging.Core{
		Encoders: map[flogging.Encoding]zapcore.Encoder{
			flogging.CONSOLE: zapcore.NewConsoleEncoder(encoderConfig),
		},
		Selector: output,
		Output:   output,
	}

	entry := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "no bugs for me",
	}
	err := core.Write(entry, nil)
	assert.NoError(t, err)
	assert.False(t, output.syncCalled)

	entry = zapcore.Entry{
		Level:   zapcore.PanicLevel,
		Message: "gah!",
	}
	err = core.Write(entry, nil)
	assert.NoError(t, err)
	assert.True(t, output.syncCalled)
}

type brokenEncoder struct{ zapcore.Encoder }

func (b *brokenEncoder) EncodeEntry(zapcore.Entry, []zapcore.Field) (*buffer.Buffer, error) {
	return nil, errors.New("broken encoder")
}

func TestCoreWriteEncodeFail(t *testing.T) {
	output := &sw{}
	core := &flogging.Core{
		Encoders: map[flogging.Encoding]zapcore.Encoder{
			flogging.CONSOLE: &brokenEncoder{},
		},
		Selector: output,
		Output:   output,
	}

	entry := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "no bugs for me",
	}
	err := core.Write(entry, nil)
	assert.EqualError(t, err, "broken encoder")
}

func TestCoreSync(t *testing.T) {
	syncWriter := &sw{}
	core := &flogging.Core{
		Output: syncWriter,
	}

	err := core.Sync()
	assert.NoError(t, err)
	assert.True(t, syncWriter.syncCalled)

	syncWriter.syncErr = errors.New("bummer")
	err = core.Sync()
	assert.EqualError(t, err, "bummer")
}

func TestObserverCheck(t *testing.T) {
	observer := &mock.Observer{}
	entry := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "message",
	}
	checkedEntry := &zapcore.CheckedEntry{}

	levels := &flogging.LoggerLevels{}
	levels.ActivateSpec("debug")
	core := &flogging.Core{
		LevelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool { return true }),
		Levels:       levels,
		Observer:     observer,
	}

	ce := core.Check(entry, checkedEntry)
	assert.Exactly(t, ce, checkedEntry)

	assert.Equal(t, 1, observer.CheckCallCount())
	observedEntry, observedCE := observer.CheckArgsForCall(0)
	assert.Equal(t, entry, observedEntry)
	assert.Equal(t, ce, observedCE)
}

func TestObserverWriteEntry(t *testing.T) {
	observer := &mock.Observer{}
	entry := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "message",
	}
	fields := []zapcore.Field{
		{Key: "key1", Type: zapcore.SkipType},
		{Key: "key2", Type: zapcore.SkipType},
	}

	levels := &flogging.LoggerLevels{}
	levels.ActivateSpec("debug")
	selector := &sw{}
	output := &sw{}
	core := &flogging.Core{
		LevelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool { return true }),
		Levels:       levels,
		Selector:     selector,
		Encoders: map[flogging.Encoding]zapcore.Encoder{
			flogging.CONSOLE: zapcore.NewConsoleEncoder(zapcore.EncoderConfig{}),
		},
		Output:   output,
		Observer: observer,
	}

	err := core.Write(entry, fields)
	assert.NoError(t, err)

	assert.Equal(t, 1, observer.WriteEntryCallCount())
	observedEntry, observedFields := observer.WriteEntryArgsForCall(0)
	assert.Equal(t, entry, observedEntry)
	assert.Equal(t, fields, observedFields)
}
