
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM Corp.2018保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package cluster_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/flogging"
	false_crypto "github.com/hyperledger/fabric/common/mocks/crypto"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/roundrobin"
)

//保护GRPC平衡器注册
var gRPCBalancerLock = sync.Mutex{}

func init() {
	factory.InitFactories(nil)
}

type wrappedBalancer struct {
	balancer.Balancer
	cd *countingDialer
}

func (wb *wrappedBalancer) Close() {
	defer atomic.AddUint32(&wb.cd.connectionCount, ^uint32(0))
	wb.Balancer.Close()
}

type countingDialer struct {
	name            string
	baseBuilder     balancer.Builder
	connectionCount uint32
}

func newCountingDialer() *countingDialer {
	gRPCBalancerLock.Lock()
	builder := balancer.Get(roundrobin.Name)
	gRPCBalancerLock.Unlock()

	buff := make([]byte, 16)
	rand.Read(buff)
	cb := &countingDialer{
		name:        hex.EncodeToString(buff),
		baseBuilder: builder,
	}

	gRPCBalancerLock.Lock()
	balancer.Register(cb)
	gRPCBalancerLock.Unlock()

	return cb
}

func (d *countingDialer) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	defer atomic.AddUint32(&d.connectionCount, 1)
	return &wrappedBalancer{Balancer: d.baseBuilder.Build(cc, opts), cd: d}
}

func (d *countingDialer) Name() string {
	return d.name
}

func (d *countingDialer) assertAllConnectionsClosed(t *testing.T) {
	timeLimit := time.Now().Add(timeout)
	for atomic.LoadUint32(&d.connectionCount) != uint32(0) && time.Now().Before(timeLimit) {
		time.Sleep(time.Millisecond)
	}
	assert.Equal(t, uint32(0), atomic.LoadUint32(&d.connectionCount))
}

func (d *countingDialer) Dial(address string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	gRPCBalancerLock.Lock()
	balancer := grpc.WithBalancerName(d.name)
	gRPCBalancerLock.Unlock()
	return grpc.DialContext(ctx, address, grpc.WithBlock(), grpc.WithInsecure(), balancer)
}

func noopBlockVerifierf(_ []*common.Block) error {
	return nil
}

func readSeekEnvelope(stream orderer.AtomicBroadcast_DeliverServer) (*orderer.SeekInfo, string, error) {
	env, err := stream.Recv()
	if err != nil {
		return nil, "", err
	}
	payload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, "", err
	}
	seekInfo := &orderer.SeekInfo{}
	if err = proto.Unmarshal(payload.Data, seekInfo); err != nil {
		return nil, "", err
	}
	chdr := &common.ChannelHeader{}
	if err = proto.Unmarshal(payload.Header.ChannelHeader, chdr); err != nil {
		return nil, "", err
	}
	return seekInfo, chdr.ChannelId, nil
}

type deliverServer struct {
	t *testing.T
	sync.Mutex
	err            error
	srv            *comm.GRPCServer
	seekAssertions chan func(*orderer.SeekInfo, string)
	blockResponses chan *orderer.DeliverResponse
}

func (ds *deliverServer) isFaulty() bool {
	ds.Lock()
	defer ds.Unlock()
	return ds.err != nil
}

func (*deliverServer) Broadcast(orderer.AtomicBroadcast_BroadcastServer) error {
	panic("implement me")
}

func (ds *deliverServer) Deliver(stream orderer.AtomicBroadcast_DeliverServer) error {
	ds.Lock()
	err := ds.err
	ds.Unlock()

	if err != nil {
		return err
	}
	seekInfo, channel, err := readSeekEnvelope(stream)
	if err != nil {
		panic(err)
	}
//获取NextSeek断言并确保NextSeek是预期的类型
	seekAssert := <-ds.seekAssertions
	seekAssert(seekInfo, channel)

	if seekInfo.GetStart().GetSpecified() != nil {
		return ds.deliverBlocks(stream)
	}
	if seekInfo.GetStart().GetNewest() != nil {
		resp := <-ds.blocks()
		return stream.Send(resp)
	}
	panic(fmt.Sprintf("expected either specified or newest seek but got %v", seekInfo.GetStart()))
}

