
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
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/common/crypto"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//marshalorpanic序列化protobuf消息，如果此
//操作失败
func MarshalOrPanic(pb proto.Message) []byte {
	data, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}
	return data
}

//marshal序列化protobuf消息。
func Marshal(pb proto.Message) ([]byte, error) {
	return proto.Marshal(pb)
}

//CreateNoncoorpanic使用公共/加密包生成一个nonce
//如果这个操作失败的话会很恐慌。
func CreateNonceOrPanic() []byte {
	nonce, err := CreateNonce()
	if err != nil {
		panic(err)
	}
	return nonce
}

//creatence使用common/crypto包生成一个nonce。
func CreateNonce() ([]byte, error) {
	nonce, err := crypto.GetRandomNonce()
	return nonce, errors.WithMessage(err, "error generating random nonce")
}

//将字节取消标记为有效负载结构或PANIC
//关于误差
func UnmarshalPayloadOrPanic(encoded []byte) *cb.Payload {
	payload, err := UnmarshalPayload(encoded)
	if err != nil {
		panic(err)
	}
	return payload
}

//将字节取消标记为有效负载结构
func UnmarshalPayload(encoded []byte) (*cb.Payload, error) {
	payload := &cb.Payload{}
	err := proto.Unmarshal(encoded, payload)
	return payload, errors.Wrap(err, "error unmarshaling Payload")
}

//将字节取消标记为信封结构或panics
//关于误差
func UnmarshalEnvelopeOrPanic(encoded []byte) *cb.Envelope {
	envelope, err := UnmarshalEnvelope(encoded)
	if err != nil {
		panic(err)
	}
	return envelope
}

//将字节取消标记为信封结构
func UnmarshalEnvelope(encoded []byte) (*cb.Envelope, error) {
	envelope := &cb.Envelope{}
	err := proto.Unmarshal(encoded, envelope)
	return envelope, errors.Wrap(err, "error unmarshaling Envelope")
}

//将字节取消标记为块结构或panics
//关于误差
func UnmarshalBlockOrPanic(encoded []byte) *cb.Block {
	block, err := UnmarshalBlock(encoded)
	if err != nil {
		panic(err)
	}
	return block
}

//取消标记块将字节取消标记为块结构
func UnmarshalBlock(encoded []byte) (*cb.Block, error) {
	block := &cb.Block{}
	err := proto.Unmarshal(encoded, block)
	return block, errors.Wrap(err, "error unmarshaling Block")
}

//unmashlendevelopeoftype取消标记指定类型的信封，
//包括有效载荷数据的解组
func UnmarshalEnvelopeOfType(envelope *cb.Envelope, headerType cb.HeaderType, message proto.Message) (*cb.ChannelHeader, error) {
	payload, err := UnmarshalPayload(envelope.Payload)
	if err != nil {
		return nil, err
	}

	if payload.Header == nil {
		return nil, errors.New("envelope must have a Header")
	}

	chdr, err := UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, err
	}

	if chdr.Type != int32(headerType) {
		return nil, errors.Errorf("invalid type %s, expected %s", cb.HeaderType(chdr.Type), headerType)
	}

	err = proto.Unmarshal(payload.Data, message)
	err = errors.Wrapf(err, "error unmarshaling message for type %s", headerType)
	return chdr, err
}

//ExtractEnvelopeOrpanic从给定的块中检索请求的信封
//取消标记——如果这些操作中的任何一个失败，它都会恐慌。
func ExtractEnvelopeOrPanic(block *cb.Block, index int) *cb.Envelope {
	envelope, err := ExtractEnvelope(block, index)
	if err != nil {
		panic(err)
	}
	return envelope
}

//提取信封从给定的块中检索请求的信封，并
//解封它
func ExtractEnvelope(block *cb.Block, index int) (*cb.Envelope, error) {
	if block.Data == nil {
		return nil, errors.New("block data is nil")
	}

	envelopeCount := len(block.Data.Data)
	if index < 0 || index >= envelopeCount {
		return nil, errors.New("envelope index out of bounds")
	}
	marshaledEnvelope := block.Data.Data[index]
	envelope, err := GetEnvelopeFromBlock(marshaledEnvelope)
	err = errors.WithMessage(err, fmt.Sprintf("block data does not carry an envelope at index %d", index))
	return envelope, err
}

//extractpayloadorpanic检索给定信封的有效负载，并
//取消标记——如果这些操作中的任何一个失败，它都会恐慌。
func ExtractPayloadOrPanic(envelope *cb.Envelope) *cb.Payload {
	payload, err := ExtractPayload(envelope)
	if err != nil {
		panic(err)
	}
	return payload
}

//ExtractPayload检索给定信封的有效负载并将其取消标记。
func ExtractPayload(envelope *cb.Envelope) (*cb.Payload, error) {
	payload := &cb.Payload{}
	err := proto.Unmarshal(envelope.Payload, payload)
	err = errors.Wrap(err, "no payload in envelope")
	return payload, err
}

