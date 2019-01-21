
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package endorser

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/validation"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/transientstore"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var endorserLogger = flogging.MustGetLogger("endorser")

//JIRA发行的文件背书人的流动及其与
//生命周期链代码-https://jira.hyperledger.org/browse/fab-181

type privateDataDistributor func(channel string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error

//支持包含背书人执行其任务所需的功能
type Support interface {
	crypto.SignerSupport
//如果提供的链码为
//IA系统链码，不可调用
	IsSysCCAndNotInvokableExternal(name string) bool

//GetTxSimulator返回指定分类帐的事务模拟器
//客户可以获得多个这样的模拟器；它们是独一无二的
//通过提供的txid
	GetTxSimulator(ledgername string, txid string) (ledger.TxSimulator, error)

//GetHistoryQueryExecutor为
//指定的分类帐
	GetHistoryQueryExecutor(ledgername string) (ledger.HistoryQueryExecutor, error)

//GetTransactionByID按ID检索事务
	GetTransactionByID(chid, txID string) (*pb.ProcessedTransaction, error)

//如果名称与系统链码匹配，IsSyscc将返回true
//系统链代码名称为系统、链范围
	IsSysCC(name string) bool

//执行-执行建议，返回链码的原始响应
	Execute(txParams *ccprovider.TransactionParams, cid, name, version, txid string, signedProp *pb.SignedProposal, prop *pb.Proposal, input *pb.ChaincodeInput) (*pb.Response, *pb.ChaincodeEvent, error)

//executeLegacyInit-执行部署建议，返回链代码的原始响应
	ExecuteLegacyInit(txParams *ccprovider.TransactionParams, cid, name, version, txid string, signedProp *pb.SignedProposal, prop *pb.Proposal, spec *pb.ChaincodeDeploymentSpec) (*pb.Response, *pb.ChaincodeEvent, error)

//getchaincodedefinition返回具有所提供名称的链代码的ccprovider.chaincodedefinition
	GetChaincodeDefinition(chaincodeID string, txsim ledger.QueryExecutor) (ccprovider.ChaincodeDefinition, error)

//checkacl使用
//可从中提取ID以针对策略进行测试的已签名的路径
	CheckACL(signedProp *pb.SignedProposal, chdr *common.ChannelHeader, shdr *common.SignatureHeader, hdrext *pb.ChaincodeHeaderExtension) error

//如果cds包字节描述链码，isjavacc返回true
//这要求Java运行时环境执行。
	IsJavaCC(buf []byte) (bool, error)

//如果提供的
//chaincodedefinition与存储在分类帐上的实例化策略不同
	CheckInstantiationPolicy(name, version string, cd ccprovider.ChaincodeDefinition) error

//getchaincodedeploymentspecfs返回fs中链码的deploymentspec
	GetChaincodeDeploymentSpecFS(cds *pb.ChaincodeDeploymentSpec) (*pb.ChaincodeDeploymentSpec, error)

//GetApplicationConfig返回通道的configTxApplication.sharedConfig
//以及应用程序配置是否存在
	GetApplicationConfig(cid string) (channelconfig.Application, bool)

//NewQueryCreator创建新的QueryCreator
	NewQueryCreator(channel string) (QueryCreator, error)

//背书WithPlugin用插件背书响应
	EndorseWithPlugin(ctx Context) (*pb.ProposalResponse, error)

//GetLedgerHeight返回给定ChannelID的分类帐高度
	GetLedgerHeight(channelID string) (uint64, error)
}

//背书人提供背书人服务流程建议书
type Endorser struct {
	distributePrivateData privateDataDistributor
	s                     Support
	PlatformRegistry      *platforms.Registry
	PvtRWSetAssembler
	Metrics *EndorserMetrics
}

//validateResult提供认可的验证结果
type validateResult struct {
	prop    *pb.Proposal
	hdrExt  *pb.ChaincodeHeaderExtension
	chainID string
	txid    string
	resp    *pb.ProposalResponse
}

//New背书服务器创建并返回一个新的背书服务器实例。
func NewEndorserServer(privDist privateDataDistributor, s Support, pr *platforms.Registry, metricsProv metrics.Provider) *Endorser {
	e := &Endorser{
		distributePrivateData: privDist,
		s:                     s,
		PlatformRegistry:      pr,
		PvtRWSetAssembler:     &rwSetAssembler{},
		Metrics:               NewEndorserMetrics(metricsProv),
	}
	return e
}

//调用指定的链码（系统或用户）
func (e *Endorser) callChaincode(txParams *ccprovider.TransactionParams, version string, input *pb.ChaincodeInput, cid *pb.ChaincodeID) (*pb.Response, *pb.ChaincodeEvent, error) {
	endorserLogger.Infof("[%s][%s] Entry chaincode: %s", txParams.ChannelID, shorttxid(txParams.TxID), cid)
	defer func(start time.Time) {
		logger := endorserLogger.WithOptions(zap.AddCallerSkip(1))
		elapsedMilliseconds := time.Since(start).Round(time.Millisecond) / time.Millisecond
		logger.Infof("[%s][%s] Exit chaincode: %s (%dms)", txParams.ChannelID, shorttxid(txParams.TxID), cid, elapsedMilliseconds)
	}(time.Now())

	var err error
	var res *pb.Response
	var ccevent *pb.ChaincodeEvent

//这是系统链码吗
	res, ccevent, err = e.s.Execute(txParams, txParams.ChannelID, cid.Name, version, txParams.TxID, txParams.SignedProp, txParams.Proposal, input)
	if err != nil {
		return nil, nil, err
	}

//根据文件，任何小于400的信息都可以作为Tx发送。
//结构错误将始终大于等于400（即，明确的错误）
//“LSCC”将以状态200或500响应（即，明确的“OK”或“错误”）。
	if res.Status >= shim.ERRORTHRESHOLD {
		return res, nil, nil
	}

//-----开始-可能需要在LSCC中完成的部分------
//如果这是部署链代码的调用，我们需要一个机制
//将TXSimulator传递到LSCC。直到解决这个问题
//特殊代码在这里进行实际部署、升级以收集
//一个TXSimulator下的所有状态
//
//注意，如果有错误，所有模拟，包括链码
//LSCC中的表更改将被丢弃
	if cid.Name == "lscc" && len(input.Args) >= 3 && (string(input.Args[0]) == "deploy" || string(input.Args[0]) == "upgrade") {
		userCDS, err := putils.GetChaincodeDeploymentSpec(input.Args[2], e.PlatformRegistry)
		if err != nil {
			return nil, nil, err
		}

		var cds *pb.ChaincodeDeploymentSpec
		cds, err = e.SanitizeUserCDS(userCDS)
		if err != nil {
			return nil, nil, err
		}

//这不应该是系统链代码
		if e.s.IsSysCC(cds.ChaincodeSpec.ChaincodeId.Name) {
			return nil, nil, errors.Errorf("attempting to deploy a system chaincode %s/%s", cds.ChaincodeSpec.ChaincodeId.Name, txParams.ChannelID)
		}

		_, _, err = e.s.ExecuteLegacyInit(txParams, txParams.ChannelID, cds.ChaincodeSpec.ChaincodeId.Name, cds.ChaincodeSpec.ChaincodeId.Version, txParams.TxID, txParams.SignedProp, txParams.Proposal, cds)
		if err != nil {
//递增失败以指示实例化/升级失败
			meterLabels := []string{
				"channel", txParams.ChannelID,
				"chaincode", cds.ChaincodeSpec.ChaincodeId.Name + ":" + cds.ChaincodeSpec.ChaincodeId.Version,
			}
			e.Metrics.InitFailed.With(meterLabels...).Add(1)
			return nil, nil, err
		}
	}
//-----结束------

	return res, ccevent, err
}

func (e *Endorser) SanitizeUserCDS(userCDS *pb.ChaincodeDeploymentSpec) (*pb.ChaincodeDeploymentSpec, error) {
	fsCDS, err := e.s.GetChaincodeDeploymentSpecFS(userCDS)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot deploy a chaincode which is not installed")
	}

	sanitizedCDS := proto.Clone(fsCDS).(*pb.ChaincodeDeploymentSpec)
	sanitizedCDS.CodePackage = nil
	sanitizedCDS.ChaincodeSpec.Input = userCDS.ChaincodeSpec.Input

	return sanitizedCDS, nil
}

