
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
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	common2 "github.com/hyperledger/fabric/gossip/common"
	discovery2 "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/discovery"
	"github.com/pkg/errors"
)

var (
	logger = flogging.MustGetLogger("discovery")
)

var accessDenied = wrapError(errors.New("access denied"))

//certhashextractor从给定上下文中提取TLS证书
//并返回其哈希值
type certHashExtractor func(ctx context.Context) []byte

//Dispatcher定义了一个用于发送查询的函数
type dispatcher func(q *discovery.Query) *discovery.QueryResult

type service struct {
	config             Config
	channelDispatchers map[discovery.QueryType]dispatcher
	localDispatchers   map[discovery.QueryType]dispatcher
	auth               *authCache
	Support
}

//config定义发现服务的配置
type Config struct {
	TLS                          bool
	AuthCacheEnabled             bool
	AuthCacheMaxSize             int
	AuthCachePurgeRetentionRatio float64
}

//string返回此配置的字符串表示形式
func (c Config) String() string {
	if c.AuthCacheEnabled {
		return fmt.Sprintf("TLS: %t, authCacheMaxSize: %d, authCachePurgeRatio: %f", c.TLS, c.AuthCacheMaxSize, c.AuthCachePurgeRetentionRatio)
	}
	return fmt.Sprintf("TLS: %t, auth cache disabled", c.TLS)
}

//Petri映射将PKI ID映射到对等点
type peerMapping map[string]*discovery.Peer

//NewService创建新的发现服务实例
func NewService(config Config, sup Support) *service {
	s := &service{
		auth: newAuthCache(sup, authCacheConfig{
			enabled:             config.AuthCacheEnabled,
			maxCacheSize:        config.AuthCacheMaxSize,
			purgeRetentionRatio: config.AuthCachePurgeRetentionRatio,
		}),
		Support: sup,
	}
	s.channelDispatchers = map[discovery.QueryType]dispatcher{
		discovery.ConfigQueryType:         s.configQuery,
		discovery.ChaincodeQueryType:      s.chaincodeQuery,
		discovery.PeerMembershipQueryType: s.channelMembershipResponse,
	}
	s.localDispatchers = map[discovery.QueryType]dispatcher{
		discovery.LocalMembershipQueryType: s.localMembershipResponse,
	}
	logger.Info("Created with config", config)
	return s
}

func (s *service) Discover(ctx context.Context, request *discovery.SignedRequest) (*discovery.Response, error) {
	addr := util.ExtractRemoteAddress(ctx)
	req, err := validateStructure(ctx, request, s.config.TLS, comm.ExtractCertificateHashFromContext)
	if err != nil {
		logger.Warningf("Request from %s is malformed or invalid: %v", addr, err)
		return nil, err
	}
	logger.Debugf("Processing request from %s: %v", addr, req)
	var res []*discovery.QueryResult
	for _, q := range req.Queries {
		res = append(res, s.processQuery(q, request, req.Authentication.ClientIdentity, addr))
	}
	logger.Debugf("Returning to %s a response containing: %v", addr, res)
	return &discovery.Response{
		Results: res,
	}, nil
}

func (s *service) processQuery(query *discovery.Query, request *discovery.SignedRequest, identity []byte, addr string) *discovery.QueryResult {
	if query.Channel != "" && !s.ChannelExists(query.Channel) {
		logger.Warning("got query for channel", query.Channel, "from", addr, "but it doesn't exist")
		return accessDenied
	}
	if err := s.auth.EligibleForService(query.Channel, common.SignedData{
		Data:      request.Payload,
		Signature: request.Signature,
		Identity:  identity,
	}); err != nil {
		logger.Warning("got query for channel", query.Channel, "from", addr, "but it isn't eligible:", err)
		return accessDenied
	}
	return s.dispatch(query)
}

func (s *service) dispatch(q *discovery.Query) *discovery.QueryResult {
	dispatchers := s.channelDispatchers
//确保本地查询仅路由到无通道调度程序
	if q.Channel == "" {
		dispatchers = s.localDispatchers
	}
	dispatchQuery, exists := dispatchers[q.GetType()]
	if !exists {
		return wrapError(errors.New("unknown or missing request type"))
	}
	return dispatchQuery(q)
}

func (s *service) chaincodeQuery(q *discovery.Query) *discovery.QueryResult {
	if err := validateCCQuery(q.GetCcQuery()); err != nil {
		return wrapError(err)
	}
	var descriptors []*discovery.EndorsementDescriptor
	for _, interest := range q.GetCcQuery().Interests {
		desc, err := s.PeersForEndorsement(common2.ChainID(q.Channel), interest)
		if err != nil {
			logger.Errorf("Failed constructing descriptor for chaincode %s,: %v", interest, err)
			return wrapError(errors.Errorf("failed constructing descriptor for %v", interest))
		}
		descriptors = append(descriptors, desc)
	}

	return &discovery.QueryResult{
		Result: &discovery.QueryResult_CcQueryRes{
			CcQueryRes: &discovery.ChaincodeQueryResult{
				Content: descriptors,
			},
		},
	}
}

