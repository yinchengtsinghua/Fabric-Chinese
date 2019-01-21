
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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

//支持是用于注入依赖项的接口
type Support interface {
//GetQueryExecutorForLedger返回指定通道的查询执行器
	GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error)

//GetIdentityDeserializer返回IdentityDeserializer
//指定链的实例
	GetIdentityDeserializer(chainID string) msp.IdentityDeserializer
}

//Stategetter从状态中检索数据
type State interface {
//GetState retrieves the value for the given key in the given namespace
	GetState(namespace string, key string) ([]byte, error)
}

type NoSuchCollectionError common.CollectionCriteria

func (f NoSuchCollectionError) Error() string {
	return fmt.Sprintf("collection %s/%s/%s could not be found", f.Channel, f.Namespace, f.Collection)
}

type simpleCollectionStore struct {
	s Support
}

//NewSimpleCollectionStore返回存储在备份中的集合
//由指定的Ledgergetter提供的分类帐
//由提供的
//collectionnamer函数
func NewSimpleCollectionStore(s Support) CollectionStore {
	return &simpleCollectionStore{s}
}

func (c *simpleCollectionStore) retrieveCollectionConfigPackage(cc common.CollectionCriteria, qe ledger.QueryExecutor) (*common.CollectionConfigPackage, error) {
	if qe != nil {
		return RetrieveCollectionConfigPackageFromState(cc, qe)
	}

	qe, err := c.s.GetQueryExecutorForLedger(cc.Channel)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("could not retrieve query executor for collection criteria %#v", cc))
	}
	defer qe.Done()
	return RetrieveCollectionConfigPackageFromState(cc, qe)
}

//RetrieveCollectionConfigPackageFromState从给定的键从给定的状态检索集合配置包
func RetrieveCollectionConfigPackageFromState(cc common.CollectionCriteria, state State) (*common.CollectionConfigPackage, error) {
	cb, err := state.GetState("lscc", BuildCollectionKVSKey(cc.Namespace))
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("error while retrieving collection for collection criteria %#v", cc))
	}
	if cb == nil {
		return nil, NoSuchCollectionError(cc)
	}
	conf, err := ParseCollectionConfig(cb)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid configuration for collection criteria %#v", cc)
	}
	return conf, nil
}

//PARSECURLISTION CONFIG从给定的序列化表示解析集合配置
func ParseCollectionConfig(colBytes []byte) (*common.CollectionConfigPackage, error) {
	collections := &common.CollectionConfigPackage{}
	err := proto.Unmarshal(colBytes, collections)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return collections, nil
}

func (c *simpleCollectionStore) retrieveCollectionConfig(cc common.CollectionCriteria, qe ledger.QueryExecutor) (*common.StaticCollectionConfig, error) {
	collections, err := c.retrieveCollectionConfigPackage(cc, qe)
	if err != nil {
		return nil, err
	}
	if collections == nil {
		return nil, nil
	}
	for _, cconf := range collections.Config {
		switch cconf := cconf.Payload.(type) {
		case *common.CollectionConfig_StaticCollectionConfig:
			if cconf.StaticCollectionConfig.Name == cc.Collection {
				return cconf.StaticCollectionConfig, nil
			}
		default:
			return nil, errors.New("unexpected collection type")
		}
	}
	return nil, NoSuchCollectionError(cc)
}

func (c *simpleCollectionStore) retrieveSimpleCollection(cc common.CollectionCriteria, qe ledger.QueryExecutor) (*SimpleCollection, error) {
	staticCollectionConfig, err := c.retrieveCollectionConfig(cc, qe)
	if err != nil {
		return nil, err
	}
	sc := &SimpleCollection{}
	err = sc.Setup(staticCollectionConfig, c.s.GetIdentityDeserializer(cc.Channel))
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("error setting up collection for collection criteria %#v", cc))
	}
	return sc, nil
}

func (c *simpleCollectionStore) AccessFilter(channelName string, collectionPolicyConfig *common.CollectionPolicyConfig) (Filter, error) {
	sc := &SimpleCollection{}
	err := sc.setupAccessPolicy(collectionPolicyConfig, c.s.GetIdentityDeserializer(channelName))
	if err != nil {
		return nil, err
	}
	return sc.AccessFilter(), nil
}

func (c *simpleCollectionStore) RetrieveCollection(cc common.CollectionCriteria) (Collection, error) {
	return c.retrieveSimpleCollection(cc, nil)
}

func (c *simpleCollectionStore) RetrieveCollectionAccessPolicy(cc common.CollectionCriteria) (CollectionAccessPolicy, error) {
	return c.retrieveSimpleCollection(cc, nil)
}

func (c *simpleCollectionStore) RetrieveCollectionConfigPackage(cc common.CollectionCriteria) (*common.CollectionConfigPackage, error) {
	return c.retrieveCollectionConfigPackage(cc, nil)
}

//RetrieveCollectionPersistenceConfigs检索集合的与持久性相关的配置
func (c *simpleCollectionStore) RetrieveCollectionPersistenceConfigs(cc common.CollectionCriteria) (CollectionPersistenceConfigs, error) {
	staticCollectionConfig, err := c.retrieveCollectionConfig(cc, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleCollectionPersistenceConfigs{staticCollectionConfig.BlockToLive}, nil
}

func (c *simpleCollectionStore) HasReadAccess(cc common.CollectionCriteria, signedProposal *pb.SignedProposal, qe ledger.QueryExecutor) (bool, error) {
	accessPolicy, err := c.retrieveSimpleCollection(cc, qe)
	if err != nil {
		return false, err
	}

	if !accessPolicy.IsMemberOnlyRead() {
		return true, nil
	}

	signedData, err := getSignedData(signedProposal)
	if err != nil {
		return false, err
	}

	hasReadAccess := accessPolicy.AccessFilter()
	return hasReadAccess(signedData), nil
}

func getSignedData(signedProposal *pb.SignedProposal) (common.SignedData, error) {
	proposal, err := utils.GetProposal(signedProposal.ProposalBytes)
	if err != nil {
		return common.SignedData{}, err
	}

	creator, _, err := utils.GetChaincodeProposalContext(proposal)
	if err != nil {
		return common.SignedData{}, err
	}

	return common.SignedData{
		Data:      signedProposal.ProposalBytes,
		Identity:  creator,
		Signature: signedProposal.Signature,
	}, nil
}
