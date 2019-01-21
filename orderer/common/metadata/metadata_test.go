
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


package metadata_test

import (
	"fmt"
	"runtime"
	"testing"

	common "github.com/hyperledger/fabric/common/metadata"
	"github.com/hyperledger/fabric/orderer/common/metadata"
	"github.com/stretchr/testify/assert"
)

func TestGetVersionInfo(t *testing.T) {
//对于开发版本，此测试总是失败，因为
//common.version未设置，返回的字符串为“开发版本”
//在此设置此测试以避免此情况。
	if common.Version == "" {
		common.Version = "testVersion"
	}

	expected := fmt.Sprintf(
		"%s:\n Version: %s\n Commit SHA: %s\n Go version: %s\n OS/Arch: %s\n",
		metadata.ProgramName, common.Version,
		common.CommitSHA,
		runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	)
	assert.Equal(t, expected, metadata.GetVersionInfo())
}
