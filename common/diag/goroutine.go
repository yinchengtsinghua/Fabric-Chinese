
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package diag

import (
	"bytes"
	"runtime/pprof"
)

type Logger interface {
	Infof(template string, args ...interface{})
	Errorf(template string, args ...interface{})
}

func CaptureGoRoutines() (string, error) {
	var buf bytes.Buffer
	err := pprof.Lookup("goroutine").WriteTo(&buf, 2)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func LogGoRoutines(logger Logger) {
	output, err := CaptureGoRoutines()
	if err != nil {
		logger.Errorf("failed to capture go routines: %s", err)
		return
	}

	logger.Infof("Go routines report:\n%s", output)
}
