
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
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/peer/common"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/cobra"
)

func fetchCmd(cf *ChannelCmdFactory) *cobra.Command {
	fetchCmd := &cobra.Command{
		Use:   "fetch <newest|oldest|config|(number)> [outputfile]",
		Short: "Fetch a block",
		Long:  "Fetch a specified block, writing it to a file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetch(cmd, args, cf)
		},
	}
	flagList := []string{
		"channelID",
	}
	attachFlags(fetchCmd, flagList)

	return fetchCmd
}

func fetch(cmd *cobra.Command, args []string, cf *ChannelCmdFactory) error {
	if len(args) == 0 {
		return fmt.Errorf("fetch target required, oldest, newest, config, or a number")
	}
	if len(args) > 2 {
		return fmt.Errorf("trailing args detected")
	}
//对命令行的分析已完成，因此沉默命令用法
	cmd.SilenceUsage = true

//默认为从订购方获取
	ordererRequired := OrdererRequired
	peerDeliverRequired := PeerDeliverNotRequired
	if len(strings.Split(common.OrderingEndpoint, ":")) != 2 {
//如果没有提供订购方终结点，请连接到对等方的交货服务
		ordererRequired = OrdererNotRequired
		peerDeliverRequired = PeerDeliverRequired
	}
	var err error
	if cf == nil {
		cf, err = InitCmdFactory(EndorserNotRequired, peerDeliverRequired, ordererRequired)
		if err != nil {
			return err
		}
	}

	var block *cb.Block

	switch args[0] {
	case "oldest":
		block, err = cf.DeliverClient.GetOldestBlock()
	case "newest":
		block, err = cf.DeliverClient.GetNewestBlock()
	case "config":
		iBlock, err2 := cf.DeliverClient.GetNewestBlock()
		if err2 != nil {
			return err2
		}
		lc, err2 := utils.GetLastConfigIndexFromBlock(iBlock)
		if err2 != nil {
			return err2
		}
		block, err = cf.DeliverClient.GetSpecifiedBlock(lc)
	default:
		num, err2 := strconv.Atoi(args[0])
		if err2 != nil {
			return fmt.Errorf("fetch target illegal: %s", args[0])
		}
		block, err = cf.DeliverClient.GetSpecifiedBlock(uint64(num))
	}

	if err != nil {
		return err
	}

	b, err := proto.Marshal(block)
	if err != nil {
		return err
	}

	var file string
	if len(args) == 1 {
		file = channelID + "_" + args[0] + ".block"
	} else {
		file = args[1]
	}

	if err = ioutil.WriteFile(file, b, 0644); err != nil {
		return err
	}

	return nil
}
