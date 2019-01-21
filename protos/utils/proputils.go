
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
	"encoding/binary"
	"encoding/hex"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//get chaincodeinvocationspec从建议获取chaincodeinvocationspec
func GetChaincodeInvocationSpec(prop *peer.Proposal) (*peer.ChaincodeInvocationSpec, error) {
	if prop == nil {
		return nil, errors.New("proposal is nil")
	}
	_, err := GetHeader(prop.Header)
	if err != nil {
		return nil, err
	}
	ccPropPayload, err := GetChaincodeProposalPayload(prop.Payload)
	if err != nil {
		return nil, err
	}
	cis := &peer.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(ccPropPayload.Input, cis)
	return cis, errors.Wrap(err, "error unmarshaling ChaincodeInvocationSpec")
}

//getchaincoderoposalContext返回creator和transient
func GetChaincodeProposalContext(prop *peer.Proposal) ([]byte, map[string][]byte, error) {
	if prop == nil {
		return nil, nil, errors.New("proposal is nil")
	}
	if len(prop.Header) == 0 {
		return nil, nil, errors.New("proposal's header is nil")
	}
	if len(prop.Payload) == 0 {
		return nil, nil, errors.New("proposal's payload is nil")
	}
//回到头上
	hdr, err := GetHeader(prop.Header)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "error extracting header from proposal")
	}
	if hdr == nil {
		return nil, nil, errors.New("unmarshaled header is nil")
	}

	chdr, err := UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "error extracting channel header from proposal")
	}

	if err = validateChannelHeaderType(chdr, []common.HeaderType{common.HeaderType_ENDORSER_TRANSACTION, common.HeaderType_CONFIG}); err != nil {
		return nil, nil, errors.WithMessage(err, "invalid proposal")
	}

	shdr, err := GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "error extracting signature header from proposal")
	}

	ccPropPayload, err := GetChaincodeProposalPayload(prop.Payload)
	if err != nil {
		return nil, nil, err
	}

	return shdr.Creator, ccPropPayload.TransientMap, nil
}

func validateChannelHeaderType(chdr *common.ChannelHeader, expectedTypes []common.HeaderType) error {
	for _, t := range expectedTypes {
		if common.HeaderType(chdr.Type) == t {
			return nil
		}
	}
	return errors.Errorf("invalid channel header type. expected one of %s, received %s", expectedTypes, common.HeaderType(chdr.Type))
}

//从字节获取头
func GetHeader(bytes []byte) (*common.Header, error) {
	hdr := &common.Header{}
	err := proto.Unmarshal(bytes, hdr)
	return hdr, errors.Wrap(err, "error unmarshaling Header")
}

//getnonce返回建议中使用的nonce
func GetNonce(prop *peer.Proposal) ([]byte, error) {
	if prop == nil {
		return nil, errors.New("proposal is nil")
	}

//回到头上
	hdr, err := GetHeader(prop.Header)
	if err != nil {
		return nil, err
	}

	chdr, err := UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, err
	}

	if err = validateChannelHeaderType(chdr, []common.HeaderType{common.HeaderType_ENDORSER_TRANSACTION, common.HeaderType_CONFIG}); err != nil {
		return nil, errors.WithMessage(err, "invalid proposal")
	}

	shdr, err := GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, err
	}

	if hdr.SignatureHeader == nil {
		return nil, errors.New("invalid signature header. cannot be nil")
	}

	return shdr.Nonce, nil
}

//get chaincode header extension获取给定头段的链码头扩展
func GetChaincodeHeaderExtension(hdr *common.Header) (*peer.ChaincodeHeaderExtension, error) {
	chdr, err := UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, err
	}

	chaincodeHdrExt := &peer.ChaincodeHeaderExtension{}
	err = proto.Unmarshal(chdr.Extension, chaincodeHdrExt)
	return chaincodeHdrExt, errors.Wrap(err, "error unmarshaling ChaincodeHeaderExtension")
}

//GetProposalResponse给定的建议（字节）
func GetProposalResponse(prBytes []byte) (*peer.ProposalResponse, error) {
	proposalResponse := &peer.ProposalResponse{}
	err := proto.Unmarshal(prBytes, proposalResponse)
	return proposalResponse, errors.Wrap(err, "error unmarshaling ProposalResponse")
}

//getchaincodedeploymentspec返回给定参数的chaincodedeploymentspec
func GetChaincodeDeploymentSpec(code []byte, pr *platforms.Registry) (*peer.ChaincodeDeploymentSpec, error) {
	cds := &peer.ChaincodeDeploymentSpec{}
	err := proto.Unmarshal(code, cds)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling ChaincodeDeploymentSpec")
	}

