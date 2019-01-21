
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


package comm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/util"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

type handler func(message *proto.SignedGossipMessage)

type blockingBehavior bool

const (
	blockingSend    = blockingBehavior(true)
	nonBlockingSend = blockingBehavior(false)
)

type connFactory interface {
	createConnection(endpoint string, pkiID common.PKIidType) (*connection, error)
}

type connectionStore struct {
logger           util.Logger            //记录器
isClosing        bool                   //此连接存储是否正在关闭
connFactory      connFactory            //创建到远程对等机的连接
sync.RWMutex                            //同步访问共享变量
pki2Conn         map[string]*connection //pkiid到连接的映射
destinationLocks map[string]*sync.Mutex //pkiid和锁之间的映射，
//用于防止同时建立到同一远程端点的连接
}

func newConnStore(connFactory connFactory, logger util.Logger) *connectionStore {
	return &connectionStore{
		connFactory:      connFactory,
		isClosing:        false,
		pki2Conn:         make(map[string]*connection),
		destinationLocks: make(map[string]*sync.Mutex),
		logger:           logger,
	}
}

func (cs *connectionStore) getConnection(peer *RemotePeer) (*connection, error) {
	cs.RLock()
	isClosing := cs.isClosing
	cs.RUnlock()

	if isClosing {
		return nil, fmt.Errorf("Shutting down")
	}

	pkiID := peer.PKIID
	endpoint := peer.Endpoint

	cs.Lock()
	destinationLock, hasConnected := cs.destinationLocks[string(pkiID)]
	if !hasConnected {
		destinationLock = &sync.Mutex{}
		cs.destinationLocks[string(pkiID)] = destinationLock
	}
	cs.Unlock()

	destinationLock.Lock()

	cs.RLock()
	conn, exists := cs.pki2Conn[string(pkiID)]
	if exists {
		cs.RUnlock()
		destinationLock.Unlock()
		return conn, nil
	}
	cs.RUnlock()

	createdConnection, err := cs.connFactory.createConnection(endpoint, pkiID)

	destinationLock.Unlock()

	cs.RLock()
	isClosing = cs.isClosing
	cs.RUnlock()
	if isClosing {
		return nil, fmt.Errorf("ConnStore is closing")
	}

	cs.Lock()
	delete(cs.destinationLocks, string(pkiID))
	defer cs.Unlock()

//再次检查，是否有人在创建连接时与我们连接？
	conn, exists = cs.pki2Conn[string(pkiID)]

	if exists {
		if createdConnection != nil {
			createdConnection.close()
		}
		return conn, nil
	}

//没有人连接我们，我们连接失败！
	if err != nil {
		return nil, err
	}

//在代码的这一点上，我们创建了一个到远程对等机的连接
	conn = createdConnection
	cs.pki2Conn[string(createdConnection.pkiID)] = conn

	go conn.serviceConnection()

	return conn, nil
}

func (cs *connectionStore) connNum() int {
	cs.RLock()
	defer cs.RUnlock()
	return len(cs.pki2Conn)
}

func (cs *connectionStore) closeConn(peer *RemotePeer) {
	cs.Lock()
	defer cs.Unlock()

	if conn, exists := cs.pki2Conn[string(peer.PKIID)]; exists {
		conn.close()
		delete(cs.pki2Conn, string(conn.pkiID))
	}
}

func (cs *connectionStore) shutdown() {
	cs.Lock()
	cs.isClosing = true
	pkiIds2conn := cs.pki2Conn

	var connections2Close []*connection
	for _, conn := range pkiIds2conn {
		connections2Close = append(connections2Close, conn)
	}
	cs.Unlock()

	wg := sync.WaitGroup{}
	for _, conn := range connections2Close {
		wg.Add(1)
		go func(conn *connection) {
			cs.closeByPKIid(conn.pkiID)
			wg.Done()
		}(conn)
	}
	wg.Wait()
}

func (cs *connectionStore) onConnected(serverStream proto.Gossip_GossipStreamServer, connInfo *proto.ConnectionInfo) *connection {
	cs.Lock()
	defer cs.Unlock()

	if c, exists := cs.pki2Conn[string(connInfo.ID)]; exists {
		c.close()
	}

	return cs.registerConn(connInfo, serverStream)
}

func (cs *connectionStore) registerConn(connInfo *proto.ConnectionInfo, serverStream proto.Gossip_GossipStreamServer) *connection {
	conn := newConnection(nil, nil, nil, serverStream)
	conn.pkiID = connInfo.ID
	conn.info = connInfo
	conn.logger = cs.logger
	cs.pki2Conn[string(connInfo.ID)] = conn
	return conn
}

func (cs *connectionStore) closeByPKIid(pkiID common.PKIidType) {
	cs.Lock()
	defer cs.Unlock()
	if conn, exists := cs.pki2Conn[string(pkiID)]; exists {
		conn.close()
		delete(cs.pki2Conn, string(pkiID))
	}
}

func newConnection(cl proto.GossipClient, c *grpc.ClientConn, cs proto.Gossip_GossipStreamClient, ss proto.Gossip_GossipStreamServer) *connection {
	connection := &connection{
		outBuff:      make(chan *msgSending, util.GetIntOrDefault("peer.gossip.sendBuffSize", defSendBuffSize)),
		cl:           cl,
		conn:         c,
		clientStream: cs,
		serverStream: ss,
		stopFlag:     int32(0),
		stopChan:     make(chan struct{}, 1),
	}
	return connection
}

