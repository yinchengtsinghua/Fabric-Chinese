
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


package lscc

import (
	"fmt"
	"regexp"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/aclmgmt/resources"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/ccmetadata"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/core/policyprovider"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	mb "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//生命周期系统链码管理部署的链码
//在这个对等体上。它通过调用提议来管理链码。
//“args”：[“部署”，<chaincodedeploymentspec>]
//“args”：[“升级”，<chaincodedeploymentspec>]
//“args”：[“停止”，<chaincodeinvocationspec>]
//“args”：[“开始”，<chaincodeinvocationspec>]

var logger = flogging.MustGetLogger("lscc")

const (
//链码生命周期命令

//安装安装命令
	INSTALL = "install"

//部署部署命令
	DEPLOY = "deploy"

//升级升级chaincode
	UPGRADE = "upgrade"

//CCEXISTS get chaincode
	CCEXISTS = "getid"

//chaincodeexists获取chaincode别名
	CHAINCODEEXISTS = "ChaincodeExists"

//getdepspec获取chaincodedeploymentspec
	GETDEPSPEC = "getdepspec"

//获取chaincodedeploymentspec别名
	GETDEPLOYMENTSPEC = "GetDeploymentSpec"

//获取链码数据
	GETCCDATA = "getccdata"

//get chaincodedata获取chaincodedata别名
	GETCHAINCODEDATA = "GetChaincodeData"

//getchaincodes获取通道上的实例化链码
	GETCHAINCODES = "getchaincodes"

//getchaincodesalias获取通道上的实例化链码
	GETCHAINCODESALIAS = "GetChaincodes"

//GetInstalledChaincodes获取对等机上已安装的链代码
	GETINSTALLEDCHAINCODES = "getinstalledchaincodes"

//getInstalledChaincodesalias获取对等机上已安装的链代码
	GETINSTALLEDCHAINCODESALIAS = "GetInstalledChaincodes"

//getCollectionsConfig获取链码的集合配置
	GETCOLLECTIONSCONFIG = "GetCollectionsConfig"

//getCollectionsConfigalias获取链码的集合配置
	GETCOLLECTIONSCONFIGALIAS = "getcollectionsconfig"

	allowedChaincodeName = "^[a-zA-Z0-9]+([-_][a-zA-Z0-9]+)*$"
	allowedCharsVersion  = "[A-Za-z0-9_.+-]+"
)

//文件系统支持包含lscc执行其任务所需的函数
type FilesystemSupport interface {
//PutChaincodeToLocalStorage存储提供的链代码
//包到本地存储（即文件系统）
	PutChaincodeToLocalStorage(ccprovider.CCPackage) error

//getchaincoderomlocalstorage检索链码包
//对于请求的链码，由名称和版本指定
	GetChaincodeFromLocalStorage(ccname string, ccversion string) (ccprovider.CCPackage, error)

//getchaincodesfromlocalstorage返回所有链码的数组
//以前保存到本地存储的数据
	GetChaincodesFromLocalStorage() (*pb.ChaincodeQueryResponse, error)

//GetInstantiationPolicy返回
//提供的链码（如果未指定，则为通道的默认值）
	GetInstantiationPolicy(channel string, ccpack ccprovider.CCPackage) ([]byte, error)

//checkInstantiationPolicy检查提供的签名建议是否
//符合提供的实例化策略
	CheckInstantiationPolicy(signedProposal *pb.SignedProposal, chainName string, instantiationPolicy []byte) error
}

//-------LSCC-----------

//LifecycleSyscc实现了链码生命周期及其周围的策略
type LifeCycleSysCC struct {
//aclprovider负责访问控制评估
	ACLProvider aclmgmt.ACLProvider

//SCCProvider是传递到系统链码的接口。
//访问系统的其他部分
	SCCProvider sysccprovider.SystemChaincodeProvider

//policyChecker是用于执行
//访问控制
	PolicyChecker policy.PolicyChecker

//支持提供了
//静态函数
	Support FilesystemSupport

	PlatformRegistry *platforms.Registry
}

//新建创建LSCC的新实例
//通常每个对等机只有一个
func New(sccp sysccprovider.SystemChaincodeProvider, ACLProvider aclmgmt.ACLProvider, platformRegistry *platforms.Registry) *LifeCycleSysCC {
	return &LifeCycleSysCC{
		Support:          &supportImpl{},
		PolicyChecker:    policyprovider.GetPolicyChecker(),
		SCCProvider:      sccp,
		ACLProvider:      ACLProvider,
		PlatformRegistry: platformRegistry,
	}
}

func (lscc *LifeCycleSysCC) Name() string              { return "lscc" }
func (lscc *LifeCycleSysCC) Path() string              { return "github.com/hyperledger/fabric/core/scc/lscc" }
func (lscc *LifeCycleSysCC) InitArgs() [][]byte        { return nil }
func (lscc *LifeCycleSysCC) Chaincode() shim.Chaincode { return lscc }
func (lscc *LifeCycleSysCC) InvokableExternal() bool   { return true }
func (lscc *LifeCycleSysCC) InvokableCC2CC() bool      { return true }
func (lscc *LifeCycleSysCC) Enabled() bool             { return true }

func (lscc *LifeCycleSysCC) ChaincodeContainerInfo(chaincodeName string, qe ledger.QueryExecutor) (*ccprovider.ChaincodeContainerInfo, error) {
	chaincodeDataBytes, err := qe.GetState("lscc", chaincodeName)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve state for chaincode %s", chaincodeName)
	}

	if chaincodeDataBytes == nil {
		return nil, errors.Errorf("chaincode %s not found", chaincodeName)
	}

//注意，尽管用
//下面的“chaincodedefinition”调用、“getccode”调用为我们提供了安全性
//由于副作用，所以我们现在必须保持原样。
	cds, _, err := lscc.getCCCode(chaincodeName, chaincodeDataBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get chaincode code")
	}

	return ccprovider.DeploymentSpecToChaincodeContainerInfo(cds), nil
}

