
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


//包cscc chaincode configer提供用于管理的功能
//正在重新配置网络时的配置事务。这个
//配置事务从订购服务到达提交者
//谁叫这个链码。链码还提供了对等配置
//连接链或获取配置数据等服务。
package cscc

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/config"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//新建创建CSCC的新实例。
//通常，每个对等实例只创建一个。
func New(ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider, aclProvider aclmgmt.ACLProvider) *PeerConfiger {
	return &PeerConfiger{
		policyChecker: policy.NewPolicyChecker(
			peer.NewChannelPolicyManagerGetter(),
			mgmt.GetLocalMSP(),
			mgmt.NewLocalMSPPrincipalGetter(),
		),
		configMgr:   peer.NewConfigSupport(),
		ccp:         ccp,
		sccp:        sccp,
		aclProvider: aclProvider,
	}
}

func (e *PeerConfiger) Name() string              { return "cscc" }
func (e *PeerConfiger) Path() string              { return "github.com/hyperledger/fabric/core/scc/cscc" }
func (e *PeerConfiger) InitArgs() [][]byte        { return nil }
func (e *PeerConfiger) Chaincode() shim.Chaincode { return e }
func (e *PeerConfiger) InvokableExternal() bool   { return true }
func (e *PeerConfiger) InvokableCC2CC() bool      { return false }
func (e *PeerConfiger) Enabled() bool             { return true }

//PeerConfiger实现对等机的配置处理程序。对于每一个
//来自订购服务的配置事务
//提交者调用此系统链码来处理事务。
type PeerConfiger struct {
	policyChecker policy.PolicyChecker
	configMgr     config.Manager
	ccp           ccprovider.ChaincodeProvider
	sccp          sysccprovider.SystemChaincodeProvider
	aclProvider   aclmgmt.ACLProvider
}

var cnflogger = flogging.MustGetLogger("cscc")

//这些是来自invoke first参数的函数名
const (
	JoinChain                string = "JoinChain"
	GetConfigBlock           string = "GetConfigBlock"
	GetChannels              string = "GetChannels"
	GetConfigTree            string = "GetConfigTree"
	SimulateConfigTreeUpdate string = "SimulateConfigTreeUpdate"
)

//从scc的角度来看，init基本上是无用的。
func (e *PeerConfiger) Init(stub shim.ChaincodeStubInterface) pb.Response {
	cnflogger.Info("Init CSCC")
	return shim.Success(nil)
}

//调用以下对象：
//处理加入链（由应用程序作为交易建议调用）
//获取当前配置块（由应用程序调用）
//更新配置块（由提交者调用）
//Peer用2个参数调用此函数：
//args[0]是函数名，必须是joinchain、getconfigblock或
//更新配置块
//args[1]是配置块，如果args[0]是joinchain或
//updateConfigBlock；否则它是链ID
//TODO:改进SCC接口以避免封送/取消封送参数
func (e *PeerConfiger) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()

	if len(args) < 1 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments, %d", len(args)))
	}

	fname := string(args[0])

	if fname != GetChannels && len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments, %d", len(args)))
	}

	cnflogger.Debugf("Invoke function: %s", fname)

//处理ACL：
//1。获取已签署的建议
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed getting signed proposal from stub: [%s]", err))
	}

	return e.InvokeNoShim(args, sp)
}

func (e *PeerConfiger) InvokeNoShim(args [][]byte, sp *pb.SignedProposal) pb.Response {
	var err error
	fname := string(args[0])

	switch fname {
	case JoinChain:
		if args[1] == nil {
			return shim.Error("Cannot join the channel <nil> configuration block provided")
		}

		block, err := utils.GetBlockFromBlockBytes(args[1])
		if err != nil {
			return shim.Error(fmt.Sprintf("Failed to reconstruct the genesis block, %s", err))
		}

		cid, err := utils.GetChainIDFromBlock(block)
		if err != nil {
			return shim.Error(fmt.Sprintf("\"JoinChain\" request failed to extract "+
				"channel id from the block due to [%s]", err))
		}

		if err := validateConfigBlock(block); err != nil {
			return shim.Error(fmt.Sprintf("\"JoinChain\" for chainID = %s failed because of validation "+
				"of configuration block, because of %s", cid, err))
		}

//2。检查本地MSP管理员策略
//TODO:一旦它将支持无链ACL，就转到ACLProvider。
		if err = e.policyChecker.CheckPolicyNoChannel(mgmt.Admins, sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s][%s]: [%s]", fname, cid, err))
		}

//如果txsfilter尚不存在，则初始化它。我们可以安全地这样做，因为
//不管怎样，这就是创世块。
		txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
		if len(txsFilter) == 0 {
//添加硬编码到valid的验证代码数组
			txsFilter = util.NewTxValidationFlagsSetValue(len(block.Data.Data), pb.TxValidationCode_VALID)
			block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
		}

		return joinChain(cid, block, e.ccp, e.sccp)
	case GetConfigBlock:
//2。核对政策
		if err = e.aclProvider.CheckACL(resources.Cscc_GetConfigBlock, string(args[1]), sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s][%s]: %s", fname, args[1], err))
		}

		return getConfigBlock(args[1])
	case GetConfigTree:
