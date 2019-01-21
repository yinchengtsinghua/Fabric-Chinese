
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
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/ccpackage"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/peer/common"
	pcommon "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/cobra"
)

var chaincodeInstallCmd *cobra.Command

const installCmdName = "install"

const installDesc = "Package the specified chaincode into a deployment spec and save it on the peer's path."

//installCmd返回chaincode deploy的cobra命令
func installCmd(cf *ChaincodeCmdFactory) *cobra.Command {
	chaincodeInstallCmd = &cobra.Command{
		Use:       "install",
		Short:     fmt.Sprint(installDesc),
		Long:      fmt.Sprint(installDesc),
		ValidArgs: []string{"1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var ccpackfile string
			if len(args) > 0 {
				ccpackfile = args[0]
			}
			return chaincodeInstall(cmd, ccpackfile, cf)
		},
	}
	flagList := []string{
		"lang",
		"ctor",
		"path",
		"name",
		"version",
		"peerAddresses",
		"tlsRootCertFiles",
		"connectionProfile",
	}
	attachFlags(chaincodeInstallCmd, flagList)

	return chaincodeInstallCmd
}

//将depspec安装到“peer.address”
func install(msg proto.Message, cf *ChaincodeCmdFactory) error {
	creator, err := cf.Signer.Serialize()
	if err != nil {
		return fmt.Errorf("Error serializing identity for %s: %s", cf.Signer.GetIdentifier(), err)
	}

	prop, _, err := utils.CreateInstallProposalFromCDS(msg, creator)
	if err != nil {
		return fmt.Errorf("Error creating proposal  %s: %s", chainFuncName, err)
	}

	var signedProp *pb.SignedProposal
	signedProp, err = utils.GetSignedProposal(prop, cf.Signer)
	if err != nil {
		return fmt.Errorf("Error creating signed proposal  %s: %s", chainFuncName, err)
	}

//当前只支持一个对等机安装
	proposalResponse, err := cf.EndorserClients[0].ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return fmt.Errorf("Error endorsing %s: %s", chainFuncName, err)
	}

	if proposalResponse != nil {
		if proposalResponse.Response.Status != int32(pcommon.Status_SUCCESS) {
			return errors.Errorf("Bad response: %d - %s", proposalResponse.Response.Status, proposalResponse.Response.Message)
		}
		logger.Infof("Installed remotely %v", proposalResponse)
	} else {
		return errors.New("Error during install: received nil proposal response")
	}

	return nil
}

//genchaincodedeploymentspec创建chaincodedeploymentspec作为要安装的包
func genChaincodeDeploymentSpec(cmd *cobra.Command, chaincodeName, chaincodeVersion string) (*pb.ChaincodeDeploymentSpec, error) {
	if existed, _ := ccprovider.ChaincodePackageExists(chaincodeName, chaincodeVersion); existed {
		return nil, fmt.Errorf("chaincode %s:%s already exists", chaincodeName, chaincodeVersion)
	}

	spec, err := getChaincodeSpec(cmd)
	if err != nil {
		return nil, err
	}

	cds, err := getChaincodeDeploymentSpec(spec, true)
	if err != nil {
		return nil, fmt.Errorf("error getting chaincode code %s: %s", chaincodeName, err)
	}

	return cds, nil
}

//get package from file从文件和提取的chaincodedeploymentspec获取chaincode包
func getPackageFromFile(ccpackfile string) (proto.Message, *pb.ChaincodeDeploymentSpec, error) {
	b, err := ioutil.ReadFile(ccpackfile)
	if err != nil {
		return nil, nil, err
	}

//字节应该是有效的包（cd或signedcds）
	ccpack, err := ccprovider.GetCCPackage(b)
	if err != nil {
		return nil, nil, err
	}

//CD或信封
	o := ccpack.GetPackageObject()

//先试用光盘
	cds, ok := o.(*pb.ChaincodeDeploymentSpec)
	if !ok || cds == nil {
//下一个试试信封
		env, ok := o.(*pcommon.Envelope)
		if !ok || env == nil {
			return nil, nil, fmt.Errorf("error extracting valid chaincode package")
		}

//这将检查有效的包裹信封
		_, sCDS, err := ccpackage.ExtractSignedCCDepSpec(env)
		if err != nil {
			return nil, nil, fmt.Errorf("error extracting valid signed chaincode package(%s)", err)
		}

//…最后拿到CD
		cds, err = utils.GetChaincodeDeploymentSpec(sCDS.ChaincodeDeploymentSpec, platformRegistry)
		if err != nil {
			return nil, nil, fmt.Errorf("error extracting chaincode deployment spec(%s)", err)
		}
	}

	return o, cds, nil
}

//chaincodeinstall安装chaincode。如果是远程安装，是否通过LSCC调用
func chaincodeInstall(cmd *cobra.Command, ccpackfile string, cf *ChaincodeCmdFactory) error {
//对命令行的分析已完成，因此沉默命令用法
	cmd.SilenceUsage = true

	var err error
	if cf == nil {
		cf, err = InitCmdFactory(cmd.Name(), true, false)
		if err != nil {
			return err
		}
	}

	var ccpackmsg proto.Message
	if ccpackfile == "" {
		if chaincodePath == common.UndefinedParamValue || chaincodeVersion == common.UndefinedParamValue || chaincodeName == common.UndefinedParamValue {
			return fmt.Errorf("Must supply value for %s name, path and version parameters.", chainFuncName)
		}
//生成原始chaincodedeploymentspec
		ccpackmsg, err = genChaincodeDeploymentSpec(cmd, chaincodeName, chaincodeVersion)
		if err != nil {
			return err
		}
	} else {
//读取由“package”子命令生成的包（可能已签名
//由多个拥有者使用“signpackage”子命令）
		var cds *pb.ChaincodeDeploymentSpec
		ccpackmsg, cds, err = getPackageFromFile(ccpackfile)

		if err != nil {
			return err
		}

//从CD获取链码详细信息
		cName := cds.ChaincodeSpec.ChaincodeId.Name
		cVersion := cds.ChaincodeSpec.ChaincodeId.Version

//如果用户提供了chaincodename，请使用它进行验证
		if chaincodeName != "" && chaincodeName != cName {
			return fmt.Errorf("chaincode name %s does not match name %s in package", chaincodeName, cName)
		}

//如果用户提供了chaincodeversion，请使用它进行验证
		if chaincodeVersion != "" && chaincodeVersion != cVersion {
			return fmt.Errorf("chaincode version %s does not match version %s in packages", chaincodeVersion, cVersion)
		}
	}

	err = install(ccpackmsg, cf)

	return err
}
