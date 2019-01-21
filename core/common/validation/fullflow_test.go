
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
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/mocks/config"
	mmsp "github.com/hyperledger/fabric/common/mocks/msp"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func getProposal(channel string) (*peer.Proposal, error) {
	cis := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: getChaincodeID(),
			Type:        peer.ChaincodeSpec_GOLANG}}

	proposal, _, err := utils.CreateProposalFromCIS(common.HeaderType_ENDORSER_TRANSACTION, channel, cis, signerSerialized)
	return proposal, err
}

func getChaincodeID() *peer.ChaincodeID {
	return &peer.ChaincodeID{Name: "foo", Version: "v1"}
}

func createSignedTxTwoActions(proposal *peer.Proposal, signer msp.SigningIdentity, resps ...*peer.ProposalResponse) (*common.Envelope, error) {
	if len(resps) == 0 {
		return nil, fmt.Errorf("At least one proposal response is necessary")
	}

//the original header
	hdr, err := utils.GetHeader(proposal.Header)
	if err != nil {
		return nil, fmt.Errorf("Could not unmarshal the proposal header")
	}

//原始有效载荷
	pPayl, err := utils.GetChaincodeProposalPayload(proposal.Payload)
	if err != nil {
		return nil, fmt.Errorf("Could not unmarshal the proposal payload")
	}

//填写背书
	endorsements := make([]*peer.Endorsement, len(resps))
	for n, r := range resps {
		endorsements[n] = r.Endorsement
	}

//create ChaincodeEndorsedAction
	cea := &peer.ChaincodeEndorsedAction{ProposalResponsePayload: resps[0].Payload, Endorsements: endorsements}

//获取将转到事务的建议负载的字节
	propPayloadBytes, err := utils.GetBytesProposalPayloadForTx(pPayl, nil)
	if err != nil {
		return nil, err
	}

//序列化chaincode操作负载
	cap := &peer.ChaincodeActionPayload{ChaincodeProposalPayload: propPayloadBytes, Action: cea}
	capBytes, err := utils.GetBytesChaincodeActionPayload(cap)
	if err != nil {
		return nil, err
	}

//创建交易记录
	taa := &peer.TransactionAction{Header: hdr.SignatureHeader, Payload: capBytes}
	taas := make([]*peer.TransactionAction, 2)
	taas[0] = taa
	taas[1] = taa
	tx := &peer.Transaction{Actions: taas}

//序列化Tx
	txBytes, err := utils.GetBytesTransaction(tx)
	if err != nil {
		return nil, err
	}

//创建有效载荷
	payl := &common.Payload{Header: hdr, Data: txBytes}
	paylBytes, err := utils.GetBytesPayload(payl)
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

func TestGoodPath(t *testing.T) {
//得到一个玩具建议
	prop, err := getProposal(util.GetTestChainID())
	if err != nil {
		t.Fatalf("getProposal failed, err %s", err)
		return
	}

//签字
	sProp, err := utils.GetSignedProposal(prop, signer)
	if err != nil {
		t.Fatalf("GetSignedProposal failed, err %s", err)
		return
	}

//验证它
	_, _, _, err = ValidateProposalMessage(sProp)
	if err != nil {
		t.Fatalf("ValidateProposalMessage failed, err %s", err)
		return
	}

	response := &peer.Response{Status: 200}
	simRes := []byte("simulation_result")

//签署以获得提案响应
	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response, simRes, nil, getChaincodeID(), nil, signer)
	if err != nil {
		t.Fatalf("CreateProposalResponse failed, err %s", err)
		return
	}

//根据该提议和背书进行交易
	tx, err := utils.CreateSignedTx(prop, signer, presp)
	if err != nil {
		t.Fatalf("CreateSignedTx failed, err %s", err)
		return
	}

//验证事务
	payl, txResult := ValidateTransaction(tx, &config.MockApplicationCapabilities{})
	if txResult != peer.TxValidationCode_VALID {
		t.Fatalf("ValidateTransaction failed, err %s", err)
		return
	}

	txx, err := utils.GetTransaction(payl.Data)
	if err != nil {
		t.Fatalf("GetTransaction failed, err %s", err)
		return
	}

	act := txx.Actions

//期望一个动作
	if len(act) != 1 {
		t.Fatalf("Ivalid number of TransactionAction, expected 1, got %d", len(act))
		return
	}

//获取操作的有效负载
	_, simResBack, err := utils.GetPayloads(act[0])
	if err != nil {
		t.Fatalf("GetPayloads failed, err %s", err)
		return
	}

//将其与原始动作进行比较，并期望其相等
	if string(simRes) != string(simResBack.Results) {
		t.Fatal("Simulation results are different")
		return
	}
}

