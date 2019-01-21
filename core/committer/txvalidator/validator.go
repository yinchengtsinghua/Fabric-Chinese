
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


package txvalidator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/configtx"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/common/validation"
	"github.com/hyperledger/fabric/core/ledger"
	ledgerUtil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//支持提供评估VSCC所需的所有信息
type Support interface {
//Acquire实现类似于Acquire语义的信号量
	Acquire(ctx context.Context, n int64) error

//release实现类似于信号量的释放语义
	Release(n int64)

//Ledger返回与此验证器关联的分类帐
	Ledger() ledger.PeerLedger

//msp manager返回此频道的msp管理器
	MSPManager() msp.MSPManager

//应用尝试将configtx应用到新配置
	Apply(configtx *common.ConfigEnvelope) error

//getmspids返回应用程序msps的ID
//在通道中定义的
	GetMSPIDs(cid string) []string

//功能定义此通道的应用程序部分的功能
	Capabilities() channelconfig.ApplicationCapabilities
}

//用于定义用于验证块事务的API的验证程序接口
//并返回位数组掩码，指示无效事务，其中
//未通过验证。
type Validator interface {
	Validate(block *common.Block) error
}

//用于分离Tx验证器的专用接口
//和VSCC执行，以增加
//txvalidator的可测试性
type vsccValidator interface {
	VSCCValidateTx(seq int, payload *common.Payload, envBytes []byte, block *common.Block) (error, peer.TxValidationCode)
}

//验证接口的实现，保持
//参考分类帐以启用Tx模拟
//以及VSCC的执行
type TxValidator struct {
	ChainID string
	Support Support
	Vscc    vsccValidator
}

var logger = flogging.MustGetLogger("committer.txvalidator")

type blockValidationRequest struct {
	block *common.Block
	d     []byte
	tIdx  int
}

type blockValidationResult struct {
	tIdx                 int
	validationCode       peer.TxValidationCode
	txsChaincodeName     *sysccprovider.ChaincodeInstance
	txsUpgradedChaincode *sysccprovider.ChaincodeInstance
	err                  error
	txid                 string
}

//newtxvalidator创建新的事务验证程序
func NewTxValidator(chainID string, support Support, sccp sysccprovider.SystemChaincodeProvider, pm PluginMapper) *TxValidator {
//封装接口实现
	pluginValidator := NewPluginValidator(pm, support.Ledger(), &dynamicDeserializer{support: support}, &dynamicCapabilities{support: support})
	return &TxValidator{
		ChainID: chainID,
		Support: support,
		Vscc:    newVSCCValidator(chainID, support, sccp, pluginValidator)}
}

func (v *TxValidator) chainExists(chain string) bool {
//TODO:实现此功能！
	return true
}

