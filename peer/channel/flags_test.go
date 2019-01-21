
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016-2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package channel

import (
	"testing"

	"github.com/hyperledger/fabric/peer/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestOrdererFlags(t *testing.T) {

	var (
		ca       = "root.crt"
		key      = "client.key"
		cert     = "client.crt"
		endpoint = "orderer.example.com:7050"
		sn       = "override.example.com"
	)

	testCmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			t.Logf("rootcert: %s", viper.GetString("orderer.tls.rootcert.file"))
			assert.Equal(t, ca, viper.GetString("orderer.tls.rootcert.file"))
			assert.Equal(t, key, viper.GetString("orderer.tls.clientKey.file"))
			assert.Equal(t, cert, viper.GetString("orderer.tls.clientCert.file"))
			assert.Equal(t, endpoint, viper.GetString("orderer.address"))
			assert.Equal(t, sn, viper.GetString("orderer.tls.serverhostoverride"))
			assert.Equal(t, true, viper.GetBool("orderer.tls.enabled"))
			assert.Equal(t, true, viper.GetBool("orderer.tls.clientAuthRequired"))
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			common.SetOrdererEnv(cmd, args)
		},
	}

	runCmd := Cmd(nil)

	runCmd.AddCommand(testCmd)

	runCmd.SetArgs([]string{"test", "--cafile", ca, "--keyfile", key,
		"--certfile", cert, "--orderer", endpoint, "--tls", "--clientauth",
		"--ordererTLSHostnameOverride", sn})
	err := runCmd.Execute()
	assert.NoError(t, err)

//再次检查env
	t.Logf("address: %s", viper.GetString("orderer.address"))
	assert.Equal(t, ca, viper.GetString("orderer.tls.rootcert.file"))
	assert.Equal(t, key, viper.GetString("orderer.tls.clientKey.file"))
	assert.Equal(t, cert, viper.GetString("orderer.tls.clientCert.file"))
	assert.Equal(t, endpoint, viper.GetString("orderer.address"))
	assert.Equal(t, sn, viper.GetString("orderer.tls.serverhostoverride"))
	assert.Equal(t, true, viper.GetBool("orderer.tls.enabled"))
	assert.Equal(t, true, viper.GetBool("orderer.tls.clientAuthRequired"))
}
