
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
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestNameToLevel(t *testing.T) {
	var tests = []struct {
		names []string
		level zapcore.Level
	}{
		{names: []string{"PAYLOAD", "payload"}, level: flogging.PayloadLevel},
		{names: []string{"DEBUG", "debug"}, level: zapcore.DebugLevel},
		{names: []string{"INFO", "info"}, level: zapcore.InfoLevel},
		{names: []string{"WARNING", "warning", "WARN", "warn"}, level: zapcore.WarnLevel},
		{names: []string{"ERROR", "error"}, level: zapcore.ErrorLevel},
		{names: []string{"DPANIC", "dpanic"}, level: zapcore.DPanicLevel},
		{names: []string{"PANIC", "panic"}, level: zapcore.PanicLevel},
		{names: []string{"FATAL", "fatal"}, level: zapcore.FatalLevel},
		{names: []string{"NOTICE", "notice"}, level: zapcore.InfoLevel},
		{names: []string{"CRITICAL", "critical"}, level: zapcore.ErrorLevel},
		{names: []string{"unexpected", "invalid"}, level: zapcore.InfoLevel},
	}

	for _, tc := range tests {
		for _, name := range tc.names {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, tc.level, flogging.NameToLevel(name))
			})
		}
	}
}

func TestIsValidLevel(t *testing.T) {
	validNames := []string{
		"PAYLOAD", "payload",
		"DEBUG", "debug",
		"INFO", "info",
		"WARNING", "warning",
		"WARN", "warn",
		"ERROR", "error",
		"DPANIC", "dpanic",
		"PANIC", "panic",
		"FATAL", "fatal",
		"NOTICE", "notice",
		"CRITICAL", "critical",
	}
	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			assert.True(t, flogging.IsValidLevel(name))
		})
	}

	invalidNames := []string{
		"george", "bob",
		"warnings", "inf",
"DISABLED", "disabled", //只能以编程方式使用
	}
	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			assert.False(t, flogging.IsValidLevel(name))
		})
	}
}
