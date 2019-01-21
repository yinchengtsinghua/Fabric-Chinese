
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


package etcdraft

import (
	"bytes"
	"context"
	"encoding/pem"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/wal"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//DefaultSnapshotChupentries是默认条目数
//在拍摄快照时保存在内存中。这是为了
//慢追随者赶上。
const DefaultSnapshotCatchUpEntries = uint64(500)

//去：生成mokery-dir。-名称配置器-大小写下划线-输出/模拟/

//配置器用于配置通信层
//当链条启动时。
type Configurator interface {
	Configure(channel string, newNodes []cluster.RemoteNode)
}

//go：生成伪造者-o mocks/mock-rpc.go。RPC

//RPC用于模拟测试中的传输层。
type RPC interface {
	Step(dest uint64, msg *orderer.StepRequest) (*orderer.StepResponse, error)
	SendSubmit(dest uint64, request *orderer.SubmitRequest) error
}

//去：生成伪造者-o mocks/mock-blockpuller.go。拦网者

//BlockPuller用于从其他OSN中拉出块。
type BlockPuller interface {
	PullBlock(seq uint64) *common.Block
	Close()
}

type block struct {
	b *common.Block

//i是与块相关的ETCD/筏入口索引。
//它作为块元数据持久化，因此我们知道在哪里
//重新启动后继续漂流。
	i uint64
}

//选项包含与链相关的所有配置。
type Options struct {
	RaftID uint64

	Clock clock.Clock

	WALDir       string
	SnapDir      string
	SnapInterval uint64

//这主要是为测试目的而配置的。用户不是
//应更改此。而是使用默认快照TChupentries。
	SnapshotCatchUpEntries uint64

	MemoryStorage MemoryStorage
	Logger        *flogging.FabricLogger

	TickInterval    time.Duration
	ElectionTick    int
	HeartbeatTick   int
	MaxSizePerMsg   uint64
	MaxInflightMsgs int

	RaftMetadata *etcdraft.RaftMetadata
}

//链实现共识。链接口。
type Chain struct {
	configurator Configurator

//对'sendsubmit'的访问应序列化，因为grpc不是线程安全的
	submitLock sync.Mutex
	rpc        RPC

	raftID    uint64
	channelID string

	submitC  chan *orderer.SubmitRequest
	commitC  chan block
observeC chan<- uint64         //通知外部观察者领导者的更改（作为测试的参数可选传入）
haltC    chan struct{}         //向Goroutines发出信号，表明链条正在停止。
doneC    chan struct{}         //链条停止时关闭
resignC  chan struct{}         //通知节点它不再是领导者
startC   chan struct{}         //节点启动时关闭
snapC    chan *raftpb.Snapshot //抓拍信号

configChangeAppliedC   chan struct{} //通知已应用raft配置更改
configChangeInProgress bool          //用于指示等待应用raft配置更改的节点的标志
	raftMetadataLock       sync.RWMutex

clock clock.Clock //测试可以注入假时钟

	support      consensus.ConsenterSupport
	BlockCreator *blockCreator

	leader       uint64
	appliedIndex uint64

//快照所需
	lastSnapBlockNum uint64
syncLock         sync.Mutex       //保护对SyncC的操作
syncC            chan struct{}    //指示正在进行同步
confState        raftpb.ConfState //ETCDraft要求在快照中持久化ConfState
puller           BlockPuller      //交付客户端以从其他OSN拉块

fresh bool //指示这是否是一个新的筏形节点

	node    raft.Node
	storage *RaftStorage
	opts    Options

	logger *flogging.FabricLogger
}

//newchain构造一个链对象。
func NewChain(
	support consensus.ConsenterSupport,
	opts Options,
	conf Configurator,
	rpc RPC,
	puller BlockPuller,
	observeC chan<- uint64) (*Chain, error) {

	lg := opts.Logger.With("channel", support.ChainID(), "node", opts.RaftID)

	fresh := !wal.Exist(opts.WALDir)

	appliedi := opts.RaftMetadata.RaftIndex
	storage, err := CreateStorage(lg, appliedi, opts.WALDir, opts.SnapDir, opts.MemoryStorage)
	if err != nil {
		return nil, errors.Errorf("failed to restore persisted raft data: %s", err)
	}

	if opts.SnapshotCatchUpEntries == 0 {
		storage.SnapshotCatchUpEntries = DefaultSnapshotCatchUpEntries
	} else {
		storage.SnapshotCatchUpEntries = opts.SnapshotCatchUpEntries
	}

//获取上次快照中的块号（如果存在）
	var snapBlkNum uint64
	if s := storage.Snapshot(); !raft.IsEmptySnap(s) {
		b := utils.UnmarshalBlockOrPanic(s.Data)
		snapBlkNum = b.Header.Number
	}

	lastBlock := support.Block(support.Height() - 1)

	return &Chain{
		configurator:         conf,
		rpc:                  rpc,
		channelID:            support.ChainID(),
		raftID:               opts.RaftID,
		submitC:              make(chan *orderer.SubmitRequest),
		commitC:              make(chan block),
		haltC:                make(chan struct{}),
		doneC:                make(chan struct{}),
		resignC:              make(chan struct{}),
		startC:               make(chan struct{}),
		syncC:                make(chan struct{}),
		snapC:                make(chan *raftpb.Snapshot),
		configChangeAppliedC: make(chan struct{}),
		observeC:             observeC,
		support:              support,
		fresh:                fresh,
		BlockCreator:         newBlockCreator(lastBlock, lg),
		appliedIndex:         appliedi,
		lastSnapBlockNum:     snapBlkNum,
		puller:               puller,
		clock:                opts.Clock,
		logger:               lg,
		storage:              storage,
		opts:                 opts,
	}, nil
}

//Start指示订购者开始供应链并保持其最新状态。
func (c *Chain) Start() {
	c.logger.Infof("Starting Raft node")

//不要在配置中使用应用的选项，请参阅https://github.com/etcd-io/etcd/issues/10217
//我们防止在“EntriesToApply”中重放写入的块。
	config := &raft.Config{
		ID:              c.raftID,
		ElectionTick:    c.opts.ElectionTick,
		HeartbeatTick:   c.opts.HeartbeatTick,
		MaxSizePerMsg:   c.opts.MaxSizePerMsg,
		MaxInflightMsgs: c.opts.MaxInflightMsgs,
		Logger:          c.logger,
		Storage:         c.opts.MemoryStorage,
//prevote防止重新连接的节点干扰网络。
//更多详情请参见ETCD/RAFT文件。
		PreVote:                   true,
DisableProposalForwarding: true, //这可以防止追随者意外地提出阻止
	}

	if err := c.configureComm(); err != nil {
		c.logger.Errorf("Failed to start chain, aborting: +%v", err)
		close(c.doneC)
		return
	}

	raftPeers := RaftPeers(c.opts.RaftMetadata.Consenters)

	if c.fresh {
		c.logger.Info("starting new raft node")
		c.node = raft.StartNode(config, raftPeers)
	} else {
		c.logger.Info("restarting raft node")
		c.node = raft.RestartNode(config)
	}

	close(c.startC)

	go c.serveRaft()
	go c.serveRequest()
}

//订单提交常规类型的交易进行订购。
func (c *Chain) Order(env *common.Envelope, configSeq uint64) error {
	return c.Submit(&orderer.SubmitRequest{LastValidationSeq: configSeq, Content: env, Channel: c.channelID}, 0)
}

//配置提交用于排序的配置类型事务。
func (c *Chain) Configure(env *common.Envelope, configSeq uint64) error {
	if err := c.checkConfigUpdateValidity(env); err != nil {
		return err
	}
	return c.Submit(&orderer.SubmitRequest{LastValidationSeq: configSeq, Content: env, Channel: c.channelID}, 0)
}

//验证配置更新是否为设计文档中所述的A或B类型。
func (c *Chain) checkConfigUpdateValidity(ctx *common.Envelope) error {
	var err error
	payload, err := utils.UnmarshalPayload(ctx.Payload)
	if err != nil {
		return err
	}
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return err
	}

	switch chdr.Type {
	case int32(common.HeaderType_ORDERER_TRANSACTION):
		return nil
	case int32(common.HeaderType_CONFIG):
		configUpdate, err := configtx.UnmarshalConfigUpdateFromPayload(payload)
		if err != nil {
			return err
		}

//检查写入集中是否只更新了conensustype
		if ordererConfigGroup, ok := configUpdate.WriteSet.Groups["Orderer"]; ok {
			if val, ok := ordererConfigGroup.Values["ConsensusType"]; ok {
				return c.checkConsentersSet(val)
			}
		}
		return nil

	default:
		return errors.Errorf("config transaction has unknown header type")
	}
}

