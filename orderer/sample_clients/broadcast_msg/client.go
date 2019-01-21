
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
	"sync"

	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"google.golang.org/grpc"
	"gopkg.in/cheggaaa/pb.v1"
)

type broadcastClient struct {
	client    ab.AtomicBroadcast_BroadcastClient
	signer    crypto.LocalSigner
	channelID string
}

//newbroadcastclient创建一个简单的BroadcastClient接口实例
func newBroadcastClient(client ab.AtomicBroadcast_BroadcastClient, channelID string, signer crypto.LocalSigner) *broadcastClient {
	return &broadcastClient{client: client, channelID: channelID, signer: signer}
}

func (s *broadcastClient) broadcast(transaction []byte) error {
	env, err := utils.CreateSignedEnvelope(cb.HeaderType_MESSAGE, s.channelID, s.signer, &cb.ConfigValue{Value: transaction}, 0, 0)
	if err != nil {
		panic(err)
	}
	return s.client.Send(env)
}

func (s *broadcastClient) getAck() error {
	msg, err := s.client.Recv()
	if err != nil {
		return err
	}
	if msg.Status != cb.Status_SUCCESS {
		return fmt.Errorf("Got unexpected status: %v - %s", msg.Status, msg.Info)
	}
	return nil
}

func main() {
	conf, err := localconfig.Load()
	if err != nil {
		fmt.Println("failed to load config:", err)
		os.Exit(1)
	}

//加载本地MSP
	err = mspmgmt.LoadLocalMsp(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID)
if err != nil { //处理读取配置文件时的错误
		fmt.Println("Failed to initialize local MSP:", err)
		os.Exit(0)
	}

	signer := localmsp.NewSigner()

	var channelID string
	var serverAddr string
	var messages uint64
	var goroutines uint64
	var msgSize uint64
	var bar *pb.ProgressBar

	flag.StringVar(&serverAddr, "server", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The RPC server to connect to.")
	flag.StringVar(&channelID, "channelID", localconfig.Defaults.General.SystemChannel, "The channel ID to broadcast to.")
	flag.Uint64Var(&messages, "messages", 1, "The number of messages to broadcast.")
	flag.Uint64Var(&goroutines, "goroutines", 1, "The number of concurrent go routines to broadcast the messages on")
	flag.Uint64Var(&msgSize, "size", 1024, "The size in bytes of the data section for the payload")
	flag.Parse()

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	defer func() {
		_ = conn.Close()
	}()
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	msgsPerGo := messages / goroutines
	roundMsgs := msgsPerGo * goroutines
	if roundMsgs != messages {
		fmt.Println("Rounding messages to", roundMsgs)
	}
	bar = pb.New64(int64(roundMsgs))
	bar.ShowPercent = true
	bar.ShowSpeed = true
	bar = bar.Start()

	msgData := make([]byte, msgSize)

	var wg sync.WaitGroup
	wg.Add(int(goroutines))
	for i := uint64(0); i < goroutines; i++ {
		go func(i uint64, pb *pb.ProgressBar) {
			client, err := ab.NewAtomicBroadcastClient(conn).Broadcast(context.TODO())
			if err != nil {
				fmt.Println("Error connecting:", err)
				return
			}

			s := newBroadcastClient(client, channelID, signer)
			done := make(chan (struct{}))
			go func() {
				for i := uint64(0); i < msgsPerGo; i++ {
					err = s.getAck()
					if err == nil && bar != nil {
						bar.Increment()
					}
				}
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				close(done)
			}()
			for i := uint64(0); i < msgsPerGo; i++ {
				if err := s.broadcast(msgData); err != nil {
					panic(err)
				}
			}
			<-done
			wg.Done()
			client.CloseSend()
		}(i, bar)
	}

	wg.Wait()
	bar.FinishPrint("----------------------broadcast message finish-------------------------------")
}
