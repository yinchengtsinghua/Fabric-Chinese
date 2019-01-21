
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


package utils

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//getpayloads获取事务操作中的底层有效负载对象
func GetPayloads(txActions *peer.TransactionAction) (*peer.ChaincodeActionPayload, *peer.ChaincodeAction, error) {
//TODO:传入tx类型（在接下来的内容中，我们假设
//类型为背书人交易）
	ccPayload, err := GetChaincodeActionPayload(txActions.Payload)
	if err != nil {
		return nil, nil, err
	}

	if ccPayload.Action == nil || ccPayload.Action.ProposalResponsePayload == nil {
		return nil, nil, errors.New("no payload in ChaincodeActionPayload")
	}
	pRespPayload, err := GetProposalResponsePayload(ccPayload.Action.ProposalResponsePayload)
	if err != nil {
		return nil, nil, err
	}

	if pRespPayload.Extension == nil {
		return nil, nil, errors.New("response payload is missing extension")
	}

	respPayload, err := GetChaincodeAction(pRespPayload.Extension)
	if err != nil {
		return ccPayload, nil, err
	}
	return ccPayload, respPayload, nil
}

//GetEnvelopeFromBlock从块的数据字段获取信封。
func GetEnvelopeFromBlock(data []byte) (*common.Envelope, error) {
//块始终以信封开头
	var err error
	env := &common.Envelope{}
	if err = proto.Unmarshal(data, env); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling Envelope")
	}

	return env, nil
}

//CreateSignedDevelope创建所需类型的签名信封，
//已封送datamsg并签名
func CreateSignedEnvelope(txType common.HeaderType, channelID string, signer crypto.LocalSigner, dataMsg proto.Message, msgVersion int32, epoch uint64) (*common.Envelope, error) {
	return CreateSignedEnvelopeWithTLSBinding(txType, channelID, signer, dataMsg, msgVersion, epoch, nil)
}

//CreateSignedDevelopeWithtlsBinding创建所需的签名信封
//键入，并用封送的datamsg对其进行签名。它还包括一个tls证书哈希
//进入通道标题
func CreateSignedEnvelopeWithTLSBinding(txType common.HeaderType, channelID string, signer crypto.LocalSigner, dataMsg proto.Message, msgVersion int32, epoch uint64, tlsCertHash []byte) (*common.Envelope, error) {
	payloadChannelHeader := MakeChannelHeader(txType, msgVersion, channelID, epoch)
	payloadChannelHeader.TlsCertHash = tlsCertHash
	var err error
	payloadSignatureHeader := &common.SignatureHeader{}

	if signer != nil {
		payloadSignatureHeader, err = signer.NewSignatureHeader()
		if err != nil {
			return nil, err
		}
	}

	data, err := proto.Marshal(dataMsg)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling")
	}

	paylBytes := MarshalOrPanic(
		&common.Payload{
			Header: MakePayloadHeader(payloadChannelHeader, payloadSignatureHeader),
			Data:   data,
		},
	)

	var sig []byte
	if signer != nil {
		sig, err = signer.Sign(paylBytes)
		if err != nil {
			return nil, err
		}
	}

	env := &common.Envelope{
		Payload:   paylBytes,
		Signature: sig,
	}

	return env, nil
}

//CreateSignedTx从建议、背书、
//签名人。当客户端调用此函数时，
//为建议收集足够的背书以创建交易和
//提交同行订购
func CreateSignedTx(proposal *peer.Proposal, signer msp.SigningIdentity, resps ...*peer.ProposalResponse) (*common.Envelope, error) {
	if len(resps) == 0 {
		return nil, errors.New("at least one proposal response is required")
	}

//原始标题
	hdr, err := GetHeader(proposal.Header)
	if err != nil {
		return nil, err
	}

//原始有效载荷
	pPayl, err := GetChaincodeProposalPayload(proposal.Payload)
	if err != nil {
		return nil, err
	}

//检查签名者是否与头中引用的签名者相同
//托多：也许值得移除？
	signerBytes, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	shdr, err := GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, err
	}

	if bytes.Compare(signerBytes, shdr.Creator) != 0 {
		return nil, errors.New("signer must be the same as the one referenced in the header")
	}

//获取标题扩展，这样我们就有了可见性字段
	hdrExt, err := GetChaincodeHeaderExtension(hdr)
	if err != nil {
		return nil, err
	}

//确保所有操作按位相等并且成功
	var a1 []byte
	for n, r := range resps {
		if n == 0 {
			a1 = r.Payload
			if r.Response.Status < 200 || r.Response.Status >= 400 {
				return nil, errors.Errorf("proposal response was not successful, error code %d, msg %s", r.Response.Status, r.Response.Message)
			}
			continue
		}

		if bytes.Compare(a1, r.Payload) != 0 {
			return nil, errors.New("ProposalResponsePayloads do not match")
		}
	}

