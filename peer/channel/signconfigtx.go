
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


package channel

import (
	"io/ioutil"

	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/cobra"
)

func signconfigtxCmd(cf *ChannelCmdFactory) *cobra.Command {
	signconfigtxCmd := &cobra.Command{
		Use:   "signconfigtx",
		Short: "Signs a configtx update.",
		Long:  "Signs the supplied configtx update file in place on the filesystem. Requires '-f'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sign(cmd, args, cf)
		},
	}
	flagList := []string{
		"file",
	}
	attachFlags(signconfigtxCmd, flagList)

	return signconfigtxCmd
}

func sign(cmd *cobra.Command, args []string, cf *ChannelCmdFactory) error {
	if channelTxFile == "" {
		return InvalidCreateTx("No configtx file name supplied")
	}
//对命令行的分析已完成，因此沉默命令用法
	cmd.SilenceUsage = true

	var err error
	if cf == nil {
		cf, err = InitCmdFactory(EndorserNotRequired, PeerDeliverNotRequired, OrdererNotRequired)
		if err != nil {
			return err
		}
	}

	fileData, err := ioutil.ReadFile(channelTxFile)
	if err != nil {
		return ConfigTxFileNotFound(err.Error())
	}

	ctxEnv, err := utils.UnmarshalEnvelope(fileData)
	if err != nil {
		return err
	}

	sCtxEnv, err := sanityCheckAndSignConfigTx(ctxEnv)
	if err != nil {
		return err
	}

	sCtxEnvData := utils.MarshalOrPanic(sCtxEnv)

	return ioutil.WriteFile(channelTxFile, sCtxEnvData, 0660)
}
