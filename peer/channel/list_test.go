
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
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/peer/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

func TestListChannels(t *testing.T) {
	InitMSP()

	mockChannelResponse := &pb.ChannelQueryResponse{
		Channels: []*pb.ChannelInfo{{
			ChannelId: "TEST_LIST_CHANNELS",
		}},
	}

	mockPayload, err := proto.Marshal(mockChannelResponse)
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

	cmd := listCmd(mockCF)
	AddFlags(cmd)
	if err := cmd.Execute(); err != nil {
		t.Fail()
		t.Error(err)
	}

	testListChannelsEmptyCF(t, mockCF)
}

func testListChannelsEmptyCF(t *testing.T, mockCF *ChannelCmdFactory) {
	cmd := listCmd(nil)
	AddFlags(cmd)

//错误案例1:没有订购方终结点
	getEndorserClient := common.GetEndorserClientFnc
	getBroadcastClient := common.GetBroadcastClientFnc
	getDefaultSigner := common.GetDefaultSignerFnc
	defer func() {
		common.GetEndorserClientFnc = getEndorserClient
		common.GetBroadcastClientFnc = getBroadcastClient
		common.GetDefaultSignerFnc = getDefaultSigner
	}()
	common.GetDefaultSignerFnc = func() (msp.SigningIdentity, error) {
		return nil, errors.New("error")
	}
	common.GetEndorserClientFnc = func(string, string) (pb.EndorserClient, error) {
		return mockCF.EndorserClient, nil
	}
	common.GetBroadcastClientFnc = func() (common.BroadcastClient, error) {
		broadcastClient := common.GetMockBroadcastClient(nil)
		return broadcastClient, nil
	}

	err := cmd.Execute()
	assert.Error(t, err, "Error expected because GetDefaultSignerFnc returns an error")

	common.GetDefaultSignerFnc = getDefaultSigner
	common.GetEndorserClientFnc = func(string, string) (pb.EndorserClient, error) {
		return nil, errors.New("error")
	}
	err = cmd.Execute()
	assert.Error(t, err, "Error expected because GetEndorserClientFnc returns an error")

	common.GetEndorserClientFnc = func(string, string) (pb.EndorserClient, error) {
		return mockCF.EndorserClient, nil
	}
	err = cmd.Execute()
	assert.NoError(t, err, "Error occurred while executing list command")
}