//当链：
//-正在使用快照赶上其他节点
//
//在其他情况下，它会立即返回。
func (c *Chain) WaitReady() error {
	if err := c.isRunning(); err != nil {
		return err
	}

	c.syncLock.Lock()
	ch := c.syncC
	c.syncLock.Unlock()

	select {
	case <-ch:
	case <-c.doneC:
		return errors.Errorf("chain is stopped")
	}

	return nil
}

//ERRORED返回一个在链停止时关闭的通道。
func (c *Chain) Errored() <-chan struct{} {
	return c.doneC
}

//停止停止链条。
func (c *Chain) Halt() {
	select {
	case <-c.startC:
	default:
		c.logger.Warnf("Attempted to halt a chain that has not started")
		return
	}

	select {
	case c.haltC <- struct{}{}:
	case <-c.doneC:
		return
	}
	<-c.doneC
}

func (c *Chain) isRunning() error {
	select {
	case <-c.startC:
	default:
		return errors.Errorf("chain is not started")
	}

	select {
	case <-c.doneC:
		return errors.Errorf("chain is stopped")
	default:
	}

	return nil
}

//step将给定的stepRequest消息传递给raft.node实例
func (c *Chain) Step(req *orderer.StepRequest, sender uint64) error {
	if err := c.isRunning(); err != nil {
		return err
	}

	stepMsg := &raftpb.Message{}
	if err := proto.Unmarshal(req.Payload, stepMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StepRequest payload to Raft Message: %s", err)
	}

	if err := c.node.Step(context.TODO(), *stepMsg); err != nil {
		return fmt.Errorf("failed to process Raft Step message: %s", err)
	}

	return nil
}

