
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


package chaincode

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/peer/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestQueryCmd(t *testing.T) {
	mockCF, err := getMockChaincodeCmdFactory()
	assert.NoError(t, err, "Error getting mock chaincode command factory")
//重置channelid，它可能是由以前的测试设置的
	channelID = ""

//失败案例：不带-c选项运行查询命令
	args := []string{"-n", "example02", "-c", "{\"Args\": [\"query\",\"a\"]}"}
	cmd := newQueryCmdForTest(mockCF, args)
	err = cmd.Execute()
	assert.Error(t, err, "'peer chaincode query' command should have failed without -C flag")

//成功案例：不带-r或-x选项运行查询命令
	args = []string{"-C", "mychannel", "-n", "example02", "-c", "{\"Args\": [\"query\",\"a\"]}"}
	cmd = newQueryCmdForTest(mockCF, args)
	err = cmd.Execute()
	assert.NoError(t, err, "Run chaincode query cmd error")

//成功案例：使用-r选项运行查询命令
	args = []string{"-r", "-C", "mychannel", "-n", "example02", "-c", "{\"Args\": [\"query\",\"a\"]}"}
	cmd = newQueryCmdForTest(mockCF, args)
	err = cmd.Execute()
	assert.NoError(t, err, "Run chaincode query cmd error")
	chaincodeQueryRaw = false

//成功案例：使用-x选项运行查询命令
	args = []string{"-x", "-C", "mychannel", "-n", "example02", "-c", "{\"Args\": [\"query\",\"a\"]}"}
	cmd = newQueryCmdForTest(mockCF, args)
	err = cmd.Execute()
	assert.NoError(t, err, "Run chaincode query cmd error")

//失败案例：使用-x和-r选项运行查询命令
	args = []string{"-r", "-x", "-C", "mychannel", "-n", "example02", "-c", "{\"Args\": [\"query\",\"a\"]}"}
	cmd = newQueryCmdForTest(mockCF, args)
	err = cmd.Execute()
	assert.Error(t, err, "Expected error executing query command with both -r and -x options")

//失败案例：使用mock chaincode cmd factory build运行查询命令以返回错误
	mockCF, err = getMockChaincodeCmdFactoryWithErr()
	assert.NoError(t, err, "Error getting mock chaincode command factory")
	args = []string{"-r", "-n", "example02", "-c", "{\"Args\": [\"query\",\"a\"]}"}
	cmd = newQueryCmdForTest(mockCF, args)
	err = cmd.Execute()
	assert.Error(t, err, "Expected error executing query command")
}

func TestQueryCmdEndorsementFailure(t *testing.T) {
	args := []string{"-C", "mychannel", "-n", "example02", "-c", "{\"Args\": [\"queryinvalid\",\"a\"]}"}
	ccRespStatus := [2]int32{502, 400}
	ccRespPayload := [][]byte{[]byte("Invalid function name"), []byte("Incorrect parameters")}

	for i := 0; i < 2; i++ {
		mockCF, err := getMockChaincodeCmdFactoryEndorsementFailure(ccRespStatus[i], ccRespPayload[i])
		assert.NoError(t, err, "Error getting mock chaincode command factory")

		cmd := newQueryCmdForTest(mockCF, args)
		err = cmd.Execute()
		assert.Error(t, err)
		assert.Regexp(t, "endorsement failure during query", err.Error())
		assert.Regexp(t, fmt.Sprintf("response: status:%d payload:\"%s\"", ccRespStatus[i], ccRespPayload[i]), err.Error())
	}

//失败-无提案响应
	mockCF, err := getMockChaincodeCmdFactory()
	assert.NoError(t, err, "Error getting mock chaincode command factory")
	mockCF.EndorserClients[0] = common.GetMockEndorserClient(nil, nil)

	cmd := newQueryCmdForTest(mockCF, args)
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Regexp(t, "error during query: received nil proposal response", err.Error())
}

func newQueryCmdForTest(cf *ChaincodeCmdFactory, args []string) *cobra.Command {
	cmd := queryCmd(cf)
	addFlags(cmd)
	cmd.SetArgs(args)
	return cmd
}