func (ds *deliverServer) deliverBlocks(stream orderer.AtomicBroadcast_DeliverServer) error {
	for {
		blockChan := ds.blocks()
		response := <-blockChan
//零响应是测试发出的关闭流的信号。
//这是为了避免从块缓冲区读取，因此
//不小心吃了一块被压住要拉的木块
//稍后的测试。
		if response == nil {
			return nil
		}
		if err := stream.Send(response); err != nil {
			return err
		}
	}
}

func (ds *deliverServer) blocks() chan *orderer.DeliverResponse {
	ds.Lock()
	defer ds.Unlock()
	blockChan := ds.blockResponses
	return blockChan
}

func (ds *deliverServer) setBlocks(blocks chan *orderer.DeliverResponse) {
	ds.Lock()
	defer ds.Unlock()
	ds.blockResponses = blocks
}

func (ds *deliverServer) port() int {
	_, portStr, err := net.SplitHostPort(ds.srv.Address())
	assert.NoError(ds.t, err)

	port, err := strconv.ParseInt(portStr, 10, 32)
	assert.NoError(ds.t, err)
	return int(port)
}

func (ds *deliverServer) resurrect() {
	var err error
//将响应通道复制到新通道中
	respChan := make(chan *orderer.DeliverResponse, 100)
	for resp := range ds.blocks() {
		respChan <- resp
	}
	ds.blockResponses = respChan
	ds.srv.Stop()
//并在该端口上重新创建GRPC服务器
	ds.srv, err = comm.NewGRPCServer(fmt.Sprintf("127.0.0.1:%d", ds.port()), comm.ServerConfig{})
	assert.NoError(ds.t, err)
	orderer.RegisterAtomicBroadcastServer(ds.srv.Server(), ds)
	go ds.srv.Start()
}

func (ds *deliverServer) stop() {
	ds.srv.Stop()
	close(ds.blocks())
}

func (ds *deliverServer) enqueueResponse(seq uint64) {
	ds.blocks() <- &orderer.DeliverResponse{
		Type: &orderer.DeliverResponse_Block{Block: common.NewBlock(seq, nil)},
	}
}

func (ds *deliverServer) addExpectProbeAssert() {
	ds.seekAssertions <- func(info *orderer.SeekInfo, _ string) {
		assert.NotNil(ds.t, info.GetStart().GetNewest())
	}
}

func (ds *deliverServer) addExpectPullAssert(seq uint64) {
	ds.seekAssertions <- func(info *orderer.SeekInfo, _ string) {
		assert.NotNil(ds.t, info.GetStart().GetSpecified())
		assert.Equal(ds.t, seq, info.GetStart().GetSpecified().Number)
	}
}

func newClusterNode(t *testing.T) *deliverServer {
	srv, err := comm.NewGRPCServer("127.0.0.1:0", comm.ServerConfig{})
	if err != nil {
		panic(err)
	}
	ds := &deliverServer{
		t:              t,
		seekAssertions: make(chan func(*orderer.SeekInfo, string), 100),
		blockResponses: make(chan *orderer.DeliverResponse, 100),
		srv:            srv,
	}
	orderer.RegisterAtomicBroadcastServer(srv.Server(), ds)
	go srv.Start()
	return ds
}

func newBlockPuller(dialer *countingDialer, orderers ...string) *cluster.BlockPuller {
	return &cluster.BlockPuller{
		Dialer:              dialer,
		Channel:             "mychannel",
		Signer:              &false_crypto.LocalSigner{},
		Endpoints:           orderers,
		FetchTimeout:        time.Second,
MaxTotalBufferBytes: 1024 * 1024, //1MB
		RetryTimeout:        time.Millisecond * 10,
		VerifyBlockSequence: noopBlockVerifierf,
		Logger:              flogging.MustGetLogger("test"),
	}
}

