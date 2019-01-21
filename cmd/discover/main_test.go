
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


package main

import (
	"os/exec"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func TestMissingArguments(t *testing.T) {
	gt := NewGomegaWithT(t)
	discover, err := Build("github.com/hyperledger/fabric/cmd/discover")
	gt.Expect(err).NotTo(HaveOccurred())
	defer CleanupBuildArtifacts()

//缺少密钥和证书标志
	cmd := exec.Command(discover, "--configFile", "conf.yaml", "--MSP", "SampleOrg", "saveConfig")
	process, err := Start(cmd, nil, nil)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Eventually(process).Should(Exit(1))
	gt.Expect(process.Err).To(gbytes.Say("empty string that is mandatory"))
}
