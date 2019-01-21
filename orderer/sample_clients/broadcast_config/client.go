
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有IBM公司。保留所有权利。
//SPDX许可证标识符：Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"google.golang.org/grpc"
)

type broadcastClient struct {
	ab.AtomicBroadcast_BroadcastClient
}

func (bc *broadcastClient) broadcast(env *cb.Envelope) error {
	var err error
	var resp *ab.BroadcastResponse

	err = bc.Send(env)
	if err != nil {
		return err
	}

	resp, err = bc.Recv()
	if err != nil {
		return err
	}

	fmt.Println("Status:", resp)
	return nil
}

//cmdimpl保存命令及其参数。
type cmdImpl struct {
	name string
	args argsImpl
}

//argsimpl保存所有可能命令的所有可能参数。
type argsImpl struct {
	consensusType  string
	creationPolicy string
	chainID        string
}

var conf *localconfig.TopLevel
var genConf *genesisconfig.Profile

func init() {
	var err error
	conf, err = localconfig.Load()
	if err != nil {
		fmt.Println("failed to load config:", err)
		os.Exit(1)
	}

//加载本地MSP
	err = mspmgmt.LoadLocalMsp(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID)
	if err != nil {
		panic(fmt.Errorf("Failed to initialize local MSP: %s", err))
	}

	genConf = genesisconfig.Load(conf.General.GenesisProfile)
}

func main() {
	cmd := new(cmdImpl)
	var srv string

	flag.StringVar(&srv, "server", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The RPC server to connect to.")
	flag.StringVar(&cmd.name, "cmd", "newChain", "The action that this client is requesting via the config transaction.")
	flag.StringVar(&cmd.args.consensusType, "consensusType", genConf.Orderer.OrdererType, "In case of a newChain command, the type of consensus the ordering service is running on.")
	flag.StringVar(&cmd.args.creationPolicy, "creationPolicy", "AcceptAllPolicy", "In case of a newChain command, the chain creation policy this request should be validated against.")
	flag.StringVar(&cmd.args.chainID, "chainID", "mychannel", "In case of a newChain command, the chain ID to create.")
	flag.Parse()

	conn, err := grpc.Dial(srv, grpc.WithInsecure())
	defer func() {
		_ = conn.Close()
	}()
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	client, err := ab.NewAtomicBroadcastClient(conn).Broadcast(context.TODO())
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	bc := &broadcastClient{client}

	switch cmd.name {
	case "newChain":
		env := newChainRequest(cmd.args.consensusType, cmd.args.creationPolicy, cmd.args.chainID)
		fmt.Println("Requesting the creation of chain", cmd.args.chainID)
		fmt.Println(bc.broadcast(env))
	default:
		panic("Invalid command given")
	}
}
