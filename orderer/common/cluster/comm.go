
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


package cluster

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
)

const (
//DefaultRpcTimeout是默认的RPC超时
//RPCs使用
	DefaultRPCTimeout = time.Second * 5
)

//channelextractor提取给定消息的通道，
//或者返回一个空字符串（如果不可能的话）
type ChannelExtractor interface {
	TargetChannel(message proto.Message) string
}

//去：生成mokery-dir。-名称处理程序-大小写下划线-输出/模拟/

//处理程序处理step（）和submit（）请求并返回相应的响应
type Handler interface {
	OnStep(channel string, sender uint64, req *orderer.StepRequest) (*orderer.StepResponse, error)
	OnSubmit(channel string, sender uint64, req *orderer.SubmitRequest) (*orderer.SubmitResponse, error)
}

//远程节点表示群集成员
type RemoteNode struct {
//ID在所有成员中都是唯一的，不能为0。
	ID uint64
//Endpoint是节点的终结点，用%s:%d格式表示
	Endpoint string
//servertlscert是节点的der编码的tls服务器证书
	ServerTLSCert []byte
//clienttlscert是节点的der编码的tls客户端证书
	ClientTLSCert []byte
}

//字符串返回此远程节点的字符串表示形式
func (rm RemoteNode) String() string {
	return fmt.Sprintf("ID: %d\nEndpoint: %s\nServerTLSCert:%s ClientTLSCert:%s",
		rm.ID, rm.Endpoint, DERtoPEM(rm.ServerTLSCert), DERtoPEM(rm.ClientTLSCert))
}

//去：生成mokery-dir。-name communicator-case underline-output./mocks/

//Communicator为同意者定义通信
type Communicator interface {
//远程返回上下文中给定远程节点ID的远程上下文
//指定通道的错误，或无法建立连接时的错误，或
//未配置频道
	Remote(channel string, id uint64) (*RemoteContext, error)
//配置将通信配置为连接到所有
//给定成员，并断开与给定成员以外的任何成员的连接
//成员。
	Configure(channel string, members []RemoteNode)
//关机关闭通讯器
	Shutdown()
}

//membersByChannel是来自通道名称的映射
//到成员映射
type MembersByChannel map[string]MemberMapping

//通信实现通讯器
type Comm struct {
	shutdown     bool
	Lock         sync.RWMutex
	Logger       *flogging.FabricLogger
	ChanExt      ChannelExtractor
	H            Handler
	Connections  *ConnectionStore
	Chan2Members MembersByChannel
	RPCTimeout   time.Duration
}

type requestContext struct {
	channel string
	sender  uint64
}

//DispatchSubmit标识提交请求的通道和发送者并传递它
//到基础处理程序
func (c *Comm) DispatchSubmit(ctx context.Context, request *orderer.SubmitRequest) (*orderer.SubmitResponse, error) {
	c.Logger.Debug(request.Channel)
	reqCtx, err := c.requestContext(ctx, request)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c.H.OnSubmit(reqCtx.channel, reqCtx.sender, request)
}

//DispatchStep标识步骤请求的通道和发送方并传递它
//到基础处理程序
func (c *Comm) DispatchStep(ctx context.Context, request *orderer.StepRequest) (*orderer.StepResponse, error) {
	reqCtx, err := c.requestContext(ctx, request)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c.H.OnStep(reqCtx.channel, reqCtx.sender, request)
}

//ClassifierRequest标识请求和返回的发送方和通道
//它包装在一个请求上下文中
func (c *Comm) requestContext(ctx context.Context, msg proto.Message) (*requestContext, error) {
	channel := c.ChanExt.TargetChannel(msg)
	if channel == "" {
		return nil, errors.Errorf("badly formatted message, cannot extract channel")
	}
	c.Lock.RLock()
	mapping, exists := c.Chan2Members[channel]
	c.Lock.RUnlock()

	if !exists {
		return nil, errors.Errorf("channel %s doesn't exist", channel)
	}

	cert := comm.ExtractCertificateFromContext(ctx)
	if len(cert) == 0 {
		return nil, errors.Errorf("no TLS certificate sent")
	}
	stub := mapping.LookupByClientCert(cert)
	if stub == nil {
		return nil, errors.Errorf("certificate extracted from TLS connection isn't authorized")
	}
	return &requestContext{
		channel: channel,
		sender:  stub.ID,
	}, nil
}

