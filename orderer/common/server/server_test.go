
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package server

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	localconfig "github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/common/multichannel"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

func TestBroadcastNoPanic(t *testing.T) {
//推迟从恐慌中恢复
	_ = (&server{}).Broadcast(nil)
}

func TestDeliverNoPanic(t *testing.T) {
//推迟从恐慌中恢复
	_ = (&server{}).Deliver(nil)
}

type recvr interface {
	Recv() (*cb.Envelope, error)
}

type mockSrv struct {
	grpc.ServerStream
	msg *cb.Envelope
	err error
}

func (mockSrv) Context() context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{})
}

type mockBroadcastSrv mockSrv

func (mbs *mockBroadcastSrv) Recv() (*cb.Envelope, error) {
	return mbs.msg, mbs.err
}

func (mbs *mockBroadcastSrv) Send(br *ab.BroadcastResponse) error {
	panic("Unimplimented")
}

type mockDeliverSrv mockSrv

func (mds *mockDeliverSrv) CreateStatusReply(status cb.Status) proto.Message {
	return &ab.DeliverResponse{
		Type: &ab.DeliverResponse_Status{Status: status},
	}
}

func (mds *mockDeliverSrv) CreateBlockReply(block *cb.Block) proto.Message {
	return &ab.DeliverResponse{
		Type: &ab.DeliverResponse_Block{Block: block},
	}
}

func (mds *mockDeliverSrv) Recv() (*cb.Envelope, error) {
	return mds.msg, mds.err
}

func (mds *mockDeliverSrv) Send(br *ab.DeliverResponse) error {
	panic("Unimplimented")
}

func testMsgTrace(handler func(dir string, msg *cb.Envelope) recvr, t *testing.T) {
	dir, err := ioutil.TempDir("", "TestMsgTrace")
	if err != nil {
		t.Fatalf("Could not create temp dir")
	}
	defer os.RemoveAll(dir)

	msg := &cb.Envelope{Payload: []byte("somedata")}

	r := handler(dir, msg)

	rMsg, err := r.Recv()
	assert.Equal(t, msg, rMsg)
	assert.Nil(t, err)

	var fileData []byte
	for i := 0; i < 100; i++ {
//写入跟踪文件是故意不阻塞的，最多等待一秒钟，每隔10毫秒检查一次，以查看文件现在是否存在。
		time.Sleep(10 * time.Millisecond)
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			assert.Nil(t, err)
			if path == dir {
				return nil
			}
			assert.Nil(t, fileData, "Should only be one file")
			fileData, err = ioutil.ReadFile(path)
			assert.Nil(t, err)
			return nil
		})
		if fileData != nil {
			break
		}
	}

	assert.Equal(t, utils.MarshalOrPanic(msg), fileData)
}

func TestBroadcastMsgTrace(t *testing.T) {
	testMsgTrace(func(dir string, msg *cb.Envelope) recvr {
		return &broadcastMsgTracer{
			AtomicBroadcast_BroadcastServer: &mockBroadcastSrv{
				msg: msg,
			},
			msgTracer: msgTracer{
				debug: &localconfig.Debug{
					BroadcastTraceDir: dir,
				},
				function: "Broadcast",
			},
		}
	}, t)
}

func TestDeliverMsgTrace(t *testing.T) {
	testMsgTrace(func(dir string, msg *cb.Envelope) recvr {
		return &deliverMsgTracer{
			Receiver: &mockDeliverSrv{
				msg: msg,
			},
			msgTracer: msgTracer{
				debug: &localconfig.Debug{
					DeliverTraceDir: dir,
				},
				function: "Deliver",
			},
		}
	}, t)
}

func TestDeliverNoChannel(t *testing.T) {
	r := &multichannel.Registrar{}
	ds := &deliverSupport{Registrar: r}
	chain := ds.GetChain("mychannel")
	assert.Nil(t, chain)
	assert.True(t, chain == nil)
}