//FAB-2122：根据平台特定要求验证CDS
	return cds, pr.ValidateDeploymentSpec(cds.CCType(), cds.Bytes())
}

//getchaincodeaction获取给定chaincode action字节的chaincodeaction
func GetChaincodeAction(caBytes []byte) (*peer.ChaincodeAction, error) {
	chaincodeAction := &peer.ChaincodeAction{}
	err := proto.Unmarshal(caBytes, chaincodeAction)
	return chaincodeAction, errors.Wrap(err, "error unmarshaling ChaincodeAction")
}

//GetResponse获取给定响应字节的响应
func GetResponse(resBytes []byte) (*peer.Response, error) {
	response := &peer.Response{}
	err := proto.Unmarshal(resBytes, response)
	return response, errors.Wrap(err, "error unmarshaling Response")
}

//getchaincodevents获取给定chaincode事件字节的chaincodeevents
func GetChaincodeEvents(eBytes []byte) (*peer.ChaincodeEvent, error) {
	chaincodeEvent := &peer.ChaincodeEvent{}
	err := proto.Unmarshal(eBytes, chaincodeEvent)
	return chaincodeEvent, errors.Wrap(err, "error unmarshaling ChaicnodeEvent")
}

//GetProposalResponsePayLoad获取提案响应负载
func GetProposalResponsePayload(prpBytes []byte) (*peer.ProposalResponsePayload, error) {
	prp := &peer.ProposalResponsePayload{}
	err := proto.Unmarshal(prpBytes, prp)
	return prp, errors.Wrap(err, "error unmarshaling ProposalResponsePayload")
}

//GetProposal从其字节返回建议消息
func GetProposal(propBytes []byte) (*peer.Proposal, error) {
	prop := &peer.Proposal{}
	err := proto.Unmarshal(propBytes, prop)
	return prop, errors.Wrap(err, "error unmarshaling Proposal")
}

//从信封消息获取有效负载
func GetPayload(e *common.Envelope) (*common.Payload, error) {
	payload := &common.Payload{}
	err := proto.Unmarshal(e.Payload, payload)
	return payload, errors.Wrap(err, "error unmarshaling Payload")
}

//GetTransaction从字节获取事务
func GetTransaction(txBytes []byte) (*peer.Transaction, error) {
	tx := &peer.Transaction{}
	err := proto.Unmarshal(txBytes, tx)
	return tx, errors.Wrap(err, "error unmarshaling Transaction")

}

//get chaincodeactionpayload从字节获取chaincodeactionpayload
func GetChaincodeActionPayload(capBytes []byte) (*peer.ChaincodeActionPayload, error) {
	cap := &peer.ChaincodeActionPayload{}
	err := proto.Unmarshal(capBytes, cap)
	return cap, errors.Wrap(err, "error unmarshaling ChaincodeActionPayload")
}

//getchaincodeProposalPayload从字节获取chaincodeProposalPayload
func GetChaincodeProposalPayload(bytes []byte) (*peer.ChaincodeProposalPayload, error) {
	cpp := &peer.ChaincodeProposalPayload{}
	err := proto.Unmarshal(bytes, cpp)
	return cpp, errors.Wrap(err, "error unmarshaling ChaincodeProposalPayload")
}

//GetSignatureHeader从字节获取SignatureHeader
func GetSignatureHeader(bytes []byte) (*common.SignatureHeader, error) {
	sh := &common.SignatureHeader{}
	err := proto.Unmarshal(bytes, sh)
	return sh, errors.Wrap(err, "error unmarshaling SignatureHeader")
}

//CreateChaincodeProposal根据给定的输入创建建议。
//它返回建议和与建议关联的事务ID
func CreateChaincodeProposal(typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte) (*peer.Proposal, string, error) {
	return CreateChaincodeProposalWithTransient(typ, chainID, cis, creator, nil)
}

//CreateChaincodeProposalWithTransient从给定输入创建建议
//它返回建议和与建议关联的事务ID
func CreateChaincodeProposalWithTransient(typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte, transientMap map[string][]byte) (*peer.Proposal, string, error) {
//生成一个随机的nonce
	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return nil, "", err
	}

//计算TXID
	txid, err := ComputeTxID(nonce, creator)
	if err != nil {
		return nil, "", err
	}

	return CreateChaincodeProposalWithTxIDNonceAndTransient(txid, typ, chainID, cis, nonce, creator, transientMap)
}