//远程获取链接到上下文上目标节点的远程上下文
//给定通道的
func (c *Comm) Remote(channel string, id uint64) (*RemoteContext, error) {
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	if c.shutdown {
		return nil, errors.New("communication has been shut down")
	}

	mapping, exists := c.Chan2Members[channel]
	if !exists {
		return nil, errors.Errorf("channel %s doesn't exist", channel)
	}
	stub := mapping.ByID(id)
	if stub == nil {
		return nil, errors.Errorf("node %d doesn't exist in channel %s's membership", id, channel)
	}

	if stub.Active() {
		return stub.RemoteContext, nil
	}

	err := stub.Activate(c.createRemoteContext(stub))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return stub.RemoteContext, nil
}

//配置使用给定的远程节点配置通道
func (c *Comm) Configure(channel string, newNodes []RemoteNode) {
	c.Logger.Infof("Entering, channel: %s, nodes: %v", channel, newNodes)
	defer c.Logger.Infof("Exiting")

	c.Lock.Lock()
	defer c.Lock.Unlock()

	if c.shutdown {
		return
	}

	beforeConfigChange := c.serverCertsInUse()
//使用新节点更新通道范围映射
	c.applyMembershipConfig(channel, newNodes)
//关闭与新成员身份中不存在的节点的连接
	c.cleanUnusedConnections(beforeConfigChange)
}

//shutdown关闭实例
func (c *Comm) Shutdown() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.shutdown = true
	for _, members := range c.Chan2Members {
		for _, member := range members {
			c.Connections.Disconnect(member.ServerTLSCert)
		}
	}
}

//清除未使用的连接断开所有未使用的连接
//在召唤的时刻
func (c *Comm) cleanUnusedConnections(serverCertsBeforeConfig StringSet) {
//重新配置后扫描所有节点
	serverCertsAfterConfig := c.serverCertsInUse()
//过滤掉重新配置后保留的证书
	serverCertsBeforeConfig.subtract(serverCertsAfterConfig)
//关闭所有这些节点的连接，因为它们现在不应该使用
	for serverCertificate := range serverCertsBeforeConfig {
		c.Connections.Disconnect([]byte(serverCertificate))
	}
}

//servercertsine返回正在使用的服务器证书
//表示为字符串。
func (c *Comm) serverCertsInUse() StringSet {
	endpointsInUse := make(StringSet)
	for _, mapping := range c.Chan2Members {
		endpointsInUse.union(mapping.ServerCertificates())
	}
	return endpointsInUse
}

//applymembershipconfig为给定通道设置给定的远程节点
func (c *Comm) applyMembershipConfig(channel string, newNodes []RemoteNode) {
	mapping := c.getOrCreateMapping(channel)
	newNodeIDs := make(map[uint64]struct{})

	for _, node := range newNodes {
		newNodeIDs[node.ID] = struct{}{}
		c.updateStubInMapping(mapping, node)
	}

//删除没有相应节点的所有存根
//在新节点中
	for id, stub := range mapping {
		if _, exists := newNodeIDs[id]; exists {
			c.Logger.Info(id, "exists in both old and new membership, skipping its deactivation")
			continue
		}
		c.Logger.Info("Deactivated node", id, "who's endpoint is", stub.Endpoint, "as it's removed from membership")
		delete(mapping, id)
		stub.Deactivate()
	}
}

//updatestubinmapping更新给定的远程节点并将其添加到membermapping
func (c *Comm) updateStubInMapping(mapping MemberMapping, node RemoteNode) {
	stub := mapping.ByID(node.ID)
	if stub == nil {
		c.Logger.Info("Allocating a new stub for node", node.ID, "with endpoint of", node.Endpoint)
		stub = &Stub{}
	}

//检查节点的TLS服务器证书是否被替换
//如果是这样的话-那么关闭存根，触发
//重新创建其GRPC连接
	if !bytes.Equal(stub.ServerTLSCert, node.ServerTLSCert) {
		c.Logger.Info("Deactivating node", node.ID, "with endpoint of", node.Endpoint, "due to TLS certificate change")
		stub.Deactivate()
	}

//用新数据覆盖存根节点数据
	stub.RemoteNode = node

//将存根放入映射中
	mapping.Put(stub)

//
	if stub.Active() {
		return
	}

//激活存根
	stub.Activate(c.createRemoteContext(stub))
}

//CreateRemoteStub返回一个创建远程上下文的函数。
//它用作stub.activate（）的参数，以便激活
//原子性的树桩。
func (c *Comm) createRemoteContext(stub *Stub) func() (*RemoteContext, error) {
	return func() (*RemoteContext, error) {
		timeout := c.RPCTimeout
		if timeout == time.Duration(0) {
			timeout = DefaultRPCTimeout
		}

		c.Logger.Debug("Connecting to", stub.RemoteNode, "with gRPC timeout of", timeout)

		conn, err := c.Connections.Connection(stub.Endpoint, stub.ServerTLSCert)
		if err != nil {
			c.Logger.Warningf("Unable to obtain connection to %d(%s): %v", stub.ID, stub.Endpoint, err)
			return nil, err
		}

		clusterClient := orderer.NewClusterClient(conn)

		rc := &RemoteContext{
			RPCTimeout: timeout,
			Client:     clusterClient,
			onAbort: func() {
				c.Logger.Info("Aborted connection to", stub.ID, stub.Endpoint)
				stub.RemoteContext = nil
			},
		}
		return rc, nil
	}
}