//makechannelheader创建一个channelheader。
func MakeChannelHeader(headerType cb.HeaderType, version int32, chainID string, epoch uint64) *cb.ChannelHeader {
	return &cb.ChannelHeader{
		Type:    int32(headerType),
		Version: version,
		Timestamp: &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   0,
		},
		ChannelId: chainID,
		Epoch:     epoch,
	}
}

//MakeSignatureHeader创建一个SignatureHeader。
func MakeSignatureHeader(serializedCreatorCertChain []byte, nonce []byte) *cb.SignatureHeader {
	return &cb.SignatureHeader{
		Creator: serializedCreatorCertChain,
		Nonce:   nonce,
	}
}

//setxid根据提供的签名头生成事务ID
//并在通道标题中设置txid字段
func SetTxID(channelHeader *cb.ChannelHeader, signatureHeader *cb.SignatureHeader) error {
	txid, err := ComputeTxID(
		signatureHeader.Nonce,
		signatureHeader.Creator,
	)
	if err != nil {
		return err
	}
	channelHeader.TxId = txid
	return nil
}

//makePayloadHeader创建有效负载头。
func MakePayloadHeader(ch *cb.ChannelHeader, sh *cb.SignatureHeader) *cb.Header {
	return &cb.Header{
		ChannelHeader:   MarshalOrPanic(ch),
		SignatureHeader: MarshalOrPanic(sh),
	}
}

//NewSignatureHeaderOrpanic返回签名头并在出错时恐慌。
func NewSignatureHeaderOrPanic(signer crypto.LocalSigner) *cb.SignatureHeader {
	if signer == nil {
		panic(errors.New("invalid signer. cannot be nil"))
	}

	signatureHeader, err := signer.NewSignatureHeader()
	if err != nil {
		panic(fmt.Errorf("failed generating a new SignatureHeader: %s", err))
	}
	return signatureHeader
}

//signorpanic对消息进行签名，并在出错时惊慌失措。
func SignOrPanic(signer crypto.LocalSigner, msg []byte) []byte {
	if signer == nil {
		panic(errors.New("invalid signer. cannot be nil"))
	}

	sigma, err := signer.Sign(msg)
	if err != nil {
		panic(fmt.Errorf("failed generating signature: %s", err))
	}
	return sigma
}

//UnmarshalChannelHeader从字节返回ChannelHeader
func UnmarshalChannelHeader(bytes []byte) (*cb.ChannelHeader, error) {
	chdr := &cb.ChannelHeader{}
	err := proto.Unmarshal(bytes, chdr)
	return chdr, errors.Wrap(err, "error unmarshaling ChannelHeader")
}

//取消对频道头或频道头的标记将字节取消对频道头或频道头的标记
//关于误差
func UnmarshalChannelHeaderOrPanic(bytes []byte) *cb.ChannelHeader {
	chdr, err := UnmarshalChannelHeader(bytes)
	if err != nil {
		panic(err)
	}
	return chdr
}

//unmashalchaincodeid从字节返回chaincodeid
func UnmarshalChaincodeID(bytes []byte) (*pb.ChaincodeID, error) {
	ccid := &pb.ChaincodeID{}
	err := proto.Unmarshal(bytes, ccid)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling ChaincodeID")
	}

	return ccid, nil
}

//只要给定的块包含配置，isconfigBlock就会进行验证。
//更新交易记录
func IsConfigBlock(block *cb.Block) bool {
	envelope, err := ExtractEnvelope(block, 0)
	if err != nil {
		return false
	}

	payload, err := GetPayload(envelope)
	if err != nil {
		return false
	}

	if payload.Header == nil {
		return false
	}

	hdr, err := UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return false
	}

	return cb.HeaderType(hdr.Type) == cb.HeaderType_CONFIG || cb.HeaderType(hdr.Type) == cb.HeaderType_ORDERER_TRANSACTION
}

//channelheader返回给定*cb.envelope的*cb.channelheader。
func ChannelHeader(env *cb.Envelope) (*cb.ChannelHeader, error) {
	envPayload, err := UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, err
	}

	if envPayload.Header == nil {
		return nil, errors.New("header not set")
	}

	if envPayload.Header.ChannelHeader == nil {
		return nil, errors.New("channel header not set")
	}

	chdr, err := UnmarshalChannelHeader(envPayload.Header.ChannelHeader)
	if err != nil {
		return nil, errors.WithMessage(err, "error unmarshaling channel header")
	}

	return chdr, nil
}

//channel id返回给定*cb.envelope的通道ID。
func ChannelID(env *cb.Envelope) (string, error) {
	chdr, err := ChannelHeader(env)
	if err != nil {
		return "", errors.WithMessage(err, "error retrieving channel header")
	}

	return chdr.ChannelId, nil
}

//EnvelopeToConfigUpdate用于从
//类型配置\更新
func EnvelopeToConfigUpdate(configtx *cb.Envelope) (*cb.ConfigUpdateEnvelope, error) {
	configUpdateEnv := &cb.ConfigUpdateEnvelope{}
	_, err := UnmarshalEnvelopeOfType(configtx, cb.HeaderType_CONFIG_UPDATE, configUpdateEnv)
	if err != nil {
		return nil, err
	}
	return configUpdateEnv, nil
}
