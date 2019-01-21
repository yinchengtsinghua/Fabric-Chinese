
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


package endorsement

import (
	"fmt"

	"github.com/hyperledger/fabric/common/chaincode"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/graph"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/policies/inquire"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	. "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/pkg/errors"
)

var (
	logger = flogging.MustGetLogger("discovery.endorsement")
)

type principalEvaluator interface {
//satisfies principal返回给定的对等标识是否满足某个主体
//on a given channel
	SatisfiesPrincipal(channel string, identity []byte, principal *msp.MSPPrincipal) error
}

type chaincodeMetadataFetcher interface {
//chaincode metadata返回出现在分类帐中的chaincode的元数据，
//or nil if the channel doesn't exist, or the chaincode isn't found in the ledger
	Metadata(channel string, cc string, loadCollections bool) *chaincode.Metadata
}

type policyFetcher interface {
//policyByChainCode返回可查询哪些标识的策略
//满足它
	PolicyByChaincode(channel string, cc string) policies.InquireablePolicy
}

type gossipSupport interface {
//IdentityInfo返回有关对等方的标识信息
	IdentityInfo() api.PeerIdentitySet

//peersofchannel返回被认为是活动的网络成员
//也订阅了给定的频道
	PeersOfChannel(common.ChainID) Members

//对等方返回被认为是活动的网络成员
	Peers() Members
}

type endorsementAnalyzer struct {
	gossipSupport
	principalEvaluator
	policyFetcher
	chaincodeMetadataFetcher
}

//New背书Analyzer在给定的支持之外构造一个New背书Analyzer
func NewEndorsementAnalyzer(gs gossipSupport, pf policyFetcher, pe principalEvaluator, mf chaincodeMetadataFetcher) *endorsementAnalyzer {
	return &endorsementAnalyzer{
		gossipSupport:            gs,
		policyFetcher:            pf,
		principalEvaluator:       pe,
		chaincodeMetadataFetcher: mf,
	}
}

type peerPrincipalEvaluator func(member NetworkMember, principal *msp.MSPPrincipal) bool

//PeersforElement返回给定对等方、通道和链码集的认可描述符
func (ea *endorsementAnalyzer) PeersForEndorsement(chainID common.ChainID, interest *discovery.ChaincodeInterest) (*discovery.EndorsementDescriptor, error) {
	chanMembership, err := ea.PeersAuthorizedByCriteria(chainID, interest)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	channelMembersById := chanMembership.ByID()
//仅选择已加入频道的活动消息
	aliveMembership := ea.Peers().Intersect(chanMembership)
	membersById := aliveMembership.ByID()
//计算成员的pki ID与其标识之间的映射
	identitiesOfMembers := computeIdentitiesOfMembers(ea.IdentityInfo(), membersById)
	principalsSets, err := ea.computePrincipalSets(chainID, interest)
	if err != nil {
		logger.Warningf("Principal set computation failed: %v", err)
		return nil, errors.WithStack(err)
	}

	return ea.computeEndorsementResponse(&context{
		chaincode:           interest.Chaincodes[0].Name,
		channel:             string(chainID),
		principalsSets:      principalsSets,
		channelMembersById:  channelMembersById,
		aliveMembership:     aliveMembership,
		identitiesOfMembers: identitiesOfMembers,
	})
}

