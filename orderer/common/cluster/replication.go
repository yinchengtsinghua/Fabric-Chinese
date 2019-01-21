
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


package cluster

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"time"

	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

const (
//retryTimeout是拉块器重试的时间。
	RetryTimeout = time.Second * 10
)

//pullerConfigFromTopLevelConfig从顶级配置创建pullerConfig，
//以及来自签名者和TLS密钥证书对。
//pullerconfig的通道初始化为系统通道。
func PullerConfigFromTopLevelConfig(systemChannel string, conf *localconfig.TopLevel, tlsKey, tlsCert []byte, signer crypto.LocalSigner) PullerConfig {
	return PullerConfig{
		Channel:             systemChannel,
		MaxTotalBufferBytes: conf.General.Cluster.ReplicationBufferSize,
		Timeout:             conf.General.Cluster.RPCTimeout,
		TLSKey:              tlsKey,
		TLSCert:             tlsCert,
		Signer:              signer,
	}
}

//去：生成mokery-dir。-名称LedgerWriter-大小写下划线-输出模拟/

//LedgerWriter允许调用方编写块并检查高度
type LedgerWriter interface {
//将新块追加到分类帐
	Append(block *common.Block) error

//height返回分类帐上的块数
	Height() uint64
}

//去：生成mokery-dir。-名称分类目录-案例下划线-输出模拟/

//LedgerFactory按链ID检索或创建新的分类帐
type LedgerFactory interface {
//getorcreate获取现有分类帐（如果存在）
//或者如果没有创建它
	GetOrCreate(chainID string) (LedgerWriter, error)
}

//去：生成mokery-dir。-name channellister-case underline-输出模拟/

//channellister返回频道列表
type ChannelLister interface {
//通道返回通道列表
	Channels() []string
//关闭关闭频道列表器
	Close()
}

//复制器复制链
type Replicator struct {
	SystemChannel    string
	ChannelLister    ChannelLister
	Logger           *flogging.FabricLogger
	Puller           *BlockPuller
	BootBlock        *common.Block
	AmIPartOfChannel selfMembershipPredicate
	LedgerFactory    LedgerFactory
}

//is replication needed返回是否需要复制，
//或者集群节点可以恢复标准引导流。
func (r *Replicator) IsReplicationNeeded() (bool, error) {
	systemChannelLedger, err := r.LedgerFactory.GetOrCreate(r.SystemChannel)
	if err != nil {
		return false, err
	}

	height := systemChannelLedger.Height()
	var lastBlockSeq uint64
//如果高度为0，则LastBlockSeq将为2^64-1，
//所以让它0来处理溢出。
	if height == 0 {
		lastBlockSeq = 0
	} else {
		lastBlockSeq = height - 1
	}

	if r.BootBlock.Header.Number > lastBlockSeq {
		return true, nil
	}
	return false, nil
}

//复制者拉着锁链并承诺。
func (r *Replicator) ReplicateChains() {
	channels := r.discoverChannels()
	channels2Pull := r.channelsToPull(channels)
	r.Logger.Info("Found myself in", len(channels2Pull), "channels:", channels2Pull)
	for _, channel := range channels2Pull {
		r.PullChannel(channel)
	}
//最后，拉动系统链条
	if err := r.PullChannel(r.SystemChannel); err != nil {
		r.Logger.Panicf("Failed pulling system channel: %v", err)
	}
}

func (r *Replicator) discoverChannels() []string {
	r.Logger.Debug("Entering")
	defer r.Logger.Debug("Exiting")
	channels := r.ChannelLister.Channels()
	r.Logger.Info("Discovered", len(channels), "channels:", channels)
	r.ChannelLister.Close()
	return channels
}