//SimulateToposal通过调用链代码来模拟提案
func (e *Endorser) SimulateProposal(txParams *ccprovider.TransactionParams, cid *pb.ChaincodeID) (ccprovider.ChaincodeDefinition, *pb.Response, []byte, *pb.ChaincodeEvent, error) {
	endorserLogger.Debugf("[%s][%s] Entry chaincode: %s", txParams.ChannelID, shorttxid(txParams.TxID), cid)
	defer endorserLogger.Debugf("[%s][%s] Exit", txParams.ChannelID, shorttxid(txParams.TxID))
//我们希望有效载荷是一个chaincodeinvocationspec
//如果我们在未来支持其他有效载荷，这是非常明显的一点。
//作为应该改变的东西
	cis, err := putils.GetChaincodeInvocationSpec(txParams.Proposal)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var cdLedger ccprovider.ChaincodeDefinition
	var version string

	if !e.s.IsSysCC(cid.Name) {
		cdLedger, err = e.s.GetChaincodeDefinition(cid.Name, txParams.TXSimulator)
		if err != nil {
			return nil, nil, nil, nil, errors.WithMessage(err, fmt.Sprintf("make sure the chaincode %s has been successfully instantiated and try again", cid.Name))
		}
		version = cdLedger.CCVersion()

		err = e.s.CheckInstantiationPolicy(cid.Name, version, cdLedger)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		version = util.GetSysCCVersion()
	}

//3。执行建议并获得模拟结果
	var simResult *ledger.TxSimulationResults
	var pubSimResBytes []byte
	var res *pb.Response
	var ccevent *pb.ChaincodeEvent
	res, ccevent, err = e.callChaincode(txParams, version, cis.ChaincodeSpec.Input, cid)
	if err != nil {
		endorserLogger.Errorf("[%s][%s] failed to invoke chaincode %s, error: %+v", txParams.ChannelID, shorttxid(txParams.TxID), cid, err)
		return nil, nil, nil, nil, err
	}

	if txParams.TXSimulator != nil {
		if simResult, err = txParams.TXSimulator.GetTxSimulationResults(); err != nil {
			txParams.TXSimulator.Done()
			return nil, nil, nil, nil, err
		}

		if simResult.PvtSimulationResults != nil {
			if cid.Name == "lscc" {
//TODO:一旦可以在lscc外部存储集合配置，就将其移除
				txParams.TXSimulator.Done()
				return nil, nil, nil, nil, errors.New("Private data is forbidden to be used in instantiate")
			}
			pvtDataWithConfig, err := e.AssemblePvtRWSet(simResult.PvtSimulationResults, txParams.TXSimulator)
//若要读取集合，配置需要先读取集合更新
//释放锁，因此txparams.txsimulator.done（）移到这里
			txParams.TXSimulator.Done()

			if err != nil {
				return nil, nil, nil, nil, errors.WithMessage(err, "failed to obtain collections config")
			}
			endorsedAt, err := e.s.GetLedgerHeight(txParams.ChannelID)
			if err != nil {
				return nil, nil, nil, nil, errors.WithMessage(err, fmt.Sprint("failed to obtain ledger height for channel", txParams.ChannelID))
			}
//添加交易记录背书时的分类帐高度，
//“背书日期”是从块存储中获得的，有时可能是“背书高度+1”。
//但是，因为我们只使用这个高度来选择配置（distributeprivatedata中的第三个参数），并且
//管理孤立的私有写集（distributeprivatedata中的第四个参数）的临时存储清除，这暂时有效。
//理想情况下，Ledger应该在模拟器中作为第一类函数“getheight（）”添加支持。
			pvtDataWithConfig.EndorsedAt = endorsedAt
			if err := e.distributePrivateData(txParams.ChannelID, txParams.TxID, pvtDataWithConfig, endorsedAt); err != nil {
				return nil, nil, nil, nil, err
			}
		}

		txParams.TXSimulator.Done()
		if pubSimResBytes, err = simResult.GetPubSimulationBytes(); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	return cdLedger, res, pubSimResBytes, ccevent, nil
}

//通过致电亚太经社会批准该提案
func (e *Endorser) endorseProposal(_ context.Context, chainID string, txid string, signedProp *pb.SignedProposal, proposal *pb.Proposal, response *pb.Response, simRes []byte, event *pb.ChaincodeEvent, visibility []byte, ccid *pb.ChaincodeID, txsim ledger.TxSimulator, cd ccprovider.ChaincodeDefinition) (*pb.ProposalResponse, error) {
	endorserLogger.Debugf("[%s][%s] Entry chaincode: %s", chainID, shorttxid(txid), ccid)
	defer endorserLogger.Debugf("[%s][%s] Exit", chainID, shorttxid(txid))

	isSysCC := cd == nil
//1）提取需要签署此链码的ESCC的名称。
	var escc string
//即“LSCC”或系统链码
	if isSysCC {
		escc = "escc"
	} else {
		escc = cd.Endorsement()
	}

	endorserLogger.Debugf("[%s][%s] escc for chaincode %s is %s", chainID, shorttxid(txid), ccid, escc)

//封送事件字节
	var err error
	var eventBytes []byte
	if event != nil {
		eventBytes, err = putils.GetBytesChaincodeEvent(event)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal event bytes")
		}
	}

//设置执行链码的版本
	if isSysCC {
//如果我们想要混合面料，我们应该
//将Syscc版本设置为“”。
		ccid.Version = util.GetSysCCVersion()
	} else {
		ccid.Version = cd.CCVersion()
	}

	ctx := Context{
		PluginName:     escc,
		Channel:        chainID,
		SignedProposal: signedProp,
		ChaincodeID:    ccid,
		Event:          eventBytes,
		SimRes:         simRes,
		Response:       response,
		Visibility:     visibility,
		Proposal:       proposal,
		TxID:           txid,
	}
	return e.s.EndorseWithPlugin(ctx)
}

