
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
	"context"
	"errors"
	"fmt"

	protcommon "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/cobra"
)

var chaincodeUpgradeCmd *cobra.Command

const upgradeCmdName = "upgrade"

//upgradeCmd返回用于链码升级的COBRA命令
func upgradeCmd(cf *ChaincodeCmdFactory) *cobra.Command {
	chaincodeUpgradeCmd = &cobra.Command{
		Use:       upgradeCmdName,
		Short:     "Upgrade chaincode.",
		Long:      "Upgrade an existing chaincode with the specified one. The new chaincode will immediately replace the existing chaincode upon the transaction committed.",
		ValidArgs: []string{"1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return chaincodeUpgrade(cmd, args, cf)
		},
	}
	flagList := []string{
		"lang",
		"ctor",
		"path",
		"name",
		"channelID",
		"version",
		"policy",
		"escc",
		"vscc",
		"peerAddresses",
		"tlsRootCertFiles",
		"connectionProfile",
		"collections-config",
	}
	attachFlags(chaincodeUpgradeCmd, flagList)

	return chaincodeUpgradeCmd
}

//通过背书器升级命令
func upgrade(cmd *cobra.Command, cf *ChaincodeCmdFactory) (*protcommon.Envelope, error) {
	spec, err := getChaincodeSpec(cmd)
	if err != nil {
		return nil, err
	}

	cds, err := getChaincodeDeploymentSpec(spec, false)
	if err != nil {
		return nil, fmt.Errorf("error getting chaincode code %s: %s", chaincodeName, err)
	}

	creator, err := cf.Signer.Serialize()
	if err != nil {
		return nil, fmt.Errorf("error serializing identity for %s: %s", cf.Signer.GetIdentifier(), err)
	}

	prop, _, err := utils.CreateUpgradeProposalFromCDS(channelID, cds, creator, policyMarshalled, []byte(escc), []byte(vscc), collectionConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("error creating proposal %s: %s", chainFuncName, err)
	}
	logger.Debugf("Get upgrade proposal for chaincode <%v>", spec.ChaincodeId)

	var signedProp *pb.SignedProposal
	signedProp, err = utils.GetSignedProposal(prop, cf.Signer)
	if err != nil {
		return nil, fmt.Errorf("error creating signed proposal  %s: %s", chainFuncName, err)
	}

//当前仅支持一个对等机升级
	proposalResponse, err := cf.EndorserClients[0].ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, fmt.Errorf("error endorsing %s: %s", chainFuncName, err)
	}
	logger.Debugf("endorse upgrade proposal, get response <%v>", proposalResponse.Response)

	if proposalResponse != nil {
//汇编一个已签名的事务（这是一个信封消息）
		env, err := utils.CreateSignedTx(prop, cf.Signer, proposalResponse)
		if err != nil {
			return nil, fmt.Errorf("could not assemble transaction, err %s", err)
		}
		logger.Debug("Get Signed envelope")
		return env, nil
	}

	return nil, nil
}

//chaincodeupgrade升级chaincode。一旦成功，新的链代码
//版本打印到stdout
func chaincodeUpgrade(cmd *cobra.Command, args []string, cf *ChaincodeCmdFactory) error {
	if channelID == "" {
		return errors.New("The required parameter 'channelID' is empty. Rerun the command with -C flag")
	}
//对命令行的分析已完成，因此沉默命令用法
	cmd.SilenceUsage = true

	var err error
	if cf == nil {
		cf, err = InitCmdFactory(cmd.Name(), true, true)
		if err != nil {
			return err
		}
	}
	defer cf.BroadcastClient.Close()

	env, err := upgrade(cmd, cf)
	if err != nil {
		return err
	}

	if env != nil {
		logger.Debug("Send signed envelope to orderer")
		err = cf.BroadcastClient.Send(env)
		return err
	}

	return nil
}