//validate执行块的验证。验证
//块中的每个事务都是并行执行的。
//方法如下：提交线程启动
//Goroutine中的Tx验证函数（使用信号量
//同时验证goroutine的数目）。委托人
//然后线程读取验证结果（按完成顺序）
//从结果频道。虎尾鹦鹉
//执行块中TXS的验证并将
//结果通道中的验证结果。一些值得注意的事实：
//1）为了保持方法简单，提交线程排队
//块中的所有事务，然后继续读取
//结果。
//2）对于并行验证工作，重要的是
//验证功能不会更改系统的状态。
//否则，验证的执行顺序很重要。
//我们必须求助于顺序验证（或某种锁定）。
//这目前是正确的，因为唯一影响
//状态是接收配置事务的时间，但它们是
//保证在街区里独处。如果/当这个假设
//违反了，必须更改此代码。
func (v *TxValidator) Validate(block *common.Block) error {
	var err error
	var errPos int

startValidation := time.Now() //记录验证块持续时间的计时器
	logger.Debugf("[%s] START Block Validation for block [%d]", v.ChainID, block.Header.Number)

//在这里初始化trans为有效，然后在下面的invalidation时设置invalidation reason code
	txsfltr := ledgerUtil.NewTxValidationFlags(len(block.Data.Data))
//txschaincodenames将tx调用的所有链码记录在一个块中。
	txsChaincodeNames := make(map[int]*sysccprovider.ChaincodeInstance)
//upgraded chaincodes记录块中升级的所有链码
	txsUpgradedChaincodes := make(map[int]*sysccprovider.ChaincodeInstance)
//TXID数组
	txidArray := make([]string, len(block.Data.Data))

	results := make(chan *blockValidationResult)
	go func() {
		for tIdx, d := range block.Data.Data {
//确保我们没有太多的并发验证人员
			v.Support.Acquire(context.Background(), 1)

			go func(index int, data []byte) {
				defer v.Support.Release(1)

				v.validateTx(&blockValidationRequest{
					d:     data,
					block: block,
					tIdx:  index,
				}, results)
			}(tIdx, d)
		}
	}()

	logger.Debugf("expecting %d block validation responses", len(block.Data.Data))

//现在我们按照回复的顺序来阅读回复。
	for i := 0; i < len(block.Data.Data); i++ {
		res := <-results

		if res.err != nil {
//如果有错误，我们缓冲它的值，等待
//所有工人完成验证，然后返回
//此块中第一个Tx返回错误的错误
			logger.Debugf("got terminal error %s for idx %d", res.err, res.tIdx)

			if err == nil || res.tIdx < errPos {
				err = res.err
				errPos = res.tIdx
			}
		} else {
//如果没有错误，我们设置txsfltr，然后设置
//txschaincodenames和txsupgradedchaincodes图
			logger.Debugf("got result for idx %d, code %d", res.tIdx, res.validationCode)

			txsfltr.SetFlag(res.tIdx, res.validationCode)

			if res.validationCode == peer.TxValidationCode_VALID {
				if res.txsChaincodeName != nil {
					txsChaincodeNames[res.tIdx] = res.txsChaincodeName
				}
				if res.txsUpgradedChaincode != nil {
					txsUpgradedChaincodes[res.tIdx] = res.txsUpgradedChaincode
				}
				txidArray[res.tIdx] = res.txid
			}
		}
	}

//如果我们在这里，所有的工人都已经完成了验证。
//如果有错误，我们从第一个返回错误
//此块中返回错误的Tx
	if err != nil {
		return err
	}

//如果我们使用此功能进行操作，则会将具有txid的任何事务标记为无效
//与此块中以前的Tx相同
	if v.Support.Capabilities().ForbidDuplicateTXIdInBlock() {
		markTXIdDuplicates(txidArray, txsfltr)
	}

//如果我们在这里，所有的工人都已经完成了验证和
//未报告错误；我们设置了Tx过滤器并返回
//成功
	v.invalidTXsForUpgradeCC(txsChaincodeNames, txsUpgradedChaincodes, txsfltr)

//确保没有跳过验证的事务
	err = v.allValidated(txsfltr, block)
	if err != nil {
		return err
	}

//初始化元数据结构
	utils.InitBlockMetadata(block)

	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsfltr

elapsedValidation := time.Since(startValidation) / time.Millisecond //MS持续时间
	logger.Infof("[%s] Validated block [%d] in %dms", v.ChainID, block.Header.Number, elapsedValidation)

	return nil
}

//如果某些验证标志尚未设置，则allvalidated返回错误。
//验证期间
func (v *TxValidator) allValidated(txsfltr ledgerUtil.TxValidationFlags, block *common.Block) error {
	for id, f := range txsfltr {
		if peer.TxValidationCode(f) == peer.TxValidationCode_NOT_VALIDATED {
			return errors.Errorf("transaction %d in block %d has skipped validation", id, block.Header.Number)
		}
	}

	return nil
}

func markTXIdDuplicates(txids []string, txsfltr ledgerUtil.TxValidationFlags) {
	txidMap := make(map[string]struct{})

	for id, txid := range txids {
		if txid == "" {
			continue
		}

		_, in := txidMap[txid]
		if in {
			logger.Error("Duplicate txid", txid, "found, skipping")
			txsfltr.SetFlag(id, peer.TxValidationCode_DUPLICATE_TXID)
		} else {
			txidMap[txid] = struct{}{}
		}
	}
}