func (lscc *LifeCycleSysCC) ChaincodeDefinition(chaincodeName string, qe ledger.QueryExecutor) (ccprovider.ChaincodeDefinition, error) {
	chaincodeDataBytes, err := qe.GetState("lscc", chaincodeName)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve state for chaincode %s", chaincodeName)
	}

	if chaincodeDataBytes == nil {
		return nil, errors.Errorf("chaincode %s not found", chaincodeName)
	}

	chaincodeData := &ccprovider.ChaincodeData{}
	err = proto.Unmarshal(chaincodeDataBytes, chaincodeData)
	if err != nil {
		return nil, errors.Wrapf(err, "chaincode %s has bad definition", chaincodeName)
	}

	return chaincodeData, nil
}

//在给定链上创建链代码
func (lscc *LifeCycleSysCC) putChaincodeData(stub shim.ChaincodeStubInterface, cd *ccprovider.ChaincodeData) error {
	cdbytes, err := proto.Marshal(cd)
	if err != nil {
		return err
	}

	if cdbytes == nil {
		return MarshallErr(cd.Name)
	}

	err = stub.PutState(cd.Name, cdbytes)

	return err
}

//CheckCollectionMemberPolicy检查提供的集合配置是否
//complies to the given msp configuration
func checkCollectionMemberPolicy(collectionConfig *common.CollectionConfig, mspmgr msp.MSPManager) error {
	if mspmgr == nil {
		return fmt.Errorf("msp manager not set")
	}
	msps, err := mspmgr.GetMSPs()
	if err != nil {
		return errors.Wrapf(err, "error getting channel msp")
	}
	if collectionConfig == nil {
		return fmt.Errorf("collection configuration is not set")
	}
	coll := collectionConfig.GetStaticCollectionConfig()
	if coll == nil {
		return fmt.Errorf("collection configuration is empty")
	}
	if coll.MemberOrgsPolicy == nil {
		return fmt.Errorf("collection member policy is not set")
	}
	if coll.MemberOrgsPolicy.GetSignaturePolicy() == nil {
		return fmt.Errorf("collection member org policy is empty")
	}
//确保列出的组织实际上是频道的一部分
//检查签名策略中的所有主体
	for _, principal := range coll.MemberOrgsPolicy.GetSignaturePolicy().Identities {
		found := false
		var orgID string
//成员组织策略只支持某些主体类型
		switch principal.PrincipalClassification {

		case mb.MSPPrincipal_ROLE:
			msprole := &mb.MSPRole{}
			err := proto.Unmarshal(principal.Principal, msprole)
			if err != nil {
				return errors.Wrapf(err, "collection-name: %s -- cannot unmarshal identities", coll.GetName())
			}
			orgID = msprole.MspIdentifier
//MSP映射是使用MSP ID建立索引的-这种行为是特定于实现的，使得下面的检查有点黑客行为
			for mspid := range msps {
				if mspid == orgID {
					found = true
					break
				}
			}

		case mb.MSPPrincipal_ORGANIZATION_UNIT:
			mspou := &mb.OrganizationUnit{}
			err := proto.Unmarshal(principal.Principal, mspou)
			if err != nil {
				return errors.Wrapf(err, "collection-name: %s -- cannot unmarshal identities", coll.GetName())
			}
			orgID = mspou.MspIdentifier
//MSP映射是使用MSP ID建立索引的-这种行为是特定于实现的，使得下面的检查有点黑客行为
			for mspid := range msps {
				if mspid == orgID {
					found = true
					break
				}
			}

		case mb.MSPPrincipal_IDENTITY:
			orgID = "identity principal"
			for _, msp := range msps {
				_, err := msp.DeserializeIdentity(principal.Principal)
				if err == nil {
					found = true
					break
				}
			}
		default:
			return fmt.Errorf("collection-name: %s -- principal type %v is not supported", coll.GetName(), principal.PrincipalClassification)
		}
		if !found {
			logger.Warningf("collection-name: %s collection member %s is not part of the channel", coll.GetName(), orgID)
		}
	}

	return nil
}

