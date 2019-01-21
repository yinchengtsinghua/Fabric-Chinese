
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *
 *版权所有IBM公司。保留所有权利。
 *
 *SPDX许可证标识符：Apache-2.0
 */
 *
 **/


package txvalidator

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	coreUtil "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/handlers/validation/api"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//vsccvalidatorimpl是用于调用
//VSCC链码和验证块事务
type VsccValidatorImpl struct {
	chainID         string
	support         Support
	sccprovider     sysccprovider.SystemChaincodeProvider
	pluginValidator *PluginValidator
}

//new vscc validator创建新的vscc validator
func newVSCCValidator(chainID string, support Support, sccp sysccprovider.SystemChaincodeProvider, pluginValidator *PluginValidator) *VsccValidatorImpl {
	return &VsccValidatorImpl{
		chainID:         chainID,
		support:         support,
		sccprovider:     sccp,
		pluginValidator: pluginValidator,
	}
}

//vsccvalidatetx对事务执行vscc验证
func (v *VsccValidatorImpl) VSCCValidateTx(seq int, payload *common.Payload, envBytes []byte, block *common.Block) (error, peer.TxValidationCode) {
	chainID := v.chainID
	logger.Debugf("[%s] VSCCValidateTx starts for bytes %p", chainID, envBytes)

//获取头扩展名，这样我们就有了chaincode id
	hdrExt, err := utils.GetChaincodeHeaderExtension(payload.Header)
	if err != nil {
		return err, peer.TxValidationCode_BAD_HEADER_EXTENSION
	}

//获取频道标题
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return err, peer.TxValidationCode_BAD_CHANNEL_HEADER
	}

 /*获取我们要编写东西的名称空间列表；
    首先，我们建立了一些关于这个调用的事实：
    1）它写入哪些名称空间？
    2）它是否写入LSCC的命名空间？
    3）它是否写入任何无法调用的CC？*/

	writesToLSCC := false
	writesToNonInvokableSCC := false
	respPayload, err := utils.GetActionFromEnvelope(envBytes)
	if err != nil {
		return errors.WithMessage(err, "GetActionFromEnvelope failed"), peer.TxValidationCode_BAD_RESPONSE_PAYLOAD
	}
	txRWSet := &rwsetutil.TxRwSet{}
	if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
		return errors.WithMessage(err, "txRWSet.FromProtoBytes failed"), peer.TxValidationCode_BAD_RWSET
	}

//验证头扩展和响应负载是否包含chaincodeid
	if hdrExt.ChaincodeId == nil {
		return errors.New("nil ChaincodeId in header extension"), peer.TxValidationCode_INVALID_OTHER_REASON
	}

	if respPayload.ChaincodeId == nil {
		return errors.New("nil ChaincodeId in ChaincodeAction"), peer.TxValidationCode_INVALID_OTHER_REASON
	}

//获取我们调用的CC的名称和版本
	ccID := hdrExt.ChaincodeId.Name
	ccVer := respPayload.ChaincodeId.Version

//CCID的健全性检查
	if ccID == "" {
		err = errors.New("invalid chaincode ID")
		logger.Errorf("%+v", err)
		return err, peer.TxValidationCode_INVALID_OTHER_REASON
	}
	if ccID != respPayload.ChaincodeId.Name {
		err = errors.Errorf("inconsistent ccid info (%s/%s)", ccID, respPayload.ChaincodeId.Name)
		logger.Errorf("%+v", err)
		return err, peer.TxValidationCode_INVALID_OTHER_REASON
	}
//CCVER的健全性检查
	if ccVer == "" {
		err = errors.New("invalid chaincode version")
		logger.Errorf("%+v", err)
		return err, peer.TxValidationCode_INVALID_OTHER_REASON
	}

	var wrNamespace []string
	alwaysEnforceOriginalNamespace := v.support.Capabilities().V1_2Validation()
	if alwaysEnforceOriginalNamespace {
		wrNamespace = append(wrNamespace, ccID)
		if respPayload.Events != nil {
			ccEvent := &peer.ChaincodeEvent{}
			if err = proto.Unmarshal(respPayload.Events, ccEvent); err != nil {
				return errors.Wrapf(err, "invalid chaincode event"), peer.TxValidationCode_INVALID_OTHER_REASON
			}
			if ccEvent.ChaincodeId != ccID {
				return errors.Errorf("chaincode event chaincode id does not match chaincode action chaincode id"), peer.TxValidationCode_INVALID_OTHER_REASON
			}
		}
	}

	for _, ns := range txRWSet.NsRwSets {
		if !v.txWritesToNamespace(ns) {
			continue
		}

//检查以确保尚未填充此链码
//避免检查同一命名空间两次的名称
		if ns.NameSpace != ccID || !alwaysEnforceOriginalNamespace {
			wrNamespace = append(wrNamespace, ns.NameSpace)
		}

		if !writesToLSCC && ns.NameSpace == "lscc" {
			writesToLSCC = true
		}

		if !writesToNonInvokableSCC && v.sccprovider.IsSysCCAndNotInvokableCC2CC(ns.NameSpace) {
			writesToNonInvokableSCC = true
		}

		if !writesToNonInvokableSCC && v.sccprovider.IsSysCCAndNotInvokableExternal(ns.NameSpace) {
			writesToNonInvokableSCC = true
		}
	}