func (v *TxValidator) validateTx(req *blockValidationRequest, results chan<- *blockValidationResult) {
	block := req.block
	d := req.d
	tIdx := req.tIdx
	txID := ""

	if d == nil {
		results <- &blockValidationResult{
			tIdx: tIdx,
		}
		return
	}

	if env, err := utils.GetEnvelopeFromBlock(d); err != nil {
		logger.Warningf("Error getting tx from block: %+v", err)
		results <- &blockValidationResult{
			tIdx:           tIdx,
			validationCode: peer.TxValidationCode_INVALID_OTHER_REASON,
		}
		return
	} else if env != nil {
//验证事务：这里我们检查事务
//格式正确，签名正确，并且
//对Tx持有的背书的连锁约束性建议。我们做
//不过，不要检查背书的有效性。那是一个
//下面的VSCC作业
		logger.Debugf("[%s] validateTx starts for block %p env %p txn %d", v.ChainID, block, env, tIdx)
		defer logger.Debugf("[%s] validateTx completes for block %p env %p txn %d", v.ChainID, block, env, tIdx)
		var payload *common.Payload
		var err error
		var txResult peer.TxValidationCode
		var txsChaincodeName *sysccprovider.ChaincodeInstance
		var txsUpgradedChaincode *sysccprovider.ChaincodeInstance

		if payload, txResult = validation.ValidateTransaction(env, v.Support.Capabilities()); txResult != peer.TxValidationCode_VALID {
			logger.Errorf("Invalid transaction with index %d", tIdx)
			results <- &blockValidationResult{
				tIdx:           tIdx,
				validationCode: txResult,
			}
			return
		}

		chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			logger.Warningf("Could not unmarshal channel header, err %s, skipping", err)
			results <- &blockValidationResult{
				tIdx:           tIdx,
				validationCode: peer.TxValidationCode_INVALID_OTHER_REASON,
			}
			return
		}

		channel := chdr.ChannelId
		logger.Debugf("Transaction is for channel %s", channel)

		if !v.chainExists(channel) {
			logger.Errorf("Dropping transaction for non-existent channel %s", channel)
			results <- &blockValidationResult{
				tIdx:           tIdx,
				validationCode: peer.TxValidationCode_TARGET_CHAIN_NOT_FOUND,
			}
			return
		}

		if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {

			txID = chdr.TxId

//检查重复的交易记录
			erroneousResultEntry := v.checkTxIdDupsLedger(tIdx, chdr, v.Support.Ledger())
			if erroneousResultEntry != nil {
				results <- erroneousResultEntry
				return
			}

//使用vscc和策略验证tx
			logger.Debug("Validating transaction vscc tx validate")
			err, cde := v.Vscc.VSCCValidateTx(tIdx, payload, d, block)
			if err != nil {
				logger.Errorf("VSCCValidateTx for transaction txId = %s returned error: %s", txID, err)
				switch err.(type) {
				case *commonerrors.VSCCExecutionFailureError:
					results <- &blockValidationResult{
						tIdx: tIdx,
						err:  err,
					}
					return
				case *commonerrors.VSCCInfoLookupFailureError:
					results <- &blockValidationResult{
						tIdx: tIdx,
						err:  err,
					}
					return
				default:
					results <- &blockValidationResult{
						tIdx:           tIdx,
						validationCode: cde,
					}
					return
				}
			}

			invokeCC, upgradeCC, err := v.getTxCCInstance(payload)
			if err != nil {
				logger.Errorf("Get chaincode instance from transaction txId = %s returned error: %+v", txID, err)
				results <- &blockValidationResult{
					tIdx:           tIdx,
					validationCode: peer.TxValidationCode_INVALID_OTHER_REASON,
				}
				return
			}
			txsChaincodeName = invokeCC
			if upgradeCC != nil {
				logger.Infof("Find chaincode upgrade transaction for chaincode %s on channel %s with new version %s", upgradeCC.ChaincodeName, upgradeCC.ChainID, upgradeCC.ChaincodeVersion)
				txsUpgradedChaincode = upgradeCC
			}
//FAB-12971在v1.4切割前在块下方注释。将在v1.4之后取消注释。
   /*
    else if common.headertype（chdr.type）==common.headertype_token_transaction_

     txid=chdr.txid
     如果！v.support.capabilities（）.fabtoken（）
      logger.errorf（“fabtoken功能未启用。块〔%d〕中不支持的事务类型〔%s〕，事务〔%d〕“，
       common.header type（chdr.type），block.header.number，tidx）
      结果<-&blockvalidationresult_
       tidx：tidx，
       验证码：peer.tx validationcode_unsupported_tx_payload，
      }
      返回
     }

     //检查分类帐中是否有此类交易的副本，以及
     //获取确认错误类型的相应结果
     错误结果条目：=v.checktxiddupsledger（tidx，chdr，v.support.ledger（））
     如果结果输入错误！= nIL{
      结果<-错误结果条目
      返回
     }

     //设置调用字段的命名空间
     txschaincodename=&sysccprovider.chaincodeinstance_
      链ID:通道，
      chaincodename:“令牌”，
      链码版本：“”
   **/

		} else if common.HeaderType(chdr.Type) == common.HeaderType_CONFIG {
			configEnvelope, err := configtx.UnmarshalConfigEnvelope(payload.Data)
			if err != nil {
				err = errors.WithMessage(err, "error unmarshalling config which passed initial validity checks")
				logger.Criticalf("%+v", err)
				results <- &blockValidationResult{
					tIdx: tIdx,
					err:  err,
				}
				return
			}

			if err := v.Support.Apply(configEnvelope); err != nil {
				err = errors.WithMessage(err, "error validating config which passed initial validity checks")
				logger.Criticalf("%+v", err)
				results <- &blockValidationResult{
					tIdx: tIdx,
					err:  err,
				}
				return
			}
			logger.Debugf("config transaction received for chain %s", channel)
		} else {
			logger.Warningf("Unknown transaction type [%s] in block number [%d] transaction index [%d]",
				common.HeaderType(chdr.Type), block.Header.Number, tIdx)
			results <- &blockValidationResult{
				tIdx:           tIdx,
				validationCode: peer.TxValidationCode_UNKNOWN_TX_TYPE,
			}
			return
		}

		if _, err := proto.Marshal(env); err != nil {
			logger.Warningf("Cannot marshal transaction: %s", err)
			results <- &blockValidationResult{
				tIdx:           tIdx,
				validationCode: peer.TxValidationCode_MARSHAL_TX_ERROR,
			}
			return
		}
//在这里传递成功，事务有效
		results <- &blockValidationResult{
			tIdx:                 tIdx,
			txsChaincodeName:     txsChaincodeName,
			txsUpgradedChaincode: txsUpgradedChaincode,
			validationCode:       peer.TxValidationCode_VALID,
			txid:                 txID,
		}
		return
	} else {
		logger.Warning("Nil tx from block")
		results <- &blockValidationResult{
			tIdx:           tIdx,
			validationCode: peer.TxValidationCode_NIL_ENVELOPE,
		}
		return
	}
}

