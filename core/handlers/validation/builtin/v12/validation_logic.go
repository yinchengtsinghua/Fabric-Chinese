
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


package v12

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/car"
	"github.com/hyperledger/fabric/core/chaincode/platforms/ccmetadata"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/chaincode/platforms/java"
	"github.com/hyperledger/fabric/core/chaincode/platforms/node"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/capabilities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/identities"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/policies"
	. "github.com/hyperledger/fabric/core/handlers/validation/api/state"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/scc/lscc"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("vscc")

const (
	DUPLICATED_IDENTITY_ERROR = "Endorsement policy evaluation failure might be caused by duplicated identities"
)

var validCollectionNameRegex = regexp.MustCompile(ccmetadata.AllowedCharsCollectionName)

//go:generate mokery-dir../../api/capabilities/-name capabilities-case underline-output mocks/
//go:generate mokery-dir../../api/state/-name statefetcher-case underline-output mocks/
//go:generate mokery-dir../../api/identities/-name identitydeserializer-case underline-output mocks/
//go:generate mokery-dir../../api/policies/-name policyEvaluator-case underline-output mocks/

//新建创建默认VSCC的新实例
//通常，每个对等机只调用一次。
func New(c Capabilities, s StateFetcher, d IdentityDeserializer, pe PolicyEvaluator) *Validator {
	return &Validator{
		capabilities:    c,
		stateFetcher:    s,
		deserializer:    d,
		policyEvaluator: pe,
	}
}

//验证程序实现默认事务验证策略，
//检查读写设置和背书的正确性
//针对作为参数提供给
//每次调用
type Validator struct {
	deserializer    IdentityDeserializer
	capabilities    Capabilities
	stateFetcher    StateFetcher
	policyEvaluator PolicyEvaluator
}

//validate验证与带有背书的交易相对应的给定信封
//以其序列化形式提供的策略
func (vscc *Validator) Validate(
	block *common.Block,
	namespace string,
	txPosition int,
	actionPosition int,
	policyBytes []byte,
) commonerrors.TxValidationError {
//拿到信封…
	env, err := utils.GetEnvelopeFromBlock(block.Data.Data[txPosition])
	if err != nil {
		logger.Errorf("VSCC error: GetEnvelope failed, err %s", err)
		return policyErr(err)
	}

//…有效载荷…
	payl, err := utils.GetPayload(env)
	if err != nil {
		logger.Errorf("VSCC error: GetPayload failed, err %s", err)
		return policyErr(err)
	}

	chdr, err := utils.UnmarshalChannelHeader(payl.Header.ChannelHeader)
	if err != nil {
		return policyErr(err)
	}

//验证有效负载类型
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		logger.Errorf("Only Endorser Transactions are supported, provided type %d", chdr.Type)
		return policyErr(fmt.Errorf("Only Endorser Transactions are supported, provided type %d", chdr.Type))
	}

//…交易…
	tx, err := utils.GetTransaction(payl.Data)
	if err != nil {
		logger.Errorf("VSCC error: GetTransaction failed, err %s", err)
		return policyErr(err)
	}

	cap, err := utils.GetChaincodeActionPayload(tx.Actions[actionPosition].Payload)
	if err != nil {
		logger.Errorf("VSCC error: GetChaincodeActionPayload failed, err %s", err)
		return policyErr(err)
	}

	signatureSet, err := vscc.deduplicateIdentity(cap)
	if err != nil {
		return policyErr(err)
	}

//根据策略评估签名集
	err = vscc.policyEvaluator.Evaluate(policyBytes, signatureSet)
	if err != nil {
		logger.Warningf("Endorsement policy failure for transaction txid=%s, err: %s", chdr.GetTxId(), err.Error())
		if len(signatureSet) < len(cap.Action.Endorsements) {
//警告：存在重复的标识，因此可能导致背书失败。
			return policyErr(errors.New(DUPLICATED_IDENTITY_ERROR))
		}
		return policyErr(fmt.Errorf("VSCC error: endorsement policy failure, err: %s", err))
	}

//执行特定于LSCC的额外验证
	if namespace == "lscc" {
		logger.Debugf("VSCC info: doing special validation for LSCC")
		err := vscc.ValidateLSCCInvocation(chdr.ChannelId, env, cap, payl, vscc.capabilities)
		if err != nil {
			logger.Errorf("VSCC error: ValidateLSCCInvocation failed, err %s", err)
			return err
		}
	}

	return nil
}