//PutChaincodeCollectionData为链代码添加收集数据
func (lscc *LifeCycleSysCC) putChaincodeCollectionData(stub shim.ChaincodeStubInterface, cd *ccprovider.ChaincodeData, collectionConfigBytes []byte) error {
	if cd == nil {
		return errors.New("nil ChaincodeData")
	}

	if len(collectionConfigBytes) == 0 {
		logger.Debug("No collection configuration specified")
		return nil
	}

	collections := &common.CollectionConfigPackage{}
	err := proto.Unmarshal(collectionConfigBytes, collections)
	if err != nil {
		return errors.Errorf("invalid collection configuration supplied for chaincode %s:%s", cd.Name, cd.Version)
	}

	mspmgr := mgmt.GetManagerForChain(stub.GetChannelID())
	if mspmgr == nil {
		return fmt.Errorf("could not get MSP manager for channel %s", stub.GetChannelID())
	}
	for _, collectionConfig := range collections.Config {
		err = checkCollectionMemberPolicy(collectionConfig, mspmgr)
		if err != nil {
			return errors.Wrapf(err, "collection member policy check failed")
		}
	}

	key := privdata.BuildCollectionKVSKey(cd.Name)

	err = stub.PutState(key, collectionConfigBytes)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("error putting collection for chaincode %s:%s", cd.Name, cd.Version))
	}

	return nil
}

//getchaincodecollectiondata检索集合配置。
func (lscc *LifeCycleSysCC) getChaincodeCollectionData(stub shim.ChaincodeStubInterface, chaincodeName string) pb.Response {
	key := privdata.BuildCollectionKVSKey(chaincodeName)
	collectionsConfigBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(collectionsConfigBytes) == 0 {
		return shim.Error(fmt.Sprintf("collections config not defined for chaincode %s", chaincodeName))
	}
	return shim.Success(collectionsConfigBytes)
}

//checks for existence of chaincode on the given channel
func (lscc *LifeCycleSysCC) getCCInstance(stub shim.ChaincodeStubInterface, ccname string) ([]byte, error) {
	cdbytes, err := stub.GetState(ccname)
	if err != nil {
		return nil, TXNotFoundErr(err.Error())
	}
	if cdbytes == nil {
		return nil, NotFoundErr(ccname)
	}

	return cdbytes, nil
}

//从字节中取出CD
func (lscc *LifeCycleSysCC) getChaincodeData(ccname string, cdbytes []byte) (*ccprovider.ChaincodeData, error) {
	cd := &ccprovider.ChaincodeData{}
	err := proto.Unmarshal(cdbytes, cd)
	if err != nil {
		return nil, MarshallErr(ccname)
	}

//这不应该发生，但一个健全的检查不是一件坏事
	if cd.Name != ccname {
		return nil, ChaincodeMismatchErr(fmt.Sprintf("%s!=%s", ccname, cd.Name))
	}

	return cd, nil
}

