
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


package comm_test

import (
	"context"
	"sync"
	"testing"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/comm/mock"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

//go：生成伪造者-o mock/semaphore.go-forke-name信号量。旗语
//go：生成伪造者-o mock/new_信号量。go-fake name new semaphore。新闻电子邮件
//go：生成仿冒者-o mock/server_stream.go-forke name server stream。服务器流
//go：生成伪造者-o mock/stream_handler.go-forke name stream handler。流处理器
//go：生成伪造者-o mock/unary处理程序。go-伪造名称unary handler。无核处理机

type serverStream interface{ grpc.ServerStream }
type streamHandler grpc.StreamHandler
type unaryHandler grpc.UnaryHandler

func TestThrottle(t *testing.T) {
	gt := NewGomegaWithT(t)

	wg := sync.WaitGroup{}
	done := make(chan struct{})
	unaryHandler := &mock.UnaryHandler{}
	unaryHandler.Stub = func(context.Context, interface{}) (interface{}, error) {
		wg.Done()
		<-done
		return nil, nil
	}
	streamHandler := &mock.StreamHandler{}
	streamHandler.Stub = func(interface{}, grpc.ServerStream) error {
		wg.Done()
		<-done
		return nil
	}
	serverStream := &mock.ServerStream{}
	serverStream.ContextReturns(context.Background())

	throttle := comm.NewThrottle(2)

//将请求运行到限制点
	wg.Add(2)
	go throttle.UnaryServerIntercptor(context.Background(), nil, nil, unaryHandler.Spy)
	go throttle.StreamServerInterceptor(nil, serverStream, nil, streamHandler.Spy)
	wg.Wait()

	unaryComplete := make(chan struct{})
	blockedUnaryHandler := &mock.UnaryHandler{}
	blockedUnaryHandler.Stub = func(context.Context, interface{}) (interface{}, error) {
		close(unaryComplete)
		return nil, nil
	}
	go throttle.UnaryServerIntercptor(context.Background(), nil, nil, blockedUnaryHandler.Spy)
	gt.Consistently(unaryComplete).ShouldNot(BeClosed())

	streamComplete := make(chan struct{})
	blockedStreamHandler := &mock.StreamHandler{}
	blockedStreamHandler.Stub = func(interface{}, grpc.ServerStream) error {
		close(streamComplete)
		return nil
	}
	go throttle.StreamServerInterceptor(nil, serverStream, nil, blockedStreamHandler.Spy)
	gt.Consistently(streamComplete).ShouldNot(BeClosed())

	close(done)
	gt.Eventually(unaryComplete).Should(BeClosed())
	gt.Eventually(streamComplete).Should(BeClosed())
}

func TestUnaryServerInterceptor(t *testing.T) {
	gt := NewGomegaWithT(t)

	semaphore := &mock.Semaphore{}
	newSemaphore := &mock.NewSemaphore{}
	newSemaphore.Returns(semaphore)

	key := struct{}{}
	ctx := context.WithValue(context.Background(), key, "value")
	handler := &mock.UnaryHandler{}
	handler.Returns("result", nil)

	throttle := comm.NewThrottle(3, comm.WithNewSemaphore(newSemaphore.Spy))

	result, err := throttle.UnaryServerIntercptor(ctx, "request", nil, handler.Spy)
	gt.Expect(err).NotTo(HaveOccurred())
	gt.Expect(result).To(Equal("result"))

	gt.Expect(handler.CallCount()).To(Equal(1))
	c, r := handler.ArgsForCall(0)
	gt.Expect(c).To(BeIdenticalTo(ctx))
	gt.Expect(r).To(Equal("request"))

	gt.Expect(semaphore.AcquireCallCount()).To(Equal(1))

	gt.Expect(semaphore.ReleaseCallCount()).To(Equal(1))
}

func TestUnaryServerInterceptorAcquireFail(t *testing.T) {
	gt := NewGomegaWithT(t)

	semaphore := &mock.Semaphore{}
	semaphore.AcquireReturns(errors.New("you're-dead-to-me"))
	newSemaphore := &mock.NewSemaphore{}
	newSemaphore.Returns(semaphore)

	throttle := comm.NewThrottle(3, comm.WithNewSemaphore(newSemaphore.Spy))
	ctx := context.Background()

	_, err := throttle.UnaryServerIntercptor(ctx, "request", nil, nil)
	gt.Expect(err).To(MatchError("you're-dead-to-me"))

	gt.Expect(semaphore.AcquireCallCount()).To(Equal(1))
	c := semaphore.AcquireArgsForCall(0)
	gt.Expect(c).To(Equal(ctx))
	gt.Expect(semaphore.ReleaseCallCount()).To(Equal(0))
}

func TestStreamServerInterceptor(t *testing.T) {
	gt := NewGomegaWithT(t)

	semaphore := &mock.Semaphore{}
	newSemaphore := &mock.NewSemaphore{}
	newSemaphore.Returns(semaphore)

	key := struct{}{}
	expectedSrv := struct{ string }{"server"}
	ctx := context.WithValue(context.Background(), key, "value")

	serverStream := &mock.ServerStream{}
	serverStream.ContextReturns(ctx)

	handler := &mock.StreamHandler{}

	throttle := comm.NewThrottle(3, comm.WithNewSemaphore(newSemaphore.Spy))
	err := throttle.StreamServerInterceptor(expectedSrv, serverStream, nil, handler.Spy)
	gt.Expect(err).NotTo(HaveOccurred())

	gt.Expect(handler.CallCount()).To(Equal(1))
	srv, ss := handler.ArgsForCall(0)
	gt.Expect(srv).To(Equal(expectedSrv))
	gt.Expect(ss).To(Equal(serverStream))

	gt.Expect(serverStream.ContextCallCount()).To(Equal(1))
	gt.Expect(semaphore.AcquireCallCount()).To(Equal(1))
	c := semaphore.AcquireArgsForCall(0)
	gt.Expect(c).To(Equal(ctx))
	gt.Expect(semaphore.ReleaseCallCount()).To(Equal(1))
}

func TestStreamServerInterceptorAcquireFail(t *testing.T) {
	gt := NewGomegaWithT(t)

	semaphore := &mock.Semaphore{}
	semaphore.AcquireReturns(errors.New("your-name-is-mud"))
	newSemaphore := &mock.NewSemaphore{}
	newSemaphore.Returns(semaphore)

	throttle := comm.NewThrottle(3, comm.WithNewSemaphore(newSemaphore.Spy))
	ctx := context.Background()

	serverStream := &mock.ServerStream{}
	serverStream.ContextReturns(ctx)

	err := throttle.StreamServerInterceptor(nil, serverStream, nil, nil)
	gt.Expect(err).To(MatchError("your-name-is-mud"))

	gt.Expect(semaphore.AcquireCallCount()).To(Equal(1))
	gt.Expect(semaphore.ReleaseCallCount()).To(Equal(0))
}