//CreateChaincodeProposalWithTxidAndTransient根据给定的
//输入。它返回与
//建议
func CreateChaincodeProposalWithTxIDAndTransient(typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte, txid string, transientMap map[string][]byte) (*peer.Proposal, string, error) {
//生成一个随机的nonce
	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return nil, "", err
	}

//计算TxID，除非测试提供
	if txid == "" {
		txid, err = ComputeTxID(nonce, creator)
		if err != nil {
			return nil, "", err
		}
	}

	return CreateChaincodeProposalWithTxIDNonceAndTransient(txid, typ, chainID, cis, nonce, creator, transientMap)
}

//CreateChaincodeProposalWithTxidUncleandTransient从创建建议
//给定输入
func CreateChaincodeProposalWithTxIDNonceAndTransient(txid string, typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, nonce, creator []byte, transientMap map[string][]byte) (*peer.Proposal, string, error) {
	ccHdrExt := &peer.ChaincodeHeaderExtension{ChaincodeId: cis.ChaincodeSpec.ChaincodeId}
	ccHdrExtBytes, err := proto.Marshal(ccHdrExt)
	if err != nil {
		return nil, "", errors.Wrap(err, "error marshaling ChaincodeHeaderExtension")
	}

	cisBytes, err := proto.Marshal(cis)
	if err != nil {
		return nil, "", errors.Wrap(err, "error marshaling ChaincodeInvocationSpec")
	}

	ccPropPayload := &peer.ChaincodeProposalPayload{Input: cisBytes, TransientMap: transientMap}
	ccPropPayloadBytes, err := proto.Marshal(ccPropPayload)
	if err != nil {
		return nil, "", errors.Wrap(err, "error marshaling ChaincodeProposalPayload")
	}

//托多：epoch现在设置为零。一旦我们
//找一个更合适的机制来处理它。
	var epoch uint64

	timestamp := util.CreateUtcTimestamp()

	hdr := &common.Header{
		ChannelHeader: MarshalOrPanic(
			&common.ChannelHeader{
				Type:      int32(typ),
				TxId:      txid,
				Timestamp: timestamp,
				ChannelId: chainID,
				Extension: ccHdrExtBytes,
				Epoch:     epoch,
			},
		),
		SignatureHeader: MarshalOrPanic(
			&common.SignatureHeader{
				Nonce:   nonce,
				Creator: creator,
			},
		),
	}

	hdrBytes, err := proto.Marshal(hdr)
	if err != nil {
		return nil, "", err
	}

	prop := &peer.Proposal{
		Header:  hdrBytes,
		Payload: ccPropPayloadBytes,
	}
	return prop, txid, nil
}

//GetBytesProposalResponsePayLoad获取建议响应负载
func GetBytesProposalResponsePayload(hash []byte, response *peer.Response, result []byte, event []byte, ccid *peer.ChaincodeID) ([]byte, error) {
	cAct := &peer.ChaincodeAction{
		Events: event, Results: result,
		Response:    response,
		ChaincodeId: ccid,
	}
	cActBytes, err := proto.Marshal(cAct)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling ChaincodeAction")
	}

	prp := &peer.ProposalResponsePayload{
		Extension:    cActBytes,
		ProposalHash: hash,
	}
	prpBytes, err := proto.Marshal(prp)
	return prpBytes, errors.Wrap(err, "error marshaling ProposalResponsePayload")
}

//GetBytesChainCodeProposalPayload获取链码建议负载
func GetBytesChaincodeProposalPayload(cpp *peer.ChaincodeProposalPayload) ([]byte, error) {
	cppBytes, err := proto.Marshal(cpp)
	return cppBytes, errors.Wrap(err, "error marshaling ChaincodeProposalPayload")
}

//GetBytesResponse获取响应字节数
func GetBytesResponse(res *peer.Response) ([]byte, error) {
	resBytes, err := proto.Marshal(res)
	return resBytes, errors.Wrap(err, "error marshaling Response")
}

//GetByteschaincodeEvent获取chaincodeEvent的字节数
func GetBytesChaincodeEvent(event *peer.ChaincodeEvent) ([]byte, error) {
	eventBytes, err := proto.Marshal(event)
	return eventBytes, errors.Wrap(err, "error marshaling ChaincodeEvent")
}

//GetBytesChainCodeActionPayload从
//消息
func GetBytesChaincodeActionPayload(cap *peer.ChaincodeActionPayload) ([]byte, error) {
	capBytes, err := proto.Marshal(cap)
	return capBytes, errors.Wrap(err, "error marshaling ChaincodeActionPayload")
}

//GetBytesProposalResponse获取建议字节响应
func GetBytesProposalResponse(pr *peer.ProposalResponse) ([]byte, error) {
	respBytes, err := proto.Marshal(pr)
	return respBytes, errors.Wrap(err, "error marshaling ProposalResponse")
}