//检查给定链上是否存在链码
func (lscc *LifeCycleSysCC) getCCCode(ccname string, cdbytes []byte) (*pb.ChaincodeDeploymentSpec, []byte, error) {
	cd, err := lscc.getChaincodeData(ccname, cdbytes)
	if err != nil {
		return nil, nil, err
	}

	ccpack, err := lscc.Support.GetChaincodeFromLocalStorage(ccname, cd.Version)
	if err != nil {
		return nil, nil, InvalidDeploymentSpecErr(err.Error())
	}

//这是一个巨大的考验，也是每次发射都要经历的原因。
//getchaincode呼叫。我们根据
//fs中的链码
	if err = ccpack.ValidateCC(cd); err != nil {
		return nil, nil, InvalidCCOnFSError(err.Error())
	}

//这些保证是非零的，因为我们有一个有效的ccpack
	depspec := ccpack.GetDepSpec()
	depspecbytes := ccpack.GetDepSpecBytes()

	return depspec, depspecbytes, nil
}

//getchaincodes返回此lscc通道上实例化的所有链码
func (lscc *LifeCycleSysCC) getChaincodes(stub shim.ChaincodeStubInterface) pb.Response {
//从lscc获取所有行
	itr, err := stub.GetStateByRange("", "")

	if err != nil {
		return shim.Error(err.Error())
	}
	defer itr.Close()

//用于存储LSCC中所有链码条目的元数据的数组
	var ccInfoArray []*pb.ChaincodeInfo

	for itr.HasNext() {
		response, err := itr.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

//collectionconfig不是chaincodedata
		if privdata.IsCollectionConfigKey(response.Key) {
			continue
		}

		ccdata := &ccprovider.ChaincodeData{}
		if err = proto.Unmarshal(response.Value, ccdata); err != nil {
			return shim.Error(err.Error())
		}

		var path string
		var input string

//如果系统上没有安装chaincode，我们将无法
//名称和版本以外的数据
		ccpack, err := lscc.Support.GetChaincodeFromLocalStorage(ccdata.Name, ccdata.Version)
		if err == nil {
			path = ccpack.GetDepSpec().GetChaincodeSpec().ChaincodeId.Path
			input = ccpack.GetDepSpec().GetChaincodeSpec().Input.String()
		}

//将此特定链码的元数据添加到所有链码的数组中
		ccInfo := &pb.ChaincodeInfo{Name: ccdata.Name, Version: ccdata.Version, Path: path, Input: input, Escc: ccdata.Escc, Vscc: ccdata.Vscc}
		ccInfoArray = append(ccInfoArray, ccInfo)
	}
//向查询中添加包含所有实例化链码信息的数组
//响应原
	cqr := &pb.ChaincodeQueryResponse{Chaincodes: ccInfoArray}

	cqrbytes, err := proto.Marshal(cqr)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(cqrbytes)
}

//GetInstalledChaincodes返回对等机上安装的所有链码
func (lscc *LifeCycleSysCC) getInstalledChaincodes() pb.Response {
//获取链码查询响应协议，其中包含有关所有
//已安装的链码
	cqr, err := lscc.Support.GetChaincodesFromLocalStorage()
	if err != nil {
		return shim.Error(err.Error())
	}

	cqrbytes, err := proto.Marshal(cqr)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(cqrbytes)
}

//检查频道名称的有效性
func (lscc *LifeCycleSysCC) isValidChannelName(channel string) bool {
//我们可能需要更多的支票
	if channel == "" {
		return false
	}
	return true
}

//isvalidchaincodename检查链码名称的有效性。链码名称
//不应为空，并且只应包含字母数字、“uu”和“-”
func (lscc *LifeCycleSysCC) isValidChaincodeName(chaincodeName string) error {
	if chaincodeName == "" {
		return EmptyChaincodeNameErr("")
	}

	if !isValidCCNameOrVersion(chaincodeName, allowedChaincodeName) {
		return InvalidChaincodeNameErr(chaincodeName)
	}

	return nil
}

//isvalidchaincodeversion检查链码版本的有效性。版本
//不应为空，并且只应包含字母数字、“u”、“-”，
//“+”和“。”
func (lscc *LifeCycleSysCC) isValidChaincodeVersion(chaincodeName string, version string) error {
	if version == "" {
		return EmptyVersionErr(chaincodeName)
	}

	if !isValidCCNameOrVersion(version, allowedCharsVersion) {
		return InvalidVersionErr(version)
	}

	return nil
}

func isValidCCNameOrVersion(ccNameOrVersion string, regExp string) bool {
	re, _ := regexp.Compile(regExp)

	matched := re.FindString(ccNameOrVersion)
	if len(matched) != len(ccNameOrVersion) {
		return false
	}

	return true
}

