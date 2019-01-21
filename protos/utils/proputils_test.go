
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


package utils_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func createCIS() *pb.ChaincodeInvocationSpec {
	return &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        pb.ChaincodeSpec_GOLANG,
			ChaincodeId: &pb.ChaincodeID{Name: "chaincode_name"},
			Input:       &pb.ChaincodeInput{Args: [][]byte{[]byte("arg1"), []byte("arg2")}}}}
}

func TestNilProposal(t *testing.T) {
//将nil传递给接受peer.proposal的所有函数
	_, err := utils.GetChaincodeInvocationSpec(nil)
	assert.Error(t, err, "Expected error with nil proposal")
	_, _, err = utils.GetChaincodeProposalContext(nil)
	assert.Error(t, err, "Expected error with nil proposal")
	_, err = utils.GetNonce(nil)
	assert.Error(t, err, "Expected error with nil proposal")
	_, err = utils.GetBytesProposal(nil)
	assert.Error(t, err, "Expected error with nil proposal")
	_, err = utils.ComputeProposalBinding(nil)
	assert.Error(t, err, "Expected error with nil proposal")
}

func TestBadProposalHeaders(t *testing.T) {
//注意：有很多重复的提案验证代码
//在将来应该重构的多个函数中。
//现在，只需合并测试用例

//空标题
	prop := &pb.Proposal{
		Header: []byte{},
	}
	_, _, err := utils.GetChaincodeProposalContext(prop)
	assert.Error(t, err, "Expected error with empty proposal header")
	_, err = utils.ComputeProposalBinding(prop)
	assert.Error(t, err, "Expected error with empty proposal header")

//空载
	prop = &pb.Proposal{
		Header: []byte("header"),
	}
	_, _, err = utils.GetChaincodeProposalContext(prop)
	assert.Error(t, err, "Expected error with empty proposal payload")

//格式不正确的建议标题
	prop = &pb.Proposal{
		Header:  []byte("bad header"),
		Payload: []byte("payload"),
	}
	_, err = utils.GetHeader(prop.Header)
	assert.Error(t, err, "Expected error with malformed proposal header")
	_, err = utils.GetChaincodeInvocationSpec(prop)
	assert.Error(t, err, "Expected error with malformed proposal header")
	_, _, err = utils.GetChaincodeProposalContext(prop)
	assert.Error(t, err, "Expected error with malformed proposal header")
	_, err = utils.GetNonce(prop)
	assert.Error(t, err, "Expected error with malformed proposal header")
	_, err = utils.ComputeProposalBinding(prop)
	assert.Error(t, err, "Expected error with malformed proposal header")

//签名头格式不正确
	chdr, _ := proto.Marshal(&common.ChannelHeader{
		Type: int32(common.HeaderType_ENDORSER_TRANSACTION),
	})
	hdr := &common.Header{
		ChannelHeader:   chdr,
		SignatureHeader: []byte("bad signature header"),
	}
	_, err = utils.GetSignatureHeader(hdr.SignatureHeader)
	assert.Error(t, err, "Expected error with malformed signature header")
	hdrBytes, _ := proto.Marshal(hdr)
	prop.Header = hdrBytes
	_, err = utils.GetChaincodeInvocationSpec(prop)
	assert.Error(t, err, "Expected error with malformed signature header")
	_, _, err = utils.GetChaincodeProposalContext(prop)
	assert.Error(t, err, "Expected error with malformed signature header")
	_, err = utils.GetNonce(prop)
	assert.Error(t, err, "Expected error with malformed signature header")
	_, err = utils.ComputeProposalBinding(prop)
	assert.Error(t, err, "Expected error with malformed signature header")

//错误的频道标题类型
	chdr, _ = proto.Marshal(&common.ChannelHeader{
		Type: int32(common.HeaderType_DELIVER_SEEK_INFO),
	})
	hdr.ChannelHeader = chdr
	hdrBytes, _ = proto.Marshal(hdr)
	prop.Header = hdrBytes
	_, _, err = utils.GetChaincodeProposalContext(prop)
	assert.Error(t, err, "Expected error with wrong header type")
	assert.Contains(t, err.Error(), "invalid proposal: invalid channel header type")
	_, err = utils.GetNonce(prop)
	assert.Error(t, err, "Expected error with wrong header type")

//通道头格式错误
	hdr.ChannelHeader = []byte("bad channel header")
	hdrBytes, _ = proto.Marshal(hdr)
	prop.Header = hdrBytes
	_, _, err = utils.GetChaincodeProposalContext(prop)
	assert.Error(t, err, "Expected error with malformed channel header")
	_, err = utils.GetNonce(prop)
	assert.Error(t, err, "Expected error with malformed channel header")
	_, err = utils.GetChaincodeHeaderExtension(hdr)
	assert.Error(t, err, "Expected error with malformed channel header")
	_, err = utils.ComputeProposalBinding(prop)
	assert.Error(t, err, "Expected error with malformed channel header")

}