//GetBytesProposal返回建议消息的字节数
func GetBytesProposal(prop *peer.Proposal) ([]byte, error) {
	propBytes, err := proto.Marshal(prop)
	return propBytes, errors.Wrap(err, "error marshaling Proposal")
}

//GetBytesHeader从消息中获取头的字节数
func GetBytesHeader(hdr *common.Header) ([]byte, error) {
	bytes, err := proto.Marshal(hdr)
	return bytes, errors.Wrap(err, "error marshaling Header")
}

//GetBytesSignatureHeader从消息中获取SignatureHeader的字节数
func GetBytesSignatureHeader(hdr *common.SignatureHeader) ([]byte, error) {
	bytes, err := proto.Marshal(hdr)
	return bytes, errors.Wrap(err, "error marshaling SignatureHeader")
}

//GetBytesTransaction从消息中获取事务的字节数
func GetBytesTransaction(tx *peer.Transaction) ([]byte, error) {
	bytes, err := proto.Marshal(tx)
	return bytes, errors.Wrap(err, "error unmarshaling Transaction")
}

//get bytes payload从消息中获取有效负载的字节数
func GetBytesPayload(payl *common.Payload) ([]byte, error) {
	bytes, err := proto.Marshal(payl)
	return bytes, errors.Wrap(err, "error marshaling Payload")
}

//GetBytesInvelope从消息中获取信封的字节数
func GetBytesEnvelope(env *common.Envelope) ([]byte, error) {
	bytes, err := proto.Marshal(env)
	return bytes, errors.Wrap(err, "error marshaling Envelope")
}

//GetActionFromEnvelope从
//序列化信封
//TODO:根据FAB-11831修复函数名
func GetActionFromEnvelope(envBytes []byte) (*peer.ChaincodeAction, error) {
	env, err := GetEnvelopeFromBlock(envBytes)
	if err != nil {
		return nil, err
	}
	return GetActionFromEnvelopeMsg(env)
}

func GetActionFromEnvelopeMsg(env *common.Envelope) (*peer.ChaincodeAction, error) {
	payl, err := GetPayload(env)
	if err != nil {
		return nil, err
	}

	tx, err := GetTransaction(payl.Data)
	if err != nil {
		return nil, err
	}

	if len(tx.Actions) == 0 {
		return nil, errors.New("at least one TransactionAction required")
	}

	_, respPayload, err := GetPayloads(tx.Actions[0])
	return respPayload, err
}

//CreateProposalFromCisandXid返回给定序列化标识的建议
//以及一个链码调用pec
func CreateProposalFromCISAndTxid(txid string, typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte) (*peer.Proposal, string, error) {
	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return nil, "", err
	}
	return CreateChaincodeProposalWithTxIDNonceAndTransient(txid, typ, chainID, cis, nonce, creator, nil)
}

//CreateProposalFromCIS返回给定序列化标识和
//链码调用spec
func CreateProposalFromCIS(typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte) (*peer.Proposal, string, error) {
	return CreateChaincodeProposal(typ, chainID, cis, creator)
}

//creategetchaincodesproposal返回给定的getchaincodes建议
//序列化标识
func CreateGetChaincodesProposal(chainID string, creator []byte) (*peer.Proposal, string, error) {
	ccinp := &peer.ChaincodeInput{Args: [][]byte{[]byte("getchaincodes")}}
	lsccSpec := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			Type:        peer.ChaincodeSpec_GOLANG,
			ChaincodeId: &peer.ChaincodeID{Name: "lscc"},
			Input:       ccinp,
		},
	}
	return CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, chainID, lsccSpec, creator)
}

//CreateGetInstalledChaincodesProposal返回GetInstalledChaincodes
//提供了序列化标识的建议
func CreateGetInstalledChaincodesProposal(creator []byte) (*peer.Proposal, string, error) {
	ccinp := &peer.ChaincodeInput{Args: [][]byte{[]byte("getinstalledchaincodes")}}
	lsccSpec := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			Type:        peer.ChaincodeSpec_GOLANG,
			ChaincodeId: &peer.ChaincodeID{Name: "lscc"},
			Input:       ccinp,
		},
	}
	return CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, "", lsccSpec, creator)
}

//CreateSinstallProposalFromCDS返回一个给定序列化的安装建议
//标识和链代码部署
func CreateInstallProposalFromCDS(ccpack proto.Message, creator []byte) (*peer.Proposal, string, error) {
	return createProposalFromCDS("", ccpack, creator, "install")
}

