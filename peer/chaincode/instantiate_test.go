
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstantiateCmd(t *testing.T) {
	mockCF, err := getMockChaincodeCmdFactory()
	assert.NoError(t, err, "Error getting mock chaincode command factory")

//基本功能测试
	var tests = []struct {
		name          string
		args          []string
		errorExpected bool
		errMsg        string
	}{
		{
			name:          "successful",
			args:          []string{"-n", "example02", "-v", "anotherversion", "-C", "mychannel", "-c", "{\"Args\": [\"init\",\"a\",\"100\",\"b\",\"200\"]}"},
			errorExpected: false,
			errMsg:        "Run chaincode instantiate cmd error",
		},
		{
			name:          "no option",
			args:          []string{},
			errorExpected: true,
			errMsg:        "Expected error executing instantiate command without required options",
		},
		{
			name:          "missing version",
			args:          []string{"-n", "example02", "-C", "mychannel", "-c", "{\"Args\": [\"init\",\"a\",\"100\",\"b\",\"200\"]}"},
			errorExpected: true,
			errMsg:        "Expected error executing instantiate command without the -v option",
		},
		{
			name:          "missing name",
			args:          []string{"-v", "anotherversion", "-C", "mychannel", "-c", "{\"Args\": [\"init\",\"a\",\"100\",\"b\",\"200\"]}"},
			errorExpected: true,
			errMsg:        "Expected error executing instantiate command without the -n option",
		},
		{
			name:          "missing channelID",
			args:          []string{"-n", "example02", "-v", "anotherversion", "-c", "{\"Args\": [\"init\",\"a\",\"100\",\"b\",\"200\"]}"},
			errorExpected: true,
			errMsg:        "Expected error executing instantiate command without the -C option",
		},
		{
			name:          "missing ctor",
			args:          []string{"-n", "example02", "-C", "mychannel", "-v", "anotherversion"},
			errorExpected: true,
			errMsg:        "Expected error executing instantiate command without the -c option",
		},
		{
			name:          "successful with policy",
			args:          []string{"-P", "OR('MSP.member', 'MSP.WITH.DOTS.member', 'MSP-WITH-DASHES.member')", "-n", "example02", "-v", "anotherversion", "-C", "mychannel", "-c", "{\"Args\": [\"init\",\"a\",\"100\",\"b\",\"200\"]}"},
			errorExpected: false,
			errMsg:        "Run chaincode instantiate cmd error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetFlags()
			cmd := instantiateCmd(mockCF)
			addFlags(cmd)
			cmd.SetArgs(test.args)
			err = cmd.Execute()
			checkError(t, err, test.errorExpected, test.errMsg)
		})
	}
}

func checkError(t *testing.T, err error, expectedError bool, msg string) {
	if expectedError {
		assert.Error(t, err, msg)
	} else {
		assert.NoError(t, err, msg)
	}
}
