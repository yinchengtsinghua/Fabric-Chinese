
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


package common

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric/cmd/common/comm"
	"github.com/hyperledger/fabric/cmd/common/signer"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	saveConfigCommand = "saveConfig"
)

var (
//用于终止CLI的函数
	terminate = os.Exit
//用于将输出重定向到的函数
	outWriter io.Writer = os.Stderr

//CLI参数
	mspID                                     *string
	tlsCA, tlsCert, tlsKey, userKey, userCert **os.File
	configFile                                *string
)

//cli command定义添加到cli的命令
//通过外部消费者。
type CLICommand func(Config) error

//cli定义命令行解释器
type CLI struct {
	app         *kingpin.Application
	dispatchers map[string]CLICommand
}

//new cli创建具有给定名称和帮助消息的新cli
func NewCLI(name, help string) *CLI {
	return &CLI{
		app:         kingpin.New(name, help),
		dispatchers: make(map[string]CLICommand),
	}
}

//命令将新的顶级命令添加到CLI
func (cli *CLI) Command(name, help string, onCommand CLICommand) *kingpin.CmdClause {
	cmd := cli.app.Command(name, help)
	cli.dispatchers[name] = onCommand
	return cmd
}

//run使cli处理参数并执行带有标志的命令
func (cli *CLI) Run(args []string) {
	configFile = cli.app.Flag("configFile", "Specifies the config file to load the configuration from").String()
	persist := cli.app.Command(saveConfigCommand, fmt.Sprintf("Save the config passed by flags into the file specified by --configFile"))
	configureFlags(cli.app)

	command := kingpin.MustParse(cli.app.Parse(args))
	if command == persist.FullCommand() {
		if *configFile == "" {
			out("--configFile must be used to specify the configuration file")
			return
		}
		persistConfig(parseFlagsToConfig(), *configFile)
		return
	}

	var conf Config
	if *configFile == "" {
		conf = parseFlagsToConfig()
	} else {
		conf = loadConfig(*configFile)
	}

	f, exists := cli.dispatchers[command]
	if !exists {
		out("Unknown command:", command)
		terminate(1)
		return
	}
	err := f(conf)
	if err != nil {
		out(err)
		terminate(1)
		return
	}
}

func configureFlags(persistCommand *kingpin.Application) {
//TLS标志
	tlsCA = persistCommand.Flag("peerTLSCA", "Sets the TLS CA certificate file path that verifies the TLS peer's certificate").File()
	tlsCert = persistCommand.Flag("tlsCert", "(Optional) Sets the client TLS certificate file path that is used when the peer enforces client authentication").File()
	tlsKey = persistCommand.Flag("tlsKey", "(Optional) Sets the client TLS key file path that is used when the peer enforces client authentication").File()
//注册标志
	userKey = persistCommand.Flag("userKey", "Sets the user's key file path that is used to sign messages sent to the peer").File()
	userCert = persistCommand.Flag("userCert", "Sets the user's certificate file path that is used to authenticate the messages sent to the peer").File()
	mspID = persistCommand.Flag("MSP", "Sets the MSP ID of the user, which represents the CA(s) that issued its user certificate").String()
}

func persistConfig(conf Config, file string) {
	if err := conf.ToFile(file); err != nil {
		out("Failed persisting configuration:", err)
		terminate(1)
	}
}

func loadConfig(file string) Config {
	conf, err := ConfigFromFile(file)
	if err != nil {
		out("Failed loading config", err)
		terminate(1)
		return Config{}
	}
	return conf
}

func parseFlagsToConfig() Config {
	conf := Config{
		SignerConfig: signer.Config{
			MSPID:        *mspID,
			IdentityPath: evaluateFileFlag(userCert),
			KeyPath:      evaluateFileFlag(userKey),
		},
		TLSConfig: comm.Config{
			KeyPath:        evaluateFileFlag(tlsKey),
			CertPath:       evaluateFileFlag(tlsCert),
			PeerCACertPath: evaluateFileFlag(tlsCA),
		},
	}
	return conf
}

func evaluateFileFlag(f **os.File) string {
	if f == nil {
		return ""
	}
	if *f == nil {
		return ""
	}
	path, err := filepath.Abs((*f).Name())
	if err != nil {
		out("Failed listing", (*f).Name(), ":", err)
		terminate(1)
	}
	return path
}
func out(a ...interface{}) {
	fmt.Fprintln(outWriter, a...)
}