func (ea *endorsementAnalyzer) PeersAuthorizedByCriteria(chainID common.ChainID, interest *discovery.ChaincodeInterest) (Members, error) {
	peersOfChannel := ea.PeersOfChannel(chainID)
	if interest == nil || len(interest.Chaincodes) == 0 {
		return peersOfChannel, nil
	}
	identities := ea.IdentityInfo()
	identitiesByID := identities.ByID()
	metadataAndCollectionFilters, err := loadMetadataAndFilters(metadataAndFilterContext{
		identityInfoByID: identitiesByID,
		interest:         interest,
		chainID:          chainID,
		evaluator:        ea,
		fetch:            ea,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	metadata := metadataAndCollectionFilters.md
//Filter out peers that don't have the chaincode installed on them
	chanMembership := peersOfChannel.Filter(peersWithChaincode(metadata...))
//筛选出未经链码调用链的集合配置授权的对等方
	return chanMembership.Filter(metadataAndCollectionFilters.isMemberAuthorized), nil
}

type context struct {
	chaincode           string
	channel             string
	aliveMembership     Members
	principalsSets      []policies.PrincipalSet
	channelMembersById  map[string]NetworkMember
	identitiesOfMembers memberIdentities
}

func (ea *endorsementAnalyzer) computeEndorsementResponse(ctx *context) (*discovery.EndorsementDescriptor, error) {
//mapPrincipalsToGroups returns a mapping from principals to their corresponding groups.
//组只是人类可读的表示，它掩盖了它们背后的主体。
	principalGroups := mapPrincipalsToGroups(ctx.principalsSets)
//PrincipalsTopersgraph计算二部图（v1 u v2，e）
//这样，v1是对等的，v2是主体，
//如果对等方满足主体，则每个e=（对等方，主体）都在e中。
	satGraph := principalsToPeersGraph(principalAndPeerData{
		members: ctx.aliveMembership,
		pGrps:   principalGroups,
	}, ea.satisfiesPrincipal(ctx.channel, ctx.identitiesOfMembers))

	layouts := computeLayouts(ctx.principalsSets, principalGroups, satGraph)
	if len(layouts) == 0 {
		return nil, errors.New("cannot satisfy any principal combination")
	}

	criteria := &peerMembershipCriteria{
		possibleLayouts: layouts,
		satGraph:        satGraph,
		chanMemberById:  ctx.channelMembersById,
		idOfMembers:     ctx.identitiesOfMembers,
	}

	return &discovery.EndorsementDescriptor{
		Chaincode:         ctx.chaincode,
		Layouts:           layouts,
		EndorsersByGroups: endorsersByGroup(criteria),
	}, nil
}

func (ea *endorsementAnalyzer) computePrincipalSets(chainID common.ChainID, interest *discovery.ChaincodeInterest) (policies.PrincipalSets, error) {
	var inquireablePolicies []policies.InquireablePolicy
	for _, chaincode := range interest.Chaincodes {
		pol := ea.PolicyByChaincode(string(chainID), chaincode.Name)
		if pol == nil {
			logger.Debug("Policy for chaincode '", chaincode, "'doesn't exist")
			return nil, errors.New("policy not found")
		}
		inquireablePolicies = append(inquireablePolicies, pol)
	}

	var cpss []inquire.ComparablePrincipalSets

	for _, policy := range inquireablePolicies {
		var cmpsets inquire.ComparablePrincipalSets
		for _, ps := range policy.SatisfiedBy() {
			cps := inquire.NewComparablePrincipalSet(ps)
			if cps == nil {
				return nil, errors.New("failed creating a comparable principal set")
			}
			cmpsets = append(cmpsets, cps)
		}
		if len(cmpsets) == 0 {
			return nil, errors.New("chaincode isn't installed on sufficient organizations required by the endorsement policy")
		}
		cpss = append(cpss, cmpsets)
	}

	cps, err := mergePrincipalSets(cpss)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return cps.ToPrincipalSets(), nil
}

type metadataAndFilterContext struct {
	chainID          common.ChainID
	interest         *discovery.ChaincodeInterest
	fetch            chaincodeMetadataFetcher
	identityInfoByID map[string]api.PeerIdentityInfo
	evaluator        principalEvaluator
}

//metadataAndColFilter holds metadata and member filters
type metadataAndColFilter struct {
	md                 []*chaincode.Metadata
	isMemberAuthorized memberFilter
}

func loadMetadataAndFilters(ctx metadataAndFilterContext) (*metadataAndColFilter, error) {
	var metadata []*chaincode.Metadata
	var filters []identityFilter

	for _, chaincode := range ctx.interest.Chaincodes {
		ccMD := ctx.fetch.Metadata(string(ctx.chainID), chaincode.Name, len(chaincode.CollectionNames) > 0)
		if ccMD == nil {
			return nil, errors.Errorf("No metadata was found for chaincode %s in channel %s", chaincode.Name, string(ctx.chainID))
		}
		metadata = append(metadata, ccMD)
		if len(chaincode.CollectionNames) == 0 {
			continue
		}
		principalSetByCollections, err := principalsFromCollectionConfig(ccMD.CollectionsConfig)
		if err != nil {
			logger.Warningf("Failed initializing collection filter for chaincode %s: %v", chaincode.Name, err)
			return nil, errors.WithStack(err)
		}
		filter, err := principalSetByCollections.toIdentityFilter(string(ctx.chainID), ctx.evaluator, chaincode)
		if err != nil {
			logger.Warningf("Failed computing collection principal sets for chaincode %s due to %v", chaincode.Name, err)
			return nil, errors.WithStack(err)
		}
		filters = append(filters, filter)
	}

	return computeFiltersWithMetadata(filters, metadata, ctx.identityInfoByID), nil
}

func computeFiltersWithMetadata(filters identityFilters, metadata []*chaincode.Metadata, identityInfoByID map[string]api.PeerIdentityInfo) *metadataAndColFilter {
	if len(filters) == 0 {
		return &metadataAndColFilter{
			md:                 metadata,
			isMemberAuthorized: noopMemberFilter,
		}
	}
	filter := filters.combine().toMemberFilter(identityInfoByID)
	return &metadataAndColFilter{
		isMemberAuthorized: filter,
		md:                 metadata,
	}
}

//IdentityFilter接受或拒绝对等标识
type identityFilter func(api.PeerIdentityType) bool

//IdentityFilters聚合多个IdentityFilters
type identityFilters []identityFilter

//memberfilter接受或拒绝networkmembers
type memberFilter func(member NetworkMember) bool

//NoopMemberFilter接受每个NetworkMember
func noopMemberFilter(_ NetworkMember) bool {
	return true
}

//Combine将所有IdentityFilter组合为一个只接受标识的IdentityFilter
//which all the original filters accept
func (filters identityFilters) combine() identityFilter {
	return func(identity api.PeerIdentityType) bool {
		for _, f := range filters {
			if !f(identity) {
				return false
			}
		}
		return true
	}
}

//to memberfilter根据给定的映射将此Identityfilter转换为memberfilter
//从pki-id作为字符串，到保存对等标识的peerIdentityInfo
func (idf identityFilter) toMemberFilter(identityInfoByID map[string]api.PeerIdentityInfo) memberFilter {
	return func(member NetworkMember) bool {
		identity, exists := identityInfoByID[string(member.PKIid)]
		if !exists {
			return false
		}
		return idf(identity.Identity)
	}
}

func (ea *endorsementAnalyzer) satisfiesPrincipal(channel string, identitiesOfMembers memberIdentities) peerPrincipalEvaluator {
	return func(member NetworkMember, principal *msp.MSPPrincipal) bool {
		err := ea.SatisfiesPrincipal(channel, identitiesOfMembers.identityByPKIID(member.PKIid), principal)
		if err == nil {
//TODO:以可读的形式记录主体
			logger.Debug(member, "satisfies principal", principal)
			return true
		}
		logger.Debug(member, "doesn't satisfy principal", principal, ":", err)
		return false
	}
}

type peerMembershipCriteria struct {
	satGraph        *principalPeerGraph
	idOfMembers     memberIdentities
	chanMemberById  map[string]NetworkMember
	possibleLayouts layouts
}

//背书人按组计算从组到对等方的映射。
//每组包括，在一些布局中找到，这意味着
//有一些主要的组合包括
//组。
//这意味着如果一个组不包含在结果中，则没有
//本金组合（包括与本集团对应的本金）；
//这样就有足够的同龄人来满足主要的组合。
func endorsersByGroup(criteria *peerMembershipCriteria) map[string]*discovery.Peers {
	satGraph := criteria.satGraph
	idOfMembers := criteria.idOfMembers
	chanMemberById := criteria.chanMemberById
	includedGroups := criteria.possibleLayouts.groupsSet()

	res := make(map[string]*discovery.Peers)
//将背书人映射到相应的组。
//迭代主体，并将对等点放入与主体顶点对应的每个组中。
	for grp, principalVertex := range satGraph.principalVertices {
		if _, exists := includedGroups[grp]; !exists {
//如果在任何布局中都找不到当前组，则跳过相应的主体
			continue
		}
		peerList := &discovery.Peers{}
		res[grp] = peerList
		for _, peerVertex := range principalVertex.Neighbors() {
			member := peerVertex.Data.(NetworkMember)
			peerList.Peers = append(peerList.Peers, &discovery.Peer{
				Identity:       idOfMembers.identityByPKIID(member.PKIid),
				StateInfo:      chanMemberById[string(member.PKIid)].Envelope,
				MembershipInfo: member.Envelope,
			})
		}
	}
	return res
}

//计算布局计算所有可能的主体组合
//它可以用来满足背书政策，给出了一个图表
//将每个对等点映射到其满足的主体的可用对等点。
//每一个这样的组合称为布局，因为它映射
//一个组（主体的别名）到需要认可的对等方的阈值，
//满足相应的主体。
func computeLayouts(principalsSets []policies.PrincipalSet, principalGroups principalGroupMapper, satGraph *principalPeerGraph) []*discovery.Layout {
	var layouts []*discovery.Layout
//Principalset是主体组合的集合，
//这样，每个组合（给予足够的同行）都满足背书政策。
	for _, principalSet := range principalsSets {
		layout := &discovery.Layout{
			QuantitiesByGroup: make(map[string]uint32),
		}
//因为Principalset有重复，我们首先
//计算从主体到集合中重复的映射。
		for principal, plurality := range principalSet.UniqueSet() {
			key := principalKey{
				cls:       int32(principal.PrincipalClassification),
				principal: string(principal.Principal),
			}
//我们将主体映射到一个组，该组是主体的别名。
			layout.QuantitiesByGroup[principalGroups.group(key)] = uint32(plurality)
		}
//检查布局是否能满足当前已知的对等点
//通过迭代当前布局并确保
//每个主顶点连接到至少<multiple>对等顶点。
		if isLayoutSatisfied(layout.QuantitiesByGroup, satGraph) {
//如果是这样，那么将布局添加到布局中，因为我们有足够的对等方来满足
//主要组合
			layouts = append(layouts, layout)
		}
	}
	return layouts
}

func isLayoutSatisfied(layout map[string]uint32, satGraph *principalPeerGraph) bool {
	for grp, plurality := range layout {
//我们是否有多个连接到主体的<multiple>对等端？
		if len(satGraph.principalVertices[grp].Neighbors()) < int(plurality) {
			return false
		}
	}
	return true
}

type principalPeerGraph struct {
	peerVertices      []*graph.Vertex
	principalVertices map[string]*graph.Vertex
}

type principalAndPeerData struct {
	members Members
	pGrps   principalGroupMapper
}

func principalsToPeersGraph(data principalAndPeerData, satisfiesPrincipal peerPrincipalEvaluator) *principalPeerGraph {
//创建对等顶点
	peerVertices := make([]*graph.Vertex, len(data.members))
	for i, member := range data.members {
		peerVertices[i] = graph.NewVertex(string(member.PKIid), member)
	}

//创建主要顶点
	principalVertices := make(map[string]*graph.Vertex)
	for pKey, grp := range data.pGrps {
		principalVertices[grp] = graph.NewVertex(grp, pKey.toPrincipal())
	}

//连接主体和对等方
	for _, principalVertex := range principalVertices {
		for _, peerVertex := range peerVertices {
//如果当前对等点满足主体，则将其相应顶点与边连接。
			principal := principalVertex.Data.(*msp.MSPPrincipal)
			member := peerVertex.Data.(NetworkMember)
			if satisfiesPrincipal(member, principal) {
				peerVertex.AddNeighbor(principalVertex)
			}
		}
	}
	return &principalPeerGraph{
		peerVertices:      peerVertices,
		principalVertices: principalVertices,
	}
}

func mapPrincipalsToGroups(principalsSets []policies.PrincipalSet) principalGroupMapper {
	groupMapper := make(principalGroupMapper)
	totalPrincipals := make(map[principalKey]struct{})
	for _, principalSet := range principalsSets {
		for _, principal := range principalSet {
			totalPrincipals[principalKey{
				principal: string(principal.Principal),
				cls:       int32(principal.PrincipalClassification),
			}] = struct{}{}
		}
	}
	for principal := range totalPrincipals {
		groupMapper.group(principal)
	}
	return groupMapper
}

type memberIdentities map[string]api.PeerIdentityType

func (m memberIdentities) identityByPKIID(id common.PKIidType) api.PeerIdentityType {
	return m[string(id)]
}

func computeIdentitiesOfMembers(identitySet api.PeerIdentitySet, members map[string]NetworkMember) memberIdentities {
	identitiesByPKIID := make(map[string]api.PeerIdentityType)
	identitiesOfMembers := make(map[string]api.PeerIdentityType, len(members))
	for _, identity := range identitySet {
		identitiesByPKIID[string(identity.PKIId)] = identity.Identity
	}
	for _, member := range members {
		if identity, exists := identitiesByPKIID[string(member.PKIid)]; exists {
			identitiesOfMembers[string(member.PKIid)] = identity
		}
	}
	return identitiesOfMembers
}

//PrincipalGroupMapper将主体映射到组名称
type principalGroupMapper map[principalKey]string

func (mapper principalGroupMapper) group(principal principalKey) string {
	if grp, exists := mapper[principal]; exists {
		return grp
	}
	grp := fmt.Sprintf("G%d", len(mapper))
	mapper[principal] = grp
	return grp
}

type principalKey struct {
	cls       int32
	principal string
}

func (pk principalKey) toPrincipal() *msp.MSPPrincipal {
	return &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_Classification(pk.cls),
		Principal:               []byte(pk.principal),
	}
}

//布局是多个布局的集合
type layouts []*discovery.Layout

//Groupsset返回布局包含的一组组
func (l layouts) groupsSet() map[string]struct{} {
	m := make(map[string]struct{})
	for _, layout := range l {
		for grp := range layout.QuantitiesByGroup {
			m[grp] = struct{}{}
		}
	}
	return m
}

func peersWithChaincode(metadata ...*chaincode.Metadata) func(member NetworkMember) bool {
	return func(member NetworkMember) bool {
		if member.Properties == nil {
			return false
		}
		for _, ccMD := range metadata {
			var found bool
			for _, cc := range member.Properties.Chaincodes {
				if cc.Name == ccMD.Name && cc.Version == ccMD.Version {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
}

func mergePrincipalSets(cpss []inquire.ComparablePrincipalSets) (inquire.ComparablePrincipalSets, error) {
//首先获取第一个可比较原则集
	var cps inquire.ComparablePrincipalSets
	cps, cpss, err := popComparablePrincipalSets(cpss)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, cps2 := range cpss {
		cps = inquire.Merge(cps, cps2)
	}
	return cps, nil
}

func popComparablePrincipalSets(sets []inquire.ComparablePrincipalSets) (inquire.ComparablePrincipalSets, []inquire.ComparablePrincipalSets, error) {
	if len(sets) == 0 {
		return nil, nil, errors.New("no principal sets remained after filtering")
	}
	cps, cpss := sets[0], sets[1:]
	return cps, cpss, nil
}