func TestTXWithTwoActionsRejected(t *testing.T) {
//得到一个玩具建议
	prop, err := getProposal(util.GetTestChainID())
	if err != nil {
		t.Fatalf("getProposal failed, err %s", err)
		return
	}

	response := &peer.Response{Status: 200}
	simRes := []byte("simulation_result")

//签署以获得提案响应
	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response, simRes, nil, &peer.ChaincodeID{Name: "somename", Version: "someversion"}, nil, signer)
	if err != nil {
		t.Fatalf("CreateProposalResponse failed, err %s", err)
		return
	}

//根据该提议和背书进行交易
	tx, err := createSignedTxTwoActions(prop, signer, presp)
	if err != nil {
		t.Fatalf("CreateSignedTx failed, err %s", err)
		return
	}

//验证事务
	_, txResult := ValidateTransaction(tx, &config.MockApplicationCapabilities{})
	if txResult == peer.TxValidationCode_VALID {
		t.Fatalf("ValidateTransaction should have failed")
		return
	}
}

func TestBadProp(t *testing.T) {
//得到一个玩具建议
	prop, err := getProposal(util.GetTestChainID())
	if err != nil {
		t.Fatalf("getProposal failed, err %s", err)
		return
	}

//签字
	sProp, err := utils.GetSignedProposal(prop, signer)
	if err != nil {
		t.Fatalf("GetSignedProposal failed, err %s", err)
		return
	}

//弄乱了签名
	sigOrig := sProp.Signature
	for i := 0; i < len(sigOrig); i++ {
		sigCopy := make([]byte, len(sigOrig))
		copy(sigCopy, sigOrig)
		sigCopy[i] = byte(int(sigCopy[i]+1) % 255)
//验证它-它应该失败
		_, _, _, err = ValidateProposalMessage(&peer.SignedProposal{ProposalBytes: sProp.ProposalBytes, Signature: sigCopy})
		if err == nil {
			t.Fatal("ValidateProposalMessage should have failed")
			return
		}
	}

//再次签字
	sProp, err = utils.GetSignedProposal(prop, signer)
	if err != nil {
		t.Fatalf("GetSignedProposal failed, err %s", err)
		return
	}

//干扰信息
	pbytesOrig := sProp.ProposalBytes
	for i := 0; i < len(pbytesOrig); i++ {
		pbytesCopy := make([]byte, len(pbytesOrig))
		copy(pbytesCopy, pbytesOrig)
		pbytesCopy[i] = byte(int(pbytesCopy[i]+1) % 255)
//验证它-它应该失败
		_, _, _, err = ValidateProposalMessage(&peer.SignedProposal{ProposalBytes: pbytesCopy, Signature: sProp.Signature})
		if err == nil {
			t.Fatal("ValidateProposalMessage should have failed")
			return
		}
	}

//get a bad signing identity
	badSigner, err := mmsp.NewNoopMsp().GetDefaultSigningIdentity()
	if err != nil {
		t.Fatal("Couldn't get noop signer")
		return
	}

//用不好的签名人再次签名
	sProp, err = utils.GetSignedProposal(prop, badSigner)
	if err != nil {
		t.Fatalf("GetSignedProposal failed, err %s", err)
		return
	}

//验证它-它应该失败
	_, _, _, err = ValidateProposalMessage(sProp)
	if err == nil {
		t.Fatal("ValidateProposalMessage should have failed")
		return
	}
}

func corrupt(bytes []byte) {
	rand.Seed(time.Now().UnixNano())
	bytes[rand.Intn(len(bytes))]--
}

func TestBadTx(t *testing.T) {
//得到一个玩具建议
	prop, err := getProposal(util.GetTestChainID())
	if err != nil {
		t.Fatalf("getProposal failed, err %s", err)
		return
	}

	response := &peer.Response{Status: 200}
	simRes := []byte("simulation_result")

//签署以获得提案响应
	presp, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response, simRes, nil, getChaincodeID(), nil, signer)
	if err != nil {
		t.Fatalf("CreateProposalResponse failed, err %s", err)
		return
	}

//根据该提议和背书进行交易
	tx, err := utils.CreateSignedTx(prop, signer, presp)
	if err != nil {
		t.Fatalf("CreateSignedTx failed, err %s", err)
		return
	}

//处理事务负载
	paylOrig := tx.Payload
	for i := 0; i < len(paylOrig); i++ {
		paylCopy := make([]byte, len(paylOrig))
		copy(paylCopy, paylOrig)
		paylCopy[i] = byte(int(paylCopy[i]+1) % 255)
//验证它应该失败的事务
		_, txResult := ValidateTransaction(&common.Envelope{Signature: tx.Signature, Payload: paylCopy}, &config.MockApplicationCapabilities{})
		if txResult == peer.TxValidationCode_VALID {
			t.Fatal("ValidateTransaction should have failed")
			return
		}
	}

//根据该提议和背书进行交易
	tx, err = utils.CreateSignedTx(prop, signer, presp)
	if err != nil {
		t.Fatalf("CreateSignedTx failed, err %s", err)
		return
	}