//checkInstantiationPolicy根据签名的建议评估实例化策略
func (vscc *Validator) checkInstantiationPolicy(chainName string, env *common.Envelope, instantiationPolicy []byte, payl *common.Payload) commonerrors.TxValidationError {
//获取签名标题
	shdr, err := utils.GetSignatureHeader(payl.Header.SignatureHeader)
	if err != nil {
		return policyErr(err)
	}

//构造签名数据，我们可以根据
	sd := []*common.SignedData{{
		Data:      env.Payload,
		Identity:  shdr.Creator,
		Signature: env.Signature,
	}}
	err = vscc.policyEvaluator.Evaluate(instantiationPolicy, sd)
	if err != nil {
		return policyErr(fmt.Errorf("chaincode instantiation policy violated, error %s", err))
	}
	return nil
}

func validateNewCollectionConfigs(newCollectionConfigs []*common.CollectionConfig) error {
	newCollectionsMap := make(map[string]bool, len(newCollectionConfigs))
//处理一组集合配置中的每个集合配置
	for _, newCollectionConfig := range newCollectionConfigs {

		newCollection := newCollectionConfig.GetStaticCollectionConfig()
		if newCollection == nil {
			return errors.New("unknown collection configuration type")
		}

//确保没有重复的集合名称
		collectionName := newCollection.GetName()

		if err := validateCollectionName(collectionName); err != nil {
			return err
		}

		if _, ok := newCollectionsMap[collectionName]; !ok {
			newCollectionsMap[collectionName] = true
		} else {
			return fmt.Errorf("collection-name: %s -- found duplicate collection configuration", collectionName)
		}

//验证集合配置中存在的与八卦相关的参数
		maximumPeerCount := newCollection.GetMaximumPeerCount()
		requiredPeerCount := newCollection.GetRequiredPeerCount()
		if maximumPeerCount < requiredPeerCount {
			return fmt.Errorf("collection-name: %s -- maximum peer count (%d) cannot be greater than the required peer count (%d)",
				collectionName, maximumPeerCount, requiredPeerCount)

		}
		if requiredPeerCount < 0 {
			return fmt.Errorf("collection-name: %s -- requiredPeerCount (%d) cannot be less than zero (%d)",
				collectionName, maximumPeerCount, requiredPeerCount)

		}

//确保签名策略有意义（仅由ORS组成）
		err := validateSpOrConcat(newCollection.MemberOrgsPolicy.GetSignaturePolicy().Rule)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("collection-name: %s -- error in member org policy", collectionName))
		}
	}
	return nil
}

