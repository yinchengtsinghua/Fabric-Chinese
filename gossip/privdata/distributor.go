
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
	"sync"
	"sync/atomic"

	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/gossip/api"
	gossipCommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	gossip2 "github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//gossip adapter八卦模块所需的API适配器
type gossipAdapter interface {
//sendbyCriteria将给定消息发送给与给定发送条件匹配的所有对等方
	SendByCriteria(message *proto.SignedGossipMessage, criteria gossip2.SendCriteria) error

//PeerFilter接收子通道SelectionCriteria并返回一个选择
//只有符合给定标准的对等身份，并且他们发布了自己的渠道参与
	PeerFilter(channel gossipCommon.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error)
}

//用于定义分发私有数据的API的pvtDataDistributor接口
type PvtDataDistributor interface {
//基于策略可靠地分发广播私有数据读写集
	Distribute(txID string, privData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error
}

//IdentityDeserializerFactory是要创建的工厂接口
//给定通道的IdentityDeserializer
type IdentityDeserializerFactory interface {
//GetIdentityDeserializer返回IdentityDeserializer
//指定链的实例
	GetIdentityDeserializer(chainID string) msp.IdentityDeserializer
}

//DistributorImpl私有数据分发接口的实现
type distributorImpl struct {
	chainID string
	gossipAdapter
	CollectionAccessFactory
}

//CollectionAccessFactory用于生成集合访问策略的接口
type CollectionAccessFactory interface {
//基于集合配置的访问策略
	AccessPolicy(config *common.CollectionConfig, chainID string) (privdata.CollectionAccessPolicy, error)
}

//策略访问工厂CollectionAccessFactory的实现
type policyAccessFactory struct {
	IdentityDeserializerFactory
}

func (p *policyAccessFactory) AccessPolicy(config *common.CollectionConfig, chainID string) (privdata.CollectionAccessPolicy, error) {
	colAP := &privdata.SimpleCollection{}
	switch cconf := config.Payload.(type) {
	case *common.CollectionConfig_StaticCollectionConfig:
		err := colAP.Setup(cconf.StaticCollectionConfig, p.GetIdentityDeserializer(chainID))
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("error setting up collection  %#v", cconf.StaticCollectionConfig.Name))
		}
	default:
		return nil, errors.New("unexpected collection type")
	}
	return colAP, nil
}

//新收进厂
func NewCollectionAccessFactory(factory IdentityDeserializerFactory) CollectionAccessFactory {
	return &policyAccessFactory{
		IdentityDeserializerFactory: factory,
	}
}

//newdistributor能够发送的私有数据分发服务器的构造函数
//基础集合的专用读写集
func NewDistributor(chainID string, gossip gossipAdapter, factory CollectionAccessFactory) PvtDataDistributor {
	return &distributorImpl{
		chainID:                 chainID,
		gossipAdapter:           gossip,
		CollectionAccessFactory: factory,
	}
}