type connection struct {
	cancel       context.CancelFunc
	info         *proto.ConnectionInfo
	outBuff      chan *msgSending
logger       util.Logger                     //记录器
pkiID        common.PKIidType                //远程终结点的pkiid
handler      handler                         //接收消息时调用的函数
conn         *grpc.ClientConn                //到远程端点的GRPC连接
cl           proto.GossipClient              //远程端点的GRPC存根
clientStream proto.Gossip_GossipStreamClient //客户端流到远程终结点
serverStream proto.Gossip_GossipStreamServer //服务器端流到远程终结点
stopFlag     int32                           //指示此连接是否正在停止
stopChan     chan struct{}                   //从另一个go例程停止服务器端grpc调用的方法
sync.RWMutex                                 //同步访问共享变量
}

func (conn *connection) close() {
	if conn.toDie() {
		return
	}

	amIFirst := atomic.CompareAndSwapInt32(&conn.stopFlag, int32(0), int32(1))
	if !amIFirst {
		return
	}

	conn.stopChan <- struct{}{}

	conn.drainOutputBuffer()
	conn.Lock()
	defer conn.Unlock()

	if conn.clientStream != nil {
		conn.clientStream.CloseSend()
	}
	if conn.conn != nil {
		conn.conn.Close()
	}

	if conn.cancel != nil {
		conn.cancel()
	}
}

func (conn *connection) toDie() bool {
	return atomic.LoadInt32(&(conn.stopFlag)) == int32(1)
}

func (conn *connection) send(msg *proto.SignedGossipMessage, onErr func(error), shouldBlock blockingBehavior) {
	if conn.toDie() {
		conn.logger.Debug("Aborting send() to ", conn.info.Endpoint, "because connection is closing")
		return
	}

	m := &msgSending{
		envelope: msg.Envelope,
		onErr:    onErr,
	}

	if len(conn.outBuff) == cap(conn.outBuff) {
		if conn.logger.IsEnabledFor(zapcore.DebugLevel) {
			conn.logger.Debug("Buffer to", conn.info.Endpoint, "overflowed, dropping message", msg.String())
		}
		if !shouldBlock {
			return
		}
	}

	conn.outBuff <- m
}

func (conn *connection) serviceConnection() error {
	errChan := make(chan error, 1)
	msgChan := make(chan *proto.SignedGossipMessage, util.GetIntOrDefault("peer.gossip.recvBuffSize", defRecvBuffSize))
	quit := make(chan struct{})
//在readFromStream（）中异步调用stream.recv（），
//然后等待recv（）调用结束，
//或者关闭连接的信号，它退出
//方法并使recv（）调用在
//readFromStream（）方法
	go conn.readFromStream(errChan, quit, msgChan)

	go conn.writeToStream()

	for !conn.toDie() {
		select {
		case stop := <-conn.stopChan:
			conn.logger.Debug("Closing reading from stream")
			conn.stopChan <- stop
			return nil
		case err := <-errChan:
			return err
		case msg := <-msgChan:
			conn.handler(msg)
		}
	}
	return nil
}

func (conn *connection) writeToStream() {
	for !conn.toDie() {
		stream := conn.getStream()
		if stream == nil {
			conn.logger.Error(conn.pkiID, "Stream is nil, aborting!")
			return
		}
		select {
		case m := <-conn.outBuff:
			err := stream.Send(m.envelope)
			if err != nil {
				go m.onErr(err)
				return
			}
		case stop := <-conn.stopChan:
			conn.logger.Debug("Closing writing to stream")
			conn.stopChan <- stop
			return
		}
	}
}

func (conn *connection) drainOutputBuffer() {
//排空输出缓冲器
	for len(conn.outBuff) > 0 {
		<-conn.outBuff
	}
}

func (conn *connection) readFromStream(errChan chan error, quit chan struct{}, msgChan chan *proto.SignedGossipMessage) {
	for !conn.toDie() {
		stream := conn.getStream()
		if stream == nil {
			conn.logger.Error(conn.pkiID, "Stream is nil, aborting!")
			errChan <- fmt.Errorf("Stream is nil")
			return
		}
		envelope, err := stream.Recv()
		if conn.toDie() {
			conn.logger.Debug(conn.pkiID, "canceling read because closing")
			return
		}
		if err != nil {
			errChan <- err
			conn.logger.Debugf("Got error, aborting: %v", err)
			return
		}
		msg, err := envelope.ToGossipMessage()
		if err != nil {
			errChan <- err
			conn.logger.Warningf("Got error, aborting: %v", err)
		}
		select {
		case msgChan <- msg:
		case <-quit:
			return
		}
	}
}

func (conn *connection) getStream() stream {
	conn.Lock()
	defer conn.Unlock()

	if conn.clientStream != nil && conn.serverStream != nil {
		e := errors.New("Both client and server stream are not nil, something went wrong")
		conn.logger.Errorf("%+v", e)
	}

	if conn.clientStream != nil {
		return conn.clientStream
	}

	if conn.serverStream != nil {
		return conn.serverStream
	}

	return nil
}

type msgSending struct {
	envelope *proto.Envelope
	onErr    func(error)
}
