
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


package privdata

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	privdatacommon "github.com/hyperledger/fabric/gossip/privdata/common"
	"github.com/hyperledger/fabric/gossip/util"
	fcommon "github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
)

const (
	membershipPollingBackoff    = time.Second
	responseWaitTime            = time.Second * 5
	maxMembershipPollIterations = 5
	btlPullMarginDefault        = 10
)

//dig2pvtrwsetwithconfig（配置）
type Dig2PvtRWSetWithConfig map[privdatacommon.DigKey]*util.PrivateRWSetWithConfig

//定义支持API的privatedataretriever接口
//检索所需的私有数据
type PrivateDataRetriever interface {
//collectionrwset返回给定txid的collectionpvtreadwriteset字节数，并从临时存储返回集合。
	CollectionRWSet(dig []*proto.PvtDataDigest, blockNum uint64) (Dig2PvtRWSetWithConfig, bool, error)
}

//八卦定义八卦模块为协调器提供的功能
type gossip interface {
//发送向远程对等发送消息
	Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer)

//peersofchannel返回被认为是活动的网络成员
//也订阅了给定的频道
	PeersOfChannel(common.ChainID) []discovery.NetworkMember

//PeerFilter接收子通道SelectionCriteria并返回一个选择
//只有符合给定标准的对等身份，并且他们发布了自己的渠道参与
	PeerFilter(channel common.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error)

//accept返回与某个谓词匹配的其他节点发送的消息的专用只读通道。
//如果passthrough为false，则消息将由八卦层预先处理。
//如果passthrough是真的，那么八卦层不会介入，消息也不会
//可用于将答复发送回发件人
	Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage)
}

type puller struct {
	pubSub        *util.PubSub
	stopChan      chan struct{}
	msgChan       <-chan proto.ReceivedMessage
	channel       string
	cs            privdata.CollectionStore
	btlPullMargin uint64
	gossip
	PrivateDataRetriever
	CollectionAccessFactory
}

//new puller创建新的私有数据puller
func NewPuller(cs privdata.CollectionStore, g gossip, dataRetriever PrivateDataRetriever, factory CollectionAccessFactory, channel string) *puller {
	p := &puller{
		pubSub:                  util.NewPubSub(),
		stopChan:                make(chan struct{}),
		channel:                 channel,
		cs:                      cs,
		btlPullMargin:           getBtlPullMargin(),
		gossip:                  g,
		PrivateDataRetriever:    dataRetriever,
		CollectionAccessFactory: factory,
	}
	_, p.msgChan = p.Accept(func(o interface{}) bool {
		msg := o.(proto.ReceivedMessage).GetGossipMessage()
		if !bytes.Equal(msg.Channel, []byte(p.channel)) {
			return false
		}
		return msg.IsPrivateDataMsg()
	}, true)
	go p.listen()
	return p
}

func (p *puller) listen() {
	for {
		select {
		case <-p.stopChan:
			return
		case msg := <-p.msgChan:
			if msg == nil {
//通信模块停止，因此该通道
//关闭
				return
			}
			if msg.GetGossipMessage().GetPrivateRes() != nil {
				p.handleResponse(msg)
			}
			if msg.GetGossipMessage().GetPrivateReq() != nil {
				p.handleRequest(msg)
			}
		}
	}
}

func (p *puller) handleRequest(message proto.ReceivedMessage) {
	logger.Debug("Got", message.GetGossipMessage(), "from", message.GetConnectionInfo().Endpoint)
	message.Respond(&proto.GossipMessage{
		Channel: []byte(p.channel),
		Tag:     proto.GossipMessage_CHAN_ONLY,
		Nonce:   message.GetGossipMessage().Nonce,
		Content: &proto.GossipMessage_PrivateRes{
			PrivateRes: &proto.RemotePvtDataResponse{
				Elements: p.createResponse(message),
			},
		},
	})
}

