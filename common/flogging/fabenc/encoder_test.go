
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


package fabenc_test

import (
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/flogging/fabenc"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func TestEncodeEntry(t *testing.T) {
	startTime := time.Now()
	var tests = []struct {
		name     string
		spec     string
		fields   []zapcore.Field
		expected string
	}{
		{name: "empty spec and nil fields", spec: "", fields: nil, expected: "\n"},
		{name: "empty spec with fields", spec: "", fields: []zapcore.Field{zap.String("key", "value")}, expected: `{"key": "value"}` + "\n"},
		{name: "simple spec and nil fields", spec: "simple-string", expected: "simple-string\n"},
		{name: "simple spec and empty fields", spec: "simple-string", fields: []zapcore.Field{}, expected: "simple-string\n"},
		{name: "simple spec with fields", spec: "simple-string", fields: []zapcore.Field{zap.String("key", "value")}, expected: `simple-string {"key": "value"}` + "\n"},
		{name: "duration", spec: "", fields: []zapcore.Field{zap.Duration("duration", time.Second)}, expected: `{"duration": "1s"}` + "\n"},
		{name: "time", spec: "", fields: []zapcore.Field{zap.Time("time", startTime)}, expected: fmt.Sprintf(`{"time": "%s"}`+"\n", startTime.Format("2006-01-02T15:04:05.999Z07:00"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			formatters, err := fabenc.ParseFormat(tc.spec)
			assert.NoError(t, err)

			enc := fabenc.NewFormatEncoder(formatters...)

			pc, file, l, ok := runtime.Caller(0)
			line, err := enc.EncodeEntry(
				zapcore.Entry{
//应完全忽略条目信息
					Level:      zapcore.InfoLevel,
					Time:       startTime,
					LoggerName: "logger-name",
					Message:    "message",
					Caller:     zapcore.NewEntryCaller(pc, file, l, ok),
					Stack:      "stack",
				},
				tc.fields,
			)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, line.String())
		})
	}
}

type brokenEncoder struct{ zapcore.Encoder }

func (b *brokenEncoder) EncodeEntry(zapcore.Entry, []zapcore.Field) (*buffer.Buffer, error) {
	return nil, errors.New("broken encoder")
}

func TestEncodeFieldsFailed(t *testing.T) {
	enc := fabenc.NewFormatEncoder()
	enc.Encoder = &brokenEncoder{}

	_, err := enc.EncodeEntry(zapcore.Entry{}, nil)
	assert.EqualError(t, err, "broken encoder")
}

func TestFormatEncoderClone(t *testing.T) {
	enc := fabenc.NewFormatEncoder()
	cloned := enc.Clone()
	assert.Equal(t, enc, cloned)
}
