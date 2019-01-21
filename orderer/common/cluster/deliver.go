
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


package cluster

import (
	"context"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

//blockpuller从远程排序节点拉块。
//它的操作不是线程安全的。
type BlockPuller struct {
//配置
	MaxTotalBufferBytes int
	Signer              crypto.LocalSigner
	TLSCert             []byte
	Channel             string
	FetchTimeout        time.Duration
	RetryTimeout        time.Duration
	Logger              *flogging.FabricLogger
	Dialer              Dialer
	VerifyBlockSequence BlockSequenceVerifier
	Endpoints           []string
//内部状态
	stream       *ImpatientStream
	blockBuff    []*common.Block
	latestSeq    uint64
	endpoint     string
	conn         *grpc.ClientConn
	cancelStream func()
}

//克隆返回已初始化的此BlockPuller的副本
//对于给定的通道
func (p *BlockPuller) Clone() *BlockPuller {
//按值克隆
	copy := *p
//重置内部状态
	copy.stream = nil
	copy.blockBuff = nil
	copy.latestSeq = 0
	copy.endpoint = ""
	copy.conn = nil
	copy.cancelStream = nil
	return &copy
}

//关闭使blockpuller关闭连接和流
//使用远程终结点。
func (p *BlockPuller) Close() {
	if p.cancelStream != nil {
		p.cancelStream()
	}
	p.cancelStream = nil

	if p.conn != nil {
		p.conn.Close()
	}
	p.conn = nil
	p.endpoint = ""
	p.latestSeq = 0
}

//PullBlock块，直到获取具有给定序列的块为止
//来自某个远程排序节点。
func (p *BlockPuller) PullBlock(seq uint64) *common.Block {
	for {
		block := p.tryFetchBlock(seq)
		if block != nil {
			return block
		}
	}
}

//HeightsByEndpoints按排序器的端点返回块高度
func (p *BlockPuller) HeightsByEndpoints() map[string]uint64 {
	res := make(map[string]uint64)
	for endpoint, endpointInfo := range p.probeEndpoints(1).byEndpoints() {
		endpointInfo.conn.Close()
		res[endpoint] = endpointInfo.lastBlockSeq + 1
	}
	p.Logger.Info("Returning the heights of OSNs mapped by endpoints", res)
	return res
}

func (p *BlockPuller) tryFetchBlock(seq uint64) *common.Block {
	var reConnected bool
	for p.isDisconnected() {
		reConnected = true
		p.connectToSomeEndpoint(seq)
		if p.isDisconnected() {
			time.Sleep(p.RetryTimeout)
		}
	}

	block := p.popBlock(seq)
	if block != nil {
		return block
	}
//否则，缓冲区为空。所以我们需要拉滑轮
//重新填满它。
	if err := p.pullBlocks(seq, reConnected); err != nil {
		p.Logger.Errorf("Failed pulling blocks: %v", err)
//出了点问题，断开连接。返回零
		p.Close()
//如果缓冲区中有块，则返回它。
		if len(p.blockBuff) > 0 {
			return p.blockBuff[0]
		}
		return nil
	}

	if err := p.VerifyBlockSequence(p.blockBuff); err != nil {
		p.Close()
		p.Logger.Errorf("Failed verifying received blocks: %v", err)
		return nil
	}

//此时，缓冲区已满，因此请移动它并返回第一个块。
	return p.popBlock(seq)
}

func (p *BlockPuller) setCancelStreamFunc(f func()) {
	p.cancelStream = f
}

func (p *BlockPuller) pullBlocks(seq uint64, reConnected bool) error {
	env, err := p.seekNextEnvelope(seq)
	if err != nil {
		p.Logger.Errorf("Failed creating seek envelope: %v", err)
		return err
	}

	stream, err := p.obtainStream(reConnected, env, seq)
	if err != nil {
		return err
	}

	var totalSize int
	p.blockBuff = nil
	nextExpectedSequence := seq
	for totalSize < p.MaxTotalBufferBytes && nextExpectedSequence <= p.latestSeq {
		resp, err := stream.Recv()
		if err != nil {
			p.Logger.Errorf("Failed receiving next block from %s: %v", p.endpoint, err)
			return err
		}

		block, err := extractBlockFromResponse(resp)
		if err != nil {
			p.Logger.Errorf("Received a bad block from %s: %v", p.endpoint, err)
			return err
		}
		seq := block.Header.Number
		if seq != nextExpectedSequence {
			p.Logger.Errorf("Expected to receive sequence %d but got %d instead", nextExpectedSequence, seq)
			return errors.Errorf("got unexpected sequence from %s - (%d) instead of (%d)", p.endpoint, seq, nextExpectedSequence)
		}
		size := blockSize(block)
		totalSize += size
		p.blockBuff = append(p.blockBuff, block)
		nextExpectedSequence++
		p.Logger.Infof("Got block %d of size %dKB from %s", seq, size/1024, p.endpoint)
	}
	return nil
}

func (p *BlockPuller) obtainStream(reConnected bool, env *common.Envelope, seq uint64) (*ImpatientStream, error) {
	var stream *ImpatientStream
	var err error
	if reConnected {
		p.Logger.Infof("Sending request for block %d to %s", seq, p.endpoint)
		stream, err = p.requestBlocks(p.endpoint, NewImpatientStream(p.conn, p.FetchTimeout), env)
		if err != nil {
			return nil, err
		}
//流已成功建立。
//在这个函数的下一个迭代中，重用它。
		p.stream = stream
	} else {
//重用上一个流
		stream = p.stream
	}

	p.setCancelStreamFunc(stream.cancelFunc)
	return stream, nil
}

//popblock从内存缓冲区中弹出一个块并返回它，
//如果缓冲区为空或块不匹配，则返回nil
//给定的所需序列。
func (p *BlockPuller) popBlock(seq uint64) *common.Block {
	if len(p.blockBuff) == 0 {
		return nil
	}
	block, rest := p.blockBuff[0], p.blockBuff[1:]
	p.blockBuff = rest
//如果请求的块序列错误，则丢弃缓冲区
//重新开始获取块。
	if seq != block.Header.Number {
		p.blockBuff = nil
		return nil
	}
	return block
}

func (p *BlockPuller) isDisconnected() bool {
	return p.conn == nil
}

//ConnectToSomeEndpoint使BlockPuller连接到具有
//给定的最小块序列。
func (p *BlockPuller) connectToSomeEndpoint(minRequestedSequence uint64) {
//并行探测所有端点，使用给定的最小块序列搜索端点
//然后根据它们的端点将它们排序到地图上。
	endpointsInfo := p.probeEndpoints(minRequestedSequence).byEndpoints()
	if len(endpointsInfo) == 0 {
		p.Logger.Warningf("Could not connect to any endpoint of %v", p.Endpoints)
		return
	}

//从可用终结点中选择随机终结点
	chosenEndpoint := randomEndpoint(endpointsInfo)
//断开除此终结点之外的所有连接
	for endpoint, endpointInfo := range endpointsInfo {
		if endpoint == chosenEndpoint {
			continue
		}
		endpointInfo.conn.Close()
	}

	p.conn = endpointsInfo[chosenEndpoint].conn
	p.endpoint = chosenEndpoint
	p.latestSeq = endpointsInfo[chosenEndpoint].lastBlockSeq

	p.Logger.Infof("Connected to %s with last block seq of %d", p.endpoint, p.latestSeq)
}

//probeendpoints到达所有已知端点并返回最新的块序列
//以及与端点的GRPC连接。
func (p *BlockPuller) probeEndpoints(minRequestedSequence uint64) *endpointInfoBucket {
	endpointsInfo := make(chan *endpointInfo, len(p.Endpoints))

	var wg sync.WaitGroup
	wg.Add(len(p.Endpoints))

	for _, endpoint := range p.Endpoints {
		go func(endpoint string) {
			defer wg.Done()
			endpointInfo, err := p.probeEndpoint(endpoint, minRequestedSequence)
			if err != nil {
				return
			}
			endpointsInfo <- endpointInfo
		}(endpoint)
	}
	wg.Wait()

	close(endpointsInfo)
	return &endpointInfoBucket{
		bucket: endpointsInfo,
		logger: p.Logger,
	}
}

//probeendpoint返回具有给定的
//需要最小的序列，或者出错时出错。
func (p *BlockPuller) probeEndpoint(endpoint string, minRequestedSequence uint64) (*endpointInfo, error) {
	conn, err := p.Dialer.Dial(endpoint)
	if err != nil {
		p.Logger.Warningf("Failed connecting to %s: %v", endpoint, err)
		return nil, err
	}

	lastBlockSeq, err := p.fetchLastBlockSeq(minRequestedSequence, endpoint, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &endpointInfo{conn: conn, lastBlockSeq: lastBlockSeq, endpoint: endpoint}, nil
}

//random endpoint返回给定endpointinfo的随机端点
func randomEndpoint(endpointsToHeight map[string]*endpointInfo) string {
	var candidates []string
	for endpoint := range endpointsToHeight {
		candidates = append(candidates, endpoint)
	}

	rand.Seed(time.Now().UnixNano())
	return candidates[rand.Intn(len(candidates))]
}

//fetchlastblockseq返回具有给定GRPC连接的端点的最后一个块序列。
func (p *BlockPuller) fetchLastBlockSeq(minRequestedSequence uint64, endpoint string, conn *grpc.ClientConn) (uint64, error) {
	env, err := p.seekLastEnvelope()
	if err != nil {
		p.Logger.Errorf("Failed creating seek envelope for %s: %v", endpoint, err)
		return 0, err
	}

	stream, err := p.requestBlocks(endpoint, NewImpatientStream(conn, p.FetchTimeout), env)
	if err != nil {
		return 0, err
	}
	defer stream.abort()

	resp, err := stream.Recv()
	if err != nil {
		p.Logger.Errorf("Failed receiving the latest block from %s: %v", endpoint, err)
		return 0, err
	}

	block, err := extractBlockFromResponse(resp)
	if err != nil {
		p.Logger.Errorf("Received a bad block from %s: %v", endpoint, err)
		return 0, err
	}
	stream.CloseSend()

	seq := block.Header.Number
	if seq < minRequestedSequence {
		err := errors.Errorf("minimum requested sequence is %d but %s is at sequence %d", minRequestedSequence, endpoint, seq)
		p.Logger.Infof("Skipping pulling from %s: %v", endpoint, err)
		return 0, err
	}

	p.Logger.Infof("%s is at block sequence of %d", endpoint, seq)
	return block.Header.Number, nil
}

//请求块从给定的端点开始请求块，使用给定的不耐烦的reamcreator通过发送
//给定的信封。
//它返回一个用于拉块的流，或者在出错时返回错误。
func (p *BlockPuller) requestBlocks(endpoint string, newStream ImpatientStreamCreator, env *common.Envelope) (*ImpatientStream, error) {
	stream, err := newStream()
	if err != nil {
		p.Logger.Warningf("Failed establishing deliver stream with %s", endpoint)
		return nil, err
	}

	if err := stream.Send(env); err != nil {
		p.Logger.Errorf("Failed sending seek envelope to %s: %v", endpoint, err)
		stream.abort()
		return nil, err
	}
	return stream, nil
}

func extractBlockFromResponse(resp *orderer.DeliverResponse) (*common.Block, error) {
	switch t := resp.Type.(type) {
	case *orderer.DeliverResponse_Block:
		block := t.Block
		if block == nil {
			return nil, errors.New("block is nil")
		}
		if block.Data == nil {
			return nil, errors.New("block data is nil")
		}
		if block.Header == nil {
			return nil, errors.New("block header is nil")
		}
		if block.Metadata == nil || len(block.Metadata.Metadata) == 0 {
			return nil, errors.New("block metadata is empty")
		}
		return block, nil
	default:
		return nil, errors.Errorf("response is of type %v, but expected a block", reflect.TypeOf(resp.Type))
	}
}

func (p *BlockPuller) seekLastEnvelope() (*common.Envelope, error) {
	return utils.CreateSignedEnvelopeWithTLSBinding(
		common.HeaderType_DELIVER_SEEK_INFO,
		p.Channel,
		p.Signer,
		last(),
		int32(0),
		uint64(0),
		util.ComputeSHA256(p.TLSCert),
	)
}

func (p *BlockPuller) seekNextEnvelope(startSeq uint64) (*common.Envelope, error) {
	return utils.CreateSignedEnvelopeWithTLSBinding(
		common.HeaderType_DELIVER_SEEK_INFO,
		p.Channel,
		p.Signer,
		nextSeekInfo(startSeq),
		int32(0),
		uint64(0),
		util.ComputeSHA256(p.TLSCert),
	)
}

func last() *orderer.SeekInfo {
	return &orderer.SeekInfo{
		Start:    &orderer.SeekPosition{Type: &orderer.SeekPosition_Newest{Newest: &orderer.SeekNewest{}}},
		Stop:     &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: math.MaxUint64}}},
		Behavior: orderer.SeekInfo_BLOCK_UNTIL_READY,
	}
}

