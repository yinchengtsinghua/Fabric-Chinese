
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


package api

import (
	"context"

	"github.com/hyperledger/fabric/peer/chaincode/api"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"google.golang.org/grpc"
)

//go：生成伪造者-o../mock/deliverclient.go-forke-name deliverclient。交付客户

//DeliverClient为Delivery客户端定义接口
type DeliverClient interface {
	Deliver(ctx context.Context, opts ...grpc.CallOption) (DeliverService, error)
}

//go：生成仿冒者-o../mock/deliverservice.go-forke name deliverservice。交付服务

//DeliverService定义用于传递块的接口
type DeliverService interface {
	Send(*cb.Envelope) error
	Recv() (*ab.DeliverResponse, error)
	CloseSend() error
}

//go:生成仿冒者-o../mock/peerdeliverclient.go-仿冒名称peerdeliverclient。对等传送客户端

//PeerDeliverClient定义对等传递客户端的接口
type PeerDeliverClient interface {
	Deliver(ctx context.Context, opts ...grpc.CallOption) (api.Deliver, error)
	DeliverFiltered(ctx context.Context, opts ...grpc.CallOption) (api.Deliver, error)
}
