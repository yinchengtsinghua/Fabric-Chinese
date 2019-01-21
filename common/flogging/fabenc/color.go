
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
)

type Color uint8

const ColorNone Color = 0

const (
	ColorBlack Color = iota + 30
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

func (c Color) Normal() string {
	return fmt.Sprintf("\x1b[%dm", c)
}

func (c Color) Bold() string {
	if c == ColorNone {
		return c.Normal()
	}
	return fmt.Sprintf("\x1b[%d;1m", c)
}

func ResetColor() string { return ColorNone.Normal() }
