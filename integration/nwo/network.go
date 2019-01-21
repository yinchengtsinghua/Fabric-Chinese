
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


package nwo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration/helpers"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	"github.com/hyperledger/fabric/integration/runner"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
	"gopkg.in/yaml.v2"
)

//组织为组织的信息建模。它包括
//用密码填充MSP所需的信息。
type Organization struct {
	MSPID         string `yaml:"msp_id,omitempty"`
	Name          string `yaml:"name,omitempty"`
	Domain        string `yaml:"domain,omitempty"`
	EnableNodeOUs bool   `yaml:"enable_node_organizational_units"`
	Users         int    `yaml:"users,omitempty"`
	CA            *CA    `yaml:"ca,omitempty"`
}

type CA struct {
	Hostname string `yaml:"hostname,omitempty"`
}

//联合体是一个命名的组织集合。它用于填充
//订购方生成块配置文件。
type Consortium struct {
	Name          string   `yaml:"name,omitempty"`
	Organizations []string `yaml:"organizations,omitempty"`
}

//共识表明订购者类型以及经纪人和动物园管理员的数量
//实例。
type Consensus struct {
	Type       string `yaml:"type,omitempty"`
	Brokers    int    `yaml:"brokers,omitempty"`
	ZooKeepers int    `yaml:"zookeepers,omitempty"`
}

//SystemChannel声明网络系统通道的名称及其
//关联的ConfigTxGen配置文件名。
type SystemChannel struct {
	Name    string `yaml:"name,omitempty"`
	Profile string `yaml:"profile,omitempty"`
}

//通道将通道名与configtxgen配置文件名关联。
type Channel struct {
	Name    string `yaml:"name,omitempty"`
	Profile string `yaml:"profile,omitempty"`
}

//医嘱者定义医嘱者实例及其所属组织。
type Orderer struct {
	Name         string `yaml:"name,omitempty"`
	Organization string `yaml:"organization,omitempty"`
}

//ID为医嘱者实例提供唯一标识符。
func (o Orderer) ID() string {
	return fmt.Sprintf("%s.%s", o.Organization, o.Name)
}

//对等机定义对等机实例、所属组织以及
//同行应该加入的渠道。
type Peer struct {
	Name         string         `yaml:"name,omitempty"`
	Organization string         `yaml:"organization,omitempty"`
	Channels     []*PeerChannel `yaml:"channels,omitempty"`
}

//对等通道应加入的通道的对等通道名称，以及
//不是对等端应该是通道的锚。
type PeerChannel struct {
	Name   string `yaml:"name,omitempty"`
	Anchor bool   `yaml:"anchor"`
}

//ID为对等实例提供唯一标识符。
func (p *Peer) ID() string {
	return fmt.Sprintf("%s.%s", p.Organization, p.Name)
}

//如果此对等端是其加入的任何通道的锚，则anchor返回true。
func (p *Peer) Anchor() bool {
	for _, c := range p.Channels {
		if c.Anchor {
			return true
		}
	}
	return false
}

//概要文件封装了configtxgen概要文件的基本信息。
type Profile struct {
	Name          string   `yaml:"name,omitempty"`
	Orderers      []string `yaml:"orderers,omitempty"`
	Consortium    string   `yaml:"consortium,omitempty"`
	Organizations []string `yaml:"organizations,omitempty"`
}

//网络包含有关结构网络的信息。
type Network struct {
	RootDir           string
	StartPort         uint16
	Components        *Components
	DockerClient      *docker.Client
	NetworkID         string
	EventuallyTimeout time.Duration
	MetricsProvider   string
	StatsdEndpoint    string

	PortsByBrokerID  map[string]Ports
	PortsByOrdererID map[string]Ports
	PortsByPeerID    map[string]Ports
	Organizations    []*Organization
	SystemChannel    *SystemChannel
	Channels         []*Channel
	Consensus        *Consensus
	Orderers         []*Orderer
	Peers            []*Peer
	Profiles         []*Profile
	Consortiums      []*Consortium
	Templates        *Templates

	colorIndex uint
}

//新建从简单配置创建网络。所有生成或管理的
//网络的项目将位于rootdir下。端口将是
//从指定的起始端口按顺序分配。
func New(c *Config, rootDir string, client *docker.Client, startPort int, components *Components) *Network {
	network := &Network{
		StartPort:    uint16(startPort),
		RootDir:      rootDir,
		Components:   components,
		DockerClient: client,

		NetworkID:         helpers.UniqueName(),
		EventuallyTimeout: time.Minute,
		MetricsProvider:   "prometheus",
		PortsByBrokerID:   map[string]Ports{},
		PortsByOrdererID:  map[string]Ports{},
		PortsByPeerID:     map[string]Ports{},

		Organizations: c.Organizations,
		Consensus:     c.Consensus,
		Orderers:      c.Orderers,
		Peers:         c.Peers,
		SystemChannel: c.SystemChannel,
		Channels:      c.Channels,
		Profiles:      c.Profiles,
		Consortiums:   c.Consortiums,
		Templates:     c.Templates,
	}

	if network.Templates == nil {
		network.Templates = &Templates{}
	}

	for i := 0; i < network.Consensus.Brokers; i++ {
		ports := Ports{}
		for _, portName := range BrokerPortNames() {
			ports[portName] = network.ReservePort()
		}
		network.PortsByBrokerID[strconv.Itoa(i)] = ports
	}

	for _, o := range c.Orderers {
		ports := Ports{}
		for _, portName := range OrdererPortNames() {
			ports[portName] = network.ReservePort()
		}
		network.PortsByOrdererID[o.ID()] = ports
	}

	for _, p := range c.Peers {
		ports := Ports{}
		for _, portName := range PeerPortNames() {
			ports[portName] = network.ReservePort()
		}
		network.PortsByPeerID[p.ID()] = ports
	}
	return network
}