//我们已经收集了进行验证所需的所有信息；
//验证将根据
//链码（系统与应用）

	if !v.sccprovider.IsSysCC(ccID) {
//如果我们在这里，我们知道这是应用程序链代码的调用；
//首先，我们确保：
//1）我们不向lscc写入-应用程序链代码可以自由调用lscc
//例如，获取有关自身或其他链码的信息；但是
//这些合法调用只能从lscc的名称空间中进行；当前
//lscc只有两个函数写入其命名空间：deploy和upgrade以及
//应用程序链代码也不应使用
		if writesToLSCC {
			return errors.Errorf("chaincode %s attempted to write to the namespace of LSCC", ccID),
				peer.TxValidationCode_ILLEGAL_WRITESET
		}
//2）我们不会向无法调用的链代码的命名空间写入-if
//不能首先调用链码，没有合法的
//事务具有写入它的写入集的方式；另外
//我们没有任何手段来核实交易是否有权
//因为在v1中，系统链码没有
//任何要说的背书政策。所以如果不能调用chaincode
//无法通过调用应用程序链代码将其写入
		if writesToNonInvokableSCC {
			return errors.Errorf("chaincode %s attempted to write to the namespace of a system chaincode that cannot be invoked", ccID),
				peer.TxValidationCode_ILLEGAL_WRITESET
		}

//根据链码的认可策略验证*每个*读写集
		for _, ns := range wrNamespace {
//获取最新的链码版本、VSCC和验证策略
			txcc, vscc, policy, err := v.GetInfoForValidate(chdr, ns)
			if err != nil {
				logger.Errorf("GetInfoForValidate for txId = %s returned error: %+v", chdr.TxId, err)
				return err, peer.TxValidationCode_INVALID_OTHER_REASON
			}

//如果命名空间与最初的CC相对应
//调用时，我们检查
//调用对应于lscc返回的版本
			if ns == ccID && txcc.ChaincodeVersion != ccVer {
				err = errors.Errorf("chaincode %s:%s/%s didn't match %s:%s/%s in lscc", ccID, ccVer, chdr.ChannelId, txcc.ChaincodeName, txcc.ChaincodeVersion, chdr.ChannelId)
				logger.Errorf("%+v", err)
				return err, peer.TxValidationCode_EXPIRED_CHAINCODE
			}

//进行VSCC验证
			ctx := &Context{
				Seq:       seq,
				Envelope:  envBytes,
				Block:     block,
				TxID:      chdr.TxId,
				Channel:   chdr.ChannelId,
				Namespace: ns,
				Policy:    policy,
				VSCCName:  vscc.ChaincodeName,
			}
			if err = v.VSCCValidateTxForCC(ctx); err != nil {
				switch err.(type) {
				case *commonerrors.VSCCEndorsementPolicyError:
					return err, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE
				default:
					return err, peer.TxValidationCode_INVALID_OTHER_REASON
				}
			}
		}
	} else {
//确保我们可以调用这个系统链码-如果链码
//无法通过对该对等方的建议调用，我们必须删除
//交易；如果没有，我们就不知道如何决定
//是否有效，因为在v1中，系统链码没有认可策略
		if v.sccprovider.IsSysCCAndNotInvokableExternal(ccID) {
			return errors.Errorf("committing an invocation of cc %s is illegal", ccID),
				peer.TxValidationCode_ILLEGAL_WRITESET
		}

//获取最新的链码版本、VSCC和验证策略
		_, vscc, policy, err := v.GetInfoForValidate(chdr, ccID)
		if err != nil {
			logger.Errorf("GetInfoForValidate for txId = %s returned error: %+v", chdr.TxId, err)
			return err, peer.TxValidationCode_INVALID_OTHER_REASON
		}

//将该事务验证为对该系统链码的调用；
//VSCC必须为此系统链码进行自定义验证
//目前，vscc只对lscc进行自定义验证；如果是hlf
//用户创建可从外部调用的新系统链代码
//他们必须修改VSCC以提供适当的验证
		ctx := &Context{
			Seq:       seq,
			Envelope:  envBytes,
			Block:     block,
			TxID:      chdr.TxId,
			Channel:   chdr.ChannelId,
			Namespace: ccID,
			Policy:    policy,
			VSCCName:  vscc.ChaincodeName,
		}
		if err = v.VSCCValidateTxForCC(ctx); err != nil {
			switch err.(type) {
			case *commonerrors.VSCCEndorsementPolicyError:
				return err, peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE
			default:
				return err, peer.TxValidationCode_INVALID_OTHER_REASON
			}
		}
	}
	logger.Debugf("[%s] VSCCValidateTx completes env bytes %p", chainID, envBytes)
	return nil, peer.TxValidationCode_VALID
}