//提交将传入请求转发到：
//-本地服务器请求Goroutine（如果这是领导者）
//-通过运输机制的实际领导者
//如果还没有选举出领导人，这个呼吁就失败了。
func (c *Chain) Submit(req *orderer.SubmitRequest, sender uint64) error {
	if err := c.isRunning(); err != nil {
		return err
	}

	lead := atomic.LoadUint64(&c.leader)

	if lead == raft.None {
		return errors.Errorf("no Raft leader")
	}

	if lead == c.raftID {
		select {
		case c.submitC <- req:
			return nil
		case <-c.doneC:
			return errors.Errorf("chain is stopped")
		}
	}

	c.logger.Debugf("Forwarding submit request to Raft leader %d", lead)
	c.submitLock.Lock()
	defer c.submitLock.Unlock()
	return c.rpc.SendSubmit(lead, req)
}

func (c *Chain) serveRequest() {
	ticking := false
	timer := c.clock.NewTimer(time.Second)
//我们需要一个停止计时而不是零，
//因为我们将选择等待计时器.c（）。
	if !timer.Stop() {
		<-timer.C()
	}

//如果计时器已启动，则这是一个禁止操作
	start := func() {
		if !ticking {
			ticking = true
			timer.Reset(c.support.SharedConfig().BatchTimeout())
		}
	}

	stop := func() {
		if !timer.Stop() && ticking {
//我们只需要在计时器过期（未显式停止）时排出通道中的内容。
			<-timer.C()
		}
		ticking = false
	}

	if s := c.storage.Snapshot(); !raft.IsEmptySnap(s) {
		if err := c.catchUp(&s); err != nil {
			c.logger.Errorf("Failed to recover from snapshot taken at Term %d and Index %d: %s",
				s.Metadata.Term, s.Metadata.Index, err)
		}
	} else {
		close(c.syncC)
	}

	for {
		select {
		case msg := <-c.submitC:
			batches, pending, err := c.ordered(msg)
			if err != nil {
				c.logger.Errorf("Failed to order message: %s", err)
			}
			if pending {
start() //如果计时器已启动，则无操作
			} else {
				stop()
			}

			if err := c.commitBatches(batches...); err != nil {
				c.logger.Errorf("Failed to commit block: %s", err)
			}

		case b := <-c.commitC:
			c.writeBlock(b)

		case <-c.resignC:
			_ = c.support.BlockCutter().Cut()
			c.BlockCreator.resetCreatedBlocks()
			stop()

		case <-timer.C():
			ticking = false

			batch := c.support.BlockCutter().Cut()
			if len(batch) == 0 {
				c.logger.Warningf("Batch timer expired with no pending requests, this might indicate a bug")
				continue
			}

			c.logger.Debugf("Batch timer expired, creating block")
			if err := c.commitBatches(batch); err != nil {
				c.logger.Errorf("Failed to commit block: %s", err)
			}

		case sn := <-c.snapC:
			if err := c.catchUp(sn); err != nil {
				c.logger.Errorf("Failed to recover from snapshot taken at Term %d and Index %d: %s",
					sn.Metadata.Term, sn.Metadata.Index, err)
			}

		case <-c.doneC:
			c.logger.Infof("Stop serving requests")
			return
		}
	}
}