//configtxpath返回生成的configtxgen配置的路径
//文件。
func (n *Network) ConfigTxConfigPath() string {
	return filepath.Join(n.RootDir, "configtx.yaml")
}

//cryptoPath返回到cryptogen将放置其
//生成的工件。
func (n *Network) CryptoPath() string {
	return filepath.Join(n.RootDir, "crypto")
}

//CryptoConfigPath返回生成的加密配置的路径
//文件。
func (n *Network) CryptoConfigPath() string {
	return filepath.Join(n.RootDir, "crypto-config.yaml")
}

//OutputBlockPath返回命名系统的Genesis块的路径
//通道。
func (n *Network) OutputBlockPath(channelName string) string {
	return filepath.Join(n.RootDir, fmt.Sprintf("%s_block.pb", channelName))
}

//createChannelTxPath返回的创建通道事务的路径
//指定的频道。
func (n *Network) CreateChannelTxPath(channelName string) string {
	return filepath.Join(n.RootDir, fmt.Sprintf("%s_tx.pb", channelName))
}

//orderdir返回指定的配置目录的路径
//订购者。
func (n *Network) OrdererDir(o *Orderer) string {
	return filepath.Join(n.RootDir, "orderers", o.ID())
}

//orderconfigpath返回的order配置文档的路径
//指定的订购者。
func (n *Network) OrdererConfigPath(o *Orderer) string {
	return filepath.Join(n.OrdererDir(o), "orderer.yaml")
}

//readerderconfig取消排序器的order.yaml并返回
//接近其内容的对象。
func (n *Network) ReadOrdererConfig(o *Orderer) *fabricconfig.Orderer {
	var orderer fabricconfig.Orderer
	ordererBytes, err := ioutil.ReadFile(n.OrdererConfigPath(o))
	Expect(err).NotTo(HaveOccurred())

	err = yaml.Unmarshal(ordererBytes, &orderer)
	Expect(err).NotTo(HaveOccurred())

	return &orderer
}

//WriteOrdererConfig将提供的配置序列化为指定的
//订购方的订购方.yaml文档。
func (n *Network) WriteOrdererConfig(o *Orderer, config *fabricconfig.Orderer) {
	ordererBytes, err := yaml.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(n.OrdererConfigPath(o), ordererBytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}

//peerdir返回指定的配置目录的路径
//同龄人。
func (n *Network) PeerDir(p *Peer) string {
	return filepath.Join(n.RootDir, "peers", p.ID())
}

//peerconfigPath返回的对等配置文档的路径
//指定的对等体。
func (n *Network) PeerConfigPath(p *Peer) string {
	return filepath.Join(n.PeerDir(p), "core.yaml")
}

//readpeerconfig取消标记对等机的core.yaml并返回一个对象
//近似其内容。
func (n *Network) ReadPeerConfig(p *Peer) *fabricconfig.Core {
	var core fabricconfig.Core
	coreBytes, err := ioutil.ReadFile(n.PeerConfigPath(p))
	Expect(err).NotTo(HaveOccurred())

	err = yaml.Unmarshal(coreBytes, &core)
	Expect(err).NotTo(HaveOccurred())

	return &core
}

//WritePeerConfig将提供的配置序列化为指定的
//Peer的core.yaml文档。
func (n *Network) WritePeerConfig(p *Peer, config *fabricconfig.Core) {
	coreBytes, err := yaml.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(n.PeerConfigPath(p), coreBytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}

//peerUserCryptoDir返回包含
//对等机的指定用户的证书和密钥。
func (n *Network) peerUserCryptoDir(p *Peer, user, cryptoMaterialType string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return n.userCryptoDir(org, "peerOrganizations", user, cryptoMaterialType)
}

//orderUserCryptoDir返回包含
//订购方指定用户的证书和密钥。
func (n *Network) ordererUserCryptoDir(o *Orderer, user, cryptoMaterialType string) string {
	org := n.Organization(o.Organization)
	Expect(org).NotTo(BeNil())

	return n.userCryptoDir(org, "ordererOrganizations", user, cryptoMaterialType)
}

//usercryptodir返回具有对等组织或订购方组织的加密材料的文件夹路径
//特定用户
func (n *Network) userCryptoDir(org *Organization, nodeOrganizationType, user, cryptoMaterialType string) string {
	return filepath.Join(
		n.RootDir,
		"crypto",
		nodeOrganizationType,
		org.Domain,
		"users",
		fmt.Sprintf("%s@%s", user, org.Domain),
		cryptoMaterialType,
	)
}

//peerusermspdir返回包含
//对等机的指定用户的证书和密钥。
func (n *Network) PeerUserMSPDir(p *Peer, user string) string {
	return n.peerUserCryptoDir(p, user, "msp")
}

//orderUserMSPDir返回包含
//对等机的指定用户的证书和密钥。
func (n *Network) OrdererUserMSPDir(o *Orderer, user string) string {
	return n.ordererUserCryptoDir(o, user, "msp")
}

//peerusertlsdir返回包含
//对等机的指定用户的证书和密钥。
func (n *Network) PeerUserTLSDir(p *Peer, user string) string {
	return n.peerUserCryptoDir(p, user, "tls")
}

//peerusercert返回中指定用户的证书路径
//对等组织。
func (n *Network) PeerUserCert(p *Peer, user string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.PeerUserMSPDir(p, user),
		"signcerts",
		fmt.Sprintf("%s@%s-cert.pem", user, org.Domain),
	)
}

//PeerUserKey返回中指定用户的私钥路径。
//对等组织。
func (n *Network) PeerUserKey(p *Peer, user string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	keystore := filepath.Join(
		n.PeerUserMSPDir(p, user),
		"keystore",
	)

//文件名是ski和非确定性的
	keys, err := ioutil.ReadDir(keystore)
	Expect(err).NotTo(HaveOccurred())
	Expect(keys).To(HaveLen(1))

	return filepath.Join(keystore, keys[0].Name())
}

//peerLocalCryptoDir返回对等机的本地加密目录的路径。
func (n *Network) peerLocalCryptoDir(p *Peer, cryptoType string) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.RootDir,
		"crypto",
		"peerOrganizations",
		org.Domain,
		"peers",
		fmt.Sprintf("%s.%s", p.Name, org.Domain),
		cryptoType,
	)
}

