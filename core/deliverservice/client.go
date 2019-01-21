
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


package deliverclient

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"google.golang.org/grpc"
)

//broadcastSetup is a function that is called by the broadcastClient immediately after each
//成功连接到订购服务
type broadcastSetup func(blocksprovider.BlocksDeliverer) error

//RetryPolicy作为参数接收尝试失败的次数
//以及指定自第一次尝试以来经过的总时间的持续时间。
//如果需要进一步尝试，它将返回：
//-下一次尝试的持续时间，真
//否则，持续时间为零，错误
type retryPolicy func(attemptNum int, elapsedTime time.Duration) (time.Duration, bool)

//clientFactory从clientconn创建GRPC广播客户端
type clientFactory func(*grpc.ClientConn) orderer.AtomicBroadcastClient

type broadcastClient struct {
	stopFlag     int32
	stopChan     chan struct{}
	createClient clientFactory
	shouldRetry  retryPolicy
	onConnect    broadcastSetup
	prod         comm.ConnectionProducer

	mutex           sync.Mutex
	blocksDeliverer blocksprovider.BlocksDeliverer
	conn            *connection
	endpoint        string
}

//newbroadcastclient返回具有给定参数的广播客户端
func NewBroadcastClient(prod comm.ConnectionProducer, clFactory clientFactory, onConnect broadcastSetup, bos retryPolicy) *broadcastClient {
	return &broadcastClient{prod: prod, onConnect: onConnect, shouldRetry: bos, createClient: clFactory, stopChan: make(chan struct{}, 1)}
}

//recv从订购服务接收消息
func (bc *broadcastClient) Recv() (*orderer.DeliverResponse, error) {
	o, err := bc.try(func() (interface{}, error) {
		if bc.shouldStop() {
			return nil, errors.New("closing")
		}
		return bc.tryReceive()
	})
	if err != nil {
		return nil, err
	}
	return o.(*orderer.DeliverResponse), nil
}

//发送向订购服务发送消息
func (bc *broadcastClient) Send(msg *common.Envelope) error {
	_, err := bc.try(func() (interface{}, error) {
		if bc.shouldStop() {
			return nil, errors.New("closing")
		}
		return bc.trySend(msg)
	})
	return err
}

func (bc *broadcastClient) trySend(msg *common.Envelope) (interface{}, error) {
	bc.mutex.Lock()
	stream := bc.blocksDeliverer
	bc.mutex.Unlock()
	if stream == nil {
		return nil, errors.New("client stream has been closed")
	}
	return nil, stream.Send(msg)
}

func (bc *broadcastClient) tryReceive() (*orderer.DeliverResponse, error) {
	bc.mutex.Lock()
	stream := bc.blocksDeliverer
	bc.mutex.Unlock()
	if stream == nil {
		return nil, errors.New("client stream has been closed")
	}
	return stream.Recv()
}

func (bc *broadcastClient) try(action func() (interface{}, error)) (interface{}, error) {
	attempt := 0
	var totalRetryTime time.Duration
	var backoffDuration time.Duration
	retry := true
	resetAttemptCounter := func() {
		attempt = 0
		totalRetryTime = 0
	}
	for retry && !bc.shouldStop() {
		resp, err := bc.doAction(action, resetAttemptCounter)
		if err != nil {
			attempt++
			backoffDuration, retry = bc.shouldRetry(attempt, totalRetryTime)
			if !retry {
				logger.Warning("Got error:", err, "at", attempt, "attempt. Ceasing to retry")
				break
			}
			logger.Warning("Got error:", err, ", at", attempt, "attempt. Retrying in", backoffDuration)
			totalRetryTime += backoffDuration
			bc.sleep(backoffDuration)
			continue
		}
		return resp, nil
	}
	if bc.shouldStop() {
		return nil, errors.New("client is closing")
	}
	return nil, fmt.Errorf("attempts (%d) or elapsed time (%v) exhausted", attempt, totalRetryTime)
}

func (bc *broadcastClient) doAction(action func() (interface{}, error), actionOnNewConnection func()) (interface{}, error) {
	bc.mutex.Lock()
	conn := bc.conn
	bc.mutex.Unlock()
	if conn == nil {
		err := bc.connect()
		if err != nil {
			return nil, err
		}
		actionOnNewConnection()
	}
	resp, err := action()
	if err != nil {
		bc.Disconnect(false)
		return nil, err
	}
	return resp, nil
}