func (c *Chain) writeBlock(b block) {
	c.BlockCreator.commitBlock(b.b)
	if utils.IsConfigBlock(b.b) {
		if err := c.writeConfigBlock(b); err != nil {
			c.logger.Panicf("failed to write configuration block, %+v", err)
		}
		return
	}

	c.raftMetadataLock.Lock()
	c.opts.RaftMetadata.RaftIndex = b.i
	m := utils.MarshalOrPanic(c.opts.RaftMetadata)
	c.raftMetadataLock.Unlock()

	c.support.WriteBlock(b.b, m)
}

//在“msg”内容中订购信封。SubmitRequest。
//退换商品
//--批次[]*common.envelope；切割的批次，
//--挂起的bool；如果有等待订购的信封，
//--错误；遇到的错误（如果有）。
//它负责配置消息以及消息的重新验证（如果配置序列高级的话）。
func (c *Chain) ordered(msg *orderer.SubmitRequest) (batches [][]*common.Envelope, pending bool, err error) {
	seq := c.support.Sequence()

	if c.isConfig(msg.Content) {
//配置文件
		if msg.LastValidationSeq < seq {
			msg.Content, _, err = c.support.ProcessConfigMsg(msg.Content)
			if err != nil {
				return nil, true, errors.Errorf("bad config message: %s", err)
			}
		}
		batch := c.support.BlockCutter().Cut()
		batches = [][]*common.Envelope{}
		if len(batch) != 0 {
			batches = append(batches, batch)
		}
		batches = append(batches, []*common.Envelope{msg.Content})
		return batches, false, nil
	}
//这是正常的信息
	if msg.LastValidationSeq < seq {
		if _, err := c.support.ProcessNormalMsg(msg.Content); err != nil {
			return nil, true, errors.Errorf("bad normal message: %s", err)
		}
	}
	batches, pending = c.support.BlockCutter().Ordered(msg.Content)
	return batches, pending, nil

}

func (c *Chain) commitBatches(batches ...[]*common.Envelope) error {
	for _, batch := range batches {
		b := c.BlockCreator.createNextBlock(batch)
		data := utils.MarshalOrPanic(b)
		if err := c.node.Propose(context.TODO(), data); err != nil {
			return errors.Errorf("failed to propose data to Raft node: %s", err)
		}

//如果是配置块，则等待该块的提交
		if utils.IsConfigBlock(b) {
//我们需要循环来解释在配置块到达之前可能正在飞行的正常块。
		commitConfigLoop:
			for {
				select {
				case block := <-c.commitC:
					c.writeBlock(block)
//因为这是一直在寻找的配置块，所以我们中断了循环
					if bytes.Equal(b.Header.Bytes(), block.b.Header.Bytes()) {
						break commitConfigLoop
					}

				case <-c.resignC:
					return errors.Errorf("aborted block committing: lost leadership")

				case <-c.doneC:
					return nil
				}
			}
		}
	}

	return nil
}

