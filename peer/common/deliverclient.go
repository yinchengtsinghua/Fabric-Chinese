
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


package common

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/peer/common/api"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var (
	logger = flogging.MustGetLogger("cli.common")

	seekNewest = &ab.SeekPosition{
		Type: &ab.SeekPosition_Newest{
			Newest: &ab.SeekNewest{},
		},
	}
	seekOldest = &ab.SeekPosition{
		Type: &ab.SeekPosition_Oldest{
			Oldest: &ab.SeekOldest{},
		},
	}
)

//DeliverClient保存连接客户端所需的信息
//到订购方/对等交付服务
type DeliverClient struct {
	Service     api.DeliverService
	ChannelID   string
	TLSCertHash []byte
}

func (d *DeliverClient) seekSpecified(blockNumber uint64) error {
	seekPosition := &ab.SeekPosition{
		Type: &ab.SeekPosition_Specified{
			Specified: &ab.SeekSpecified{
				Number: blockNumber,
			},
		},
	}
	env := seekHelper(d.ChannelID, seekPosition, d.TLSCertHash)
	return d.Service.Send(env)
}

func (d *DeliverClient) seekOldest() error {
	env := seekHelper(d.ChannelID, seekOldest, d.TLSCertHash)
	return d.Service.Send(env)
}

func (d *DeliverClient) seekNewest() error {
	env := seekHelper(d.ChannelID, seekNewest, d.TLSCertHash)
	return d.Service.Send(env)
}

func (d *DeliverClient) readBlock() (*cb.Block, error) {
	msg, err := d.Service.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "error receiving")
	}
	switch t := msg.Type.(type) {
	case *ab.DeliverResponse_Status:
		logger.Infof("Got status: %v", t)
		return nil, errors.Errorf("can't read the block: %v", t)
	case *ab.DeliverResponse_Block:
		logger.Infof("Received block: %v", t.Block.Header.Number)
d.Service.Recv() //刷新成功消息
		return t.Block, nil
	default:
		return nil, errors.Errorf("response error: unknown type %T", t)
	}
}

//GetSpecifiedBlock从对等方/订购方的交货中获取指定的块
//服务
func (d *DeliverClient) GetSpecifiedBlock(num uint64) (*cb.Block, error) {
	err := d.seekSpecified(num)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting specified block")
	}

	return d.readBlock()
}

//GetOldestBlock从对等方/订购方的传递服务获取最旧的块
func (d *DeliverClient) GetOldestBlock() (*cb.Block, error) {
	err := d.seekOldest()
	if err != nil {
		return nil, errors.WithMessage(err, "error getting oldest block")
	}

	return d.readBlock()
}

//getnewestblock从对等方/订购方的传递服务获取最新的块
func (d *DeliverClient) GetNewestBlock() (*cb.Block, error) {
	err := d.seekNewest()
	if err != nil {
		return nil, errors.WithMessage(err, "error getting newest block")
	}

	return d.readBlock()
}

//关闭关闭传递客户端的连接
func (d *DeliverClient) Close() error {
	return d.Service.CloseSend()
}

func seekHelper(channelID string, position *ab.SeekPosition, tlsCertHash []byte) *cb.Envelope {
	seekInfo := &ab.SeekInfo{
		Start:    position,
		Stop:     position,
		Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
	}

	env, err := utils.CreateSignedEnvelopeWithTLSBinding(
		cb.HeaderType_DELIVER_SEEK_INFO,
		channelID,
		localmsp.NewSigner(),
		seekInfo,
		int32(0),
		uint64(0),
		tlsCertHash,
	)
	if err != nil {
		logger.Errorf("Error signing envelope:  %s", err)
		return nil
	}

	return env
}

type ordererDeliverService struct {
	ab.AtomicBroadcast_DeliverClient
}

//newdeliverclientfororder从orderclient创建新的deliverclient
func NewDeliverClientForOrderer(channelID string) (*DeliverClient, error) {
	var tlsCertHash []byte
	oc, err := NewOrdererClientFromEnv()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create deliver client")
	}

	dc, err := oc.Deliver()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create deliver client")
	}
//检查客户端证书并创建哈希（如果存在）
	if len(oc.Certificate().Certificate) > 0 {
		tlsCertHash = util.ComputeSHA256(oc.Certificate().Certificate[0])
	}
	ds := &ordererDeliverService{dc}
	o := &DeliverClient{
		Service:     ds,
		ChannelID:   channelID,
		TLSCertHash: tlsCertHash,
	}
	return o, nil
}

type peerDeliverService struct {
	pb.Deliver_DeliverClient
}

//NewDeliverClientforPeer从对等客户端创建新的DeliverClient
func NewDeliverClientForPeer(channelID string) (*DeliverClient, error) {
	var tlsCertHash []byte
	pc, err := NewPeerClientFromEnv()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create deliver client")
	}

	d, err := pc.Deliver()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create deliver client")
	}

//检查客户端证书并创建哈希（如果存在）
	if len(pc.Certificate().Certificate) > 0 {
		tlsCertHash = util.ComputeSHA256(pc.Certificate().Certificate[0])
	}
	ds := &peerDeliverService{d}
	p := &DeliverClient{
		Service:     ds,
		ChannelID:   channelID,
		TLSCertHash: tlsCertHash,
	}
	return p, nil
}

func (p *peerDeliverService) Recv() (*ab.DeliverResponse, error) {
	pbResp, err := p.Deliver_DeliverClient.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "error receiving from peer deliver service")
	}

	abResp := &ab.DeliverResponse{}

	switch t := pbResp.Type.(type) {
	case *pb.DeliverResponse_Status:
		abResp.Type = &ab.DeliverResponse_Status{Status: t.Status}
	case *pb.DeliverResponse_Block:
		abResp.Type = &ab.DeliverResponse_Block{Block: t.Block}
	default:
		return nil, errors.Errorf("response error: unknown type %T", t)
	}

	return abResp, nil
}
