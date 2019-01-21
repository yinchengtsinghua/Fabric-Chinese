
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有2017 Hitachi America，Ltd.

根据Apache许可证2.0版（以下简称“许可证”）获得许可；
除非符合许可证，否则您不能使用此文件。
您可以在以下网址获得许可证副本：

    http://www.apache.org/licenses/license-2.0

除非适用法律要求或书面同意，软件
根据许可证分发是按“原样”分发的，
无任何明示或暗示的保证或条件。
有关管理权限和
许可证限制。
**/


package node

import (
	"context"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/admin"
	"github.com/hyperledger/fabric/core/comm"
	testpb "github.com/hyperledger/fabric/core/comm/testdata/grpc"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/msp"
	common2 "github.com/hyperledger/fabric/peer/common"
	"github.com/hyperledger/fabric/peer/mocks"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type testServiceServer struct{}

func (tss *testServiceServer) EmptyCall(context.Context, *testpb.Empty) (*testpb.Empty, error) {
	return new(testpb.Empty), nil
}

type mockEvaluator struct {
}

func (*mockEvaluator) Evaluate(signatureSet []*common.SignedData) error {
	return nil
}

func TestStatusCmd(t *testing.T) {
	signer := &mocks.Signer{}
	common2.GetDefaultSignerFnc = func() (msp.SigningIdentity, error) {
		return signer, nil
	}
	viper.Set("peer.address", "localhost:7070")
	peerServer, err := peer.NewPeerServer("localhost:7070", comm.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create peer server (%s)", err)
	} else {
		pb.RegisterAdminServer(peerServer.Server(), admin.NewAdminServer(&mockEvaluator{}))
		go peerServer.Start()
		defer peerServer.Stop()

		cmd := statusCmd()
		if err := cmd.Execute(); err != nil {
			t.Fail()
			t.Errorf("expected status command to succeed")
		}
	}
}

func TestStatus(t *testing.T) {
	defer viper.Reset()

	signer := &mocks.Signer{}
	common2.GetDefaultSignerFnc = func() (msp.SigningIdentity, error) {
		return signer, nil
	}
	var tests = []struct {
		name          string
		peerAddress   string
		listenAddress string
		timeout       time.Duration
		shouldSucceed bool
	}{
		{
			name:          "status function to success",
			peerAddress:   "localhost:7071",
			listenAddress: "localhost:7071",
			timeout:       time.Second,
			shouldSucceed: true,
		},
		{
			name:          "admin client error",
			peerAddress:   "",
			listenAddress: "localhost:7072",
			timeout:       100 * time.Millisecond,
			shouldSucceed: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Running test: %s", test.name)
			viper.Set("peer.address", test.peerAddress)
			viper.Set("peer.client.connTimeout", test.timeout)
			peerServer, err := peer.NewPeerServer(test.listenAddress, comm.ServerConfig{})
			if err != nil {
				t.Fatalf("Failed to create peer server (%s)", err)
			} else {
				pb.RegisterAdminServer(peerServer.Server(), admin.NewAdminServer(&mockEvaluator{}))
				go peerServer.Start()
				defer peerServer.Stop()
				if test.shouldSucceed {
					assert.NoError(t, status())
				} else {
					assert.Error(t, status())
				}
			}
		})
	}
}

func TestStatusWithGetStatusError(t *testing.T) {
	defer viper.Reset()

	viper.Set("peer.address", "localhost:7073")
	peerServer, err := peer.NewPeerServer(":7073", comm.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create peer server (%s)", err)
	}
	testpb.RegisterTestServiceServer(peerServer.Server(), &testServiceServer{})
	go peerServer.Start()
	defer peerServer.Stop()
	assert.Error(t, status())
}