//预处理检查tx建议头、唯一性和acl
func (e *Endorser) preProcess(signedProp *pb.SignedProposal) (*validateResult, error) {
	vr := &validateResult{}
//首先，我们检查消息是否有效
	prop, hdr, hdrExt, err := validation.ValidateProposalMessage(signedProp)

	if err != nil {
		e.Metrics.ProposalValidationFailed.Add(1)
		vr.resp = &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}
		return vr, err
	}

	chdr, err := putils.UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		vr.resp = &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}
		return vr, err
	}

	shdr, err := putils.GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		vr.resp = &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}
		return vr, err
	}

//对安全敏感系统链码的块调用
	if e.s.IsSysCCAndNotInvokableExternal(hdrExt.ChaincodeId.Name) {
		endorserLogger.Errorf("Error: an attempt was made by %#v to invoke system chaincode %s", shdr.Creator, hdrExt.ChaincodeId.Name)
		err = errors.Errorf("chaincode %s cannot be invoked through a proposal", hdrExt.ChaincodeId.Name)
		vr.resp = &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}
		return vr, err
	}

	chainID := chdr.ChannelId
	txid := chdr.TxId
	endorserLogger.Debugf("[%s][%s] processing txid: %s", chainID, shorttxid(txid), txid)

	if chainID != "" {
//为故障度量提供上下文的标签
		meterLabels := []string{
			"channel", chainID,
			"chaincode", hdrExt.ChaincodeId.Name + ":" + hdrExt.ChaincodeId.Version,
		}

//在这里，我们处理针对一个链的提案的唯一性检查和ACL。
//请注意，validateProposalMessage已验证TxID的计算是否正确。
		if _, err = e.s.GetTransactionByID(chainID, txid); err == nil {
//由于事务重复而导致增量失败。有助于在
//除良性重试外
			e.Metrics.DuplicateTxsFailure.With(meterLabels...).Add(1)
			err = errors.Errorf("duplicate transaction found [%s]. Creator [%x]", txid, shdr.Creator)
			vr.resp = &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}
			return vr, err
		}