func TestGetNonce(t *testing.T) {
	chdr, _ := proto.Marshal(&common.ChannelHeader{
		Type: int32(common.HeaderType_ENDORSER_TRANSACTION),
	})
	hdr, _ := proto.Marshal(&common.Header{
		ChannelHeader:   chdr,
		SignatureHeader: []byte{},
	})
	prop := &pb.Proposal{
		Header: hdr,
	}
	_, err := utils.GetNonce(prop)
	assert.Error(t, err, "Expected error with nil signature header")

	shdr, _ := proto.Marshal(&common.SignatureHeader{
		Nonce: []byte("nonce"),
	})
	hdr, _ = proto.Marshal(&common.Header{
		ChannelHeader:   chdr,
		SignatureHeader: shdr,
	})
	prop = &pb.Proposal{
		Header: hdr,
	}
	nonce, err := utils.GetNonce(prop)
	assert.NoError(t, err, "Unexpected error getting nonce")
	assert.Equal(t, "nonce", string(nonce), "Failed to return the expected nonce")

}

func TestGetChaincodeDeploymentSpec(t *testing.T) {
	pr := platforms.NewRegistry(&golang.Platform{})

	_, err := utils.GetChaincodeDeploymentSpec([]byte("bad spec"), pr)
	assert.Error(t, err, "Expected error with malformed spec")

	cds, _ := proto.Marshal(&pb.ChaincodeDeploymentSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type: pb.ChaincodeSpec_GOLANG,
		},
	})
	_, err = utils.GetChaincodeDeploymentSpec(cds, pr)
	assert.NoError(t, err, "Unexpected error getting deployment spec")

	cds, _ = proto.Marshal(&pb.ChaincodeDeploymentSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type: pb.ChaincodeSpec_UNDEFINED,
		},
	})
	_, err = utils.GetChaincodeDeploymentSpec(cds, pr)
	assert.Error(t, err, "Expected error with invalid spec type")

}

func TestCDSProposals(t *testing.T) {
	var prop *pb.Proposal
	var err error
	var txid string
	creator := []byte("creator")
	cds := &pb.ChaincodeDeploymentSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type: pb.ChaincodeSpec_GOLANG,
		},
	}
	policy := []byte("policy")
	escc := []byte("escc")
	vscc := []byte("vscc")
	chainID := "testchainid"

//安装
	prop, txid, err = utils.CreateInstallProposalFromCDS(cds, creator)
	assert.NotNil(t, prop, "Install proposal should not be nil")
	assert.NoError(t, err, "Unexpected error creating install proposal")
	assert.NotEqual(t, "", txid, "txid should not be empty")

//部署
	prop, txid, err = utils.CreateDeployProposalFromCDS(chainID, cds, creator, policy, escc, vscc, nil)
	assert.NotNil(t, prop, "Deploy proposal should not be nil")
	assert.NoError(t, err, "Unexpected error creating deploy proposal")
	assert.NotEqual(t, "", txid, "txid should not be empty")

//升级
	prop, txid, err = utils.CreateUpgradeProposalFromCDS(chainID, cds, creator, policy, escc, vscc, nil)
	assert.NotNil(t, prop, "Upgrade proposal should not be nil")
	assert.NoError(t, err, "Unexpected error creating upgrade proposal")
	assert.NotEqual(t, "", txid, "txid should not be empty")

}

func TestComputeProposalBinding(t *testing.T) {
	expectedDigestHex := "5093dd4f4277e964da8f4afbde0a9674d17f2a6a5961f0670fc21ae9b67f2983"
	expectedDigest, _ := hex.DecodeString(expectedDigestHex)
	chdr, _ := proto.Marshal(&common.ChannelHeader{
		Epoch: uint64(10),
	})
	shdr, _ := proto.Marshal(&common.SignatureHeader{
		Nonce:   []byte("nonce"),
		Creator: []byte("creator"),
	})
	hdr, _ := proto.Marshal(&common.Header{
		ChannelHeader:   chdr,
		SignatureHeader: shdr,
	})
	prop := &pb.Proposal{
		Header: hdr,
	}
	binding, _ := utils.ComputeProposalBinding(prop)
	assert.Equal(t, expectedDigest, binding, "Binding does not match expected digest")
}

