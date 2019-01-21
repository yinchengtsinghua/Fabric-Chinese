
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
*/


package flogging

import (
	"fmt"
	"io"
	"os"
	"strings"

	logging "github.com/op/go-logging"
)

//
//
//更高级别的对等机。

//setformat（字符串）logging.formatter
//initbackend（logging.formatter，io.writer）
//defaultLevel（）字符串
//initfromspec（字符串）字符串

//setformat设置日志记录格式。
func SetFormat(formatSpec string) logging.Formatter {
	if formatSpec == "" {
		formatSpec = defaultFormat
	}
	return logging.MustStringFormatter(formatSpec)
}

//initbackend根据设置日志后端
//提供的日志格式化程序和I/O编写器。
func InitBackend(formatter logging.Formatter, output io.Writer) {
	backend := logging.NewLogBackend(output, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, formatter)
	logging.SetBackend(backendFormatter).SetLevel(logging.INFO, "")
}

//DefaultLevel返回分析失败时记录器使用的回退值。
func DefaultLevel() string {
	return strings.ToUpper(Global.DefaultLevel().String())
}

//initfromspec根据提供的规范初始化日志。它是
//暴露在外部，以便flogging包的使用者可以分析
//自己的日志规范。日志规范的格式如下：
//[<logger>[，<logger>…]=]<level>[：[<logger>[，<logger>…]=]<logger>…]
func InitFromSpec(spec string) string {
	err := Global.ActivateSpec(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to activate logging spec: %s", err)
	}
	return DefaultLevel()
}