//peerlocalmspdir返回对等机的本地msp目录的路径。
func (n *Network) PeerLocalMSPDir(p *Peer) string {
	return n.peerLocalCryptoDir(p, "msp")
}

//peerlocaltlsdir返回对等机的本地tls目录的路径。
func (n *Network) PeerLocalTLSDir(p *Peer) string {
	return n.peerLocalCryptoDir(p, "tls")
}

//peercert返回对等方证书的路径。
func (n *Network) PeerCert(p *Peer) string {
	org := n.Organization(p.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.PeerLocalMSPDir(p),
		"signcerts",
		fmt.Sprintf("%s.%s-cert.pem", p.Name, org.Domain),
	)
}

//peerorgmspdir返回对等组织的msp目录的路径。
func (n *Network) PeerOrgMSPDir(org *Organization) string {
	return filepath.Join(
		n.RootDir,
		"crypto",
		"peerOrganizations",
		org.Domain,
		"msp",
	)
}

//orderorgmspdir返回排序器的msp目录的路径
//组织。
func (n *Network) OrdererOrgMSPDir(o *Organization) string {
	return filepath.Join(
		n.RootDir,
		"crypto",
		"ordererOrganizations",
		o.Domain,
		"msp",
	)
}

//orderLocalRyptoDir返回的本地加密目录的路径
//订购者。
func (n *Network) OrdererLocalCryptoDir(o *Orderer, cryptoType string) string {
	org := n.Organization(o.Organization)
	Expect(org).NotTo(BeNil())

	return filepath.Join(
		n.RootDir,
		"crypto",
		"ordererOrganizations",
		org.Domain,
		"orderers",
		fmt.Sprintf("%s.%s", o.Name, org.Domain),
		cryptoType,
	)
}

//orderlocalmspdir返回的本地msp目录的路径
//订购者。
func (n *Network) OrdererLocalMSPDir(o *Orderer) string {
	return n.OrdererLocalCryptoDir(o, "msp")
}

//orderLocalLSDir返回本地tls目录的路径
//订购者。
func (n *Network) OrdererLocalTLSDir(o *Orderer) string {
	return n.OrdererLocalCryptoDir(o, "tls")
}

//profileforchannel获取与
//指定通道。
func (n *Network) ProfileForChannel(channelName string) string {
	for _, ch := range n.Channels {
		if ch.Name == channelName {
			return ch.Profile
		}
	}
	return ""
}

//cacertsbundlepath返回的CA证书束的路径
//网络。当连接到对等机时使用此捆绑包。
func (n *Network) CACertsBundlePath() string {
	return filepath.Join(
		n.RootDir,
		"crypto",
		"ca-certs.pem",
	)
}

//GenerateConfigTree生成需要的配置文档
//引导一个结构网络。将为生成配置文件
//Cryptogen、configtxgen，以及每个对等机和排序器。的内容
//文档将基于用于创建网络的配置。
//
//当此方法完成时，生成的树将类似于
//这是：
//
//$rootdir/configtx.yaml
//$rootdir/crypto-config.yaml
//$rootdir/orderers/order0.order-org/orderer.yaml
//$rootdir/peers/peer0.org1/core.yaml
//$rootdir/peers/peer0.org2/core.yaml
//$rootdir/peers/peer1.org1/core.yaml
//$rootdir/peers/peer1.org2/core.yaml
//
func (n *Network) GenerateConfigTree() {
	n.GenerateCryptoConfig()
	n.GenerateConfigTxConfig()
	for _, o := range n.Orderers {
		n.GenerateOrdererConfig(o)
	}
	for _, p := range n.Peers {
		n.GenerateCoreConfig(p)
	}
}

