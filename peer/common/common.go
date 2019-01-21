
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


package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/viperutil"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/scc/cscc"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/peer/common/api"
	pcommon "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//UndefinedParamValue定义命令行中未定义的参数将初始化为什么
const UndefinedParamValue = ""
const CmdRoot = "core"

var mainLogger = flogging.MustGetLogger("main")
var logOutput = os.Stderr

var (
	defaultConnTimeout = 3 * time.Second
//这些函数变量（XYZFNC）可用于调用相应的XYZ函数
//这将允许调用包在其单元测试用例中模拟这些函数。

//Get背书器Clientfnc是一个返回新背书器客户端连接的函数
//到使用tls根证书文件提供的对等地址，
//默认情况下，它设置为Get背书客户端函数
	GetEndorserClientFnc func(address, tlsRootCertFile string) (pb.EndorserClient, error)

//GetPeerDeliverClientFnc是一个返回新的传递客户端连接的函数
//到使用tls根证书文件提供的对等地址，
//默认设置为GetDeliverClient函数
	GetPeerDeliverClientFnc func(address, tlsRootCertFile string) (api.PeerDeliverClient, error)

//GetDeliverClientFnc是一个返回新的传递客户端连接的函数
//到使用tls根证书文件提供的对等地址，
//默认设置为GetDeliverClient函数
	GetDeliverClientFnc func(address, tlsRootCertFile string) (pb.Deliver_DeliverClient, error)

//GetDefaultSignerFnc是一个返回默认签名者（default/perr）的函数。
//默认设置为GetDefaultSigner函数
	GetDefaultSignerFnc func() (msp.SigningIdentity, error)

//GetBroadcastClientFnc返回BroadcastClient接口的实例
//默认设置为getBroadcastClient函数
	GetBroadcastClientFnc func() (BroadcastClient, error)

//getorderendpointtochainfnc返回给定链的排序器端点
//默认情况下，它被设置为getorderendpointtochain函数
	GetOrdererEndpointOfChainFnc func(chainID string, signer msp.SigningIdentity,
		endorserClient pb.EndorserClient) ([]string, error)

//getCertificateFnc是返回客户端TLS证书的函数
	GetCertificateFnc func() (tls.Certificate, error)
)

type commonClient struct {
	*comm.GRPCClient
	address string
	sn      string
}

func init() {
	GetEndorserClientFnc = GetEndorserClient
	GetDefaultSignerFnc = GetDefaultSigner
	GetBroadcastClientFnc = GetBroadcastClient
	GetOrdererEndpointOfChainFnc = GetOrdererEndpointOfChain
	GetDeliverClientFnc = GetDeliverClient
	GetPeerDeliverClientFnc = GetPeerDeliverClient
	GetCertificateFnc = GetCertificate
}

//initconfig初始化viper配置
func InitConfig(cmdRoot string) error {

	err := config.InitViper(nil, cmdRoot)
	if err != nil {
		return err
	}

err = viper.ReadInConfig() //查找并读取配置文件
if err != nil {            //处理读取配置文件时的错误
//我们使用的viper版本声明，当实际上找不到该文件时，不支持配置类型
//显示更有用的消息以避免混淆用户。
		if strings.Contains(fmt.Sprint(err), "Unsupported Config Type") {
			return errors.New(fmt.Sprintf("Could not find config file. "+
				"Please make sure that FABRIC_CFG_PATH is set to a path "+
				"which contains %s.yaml", cmdRoot))
		} else {
			return errors.WithMessage(err, fmt.Sprintf("error when reading %s config file", cmdRoot))
		}
	}

	return nil
}

//InitCrypto初始化此对等机的加密
func InitCrypto(mspMgrConfigDir, localMSPID, localMSPType string) error {
	var err error
//检查MSP文件夹是否存在
	fi, err := os.Stat(mspMgrConfigDir)
	if os.IsNotExist(err) || !fi.IsDir() {
//无需尝试从不可用的文件夹加载MSP
		return errors.Errorf("cannot init crypto, folder \"%s\" does not exist", mspMgrConfigDir)
	}
//检查localmspid是否存在
	if localMSPID == "" {
		return errors.New("the local MSP must have an ID")
	}

//初始化BCCSP
	SetBCCSPKeystorePath()
	var bccspConfig *factory.FactoryOpts
	err = viperutil.EnhancedExactUnmarshalKey("peer.BCCSP", &bccspConfig)
	if err != nil {
		return errors.WithMessage(err, "could not parse YAML config")
	}

	err = mspmgmt.LoadLocalMspWithType(mspMgrConfigDir, bccspConfig, localMSPID, localMSPType)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("error when setting up MSP of type %s from directory %s", localMSPType, mspMgrConfigDir))
	}

	return nil
}

//setbccspkeystorepath为sw bccsp提供程序设置文件密钥库路径
//相对于配置文件的绝对路径
func SetBCCSPKeystorePath() {
	viper.Set("peer.BCCSP.SW.FileKeyStore.KeyStore",
		config.GetPath("peer.BCCSP.SW.FileKeyStore.KeyStore"))
}

//getdefaultsigner为cli返回默认签名者（default/perr）
func GetDefaultSigner() (msp.SigningIdentity, error) {
	signer, err := mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		return nil, errors.WithMessage(err, "error obtaining the default signing identity")
	}

	return signer, err
}

