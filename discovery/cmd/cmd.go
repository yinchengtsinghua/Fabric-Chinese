
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


package discovery

import (
	"os"
	"time"

	"github.com/hyperledger/fabric/cmd/common"
	"github.com/hyperledger/fabric/discovery/client"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	PeersCommand     = "peers"
	ConfigCommand    = "config"
	EndorsersCommand = "endorsers"
)

var (
//ResponseParserWriter定义stdout
	responseParserWriter = os.Stdout
)

const (
	defaultTimeout = time.Second * 10
)

//去：生成mokery-dir。-名称存根-大小写下划线-输出模拟/

//存根表示远程发现服务
type Stub interface {
//发送发送请求并接收响应
	Send(server string, conf common.Config, req *discovery.Request) (ServiceResponse, error)
}

//去：生成mokery-dir。-name responseparser-case underline-输出模拟/

//ResponseParser分析从服务器发送的响应
type ResponseParser interface {
//ParseResponse解析响应并在发出数据时使用给定的输出
	ParseResponse(channel string, response ServiceResponse) error
}

//去：生成mokery-dir。-name命令注册器-case下划线-output mocks/

//命令注册寄存器命令
type CommandRegistrar interface {
//命令将新的顶级命令添加到CLI
	Command(name, help string, onCommand common.CLICommand) *kingpin.CmdClause
}

//addcommands将发现命令注册到给定的commandregistrar
func AddCommands(cli CommandRegistrar) {
	peerCmd := NewPeerCmd(&ClientStub{}, &PeerResponseParser{Writer: responseParserWriter})
	peers := cli.Command(PeersCommand, "Discover peers", peerCmd.Execute)
	server := peers.Flag("server", "Sets the endpoint of the server to connect").String()
	channel := peers.Flag("channel", "Sets the channel the query is intended to").String()
	peerCmd.SetServer(server)
	peerCmd.SetChannel(channel)

	configCmd := NewConfigCmd(&ClientStub{}, &ConfigResponseParser{Writer: responseParserWriter})
	config := cli.Command(ConfigCommand, "Discover channel config", configCmd.Execute)
	server = config.Flag("server", "Sets the endpoint of the server to connect").String()
	channel = config.Flag("channel", "Sets the channel the query is intended to").String()
	configCmd.SetServer(server)
	configCmd.SetChannel(channel)

	endorserCmd := NewEndorsersCmd(&RawStub{}, &EndorserResponseParser{Writer: responseParserWriter})
	endorsers := cli.Command(EndorsersCommand, "Discover chaincode endorsers", endorserCmd.Execute)
	chaincodes := endorsers.Flag("chaincode", "Specifies the chaincode name(s)").Strings()
	collections := endorsers.Flag("collection", "Specifies the collection name(s) as a mapping from chaincode to a comma separated list of collections").PlaceHolder("CC:C1,C2").StringMap()
	server = endorsers.Flag("server", "Sets the endpoint of the server to connect").String()
	channel = endorsers.Flag("channel", "Sets the channel the query is intended to").String()
	endorserCmd.SetChannel(channel)
	endorserCmd.SetServer(server)
	endorserCmd.SetChaincodes(chaincodes)
	endorserCmd.SetCollections(collections)
}