//填写背书
	endorsements := make([]*peer.Endorsement, len(resps))
	for n, r := range resps {
		endorsements[n] = r.Endorsement
	}

//创建链代码认可的操作
	cea := &peer.ChaincodeEndorsedAction{ProposalResponsePayload: resps[0].Payload, Endorsements: endorsements}

//获取将转到事务的建议负载的字节
	propPayloadBytes, err := GetBytesProposalPayloadForTx(pPayl, hdrExt.PayloadVisibility)
	if err != nil {
		return nil, err
	}

//序列化chaincode操作负载
	cap := &peer.ChaincodeActionPayload{ChaincodeProposalPayload: propPayloadBytes, Action: cea}
	capBytes, err := GetBytesChaincodeActionPayload(cap)
	if err != nil {
		return nil, err
	}

//创建交易记录
	taa := &peer.TransactionAction{Header: hdr.SignatureHeader, Payload: capBytes}
	taas := make([]*peer.TransactionAction, 1)
	taas[0] = taa
	tx := &peer.Transaction{Actions: taas}

//序列化Tx
	txBytes, err := GetBytesTransaction(tx)
	if err != nil {
		return nil, err
	}

//创建有效载荷
	payl := &common.Payload{Header: hdr, Data: txBytes}
	paylBytes, err := GetBytesPayload(payl)
	if err != nil {
		return nil, err
	}

//签署有效载荷
	sig, err := signer.Sign(paylBytes)
	if err != nil {
		return nil, err
	}

//这是信封
	return &common.Envelope{Payload: paylBytes, Signature: sig}, nil
}

//CreateProposalResponse创建建议响应。
func CreateProposalResponse(hdrbytes []byte, payl []byte, response *peer.Response, results []byte, events []byte, ccid *peer.ChaincodeID, visibility []byte, signingEndorser msp.SigningIdentity) (*peer.ProposalResponse, error) {
	hdr, err := GetHeader(hdrbytes)
	if err != nil {
		return nil, err
	}

//获取给定建议头、有效负载和
//请求的可见性
	pHashBytes, err := GetProposalHash1(hdr, payl, visibility)
	if err != nil {
		return nil, errors.WithMessage(err, "error computing proposal hash")
	}

//获取提案响应负载的字节数-我们需要对它们进行签名
	prpBytes, err := GetBytesProposalResponsePayload(pHashBytes, response, results, events, ccid)
	if err != nil {
		return nil, err
	}

//序列化签名标识
	endorser, err := signingEndorser.Serialize()
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("error serializing signing identity for %s", signingEndorser.GetIdentifier()))
	}

//签署提案响应和序列化
//具有此背书人密钥的背书人身份
	signature, err := signingEndorser.Sign(append(prpBytes, endorser...))
	if err != nil {
		return nil, errors.WithMessage(err, "could not sign the proposal response payload")
	}

	resp := &peer.ProposalResponse{
//时间戳：todo！
Version: 1, //TODO:选择正确的版本号
		Endorsement: &peer.Endorsement{
			Signature: signature,
			Endorser:  endorser,
		},
		Payload: prpBytes,
		Response: &peer.Response{
			Status:  200,
			Message: "OK",
		},
	}

	return resp, nil
}

//CreateProposalResponseFailure为以下情况创建建议响应：
//背书提议因背书失败或
//链码故障（链码响应状态>=Shim.ErrorThreshold）
func CreateProposalResponseFailure(hdrbytes []byte, payl []byte, response *peer.Response, results []byte, events []byte, ccid *peer.ChaincodeID, visibility []byte) (*peer.ProposalResponse, error) {
	hdr, err := GetHeader(hdrbytes)
	if err != nil {
		return nil, err
	}

//获取给定建议头、有效负载和请求的可见性的建议哈希
	pHashBytes, err := GetProposalHash1(hdr, payl, visibility)
	if err != nil {
		return nil, errors.WithMessage(err, "error computing proposal hash")
	}

//获取建议响应负载的字节数
	prpBytes, err := GetBytesProposalResponsePayload(pHashBytes, response, results, events, ccid)
	if err != nil {
		return nil, err
	}

	resp := &peer.ProposalResponse{
//时间戳：todo！
		Payload:  prpBytes,
		Response: response,
	}

	return resp, nil
}

//GetSignedProposal返回一个已签名的建议，给出建议消息和
//签名身份
func GetSignedProposal(prop *peer.Proposal, signer msp.SigningIdentity) (*peer.SignedProposal, error) {
//检查无论点
	if prop == nil || signer == nil {
		return nil, errors.New("nil arguments")
	}

	propBytes, err := GetBytesProposal(prop)
	if err != nil {
		return nil, err
	}

	signature, err := signer.Sign(propBytes)
	if err != nil {
		return nil, err
	}

	return &peer.SignedProposal{ProposalBytes: propBytes, Signature: signature}, nil
}

