
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


package testutils

import (
	"fmt"
	"os"

	"github.com/hyperledger/fabric/common/crypto"
	mmsp "github.com/hyperledger/fabric/common/mocks/msp"
	"github.com/hyperledger/fabric/msp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
)

var (
	signer msp.SigningIdentity
)

func init() {
	var err error
//设置MSP管理器，以便我们可以签名/验证
	err = msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		fmt.Printf("Could not load msp config, err %s", err)
		os.Exit(-1)
		return
	}
	signer, err = mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		os.Exit(-1)
		fmt.Printf("Could not initialize msp/signer")
		return
	}
}

//constructBytesProposalResponsePayLoad为具有默认签名者的测试构造ProposalResponsePayLoad字节。
func ConstructBytesProposalResponsePayload(chainID string, ccid *pb.ChaincodeID, pResponse *pb.Response, simulationResults []byte) ([]byte, error) {
	ss, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	prop, _, err := putils.CreateChaincodeProposal(common.HeaderType_ENDORSER_TRANSACTION, chainID, &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{ChaincodeId: ccid}}, ss)
	if err != nil {
		return nil, err
	}

	presp, err := putils.CreateProposalResponse(prop.Header, prop.Payload, pResponse, simulationResults, nil, ccid, nil, signer)
	if err != nil {
		return nil, err
	}

	return presp.Payload, nil
}

//ConstructSignedXenvWithDefaultSigner为带有默认签名者的测试构造一个事务信封。
//此方法帮助其他模块使用提供的参数构造事务。
func ConstructSignedTxEnvWithDefaultSigner(chainID string, ccid *pb.ChaincodeID, response *pb.Response, simulationResults []byte, txid string, events []byte, visibility []byte) (*common.Envelope, string, error) {
	return ConstructSignedTxEnv(chainID, ccid, response, simulationResults, txid, events, visibility, signer)
}

//ConstructSignedXenv构造用于测试的事务信封
func ConstructSignedTxEnv(chainID string, ccid *pb.ChaincodeID, pResponse *pb.Response, simulationResults []byte, txid string, events []byte, visibility []byte, signer msp.SigningIdentity) (*common.Envelope, string, error) {
	ss, err := signer.Serialize()
	if err != nil {
		return nil, "", err
	}

	var prop *pb.Proposal
	if txid == "" {
//如果没有设置txid，那么我们需要在创建建议消息时生成一个。
		prop, txid, err = putils.CreateChaincodeProposal(common.HeaderType_ENDORSER_TRANSACTION, chainID, &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{ChaincodeId: ccid}}, ss)

	} else {
//如果设置了txid，则不应生成txid，而应重新使用给定的txid
		nonce, err := crypto.GetRandomNonce()
		if err != nil {
			return nil, "", err
		}
		prop, txid, err = putils.CreateChaincodeProposalWithTxIDNonceAndTransient(txid, common.HeaderType_ENDORSER_TRANSACTION, chainID, &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{ChaincodeId: ccid}}, nonce, ss, nil)
	}
	if err != nil {
		return nil, "", err
	}

	presp, err := putils.CreateProposalResponse(prop.Header, prop.Payload, pResponse, simulationResults, nil, ccid, nil, signer)
	if err != nil {
		return nil, "", err
	}

	env, err := putils.CreateSignedTx(prop, signer, presp)
	if err != nil {
		return nil, "", err
	}
	return env, txid, nil
}

var mspLcl msp.MSP
var sigId msp.SigningIdentity

//constructUnsignedXenv从给定的输入创建一个事务信封
func ConstructUnsignedTxEnv(chainID string, ccid *pb.ChaincodeID, response *pb.Response, simulationResults []byte, txid string, events []byte, visibility []byte) (*common.Envelope, string, error) {
	if mspLcl == nil {
		mspLcl = mmsp.NewNoopMsp()
		sigId, _ = mspLcl.GetDefaultSigningIdentity()
	}

	return ConstructSignedTxEnv(chainID, ccid, response, simulationResults, txid, events, visibility, sigId)
}
