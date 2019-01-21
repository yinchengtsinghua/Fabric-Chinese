
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
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

var (
	configTypes = []discovery.QueryType{discovery.ConfigQueryType, discovery.PeerMembershipQueryType, discovery.ChaincodeQueryType, discovery.LocalMembershipQueryType}
)

//客户端与发现服务器交互
type Client struct {
	createConnection Dialer
	signRequest      Signer
}

//新建请求创建新请求
func NewRequest() *Request {
	r := &Request{
		invocationChainMapping: make(map[int][]InvocationChain),
		queryMapping:           make(map[discovery.QueryType]map[string]int),
		Request:                &discovery.Request{},
	}
//预填充类型
	for _, queryType := range configTypes {
		r.queryMapping[queryType] = make(map[string]int)
	}
	return r
}

//请求在其内部聚合多个查询
type Request struct {
	lastChannel string
	lastIndex   int
//从查询类型到通道的映射到响应中的预期索引
	queryMapping map[discovery.QueryType]map[string]int
//响应调用链的期望索引映射
	invocationChainMapping map[int][]InvocationChain
	*discovery.Request
}

//addconfigquery向请求添加config查询
func (req *Request) AddConfigQuery() *Request {
	ch := req.lastChannel
	q := &discovery.Query_ConfigQuery{
		ConfigQuery: &discovery.ConfigQuery{},
	}
	req.Queries = append(req.Queries, &discovery.Query{
		Channel: ch,
		Query:   q,
	})
	req.addQueryMapping(discovery.ConfigQueryType, ch)
	return req
}

//AddEndorsersQuery adds to the request a query for given chaincodes
//兴趣是客户想要查询的链码兴趣。
//给定通道的所有兴趣都应在聚合切片中提供
func (req *Request) AddEndorsersQuery(interests ...*discovery.ChaincodeInterest) (*Request, error) {
	if err := validateInterests(interests...); err != nil {
		return nil, err
	}
	ch := req.lastChannel
	q := &discovery.Query_CcQuery{
		CcQuery: &discovery.ChaincodeQuery{
			Interests: interests,
		},
	}
	req.Queries = append(req.Queries, &discovery.Query{
		Channel: ch,
		Query:   q,
	})
	var invocationChains []InvocationChain
	for _, interest := range interests {
		invocationChains = append(invocationChains, interest.Chaincodes)
	}
	req.addChaincodeQueryMapping(invocationChains)
	req.addQueryMapping(discovery.ChaincodeQueryType, ch)
	return req, nil
}

//AddLocalPeersQuery adds to the request a local peer query
func (req *Request) AddLocalPeersQuery() *Request {
	q := &discovery.Query_LocalPeers{
		LocalPeers: &discovery.LocalPeerQuery{},
	}
	req.Queries = append(req.Queries, &discovery.Query{
		Query: q,
	})
	var ic InvocationChain
	req.addQueryMapping(discovery.LocalMembershipQueryType, channnelAndInvocationChain("", ic))
	return req
}

//addpeersquery向请求添加对等查询
func (req *Request) AddPeersQuery(invocationChain ...*discovery.ChaincodeCall) *Request {
	ch := req.lastChannel
	q := &discovery.Query_PeerQuery{
		PeerQuery: &discovery.PeerMembershipQuery{
			Filter: &discovery.ChaincodeInterest{
				Chaincodes: invocationChain,
			},
		},
	}
	req.Queries = append(req.Queries, &discovery.Query{
		Channel: ch,
		Query:   q,
	})
	var ic InvocationChain
	if len(invocationChain) > 0 {
		ic = InvocationChain(invocationChain)
	}
	req.addChaincodeQueryMapping([]InvocationChain{ic})
	req.addQueryMapping(discovery.PeerMembershipQueryType, channnelAndInvocationChain(ch, ic))
	return req
}

