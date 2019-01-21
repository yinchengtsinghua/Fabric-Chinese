
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


package channel

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/peer/common"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

func TestGetChannelInfo(t *testing.T) {
	InitMSP()
	resetFlags()

	mockBlockchainInfo := &cb.BlockchainInfo{
		Height:            1,
		CurrentBlockHash:  []byte("CurrentBlockHash"),
		PreviousBlockHash: []byte("PreviousBlockHash"),
	}
	mockPayload, err := proto.Marshal(mockBlockchainInfo)
	assert.NoError(t, err)

	mockResponse := &pb.ProposalResponse{
		Response: &pb.Response{
			Status:  200,
			Payload: mockPayload,
		},
		Endorsement: &pb.Endorsement{},
	}

	signer, err := common.GetDefaultSigner()
	assert.NoError(t, err)

	mockCF := &ChannelCmdFactory{
		EndorserClient:   common.GetMockEndorserClient(mockResponse, nil),
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
	}

	cmd := getinfoCmd(mockCF)
	AddFlags(cmd)

	args := []string{"-c", mockChannel}
	cmd.SetArgs(args)

	assert.NoError(t, cmd.Execute())
}

func TestGetChannelInfoMissingChannelID(t *testing.T) {
	InitMSP()
	resetFlags()

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		Signer: signer,
	}

	cmd := getinfoCmd(mockCF)

	AddFlags(cmd)

	cmd.SetArgs([]string{})

	assert.Error(t, cmd.Execute())
}