//checktxiddupsledger返回一个vlockvalidationresult，并分别使用
//仅当存在具有相同事务标识符的事务时才出现错误代码
//在分类帐中或不能决定是否存在此类交易；
//如果函数已确保没有此类重复，则返回nil，例如
//它的使用者可以继续进行事务处理
func (v *TxValidator) checkTxIdDupsLedger(tIdx int, chdr *common.ChannelHeader, ldgr ledger.PeerLedger) (errorTuple *blockValidationResult) {

//检索输入头的事务标识符
	txID := chdr.TxId

//在分类帐中查找具有相同标识符的交易记录
	_, err := ldgr.GetTransactionByID(txID)

//如果返回的错误为零，则表示已经存在一个Tx-In
//具有提供的ID的分类帐
	if err == nil {
		logger.Error("Duplicate transaction found, ", txID, ", skipping")
		return &blockValidationResult{
			tIdx:           tIdx,
			validationCode: peer.TxValidationCode_DUPLICATE_TXID,
		}
	}

//如果返回的错误不是blkstorage.notfoundinindexerr类型，则表示
//我们无法验证具有所提供ID的Tx是否在分类帐中
	if _, isNotFoundInIndexErrType := err.(ledger.NotFoundInIndexErr); !isNotFoundInIndexErrType {
		logger.Errorf("Ledger failure while attempting to detect duplicate status for "+
			"txid %s, err '%s'. Aborting", txID, err)
		return &blockValidationResult{
			tIdx: tIdx,
			err:  err,
		}
	}

//否则就意味着没有具有相同标识符的事务
//居住在分类帐中
	return nil
}