func TestBlockPullerBasicHappyPath(t *testing.T) {
//场景：单订货节点，
//拉块器拉块5到10。
	osn := newClusterNode(t)
	defer osn.stop()

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn.srv.Address())

//第一个SEEK请求请求请求最新的块
	osn.addExpectProbeAssert()
//第一个反应是高度是10
	osn.enqueueResponse(10)
//下一个搜索请求是针对块5的
	osn.addExpectPullAssert(5)
//后来的反应是障碍本身
	for i := 5; i <= 10; i++ {
		osn.enqueueResponse(uint64(i))
	}

	for i := 5; i <= 10; i++ {
		assert.Equal(t, uint64(i), bp.PullBlock(uint64(i)).Header.Number)
	}
	assert.Len(t, osn.blockResponses, 0)

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerDuplicate(t *testing.T) {
//场景：订购节点的地址
//在配置中找到两次，但是
//不会造成问题。
	osn := newClusterNode(t)
	defer osn.stop()

	dialer := newCountingDialer()
//添加地址两次
	bp := newBlockPuller(dialer, osn.srv.Address(), osn.srv.Address())

//第一个SEEK请求请求最新的块（两次）
	osn.addExpectProbeAssert()
	osn.addExpectProbeAssert()
//第一个反应是高度为3
	osn.enqueueResponse(3)
	osn.enqueueResponse(3)
//下一个SEEK请求是针对块1的，仅来自某些OSN
	osn.addExpectPullAssert(1)
//后来的反应是障碍本身
	for i := 1; i <= 3; i++ {
		osn.enqueueResponse(uint64(i))
	}

	for i := 1; i <= 3; i++ {
		assert.Equal(t, uint64(i), bp.PullBlock(uint64(i)).Header.Number)
	}
	assert.Len(t, osn.blockResponses, 0)

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerHeavyBlocks(t *testing.T) {
//场景：单订货节点，
//拉块器每拉50块
//重1K，但缓冲器只能容纳
//10公里，所以应该把50个街区分成10个街区，
//一次验证5个序列。

	osn := newClusterNode(t)
	defer osn.stop()
	osn.addExpectProbeAssert()
	osn.addExpectPullAssert(1)
osn.enqueueResponse(100) //最后一个序列是100

	enqueueBlockBatch := func(start, end uint64) {
		for seq := start; seq <= end; seq++ {
			resp := &orderer.DeliverResponse{
				Type: &orderer.DeliverResponse_Block{
					Block: common.NewBlock(seq, nil),
				},
			}
			data := resp.GetBlock().Data.Data
			resp.GetBlock().Data.Data = append(data, make([]byte, 1024))
			osn.blockResponses <- resp
		}
	}

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn.srv.Address())
	var gotBlockMessageCount int
	bp.Logger = bp.Logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if strings.Contains(entry.Message, "Got block") {
			gotBlockMessageCount++
		}
		return nil
	}))
bp.MaxTotalBufferBytes = 1024 * 10 //10K

//仅将下一批排入医嘱者节点。
//这样可以确保只有10个块被提取到缓冲区中。
//而不是更多。
	for i := uint64(0); i < 5; i++ {
		enqueueBlockBatch(i*10+uint64(1), i*10+uint64(10))
		for seq := i*10 + uint64(1); seq <= i*10+uint64(10); seq++ {
			assert.Equal(t, seq, bp.PullBlock(seq).Header.Number)
		}
	}

	assert.Equal(t, 50, gotBlockMessageCount)
	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerClone(t *testing.T) {
//场景：我们有一个连接了
//到一个排序节点，然后我们克隆它。
//我们希望新的拉块器是干净的。
//并且不与它的起源共享任何内部状态。
	osn1 := newClusterNode(t)
	defer osn1.stop()

	osn1.addExpectProbeAssert()
	osn1.addExpectPullAssert(1)
//最后一个块序列是100
	osn1.enqueueResponse(100)
	osn1.enqueueResponse(1)
//拉块器应在拉完后断开。
//一个街区。因此，向服务器端发送信号以避免
//在拉动块1后抓住下一个块。
	osn1.blockResponses <- nil

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn1.srv.Address())
	bp.FetchTimeout = time.Millisecond * 100
