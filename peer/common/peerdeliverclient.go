
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
	"context"

	ccapi "github.com/hyperledger/fabric/peer/chaincode/api"
	pb "github.com/hyperledger/fabric/protos/peer"
	grpc "google.golang.org/grpc"
)

//PeerDeliverClient保存连接客户端所需的信息
//到对等交付服务
type PeerDeliverClient struct {
	Client pb.DeliverClient
}

//传递将客户端连接到传递RPC
func (dc PeerDeliverClient) Deliver(ctx context.Context, opts ...grpc.CallOption) (ccapi.Deliver, error) {
	d, err := dc.Client.Deliver(ctx, opts...)
	return d, err
}

//deliverfiltered将客户端连接到deliverfiltered rpc
func (dc PeerDeliverClient) DeliverFiltered(ctx context.Context, opts ...grpc.CallOption) (ccapi.Deliver, error) {
	df, err := dc.Client.DeliverFiltered(ctx, opts...)
	return df, err
}