func nextSeekInfo(startSeq uint64) *orderer.SeekInfo {
	return &orderer.SeekInfo{
		Start:    &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: startSeq}}},
		Stop:     &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: math.MaxUint64}}},
		Behavior: orderer.SeekInfo_BLOCK_UNTIL_READY,
	}
}

func blockSize(block *common.Block) int {
	return len(utils.MarshalOrPanic(block))
}

type endpointInfo struct {
	endpoint     string
	conn         *grpc.ClientConn
	lastBlockSeq uint64
}

type endpointInfoBucket struct {
	bucket <-chan *endpointInfo
	logger *flogging.FabricLogger
}

func (eib endpointInfoBucket) byEndpoints() map[string]*endpointInfo {
	infoByEndpoints := make(map[string]*endpointInfo)
	for endpointInfo := range eib.bucket {
		if _, exists := infoByEndpoints[endpointInfo.endpoint]; exists {
			eib.logger.Warningf("Duplicate endpoint found(%s), skipping it", endpointInfo.endpoint)
			endpointInfo.conn.Close()
			continue
		}
		infoByEndpoints[endpointInfo.endpoint] = endpointInfo
	}
	return infoByEndpoints
}

//急流创造者创造了急流
type ImpatientStreamCreator func() (*ImpatientStream, error)