//基于策略可靠地分发广播私有数据读写集
func (d *distributorImpl) Distribute(txID string, privData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error {
	disseminationPlan, err := d.computeDisseminationPlan(txID, privData, blkHt)
	if err != nil {
		return errors.WithStack(err)
	}
	return d.disseminate(disseminationPlan)
}

type dissemination struct {
	msg      *proto.SignedGossipMessage
	criteria gossip2.SendCriteria
}

func (d *distributorImpl) computeDisseminationPlan(txID string,
	privDataWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo,
	blkHt uint64) ([]*dissemination, error) {
	privData := privDataWithConfig.PvtRwset
	var disseminationPlan []*dissemination
	for _, pvtRwset := range privData.NsPvtRwset {
		namespace := pvtRwset.Namespace
		configPackage, found := privDataWithConfig.CollectionConfigs[namespace]
		if !found {
			logger.Error("Collection config package for", namespace, "chaincode is not provided")
			return nil, errors.New(fmt.Sprint("collection config package for", namespace, "chaincode is not provided"))
		}

		for _, collection := range pvtRwset.CollectionPvtRwset {
			colCP, err := d.getCollectionConfig(configPackage, collection)
			collectionName := collection.CollectionName
			if err != nil {
				logger.Error("Could not find collection access policy for", namespace, " and collection", collectionName, "error", err)
				return nil, errors.WithMessage(err, fmt.Sprint("could not find collection access policy for", namespace, " and collection", collectionName, "error", err))
			}

			colAP, err := d.AccessPolicy(colCP, d.chainID)
			if err != nil {
				logger.Error("Could not obtain collection access policy, collection name", collectionName, "due to", err)
				return nil, errors.Wrap(err, fmt.Sprint("Could not obtain collection access policy, collection name", collectionName, "due to", err))
			}

			colFilter := colAP.AccessFilter()
			if colFilter == nil {
				logger.Error("Collection access policy for", collectionName, "has no filter")
				return nil, errors.Errorf("No collection access policy filter computed for %v", collectionName)
			}

			pvtDataMsg, err := d.createPrivateDataMessage(txID, namespace, collection, &common.CollectionConfigPackage{Config: []*common.CollectionConfig{colCP}}, blkHt)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			dPlan, err := d.disseminationPlanForMsg(colAP, colFilter, pvtDataMsg)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			disseminationPlan = append(disseminationPlan, dPlan...)
		}
	}
	return disseminationPlan, nil
}

func (d *distributorImpl) getCollectionConfig(config *common.CollectionConfigPackage, collection *rwset.CollectionPvtReadWriteSet) (*common.CollectionConfig, error) {
	for _, c := range config.Config {
		if staticConfig := c.GetStaticCollectionConfig(); staticConfig != nil {
			if staticConfig.Name == collection.CollectionName {
				return c, nil
			}
		}
	}
	return nil, errors.New(fmt.Sprint("no configuration for collection", collection.CollectionName, "found"))
}

func (d *distributorImpl) disseminationPlanForMsg(colAP privdata.CollectionAccessPolicy, colFilter privdata.Filter, pvtDataMsg *proto.SignedGossipMessage) ([]*dissemination, error) {
	var disseminationPlan []*dissemination
	routingFilter, err := d.gossipAdapter.PeerFilter(gossipCommon.ChainID(d.chainID), func(signature api.PeerSignature) bool {
		return colFilter(common.SignedData{
			Data:      signature.Message,
			Signature: signature.Signature,
			Identity:  []byte(signature.PeerIdentity),
		})
	})

	if err != nil {
		logger.Error("Failed to retrieve peer routing filter for channel", d.chainID, ":", err)
		return nil, err
	}

	sc := gossip2.SendCriteria{
		Timeout:  viper.GetDuration("peer.gossip.pvtData.pushAckTimeout"),
		Channel:  gossipCommon.ChainID(d.chainID),
		MaxPeers: colAP.MaximumPeerCount(),
		MinAck:   colAP.RequiredPeerCount(),
		IsEligible: func(member discovery.NetworkMember) bool {
			return routingFilter(member)
		},
	}
	disseminationPlan = append(disseminationPlan, &dissemination{
		criteria: sc,
		msg:      pvtDataMsg,
	})
	return disseminationPlan, nil
}

func (d *distributorImpl) disseminate(disseminationPlan []*dissemination) error {
	var failures uint32
	var wg sync.WaitGroup
	wg.Add(len(disseminationPlan))
	for _, dis := range disseminationPlan {
		go func(dis *dissemination) {
			defer wg.Done()
			err := d.SendByCriteria(dis.msg, dis.criteria)
			if err != nil {
				atomic.AddUint32(&failures, 1)
				m := dis.msg.GetPrivateData().Payload
				logger.Error("Failed disseminating private RWSet for TxID", m.TxId, ", namespace", m.Namespace, "collection", m.CollectionName, ":", err)
			}
		}(dis)
	}
	wg.Wait()
	failureCount := atomic.LoadUint32(&failures)
	if failureCount != 0 {
		return errors.Errorf("Failed disseminating %d out of %d private RWSets", failureCount, len(disseminationPlan))
	}
	return nil
}

func (d *distributorImpl) createPrivateDataMessage(txID, namespace string,
	collection *rwset.CollectionPvtReadWriteSet,
	ccp *common.CollectionConfigPackage,
	blkHt uint64) (*proto.SignedGossipMessage, error) {
	msg := &proto.GossipMessage{
		Channel: []byte(d.chainID),
		Nonce:   util.RandomUInt64(),
		Tag:     proto.GossipMessage_CHAN_ONLY,
		Content: &proto.GossipMessage_PrivateData{
			PrivateData: &proto.PrivateDataMessage{
				Payload: &proto.PrivatePayload{
					Namespace:         namespace,
					CollectionName:    collection.CollectionName,
					TxId:              txID,
					PrivateRwset:      collection.Rwset,
					PrivateSimHeight:  blkHt,
					CollectionConfigs: ccp,
				},
			},
		},
	}

	pvtDataMsg, err := msg.NoopSign()
	if err != nil {
		return nil, err
	}
	return pvtDataMsg, nil
}
