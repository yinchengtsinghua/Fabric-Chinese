
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


package nwo

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/tools/configtxlator/update"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

//getconfigblock检索通道的当前配置块，并
//解封它。
func GetConfigBlock(n *Network, peer *Peer, orderer *Orderer, channel string) *common.Config {
	tempDir, err := ioutil.TempDir("", "getConfigBlock")
	Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempDir)

//获取配置块
	output := filepath.Join(tempDir, "config_block.pb")
	sess, err := n.PeerAdminSession(peer, commands.ChannelFetch{
		ChannelID:  channel,
		Block:      "config",
		Orderer:    n.OrdererAddress(orderer, ListenPort),
		OutputFile: output,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Received block: "))

//取消配置块字节的标记
	configBlock := UnmarshalBlockFromFile(output)

//取消信封字节的标记
	envelope, err := utils.GetEnvelopeFromBlock(configBlock.Data.Data[0])
	Expect(err).NotTo(HaveOccurred())

//取消有效负载字节的标记
	payload, err := utils.GetPayload(envelope)
	Expect(err).NotTo(HaveOccurred())

//取消配置信封字节的标记
	configEnv := &common.ConfigEnvelope{}
	err = proto.Unmarshal(payload.Data, configEnv)
	Expect(err).NotTo(HaveOccurred())

//克隆配置
	return configEnv.Config
}

//updateConfig计算、签名并提交配置更新并等待
//以完成更新。
func UpdateConfig(n *Network, orderer *Orderer, channel string, current, updated *common.Config, submitter *Peer, additionalSigners ...*Peer) {
	tempDir, err := ioutil.TempDir("", "updateConfig")
	Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempDir)

//计算更新
	configUpdate, err := update.Compute(current, updated)
	Expect(err).NotTo(HaveOccurred())
	configUpdate.ChannelId = channel

	signedEnvelope, err := utils.CreateSignedEnvelope(
		common.HeaderType_CONFIG_UPDATE,
		channel,
nil, //本地签名者
		&common.ConfigUpdateEnvelope{ConfigUpdate: utils.MarshalOrPanic(configUpdate)},
0, //消息版本
0, //时代
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(signedEnvelope).NotTo(BeNil())

	updateFile := filepath.Join(tempDir, "update.pb")
	err = ioutil.WriteFile(updateFile, utils.MarshalOrPanic(signedEnvelope), 0600)
	Expect(err).NotTo(HaveOccurred())

	for _, signer := range additionalSigners {
		sess, err := n.PeerAdminSession(signer, commands.SignConfigTx{File: updateFile})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}

//获取当前配置块编号
	currentBlockNumber := CurrentConfigBlockNumber(n, submitter, orderer, channel)

	sess, err := n.PeerAdminSession(submitter, commands.ChannelUpdate{
		ChannelID: channel,
		Orderer:   n.OrdererAddress(orderer, ListenPort),
		File:      updateFile,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Successfully submitted channel update"))

//等待将该块提交给
//已加入频道
	for _, peer := range n.PeersWithChannel(channel) {
		ccb := func() uint64 { return CurrentConfigBlockNumber(n, peer, orderer, channel) }
		Eventually(ccb, n.EventuallyTimeout).Should(BeNumerically(">", currentBlockNumber))
	}
}

//updateOrdererConfig计算、签名并提交一个需要订购者签名和等待的配置更新
//以完成更新。
func UpdateOrdererConfig(n *Network, orderer *Orderer, channel string, current, updated *common.Config, submitter *Peer, additionalSigners ...*Orderer) {
	tempDir, err := ioutil.TempDir("", "updateConfig")
	Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempDir)

//计算更新
	configUpdate, err := update.Compute(current, updated)
	Expect(err).NotTo(HaveOccurred())
	configUpdate.ChannelId = channel

	signedEnvelope, err := utils.CreateSignedEnvelope(
		common.HeaderType_CONFIG_UPDATE,
		channel,
nil, //本地签名者
		&common.ConfigUpdateEnvelope{ConfigUpdate: utils.MarshalOrPanic(configUpdate)},
0, //消息版本
0, //时代
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(signedEnvelope).NotTo(BeNil())

	updateFile := filepath.Join(tempDir, "update.pb")
	err = ioutil.WriteFile(updateFile, utils.MarshalOrPanic(signedEnvelope), 0600)
	Expect(err).NotTo(HaveOccurred())

	for _, signer := range additionalSigners {
		sess, err := n.OrdererAdminSession(signer, submitter, commands.SignConfigTx{File: updateFile})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}

//获取当前配置块编号
	currentBlockNumber := CurrentConfigBlockNumber(n, submitter, orderer, channel)

	sess, err := n.PeerAdminSession(submitter, commands.ChannelUpdate{
		ChannelID: channel,
		Orderer:   n.OrdererAddress(orderer, ListenPort),
		File:      updateFile,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Successfully submitted channel update"))

//等待提交块
	ccb := func() uint64 { return CurrentConfigBlockNumber(n, submitter, orderer, channel) }
	Eventually(ccb, n.EventuallyTimeout).Should(BeNumerically(">", currentBlockNumber))
}

//CurrentConfigBlockNumber从
//当前配置块。这可用于检测Whena配置更改
//已经完成。
func CurrentConfigBlockNumber(n *Network, peer *Peer, orderer *Orderer, channel string) uint64 {
	tempDir, err := ioutil.TempDir("", "currentConfigBlock")
	Expect(err).NotTo(HaveOccurred())
	defer os.RemoveAll(tempDir)

//获取配置块
	output := filepath.Join(tempDir, "config_block.pb")
	sess, err := n.PeerAdminSession(peer, commands.ChannelFetch{
		ChannelID:  channel,
		Block:      "config",
		Orderer:    n.OrdererAddress(orderer, ListenPort),
		OutputFile: output,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Received block: "))

//取消配置块字节的标记
	configBlock := UnmarshalBlockFromFile(output)
	return configBlock.Header.Number
}

//unmashalblockfromfile从文件中取消对原型编码块的标记。
func UnmarshalBlockFromFile(blockFile string) *common.Block {
	blockBytes, err := ioutil.ReadFile(blockFile)
	Expect(err).NotTo(HaveOccurred())

	block, err := utils.UnmarshalBlock(blockBytes)
	Expect(err).NotTo(HaveOccurred())

	return block
}
