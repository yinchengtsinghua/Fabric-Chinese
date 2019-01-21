
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


package peer

import (
	"fmt"
	"sync"
	"time"

	pb "github.com/hyperledger/fabric/protos/peer"
)

//mockresponseset用于处理CC到对等通信
//例如get/put/del state。mockResponse包含
//
//
type MockResponseSet struct {
//
//响应集
	DoneFunc func(int, error)

//当输入没有被调用时，任何步骤都会调用errorfunc。
//匹配收到的消息
	ErrorFunc func(int, error)

//
//和发送响应（可选）
	Responses []*MockResponse
}

//
//和发送响应（可选）
type MockResponse struct {
	RecvMsg *pb.ChaincodeMessage
	RespMsg interface{}
}

//mockcccomm实现了链码和对等端之间的模拟通信。
//
//
type MockCCComm struct {
	name        string
	bailOnError bool
	keepAlive   *pb.ChaincodeMessage
	recvStream  chan *pb.ChaincodeMessage
	sendStream  chan *pb.ChaincodeMessage
	respIndex   int
	respLock    sync.Mutex
	respSet     *MockResponseSet
	pong        bool
	skipClose   bool
}

func (s *MockCCComm) SetName(newname string) {
	s.name = newname
}

//
func (s *MockCCComm) Send(msg *pb.ChaincodeMessage) error {
	s.sendStream <- msg
	return nil
}

//
func (s *MockCCComm) Recv() (*pb.ChaincodeMessage, error) {
	msg := <-s.recvStream
	return msg, nil
}

//
func (s *MockCCComm) CloseSend() error {
	return nil
}

//
func (s *MockCCComm) GetRecvStream() chan *pb.ChaincodeMessage {
	return s.recvStream
}

//
func (s *MockCCComm) GetSendStream() chan *pb.ChaincodeMessage {
	return s.sendStream
}

//
func (s *MockCCComm) Quit() {
	if !s.skipClose {
		close(s.recvStream)
		close(s.sendStream)
	}
}

//
func (s *MockCCComm) SetBailOnError(b bool) {
	s.bailOnError = b
}

//
func (s *MockCCComm) SetPong(val bool) {
	s.pong = val
}

//setkeepalive设置keepalive。此mut只能在服务器上完成
func (s *MockCCComm) SetKeepAlive(ka *pb.ChaincodeMessage) {
	s.keepAlive = ka
}

//
func (s *MockCCComm) SetResponses(respSet *MockResponseSet) {
	s.respLock.Lock()
	s.respSet = respSet
	s.respIndex = 0
	s.respLock.Unlock()
}

//保持活力
func (s *MockCCComm) ka(done <-chan struct{}) {
	for {
		if s.keepAlive == nil {
			return
		}
		s.Send(s.keepAlive)
		select {
		case <-time.After(10 * time.Millisecond):
		case <-done:
			return
		}
	}
}

//
func (s *MockCCComm) Run(done <-chan struct{}) error {
//启动keepalive
	go s.ka(done)
	defer s.Quit()

	for {
		msg, err := s.Recv()

//流可能刚刚关闭
		if msg == nil {
			return err
		}

		if err != nil {
			return err
		}

		if err = s.respond(msg); err != nil {
			if s.bailOnError {
				return err
			}
		}
	}
}

func (s *MockCCComm) respond(msg *pb.ChaincodeMessage) error {
	if msg != nil && msg.Type == pb.ChaincodeMessage_KEEPALIVE {
//
		if s.pong {
			return s.Send(msg)
		}
		return nil
	}

	s.respLock.Lock()
	defer s.respLock.Unlock()

	var err error
	if s.respIndex < len(s.respSet.Responses) {
		mockResp := s.respSet.Responses[s.respIndex]
		if mockResp.RecvMsg != nil {
			if msg.Type != mockResp.RecvMsg.Type {
				if s.respSet.ErrorFunc != nil {
					s.respSet.ErrorFunc(s.respIndex, fmt.Errorf("Invalid message expected %d received %d", int32(mockResp.RecvMsg.Type), int32(msg.Type)))
					s.respIndex = s.respIndex + 1
					return nil
				}
			}
		}

		if mockResp.RespMsg != nil {
			var ccMsg *pb.ChaincodeMessage
			if ccMsg, _ = mockResp.RespMsg.(*pb.ChaincodeMessage); ccMsg == nil {
				if ccMsgFunc, ok := mockResp.RespMsg.(func(*pb.ChaincodeMessage) *pb.ChaincodeMessage); ok && ccMsgFunc != nil {
					ccMsg = ccMsgFunc(msg)
				}
			}

			if ccMsg == nil {
				panic("----no pb.ChaincodeMessage---")
			}
			err = s.Send(ccMsg)
		}

		s.respIndex = s.respIndex + 1

		if s.respIndex == len(s.respSet.Responses) {
			if s.respSet.DoneFunc != nil {
				s.respSet.DoneFunc(s.respIndex, nil)
			}
		}
	}
	return err
}
