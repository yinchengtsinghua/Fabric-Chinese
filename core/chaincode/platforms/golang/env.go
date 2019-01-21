
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有2017-greg haskins<gregory.haskins@gmail.com>

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

   http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package golang

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Env map[string]string

func getEnv() Env {
	env := make(Env)
	for _, entry := range os.Environ() {
		tokens := strings.SplitN(entry, "=", 2)
		if len(tokens) > 1 {
			env[tokens[0]] = tokens[1]
		}
	}

	return env
}

func getGoEnv() (Env, error) {
	env := getEnv()

	goenvbytes, err := runProgram(env, 10*time.Second, "go", "env")
	if err != nil {
		return nil, err
	}

	goenv := make(Env)

	envout := strings.Split(string(goenvbytes), "\n")
	for _, entry := range envout {
		tokens := strings.SplitN(entry, "=", 2)
		if len(tokens) > 1 {
			goenv[tokens[0]] = strings.Trim(tokens[1], "\"")
		}
	}

	return goenv, nil
}

func flattenEnv(env Env) []string {
	result := make([]string, 0)
	for k, v := range env {
		result = append(result, k+"="+v)
	}

	return result
}

type Paths map[string]bool

func splitEnvPaths(value string) Paths {
	_paths := filepath.SplitList(value)
	paths := make(Paths)
	for _, path := range _paths {
		paths[path] = true
	}
	return paths
}

func flattenEnvPaths(paths Paths) string {

	_paths := make([]string, 0)
	for path := range paths {
		_paths = append(_paths, path)
	}

	return strings.Join(_paths, string(os.PathListSeparator))
}
