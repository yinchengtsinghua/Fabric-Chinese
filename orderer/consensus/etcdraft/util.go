
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
	"encoding/pem"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/configtx"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//成员身份更改保留有关成员身份的信息
//配置更新期间引入的更改
type MembershipChanges struct {
	AddedNodes   []*etcdraft.Consenter
	RemovedNodes []*etcdraft.Consenter
	TotalChanges uint32
}

//根据成员资格更改和raftmetadata方法计算更新raftmetadata和confchange
//将应用于raft集群配置的更新，以及更新之间的映射
//元数据中的同意者及其ID
func (mc *MembershipChanges) UpdateRaftMetadataAndConfChange(raftMetadata *etcdraft.RaftMetadata) *raftpb.ConfChange {
	if mc == nil || mc.TotalChanges == 0 {
		return nil
	}

	var confChange *raftpb.ConfChange

//产生相应的筏形结构变化
	if len(mc.AddedNodes) > 0 {
		nodeID := raftMetadata.NextConsenterId
		raftMetadata.Consenters[nodeID] = mc.AddedNodes[0]
		raftMetadata.NextConsenterId++
		confChange = &raftpb.ConfChange{
			ID:     raftMetadata.ConfChangeCounts,
			NodeID: nodeID,
			Type:   raftpb.ConfChangeAddNode,
		}
		raftMetadata.ConfChangeCounts++
		return confChange
	}

	if len(mc.RemovedNodes) > 0 {
		for _, c := range mc.RemovedNodes {
			for nodeID, node := range raftMetadata.Consenters {
				if bytes.Equal(c.ClientTlsCert, node.ClientTlsCert) {
					delete(raftMetadata.Consenters, nodeID)
					confChange = &raftpb.ConfChange{
						ID:     raftMetadata.ConfChangeCounts,
						NodeID: nodeID,
						Type:   raftpb.ConfChangeRemoveNode,
					}
					raftMetadata.ConfChangeCounts++
					break
				}
			}
		}
	}

	return confChange
}

//EndpointConfigFromSupport从同意者支持提取TLS CA证书和终结点
func EndpointconfigFromFromSupport(support consensus.ConsenterSupport) (*cluster.EndpointConfig, error) {
	lastConfigBlock, err := lastConfigBlockFromSupport(support)
	if err != nil {
		return nil, err
	}
	endpointconf, err := cluster.EndpointconfigFromConfigBlock(lastConfigBlock)
	if err != nil {
		return nil, err
	}
	return endpointconf, nil
}

func lastConfigBlockFromSupport(support consensus.ConsenterSupport) (*common.Block, error) {
	lastBlockSeq := support.Height() - 1
	lastBlock := support.Block(lastBlockSeq)
	if lastBlock == nil {
		return nil, errors.Errorf("unable to retrieve block %d", lastBlockSeq)
	}
	lastConfigBlock, err := LastConfigBlock(lastBlock, support)
	if err != nil {
		return nil, err
	}
	return lastConfigBlock, nil
}

//last config block返回相对于给定块的最后一个配置块。
func LastConfigBlock(block *common.Block, support consensus.ConsenterSupport) (*common.Block, error) {
	if block == nil {
		return nil, errors.New("nil block")
	}
	if support == nil {
		return nil, errors.New("nil support")
	}
	if block.Metadata == nil || len(block.Metadata.Metadata) <= int(common.BlockMetadataIndex_LAST_CONFIG) {
		return nil, errors.New("no metadata in block")
	}
	lastConfigBlockNum, err := utils.GetLastConfigIndexFromBlock(block)
	if err != nil {
		return nil, err
	}
	lastConfigBlock := support.Block(lastConfigBlockNum)
	if lastConfigBlock == nil {
		return nil, errors.Errorf("unable to retrieve last config block %d", lastConfigBlockNum)
	}
	return lastConfigBlock, nil
}