func channnelAndInvocationChain(ch string, ic InvocationChain) string {
	return fmt.Sprintf("%s %s", ch, ic.String())
}

//ofchannel设置添加到给定通道上下文中的下一个查询
func (req *Request) OfChannel(ch string) *Request {
	req.lastChannel = ch
	return req
}

func (req *Request) addChaincodeQueryMapping(invocationChains []InvocationChain) {
	req.invocationChainMapping[req.lastIndex] = invocationChains
}

func (req *Request) addQueryMapping(queryType discovery.QueryType, key string) {
	req.queryMapping[queryType][key] = req.lastIndex
	req.lastIndex++
}

//发送发送请求并返回响应，或者失败时出错
func (c *Client) Send(ctx context.Context, req *Request, auth *discovery.AuthInfo) (Response, error) {
	reqToBeSent := *req.Request
	reqToBeSent.Authentication = auth
	payload, err := proto.Marshal(&reqToBeSent)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshaling Request to bytes")
	}

	sig, err := c.signRequest(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed signing Request")
	}

	conn, err := c.createConnection()
	if err != nil {
		return nil, errors.Wrap(err, "failed connecting to discovery service")
	}

	cl := discovery.NewDiscoveryClient(conn)
	resp, err := cl.Discover(ctx, &discovery.SignedRequest{
		Payload:   payload,
		Signature: sig,
	})
	if err != nil {
		return nil, errors.Wrap(err, "discovery service refused our Request")
	}
	if n := len(resp.Results); n != req.lastIndex {
		return nil, errors.Errorf("Sent %d queries but received %d responses back", req.lastIndex, n)
	}
	return req.computeResponse(resp)
}

type resultOrError interface {
}

type response map[key]resultOrError

type localResponse struct {
	response
}

func (cr *localResponse) Peers() ([]*Peer, error) {
	return parsePeers(discovery.LocalMembershipQueryType, cr.response, "")
}

type channelResponse struct {
	response
	channel string
}

func (cr *channelResponse) Config() (*discovery.ConfigResult, error) {
	res, exists := cr.response[key{
		queryType: discovery.ConfigQueryType,
		k:         cr.channel,
	}]

	if !exists {
		return nil, ErrNotFound
	}

	if config, isConfig := res.(*discovery.ConfigResult); isConfig {
		return config, nil
	}

	return nil, res.(error)
}

func parsePeers(queryType discovery.QueryType, r response, channel string, invocationChain ...*discovery.ChaincodeCall) ([]*Peer, error) {
	peerKeys := key{
		queryType: queryType,
		k:         fmt.Sprintf("%s %s", channel, InvocationChain(invocationChain).String()),
	}
	res, exists := r[peerKeys]

	if !exists {
		return nil, ErrNotFound
	}

	if peers, isPeers := res.([]*Peer); isPeers {
		return peers, nil
	}

	return nil, res.(error)
}

func (cr *channelResponse) Peers(invocationChain ...*discovery.ChaincodeCall) ([]*Peer, error) {
	return parsePeers(discovery.PeerMembershipQueryType, cr.response, cr.channel, invocationChain...)
}

func (cr *channelResponse) Endorsers(invocationChain InvocationChain, f Filter) (Endorsers, error) {
//如果我们有一个没有链码字段的键，
//这意味着这是服务返回的错误
	if err, exists := cr.response[key{
		queryType: discovery.ChaincodeQueryType,
		k:         cr.channel,
	}]; exists {
		return nil, err.(error)
	}

//否则，服务返回的响应不是错误
	res, exists := cr.response[key{
		queryType:       discovery.ChaincodeQueryType,
		k:               cr.channel,
		invocationChain: invocationChain.String(),
	}]

	if !exists {
		return nil, ErrNotFound
	}

	desc := res.(*endorsementDescriptor)
	rand.Seed(time.Now().Unix())
//我们迭代所有的布局，找到一个我们有足够的对等选择的布局。
	for _, index := range rand.Perm(len(desc.layouts)) {
		layout := desc.layouts[index]
		endorsers, canLayoutBeSatisfied := selectPeersForLayout(desc.endorsersByGroups, layout, f)
		if canLayoutBeSatisfied {
			return endorsers, nil
		}
	}
	return nil, errors.New("no endorsement combination can be satisfied")
}

