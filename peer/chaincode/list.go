
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
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var getInstalledChaincodes bool
var getInstantiatedChaincodes bool
var chaincodeListCmd *cobra.Command

const list_cmdname = "list"

//installCmd返回chaincode deploy的cobra命令
func listCmd(cf *ChaincodeCmdFactory) *cobra.Command {
	chaincodeListCmd = &cobra.Command{
		Use:   "list",
		Short: "Get the instantiated chaincodes on a channel or installed chaincodes on a peer.",
		Long:  "Get the instantiated chaincodes in the channel if specify channel, or get installed chaincodes on the peer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getChaincodes(cmd, cf)
		},
	}

	flagList := []string{
		"channelID",
		"installed",
		"instantiated",
		"peerAddresses",
		"tlsRootCertFiles",
		"connectionProfile",
	}
	attachFlags(chaincodeListCmd, flagList)

	return chaincodeListCmd
}

func getChaincodes(cmd *cobra.Command, cf *ChaincodeCmdFactory) error {
	if getInstantiatedChaincodes && channelID == "" {
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

	creator, err := cf.Signer.Serialize()
	if err != nil {
		return fmt.Errorf("Error serializing identity for %s: %s", cf.Signer.GetIdentifier(), err)
	}

	var prop *pb.Proposal
	if getInstalledChaincodes && (!getInstantiatedChaincodes) {
		prop, _, err = utils.CreateGetInstalledChaincodesProposal(creator)
	} else if getInstantiatedChaincodes && (!getInstalledChaincodes) {
		prop, _, err = utils.CreateGetChaincodesProposal(channelID, creator)
	} else {
		return fmt.Errorf("Must explicitly specify \"--installed\" or \"--instantiated\"")
	}

	if err != nil {
		return fmt.Errorf("Error creating proposal %s: %s", chainFuncName, err)
	}

	var signedProp *pb.SignedProposal
	signedProp, err = utils.GetSignedProposal(prop, cf.Signer)
	if err != nil {
		return fmt.Errorf("Error creating signed proposal  %s: %s", chainFuncName, err)
	}

//列表当前仅支持一个对等机
	proposalResponse, err := cf.EndorserClients[0].ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return errors.Errorf("Error endorsing %s: %s", chainFuncName, err)
	}

	if proposalResponse.Response == nil {
		return errors.Errorf("Proposal response had nil 'response'")
	}

	if proposalResponse.Response.Status != int32(cb.Status_SUCCESS) {
		return errors.Errorf("Bad response: %d - %s", proposalResponse.Response.Status, proposalResponse.Response.Message)
	}

	cqr := &pb.ChaincodeQueryResponse{}
	err = proto.Unmarshal(proposalResponse.Response.Payload, cqr)
	if err != nil {
		return err
	}

	if getInstalledChaincodes {
		fmt.Println("Get installed chaincodes on peer:")
	} else {
		fmt.Printf("Get instantiated chaincodes on channel %s:\n", channelID)
	}
	for _, chaincode := range cqr.Chaincodes {
		fmt.Printf("%v\n", ccInfo{chaincode}.String())
	}
	return nil
}

type ccInfo struct {
	*pb.ChaincodeInfo
}

func (cci ccInfo) String() string {
	b := bytes.Buffer{}
	md := reflect.ValueOf(*cci.ChaincodeInfo)
	md2 := reflect.Indirect(reflect.ValueOf(*cci.ChaincodeInfo)).Type()
	for i := 0; i < md.NumField(); i++ {
		f := md.Field(i)
		val := f.String()
		if isBytes(f) {
			val = hex.EncodeToString(f.Bytes())
		}
		if len(val) == 0 {
			continue
		}
//跳过内部生成的原型字段
		if strings.HasPrefix(md2.Field(i).Name, "XXX") {
			continue
		}
		b.WriteString(fmt.Sprintf("%s: %s, ", md2.Field(i).Name, val))
	}
	return b.String()[:len(b.String())-2]

}

func isBytes(v reflect.Value) bool {
	return v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8
}