func (v *VsccValidatorImpl) VSCCValidateTxForCC(ctx *Context) error {
	logger.Debug("Validating", ctx, "with plugin")
	err := v.pluginValidator.ValidateWithPlugin(ctx)
	if err == nil {
		return nil
	}
//如果错误是可插入的验证执行错误，请将其强制转换为常见错误ExecutionFailureError。
	if e, isExecutionError := err.(*validation.ExecutionFailureError); isExecutionError {
		return &commonerrors.VSCCExecutionFailureError{Err: e}
	}
//否则，将其视为背书错误。
	return &commonerrors.VSCCEndorsementPolicyError{Err: err}
}

func (v *VsccValidatorImpl) getCDataForCC(chid, ccid string) (ccprovider.ChaincodeDefinition, error) {
	l := v.support.Ledger()
	if l == nil {
		return nil, errors.New("nil ledger instance")
	}

	qe, err := l.NewQueryExecutor()
	if err != nil {
		return nil, errors.WithMessage(err, "could not retrieve QueryExecutor")
	}
	defer qe.Done()

	bytes, err := qe.GetState("lscc", ccid)
	if err != nil {
		return nil, &commonerrors.VSCCInfoLookupFailureError{
			Reason: fmt.Sprintf("Could not retrieve state for chaincode %s, error %s", ccid, err),
		}
	}

	if bytes == nil {
		return nil, errors.Errorf("lscc's state for [%s] not found.", ccid)
	}

	cd := &ccprovider.ChaincodeData{}
	err = proto.Unmarshal(bytes, cd)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling ChaincodeQueryResponse failed")
	}

	if cd.Vscc == "" {
		return nil, errors.Errorf("lscc's state for [%s] is invalid, vscc field must be set", ccid)
	}

	if len(cd.Policy) == 0 {
		return nil, errors.Errorf("lscc's state for [%s] is invalid, policy field must be set", ccid)
	}

	return cd, err
}

//getinfoforvalidate从lscc获取tx、vscc和policy的chaincodeinstance（最新版本）
func (v *VsccValidatorImpl) GetInfoForValidate(chdr *common.ChannelHeader, ccID string) (*sysccprovider.ChaincodeInstance, *sysccprovider.ChaincodeInstance, []byte, error) {
	cc := &sysccprovider.ChaincodeInstance{
		ChainID:          chdr.ChannelId,
		ChaincodeName:    ccID,
		ChaincodeVersion: coreUtil.GetSysCCVersion(),
	}
	vscc := &sysccprovider.ChaincodeInstance{
		ChainID:          chdr.ChannelId,
ChaincodeName:    "vscc",                     //系统链码的默认VSCC
ChaincodeVersion: coreUtil.GetSysCCVersion(), //获取VSCC版本
	}
	var policy []byte
	var err error
	if !v.sccprovider.IsSysCC(ccID) {
//当我们验证的链码不是
//系统CC，我们需要让CC给我们名字
//以及应该使用的策略

//获取VSCC的名称和策略
		cd, err := v.getCDataForCC(chdr.ChannelId, ccID)
		if err != nil {
			msg := fmt.Sprintf("Unable to get chaincode data from ledger for txid %s, due to %s", chdr.TxId, err)
			logger.Errorf(msg)
			return nil, nil, nil, err
		}
		cc.ChaincodeName = cd.CCName()
		cc.ChaincodeVersion = cd.CCVersion()
		vscc.ChaincodeName, policy = cd.Validation()
	} else {
//在验证系统CC时，我们使用默认值
//VSCC和需要一个签名的默认策略
//来自频道的任何成员
		p := cauthdsl.SignedByAnyMember(v.support.GetMSPIDs(chdr.ChannelId))
		policy, err = utils.Marshal(p)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return cc, vscc, policy, nil
}

//如果提供的NSRwset
//执行分类帐写入
func (v *VsccValidatorImpl) txWritesToNamespace(ns *rwsetutil.NsRwSet) bool {
//首先检查公共写入
	if ns.KvRwSet != nil && len(ns.KvRwSet.Writes) > 0 {
		return true
	}

//只有当我们支持收集数据的能力
	if v.support.Capabilities().PrivateChannelData() {
//检查所有集合的私人写入
		for _, c := range ns.CollHashedRwSets {
			if c.HashedRwSet != nil && len(c.HashedRwSet.HashedWrites) > 0 {
				return true
			}

//只有在我们支持私有元数据写入功能的情况下才能查看它
			if v.support.Capabilities().KeyLevelEndorsement() {
//私有元数据更新
				if c.HashedRwSet != nil && len(c.HashedRwSet.MetadataWrites) > 0 {
					return true
				}
			}
		}
	}

//只有在我们支持元数据写入功能的情况下才能查看元数据写入
	if v.support.Capabilities().KeyLevelEndorsement() {
//公共元数据更新
		if ns.KvRwSet != nil && len(ns.KvRwSet.MetadataWrites) > 0 {
			return true
		}
	}

	return false
}