type filter struct {
	ef ExclusionFilter
	ps PrioritySelector
}

//newfilter返回使用给定排除筛选器和优先级选择器的认可器筛选器
//对背书人进行筛选和排序
func NewFilter(ps PrioritySelector, ef ExclusionFilter) Filter {
	return &filter{
		ef: ef,
		ps: ps,
	}
}

//filter返回已筛选和排序的背书人列表
func (f *filter) Filter(endorsers Endorsers) Endorsers {
	return endorsers.Shuffle().Filter(f.ef).Sort(f.ps)
}

//nofilter返回noop筛选器
var NoFilter = NewFilter(NoPriorities, NoExclusion)

func selectPeersForLayout(endorsersByGroups map[string][]*Peer, layout map[string]int, f Filter) (Endorsers, bool) {
	var endorsers []*Peer
	for grp, count := range layout {
		endorsersOfGrp := f.Filter(Endorsers(endorsersByGroups[grp]))

//无法为此布局选择足够的对等方，因为当前组
//需要比我们可以选择的对等点更多
		if len(endorsersOfGrp) < count {
			return nil, false
		}
		endorsersOfGrp = endorsersOfGrp[:count]
		endorsers = append(endorsers, endorsersOfGrp...)
	}
//可以满足当前（随机选择的）布局，因此返回它
//而不是检查下一个。
	return endorsers, true
}

func (resp response) ForLocal() LocalResponse {
	return &localResponse{
		response: resp,
	}
}

func (resp response) ForChannel(ch string) ChannelResponse {
	return &channelResponse{
		channel:  ch,
		response: resp,
	}
}

type key struct {
	queryType       discovery.QueryType
	k               string
	invocationChain string
}

func (req *Request) computeResponse(r *discovery.Response) (response, error) {
	var err error
	resp := make(response)
	for configType, channel2index := range req.queryMapping {
		switch configType {
		case discovery.ConfigQueryType:
			err = resp.mapConfig(channel2index, r)
		case discovery.ChaincodeQueryType:
			err = resp.mapEndorsers(channel2index, r, req.invocationChainMapping)
		case discovery.PeerMembershipQueryType:
			err = resp.mapPeerMembership(channel2index, r, discovery.PeerMembershipQueryType)
		case discovery.LocalMembershipQueryType:
			err = resp.mapPeerMembership(channel2index, r, discovery.LocalMembershipQueryType)
		}
		if err != nil {
			return nil, err
		}
	}

	return resp, err
}

func (resp response) mapConfig(channel2index map[string]int, r *discovery.Response) error {
	for ch, index := range channel2index {
		config, err := r.ConfigAt(index)
		if config == nil && err == nil {
			return errors.Errorf("expected QueryResult of either ConfigResult or Error but got %v instead", r.Results[index])
		}
		key := key{
			queryType: discovery.ConfigQueryType,
			k:         ch,
		}

		if err != nil {
			resp[key] = errors.New(err.Content)
			continue
		}

		resp[key] = config
	}
	return nil
}

func (resp response) mapPeerMembership(key2Index map[string]int, r *discovery.Response, qt discovery.QueryType) error {
	for k, index := range key2Index {
		membersRes, err := r.MembershipAt(index)
		if membersRes == nil && err == nil {
			return errors.Errorf("expected QueryResult of either PeerMembershipResult or Error but got %v instead", r.Results[index])
		}

		key := key{
			queryType: qt,
			k:         k,
		}

		if err != nil {
			resp[key] = errors.New(err.Content)
			continue
		}

		peers, err2 := peersForChannel(membersRes, qt)
		if err2 != nil {
			return errors.Wrap(err2, "failed constructing peer membership out of response")
		}

		resp[key] = peers
	}
	return nil
}