//pullchannel从某个订购者中提取给定的通道，
//并将其提交到分类帐。
func (r *Replicator) PullChannel(channel string) error {
	r.Logger.Info("Pulling channel", channel)
	puller := r.Puller.Clone()
	defer puller.Close()
	puller.Channel = channel

	endpoint, latestHeight := latestHeightAndEndpoint(puller)
	if endpoint == "" {
		return errors.Errorf("failed obtaining the latest block for channel %s", channel)
	}
	r.Logger.Info("Latest block height for channel", channel, "is", latestHeight)
//确保如果我们拉动系统通道，则后面的光大于或等于
//系统通道的引导程序块。
//否则，我们会留下一个空白。
	if channel == r.SystemChannel && latestHeight-1 < r.BootBlock.Header.Number {
		return errors.Errorf("latest height found among system channel(%s) orderers is %d, but the boot block's "+
			"sequence is %d", r.SystemChannel, latestHeight, r.BootBlock.Header.Number)
	}
	return r.pullChannelBlocks(channel, puller, latestHeight)
}

func (r *Replicator) pullChannelBlocks(channel string, puller ChainPuller, latestHeight uint64) error {
	ledger, err := r.LedgerFactory.GetOrCreate(channel)
	if err != nil {
		r.Logger.Panicf("Failed to create a ledger for channel %s: %v", channel, err)
	}
//拉动Genesis块并记住它的散列值。
	genesisBlock := puller.PullBlock(0)
	r.appendBlock(genesisBlock, ledger)
	actualPrevHash := genesisBlock.Header.Hash()

	for seq := uint64(1); seq < latestHeight; seq++ {
		block := puller.PullBlock(seq)
		reportedPrevHash := block.Header.PreviousHash
		if !bytes.Equal(reportedPrevHash, actualPrevHash) {
			return errors.Errorf("block header mismatch on sequence %d, expected %x, got %x",
				block.Header.Number, actualPrevHash, reportedPrevHash)
		}
		actualPrevHash = block.Header.Hash()
		if channel == r.SystemChannel && block.Header.Number == r.BootBlock.Header.Number {
			r.compareBootBlockWithSystemChannelLastConfigBlock(block)
			r.appendBlock(block, ledger)
//无需再从系统通道中拉出更多块
			return nil
		}
		r.appendBlock(block, ledger)
	}
	return nil
}

func (r *Replicator) appendBlock(block *common.Block, ledger LedgerWriter) {
	if err := ledger.Append(block); err != nil {
		r.Logger.Panicf("Failed to write block %d: %v", block.Header.Number, err)
	}
}

func (r *Replicator) compareBootBlockWithSystemChannelLastConfigBlock(block *common.Block) {
//覆盖接收块的数据哈希
	block.Header.DataHash = block.Data.Hash()

	bootBlockHash := r.BootBlock.Header.Hash()
	retrievedBlockHash := block.Header.Hash()
	if bytes.Equal(bootBlockHash, retrievedBlockHash) {
		return
	}
	r.Logger.Panicf("Block header mismatch on last system channel block, expected %s, got %s",
		hex.EncodeToString(bootBlockHash), hex.EncodeToString(retrievedBlockHash))
}

func (r *Replicator) channelsToPull(channels []string) []string {
	r.Logger.Info("Will now pull channels:", channels)
	var channelsToPull []string
	for _, channel := range channels {
		r.Logger.Info("Pulling chain for", channel)
		puller := r.Puller.Clone()
		puller.Channel = channel
//当我们检查是否在通道中时禁用拉器缓冲，
//因为我们只需要知道一个街区。
		bufferSize := puller.MaxTotalBufferBytes
		puller.MaxTotalBufferBytes = 1
		err := Participant(puller, r.AmIPartOfChannel)
		puller.Close()
//恢复以前的缓冲区大小
		puller.MaxTotalBufferBytes = bufferSize
		if err == ErrNotInChannel {
			r.Logger.Info("I do not belong to channel", channel, ", skipping chain retrieval")
			continue
		}
		if err != nil {
			r.Logger.Panicf("Failed classifying whether I belong to channel %s: %v, skipping chain retrieval", channel, err)
			continue
		}
		channelsToPull = append(channelsToPull, channel)
	}
	return channelsToPull
}

//pullerconfig配置blockpuller。
type PullerConfig struct {
	TLSKey              []byte
	TLSCert             []byte
	Timeout             time.Duration
	Signer              crypto.LocalSigner
	Channel             string
	MaxTotalBufferBytes int
}