func (c *Chain) catchUp(snap *raftpb.Snapshot) error {
	b, err := utils.UnmarshalBlock(snap.Data)
	if err != nil {
		return errors.Errorf("failed to unmarshal snapshot data to block: %s", err)
	}

	c.logger.Infof("Catching up with snapshot taken at block %d", b.Header.Number)

	next := c.support.Height()
	if next > b.Header.Number {
		c.logger.Warnf("Snapshot is at block %d, local block number is %d, no sync needed", b.Header.Number, next-1)
		return nil
	}

	c.syncLock.Lock()
	c.syncC = make(chan struct{})
	c.syncLock.Unlock()
	defer func() {
		close(c.syncC)
		c.puller.Close()
	}()

	for next <= b.Header.Number {
		block := c.puller.PullBlock(next)
		if block == nil {
			return errors.Errorf("failed to fetch block %d from cluster", next)
		}

		c.BlockCreator.commitBlock(block)
		if utils.IsConfigBlock(block) {
			c.support.WriteConfigBlock(block, nil)
		} else {
			c.support.WriteBlock(block, nil)
		}

		next++
	}

	c.logger.Infof("Finished syncing with cluster up to block %d (incl.)", b.Header.Number)
	return nil
}

func (c *Chain) serveRaft() {
	ticker := c.clock.NewTicker(c.opts.TickInterval)

	for {
		select {
		case <-ticker.C():
			c.node.Tick()

		case rd := <-c.node.Ready():
			if err := c.storage.Store(rd.Entries, rd.HardState, rd.Snapshot); err != nil {
				c.logger.Panicf("Failed to persist etcd/raft data: %s", err)
			}

			if !raft.IsEmptySnap(rd.Snapshot) {
				c.snapC <- &rd.Snapshot

				b := utils.UnmarshalBlockOrPanic(rd.Snapshot.Data)
				c.lastSnapBlockNum = b.Header.Number
				c.confState = rd.Snapshot.Metadata.ConfState
				c.appliedIndex = rd.Snapshot.Metadata.Index
			}

			c.apply(rd.CommittedEntries)
			c.node.Advance()

//Todo（Jay_Guo）Leader可以在复制的同时写入磁盘
//给追随者和他们写磁盘。检查论文中的10.2.1
			c.send(rd.Messages)

			if rd.SoftState != nil {
				newLead := atomic.LoadUint64(&rd.SoftState.Lead)
				lead := atomic.LoadUint64(&c.leader)
				if newLead != lead {
					c.logger.Infof("Raft leader changed: %d -> %d", lead, newLead)
					atomic.StoreUint64(&c.leader, newLead)

					if lead == c.raftID {
						c.resignC <- struct{}{}
					}

//成为领导者，正在进行配置更改
					if newLead == c.raftID && c.configChangeInProgress {
//需要读取副本集的最新配置更新
//完成重新配置
						c.handleReconfigurationFailover()
					}

//通知外部观察员
					select {
					case c.observeC <- newLead:
					default:
					}
				}
			}

		case <-c.haltC:
			ticker.Stop()
			c.node.Stop()
			c.storage.Close()
			c.logger.Infof("Raft node stopped")
close(c.doneC) //关闭所有项目后关闭
			return
		}
	}
}

func (c *Chain) apply(ents []raftpb.Entry) {
	if len(ents) == 0 {
		return
	}

	if ents[0].Index > c.appliedIndex+1 {
		c.logger.Panicf("first index of committed entry[%d] should <= appliedIndex[%d]+1", ents[0].Index, c.appliedIndex)
	}

	var appliedb uint64
	var position int
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
//我们需要严格避免重新应用正常条目，
//否则，我们将两次写入同一块。
			if len(ents[i].Data) == 0 || ents[i].Index <= c.appliedIndex {
				break
			}

			b := utils.UnmarshalBlockOrPanic(ents[i].Data)
//当给定的块包含更新时需要检查
//这将导致会员资格的改变，最终
//到集群重新配置
			c.raftMetadataLock.RLock()
			m := c.opts.RaftMetadata
			c.raftMetadataLock.RUnlock()

			isConfigMembershipUpdate, err := IsMembershipUpdate(b, m)
			if err != nil {
				c.logger.Warnf("Error while attempting to determine membership update, due to %s", err)
			}
//如果发生错误，isConfigMembershipUpdate将为false，因此将跳过中的设置配置更改。
//进步
			if isConfigMembershipUpdate {
//仅当配置块
//并更新了raft副本集
				c.configChangeInProgress = true
			}

			c.commitC <- block{b, ents[i].Index}

			appliedb = b.Header.Number
			position = i

		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			if err := cc.Unmarshal(ents[i].Data); err != nil {
				c.logger.Warnf("Failed to unmarshal ConfChange data: %s", err)
				continue
			}

			c.confState = *c.node.ApplyConfChange(cc)

			if c.configChangeInProgress {
//已应用配置更改的信号
				c.configChangeAppliedC <- struct{}{}
//设置回旗
				c.configChangeInProgress = false
			}
		}

		if ents[i].Index > c.appliedIndex {
			c.appliedIndex = ents[i].Index
		}
	}

	if c.opts.SnapInterval == 0 || appliedb == 0 {
//未启用快照（SnapInterval==0）或
//此轮中未写入任何块（appliedb==0）
		return
	}

	if appliedb-c.lastSnapBlockNum >= c.opts.SnapInterval {
		c.logger.Infof("Taking snapshot at block %d, last snapshotted block number is %d", appliedb, c.lastSnapBlockNum)
		if err := c.storage.TakeSnapshot(c.appliedIndex, &c.confState, ents[position].Data); err != nil {
			c.logger.Fatalf("Failed to create snapshot at index %d", c.appliedIndex)
		}

		c.lastSnapBlockNum = appliedb
	}
}

