
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


package nwo_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	yaml "gopkg.in/yaml.v2"
)

var _ = Describe("Network", func() {
	var (
		client  *docker.Client
		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "nwo")
		Expect(err).NotTo(HaveOccurred())

		client, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("solo network", func() {
		var network *nwo.Network
		var process ifrit.Process

		BeforeEach(func() {
			soloBytes, err := ioutil.ReadFile("solo.yaml")
			Expect(err).NotTo(HaveOccurred())

			var config *nwo.Config
			err = yaml.Unmarshal(soloBytes, &config)
			Expect(err).NotTo(HaveOccurred())
			network = nwo.New(config, tempDir, client, 44444, components)

//生成配置并引导网络
			network.GenerateConfigTree()
			network.Bootstrap()

//开始所有的织物加工
			networkRunner := network.NetworkGroupRunner()
			process = ifrit.Invoke(networkRunner)
			Eventually(process.Ready(), network.EventuallyTimeout).Should(BeClosed())
		})

		AfterEach(func() {
//关闭流程和清理
			process.Signal(syscall.SIGTERM)
			Eventually(process.Wait(), network.EventuallyTimeout).Should(Receive())
			network.Cleanup()
		})

		It("deploys and executes chaincode (simple)", func() {
			orderer := network.Orderer("orderer0")
			peer := network.Peer("org1", "peer2")

			chaincode := nwo.Chaincode{
				Name:    "mycc",
				Version: "0.0",
				Path:    "github.com/hyperledger/fabric/integration/chaincode/simple/cmd",
				Ctor:    `{"Args":["init","a","100","b","200"]}`,
				Policy:  `AND ('Org1ExampleCom.member','Org2ExampleCom.member')`,
			}

			network.CreateAndJoinChannels(orderer)
			nwo.DeployChaincode(network, "testchannel", orderer, chaincode)
			RunQueryInvokeQuery(network, orderer, peer)
		})
	})

	Describe("kafka network", func() {
		var (
			config    nwo.Config
			network   *nwo.Network
			processes map[string]ifrit.Process
		)

		BeforeEach(func() {
			soloBytes, err := ioutil.ReadFile("solo.yaml")
			Expect(err).NotTo(HaveOccurred())

			err = yaml.Unmarshal(soloBytes, &config)
			Expect(err).NotTo(HaveOccurred())

//从Solo切换到Kafka
			config.Consensus.Type = "kafka"
			config.Consensus.ZooKeepers = 1
			config.Consensus.Brokers = 1

			network = nwo.New(&config, tempDir, client, 55555, components)
			network.GenerateConfigTree()
			network.Bootstrap()
			processes = map[string]ifrit.Process{}
		})

		AfterEach(func() {
			for _, p := range processes {
				p.Signal(syscall.SIGTERM)
				Eventually(p.Wait(), network.EventuallyTimeout).Should(Receive())
			}
			network.Cleanup()
		})

		It("deploys and executes chaincode (the hard way)", func() {
//这演示了如何控制构成网络的进程。
//如果您不关心流程集合（如经纪人或
//医嘱者）使用Group Runner来管理这些流程。
			zookeepers := []string{}
			for i := 0; i < network.Consensus.ZooKeepers; i++ {
				zk := network.ZooKeeperRunner(i)
				zookeepers = append(zookeepers, fmt.Sprintf("%s:2181", zk.Name))

				p := ifrit.Invoke(zk)
				processes[zk.Name] = p
				Eventually(p.Ready(), network.EventuallyTimeout).Should(BeClosed())
			}

			for i := 0; i < network.Consensus.Brokers; i++ {
				b := network.BrokerRunner(i, zookeepers)
				p := ifrit.Invoke(b)
				processes[b.Name] = p
				Eventually(p.Ready(), network.EventuallyTimeout).Should(BeClosed())
			}

			for _, o := range network.Orderers {
				or := network.OrdererRunner(o)
				p := ifrit.Invoke(or)
				processes[o.ID()] = p
				Eventually(p.Ready(), network.EventuallyTimeout).Should(BeClosed())
			}

			for _, peer := range network.Peers {
				pr := network.PeerRunner(peer)
				p := ifrit.Invoke(pr)
				processes[peer.ID()] = p
				Eventually(p.Ready(), network.EventuallyTimeout).Should(BeClosed())
			}

			orderer := network.Orderer("orderer0")
			testPeers := network.PeersWithChannel("testchannel")
			network.CreateChannel("testchannel", orderer, testPeers[0])
			network.JoinChannel("testchannel", orderer, testPeers...)

			chaincode := nwo.Chaincode{
				Name:    "mycc",
				Version: "0.0",
				Path:    "github.com/hyperledger/fabric/integration/chaincode/simple/cmd",
				Ctor:    `{"Args":["init","a","100","b","200"]}`,
				Policy:  `AND ('Org1ExampleCom.member','Org2ExampleCom.member')`,
			}
			nwo.InstallChaincode(network, chaincode, testPeers...)
			nwo.InstantiateChaincode(network, "testchannel", orderer, chaincode, testPeers[0])
			nwo.EnsureInstantiated(network, "testchannel", "mycc", "0.0", testPeers...)

			RunQueryInvokeQuery(network, orderer, testPeers[0])
		})
	})
})

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
			n.PeerAddress(n.Peer("org1", "peer1"), nwo.ListenPort),
			n.PeerAddress(n.Peer("org2", "peer2"), nwo.ListenPort),
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
