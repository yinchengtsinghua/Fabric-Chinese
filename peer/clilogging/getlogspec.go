
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


package clilogging

import (
	"context"

	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/cobra"
)

func getLogSpecCmd(cf *LoggingCmdFactory) *cobra.Command {
	var loggingGetLogSpecCmd = &cobra.Command{
		Use:   "getlogspec",
		Short: "Returns the active log spec.",
		Long:  `Returns the active logging specification of the peer.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return getLogSpec(cf, cmd, args)
		},
	}

	return loggingGetLogSpecCmd
}

func getLogSpec(cf *LoggingCmdFactory, cmd *cobra.Command, args []string) (err error) {
	err = checkLoggingCmdParams(cmd, args)
	if err == nil {
//对命令行的分析已完成，因此沉默命令用法
		cmd.SilenceUsage = true

		if cf == nil {
			cf, err = InitCmdFactory()
			if err != nil {
				return err
			}
		}
		env := cf.wrapWithEnvelope(&pb.AdminOperation{})
		logResponse, err := cf.AdminClient.GetLogSpec(context.Background(), env)
		if err != nil {
			return err
		}
		logger.Infof("Current logging spec: %s", logResponse.LogSpec)
	}
	return err
}