func (p *puller) createResponse(message proto.ReceivedMessage) []*proto.PvtDataElement {
	authInfo := message.GetConnectionInfo().Auth
	var returned []*proto.PvtDataElement
	connectionEndpoint := message.GetConnectionInfo().Endpoint

	defer func() {
		logger.Debug("Returning", connectionEndpoint, len(returned), "elements")
	}()

	msg := message.GetGossipMessage()
//按块编号对所有摘要进行分组
	block2dig := groupDigestsByBlockNum(msg.GetPrivateReq().Digests)

	for blockNum, digests := range block2dig {
		dig2rwSets, wasFetchedFromLedger, err := p.CollectionRWSet(digests, blockNum)
		if err != nil {
			logger.Warningf("could not obtain private collection rwset for block %d, because of %s, continue...", blockNum, err)
			continue
		}
		returned = append(returned, p.filterNotEligible(dig2rwSets, wasFetchedFromLedger, fcommon.SignedData{
			Identity:  message.GetConnectionInfo().Identity,
			Data:      authInfo.SignedData,
			Signature: authInfo.Signature,
		}, connectionEndpoint)...)
	}
	return returned
}

//GroupDigestsbyblocknum按块序列号分组所有摘要
func groupDigestsByBlockNum(digests []*proto.PvtDataDigest) map[uint64][]*proto.PvtDataDigest {
	results := make(map[uint64][]*proto.PvtDataDigest)
	for _, dig := range digests {
		results[dig.BlockSeq] = append(results[dig.BlockSeq], dig)
	}
	return results
}

func (p *puller) handleResponse(message proto.ReceivedMessage) {
	msg := message.GetGossipMessage().GetPrivateRes()
	logger.Debug("Got", msg, "from", message.GetConnectionInfo().Endpoint)
	for _, el := range msg.Elements {
		if el.Digest == nil {
			logger.Warning("Got nil digest from", message.GetConnectionInfo().Endpoint, "aborting")
			return
		}
		hash, err := el.Digest.Hash()
		if err != nil {
			logger.Warning("Failed hashing digest from", message.GetConnectionInfo().Endpoint, "aborting")
			return
		}
		p.pubSub.Publish(hash, el)
	}
}

func (p *puller) waitForMembership() []discovery.NetworkMember {
	polIteration := 0
	for {
		members := p.PeersOfChannel(common.ChainID(p.channel))
		if len(members) != 0 {
			return members
		}
		polIteration++
		if polIteration == maxMembershipPollIterations {
			return nil
		}
		time.Sleep(membershipPollingBackoff)
	}
}

func (p *puller) fetch(dig2src dig2sources) (*privdatacommon.FetchedPvtDataContainer, error) {
//ComputeFilters返回从摘要到路由筛选器的映射
	dig2Filter, err := p.computeFilters(dig2src)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return p.fetchPrivateData(dig2Filter)
}

