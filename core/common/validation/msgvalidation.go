
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2016保留所有权利。

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


package validation

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/flogging"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var putilsLogger = flogging.MustGetLogger("protoutils")

//validateChaincodeProposalMessage检查chaincode类型的建议消息的有效性
func validateChaincodeProposalMessage(prop *pb.Proposal, hdr *common.Header) (*pb.ChaincodeHeaderExtension, error) {
	if prop == nil || hdr == nil {
		return nil, errors.New("nil arguments")
	}

	putilsLogger.Debugf("validateChaincodeProposalMessage starts for proposal %p, header %p", prop, hdr)

//4）根据头类型（假设它是chaincode），查看扩展
	chaincodeHdrExt, err := utils.GetChaincodeHeaderExtension(hdr)
	if err != nil {
		return nil, errors.New("invalid header extension for type CHAINCODE")
	}

	if chaincodeHdrExt.ChaincodeId == nil {
		return nil, errors.New("ChaincodeHeaderExtension.ChaincodeId is nil")
	}

	putilsLogger.Debugf("validateChaincodeProposalMessage info: header extension references chaincode %s", chaincodeHdrExt.ChaincodeId)

//-确保chaincodeid正确（？）
//托多：我们应该这样做吗？如果是，使用哪个接口？

//-确保可见性字段具有我们理解的某些值
//目前，结构只支持完全可见性：这意味着
//there are no restrictions on which parts of the proposal payload will
//be visible in the final transaction; this default approach requires
//payloadVisibility字段中没有其他说明
//因此，预计为零；然而，织物可延伸至
//编码更详细的可见性机制，这些机制应编码为
//this field (and handled appropriately by the peer)
	if chaincodeHdrExt.PayloadVisibility != nil {
		return nil, errors.New("invalid payload visibility field")
	}

	return chaincodeHdrExt, nil
}

//validateProposalMessage检查已签名的Proposal消息的有效性
//此函数返回header和chaincodeheaderextension消息，因为它们
//已取消标记和验证
func ValidateProposalMessage(signedProp *pb.SignedProposal) (*pb.Proposal, *common.Header, *pb.ChaincodeHeaderExtension, error) {
	if signedProp == nil {
		return nil, nil, nil, errors.New("nil arguments")
	}

	putilsLogger.Debugf("ValidateProposalMessage starts for signed proposal %p", signedProp)

//从SignedProp中提取建议消息
	prop, err := utils.GetProposal(signedProp.ProposalBytes)
	if err != nil {
		return nil, nil, nil, err
	}

//1）查看ProposalHeader
	hdr, err := utils.GetHeader(prop.Header)
	if err != nil {
		return nil, nil, nil, err
	}

//验证标题
	chdr, shdr, err := validateCommonHeader(hdr)
	if err != nil {
		return nil, nil, nil, err
	}

//验证签名
	err = checkSignatureFromCreator(shdr.Creator, signedProp.Signature, signedProp.ProposalBytes, chdr.ChannelId)
	if err != nil {
//在对等机上记录准确的消息，但返回一般错误消息到
//避免恶意用户扫描频道
		putilsLogger.Warningf("channel [%s]: %s", chdr.ChannelId, err)
		sId := &msp.SerializedIdentity{}
		err := proto.Unmarshal(shdr.Creator, sId)
		if err != nil {
//在这里记录错误，但仍然只返回一般错误
			err = errors.Wrap(err, "could not deserialize a SerializedIdentity")
			putilsLogger.Warningf("channel [%s]: %s", chdr.ChannelId, err)
		}
		return nil, nil, nil, errors.Errorf("access denied: channel [%s] creator org [%s]", chdr.ChannelId, sId.Mspid)
	}

//验证是否正确计算了事务ID。
//需要进行此检查以确保在分类帐中查找
//对于相同的txid捕获重复项。
	err = utils.CheckTxID(
		chdr.TxId,
		shdr.Nonce,
		shdr.Creator)
	if err != nil {
		return nil, nil, nil, err
	}

//根据头中指定的类型继续验证
	switch common.HeaderType(chdr.Type) {
	case common.HeaderType_CONFIG:
//哪些类型不同验证相同
//也就是说，将提案验证为链码。如果我们需要其他的
//配置的特殊验证，我们必须实现
//特殊验证
		fallthrough
	case common.HeaderType_ENDORSER_TRANSACTION:
//validation of the proposal message knowing it's of type CHAINCODE
		chaincodeHdrExt, err := validateChaincodeProposalMessage(prop, hdr)
		if err != nil {
			return nil, nil, nil, err
		}

		return prop, hdr, chaincodeHdrExt, err
	default:
//注：我们很可能需要一个案子
		return nil, nil, nil, errors.Errorf("unsupported proposal type %d", common.HeaderType(chdr.Type))
	}
}

