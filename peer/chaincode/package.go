
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


package chaincode

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/common/ccpackage"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	pcommon "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/cobra"
)

var chaincodePackageCmd *cobra.Command
var createSignedCCDepSpec bool
var signCCDepSpec bool
var instantiationPolicy string

const packageCmdName = "package"
const packageDesc = "Package the specified chaincode into a deployment spec."

type ccDepSpecFactory func(spec *pb.ChaincodeSpec) (*pb.ChaincodeDeploymentSpec, error)

func defaultCDSFactory(spec *pb.ChaincodeSpec) (*pb.ChaincodeDeploymentSpec, error) {
	return getChaincodeDeploymentSpec(spec, true)
}

//deployCmd返回chaincode deploy的cobra命令
func packageCmd(cf *ChaincodeCmdFactory, cdsFact ccDepSpecFactory) *cobra.Command {
	chaincodePackageCmd = &cobra.Command{
		Use:       "package",
		Short:     packageDesc,
		Long:      packageDesc,
		ValidArgs: []string{"1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("output file not specified or invalid number of args (filename should be the only arg)")
			}
//UT将提供自己的模拟工厂
			if cdsFact == nil {
				cdsFact = defaultCDSFactory
			}
			return chaincodePackage(cmd, args, cdsFact, cf)
		},
	}
	flagList := []string{
		"lang",
		"ctor",
		"path",
		"name",
		"version",
	}
	attachFlags(chaincodePackageCmd, flagList)

	chaincodePackageCmd.Flags().BoolVarP(&createSignedCCDepSpec, "cc-package", "s", false, "create CC deployment spec for owner endorsements instead of raw CC deployment spec")
	chaincodePackageCmd.Flags().BoolVarP(&signCCDepSpec, "sign", "S", false, "if creating CC deployment spec package for owner endorsements, also sign it with local MSP")
	chaincodePackageCmd.Flags().StringVarP(&instantiationPolicy, "instantiate-policy", "i", "", "instantiation policy for the chaincode")

	return chaincodePackageCmd
}

func getInstantiationPolicy(policy string) (*pcommon.SignaturePolicyEnvelope, error) {
	p, err := cauthdsl.FromString(policy)
	if err != nil {
		return nil, fmt.Errorf("Invalid policy %s, err %s", policy, err)
	}
	return p, nil
}

//getchaincodeinstallpackage返回原始chaincodedeploymentspec或
//带有chaincodedeploymentspec和（可选）签名的信封
func getChaincodeInstallPackage(cds *pb.ChaincodeDeploymentSpec, cf *ChaincodeCmdFactory) ([]byte, error) {
//这可以是原始的chaincodedeploymentspec或带有签名的信封
	var objToWrite proto.Message

//从默认CD开始
	objToWrite = cds

	var err error

	var owner msp.SigningIdentity

//创建链码包…
	if createSignedCCDepSpec {
//…还可以选择获取签名者以便对包进行签名
//由当地MSP提供。这个包裹可以给其他车主
//使用“peer chaincode sign<package file>进行签名
		if signCCDepSpec {
			if cf.Signer == nil {
				return nil, fmt.Errorf("Error getting signer")
			}
			owner = cf.Signer
		}
	}

	ip := instantiationPolicy
	if ip == "" {
//如果未提供实例化策略，则默认
//至“管理员必须签署链码实例化建议”
		mspid, err := mspmgmt.GetLocalMSP().GetIdentifier()
		if err != nil {
			return nil, err
		}
		ip = "AND('" + mspid + ".admin')"
	}

	sp, err := getInstantiationPolicy(ip)
	if err != nil {
		return nil, err
	}

//我们得到chaincode_包类型的信封
	objToWrite, err = ccpackage.OwnerCreateSignedCCDepSpec(cds, sp, owner)
	if err != nil {
		return nil, err
	}

//将proto对象转换为字节
	bytesToWrite, err := proto.Marshal(objToWrite)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling chaincode package : %s", err)
	}

	return bytesToWrite, nil
}

//chaincode package创建chaincode包。成功后，链代码名称
//（hash）被打印到stdout，以供后续与chaincode相关的cli使用。
//命令。
func chaincodePackage(cmd *cobra.Command, args []string, cdsFact ccDepSpecFactory, cf *ChaincodeCmdFactory) error {
	if cdsFact == nil {
		return fmt.Errorf("Error chaincode deployment spec factory not specified")
	}
//对命令行的分析已完成，因此沉默命令用法
	cmd.SilenceUsage = true

	var err error
	if cf == nil {
		cf, err = InitCmdFactory(cmd.Name(), false, false)
		if err != nil {
			return err
		}
	}
	spec, err := getChaincodeSpec(cmd)
	if err != nil {
		return err
	}

	cds, err := cdsFact(spec)
	if err != nil {
		return fmt.Errorf("error getting chaincode code %s: %s", chaincodeName, err)
	}

	var bytesToWrite []byte
	if createSignedCCDepSpec {
		bytesToWrite, err = getChaincodeInstallPackage(cds, cf)
		if err != nil {
			return err
		}
	} else {
		bytesToWrite = utils.MarshalOrPanic(cds)
	}

	logger.Debugf("Packaged chaincode into deployment spec of size <%d>, with args = %v", len(bytesToWrite), args)
	fileToWrite := args[0]
	err = ioutil.WriteFile(fileToWrite, bytesToWrite, 0700)
	if err != nil {
		logger.Errorf("failed writing deployment spec to file [%s]: [%s]", fileToWrite, err)
		return err
	}

	return err
}