//仅检查应用程序链代码的acl；acls
//在其他地方检查系统链码
		if !e.s.IsSysCC(hdrExt.ChaincodeId.Name) {
//检查提案是否符合频道作者的要求
			if err = e.s.CheckACL(signedProp, chdr, shdr, hdrExt); err != nil {
				e.Metrics.ProposalACLCheckFailed.With(meterLabels...).Add(1)
				vr.resp = &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}
				return vr, err
			}
		}
	} else {
//无链建议不影响/不能影响分类帐，不能作为交易提交。
//忽略唯一性检查；另外，没有链接的建议不会使用策略进行验证。
//链的，因为根据定义，没有链；它们是根据本地
//通过调用上面的validateProposalMessage来代替对等机的MSP
	}

	vr.prop, vr.hdrExt, vr.chainID, vr.txid = prop, hdrExt, chainID, txid
	return vr, nil
}

//处理建议处理建议
func (e *Endorser) ProcessProposal(ctx context.Context, signedProp *pb.SignedProposal) (*pb.ProposalResponse, error) {
//计算成功批准的建议的运行时间度量的开始时间
	startTime := time.Now()
	e.Metrics.ProposalsReceived.Add(1)

	addr := util.ExtractRemoteAddress(ctx)
	endorserLogger.Debug("Entering: request from", addr)

//用于捕获建议持续时间度量的变量
	var chainID string
	var hdrExt *pb.ChaincodeHeaderExtension
	var success bool
	defer func() {
//捕获建议持续时间度量。hdrex==nil表示早期故障
//我们不捕获延迟度量。但是提议失败了
//反度量应该能揭示这些失败。
		if hdrExt != nil {
			meterLabels := []string{
				"channel", chainID,
				"chaincode", hdrExt.ChaincodeId.Name + ":" + hdrExt.ChaincodeId.Version,
				"success", strconv.FormatBool(success),
			}
			e.Metrics.ProposalDuration.With(meterLabels...).Observe(time.Since(startTime).Seconds())
		}

		endorserLogger.Debug("Exit: request from", addr)
	}()

//0—检查和验证
	vr, err := e.preProcess(signedProp)
	if err != nil {
		resp := vr.resp
		return resp, err
	}

	prop, hdrExt, chainID, txid := vr.prop, vr.hdrExt, vr.chainID, vr.txid

//为本提案获得一次Tx模拟器。这是零
//对于无链提案
//还可以获取历史查询的历史查询执行器，因为TX模拟器不包括历史记录
	var txsim ledger.TxSimulator
	var historyQueryExecutor ledger.HistoryQueryExecutor
	if acquireTxSimulator(chainID, vr.hdrExt.ChaincodeId) {
		if txsim, err = e.s.GetTxSimulator(chainID, txid); err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, nil
		}

//txsim在statedb上获取共享锁。因为这将影响块提交（即提交
//对于statedb的有效写入集，我们必须尽早释放锁。
//因此，只要对tx进行模拟，这个txsim对象就会在simulateProposal（）中关闭，并且
//如果私密数据需要，在传播流言之前收集RWSET。为了安全，我们
//添加以下defer语句，在出现错误时非常有用。注意呼叫
//txsim.done（）多次不会导致任何问题。如果TXSIM已经存在
//释放后，下面的txsim.done（）只返回。
		defer txsim.Done()

		if historyQueryExecutor, err = e.s.GetHistoryQueryExecutor(chainID); err != nil {
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, nil
		}
	}

	txParams := &ccprovider.TransactionParams{
		ChannelID:            chainID,
		TxID:                 txid,
		SignedProp:           signedProp,
		Proposal:             prop,
		TXSimulator:          txsim,
		HistoryQueryExecutor: historyQueryExecutor,
	}