//如果流等待消息的时间过长，则不耐烦流将中止该流。
type ImpatientStream struct {
	waitTimeout time.Duration
	orderer.AtomicBroadcast_DeliverClient
	cancelFunc func()
}

func (stream *ImpatientStream) abort() {
	stream.cancelFunc()
}

//接收阻塞，直到从流或
//超时过期。
func (stream *ImpatientStream) Recv() (*orderer.DeliverResponse, error) {
//初始化超时以在流过期时取消该流
	timeout := time.NewTimer(stream.waitTimeout)
	defer timeout.Stop()

	responseChan := make(chan errorAndResponse, 1)

//receive waitgroup确保下面的goroutine在
//此函数退出。
	var receive sync.WaitGroup
	receive.Add(1)
	defer receive.Wait()

	go func() {
		defer receive.Done()
		resp, err := stream.AtomicBroadcast_DeliverClient.Recv()
		responseChan <- errorAndResponse{err: err, resp: resp}
	}()

	select {
	case <-timeout.C:
		stream.cancelFunc()
		return nil, errors.Errorf("didn't receive a response within %v", stream.waitTimeout)
	case respAndErr := <-responseChan:
		return respAndErr.resp, respAndErr.err
	}
}

//新的不耐烦流返回创建不耐烦流的不耐烦流生成器。
func NewImpatientStream(conn *grpc.ClientConn, waitTimeout time.Duration) ImpatientStreamCreator {
	return func() (*ImpatientStream, error) {
		abc := orderer.NewAtomicBroadcastClient(conn)
		ctx, cancel := context.WithCancel(context.Background())

		stream, err := abc.Deliver(ctx)
		if err != nil {
			cancel()
			return nil, err
		}

		once := &sync.Once{}
		return &ImpatientStream{
			waitTimeout: waitTimeout,
//调用close（）时，流可能被取消，但
//当超时过期时，请确保只调用一次。
			cancelFunc: func() {
				once.Do(cancel)
			},
			AtomicBroadcast_DeliverClient: stream,
		}, nil
	}
}

type errorAndResponse struct {
	err  error
	resp *orderer.DeliverResponse
}