//一次拉一块，不要缓冲。
	bp.MaxTotalBufferBytes = 1
//克隆拉块器
	bpClone := bp.Clone()
//并覆盖其频道
	bpClone.Channel = "foo"
//确保通道更改不会反映在原始拉具中。
	assert.Equal(t, "mychannel", bp.Channel)

	block := bp.PullBlock(1)
	assert.Equal(t, uint64(1), block.Header.Number)

//关闭原始拉块器后，
//克隆不应受到影响
	bp.Close()
	dialer.assertAllConnectionsClosed(t)

//克隆块拉具不应缓存内部状态
//从它的起始块拉具，因此它应该再次探测最后一个块
//按顺序排列，就好像它是一个新的拉块器。
	osn1.addExpectProbeAssert()
	osn1.addExpectPullAssert(2)
	osn1.enqueueResponse(100)
	osn1.enqueueResponse(2)

	block = bpClone.PullBlock(2)
	assert.Equal(t, uint64(2), block.Header.Number)

	bpClone.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerHeightsByEndpoints(t *testing.T) {
//场景：我们请求来自所有已知排序节点的最新块。
//一个订购节点不可用（脱机）。
//一个排序节点没有该通道的块。
//其余节点返回最新的块。
	osn1 := newClusterNode(t)

	osn2 := newClusterNode(t)
	defer osn2.stop()

	osn3 := newClusterNode(t)
	defer osn3.stop()

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn1.srv.Address(), osn2.srv.Address(), osn3.srv.Address())

//第一个SEEK请求从某个排序节点请求最新的块
	osn1.addExpectProbeAssert()
	osn2.addExpectProbeAssert()
	osn3.addExpectProbeAssert()

//第一个订购节点脱机
	osn1.stop()
//第二个回答错误
	osn2.blockResponses <- &orderer.DeliverResponse{
		Type: &orderer.DeliverResponse_Status{Status: common.Status_FORBIDDEN},
	}
//第三个返回最新的块
	osn3.enqueueResponse(5)

	res := bp.HeightsByEndpoints()
	expected := map[string]uint64{
		osn3.srv.Address(): 6,
	}
	assert.Equal(t, expected, res)

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerMultipleOrderers(t *testing.T) {
//场景：3个订购节点，
//拉块器从一些
//订购者节点。
//我们确保只有一个订购者节点发送数据块。

	osn1 := newClusterNode(t)
	defer osn1.stop()

	osn2 := newClusterNode(t)
	defer osn2.stop()

	osn3 := newClusterNode(t)
	defer osn3.stop()

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn1.srv.Address(), osn2.srv.Address(), osn3.srv.Address())

//第一个SEEK请求从某个排序节点请求最新的块
	osn1.addExpectProbeAssert()
	osn2.addExpectProbeAssert()
	osn3.addExpectProbeAssert()
//第一个反应是高度为5
	osn1.enqueueResponse(5)
	osn2.enqueueResponse(5)
	osn3.enqueueResponse(5)

//下一个搜索请求是针对块3的
	osn1.addExpectPullAssert(3)
	osn2.addExpectPullAssert(3)
	osn3.addExpectPullAssert(3)

//后来的反应是障碍本身
	for i := 3; i <= 5; i++ {
		osn1.enqueueResponse(uint64(i))
		osn2.enqueueResponse(uint64(i))
		osn3.enqueueResponse(uint64(i))
	}

	initialTotalBlockAmount := len(osn1.blockResponses) + len(osn2.blockResponses) + len(osn3.blockResponses)

	for i := 3; i <= 5; i++ {
		assert.Equal(t, uint64(i), bp.PullBlock(uint64(i)).Header.Number)
	}