func (p *puller) FetchReconciledItems(dig2collectionConfig privdatacommon.Dig2CollectionConfig) (*privdatacommon.FetchedPvtDataContainer, error) {
//ComputeFilters返回从摘要到路由筛选器的映射
	dig2Filter, err := p.computeReconciliationFilters(dig2collectionConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return p.fetchPrivateData(dig2Filter)
}

func (p *puller) fetchPrivateData(dig2Filter digestToFilterMapping) (*privdatacommon.FetchedPvtDataContainer, error) {
//获取每个通道的对等点列表
	allFilters := dig2Filter.flattenFilterValues()
	members := p.waitForMembership()
	logger.Debug("Total members in channel:", members)
	members = filter.AnyMatch(members, allFilters...)
	logger.Debug("Total members that fit some digest:", members)
	if len(members) == 0 {
		logger.Warning("Do not know any peer in the channel(", p.channel, ") that matches the policies , aborting")
		return nil, errors.New("Empty membership")
	}
	members = randomizeMemberList(members)
	res := &privdatacommon.FetchedPvtDataContainer{}
//将请求分发给对等方，并获取其所有消息的订阅
//MatchDigestTopeer返回一个对等点到Digests的映射，我们会要求它
	var peer2digests peer2Digests
//我们希望收集所有表示为摘要的私有RWset
	itemsLeftToCollect := len(dig2Filter)
//只要我们还有一些数据需要收集，新成员需要数据：
	for itemsLeftToCollect > 0 && len(members) > 0 {
		purgedPvt := p.getPurgedCollections(members, dig2Filter)
//需要从映射中删除清除的摘要
		for _, dig := range purgedPvt {
			res.PurgedElements = append(res.PurgedElements, &proto.PvtDataDigest{
				TxId:       dig.TxId,
				BlockSeq:   dig.BlockSeq,
				SeqInBlock: dig.SeqInBlock,
				Namespace:  dig.Namespace,
				Collection: dig.Collection,
			})
//删除摘要，这样我们甚至不会尝试提取清除的数据
			delete(dig2Filter, dig)
			itemsLeftToCollect--
		}

		if itemsLeftToCollect == 0 {
			logger.Debug("No items left to collect")
			return res, nil
		}

		peer2digests, members = p.assignDigestsToPeers(members, dig2Filter)
		if len(peer2digests) == 0 {
			logger.Warning("No available peers for digests request, "+
				"cannot pull missing private data for following digests [%+v], peer membership: [%+v]",
				dig2Filter.digests(), members)
			return res, nil
		}

		logger.Debug("Matched", len(dig2Filter), "digests to", len(peer2digests), "peer(s)")
		subscriptions := p.scatterRequests(peer2digests)
		responses := p.gatherResponses(subscriptions)
		for _, resp := range responses {
			if len(resp.Payload) == 0 {
				logger.Debug("Got empty response for", resp.Digest)
				continue
			}
			delete(dig2Filter, privdatacommon.DigKey{
				TxId:       resp.Digest.TxId,
				BlockSeq:   resp.Digest.BlockSeq,
				SeqInBlock: resp.Digest.SeqInBlock,
				Namespace:  resp.Digest.Namespace,
				Collection: resp.Digest.Collection,
			})
			itemsLeftToCollect--
		}
		res.AvailableElements = append(res.AvailableElements, responses...)
	}
	return res, nil
}

func (p *puller) gatherResponses(subscriptions []util.Subscription) []*proto.PvtDataElement {
	var res []*proto.PvtDataElement
	privateElements := make(chan *proto.PvtDataElement, len(subscriptions))
	var wg sync.WaitGroup
	wg.Add(len(subscriptions))
//侦听所有订阅，然后将其添加到单个通道中
	for _, sub := range subscriptions {
		go func(sub util.Subscription) {
			defer wg.Done()
			el, err := sub.Listen()
			if err != nil {
				return
			}
			privateElements <- el.(*proto.PvtDataElement)
		}(sub)
	}
//等待所有订阅返回或超时
	wg.Wait()
//关闭通道，在迭代时不要阻塞它。
	close(privateElements)
//聚合元素以将其作为切片返回
	for el := range privateElements {
		res = append(res, el)
	}
	return res
}

func (p *puller) scatterRequests(peersDigestMapping peer2Digests) []util.Subscription {
	var subscriptions []util.Subscription
	for peer, digests := range peersDigestMapping {
		msg := &proto.GossipMessage{
			Tag:     proto.GossipMessage_CHAN_ONLY,
			Channel: []byte(p.channel),
			Nonce:   util.RandomUInt64(),
			Content: &proto.GossipMessage_PrivateReq{
				PrivateReq: &proto.RemotePvtDataRequest{
					Digests: digestsAsPointerSlice(digests),
				},
			},
		}

//发送前订阅所有摘要
		for _, dig := range msg.GetPrivateReq().Digests {
			hash, err := dig.Hash()
			if err != nil {
//不应该发生在我们自己建立这个信息的时候
				logger.Warning("Failed creating digest", err)
				continue
			}
			sub := p.pubSub.Subscribe(hash, responseWaitTime)
			subscriptions = append(subscriptions, sub)
		}
		logger.Debug("Sending", peer.endpoint, "request", msg.GetPrivateReq().Digests)
		p.Send(msg, peer.AsRemotePeer())
	}
	return subscriptions
}

type peer2Digests map[remotePeer][]proto.PvtDataDigest
type noneSelectedPeers []discovery.NetworkMember

func (p *puller) assignDigestsToPeers(members []discovery.NetworkMember, dig2Filter digestToFilterMapping) (peer2Digests, noneSelectedPeers) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debug("Matching", members, "to", dig2Filter.String())
	}
	res := make(map[remotePeer][]proto.PvtDataDigest)
