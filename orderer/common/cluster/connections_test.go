
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
	"sync"
	"testing"

	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/cluster/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestConcurrentConnections(t *testing.T) {
	t.Parallel()
//场景：让100个goroutine同时尝试创建连接，
//等到他们中的一个成功，然后等到他们都回来，
//并确保它们都返回相同的连接引用
	n := 100
	var wg sync.WaitGroup
	wg.Add(n)
	dialer := &mocks.SecureDialer{}
	conn := &grpc.ClientConn{}
	dialer.On("Dial", mock.Anything, mock.Anything).Return(conn, nil)
	connStore := cluster.NewConnectionStore(dialer)
	connect := func() {
		defer wg.Done()
		conn2, err := connStore.Connection("", nil)
		assert.NoError(t, err)
		assert.True(t, conn2 == conn)
	}
	for i := 0; i < n; i++ {
		go connect()
	}
	wg.Wait()
	dialer.AssertNumberOfCalls(t, "Dial", 1)
}

type connectionMapperSpy struct {
	lookupDelay   chan struct{}
	lookupInvoked chan struct{}
	cluster.ConnectionMapper
}

func (cms *connectionMapperSpy) Lookup(cert []byte) (*grpc.ClientConn, bool) {
//已调用lookup（）的信号
	cms.lookupInvoked <- struct{}{}
//等待主测试发出前进信号。
//这是必需的，因为我们需要确保所有实例
//ConnectionMapperSpy调用的查找（）的
	<-cms.lookupDelay
	return cms.ConnectionMapper.Lookup(cert)
}

func TestConcurrentLookupMiss(t *testing.T) {
	t.Parallel()
//场景：进行2次并发连接尝试，
//前2个查找操作被延迟，
//这使得连接存储尝试连接
//同时两次。
//不管怎样，都应该创建一个连接。

	dialer := &mocks.SecureDialer{}
	conn := &grpc.ClientConn{}
	dialer.On("Dial", mock.Anything, mock.Anything).Return(conn, nil)

	connStore := cluster.NewConnectionStore(dialer)
//使用拦截lookup（）调用的间谍包装连接映射
	spy := &connectionMapperSpy{
		ConnectionMapper: connStore.Connections,
		lookupDelay:      make(chan struct{}, 2),
		lookupInvoked:    make(chan struct{}, 2),
	}
	connStore.Connections = spy

	var goroutinesExited sync.WaitGroup
	goroutinesExited.Add(2)

	for i := 0; i < 2; i++ {
		go func() {
			defer goroutinesExited.Done()
			conn2, err := connStore.Connection("", nil)
			assert.NoError(t, err)
//确保所有connection（）调用返回相同的引用
//GRPC连接。
			assert.True(t, conn2 == conn)
		}()
	}
//等待两者都调用lookup（）。
//虎尾鹦鹉
	<-spy.lookupInvoked
	<-spy.lookupInvoked
//向Goroutines发出信号以完成lookup（）调用
	spy.lookupDelay <- struct{}{}
	spy.lookupDelay <- struct{}{}
//关闭通道，以便不会阻止后续的lookup（）操作
	close(spy.lookupDelay)
//等待所有Goroutine退出
	goroutinesExited.Wait()
}
