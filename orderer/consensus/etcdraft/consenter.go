
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
	"path"
	"reflect"
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/coreos/etcd/raft"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/viperutil"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/common/multichannel"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/pkg/errors"
)

//去：生成mokery-dir。-name chainegetter-case underline-output mock

//chainegetter获取给定通道的chainSupport实例
type ChainGetter interface {
//getchain获取给定通道的链支持。
//当给定通道的链支持时返回nil，false
//找不到。
	GetChain(chainID string) *multichannel.ChainSupport
}

//配置包含ETCDraft配置
type Config struct {
WALDir  string //<my channel>的wal数据存储在waldir/<my channel>
SnapDir string //<my channel>的快照存储在snapdir/<my channel>
}

//同意人执行ETDDraft同意人
type Consenter struct {
	Dialer        *cluster.PredicateDialer
	Communication cluster.Communicator
	*Dispatcher
	Chains         ChainGetter
	Logger         *flogging.FabricLogger
	EtcdRaftConfig Config
	OrdererConfig  localconfig.TopLevel
	Cert           []byte
}

//TargetChannel从给定的proto.message中提取通道。
//失败时返回空字符串。
func (c *Consenter) TargetChannel(message proto.Message) string {
	switch req := message.(type) {
	case *orderer.StepRequest:
		return req.Channel
	case *orderer.SubmitRequest:
		return req.Channel
	default:
		return ""
	}
}

//ReceiverByChain返回给定channelID或nil的messageReceiver
//如果找不到。
func (c *Consenter) ReceiverByChain(channelID string) MessageReceiver {
	cs := c.Chains.GetChain(channelID)
	if cs == nil {
		return nil
	}
	if cs.Chain == nil {
		c.Logger.Panicf("Programming error - Chain %s is nil although it exists in the mapping", channelID)
	}
	if etcdRaftChain, isEtcdRaftChain := cs.Chain.(*Chain); isEtcdRaftChain {
		return etcdRaftChain
	}
	c.Logger.Warningf("Chain %s is of type %v and not etcdraft.Chain", channelID, reflect.TypeOf(cs.Chain))
	return nil
}

func (c *Consenter) detectSelfID(consenters map[uint64]*etcdraft.Consenter) (uint64, error) {
	var serverCertificates []string
	for nodeID, cst := range consenters {
		serverCertificates = append(serverCertificates, string(cst.ServerTlsCert))
		if bytes.Equal(c.Cert, cst.ServerTlsCert) {
			return nodeID, nil
		}
	}

	c.Logger.Error("Could not find", string(c.Cert), "among", serverCertificates)
	return 0, errors.Errorf("failed to detect own Raft ID because no matching certificate found")
}

//handlechain返回新的链实例或失败时出错
func (c *Consenter) HandleChain(support consensus.ConsenterSupport, metadata *common.Metadata) (consensus.Chain, error) {
	m := &etcdraft.Metadata{}
	if err := proto.Unmarshal(support.SharedConfig().ConsensusMetadata(), m); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal consensus metadata")
	}

	if m.Options == nil {
		return nil, errors.New("etcdraft options have not been provided")
	}

//确定每个节点到其ID的raft副本集映射
//对于新启动的链，我们需要读取并初始化raft
//通过在Conseter及其ID之间创建映射来创建元数据。
//如果链重新启动，我们将恢复raft元数据
//来自最近提交的块元数据的信息
//字段。
	raftMetadata, err := readRaftMetadata(metadata, m)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read Raft metadata")
	}

	id, err := c.detectSelfID(raftMetadata.Consenters)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	bp, err := newBlockPuller(support, c.Dialer, c.OrdererConfig.General.Cluster)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	opts := Options{
		RaftID:        id,
		Clock:         clock.NewClock(),
		MemoryStorage: raft.NewMemoryStorage(),
		Logger:        c.Logger,

		TickInterval:    time.Duration(m.Options.TickInterval) * time.Millisecond,
		ElectionTick:    int(m.Options.ElectionTick),
		HeartbeatTick:   int(m.Options.HeartbeatTick),
		MaxInflightMsgs: int(m.Options.MaxInflightMsgs),
		MaxSizePerMsg:   m.Options.MaxSizePerMsg,
		SnapInterval:    m.Options.SnapshotInterval,

		RaftMetadata: raftMetadata,

		WALDir:  path.Join(c.EtcdRaftConfig.WALDir, support.ChainID()),
		SnapDir: path.Join(c.EtcdRaftConfig.SnapDir, support.ChainID()),
	}

	rpc := &cluster.RPC{
		Channel:             support.ChainID(),
		Comm:                c.Communication,
		DestinationToStream: make(map[uint64]orderer.Cluster_SubmitClient),
	}
	return NewChain(support, opts, c.Communication, rpc, bp, nil)
}

func readRaftMetadata(blockMetadata *common.Metadata, configMetadata *etcdraft.Metadata) (*etcdraft.RaftMetadata, error) {
	m := &etcdraft.RaftMetadata{
		Consenters:      map[uint64]*etcdraft.Consenter{},
		NextConsenterId: 1,
	}
if blockMetadata != nil && len(blockMetadata.Value) != 0 { //我们有同意者从街区地图
		if err := proto.Unmarshal(blockMetadata.Value, m); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal block's metadata")
		}
		return m, nil
	}

//需要从配置中读取同意者
	for _, consenter := range configMetadata.Consenters {
		m.Consenters[m.NextConsenterId] = consenter
		m.NextConsenterId++
	}

	return m, nil
}

//新建一个ETCDraft同意者
func New(clusterDialer *cluster.PredicateDialer, conf *localconfig.TopLevel,
	srvConf comm.ServerConfig, srv *comm.GRPCServer, r *multichannel.Registrar) *Consenter {
	logger := flogging.MustGetLogger("orderer.consensus.etcdraft")

	var cfg Config
	if err := viperutil.Decode(conf.Consensus, &cfg); err != nil {
		logger.Panicf("Failed to decode etcdraft configuration: %s", err)
	}

	consenter := &Consenter{
		Cert:           srvConf.SecOpts.Certificate,
		Logger:         logger,
		Chains:         r,
		EtcdRaftConfig: cfg,
		OrdererConfig:  *conf,
		Dialer:         clusterDialer,
	}
	consenter.Dispatcher = &Dispatcher{
		Logger:        logger,
		ChainSelector: consenter,
	}

	comm := createComm(clusterDialer, conf, consenter)
	consenter.Communication = comm
	svc := &cluster.Service{
		StepLogger: flogging.MustGetLogger("orderer.common.cluster.step"),
		Logger:     flogging.MustGetLogger("orderer.common.cluster"),
		Dispatcher: comm,
	}
	orderer.RegisterClusterServer(srv.Server(), svc)
	return consenter
}

func createComm(clusterDialer *cluster.PredicateDialer,
	conf *localconfig.TopLevel,
	c *Consenter) *cluster.Comm {
	comm := &cluster.Comm{
		Logger:       flogging.MustGetLogger("orderer.common.cluster"),
		Chan2Members: make(map[string]cluster.MemberMapping),
		Connections:  cluster.NewConnectionStore(clusterDialer),
		RPCTimeout:   conf.General.Cluster.RPCTimeout,
		ChanExt:      c,
		H:            c,
	}
	c.Communication = comm
	return comm
}