//validateforconcat检查所提供的签名策略是否只是标识的OR连接
func validateSpOrConcat(sp *common.SignaturePolicy) error {
	if sp.GetNOutOf() == nil {
		return nil
	}
//检查n==1（或串联）
	if sp.GetNOutOf().N != 1 {
		return errors.New(fmt.Sprintf("signature policy is not an OR concatenation, NOutOf %d", sp.GetNOutOf().N))
	}
//递归到所有子规则中
	for _, rule := range sp.GetNOutOf().Rules {
		err := validateSpOrConcat(rule)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkForMissingCollections(newCollectionsMap map[string]*common.StaticCollectionConfig, oldCollectionConfigs []*common.CollectionConfig,
) error {
	var missingCollections []string

//在新集合配置包中，确保每个旧集合都有一个条目。任何
//允许新集合的数目。
	for _, oldCollectionConfig := range oldCollectionConfigs {

		oldCollection := oldCollectionConfig.GetStaticCollectionConfig()
//不能是零
		if oldCollection == nil {
			return policyErr(fmt.Errorf("unknown collection configuration type"))
		}

//所有旧集合都必须存在于新集合配置包中
		oldCollectionName := oldCollection.GetName()
		_, ok := newCollectionsMap[oldCollectionName]
		if !ok {
			missingCollections = append(missingCollections, oldCollectionName)
		}
	}

	if len(missingCollections) > 0 {
		return policyErr(fmt.Errorf("the following existing collections are missing in the new collection configuration package: %v",
			missingCollections))
	}

	return nil
}

func checkForModifiedCollectionsBTL(newCollectionsMap map[string]*common.StaticCollectionConfig, oldCollectionConfigs []*common.CollectionConfig,
) error {
	var modifiedCollectionsBTL []string

//在新的collection config包中，确保“要生存的块”值不是
//已为现有集合修改。
	for _, oldCollectionConfig := range oldCollectionConfigs {

		oldCollection := oldCollectionConfig.GetStaticCollectionConfig()
//不能是零
		if oldCollection == nil {
			return policyErr(fmt.Errorf("unknown collection configuration type"))
		}

		oldCollectionName := oldCollection.GetName()
		newCollection, _ := newCollectionsMap[oldCollectionName]
//blocktoLive无法更改
		if newCollection.GetBlockToLive() != oldCollection.GetBlockToLive() {
			modifiedCollectionsBTL = append(modifiedCollectionsBTL, oldCollectionName)
		}
	}

	if len(modifiedCollectionsBTL) > 0 {
		return policyErr(fmt.Errorf("the BlockToLive in the following existing collections must not be modified: %v",
			modifiedCollectionsBTL))
	}

	return nil
}

func validateNewCollectionConfigsAgainstOld(newCollectionConfigs []*common.CollectionConfig, oldCollectionConfigs []*common.CollectionConfig,
) error {
	newCollectionsMap := make(map[string]*common.StaticCollectionConfig, len(newCollectionConfigs))

	for _, newCollectionConfig := range newCollectionConfigs {
		newCollection := newCollectionConfig.GetStaticCollectionConfig()
//集合对象本身存储为值，以便
//检查要激活的块是否已更改--FAB-7810
		newCollectionsMap[newCollection.GetName()] = newCollection
	}

	if err := checkForMissingCollections(newCollectionsMap, oldCollectionConfigs); err != nil {
		return err
	}

	if err := checkForModifiedCollectionsBTL(newCollectionsMap, oldCollectionConfigs); err != nil {
		return err
	}

	return nil
}

func validateCollectionName(collectionName string) error {
	if collectionName == "" {
		return fmt.Errorf("empty collection-name is not allowed")
	}
	match := validCollectionNameRegex.FindString(collectionName)
	if len(match) != len(collectionName) {
		return fmt.Errorf("collection-name: %s not allowed. A valid collection name follows the pattern: %s",
			collectionName, ccmetadata.AllowedCharsCollectionName)
	}
	return nil
}

//validaterwsetandcollection对rwset执行验证
//一个lscc部署操作，然后它验证任何集合
//配置
func (vscc *Validator) validateRWSetAndCollection(
	lsccrwset *kvrwset.KVRWSet,
	cdRWSet *ccprovider.ChaincodeData,
	lsccArgs [][]byte,
	lsccFunc string,
	ac channelconfig.ApplicationCapabilities,
	channelName string,
) commonerrors.TxValidationError {
 /***************************************/
 /*安全检查0.a-RWSET的验证*/
 /***************************************/
//只能写入一到两次
	if len(lsccrwset.Writes) > 2 {
		return policyErr(fmt.Errorf("LSCC can only issue one or two putState upon deploy"))
	}

 /******************************************************/
 /*安全检查0.b-收集数据的验证*/
 /******************************************************/
	var collectionsConfigArg []byte
	if len(lsccArgs) > 5 {
		collectionsConfigArg = lsccArgs[5]
	}

	var collectionsConfigLedger []byte
	if len(lsccrwset.Writes) == 2 {
		key := privdata.BuildCollectionKVSKey(cdRWSet.Name)
		if lsccrwset.Writes[1].Key != key {
			return policyErr(fmt.Errorf("invalid key for the collection of chaincode %s:%s; expected '%s', received '%s'",
				cdRWSet.Name, cdRWSet.Version, key, lsccrwset.Writes[1].Key))

		}

		collectionsConfigLedger = lsccrwset.Writes[1].Value
	}

	if !bytes.Equal(collectionsConfigArg, collectionsConfigLedger) {
		return policyErr(fmt.Errorf("collection configuration arguments supplied for chaincode %s:%s do not match the configuration in the lscc writeset",
			cdRWSet.Name, cdRWSet.Version))

	}

	channelState, err := vscc.stateFetcher.FetchState()
	if err != nil {
		return &commonerrors.VSCCExecutionFailureError{Err: fmt.Errorf("failed obtaining query executor: %v", err)}
	}
	defer channelState.Done()

	state := &state{channelState}

//可能不需要在v1.1中添加以下条件检查，因为不可能在
//链代码部署前的lscc命名空间。为了避免v1.2中出现分叉，将保留以下条件。
	if lsccFunc == lscc.DEPLOY {
		colCriteria := common.CollectionCriteria{Channel: channelName, Namespace: cdRWSet.Name}
		ccp, err := privdata.RetrieveCollectionConfigPackageFromState(colCriteria, state)
		if err != nil {
//如果我们收到除NoSuchCollectionError以外的任何错误，则失败。
//因为这意味着在查找
//旧款收藏
			if _, ok := err.(privdata.NoSuchCollectionError); !ok {
				return &commonerrors.VSCCExecutionFailureError{Err: fmt.Errorf("unable to check whether collection existed earlier for chaincode %s:%s",
					cdRWSet.Name, cdRWSet.Version),
				}
			}
		}
		if ccp != nil {
			return policyErr(fmt.Errorf("collection data should not exist for chaincode %s:%s", cdRWSet.Name, cdRWSet.Version))
		}
	}

//TODO:一旦新的链码生命周期可用（FAB-8724），以下验证
//在validatelsccinvocation中执行的其他验证可以移动到lscc本身。
	newCollectionConfigPackage := &common.CollectionConfigPackage{}

	if collectionsConfigArg != nil {
		err := proto.Unmarshal(collectionsConfigArg, newCollectionConfigPackage)
		if err != nil {
			return policyErr(fmt.Errorf("invalid collection configuration supplied for chaincode %s:%s",
				cdRWSet.Name, cdRWSet.Version))
		}
	} else {
		return nil
	}

	if ac.V1_2Validation() {
		newCollectionConfigs := newCollectionConfigPackage.GetConfig()
		if err := validateNewCollectionConfigs(newCollectionConfigs); err != nil {
			return policyErr(err)
		}

		if lsccFunc == lscc.UPGRADE {

			collectionCriteria := common.CollectionCriteria{Channel: channelName, Namespace: cdRWSet.Name}
//OldCollectionConfigPackage表示分类帐中的现有集合配置包
			oldCollectionConfigPackage, err := privdata.RetrieveCollectionConfigPackageFromState(collectionCriteria, state)
			if err != nil {
//如果我们收到除NoSuchCollectionError以外的任何错误，则失败。
//因为这意味着在查找
//旧款收藏
				if _, ok := err.(privdata.NoSuchCollectionError); !ok {
					return &commonerrors.VSCCExecutionFailureError{Err: fmt.Errorf("unable to check whether collection existed earlier for chaincode %s:%s: %v",
						cdRWSet.Name, cdRWSet.Version, err),
					}
				}
			}

//OldCollectionConfigPackage表示分类帐中的现有集合配置包
			if oldCollectionConfigPackage != nil {
				oldCollectionConfigs := oldCollectionConfigPackage.GetConfig()
				if err := validateNewCollectionConfigsAgainstOld(newCollectionConfigs, oldCollectionConfigs); err != nil {
					return policyErr(err)
				}

			}
		}
	}

	return nil
}

func (vscc *Validator) ValidateLSCCInvocation(
	chid string,
	env *common.Envelope,
	cap *pb.ChaincodeActionPayload,
	payl *common.Payload,
	ac channelconfig.ApplicationCapabilities,
) commonerrors.TxValidationError {
	cpp, err := utils.GetChaincodeProposalPayload(cap.ChaincodeProposalPayload)
	if err != nil {
		logger.Errorf("VSCC error: GetChaincodeProposalPayload failed, err %s", err)
		return policyErr(err)
	}

	cis := &pb.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(cpp.Input, cis)
	if err != nil {
		logger.Errorf("VSCC error: Unmarshal ChaincodeInvocationSpec failed, err %s", err)
		return policyErr(err)
	}

	if cis.ChaincodeSpec == nil ||
		cis.ChaincodeSpec.Input == nil ||
		cis.ChaincodeSpec.Input.Args == nil {
		logger.Errorf("VSCC error: committing invalid vscc invocation")
		return policyErr(fmt.Errorf("malformed chaincode invocation spec"))
	}

	lsccFunc := string(cis.ChaincodeSpec.Input.Args[0])
	lsccArgs := cis.ChaincodeSpec.Input.Args[1:]

	logger.Debugf("VSCC info: ValidateLSCCInvocation acting on %s %#v", lsccFunc, lsccArgs)

	switch lsccFunc {
	case lscc.UPGRADE, lscc.DEPLOY:
		logger.Debugf("VSCC info: validating invocation of lscc function %s on arguments %#v", lsccFunc, lsccArgs)

		if len(lsccArgs) < 2 {
			return policyErr(fmt.Errorf("Wrong number of arguments for invocation lscc(%s): expected at least 2, received %d", lsccFunc, len(lsccArgs)))
		}

		if (!ac.PrivateChannelData() && len(lsccArgs) > 5) ||
			(ac.PrivateChannelData() && len(lsccArgs) > 6) {
			return policyErr(fmt.Errorf("Wrong number of arguments for invocation lscc(%s): received %d", lsccFunc, len(lsccArgs)))
		}

		cdsArgs, err := utils.GetChaincodeDeploymentSpec(lsccArgs[1], platforms.NewRegistry(
//我们绝对不应该在VSCC中有这种外部依赖
//因为添加平台可能导致不确定性。这又是另一个
//提交时所有自定义LSCC验证都没有的原因
//长期保持确定性的希望需要消除。
			&golang.Platform{},
			&node.Platform{},
			&java.Platform{},
			&car.Platform{},
		))

		if err != nil {
			return policyErr(fmt.Errorf("GetChaincodeDeploymentSpec error %s", err))
		}

		if cdsArgs == nil || cdsArgs.ChaincodeSpec == nil || cdsArgs.ChaincodeSpec.ChaincodeId == nil ||
			cap.Action == nil || cap.Action.ProposalResponsePayload == nil {
			return policyErr(fmt.Errorf("VSCC error: invocation of lscc(%s) does not have appropriate arguments", lsccFunc))
		}

//得到RWSET
		pRespPayload, err := utils.GetProposalResponsePayload(cap.Action.ProposalResponsePayload)
		if err != nil {
			return policyErr(fmt.Errorf("GetProposalResponsePayload error %s", err))
		}
		if pRespPayload.Extension == nil {
			return policyErr(fmt.Errorf("nil pRespPayload.Extension"))
		}
		respPayload, err := utils.GetChaincodeAction(pRespPayload.Extension)
		if err != nil {
			return policyErr(fmt.Errorf("GetChaincodeAction error %s", err))
		}
		txRWSet := &rwsetutil.TxRwSet{}
		if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
			return policyErr(fmt.Errorf("txRWSet.FromProtoBytes error %s", err))
		}

//提取lscc的rwset
		var lsccrwset *kvrwset.KVRWSet
		for _, ns := range txRWSet.NsRwSets {
			logger.Debugf("Namespace %s", ns.NameSpace)
			if ns.NameSpace == "lscc" {
				lsccrwset = ns.KvRwSet
				break
			}
		}

//从分类帐中检索当前链码的条目
		cdLedger, ccExistsOnLedger, err := vscc.getInstantiatedCC(chid, cdsArgs.ChaincodeSpec.ChaincodeId.Name)
		if err != nil {
			return &commonerrors.VSCCExecutionFailureError{Err: err}
		}

  /**********************************/
  /*安全检查0-RWSET验证*/
  /**********************************/
//必须有一个写入集
		if lsccrwset == nil {
			return policyErr(fmt.Errorf("No read write set for lscc was found"))
		}
//必须至少有一次写入
		if len(lsccrwset.Writes) < 1 {
			return policyErr(fmt.Errorf("LSCC must issue at least one single putState upon deploy/upgrade"))
		}
//第一个键名必须是部署规范中提供的链代码ID
		if lsccrwset.Writes[0].Key != cdsArgs.ChaincodeSpec.ChaincodeId.Name {
			return policyErr(fmt.Errorf("expected key %s, found %s", cdsArgs.ChaincodeSpec.ChaincodeId.Name, lsccrwset.Writes[0].Key))
		}
//该值必须是chaincodedata结构
		cdRWSet := &ccprovider.ChaincodeData{}
		err = proto.Unmarshal(lsccrwset.Writes[0].Value, cdRWSet)
		if err != nil {
			return policyErr(fmt.Errorf("unmarhsalling of ChaincodeData failed, error %s", err))
		}
//lsccwriteset中的chaincode名称必须与部署规范中的chaincode名称匹配
		if cdRWSet.Name != cdsArgs.ChaincodeSpec.ChaincodeId.Name {
			return policyErr(fmt.Errorf("expected cc name %s, found %s", cdsArgs.ChaincodeSpec.ChaincodeId.Name, cdRWSet.Name))
		}
//lsccwriteset中的链代码版本必须与部署规范中的链代码版本匹配
		if cdRWSet.Version != cdsArgs.ChaincodeSpec.ChaincodeId.Version {
			return policyErr(fmt.Errorf("expected cc version %s, found %s", cdsArgs.ChaincodeSpec.ChaincodeId.Version, cdRWSet.Version))
		}
//它必须只写入2个名称空间：我们正在部署/升级的lscc和cc
		for _, ns := range txRWSet.NsRwSets {
			if ns.NameSpace != "lscc" && ns.NameSpace != cdRWSet.Name && len(ns.KvRwSet.Writes) > 0 {
				return policyErr(fmt.Errorf("LSCC invocation is attempting to write to namespace %s", ns.NameSpace))
			}
		}

		logger.Debugf("Validating %s for cc %s version %s", lsccFunc, cdRWSet.Name, cdRWSet.Version)

		switch lsccFunc {
		case lscc.DEPLOY:

   /****************************************************************/
   /*安全检查1-CC不在实例化CC的LCCC表中*/
   /****************************************************************/
			if ccExistsOnLedger {
				return policyErr(fmt.Errorf("Chaincode %s is already instantiated", cdsArgs.ChaincodeSpec.ChaincodeId.Name))
			}

   /****************************************************************/
   /*安全检查2-RWSet的验证（以及集合的验证（如果启用）*/
   /****************************************************************/
			if ac.PrivateChannelData() {
//对集合进行额外验证
				err := vscc.validateRWSetAndCollection(lsccrwset, cdRWSet, lsccArgs, lsccFunc, ac, chid)
				if err != nil {
					return err
				}
			} else {
//只能写入一个分类帐
				if len(lsccrwset.Writes) != 1 {
					return policyErr(fmt.Errorf("LSCC can only issue a single putState upon deploy"))
				}
			}

   /*************************************************/
   /*安全检查3-检查实例化策略*/
   /*************************************************/
			pol := cdRWSet.InstantiationPolicy
			if pol == nil {
				return policyErr(fmt.Errorf("no instantiation policy was specified"))
			}
//Fixme:我们能把CD包从
//用于验证是否指定策略的文件系统
//这和磁盘上的一样吗？
//优点：我们可以防止替换策略的攻击
//缺点：这是一个非决定论的观点
			err := vscc.checkInstantiationPolicy(chid, env, pol, payl)
			if err != nil {
				return err
			}

		case lscc.UPGRADE:
   /***********************************************************/
   /*实例化CC的LCCC表中的安全检查1-CC*/
   /***********************************************************/
			if !ccExistsOnLedger {
				return policyErr(fmt.Errorf("Upgrading non-existent chaincode %s", cdsArgs.ChaincodeSpec.ChaincodeId.Name))
			}

   /******************************************************/
   /*安全检查2-现有CC的版本不同*/
   /******************************************************/
			if cdLedger.Version == cdsArgs.ChaincodeSpec.ChaincodeId.Version {
				return policyErr(fmt.Errorf("Existing version of the cc on the ledger (%s) should be different from the upgraded one", cdsArgs.ChaincodeSpec.ChaincodeId.Version))
			}

   /****************************************************************/
   /*安全检查3 RWSET的验证（以及集合的验证（如果启用）*/
   /****************************************************************/
//只有在v1.2中，才能在链代码升级期间更新集合
			if ac.V1_2Validation() {
//对集合进行额外验证
				err := vscc.validateRWSetAndCollection(lsccrwset, cdRWSet, lsccArgs, lsccFunc, ac, chid)
				if err != nil {
					return err
				}
			} else {
//只能写入一个分类帐
				if len(lsccrwset.Writes) != 1 {
					return policyErr(fmt.Errorf("LSCC can only issue a single putState upon upgrade"))
				}
			}

   /*************************************************/
   /*安全检查4-检查实例化策略*/
   /*************************************************/
			pol := cdLedger.InstantiationPolicy
			if pol == nil {
				return policyErr(fmt.Errorf("No instantiation policy was specified"))
			}
//Fixme:我们能把CD包从
//用于验证是否指定策略的文件系统
//这和磁盘上的一样吗？
//优点：我们可以防止替换策略的攻击
//缺点：这是一个非决定论的观点
			err := vscc.checkInstantiationPolicy(chid, env, pol, payl)
			if err != nil {
				return err
			}

   /****************************************************************/
   /*安全检查5-检查rwset中的实例化策略*/
   /****************************************************************/
			if ac.V1_1Validation() {
				polNew := cdRWSet.InstantiationPolicy
				if polNew == nil {
					return policyErr(fmt.Errorf("No instantiation policy was specified"))
				}

//如果它们是相同的策略，那么再检查一遍就没有意义了
				if !bytes.Equal(polNew, pol) {
					err = vscc.checkInstantiationPolicy(chid, env, polNew, payl)
					if err != nil {
						return err
					}
				}
			}
		}

//一切都好！
		return nil
	default:
		return policyErr(fmt.Errorf("VSCC error: committing an invocation of function %s of lscc is invalid", lsccFunc))
	}
}

func (vscc *Validator) getInstantiatedCC(chid, ccid string) (cd *ccprovider.ChaincodeData, exists bool, err error) {
	qe, err := vscc.stateFetcher.FetchState()
	if err != nil {
		err = fmt.Errorf("could not retrieve QueryExecutor for channel %s, error %s", chid, err)
		return
	}
	defer qe.Done()
	channelState := &state{qe}
	bytes, err := channelState.GetState("lscc", ccid)
	if err != nil {
		err = fmt.Errorf("could not retrieve state for chaincode %s on channel %s, error %s", ccid, chid, err)
		return
	}

	if bytes == nil {
		return
	}

	cd = &ccprovider.ChaincodeData{}
	err = proto.Unmarshal(bytes, cd)
	if err != nil {
		err = fmt.Errorf("unmarshalling ChaincodeQueryResponse failed, error %s", err)
		return
	}

	exists = true
	return
}

func (vscc *Validator) deduplicateIdentity(cap *pb.ChaincodeActionPayload) ([]*common.SignedData, error) {
//这是签名邮件的第一部分
	prespBytes := cap.Action.ProposalResponsePayload

//为评估生成签名集
	signatureSet := []*common.SignedData{}
	signatureMap := make(map[string]struct{})
//循环遍历每个背书并构建签名集
	for _, endorsement := range cap.Action.Endorsements {
//取消标记批注器字节
		serializedIdentity := &msp.SerializedIdentity{}
		if err := proto.Unmarshal(endorsement.Endorser, serializedIdentity); err != nil {
			logger.Errorf("Unmarshal endorser error: %s", err)
			return nil, policyErr(fmt.Errorf("Unmarshal endorser error: %s", err))
		}
		identity := serializedIdentity.Mspid + string(serializedIdentity.IdBytes)
		if _, ok := signatureMap[identity]; ok {
//已添加具有相同身份的批注
			logger.Warningf("Ignoring duplicated identity, Mspid: %s, pem:\n%s", serializedIdentity.Mspid, serializedIdentity.IdBytes)
			continue
		}
		data := make([]byte, len(prespBytes)+len(endorsement.Endorser))
		copy(data, prespBytes)
		copy(data[len(prespBytes):], endorsement.Endorser)
		signatureSet = append(signatureSet, &common.SignedData{
//设置已签名的数据；建议响应字节和背书人ID的串联
			Data: data,
//设置在消息上签名的标识：它是背书人
			Identity: endorsement.Endorser,
//设置签名
			Signature: endorsement.Signature})
		signatureMap[identity] = struct{}{}
	}

	logger.Debugf("Signature set is of size %d out of %d endorsement(s)", len(signatureSet), len(cap.Action.Endorsements))
	return signatureSet, nil
}

type state struct {
	State
}

//GetState检索给定命名空间中给定键的值
func (s *state) GetState(namespace string, key string) ([]byte, error) {
	values, err := s.GetStateMultipleKeys(namespace, []string{key})
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values[0], nil
}

func policyErr(err error) *commonerrors.VSCCEndorsementPolicyError {
	return &commonerrors.VSCCEndorsementPolicyError{
		Err: err,
	}
}