//这可能是对无链系统CC的请求

//TODO:如果提案具有扩展名，则它将是chaincodeaction类型；
//如果存在，则意味着不执行模拟，因为
//我们正在尝试模拟一个提交的对等体。另一方面，我们需要
//在签署前验证所提供的操作

//1——模拟
	cd, res, simulationResult, ccevent, err := e.SimulateProposal(txParams, hdrExt.ChaincodeId)
	if err != nil {
		return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, nil
	}
	if res != nil {
		if res.Status >= shim.ERROR {
			endorserLogger.Errorf("[%s][%s] simulateProposal() resulted in chaincode %s response status %d for txid: %s", chainID, shorttxid(txid), hdrExt.ChaincodeId, res.Status, txid)
			var cceventBytes []byte
			if ccevent != nil {
				cceventBytes, err = putils.GetBytesChaincodeEvent(ccevent)
				if err != nil {
					return nil, errors.Wrap(err, "failed to marshal event bytes")
				}
			}
			pResp, err := putils.CreateProposalResponseFailure(prop.Header, prop.Payload, res, simulationResult, cceventBytes, hdrExt.ChaincodeId, hdrExt.PayloadVisibility)
			if err != nil {
				return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, nil
			}

			return pResp, nil
		}
	}

//2——签署并获得一个整理好的提议响应消息
	var pResp *pb.ProposalResponse