//断言OSN中块的累计数量下降了6：
//第3、4、5个街区被拉了——那是3个街区。
//5号区块在探测阶段被拉了3次。
	finalTotalBlockAmount := len(osn1.blockResponses) + len(osn2.blockResponses) + len(osn3.blockResponses)
	assert.Equal(t, initialTotalBlockAmount-6, finalTotalBlockAmount)

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerFailover(t *testing.T) {
//脚本：
//预计拉块器将拉块1至3。
//有两个排序节点，但一开始只有节点1可用。
//拉块器首先与之相连，但当它拉动第一块时，
//它崩溃了。
//然后生成第二个排序器，并期望块拉具
//连接到它并拉动其余的块。

	osn1 := newClusterNode(t)
	osn1.addExpectProbeAssert()
	osn1.addExpectPullAssert(1)
	osn1.enqueueResponse(3)
	osn1.enqueueResponse(1)

	osn2 := newClusterNode(t)
	defer osn2.stop()

	osn2.addExpectProbeAssert()
	osn2.addExpectPullAssert(2)
//第一个反应是探针
	osn2.enqueueResponse(3)
//接下来的两个响应是拉动，而第一个块
//被跳过，因为它应该是从节点1检索到的
	osn2.enqueueResponse(2)
	osn2.enqueueResponse(3)

//首先，我们停止节点2以确保块拉具在创建时无法连接到它。
	osn2.stop()

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn1.srv.Address(), osn2.srv.Address())

//我们不想依赖于获取超时，而是纯故障转移逻辑，
//所以让获取超时变得很大
	bp.FetchTimeout = time.Hour

//配置块拉器记录器，以便在块拉器出现后向等待组发送信号
//收到第一个块。
	var pulledBlock1 sync.WaitGroup
	pulledBlock1.Add(1)
	bp.Logger = bp.Logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if strings.Contains(entry.Message, "Got block 1 of size") {
			pulledBlock1.Done()
		}
		return nil
	}))

	go func() {
//等待拉块器拉动第一个块
		pulledBlock1.Wait()
//现在，崩溃节点1并恢复节点2
		osn1.stop()
		osn2.resurrect()
	}()

//断言接收1至3区
	assert.Equal(t, uint64(1), bp.PullBlock(uint64(1)).Header.Number)
	assert.Equal(t, uint64(2), bp.PullBlock(uint64(2)).Header.Number)
	assert.Equal(t, uint64(3), bp.PullBlock(uint64(3)).Header.Number)

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerNoneResponsiveOrderer(t *testing.T) {
//场景：有两个排序节点和块拉具
//连接到其中一个。
//它获取第一个块，但第二个块没有发送
//时间太长，拉块器应该中止，然后尝试另一个。
//节点。

	osn1 := newClusterNode(t)
	defer osn1.stop()

	osn2 := newClusterNode(t)
	defer osn2.stop()

	osn1.addExpectProbeAssert()
	osn2.addExpectProbeAssert()
	osn1.enqueueResponse(3)
	osn2.enqueueResponse(3)

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn1.srv.Address(), osn2.srv.Address())
	bp.FetchTimeout = time.Millisecond * 100

	notInUseOrdererNode := osn2
//配置记录器以告诉我们谁是订购者节点
//未连接到。这是通过截取适当的消息来完成的
	var waitForConnection sync.WaitGroup
	waitForConnection.Add(1)
	bp.Logger = bp.Logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if !strings.Contains(entry.Message, "Sending request for block 1") {
			return nil
		}
		defer waitForConnection.Done()
		s := entry.Message[len("Sending request for block 1 to 127.0.0.1:"):]
		port, err := strconv.ParseInt(s, 10, 32)
		assert.NoError(t, err)
