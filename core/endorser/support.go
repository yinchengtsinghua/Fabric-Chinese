
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


package endorser

import (
	"fmt"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/handlers/decoration"
	. "github.com/hyperledger/fabric/core/handlers/endorsement/api/identities"
	"github.com/hyperledger/fabric/core/handlers/library"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/scc"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//SupportImpl提供了背书器的实现。支持接口
//向对等机的各种静态方法发出调用
type SupportImpl struct {
	*PluginEndorser
	crypto.SignerSupport
	Peer             peer.Operations
	PeerSupport      peer.Support
	ChaincodeSupport *chaincode.ChaincodeSupport
	SysCCProvider    *scc.Provider
	ACLProvider      aclmgmt.ACLProvider
}

func (s *SupportImpl) NewQueryCreator(channel string) (QueryCreator, error) {
	lgr := s.Peer.GetLedger(channel)
	if lgr == nil {
		return nil, errors.Errorf("channel %s doesn't exist", channel)
	}
	return lgr, nil
}

func (s *SupportImpl) SigningIdentityForRequest(*pb.SignedProposal) (SigningIdentity, error) {
	return s.SignerSupport, nil
}

//如果提供的链码为
//IA系统链码，不可调用
func (s *SupportImpl) IsSysCCAndNotInvokableExternal(name string) bool {
	return s.SysCCProvider.IsSysCCAndNotInvokableExternal(name)
}

//GetTxSimulator返回指定分类帐的事务模拟器
//客户可以获得多个这样的模拟器；它们是独一无二的
//通过提供的txid
func (s *SupportImpl) GetTxSimulator(ledgername string, txid string) (ledger.TxSimulator, error) {
	lgr := s.Peer.GetLedger(ledgername)
	if lgr == nil {
		return nil, errors.Errorf("Channel does not exist: %s", ledgername)
	}
	return lgr.NewTxSimulator(txid)
}

//GetHistoryQueryExecutor为
//指定的分类帐
func (s *SupportImpl) GetHistoryQueryExecutor(ledgername string) (ledger.HistoryQueryExecutor, error) {
	lgr := s.Peer.GetLedger(ledgername)
	if lgr == nil {
		return nil, errors.Errorf("Channel does not exist: %s", ledgername)
	}
	return lgr.NewHistoryQueryExecutor()
}

//GetTransactionByID按ID检索事务
func (s *SupportImpl) GetTransactionByID(chid, txID string) (*pb.ProcessedTransaction, error) {
	lgr := s.Peer.GetLedger(chid)
	if lgr == nil {
		return nil, errors.Errorf("failed to look up the ledger for Channel %s", chid)
	}
	tx, err := lgr.GetTransactionByID(txID)
	if err != nil {
		return nil, errors.WithMessage(err, "GetTransactionByID failed")
	}
	return tx, nil
}

//GetLedgerHeight返回给定ChannelID的分类帐高度
func (s *SupportImpl) GetLedgerHeight(channelID string) (uint64, error) {
	lgr := s.Peer.GetLedger(channelID)
	if lgr == nil {
		return 0, errors.Errorf("failed to look up the ledger for Channel %s", channelID)
	}

	info, err := lgr.GetBlockchainInfo()
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("failed to obtain information for Channel %s", channelID))
	}

	return info.Height, nil
}

//如果名称与系统链码匹配，IsSyscc将返回true
//系统链代码名称为系统、链范围
func (s *SupportImpl) IsSysCC(name string) bool {
	return s.SysCCProvider.IsSysCC(name)
}

//getchaincode从fs返回ccpackage
func (s *SupportImpl) GetChaincodeDeploymentSpecFS(cds *pb.ChaincodeDeploymentSpec) (*pb.ChaincodeDeploymentSpec, error) {
	ccpack, err := ccprovider.GetChaincodeFromFS(cds.ChaincodeSpec.ChaincodeId.Name, cds.ChaincodeSpec.ChaincodeId.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get chaincode from fs")
	}

	return ccpack.GetDepSpec(), nil
}

//执行部署建议并返回chaincode响应
func (s *SupportImpl) ExecuteLegacyInit(txParams *ccprovider.TransactionParams, cid, name, version, txid string, signedProp *pb.SignedProposal, prop *pb.Proposal, cds *pb.ChaincodeDeploymentSpec) (*pb.Response, *pb.ChaincodeEvent, error) {
	cccid := &ccprovider.CCContext{
		Name:    name,
		Version: version,
	}

	return s.ChaincodeSupport.ExecuteLegacyInit(txParams, cccid, cds)
}

//执行一个建议并返回chaincode响应
func (s *SupportImpl) Execute(txParams *ccprovider.TransactionParams, cid, name, version, txid string, signedProp *pb.SignedProposal, prop *pb.Proposal, input *pb.ChaincodeInput) (*pb.Response, *pb.ChaincodeEvent, error) {
	cccid := &ccprovider.CCContext{
		Name:    name,
		Version: version,
	}

//装饰链码输入
	decorators := library.InitRegistry(library.Config{}).Lookup(library.Decoration).([]decoration.Decorator)
	input.Decorations = make(map[string][]byte)
	input = decoration.Apply(prop, input, decorators...)
	txParams.ProposalDecorations = input.Decorations

	return s.ChaincodeSupport.Execute(txParams, cccid, input)
}

//getchaincodedefinition返回具有所提供名称的链代码的ccprovider.chaincodedefinition
func (s *SupportImpl) GetChaincodeDefinition(chaincodeName string, txsim ledger.QueryExecutor) (ccprovider.ChaincodeDefinition, error) {
	return s.ChaincodeSupport.Lifecycle.ChaincodeDefinition(chaincodeName, txsim)
}

//checkacl使用
//可从中提取ID以针对策略进行测试的已签名的路径
func (s *SupportImpl) CheckACL(signedProp *pb.SignedProposal, chdr *common.ChannelHeader, shdr *common.SignatureHeader, hdrext *pb.ChaincodeHeaderExtension) error {
	return s.ACLProvider.CheckACL(resources.Peer_Propose, chdr.ChannelId, signedProp)
}

//如果cds包字节描述链码，isjavacc返回true
//这要求Java运行时环境执行。
func (s *SupportImpl) IsJavaCC(buf []byte) (bool, error) {
//内部DEP规范将包含类型
	ccpack, err := ccprovider.GetCCPackage(buf)
	if err != nil {
		return false, err
	}
	cds := ccpack.GetDepSpec()
	return (cds.ChaincodeSpec.Type == pb.ChaincodeSpec_JAVA), nil
}

//如果提供的
//chaincodedefinition与存储在分类帐上的实例化策略不同
func (s *SupportImpl) CheckInstantiationPolicy(name, version string, cd ccprovider.ChaincodeDefinition) error {
	return ccprovider.CheckInstantiationPolicy(name, version, cd.(*ccprovider.ChaincodeData))
}

//GetApplicationConfig返回通道的configTxApplication.sharedConfig
//以及应用程序配置是否存在
func (s *SupportImpl) GetApplicationConfig(cid string) (channelconfig.Application, bool) {
	return s.PeerSupport.GetApplicationConfig(cid)
}
