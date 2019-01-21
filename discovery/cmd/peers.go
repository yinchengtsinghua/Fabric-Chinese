
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


package discovery

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/cmd/common"
	"github.com/hyperledger/fabric/discovery/client"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

//newpeerCmd使用给定的存根和响应分析器创建一个新的peerCmd
func NewPeerCmd(stub Stub, parser ResponseParser) *PeerCmd {
	return &PeerCmd{
		stub:   stub,
		parser: parser,
	}
}

//PeerCmd executes channelPeer listing command
type PeerCmd struct {
	stub    Stub
	server  *string
	channel *string
	parser  ResponseParser
}

//setserver设置peerCmd的服务器
func (pc *PeerCmd) SetServer(server *string) {
	pc.server = server
}

//setchannel设置peerCmd的通道
func (pc *PeerCmd) SetChannel(channel *string) {
	pc.channel = channel
}

//执行执行命令
func (pc *PeerCmd) Execute(conf common.Config) error {
	channel := ""

	if pc.channel != nil {
		channel = *pc.channel
	}

	if pc.server == nil || *pc.server == "" {
		return errors.New("no server specified")
	}

	server := *pc.server

	req := discovery.NewRequest()
	if channel != "" {
		req = req.OfChannel(channel)
		req = req.AddPeersQuery()
	} else {
		req = req.AddLocalPeersQuery()
	}
	res, err := pc.stub.Send(server, conf, req)
	if err != nil {
		return err
	}
	return pc.parser.ParseResponse(channel, res)
}

//PeerResponseParser分析通道对等响应
type PeerResponseParser struct {
	io.Writer
}

//ParseResponse解析给定通道的给定响应
func (parser *PeerResponseParser) ParseResponse(channel string, res ServiceResponse) error {
	var listPeers peerLister
	if channel == "" {
		listPeers = res.ForLocal()
	} else {
		listPeers = &simpleChannelResponse{res.ForChannel(channel)}
	}
	peers, err := listPeers.Peers()
	if err != nil {
		return err
	}

	channelState := channel != ""
	b, _ := json.MarshalIndent(assemblePeers(peers, channelState), "", "\t")
	fmt.Fprintln(parser.Writer, string(b))
	return nil
}

func assemblePeers(peers []*discovery.Peer, withChannelState bool) interface{} {
	if withChannelState {
		var peerSlices []channelPeer
		for _, p := range peers {
			peerSlices = append(peerSlices, rawPeerToChannelPeer(p))
		}
		return peerSlices
	}
	var peerSlices []localPeer
	for _, p := range peers {
		peerSlices = append(peerSlices, rawPeerToLocalPeer(p))
	}
	return peerSlices
}

type channelPeer struct {
	MSPID        string
	LedgerHeight uint64
	Endpoint     string
	Identity     string
	Chaincodes   []string
}

type localPeer struct {
	MSPID    string
	Endpoint string
	Identity string
}

type peerLister interface {
	Peers() ([]*discovery.Peer, error)
}

type simpleChannelResponse struct {
	discovery.ChannelResponse
}

func (scr *simpleChannelResponse) Peers() ([]*discovery.Peer, error) {
	return scr.ChannelResponse.Peers()
}

func rawPeerToChannelPeer(p *discovery.Peer) channelPeer {
	var ledgerHeight uint64
	var ccs []string
	if p.StateInfoMessage != nil && p.StateInfoMessage.GetStateInfo() != nil && p.StateInfoMessage.GetStateInfo().Properties != nil {
		properties := p.StateInfoMessage.GetStateInfo().Properties
		ledgerHeight = properties.LedgerHeight
		for _, cc := range properties.Chaincodes {
			if cc == nil {
				continue
			}
			ccs = append(ccs, cc.Name)
		}
	}
	var endpoint string
	if p.AliveMessage != nil && p.AliveMessage.GetAliveMsg() != nil && p.AliveMessage.GetAliveMsg().Membership != nil {
		endpoint = p.AliveMessage.GetAliveMsg().Membership.Endpoint
	}
	sID := &msp.SerializedIdentity{}
	proto.Unmarshal(p.Identity, sID)
	return channelPeer{
		MSPID:        p.MSPID,
		Endpoint:     endpoint,
		LedgerHeight: ledgerHeight,
		Identity:     string(sID.IdBytes),
		Chaincodes:   ccs,
	}
}

func rawPeerToLocalPeer(p *discovery.Peer) localPeer {
	var endpoint string
	if p.AliveMessage != nil && p.AliveMessage.GetAliveMsg() != nil && p.AliveMessage.GetAliveMsg().Membership != nil {
		endpoint = p.AliveMessage.GetAliveMsg().Membership.Endpoint
	}
	sID := &msp.SerializedIdentity{}
	proto.Unmarshal(p.Identity, sID)
	return localPeer{
		MSPID:    p.MSPID,
		Endpoint: endpoint,
		Identity: string(sID.IdBytes),
	}
}