//创建对等点和摘要之间的映射以请求
	for dig, collectionFilter := range dig2Filter {
//查找首选对等机
		selectedPeer := filter.First(members, collectionFilter.preferredPeer)
		if selectedPeer == nil {
			logger.Debug("No preferred peer found for", dig)
//找到集合中的某个对等点
			selectedPeer = filter.First(members, collectionFilter.anyPeer)
		}
		if selectedPeer == nil {
			logger.Debug("No peer matches txID", dig.TxId, "collection", dig.Collection)
			continue
		}
//将对等机添加到从对等机到摘要切片的映射
		peer := remotePeer{pkiID: string(selectedPeer.PKIID), endpoint: selectedPeer.Endpoint}
		res[peer] = append(res[peer], proto.PvtDataDigest{
			TxId:       dig.TxId,
			BlockSeq:   dig.BlockSeq,
			SeqInBlock: dig.SeqInBlock,
			Namespace:  dig.Namespace,
			Collection: dig.Collection,
		})
	}

	var noneSelectedPeers []discovery.NetworkMember
	for _, member := range members {
		peer := remotePeer{endpoint: member.PreferredEndpoint(), pkiID: string(member.PKIid)}
		if _, selected := res[peer]; !selected {
			noneSelectedPeers = append(noneSelectedPeers, member)
		}
	}

	return res, noneSelectedPeers
}

type collectionRoutingFilter struct {
	anyPeer       filter.RoutingFilter
	preferredPeer filter.RoutingFilter
}

type digestToFilterMapping map[privdatacommon.DigKey]collectionRoutingFilter

func (dig2f digestToFilterMapping) flattenFilterValues() []filter.RoutingFilter {
	var filters []filter.RoutingFilter
	for _, f := range dig2f {
		filters = append(filters, f.preferredPeer)
		filters = append(filters, f.anyPeer)
	}
	return filters
}

func (dig2f digestToFilterMapping) digests() []proto.PvtDataDigest {
	var digs []proto.PvtDataDigest
	for d := range dig2f {
		digs = append(digs, proto.PvtDataDigest{
			TxId:       d.TxId,
			BlockSeq:   d.BlockSeq,
			SeqInBlock: d.SeqInBlock,
			Namespace:  d.Namespace,
			Collection: d.Collection,
		})
	}
	return digs
}

//字符串返回DigestToFilterMapping的字符串表示形式
func (dig2f digestToFilterMapping) String() string {
	var buffer bytes.Buffer
	collection2TxID := make(map[string][]string)
	for dig := range dig2f {
		collection2TxID[dig.Collection] = append(collection2TxID[dig.Collection], dig.TxId)
	}
	for col, txIDs := range collection2TxID {
		buffer.WriteString(fmt.Sprintf("{%s: %v}", col, txIDs))
	}
	return buffer.String()
}