//GenerateCckey为特定通道中的链码生成唯一标识符
func (v *TxValidator) generateCCKey(ccName, chainID string) string {
	return fmt.Sprintf("%s/%s", ccName, chainID)
}

//invalidtxsforpupgradecc无效由于chaincode升级txs而应无效的所有txs
func (v *TxValidator) invalidTXsForUpgradeCC(txsChaincodeNames map[int]*sysccprovider.ChaincodeInstance, txsUpgradedChaincodes map[int]*sysccprovider.ChaincodeInstance, txsfltr ledgerUtil.TxValidationFlags) {
	if len(txsUpgradedChaincodes) == 0 {
		return
	}

//如果有两个或多个TXS升级同一个CC，则以前的CC升级TXS无效
	finalValidUpgradeTXs := make(map[string]int)
	upgradedChaincodes := make(map[string]*sysccprovider.ChaincodeInstance)
	for tIdx, cc := range txsUpgradedChaincodes {
		if cc == nil {
			continue
		}
		upgradedCCKey := v.generateCCKey(cc.ChaincodeName, cc.ChainID)

		if finalIdx, exist := finalValidUpgradeTXs[upgradedCCKey]; !exist {
			finalValidUpgradeTXs[upgradedCCKey] = tIdx
			upgradedChaincodes[upgradedCCKey] = cc
		} else if finalIdx < tIdx {
			logger.Infof("Invalid transaction with index %d: chaincode was upgraded by latter tx", finalIdx)
			txsfltr.SetFlag(finalIdx, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)

//记录后一个CC升级Tx信息
			finalValidUpgradeTXs[upgradedCCKey] = tIdx
			upgradedChaincodes[upgradedCCKey] = cc
		} else {
			logger.Infof("Invalid transaction with index %d: chaincode was upgraded by latter tx", tIdx)
			txsfltr.SetFlag(tIdx, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
		}
	}

//调用升级后的链码的TXS无效
	for tIdx, cc := range txsChaincodeNames {
		if cc == nil {
			continue
		}
		ccKey := v.generateCCKey(cc.ChaincodeName, cc.ChainID)
		if _, exist := upgradedChaincodes[ccKey]; exist {
			if txsfltr.IsValid(tIdx) {
				logger.Infof("Invalid transaction with index %d: chaincode was upgraded in the same block", tIdx)
				txsfltr.SetFlag(tIdx, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
			}
		}
	}
}

func (v *TxValidator) getTxCCInstance(payload *common.Payload) (invokeCCIns, upgradeCCIns *sysccprovider.ChaincodeInstance, err error) {
//这是重复的解包工作，但使测试更容易。
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, nil, err
	}

//链ID
chainID := chdr.ChannelId //到目前为止，它保证是一个现有的渠道

//链状的
	hdrExt, err := utils.GetChaincodeHeaderExtension(payload.Header)
	if err != nil {
		return nil, nil, err
	}
	invokeCC := hdrExt.ChaincodeId
	invokeIns := &sysccprovider.ChaincodeInstance{ChainID: chainID, ChaincodeName: invokeCC.Name, ChaincodeVersion: invokeCC.Version}

//交易
	tx, err := utils.GetTransaction(payload.Data)
	if err != nil {
		logger.Errorf("GetTransaction failed: %+v", err)
		return invokeIns, nil, nil
	}

//链码操作负载
	cap, err := utils.GetChaincodeActionPayload(tx.Actions[0].Payload)
	if err != nil {
		logger.Errorf("GetChaincodeActionPayload failed: %+v", err)
		return invokeIns, nil, nil
	}

//链码建议有效载荷
	cpp, err := utils.GetChaincodeProposalPayload(cap.ChaincodeProposalPayload)
	if err != nil {
		logger.Errorf("GetChaincodeProposalPayload failed: %+v", err)
		return invokeIns, nil, nil
	}

//链码调用spec
	cis := &peer.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(cpp.Input, cis)
	if err != nil {
		logger.Errorf("GetChaincodeInvokeSpec failed: %+v", err)
		return invokeIns, nil, nil
	}

	if invokeCC.Name == "lscc" {
		if string(cis.ChaincodeSpec.Input.Args[0]) == "upgrade" {
			upgradeIns, err := v.getUpgradeTxInstance(chainID, cis.ChaincodeSpec.Input.Args[2])
			if err != nil {
				return invokeIns, nil, nil
			}
			return invokeIns, upgradeIns, nil
		}
	}

	return invokeIns, nil, nil
}

func (v *TxValidator) getUpgradeTxInstance(chainID string, cdsBytes []byte) (*sysccprovider.ChaincodeInstance, error) {
	cds, err := utils.GetChaincodeDeploymentSpec(cdsBytes, platforms.NewRegistry(&golang.Platform{}))
	if err != nil {
		return nil, err
	}

	return &sysccprovider.ChaincodeInstance{
		ChainID:          chainID,
		ChaincodeName:    cds.ChaincodeSpec.ChaincodeId.Name,
		ChaincodeVersion: cds.ChaincodeSpec.ChaincodeId.Version,
	}, nil
}

type dynamicDeserializer struct {
	support Support
}

func (ds *dynamicDeserializer) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	return ds.support.MSPManager().DeserializeIdentity(serializedIdentity)
}