//直到我们实现了系统链码的全局ESCC、CSCC
//无链提案（如CSCC）无需背书。
	if chainID == "" {
		pResp = &pb.ProposalResponse{Response: res}
	} else {
//注意：为了支持oposal（），我们将传递已发布的txsim。因此，如果我们尝试使用此txsim，就会发生错误。
		pResp, err = e.endorseProposal(ctx, chainID, txid, signedProp, prop, res, simulationResult, ccevent, hdrExt.PayloadVisibility, hdrExt.ChaincodeId, txsim, cd)

//如果错误，捕获认可失败度量
		meterLabels := []string{
			"channel", chainID,
			"chaincode", hdrExt.ChaincodeId.Name + ":" + hdrExt.ChaincodeId.Version,
		}

		if err != nil {
			meterLabels = append(meterLabels, "chaincodeerror", strconv.FormatBool(false))
			e.Metrics.EndorsementsFailed.With(meterLabels...).Add(1)
			return &pb.ProposalResponse{Response: &pb.Response{Status: 500, Message: err.Error()}}, nil
		}
		if pResp.Response.Status >= shim.ERRORTHRESHOLD {
//默认的ESCC将有关阈值的所有状态代码视为错误，但未通过认可。
//有助于将其作为单独的度量进行跟踪
			meterLabels = append(meterLabels, "chaincodeerror", strconv.FormatBool(true))
			e.Metrics.EndorsementsFailed.With(meterLabels...).Add(1)
			endorserLogger.Debugf("[%s][%s] endorseProposal() resulted in chaincode %s error for txid: %s", chainID, shorttxid(txid), hdrExt.ChaincodeId, txid)
			return pResp, nil
		}
	}

//设置提案响应负载-it
//包含来自的“返回值”
//链码调用
	pResp.Response = res

//失败的提案总数=提案收到成功的提案
	e.Metrics.SuccessfulProposals.Add(1)
	success = true

	return pResp, nil
}

//确定事务模拟器是否应该
//为提案而获得。
func acquireTxSimulator(chainID string, ccid *pb.ChaincodeID) bool {
	if chainID == "" {
		return false
	}

//“锁定”。
//不要为查询和配置系统链代码获取模拟器。
//这些不需要模拟器，它的读锁会导致死锁。
	switch ccid.Name {
	case "qscc", "cscc":
		return false
	default:
		return true
	}
}

//shorttxid复制chaincode包函数以缩短txid。
//~~要在包之间使用一个通用的shortxid实用程序。~~
//TODO使用事务ID的形式类型并使其成为字符串
func shorttxid(txid string) string {
	if len(txid) < 8 {
		return txid
	}
	return txid[0:8]
}