//getorderEndpointToChain返回给定链的排序器终结点
func GetOrdererEndpointOfChain(chainID string, signer msp.SigningIdentity, endorserClient pb.EndorserClient) ([]string, error) {
//查询CSCC链配置块
	invocation := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]),
			ChaincodeId: &pb.ChaincodeID{Name: "cscc"},
			Input:       &pb.ChaincodeInput{Args: [][]byte{[]byte(cscc.GetConfigBlock), []byte(chainID)}},
		},
	}

	creator, err := signer.Serialize()
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("error serializing identity for %s", signer.GetIdentifier()))
	}

	prop, _, err := putils.CreateProposalFromCIS(pcommon.HeaderType_CONFIG, "", invocation, creator)
	if err != nil {
		return nil, errors.WithMessage(err, "error creating GetConfigBlock proposal")
	}

	signedProp, err := putils.GetSignedProposal(prop, signer)
	if err != nil {
		return nil, errors.WithMessage(err, "error creating signed GetConfigBlock proposal")
	}

	proposalResp, err := endorserClient.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, errors.WithMessage(err, "error endorsing GetConfigBlock")
	}

	if proposalResp == nil {
		return nil, errors.WithMessage(err, "error nil proposal response")
	}

	if proposalResp.Response.Status != 0 && proposalResp.Response.Status != 200 {
		return nil, errors.Errorf("error bad proposal response %d: %s", proposalResp.Response.Status, proposalResp.Response.Message)
	}

//分析配置块
	block, err := putils.GetBlockFromBlockBytes(proposalResp.Response.Payload)
	if err != nil {
		return nil, errors.WithMessage(err, "error unmarshaling config block")
	}

	envelopeConfig, err := putils.ExtractEnvelope(block, 0)
	if err != nil {
		return nil, errors.WithMessage(err, "error extracting config block envelope")
	}
	bundle, err := channelconfig.NewBundleFromEnvelope(envelopeConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "error loading config block")
	}

	return bundle.ChannelConfig().OrdererAddresses(), nil
}

//checkloglevel检查给定的日志级别字符串是否有效
func CheckLogLevel(level string) error {
	if !flogging.IsValidLevel(level) {
		return errors.Errorf("invalid log level provided - %s", level)
	}
	return nil
}

func configFromEnv(prefix string) (address, override string, clientConfig comm.ClientConfig, err error) {
	address = viper.GetString(prefix + ".address")
	override = viper.GetString(prefix + ".tls.serverhostoverride")
	clientConfig = comm.ClientConfig{}
	connTimeout := viper.GetDuration(prefix + ".client.connTimeout")
	if connTimeout == time.Duration(0) {
		connTimeout = defaultConnTimeout
	}
	clientConfig.Timeout = connTimeout
	secOpts := &comm.SecureOptions{
		UseTLS:            viper.GetBool(prefix + ".tls.enabled"),
		RequireClientCert: viper.GetBool(prefix + ".tls.clientAuthRequired")}
	if secOpts.UseTLS {
		caPEM, res := ioutil.ReadFile(config.GetPath(prefix + ".tls.rootcert.file"))
		if res != nil {
			err = errors.WithMessage(res,
				fmt.Sprintf("unable to load %s.tls.rootcert.file", prefix))
			return
		}
		secOpts.ServerRootCAs = [][]byte{caPEM}
	}
	if secOpts.RequireClientCert {
		keyPEM, res := ioutil.ReadFile(config.GetPath(prefix + ".tls.clientKey.file"))
		if res != nil {
			err = errors.WithMessage(res,
				fmt.Sprintf("unable to load %s.tls.clientKey.file", prefix))
			return
		}
		secOpts.Key = keyPEM
		certPEM, res := ioutil.ReadFile(config.GetPath(prefix + ".tls.clientCert.file"))
		if res != nil {
			err = errors.WithMessage(res,
				fmt.Sprintf("unable to load %s.tls.clientCert.file", prefix))
			return
		}
		secOpts.Certificate = certPEM
	}
	clientConfig.SecOpts = secOpts
	return
}

func InitCmd(cmd *cobra.Command, args []string) {
	err := InitConfig(CmdRoot)
if err != nil { //处理读取配置文件时的错误
		mainLogger.Errorf("Fatal error when initializing %s config : %s", CmdRoot, err)
		os.Exit(1)
	}

//读取旧的日志记录级别设置，如果设置，
//通知用户fabric_logging_spec env变量
	var loggingLevel string
	if viper.GetString("logging_level") != "" {
		loggingLevel = viper.GetString("logging_level")
	} else {
		loggingLevel = viper.GetString("logging.level")
	}
	if loggingLevel != "" {
		mainLogger.Warning("CORE_LOGGING_LEVEL is no longer supported, please use the FABRIC_LOGGING_SPEC environment variable")
	}

	loggingSpec := os.Getenv("FABRIC_LOGGING_SPEC")
	loggingFormat := os.Getenv("FABRIC_LOGGING_FORMAT")

	flogging.Init(flogging.Config{
		Format:  loggingFormat,
		Writer:  logOutput,
		LogSpec: loggingSpec,
	})

//初始化MSP
	var mspMgrConfigDir = config.GetPath("peer.mspConfigPath")
	var mspID = viper.GetString("peer.localMspId")
	var mspType = viper.GetString("peer.localMspType")
	if mspType == "" {
		mspType = msp.ProviderTypeToString(msp.FABRIC)
	}
	err = InitCrypto(mspMgrConfigDir, mspID, mspType)
if err != nil { //处理读取配置文件时的错误
		mainLogger.Errorf("Cannot run peer because %s", err.Error())
		os.Exit(1)
	}

	runtime.GOMAXPROCS(viper.GetInt("peer.gomaxprocs"))
}
