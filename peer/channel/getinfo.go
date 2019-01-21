
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
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/scc/qscc"
	"github.com/hyperledger/fabric/peer/common"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func getinfoCmd(cf *ChannelCmdFactory) *cobra.Command {
	getinfoCmd := &cobra.Command{
		Use:   "getinfo",
		Short: "get blockchain information of a specified channel.",
		Long:  "get blockchain information of a specified channel. Requires '-c'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getinfo(cmd, cf)
		},
	}
	flagList := []string{
		"channelID",
	}
	attachFlags(getinfoCmd, flagList)

	return getinfoCmd
}
func (cc *endorserClient) getBlockChainInfo() (*cb.BlockchainInfo, error) {
	var err error

	invocation := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]),
			ChaincodeId: &pb.ChaincodeID{Name: "qscc"},
			Input:       &pb.ChaincodeInput{Args: [][]byte{[]byte(qscc.GetChainInfo), []byte(channelID)}},
		},
	}

	var prop *pb.Proposal
	c, _ := cc.cf.Signer.Serialize()
	prop, _, err = utils.CreateProposalFromCIS(cb.HeaderType_ENDORSER_TRANSACTION, "", invocation, c)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot create proposal")
	}

	var signedProp *pb.SignedProposal
	signedProp, err = utils.GetSignedProposal(prop, cc.cf.Signer)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot create signed proposal")
	}

	proposalResp, err := cc.cf.EndorserClient.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed sending proposal")
	}

	if proposalResp.Response == nil || proposalResp.Response.Status != 200 {
		return nil, errors.Errorf("received bad response, status %d: %s", proposalResp.Response.Status, proposalResp.Response.Message)
	}

	blockChainInfo := &cb.BlockchainInfo{}
	err = proto.Unmarshal(proposalResp.Response.Payload, blockChainInfo)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read qscc response")
	}

	return blockChainInfo, nil

}

func getinfo(cmd *cobra.Command, cf *ChannelCmdFactory) error {
//由“-c”命令填充的全局chainID
	if channelID == common.UndefinedParamValue {
		return errors.New("Must supply channel ID")
	}
//对命令行的分析已完成，因此沉默命令用法
	cmd.SilenceUsage = true

	var err error
	if cf == nil {
		cf, err = InitCmdFactory(EndorserRequired, PeerDeliverNotRequired, OrdererNotRequired)
		if err != nil {
			return err
		}
	}

	client := &endorserClient{cf}

	blockChainInfo, err := client.getBlockChainInfo()
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(blockChainInfo)
	if err != nil {
		return err
	}

	fmt.Printf("Blockchain info: %s\n", string(jsonBytes))

	return nil
}