func (c *Chain) send(msgs []raftpb.Message) {
	for _, msg := range msgs {
		if msg.To == 0 {
			continue
		}

		status := raft.SnapshotFinish

		msgBytes := utils.MarshalOrPanic(&msg)
		_, err := c.rpc.Step(msg.To, &orderer.StepRequest{Channel: c.support.ChainID(), Payload: msgBytes})
		if err != nil {
//TODO如果邮件传递失败，我们应该调用ReportUnreachable
			c.logger.Errorf("Failed to send StepRequest to %d, because: %s", msg.To, err)

			status = raft.SnapshotFailure
		}

		if msg.Type == raftpb.MsgSnap {
			c.node.ReportSnapshot(msg.To, status)
		}
	}
}

func (c *Chain) isConfig(env *common.Envelope) bool {
	h, err := utils.ChannelHeader(env)
	if err != nil {
		c.logger.Panicf("failed to extract channel header from envelope")
	}

	return h.Type == int32(common.HeaderType_CONFIG) || h.Type == int32(common.HeaderType_ORDERER_TRANSACTION)
}

func (c *Chain) configureComm() error {
	nodes, err := c.remotePeers()
	if err != nil {
		return err
	}

	c.configurator.Configure(c.channelID, nodes)
	return nil
}

func (c *Chain) remotePeers() ([]cluster.RemoteNode, error) {
	var nodes []cluster.RemoteNode
	for raftID, consenter := range c.opts.RaftMetadata.Consenters {
//不需要了解你自己
		if raftID == c.raftID {
			continue
		}
		serverCertAsDER, err := c.pemToDER(consenter.ServerTlsCert, raftID, "server")
		if err != nil {
			return nil, errors.WithStack(err)
		}
		clientCertAsDER, err := c.pemToDER(consenter.ClientTlsCert, raftID, "client")
		if err != nil {
			return nil, errors.WithStack(err)
		}
		nodes = append(nodes, cluster.RemoteNode{
			ID:            raftID,
			Endpoint:      fmt.Sprintf("%s:%d", consenter.Host, consenter.Port),
			ServerTLSCert: serverCertAsDER,
			ClientTLSCert: clientCertAsDER,
		})
	}
	return nodes, nil
}

func (c *Chain) pemToDER(pemBytes []byte, id uint64, certType string) ([]byte, error) {
	bl, _ := pem.Decode(pemBytes)
	if bl == nil {
		c.logger.Errorf("Rejecting PEM block of %s TLS cert for node %d, offending PEM is: %s", certType, id, string(pemBytes))
		return nil, errors.Errorf("invalid PEM block")
	}
	return bl.Bytes, nil
}

//CheckConsenterset验证配置值中提供的同意器集的正确性
func (c *Chain) checkConsentersSet(configValue *common.ConfigValue) error {
//从配置中读取元数据更新
	updatedMetadata, err := MetadataFromConfigValue(configValue)
	if err != nil {
		return err
	}

	c.raftMetadataLock.RLock()
	changes := ComputeMembershipChanges(c.opts.RaftMetadata.Consenters, updatedMetadata.Consenters)
	c.raftMetadataLock.RUnlock()

	if changes.TotalChanges > 1 {
		return errors.New("update of more than one consenters at a time is not supported")
	}

	return nil
}

