
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


package service

import (
	"reflect"

	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/protos/peer"
)

//config枚举八卦所需的配置方法
type Config interface {
//chainID返回此通道的chainID
	ChainID() string

//组织将组织ID的映射返回到ApplicationOrgConfig
	Organizations() map[string]channelconfig.ApplicationOrg

//序列应返回当前配置的序列号
	Sequence() uint64

//orderAddresses返回要连接以调用广播/传递的有效订购者地址列表
	OrdererAddresses() []string
}

//ConfigProcessor接收配置更新
type ConfigProcessor interface {
//每当初始化或更新通道的配置时，应调用processconfig。
	ProcessConfigUpdate(config Config)
}

type configStore struct {
	anchorPeers []*peer.AnchorPeer
	orgMap      map[string]channelconfig.ApplicationOrg
}

type configEventReceiver interface {
	updateAnchors(config Config)
	updateEndpoints(chainID string, endpoints []string)
}

type configEventer struct {
	lastConfig *configStore
	receiver   configEventReceiver
}

func newConfigEventer(receiver configEventReceiver) *configEventer {
	return &configEventer{
		receiver: receiver,
	}
}

//每当初始化或更新通道的配置时，应调用processconfigupdate。
//当更新配置时，它调用configeventreceiver中的关联方法
//但只有当配置值实际更改时
//注意，更改序列号将被忽略为更改配置
func (ce *configEventer) ProcessConfigUpdate(config Config) {
	logger.Debugf("Processing new config for channel %s", config.ChainID())
	orgMap := cloneOrgConfig(config.Organizations())
	if ce.lastConfig != nil && reflect.DeepEqual(ce.lastConfig.orgMap, orgMap) {
		logger.Debugf("Ignoring new config for channel %s because it contained no anchor peer updates", config.ChainID())
	} else {

		var newAnchorPeers []*peer.AnchorPeer
		for _, group := range config.Organizations() {
			newAnchorPeers = append(newAnchorPeers, group.AnchorPeers()...)
		}

		newConfig := &configStore{
			orgMap:      orgMap,
			anchorPeers: newAnchorPeers,
		}
		ce.lastConfig = newConfig

		logger.Debugf("Calling out because config was updated for channel %s", config.ChainID())
		ce.receiver.updateAnchors(config)
	}
	ce.receiver.updateEndpoints(config.ChainID(), config.OrdererAddresses())
}

func cloneOrgConfig(src map[string]channelconfig.ApplicationOrg) map[string]channelconfig.ApplicationOrg {
	clone := make(map[string]channelconfig.ApplicationOrg)
	for k, v := range src {
		clone[k] = &appGrp{
			name:        v.Name(),
			mspID:       v.MSPID(),
			anchorPeers: v.AnchorPeers(),
		}
	}
	return clone
}

type appGrp struct {
	name        string
	mspID       string
	anchorPeers []*peer.AnchorPeer
}

func (ag *appGrp) Name() string {
	return ag.name
}

func (ag *appGrp) MSPID() string {
	return ag.mspID
}

func (ag *appGrp) AnchorPeers() []*peer.AnchorPeer {
	return ag.anchorPeers
}