func (s *service) configQuery(q *discovery.Query) *discovery.QueryResult {
	conf, err := s.Config(q.Channel)
	if err != nil {
		logger.Errorf("Failed fetching config for channel %s: %v", q.Channel, err)
		return wrapError(errors.Errorf("failed fetching config for channel %s", q.Channel))
	}
	return &discovery.QueryResult{
		Result: &discovery.QueryResult_ConfigResult{
			ConfigResult: conf,
		},
	}
}

func wrapPeerResponse(peersByOrg map[string]*discovery.Peers) *discovery.QueryResult {
	return &discovery.QueryResult{
		Result: &discovery.QueryResult_Members{
			Members: &discovery.PeerMembershipResult{
				PeersByOrg: peersByOrg,
			},
		},
	}
}

func (s *service) channelMembershipResponse(q *discovery.Query) *discovery.QueryResult {
	chanPeers, err := s.PeersAuthorizedByCriteria(common2.ChainID(q.Channel), q.GetPeerQuery().Filter)
	if err != nil {
		return wrapError(err)
	}
	membersByOrgs := make(map[string]*discovery.Peers)
	chanPeerByID := discovery2.Members(chanPeers).ByID()
	for org, ids2Peers := range s.computeMembership(q) {
		membersByOrgs[org] = &discovery.Peers{}
		for id, peer := range ids2Peers {
//检查对等端是否在通道视图中
			stateInfoMsg, exists := chanPeerByID[string(id)]
//如果对等端不在通道视图中，请跳过它，不要将其包含在响应中。
			if !exists {
				continue
			}
			peer.StateInfo = stateInfoMsg.Envelope
			membersByOrgs[org].Peers = append(membersByOrgs[org].Peers, peer)
		}
	}
	return wrapPeerResponse(membersByOrgs)
}

func (s *service) localMembershipResponse(q *discovery.Query) *discovery.QueryResult {
	membersByOrgs := make(map[string]*discovery.Peers)
	for org, ids2Peers := range s.computeMembership(q) {
		membersByOrgs[org] = &discovery.Peers{}
		for _, peer := range ids2Peers {
			membersByOrgs[org].Peers = append(membersByOrgs[org].Peers, peer)
		}
	}
	return wrapPeerResponse(membersByOrgs)
}

func (s *service) computeMembership(_ *discovery.Query) map[string]peerMapping {
	peersByOrg := make(map[string]peerMapping)
	peerAliveInfo := discovery2.Members(s.Peers()).ByID()
	for org, peerIdentities := range s.IdentityInfo().ByOrg() {
		peersForCurrentOrg := make(peerMapping)
		peersByOrg[org] = peersForCurrentOrg
		for _, id := range peerIdentities {
//检查活动成员身份视图中是否存在对等机
			aliveInfo, exists := peerAliveInfo[string(id.PKIId)]
			if !exists {
				continue
			}
			peersForCurrentOrg[string(id.PKIId)] = &discovery.Peer{
				Identity:       id.Identity,
				MembershipInfo: aliveInfo.Envelope,
			}
		}
	}
	return peersByOrg
}

//validatestructure验证请求是否包含所有需要的字段，以及是否正确计算这些字段。
func validateStructure(ctx context.Context, request *discovery.SignedRequest, tlsEnabled bool, certHashFromContext certHashExtractor) (*discovery.Request, error) {
	if request == nil {
		return nil, errors.New("nil request")
	}
	req, err := request.ToRequest()
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing request")
	}
	if req.Authentication == nil {
		return nil, errors.New("access denied, no authentication info in request")
	}
	if len(req.Authentication.ClientIdentity) == 0 {
		return nil, errors.New("access denied, client identity wasn't supplied")
	}
	if !tlsEnabled {
		return req, nil
	}
	computedHash := certHashFromContext(ctx)
	if len(computedHash) == 0 {
		return nil, errors.New("client didn't send a TLS certificate")
	}
	if !bytes.Equal(computedHash, req.Authentication.ClientTlsCertHash) {
		claimed := hex.EncodeToString(req.Authentication.ClientTlsCertHash)
		logger.Warningf("client claimed TLS hash %s doesn't match computed TLS hash from gRPC stream %s", claimed, hex.EncodeToString(computedHash))
		return nil, errors.New("client claimed TLS hash doesn't match computed TLS hash from gRPC stream")
	}
	return req, nil
}

func validateCCQuery(ccQuery *discovery.ChaincodeQuery) error {
	if len(ccQuery.Interests) == 0 {
		return errors.New("chaincode query must have at least one chaincode interest")
	}
	for _, interest := range ccQuery.Interests {
		if interest == nil {
			return errors.New("chaincode interest is nil")
		}
		if len(interest.Chaincodes) == 0 {
			return errors.New("chaincode interest must contain at least one chaincode")
		}
		for _, cc := range interest.Chaincodes {
			if cc.Name == "" {
				return errors.New("chaincode name in interest cannot be empty")
			}
		}
	}
	return nil
}

func wrapError(err error) *discovery.QueryResult {
	return &discovery.QueryResult{
		Result: &discovery.QueryResult_Error{
			Error: &discovery.Error{
				Content: err.Error(),
			},
		},
	}
}