//更新成员身份用新成员身份更改更新raft元数据，将raft更改应用于副本集
//通过提出配置更改和阻塞直到应用
func (c *Chain) updateMembership(metadata *etcdraft.RaftMetadata, change *raftpb.ConfChange) error {
	lead := atomic.LoadUint64(&c.leader)
//领导提出配置变更
	if lead == c.raftID {
//ProposeConfChange仅在节点停止时返回错误。
		if err := c.node.ProposeConfChange(context.TODO(), *change); err != nil {
			c.logger.Warnf("Failed to propose configuration update to Raft node: %s", err)
			return nil
		}
	}

	var err error

	for {
		select {
case <-c.configChangeAppliedC: //筏组的筏形结构变化已经应用
//块提交后更新元数据
			c.raftMetadataLock.Lock()
			c.opts.RaftMetadata = metadata
			c.raftMetadataLock.Unlock()

//现在我们需要用新的更新重新配置通信层
			return c.configureComm()
		case <-c.resignC:
			c.logger.Debug("Raft cluster leader has changed, new leader should re-propose Raft config change based on last config block")
		case <-c.doneC:
			c.logger.Debug("Shutting down node, aborting config change update")
			return err
		}
	}
}

//WriteConfigBlock在中将配置块写入分类帐
//添加提取关于raft复制集的更新，如果有
//更改是否也会更新群集成员身份？
func (c *Chain) writeConfigBlock(b block) error {
	metadata, raftMetadata := c.newRaftMetadata(b.b)

	var changes *MembershipChanges
	if metadata != nil {
		changes = ComputeMembershipChanges(raftMetadata.Consenters, metadata.Consenters)
	}

	confChange := changes.UpdateRaftMetadataAndConfChange(raftMetadata)
	raftMetadata.RaftIndex = b.i

	raftMetadataBytes := utils.MarshalOrPanic(raftMetadata)
//用元数据写入块
	c.support.WriteConfigBlock(b.b, raftMetadataBytes)
	if confChange != nil {
		if err := c.updateMembership(raftMetadata, confChange); err != nil {
			return errors.Wrap(err, "failed to update Raft with consenters membership changes")
		}
	}
	return nil
}

//handlerConfigurationFailover读取最后一个配置块并建议
//新筏形结构
func (c *Chain) handleReconfigurationFailover() {
	b := c.support.Block(c.support.Height() - 1)
	if b == nil {
		c.logger.Panic("nil block, failed to read last written block")
	}
	if !utils.IsConfigBlock(b) {
//在serverreq go例程上下文中离开updateMembership的节点（领队或跟随者），
//*iff*配置条目已出现并成功应用。
//虽然它在updateMembership中被阻止，但它不能提交任何其他块，
//因此，我们保证最后一个块是配置块
		c.logger.Panic("while handling reconfiguration failover last expected block should be configuration")
	}

	metadata, raftMetadata := c.newRaftMetadata(b)

	var changes *MembershipChanges
	if metadata != nil {
		changes = ComputeMembershipChanges(raftMetadata.Consenters, metadata.Consenters)
	}

	confChange := changes.UpdateRaftMetadataAndConfChange(raftMetadata)
	if err := c.node.ProposeConfChange(context.TODO(), *confChange); err != nil {
		c.logger.Warnf("failed to propose configuration update to Raft node: %s", err)
	}
}

//newraftmetadata从配置块中提取raft元数据
func (c *Chain) newRaftMetadata(block *common.Block) (*etcdraft.Metadata, *etcdraft.RaftMetadata) {
	metadata, err := ConsensusMetadataFromConfigBlock(block)
	if err != nil {
		c.logger.Panicf("error reading consensus metadata: %s", err)
	}
	c.raftMetadataLock.RLock()
	raftMetadata := proto.Clone(c.opts.RaftMetadata).(*etcdraft.RaftMetadata)
//proto.clone不复制空映射，因此需要在
//克隆
	if raftMetadata.Consenters == nil {
		raftMetadata.Consenters = map[uint64]*etcdraft.Consenter{}
	}
	c.raftMetadataLock.RUnlock()
	return metadata, raftMetadata
}