func (bc *broadcastClient) sleep(duration time.Duration) {
	select {
	case <-time.After(duration):
	case <-bc.stopChan:
	}
}

func (bc *broadcastClient) connect() error {
	bc.mutex.Lock()
	bc.endpoint = ""
	bc.mutex.Unlock()
	conn, endpoint, err := bc.prod.NewConnection()
	logger.Debug("Connected to", endpoint)
	if err != nil {
		logger.Error("Failed obtaining connection:", err)
		return err
	}
	ctx, cf := context.WithCancel(context.Background())
	logger.Debug("Establishing gRPC stream with", endpoint, "...")
	abc, err := bc.createClient(conn).Deliver(ctx)
	if err != nil {
		logger.Error("Connection to ", endpoint, "established but was unable to create gRPC stream:", err)
		conn.Close()
		cf()
		return err
	}
	err = bc.afterConnect(conn, abc, cf, endpoint)
	if err == nil {
		return nil
	}
	logger.Warning("Failed running post-connection procedures:", err)
//如果我们到达这里，请确保连接已关闭
//在我们回来之前就取消了
	bc.Disconnect(false)
	return err
}

func (bc *broadcastClient) afterConnect(conn *grpc.ClientConn, abc orderer.AtomicBroadcast_DeliverClient, cf context.CancelFunc, endpoint string) error {
	logger.Debug("Entering")
	defer logger.Debug("Exiting")
	bc.mutex.Lock()
	bc.endpoint = endpoint
	bc.conn = &connection{ClientConn: conn, cancel: cf}
	bc.blocksDeliverer = abc
	if bc.shouldStop() {
		bc.mutex.Unlock()
		return errors.New("closing")
	}
	bc.mutex.Unlock()
//如果客户机在此点关闭-在OnConnect之前，
//OnConnect对该对象的任何使用都将返回一个错误。
	err := bc.onConnect(bc)
//如果客户机在OnConnect之后，但在
//the following lock- this method would return an error because
//客户端已关闭。
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	if bc.shouldStop() {
		return errors.New("closing")
	}
//如果在该方法退出后立即关闭客户端，
//这是因为此方法返回了零而不是错误。
//所以-connect（）也将返回nil，以及goroutine的流
//返回到doaction（），在这里调用action（）-并配置
//检查客户端是否已关闭。
	if err == nil {
		return nil
	}
	logger.Error("Failed setting up broadcast:", err)
	return err
}

func (bc *broadcastClient) shouldStop() bool {
	return atomic.LoadInt32(&bc.stopFlag) == int32(1)
}

//关闭使客户端关闭其连接并关闭
func (bc *broadcastClient) Close() {
	logger.Debug("Entering")
	defer logger.Debug("Exiting")
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	if bc.shouldStop() {
		return
	}
	atomic.StoreInt32(&bc.stopFlag, int32(1))
	bc.stopChan <- struct{}{}
	if bc.conn == nil {
		return
	}
	bc.endpoint = ""
	bc.conn.Close()
}

//如果disableendpoint设置为true，则disconnect将使客户端关闭现有连接，并使当前终结点在时间间隔内不可用。
func (bc *broadcastClient) Disconnect(disableEndpoint bool) {
	logger.Debug("Entering")
	defer logger.Debug("Exiting")
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	if disableEndpoint && bc.endpoint != "" {
		bc.prod.DisableEndpoint(bc.endpoint)
	}
	bc.endpoint = ""
	if bc.conn == nil {
		return
	}
	bc.conn.Close()
	bc.conn = nil
	bc.blocksDeliverer = nil
}

//updateEndpoints将端点更新为新值
func (bc *broadcastClient) UpdateEndpoints(endpoints []string) {
	bc.prod.UpdateEndpoints(endpoints)
}

//GetEndpoints返回对服务终结点的排序
func (bc *broadcastClient) GetEndpoints() []string {
	return bc.prod.GetEndpoints()
}

type connection struct {
	sync.Once
	*grpc.ClientConn
	cancel context.CancelFunc
}

func (c *connection) Close() error {
	var err error
	c.Once.Do(func() {
		c.cancel()
		err = c.ClientConn.Close()
	})
	return err
}
