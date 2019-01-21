
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


package nwo

import (
	"os"
	"os/exec"
)

type Command interface {
	Args() []string
	SessionName() string
}

type Enver interface {
	Env() []string
}

type WorkingDirer interface {
	WorkingDir() string
}

func ConnectsToOrderer(c Command) bool {
	for _, arg := range c.Args() {
		if arg == "--orderer" {
			return true
		}
	}
	return false
}

func NewCommand(path string, command Command) *exec.Cmd {
	cmd := exec.Command(path, command.Args()...)
	cmd.Env = os.Environ()
	if ce, ok := command.(Enver); ok {
		cmd.Env = append(cmd.Env, ce.Env()...)
	}
	if wd, ok := command.(WorkingDirer); ok {
		cmd.Dir = wd.WorkingDir()
	}
	return cmd
}