//2。核对政策
		if err = e.aclProvider.CheckACL(resources.Cscc_GetConfigTree, string(args[1]), sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s][%s]: %s", fname, args[1], err))
		}

		return e.getConfigTree(args[1])
	case SimulateConfigTreeUpdate:
//核对政策
		if err = e.aclProvider.CheckACL(resources.Cscc_SimulateConfigTreeUpdate, string(args[1]), sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s][%s]: %s", fname, args[1], err))
		}
		return e.simulateConfigTreeUpdate(args[1], args[2])
	case GetChannels:
//2。检查本地MSP成员策略
//TODO:一旦它将支持无链ACL，就转到ACLProvider。
		if err = e.policyChecker.CheckPolicyNoChannel(mgmt.Members, sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s]: %s", fname, err))
		}

		return getChannels()

	}
	return shim.Error(fmt.Sprintf("Requested function %s not found.", fname))
}

//validateConfigBlock验证配置块，以查看它何时包含有效的配置事务。
func validateConfigBlock(block *common.Block) error {
	envelopeConfig, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return errors.Errorf("Failed to %s", err)
	}

	configEnv := &common.ConfigEnvelope{}
	_, err = utils.UnmarshalEnvelopeOfType(envelopeConfig, common.HeaderType_CONFIG, configEnv)
	if err != nil {
		return errors.Errorf("Bad configuration envelope: %s", err)
	}

	if configEnv.Config == nil {
		return errors.New("Nil config envelope Config")
	}

	if configEnv.Config.ChannelGroup == nil {
		return errors.New("Nil channel group")
	}

	if configEnv.Config.ChannelGroup.Groups == nil {
		return errors.New("No channel configuration groups are available")
	}

	_, exists := configEnv.Config.ChannelGroup.Groups[channelconfig.ApplicationGroupKey]
	if !exists {
		return errors.Errorf("Invalid configuration block, missing %s "+
			"configuration group", channelconfig.ApplicationGroupKey)
	}

	return nil
}

//join chain将连接配置块中指定的链。
//因为它是第一个区块，所以它是包含构造的Genesis区块。
//对于这个链，所以我们要用这个信息更新链对象
func joinChain(chainID string, block *common.Block, ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider) pb.Response {
	if err := peer.CreateChainFromBlock(block, ccp, sccp); err != nil {
		return shim.Error(err.Error())
	}

	peer.InitChain(chainID)

	return shim.Success(nil)
}

//返回指定chainID的当前配置块。如果
//对等不属于链，返回错误
func getConfigBlock(chainID []byte) pb.Response {
	if chainID == nil {
		return shim.Error("ChainID must not be nil.")
	}
	block := peer.GetCurrConfigBlock(string(chainID))
	if block == nil {
		return shim.Error(fmt.Sprintf("Unknown chain ID, %s", string(chainID)))
	}
	blockBytes, err := utils.Marshal(block)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(blockBytes)
}

//getconfigtree返回指定chainID的当前通道配置。
//如果对等机不属于链，则返回错误
func (e *PeerConfiger) getConfigTree(chainID []byte) pb.Response {
	if chainID == nil {
		return shim.Error("Chain ID must not be nil")
	}
	channelCfg := e.configMgr.GetChannelConfig(string(chainID)).ConfigProto()
	if channelCfg == nil {
		return shim.Error(fmt.Sprintf("Unknown chain ID, %s", string(chainID)))
	}
	agCfg := &pb.ConfigTree{ChannelConfig: channelCfg}
	configBytes, err := utils.Marshal(agCfg)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(configBytes)
}

func (e *PeerConfiger) simulateConfigTreeUpdate(chainID []byte, envb []byte) pb.Response {
	if chainID == nil {
		return shim.Error("Chain ID must not be nil")
	}
	if envb == nil {
		return shim.Error("Config delta bytes must not be nil")
	}
	env := &common.Envelope{}
	err := proto.Unmarshal(envb, env)
	if err != nil {
		return shim.Error(err.Error())
	}
	cfg, err := supportByType(e, chainID, env)
	if err != nil {
		return shim.Error(err.Error())
	}
	_, err = cfg.ProposeConfigUpdate(env)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte("Simulation is successful"))
}

func supportByType(pc *PeerConfiger, chainID []byte, env *common.Envelope) (config.Config, error) {
	payload := &common.Payload{}

	if err := proto.Unmarshal(env.Payload, payload); err != nil {
		return nil, errors.Errorf("failed unmarshaling payload: %v", err)
	}

	channelHdr := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, channelHdr); err != nil {
		return nil, errors.Errorf("failed unmarshaling payload header: %v", err)
	}

	switch common.HeaderType(channelHdr.Type) {
	case common.HeaderType_CONFIG_UPDATE:
		return pc.configMgr.GetChannelConfig(string(chainID)), nil
	}
	return nil, errors.Errorf("invalid payload header type: %d", channelHdr.Type)
}

//getchannels返回有关此对等方的所有通道的信息
func getChannels() pb.Response {
	channelInfoArray := peer.GetChannelsInfo()

//添加包含此对等方所有通道信息的数组
	cqr := &pb.ChannelQueryResponse{Channels: channelInfoArray}

	cqrbytes, err := proto.Marshal(cqr)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(cqrbytes)
}