func peersForChannel(membersRes *discovery.PeerMembershipResult, qt discovery.QueryType) ([]*Peer, error) {
	var peers []*Peer
	for org, peersOfCurrentOrg := range membersRes.PeersByOrg {
		for _, peer := range peersOfCurrentOrg.Peers {
			aliveMsg, err := peer.MembershipInfo.ToGossipMessage()
			if err != nil {
				return nil, errors.Wrap(err, "failed unmarshaling alive message")
			}
			var stateInfoMsg *gossip.SignedGossipMessage
			if isStateInfoExpected(qt) {
				stateInfoMsg, err = peer.StateInfo.ToGossipMessage()
				if err != nil {
					return nil, errors.Wrap(err, "failed unmarshaling stateInfo message")
				}
				if err := validateStateInfoMessage(stateInfoMsg); err != nil {
					return nil, errors.Wrap(err, "failed validating stateInfo message")
				}
			}
			if err := validateAliveMessage(aliveMsg); err != nil {
				return nil, errors.Wrap(err, "failed validating alive message")
			}
			peers = append(peers, &Peer{
				MSPID:            org,
				Identity:         peer.Identity,
				AliveMessage:     aliveMsg,
				StateInfoMessage: stateInfoMsg,
			})
		}
	}
	return peers, nil
}

func isStateInfoExpected(qt discovery.QueryType) bool {
	return qt != discovery.LocalMembershipQueryType
}

func (resp response) mapEndorsers(
	channel2index map[string]int,
	r *discovery.Response,
	chaincodeQueryMapping map[int][]InvocationChain) error {
	for ch, index := range channel2index {
		ccQueryRes, err := r.EndorsersAt(index)
		if ccQueryRes == nil && err == nil {
			return errors.Errorf("expected QueryResult of either ChaincodeQueryResult or Error but got %v instead", r.Results[index])
		}

		if err != nil {
			key := key{
				queryType: discovery.ChaincodeQueryType,
				k:         ch,
			}
			resp[key] = errors.New(err.Content)
			continue
		}

		if err := resp.mapEndorsersOfChannel(ccQueryRes, ch, chaincodeQueryMapping[index]); err != nil {
			return errors.Wrapf(err, "failed assembling endorsers of channel %s", ch)
		}
	}
	return nil
}

func (resp response) mapEndorsersOfChannel(ccRs *discovery.ChaincodeQueryResult, channel string, invocationChain []InvocationChain) error {
	if len(ccRs.Content) < len(invocationChain) {
		return errors.Errorf("expected %d endorsement descriptors but got only %d", len(invocationChain), len(ccRs.Content))
	}
	for i, desc := range ccRs.Content {
		expectedCCName := invocationChain[i][0].Name
		if desc.Chaincode != expectedCCName {
			return errors.Errorf("expected chaincode %s but got endorsement descriptor for %s", expectedCCName, desc.Chaincode)
		}
		key := key{
			queryType:       discovery.ChaincodeQueryType,
			k:               channel,
			invocationChain: invocationChain[i].String(),
		}

		descriptor, err := resp.createEndorsementDescriptor(desc, channel)
		if err != nil {
			return err
		}
		resp[key] = descriptor
	}

	return nil
}