func TestProposal(t *testing.T) {
//从chaincodeinvocationspec创建建议
	prop, _, err := utils.CreateChaincodeProposalWithTransient(
		common.HeaderType_ENDORSER_TRANSACTION,
		util.GetTestChainID(), createCIS(),
		[]byte("creator"),
		map[string][]byte{"certx": []byte("transient")})
	if err != nil {
		t.Fatalf("Could not create chaincode proposal, err %s\n", err)
		return
	}

//序列化建议
	pBytes, err := utils.GetBytesProposal(prop)
	if err != nil {
		t.Fatalf("Could not serialize the chaincode proposal, err %s\n", err)
		return
	}

//反序列化它并期望它是相同的
	propBack, err := utils.GetProposal(pBytes)
	if err != nil {
		t.Fatalf("Could not deserialize the chaincode proposal, err %s\n", err)
		return
	}
	if !proto.Equal(prop, propBack) {
		t.Fatalf("Proposal and deserialized proposals don't match\n")
		return
	}

//回到头上
	hdr, err := utils.GetHeader(prop.Header)
	if err != nil {
		t.Fatalf("Could not extract the header from the proposal, err %s\n", err)
	}

	hdrBytes, err := utils.GetBytesHeader(hdr)
	if err != nil {
		t.Fatalf("Could not marshal the header, err %s\n", err)
	}

	hdr, err = utils.GetHeader(hdrBytes)
	if err != nil {
		t.Fatalf("Could not unmarshal the header, err %s\n", err)
	}

	chdr, err := utils.UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		t.Fatalf("Could not unmarshal channel header, err %s", err)
	}

	shdr, err := utils.GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		t.Fatalf("Could not unmarshal signature header, err %s", err)
	}

	_, err = utils.GetBytesSignatureHeader(shdr)
	if err != nil {
		t.Fatalf("Could not marshal signature header, err %s", err)
	}

//收割台的健全性检查
	if chdr.Type != int32(common.HeaderType_ENDORSER_TRANSACTION) ||
		shdr.Nonce == nil ||
		string(shdr.Creator) != "creator" {
		t.Fatalf("Invalid header after unmarshalling\n")
		return
	}

//取回收割台延伸部分
	hdrExt, err := utils.GetChaincodeHeaderExtension(hdr)
	if err != nil {
		t.Fatalf("Could not extract the header extensions from the proposal, err %s\n", err)
		return
	}

//收割台延伸部分的健全性检查
	if string(hdrExt.ChaincodeId.Name) != "chaincode_name" {
		t.Fatalf("Invalid header extension after unmarshalling\n")
		return
	}

//取回chaincodeinvocationspec
	cis, err := utils.GetChaincodeInvocationSpec(prop)
	if err != nil {
		t.Fatalf("Could not extract chaincode invocation spec from header, err %s\n", err)
		return
	}

//独联体的健全性检查
	if cis.ChaincodeSpec.Type != pb.ChaincodeSpec_GOLANG ||
		cis.ChaincodeSpec.ChaincodeId.Name != "chaincode_name" ||
		len(cis.ChaincodeSpec.Input.Args) != 2 ||
		string(cis.ChaincodeSpec.Input.Args[0]) != "arg1" ||
		string(cis.ChaincodeSpec.Input.Args[1]) != "arg2" {
		t.Fatalf("Invalid chaincode invocation spec after unmarshalling\n")
		return
	}

	creator, transient, err := utils.GetChaincodeProposalContext(prop)
	if err != nil {
		t.Fatalf("Failed getting chaincode proposal context [%s]", err)
	}
	if string(creator) != "creator" {
		t.Fatalf("Failed checking Creator field. Invalid value, expectext 'creator', got [%s]", string(creator))
		return
	}
	value, ok := transient["certx"]
	if !ok || string(value) != "transient" {
		t.Fatalf("Failed checking Transient field. Invalid value, expectext 'transient', got [%s]", string(value))
		return
	}
}

