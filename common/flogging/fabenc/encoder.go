
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


package fabenc

import (
	"io"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

//格式编码器是一个zapcore.encoder，它根据
//转到基于日志记录的格式说明符。
type FormatEncoder struct {
	zapcore.Encoder
	formatters []Formatter
	pool       buffer.Pool
}

//格式化程序用于格式化和写入来自ZAP日志条目的数据。
type Formatter interface {
	Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field)
}

func NewFormatEncoder(formatters ...Formatter) *FormatEncoder {
	return &FormatEncoder{
		Encoder: zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
MessageKey:     "", //使残废
LevelKey:       "", //使残废
TimeKey:        "", //使残废
NameKey:        "", //使残废
CallerKey:      "", //使残废
StacktraceKey:  "", //使残废
			LineEnding:     "\n",
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(t.Format("2006-01-02T15:04:05.999Z07:00"))
			},
		}),
		formatters: formatters,
		pool:       buffer.NewPool(),
	}
}

//克隆使用相同的配置创建此编码器的新实例。
func (f *FormatEncoder) Clone() zapcore.Encoder {
	return &FormatEncoder{
		Encoder:    f.Encoder.Clone(),
		formatters: f.formatters,
		pool:       f.pool,
	}
}

//encodeEntry格式化zap日志记录。结构化字段的格式由
//并作为json附加到格式化条目的末尾。
//所有条目都以换行符结尾。
func (f *FormatEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := f.pool.Get()
	for _, f := range f.formatters {
		f.Format(line, entry, fields)
	}

	encodedFields, err := f.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}
	if line.Len() > 0 && encodedFields.Len() != 1 {
		line.AppendString(" ")
	}
	line.AppendString(encodedFields.String())
	encodedFields.Free()

	return line, nil
}
