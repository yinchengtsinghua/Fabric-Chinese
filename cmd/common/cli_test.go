
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


package common

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/cmd/common/signer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCLI(t *testing.T) {
	var testCmdInvoked bool
	var exited bool
//重写退出
	terminate = func(_ int) {
		exited = true
	}
//通过写入此缓冲区覆盖stdout
	testBuff := &bytes.Buffer{}
	outWriter = testBuff

	var returnValue error
	cli := NewCLI("cli", "cli help")
	testCommand := func(conf Config) error {
//如果我们退出，命令没有执行
		if exited {
			return nil
		}
//否则，命令会被执行-所以确保它被执行
//使用预期的配置
		assert.Equal(t, Config{
			SignerConfig: signer.Config{
				MSPID:        "SampleOrg",
				KeyPath:      "key.pem",
				IdentityPath: "cert.pem",
			},
		}, conf)
		testCmdInvoked = true
		return returnValue
	}
	cli.Command("test", "test help", testCommand)

	t.Run("Loading a non existent config", func(t *testing.T) {
		defer testBuff.Reset()
//用testdata覆盖用户主目录
		dir := filepath.Join("testdata", "non_existent_config")
		cli.Run([]string{"test", "--configFile", filepath.Join(dir, "config.yaml")})
		assert.Contains(t, testBuff.String(), fmt.Sprint("Failed loading config open ", dir))
		assert.Contains(t, testBuff.String(), "config.yaml: no such file or directory")
		assert.True(t, exited)
	})

	t.Run("Loading a valid config and the command succeeds", func(t *testing.T) {
		defer testBuff.Reset()
		testCmdInvoked = false
		exited = false
//用testdata覆盖用户主目录
		dir := filepath.Join("testdata", "valid_config")
//确保有效的配置会导致运行我们的命令
		cli.Run([]string{"test", "--configFile", filepath.Join(dir, "config.yaml")})
		assert.True(t, testCmdInvoked)
		assert.False(t, exited)
	})

	t.Run("Loading a valid config but the command fails", func(t *testing.T) {
		returnValue = errors.New("something went wrong")
		defer func() {
			returnValue = nil
		}()
		defer testBuff.Reset()
		testCmdInvoked = false
		exited = false
//用testdata覆盖用户主目录
		dir := filepath.Join("testdata", "valid_config")
//确保有效的配置会导致运行我们的命令
		cli.Run([]string{"test", "--configFile", filepath.Join(dir, "config.yaml")})
		assert.True(t, testCmdInvoked)
		assert.True(t, exited)
		assert.Contains(t, testBuff.String(), "something went wrong")
	})

	t.Run("Saving a config", func(t *testing.T) {
		defer testBuff.Reset()
		testCmdInvoked = false
		exited = false
		dir := filepath.Join(os.TempDir(), fmt.Sprintf("config%d", rand.Int()))
		os.Mkdir(dir, 0700)
		defer os.RemoveAll(dir)

		userCert := filepath.Join(dir, "cert.pem")
		userKey := filepath.Join(dir, "key.pem")
		userCertFlag := fmt.Sprintf("--userCert=%s", userCert)
		userKeyFlag := fmt.Sprintf("--userKey=%s", userKey)
		os.Create(userCert)
		os.Create(userKey)

		cli.Run([]string{saveConfigCommand, "--MSP=SampleOrg", userCertFlag, userKeyFlag})
		assert.Contains(t, testBuff.String(), "--configFile must be used to specify the configuration file")
		testBuff.Reset()
//保留配置
		cli.Run([]string{saveConfigCommand, "--MSP=SampleOrg", userCertFlag, userKeyFlag, "--configFile", filepath.Join(dir, "config.yaml")})

//运行其他命令，并确保配置已成功持久化
		cli.Command("assert", "", func(conf Config) error {
			assert.Equal(t, Config{
				SignerConfig: signer.Config{
					MSPID:        "SampleOrg",
					KeyPath:      userKey,
					IdentityPath: userCert,
				},
			}, conf)
			return nil
		})
		cli.Run([]string{"assert", "--configFile", filepath.Join(dir, "config.yaml")})
	})
}
