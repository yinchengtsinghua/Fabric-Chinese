
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
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/zap/zapcore"
)

//FormatRegexp分为三组：
//1。格式动词
//2。不与''组合的可选冒号？：
//三。可选的非贪婪格式指令
//
//分组简化了规范解析期间的动词处理。
var formatRegexp = regexp.MustCompile(`%{(color|id|level|message|module|shortfunc|time)(?::(.*?))?}`)

//ParseFormat分析日志格式规范并返回一个格式化程序切片
//
//
//此格式化程序支持的op loggng说明符是：
//
//-%id-唯一的日志序列号
//-%级别-条目的日志级别
//-%消息-日志消息
//-%模块-ZAP记录器名称
//-%shortfunc-创建日志记录的函数的名称
//-%时间-创建日志项的时间
//
//说明符可以包括可选的格式动词：
//-颜色：重置粗体
//-id:一个没有前导百分比的fmt样式数字格式化程序
//-级别：不带前导百分比的Fmt样式字符串格式化程序
//-message:没有前导百分比的fmt样式字符串格式化程序
//-模块：不带前导百分比的Fmt样式字符串格式化程序
//
func ParseFormat(spec string) ([]Formatter, error) {
	cursor := 0
	formatters := []Formatter{}

//遍历regex组并转换为格式化程序
	matches := formatRegexp.FindAllStringSubmatchIndex(spec, -1)
	for _, m := range matches {
		start, end := m[0], m[1]
		verbStart, verbEnd := m[2], m[3]
		formatStart, formatEnd := m[4], m[5]

		if start > cursor {
			formatters = append(formatters, StringFormatter{Value: spec[cursor:start]})
		}

		var format string
		if formatStart >= 0 {
			format = spec[formatStart:formatEnd]
		}

		formatter, err := NewFormatter(spec[verbStart:verbEnd], format)
		if err != nil {
			return nil, err
		}

		formatters = append(formatters, formatter)
		cursor = end
	}

//处理任何尾部后缀
	if cursor != len(spec) {
		formatters = append(formatters, StringFormatter{Value: spec[cursor:]})
	}

	return formatters, nil
}

//多格式设置工具将多个格式设置工具显示为单个格式设置工具。它可以
//用于更改与编码器关联的格式化程序集
//运行时。
type MultiFormatter struct {
	mutex      sync.RWMutex
	formatters []Formatter
}

//newmultiformater创建一个新的multiformater，该multiformater委托给
//提供了格式化程序。格式化程序按顺序使用
//提出了。
func NewMultiFormatter(formatters ...Formatter) *MultiFormatter {
	return &MultiFormatter{
		formatters: formatters,
	}
}

//格式迭代其委托以将日志记录格式化为提供的
//缓冲器。
func (m *MultiFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	m.mutex.RLock()
	for i := range m.formatters {
		m.formatters[i].Format(w, entry, fields)
	}
	m.mutex.RUnlock()
}

//setFormatters替换委托格式化程序。
func (m *MultiFormatter) SetFormatters(formatters []Formatter) {
	m.mutex.Lock()
	m.formatters = formatters
	m.mutex.Unlock()
}

//StringFormatter格式化固定字符串。
type StringFormatter struct{ Value string }

//FORMAT将格式化程序的固定字符串写入提供的写入程序。
func (s StringFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	fmt.Fprintf(w, "%s", s.Value)
}

//newformatter为提供的动词创建格式化程序。当格式为
//未提供，将使用动词的默认格式。
func NewFormatter(verb, format string) (Formatter, error) {
	switch verb {
	case "color":
		return newColorFormatter(format)
	case "id":
		return newSequenceFormatter(format), nil
	case "level":
		return newLevelFormatter(format), nil
	case "message":
		return newMessageFormatter(format), nil
	case "module":
		return newModuleFormatter(format), nil
	case "shortfunc":
		return newShortFuncFormatter(format), nil
	case "time":
		return newTimeFormatter(format), nil
	default:
		return nil, fmt.Errorf("unknown verb: %s", verb)
	}
}

//颜色格式化程序格式化SGR颜色代码。
type ColorFormatter struct {
Bold  bool //设置粗体属性
Reset bool //重置颜色和属性
}

func newColorFormatter(f string) (ColorFormatter, error) {
	switch f {
	case "bold":
		return ColorFormatter{Bold: true}, nil
	case "reset":
		return ColorFormatter{Reset: true}, nil
	case "":
		return ColorFormatter{}, nil
	default:
		return ColorFormatter{}, fmt.Errorf("invalid color option: %s", f)
	}
}