func TestProposalWithTxID(t *testing.T) {
//从chaincodeinvocationspec创建建议
	prop, txid, err := utils.CreateChaincodeProposalWithTxIDAndTransient(
		common.HeaderType_ENDORSER_TRANSACTION,
		util.GetTestChainID(),
		createCIS(),
		[]byte("creator"),
		"testtx",
		map[string][]byte{"certx": []byte("transient")},
	)
	assert.Nil(t, err)
	assert.NotNil(t, prop)
	assert.Equal(t, txid, "testtx")

	prop, txid, err = utils.CreateChaincodeProposalWithTxIDAndTransient(
		common.HeaderType_ENDORSER_TRANSACTION,
		util.GetTestChainID(),
		createCIS(),
		[]byte("creator"),
		"",
		map[string][]byte{"certx": []byte("transient")},
	)
	assert.Nil(t, err)
	assert.NotNil(t, prop)
	assert.NotEmpty(t, txid)
}

func TestProposalResponse(t *testing.T) {
	events := &pb.ChaincodeEvent{
		ChaincodeId: "ccid",
		EventName:   "EventName",
		Payload:     []byte("EventPayload"),
		TxId:        "TxID"}
	ccid := &pb.ChaincodeID{
		Name:    "ccid",
		Version: "v1",
	}

	pHashBytes := []byte("proposal_hash")
	pResponse := &pb.Response{Status: 200}
	results := []byte("results")
	eventBytes, err := utils.GetBytesChaincodeEvent(events)
	if err != nil {
		t.Fatalf("Failure while marshalling the ProposalResponsePayload")
		return
	}

//获取响应的字节数
	pResponseBytes, err := utils.GetBytesResponse(pResponse)
	if err != nil {
		t.Fatalf("Failure while marshalling the Response")
		return
	}

//从字节获取响应
	_, err = utils.GetResponse(pResponseBytes)
	if err != nil {
		t.Fatalf("Failure while unmarshalling the Response")
		return
	}

//获取ProposalResponsePayLoad的字节数
	prpBytes, err := utils.GetBytesProposalResponsePayload(pHashBytes, pResponse, results, eventBytes, ccid)
	if err != nil {
		t.Fatalf("Failure while marshalling the ProposalResponsePayload")
		return
	}

//获取ProposalResponsePayLoad消息
	prp, err := utils.GetProposalResponsePayload(prpBytes)
	if err != nil {
		t.Fatalf("Failure while unmarshalling the ProposalResponsePayload")
		return
	}

//获取chaincodeaction消息
	act, err := utils.GetChaincodeAction(prp.Extension)
	if err != nil {
		t.Fatalf("Failure while unmarshalling the ChaincodeAction")
		return
	}

//行动的健全性检查
	if string(act.Results) != "results" {
		t.Fatalf("Invalid actions after unmarshalling")
		return
	}

	event, err := utils.GetChaincodeEvents(act.Events)
	if err != nil {
		t.Fatalf("Failure while unmarshalling the ChainCodeEvents")
		return
	}

//活动的健全性检查
	if string(event.ChaincodeId) != "ccid" {
		t.Fatalf("Invalid actions after unmarshalling")
		return
	}

	pr := &pb.ProposalResponse{
		Payload:     prpBytes,
		Endorsement: &pb.Endorsement{Endorser: []byte("endorser"), Signature: []byte("signature")},
Version:     1, //TODO:选择正确的版本号
		Response:    &pb.Response{Status: 200, Message: "OK"}}

//创建建议响应
	prBytes, err := utils.GetBytesProposalResponse(pr)
	if err != nil {
		t.Fatalf("Failure while marshalling the ProposalResponse")
		return
	}

//获取提案响应消息
	prBack, err := utils.GetProposalResponse(prBytes)
	if err != nil {
		t.Fatalf("Failure while unmarshalling the ProposalResponse")
		return
	}

//公共关系健康检查
	if prBack.Response.Status != 200 ||
		string(prBack.Endorsement.Signature) != "signature" ||
		string(prBack.Endorsement.Endorser) != "endorser" ||
		bytes.Compare(prBack.Payload, prpBytes) != 0 {
		t.Fatalf("Invalid ProposalResponse after unmarshalling")
		return
	}
}