//如果osn2是我们连接的当前订购方，
//我们没有联系的订购方是OSN1
		if osn2.port() == int(port) {
			notInUseOrdererNode = osn1
//将块1排队到当前排序器块拉具
//连接到
			osn2.enqueueResponse(1)
			osn2.addExpectPullAssert(1)
		} else {
//我们已连接到OSN1，因此将块1排队
			osn1.enqueueResponse(1)
			osn1.addExpectPullAssert(1)
		}
		return nil
	}))

	go func() {
		waitForConnection.Wait()
//将我们连接到的订购者的高度排队
		notInUseOrdererNode.enqueueResponse(3)
		notInUseOrdererNode.addExpectProbeAssert()
//将块2和3排队到未连接到的医嘱者节点。
		notInUseOrdererNode.addExpectPullAssert(2)
		notInUseOrdererNode.enqueueResponse(2)
		notInUseOrdererNode.enqueueResponse(3)
	}()

//断言接收1至3区
	assert.Equal(t, uint64(1), bp.PullBlock(uint64(1)).Header.Number)
	assert.Equal(t, uint64(2), bp.PullBlock(uint64(2)).Header.Number)
	assert.Equal(t, uint64(3), bp.PullBlock(uint64(3)).Header.Number)

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerNoOrdererAliveAtStartup(t *testing.T) {
//场景：单一排序节点，以及块拉具
//启动-订购者找不到。
	osn := newClusterNode(t)
	osn.stop()
	defer osn.stop()

	dialer := newCountingDialer()
	bp := newBlockPuller(dialer, osn.srv.Address())

//将记录器配置为等待连接尝试失败
	var connectionAttempt sync.WaitGroup
	connectionAttempt.Add(1)
	bp.Logger = bp.Logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if strings.Contains(entry.Message, "Failed connecting to") {
			connectionAttempt.Done()
		}
		return nil
	}))

	go func() {
		connectionAttempt.Wait()
		osn.resurrect()
//第一个SEEK请求请求请求最新的块
		osn.addExpectProbeAssert()
//第一个响应表示最后一个序列是2
		osn.enqueueResponse(2)
//下一个查找请求是针对块1的
		osn.addExpectPullAssert(1)
//订购方返回块1和块2
		osn.enqueueResponse(1)
		osn.enqueueResponse(2)
	}()

	assert.Equal(t, uint64(1), bp.PullBlock(1).Header.Number)
	assert.Equal(t, uint64(2), bp.PullBlock(2).Header.Number)

	bp.Close()
	dialer.assertAllConnectionsClosed(t)
}

