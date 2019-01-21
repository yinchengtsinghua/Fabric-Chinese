
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
	"fmt"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/car"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/chaincode/platforms/java"
	"github.com/hyperledger/fabric/core/chaincode/platforms/node"
	"github.com/hyperledger/fabric/peer/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	chainFuncName = "chaincode"
	chainCmdDes   = "Operate a chaincode: install|instantiate|invoke|package|query|signpackage|upgrade|list."
)

var logger = flogging.MustGetLogger("chaincodeCmd")

//但是，这是一个可怕的单打黑客。
//它只是使一个潜在的依赖显式。
//它应该和另一个包装一起拆下
//范围变量
var platformRegistry = platforms.NewRegistry(
	&golang.Platform{},
	&car.Platform{},
	&java.Platform{},
	&node.Platform{},
)

func addFlags(cmd *cobra.Command) {
	common.AddOrdererFlags(cmd)
	flags := cmd.PersistentFlags()
	flags.StringVarP(&transient, "transient", "", "", "Transient map of arguments in JSON encoding")
}

//cmd返回chaincode的cobra命令
func Cmd(cf *ChaincodeCmdFactory) *cobra.Command {
	addFlags(chaincodeCmd)

	chaincodeCmd.AddCommand(installCmd(cf))
	chaincodeCmd.AddCommand(instantiateCmd(cf))
	chaincodeCmd.AddCommand(invokeCmd(cf))
	chaincodeCmd.AddCommand(packageCmd(cf, nil))
	chaincodeCmd.AddCommand(queryCmd(cf))
	chaincodeCmd.AddCommand(signpackageCmd(cf))
	chaincodeCmd.AddCommand(upgradeCmd(cf))
	chaincodeCmd.AddCommand(listCmd(cf))

	return chaincodeCmd
}

//链码相关变量。
var (
	chaincodeLang         string
	chaincodeCtorJSON     string
	chaincodePath         string
	chaincodeName         string
chaincodeUsr          string //不使用
	chaincodeQueryRaw     bool
	chaincodeQueryHex     bool
	channelID             string
	chaincodeVersion      string
	policy                string
	escc                  string
	vscc                  string
	policyMarshalled      []byte
	transient             string
	collectionsConfigFile string
	collectionConfigBytes []byte
	peerAddresses         []string
	tlsRootCertFiles      []string
	connectionProfile     string
	waitForEvent          bool
	waitForEventTimeout   time.Duration
)

var chaincodeCmd = &cobra.Command{
	Use:   chainFuncName,
	Short: fmt.Sprint(chainCmdDes),
	Long:  fmt.Sprint(chainCmdDes),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		common.InitCmd(cmd, args)
		common.SetOrdererEnv(cmd, args)
	},
}

var flags *pflag.FlagSet

func init() {
	resetFlags()
}

//明确定义一种方法来促进测试
func resetFlags() {
	flags = &pflag.FlagSet{}

	flags.StringVarP(&chaincodeLang, "lang", "l", "golang",
		fmt.Sprintf("Language the %s is written in", chainFuncName))
	flags.StringVarP(&chaincodeCtorJSON, "ctor", "c", "{}",
		fmt.Sprintf("Constructor message for the %s in JSON format", chainFuncName))
	flags.StringVarP(&chaincodePath, "path", "p", common.UndefinedParamValue,
		fmt.Sprintf("Path to %s", chainFuncName))
	flags.StringVarP(&chaincodeName, "name", "n", common.UndefinedParamValue,
		fmt.Sprint("Name of the chaincode"))
	flags.StringVarP(&chaincodeVersion, "version", "v", common.UndefinedParamValue,
		fmt.Sprint("Version of the chaincode specified in install/instantiate/upgrade commands"))
	flags.StringVarP(&chaincodeUsr, "username", "u", common.UndefinedParamValue,
		fmt.Sprint("Username for chaincode operations when security is enabled"))
	flags.StringVarP(&channelID, "channelID", "C", "",
		fmt.Sprint("The channel on which this command should be executed"))
	flags.StringVarP(&policy, "policy", "P", common.UndefinedParamValue,
		fmt.Sprint("The endorsement policy associated to this chaincode"))
	flags.StringVarP(&escc, "escc", "E", common.UndefinedParamValue,
		fmt.Sprint("The name of the endorsement system chaincode to be used for this chaincode"))
	flags.StringVarP(&vscc, "vscc", "V", common.UndefinedParamValue,
		fmt.Sprint("The name of the verification system chaincode to be used for this chaincode"))
	flags.BoolVarP(&getInstalledChaincodes, "installed", "", false,
		"Get the installed chaincodes on a peer")
	flags.BoolVarP(&getInstantiatedChaincodes, "instantiated", "", false,
		"Get the instantiated chaincodes on a channel")
	flags.StringVar(&collectionsConfigFile, "collections-config", common.UndefinedParamValue,
		fmt.Sprint("The fully qualified path to the collection JSON file including the file name"))
	flags.StringArrayVarP(&peerAddresses, "peerAddresses", "", []string{common.UndefinedParamValue},
		fmt.Sprint("The addresses of the peers to connect to"))
	flags.StringArrayVarP(&tlsRootCertFiles, "tlsRootCertFiles", "", []string{common.UndefinedParamValue},
		fmt.Sprint("If TLS is enabled, the paths to the TLS root cert files of the peers to connect to. The order and number of certs specified should match the --peerAddresses flag"))
	flags.StringVarP(&connectionProfile, "connectionProfile", "", common.UndefinedParamValue,
		fmt.Sprint("Connection profile that provides the necessary connection information for the network. Note: currently only supported for providing peer connection information"))
	flags.BoolVar(&waitForEvent, "waitForEvent", false,
		fmt.Sprint("Whether to wait for the event from each peer's deliver filtered service signifying that the 'invoke' transaction has been committed successfully"))
	flags.DurationVar(&waitForEventTimeout, "waitForEventTimeout", 30*time.Second,
		fmt.Sprint("Time to wait for the event from each peer's deliver filtered service signifying that the 'invoke' transaction has been committed successfully"))
}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		} else {
			logger.Fatalf("Could not find flag '%s' to attach to command '%s'", name, cmd.Name())
		}
	}
}