//BlockPullerFromConfigBlock返回一个不验证块上签名的BlockPuller。
func BlockPullerFromConfigBlock(conf PullerConfig, block *common.Block) (*BlockPuller, error) {
	if block == nil {
		return nil, errors.New("nil block")
	}

	endpointconfig, err := EndpointconfigFromConfigBlock(block)
	if err != nil {
		return nil, err
	}

	dialer := &StandardDialer{
		Dialer: NewTLSPinningDialer(comm.ClientConfig{
			Timeout: conf.Timeout,
			SecOpts: &comm.SecureOptions{
				ServerRootCAs:     endpointconfig.TLSRootCAs,
				Certificate:       conf.TLSCert,
				Key:               conf.TLSKey,
				RequireClientCert: true,
				UseTLS:            true,
			},
		})}

	tlsCertAsDER, _ := pem.Decode(conf.TLSCert)
	if tlsCertAsDER == nil {
		return nil, errors.Errorf("unable to decode TLS certificate PEM: %s", base64.StdEncoding.EncodeToString(conf.TLSCert))
	}

	return &BlockPuller{
		Logger:  flogging.MustGetLogger("orderer.common.cluster.replication"),
		Dialer:  dialer,
		TLSCert: tlsCertAsDER.Bytes,
		VerifyBlockSequence: func(blocks []*common.Block) error {
			return VerifyBlocks(blocks, &NoopBlockVerifier{})
		},
		MaxTotalBufferBytes: conf.MaxTotalBufferBytes,
		Endpoints:           endpointconfig.Endpoints,
		RetryTimeout:        RetryTimeout,
		FetchTimeout:        conf.Timeout,
		Channel:             conf.Channel,
		Signer:              conf.Signer,
	}, nil
}

//NoopBlockVerifier不验证块签名
type NoopBlockVerifier struct{}

//VerifyBlockSignature接受块上的所有签名。
func (*NoopBlockVerifier) VerifyBlockSignature(sd []*common.SignedData, config *common.ConfigEnvelope) error {
	return nil
}

//去：生成mokery-dir。-name chainpuller-case underline-输出模拟/

//拉链器从链条上拉块
type ChainPuller interface {
//PullBlock将给定的块从某个排序器节点中拉出
	PullBlock(seq uint64) *common.Block

//HeightsByEndpoints按排序器的端点返回块高度
	HeightsByEndpoints() map[string]uint64

//关闭关闭链条拉具
	Close()
}

//链条检查员走过链条
type ChainInspector struct {
	Logger          *flogging.FabricLogger
	Puller          ChainPuller
	LastConfigBlock *common.Block
}

//errnotinchannel表示排序节点不在通道中
var ErrNotInChannel = errors.New("not in the channel")

//SelfMembershipPredicate确定是否在给定的配置块中找到调用方
type selfMembershipPredicate func(configBlock *common.Block) error

//参与者返回调用方是否参与链。
//它接收一个链条拉具，该拉具应该已经针对链条进行校准，
//以及用于检测调用方是否应为链提供服务的SelfMembershipPredicate。
//如果调用方参与链，则返回nil。
//如果调用方不参与链，它可能返回NotInchannelError错误。
func Participant(puller ChainPuller, analyzeLastConfBlock selfMembershipPredicate) error {
	endpoint, latestHeight := latestHeightAndEndpoint(puller)
	if endpoint == "" {
		return errors.New("no available orderer")
	}
	lastBlock := puller.PullBlock(latestHeight - 1)
	lastConfNumber, err := lastConfigFromBlock(lastBlock)
	if err != nil {
		return err
	}
//最后一个配置块小于最新高度，
//服务器端的块迭代器是顺序迭代器。
//因此，如果我们想拉一个更早的块，我们需要重置拉器。
	puller.Close()
	lastConfigBlock := puller.PullBlock(lastConfNumber)
	return analyzeLastConfBlock(lastConfigBlock)
}

func latestHeightAndEndpoint(puller ChainPuller) (string, uint64) {
	var maxHeight uint64
	var mostUpToDateEndpoint string
	for endpoint, height := range puller.HeightsByEndpoints() {
		if height > maxHeight {
			maxHeight = height
			mostUpToDateEndpoint = endpoint
		}
	}
	return mostUpToDateEndpoint, maxHeight
}

