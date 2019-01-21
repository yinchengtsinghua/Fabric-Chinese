
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//
//版权所有IBM公司。保留所有权利。
//
//SPDX许可证标识符：Apache-2.0
//

package peer

import (
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//操作向运行的包级函数公开接口
//单件包装上。这是迈向从包装开始的一步
//对等实例级数据的级别数据。
type Operations interface {
	CreateChainFromBlock(cb *common.Block, ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider) error
	GetChannelConfig(cid string) channelconfig.Resources
	GetChannelsInfo() []*pb.ChannelInfo
	GetCurrConfigBlock(cid string) *common.Block
	GetLedger(cid string) ledger.PeerLedger
	GetMSPIDs(cid string) []string
	GetPolicyManager(cid string) policies.Manager
	InitChain(cid string)
	Initialize(init func(string), ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider, pm txvalidator.PluginMapper, pr *platforms.Registry, deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider, membershipProvider ledger.MembershipInfoProvider, metricsProvider metrics.Provider)
}

type peerImpl struct {
	createChainFromBlock func(cb *common.Block, ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider) error
	getChannelConfig     func(cid string) channelconfig.Resources
	getChannelsInfo      func() []*pb.ChannelInfo
	getCurrConfigBlock   func(cid string) *common.Block
	getLedger            func(cid string) ledger.PeerLedger
	getMSPIDs            func(cid string) []string
	getPolicyManager     func(cid string) policies.Manager
	initChain            func(cid string)
	initialize           func(init func(string), ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider, mapper txvalidator.PluginMapper, pr *platforms.Registry, deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider, membershipProvider ledger.MembershipInfoProvider, metricsProvider metrics.Provider)
}

//默认设置在对等接口的实现中提供
//访问包级别状态。
var Default Operations = &peerImpl{
	createChainFromBlock: CreateChainFromBlock,
	getChannelConfig:     GetChannelConfig,
	getChannelsInfo:      GetChannelsInfo,
	getCurrConfigBlock:   GetCurrConfigBlock,
	getLedger:            GetLedger,
	getMSPIDs:            GetMSPIDs,
	getPolicyManager:     GetPolicyManager,
	initChain:            InitChain,
	initialize:           Initialize,
}

var DefaultSupport Support = &supportImpl{operations: Default}

func (p *peerImpl) CreateChainFromBlock(cb *common.Block, ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider) error {
	return p.createChainFromBlock(cb, ccp, sccp)
}
func (p *peerImpl) GetChannelConfig(cid string) channelconfig.Resources {
	return p.getChannelConfig(cid)
}
func (p *peerImpl) GetChannelsInfo() []*pb.ChannelInfo           { return p.getChannelsInfo() }
func (p *peerImpl) GetCurrConfigBlock(cid string) *common.Block  { return p.getCurrConfigBlock(cid) }
func (p *peerImpl) GetLedger(cid string) ledger.PeerLedger       { return p.getLedger(cid) }
func (p *peerImpl) GetMSPIDs(cid string) []string                { return p.getMSPIDs(cid) }
func (p *peerImpl) GetPolicyManager(cid string) policies.Manager { return p.getPolicyManager(cid) }
func (p *peerImpl) InitChain(cid string)                         { p.initChain(cid) }
func (p *peerImpl) Initialize(init func(string), ccp ccprovider.ChaincodeProvider, sccp sysccprovider.SystemChaincodeProvider, mapper txvalidator.PluginMapper, pr *platforms.Registry, deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider, membershipProvider ledger.MembershipInfoProvider, metricsProvider metrics.Provider) {
	p.initialize(init, ccp, sccp, mapper, pr, deployedCCInfoProvider, membershipProvider, metricsProvider)
}
