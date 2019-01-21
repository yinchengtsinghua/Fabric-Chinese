
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2017保留所有权利。

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


package mocks

import (
	"math"
	"testing"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type clStream struct {
	grpc.ServerStream
}

func (cs *clStream) Send(*orderer.DeliverResponse) error {
	return nil
}
func (cs *clStream) Recv() (*common.Envelope, error) {
	seekInfo := &orderer.SeekInfo{
		Start:    &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: 0}}},
		Stop:     &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: math.MaxUint64}}},
		Behavior: orderer.SeekInfo_BLOCK_UNTIL_READY,
	}
	si, _ := pb.Marshal(seekInfo)
	payload := &common.Payload{}
	payload.Data = si
	b, err := pb.Marshal(payload)
	if err != nil {
		panic(err)
	}
	e := &common.Envelope{Payload: b}
	return e, nil
}

func TestOrderer(t *testing.T) {
	o := NewOrderer(8000, t)

	go func() {
		time.Sleep(time.Second)
		o.SendBlock(uint64(0))
		o.Shutdown()
	}()

	assert.Panics(t, func() {
		o.Broadcast(nil)
	})
	o.SetNextExpectedSeek(uint64(0))
	o.Deliver(&clStream{})
}