//createdDeployProposalFromCDS返回一个给定序列化的部署建议
//标识和链代码部署
func CreateDeployProposalFromCDS(
	chainID string,
	cds *peer.ChaincodeDeploymentSpec,
	creator []byte,
	policy []byte,
	escc []byte,
	vscc []byte,
	collectionConfig []byte) (*peer.Proposal, string, error) {
	if collectionConfig == nil {
		return createProposalFromCDS(chainID, cds, creator, "deploy", policy, escc, vscc)
	}
	return createProposalFromCDS(chainID, cds, creator, "deploy", policy, escc, vscc, collectionConfig)
}

//CreateUpgradeProposalFromCDS返回给定序列化的升级建议
//标识和链代码部署
func CreateUpgradeProposalFromCDS(
	chainID string,
	cds *peer.ChaincodeDeploymentSpec,
	creator []byte,
	policy []byte,
	escc []byte,
	vscc []byte,
	collectionConfig []byte) (*peer.Proposal, string, error) {
	if collectionConfig == nil {
		return createProposalFromCDS(chainID, cds, creator, "upgrade", policy, escc, vscc)
	}
	return createProposalFromCDS(chainID, cds, creator, "upgrade", policy, escc, vscc, collectionConfig)
}

//CreateProposalFromCDS返回给定的部署或升级建议
//序列化标识和chaincodedeploymentspec
func createProposalFromCDS(chainID string, msg proto.Message, creator []byte, propType string, args ...[]byte) (*peer.Proposal, string, error) {
//在新模式下，CD将为零，“部署”和“升级”将被实例化。
	var ccinp *peer.ChaincodeInput
	var b []byte
	var err error
	if msg != nil {
		b, err = proto.Marshal(msg)
		if err != nil {
			return nil, "", err
		}
	}
	switch propType {
	case "deploy":
		fallthrough
	case "upgrade":
		cds, ok := msg.(*peer.ChaincodeDeploymentSpec)
		if !ok || cds == nil {
			return nil, "", errors.New("invalid message for creating lifecycle chaincode proposal")
		}
		Args := [][]byte{[]byte(propType), []byte(chainID), b}
		Args = append(Args, args...)

		ccinp = &peer.ChaincodeInput{Args: Args}
	case "install":
		ccinp = &peer.ChaincodeInput{Args: [][]byte{[]byte(propType), b}}
	}

//将部署包装到lscc的调用规范中…
	lsccSpec := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			Type:        peer.ChaincodeSpec_GOLANG,
			ChaincodeId: &peer.ChaincodeID{Name: "lscc"},
			Input:       ccinp,
		},
	}

//…并得到它的建议
	return CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, chainID, lsccSpec, creator)
}

//computetxid将txid计算为哈希
//超越了nonce和creator的连接。
func ComputeTxID(nonce, creator []byte) (string, error) {
//TODO:获取要从中使用的哈希函数
//信道配置
	digest, err := factory.GetDefault().Hash(
		append(nonce, creator...),
		&bccsp.SHA256Opts{})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest), nil
}

//checktxid检查txid是否等于计算的哈希值
//超越了nonce和creator的连接。
func CheckTxID(txid string, nonce, creator []byte) error {
	computedTxID, err := ComputeTxID(nonce, creator)
	if err != nil {
		return errors.WithMessage(err, "error computing target txid")
	}

	if txid != computedTxID {
		return errors.Errorf("invalid txid. got [%s], expected [%s]", txid, computedTxID)
	}

	return nil
}

//计算提案绑定计算提案的绑定
func ComputeProposalBinding(proposal *peer.Proposal) ([]byte, error) {
	if proposal == nil {
		return nil, errors.New("proposal is nil")
	}
	if len(proposal.Header) == 0 {
		return nil, errors.New("proposal's header is nil")
	}

	h, err := GetHeader(proposal.Header)
	if err != nil {
		return nil, err
	}

	chdr, err := UnmarshalChannelHeader(h.ChannelHeader)
	if err != nil {
		return nil, err
	}
	shdr, err := GetSignatureHeader(h.SignatureHeader)
	if err != nil {
		return nil, err
	}

	return computeProposalBindingInternal(shdr.Nonce, shdr.Creator, chdr.Epoch)
}

func computeProposalBindingInternal(nonce, creator []byte, epoch uint64) ([]byte, error) {
	epochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBytes, epoch)

//TODO:向Genesis块添加用于
//结合计算
	return factory.GetDefault().Hash(
		append(append(nonce, creator...), epochBytes...),
		&bccsp.SHA256Opts{})
}