//LevelColor返回与特定ZAP日志记录级别关联的颜色。
func (c ColorFormatter) LevelColor(l zapcore.Level) Color {
	switch l {
	case zapcore.DebugLevel:
		return ColorCyan
	case zapcore.InfoLevel:
		return ColorBlue
	case zapcore.WarnLevel:
		return ColorYellow
	case zapcore.ErrorLevel:
		return ColorRed
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		return ColorMagenta
	case zapcore.FatalLevel:
		return ColorMagenta
	default:
		return ColorNone
	}
}

//FORMAT将SGR颜色代码写入提供的写入程序。
func (c ColorFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	switch {
	case c.Reset:
		fmt.Fprintf(w, ResetColor())
	case c.Bold:
		fmt.Fprintf(w, c.LevelColor(entry.Level).Bold())
	default:
		fmt.Fprintf(w, c.LevelColor(entry.Level).Normal())
	}
}

//级别格式化程序格式化日志级别。
type LevelFormatter struct{ FormatVerb string }

func newLevelFormatter(f string) LevelFormatter {
	return LevelFormatter{FormatVerb: "%" + stringOrDefault(f, "s")}
}

//FORMAT将日志级别写入提供的写入程序。
func (l LevelFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	fmt.Fprintf(w, l.FormatVerb, entry.Level.CapitalString())
}

//messageformatter格式化日志消息。
type MessageFormatter struct{ FormatVerb string }

func newMessageFormatter(f string) MessageFormatter {
	return MessageFormatter{FormatVerb: "%" + stringOrDefault(f, "s")}
}

//FORMAT将日志条目消息写入提供的写入程序。
func (m MessageFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	fmt.Fprintf(w, m.FormatVerb, strings.TrimRight(entry.Message, "\n"))
}

//moduleForMatter格式化zap记录器名称。
type ModuleFormatter struct{ FormatVerb string }

func newModuleFormatter(f string) ModuleFormatter {
	return ModuleFormatter{FormatVerb: "%" + stringOrDefault(f, "s")}
}

//FORMAT将ZAP记录器名称写入指定的写入程序。
func (m ModuleFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	fmt.Fprintf(w, m.FormatVerb, entry.LoggerName)
}

//sequence维护所有sequeneformatter共享的全局序列号
//实例。
var sequence uint64

//setsequence显式设置全局序列号。
func SetSequence(s uint64) { atomic.StoreUint64(&sequence, s) }

//SequenceFormatter格式化全局序列号。
type SequenceFormatter struct{ FormatVerb string }

func newSequenceFormatter(f string) SequenceFormatter {
	return SequenceFormatter{FormatVerb: "%" + stringOrDefault(f, "d")}
}

//SequenceFormatter递增全局序列号并将其写入
//提供了作者。
func (s SequenceFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	fmt.Fprintf(w, s.FormatVerb, atomic.AddUint64(&sequence, 1))
}

//shortfuncormatter格式化创建日志记录的函数的名称。
type ShortFuncFormatter struct{ FormatVerb string }

func newShortFuncFormatter(f string) ShortFuncFormatter {
	return ShortFuncFormatter{FormatVerb: "%" + stringOrDefault(f, "s")}
}

//FORMAT将调用函数名写入提供的编写器。名称是从
//运行时、包和行号将被丢弃。
func (s ShortFuncFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	f := runtime.FuncForPC(entry.Caller.PC)
	if f == nil {
		fmt.Fprintf(w, s.FormatVerb, "(unknown)")
		return
	}

	fname := f.Name()
	funcIdx := strings.LastIndex(fname, ".")
	fmt.Fprintf(w, s.FormatVerb, fname[funcIdx+1:])
}

//TimeFormatter格式化来自zap日志项的时间。
type TimeFormatter struct{ Layout string }

func newTimeFormatter(f string) TimeFormatter {
	return TimeFormatter{Layout: stringOrDefault(f, "2006-01-02T15:04:05.999Z07:00")}
}

//FORMAT将日志记录时间戳写入提供的写入程序。
func (t TimeFormatter) Format(w io.Writer, entry zapcore.Entry, fields []zapcore.Field) {
	fmt.Fprint(w, entry.Time.Format(t.Layout))
}

func stringOrDefault(str, dflt string) string {
	if str != "" {
		return str
	}
	return dflt
}