//mockSignedOnersProposalOrpanic使用
//传递的参数
func MockSignedEndorserProposalOrPanic(chainID string, cs *peer.ChaincodeSpec, creator, signature []byte) (*peer.SignedProposal, *peer.Proposal) {
	prop, _, err := CreateChaincodeProposal(
		common.HeaderType_ENDORSER_TRANSACTION,
		chainID,
		&peer.ChaincodeInvocationSpec{ChaincodeSpec: cs},
		creator)
	if err != nil {
		panic(err)
	}

	propBytes, err := GetBytesProposal(prop)
	if err != nil {
		panic(err)
	}

	return &peer.SignedProposal{ProposalBytes: propBytes, Signature: signature}, prop
}

func MockSignedEndorserProposal2OrPanic(chainID string, cs *peer.ChaincodeSpec, signer msp.SigningIdentity) (*peer.SignedProposal, *peer.Proposal) {
	serializedSigner, err := signer.Serialize()
	if err != nil {
		panic(err)
	}

	prop, _, err := CreateChaincodeProposal(
		common.HeaderType_ENDORSER_TRANSACTION,
		chainID,
		&peer.ChaincodeInvocationSpec{ChaincodeSpec: &peer.ChaincodeSpec{}},
		serializedSigner)
	if err != nil {
		panic(err)
	}

	sProp, err := GetSignedProposal(prop, signer)
	if err != nil {
		panic(err)
	}

	return sProp, prop
}

//GetBytesProposalPayloadfortx接受chaincodeProposalPayload并返回
//它的序列化版本根据可见性字段
func GetBytesProposalPayloadForTx(payload *peer.ChaincodeProposalPayload, visibility []byte) ([]byte, error) {
//检查无论点
	if payload == nil {
		return nil, errors.New("nil arguments")
	}

//从有效负载中去掉瞬态字节-需要这样做否
//无论是能见度模式
	cppNoTransient := &peer.ChaincodeProposalPayload{Input: payload.Input, TransientMap: nil}
	cppBytes, err := GetBytesChaincodeProposalPayload(cppNoTransient)
	if err != nil {
		return nil, err
	}

//目前，结构只支持完全可见性：这意味着
//对提案有效载荷的哪一部分没有限制
//在最终事务中可见；此默认方法要求
//但是，payloadVisibility字段中没有其他说明；
//织物可以延伸以编码更精细的可视性。
//应在此字段中编码的机制（并处理
//同龄人适当）

	return cppBytes, nil
}

//getProposalHash2获取建议哈希-此版本
//由提交者调用，其中可见性策略
//已经被强制执行了，所以我们已经得到了
//我们得上车
func GetProposalHash2(header *common.Header, ccPropPayl []byte) ([]byte, error) {
//检查无论点
	if header == nil ||
		header.ChannelHeader == nil ||
		header.SignatureHeader == nil ||
		ccPropPayl == nil {
		return nil, errors.New("nil arguments")
	}

	hash, err := factory.GetDefault().GetHash(&bccsp.SHA256Opts{})
	if err != nil {
		return nil, errors.WithMessage(err, "error instantiating hash function")
	}
//散列序列化的通道头对象
	hash.Write(header.ChannelHeader)
//散列序列化签名头对象
	hash.Write(header.SignatureHeader)
//散列给定的chaincode建议负载的字节
	hash.Write(ccPropPayl)
	return hash.Sum(nil), nil
}

//GetProposalHash1在清理
//根据可见性规则的chaincode建议有效负载
func GetProposalHash1(header *common.Header, ccPropPayl []byte, visibility []byte) ([]byte, error) {
//检查无论点
	if header == nil ||
		header.ChannelHeader == nil ||
		header.SignatureHeader == nil ||
		ccPropPayl == nil {
		return nil, errors.New("nil arguments")
	}

//取消标记链码建议负载
	cpp, err := GetChaincodeProposalPayload(ccPropPayl)
	if err != nil {
		return nil, err
	}

	ppBytes, err := GetBytesProposalPayloadForTx(cpp, visibility)
	if err != nil {
		return nil, err
	}

	hash2, err := factory.GetDefault().GetHash(&bccsp.SHA256Opts{})
	if err != nil {
		return nil, errors.WithMessage(err, "error instantiating hash function")
	}
//散列序列化的通道头对象
	hash2.Write(header.ChannelHeader)
//散列序列化签名头对象
	hash2.Write(header.SignatureHeader)
//将转到Tx的chaincode建议有效负载部分的哈希
	hash2.Write(ppBytes)
	return hash2.Sum(nil), nil
}