//getorCreateMapping为给定通道创建一个成员映射
//或者返回现有的。
func (c *Comm) getOrCreateMapping(channel string) MemberMapping {
//如果一个映射还不存在，就懒散地创建它。
	mapping, exists := c.Chan2Members[channel]
	if !exists {
		mapping = make(MemberMapping)
		c.Chan2Members[channel] = mapping
	}
	return mapping
}

//存根保存有关远程节点的所有信息，
//包括它的远程上下文，并序列化
//一些操作。
type Stub struct {
	lock sync.RWMutex
	RemoteNode
	*RemoteContext
}

//active返回存根
//是否有效
func (stub *Stub) Active() bool {
	stub.lock.RLock()
	defer stub.lock.RUnlock()
	return stub.isActive()
}

//active返回存根
//是否激活。
func (stub *Stub) isActive() bool {
	return stub.RemoteContext != nil
}

//停用停用存根并
//停止所有通信操作
//调用它。
func (stub *Stub) Deactivate() {
	stub.lock.Lock()
	defer stub.lock.Unlock()
	if !stub.isActive() {
		return
	}
	stub.RemoteContext.Abort()
	stub.RemoteContext = nil
}

//activate使用给定的函数回调创建远程上下文
//以原子方式-如果在此存根上调用两个并行调用，
//只有一次调用CreateRemoteStub。
func (stub *Stub) Activate(createRemoteContext func() (*RemoteContext, error)) error {
	stub.lock.Lock()
	defer stub.lock.Unlock()
//在等待锁的时候检查存根是否已经被激活
	if stub.isActive() {
		return nil
	}
	remoteStub, err := createRemoteContext()
	if err != nil {
		return errors.WithStack(err)
	}

	stub.RemoteContext = remoteStub
	return nil
}

//远程上下文与远程群集交互
//节点。每个调用都可以通过调用abort（）中止
type RemoteContext struct {
	RPCTimeout         time.Duration
	onAbort            func()
	Client             orderer.ClusterClient
	stepLock           sync.Mutex
	cancelStep         func()
	submitLock         sync.Mutex
	cancelSubmitStream func()
	submitStream       orderer.Cluster_SubmitClient
}

//SubmitStream创建新的提交流
func (rc *RemoteContext) SubmitStream() (orderer.Cluster_SubmitClient, error) {
	rc.submitLock.Lock()
	defer rc.submitLock.Unlock()
//关闭上一个提交流以防止资源泄漏
	rc.closeSubmitStream()

	ctx, cancel := context.WithCancel(context.TODO())
	submitStream, err := rc.Client.Submit(ctx)
	if err != nil {
		cancel()
		return nil, errors.WithStack(err)
	}
	rc.submitStream = submitStream
	rc.cancelSubmitStream = cancel
	return rc.submitStream, nil
}

//步骤将特定于实现的消息传递给另一个集群成员。
func (rc *RemoteContext) Step(req *orderer.StepRequest) (*orderer.StepResponse, error) {
	ctx, abort := context.WithCancel(context.TODO())
	ctx, cancel := context.WithTimeout(ctx, rc.RPCTimeout)
	defer cancel()

	rc.stepLock.Lock()
	rc.cancelStep = abort
	rc.stepLock.Unlock()

	return rc.Client.Step(ctx, req)
}

//abort中止远程上下文使用的上下文，
//从而有效地导致嵌入式系统上的所有操作
//clusterclient结束。
func (rc *RemoteContext) Abort() {
	rc.stepLock.Lock()
	defer rc.stepLock.Unlock()

	rc.submitLock.Lock()
	defer rc.submitLock.Unlock()

	if rc.cancelStep != nil {
		rc.cancelStep()
		rc.cancelStep = nil
	}

	rc.closeSubmitStream()
	rc.onAbort()
}

//CloseSubmitStream关闭提交流
//并调用其取消函数
func (rc *RemoteContext) closeSubmitStream() {
	if rc.cancelSubmitStream != nil {
		rc.cancelSubmitStream()
		rc.cancelSubmitStream = nil
	}

	if rc.submitStream != nil {
		rc.submitStream.CloseSend()
		rc.submitStream = nil
	}
}
