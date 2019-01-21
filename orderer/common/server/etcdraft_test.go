
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


package server_test

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

func TestSpawnEtcdRaft(t *testing.T) {
	t.Skip()
	gt := NewGomegaWithT(t)

	cwd, err := filepath.Abs(".")
	gt.Expect(err).NotTo(HaveOccurred())

//创建用于存储系统通道的Genesis块的tempdir
	tempDir, err := ioutil.TempDir("", "etcdraft-orderer-launch")
	gt.Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempDir)

//将Fabric根文件夹设置为便于导航到sampleconfig文件夹
	fabricRootDir, err := filepath.Abs(filepath.Join("..", "..", ".."))
	gt.Expect(err).NotTo(HaveOccurred())

//构建configtxgen二进制文件
	configtxgen, err := gexec.Build("github.com/hyperledger/fabric/common/tools/configtxgen")
	gt.Expect(err).NotTo(HaveOccurred())

//生成排序器二进制文件
	orderer, err := gexec.Build("github.com/hyperledger/fabric/orderer")
	gt.Expect(err).NotTo(HaveOccurred())

	defer gexec.CleanupBuildArtifacts()

//为系统通道创建Genesis块
	genesisBlockPath := filepath.Join(tempDir, "genesis.block")
	cmd := exec.Command(configtxgen, "-channelID", "system", "-profile", "SampleDevModeEtcdRaft",
		"-outputBlock", genesisBlockPath)
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABRIC_CFG_PATH=%s", filepath.Join(cwd, "testdata")))
	configtxgenProcess, err := gexec.Start(cmd, nil, nil)
	gt.Expect(err).NotTo(HaveOccurred())

	gt.Eventually(configtxgenProcess, time.Minute).Should(gexec.Exit(0))
	gt.Expect(configtxgenProcess.Err).To(gbytes.Say("Writing genesis block"))

//发射OSN
	ordererProcess := launchOrderer(gt, cmd, orderer, tempDir, genesisBlockPath, fabricRootDir)
	gt.Eventually(ordererProcess.Err, time.Minute).Should(gbytes.Say("Beginning to serve requests"))
	gt.Eventually(ordererProcess.Err, time.Minute).Should(gbytes.Say("becomeLeader"))
	ordererProcess.Kill()
}

func launchOrderer(gt *GomegaWithT, cmd *exec.Cmd, orderer, tempDir, genesisBlockPath, fabricRootDir string) *gexec.Session {
	cwd, err := filepath.Abs(".")
	gt.Expect(err).NotTo(HaveOccurred())

//启动订购程序进程
	cmd = exec.Command(orderer)
	cmd.Env = []string{
		"ORDERER_GENERAL_LISTENPORT=5611",
		"ORDERER_GENERAL_GENESISMETHOD=file",
		"ORDERER_GENERAL_SYSTEMCHANNEL=system",
		"ORDERER_GENERAL_TLS_CLIENTAUTHREQUIRED=true",
		"ORDERER_GENERAL_TLS_ENABLED=true",
		"ORDERER_OPERATIONS_TLS_ENABLED=false",
		fmt.Sprintf("ORDERER_FILELEDGER_LOCATION=%s", filepath.Join(tempDir, "ledger")),
		fmt.Sprintf("ORDERER_GENERAL_GENESISFILE=%s", genesisBlockPath),
		fmt.Sprintf("ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE=%s", filepath.Join(cwd, "testdata", "tls", "server.crt")),
		fmt.Sprintf("ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY=%s", filepath.Join(cwd, "testdata", "tls", "server.key")),
		fmt.Sprintf("ORDERER_GENERAL_CLUSTER_ROOTCAS=[%s]", filepath.Join(cwd, "testdata", "tls", "ca.crt")),
		fmt.Sprintf("ORDERER_GENERAL_TLS_ROOTCAS=[%s]", filepath.Join(cwd, "testdata", "tls", "ca.crt")),
		fmt.Sprintf("ORDERER_GENERAL_TLS_CERTIFICATE=%s", filepath.Join(cwd, "testdata", "tls", "server.crt")),
		fmt.Sprintf("ORDERER_GENERAL_TLS_PRIVATEKEY=%s", filepath.Join(cwd, "testdata", "tls", "server.key")),
		fmt.Sprintf("ORDERER_CONSENSUS_WALDIR=%s", filepath.Join(tempDir, "wal")),
		fmt.Sprintf("ORDERER_CONSENSUS_SNAPDIR=%s", filepath.Join(tempDir, "snapshot")),
		fmt.Sprintf("FABRIC_CFG_PATH=%s", filepath.Join(fabricRootDir, "sampleconfig")),
	}
	sess, err := gexec.Start(cmd, nil, nil)
	gt.Expect(err).NotTo(HaveOccurred())
	return sess
}
