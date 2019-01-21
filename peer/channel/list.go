
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package channel

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/scc/cscc"
	common2 "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/cobra"
)

type endorserClient struct {
	cf *ChannelCmdFactory
}

func listCmd(cf *ChannelCmdFactory) *cobra.Command {
//在通道启动命令上设置标志。
	return &cobra.Command{
		Use:   "list",
		Short: "List of channels peer has joined.",
		Long:  "List of channels peer has joined.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("trailing args detected: %s", args)
			}
//对命令行的分析已完成，因此沉默命令用法
			cmd.SilenceUsage = true
			return list(cf)
		},
	}
}

func (cc *endorserClient) getChannels() ([]*pb.ChannelInfo, error) {
	var err error

	invocation := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]),
			ChaincodeId: &pb.ChaincodeID{Name: "cscc"},
			Input:       &pb.ChaincodeInput{Args: [][]byte{[]byte(cscc.GetChannels)}},
		},
	}

	var prop *pb.Proposal
	c, _ := cc.cf.Signer.Serialize()
	prop, _, err = utils.CreateProposalFromCIS(common2.HeaderType_ENDORSER_TRANSACTION, "", invocation, c)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot create proposal, due to %s", err))
	}

	var signedProp *pb.SignedProposal
	signedProp, err = utils.GetSignedProposal(prop, cc.cf.Signer)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot create signed proposal, due to %s", err))
	}

	proposalResp, err := cc.cf.EndorserClient.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed sending proposal, got %s", err))
	}

	if proposalResp.Response == nil || proposalResp.Response.Status != 200 {
		return nil, errors.New(fmt.Sprintf("Received bad response, status %d: %s", proposalResp.Response.Status, proposalResp.Response.Message))
	}

	var channelQueryResponse pb.ChannelQueryResponse
	err = proto.Unmarshal(proposalResp.Response.Payload, &channelQueryResponse)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot read channels list response, %s", err))
	}

	return channelQueryResponse.Channels, nil
}

func list(cf *ChannelCmdFactory) error {
	var err error
	if cf == nil {
		cf, err = InitCmdFactory(EndorserRequired, PeerDeliverNotRequired, OrdererNotRequired)
		if err != nil {
			return err
		}
	}

	client := &endorserClient{cf}

	if channels, err := client.getChannels(); err != nil {
		return err
	} else {
		fmt.Println("Channels peers has joined: ")

		for _, channel := range channels {
			fmt.Printf("%s\n", channel.ChannelId)
		}
	}

	return nil
}
