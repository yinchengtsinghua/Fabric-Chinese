
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


package inproccontroller

import (
	"testing"

	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

func TestSend(t *testing.T) {
	ch := make(chan *pb.ChaincodeMessage)

	stream := newInProcStream(ch, ch)

//良好发送（非阻塞发送和接收）
	msg := &pb.ChaincodeMessage{}
	go stream.Send(msg)
	msg2, _ := stream.Recv()
	assert.Equal(t, msg, msg2, "send != recv")

//关闭频道
	close(ch)

//发送错误，应死机、解锁并返回错误
	err := stream.Send(msg)
	assert.NotNil(t, err, "should have errored on panic")
}

func TestRecvChannelClosedError(t *testing.T) {
	ch := make(chan *pb.ChaincodeMessage)

	stream := newInProcStream(ch, ch)

//关闭频道
	close(ch)

//尝试调用关闭的接收通道应返回错误
	_, err := stream.Recv()
	if assert.Error(t, err, "Should return an error") {
		assert.Contains(t, err.Error(), "channel is closed")
	}
}