func TestBlockPullerFailures(t *testing.T) {
//场景：单个订购节点出现故障，但稍后
//它恢复了。
	failureError := errors.New("oops, something went wrong")
	failStream := func(osn *deliverServer, _ *cluster.BlockPuller) {
		osn.Lock()
		osn.err = failureError
		osn.Unlock()
	}

	malformBlockSignatureAndRecreateOSNBuffer := func(osn *deliverServer, bp *cluster.BlockPuller) {
		bp.VerifyBlockSequence = func([]*common.Block) error {
			close(osn.blocks())
			osn.setBlocks(make(chan *orderer.DeliverResponse, 100))
			osn.enqueueResponse(1)
			osn.enqueueResponse(2)
			osn.enqueueResponse(3)
			return errors.New("bad signature")
		}
	}

	noFailFunc := func(_ *deliverServer, _ *cluster.BlockPuller) {}

	recover := func(osn *deliverServer, bp *cluster.BlockPuller) func(entry zapcore.Entry) error {
		return func(entry zapcore.Entry) error {
			if osn.isFaulty() && strings.Contains(entry.Message, failureError.Error()) {
				osn.Lock()
				osn.err = nil
				osn.Unlock()
			}
			if strings.Contains(entry.Message, "Failed verifying") {
				bp.VerifyBlockSequence = noopBlockVerifierf
			}
			return nil
		}
	}

	failAfterConnection := func(osn *deliverServer, logTrigger string, failFunc func()) func(entry zapcore.Entry) error {
		once := &sync.Once{}
		return func(entry zapcore.Entry) error {
			if !osn.isFaulty() && strings.Contains(entry.Message, logTrigger) {
				once.Do(func() {
					failFunc()
				})
			}
			return nil
		}
	}

	for _, testCase := range []struct {
		name       string
		logTrigger string
		beforeFunc func(*deliverServer, *cluster.BlockPuller)
		failFunc   func(*deliverServer, *cluster.BlockPuller)
	}{
		{
			name:       "failure at probe",
			logTrigger: "skip this for this test case",
			beforeFunc: func(osn *deliverServer, bp *cluster.BlockPuller) {
				failStream(osn, nil)
//第一个查找请求请求最新的块，但失败
				osn.addExpectProbeAssert()
//返回最后一个块序列
				osn.enqueueResponse(3)
//下一个查找请求是针对块1的
				osn.addExpectPullAssert(1)
			},
			failFunc: noFailFunc,
		},
		{
			name:       "failure at pull",
			logTrigger: "Sending request for block 1",
			beforeFunc: func(osn *deliverServer, bp *cluster.BlockPuller) {
//第一个SEEK请求请求请求最新的块并成功
				osn.addExpectProbeAssert()
//但是，当拉块器试图拉时，流失败了，所以它应该重新连接。
				osn.addExpectProbeAssert()
//我们需要发送最新的序列两次，因为流在第一次之后失败。
				osn.enqueueResponse(3)
				osn.enqueueResponse(3)
				osn.addExpectPullAssert(1)
			},
			failFunc: failStream,
		},
		{
			name:       "failure at verifying pulled block",
			logTrigger: "Sending request for block 1",
			beforeFunc: func(osn *deliverServer, bp *cluster.BlockPuller) {
//第一个SEEK请求请求请求最新的块并成功
				osn.addExpectProbeAssert()
				osn.enqueueResponse(3)
//然后，它将所有3个块都拉出来，但无法验证它们。
				osn.addExpectPullAssert(1)
				osn.enqueueResponse(1)
				osn.enqueueResponse(2)
				osn.enqueueResponse(3)
//所以它再次探测，然后再次请求块1。
				osn.addExpectProbeAssert()
				osn.addExpectPullAssert(1)
			},
			failFunc: malformBlockSignatureAndRecreateOSNBuffer,
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			osn := newClusterNode(t)
			defer osn.stop()

			dialer := newCountingDialer()
			bp := newBlockPuller(dialer, osn.srv.Address())

			testCase.beforeFunc(osn, bp)

//将记录器配置为在适当的时间触发故障。
			fail := func() {
				testCase.failFunc(osn, bp)
			}
			bp.Logger = bp.Logger.WithOptions(zap.Hooks(recover(osn, bp), failAfterConnection(osn, testCase.logTrigger, fail)))

//订购方发送块1到3
			osn.enqueueResponse(1)
			osn.enqueueResponse(2)
			osn.enqueueResponse(3)

			assert.Equal(t, uint64(1), bp.PullBlock(uint64(1)).Header.Number)
			assert.Equal(t, uint64(2), bp.PullBlock(uint64(2)).Header.Number)
			assert.Equal(t, uint64(3), bp.PullBlock(uint64(3)).Header.Number)

			bp.Close()
			dialer.assertAllConnectionsClosed(t)
		})
	}
}