func (p *puller) computeFilters(dig2src dig2sources) (digestToFilterMapping, error) {
	filters := make(map[privdatacommon.DigKey]collectionRoutingFilter)
	for digest, sources := range dig2src {
		anyPeerInCollection, err := p.getLatestCollectionConfigRoutingFilter(digest.Namespace, digest.Collection)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		sources := sources
		endorserPeer, err := p.PeerFilter(common.ChainID(p.channel), func(peerSignature api.PeerSignature) bool {
			for _, endorsement := range sources {
				if bytes.Equal(endorsement.Endorser, []byte(peerSignature.PeerIdentity)) {
					return true
				}
			}
			return false
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		filters[digest] = collectionRoutingFilter{
			anyPeer:       anyPeerInCollection,
			preferredPeer: endorserPeer,
		}
	}
	return filters, nil
}

func (p *puller) computeReconciliationFilters(dig2collectionConfig privdatacommon.Dig2CollectionConfig) (digestToFilterMapping, error) {
	filters := make(map[privdatacommon.DigKey]collectionRoutingFilter)
	for digest, originalCollectionConfig := range dig2collectionConfig {
		anyPeerInCollection, err := p.getLatestCollectionConfigRoutingFilter(digest.Namespace, digest.Collection)
		if err != nil {
			return nil, err
		}

		originalConfigFilter, err := p.cs.AccessFilter(p.channel, originalCollectionConfig.MemberOrgsPolicy)
		if err != nil {
			return nil, err
		}
		if originalConfigFilter == nil {
			return nil, errors.Errorf("Failed obtaining original collection filter for channel %s, config %s", p.channel, digest.Collection)
		}

//获取创建丢失数据时集合配置中的对等方
		peerFromDataCreation, err := p.getMatchAllRoutingFilter(originalConfigFilter)
		if err != nil {
			return nil, err
		}

//更喜欢从数据创建时就在集合中的对等方，而不是稍后添加的对等方。
//假设对等机在集合配置中的时间越长，它拥有数据的机会就越大。
		preferredPeer := func(member discovery.NetworkMember) bool {
			return peerFromDataCreation(member) && anyPeerInCollection(member)
		}

		filters[digest] = collectionRoutingFilter{
			anyPeer:       anyPeerInCollection,
			preferredPeer: preferredPeer,
		}
	}
	return filters, nil
}

func (p *puller) getLatestCollectionConfigRoutingFilter(chaincode string, collection string) (filter.RoutingFilter, error) {
	cc := fcommon.CollectionCriteria{
		Channel:    p.channel,
		Collection: collection,
		Namespace:  chaincode,
	}

	latestCollectionConfig, err := p.cs.RetrieveCollectionAccessPolicy(cc)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed obtaining collection policy for channel %s, chaincode %s, config %s", p.channel, chaincode, collection))
	}

	filt := latestCollectionConfig.AccessFilter()
	if filt == nil {
		return nil, errors.Errorf("Failed obtaining collection filter for channel %s, chaincode %s, collection %s", p.channel, chaincode, collection)
	}

	anyPeerInCollection, err := p.getMatchAllRoutingFilter(filt)
	if err != nil {
		return nil, err
	}

	return anyPeerInCollection, nil
}

func (p *puller) getMatchAllRoutingFilter(filt privdata.Filter) (filter.RoutingFilter, error) {
	routingFilter, err := p.PeerFilter(common.ChainID(p.channel), func(peerSignature api.PeerSignature) bool {
		return filt(fcommon.SignedData{
			Signature: peerSignature.Signature,
			Identity:  peerSignature.PeerIdentity,
			Data:      peerSignature.Message,
		})
	})
	return routingFilter, err
}

func (p *puller) getPurgedCollections(members []discovery.NetworkMember, dig2Filter digestToFilterMapping) []privdatacommon.DigKey {
	var res []privdatacommon.DigKey
	for dig := range dig2Filter {
		purged, err := p.purgedFilter(dig)
		if err != nil {
			logger.Debug("Failed to obtain purged filter for digest %v", dig, "error", err)
			continue
		}

		membersWithPurgedData := filter.AnyMatch(members, purged)
//至少一个对等机已清除数据
		if len(membersWithPurgedData) > 0 {
			logger.Debugf("Private data on channel [%s], chaincode [%s], collection name [%s] for txID = [%s],"+
				"has been purged at peers [%v]", p.channel, dig.Namespace,
				dig.Collection, dig.TxId, membersWithPurgedData)
			res = append(res, dig)
		}
	}
	return res
}

func (p *puller) purgedFilter(dig privdatacommon.DigKey) (filter.RoutingFilter, error) {
	cc := fcommon.CollectionCriteria{
		Channel:    p.channel,
		TxId:       dig.TxId,
		Collection: dig.Collection,
		Namespace:  dig.Namespace,
	}
	colPersistConfig, err := p.cs.RetrieveCollectionPersistenceConfigs(cc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return func(peer discovery.NetworkMember) bool {
		if peer.Properties == nil {
			logger.Debugf("No properties provided for peer %s", peer.Endpoint)
			return false
		}
//btl等于零，语义为永不过期
		if colPersistConfig.BlockToLive() == uint64(0) {
			return false
		}
//句柄溢出
		expirationSeqNum := addWithOverflow(dig.BlockSeq, colPersistConfig.BlockToLive())
		peerLedgerHeightWithMargin := addWithOverflow(peer.Properties.LedgerHeight, p.btlPullMargin)

		isPurged := peerLedgerHeightWithMargin >= expirationSeqNum
		if isPurged {
			logger.Debugf("skipping peer [%s], since pvt for channel [%s], txID = [%s], "+
				"collection [%s] has been purged or will soon be purged, BTL=[%d]",
				peer.Endpoint, p.channel, cc.TxId, cc.Collection, colPersistConfig.BlockToLive())
		}
		return isPurged
	}, nil
}

func (p *puller) filterNotEligible(dig2rwSets Dig2PvtRWSetWithConfig, shouldCheckLatestConfig bool, signedData fcommon.SignedData, endpoint string) []*proto.PvtDataElement {
	var returned []*proto.PvtDataElement
	for d, rwSets := range dig2rwSets {
		if rwSets == nil {
			logger.Errorf("No private rwset for [%s] channel, chaincode [%s], collection [%s], txID = [%s] is available, skipping...",
				p.channel, d.Namespace, d.Collection, d.TxId)
			continue
		}
		logger.Debug("Found", len(rwSets.RWSet), "for TxID", d.TxId, ", collection", d.Collection, "for", endpoint)
		if len(rwSets.RWSet) == 0 {
			continue
		}

		eligibleForCollection := shouldCheckLatestConfig && p.isEligibleByLatestConfig(p.channel, d.Collection, d.Namespace, signedData)

		if !eligibleForCollection {
			colAP, err := p.AccessPolicy(rwSets.CollectionConfig, p.channel)
			if err != nil {
				logger.Debug("No policy found for channel", p.channel, ", collection", d.Collection, "txID", d.TxId, ":", err, "skipping...")
				continue
			}
			colFilter := colAP.AccessFilter()
			if colFilter == nil {
				logger.Debug("Collection ", d.Collection, " has no access filter, txID", d.TxId, "skipping...")
				continue
			}
			eligibleForCollection = colFilter(signedData)
		}

		if !eligibleForCollection {
			logger.Debug("Peer", endpoint, "isn't eligible for txID", d.TxId, "at collection", d.Collection)
			continue
		}

		returned = append(returned, &proto.PvtDataElement{
			Digest: &proto.PvtDataDigest{
				TxId:       d.TxId,
				BlockSeq:   d.BlockSeq,
				Collection: d.Collection,
				Namespace:  d.Namespace,
				SeqInBlock: d.SeqInBlock,
			},
			Payload: util.PrivateRWSets(rwSets.RWSet...),
		})
	}
	return returned
}

func (p *puller) isEligibleByLatestConfig(channel string, collection string, chaincode string, signedData fcommon.SignedData) bool {
	cc := fcommon.CollectionCriteria{
		Channel:    channel,
		Collection: collection,
		Namespace:  chaincode,
	}

	latestCollectionConfig, err := p.cs.RetrieveCollectionAccessPolicy(cc)
	if err != nil {
		return false
	}

	collectionFilter := latestCollectionConfig.AccessFilter()
	return collectionFilter(signedData)
}

func randomizeMemberList(members []discovery.NetworkMember) []discovery.NetworkMember {
	rand.Seed(time.Now().UnixNano())
	res := make([]discovery.NetworkMember, len(members))
	for i, j := range rand.Perm(len(members)) {
		res[i] = members[j]
	}
	return res
}

func digestsAsPointerSlice(digests []proto.PvtDataDigest) []*proto.PvtDataDigest {
	res := make([]*proto.PvtDataDigest, len(digests))
	for i, dig := range digests {
//重新引入dig变量进行分配
//每次迭代的新地址
		dig := dig
		res[i] = &dig
	}
	return res
}

type remotePeer struct {
	endpoint string
	pkiID    string
}

//asremotepeer将此远程对等转换为comm.remotepeer
func (rp remotePeer) AsRemotePeer() *comm.RemotePeer {
	return &comm.RemotePeer{
		PKIID:    common.PKIidType(rp.pkiID),
		Endpoint: rp.endpoint,
	}
}

func getBtlPullMargin() uint64 {
	var result uint64
	if viper.IsSet("peer.gossip.pvtData.btlPullMargin") {
		btlMarginVal := viper.GetInt("peer.gossip.pvtData.btlPullMargin")
		if btlMarginVal < 0 {
			result = btlPullMarginDefault
		} else {
			result = uint64(btlMarginVal)
		}
	} else {
		result = btlPullMarginDefault
	}
	return result
}

func addWithOverflow(a uint64, b uint64) uint64 {
	res := a + b
	if res < a {
		return math.MaxUint64
	}
	return res
}