func TestEnvelope(t *testing.T) {
//从chaincodeinvocationspec创建建议
	prop, _, err := utils.CreateChaincodeProposal(common.HeaderType_ENDORSER_TRANSACTION, util.GetTestChainID(), createCIS(), signerSerialized)
	if err != nil {
		t.Fatalf("Could not create chaincode proposal, err %s\n", err)
		return
	}

	response := &pb.Response{Status: 200, Payload: []byte("payload")}
	result := []byte("res")
	ccid := &pb.ChaincodeID{Name: "foo", Version: "v1"}

	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response, result, nil, ccid, nil, signer)
	if err != nil {
		t.Fatalf("Could not create proposal response, err %s\n", err)
		return
	}

	tx, err := utils.CreateSignedTx(prop, signer, presp)
	if err != nil {
		t.Fatalf("Could not create signed tx, err %s\n", err)
		return
	}

	envBytes, err := utils.GetBytesEnvelope(tx)
	if err != nil {
		t.Fatalf("Could not marshal envelope, err %s\n", err)
		return
	}

	tx, err = utils.GetEnvelopeFromBlock(envBytes)
	if err != nil {
		t.Fatalf("Could not unmarshal envelope, err %s\n", err)
		return
	}

	act2, err := utils.GetActionFromEnvelope(envBytes)
	if err != nil {
		t.Fatalf("Could not extract actions from envelop, err %s\n", err)
		return
	}

	if act2.Response.Status != response.Status {
		t.Fatalf("response staus don't match")
		return
	}
	if bytes.Compare(act2.Response.Payload, response.Payload) != 0 {
		t.Fatalf("response payload don't match")
		return
	}

	if bytes.Compare(act2.Results, result) != 0 {
		t.Fatalf("results don't match")
		return
	}

	txpayl, err := utils.GetPayload(tx)
	if err != nil {
		t.Fatalf("Could not unmarshal payload, err %s\n", err)
		return
	}

	tx2, err := utils.GetTransaction(txpayl.Data)
	if err != nil {
		t.Fatalf("Could not unmarshal Transaction, err %s\n", err)
		return
	}

	sh, err := utils.GetSignatureHeader(tx2.Actions[0].Header)
	if err != nil {
		t.Fatalf("Could not unmarshal SignatureHeader, err %s\n", err)
		return
	}

	if bytes.Compare(sh.Creator, signerSerialized) != 0 {
		t.Fatalf("creator does not match")
		return
	}

	cap, err := utils.GetChaincodeActionPayload(tx2.Actions[0].Payload)
	if err != nil {
		t.Fatalf("Could not unmarshal ChaincodeActionPayload, err %s\n", err)
		return
	}
	assert.NotNil(t, cap)

	prp, err := utils.GetProposalResponsePayload(cap.Action.ProposalResponsePayload)
	if err != nil {
		t.Fatalf("Could not unmarshal ProposalResponsePayload, err %s\n", err)
		return
	}

	ca, err := utils.GetChaincodeAction(prp.Extension)
	if err != nil {
		t.Fatalf("Could not unmarshal ChaincodeAction, err %s\n", err)
		return
	}

	if ca.Response.Status != response.Status {
		t.Fatalf("response staus don't match")
		return
	}
	if bytes.Compare(ca.Response.Payload, response.Payload) != 0 {
		t.Fatalf("response payload don't match")
		return
	}

	if bytes.Compare(ca.Results, result) != 0 {
		t.Fatalf("results don't match")
		return
	}
}

func TestProposalTxID(t *testing.T) {
	nonce := []byte{1}
	creator := []byte{2}

	txid, err := utils.ComputeTxID(nonce, creator)
	assert.NotEmpty(t, txid, "TxID cannot be empty.")
	assert.NoError(t, err, "Failed computing txID")
	assert.Nil(t, utils.CheckTxID(txid, nonce, creator))
	assert.Error(t, utils.CheckTxID("", nonce, creator))

	txid, err = utils.ComputeTxID(nil, nil)
	assert.NotEmpty(t, txid, "TxID cannot be empty.")
	assert.NoError(t, err, "Failed computing txID")
}

func TestComputeProposalTxID(t *testing.T) {
	txid, err := utils.ComputeTxID([]byte{1}, []byte{1})
	assert.NoError(t, err, "Failed computing TxID")

//计算computetxid计算的函数，
//即base64（sha256（nonce creator）
	hf := sha256.New()
	hf.Write([]byte{1})
	hf.Write([]byte{1})
	hashOut := hf.Sum(nil)
	txid2 := hex.EncodeToString(hashOut)

	t.Logf("% x\n", hashOut)
	t.Logf("% s\n", txid)
	t.Logf("% s\n", txid2)

	assert.Equal(t, txid, txid2)
}

var signer msp.SigningIdentity
var signerSerialized []byte

func TestMain(m *testing.M) {
//设置MSP管理器，以便我们可以签名/验证
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		os.Exit(-1)
		fmt.Printf("Could not initialize msp")
		return
	}
	signer, err = mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		os.Exit(-1)
		fmt.Printf("Could not get signer")
		return
	}

	signerSerialized, err = signer.Serialize()
	if err != nil {
		os.Exit(-1)
		fmt.Printf("Could not serialize identity")
		return
	}

	os.Exit(m.Run())
}
