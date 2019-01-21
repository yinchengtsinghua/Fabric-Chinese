
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 版权所有Digital Asset Holdings，LLC 2017保留所有权利。

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


package channel

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hyperledger/fabric/peer/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMissingBlockFile(t *testing.T) {
	defer resetFlags()

	resetFlags()

	cmd := joinCmd(nil)
	AddFlags(cmd)
	args := []string{}
	cmd.SetArgs(args)

	assert.Error(t, cmd.Execute(), "expected join command to fail due to missing blockfilepath")
}

func TestJoin(t *testing.T) {
	defer resetFlags()

	InitMSP()
	resetFlags()

	dir, err := ioutil.TempDir("/tmp", "jointest")
	assert.NoError(t, err, "Could not create the directory %s", dir)
	mockblockfile := filepath.Join(dir, "mockjointest.block")
	err = ioutil.WriteFile(mockblockfile, []byte(""), 0644)
	assert.NoError(t, err, "Could not write to the file %s", mockblockfile)
	defer os.RemoveAll(dir)
	signer, err := common.GetDefaultSigner()
	assert.NoError(t, err, "Get default signer error: %v", err)

	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 200},
		Endorsement: &pb.Endorsement{},
	}

	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)

	mockCF := &ChannelCmdFactory{
		EndorserClient:   mockEndorserClient,
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
	}

	cmd := joinCmd(mockCF)
	AddFlags(cmd)

	args := []string{"-b", mockblockfile}
	cmd.SetArgs(args)

	assert.NoError(t, cmd.Execute(), "expected join command to succeed")
}

func TestJoinNonExistentBlock(t *testing.T) {
	defer resetFlags()

	InitMSP()
	resetFlags()

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 200},
		Endorsement: &pb.Endorsement{},
	}

	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)

	mockCF := &ChannelCmdFactory{
		EndorserClient:   mockEndorserClient,
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
	}

	cmd := joinCmd(mockCF)

	AddFlags(cmd)

	args := []string{"-b", "mockchain.block"}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.Error(t, err, "expected join command to fail")
	assert.IsType(t, GBFileNotFoundErr(err.Error()), err, "expected error type of GBFileNotFoundErr")
}

func TestBadProposalResponse(t *testing.T) {
	defer resetFlags()

	InitMSP()
	resetFlags()

	mockblockfile := "/tmp/mockjointest.block"
	ioutil.WriteFile(mockblockfile, []byte(""), 0644)
	defer os.Remove(mockblockfile)
	signer, err := common.GetDefaultSigner()
	assert.NoError(t, err, "Get default signer error: %v", err)

	mockResponse := &pb.ProposalResponse{
		Response:    &pb.Response{Status: 500},
		Endorsement: &pb.Endorsement{},
	}

	mockEndorserClient := common.GetMockEndorserClient(mockResponse, nil)

	mockCF := &ChannelCmdFactory{
		EndorserClient:   mockEndorserClient,
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
	}

	cmd := joinCmd(mockCF)

	AddFlags(cmd)

	args := []string{"-b", mockblockfile}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.Error(t, err, "expected join command to fail")
	assert.IsType(t, ProposalFailedErr(err.Error()), err, "expected error type of ProposalFailedErr")
}

func TestJoinNilCF(t *testing.T) {
	defer viper.Reset()
	defer resetFlags()

	InitMSP()
	resetFlags()

	dir, err := ioutil.TempDir("/tmp", "jointest")
	assert.NoError(t, err, "Could not create the directory %s", dir)
	mockblockfile := filepath.Join(dir, "mockjointest.block")
	defer os.RemoveAll(dir)
	viper.Set("peer.client.connTimeout", 10*time.Millisecond)
	cmd := joinCmd(nil)
	AddFlags(cmd)
	args := []string{"-b", mockblockfile}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endorser client failed to connect to")
}