func lastConfigFromBlock(block *common.Block) (uint64, error) {
	if block.Metadata == nil || len(block.Metadata.Metadata) <= int(common.BlockMetadataIndex_LAST_CONFIG) {
		return 0, errors.New("no metadata in block")
	}
	return utils.GetLastConfigIndexFromBlock(block)
}

//关闭关闭链检查器
func (ci *ChainInspector) Close() {
	ci.Puller.Close()
}

//通道返回通道列表
//存在于链条中的
func (ci *ChainInspector) Channels() []string {
	channels := make(map[string]struct{})
	lastConfigBlockNum := ci.LastConfigBlock.Header.Number
	var block *common.Block
	var prevHash []byte
	for seq := uint64(1); seq < lastConfigBlockNum; seq++ {
		block = ci.Puller.PullBlock(seq)
		ci.validateHashPointer(block, prevHash)
		channel, err := IsNewChannelBlock(block)
		if err != nil {
//如果我们未能对一个块进行分类，那么系统链中就出现了问题。
//我们正试图拉，所以放弃。
			ci.Logger.Panic("Failed classifying block", seq, ":", err)
			continue
		}
//为下一次迭代设置上一个哈希
		prevHash = block.Header.Hash()
		if channel == "" {
			ci.Logger.Info("Block", seq, "doesn't contain a new channel")
			continue
		}
		ci.Logger.Info("Block", seq, "contains channel", channel)
		channels[channel] = struct{}{}
	}
//此时，block保存对最后一个拉入的块的引用。
//我们确保提取的最后一个块的哈希是前一个哈希
//初始化了最后一个configblock。
//我们不需要验证我们拉的所有滑轮的整个链条，
//因为块拉器调用它拉的所有块的verifyblockhash。
	last2Blocks := []*common.Block{block, ci.LastConfigBlock}
	if err := VerifyBlockHash(1, last2Blocks); err != nil {
		ci.Logger.Panic("System channel pulled doesn't match the boot last config block:", err)
	}

	return flattenChannelMap(channels)
}

func (ci *ChainInspector) validateHashPointer(block *common.Block, prevHash []byte) {
	if prevHash == nil {
		return
	}
	if bytes.Equal(block.Header.PreviousHash, prevHash) {
		return
	}
	ci.Logger.Panicf("Claimed previous hash of block %d is %x but actual previous hash is %x",
		block.Header.Number, block.Header.PreviousHash, prevHash)
}

func flattenChannelMap(m map[string]struct{}) []string {
	var res []string
	for channel := range m {
		res = append(res, channel)
	}
	return res
}

//IsNewChannelBlock返回通道的名称，以防
//它保存一个通道创建事务，否则为空字符串。
func IsNewChannelBlock(block *common.Block) (string, error) {
	if block == nil {
		return "", errors.New("nil block")
	}
	env, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return "", err
	}
	payload, err := utils.ExtractPayload(env)
	if err != nil {
		return "", err
	}
	if payload.Header == nil {
		return "", errors.New("nil header in payload")
	}
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return "", err
	}
//该事务是订购方事务
	if common.HeaderType(chdr.Type) != common.HeaderType_ORDERER_TRANSACTION {
		return "", nil
	}
	systemChannelName := chdr.ChannelId
	innerEnvelope, err := utils.UnmarshalEnvelope(payload.Data)
	if err != nil {
		return "", err
	}
	innerPayload, err := utils.UnmarshalPayload(innerEnvelope.Payload)
	if err != nil {
		return "", err
	}
	if innerPayload.Header == nil {
		return "", errors.New("inner payload's header is nil")
	}
	chdr, err = utils.UnmarshalChannelHeader(innerPayload.Header.ChannelHeader)
	if err != nil {
		return "", err
	}
//内部负载的头是一个配置事务
	if common.HeaderType(chdr.Type) != common.HeaderType_CONFIG {
		return "", nil
	}
//在任何情况下，排除所有系统通道事务
	if chdr.ChannelId == systemChannelName {
		return "", nil
	}
	return chdr.ChannelId, nil
}