//引导程序生成加密材料、订购程序系统通道
//并创建运行结构所需的通道事务
//网络。
//
//密码工具用于从
//$rootdir/crypto-config.yaml。生成的工件将放置在
//$rootdir/加密/…
//
//gensis块是从
//systemchannel.profile属性。块被写入
//$rootdir/$systemchannel.name u block.pb。
//
//为引用的每个通道生成创建通道事务
//使用频道的配置文件属性的网络。交易是
//写入$rootdir/$channel.name u tx.pb。
func (n *Network) Bootstrap() {
	_, err := n.DockerClient.CreateNetwork(
		docker.CreateNetworkOptions{
			Name:   n.NetworkID,
			Driver: "bridge",
		},
	)
	Expect(err).NotTo(HaveOccurred())

	sess, err := n.Cryptogen(commands.Generate{
		Config: n.CryptoConfigPath(),
		Output: n.CryptoPath(),
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

	sess, err = n.ConfigTxGen(commands.OutputBlock{
		ChannelID:   n.SystemChannel.Name,
		Profile:     n.SystemChannel.Profile,
		ConfigPath:  n.RootDir,
		OutputBlock: n.OutputBlockPath(n.SystemChannel.Name),
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

	for _, c := range n.Channels {
		sess, err := n.ConfigTxGen(commands.CreateChannelTx{
			ChannelID:             c.Name,
			Profile:               c.Profile,
			ConfigPath:            n.RootDir,
			OutputCreateChannelTx: n.CreateChannelTxPath(c.Name),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}

	n.concatenateTLSCACertificates()
}

//concatenatelscacertificates将所有TLS CA证书连接到
//对等客户端要使用的单个文件。
func (n *Network) concatenateTLSCACertificates() {
	bundle := &bytes.Buffer{}
	for _, tlsCertPath := range n.listTLSCACertificates() {
		certBytes, err := ioutil.ReadFile(tlsCertPath)
		Expect(err).NotTo(HaveOccurred())
		bundle.Write(certBytes)
	}
	err := ioutil.WriteFile(n.CACertsBundlePath(), bundle.Bytes(), 0660)
	Expect(err).NotTo(HaveOccurred())
}

//listlscacertificates返回中所有TLS CA证书的路径
//网络，跨所有组织。
func (n *Network) listTLSCACertificates() []string {
	fileName2Path := make(map[string]string)
	filepath.Walk(filepath.Join(n.RootDir, "crypto"), func(path string, info os.FileInfo, err error) error {
//文件以“tlsca”开头，其中包含“-cert.pem”
		if strings.HasPrefix(info.Name(), "tlsca") && strings.Contains(info.Name(), "-cert.pem") {
			fileName2Path[info.Name()] = path
		}
		return nil
	})

	var tlsCACertificates []string
	for _, path := range fileName2Path {
		tlsCACertificates = append(tlsCACertificates, path)
	}
	return tlsCACertificates
}

//清理尝试清理Docker相关的工件，可能
//已由网络创建。
func (n *Network) Cleanup() {
	nw, err := n.DockerClient.NetworkInfo(n.NetworkID)
	Expect(err).NotTo(HaveOccurred())

	err = n.DockerClient.RemoveNetwork(nw.ID)
	Expect(err).NotTo(HaveOccurred())

	containers, err := n.DockerClient.ListContainers(docker.ListContainersOptions{All: true})
	Expect(err).NotTo(HaveOccurred())
	for _, c := range containers {
		for _, name := range c.Names {
			if strings.HasPrefix(name, "/"+n.NetworkID) {
				err := n.DockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: c.ID, Force: true})
				Expect(err).NotTo(HaveOccurred())
				break
			}
		}
	}

	images, err := n.DockerClient.ListImages(docker.ListImagesOptions{All: true})
	Expect(err).NotTo(HaveOccurred())
	for _, i := range images {
		for _, tag := range i.RepoTags {
			if strings.HasPrefix(tag, n.NetworkID) {
				err := n.DockerClient.RemoveImage(i.ID)
				Expect(err).NotTo(HaveOccurred())
				break
			}
		}
	}
}

//CreateAndJoinChannels将创建配置中指定的所有通道，
//被同行引用。然后，引用对等点将加入到
//通道（S）。
//
//必须先运行网络，然后才能调用此网络。
func (n *Network) CreateAndJoinChannels(o *Orderer) {
	for _, c := range n.Channels {
		n.CreateAndJoinChannel(o, c.Name)
	}
}

//CreateAndJoinChannel将创建指定的通道。引用
//然后，对等端将加入通道。
//
//必须先运行网络，然后才能调用此网络。
func (n *Network) CreateAndJoinChannel(o *Orderer, channelName string) {
	peers := n.PeersWithChannel(channelName)
	if len(peers) == 0 {
		return
	}

	n.CreateChannel(channelName, o, peers[0])
	n.JoinChannel(channelName, o, peers...)
}

//updateChannelAnchors确定指定通道的定位点对等点，
//为每个组织创建一个锚定对等更新事务，并提交
//向医嘱者更新交易记录。
func (n *Network) UpdateChannelAnchors(o *Orderer, channelName string) {
	tempFile, err := ioutil.TempFile("", "update-anchors")
	Expect(err).NotTo(HaveOccurred())
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	peersByOrg := map[string]*Peer{}
	for _, p := range n.AnchorsForChannel(channelName) {
		peersByOrg[p.Organization] = p
	}

	for orgName, p := range peersByOrg {
		anchorUpdate := commands.OutputAnchorPeersUpdate{
			OutputAnchorPeersUpdate: tempFile.Name(),
			ChannelID:               channelName,
			Profile:                 n.ProfileForChannel(channelName),
			ConfigPath:              n.RootDir,
			AsOrg:                   orgName,
		}
		sess, err := n.ConfigTxGen(anchorUpdate)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

		sess, err = n.PeerAdminSession(p, commands.ChannelUpdate{
			ChannelID: channelName,
			Orderer:   n.OrdererAddress(o, ListenPort),
			File:      tempFile.Name(),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}
}

//CreateChannel将向
//指定的订购者。渠道交易必须存在于该位置
//由CreateChannelTxPath返回。
//
//调用此命令时，订购方必须正在运行。
func (n *Network) CreateChannel(channelName string, o *Orderer, p *Peer) {
	createChannel := func() int {
		sess, err := n.PeerAdminSession(p, commands.ChannelCreate{
			ChannelID:   channelName,
			Orderer:     n.OrdererAddress(o, ListenPort),
			File:        n.CreateChannelTxPath(channelName),
			OutputBlock: "/dev/null",
		})
		Expect(err).NotTo(HaveOccurred())
		return sess.Wait(n.EventuallyTimeout).ExitCode()
	}

	Eventually(createChannel, n.EventuallyTimeout).Should(Equal(0))
}

//JoinChannel将把对等端加入指定的通道。订购方用于
//获取通道的当前配置块。
//
//在调用之前，必须运行医嘱者和列出的对等方。
func (n *Network) JoinChannel(name string, o *Orderer, peers ...*Peer) {
	if len(peers) == 0 {
		return
	}

	tempFile, err := ioutil.TempFile("", "genesis-block")
	Expect(err).NotTo(HaveOccurred())
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	sess, err := n.PeerAdminSession(peers[0], commands.ChannelFetch{
		Block:      "0",
		ChannelID:  name,
		Orderer:    n.OrdererAddress(o, ListenPort),
		OutputFile: tempFile.Name(),
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))

	for _, p := range peers {
		sess, err := n.PeerAdminSession(p, commands.ChannelJoin{
			BlockPath: tempFile.Name(),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, n.EventuallyTimeout).Should(gexec.Exit(0))
	}
}

//cryptogen为提供的cryptogen命令启动gexec.session。
func (n *Network) Cryptogen(command Command) (*gexec.Session, error) {
	cmd := NewCommand(n.Components.Cryptogen(), command)
	return n.StartSession(cmd, command.SessionName())
}

//configtxgen为提供的configtxgen命令启动gexec.session。
func (n *Network) ConfigTxGen(command Command) (*gexec.Session, error) {
	cmd := NewCommand(n.Components.ConfigTxGen(), command)
	return n.StartSession(cmd, command.SessionName())
}

//discover为提供的discover命令启动gexec.session。
func (n *Network) Discover(command Command) (*gexec.Session, error) {
	cmd := NewCommand(n.Components.Discover(), command)
	cmd.Args = append(cmd.Args, "--peerTLSCA", n.CACertsBundlePath())
	return n.StartSession(cmd, command.SessionName())
}

//ZooKeeperRunner返回ZooKeeper实例的运行程序。
func (n *Network) ZooKeeperRunner(idx int) *runner.ZooKeeper {
	colorCode := n.nextColor()
	name := fmt.Sprintf("zookeeper-%d-%s", idx, n.NetworkID)

	return &runner.ZooKeeper{
ZooMyID:     idx + 1, //ID必须介于1和255之间
		Client:      n.DockerClient,
		Name:        name,
		NetworkName: n.NetworkID,
		OutputStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
		ErrorStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
	}
}

func (n *Network) minBrokersInSync() int {
	if n.Consensus.Brokers < 2 {
		return n.Consensus.Brokers
	}
	return 2
}

func (n *Network) defaultBrokerReplication() int {
	if n.Consensus.Brokers < 3 {
		return n.Consensus.Brokers
	}
	return 3
}

//broker runner返回kafka代理实例的运行程序。
func (n *Network) BrokerRunner(id int, zookeepers []string) *runner.Kafka {
	colorCode := n.nextColor()
	name := fmt.Sprintf("kafka-%d-%s", id, n.NetworkID)

	return &runner.Kafka{
		BrokerID:                 id + 1,
		Client:                   n.DockerClient,
		AdvertisedListeners:      "127.0.0.1",
		HostPort:                 int(n.PortsByBrokerID[strconv.Itoa(id)][HostPort]),
		Name:                     name,
		NetworkName:              n.NetworkID,
		MinInsyncReplicas:        n.minBrokersInSync(),
		DefaultReplicationFactor: n.defaultBrokerReplication(),
		ZooKeeperConnect:         strings.Join(zookeepers, ","),
		OutputStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
		ErrorStream: gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", colorCode, name),
			ginkgo.GinkgoWriter,
		),
	}
}

//brokerGrouprunner返回一个管理组成的进程的运行程序
//卡夫卡织物经纪人网络。
func (n *Network) BrokerGroupRunner() ifrit.Runner {
	members := grouper.Members{}
	zookeepers := []string{}

	for i := 0; i < n.Consensus.ZooKeepers; i++ {
		zk := n.ZooKeeperRunner(i)
		zookeepers = append(zookeepers, fmt.Sprintf("%s:2181", zk.Name))
		members = append(members, grouper.Member{Name: zk.Name, Runner: zk})
	}

	for i := 0; i < n.Consensus.Brokers; i++ {
		kafka := n.BrokerRunner(i, zookeepers)
		members = append(members, grouper.Member{Name: kafka.Name, Runner: kafka})
	}

	return grouper.NewOrdered(syscall.SIGTERM, members)
}

//orderrunner返回指定医嘱者的ifrit.runner。赛跑者
//可用于启动和管理医嘱流程。
func (n *Network) OrdererRunner(o *Orderer) *ginkgomon.Runner {
	cmd := exec.Command(n.Components.Orderer())
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABRIC_CFG_PATH=%s", n.OrdererDir(o)))

	config := ginkgomon.Config{
		AnsiColorCode:     n.nextColor(),
		Name:              o.ID(),
		Command:           cmd,
		StartCheck:        "Beginning to serve requests",
		StartCheckTimeout: 15 * time.Second,
	}

	if n.Consensus.Brokers != 0 {
		config.StartCheck = "Start phase completed successfully"
		config.StartCheckTimeout = 30 * time.Second
	}

	return ginkgomon.New(config)
}

//ordergrouprunner返回可用于启动和停止所有操作的运行程序
//网络中的订购者。
func (n *Network) OrdererGroupRunner() ifrit.Runner {
	members := grouper.Members{}
	for _, o := range n.Orderers {
		members = append(members, grouper.Member{Name: o.ID(), Runner: n.OrdererRunner(o)})
	}
	return grouper.NewParallel(syscall.SIGTERM, members)
}

//PeerRunner返回指定对等机的ifrit.runner。跑步者可以
//用于启动和管理对等进程。
func (n *Network) PeerRunner(p *Peer) *ginkgomon.Runner {
	cmd := n.peerCommand(
		commands.NodeStart{PeerID: p.ID()},
		fmt.Sprintf("FABRIC_CFG_PATH=%s", n.PeerDir(p)),
	)

	return ginkgomon.New(ginkgomon.Config{
		AnsiColorCode:     n.nextColor(),
		Name:              p.ID(),
		Command:           cmd,
		StartCheck:        `Started peer with ID=.*, .*, address=`,
		StartCheckTimeout: 15 * time.Second,
	})
}

//PeerGroupRunner返回可用于启动和停止所有
//网络中的对等点。
func (n *Network) PeerGroupRunner() ifrit.Runner {
	members := grouper.Members{}
	for _, p := range n.Peers {
		members = append(members, grouper.Member{Name: p.ID(), Runner: n.PeerRunner(p)})
	}
	return grouper.NewParallel(syscall.SIGTERM, members)
}

//NetworkGroupRunner返回可用于启动和停止
//整个织物网络。
func (n *Network) NetworkGroupRunner() ifrit.Runner {
	members := grouper.Members{
		{Name: "brokers", Runner: n.BrokerGroupRunner()},
		{Name: "orderers", Runner: n.OrdererGroupRunner()},
		{Name: "peers", Runner: n.PeerGroupRunner()},
	}
	return grouper.NewOrdered(syscall.SIGTERM, members)
}

func (n *Network) peerCommand(command Command, env ...string) *exec.Cmd {
	cmd := NewCommand(n.Components.Peer(), command)
	cmd.Env = append(cmd.Env, env...)
	if ConnectsToOrderer(command) {
		cmd.Args = append(cmd.Args, "--tls")
		cmd.Args = append(cmd.Args, "--cafile", n.CACertsBundlePath())
	}

//如果我们有一个具有多个证书的对等调用，
//我们需要模拟正确的peer cli用法，
//所以我们计算——对等地址用法的数量
//我们有并添加了相同的（连接的TLS CA证书文件）
//绕过对等客户端健全性检查的次数相同
	requiredPeerAddresses := flagCount("--peerAddresses", cmd.Args)
	for i := 0; i < requiredPeerAddresses; i++ {
		cmd.Args = append(cmd.Args, "--tlsRootCertFiles")
		cmd.Args = append(cmd.Args, n.CACertsBundlePath())
	}
	return cmd
}

func flagCount(flag string, args []string) int {
	var c int
	for _, arg := range args {
		if arg == flag {
			c++
		}
	}
	return c
}

//PeerAdminSession作为提供的对等管理启动gexec.session。
//对等命令。这可供短时间运行的peer cli命令使用。
//在对等配置上下文中执行的。
func (n *Network) PeerAdminSession(p *Peer, command Command) (*gexec.Session, error) {
	return n.PeerUserSession(p, "Admin", command)
}

//PeerUserSession作为所提供对等机的对等用户启动gexec.session
//命令。这将由短时间运行的peer cli命令使用，这些命令
//在对等配置的上下文中执行。
func (n *Network) PeerUserSession(p *Peer, user string, command Command) (*gexec.Session, error) {
	cmd := n.peerCommand(
		command,
		fmt.Sprintf("FABRIC_CFG_PATH=%s", n.PeerDir(p)),
		fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=%s", n.PeerUserMSPDir(p, user)),
	)
	return n.StartSession(cmd, command.SessionName())
}

//医嘱管理会话作为医嘱者节点管理用户执行gexec.session。主要用于
//生成医嘱者配置更新
func (n *Network) OrdererAdminSession(o *Orderer, p *Peer, command Command) (*gexec.Session, error) {
	cmd := n.peerCommand(
		command,
		"CORE_PEER_LOCALMSPID=OrdererMSP",
		fmt.Sprintf("FABRIC_CFG_PATH=%s", n.PeerDir(p)),
		fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=%s", n.OrdererUserMSPDir(o, "Admin")),
	)
	return n.StartSession(cmd, command.SessionName())
}

//Peer返回有关命名组织中命名对等机的信息。
func (n *Network) Peer(orgName, peerName string) *Peer {
	for _, p := range n.PeersInOrg(orgName) {
		if p.Name == peerName {
			return p
		}
	}
	return nil
}

//函数从作为参数传递的对等端和链码创建新的发现的对等端。
func (n *Network) DiscoveredPeer(p *Peer, chaincodes ...string) DiscoveredPeer {
	peerCert, err := ioutil.ReadFile(n.PeerCert(p))
	Expect(err).NotTo(HaveOccurred())

	return DiscoveredPeer{
		MSPID:      n.Organization(p.Organization).MSPID,
		Endpoint:   fmt.Sprintf("127.0.0.1:%d", n.PeerPort(p, ListenPort)),
		Identity:   string(peerCert),
		Chaincodes: chaincodes,
	}
}

//排序器返回有关命名排序器的信息。
func (n *Network) Orderer(name string) *Orderer {
	for _, o := range n.Orderers {
		if o.Name == name {
			return o
		}
	}
	return nil
}

//组织返回有关命名组织的信息。
func (n *Network) Organization(orgName string) *Organization {
	for _, org := range n.Organizations {
		if org.Name == orgName {
			return org
		}
	}
	return nil
}

//联合体返回有关指定联合体的信息。
func (n *Network) Consortium(name string) *Consortium {
	for _, c := range n.Consortiums {
		if c.Name == name {
			return c
		}
	}
	return nil
}

//PeerOrgs返回与至少一个对等方关联的所有组织。
func (n *Network) PeerOrgs() []*Organization {
	orgsByName := map[string]*Organization{}
	for _, p := range n.Peers {
		orgsByName[p.Organization] = n.Organization(p.Organization)
	}

	orgs := []*Organization{}
	for _, org := range orgsByName {
		orgs = append(orgs, org)
	}
	return orgs
}

//peerswithchannel返回加入已命名的
//通道。
func (n *Network) PeersWithChannel(chanName string) []*Peer {
	peers := []*Peer{}
	for _, p := range n.Peers {
		for _, c := range p.Channels {
			if c.Name == chanName {
				peers = append(peers, p)
			}
		}
	}
	return peers
}

//AnchorsForchannel返回作为
//命名通道。
func (n *Network) AnchorsForChannel(chanName string) []*Peer {
	anchors := []*Peer{}
	for _, p := range n.Peers {
		for _, pc := range p.Channels {
			if pc.Name == chanName && pc.Anchor {
				anchors = append(anchors, p)
			}
		}
	}
	return anchors
}

//anchorsingorg返回作为至少一个频道锚定的所有对等端
//在指定的组织中。
func (n *Network) AnchorsInOrg(orgName string) []*Peer {
	anchors := []*Peer{}
	for _, p := range n.PeersInOrg(orgName) {
		if p.Anchor() {
			anchors = append(anchors, p)
			break
		}
	}

//没有明确的锚定意味着所有对等方都是锚定。
	if len(anchors) == 0 {
		anchors = n.PeersInOrg(orgName)
	}

	return anchors
}

//orderSinorg返回由命名的Organiztion拥有的所有order实例。
func (n *Network) OrderersInOrg(orgName string) []*Orderer {
	orderers := []*Orderer{}
	for _, o := range n.Orderers {
		if o.Organization == orgName {
			orderers = append(orderers, o)
		}
	}
	return orderers
}

//ORGSForderers返回拥有至少一个
//指定的订购者。
func (n *Network) OrgsForOrderers(ordererNames []string) []*Organization {
	orgsByName := map[string]*Organization{}
	for _, name := range ordererNames {
		orgName := n.Orderer(name).Organization
		orgsByName[orgName] = n.Organization(orgName)
	}
	orgs := []*Organization{}
	for _, org := range orgsByName {
		orgs = append(orgs, org)
	}
	return orgs
}

//orderorgs返回至少拥有一个组织实例的所有组织实例
//订购者。
func (n *Network) OrdererOrgs() []*Organization {
	orgsByName := map[string]*Organization{}
	for _, o := range n.Orderers {
		orgsByName[o.Organization] = n.Organization(o.Organization)
	}

	orgs := []*Organization{}
	for _, org := range orgsByName {
		orgs = append(orgs, org)
	}
	return orgs
}

//PeerSinorg返回由指定的
//组织。
func (n *Network) PeersInOrg(orgName string) []*Peer {
	peers := []*Peer{}
	for _, o := range n.Peers {
		if o.Organization == orgName {
			peers = append(peers, o)
		}
	}
	return peers
}

//reserveport分配下一个可用端口。
func (n *Network) ReservePort() uint16 {
	n.StartPort++
	return n.StartPort - 1
}

type PortName string
type Ports map[PortName]uint16

const (
	ChaincodePort  PortName = "Chaincode"
	EventsPort     PortName = "Events"
	HostPort       PortName = "HostPort"
	ListenPort     PortName = "Listen"
	ProfilePort    PortName = "Profile"
	OperationsPort PortName = "Operations"
)

//PeerPortNames返回需要为对等机保留的端口列表。
func PeerPortNames() []PortName {
	return []PortName{ListenPort, ChaincodePort, EventsPort, ProfilePort, OperationsPort}
}

//orderportname返回需要为
//订购者。
func OrdererPortNames() []PortName {
	return []PortName{ListenPort, ProfilePort, OperationsPort}
}

//brokerPortNames返回需要为
//卡夫卡经纪人。
func BrokerPortNames() []PortName {
	return []PortName{HostPort}
}

//brokeraddresss返回网络的代理地址列表。
func (n *Network) BrokerAddresses(portName PortName) []string {
	addresses := []string{}
	for _, ports := range n.PortsByBrokerID {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:%d", ports[portName]))
	}
	return addresses
}

//orderAddress返回排序器公开的地址（主机和端口）
//用于指定端口。命令行工具在
//连接到订购方。
//
//这假定订购方正在侦听0.0.0.0或127.0.0.1，并且
//在环回地址上可用。
func (n *Network) OrdererAddress(o *Orderer, portName PortName) string {
	return fmt.Sprintf("127.0.0.1:%d", n.OrdererPort(o, portName))
}

//orderport返回为order实例保留的命名端口。
func (n *Network) OrdererPort(o *Orderer, portName PortName) uint16 {
	ordererPorts := n.PortsByOrdererID[o.ID()]
	Expect(ordererPorts).NotTo(BeNil())
	return ordererPorts[portName]
}

//PeerAddress返回对等端为
//命名端口。命令行工具在
//连接到对等机。
//
//这假定对等机正在0.0.0.0上侦听，并且在
//环回地址。
func (n *Network) PeerAddress(p *Peer, portName PortName) string {
	return fmt.Sprintf("127.0.0.1:%d", n.PeerPort(p, portName))
}

//peer port返回为对等实例保留的命名端口。
func (n *Network) PeerPort(p *Peer, portName PortName) uint16 {
	peerPorts := n.PortsByPeerID[p.ID()]
	Expect(peerPorts).NotTo(BeNil())
	return peerPorts[portName]
}

func (n *Network) nextColor() string {
	color := n.colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	n.colorIndex++
	return fmt.Sprintf("%dm", color)
}

//StartSession执行命令会话。这应该用于启动
//预期要运行到完成的命令行工具。
func (n *Network) StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
	ansiColorCode := n.nextColor()
	return gexec.Start(
		cmd,
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
	)
}

func (n *Network) GenerateCryptoConfig() {
	crypto, err := os.Create(n.CryptoConfigPath())
	Expect(err).NotTo(HaveOccurred())
	defer crypto.Close()

	t, err := template.New("crypto").Parse(n.Templates.CryptoTemplate())
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter("[crypto-config.yaml] ", ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(crypto, pw), n)
	Expect(err).NotTo(HaveOccurred())
}

func (n *Network) GenerateConfigTxConfig() {
	config, err := os.Create(n.ConfigTxConfigPath())
	Expect(err).NotTo(HaveOccurred())
	defer config.Close()

	t, err := template.New("configtx").Parse(n.Templates.ConfigTxTemplate())
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter("[configtx.yaml] ", ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(config, pw), n)
	Expect(err).NotTo(HaveOccurred())
}

func (n *Network) GenerateOrdererConfig(o *Orderer) {
	err := os.MkdirAll(n.OrdererDir(o), 0755)
	Expect(err).NotTo(HaveOccurred())

	orderer, err := os.Create(n.OrdererConfigPath(o))
	Expect(err).NotTo(HaveOccurred())
	defer orderer.Close()

	t, err := template.New("orderer").Funcs(template.FuncMap{
		"Orderer":    func() *Orderer { return o },
		"ToLower":    func(s string) string { return strings.ToLower(s) },
		"ReplaceAll": func(s, old, new string) string { return strings.Replace(s, old, new, -1) },
	}).Parse(n.Templates.OrdererTemplate())
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter(fmt.Sprintf("[%s#orderer.yaml] ", o.ID()), ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(orderer, pw), n)
	Expect(err).NotTo(HaveOccurred())
}

func (n *Network) GenerateCoreConfig(p *Peer) {
	err := os.MkdirAll(n.PeerDir(p), 0755)
	Expect(err).NotTo(HaveOccurred())

	core, err := os.Create(n.PeerConfigPath(p))
	Expect(err).NotTo(HaveOccurred())
	defer core.Close()

	t, err := template.New("peer").Funcs(template.FuncMap{
		"Peer":       func() *Peer { return p },
		"ToLower":    func(s string) string { return strings.ToLower(s) },
		"ReplaceAll": func(s, old, new string) string { return strings.Replace(s, old, new, -1) },
	}).Parse(n.Templates.CoreTemplate())
	Expect(err).NotTo(HaveOccurred())

	pw := gexec.NewPrefixedWriter(fmt.Sprintf("[%s#core.yaml] ", p.ID()), ginkgo.GinkgoWriter)
	err = t.Execute(io.MultiWriter(core, pw), n)
	Expect(err).NotTo(HaveOccurred())
}
