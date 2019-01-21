
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


package statebased

import (
	"fmt"
	"sync"

	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/core/handlers/validation/api/policies"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

/*********************************************************************/
/*********************************************************************/

type policyChecker struct {
	someEPChecked bool
	ccEPChecked   bool
	vpmgr         KeyLevelValidationParameterManager
	policySupport validation.PolicyEvaluator
	ccEP          []byte
	signatureSet  []*common.SignedData
}

func (p *policyChecker) checkCCEPIfCondition(cc string, blockNum, txNum uint64, condition bool) commonerrors.TxValidationError {
	if condition {
		return nil
	}

//根据CC EP验证
	err := p.policySupport.Evaluate(p.ccEP, p.signatureSet)
	if err != nil {
		return policyErr(errors.Wrapf(err, "validation of endorsement policy for chaincode %s in tx %d:%d failed", cc, blockNum, txNum))
	}

	p.ccEPChecked = true
	p.someEPChecked = true
	return nil
}

func (p *policyChecker) checkCCEPIfNotChecked(cc string, blockNum, txNum uint64) commonerrors.TxValidationError {
	return p.checkCCEPIfCondition(cc, blockNum, txNum, p.ccEPChecked)
}

func (p *policyChecker) checkCCEPIfNoEPChecked(cc string, blockNum, txNum uint64) commonerrors.TxValidationError {
	return p.checkCCEPIfCondition(cc, blockNum, txNum, p.someEPChecked)
}

func (p *policyChecker) checkSBAndCCEP(cc, coll, key string, blockNum, txNum uint64) commonerrors.TxValidationError {
//查看是否有此密钥的密钥级别验证参数
	vp, err := p.vpmgr.GetValidationParameterForKey(cc, coll, key, blockNum, txNum)
	if err != nil {
//GetValidationParameterWorkey的错误处理遵循以下基本原理：
		switch err := errors.Cause(err).(type) {
//1）如果由于验证参数已更新而发生冲突
//通过此块中的另一个事务，我们将得到validationParameterUpdatedError。
//这将导致通过调用policyErr使事务无效
		case *ValidationParameterUpdatedError:
			return policyErr(err)
//2）如果分类账返回“确定”错误，即
//通道中的每个对等点也将返回（例如链接到
//尝试从未定义的集合中检索元数据）应为
//已记录并忽略。分类帐将采取最适当的行动
//when performing its side of the validation.
		case *ledger.CollConfigNotDefinedError, *ledger.InvalidCollNameError:
			logger.Warningf(errors.WithMessage(err, "skipping key-level validation").Error())
			err = nil
//3）任何其他类型的错误都应返回执行失败，这将
//lead to halting the processing on this channel. Note that any non-categorized
//确定性错误会被默认捕获并导致
//处理停止。这肯定是一个错误，但是-如果没有
//分类帐返回的单一、定义明确的确定性错误，它是
//最好是谨慎行事，而不是停止处理（因为
//deterministic error is treated like an I/O one) rather than risking a fork
//(in case an I/O error is treated as a deterministic one).
		default:
			return &commonerrors.VSCCExecutionFailureError{
				Err: err,
			}
		}
	}

//如果没有指定密钥级验证参数，则需要保留常规的CC认可策略
	if len(vp) == 0 {
		return p.checkCCEPIfNotChecked(cc, blockNum, txNum)
	}

//验证关键级VP
	err = p.policySupport.Evaluate(vp, p.signatureSet)
	if err != nil {
		return policyErr(errors.Wrapf(err, "validation of key %s (coll'%s':ns'%s') in tx %d:%d failed", key, coll, cc, blockNum, txNum))
	}

	p.someEPChecked = true

	return nil
}

/*********************************************************************/
/*********************************************************************/

type blockDependency struct {
	mutex     sync.Mutex
	blockNum  uint64
	txDepOnce []sync.Once
}

//keylevalidator根据密钥级ep验证实现
type KeyLevelValidator struct {
	vpmgr         KeyLevelValidationParameterManager
	policySupport validation.PolicyEvaluator
	blockDep      blockDependency
}

func NewKeyLevelValidator(policySupport validation.PolicyEvaluator, vpmgr KeyLevelValidationParameterManager) *KeyLevelValidator {
	return &KeyLevelValidator{
		vpmgr:         vpmgr,
		policySupport: policySupport,
		blockDep:      blockDependency{},
	}
}

func (klv *KeyLevelValidator) invokeOnce(block *common.Block, txnum uint64) *sync.Once {
	klv.blockDep.mutex.Lock()
	defer klv.blockDep.mutex.Unlock()

	if klv.blockDep.blockNum != block.Header.Number {
		klv.blockDep.blockNum = block.Header.Number
		klv.blockDep.txDepOnce = make([]sync.Once, len(block.Data.Data))
	}

	return &klv.blockDep.txDepOnce[txnum]
}

func (klv *KeyLevelValidator) extractDependenciesForTx(blockNum, txNum uint64, envelopeBytes []byte) {
	env, err := utils.GetEnvelopeFromBlock(envelopeBytes)
	if err != nil {
		logger.Warningf("while executing GetEnvelopeFromBlock got error '%s', skipping tx at height (%d,%d)", err, blockNum, txNum)
		return
	}

	payl, err := utils.GetPayload(env)
	if err != nil {
		logger.Warningf("while executing GetPayload got error '%s', skipping tx at height (%d,%d)", err, blockNum, txNum)
		return
	}

	tx, err := utils.GetTransaction(payl.Data)
	if err != nil {
		logger.Warningf("while executing GetTransaction got error '%s', skipping tx at height (%d,%d)", err, blockNum, txNum)
		return
	}

	cap, err := utils.GetChaincodeActionPayload(tx.Actions[0].Payload)
	if err != nil {
		logger.Warningf("while executing GetChaincodeActionPayload got error '%s', skipping tx at height (%d,%d)", err, blockNum, txNum)
		return
	}

	pRespPayload, err := utils.GetProposalResponsePayload(cap.Action.ProposalResponsePayload)
	if err != nil {
		logger.Warningf("while executing GetProposalResponsePayload got error '%s', skipping tx at height (%d,%d)", err, blockNum, txNum)
		return
	}

	respPayload, err := utils.GetChaincodeAction(pRespPayload.Extension)
	if err != nil {
		logger.Warningf("while executing GetChaincodeAction got error '%s', skipping tx at height (%d,%d)", err, blockNum, txNum)
		return
	}

	klv.vpmgr.ExtractValidationParameterDependency(blockNum, txNum, respPayload.Results)
}

//prevalidate实现statebasedvalidator接口的功能
func (klv *KeyLevelValidator) PreValidate(txNum uint64, block *common.Block) {
	for i := int64(txNum); i >= 0; i-- {
		txPosition := uint64(i)

		klv.invokeOnce(block, txPosition).Do(
			func() {
				klv.extractDependenciesForTx(block.Header.Number, txPosition, block.Data.Data[txPosition])
			})
	}
}

//validate实现statebasedvalidator接口的功能
func (klv *KeyLevelValidator) Validate(cc string, blockNum, txNum uint64, rwsetBytes, prp, ccEP []byte, endorsements []*peer.Endorsement) commonerrors.TxValidationError {
//构造签名集
	signatureSet := []*common.SignedData{}
	for _, endorsement := range endorsements {
		data := make([]byte, len(prp)+len(endorsement.Endorser))
		copy(data, prp)
		copy(data[len(prp):], endorsement.Endorser)

		signatureSet = append(signatureSet, &common.SignedData{
//设置已签名的数据；建议响应字节和背书人ID的串联
			Data: data,
//设置在消息上签名的标识：它是背书人
			Identity: endorsement.Endorser,
//设置签名
			Signature: endorsement.Signature})
	}

//构造策略检查器对象
	policyChecker := policyChecker{
		ccEP:          ccEP,
		policySupport: klv.policySupport,
		signatureSet:  signatureSet,
		vpmgr:         klv.vpmgr,
	}

//打开RWSET
	rwset := &rwsetutil.TxRwSet{}
	if err := rwset.FromProtoBytes(rwsetBytes); err != nil {
		return policyErr(errors.WithMessage(err, fmt.Sprintf("txRWSet.FromProtoBytes failed on tx (%d,%d)", blockNum, txNum)))
	}

//迭代rwset中的所有写入操作
	for _, nsRWSet := range rwset.NsRwSets {
//跳过其他命名空间
		if nsRWSet.NameSpace != cc {
			continue
		}

//公共写作
//我们根据密钥级验证参数验证写入
//如果有或整个连锁码的背书政策
		for _, pubWrite := range nsRWSet.KvRwSet.Writes {
			err := policyChecker.checkSBAndCCEP(cc, "", pubWrite.Key, blockNum, txNum)
			if err != nil {
				return err
			}
		}
//public metadata writes
//我们根据密钥级验证参数验证写入
//如果有或整个连锁码的背书政策
		for _, pubMdWrite := range nsRWSet.KvRwSet.MetadataWrites {
			err := policyChecker.checkSBAndCCEP(cc, "", pubMdWrite.Key, blockNum, txNum)
			if err != nil {
				return err
			}
		}
//在集合中写入
//我们根据密钥级验证参数验证写入
//如果有或整个连锁码的背书政策
		for _, collRWSet := range nsRWSet.CollHashedRwSets {
			coll := collRWSet.CollectionName
			for _, hashedWrite := range collRWSet.HashedRwSet.HashedWrites {
				key := string(hashedWrite.KeyHash)
				err := policyChecker.checkSBAndCCEP(cc, coll, key, blockNum, txNum)
				if err != nil {
					return err
				}
			}
		}
//集合中的元数据写入
//我们根据密钥级验证参数验证写入
//如果有或整个连锁码的背书政策
		for _, collRWSet := range nsRWSet.CollHashedRwSets {
			coll := collRWSet.CollectionName
			for _, hashedMdWrite := range collRWSet.HashedRwSet.MetadataWrites {
				key := string(hashedMdWrite.KeyHash)
				err := policyChecker.checkSBAndCCEP(cc, coll, key, blockNum, txNum)
				if err != nil {
					return err
				}
			}
		}
	}

//我们确保我们至少检查CIPP以兑现FAB-943。
	return policyChecker.checkCCEPIfNoEPChecked(cc, blockNum, txNum)
}

//postvalidate实现statebasedvalidator接口的功能
func (klv *KeyLevelValidator) PostValidate(cc string, blockNum, txNum uint64, err error) {
	klv.vpmgr.SetTxValidationResult(cc, blockNum, txNum, err)
}

func policyErr(err error) *commonerrors.VSCCEndorsementPolicyError {
	return &commonerrors.VSCCEndorsementPolicyError{
		Err: err,
	}
}