//new block puller创建新的块puller
func newBlockPuller(support consensus.ConsenterSupport,
	baseDialer *cluster.PredicateDialer,
	clusterConfig localconfig.Cluster) (*cluster.BlockPuller, error) {

	verifyBlockSequence := func(blocks []*common.Block) error {
		return cluster.VerifyBlocks(blocks, support)
	}

	secureConfig, err := baseDialer.ClientConfig()
	if err != nil {
		return nil, err
	}
	secureConfig.AsyncConnect = false
	stdDialer := &cluster.StandardDialer{
		Dialer: cluster.NewTLSPinningDialer(secureConfig),
	}

//从配置中提取TLS CA证书和终结点，
	endpointConfig, err := EndpointconfigFromFromSupport(support)
	if err != nil {
		return nil, err
	}
//并覆盖它们。
	secureConfig.SecOpts.ServerRootCAs = endpointConfig.TLSRootCAs
	stdDialer.Dialer.SetConfig(secureConfig)

	der, _ := pem.Decode(secureConfig.SecOpts.Certificate)
	if der == nil {
		return nil, errors.Errorf("client certificate isn't in PEM format: %v",
			string(secureConfig.SecOpts.Certificate))
	}

	return &cluster.BlockPuller{
		VerifyBlockSequence: verifyBlockSequence,
		Logger:              flogging.MustGetLogger("orderer.common.cluster.puller"),
		RetryTimeout:        clusterConfig.ReplicationRetryTimeout,
		MaxTotalBufferBytes: clusterConfig.ReplicationBufferSize,
		FetchTimeout:        clusterConfig.ReplicationPullTimeout,
		Endpoints:           endpointConfig.Endpoints,
		Signer:              support,
		TLSCert:             der.Bytes,
		Channel:             support.ChainID(),
		Dialer:              stdDialer,
	}, nil
}

//raftpeers将同意者映射到raft.peer的切片。
func RaftPeers(consenters map[uint64]*etcdraft.Consenter) []raft.Peer {
	var peers []raft.Peer

	for raftID := range consenters {
		peers = append(peers, raft.Peer{ID: raftID})
	}
	return peers
}

//同意者映射同意者到密钥为客户端TLS证书的集合中
func ConsentersToMap(consenters []*etcdraft.Consenter) map[string]struct{} {
	set := map[string]struct{}{}
	for _, c := range consenters {
		set[string(c.ClientTlsCert)] = struct{}{}
	}
	return set
}

//membershipbycert将同意者映射转换为由映射封装的集
//其中密钥是客户端TLS证书
func MembershipByCert(consenters map[uint64]*etcdraft.Consenter) map[string]struct{} {
	set := map[string]struct{}{}
	for _, c := range consenters {
		set[string(c.ClientTlsCert)] = struct{}{}
	}
	return set
}

//ComputemMembershipChanges根据有关新使用者和返回的信息计算成员身份更新
//两片：一片添加的同意者和一片要移除的同意者
func ComputeMembershipChanges(oldConsenters map[uint64]*etcdraft.Consenter, newConsenters []*etcdraft.Consenter) *MembershipChanges {
	result := &MembershipChanges{
		AddedNodes:   []*etcdraft.Consenter{},
		RemovedNodes: []*etcdraft.Consenter{},
	}

	currentConsentersSet := MembershipByCert(oldConsenters)
	for _, c := range newConsenters {
		if _, exists := currentConsentersSet[string(c.ClientTlsCert)]; !exists {
			result.AddedNodes = append(result.AddedNodes, c)
			result.TotalChanges++
		}
	}

	newConsentersSet := ConsentersToMap(newConsenters)
	for _, c := range oldConsenters {
		if _, exists := newConsentersSet[string(c.ClientTlsCert)]; !exists {
			result.RemovedNodes = append(result.RemovedNodes, c)
			result.TotalChanges++
		}
	}

	return result
}

//MetadataFromConfigValue从配置值读取配置更新并将其转换为raft元数据
func MetadataFromConfigValue(configValue *common.ConfigValue) (*etcdraft.Metadata, error) {
	consensusTypeValue := &orderer.ConsensusType{}
	if err := proto.Unmarshal(configValue.Value, consensusTypeValue); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal consensusType config update")
	}

	updatedMetadata := &etcdraft.Metadata{}
	if err := proto.Unmarshal(consensusTypeValue.Metadata, updatedMetadata); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal updated (new) etcdraft metadata configuration")
	}

	return updatedMetadata, nil
}