//给一个创建者，一条信息和一个签名，
//如果创建者
//是有效的证书，签名有效
func checkSignatureFromCreator(creatorBytes []byte, sig []byte, msg []byte, ChainID string) error {
	putilsLogger.Debugf("begin")

//check for nil argument
	if creatorBytes == nil || sig == nil || msg == nil {
		return errors.New("nil arguments")
	}

	mspObj := mspmgmt.GetIdentityDeserializer(ChainID)
	if mspObj == nil {
		return errors.Errorf("could not get msp for channel [%s]", ChainID)
	}

//获取创建者的身份
	creator, err := mspObj.DeserializeIdentity(creatorBytes)
	if err != nil {
		return errors.WithMessage(err, "MSP error")
	}

	putilsLogger.Debugf("creator is %s", creator.GetIdentifier())

//确保创建者是有效的证书
	err = creator.Validate()
	if err != nil {
		return errors.WithMessage(err, "creator certificate is not valid")
	}

	putilsLogger.Debugf("creator is valid")

//验证签名
	err = creator.Verify(msg, sig)
	if err != nil {
		return errors.WithMessage(err, "creator's signature over the proposal is not valid")
	}

	putilsLogger.Debugf("exits successfully")

	return nil
}

//检查有效的SignatureHeader
func validateSignatureHeader(sHdr *common.SignatureHeader) error {
//检查无论点
	if sHdr == nil {
		return errors.New("nil SignatureHeader provided")
	}

//确保有一次
	if sHdr.Nonce == nil || len(sHdr.Nonce) == 0 {
		return errors.New("invalid nonce specified in the header")
	}

//确保有创造者
	if sHdr.Creator == nil || len(sHdr.Creator) == 0 {
		return errors.New("invalid creator specified in the header")
	}

	return nil
}

//检查有效的channelheader
func validateChannelHeader(cHdr *common.ChannelHeader) error {
//检查无论点
	if cHdr == nil {
		return errors.New("nil ChannelHeader provided")
	}

//验证头类型
	if common.HeaderType(cHdr.Type) != common.HeaderType_ENDORSER_TRANSACTION &&
		common.HeaderType(cHdr.Type) != common.HeaderType_CONFIG_UPDATE &&
		common.HeaderType(cHdr.Type) != common.HeaderType_CONFIG &&
		common.HeaderType(cHdr.Type) != common.HeaderType_TOKEN_TRANSACTION {
		return errors.Errorf("invalid header type %s", common.HeaderType(cHdr.Type))
	}

	putilsLogger.Debugf("validateChannelHeader info: header type %d", common.HeaderType(cHdr.Type))

//TODO:验证chdr.chainID中的chainID

//验证chdr.epoch中的epoch
//Currently we enforce that Epoch is 0.
//TODO:此检查将在epoch管理之后修改
//将到位。
	if cHdr.Epoch != 0 {
		return errors.Errorf("invalid Epoch in ChannelHeader. Expected 0, got [%d]", cHdr.Epoch)
	}

//TODO:验证chdr.version中的版本

	return nil
}

//检查有效的标题
func validateCommonHeader(hdr *common.Header) (*common.ChannelHeader, *common.SignatureHeader, error) {
	if hdr == nil {
		return nil, nil, errors.New("nil header")
	}

	chdr, err := utils.UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, nil, err
	}

	shdr, err := utils.GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, nil, err
	}

	err = validateChannelHeader(chdr)
	if err != nil {
		return nil, nil, err
	}

	err = validateSignatureHeader(shdr)
	if err != nil {
		return nil, nil, err
	}

	return chdr, shdr, nil
}

//validateConfigTransaction验证
//假设其类型为config的事务
func validateConfigTransaction(data []byte, hdr *common.Header) error {
	putilsLogger.Debugf("validateConfigTransaction starts for data %p, header %s", data, hdr)

//检查无论点
	if data == nil || hdr == nil {
		return errors.New("nil arguments")
	}

//这里不需要进行此验证，configtx.validator将处理此验证

	return nil
}