func isValidStatedbArtifactsTar(statedbArtifactsTar []byte) error {
//从存档中提取元数据文件
//为databasetype传递空字符串将验证中的所有项目
//档案馆
	archiveFiles, err := ccprovider.ExtractFileEntries(statedbArtifactsTar, "")
	if err != nil {
		return err
	}
//遍历文件并验证
	for _, archiveDirectoryFiles := range archiveFiles {
		for _, fileEntry := range archiveDirectoryFiles {
			indexData := fileEntry.FileContent
//验证基于传递的文件名，例如meta-inf/statedb/couchdb/indexes/indexname.json
			err = ccmetadata.ValidateMetadataFile(fileEntry.FileHeader.Name, indexData)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//ExecuteInstall实现“安装”调用事务
func (lscc *LifeCycleSysCC) executeInstall(stub shim.ChaincodeStubInterface, ccbytes []byte) error {
	ccpack, err := ccprovider.GetCCPackage(ccbytes)
	if err != nil {
		return err
	}

	cds := ccpack.GetDepSpec()

	if cds == nil {
		return fmt.Errorf("nil deployment spec from from the CC package")
	}

	if err = lscc.isValidChaincodeName(cds.ChaincodeSpec.ChaincodeId.Name); err != nil {
		return err
	}

	if err = lscc.isValidChaincodeVersion(cds.ChaincodeSpec.ChaincodeId.Name, cds.ChaincodeSpec.ChaincodeId.Version); err != nil {
		return err
	}

	if lscc.SCCProvider.IsSysCC(cds.ChaincodeSpec.ChaincodeId.Name) {
		return errors.Errorf("cannot install: %s is the name of a system chaincode", cds.ChaincodeSpec.ChaincodeId.Name)
	}

//从chaincode包中获取任何statedb项目，例如couchdb index definitions
	statedbArtifactsTar, err := ccprovider.ExtractStatedbArtifactsFromCCPackage(ccpack, lscc.PlatformRegistry)
	if err != nil {
		return err
	}

	if err = isValidStatedbArtifactsTar(statedbArtifactsTar); err != nil {
		return InvalidStatedbArtifactsErr(err.Error())
	}

	chaincodeDefinition := &cceventmgmt.ChaincodeDefinition{
		Name:    ccpack.GetChaincodeData().Name,
		Version: ccpack.GetChaincodeData().Version,
Hash:    ccpack.GetId()} //注意-链码“id”是链码的散列（codehash metadatahash），也就是指纹。

//handlechaincodeinstall将把任何statedb工件（例如couchdb索引）应用到
//链码已经实例化的任何通道的statedb
//注意-此步骤是在PutChaincodeToLocalStorage（）之前完成的，因为此步骤是等幂的，在开始背书之前是无害的。
//也就是说，如果部署索引时出错，可以在以后安全地重新尝试chaincode安装。
	err = cceventmgmt.GetMgr().HandleChaincodeInstall(chaincodeDefinition, statedbArtifactsTar)
	defer func() {
		cceventmgmt.GetMgr().ChaincodeInstallDone(err == nil)
	}()
	if err != nil {
		return err
	}

//Finally, if everything is good above, install the chaincode to local peer file system so that endorsements can start
	if err = lscc.Support.PutChaincodeToLocalStorage(ccpack); err != nil {
		return err
	}

	logger.Infof("Installed Chaincode [%s] Version [%s] to peer", ccpack.GetChaincodeData().Name, ccpack.GetChaincodeData().Version)

	return nil
}

//ExecuteDeployorUpgrade将代码路径路由到ExecuteDeploy或ExecuteUpgrade
//取决于它的函数参数
func (lscc *LifeCycleSysCC) executeDeployOrUpgrade(
	stub shim.ChaincodeStubInterface,
	chainname string,
	cds *pb.ChaincodeDeploymentSpec,
	policy, escc, vscc, collectionConfigBytes []byte,
	function string,
) (*ccprovider.ChaincodeData, error) {

	chaincodeName := cds.ChaincodeSpec.ChaincodeId.Name
	chaincodeVersion := cds.ChaincodeSpec.ChaincodeId.Version

	if err := lscc.isValidChaincodeName(chaincodeName); err != nil {
		return nil, err
	}

	if err := lscc.isValidChaincodeVersion(chaincodeName, chaincodeVersion); err != nil {
		return nil, err
	}

	ccpack, err := lscc.Support.GetChaincodeFromLocalStorage(chaincodeName, chaincodeVersion)
	if err != nil {
		retErrMsg := fmt.Sprintf("cannot get package for chaincode (%s:%s)", chaincodeName, chaincodeVersion)
		logger.Errorf("%s-err:%s", retErrMsg, err)
		return nil, fmt.Errorf("%s", retErrMsg)
	}
	cd := ccpack.GetChaincodeData()

	switch function {
	case DEPLOY:
		return lscc.executeDeploy(stub, chainname, cds, policy, escc, vscc, cd, ccpack, collectionConfigBytes)
	case UPGRADE:
		return lscc.executeUpgrade(stub, chainname, cds, policy, escc, vscc, cd, ccpack, collectionConfigBytes)
	default:
		logger.Panicf("Programming error, unexpected function '%s'", function)
panic("") //无法访问的代码
	}
}

//ExecuteDeploy实现“实例化”调用事务
func (lscc *LifeCycleSysCC) executeDeploy(
	stub shim.ChaincodeStubInterface,
	chainname string,
	cds *pb.ChaincodeDeploymentSpec,
	policy []byte,
	escc []byte,
	vscc []byte,
	cdfs *ccprovider.ChaincodeData,
	ccpackfs ccprovider.CCPackage,
	collectionConfigBytes []byte,
) (*ccprovider.ChaincodeData, error) {
//只是测试LSCC中是否存在链码
	chaincodeName := cds.ChaincodeSpec.ChaincodeId.Name
	_, err := lscc.getCCInstance(stub, chaincodeName)
	if err == nil {
		return nil, ExistsErr(chaincodeName)
	}

//保留链码特定的数据并填充通道特定的数据
	cdfs.Escc = string(escc)
	cdfs.Vscc = string(vscc)
	cdfs.Policy = policy

//检索和评估实例化策略
	cdfs.InstantiationPolicy, err = lscc.Support.GetInstantiationPolicy(chainname, ccpackfs)
	if err != nil {
		return nil, err
	}
//获取签名的实例化建议
	signedProp, err := stub.GetSignedProposal()
	if err != nil {
		return nil, err
	}
	err = lscc.Support.CheckInstantiationPolicy(signedProp, chainname, cdfs.InstantiationPolicy)
	if err != nil {
		return nil, err
	}

	err = lscc.putChaincodeData(stub, cdfs)
	if err != nil {
		return nil, err
	}

	err = lscc.putChaincodeCollectionData(stub, cdfs, collectionConfigBytes)
	if err != nil {
		return nil, err
	}

	return cdfs, nil
}

//ExecuteUpgrade实现“升级”调用事务。
func (lscc *LifeCycleSysCC) executeUpgrade(stub shim.ChaincodeStubInterface, chainName string, cds *pb.ChaincodeDeploymentSpec, policy []byte, escc []byte, vscc []byte, cdfs *ccprovider.ChaincodeData, ccpackfs ccprovider.CCPackage, collectionConfigBytes []byte) (*ccprovider.ChaincodeData, error) {

	chaincodeName := cds.ChaincodeSpec.ChaincodeId.Name

//仅检查是否存在chaincode实例（必须存在于通道上）
//我们不关心fs上的旧链码。特别是，用户甚至可以
//已经删除它
	cdbytes, _ := lscc.getCCInstance(stub, chaincodeName)
	if cdbytes == nil {
		return nil, NotFoundErr(chaincodeName)
	}

//我们需要CD来比较版本
	cdLedger, err := lscc.getChaincodeData(chaincodeName, cdbytes)
	if err != nil {
		return nil, err
	}

//如果版本相同，则不升级
	if cdLedger.Version == cds.ChaincodeSpec.ChaincodeId.Version {
		return nil, IdenticalVersionErr(chaincodeName)
	}

//如果违反实例化策略，则不升级
	if cdLedger.InstantiationPolicy == nil {
		return nil, InstantiationPolicyMissing("")
	}
//获取签名的实例化建议
	signedProp, err := stub.GetSignedProposal()
	if err != nil {
		return nil, err
	}
	err = lscc.Support.CheckInstantiationPolicy(signedProp, chainName, cdLedger.InstantiationPolicy)
	if err != nil {
		return nil, err
	}

//保留链码特定的数据并填充通道特定的数据
	cdfs.Escc = string(escc)
	cdfs.Vscc = string(vscc)
	cdfs.Policy = policy

//检索和评估新的实例化策略
	cdfs.InstantiationPolicy, err = lscc.Support.GetInstantiationPolicy(chainName, ccpackfs)
	if err != nil {
		return nil, err
	}
	err = lscc.Support.CheckInstantiationPolicy(signedProp, chainName, cdfs.InstantiationPolicy)
	if err != nil {
		return nil, err
	}

	err = lscc.putChaincodeData(stub, cdfs)
	if err != nil {
		return nil, err
	}

	ac, exists := lscc.SCCProvider.GetApplicationConfig(chainName)
	if !exists {
		logger.Panicf("programming error, non-existent appplication config for channel '%s'", chainName)
	}

	if ac.Capabilities().CollectionUpgrade() {
		err = lscc.putChaincodeCollectionData(stub, cdfs, collectionConfigBytes)
		if err != nil {
			return nil, err
		}
	} else {
		if collectionConfigBytes != nil {
			return nil, errors.New(CollectionsConfigUpgradesNotAllowed("").Error())
		}
	}

	lifecycleEvent := &pb.LifecycleEvent{ChaincodeName: chaincodeName}
	lifecycleEventBytes := utils.MarshalOrPanic(lifecycleEvent)
	stub.SetEvent(UPGRADE, lifecycleEventBytes)
	return cdfs, nil
}

//----------链码存根接口实现----------

//init对scc基本上是无用的
func (lscc *LifeCycleSysCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//invoke实现生命周期函数“deploy”、“start”、“stop”、“upgrade”。
//deploy的参数-[]byte（“deploy”），[]byte（<chainname>），<unmarshalled pb.chaincodedeploymentspec>
//
//invoke还实现一些类似查询的函数
//获取chaincode参数-[]byte（“getid”），[]byte（<chainname>），[]byte（<chaincodename>）
func (lscc *LifeCycleSysCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 1 {
		return shim.Error(InvalidArgsLenErr(len(args)).Error())
	}

	function := string(args[0])

//处理ACL：
//1。获取已签署的建议
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed retrieving signed proposal on executing %s with error %s", function, err))
	}

	switch function {
	case INSTALL:
		if len(args) < 2 {
			return shim.Error(InvalidArgsLenErr(len(args)).Error())
		}

//2。检查本地MSP管理员策略
		if err = lscc.PolicyChecker.CheckPolicyNoChannel(mgmt.Admins, sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s]: %s", function, err))
		}

		depSpec := args[1]

		err := lscc.executeInstall(stub, depSpec)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte("OK"))
	case DEPLOY, UPGRADE:
//我们期望至少有3个参数，函数
//name, the chain name and deployment spec
		if len(args) < 3 {
			return shim.Error(InvalidArgsLenErr(len(args)).Error())
		}

//链码应该与之关联的通道。它
//应使用寄存器调用创建
		channel := string(args[1])

		if !lscc.isValidChannelName(channel) {
			return shim.Error(InvalidChannelNameErr(channel).Error())
		}

		ac, exists := lscc.SCCProvider.GetApplicationConfig(channel)
		if !exists {
			logger.Panicf("programming error, non-existent appplication config for channel '%s'", channel)
		}

//参数的最大数目取决于通道的功能
		if !ac.Capabilities().PrivateChannelData() && len(args) > 6 {
			return shim.Error(PrivateChannelDataNotAvailable("").Error())
		}
		if ac.Capabilities().PrivateChannelData() && len(args) > 7 {
			return shim.Error(InvalidArgsLenErr(len(args)).Error())
		}

		depSpec := args[2]
		cds := &pb.ChaincodeDeploymentSpec{}
		err := proto.Unmarshal(depSpec, cds)
		if err != nil {
			return shim.Error(fmt.Sprintf("error unmarshaling ChaincodeDeploymentSpec: %s", err))
		}

//这里的可选参数（可以为零，也可以不存在）
//args[3]是代表背书策略的已整理签名policyInvelope
//args[4]是ESCC的名称
//args[5]是VSCC的名称
//args[6]是一个已整理的collectionconfigpackage结构
		var EP []byte
		if len(args) > 3 && len(args[3]) > 0 {
			EP = args[3]
		} else {
			p := cauthdsl.SignedByAnyMember(peer.GetMSPIDs(channel))
			EP, err = utils.Marshal(p)
			if err != nil {
				return shim.Error(err.Error())
			}
		}

		var escc []byte
		if len(args) > 4 && len(args[4]) > 0 {
			escc = args[4]
		} else {
			escc = []byte("escc")
		}

		var vscc []byte
		if len(args) > 5 && len(args[5]) > 0 {
			vscc = args[5]
		} else {
			vscc = []byte("vscc")
		}

		var collectionsConfig []byte
//只有当
//我们支持privatechanneldata功能
		if ac.Capabilities().PrivateChannelData() && len(args) > 6 {
			collectionsConfig = args[6]
		}

		cd, err := lscc.executeDeployOrUpgrade(stub, channel, cds, EP, escc, vscc, collectionsConfig, function)
		if err != nil {
			return shim.Error(err.Error())
		}
		cdbytes, err := proto.Marshal(cd)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(cdbytes)
	case CCEXISTS, CHAINCODEEXISTS, GETDEPSPEC, GETDEPLOYMENTSPEC, GETCCDATA, GETCHAINCODEDATA:
		if len(args) != 3 {
			return shim.Error(InvalidArgsLenErr(len(args)).Error())
		}

		channel := string(args[1])
		ccname := string(args[2])

//2。检查ACL资源的策略
		var resource string
		switch function {
		case CCEXISTS, CHAINCODEEXISTS:
			resource = resources.Lscc_ChaincodeExists
		case GETDEPSPEC, GETDEPLOYMENTSPEC:
			resource = resources.Lscc_GetDeploymentSpec
		case GETCCDATA, GETCHAINCODEDATA:
			resource = resources.Lscc_GetChaincodeData
		}
		if err = lscc.ACLProvider.CheckACL(resource, channel, sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s][%s]: %s", function, channel, err))
		}

		cdbytes, err := lscc.getCCInstance(stub, ccname)
		if err != nil {
			logger.Errorf("error getting chaincode %s on channel [%s]: %s", ccname, channel, err)
			return shim.Error(err.Error())
		}

		switch function {
		case CCEXISTS, CHAINCODEEXISTS:
			cd, err := lscc.getChaincodeData(ccname, cdbytes)
			if err != nil {
				return shim.Error(err.Error())
			}
			return shim.Success([]byte(cd.Name))
		case GETCCDATA, GETCHAINCODEDATA:
			return shim.Success(cdbytes)
		case GETDEPSPEC, GETDEPLOYMENTSPEC:
			_, depspecbytes, err := lscc.getCCCode(ccname, cdbytes)
			if err != nil {
				return shim.Error(err.Error())
			}
			return shim.Success(depspecbytes)
		default:
			panic("unreachable")
		}
	case GETCHAINCODES, GETCHAINCODESALIAS:
		if len(args) != 1 {
			return shim.Error(InvalidArgsLenErr(len(args)).Error())
		}

		if err = lscc.ACLProvider.CheckACL(resources.Lscc_GetInstantiatedChaincodes, stub.GetChannelID(), sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s][%s]: %s", function, stub.GetChannelID(), err))
		}

		return lscc.getChaincodes(stub)
	case GETINSTALLEDCHAINCODES, GETINSTALLEDCHAINCODESALIAS:
		if len(args) != 1 {
			return shim.Error(InvalidArgsLenErr(len(args)).Error())
		}

//2。检查本地MSP管理员策略
		if err = lscc.PolicyChecker.CheckPolicyNoChannel(mgmt.Admins, sp); err != nil {
			return shim.Error(fmt.Sprintf("access denied for [%s]: %s", function, err))
		}

		return lscc.getInstalledChaincodes()
	case GETCOLLECTIONSCONFIG, GETCOLLECTIONSCONFIGALIAS:
		if len(args) != 2 {
			return shim.Error(InvalidArgsLenErr(len(args)).Error())
		}

		chaincodeName := string(args[1])

		logger.Debugf("GetCollectionsConfig, chaincodeName:%s, start to check ACL for current identity policy", chaincodeName)
		if err = lscc.ACLProvider.CheckACL(resources.Lscc_GetCollectionsConfig, stub.GetChannelID(), sp); err != nil {
			logger.Debugf("ACL Check Failed for channel:%s, chaincode:%s", stub.GetChannelID(), chaincodeName)
			return shim.Error(fmt.Sprintf("access denied for [%s]: %s", function, err))
		}

		return lscc.getChaincodeCollectionData(stub, chaincodeName)
	}

	return shim.Error(InvalidFunctionErr(function).Error())
}
