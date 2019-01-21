
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


package cluster_test

import (
	"context"
	"io"
	"testing"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/cluster/mocks"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	submitRequest1  = &orderer.SubmitRequest{}
	submitRequest2  = &orderer.SubmitRequest{}
	submitResponse1 = &orderer.SubmitResponse{}
	submitResponse2 = &orderer.SubmitResponse{}
	stepRequest     = &orderer.StepRequest{}
	stepResponse    = &orderer.StepResponse{}
)

func TestStep(t *testing.T) {
	t.Parallel()
	dispatcher := &mocks.Dispatcher{}

	svc := &cluster.Service{
		Logger:     flogging.MustGetLogger("test"),
		StepLogger: flogging.MustGetLogger("test"),
		Dispatcher: dispatcher,
	}

	t.Run("Success", func(t *testing.T) {
		dispatcher.On("DispatchStep", mock.Anything, stepRequest).Return(stepResponse, nil).Once()
		res, err := svc.Step(context.Background(), stepRequest)
		assert.NoError(t, err)
		assert.Equal(t, stepResponse, res)
	})

	t.Run("Failure", func(t *testing.T) {
		dispatcher.On("DispatchStep", mock.Anything, stepRequest).Return(nil, errors.New("oops")).Once()
		_, err := svc.Step(context.Background(), stepRequest)
		assert.EqualError(t, err, "oops")
	})
}

func TestSubmitSuccess(t *testing.T) {
	t.Parallel()
	dispatcher := &mocks.Dispatcher{}

	stream := &mocks.SubmitStream{}
	stream.On("Context").Return(context.Background())
//发送到流2消息，然后关闭流
	stream.On("Recv").Return(submitRequest1, nil).Once()
	stream.On("Recv").Return(submitRequest2, nil).Once()
	stream.On("Recv").Return(nil, io.EOF).Once()
//应为每个对应的接收调用发送
	stream.On("Send", submitResponse1).Return(nil).Twice()

	responses := make(chan *orderer.SubmitRequest, 2)
	responses <- submitRequest1
	responses <- submitRequest2

	dispatcher.On("DispatchSubmit", mock.Anything, mock.Anything).Return(submitResponse1, nil).Once()
	dispatcher.On("DispatchSubmit", mock.Anything, mock.Anything).Return(submitResponse2, nil).Once()
//确保我们按顺序传递发送提交的请求
	dispatcher.On("DispatchSubmit", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		expectedRequest := <-responses
		actualRequest := args.Get(1).(*orderer.SubmitRequest)
		assert.True(t, expectedRequest == actualRequest)
	})

	svc := &cluster.Service{
		Logger:     flogging.MustGetLogger("test"),
		StepLogger: flogging.MustGetLogger("test"),
		Dispatcher: dispatcher,
	}

	err := svc.Submit(stream)
	assert.NoError(t, err)
	dispatcher.AssertNumberOfCalls(t, "DispatchSubmit", 2)
}

type tuple struct {
	msg interface{}
	err error
}

func (t tuple) asArray() []interface{} {
	return []interface{}{t.msg, t.err}
}

func TestSubmitFailure(t *testing.T) {
	t.Parallel()
	oops := errors.New("oops")
	testCases := []struct {
		name               string
		receiveReturns     []tuple
		sendReturns        []error
		dispatchReturns    []interface{}
		expectedDispatches int
	}{
		{
			name: "Recv() fails",
			receiveReturns: []tuple{
				{msg: nil, err: oops},
			},
		},
		{
			name: "Send() fails",
			receiveReturns: []tuple{
				{msg: submitRequest1},
			},
			expectedDispatches: 1,
			dispatchReturns:    []interface{}{submitResponse1, nil},
			sendReturns:        []error{oops},
		},
		{
			name: "DispatchSubmit() fails",
			receiveReturns: []tuple{
				{msg: submitRequest1},
			},
			expectedDispatches: 1,
			dispatchReturns:    []interface{}{nil, oops},
		},
		{
			name: "Recv() and Send() succeed, and then Recv() fails",
			receiveReturns: []tuple{
				{msg: submitRequest1},
				{msg: nil, err: oops},
			},
			expectedDispatches: 1,
			dispatchReturns:    []interface{}{submitResponse1, nil},
			sendReturns:        []error{nil, nil},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			dispatcher := &mocks.Dispatcher{}
			stream := &mocks.SubmitStream{}
			stream.On("Context").Return(context.Background())
			for _, recv := range testCase.receiveReturns {
				stream.On("Recv").Return(recv.asArray()...).Once()
			}
			for _, send := range testCase.sendReturns {
				stream.On("Send", mock.Anything).Return(send).Once()
			}
			defer dispatcher.AssertNumberOfCalls(t, "DispatchSubmit", testCase.expectedDispatches)
			dispatcher.On("DispatchSubmit", mock.Anything, mock.Anything).Return(testCase.dispatchReturns...)
			svc := &cluster.Service{
				Logger:     flogging.MustGetLogger("test"),
				StepLogger: flogging.MustGetLogger("test"),
				Dispatcher: dispatcher,
			}
			err := svc.Submit(stream)
			assert.EqualError(t, err, oops.Error())
		})
	}
}

func TestServiceGRPC(t *testing.T) {
	t.Parallel()
//检查服务是否正确实现GRPC接口
	srv, err := comm.NewGRPCServer("127.0.0.1:0", comm.ServerConfig{})
	assert.NoError(t, err)
	orderer.RegisterClusterServer(srv.Server(), &cluster.Service{
		Logger:     flogging.MustGetLogger("test"),
		StepLogger: flogging.MustGetLogger("test"),
	})
}
