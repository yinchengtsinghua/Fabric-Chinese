
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


package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func TestPluginLoadingFailure(t *testing.T) {
	gt := NewGomegaWithT(t)
	peer, err := gexec.Build("github.com/hyperledger/fabric/peer")
	gt.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	parentDir, err := filepath.Abs("..")
	gt.Expect(err).NotTo(HaveOccurred())

	tempDir, err := ioutil.TempDir("", "plugin-failure")
	gt.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempDir)

	for _, plugin := range []string{
		"ENDORSERS_ESCC",
		"VALIDATORS_VSCC",
	} {
		plugin := plugin
		t.Run(plugin, func(t *testing.T) {
			cmd := exec.Command(peer, "node", "start")
			cmd.Env = []string{
				fmt.Sprintf("CORE_PEER_FILESYSTEMPATH=%s", tempDir),
				fmt.Sprintf("CORE_PEER_HANDLERS_%s_LIBRARY=testdata/invalid_plugins/invalidplugin.so", plugin),
				fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=%s", "msp"),
				fmt.Sprintf("FABRIC_CFG_PATH=%s", filepath.Join(parentDir, "sampleconfig")),
				"CORE_OPERATIONS_TLS_ENABLED=false",
			}

			sess, err := gexec.Start(cmd, nil, nil)
			gt.Expect(err).NotTo(HaveOccurred())
			gt.Eventually(sess, time.Minute).Should(gexec.Exit(2))

			gt.Expect(sess.Err).To(gbytes.Say("panic: Error opening plugin at path testdata/invalid_plugins/invalidplugin.so"))
			gt.Expect(sess.Err).To(gbytes.Say("plugin.Open"))
		})
	}
}
