
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
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type protoMarshaler struct {
	jsonpb.Marshaler
	message proto.Message
}

func (m *protoMarshaler) MarshalJSON() ([]byte, error) {
	out, err := m.Marshaler.MarshalToString(m.message)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

func ProtoMessage(key string, val interface{}) zapcore.Field {
	if pm, ok := val.(proto.Message); ok {
		return zap.Reflect(key, &protoMarshaler{message: pm})
	}
	return zap.Any(key, val)
}

func Error(err error) zapcore.Field {
	if err == nil {
		return zap.Skip()
	}

//包装错误，使其不再实现fmt.formatter。这将阻止
//从添加“verboseerror”字段到包含
//完整的堆栈跟踪。
	return zap.Error(struct{ error }{err})
}