func TestBlockPullerBadBlocks(t *testing.T) {
//场景：排序节点发送格式错误的块。

	removeHeader := func(resp *orderer.DeliverResponse) *orderer.DeliverResponse {
		resp.GetBlock().Header = nil
		return resp
	}

	removeData := func(resp *orderer.DeliverResponse) *orderer.DeliverResponse {
		resp.GetBlock().Data = nil
		return resp
	}

	removeMetadata := func(resp *orderer.DeliverResponse) *orderer.DeliverResponse {
		resp.GetBlock().Metadata = nil
		return resp
	}

	changeType := func(resp *orderer.DeliverResponse) *orderer.DeliverResponse {
		resp.Type = &orderer.DeliverResponse_Status{
			Status: common.Status_SUCCESS,
		}
		return resp
	}

	changeSequence := func(resp *orderer.DeliverResponse) *orderer.DeliverResponse {
		resp.GetBlock().Header.Number = 3
		return resp
	}

	testcases := []struct {
		name           string
		corruptBlock   func(block *orderer.DeliverResponse) *orderer.DeliverResponse
		expectedErrMsg string
	}{
		{
			name:           "nil header",
			corruptBlock:   removeHeader,
			expectedErrMsg: "block header is nil",
		},
		{
			name:           "nil data",
			corruptBlock:   removeData,
			expectedErrMsg: "block data is nil",
		},
		{
			name:           "nil metadata",
			corruptBlock:   removeMetadata,
			expectedErrMsg: "block metadata is empty",
		},
		{
			name:           "wrong type",
			corruptBlock:   changeType,
			expectedErrMsg: "response is of type",
		},
		{
			name:           "wrong number",
			corruptBlock:   changeSequence,
			expectedErrMsg: "got unexpected sequence",
		},
	}

	for _, testCase := range testcases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {

			osn := newClusterNode(t)
			defer osn.stop()

			osn.addExpectProbeAssert()
			osn.addExpectPullAssert(10)

			dialer := newCountingDialer()
			bp := newBlockPuller(dialer, osn.srv.Address())

			osn.enqueueResponse(10)
			osn.enqueueResponse(10)
//取第一
			block := <-osn.blockResponses
//将探测响应排队
//将损坏后的拉响应重新插入到队列的尾部
			osn.blockResponses <- testCase.corruptBlock(block)
//〔10 10＊〕
//期待拉块器重新尝试，这次给它想要的。
			osn.addExpectProbeAssert()
			osn.addExpectPullAssert(10)

//等待直到木块被拉起并检测到延展性。
			var detectedBadBlock sync.WaitGroup
			detectedBadBlock.Add(1)
			bp.Logger = bp.Logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
				if strings.Contains(entry.Message, fmt.Sprintf("Failed pulling blocks: %s", testCase.expectedErrMsg)) {
					detectedBadBlock.Done()
//关闭通道以关闭当前服务器端传递流
					close(osn.blocks())
//重新设置块缓冲区，使其能够再次写入。
					osn.setBlocks(make(chan *orderer.DeliverResponse, 100))
//在其后面放置一个正确的块，1个用于探测，1个用于获取
					osn.enqueueResponse(10)
					osn.enqueueResponse(10)
				}
				return nil
			}))

			bp.PullBlock(10)
			detectedBadBlock.Wait()

			bp.Close()
			dialer.assertAllConnectionsClosed(t)
		})
	}
}

func TestImpatientStreamFailure(t *testing.T) {
	osn := newClusterNode(t)
	dialer := newCountingDialer()
	defer dialer.assertAllConnectionsClosed(t)
//等待OSN启动
//通过尝试连接到它
	var conn *grpc.ClientConn
	var err error

	gt := gomega.NewGomegaWithT(t)
	gt.Eventually(func() (bool, error) {
		conn, err = dialer.Dial(osn.srv.Address())
		return true, err
	}).Should(gomega.BeTrue())
	newStream := cluster.NewImpatientStream(conn, time.Millisecond*100)
	defer conn.Close()
//关闭OSN
	osn.stop()
//确保端口不再打开
	gt.Eventually(func() (bool, error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		conn, _ := grpc.DialContext(ctx, osn.srv.Address(), grpc.WithBlock(), grpc.WithInsecure())
		if conn != nil {
			conn.Close()
			return false, nil
		}
		return true, nil
	}).Should(gomega.BeTrue())
	stream, err := newStream()
	if err != nil {
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
}
