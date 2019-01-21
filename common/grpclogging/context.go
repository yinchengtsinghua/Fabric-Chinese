
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

	"go.uber.org/zap/zapcore"
)

type fieldKeyType struct{}

var fieldKey = &fieldKeyType{}

func ZapFields(ctx context.Context) []zapcore.Field {
	fields, ok := ctx.Value(fieldKey).([]zapcore.Field)
	if ok {
		return fields
	}
	return nil
}

func Fields(ctx context.Context) []interface{} {
	fields, ok := ctx.Value(fieldKey).([]zapcore.Field)
	if !ok {
		return nil
	}
	genericFields := make([]interface{}, len(fields))
	for i := range fields {
		genericFields[i] = fields[i]
	}
	return genericFields
}

func WithFields(ctx context.Context, fields []zapcore.Field) context.Context {
	return context.WithValue(ctx, fieldKey, fields)
}
