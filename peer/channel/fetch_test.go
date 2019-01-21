
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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/config/configtest"
	"github.com/hyperledger/fabric/peer/common"
	"github.com/hyperledger/fabric/peer/common/mock"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	defer resetFlags()
	InitMSP()
	resetFlags()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"

	signer, err := common.GetDefaultSigner()
	if err != nil {
		t.Fatalf("Get default signer error: %v", err)
	}

	mockCF := &ChannelCmdFactory{
		BroadcastFactory: mockBroadcastClientFactory,
		Signer:           signer,
		DeliverClient:    getMockDeliverClient(mockchain),
	}

	tempDir, err := ioutil.TempDir("", "fetch-output")
	if err != nil {
		t.Fatalf("failed to create temporary directory")
	}
	defer os.RemoveAll(tempDir)

	cmd := fetchCmd(mockCF)
	AddFlags(cmd)

//成功案例-block和outputblockpath
	blocksToFetch := []string{"oldest", "newest", "config", "1"}
	for _, block := range blocksToFetch {
		outputBlockPath := filepath.Join(tempDir, block+".block")
		args := []string{"-c", mockchain, block, outputBlockPath}
		cmd.SetArgs(args)

		err = cmd.Execute()
		assert.NoError(t, err, "fetch command expected to succeed")

		if _, err := os.Stat(outputBlockPath); os.IsNotExist(err) {
//路径/目标/不存在的任何内容
			t.Error("expected configuration block to be fetched")
			t.Fail()
		}
	}

//故障案例
	blocksToFetchBad := []string{"banana"}
	for _, block := range blocksToFetchBad {
		outputBlockPath := filepath.Join(tempDir, block+".block")
		args := []string{"-c", mockchain, block, outputBlockPath}
		cmd.SetArgs(args)

		err = cmd.Execute()
		assert.Error(t, err, "fetch command expected to fail")
		assert.Regexp(t, err.Error(), fmt.Sprintf("fetch target illegal: %s", block))

		if fileInfo, _ := os.Stat(outputBlockPath); fileInfo != nil {
//路径/目标/存在的任何内容
			t.Error("expected configuration block to not be fetched")
			t.Fail()
		}
	}
}

func TestFetchArgs(t *testing.T) {
//失败-无参数
	cmd := fetchCmd(nil)
	AddFlags(cmd)
	err := cmd.Execute()
	assert.Error(t, err, "fetch command expected to fail")
	assert.Contains(t, err.Error(), "fetch target required")

//失败-参数太多
	args := []string{"strawberry", "kiwi", "lemonade"}
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.Error(t, err, "fetch command expected to fail")
	assert.Contains(t, err.Error(), "trailing args detected")
}

func TestFetchNilCF(t *testing.T) {
	defer viper.Reset()
	defer resetFlags()

	InitMSP()
	resetFlags()
	cleanup := configtest.SetDevFabricConfigPath(t)
	defer cleanup()

	mockchain := "mockchain"
	viper.Set("peer.client.connTimeout", 10*time.Millisecond)
	cmd := fetchCmd(nil)
	AddFlags(cmd)
	args := []string{"-c", mockchain, "oldest"}
	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.Error(t, err, "fetch command expected to fail")
	assert.Contains(t, err.Error(), "deliver client failed to connect to")
}

func getMockDeliverClient(channelID string) *common.DeliverClient {
	p := getMockDeliverClientWithBlock(channelID, createTestBlock())
	return p
}

func getMockDeliverClientWithBlock(channelID string, block *cb.Block) *common.DeliverClient {
	p := &common.DeliverClient{
		Service:     getMockDeliverService(block),
		ChannelID:   channelID,
		TLSCertHash: []byte("tlscerthash"),
	}
	return p
}

func getMockDeliverService(block *cb.Block) *mock.DeliverService {
	mockD := &mock.DeliverService{}
	blockResponse := &ab.DeliverResponse{
		Type: &ab.DeliverResponse_Block{
			Block: block,
		},
	}
	statusResponse := &ab.DeliverResponse{
		Type: &ab.DeliverResponse_Status{Status: cb.Status_SUCCESS},
	}
	mockD.RecvStub = func() (*ab.DeliverResponse, error) {
//备用返回块和状态
		if mockD.RecvCallCount()%2 == 1 {
			return blockResponse, nil
		}
		return statusResponse, nil
	}
	mockD.CloseSendReturns(nil)
	return mockD
}

func createTestBlock() *cb.Block {
	lc := &cb.LastConfig{Index: 0}
	lcBytes := putils.MarshalOrPanic(lc)
	metadata := &cb.Metadata{
		Value: lcBytes,
	}
	metadataBytes := putils.MarshalOrPanic(metadata)
	blockMetadata := make([][]byte, cb.BlockMetadataIndex_LAST_CONFIG+1)
	blockMetadata[cb.BlockMetadataIndex_LAST_CONFIG] = metadataBytes
	block := &cb.Block{
		Header: &cb.BlockHeader{
			Number: 0,
		},
		Metadata: &cb.BlockMetadata{
			Metadata: blockMetadata,
		},
	}

	return block
}