func (ds *dynamicDeserializer) IsWellFormed(identity *mspprotos.SerializedIdentity) error {
	return ds.support.MSPManager().IsWellFormed(identity)
}

type dynamicCapabilities struct {
	support Support
}

func (ds *dynamicCapabilities) ACLs() bool {
	return ds.support.Capabilities().ACLs()
}

func (ds *dynamicCapabilities) CollectionUpgrade() bool {
	return ds.support.Capabilities().CollectionUpgrade()
}

//如果支持fabric token函数，fabtoken返回true。
func (ds *dynamicCapabilities) FabToken() bool {
	return ds.support.Capabilities().FabToken()
}

func (ds *dynamicCapabilities) ForbidDuplicateTXIdInBlock() bool {
	return ds.support.Capabilities().ForbidDuplicateTXIdInBlock()
}

func (ds *dynamicCapabilities) KeyLevelEndorsement() bool {
	return ds.support.Capabilities().KeyLevelEndorsement()
}

func (ds *dynamicCapabilities) MetadataLifecycle() bool {
	return ds.support.Capabilities().MetadataLifecycle()
}

func (ds *dynamicCapabilities) PrivateChannelData() bool {
	return ds.support.Capabilities().PrivateChannelData()
}

func (ds *dynamicCapabilities) Supported() error {
	return ds.support.Capabilities().Supported()
}

func (ds *dynamicCapabilities) V1_1Validation() bool {
	return ds.support.Capabilities().V1_1Validation()
}

func (ds *dynamicCapabilities) V1_2Validation() bool {
	return ds.support.Capabilities().V1_2Validation()
}

func (ds *dynamicCapabilities) V1_3Validation() bool {
	return ds.support.Capabilities().V1_3Validation()
}
