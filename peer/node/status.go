
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


package node

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/peer/common"
	common2 "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return nodeStatusCmd
}

var nodeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Returns status of the node.",
	Long:  `Returns the status of the running node.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected: %s", args)
		}
//对命令行的分析已完成，因此沉默命令用法
		cmd.SilenceUsage = true
		return status()
	},
}

func status() (err error) {
	adminClient, err := common.GetAdminClient()
	if err != nil {
		logger.Warningf("%s", err)
		fmt.Println(&pb.ServerStatus{Status: pb.ServerStatus_UNKNOWN})
		return err
	}
	signer, err := common.GetDefaultSignerFnc()
	if err != nil {
		fmt.Println(&pb.ServerStatus{Status: pb.ServerStatus_UNKNOWN})
		return errors.Errorf("failed obtaining default signer: %v", err)
	}

	localSigner := crypto.NewSignatureHeaderCreator(signer)
	wrapEnv := func(msg proto.Message) *common2.Envelope {
		env, err := utils.CreateSignedEnvelope(common2.HeaderType_PEER_ADMIN_OPERATION, "", localSigner, msg, 0, 0)
		if err != nil {
			logger.Panicf("Failed signing: %v", err)
		}
		return env
	}

	status, err := adminClient.GetStatus(context.Background(), wrapEnv(&pb.AdminOperation{}))
	if err != nil {
		logger.Infof("Error trying to get status from local peer: %s", err)
		err = fmt.Errorf("Error trying to connect to local peer: %s", err)
		fmt.Println(&pb.ServerStatus{Status: pb.ServerStatus_UNKNOWN})
		return err
	}
	fmt.Println(status)
	return nil
}
