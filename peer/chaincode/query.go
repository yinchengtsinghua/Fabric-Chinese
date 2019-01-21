
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


package chaincode

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var chaincodeQueryCmd *cobra.Command

//querycmd返回用于链码查询的COBRA命令
func queryCmd(cf *ChaincodeCmdFactory) *cobra.Command {
	chaincodeQueryCmd = &cobra.Command{
		Use:       "query",
		Short:     fmt.Sprintf("Query using the specified %s.", chainFuncName),
		Long:      fmt.Sprintf("Get endorsed result of %s function call and print it. It won't generate transaction.", chainFuncName),
		ValidArgs: []string{"1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return chaincodeQuery(cmd, cf)
		},
	}
	flagList := []string{
		"ctor",
		"name",
		"channelID",
		"peerAddresses",
		"tlsRootCertFiles",
		"connectionProfile",
	}
	attachFlags(chaincodeQueryCmd, flagList)

	chaincodeQueryCmd.Flags().BoolVarP(&chaincodeQueryRaw, "raw", "r", false,
		"If true, output the query value as raw bytes, otherwise format as a printable string")
	chaincodeQueryCmd.Flags().BoolVarP(&chaincodeQueryHex, "hex", "x", false,
		"If true, output the query value byte array in hexadecimal. Incompatible with --raw")

	return chaincodeQueryCmd
}

func chaincodeQuery(cmd *cobra.Command, cf *ChaincodeCmdFactory) error {
	if channelID == "" {
		return errors.New("The required parameter 'channelID' is empty. Rerun the command with -C flag")
	}
//对命令行的分析已完成，因此沉默命令用法
	cmd.SilenceUsage = true

	var err error
	if cf == nil {
		cf, err = InitCmdFactory(cmd.Name(), true, false)
		if err != nil {
			return err
		}
	}

	return chaincodeInvokeOrQuery(cmd, false, cf)
}