//处理事务负载
	corrupt(tx.Signature)

//验证它应该失败的事务
	_, txResult := ValidateTransaction(tx, &config.MockApplicationCapabilities{})
	if txResult == peer.TxValidationCode_VALID {
		t.Fatal("ValidateTransaction should have failed")
		return
	}
}

func Test2EndorsersAgree(t *testing.T) {
//得到一个玩具建议
	prop, err := getProposal(util.GetTestChainID())
	if err != nil {
		t.Fatalf("getProposal failed, err %s", err)
		return
	}

	response1 := &peer.Response{Status: 200}
	simRes1 := []byte("simulation_result")

//签署以获得提案响应
	presp1, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response1, simRes1, nil, getChaincodeID(), nil, signer)
	if err != nil {
		t.Fatalf("CreateProposalResponse failed, err %s", err)
		return
	}

	response2 := &peer.Response{Status: 200}
	simRes2 := []byte("simulation_result")

//签署以获得提案响应
	presp2, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response2, simRes2, nil, getChaincodeID(), nil, signer)
	if err != nil {
		t.Fatalf("CreateProposalResponse failed, err %s", err)
		return
	}

//根据该提议和背书进行交易
	tx, err := utils.CreateSignedTx(prop, signer, presp1, presp2)
	if err != nil {
		t.Fatalf("CreateSignedTx failed, err %s", err)
		return
	}

//验证事务
	_, txResult := ValidateTransaction(tx, &config.MockApplicationCapabilities{})
	if txResult != peer.TxValidationCode_VALID {
		t.Fatalf("ValidateTransaction failed, err %s", err)
		return
	}
}

func Test2EndorsersDisagree(t *testing.T) {
//得到一个玩具建议
	prop, err := getProposal(util.GetTestChainID())
	if err != nil {
		t.Fatalf("getProposal failed, err %s", err)
		return
	}

	response1 := &peer.Response{Status: 200}
	simRes1 := []byte("simulation_result1")

//签署以获得提案响应
	presp1, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response1, simRes1, nil, getChaincodeID(), nil, signer)
	if err != nil {
		t.Fatalf("CreateProposalResponse failed, err %s", err)
		return
	}

	response2 := &peer.Response{Status: 200}
	simRes2 := []byte("simulation_result2")

//签署以获得提案响应
	presp2, err := utils.CreateProposalResponse(prop.Header, prop.Payload, response2, simRes2, nil, getChaincodeID(), nil, signer)
	if err != nil {
		t.Fatalf("CreateProposalResponse failed, err %s", err)
		return
	}

//根据该提议和背书进行交易
	_, err = utils.CreateSignedTx(prop, signer, presp1, presp2)
	if err == nil {
		t.Fatal("CreateSignedTx should have failed")
		return
	}
}

func TestInvocationsBadArgs(t *testing.T) {
	_, code := ValidateTransaction(nil, &config.MockApplicationCapabilities{})
	assert.Equal(t, code, peer.TxValidationCode_NIL_ENVELOPE)
	err := validateEndorserTransaction(nil, nil)
	assert.Error(t, err)
	err = validateConfigTransaction(nil, nil)
	assert.Error(t, err)
	_, _, err = validateCommonHeader(nil)
	assert.Error(t, err)
	err = validateChannelHeader(nil)
	assert.Error(t, err)
	err = validateChannelHeader(&common.ChannelHeader{})
	assert.Error(t, err)
	err = validateSignatureHeader(nil)
	assert.Error(t, err)
	err = validateSignatureHeader(&common.SignatureHeader{})
	assert.Error(t, err)
	err = validateSignatureHeader(&common.SignatureHeader{Nonce: []byte("a")})
	assert.Error(t, err)
	err = checkSignatureFromCreator(nil, nil, nil, "")
	assert.Error(t, err)
	_, _, _, err = ValidateProposalMessage(nil)
	assert.Error(t, err)
	_, err = validateChaincodeProposalMessage(nil, nil)
	assert.Error(t, err)
	_, err = validateChaincodeProposalMessage(&peer.Proposal{}, &common.Header{ChannelHeader: []byte("a"), SignatureHeader: []byte("a")})
	assert.Error(t, err)
}

var signer msp.SigningIdentity
var signerSerialized []byte
var signerMSPId string

func TestMain(m *testing.M) {
//设置加密算法
//设置MSP管理器，以便我们可以签名/验证
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		fmt.Printf("Could not initialize msp, err %s", err)
		os.Exit(-1)
		return
	}

	signer, err = mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		fmt.Println("Could not get signer")
		os.Exit(-1)
		return
	}
	signerMSPId = signer.GetMSPIdentifier()

	signerSerialized, err = signer.Serialize()
	if err != nil {
		fmt.Println("Could not serialize identity")
		os.Exit(-1)
		return
	}

	os.Exit(m.Run())
}
