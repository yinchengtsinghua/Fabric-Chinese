
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package pluggable

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("EndToEnd", func() {
	var (
		testDir   string
		client    *docker.Client
		network   *nwo.Network
		chaincode nwo.Chaincode
		process   ifrit.Process

		endorsementPluginPath string
		validationPluginPath  string
	)

	BeforeEach(func() {
		var err error
		testDir, err = ioutil.TempDir("", "pluggable-suite")
		Expect(err).NotTo(HaveOccurred())

//编译插件
		endorsementPluginPath = compilePlugin("endorsement")
		validationPluginPath = compilePlugin("validation")

//创建用于认可和验证激活的目录
		dir := filepath.Join(testDir, "endorsement")
		err = os.Mkdir(dir, 0700)
		Expect(err).NotTo(HaveOccurred())
		SetEndorsementPluginActivationFolder(dir)

		dir = filepath.Join(testDir, "validation")
		err = os.Mkdir(dir, 0700)
		Expect(err).NotTo(HaveOccurred())
		SetValidationPluginActivationFolder(dir)

//通过减少我们提到的同龄人数量来加快测试速度
		soloConfig := nwo.BasicSolo()
		soloConfig.RemovePeer("Org1", "peer1")
		soloConfig.RemovePeer("Org2", "peer1")
		Expect(soloConfig.Peers).To(HaveLen(2))

//码头客户
		client, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())

		network = nwo.New(soloConfig, testDir, client, 33000, components)
		network.GenerateConfigTree()

//修改配置
		configurePlugins(network, endorsementPluginPath, validationPluginPath)

//生成网络配置
		network.Bootstrap()

		networkRunner := network.NetworkGroupRunner()
		process = ifrit.Invoke(networkRunner)
		Eventually(process.Ready()).Should(BeClosed())

		chaincode = nwo.Chaincode{
			Name:    "mycc",
			Version: "0.0",
			Path:    "github.com/hyperledger/fabric/integration/chaincode/simple/cmd",
			Ctor:    `{"Args":["init","a","100","b","200"]}`,
			Policy:  `OR ('Org1MSP.member','Org2MSP.member')`,
		}
		orderer := network.Orderer("orderer")
		network.CreateAndJoinChannel(orderer, "testchannel")
		nwo.DeployChaincode(network, "testchannel", orderer, chaincode)
	})

	AfterEach(func() {
//停止网络
		process.Signal(syscall.SIGTERM)
		Eventually(process.Wait()).Should(Receive())

//清除网络项目
		network.Cleanup()
		os.RemoveAll(testDir)

//清理已编译的插件
		os.Remove(endorsementPluginPath)
		os.Remove(validationPluginPath)
	})

	It("executes a basic solo network with specified plugins", func() {
//确保插件已激活
		peerCount := len(network.Peers)
		activations := CountEndorsementPluginActivations()
		Expect(activations).To(Equal(peerCount))
		activations = CountValidationPluginActivations()
		Expect(activations).To(Equal(peerCount))

		RunQueryInvokeQuery(network, network.Orderer("orderer"), network.Peer("Org1", "peer0"))
	})
})

//compilePlugin编译给定类型的插件并返回插件文件的路径
func compilePlugin(pluginType string) string {
	pluginFilePath := filepath.Join("testdata", "plugins", pluginType, "plugin.so")
	cmd := exec.Command(
		"go", "build", "-buildmode=plugin",
		"-o", pluginFilePath,
		fmt.Sprintf("github.com/hyperledger/fabric/integration/pluggable/testdata/plugins/%s", pluginType),
	)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, time.Minute).Should(gexec.Exit(0))

	Expect(pluginFilePath).To(BeARegularFile())
	return pluginFilePath
}

func configurePlugins(network *nwo.Network, endorsement, validation string) {
	for _, p := range network.Peers {
		core := network.ReadPeerConfig(p)
		core.Peer.Handlers.Endorsers = fabricconfig.HandlerMap{
			"escc": fabricconfig.Handler{Name: "plugin-escc", Library: endorsement},
		}
		core.Peer.Handlers.Validators = fabricconfig.HandlerMap{
			"vscc": fabricconfig.Handler{Name: "plugin-vscc", Library: validation},
		}
		network.WritePeerConfig(p, core)
	}
}

func RunQueryInvokeQuery(n *nwo.Network, orderer *nwo.Orderer, peer *nwo.Peer) {
	By("querying the chaincode")
	sess, err := n.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
		ChannelID: "testchannel",
		Name:      "mycc",
		Ctor:      `{"Args":["query","a"]}`,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess).To(gbytes.Say("100"))

	sess, err = n.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: "testchannel",
		Orderer:   n.OrdererAddress(orderer, nwo.ListenPort),
		Name:      "mycc",
		Ctor:      `{"Args":["invoke","a","b","10"]}`,
		PeerAddresses: []string{
			n.PeerAddress(n.Peer("Org1", "peer0"), nwo.ListenPort),
			n.PeerAddress(n.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	sess, err = n.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
		ChannelID: "testchannel",
		Name:      "mycc",
		Ctor:      `{"Args":["query","a"]}`,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess).To(gbytes.Say("90"))
}