//MetadataFromConfigUpdate从配置更新中提取共识元数据
func MetadataFromConfigUpdate(update *common.ConfigUpdate) (*etcdraft.Metadata, error) {
	if ordererConfigGroup, ok := update.WriteSet.Groups["Orderer"]; ok {
		if val, ok := ordererConfigGroup.Values["ConsensusType"]; ok {
			return MetadataFromConfigValue(val)
		}
	}
	return nil, nil
}

//configDevelopeFromBlock基于
//配置类型，即headertype_order_transaction或headertype_config
func ConfigEnvelopeFromBlock(block *common.Block) (*common.Envelope, error) {
	if block == nil {
		return nil, errors.New("nil block")
	}

	envelope, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract envelop from the block")
	}

	channelHeader, err := utils.ChannelHeader(envelope)
	if err != nil {
		return nil, errors.Wrap(err, "cannot extract channel header")
	}

	switch channelHeader.Type {
	case int32(common.HeaderType_ORDERER_TRANSACTION):
		payload, err := utils.UnmarshalPayload(envelope.Payload)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal envelope to extract config payload for orderer transaction")
		}
		configEnvelop, err := utils.UnmarshalEnvelope(payload.Data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal config envelope for orderer type transaction")
		}

		return configEnvelop, nil
	case int32(common.HeaderType_CONFIG):
		return envelope, nil
	default:
		return nil, errors.Errorf("unexpected header type: %v", channelHeader.Type)
	}
}

//conensusMetadataFromConfigBlock从配置块读取共识元数据更新
func ConsensusMetadataFromConfigBlock(block *common.Block) (*etcdraft.Metadata, error) {
	if block == nil {
		return nil, errors.New("nil block")
	}

	if !utils.IsConfigBlock(block) {
		return nil, errors.New("not a config block")
	}

	configEnvelope, err := ConfigEnvelopeFromBlock(block)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read config update")
	}

	payload, err := utils.ExtractPayload(configEnvelope)
	if err != nil {
		return nil, errors.Wrap(err, "failed extract payload from config envelope")
	}
//获取配置更新
	configUpdate, err := configtx.UnmarshalConfigUpdateFromPayload(payload)
	if err != nil {
		return nil, errors.Wrap(err, "could not read config update")
	}

	return MetadataFromConfigUpdate(configUpdate)
}

//每当block为config block并携带时，ismembershipupdate检查
//RAFT集群成员更新
func IsMembershipUpdate(block *common.Block, currentMetadata *etcdraft.RaftMetadata) (bool, error) {
	if !utils.IsConfigBlock(block) {
		return false, nil
	}

	metadata, err := ConsensusMetadataFromConfigBlock(block)
	if err != nil {
		return false, errors.Wrap(err, "error reading consensus metadata")
	}

	if metadata != nil {
		changes := ComputeMembershipChanges(currentMetadata.Consenters, metadata.Consenters)

		return changes.TotalChanges > 0, nil
	}

	return false, nil
}

//同意证书表示同意人的TLS证书
type ConsenterCertificate []byte

//isconsenterfchannel返回调用方是否同意某个通道
//通过检查给定的配置块。
//如果为真，则返回零，否则返回错误。
func (conCert ConsenterCertificate) IsConsenterOfChannel(configBlock *common.Block) error {
	if configBlock == nil {
		return errors.New("nil block")
	}
	envelopeConfig, err := utils.ExtractEnvelope(configBlock, 0)
	if err != nil {
		return err
	}
	bundle, err := channelconfig.NewBundleFromEnvelope(envelopeConfig)
	if err != nil {
		return err
	}
	oc, exists := bundle.OrdererConfig()
	if !exists {
		return errors.New("no orderer config in bundle")
	}
	m := &etcdraft.Metadata{}
	if err := proto.Unmarshal(oc.ConsensusMetadata(), m); err != nil {
		return err
	}

	for _, consenter := range m.Consenters {
		if bytes.Equal(conCert, consenter.ServerTlsCert) || bytes.Equal(conCert, consenter.ClientTlsCert) {
			return nil
		}
	}
	return cluster.ErrNotInChannel
}