//validateEndorSertTransaction验证
//假设其类型为背书人交易的交易
func validateEndorserTransaction(data []byte, hdr *common.Header) error {
	putilsLogger.Debugf("validateEndorserTransaction starts for data %p, header %s", data, hdr)

//检查无论点
	if data == nil || hdr == nil {
		return errors.New("nil arguments")
	}

//如果类型为“背书人交易”，则取消标记交易消息
	tx, err := utils.GetTransaction(data)
	if err != nil {
		return err
	}

//检查无论点
	if tx == nil {
		return errors.New("nil transaction")
	}

//ToDO：验证TX.版本

//TODO:验证链码头扩展

//HLF版本1只支持每个事务一个操作
	if len(tx.Actions) != 1 {
		return errors.Errorf("only one action per transaction is supported, tx contains %d", len(tx.Actions))
	}

	putilsLogger.Debugf("validateEndorserTransaction info: there are %d actions", len(tx.Actions))

	for _, act := range tx.Actions {
//检查无论点
		if act == nil {
			return errors.New("nil action")
		}

//如果类型为“背书人”交易，则取消签名人的标记。
		sHdr, err := utils.GetSignatureHeader(act.Header)
		if err != nil {
			return err
		}

//验证签名负责人-这里我们实际上只
//care about the nonce since the creator is in the outer header
		err = validateSignatureHeader(sHdr)
		if err != nil {
			return err
		}

		putilsLogger.Debugf("validateEndorserTransaction info: signature header is valid")

//如果类型为“认可者事务”，则取消标记chaincodeactionPayload
		ccActionPayload, err := utils.GetChaincodeActionPayload(act.Payload)
		if err != nil {
			return err
		}

//提取提案响应负载
		prp, err := utils.GetProposalResponsePayload(ccActionPayload.Action.ProposalResponsePayload)
		if err != nil {
			return err
		}

//build the original header by stitching together
//公共通道头和每个操作签名头
		hdrOrig := &common.Header{ChannelHeader: hdr.ChannelHeader, SignatureHeader: act.Header}

//计算ProposalHash
		pHash, err := utils.GetProposalHash2(hdrOrig, ccActionPayload.ChaincodeProposalPayload)
		if err != nil {
			return err
		}

//确保建议哈希匹配
		if bytes.Compare(pHash, prp.ProposalHash) != 0 {
			return errors.New("proposal hash does not match")
		}
	}

	return nil
}

//validateTransaction检查事务信封的格式是否正确
func ValidateTransaction(e *common.Envelope, c channelconfig.ApplicationCapabilities) (*common.Payload, pb.TxValidationCode) {
	putilsLogger.Debugf("ValidateTransactionEnvelope starts for envelope %p", e)

//检查无论点
	if e == nil {
		putilsLogger.Errorf("Error: nil envelope")
		return nil, pb.TxValidationCode_NIL_ENVELOPE
	}

//从信封中获取有效载荷
	payload, err := utils.GetPayload(e)
	if err != nil {
		putilsLogger.Errorf("GetPayload returns err %s", err)
		return nil, pb.TxValidationCode_BAD_PAYLOAD
	}

	putilsLogger.Debugf("Header is %s", payload.Header)

//验证标题
	chdr, shdr, err := validateCommonHeader(payload.Header)
	if err != nil {
		putilsLogger.Errorf("validateCommonHeader returns err %s", err)
		return nil, pb.TxValidationCode_BAD_COMMON_HEADER
	}

//验证信封中的签名
	err = checkSignatureFromCreator(shdr.Creator, e.Signature, e.Payload, chdr.ChannelId)
	if err != nil {
		putilsLogger.Errorf("checkSignatureFromCreator returns err %s", err)
		return nil, pb.TxValidationCode_BAD_CREATOR_SIGNATURE
	}

//TODO：确保创建者可以与我们进行交易（一些ACL？）哪一组API应该向我们提供这些信息？

//根据头中指定的类型继续验证
	switch common.HeaderType(chdr.Type) {
	case common.HeaderType_ENDORSER_TRANSACTION:
//验证是否正确计算了事务ID。
//需要进行此检查以确保在分类帐中查找
//对于相同的txid捕获重复项。
		err = utils.CheckTxID(
			chdr.TxId,
			shdr.Nonce,
			shdr.Creator)

		if err != nil {
			putilsLogger.Errorf("CheckTxID returns err %s", err)
			return nil, pb.TxValidationCode_BAD_PROPOSAL_TXID
		}

		err = validateEndorserTransaction(payload.Data, payload.Header)
		putilsLogger.Debugf("ValidateTransactionEnvelope returns err %s", err)

		if err != nil {
			putilsLogger.Errorf("validateEndorserTransaction returns err %s", err)
			return payload, pb.TxValidationCode_INVALID_ENDORSER_TRANSACTION
		} else {
			return payload, pb.TxValidationCode_VALID
		}
	case common.HeaderType_CONFIG:
//配置事务中有将被验证的签名，特别是在Genesis中可能没有创建者或
//最外层信封上的签名

		err = validateConfigTransaction(payload.Data, payload.Header)

		if err != nil {
			putilsLogger.Errorf("validateConfigTransaction returns err %s", err)
			return payload, pb.TxValidationCode_INVALID_CONFIG_TRANSACTION
		} else {
			return payload, pb.TxValidationCode_VALID
		}
	case common.HeaderType_TOKEN_TRANSACTION:
//验证是否正确计算了事务ID。
//需要进行此检查以确保在分类帐中查找
//对于相同的txid捕获重复项。
		err = utils.CheckTxID(
			chdr.TxId,
			shdr.Nonce,
			shdr.Creator)

		if err != nil {
			putilsLogger.Errorf("CheckTxID returns err %s", err)
			return nil, pb.TxValidationCode_BAD_PROPOSAL_TXID
		}

		return payload, pb.TxValidationCode_VALID
	default:
		return nil, pb.TxValidationCode_UNSUPPORTED_TX_PAYLOAD
	}
}