func (resp response) createEndorsementDescriptor(desc *discovery.EndorsementDescriptor, channel string) (*endorsementDescriptor, error) {
	descriptor := &endorsementDescriptor{
		layouts:           []map[string]int{},
		endorsersByGroups: make(map[string][]*Peer),
	}
	for _, l := range desc.Layouts {
		currentLayout := make(map[string]int)
		descriptor.layouts = append(descriptor.layouts, currentLayout)
		for grp, count := range l.QuantitiesByGroup {
			if _, exists := desc.EndorsersByGroups[grp]; !exists {
				return nil, errors.Errorf("group %s isn't mapped to endorsers, but exists in a layout", grp)
			}
			currentLayout[grp] = int(count)
		}
	}

	for grp, peers := range desc.EndorsersByGroups {
		var endorsers []*Peer
		for _, p := range peers.Peers {
			peer, err := endorser(p, desc.Chaincode, channel)
			if err != nil {
				return nil, errors.Wrap(err, "failed creating endorser object")
			}
			endorsers = append(endorsers, peer)
		}
		descriptor.endorsersByGroups[grp] = endorsers
	}

	return descriptor, nil
}

func endorser(peer *discovery.Peer, chaincode, channel string) (*Peer, error) {
	if peer.MembershipInfo == nil || peer.StateInfo == nil {
		return nil, errors.Errorf("received empty envelope(s) for endorsers for chaincode %s, channel %s", chaincode, channel)
	}
	aliveMsg, err := peer.MembershipInfo.ToGossipMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling gossip envelope to alive message")
	}
	stateInfMsg, err := peer.StateInfo.ToGossipMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling gossip envelope to state info message")
	}
	if err := validateAliveMessage(aliveMsg); err != nil {
		return nil, errors.Wrap(err, "failed validating alive message")
	}
	if err := validateStateInfoMessage(stateInfMsg); err != nil {
		return nil, errors.Wrap(err, "failed validating stateInfo message")
	}
	sID := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(peer.Identity, sID); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling peer's identity")
	}
	return &Peer{
		Identity:         peer.Identity,
		StateInfoMessage: stateInfMsg,
		AliveMessage:     aliveMsg,
		MSPID:            sID.Mspid,
	}, nil
}

type endorsementDescriptor struct {
	endorsersByGroups map[string][]*Peer
	layouts           []map[string]int
}

//new client创建新的客户端实例
func NewClient(createConnection Dialer, s Signer, signerCacheSize uint) *Client {
	return &Client{
		createConnection: createConnection,
		signRequest:      NewMemoizeSigner(s, signerCacheSize).Sign,
	}
}

func validateAliveMessage(message *gossip.SignedGossipMessage) error {
	am := message.GetAliveMsg()
	if am == nil {
		return errors.New("message isn't an alive message")
	}
	m := am.Membership
	if m == nil {
		return errors.New("membership is empty")
	}
	if am.Timestamp == nil {
		return errors.New("timestamp is nil")
	}
	return nil
}

func validateStateInfoMessage(message *gossip.SignedGossipMessage) error {
	si := message.GetStateInfo()
	if si == nil {
		return errors.New("message isn't a stateInfo message")
	}
	if si.Timestamp == nil {
		return errors.New("timestamp is nil")
	}
	if si.Properties == nil {
		return errors.New("properties is nil")
	}
	return nil
}

func validateInterests(interests ...*discovery.ChaincodeInterest) error {
	if len(interests) == 0 {
		return errors.New("no chaincode interests given")
	}
	for _, interest := range interests {
		if interest == nil {
			return errors.New("chaincode interest is nil")
		}
		if err := InvocationChain(interest.Chaincodes).ValidateInvocationChain(); err != nil {
			return err
		}
	}
	return nil
}

//调用链聚合链编解码器调用
type InvocationChain []*discovery.ChaincodeCall

//字符串返回此调用链的字符串表示形式
func (ic InvocationChain) String() string {
	s, _ := json.Marshal(ic)
	return string(s)
}

//validateInvocationChain验证InvocationChain的结构
func (ic InvocationChain) ValidateInvocationChain() error {
	if len(ic) == 0 {
		return errors.New("invocation chain should not be empty")
	}
	for _, cc := range ic {
		if cc.Name == "" {
			return errors.New("chaincode name should not be empty")
		}
	}
	return nil
}
